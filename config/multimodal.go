package config

import (
	"fmt"
	"time"

	"semantic-text-processor/models"
)

// MultimodalConfig 多模態系統配置
type MultimodalConfig struct {
	Storage   MultimodalStorageConfig   `json:"storage" yaml:"storage"`
	Vision    VisionConfig    `json:"vision" yaml:"vision"`
	Embedding MultimodalEmbeddingConfig `json:"embedding" yaml:"embedding"`
	Processing ProcessingConfig `json:"processing" yaml:"processing"`
}

// MultimodalStorageConfig 多模態儲存配置
type MultimodalStorageConfig struct {
	Primary  models.StorageType            `json:"primary" yaml:"primary"`
	Fallback models.StorageType            `json:"fallback,omitempty" yaml:"fallback,omitempty"`
	Configs  map[string]StorageAdapterConfig `json:"configs" yaml:"configs"`
}

// StorageAdapterConfig 儲存適配器配置
type StorageAdapterConfig struct {
	// 本地儲存配置
	BasePath string `json:"base_path,omitempty" yaml:"base_path,omitempty"`
	BaseURL  string `json:"base_url,omitempty" yaml:"base_url,omitempty"`
	
	// Supabase 儲存配置
	URL    string `json:"url,omitempty" yaml:"url,omitempty"`
	APIKey string `json:"api_key,omitempty" yaml:"api_key,omitempty"`
	Bucket string `json:"bucket,omitempty" yaml:"bucket,omitempty"`
	
	// Google Drive 配置（未來使用）
	CredentialsPath string `json:"credentials_path,omitempty" yaml:"credentials_path,omitempty"`
	FolderID        string `json:"folder_id,omitempty" yaml:"folder_id,omitempty"`
	
	// 通用配置
	MaxFileSize int64         `json:"max_file_size,omitempty" yaml:"max_file_size,omitempty"`
	Timeout     time.Duration `json:"timeout,omitempty" yaml:"timeout,omitempty"`
}

// VisionConfig Vision AI 配置
type VisionConfig struct {
	Provider    string        `json:"provider" yaml:"provider"` // "openai" or "anthropic"
	APIKey      string        `json:"api_key" yaml:"api_key"`
	Model       string        `json:"model" yaml:"model"`
	MaxTokens   int           `json:"max_tokens" yaml:"max_tokens"`
	Temperature float64       `json:"temperature" yaml:"temperature"`
	Language    string        `json:"language" yaml:"language"`
	DetailLevel string        `json:"detail_level" yaml:"detail_level"` // "low", "medium", "high"
	Timeout     time.Duration `json:"timeout" yaml:"timeout"`
	RetryCount  int           `json:"retry_count" yaml:"retry_count"`
}

// MultimodalEmbeddingConfig 多模態向量生成配置
type MultimodalEmbeddingConfig struct {
	// 文字向量配置
	TextProvider string        `json:"text_provider" yaml:"text_provider"` // "openai"
	TextModel    string        `json:"text_model" yaml:"text_model"`
	TextAPIKey   string        `json:"text_api_key" yaml:"text_api_key"`
	
	// 圖片向量配置
	ImageProvider string        `json:"image_provider" yaml:"image_provider"` // "clip" or "openai"
	ImageModel    string        `json:"image_model" yaml:"image_model"`
	ImageAPIKey   string        `json:"image_api_key,omitempty" yaml:"image_api_key,omitempty"`
	
	// CLIP 本地配置
	CLIPModelPath string `json:"clip_model_path,omitempty" yaml:"clip_model_path,omitempty"`
	
	// 通用配置
	Dimensions int           `json:"dimensions" yaml:"dimensions"`
	Timeout    time.Duration `json:"timeout" yaml:"timeout"`
	RetryCount int           `json:"retry_count" yaml:"retry_count"`
}

// ProcessingConfig 處理配置
type ProcessingConfig struct {
	// 批次處理配置
	BatchSize        int           `json:"batch_size" yaml:"batch_size"`
	MaxConcurrency   int           `json:"max_concurrency" yaml:"max_concurrency"`
	ProcessingTimeout time.Duration `json:"processing_timeout" yaml:"processing_timeout"`
	
	// 圖片處理配置
	MaxImageSize     int64    `json:"max_image_size" yaml:"max_image_size"`
	SupportedFormats []string `json:"supported_formats" yaml:"supported_formats"`
	AutoAnalyze      bool     `json:"auto_analyze" yaml:"auto_analyze"`
	AutoEmbed        bool     `json:"auto_embed" yaml:"auto_embed"`
	
	// 去重配置
	EnableDeduplication bool    `json:"enable_deduplication" yaml:"enable_deduplication"`
	SimilarityThreshold float64 `json:"similarity_threshold" yaml:"similarity_threshold"`
	
	// 快取配置
	EnableCache    bool          `json:"enable_cache" yaml:"enable_cache"`
	CacheTTL       time.Duration `json:"cache_ttl" yaml:"cache_ttl"`
	CacheSize      int           `json:"cache_size" yaml:"cache_size"`
}

// DefaultMultimodalConfig 預設多模態配置
func DefaultMultimodalConfig() *MultimodalConfig {
	return &MultimodalConfig{
		Storage: MultimodalStorageConfig{
			Primary:  models.StorageTypeSupabase,
			Fallback: models.StorageTypeLocal,
			Configs: map[string]StorageAdapterConfig{
				string(models.StorageTypeLocal): {
					BasePath:    "/tmp/ink-images",
					BaseURL:     "file:///tmp/ink-images",
					MaxFileSize: 10 * 1024 * 1024, // 10MB
					Timeout:     30 * time.Second,
				},
				string(models.StorageTypeSupabase): {
					Bucket:      "ink-images",
					MaxFileSize: 50 * 1024 * 1024, // 50MB
					Timeout:     60 * time.Second,
				},
			},
		},
		Vision: VisionConfig{
			Provider:    "openai",
			Model:       "gpt-4-vision-preview",
			MaxTokens:   1000,
			Temperature: 0.1,
			Language:    "zh-TW",
			DetailLevel: "medium",
			Timeout:     30 * time.Second,
			RetryCount:  3,
		},
		Embedding: MultimodalEmbeddingConfig{
			TextProvider:  "openai",
			TextModel:     "text-embedding-3-small",
			ImageProvider: "clip",
			ImageModel:    "clip-vit-b-32",
			Dimensions:    512,
			Timeout:       30 * time.Second,
			RetryCount:    3,
		},
		Processing: ProcessingConfig{
			BatchSize:           10,
			MaxConcurrency:      5,
			ProcessingTimeout:   5 * time.Minute,
			MaxImageSize:        50 * 1024 * 1024, // 50MB
			SupportedFormats:    []string{".png", ".jpg", ".jpeg", ".gif", ".webp", ".bmp"},
			AutoAnalyze:         true,
			AutoEmbed:           true,
			EnableDeduplication: true,
			SimilarityThreshold: 0.95,
			EnableCache:         true,
			CacheTTL:            1 * time.Hour,
			CacheSize:           1000,
		},
	}
}

// Validate 驗證配置
func (c *MultimodalConfig) Validate() error {
	// 驗證儲存配置
	if c.Storage.Primary == "" {
		return fmt.Errorf("primary storage type is required")
	}
	
	primaryConfig, exists := c.Storage.Configs[string(c.Storage.Primary)]
	if !exists {
		return fmt.Errorf("missing config for primary storage type: %s", c.Storage.Primary)
	}
	
	if err := c.validateStorageConfig(c.Storage.Primary, primaryConfig); err != nil {
		return fmt.Errorf("invalid primary storage config: %w", err)
	}
	
	// 驗證 Vision 配置
	if c.Vision.APIKey == "" {
		return fmt.Errorf("vision API key is required")
	}
	
	// 驗證 Embedding 配置
	if c.Embedding.TextAPIKey == "" {
		return fmt.Errorf("text embedding API key is required")
	}
	
	if c.Embedding.Dimensions <= 0 {
		return fmt.Errorf("embedding dimensions must be positive")
	}
	
	// 驗證處理配置
	if c.Processing.BatchSize <= 0 {
		return fmt.Errorf("batch size must be positive")
	}
	
	if c.Processing.MaxConcurrency <= 0 {
		return fmt.Errorf("max concurrency must be positive")
	}
	
	return nil
}

// validateStorageConfig 驗證儲存配置
func (c *MultimodalConfig) validateStorageConfig(storageType models.StorageType, config StorageAdapterConfig) error {
	switch storageType {
	case models.StorageTypeLocal:
		if config.BasePath == "" {
			return fmt.Errorf("base_path is required for local storage")
		}
		if config.BaseURL == "" {
			return fmt.Errorf("base_url is required for local storage")
		}
	case models.StorageTypeSupabase:
		if config.URL == "" {
			return fmt.Errorf("url is required for supabase storage")
		}
		if config.APIKey == "" {
			return fmt.Errorf("api_key is required for supabase storage")
		}
		if config.Bucket == "" {
			return fmt.Errorf("bucket is required for supabase storage")
		}
	case models.StorageTypeGoogleDrive:
		if config.CredentialsPath == "" {
			return fmt.Errorf("credentials_path is required for google drive storage")
		}
		if config.FolderID == "" {
			return fmt.Errorf("folder_id is required for google drive storage")
		}
	}
	
	return nil
}

// ToStorageAdapterConfig 轉換為儲存適配器配置
func (c *StorageAdapterConfig) ToMap() map[string]interface{} {
	config := make(map[string]interface{})
	
	if c.BasePath != "" {
		config["base_path"] = c.BasePath
	}
	if c.BaseURL != "" {
		config["base_url"] = c.BaseURL
	}
	if c.URL != "" {
		config["url"] = c.URL
	}
	if c.APIKey != "" {
		config["api_key"] = c.APIKey
	}
	if c.Bucket != "" {
		config["bucket"] = c.Bucket
	}
	if c.CredentialsPath != "" {
		config["credentials_path"] = c.CredentialsPath
	}
	if c.FolderID != "" {
		config["folder_id"] = c.FolderID
	}
	if c.MaxFileSize > 0 {
		config["max_file_size"] = c.MaxFileSize
	}
	if c.Timeout > 0 {
		config["timeout"] = c.Timeout
	}
	
	return config
}