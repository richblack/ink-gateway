package services

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInMemoryCache_SetAndGet(t *testing.T) {
	cache := NewInMemoryCache(10, time.Minute)
	defer cache.Stop()
	
	ctx := context.Background()
	
	// Test setting and getting a value
	testValue := map[string]interface{}{
		"id":   "test-123",
		"name": "Test Item",
		"count": 42,
	}
	
	err := cache.Set(ctx, "test-key", testValue, time.Hour)
	require.NoError(t, err)
	
	var retrieved map[string]interface{}
	err = cache.Get(ctx, "test-key", &retrieved)
	require.NoError(t, err)
	
	assert.Equal(t, testValue["id"], retrieved["id"])
	assert.Equal(t, testValue["name"], retrieved["name"])
	assert.Equal(t, float64(42), retrieved["count"]) // JSON unmarshaling converts numbers to float64
}

func TestInMemoryCache_Miss(t *testing.T) {
	cache := NewInMemoryCache(10, time.Minute)
	defer cache.Stop()
	
	ctx := context.Background()
	
	var result string
	err := cache.Get(ctx, "nonexistent-key", &result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cache miss")
}

func TestInMemoryCache_Expiration(t *testing.T) {
	cache := NewInMemoryCache(10, time.Hour) // Long cleanup interval to avoid interference
	defer cache.Stop()
	
	ctx := context.Background()
	
	// Set value with short TTL
	err := cache.Set(ctx, "expire-key", "test-value", time.Millisecond*50)
	require.NoError(t, err)
	
	// Should be available immediately
	var result string
	err = cache.Get(ctx, "expire-key", &result)
	require.NoError(t, err)
	assert.Equal(t, "test-value", result)
	
	// Wait for expiration
	time.Sleep(time.Millisecond * 100)
	
	// Should be expired now
	err = cache.Get(ctx, "expire-key", &result)
	assert.Error(t, err)
	// The error could be either "expired" or "not found" depending on cleanup timing
	assert.True(t, 
		strings.Contains(err.Error(), "expired") || strings.Contains(err.Error(), "not found"),
		"Expected error to contain 'expired' or 'not found', got: %s", err.Error())
}

func TestInMemoryCache_Eviction(t *testing.T) {
	cache := NewInMemoryCache(3, time.Minute) // Small cache size
	defer cache.Stop()
	
	ctx := context.Background()
	
	// Fill cache to capacity
	for i := 0; i < 3; i++ {
		err := cache.Set(ctx, fmt.Sprintf("key-%d", i), fmt.Sprintf("value-%d", i), time.Hour)
		require.NoError(t, err)
	}
	
	// Add one more item, should evict oldest
	err := cache.Set(ctx, "key-3", "value-3", time.Hour)
	require.NoError(t, err)
	
	// First item should be evicted
	var result string
	err = cache.Get(ctx, "key-0", &result)
	assert.Error(t, err)
	
	// Last item should still be there
	err = cache.Get(ctx, "key-3", &result)
	require.NoError(t, err)
	assert.Equal(t, "value-3", result)
	
	// Check eviction stats
	stats := cache.GetStats()
	assert.Equal(t, int64(1), stats.Evictions)
}

func TestInMemoryCache_Delete(t *testing.T) {
	cache := NewInMemoryCache(10, time.Minute)
	defer cache.Stop()
	
	ctx := context.Background()
	
	// Set and verify
	err := cache.Set(ctx, "delete-key", "delete-value", time.Hour)
	require.NoError(t, err)
	
	var result string
	err = cache.Get(ctx, "delete-key", &result)
	require.NoError(t, err)
	
	// Delete and verify
	err = cache.Delete(ctx, "delete-key")
	require.NoError(t, err)
	
	err = cache.Get(ctx, "delete-key", &result)
	assert.Error(t, err)
}

func TestInMemoryCache_Clear(t *testing.T) {
	cache := NewInMemoryCache(10, time.Minute)
	defer cache.Stop()
	
	ctx := context.Background()
	
	// Add multiple items
	for i := 0; i < 5; i++ {
		err := cache.Set(ctx, fmt.Sprintf("clear-key-%d", i), fmt.Sprintf("value-%d", i), time.Hour)
		require.NoError(t, err)
	}
	
	// Verify items exist
	stats := cache.GetStats()
	assert.Equal(t, 5, stats.Size)
	
	// Clear cache
	err := cache.Clear(ctx)
	require.NoError(t, err)
	
	// Verify cache is empty
	stats = cache.GetStats()
	assert.Equal(t, 0, stats.Size)
	
	// Verify items are gone
	var result string
	err = cache.Get(ctx, "clear-key-0", &result)
	assert.Error(t, err)
}

func TestInMemoryCache_Stats(t *testing.T) {
	cache := NewInMemoryCache(10, time.Minute)
	defer cache.Stop()
	
	ctx := context.Background()
	
	// Initial stats
	stats := cache.GetStats()
	assert.Equal(t, int64(0), stats.Hits)
	assert.Equal(t, int64(0), stats.Misses)
	assert.Equal(t, float64(0), stats.HitRate)
	
	// Add item and test hit
	err := cache.Set(ctx, "stats-key", "stats-value", time.Hour)
	require.NoError(t, err)
	
	var result string
	err = cache.Get(ctx, "stats-key", &result)
	require.NoError(t, err)
	
	// Test miss
	err = cache.Get(ctx, "nonexistent", &result)
	assert.Error(t, err)
	
	// Check updated stats
	stats = cache.GetStats()
	assert.Equal(t, int64(1), stats.Hits)
	assert.Equal(t, int64(1), stats.Misses)
	assert.Equal(t, float64(0.5), stats.HitRate)
}

func TestCacheConfig_Defaults(t *testing.T) {
	config := DefaultCacheConfig()
	
	assert.Equal(t, 1000, config.MaxSize)
	assert.Equal(t, 5*time.Minute, config.CleanupInterval)
	assert.Equal(t, 30*time.Minute, config.DefaultTTL)
	assert.True(t, config.Enabled)
}

func BenchmarkInMemoryCache_Set(b *testing.B) {
	cache := NewInMemoryCache(10000, time.Minute)
	defer cache.Stop()
	
	ctx := context.Background()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("bench-key-%d", i)
		value := fmt.Sprintf("bench-value-%d", i)
		cache.Set(ctx, key, value, time.Hour)
	}
}

func BenchmarkInMemoryCache_Get(b *testing.B) {
	cache := NewInMemoryCache(10000, time.Minute)
	defer cache.Stop()
	
	ctx := context.Background()
	
	// Pre-populate cache
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("bench-key-%d", i)
		value := fmt.Sprintf("bench-value-%d", i)
		cache.Set(ctx, key, value, time.Hour)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("bench-key-%d", i%1000)
		var result string
		cache.Get(ctx, key, &result)
	}
}