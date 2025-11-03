package handlers

import (
	"context"

	"semantic-text-processor/models"

	"github.com/stretchr/testify/mock"
)

// MockSupabaseClient for testing handlers
type MockSupabaseClient struct {
	mock.Mock
}

func (m *MockSupabaseClient) InsertText(ctx context.Context, text *models.TextRecord) error {
	args := m.Called(ctx, text)
	return args.Error(0)
}

func (m *MockSupabaseClient) GetTexts(ctx context.Context, pagination *models.Pagination) (*models.TextList, error) {
	args := m.Called(ctx, pagination)
	return args.Get(0).(*models.TextList), args.Error(1)
}

func (m *MockSupabaseClient) GetTextByID(ctx context.Context, id string) (*models.TextDetail, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*models.TextDetail), args.Error(1)
}

func (m *MockSupabaseClient) UpdateText(ctx context.Context, text *models.TextRecord) error {
	args := m.Called(ctx, text)
	return args.Error(0)
}

func (m *MockSupabaseClient) DeleteText(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockSupabaseClient) InsertChunk(ctx context.Context, chunk *models.ChunkRecord) error {
	args := m.Called(ctx, chunk)
	return args.Error(0)
}

func (m *MockSupabaseClient) InsertChunks(ctx context.Context, chunks []models.ChunkRecord) error {
	args := m.Called(ctx, chunks)
	return args.Error(0)
}

func (m *MockSupabaseClient) GetChunkByID(ctx context.Context, id string) (*models.ChunkRecord, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ChunkRecord), args.Error(1)
}

func (m *MockSupabaseClient) GetChunkByContent(ctx context.Context, content string) (*models.ChunkRecord, error) {
	args := m.Called(ctx, content)
	return args.Get(0).(*models.ChunkRecord), args.Error(1)
}

func (m *MockSupabaseClient) UpdateChunk(ctx context.Context, chunk *models.ChunkRecord) error {
	args := m.Called(ctx, chunk)
	return args.Error(0)
}

func (m *MockSupabaseClient) DeleteChunk(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockSupabaseClient) GetChunksByTextID(ctx context.Context, textID string) ([]models.ChunkRecord, error) {
	args := m.Called(ctx, textID)
	return args.Get(0).([]models.ChunkRecord), args.Error(1)
}

func (m *MockSupabaseClient) SearchChunks(ctx context.Context, query string, filters map[string]interface{}) ([]models.ChunkRecord, error) {
	args := m.Called(ctx, query, filters)
	return args.Get(0).([]models.ChunkRecord), args.Error(1)
}

func (m *MockSupabaseClient) GetChunkHierarchy(ctx context.Context, rootChunkID string) (*models.ChunkHierarchy, error) {
	args := m.Called(ctx, rootChunkID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ChunkHierarchy), args.Error(1)
}

func (m *MockSupabaseClient) GetChildrenChunks(ctx context.Context, parentChunkID string) ([]models.ChunkRecord, error) {
	args := m.Called(ctx, parentChunkID)
	return args.Get(0).([]models.ChunkRecord), args.Error(1)
}

func (m *MockSupabaseClient) GetSiblingChunks(ctx context.Context, chunkID string) ([]models.ChunkRecord, error) {
	args := m.Called(ctx, chunkID)
	return args.Get(0).([]models.ChunkRecord), args.Error(1)
}

func (m *MockSupabaseClient) MoveChunk(ctx context.Context, req *models.MoveChunkRequest) error {
	args := m.Called(ctx, req)
	return args.Error(0)
}

func (m *MockSupabaseClient) BulkUpdateChunks(ctx context.Context, req *models.BulkUpdateRequest) error {
	args := m.Called(ctx, req)
	return args.Error(0)
}

// Stub implementations for other interface methods
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
func (m *MockSupabaseClient) SearchByTag(ctx context.Context, tagContent string) ([]models.ChunkWithTags, error) { return nil, nil }
func (m *MockSupabaseClient) InsertEmbeddings(ctx context.Context, embeddings []models.EmbeddingRecord) error {
	args := m.Called(ctx, embeddings)
	return args.Error(0)
}
func (m *MockSupabaseClient) SearchSimilar(ctx context.Context, queryVector []float64, limit int) ([]models.SimilarityResult, error) { return nil, nil }
func (m *MockSupabaseClient) InsertGraphNodes(ctx context.Context, nodes []models.GraphNode) error {
	args := m.Called(ctx, nodes)
	return args.Error(0)
}
func (m *MockSupabaseClient) InsertGraphEdges(ctx context.Context, edges []models.GraphEdge) error {
	args := m.Called(ctx, edges)
	return args.Error(0)
}
func (m *MockSupabaseClient) SearchGraph(ctx context.Context, query *models.GraphQuery) (*models.GraphResult, error) { return nil, nil }
func (m *MockSupabaseClient) GetNodesByEntity(ctx context.Context, entityName string) ([]models.GraphNode, error) { return nil, nil }
func (m *MockSupabaseClient) GetNodeNeighbors(ctx context.Context, nodeID string, maxDepth int) (*models.GraphResult, error) { return nil, nil }
func (m *MockSupabaseClient) FindPathBetweenNodes(ctx context.Context, sourceNodeID, targetNodeID string, maxDepth int) (*models.GraphResult, error) { return nil, nil }
func (m *MockSupabaseClient) GetNodesByChunk(ctx context.Context, chunkID string) ([]models.GraphNode, error) { return nil, nil }
func (m *MockSupabaseClient) GetEdgesByRelationType(ctx context.Context, relationType string) ([]models.GraphEdge, error) { return nil, nil }
func (m *MockSupabaseClient) HealthCheck(ctx context.Context) error { return nil }

// Helper function to create int pointers
func intPtr(i int) *int {
	return &i
}

// Helper function to create string pointers
func stringPtr(s string) *string {
	return &s
}