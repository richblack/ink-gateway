package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// CLIPEmbeddingService CLIP 圖片向量化服務
type CLIPEmbeddingService struct {
	apiURL     string
	httpClient *http.Client
	model      string
	dimensions int
}

// NewCLIPEmbeddingService 建立新的 CLIP 向量化服務
func NewCLIPEmbeddingService(apiURL string) ImageEmbeddingService {
	return &CLIPEmbeddingService{
		apiURL:     apiURL,
		model:      "clip-vit-b-32",
		dimensions: 512,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// NewCLIPEmbeddingServiceWithConfig 使用配置建立 CLIP 向量化服務
func NewCLIPEmbeddingServiceWithConfig(apiURL string, config *CLIPConfig) ImageEmbeddingService {
	return &CLIPEmbeddingService{
		apiURL:     apiURL,
		model:      config.Model,
		dimensions: config.Dimensions,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// CLIPConfig CLIP 配置
type CLIPConfig struct {
	Model      string        `json:"model"`
	Dimensions int           `json:"dimensions"`
	Timeout    time.Duration `json:"timeout"`
}

// GenerateEmbedding 生成單張圖片的向量
func (c *CLIPEmbeddingService) GenerateEmbedding(ctx context.Context, imageURL string) ([]float64, error) {
	embeddings, err := c.GenerateBatchEmbeddings(ctx, []string{imageURL})
	if err != nil {
		return nil, err
	}
	
	if len(embeddings) == 0 {
		return nil, fmt.Errorf("no embedding returned for image")
	}
	
	return embeddings[0], nil
}

// GenerateBatchEmbeddings 批次生成多張圖片的向量
func (c *CLIPEmbeddingService) GenerateBatchEmbeddings(ctx context.Context, imageURLs []string) ([][]float64, error) {
	if len(imageURLs) == 0 {
		return [][]float64{}, nil
	}
	
	// 建立請求
	request := map[string]interface{}{
		"images": imageURLs,
		"model":  c.model,
	}
	
	// 執行 API 呼叫
	response, err := c.callAPI(ctx, request)
	if err != nil {
		return nil, NewMediaEmbeddingError(c.model, "image", err)
	}
	
	// 解析回應
	embeddings, err := c.parseResponse(response, len(imageURLs))
	if err != nil {
		return nil, NewMediaEmbeddingError(c.model, "image", err)
	}
	
	return embeddings, nil
}

// callAPI 呼叫 CLIP API
func (c *CLIPEmbeddingService) callAPI(ctx context.Context, request map[string]interface{}) (map[string]interface{}, error) {
	// 序列化請求
	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	
	// 建立 HTTP 請求
	req, err := http.NewRequestWithContext(ctx, "POST", c.apiURL+"/embeddings", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	// 設定標頭
	req.Header.Set("Content-Type", "application/json")
	
	// 執行請求
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	
	// 讀取回應
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	
	// 檢查 HTTP 狀態碼
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}
	
	// 解析 JSON 回應
	var response map[string]interface{}
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}
	
	return response, nil
}

// parseResponse 解析 API 回應
func (c *CLIPEmbeddingService) parseResponse(response map[string]interface{}, expectedCount int) ([][]float64, error) {
	// 取得 embeddings
	embeddingsInterface, ok := response["embeddings"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("no embeddings in response")
	}
	
	if len(embeddingsInterface) != expectedCount {
		return nil, fmt.Errorf("expected %d embeddings, got %d", expectedCount, len(embeddingsInterface))
	}
	
	// 轉換為 float64 切片
	embeddings := make([][]float64, len(embeddingsInterface))
	for i, embInterface := range embeddingsInterface {
		embSlice, ok := embInterface.([]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid embedding format at index %d", i)
		}
		
		embedding := make([]float64, len(embSlice))
		for j, val := range embSlice {
			floatVal, ok := val.(float64)
			if !ok {
				return nil, fmt.Errorf("invalid embedding value at index %d, %d", i, j)
			}
			embedding[j] = floatVal
		}
		
		// 驗證向量維度
		if len(embedding) != c.dimensions {
			return nil, fmt.Errorf("expected embedding dimension %d, got %d", c.dimensions, len(embedding))
		}
		
		embeddings[i] = embedding
	}
	
	return embeddings, nil
}

// LocalCLIPService 本地 CLIP 服務（使用本地模型）
type LocalCLIPService struct {
	modelPath  string
	dimensions int
	model      string
}

// NewLocalCLIPService 建立本地 CLIP 服務
func NewLocalCLIPService(modelPath string) ImageEmbeddingService {
	return &LocalCLIPService{
		modelPath:  modelPath,
		dimensions: 512,
		model:      "clip-vit-b-32-local",
	}
}

// GenerateEmbedding 生成單張圖片的向量（本地實作）
func (l *LocalCLIPService) GenerateEmbedding(ctx context.Context, imageURL string) ([]float64, error) {
	// 這裡應該整合本地的 CLIP 模型
	// 由於需要 Python 環境和模型檔案，這裡提供一個框架實作
	
	// TODO: 實作本地 CLIP 模型呼叫
	// 可能的實作方式：
	// 1. 使用 CGO 呼叫 Python
	// 2. 使用 subprocess 呼叫 Python 腳本
	// 3. 使用 ONNX Runtime Go 載入 ONNX 模型
	
	return nil, fmt.Errorf("local CLIP service not implemented yet")
}

// GenerateBatchEmbeddings 批次生成多張圖片的向量（本地實作）
func (l *LocalCLIPService) GenerateBatchEmbeddings(ctx context.Context, imageURLs []string) ([][]float64, error) {
	// 本地批次處理實作
	embeddings := make([][]float64, len(imageURLs))
	
	for i, imageURL := range imageURLs {
		embedding, err := l.GenerateEmbedding(ctx, imageURL)
		if err != nil {
			return nil, fmt.Errorf("failed to generate embedding for image %d: %w", i, err)
		}
		embeddings[i] = embedding
	}
	
	return embeddings, nil
}

// MockImageEmbeddingService 模擬圖片向量化服務（用於測試）
type MockImageEmbeddingService struct {
	embeddings map[string][]float64
	delay      time.Duration
	shouldFail bool
	dimensions int
}

// NewMockImageEmbeddingService 建立模擬圖片向量化服務
func NewMockImageEmbeddingService() *MockImageEmbeddingService {
	return &MockImageEmbeddingService{
		embeddings: make(map[string][]float64),
		delay:      0,
		shouldFail: false,
		dimensions: 512,
	}
}

// SetEmbedding 設定特定 URL 的向量
func (m *MockImageEmbeddingService) SetEmbedding(imageURL string, embedding []float64) {
	m.embeddings[imageURL] = embedding
}

// SetDelay 設定回應延遲
func (m *MockImageEmbeddingService) SetDelay(delay time.Duration) {
	m.delay = delay
}

// SetShouldFail 設定是否應該失敗
func (m *MockImageEmbeddingService) SetShouldFail(shouldFail bool) {
	m.shouldFail = shouldFail
}

// GenerateEmbedding 模擬向量生成
func (m *MockImageEmbeddingService) GenerateEmbedding(ctx context.Context, imageURL string) ([]float64, error) {
	// 模擬延遲
	if m.delay > 0 {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(m.delay):
		}
	}
	
	// 模擬失敗
	if m.shouldFail {
		return nil, fmt.Errorf("mock embedding service failure")
	}
	
	// 檢查是否有預設向量
	if embedding, exists := m.embeddings[imageURL]; exists {
		return embedding, nil
	}
	
	// 生成模擬向量（基於 URL 的確定性向量）
	embedding := make([]float64, m.dimensions)
	hash := simpleHash(imageURL)
	
	for i := 0; i < m.dimensions; i++ {
		// 使用簡單的偽隨機生成確定性向量
		embedding[i] = float64((hash+uint32(i))%1000) / 1000.0 - 0.5
	}
	
	return embedding, nil
}

// GenerateBatchEmbeddings 模擬批次向量生成
func (m *MockImageEmbeddingService) GenerateBatchEmbeddings(ctx context.Context, imageURLs []string) ([][]float64, error) {
	embeddings := make([][]float64, len(imageURLs))
	
	for i, imageURL := range imageURLs {
		embedding, err := m.GenerateEmbedding(ctx, imageURL)
		if err != nil {
			return nil, err
		}
		embeddings[i] = embedding
	}
	
	return embeddings, nil
}

// simpleHash 簡單的字串雜湊函數
func simpleHash(s string) uint32 {
	var hash uint32 = 5381
	for _, c := range s {
		hash = ((hash << 5) + hash) + uint32(c)
	}
	return hash
}