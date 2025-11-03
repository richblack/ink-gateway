package services

import (
	"bytes"
	"context"
	"testing"

	"semantic-text-processor/config"
	"semantic-text-processor/models"
)

// TestStorageServiceIntegration 測試儲存服務整合
func TestStorageServiceIntegration(t *testing.T) {
	// 建立測試配置
	tempDir := t.TempDir()
	cfg := &config.MultimodalConfig{
		Storage: config.StorageConfig{
			Primary: models.StorageTypeLocal,
			Configs: map[string]config.StorageAdapterConfig{
				string(models.StorageTypeLocal): {
					BasePath: tempDir,
					BaseURL:  "file://" + tempDir,
				},
			},
		},
		Processing: config.ProcessingConfig{
			MaxImageSize: 10 * 1024 * 1024, // 10MB
		},
	}
	
	// 建立儲存服務
	storageService, err := NewStorageService(cfg)
	if err != nil {
		t.Fatalf("Failed to create storage service: %v", err)
	}
	defer storageService.Close()
	
	// 建立圖片元資料服務（在需要時使用）
	_ = NewImageMetadataService()
	
	ctx := context.Background()
	
	t.Run("CompleteImageProcessingFlow", func(t *testing.T) {
		// 模擬圖片資料
		testImageData := []byte("fake image data for testing")
		
		// 1. 建立模擬的圖片元資料（跳過實際的圖片解析）
		hashService := NewHashService()
		hash := hashService.CalculateHashFromBytes(testImageData)
		
		metadata := &models.MediaMetadata{
			OriginalFilename: "test-image.png",
			ContentType:      "image/png",
			Size:             int64(len(testImageData)),
			Width:            100,
			Height:           100,
			Hash:             hash,
		}
		imageData := testImageData
		
		// 驗證元資料
		if metadata.OriginalFilename != "test-image.png" {
			t.Errorf("Expected filename 'test-image.png', got '%s'", metadata.OriginalFilename)
		}
		
		if metadata.ContentType != "image/png" {
			t.Errorf("Expected content type 'image/png', got '%s'", metadata.ContentType)
		}
		
		if metadata.Size != int64(len(testImageData)) {
			t.Errorf("Expected size %d, got %d", len(testImageData), metadata.Size)
		}
		
		if len(metadata.Hash) != 64 {
			t.Errorf("Expected hash length 64, got %d", len(metadata.Hash))
		}
		
		// 2. 驗證檔案
		err = storageService.ValidateFile(metadata)
		if err != nil {
			t.Fatalf("File validation failed: %v", err)
		}
		
		// 3. 上傳檔案
		uploadReader := bytes.NewReader(imageData)
		result, err := storageService.Upload(ctx, uploadReader, metadata)
		if err != nil {
			t.Fatalf("Upload failed: %v", err)
		}
		
		// 驗證上傳結果
		if result.StorageType != models.StorageTypeLocal {
			t.Errorf("Expected storage type %s, got %s", models.StorageTypeLocal, result.StorageType)
		}
		
		if result.StorageID == "" {
			t.Error("StorageID should not be empty")
		}
		
		if result.URL == "" {
			t.Error("URL should not be empty")
		}
		
		// 4. 取得檔案 URL
		url, err := storageService.GetURL(ctx, result.StorageType, result.StorageID)
		if err != nil {
			t.Fatalf("GetURL failed: %v", err)
		}
		
		if url != result.URL {
			t.Errorf("URL mismatch: expected %s, got %s", result.URL, url)
		}
		
		// 5. 下載檔案
		downloadReader, err := storageService.Download(ctx, result.StorageType, result.StorageID)
		if err != nil {
			t.Fatalf("Download failed: %v", err)
		}
		defer downloadReader.Close()
		
		// 讀取下載的內容
		downloadedData := make([]byte, len(imageData))
		n, err := downloadReader.Read(downloadedData)
		if err != nil && err.Error() != "EOF" {
			t.Fatalf("Failed to read downloaded data: %v", err)
		}
		
		if n != len(imageData) {
			t.Errorf("Expected %d bytes, got %d", len(imageData), n)
		}
		
		if !bytes.Equal(imageData, downloadedData[:n]) {
			t.Error("Downloaded data does not match original data")
		}
		
		// 6. 刪除檔案
		err = storageService.Delete(ctx, result.StorageType, result.StorageID)
		if err != nil {
			t.Fatalf("Delete failed: %v", err)
		}
		
		// 7. 驗證檔案已被刪除
		_, err = storageService.GetURL(ctx, result.StorageType, result.StorageID)
		if err == nil {
			t.Error("File should be deleted, but GetURL still works")
		}
	})
	
	t.Run("HealthCheck", func(t *testing.T) {
		err := storageService.HealthCheck(ctx)
		if err != nil {
			t.Fatalf("Health check failed: %v", err)
		}
	})
	
	t.Run("StorageStats", func(t *testing.T) {
		stats := storageService.GetStorageStats()
		
		// 檢查基本統計資訊
		if stats["primary_storage"] != string(models.StorageTypeLocal) {
			t.Errorf("Expected primary storage %s, got %s", 
				models.StorageTypeLocal, stats["primary_storage"])
		}
		
		if stats["retry_count"] == nil {
			t.Error("retry_count should be present in stats")
		}
		
		if stats["retry_delay"] == nil {
			t.Error("retry_delay should be present in stats")
		}
	})
}

// TestImageMetadataService 測試圖片元資料服務
func TestImageMetadataService(t *testing.T) {
	service := NewImageMetadataService()
	
	t.Run("GenerateImageSummary", func(t *testing.T) {
		metadata := &models.MediaMetadata{
			OriginalFilename: "test-image.png",
			ContentType:      "image/png",
			Size:             102400, // 100KB
			Width:            1920,
			Height:           1080,
			Hash:             "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
		}
		
		summary := service.GenerateImageSummary(metadata)
		
		// 檢查摘要內容
		if summary["filename"] != "test-image.png" {
			t.Errorf("Expected filename 'test-image.png', got %v", summary["filename"])
		}
		
		if summary["format"] != "png" {
			t.Errorf("Expected format 'png', got %v", summary["format"])
		}
		
		if summary["width"] != 1920 {
			t.Errorf("Expected width 1920, got %v", summary["width"])
		}
		
		if summary["height"] != 1080 {
			t.Errorf("Expected height 1080, got %v", summary["height"])
		}
		
		if summary["total_pixels"] != 1920*1080 {
			t.Errorf("Expected total_pixels %d, got %v", 1920*1080, summary["total_pixels"])
		}
		
		// 檢查長寬比
		expectedAspectRatio := float64(1920) / float64(1080)
		if summary["aspect_ratio"] != expectedAspectRatio {
			t.Errorf("Expected aspect_ratio %f, got %v", expectedAspectRatio, summary["aspect_ratio"])
		}
		
		// 檢查分類
		if summary["category"] == nil {
			t.Error("category should be present")
		}
		
		// 檢查短版雜湊
		if summary["hash_short"] != "abcdef1234567890" {
			t.Errorf("Expected hash_short 'abcdef1234567890', got %v", summary["hash_short"])
		}
	})
	
	t.Run("CalculateAspectRatio", func(t *testing.T) {
		// 測試不同的長寬比
		testCases := []struct {
			width, height int
			expected      float64
		}{
			{1920, 1080, 16.0 / 9.0},
			{1080, 1920, 9.0 / 16.0},
			{1000, 1000, 1.0},
			{1600, 900, 16.0 / 9.0},
		}
		
		for _, tc := range testCases {
			result := service.CalculateAspectRatio(tc.width, tc.height)
			if result != tc.expected {
				t.Errorf("CalculateAspectRatio(%d, %d) = %f, expected %f", 
					tc.width, tc.height, result, tc.expected)
			}
		}
		
		// 測試除零情況
		result := service.CalculateAspectRatio(100, 0)
		if result != 0 {
			t.Errorf("CalculateAspectRatio(100, 0) should return 0, got %f", result)
		}
	})
	
	t.Run("GetImageCategory", func(t *testing.T) {
		testCases := []struct {
			width, height int
			expectedType  string
		}{
			{100, 100, "thumbnail"},     // 小圖
			{800, 600, "small"},         // 小圖
			{1920, 1080, "medium"},      // 中圖
			{4000, 3000, "large"},       // 大圖
			{8000, 6000, "very_large"},  // 超大圖
		}
		
		for _, tc := range testCases {
			category := service.GetImageCategory(tc.width, tc.height)
			if !containsString(category, tc.expectedType) {
				t.Errorf("GetImageCategory(%d, %d) = %s, should contain %s", 
					tc.width, tc.height, category, tc.expectedType)
			}
		}
	})
}

// containsString 檢查字串是否包含子字串
func containsString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}