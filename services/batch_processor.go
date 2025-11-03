package services

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"semantic-text-processor/models"
)

// BatchProcessor 批次處理協調器
type BatchProcessor struct {
	mediaProcessor   MediaProcessor
	folderScanner    *FolderScanner
	maxConcurrency   int
	defaultTimeout   time.Duration
	activeBatches    map[string]*BatchJob
	batchesMutex     sync.RWMutex
	jobCounter       int64
}

// NewBatchProcessor 建立新的批次處理協調器
func NewBatchProcessor(mediaProcessor MediaProcessor, folderScanner *FolderScanner) *BatchProcessor {
	return &BatchProcessor{
		mediaProcessor:  mediaProcessor,
		folderScanner:   folderScanner,
		maxConcurrency:  5,
		defaultTimeout:  30 * time.Minute,
		activeBatches:   make(map[string]*BatchJob),
	}
}

// BatchJob 批次處理任務
type BatchJob struct {
	ID              string
	Status          *models.BatchProcessStatus
	Request         *models.BatchProcessRequest
	Context         context.Context
	CancelFunc      context.CancelFunc
	Results         []*models.ProcessImageResult
	Errors          []models.BatchError
	ProgressChan    chan BatchProgress
	CompletionChan  chan *models.BatchProcessResult
	mutex           sync.RWMutex
	pauseChan       chan struct{}
	resumeChan      chan struct{}
	isPaused        bool
}

// BatchProgress 批次處理進度
type BatchProgress struct {
	BatchID         string
	ProcessedFiles  int
	TotalFiles      int
	CurrentFile     string
	Status          string
	LastUpdate      time.Time
	EstimatedTime   time.Duration
}

// StartBatchProcess 開始批次處理
func (b *BatchProcessor) StartBatchProcess(ctx context.Context, req *models.BatchProcessRequest) (*BatchJob, error) {
	// 生成批次 ID
	batchID := b.generateBatchID()
	
	// 建立批次狀態
	status := &models.BatchProcessStatus{
		BatchID:        batchID,
		TotalFiles:     len(req.Files),
		ProcessedFiles: 0,
		FailedFiles:    0,
		Status:         "starting",
		StartedAt:      time.Now(),
		Errors:         make([]models.BatchError, 0),
	}
	
	// 建立可取消的 context
	jobCtx, cancelFunc := context.WithCancel(ctx)
	
	// 建立批次任務
	job := &BatchJob{
		ID:             batchID,
		Status:         status,
		Request:        req,
		Context:        jobCtx,
		CancelFunc:     cancelFunc,
		Results:        make([]*models.ProcessImageResult, 0),
		Errors:         make([]models.BatchError, 0),
		ProgressChan:   make(chan BatchProgress, 100),
		CompletionChan: make(chan *models.BatchProcessResult, 1),
		pauseChan:      make(chan struct{}),
		resumeChan:     make(chan struct{}),
		isPaused:       false,
	}
	
	// 註冊批次任務
	b.batchesMutex.Lock()
	b.activeBatches[batchID] = job
	b.batchesMutex.Unlock()
	
	// 啟動處理 goroutine
	go b.processBatch(job)
	
	return job, nil
}

// GetBatchStatus 取得批次狀態
func (b *BatchProcessor) GetBatchStatus(batchID string) (*models.BatchProcessStatus, error) {
	b.batchesMutex.RLock()
	job, exists := b.activeBatches[batchID]
	b.batchesMutex.RUnlock()
	
	if !exists {
		return nil, fmt.Errorf("batch not found: %s", batchID)
	}
	
	job.mutex.RLock()
	defer job.mutex.RUnlock()
	
	// 複製狀態以避免併發修改
	statusCopy := *job.Status
	statusCopy.Errors = make([]models.BatchError, len(job.Errors))
	copy(statusCopy.Errors, job.Errors)
	
	return &statusCopy, nil
}

// PauseBatch 暫停批次處理
func (b *BatchProcessor) PauseBatch(batchID string) error {
	b.batchesMutex.RLock()
	job, exists := b.activeBatches[batchID]
	b.batchesMutex.RUnlock()
	
	if !exists {
		return fmt.Errorf("batch not found: %s", batchID)
	}
	
	job.mutex.Lock()
	defer job.mutex.Unlock()
	
	if job.Status.Status != "processing" {
		return fmt.Errorf("batch is not in processing state: %s", job.Status.Status)
	}
	
	if !job.isPaused {
		job.isPaused = true
		job.Status.Status = "paused"
		close(job.pauseChan)
		job.pauseChan = make(chan struct{})
	}
	
	return nil
}

// ResumeBatch 恢復批次處理
func (b *BatchProcessor) ResumeBatch(batchID string) error {
	b.batchesMutex.RLock()
	job, exists := b.activeBatches[batchID]
	b.batchesMutex.RUnlock()
	
	if !exists {
		return fmt.Errorf("batch not found: %s", batchID)
	}
	
	job.mutex.Lock()
	defer job.mutex.Unlock()
	
	if job.Status.Status != "paused" {
		return fmt.Errorf("batch is not paused: %s", job.Status.Status)
	}
	
	if job.isPaused {
		job.isPaused = false
		job.Status.Status = "processing"
		close(job.resumeChan)
		job.resumeChan = make(chan struct{})
	}
	
	return nil
}

// CancelBatch 取消批次處理
func (b *BatchProcessor) CancelBatch(batchID string) error {
	b.batchesMutex.RLock()
	job, exists := b.activeBatches[batchID]
	b.batchesMutex.RUnlock()
	
	if !exists {
		return fmt.Errorf("batch not found: %s", batchID)
	}
	
	job.mutex.Lock()
	defer job.mutex.Unlock()
	
	if job.Status.Status == "completed" || job.Status.Status == "cancelled" {
		return fmt.Errorf("batch already finished: %s", job.Status.Status)
	}
	
	job.Status.Status = "cancelling"
	job.CancelFunc()
	
	return nil
}

// GetActiveBatches 取得所有活躍的批次
func (b *BatchProcessor) GetActiveBatches() map[string]*models.BatchProcessStatus {
	b.batchesMutex.RLock()
	defer b.batchesMutex.RUnlock()
	
	result := make(map[string]*models.BatchProcessStatus)
	for id, job := range b.activeBatches {
		job.mutex.RLock()
		statusCopy := *job.Status
		job.mutex.RUnlock()
		result[id] = &statusCopy
	}
	
	return result
}

// GetProgressChannel 取得進度通道
func (b *BatchProcessor) GetProgressChannel(batchID string) (<-chan BatchProgress, error) {
	b.batchesMutex.RLock()
	job, exists := b.activeBatches[batchID]
	b.batchesMutex.RUnlock()
	
	if !exists {
		return nil, fmt.Errorf("batch not found: %s", batchID)
	}
	
	return job.ProgressChan, nil
}

// WaitForCompletion 等待批次完成
func (b *BatchProcessor) WaitForCompletion(batchID string, timeout time.Duration) (*models.BatchProcessResult, error) {
	b.batchesMutex.RLock()
	job, exists := b.activeBatches[batchID]
	b.batchesMutex.RUnlock()
	
	if !exists {
		return nil, fmt.Errorf("batch not found: %s", batchID)
	}
	
	select {
	case result := <-job.CompletionChan:
		return result, nil
	case <-time.After(timeout):
		return nil, fmt.Errorf("batch completion timeout")
	case <-job.Context.Done():
		return nil, job.Context.Err()
	}
}

// CleanupCompletedBatches 清理已完成的批次
func (b *BatchProcessor) CleanupCompletedBatches(olderThan time.Duration) int {
	b.batchesMutex.Lock()
	defer b.batchesMutex.Unlock()
	
	cutoff := time.Now().Add(-olderThan)
	cleaned := 0
	
	for id, job := range b.activeBatches {
		job.mutex.RLock()
		isCompleted := job.Status.Status == "completed" || job.Status.Status == "cancelled" || job.Status.Status == "failed"
		completedTime := job.Status.CompletedAt
		job.mutex.RUnlock()
		
		if isCompleted && completedTime != nil && completedTime.Before(cutoff) {
			delete(b.activeBatches, id)
			cleaned++
		}
	}
	
	return cleaned
}

// 私有方法

// processBatch 處理批次任務
func (b *BatchProcessor) processBatch(job *BatchJob) {
	defer func() {
		// 清理資源
		close(job.ProgressChan)
		
		// 從活躍批次中移除（延遲移除，讓客戶端有時間取得最終狀態）
		go func() {
			time.Sleep(5 * time.Minute)
			b.batchesMutex.Lock()
			delete(b.activeBatches, job.ID)
			b.batchesMutex.Unlock()
		}()
	}()
	
	// 更新狀態為處理中
	job.mutex.Lock()
	job.Status.Status = "processing"
	job.mutex.Unlock()
	
	// 建立工作池
	concurrency := job.Request.Concurrency
	if concurrency <= 0 {
		concurrency = b.maxConcurrency
	}
	
	semaphore := make(chan struct{}, concurrency)
	resultChan := make(chan *models.ProcessImageResult, len(job.Request.Files))
	errorChan := make(chan models.BatchError, len(job.Request.Files))
	
	startTime := time.Now()
	
	// 處理每個檔案
	for i, filePath := range job.Request.Files {
		select {
		case <-job.Context.Done():
			// 批次被取消
			b.finalizeBatch(job, "cancelled", nil)
			return
		default:
		}

		// 檢查是否暫停
		if job.isPaused {
			select {
			case <-job.resumeChan:
				// 恢復處理
			case <-job.Context.Done():
				b.finalizeBatch(job, "cancelled", nil)
				return
			}
		}

		// 建立 MediaFile 結構
		mediaFile := models.MediaFile{
			Path:     filePath,
			Filename: filepath.Base(filePath),
		}

		// 更新進度
		progress := BatchProgress{
			BatchID:        job.ID,
			ProcessedFiles: i,
			TotalFiles:     len(job.Request.Files),
			CurrentFile:    mediaFile.Filename,
			Status:         "processing",
			LastUpdate:     time.Now(),
		}

		// 估算剩餘時間
		if i > 0 {
			elapsed := time.Since(startTime)
			avgTimePerFile := elapsed / time.Duration(i)
			remaining := time.Duration(len(job.Request.Files)-i) * avgTimePerFile
			progress.EstimatedTime = remaining
		}

		select {
		case job.ProgressChan <- progress:
		default:
			// 進度通道滿了，跳過
		}

		// 啟動處理 goroutine
		go func(index int, file models.MediaFile) {
			// 取得信號量
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// 處理單個檔案
			result, err := b.processSingleFile(job.Context, file, job.Request)
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
	for i := 0; i < len(job.Request.Files); i++ {
		select {
		case result := <-resultChan:
			job.mutex.Lock()
			job.Results = append(job.Results, result)
			job.Status.ProcessedFiles++
			job.mutex.Unlock()
			
		case err := <-errorChan:
			job.mutex.Lock()
			job.Errors = append(job.Errors, err)
			job.Status.FailedFiles++
			job.Status.Errors = append(job.Status.Errors, err)
			job.mutex.Unlock()
			
		case <-job.Context.Done():
			b.finalizeBatch(job, "cancelled", job.Context.Err())
			return
		}
	}
	
	// 完成處理
	b.finalizeBatch(job, "completed", nil)
}

// processSingleFile 處理單個檔案
func (b *BatchProcessor) processSingleFile(ctx context.Context, mediaFile models.MediaFile, req *models.BatchProcessRequest) (*models.ProcessImageResult, error) {
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
	return b.mediaProcessor.ProcessImage(ctx, processReq)
}

// finalizeBatch 完成批次處理
func (b *BatchProcessor) finalizeBatch(job *BatchJob, status string, err error) {
	job.mutex.Lock()
	defer job.mutex.Unlock()
	
	job.Status.Status = status
	completedAt := time.Now()
	job.Status.CompletedAt = &completedAt

	// 轉換 Results 從 []*ProcessImageResult 到 []ProcessImageResult
	results := make([]models.ProcessImageResult, len(job.Results))
	for i, r := range job.Results {
		if r != nil {
			results[i] = *r
		}
	}

	// 建立最終結果
	result := &models.BatchProcessResult{
		BatchID: job.ID,
		Status:  *job.Status,
		Results: results,
	}
	
	// 發送完成通知
	select {
	case job.CompletionChan <- result:
	default:
		// 通道已滿或已關閉
	}
	
	// 發送最終進度
	finalProgress := BatchProgress{
		BatchID:        job.ID,
		ProcessedFiles: job.Status.ProcessedFiles,
		TotalFiles:     job.Status.TotalFiles,
		Status:         status,
		LastUpdate:     time.Now(),
	}
	
	select {
	case job.ProgressChan <- finalProgress:
	default:
	}
}

// generateBatchID 生成批次 ID
func (b *BatchProcessor) generateBatchID() string {
	counter := atomic.AddInt64(&b.jobCounter, 1)
	return fmt.Sprintf("batch_%d_%d", time.Now().UnixNano(), counter)
}

// SetMaxConcurrency 設定最大並行數
func (b *BatchProcessor) SetMaxConcurrency(concurrency int) {
	if concurrency > 0 {
		b.maxConcurrency = concurrency
	}
}

// SetDefaultTimeout 設定預設超時時間
func (b *BatchProcessor) SetDefaultTimeout(timeout time.Duration) {
	if timeout > 0 {
		b.defaultTimeout = timeout
	}
}

// GetStats 取得批次處理統計資訊
func (b *BatchProcessor) GetStats() map[string]interface{} {
	b.batchesMutex.RLock()
	defer b.batchesMutex.RUnlock()
	
	stats := map[string]interface{}{
		"active_batches":   len(b.activeBatches),
		"max_concurrency": b.maxConcurrency,
		"default_timeout": b.defaultTimeout.String(),
	}
	
	// 統計各狀態的批次數量
	statusCounts := make(map[string]int)
	for _, job := range b.activeBatches {
		job.mutex.RLock()
		status := job.Status.Status
		job.mutex.RUnlock()
		statusCounts[status]++
	}
	
	stats["status_counts"] = statusCounts
	
	return stats
}