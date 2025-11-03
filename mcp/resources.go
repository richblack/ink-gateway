package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"semantic-text-processor/models"
)

// ChunkResource 知識塊資源
type ChunkResource struct {
	chunkID string
	server  *MCPServer
}

// NewChunkResource 建立知識塊資源
func NewChunkResource(chunkID string, server *MCPServer) *ChunkResource {
	return &ChunkResource{
		chunkID: chunkID,
		server:  server,
	}
}

func (r *ChunkResource) GetURI() string {
	return fmt.Sprintf("ink://chunks/%s", r.chunkID)
}

func (r *ChunkResource) GetName() string {
	return fmt.Sprintf("Chunk %s", r.chunkID)
}

func (r *ChunkResource) GetDescription() string {
	return fmt.Sprintf("Content and metadata for chunk %s", r.chunkID)
}

func (r *ChunkResource) GetMimeType() string {
	return "application/json"
}

func (r *ChunkResource) Read(ctx context.Context) ([]byte, error) {
	// 取得 chunk 資料
	chunk, err := r.server.services.ChunkService.GetChunk(ctx, r.chunkID)
	if err != nil {
		return nil, fmt.Errorf("failed to get chunk: %w", err)
	}

	// 建立資源內容
	resource := map[string]interface{}{
		"chunk_id":     chunk.ChunkID,
		"content":      chunk.Contents,
		"is_page":      chunk.IsPage,
		"is_tag":       chunk.IsTag,
		"is_template":  chunk.IsTemplate,
		"is_slot":      chunk.IsSlot,
		"tags":         chunk.Tags,
		"metadata":     chunk.Metadata,
		"created_time": chunk.CreatedTime.Format("2006-01-02T15:04:05Z07:00"),
		"last_updated": chunk.LastUpdated.Format("2006-01-02T15:04:05Z07:00"),
	}

	if chunk.Page != nil {
		resource["page"] = *chunk.Page
	}

	if chunk.Parent != nil {
		resource["parent"] = *chunk.Parent
	}

	if chunk.Ref != nil {
		resource["ref"] = *chunk.Ref
	}

	// 如果是圖片 chunk，加入圖片特定資訊
	if chunk.IsImageChunk() {
		resource["image_url"] = chunk.GetImageURL()
		resource["image_hash"] = chunk.GetImageHash()
		
		// 加入 AI 分析結果
		if analysis, err := models.ExtractAIAnalysis(chunk.Metadata); err == nil {
			resource["ai_analysis"] = map[string]interface{}{
				"description": analysis.Description,
				"tags":        analysis.Tags,
				"model":       analysis.Model,
				"confidence":  analysis.Confidence,
				"analyzed_at": analysis.AnalyzedAt.Format("2006-01-02T15:04:05Z07:00"),
			}
		}
	}

	// 如果有向量資訊，加入向量 metadata
	if chunk.HasVector() {
		resource["vector_info"] = map[string]interface{}{
			"vector_type":     chunk.GetVectorType(),
			"vector_model":    chunk.GetVectorModel(),
			"vector_metadata": chunk.VectorMetadata,
			"vector_length":   len(chunk.Vector),
		}
	}

	return json.Marshal(resource)
}

// ImageResource 圖片資源
type ImageResource struct {
	chunkID string
	server  *MCPServer
}

// NewImageResource 建立圖片資源
func NewImageResource(chunkID string, server *MCPServer) *ImageResource {
	return &ImageResource{
		chunkID: chunkID,
		server:  server,
	}
}

func (r *ImageResource) GetURI() string {
	return fmt.Sprintf("ink://images/%s", r.chunkID)
}

func (r *ImageResource) GetName() string {
	return fmt.Sprintf("Image %s", r.chunkID)
}

func (r *ImageResource) GetDescription() string {
	return fmt.Sprintf("Image content and analysis for chunk %s", r.chunkID)
}

func (r *ImageResource) GetMimeType() string {
	return "application/json"
}

func (r *ImageResource) Read(ctx context.Context) ([]byte, error) {
	// 取得 chunk 資料
	chunk, err := r.server.services.ChunkService.GetChunk(ctx, r.chunkID)
	if err != nil {
		return nil, fmt.Errorf("failed to get chunk: %w", err)
	}

	// 檢查是否為圖片 chunk
	if !chunk.IsImageChunk() {
		return nil, fmt.Errorf("chunk %s is not an image chunk", r.chunkID)
	}

	// 建立圖片資源內容
	resource := map[string]interface{}{
		"chunk_id":   chunk.ChunkID,
		"image_url":  chunk.GetImageURL(),
		"image_hash": chunk.GetImageHash(),
	}

	// 提取儲存資訊
	if storageInfo, err := models.ExtractStorageInfo(chunk.Metadata); err == nil {
		resource["storage"] = map[string]interface{}{
			"storage_type":      string(storageInfo.StorageType),
			"storage_id":        storageInfo.StorageID,
			"original_filename": storageInfo.OriginalFilename,
		}
	}

	// 提取圖片屬性
	if chunk.Metadata != nil {
		if imageProps, ok := chunk.Metadata["image_properties"].(map[string]interface{}); ok {
			resource["properties"] = imageProps
		}
	}

	// 提取 AI 分析結果
	if analysis, err := models.ExtractAIAnalysis(chunk.Metadata); err == nil {
		resource["ai_analysis"] = map[string]interface{}{
			"description": analysis.Description,
			"tags":        analysis.Tags,
			"model":       analysis.Model,
			"confidence":  analysis.Confidence,
			"analyzed_at": analysis.AnalyzedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	// 向量資訊
	if chunk.HasVector() {
		resource["vector_info"] = map[string]interface{}{
			"vector_type":     chunk.GetVectorType(),
			"vector_model":    chunk.GetVectorModel(),
			"vector_metadata": chunk.VectorMetadata,
			"vector_length":   len(chunk.Vector),
		}
	}

	return json.Marshal(resource)
}

// SearchResultsResource 搜尋結果資源
type SearchResultsResource struct {
	searchID string
	results  *models.MultimodalSearchResponse
	server   *MCPServer
}

// NewSearchResultsResource 建立搜尋結果資源
func NewSearchResultsResource(searchID string, results *models.MultimodalSearchResponse, server *MCPServer) *SearchResultsResource {
	return &SearchResultsResource{
		searchID: searchID,
		results:  results,
		server:   server,
	}
}

func (r *SearchResultsResource) GetURI() string {
	return fmt.Sprintf("ink://search/%s", r.searchID)
}

func (r *SearchResultsResource) GetName() string {
	return fmt.Sprintf("Search Results %s", r.searchID)
}

func (r *SearchResultsResource) GetDescription() string {
	return fmt.Sprintf("Search results for query: %s", r.results.Query)
}

func (r *SearchResultsResource) GetMimeType() string {
	return "application/json"
}

func (r *SearchResultsResource) Read(ctx context.Context) ([]byte, error) {
	// 建立搜尋結果資源內容
	resource := map[string]interface{}{
		"search_id":    r.searchID,
		"query":        r.results.Query,
		"total_count":  r.results.TotalCount,
		"search_time":  r.results.SearchTime.String(),
		"match_types":  r.results.MatchTypes,
		"results":      make([]map[string]interface{}, 0),
	}

	// 轉換搜尋結果
	var results []map[string]interface{}
	for _, result := range r.results.Results {
		resultData := map[string]interface{}{
			"chunk_id":    result.Chunk.ChunkID,
			"content":     result.Chunk.Contents,
			"similarity":  result.Similarity,
			"match_type":  result.MatchType,
			"explanation": result.Explanation,
			"tags":        result.Chunk.Tags,
			"created_at":  result.Chunk.CreatedTime.Format("2006-01-02T15:04:05Z07:00"),
		}

		if result.Chunk.IsImageChunk() {
			resultData["image_url"] = result.Chunk.GetImageURL()
			resultData["image_hash"] = result.Chunk.GetImageHash()
		}

		results = append(results, resultData)
	}

	resource["results"] = results

	return json.Marshal(resource)
}

// InkSearchPrompt 搜尋提示
type InkSearchPrompt struct {
	server *MCPServer
}

// NewInkSearchPrompt 建立搜尋提示
func NewInkSearchPrompt(server *MCPServer) *InkSearchPrompt {
	return &InkSearchPrompt{server: server}
}

func (p *InkSearchPrompt) GetName() string {
	return "ink_search_assistant"
}

func (p *InkSearchPrompt) GetDescription() string {
	return "Assistant prompt for helping users search their knowledge base effectively"
}

func (p *InkSearchPrompt) GetArguments() []MCPPromptArgument {
	return []MCPPromptArgument{
		{
			Name:        "search_context",
			Description: "Context about what the user is looking for",
			Required:    true,
		},
		{
			Name:        "content_type",
			Description: "Type of content to search for (text, image, or both)",
			Required:    false,
		},
	}
}

func (p *InkSearchPrompt) Generate(ctx context.Context, args map[string]interface{}) (string, error) {
	searchContext, ok := args["search_context"].(string)
	if !ok || searchContext == "" {
		return "", fmt.Errorf("search_context argument is required")
	}

	contentType, _ := args["content_type"].(string)
	if contentType == "" {
		contentType = "both"
	}

	var prompt strings.Builder
	prompt.WriteString("I'm here to help you search your knowledge base effectively. ")
	prompt.WriteString(fmt.Sprintf("You're looking for: %s\n\n", searchContext))

	switch contentType {
	case "text":
		prompt.WriteString("I'll focus on searching text content and descriptions. ")
		prompt.WriteString("I can help you find relevant notes, documents, and text-based information.")
	case "image":
		prompt.WriteString("I'll focus on searching images and visual content. ")
		prompt.WriteString("I can help you find relevant images, diagrams, screenshots, and visual materials based on their AI-generated descriptions and tags.")
	case "both":
		prompt.WriteString("I'll search both text and image content to give you comprehensive results. ")
		prompt.WriteString("I can find relevant text content, images, and mixed media that match your needs.")
	}

	prompt.WriteString("\n\nI can use the following search capabilities:\n")
	prompt.WriteString("- **Text search**: Find content based on text similarity\n")
	prompt.WriteString("- **Image search**: Find images based on visual similarity or AI descriptions\n")
	prompt.WriteString("- **Hybrid search**: Combine text and image search with custom weights\n")
	prompt.WriteString("- **Similar images**: Find images similar to a reference image\n")
	prompt.WriteString("- **Slide recommendations**: Get image suggestions for presentation slides\n\n")

	prompt.WriteString("What specific information would you like me to search for?")

	return prompt.String(), nil
}

// InkImageAnalysisPrompt 圖片分析提示
type InkImageAnalysisPrompt struct {
	server *MCPServer
}

// NewInkImageAnalysisPrompt 建立圖片分析提示
func NewInkImageAnalysisPrompt(server *MCPServer) *InkImageAnalysisPrompt {
	return &InkImageAnalysisPrompt{server: server}
}

func (p *InkImageAnalysisPrompt) GetName() string {
	return "ink_image_analysis_assistant"
}

func (p *InkImageAnalysisPrompt) GetDescription() string {
	return "Assistant prompt for helping users analyze and understand images"
}

func (p *InkImageAnalysisPrompt) GetArguments() []MCPPromptArgument {
	return []MCPPromptArgument{
		{
			Name:        "analysis_purpose",
			Description: "Purpose of the image analysis (documentation, categorization, search, etc.)",
			Required:    true,
		},
		{
			Name:        "detail_level",
			Description: "Level of detail needed (low, medium, high)",
			Required:    false,
		},
	}
}

func (p *InkImageAnalysisPrompt) Generate(ctx context.Context, args map[string]interface{}) (string, error) {
	analysisPurpose, ok := args["analysis_purpose"].(string)
	if !ok || analysisPurpose == "" {
		return "", fmt.Errorf("analysis_purpose argument is required")
	}

	detailLevel, _ := args["detail_level"].(string)
	if detailLevel == "" {
		detailLevel = "medium"
	}

	var prompt strings.Builder
	prompt.WriteString("I'm here to help you analyze and understand images in your knowledge base. ")
	prompt.WriteString(fmt.Sprintf("Your analysis purpose: %s\n\n", analysisPurpose))

	switch detailLevel {
	case "low":
		prompt.WriteString("I'll provide a basic analysis focusing on the main elements and overall content.")
	case "high":
		prompt.WriteString("I'll provide a detailed analysis including fine details, text content, technical elements, and comprehensive descriptions.")
	default:
		prompt.WriteString("I'll provide a balanced analysis covering the main content, key elements, and relevant details.")
	}

	prompt.WriteString("\n\nI can help you with:\n")
	prompt.WriteString("- **Content Analysis**: Describe what's in the image\n")
	prompt.WriteString("- **Technical Analysis**: Identify technical elements, diagrams, code, etc.\n")
	prompt.WriteString("- **Categorization**: Suggest appropriate tags and categories\n")
	prompt.WriteString("- **Context Understanding**: Explain the purpose and use case\n")
	prompt.WriteString("- **Search Optimization**: Generate descriptions that improve searchability\n\n")

	switch analysisPurpose {
	case "documentation":
		prompt.WriteString("For documentation purposes, I'll focus on creating clear, descriptive content that helps with organization and retrieval.")
	case "categorization":
		prompt.WriteString("For categorization, I'll focus on identifying key characteristics and suggesting relevant tags and categories.")
	case "search":
		prompt.WriteString("For search optimization, I'll focus on generating comprehensive descriptions with relevant keywords.")
	default:
		prompt.WriteString("I'll provide a comprehensive analysis suitable for your specific needs.")
	}

	prompt.WriteString("\n\nPlease provide the image you'd like me to analyze.")

	return prompt.String(), nil
}

// registerResources 註冊資源到伺服器
func (s *MCPServer) registerResources() {
	// 資源會動態註冊，這裡提供註冊方法
	log.Printf("Resource registration system ready")
}

// registerPrompts 註冊提示到伺服器
func (s *MCPServer) registerPrompts() {
	s.RegisterPrompt(NewInkSearchPrompt(s))
	s.RegisterPrompt(NewInkImageAnalysisPrompt(s))
}

// CreateChunkResource 建立並註冊知識塊資源
func (s *MCPServer) CreateChunkResource(chunkID string) {
	resource := NewChunkResource(chunkID, s)
	s.RegisterResource(resource)
}

// CreateImageResource 建立並註冊圖片資源
func (s *MCPServer) CreateImageResource(chunkID string) {
	resource := NewImageResource(chunkID, s)
	s.RegisterResource(resource)
}

// CreateSearchResultsResource 建立並註冊搜尋結果資源
func (s *MCPServer) CreateSearchResultsResource(searchID string, results *models.MultimodalSearchResponse) {
	resource := NewSearchResultsResource(searchID, results, s)
	s.RegisterResource(resource)
}