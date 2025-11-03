package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"semantic-text-processor/models"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockSearchService for testing
type MockSearchService struct {
	mock.Mock
}

func (m *MockSearchService) SemanticSearch(ctx context.Context, query string, limit int) ([]models.SimilarityResult, error) {
	args := m.Called(ctx, query, limit)
	return args.Get(0).([]models.SimilarityResult), args.Error(1)
}

func (m *MockSearchService) SemanticSearchWithFilters(ctx context.Context, req *models.SemanticSearchRequest) (*models.SemanticSearchResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*models.SemanticSearchResponse), args.Error(1)
}

func (m *MockSearchService) HybridSearch(ctx context.Context, query string, limit int, semanticWeight float64) ([]models.SimilarityResult, error) {
	args := m.Called(ctx, query, limit, semanticWeight)
	return args.Get(0).([]models.SimilarityResult), args.Error(1)
}

func (m *MockSearchService) GraphSearch(ctx context.Context, query *models.GraphQuery) (*models.GraphResult, error) {
	args := m.Called(ctx, query)
	return args.Get(0).(*models.GraphResult), args.Error(1)
}

func (m *MockSearchService) SearchByTag(ctx context.Context, tagContent string) ([]models.ChunkWithTags, error) {
	args := m.Called(ctx, tagContent)
	return args.Get(0).([]models.ChunkWithTags), args.Error(1)
}

func (m *MockSearchService) SearchChunks(ctx context.Context, query string, filters map[string]interface{}) ([]models.ChunkRecord, error) {
	args := m.Called(ctx, query, filters)
	return args.Get(0).([]models.ChunkRecord), args.Error(1)
}

func TestSearchHandler_SemanticSearch(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		mockResponse   *models.SemanticSearchResponse
		mockError      error
		expectedStatus int
		expectedError  string
	}{
		{
			name: "successful semantic search",
			requestBody: models.SemanticSearchRequest{
				Query: "test query",
				Limit: 5,
			},
			mockResponse: &models.SemanticSearchResponse{
				Results: []models.SimilarityResult{
					{
						Chunk: models.ChunkRecord{
							ID:      "chunk-1",
							Content: "test content",
						},
						Similarity: 0.95,
					},
				},
				TotalCount: 1,
				Query:      "test query",
				Limit:      5,
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "empty query",
			requestBody:    models.SemanticSearchRequest{Query: ""},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "query is required",
		},
		{
			name: "invalid request body",
			requestBody: map[string]interface{}{
				"invalid": "data",
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockService := new(MockSearchService)
			handler := NewSearchHandler(mockService)

			// Setup mock expectations
			if tt.mockResponse != nil {
				mockService.On("SemanticSearchWithFilters", mock.Anything, mock.AnythingOfType("*models.SemanticSearchRequest")).
					Return(tt.mockResponse, tt.mockError)
			}

			// Create request
			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest("POST", "/api/v1/search/semantic", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			
			// Create response recorder
			rr := httptest.NewRecorder()

			// Execute
			handler.SemanticSearch(rr, req)

			// Assert
			assert.Equal(t, tt.expectedStatus, rr.Code)

			if tt.expectedStatus == http.StatusOK {
				var response models.SemanticSearchResponse
				err := json.Unmarshal(rr.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, tt.mockResponse.Query, response.Query)
				assert.Equal(t, tt.mockResponse.TotalCount, response.TotalCount)
			}

			if tt.expectedError != "" {
				assert.Contains(t, rr.Body.String(), tt.expectedError)
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestSearchHandler_GraphSearch(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		mockResponse   *models.GraphResult
		mockError      error
		expectedStatus int
	}{
		{
			name: "successful graph search",
			requestBody: models.GraphQuery{
				EntityName: "test entity",
				MaxDepth:   2,
				Limit:      10,
			},
			mockResponse: &models.GraphResult{
				Nodes: []models.GraphNode{
					{
						ID:         "node-1",
						EntityName: "test entity",
						EntityType: "person",
					},
				},
				Edges: []models.GraphEdge{},
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockService := new(MockSearchService)
			handler := NewSearchHandler(mockService)

			// Setup mock expectations
			if tt.mockResponse != nil {
				mockService.On("GraphSearch", mock.Anything, mock.AnythingOfType("*models.GraphQuery")).
					Return(tt.mockResponse, tt.mockError)
			}

			// Create request
			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest("POST", "/api/v1/search/graph", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			
			// Create response recorder
			rr := httptest.NewRecorder()

			// Execute
			handler.GraphSearch(rr, req)

			// Assert
			assert.Equal(t, tt.expectedStatus, rr.Code)

			if tt.expectedStatus == http.StatusOK {
				var response models.GraphResult
				err := json.Unmarshal(rr.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, len(tt.mockResponse.Nodes), len(response.Nodes))
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestSearchHandler_SearchByTag(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		mockResponse   []models.ChunkWithTags
		mockError      error
		expectedStatus int
		expectedError  string
	}{
		{
			name: "successful tag search",
			requestBody: models.TagSearchRequest{
				TagContent: "important",
			},
			mockResponse: []models.ChunkWithTags{
				{
					Chunk: &models.ChunkRecord{
						ID:      "chunk-1",
						Content: "tagged content",
					},
					Tags: []models.ChunkRecord{
						{
							ID:      "tag-1",
							Content: "important",
						},
					},
				},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "empty tag content",
			requestBody:    models.TagSearchRequest{TagContent: ""},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "tag content is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockService := new(MockSearchService)
			handler := NewSearchHandler(mockService)

			// Setup mock expectations
			if tt.mockResponse != nil {
				mockService.On("SearchByTag", mock.Anything, mock.AnythingOfType("string")).
					Return(tt.mockResponse, tt.mockError)
			}

			// Create request
			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest("POST", "/api/v1/search/tags", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			
			// Create response recorder
			rr := httptest.NewRecorder()

			// Execute
			handler.SearchByTag(rr, req)

			// Assert
			assert.Equal(t, tt.expectedStatus, rr.Code)

			if tt.expectedStatus == http.StatusOK {
				var response []models.ChunkWithTags
				err := json.Unmarshal(rr.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, len(tt.mockResponse), len(response))
			}

			if tt.expectedError != "" {
				assert.Contains(t, rr.Body.String(), tt.expectedError)
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestSearchHandler_SearchChunks(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		mockResponse   []models.ChunkRecord
		mockError      error
		expectedStatus int
		expectedError  string
	}{
		{
			name: "successful chunk search",
			requestBody: models.ChunkSearchRequest{
				Query: "search term",
				Filters: map[string]interface{}{
					"text_id": "text-123",
				},
			},
			mockResponse: []models.ChunkRecord{
				{
					ID:      "chunk-1",
					Content: "content with search term",
					TextID:  "text-123",
				},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "empty query",
			requestBody:    models.ChunkSearchRequest{Query: ""},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "query is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockService := new(MockSearchService)
			handler := NewSearchHandler(mockService)

			// Setup mock expectations
			if tt.mockResponse != nil {
				mockService.On("SearchChunks", mock.Anything, mock.AnythingOfType("string"), mock.Anything).
					Return(tt.mockResponse, tt.mockError)
			}

			// Create request
			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest("POST", "/api/v1/search/chunks", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			
			// Create response recorder
			rr := httptest.NewRecorder()

			// Execute
			handler.SearchChunks(rr, req)

			// Assert
			assert.Equal(t, tt.expectedStatus, rr.Code)

			if tt.expectedStatus == http.StatusOK {
				var response []models.ChunkRecord
				err := json.Unmarshal(rr.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, len(tt.mockResponse), len(response))
			}

			if tt.expectedError != "" {
				assert.Contains(t, rr.Body.String(), tt.expectedError)
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestSearchHandler_HybridSearch(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		mockResponse   []models.SimilarityResult
		mockError      error
		expectedStatus int
		expectedError  string
	}{
		{
			name: "successful hybrid search",
			requestBody: models.HybridSearchRequest{
				Query:          "hybrid query",
				Limit:          5,
				SemanticWeight: 0.7,
			},
			mockResponse: []models.SimilarityResult{
				{
					Chunk: models.ChunkRecord{
						ID:      "chunk-1",
						Content: "hybrid content",
					},
					Similarity: 0.85,
				},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "empty query",
			requestBody:    models.HybridSearchRequest{Query: ""},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "query is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockService := new(MockSearchService)
			handler := NewSearchHandler(mockService)

			// Setup mock expectations
			if tt.mockResponse != nil {
				mockService.On("HybridSearch", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("int"), mock.AnythingOfType("float64")).
					Return(tt.mockResponse, tt.mockError)
			}

			// Create request
			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest("POST", "/api/v1/search/hybrid", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			
			// Create response recorder
			rr := httptest.NewRecorder()

			// Execute
			handler.HybridSearch(rr, req)

			// Assert
			assert.Equal(t, tt.expectedStatus, rr.Code)

			if tt.expectedStatus == http.StatusOK {
				var response []models.SimilarityResult
				err := json.Unmarshal(rr.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, len(tt.mockResponse), len(response))
			}

			if tt.expectedError != "" {
				assert.Contains(t, rr.Body.String(), tt.expectedError)
			}

			mockService.AssertExpectations(t)
		})
	}
}