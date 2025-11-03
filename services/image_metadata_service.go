package services

import (
	"bytes"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"path/filepath"
	"strings"

	"semantic-text-processor/models"
)

// ImageMetadataService 圖片元資料提取服務
type ImageMetadataService struct {
	hashService *HashService
}

// NewImageMetadataService 建立新的圖片元資料服務
func NewImageMetadataService() *ImageMetadataService {
	return &ImageMetadataService{
		hashService: NewHashService(),
	}
}

// ExtractMetadata 從圖片檔案提取完整元資料
func (s *ImageMetadataService) ExtractMetadata(reader io.Reader, originalFilename string) (*models.MediaMetadata, error) {
	// 讀取所有資料到記憶體（用於多次處理）
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read image data: %w", err)
	}
	
	// 計算檔案雜湊
	hash := s.hashService.CalculateHashFromBytes(data)
	
	// 取得圖片尺寸
	width, height, err := s.getImageDimensions(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to get image dimensions: %w", err)
	}
	
	// 建立元資料
	metadata := &models.MediaMetadata{
		OriginalFilename: originalFilename,
		ContentType:      models.GetImageContentType(originalFilename),
		Size:             int64(len(data)),
		Width:            width,
		Height:           height,
		Hash:             hash,
	}
	
	return metadata, nil
}

// ExtractMetadataFromStream 從串流提取元資料（不載入全部到記憶體）
func (s *ImageMetadataService) ExtractMetadataFromStream(reader io.Reader, originalFilename string) (*models.MediaMetadata, []byte, error) {
	// 使用 TeeHashReader 同時讀取和計算雜湊
	hashReader := NewTeeHashReader(reader)
	
	// 讀取所有資料
	data, err := io.ReadAll(hashReader)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read image data: %w", err)
	}
	
	// 取得雜湊和大小
	hash := hashReader.GetHash()
	size := hashReader.GetSize()
	
	// 取得圖片尺寸
	width, height, err := s.getImageDimensions(bytes.NewReader(data))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get image dimensions: %w", err)
	}
	
	// 建立元資料
	metadata := &models.MediaMetadata{
		OriginalFilename: originalFilename,
		ContentType:      models.GetImageContentType(originalFilename),
		Size:             size,
		Width:            width,
		Height:           height,
		Hash:             hash,
	}
	
	return metadata, data, nil
}

// getImageDimensions 取得圖片尺寸
func (s *ImageMetadataService) getImageDimensions(reader io.Reader) (width, height int, err error) {
	// 使用 Go 標準庫解析圖片
	config, _, err := image.DecodeConfig(reader)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to decode image config: %w", err)
	}
	
	return config.Width, config.Height, nil
}

// ValidateImageFormat 驗證圖片格式
func (s *ImageMetadataService) ValidateImageFormat(reader io.Reader, filename string) error {
	// 檢查副檔名
	if !models.IsImageFile(filename) {
		return models.ErrUnsupportedImageFormat
	}
	
	// 嘗試解析圖片標頭
	_, format, err := image.DecodeConfig(reader)
	if err != nil {
		return fmt.Errorf("invalid image format: %w", err)
	}
	
	// 檢查支援的格式
	supportedFormats := []string{"jpeg", "png", "gif", "webp"}
	formatSupported := false
	for _, supported := range supportedFormats {
		if format == supported {
			formatSupported = true
			break
		}
	}
	
	if !formatSupported {
		return fmt.Errorf("unsupported image format: %s", format)
	}
	
	return nil
}

// GetImageInfo 取得圖片基本資訊
func (s *ImageMetadataService) GetImageInfo(reader io.Reader) (*ImageInfo, error) {
	config, format, err := image.DecodeConfig(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}
	
	return &ImageInfo{
		Width:      config.Width,
		Height:     config.Height,
		Format:     format,
		ColorModel: fmt.Sprintf("%T", config.ColorModel),
	}, nil
}

// ImageInfo 圖片基本資訊
type ImageInfo struct {
	Width      int    `json:"width"`
	Height     int    `json:"height"`
	Format     string `json:"format"`
	ColorModel string `json:"color_model"`
}

// CalculateAspectRatio 計算長寬比
func (s *ImageMetadataService) CalculateAspectRatio(width, height int) float64 {
	if height == 0 {
		return 0
	}
	return float64(width) / float64(height)
}

// GetImageCategory 根據尺寸和長寬比分類圖片
func (s *ImageMetadataService) GetImageCategory(width, height int) string {
	aspectRatio := s.CalculateAspectRatio(width, height)
	totalPixels := width * height
	
	// 根據解析度分類
	var sizeCategory string
	switch {
	case totalPixels < 100000: // < 0.1MP
		sizeCategory = "thumbnail"
	case totalPixels < 1000000: // < 1MP
		sizeCategory = "small"
	case totalPixels < 5000000: // < 5MP
		sizeCategory = "medium"
	case totalPixels < 20000000: // < 20MP
		sizeCategory = "large"
	default:
		sizeCategory = "very_large"
	}
	
	// 根據長寬比分類
	var orientationCategory string
	switch {
	case aspectRatio > 1.5:
		orientationCategory = "landscape"
	case aspectRatio < 0.67:
		orientationCategory = "portrait"
	default:
		orientationCategory = "square"
	}
	
	return fmt.Sprintf("%s_%s", sizeCategory, orientationCategory)
}

// IsValidImageSize 檢查圖片尺寸是否有效
func (s *ImageMetadataService) IsValidImageSize(width, height int, maxSize int64) bool {
	if width <= 0 || height <= 0 {
		return false
	}
	
	// 檢查是否超過最大像素數
	totalPixels := int64(width) * int64(height)
	maxPixels := maxSize / 4 // 假設每像素 4 bytes (RGBA)
	
	return totalPixels <= maxPixels
}

// GenerateImageSummary 生成圖片摘要
func (s *ImageMetadataService) GenerateImageSummary(metadata *models.MediaMetadata) map[string]interface{} {
	aspectRatio := s.CalculateAspectRatio(metadata.Width, metadata.Height)
	category := s.GetImageCategory(metadata.Width, metadata.Height)
	
	// 取得檔案格式
	format := strings.ToLower(strings.TrimPrefix(filepath.Ext(metadata.OriginalFilename), "."))
	
	return map[string]interface{}{
		"filename":     metadata.OriginalFilename,
		"format":       format,
		"content_type": metadata.ContentType,
		"size_bytes":   metadata.Size,
		"size_mb":      float64(metadata.Size) / (1024 * 1024),
		"width":        metadata.Width,
		"height":       metadata.Height,
		"aspect_ratio": aspectRatio,
		"category":     category,
		"total_pixels": metadata.Width * metadata.Height,
		"hash":         metadata.Hash,
		"hash_short":   metadata.Hash[:16], // 短版雜湊用於顯示
	}
}