-- Unified Chunk System Database Schema
-- This script creates the unified database structure for the Semantic Text Processor
-- All content types (text, tags, templates, slots, pages) are stored in a single chunks table

-- Drop existing tables if they exist (for development/testing)
DROP TABLE IF EXISTS chunk_search_cache CASCADE;
DROP TABLE IF EXISTS chunk_hierarchy CASCADE;
DROP TABLE IF EXISTS chunk_tags CASCADE;
DROP TABLE IF EXISTS chunks CASCADE;

-- Drop materialized views if they exist
DROP MATERIALIZED VIEW IF EXISTS tag_statistics CASCADE;

-- Main chunks table - stores all content types
CREATE TABLE chunks (
    chunk_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    contents TEXT NOT NULL,
    parent UUID REFERENCES chunks(chunk_id) ON DELETE SET NULL,
    page UUID REFERENCES chunks(chunk_id) ON DELETE SET NULL,
    is_page BOOLEAN DEFAULT FALSE,
    is_tag BOOLEAN DEFAULT FALSE,
    is_template BOOLEAN DEFAULT FALSE,
    is_slot BOOLEAN DEFAULT FALSE,
    ref TEXT,
    tags JSONB, -- Array of tag chunk_ids for backup queries
    metadata JSONB, -- Extensible field for future features
    created_time TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_updated TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Auxiliary table for tag relationships (many-to-many optimization)
CREATE TABLE chunk_tags (
    source_chunk_id UUID NOT NULL,
    tag_chunk_id UUID NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    PRIMARY KEY (source_chunk_id, tag_chunk_id),
    FOREIGN KEY (source_chunk_id) REFERENCES chunks(chunk_id) ON DELETE CASCADE,
    FOREIGN KEY (tag_chunk_id) REFERENCES chunks(chunk_id) ON DELETE CASCADE
);

-- Auxiliary table for hierarchy relationships (performance optimization)
CREATE TABLE chunk_hierarchy (
    ancestor_id UUID NOT NULL,
    descendant_id UUID NOT NULL,
    depth INTEGER NOT NULL,
    path_ids UUID[] NOT NULL, -- Complete path from root to descendant
    
    PRIMARY KEY (ancestor_id, descendant_id),
    FOREIGN KEY (ancestor_id) REFERENCES chunks(chunk_id) ON DELETE CASCADE,
    FOREIGN KEY (descendant_id) REFERENCES chunks(chunk_id) ON DELETE CASCADE
);

-- Auxiliary table for search result caching
CREATE TABLE chunk_search_cache (
    search_hash VARCHAR(64) PRIMARY KEY,
    query_params JSONB NOT NULL,
    chunk_ids UUID[] NOT NULL,
    result_count INTEGER NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    hit_count INTEGER DEFAULT 0
);

-- ============================================================================
-- INDEXES FOR PERFORMANCE OPTIMIZATION
-- ============================================================================

-- Main table indexes
CREATE INDEX idx_chunks_is_page ON chunks(is_page) WHERE is_page = true;
CREATE INDEX idx_chunks_is_tag ON chunks(is_tag) WHERE is_tag = true;
CREATE INDEX idx_chunks_is_template ON chunks(is_template) WHERE is_template = true;
CREATE INDEX idx_chunks_is_slot ON chunks(is_slot) WHERE is_slot = true;
CREATE INDEX idx_chunks_parent ON chunks(parent);
CREATE INDEX idx_chunks_page ON chunks(page);
CREATE INDEX idx_chunks_ref ON chunks(ref) WHERE ref IS NOT NULL;

-- Full-text search index
CREATE INDEX idx_chunks_contents_fts ON chunks USING gin(to_tsvector('english', contents));

-- JSONB indexes for tags and metadata
CREATE INDEX idx_chunks_tags_gin ON chunks USING gin(tags);
CREATE INDEX idx_chunks_metadata_gin ON chunks USING gin(metadata);

-- Time-based indexes
CREATE INDEX idx_chunks_created_time ON chunks(created_time DESC);
CREATE INDEX idx_chunks_updated_time ON chunks(last_updated DESC);

-- Composite indexes for common queries
CREATE INDEX idx_chunks_type_created ON chunks(is_page, is_tag, is_template, is_slot, created_time DESC);
CREATE INDEX idx_chunks_page_parent ON chunks(page, parent) WHERE page IS NOT NULL;

-- Partial indexes for active content
CREATE INDEX idx_chunks_active_tags ON chunks(chunk_id) WHERE is_tag = true AND last_updated > NOW() - INTERVAL '30 days';

-- Expression index for content length
CREATE INDEX idx_chunks_content_length ON chunks((length(contents))) WHERE length(contents) > 1000;

-- ============================================================================
-- CHUNK_TAGS TABLE INDEXES
-- ============================================================================

-- Bidirectional query indexes
CREATE INDEX idx_chunk_tags_source ON chunk_tags(source_chunk_id);
CREATE INDEX idx_chunk_tags_tag ON chunk_tags(tag_chunk_id);
CREATE INDEX idx_chunk_tags_created ON chunk_tags(created_at DESC);

-- Composite index for tag statistics
CREATE INDEX idx_chunk_tags_tag_created ON chunk_tags(tag_chunk_id, created_at DESC);

-- ============================================================================
-- CHUNK_HIERARCHY TABLE INDEXES
-- ============================================================================

-- Hierarchy query indexes
CREATE INDEX idx_hierarchy_ancestor ON chunk_hierarchy(ancestor_id, depth);
CREATE INDEX idx_hierarchy_descendant ON chunk_hierarchy(descendant_id, depth);
CREATE INDEX idx_hierarchy_depth ON chunk_hierarchy(depth);
CREATE INDEX idx_hierarchy_path ON chunk_hierarchy USING gin(path_ids);

-- Composite index for common hierarchy queries
CREATE INDEX idx_hierarchy_ancestor_depth_desc ON chunk_hierarchy(ancestor_id, depth DESC, descendant_id);

-- ============================================================================
-- CHUNK_SEARCH_CACHE TABLE INDEXES
-- ============================================================================

CREATE INDEX idx_search_cache_expires ON chunk_search_cache(expires_at);
CREATE INDEX idx_search_cache_params ON chunk_search_cache USING gin(query_params);
CREATE INDEX idx_search_cache_created ON chunk_search_cache(created_at DESC);
CREATE INDEX idx_search_cache_hit_count ON chunk_search_cache(hit_count DESC);-- =========
===================================================================
-- MATERIALIZED VIEWS FOR PERFORMANCE
-- ============================================================================

-- Tag statistics materialized view
CREATE MATERIALIZED VIEW tag_statistics AS
SELECT 
    tag_chunk_id,
    COUNT(*) as usage_count,
    MAX(created_at) as last_used,
    MIN(created_at) as first_used
FROM chunk_tags 
GROUP BY tag_chunk_id;

CREATE UNIQUE INDEX idx_tag_statistics_tag ON tag_statistics(tag_chunk_id);
CREATE INDEX idx_tag_statistics_usage ON tag_statistics(usage_count DESC);
CREATE INDEX idx_tag_statistics_last_used ON tag_statistics(last_used DESC);

-- ============================================================================
-- DATABASE TRIGGERS FOR DATA CONSISTENCY
-- ============================================================================

-- Function to automatically update last_updated timestamp
CREATE OR REPLACE FUNCTION update_last_updated_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.last_updated = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger for automatic timestamp updates
CREATE TRIGGER trigger_chunks_update_timestamp
    BEFORE UPDATE ON chunks
    FOR EACH ROW EXECUTE FUNCTION update_last_updated_column();

-- Function to sync chunk_tags auxiliary table
CREATE OR REPLACE FUNCTION sync_chunk_tags()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' OR TG_OP = 'UPDATE' THEN
        -- Clear old tag relationships
        DELETE FROM chunk_tags WHERE source_chunk_id = NEW.chunk_id;
        
        -- Insert new tag relationships if tags exist
        IF NEW.tags IS NOT NULL AND jsonb_array_length(NEW.tags) > 0 THEN
            INSERT INTO chunk_tags (source_chunk_id, tag_chunk_id)
            SELECT NEW.chunk_id, tag_id::text::uuid
            FROM jsonb_array_elements_text(NEW.tags) AS tag_id
            WHERE tag_id ~ '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$';
        END IF;
        
        RETURN NEW;
    ELSIF TG_OP = 'DELETE' THEN
        -- Clear tag relationships
        DELETE FROM chunk_tags WHERE source_chunk_id = OLD.chunk_id;
        RETURN OLD;
    END IF;
    
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

-- Trigger to maintain chunk_tags consistency
CREATE TRIGGER trigger_sync_chunk_tags
    AFTER INSERT OR UPDATE OF tags OR DELETE ON chunks
    FOR EACH ROW EXECUTE FUNCTION sync_chunk_tags();-
- Function to sync chunk_hierarchy auxiliary table
CREATE OR REPLACE FUNCTION sync_chunk_hierarchy()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' OR TG_OP = 'UPDATE' THEN
        -- Clear old hierarchy relationships for this chunk
        DELETE FROM chunk_hierarchy WHERE descendant_id = NEW.chunk_id;
        
        -- Rebuild hierarchy relationships using recursive CTE
        WITH RECURSIVE hierarchy AS (
            -- Self-reference (depth = 0)
            SELECT NEW.chunk_id as ancestor_id, NEW.chunk_id as descendant_id, 0 as depth, ARRAY[NEW.chunk_id] as path_ids
            
            UNION ALL
            
            -- Recursive part: find all ancestors
            SELECT c.chunk_id as ancestor_id, NEW.chunk_id as descendant_id, h.depth + 1, ARRAY[c.chunk_id] || h.path_ids
            FROM hierarchy h
            JOIN chunks c ON h.ancestor_id = c.parent
            WHERE c.parent IS NOT NULL AND h.depth < 100 -- Prevent infinite recursion
        )
        INSERT INTO chunk_hierarchy (ancestor_id, descendant_id, depth, path_ids)
        SELECT ancestor_id, descendant_id, depth, path_ids FROM hierarchy;
        
        -- Also need to update hierarchy for all descendants if this chunk moved
        IF TG_OP = 'UPDATE' AND (OLD.parent IS DISTINCT FROM NEW.parent) THEN
            -- Rebuild hierarchy for all descendants
            WITH RECURSIVE descendants AS (
                SELECT chunk_id FROM chunks WHERE parent = NEW.chunk_id
                UNION ALL
                SELECT c.chunk_id FROM chunks c
                JOIN descendants d ON c.parent = d.chunk_id
            )
            DELETE FROM chunk_hierarchy WHERE descendant_id IN (SELECT chunk_id FROM descendants);
            
            -- Trigger hierarchy rebuild for descendants (will be handled by their triggers)
            UPDATE chunks SET last_updated = NOW() WHERE chunk_id IN (SELECT chunk_id FROM descendants);
        END IF;
        
        RETURN NEW;
    ELSIF TG_OP = 'DELETE' THEN
        -- Clear all hierarchy relationships involving this chunk
        DELETE FROM chunk_hierarchy WHERE ancestor_id = OLD.chunk_id OR descendant_id = OLD.chunk_id;
        RETURN OLD;
    END IF;
    
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

-- Trigger to maintain chunk_hierarchy consistency
CREATE TRIGGER trigger_sync_chunk_hierarchy
    AFTER INSERT OR UPDATE OF parent OR DELETE ON chunks
    FOR EACH ROW EXECUTE FUNCTION sync_chunk_hierarchy();

-- Function to refresh tag statistics materialized view
CREATE OR REPLACE FUNCTION refresh_tag_statistics()
RETURNS TRIGGER AS $$
BEGIN
    -- Refresh the materialized view when tag relationships change
    REFRESH MATERIALIZED VIEW CONCURRENTLY tag_statistics;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

-- Trigger to refresh tag statistics (with some delay to batch updates)
CREATE TRIGGER trigger_refresh_tag_statistics
    AFTER INSERT OR DELETE ON chunk_tags
    FOR EACH STATEMENT EXECUTE FUNCTION refresh_tag_statistics();

-- ============================================================================
-- UTILITY FUNCTIONS
-- ============================================================================

-- Function to clean expired search cache entries
CREATE OR REPLACE FUNCTION cleanup_expired_search_cache()
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    DELETE FROM chunk_search_cache WHERE expires_at < NOW();
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- Function to get chunk hierarchy path as text
CREATE OR REPLACE FUNCTION get_chunk_path(chunk_uuid UUID)
RETURNS TEXT AS $$
DECLARE
    path_text TEXT;
BEGIN
    SELECT string_agg(c.contents, ' > ' ORDER BY ch.depth DESC)
    INTO path_text
    FROM chunk_hierarchy ch
    JOIN chunks c ON ch.ancestor_id = c.chunk_id
    WHERE ch.descendant_id = chunk_uuid AND ch.depth > 0;
    
    RETURN COALESCE(path_text, '');
END;
$$ LANGUAGE plpgsql;-- =====
=======================================================================
-- INITIAL DATA AND CONSTRAINTS
-- ============================================================================

-- Add check constraints for data integrity
ALTER TABLE chunks ADD CONSTRAINT check_chunk_type 
    CHECK (is_page OR is_tag OR is_template OR is_slot OR 
           (NOT is_page AND NOT is_tag AND NOT is_template AND NOT is_slot));

ALTER TABLE chunks ADD CONSTRAINT check_contents_not_empty 
    CHECK (length(trim(contents)) > 0);

ALTER TABLE chunk_hierarchy ADD CONSTRAINT check_no_self_reference 
    CHECK (ancestor_id != descendant_id OR depth = 0);

ALTER TABLE chunk_hierarchy ADD CONSTRAINT check_positive_depth 
    CHECK (depth >= 0);

ALTER TABLE chunk_search_cache ADD CONSTRAINT check_positive_result_count 
    CHECK (result_count >= 0);

ALTER TABLE chunk_search_cache ADD CONSTRAINT check_expires_after_created 
    CHECK (expires_at > created_at);

-- ============================================================================
-- COMMENTS FOR DOCUMENTATION
-- ============================================================================

COMMENT ON TABLE chunks IS 'Unified table storing all content types: text, tags, templates, slots, and pages';
COMMENT ON COLUMN chunks.chunk_id IS 'Primary key UUID for the chunk';
COMMENT ON COLUMN chunks.contents IS 'The actual content/text of the chunk';
COMMENT ON COLUMN chunks.parent IS 'Reference to parent chunk for hierarchical structure';
COMMENT ON COLUMN chunks.page IS 'Reference to the page this chunk belongs to';
COMMENT ON COLUMN chunks.is_page IS 'Boolean flag indicating if this chunk is a page';
COMMENT ON COLUMN chunks.is_tag IS 'Boolean flag indicating if this chunk is a tag';
COMMENT ON COLUMN chunks.is_template IS 'Boolean flag indicating if this chunk is a template';
COMMENT ON COLUMN chunks.is_slot IS 'Boolean flag indicating if this chunk is a slot';
COMMENT ON COLUMN chunks.ref IS 'Optional reference identifier for external systems';
COMMENT ON COLUMN chunks.tags IS 'JSONB array of tag chunk_ids for backup queries';
COMMENT ON COLUMN chunks.metadata IS 'Extensible JSONB field for future features';

COMMENT ON TABLE chunk_tags IS 'Auxiliary table for optimizing many-to-many tag relationships';
COMMENT ON COLUMN chunk_tags.source_chunk_id IS 'The chunk that has the tag';
COMMENT ON COLUMN chunk_tags.tag_chunk_id IS 'The tag chunk being applied';

COMMENT ON TABLE chunk_hierarchy IS 'Auxiliary table for optimizing hierarchical queries';
COMMENT ON COLUMN chunk_hierarchy.ancestor_id IS 'The ancestor chunk in the hierarchy';
COMMENT ON COLUMN chunk_hierarchy.descendant_id IS 'The descendant chunk in the hierarchy';
COMMENT ON COLUMN chunk_hierarchy.depth IS 'The depth/distance between ancestor and descendant';
COMMENT ON COLUMN chunk_hierarchy.path_ids IS 'Complete path from root to descendant as UUID array';

COMMENT ON TABLE chunk_search_cache IS 'Cache table for storing complex search query results';
COMMENT ON COLUMN chunk_search_cache.search_hash IS 'Hash of the search query for cache key';
COMMENT ON COLUMN chunk_search_cache.query_params IS 'JSONB representation of search parameters';
COMMENT ON COLUMN chunk_search_cache.chunk_ids IS 'Array of chunk UUIDs in the search result';

COMMENT ON MATERIALIZED VIEW tag_statistics IS 'Materialized view providing tag usage statistics';

-- ============================================================================
-- SCHEMA CREATION COMPLETE
-- ============================================================================

-- Display success message
DO $$
BEGIN
    RAISE NOTICE 'Unified Chunk System schema created successfully!';
    RAISE NOTICE 'Tables created: chunks, chunk_tags, chunk_hierarchy, chunk_search_cache';
    RAISE NOTICE 'Materialized view created: tag_statistics';
    RAISE NOTICE 'Triggers and functions created for data consistency';
    RAISE NOTICE 'All indexes and constraints applied';
END $$;