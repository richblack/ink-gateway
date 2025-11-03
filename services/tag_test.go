package services

import (
	"context"
	"fmt"
	"testing"
	"time"

	"semantic-text-processor/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockSupabaseClientForTag for testing tag service
type MockSupabaseClientForTag struct {
	mock.Mock
}

func (m *MockSupabaseClientForTag) AddTag(ctx context.Context, chunkID string, tagContent string) error {
	args := m.Called(ctx, chunkID, tagContent)
	return args.Error(0)
}

func (m *MockSupabaseClientForTag) RemoveTag(ctx context.Context, chunkID string, tagChunkID string) error {
	args := m.Called(ctx, chunkID, tagChunkID)
	return args.Error(0)
}

func (m *MockSupabaseClientForTag) GetChunkTags(ctx context.Context, chunkID string) ([]models.ChunkRecord, error) {
	args := m.Called(ctx, chunkID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.ChunkRecord), args.Error(1)
}

func (m *MockSupabaseClientForTag) GetChunksByTag(ctx context.Context, tagContent string) ([]models.ChunkRecord, error) {
	args := m.Called(ctx, tagContent)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.ChunkRecord), args.Error(1)
}



// Implement other required methods as no-ops for this test
func (m *MockSupabaseClientForTag) InsertText(ctx context.Context, text *models.TextRecord) error { return nil }
func (m *MockSupabaseClientForTag) GetTexts(ctx context.Context, pagination *models.Pagination) (*models.TextList, error) { return nil, nil }
func (m *MockSupabaseClientForTag) GetTextByID(ctx context.Context, id string) (*models.TextDetail, error) { return nil, nil }
func (m *MockSupabaseClientForTag) UpdateText(ctx context.Context, text *models.TextRecord) error { return nil }
func (m *MockSupabaseClientForTag) DeleteText(ctx context.Context, id string) error { return nil }
func (m *MockSupabaseClientForTag) InsertChunk(ctx context.Context, chunk *models.ChunkRecord) error { return nil }
func (m *MockSupabaseClientForTag) InsertChunks(ctx context.Context, chunks []models.ChunkRecord) error { return nil }
func (m *MockSupabaseClientForTag) GetChunkByID(ctx context.Context, id string) (*models.ChunkRecord, error) { return nil, nil }
func (m *MockSupabaseClientForTag) GetChunkByContent(ctx context.Context, content string) (*models.ChunkRecord, error) { return nil, nil }
func (m *MockSupabaseClientForTag) UpdateChunk(ctx context.Context, chunk *models.ChunkRecord) error { return nil }
func (m *MockSupabaseClientForTag) DeleteChunk(ctx context.Context, id string) error { return nil }
func (m *MockSupabaseClientForTag) GetChunksByTextID(ctx context.Context, textID string) ([]models.ChunkRecord, error) { return nil, nil }
func (m *MockSupabaseClientForTag) CreateTemplate(ctx context.Context, templateName string, slotNames []string) (*models.TemplateWithInstances, error) { return nil, nil }
func (m *MockSupabaseClientForTag) GetTemplateByContent(ctx context.Context, templateContent string) (*models.TemplateWithInstances, error) { return nil, nil }
func (m *MockSupabaseClientForTag) GetAllTemplates(ctx context.Context) ([]models.TemplateWithInstances, error) { return nil, nil }
func (m *MockSupabaseClientForTag) CreateTemplateInstance(ctx context.Context, req *models.CreateInstanceRequest) (*models.TemplateInstance, error) { return nil, nil }
func (m *MockSupabaseClientForTag) GetTemplateInstances(ctx context.Context, templateChunkID string) ([]models.TemplateInstance, error) { return nil, nil }
func (m *MockSupabaseClientForTag) UpdateSlotValue(ctx context.Context, instanceChunkID, slotName, value string) error { return nil }
func (m *MockSupabaseClientForTag) GetChunkHierarchy(ctx context.Context, rootChunkID string) (*models.ChunkHierarchy, error) { return nil, nil }
func (m *MockSupabaseClientForTag) GetChildrenChunks(ctx context.Context, parentChunkID string) ([]models.ChunkRecord, error) { return nil, nil }
func (m *MockSupabaseClientForTag) GetSiblingChunks(ctx context.Context, chunkID string) ([]models.ChunkRecord, error) { return nil, nil }
func (m *MockSupabaseClientForTag) MoveChunk(ctx context.Context, req *models.MoveChunkRequest) error { return nil }
func (m *MockSupabaseClientForTag) BulkUpdateChunks(ctx context.Context, req *models.BulkUpdateRequest) error { return nil }
func (m *MockSupabaseClientForTag) SearchChunks(ctx context.Context, query string, filters map[string]interface{}) ([]models.ChunkRecord, error) { return nil, nil }
func (m *MockSupabaseClientForTag) SearchByTag(ctx context.Context, tagContent string) ([]models.ChunkWithTags, error) { return nil, nil }
func (m *MockSupabaseClientForTag) InsertEmbeddings(ctx context.Context, embeddings []models.EmbeddingRecord) error { return nil }
func (m *MockSupabaseClientForTag) SearchSimilar(ctx context.Context, queryVector []float64, limit int) ([]models.SimilarityResult, error) { return nil, nil }
func (m *MockSupabaseClientForTag) InsertGraphNodes(ctx context.Context, nodes []models.GraphNode) error { return nil }
func (m *MockSupabaseClientForTag) InsertGraphEdges(ctx context.Context, edges []models.GraphEdge) error { return nil }
func (m *MockSupabaseClientForTag) SearchGraph(ctx context.Context, query *models.GraphQuery) (*models.GraphResult, error) { return nil, nil }
func (m *MockSupabaseClientForTag) GetNodesByEntity(ctx context.Context, entityName string) ([]models.GraphNode, error) { return nil, nil }
func (m *MockSupabaseClientForTag) GetNodeNeighbors(ctx context.Context, nodeID string, maxDepth int) (*models.GraphResult, error) { return nil, nil }
func (m *MockSupabaseClientForTag) FindPathBetweenNodes(ctx context.Context, sourceNodeID, targetNodeID string, maxDepth int) (*models.GraphResult, error) { return nil, nil }
func (m *MockSupabaseClientForTag) GetNodesByChunk(ctx context.Context, chunkID string) ([]models.GraphNode, error) { return nil, nil }
func (m *MockSupabaseClientForTag) GetEdgesByRelationType(ctx context.Context, relationType string) ([]models.GraphEdge, error) { return nil, nil }
func (m *MockSupabaseClientForTag) HealthCheck(ctx context.Context) error { return nil }

func TestTagService_AddTag(t *testing.T) {
	tests := []struct {
		name          string
		chunkID       string
		tagContent    string
		mockError     error
		expectedError string
	}{
		{
			name:          "successful tag addition",
			chunkID:       "chunk-123",
			tagContent:    "important",
			mockError:     nil,
			expectedError: "",
		},
		{
			name:          "empty chunk ID",
			chunkID:       "",
			tagContent:    "important",
			mockError:     nil,
			expectedError: "chunk ID is required",
		},
		{
			name:          "empty tag content",
			chunkID:       "chunk-123",
			tagContent:    "",
			mockError:     nil,
			expectedError: "tag content is required",
		},
		{
			name:          "supabase client error",
			chunkID:       "chunk-123",
			tagContent:    "important",
			mockError:     fmt.Errorf("database error"),
			expectedError: "database error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockClient := new(MockSupabaseClientForTag)
			service := NewTagService(mockClient)

			// Setup mock expectations
			if tt.chunkID != "" && tt.tagContent != "" {
				mockClient.On("AddTag", mock.Anything, tt.chunkID, tt.tagContent).Return(tt.mockError)
			}

			// Execute
			err := service.AddTag(context.Background(), tt.chunkID, tt.tagContent)

			// Assert
			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
			}

			mockClient.AssertExpectations(t)
		})
	}
}

func TestTagService_RemoveTag(t *testing.T) {
	tests := []struct {
		name          string
		chunkID       string
		tagChunkID    string
		mockError     error
		expectedError string
	}{
		{
			name:          "successful tag removal",
			chunkID:       "chunk-123",
			tagChunkID:    "tag-456",
			mockError:     nil,
			expectedError: "",
		},
		{
			name:          "empty chunk ID",
			chunkID:       "",
			tagChunkID:    "tag-456",
			mockError:     nil,
			expectedError: "chunk ID is required",
		},
		{
			name:          "empty tag chunk ID",
			chunkID:       "chunk-123",
			tagChunkID:    "",
			mockError:     nil,
			expectedError: "tag chunk ID is required",
		},
		{
			name:          "supabase client error",
			chunkID:       "chunk-123",
			tagChunkID:    "tag-456",
			mockError:     fmt.Errorf("database error"),
			expectedError: "database error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockClient := new(MockSupabaseClientForTag)
			service := NewTagService(mockClient)

			// Setup mock expectations
			if tt.chunkID != "" && tt.tagChunkID != "" {
				mockClient.On("RemoveTag", mock.Anything, tt.chunkID, tt.tagChunkID).Return(tt.mockError)
			}

			// Execute
			err := service.RemoveTag(context.Background(), tt.chunkID, tt.tagChunkID)

			// Assert
			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
			}

			mockClient.AssertExpectations(t)
		})
	}
}

func TestTagService_GetChunkTags(t *testing.T) {
	tests := []struct {
		name          string
		chunkID       string
		mockResponse  []models.ChunkRecord
		mockError     error
		expectedError string
		expectedCount int
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
			mockError:     nil,
			expectedError: "",
			expectedCount: 2,
		},
		{
			name:          "empty chunk ID",
			chunkID:       "",
			mockResponse:  nil,
			mockError:     nil,
			expectedError: "chunk ID is required",
			expectedCount: 0,
		},
		{
			name:          "no tags found",
			chunkID:       "chunk-123",
			mockResponse:  []models.ChunkRecord{},
			mockError:     nil,
			expectedError: "",
			expectedCount: 0,
		},
		{
			name:          "supabase client error",
			chunkID:       "chunk-123",
			mockResponse:  nil,
			mockError:     fmt.Errorf("database error"),
			expectedError: "database error",
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockClient := new(MockSupabaseClientForTag)
			service := NewTagService(mockClient)

			// Setup mock expectations
			if tt.chunkID != "" {
				mockClient.On("GetChunkTags", mock.Anything, tt.chunkID).Return(tt.mockResponse, tt.mockError)
			}

			// Execute
			result, err := service.GetChunkTags(context.Background(), tt.chunkID)

			// Assert
			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedCount, len(result))
			}

			mockClient.AssertExpectations(t)
		})
	}
}

func TestTagService_GetChunksByTag(t *testing.T) {
	tests := []struct {
		name          string
		tagContent    string
		mockResponse  []models.ChunkRecord
		mockError     error
		expectedError string
		expectedCount int
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
			mockError:     nil,
			expectedError: "",
			expectedCount: 2,
		},
		{
			name:          "empty tag content",
			tagContent:    "",
			mockResponse:  nil,
			mockError:     nil,
			expectedError: "tag content is required",
			expectedCount: 0,
		},
		{
			name:          "no chunks found",
			tagContent:    "nonexistent",
			mockResponse:  []models.ChunkRecord{},
			mockError:     nil,
			expectedError: "",
			expectedCount: 0,
		},
		{
			name:          "supabase client error",
			tagContent:    "important",
			mockResponse:  nil,
			mockError:     fmt.Errorf("database error"),
			expectedError: "database error",
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockClient := new(MockSupabaseClientForTag)
			service := NewTagService(mockClient)

			// Setup mock expectations
			if tt.tagContent != "" {
				mockClient.On("GetChunksByTag", mock.Anything, tt.tagContent).Return(tt.mockResponse, tt.mockError)
			}

			// Execute
			result, err := service.GetChunksByTag(context.Background(), tt.tagContent)

			// Assert
			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedCount, len(result))
			}

			mockClient.AssertExpectations(t)
		})
	}
}

func TestTagService_AddTagWithInheritance(t *testing.T) {
	tests := []struct {
		name          string
		chunkID       string
		tagContent    string
		mockError     error
		expectedError string
	}{
		{
			name:          "empty chunk ID",
			chunkID:       "",
			tagContent:    "important",
			mockError:     nil,
			expectedError: "chunk ID is required",
		},
		{
			name:          "empty tag content",
			chunkID:       "chunk-123",
			tagContent:    "",
			mockError:     nil,
			expectedError: "tag content is required",
		},
		{
			name:          "add tag error",
			chunkID:       "chunk-123",
			tagContent:    "important",
			mockError:     fmt.Errorf("database error"),
			expectedError: "failed to add tag to chunk",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockClient := new(MockSupabaseClientForTag)
			service := NewTagService(mockClient)

			// Setup mock expectations
			if tt.chunkID != "" && tt.tagContent != "" {
				// Expect AddTag call for parent
				mockClient.On("AddTag", mock.Anything, tt.chunkID, tt.tagContent).Return(tt.mockError)
				
				if tt.mockError == nil {
					// Expect GetChildrenChunks call (return empty for simplicity)
					mockClient.On("GetChildrenChunks", mock.Anything, tt.chunkID).Return([]models.ChunkRecord{}, nil)
				}
			}

			// Execute
			err := service.AddTagWithInheritance(context.Background(), tt.chunkID, tt.tagContent)

			// Assert
			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
			}

			mockClient.AssertExpectations(t)
		})
	}
}

func TestTagService_GetInheritedTags(t *testing.T) {
	tests := []struct {
		name          string
		chunkID       string
		expectedError string
	}{
		{
			name:          "empty chunk ID",
			chunkID:       "",
			expectedError: "chunk ID is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockClient := new(MockSupabaseClientForTag)
			service := NewTagService(mockClient)

			// Execute
			result, err := service.GetInheritedTags(context.Background(), tt.chunkID)

			// Assert
			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
			}

			mockClient.AssertExpectations(t)
		})
	}
}

func TestTagService_DeduplicateTags(t *testing.T) {
	service := &tagService{}
	
	tags := []models.ChunkRecord{
		{ID: "tag-1", Content: "tag1"},
		{ID: "tag-2", Content: "tag2"},
		{ID: "tag-1", Content: "tag1"}, // duplicate
		{ID: "tag-3", Content: "tag3"},
		{ID: "tag-2", Content: "tag2"}, // duplicate
	}
	
	result := service.deduplicateTags(tags)
	
	assert.Equal(t, 3, len(result))
	
	// Check that all unique IDs are present
	ids := make(map[string]bool)
	for _, tag := range result {
		ids[tag.ID] = true
	}
	
	assert.True(t, ids["tag-1"])
	assert.True(t, ids["tag-2"])
	assert.True(t, ids["tag-3"])
}