package services

import (
	"context"
	"semantic-text-processor/models"
)

// TextProcessor handles text processing and chunking
type TextProcessor interface {
	ProcessText(ctx context.Context, text string) (*models.ProcessResult, error)
	ChunkText(ctx context.Context, text string) ([]models.ChunkRecord, error)
	GenerateEmbeddings(ctx context.Context, chunks []models.ChunkRecord) ([]models.EmbeddingRecord, error)
	ExtractKnowledge(ctx context.Context, chunks []models.ChunkRecord) (*models.GraphResult, error)
}

// SupabaseClient interface for dependency injection (re-export from clients)
type SupabaseClient interface {
	// Basic Text operations
	InsertText(ctx context.Context, text *models.TextRecord) error
	GetTexts(ctx context.Context, pagination *models.Pagination) (*models.TextList, error)
	GetTextByID(ctx context.Context, id string) (*models.TextDetail, error)
	UpdateText(ctx context.Context, text *models.TextRecord) error
	DeleteText(ctx context.Context, id string) error

	// Chunk operations
	InsertChunk(ctx context.Context, chunk *models.ChunkRecord) error
	InsertChunks(ctx context.Context, chunks []models.ChunkRecord) error
	GetChunkByID(ctx context.Context, id string) (*models.ChunkRecord, error)
	GetChunkByContent(ctx context.Context, content string) (*models.ChunkRecord, error)
	UpdateChunk(ctx context.Context, chunk *models.ChunkRecord) error
	DeleteChunk(ctx context.Context, id string) error
	GetChunksByTextID(ctx context.Context, textID string) ([]models.ChunkRecord, error)

	// Template operations
	CreateTemplate(ctx context.Context, templateName string, slotNames []string) (*models.TemplateWithInstances, error)
	GetTemplateByContent(ctx context.Context, templateContent string) (*models.TemplateWithInstances, error)
	GetAllTemplates(ctx context.Context) ([]models.TemplateWithInstances, error)

	// Template instance operations
	CreateTemplateInstance(ctx context.Context, req *models.CreateInstanceRequest) (*models.TemplateInstance, error)
	GetTemplateInstances(ctx context.Context, templateChunkID string) ([]models.TemplateInstance, error)
	UpdateSlotValue(ctx context.Context, instanceChunkID, slotName, value string) error

	// Tag operations
	AddTag(ctx context.Context, chunkID string, tagContent string) error
	RemoveTag(ctx context.Context, chunkID string, tagChunkID string) error
	GetChunkTags(ctx context.Context, chunkID string) ([]models.ChunkRecord, error)
	GetChunksByTag(ctx context.Context, tagContent string) ([]models.ChunkRecord, error)

	// Hierarchy operations
	GetChunkHierarchy(ctx context.Context, rootChunkID string) (*models.ChunkHierarchy, error)
	GetChildrenChunks(ctx context.Context, parentChunkID string) ([]models.ChunkRecord, error)
	GetSiblingChunks(ctx context.Context, chunkID string) ([]models.ChunkRecord, error)
	MoveChunk(ctx context.Context, req *models.MoveChunkRequest) error
	BulkUpdateChunks(ctx context.Context, req *models.BulkUpdateRequest) error

	// Search operations
	SearchChunks(ctx context.Context, query string, filters map[string]interface{}) ([]models.ChunkRecord, error)
	SearchByTag(ctx context.Context, tagContent string) ([]models.ChunkWithTags, error)

	// Vector operations
	InsertEmbeddings(ctx context.Context, embeddings []models.EmbeddingRecord) error
	SearchSimilar(ctx context.Context, queryVector []float64, limit int) ([]models.SimilarityResult, error)

	// Graph operations
	InsertGraphNodes(ctx context.Context, nodes []models.GraphNode) error
	InsertGraphEdges(ctx context.Context, edges []models.GraphEdge) error
	SearchGraph(ctx context.Context, query *models.GraphQuery) (*models.GraphResult, error)
	GetNodesByEntity(ctx context.Context, entityName string) ([]models.GraphNode, error)
	GetNodeNeighbors(ctx context.Context, nodeID string, maxDepth int) (*models.GraphResult, error)
	FindPathBetweenNodes(ctx context.Context, sourceNodeID, targetNodeID string, maxDepth int) (*models.GraphResult, error)
	GetNodesByChunk(ctx context.Context, chunkID string) ([]models.GraphNode, error)
	GetEdgesByRelationType(ctx context.Context, relationType string) ([]models.GraphEdge, error)

	// Health check
	HealthCheck(ctx context.Context) error
}

// LLMService handles LLM API interactions
type LLMService interface {
	ChunkText(ctx context.Context, text string) ([]string, error)
	ExtractEntities(ctx context.Context, text string) ([]models.GraphNode, error)
}

// EmbeddingService handles embedding generation
type EmbeddingService interface {
	GenerateEmbedding(ctx context.Context, text string) ([]float64, error)
	GenerateBatchEmbeddings(ctx context.Context, texts []string) ([][]float64, error)
}

// TemplateService handles template operations
type TemplateService interface {
	CreateTemplate(ctx context.Context, req *models.CreateTemplateRequest) (*models.TemplateWithInstances, error)
	GetTemplate(ctx context.Context, templateContent string) (*models.TemplateWithInstances, error)
	GetAllTemplates(ctx context.Context) ([]models.TemplateWithInstances, error)
	CreateInstance(ctx context.Context, req *models.CreateInstanceRequest) (*models.TemplateInstance, error)
	UpdateSlotValue(ctx context.Context, instanceChunkID, slotName, value string) error
}

// TagService handles tag operations
type TagService interface {
	AddTag(ctx context.Context, chunkID string, tagContent string) error
	RemoveTag(ctx context.Context, chunkID string, tagChunkID string) error
	GetChunkTags(ctx context.Context, chunkID string) ([]models.ChunkRecord, error)
	GetChunksByTag(ctx context.Context, tagContent string) ([]models.ChunkRecord, error)
	
	// Tag inheritance methods
	AddTagWithInheritance(ctx context.Context, chunkID string, tagContent string) error
	RemoveTagWithInheritance(ctx context.Context, chunkID string, tagChunkID string) error
	GetInheritedTags(ctx context.Context, chunkID string) ([]models.ChunkRecord, error)
}

// SearchService handles various search operations
type SearchService interface {
	SemanticSearch(ctx context.Context, query string, limit int) ([]models.SimilarityResult, error)
	SemanticSearchWithFilters(ctx context.Context, req *models.SemanticSearchRequest) (*models.SemanticSearchResponse, error)
	HybridSearch(ctx context.Context, query string, limit int, semanticWeight float64) ([]models.SimilarityResult, error)
	GraphSearch(ctx context.Context, query *models.GraphQuery) (*models.GraphResult, error)
	SearchByTag(ctx context.Context, tagContent string) ([]models.ChunkWithTags, error)
	SearchChunks(ctx context.Context, query string, filters map[string]interface{}) ([]models.ChunkRecord, error)
}