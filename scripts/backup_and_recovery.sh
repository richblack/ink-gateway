#!/bin/bash

# Semantic Text Processor - Backup and Disaster Recovery Script
# This script provides comprehensive backup and recovery capabilities for the production system

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
CONFIG_FILE="${PROJECT_ROOT}/config/backup.conf"

# Default configuration
DEFAULT_BACKUP_DIR="/backups"
DEFAULT_RETENTION_DAYS=30
DEFAULT_S3_BUCKET=""
DEFAULT_ENCRYPTION_KEY=""
DEFAULT_COMPRESSION_LEVEL=6
DEFAULT_PARALLEL_JOBS=4

# Load configuration
if [[ -f "$CONFIG_FILE" ]]; then
    source "$CONFIG_FILE"
else
    echo "Warning: Configuration file not found at $CONFIG_FILE, using defaults"
fi

# Set configuration with defaults
BACKUP_DIR="${BACKUP_DIR:-$DEFAULT_BACKUP_DIR}"
RETENTION_DAYS="${RETENTION_DAYS:-$DEFAULT_RETENTION_DAYS}"
S3_BUCKET="${S3_BUCKET:-$DEFAULT_S3_BUCKET}"
ENCRYPTION_KEY="${ENCRYPTION_KEY:-$DEFAULT_ENCRYPTION_KEY}"
COMPRESSION_LEVEL="${COMPRESSION_LEVEL:-$DEFAULT_COMPRESSION_LEVEL}"
PARALLEL_JOBS="${PARALLEL_JOBS:-$DEFAULT_PARALLEL_JOBS}"

# Environment variables
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BACKUP_PREFIX="semantic_processor_backup"
LOG_FILE="${BACKUP_DIR}/logs/backup_${TIMESTAMP}.log"

# Database configuration
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_NAME="${DB_NAME:-semantic_processor}"
DB_USER="${DB_USER:-postgres}"
DB_PASSWORD="${DB_PASSWORD:-}"

# Ensure log directory exists
mkdir -p "$(dirname "$LOG_FILE")"

# Logging functions
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*" | tee -a "$LOG_FILE"
}

error() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] ERROR: $*" | tee -a "$LOG_FILE" >&2
}

success() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] SUCCESS: $*" | tee -a "$LOG_FILE"
}

# Utility functions
check_dependencies() {
    local deps=("pg_dump" "pg_restore" "docker" "gzip" "tar")

    if [[ -n "$S3_BUCKET" ]]; then
        deps+=("aws")
    fi

    if [[ -n "$ENCRYPTION_KEY" ]]; then
        deps+=("gpg")
    fi

    for dep in "${deps[@]}"; do
        if ! command -v "$dep" &> /dev/null; then
            error "Required dependency '$dep' not found"
            exit 1
        fi
    done
}

get_container_id() {
    local service_name="$1"
    docker ps --filter "name=$service_name" --format "{{.ID}}" | head -1
}

wait_for_database() {
    local max_attempts=30
    local attempt=1

    log "Waiting for database to be ready..."

    while [[ $attempt -le $max_attempts ]]; do
        if pg_isready -h "$DB_HOST" -p "$DB_PORT" -d "$DB_NAME" -U "$DB_USER" &>/dev/null; then
            log "Database is ready"
            return 0
        fi

        log "Database not ready, attempt $attempt/$max_attempts"
        sleep 2
        ((attempt++))
    done

    error "Database did not become ready within timeout"
    return 1
}

# Backup functions
backup_database() {
    local backup_file="$1"

    log "Starting database backup to $backup_file"

    # Set password for pg_dump
    export PGPASSWORD="$DB_PASSWORD"

    # Create database backup with custom format for better compression and parallel restore
    if pg_dump \
        --host="$DB_HOST" \
        --port="$DB_PORT" \
        --username="$DB_USER" \
        --dbname="$DB_NAME" \
        --format=custom \
        --compress="$COMPRESSION_LEVEL" \
        --jobs="$PARALLEL_JOBS" \
        --verbose \
        --file="$backup_file" \
        2>>"$LOG_FILE"; then

        success "Database backup completed: $backup_file"
        return 0
    else
        error "Database backup failed"
        return 1
    fi
}

backup_application_state() {
    local backup_dir="$1"

    log "Backing up application state to $backup_dir"

    # Create application state backup directory
    mkdir -p "$backup_dir/app_state"

    # Backup Docker volumes
    log "Backing up Docker volumes..."
    docker run --rm \
        -v semantic-processor_app-logs:/source:ro \
        -v "$backup_dir/app_state:/backup" \
        alpine tar czf /backup/app_logs.tar.gz -C /source . 2>>"$LOG_FILE"

    # Backup configuration files
    log "Backing up configuration files..."
    if [[ -d "$PROJECT_ROOT/config" ]]; then
        tar czf "$backup_dir/app_state/config.tar.gz" -C "$PROJECT_ROOT" config 2>>"$LOG_FILE"
    fi

    # Backup SSL certificates
    if [[ -d "$PROJECT_ROOT/deployments/nginx/ssl" ]]; then
        tar czf "$backup_dir/app_state/ssl_certs.tar.gz" -C "$PROJECT_ROOT/deployments/nginx" ssl 2>>"$LOG_FILE"
    fi

    # Backup environment files
    if [[ -f "$PROJECT_ROOT/.env" ]]; then
        cp "$PROJECT_ROOT/.env" "$backup_dir/app_state/.env.backup"
    fi

    success "Application state backup completed"
}

backup_monitoring_data() {
    local backup_dir="$1"

    log "Backing up monitoring data to $backup_dir"

    mkdir -p "$backup_dir/monitoring"

    # Backup Prometheus data
    if docker ps --filter "name=prometheus" --format "{{.Names}}" | grep -q prometheus; then
        log "Backing up Prometheus data..."
        docker run --rm \
            -v semantic-processor_prometheus-data:/source:ro \
            -v "$backup_dir/monitoring:/backup" \
            alpine tar czf /backup/prometheus_data.tar.gz -C /source . 2>>"$LOG_FILE"
    fi

    # Backup Grafana data
    if docker ps --filter "name=grafana" --format "{{.Names}}" | grep -q grafana; then
        log "Backing up Grafana data..."
        docker run --rm \
            -v semantic-processor_grafana-data:/source:ro \
            -v "$backup_dir/monitoring:/backup" \
            alpine tar czf /backup/grafana_data.tar.gz -C /source . 2>>"$LOG_FILE"
    fi

    success "Monitoring data backup completed"
}

encrypt_backup() {
    local backup_file="$1"
    local encrypted_file="${backup_file}.gpg"

    if [[ -z "$ENCRYPTION_KEY" ]]; then
        log "No encryption key provided, skipping encryption"
        return 0
    fi

    log "Encrypting backup file: $backup_file"

    if gpg --symmetric \
        --cipher-algo AES256 \
        --passphrase "$ENCRYPTION_KEY" \
        --batch \
        --yes \
        --output "$encrypted_file" \
        "$backup_file" 2>>"$LOG_FILE"; then

        # Remove unencrypted file
        rm "$backup_file"
        success "Backup encrypted: $encrypted_file"
        echo "$encrypted_file"
    else
        error "Backup encryption failed"
        return 1
    fi
}

upload_to_s3() {
    local backup_file="$1"
    local s3_key="semantic-processor/backups/$(basename "$backup_file")"

    if [[ -z "$S3_BUCKET" ]]; then
        log "No S3 bucket configured, skipping upload"
        return 0
    fi

    log "Uploading backup to S3: s3://$S3_BUCKET/$s3_key"

    if aws s3 cp "$backup_file" "s3://$S3_BUCKET/$s3_key" \
        --storage-class STANDARD_IA 2>>"$LOG_FILE"; then
        success "Backup uploaded to S3"
    else
        error "S3 upload failed"
        return 1
    fi
}

create_backup_manifest() {
    local backup_dir="$1"
    local manifest_file="$backup_dir/backup_manifest.json"

    log "Creating backup manifest: $manifest_file"

    cat > "$manifest_file" << EOF
{
    "backup_id": "${BACKUP_PREFIX}_${TIMESTAMP}",
    "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
    "version": "$(cat $PROJECT_ROOT/VERSION 2>/dev/null || echo 'unknown')",
    "type": "full",
    "components": {
        "database": true,
        "application_state": true,
        "monitoring_data": true
    },
    "files": [
$(find "$backup_dir" -type f -name "*.tar.gz" -o -name "*.dump" -o -name "*.gpg" | sed 's/.*/"&"/' | paste -sd, -)
    ],
    "retention_date": "$(date -u -d "+$RETENTION_DAYS days" +%Y-%m-%dT%H:%M:%SZ)",
    "checksum": "$(find "$backup_dir" -type f -exec sha256sum {} \; | sort | sha256sum | cut -d' ' -f1)"
}
EOF

    success "Backup manifest created"
}

# Recovery functions
restore_database() {
    local backup_file="$1"
    local target_db="${2:-$DB_NAME}"

    log "Starting database restoration from $backup_file to $target_db"

    # Check if backup file exists
    if [[ ! -f "$backup_file" ]]; then
        error "Backup file not found: $backup_file"
        return 1
    fi

    # Decrypt if needed
    local restore_file="$backup_file"
    if [[ "$backup_file" == *.gpg ]]; then
        if [[ -z "$ENCRYPTION_KEY" ]]; then
            error "Encrypted backup found but no encryption key provided"
            return 1
        fi

        log "Decrypting backup file..."
        restore_file="${backup_file%.gpg}"

        if ! gpg --decrypt \
            --passphrase "$ENCRYPTION_KEY" \
            --batch \
            --yes \
            --output "$restore_file" \
            "$backup_file" 2>>"$LOG_FILE"; then
            error "Failed to decrypt backup file"
            return 1
        fi
    fi

    # Set password for pg_restore
    export PGPASSWORD="$DB_PASSWORD"

    # Drop existing database (if exists and not production)
    if [[ "$target_db" != "$DB_NAME" ]] || [[ "${ALLOW_DB_DROP:-false}" == "true" ]]; then
        log "Dropping existing database: $target_db"
        dropdb --host="$DB_HOST" --port="$DB_PORT" --username="$DB_USER" "$target_db" 2>/dev/null || true
    fi

    # Create database
    log "Creating database: $target_db"
    createdb --host="$DB_HOST" --port="$DB_PORT" --username="$DB_USER" "$target_db"

    # Restore database
    if pg_restore \
        --host="$DB_HOST" \
        --port="$DB_PORT" \
        --username="$DB_USER" \
        --dbname="$target_db" \
        --jobs="$PARALLEL_JOBS" \
        --verbose \
        "$restore_file" 2>>"$LOG_FILE"; then

        success "Database restoration completed"

        # Clean up decrypted file if it was created
        if [[ "$restore_file" != "$backup_file" ]]; then
            rm "$restore_file"
        fi

        return 0
    else
        error "Database restoration failed"
        return 1
    fi
}

restore_application_state() {
    local backup_dir="$1"

    log "Restoring application state from $backup_dir"

    # Restore configuration files
    if [[ -f "$backup_dir/app_state/config.tar.gz" ]]; then
        log "Restoring configuration files..."
        tar xzf "$backup_dir/app_state/config.tar.gz" -C "$PROJECT_ROOT" 2>>"$LOG_FILE"
    fi

    # Restore SSL certificates
    if [[ -f "$backup_dir/app_state/ssl_certs.tar.gz" ]]; then
        log "Restoring SSL certificates..."
        mkdir -p "$PROJECT_ROOT/deployments/nginx/ssl"
        tar xzf "$backup_dir/app_state/ssl_certs.tar.gz" -C "$PROJECT_ROOT/deployments/nginx" 2>>"$LOG_FILE"
    fi

    # Restore environment file
    if [[ -f "$backup_dir/app_state/.env.backup" ]]; then
        log "Restoring environment file..."
        cp "$backup_dir/app_state/.env.backup" "$PROJECT_ROOT/.env"
    fi

    success "Application state restoration completed"
}

verify_backup() {
    local backup_dir="$1"
    local manifest_file="$backup_dir/backup_manifest.json"

    log "Verifying backup integrity: $backup_dir"

    if [[ ! -f "$manifest_file" ]]; then
        error "Backup manifest not found: $manifest_file"
        return 1
    fi

    # Verify checksum
    local expected_checksum=$(jq -r '.checksum' "$manifest_file")
    local actual_checksum=$(find "$backup_dir" -type f -not -name "backup_manifest.json" -exec sha256sum {} \; | sort | sha256sum | cut -d' ' -f1)

    if [[ "$expected_checksum" == "$actual_checksum" ]]; then
        success "Backup integrity verified"
        return 0
    else
        error "Backup integrity check failed"
        error "Expected: $expected_checksum"
        error "Actual: $actual_checksum"
        return 1
    fi
}

cleanup_old_backups() {
    log "Cleaning up backups older than $RETENTION_DAYS days"

    find "$BACKUP_DIR" -name "${BACKUP_PREFIX}_*" -type d -mtime "+$RETENTION_DAYS" -exec rm -rf {} \; 2>>"$LOG_FILE"

    # Clean up S3 backups if configured
    if [[ -n "$S3_BUCKET" ]]; then
        local cutoff_date=$(date -u -d "-$RETENTION_DAYS days" +%Y-%m-%d)
        aws s3 ls "s3://$S3_BUCKET/semantic-processor/backups/" | \
        awk '{print $1 " " $2 " " $4}' | \
        while read date time file; do
            if [[ "$date" < "$cutoff_date" ]]; then
                log "Deleting old S3 backup: $file"
                aws s3 rm "s3://$S3_BUCKET/semantic-processor/backups/$file" 2>>"$LOG_FILE"
            fi
        done
    fi

    success "Old backup cleanup completed"
}

# Main functions
full_backup() {
    log "Starting full backup process"

    local backup_id="${BACKUP_PREFIX}_${TIMESTAMP}"
    local backup_dir="$BACKUP_DIR/$backup_id"

    # Create backup directory
    mkdir -p "$backup_dir"

    # Wait for database to be ready
    if ! wait_for_database; then
        error "Database not available, backup aborted"
        return 1
    fi

    # Backup database
    local db_backup_file="$backup_dir/database.dump"
    if ! backup_database "$db_backup_file"; then
        error "Database backup failed, aborting"
        return 1
    fi

    # Backup application state
    if ! backup_application_state "$backup_dir"; then
        error "Application state backup failed, continuing..."
    fi

    # Backup monitoring data
    if ! backup_monitoring_data "$backup_dir"; then
        error "Monitoring data backup failed, continuing..."
    fi

    # Create backup manifest
    create_backup_manifest "$backup_dir"

    # Verify backup
    if ! verify_backup "$backup_dir"; then
        error "Backup verification failed"
        return 1
    fi

    # Encrypt backup if configured
    if [[ -n "$ENCRYPTION_KEY" ]]; then
        local archive_file="$BACKUP_DIR/${backup_id}.tar.gz"
        tar czf "$archive_file" -C "$BACKUP_DIR" "$backup_id" 2>>"$LOG_FILE"

        local encrypted_file
        if encrypted_file=$(encrypt_backup "$archive_file"); then
            # Upload encrypted backup to S3
            upload_to_s3 "$encrypted_file"
        fi

        # Remove uncompressed backup directory
        rm -rf "$backup_dir"
    else
        # Create compressed archive
        local archive_file="$BACKUP_DIR/${backup_id}.tar.gz"
        tar czf "$archive_file" -C "$BACKUP_DIR" "$backup_id" 2>>"$LOG_FILE"

        # Upload to S3
        upload_to_s3 "$archive_file"

        # Remove uncompressed backup directory
        rm -rf "$backup_dir"
    fi

    # Cleanup old backups
    cleanup_old_backups

    success "Full backup process completed: $backup_id"
}

disaster_recovery() {
    local backup_path="$1"
    local recovery_mode="${2:-full}"

    log "Starting disaster recovery from: $backup_path"
    log "Recovery mode: $recovery_mode"

    # Extract backup if it's an archive
    local backup_dir="$backup_path"
    if [[ -f "$backup_path" ]]; then
        local extract_dir="$BACKUP_DIR/recovery_$(date +%s)"
        mkdir -p "$extract_dir"

        if [[ "$backup_path" == *.tar.gz ]]; then
            tar xzf "$backup_path" -C "$extract_dir" 2>>"$LOG_FILE"
            backup_dir="$extract_dir/$(ls "$extract_dir" | head -1)"
        elif [[ "$backup_path" == *.gpg ]]; then
            if [[ -z "$ENCRYPTION_KEY" ]]; then
                error "Encrypted backup found but no encryption key provided"
                return 1
            fi

            local decrypted_file="${backup_path%.gpg}"
            gpg --decrypt \
                --passphrase "$ENCRYPTION_KEY" \
                --batch \
                --yes \
                --output "$decrypted_file" \
                "$backup_path" 2>>"$LOG_FILE"

            tar xzf "$decrypted_file" -C "$extract_dir" 2>>"$LOG_FILE"
            backup_dir="$extract_dir/$(ls "$extract_dir" | head -1)"
            rm "$decrypted_file"
        fi
    fi

    # Verify backup before proceeding
    if ! verify_backup "$backup_dir"; then
        error "Backup verification failed, recovery aborted"
        return 1
    fi

    # Stop services
    log "Stopping services for recovery..."
    docker-compose -f "$PROJECT_ROOT/deployments/production/docker-compose.prod.yml" down 2>>"$LOG_FILE"

    # Restore database
    if [[ "$recovery_mode" == "full" ]] || [[ "$recovery_mode" == "database" ]]; then
        local db_backup_file="$backup_dir/database.dump"
        if [[ -f "$db_backup_file" ]]; then
            restore_database "$db_backup_file"
        else
            error "Database backup file not found: $db_backup_file"
        fi
    fi

    # Restore application state
    if [[ "$recovery_mode" == "full" ]] || [[ "$recovery_mode" == "application" ]]; then
        restore_application_state "$backup_dir"
    fi

    # Start services
    log "Starting services after recovery..."
    docker-compose -f "$PROJECT_ROOT/deployments/production/docker-compose.prod.yml" up -d 2>>"$LOG_FILE"

    # Wait for services to be ready
    sleep 30

    # Verify recovery
    if verify_recovery; then
        success "Disaster recovery completed successfully"
    else
        error "Disaster recovery verification failed"
        return 1
    fi
}

verify_recovery() {
    log "Verifying recovery..."

    # Check if database is accessible
    if ! wait_for_database; then
        error "Database is not accessible after recovery"
        return 1
    fi

    # Check if application is responding
    local max_attempts=30
    local attempt=1

    while [[ $attempt -le $max_attempts ]]; do
        if curl -f http://localhost:8080/health &>/dev/null; then
            success "Application is responding"
            return 0
        fi

        log "Application not responding, attempt $attempt/$max_attempts"
        sleep 2
        ((attempt++))
    done

    error "Application is not responding after recovery"
    return 1
}

show_usage() {
    cat << EOF
Usage: $0 [COMMAND] [OPTIONS]

Commands:
    backup              Create a full backup
    restore [PATH]      Restore from backup path
    verify [PATH]       Verify backup integrity
    cleanup             Clean up old backups
    status              Show backup status

Examples:
    $0 backup
    $0 restore /backups/semantic_processor_backup_20240101_120000
    $0 verify /backups/semantic_processor_backup_20240101_120000.tar.gz
    $0 cleanup

Environment Variables:
    DB_HOST, DB_PORT, DB_NAME, DB_USER, DB_PASSWORD
    BACKUP_DIR, RETENTION_DAYS, S3_BUCKET, ENCRYPTION_KEY

Configuration:
    Edit $CONFIG_FILE to set default values
EOF
}

# Main execution
main() {
    local command="${1:-}"

    if [[ $# -eq 0 ]]; then
        show_usage
        exit 1
    fi

    # Check dependencies
    check_dependencies

    case "$command" in
        backup)
            full_backup
            ;;
        restore)
            if [[ $# -lt 2 ]]; then
                error "Restore command requires backup path"
                exit 1
            fi
            disaster_recovery "$2" "${3:-full}"
            ;;
        verify)
            if [[ $# -lt 2 ]]; then
                error "Verify command requires backup path"
                exit 1
            fi
            verify_backup "$2"
            ;;
        cleanup)
            cleanup_old_backups
            ;;
        status)
            echo "Backup directory: $BACKUP_DIR"
            echo "Retention days: $RETENTION_DAYS"
            echo "S3 bucket: ${S3_BUCKET:-none}"
            echo "Encryption: ${ENCRYPTION_KEY:+enabled}"
            echo ""
            echo "Recent backups:"
            ls -la "$BACKUP_DIR" | grep "${BACKUP_PREFIX}_" | tail -5
            ;;
        *)
            error "Unknown command: $command"
            show_usage
            exit 1
            ;;
    esac
}

# Execute main function with all arguments
main "$@"