package services

import (
	"errors"
	"fmt"
)

// 媒體處理相關錯誤
var (
	ErrUnsupportedImageFormat = errors.New("unsupported image format")
	ErrImageTooLarge         = errors.New("image file too large")
	ErrStorageNotAvailable   = errors.New("storage service not available")
	ErrVisionAPIFailed       = errors.New("vision API analysis failed")
	ErrEmbeddingFailed       = errors.New("embedding generation failed")
	ErrDuplicateImage        = errors.New("duplicate image detected")
	ErrInvalidImageData      = errors.New("invalid image data")
	ErrHashCalculationFailed = errors.New("hash calculation failed")
	ErrBatchProcessingFailed = errors.New("batch processing failed")
	ErrStorageAdapterNotFound = errors.New("storage adapter not found")
	ErrInvalidStorageConfig  = errors.New("invalid storage configuration")
)

// MediaProcessingError 媒體處理錯誤
type MediaProcessingError struct {
	Operation string
	Filename  string
	Cause     error
}

func (e *MediaProcessingError) Error() string {
	if e.Filename != "" {
		return fmt.Sprintf("media processing failed [%s] for file %s: %v", 
			e.Operation, e.Filename, e.Cause)
	}
	return fmt.Sprintf("media processing failed [%s]: %v", e.Operation, e.Cause)
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

// StorageError 儲存相關錯誤
type StorageError struct {
	StorageType string
	Operation   string
	StorageID   string
	Cause       error
}

func (e *StorageError) Error() string {
	if e.StorageID != "" {
		return fmt.Sprintf("storage error [%s:%s] for %s: %v", 
			e.StorageType, e.Operation, e.StorageID, e.Cause)
	}
	return fmt.Sprintf("storage error [%s:%s]: %v", 
		e.StorageType, e.Operation, e.Cause)
}

func (e *StorageError) Unwrap() error {
	return e.Cause
}

// NewStorageError 建立儲存錯誤
func NewStorageError(storageType, operation, storageID string, cause error) *StorageError {
	return &StorageError{
		StorageType: storageType,
		Operation:   operation,
		StorageID:   storageID,
		Cause:       cause,
	}
}

// VisionAPIError Vision API 相關錯誤
type VisionAPIError struct {
	Model    string
	ImageURL string
	Cause    error
}

func (e *VisionAPIError) Error() string {
	return fmt.Sprintf("vision API error [%s] for image %s: %v", 
		e.Model, e.ImageURL, e.Cause)
}

func (e *VisionAPIError) Unwrap() error {
	return e.Cause
}

// NewVisionAPIError 建立 Vision API 錯誤
func NewVisionAPIError(model, imageURL string, cause error) *VisionAPIError {
	return &VisionAPIError{
		Model:    model,
		ImageURL: imageURL,
		Cause:    cause,
	}
}

// MediaEmbeddingError 媒體向量生成相關錯誤
type MediaEmbeddingError struct {
	Model    string
	DataType string // "text" or "image"
	Cause    error
}

func (e *MediaEmbeddingError) Error() string {
	return fmt.Sprintf("media embedding error [%s:%s]: %v", 
		e.Model, e.DataType, e.Cause)
}

func (e *MediaEmbeddingError) Unwrap() error {
	return e.Cause
}

// NewMediaEmbeddingError 建立媒體向量生成錯誤
func NewMediaEmbeddingError(model, dataType string, cause error) *MediaEmbeddingError {
	return &MediaEmbeddingError{
		Model:    model,
		DataType: dataType,
		Cause:    cause,
	}
}

// BatchProcessingError 批次處理錯誤
type BatchProcessingError struct {
	BatchID     string
	TotalFiles  int
	FailedFiles int
	Errors      []error
}

func (e *BatchProcessingError) Error() string {
	return fmt.Sprintf("batch processing error [%s]: %d/%d files failed", 
		e.BatchID, e.FailedFiles, e.TotalFiles)
}

// GetErrors 取得所有錯誤
func (e *BatchProcessingError) GetErrors() []error {
	return e.Errors
}

// NewBatchProcessingError 建立批次處理錯誤
func NewBatchProcessingError(batchID string, totalFiles, failedFiles int, errors []error) *BatchProcessingError {
	return &BatchProcessingError{
		BatchID:     batchID,
		TotalFiles:  totalFiles,
		FailedFiles: failedFiles,
		Errors:      errors,
	}
}

// IsRetryableError 檢查錯誤是否可重試
func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}
	
	// 檢查特定的錯誤類型
	switch {
	case errors.Is(err, ErrStorageNotAvailable):
		return true
	case errors.Is(err, ErrVisionAPIFailed):
		return true
	case errors.Is(err, ErrEmbeddingFailed):
		return true
	default:
		// 檢查是否為網路相關錯誤
		errStr := err.Error()
		retryablePatterns := []string{
			"timeout",
			"connection refused",
			"temporary failure",
			"rate limit",
			"service unavailable",
			"internal server error",
		}
		
		for _, pattern := range retryablePatterns {
			if contains(errStr, pattern) {
				return true
			}
		}
		
		return false
	}
}

// IsPermanentError 檢查錯誤是否為永久性錯誤
func IsPermanentError(err error) bool {
	if err == nil {
		return false
	}
	
	switch {
	case errors.Is(err, ErrUnsupportedImageFormat):
		return true
	case errors.Is(err, ErrImageTooLarge):
		return true
	case errors.Is(err, ErrInvalidImageData):
		return true
	case errors.Is(err, ErrInvalidStorageConfig):
		return true
	default:
		return false
	}
}

// contains 檢查字串是否包含子字串（不區分大小寫）
func contains(s, substr string) bool {
	return len(s) >= len(substr) && 
		   (s == substr || 
		    (len(s) > len(substr) && 
		     containsIgnoreCase(s, substr)))
}

// containsIgnoreCase 不區分大小寫的字串包含檢查
func containsIgnoreCase(s, substr string) bool {
	s = toLower(s)
	substr = toLower(substr)
	
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// toLower 簡單的小寫轉換
func toLower(s string) string {
	result := make([]byte, len(s))
	for i, b := range []byte(s) {
		if b >= 'A' && b <= 'Z' {
			result[i] = b + 32
		} else {
			result[i] = b
		}
	}
	return string(result)
}