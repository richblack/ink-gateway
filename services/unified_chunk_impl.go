package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"semantic-text-processor/models"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// unifiedChunkService implements UnifiedChunkService interface
type unifiedChunkService struct {
	db      *sql.DB
	cache   CacheService
	monitor QueryPerformanceMonitor
}

// NewUnifiedChunkService creates a new instance of UnifiedChunkService
func NewUnifiedChunkService(db *sql.DB, cache CacheService, monitor QueryPerformanceMonitor) UnifiedChunkService {
	return &unifiedChunkService{
		db:      db,
		cache:   cache,
		monitor: monitor,
	}
}

// CreateChunk creates a new chunk in the unified table
func (s *unifiedChunkService) CreateChunk(ctx context.Context, chunk *models.UnifiedChunkRecord) error {
	start := time.Now()
	defer func() {
		if s.monitor != nil {
			s.monitor.RecordQuery("create_chunk", time.Since(start), 1)
		}
	}()

	// Generate UUID if not provided
	if chunk.ChunkID == "" {
		chunk.ChunkID = uuid.New().String()
	}

	// Set timestamps
	now := time.Now()
	chunk.CreatedTime = now
	chunk.LastUpdated = now

	query := `
		INSERT INTO chunks (
			chunk_id, contents, parent, page, is_page, is_tag, is_template, is_slot,
			ref, tags, metadata, created_time, last_updated
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
		)`

	_, err := s.db.ExecContext(ctx, query,
		chunk.ChunkID, chunk.Contents, chunk.Parent, chunk.Page,
		chunk.IsPage, chunk.IsTag, chunk.IsTemplate, chunk.IsSlot,
		chunk.Ref, pq.Array(chunk.Tags), chunk.Metadata,
		chunk.CreatedTime, chunk.LastUpdated,
	)

	if err != nil {
		return fmt.Errorf("failed to create chunk: %w", err)
	}

	// Invalidate related caches
	s.invalidateChunkCaches(ctx, chunk.ChunkID)

	return nil
}

// GetChunk retrieves a chunk by ID
func (s *unifiedChunkService) GetChunk(ctx context.Context, chunkID string) (*models.UnifiedChunkRecord, error) {
	start := time.Now()
	defer func() {
		if s.monitor != nil {
			s.monitor.RecordQuery("get_chunk", time.Since(start), 1)
		}
	}()

	// Check cache first
	cacheKey := fmt.Sprintf("chunk:%s", chunkID)
	if cached, found := s.cache.GetDirect(ctx, cacheKey); found {
		return cached.(*models.UnifiedChunkRecord), nil
	}

	query := `
		SELECT chunk_id, contents, parent, page, is_page, is_tag, is_template, is_slot,
			   ref, tags, metadata, created_time, last_updated
		FROM chunks
		WHERE chunk_id = $1`

	var chunk models.UnifiedChunkRecord
	var tags pq.StringArray
	var metadataBytes []byte

	err := s.db.QueryRowContext(ctx, query, chunkID).Scan(
		&chunk.ChunkID, &chunk.Contents, &chunk.Parent, &chunk.Page,
		&chunk.IsPage, &chunk.IsTag, &chunk.IsTemplate, &chunk.IsSlot,
		&chunk.Ref, &tags, &metadataBytes,
		&chunk.CreatedTime, &chunk.LastUpdated,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("chunk not found: %s", chunkID)
		}
		return nil, fmt.Errorf("failed to get chunk: %w", err)
	}

	chunk.Tags = []string(tags)

	// Parse metadata JSON if present
	if len(metadataBytes) > 0 {
		if err := json.Unmarshal(metadataBytes, &chunk.Metadata); err != nil {
			log.Printf("Warning: failed to parse metadata for chunk %s: %v", chunkID, err)
			chunk.Metadata = make(map[string]interface{})
		}
	} else {
		chunk.Metadata = make(map[string]interface{})
	}

	// Cache the result
	s.cache.Set(ctx, cacheKey, &chunk, 5*time.Minute)

	return &chunk, nil
}

// UpdateChunk updates an existing chunk
func (s *unifiedChunkService) UpdateChunk(ctx context.Context, chunk *models.UnifiedChunkRecord) error {
	start := time.Now()
	defer func() {
		s.monitor.RecordQuery("update_chunk", time.Since(start), 1)
	}()

	// Update timestamp
	chunk.LastUpdated = time.Now()

	query := `
		UPDATE chunks SET
			contents = $2, parent = $3, page = $4, is_page = $5, is_tag = $6,
			is_template = $7, is_slot = $8, ref = $9, tags = $10, metadata = $11,
			last_updated = $12
		WHERE chunk_id = $1`

	result, err := s.db.ExecContext(ctx, query,
		chunk.ChunkID, chunk.Contents, chunk.Parent, chunk.Page,
		chunk.IsPage, chunk.IsTag, chunk.IsTemplate, chunk.IsSlot,
		chunk.Ref, pq.Array(chunk.Tags), chunk.Metadata,
		chunk.LastUpdated,
	)

	if err != nil {
		return fmt.Errorf("failed to update chunk: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("chunk not found: %s", chunk.ChunkID)
	}

	// Invalidate related caches
	s.invalidateChunkCaches(ctx, chunk.ChunkID)

	return nil
}

// DeleteChunk deletes a chunk by ID
func (s *unifiedChunkService) DeleteChunk(ctx context.Context, chunkID string) error {
	start := time.Now()
	defer func() {
		s.monitor.RecordQuery("delete_chunk", time.Since(start), 1)
	}()

	query := `DELETE FROM chunks WHERE chunk_id = $1`

	result, err := s.db.ExecContext(ctx, query, chunkID)
	if err != nil {
		return fmt.Errorf("failed to delete chunk: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("chunk not found: %s", chunkID)
	}

	// Invalidate related caches
	s.invalidateChunkCaches(ctx, chunkID)

	return nil
}

// BatchCreateChunks creates multiple chunks in a single transaction
func (s *unifiedChunkService) BatchCreateChunks(ctx context.Context, chunks []models.UnifiedChunkRecord) error {
	start := time.Now()
	defer func() {
		s.monitor.RecordQuery("batch_create_chunks", time.Since(start), len(chunks))
	}()

	if len(chunks) == 0 {
		return nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	query := `
		INSERT INTO chunks (
			chunk_id, contents, parent, page, is_page, is_tag, is_template, is_slot,
			ref, tags, metadata, created_time, last_updated
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
		)`

	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	now := time.Now()
	for i := range chunks {
		chunk := &chunks[i]
		
		// Generate UUID if not provided
		if chunk.ChunkID == "" {
			chunk.ChunkID = uuid.New().String()
		}

		// Set timestamps
		chunk.CreatedTime = now
		chunk.LastUpdated = now

		_, err = stmt.ExecContext(ctx,
			chunk.ChunkID, chunk.Contents, chunk.Parent, chunk.Page,
			chunk.IsPage, chunk.IsTag, chunk.IsTemplate, chunk.IsSlot,
			chunk.Ref, pq.Array(chunk.Tags), chunk.Metadata,
			chunk.CreatedTime, chunk.LastUpdated,
		)

		if err != nil {
			return fmt.Errorf("failed to insert chunk %s: %w", chunk.ChunkID, err)
		}
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Invalidate caches for all created chunks
	for _, chunk := range chunks {
		s.invalidateChunkCaches(ctx, chunk.ChunkID)
	}

	return nil
}

// BatchUpdateChunks updates multiple chunks in a single transaction
func (s *unifiedChunkService) BatchUpdateChunks(ctx context.Context, chunks []models.UnifiedChunkRecord) error {
	start := time.Now()
	defer func() {
		s.monitor.RecordQuery("batch_update_chunks", time.Since(start), len(chunks))
	}()

	if len(chunks) == 0 {
		return nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	query := `
		UPDATE chunks SET
			contents = $2, parent = $3, page = $4, is_page = $5, is_tag = $6,
			is_template = $7, is_slot = $8, ref = $9, tags = $10, metadata = $11,
			last_updated = $12
		WHERE chunk_id = $1`

	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	now := time.Now()
	for i := range chunks {
		chunk := &chunks[i]
		chunk.LastUpdated = now

		_, err = stmt.ExecContext(ctx,
			chunk.ChunkID, chunk.Contents, chunk.Parent, chunk.Page,
			chunk.IsPage, chunk.IsTag, chunk.IsTemplate, chunk.IsSlot,
			chunk.Ref, pq.Array(chunk.Tags), chunk.Metadata,
			chunk.LastUpdated,
		)

		if err != nil {
			return fmt.Errorf("failed to update chunk %s: %w", chunk.ChunkID, err)
		}
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Invalidate caches for all updated chunks
	for _, chunk := range chunks {
		s.invalidateChunkCaches(ctx, chunk.ChunkID)
	}

	return nil
}

// Helper methods for cache management and query execution
func (s *unifiedChunkService) invalidateChunkCaches(ctx context.Context, chunkID string) {
	patterns := []string{
		fmt.Sprintf("chunk:%s", chunkID),
		fmt.Sprintf("chunk_tags:%s", chunkID),
		fmt.Sprintf("chunk_children:%s", chunkID),
		fmt.Sprintf("chunk_descendants:%s:*", chunkID),
		fmt.Sprintf("chunk_ancestors:%s", chunkID),
		"chunks_by_tag:*",
		"chunks_by_tags:*",
		"chunk_children:*",
		"chunk_descendants:*",
		"chunk_ancestors:*",
	}

	for _, pattern := range patterns {
		s.cache.DeletePattern(ctx, pattern)
	}
}

// ============================================================================
// TAG OPERATIONS IMPLEMENTATION
// ============================================================================

// AddTags adds tags to a chunk, maintaining both main table and auxiliary table consistency
func (s *unifiedChunkService) AddTags(ctx context.Context, chunkID string, tagChunkIDs []string) error {
	start := time.Now()
	defer func() {
		s.monitor.RecordQuery("add_tags", time.Since(start), len(tagChunkIDs))
	}()

	if len(tagChunkIDs) == 0 {
		return nil
	}

	// Validate that all tag chunk IDs exist and are actually tags
	for _, tagID := range tagChunkIDs {
		var isTag bool
		err := s.db.QueryRowContext(ctx, "SELECT is_tag FROM chunks WHERE chunk_id = $1", tagID).Scan(&isTag)
		if err != nil {
			if err == sql.ErrNoRows {
				return fmt.Errorf("tag chunk not found: %s", tagID)
			}
			return fmt.Errorf("failed to validate tag chunk %s: %w", tagID, err)
		}
		if !isTag {
			return fmt.Errorf("chunk %s is not a tag", tagID)
		}
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Get current tags from the main table
	var currentTags pq.StringArray
	err = tx.QueryRowContext(ctx, "SELECT COALESCE(tags, '[]'::jsonb) FROM chunks WHERE chunk_id = $1", chunkID).Scan(&currentTags)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("chunk not found: %s", chunkID)
		}
		return fmt.Errorf("failed to get current tags: %w", err)
	}

	// Merge new tags with existing ones (avoid duplicates)
	tagSet := make(map[string]bool)
	for _, tag := range currentTags {
		tagSet[tag] = true
	}
	for _, tag := range tagChunkIDs {
		tagSet[tag] = true
	}

	// Convert back to slice
	allTags := make([]string, 0, len(tagSet))
	for tag := range tagSet {
		allTags = append(allTags, tag)
	}

	// Update main table with merged tags
	_, err = tx.ExecContext(ctx, 
		"UPDATE chunks SET tags = $1, last_updated = NOW() WHERE chunk_id = $2",
		pq.Array(allTags), chunkID)
	if err != nil {
		return fmt.Errorf("failed to update main table tags: %w", err)
	}

	// The auxiliary table will be updated automatically by the database trigger
	// But we can also manually ensure consistency by inserting new relationships
	for _, tagID := range tagChunkIDs {
		_, err = tx.ExecContext(ctx,
			"INSERT INTO chunk_tags (source_chunk_id, tag_chunk_id) VALUES ($1, $2) ON CONFLICT DO NOTHING",
			chunkID, tagID)
		if err != nil {
			return fmt.Errorf("failed to insert tag relationship: %w", err)
		}
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Invalidate related caches
	s.invalidateTagCaches(ctx, chunkID, tagChunkIDs)

	return nil
}

// RemoveTags removes tags from a chunk, maintaining both main table and auxiliary table consistency
func (s *unifiedChunkService) RemoveTags(ctx context.Context, chunkID string, tagChunkIDs []string) error {
	start := time.Now()
	defer func() {
		s.monitor.RecordQuery("remove_tags", time.Since(start), len(tagChunkIDs))
	}()

	if len(tagChunkIDs) == 0 {
		return nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Get current tags from the main table
	var currentTags pq.StringArray
	err = tx.QueryRowContext(ctx, "SELECT COALESCE(tags, '[]'::jsonb) FROM chunks WHERE chunk_id = $1", chunkID).Scan(&currentTags)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("chunk not found: %s", chunkID)
		}
		return fmt.Errorf("failed to get current tags: %w", err)
	}

	// Remove specified tags
	tagsToRemove := make(map[string]bool)
	for _, tag := range tagChunkIDs {
		tagsToRemove[tag] = true
	}

	remainingTags := make([]string, 0)
	for _, tag := range currentTags {
		if !tagsToRemove[tag] {
			remainingTags = append(remainingTags, tag)
		}
	}

	// Update main table with remaining tags
	_, err = tx.ExecContext(ctx,
		"UPDATE chunks SET tags = $1, last_updated = NOW() WHERE chunk_id = $2",
		pq.Array(remainingTags), chunkID)
	if err != nil {
		return fmt.Errorf("failed to update main table tags: %w", err)
	}

	// Remove relationships from auxiliary table
	_, err = tx.ExecContext(ctx,
		"DELETE FROM chunk_tags WHERE source_chunk_id = $1 AND tag_chunk_id = ANY($2)",
		chunkID, pq.Array(tagChunkIDs))
	if err != nil {
		return fmt.Errorf("failed to remove tag relationships: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Invalidate related caches
	s.invalidateTagCaches(ctx, chunkID, tagChunkIDs)

	return nil
}

// GetChunkTags retrieves all tags associated with a chunk
func (s *unifiedChunkService) GetChunkTags(ctx context.Context, chunkID string) ([]models.UnifiedChunkRecord, error) {
	start := time.Now()
	defer func() {
		s.monitor.RecordQuery("get_chunk_tags", time.Since(start), 0)
	}()

	// Check cache first
	cacheKey := fmt.Sprintf("chunk_tags:%s", chunkID)
	if cached, found := s.cache.GetDirect(ctx, cacheKey); found {
		return cached.([]models.UnifiedChunkRecord), nil
	}

	query := `
		SELECT c.chunk_id, c.contents, c.parent, c.page, c.is_page, c.is_tag, 
			   c.is_template, c.is_slot, c.ref, c.tags, c.metadata, 
			   c.created_time, c.last_updated
		FROM chunks c
		JOIN chunk_tags ct ON c.chunk_id = ct.tag_chunk_id
		WHERE ct.source_chunk_id = $1 AND c.is_tag = true
		ORDER BY c.contents ASC
	`

	rows, err := s.db.QueryContext(ctx, query, chunkID)
	if err != nil {
		return nil, fmt.Errorf("failed to query chunk tags: %w", err)
	}
	defer rows.Close()

	var tags []models.UnifiedChunkRecord
	for rows.Next() {
		var tag models.UnifiedChunkRecord
		var tagArray pq.StringArray

		err := rows.Scan(
			&tag.ChunkID, &tag.Contents, &tag.Parent, &tag.Page,
			&tag.IsPage, &tag.IsTag, &tag.IsTemplate, &tag.IsSlot,
			&tag.Ref, &tagArray, &tag.Metadata,
			&tag.CreatedTime, &tag.LastUpdated,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan tag row: %w", err)
		}

		tag.Tags = []string(tagArray)
		tags = append(tags, tag)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating tag rows: %w", err)
	}

	// Cache the result
	s.cache.Set(ctx, cacheKey, tags, 5*time.Minute)

	// Update performance metrics
	s.monitor.RecordQuery("get_chunk_tags", time.Since(start), len(tags))

	return tags, nil
}

// GetChunksByTag retrieves all chunks that have a specific tag
func (s *unifiedChunkService) GetChunksByTag(ctx context.Context, tagChunkID string) ([]models.UnifiedChunkRecord, error) {
	start := time.Now()
	defer func() {
		s.monitor.RecordQuery("get_chunks_by_tag", time.Since(start), 0)
	}()

	// Check cache first
	cacheKey := fmt.Sprintf("chunks_by_tag:%s", tagChunkID)
	if cached, found := s.cache.GetDirect(ctx, cacheKey); found {
		return cached.([]models.UnifiedChunkRecord), nil
	}

	// Validate that the tag chunk exists and is actually a tag
	var isTag bool
	err := s.db.QueryRowContext(ctx, "SELECT is_tag FROM chunks WHERE chunk_id = $1", tagChunkID).Scan(&isTag)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("tag chunk not found: %s", tagChunkID)
		}
		return nil, fmt.Errorf("failed to validate tag chunk: %w", err)
	}
	if !isTag {
		return nil, fmt.Errorf("chunk %s is not a tag", tagChunkID)
	}

	query := `
		SELECT c.chunk_id, c.contents, c.parent, c.page, c.is_page, c.is_tag,
			   c.is_template, c.is_slot, c.ref, c.tags, c.metadata,
			   c.created_time, c.last_updated
		FROM chunks c
		JOIN chunk_tags ct ON c.chunk_id = ct.source_chunk_id
		WHERE ct.tag_chunk_id = $1
		ORDER BY c.created_time DESC
	`

	rows, err := s.db.QueryContext(ctx, query, tagChunkID)
	if err != nil {
		return nil, fmt.Errorf("failed to query chunks by tag: %w", err)
	}
	defer rows.Close()

	var chunks []models.UnifiedChunkRecord
	for rows.Next() {
		var chunk models.UnifiedChunkRecord
		var tagArray pq.StringArray

		err := rows.Scan(
			&chunk.ChunkID, &chunk.Contents, &chunk.Parent, &chunk.Page,
			&chunk.IsPage, &chunk.IsTag, &chunk.IsTemplate, &chunk.IsSlot,
			&chunk.Ref, &tagArray, &chunk.Metadata,
			&chunk.CreatedTime, &chunk.LastUpdated,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan chunk row: %w", err)
		}

		chunk.Tags = []string(tagArray)
		chunks = append(chunks, chunk)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating chunk rows: %w", err)
	}

	// Cache the result
	s.cache.Set(ctx, cacheKey, chunks, 5*time.Minute)

	// Update performance metrics
	s.monitor.RecordQuery("get_chunks_by_tag", time.Since(start), len(chunks))

	return chunks, nil
}

// GetChunksByTags retrieves chunks that match multiple tags with AND/OR logic
func (s *unifiedChunkService) GetChunksByTags(ctx context.Context, tagChunkIDs []string, matchType string) ([]models.UnifiedChunkRecord, error) {
	start := time.Now()
	defer func() {
		s.monitor.RecordQuery("get_chunks_by_tags", time.Since(start), 0)
	}()

	if len(tagChunkIDs) == 0 {
		return []models.UnifiedChunkRecord{}, nil
	}

	// Validate match type
	if matchType != "AND" && matchType != "OR" {
		return nil, fmt.Errorf("invalid match type: %s (must be 'AND' or 'OR')", matchType)
	}

	// Check cache first
	cacheKey := fmt.Sprintf("chunks_by_tags:%s:%s", matchType, fmt.Sprintf("%v", tagChunkIDs))
	if cached, found := s.cache.GetDirect(ctx, cacheKey); found {
		return cached.([]models.UnifiedChunkRecord), nil
	}

	// Validate that all tag chunks exist and are actually tags
	for _, tagID := range tagChunkIDs {
		var isTag bool
		err := s.db.QueryRowContext(ctx, "SELECT is_tag FROM chunks WHERE chunk_id = $1", tagID).Scan(&isTag)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, fmt.Errorf("tag chunk not found: %s", tagID)
			}
			return nil, fmt.Errorf("failed to validate tag chunk %s: %w", tagID, err)
		}
		if !isTag {
			return nil, fmt.Errorf("chunk %s is not a tag", tagID)
		}
	}

	var query string
	var args []interface{}

	if matchType == "AND" {
		// AND logic: chunks must have ALL specified tags
		query = `
			SELECT c.chunk_id, c.contents, c.parent, c.page, c.is_page, c.is_tag,
				   c.is_template, c.is_slot, c.ref, c.tags, c.metadata,
				   c.created_time, c.last_updated
			FROM chunks c
			WHERE c.chunk_id IN (
				SELECT source_chunk_id 
				FROM chunk_tags 
				WHERE tag_chunk_id = ANY($1)
				GROUP BY source_chunk_id 
				HAVING COUNT(DISTINCT tag_chunk_id) = $2
			)
			ORDER BY c.created_time DESC
		`
		args = []interface{}{pq.Array(tagChunkIDs), len(tagChunkIDs)}
	} else {
		// OR logic: chunks must have ANY of the specified tags
		query = `
			SELECT DISTINCT c.chunk_id, c.contents, c.parent, c.page, c.is_page, c.is_tag,
				   c.is_template, c.is_slot, c.ref, c.tags, c.metadata,
				   c.created_time, c.last_updated
			FROM chunks c
			JOIN chunk_tags ct ON c.chunk_id = ct.source_chunk_id
			WHERE ct.tag_chunk_id = ANY($1)
			ORDER BY c.created_time DESC
		`
		args = []interface{}{pq.Array(tagChunkIDs)}
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query chunks by tags: %w", err)
	}
	defer rows.Close()

	var chunks []models.UnifiedChunkRecord
	for rows.Next() {
		var chunk models.UnifiedChunkRecord
		var tagArray pq.StringArray

		err := rows.Scan(
			&chunk.ChunkID, &chunk.Contents, &chunk.Parent, &chunk.Page,
			&chunk.IsPage, &chunk.IsTag, &chunk.IsTemplate, &chunk.IsSlot,
			&chunk.Ref, &tagArray, &chunk.Metadata,
			&chunk.CreatedTime, &chunk.LastUpdated,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan chunk row: %w", err)
		}

		chunk.Tags = []string(tagArray)
		chunks = append(chunks, chunk)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating chunk rows: %w", err)
	}

	// Cache the result
	s.cache.Set(ctx, cacheKey, chunks, 5*time.Minute)

	// Update performance metrics
	s.monitor.RecordQuery("get_chunks_by_tags", time.Since(start), len(chunks))

	return chunks, nil
}

// Helper function to invalidate tag-related caches
func (s *unifiedChunkService) invalidateTagCaches(ctx context.Context, chunkID string, tagChunkIDs []string) {
	patterns := []string{
		fmt.Sprintf("chunk_tags:%s", chunkID),
		"chunks_by_tag:*",
		"chunks_by_tags:*",
	}

	// Also invalidate caches for specific tags
	for _, tagID := range tagChunkIDs {
		patterns = append(patterns, fmt.Sprintf("chunks_by_tag:%s", tagID))
	}

	for _, pattern := range patterns {
		s.cache.DeletePattern(ctx, pattern)
	}
}

// ============================================================================
// HIERARCHY OPERATIONS IMPLEMENTATION
// ============================================================================

// GetChildren retrieves direct children of a parent chunk
func (s *unifiedChunkService) GetChildren(ctx context.Context, parentChunkID string) ([]models.UnifiedChunkRecord, error) {
	start := time.Now()
	defer func() {
		s.monitor.RecordQuery("get_children", time.Since(start), 0)
	}()

	// Check cache first
	cacheKey := fmt.Sprintf("chunk_children:%s", parentChunkID)
	if cached, found := s.cache.GetDirect(ctx, cacheKey); found {
		return cached.([]models.UnifiedChunkRecord), nil
	}

	// Validate that parent chunk exists
	var exists bool
	err := s.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM chunks WHERE chunk_id = $1)", parentChunkID).Scan(&exists)
	if err != nil {
		return nil, fmt.Errorf("failed to validate parent chunk: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("parent chunk not found: %s", parentChunkID)
	}

	// Query direct children using the hierarchy auxiliary table for optimal performance
	query := `
		SELECT c.chunk_id, c.contents, c.parent, c.page, c.is_page, c.is_tag,
			   c.is_template, c.is_slot, c.ref, c.tags, c.metadata,
			   c.created_time, c.last_updated
		FROM chunks c
		JOIN chunk_hierarchy ch ON c.chunk_id = ch.descendant_id
		WHERE ch.ancestor_id = $1 AND ch.depth = 1
		ORDER BY c.created_time ASC
	`

	rows, err := s.db.QueryContext(ctx, query, parentChunkID)
	if err != nil {
		return nil, fmt.Errorf("failed to query children: %w", err)
	}
	defer rows.Close()

	var children []models.UnifiedChunkRecord
	for rows.Next() {
		var child models.UnifiedChunkRecord
		var tagArray pq.StringArray

		err := rows.Scan(
			&child.ChunkID, &child.Contents, &child.Parent, &child.Page,
			&child.IsPage, &child.IsTag, &child.IsTemplate, &child.IsSlot,
			&child.Ref, &tagArray, &child.Metadata,
			&child.CreatedTime, &child.LastUpdated,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan child row: %w", err)
		}

		child.Tags = []string(tagArray)
		children = append(children, child)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating child rows: %w", err)
	}

	// Cache the result
	s.cache.Set(ctx, cacheKey, children, 5*time.Minute)

	// Update performance metrics
	s.monitor.RecordQuery("get_children", time.Since(start), len(children))

	return children, nil
}

// GetDescendants retrieves all descendants of an ancestor chunk with optional depth limit
func (s *unifiedChunkService) GetDescendants(ctx context.Context, ancestorChunkID string, maxDepth int) ([]models.UnifiedChunkRecord, error) {
	start := time.Now()
	defer func() {
		s.monitor.RecordQuery("get_descendants", time.Since(start), 0)
	}()

	// Check cache first
	cacheKey := fmt.Sprintf("chunk_descendants:%s:%d", ancestorChunkID, maxDepth)
	if cached, found := s.cache.GetDirect(ctx, cacheKey); found {
		return cached.([]models.UnifiedChunkRecord), nil
	}

	// Validate that ancestor chunk exists
	var exists bool
	err := s.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM chunks WHERE chunk_id = $1)", ancestorChunkID).Scan(&exists)
	if err != nil {
		return nil, fmt.Errorf("failed to validate ancestor chunk: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("ancestor chunk not found: %s", ancestorChunkID)
	}

	// Build query with optional depth limit
	query := `
		SELECT c.chunk_id, c.contents, c.parent, c.page, c.is_page, c.is_tag,
			   c.is_template, c.is_slot, c.ref, c.tags, c.metadata,
			   c.created_time, c.last_updated, ch.depth, ch.path_ids
		FROM chunks c
		JOIN chunk_hierarchy ch ON c.chunk_id = ch.descendant_id
		WHERE ch.ancestor_id = $1 AND ch.depth > 0
	`

	args := []interface{}{ancestorChunkID}
	if maxDepth > 0 {
		query += " AND ch.depth <= $2"
		args = append(args, maxDepth)
	}

	query += " ORDER BY ch.depth ASC, c.created_time ASC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query descendants: %w", err)
	}
	defer rows.Close()

	var descendants []models.UnifiedChunkRecord
	for rows.Next() {
		var descendant models.UnifiedChunkRecord
		var tagArray pq.StringArray
		var depth int
		var pathIDs pq.StringArray

		err := rows.Scan(
			&descendant.ChunkID, &descendant.Contents, &descendant.Parent, &descendant.Page,
			&descendant.IsPage, &descendant.IsTag, &descendant.IsTemplate, &descendant.IsSlot,
			&descendant.Ref, &tagArray, &descendant.Metadata,
			&descendant.CreatedTime, &descendant.LastUpdated, &depth, &pathIDs,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan descendant row: %w", err)
		}

		descendant.Tags = []string(tagArray)
		// Store hierarchy information in metadata for client use
		if descendant.Metadata == nil {
			descendant.Metadata = make(map[string]interface{})
		}
		descendant.Metadata["hierarchy_depth"] = depth
		descendant.Metadata["hierarchy_path"] = []string(pathIDs)

		descendants = append(descendants, descendant)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating descendant rows: %w", err)
	}

	// Cache the result
	s.cache.Set(ctx, cacheKey, descendants, 5*time.Minute)

	// Update performance metrics
	s.monitor.RecordQuery("get_descendants", time.Since(start), len(descendants))

	return descendants, nil
}

// GetAncestors retrieves all ancestors of a chunk (path from root to chunk)
func (s *unifiedChunkService) GetAncestors(ctx context.Context, chunkID string) ([]models.UnifiedChunkRecord, error) {
	start := time.Now()
	defer func() {
		s.monitor.RecordQuery("get_ancestors", time.Since(start), 0)
	}()

	// Check cache first
	cacheKey := fmt.Sprintf("chunk_ancestors:%s", chunkID)
	if cached, found := s.cache.GetDirect(ctx, cacheKey); found {
		return cached.([]models.UnifiedChunkRecord), nil
	}

	// Validate that chunk exists
	var exists bool
	err := s.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM chunks WHERE chunk_id = $1)", chunkID).Scan(&exists)
	if err != nil {
		return nil, fmt.Errorf("failed to validate chunk: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("chunk not found: %s", chunkID)
	}

	// Query ancestors using the hierarchy auxiliary table
	query := `
		SELECT c.chunk_id, c.contents, c.parent, c.page, c.is_page, c.is_tag,
			   c.is_template, c.is_slot, c.ref, c.tags, c.metadata,
			   c.created_time, c.last_updated, ch.depth
		FROM chunks c
		JOIN chunk_hierarchy ch ON c.chunk_id = ch.ancestor_id
		WHERE ch.descendant_id = $1 AND ch.depth > 0
		ORDER BY ch.depth DESC
	`

	rows, err := s.db.QueryContext(ctx, query, chunkID)
	if err != nil {
		return nil, fmt.Errorf("failed to query ancestors: %w", err)
	}
	defer rows.Close()

	var ancestors []models.UnifiedChunkRecord
	for rows.Next() {
		var ancestor models.UnifiedChunkRecord
		var tagArray pq.StringArray
		var depth int

		err := rows.Scan(
			&ancestor.ChunkID, &ancestor.Contents, &ancestor.Parent, &ancestor.Page,
			&ancestor.IsPage, &ancestor.IsTag, &ancestor.IsTemplate, &ancestor.IsSlot,
			&ancestor.Ref, &tagArray, &ancestor.Metadata,
			&ancestor.CreatedTime, &ancestor.LastUpdated, &depth,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan ancestor row: %w", err)
		}

		ancestor.Tags = []string(tagArray)
		// Store hierarchy information in metadata for client use
		if ancestor.Metadata == nil {
			ancestor.Metadata = make(map[string]interface{})
		}
		ancestor.Metadata["hierarchy_depth"] = depth

		ancestors = append(ancestors, ancestor)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating ancestor rows: %w", err)
	}

	// Cache the result
	s.cache.Set(ctx, cacheKey, ancestors, 5*time.Minute)

	// Update performance metrics
	s.monitor.RecordQuery("get_ancestors", time.Since(start), len(ancestors))

	return ancestors, nil
}

// MoveChunk moves a chunk to a new parent, automatically updating the hierarchy auxiliary table
func (s *unifiedChunkService) MoveChunk(ctx context.Context, chunkID, newParentID string) error {
	start := time.Now()
	defer func() {
		s.monitor.RecordQuery("move_chunk", time.Since(start), 1)
	}()

	// Validate that chunk exists
	var exists bool
	err := s.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM chunks WHERE chunk_id = $1)", chunkID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to validate chunk: %w", err)
	}
	if !exists {
		return fmt.Errorf("chunk not found: %s", chunkID)
	}

	// Validate new parent exists (if not null)
	if newParentID != "" {
		err = s.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM chunks WHERE chunk_id = $1)", newParentID).Scan(&exists)
		if err != nil {
			return fmt.Errorf("failed to validate new parent chunk: %w", err)
		}
		if !exists {
			return fmt.Errorf("new parent chunk not found: %s", newParentID)
		}

		// Check for circular reference - ensure new parent is not a descendant of the chunk being moved
		var isDescendant bool
		err = s.db.QueryRowContext(ctx, 
			"SELECT EXISTS(SELECT 1 FROM chunk_hierarchy WHERE ancestor_id = $1 AND descendant_id = $2)",
			chunkID, newParentID).Scan(&isDescendant)
		if err != nil {
			return fmt.Errorf("failed to check for circular reference: %w", err)
		}
		if isDescendant {
			return fmt.Errorf("cannot move chunk to its own descendant: circular reference detected")
		}
	}

	// Begin transaction for atomic operation
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Update the parent field in the main table
	var parentPtr *string
	if newParentID != "" {
		parentPtr = &newParentID
	}

	_, err = tx.ExecContext(ctx,
		"UPDATE chunks SET parent = $1, last_updated = NOW() WHERE chunk_id = $2",
		parentPtr, chunkID)
	if err != nil {
		return fmt.Errorf("failed to update chunk parent: %w", err)
	}

	// The hierarchy auxiliary table will be automatically updated by the database trigger
	// But we can also manually ensure consistency by clearing and rebuilding hierarchy for this chunk
	// This is handled by the trigger, but we could add manual verification here if needed

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Invalidate related caches
	s.invalidateHierarchyCaches(ctx, chunkID, newParentID)

	return nil
}

// Helper function to invalidate hierarchy-related caches
func (s *unifiedChunkService) invalidateHierarchyCaches(ctx context.Context, chunkID, parentID string) {
	patterns := []string{
		fmt.Sprintf("chunk_children:%s", chunkID),
		fmt.Sprintf("chunk_descendants:%s:*", chunkID),
		fmt.Sprintf("chunk_ancestors:%s", chunkID),
		"chunk_children:*",
		"chunk_descendants:*",
		"chunk_ancestors:*",
	}

	// Also invalidate caches for the new parent
	if parentID != "" {
		patterns = append(patterns, 
			fmt.Sprintf("chunk_children:%s", parentID),
			fmt.Sprintf("chunk_descendants:%s:*", parentID),
		)
	}

	for _, pattern := range patterns {
		s.cache.DeletePattern(ctx, pattern)
	}
}

func (s *unifiedChunkService) SearchChunks(ctx context.Context, query *models.SearchQuery) (*models.SearchResult, error) {
	return nil, fmt.Errorf("not implemented - will be implemented in later tasks")
}

func (s *unifiedChunkService) SearchByContent(ctx context.Context, content string, filters map[string]interface{}) ([]models.UnifiedChunkRecord, error) {
	return nil, fmt.Errorf("not implemented - will be implemented in later tasks")
}