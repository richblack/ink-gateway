package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"semantic-text-processor/models"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockTextProcessor implements TextProcessor interface for testing
type MockTextProcessor struct {
	mock.Mock
}

func (m *MockTextProcessor) ProcessText(ctx context.Context, text string) (*models.ProcessResult, error) {
	args := m.Called(ctx, text)
	return args.Get(0).(*models.ProcessResult), args.Error(1)
}

func (m *MockTextProcessor) ChunkText(ctx context.Context, text string) ([]models.ChunkRecord, error) {
	args := m.Called(ctx, text)
	return args.Get(0).([]models.ChunkRecord), args.Error(1)
}

func (m *MockTextProcessor) GenerateEmbeddings(ctx context.Context, chunks []models.ChunkRecord) ([]models.EmbeddingRecord, error) {
	args := m.Called(ctx, chunks)
	return args.Get(0).([]models.EmbeddingRecord), args.Error(1)
}

func (m *MockTextProcessor) ExtractKnowledge(ctx context.Context, chunks []models.ChunkRecord) (*models.GraphResult, error) {
	args := m.Called(ctx, chunks)
	return args.Get(0).(*models.GraphResult), args.Error(1)
}



// Test helper functions
func setupTextHandler() (*TextHandler, *MockTextProcessor, *MockSupabaseClient) {
	mockTextProcessor := new(MockTextProcessor)
	mockSupabaseClient := new(MockSupabaseClient)
	handler := NewTextHandler(mockTextProcessor, mockSupabaseClient)
	return handler, mockTextProcessor, mockSupabaseClient
}

func createTestRouter(handler *TextHandler) *mux.Router {
	router := mux.NewRouter()
	api := router.PathPrefix("/api/v1").Subrouter()
	
	api.HandleFunc("/texts", handler.CreateText).Methods("POST")
	api.HandleFunc("/texts", handler.GetTexts).Methods("GET")
	api.HandleFunc("/texts/{id}", handler.GetTextByID).Methods("GET")
	api.HandleFunc("/texts/{id}", handler.UpdateText).Methods("PUT")
	api.HandleFunc("/texts/{id}", handler.DeleteText).Methods("DELETE")
	api.HandleFunc("/texts/{id}/structure", handler.GetTextStructure).Methods("GET")
	api.HandleFunc("/texts/{id}/structure", handler.UpdateTextStructure).Methods("PUT")
	
	return router
}

// Test CreateText endpoint
func TestCreateText(t *testing.T) {
	handler, mockTextProcessor, mockSupabaseClient := setupTextHandler()
	router := createTestRouter(handler)

	t.Run("successful text creation", func(t *testing.T) {
		// Prepare test data
		testText := "This is a test text for processing."
		testTextID := "test-text-id-123"
		
		processResult := &models.ProcessResult{
			TextID: testTextID,
			Chunks: []models.ChunkRecord{
				{
					ID:      "chunk-1",
					TextID:  testTextID,
					Content: "This is a test text for processing.",
				},
			},
			Status:      "completed",
			ProcessedAt: time.Now(),
		}
		
		embeddings := []models.EmbeddingRecord{
			{
				ID:      "embedding-1",
				ChunkID: "chunk-1",
				Vector:  []float64{0.1, 0.2, 0.3},
			},
		}
		
		graphResult := &models.GraphResult{
			Nodes: []models.GraphNode{
				{
					ID:         "node-1",
					ChunkID:    "chunk-1",
					EntityName: "test",
					EntityType: "concept",
				},
			},
			Edges: []models.GraphEdge{},
		}

		// Set up mocks
		mockTextProcessor.On("ProcessText", mock.Anything, testText).Return(processResult, nil)
		mockSupabaseClient.On("InsertText", mock.Anything, mock.AnythingOfType("*models.TextRecord")).Return(nil)
		mockSupabaseClient.On("InsertChunks", mock.Anything, processResult.Chunks).Return(nil)
		mockTextProcessor.On("GenerateEmbeddings", mock.Anything, processResult.Chunks).Return(embeddings, nil)
		mockSupabaseClient.On("InsertEmbeddings", mock.Anything, embeddings).Return(nil)
		mockTextProcessor.On("ExtractKnowledge", mock.Anything, processResult.Chunks).Return(graphResult, nil)
		mockSupabaseClient.On("InsertGraphNodes", mock.Anything, graphResult.Nodes).Return(nil)
		mockSupabaseClient.On("InsertGraphEdges", mock.Anything, graphResult.Edges).Return(nil)

		// Create request
		reqBody := models.CreateTextRequest{
			Content: testText,
			Title:   "Test Text",
		}
		jsonBody, _ := json.Marshal(reqBody)

		// Make request
		req := httptest.NewRequest("POST", "/api/v1/texts", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		// Assert response
		assert.Equal(t, http.StatusCreated, w.Code)
		
		var response models.CreateTextResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, testTextID, response.ID)
		assert.Equal(t, "completed", response.Status)
		assert.Equal(t, 1, response.ChunkCount)

		// Verify all mocks were called
		mockTextProcessor.AssertExpectations(t)
		mockSupabaseClient.AssertExpectations(t)
	})

	t.Run("invalid request body", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/texts", bytes.NewBuffer([]byte("invalid json")))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		
		var errorResp ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &errorResp)
		assert.NoError(t, err)
		assert.Equal(t, "invalid request body", errorResp.Message)
	})

	t.Run("empty content", func(t *testing.T) {
		reqBody := models.CreateTextRequest{
			Content: "",
			Title:   "Empty Test",
		}
		jsonBody, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/v1/texts", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		
		var errorResp ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &errorResp)
		assert.NoError(t, err)
		assert.Equal(t, "content is required", errorResp.Message)
	})

	t.Run("text processing failure", func(t *testing.T) {
		testText := "This will fail processing."
		
		// Set up mock to return error
		mockTextProcessor.On("ProcessText", mock.Anything, testText).Return((*models.ProcessResult)(nil), fmt.Errorf("processing failed"))

		reqBody := models.CreateTextRequest{
			Content: testText,
			Title:   "Failing Test",
		}
		jsonBody, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/v1/texts", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		
		var errorResp ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &errorResp)
		assert.NoError(t, err)
		assert.Equal(t, "failed to process text", errorResp.Message)

		mockTextProcessor.AssertExpectations(t)
	})
}

// Test GetTexts endpoint
func TestGetTexts(t *testing.T) {
	handler, _, mockSupabaseClient := setupTextHandler()
	router := createTestRouter(handler)

	t.Run("successful get texts with pagination", func(t *testing.T) {
		// Prepare test data
		expectedTexts := &models.TextList{
			Texts: []models.TextRecord{
				{
					ID:      "text-1",
					Content: "First text",
					Title:   "First",
					Status:  "completed",
				},
				{
					ID:      "text-2",
					Content: "Second text",
					Title:   "Second",
					Status:  "completed",
				},
			},
			Pagination: models.Pagination{
				Page:     1,
				PageSize: 20,
				Total:    2,
			},
		}

		// Set up mock
		mockSupabaseClient.On("GetTexts", mock.Anything, mock.AnythingOfType("*models.Pagination")).Return(expectedTexts, nil)

		// Make request
		req := httptest.NewRequest("GET", "/api/v1/texts?page=1&page_size=20", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		// Assert response
		assert.Equal(t, http.StatusOK, w.Code)
		
		var response models.TextList
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, 2, len(response.Texts))
		assert.Equal(t, "text-1", response.Texts[0].ID)
		assert.Equal(t, 1, response.Pagination.Page)
		assert.Equal(t, 20, response.Pagination.PageSize)

		mockSupabaseClient.AssertExpectations(t)
	})

	t.Run("default pagination parameters", func(t *testing.T) {
		expectedTexts := &models.TextList{
			Texts: []models.TextRecord{},
			Pagination: models.Pagination{
				Page:     1,
				PageSize: 20,
				Total:    0,
			},
		}

		mockSupabaseClient.On("GetTexts", mock.Anything, mock.MatchedBy(func(p *models.Pagination) bool {
			return p.Page == 1 && p.PageSize == 20
		})).Return(expectedTexts, nil)

		req := httptest.NewRequest("GET", "/api/v1/texts", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		mockSupabaseClient.AssertExpectations(t)
	})

	t.Run("database error", func(t *testing.T) {
		// Create a fresh mock for this test
		_, _, freshMockSupabaseClient := setupTextHandler()
		freshHandler := NewTextHandler(new(MockTextProcessor), freshMockSupabaseClient)
		freshRouter := createTestRouter(freshHandler)
		
		freshMockSupabaseClient.On("GetTexts", mock.Anything, mock.AnythingOfType("*models.Pagination")).Return((*models.TextList)(nil), fmt.Errorf("database error"))

		req := httptest.NewRequest("GET", "/api/v1/texts", nil)
		w := httptest.NewRecorder()

		freshRouter.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		
		var errorResp ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &errorResp)
		assert.NoError(t, err)
		assert.Equal(t, "failed to get texts", errorResp.Message)

		freshMockSupabaseClient.AssertExpectations(t)
	})
}

// Test GetTextByID endpoint
func TestGetTextByID(t *testing.T) {
	handler, _, mockSupabaseClient := setupTextHandler()
	router := createTestRouter(handler)

	t.Run("successful get text by ID", func(t *testing.T) {
		testID := "test-text-123"
		expectedText := &models.TextDetail{
			Text: models.TextRecord{
				ID:      testID,
				Content: "Test content",
				Title:   "Test Title",
				Status:  "completed",
			},
			Chunks: []models.ChunkRecord{
				{
					ID:      "chunk-1",
					TextID:  testID,
					Content: "Test content",
				},
			},
		}

		mockSupabaseClient.On("GetTextByID", mock.Anything, testID).Return(expectedText, nil)

		req := httptest.NewRequest("GET", "/api/v1/texts/"+testID, nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		
		var response models.TextDetail
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, testID, response.Text.ID)
		assert.Equal(t, 1, len(response.Chunks))

		mockSupabaseClient.AssertExpectations(t)
	})

	t.Run("text not found", func(t *testing.T) {
		testID := "nonexistent-text"
		
		mockSupabaseClient.On("GetTextByID", mock.Anything, testID).Return((*models.TextDetail)(nil), fmt.Errorf("text not found"))

		req := httptest.NewRequest("GET", "/api/v1/texts/"+testID, nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		
		var errorResp ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &errorResp)
		assert.NoError(t, err)
		assert.Equal(t, "text not found", errorResp.Message)

		mockSupabaseClient.AssertExpectations(t)
	})
}

// Test UpdateText endpoint
func TestUpdateText(t *testing.T) {
	handler, _, mockSupabaseClient := setupTextHandler()
	router := createTestRouter(handler)

	t.Run("successful text update", func(t *testing.T) {
		testID := "test-text-123"
		existingText := &models.TextDetail{
			Text: models.TextRecord{
				ID:      testID,
				Content: "Original content",
				Title:   "Original Title",
				Status:  "completed",
			},
			Chunks: []models.ChunkRecord{},
		}

		updateReq := models.UpdateTextRequest{
			Title:   stringPtr("Updated Title"),
			Content: stringPtr("Updated content"),
		}

		mockSupabaseClient.On("GetTextByID", mock.Anything, testID).Return(existingText, nil)
		mockSupabaseClient.On("UpdateText", mock.Anything, mock.MatchedBy(func(text *models.TextRecord) bool {
			return text.ID == testID && text.Title == "Updated Title" && text.Content == "Updated content" && text.Status == "pending_reprocess"
		})).Return(nil)

		jsonBody, _ := json.Marshal(updateReq)
		req := httptest.NewRequest("PUT", "/api/v1/texts/"+testID, bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		
		var response models.TextRecord
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, testID, response.ID)
		assert.Equal(t, "Updated Title", response.Title)
		assert.Equal(t, "Updated content", response.Content)
		assert.Equal(t, "pending_reprocess", response.Status)

		mockSupabaseClient.AssertExpectations(t)
	})

	t.Run("text not found for update", func(t *testing.T) {
		testID := "nonexistent-text"
		updateReq := models.UpdateTextRequest{
			Title: stringPtr("Updated Title"),
		}

		mockSupabaseClient.On("GetTextByID", mock.Anything, testID).Return((*models.TextDetail)(nil), fmt.Errorf("text not found"))

		jsonBody, _ := json.Marshal(updateReq)
		req := httptest.NewRequest("PUT", "/api/v1/texts/"+testID, bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)

		mockSupabaseClient.AssertExpectations(t)
	})
}

// Test DeleteText endpoint
func TestDeleteText(t *testing.T) {
	handler, _, mockSupabaseClient := setupTextHandler()
	router := createTestRouter(handler)

	t.Run("successful text deletion", func(t *testing.T) {
		testID := "test-text-123"
		
		mockSupabaseClient.On("DeleteText", mock.Anything, testID).Return(nil)

		req := httptest.NewRequest("DELETE", "/api/v1/texts/"+testID, nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
		assert.Empty(t, w.Body.String())

		mockSupabaseClient.AssertExpectations(t)
	})

	t.Run("deletion failure", func(t *testing.T) {
		// Create a fresh mock for this test
		_, _, freshMockSupabaseClient := setupTextHandler()
		freshHandler := NewTextHandler(new(MockTextProcessor), freshMockSupabaseClient)
		freshRouter := createTestRouter(freshHandler)
		
		testID := "test-text-123"
		freshMockSupabaseClient.On("DeleteText", mock.Anything, testID).Return(fmt.Errorf("deletion failed"))

		req := httptest.NewRequest("DELETE", "/api/v1/texts/"+testID, nil)
		w := httptest.NewRecorder()

		freshRouter.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		
		var errorResp ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &errorResp)
		assert.NoError(t, err)
		assert.Equal(t, "failed to delete text", errorResp.Message)

		freshMockSupabaseClient.AssertExpectations(t)
	})
}

// Test GetTextStructure endpoint
func TestGetTextStructure(t *testing.T) {
	handler, _, mockSupabaseClient := setupTextHandler()
	router := createTestRouter(handler)

	t.Run("successful get text structure", func(t *testing.T) {
		testID := "test-text-123"
		// Create chunks with explicit nil and string pointer values
		chunk1 := models.ChunkRecord{
			ID:             "chunk-1",
			TextID:         testID,
			Content:        "Root chunk",
			ParentChunkID:  nil, // Explicitly nil for root chunk
			IndentLevel:    0,
			SequenceNumber: intPtr(0),
		}
		
		parentID := "chunk-1" // Create the string first
		chunk2 := models.ChunkRecord{
			ID:             "chunk-2",
			TextID:         testID,
			Content:        "Child chunk",
			ParentChunkID:  &parentID, // Use address of the string
			IndentLevel:    1,
			SequenceNumber: intPtr(0),
		}
		
		chunks := []models.ChunkRecord{chunk1, chunk2}

		mockSupabaseClient.On("GetChunksByTextID", mock.Anything, testID).Return(chunks, nil)

		req := httptest.NewRequest("GET", "/api/v1/texts/"+testID+"/structure", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		
		var response models.BulletStructure
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		// Should have exactly 1 root chunk
		assert.Equal(t, 1, len(response.RootChunks))
		
		// The root chunk should be chunk-1
		rootChunk := response.RootChunks[0]
		assert.Equal(t, "chunk-1", rootChunk.Chunk.ID)
		assert.Nil(t, rootChunk.Chunk.ParentChunkID)
		assert.Equal(t, 1, len(rootChunk.Children))
		
		// The child should be chunk-2
		childChunk := rootChunk.Children[0]
		assert.Equal(t, "chunk-2", childChunk.Chunk.ID)
		assert.NotNil(t, childChunk.Chunk.ParentChunkID)
		assert.Equal(t, "chunk-1", *childChunk.Chunk.ParentChunkID)

		mockSupabaseClient.AssertExpectations(t)
	})

	t.Run("chunks not found", func(t *testing.T) {
		testID := "nonexistent-text"
		
		mockSupabaseClient.On("GetChunksByTextID", mock.Anything, testID).Return(([]models.ChunkRecord)(nil), fmt.Errorf("chunks not found"))

		req := httptest.NewRequest("GET", "/api/v1/texts/"+testID+"/structure", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)

		mockSupabaseClient.AssertExpectations(t)
	})
}

// Test UpdateTextStructure endpoint
func TestUpdateTextStructure(t *testing.T) {
	handler, _, mockSupabaseClient := setupTextHandler()
	router := createTestRouter(handler)

	t.Run("successful structure update", func(t *testing.T) {
		testID := "test-text-123"
		bulkUpdateReq := models.BulkUpdateRequest{
			Updates: []models.ChunkUpdate{
				{
					ChunkID:        "chunk-1",
					Content:        stringPtr("Updated content"),
					IndentLevel:    intPtr(1),
					SequenceNumber: intPtr(0),
				},
			},
		}

		mockSupabaseClient.On("BulkUpdateChunks", mock.Anything, &bulkUpdateReq).Return(nil)

		jsonBody, _ := json.Marshal(bulkUpdateReq)
		req := httptest.NewRequest("PUT", "/api/v1/texts/"+testID+"/structure", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
		assert.Empty(t, w.Body.String())

		mockSupabaseClient.AssertExpectations(t)
	})

	t.Run("bulk update failure", func(t *testing.T) {
		testID := "test-text-123"
		bulkUpdateReq := models.BulkUpdateRequest{
			Updates: []models.ChunkUpdate{
				{
					ChunkID: "chunk-1",
					Content: stringPtr("Updated content"),
				},
			},
		}

		mockSupabaseClient.On("BulkUpdateChunks", mock.Anything, &bulkUpdateReq).Return(fmt.Errorf("update failed"))

		jsonBody, _ := json.Marshal(bulkUpdateReq)
		req := httptest.NewRequest("PUT", "/api/v1/texts/"+testID+"/structure", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)

		mockSupabaseClient.AssertExpectations(t)
	})
}



