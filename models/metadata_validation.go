package models

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// ImageMetadataSchema 圖片 metadata 的結構定義
type ImageMetadataSchema struct {
	MediaType      string                 `json:"media_type" validate:"required,eq=image"`
	Storage        StorageMetadata        `json:"storage" validate:"required"`
	ImageProperties ImageProperties       `json:"image_properties" validate:"required"`
	AIAnalysis     *AIAnalysisMetadata    `json:"ai_analysis,omitempty"`
	Embeddings     *EmbeddingMetadata     `json:"embeddings,omitempty"`
	ProcessingInfo *ProcessingMetadata    `json:"processing_info,omitempty"`
}

// StorageMetadata 儲存相關元資料
type StorageMetadata struct {
	Type             string    `json:"type" validate:"required,oneof=local supabase google_drive google_photos nas"`
	StorageID        string    `json:"storage_id" validate:"required"`
	URL              string    `json:"url" validate:"required,url"`
	OriginalFilename string    `json:"original_filename" validate:"required"`
	FileHash         string    `json:"file_hash" validate:"required,len=64"` // SHA256 hash
	UploadedAt       time.Time `json:"uploaded_at" validate:"required"`
}

// ImageProperties 圖片屬性
type ImageProperties struct {
	Format    string `json:"format" validate:"required,oneof=png jpg jpeg gif webp bmp"`
	SizeBytes int64  `json:"size_bytes" validate:"required,min=1"`
	Width     int    `json:"width" validate:"required,min=1"`
	Height    int    `json:"height" validate:"required,min=1"`
	MimeType  string `json:"mime_type" validate:"required"`
}

// AIAnalysisMetadata AI 分析元資料
type AIAnalysisMetadata struct {
	Description string    `json:"description" validate:"required"`
	Model       string    `json:"model" validate:"required"`
	Tags        []string  `json:"tags,omitempty"`
	Confidence  float64   `json:"confidence" validate:"min=0,max=1"`
	AnalyzedAt  time.Time `json:"analyzed_at" validate:"required"`
}

// EmbeddingMetadata 向量元資料
type EmbeddingMetadata struct {
	Image *EmbeddingInfo `json:"image,omitempty"`
	Text  *EmbeddingInfo `json:"text,omitempty"`
}

// EmbeddingInfo 向量資訊
type EmbeddingInfo struct {
	Model       string    `json:"model" validate:"required"`
	Dimensions  int       `json:"dimensions" validate:"required,min=1"`
	EmbeddingID string    `json:"embedding_id" validate:"required"`
	CreatedAt   time.Time `json:"created_at" validate:"required"`
}

// ProcessingMetadata 處理相關元資料
type ProcessingMetadata struct {
	ProcessedAt   time.Time `json:"processed_at" validate:"required"`
	ProcessingID  string    `json:"processing_id,omitempty"`
	BatchID       string    `json:"batch_id,omitempty"`
	SourcePath    string    `json:"source_path,omitempty"`
	ProcessedBy   string    `json:"processed_by,omitempty"` // 處理者標識
}

// ValidateImageMetadata 驗證圖片 metadata 結構
func ValidateImageMetadata(metadata map[string]interface{}) error {
	// 檢查是否為圖片類型
	mediaType, ok := metadata["media_type"].(string)
	if !ok || mediaType != "image" {
		return fmt.Errorf("invalid or missing media_type, expected 'image'")
	}
	
	// 驗證必要的頂層欄位
	requiredFields := []string{"storage", "image_properties"}
	for _, field := range requiredFields {
		if _, exists := metadata[field]; !exists {
			return fmt.Errorf("missing required field: %s", field)
		}
	}
	
	// 驗證 storage 結構
	if err := validateStorageMetadata(metadata["storage"]); err != nil {
		return fmt.Errorf("invalid storage metadata: %w", err)
	}
	
	// 驗證 image_properties 結構
	if err := validateImageProperties(metadata["image_properties"]); err != nil {
		return fmt.Errorf("invalid image_properties metadata: %w", err)
	}
	
	// 驗證可選的 ai_analysis 結構
	if aiAnalysis, exists := metadata["ai_analysis"]; exists {
		if err := validateAIAnalysisMetadata(aiAnalysis); err != nil {
			return fmt.Errorf("invalid ai_analysis metadata: %w", err)
		}
	}
	
	// 驗證可選的 embeddings 結構
	if embeddings, exists := metadata["embeddings"]; exists {
		if err := validateEmbeddingMetadata(embeddings); err != nil {
			return fmt.Errorf("invalid embeddings metadata: %w", err)
		}
	}
	
	return nil
}

// validateStorageMetadata 驗證儲存元資料
func validateStorageMetadata(storage interface{}) error {
	storageMap, ok := storage.(map[string]interface{})
	if !ok {
		return fmt.Errorf("storage must be an object")
	}
	
	// 檢查必要欄位
	requiredFields := map[string]string{
		"type":              "string",
		"storage_id":        "string",
		"url":               "string",
		"original_filename": "string",
		"file_hash":         "string",
		"uploaded_at":       "string",
	}
	
	for field, expectedType := range requiredFields {
		value, exists := storageMap[field]
		if !exists {
			return fmt.Errorf("missing required storage field: %s", field)
		}
		
		if expectedType == "string" {
			if _, ok := value.(string); !ok {
				return fmt.Errorf("storage field %s must be a string", field)
			}
		}
	}
	
	// 驗證儲存類型
	storageType := storageMap["type"].(string)
	validTypes := []string{"local", "supabase", "google_drive", "google_photos", "nas"}
	if !contains(validTypes, storageType) {
		return fmt.Errorf("invalid storage type: %s", storageType)
	}
	
	// 驗證檔案雜湊長度（SHA256 應該是 64 字元）
	fileHash := storageMap["file_hash"].(string)
	if len(fileHash) != 64 {
		return fmt.Errorf("file_hash must be 64 characters (SHA256)")
	}
	
	return nil
}

// validateImageProperties 驗證圖片屬性
func validateImageProperties(properties interface{}) error {
	propsMap, ok := properties.(map[string]interface{})
	if !ok {
		return fmt.Errorf("image_properties must be an object")
	}
	
	// 檢查必要欄位
	requiredFields := []string{"format", "size_bytes", "width", "height", "mime_type"}
	for _, field := range requiredFields {
		if _, exists := propsMap[field]; !exists {
			return fmt.Errorf("missing required image_properties field: %s", field)
		}
	}
	
	// 驗證格式
	format, ok := propsMap["format"].(string)
	if !ok {
		return fmt.Errorf("format must be a string")
	}
	
	validFormats := []string{"png", "jpg", "jpeg", "gif", "webp", "bmp"}
	if !contains(validFormats, strings.ToLower(format)) {
		return fmt.Errorf("invalid image format: %s", format)
	}
	
	// 驗證數值欄位
	if err := validatePositiveNumber(propsMap["size_bytes"], "size_bytes"); err != nil {
		return err
	}
	if err := validatePositiveNumber(propsMap["width"], "width"); err != nil {
		return err
	}
	if err := validatePositiveNumber(propsMap["height"], "height"); err != nil {
		return err
	}
	
	return nil
}

// validateAIAnalysisMetadata 驗證 AI 分析元資料
func validateAIAnalysisMetadata(analysis interface{}) error {
	analysisMap, ok := analysis.(map[string]interface{})
	if !ok {
		return fmt.Errorf("ai_analysis must be an object")
	}
	
	// 檢查必要欄位
	requiredFields := []string{"description", "model", "analyzed_at"}
	for _, field := range requiredFields {
		if _, exists := analysisMap[field]; !exists {
			return fmt.Errorf("missing required ai_analysis field: %s", field)
		}
	}
	
	// 驗證 confidence 欄位（如果存在）
	if confidence, exists := analysisMap["confidence"]; exists {
		if confFloat, ok := confidence.(float64); ok {
			if confFloat < 0 || confFloat > 1 {
				return fmt.Errorf("confidence must be between 0 and 1")
			}
		} else {
			return fmt.Errorf("confidence must be a number")
		}
	}
	
	return nil
}

// validateEmbeddingMetadata 驗證向量元資料
func validateEmbeddingMetadata(embeddings interface{}) error {
	embeddingsMap, ok := embeddings.(map[string]interface{})
	if !ok {
		return fmt.Errorf("embeddings must be an object")
	}
	
	// 驗證 image 向量資訊（如果存在）
	if imageEmb, exists := embeddingsMap["image"]; exists {
		if err := validateEmbeddingInfo(imageEmb, "image"); err != nil {
			return err
		}
	}
	
	// 驗證 text 向量資訊（如果存在）
	if textEmb, exists := embeddingsMap["text"]; exists {
		if err := validateEmbeddingInfo(textEmb, "text"); err != nil {
			return err
		}
	}
	
	return nil
}

// validateEmbeddingInfo 驗證向量資訊
func validateEmbeddingInfo(embInfo interface{}, embType string) error {
	embMap, ok := embInfo.(map[string]interface{})
	if !ok {
		return fmt.Errorf("%s embedding must be an object", embType)
	}
	
	// 檢查必要欄位
	requiredFields := []string{"model", "dimensions", "embedding_id", "created_at"}
	for _, field := range requiredFields {
		if _, exists := embMap[field]; !exists {
			return fmt.Errorf("missing required %s embedding field: %s", embType, field)
		}
	}
	
	// 驗證 dimensions
	if err := validatePositiveNumber(embMap["dimensions"], "dimensions"); err != nil {
		return fmt.Errorf("invalid %s embedding dimensions: %w", embType, err)
	}
	
	return nil
}

// validatePositiveNumber 驗證正數
func validatePositiveNumber(value interface{}, fieldName string) error {
	switch v := value.(type) {
	case int:
		if v <= 0 {
			return fmt.Errorf("%s must be positive", fieldName)
		}
	case int64:
		if v <= 0 {
			return fmt.Errorf("%s must be positive", fieldName)
		}
	case float64:
		if v <= 0 {
			return fmt.Errorf("%s must be positive", fieldName)
		}
	default:
		return fmt.Errorf("%s must be a number", fieldName)
	}
	return nil
}

// contains 檢查切片是否包含指定元素
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// SerializeImageMetadata 序列化圖片 metadata
func SerializeImageMetadata(schema *ImageMetadataSchema) (map[string]interface{}, error) {
	data, err := json.Marshal(schema)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal image metadata: %w", err)
	}
	
	var metadata map[string]interface{}
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal image metadata: %w", err)
	}
	
	return metadata, nil
}

// DeserializeImageMetadata 反序列化圖片 metadata
func DeserializeImageMetadata(metadata map[string]interface{}) (*ImageMetadataSchema, error) {
	// 先驗證結構
	if err := ValidateImageMetadata(metadata); err != nil {
		return nil, fmt.Errorf("metadata validation failed: %w", err)
	}
	
	data, err := json.Marshal(metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata for deserialization: %w", err)
	}
	
	var schema ImageMetadataSchema
	if err := json.Unmarshal(data, &schema); err != nil {
		return nil, fmt.Errorf("failed to unmarshal image metadata schema: %w", err)
	}
	
	return &schema, nil
}