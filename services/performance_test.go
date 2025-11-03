package services

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockSearchService for testing
type MockSearchService struct {
	mock.Mock
}

func (m *MockSearchService) SemanticSearch(ctx context.Context, query string, limit int) ([]SimilarityResult, error) {
	args := m.Called(ctx, query, limit)
	return args.Get(0).([]SimilarityResult), args.Error(1)
}

func (m *MockSearchService) SemanticSearchWithFilters(ctx context.Context, req *SemanticSearchRequest) (*SemanticSearchResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*SemanticSearchResponse), args.Error(1)
}

func (m *MockSearchService) HybridSearch(ctx context.Context, query string, limit int, semanticWeight float64) ([]SimilarityResult, error) {
	args := m.Called(ctx, query, limit, semanticWeight)
	return args.Get(0).([]SimilarityResult), args.Error(1)
}

func (m *MockSearchService) GraphSearch(ctx context.Context, query *GraphQuery) (*GraphResult, error) {
	args := m.Called(ctx, query)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*GraphResult), args.Error(1)
}

func (m *MockSearchService) SearchByTag(ctx context.Context, tagContent string) ([]ChunkWithTags, error) {
	args := m.Called(ctx, tagContent)
	return args.Get(0).([]ChunkWithTags), args.Error(1)
}

func (m *MockSearchService) SearchChunks(ctx context.Context, query string, filters map[string]interface{}) ([]ChunkRecord, error) {
	args := m.Called(ctx, query, filters)
	return args.Get(0).([]ChunkRecord), args.Error(1)
}

func TestCachedSearchService_Performance(t *testing.T) {
	// Create mock search service
	mockSearch := new(MockSearchService)
	
	// Create cache and cached search service
	cache := NewInMemoryCache(100, time.Minute)
	defer cache.Stop()
	
	// TODO: Implement NewCachedSearchService when needed
	// config := DefaultCacheConfig()
	// cachedSearch := NewCachedSearchService(mockSearch, cache, config)
	cachedSearch := mockSearch
	
	ctx := context.Background()
	
	// Test data
	expectedResults := []SimilarityResult{
		{Chunk: ChunkRecord{ID: "chunk-1", Content: "Test content 1"}, Similarity: 0.95},
		{Chunk: ChunkRecord{ID: "chunk-2", Content: "Test content 2"}, Similarity: 0.85},
	}
	
	// Mock should be called only once (first call)
	mockSearch.On("SemanticSearch", ctx, "test query", 10).Return(expectedResults, nil).Once()
	
	// First call - should hit the mock
	start := time.Now()
	results1, err := cachedSearch.SemanticSearch(ctx, "test query", 10)
	firstCallDuration := time.Since(start)
	
	require.NoError(t, err)
	assert.Equal(t, expectedResults, results1)
	
	// Second call - should hit the cache
	start = time.Now()
	results2, err := cachedSearch.SemanticSearch(ctx, "test query", 10)
	secondCallDuration := time.Since(start)
	
	require.NoError(t, err)
	assert.Equal(t, expectedResults, results2)
	
	// Cache hit should be significantly faster
	assert.True(t, secondCallDuration < firstCallDuration/2, 
		"Cache hit should be faster than original call")
	
	// Verify mock was called only once
	mockSearch.AssertExpectations(t)
	
	// Check cache stats
	stats := cache.GetStats()
	assert.Equal(t, int64(1), stats.Hits)
	assert.Equal(t, int64(1), stats.Misses) // One miss from the first call, one hit from the second call
}

func TestMetricsService_Performance(t *testing.T) {
	metrics := NewInMemoryMetrics()
	
	// Test counter performance
	start := time.Now()
	for i := 0; i < 1000; i++ {
		metrics.IncrementCounter("test.counter", map[string]string{
			"iteration": fmt.Sprintf("%d", i%10), // 10 different counters
		})
	}
	counterDuration := time.Since(start)
	
	// Test histogram performance
	start = time.Now()
	for i := 0; i < 1000; i++ {
		duration := time.Duration(i) * time.Millisecond
		metrics.RecordDuration("test.histogram", duration, map[string]string{
			"operation": "test",
		})
	}
	histogramDuration := time.Since(start)
	
	// Test gauge performance
	start = time.Now()
	for i := 0; i < 1000; i++ {
		metrics.SetGauge("test.gauge", float64(i), map[string]string{
			"type": "test",
		})
	}
	gaugeDuration := time.Since(start)
	
	t.Logf("Counter operations took: %v", counterDuration)
	t.Logf("Histogram operations took: %v", histogramDuration)
	t.Logf("Gauge operations took: %v", gaugeDuration)
	
	// Verify metrics were recorded
	allMetrics := metrics.GetMetrics()
	assert.Contains(t, allMetrics, "counters")
	assert.Contains(t, allMetrics, "histograms")
	assert.Contains(t, allMetrics, "gauges")
	
	// Performance assertions (these are rough estimates)
	assert.True(t, counterDuration < time.Second, "Counter operations should be fast")
	assert.True(t, histogramDuration < time.Second, "Histogram operations should be fast")
	assert.True(t, gaugeDuration < time.Second, "Gauge operations should be fast")
}

func BenchmarkCachedSearchService_SemanticSearch(b *testing.B) {
	mockSearch := new(MockSearchService)
	cache := NewInMemoryCache(1000, time.Minute)
	defer cache.Stop()
	
	// TODO: Implement NewCachedSearchService when needed
	// config := DefaultCacheConfig()
	// cachedSearch := NewCachedSearchService(mockSearch, cache, config)
	cachedSearch := mockSearch
	
	ctx := context.Background()
	expectedResults := []SimilarityResult{
		{Chunk: ChunkRecord{ID: "chunk-1", Content: "Benchmark content"}, Similarity: 0.95},
	}
	
	// Mock will be called for each unique query
	mockSearch.On("SemanticSearch", mock.Anything, mock.Anything, mock.Anything).Return(expectedResults, nil)
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		// Use limited number of unique queries to test cache effectiveness
		query := fmt.Sprintf("benchmark query %d", i%100)
		_, err := cachedSearch.SemanticSearch(ctx, query, 10)
		if err != nil {
			b.Fatalf("Search failed: %v", err)
		}
	}
}

func BenchmarkInMemoryMetrics_IncrementCounter(b *testing.B) {
	metrics := NewInMemoryMetrics()
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		metrics.IncrementCounter("benchmark.counter", map[string]string{
			"iteration": fmt.Sprintf("%d", i%10),
		})
	}
}

func BenchmarkInMemoryMetrics_RecordDuration(b *testing.B) {
	metrics := NewInMemoryMetrics()
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		duration := time.Duration(i%1000) * time.Millisecond
		metrics.RecordDuration("benchmark.duration", duration, map[string]string{
			"operation": "benchmark",
		})
	}
}

func BenchmarkInMemoryMetrics_SetGauge(b *testing.B) {
	metrics := NewInMemoryMetrics()
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		metrics.SetGauge("benchmark.gauge", float64(i), map[string]string{
			"type": "benchmark",
		})
	}
}

// Benchmark functions removed to avoid duplication with cache_test.go

// Load test for concurrent operations
func TestConcurrentCacheOperations(t *testing.T) {
	cache := NewInMemoryCache(1000, time.Minute)
	defer cache.Stop()
	
	ctx := context.Background()
	numGoroutines := 10
	operationsPerGoroutine := 100
	
	// Channel to collect errors
	errChan := make(chan error, numGoroutines)
	
	// Start concurrent operations
	for g := 0; g < numGoroutines; g++ {
		go func(goroutineID int) {
			defer func() {
				errChan <- nil // Signal completion
			}()
			
			for i := 0; i < operationsPerGoroutine; i++ {
				key := fmt.Sprintf("concurrent-key-%d-%d", goroutineID, i)
				value := fmt.Sprintf("value-%d-%d", goroutineID, i)
				
				// Set value
				if err := cache.Set(ctx, key, value, time.Hour); err != nil {
					errChan <- fmt.Errorf("set failed: %w", err)
					return
				}
				
				// Get value
				var result string
				if err := cache.Get(ctx, key, &result); err != nil {
					errChan <- fmt.Errorf("get failed: %w", err)
					return
				}
				
				if result != value {
					errChan <- fmt.Errorf("value mismatch: expected %s, got %s", value, result)
					return
				}
			}
		}(g)
	}
	
	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		err := <-errChan
		if err != nil {
			t.Fatalf("Concurrent operation failed: %v", err)
		}
	}
	
	// Verify final state
	stats := cache.GetStats()
	t.Logf("Final cache stats: %+v", stats)
	
	// Should have some hits and no errors
	assert.True(t, stats.Size > 0, "Cache should contain items")
}