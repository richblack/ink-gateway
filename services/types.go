package services

import "semantic-text-processor/models"

// Type aliases for cleaner code across the services package
type (
	SimilarityResult         = models.SimilarityResult
	SemanticSearchRequest    = models.SemanticSearchRequest
	SemanticSearchResponse   = models.SemanticSearchResponse
	GraphQuery               = models.GraphQuery
	GraphResult              = models.GraphResult
	ChunkWithTags            = models.ChunkWithTags
	ChunkRecord              = models.ChunkRecord
	TextDetail               = models.TextDetail
	TextRecord               = models.TextRecord
	Pagination               = models.Pagination
	TextList                 = models.TextList
	TemplateWithInstances    = models.TemplateWithInstances
	TemplateInstance         = models.TemplateInstance
	CreateInstanceRequest    = models.CreateInstanceRequest
	ChunkHierarchy           = models.ChunkHierarchy
	MoveChunkRequest         = models.MoveChunkRequest
	BulkUpdateRequest        = models.BulkUpdateRequest
	EmbeddingRecord          = models.EmbeddingRecord
	GraphNode                = models.GraphNode
	GraphEdge                = models.GraphEdge
	ProcessResult            = models.ProcessResult
)