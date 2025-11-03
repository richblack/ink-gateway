# Data Migration and Upgrade Guide

## Table of Contents

1. [Overview](#overview)
2. [Pre-Migration Planning](#pre-migration-planning)
3. [Database Schema Migrations](#database-schema-migrations)
4. [Unified Chunk System Migration](#unified-chunk-system-migration)
5. [Data Consistency Verification](#data-consistency-verification)
6. [Application Upgrade Procedures](#application-upgrade-procedures)
7. [Rollback Procedures](#rollback-procedures)
8. [Post-Migration Validation](#post-migration-validation)
9. [Performance Optimization After Migration](#performance-optimization-after-migration)
10. [Troubleshooting Migration Issues](#troubleshooting-migration-issues)

## Overview

This guide provides comprehensive procedures for migrating data and upgrading the Semantic Text Processor system, particularly focusing on transitions to the unified chunk system and new table structures.

### Migration Types

1. **Schema Migrations**: Database structure updates
2. **Data Format Migrations**: Content structure changes
3. **System Upgrades**: Application version updates
4. **Feature Migrations**: Legacy to new feature transitions

### Critical Considerations

- **Zero-downtime**: Procedures support production systems
- **Data Integrity**: All migrations preserve data consistency
- **Rollback Safety**: Every migration includes rollback procedures
- **Performance Impact**: Minimal impact on system performance

## Pre-Migration Planning

### System Assessment

**1. Current System Analysis**:
```bash
# Check current schema version
curl http://localhost:8080/api/v1/health | jq '.components.database.schema_version'

# Analyze data volume
curl "http://localhost:8080/api/v1/texts?page_size=1" | jq '.pagination.total'
curl "http://localhost:8080/api/v1/chunks?page_size=1" | jq '.pagination.total'

# Check system resources
free -h
df -h
iostat -x 1 3
```

**2. Backup Procedures**:
```bash
# Create complete system backup
mkdir -p /backup/$(date +%Y%m%d_%H%M%S)
export BACKUP_DIR="/backup/$(date +%Y%m%d_%H%M%S)"

# Backup application configuration
cp -r /etc/semantic-text-processor/ $BACKUP_DIR/config/

# Backup Supabase data (via Supabase CLI)
supabase db dump --db-url "$SUPABASE_URL" > $BACKUP_DIR/database_dump.sql

# Backup application binary
cp /usr/local/bin/semantic-text-processor $BACKUP_DIR/
```

**3. Migration Checklist**:
- [ ] Full system backup completed
- [ ] Migration scripts tested in staging
- [ ] Rollback procedures verified
- [ ] Maintenance window scheduled
- [ ] Monitoring alerts configured
- [ ] Team notification sent
- [ ] Post-migration validation plan ready

### Resource Planning

**1. Disk Space Requirements**:
```bash
# Calculate required space (typically 2-3x current data size)
CURRENT_SIZE=$(du -sh /var/lib/semantic-text-processor | cut -f1)
echo "Current data size: $CURRENT_SIZE"
echo "Required free space: $(echo "$CURRENT_SIZE * 3" | bc)B"

# Check available space
df -h /var/lib/semantic-text-processor
```

**2. Memory and CPU Planning**:
```bash
# Migration typically requires 50-100% more resources temporarily
CURRENT_MEMORY=$(ps -p $(pgrep semantic-text-processor) -o rss= | awk '{print $1/1024}')
echo "Current memory usage: ${CURRENT_MEMORY}MB"
echo "Recommended memory during migration: $((CURRENT_MEMORY * 2))MB"
```

## Database Schema Migrations

### Unified Chunk System Migration

**Migration Objective**: Consolidate separate tables (texts, chunks, templates, tags) into the unified chunk system.

**1. Schema Preparation**:
```sql
-- Create new unified chunk table (if not exists)
CREATE TABLE IF NOT EXISTS chunks_unified (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    text_id UUID REFERENCES texts(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    is_template BOOLEAN DEFAULT FALSE,
    is_slot BOOLEAN DEFAULT FALSE,
    parent_chunk_id UUID REFERENCES chunks_unified(id) ON DELETE CASCADE,
    template_chunk_id UUID REFERENCES chunks_unified(id) ON DELETE SET NULL,
    slot_value TEXT,
    indent_level INTEGER DEFAULT 0,
    sequence_number INTEGER,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indexes for performance
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_chunks_unified_text_id
    ON chunks_unified(text_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_chunks_unified_parent_id
    ON chunks_unified(parent_chunk_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_chunks_unified_template_id
    ON chunks_unified(template_chunk_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_chunks_unified_is_template
    ON chunks_unified(is_template);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_chunks_unified_metadata
    ON chunks_unified USING GIN(metadata);
```

**2. Data Migration Script**:
```sql
-- Start migration transaction
BEGIN;

-- Migrate regular chunks
INSERT INTO chunks_unified (
    id, text_id, content, is_template, parent_chunk_id,
    indent_level, sequence_number, created_at, updated_at
)
SELECT
    id, text_id, content, FALSE as is_template, parent_chunk_id,
    indent_level, sequence_number, created_at, updated_at
FROM chunks_legacy
WHERE NOT EXISTS (SELECT 1 FROM chunks_unified WHERE id = chunks_legacy.id);

-- Migrate template chunks
INSERT INTO chunks_unified (
    id, text_id, content, is_template, is_slot,
    indent_level, sequence_number, created_at, updated_at
)
SELECT
    t.id, t.text_id, t.content, TRUE as is_template, FALSE as is_slot,
    0 as indent_level, 0 as sequence_number, t.created_at, t.updated_at
FROM templates_legacy t
WHERE NOT EXISTS (SELECT 1 FROM chunks_unified WHERE id = t.id);

-- Migrate template slots
INSERT INTO chunks_unified (
    id, text_id, content, is_template, is_slot, parent_chunk_id,
    template_chunk_id, slot_value, indent_level, sequence_number,
    created_at, updated_at
)
SELECT
    ts.id, t.text_id, ts.slot_name as content, FALSE as is_template,
    TRUE as is_slot, t.id as parent_chunk_id, t.id as template_chunk_id,
    COALESCE(tsi.slot_value, '') as slot_value,
    1 as indent_level, ts.slot_order as sequence_number,
    ts.created_at, ts.updated_at
FROM template_slots_legacy ts
JOIN templates_legacy t ON ts.template_id = t.id
LEFT JOIN template_slot_instances_legacy tsi ON ts.id = tsi.slot_id
WHERE NOT EXISTS (SELECT 1 FROM chunks_unified WHERE id = ts.id);

-- Migrate chunk tags (many-to-many relationship)
-- Tags become special chunks with is_template=FALSE, is_slot=FALSE
INSERT INTO chunks_unified (
    text_id, content, is_template, is_slot,
    indent_level, sequence_number, metadata, created_at, updated_at
)
SELECT DISTINCT
    c.text_id, ct.tag_content as content, FALSE as is_template, FALSE as is_slot,
    0 as indent_level, 0 as sequence_number,
    jsonb_build_object('type', 'tag', 'original_chunk_ids',
        jsonb_agg(DISTINCT ct.chunk_id)) as metadata,
    MIN(ct.created_at) as created_at, NOW() as updated_at
FROM chunk_tags_legacy ct
JOIN chunks_legacy c ON ct.chunk_id = c.id
GROUP BY c.text_id, ct.tag_content
ON CONFLICT DO NOTHING;

-- Create chunk-tag relationships in metadata
UPDATE chunks_unified
SET metadata = metadata || jsonb_build_object('tags',
    (SELECT jsonb_agg(DISTINCT tag_chunks.content)
     FROM chunks_unified tag_chunks
     WHERE tag_chunks.metadata->>'type' = 'tag'
     AND tag_chunks.metadata->'original_chunk_ids' ? chunks_unified.id::text))
WHERE id IN (
    SELECT DISTINCT chunk_id::uuid
    FROM chunk_tags_legacy ct
    WHERE chunk_id IS NOT NULL
);

COMMIT;
```

**3. Migration Verification**:
```sql
-- Verify migration completeness
SELECT
    'chunks_legacy' as table_name, COUNT(*) as count
FROM chunks_legacy
UNION ALL
SELECT
    'chunks_unified' as table_name, COUNT(*) as count
FROM chunks_unified
UNION ALL
SELECT
    'templates_legacy' as table_name, COUNT(*) as count
FROM templates_legacy
UNION ALL
SELECT
    'chunk_tags_legacy' as table_name, COUNT(*) as count
FROM chunk_tags_legacy;

-- Check data integrity
SELECT
    COUNT(*) as orphaned_chunks
FROM chunks_unified
WHERE parent_chunk_id IS NOT NULL
AND parent_chunk_id NOT IN (SELECT id FROM chunks_unified);

-- Verify template relationships
SELECT
    COUNT(*) as invalid_template_refs
FROM chunks_unified
WHERE template_chunk_id IS NOT NULL
AND template_chunk_id NOT IN (SELECT id FROM chunks_unified WHERE is_template = TRUE);
```

### Embedding System Migration

**1. Vector Index Migration**:
```sql
-- Create new embedding table structure
CREATE TABLE IF NOT EXISTS embeddings_v2 (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    chunk_id UUID REFERENCES chunks_unified(id) ON DELETE CASCADE,
    vector vector(1536),  -- Adjust dimension as needed
    model_name TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(chunk_id, model_name)
);

-- Create vector index
CREATE INDEX CONCURRENTLY embeddings_v2_vector_idx
    ON embeddings_v2 USING ivfflat (vector vector_cosine_ops)
    WITH (lists = 100);

-- Migrate existing embeddings
INSERT INTO embeddings_v2 (chunk_id, vector, model_name, created_at)
SELECT
    e.chunk_id, e.vector, COALESCE(e.model_name, 'text-embedding-ada-002') as model_name, e.created_at
FROM embeddings_legacy e
JOIN chunks_unified c ON e.chunk_id = c.id
ON CONFLICT (chunk_id, model_name) DO NOTHING;
```

### Graph System Migration

**1. Apache AGE Schema Update**:
```sql
-- Create new graph schema for unified chunks
SELECT create_graph('knowledge_graph_v2');

-- Migrate nodes
SELECT * FROM cypher('knowledge_graph_v2', $$
    LOAD CSV WITH HEADERS FROM 'file:///tmp/nodes_export.csv' AS row
    CREATE (n:Entity {
        id: row.chunk_id,
        name: row.entity_name,
        type: row.entity_type,
        properties: row.properties
    })
$$) AS (n agtype);

-- Migrate relationships
SELECT * FROM cypher('knowledge_graph_v2', $$
    LOAD CSV WITH HEADERS FROM 'file:///tmp/edges_export.csv' AS row
    MATCH (a:Entity {id: row.source_chunk_id})
    MATCH (b:Entity {id: row.target_chunk_id})
    CREATE (a)-[r:RELATES {
        type: row.relationship_type,
        properties: row.properties
    }]->(b)
$$) AS (r agtype);
```

## Data Consistency Verification

### Automated Consistency Checks

**1. Referential Integrity Check**:
```bash
#!/bin/bash
# consistency_check.sh

echo "Running data consistency checks..."

# Check for orphaned chunks
ORPHANED_CHUNKS=$(curl -s "$API_BASE/admin/consistency/orphaned-chunks" | jq '.count')
if [ "$ORPHANED_CHUNKS" -gt 0 ]; then
    echo "WARNING: Found $ORPHANED_CHUNKS orphaned chunks"
    exit 1
fi

# Check template consistency
INVALID_TEMPLATES=$(curl -s "$API_BASE/admin/consistency/invalid-templates" | jq '.count')
if [ "$INVALID_TEMPLATES" -gt 0 ]; then
    echo "WARNING: Found $INVALID_TEMPLATES invalid template references"
    exit 1
fi

# Check embedding consistency
MISSING_EMBEDDINGS=$(curl -s "$API_BASE/admin/consistency/missing-embeddings" | jq '.count')
if [ "$MISSING_EMBEDDINGS" -gt 0 ]; then
    echo "WARNING: Found $MISSING_EMBEDDINGS chunks without embeddings"
fi

echo "Consistency checks passed!"
```

**2. Performance Validation**:
```bash
#!/bin/bash
# performance_validation.sh

echo "Running performance validation..."

# Test search performance
SEARCH_TIME=$(time curl -s -X POST "$API_BASE/search/semantic" \
    -H "Content-Type: application/json" \
    -d '{"query": "test query", "limit": 10}' | \
    grep real | awk '{print $2}')

if (( $(echo "$SEARCH_TIME > 1.0" | bc -l) )); then
    echo "WARNING: Search performance degraded: ${SEARCH_TIME}s"
fi

# Test API response times
API_RESPONSE_TIME=$(curl -o /dev/null -s -w '%{time_total}\n' "$API_BASE/health")
if (( $(echo "$API_RESPONSE_TIME > 0.5" | bc -l) )); then
    echo "WARNING: API response time degraded: ${API_RESPONSE_TIME}s"
fi

echo "Performance validation completed!"
```

## Application Upgrade Procedures

### Rolling Deployment Strategy

**1. Blue-Green Deployment**:
```bash
#!/bin/bash
# blue_green_deploy.sh

# Current (blue) environment
BLUE_PORT=8080
GREEN_PORT=8081

echo "Starting green deployment..."

# Deploy new version to green environment
docker run -d \
    --name semantic-text-processor-green \
    -p $GREEN_PORT:8080 \
    -e SUPABASE_URL="$SUPABASE_URL" \
    -e SUPABASE_API_KEY="$SUPABASE_API_KEY" \
    semantic-text-processor:latest

# Wait for green to be healthy
sleep 30
GREEN_HEALTH=$(curl -s http://localhost:$GREEN_PORT/api/v1/health | jq -r '.status')

if [ "$GREEN_HEALTH" = "healthy" ]; then
    echo "Green environment healthy, switching traffic..."

    # Update load balancer to point to green
    # This is environment-specific (nginx, haproxy, etc.)
    update_load_balancer_config $GREEN_PORT

    # Wait for traffic to drain from blue
    sleep 60

    # Stop blue environment
    docker stop semantic-text-processor-blue
    docker rm semantic-text-processor-blue

    # Rename green to blue for next deployment
    docker rename semantic-text-processor-green semantic-text-processor-blue

    echo "Deployment completed successfully!"
else
    echo "Green environment unhealthy, rolling back..."
    docker stop semantic-text-processor-green
    docker rm semantic-text-processor-green
    exit 1
fi
```

**2. Configuration Migration**:
```bash
#!/bin/bash
# config_migration.sh

# Backup current configuration
cp /etc/semantic-text-processor/config.yaml \
   /backup/config_$(date +%Y%m%d_%H%M%S).yaml

# Update configuration with new unified chunk system settings
cat > /etc/semantic-text-processor/config.yaml << EOF
server:
  port: "8080"
  read_timeout: "30s"
  write_timeout: "30s"
  idle_timeout: "120s"

database:
  max_open_connections: 25
  max_idle_connections: 5
  connection_lifetime: "5m"

features:
  use_unified_handlers: true
  use_enhanced_search: true
  enable_batch_operations: true

cache:
  enabled: true
  max_size: 1000
  default_ttl: "1h"

performance:
  metrics_enabled: true
  monitoring_enabled: true
  slow_query_threshold: "500ms"

logging:
  level: "info"
  format: "json"
  output: "/var/log/semantic-text-processor.log"
EOF

# Validate configuration
semantic-text-processor --config-check || exit 1

echo "Configuration migration completed!"
```

## Rollback Procedures

### Emergency Rollback

**1. Application Rollback**:
```bash
#!/bin/bash
# emergency_rollback.sh

echo "Starting emergency rollback..."

# Stop current service
systemctl stop semantic-text-processor

# Restore previous binary
cp /backup/semantic-text-processor-previous \
   /usr/local/bin/semantic-text-processor

# Restore previous configuration
cp /backup/config_previous.yaml \
   /etc/semantic-text-processor/config.yaml

# Start service with previous version
systemctl start semantic-text-processor

# Verify rollback
sleep 10
HEALTH_STATUS=$(curl -s http://localhost:8080/api/v1/health | jq -r '.status')

if [ "$HEALTH_STATUS" = "healthy" ]; then
    echo "Rollback completed successfully!"
else
    echo "Rollback failed! Manual intervention required."
    exit 1
fi
```

**2. Database Rollback**:
```sql
-- Rollback database changes (use with extreme caution)
BEGIN;

-- Rename tables back to legacy names
ALTER TABLE chunks_unified RENAME TO chunks_unified_backup;
ALTER TABLE chunks_legacy RENAME TO chunks;
ALTER TABLE templates_legacy RENAME TO templates;
ALTER TABLE chunk_tags_legacy RENAME TO chunk_tags;

-- Drop new indexes
DROP INDEX IF EXISTS idx_chunks_unified_text_id;
DROP INDEX IF EXISTS idx_chunks_unified_parent_id;
DROP INDEX IF EXISTS idx_chunks_unified_template_id;

-- Restore original indexes
CREATE INDEX IF NOT EXISTS idx_chunks_text_id ON chunks(text_id);
CREATE INDEX IF NOT EXISTS idx_chunks_parent_id ON chunks(parent_chunk_id);

COMMIT;
```

## Post-Migration Validation

### Comprehensive System Testing

**1. Functional Testing**:
```bash
#!/bin/bash
# post_migration_tests.sh

echo "Running post-migration validation..."

# Test text creation
TEXT_RESPONSE=$(curl -s -X POST "$API_BASE/texts" \
    -H "Content-Type: application/json" \
    -d '{"content": "Test migration content", "title": "Migration Test"}')
TEXT_ID=$(echo "$TEXT_RESPONSE" | jq -r '.id')

if [ "$TEXT_ID" = "null" ]; then
    echo "FAIL: Text creation failed"
    exit 1
fi

# Test chunk operations
CHUNKS_RESPONSE=$(curl -s "$API_BASE/texts/$TEXT_ID")
CHUNK_COUNT=$(echo "$CHUNKS_RESPONSE" | jq '.chunks | length')

if [ "$CHUNK_COUNT" -eq 0 ]; then
    echo "FAIL: No chunks created for text"
    exit 1
fi

# Test search functionality
SEARCH_RESPONSE=$(curl -s -X POST "$API_BASE/search/semantic" \
    -H "Content-Type: application/json" \
    -d '{"query": "migration", "limit": 5}')
SEARCH_RESULTS=$(echo "$SEARCH_RESPONSE" | jq '.results | length')

if [ "$SEARCH_RESULTS" -eq 0 ]; then
    echo "WARN: No search results found"
fi

# Test template creation
TEMPLATE_RESPONSE=$(curl -s -X POST "$API_BASE/templates" \
    -H "Content-Type: application/json" \
    -d '{"template_name": "Test Template", "slot_names": ["test_slot"]}')
TEMPLATE_ID=$(echo "$TEMPLATE_RESPONSE" | jq -r '.template_id')

if [ "$TEMPLATE_ID" = "null" ]; then
    echo "FAIL: Template creation failed"
    exit 1
fi

# Clean up test data
curl -s -X DELETE "$API_BASE/texts/$TEXT_ID"

echo "Post-migration validation completed successfully!"
```

**2. Performance Baseline Comparison**:
```bash
#!/bin/bash
# performance_comparison.sh

echo "Comparing pre/post migration performance..."

# Load baseline metrics
BASELINE_FILE="/backup/performance_baseline.json"
if [ ! -f "$BASELINE_FILE" ]; then
    echo "WARNING: No baseline file found"
    exit 1
fi

# Get current metrics
CURRENT_METRICS=$(curl -s "$API_BASE/metrics")

# Compare key metrics
BASELINE_RESPONSE_TIME=$(jq -r '.histograms.http_request_duration.p95' "$BASELINE_FILE")
CURRENT_RESPONSE_TIME=$(echo "$CURRENT_METRICS" | jq -r '.histograms.http_request_duration.p95')

# Check for significant performance degradation (>20%)
if (( $(echo "$CURRENT_RESPONSE_TIME > $BASELINE_RESPONSE_TIME * 1.2" | bc -l) )); then
    echo "WARNING: Response time degraded from ${BASELINE_RESPONSE_TIME}ms to ${CURRENT_RESPONSE_TIME}ms"
fi

echo "Performance comparison completed!"
```

## Performance Optimization After Migration

### Index Optimization

**1. Analyze and Optimize Indexes**:
```sql
-- Analyze table statistics after migration
ANALYZE chunks_unified;
ANALYZE embeddings_v2;

-- Check index usage
SELECT
    schemaname, tablename, indexname, idx_scan, idx_tup_read, idx_tup_fetch
FROM pg_stat_user_indexes
WHERE tablename = 'chunks_unified'
ORDER BY idx_scan DESC;

-- Create additional indexes based on usage patterns
CREATE INDEX CONCURRENTLY idx_chunks_unified_content_search
    ON chunks_unified USING gin(to_tsvector('english', content));

CREATE INDEX CONCURRENTLY idx_chunks_unified_metadata_tags
    ON chunks_unified USING gin((metadata->'tags'));
```

**2. Cache Warming**:
```bash
#!/bin/bash
# cache_warming.sh

echo "Warming cache with frequently accessed data..."

# Pre-load popular searches
curl -s -X POST "$API_BASE/search/semantic" \
    -H "Content-Type: application/json" \
    -d '{"query": "common search term 1", "limit": 10}' > /dev/null

curl -s -X POST "$API_BASE/search/semantic" \
    -H "Content-Type: application/json" \
    -d '{"query": "common search term 2", "limit": 10}' > /dev/null

# Pre-load templates
curl -s "$API_BASE/templates" > /dev/null

# Check cache warming effectiveness
CACHE_STATS=$(curl -s "$API_BASE/cache/stats")
CACHE_SIZE=$(echo "$CACHE_STATS" | jq '.size')

echo "Cache warmed with $CACHE_SIZE items"
```

## Troubleshooting Migration Issues

### Common Migration Problems

**1. Memory Issues During Migration**:
```bash
# Symptoms: Out of memory errors, slow migration
# Solution: Increase memory limits and process in batches

# Batch processing approach
export MIGRATION_BATCH_SIZE=1000
export MIGRATION_MEMORY_LIMIT=4GB

# Monitor memory usage during migration
watch -n 5 'free -h; echo "---"; ps aux | grep semantic-text-processor | head -5'
```

**2. Database Connection Timeouts**:
```bash
# Symptoms: Connection timeout errors during migration
# Solution: Increase connection timeouts and pool size

export DB_MIGRATION_TIMEOUT=300s
export DB_MIGRATION_MAX_CONNECTIONS=50

# Test connection stability
for i in {1..10}; do
    curl -s "$SUPABASE_URL/rest/v1/" \
        -H "apikey: $SUPABASE_API_KEY" \
        -H "Content-Type: application/json"
    sleep 1
done
```

**3. Data Corruption Detection**:
```sql
-- Check for data corruption after migration
SELECT
    COUNT(*) as total_chunks,
    COUNT(DISTINCT text_id) as unique_texts,
    COUNT(CASE WHEN parent_chunk_id IS NOT NULL THEN 1 END) as child_chunks,
    COUNT(CASE WHEN is_template = TRUE THEN 1 END) as template_chunks,
    COUNT(CASE WHEN is_slot = TRUE THEN 1 END) as slot_chunks
FROM chunks_unified;

-- Verify JSON metadata integrity
SELECT COUNT(*) as invalid_metadata
FROM chunks_unified
WHERE NOT (metadata::text)::json IS NOT NULL;

-- Check for circular references in hierarchy
WITH RECURSIVE chunk_hierarchy AS (
    SELECT id, parent_chunk_id, 1 as level
    FROM chunks_unified
    WHERE parent_chunk_id IS NULL

    UNION ALL

    SELECT c.id, c.parent_chunk_id, ch.level + 1
    FROM chunks_unified c
    JOIN chunk_hierarchy ch ON c.parent_chunk_id = ch.id
    WHERE ch.level < 20  -- Prevent infinite recursion
)
SELECT COUNT(*) as potential_circular_refs
FROM chunk_hierarchy
WHERE level > 15;  -- Flag very deep hierarchies
```

### Migration Recovery Procedures

**1. Partial Migration Recovery**:
```bash
#!/bin/bash
# partial_recovery.sh

echo "Starting partial migration recovery..."

# Identify incomplete migrations
INCOMPLETE_CHUNKS=$(curl -s "$API_BASE/admin/migration/status" | jq '.incomplete_chunks')

if [ "$INCOMPLETE_CHUNKS" -gt 0 ]; then
    echo "Found $INCOMPLETE_CHUNKS incomplete chunk migrations"

    # Resume migration from checkpoint
    curl -X POST "$API_BASE/admin/migration/resume" \
        -H "Content-Type: application/json" \
        -d '{"checkpoint": "chunks_migration", "batch_size": 500}'
fi

# Verify recovery
sleep 30
RECOVERY_STATUS=$(curl -s "$API_BASE/admin/migration/status" | jq -r '.status')

if [ "$RECOVERY_STATUS" = "completed" ]; then
    echo "Recovery completed successfully!"
else
    echo "Recovery failed, manual intervention required"
    exit 1
fi
```

**2. Emergency Data Export**:
```bash
#!/bin/bash
# emergency_export.sh

echo "Starting emergency data export..."

# Export all texts and chunks
mkdir -p /emergency_backup/$(date +%Y%m%d_%H%M%S)
EXPORT_DIR="/emergency_backup/$(date +%Y%m%d_%H%M%S)"

# Export texts
curl -s "$API_BASE/texts?page_size=100" | jq '.texts' > "$EXPORT_DIR/texts.json"

# Export chunks
curl -s "$API_BASE/chunks?page_size=100" | jq '.chunks' > "$EXPORT_DIR/chunks.json"

# Export templates
curl -s "$API_BASE/templates" | jq '.' > "$EXPORT_DIR/templates.json"

# Create checksums
cd "$EXPORT_DIR"
sha256sum *.json > checksums.txt

echo "Emergency export completed in $EXPORT_DIR"
```

### Post-Migration Monitoring

**1. Extended Monitoring Period**:
```bash
#!/bin/bash
# post_migration_monitoring.sh

echo "Starting extended post-migration monitoring..."

# Monitor for 24 hours with detailed logging
for i in {1..1440}; do  # 24 hours * 60 minutes
    TIMESTAMP=$(date '+%Y-%m-%d %H:%M:%S')

    # Health check
    HEALTH=$(curl -s http://localhost:8080/api/v1/health | jq -r '.status')

    # Performance metrics
    METRICS=$(curl -s http://localhost:8080/api/v1/metrics)
    RESPONSE_TIME=$(echo "$METRICS" | jq -r '.histograms.http_request_duration.p95')
    MEMORY_USAGE=$(echo "$METRICS" | jq -r '.gauges.memory_usage_bytes')

    # Cache performance
    CACHE_STATS=$(curl -s http://localhost:8080/api/v1/cache/stats)
    HIT_RATE=$(echo "$CACHE_STATS" | jq -r '.hit_rate')

    # Log metrics
    echo "$TIMESTAMP,$HEALTH,$RESPONSE_TIME,$MEMORY_USAGE,$HIT_RATE" >> /var/log/post_migration_monitoring.csv

    # Alert on issues
    if [ "$HEALTH" != "healthy" ] || [ "$(echo "$RESPONSE_TIME > 1000" | bc)" -eq 1 ]; then
        echo "ALERT: System health degraded at $TIMESTAMP"
        # Send alert notification here
    fi

    sleep 60  # Monitor every minute
done

echo "Extended monitoring completed!"
```

---

This migration guide provides comprehensive procedures for safely upgrading and migrating the Semantic Text Processor system. Always test migrations in a staging environment before applying to production, and ensure you have verified backup and rollback procedures before beginning any migration.