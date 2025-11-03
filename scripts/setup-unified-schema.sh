#!/bin/bash

# Setup script for Unified Chunk System Database Schema
# This script creates the unified database structure with proper error handling

set -e  # Exit on any error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
DATABASE_DIR="$PROJECT_ROOT/database"
SCHEMA_FILE="$DATABASE_DIR/unified_chunk_schema.sql"
INDEX_FILE="$DATABASE_DIR/index_optimization.sql"

# Default values
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_NAME="${DB_NAME:-semantic_processor}"
DB_USER="${DB_USER:-postgres}"
DB_SCHEMA="${DB_SCHEMA:-public}"

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to check if required files exist
check_files() {
    print_status "Checking required files..."
    
    if [[ ! -f "$SCHEMA_FILE" ]]; then
        print_error "Schema file not found: $SCHEMA_FILE"
        exit 1
    fi
    
    if [[ ! -f "$INDEX_FILE" ]]; then
        print_warning "Index optimization file not found: $INDEX_FILE"
        print_warning "Continuing without index optimization..."
    fi
    
    print_success "Required files found"
}

# Function to check database connection
check_connection() {
    print_status "Testing database connection..."
    
    if ! psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "SELECT 1;" > /dev/null 2>&1; then
        print_error "Cannot connect to database"
        print_error "Please check your database connection parameters:"
        print_error "  Host: $DB_HOST"
        print_error "  Port: $DB_PORT"
        print_error "  Database: $DB_NAME"
        print_error "  User: $DB_USER"
        exit 1
    fi
    
    print_success "Database connection successful"
}

# Function to backup existing schema (if any)
backup_existing_schema() {
    print_status "Checking for existing schema..."
    
    local backup_file="$DATABASE_DIR/backup_$(date +%Y%m%d_%H%M%S).sql"
    
    # Check if chunks table exists
    if psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "SELECT 1 FROM information_schema.tables WHERE table_name = 'chunks';" | grep -q "1 row"; then
        print_warning "Existing chunks table found. Creating backup..."
        
        pg_dump -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" \
                --schema-only --table=chunks --table=chunk_tags --table=chunk_hierarchy --table=chunk_search_cache \
                > "$backup_file" 2>/dev/null || true
        
        if [[ -f "$backup_file" && -s "$backup_file" ]]; then
            print_success "Backup created: $backup_file"
        else
            print_warning "Backup creation failed or empty"
            rm -f "$backup_file"
        fi
    else
        print_status "No existing schema found"
    fi
}

# Function to execute SQL file with error handling
execute_sql_file() {
    local file_path="$1"
    local description="$2"
    
    print_status "Executing $description..."
    
    if psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -f "$file_path" > /dev/null; then
        print_success "$description completed successfully"
    else
        print_error "$description failed"
        exit 1
    fi
}

# Function to validate schema creation
validate_schema() {
    print_status "Validating schema creation..."
    
    local tables=("chunks" "chunk_tags" "chunk_hierarchy" "chunk_search_cache")
    local missing_tables=()
    
    for table in "${tables[@]}"; do
        if ! psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "SELECT 1 FROM information_schema.tables WHERE table_name = '$table';" | grep -q "1 row"; then
            missing_tables+=("$table")
        fi
    done
    
    if [[ ${#missing_tables[@]} -eq 0 ]]; then
        print_success "All tables created successfully"
    else
        print_error "Missing tables: ${missing_tables[*]}"
        exit 1
    fi
    
    # Check materialized view
    if psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "SELECT 1 FROM information_schema.views WHERE table_name = 'tag_statistics';" | grep -q "1 row"; then
        print_success "Materialized view created successfully"
    else
        print_warning "Materialized view 'tag_statistics' not found"
    fi
    
    # Check triggers
    local trigger_count=$(psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -t -c "SELECT COUNT(*) FROM information_schema.triggers WHERE trigger_name LIKE 'trigger_%chunk%';")
    
    if [[ $trigger_count -ge 3 ]]; then
        print_success "Database triggers created successfully ($trigger_count triggers)"
    else
        print_warning "Expected at least 3 triggers, found $trigger_count"
    fi
}

# Function to display schema information
display_schema_info() {
    print_status "Schema Information:"
    
    echo "Tables created:"
    psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "
        SELECT 
            table_name,
            (SELECT COUNT(*) FROM information_schema.columns WHERE table_name = t.table_name) as column_count
        FROM information_schema.tables t 
        WHERE table_name IN ('chunks', 'chunk_tags', 'chunk_hierarchy', 'chunk_search_cache')
        ORDER BY table_name;
    "
    
    echo -e "\nIndexes created:"
    psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "
        SELECT 
            tablename,
            COUNT(*) as index_count
        FROM pg_indexes 
        WHERE tablename IN ('chunks', 'chunk_tags', 'chunk_hierarchy', 'chunk_search_cache')
        GROUP BY tablename
        ORDER BY tablename;
    "
}

# Main execution
main() {
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}  Unified Chunk System Schema Setup    ${NC}"
    echo -e "${BLUE}========================================${NC}"
    echo
    
    # Check prerequisites
    check_files
    check_connection
    
    # Backup existing schema if needed
    backup_existing_schema
    
    # Execute schema creation
    execute_sql_file "$SCHEMA_FILE" "unified chunk schema"
    
    # Execute index optimization if file exists
    if [[ -f "$INDEX_FILE" ]]; then
        execute_sql_file "$INDEX_FILE" "index optimization"
    fi
    
    # Validate creation
    validate_schema
    
    # Display information
    display_schema_info
    
    echo
    print_success "Unified Chunk System schema setup completed successfully!"
    echo -e "${BLUE}========================================${NC}"
}

# Help function
show_help() {
    echo "Usage: $0 [OPTIONS]"
    echo
    echo "Setup the Unified Chunk System database schema"
    echo
    echo "Environment Variables:"
    echo "  DB_HOST     Database host (default: localhost)"
    echo "  DB_PORT     Database port (default: 5432)"
    echo "  DB_NAME     Database name (default: semantic_processor)"
    echo "  DB_USER     Database user (default: postgres)"
    echo "  DB_SCHEMA   Database schema (default: public)"
    echo
    echo "Options:"
    echo "  -h, --help  Show this help message"
    echo
    echo "Example:"
    echo "  DB_HOST=localhost DB_NAME=mydb $0"
}

# Parse command line arguments
case "${1:-}" in
    -h|--help)
        show_help
        exit 0
        ;;
    *)
        main "$@"
        ;;
esac