package models

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// StorageType 儲存類型
type StorageType string

const (
	StorageTypeLocal    StorageType = "local"
	StorageTypeSupabase StorageType = "supabase"
	StorageTypeGoogleDrive StorageType = "google_drive"
	StorageTypeGooglePhotos StorageType = "google_photos"
	StorageTypeNAS      StorageType = "nas"
)

// MediaMetadata 圖片元資料
type MediaMetadata struct {
	OriginalFilename string `json:"original_filename"`
	ContentType      string `json:"content_type"`
	Size             int64  `json:"size"`
	Width            int    `json:"width"`
	Height           int    `json:"height"`
	Hash             string `json:"hash"` // SHA256
}

// StorageResult 上傳結果
type StorageResult struct {
	StorageID   string      `json:"storage_id"`
	URL         string      `json:"url"`
	StorageType StorageType `json:"storage_type"`
	UploadedAt  time.Time   `json:"uploaded_at"`
}

// MediaFile 掃描到的媒體檔案
type MediaFile struct {
	Path         string    `json:"path"`
	Filename     string    `json:"filename"`
	Size         int64     `json:"size"`
	ModifiedAt   time.Time `json:"modified_at"`
	ContentType  string    `json:"content_type"`
	Hash         string    `json:"hash"`
}

// ProcessImageRequest 圖片處理請求
type ProcessImageRequest struct {
	File             []byte      `json:"file,omitempty"`
	FilePath         string      `json:"file_path,omitempty"`
	OriginalFilename string      `json:"original_filename"`
	PageID           *string     `json:"page_id,omitempty"`
	Tags             []string    `json:"tags,omitempty"`
	AutoAnalyze      bool        `json:"auto_analyze"`
	AutoEmbed        bool        `json:"auto_embed"`
	StorageType      StorageType `json:"storage_type"`
}

// ProcessImageResult 圖片處理結果
type ProcessImageResult struct {
	ChunkID      string            `json:"chunk_id"`
	StorageID    string            `json:"storage_id"`
	URL          string            `json:"url"`
	Hash         string            `json:"hash"`
	Analysis     *ImageAnalysis    `json:"analysis,omitempty"`
	EmbeddingIDs map[string]string `json:"embedding_ids,omitempty"` // "image" -> embedding_id, "text" -> embedding_id
}

// ImageAnalysis AI 圖片分析結果
type ImageAnalysis struct {
	Description string    `json:"description"`
	Tags        []string  `json:"tags"`
	Model       string    `json:"model"`
	Confidence  float64   `json:"confidence"`
	AnalyzedAt  time.Time `json:"analyzed_at"`
}

// BatchProcessRequest 批次處理請求
type BatchProcessRequest struct {
	FolderPath   string      `json:"folder_path"`
	Files        []string    `json:"files,omitempty"`
	PageID       *string     `json:"page_id,omitempty"`
	Tags         []string    `json:"tags,omitempty"`
	AutoAnalyze  bool        `json:"auto_analyze"`
	AutoEmbed    bool        `json:"auto_embed"`
	StorageType  StorageType `json:"storage_type"`
	Concurrency  int         `json:"concurrency"`
	FilePatterns []string    `json:"file_patterns,omitempty"` // 檔案過濾模式
}

// BatchProcessResult 批次處理結果
type BatchProcessResult struct {
	BatchID        string               `json:"batch_id"`
	TotalFiles     int                  `json:"total_files"`
	ProcessedFiles int                  `json:"processed_files"`
	FailedFiles    int                  `json:"failed_files"`
	Status         BatchProcessStatus   `json:"status"`
	StartedAt      time.Time            `json:"started_at"`
	CompletedAt    *time.Time           `json:"completed_at,omitempty"`
	Results        []ProcessImageResult `json:"results,omitempty"`
	Errors         []BatchError         `json:"errors,omitempty"`
}

// BatchProcessStatusType 批次處理狀態類型
type BatchProcessStatusType string

const (
	BatchStatusPending    BatchProcessStatusType = "pending"
	BatchStatusProcessing BatchProcessStatusType = "processing"
	BatchStatusCompleted  BatchProcessStatusType = "completed"
	BatchStatusFailed     BatchProcessStatusType = "failed"
	BatchStatusPaused     BatchProcessStatusType = "paused"
	BatchStatusCancelled  BatchProcessStatusType = "cancelled"
)

// BatchProcessStatus 批次處理狀態結構
type BatchProcessStatus struct {
	BatchID        string       `json:"batch_id"`
	TotalFiles     int          `json:"total_files"`
	ProcessedFiles int          `json:"processed_files"`
	FailedFiles    int          `json:"failed_files"`
	Status         string       `json:"status"`
	StartedAt      time.Time    `json:"started_at"`
	CompletedAt    *time.Time   `json:"completed_at,omitempty"`
	Errors         []BatchError `json:"errors"`
}

// BatchError 批次處理錯誤
type BatchError struct {
	Filename  string    `json:"filename"`
	Error     string    `json:"error"`
	Timestamp time.Time `json:"timestamp"`
}

// ImageRecommendation 圖片推薦
type ImageRecommendation struct {
	ChunkID        string   `json:"chunk_id"`
	ImageURL       string   `json:"image_url"`
	Description    string   `json:"description"`
	RelevanceScore float64  `json:"relevance_score"`
	Reason         string   `json:"reason"`
	Tags           []string `json:"tags"`
}

// SlideImageRequest Slide Generator 圖片推薦請求
type SlideImageRequest struct {
	TextContent    string  `json:"text_content"`
	Context        string  `json:"context,omitempty"`
	MaxSuggestions int     `json:"max_suggestions"`
	MinRelevance   float64 `json:"min_relevance"`
}

// ImageRecommendationResponse 圖片推薦回應
type ImageRecommendationResponse struct {
	Suggestions []ImageRecommendation `json:"suggestions"`
	TotalCount  int                   `json:"total_count"`
	SearchTime  time.Duration         `json:"search_time"`
}

// AnalysisOptions 分析選項
type AnalysisOptions struct {
	DetailLevel string `json:"detail_level"` // "low", "medium", "high"
	Language    string `json:"language"`     // "zh-TW", "en"
	MaxTokens   int    `json:"max_tokens"`
}

// 多模態系統特定錯誤
var (
	ErrUnsupportedImageFormat = errors.New("unsupported image format")
	ErrImageTooLarge         = errors.New("image file too large")
	ErrStorageNotAvailable   = errors.New("storage service not available")
	ErrVisionAPIFailed       = errors.New("vision API analysis failed")
	ErrEmbeddingFailed       = errors.New("embedding generation failed")
	ErrDuplicateImage        = errors.New("duplicate image detected")
	ErrInvalidImageData      = errors.New("invalid image data")
	ErrHashCalculationFailed = errors.New("hash calculation failed")
)

// MediaProcessingError 媒體處理錯誤
type MediaProcessingError struct {
	Operation string
	Filename  string
	Cause     error
}

func (e *MediaProcessingError) Error() string {
	return fmt.Sprintf("media processing failed [%s] for file %s: %v", 
		e.Operation, e.Filename, e.Cause)
}

func (e *MediaProcessingError) Unwrap() error {
	return e.Cause
}

// NewMediaProcessingError 建立媒體處理錯誤
func NewMediaProcessingError(operation, filename string, cause error) *MediaProcessingError {
	return &MediaProcessingError{
		Operation: operation,
		Filename:  filename,
		Cause:     cause,
	}
}

// IsImageFile 檢查檔案是否為支援的圖片格式
func IsImageFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	supportedFormats := []string{".png", ".jpg", ".jpeg", ".gif", ".webp", ".bmp"}
	
	for _, format := range supportedFormats {
		if ext == format {
			return true
		}
	}
	return false
}

// GetImageContentType 根據檔案副檔名取得 Content-Type
func GetImageContentType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	
	switch ext {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".bmp":
		return "image/bmp"
	default:
		return "application/octet-stream"
	}
}