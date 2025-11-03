package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"semantic-text-processor/models"
)

// GPT4VisionService GPT-4 Vision API 服務實作
type GPT4VisionService struct {
	apiKey     string
	model      string
	httpClient *http.Client
	baseURL    string
	maxTokens  int
	temperature float64
	language   string
}

// NewGPT4VisionService 建立新的 GPT-4 Vision 服務
func NewGPT4VisionService(apiKey, model string) VisionAIService {
	return &GPT4VisionService{
		apiKey:      apiKey,
		model:       model,
		baseURL:     "https://api.openai.com/v1",
		maxTokens:   1000,
		temperature: 0.1,
		language:    "zh-TW",
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// NewGPT4VisionServiceWithConfig 使用配置建立 GPT-4 Vision 服務
func NewGPT4VisionServiceWithConfig(apiKey string, config *VisionConfig) VisionAIService {
	service := &GPT4VisionService{
		apiKey:      apiKey,
		model:       config.Model,
		baseURL:     "https://api.openai.com/v1",
		maxTokens:   config.MaxTokens,
		temperature: config.Temperature,
		language:    config.Language,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}
	
	return service
}

// VisionConfig Vision AI 配置
type VisionConfig struct {
	Model       string        `json:"model"`
	MaxTokens   int           `json:"max_tokens"`
	Temperature float64       `json:"temperature"`
	Language    string        `json:"language"`
	DetailLevel string        `json:"detail_level"`
	Timeout     time.Duration `json:"timeout"`
	RetryCount  int           `json:"retry_count"`
}

// AnalyzeImage 分析圖片並生成描述
func (g *GPT4VisionService) AnalyzeImage(ctx context.Context, imageURL string, options *models.AnalysisOptions) (*models.ImageAnalysis, error) {
	// 建立分析提示詞
	prompt := g.buildPrompt(options)
	
	// 建立 API 請求
	request := g.buildAPIRequest(imageURL, prompt, options)
	
	// 執行 API 呼叫
	response, err := g.callAPI(ctx, request)
	if err != nil {
		return nil, NewVisionAPIError(g.model, imageURL, err)
	}
	
	// 解析回應
	analysis, err := g.parseResponse(response)
	if err != nil {
		return nil, NewVisionAPIError(g.model, imageURL, err)
	}
	
	return analysis, nil
}

// buildPrompt 建立分析提示詞
func (g *GPT4VisionService) buildPrompt(options *models.AnalysisOptions) string {
	language := g.language
	if options != nil && options.Language != "" {
		language = options.Language
	}
	
	var prompt string
	
	switch language {
	case "zh-TW", "zh-CN", "zh":
		prompt = `請詳細分析這張圖片，並提供以下資訊：

1. **圖片類型**: 這是什麼類型的圖片？（截圖、圖表、照片、插圖等）

2. **主要內容**: 描述圖片中的主要物件、元素和內容

3. **技術相關**: 如果是技術相關的圖片，請描述：
   - 系統架構、流程圖、程式碼截圖等技術細節
   - 使用的技術、工具或框架
   - 圖表中的關鍵概念和關係

4. **視覺特徵**: 
   - 顏色、佈局、風格
   - 文字內容（如果有的話）
   - 重要的視覺元素

5. **用途和情境**: 這張圖片可能的用途或使用情境

6. **關鍵標籤**: 提供 5-10 個最相關的標籤，用逗號分隔

請用繁體中文回答，內容要詳細且準確。`
		
	default:
		prompt = `Please analyze this image in detail and provide the following information:

1. **Image Type**: What type of image is this? (screenshot, diagram, photo, illustration, etc.)

2. **Main Content**: Describe the main objects, elements, and content in the image

3. **Technical Details**: If this is a technical image, please describe:
   - System architecture, flowcharts, code screenshots, etc.
   - Technologies, tools, or frameworks used
   - Key concepts and relationships in diagrams

4. **Visual Features**:
   - Colors, layout, style
   - Text content (if any)
   - Important visual elements

5. **Purpose and Context**: Possible uses or contexts for this image

6. **Key Tags**: Provide 5-10 most relevant tags, separated by commas

Please provide a detailed and accurate analysis.`
	}
	
	// 根據詳細程度調整提示詞
	if options != nil {
		switch options.DetailLevel {
		case "low":
			prompt = "請簡要描述這張圖片的主要內容和類型，並提供 3-5 個相關標籤。"
		case "high":
			prompt += "\n\n請特別注意細節，包括背景元素、小字內容、顏色搭配等所有可見的資訊。"
		}
	}
	
	return prompt
}

// buildAPIRequest 建立 API 請求
func (g *GPT4VisionService) buildAPIRequest(imageURL, prompt string, options *models.AnalysisOptions) map[string]interface{} {
	maxTokens := g.maxTokens
	if options != nil && options.MaxTokens > 0 {
		maxTokens = options.MaxTokens
	}
	
	request := map[string]interface{}{
		"model":       g.model,
		"max_tokens":  maxTokens,
		"temperature": g.temperature,
		"messages": []map[string]interface{}{
			{
				"role": "user",
				"content": []map[string]interface{}{
					{
						"type": "text",
						"text": prompt,
					},
					{
						"type": "image_url",
						"image_url": map[string]interface{}{
							"url": imageURL,
							"detail": g.getDetailLevel(options),
						},
					},
				},
			},
		},
	}
	
	return request
}

// getDetailLevel 取得詳細程度設定
func (g *GPT4VisionService) getDetailLevel(options *models.AnalysisOptions) string {
	if options == nil || options.DetailLevel == "" {
		return "auto"
	}
	
	switch options.DetailLevel {
	case "low":
		return "low"
	case "high":
		return "high"
	default:
		return "auto"
	}
}

// callAPI 呼叫 OpenAI API
func (g *GPT4VisionService) callAPI(ctx context.Context, request map[string]interface{}) (map[string]interface{}, error) {
	// 序列化請求
	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	
	// 建立 HTTP 請求
	req, err := http.NewRequestWithContext(ctx, "POST", g.baseURL+"/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	// 設定標頭
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+g.apiKey)
	
	// 執行請求
	resp, err := g.httpClient.Do(req)
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
func (g *GPT4VisionService) parseResponse(response map[string]interface{}) (*models.ImageAnalysis, error) {
	// 取得 choices
	choices, ok := response["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}
	
	// 取得第一個 choice
	choice, ok := choices[0].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid choice format")
	}
	
	// 取得 message
	message, ok := choice["message"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("no message in choice")
	}
	
	// 取得 content
	content, ok := message["content"].(string)
	if !ok {
		return nil, fmt.Errorf("no content in message")
	}
	
	// 解析內容並提取標籤
	description, tags := g.parseContent(content)
	
	// 計算信心度（基於回應長度和內容品質）
	confidence := g.calculateConfidence(content)
	
	return &models.ImageAnalysis{
		Description: description,
		Tags:        tags,
		Model:       g.model,
		Confidence:  confidence,
		AnalyzedAt:  time.Now(),
	}, nil
}

// parseContent 解析內容並提取標籤
func (g *GPT4VisionService) parseContent(content string) (string, []string) {
	// 簡單的標籤提取邏輯
	// 在實際實作中，可以使用更複雜的 NLP 技術
	
	description := content
	var tags []string
	
	// 尋找標籤相關的關鍵字
	keywords := []string{
		"架構", "系統", "流程", "圖表", "截圖", "程式碼", "介面",
		"資料庫", "API", "服務", "網路", "安全", "監控", "部署",
		"architecture", "system", "flow", "diagram", "screenshot", "code", "interface",
		"database", "api", "service", "network", "security", "monitoring", "deployment",
	}
	
	for _, keyword := range keywords {
		if contains(content, keyword) {
			tags = append(tags, keyword)
		}
		
		// 限制標籤數量
		if len(tags) >= 10 {
			break
		}
	}
	
	// 如果沒有找到標籤，提供預設標籤
	if len(tags) == 0 {
		tags = []string{"image", "analysis"}
	}
	
	return description, tags
}

// calculateConfidence 計算信心度
func (g *GPT4VisionService) calculateConfidence(content string) float64 {
	// 基於內容長度和品質的簡單信心度計算
	length := len(content)
	
	switch {
	case length < 50:
		return 0.3 // 內容太短，信心度低
	case length < 200:
		return 0.6 // 中等長度
	case length < 500:
		return 0.8 // 較長的描述
	default:
		return 0.9 // 詳細的描述
	}
}

// MockVisionAIService 模擬 Vision AI 服務（用於測試）
type MockVisionAIService struct {
	responses map[string]*models.ImageAnalysis
	delay     time.Duration
	shouldFail bool
}

// NewMockVisionAIService 建立模擬 Vision AI 服務
func NewMockVisionAIService() *MockVisionAIService {
	return &MockVisionAIService{
		responses: make(map[string]*models.ImageAnalysis),
		delay:     0,
		shouldFail: false,
	}
}

// SetResponse 設定特定 URL 的回應
func (m *MockVisionAIService) SetResponse(imageURL string, analysis *models.ImageAnalysis) {
	m.responses[imageURL] = analysis
}

// SetDelay 設定回應延遲
func (m *MockVisionAIService) SetDelay(delay time.Duration) {
	m.delay = delay
}

// SetShouldFail 設定是否應該失敗
func (m *MockVisionAIService) SetShouldFail(shouldFail bool) {
	m.shouldFail = shouldFail
}

// AnalyzeImage 模擬圖片分析
func (m *MockVisionAIService) AnalyzeImage(ctx context.Context, imageURL string, options *models.AnalysisOptions) (*models.ImageAnalysis, error) {
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
		return nil, fmt.Errorf("mock vision service failure")
	}
	
	// 檢查是否有預設回應
	if response, exists := m.responses[imageURL]; exists {
		return response, nil
	}
	
	// 預設回應
	return &models.ImageAnalysis{
		Description: fmt.Sprintf("Mock analysis for image: %s", imageURL),
		Tags:        []string{"mock", "test", "image"},
		Model:       "mock-vision-model",
		Confidence:  0.8,
		AnalyzedAt:  time.Now(),
	}, nil
}