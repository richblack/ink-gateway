package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"semantic-text-processor/models"
)

// FolderScanner 資料夾掃描服務
type FolderScanner struct {
	supportedFormats map[string]bool
	maxDepth         int
	maxFileSize      int64
}

// NewFolderScanner 建立新的資料夾掃描服務
func NewFolderScanner() *FolderScanner {
	return &FolderScanner{
		supportedFormats: map[string]bool{
			".png":  true,
			".jpg":  true,
			".jpeg": true,
			".gif":  true,
			".webp": true,
			".bmp":  true,
			".tiff": true,
			".tif":  true,
		},
		maxDepth:    10,  // 預設最大深度
		maxFileSize: 50 * 1024 * 1024, // 50MB
	}
}

// NewFolderScannerWithConfig 使用配置建立資料夾掃描服務
func NewFolderScannerWithConfig(config *FolderScannerConfig) *FolderScanner {
	scanner := NewFolderScanner()
	
	if config.MaxDepth > 0 {
		scanner.maxDepth = config.MaxDepth
	}
	
	if config.MaxFileSize > 0 {
		scanner.maxFileSize = config.MaxFileSize
	}
	
	if len(config.SupportedFormats) > 0 {
		scanner.supportedFormats = make(map[string]bool)
		for _, format := range config.SupportedFormats {
			scanner.supportedFormats[strings.ToLower(format)] = true
		}
	}
	
	return scanner
}

// FolderScannerConfig 資料夾掃描配置
type FolderScannerConfig struct {
	MaxDepth         int      `json:"max_depth"`
	MaxFileSize      int64    `json:"max_file_size"`
	SupportedFormats []string `json:"supported_formats"`
}

// ScanFolder 掃描資料夾中的圖片檔案
func (f *FolderScanner) ScanFolder(ctx context.Context, folderPath string) ([]models.MediaFile, error) {
	// 檢查資料夾是否存在
	if _, err := os.Stat(folderPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("folder does not exist: %s", folderPath)
	}
	
	var files []models.MediaFile
	
	err := f.scanRecursive(ctx, folderPath, 0, &files)
	if err != nil {
		return nil, fmt.Errorf("failed to scan folder %s: %w", folderPath, err)
	}
	
	return files, nil
}

// ScanFolderWithFilter 使用過濾器掃描資料夾
func (f *FolderScanner) ScanFolderWithFilter(ctx context.Context, folderPath string, filter *ScanFilter) ([]models.MediaFile, error) {
	files, err := f.ScanFolder(ctx, folderPath)
	if err != nil {
		return nil, err
	}
	
	if filter == nil {
		return files, nil
	}
	
	return f.applyFilter(files, filter), nil
}

// ScanFilter 掃描過濾器
type ScanFilter struct {
	MinSize      int64     `json:"min_size"`
	MaxSize      int64     `json:"max_size"`
	Formats      []string  `json:"formats"`
	ModifiedAfter *time.Time `json:"modified_after"`
	ModifiedBefore *time.Time `json:"modified_before"`
	NamePattern  string    `json:"name_pattern"`
}

// GetScanStats 取得掃描統計資訊
func (f *FolderScanner) GetScanStats(ctx context.Context, folderPath string) (*ScanStats, error) {
	files, err := f.ScanFolder(ctx, folderPath)
	if err != nil {
		return nil, err
	}
	
	stats := &ScanStats{
		TotalFiles:    len(files),
		TotalSize:     0,
		FormatCounts:  make(map[string]int),
		LargestFile:   "",
		LargestSize:   0,
		SmallestFile:  "",
		SmallestSize:  int64(^uint64(0) >> 1), // Max int64
		ScanTime:      time.Now(),
	}
	
	for _, file := range files {
		stats.TotalSize += file.Size
		
		// 統計格式
		ext := strings.ToLower(filepath.Ext(file.Filename))
		stats.FormatCounts[ext]++
		
		// 找最大檔案
		if file.Size > stats.LargestSize {
			stats.LargestSize = file.Size
			stats.LargestFile = file.Filename
		}
		
		// 找最小檔案
		if file.Size < stats.SmallestSize {
			stats.SmallestSize = file.Size
			stats.SmallestFile = file.Filename
		}
	}
	
	// 如果沒有檔案，重置最小檔案大小
	if len(files) == 0 {
		stats.SmallestSize = 0
	}
	
	return stats, nil
}

// ScanStats 掃描統計資訊
type ScanStats struct {
	TotalFiles    int            `json:"total_files"`
	TotalSize     int64          `json:"total_size"`
	FormatCounts  map[string]int `json:"format_counts"`
	LargestFile   string         `json:"largest_file"`
	LargestSize   int64          `json:"largest_size"`
	SmallestFile  string         `json:"smallest_file"`
	SmallestSize  int64          `json:"smallest_size"`
	ScanTime      time.Time      `json:"scan_time"`
}

// ValidateFolder 驗證資料夾是否可掃描
func (f *FolderScanner) ValidateFolder(folderPath string) error {
	// 檢查路徑是否存在
	info, err := os.Stat(folderPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("folder does not exist: %s", folderPath)
	}
	if err != nil {
		return fmt.Errorf("failed to access folder %s: %w", folderPath, err)
	}
	
	// 檢查是否為目錄
	if !info.IsDir() {
		return fmt.Errorf("path is not a directory: %s", folderPath)
	}
	
	// 檢查讀取權限
	file, err := os.Open(folderPath)
	if err != nil {
		return fmt.Errorf("no read permission for folder %s: %w", folderPath, err)
	}
	file.Close()
	
	return nil
}

// 私有方法

// scanRecursive 遞迴掃描資料夾
func (f *FolderScanner) scanRecursive(ctx context.Context, folderPath string, depth int, files *[]models.MediaFile) error {
	// 檢查深度限制
	if depth > f.maxDepth {
		return nil
	}
	
	// 檢查 context 是否被取消
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	
	// 讀取目錄內容
	entries, err := os.ReadDir(folderPath)
	if err != nil {
		return fmt.Errorf("failed to read directory %s: %w", folderPath, err)
	}
	
	for _, entry := range entries {
		fullPath := filepath.Join(folderPath, entry.Name())
		
		if entry.IsDir() {
			// 遞迴掃描子目錄
			if err := f.scanRecursive(ctx, fullPath, depth+1, files); err != nil {
				return err
			}
		} else {
			// 檢查是否為支援的圖片格式
			if f.isImageFile(entry.Name()) {
				mediaFile, err := f.createMediaFile(fullPath, entry)
				if err != nil {
					// 記錄錯誤但繼續處理其他檔案
					continue
				}
				
				// 檢查檔案大小限制
				if mediaFile.Size <= f.maxFileSize {
					*files = append(*files, *mediaFile)
				}
			}
		}
	}
	
	return nil
}

// isImageFile 檢查是否為圖片檔案
func (f *FolderScanner) isImageFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return f.supportedFormats[ext]
}

// createMediaFile 建立 MediaFile 物件
func (f *FolderScanner) createMediaFile(fullPath string, entry os.DirEntry) (*models.MediaFile, error) {
	info, err := entry.Info()
	if err != nil {
		return nil, fmt.Errorf("failed to get file info for %s: %w", fullPath, err)
	}
	
	return &models.MediaFile{
		Path:        fullPath,
		Filename:    entry.Name(),
		Size:        info.Size(),
		ModifiedAt:  info.ModTime(),
		ContentType: models.GetImageContentType(entry.Name()),
	}, nil
}

// applyFilter 套用過濾器
func (f *FolderScanner) applyFilter(files []models.MediaFile, filter *ScanFilter) []models.MediaFile {
	var filtered []models.MediaFile
	
	for _, file := range files {
		if f.matchesFilter(file, filter) {
			filtered = append(filtered, file)
		}
	}
	
	return filtered
}

// matchesFilter 檢查檔案是否符合過濾條件
func (f *FolderScanner) matchesFilter(file models.MediaFile, filter *ScanFilter) bool {
	// 檢查檔案大小
	if filter.MinSize > 0 && file.Size < filter.MinSize {
		return false
	}
	
	if filter.MaxSize > 0 && file.Size > filter.MaxSize {
		return false
	}
	
	// 檢查格式
	if len(filter.Formats) > 0 {
		ext := strings.ToLower(filepath.Ext(file.Filename))
		found := false
		for _, format := range filter.Formats {
			if strings.ToLower(format) == ext {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	
	// 檢查修改時間
	if filter.ModifiedAfter != nil && file.ModifiedAt.Before(*filter.ModifiedAfter) {
		return false
	}
	
	if filter.ModifiedBefore != nil && file.ModifiedAt.After(*filter.ModifiedBefore) {
		return false
	}
	
	// 檢查檔名模式
	if filter.NamePattern != "" {
		matched, err := filepath.Match(filter.NamePattern, file.Filename)
		if err != nil || !matched {
			return false
		}
	}
	
	return true
}

// GetSupportedFormats 取得支援的格式列表
func (f *FolderScanner) GetSupportedFormats() []string {
	formats := make([]string, 0, len(f.supportedFormats))
	for format := range f.supportedFormats {
		formats = append(formats, format)
	}
	return formats
}

// SetMaxDepth 設定最大掃描深度
func (f *FolderScanner) SetMaxDepth(depth int) {
	if depth > 0 {
		f.maxDepth = depth
	}
}

// SetMaxFileSize 設定最大檔案大小
func (f *FolderScanner) SetMaxFileSize(size int64) {
	if size > 0 {
		f.maxFileSize = size
	}
}

// AddSupportedFormat 新增支援的格式
func (f *FolderScanner) AddSupportedFormat(format string) {
	f.supportedFormats[strings.ToLower(format)] = true
}

// RemoveSupportedFormat 移除支援的格式
func (f *FolderScanner) RemoveSupportedFormat(format string) {
	delete(f.supportedFormats, strings.ToLower(format))
}