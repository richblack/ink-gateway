package services

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"semantic-text-processor/models"
)

// localStorageAdapter 本地檔案系統儲存適配器
type localStorageAdapter struct {
	basePath string
	baseURL  string
}

// NewLocalStorageAdapter 建立新的本地儲存適配器
func NewLocalStorageAdapter(basePath, baseURL string) MediaStorageAdapter {
	return &localStorageAdapter{
		basePath: basePath,
		baseURL:  baseURL,
	}
}

// Upload 上傳檔案到本地檔案系統
func (l *localStorageAdapter) Upload(ctx context.Context, file io.Reader, metadata *models.MediaMetadata) (*models.StorageResult, error) {
	// 1. 生成唯一檔名 (使用 hash + 時間戳)
	timestamp := time.Now().Unix()
	ext := filepath.Ext(metadata.OriginalFilename)
	storageID := fmt.Sprintf("%s_%d%s", metadata.Hash[:16], timestamp, ext)
	
	// 2. 建立目錄結構 (按日期分組)
	dateDir := time.Now().Format("2006/01/02")
	fullDir := filepath.Join(l.basePath, dateDir)
	
	if err := os.MkdirAll(fullDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory %s: %w", fullDir, err)
	}
	
	// 3. 儲存檔案
	destPath := filepath.Join(fullDir, storageID)
	destFile, err := os.Create(destPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create file %s: %w", destPath, err)
	}
	defer destFile.Close()
	
	_, err = io.Copy(destFile, file)
	if err != nil {
		// 清理失敗的檔案
		os.Remove(destPath)
		return nil, fmt.Errorf("failed to write file %s: %w", destPath, err)
	}
	
	// 4. 返回結果
	relativePath := filepath.Join(dateDir, storageID)
	url := fmt.Sprintf("%s/%s", strings.TrimRight(l.baseURL, "/"), strings.ReplaceAll(relativePath, "\\", "/"))
	
	return &models.StorageResult{
		StorageID:   relativePath,
		URL:         url,
		StorageType: models.StorageTypeLocal,
		UploadedAt:  time.Now(),
	}, nil
}

// GetURL 根據 storage_id 取得存取 URL
func (l *localStorageAdapter) GetURL(ctx context.Context, storageID string) (string, error) {
	// 檢查檔案是否存在
	fullPath := filepath.Join(l.basePath, storageID)
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return "", fmt.Errorf("file not found: %s", storageID)
	}
	
	url := fmt.Sprintf("%s/%s", strings.TrimRight(l.baseURL, "/"), strings.ReplaceAll(storageID, "\\", "/"))
	return url, nil
}

// Download 下載檔案內容
func (l *localStorageAdapter) Download(ctx context.Context, storageID string) (io.ReadCloser, error) {
	fullPath := filepath.Join(l.basePath, storageID)
	
	file, err := os.Open(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file not found: %s", storageID)
		}
		return nil, fmt.Errorf("failed to open file %s: %w", storageID, err)
	}
	
	return file, nil
}

// Delete 刪除檔案
func (l *localStorageAdapter) Delete(ctx context.Context, storageID string) error {
	fullPath := filepath.Join(l.basePath, storageID)
	
	err := os.Remove(fullPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete file %s: %w", storageID, err)
	}
	
	// 嘗試清理空的目錄
	dir := filepath.Dir(fullPath)
	l.cleanupEmptyDirs(dir)
	
	return nil
}

// ScanFolder 掃描資料夾中的圖片檔案
func (l *localStorageAdapter) ScanFolder(ctx context.Context, folderPath string) ([]models.MediaFile, error) {
	var files []models.MediaFile
	
	err := filepath.Walk(folderPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		// 跳過目錄
		if info.IsDir() {
			return nil
		}
		
		// 檢查是否為圖片檔案
		if !models.IsImageFile(info.Name()) {
			return nil
		}
		
		// 檢查 context 是否被取消
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		
		files = append(files, models.MediaFile{
			Path:        path,
			Filename:    info.Name(),
			Size:        info.Size(),
			ModifiedAt:  info.ModTime(),
			ContentType: models.GetImageContentType(info.Name()),
		})
		
		return nil
	})
	
	if err != nil {
		return nil, fmt.Errorf("failed to scan folder %s: %w", folderPath, err)
	}
	
	return files, nil
}

// GetStorageType 取得儲存類型
func (l *localStorageAdapter) GetStorageType() models.StorageType {
	return models.StorageTypeLocal
}

// HealthCheck 健康檢查
func (l *localStorageAdapter) HealthCheck(ctx context.Context) error {
	// 檢查基礎目錄是否存在且可寫
	if err := os.MkdirAll(l.basePath, 0755); err != nil {
		return fmt.Errorf("base path not accessible: %w", err)
	}
	
	// 嘗試建立測試檔案
	testFile := filepath.Join(l.basePath, ".health_check")
	file, err := os.Create(testFile)
	if err != nil {
		return fmt.Errorf("cannot write to base path: %w", err)
	}
	file.Close()
	
	// 清理測試檔案
	os.Remove(testFile)
	
	return nil
}

// cleanupEmptyDirs 清理空的目錄
func (l *localStorageAdapter) cleanupEmptyDirs(dir string) {
	// 不要清理基礎目錄
	if dir == l.basePath {
		return
	}
	
	// 檢查目錄是否為空
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) > 0 {
		return
	}
	
	// 刪除空目錄
	if err := os.Remove(dir); err == nil {
		// 遞迴清理父目錄
		l.cleanupEmptyDirs(filepath.Dir(dir))
	}
}