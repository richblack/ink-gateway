# Ink Multimodal MCP Server

這是一個基於 Model Context Protocol (MCP) 的多模態知識庫伺服器，支援文字和圖片的智能處理與搜尋。

## 功能特色

### 核心工具 (Core Tools)
- **ink_search_chunks**: 多模態搜尋（文字、圖片或混合搜尋）
- **ink_analyze_image**: AI 圖片分析
- **ink_upload_image**: 圖片上傳與處理
- **ink_create_chunk**: 建立知識塊

### 進階工具 (Advanced Tools)
- **ink_batch_process_images**: 批次處理圖片
- **ink_get_images_for_slides**: 投影片圖片推薦
- **ink_search_images**: 圖片搜尋
- **ink_hybrid_search**: 混合搜尋

### 資源 (Resources)
- **ink://chunks/{chunk_id}**: 知識塊資源
- **ink://images/{chunk_id}**: 圖片資源
- **ink://search/{search_id}**: 搜尋結果資源

### 提示 (Prompts)
- **ink_search_assistant**: 搜尋助手提示
- **ink_image_analysis_assistant**: 圖片分析助手提示

## 安裝與設定

### 1. 環境變數設定

建立 `.env` 文件：

```bash
# Supabase 設定
SUPABASE_URL=your_supabase_url
SUPABASE_API_KEY=your_supabase_api_key

# AI 服務設定
LLM_API_KEY=your_llm_api_key
EMBEDDING_API_KEY=your_embedding_api_key

# 伺服器設定
SERVER_PORT=8080
```

### 2. 編譯與執行

```bash
# 編譯 MCP 伺服器
go build -o bin/mcp-server ./cmd/mcp-server

# 執行 MCP 伺服器
./bin/mcp-server
```

### 3. MCP 客戶端設定

將 `mcp.json` 配置文件加入到你的 MCP 客戶端設定中：

```json
{
  "mcpServers": {
    "ink-multimodal": {
      "command": "go",
      "args": ["run", "./cmd/mcp-server"],
      "env": {
        "SUPABASE_URL": "${SUPABASE_URL}",
        "SUPABASE_API_KEY": "${SUPABASE_API_KEY}",
        "LLM_API_KEY": "${LLM_API_KEY}",
        "EMBEDDING_API_KEY": "${EMBEDDING_API_KEY}"
      },
      "disabled": false,
      "autoApprove": [
        "ink_search_chunks",
        "ink_analyze_image",
        "ink_create_chunk"
      ]
    }
  }
}
```

## 使用範例

### 搜尋知識塊

```json
{
  "tool": "ink_search_chunks",
  "arguments": {
    "query": "機器學習算法",
    "search_type": "hybrid",
    "limit": 10,
    "min_similarity": 0.7
  }
}
```

### 分析圖片

```json
{
  "tool": "ink_analyze_image",
  "arguments": {
    "image_url": "https://example.com/image.jpg",
    "detail_level": "high",
    "language": "zh-TW"
  }
}
```

### 上傳圖片

```json
{
  "tool": "ink_upload_image",
  "arguments": {
    "image_path": "/path/to/image.jpg",
    "page_id": "page_123",
    "tags": ["機器學習", "圖表"],
    "auto_analyze": true,
    "auto_embed": true
  }
}
```

### 批次處理圖片

```json
{
  "tool": "ink_batch_process_images",
  "arguments": {
    "folder_path": "/path/to/images",
    "page_id": "presentation_slides",
    "tags": ["投影片", "圖片"],
    "auto_analyze": true,
    "concurrency": 3
  }
}
```

### 投影片圖片推薦

```json
{
  "tool": "ink_get_images_for_slides",
  "arguments": {
    "slide_title": "機器學習概述",
    "slide_content": "介紹機器學習的基本概念和應用",
    "max_suggestions": 5,
    "min_relevance": 0.6
  }
}
```

## 資源存取

### 存取知識塊資源

```
ink://chunks/chunk_123
```

### 存取圖片資源

```
ink://images/chunk_456
```

### 存取搜尋結果

```
ink://search/search_789
```

## 提示使用

### 搜尋助手

```json
{
  "prompt": "ink_search_assistant",
  "arguments": {
    "search_context": "我想找關於深度學習的資料",
    "content_type": "both"
  }
}
```

### 圖片分析助手

```json
{
  "prompt": "ink_image_analysis_assistant",
  "arguments": {
    "analysis_purpose": "documentation",
    "detail_level": "high"
  }
}
```

## 架構說明

### 服務層次

1. **MCP 協議層**: 處理 MCP 通訊協議
2. **工具層**: 實作各種 MCP 工具
3. **服務層**: 核心業務邏輯
4. **資料層**: 資料庫和儲存服務

### 主要組件

- **MCPServer**: MCP 協議伺服器
- **MediaProcessor**: 媒體處理服務
- **MultimodalSearch**: 多模態搜尋服務
- **BatchProcessor**: 批次處理服務
- **ImageSimilarity**: 圖片相似度搜尋
- **SlideRecommendation**: 投影片推薦服務

## 開發與測試

### 執行測試

```bash
# 執行所有測試
make test

# 執行 MCP 相關測試
go test ./mcp/...
```

### 開發模式

```bash
# 開發模式執行
make run

# 或直接執行
go run ./cmd/mcp-server
```

## 故障排除

### 常見問題

1. **連線失敗**: 檢查環境變數設定
2. **權限錯誤**: 確認 API 金鑰正確
3. **圖片處理失敗**: 檢查圖片格式和大小
4. **搜尋無結果**: 確認資料庫中有相關內容

### 日誌檢查

MCP 伺服器會輸出詳細的日誌資訊，包括：
- 工具執行狀態
- 錯誤訊息
- 效能指標

## 貢獻指南

1. Fork 專案
2. 建立功能分支
3. 提交變更
4. 建立 Pull Request

## 授權

MIT License