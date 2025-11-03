package services

import (
	"context"
	"io"

	"semantic-text-processor/models"
)

// MediaStorageAdapter 定義統一的儲存介面
type MediaStorageAdapter interface {
	// 上傳檔案並返回儲存資訊
	Upload(ctx context.Context, file io.Reader, metadata *models.MediaMetadata) (*models.StorageResult, error)
	
	// 根據 storage_id 取得存取 URL
	GetURL(ctx context.Context, storageID string) (string, error)
	
	// 下載檔案內容
	Download(ctx context.Context, storageID string) (io.ReadCloser, error)
	
	// 刪除檔案
	Delete(ctx context.Context, storageID string) error
	
	// 掃描資料夾中的圖片檔案
	ScanFolder(ctx context.Context, folderPath string) ([]models.MediaFile, error)
	
	// 取得儲存類型
	GetStorageType() models.StorageType
	
	// 健康檢查
	HealthCheck(ctx context.Context) error
}

// MediaProcessor 圖片處理服務介面
type MediaProcessor interface {
	// 處理單張圖片（上傳、分析、索引）
	ProcessImage(ctx context.Context, req *models.ProcessImageRequest) (*models.ProcessImageResult, error)
	
	// 批次處理圖片
	BatchProcessImages(ctx context.Context, req *models.BatchProcessRequest) (*models.BatchProcessResult, error)
	
	// 分析圖片內容
	AnalyzeImage(ctx context.Context, imageURL string) (*models.ImageAnalysis, error)
	
	// 生成圖片向量
	GenerateImageEmbedding(ctx context.Context, imageURL string) ([]float64, error)
	
	// 計算檔案雜湊
	CalculateHash(ctx context.Context, file io.Reader) (string, error)
}

// VisionAIService Vision AI 服務介面
type VisionAIService interface {
	AnalyzeImage(ctx context.Context, imageURL string, options *models.AnalysisOptions) (*models.ImageAnalysis, error)
}

// ImageEmbeddingService 圖片向量化服務介面
type ImageEmbeddingService interface {
	GenerateEmbedding(ctx context.Context, imageURL string) ([]float64, error)
	GenerateBatchEmbeddings(ctx context.Context, imageURLs []string) ([][]float64, error)
}

// MultimodalSearchService 多模態搜尋服務介面
type MultimodalSearchService interface {
	// 文字搜尋（包含圖片 AI 描述）
	SearchText(ctx context.Context, req *models.MultimodalSearchRequest) (*models.MultimodalSearchResponse, error)
	
	// 圖片搜尋（向量相似度）
	SearchImages(ctx context.Context, req *models.MultimodalSearchRequest) (*models.MultimodalSearchResponse, error)
	
	// 混合搜尋（文字+圖片）
	HybridSearch(ctx context.Context, req *models.MultimodalSearchRequest) (*models.MultimodalSearchResponse, error)
	
	// 以圖搜圖
	SearchByImage(ctx context.Context, imageURL string, limit int, minSimilarity float64) (*models.MultimodalSearchResponse, error)
	
	// 為 Slide Generator 推薦圖片
	RecommendImagesForSlides(ctx context.Context, req *models.SlideImageRequest) (*models.ImageRecommendationResponse, error)
}

// StorageAdapterFactory 儲存適配器工廠
type StorageAdapterFactory interface {
	CreateAdapter(storageType models.StorageType, config map[string]interface{}) (MediaStorageAdapter, error)
	GetSupportedTypes() []models.StorageType
}