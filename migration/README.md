# Data Migration and Backward Compatibility System

This directory contains the complete implementation of **Task 7** from the unified-chunk-system project, providing comprehensive data migration scripts and API backward compatibility layers.

## Overview

The migration system implements the SPARC methodology (Specification, Pseudocode, Architecture, Refinement, Code) to ensure systematic and high-quality migration from the old separated schema to the new unified chunk system.

## Features

### Task 7.1: Data Migration Scripts
- **Comprehensive Migration Pipeline**: Orchestrated migration with progress tracking
- **Data Integrity Verification**: Cryptographic checksums and relationship validation
- **Error Handling & Recovery**: Retry mechanisms with intelligent error categorization
- **Rollback Mechanisms**: Granular rollback with incremental checkpoints
- **Progress Monitoring**: Real-time progress tracking with bottleneck detection

### Task 7.2: API Backward Compatibility
- **Legacy API Adapter**: Maintains existing API contracts
- **Response Transformation**: Seamless conversion between old and new formats
- **Deprecation Management**: Usage tracking and migration guidance
- **Version Control**: API versioning with deprecation warnings

## Architecture Components

### Core Migration System
- **MigrationOrchestrator**: Coordinates the entire migration process
- **DataTransformer**: Transforms data between old and new schemas
- **ProgressMonitor**: Tracks migration progress and performance
- **RollbackManager**: Handles rollback operations and checkpoints
- **MigrationValidator**: Validates migration results and data integrity

### Compatibility Layer
- **CompatibilityLayer**: Main backward compatibility interface
- **LegacySupabaseAdapter**: Wraps existing SupabaseClient interface
- **APIVersionManager**: Manages API versions and deprecation
- **DeprecationManager**: Tracks usage of deprecated features

## File Structure

```
migration/
├── SPARC_PSEUDOCODE.md           # SPARC Phase 2: Algorithm design
├── SPARC_ARCHITECTURE.md         # SPARC Phase 3: System architecture
├── SPARC_REFINEMENT.md           # SPARC Phase 4: Safety optimizations
├── migration_orchestrator.go     # Main migration coordination
├── data_transformer.go           # Data transformation logic
├── rollback_manager.go           # Rollback and checkpoint management
├── progress_monitor.go           # Progress tracking and analytics
├── migration_validator.go        # Validation and verification
├── compatibility_layer.go        # Backward compatibility interface
├── legacy_supabase_adapter.go    # Legacy API adapter
├── migration_cli.go              # Command-line interface
├── main.go                       # CLI entry point
└── README.md                     # This file
```

## Usage

### 1. Configuration

Create a configuration file (e.g., `migration_config.yaml`):

```yaml
source_database:
  host: localhost
  port: 5432
  user: postgres
  password: your_password
  database: semantic_text_processor
  ssl_mode: disable

target_database:
  host: localhost
  port: 5432
  user: postgres
  password: your_password
  database: semantic_text_processor_unified
  ssl_mode: disable

migration:
  batch_size: 1000
  max_retries: 3
  timeout_per_batch: 30s
  parallel_workers: 2
  validation_level: standard
  enable_rollback: true
  backup_before_migrate: true
  dry_run: false
```

### 2. Running Migration

```bash
# Generate sample configuration
go run main.go -command=generate-config -config=migration_config.yaml

# Run migration
go run main.go -command=migrate -config=migration_config.yaml

# Dry run migration
go run main.go -command=migrate -config=migration_config.yaml -dry-run

# Validate without migrating
go run main.go -command=validate -config=migration_config.yaml

# Check migration status
go run main.go -command=status

# Check specific migration status
go run main.go -command=status -migration-id=<migration-id>

# Rollback migration
go run main.go -command=rollback -migration-id=<migration-id>

# Generate compatibility report
go run main.go -command=compatibility-report
```

### 3. Using the Compatibility Layer

```go
// Replace existing SupabaseClient with backward-compatible adapter
func setupBackwardCompatibility(unifiedService UnifiedChunkService) clients.SupabaseClient {
    return migration.NewLegacySupabaseAdapter(unifiedService)
}

// Example usage in existing code
func existingFunction() {
    // This code continues to work unchanged
    client := setupBackwardCompatibility(newUnifiedService)

    chunk := &models.ChunkRecord{
        Content: "example content",
        TextID:  "text-123",
    }

    // Legacy API call works transparently
    err := client.InsertChunk(ctx, chunk)
    // Automatically transformed to unified format behind the scenes
}
```

## Migration Process

### Phase 1: Validation
- Verify database connectivity
- Check schema compatibility
- Count total records for progress tracking

### Phase 2: Schema Preparation
- Create migration tracking tables
- Prepare target indexes
- Disable foreign key constraints during migration

### Phase 3: Data Extraction
- Extract data from old schema tables
- Process in configurable batches
- Track progress per table

### Phase 4: Data Transformation
- Transform chunks to unified format
- Migrate tag relationships
- Build hierarchy relationships

### Phase 5: Data Loading
- Load transformed data to target
- Handle conflicts and duplicates
- Validate referential integrity

### Phase 6: Relationship Migration
- Migrate vector embedding references
- Migrate graph node references
- Refresh materialized views

### Phase 7: Validation & Verification
- Validate row counts
- Check data integrity
- Validate relationships
- Re-enable constraints

## Rollback Capabilities

### Rollback Strategies
1. **Backup Restoration**: Full restore from backup
2. **Incremental Revert**: Step-by-step reversal of changes
3. **Schema-Only Rollback**: Revert schema changes only

### Checkpoint System
- Automatic checkpoints at each phase
- Schema snapshots and data summaries
- Granular rollback to any checkpoint

## Monitoring and Observability

### Progress Tracking
- Real-time progress updates
- Throughput analysis and bottleneck detection
- Estimated completion times
- Performance metrics collection

### Error Handling
- Categorized error types with appropriate recovery strategies
- Retry mechanisms with exponential backoff
- Error analytics and pattern recognition

### Alerting
- Configurable thresholds for performance metrics
- Automatic alerts for critical issues
- Comprehensive error reporting

## Security Features

### Access Control
- Role-based permissions for migration operations
- Audit trail for all migration activities
- Encrypted data in transit and at rest

### Data Protection
- Backup creation before migration
- Checksum validation for data integrity
- Rollback capabilities for data safety

## Performance Optimizations

### Adaptive Processing
- Dynamic batch sizing based on system performance
- Resource-aware worker pool scaling
- Intelligent connection pool management

### Efficiency Features
- Parallel processing with configurable workers
- Prepared statements for optimal database performance
- Index optimization during migration

## Testing and Validation

### Validation Levels
- **Basic**: Row counts and basic integrity checks
- **Standard**: Data checksums and relationship validation
- **Rigorous**: Comprehensive validation with performance checks
- **Forensic**: Deep inspection with audit trails

### Test Coverage
- Unit tests for all transformation logic
- Integration tests for end-to-end migration
- Performance regression tests
- Rollback scenario testing

## Best Practices

### Before Migration
1. **Create backups** of all source data
2. **Test migration** in staging environment
3. **Verify compatibility** of existing applications
4. **Plan rollback strategy** and test rollback procedures

### During Migration
1. **Monitor progress** and performance metrics
2. **Watch for errors** and address them promptly
3. **Keep stakeholders informed** of migration status
4. **Have rollback ready** in case of critical issues

### After Migration
1. **Validate results** thoroughly
2. **Monitor application performance** with new schema
3. **Update application code** to use unified APIs gradually
4. **Plan deprecation timeline** for legacy APIs

## Troubleshooting

### Common Issues
1. **Connection timeouts**: Increase timeout values or reduce batch sizes
2. **Memory issues**: Reduce parallel workers or batch sizes
3. **Lock contentions**: Adjust transaction boundaries
4. **Performance degradation**: Check for missing indexes or analyze query plans

### Error Recovery
1. **Retry failed batches** with exponential backoff
2. **Skip corrupted records** and log for manual review
3. **Rollback to last checkpoint** if errors persist
4. **Contact support** with migration logs and error details

## Contributing

### Development Setup
1. Install Go 1.19+
2. Install PostgreSQL 14+
3. Set up test databases
4. Run tests: `go test ./...`

### Code Standards
- Follow SPARC methodology for new features
- Comprehensive error handling and logging
- Unit tests for all new functionality
- Documentation for all public interfaces

## Support

For issues, questions, or contributions:
1. Check existing documentation and troubleshooting guides
2. Review migration logs for error details
3. Test in staging environment before production
4. Contact the development team with specific error messages and context

---

**Generated with SPARC methodology for systematic and reliable software development**