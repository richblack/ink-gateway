-- Schema Validation Script for Unified Chunk System
-- This script validates that the schema was created correctly and all constraints work

-- ============================================================================
-- VALIDATION TESTS
-- ============================================================================

DO $$
DECLARE
    test_chunk_id UUID;
    test_tag_id UUID;
    test_parent_id UUID;
    test_count INTEGER;
    validation_errors TEXT[] := ARRAY[]::TEXT[];
BEGIN
    RAISE NOTICE 'Starting Unified Chunk System schema validation...';
    
    -- Test 1: Check if all required tables exist
    RAISE NOTICE 'Test 1: Checking table existence...';
    
    SELECT COUNT(*) INTO test_count
    FROM information_schema.tables 
    WHERE table_name IN ('chunks', 'chunk_tags', 'chunk_hierarchy', 'chunk_search_cache');
    
    IF test_count != 4 THEN
        validation_errors := array_append(validation_errors, 'Missing required tables');
    END IF;
    
    -- Test 2: Check if materialized view exists
    RAISE NOTICE 'Test 2: Checking materialized view...';
    
    SELECT COUNT(*) INTO test_count
    FROM information_schema.views 
    WHERE table_name = 'tag_statistics';
    
    IF test_count != 1 THEN
        validation_errors := array_append(validation_errors, 'Missing tag_statistics materialized view');
    END IF;
    
    -- Test 3: Check if triggers exist
    RAISE NOTICE 'Test 3: Checking triggers...';
    
    SELECT COUNT(*) INTO test_count
    FROM information_schema.triggers 
    WHERE trigger_name LIKE 'trigger_%chunk%';
    
    IF test_count < 3 THEN
        validation_errors := array_append(validation_errors, 'Missing required triggers');
    END IF;
    
    -- Test 4: Test basic chunk insertion
    RAISE NOTICE 'Test 4: Testing basic chunk insertion...';
    
    BEGIN
        INSERT INTO chunks (contents, is_page) 
        VALUES ('Test page content', true) 
        RETURNING chunk_id INTO test_chunk_id;
        
        -- Verify the chunk was inserted
        SELECT COUNT(*) INTO test_count
        FROM chunks WHERE chunk_id = test_chunk_id;
        
        IF test_count != 1 THEN
            validation_errors := array_append(validation_errors, 'Basic chunk insertion failed');
        END IF;
        
    EXCEPTION WHEN OTHERS THEN
        validation_errors := array_append(validation_errors, 'Basic chunk insertion error: ' || SQLERRM);
    END;
    
    -- Test 5: Test tag creation and relationship
    RAISE NOTICE 'Test 5: Testing tag relationships...';
    
    BEGIN
        -- Create a tag chunk
        INSERT INTO chunks (contents, is_tag) 
        VALUES ('test-tag', true) 
        RETURNING chunk_id INTO test_tag_id;
        
        -- Create a content chunk with the tag
        INSERT INTO chunks (contents, tags) 
        VALUES ('Content with tag', jsonb_build_array(test_tag_id::text))
        RETURNING chunk_id INTO test_parent_id;
        
        -- Check if the tag relationship was created in auxiliary table
        SELECT COUNT(*) INTO test_count
        FROM chunk_tags 
        WHERE source_chunk_id = test_parent_id AND tag_chunk_id = test_tag_id;
        
        IF test_count != 1 THEN
            validation_errors := array_append(validation_errors, 'Tag relationship trigger failed');
        END IF;
        
    EXCEPTION WHEN OTHERS THEN
        validation_errors := array_append(validation_errors, 'Tag relationship test error: ' || SQLERRM);
    END;
    
    -- Test 6: Test hierarchy relationships
    RAISE NOTICE 'Test 6: Testing hierarchy relationships...';
    
    BEGIN
        -- Create a child chunk
        INSERT INTO chunks (contents, parent) 
        VALUES ('Child content', test_parent_id);
        
        -- Check if hierarchy relationship was created
        SELECT COUNT(*) INTO test_count
        FROM chunk_hierarchy 
        WHERE ancestor_id = test_parent_id AND depth = 1;
        
        IF test_count < 1 THEN
            validation_errors := array_append(validation_errors, 'Hierarchy relationship trigger failed');
        END IF;
        
    EXCEPTION WHEN OTHERS THEN
        validation_errors := array_append(validation_errors, 'Hierarchy relationship test error: ' || SQLERRM);
    END;
    
    -- Test 7: Test constraints
    RAISE NOTICE 'Test 7: Testing constraints...';
    
    BEGIN
        -- Try to insert chunk with empty content (should fail)
        INSERT INTO chunks (contents) VALUES ('');
        validation_errors := array_append(validation_errors, 'Empty content constraint not working');
    EXCEPTION WHEN check_violation THEN
        -- This is expected
        NULL;
    EXCEPTION WHEN OTHERS THEN
        validation_errors := array_append(validation_errors, 'Unexpected constraint error: ' || SQLERRM);
    END;
    
    -- Test 8: Test indexes exist
    RAISE NOTICE 'Test 8: Checking indexes...';
    
    SELECT COUNT(*) INTO test_count
    FROM pg_indexes 
    WHERE tablename IN ('chunks', 'chunk_tags', 'chunk_hierarchy', 'chunk_search_cache');
    
    IF test_count < 15 THEN
        validation_errors := array_append(validation_errors, 'Missing required indexes (found ' || test_count || ', expected at least 15)');
    END IF;
    
    -- Test 9: Test utility functions
    RAISE NOTICE 'Test 9: Testing utility functions...';
    
    BEGIN
        -- Test cleanup function
        PERFORM cleanup_expired_search_cache();
        
        -- Test path function
        PERFORM get_chunk_path(test_chunk_id);
        
    EXCEPTION WHEN OTHERS THEN
        validation_errors := array_append(validation_errors, 'Utility function error: ' || SQLERRM);
    END;
    
    -- Clean up test data
    RAISE NOTICE 'Cleaning up test data...';
    DELETE FROM chunks WHERE contents LIKE 'Test%' OR contents LIKE 'Content%' OR contents = 'test-tag' OR contents = 'Child content';
    
    -- Report results
    IF array_length(validation_errors, 1) IS NULL THEN
        RAISE NOTICE '✅ All validation tests passed successfully!';
        RAISE NOTICE 'Schema is ready for use.';
    ELSE
        RAISE NOTICE '❌ Validation failed with the following errors:';
        FOR i IN 1..array_length(validation_errors, 1) LOOP
            RAISE NOTICE '  - %', validation_errors[i];
        END LOOP;
        RAISE EXCEPTION 'Schema validation failed';
    END IF;
    
END $$;

-- ============================================================================
-- PERFORMANCE VALIDATION
-- ============================================================================

-- Test query performance on empty tables
EXPLAIN (ANALYZE, BUFFERS) 
SELECT c.* FROM chunks c
JOIN chunk_tags ct ON c.chunk_id = ct.source_chunk_id
WHERE ct.tag_chunk_id = gen_random_uuid()
LIMIT 10;

-- Test hierarchy query performance
EXPLAIN (ANALYZE, BUFFERS)
SELECT c.*, ch.depth FROM chunks c
JOIN chunk_hierarchy ch ON c.chunk_id = ch.descendant_id
WHERE ch.ancestor_id = gen_random_uuid() AND ch.depth <= 3
ORDER BY ch.depth;

-- Test full-text search performance
EXPLAIN (ANALYZE, BUFFERS)
SELECT * FROM chunks 
WHERE to_tsvector('english', contents) @@ to_tsquery('english', 'test')
LIMIT 10;

-- ============================================================================
-- SCHEMA INFORMATION SUMMARY
-- ============================================================================

SELECT 'Schema validation completed. Summary:' as status;

SELECT 
    'Tables' as object_type,
    COUNT(*) as count
FROM information_schema.tables 
WHERE table_name IN ('chunks', 'chunk_tags', 'chunk_hierarchy', 'chunk_search_cache')

UNION ALL

SELECT 
    'Indexes' as object_type,
    COUNT(*) as count
FROM pg_indexes 
WHERE tablename IN ('chunks', 'chunk_tags', 'chunk_hierarchy', 'chunk_search_cache')

UNION ALL

SELECT 
    'Triggers' as object_type,
    COUNT(*) as count
FROM information_schema.triggers 
WHERE trigger_name LIKE 'trigger_%chunk%'

UNION ALL

SELECT 
    'Functions' as object_type,
    COUNT(*) as count
FROM information_schema.routines 
WHERE routine_name IN ('sync_chunk_tags', 'sync_chunk_hierarchy', 'cleanup_expired_search_cache', 'get_chunk_path')

ORDER BY object_type;