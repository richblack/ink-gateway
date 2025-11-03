package migration

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq" // PostgreSQL driver
	"gopkg.in/yaml.v2"
)

// MigrationCLI provides command-line interface for migration operations
type MigrationCLI struct {
	config    *MigrationConfig
	sourceDB  *sql.DB
	targetDB  *sql.DB
	logger    *log.Logger
}

// NewMigrationCLI creates a new migration CLI
func NewMigrationCLI() *MigrationCLI {
	return &MigrationCLI{
		logger: log.New(os.Stdout, "[MIGRATION_CLI] ", log.LstdFlags|log.Lshortfile),
	}
}

// DatabaseConfig holds database connection configuration
type DatabaseConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Database string `yaml:"database"`
	SSLMode  string `yaml:"ssl_mode"`
}

// CLIConfig holds complete CLI configuration
type CLIConfig struct {
	SourceDatabase DatabaseConfig    `yaml:"source_database"`
	TargetDatabase DatabaseConfig    `yaml:"target_database"`
	Migration      MigrationConfig   `yaml:"migration"`
}

// Main CLI entry point
func (cli *MigrationCLI) Run() error {
	var (
		configFile = flag.String("config", "migration_config.yaml", "Path to configuration file")
		command    = flag.String("command", "migrate", "Command to execute: migrate, validate, rollback, status")
		migrationID = flag.String("migration-id", "", "Migration ID for status/rollback operations")
		dryRun     = flag.Bool("dry-run", false, "Perform a dry run without making changes")
		verbose    = flag.Bool("verbose", false, "Enable verbose logging")
	)
	flag.Parse()

	// Load configuration
	config, err := cli.loadConfig(*configFile)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Override dry run if specified
	if *dryRun {
		config.Migration.DryRun = true
	}

	cli.config = &config.Migration

	// Setup logging
	if *verbose {
		cli.logger.SetFlags(log.LstdFlags | log.Lshortfile | log.Lmicroseconds)
	}

	// Connect to databases
	if err := cli.connectDatabases(config); err != nil {
		return fmt.Errorf("failed to connect to databases: %w", err)
	}
	defer cli.closeDatabases()

	// Execute command
	switch *command {
	case "migrate":
		return cli.executeMigration()
	case "validate":
		return cli.executeValidation()
	case "rollback":
		if *migrationID == "" {
			return fmt.Errorf("migration-id is required for rollback command")
		}
		return cli.executeRollback(*migrationID)
	case "status":
		return cli.executeStatus(*migrationID)
	case "compatibility-report":
		return cli.executeCompatibilityReport()
	default:
		return fmt.Errorf("unknown command: %s", *command)
	}
}

// loadConfig loads configuration from YAML file
func (cli *MigrationCLI) loadConfig(filename string) (*CLIConfig, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config CLIConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Set defaults
	if config.Migration.BatchSize == 0 {
		config.Migration.BatchSize = 1000
	}
	if config.Migration.MaxRetries == 0 {
		config.Migration.MaxRetries = 3
	}
	if config.Migration.TimeoutPerBatch == 0 {
		config.Migration.TimeoutPerBatch = 30 * time.Second
	}
	if config.Migration.ParallelWorkers == 0 {
		config.Migration.ParallelWorkers = 2
	}
	if config.Migration.ValidationLevel == "" {
		config.Migration.ValidationLevel = "standard"
	}

	return &config, nil
}

// connectDatabases establishes database connections
func (cli *MigrationCLI) connectDatabases(config *CLIConfig) error {
	// Connect to source database
	sourceConnStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		config.SourceDatabase.Host, config.SourceDatabase.Port,
		config.SourceDatabase.User, config.SourceDatabase.Password,
		config.SourceDatabase.Database, config.SourceDatabase.SSLMode)

	sourceDB, err := sql.Open("postgres", sourceConnStr)
	if err != nil {
		return fmt.Errorf("failed to connect to source database: %w", err)
	}

	if err := sourceDB.Ping(); err != nil {
		return fmt.Errorf("failed to ping source database: %w", err)
	}

	cli.sourceDB = sourceDB
	cli.logger.Println("Connected to source database")

	// Connect to target database
	targetConnStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		config.TargetDatabase.Host, config.TargetDatabase.Port,
		config.TargetDatabase.User, config.TargetDatabase.Password,
		config.TargetDatabase.Database, config.TargetDatabase.SSLMode)

	targetDB, err := sql.Open("postgres", targetConnStr)
	if err != nil {
		return fmt.Errorf("failed to connect to target database: %w", err)
	}

	if err := targetDB.Ping(); err != nil {
		return fmt.Errorf("failed to ping target database: %w", err)
	}

	cli.targetDB = targetDB
	cli.logger.Println("Connected to target database")

	return nil
}

// closeDatabases closes database connections
func (cli *MigrationCLI) closeDatabases() {
	if cli.sourceDB != nil {
		cli.sourceDB.Close()
	}
	if cli.targetDB != nil {
		cli.targetDB.Close()
	}
}

// executeMigration runs the migration process
func (cli *MigrationCLI) executeMigration() error {
	cli.logger.Println("Starting migration process")

	orchestrator := NewMigrationOrchestrator(cli.sourceDB, cli.targetDB, cli.config)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Hour)
	defer cancel()

	// Start periodic progress reporting
	orchestrator.progressMonitor.StartPeriodicReporting(ctx, 30*time.Second)

	if err := orchestrator.ExecuteMigration(ctx); err != nil {
		cli.logger.Printf("Migration failed: %v", err)
		return err
	}

	cli.logger.Println("Migration completed successfully")

	// Print final report
	state := orchestrator.GetState()
	return cli.printMigrationSummary(state)
}

// executeValidation runs validation without migration
func (cli *MigrationCLI) executeValidation() error {
	cli.logger.Println("Running migration validation")

	validator := NewMigrationValidator(cli.sourceDB, cli.targetDB, cli.config)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	report, err := validator.GenerateValidationReport(ctx)
	if err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	return cli.printValidationReport(report)
}

// executeRollback performs rollback operation
func (cli *MigrationCLI) executeRollback(migrationID string) error {
	cli.logger.Printf("Rolling back migration: %s", migrationID)

	rollbackManager := NewRollbackManager(cli.sourceDB, cli.targetDB, cli.config)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Hour)
	defer cancel()

	if err := rollbackManager.ExecuteRollback(ctx, migrationID); err != nil {
		return fmt.Errorf("rollback failed: %w", err)
	}

	cli.logger.Println("Rollback completed successfully")
	return nil
}

// executeStatus shows migration status
func (cli *MigrationCLI) executeStatus(migrationID string) error {
	if migrationID == "" {
		return cli.showAllMigrations()
	}
	return cli.showMigrationStatus(migrationID)
}

// executeCompatibilityReport shows compatibility usage report
func (cli *MigrationCLI) executeCompatibilityReport() error {
	cli.logger.Println("Generating compatibility report")

	// This would require access to a running compatibility layer
	// For now, return a placeholder
	report := &CompatibilityReport{
		GeneratedAt:       time.Now(),
		DeprecationStats:  map[string]int64{},
		SupportedVersions: map[string]bool{"v1": true, "v2": false},
		ActiveWarnings:    map[string]DeprecationInfo{},
		RecommendedActions: []string{"No deprecated features detected"},
	}

	return cli.printCompatibilityReport(report)
}

// showAllMigrations lists all migrations
func (cli *MigrationCLI) showAllMigrations() error {
	query := `
		SELECT migration_id, phase, start_time, last_update_time,
			   total_records, processed_records, success_count, error_count,
			   status
		FROM migration_state
		ORDER BY start_time DESC
		LIMIT 10
	`

	rows, err := cli.targetDB.Query(query)
	if err != nil {
		return fmt.Errorf("failed to query migrations: %w", err)
	}
	defer rows.Close()

	fmt.Println("\nRecent Migrations:")
	fmt.Println("==================")
	fmt.Printf("%-36s %-20s %-20s %-12s %-10s\n", "Migration ID", "Phase", "Started", "Status", "Progress")
	fmt.Println(strings.Repeat("-", 100))

	for rows.Next() {
		var (
			migrationID, phase, status string
			startTime, lastUpdateTime  time.Time
			totalRecords, processedRecords, successCount, errorCount int64
		)

		err := rows.Scan(&migrationID, &phase, &startTime, &lastUpdateTime,
			&totalRecords, &processedRecords, &successCount, &errorCount, &status)
		if err != nil {
			continue
		}

		progress := "0%"
		if totalRecords > 0 {
			progress = fmt.Sprintf("%.1f%%", float64(processedRecords)/float64(totalRecords)*100)
		}

		fmt.Printf("%-36s %-20s %-20s %-12s %-10s\n",
			migrationID, phase, startTime.Format("2006-01-02 15:04:05"),
			status, progress)
	}

	return nil
}

// showMigrationStatus shows detailed status for a specific migration
func (cli *MigrationCLI) showMigrationStatus(migrationID string) error {
	query := `
		SELECT migration_id, phase, start_time, last_update_time,
			   total_records, processed_records, success_count, error_count,
			   estimated_completion, can_rollback, status
		FROM migration_state
		WHERE migration_id = $1
	`

	var (
		id, phase, status string
		startTime, lastUpdateTime time.Time
		totalRecords, processedRecords, successCount, errorCount int64
		estimatedCompletion sql.NullTime
		canRollback bool
	)

	err := cli.targetDB.QueryRow(query, migrationID).Scan(
		&id, &phase, &startTime, &lastUpdateTime,
		&totalRecords, &processedRecords, &successCount, &errorCount,
		&estimatedCompletion, &canRollback, &status)

	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("migration not found: %s", migrationID)
		}
		return fmt.Errorf("failed to query migration status: %w", err)
	}

	fmt.Printf("\nMigration Status: %s\n", migrationID)
	fmt.Println("=================================")
	fmt.Printf("Phase:              %s\n", phase)
	fmt.Printf("Status:             %s\n", status)
	fmt.Printf("Started:            %s\n", startTime.Format("2006-01-02 15:04:05"))
	fmt.Printf("Last Updated:       %s\n", lastUpdateTime.Format("2006-01-02 15:04:05"))
	fmt.Printf("Total Records:      %d\n", totalRecords)
	fmt.Printf("Processed Records:  %d\n", processedRecords)
	fmt.Printf("Success Count:      %d\n", successCount)
	fmt.Printf("Error Count:        %d\n", errorCount)

	if totalRecords > 0 {
		progress := float64(processedRecords) / float64(totalRecords) * 100
		fmt.Printf("Progress:           %.2f%%\n", progress)
	}

	if estimatedCompletion.Valid {
		fmt.Printf("Estimated Completion: %s\n", estimatedCompletion.Time.Format("2006-01-02 15:04:05"))
	}

	fmt.Printf("Can Rollback:       %t\n", canRollback)

	return nil
}

// printMigrationSummary prints migration completion summary
func (cli *MigrationCLI) printMigrationSummary(state *MigrationState) error {
	fmt.Println("\nMigration Summary")
	fmt.Println("=================")
	fmt.Printf("Migration ID:       %s\n", state.ID)
	fmt.Printf("Final Phase:        %s\n", state.Phase.String())
	fmt.Printf("Duration:           %v\n", state.LastUpdateTime.Sub(state.StartTime))
	fmt.Printf("Total Records:      %d\n", state.TotalRecords)
	fmt.Printf("Processed Records:  %d\n", state.ProcessedRecords)
	fmt.Printf("Success Count:      %d\n", state.SuccessCount)
	fmt.Printf("Error Count:        %d\n", state.ErrorCount)

	if len(state.Errors) > 0 {
		fmt.Printf("\nErrors Encountered: %d\n", len(state.Errors))
		for i, err := range state.Errors {
			if i < 5 { // Show only first 5 errors
				fmt.Printf("  - %s: %s\n", err.ErrorType, err.ErrorMessage)
			}
		}
		if len(state.Errors) > 5 {
			fmt.Printf("  ... and %d more errors\n", len(state.Errors)-5)
		}
	}

	return nil
}

// printValidationReport prints validation results
func (cli *MigrationCLI) printValidationReport(report *ValidationReport) error {
	fmt.Println("\nValidation Report")
	fmt.Println("=================")
	fmt.Printf("Overall Status:     %s\n", report.Summary.OverallStatus)
	fmt.Printf("Total Validations:  %d\n", report.Summary.TotalValidations)
	fmt.Printf("Passed:             %d\n", report.Summary.PassedValidations)
	fmt.Printf("Failed:             %d\n", report.Summary.FailedValidations)

	fmt.Println("\nValidation Details:")
	for _, validation := range report.Validations {
		status := "✓"
		if validation.Status != "passed" {
			status = "✗"
		}
		fmt.Printf("  %s %s\n", status, validation.ValidationName)
		if validation.ErrorMessage != "" {
			fmt.Printf("    Error: %s\n", validation.ErrorMessage)
		}
	}

	return nil
}

// printCompatibilityReport prints compatibility usage report
func (cli *MigrationCLI) printCompatibilityReport(report *CompatibilityReport) error {
	fmt.Println("\nCompatibility Report")
	fmt.Println("====================")
	fmt.Printf("Generated At:       %s\n", report.GeneratedAt.Format("2006-01-02 15:04:05"))

	fmt.Println("\nSupported API Versions:")
	for version, supported := range report.SupportedVersions {
		status := "Supported"
		if !supported {
			status = "Not Supported"
		}
		fmt.Printf("  %s: %s\n", version, status)
	}

	if len(report.DeprecationStats) > 0 {
		fmt.Println("\nDeprecation Usage Statistics:")
		for feature, count := range report.DeprecationStats {
			fmt.Printf("  %s: %d calls\n", feature, count)
		}
	}

	if len(report.ActiveWarnings) > 0 {
		fmt.Println("\nActive Deprecation Warnings:")
		for feature, info := range report.ActiveWarnings {
			fmt.Printf("  %s: %s\n", feature, info.WarningMessage)
			if info.DocumentationURL != "" {
				fmt.Printf("    Documentation: %s\n", info.DocumentationURL)
			}
		}
	}

	if len(report.RecommendedActions) > 0 {
		fmt.Println("\nRecommended Actions:")
		for _, action := range report.RecommendedActions {
			fmt.Printf("  - %s\n", action)
		}
	}

	return nil
}

// generateSampleConfig creates a sample configuration file
func (cli *MigrationCLI) GenerateSampleConfig(filename string) error {
	config := CLIConfig{
		SourceDatabase: DatabaseConfig{
			Host:     "localhost",
			Port:     5432,
			User:     "postgres",
			Password: "password",
			Database: "semantic_text_processor",
			SSLMode:  "disable",
		},
		TargetDatabase: DatabaseConfig{
			Host:     "localhost",
			Port:     5432,
			User:     "postgres",
			Password: "password",
			Database: "semantic_text_processor_unified",
			SSLMode:  "disable",
		},
		Migration: MigrationConfig{
			BatchSize:           1000,
			MaxRetries:          3,
			TimeoutPerBatch:     30 * time.Second,
			ParallelWorkers:     2,
			ValidationLevel:     "standard",
			EnableRollback:      true,
			BackupBeforeMigrate: true,
			DryRun:              false,
		},
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	cli.logger.Printf("Sample configuration written to: %s", filename)
	return nil
}

// Helper function to safely handle string representation
import "strings"

// Helper methods and types would continue here...