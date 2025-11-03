package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"semantic-text-processor/models"
)

// supabaseStorageAdapter Supabase Storage 適配器
type supabaseStorageAdapter struct {
	url        string
	apiKey     string
	bucket     string
	httpClient *http.Client
}

// NewSupabaseStorageAdapter 建立新的 Supabase 儲存適配器
func NewSupabaseStorageAdapter(url, apiKey, bucket string) MediaStorageAdapter {
	return &supabaseStorageAdapter{
		url:    strings.TrimRight(url, "/"),
		apiKey: apiKey,
		bucket: bucket,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Upload 上傳檔案到 Supabase Storage
func (s *supabaseStorageAdapter) Upload(ctx context.Context, file io.Reader, metadata *models.MediaMetadata) (*models.StorageResult, error) {
	// 1. 生成檔案路徑 (使用 hash 避免重複)
	timestamp := time.Now().Unix()
	ext := filepath.Ext(metadata.OriginalFilename)
	filename := fmt.Sprintf("%s_%d%s", metadata.Hash[:16], timestamp, ext)
	filePath := fmt.Sprintf("images/%s/%s", time.Now().Format("2006/01"), filename)
	
	// 2. 讀取檔案內容
	fileData, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read file data: %w", err)
	}
	
	// 3. 建立上傳請求
	uploadURL := fmt.Sprintf("%s/storage/v1/object/%s/%s", s.url, s.bucket, filePath)
	
	req, err := http.NewRequestWithContext(ctx, "POST", uploadURL, bytes.NewReader(fileData))
	if err != nil {
		return nil, fmt.Errorf("failed to create upload request: %w", err)
	}
	
	// 設定標頭
	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("Content-Type", metadata.ContentType)
	req.Header.Set("x-upsert", "false") // 不覆蓋現有檔案
	
	// 4. 執行上傳
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("upload request failed: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("upload failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	// 5. 取得公開 URL
	publicURL := fmt.Sprintf("%s/storage/v1/object/public/%s/%s", s.url, s.bucket, filePath)
	
	return &models.StorageResult{
		StorageID:   filePath,
		URL:         publicURL,
		StorageType: models.StorageTypeSupabase,
		UploadedAt:  time.Now(),
	}, nil
}

// GetURL 根據 storage_id 取得存取 URL
func (s *supabaseStorageAdapter) GetURL(ctx context.Context, storageID string) (string, error) {
	// 檢查檔案是否存在
	checkURL := fmt.Sprintf("%s/storage/v1/object/info/public/%s/%s", s.url, s.bucket, storageID)
	
	req, err := http.NewRequestWithContext(ctx, "GET", checkURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create check request: %w", err)
	}
	
	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("check request failed: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode == 404 {
		return "", fmt.Errorf("file not found: %s", storageID)
	}
	
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("check failed with status %d", resp.StatusCode)
	}
	
	// 返回公開 URL
	publicURL := fmt.Sprintf("%s/storage/v1/object/public/%s/%s", s.url, s.bucket, storageID)
	return publicURL, nil
}

// Download 下載檔案內容
func (s *supabaseStorageAdapter) Download(ctx context.Context, storageID string) (io.ReadCloser, error) {
	downloadURL := fmt.Sprintf("%s/storage/v1/object/%s/%s", s.url, s.bucket, storageID)
	
	req, err := http.NewRequestWithContext(ctx, "GET", downloadURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create download request: %w", err)
	}
	
	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download request failed: %w", err)
	}
	
	if resp.StatusCode == 404 {
		resp.Body.Close()
		return nil, fmt.Errorf("file not found: %s", storageID)
	}
	
	if resp.StatusCode >= 400 {
		resp.Body.Close()
		return nil, fmt.Errorf("download failed with status %d", resp.StatusCode)
	}
	
	return resp.Body, nil
}

// Delete 刪除檔案
func (s *supabaseStorageAdapter) Delete(ctx context.Context, storageID string) error {
	deleteURL := fmt.Sprintf("%s/storage/v1/object/%s/%s", s.url, s.bucket, storageID)
	
	req, err := http.NewRequestWithContext(ctx, "DELETE", deleteURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create delete request: %w", err)
	}
	
	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("delete request failed: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode == 404 {
		// 檔案不存在，視為成功
		return nil
	}
	
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("delete failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	return nil
}

// ScanFolder Supabase Storage 不支援本地資料夾掃描
func (s *supabaseStorageAdapter) ScanFolder(ctx context.Context, folderPath string) ([]models.MediaFile, error) {
	// Supabase Storage 是雲端服務，不支援掃描本地資料夾
	// 這個方法可以用來列出 bucket 中的檔案
	return s.listBucketFiles(ctx, folderPath)
}

// listBucketFiles 列出 bucket 中的檔案
func (s *supabaseStorageAdapter) listBucketFiles(ctx context.Context, prefix string) ([]models.MediaFile, error) {
	listURL := fmt.Sprintf("%s/storage/v1/object/list/%s", s.url, s.bucket)
	
	requestBody := map[string]interface{}{
		"limit":  1000,
		"offset": 0,
	}
	
	if prefix != "" {
		requestBody["prefix"] = prefix
	}
	
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	
	req, err := http.NewRequestWithContext(ctx, "POST", listURL, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create list request: %w", err)
	}
	
	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("list request failed: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("list failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	var files []struct {
		Name         string    `json:"name"`
		Size         int64     `json:"size"`
		LastModified time.Time `json:"updated_at"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&files); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	
	var mediaFiles []models.MediaFile
	for _, file := range files {
		if models.IsImageFile(file.Name) {
			mediaFiles = append(mediaFiles, models.MediaFile{
				Path:        file.Name,
				Filename:    filepath.Base(file.Name),
				Size:        file.Size,
				ModifiedAt:  file.LastModified,
				ContentType: models.GetImageContentType(file.Name),
			})
		}
	}
	
	return mediaFiles, nil
}

// GetStorageType 取得儲存類型
func (s *supabaseStorageAdapter) GetStorageType() models.StorageType {
	return models.StorageTypeSupabase
}

// HealthCheck 健康檢查
func (s *supabaseStorageAdapter) HealthCheck(ctx context.Context) error {
	// 檢查 bucket 是否存在且可存取
	listURL := fmt.Sprintf("%s/storage/v1/object/list/%s", s.url, s.bucket)
	
	requestBody := map[string]interface{}{
		"limit":  1,
		"offset": 0,
	}
	
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("failed to marshal health check request: %w", err)
	}
	
	req, err := http.NewRequestWithContext(ctx, "POST", listURL, bytes.NewReader(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}
	
	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("health check request failed: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("health check failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	return nil
}