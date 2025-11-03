# SPARC Phase 2: Pseudocode Design

## Migration Approach Overview

```pseudocode
MIGRATION_STRATEGY {
    1. PRE_MIGRATION_VALIDATION
        - Verify old schema exists and is accessible
        - Check data integrity in source tables
        - Validate required permissions and connections
        - Create backup of existing data

    2. SCHEMA_PREPARATION
        - Create unified schema if not exists
        - Prepare migration staging tables
        - Set up monitoring and logging tables

    3. DATA_TRANSFORMATION_PIPELINE
        FOR each source table:
            - Extract data in batches (configurable batch_size)
            - Transform to unified structure
            - Validate transformed data
            - Load to target with conflict resolution
            - Update progress tracking

    4. RELATIONSHIP_MIGRATION
        - Migrate chunk_tags relationships
        - Build chunk_hierarchy from parent references
        - Preserve vector_db and graph_db associations

    5. VALIDATION_AND_VERIFICATION
        - Compare row counts (source vs target)
        - Validate relationship integrity
        - Run data consistency checks
        - Performance benchmarking

    6. ROLLBACK_PREPARATION
        - Create rollback scripts
        - Document rollback procedures
        - Test rollback in staging environment
}
```

## Data Transformation Logic

```pseudocode
TRANSFORM_CHUNK_DATA(old_chunk) {
    unified_chunk = UnifiedChunkRecord{
        chunk_id: old_chunk.id,
        contents: old_chunk.content,
        parent: old_chunk.parent_chunk_id,
        page: DERIVE_PAGE_FROM_TEXT(old_chunk.text_id),
        is_page: old_chunk.text_id != null AND old_chunk.parent_chunk_id == null,
        is_tag: DETECT_TAG_CONTENT(old_chunk.content),
        is_template: old_chunk.is_template,
        is_slot: old_chunk.is_slot,
        ref: old_chunk.text_id,
        tags: CONVERT_TAGS_TO_ARRAY(old_chunk.id),
        metadata: MERGE_METADATA(old_chunk.metadata, old_chunk),
        created_time: old_chunk.created_at,
        last_updated: old_chunk.updated_at
    }
    return unified_chunk
}

DETECT_TAG_CONTENT(content) {
    // Tag detection logic: look for existing chunk_tags relationships
    return EXISTS_IN_CHUNK_TAGS_AS_TAG(content)
}

DERIVE_PAGE_FROM_TEXT(text_id) {
    // Find the root chunk for this text that represents the page
    root_chunk = GET_ROOT_CHUNK_FOR_TEXT(text_id)
    return root_chunk.id if root_chunk exists else null
}
```

## Backward Compatibility Strategy

```pseudocode
API_COMPATIBILITY_LAYER {
    // Adapter pattern to maintain existing API contracts

    LEGACY_INSERT_CHUNK(chunk_record) {
        unified_chunk = TRANSFORM_LEGACY_TO_UNIFIED(chunk_record)
        result = UNIFIED_SERVICE.CreateChunk(unified_chunk)
        return TRANSFORM_UNIFIED_TO_LEGACY(result)
    }

    LEGACY_GET_CHUNK_BY_ID(id) {
        unified_chunk = UNIFIED_SERVICE.GetChunk(id)
        return TRANSFORM_UNIFIED_TO_LEGACY(unified_chunk)
    }

    LEGACY_GET_CHUNKS_BY_TEXT_ID(text_id) {
        search_query = SearchQuery{
            ref: text_id,
            is_page: false  // Exclude page chunks to match old behavior
        }
        unified_chunks = UNIFIED_SERVICE.SearchChunks(search_query)
        return TRANSFORM_ARRAY_UNIFIED_TO_LEGACY(unified_chunks)
    }
}
```

## Progress Monitoring

```pseudocode
MIGRATION_PROGRESS_TRACKING {
    migration_status = {
        migration_id: UUID,
        start_time: timestamp,
        current_phase: enum(validation, schema, data, relationships, verification),
        tables_processed: map[table_name]table_progress,
        total_records: int,
        processed_records: int,
        success_count: int,
        error_count: int,
        estimated_completion: timestamp,
        can_rollback: boolean
    }

    UPDATE_PROGRESS(table, processed_count, total_count) {
        migration_status.tables_processed[table].processed = processed_count
        migration_status.tables_processed[table].total = total_count
        migration_status.processed_records = SUM(all processed counts)
        CALCULATE_ESTIMATED_COMPLETION()
        PERSIST_STATUS_TO_MONITORING_TABLE()
    }
}
```

## Error Handling and Rollback

```pseudocode
ERROR_HANDLING_STRATEGY {
    ON_MIGRATION_ERROR(error, context) {
        LOG_ERROR(error, context)
        INCREMENT_ERROR_COUNT()

        IF error.severity == CRITICAL {
            INITIATE_ROLLBACK()
            RETURN migration_failed
        }

        IF error.is_retryable AND retry_count < MAX_RETRIES {
            WAIT(exponential_backoff)
            RETRY_OPERATION()
        }

        CONTINUE_WITH_NEXT_BATCH()
    }

    ROLLBACK_MIGRATION() {
        1. STOP_CURRENT_OPERATIONS()
        2. RESTORE_FROM_BACKUP()
        3. REVERT_SCHEMA_CHANGES()
        4. VALIDATE_ROLLBACK_SUCCESS()
        5. NOTIFY_ADMINISTRATORS()
    }
}
```