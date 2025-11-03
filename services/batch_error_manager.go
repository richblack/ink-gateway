package services

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"semantic-text-processor/models"
)

// BatchErrorManager 批次處理錯誤管理器
type BatchErrorManager struct {
	errorHistory    map[string][]models.BatchError
	errorStats      map[string]*ErrorStatistics
	retryPolicies   map[string]*RetryPolicy
	mutex           sync.RWMutex
	maxHistorySize  int
	retryAttempts   map[string]int
	retryMutex      sync.RWMutex
}

// NewBatchErrorManager 建立新的批次錯誤管理器
func NewBatchErrorManager() *BatchErrorManager {
	return &BatchErrorManager{
		errorHistory:   make(map[string][]models.BatchError),
		errorStats:     make(map[string]*ErrorStatistics),
		retryPolicies:  make(map[string]*RetryPolicy),
		maxHistorySize: 1000,
		retryAttempts:  make(map[string]int),
	}
}

// ErrorStatistics 錯誤統計資訊
type ErrorStatistics struct {
	ErrorType       string            `json:"error_type"`
	TotalCount      int               `json:"total_count"`
	FirstOccurrence time.Time         `json:"first_occurrence"`
	LastOccurrence  time.Time         `json:"last_occurrence"`
	FilePatterns    map[string]int    `json:"file_patterns"`
	ErrorMessages   map[string]int    `json:"error_messages"`
	HourlyCount     map[string]int    `json:"hourly_count"`
}

// RetryPolicy 重試策略
type RetryPolicy struct {
	MaxRetries      int           `json:"max_retries"`
	InitialDelay    time.Duration `json:"initial_delay"`
	MaxDelay        time.Duration `json:"max_delay"`
	BackoffFactor   float64       `json:"backoff_factor"`
	RetryableErrors []string      `json:"retryable_errors"`
}

// ErrorReport 錯誤報告
type ErrorReport struct {
	BatchID         string                     `json:"batch_id"`
	TotalErrors     int                        `json:"total_errors"`
	ErrorsByType    map[string]int             `json:"errors_by_type"`
	ErrorsByFile    map[string][]models.BatchError `json:"errors_by_file"`
	TopErrors       []ErrorSummary             `json:"top_errors"`
	Recommendations []string                   `json:"recommendations"`
	GeneratedAt     time.Time                  `json:"generated_at"`
}

// ErrorSummary 錯誤摘要
type ErrorSummary struct {
	ErrorMessage string `json:"error_message"`
	Count        int    `json:"count"`
	Files        []string `json:"files"`
}

// RecordError 記錄錯誤
func (e *BatchErrorManager) RecordError(batchID string, err models.BatchError) {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	
	// 記錄到歷史
	if _, exists := e.errorHistory[batchID]; !exists {
		e.errorHistory[batchID] = make([]models.BatchError, 0)
	}
	
	e.errorHistory[batchID] = append(e.errorHistory[batchID], err)
	
	// 限制歷史大小
	if len(e.errorHistory[batchID]) > e.maxHistorySize {
		e.errorHistory[batchID] = e.errorHistory[batchID][1:]
	}
	
	// 更新統計
	e.updateErrorStatistics(err)
}

// RecordErrors 批次記錄錯誤
func (e *BatchErrorManager) RecordErrors(batchID string, errors []models.BatchError) {
	for _, err := range errors {
		e.RecordError(batchID, err)
	}
}

// GetErrorHistory 取得錯誤歷史
func (e *BatchErrorManager) GetErrorHistory(batchID string) ([]models.BatchError, error) {
	e.mutex.RLock()
	defer e.mutex.RUnlock()
	
	history, exists := e.errorHistory[batchID]
	if !exists {
		return nil, fmt.Errorf("no error history found for batch: %s", batchID)
	}
	
	// 複製切片以避免併發修改
	result := make([]models.BatchError, len(history))
	copy(result, history)
	
	return result, nil
}

// GenerateErrorReport 生成錯誤報告
func (e *BatchErrorManager) GenerateErrorReport(batchID string) (*ErrorReport, error) {
	e.mutex.RLock()
	defer e.mutex.RUnlock()
	
	errors, exists := e.errorHistory[batchID]
	if !exists {
		return nil, fmt.Errorf("no errors found for batch: %s", batchID)
	}
	
	report := &ErrorReport{
		BatchID:         batchID,
		TotalErrors:     len(errors),
		ErrorsByType:    make(map[string]int),
		ErrorsByFile:    make(map[string][]models.BatchError),
		TopErrors:       make([]ErrorSummary, 0),
		Recommendations: make([]string, 0),
		GeneratedAt:     time.Now(),
	}
	
	// 統計錯誤類型
	errorMessages := make(map[string]int)
	for _, err := range errors {
		errorType := e.classifyError(err.Error)
		report.ErrorsByType[errorType]++
		
		// 按檔案分組
		if _, exists := report.ErrorsByFile[err.Filename]; !exists {
			report.ErrorsByFile[err.Filename] = make([]models.BatchError, 0)
		}
		report.ErrorsByFile[err.Filename] = append(report.ErrorsByFile[err.Filename], err)
		
		// 統計錯誤訊息
		errorMessages[err.Error]++
	}
	
	// 生成 Top 錯誤
	report.TopErrors = e.generateTopErrors(errorMessages)
	
	// 生成建議
	report.Recommendations = e.generateRecommendations(report)
	
	return report, nil
}

// ShouldRetry 判斷是否應該重試
func (e *BatchErrorManager) ShouldRetry(filename, errorMsg string) bool {
	e.retryMutex.RLock()
	defer e.retryMutex.RUnlock()
	
	// 檢查重試次數
	key := fmt.Sprintf("%s:%s", filename, errorMsg)
	attempts := e.retryAttempts[key]
	
	// 取得重試策略
	errorType := e.classifyError(errorMsg)
	policy, exists := e.retryPolicies[errorType]
	if !exists {
		policy = e.getDefaultRetryPolicy()
	}
	
	// 檢查是否為可重試錯誤
	if !e.isRetryableError(errorMsg, policy.RetryableErrors) {
		return false
	}
	
	return attempts < policy.MaxRetries
}

// GetRetryDelay 取得重試延遲時間
func (e *BatchErrorManager) GetRetryDelay(filename, errorMsg string) time.Duration {
	e.retryMutex.RLock()
	defer e.retryMutex.RUnlock()
	
	key := fmt.Sprintf("%s:%s", filename, errorMsg)
	attempts := e.retryAttempts[key]
	
	errorType := e.classifyError(errorMsg)
	policy, exists := e.retryPolicies[errorType]
	if !exists {
		policy = e.getDefaultRetryPolicy()
	}
	
	// 計算指數退避延遲
	delay := time.Duration(float64(policy.InitialDelay) * 
		pow(policy.BackoffFactor, float64(attempts)))
	
	if delay > policy.MaxDelay {
		delay = policy.MaxDelay
	}
	
	return delay
}

// IncrementRetryCount 增加重試計數
func (e *BatchErrorManager) IncrementRetryCount(filename, errorMsg string) {
	e.retryMutex.Lock()
	defer e.retryMutex.Unlock()
	
	key := fmt.Sprintf("%s:%s", filename, errorMsg)
	e.retryAttempts[key]++
}

// ResetRetryCount 重置重試計數
func (e *BatchErrorManager) ResetRetryCount(filename, errorMsg string) {
	e.retryMutex.Lock()
	defer e.retryMutex.Unlock()
	
	key := fmt.Sprintf("%s:%s", filename, errorMsg)
	delete(e.retryAttempts, key)
}

// SetRetryPolicy 設定重試策略
func (e *BatchErrorManager) SetRetryPolicy(errorType string, policy *RetryPolicy) {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	
	e.retryPolicies[errorType] = policy
}

// GetErrorStatistics 取得錯誤統計
func (e *BatchErrorManager) GetErrorStatistics() map[string]*ErrorStatistics {
	e.mutex.RLock()
	defer e.mutex.RUnlock()
	
	// 複製統計資料
	result := make(map[string]*ErrorStatistics)
	for k, v := range e.errorStats {
		statsCopy := *v
		result[k] = &statsCopy
	}
	
	return result
}

// CleanupOldErrors 清理舊錯誤記錄
func (e *BatchErrorManager) CleanupOldErrors(olderThan time.Duration) int {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	
	cutoff := time.Now().Add(-olderThan)
	cleaned := 0
	
	for batchID, errors := range e.errorHistory {
		var filteredErrors []models.BatchError
		for _, err := range errors {
			if err.Timestamp.After(cutoff) {
				filteredErrors = append(filteredErrors, err)
			} else {
				cleaned++
			}
		}
		
		if len(filteredErrors) == 0 {
			delete(e.errorHistory, batchID)
		} else {
			e.errorHistory[batchID] = filteredErrors
		}
	}
	
	return cleaned
}

// 私有方法

// updateErrorStatistics 更新錯誤統計
func (e *BatchErrorManager) updateErrorStatistics(err models.BatchError) {
	errorType := e.classifyError(err.Error)
	
	if _, exists := e.errorStats[errorType]; !exists {
		e.errorStats[errorType] = &ErrorStatistics{
			ErrorType:       errorType,
			TotalCount:      0,
			FirstOccurrence: err.Timestamp,
			LastOccurrence:  err.Timestamp,
			FilePatterns:    make(map[string]int),
			ErrorMessages:   make(map[string]int),
			HourlyCount:     make(map[string]int),
		}
	}
	
	stats := e.errorStats[errorType]
	stats.TotalCount++
	stats.LastOccurrence = err.Timestamp
	
	if err.Timestamp.Before(stats.FirstOccurrence) {
		stats.FirstOccurrence = err.Timestamp
	}
	
	// 統計檔案模式
	fileExt := getFileExtension(err.Filename)
	stats.FilePatterns[fileExt]++
	
	// 統計錯誤訊息
	stats.ErrorMessages[err.Error]++
	
	// 統計每小時錯誤數
	hourKey := err.Timestamp.Format("2006-01-02-15")
	stats.HourlyCount[hourKey]++
}

// classifyError 分類錯誤
func (e *BatchErrorManager) classifyError(errorMsg string) string {
	errorMsg = strings.ToLower(errorMsg)
	
	switch {
	case strings.Contains(errorMsg, "network") || strings.Contains(errorMsg, "connection"):
		return "network_error"
	case strings.Contains(errorMsg, "timeout"):
		return "timeout_error"
	case strings.Contains(errorMsg, "permission") || strings.Contains(errorMsg, "access"):
		return "permission_error"
	case strings.Contains(errorMsg, "not found") || strings.Contains(errorMsg, "no such file"):
		return "file_not_found"
	case strings.Contains(errorMsg, "format") || strings.Contains(errorMsg, "invalid"):
		return "format_error"
	case strings.Contains(errorMsg, "size") || strings.Contains(errorMsg, "too large"):
		return "size_error"
	case strings.Contains(errorMsg, "memory") || strings.Contains(errorMsg, "out of memory"):
		return "memory_error"
	case strings.Contains(errorMsg, "api") || strings.Contains(errorMsg, "service"):
		return "api_error"
	default:
		return "unknown_error"
	}
}

// generateTopErrors 生成 Top 錯誤
func (e *BatchErrorManager) generateTopErrors(errorMessages map[string]int) []ErrorSummary {
	type errorCount struct {
		message string
		count   int
	}
	
	var errors []errorCount
	for msg, count := range errorMessages {
		errors = append(errors, errorCount{message: msg, count: count})
	}
	
	// 按錯誤數量排序
	sort.Slice(errors, func(i, j int) bool {
		return errors[i].count > errors[j].count
	})
	
	// 取前 10 個
	maxCount := 10
	if len(errors) < maxCount {
		maxCount = len(errors)
	}
	
	result := make([]ErrorSummary, maxCount)
	for i := 0; i < maxCount; i++ {
		result[i] = ErrorSummary{
			ErrorMessage: errors[i].message,
			Count:        errors[i].count,
			Files:        []string{}, // 這裡可以進一步實作檔案列表
		}
	}
	
	return result
}

// generateRecommendations 生成建議
func (e *BatchErrorManager) generateRecommendations(report *ErrorReport) []string {
	var recommendations []string
	
	// 基於錯誤類型生成建議
	for errorType, count := range report.ErrorsByType {
		percentage := float64(count) / float64(report.TotalErrors) * 100
		
		switch errorType {
		case "network_error":
			if percentage > 20 {
				recommendations = append(recommendations, "網路錯誤較多，建議檢查網路連線穩定性")
			}
		case "timeout_error":
			if percentage > 15 {
				recommendations = append(recommendations, "超時錯誤較多，建議增加處理超時時間")
			}
		case "permission_error":
			if percentage > 10 {
				recommendations = append(recommendations, "權限錯誤較多，建議檢查檔案存取權限")
			}
		case "file_not_found":
			if percentage > 25 {
				recommendations = append(recommendations, "檔案不存在錯誤較多，建議檢查檔案路徑")
			}
		case "format_error":
			if percentage > 30 {
				recommendations = append(recommendations, "格式錯誤較多，建議檢查檔案格式支援")
			}
		case "size_error":
			if percentage > 20 {
				recommendations = append(recommendations, "檔案大小錯誤較多，建議調整檔案大小限制")
			}
		}
	}
	
	// 基於錯誤總數生成建議
	if report.TotalErrors > 100 {
		recommendations = append(recommendations, "錯誤數量較多，建議分批處理或檢查系統資源")
	}
	
	return recommendations
}

// getDefaultRetryPolicy 取得預設重試策略
func (e *BatchErrorManager) getDefaultRetryPolicy() *RetryPolicy {
	return &RetryPolicy{
		MaxRetries:    3,
		InitialDelay:  time.Second,
		MaxDelay:      30 * time.Second,
		BackoffFactor: 2.0,
		RetryableErrors: []string{
			"network_error",
			"timeout_error",
			"api_error",
		},
	}
}

// isRetryableError 檢查是否為可重試錯誤
func (e *BatchErrorManager) isRetryableError(errorMsg string, retryableErrors []string) bool {
	errorType := e.classifyError(errorMsg)
	
	for _, retryableType := range retryableErrors {
		if errorType == retryableType {
			return true
		}
	}
	
	return false
}

// getFileExtension 取得檔案副檔名
func getFileExtension(filename string) string {
	if idx := strings.LastIndex(filename, "."); idx != -1 {
		return strings.ToLower(filename[idx:])
	}
	return "no_extension"
}

// pow 計算冪次方（簡單實作）
func pow(base, exp float64) float64 {
	if exp == 0 {
		return 1
	}
	result := base
	for i := 1; i < int(exp); i++ {
		result *= base
	}
	return result
}