package mcp

import (
	"context"
	"fmt"
	"strings"

	"semantic-text-processor/models"
	"semantic-text-processor/services"
)

// InkBatchProcessImagesTool 批次處理圖片工具
type InkBatchProcessImagesTool struct {
	server *MCPServer
}

// NewInkBatchProcessImagesTool 建立批次處理圖片工具
func NewInkBatchProcessImagesTool(server *MCPServer) *InkBatchProcessImagesTool {
	return &InkBatchProcessImagesTool{server: server}
}

func (t *InkBatchProcessImagesTool) GetName() string {
	return "ink_batch_process_images"
}

func (t *InkBatchProcessImagesTool) GetDescription() string {
	return "Start batch processing of images in a folder"
}

func (t *InkBatchProcessImagesTool) GetInputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"folder_path": map[string]interface{}{
				"type":        "string",
				"description": "Path to the folder containing images",
			},
			"page_id": map[string]interface{}{
				"type":        "string",
				"description": "Page ID to associate with all images (optional)",
			},
			"tags": map[string]interface{}{
				"type":        "array",
				"items":       map[string]interface{}{"type": "string"},
				"description": "Tags to associate with all images",
			},
			"auto_analyze": map[string]interface{}{
				"type":        "boolean",
				"description": "Whether to automatically analyze images",
				"default":     true,
			},
			"auto_embed": map[string]interface{}{
				"type":        "boolean",
				"description": "Whether to automatically generate embeddings",
				"default":     true,
			},
			"concurrency": map[string]interface{}{
				"type":        "integer",
				"description": "Number of concurrent processing threads",
				"default":     3,
			},
		},
		"required": []string{"folder_path"},
	}
}

func (t *InkBatchProcessImagesTool) Execute(ctx context.Context, params map[string]interface{}) (*MCPToolResult, error) {
	folderPath, ok := params["folder_path"].(string)
	if !ok || folderPath == "" {
		return &MCPToolResult{
			Content: []MCPContent{{Type: "text", Text: "Error: folder_path parameter is required"}},
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

	concurrency := 3
	if conc, ok := params["concurrency"].(float64); ok {
		concurrency = int(conc)
	}

	// 掃描資料夾
	folderScanner := services.NewFolderScanner()
	files, err := folderScanner.ScanFolder(ctx, folderPath)
	if err != nil {
		return &MCPToolResult{
			Content: []MCPContent{{Type: "text", Text: fmt.Sprintf("Failed to scan folder: %v", err)}},
			IsError: true,
		}, nil
	}

	if len(files) == 0 {
		return &MCPToolResult{
			Content: []MCPContent{{Type: "text", Text: "No image files found in the specified folder"}},
			IsError: false,
		}, nil
	}

	// 轉換 MediaFile 為檔案路徑
	filePaths := make([]string, len(files))
	for i, file := range files {
		filePaths[i] = file.Path
	}

	// 建立批次處理請求
	batchReq := &models.BatchProcessRequest{
		Files:       filePaths,
		PageID:      pageID,
		Tags:        tags,
		AutoAnalyze: autoAnalyze,
		AutoEmbed:   autoEmbed,
		StorageType: models.StorageTypeSupabase,
		Concurrency: concurrency,
	}

	// 開始批次處理
	batchJob, err := t.server.services.BatchProcessor.StartBatchProcess(ctx, batchReq)
	if err != nil {
		return &MCPToolResult{
			Content: []MCPContent{{Type: "text", Text: fmt.Sprintf("Failed to start batch processing: %v", err)}},
			IsError: true,
		}, nil
	}

	return &MCPToolResult{
		Content: []MCPContent{{
			Type: "text",
			Text: fmt.Sprintf("Batch processing started successfully!\n\nBatch ID: %s\nTotal Files: %d\nStatus: %s\n\nYou can check the progress using the batch ID.",
				batchJob.ID, len(files), batchJob.Status.Status),
		}},
		IsError: false,
	}, nil
}

// InkGetImagesForSlidesTool Slide Generator 圖片推薦工具
type InkGetImagesForSlidesTool struct {
	server *MCPServer
}

// NewInkGetImagesForSlidesTool 建立 Slide Generator 圖片推薦工具
func NewInkGetImagesForSlidesTool(server *MCPServer) *InkGetImagesForSlidesTool {
	return &InkGetImagesForSlidesTool{server: server}
}

func (t *InkGetImagesForSlidesTool) GetName() string {
	return "ink_get_images_for_slides"
}

func (t *InkGetImagesForSlidesTool) GetDescription() string {
	return "Get image recommendations for slide content"
}

func (t *InkGetImagesForSlidesTool) GetInputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"slide_title": map[string]interface{}{
				"type":        "string",
				"description": "Title of the slide",
			},
			"slide_content": map[string]interface{}{
				"type":        "string",
				"description": "Content of the slide",
			},
			"slide_context": map[string]interface{}{
				"type":        "string",
				"description": "Additional context about the presentation",
			},
			"max_suggestions": map[string]interface{}{
				"type":        "integer",
				"description": "Maximum number of image suggestions",
				"default":     5,
			},
			"min_relevance": map[string]interface{}{
				"type":        "number",
				"description": "Minimum relevance score for suggestions",
				"default":     0.6,
			},
		},
		"required": []string{"slide_content"},
	}
}

func (t *InkGetImagesForSlidesTool) Execute(ctx context.Context, params map[string]interface{}) (*MCPToolResult, error) {
	slideContent, ok := params["slide_content"].(string)
	if !ok || slideContent == "" {
		return &MCPToolResult{
			Content: []MCPContent{{Type: "text", Text: "Error: slide_content parameter is required"}},
			IsError: true,
		}, nil
	}

	// 解析參數
	slideTitle, _ := params["slide_title"].(string)
	slideContext, _ := params["slide_context"].(string)

	maxSuggestions := 5
	if max, ok := params["max_suggestions"].(float64); ok {
		maxSuggestions = int(max)
	}

	minRelevance := 0.6
	if min, ok := params["min_relevance"].(float64); ok {
		minRelevance = min
	}

	// 建立推薦請求
	req := &services.SlideImageRecommendationRequest{
		SlideTitle:     slideTitle,
		SlideContent:   slideContent,
		SlideContext:   slideContext,
		MaxSuggestions: maxSuggestions,
		MinRelevance:   minRelevance,
	}

	// 執行推薦
	response, err := t.server.services.SlideRecommendation.RecommendImagesForSlide(ctx, req)
	if err != nil {
		return &MCPToolResult{
			Content: []MCPContent{{Type: "text", Text: fmt.Sprintf("Failed to get image recommendations: %v", err)}},
			IsError: true,
		}, nil
	}

	// 格式化結果
	var resultText strings.Builder
	resultText.WriteString(fmt.Sprintf("**Image Recommendations for Slide**\n\n"))
	if slideTitle != "" {
		resultText.WriteString(fmt.Sprintf("**Slide Title:** %s\n", slideTitle))
	}
	resultText.WriteString(fmt.Sprintf("**Found %d recommendations:**\n\n", len(response.Recommendations)))

	for i, rec := range response.Recommendations {
		resultText.WriteString(fmt.Sprintf("%d. **%s** (relevance: %.3f)\n", i+1, rec.Title, rec.RelevanceScore))
		resultText.WriteString(fmt.Sprintf("   Image URL: %s\n", rec.ImageURL))
		resultText.WriteString(fmt.Sprintf("   Description: %s\n", rec.Description))
		resultText.WriteString(fmt.Sprintf("   Reason: %s\n", rec.MatchReason))
		if len(rec.Tags) > 0 {
			resultText.WriteString(fmt.Sprintf("   Tags: %s\n", strings.Join(rec.Tags, ", ")))
		}
		resultText.WriteString("\n")
	}

	return &MCPToolResult{
		Content: []MCPContent{{Type: "text", Text: resultText.String()}},
		IsError: false,
	}, nil
}

// InkSearchImagesTool 圖片搜尋工具
type InkSearchImagesTool struct {
	server *MCPServer
}

// NewInkSearchImagesTool 建立圖片搜尋工具
func NewInkSearchImagesTool(server *MCPServer) *InkSearchImagesTool {
	return &InkSearchImagesTool{server: server}
}

func (t *InkSearchImagesTool) GetName() string {
	return "ink_search_images"
}

func (t *InkSearchImagesTool) GetDescription() string {
	return "Search for similar images using image similarity search"
}

func (t *InkSearchImagesTool) GetInputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"image_url": map[string]interface{}{
				"type":        "string",
				"description": "URL of the reference image",
			},
			"chunk_id": map[string]interface{}{
				"type":        "string",
				"description": "Chunk ID of the reference image (alternative to image_url)",
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
			"include_metadata": map[string]interface{}{
				"type":        "boolean",
				"description": "Whether to include image metadata",
				"default":     true,
			},
		},
		"oneOf": []map[string]interface{}{
			{"required": []string{"image_url"}},
			{"required": []string{"chunk_id"}},
		},
	}
}

func (t *InkSearchImagesTool) Execute(ctx context.Context, params map[string]interface{}) (*MCPToolResult, error) {
	imageURL, hasURL := params["image_url"].(string)
	chunkID, hasChunkID := params["chunk_id"].(string)

	if (!hasURL || imageURL == "") && (!hasChunkID || chunkID == "") {
		return &MCPToolResult{
			Content: []MCPContent{{Type: "text", Text: "Error: either image_url or chunk_id parameter is required"}},
			IsError: true,
		}, nil
	}

	// 解析參數
	limit := 10
	if l, ok := params["limit"].(float64); ok {
		limit = int(l)
	}

	minSimilarity := 0.7
	if min, ok := params["min_similarity"].(float64); ok {
		minSimilarity = min
	}

	includeMetadata := true
	if include, ok := params["include_metadata"].(bool); ok {
		includeMetadata = include
	}

	// 建立搜尋選項
	options := &services.ImageSearchOptions{
		Limit:               limit,
		SimilarityThreshold: minSimilarity,
		IncludeMetadata:     includeMetadata,
		SortBy:              "similarity",
		SortOrder:           "desc",
	}

	// 執行搜尋
	var searchResponse *services.ImageSearchResponse
	var err error

	if hasChunkID && chunkID != "" {
		searchResponse, err = t.server.services.ImageSimilarity.SearchByChunkID(ctx, chunkID, options)
	} else {
		searchResponse, err = t.server.services.ImageSimilarity.SearchByImageURL(ctx, imageURL, options)
	}

	if err != nil {
		return &MCPToolResult{
			Content: []MCPContent{{Type: "text", Text: fmt.Sprintf("Image search failed: %v", err)}},
			IsError: true,
		}, nil
	}

	// 格式化結果
	var resultText strings.Builder
	resultText.WriteString(fmt.Sprintf("**Similar Images Search Results**\n\n"))
	resultText.WriteString(fmt.Sprintf("Query: %s\n", searchResponse.QuerySource))
	resultText.WriteString(fmt.Sprintf("Found %d similar images in %s:\n\n", 
		searchResponse.TotalCount, searchResponse.SearchTime.String()))

	for i, result := range searchResponse.Results {
		resultText.WriteString(fmt.Sprintf("%d. **%s** (similarity: %.3f)\n", 
			i+1, result.ChunkID, result.Similarity))
		resultText.WriteString(fmt.Sprintf("   Image URL: %s\n", result.ImageURL))
		resultText.WriteString(fmt.Sprintf("   Description: %s\n", result.Description))
		if len(result.Tags) > 0 {
			resultText.WriteString(fmt.Sprintf("   Tags: %s\n", strings.Join(result.Tags, ", ")))
		}
		resultText.WriteString(fmt.Sprintf("   Created: %s\n", result.CreatedAt.Format("2006-01-02 15:04:05")))
		resultText.WriteString("\n")
	}

	return &MCPToolResult{
		Content: []MCPContent{{Type: "text", Text: resultText.String()}},
		IsError: false,
	}, nil
}

// InkHybridSearchTool 混合搜尋工具
type InkHybridSearchTool struct {
	server *MCPServer
}

// NewInkHybridSearchTool 建立混合搜尋工具
func NewInkHybridSearchTool(server *MCPServer) *InkHybridSearchTool {
	return &InkHybridSearchTool{server: server}
}

func (t *InkHybridSearchTool) GetName() string {
	return "ink_hybrid_search"
}

func (t *InkHybridSearchTool) GetDescription() string {
	return "Perform hybrid search combining text and image queries with custom weights"
}

func (t *InkHybridSearchTool) GetInputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"text_query": map[string]interface{}{
				"type":        "string",
				"description": "Text search query",
			},
			"image_query": map[string]interface{}{
				"type":        "string",
				"description": "Image URL for image-based search",
			},
			"text_weight": map[string]interface{}{
				"type":        "number",
				"description": "Weight for text search results (0.0-1.0)",
				"default":     0.6,
			},
			"image_weight": map[string]interface{}{
				"type":        "number",
				"description": "Weight for image search results (0.0-1.0)",
				"default":     0.4,
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
		"anyOf": []map[string]interface{}{
			{"required": []string{"text_query"}},
			{"required": []string{"image_query"}},
		},
	}
}

func (t *InkHybridSearchTool) Execute(ctx context.Context, params map[string]interface{}) (*MCPToolResult, error) {
	textQuery, _ := params["text_query"].(string)
	imageQuery, _ := params["image_query"].(string)

	if textQuery == "" && imageQuery == "" {
		return &MCPToolResult{
			Content: []MCPContent{{Type: "text", Text: "Error: at least one of text_query or image_query is required"}},
			IsError: true,
		}, nil
	}

	// 解析參數
	textWeight := 0.6
	if tw, ok := params["text_weight"].(float64); ok {
		textWeight = tw
	}

	imageWeight := 0.4
	if iw, ok := params["image_weight"].(float64); ok {
		imageWeight = iw
	}

	limit := 10
	if l, ok := params["limit"].(float64); ok {
		limit = int(l)
	}

	minSimilarity := 0.7
	if min, ok := params["min_similarity"].(float64); ok {
		minSimilarity = min
	}

	// 建立搜尋請求
	searchReq := &models.MultimodalSearchRequest{
		TextQuery:  textQuery,
		ImageQuery: imageQuery,
		VectorType: "all",
		Weights: &models.SearchWeights{
			Text:  textWeight,
			Image: imageWeight,
		},
		Limit:         limit,
		MinSimilarity: minSimilarity,
	}

	// 執行混合搜尋
	searchResponse, err := t.server.services.MultimodalSearch.HybridSearch(ctx, searchReq)
	if err != nil {
		return &MCPToolResult{
			Content: []MCPContent{{Type: "text", Text: fmt.Sprintf("Hybrid search failed: %v", err)}},
			IsError: true,
		}, nil
	}

	// 格式化結果
	var resultText strings.Builder
	resultText.WriteString(fmt.Sprintf("**Hybrid Search Results**\n\n"))
	resultText.WriteString(fmt.Sprintf("Text Query: %s\n", textQuery))
	resultText.WriteString(fmt.Sprintf("Image Query: %s\n", imageQuery))
	resultText.WriteString(fmt.Sprintf("Weights: Text=%.2f, Image=%.2f\n", textWeight, imageWeight))
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
		resultText.WriteString(fmt.Sprintf("   Match Type: %s\n", result.MatchType))
		resultText.WriteString(fmt.Sprintf("   Explanation: %s\n", result.Explanation))
		resultText.WriteString("\n")
	}

	return &MCPToolResult{
		Content: []MCPContent{{Type: "text", Text: resultText.String()}},
		IsError: false,
	}, nil
}

// registerAdvancedTools 註冊進階工具到伺服器
func (s *MCPServer) registerAdvancedTools() {
	s.RegisterTool(NewInkBatchProcessImagesTool(s))
	s.RegisterTool(NewInkGetImagesForSlidesTool(s))
	s.RegisterTool(NewInkSearchImagesTool(s))
	s.RegisterTool(NewInkHybridSearchTool(s))
}