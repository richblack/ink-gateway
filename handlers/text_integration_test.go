package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"semantic-text-processor/models"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// TestTextProcessingIntegration tests the complete text processing workflow
func TestTextProcessingIntegration(t *testing.T) {
	// Setup
	handler, mockTextProcessor, mockSupabaseClient := setupTextHandler()
	router := createTestRouter(handler)

	// Test data
	testText := "# Introduction\nThis is a sample document.\n\n## Section 1\nFirst section content.\n\n### Subsection 1.1\nSubsection content."
	testTextID := "integration-test-123"
	
	// Expected chunks from processing - create with proper parent relationships
	chunk1ID := "chunk-1"
	chunk3ID := "chunk-3"
	chunk5ID := "chunk-5"
	
	expectedChunks := []models.ChunkRecord{
		{
			ID:             "chunk-1",
			TextID:         testTextID,
			Content:        "# Introduction",
			ParentChunkID:  nil, // Root chunk
			IndentLevel:    0,
			SequenceNumber: intPtr(0),
		},
		{
			ID:             "chunk-2",
			TextID:         testTextID,
			Content:        "This is a sample document.",
			ParentChunkID:  &chunk1ID, // Child of chunk-1
			IndentLevel:    1,
			SequenceNumber: intPtr(0),
		},
		{
			ID:             "chunk-3",
			TextID:         testTextID,
			Content:        "## Section 1",
			ParentChunkID:  nil, // Root chunk
			IndentLevel:    0,
			SequenceNumber: intPtr(1),
		},
		{
			ID:             "chunk-4",
			TextID:         testTextID,
			Content:        "First section content.",
			ParentChunkID:  &chunk3ID, // Child of chunk-3
			IndentLevel:    1,
			SequenceNumber: intPtr(0),
		},
		{
			ID:             "chunk-5",
			TextID:         testTextID,
			Content:        "### Subsection 1.1",
			ParentChunkID:  &chunk3ID, // Child of chunk-3
			IndentLevel:    1,
			SequenceNumber: intPtr(1),
		},
		{
			ID:             "chunk-6",
			TextID:         testTextID,
			Content:        "Subsection content.",
			ParentChunkID:  &chunk5ID, // Child of chunk-5
			IndentLevel:    2,
			SequenceNumber: intPtr(0),
		},
	}

	processResult := &models.ProcessResult{
		TextID:      testTextID,
		Chunks:      expectedChunks,
		Status:      "completed",
		ProcessedAt: time.Now(),
	}

	// Expected embeddings
	expectedEmbeddings := []models.EmbeddingRecord{
		{ID: "emb-1", ChunkID: "chunk-1", Vector: []float64{0.1, 0.2, 0.3}},
		{ID: "emb-2", ChunkID: "chunk-2", Vector: []float64{0.4, 0.5, 0.6}},
		{ID: "emb-3", ChunkID: "chunk-3", Vector: []float64{0.7, 0.8, 0.9}},
		{ID: "emb-4", ChunkID: "chunk-4", Vector: []float64{0.1, 0.3, 0.5}},
		{ID: "emb-5", ChunkID: "chunk-5", Vector: []float64{0.2, 0.4, 0.6}},
		{ID: "emb-6", ChunkID: "chunk-6", Vector: []float64{0.3, 0.5, 0.7}},
	}

	// Expected knowledge graph
	expectedGraph := &models.GraphResult{
		Nodes: []models.GraphNode{
			{ID: "node-1", ChunkID: "chunk-1", EntityName: "Introduction", EntityType: "section"},
			{ID: "node-2", ChunkID: "chunk-3", EntityName: "Section 1", EntityType: "section"},
			{ID: "node-3", ChunkID: "chunk-5", EntityName: "Subsection 1.1", EntityType: "subsection"},
		},
		Edges: []models.GraphEdge{
			{ID: "edge-1", SourceNodeID: "node-2", TargetNodeID: "node-3", RelationshipType: "contains"},
		},
	}

	t.Run("complete text processing workflow", func(t *testing.T) {
		// Set up all mocks for the complete workflow
		mockTextProcessor.On("ProcessText", mock.Anything, testText).Return(processResult, nil)
		mockSupabaseClient.On("InsertText", mock.Anything, mock.AnythingOfType("*models.TextRecord")).Return(nil)
		mockSupabaseClient.On("InsertChunks", mock.Anything, expectedChunks).Return(nil)
		mockTextProcessor.On("GenerateEmbeddings", mock.Anything, expectedChunks).Return(expectedEmbeddings, nil)
		mockSupabaseClient.On("InsertEmbeddings", mock.Anything, expectedEmbeddings).Return(nil)
		mockTextProcessor.On("ExtractKnowledge", mock.Anything, expectedChunks).Return(expectedGraph, nil)
		mockSupabaseClient.On("InsertGraphNodes", mock.Anything, expectedGraph.Nodes).Return(nil)
		mockSupabaseClient.On("InsertGraphEdges", mock.Anything, expectedGraph.Edges).Return(nil)

		// Step 1: Create text
		createReq := models.CreateTextRequest{
			Content: testText,
			Title:   "Integration Test Document",
		}
		jsonBody, _ := json.Marshal(createReq)

		req := httptest.NewRequest("POST", "/api/v1/texts", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		// Verify creation response
		assert.Equal(t, http.StatusCreated, w.Code)
		
		var createResponse models.CreateTextResponse
		err := json.Unmarshal(w.Body.Bytes(), &createResponse)
		assert.NoError(t, err)
		assert.Equal(t, testTextID, createResponse.ID)
		assert.Equal(t, "completed", createResponse.Status)
		assert.Equal(t, 6, createResponse.ChunkCount)

		// Verify all mocks were called
		mockTextProcessor.AssertExpectations(t)
		mockSupabaseClient.AssertExpectations(t)
	})

	t.Run("retrieve and verify text structure", func(t *testing.T) {
		// Reset mocks for this test
		mockTextProcessor.ExpectedCalls = nil
		mockSupabaseClient.ExpectedCalls = nil

		// Set up mock for getting text structure
		mockSupabaseClient.On("GetChunksByTextID", mock.Anything, testTextID).Return(expectedChunks, nil)

		// Get text structure
		req := httptest.NewRequest("GET", "/api/v1/texts/"+testTextID+"/structure", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		// Verify structure response
		assert.Equal(t, http.StatusOK, w.Code)
		
		var structureResponse models.BulletStructure
		err := json.Unmarshal(w.Body.Bytes(), &structureResponse)
		assert.NoError(t, err)

		// Verify hierarchical structure
		assert.Equal(t, 2, len(structureResponse.RootChunks)) // Introduction and Section 1
		assert.Equal(t, 2, structureResponse.MaxDepth)       // 3 levels: 0, 1, 2

		// Verify Introduction section
		introSection := findChunkInRoots(structureResponse.RootChunks, "chunk-1")
		assert.NotNil(t, introSection)
		assert.Equal(t, "# Introduction", introSection.Chunk.Content)
		assert.Equal(t, 1, len(introSection.Children))
		assert.Equal(t, "This is a sample document.", introSection.Children[0].Chunk.Content)

		// Verify Section 1
		section1 := findChunkInRoots(structureResponse.RootChunks, "chunk-3")
		assert.NotNil(t, section1)
		assert.Equal(t, "## Section 1", section1.Chunk.Content)
		assert.Equal(t, 2, len(section1.Children)) // Content + Subsection

		// Verify Subsection 1.1
		subsection := findChunkInChildren(section1.Children, "chunk-5")
		assert.NotNil(t, subsection)
		assert.Equal(t, "### Subsection 1.1", subsection.Chunk.Content)
		assert.Equal(t, 1, len(subsection.Children))
		assert.Equal(t, "Subsection content.", subsection.Children[0].Chunk.Content)

		mockSupabaseClient.AssertExpectations(t)
	})

	t.Run("update text structure", func(t *testing.T) {
		// Reset mocks
		mockSupabaseClient.ExpectedCalls = nil

		// Prepare bulk update request (move subsection content to be under introduction)
		newParentID := "chunk-1"
		bulkUpdate := models.BulkUpdateRequest{
			Updates: []models.ChunkUpdate{
				{
					ChunkID:        "chunk-6",
					ParentChunkID:  &newParentID, // Move to under Introduction
					IndentLevel:    intPtr(1),
					SequenceNumber: intPtr(1),
				},
			},
		}

		mockSupabaseClient.On("BulkUpdateChunks", mock.Anything, &bulkUpdate).Return(nil)

		// Update structure
		jsonBody, _ := json.Marshal(bulkUpdate)
		req := httptest.NewRequest("PUT", "/api/v1/texts/"+testTextID+"/structure", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		// Verify update response
		assert.Equal(t, http.StatusNoContent, w.Code)
		assert.Empty(t, w.Body.String())

		mockSupabaseClient.AssertExpectations(t)
	})

	t.Run("retrieve text list with pagination", func(t *testing.T) {
		// Reset mocks
		mockSupabaseClient.ExpectedCalls = nil

		// Set up mock for text list
		expectedTextList := &models.TextList{
			Texts: []models.TextRecord{
				{
					ID:      testTextID,
					Content: testText,
					Title:   "Integration Test Document",
					Status:  "completed",
				},
			},
			Pagination: models.Pagination{
				Page:     1,
				PageSize: 20,
				Total:    1,
			},
		}

		mockSupabaseClient.On("GetTexts", mock.Anything, mock.AnythingOfType("*models.Pagination")).Return(expectedTextList, nil)

		// Get text list
		req := httptest.NewRequest("GET", "/api/v1/texts?page=1&page_size=20", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		// Verify list response
		assert.Equal(t, http.StatusOK, w.Code)
		
		var listResponse models.TextList
		err := json.Unmarshal(w.Body.Bytes(), &listResponse)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(listResponse.Texts))
		assert.Equal(t, testTextID, listResponse.Texts[0].ID)
		assert.Equal(t, "Integration Test Document", listResponse.Texts[0].Title)
		assert.Equal(t, 1, listResponse.Pagination.Page)
		assert.Equal(t, 20, listResponse.Pagination.PageSize)
		assert.Equal(t, 1, listResponse.Pagination.Total)

		mockSupabaseClient.AssertExpectations(t)
	})

	t.Run("retrieve text details", func(t *testing.T) {
		// Reset mocks
		mockSupabaseClient.ExpectedCalls = nil

		// Set up mock for text details
		expectedTextDetail := &models.TextDetail{
			Text: models.TextRecord{
				ID:      testTextID,
				Content: testText,
				Title:   "Integration Test Document",
				Status:  "completed",
			},
			Chunks: expectedChunks,
		}

		mockSupabaseClient.On("GetTextByID", mock.Anything, testTextID).Return(expectedTextDetail, nil)

		// Get text details
		req := httptest.NewRequest("GET", "/api/v1/texts/"+testTextID, nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		// Verify details response
		assert.Equal(t, http.StatusOK, w.Code)
		
		var detailResponse models.TextDetail
		err := json.Unmarshal(w.Body.Bytes(), &detailResponse)
		assert.NoError(t, err)
		assert.Equal(t, testTextID, detailResponse.Text.ID)
		assert.Equal(t, "Integration Test Document", detailResponse.Text.Title)
		assert.Equal(t, 6, len(detailResponse.Chunks))

		mockSupabaseClient.AssertExpectations(t)
	})
}

// Helper functions for integration tests

func findChunkInRoots(roots []models.ChunkHierarchy, chunkID string) *models.ChunkHierarchy {
	for i := range roots {
		if roots[i].Chunk.ID == chunkID {
			return &roots[i]
		}
	}
	return nil
}

func findChunkInChildren(children []models.ChunkHierarchy, chunkID string) *models.ChunkHierarchy {
	for i := range children {
		if children[i].Chunk.ID == chunkID {
			return &children[i]
		}
	}
	return nil
}

// TestTextProcessingErrorHandling tests error scenarios in the text processing workflow
func TestTextProcessingErrorHandling(t *testing.T) {
	handler, mockTextProcessor, mockSupabaseClient := setupTextHandler()
	router := createTestRouter(handler)

	t.Run("LLM processing failure with graceful degradation", func(t *testing.T) {
		testText := "This text will fail LLM processing"
		
		// Mock LLM failure
		mockTextProcessor.On("ProcessText", mock.Anything, testText).Return((*models.ProcessResult)(nil), 
			context.DeadlineExceeded) // Simulate timeout

		createReq := models.CreateTextRequest{
			Content: testText,
			Title:   "Failing Test",
		}
		jsonBody, _ := json.Marshal(createReq)

		req := httptest.NewRequest("POST", "/api/v1/texts", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		// Should return error
		assert.Equal(t, http.StatusInternalServerError, w.Code)
		
		var errorResp ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &errorResp)
		assert.NoError(t, err)
		assert.Equal(t, "failed to process text", errorResp.Message)
		assert.Contains(t, errorResp.Details, "context deadline exceeded")

		mockTextProcessor.AssertExpectations(t)
	})

	t.Run("database failure during text insertion", func(t *testing.T) {
		testText := "This text will fail database insertion"
		testTextID := "failing-db-test"
		
		processResult := &models.ProcessResult{
			TextID: testTextID,
			Chunks: []models.ChunkRecord{
				{ID: "chunk-1", TextID: testTextID, Content: testText},
			},
			Status:      "completed",
			ProcessedAt: time.Now(),
		}

		// Mock successful processing but database failure
		mockTextProcessor.On("ProcessText", mock.Anything, testText).Return(processResult, nil)
		mockSupabaseClient.On("InsertText", mock.Anything, mock.AnythingOfType("*models.TextRecord")).
			Return(context.DeadlineExceeded) // Simulate database timeout

		createReq := models.CreateTextRequest{
			Content: testText,
			Title:   "DB Failing Test",
		}
		jsonBody, _ := json.Marshal(createReq)

		req := httptest.NewRequest("POST", "/api/v1/texts", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		// Should return database error
		assert.Equal(t, http.StatusInternalServerError, w.Code)
		
		var errorResp ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &errorResp)
		assert.NoError(t, err)
		assert.Equal(t, "failed to save text", errorResp.Message)

		mockTextProcessor.AssertExpectations(t)
		mockSupabaseClient.AssertExpectations(t)
	})

	t.Run("partial failure - embeddings fail but text processing succeeds", func(t *testing.T) {
		// Create fresh mocks for this test
		freshHandler, freshMockTextProcessor, freshMockSupabaseClient := setupTextHandler()
		freshRouter := createTestRouter(freshHandler)
		
		testText := "This text will have embedding failure"
		testTextID := "partial-fail-test"
		
		chunks := []models.ChunkRecord{
			{ID: "chunk-1", TextID: testTextID, Content: testText},
		}
		
		processResult := &models.ProcessResult{
			TextID:      testTextID,
			Chunks:      chunks,
			Status:      "completed",
			ProcessedAt: time.Now(),
		}

		// Mock successful processing and text/chunk insertion
		freshMockTextProcessor.On("ProcessText", mock.Anything, testText).Return(processResult, nil)
		freshMockSupabaseClient.On("InsertText", mock.Anything, mock.AnythingOfType("*models.TextRecord")).Return(nil)
		freshMockSupabaseClient.On("InsertChunks", mock.Anything, chunks).Return(nil)
		
		// Mock embedding failure (should not fail the whole request)
		freshMockTextProcessor.On("GenerateEmbeddings", mock.Anything, chunks).
			Return(([]models.EmbeddingRecord)(nil), context.DeadlineExceeded)
		
		// Mock successful knowledge extraction (should still work)
		graphResult := &models.GraphResult{
			Nodes: []models.GraphNode{{ID: "node-1", ChunkID: "chunk-1", EntityName: "test", EntityType: "concept"}},
			Edges: []models.GraphEdge{},
		}
		freshMockTextProcessor.On("ExtractKnowledge", mock.Anything, chunks).Return(graphResult, nil)
		freshMockSupabaseClient.On("InsertGraphNodes", mock.Anything, graphResult.Nodes).Return(nil)
		freshMockSupabaseClient.On("InsertGraphEdges", mock.Anything, graphResult.Edges).Return(nil)

		createReq := models.CreateTextRequest{
			Content: testText,
			Title:   "Partial Fail Test",
		}
		jsonBody, _ := json.Marshal(createReq)

		req := httptest.NewRequest("POST", "/api/v1/texts", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		freshRouter.ServeHTTP(w, req)

		// Should still succeed despite embedding failure
		assert.Equal(t, http.StatusCreated, w.Code)
		
		var createResponse models.CreateTextResponse
		err := json.Unmarshal(w.Body.Bytes(), &createResponse)
		assert.NoError(t, err)
		assert.Equal(t, testTextID, createResponse.ID)
		assert.Equal(t, "completed", createResponse.Status)

		freshMockTextProcessor.AssertExpectations(t)
		freshMockSupabaseClient.AssertExpectations(t)
	})
}