package mcp

import (
	"context"
	"fmt"
	"strings"

	"semantic-text-processor/models"
)

// InkSearchTextTool 文字搜尋工具（純文字內容搜尋）
type InkSearchTextTool struct {
	server *MCPServer
}

// NewInkSearchTextTool 建立文字搜尋工具
func NewInkSearchTextTool(server *MCPServer) *InkSearchTextTool {
	return &InkSearchTextTool{server: server}
}

func (t *InkSearchTextTool) GetName() string {
	return "ink_search_text"
}

func (t *InkSearchTextTool) GetDescription() string {
	return "Search for text chunks by content. Finds chunks containing specific text or matching search criteria."
}

func (t *InkSearchTextTool) GetInputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"query": map[string]interface{}{
				"type":        "string",
				"description": "Search query text - the content to search for",
			},
			"tags": map[string]interface{}{
				"type":        "array",
				"items":       map[string]interface{}{"type": "string"},
				"description": "Filter by tags (optional)",
			},
			"is_page": map[string]interface{}{
				"type":        "boolean",
				"description": "Filter by page chunks only (optional)",
			},
			"limit": map[string]interface{}{
				"type":        "integer",
				"description": "Maximum number of results to return (default: 10)",
				"default":     10,
				"minimum":     1,
				"maximum":     100,
			},
		},
		"required": []string{"query"},
	}
}

func (t *InkSearchTextTool) Execute(ctx context.Context, params map[string]interface{}) (*MCPToolResult, error) {
	// 檢查服務是否可用
	if t.server.services.ChunkService == nil {
		return &MCPToolResult{
			Content: []MCPContent{{Type: "text", Text: "Error: Chunk service is not available"}},
			IsError: true,
		}, nil
	}

	// 解析參數
	query, ok := params["query"].(string)
	if !ok || query == "" {
		return &MCPToolResult{
			Content: []MCPContent{{Type: "text", Text: "Error: query parameter is required"}},
			IsError: true,
		}, nil
	}

	limit := 10
	if limitFloat, ok := params["limit"].(float64); ok {
		limit = int(limitFloat)
	}

	// 建立搜尋請求
	searchQuery := &models.SearchQuery{
		Content: query,
		Limit:   limit,
	}

	// 處理可選參數
	if tagsInterface, ok := params["tags"].([]interface{}); ok {
		var tags []string
		for _, tag := range tagsInterface {
			if tagStr, ok := tag.(string); ok {
				tags = append(tags, tagStr)
			}
		}
		if len(tags) > 0 {
			searchQuery.Tags = tags
			searchQuery.TagLogic = "OR" // 預設使用 OR 邏輯
		}
	}

	if isPage, ok := params["is_page"].(bool); ok {
		searchQuery.IsPage = &isPage
	}

	// 執行搜尋
	searchResult, err := t.server.services.ChunkService.SearchChunks(ctx, searchQuery)
	if err != nil {
		return &MCPToolResult{
			Content: []MCPContent{{Type: "text", Text: fmt.Sprintf("Search failed: %v", err)}},
			IsError: true,
		}, nil
	}

	// 格式化結果
	var resultText strings.Builder
	resultText.WriteString(fmt.Sprintf("Found %d results (total: %d):\n\n",
		len(searchResult.Chunks), searchResult.TotalCount))

	for i, chunk := range searchResult.Chunks {
		resultText.WriteString(fmt.Sprintf("**Result %d**\n", i+1))
		resultText.WriteString(fmt.Sprintf("Chunk ID: %s\n", chunk.ChunkID))

		// 顯示內容（限制長度）
		content := chunk.Contents
		if len(content) > 500 {
			content = content[:500] + "..."
		}
		resultText.WriteString(fmt.Sprintf("Content: %s\n", content))

		// 顯示頁面信息
		if chunk.Page != nil {
			resultText.WriteString(fmt.Sprintf("Page: %s\n", *chunk.Page))
		}

		resultText.WriteString(fmt.Sprintf("Is Page: %v\n", chunk.IsPage))

		// 顯示建立時間
		resultText.WriteString(fmt.Sprintf("Created: %v\n", chunk.CreatedTime))

		resultText.WriteString("\n")
	}

	if len(searchResult.Chunks) == 0 {
		resultText.WriteString("No results found. Try adjusting your query.\n")
	}

	return &MCPToolResult{
		Content: []MCPContent{{Type: "text", Text: resultText.String()}},
		IsError: false,
	}, nil
}

// InkCreateTextChunkTool 建立文字 chunk 工具
type InkCreateTextChunkTool struct {
	server *MCPServer
}

// NewInkCreateTextChunkTool 建立文字 chunk 工具
func NewInkCreateTextChunkTool(server *MCPServer) *InkCreateTextChunkTool {
	return &InkCreateTextChunkTool{server: server}
}

func (t *InkCreateTextChunkTool) GetName() string {
	return "ink_create_text_chunk"
}

func (t *InkCreateTextChunkTool) GetDescription() string {
	return "Create a new text chunk in the knowledge base"
}

func (t *InkCreateTextChunkTool) GetInputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"content": map[string]interface{}{
				"type":        "string",
				"description": "Text content to store",
			},
			"page": map[string]interface{}{
				"type":        "string",
				"description": "Optional page ID to associate with this chunk",
			},
			"parent": map[string]interface{}{
				"type":        "string",
				"description": "Optional parent chunk ID",
			},
			"is_page": map[string]interface{}{
				"type":        "boolean",
				"description": "Whether this chunk represents a full page",
				"default":     false,
			},
		},
		"required": []string{"content"},
	}
}

func (t *InkCreateTextChunkTool) Execute(ctx context.Context, params map[string]interface{}) (*MCPToolResult, error) {
	// 檢查服務是否可用
	if t.server.services.ChunkService == nil {
		return &MCPToolResult{
			Content: []MCPContent{{Type: "text", Text: "Error: Chunk service is not available"}},
			IsError: true,
		}, nil
	}

	// 解析參數
	content, ok := params["content"].(string)
	if !ok || content == "" {
		return &MCPToolResult{
			Content: []MCPContent{{Type: "text", Text: "Error: content parameter is required"}},
			IsError: true,
		}, nil
	}

	// 建立 chunk 記錄
	chunk := &models.UnifiedChunkRecord{
		Contents: content,
		IsPage:   false,
	}

	// 處理可選參數
	if page, ok := params["page"].(string); ok && page != "" {
		chunk.Page = &page
	}

	if parent, ok := params["parent"].(string); ok && parent != "" {
		chunk.Parent = &parent
	}

	if isPage, ok := params["is_page"].(bool); ok {
		chunk.IsPage = isPage
	}

	// 建立 chunk
	err := t.server.services.ChunkService.CreateChunk(ctx, chunk)
	if err != nil {
		return &MCPToolResult{
			Content: []MCPContent{{Type: "text", Text: fmt.Sprintf("Failed to create chunk: %v", err)}},
			IsError: true,
		}, nil
	}

	// 格式化結果
	var resultText strings.Builder
	resultText.WriteString("✅ Chunk created successfully!\n\n")
	resultText.WriteString(fmt.Sprintf("Chunk ID: %s\n", chunk.ChunkID))

	if chunk.Page != nil {
		resultText.WriteString(fmt.Sprintf("Page: %s\n", *chunk.Page))
	}

	if chunk.Parent != nil {
		resultText.WriteString(fmt.Sprintf("Parent: %s\n", *chunk.Parent))
	}

	resultText.WriteString(fmt.Sprintf("Is Page: %v\n", chunk.IsPage))
	resultText.WriteString(fmt.Sprintf("Created: %v\n", chunk.CreatedTime))

	return &MCPToolResult{
		Content: []MCPContent{{Type: "text", Text: resultText.String()}},
		IsError: false,
	}, nil
}

// InkGetChunkTool 取得特定 chunk 工具
type InkGetChunkTool struct {
	server *MCPServer
}

// NewInkGetChunkTool 建立取得 chunk 工具
func NewInkGetChunkTool(server *MCPServer) *InkGetChunkTool {
	return &InkGetChunkTool{server: server}
}

func (t *InkGetChunkTool) GetName() string {
	return "ink_get_chunk"
}

func (t *InkGetChunkTool) GetDescription() string {
	return "Get detailed information about a specific chunk by its ID"
}

func (t *InkGetChunkTool) GetInputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"chunk_id": map[string]interface{}{
				"type":        "string",
				"description": "The unique ID of the chunk to retrieve",
			},
		},
		"required": []string{"chunk_id"},
	}
}

func (t *InkGetChunkTool) Execute(ctx context.Context, params map[string]interface{}) (*MCPToolResult, error) {
	// 檢查服務是否可用
	if t.server.services.ChunkService == nil {
		return &MCPToolResult{
			Content: []MCPContent{{Type: "text", Text: "Error: Chunk service is not available"}},
			IsError: true,
		}, nil
	}

	// 解析參數
	chunkID, ok := params["chunk_id"].(string)
	if !ok || chunkID == "" {
		return &MCPToolResult{
			Content: []MCPContent{{Type: "text", Text: "Error: chunk_id parameter is required"}},
			IsError: true,
		}, nil
	}

	// 取得 chunk
	chunk, err := t.server.services.ChunkService.GetChunk(ctx, chunkID)
	if err != nil {
		return &MCPToolResult{
			Content: []MCPContent{{Type: "text", Text: fmt.Sprintf("Failed to get chunk: %v", err)}},
			IsError: true,
		}, nil
	}

	// 格式化結果
	var resultText strings.Builder
	resultText.WriteString(fmt.Sprintf("**Chunk Details**\n\n"))
	resultText.WriteString(fmt.Sprintf("Chunk ID: %s\n", chunk.ChunkID))

	if chunk.Page != nil {
		resultText.WriteString(fmt.Sprintf("Page: %s\n", *chunk.Page))
	}

	if chunk.Parent != nil {
		resultText.WriteString(fmt.Sprintf("Parent: %s\n", *chunk.Parent))
	}

	resultText.WriteString(fmt.Sprintf("Is Page: %v\n", chunk.IsPage))
	resultText.WriteString(fmt.Sprintf("Is Tag: %v\n", chunk.IsTag))
	resultText.WriteString(fmt.Sprintf("Is Template: %v\n", chunk.IsTemplate))
	resultText.WriteString(fmt.Sprintf("Is Slot: %v\n", chunk.IsSlot))
	resultText.WriteString(fmt.Sprintf("Created: %v\n", chunk.CreatedTime))
	resultText.WriteString(fmt.Sprintf("Updated: %v\n", chunk.LastUpdated))

	resultText.WriteString(fmt.Sprintf("\n**Content:**\n%s\n", chunk.Contents))

	return &MCPToolResult{
		Content: []MCPContent{{Type: "text", Text: resultText.String()}},
		IsError: false,
	}, nil
}
