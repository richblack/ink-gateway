package migration

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"semantic-text-processor/models"
)

// DataTransformer handles transformation of data from old schema to unified schema
type DataTransformer struct {
	config *MigrationConfig
	logger *log.Logger
}

// NewDataTransformer creates a new data transformer
func NewDataTransformer(config *MigrationConfig) *DataTransformer {
	return &DataTransformer{
		config: config,
		logger: log.New(log.Writer(), "[DATA_TRANSFORMER] ", log.LstdFlags|log.Lshortfile),
	}
}

// ChunkTransformResult represents the result of transforming a chunk
type ChunkTransformResult struct {
	UnifiedChunk *models.UnifiedChunkRecord
	TagRelations []models.ChunkTagRelation
	Error        error
}

// TransformChunksToUnified transforms chunks from old schema to unified schema
func (dt *DataTransformer) TransformChunksToUnified(ctx context.Context, sourceDB, targetDB *sql.DB) error {
	dt.logger.Println("Starting chunk transformation to unified format")

	// Query to extract chunks with related data
	query := `
		SELECT
			c.id,
			c.text_id,
			c.content,
			c.is_template,
			c.is_slot,
			c.parent_chunk_id,
			c.template_chunk_id,
			c.slot_value,
			c.indent_level,
			c.sequence_number,
			c.metadata,
			c.created_at,
			c.updated_at,
			t.title as text_title,
			t.content as text_content
		FROM content_db.chunks c
		LEFT JOIN content_db.texts t ON c.text_id = t.id
		ORDER BY c.created_at
	`

	rows, err := sourceDB.QueryContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to query source chunks: %w", err)
	}
	defer rows.Close()

	// Process chunks in batches
	batch := make([]*ChunkTransformResult, 0, dt.config.BatchSize)
	processedCount := 0

	for rows.Next() {
		result := dt.transformSingleChunk(rows)
		if result.Error != nil {
			dt.logger.Printf("Error transforming chunk: %v", result.Error)
			continue
		}

		batch = append(batch, result)

		// Process batch when full
		if len(batch) >= dt.config.BatchSize {
			if err := dt.processBatch(ctx, targetDB, batch); err != nil {
				return fmt.Errorf("failed to process batch: %w", err)
			}
			processedCount += len(batch)
			dt.logger.Printf("Processed %d chunks", processedCount)
			batch = batch[:0] // Reset batch
		}
	}

	// Process remaining chunks
	if len(batch) > 0 {
		if err := dt.processBatch(ctx, targetDB, batch); err != nil {
			return fmt.Errorf("failed to process final batch: %w", err)
		}
		processedCount += len(batch)
	}

	dt.logger.Printf("Chunk transformation completed. Total processed: %d", processedCount)
	return nil
}

// transformSingleChunk transforms a single chunk row to unified format
func (dt *DataTransformer) transformSingleChunk(rows *sql.Rows) *ChunkTransformResult {
	var (
		id              string
		textID          string
		content         string
		isTemplate      bool
		isSlot          bool
		parentChunkID   sql.NullString
		templateChunkID sql.NullString
		slotValue       sql.NullString
		indentLevel     int
		sequenceNumber  sql.NullInt64
		metadataJSON    sql.NullString
		createdAt       time.Time
		updatedAt       time.Time
		textTitle       sql.NullString
		textContent     sql.NullString
	)

	err := rows.Scan(
		&id, &textID, &content, &isTemplate, &isSlot,
		&parentChunkID, &templateChunkID, &slotValue,
		&indentLevel, &sequenceNumber, &metadataJSON,
		&createdAt, &updatedAt, &textTitle, &textContent,
	)

	if err != nil {
		return &ChunkTransformResult{Error: fmt.Errorf("failed to scan chunk row: %w", err)}
	}

	// Parse metadata
	var metadata map[string]interface{}
	if metadataJSON.Valid {
		if err := json.Unmarshal([]byte(metadataJSON.String), &metadata); err != nil {
			dt.logger.Printf("Failed to parse metadata for chunk %s: %v", id, err)
			metadata = make(map[string]interface{})
		}
	} else {
		metadata = make(map[string]interface{})
	}

	// Enhance metadata with original fields
	metadata["original_text_id"] = textID
	if slotValue.Valid {
		metadata["slot_value"] = slotValue.String
	}
	if sequenceNumber.Valid {
		metadata["sequence_number"] = sequenceNumber.Int64
	}
	metadata["indent_level"] = indentLevel

	// Determine chunk type flags
	isPage := dt.determineIsPage(textID, parentChunkID, isTemplate, isSlot)
	isTag := dt.determineIsTag(content)

	// Create unified chunk record
	unifiedChunk := &models.UnifiedChunkRecord{
		ChunkID:     id,
		Contents:    content,
		Parent:      dt.nullStringToPointer(parentChunkID),
		Page:        dt.derivePageReference(textID, parentChunkID),
		IsPage:      isPage,
		IsTag:       isTag,
		IsTemplate:  isTemplate,
		IsSlot:      isSlot,
		Ref:         &textID,
		Tags:        []string{}, // Will be populated in tag transformation
		Metadata:    metadata,
		CreatedTime: createdAt,
		LastUpdated: updatedAt,
	}

	return &ChunkTransformResult{
		UnifiedChunk: unifiedChunk,
		TagRelations: []models.ChunkTagRelation{}, // Will be populated separately
	}
}

// processBatch processes a batch of transformed chunks
func (dt *DataTransformer) processBatch(ctx context.Context, targetDB *sql.DB, batch []*ChunkTransformResult) error {
	if dt.config.DryRun {
		dt.logger.Printf("DRY RUN: Would insert %d chunks", len(batch))
		return nil
	}

	tx, err := targetDB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Prepare insert statement
	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO chunks (
			chunk_id, contents, parent, page, is_page, is_tag,
			is_template, is_slot, ref, tags, metadata, created_time, last_updated
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		ON CONFLICT (chunk_id) DO UPDATE SET
			contents = EXCLUDED.contents,
			parent = EXCLUDED.parent,
			page = EXCLUDED.page,
			is_page = EXCLUDED.is_page,
			is_tag = EXCLUDED.is_tag,
			is_template = EXCLUDED.is_template,
			is_slot = EXCLUDED.is_slot,
			ref = EXCLUDED.ref,
			tags = EXCLUDED.tags,
			metadata = EXCLUDED.metadata,
			last_updated = EXCLUDED.last_updated
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare insert statement: %w", err)
	}
	defer stmt.Close()

	// Insert each chunk in the batch
	for _, result := range batch {
		chunk := result.UnifiedChunk

		// Convert tags slice to JSON
		tagsJSON, err := json.Marshal(chunk.Tags)
		if err != nil {
			return fmt.Errorf("failed to marshal tags for chunk %s: %w", chunk.ChunkID, err)
		}

		// Convert metadata to JSON
		metadataJSON, err := json.Marshal(chunk.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata for chunk %s: %w", chunk.ChunkID, err)
		}

		_, err = stmt.ExecContext(ctx,
			chunk.ChunkID, chunk.Contents, chunk.Parent, chunk.Page,
			chunk.IsPage, chunk.IsTag, chunk.IsTemplate, chunk.IsSlot,
			chunk.Ref, tagsJSON, metadataJSON, chunk.CreatedTime, chunk.LastUpdated,
		)
		if err != nil {
			return fmt.Errorf("failed to insert chunk %s: %w", chunk.ChunkID, err)
		}
	}

	return tx.Commit()
}

// TransformTagRelationships transforms tag relationships from old to new schema
func (dt *DataTransformer) TransformTagRelationships(ctx context.Context, sourceDB, targetDB *sql.DB) error {
	dt.logger.Println("Starting tag relationship transformation")

	// Query old tag relationships
	query := `
		SELECT
			ct.chunk_id,
			ct.tag_chunk_id,
			ct.created_at,
			tag_chunk.content as tag_content
		FROM content_db.chunk_tags ct
		JOIN content_db.chunks tag_chunk ON ct.tag_chunk_id = tag_chunk.id
		ORDER BY ct.created_at
	`

	rows, err := sourceDB.QueryContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to query source tag relationships: %w", err)
	}
	defer rows.Close()

	// Process in batches
	batch := make([]models.ChunkTagRelation, 0, dt.config.BatchSize)
	processedCount := 0

	for rows.Next() {
		var (
			chunkID    string
			tagChunkID string
			createdAt  time.Time
			tagContent string
		)

		if err := rows.Scan(&chunkID, &tagChunkID, &createdAt, &tagContent); err != nil {
			dt.logger.Printf("Error scanning tag relationship: %v", err)
			continue
		}

		// Create tag relation for new schema
		tagRelation := models.ChunkTagRelation{
			SourceChunkID: chunkID,
			TagChunkID:    tagChunkID,
			CreatedAt:     createdAt,
		}

		batch = append(batch, tagRelation)

		// Process batch when full
		if len(batch) >= dt.config.BatchSize {
			if err := dt.insertTagRelationsBatch(ctx, targetDB, batch); err != nil {
				return fmt.Errorf("failed to insert tag relations batch: %w", err)
			}
			processedCount += len(batch)
			dt.logger.Printf("Processed %d tag relationships", processedCount)
			batch = batch[:0]
		}
	}

	// Process remaining relationships
	if len(batch) > 0 {
		if err := dt.insertTagRelationsBatch(ctx, targetDB, batch); err != nil {
			return fmt.Errorf("failed to insert final tag relations batch: %w", err)
		}
		processedCount += len(batch)
	}

	// Update tags array in chunks table
	if err := dt.updateChunkTagsArray(ctx, targetDB); err != nil {
		return fmt.Errorf("failed to update chunk tags array: %w", err)
	}

	dt.logger.Printf("Tag relationship transformation completed. Total processed: %d", processedCount)
	return nil
}

// insertTagRelationsBatch inserts a batch of tag relationships
func (dt *DataTransformer) insertTagRelationsBatch(ctx context.Context, targetDB *sql.DB, batch []models.ChunkTagRelation) error {
	if dt.config.DryRun {
		dt.logger.Printf("DRY RUN: Would insert %d tag relationships", len(batch))
		return nil
	}

	tx, err := targetDB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO chunk_tags (source_chunk_id, tag_chunk_id, created_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (source_chunk_id, tag_chunk_id) DO NOTHING
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare insert statement: %w", err)
	}
	defer stmt.Close()

	for _, relation := range batch {
		_, err = stmt.ExecContext(ctx, relation.SourceChunkID, relation.TagChunkID, relation.CreatedAt)
		if err != nil {
			return fmt.Errorf("failed to insert tag relation: %w", err)
		}
	}

	return tx.Commit()
}

// updateChunkTagsArray updates the tags array in the chunks table
func (dt *DataTransformer) updateChunkTagsArray(ctx context.Context, targetDB *sql.DB) error {
	dt.logger.Println("Updating chunk tags arrays")

	if dt.config.DryRun {
		dt.logger.Println("DRY RUN: Would update chunk tags arrays")
		return nil
	}

	// Update tags array based on chunk_tags relationships
	query := `
		UPDATE chunks
		SET tags = (
			SELECT COALESCE(jsonb_agg(ct.tag_chunk_id), '[]'::jsonb)
			FROM chunk_tags ct
			WHERE ct.source_chunk_id = chunks.chunk_id
		)
		WHERE EXISTS (
			SELECT 1 FROM chunk_tags ct WHERE ct.source_chunk_id = chunks.chunk_id
		)
	`

	result, err := targetDB.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to update chunk tags arrays: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	dt.logger.Printf("Updated tags arrays for %d chunks", rowsAffected)

	return nil
}

// BuildHierarchyRelationships builds the chunk_hierarchy table
func (dt *DataTransformer) BuildHierarchyRelationships(ctx context.Context, targetDB *sql.DB) error {
	dt.logger.Println("Building hierarchy relationships")

	if dt.config.DryRun {
		dt.logger.Println("DRY RUN: Would build hierarchy relationships")
		return nil
	}

	// Build hierarchy using recursive CTE
	query := `
		INSERT INTO chunk_hierarchy (ancestor_id, descendant_id, depth, path_ids)
		WITH RECURSIVE hierarchy AS (
			-- Self-references (depth = 0)
			SELECT
				chunk_id as ancestor_id,
				chunk_id as descendant_id,
				0 as depth,
				ARRAY[chunk_id] as path_ids
			FROM chunks

			UNION ALL

			-- Recursive part: find all ancestors
			SELECT
				c.chunk_id as ancestor_id,
				h.descendant_id,
				h.depth + 1,
				ARRAY[c.chunk_id] || h.path_ids
			FROM hierarchy h
			JOIN chunks c ON h.ancestor_id = c.parent
			WHERE c.parent IS NOT NULL AND h.depth < 100 -- Prevent infinite recursion
		)
		SELECT ancestor_id, descendant_id, depth, path_ids
		FROM hierarchy
		ON CONFLICT (ancestor_id, descendant_id) DO NOTHING
	`

	result, err := targetDB.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to build hierarchy relationships: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	dt.logger.Printf("Built hierarchy relationships: %d entries", rowsAffected)

	return nil
}

// Helper methods

func (dt *DataTransformer) determineIsPage(textID string, parentChunkID sql.NullString, isTemplate, isSlot bool) bool {
	// A chunk is a page if it's not a template/slot and has no parent (root level in a text)
	return !isTemplate && !isSlot && !parentChunkID.Valid
}

func (dt *DataTransformer) determineIsTag(content string) bool {
	// Simple heuristic: content starting with # or being very short might be a tag
	// This could be enhanced with more sophisticated tag detection logic
	trimmed := strings.TrimSpace(content)
	return strings.HasPrefix(trimmed, "#") && len(trimmed) < 50
}

func (dt *DataTransformer) derivePageReference(textID string, parentChunkID sql.NullString) *string {
	// For now, use text_id as page reference
	// This could be enhanced to find the actual root chunk ID for the text
	return &textID
}

func (dt *DataTransformer) nullStringToPointer(ns sql.NullString) *string {
	if ns.Valid {
		return &ns.String
	}
	return nil
}

// ValidationResult represents the result of data transformation validation
type ValidationResult struct {
	SourceCount      int64                  `json:"source_count"`
	TargetCount      int64                  `json:"target_count"`
	ValidationErrors []ValidationError     `json:"validation_errors"`
	Summary          map[string]interface{} `json:"summary"`
}

// ValidationError represents a validation error found during transformation
type ValidationError struct {
	Type        string                 `json:"type"`
	Description string                 `json:"description"`
	Context     map[string]interface{} `json:"context"`
	Severity    string                 `json:"severity"` // "warning", "error", "critical"
}

// ValidateTransformation validates the transformation results
func (dt *DataTransformer) ValidateTransformation(ctx context.Context, sourceDB, targetDB *sql.DB) (*ValidationResult, error) {
	dt.logger.Println("Validating transformation results")

	result := &ValidationResult{
		ValidationErrors: make([]ValidationError, 0),
		Summary:          make(map[string]interface{}),
	}

	// Count source records
	err := sourceDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM content_db.chunks").Scan(&result.SourceCount)
	if err != nil {
		return nil, fmt.Errorf("failed to count source chunks: %w", err)
	}

	// Count target records
	err = targetDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM chunks").Scan(&result.TargetCount)
	if err != nil {
		return nil, fmt.Errorf("failed to count target chunks: %w", err)
	}

	// Validate counts match
	if result.SourceCount != result.TargetCount {
		result.ValidationErrors = append(result.ValidationErrors, ValidationError{
			Type:        "count_mismatch",
			Description: fmt.Sprintf("Source count (%d) doesn't match target count (%d)", result.SourceCount, result.TargetCount),
			Severity:    "critical",
			Context: map[string]interface{}{
				"source_count": result.SourceCount,
				"target_count": result.TargetCount,
				"difference":   result.SourceCount - result.TargetCount,
			},
		})
	}

	// Validate chunk_tags relationships
	if err := dt.validateTagRelationships(ctx, sourceDB, targetDB, result); err != nil {
		return nil, fmt.Errorf("failed to validate tag relationships: %w", err)
	}

	// Validate hierarchy relationships
	if err := dt.validateHierarchyRelationships(ctx, targetDB, result); err != nil {
		return nil, fmt.Errorf("failed to validate hierarchy relationships: %w", err)
	}

	result.Summary["total_validation_errors"] = len(result.ValidationErrors)
	result.Summary["critical_errors"] = dt.countErrorsBySeverity(result.ValidationErrors, "critical")
	result.Summary["errors"] = dt.countErrorsBySeverity(result.ValidationErrors, "error")
	result.Summary["warnings"] = dt.countErrorsBySeverity(result.ValidationErrors, "warning")

	dt.logger.Printf("Validation completed. Found %d validation issues", len(result.ValidationErrors))
	return result, nil
}

func (dt *DataTransformer) validateTagRelationships(ctx context.Context, sourceDB, targetDB *sql.DB, result *ValidationResult) error {
	var sourceTagCount, targetTagCount int64

	// Count source tag relationships
	err := sourceDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM content_db.chunk_tags").Scan(&sourceTagCount)
	if err != nil {
		return err
	}

	// Count target tag relationships
	err = targetDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM chunk_tags").Scan(&targetTagCount)
	if err != nil {
		return err
	}

	if sourceTagCount != targetTagCount {
		result.ValidationErrors = append(result.ValidationErrors, ValidationError{
			Type:        "tag_relationship_count_mismatch",
			Description: fmt.Sprintf("Source tag relationships (%d) don't match target (%d)", sourceTagCount, targetTagCount),
			Severity:    "error",
			Context: map[string]interface{}{
				"source_count": sourceTagCount,
				"target_count": targetTagCount,
			},
		})
	}

	return nil
}

func (dt *DataTransformer) validateHierarchyRelationships(ctx context.Context, targetDB *sql.DB, result *ValidationResult) error {
	// Check for orphaned hierarchy relationships
	query := `
		SELECT COUNT(*)
		FROM chunk_hierarchy ch
		LEFT JOIN chunks c1 ON ch.ancestor_id = c1.chunk_id
		LEFT JOIN chunks c2 ON ch.descendant_id = c2.chunk_id
		WHERE c1.chunk_id IS NULL OR c2.chunk_id IS NULL
	`

	var orphanedCount int64
	err := targetDB.QueryRowContext(ctx, query).Scan(&orphanedCount)
	if err != nil {
		return err
	}

	if orphanedCount > 0 {
		result.ValidationErrors = append(result.ValidationErrors, ValidationError{
			Type:        "orphaned_hierarchy_relationships",
			Description: fmt.Sprintf("Found %d orphaned hierarchy relationships", orphanedCount),
			Severity:    "error",
			Context: map[string]interface{}{
				"orphaned_count": orphanedCount,
			},
		})
	}

	return nil
}

func (dt *DataTransformer) countErrorsBySeverity(errors []ValidationError, severity string) int {
	count := 0
	for _, err := range errors {
		if err.Severity == severity {
			count++
		}
	}
	return count
}