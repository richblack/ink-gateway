package models

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// MetadataHelper 提供 metadata 相關的輔助函數
type MetadataHelper struct{}

// NewMetadataHelper 建立新的 metadata 輔助工具
func NewMetadataHelper() *MetadataHelper {
	return &MetadataHelper{}
}

// IsImageChunk 檢查 chunk 是否為圖片類型
func (h *MetadataHelper) IsImageChunk(chunk *UnifiedChunkRecord) bool {
	return chunk.IsImageChunk()
}

// GetImageURL 取得圖片 URL
func (h *MetadataHelper) GetImageURL(chunk *UnifiedChunkRecord) string {
	return chunk.GetImageURL()
}

// GetImageHash 取得圖片雜湊值
func (h *MetadataHelper) GetImageHash(chunk *UnifiedChunkRecord) string {
	return chunk.GetImageHash()
}

// GetStorageType 取得儲存類型
func (h *MetadataHelper) GetStorageType(chunk *UnifiedChunkRecord) StorageType {
	if !chunk.IsImageChunk() || chunk.Metadata == nil {
		return ""
	}
	
	storage, ok := chunk.Metadata["storage"].(map[string]interface{})
	if !ok {
		return ""
	}
	
	storageType, ok := storage["type"].(string)
	if !ok {
		return ""
	}
	
	return StorageType(storageType)
}

// GetStorageID 取得儲存 ID
func (h *MetadataHelper) GetStorageID(chunk *UnifiedChunkRecord) string {
	if !chunk.IsImageChunk() || chunk.Metadata == nil {
		return ""
	}
	
	storage, ok := chunk.Metadata["storage"].(map[string]interface{})
	if !ok {
		return ""
	}
	
	storageID, ok := storage["storage_id"].(string)
	if !ok {
		return ""
	}
	
	return storageID
}

// GetOriginalFilename 取得原始檔名
func (h *MetadataHelper) GetOriginalFilename(chunk *UnifiedChunkRecord) string {
	if !chunk.IsImageChunk() || chunk.Metadata == nil {
		return ""
	}
	
	storage, ok := chunk.Metadata["storage"].(map[string]interface{})
	if !ok {
		return ""
	}
	
	filename, ok := storage["original_filename"].(string)
	if !ok {
		return ""
	}
	
	return filename
}

// GetImageFormat 取得圖片格式
func (h *MetadataHelper) GetImageFormat(chunk *UnifiedChunkRecord) string {
	if !chunk.IsImageChunk() || chunk.Metadata == nil {
		return ""
	}
	
	properties, ok := chunk.Metadata["image_properties"].(map[string]interface{})
	if !ok {
		return ""
	}
	
	format, ok := properties["format"].(string)
	if !ok {
		return ""
	}
	
	return format
}

// GetImageSize 取得圖片尺寸
func (h *MetadataHelper) GetImageSize(chunk *UnifiedChunkRecord) (width, height int) {
	if !chunk.IsImageChunk() || chunk.Metadata == nil {
		return 0, 0
	}
	
	properties, ok := chunk.Metadata["image_properties"].(map[string]interface{})
	if !ok {
		return 0, 0
	}
	
	width, _ = properties["width"].(int)
	height, _ = properties["height"].(int)
	
	return width, height
}

// GetImageSizeBytes 取得圖片檔案大小
func (h *MetadataHelper) GetImageSizeBytes(chunk *UnifiedChunkRecord) int64 {
	if !chunk.IsImageChunk() || chunk.Metadata == nil {
		return 0
	}
	
	properties, ok := chunk.Metadata["image_properties"].(map[string]interface{})
	if !ok {
		return 0
	}
	
	// 處理不同的數值類型
	switch size := properties["size_bytes"].(type) {
	case int64:
		return size
	case int:
		return int64(size)
	case float64:
		return int64(size)
	default:
		return 0
	}
}

// GetAIDescription 取得 AI 生成的描述
func (h *MetadataHelper) GetAIDescription(chunk *UnifiedChunkRecord) string {
	if !chunk.IsImageChunk() || chunk.Metadata == nil {
		return ""
	}
	
	analysis, ok := chunk.Metadata["ai_analysis"].(map[string]interface{})
	if !ok {
		return ""
	}
	
	description, ok := analysis["description"].(string)
	if !ok {
		return ""
	}
	
	return description
}

// GetAITags 取得 AI 生成的標籤
func (h *MetadataHelper) GetAITags(chunk *UnifiedChunkRecord) []string {
	if !chunk.IsImageChunk() || chunk.Metadata == nil {
		return nil
	}
	
	analysis, ok := chunk.Metadata["ai_analysis"].(map[string]interface{})
	if !ok {
		return nil
	}
	
	tagsInterface, ok := analysis["tags"].([]interface{})
	if !ok {
		return nil
	}
	
	tags := make([]string, 0, len(tagsInterface))
	for _, tag := range tagsInterface {
		if tagStr, ok := tag.(string); ok {
			tags = append(tags, tagStr)
		}
	}
	
	return tags
}

// GetAIModel 取得 AI 分析使用的模型
func (h *MetadataHelper) GetAIModel(chunk *UnifiedChunkRecord) string {
	if !chunk.IsImageChunk() || chunk.Metadata == nil {
		return ""
	}
	
	analysis, ok := chunk.Metadata["ai_analysis"].(map[string]interface{})
	if !ok {
		return ""
	}
	
	model, ok := analysis["model"].(string)
	if !ok {
		return ""
	}
	
	return model
}

// GetAIConfidence 取得 AI 分析的信心度
func (h *MetadataHelper) GetAIConfidence(chunk *UnifiedChunkRecord) float64 {
	if !chunk.IsImageChunk() || chunk.Metadata == nil {
		return 0
	}
	
	analysis, ok := chunk.Metadata["ai_analysis"].(map[string]interface{})
	if !ok {
		return 0
	}
	
	confidence, ok := analysis["confidence"].(float64)
	if !ok {
		return 0
	}
	
	return confidence
}

// GetUploadedAt 取得上傳時間
func (h *MetadataHelper) GetUploadedAt(chunk *UnifiedChunkRecord) time.Time {
	if !chunk.IsImageChunk() || chunk.Metadata == nil {
		return time.Time{}
	}
	
	storage, ok := chunk.Metadata["storage"].(map[string]interface{})
	if !ok {
		return time.Time{}
	}
	
	uploadedAtStr, ok := storage["uploaded_at"].(string)
	if !ok {
		return time.Time{}
	}
	
	uploadedAt, err := time.Parse(time.RFC3339, uploadedAtStr)
	if err != nil {
		return time.Time{}
	}
	
	return uploadedAt
}

// GetAnalyzedAt 取得分析時間
func (h *MetadataHelper) GetAnalyzedAt(chunk *UnifiedChunkRecord) time.Time {
	if !chunk.IsImageChunk() || chunk.Metadata == nil {
		return time.Time{}
	}
	
	analysis, ok := chunk.Metadata["ai_analysis"].(map[string]interface{})
	if !ok {
		return time.Time{}
	}
	
	analyzedAtStr, ok := analysis["analyzed_at"].(string)
	if !ok {
		return time.Time{}
	}
	
	analyzedAt, err := time.Parse(time.RFC3339, analyzedAtStr)
	if err != nil {
		return time.Time{}
	}
	
	return analyzedAt
}

// GetEmbeddingIDs 取得向量 ID
func (h *MetadataHelper) GetEmbeddingIDs(chunk *UnifiedChunkRecord) map[string]string {
	if !chunk.IsImageChunk() || chunk.Metadata == nil {
		return nil
	}
	
	embeddings, ok := chunk.Metadata["embeddings"].(map[string]interface{})
	if !ok {
		return nil
	}
	
	result := make(map[string]string)
	
	// 取得圖片向量 ID
	if imageEmb, ok := embeddings["image"].(map[string]interface{}); ok {
		if embID, ok := imageEmb["embedding_id"].(string); ok {
			result["image"] = embID
		}
	}
	
	// 取得文字向量 ID
	if textEmb, ok := embeddings["text"].(map[string]interface{}); ok {
		if embID, ok := textEmb["embedding_id"].(string); ok {
			result["text"] = embID
		}
	}
	
	return result
}

// UpdateImageMetadata 更新圖片 metadata
func (h *MetadataHelper) UpdateImageMetadata(chunk *UnifiedChunkRecord, updates map[string]interface{}) error {
	if !chunk.IsImageChunk() {
		return fmt.Errorf("chunk is not an image chunk")
	}
	
	if chunk.Metadata == nil {
		chunk.Metadata = make(map[string]interface{})
	}
	
	// 深度合併更新
	for key, value := range updates {
		if existingValue, exists := chunk.Metadata[key]; exists {
			// 如果是 map，進行深度合併
			if existingMap, ok := existingValue.(map[string]interface{}); ok {
				if updateMap, ok := value.(map[string]interface{}); ok {
					for subKey, subValue := range updateMap {
						existingMap[subKey] = subValue
					}
					continue
				}
			}
		}
		
		// 直接覆蓋或新增
		chunk.Metadata[key] = value
	}
	
	// 更新最後修改時間
	chunk.LastUpdated = time.Now()
	
	// 驗證更新後的 metadata
	return ValidateImageMetadata(chunk.Metadata)
}

// AddAIAnalysis 新增 AI 分析結果
func (h *MetadataHelper) AddAIAnalysis(chunk *UnifiedChunkRecord, analysis *ImageAnalysis) error {
	if !chunk.IsImageChunk() {
		return fmt.Errorf("chunk is not an image chunk")
	}
	
	aiAnalysis := map[string]interface{}{
		"description": analysis.Description,
		"model":       analysis.Model,
		"tags":        analysis.Tags,
		"confidence":  analysis.Confidence,
		"analyzed_at": analysis.AnalyzedAt.Format(time.RFC3339),
	}
	
	return h.UpdateImageMetadata(chunk, map[string]interface{}{
		"ai_analysis": aiAnalysis,
	})
}

// AddEmbeddingInfo 新增向量資訊
func (h *MetadataHelper) AddEmbeddingInfo(chunk *UnifiedChunkRecord, embeddingType, model, embeddingID string, dimensions int) error {
	if !chunk.IsImageChunk() {
		return fmt.Errorf("chunk is not an image chunk")
	}
	
	embeddingInfo := map[string]interface{}{
		"model":        model,
		"dimensions":   dimensions,
		"embedding_id": embeddingID,
		"created_at":   time.Now().Format(time.RFC3339),
	}
	
	embeddings := make(map[string]interface{})
	if chunk.Metadata != nil {
		if existingEmbeddings, ok := chunk.Metadata["embeddings"].(map[string]interface{}); ok {
			embeddings = existingEmbeddings
		}
	}
	
	embeddings[embeddingType] = embeddingInfo
	
	return h.UpdateImageMetadata(chunk, map[string]interface{}{
		"embeddings": embeddings,
	})
}

// GetMetadataSummary 取得 metadata 摘要
func (h *MetadataHelper) GetMetadataSummary(chunk *UnifiedChunkRecord) map[string]interface{} {
	if !chunk.IsImageChunk() {
		return map[string]interface{}{
			"is_image": false,
		}
	}
	
	width, height := h.GetImageSize(chunk)
	
	summary := map[string]interface{}{
		"is_image":          true,
		"storage_type":      string(h.GetStorageType(chunk)),
		"original_filename": h.GetOriginalFilename(chunk),
		"format":            h.GetImageFormat(chunk),
		"size_bytes":        h.GetImageSizeBytes(chunk),
		"width":             width,
		"height":            height,
		"has_ai_analysis":   h.GetAIDescription(chunk) != "",
		"ai_model":          h.GetAIModel(chunk),
		"ai_confidence":     h.GetAIConfidence(chunk),
		"uploaded_at":       h.GetUploadedAt(chunk),
		"analyzed_at":       h.GetAnalyzedAt(chunk),
	}
	
	// 新增向量資訊
	embeddingIDs := h.GetEmbeddingIDs(chunk)
	summary["has_image_embedding"] = embeddingIDs["image"] != ""
	summary["has_text_embedding"] = embeddingIDs["text"] != ""
	summary["embedding_ids"] = embeddingIDs
	
	return summary
}

// ExtractStorageInfo 從 metadata 中提取儲存資訊
func ExtractStorageInfo(metadata map[string]interface{}) (*StorageInfo, error) {
	if metadata == nil {
		return nil, fmt.Errorf("metadata is nil")
	}
	
	storage, ok := metadata["storage"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("storage info not found in metadata")
	}
	
	storageInfo := &StorageInfo{}
	
	if storageID, ok := storage["storage_id"].(string); ok {
		storageInfo.StorageID = storageID
	}
	
	if url, ok := storage["url"].(string); ok {
		storageInfo.URL = url
	}
	
	if fileHash, ok := storage["file_hash"].(string); ok {
		storageInfo.FileHash = fileHash
	}
	
	if storageType, ok := storage["type"].(string); ok {
		storageInfo.StorageType = StorageType(storageType)
	}
	
	if filename, ok := storage["original_filename"].(string); ok {
		storageInfo.OriginalFilename = filename
	}
	
	return storageInfo, nil
}

// ExtractAIAnalysis 從 metadata 中提取 AI 分析結果
func ExtractAIAnalysis(metadata map[string]interface{}) (*ImageAnalysis, error) {
	if metadata == nil {
		return nil, fmt.Errorf("metadata is nil")
	}
	
	analysis, ok := metadata["ai_analysis"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("ai_analysis not found in metadata")
	}
	
	result := &ImageAnalysis{}
	
	if description, ok := analysis["description"].(string); ok {
		result.Description = description
	}
	
	if model, ok := analysis["model"].(string); ok {
		result.Model = model
	}
	
	if confidence, ok := analysis["confidence"].(float64); ok {
		result.Confidence = confidence
	}
	
	if tagsInterface, ok := analysis["tags"].([]interface{}); ok {
		tags := make([]string, 0, len(tagsInterface))
		for _, tag := range tagsInterface {
			if tagStr, ok := tag.(string); ok {
				tags = append(tags, tagStr)
			}
		}
		result.Tags = tags
	}
	
	if analyzedAtStr, ok := analysis["analyzed_at"].(string); ok {
		if analyzedAt, err := time.Parse(time.RFC3339, analyzedAtStr); err == nil {
			result.AnalyzedAt = analyzedAt
		}
	}
	
	return result, nil
}

// CreateImageMetadata 建立圖片 metadata
func CreateImageMetadata(storageResult *StorageResult, metadata *MediaMetadata) map[string]interface{} {
	return map[string]interface{}{
		"media_type": "image",
		"storage": map[string]interface{}{
			"type":              string(storageResult.StorageType),
			"storage_id":        storageResult.StorageID,
			"url":               storageResult.URL,
			"original_filename": metadata.OriginalFilename,
			"file_hash":         metadata.Hash,
			"uploaded_at":       storageResult.UploadedAt.Format(time.RFC3339),
		},
		"image_properties": map[string]interface{}{
			"format":     getImageFormat(metadata.OriginalFilename),
			"size_bytes": metadata.Size,
			"width":      metadata.Width,
			"height":     metadata.Height,
			"mime_type":  metadata.ContentType,
		},
	}
}

// UpdateAIAnalysis 更新 metadata 中的 AI 分析結果
func UpdateAIAnalysis(metadata map[string]interface{}, analysis *ImageAnalysis) map[string]interface{} {
	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	
	// 複製現有 metadata
	updatedMetadata := make(map[string]interface{})
	for k, v := range metadata {
		updatedMetadata[k] = v
	}
	
	// 新增或更新 AI 分析結果
	updatedMetadata["ai_analysis"] = map[string]interface{}{
		"description": analysis.Description,
		"model":       analysis.Model,
		"tags":        analysis.Tags,
		"confidence":  analysis.Confidence,
		"analyzed_at": analysis.AnalyzedAt.Format(time.RFC3339),
	}
	
	return updatedMetadata
}

// StorageInfo 儲存資訊結構
type StorageInfo struct {
	StorageID        string
	URL              string
	FileHash         string
	StorageType      StorageType
	OriginalFilename string
}

// getImageFormat 從檔名取得圖片格式
func getImageFormat(filename string) string {
	ext := filepath.Ext(filename)
	if len(ext) > 1 {
		return strings.ToLower(ext[1:]) // 移除 "." 並轉小寫
	}
	return ""
}