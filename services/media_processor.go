package services

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"semantic-text-processor/models"
)

// mediaProcessor MediaProcessor 核心服務實作
type mediaProcessor struct {
	storageService    *StorageService
	visionService     VisionAIService
	embeddingService  ImageEmbeddingService
	hashService       *HashService
	metadataService   *ImageMetadataService
	chunkService      UnifiedChunkService
}

// NewMediaProcessor 建立新的 MediaProcessor 服務
func NewMediaProcessor(
	storageService *StorageService,
	visionService VisionAIService,
	embeddingService ImageEmbeddingService,
	chunkService UnifiedChunkService,
) MediaProcessor {
	return &mediaProcessor{
		storageService:   storageService,
		visionService:    visionService,
		embeddingService: embeddingService,
		hashService:      NewHashService(),
		metadataService:  NewImageMetadataService(),
		chunkService:     chunkService,
	}
}

// ProcessImage 處理單張圖片（上傳、分析、向量化、儲存）
func (m *mediaProcessor) ProcessImage(ctx context.Context, req *models.ProcessImageRequest) (*models.ProcessImageResult, error) {
	// 1. 計算檔案雜湊
	hashReader := NewHashAndSizeReader(bytes.NewReader(req.File))
	hash, size := hashReader.GetMetadata()
	
	// 2. 檢查是否已存在相同雜湊的圖片
	existingChunk, err := m.checkDuplicateImage(ctx, hash)
	if err != nil {
		return nil, fmt.Errorf("failed to check duplicate image: %w", err)
	}
	
	if existingChunk != nil {
		// 圖片已存在，返回現有資訊
		return m.buildResultFromExistingChunk(existingChunk)
	}
	
	// 3. 提取圖片元資料
	metadata, err := m.metadataService.ExtractMetadata(hashReader, req.OriginalFilename)
	if err != nil {
		return nil, fmt.Errorf("failed to extract metadata: %w", err)
	}
	
	// 更新元資料
	metadata.Hash = hash
	metadata.Size = size
	
	// 4. 驗證檔案
	if err := m.storageService.ValidateFile(metadata); err != nil {
		return nil, fmt.Errorf("file validation failed: %w", err)
	}
	
	// 5. 上傳到儲存服務
	storageResult, err := m.storageService.Upload(ctx, hashReader, metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to upload image: %w", err)
	}
	
	// 6. 建立 UnifiedChunk 記錄
	chunk, err := m.createImageChunk(ctx, req, storageResult, metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to create chunk: %w", err)
	}
	
	result := &models.ProcessImageResult{
		ChunkID:      chunk.ChunkID,
		StorageID:    storageResult.StorageID,
		URL:          storageResult.URL,
		Hash:         hash,
		EmbeddingIDs: make(map[string]string),
	}
	
	// 7. AI 分析（如果啟用）
	if req.AutoAnalyze {
		analysis, err := m.analyzeImage(ctx, storageResult.URL)
		if err != nil {
			// AI 分析失敗不影響整體流程，記錄錯誤
			// TODO: 記錄警告日誌
		} else {
			result.Analysis = analysis
			
			// 更新 chunk 的 AI 分析結果
			if err := m.updateChunkWithAnalysis(ctx, chunk.ChunkID, analysis); err != nil {
				// 更新失敗不影響整體流程
			}
		}
	}
	
	// 8. 生成向量（如果啟用）
	if req.AutoEmbed {
		embeddingIDs, err := m.generateEmbeddings(ctx, chunk.ChunkID, storageResult.URL, result.Analysis)
		if err != nil {
			// 向量生成失敗不影響整體流程，記錄錯誤
			// TODO: 記錄警告日誌
		} else {
			result.EmbeddingIDs = embeddingIDs
		}
	}
	
	return result, nil
}

// BatchProcessImages 批次處理圖片
func (m *mediaProcessor) BatchProcessImages(ctx context.Context, req *models.BatchProcessRequest) (*models.BatchProcessResult, error) {
	// 建立批次處理狀態
	batchID := generateBatchID()
	status := &models.BatchProcessStatus{
		BatchID:        batchID,
		TotalFiles:     len(req.Files),
		ProcessedFiles: 0,
		FailedFiles:    0,
		Status:         "processing",
		StartedAt:      time.Now(),
		Errors:         make([]models.BatchError, 0),
	}
	
	// 建立結果通道
	resultChan := make(chan *models.ProcessImageResult, len(req.Files))
	errorChan := make(chan models.BatchError, len(req.Files))
	
	// 建立工作池
	concurrency := req.Concurrency
	if concurrency <= 0 {
		concurrency = 3 // 預設並行數
	}
	
	semaphore := make(chan struct{}, concurrency)

	// 處理每個檔案
	for i, filePath := range req.Files {
		// 建立 MediaFile 結構
		mediaFile := models.MediaFile{
			Path:     filePath,
			Filename: filepath.Base(filePath),
		}

		go func(index int, file models.MediaFile) {
			// 取得信號量
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// 處理單個檔案
			result, err := m.processSingleFile(ctx, file, req)
			if err != nil {
				errorChan <- models.BatchError{
					Filename:  file.Filename,
					Error:     err.Error(),
					Timestamp: time.Now(),
				}
			} else {
				resultChan <- result
			}
		}(i, mediaFile)
	}
	
	// 收集結果
	var results []*models.ProcessImageResult
	var errors []models.BatchError
	
	for i := 0; i < len(req.Files); i++ {
		select {
		case result := <-resultChan:
			results = append(results, result)
			status.ProcessedFiles++
		case err := <-errorChan:
			errors = append(errors, err)
			status.FailedFiles++
		case <-ctx.Done():
			status.Status = "cancelled"

			// 轉換 Results
			convertedResults := make([]models.ProcessImageResult, len(results))
			for i, r := range results {
				if r != nil {
					convertedResults[i] = *r
				}
			}

			return &models.BatchProcessResult{
				BatchID: batchID,
				Status:  *status,
				Results: convertedResults,
			}, ctx.Err()
		}
	}

	// 更新最終狀態
	status.Status = "completed"
	status.CompletedAt = &time.Time{}
	*status.CompletedAt = time.Now()
	status.Errors = errors

	// 轉換 Results
	convertedResults := make([]models.ProcessImageResult, len(results))
	for i, r := range results {
		if r != nil {
			convertedResults[i] = *r
		}
	}

	return &models.BatchProcessResult{
		BatchID: batchID,
		Status:  *status,
		Results: convertedResults,
	}, nil
}

// AnalyzeImage 分析圖片內容
func (m *mediaProcessor) AnalyzeImage(ctx context.Context, imageURL string) (*models.ImageAnalysis, error) {
	return m.analyzeImage(ctx, imageURL)
}

// GenerateImageEmbedding 生成圖片向量
func (m *mediaProcessor) GenerateImageEmbedding(ctx context.Context, imageURL string) ([]float64, error) {
	return m.embeddingService.GenerateEmbedding(ctx, imageURL)
}

// CalculateHash 計算檔案雜湊
func (m *mediaProcessor) CalculateHash(ctx context.Context, file io.Reader) (string, error) {
	return m.hashService.CalculateHash(file)
}

// 私有方法

// checkDuplicateImage 檢查重複圖片
func (m *mediaProcessor) checkDuplicateImage(ctx context.Context, hash string) (*models.UnifiedChunkRecord, error) {
	// 根據雜湊查詢現有的圖片 chunk
	// 這需要在 ChunkService 中實作 FindByHash 方法
	// 暫時返回 nil，表示沒有重複
	return nil, nil
}

// buildResultFromExistingChunk 從現有 chunk 建立結果
func (m *mediaProcessor) buildResultFromExistingChunk(chunk *models.UnifiedChunkRecord) (*models.ProcessImageResult, error) {
	// 從 chunk metadata 中提取儲存資訊
	storageInfo, err := models.ExtractStorageInfo(chunk.Metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to extract storage info: %w", err)
	}
	
	// 從 chunk metadata 中提取 AI 分析結果
	analysis, _ := models.ExtractAIAnalysis(chunk.Metadata)
	
	// 查詢相關的向量 ID
	embeddingIDs := make(map[string]string)
	// TODO: 實作向量 ID 查詢邏輯
	
	return &models.ProcessImageResult{
		ChunkID:      chunk.ChunkID,
		StorageID:    storageInfo.StorageID,
		URL:          storageInfo.URL,
		Hash:         storageInfo.FileHash,
		Analysis:     analysis,
		EmbeddingIDs: embeddingIDs,
	}, nil
}

// createImageChunk 建立圖片 chunk
func (m *mediaProcessor) createImageChunk(ctx context.Context, req *models.ProcessImageRequest, storageResult *models.StorageResult, metadata *models.MediaMetadata) (*models.UnifiedChunkRecord, error) {
	// 建立圖片 metadata
	imageMetadata := models.CreateImageMetadata(storageResult, metadata)

	// 建立 UnifiedChunkRecord
	chunk := &models.UnifiedChunkRecord{
		Contents: fmt.Sprintf("Image: %s", metadata.OriginalFilename),
		Page:     req.PageID,
		Tags:     req.Tags,
		Metadata: imageMetadata,
	}
	
	// 建立 chunk
	err := m.chunkService.CreateChunk(ctx, chunk)
	if err != nil {
		return nil, fmt.Errorf("failed to create chunk: %w", err)
	}
	
	return chunk, nil
}

// analyzeImage 分析圖片
func (m *mediaProcessor) analyzeImage(ctx context.Context, imageURL string) (*models.ImageAnalysis, error) {
	options := &models.AnalysisOptions{
		DetailLevel: "medium",
		Language:    "zh-TW",
		MaxTokens:   1000,
	}
	
	return m.visionService.AnalyzeImage(ctx, imageURL, options)
}

// updateChunkWithAnalysis 更新 chunk 的 AI 分析結果
func (m *mediaProcessor) updateChunkWithAnalysis(ctx context.Context, chunkID string, analysis *models.ImageAnalysis) error {
	// 取得現有 chunk
	chunk, err := m.chunkService.GetChunk(ctx, chunkID)
	if err != nil {
		return fmt.Errorf("failed to get chunk: %w", err)
	}
	
	// 更新 metadata 中的 AI 分析結果
	updatedMetadata := models.UpdateAIAnalysis(chunk.Metadata, analysis)
	
	// 更新 chunk 內容和 metadata
	chunk.Contents = analysis.Description // 使用 AI 描述作為內容
	chunk.Metadata = updatedMetadata
	
	err = m.chunkService.UpdateChunk(ctx, chunk)
	return err
}

// generateEmbeddings 生成向量
func (m *mediaProcessor) generateEmbeddings(ctx context.Context, chunkID, imageURL string, analysis *models.ImageAnalysis) (map[string]string, error) {
	embeddingIDs := make(map[string]string)
	
	// 1. 生成圖片向量
	imageEmbedding, err := m.embeddingService.GenerateEmbedding(ctx, imageURL)
	if err != nil {
		return nil, fmt.Errorf("failed to generate image embedding: %w", err)
	}
	
	// 儲存圖片向量到 chunk 中
	// 由於 UnifiedChunkRecord 已經包含向量欄位，我們直接更新 chunk
	chunk, err := m.chunkService.GetChunk(ctx, chunkID)
	if err != nil {
		return nil, fmt.Errorf("failed to get chunk for embedding: %w", err)
	}
	
	// 更新 chunk 的向量資訊
	vectorType := "image"
	vectorModel := "clip-vit-b-32"
	chunk.Vector = imageEmbedding
	chunk.VectorType = &vectorType
	chunk.VectorModel = &vectorModel
	chunk.VectorMetadata = map[string]interface{}{"source": "image"}
	
	err = m.chunkService.UpdateChunk(ctx, chunk)
	if err != nil {
		return nil, fmt.Errorf("failed to update chunk with embedding: %w", err)
	}
	
	embeddingIDs["image"] = chunkID // 使用 chunk ID 作為 embedding ID
	
	// 2. 生成文字向量（如果有 AI 分析結果）
	if analysis != nil && analysis.Description != "" {
		// 這裡需要使用文字向量化服務
		// 暫時跳過，因為需要整合現有的 EmbeddingService
		// TODO: 整合文字向量化
	}
	
	return embeddingIDs, nil
}

// processSingleFile 處理單個檔案
func (m *mediaProcessor) processSingleFile(ctx context.Context, mediaFile models.MediaFile, req *models.BatchProcessRequest) (*models.ProcessImageResult, error) {
	// 開啟檔案
	file, err := openMediaFile(mediaFile.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// 讀取檔案內容
	fileData, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// 建立處理請求
	processReq := &models.ProcessImageRequest{
		File:             fileData,
		OriginalFilename: mediaFile.Filename,
		PageID:          req.PageID,
		Tags:            req.Tags,
		AutoAnalyze:     req.AutoAnalyze,
		AutoEmbed:       req.AutoEmbed,
		StorageType:     req.StorageType,
	}
	
	// 處理圖片
	return m.ProcessImage(ctx, processReq)
}

// 輔助函數

// generateBatchID 生成批次 ID
func generateBatchID() string {
	return fmt.Sprintf("batch_%d", time.Now().UnixNano())
}

// openMediaFile 開啟媒體檔案
func openMediaFile(path string) (io.ReadCloser, error) {
	// 檢查是否為 URL
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return nil, fmt.Errorf("URL-based media files not supported yet")
	}
	
	// 開啟本地檔案
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open local file %s: %w", path, err)
	}
	
	return file, nil
}