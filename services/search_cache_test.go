package services

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "github.com/lib/pq"
)

// MockQueryPerformanceMonitor for testing
type MockQueryPerformanceMonitor struct {
	queries []QueryRecord
}

type QueryRecord struct {
	QueryType string
	Duration  time.Duration
	RowCount  int
}

func (m *MockQueryPerformanceMonitor) RecordQuery(queryType string, duration time.Duration, rowCount int) {
	m.queries = append(m.queries, QueryRecord{
		QueryType: queryType,
		Duration:  duration,
		RowCount:  rowCount,
	})
}

func (m *MockQueryPerformanceMonitor) GetQueryStats() QueryStatistics {
	return QueryStatistics{}
}

func (m *MockQueryPerformanceMonitor) GetSlowQueries(limit int) []SlowQueryRecord {
	return []SlowQueryRecord{}
}

func (m *MockQueryPerformanceMonitor) RecordSlowQuery(query string, duration time.Duration, params map[string]interface{}) {
	// Mock implementation
}

// setupTestDB creates an in-memory SQLite database for testing
func setupTestSearchCacheDB(t *testing.T) *sql.DB {
	// For testing, we'll use a simple in-memory setup
	// In a real test environment, you'd want to use a test PostgreSQL instance
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	
	// Create the search cache table (simplified for SQLite)
	createTableSQL := `
		CREATE TABLE chunk_search_cache (
			search_hash TEXT PRIMARY KEY,
			query_params TEXT NOT NULL,
			chunk_ids TEXT NOT NULL,
			result_count INTEGER NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			expires_at DATETIME NOT NULL,
			hit_count INTEGER DEFAULT 0
		)
	`
	
	_, err = db.Exec(createTableSQL)
	require.NoError(t, err)
	
	return db
}

func TestDatabaseSearchCache_SetAndGetCachedSearch(t *testing.T) {
	db := setupTestSearchCacheDB(t)
	defer db.Close()
	
	monitor := &MockQueryPerformanceMonitor{}
	config := DefaultSearchCacheConfig()
	config.CleanupInterval = 0 // Disable background cleanup for tests
	
	cache := NewDatabaseSearchCache(db, config, monitor)
	ctx := context.Background()
	
	// Test data
	queryParams := map[string]interface{}{
		"content": "test query",
		"tags":    []string{"tag1", "tag2"},
		"limit":   10,
	}
	chunkIDs := []string{"chunk1", "chunk2", "chunk3"}
	ttl := 5 * time.Minute
	
	// Set cached search
	err := cache.SetCachedSearch(ctx, queryParams, chunkIDs, ttl)
	assert.NoError(t, err)
	
	// Get cached search
	entry, err := cache.GetCachedSearch(ctx, queryParams)
	assert.NoError(t, err)
	assert.NotNil(t, entry)
	
	if entry != nil {
		assert.Equal(t, chunkIDs, entry.ChunkIDs)
		assert.Equal(t, len(chunkIDs), entry.ResultCount)
		assert.Equal(t, queryParams, entry.QueryParams)
		assert.True(t, entry.ExpiresAt.After(time.Now()))
	}
	
	// Verify monitoring was called
	assert.True(t, len(monitor.queries) > 0)
	
	// Test cache miss with different parameters
	differentParams := map[string]interface{}{
		"content": "different query",
	}
	
	entry, err = cache.GetCachedSearch(ctx, differentParams)
	assert.NoError(t, err)
	assert.Nil(t, entry) // Should be cache miss
}

func TestDatabaseSearchCache_ExpiredEntries(t *testing.T) {
	db := setupTestSearchCacheDB(t)
	defer db.Close()
	
	monitor := &MockQueryPerformanceMonitor{}
	config := DefaultSearchCacheConfig()
	config.CleanupInterval = 0
	
	cache := NewDatabaseSearchCache(db, config, monitor)
	ctx := context.Background()
	
	// Set an entry that expires immediately
	queryParams := map[string]interface{}{"content": "test"}
	chunkIDs := []string{"chunk1"}
	
	err := cache.SetCachedSearch(ctx, queryParams, chunkIDs, -1*time.Hour) // Already expired
	assert.NoError(t, err)
	
	// Try to get the expired entry
	entry, err := cache.GetCachedSearch(ctx, queryParams)
	assert.NoError(t, err)
	assert.Nil(t, entry) // Should not return expired entry
}

func TestDatabaseSearchCache_CleanupExpiredEntries(t *testing.T) {
	db := setupTestSearchCacheDB(t)
	defer db.Close()
	
	monitor := &MockQueryPerformanceMonitor{}
	config := DefaultSearchCacheConfig()
	config.CleanupInterval = 0
	
	cache := NewDatabaseSearchCache(db, config, monitor)
	ctx := context.Background()
	
	// Add some expired entries
	for i := 0; i < 3; i++ {
		queryParams := map[string]interface{}{"content": fmt.Sprintf("test%d", i)}
		chunkIDs := []string{fmt.Sprintf("chunk%d", i)}
		err := cache.SetCachedSearch(ctx, queryParams, chunkIDs, -1*time.Hour)
		assert.NoError(t, err)
	}
	
	// Add some valid entries
	for i := 0; i < 2; i++ {
		queryParams := map[string]interface{}{"content": fmt.Sprintf("valid%d", i)}
		chunkIDs := []string{fmt.Sprintf("validchunk%d", i)}
		err := cache.SetCachedSearch(ctx, queryParams, chunkIDs, 1*time.Hour)
		assert.NoError(t, err)
	}
	
	// Cleanup expired entries
	deletedCount, err := cache.CleanupExpiredEntries(ctx)
	assert.NoError(t, err)
	assert.Equal(t, 3, deletedCount)
	
	// Verify only valid entries remain
	var remainingCount int
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM chunk_search_cache").Scan(&remainingCount)
	assert.NoError(t, err)
	assert.Equal(t, 2, remainingCount)
}

func TestDatabaseSearchCache_InvalidateCache(t *testing.T) {
	db := setupTestSearchCacheDB(t)
	defer db.Close()
	
	monitor := &MockQueryPerformanceMonitor{}
	config := DefaultSearchCacheConfig()
	config.CleanupInterval = 0
	
	cache := NewDatabaseSearchCache(db, config, monitor)
	ctx := context.Background()
	
	// Add test entries
	testEntries := []map[string]interface{}{
		{"content": "test1"},
		{"content": "test2"},
		{"content": "other1"},
	}
	
	for _, params := range testEntries {
		err := cache.SetCachedSearch(ctx, params, []string{"chunk1"}, 1*time.Hour)
		assert.NoError(t, err)
	}
	
	// Test pattern invalidation (this would work with PostgreSQL LIKE)
	err := cache.InvalidateSearchCache(ctx, []string{"*"})
	assert.NoError(t, err)
	
	// Verify all entries are gone
	var count int
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM chunk_search_cache").Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestDatabaseSearchCache_UpdateHitCount(t *testing.T) {
	db := setupTestSearchCacheDB(t)
	defer db.Close()
	
	monitor := &MockQueryPerformanceMonitor{}
	config := DefaultSearchCacheConfig()
	config.CleanupInterval = 0
	
	cache := NewDatabaseSearchCache(db, config, monitor)
	ctx := context.Background()
	
	// Set a cached search
	queryParams := map[string]interface{}{"content": "test"}
	chunkIDs := []string{"chunk1"}
	
	err := cache.SetCachedSearch(ctx, queryParams, chunkIDs, 1*time.Hour)
	assert.NoError(t, err)
	
	// Get the entry to get its hash
	entry, err := cache.GetCachedSearch(ctx, queryParams)
	assert.NoError(t, err)
	assert.NotNil(t, entry)
	
	initialHitCount := entry.HitCount
	
	// Update hit count
	err = cache.UpdateHitCount(ctx, entry.SearchHash)
	assert.NoError(t, err)
	
	// Verify hit count was updated
	var hitCount int
	err = db.QueryRowContext(ctx, "SELECT hit_count FROM chunk_search_cache WHERE search_hash = ?", entry.SearchHash).Scan(&hitCount)
	assert.NoError(t, err)
	assert.Equal(t, initialHitCount+1, hitCount)
}

func TestDatabaseSearchCache_GetCacheStats(t *testing.T) {
	db := setupTestSearchCacheDB(t)
	defer db.Close()
	
	monitor := &MockQueryPerformanceMonitor{}
	config := DefaultSearchCacheConfig()
	config.CleanupInterval = 0
	config.StatsEnabled = true
	
	cache := NewDatabaseSearchCache(db, config, monitor)
	ctx := context.Background()
	
	// Add some test entries
	for i := 0; i < 5; i++ {
		queryParams := map[string]interface{}{"content": fmt.Sprintf("test%d", i)}
		chunkIDs := []string{fmt.Sprintf("chunk%d", i)}
		
		var ttl time.Duration
		if i < 2 {
			ttl = -1 * time.Hour // Expired
		} else {
			ttl = 1 * time.Hour // Valid
		}
		
		err := cache.SetCachedSearch(ctx, queryParams, chunkIDs, ttl)
		assert.NoError(t, err)
	}
	
	// Get cache stats
	stats, err := cache.GetCacheStats(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, stats)
	
	assert.Equal(t, 5, stats.TotalEntries)
	assert.Equal(t, 2, stats.ExpiredEntries)
	assert.True(t, stats.CacheSize > 0)
}

func TestDatabaseSearchCache_GetOptimizationSuggestions(t *testing.T) {
	db := setupTestSearchCacheDB(t)
	defer db.Close()
	
	monitor := &MockQueryPerformanceMonitor{}
	config := DefaultSearchCacheConfig()
	config.CleanupInterval = 0
	config.OptimizationEnabled = true
	config.StatsEnabled = true
	
	cache := NewDatabaseSearchCache(db, config, monitor)
	ctx := context.Background()
	
	// Add entries to trigger optimization suggestions
	// Add many expired entries to trigger expiration suggestion
	for i := 0; i < 10; i++ {
		queryParams := map[string]interface{}{"content": fmt.Sprintf("expired%d", i)}
		chunkIDs := []string{fmt.Sprintf("chunk%d", i)}
		err := cache.SetCachedSearch(ctx, queryParams, chunkIDs, -1*time.Hour)
		assert.NoError(t, err)
	}
	
	// Add a few valid entries
	for i := 0; i < 2; i++ {
		queryParams := map[string]interface{}{"content": fmt.Sprintf("valid%d", i)}
		chunkIDs := []string{fmt.Sprintf("validchunk%d", i)}
		err := cache.SetCachedSearch(ctx, queryParams, chunkIDs, 1*time.Hour)
		assert.NoError(t, err)
	}
	
	// Get optimization suggestions
	suggestions, err := cache.GetOptimizationSuggestions(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, suggestions)
	
	// Should have suggestions due to high expired entry ratio
	assert.True(t, len(suggestions) > 0)
	
	// Check for expiration suggestion
	hasExpirationSuggestion := false
	for _, suggestion := range suggestions {
		if suggestion.Type == "expiration" {
			hasExpirationSuggestion = true
			assert.Equal(t, "medium", suggestion.Priority)
			assert.Contains(t, suggestion.Description, "expired entries")
			break
		}
	}
	assert.True(t, hasExpirationSuggestion)
}

func TestDatabaseSearchCache_ConcurrentAccess(t *testing.T) {
	db := setupTestSearchCacheDB(t)
	defer db.Close()
	
	monitor := &MockQueryPerformanceMonitor{}
	config := DefaultSearchCacheConfig()
	config.CleanupInterval = 0
	
	cache := NewDatabaseSearchCache(db, config, monitor)
	ctx := context.Background()
	
	// Test concurrent reads and writes
	done := make(chan bool, 10)
	
	// Start multiple goroutines doing cache operations
	for i := 0; i < 5; i++ {
		go func(id int) {
			defer func() { done <- true }()
			
			queryParams := map[string]interface{}{"content": fmt.Sprintf("concurrent%d", id)}
			chunkIDs := []string{fmt.Sprintf("chunk%d", id)}
			
			// Set cache entry
			err := cache.SetCachedSearch(ctx, queryParams, chunkIDs, 1*time.Hour)
			assert.NoError(t, err)
			
			// Get cache entry
			entry, err := cache.GetCachedSearch(ctx, queryParams)
			assert.NoError(t, err)
			assert.NotNil(t, entry)
		}(i)
	}
	
	// Start goroutines doing cleanup
	for i := 0; i < 2; i++ {
		go func() {
			defer func() { done <- true }()
			
			_, err := cache.CleanupExpiredEntries(ctx)
			assert.NoError(t, err)
		}()
	}
	
	// Wait for all goroutines to complete
	for i := 0; i < 7; i++ {
		<-done
	}
	
	// Verify final state
	var count int
	err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM chunk_search_cache").Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 5, count) // Should have 5 entries from concurrent writes
}

func TestDatabaseSearchCache_HashConsistency(t *testing.T) {
	monitor := &MockQueryPerformanceMonitor{}
	config := DefaultSearchCacheConfig()
	
	cache := &DatabaseSearchCache{
		config:  config,
		monitor: monitor,
	}
	
	// Test that same parameters generate same hash
	params1 := map[string]interface{}{
		"content": "test",
		"tags":    []string{"tag1", "tag2"},
		"limit":   10,
	}
	
	params2 := map[string]interface{}{
		"limit":   10,
		"content": "test",
		"tags":    []string{"tag1", "tag2"},
	}
	
	hash1 := cache.generateSearchHash(params1)
	hash2 := cache.generateSearchHash(params2)
	
	assert.Equal(t, hash1, hash2, "Same parameters should generate same hash regardless of order")
	
	// Test that different parameters generate different hashes
	params3 := map[string]interface{}{
		"content": "different",
		"tags":    []string{"tag1", "tag2"},
		"limit":   10,
	}
	
	hash3 := cache.generateSearchHash(params3)
	assert.NotEqual(t, hash1, hash3, "Different parameters should generate different hashes")
}

func TestDatabaseSearchCache_ConfigDisabled(t *testing.T) {
	db := setupTestSearchCacheDB(t)
	defer db.Close()
	
	monitor := &MockQueryPerformanceMonitor{}
	config := DefaultSearchCacheConfig()
	config.StatsEnabled = false
	config.OptimizationEnabled = false
	
	cache := NewDatabaseSearchCache(db, config, monitor)
	ctx := context.Background()
	
	// Test that disabled features return empty results
	stats, err := cache.GetCacheStats(ctx)
	assert.NoError(t, err)
	assert.Equal(t, &SearchCacheStats{}, stats)
	
	suggestions, err := cache.GetOptimizationSuggestions(ctx)
	assert.NoError(t, err)
	assert.Equal(t, []OptimizationSuggestion{}, suggestions)
}

// Benchmark tests
func BenchmarkDatabaseSearchCache_SetCachedSearch(b *testing.B) {
	db := setupTestSearchCacheDB(&testing.T{})
	defer db.Close()
	
	monitor := &MockQueryPerformanceMonitor{}
	config := DefaultSearchCacheConfig()
	config.CleanupInterval = 0
	
	cache := NewDatabaseSearchCache(db, config, monitor)
	ctx := context.Background()
	
	queryParams := map[string]interface{}{
		"content": "benchmark test",
		"tags":    []string{"tag1", "tag2"},
		"limit":   100,
	}
	chunkIDs := make([]string, 100)
	for i := 0; i < 100; i++ {
		chunkIDs[i] = fmt.Sprintf("chunk%d", i)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		params := make(map[string]interface{})
		for k, v := range queryParams {
			params[k] = v
		}
		params["iteration"] = i // Make each iteration unique
		
		err := cache.SetCachedSearch(ctx, params, chunkIDs, 1*time.Hour)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDatabaseSearchCache_GetCachedSearch(b *testing.B) {
	db := setupTestSearchCacheDB(&testing.T{})
	defer db.Close()
	
	monitor := &MockQueryPerformanceMonitor{}
	config := DefaultSearchCacheConfig()
	config.CleanupInterval = 0
	
	cache := NewDatabaseSearchCache(db, config, monitor)
	ctx := context.Background()
	
	// Pre-populate cache
	queryParams := map[string]interface{}{
		"content": "benchmark test",
		"tags":    []string{"tag1", "tag2"},
		"limit":   100,
	}
	chunkIDs := make([]string, 100)
	for i := 0; i < 100; i++ {
		chunkIDs[i] = fmt.Sprintf("chunk%d", i)
	}
	
	err := cache.SetCachedSearch(ctx, queryParams, chunkIDs, 1*time.Hour)
	if err != nil {
		b.Fatal(err)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		entry, err := cache.GetCachedSearch(ctx, queryParams)
		if err != nil {
			b.Fatal(err)
		}
		if entry == nil {
			b.Fatal("Expected cache hit")
		}
	}
}