package migration

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"semantic-text-processor/models"
)

// MigrationOrchestrator coordinates the entire migration process
type MigrationOrchestrator struct {
	sourceDB        *sql.DB
	targetDB        *sql.DB
	config          *MigrationConfig
	progressMonitor *ProgressMonitor
	rollbackManager *RollbackManager
	logger          *log.Logger
	state           *MigrationState
	mu              sync.RWMutex
}

// MigrationConfig holds configuration for the migration process
type MigrationConfig struct {
	BatchSize           int           `yaml:"batch_size" json:"batch_size"`
	MaxRetries          int           `yaml:"max_retries" json:"max_retries"`
	TimeoutPerBatch     time.Duration `yaml:"timeout_per_batch" json:"timeout_per_batch"`
	ParallelWorkers     int           `yaml:"parallel_workers" json:"parallel_workers"`
	ValidationLevel     string        `yaml:"validation_level" json:"validation_level"`
	EnableRollback      bool          `yaml:"enable_rollback" json:"enable_rollback"`
	BackupBeforeMigrate bool          `yaml:"backup_before_migrate" json:"backup_before_migrate"`
	DryRun              bool          `yaml:"dry_run" json:"dry_run"`
}

// MigrationPhase represents the current phase of migration
type MigrationPhase int

const (
	PhaseValidation MigrationPhase = iota
	PhaseSchemaPreparation
	PhaseDataExtraction
	PhaseDataTransformation
	PhaseDataLoading
	PhaseRelationshipMigration
	PhaseValidationAndVerification
	PhaseComplete
	PhaseRollback
	PhaseFailed
)

func (p MigrationPhase) String() string {
	phases := []string{
		"validation", "schema_preparation", "data_extraction",
		"data_transformation", "data_loading", "relationship_migration",
		"validation_and_verification", "complete", "rollback", "failed",
	}
	if int(p) < len(phases) {
		return phases[p]
	}
	return "unknown"
}

// MigrationState tracks the current state of migration
type MigrationState struct {
	ID                  string                    `json:"id"`
	Phase               MigrationPhase            `json:"phase"`
	StartTime           time.Time                 `json:"start_time"`
	LastUpdateTime      time.Time                 `json:"last_update_time"`
	TablesProcessed     map[string]*TableProgress `json:"tables_processed"`
	TotalRecords        int64                     `json:"total_records"`
	ProcessedRecords    int64                     `json:"processed_records"`
	SuccessCount        int64                     `json:"success_count"`
	ErrorCount          int64                     `json:"error_count"`
	EstimatedCompletion time.Time                 `json:"estimated_completion"`
	CanRollback         bool                      `json:"can_rollback"`
	Errors              []MigrationError          `json:"errors"`
}

// TableProgress tracks progress for individual tables
type TableProgress struct {
	TableName      string    `json:"table_name"`
	TotalRows      int64     `json:"total_rows"`
	ProcessedRows  int64     `json:"processed_rows"`
	SuccessRows    int64     `json:"success_rows"`
	ErrorRows      int64     `json:"error_rows"`
	StartTime      time.Time `json:"start_time"`
	EndTime        *time.Time `json:"end_time,omitempty"`
	CurrentBatch   int       `json:"current_batch"`
	EstimatedETA   time.Time `json:"estimated_eta"`
}

// MigrationError represents an error that occurred during migration
type MigrationError struct {
	ID             string                 `json:"id"`
	ErrorType      string                 `json:"error_type"`
	ErrorMessage   string                 `json:"error_message"`
	ErrorContext   map[string]interface{} `json:"error_context"`
	SourceTable    string                 `json:"source_table"`
	SourceRecordID string                 `json:"source_record_id"`
	RetryCount     int                    `json:"retry_count"`
	IsResolved     bool                   `json:"is_resolved"`
	CreatedAt      time.Time              `json:"created_at"`
}

// NewMigrationOrchestrator creates a new migration orchestrator
func NewMigrationOrchestrator(sourceDB, targetDB *sql.DB, config *MigrationConfig) *MigrationOrchestrator {
	logger := log.New(log.Writer(), "[MIGRATION] ", log.LstdFlags|log.Lshortfile)

	state := &MigrationState{
		ID:               uuid.New().String(),
		Phase:            PhaseValidation,
		StartTime:        time.Now(),
		LastUpdateTime:   time.Now(),
		TablesProcessed:  make(map[string]*TableProgress),
		CanRollback:      config.EnableRollback,
		Errors:           make([]MigrationError, 0),
	}

	return &MigrationOrchestrator{
		sourceDB:        sourceDB,
		targetDB:        targetDB,
		config:          config,
		progressMonitor: NewProgressMonitor(),
		rollbackManager: NewRollbackManager(sourceDB, targetDB, config),
		logger:          logger,
		state:           state,
	}
}

// ExecuteMigration runs the complete migration process
func (m *MigrationOrchestrator) ExecuteMigration(ctx context.Context) error {
	m.logger.Printf("Starting migration %s", m.state.ID)

	// Save initial state
	if err := m.persistState(); err != nil {
		return fmt.Errorf("failed to persist initial state: %w", err)
	}

	// Create backup if enabled
	if m.config.BackupBeforeMigrate {
		if err := m.createBackup(ctx); err != nil {
			return fmt.Errorf("backup creation failed: %w", err)
		}
	}

	// Execute migration phases
	phases := []func(context.Context) error{
		m.executeValidationPhase,
		m.executeSchemaPreparationPhase,
		m.executeDataExtractionPhase,
		m.executeDataTransformationPhase,
		m.executeDataLoadingPhase,
		m.executeRelationshipMigrationPhase,
		m.executeValidationAndVerificationPhase,
	}

	for i, phase := range phases {
		m.updatePhase(MigrationPhase(i))

		if err := phase(ctx); err != nil {
			m.state.Phase = PhaseFailed
			m.recordError("phase_execution", err.Error(), map[string]interface{}{
				"phase": MigrationPhase(i).String(),
			})

			if m.config.EnableRollback {
				m.logger.Printf("Phase failed, initiating rollback: %v", err)
				return m.rollbackManager.ExecuteRollback(ctx, m.state.ID)
			}
			return fmt.Errorf("migration failed at phase %s: %w", MigrationPhase(i).String(), err)
		}

		m.persistState()
	}

	m.state.Phase = PhaseComplete
	m.persistState()
	m.logger.Printf("Migration %s completed successfully", m.state.ID)
	return nil
}

// executeValidationPhase validates source data and target schema
func (m *MigrationOrchestrator) executeValidationPhase(ctx context.Context) error {
	m.logger.Println("Executing validation phase")

	// Validate source database connectivity
	if err := m.sourceDB.PingContext(ctx); err != nil {
		return fmt.Errorf("source database connectivity failed: %w", err)
	}

	// Validate target database connectivity
	if err := m.targetDB.PingContext(ctx); err != nil {
		return fmt.Errorf("target database connectivity failed: %w", err)
	}

	// Validate source schema exists
	tables := []string{"content_db.texts", "content_db.chunks", "content_db.chunk_tags",
		"content_db.template_slots", "vector_db.embeddings", "graph_db.graph_nodes",
		"graph_db.graph_edges"}

	for _, table := range tables {
		exists, err := m.tableExists(m.sourceDB, table)
		if err != nil {
			return fmt.Errorf("failed to check table %s: %w", table, err)
		}
		if !exists {
			return fmt.Errorf("source table %s does not exist", table)
		}
	}

	// Validate target schema exists
	targetTables := []string{"chunks", "chunk_tags", "chunk_hierarchy", "chunk_search_cache"}
	for _, table := range targetTables {
		exists, err := m.tableExists(m.targetDB, table)
		if err != nil {
			return fmt.Errorf("failed to check target table %s: %w", table, err)
		}
		if !exists {
			return fmt.Errorf("target table %s does not exist", table)
		}
	}

	// Count total records for progress tracking
	totalRecords, err := m.countTotalRecords(ctx)
	if err != nil {
		return fmt.Errorf("failed to count total records: %w", err)
	}

	m.state.TotalRecords = totalRecords
	m.logger.Printf("Validation completed. Total records to migrate: %d", totalRecords)

	return nil
}

// executeSchemaPreparationPhase prepares the target schema for migration
func (m *MigrationOrchestrator) executeSchemaPreparationPhase(ctx context.Context) error {
	m.logger.Println("Executing schema preparation phase")

	// Create migration tracking tables
	if err := m.createMigrationTrackingTables(ctx); err != nil {
		return fmt.Errorf("failed to create migration tracking tables: %w", err)
	}

	// Disable foreign key constraints during migration
	if !m.config.DryRun {
		if err := m.disableForeignKeyConstraints(ctx); err != nil {
			return fmt.Errorf("failed to disable foreign key constraints: %w", err)
		}
	}

	// Create indexes for performance (will be rebuilt after migration)
	if err := m.prepareTargetIndexes(ctx); err != nil {
		return fmt.Errorf("failed to prepare target indexes: %w", err)
	}

	m.logger.Println("Schema preparation completed")
	return nil
}

// executeDataExtractionPhase extracts data from source tables
func (m *MigrationOrchestrator) executeDataExtractionPhase(ctx context.Context) error {
	m.logger.Println("Executing data extraction phase")

	// Extract data from each source table
	extractors := map[string]func(context.Context) error{
		"content_db.texts":          m.extractTexts,
		"content_db.chunks":         m.extractChunks,
		"content_db.chunk_tags":     m.extractChunkTags,
		"content_db.template_slots": m.extractTemplateSlots,
		"vector_db.embeddings":      m.extractEmbeddings,
		"graph_db.graph_nodes":      m.extractGraphNodes,
		"graph_db.graph_edges":      m.extractGraphEdges,
	}

	for tableName, extractor := range extractors {
		m.logger.Printf("Extracting data from %s", tableName)

		m.state.TablesProcessed[tableName] = &TableProgress{
			TableName: tableName,
			StartTime: time.Now(),
		}

		if err := extractor(ctx); err != nil {
			return fmt.Errorf("failed to extract from %s: %w", tableName, err)
		}

		progress := m.state.TablesProcessed[tableName]
		endTime := time.Now()
		progress.EndTime = &endTime
	}

	m.logger.Println("Data extraction completed")
	return nil
}

// executeDataTransformationPhase transforms extracted data to unified format
func (m *MigrationOrchestrator) executeDataTransformationPhase(ctx context.Context) error {
	m.logger.Println("Executing data transformation phase")

	transformer := NewDataTransformer(m.config)

	// Transform chunks data to unified format
	if err := transformer.TransformChunksToUnified(ctx, m.sourceDB, m.targetDB); err != nil {
		return fmt.Errorf("failed to transform chunks: %w", err)
	}

	// Transform tag relationships
	if err := transformer.TransformTagRelationships(ctx, m.sourceDB, m.targetDB); err != nil {
		return fmt.Errorf("failed to transform tag relationships: %w", err)
	}

	// Build hierarchy relationships
	if err := transformer.BuildHierarchyRelationships(ctx, m.targetDB); err != nil {
		return fmt.Errorf("failed to build hierarchy relationships: %w", err)
	}

	m.logger.Println("Data transformation completed")
	return nil
}

// executeDataLoadingPhase loads transformed data into target tables
func (m *MigrationOrchestrator) executeDataLoadingPhase(ctx context.Context) error {
	m.logger.Println("Executing data loading phase")

	loader := NewDataLoader(m.config)

	// Load data in batches to manage memory and transaction size
	if err := loader.LoadTransformedData(ctx, m.targetDB, m.progressMonitor); err != nil {
		return fmt.Errorf("failed to load transformed data: %w", err)
	}

	m.logger.Println("Data loading completed")
	return nil
}

// executeRelationshipMigrationPhase migrates relationships and auxiliary data
func (m *MigrationOrchestrator) executeRelationshipMigrationPhase(ctx context.Context) error {
	m.logger.Println("Executing relationship migration phase")

	// Migrate vector embeddings references
	if err := m.migrateVectorReferences(ctx); err != nil {
		return fmt.Errorf("failed to migrate vector references: %w", err)
	}

	// Migrate graph node references
	if err := m.migrateGraphReferences(ctx); err != nil {
		return fmt.Errorf("failed to migrate graph references: %w", err)
	}

	// Update materialized views
	if err := m.refreshMaterializedViews(ctx); err != nil {
		return fmt.Errorf("failed to refresh materialized views: %w", err)
	}

	m.logger.Println("Relationship migration completed")
	return nil
}

// executeValidationAndVerificationPhase validates the migration results
func (m *MigrationOrchestrator) executeValidationAndVerificationPhase(ctx context.Context) error {
	m.logger.Println("Executing validation and verification phase")

	validator := NewMigrationValidator(m.sourceDB, m.targetDB, m.config)

	// Validate row counts
	if err := validator.ValidateRowCounts(ctx); err != nil {
		return fmt.Errorf("row count validation failed: %w", err)
	}

	// Validate data integrity
	if err := validator.ValidateDataIntegrity(ctx); err != nil {
		return fmt.Errorf("data integrity validation failed: %w", err)
	}

	// Validate relationships
	if err := validator.ValidateRelationships(ctx); err != nil {
		return fmt.Errorf("relationship validation failed: %w", err)
	}

	// Re-enable foreign key constraints
	if !m.config.DryRun {
		if err := m.enableForeignKeyConstraints(ctx); err != nil {
			return fmt.Errorf("failed to re-enable foreign key constraints: %w", err)
		}
	}

	// Rebuild indexes for optimal performance
	if err := m.rebuildTargetIndexes(ctx); err != nil {
		return fmt.Errorf("failed to rebuild target indexes: %w", err)
	}

	m.logger.Println("Validation and verification completed")
	return nil
}

// Helper methods

func (m *MigrationOrchestrator) updatePhase(phase MigrationPhase) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.state.Phase = phase
	m.state.LastUpdateTime = time.Now()
	m.logger.Printf("Migration phase updated to: %s", phase.String())
}

func (m *MigrationOrchestrator) recordError(errorType, message string, context map[string]interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()

	error := MigrationError{
		ID:           uuid.New().String(),
		ErrorType:    errorType,
		ErrorMessage: message,
		ErrorContext: context,
		CreatedAt:    time.Now(),
	}

	m.state.Errors = append(m.state.Errors, error)
	m.state.ErrorCount++
}

func (m *MigrationOrchestrator) tableExists(db *sql.DB, tableName string) (bool, error) {
	var count int
	query := `
		SELECT COUNT(*)
		FROM information_schema.tables
		WHERE table_schema || '.' || table_name = $1`

	err := db.QueryRow(query, tableName).Scan(&count)
	return count > 0, err
}

func (m *MigrationOrchestrator) countTotalRecords(ctx context.Context) (int64, error) {
	var total int64

	tables := []string{
		"content_db.texts", "content_db.chunks", "content_db.chunk_tags",
		"content_db.template_slots", "vector_db.embeddings",
		"graph_db.graph_nodes", "graph_db.graph_edges",
	}

	for _, table := range tables {
		var count int64
		query := fmt.Sprintf("SELECT COUNT(*) FROM %s", table)
		if err := m.sourceDB.QueryRowContext(ctx, query).Scan(&count); err != nil {
			return 0, fmt.Errorf("failed to count records in %s: %w", table, err)
		}
		total += count
	}

	return total, nil
}

func (m *MigrationOrchestrator) createMigrationTrackingTables(ctx context.Context) error {
	schema := `
		-- Migration state tracking
		CREATE TABLE IF NOT EXISTS migration_state (
			migration_id UUID PRIMARY KEY,
			phase VARCHAR(50) NOT NULL,
			start_time TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			last_update_time TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			total_records BIGINT DEFAULT 0,
			processed_records BIGINT DEFAULT 0,
			success_count BIGINT DEFAULT 0,
			error_count BIGINT DEFAULT 0,
			estimated_completion TIMESTAMP WITH TIME ZONE,
			can_rollback BOOLEAN DEFAULT true,
			config JSONB,
			status VARCHAR(20) DEFAULT 'in_progress'
		);

		-- Table-level progress tracking
		CREATE TABLE IF NOT EXISTS migration_table_progress (
			migration_id UUID,
			table_name VARCHAR(100) NOT NULL,
			total_rows BIGINT DEFAULT 0,
			processed_rows BIGINT DEFAULT 0,
			success_rows BIGINT DEFAULT 0,
			error_rows BIGINT DEFAULT 0,
			start_time TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			end_time TIMESTAMP WITH TIME ZONE,
			PRIMARY KEY (migration_id, table_name)
		);

		-- Error tracking
		CREATE TABLE IF NOT EXISTS migration_errors (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			migration_id UUID,
			error_type VARCHAR(50) NOT NULL,
			error_message TEXT NOT NULL,
			error_context JSONB,
			source_table VARCHAR(100),
			source_record_id VARCHAR(255),
			retry_count INTEGER DEFAULT 0,
			is_resolved BOOLEAN DEFAULT false,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);
	`

	_, err := m.targetDB.ExecContext(ctx, schema)
	return err
}

func (m *MigrationOrchestrator) persistState() error {
	configJSON, _ := json.Marshal(m.config)
	stateJSON, _ := json.Marshal(m.state)

	query := `
		INSERT INTO migration_state (
			migration_id, phase, start_time, last_update_time,
			total_records, processed_records, success_count, error_count,
			estimated_completion, can_rollback, config, status
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (migration_id) DO UPDATE SET
			phase = EXCLUDED.phase,
			last_update_time = EXCLUDED.last_update_time,
			processed_records = EXCLUDED.processed_records,
			success_count = EXCLUDED.success_count,
			error_count = EXCLUDED.error_count,
			estimated_completion = EXCLUDED.estimated_completion,
			status = EXCLUDED.status
	`

	status := "in_progress"
	if m.state.Phase == PhaseComplete {
		status = "completed"
	} else if m.state.Phase == PhaseFailed {
		status = "failed"
	}

	_, err := m.targetDB.Exec(query,
		m.state.ID, m.state.Phase.String(), m.state.StartTime, m.state.LastUpdateTime,
		m.state.TotalRecords, m.state.ProcessedRecords, m.state.SuccessCount, m.state.ErrorCount,
		m.state.EstimatedCompletion, m.state.CanRollback, configJSON, status)

	return err
}

func (m *MigrationOrchestrator) createBackup(ctx context.Context) error {
	// Implementation would depend on the specific backup strategy
	// This is a placeholder for the backup functionality
	m.logger.Println("Creating backup (placeholder implementation)")
	return nil
}

// Placeholder implementations for extraction methods
func (m *MigrationOrchestrator) extractTexts(ctx context.Context) error {
	m.logger.Println("Extracting texts (placeholder)")
	return nil
}

func (m *MigrationOrchestrator) extractChunks(ctx context.Context) error {
	m.logger.Println("Extracting chunks (placeholder)")
	return nil
}

func (m *MigrationOrchestrator) extractChunkTags(ctx context.Context) error {
	m.logger.Println("Extracting chunk tags (placeholder)")
	return nil
}

func (m *MigrationOrchestrator) extractTemplateSlots(ctx context.Context) error {
	m.logger.Println("Extracting template slots (placeholder)")
	return nil
}

func (m *MigrationOrchestrator) extractEmbeddings(ctx context.Context) error {
	m.logger.Println("Extracting embeddings (placeholder)")
	return nil
}

func (m *MigrationOrchestrator) extractGraphNodes(ctx context.Context) error {
	m.logger.Println("Extracting graph nodes (placeholder)")
	return nil
}

func (m *MigrationOrchestrator) extractGraphEdges(ctx context.Context) error {
	m.logger.Println("Extracting graph edges (placeholder)")
	return nil
}

func (m *MigrationOrchestrator) disableForeignKeyConstraints(ctx context.Context) error {
	_, err := m.targetDB.ExecContext(ctx, "SET session_replication_role = replica;")
	return err
}

func (m *MigrationOrchestrator) enableForeignKeyConstraints(ctx context.Context) error {
	_, err := m.targetDB.ExecContext(ctx, "SET session_replication_role = DEFAULT;")
	return err
}

func (m *MigrationOrchestrator) prepareTargetIndexes(ctx context.Context) error {
	m.logger.Println("Preparing target indexes")
	return nil
}

func (m *MigrationOrchestrator) rebuildTargetIndexes(ctx context.Context) error {
	m.logger.Println("Rebuilding target indexes")
	return nil
}

func (m *MigrationOrchestrator) migrateVectorReferences(ctx context.Context) error {
	m.logger.Println("Migrating vector references")
	return nil
}

func (m *MigrationOrchestrator) migrateGraphReferences(ctx context.Context) error {
	m.logger.Println("Migrating graph references")
	return nil
}

func (m *MigrationOrchestrator) refreshMaterializedViews(ctx context.Context) error {
	_, err := m.targetDB.ExecContext(ctx, "REFRESH MATERIALIZED VIEW CONCURRENTLY tag_statistics;")
	return err
}

// GetState returns the current migration state
func (m *MigrationOrchestrator) GetState() *MigrationState {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return a copy to prevent external modification
	stateCopy := *m.state
	return &stateCopy
}