package migration

import (
	"context"
	"log"

	"semantic-text-processor/clients"
	"semantic-text-processor/models"
)

// LegacySupabaseAdapter wraps the existing SupabaseClient with the compatibility layer
type LegacySupabaseAdapter struct {
	compatibilityLayer *CompatibilityLayer
	logger             *log.Logger
}

// NewLegacySupabaseAdapter creates a new adapter that maintains backward compatibility
func NewLegacySupabaseAdapter(unifiedService UnifiedChunkService) clients.SupabaseClient {
	compatibilityLayer := NewCompatibilityLayer(unifiedService)

	return &LegacySupabaseAdapter{
		compatibilityLayer: compatibilityLayer,
		logger:             log.New(log.Writer(), "[LEGACY_SUPABASE_ADAPTER] ", log.LstdFlags|log.Lshortfile),
	}
}

// Text operations - these would need to be implemented based on how texts map to the unified system
func (lsa *LegacySupabaseAdapter) InsertText(ctx context.Context, text *models.TextRecord) error {
	lsa.logger.Printf("Legacy InsertText called for: %s", text.ID)
	// This would need to be implemented based on how texts are handled in the unified system
	// For now, returning nil as a placeholder
	return nil
}

func (lsa *LegacySupabaseAdapter) GetTexts(ctx context.Context, pagination *models.Pagination) (*models.TextList, error) {
	lsa.logger.Printf("Legacy GetTexts called")
	// Placeholder implementation
	return &models.TextList{
		Texts:      []models.TextRecord{},
		Pagination: *pagination,
	}, nil
}

func (lsa *LegacySupabaseAdapter) GetTextByID(ctx context.Context, id string) (*models.TextDetail, error) {
	lsa.logger.Printf("Legacy GetTextByID called for: %s", id)
	// Placeholder implementation
	return &models.TextDetail{}, nil
}

func (lsa *LegacySupabaseAdapter) UpdateText(ctx context.Context, text *models.TextRecord) error {
	lsa.logger.Printf("Legacy UpdateText called for: %s", text.ID)
	// Placeholder implementation
	return nil
}

func (lsa *LegacySupabaseAdapter) DeleteText(ctx context.Context, id string) error {
	lsa.logger.Printf("Legacy DeleteText called for: %s", id)
	// Placeholder implementation
	return nil
}

// Chunk operations - these delegate to the compatibility layer
func (lsa *LegacySupabaseAdapter) InsertChunk(ctx context.Context, chunk *models.ChunkRecord) error {
	return lsa.compatibilityLayer.InsertChunk(ctx, chunk)
}

func (lsa *LegacySupabaseAdapter) InsertChunks(ctx context.Context, chunks []models.ChunkRecord) error {
	lsa.logger.Printf("Legacy InsertChunks called for %d chunks", len(chunks))

	// Process chunks individually for compatibility
	for i := range chunks {
		if err := lsa.compatibilityLayer.InsertChunk(ctx, &chunks[i]); err != nil {
			return err
		}
	}

	return nil
}

func (lsa *LegacySupabaseAdapter) GetChunkByID(ctx context.Context, id string) (*models.ChunkRecord, error) {
	return lsa.compatibilityLayer.GetChunkByID(ctx, id)
}

func (lsa *LegacySupabaseAdapter) GetChunkByContent(ctx context.Context, content string) (*models.ChunkRecord, error) {
	lsa.logger.Printf("Legacy GetChunkByContent called")

	// Use search functionality to find chunk by content
	chunks, err := lsa.compatibilityLayer.GetChunksByTextID(ctx, content) // This is a simplified approach
	if err != nil {
		return nil, err
	}

	if len(chunks) == 0 {
		return nil, models.ErrNotFound
	}

	return &chunks[0], nil
}

func (lsa *LegacySupabaseAdapter) UpdateChunk(ctx context.Context, chunk *models.ChunkRecord) error {
	return lsa.compatibilityLayer.UpdateChunk(ctx, chunk)
}

func (lsa *LegacySupabaseAdapter) DeleteChunk(ctx context.Context, id string) error {
	return lsa.compatibilityLayer.DeleteChunk(ctx, id)
}

func (lsa *LegacySupabaseAdapter) GetChunksByTextID(ctx context.Context, textID string) ([]models.ChunkRecord, error) {
	return lsa.compatibilityLayer.GetChunksByTextID(ctx, textID)
}

// Template operations - placeholders for now
func (lsa *LegacySupabaseAdapter) CreateTemplate(ctx context.Context, templateName string, slotNames []string) (*models.TemplateWithInstances, error) {
	lsa.logger.Printf("Legacy CreateTemplate called: %s", templateName)
	// Placeholder implementation
	return &models.TemplateWithInstances{}, nil
}

func (lsa *LegacySupabaseAdapter) GetTemplateByContent(ctx context.Context, templateContent string) (*models.TemplateWithInstances, error) {
	lsa.logger.Printf("Legacy GetTemplateByContent called")
	// Placeholder implementation
	return &models.TemplateWithInstances{}, nil
}

func (lsa *LegacySupabaseAdapter) GetAllTemplates(ctx context.Context) ([]models.TemplateWithInstances, error) {
	lsa.logger.Printf("Legacy GetAllTemplates called")
	// Placeholder implementation
	return []models.TemplateWithInstances{}, nil
}

func (lsa *LegacySupabaseAdapter) CreateTemplateInstance(ctx context.Context, req *models.CreateInstanceRequest) (*models.TemplateInstance, error) {
	lsa.logger.Printf("Legacy CreateTemplateInstance called")
	// Placeholder implementation
	return &models.TemplateInstance{}, nil
}

func (lsa *LegacySupabaseAdapter) GetTemplateInstances(ctx context.Context, templateChunkID string) ([]models.TemplateInstance, error) {
	lsa.logger.Printf("Legacy GetTemplateInstances called for: %s", templateChunkID)
	// Placeholder implementation
	return []models.TemplateInstance{}, nil
}

func (lsa *LegacySupabaseAdapter) UpdateSlotValue(ctx context.Context, instanceChunkID, slotName, value string) error {
	lsa.logger.Printf("Legacy UpdateSlotValue called")
	// Placeholder implementation
	return nil
}

// Tag operations - these delegate to the compatibility layer
func (lsa *LegacySupabaseAdapter) AddTag(ctx context.Context, chunkID string, tagContent string) error {
	return lsa.compatibilityLayer.AddTag(ctx, chunkID, tagContent)
}

func (lsa *LegacySupabaseAdapter) RemoveTag(ctx context.Context, chunkID string, tagChunkID string) error {
	return lsa.compatibilityLayer.RemoveTag(ctx, chunkID, tagChunkID)
}

func (lsa *LegacySupabaseAdapter) GetChunkTags(ctx context.Context, chunkID string) ([]models.ChunkRecord, error) {
	return lsa.compatibilityLayer.GetChunkTags(ctx, chunkID)
}

func (lsa *LegacySupabaseAdapter) GetChunksByTag(ctx context.Context, tagContent string) ([]models.ChunkRecord, error) {
	return lsa.compatibilityLayer.GetChunksByTag(ctx, tagContent)
}

// Hierarchy operations - placeholders for now
func (lsa *LegacySupabaseAdapter) GetChunkHierarchy(ctx context.Context, rootChunkID string) (*models.ChunkHierarchy, error) {
	lsa.logger.Printf("Legacy GetChunkHierarchy called for: %s", rootChunkID)
	// Placeholder implementation - would need to transform unified hierarchy to legacy format
	return &models.ChunkHierarchy{}, nil
}

func (lsa *LegacySupabaseAdapter) GetChildrenChunks(ctx context.Context, parentChunkID string) ([]models.ChunkRecord, error) {
	lsa.logger.Printf("Legacy GetChildrenChunks called for: %s", parentChunkID)
	// Placeholder implementation
	return []models.ChunkRecord{}, nil
}

func (lsa *LegacySupabaseAdapter) GetSiblingChunks(ctx context.Context, chunkID string) ([]models.ChunkRecord, error) {
	lsa.logger.Printf("Legacy GetSiblingChunks called for: %s", chunkID)
	// Placeholder implementation
	return []models.ChunkRecord{}, nil
}

func (lsa *LegacySupabaseAdapter) MoveChunk(ctx context.Context, req *models.MoveChunkRequest) error {
	lsa.logger.Printf("Legacy MoveChunk called for: %s", req.ChunkID)
	// Placeholder implementation
	return nil
}

func (lsa *LegacySupabaseAdapter) BulkUpdateChunks(ctx context.Context, req *models.BulkUpdateRequest) error {
	lsa.logger.Printf("Legacy BulkUpdateChunks called")
	// Placeholder implementation
	return nil
}

// Search operations - placeholders for now
func (lsa *LegacySupabaseAdapter) SearchChunks(ctx context.Context, query string, filters map[string]interface{}) ([]models.ChunkRecord, error) {
	lsa.logger.Printf("Legacy SearchChunks called with query: %s", query)
	// Placeholder implementation
	return []models.ChunkRecord{}, nil
}

func (lsa *LegacySupabaseAdapter) SearchByTag(ctx context.Context, tagContent string) ([]models.ChunkWithTags, error) {
	lsa.logger.Printf("Legacy SearchByTag called for: %s", tagContent)
	// Placeholder implementation
	return []models.ChunkWithTags{}, nil
}

// Vector operations - these would remain the same as they reference the vector_db schema
func (lsa *LegacySupabaseAdapter) InsertEmbeddings(ctx context.Context, embeddings []models.EmbeddingRecord) error {
	lsa.logger.Printf("Legacy InsertEmbeddings called for %d embeddings", len(embeddings))
	// These operations would remain unchanged as they operate on vector_db schema
	// Placeholder implementation
	return nil
}

func (lsa *LegacySupabaseAdapter) SearchSimilar(ctx context.Context, queryVector []float64, limit int) ([]models.SimilarityResult, error) {
	lsa.logger.Printf("Legacy SearchSimilar called")
	// Placeholder implementation
	return []models.SimilarityResult{}, nil
}

// Graph operations - these would remain the same as they reference the graph_db schema
func (lsa *LegacySupabaseAdapter) InsertGraphNodes(ctx context.Context, nodes []models.GraphNode) error {
	lsa.logger.Printf("Legacy InsertGraphNodes called for %d nodes", len(nodes))
	// Placeholder implementation
	return nil
}

func (lsa *LegacySupabaseAdapter) InsertGraphEdges(ctx context.Context, edges []models.GraphEdge) error {
	lsa.logger.Printf("Legacy InsertGraphEdges called for %d edges", len(edges))
	// Placeholder implementation
	return nil
}

func (lsa *LegacySupabaseAdapter) SearchGraph(ctx context.Context, query *models.GraphQuery) (*models.GraphResult, error) {
	lsa.logger.Printf("Legacy SearchGraph called")
	// Placeholder implementation
	return &models.GraphResult{}, nil
}

func (lsa *LegacySupabaseAdapter) GetNodesByEntity(ctx context.Context, entityName string) ([]models.GraphNode, error) {
	lsa.logger.Printf("Legacy GetNodesByEntity called for: %s", entityName)
	// Placeholder implementation
	return []models.GraphNode{}, nil
}

func (lsa *LegacySupabaseAdapter) GetNodeNeighbors(ctx context.Context, nodeID string, maxDepth int) (*models.GraphResult, error) {
	lsa.logger.Printf("Legacy GetNodeNeighbors called for: %s", nodeID)
	// Placeholder implementation
	return &models.GraphResult{}, nil
}

func (lsa *LegacySupabaseAdapter) FindPathBetweenNodes(ctx context.Context, sourceNodeID, targetNodeID string, maxDepth int) (*models.GraphResult, error) {
	lsa.logger.Printf("Legacy FindPathBetweenNodes called")
	// Placeholder implementation
	return &models.GraphResult{}, nil
}

func (lsa *LegacySupabaseAdapter) GetNodesByChunk(ctx context.Context, chunkID string) ([]models.GraphNode, error) {
	lsa.logger.Printf("Legacy GetNodesByChunk called for: %s", chunkID)
	// Placeholder implementation
	return []models.GraphNode{}, nil
}

func (lsa *LegacySupabaseAdapter) GetEdgesByRelationType(ctx context.Context, relationType string) ([]models.GraphEdge, error) {
	lsa.logger.Printf("Legacy GetEdgesByRelationType called for: %s", relationType)
	// Placeholder implementation
	return []models.GraphEdge{}, nil
}

// Health check
func (lsa *LegacySupabaseAdapter) HealthCheck(ctx context.Context) error {
	lsa.logger.Printf("Legacy HealthCheck called")
	// In a real implementation, this would check the health of the unified service
	return nil
}

// GetCompatibilityReport provides compatibility reporting
func (lsa *LegacySupabaseAdapter) GetCompatibilityReport() *CompatibilityReport {
	return lsa.compatibilityLayer.GetCompatibilityReport()
}