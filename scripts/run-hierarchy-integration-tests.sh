#!/bin/bash

# Script to run hierarchy operations integration tests for the Unified Chunk System
# This script sets up the test environment and runs hierarchy-specific integration tests

set -e

echo "🔧 Setting up Hierarchy Operations Integration Tests..."

# Check if required environment variables are set
if [ -z "$DB_HOST" ]; then
    export DB_HOST="localhost"
fi

if [ -z "$DB_PORT" ]; then
    export DB_PORT="5432"
fi

if [ -z "$DB_NAME" ]; then
    export DB_NAME="semantic_processor_test"
fi

if [ -z "$DB_USER" ]; then
    export DB_USER="postgres"
fi

if [ -z "$DB_PASSWORD" ]; then
    export DB_PASSWORD="postgres"
fi

echo "📊 Database Configuration:"
echo "  Host: $DB_HOST"
echo "  Port: $DB_PORT"
echo "  Database: $DB_NAME"
echo "  User: $DB_USER"

# Test database connection
echo "🔍 Testing database connection..."
if ! pg_isready -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" >/dev/null 2>&1; then
    echo "❌ Database connection failed. Please ensure PostgreSQL is running and accessible."
    echo "   Connection string: host=$DB_HOST port=$DB_PORT user=$DB_USER dbname=$DB_NAME"
    exit 1
fi

echo "✅ Database connection successful"

# Check if unified schema exists
echo "🔍 Checking if unified schema exists..."
SCHEMA_EXISTS=$(PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -t -c "SELECT EXISTS(SELECT 1 FROM information_schema.tables WHERE table_name = 'chunks');" 2>/dev/null | tr -d ' \n')

if [ "$SCHEMA_EXISTS" != "t" ]; then
    echo "⚠️  Unified schema not found. Setting up schema..."
    
    # Run the unified schema setup script
    if [ -f "scripts/setup-unified-schema.sh" ]; then
        bash scripts/setup-unified-schema.sh
    else
        echo "❌ Schema setup script not found. Please run setup-unified-schema.sh first."
        exit 1
    fi
else
    echo "✅ Unified schema found"
fi

# Verify required tables exist
echo "🔍 Verifying required tables..."
REQUIRED_TABLES=("chunks" "chunk_tags" "chunk_hierarchy" "chunk_search_cache")

for table in "${REQUIRED_TABLES[@]}"; do
    TABLE_EXISTS=$(PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -t -c "SELECT EXISTS(SELECT 1 FROM information_schema.tables WHERE table_name = '$table');" 2>/dev/null | tr -d ' \n')
    
    if [ "$TABLE_EXISTS" != "t" ]; then
        echo "❌ Required table '$table' not found"
        exit 1
    else
        echo "✅ Table '$table' found"
    fi
done

# Check if triggers exist
echo "🔍 Verifying database triggers..."
TRIGGER_EXISTS=$(PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -t -c "SELECT EXISTS(SELECT 1 FROM information_schema.triggers WHERE trigger_name = 'trigger_sync_chunk_hierarchy');" 2>/dev/null | tr -d ' \n')

if [ "$TRIGGER_EXISTS" != "t" ]; then
    echo "⚠️  Hierarchy trigger not found. This may affect hierarchy operations."
else
    echo "✅ Hierarchy trigger found"
fi

# Set environment variable to enable integration tests
export RUN_INTEGRATION_TESTS=true

echo ""
echo "🚀 Running Hierarchy Operations Integration Tests..."
echo ""

# Run hierarchy-specific integration tests
go test -v -timeout=30m -run "TestUnifiedChunkService_HierarchyOperations" ./services

TEST_EXIT_CODE=$?

echo ""
if [ $TEST_EXIT_CODE -eq 0 ]; then
    echo "✅ All hierarchy integration tests passed!"
    
    # Display performance summary if available
    echo ""
    echo "📊 Test Summary:"
    echo "  - Hierarchy operations (GetChildren, GetDescendants, GetAncestors, MoveChunk) ✅"
    echo "  - Error handling and validation ✅"
    echo "  - Cache invalidation ✅"
    echo "  - Performance benchmarks ✅"
    
else
    echo "❌ Some hierarchy integration tests failed (exit code: $TEST_EXIT_CODE)"
    echo ""
    echo "🔧 Troubleshooting tips:"
    echo "  1. Ensure the database schema is up to date"
    echo "  2. Check database permissions"
    echo "  3. Verify all required triggers are installed"
    echo "  4. Check database logs for errors"
fi

echo ""
echo "🏁 Hierarchy integration tests completed"

exit $TEST_EXIT_CODE