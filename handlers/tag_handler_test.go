package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"semantic-text-processor/models"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockTagService for testing
type MockTagService struct {
	mock.Mock
}

func (m *MockTagService) AddTag(ctx context.Context, chunkID string, tagContent string) error {
	args := m.Called(ctx, chunkID, tagContent)
	return args.Error(0)
}

func (m *MockTagService) RemoveTag(ctx context.Context, chunkID string, tagChunkID string) error {
	args := m.Called(ctx, chunkID, tagChunkID)
	return args.Error(0)
}

func (m *MockTagService) GetChunkTags(ctx context.Context, chunkID string) ([]models.ChunkRecord, error) {
	args := m.Called(ctx, chunkID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.ChunkRecord), args.Error(1)
}

func (m *MockTagService) GetChunksByTag(ctx context.Context, tagContent string) ([]models.ChunkRecord, error) {
	args := m.Called(ctx, tagContent)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.ChunkRecord), args.Error(1)
}

func (m *MockTagService) AddTagWithInheritance(ctx context.Context, chunkID string, tagContent string) error {
	args := m.Called(ctx, chunkID, tagContent)
	return args.Error(0)
}

func (m *MockTagService) RemoveTagWithInheritance(ctx context.Context, chunkID string, tagChunkID string) error {
	args := m.Called(ctx, chunkID, tagChunkID)
	return args.Error(0)
}

func (m *MockTagService) GetInheritedTags(ctx context.Context, chunkID string) ([]models.ChunkRecord, error) {
	args := m.Called(ctx, chunkID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.ChunkRecord), args.Error(1)
}

func TestTagHandler_AddTag(t *testing.T) {
	tests := []struct {
		name           string
		chunkID        string
		requestBody    models.AddTagRequest
		mockError      error
		expectedStatus int
	}{
		{
			name:    "successful tag addition",
			chunkID: "chunk-123",
			requestBody: models.AddTagRequest{
				TagContent: "important",
			},
			mockError:      nil,
			expectedStatus: http.StatusCreated,
		},
		{
			name:    "empty chunk ID",
			chunkID: "",
			requestBody: models.AddTagRequest{
				TagContent: "important",
			},
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:    "empty tag content",
			chunkID: "chunk-123",
			requestBody: models.AddTagRequest{
				TagContent: "",
			},
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:    "service error",
			chunkID: "chunk-123",
			requestBody: models.AddTagRequest{
				TagContent: "important",
			},
			mockError:      fmt.Errorf("service error"),
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockService := new(MockTagService)
			handler := NewTagHandler(mockService)

			// Setup mock expectations
			if tt.chunkID != "" && tt.requestBody.TagContent != "" {
				mockService.On("AddTag", mock.Anything, tt.chunkID, tt.requestBody.TagContent).Return(tt.mockError)
			}

			// Create request
			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest("POST", "/api/v1/chunks/"+tt.chunkID+"/tags", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			// Setup mux vars
			req = mux.SetURLVars(req, map[string]string{"id": tt.chunkID})

			// Execute
			handler.AddTag(w, req)

			// Assert
			assert.Equal(t, tt.expectedStatus, w.Code)

			mockService.AssertExpectations(t)
		})
	}
}

func TestTagHandler_RemoveTag(t *testing.T) {
	tests := []struct {
		name           string
		chunkID        string
		tagID          string
		mockError      error
		expectedStatus int
	}{
		{
			name:           "successful tag removal",
			chunkID:        "chunk-123",
			tagID:          "tag-456",
			mockError:      nil,
			expectedStatus: http.StatusNoContent,
		},
		{
			name:           "empty chunk ID",
			chunkID:        "",
			tagID:          "tag-456",
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "empty tag ID",
			chunkID:        "chunk-123",
			tagID:          "",
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "service error",
			chunkID:        "chunk-123",
			tagID:          "tag-456",
			mockError:      fmt.Errorf("service error"),
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockService := new(MockTagService)
			handler := NewTagHandler(mockService)

			// Setup mock expectations
			if tt.chunkID != "" && tt.tagID != "" {
				mockService.On("RemoveTag", mock.Anything, tt.chunkID, tt.tagID).Return(tt.mockError)
			}

			// Create request
			req := httptest.NewRequest("DELETE", "/api/v1/chunks/"+tt.chunkID+"/tags/"+tt.tagID, nil)
			w := httptest.NewRecorder()

			// Setup mux vars
			req = mux.SetURLVars(req, map[string]string{
				"id":    tt.chunkID,
				"tagId": tt.tagID,
			})

			// Execute
			handler.RemoveTag(w, req)

			// Assert
			assert.Equal(t, tt.expectedStatus, w.Code)

			mockService.AssertExpectations(t)
		})
	}
}

func TestTagHandler_GetChunkTags(t *testing.T) {
	tests := []struct {
		name           string
		chunkID        string
		mockResponse   []models.ChunkRecord
		mockError      error
		expectedStatus int
		expectedCount  int
	}{
		{
			name:    "successful get chunk tags",
			chunkID: "chunk-123",
			mockResponse: []models.ChunkRecord{
				{
					ID:        "tag-1",
					Content:   "important",
					CreatedAt: time.Now(),
				},
				{
					ID:        "tag-2",
					Content:   "urgent",
					CreatedAt: time.Now(),
				},
			},
			mockError:      nil,
			expectedStatus: http.StatusOK,
			expectedCount:  2,
		},
		{
			name:           "empty chunk ID",
			chunkID:        "",
			mockResponse:   nil,
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
			expectedCount:  0,
		},
		{
			name:           "no tags found",
			chunkID:        "chunk-123",
			mockResponse:   []models.ChunkRecord{},
			mockError:      nil,
			expectedStatus: http.StatusOK,
			expectedCount:  0,
		},
		{
			name:           "service error",
			chunkID:        "chunk-123",
			mockResponse:   nil,
			mockError:      fmt.Errorf("service error"),
			expectedStatus: http.StatusInternalServerError,
			expectedCount:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockService := new(MockTagService)
			handler := NewTagHandler(mockService)

			// Setup mock expectations
			if tt.chunkID != "" {
				mockService.On("GetChunkTags", mock.Anything, tt.chunkID).Return(tt.mockResponse, tt.mockError)
			}

			// Create request
			req := httptest.NewRequest("GET", "/api/v1/chunks/"+tt.chunkID+"/tags", nil)
			w := httptest.NewRecorder()

			// Setup mux vars
			req = mux.SetURLVars(req, map[string]string{"id": tt.chunkID})

			// Execute
			handler.GetChunkTags(w, req)

			// Assert
			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusOK {
				var response []models.ChunkRecord
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedCount, len(response))
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestTagHandler_GetChunksByTag(t *testing.T) {
	tests := []struct {
		name           string
		tagContent     string
		mockResponse   []models.ChunkRecord
		mockError      error
		expectedStatus int
		expectedCount  int
	}{
		{
			name:       "successful get chunks by tag",
			tagContent: "important",
			mockResponse: []models.ChunkRecord{
				{
					ID:        "chunk-1",
					Content:   "Important chunk 1",
					CreatedAt: time.Now(),
				},
				{
					ID:        "chunk-2",
					Content:   "Important chunk 2",
					CreatedAt: time.Now(),
				},
			},
			mockError:      nil,
			expectedStatus: http.StatusOK,
			expectedCount:  2,
		},
		{
			name:           "empty tag content",
			tagContent:     "",
			mockResponse:   nil,
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
			expectedCount:  0,
		},
		{
			name:           "no chunks found",
			tagContent:     "nonexistent",
			mockResponse:   []models.ChunkRecord{},
			mockError:      nil,
			expectedStatus: http.StatusOK,
			expectedCount:  0,
		},
		{
			name:           "service error",
			tagContent:     "important",
			mockResponse:   nil,
			mockError:      fmt.Errorf("service error"),
			expectedStatus: http.StatusInternalServerError,
			expectedCount:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockService := new(MockTagService)
			handler := NewTagHandler(mockService)

			// Setup mock expectations
			if tt.tagContent != "" {
				mockService.On("GetChunksByTag", mock.Anything, tt.tagContent).Return(tt.mockResponse, tt.mockError)
			}

			// Create request
			req := httptest.NewRequest("GET", "/api/v1/tags/"+tt.tagContent+"/chunks", nil)
			w := httptest.NewRecorder()

			// Setup mux vars
			req = mux.SetURLVars(req, map[string]string{"content": tt.tagContent})

			// Execute
			handler.GetChunksByTag(w, req)

			// Assert
			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusOK {
				var response []models.ChunkRecord
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedCount, len(response))
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestTagHandler_AddTagWithInheritance(t *testing.T) {
	tests := []struct {
		name           string
		chunkID        string
		requestBody    models.AddTagRequest
		mockError      error
		expectedStatus int
	}{
		{
			name:    "successful tag addition with inheritance",
			chunkID: "chunk-123",
			requestBody: models.AddTagRequest{
				TagContent: "important",
			},
			mockError:      nil,
			expectedStatus: http.StatusCreated,
		},
		{
			name:    "empty chunk ID",
			chunkID: "",
			requestBody: models.AddTagRequest{
				TagContent: "important",
			},
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:    "empty tag content",
			chunkID: "chunk-123",
			requestBody: models.AddTagRequest{
				TagContent: "",
			},
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:    "service error",
			chunkID: "chunk-123",
			requestBody: models.AddTagRequest{
				TagContent: "important",
			},
			mockError:      fmt.Errorf("service error"),
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockService := new(MockTagService)
			handler := NewTagHandler(mockService)

			// Setup mock expectations
			if tt.chunkID != "" && tt.requestBody.TagContent != "" {
				mockService.On("AddTagWithInheritance", mock.Anything, tt.chunkID, tt.requestBody.TagContent).Return(tt.mockError)
			}

			// Create request
			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest("POST", "/api/v1/chunks/"+tt.chunkID+"/tags/inherit", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			// Setup mux vars
			req = mux.SetURLVars(req, map[string]string{"id": tt.chunkID})

			// Execute
			handler.AddTagWithInheritance(w, req)

			// Assert
			assert.Equal(t, tt.expectedStatus, w.Code)

			mockService.AssertExpectations(t)
		})
	}
}

func TestTagHandler_RemoveTagWithInheritance(t *testing.T) {
	tests := []struct {
		name           string
		chunkID        string
		tagID          string
		mockError      error
		expectedStatus int
	}{
		{
			name:           "successful tag removal with inheritance",
			chunkID:        "chunk-123",
			tagID:          "tag-456",
			mockError:      nil,
			expectedStatus: http.StatusNoContent,
		},
		{
			name:           "empty chunk ID",
			chunkID:        "",
			tagID:          "tag-456",
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "empty tag ID",
			chunkID:        "chunk-123",
			tagID:          "",
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "service error",
			chunkID:        "chunk-123",
			tagID:          "tag-456",
			mockError:      fmt.Errorf("service error"),
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockService := new(MockTagService)
			handler := NewTagHandler(mockService)

			// Setup mock expectations
			if tt.chunkID != "" && tt.tagID != "" {
				mockService.On("RemoveTagWithInheritance", mock.Anything, tt.chunkID, tt.tagID).Return(tt.mockError)
			}

			// Create request
			req := httptest.NewRequest("DELETE", "/api/v1/chunks/"+tt.chunkID+"/tags/"+tt.tagID+"/inherit", nil)
			w := httptest.NewRecorder()

			// Setup mux vars
			req = mux.SetURLVars(req, map[string]string{
				"id":    tt.chunkID,
				"tagId": tt.tagID,
			})

			// Execute
			handler.RemoveTagWithInheritance(w, req)

			// Assert
			assert.Equal(t, tt.expectedStatus, w.Code)

			mockService.AssertExpectations(t)
		})
	}
}

func TestTagHandler_GetInheritedTags(t *testing.T) {
	tests := []struct {
		name           string
		chunkID        string
		mockResponse   []models.ChunkRecord
		mockError      error
		expectedStatus int
		expectedCount  int
	}{
		{
			name:    "successful get inherited tags",
			chunkID: "chunk-123",
			mockResponse: []models.ChunkRecord{
				{
					ID:        "tag-1",
					Content:   "direct-tag",
					CreatedAt: time.Now(),
				},
				{
					ID:        "tag-2",
					Content:   "inherited-tag",
					CreatedAt: time.Now(),
				},
			},
			mockError:      nil,
			expectedStatus: http.StatusOK,
			expectedCount:  2,
		},
		{
			name:           "empty chunk ID",
			chunkID:        "",
			mockResponse:   nil,
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
			expectedCount:  0,
		},
		{
			name:           "no tags found",
			chunkID:        "chunk-123",
			mockResponse:   []models.ChunkRecord{},
			mockError:      nil,
			expectedStatus: http.StatusOK,
			expectedCount:  0,
		},
		{
			name:           "service error",
			chunkID:        "chunk-123",
			mockResponse:   nil,
			mockError:      fmt.Errorf("service error"),
			expectedStatus: http.StatusInternalServerError,
			expectedCount:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockService := new(MockTagService)
			handler := NewTagHandler(mockService)

			// Setup mock expectations
			if tt.chunkID != "" {
				mockService.On("GetInheritedTags", mock.Anything, tt.chunkID).Return(tt.mockResponse, tt.mockError)
			}

			// Create request
			req := httptest.NewRequest("GET", "/api/v1/chunks/"+tt.chunkID+"/tags/inherited", nil)
			w := httptest.NewRecorder()

			// Setup mux vars
			req = mux.SetURLVars(req, map[string]string{"id": tt.chunkID})

			// Execute
			handler.GetInheritedTags(w, req)

			// Assert
			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusOK {
				var response []models.ChunkRecord
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedCount, len(response))
			}

			mockService.AssertExpectations(t)
		})
	}
}