package migration

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
)

// RollbackManager handles rollback operations for migrations
type RollbackManager struct {
	sourceDB *sql.DB
	targetDB *sql.DB
	config   *MigrationConfig
	logger   *log.Logger
}

// RollbackStrategy defines different rollback approaches
type RollbackStrategy int

const (
	RollbackToBackup RollbackStrategy = iota
	RollbackWithRevert
	RollbackSchemaOnly
)

// RollbackPlan defines how rollback should be executed
type RollbackPlan struct {
	Strategy        RollbackStrategy      `json:"strategy"`
	BackupLocation  string               `json:"backup_location"`
	RevertScripts   []string             `json:"revert_scripts"`
	ValidationSteps []ValidationStep     `json:"validation_steps"`
	Checkpoints     []RollbackCheckpoint `json:"checkpoints"`
}

// ValidationStep represents a step in rollback validation
type ValidationStep struct {
	Name        string                 `json:"name"`
	Query       string                 `json:"query"`
	ExpectedResult interface{}         `json:"expected_result"`
	Critical    bool                   `json:"critical"`
}

// RollbackCheckpoint represents a point-in-time state during migration
type RollbackCheckpoint struct {
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	MigrationPhase  string                 `json:"migration_phase"`
	Timestamp       time.Time              `json:"timestamp"`
	BackupLocation  string                 `json:"backup_location"`
	SchemaSnapshot  map[string]interface{} `json:"schema_snapshot"`
	DataSummary     map[string]interface{} `json:"data_summary"`
}

// NewRollbackManager creates a new rollback manager
func NewRollbackManager(sourceDB, targetDB *sql.DB, config *MigrationConfig) *RollbackManager {
	return &RollbackManager{
		sourceDB: sourceDB,
		targetDB: targetDB,
		config:   config,
		logger:   log.New(log.Writer(), "[ROLLBACK_MANAGER] ", log.LstdFlags|log.Lshortfile),
	}
}

// ExecuteRollback executes a rollback for the specified migration
func (rm *RollbackManager) ExecuteRollback(ctx context.Context, migrationID string) error {
	rm.logger.Printf("Starting rollback for migration %s", migrationID)

	// Get migration state
	state, err := rm.getMigrationState(ctx, migrationID)
	if err != nil {
		return fmt.Errorf("failed to get migration state: %w", err)
	}

	if !state.CanRollback {
		return fmt.Errorf("migration %s cannot be rolled back", migrationID)
	}

	// Create rollback plan
	plan, err := rm.createRollbackPlan(ctx, state)
	if err != nil {
		return fmt.Errorf("failed to create rollback plan: %w", err)
	}

	// Execute rollback based on strategy
	switch plan.Strategy {
	case RollbackToBackup:
		err = rm.rollbackToBackup(ctx, plan)
	case RollbackWithRevert:
		err = rm.rollbackWithRevert(ctx, plan)
	case RollbackSchemaOnly:
		err = rm.rollbackSchemaOnly(ctx, plan)
	default:
		return fmt.Errorf("unknown rollback strategy: %v", plan.Strategy)
	}

	if err != nil {
		return fmt.Errorf("rollback execution failed: %w", err)
	}

	// Update migration state
	if err := rm.updateMigrationStateAfterRollback(ctx, migrationID); err != nil {
		rm.logger.Printf("Failed to update migration state after rollback: %v", err)
	}

	rm.logger.Printf("Rollback completed successfully for migration %s", migrationID)
	return nil
}

// CreateCheckpoint creates a rollback checkpoint
func (rm *RollbackManager) CreateCheckpoint(ctx context.Context, migrationID, name, phase string) (*RollbackCheckpoint, error) {
	rm.logger.Printf("Creating checkpoint '%s' for migration %s", name, migrationID)

	checkpoint := &RollbackCheckpoint{
		ID:             uuid.New().String(),
		Name:           name,
		MigrationPhase: phase,
		Timestamp:      time.Now(),
	}

	// Capture schema snapshot
	schemaSnapshot, err := rm.captureSchemaSnapshot(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to capture schema snapshot: %w", err)
	}
	checkpoint.SchemaSnapshot = schemaSnapshot

	// Capture data summary
	dataSummary, err := rm.captureDataSummary(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to capture data summary: %w", err)
	}
	checkpoint.DataSummary = dataSummary

	// Persist checkpoint
	if err := rm.persistCheckpoint(ctx, migrationID, checkpoint); err != nil {
		return nil, fmt.Errorf("failed to persist checkpoint: %w", err)
	}

	rm.logger.Printf("Checkpoint '%s' created successfully", name)
	return checkpoint, nil
}

// ValidateRollback validates that rollback was successful
func (rm *RollbackManager) ValidateRollback(ctx context.Context, plan *RollbackPlan) error {
	rm.logger.Println("Validating rollback results")

	for _, step := range plan.ValidationSteps {
		rm.logger.Printf("Executing validation step: %s", step.Name)

		var result interface{}
		err := rm.targetDB.QueryRowContext(ctx, step.Query).Scan(&result)
		if err != nil {
			if step.Critical {
				return fmt.Errorf("critical validation step '%s' failed: %w", step.Name, err)
			}
			rm.logger.Printf("Warning: validation step '%s' failed: %v", step.Name, err)
			continue
		}

		// Compare result with expected
		if !rm.compareResults(result, step.ExpectedResult) {
			if step.Critical {
				return fmt.Errorf("critical validation step '%s' result mismatch: got %v, expected %v",
					step.Name, result, step.ExpectedResult)
			}
			rm.logger.Printf("Warning: validation step '%s' result mismatch: got %v, expected %v",
				step.Name, result, step.ExpectedResult)
		}
	}

	rm.logger.Println("Rollback validation completed")
	return nil
}

// Private methods

func (rm *RollbackManager) getMigrationState(ctx context.Context, migrationID string) (*MigrationState, error) {
	query := `
		SELECT migration_id, phase, start_time, last_update_time,
			   total_records, processed_records, success_count, error_count,
			   estimated_completion, can_rollback, config, status
		FROM migration_state
		WHERE migration_id = $1
	`

	var state MigrationState
	var phaseStr, status string
	var configJSON sql.NullString
	var estimatedCompletion sql.NullTime

	err := rm.targetDB.QueryRowContext(ctx, query, migrationID).Scan(
		&state.ID, &phaseStr, &state.StartTime, &state.LastUpdateTime,
		&state.TotalRecords, &state.ProcessedRecords, &state.SuccessCount, &state.ErrorCount,
		&estimatedCompletion, &state.CanRollback, &configJSON, &status,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to query migration state: %w", err)
	}

	// Parse phase
	phases := map[string]MigrationPhase{
		"validation": PhaseValidation, "schema_preparation": PhaseSchemaPreparation,
		"data_extraction": PhaseDataExtraction, "data_transformation": PhaseDataTransformation,
		"data_loading": PhaseDataLoading, "relationship_migration": PhaseRelationshipMigration,
		"validation_and_verification": PhaseValidationAndVerification,
		"complete": PhaseComplete, "rollback": PhaseRollback, "failed": PhaseFailed,
	}
	state.Phase = phases[phaseStr]

	if estimatedCompletion.Valid {
		state.EstimatedCompletion = estimatedCompletion.Time
	}

	return &state, nil
}

func (rm *RollbackManager) createRollbackPlan(ctx context.Context, state *MigrationState) (*RollbackPlan, error) {
	plan := &RollbackPlan{
		Strategy:        RollbackWithRevert, // Default strategy
		RevertScripts:   []string{},
		ValidationSteps: []ValidationStep{},
		Checkpoints:     []RollbackCheckpoint{},
	}

	// Get checkpoints for this migration
	checkpoints, err := rm.getCheckpoints(ctx, state.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get checkpoints: %w", err)
	}
	plan.Checkpoints = checkpoints

	// Define revert scripts based on migration phase
	plan.RevertScripts = rm.generateRevertScripts(state.Phase)

	// Define validation steps
	plan.ValidationSteps = []ValidationStep{
		{
			Name:           "verify_source_schema_intact",
			Query:          "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'content_db'",
			ExpectedResult: int64(4), // Expected number of content_db tables
			Critical:       true,
		},
		{
			Name:           "verify_target_schema_clean",
			Query:          "SELECT COUNT(*) FROM chunks",
			ExpectedResult: int64(0), // Should be empty after rollback
			Critical:       false,
		},
		{
			Name:           "verify_migration_tracking_updated",
			Query:          "SELECT status FROM migration_state WHERE migration_id = $1",
			ExpectedResult: "rolled_back",
			Critical:       true,
		},
	}

	return plan, nil
}

func (rm *RollbackManager) rollbackToBackup(ctx context.Context, plan *RollbackPlan) error {
	rm.logger.Println("Executing rollback to backup")

	if plan.BackupLocation == "" {
		return fmt.Errorf("backup location not specified")
	}

	// This would implement backup restoration
	// Implementation depends on backup strategy (pg_dump, filesystem backup, etc.)
	rm.logger.Printf("Restoring from backup: %s", plan.BackupLocation)

	// Placeholder implementation
	return fmt.Errorf("backup rollback not implemented yet")
}

func (rm *RollbackManager) rollbackWithRevert(ctx context.Context, plan *RollbackPlan) error {
	rm.logger.Println("Executing rollback with revert scripts")

	tx, err := rm.targetDB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin rollback transaction: %w", err)
	}
	defer tx.Rollback()

	// Execute revert scripts
	for _, script := range plan.RevertScripts {
		rm.logger.Printf("Executing revert script: %s", script)
		_, err := tx.ExecContext(ctx, script)
		if err != nil {
			return fmt.Errorf("failed to execute revert script: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit rollback transaction: %w", err)
	}

	// Validate rollback
	return rm.ValidateRollback(ctx, plan)
}

func (rm *RollbackManager) rollbackSchemaOnly(ctx context.Context, plan *RollbackPlan) error {
	rm.logger.Println("Executing schema-only rollback")

	// Only revert schema changes, keep data
	schemaRevertScripts := []string{
		"DROP TABLE IF EXISTS chunk_search_cache CASCADE",
		"DROP TABLE IF EXISTS chunk_hierarchy CASCADE",
		"DROP TABLE IF EXISTS chunk_tags CASCADE",
		"DROP TABLE IF EXISTS chunks CASCADE",
		"DROP MATERIALIZED VIEW IF EXISTS tag_statistics CASCADE",
	}

	tx, err := rm.targetDB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin schema rollback transaction: %w", err)
	}
	defer tx.Rollback()

	for _, script := range schemaRevertScripts {
		rm.logger.Printf("Executing schema revert: %s", script)
		_, err := tx.ExecContext(ctx, script)
		if err != nil {
			rm.logger.Printf("Warning: schema revert failed (may be expected): %v", err)
		}
	}

	return tx.Commit()
}

func (rm *RollbackManager) generateRevertScripts(phase MigrationPhase) []string {
	scripts := []string{}

	switch phase {
	case PhaseComplete, PhaseValidationAndVerification:
		// Full revert required
		scripts = append(scripts,
			"DELETE FROM chunk_search_cache",
			"DELETE FROM chunk_hierarchy",
			"DELETE FROM chunk_tags",
			"DELETE FROM chunks",
			"REFRESH MATERIALIZED VIEW tag_statistics",
		)
	case PhaseRelationshipMigration:
		// Revert relationship data
		scripts = append(scripts,
			"DELETE FROM chunk_hierarchy",
			"UPDATE chunks SET tags = '[]'::jsonb",
		)
	case PhaseDataLoading, PhaseDataTransformation:
		// Revert data changes
		scripts = append(scripts,
			"DELETE FROM chunk_tags",
			"DELETE FROM chunks",
		)
	case PhaseDataExtraction, PhaseSchemaPreparation:
		// Minimal revert needed
		scripts = append(scripts,
			"TRUNCATE TABLE chunks RESTART IDENTITY CASCADE",
		)
	}

	return scripts
}

func (rm *RollbackManager) captureSchemaSnapshot(ctx context.Context) (map[string]interface{}, error) {
	snapshot := make(map[string]interface{})

	// Capture table information
	query := `
		SELECT table_name, column_name, data_type, is_nullable
		FROM information_schema.columns
		WHERE table_schema = 'public'
		ORDER BY table_name, ordinal_position
	`

	rows, err := rm.targetDB.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tables := make(map[string][]map[string]interface{})
	for rows.Next() {
		var tableName, columnName, dataType, isNullable string
		if err := rows.Scan(&tableName, &columnName, &dataType, &isNullable); err != nil {
			return nil, err
		}

		if tables[tableName] == nil {
			tables[tableName] = make([]map[string]interface{}, 0)
		}

		column := map[string]interface{}{
			"name":        columnName,
			"type":        dataType,
			"is_nullable": isNullable,
		}
		tables[tableName] = append(tables[tableName], column)
	}

	snapshot["tables"] = tables
	snapshot["timestamp"] = time.Now()

	return snapshot, nil
}

func (rm *RollbackManager) captureDataSummary(ctx context.Context) (map[string]interface{}, error) {
	summary := make(map[string]interface{})

	// Get row counts for key tables
	tables := []string{"chunks", "chunk_tags", "chunk_hierarchy", "chunk_search_cache"}

	for _, table := range tables {
		var count int64
		query := fmt.Sprintf("SELECT COUNT(*) FROM %s", table)
		err := rm.targetDB.QueryRowContext(ctx, query).Scan(&count)
		if err != nil {
			// Table might not exist, which is fine
			count = 0
		}
		summary[table+"_count"] = count
	}

	summary["timestamp"] = time.Now()
	return summary, nil
}

func (rm *RollbackManager) persistCheckpoint(ctx context.Context, migrationID string, checkpoint *RollbackCheckpoint) error {
	schemaJSON, err := json.Marshal(checkpoint.SchemaSnapshot)
	if err != nil {
		return err
	}

	dataSummaryJSON, err := json.Marshal(checkpoint.DataSummary)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO migration_checkpoints (
			id, migration_id, checkpoint_name, phase, backup_location,
			schema_snapshot, data_snapshot_info, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err = rm.targetDB.ExecContext(ctx, query,
		checkpoint.ID, migrationID, checkpoint.Name, checkpoint.MigrationPhase,
		checkpoint.BackupLocation, schemaJSON, dataSummaryJSON, checkpoint.Timestamp)

	return err
}

func (rm *RollbackManager) getCheckpoints(ctx context.Context, migrationID string) ([]RollbackCheckpoint, error) {
	query := `
		SELECT id, checkpoint_name, phase, backup_location,
			   schema_snapshot, data_snapshot_info, created_at
		FROM migration_checkpoints
		WHERE migration_id = $1
		ORDER BY created_at
	`

	rows, err := rm.targetDB.QueryContext(ctx, query, migrationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var checkpoints []RollbackCheckpoint
	for rows.Next() {
		var checkpoint RollbackCheckpoint
		var schemaJSON, dataSummaryJSON sql.NullString

		err := rows.Scan(
			&checkpoint.ID, &checkpoint.Name, &checkpoint.MigrationPhase,
			&checkpoint.BackupLocation, &schemaJSON, &dataSummaryJSON,
			&checkpoint.Timestamp,
		)
		if err != nil {
			return nil, err
		}

		// Parse JSON fields
		if schemaJSON.Valid {
			if err := json.Unmarshal([]byte(schemaJSON.String), &checkpoint.SchemaSnapshot); err != nil {
				rm.logger.Printf("Failed to unmarshal schema snapshot: %v", err)
			}
		}

		if dataSummaryJSON.Valid {
			if err := json.Unmarshal([]byte(dataSummaryJSON.String), &checkpoint.DataSummary); err != nil {
				rm.logger.Printf("Failed to unmarshal data summary: %v", err)
			}
		}

		checkpoints = append(checkpoints, checkpoint)
	}

	return checkpoints, nil
}

func (rm *RollbackManager) updateMigrationStateAfterRollback(ctx context.Context, migrationID string) error {
	query := `
		UPDATE migration_state
		SET status = 'rolled_back',
			last_update_time = NOW()
		WHERE migration_id = $1
	`

	_, err := rm.targetDB.ExecContext(ctx, query, migrationID)
	return err
}

func (rm *RollbackManager) compareResults(actual, expected interface{}) bool {
	// Simple comparison - could be enhanced for complex types
	return actual == expected
}