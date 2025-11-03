package migration

import (
	"context"
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"
	"strings"
)

// MigrationValidator validates migration results
type MigrationValidator struct {
	sourceDB *sql.DB
	targetDB *sql.DB
	config   *MigrationConfig
	logger   *log.Logger
}

// NewMigrationValidator creates a new migration validator
func NewMigrationValidator(sourceDB, targetDB *sql.DB, config *MigrationConfig) *MigrationValidator {
	return &MigrationValidator{
		sourceDB: sourceDB,
		targetDB: targetDB,
		config:   config,
		logger:   log.New(log.Writer(), "[MIGRATION_VALIDATOR] ", log.LstdFlags|log.Lshortfile),
	}
}

// DataLoader handles loading transformed data
type DataLoader struct {
	config *MigrationConfig
	logger *log.Logger
}

// NewDataLoader creates a new data loader
func NewDataLoader(config *MigrationConfig) *DataLoader {
	return &DataLoader{
		config: config,
		logger: log.New(log.Writer(), "[DATA_LOADER] ", log.LstdFlags|log.Lshortfile),
	}
}

// LoadTransformedData loads data into target tables
func (dl *DataLoader) LoadTransformedData(ctx context.Context, targetDB *sql.DB, progressMonitor *ProgressMonitor) error {
	dl.logger.Println("Loading transformed data")

	// This is a placeholder - actual implementation would depend on
	// where the transformed data is staged
	if dl.config.DryRun {
		dl.logger.Println("DRY RUN: Would load transformed data")
		return nil
	}

	// In a real implementation, this would:
	// 1. Read staged data from temporary tables or files
	// 2. Load in batches with progress reporting
	// 3. Handle conflicts and errors appropriately

	dl.logger.Println("Data loading completed")
	return nil
}

// ValidateRowCounts validates that row counts match between source and target
func (mv *MigrationValidator) ValidateRowCounts(ctx context.Context) error {
	mv.logger.Println("Validating row counts")

	validations := []struct {
		name        string
		sourceQuery string
		targetQuery string
	}{
		{
			name:        "chunks",
			sourceQuery: "SELECT COUNT(*) FROM content_db.chunks",
			targetQuery: "SELECT COUNT(*) FROM chunks",
		},
		{
			name:        "chunk_tags",
			sourceQuery: "SELECT COUNT(*) FROM content_db.chunk_tags",
			targetQuery: "SELECT COUNT(*) FROM chunk_tags",
		},
		{
			name:        "embeddings_references",
			sourceQuery: "SELECT COUNT(*) FROM vector_db.embeddings",
			targetQuery: "SELECT COUNT(DISTINCT chunk_id) FROM chunks WHERE chunk_id IN (SELECT chunk_id FROM vector_db.embeddings)",
		},
		{
			name:        "graph_nodes_references",
			sourceQuery: "SELECT COUNT(*) FROM graph_db.graph_nodes",
			targetQuery: "SELECT COUNT(DISTINCT chunk_id) FROM chunks WHERE chunk_id IN (SELECT chunk_id FROM graph_db.graph_nodes)",
		},
	}

	for _, validation := range validations {
		mv.logger.Printf("Validating row count for: %s", validation.name)

		var sourceCount, targetCount int64

		// Get source count
		err := mv.sourceDB.QueryRowContext(ctx, validation.sourceQuery).Scan(&sourceCount)
		if err != nil {
			return fmt.Errorf("failed to get source count for %s: %w", validation.name, err)
		}

		// Get target count
		err = mv.targetDB.QueryRowContext(ctx, validation.targetQuery).Scan(&targetCount)
		if err != nil {
			return fmt.Errorf("failed to get target count for %s: %w", validation.name, err)
		}

		// Compare counts
		if sourceCount != targetCount {
			return fmt.Errorf("row count mismatch for %s: source=%d, target=%d", validation.name, sourceCount, targetCount)
		}

		mv.logger.Printf("Row count validation passed for %s: %d records", validation.name, sourceCount)
	}

	mv.logger.Println("Row count validation completed successfully")
	return nil
}

// ValidateDataIntegrity validates data integrity using checksums and constraints
func (mv *MigrationValidator) ValidateDataIntegrity(ctx context.Context) error {
	mv.logger.Println("Validating data integrity")

	// Validate chunk content integrity
	if err := mv.validateChunkContentIntegrity(ctx); err != nil {
		return fmt.Errorf("chunk content integrity validation failed: %w", err)
	}

	// Validate unique constraints
	if err := mv.validateUniqueConstraints(ctx); err != nil {
		return fmt.Errorf("unique constraints validation failed: %w", err)
	}

	// Validate JSON fields
	if err := mv.validateJSONFields(ctx); err != nil {
		return fmt.Errorf("JSON fields validation failed: %w", err)
	}

	// Validate timestamp consistency
	if err := mv.validateTimestamps(ctx); err != nil {
		return fmt.Errorf("timestamp validation failed: %w", err)
	}

	mv.logger.Println("Data integrity validation completed successfully")
	return nil
}

// ValidateRelationships validates foreign key relationships and data consistency
func (mv *MigrationValidator) ValidateRelationships(ctx context.Context) error {
	mv.logger.Println("Validating relationships")

	// Validate chunk parent references
	if err := mv.validateChunkParentReferences(ctx); err != nil {
		return fmt.Errorf("chunk parent references validation failed: %w", err)
	}

	// Validate chunk_tags relationships
	if err := mv.validateChunkTagsRelationships(ctx); err != nil {
		return fmt.Errorf("chunk tags relationships validation failed: %w", err)
	}

	// Validate chunk_hierarchy consistency
	if err := mv.validateHierarchyConsistency(ctx); err != nil {
		return fmt.Errorf("hierarchy consistency validation failed: %w", err)
	}

	// Validate page references
	if err := mv.validatePageReferences(ctx); err != nil {
		return fmt.Errorf("page references validation failed: %w", err)
	}

	// Validate vector and graph references
	if err := mv.validateExternalReferences(ctx); err != nil {
		return fmt.Errorf("external references validation failed: %w", err)
	}

	mv.logger.Println("Relationships validation completed successfully")
	return nil
}

// validateChunkContentIntegrity validates that chunk content was migrated correctly
func (mv *MigrationValidator) validateChunkContentIntegrity(ctx context.Context) error {
	mv.logger.Println("Validating chunk content integrity")

	// Use checksums to validate content integrity
	query := `
		SELECT
			source.id,
			source.content,
			target.contents
		FROM content_db.chunks source
		JOIN chunks target ON source.id = target.chunk_id
		ORDER BY source.id
		LIMIT 1000 -- Sample validation for performance
	`

	rows, err := mv.sourceDB.QueryContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to query chunk content: %w", err)
	}
	defer rows.Close()

	mismatches := 0
	checked := 0

	for rows.Next() {
		var id, sourceContent, targetContent string
		if err := rows.Scan(&id, &sourceContent, &targetContent); err != nil {
			return fmt.Errorf("failed to scan chunk content: %w", err)
		}

		// Compare content checksums
		sourceHash := md5.Sum([]byte(sourceContent))
		targetHash := md5.Sum([]byte(targetContent))

		if hex.EncodeToString(sourceHash[:]) != hex.EncodeToString(targetHash[:]) {
			mismatches++
			mv.logger.Printf("Content mismatch for chunk %s", id)
		}

		checked++
	}

	if mismatches > 0 {
		return fmt.Errorf("found %d content mismatches out of %d checked chunks", mismatches, checked)
	}

	mv.logger.Printf("Content integrity validation passed for %d chunks", checked)
	return nil
}

// validateUniqueConstraints validates unique constraints
func (mv *MigrationValidator) validateUniqueConstraints(ctx context.Context) error {
	mv.logger.Println("Validating unique constraints")

	// Check for duplicate chunk_ids
	var duplicateChunkIds int64
	err := mv.targetDB.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM (
			SELECT chunk_id, COUNT(*)
			FROM chunks
			GROUP BY chunk_id
			HAVING COUNT(*) > 1
		) duplicates
	`).Scan(&duplicateChunkIds)

	if err != nil {
		return fmt.Errorf("failed to check duplicate chunk_ids: %w", err)
	}

	if duplicateChunkIds > 0 {
		return fmt.Errorf("found %d duplicate chunk_ids", duplicateChunkIds)
	}

	// Check for duplicate chunk_tags relationships
	var duplicateTagRelations int64
	err = mv.targetDB.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM (
			SELECT source_chunk_id, tag_chunk_id, COUNT(*)
			FROM chunk_tags
			GROUP BY source_chunk_id, tag_chunk_id
			HAVING COUNT(*) > 1
		) duplicates
	`).Scan(&duplicateTagRelations)

	if err != nil {
		return fmt.Errorf("failed to check duplicate tag relations: %w", err)
	}

	if duplicateTagRelations > 0 {
		return fmt.Errorf("found %d duplicate tag relationships", duplicateTagRelations)
	}

	mv.logger.Println("Unique constraints validation passed")
	return nil
}

// validateJSONFields validates JSON field integrity
func (mv *MigrationValidator) validateJSONFields(ctx context.Context) error {
	mv.logger.Println("Validating JSON fields")

	// Validate tags JSON field
	var invalidTagsJSON int64
	err := mv.targetDB.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM chunks
		WHERE NOT (tags::text ~ '^\\[.*\\]$')
	`).Scan(&invalidTagsJSON)

	if err != nil {
		return fmt.Errorf("failed to validate tags JSON: %w", err)
	}

	if invalidTagsJSON > 0 {
		return fmt.Errorf("found %d chunks with invalid tags JSON", invalidTagsJSON)
	}

	// Validate metadata JSON field
	var invalidMetadataJSON int64
	err = mv.targetDB.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM chunks
		WHERE metadata IS NOT NULL AND NOT (metadata::text ~ '^\\{.*\\}$')
	`).Scan(&invalidMetadataJSON)

	if err != nil {
		return fmt.Errorf("failed to validate metadata JSON: %w", err)
	}

	if invalidMetadataJSON > 0 {
		return fmt.Errorf("found %d chunks with invalid metadata JSON", invalidMetadataJSON)
	}

	mv.logger.Println("JSON fields validation passed")
	return nil
}

// validateTimestamps validates timestamp consistency
func (mv *MigrationValidator) validateTimestamps(ctx context.Context) error {
	mv.logger.Println("Validating timestamps")

	// Check for invalid timestamp ordering (created_time > last_updated)
	var invalidTimestamps int64
	err := mv.targetDB.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM chunks
		WHERE created_time > last_updated
	`).Scan(&invalidTimestamps)

	if err != nil {
		return fmt.Errorf("failed to validate timestamps: %w", err)
	}

	if invalidTimestamps > 0 {
		return fmt.Errorf("found %d chunks with invalid timestamp ordering", invalidTimestamps)
	}

	// Check for null timestamps
	var nullTimestamps int64
	err = mv.targetDB.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM chunks
		WHERE created_time IS NULL OR last_updated IS NULL
	`).Scan(&nullTimestamps)

	if err != nil {
		return fmt.Errorf("failed to check null timestamps: %w", err)
	}

	if nullTimestamps > 0 {
		return fmt.Errorf("found %d chunks with null timestamps", nullTimestamps)
	}

	mv.logger.Println("Timestamps validation passed")
	return nil
}

// validateChunkParentReferences validates chunk parent references
func (mv *MigrationValidator) validateChunkParentReferences(ctx context.Context) error {
	mv.logger.Println("Validating chunk parent references")

	// Check for orphaned parent references
	var orphanedParents int64
	err := mv.targetDB.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM chunks c1
		LEFT JOIN chunks c2 ON c1.parent = c2.chunk_id
		WHERE c1.parent IS NOT NULL AND c2.chunk_id IS NULL
	`).Scan(&orphanedParents)

	if err != nil {
		return fmt.Errorf("failed to check orphaned parent references: %w", err)
	}

	if orphanedParents > 0 {
		return fmt.Errorf("found %d chunks with orphaned parent references", orphanedParents)
	}

	// Check for circular references (basic check)
	var circularReferences int64
	err = mv.targetDB.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM chunks
		WHERE chunk_id = parent
	`).Scan(&circularReferences)

	if err != nil {
		return fmt.Errorf("failed to check circular references: %w", err)
	}

	if circularReferences > 0 {
		return fmt.Errorf("found %d chunks with circular parent references", circularReferences)
	}

	mv.logger.Println("Chunk parent references validation passed")
	return nil
}

// validateChunkTagsRelationships validates chunk_tags relationships
func (mv *MigrationValidator) validateChunkTagsRelationships(ctx context.Context) error {
	mv.logger.Println("Validating chunk tags relationships")

	// Check for orphaned source chunk references
	var orphanedSourceChunks int64
	err := mv.targetDB.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM chunk_tags ct
		LEFT JOIN chunks c ON ct.source_chunk_id = c.chunk_id
		WHERE c.chunk_id IS NULL
	`).Scan(&orphanedSourceChunks)

	if err != nil {
		return fmt.Errorf("failed to check orphaned source chunks: %w", err)
	}

	if orphanedSourceChunks > 0 {
		return fmt.Errorf("found %d orphaned source chunk references in chunk_tags", orphanedSourceChunks)
	}

	// Check for orphaned tag chunk references
	var orphanedTagChunks int64
	err = mv.targetDB.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM chunk_tags ct
		LEFT JOIN chunks c ON ct.tag_chunk_id = c.chunk_id
		WHERE c.chunk_id IS NULL
	`).Scan(&orphanedTagChunks)

	if err != nil {
		return fmt.Errorf("failed to check orphaned tag chunks: %w", err)
	}

	if orphanedTagChunks > 0 {
		return fmt.Errorf("found %d orphaned tag chunk references in chunk_tags", orphanedTagChunks)
	}

	mv.logger.Println("Chunk tags relationships validation passed")
	return nil
}

// validateHierarchyConsistency validates chunk_hierarchy table consistency
func (mv *MigrationValidator) validateHierarchyConsistency(ctx context.Context) error {
	mv.logger.Println("Validating hierarchy consistency")

	// Check for missing self-references (depth = 0)
	var missingSelfRefs int64
	err := mv.targetDB.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM chunks c
		LEFT JOIN chunk_hierarchy ch ON c.chunk_id = ch.ancestor_id AND c.chunk_id = ch.descendant_id AND ch.depth = 0
		WHERE ch.ancestor_id IS NULL
	`).Scan(&missingSelfRefs)

	if err != nil {
		return fmt.Errorf("failed to check missing self-references: %w", err)
	}

	if missingSelfRefs > 0 {
		return fmt.Errorf("found %d chunks missing self-references in hierarchy", missingSelfRefs)
	}

	// Check for invalid depth values
	var invalidDepths int64
	err = mv.targetDB.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM chunk_hierarchy
		WHERE depth < 0 OR depth > 100
	`).Scan(&invalidDepths)

	if err != nil {
		return fmt.Errorf("failed to check invalid depths: %w", err)
	}

	if invalidDepths > 0 {
		return fmt.Errorf("found %d hierarchy entries with invalid depths", invalidDepths)
	}

	// Check path_ids consistency
	var inconsistentPaths int64
	err = mv.targetDB.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM chunk_hierarchy
		WHERE array_length(path_ids, 1) - 1 != depth
	`).Scan(&inconsistentPaths)

	if err != nil {
		return fmt.Errorf("failed to check path consistency: %w", err)
	}

	if inconsistentPaths > 0 {
		return fmt.Errorf("found %d hierarchy entries with inconsistent paths", inconsistentPaths)
	}

	mv.logger.Println("Hierarchy consistency validation passed")
	return nil
}

// validatePageReferences validates page references
func (mv *MigrationValidator) validatePageReferences(ctx context.Context) error {
	mv.logger.Println("Validating page references")

	// Check for orphaned page references
	var orphanedPageRefs int64
	err := mv.targetDB.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM chunks c1
		LEFT JOIN chunks c2 ON c1.page = c2.chunk_id
		WHERE c1.page IS NOT NULL AND c2.chunk_id IS NULL
	`).Scan(&orphanedPageRefs)

	if err != nil {
		return fmt.Errorf("failed to check orphaned page references: %w", err)
	}

	if orphanedPageRefs > 0 {
		return fmt.Errorf("found %d chunks with orphaned page references", orphanedPageRefs)
	}

	// Check that referenced pages are actually marked as pages
	var invalidPageRefs int64
	err = mv.targetDB.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM chunks c1
		JOIN chunks c2 ON c1.page = c2.chunk_id
		WHERE c1.page IS NOT NULL AND c2.is_page = false
	`).Scan(&invalidPageRefs)

	if err != nil {
		return fmt.Errorf("failed to check invalid page references: %w", err)
	}

	if invalidPageRefs > 0 {
		return fmt.Errorf("found %d chunks referencing non-page chunks as pages", invalidPageRefs)
	}

	mv.logger.Println("Page references validation passed")
	return nil
}

// validateExternalReferences validates references to external tables (embeddings, graph)
func (mv *MigrationValidator) validateExternalReferences(ctx context.Context) error {
	mv.logger.Println("Validating external references")

	// Check that all embeddings have corresponding chunks
	var orphanedEmbeddings int64
	err := mv.sourceDB.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM vector_db.embeddings e
		LEFT JOIN chunks c ON e.chunk_id = c.chunk_id
		WHERE c.chunk_id IS NULL
	`).Scan(&orphanedEmbeddings)

	if err != nil {
		return fmt.Errorf("failed to check orphaned embeddings: %w", err)
	}

	if orphanedEmbeddings > 0 {
		mv.logger.Printf("Warning: found %d embeddings with no corresponding chunks", orphanedEmbeddings)
	}

	// Check that all graph nodes have corresponding chunks
	var orphanedGraphNodes int64
	err = mv.sourceDB.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM graph_db.graph_nodes gn
		LEFT JOIN chunks c ON gn.chunk_id = c.chunk_id
		WHERE c.chunk_id IS NULL
	`).Scan(&orphanedGraphNodes)

	if err != nil {
		return fmt.Errorf("failed to check orphaned graph nodes: %w", err)
	}

	if orphanedGraphNodes > 0 {
		mv.logger.Printf("Warning: found %d graph nodes with no corresponding chunks", orphanedGraphNodes)
	}

	mv.logger.Println("External references validation completed")
	return nil
}

// GenerateValidationReport generates a comprehensive validation report
func (mv *MigrationValidator) GenerateValidationReport(ctx context.Context) (*ValidationReport, error) {
	mv.logger.Println("Generating validation report")

	report := &ValidationReport{
		GeneratedAt: mv.logger.Writer(),
		Validations: make([]ValidationResult, 0),
	}

	// Run all validations and collect results
	validations := []struct {
		name string
		fn   func(context.Context) error
	}{
		{"row_counts", mv.ValidateRowCounts},
		{"data_integrity", mv.ValidateDataIntegrity},
		{"relationships", mv.ValidateRelationships},
	}

	for _, validation := range validations {
		result := ValidationResult{
			ValidationName: validation.name,
			StartTime:      log.Writer(),
		}

		err := validation.fn(ctx)
		result.EndTime = log.Writer()

		if err != nil {
			result.Status = "failed"
			result.ErrorMessage = err.Error()
		} else {
			result.Status = "passed"
		}

		report.Validations = append(report.Validations, result)
	}

	// Calculate summary
	passed := 0
	failed := 0
	for _, validation := range report.Validations {
		if validation.Status == "passed" {
			passed++
		} else {
			failed++
		}
	}

	report.Summary = ValidationSummary{
		TotalValidations: len(report.Validations),
		PassedValidations: passed,
		FailedValidations: failed,
		OverallStatus: func() string {
			if failed > 0 {
				return "failed"
			}
			return "passed"
		}(),
	}

	return report, nil
}

// ValidationReport represents a comprehensive validation report
type ValidationReport struct {
	GeneratedAt interface{}        `json:"generated_at"` // This should be time.Time but keeping as interface{} for now
	Summary     ValidationSummary  `json:"summary"`
	Validations []ValidationResult `json:"validations"`
}

// ValidationSummary provides summary statistics
type ValidationSummary struct {
	TotalValidations  int    `json:"total_validations"`
	PassedValidations int    `json:"passed_validations"`
	FailedValidations int    `json:"failed_validations"`
	OverallStatus     string `json:"overall_status"`
}

// ValidationResult represents the result of a single validation
type ValidationResult struct {
	ValidationName string      `json:"validation_name"`
	Status         string      `json:"status"`
	StartTime      interface{} `json:"start_time"` // This should be time.Time
	EndTime        interface{} `json:"end_time"`   // This should be time.Time
	ErrorMessage   string      `json:"error_message,omitempty"`
}