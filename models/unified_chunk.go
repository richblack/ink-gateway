package models

import (
	"time"
)

// UnifiedChunkRecord represents the unified chunk structure for all content types
type UnifiedChunkRecord struct {
	ChunkID        string                 `json:"chunk_id" db:"chunk_id"`
	Contents       string                 `json:"contents" db:"contents"`
	Parent         *string                `json:"parent" db:"parent"`
	Page           *string                `json:"page" db:"page"`
	IsPage         bool                   `json:"is_page" db:"is_page"`
	IsTag          bool                   `json:"is_tag" db:"is_tag"`
	IsTemplate     bool                   `json:"is_template" db:"is_template"`
	IsSlot         bool                   `json:"is_slot" db:"is_slot"`
	Ref            *string                `json:"ref" db:"ref"`
	Tags           []string               `json:"tags" db:"tags"`
	Metadata       map[string]interface{} `json:"metadata" db:"metadata"`
	Vector         []float64              `json:"vector,omitempty" db:"vector"`
	VectorType     *string                `json:"vector_type,omitempty" db:"vector_type"`
	VectorModel    *string                `json:"vector_model,omitempty" db:"vector_model"`
	VectorMetadata map[string]interface{} `json:"vector_metadata,omitempty" db:"vector_metadata"`
	CreatedTime    time.Time              `json:"created_time" db:"created_time"`
	LastUpdated    time.Time              `json:"last_updated" db:"last_updated"`
}

// ChunkTagRelation represents the many-to-many relationship between chunks and tags
type ChunkTagRelation struct {
	SourceChunkID string    `json:"source_chunk_id" db:"source_chunk_id"`
	TagChunkID    string    `json:"tag_chunk_id" db:"tag_chunk_id"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
}

// ChunkHierarchyRelation represents the hierarchical relationship between chunks
type ChunkHierarchyRelation struct {
	AncestorID   string   `json:"ancestor_id" db:"ancestor_id"`
	DescendantID string   `json:"descendant_id" db:"descendant_id"`
	Depth        int      `json:"depth" db:"depth"`
	PathIDs      []string `json:"path_ids" db:"path_ids"`
}

// SearchCacheEntry represents cached search results
type SearchCacheEntry struct {
	SearchHash   string                 `json:"search_hash" db:"search_hash"`
	QueryParams  map[string]interface{} `json:"query_params" db:"query_params"`
	ChunkIDs     []string               `json:"chunk_ids" db:"chunk_ids"`
	ResultCount  int                    `json:"result_count" db:"result_count"`
	CreatedAt    time.Time              `json:"created_at" db:"created_at"`
	ExpiresAt    time.Time              `json:"expires_at" db:"expires_at"`
	HitCount     int                    `json:"hit_count" db:"hit_count"`
}

// UnifiedChunkWithTags represents a unified chunk with its associated tags
type UnifiedChunkWithTags struct {
	Chunk *UnifiedChunkRecord   `json:"chunk"`
	Tags  []UnifiedChunkRecord  `json:"tags"`
}

// UnifiedChunkHierarchy represents a hierarchical structure of unified chunks
type UnifiedChunkHierarchy struct {
	Chunk    *UnifiedChunkRecord     `json:"chunk"`
	Children []UnifiedChunkHierarchy `json:"children"`
	Depth    int                     `json:"depth"`
	Path     []string                `json:"path"`
}

// TagStatistics represents usage statistics for tags
type TagStatistics struct {
	TagChunkID  string    `json:"tag_chunk_id"`
	TagContent  string    `json:"tag_content"`
	UsageCount  int       `json:"usage_count"`
	LastUsed    time.Time `json:"last_used"`
}

// BatchCreateRequest represents a request to create multiple chunks
type BatchCreateRequest struct {
	Chunks []UnifiedChunkRecord `json:"chunks"`
}

// BatchUpdateRequest represents a request to update multiple chunks
type BatchUpdateRequest struct {
	Chunks []UnifiedChunkRecord `json:"chunks"`
}

// SearchQuery represents a search query with filters
type SearchQuery struct {
	Content     string                 `json:"content,omitempty"`
	Tags        []string               `json:"tags,omitempty"`
	TagLogic    string                 `json:"tag_logic,omitempty"` // "AND" or "OR"
	IsPage      *bool                  `json:"is_page,omitempty"`
	IsTag       *bool                  `json:"is_tag,omitempty"`
	IsTemplate  *bool                  `json:"is_template,omitempty"`
	IsSlot      *bool                  `json:"is_slot,omitempty"`
	Parent      *string                `json:"parent,omitempty"`
	Page        *string                `json:"page,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Limit       int                    `json:"limit,omitempty"`
	Offset      int                    `json:"offset,omitempty"`
}

// SearchResult represents search results
type SearchResult struct {
	Chunks      []UnifiedChunkRecord `json:"chunks"`
	TotalCount  int                  `json:"total_count"`
	HasMore     bool                 `json:"has_more"`
	SearchTime  time.Duration        `json:"search_time"`
	CacheHit    bool                 `json:"cache_hit"`
}

// VectorType 向量類型常數
type VectorType string

const (
	VectorTypeText  VectorType = "text"
	VectorTypeImage VectorType = "image"
)

// VectorModel 向量模型常數
type VectorModel string

const (
	VectorModelTextEmbedding3Small VectorModel = "text-embedding-3-small"
	VectorModelCLIPViTB32         VectorModel = "clip-vit-b-32"
)

// MultimodalSearchRequest 多模態搜尋請求
type MultimodalSearchRequest struct {
	TextQuery     string                 `json:"text_query,omitempty"`
	ImageQuery    string                 `json:"image_query,omitempty"` // URL 或 base64
	VectorType    string                 `json:"vector_type,omitempty"` // "text", "image", "all"
	Weights       *SearchWeights         `json:"weights,omitempty"`
	Filters       map[string]interface{} `json:"filters,omitempty"`
	Limit         int                    `json:"limit,omitempty"`
	MinSimilarity float64                `json:"min_similarity,omitempty"`
}

// SearchWeights 搜尋權重
type SearchWeights struct {
	Text  float64 `json:"text"`
	Image float64 `json:"image"`
}

// MultimodalSearchResponse 多模態搜尋回應
type MultimodalSearchResponse struct {
	Results     []MultimodalSearchResult `json:"results"`
	TotalCount  int                      `json:"total_count"`
	SearchTime  time.Duration            `json:"search_time"`
	Query       string                   `json:"query"`
	MatchTypes  []string                 `json:"match_types"`
}

// MultimodalSearchResult 多模態搜尋結果
type MultimodalSearchResult struct {
	Chunk       *UnifiedChunkRecord `json:"chunk"`
	Similarity  float64             `json:"similarity"`
	MatchType   string              `json:"match_type"` // "text", "image", "hybrid"
	Explanation string              `json:"explanation,omitempty"`
}

// VectorStatistics 向量統計
type VectorStatistics struct {
	VectorType   string    `json:"vector_type" db:"vector_type"`
	VectorModel  string    `json:"vector_model" db:"vector_model"`
	Count        int       `json:"count" db:"count"`
	FirstCreated time.Time `json:"first_created" db:"first_created"`
	LastCreated  time.Time `json:"last_created" db:"last_created"`
}

// ImageDuplicateCheck 圖片去重檢查結果
type ImageDuplicateCheck struct {
	ChunkID     string    `json:"chunk_id" db:"chunk_id"`
	StorageURL  string    `json:"storage_url" db:"storage_url"`
	CreatedTime time.Time `json:"created_time" db:"created_time"`
}

// IsImageChunk 檢查是否為圖片 chunk
func (c *UnifiedChunkRecord) IsImageChunk() bool {
	if c.Metadata == nil {
		return false
	}
	mediaType, ok := c.Metadata["media_type"].(string)
	return ok && mediaType == "image"
}

// GetImageHash 取得圖片檔案雜湊值
func (c *UnifiedChunkRecord) GetImageHash() string {
	if !c.IsImageChunk() || c.Metadata == nil {
		return ""
	}
	
	storage, ok := c.Metadata["storage"].(map[string]interface{})
	if !ok {
		return ""
	}
	
	hash, ok := storage["file_hash"].(string)
	if !ok {
		return ""
	}
	
	return hash
}

// GetImageURL 取得圖片存取 URL
func (c *UnifiedChunkRecord) GetImageURL() string {
	if !c.IsImageChunk() || c.Metadata == nil {
		return ""
	}
	
	storage, ok := c.Metadata["storage"].(map[string]interface{})
	if !ok {
		return ""
	}
	
	url, ok := storage["url"].(string)
	if !ok {
		return ""
	}
	
	return url
}

// HasVector 檢查是否有向量資料
func (c *UnifiedChunkRecord) HasVector() bool {
	return len(c.Vector) > 0 && c.VectorType != nil && c.VectorModel != nil
}

// GetVectorType 取得向量類型
func (c *UnifiedChunkRecord) GetVectorType() VectorType {
	if c.VectorType == nil {
		return ""
	}
	return VectorType(*c.VectorType)
}

// GetVectorModel 取得向量模型
func (c *UnifiedChunkRecord) GetVectorModel() VectorModel {
	if c.VectorModel == nil {
		return ""
	}
	return VectorModel(*c.VectorModel)
}