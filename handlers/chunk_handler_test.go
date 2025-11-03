package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"semantic-text-processor/models"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)



func TestChunkHandler_GetChunks(t *testing.T) {
	tests := []struct {
		name           string
		queryParams    string
		mockChunks     []models.ChunkRecord
		mockError      error
		expectedStatus int
		expectedCount  int
	}{
		{
			name:        "successful get chunks",
			queryParams: "",
			mockChunks: []models.ChunkRecord{
				{ID: "chunk1", Content: "Test chunk 1"},
				{ID: "chunk2", Content: "Test chunk 2"},
			},
			mockError:      nil,
			expectedStatus: http.StatusOK,
			expectedCount:  2,
		},
		{
			name:        "get chunks with filters",
			queryParams: "?text_id=text123&is_template=true",
			mockChunks: []models.ChunkRecord{
				{ID: "chunk1", Content: "Template chunk", IsTemplate: true},
			},
			mockError:      nil,
			expectedStatus: http.StatusOK,
			expectedCount:  1,
		},
		{
			name:           "database error",
			queryParams:    "",
			mockChunks:     nil,
			mockError:      fmt.Errorf("database error"),
			expectedStatus: http.StatusInternalServerError,
			expectedCount:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := new(MockSupabaseClient)
			handler := NewChunkHandler(mockClient)

			// Setup mock expectations
			mockClient.On("SearchChunks", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("map[string]interface {}")).
				Return(tt.mockChunks, tt.mockError)

			// Create request
			req := httptest.NewRequest("GET", "/api/v1/chunks"+tt.queryParams, nil)
			w := httptest.NewRecorder()

			// Execute
			handler.GetChunks(w, req)

			// Assert
			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusOK {
				var chunks []models.ChunkRecord
				err := json.Unmarshal(w.Body.Bytes(), &chunks)
				assert.NoError(t, err)
				assert.Len(t, chunks, tt.expectedCount)
			}

			mockClient.AssertExpectations(t)
		})
	}
}

func TestChunkHandler_CreateChunk(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    models.CreateChunkRequest
		mockError      error
		expectedStatus int
	}{
		{
			name: "successful chunk creation",
			requestBody: models.CreateChunkRequest{
				Content:     "Test chunk content",
				TextID:      "text123",
				IndentLevel: 0,
			},
			mockError:      nil,
			expectedStatus: http.StatusCreated,
		},
		{
			name: "empty content",
			requestBody: models.CreateChunkRequest{
				Content: "",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "database error",
			requestBody: models.CreateChunkRequest{
				Content: "Test chunk content",
			},
			mockError:      fmt.Errorf("database error"),
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := new(MockSupabaseClient)
			handler := NewChunkHandler(mockClient)

			// Setup mock expectations
			if tt.requestBody.Content != "" && tt.expectedStatus != http.StatusBadRequest {
				mockClient.On("InsertChunk", mock.Anything, mock.AnythingOfType("*models.ChunkRecord")).
					Return(tt.mockError)
			}

			// Create request
			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest("POST", "/api/v1/chunks", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			// Execute
			handler.CreateChunk(w, req)

			// Assert
			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusCreated {
				var chunk models.ChunkRecord
				err := json.Unmarshal(w.Body.Bytes(), &chunk)
				assert.NoError(t, err)
				assert.Equal(t, tt.requestBody.Content, chunk.Content)
			}

			mockClient.AssertExpectations(t)
		})
	}
}

func TestChunkHandler_GetChunkByID(t *testing.T) {
	tests := []struct {
		name           string
		chunkID        string
		mockChunk      *models.ChunkRecord
		mockError      error
		expectedStatus int
	}{
		{
			name:    "successful get chunk by ID",
			chunkID: "chunk123",
			mockChunk: &models.ChunkRecord{
				ID:      "chunk123",
				Content: "Test chunk content",
			},
			mockError:      nil,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "chunk not found",
			chunkID:        "nonexistent",
			mockChunk:      nil,
			mockError:      fmt.Errorf("chunk not found"),
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "empty chunk ID",
			chunkID:        "",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := new(MockSupabaseClient)
			handler := NewChunkHandler(mockClient)

			// Setup mock expectations
			if tt.chunkID != "" {
				mockClient.On("GetChunkByID", mock.Anything, tt.chunkID).
					Return(tt.mockChunk, tt.mockError)
			}

			// Create request with mux vars
			req := httptest.NewRequest("GET", "/api/v1/chunks/"+tt.chunkID, nil)
			req = mux.SetURLVars(req, map[string]string{"id": tt.chunkID})
			w := httptest.NewRecorder()

			// Execute
			handler.GetChunkByID(w, req)

			// Assert
			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusOK {
				var chunk models.ChunkRecord
				err := json.Unmarshal(w.Body.Bytes(), &chunk)
				assert.NoError(t, err)
				assert.Equal(t, tt.mockChunk.ID, chunk.ID)
			}

			mockClient.AssertExpectations(t)
		})
	}
}

func TestChunkHandler_UpdateChunk(t *testing.T) {
	tests := []struct {
		name           string
		chunkID        string
		requestBody    models.UpdateChunkRequest
		existingChunk  *models.ChunkRecord
		mockGetError   error
		mockUpdateError error
		expectedStatus int
	}{
		{
			name:    "successful chunk update",
			chunkID: "chunk123",
			requestBody: models.UpdateChunkRequest{
				Content: stringPtr("Updated content"),
			},
			existingChunk: &models.ChunkRecord{
				ID:      "chunk123",
				Content: "Original content",
			},
			mockGetError:    nil,
			mockUpdateError: nil,
			expectedStatus:  http.StatusOK,
		},
		{
			name:           "chunk not found",
			chunkID:        "nonexistent",
			requestBody:    models.UpdateChunkRequest{},
			existingChunk:  nil,
			mockGetError:   fmt.Errorf("chunk not found"),
			expectedStatus: http.StatusNotFound,
		},
		{
			name:    "update error",
			chunkID: "chunk123",
			requestBody: models.UpdateChunkRequest{
				Content: stringPtr("Updated content"),
			},
			existingChunk: &models.ChunkRecord{
				ID:      "chunk123",
				Content: "Original content",
			},
			mockGetError:    nil,
			mockUpdateError: fmt.Errorf("update failed"),
			expectedStatus:  http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := new(MockSupabaseClient)
			handler := NewChunkHandler(mockClient)

			// Setup mock expectations
			mockClient.On("GetChunkByID", mock.Anything, tt.chunkID).
				Return(tt.existingChunk, tt.mockGetError)

			if tt.mockGetError == nil && tt.existingChunk != nil {
				mockClient.On("UpdateChunk", mock.Anything, mock.AnythingOfType("*models.ChunkRecord")).
					Return(tt.mockUpdateError)
			}

			// Create request
			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest("PUT", "/api/v1/chunks/"+tt.chunkID, bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			req = mux.SetURLVars(req, map[string]string{"id": tt.chunkID})
			w := httptest.NewRecorder()

			// Execute
			handler.UpdateChunk(w, req)

			// Assert
			assert.Equal(t, tt.expectedStatus, w.Code)

			mockClient.AssertExpectations(t)
		})
	}
}

func TestChunkHandler_DeleteChunk(t *testing.T) {
	tests := []struct {
		name           string
		chunkID        string
		mockError      error
		expectedStatus int
	}{
		{
			name:           "successful chunk deletion",
			chunkID:        "chunk123",
			mockError:      nil,
			expectedStatus: http.StatusNoContent,
		},
		{
			name:           "deletion error",
			chunkID:        "chunk123",
			mockError:      fmt.Errorf("deletion failed"),
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "empty chunk ID",
			chunkID:        "",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := new(MockSupabaseClient)
			handler := NewChunkHandler(mockClient)

			// Setup mock expectations
			if tt.chunkID != "" {
				mockClient.On("DeleteChunk", mock.Anything, tt.chunkID).
					Return(tt.mockError)
			}

			// Create request
			req := httptest.NewRequest("DELETE", "/api/v1/chunks/"+tt.chunkID, nil)
			req = mux.SetURLVars(req, map[string]string{"id": tt.chunkID})
			w := httptest.NewRecorder()

			// Execute
			handler.DeleteChunk(w, req)

			// Assert
			assert.Equal(t, tt.expectedStatus, w.Code)

			mockClient.AssertExpectations(t)
		})
	}
}

func TestChunkHandler_GetChunkHierarchy(t *testing.T) {
	tests := []struct {
		name           string
		chunkID        string
		mockHierarchy  *models.ChunkHierarchy
		mockError      error
		expectedStatus int
	}{
		{
			name:    "successful get hierarchy",
			chunkID: "chunk123",
			mockHierarchy: &models.ChunkHierarchy{
				Chunk: &models.ChunkRecord{
					ID:      "chunk123",
					Content: "Root chunk",
				},
				Children: []models.ChunkHierarchy{
					{
						Chunk: &models.ChunkRecord{
							ID:      "child1",
							Content: "Child chunk 1",
						},
						Level: 1,
					},
				},
				Level: 0,
			},
			mockError:      nil,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "hierarchy not found",
			chunkID:        "nonexistent",
			mockHierarchy:  nil,
			mockError:      fmt.Errorf("hierarchy not found"),
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := new(MockSupabaseClient)
			handler := NewChunkHandler(mockClient)

			// Setup mock expectations
			mockClient.On("GetChunkHierarchy", mock.Anything, tt.chunkID).
				Return(tt.mockHierarchy, tt.mockError)

			// Create request
			req := httptest.NewRequest("GET", "/api/v1/chunks/"+tt.chunkID+"/hierarchy", nil)
			req = mux.SetURLVars(req, map[string]string{"id": tt.chunkID})
			w := httptest.NewRecorder()

			// Execute
			handler.GetChunkHierarchy(w, req)

			// Assert
			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusOK {
				var hierarchy models.ChunkHierarchy
				err := json.Unmarshal(w.Body.Bytes(), &hierarchy)
				assert.NoError(t, err)
				assert.Equal(t, tt.mockHierarchy.Chunk.ID, hierarchy.Chunk.ID)
			}

			mockClient.AssertExpectations(t)
		})
	}
}

func TestChunkHandler_MoveChunk(t *testing.T) {
	tests := []struct {
		name           string
		chunkID        string
		requestBody    models.MoveChunkRequest
		mockError      error
		expectedStatus int
	}{
		{
			name:    "successful chunk move",
			chunkID: "chunk123",
			requestBody: models.MoveChunkRequest{
				NewParentID:    stringPtr("parent456"),
				NewPosition:    1,
				NewIndentLevel: 2,
			},
			mockError:      nil,
			expectedStatus: http.StatusNoContent,
		},
		{
			name:    "move error",
			chunkID: "chunk123",
			requestBody: models.MoveChunkRequest{
				NewPosition: 1,
			},
			mockError:      fmt.Errorf("move failed"),
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := new(MockSupabaseClient)
			handler := NewChunkHandler(mockClient)

			// Setup mock expectations
			mockClient.On("MoveChunk", mock.Anything, mock.AnythingOfType("*models.MoveChunkRequest")).
				Return(tt.mockError)

			// Create request
			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest("POST", "/api/v1/chunks/"+tt.chunkID+"/move", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			req = mux.SetURLVars(req, map[string]string{"id": tt.chunkID})
			w := httptest.NewRecorder()

			// Execute
			handler.MoveChunk(w, req)

			// Assert
			assert.Equal(t, tt.expectedStatus, w.Code)

			mockClient.AssertExpectations(t)
		})
	}
}

func TestChunkHandler_BulkUpdateChunks(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    models.BulkUpdateRequest
		mockError      error
		expectedStatus int
	}{
		{
			name: "successful bulk update",
			requestBody: models.BulkUpdateRequest{
				Updates: []models.ChunkUpdate{
					{
						ChunkID: "chunk1",
						Content: stringPtr("Updated content 1"),
					},
					{
						ChunkID: "chunk2",
						Content: stringPtr("Updated content 2"),
					},
				},
			},
			mockError:      nil,
			expectedStatus: http.StatusNoContent,
		},
		{
			name: "empty updates",
			requestBody: models.BulkUpdateRequest{
				Updates: []models.ChunkUpdate{},
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "bulk update error",
			requestBody: models.BulkUpdateRequest{
				Updates: []models.ChunkUpdate{
					{
						ChunkID: "chunk1",
						Content: stringPtr("Updated content 1"),
					},
				},
			},
			mockError:      fmt.Errorf("bulk update failed"),
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := new(MockSupabaseClient)
			handler := NewChunkHandler(mockClient)

			// Setup mock expectations
			if len(tt.requestBody.Updates) > 0 && tt.expectedStatus != http.StatusBadRequest {
				mockClient.On("BulkUpdateChunks", mock.Anything, mock.AnythingOfType("*models.BulkUpdateRequest")).
					Return(tt.mockError)
			}

			// Create request
			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest("PUT", "/api/v1/chunks/bulk-update", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			// Execute
			handler.BulkUpdateChunks(w, req)

			// Assert
			assert.Equal(t, tt.expectedStatus, w.Code)

			mockClient.AssertExpectations(t)
		})
	}
}

