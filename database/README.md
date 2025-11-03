# Unified Chunk System Database Schema

This directory contains the database schema and setup scripts for the Unified Chunk System, which implements a single-table design for storing all content types (text, tags, templates, slots, pages) with auxiliary tables for performance optimization.

## Files Overview

### Core Schema Files

- **`unified_chunk_schema.sql`** - Main schema creation script with all tables, indexes, triggers, and constraints
- **`validate_schema.sql`** - Comprehensive validation script to test schema integrity
- **`index_optimization.sql`** - Additional performance optimization indexes and monitoring tools

### Setup Scripts

- **`../scripts/setup-unified-schema.sh`** - Automated setup script with error handling and validation

## Database Design

### Main Table: `chunks`

The central table storing all content types with the following key features:

- **Unified Storage**: All content types in one table
- **Type Flags**: Boolean fields (`is_page`, `is_tag`, `is_template`, `is_slot`) to distinguish content types
- **Hierarchical Support**: `parent` field for tree structures
- **Flexible Metadata**: JSONB fields for extensibility
- **Performance Optimized**: Comprehensive indexing strategy

```sql
CREATE TABLE chunks (
    chunk_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    contents TEXT NOT NULL,
    parent UUID REFERENCES chunks(chunk_id),
    page UUID REFERENCES chunks(chunk_id),
    is_page BOOLEAN DEFAULT FALSE,
    is_tag BOOLEAN DEFAULT FALSE,
    is_template BOOLEAN DEFAULT FALSE,
    is_slot BOOLEAN DEFAULT FALSE,
    ref TEXT,
    tags JSONB,
    metadata JSONB,
    created_time TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_updated TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
```

### Auxiliary Tables

#### `chunk_tags` - Tag Relationship Optimization
- Optimizes many-to-many tag relationships
- Enables millisecond-level tag queries
- Automatically synchronized with main table via triggers

#### `chunk_hierarchy` - Hierarchy Query Optimization  
- Stores pre-computed hierarchy relationships
- Eliminates recursive queries for better performance
- Includes complete path information

#### `chunk_search_cache` - Query Result Caching
- Caches complex search query results
- Configurable TTL and hit counting
- Automatic cleanup of expired entries

### Materialized Views

#### `tag_statistics`
- Pre-computed tag usage statistics
- Refreshed automatically when tag relationships change
- Optimizes tag analytics queries

## Performance Features

### Indexing Strategy

1. **Type-specific indexes** - Partial indexes for each content type
2. **Full-text search** - GIN indexes for content search
3. **JSONB indexes** - Optimized metadata and tag queries
4. **Composite indexes** - Multi-column indexes for common query patterns
5. **Time-based indexes** - Optimized temporal queries

### Automatic Triggers

1. **Tag Synchronization** - Maintains consistency between main table and `chunk_tags`
2. **Hierarchy Maintenance** - Updates `chunk_hierarchy` when parent relationships change
3. **Timestamp Updates** - Automatic `last_updated` field maintenance
4. **Statistics Refresh** - Updates materialized views when data changes

## Setup Instructions

### Prerequisites

- PostgreSQL 12+ with UUID extension
- Database user with CREATE privileges
- `psql` command-line tool

### Quick Setup

1. **Set environment variables:**
```bash
export DB_HOST=localhost
export DB_PORT=5432
export DB_NAME=semantic_processor
export DB_USER=postgres
```

2. **Run the setup script:**
```bash
./scripts/setup-unified-schema.sh
```

### Manual Setup

1. **Create the schema:**
```bash
psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -f database/unified_chunk_schema.sql
```

2. **Validate the installation:**
```bash
psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -f database/validate_schema.sql
```

3. **Optional - Add performance optimizations:**
```bash
psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -f database/index_optimization.sql
```

## Usage Examples

### Basic Operations

```sql
-- Create a page
INSERT INTO chunks (contents, is_page) 
VALUES ('My Page Title', true);

-- Create a tag
INSERT INTO chunks (contents, is_tag) 
VALUES ('important', true);

-- Create content with tags
INSERT INTO chunks (contents, tags) 
VALUES ('Some content', '["tag-uuid-1", "tag-uuid-2"]');

-- Create hierarchical content
INSERT INTO chunks (contents, parent) 
VALUES ('Child content', 'parent-uuid');
```

### Optimized Queries

```sql
-- Fast tag-based search using auxiliary table
SELECT c.* FROM chunks c
JOIN chunk_tags ct ON c.chunk_id = ct.source_chunk_id
WHERE ct.tag_chunk_id = 'tag-uuid';

-- Efficient hierarchy traversal
SELECT c.*, ch.depth FROM chunks c
JOIN chunk_hierarchy ch ON c.chunk_id = ch.descendant_id
WHERE ch.ancestor_id = 'parent-uuid' AND ch.depth <= 3;

-- Full-text search with ranking
SELECT *, ts_rank(to_tsvector('english', contents), query) as rank
FROM chunks, to_tsquery('english', 'search terms') query
WHERE to_tsvector('english', contents) @@ query
ORDER BY rank DESC;
```

## Monitoring and Maintenance

### Performance Monitoring

```sql
-- Check index usage
SELECT * FROM index_usage WHERE times_used < 100;

-- Analyze query performance
SELECT * FROM analyze_index_performance();

-- Monitor table sizes
SELECT * FROM table_sizes;
```

### Maintenance Tasks

```sql
-- Clean expired cache entries
SELECT cleanup_expired_search_cache();

-- Refresh tag statistics
REFRESH MATERIALIZED VIEW CONCURRENTLY tag_statistics;

-- Check for table bloat
SELECT * FROM table_bloat_check WHERE dead_tuple_percent > 10;
```

## Migration from Existing Schema

If you have existing data in separate tables, you'll need to:

1. **Backup existing data**
2. **Map existing content to unified structure**
3. **Migrate data with proper type flags**
4. **Validate data integrity**
5. **Update application code**

See the migration guide in the main documentation for detailed steps.

## Troubleshooting

### Common Issues

1. **Trigger failures** - Check constraint violations in source data
2. **Performance issues** - Analyze query plans and index usage
3. **Consistency errors** - Run validation script to identify issues

### Debug Queries

```sql
-- Check trigger status
SELECT * FROM information_schema.triggers 
WHERE trigger_name LIKE 'trigger_%chunk%';

-- Verify data consistency
SELECT COUNT(*) FROM chunks WHERE tags IS NOT NULL;
SELECT COUNT(*) FROM chunk_tags;

-- Check constraint violations
SELECT * FROM chunks WHERE length(trim(contents)) = 0;
```

## Schema Version

- **Version**: 1.0.0
- **Compatible with**: PostgreSQL 12+
- **Last updated**: 2025-01-20
- **Breaking changes**: This is the initial unified schema implementation

For questions or issues, refer to the main project documentation or create an issue in the project repository.