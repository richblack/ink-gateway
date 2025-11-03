package models

// API Request/Response structures

// CreateTemplateRequest for creating new templates
type CreateTemplateRequest struct {
	TemplateName string   `json:"template_name"`
	SlotNames    []string `json:"slot_names"`
}

// CreateInstanceRequest for creating template instances
type CreateInstanceRequest struct {
	TemplateChunkID string            `json:"template_chunk_id"`
	InstanceName    string            `json:"instance_name"`
	SlotValues      map[string]string `json:"slot_values"`
}

// AddTagRequest for adding tags to chunks
type AddTagRequest struct {
	ChunkID    string `json:"chunk_id"`
	TagContent string `json:"tag_content"`
}

// MoveChunkRequest for moving chunks in hierarchy
type MoveChunkRequest struct {
	ChunkID        string  `json:"chunk_id"`
	NewParentID    *string `json:"new_parent_id"`
	NewPosition    int     `json:"new_position"`
	NewIndentLevel int     `json:"new_indent_level"`
}

// BulkUpdateRequest for batch chunk updates
type BulkUpdateRequest struct {
	Updates []ChunkUpdate `json:"updates"`
}

// ChunkUpdate represents individual chunk update
type ChunkUpdate struct {
	ChunkID        string  `json:"chunk_id"`
	Content        *string `json:"content,omitempty"`
	ParentChunkID  *string `json:"parent_chunk_id,omitempty"`
	SequenceNumber *int    `json:"sequence_number,omitempty"`
	IndentLevel    *int    `json:"indent_level,omitempty"`
}

// Pagination for list queries
type Pagination struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
	Total    int `json:"total"`
}

// TextList represents paginated text results
type TextList struct {
	Texts      []TextRecord `json:"texts"`
	Pagination Pagination   `json:"pagination"`
}

// TextDetail represents detailed text information
type TextDetail struct {
	Text   TextRecord    `json:"text"`
	Chunks []ChunkRecord `json:"chunks"`
}

// SemanticSearchRequest represents parameters for semantic search
type SemanticSearchRequest struct {
	Query           string                 `json:"query"`
	Limit           int                    `json:"limit"`
	MinSimilarity   float64                `json:"min_similarity"`
	Filters         map[string]interface{} `json:"filters"`
	IncludeMetadata bool                   `json:"include_metadata"`
}

// CreateTextRequest for creating new texts
type CreateTextRequest struct {
	Content string `json:"content"`
	Title   string `json:"title,omitempty"`
}

// UpdateTextRequest for updating existing texts
type UpdateTextRequest struct {
	Content *string `json:"content,omitempty"`
	Title   *string `json:"title,omitempty"`
}

// CreateChunkRequest for creating new chunks
type CreateChunkRequest struct {
	TextID          string                 `json:"text_id,omitempty"`
	Content         string                 `json:"content"`
	IsTemplate      bool                   `json:"is_template,omitempty"`
	IsSlot          bool                   `json:"is_slot,omitempty"`
	ParentChunkID   *string                `json:"parent_chunk_id,omitempty"`
	TemplateChunkID *string                `json:"template_chunk_id,omitempty"`
	SlotValue       *string                `json:"slot_value,omitempty"`
	IndentLevel     int                    `json:"indent_level,omitempty"`
	SequenceNumber  *int                   `json:"sequence_number,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// UpdateChunkRequest for updating existing chunks
type UpdateChunkRequest struct {
	Content        *string                `json:"content,omitempty"`
	ParentChunkID  *string                `json:"parent_chunk_id,omitempty"`
	IndentLevel    *int                   `json:"indent_level,omitempty"`
	SequenceNumber *int                   `json:"sequence_number,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// TagSearchRequest for searching by tags
type TagSearchRequest struct {
	TagContent string `json:"tag_content"`
}

// ChunkSearchRequest for general chunk search
type ChunkSearchRequest struct {
	Query   string                 `json:"query"`
	Filters map[string]interface{} `json:"filters,omitempty"`
}

// HybridSearchRequest for hybrid search combining semantic and text search
type HybridSearchRequest struct {
	Query           string  `json:"query"`
	Limit           int     `json:"limit"`
	SemanticWeight  float64 `json:"semantic_weight"`
}

// UpdateSlotValueRequest for updating slot values in template instances
type UpdateSlotValueRequest struct {
	SlotName string `json:"slot_name"`
	Value    string `json:"value"`
}