package services

import (
	"context"
	"fmt"
	"semantic-text-processor/models"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockUnifiedChunkService for testing
type MockUnifiedChunkService struct {
	mock.Mock
}

func (m *MockUnifiedChunkService) CreateChunk(ctx context.Context, chunk *models.UnifiedChunkRecord) error {
	args := m.Called(ctx, chunk)
	return args.Error(0)
}

func (m *MockUnifiedChunkService) GetChunk(ctx context.Context, chunkID string) (*models.UnifiedChunkRecord, error) {
	args := m.Called(ctx, chunkID)
	return args.Get(0).(*models.UnifiedChunkRecord), args.Error(1)
}

func (m *MockUnifiedChunkService) UpdateChunk(ctx context.Context, chunk *models.UnifiedChunkRecord) error {
	args := m.Called(ctx, chunk)
	return args.Error(0)
}

func (m *MockUnifiedChunkService) DeleteChunk(ctx context.Context, chunkID string) error {
	args := m.Called(ctx, chunkID)
	return args.Error(0)
}

func (m *MockUnifiedChunkService) BatchCreateChunks(ctx context.Context, chunks []models.UnifiedChunkRecord) error {
	args := m.Called(ctx, chunks)
	return args.Error(0)
}

func (m *MockUnifiedChunkService) BatchUpdateChunks(ctx context.Context, chunks []models.UnifiedChunkRecord) error {
	args := m.Called(ctx, chunks)
	return args.Error(0)
}

func (m *MockUnifiedChunkService) AddTags(ctx context.Context, chunkID string, tagChunkIDs []string) error {
	args := m.Called(ctx, chunkID, tagChunkIDs)
	return args.Error(0)
}

func (m *MockUnifiedChunkService) RemoveTags(ctx context.Context, chunkID string, tagChunkIDs []string) error {
	args := m.Called(ctx, chunkID, tagChunkIDs)
	return args.Error(0)
}

func (m *MockUnifiedChunkService) GetChunkTags(ctx context.Context, chunkID string) ([]models.UnifiedChunkRecord, error) {
	args := m.Called(ctx, chunkID)
	return args.Get(0).([]models.UnifiedChunkRecord), args.Error(1)
}

func (m *MockUnifiedChunkService) GetChunksByTag(ctx context.Context, tagChunkID string) ([]models.UnifiedChunkRecord, error) {
	args := m.Called(ctx, tagChunkID)
	return args.Get(0).([]models.UnifiedChunkRecord), args.Error(1)
}

func (m *MockUnifiedChunkService) GetChunksByTags(ctx context.Context, tagChunkIDs []string, matchType string) ([]models.UnifiedChunkRecord, error) {
	args := m.Called(ctx, tagChunkIDs, matchType)
	return args.Get(0).([]models.UnifiedChunkRecord), args.Error(1)
}

func (m *MockUnifiedChunkService) GetChildren(ctx context.Context, parentChunkID string) ([]models.UnifiedChunkRecord, error) {
	args := m.Called(ctx, parentChunkID)
	return args.Get(0).([]models.UnifiedChunkRecord), args.Error(1)
}

func (m *MockUnifiedChunkService) GetDescendants(ctx context.Context, ancestorChunkID string, maxDepth int) ([]models.UnifiedChunkRecord, error) {
	args := m.Called(ctx, ancestorChunkID, maxDepth)
	return args.Get(0).([]models.UnifiedChunkRecord), args.Error(1)
}

func (m *MockUnifiedChunkService) GetAncestors(ctx context.Context, chunkID string) ([]models.UnifiedChunkRecord, error) {
	args := m.Called(ctx, chunkID)
	return args.Get(0).([]models.UnifiedChunkRecord), args.Error(1)
}

func (m *MockUnifiedChunkService) MoveChunk(ctx context.Context, chunkID, newParentID string) error {
	args := m.Called(ctx, chunkID, newParentID)
	return args.Error(0)
}

func (m *MockUnifiedChunkService) SearchChunks(ctx context.Context, query *models.SearchQuery) (*models.SearchResult, error) {
	args := m.Called(ctx, query)
	return args.Get(0).(*models.SearchResult), args.Error(1)
}

func (m *MockUnifiedChunkService) SearchByContent(ctx context.Context, content string, filters map[string]interface{}) ([]models.UnifiedChunkRecord, error) {
	args := m.Called(ctx, content, filters)
	return args.Get(0).([]models.UnifiedChunkRecord), args.Error(1)
}

func TestQueryCacheManager_GenerateCacheKey(t *testing.T) {
	cache := NewInMemoryCache(100, time.Minute)
	defer cache.Stop()
	
	monitor := NewInMemoryPerformanceMonitor(100*time.Millisecond, 10)
	config := DefaultQueryCacheConfig()
	
	qcm := NewQueryCacheManager(cache, monitor, config)
	
	tests := []struct {
		name       string
		keyType    string
		identifier string
		params     map[string]interface{}
		expectSame bool
	}{
		{
			name:       "simple key",
			keyType:    "chunk",
			identifier: "test-id",
			params:     nil,
			expectSame: true,
		},
		{
			name:       "key with params",
			keyType:    "search",
			identifier: "",
			params:     map[string]interface{}{"content": "test", "limit": 10},
			expectSame: true,
		},
		{
			name:       "same params different order",
			keyType:    "search",
			identifier: "",
			params:     map[string]interface{}{"limit": 10, "content": "test"},
			expectSame: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key1 := qcm.GenerateCacheKey(tt.keyType, tt.identifier, tt.params)
			key2 := qcm.GenerateCacheKey(tt.keyType, tt.identifier, tt.params)
			
			assert.NotEmpty(t, key1)
			assert.NotEmpty(t, key2)
			
			if tt.expectSame {
				assert.Equal(t, key1, key2, "Keys should be identical for same parameters")
			}
			
			// Keys should start with qcache: prefix
			assert.True(t, len(key1) > 7 && key1[:7] == "qcache:", "Key should have qcache: prefix")
		})
	}
}

func TestQueryCacheManager_CacheOperations(t *testing.T) {
	cache := NewInMemoryCache(100, time.Minute)
	defer cache.Stop()
	
	monitor := NewInMemoryPerformanceMonitor(100*time.Millisecond, 10)
	config := DefaultQueryCacheConfig()
	
	qcm := NewQueryCacheManager(cache, monitor, config)
	ctx := context.Background()
	
	// Test cache miss
	cacheKey := qcm.GenerateCacheKey("test", "key1", nil)
	var result string
	hit, err := qcm.GetCachedResult(ctx, cacheKey, &result)
	assert.NoError(t, err)
	assert.False(t, hit)
	
	// Test cache set and hit
	testData := "test-value"
	err = qcm.SetCachedResult(ctx, cacheKey, testData, "test_query")
	assert.NoError(t, err)
	
	hit, err = qcm.GetCachedResult(ctx, cacheKey, &result)
	assert.NoError(t, err)
	assert.True(t, hit)
	assert.Equal(t, testData, result)
	
	// Test cache invalidation
	patterns := []string{cacheKey}
	err = qcm.InvalidateCachePatterns(ctx, patterns)
	assert.NoError(t, err)
	
	hit, err = qcm.GetCachedResult(ctx, cacheKey, &result)
	assert.NoError(t, err)
	assert.False(t, hit)
}

func TestQueryCacheManager_ExecuteWithCache(t *testing.T) {
	cache := NewInMemoryCache(100, time.Minute)
	defer cache.Stop()
	
	monitor := NewInMemoryPerformanceMonitor(100*time.Millisecond, 10)
	config := DefaultQueryCacheConfig()
	
	qcm := NewQueryCacheManager(cache, monitor, config)
	ctx := context.Background()
	
	callCount := 0
	testData := "test-result"
	
	queryFunc := func() (interface{}, error) {
		callCount++
		return testData, nil
	}
	
	cacheKey := qcm.GenerateCacheKey("test", "execute", nil)
	
	// First call should execute the function
	var result string
	err := qcm.ExecuteWithCache(ctx, cacheKey, "test_query", queryFunc, &result)
	assert.NoError(t, err)
	assert.Equal(t, testData, result)
	assert.Equal(t, 1, callCount)
	
	// Second call should use cache
	err = qcm.ExecuteWithCache(ctx, cacheKey, "test_query", queryFunc, &result)
	assert.NoError(t, err)
	assert.Equal(t, testData, result)
	assert.Equal(t, 1, callCount) // Should not increment
	
	// Check performance monitoring
	stats := monitor.GetQueryStats()
	assert.True(t, stats.TotalQueries > 0)
}

func TestCachedUnifiedChunkService_GetChunk(t *testing.T) {
	cache := NewInMemoryCache(100, time.Minute)
	defer cache.Stop()
	
	monitor := NewInMemoryPerformanceMonitor(100*time.Millisecond, 10)
	config := DefaultQueryCacheConfig()
	qcm := NewQueryCacheManager(cache, monitor, config)
	
	mockService := &MockUnifiedChunkService{}
	cachedService := NewCachedUnifiedChunkService(mockService, qcm)
	
	ctx := context.Background()
	chunkID := "test-chunk-id"
	
	expectedChunk := &models.UnifiedChunkRecord{
		ChunkID:  chunkID,
		Contents: "Test content",
		IsPage:   true,
	}
	
	// Mock the base service call
	mockService.On("GetChunk", ctx, chunkID).Return(expectedChunk, nil).Once()
	
	// First call should hit the base service
	result1, err := cachedService.GetChunk(ctx, chunkID)
	require.NoError(t, err)
	assert.Equal(t, expectedChunk.ChunkID, result1.ChunkID)
	assert.Equal(t, expectedChunk.Contents, result1.Contents)
	
	// Second call should use cache (mock should not be called again)
	result2, err := cachedService.GetChunk(ctx, chunkID)
	require.NoError(t, err)
	assert.Equal(t, expectedChunk.ChunkID, result2.ChunkID)
	assert.Equal(t, expectedChunk.Contents, result2.Contents)
	
	// Verify mock expectations
	mockService.AssertExpectations(t)
}

func TestCachedUnifiedChunkService_GetChunksByTag(t *testing.T) {
	cache := NewInMemoryCache(100, time.Minute)
	defer cache.Stop()
	
	monitor := NewInMemoryPerformanceMonitor(100*time.Millisecond, 10)
	config := DefaultQueryCacheConfig()
	qcm := NewQueryCacheManager(cache, monitor, config)
	
	mockService := &MockUnifiedChunkService{}
	cachedService := NewCachedUnifiedChunkService(mockService, qcm)
	
	ctx := context.Background()
	tagID := "test-tag-id"
	
	expectedChunks := []models.UnifiedChunkRecord{
		{ChunkID: "chunk1", Contents: "Content 1"},
		{ChunkID: "chunk2", Contents: "Content 2"},
	}
	
	// Mock the base service call
	mockService.On("GetChunksByTag", ctx, tagID).Return(expectedChunks, nil).Once()
	
	// First call should hit the base service
	result1, err := cachedService.GetChunksByTag(ctx, tagID)
	require.NoError(t, err)
	assert.Len(t, result1, 2)
	assert.Equal(t, expectedChunks[0].ChunkID, result1[0].ChunkID)
	
	// Second call should use cache
	result2, err := cachedService.GetChunksByTag(ctx, tagID)
	require.NoError(t, err)
	assert.Len(t, result2, 2)
	assert.Equal(t, expectedChunks[0].ChunkID, result2[0].ChunkID)
	
	// Verify mock expectations
	mockService.AssertExpectations(t)
}

func TestCachedUnifiedChunkService_SearchChunks(t *testing.T) {
	cache := NewInMemoryCache(100, time.Minute)
	defer cache.Stop()
	
	monitor := NewInMemoryPerformanceMonitor(100*time.Millisecond, 10)
	config := DefaultQueryCacheConfig()
	qcm := NewQueryCacheManager(cache, monitor, config)
	
	mockService := &MockUnifiedChunkService{}
	cachedService := NewCachedUnifiedChunkService(mockService, qcm)
	
	ctx := context.Background()
	query := &models.SearchQuery{
		Content: "test search",
		Limit:   10,
	}
	
	expectedResult := &models.SearchResult{
		Chunks: []models.UnifiedChunkRecord{
			{ChunkID: "chunk1", Contents: "Test content 1"},
		},
		TotalCount: 1,
		HasMore:    false,
		SearchTime: time.Millisecond * 50,
		CacheHit:   false,
	}
	
	// Mock the base service call
	mockService.On("SearchChunks", ctx, query).Return(expectedResult, nil).Once()
	
	// First call should hit the base service
	result1, err := cachedService.SearchChunks(ctx, query)
	require.NoError(t, err)
	assert.Len(t, result1.Chunks, 1)
	assert.Equal(t, expectedResult.Chunks[0].ChunkID, result1.Chunks[0].ChunkID)
	
	// Second call should use cache
	result2, err := cachedService.SearchChunks(ctx, query)
	require.NoError(t, err)
	assert.Len(t, result2.Chunks, 1)
	assert.True(t, result2.CacheHit) // Should be marked as cache hit
	
	// Verify mock expectations
	mockService.AssertExpectations(t)
}

func TestCachedUnifiedChunkService_CacheInvalidation(t *testing.T) {
	cache := NewInMemoryCache(100, time.Minute)
	defer cache.Stop()
	
	monitor := NewInMemoryPerformanceMonitor(100*time.Millisecond, 10)
	config := DefaultQueryCacheConfig()
	qcm := NewQueryCacheManager(cache, monitor, config)
	
	mockService := &MockUnifiedChunkService{}
	cachedService := NewCachedUnifiedChunkService(mockService, qcm)
	
	ctx := context.Background()
	chunkID := "test-chunk-id"
	
	chunk := &models.UnifiedChunkRecord{
		ChunkID:  chunkID,
		Contents: "Test content",
		Tags:     []string{"tag1", "tag2"},
	}
	
	expectedChunk := &models.UnifiedChunkRecord{
		ChunkID:  chunkID,
		Contents: "Test content",
		Tags:     []string{"tag1", "tag2"},
	}
	
	// Mock the base service calls - first call for initial cache, second call after invalidation
	mockService.On("GetChunk", ctx, chunkID).Return(expectedChunk, nil).Once()
	mockService.On("UpdateChunk", ctx, chunk).Return(nil).Once()
	mockService.On("GetChunk", ctx, chunkID).Return(expectedChunk, nil).Once()
	
	// First call should cache the result
	result1, err := cachedService.GetChunk(ctx, chunkID)
	require.NoError(t, err)
	assert.Equal(t, expectedChunk.ChunkID, result1.ChunkID)
	
	// Update the chunk (should invalidate cache)
	err = cachedService.UpdateChunk(ctx, chunk)
	require.NoError(t, err)
	
	// Next call should hit the base service again (cache was invalidated)
	result2, err := cachedService.GetChunk(ctx, chunkID)
	require.NoError(t, err)
	assert.Equal(t, expectedChunk.ChunkID, result2.ChunkID)
	
	// Verify mock expectations
	mockService.AssertExpectations(t)
}

func TestQueryCacheConfig_TTLSelection(t *testing.T) {
	cache := NewInMemoryCache(100, time.Minute)
	defer cache.Stop()
	
	monitor := NewInMemoryPerformanceMonitor(100*time.Millisecond, 10)
	config := &QueryCacheConfig{
		DefaultTTL:   5 * time.Minute,
		TagQueryTTL:  10 * time.Minute,
		HierarchyTTL: 15 * time.Minute,
		SearchTTL:    3 * time.Minute,
		Enabled:      true,
	}
	
	qcm := NewQueryCacheManager(cache, monitor, config)
	
	tests := []struct {
		queryType   string
		expectedTTL time.Duration
	}{
		{"get_chunk_tags", 10 * time.Minute},
		{"get_chunks_by_tag", 10 * time.Minute},
		{"get_children", 15 * time.Minute},
		{"get_ancestors", 15 * time.Minute},
		{"search_chunks", 3 * time.Minute},
		{"get_chunk", 5 * time.Minute},
	}
	
	for _, tt := range tests {
		t.Run(tt.queryType, func(t *testing.T) {
			ttl := qcm.getTTLForQueryType(tt.queryType)
			assert.Equal(t, tt.expectedTTL, ttl)
		})
	}
}

func BenchmarkQueryCacheManager_GenerateCacheKey(b *testing.B) {
	cache := NewInMemoryCache(100, time.Minute)
	defer cache.Stop()
	
	monitor := NewInMemoryPerformanceMonitor(100*time.Millisecond, 10)
	config := DefaultQueryCacheConfig()
	qcm := NewQueryCacheManager(cache, monitor, config)
	
	params := map[string]interface{}{
		"content":   "test search content",
		"tags":      []string{"tag1", "tag2", "tag3"},
		"limit":     100,
		"offset":    0,
		"is_page":   true,
		"metadata":  map[string]interface{}{"key": "value"},
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		qcm.GenerateCacheKey("search_chunks", "", params)
	}
}

func BenchmarkCachedUnifiedChunkService_GetChunk(b *testing.B) {
	cache := NewInMemoryCache(1000, time.Minute)
	defer cache.Stop()
	
	monitor := NewInMemoryPerformanceMonitor(100*time.Millisecond, 10)
	config := DefaultQueryCacheConfig()
	qcm := NewQueryCacheManager(cache, monitor, config)
	
	mockService := &MockUnifiedChunkService{}
	cachedService := NewCachedUnifiedChunkService(mockService, qcm)
	
	ctx := context.Background()
	
	// Pre-populate cache with test data
	for i := 0; i < 100; i++ {
		chunkID := fmt.Sprintf("chunk-%d", i)
		chunk := &models.UnifiedChunkRecord{
			ChunkID:  chunkID,
			Contents: fmt.Sprintf("Content %d", i),
		}
		mockService.On("GetChunk", ctx, chunkID).Return(chunk, nil).Once()
		
		// Prime the cache
		cachedService.GetChunk(ctx, chunkID)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		chunkID := fmt.Sprintf("chunk-%d", i%100)
		cachedService.GetChunk(ctx, chunkID)
	}
}