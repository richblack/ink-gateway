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
	"github.com/stretchr/testify/require"
)

// Integration test for search APIs
func TestSearchAPIsIntegration(t *testing.T) {
	// Create a mock search service with realistic data
	mockService := &MockSearchService{}
	handler := NewSearchHandler(mockService)

	// Test data
	testChunk := models.ChunkRecord{
		ID:        "chunk-123",
		TextID:    "text-456",
		Content:   "This is a test chunk with important information",
		CreatedAt: time.Now(),
	}

	testTag := models.ChunkRecord{
		ID:      "tag-789",
		Content: "important",
	}

	testGraphNode := models.GraphNode{
		ID:         "node-123",
		ChunkID:    "chunk-123",
		EntityName: "Test Entity",
		EntityType: "person",
		Properties: map[string]interface{}{
			"description": "A test entity for graph search",
		},
	}

	testGraphEdge := models.GraphEdge{
		ID:               "edge-456",
		SourceNodeID:     "node-123",
		TargetNodeID:     "node-789",
		RelationshipType: "knows",
		Properties: map[string]interface{}{
			"since": "2023",
		},
	}

	t.Run("SemanticSearchAPI_Integration", func(t *testing.T) {
		// Setup mock response
		expectedResponse := &models.SemanticSearchResponse{
			Results: []models.SimilarityResult{
				{
					Chunk:      testChunk,
					Similarity: 0.95,
				},
			},
			TotalCount: 1,
			Query:      "test query",
			Limit:      10,
		}

		mockService.On("SemanticSearchWithFilters", 
			context.Background(), 
			&models.SemanticSearchRequest{
				Query: "test query",
				Limit: 10,
			}).Return(expectedResponse, nil)

		// Create request
		requestBody := models.SemanticSearchRequest{
			Query: "test query",
			Limit: 10,
		}
		body, _ := json.Marshal(requestBody)
		req := httptest.NewRequest("POST", "/api/v1/search/semantic", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		// Execute
		rr := httptest.NewRecorder()
		handler.SemanticSearch(rr, req)

		// Verify
		assert.Equal(t, http.StatusOK, rr.Code)
		
		var response models.SemanticSearchResponse
		err := json.Unmarshal(rr.Body.Bytes(), &response)
		require.NoError(t, err)
		
		assert.Equal(t, expectedResponse.Query, response.Query)
		assert.Equal(t, expectedResponse.TotalCount, response.TotalCount)
		assert.Len(t, response.Results, 1)
		assert.Equal(t, testChunk.ID, response.Results[0].Chunk.ID)
		assert.Equal(t, 0.95, response.Results[0].Similarity)
	})

	t.Run("GraphSearchAPI_Integration", func(t *testing.T) {
		// Setup mock response
		expectedResponse := &models.GraphResult{
			Nodes: []models.GraphNode{testGraphNode},
			Edges: []models.GraphEdge{testGraphEdge},
		}

		mockService.On("GraphSearch", 
			context.Background(), 
			&models.GraphQuery{
				EntityName: "Test Entity",
				MaxDepth:   2,
				Limit:      10,
			}).Return(expectedResponse, nil)

		// Create request
		requestBody := models.GraphQuery{
			EntityName: "Test Entity",
			MaxDepth:   2,
			Limit:      10,
		}
		body, _ := json.Marshal(requestBody)
		req := httptest.NewRequest("POST", "/api/v1/search/graph", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		// Execute
		rr := httptest.NewRecorder()
		handler.GraphSearch(rr, req)

		// Verify
		assert.Equal(t, http.StatusOK, rr.Code)
		
		var response models.GraphResult
		err := json.Unmarshal(rr.Body.Bytes(), &response)
		require.NoError(t, err)
		
		assert.Len(t, response.Nodes, 1)
		assert.Len(t, response.Edges, 1)
		assert.Equal(t, testGraphNode.EntityName, response.Nodes[0].EntityName)
		assert.Equal(t, testGraphEdge.RelationshipType, response.Edges[0].RelationshipType)
	})

	t.Run("TagSearchAPI_Integration", func(t *testing.T) {
		// Setup mock response
		expectedResponse := []models.ChunkWithTags{
			{
				Chunk: &testChunk,
				Tags:  []models.ChunkRecord{testTag},
			},
		}

		mockService.On("SearchByTag", 
			context.Background(), 
			"important").Return(expectedResponse, nil)

		// Create request
		requestBody := models.TagSearchRequest{
			TagContent: "important",
		}
		body, _ := json.Marshal(requestBody)
		req := httptest.NewRequest("POST", "/api/v1/search/tags", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		// Execute
		rr := httptest.NewRecorder()
		handler.SearchByTag(rr, req)

		// Verify
		assert.Equal(t, http.StatusOK, rr.Code)
		
		var response []models.ChunkWithTags
		err := json.Unmarshal(rr.Body.Bytes(), &response)
		require.NoError(t, err)
		
		assert.Len(t, response, 1)
		assert.Equal(t, testChunk.ID, response[0].Chunk.ID)
		assert.Len(t, response[0].Tags, 1)
		assert.Equal(t, testTag.Content, response[0].Tags[0].Content)
	})

	t.Run("ChunkSearchAPI_Integration", func(t *testing.T) {
		// Setup mock response
		expectedResponse := []models.ChunkRecord{testChunk}

		mockService.On("SearchChunks", 
			context.Background(), 
			"test query",
			map[string]interface{}{"text_id": "text-456"}).Return(expectedResponse, nil)

		// Create request
		requestBody := models.ChunkSearchRequest{
			Query: "test query",
			Filters: map[string]interface{}{
				"text_id": "text-456",
			},
		}
		body, _ := json.Marshal(requestBody)
		req := httptest.NewRequest("POST", "/api/v1/search/chunks", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		// Execute
		rr := httptest.NewRecorder()
		handler.SearchChunks(rr, req)

		// Verify
		assert.Equal(t, http.StatusOK, rr.Code)
		
		var response []models.ChunkRecord
		err := json.Unmarshal(rr.Body.Bytes(), &response)
		require.NoError(t, err)
		
		assert.Len(t, response, 1)
		assert.Equal(t, testChunk.ID, response[0].ID)
		assert.Equal(t, testChunk.Content, response[0].Content)
	})

	t.Run("HybridSearchAPI_Integration", func(t *testing.T) {
		// Setup mock response
		expectedResponse := []models.SimilarityResult{
			{
				Chunk:      testChunk,
				Similarity: 0.88, // Hybrid score combining semantic and text search
			},
		}

		mockService.On("HybridSearch", 
			context.Background(), 
			"hybrid query",
			10,
			0.7).Return(expectedResponse, nil)

		// Create request
		requestBody := models.HybridSearchRequest{
			Query:          "hybrid query",
			Limit:          10,
			SemanticWeight: 0.7,
		}
		body, _ := json.Marshal(requestBody)
		req := httptest.NewRequest("POST", "/api/v1/search/hybrid", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		// Execute
		rr := httptest.NewRecorder()
		handler.HybridSearch(rr, req)

		// Verify
		assert.Equal(t, http.StatusOK, rr.Code)
		
		var response []models.SimilarityResult
		err := json.Unmarshal(rr.Body.Bytes(), &response)
		require.NoError(t, err)
		
		assert.Len(t, response, 1)
		assert.Equal(t, testChunk.ID, response[0].Chunk.ID)
		assert.Equal(t, 0.88, response[0].Similarity)
	})
}

// Test error handling scenarios
func TestSearchAPIsErrorHandling(t *testing.T) {
	mockService := &MockSearchService{}
	handler := NewSearchHandler(mockService)

	t.Run("SemanticSearch_ServiceError", func(t *testing.T) {
		mockService.On("SemanticSearchWithFilters", 
			context.Background(), 
			&models.SemanticSearchRequest{
				Query: "error query",
				Limit: 10,
			}).Return((*models.SemanticSearchResponse)(nil), assert.AnError)

		requestBody := models.SemanticSearchRequest{
			Query: "error query",
			Limit: 10,
		}
		body, _ := json.Marshal(requestBody)
		req := httptest.NewRequest("POST", "/api/v1/search/semantic", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		handler.SemanticSearch(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		assert.Contains(t, rr.Body.String(), "failed to perform semantic search")
	})

	t.Run("GraphSearch_ServiceError", func(t *testing.T) {
		mockService.On("GraphSearch", 
			context.Background(), 
			&models.GraphQuery{
				EntityName: "Error Entity",
				MaxDepth:   2,
				Limit:      10,
			}).Return((*models.GraphResult)(nil), assert.AnError)

		requestBody := models.GraphQuery{
			EntityName: "Error Entity",
			MaxDepth:   2,
			Limit:      10,
		}
		body, _ := json.Marshal(requestBody)
		req := httptest.NewRequest("POST", "/api/v1/search/graph", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		handler.GraphSearch(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		assert.Contains(t, rr.Body.String(), "failed to perform graph search")
	})

	t.Run("TagSearch_ServiceError", func(t *testing.T) {
		mockService.On("SearchByTag", 
			context.Background(), 
			"error tag").Return(([]models.ChunkWithTags)(nil), assert.AnError)

		requestBody := models.TagSearchRequest{
			TagContent: "error tag",
		}
		body, _ := json.Marshal(requestBody)
		req := httptest.NewRequest("POST", "/api/v1/search/tags", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		handler.SearchByTag(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		assert.Contains(t, rr.Body.String(), "failed to search by tag")
	})
}