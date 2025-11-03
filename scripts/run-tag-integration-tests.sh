#!/bin/bash

# Script to run tag operations integration tests
# This script sets up the environment and runs the integration tests

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

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

# Default test database configuration
export DB_HOST="${DB_HOST:-localhost}"
export DB_PORT="${DB_PORT:-5432}"
export DB_NAME="${DB_NAME:-semantic_processor_test}"
export DB_USER="${DB_USER:-postgres}"
export DB_PASSWORD="${DB_PASSWORD:-postgres}"

# Enable integration tests
export RUN_INTEGRATION_TESTS=true

print_status "Tag Operations Integration Tests"
echo "Database Configuration:"
echo "  Host: $DB_HOST"
echo "  Port: $DB_PORT"
echo "  Database: $DB_NAME"
echo "  User: $DB_USER"
echo

# Check if database is accessible
print_status "Checking database connection..."
if ! psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "SELECT 1;" > /dev/null 2>&1; then
    print_error "Cannot connect to test database"
    print_error "Please ensure:"
    print_error "1. PostgreSQL is running"
    print_error "2. Test database '$DB_NAME' exists"
    print_error "3. User '$DB_USER' has access to the database"
    print_error "4. Unified chunk schema is installed"
    echo
    print_status "To create the test database and schema:"
    print_status "1. Create database: createdb -h $DB_HOST -p $DB_PORT -U $DB_USER $DB_NAME"
    print_status "2. Run schema setup: DB_NAME=$DB_NAME ./scripts/setup-unified-schema.sh"
    exit 1
fi

print_success "Database connection successful"

# Check if unified schema exists
print_status "Checking unified chunk schema..."
if ! psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "SELECT 1 FROM information_schema.tables WHERE table_name = 'chunks';" | grep -q "1 row"; then
    print_error "Unified chunk schema not found in test database"
    print_error "Please run the schema setup script first:"
    print_error "  DB_NAME=$DB_NAME ./scripts/setup-unified-schema.sh"
    exit 1
fi

print_success "Unified chunk schema found"

# Run the integration tests
print_status "Running tag operations integration tests..."
echo

cd "$(dirname "$0")/.."

if go test ./services -v -run "TestUnifiedChunkService_TagOperations_RealDatabase|TestUnifiedChunkService_TagOperations_Performance" -timeout 30s; then
    echo
    print_success "All tag operations integration tests passed!"
else
    echo
    print_error "Some integration tests failed"
    exit 1
fi

echo
print_status "Integration test summary:"
echo "✓ Tag creation and assignment"
echo "✓ Tag retrieval operations"
echo "✓ Multi-tag queries (AND/OR logic)"
echo "✓ Tag removal operations"
echo "✓ Cache invalidation"
echo "✓ Error handling"
echo "✓ Performance benchmarks"
echo
print_success "Tag operations implementation is working correctly!"