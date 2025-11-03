package services

import (
	"context"
	"fmt"
	"io"
	"time"

	"semantic-text-processor/config"
	"semantic-text-processor/models"
)

// StorageService 統一的儲存服務
type StorageService struct {
	manager         *StorageManager
	config          *config.MultimodalConfig
	retryCount      int
	retryDelay      time.Duration
	healthCheckTTL  time.Duration
	lastHealthCheck map[models.StorageType]time.Time
}

// NewStorageService 建立新的儲存服務
func NewStorageService(config *config.MultimodalConfig) (*StorageService, error) {
	// 轉換配置格式
	storageConfig := &StorageConfig{
		Primary:  config.Storage.Primary,
		Fallback: config.Storage.Fallback,
		Configs:  make(map[string]map[string]interface{}),
	}
	
	// 轉換適配器配置
	for name, adapterConfig := range config.Storage.Configs {
		storageConfig.Configs[name] = adapterConfig.ToMap()
	}
	
	// 建立儲存管理器
	manager, err := NewStorageManager(storageConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage manager: %w", err)
	}
	
	service := &StorageService{
		manager:         manager,
		config:          config,
		retryCount:      3,
		retryDelay:      time.Second,
		healthCheckTTL:  5 * time.Minute,
		lastHealthCheck: make(map[models.StorageType]time.Time),
	}
	
	return service, nil
}

// Upload 上傳檔案（帶重試和降級機制）
func (s *StorageService) Upload(ctx context.Context, file io.Reader, metadata *models.MediaMetadata) (*models.StorageResult, error) {
	// 先嘗試主要儲存
	result, err := s.uploadWithRetry(ctx, s.manager.GetPrimaryAdapter(), file, metadata)
	if err == nil {
		return result, nil
	}
	
	// 如果主要儲存失敗且有備用儲存，嘗試備用儲存
	fallbackAdapter := s.manager.GetFallbackAdapter()
	if fallbackAdapter != nil && IsRetryableError(err) {
		// 重新讀取檔案內容（如果可能）
		if seeker, ok := file.(io.Seeker); ok {
			seeker.Seek(0, io.SeekStart)
			
			fallbackResult, fallbackErr := s.uploadWithRetry(ctx, fallbackAdapter, file, metadata)
			if fallbackErr == nil {
				// 記錄使用了備用儲存
				return fallbackResult, nil
			}
		}
	}
	
	return nil, NewStorageError(
		string(s.manager.GetPrimaryAdapter().GetStorageType()),
		"upload",
		"",
		err,
	)
}

// uploadWithRetry 帶重試的上傳
func (s *StorageService) uploadWithRetry(ctx context.Context, adapter MediaStorageAdapter, file io.Reader, metadata *models.MediaMetadata) (*models.StorageResult, error) {
	var lastErr error
	
	for attempt := 0; attempt <= s.retryCount; attempt++ {
		if attempt > 0 {
			// 等待重試延遲
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(s.retryDelay * time.Duration(attempt)):
			}
		}
		
		result, err := adapter.Upload(ctx, file, metadata)
		if err == nil {
			return result, nil
		}
		
		lastErr = err
		
		// 檢查是否為永久性錯誤，如果是則不重試
		if IsPermanentError(err) {
			break
		}
		
		// 檢查是否為可重試錯誤
		if !IsRetryableError(err) {
			break
		}
	}
	
	return nil, lastErr
}

// GetURL 取得檔案 URL
func (s *StorageService) GetURL(ctx context.Context, storageType models.StorageType, storageID string) (string, error) {
	adapter, err := s.getAdapterByType(storageType)
	if err != nil {
		return "", err
	}
	
	return adapter.GetURL(ctx, storageID)
}

// Download 下載檔案
func (s *StorageService) Download(ctx context.Context, storageType models.StorageType, storageID string) (io.ReadCloser, error) {
	adapter, err := s.getAdapterByType(storageType)
	if err != nil {
		return nil, err
	}
	
	return adapter.Download(ctx, storageID)
}

// Delete 刪除檔案
func (s *StorageService) Delete(ctx context.Context, storageType models.StorageType, storageID string) error {
	adapter, err := s.getAdapterByType(storageType)
	if err != nil {
		return err
	}
	
	return adapter.Delete(ctx, storageID)
}

// ScanFolder 掃描資料夾
func (s *StorageService) ScanFolder(ctx context.Context, folderPath string) ([]models.MediaFile, error) {
	// 使用主要儲存適配器進行掃描
	return s.manager.GetPrimaryAdapter().ScanFolder(ctx, folderPath)
}

// HealthCheck 健康檢查
func (s *StorageService) HealthCheck(ctx context.Context) error {
	// 檢查主要儲存
	primaryAdapter := s.manager.GetPrimaryAdapter()
	if err := s.checkAdapterHealth(ctx, primaryAdapter); err != nil {
		return fmt.Errorf("primary storage health check failed: %w", err)
	}
	
	// 檢查備用儲存（如果存在）
	fallbackAdapter := s.manager.GetFallbackAdapter()
	if fallbackAdapter != nil {
		if err := s.checkAdapterHealth(ctx, fallbackAdapter); err != nil {
			// 備用儲存失敗不影響整體健康狀態，但要記錄
			// 可以在這裡記錄警告日誌
		}
	}
	
	return nil
}

// checkAdapterHealth 檢查適配器健康狀態
func (s *StorageService) checkAdapterHealth(ctx context.Context, adapter MediaStorageAdapter) error {
	storageType := adapter.GetStorageType()
	
	// 檢查是否需要進行健康檢查
	if lastCheck, exists := s.lastHealthCheck[storageType]; exists {
		if time.Since(lastCheck) < s.healthCheckTTL {
			return nil // 在 TTL 內，跳過檢查
		}
	}
	
	// 執行健康檢查
	err := adapter.HealthCheck(ctx)
	
	// 更新最後檢查時間
	if err == nil {
		s.lastHealthCheck[storageType] = time.Now()
	}
	
	return err
}

// getAdapterByType 根據類型取得適配器
func (s *StorageService) getAdapterByType(storageType models.StorageType) (MediaStorageAdapter, error) {
	primaryAdapter := s.manager.GetPrimaryAdapter()
	if primaryAdapter.GetStorageType() == storageType {
		return primaryAdapter, nil
	}
	
	fallbackAdapter := s.manager.GetFallbackAdapter()
	if fallbackAdapter != nil && fallbackAdapter.GetStorageType() == storageType {
		return fallbackAdapter, nil
	}
	
	return nil, fmt.Errorf("storage adapter not found for type: %s", storageType)
}

// GetPrimaryStorageType 取得主要儲存類型
func (s *StorageService) GetPrimaryStorageType() models.StorageType {
	return s.manager.GetPrimaryAdapter().GetStorageType()
}

// GetFallbackStorageType 取得備用儲存類型
func (s *StorageService) GetFallbackStorageType() models.StorageType {
	fallbackAdapter := s.manager.GetFallbackAdapter()
	if fallbackAdapter == nil {
		return ""
	}
	return fallbackAdapter.GetStorageType()
}

// SwitchPrimaryStorage 切換主要儲存
func (s *StorageService) SwitchPrimaryStorage(newPrimary models.StorageType) error {
	return s.manager.SwitchPrimary(newPrimary)
}

// GetStorageStats 取得儲存統計資訊
func (s *StorageService) GetStorageStats() map[string]interface{} {
	stats := map[string]interface{}{
		"primary_storage":   string(s.GetPrimaryStorageType()),
		"fallback_storage":  string(s.GetFallbackStorageType()),
		"retry_count":       s.retryCount,
		"retry_delay":       s.retryDelay.String(),
		"health_check_ttl":  s.healthCheckTTL.String(),
		"last_health_check": make(map[string]string),
	}
	
	// 轉換健康檢查時間
	healthCheckMap := stats["last_health_check"].(map[string]string)
	for storageType, lastCheck := range s.lastHealthCheck {
		healthCheckMap[string(storageType)] = lastCheck.Format(time.RFC3339)
	}
	
	return stats
}

// ValidateFile 驗證檔案
func (s *StorageService) ValidateFile(metadata *models.MediaMetadata) error {
	// 檢查檔案大小
	maxSize := s.config.Processing.MaxImageSize
	if metadata.Size > maxSize {
		return fmt.Errorf("file size %d exceeds maximum allowed size %d", metadata.Size, maxSize)
	}
	
	// 檢查檔案格式
	if !models.IsImageFile(metadata.OriginalFilename) {
		return ErrUnsupportedImageFormat
	}
	
	// 檢查檔案雜湊
	if len(metadata.Hash) != 64 {
		return fmt.Errorf("invalid file hash length: expected 64, got %d", len(metadata.Hash))
	}
	
	return nil
}

// Close 關閉儲存服務
func (s *StorageService) Close() error {
	// 清理資源
	s.lastHealthCheck = nil
	return nil
}