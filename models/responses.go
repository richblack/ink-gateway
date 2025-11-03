package models

import "time"

// Response structures for complex queries

// TemplateWithInstances represents template with its instances
type TemplateWithInstances struct {
	Template  *ChunkRecord       `json:"template"`
	Slots     []ChunkRecord      `json:"slots"`
	Instances []TemplateInstance `json:"instances"`
}

// TemplateInstance represents a template instance with slot values
type TemplateInstance struct {
	Instance   *ChunkRecord            `json:"instance"`
	SlotValues map[string]*ChunkRecord `json:"slot_values"`
}

// ChunkWithTags represents chunk with its associated tags
type ChunkWithTags struct {
	Chunk *ChunkRecord  `json:"chunk"`
	Tags  []ChunkRecord `json:"tags"`
}

// ChunkHierarchy represents hierarchical chunk structure
type ChunkHierarchy struct {
	Chunk    *ChunkRecord     `json:"chunk"`
	Children []ChunkHierarchy `json:"children"`
	Level    int              `json:"level"`
}

// BulletStructure represents complete bullet-point structure
type BulletStructure struct {
	RootChunks []ChunkHierarchy `json:"root_chunks"`
	MaxDepth   int              `json:"max_depth"`
}

// SimilarityResult represents semantic search results
type SimilarityResult struct {
	Chunk      ChunkRecord `json:"chunk"`
	Similarity float64     `json:"similarity"`
}

// GraphQuery represents graph search parameters
type GraphQuery struct {
	EntityName string `json:"entity_name"`
	MaxDepth   int    `json:"max_depth"`
	Limit      int    `json:"limit"`
}

// GraphResult represents graph search results
type GraphResult struct {
	Nodes []GraphNode `json:"nodes"`
	Edges []GraphEdge `json:"edges"`
}

// ProcessResult represents the result of text processing
type ProcessResult struct {
	TextID      string        `json:"text_id"`
	Chunks      []ChunkRecord `json:"chunks"`
	Status      string        `json:"status"`
	ProcessedAt time.Time     `json:"processed_at"`
}

// APIError represents standardized error response
type APIError struct {
	Type    string `json:"type"`
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// SemanticSearchResponse represents paginated search results
type SemanticSearchResponse struct {
	Results    []SimilarityResult `json:"results"`
	TotalCount int                `json:"total_count"`
	Query      string             `json:"query"`
	Limit      int                `json:"limit"`
}

// CreateTextResponse represents response after creating text
type CreateTextResponse struct {
	ID          string    `json:"id"`
	Status      string    `json:"status"`
	ChunkCount  int       `json:"chunk_count"`
	ProcessedAt time.Time `json:"processed_at"`
}