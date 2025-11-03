package services

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"semantic-text-processor/models"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "github.com/lib/pq"
)

// Integration tests for UnifiedChunkService tag operations
// These tests require a real PostgreSQL database with the unified schema

func setupIntegrationDB(t *testing.T) *sql.DB {
	// Skip if not running integration tests
	if os.Getenv("RUN_INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping integration test - set RUN_INTEGRATION_TESTS=true to run")
	}

	// Database connection parameters
	host := os.Getenv("DB_HOST")
	if host == "" {
		host = "localhost"
	}
	
	port := os.Getenv("DB_PORT")
	if port == "" {
		port = "5432"
	}
	
	dbname := os.Getenv("DB_NAME")
	if dbname == "" {
		dbname = "semantic_processor_test"
	}
	
	user := os.Getenv("DB_USER")
	if user == "" {
		user = "postgres"
	}
	
	password := os.Getenv("DB_PASSWORD")
	if password == "" {
		password = "postgres"
	}

	// Connection string
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		t.Fatalf("Failed to ping database: %v", err)
	}

	return db
}

func TestUnifiedChunkService_TagOperations_RealDatabase(t *testing.T) {
	db := setupIntegrationDB(t)
	defer db.Close()

	// Create services
	cache := NewInMemoryCache(100, 5*time.Minute)
	monitor := NewInMemoryPerformanceMonitor(100*time.Millisecond, 10)
	service := NewUnifiedChunkService(db, cache, monitor)

	ctx := context.Background()

	// Create test tags
	tag1 := &models.UnifiedChunkRecord{
		ChunkID:  uuid.New().String(),
		Contents: "Technology",
		IsTag:    true,
	}

	tag2 := &models.UnifiedChunkRecord{
		ChunkID:  uuid.New().String(),
		Contents: "Programming",
		IsTag:    true,
	}

	tag3 := &models.UnifiedChunkRecord{
		ChunkID:  uuid.New().String(),
		Contents: "Database",
		IsTag:    true,
	}

	// Create test chunks
	chunk1 := &models.UnifiedChunkRecord{
		ChunkID:  uuid.New().String(),
		Contents: "Go programming tutorial",
		Tags:     []string{},
	}

	chunk2 := &models.UnifiedChunkRecord{
		ChunkID:  uuid.New().String(),
		Contents: "Database design patterns",
		Tags:     []string{},
	}

	chunk3 := &models.UnifiedChunkRecord{
		ChunkID:  uuid.New().String(),
		Contents: "Advanced Go techniques",
		Tags:     []string{},
	}

	// Clean up function
	cleanup := func() {
		service.DeleteChunk(ctx, tag1.ChunkID)
		service.DeleteChunk(ctx, tag2.ChunkID)
		service.DeleteChunk(ctx, tag3.ChunkID)
		service.DeleteChunk(ctx, chunk1.ChunkID)
		service.DeleteChunk(ctx, chunk2.ChunkID)
		service.DeleteChunk(ctx, chunk3.ChunkID)
	}
	defer cleanup()

	// Create all chunks and tags
	require.NoError(t, service.CreateChunk(ctx, tag1))
	require.NoError(t, service.CreateChunk(ctx, tag2))
	require.NoError(t, service.CreateChunk(ctx, tag3))
	require.NoError(t, service.CreateChunk(ctx, chunk1))
	require.NoError(t, service.CreateChunk(ctx, chunk2))
	require.NoError(t, service.CreateChunk(ctx, chunk3))

	t.Run("AddTags", func(t *testing.T) {
		// Add tags to chunks
		err := service.AddTags(ctx, chunk1.ChunkID, []string{tag1.ChunkID, tag2.ChunkID})
		require.NoError(t, err)

		err = service.AddTags(ctx, chunk2.ChunkID, []string{tag1.ChunkID, tag3.ChunkID})
		require.NoError(t, err)

		err = service.AddTags(ctx, chunk3.ChunkID, []string{tag2.ChunkID})
		require.NoError(t, err)

		// Verify tags were added by checking the main table
		retrievedChunk1, err := service.GetChunk(ctx, chunk1.ChunkID)
		require.NoError(t, err)
		assert.Len(t, retrievedChunk1.Tags, 2)
		assert.Contains(t, retrievedChunk1.Tags, tag1.ChunkID)
		assert.Contains(t, retrievedChunk1.Tags, tag2.ChunkID)
	})

	t.Run("GetChunkTags", func(t *testing.T) {
		// Get tags for chunk1
		tags, err := service.GetChunkTags(ctx, chunk1.ChunkID)
		require.NoError(t, err)
		assert.Len(t, tags, 2)

		// Verify tag contents
		tagContents := make([]string, len(tags))
		for i, tag := range tags {
			tagContents[i] = tag.Contents
		}
		assert.Contains(t, tagContents, "Technology")
		assert.Contains(t, tagContents, "Programming")

		// Get tags for chunk2
		tags, err = service.GetChunkTags(ctx, chunk2.ChunkID)
		require.NoError(t, err)
		assert.Len(t, tags, 2)

		// Get tags for chunk3
		tags, err = service.GetChunkTags(ctx, chunk3.ChunkID)
		require.NoError(t, err)
		assert.Len(t, tags, 1)
		assert.Equal(t, "Programming", tags[0].Contents)
	})

	t.Run("GetChunksByTag", func(t *testing.T) {
		// Get chunks with Technology tag
		chunks, err := service.GetChunksByTag(ctx, tag1.ChunkID)
		require.NoError(t, err)
		assert.Len(t, chunks, 2) // chunk1 and chunk2

		chunkContents := make([]string, len(chunks))
		for i, chunk := range chunks {
			chunkContents[i] = chunk.Contents
		}
		assert.Contains(t, chunkContents, "Go programming tutorial")
		assert.Contains(t, chunkContents, "Database design patterns")

		// Get chunks with Programming tag
		chunks, err = service.GetChunksByTag(ctx, tag2.ChunkID)
		require.NoError(t, err)
		assert.Len(t, chunks, 2) // chunk1 and chunk3

		// Get chunks with Database tag
		chunks, err = service.GetChunksByTag(ctx, tag3.ChunkID)
		require.NoError(t, err)
		assert.Len(t, chunks, 1) // only chunk2
		assert.Equal(t, "Database design patterns", chunks[0].Contents)
	})

	t.Run("GetChunksByTags_AND", func(t *testing.T) {
		// Get chunks with both Technology AND Programming tags
		chunks, err := service.GetChunksByTags(ctx, []string{tag1.ChunkID, tag2.ChunkID}, "AND")
		require.NoError(t, err)
		assert.Len(t, chunks, 1) // only chunk1 has both tags
		assert.Equal(t, "Go programming tutorial", chunks[0].Contents)

		// Get chunks with both Technology AND Database tags
		chunks, err = service.GetChunksByTags(ctx, []string{tag1.ChunkID, tag3.ChunkID}, "AND")
		require.NoError(t, err)
		assert.Len(t, chunks, 1) // only chunk2 has both tags
		assert.Equal(t, "Database design patterns", chunks[0].Contents)

		// Get chunks with Programming AND Database tags (should be none)
		chunks, err = service.GetChunksByTags(ctx, []string{tag2.ChunkID, tag3.ChunkID}, "AND")
		require.NoError(t, err)
		assert.Len(t, chunks, 0) // no chunks have both tags
	})

	t.Run("GetChunksByTags_OR", func(t *testing.T) {
		// Get chunks with Technology OR Programming tags
		chunks, err := service.GetChunksByTags(ctx, []string{tag1.ChunkID, tag2.ChunkID}, "OR")
		require.NoError(t, err)
		assert.Len(t, chunks, 3) // all chunks have at least one of these tags

		// Get chunks with Programming OR Database tags
		chunks, err = service.GetChunksByTags(ctx, []string{tag2.ChunkID, tag3.ChunkID}, "OR")
		require.NoError(t, err)
		assert.Len(t, chunks, 3) // chunk1 and chunk3 have Programming, chunk2 has Database
	})

	t.Run("RemoveTags", func(t *testing.T) {
		// Remove Programming tag from chunk1
		err := service.RemoveTags(ctx, chunk1.ChunkID, []string{tag2.ChunkID})
		require.NoError(t, err)

		// Verify tag was removed
		tags, err := service.GetChunkTags(ctx, chunk1.ChunkID)
		require.NoError(t, err)
		assert.Len(t, tags, 1) // should only have Technology tag now
		assert.Equal(t, "Technology", tags[0].Contents)

		// Verify chunks by tag queries are updated
		chunks, err := service.GetChunksByTag(ctx, tag2.ChunkID)
		require.NoError(t, err)
		assert.Len(t, chunks, 1) // only chunk3 should have Programming tag now
		assert.Equal(t, "Advanced Go techniques", chunks[0].Contents)

		// Test AND query after removal
		chunks, err = service.GetChunksByTags(ctx, []string{tag1.ChunkID, tag2.ChunkID}, "AND")
		require.NoError(t, err)
		assert.Len(t, chunks, 0) // no chunks have both tags now
	})

	t.Run("CacheInvalidation", func(t *testing.T) {
		// First query should hit database and cache result
		chunks1, err := service.GetChunksByTag(ctx, tag1.ChunkID)
		require.NoError(t, err)

		// Second query should hit cache
		chunks2, err := service.GetChunksByTag(ctx, tag1.ChunkID)
		require.NoError(t, err)
		assert.Equal(t, chunks1, chunks2)

		// Add a tag to invalidate cache
		err = service.AddTags(ctx, chunk3.ChunkID, []string{tag1.ChunkID})
		require.NoError(t, err)

		// Query should return updated results
		chunks3, err := service.GetChunksByTag(ctx, tag1.ChunkID)
		require.NoError(t, err)
		assert.Len(t, chunks3, len(chunks1)+1) // should have one more chunk now
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		// Test adding non-existent tag
		nonExistentTagID := uuid.New().String()
		err := service.AddTags(ctx, chunk1.ChunkID, []string{nonExistentTagID})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "tag chunk not found")

		// Test adding tag to non-existent chunk
		nonExistentChunkID := uuid.New().String()
		err = service.AddTags(ctx, nonExistentChunkID, []string{tag1.ChunkID})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "chunk not found")

		// Test getting tags for non-existent chunk
		_, err = service.GetChunkTags(ctx, nonExistentChunkID)
		assert.NoError(t, err) // Should return empty slice, not error

		// Test getting chunks by non-existent tag
		_, err = service.GetChunksByTag(ctx, nonExistentTagID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "tag chunk not found")

		// Test adding non-tag chunk as tag
		regularChunk := &models.UnifiedChunkRecord{
			ChunkID:  uuid.New().String(),
			Contents: "Regular chunk",
			IsTag:    false,
		}
		require.NoError(t, service.CreateChunk(ctx, regularChunk))
		defer service.DeleteChunk(ctx, regularChunk.ChunkID)

		err = service.AddTags(ctx, chunk1.ChunkID, []string{regularChunk.ChunkID})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "is not a tag")
	})
}

func TestUnifiedChunkService_TagOperations_Performance(t *testing.T) {
	db := setupIntegrationDB(t)
	defer db.Close()

	cache := NewInMemoryCache(1000, 10*time.Minute)
	monitor := NewInMemoryPerformanceMonitor(50*time.Millisecond, 100)
	service := NewUnifiedChunkService(db, cache, monitor)

	ctx := context.Background()

	// Create multiple tags and chunks for performance testing
	numTags := 10
	numChunks := 50

	tags := make([]*models.UnifiedChunkRecord, numTags)
	chunks := make([]*models.UnifiedChunkRecord, numChunks)

	// Clean up function
	cleanup := func() {
		for _, tag := range tags {
			if tag != nil {
				service.DeleteChunk(ctx, tag.ChunkID)
			}
		}
		for _, chunk := range chunks {
			if chunk != nil {
				service.DeleteChunk(ctx, chunk.ChunkID)
			}
		}
	}
	defer cleanup()

	// Create tags
	for i := 0; i < numTags; i++ {
		tags[i] = &models.UnifiedChunkRecord{
			ChunkID:  uuid.New().String(),
			Contents: fmt.Sprintf("Tag_%d", i),
			IsTag:    true,
		}
		require.NoError(t, service.CreateChunk(ctx, tags[i]))
	}

	// Create chunks
	for i := 0; i < numChunks; i++ {
		chunks[i] = &models.UnifiedChunkRecord{
			ChunkID:  uuid.New().String(),
			Contents: fmt.Sprintf("Chunk content %d", i),
		}
		require.NoError(t, service.CreateChunk(ctx, chunks[i]))
	}

	t.Run("BulkTagAssignment", func(t *testing.T) {
		start := time.Now()

		// Assign random tags to chunks
		for i, chunk := range chunks {
			// Assign 2-3 random tags to each chunk
			numTagsToAssign := 2 + (i % 2)
			tagsToAssign := make([]string, numTagsToAssign)
			for j := 0; j < numTagsToAssign; j++ {
				tagsToAssign[j] = tags[(i+j)%numTags].ChunkID
			}

			err := service.AddTags(ctx, chunk.ChunkID, tagsToAssign)
			require.NoError(t, err)
		}

		duration := time.Since(start)
		t.Logf("Bulk tag assignment took: %v", duration)

		// Verify performance is reasonable (should be under 5 seconds for 50 chunks)
		assert.Less(t, duration, 5*time.Second, "Bulk tag assignment took too long")
	})

	t.Run("QueryPerformance", func(t *testing.T) {
		// Test single tag queries
		start := time.Now()
		for _, tag := range tags {
			chunks, err := service.GetChunksByTag(ctx, tag.ChunkID)
			require.NoError(t, err)
			assert.Greater(t, len(chunks), 0, "Each tag should have at least one chunk")
		}
		singleTagDuration := time.Since(start)
		t.Logf("Single tag queries took: %v", singleTagDuration)

		// Test multi-tag AND queries
		start = time.Now()
		for i := 0; i < 10; i++ {
			tag1 := tags[i%numTags]
			tag2 := tags[(i+1)%numTags]
			chunks, err := service.GetChunksByTags(ctx, []string{tag1.ChunkID, tag2.ChunkID}, "AND")
			require.NoError(t, err)
			_ = chunks // Results may vary
		}
		multiTagANDDuration := time.Since(start)
		t.Logf("Multi-tag AND queries took: %v", multiTagANDDuration)

		// Test multi-tag OR queries
		start = time.Now()
		for i := 0; i < 10; i++ {
			tag1 := tags[i%numTags]
			tag2 := tags[(i+1)%numTags]
			chunks, err := service.GetChunksByTags(ctx, []string{tag1.ChunkID, tag2.ChunkID}, "OR")
			require.NoError(t, err)
			assert.Greater(t, len(chunks), 0, "OR query should return results")
		}
		multiTagORDuration := time.Since(start)
		t.Logf("Multi-tag OR queries took: %v", multiTagORDuration)

		// All queries should be reasonably fast
		assert.Less(t, singleTagDuration, 2*time.Second, "Single tag queries took too long")
		assert.Less(t, multiTagANDDuration, 2*time.Second, "Multi-tag AND queries took too long")
		assert.Less(t, multiTagORDuration, 2*time.Second, "Multi-tag OR queries took too long")
	})

	t.Run("CacheEffectiveness", func(t *testing.T) {
		// Clear cache first
		cache.Clear(ctx)

		// First query (cache miss)
		start := time.Now()
		chunks1, err := service.GetChunksByTag(ctx, tags[0].ChunkID)
		require.NoError(t, err)
		firstQueryDuration := time.Since(start)

		// Second query (cache hit)
		start = time.Now()
		chunks2, err := service.GetChunksByTag(ctx, tags[0].ChunkID)
		require.NoError(t, err)
		secondQueryDuration := time.Since(start)

		// Results should be identical
		assert.Equal(t, chunks1, chunks2)

		// Cache hit should be significantly faster
		assert.Less(t, secondQueryDuration, firstQueryDuration/2, "Cache hit should be much faster")
		t.Logf("First query (cache miss): %v", firstQueryDuration)
		t.Logf("Second query (cache hit): %v", secondQueryDuration)

		// Check cache stats
		stats := cache.GetStats()
		assert.Greater(t, stats.Hits, int64(0), "Should have cache hits")
		t.Logf("Cache stats: %+v", stats)
	})

	// Check performance monitor stats
	stats := monitor.GetQueryStats()
	t.Logf("Performance stats: %+v", stats)
	assert.Greater(t, stats.TotalQueries, int64(0), "Should have recorded queries")
}

// Integration tests for UnifiedChunkService hierarchy operations
func TestUnifiedChunkService_HierarchyOperations_RealDatabase(t *testing.T) {
	db := setupIntegrationDB(t)
	defer db.Close()

	// Create services
	cache := NewInMemoryCache(100, 5*time.Minute)
	monitor := NewInMemoryPerformanceMonitor(100*time.Millisecond, 10)
	service := NewUnifiedChunkService(db, cache, monitor)

	ctx := context.Background()

	// Create test hierarchy:
	// Root
	// ├── Child1
	// │   ├── Grandchild1
	// │   └── Grandchild2
	// └── Child2
	//     └── Grandchild3

	root := &models.UnifiedChunkRecord{
		ChunkID:  uuid.New().String(),
		Contents: "Root",
		Parent:   nil,
	}

	child1 := &models.UnifiedChunkRecord{
		ChunkID:  uuid.New().String(),
		Contents: "Child 1",
		Parent:   &root.ChunkID,
	}

	child2 := &models.UnifiedChunkRecord{
		ChunkID:  uuid.New().String(),
		Contents: "Child 2",
		Parent:   &root.ChunkID,
	}

	grandchild1 := &models.UnifiedChunkRecord{
		ChunkID:  uuid.New().String(),
		Contents: "Grandchild 1",
		Parent:   &child1.ChunkID,
	}

	grandchild2 := &models.UnifiedChunkRecord{
		ChunkID:  uuid.New().String(),
		Contents: "Grandchild 2",
		Parent:   &child1.ChunkID,
	}

	grandchild3 := &models.UnifiedChunkRecord{
		ChunkID:  uuid.New().String(),
		Contents: "Grandchild 3",
		Parent:   &child2.ChunkID,
	}

	allChunks := []*models.UnifiedChunkRecord{root, child1, child2, grandchild1, grandchild2, grandchild3}

	// Clean up function
	cleanup := func() {
		for _, chunk := range allChunks {
			service.DeleteChunk(ctx, chunk.ChunkID)
		}
	}
	defer cleanup()

	// Create all chunks
	for _, chunk := range allChunks {
		require.NoError(t, service.CreateChunk(ctx, chunk))
	}

	t.Run("GetChildren", func(t *testing.T) {
		// Test root children
		rootChildren, err := service.GetChildren(ctx, root.ChunkID)
		require.NoError(t, err)
		assert.Len(t, rootChildren, 2)

		childContents := make([]string, len(rootChildren))
		for i, child := range rootChildren {
			childContents[i] = child.Contents
		}
		assert.Contains(t, childContents, "Child 1")
		assert.Contains(t, childContents, "Child 2")

		// Test child1 children
		child1Children, err := service.GetChildren(ctx, child1.ChunkID)
		require.NoError(t, err)
		assert.Len(t, child1Children, 2)

		grandchildContents := make([]string, len(child1Children))
		for i, grandchild := range child1Children {
			grandchildContents[i] = grandchild.Contents
		}
		assert.Contains(t, grandchildContents, "Grandchild 1")
		assert.Contains(t, grandchildContents, "Grandchild 2")

		// Test child2 children
		child2Children, err := service.GetChildren(ctx, child2.ChunkID)
		require.NoError(t, err)
		assert.Len(t, child2Children, 1)
		assert.Equal(t, "Grandchild 3", child2Children[0].Contents)

		// Test leaf node (no children)
		grandchild1Children, err := service.GetChildren(ctx, grandchild1.ChunkID)
		require.NoError(t, err)
		assert.Len(t, grandchild1Children, 0)
	})

	t.Run("GetDescendants", func(t *testing.T) {
		// Test all descendants of root (no depth limit)
		allDescendants, err := service.GetDescendants(ctx, root.ChunkID, 0)
		require.NoError(t, err)
		assert.Len(t, allDescendants, 5) // All descendants

		// Verify hierarchy metadata is included
		for _, descendant := range allDescendants {
			assert.NotNil(t, descendant.Metadata)
			assert.Contains(t, descendant.Metadata, "hierarchy_depth")
			assert.Contains(t, descendant.Metadata, "hierarchy_path")

			depth := descendant.Metadata["hierarchy_depth"].(int)
			assert.Greater(t, depth, 0)
			assert.LessOrEqual(t, depth, 2)

			pathIDs := descendant.Metadata["hierarchy_path"].([]string)
			assert.Greater(t, len(pathIDs), 1)
			assert.Equal(t, root.ChunkID, pathIDs[0]) // First element should be root
		}

		// Test descendants with depth limit
		directAndGrandchildren, err := service.GetDescendants(ctx, root.ChunkID, 2)
		require.NoError(t, err)
		assert.Len(t, directAndGrandchildren, 5) // All are within depth 2

		onlyDirectChildren, err := service.GetDescendants(ctx, root.ChunkID, 1)
		require.NoError(t, err)
		assert.Len(t, onlyDirectChildren, 2) // Only direct children

		// Test descendants of child1
		child1Descendants, err := service.GetDescendants(ctx, child1.ChunkID, 0)
		require.NoError(t, err)
		assert.Len(t, child1Descendants, 2) // grandchild1 and grandchild2

		// Test leaf node (no descendants)
		grandchild1Descendants, err := service.GetDescendants(ctx, grandchild1.ChunkID, 0)
		require.NoError(t, err)
		assert.Len(t, grandchild1Descendants, 0)
	})

	t.Run("GetAncestors", func(t *testing.T) {
		// Test ancestors of grandchild1
		grandchild1Ancestors, err := service.GetAncestors(ctx, grandchild1.ChunkID)
		require.NoError(t, err)
		assert.Len(t, grandchild1Ancestors, 2) // child1 and root

		// Verify order (should be from immediate parent to root)
		assert.Equal(t, child1.ChunkID, grandchild1Ancestors[0].ChunkID) // Immediate parent
		assert.Equal(t, root.ChunkID, grandchild1Ancestors[1].ChunkID)   // Root

		// Verify hierarchy metadata
		for i, ancestor := range grandchild1Ancestors {
			assert.NotNil(t, ancestor.Metadata)
			assert.Contains(t, ancestor.Metadata, "hierarchy_depth")
			
			depth := ancestor.Metadata["hierarchy_depth"].(int)
			assert.Equal(t, i+1, depth) // Depth should increase with distance
		}

		// Test ancestors of child1
		child1Ancestors, err := service.GetAncestors(ctx, child1.ChunkID)
		require.NoError(t, err)
		assert.Len(t, child1Ancestors, 1) // only root
		assert.Equal(t, root.ChunkID, child1Ancestors[0].ChunkID)

		// Test ancestors of root (should be empty)
		rootAncestors, err := service.GetAncestors(ctx, root.ChunkID)
		require.NoError(t, err)
		assert.Len(t, rootAncestors, 0)
	})

	t.Run("MoveChunk", func(t *testing.T) {
		// Move grandchild1 from child1 to child2
		err := service.MoveChunk(ctx, grandchild1.ChunkID, child2.ChunkID)
		require.NoError(t, err)

		// Verify the move
		child1ChildrenAfterMove, err := service.GetChildren(ctx, child1.ChunkID)
		require.NoError(t, err)
		assert.Len(t, child1ChildrenAfterMove, 1) // Should only have grandchild2 now
		assert.Equal(t, "Grandchild 2", child1ChildrenAfterMove[0].Contents)

		child2ChildrenAfterMove, err := service.GetChildren(ctx, child2.ChunkID)
		require.NoError(t, err)
		assert.Len(t, child2ChildrenAfterMove, 2) // Should have grandchild3 and grandchild1 now

		childContents := make([]string, len(child2ChildrenAfterMove))
		for i, child := range child2ChildrenAfterMove {
			childContents[i] = child.Contents
		}
		assert.Contains(t, childContents, "Grandchild 1")
		assert.Contains(t, childContents, "Grandchild 3")

		// Verify ancestors changed for moved chunk
		grandchild1AncestorsAfterMove, err := service.GetAncestors(ctx, grandchild1.ChunkID)
		require.NoError(t, err)
		assert.Len(t, grandchild1AncestorsAfterMove, 2) // child2 and root
		assert.Equal(t, child2.ChunkID, grandchild1AncestorsAfterMove[0].ChunkID)
		assert.Equal(t, root.ChunkID, grandchild1AncestorsAfterMove[1].ChunkID)

		// Verify the chunk's parent field was updated
		retrievedGrandchild1, err := service.GetChunk(ctx, grandchild1.ChunkID)
		require.NoError(t, err)
		assert.NotNil(t, retrievedGrandchild1.Parent)
		assert.Equal(t, child2.ChunkID, *retrievedGrandchild1.Parent)
	})

	t.Run("MoveChunkToRoot", func(t *testing.T) {
		// Move child1 to root (no parent)
		err := service.MoveChunk(ctx, child1.ChunkID, "")
		require.NoError(t, err)

		// Verify move to root
		rootChildrenAfterMove, err := service.GetChildren(ctx, root.ChunkID)
		require.NoError(t, err)
		assert.Len(t, rootChildrenAfterMove, 1) // Should only have child2 now
		assert.Equal(t, "Child 2", rootChildrenAfterMove[0].Contents)

		child1AncestorsAfterMove, err := service.GetAncestors(ctx, child1.ChunkID)
		require.NoError(t, err)
		assert.Len(t, child1AncestorsAfterMove, 0) // child1 is now at root level

		// Verify the chunk's parent field was updated to null
		retrievedChild1, err := service.GetChunk(ctx, child1.ChunkID)
		require.NoError(t, err)
		assert.Nil(t, retrievedChild1.Parent)

		// Verify grandchild2 still has child1 as parent and can find ancestors
		grandchild2Ancestors, err := service.GetAncestors(ctx, grandchild2.ChunkID)
		require.NoError(t, err)
		assert.Len(t, grandchild2Ancestors, 1) // only child1 now
		assert.Equal(t, child1.ChunkID, grandchild2Ancestors[0].ChunkID)
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		// Test circular reference prevention
		err := service.MoveChunk(ctx, root.ChunkID, child2.ChunkID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "circular reference")

		// Test moving to non-existent parent
		nonExistentID := uuid.New().String()
		err = service.MoveChunk(ctx, child2.ChunkID, nonExistentID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")

		// Test moving non-existent chunk
		err = service.MoveChunk(ctx, nonExistentID, child2.ChunkID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")

		// Test getting children of non-existent chunk
		_, err = service.GetChildren(ctx, nonExistentID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")

		// Test getting descendants of non-existent chunk
		_, err = service.GetDescendants(ctx, nonExistentID, 0)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")

		// Test getting ancestors of non-existent chunk
		_, err = service.GetAncestors(ctx, nonExistentID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("CacheInvalidation", func(t *testing.T) {
		// Clear cache first
		cache.Clear(ctx)

		// First query should hit database and cache result
		children1, err := service.GetChildren(ctx, root.ChunkID)
		require.NoError(t, err)

		// Second query should hit cache
		children2, err := service.GetChildren(ctx, root.ChunkID)
		require.NoError(t, err)
		assert.Equal(t, children1, children2)

		// Create a new child to invalidate cache
		newChild := &models.UnifiedChunkRecord{
			ChunkID:  uuid.New().String(),
			Contents: "New Child",
			Parent:   &root.ChunkID,
		}
		require.NoError(t, service.CreateChunk(ctx, newChild))
		defer service.DeleteChunk(ctx, newChild.ChunkID)

		// Query should return updated results
		children3, err := service.GetChildren(ctx, root.ChunkID)
		require.NoError(t, err)
		assert.Len(t, children3, len(children1)+1) // should have one more child now
	})
}

func TestUnifiedChunkService_HierarchyOperations_Performance(t *testing.T) {
	db := setupIntegrationDB(t)
	defer db.Close()

	cache := NewInMemoryCache(1000, 10*time.Minute)
	monitor := NewInMemoryPerformanceMonitor(50*time.Millisecond, 100)
	service := NewUnifiedChunkService(db, cache, monitor)

	ctx := context.Background()

	// Create a deeper hierarchy for performance testing
	// Root -> Level1 (10 nodes) -> Level2 (50 nodes) -> Level3 (100 nodes)
	
	root := &models.UnifiedChunkRecord{
		ChunkID:  uuid.New().String(),
		Contents: "Performance Test Root",
		Parent:   nil,
	}

	var allChunks []*models.UnifiedChunkRecord
	allChunks = append(allChunks, root)

	// Clean up function
	cleanup := func() {
		for _, chunk := range allChunks {
			if chunk != nil {
				service.DeleteChunk(ctx, chunk.ChunkID)
			}
		}
	}
	defer cleanup()

	// Create root
	require.NoError(t, service.CreateChunk(ctx, root))

	// Create level 1 nodes (10 nodes)
	level1Nodes := make([]*models.UnifiedChunkRecord, 10)
	for i := 0; i < 10; i++ {
		level1Nodes[i] = &models.UnifiedChunkRecord{
			ChunkID:  uuid.New().String(),
			Contents: fmt.Sprintf("Level1_Node_%d", i),
			Parent:   &root.ChunkID,
		}
		require.NoError(t, service.CreateChunk(ctx, level1Nodes[i]))
		allChunks = append(allChunks, level1Nodes[i])
	}

	// Create level 2 nodes (5 nodes per level 1 node = 50 total)
	level2Nodes := make([]*models.UnifiedChunkRecord, 50)
	for i := 0; i < 50; i++ {
		parentIndex := i / 5 // 5 children per parent
		level2Nodes[i] = &models.UnifiedChunkRecord{
			ChunkID:  uuid.New().String(),
			Contents: fmt.Sprintf("Level2_Node_%d", i),
			Parent:   &level1Nodes[parentIndex].ChunkID,
		}
		require.NoError(t, service.CreateChunk(ctx, level2Nodes[i]))
		allChunks = append(allChunks, level2Nodes[i])
	}

	// Create level 3 nodes (2 nodes per level 2 node = 100 total)
	level3Nodes := make([]*models.UnifiedChunkRecord, 100)
	for i := 0; i < 100; i++ {
		parentIndex := i / 2 // 2 children per parent
		level3Nodes[i] = &models.UnifiedChunkRecord{
			ChunkID:  uuid.New().String(),
			Contents: fmt.Sprintf("Level3_Node_%d", i),
			Parent:   &level2Nodes[parentIndex].ChunkID,
		}
		require.NoError(t, service.CreateChunk(ctx, level3Nodes[i]))
		allChunks = append(allChunks, level3Nodes[i])
	}

	t.Logf("Created hierarchy with %d total nodes", len(allChunks))

	t.Run("GetChildrenPerformance", func(t *testing.T) {
		start := time.Now()

		// Query children for all level 1 and level 2 nodes
		for _, node := range level1Nodes {
			children, err := service.GetChildren(ctx, node.ChunkID)
			require.NoError(t, err)
			assert.Len(t, children, 5) // Each level 1 node should have 5 children
		}

		for _, node := range level2Nodes {
			children, err := service.GetChildren(ctx, node.ChunkID)
			require.NoError(t, err)
			assert.Len(t, children, 2) // Each level 2 node should have 2 children
		}

		duration := time.Since(start)
		t.Logf("GetChildren queries took: %v", duration)
		assert.Less(t, duration, 5*time.Second, "GetChildren queries took too long")
	})

	t.Run("GetDescendantsPerformance", func(t *testing.T) {
		start := time.Now()

		// Query all descendants of root
		allDescendants, err := service.GetDescendants(ctx, root.ChunkID, 0)
		require.NoError(t, err)
		assert.Len(t, allDescendants, 160) // 10 + 50 + 100

		duration := time.Since(start)
		t.Logf("GetDescendants (all) took: %v", duration)
		assert.Less(t, duration, 2*time.Second, "GetDescendants query took too long")

		// Query descendants with depth limits
		start = time.Now()
		level1Descendants, err := service.GetDescendants(ctx, root.ChunkID, 1)
		require.NoError(t, err)
		assert.Len(t, level1Descendants, 10) // Only level 1 nodes

		level2Descendants, err := service.GetDescendants(ctx, root.ChunkID, 2)
		require.NoError(t, err)
		assert.Len(t, level2Descendants, 60) // Level 1 + Level 2 nodes

		duration = time.Since(start)
		t.Logf("GetDescendants (with depth limits) took: %v", duration)
		assert.Less(t, duration, 1*time.Second, "Depth-limited queries took too long")
	})

	t.Run("GetAncestorsPerformance", func(t *testing.T) {
		start := time.Now()

		// Query ancestors for all level 3 nodes
		for _, node := range level3Nodes {
			ancestors, err := service.GetAncestors(ctx, node.ChunkID)
			require.NoError(t, err)
			assert.Len(t, ancestors, 3) // Level 2, Level 1, Root
		}

		duration := time.Since(start)
		t.Logf("GetAncestors queries took: %v", duration)
		assert.Less(t, duration, 3*time.Second, "GetAncestors queries took too long")
	})

	t.Run("MoveChunkPerformance", func(t *testing.T) {
		start := time.Now()

		// Move some level 3 nodes to different parents
		for i := 0; i < 10; i++ {
			sourceNode := level3Nodes[i]
			targetParent := level2Nodes[(i+10)%50] // Move to a different level 2 parent

			err := service.MoveChunk(ctx, sourceNode.ChunkID, targetParent.ChunkID)
			require.NoError(t, err)
		}

		duration := time.Since(start)
		t.Logf("MoveChunk operations took: %v", duration)
		assert.Less(t, duration, 5*time.Second, "MoveChunk operations took too long")

		// Verify moves were successful by checking one of them
		movedNode := level3Nodes[0]
		ancestors, err := service.GetAncestors(ctx, movedNode.ChunkID)
		require.NoError(t, err)
		assert.Len(t, ancestors, 3) // Should still have 3 ancestors
		
		// The immediate parent should be the new parent
		newParent := level2Nodes[10]
		assert.Equal(t, newParent.ChunkID, ancestors[0].ChunkID)
	})

	t.Run("CacheEffectiveness", func(t *testing.T) {
		// Clear cache first
		cache.Clear(ctx)

		// First query (cache miss)
		start := time.Now()
		children1, err := service.GetChildren(ctx, root.ChunkID)
		require.NoError(t, err)
		firstQueryDuration := time.Since(start)

		// Second query (cache hit)
		start = time.Now()
		children2, err := service.GetChildren(ctx, root.ChunkID)
		require.NoError(t, err)
		secondQueryDuration := time.Since(start)

		// Results should be identical
		assert.Equal(t, children1, children2)

		// Cache hit should be significantly faster
		assert.Less(t, secondQueryDuration, firstQueryDuration/2, "Cache hit should be much faster")
		t.Logf("First query (cache miss): %v", firstQueryDuration)
		t.Logf("Second query (cache hit): %v", secondQueryDuration)

		// Check cache stats
		stats := cache.GetStats()
		assert.Greater(t, stats.Hits, int64(0), "Should have cache hits")
		t.Logf("Cache stats: %+v", stats)
	})

	// Check performance monitor stats
	stats := monitor.GetQueryStats()
	t.Logf("Performance stats: %+v", stats)
	assert.Greater(t, stats.TotalQueries, int64(0), "Should have recorded queries")
}