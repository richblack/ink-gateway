package services

import (
	"context"
	"testing"

	"semantic-text-processor/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockSupabaseClient for testing search service
type MockSupabaseClient struct {
	searchSimilarFunc func(ctx context.Context, queryVector []float64, limit int) ([]models.SimilarityResult, error)
	searchByTagFunc   func(ctx context.Context, tagContent string) ([]models.ChunkWithTags, error)
	searchChunksFunc  func(ctx context.Context, query string, filters map[string]interface{}) ([]models.ChunkRecord, error)
	searchGraphFunc   func(ctx context.Context, query *models.GraphQuery) (*models.GraphResult, error)
}

func (m *MockSupabaseClient) SearchSimilar(ctx context.Context, queryVector []float64, limit int) ([]models.SimilarityResult, error) {
	if m.searchSimilarFunc != nil {
		return m.searchSimilarFunc(ctx, queryVector, limit)
	}
	return []models.SimilarityResult{}, nil
}

func (m *MockSupabaseClient) SearchByTag(ctx context.Context, tagContent string) ([]models.ChunkWithTags, error) {
	if m.searchByTagFunc != nil {
		return m.searchByTagFunc(ctx, tagContent)
	}
	return []models.ChunkWithTags{}, nil
}

func (m *MockSupabaseClient) SearchChunks(ctx context.Context, query string, filters map[string]interface{}) ([]models.ChunkRecord, error) {
	if m.searchChunksFunc != nil {
		return m.searchChunksFunc(ctx, query, filters)
	}
	return []models.ChunkRecord{}, nil
}

func (m *MockSupabaseClient) SearchGraph(ctx context.Context, query *models.GraphQuery) (*models.GraphResult, error) {
	if m.searchGraphFunc != nil {
		return m.searchGraphFunc(ctx, query)
	}
	return &models.GraphResult{}, nil
}

// Stub implementations for other interface methods
func (m *MockSupabaseClient) InsertText(ctx context.Context, text *models.TextRecord) error { return nil }
func (m *MockSupabaseClient) GetTexts(ctx context.Context, pagination *models.Pagination) (*models.TextList, error) { return nil, nil }
func (m *MockSupabaseClient) GetTextByID(ctx context.Context, id string) (*models.TextDetail, error) { return nil, nil }
func (m *MockSupabaseClient) UpdateText(ctx context.Context, text *models.TextRecord) error { return nil }
func (m *MockSupabaseClient) DeleteText(ctx context.Context, id string) error { return nil }
func (m *MockSupabaseClient) InsertChunk(ctx context.Context, chunk *models.ChunkRecord) error { return nil }
func (m *MockSupabaseClient) InsertChunks(ctx context.Context, chunks []models.ChunkRecord) error { return nil }
func (m *MockSupabaseClient) GetChunkByID(ctx context.Context, id string) (*models.ChunkRecord, error) { return nil, nil }
func (m *MockSupabaseClient) GetChunkByContent(ctx context.Context, content string) (*models.ChunkRecord, error) { return nil, nil }
func (m *MockSupabaseClient) UpdateChunk(ctx context.Context, chunk *models.ChunkRecord) error { return nil }
func (m *MockSupabaseClient) DeleteChunk(ctx context.Context, id string) error { return nil }
func (m *MockSupabaseClient) GetChunksByTextID(ctx context.Context, textID string) ([]models.ChunkRecord, error) { return nil, nil }
func (m *MockSupabaseClient) CreateTemplate(ctx context.Context, templateName string, slotNames []string) (*models.TemplateWithInstances, error) { return nil, nil }
func (m *MockSupabaseClient) GetTemplateByContent(ctx context.Context, templateContent string) (*models.TemplateWithInstances, error) { return nil, nil }
func (m *MockSupabaseClient) GetAllTemplates(ctx context.Context) ([]models.TemplateWithInstances, error) { return nil, nil }
func (m *MockSupabaseClient) CreateTemplateInstance(ctx context.Context, req *models.CreateInstanceRequest) (*models.TemplateInstance, error) { return nil, nil }
func (m *MockSupabaseClient) GetTemplateInstances(ctx context.Context, templateChunkID string) ([]models.TemplateInstance, error) { return nil, nil }
func (m *MockSupabaseClient) UpdateSlotValue(ctx context.Context, instanceChunkID, slotName, value string) error { return nil }
func (m *MockSupabaseClient) AddTag(ctx context.Context, chunkID string, tagContent string) error { return nil }
func (m *MockSupabaseClient) RemoveTag(ctx context.Context, chunkID string, tagChunkID string) error { return nil }
func (m *MockSupabaseClient) GetChunkTags(ctx context.Context, chunkID string) ([]models.ChunkRecord, error) { return nil, nil }
func (m *MockSupabaseClient) GetChunksByTag(ctx context.Context, tagContent string) ([]models.ChunkRecord, error) { return nil, nil }
func (m *MockSupabaseClient) GetChunkHierarchy(ctx context.Context, rootChunkID string) (*models.ChunkHierarchy, error) { return nil, nil }
func (m *MockSupabaseClient) GetChildrenChunks(ctx context.Context, parentChunkID string) ([]models.ChunkRecord, error) { return nil, nil }
func (m *MockSupabaseClient) GetSiblingChunks(ctx context.Context, chunkID string) ([]models.ChunkRecord, error) { return nil, nil }
func (m *MockSupabaseClient) MoveChunk(ctx context.Context, req *models.MoveChunkRequest) error { return nil }
func (m *MockSupabaseClient) BulkUpdateChunks(ctx context.Context, req *models.BulkUpdateRequest) error { return nil }
func (m *MockSupabaseClient) InsertEmbeddings(ctx context.Context, embeddings []models.EmbeddingRecord) error { return nil }
func (m *MockSupabaseClient) InsertGraphNodes(ctx context.Context, nodes []models.GraphNode) error { return nil }
func (m *MockSupabaseClient) InsertGraphEdges(ctx context.Context, edges []models.GraphEdge) error { return nil }
func (m *MockSupabaseClient) GetNodesByEntity(ctx context.Context, entityName string) ([]models.GraphNode, error) { return nil, nil }
func (m *MockSupabaseClient) GetNodeNeighbors(ctx context.Context, nodeID string, maxDepth int) (*models.GraphResult, error) { return nil, nil }
func (m *MockSupabaseClient) FindPathBetweenNodes(ctx context.Context, sourceNodeID, targetNodeID string, maxDepth int) (*models.GraphResult, error) { return nil, nil }
func (m *MockSupabaseClient) GetNodesByChunk(ctx context.Context, chunkID string) ([]models.GraphNode, error) { return nil, nil }
func (m *MockSupabaseClient) GetEdgesByRelationType(ctx context.Context, relationType string) ([]models.GraphEdge, error) { return nil, nil }
func (m *MockSupabaseClient) HealthCheck(ctx context.Context) error { return nil }

func TestSearchService_SemanticSearch(t *testing.T) {
	tests := []struct {
		name           string
		query          string
		limit          int
		embeddingError error
		searchResults  []models.SimilarityResult
		searchError    error
		expectedError  bool
		expectedCount  int
	}{
		{
			name:  "successful search",
			query: "test query",
			limit: 5,
			searchResults: []models.SimilarityResult{
				{
					Chunk: models.ChunkRecord{
						ID:      "chunk1",
						Content: "First test chunk",
					},
					Similarity: 0.9,
				},
				{
					Chunk: models.ChunkRecord{
						ID:      "chunk2",
						Content: "Second test chunk",
					},
					Similarity: 0.8,
				},
			},
			expectedError: false,
			expectedCount: 2,
		},
		{
			name:          "empty query",
			query:         "",
			limit:         5,
			expectedError: false,
			expectedCount: 0,
		},
		{
			name:           "embedding generation error",
			query:          "test query",
			limit:          5,
			embeddingError: assert.AnError,
			expectedError:  true,
		},
		{
			name:        "search error",
			query:       "test query",
			limit:       5,
			searchError: assert.AnError,
			expectedError: true,
		},
		{
			name:          "zero limit uses default",
			query:         "test query",
			limit:         0,
			searchResults: []models.SimilarityResult{},
			expectedError: false,
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mocks
			mockSupabase := &MockSupabaseClient{
				searchSimilarFunc: func(ctx context.Context, queryVector []float64, limit int) ([]models.SimilarityResult, error) {
					if tt.searchError != nil {
						return nil, tt.searchError
					}
					return tt.searchResults, nil
				},
			}

			mockEmbedding := NewTestEmbeddingService()
			if tt.embeddingError != nil {
				mockEmbedding.SetShouldFail(true)
			}

			// Create search service
			searchService := NewSearchService(mockSupabase, mockEmbedding)

			// Test
			ctx := context.Background()
			results, err := searchService.SemanticSearch(ctx, tt.query, tt.limit)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, results)
			} else {
				assert.NoError(t, err)
				assert.Len(t, results, tt.expectedCount)
			}
		})
	}
}

func TestSearchService_SemanticSearchWithFilters(t *testing.T) {
	mockSupabase := &MockSupabaseClient{
		searchSimilarFunc: func(ctx context.Context, queryVector []float64, limit int) ([]models.SimilarityResult, error) {
			return []models.SimilarityResult{
				{
					Chunk: models.ChunkRecord{
						ID:          "chunk1",
						Content:     "First chunk",
						TextID:      "text1",
						IsTemplate:  false,
						IndentLevel: 0,
					},
					Similarity: 0.9,
				},
				{
					Chunk: models.ChunkRecord{
						ID:          "chunk2",
						Content:     "Second chunk",
						TextID:      "text2",
						IsTemplate:  true,
						IndentLevel: 1,
					},
					Similarity: 0.8,
				},
				{
					Chunk: models.ChunkRecord{
						ID:          "chunk3",
						Content:     "Third chunk",
						TextID:      "text1",
						IsTemplate:  false,
						IndentLevel: 2,
					},
					Similarity: 0.7,
				},
			}, nil
		},
	}

	mockEmbedding := NewTestEmbeddingService()
	searchService := NewSearchService(mockSupabase, mockEmbedding)
	ctx := context.Background()

	t.Run("no filters", func(t *testing.T) {
		req := &models.SemanticSearchRequest{
			Query: "test query",
			Limit: 10,
		}

		response, err := searchService.SemanticSearchWithFilters(ctx, req)
		require.NoError(t, err)
		assert.Len(t, response.Results, 3)
		assert.Equal(t, 3, response.TotalCount)
	})

	t.Run("filter by text_id", func(t *testing.T) {
		req := &models.SemanticSearchRequest{
			Query: "test query",
			Limit: 10,
			Filters: map[string]interface{}{
				"text_id": "text1",
			},
		}

		response, err := searchService.SemanticSearchWithFilters(ctx, req)
		require.NoError(t, err)
		assert.Len(t, response.Results, 2)
		for _, result := range response.Results {
			assert.Equal(t, "text1", result.Chunk.TextID)
		}
	})

	t.Run("filter by is_template", func(t *testing.T) {
		req := &models.SemanticSearchRequest{
			Query: "test query",
			Limit: 10,
			Filters: map[string]interface{}{
				"is_template": true,
			},
		}

		response, err := searchService.SemanticSearchWithFilters(ctx, req)
		require.NoError(t, err)
		assert.Len(t, response.Results, 1)
		assert.True(t, response.Results[0].Chunk.IsTemplate)
	})

	t.Run("filter by indent level range", func(t *testing.T) {
		req := &models.SemanticSearchRequest{
			Query: "test query",
			Limit: 10,
			Filters: map[string]interface{}{
				"min_indent_level": 1,
				"max_indent_level": 2,
			},
		}

		response, err := searchService.SemanticSearchWithFilters(ctx, req)
		require.NoError(t, err)
		assert.Len(t, response.Results, 2)
		for _, result := range response.Results {
			assert.GreaterOrEqual(t, result.Chunk.IndentLevel, 1)
			assert.LessOrEqual(t, result.Chunk.IndentLevel, 2)
		}
	})

	t.Run("minimum similarity threshold", func(t *testing.T) {
		req := &models.SemanticSearchRequest{
			Query:         "test query",
			Limit:         10,
			MinSimilarity: 0.75,
		}

		response, err := searchService.SemanticSearchWithFilters(ctx, req)
		require.NoError(t, err)
		assert.Len(t, response.Results, 2) // Only chunks with similarity >= 0.75
		for _, result := range response.Results {
			assert.GreaterOrEqual(t, result.Similarity, 0.75)
		}
	})

	t.Run("empty query", func(t *testing.T) {
		req := &models.SemanticSearchRequest{
			Query: "",
			Limit: 10,
		}

		response, err := searchService.SemanticSearchWithFilters(ctx, req)
		require.NoError(t, err)
		assert.Len(t, response.Results, 0)
		assert.Equal(t, 0, response.TotalCount)
	})
}

func TestSearchService_HybridSearch(t *testing.T) {
	mockSupabase := &MockSupabaseClient{
		searchSimilarFunc: func(ctx context.Context, queryVector []float64, limit int) ([]models.SimilarityResult, error) {
			return []models.SimilarityResult{
				{
					Chunk: models.ChunkRecord{
						ID:      "chunk1",
						Content: "Semantic match",
					},
					Similarity: 0.9,
				},
				{
					Chunk: models.ChunkRecord{
						ID:      "chunk2",
						Content: "Another semantic match",
					},
					Similarity: 0.8,
				},
			}, nil
		},
		searchChunksFunc: func(ctx context.Context, query string, filters map[string]interface{}) ([]models.ChunkRecord, error) {
			return []models.ChunkRecord{
				{
					ID:      "chunk2", // Overlapping with semantic results
					Content: "Text match overlap",
				},
				{
					ID:      "chunk3",
					Content: "Pure text match",
				},
			}, nil
		},
	}

	mockEmbedding := NewTestEmbeddingService()
	searchService := NewSearchService(mockSupabase, mockEmbedding)
	ctx := context.Background()

	t.Run("balanced hybrid search", func(t *testing.T) {
		results, err := searchService.HybridSearch(ctx, "test query", 5, 0.5)
		require.NoError(t, err)
		assert.LessOrEqual(t, len(results), 5)
		
		// Verify results are sorted by combined similarity
		for i := 1; i < len(results); i++ {
			assert.GreaterOrEqual(t, results[i-1].Similarity, results[i].Similarity)
		}
	})

	t.Run("semantic-heavy search", func(t *testing.T) {
		results, err := searchService.HybridSearch(ctx, "test query", 5, 0.8)
		require.NoError(t, err)
		assert.LessOrEqual(t, len(results), 5)
	})

	t.Run("text-heavy search", func(t *testing.T) {
		results, err := searchService.HybridSearch(ctx, "test query", 5, 0.2)
		require.NoError(t, err)
		assert.LessOrEqual(t, len(results), 5)
	})

	t.Run("invalid semantic weight", func(t *testing.T) {
		_, err := searchService.HybridSearch(ctx, "test query", 5, 1.5)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "semantic weight must be between 0 and 1")
	})
}

func TestSearchService_SearchByTag(t *testing.T) {
	expectedResults := []models.ChunkWithTags{
		{
			Chunk: &models.ChunkRecord{
				ID:      "chunk1",
				Content: "Tagged chunk",
			},
			Tags: []models.ChunkRecord{
				{
					ID:      "tag1",
					Content: "important",
				},
			},
		},
	}

	mockSupabase := &MockSupabaseClient{
		searchByTagFunc: func(ctx context.Context, tagContent string) ([]models.ChunkWithTags, error) {
			if tagContent == "important" {
				return expectedResults, nil
			}
			return []models.ChunkWithTags{}, nil
		},
	}

	mockEmbedding := NewTestEmbeddingService()
	searchService := NewSearchService(mockSupabase, mockEmbedding)
	ctx := context.Background()

	t.Run("successful tag search", func(t *testing.T) {
		results, err := searchService.SearchByTag(ctx, "important")
		require.NoError(t, err)
		assert.Equal(t, expectedResults, results)
	})

	t.Run("no results for tag", func(t *testing.T) {
		results, err := searchService.SearchByTag(ctx, "nonexistent")
		require.NoError(t, err)
		assert.Empty(t, results)
	})
}

func TestSearchService_SearchChunks(t *testing.T) {
	expectedResults := []models.ChunkRecord{
		{
			ID:      "chunk1",
			Content: "Matching chunk content",
		},
	}

	mockSupabase := &MockSupabaseClient{
		searchChunksFunc: func(ctx context.Context, query string, filters map[string]interface{}) ([]models.ChunkRecord, error) {
			if query == "matching" {
				return expectedResults, nil
			}
			return []models.ChunkRecord{}, nil
		},
	}

	mockEmbedding := NewTestEmbeddingService()
	searchService := NewSearchService(mockSupabase, mockEmbedding)
	ctx := context.Background()

	t.Run("successful text search", func(t *testing.T) {
		results, err := searchService.SearchChunks(ctx, "matching", nil)
		require.NoError(t, err)
		assert.Equal(t, expectedResults, results)
	})

	t.Run("no results for query", func(t *testing.T) {
		results, err := searchService.SearchChunks(ctx, "nonexistent", nil)
		require.NoError(t, err)
		assert.Empty(t, results)
	})
}