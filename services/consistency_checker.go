package services

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/lib/pq"
)

// ConsistencyChecker provides data consistency checking and repair functionality
type ConsistencyChecker interface {
	// Tag consistency
	CheckTagConsistency(ctx context.Context) ([]ConsistencyError, error)
	RepairTagConsistency(ctx context.Context, chunkID string) error
	RepairAllTagConsistencies(ctx context.Context) (int, error)
	
	// Hierarchy consistency
	CheckHierarchyConsistency(ctx context.Context) ([]ConsistencyError, error)
	RepairHierarchyConsistency(ctx context.Context, chunkID string) error
	RepairAllHierarchyConsistencies(ctx context.Context) (int, error)
	
	// Search cache consistency
	CheckSearchCacheConsistency(ctx context.Context) ([]ConsistencyError, error)
	CleanupExpiredSearchCache(ctx context.Context) (int, error)
	
	// Overall consistency check
	CheckAllConsistency(ctx context.Context) (*ConsistencyReport, error)
	RepairAllInconsistencies(ctx context.Context) (*RepairReport, error)
	
	// Validation and migration
	ValidateDataIntegrity(ctx context.Context) (*IntegrityReport, error)
	VerifyMigration(ctx context.Context, sourceTable, targetTable string) (*MigrationReport, error)
}

// ConsistencyError represents a data consistency error
type ConsistencyError struct {
	Type        string                 `json:"type"`
	ChunkID     string                 `json:"chunk_id"`
	Table       string                 `json:"table"`
	Description string                 `json:"description"`
	Details     map[string]interface{} `json:"details"`
	Severity    string                 `json:"severity"` // "low", "medium", "high", "critical"
	Timestamp   time.Time              `json:"timestamp"`
}

// ConsistencyReport represents the overall consistency status
type ConsistencyReport struct {
	CheckTime       time.Time           `json:"check_time"`
	TotalErrors     int                 `json:"total_errors"`
	ErrorsByType    map[string]int      `json:"errors_by_type"`
	ErrorsBySeverity map[string]int     `json:"errors_by_severity"`
	Errors          []ConsistencyError  `json:"errors"`
	Recommendations []string            `json:"recommendations"`
}

// RepairReport represents the results of repair operations
type RepairReport struct {
	RepairTime      time.Time          `json:"repair_time"`
	TotalRepaired   int                `json:"total_repaired"`
	RepairedByType  map[string]int     `json:"repaired_by_type"`
	FailedRepairs   []ConsistencyError `json:"failed_repairs"`
	Duration        time.Duration      `json:"duration"`
}

// IntegrityReport represents data integrity validation results
type IntegrityReport struct {
	CheckTime       time.Time                  `json:"check_time"`
	TablesChecked   []string                   `json:"tables_checked"`
	TotalRecords    map[string]int64           `json:"total_records"`
	IntegrityIssues []IntegrityIssue           `json:"integrity_issues"`
	Recommendations []string                   `json:"recommendations"`
	IsHealthy       bool                       `json:"is_healthy"`
}

// IntegrityIssue represents a data integrity issue
type IntegrityIssue struct {
	Type        string                 `json:"type"`
	Table       string                 `json:"table"`
	Description string                 `json:"description"`
	Count       int64                  `json:"count"`
	Details     map[string]interface{} `json:"details"`
	Severity    string                 `json:"severity"`
}

// MigrationReport represents migration verification results
type MigrationReport struct {
	SourceTable     string            `json:"source_table"`
	TargetTable     string            `json:"target_table"`
	SourceCount     int64             `json:"source_count"`
	TargetCount     int64             `json:"target_count"`
	MissingRecords  []string          `json:"missing_records"`
	ExtraRecords    []string          `json:"extra_records"`
	DataMismatches  []DataMismatch    `json:"data_mismatches"`
	IsComplete      bool              `json:"is_complete"`
	CompletionRate  float64           `json:"completion_rate"`
}

// DataMismatch represents a data mismatch between source and target
type DataMismatch struct {
	RecordID    string                 `json:"record_id"`
	Field       string                 `json:"field"`
	SourceValue interface{}            `json:"source_value"`
	TargetValue interface{}            `json:"target_value"`
	Details     map[string]interface{} `json:"details"`
}

// DatabaseConsistencyChecker implements ConsistencyChecker using database operations
type DatabaseConsistencyChecker struct {
	db     *sql.DB
	logger Logger
}

// NewDatabaseConsistencyChecker creates a new database consistency checker
func NewDatabaseConsistencyChecker(db *sql.DB, logger Logger) *DatabaseConsistencyChecker {
	if logger == nil {
		logger = NewDefaultLogger()
	}
	
	return &DatabaseConsistencyChecker{
		db:     db,
		logger: logger,
	}
}

// CheckTagConsistency checks consistency between main table tags and chunk_tags auxiliary table
func (cc *DatabaseConsistencyChecker) CheckTagConsistency(ctx context.Context) ([]ConsistencyError, error) {
	var errors []ConsistencyError
	
	// Check for chunks with tags in main table but missing in auxiliary table
	query := `
		SELECT c.chunk_id, c.tags, COALESCE(array_agg(ct.tag_chunk_id) FILTER (WHERE ct.tag_chunk_id IS NOT NULL), '{}') as aux_tags
		FROM chunks c
		LEFT JOIN chunk_tags ct ON c.chunk_id = ct.source_chunk_id
		WHERE c.tags IS NOT NULL AND jsonb_array_length(c.tags) > 0
		GROUP BY c.chunk_id, c.tags
		HAVING c.tags::jsonb != COALESCE(array_to_json(array_agg(ct.tag_chunk_id) FILTER (WHERE ct.tag_chunk_id IS NOT NULL))::jsonb, '[]'::jsonb)
	`
	
	rows, err := cc.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to check tag consistency: %w", err)
	}
	defer rows.Close()
	
	for rows.Next() {
		var chunkID string
		var mainTags, auxTags []string
		
		if err := rows.Scan(&chunkID, pq.Array(&mainTags), pq.Array(&auxTags)); err != nil {
			cc.logger.Error("Failed to scan tag consistency row", err)
			continue
		}
		
		errors = append(errors, ConsistencyError{
			Type:        "tag_mismatch",
			ChunkID:     chunkID,
			Table:       "chunk_tags",
			Description: "Tags in main table don't match auxiliary table",
			Details: map[string]interface{}{
				"main_tags": mainTags,
				"aux_tags":  auxTags,
			},
			Severity:  "medium",
			Timestamp: time.Now(),
		})
	}
	
	// Check for orphaned records in chunk_tags table
	orphanQuery := `
		SELECT ct.source_chunk_id, ct.tag_chunk_id, COUNT(*) as count
		FROM chunk_tags ct
		LEFT JOIN chunks c ON ct.source_chunk_id = c.chunk_id
		WHERE c.chunk_id IS NULL
		GROUP BY ct.source_chunk_id, ct.tag_chunk_id
	`
	
	rows, err = cc.db.QueryContext(ctx, orphanQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to check orphaned tag records: %w", err)
	}
	defer rows.Close()
	
	for rows.Next() {
		var sourceChunkID, tagChunkID string
		var count int
		
		if err := rows.Scan(&sourceChunkID, &tagChunkID, &count); err != nil {
			cc.logger.Error("Failed to scan orphaned tag row", err)
			continue
		}
		
		errors = append(errors, ConsistencyError{
			Type:        "orphaned_tag_relation",
			ChunkID:     sourceChunkID,
			Table:       "chunk_tags",
			Description: "Orphaned tag relationship in auxiliary table",
			Details: map[string]interface{}{
				"tag_chunk_id": tagChunkID,
				"count":        count,
			},
			Severity:  "high",
			Timestamp: time.Now(),
		})
	}
	
	return errors, nil
}

// RepairTagConsistency repairs tag consistency for a specific chunk
func (cc *DatabaseConsistencyChecker) RepairTagConsistency(ctx context.Context, chunkID string) error {
	tx, err := cc.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()
	
	// Get tags from main table
	var tags []string
	err = tx.QueryRowContext(ctx, "SELECT COALESCE(tags, '[]'::jsonb) FROM chunks WHERE chunk_id = $1", chunkID).Scan(pq.Array(&tags))
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("chunk not found: %s", chunkID)
		}
		return fmt.Errorf("failed to get chunk tags: %w", err)
	}
	
	// Clear existing tag relationships
	_, err = tx.ExecContext(ctx, "DELETE FROM chunk_tags WHERE source_chunk_id = $1", chunkID)
	if err != nil {
		return fmt.Errorf("failed to clear existing tag relationships: %w", err)
	}
	
	// Insert new tag relationships
	for _, tagID := range tags {
		if tagID != "" {
			_, err = tx.ExecContext(ctx, 
				"INSERT INTO chunk_tags (source_chunk_id, tag_chunk_id) VALUES ($1, $2) ON CONFLICT DO NOTHING",
				chunkID, tagID)
			if err != nil {
				return fmt.Errorf("failed to insert tag relationship: %w", err)
			}
		}
	}
	
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	
	cc.logger.Info("Repaired tag consistency", String("chunk_id", chunkID), Int("tag_count", len(tags)))
	return nil
}

// RepairAllTagConsistencies repairs all tag consistency issues
func (cc *DatabaseConsistencyChecker) RepairAllTagConsistencies(ctx context.Context) (int, error) {
	errors, err := cc.CheckTagConsistency(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to check tag consistency: %w", err)
	}
	
	repaired := 0
	for _, consistencyError := range errors {
		if consistencyError.Type == "tag_mismatch" {
			if err := cc.RepairTagConsistency(ctx, consistencyError.ChunkID); err != nil {
				cc.logger.Error("Failed to repair tag consistency", err, String("chunk_id", consistencyError.ChunkID))
			} else {
				repaired++
			}
		} else if consistencyError.Type == "orphaned_tag_relation" {
			// Remove orphaned relationships
			_, err := cc.db.ExecContext(ctx, 
				"DELETE FROM chunk_tags WHERE source_chunk_id = $1", 
				consistencyError.ChunkID)
			if err != nil {
				cc.logger.Error("Failed to remove orphaned tag relationship", err, String("chunk_id", consistencyError.ChunkID))
			} else {
				repaired++
			}
		}
	}
	
	return repaired, nil
}

// CheckHierarchyConsistency checks consistency of hierarchical relationships
func (cc *DatabaseConsistencyChecker) CheckHierarchyConsistency(ctx context.Context) ([]ConsistencyError, error) {
	var errors []ConsistencyError
	
	// Check for missing hierarchy records
	query := `
		SELECT c.chunk_id, c.parent
		FROM chunks c
		LEFT JOIN chunk_hierarchy ch ON c.chunk_id = ch.descendant_id AND ch.depth = 0
		WHERE ch.descendant_id IS NULL
	`
	
	rows, err := cc.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to check hierarchy consistency: %w", err)
	}
	defer rows.Close()
	
	for rows.Next() {
		var chunkID string
		var parent sql.NullString
		
		if err := rows.Scan(&chunkID, &parent); err != nil {
			cc.logger.Error("Failed to scan hierarchy consistency row", err)
			continue
		}
		
		errors = append(errors, ConsistencyError{
			Type:        "missing_hierarchy_record",
			ChunkID:     chunkID,
			Table:       "chunk_hierarchy",
			Description: "Missing self-reference in hierarchy table",
			Details: map[string]interface{}{
				"parent": parent.String,
			},
			Severity:  "medium",
			Timestamp: time.Now(),
		})
	}
	
	// Check for orphaned hierarchy records
	orphanQuery := `
		SELECT ch.descendant_id, ch.ancestor_id, ch.depth
		FROM chunk_hierarchy ch
		LEFT JOIN chunks c ON ch.descendant_id = c.chunk_id
		WHERE c.chunk_id IS NULL
	`
	
	rows, err = cc.db.QueryContext(ctx, orphanQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to check orphaned hierarchy records: %w", err)
	}
	defer rows.Close()
	
	for rows.Next() {
		var descendantID, ancestorID string
		var depth int
		
		if err := rows.Scan(&descendantID, &ancestorID, &depth); err != nil {
			cc.logger.Error("Failed to scan orphaned hierarchy row", err)
			continue
		}
		
		errors = append(errors, ConsistencyError{
			Type:        "orphaned_hierarchy_record",
			ChunkID:     descendantID,
			Table:       "chunk_hierarchy",
			Description: "Orphaned hierarchy record",
			Details: map[string]interface{}{
				"ancestor_id": ancestorID,
				"depth":       depth,
			},
			Severity:  "high",
			Timestamp: time.Now(),
		})
	}
	
	return errors, nil
}

// RepairHierarchyConsistency repairs hierarchy consistency for a specific chunk
func (cc *DatabaseConsistencyChecker) RepairHierarchyConsistency(ctx context.Context, chunkID string) error {
	tx, err := cc.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()
	
	// Clear existing hierarchy records for this chunk
	_, err = tx.ExecContext(ctx, "DELETE FROM chunk_hierarchy WHERE descendant_id = $1", chunkID)
	if err != nil {
		return fmt.Errorf("failed to clear existing hierarchy records: %w", err)
	}
	
	// Rebuild hierarchy using recursive CTE
	rebuildQuery := `
		WITH RECURSIVE hierarchy AS (
			-- Self-reference (depth = 0)
			SELECT $1::uuid as ancestor_id, $1::uuid as descendant_id, 0 as depth, ARRAY[$1::uuid] as path_ids
			
			UNION ALL
			
			-- Recursive part: find all ancestors
			SELECT c.chunk_id, $1::uuid, h.depth + 1, h.path_ids || c.chunk_id
			FROM hierarchy h
			JOIN chunks c ON h.ancestor_id = c.parent
			WHERE h.depth < 100 -- Prevent infinite recursion
		)
		INSERT INTO chunk_hierarchy (ancestor_id, descendant_id, depth, path_ids)
		SELECT ancestor_id, descendant_id, depth, path_ids FROM hierarchy
	`
	
	_, err = tx.ExecContext(ctx, rebuildQuery, chunkID)
	if err != nil {
		return fmt.Errorf("failed to rebuild hierarchy: %w", err)
	}
	
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	
	cc.logger.Info("Repaired hierarchy consistency", String("chunk_id", chunkID))
	return nil
}

// RepairAllHierarchyConsistencies repairs all hierarchy consistency issues
func (cc *DatabaseConsistencyChecker) RepairAllHierarchyConsistencies(ctx context.Context) (int, error) {
	errors, err := cc.CheckHierarchyConsistency(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to check hierarchy consistency: %w", err)
	}
	
	repaired := 0
	for _, consistencyError := range errors {
		if consistencyError.Type == "missing_hierarchy_record" {
			if err := cc.RepairHierarchyConsistency(ctx, consistencyError.ChunkID); err != nil {
				cc.logger.Error("Failed to repair hierarchy consistency", err, String("chunk_id", consistencyError.ChunkID))
			} else {
				repaired++
			}
		} else if consistencyError.Type == "orphaned_hierarchy_record" {
			// Remove orphaned records
			_, err := cc.db.ExecContext(ctx, 
				"DELETE FROM chunk_hierarchy WHERE descendant_id = $1", 
				consistencyError.ChunkID)
			if err != nil {
				cc.logger.Error("Failed to remove orphaned hierarchy record", err, String("chunk_id", consistencyError.ChunkID))
			} else {
				repaired++
			}
		}
	}
	
	return repaired, nil
}

// CheckSearchCacheConsistency checks search cache for expired entries and consistency
func (cc *DatabaseConsistencyChecker) CheckSearchCacheConsistency(ctx context.Context) ([]ConsistencyError, error) {
	var errors []ConsistencyError
	
	// Check for expired cache entries
	query := `
		SELECT search_hash, expires_at, created_at, result_count
		FROM chunk_search_cache
		WHERE expires_at < NOW()
	`
	
	rows, err := cc.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to check search cache consistency: %w", err)
	}
	defer rows.Close()
	
	for rows.Next() {
		var searchHash string
		var expiresAt, createdAt time.Time
		var resultCount int
		
		if err := rows.Scan(&searchHash, &expiresAt, &createdAt, &resultCount); err != nil {
			cc.logger.Error("Failed to scan search cache row", err)
			continue
		}
		
		errors = append(errors, ConsistencyError{
			Type:        "expired_search_cache",
			ChunkID:     "", // Not applicable for cache entries
			Table:       "chunk_search_cache",
			Description: "Expired search cache entry",
			Details: map[string]interface{}{
				"search_hash":  searchHash,
				"expires_at":   expiresAt,
				"created_at":   createdAt,
				"result_count": resultCount,
			},
			Severity:  "low",
			Timestamp: time.Now(),
		})
	}
	
	return errors, nil
}

// CleanupExpiredSearchCache removes expired search cache entries
func (cc *DatabaseConsistencyChecker) CleanupExpiredSearchCache(ctx context.Context) (int, error) {
	result, err := cc.db.ExecContext(ctx, "DELETE FROM chunk_search_cache WHERE expires_at < NOW()")
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup expired search cache: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	cc.logger.Info("Cleaned up expired search cache entries", Int64("count", rowsAffected))
	return int(rowsAffected), nil
}

// CheckAllConsistency performs a comprehensive consistency check
func (cc *DatabaseConsistencyChecker) CheckAllConsistency(ctx context.Context) (*ConsistencyReport, error) {
	start := time.Now()
	var allErrors []ConsistencyError
	
	// Check tag consistency
	tagErrors, err := cc.CheckTagConsistency(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to check tag consistency: %w", err)
	}
	allErrors = append(allErrors, tagErrors...)
	
	// Check hierarchy consistency
	hierarchyErrors, err := cc.CheckHierarchyConsistency(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to check hierarchy consistency: %w", err)
	}
	allErrors = append(allErrors, hierarchyErrors...)
	
	// Check search cache consistency
	cacheErrors, err := cc.CheckSearchCacheConsistency(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to check search cache consistency: %w", err)
	}
	allErrors = append(allErrors, cacheErrors...)
	
	// Aggregate statistics
	errorsByType := make(map[string]int)
	errorsBySeverity := make(map[string]int)
	
	for _, err := range allErrors {
		errorsByType[err.Type]++
		errorsBySeverity[err.Severity]++
	}
	
	// Generate recommendations
	var recommendations []string
	if len(tagErrors) > 0 {
		recommendations = append(recommendations, "Run RepairAllTagConsistencies to fix tag relationship issues")
	}
	if len(hierarchyErrors) > 0 {
		recommendations = append(recommendations, "Run RepairAllHierarchyConsistencies to fix hierarchy issues")
	}
	if len(cacheErrors) > 0 {
		recommendations = append(recommendations, "Run CleanupExpiredSearchCache to remove expired cache entries")
	}
	if len(allErrors) == 0 {
		recommendations = append(recommendations, "No consistency issues found - system is healthy")
	}
	
	report := &ConsistencyReport{
		CheckTime:       start,
		TotalErrors:     len(allErrors),
		ErrorsByType:    errorsByType,
		ErrorsBySeverity: errorsBySeverity,
		Errors:          allErrors,
		Recommendations: recommendations,
	}
	
	cc.logger.Info("Completed consistency check", 
		Int("total_errors", len(allErrors)),
		Duration("duration", time.Since(start)))
	
	return report, nil
}

// RepairAllInconsistencies attempts to repair all found inconsistencies
func (cc *DatabaseConsistencyChecker) RepairAllInconsistencies(ctx context.Context) (*RepairReport, error) {
	start := time.Now()
	repairedByType := make(map[string]int)
	var failedRepairs []ConsistencyError
	
	// Repair tag inconsistencies
	tagRepaired, err := cc.RepairAllTagConsistencies(ctx)
	if err != nil {
		cc.logger.Error("Failed to repair tag consistencies", err)
		failedRepairs = append(failedRepairs, ConsistencyError{
			Type:        "repair_failure",
			Description: "Failed to repair tag consistencies",
			Details:     map[string]interface{}{"error": err.Error()},
			Severity:    "high",
			Timestamp:   time.Now(),
		})
	} else {
		repairedByType["tag_issues"] = tagRepaired
	}
	
	// Repair hierarchy inconsistencies
	hierarchyRepaired, err := cc.RepairAllHierarchyConsistencies(ctx)
	if err != nil {
		cc.logger.Error("Failed to repair hierarchy consistencies", err)
		failedRepairs = append(failedRepairs, ConsistencyError{
			Type:        "repair_failure",
			Description: "Failed to repair hierarchy consistencies",
			Details:     map[string]interface{}{"error": err.Error()},
			Severity:    "high",
			Timestamp:   time.Now(),
		})
	} else {
		repairedByType["hierarchy_issues"] = hierarchyRepaired
	}
	
	// Cleanup expired cache
	cacheCleanup, err := cc.CleanupExpiredSearchCache(ctx)
	if err != nil {
		cc.logger.Error("Failed to cleanup expired cache", err)
		failedRepairs = append(failedRepairs, ConsistencyError{
			Type:        "repair_failure",
			Description: "Failed to cleanup expired search cache",
			Details:     map[string]interface{}{"error": err.Error()},
			Severity:    "low",
			Timestamp:   time.Now(),
		})
	} else {
		repairedByType["cache_cleanup"] = cacheCleanup
	}
	
	totalRepaired := tagRepaired + hierarchyRepaired + cacheCleanup
	
	report := &RepairReport{
		RepairTime:     start,
		TotalRepaired:  totalRepaired,
		RepairedByType: repairedByType,
		FailedRepairs:  failedRepairs,
		Duration:       time.Since(start),
	}
	
	cc.logger.Info("Completed repair operations", 
		Int("total_repaired", totalRepaired),
		Int("failed_repairs", len(failedRepairs)),
		Duration("duration", time.Since(start)))
	
	return report, nil
}

// ValidateDataIntegrity performs comprehensive data integrity validation
func (cc *DatabaseConsistencyChecker) ValidateDataIntegrity(ctx context.Context) (*IntegrityReport, error) {
	start := time.Now()
	var issues []IntegrityIssue
	totalRecords := make(map[string]int64)
	
	// Check main chunks table
	var chunkCount int64
	err := cc.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM chunks").Scan(&chunkCount)
	if err != nil {
		return nil, fmt.Errorf("failed to count chunks: %w", err)
	}
	totalRecords["chunks"] = chunkCount
	
	// Check chunk_tags table
	var tagRelationCount int64
	err = cc.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM chunk_tags").Scan(&tagRelationCount)
	if err != nil {
		return nil, fmt.Errorf("failed to count tag relations: %w", err)
	}
	totalRecords["chunk_tags"] = tagRelationCount
	
	// Check chunk_hierarchy table
	var hierarchyCount int64
	err = cc.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM chunk_hierarchy").Scan(&hierarchyCount)
	if err != nil {
		return nil, fmt.Errorf("failed to count hierarchy records: %w", err)
	}
	totalRecords["chunk_hierarchy"] = hierarchyCount
	
	// Check for NULL chunk_ids
	var nullChunkIds int64
	err = cc.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM chunks WHERE chunk_id IS NULL").Scan(&nullChunkIds)
	if err != nil {
		return nil, fmt.Errorf("failed to check null chunk_ids: %w", err)
	}
	
	if nullChunkIds > 0 {
		issues = append(issues, IntegrityIssue{
			Type:        "null_primary_key",
			Table:       "chunks",
			Description: "Chunks with NULL chunk_id",
			Count:       nullChunkIds,
			Severity:    "critical",
		})
	}
	
	// Check for duplicate chunk_ids
	var duplicateChunkIds int64
	err = cc.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM (
			SELECT chunk_id FROM chunks 
			GROUP BY chunk_id 
			HAVING COUNT(*) > 1
		) duplicates
	`).Scan(&duplicateChunkIds)
	if err != nil {
		return nil, fmt.Errorf("failed to check duplicate chunk_ids: %w", err)
	}
	
	if duplicateChunkIds > 0 {
		issues = append(issues, IntegrityIssue{
			Type:        "duplicate_primary_key",
			Table:       "chunks",
			Description: "Duplicate chunk_ids found",
			Count:       duplicateChunkIds,
			Severity:    "critical",
		})
	}
	
	// Check for invalid parent references
	var invalidParents int64
	err = cc.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM chunks c1
		WHERE c1.parent IS NOT NULL 
		AND NOT EXISTS (SELECT 1 FROM chunks c2 WHERE c2.chunk_id = c1.parent)
	`).Scan(&invalidParents)
	if err != nil {
		return nil, fmt.Errorf("failed to check invalid parents: %w", err)
	}
	
	if invalidParents > 0 {
		issues = append(issues, IntegrityIssue{
			Type:        "invalid_foreign_key",
			Table:       "chunks",
			Description: "Chunks with invalid parent references",
			Count:       invalidParents,
			Severity:    "high",
		})
	}
	
	// Generate recommendations
	var recommendations []string
	isHealthy := len(issues) == 0
	
	for _, issue := range issues {
		switch issue.Type {
		case "null_primary_key":
			recommendations = append(recommendations, "Remove or fix records with NULL primary keys")
		case "duplicate_primary_key":
			recommendations = append(recommendations, "Resolve duplicate primary key conflicts")
		case "invalid_foreign_key":
			recommendations = append(recommendations, "Fix or remove invalid foreign key references")
		}
	}
	
	if isHealthy {
		recommendations = append(recommendations, "Data integrity is healthy - no issues found")
	}
	
	report := &IntegrityReport{
		CheckTime:       start,
		TablesChecked:   []string{"chunks", "chunk_tags", "chunk_hierarchy"},
		TotalRecords:    totalRecords,
		IntegrityIssues: issues,
		Recommendations: recommendations,
		IsHealthy:       isHealthy,
	}
	
	cc.logger.Info("Completed data integrity validation", 
		Bool("is_healthy", isHealthy),
		Int("issues_found", len(issues)),
		Duration("duration", time.Since(start)))
	
	return report, nil
}

// VerifyMigration verifies data migration between source and target tables
func (cc *DatabaseConsistencyChecker) VerifyMigration(ctx context.Context, sourceTable, targetTable string) (*MigrationReport, error) {
	// Get source count
	var sourceCount int64
	err := cc.db.QueryRowContext(ctx, fmt.Sprintf("SELECT COUNT(*) FROM %s", sourceTable)).Scan(&sourceCount)
	if err != nil {
		return nil, fmt.Errorf("failed to count source records: %w", err)
	}
	
	// Get target count
	var targetCount int64
	err = cc.db.QueryRowContext(ctx, fmt.Sprintf("SELECT COUNT(*) FROM %s", targetTable)).Scan(&targetCount)
	if err != nil {
		return nil, fmt.Errorf("failed to count target records: %w", err)
	}
	
	// Calculate completion rate
	var completionRate float64
	if sourceCount > 0 {
		completionRate = float64(targetCount) / float64(sourceCount)
	}
	
	isComplete := targetCount >= sourceCount
	
	report := &MigrationReport{
		SourceTable:    sourceTable,
		TargetTable:    targetTable,
		SourceCount:    sourceCount,
		TargetCount:    targetCount,
		IsComplete:     isComplete,
		CompletionRate: completionRate,
		// Note: Detailed record comparison would require specific table schemas
		// This is a basic implementation that can be extended
	}
	
	cc.logger.Info("Completed migration verification", 
		String("source_table", sourceTable),
		String("target_table", targetTable),
		Int64("source_count", sourceCount),
		Int64("target_count", targetCount),
		Float64("completion_rate", completionRate))
	
	return report, nil
}