package handlers

import (
	"semantic-text-processor/models"
	"time"
)

// ModelConverter handles conversion between legacy and unified models
type ModelConverter struct{}

// NewModelConverter creates a new model converter instance
func NewModelConverter() *ModelConverter {
	return &ModelConverter{}
}

// ToUnifiedChunk converts legacy ChunkRecord to UnifiedChunkRecord
func (c *ModelConverter) ToUnifiedChunk(legacy *models.ChunkRecord) *models.UnifiedChunkRecord {
	if legacy == nil {
		return nil
	}

	unified := &models.UnifiedChunkRecord{
		ChunkID:     legacy.ID,
		Contents:    legacy.Content,
		Parent:      legacy.ParentChunkID,
		IsPage:      false, // Legacy chunks are not pages by default
		IsTag:       false, // Will be set based on context
		IsTemplate:  legacy.IsTemplate,
		IsSlot:      legacy.IsSlot,
		Ref:         legacy.TemplateChunkID,
		Tags:        []string{}, // Will be populated from relationships
		Metadata:    legacy.Metadata,
		CreatedTime: legacy.CreatedAt,
		LastUpdated: legacy.UpdatedAt,
	}

	// Set page reference from text_id if available
	if legacy.TextID != "" {
		unified.Page = &legacy.TextID
	}

	return unified
}

// FromUnifiedChunk converts UnifiedChunkRecord to legacy ChunkRecord
func (c *ModelConverter) FromUnifiedChunk(unified *models.UnifiedChunkRecord) *models.ChunkRecord {
	if unified == nil {
		return nil
	}

	legacy := &models.ChunkRecord{
		ID:              unified.ChunkID,
		Content:         unified.Contents,
		IsTemplate:      unified.IsTemplate,
		IsSlot:          unified.IsSlot,
		ParentChunkID:   unified.Parent,
		TemplateChunkID: unified.Ref,
		Metadata:        unified.Metadata,
		CreatedAt:       unified.CreatedTime,
		UpdatedAt:       unified.LastUpdated,
	}

	// Set text_id from page reference if available
	if unified.Page != nil {
		legacy.TextID = *unified.Page
	}

	// Handle slot value if it's a slot
	if unified.IsSlot && len(unified.Contents) > 0 {
		legacy.SlotValue = &unified.Contents
	}

	return legacy
}

// BatchToUnified converts multiple legacy chunks to unified chunks
func (c *ModelConverter) BatchToUnified(legacyChunks []models.ChunkRecord) []models.UnifiedChunkRecord {
	unified := make([]models.UnifiedChunkRecord, len(legacyChunks))
	for i, chunk := range legacyChunks {
		if converted := c.ToUnifiedChunk(&chunk); converted != nil {
			unified[i] = *converted
		}
	}
	return unified
}

// BatchFromUnified converts multiple unified chunks to legacy chunks
func (c *ModelConverter) BatchFromUnified(unifiedChunks []models.UnifiedChunkRecord) []models.ChunkRecord {
	legacy := make([]models.ChunkRecord, 0, len(unifiedChunks))
	for _, chunk := range unifiedChunks {
		if converted := c.FromUnifiedChunk(&chunk); converted != nil {
			legacy = append(legacy, *converted)
		}
	}
	return legacy
}

// ToUnifiedSearchQuery converts legacy filters to unified search query
func (c *ModelConverter) ToUnifiedSearchQuery(query string, filters map[string]interface{}, limit, offset int) *models.SearchQuery {
	searchQuery := &models.SearchQuery{
		Content: query,
		Limit:   limit,
		Offset:  offset,
	}

	// Convert legacy filters to unified format
	if filters != nil {
		if textID, ok := filters["text_id"].(string); ok && textID != "" {
			searchQuery.Page = &textID
		}

		if isTemplate, ok := filters["is_template"].(bool); ok {
			searchQuery.IsTemplate = &isTemplate
		}

		if isSlot, ok := filters["is_slot"].(bool); ok {
			searchQuery.IsSlot = &isSlot
		}

		if parentID, ok := filters["parent_chunk_id"].(string); ok && parentID != "" {
			searchQuery.Parent = &parentID
		}

		// Handle metadata filters
		if metadata, ok := filters["metadata"].(map[string]interface{}); ok {
			searchQuery.Metadata = metadata
		}
	}

	return searchQuery
}

// ToCreateChunkRequest converts legacy CreateChunkRequest to UnifiedChunkRecord
func (c *ModelConverter) ToCreateChunkRequest(req *models.CreateChunkRequest) *models.UnifiedChunkRecord {
	if req == nil {
		return nil
	}

	chunk := &models.UnifiedChunkRecord{
		Contents:    req.Content,
		Parent:      req.ParentChunkID,
		IsPage:      false,
		IsTag:       false,
		IsTemplate:  req.IsTemplate,
		IsSlot:      req.IsSlot,
		Ref:         req.TemplateChunkID,
		Tags:        []string{},
		Metadata:    req.Metadata,
		CreatedTime: time.Now(),
		LastUpdated: time.Now(),
	}

	// Set page reference from text_id if available
	if req.TextID != "" {
		chunk.Page = &req.TextID
	}

	return chunk
}

// ApplyUpdateRequest applies UpdateChunkRequest to UnifiedChunkRecord
func (c *ModelConverter) ApplyUpdateRequest(chunk *models.UnifiedChunkRecord, req *models.UpdateChunkRequest) {
	if chunk == nil || req == nil {
		return
	}

	if req.Content != nil {
		chunk.Contents = *req.Content
	}

	if req.ParentChunkID != nil {
		chunk.Parent = req.ParentChunkID
	}

	if req.Metadata != nil {
		chunk.Metadata = req.Metadata
	}

	// Update last modified time
	chunk.LastUpdated = time.Now()
}

// ToChunkHierarchy converts unified hierarchy to legacy format
func (c *ModelConverter) ToChunkHierarchy(unified *models.UnifiedChunkHierarchy) *models.ChunkHierarchy {
	if unified == nil {
		return nil
	}

	legacy := &models.ChunkHierarchy{
		Chunk: c.FromUnifiedChunk(unified.Chunk),
		Level: unified.Depth,
	}

	// Convert children recursively
	legacy.Children = make([]models.ChunkHierarchy, len(unified.Children))
	for i, child := range unified.Children {
		if converted := c.ToChunkHierarchy(&child); converted != nil {
			legacy.Children[i] = *converted
		}
	}

	return legacy
}

// ToChunkWithTags converts unified chunk with tags to legacy format
func (c *ModelConverter) ToChunkWithTags(unified *models.UnifiedChunkWithTags) *models.ChunkWithTags {
	if unified == nil {
		return nil
	}

	return &models.ChunkWithTags{
		Chunk: c.FromUnifiedChunk(unified.Chunk),
		Tags:  c.BatchFromUnified(unified.Tags),
	}
}

// ToBatchCreateRequest converts legacy BatchCreateRequest to unified format
func (c *ModelConverter) ToBatchCreateRequest(req *models.BulkUpdateRequest) []models.UnifiedChunkRecord {
	if req == nil || len(req.Updates) == 0 {
		return nil
	}

	chunks := make([]models.UnifiedChunkRecord, 0, len(req.Updates))
	for _, update := range req.Updates {
		chunk := &models.UnifiedChunkRecord{
			ChunkID:     update.ChunkID,
			CreatedTime: time.Now(),
			LastUpdated: time.Now(),
		}

		if update.Content != nil {
			chunk.Contents = *update.Content
		}

		if update.ParentChunkID != nil {
			chunk.Parent = update.ParentChunkID
		}

		chunks = append(chunks, *chunk)
	}

	return chunks
}