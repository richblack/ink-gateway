package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"semantic-text-processor/models"
	"semantic-text-processor/services"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockUnifiedChunkService mocks the UnifiedChunkService interface
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

// MockCacheService mocks the CacheService interface
type MockCacheService struct {
	mock.Mock
}

func (m *MockCacheService) Get(ctx context.Context, key string) (interface{}, bool) {
	args := m.Called(ctx, key)
	return args.Get(0), args.Bool(1)
}

func (m *MockCacheService) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	args := m.Called(ctx, key, value, ttl)
	return args.Error(0)
}

func (m *MockCacheService) Delete(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func (m *MockCacheService) Clear(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockCacheService) GetStats() services.CacheStats {
	args := m.Called()
	return args.Get(0).(services.CacheStats)
}

// Test Functions

func TestUnifiedChunkHandler_CreateChunk(t *testing.T) {
	mockService := new(MockUnifiedChunkService)
	mockCache := new(MockCacheService)
	logger := log.New(bytes.NewBuffer(nil), "", 0)

	handler := NewUnifiedChunkHandler(mockService, mockCache, logger, 100*time.Millisecond, true)

	// Setup mock expectations
	mockService.On("CreateChunk", mock.Anything, mock.AnythingOfType("*models.UnifiedChunkRecord")).Return(nil)

	// Create test request
	reqBody := models.CreateChunkRequest{
		Content: "Test chunk content",
		TextID:  "test-text-id",
	}
	jsonBody, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/v1/chunks", bytes.NewBuffer(jsonBody))
	w := httptest.NewRecorder()

	// Execute request
	handler.CreateChunk(w, req)

	// Verify response
	assert.Equal(t, http.StatusCreated, w.Code)

	// Verify mock calls
	mockService.AssertExpectations(t)
}

func TestUnifiedChunkHandler_GetChunkByID_WithCache(t *testing.T) {
	mockService := new(MockUnifiedChunkService)
	mockCache := new(MockCacheService)
	logger := log.New(bytes.NewBuffer(nil), "", 0)

	handler := NewUnifiedChunkHandler(mockService, mockCache, logger, 100*time.Millisecond, true)

	// Setup test data
	expectedChunk := &models.UnifiedChunkRecord{
		ChunkID:  "test-chunk-id",
		Contents: "Test content",
	}

	// Setup mock expectations for cache hit
	mockCache.On("Get", mock.Anything, "chunk:test-chunk-id").Return(expectedChunk, true)

	// Create test request with mux vars
	req := httptest.NewRequest("GET", "/api/v1/chunks/test-chunk-id", nil)
	w := httptest.NewRecorder()

	// Set up the mux context
	req = mux.SetURLVars(req, map[string]string{"id": "test-chunk-id"})

	// Execute request
	handler.GetChunkByID(w, req)

	// Verify response
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "HIT", w.Header().Get("X-Cache"))

	// Verify mock calls
	mockCache.AssertExpectations(t)
	// Service should NOT be called due to cache hit
	mockService.AssertNotCalled(t, "GetChunk")
}

func TestUnifiedChunkHandler_GetChunkByID_CacheMiss(t *testing.T) {
	mockService := new(MockUnifiedChunkService)
	mockCache := new(MockCacheService)
	logger := log.New(bytes.NewBuffer(nil), "", 0)

	handler := NewUnifiedChunkHandler(mockService, mockCache, logger, 100*time.Millisecond, true)

	// Setup test data
	expectedChunk := &models.UnifiedChunkRecord{
		ChunkID:  "test-chunk-id",
		Contents: "Test content",
	}

	// Setup mock expectations for cache miss
	mockCache.On("Get", mock.Anything, "chunk:test-chunk-id").Return(nil, false)
	mockService.On("GetChunk", mock.Anything, "test-chunk-id").Return(expectedChunk, nil)
	mockCache.On("Set", mock.Anything, "chunk:test-chunk-id", expectedChunk, 15*time.Minute).Return(nil)

	// Create test request with mux vars
	req := httptest.NewRequest("GET", "/api/v1/chunks/test-chunk-id", nil)
	w := httptest.NewRecorder()

	// Set up the mux context
	req = mux.SetURLVars(req, map[string]string{"id": "test-chunk-id"})

	// Execute request
	handler.GetChunkByID(w, req)

	// Verify response
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "MISS", w.Header().Get("X-Cache"))

	// Verify mock calls
	mockService.AssertExpectations(t)
	mockCache.AssertExpectations(t)
}

func TestUnifiedChunkHandler_BatchCreateChunks(t *testing.T) {
	mockService := new(MockUnifiedChunkService)
	mockCache := new(MockCacheService)
	logger := log.New(bytes.NewBuffer(nil), "", 0)

	handler := NewUnifiedChunkHandler(mockService, mockCache, logger, 100*time.Millisecond, true)

	// Setup test data
	reqBody := models.BatchCreateRequest{
		Chunks: []models.UnifiedChunkRecord{
			{Contents: "Chunk 1"},
			{Contents: "Chunk 2"},
		},
	}

	// Setup mock expectations
	mockService.On("BatchCreateChunks", mock.Anything, mock.AnythingOfType("[]models.UnifiedChunkRecord")).Return(nil)

	jsonBody, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/v1/chunks/batch", bytes.NewBuffer(jsonBody))
	w := httptest.NewRecorder()

	// Execute request
	handler.BatchCreateChunks(w, req)

	// Verify response
	assert.Equal(t, http.StatusCreated, w.Code)

	// Parse response body
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, float64(2), response["created_count"])

	// Verify mock calls
	mockService.AssertExpectations(t)
}

func TestUnifiedTagHandler_AddTag(t *testing.T) {
	mockService := new(MockUnifiedChunkService)
	mockCache := new(MockCacheService)
	logger := log.New(bytes.NewBuffer(nil), "", 0)

	handler := NewUnifiedTagHandler(mockService, mockCache, logger, 100*time.Millisecond, true)

	// Setup mock expectations for finding or creating tag
	searchResult := &models.SearchResult{
		Chunks: []models.UnifiedChunkRecord{{ChunkID: "tag-chunk-id", IsTag: true}},
	}
	mockService.On("SearchChunks", mock.Anything, mock.AnythingOfType("*models.SearchQuery")).Return(searchResult, nil)
	mockService.On("AddTags", mock.Anything, "test-chunk-id", []string{"tag-chunk-id"}).Return(nil)

	// Setup cache expectations
	mockCache.On("Delete", mock.Anything, "chunk_tags:test-chunk-id").Return(nil)
	mockCache.On("Delete", mock.Anything, "tag_chunks:tag-chunk-id").Return(nil)

	// Create test request
	reqBody := models.AddTagRequest{
		TagContent: "test-tag",
	}
	jsonBody, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/v1/chunks/test-chunk-id/tags", bytes.NewBuffer(jsonBody))
	w := httptest.NewRecorder()

	// Set up the mux context
	req = mux.SetURLVars(req, map[string]string{"id": "test-chunk-id"})

	// Execute request
	handler.AddTag(w, req)

	// Verify response
	assert.Equal(t, http.StatusCreated, w.Code)

	// Verify mock calls
	mockService.AssertExpectations(t)
	mockCache.AssertExpectations(t)
}

func TestUnifiedTagHandler_BatchTagOperations(t *testing.T) {
	mockService := new(MockUnifiedChunkService)
	mockCache := new(MockCacheService)
	logger := log.New(bytes.NewBuffer(nil), "", 0)

	handler := NewUnifiedTagHandler(mockService, mockCache, logger, 100*time.Millisecond, true)

	// Setup test data
	reqBody := BatchAddTagsRequest{
		Operations: []TagOperation{
			{ChunkID: "chunk1", TagIDs: []string{"tag1"}, Operation: "add"},
			{ChunkID: "chunk2", TagIDs: []string{"tag1"}, Operation: "remove"},
		},
	}

	// Setup mock expectations
	mockService.On("AddTags", mock.Anything, "chunk1", []string{"tag1"}).Return(nil)
	mockService.On("RemoveTags", mock.Anything, "chunk2", []string{"tag1"}).Return(nil)

	// Setup cache expectations
	mockCache.On("Delete", mock.Anything, "chunk_tags:chunk1").Return(nil)
	mockCache.On("Delete", mock.Anything, "chunk_tags:chunk2").Return(nil)

	jsonBody, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/v1/chunks/tags/batch", bytes.NewBuffer(jsonBody))
	w := httptest.NewRecorder()

	// Execute request
	handler.BatchTagOperations(w, req)

	// Verify response
	assert.Equal(t, http.StatusOK, w.Code)

	// Parse response body
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, float64(2), response["processed_count"])

	// Verify mock calls
	mockService.AssertExpectations(t)
	mockCache.AssertExpectations(t)
}

func TestPerformanceMonitor_SlowQuery(t *testing.T) {
	var logBuffer bytes.Buffer
	logger := log.New(&logBuffer, "", 0)

	monitor := NewPerformanceMonitor(50*time.Millisecond, logger, true)

	// Execute a slow operation
	err := monitor.MonitoredOperation("test_slow_op", func() error {
		time.Sleep(100 * time.Millisecond) // Slower than threshold
		return nil
	})

	assert.NoError(t, err)

	// Verify slow query was logged
	logOutput := logBuffer.String()
	assert.Contains(t, logOutput, "SLOW_OPERATION")
	assert.Contains(t, logOutput, "test_slow_op")
}

func TestModelConverter_ToUnifiedChunk(t *testing.T) {
	converter := NewModelConverter()

	legacyChunk := &models.ChunkRecord{
		ID:         "test-id",
		Content:    "test content",
		TextID:     "text-id",
		IsTemplate: true,
		IsSlot:     false,
	}

	unified := converter.ToUnifiedChunk(legacyChunk)

	assert.Equal(t, "test-id", unified.ChunkID)
	assert.Equal(t, "test content", unified.Contents)
	assert.Equal(t, "text-id", *unified.Page)
	assert.True(t, unified.IsTemplate)
	assert.False(t, unified.IsSlot)
	assert.False(t, unified.IsPage)
	assert.False(t, unified.IsTag)
}

func TestModelConverter_FromUnifiedChunk(t *testing.T) {
	converter := NewModelConverter()

	textID := "text-id"
	unifiedChunk := &models.UnifiedChunkRecord{
		ChunkID:    "test-id",
		Contents:   "test content",
		Page:       &textID,
		IsTemplate: true,
		IsSlot:     false,
		IsPage:     false,
		IsTag:      false,
	}

	legacy := converter.FromUnifiedChunk(unifiedChunk)

	assert.Equal(t, "test-id", legacy.ID)
	assert.Equal(t, "test content", legacy.Content)
	assert.Equal(t, "text-id", legacy.TextID)
	assert.True(t, legacy.IsTemplate)
	assert.False(t, legacy.IsSlot)
}