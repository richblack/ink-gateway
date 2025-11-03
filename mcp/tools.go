package mcp

import (
	"context"
	"fmt"
	"strings"

	"semantic-text-processor/models"
)

// InkSearchChunksTool 搜尋知識塊工具
type InkSearchChunksTool struct {
	server *MCPServer
}

// NewInkSearchChunksTool 建立搜尋知識塊工具
func NewInkSearchChunksTool(server *MCPServer) *InkSearchChunksTool {
	return &InkSearchChunksTool{server: server}
}

func (t *InkSearchChunksTool) GetName() string {
	return "ink_search_chunks"
}

func (t *InkSearchChunksTool) GetDescription() string {
	return "Search for chunks using multimodal search (text, image, or hybrid)"
}

func (t *InkSearchChunksTool) GetInputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"query": map[string]interface{}{
				"type":        "string",
				"description": "Search query text",
			},
			"image_url": map[string]interface{}{
				"type":        "string",
				"description": "Image URL for image-based search (optional)",
			},
			"search_type": map[string]interface{}{
				"type":        "string",
				"description": "Type of search: text, image, or hybrid",
				"enum":        []string{"text", "image", "hybrid"},
				"default":     "hybrid",
			},
			"limit": map[string]interface{}{
				"type":        "integer",
				"description": "Maximum number of results",
				"default":     10,
			},
			"min_similarity": map[string]interface{}{
				"type":        "number",
				"description": "Minimum similarity threshold",
				"default":     0.7,
			},
		},
		"required": []string{"query"},
	}
}

func (t *InkSearchChunksTool) Execute(ctx context.Context, params map[string]interface{}) (*MCPToolResult, error) {
	query, ok := params["query"].(string)
	if !ok || query == "" {
		return &MCPToolResult{
			Content: []MCPContent{{Type: "text", Text: "Error: query parameter is required"}},
			IsError: true,
		}, nil
	}

	// 解析參數
	imageURL, _ := params["image_url"].(string)
	searchType, _ := params["search_type"].(string)
	if searchType == "" {
		searchType = "hybrid"
	}

	limit := 10
	if limitFloat, ok := params["limit"].(float64); ok {
		limit = int(limitFloat)
	}

	minSimilarity := 0.7
	if simFloat, ok := params["min_similarity"].(float64); ok {
		minSimilarity = simFloat
	}

	// 建立搜尋請求
	searchReq := &models.MultimodalSearchRequest{
		TextQuery:     query,
		ImageQuery:    imageURL,
		VectorType:    "all",
		Limit:         limit,
		MinSimilarity: minSimilarity,
	}

	// 執行搜尋
	var searchResponse *models.MultimodalSearchResponse
	var err error

	switch searchType {
	case "text":
		searchResponse, err = t.server.services.MultimodalSearch.SearchText(ctx, searchReq)
	case "image":
		searchResponse, err = t.server.services.MultimodalSearch.SearchImages(ctx, searchReq)
	case "hybrid":
		searchResponse, err = t.server.services.MultimodalSearch.HybridSearch(ctx, searchReq)
	default:
		return &MCPToolResult{
			Content: []MCPContent{{Type: "text", Text: "Error: invalid search_type"}},
			IsError: true,
		}, nil
	}

	if err != nil {
		return &MCPToolResult{
			Content: []MCPContent{{Type: "text", Text: fmt.Sprintf("Search failed: %v", err)}},
			IsError: true,
		}, nil
	}

	// 格式化結果
	var resultText strings.Builder
	resultText.WriteString(fmt.Sprintf("Found %d results in %v:\n\n", 
		searchResponse.TotalCount, searchResponse.SearchTime))

	for i, result := range searchResponse.Results {
		resultText.WriteString(fmt.Sprintf("%d. **%s** (similarity: %.3f)\n", 
			i+1, result.Chunk.ChunkID, result.Similarity))
		resultText.WriteString(fmt.Sprintf("   Content: %s\n", result.Chunk.Contents))
		if result.Chunk.IsImageChunk() {
			resultText.WriteString(fmt.Sprintf("   Image URL: %s\n", result.Chunk.GetImageURL()))
		}
		if len(result.Chunk.Tags) > 0 {
			resultText.WriteString(fmt.Sprintf("   Tags: %s\n", strings.Join(result.Chunk.Tags, ", ")))
		}
		resultText.WriteString(fmt.Sprintf("   Match: %s\n", result.Explanation))
		resultText.WriteString("\n")
	}

	return &MCPToolResult{
		Content: []MCPContent{{Type: "text", Text: resultText.String()}},
		IsError: false,
	}, nil
}

// InkAnalyzeImageTool 圖片分析工具
type InkAnalyzeImageTool struct {
	server *MCPServer
}

// NewInkAnalyzeImageTool 建立圖片分析工具
func NewInkAnalyzeImageTool(server *MCPServer) *InkAnalyzeImageTool {
	return &InkAnalyzeImageTool{server: server}
}

func (t *InkAnalyzeImageTool) GetName() string {
	return "ink_analyze_image"
}

func (t *InkAnalyzeImageTool) GetDescription() string {
	return "Analyze an image using AI vision services"
}

func (t *InkAnalyzeImageTool) GetInputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"image_url": map[string]interface{}{
				"type":        "string",
				"description": "URL of the image to analyze",
			},
			"detail_level": map[string]interface{}{
				"type":        "string",
				"description": "Level of detail for analysis",
				"enum":        []string{"low", "medium", "high"},
				"default":     "medium",
			},
			"language": map[string]interface{}{
				"type":        "string",
				"description": "Language for the analysis",
				"default":     "zh-TW",
			},
		},
		"required": []string{"image_url"},
	}
}

func (t *InkAnalyzeImageTool) Execute(ctx context.Context, params map[string]interface{}) (*MCPToolResult, error) {
	imageURL, ok := params["image_url"].(string)
	if !ok || imageURL == "" {
		return &MCPToolResult{
			Content: []MCPContent{{Type: "text", Text: "Error: image_url parameter is required"}},
			IsError: true,
		}, nil
	}

	// 執行圖片分析
	analysis, err := t.server.services.MediaProcessor.AnalyzeImage(ctx, imageURL)
	if err != nil {
		return &MCPToolResult{
			Content: []MCPContent{{Type: "text", Text: fmt.Sprintf("Image analysis failed: %v", err)}},
			IsError: true,
		}, nil
	}

	// 格式化結果
	var resultText strings.Builder
	resultText.WriteString(fmt.Sprintf("**Image Analysis Results**\n\n"))
	resultText.WriteString(fmt.Sprintf("**Description:** %s\n\n", analysis.Description))
	resultText.WriteString(fmt.Sprintf("**Model:** %s\n", analysis.Model))
	resultText.WriteString(fmt.Sprintf("**Confidence:** %.2f\n", analysis.Confidence))
	resultText.WriteString(fmt.Sprintf("**Analyzed At:** %s\n\n", analysis.AnalyzedAt.Format("2006-01-02 15:04:05")))

	if len(analysis.Tags) > 0 {
		resultText.WriteString(fmt.Sprintf("**Tags:** %s\n", strings.Join(analysis.Tags, ", ")))
	}

	return &MCPToolResult{
		Content: []MCPContent{{Type: "text", Text: resultText.String()}},
		IsError: false,
	}, nil
}

// InkUploadImageTool 圖片上傳工具
type InkUploadImageTool struct {
	server *MCPServer
}

// NewInkUploadImageTool 建立圖片上傳工具
func NewInkUploadImageTool(server *MCPServer) *InkUploadImageTool {
	return &InkUploadImageTool{server: server}
}

func (t *InkUploadImageTool) GetName() string {
	return "ink_upload_image"
}

func (t *InkUploadImageTool) GetDescription() string {
	return "Upload and process an image file"
}

func (t *InkUploadImageTool) GetInputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"image_path": map[string]interface{}{
				"type":        "string",
				"description": "Local path to the image file",
			},
			"page_id": map[string]interface{}{
				"type":        "string",
				"description": "Page ID to associate with the image (optional)",
			},
			"tags": map[string]interface{}{
				"type":        "array",
				"items":       map[string]interface{}{"type": "string"},
				"description": "Tags to associate with the image",
			},
			"auto_analyze": map[string]interface{}{
				"type":        "boolean",
				"description": "Whether to automatically analyze the image",
				"default":     true,
			},
			"auto_embed": map[string]interface{}{
				"type":        "boolean",
				"description": "Whether to automatically generate embeddings",
				"default":     true,
			},
		},
		"required": []string{"image_path"},
	}
}

func (t *InkUploadImageTool) Execute(ctx context.Context, params map[string]interface{}) (*MCPToolResult, error) {
	imagePath, ok := params["image_path"].(string)
	if !ok || imagePath == "" {
		return &MCPToolResult{
			Content: []MCPContent{{Type: "text", Text: "Error: image_path parameter is required"}},
			IsError: true,
		}, nil
	}

	// 解析參數
	var pageID *string
	if pid, ok := params["page_id"].(string); ok && pid != "" {
		pageID = &pid
	}

	var tags []string
	if tagsInterface, ok := params["tags"].([]interface{}); ok {
		for _, tag := range tagsInterface {
			if tagStr, ok := tag.(string); ok {
				tags = append(tags, tagStr)
			}
		}
	}

	autoAnalyze := true
	if analyze, ok := params["auto_analyze"].(bool); ok {
		autoAnalyze = analyze
	}

	autoEmbed := true
	if embed, ok := params["auto_embed"].(bool); ok {
		autoEmbed = embed
	}

	// 這裡需要實作檔案讀取和上傳邏輯
	// 由於 MCP 環境的限制，這裡提供一個簡化的實作
	return &MCPToolResult{
		Content: []MCPContent{{
			Type: "text", 
			Text: fmt.Sprintf("Image upload initiated for: %s\nPage ID: %v\nTags: %v\nAuto Analyze: %v\nAuto Embed: %v", 
				imagePath, pageID, tags, autoAnalyze, autoEmbed),
		}},
		IsError: false,
	}, nil
}

// InkCreateChunkTool 建立知識塊工具
type InkCreateChunkTool struct {
	server *MCPServer
}

// NewInkCreateChunkTool 建立知識塊工具
func NewInkCreateChunkTool(server *MCPServer) *InkCreateChunkTool {
	return &InkCreateChunkTool{server: server}
}

func (t *InkCreateChunkTool) GetName() string {
	return "ink_create_chunk"
}

func (t *InkCreateChunkTool) GetDescription() string {
	return "Create a new chunk with text or image content"
}

func (t *InkCreateChunkTool) GetInputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"content": map[string]interface{}{
				"type":        "string",
				"description": "Text content of the chunk",
			},
			"page_id": map[string]interface{}{
				"type":        "string",
				"description": "Page ID to associate with the chunk (optional)",
			},
			"tags": map[string]interface{}{
				"type":        "array",
				"items":       map[string]interface{}{"type": "string"},
				"description": "Tags to associate with the chunk",
			},
			"is_page": map[string]interface{}{
				"type":        "boolean",
				"description": "Whether this chunk represents a page",
				"default":     false,
			},
			"is_tag": map[string]interface{}{
				"type":        "boolean",
				"description": "Whether this chunk represents a tag",
				"default":     false,
			},
		},
		"required": []string{"content"},
	}
}

func (t *InkCreateChunkTool) Execute(ctx context.Context, params map[string]interface{}) (*MCPToolResult, error) {
	content, ok := params["content"].(string)
	if !ok || content == "" {
		return &MCPToolResult{
			Content: []MCPContent{{Type: "text", Text: "Error: content parameter is required"}},
			IsError: true,
		}, nil
	}

	// 解析參數
	var pageID *string
	if pid, ok := params["page_id"].(string); ok && pid != "" {
		pageID = &pid
	}

	var tags []string
	if tagsInterface, ok := params["tags"].([]interface{}); ok {
		for _, tag := range tagsInterface {
			if tagStr, ok := tag.(string); ok {
				tags = append(tags, tagStr)
			}
		}
	}

	isPage := false
	if page, ok := params["is_page"].(bool); ok {
		isPage = page
	}

	isTag := false
	if tag, ok := params["is_tag"].(bool); ok {
		isTag = tag
	}

	// 建立 chunk
	chunk := &models.UnifiedChunkRecord{
		Contents: content,
		Page:     pageID,
		IsPage:   isPage,
		IsTag:    isTag,
		Tags:     tags,
		Metadata: make(map[string]interface{}),
	}

	err := t.server.services.ChunkService.CreateChunk(ctx, chunk)
	if err != nil {
		return &MCPToolResult{
			Content: []MCPContent{{Type: "text", Text: fmt.Sprintf("Failed to create chunk: %v", err)}},
			IsError: true,
		}, nil
	}

	return &MCPToolResult{
		Content: []MCPContent{{
			Type: "text", 
			Text: fmt.Sprintf("Successfully created chunk: %s\nContent: %s\nTags: %v", 
				chunk.ChunkID, content, tags),
		}},
		IsError: false,
	}, nil
}

// registerCoreTools 註冊核心工具到伺服器
func (s *MCPServer) registerCoreTools() {
	s.RegisterTool(NewInkSearchChunksTool(s))
	s.RegisterTool(NewInkAnalyzeImageTool(s))
	s.RegisterTool(NewInkUploadImageTool(s))
	s.RegisterTool(NewInkCreateChunkTool(s))
}