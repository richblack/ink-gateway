package services

import (
	"context"
	"database/sql"
	"fmt"
	"semantic-text-processor/models"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "github.com/lib/pq"
)

// Integration tests for search cache functionality
// These tests require a test database setup

func setupSearchCacheIntegrationTest(t *testing.T) (*sql.DB, SearchCacheService, UnifiedChunkService, func()) {
	// This would typically connect to a test PostgreSQL database
	// For this example, we'll use SQLite with adapted schema
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	
	// Create simplified tables for testing
	createTablesSQL := `
		CREATE TABLE chunks (
			chunk_id TEXT PRIMARY KEY,
			contents TEXT NOT NULL,
			parent TEXT,
			page TEXT,
			is_page BOOLEAN DEFAULT FALSE,
			is_tag BOOLEAN DEFAULT FALSE,
			is_template BOOLEAN DEFAULT FALSE,
			is_slot BOOLEAN DEFAULT FALSE,
			ref TEXT,
			tags TEXT, -- JSON array as text for SQLite
			metadata TEXT, -- JSON as text for SQLite
			created_time DATETIME DEFAULT CURRENT_TIMESTAMP,
			last_updated DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		
		CREATE TABLE chunk_tags (
			source_chunk_id TEXT NOT NULL,
			tag_chunk_id TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (source_chunk_id, tag_chunk_id)
		);
		
		CREATE TABLE chunk_hierarchy (
			ancestor_id TEXT NOT NULL,
			descendant_id TEXT NOT NULL,
			depth INTEGER NOT NULL,
			path_ids TEXT NOT NULL, -- JSON array as text
			PRIMARY KEY (ancestor_id, descendant_id)
		);
		
		CREATE TABLE chunk_search_cache (
			search_hash TEXT PRIMARY KEY,
			query_params TEXT NOT NULL,
			chunk_ids TEXT NOT NULL,
			result_count INTEGER NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			expires_at DATETIME NOT NULL,
			hit_count INTEGER DEFAULT 0
		);
	`
	
	_, err = db.Exec(createTablesSQL)
	require.NoError(t, err)
	
	// Create services
	monitor := &MockQueryPerformanceMonitor{}
	cache := NewInMemoryCache(1000, 5*time.Minute)
	
	baseService := NewUnifiedChunkService(db, cache, monitor)
	searchCache := NewDatabaseSearchCache(db, DefaultSearchCacheConfig(), monitor)
	
	enhancedService := NewSearchCacheEnhancedUnifiedChunkService(
		baseService,
		searchCache,
		db,
		monitor,
	)
	
	cleanup := func() {
		cache.Stop()
		db.Close()
	}
	
	return db, searchCache, enhancedService, cleanup
}

func TestSearchCacheIntegration_BasicSearchCaching(t *testing.T) {
	db, _, service, cleanup := setupSearchCacheIntegrationTest(t)
	defer cleanup()
	
	ctx := context.Background()
	
	// Create test chunks
	testChunks := []models.UnifiedChunkRecord{
		{
			ChunkID:  "chunk1",
			Contents: "This is a test document about artificial intelligence",
			IsPage:   false,
			Tags:     []string{"ai", "tech"},
			Metadata: map[string]interface{}{"category": "technology"},
		},
		{
			ChunkID:  "chunk2",
			Contents: "Another document discussing machine learning algorithms",
			IsPage:   false,
			Tags:     []string{"ml", "tech"},
			Metadata: map[string]interface{}{"category": "technology"},
		},
		{
			ChunkID:  "chunk3",
			Contents: "A page about cooking recipes and food preparation",
			IsPage:   true,
			Tags:     []string{"cooking", "food"},
			Metadata: map[string]interface{}{"category": "lifestyle"},
		},
	}
	
	// Insert test data directly into database (simplified for testing)
	for _, chunk := range testChunks {
		_, err := db.ExecContext(ctx, `
			INSERT INTO chunks (chunk_id, contents, is_page, tags, metadata)
			VALUES (?, ?, ?, ?, ?)
		`, chunk.ChunkID, chunk.Contents, chunk.IsPage, `["tag1"]`, `{"category":"test"}`)
		require.NoError(t, err)
	}
	
	// Test search query
	query := &models.SearchQuery{
		Content: "artificial intelligence",
		Limit:   10,
	}
	
	// First search - should miss cache and populate it
	result1, err := service.SearchChunks(ctx, query)
	require.NoError(t, err)
	assert.NotNil(t, result1)
	assert.False(t, result1.CacheHit)
	
	// Wait a moment for async cache population
	time.Sleep(100 * time.Millisecond)
	
	// Second search - should hit cache
	result2, err := service.SearchChunks(ctx, query)
	require.NoError(t, err)
	assert.NotNil(t, result2)
	assert.True(t, result2.CacheHit)
	
	// Results should be consistent
	assert.Equal(t, len(result1.Chunks), len(result2.Chunks))
	assert.True(t, result2.SearchTime < result1.SearchTime) // Cache should be faster
}

func TestSearchCacheIntegration_CacheInvalidation(t *testing.T) {
	db, _, service, cleanup := setupSearchCacheIntegrationTest(t)
	defer cleanup()
	
	ctx := context.Background()
	
	// Insert test chunk
	_, err := db.ExecContext(ctx, `
		INSERT INTO chunks (chunk_id, contents, is_page)
		VALUES ('test-chunk', 'test content', false)
	`)
	require.NoError(t, err)
	
	// Perform search to populate cache
	query := &models.SearchQuery{
		Content: "test content",
		Limit:   10,
	}
	
	result1, err := service.SearchChunks(ctx, query)
	require.NoError(t, err)
	assert.False(t, result1.CacheHit)
	
	// Wait for cache population
	time.Sleep(100 * time.Millisecond)
	
	// Verify cache hit
	result2, err := service.SearchChunks(ctx, query)
	require.NoError(t, err)
	assert.True(t, result2.CacheHit)
	
	// Create new chunk (should invalidate cache)
	newChunk := &models.UnifiedChunkRecord{
		ChunkID:  "new-chunk",
		Contents: "new test content",
		IsPage:   false,
	}
	
	err = service.CreateChunk(ctx, newChunk)
	require.NoError(t, err)
	
	// Wait for cache invalidation
	time.Sleep(100 * time.Millisecond)
	
	// Next search should miss cache due to invalidation
	result3, err := service.SearchChunks(ctx, query)
	require.NoError(t, err)
	assert.False(t, result3.CacheHit)
}

func TestSearchCacheIntegration_CacheStats(t *testing.T) {
	_, searchCache, service, cleanup := setupSearchCacheIntegrationTest(t)
	defer cleanup()
	
	ctx := context.Background()
	
	// Get initial stats
	stats1, err := searchCache.GetCacheStats(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, stats1.TotalEntries)
	
	// Perform some searches to populate cache
	queries := []*models.SearchQuery{
		{Content: "test1", Limit: 10},
		{Content: "test2", Limit: 10},
		{Content: "test3", Limit: 10},
	}
	
	for _, query := range queries {
		_, err := service.SearchChunks(ctx, query)
		require.NoError(t, err)
	}
	
	// Wait for cache population
	time.Sleep(200 * time.Millisecond)
	
	// Get updated stats
	stats2, err := searchCache.GetCacheStats(ctx)
	require.NoError(t, err)
	assert.True(t, stats2.TotalEntries > 0)
	assert.True(t, stats2.CacheSize > 0)
}

func TestSearchCacheIntegration_OptimizationSuggestions(t *testing.T) {
	_, searchCache, _, cleanup := setupSearchCacheIntegrationTest(t)
	defer cleanup()
	
	ctx := context.Background()
	
	// Create cache entries with different characteristics to trigger suggestions
	
	// Add expired entries
	queryParams := map[string]interface{}{"content": "expired_test"}
	chunkIDs := []string{"chunk1", "chunk2"}
	err := searchCache.SetCachedSearch(ctx, queryParams, chunkIDs, -1*time.Hour)
	require.NoError(t, err)
	
	// Add more expired entries to trigger expiration suggestion
	for i := 0; i < 5; i++ {
		params := map[string]interface{}{"content": fmt.Sprintf("expired_%d", i)}
		err := searchCache.SetCachedSearch(ctx, params, chunkIDs, -1*time.Hour)
		require.NoError(t, err)
	}
	
	// Add some valid entries
	for i := 0; i < 2; i++ {
		params := map[string]interface{}{"content": fmt.Sprintf("valid_%d", i)}
		err := searchCache.SetCachedSearch(ctx, params, chunkIDs, 1*time.Hour)
		require.NoError(t, err)
	}
	
	// Get optimization suggestions
	suggestions, err := searchCache.GetOptimizationSuggestions(ctx)
	require.NoError(t, err)
	assert.NotNil(t, suggestions)
	
	// Should have suggestions due to high expired entry ratio
	assert.True(t, len(suggestions) > 0)
	
	// Check for specific suggestion types
	suggestionTypes := make(map[string]bool)
	for _, suggestion := range suggestions {
		suggestionTypes[suggestion.Type] = true
		assert.NotEmpty(t, suggestion.Description)
		assert.NotEmpty(t, suggestion.Action)
		assert.NotEmpty(t, suggestion.Impact)
		assert.Contains(t, []string{"high", "medium", "low"}, suggestion.Priority)
	}
	
	// Should have expiration suggestion due to many expired entries
	assert.True(t, suggestionTypes["expiration"] || suggestionTypes["hit_rate"])
}

func TestSearchCacheIntegration_CleanupExpiredEntries(t *testing.T) {
	_, searchCache, _, cleanup := setupSearchCacheIntegrationTest(t)
	defer cleanup()
	
	ctx := context.Background()
	
	// Add expired entries
	expiredParams := []map[string]interface{}{
		{"content": "expired1"},
		{"content": "expired2"},
		{"content": "expired3"},
	}
	
	chunkIDs := []string{"chunk1"}
	for _, params := range expiredParams {
		err := searchCache.SetCachedSearch(ctx, params, chunkIDs, -1*time.Hour)
		require.NoError(t, err)
	}
	
	// Add valid entries
	validParams := []map[string]interface{}{
		{"content": "valid1"},
		{"content": "valid2"},
	}
	
	for _, params := range validParams {
		err := searchCache.SetCachedSearch(ctx, params, chunkIDs, 1*time.Hour)
		require.NoError(t, err)
	}
	
	// Cleanup expired entries
	deletedCount, err := searchCache.CleanupExpiredEntries(ctx)
	require.NoError(t, err)
	assert.Equal(t, 3, deletedCount)
	
	// Verify stats reflect cleanup
	stats, err := searchCache.GetCacheStats(ctx)
	require.NoError(t, err)
	assert.Equal(t, 2, stats.TotalEntries) // Only valid entries should remain
	assert.Equal(t, 0, stats.ExpiredEntries)
}

func TestSearchCacheIntegration_ConcurrentSearches(t *testing.T) {
	db, _, service, cleanup := setupSearchCacheIntegrationTest(t)
	defer cleanup()
	
	ctx := context.Background()
	
	// Insert test data
	_, err := db.ExecContext(ctx, `
		INSERT INTO chunks (chunk_id, contents, is_page)
		VALUES ('concurrent-test', 'concurrent search test content', false)
	`)
	require.NoError(t, err)
	
	// Perform concurrent searches
	numGoroutines := 10
	results := make(chan *models.SearchResult, numGoroutines)
	errors := make(chan error, numGoroutines)
	
	query := &models.SearchQuery{
		Content: "concurrent search test",
		Limit:   10,
	}
	
	// Start concurrent searches
	for i := 0; i < numGoroutines; i++ {
		go func() {
			result, err := service.SearchChunks(ctx, query)
			if err != nil {
				errors <- err
				return
			}
			results <- result
		}()
	}
	
	// Collect results
	var searchResults []*models.SearchResult
	for i := 0; i < numGoroutines; i++ {
		select {
		case result := <-results:
			searchResults = append(searchResults, result)
		case err := <-errors:
			t.Fatalf("Concurrent search failed: %v", err)
		case <-time.After(5 * time.Second):
			t.Fatal("Concurrent search timed out")
		}
	}
	
	// Verify all searches completed successfully
	assert.Equal(t, numGoroutines, len(searchResults))
	
	// At least some should be cache hits (after the first one)
	cacheHits := 0
	for _, result := range searchResults {
		if result.CacheHit {
			cacheHits++
		}
	}
	assert.True(t, cacheHits > 0, "Expected some cache hits in concurrent searches")
}

func TestSearchCacheIntegration_DifferentQueryTypes(t *testing.T) {
	db, _, service, cleanup := setupSearchCacheIntegrationTest(t)
	defer cleanup()
	
	ctx := context.Background()
	
	// Insert test data with different types
	testData := []struct {
		chunkID  string
		contents string
		isPage   bool
		isTag    bool
	}{
		{"page1", "This is a page about technology", true, false},
		{"tag1", "technology", false, true},
		{"content1", "Regular content about technology", false, false},
	}
	
	for _, data := range testData {
		_, err := db.ExecContext(ctx, `
			INSERT INTO chunks (chunk_id, contents, is_page, is_tag)
			VALUES (?, ?, ?, ?)
		`, data.chunkID, data.contents, data.isPage, data.isTag)
		require.NoError(t, err)
	}
	
	// Test different query types
	queries := []*models.SearchQuery{
		{Content: "technology", Limit: 10},                    // Content search
		{IsPage: &[]bool{true}[0], Limit: 10},                // Type filter
		{IsTag: &[]bool{true}[0], Limit: 10},                 // Tag filter
		{Content: "technology", IsPage: &[]bool{false}[0]},   // Combined filter
	}
	
	for i, query := range queries {
		t.Run(fmt.Sprintf("QueryType_%d", i), func(t *testing.T) {
			// First search - cache miss
			result1, err := service.SearchChunks(ctx, query)
			require.NoError(t, err)
			assert.False(t, result1.CacheHit)
			
			// Wait for cache population
			time.Sleep(50 * time.Millisecond)
			
			// Second search - cache hit
			result2, err := service.SearchChunks(ctx, query)
			require.NoError(t, err)
			assert.True(t, result2.CacheHit)
			
			// Results should be consistent
			assert.Equal(t, len(result1.Chunks), len(result2.Chunks))
		})
	}
}

func TestSearchCacheIntegration_TTLVariation(t *testing.T) {
	_, searchCache, service, cleanup := setupSearchCacheIntegrationTest(t)
	defer cleanup()
	
	ctx := context.Background()
	
	// Test queries with different expected TTLs
	queries := []*models.SearchQuery{
		{Content: "test content"},                    // Content search - shorter TTL
		{Tags: []string{"tag1", "tag2"}},            // Tag search - longer TTL
		{IsPage: &[]bool{true}[0]},                  // Type search - longest TTL
	}
	
	for i, query := range queries {
		t.Run(fmt.Sprintf("TTL_Test_%d", i), func(t *testing.T) {
			// Perform search to populate cache
			result, err := service.SearchChunks(ctx, query)
			require.NoError(t, err)
			assert.False(t, result.CacheHit)
			
			// Wait for cache population
			time.Sleep(100 * time.Millisecond)
			
			// Verify cache entry exists
			queryParams := map[string]interface{}{}
			if query.Content != "" {
				queryParams["content"] = query.Content
			}
			if len(query.Tags) > 0 {
				queryParams["tags"] = query.Tags
			}
			if query.IsPage != nil {
				queryParams["is_page"] = *query.IsPage
			}
			
			cacheEntry, err := searchCache.GetCachedSearch(ctx, queryParams)
			require.NoError(t, err)
			assert.NotNil(t, cacheEntry, "Cache entry should exist")
			
			if cacheEntry != nil {
				assert.True(t, cacheEntry.ExpiresAt.After(time.Now()), "Cache entry should not be expired")
			}
		})
	}
}