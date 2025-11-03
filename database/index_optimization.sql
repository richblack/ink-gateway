-- Index Optimization Script for Unified Chunk System
-- This script contains additional indexes that can be created based on usage patterns
-- Run this after the main schema is created and you have production data

-- ============================================================================
-- PERFORMANCE ANALYSIS QUERIES
-- ============================================================================

-- Query to analyze table sizes
CREATE OR REPLACE VIEW table_sizes AS
SELECT 
    schemaname,
    tablename,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) as size,
    pg_total_relation_size(schemaname||'.'||tablename) as size_bytes
FROM pg_tables 
WHERE tablename IN ('chunks', 'chunk_tags', 'chunk_hierarchy', 'chunk_search_cache')
ORDER BY size_bytes DESC;

-- Query to analyze index usage
CREATE OR REPLACE VIEW index_usage AS
SELECT 
    schemaname,
    tablename,
    indexname,
    idx_scan as times_used,
    pg_size_pretty(pg_relation_size(indexname::regclass)) as index_size,
    idx_tup_read as tuples_read,
    idx_tup_fetch as tuples_fetched
FROM pg_stat_user_indexes 
WHERE schemaname = current_schema()
ORDER BY times_used DESC;

-- ============================================================================
-- CONDITIONAL INDEXES (Create based on usage patterns)
-- ============================================================================

-- Index for frequently accessed recent content
-- CREATE INDEX CONCURRENTLY idx_chunks_recent_active 
--     ON chunks(last_updated DESC, chunk_id) 
--     WHERE last_updated > NOW() - INTERVAL '7 days';

-- Index for content length-based queries
-- CREATE INDEX CONCURRENTLY idx_chunks_content_size 
--     ON chunks((length(contents))) 
--     WHERE length(contents) BETWEEN 100 AND 10000;

-- Index for metadata queries (create based on actual metadata keys used)
-- CREATE INDEX CONCURRENTLY idx_chunks_metadata_specific 
--     ON chunks USING gin((metadata->'specific_key'));

-- Partial index for active pages only
-- CREATE INDEX CONCURRENTLY idx_chunks_active_pages 
--     ON chunks(created_time DESC, chunk_id) 
--     WHERE is_page = true AND last_updated > NOW() - INTERVAL '30 days';

-- ============================================================================
-- MAINTENANCE FUNCTIONS
-- ============================================================================

-- Function to analyze and suggest index optimizations
CREATE OR REPLACE FUNCTION analyze_index_performance()
RETURNS TABLE(
    table_name TEXT,
    index_name TEXT,
    usage_score NUMERIC,
    size_mb NUMERIC,
    recommendation TEXT
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        i.tablename::TEXT,
        i.indexname::TEXT,
        CASE 
            WHEN i.idx_scan = 0 THEN 0
            ELSE ROUND((i.idx_scan::NUMERIC / GREATEST(s.seq_scan + s.idx_scan, 1)) * 100, 2)
        END as usage_score,
        ROUND(pg_relation_size(i.indexname::regclass) / 1024.0 / 1024.0, 2) as size_mb,
        CASE 
            WHEN i.idx_scan = 0 AND pg_relation_size(i.indexname::regclass) > 1024*1024 
                THEN 'Consider dropping - unused and large'
            WHEN i.idx_scan < 10 AND pg_relation_size(i.indexname::regclass) > 10*1024*1024 
                THEN 'Low usage for size - review necessity'
            WHEN i.idx_scan > 1000 AND pg_relation_size(i.indexname::regclass) < 1024*1024 
                THEN 'High usage, small size - good index'
            ELSE 'Normal usage'
        END as recommendation
    FROM pg_stat_user_indexes i
    JOIN pg_stat_user_tables s ON i.relid = s.relid
    WHERE i.schemaname = current_schema()
    ORDER BY usage_score DESC;
END;
$$ LANGUAGE plpgsql;

-- Function to rebuild all indexes
CREATE OR REPLACE FUNCTION rebuild_all_indexes()
RETURNS TEXT AS $$
DECLARE
    index_record RECORD;
    result_text TEXT := '';
BEGIN
    FOR index_record IN 
        SELECT indexname 
        FROM pg_indexes 
        WHERE schemaname = current_schema() 
        AND tablename IN ('chunks', 'chunk_tags', 'chunk_hierarchy', 'chunk_search_cache')
    LOOP
        EXECUTE 'REINDEX INDEX CONCURRENTLY ' || index_record.indexname;
        result_text := result_text || 'Rebuilt: ' || index_record.indexname || E'\n';
    END LOOP;
    
    RETURN result_text;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- MONITORING QUERIES
-- ============================================================================

-- Query to monitor slow queries related to chunks
CREATE OR REPLACE VIEW slow_chunk_queries AS
SELECT 
    query,
    calls,
    total_time,
    mean_time,
    rows,
    100.0 * shared_blks_hit / nullif(shared_blks_hit + shared_blks_read, 0) AS hit_percent
FROM pg_stat_statements 
WHERE query ILIKE '%chunks%' 
   OR query ILIKE '%chunk_tags%' 
   OR query ILIKE '%chunk_hierarchy%'
ORDER BY mean_time DESC;

-- Query to monitor table bloat
CREATE OR REPLACE VIEW table_bloat_check AS
SELECT 
    schemaname,
    tablename,
    n_tup_ins as inserts,
    n_tup_upd as updates,
    n_tup_del as deletes,
    n_dead_tup as dead_tuples,
    CASE 
        WHEN n_live_tup > 0 
        THEN ROUND(100.0 * n_dead_tup / (n_live_tup + n_dead_tup), 2)
        ELSE 0 
    END as dead_tuple_percent
FROM pg_stat_user_tables 
WHERE schemaname = current_schema()
ORDER BY dead_tuple_percent DESC;

-- ============================================================================
-- USAGE EXAMPLES
-- ============================================================================

/*
-- To analyze current index performance:
SELECT * FROM analyze_index_performance();

-- To check table sizes:
SELECT * FROM table_sizes;

-- To monitor index usage:
SELECT * FROM index_usage WHERE times_used < 100;

-- To check for table bloat:
SELECT * FROM table_bloat_check WHERE dead_tuple_percent > 10;

-- To rebuild all indexes (use with caution in production):
SELECT rebuild_all_indexes();
*/