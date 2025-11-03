package services

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Unit tests for search cache functionality that don't require a database

func TestSearchCacheConfig_Defaults(t *testing.T) {
	config := DefaultSearchCacheConfig()
	
	assert.NotNil(t, config)
	assert.Equal(t, 15*time.Minute, config.DefaultTTL)
	assert.Equal(t, 50000, config.MaxCacheEntries)
	assert.Equal(t, 10*time.Minute, config.CleanupInterval)
	assert.Equal(t, 5, config.HitCountThreshold)
	assert.True(t, config.OptimizationEnabled)
	assert.True(t, config.StatsEnabled)
}

func TestSearchCacheFactoryConfig_Defaults(t *testing.T) {
	config := DefaultSearchCacheFactoryConfig()
	
	assert.NotNil(t, config)
	assert.True(t, config.SearchCacheEnabled)
	assert.Equal(t, 15*time.Minute, config.SearchCacheDefaultTTL)
	assert.Equal(t, 50000, config.SearchCacheMaxEntries)
	assert.Equal(t, 10*time.Minute, config.SearchCacheCleanupInterval)
	assert.True(t, config.InMemoryCacheEnabled)
	assert.Equal(t, 1000, config.InMemoryCacheMaxSize)
	assert.Equal(t, 5*time.Minute, config.InMemoryCacheCleanupInterval)
	assert.True(t, config.MonitoringEnabled)
	assert.True(t, config.StatsEnabled)
	assert.True(t, config.OptimizationEnabled)
}

func TestNoOpSearchCacheService(t *testing.T) {
	cache := &NoOpSearchCacheService{}
	ctx := context.Background()
	
	// Test GetCachedSearch
	entry, err := cache.GetCachedSearch(ctx, map[string]interface{}{"test": "value"})
	assert.NoError(t, err)
	assert.Nil(t, entry)
	
	// Test SetCachedSearch
	err = cache.SetCachedSearch(ctx, map[string]interface{}{"test": "value"}, []string{"chunk1"}, time.Minute)
	assert.NoError(t, err)
	
	// Test InvalidateSearchCache
	err = cache.InvalidateSearchCache(ctx, []string{"*"})
	assert.NoError(t, err)
	
	// Test CleanupExpiredEntries
	count, err := cache.CleanupExpiredEntries(ctx)
	assert.NoError(t, err)
	assert.Equal(t, 0, count)
	
	// Test GetCacheStats
	stats, err := cache.GetCacheStats(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, stats)
	assert.Equal(t, 0, stats.TotalEntries)
	
	// Test GetOptimizationSuggestions
	suggestions, err := cache.GetOptimizationSuggestions(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, suggestions)
	assert.Equal(t, 0, len(suggestions))
	
	// Test UpdateHitCount
	err = cache.UpdateHitCount(ctx, "test-hash")
	assert.NoError(t, err)
}

func TestNoOpQueryPerformanceMonitor(t *testing.T) {
	monitor := &NoOpQueryPerformanceMonitor{}
	
	// Test RecordQuery
	monitor.RecordQuery("test", time.Millisecond, 10)
	
	// Test GetQueryStats
	stats := monitor.GetQueryStats()
	assert.Equal(t, int64(0), stats.TotalQueries)
	
	// Test GetSlowQueries
	slowQueries := monitor.GetSlowQueries(10)
	assert.NotNil(t, slowQueries)
	assert.Equal(t, 0, len(slowQueries))
	
	// Test RecordSlowQuery
	monitor.RecordSlowQuery("SELECT * FROM test", time.Second, map[string]interface{}{"param": "value"})
}

func TestNoOpCacheService(t *testing.T) {
	cache := &NoOpCacheService{}
	ctx := context.Background()
	
	// Test Get
	var dest interface{}
	err := cache.Get(ctx, "test-key", &dest)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cache disabled")
	
	// Test GetDirect
	value, found := cache.GetDirect(ctx, "test-key")
	assert.Nil(t, value)
	assert.False(t, found)
	
	// Test Set
	err = cache.Set(ctx, "test-key", "test-value", time.Minute)
	assert.NoError(t, err)
	
	// Test Delete
	err = cache.Delete(ctx, "test-key")
	assert.NoError(t, err)
	
	// Test DeletePattern
	err = cache.DeletePattern(ctx, "test-*")
	assert.NoError(t, err)
	
	// Test Clear
	err = cache.Clear(ctx)
	assert.NoError(t, err)
	
	// Test GetStats
	stats := cache.GetStats()
	assert.Equal(t, int64(0), stats.Hits)
	assert.Equal(t, int64(0), stats.Misses)
	assert.Equal(t, 0, stats.Size)
}

func TestSearchCacheStats_Structure(t *testing.T) {
	stats := &SearchCacheStats{
		TotalEntries:    100,
		ExpiredEntries:  10,
		AverageHitCount: 5.5,
		HitRate:         0.85,
		CacheSize:       1024000,
		TopQueries: []QueryStats{
			{
				SearchHash:   "hash1",
				QueryParams:  map[string]interface{}{"content": "test"},
				HitCount:     10,
				ResultCount:  5,
				LastAccessed: time.Now(),
				CreatedAt:    time.Now(),
			},
		},
		ExpirationStats: ExpirationStats{
			EntriesExpiringSoon: 5,
			EntriesExpiredToday: 15,
			AverageTTL:          10 * time.Minute,
		},
	}
	
	assert.Equal(t, 100, stats.TotalEntries)
	assert.Equal(t, 10, stats.ExpiredEntries)
	assert.Equal(t, 5.5, stats.AverageHitCount)
	assert.Equal(t, 0.85, stats.HitRate)
	assert.Equal(t, int64(1024000), stats.CacheSize)
	assert.Equal(t, 1, len(stats.TopQueries))
	assert.Equal(t, "hash1", stats.TopQueries[0].SearchHash)
	assert.Equal(t, 5, stats.ExpirationStats.EntriesExpiringSoon)
	assert.Equal(t, 15, stats.ExpirationStats.EntriesExpiredToday)
	assert.Equal(t, 10*time.Minute, stats.ExpirationStats.AverageTTL)
}

func TestOptimizationSuggestion_Structure(t *testing.T) {
	suggestion := OptimizationSuggestion{
		Type:        "hit_rate",
		Priority:    "high",
		Description: "Cache hit rate is low",
		Action:      "Increase TTL values",
		Impact:      "Better performance",
		Data: map[string]interface{}{
			"current_hit_rate": 0.3,
			"target_hit_rate":  0.7,
		},
	}
	
	assert.Equal(t, "hit_rate", suggestion.Type)
	assert.Equal(t, "high", suggestion.Priority)
	assert.Equal(t, "Cache hit rate is low", suggestion.Description)
	assert.Equal(t, "Increase TTL values", suggestion.Action)
	assert.Equal(t, "Better performance", suggestion.Impact)
	assert.Equal(t, 0.3, suggestion.Data["current_hit_rate"])
	assert.Equal(t, 0.7, suggestion.Data["target_hit_rate"])
}

func TestSearchCacheFactory_CreateWithDisabledFeatures(t *testing.T) {
	config := &SearchCacheFactoryConfig{
		SearchCacheEnabled:      false,
		InMemoryCacheEnabled:    false,
		MonitoringEnabled:       false,
		StatsEnabled:           false,
		OptimizationEnabled:    false,
	}
	
	factory := NewSearchCacheFactory(nil, config)
	assert.NotNil(t, factory)
	
	// Test getting search cache service when disabled
	searchCache := factory.GetSearchCacheService()
	assert.NotNil(t, searchCache)
	
	// Should be a no-op implementation
	ctx := context.Background()
	entry, err := searchCache.GetCachedSearch(ctx, map[string]interface{}{"test": "value"})
	assert.NoError(t, err)
	assert.Nil(t, entry)
}

func TestSearchCacheFactory_CreateWithEnabledFeatures(t *testing.T) {
	config := DefaultSearchCacheFactoryConfig()
	
	factory := NewSearchCacheFactory(nil, config)
	assert.NotNil(t, factory)
	assert.Equal(t, config, factory.config)
}

// Benchmark tests for cache key generation and other operations
func BenchmarkSearchCacheHashGeneration(b *testing.B) {
	cache := &DatabaseSearchCache{
		config: DefaultSearchCacheConfig(),
	}
	
	queryParams := map[string]interface{}{
		"content":     "test search query",
		"tags":        []string{"tag1", "tag2", "tag3"},
		"is_page":     true,
		"limit":       100,
		"offset":      0,
		"metadata":    map[string]interface{}{"category": "test"},
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hash := cache.generateSearchHash(queryParams)
		if hash == "" {
			b.Fatal("Generated empty hash")
		}
	}
}

func BenchmarkOptimizationSuggestionCreation(b *testing.B) {
	stats := &SearchCacheStats{
		TotalEntries:    1000,
		ExpiredEntries:  250,
		AverageHitCount: 3.5,
		HitRate:         0.4,
		CacheSize:       50 * 1024 * 1024, // 50MB
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		suggestions := []OptimizationSuggestion{}
		
		// Simulate suggestion generation logic
		if stats.HitRate < 0.5 {
			suggestions = append(suggestions, OptimizationSuggestion{
				Type:        "hit_rate",
				Priority:    "high",
				Description: "Cache hit rate is low",
				Action:      "Consider increasing TTL",
				Impact:      "Better performance",
			})
		}
		
		if stats.ExpiredEntries > stats.TotalEntries/4 {
			suggestions = append(suggestions, OptimizationSuggestion{
				Type:        "expiration",
				Priority:    "medium",
				Description: "High number of expired entries",
				Action:      "Run cleanup more frequently",
				Impact:      "Better cache efficiency",
			})
		}
		
		if len(suggestions) == 0 {
			b.Fatal("No suggestions generated")
		}
	}
}