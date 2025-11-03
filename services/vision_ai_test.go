package services

import (
	"context"
	"testing"
	"time"

	"semantic-text-processor/models"
)

// TestGPT4VisionService 測試 GPT-4 Vision 服務
func TestGPT4VisionService(t *testing.T) {
	// 使用模擬服務進行測試
	mockService := NewMockVisionAIService()
	
	ctx := context.Background()
	testImageURL := "https://example.com/test-image.png"
	
	t.Run("AnalyzeImage_Success", func(t *testing.T) {
		// 設定預期回應
		expectedAnalysis := &models.ImageAnalysis{
			Description: "這是一個測試圖片，包含系統架構圖",
			Tags:        []string{"architecture", "system", "diagram"},
			Model:       "mock-vision-model",
			Confidence:  0.9,
			AnalyzedAt:  time.Now(),
		}
		
		mockService.SetResponse(testImageURL, expectedAnalysis)
		
		// 執行分析
		result, err := mockService.AnalyzeImage(ctx, testImageURL, nil)
		if err != nil {
			t.Fatalf("AnalyzeImage failed: %v", err)
		}
		
		// 驗證結果
		if result.Description != expectedAnalysis.Description {
			t.Errorf("Expected description '%s', got '%s'", 
				expectedAnalysis.Description, result.Description)
		}
		
		if len(result.Tags) != len(expectedAnalysis.Tags) {
			t.Errorf("Expected %d tags, got %d", len(expectedAnalysis.Tags), len(result.Tags))
		}
		
		if result.Model != expectedAnalysis.Model {
			t.Errorf("Expected model '%s', got '%s'", expectedAnalysis.Model, result.Model)
		}
		
		if result.Confidence != expectedAnalysis.Confidence {
			t.Errorf("Expected confidence %f, got %f", expectedAnalysis.Confidence, result.Confidence)
		}
	})
	
	t.Run("AnalyzeImage_WithOptions", func(t *testing.T) {
		options := &models.AnalysisOptions{
			DetailLevel: "high",
			Language:    "zh-TW",
			MaxTokens:   1500,
		}
		
		result, err := mockService.AnalyzeImage(ctx, testImageURL, options)
		if err != nil {
			t.Fatalf("AnalyzeImage with options failed: %v", err)
		}
		
		// 驗證基本結果
		if result.Description == "" {
			t.Error("Description should not be empty")
		}
		
		if len(result.Tags) == 0 {
			t.Error("Tags should not be empty")
		}
		
		if result.Confidence < 0 || result.Confidence > 1 {
			t.Errorf("Confidence should be between 0 and 1, got %f", result.Confidence)
		}
	})
	
	t.Run("AnalyzeImage_Failure", func(t *testing.T) {
		mockService.SetShouldFail(true)
		
		_, err := mockService.AnalyzeImage(ctx, testImageURL, nil)
		if err == nil {
			t.Error("Expected error when service is set to fail")
		}
		
		// 重置失敗狀態
		mockService.SetShouldFail(false)
	})
	
	t.Run("AnalyzeImage_ContextCancellation", func(t *testing.T) {
		// 設定延遲
		mockService.SetDelay(100 * time.Millisecond)
		
		// 建立會被取消的 context
		cancelCtx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
		defer cancel()
		
		_, err := mockService.AnalyzeImage(cancelCtx, testImageURL, nil)
		if err == nil {
			t.Error("Expected context cancellation error")
		}
		
		if err != context.DeadlineExceeded {
			t.Errorf("Expected context.DeadlineExceeded, got %v", err)
		}
		
		// 重置延遲
		mockService.SetDelay(0)
	})
}

// TestCLIPEmbeddingService 測試 CLIP 向量化服務
func TestCLIPEmbeddingService(t *testing.T) {
	// 使用模擬服務進行測試
	mockService := NewMockImageEmbeddingService()
	
	ctx := context.Background()
	testImageURL := "https://example.com/test-image.png"
	
	t.Run("GenerateEmbedding_Success", func(t *testing.T) {
		result, err := mockService.GenerateEmbedding(ctx, testImageURL)
		if err != nil {
			t.Fatalf("GenerateEmbedding failed: %v", err)
		}
		
		// 驗證向量維度
		if len(result) != 512 {
			t.Errorf("Expected embedding dimension 512, got %d", len(result))
		}
		
		// 驗證向量值範圍
		for i, val := range result {
			if val < -1.0 || val > 1.0 {
				t.Errorf("Embedding value at index %d is out of range [-1, 1]: %f", i, val)
			}
		}
	})
	
	t.Run("GenerateBatchEmbeddings_Success", func(t *testing.T) {
		imageURLs := []string{
			"https://example.com/image1.png",
			"https://example.com/image2.jpg",
			"https://example.com/image3.gif",
		}
		
		results, err := mockService.GenerateBatchEmbeddings(ctx, imageURLs)
		if err != nil {
			t.Fatalf("GenerateBatchEmbeddings failed: %v", err)
		}
		
		// 驗證結果數量
		if len(results) != len(imageURLs) {
			t.Errorf("Expected %d embeddings, got %d", len(imageURLs), len(results))
		}
		
		// 驗證每個向量
		for i, embedding := range results {
			if len(embedding) != 512 {
				t.Errorf("Embedding %d has wrong dimension: expected 512, got %d", i, len(embedding))
			}
		}
	})
	
	t.Run("GenerateEmbedding_EmptyInput", func(t *testing.T) {
		results, err := mockService.GenerateBatchEmbeddings(ctx, []string{})
		if err != nil {
			t.Fatalf("GenerateBatchEmbeddings with empty input failed: %v", err)
		}
		
		if len(results) != 0 {
			t.Errorf("Expected empty results for empty input, got %d results", len(results))
		}
	})
	
	t.Run("GenerateEmbedding_Failure", func(t *testing.T) {
		mockService.SetShouldFail(true)
		
		_, err := mockService.GenerateEmbedding(ctx, testImageURL)
		if err == nil {
			t.Error("Expected error when service is set to fail")
		}
		
		// 重置失敗狀態
		mockService.SetShouldFail(false)
	})
	
	t.Run("GenerateEmbedding_ContextCancellation", func(t *testing.T) {
		// 設定延遲
		mockService.SetDelay(100 * time.Millisecond)
		
		// 建立會被取消的 context
		cancelCtx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
		defer cancel()
		
		_, err := mockService.GenerateEmbedding(cancelCtx, testImageURL)
		if err == nil {
			t.Error("Expected context cancellation error")
		}
		
		if err != context.DeadlineExceeded {
			t.Errorf("Expected context.DeadlineExceeded, got %v", err)
		}
		
		// 重置延遲
		mockService.SetDelay(0)
	})
	
	t.Run("GenerateEmbedding_Deterministic", func(t *testing.T) {
		// 測試相同輸入產生相同輸出
		embedding1, err := mockService.GenerateEmbedding(ctx, testImageURL)
		if err != nil {
			t.Fatalf("First embedding generation failed: %v", err)
		}
		
		embedding2, err := mockService.GenerateEmbedding(ctx, testImageURL)
		if err != nil {
			t.Fatalf("Second embedding generation failed: %v", err)
		}
		
		// 驗證兩次結果相同
		if len(embedding1) != len(embedding2) {
			t.Error("Embeddings should have same length")
		}
		
		for i := 0; i < len(embedding1) && i < len(embedding2); i++ {
			if embedding1[i] != embedding2[i] {
				t.Errorf("Embeddings should be deterministic, difference at index %d: %f vs %f", 
					i, embedding1[i], embedding2[i])
				break
			}
		}
	})
}

// TestVisionAIServiceIntegration 測試 Vision AI 服務整合
func TestVisionAIServiceIntegration(t *testing.T) {
	mockVision := NewMockVisionAIService()
	mockEmbedding := NewMockImageEmbeddingService()
	
	ctx := context.Background()
	testImageURL := "https://example.com/architecture-diagram.png"
	
	t.Run("CompleteAnalysisFlow", func(t *testing.T) {
		// 1. 設定 Vision AI 回應
		expectedAnalysis := &models.ImageAnalysis{
			Description: "這是一個微服務架構圖，包含 API Gateway、Service Discovery 和多個微服務",
			Tags:        []string{"architecture", "microservices", "api-gateway", "service-discovery"},
			Model:       "gpt-4-vision-preview",
			Confidence:  0.92,
			AnalyzedAt:  time.Now(),
		}
		mockVision.SetResponse(testImageURL, expectedAnalysis)
		
		// 2. 設定 CLIP 向量回應
		expectedEmbedding := make([]float64, 512)
		for i := range expectedEmbedding {
			expectedEmbedding[i] = float64(i) / 512.0
		}
		mockEmbedding.SetEmbedding(testImageURL, expectedEmbedding)
		
		// 3. 執行 Vision 分析
		analysis, err := mockVision.AnalyzeImage(ctx, testImageURL, &models.AnalysisOptions{
			DetailLevel: "high",
			Language:    "zh-TW",
			MaxTokens:   1500,
		})
		if err != nil {
			t.Fatalf("Vision analysis failed: %v", err)
		}
		
		// 4. 執行向量生成
		embedding, err := mockEmbedding.GenerateEmbedding(ctx, testImageURL)
		if err != nil {
			t.Fatalf("Embedding generation failed: %v", err)
		}
		
		// 5. 驗證結果
		if analysis.Description != expectedAnalysis.Description {
			t.Errorf("Description mismatch")
		}
		
		if len(analysis.Tags) != len(expectedAnalysis.Tags) {
			t.Errorf("Tags count mismatch: expected %d, got %d", 
				len(expectedAnalysis.Tags), len(analysis.Tags))
		}
		
		if len(embedding) != 512 {
			t.Errorf("Embedding dimension mismatch: expected 512, got %d", len(embedding))
		}
		
		// 6. 驗證標籤內容
		tagMap := make(map[string]bool)
		for _, tag := range analysis.Tags {
			tagMap[tag] = true
		}
		
		expectedTags := []string{"architecture", "microservices"}
		for _, expectedTag := range expectedTags {
			if !tagMap[expectedTag] {
				t.Errorf("Expected tag '%s' not found in result", expectedTag)
			}
		}
	})
}