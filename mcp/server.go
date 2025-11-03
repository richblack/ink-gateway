package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"sync"

	"semantic-text-processor/services"
)

// MCPServer MCP 協議伺服器
type MCPServer struct {
	name        string
	version     string
	description string
	tools       map[string]MCPTool
	resources   map[string]MCPResource
	prompts     map[string]MCPPrompt
	services    *MCPServices
	stdin       io.Reader
	stdout      io.Writer
	stderr      io.Writer
	ctx         context.Context
	cancel      context.CancelFunc
	mu          sync.RWMutex
}

// MCPServices MCP 服務依賴
type MCPServices struct {
	MediaProcessor      services.MediaProcessor
	MultimodalSearch    services.MultimodalSearchService
	BatchProcessor      *services.BatchProcessor
	ImageSimilarity     *services.ImageSimilaritySearch
	SlideRecommendation *services.SlideImageRecommendationService
	StorageService      *services.StorageService
	ChunkService        services.UnifiedChunkService
}

// NewMCPServer 建立新的 MCP 伺服器
func NewMCPServer(name, version, description string, services *MCPServices) *MCPServer {
	ctx, cancel := context.WithCancel(context.Background())
	
	server := &MCPServer{
		name:        name,
		version:     version,
		description: description,
		tools:       make(map[string]MCPTool),
		resources:   make(map[string]MCPResource),
		prompts:     make(map[string]MCPPrompt),
		services:    services,
		stdin:       os.Stdin,
		stdout:      os.Stdout,
		stderr:      os.Stderr,
		ctx:         ctx,
		cancel:      cancel,
	}
	
	// 註冊預設工具
	server.registerDefaultTools()
	
	return server
}

// MCPMessage MCP 訊息基礎結構
type MCPMessage struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Method  string      `json:"method,omitempty"`
	Params  interface{} `json:"params,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Error   *MCPError   `json:"error,omitempty"`
}

// MCPError MCP 錯誤結構
type MCPError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// MCPTool MCP 工具介面
type MCPTool interface {
	GetName() string
	GetDescription() string
	GetInputSchema() map[string]interface{}
	Execute(ctx context.Context, params map[string]interface{}) (*MCPToolResult, error)
}

// MCPToolResult MCP 工具執行結果
type MCPToolResult struct {
	Content []MCPContent `json:"content"`
	IsError bool         `json:"isError"`
}

// MCPContent MCP 內容
type MCPContent struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
	Data string `json:"data,omitempty"`
}

// MCPResource MCP 資源介面
type MCPResource interface {
	GetURI() string
	GetName() string
	GetDescription() string
	GetMimeType() string
	Read(ctx context.Context) ([]byte, error)
}

// MCPPrompt MCP 提示介面
type MCPPrompt interface {
	GetName() string
	GetDescription() string
	GetArguments() []MCPPromptArgument
	Generate(ctx context.Context, args map[string]interface{}) (string, error)
}

// MCPPromptArgument MCP 提示參數
type MCPPromptArgument struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
}

// Start 啟動 MCP 伺服器
func (s *MCPServer) Start() error {
	log.Printf("Starting MCP Server: %s v%s", s.name, s.version)
	
	scanner := bufio.NewScanner(s.stdin)
	
	for scanner.Scan() {
		select {
		case <-s.ctx.Done():
			return s.ctx.Err()
		default:
		}
		
		line := scanner.Text()
		if line == "" {
			continue
		}
		
		if err := s.handleMessage(line); err != nil {
			log.Printf("Error handling message: %v", err)
		}
	}
	
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading from stdin: %w", err)
	}
	
	return nil
}

// Stop 停止 MCP 伺服器
func (s *MCPServer) Stop() {
	log.Printf("Stopping MCP Server: %s", s.name)
	s.cancel()
}

// handleMessage 處理 MCP 訊息
func (s *MCPServer) handleMessage(line string) error {
	var msg MCPMessage
	if err := json.Unmarshal([]byte(line), &msg); err != nil {
		return s.sendError(nil, -32700, "Parse error", err)
	}
	
	switch msg.Method {
	case "initialize":
		return s.handleInitialize(&msg)
	case "tools/list":
		return s.handleToolsList(&msg)
	case "tools/call":
		return s.handleToolsCall(&msg)
	case "resources/list":
		return s.handleResourcesList(&msg)
	case "resources/read":
		return s.handleResourcesRead(&msg)
	case "prompts/list":
		return s.handlePromptsList(&msg)
	case "prompts/get":
		return s.handlePromptsGet(&msg)
	default:
		return s.sendError(msg.ID, -32601, "Method not found", nil)
	}
}

// handleInitialize 處理初始化請求
func (s *MCPServer) handleInitialize(msg *MCPMessage) error {
	result := map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities": map[string]interface{}{
			"tools": map[string]interface{}{
				"listChanged": false,
			},
			"resources": map[string]interface{}{
				"subscribe":   false,
				"listChanged": false,
			},
			"prompts": map[string]interface{}{
				"listChanged": false,
			},
		},
		"serverInfo": map[string]interface{}{
			"name":    s.name,
			"version": s.version,
		},
	}
	
	return s.sendResult(msg.ID, result)
}

// handleToolsList 處理工具列表請求
func (s *MCPServer) handleToolsList(msg *MCPMessage) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	var tools []map[string]interface{}
	for _, tool := range s.tools {
		tools = append(tools, map[string]interface{}{
			"name":        tool.GetName(),
			"description": tool.GetDescription(),
			"inputSchema": tool.GetInputSchema(),
		})
	}
	
	result := map[string]interface{}{
		"tools": tools,
	}
	
	return s.sendResult(msg.ID, result)
}

// handleToolsCall 處理工具呼叫請求
func (s *MCPServer) handleToolsCall(msg *MCPMessage) error {
	params, ok := msg.Params.(map[string]interface{})
	if !ok {
		return s.sendError(msg.ID, -32602, "Invalid params", nil)
	}
	
	toolName, ok := params["name"].(string)
	if !ok {
		return s.sendError(msg.ID, -32602, "Missing tool name", nil)
	}
	
	arguments, ok := params["arguments"].(map[string]interface{})
	if !ok {
		arguments = make(map[string]interface{})
	}
	
	s.mu.RLock()
	tool, exists := s.tools[toolName]
	s.mu.RUnlock()
	
	if !exists {
		return s.sendError(msg.ID, -32601, "Tool not found", nil)
	}
	
	// 執行工具
	result, err := tool.Execute(s.ctx, arguments)
	if err != nil {
		return s.sendError(msg.ID, -32603, "Tool execution failed", err)
	}
	
	return s.sendResult(msg.ID, result)
}

// handleResourcesList 處理資源列表請求
func (s *MCPServer) handleResourcesList(msg *MCPMessage) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	var resources []map[string]interface{}
	for _, resource := range s.resources {
		resources = append(resources, map[string]interface{}{
			"uri":         resource.GetURI(),
			"name":        resource.GetName(),
			"description": resource.GetDescription(),
			"mimeType":    resource.GetMimeType(),
		})
	}
	
	result := map[string]interface{}{
		"resources": resources,
	}
	
	return s.sendResult(msg.ID, result)
}

// handleResourcesRead 處理資源讀取請求
func (s *MCPServer) handleResourcesRead(msg *MCPMessage) error {
	params, ok := msg.Params.(map[string]interface{})
	if !ok {
		return s.sendError(msg.ID, -32602, "Invalid params", nil)
	}
	
	uri, ok := params["uri"].(string)
	if !ok {
		return s.sendError(msg.ID, -32602, "Missing resource URI", nil)
	}
	
	s.mu.RLock()
	resource, exists := s.resources[uri]
	s.mu.RUnlock()
	
	if !exists {
		return s.sendError(msg.ID, -32601, "Resource not found", nil)
	}
	
	// 讀取資源
	data, err := resource.Read(s.ctx)
	if err != nil {
		return s.sendError(msg.ID, -32603, "Resource read failed", err)
	}
	
	result := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"uri":      resource.GetURI(),
				"mimeType": resource.GetMimeType(),
				"text":     string(data),
			},
		},
	}
	
	return s.sendResult(msg.ID, result)
}

// handlePromptsList 處理提示列表請求
func (s *MCPServer) handlePromptsList(msg *MCPMessage) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	var prompts []map[string]interface{}
	for _, prompt := range s.prompts {
		prompts = append(prompts, map[string]interface{}{
			"name":        prompt.GetName(),
			"description": prompt.GetDescription(),
			"arguments":   prompt.GetArguments(),
		})
	}
	
	result := map[string]interface{}{
		"prompts": prompts,
	}
	
	return s.sendResult(msg.ID, result)
}

// handlePromptsGet 處理提示取得請求
func (s *MCPServer) handlePromptsGet(msg *MCPMessage) error {
	params, ok := msg.Params.(map[string]interface{})
	if !ok {
		return s.sendError(msg.ID, -32602, "Invalid params", nil)
	}
	
	promptName, ok := params["name"].(string)
	if !ok {
		return s.sendError(msg.ID, -32602, "Missing prompt name", nil)
	}
	
	arguments, ok := params["arguments"].(map[string]interface{})
	if !ok {
		arguments = make(map[string]interface{})
	}
	
	s.mu.RLock()
	prompt, exists := s.prompts[promptName]
	s.mu.RUnlock()
	
	if !exists {
		return s.sendError(msg.ID, -32601, "Prompt not found", nil)
	}
	
	// 生成提示
	content, err := prompt.Generate(s.ctx, arguments)
	if err != nil {
		return s.sendError(msg.ID, -32603, "Prompt generation failed", err)
	}
	
	result := map[string]interface{}{
		"description": prompt.GetDescription(),
		"messages": []map[string]interface{}{
			{
				"role": "user",
				"content": map[string]interface{}{
					"type": "text",
					"text": content,
				},
			},
		},
	}
	
	return s.sendResult(msg.ID, result)
}

// sendResult 發送成功結果
func (s *MCPServer) sendResult(id interface{}, result interface{}) error {
	response := MCPMessage{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
	
	return s.sendMessage(response)
}

// sendError 發送錯誤回應
func (s *MCPServer) sendError(id interface{}, code int, message string, data interface{}) error {
	response := MCPMessage{
		JSONRPC: "2.0",
		ID:      id,
		Error: &MCPError{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}
	
	return s.sendMessage(response)
}

// sendMessage 發送訊息
func (s *MCPServer) sendMessage(msg MCPMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}
	
	_, err = fmt.Fprintf(s.stdout, "%s\n", data)
	return err
}

// RegisterTool 註冊工具
func (s *MCPServer) RegisterTool(tool MCPTool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tools[tool.GetName()] = tool
}

// RegisterResource 註冊資源
func (s *MCPServer) RegisterResource(resource MCPResource) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.resources[resource.GetURI()] = resource
}

// RegisterPrompt 註冊提示
func (s *MCPServer) RegisterPrompt(prompt MCPPrompt) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.prompts[prompt.GetName()] = prompt
}

// registerDefaultTools 註冊預設工具
func (s *MCPServer) registerDefaultTools() {
	log.Printf("Registering default MCP tools...")

	// 註冊文字搜尋和操作工具（只需要 ChunkService）
	if s.services.ChunkService != nil {
		s.RegisterTool(NewInkSearchTextTool(s))
		s.RegisterTool(NewInkCreateTextChunkTool(s))
		s.RegisterTool(NewInkGetChunkTool(s))
		log.Printf("Registered text tools: ink_search_text, ink_create_text_chunk, ink_get_chunk")
	} else {
		log.Printf("Warning: ChunkService not available, skipping text tools")
	}

	// 多模態工具需要額外的服務（目前尚未整合）
	if s.services.MultimodalSearch != nil {
		s.RegisterTool(NewInkSearchChunksTool(s))
		log.Printf("Registered multimodal search tool: ink_search_chunks")
	}

	if s.services.MediaProcessor != nil {
		s.RegisterTool(NewInkAnalyzeImageTool(s))
		s.RegisterTool(NewInkUploadImageTool(s))
		log.Printf("Registered image tools: ink_analyze_image, ink_upload_image")
	}

	if s.services.BatchProcessor != nil {
		s.RegisterTool(NewInkBatchProcessImagesTool(s))
		log.Printf("Registered batch processing tool: ink_batch_process_images")
	}

	if s.services.ImageSimilarity != nil {
		s.RegisterTool(NewInkSearchImagesTool(s))
		log.Printf("Registered image search tool: ink_search_images")
	}

	if s.services.SlideRecommendation != nil {
		s.RegisterTool(NewInkGetImagesForSlidesTool(s))
		log.Printf("Registered slide recommendation tool: ink_get_images_for_slides")
	}

	log.Printf("MCP tool registration complete")
}

// GetServices 取得服務
func (s *MCPServer) GetServices() *MCPServices {
	return s.services
}

// SetIO 設定輸入輸出
func (s *MCPServer) SetIO(stdin io.Reader, stdout io.Writer, stderr io.Writer) {
	s.stdin = stdin
	s.stdout = stdout
	s.stderr = stderr
}