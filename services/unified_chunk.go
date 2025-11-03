package services

import (
	"context"
	"semantic-text-processor/models"
)

// UnifiedChunkService provides unified access to all chunk operations
type UnifiedChunkService interface {
	// Basic CRUD operations
	CreateChunk(ctx context.Context, chunk *models.UnifiedChunkRecord) error
	GetChunk(ctx context.Context, chunkID string) (*models.UnifiedChunkRecord, error)
	UpdateChunk(ctx context.Context, chunk *models.UnifiedChunkRecord) error
	DeleteChunk(ctx context.Context, chunkID string) error

	// Batch operations
	BatchCreateChunks(ctx context.Context, chunks []models.UnifiedChunkRecord) error
	BatchUpdateChunks(ctx context.Context, chunks []models.UnifiedChunkRecord) error

	// Tag operations
	AddTags(ctx context.Context, chunkID string, tagChunkIDs []string) error
	RemoveTags(ctx context.Context, chunkID string, tagChunkIDs []string) error
	GetChunkTags(ctx context.Context, chunkID string) ([]models.UnifiedChunkRecord, error)
	GetChunksByTag(ctx context.Context, tagChunkID string) ([]models.UnifiedChunkRecord, error)
	GetChunksByTags(ctx context.Context, tagChunkIDs []string, matchType string) ([]models.UnifiedChunkRecord, error)

	// Hierarchy operations
	GetChildren(ctx context.Context, parentChunkID string) ([]models.UnifiedChunkRecord, error)
	GetDescendants(ctx context.Context, ancestorChunkID string, maxDepth int) ([]models.UnifiedChunkRecord, error)
	GetAncestors(ctx context.Context, chunkID string) ([]models.UnifiedChunkRecord, error)
	MoveChunk(ctx context.Context, chunkID, newParentID string) error

	// Search operations
	SearchChunks(ctx context.Context, query *models.SearchQuery) (*models.SearchResult, error)
	SearchByContent(ctx context.Context, content string, filters map[string]interface{}) ([]models.UnifiedChunkRecord, error)
}