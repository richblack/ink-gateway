package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"semantic-text-processor/models"
	"strings"
	"time"

	"github.com/lib/pq"
)

// SearchCacheEnhancedUnifiedChunkService wraps UnifiedChunkService with database search cache
type SearchCacheEnhancedUnifiedChunkService struct {
	base        UnifiedChunkService
	searchCache SearchCacheService
	db          *sql.DB
	monitor     QueryPerformanceMonitor
}

// NewSearchCacheEnhancedUnifiedChunkService creates a new search cache enhanced service
func NewSearchCacheEnhancedUnifiedChunkService(
	base UnifiedChunkService,
	searchCache SearchCacheService,
	db *sql.DB,
	monitor QueryPerformanceMonitor,
) UnifiedChunkService {
	return &SearchCacheEnhancedUnifiedChunkService{
		base:        base,
		searchCache: searchCache,
		db:          db,
		monitor:     monitor,
	}
}

// SearchChunks performs search with database cache integration
func (s *SearchCacheEnhancedUnifiedChunkService) SearchChunks(ctx context.Context, query *models.SearchQuery) (*models.SearchResult, error) {
	start := time.Now()
	
	// Convert query to cache parameters
	queryParams := s.queryToParams(query)
	
	// Try to get from database cache first
	cacheEntry, err := s.searchCache.GetCachedSearch(ctx, queryParams)
	if err != nil {
		s.monitor.RecordQuery("search_cache_error", time.Since(start), 0)
		// Continue with actual search if cache fails
	} else if cacheEntry != nil {
		// Cache hit - reconstruct result from cached chunk IDs
		chunks, err := s.getChunksByIDs(ctx, cacheEntry.ChunkIDs)
		if err != nil {
			s.monitor.RecordQuery("search_cache_reconstruction_error", time.Since(start), 0)
			// Fall back to actual search
		} else {
			result := &models.SearchResult{
				Chunks:     chunks,
				TotalCount: cacheEntry.ResultCount,
				HasMore:    false, // Simplified for cached results
				SearchTime: time.Since(start),
				CacheHit:   true,
			}
			s.monitor.RecordQuery("search_chunks_cached", time.Since(start), len(chunks))
			return result, nil
		}
	}
	
	// Cache miss or error - perform actual search
	result, err := s.performActualSearch(ctx, query)
	if err != nil {
		s.monitor.RecordQuery("search_chunks_error", time.Since(start), 0)
		return nil, err
	}
	
	result.SearchTime = time.Since(start)
	result.CacheHit = false
	
	// Cache the result asynchronously
	go func() {
		cacheCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		
		chunkIDs := make([]string, len(result.Chunks))
		for i, chunk := range result.Chunks {
			chunkIDs[i] = chunk.ChunkID
		}
		
		ttl := s.getTTLForQuery(query)
		if err := s.searchCache.SetCachedSearch(cacheCtx, queryParams, chunkIDs, ttl); err != nil {
			s.monitor.RecordQuery("search_cache_set_error", 0, 0)
		}
	}()
	
	s.monitor.RecordQuery("search_chunks", time.Since(start), len(result.Chunks))
	return result, nil
}

// SearchByContent performs content search with database cache integration
func (s *SearchCacheEnhancedUnifiedChunkService) SearchByContent(ctx context.Context, content string, filters map[string]interface{}) ([]models.UnifiedChunkRecord, error) {
	
	// Convert to SearchQuery for consistent caching
	query := &models.SearchQuery{
		Content: content,
		Limit:   1000, // Default limit
	}
	
	// Apply filters
	if filters != nil {
		if isPage, ok := filters["is_page"].(bool); ok {
			query.IsPage = &isPage
		}
		if isTag, ok := filters["is_tag"].(bool); ok {
			query.IsTag = &isTag
		}
		if isTemplate, ok := filters["is_template"].(bool); ok {
			query.IsTemplate = &isTemplate
		}
		if isSlot, ok := filters["is_slot"].(bool); ok {
			query.IsSlot = &isSlot
		}
		if parent, ok := filters["parent"].(string); ok {
			query.Parent = &parent
		}
		if page, ok := filters["page"].(string); ok {
			query.Page = &page
		}
		if limit, ok := filters["limit"].(int); ok {
			query.Limit = limit
		}
		if offset, ok := filters["offset"].(int); ok {
			query.Offset = offset
		}
	}
	
	// Use SearchChunks for consistent caching
	result, err := s.SearchChunks(ctx, query)
	if err != nil {
		return nil, err
	}
	
	return result.Chunks, nil
}

// performActualSearch executes the actual database search
func (s *SearchCacheEnhancedUnifiedChunkService) performActualSearch(ctx context.Context, query *models.SearchQuery) (*models.SearchResult, error) {
	var conditions []string
	var args []interface{}
	argIndex := 1
	
	// Build WHERE conditions
	if query.Content != "" {
		conditions = append(conditions, fmt.Sprintf("to_tsvector('english', contents) @@ plainto_tsquery('english', $%d)", argIndex))
		args = append(args, query.Content)
		argIndex++
	}
	
	if query.IsPage != nil {
		conditions = append(conditions, fmt.Sprintf("is_page = $%d", argIndex))
		args = append(args, *query.IsPage)
		argIndex++
	}
	
	if query.IsTag != nil {
		conditions = append(conditions, fmt.Sprintf("is_tag = $%d", argIndex))
		args = append(args, *query.IsTag)
		argIndex++
	}
	
	if query.IsTemplate != nil {
		conditions = append(conditions, fmt.Sprintf("is_template = $%d", argIndex))
		args = append(args, *query.IsTemplate)
		argIndex++
	}
	
	if query.IsSlot != nil {
		conditions = append(conditions, fmt.Sprintf("is_slot = $%d", argIndex))
		args = append(args, *query.IsSlot)
		argIndex++
	}
	
	if query.Parent != nil {
		conditions = append(conditions, fmt.Sprintf("parent = $%d", argIndex))
		args = append(args, *query.Parent)
		argIndex++
	}
	
	if query.Page != nil {
		conditions = append(conditions, fmt.Sprintf("page = $%d", argIndex))
		args = append(args, *query.Page)
		argIndex++
	}
	
	// Handle tag filtering
	if len(query.Tags) > 0 {
		if query.TagLogic == "AND" {
			// All tags must match
			conditions = append(conditions, fmt.Sprintf(`
				chunk_id IN (
					SELECT source_chunk_id 
					FROM chunk_tags 
					WHERE tag_chunk_id = ANY($%d)
					GROUP BY source_chunk_id 
					HAVING COUNT(DISTINCT tag_chunk_id) = $%d
				)`, argIndex, argIndex+1))
			args = append(args, pq.Array(query.Tags), len(query.Tags))
			argIndex += 2
		} else {
			// Any tag matches (OR logic)
			conditions = append(conditions, fmt.Sprintf(`
				chunk_id IN (
					SELECT DISTINCT source_chunk_id 
					FROM chunk_tags 
					WHERE tag_chunk_id = ANY($%d)
				)`, argIndex))
			args = append(args, pq.Array(query.Tags))
			argIndex++
		}
	}
	
	// Handle metadata filtering
	if query.Metadata != nil && len(query.Metadata) > 0 {
		for key, value := range query.Metadata {
			conditions = append(conditions, fmt.Sprintf("metadata->>'%s' = $%d", key, argIndex))
			args = append(args, value)
			argIndex++
		}
	}
	
	// Build the main query
	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}
	
	// Count query for total results
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM chunks %s", whereClause)
	var totalCount int
	err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&totalCount)
	if err != nil {
		return nil, fmt.Errorf("failed to count search results: %w", err)
	}
	
	// Main search query with pagination
	searchQuery := fmt.Sprintf(`
		SELECT chunk_id, contents, parent, page, is_page, is_tag, is_template, is_slot,
			   ref, tags, metadata, created_time, last_updated
		FROM chunks %s
		ORDER BY 
			CASE WHEN $%d != '' THEN ts_rank(to_tsvector('english', contents), plainto_tsquery('english', $%d)) END DESC,
			created_time DESC
	`, whereClause, argIndex, argIndex+1)
	
	// Add content parameter for ranking (even if empty)
	args = append(args, query.Content, query.Content)
	argIndex += 2
	
	// Add pagination
	if query.Limit > 0 {
		searchQuery += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, query.Limit)
		argIndex++
	}
	
	if query.Offset > 0 {
		searchQuery += fmt.Sprintf(" OFFSET $%d", argIndex)
		args = append(args, query.Offset)
		argIndex++
	}
	
	// Execute search query
	rows, err := s.db.QueryContext(ctx, searchQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute search query: %w", err)
	}
	defer rows.Close()
	
	var chunks []models.UnifiedChunkRecord
	for rows.Next() {
		var chunk models.UnifiedChunkRecord
		var tags pq.StringArray
		var metadata []byte
		
		err := rows.Scan(
			&chunk.ChunkID,
			&chunk.Contents,
			&chunk.Parent,
			&chunk.Page,
			&chunk.IsPage,
			&chunk.IsTag,
			&chunk.IsTemplate,
			&chunk.IsSlot,
			&chunk.Ref,
			&tags,
			&metadata,
			&chunk.CreatedTime,
			&chunk.LastUpdated,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan search result: %w", err)
		}
		
		chunk.Tags = []string(tags)
		
		// Parse metadata JSON
		if len(metadata) > 0 {
			chunk.Metadata = make(map[string]interface{})
			if err := json.Unmarshal(metadata, &chunk.Metadata); err != nil {
				// Continue with empty metadata if parsing fails
				chunk.Metadata = make(map[string]interface{})
			}
		}
		
		chunks = append(chunks, chunk)
	}
	
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating search results: %w", err)
	}
	
	// Determine if there are more results
	hasMore := false
	if query.Limit > 0 {
		hasMore = totalCount > query.Offset+len(chunks)
	}
	
	return &models.SearchResult{
		Chunks:     chunks,
		TotalCount: totalCount,
		HasMore:    hasMore,
	}, nil
}

// getChunksByIDs retrieves chunks by their IDs for cache reconstruction
func (s *SearchCacheEnhancedUnifiedChunkService) getChunksByIDs(ctx context.Context, chunkIDs []string) ([]models.UnifiedChunkRecord, error) {
	if len(chunkIDs) == 0 {
		return []models.UnifiedChunkRecord{}, nil
	}
	
	query := `
		SELECT chunk_id, contents, parent, page, is_page, is_tag, is_template, is_slot,
			   ref, tags, metadata, created_time, last_updated
		FROM chunks 
		WHERE chunk_id = ANY($1)
		ORDER BY array_position($1, chunk_id)
	`
	
	rows, err := s.db.QueryContext(ctx, query, pq.Array(chunkIDs))
	if err != nil {
		return nil, fmt.Errorf("failed to get chunks by IDs: %w", err)
	}
	defer rows.Close()
	
	var chunks []models.UnifiedChunkRecord
	for rows.Next() {
		var chunk models.UnifiedChunkRecord
		var tags pq.StringArray
		var metadata []byte
		
		err := rows.Scan(
			&chunk.ChunkID,
			&chunk.Contents,
			&chunk.Parent,
			&chunk.Page,
			&chunk.IsPage,
			&chunk.IsTag,
			&chunk.IsTemplate,
			&chunk.IsSlot,
			&chunk.Ref,
			&tags,
			&metadata,
			&chunk.CreatedTime,
			&chunk.LastUpdated,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan chunk: %w", err)
		}
		
		chunk.Tags = []string(tags)
		
		// Parse metadata JSON
		if len(metadata) > 0 {
			chunk.Metadata = make(map[string]interface{})
			if err := json.Unmarshal(metadata, &chunk.Metadata); err != nil {
				chunk.Metadata = make(map[string]interface{})
			}
		}
		
		chunks = append(chunks, chunk)
	}
	
	return chunks, rows.Err()
}

// queryToParams converts SearchQuery to cache parameters
func (s *SearchCacheEnhancedUnifiedChunkService) queryToParams(query *models.SearchQuery) map[string]interface{} {
	params := make(map[string]interface{})
	
	if query.Content != "" {
		params["content"] = query.Content
	}
	if len(query.Tags) > 0 {
		params["tags"] = query.Tags
		params["tag_logic"] = query.TagLogic
	}
	if query.IsPage != nil {
		params["is_page"] = *query.IsPage
	}
	if query.IsTag != nil {
		params["is_tag"] = *query.IsTag
	}
	if query.IsTemplate != nil {
		params["is_template"] = *query.IsTemplate
	}
	if query.IsSlot != nil {
		params["is_slot"] = *query.IsSlot
	}
	if query.Parent != nil {
		params["parent"] = *query.Parent
	}
	if query.Page != nil {
		params["page"] = *query.Page
	}
	if query.Metadata != nil && len(query.Metadata) > 0 {
		params["metadata"] = query.Metadata
	}
	if query.Limit > 0 {
		params["limit"] = query.Limit
	}
	if query.Offset > 0 {
		params["offset"] = query.Offset
	}
	
	return params
}

// getTTLForQuery determines appropriate TTL based on query characteristics
func (s *SearchCacheEnhancedUnifiedChunkService) getTTLForQuery(query *models.SearchQuery) time.Duration {
	// Content searches change frequently
	if query.Content != "" {
		return 5 * time.Minute
	}
	
	// Tag-based searches are more stable
	if len(query.Tags) > 0 {
		return 15 * time.Minute
	}
	
	// Type-based searches are very stable
	if query.IsPage != nil || query.IsTag != nil || query.IsTemplate != nil || query.IsSlot != nil {
		return 30 * time.Minute
	}
	
	// Default TTL
	return 10 * time.Minute
}

// Delegate all other methods to the base service

func (s *SearchCacheEnhancedUnifiedChunkService) CreateChunk(ctx context.Context, chunk *models.UnifiedChunkRecord) error {
	err := s.base.CreateChunk(ctx, chunk)
	if err != nil {
		return err
	}
	
	// Invalidate search cache when new content is created
	go func() {
		cacheCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		s.searchCache.InvalidateSearchCache(cacheCtx, []string{"*"})
	}()
	
	return nil
}

func (s *SearchCacheEnhancedUnifiedChunkService) GetChunk(ctx context.Context, chunkID string) (*models.UnifiedChunkRecord, error) {
	return s.base.GetChunk(ctx, chunkID)
}

func (s *SearchCacheEnhancedUnifiedChunkService) UpdateChunk(ctx context.Context, chunk *models.UnifiedChunkRecord) error {
	err := s.base.UpdateChunk(ctx, chunk)
	if err != nil {
		return err
	}
	
	// Invalidate search cache when content is updated
	go func() {
		cacheCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		s.searchCache.InvalidateSearchCache(cacheCtx, []string{"*"})
	}()
	
	return nil
}

func (s *SearchCacheEnhancedUnifiedChunkService) DeleteChunk(ctx context.Context, chunkID string) error {
	err := s.base.DeleteChunk(ctx, chunkID)
	if err != nil {
		return err
	}
	
	// Invalidate search cache when content is deleted
	go func() {
		cacheCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		s.searchCache.InvalidateSearchCache(cacheCtx, []string{"*"})
	}()
	
	return nil
}

func (s *SearchCacheEnhancedUnifiedChunkService) BatchCreateChunks(ctx context.Context, chunks []models.UnifiedChunkRecord) error {
	err := s.base.BatchCreateChunks(ctx, chunks)
	if err != nil {
		return err
	}
	
	// Invalidate search cache for batch operations
	go func() {
		cacheCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		s.searchCache.InvalidateSearchCache(cacheCtx, []string{"*"})
	}()
	
	return nil
}

func (s *SearchCacheEnhancedUnifiedChunkService) BatchUpdateChunks(ctx context.Context, chunks []models.UnifiedChunkRecord) error {
	err := s.base.BatchUpdateChunks(ctx, chunks)
	if err != nil {
		return err
	}
	
	// Invalidate search cache for batch operations
	go func() {
		cacheCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		s.searchCache.InvalidateSearchCache(cacheCtx, []string{"*"})
	}()
	
	return nil
}

func (s *SearchCacheEnhancedUnifiedChunkService) AddTags(ctx context.Context, chunkID string, tagChunkIDs []string) error {
	return s.base.AddTags(ctx, chunkID, tagChunkIDs)
}

func (s *SearchCacheEnhancedUnifiedChunkService) RemoveTags(ctx context.Context, chunkID string, tagChunkIDs []string) error {
	return s.base.RemoveTags(ctx, chunkID, tagChunkIDs)
}

func (s *SearchCacheEnhancedUnifiedChunkService) GetChunkTags(ctx context.Context, chunkID string) ([]models.UnifiedChunkRecord, error) {
	return s.base.GetChunkTags(ctx, chunkID)
}

func (s *SearchCacheEnhancedUnifiedChunkService) GetChunksByTag(ctx context.Context, tagChunkID string) ([]models.UnifiedChunkRecord, error) {
	return s.base.GetChunksByTag(ctx, tagChunkID)
}

func (s *SearchCacheEnhancedUnifiedChunkService) GetChunksByTags(ctx context.Context, tagChunkIDs []string, matchType string) ([]models.UnifiedChunkRecord, error) {
	return s.base.GetChunksByTags(ctx, tagChunkIDs, matchType)
}

func (s *SearchCacheEnhancedUnifiedChunkService) GetChildren(ctx context.Context, parentChunkID string) ([]models.UnifiedChunkRecord, error) {
	return s.base.GetChildren(ctx, parentChunkID)
}

func (s *SearchCacheEnhancedUnifiedChunkService) GetDescendants(ctx context.Context, ancestorChunkID string, maxDepth int) ([]models.UnifiedChunkRecord, error) {
	return s.base.GetDescendants(ctx, ancestorChunkID, maxDepth)
}

func (s *SearchCacheEnhancedUnifiedChunkService) GetAncestors(ctx context.Context, chunkID string) ([]models.UnifiedChunkRecord, error) {
	return s.base.GetAncestors(ctx, chunkID)
}

func (s *SearchCacheEnhancedUnifiedChunkService) MoveChunk(ctx context.Context, chunkID, newParentID string) error {
	return s.base.MoveChunk(ctx, chunkID, newParentID)
}