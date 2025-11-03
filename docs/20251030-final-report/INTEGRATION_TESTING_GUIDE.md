# 多模態 MCP 系統整合測試指南

本指南提供完整的 step-by-step 測試流程，包括系統啟動、插件安裝、MCP 配置和整合測試。

## 目錄
1. [環境準備](#環境準備)
2. [核心 Ink-Gateway 啟動](#核心-ink-gateway-啟動)
3. [Obsidian 插件安裝](#obsidian-插件安裝)
4. [MCP Server 配置](#mcp-server-配置)
5. [整合測試案例](#整合測試案例)
6. [故障排除](#故障排除)

## 環境準備

### 系統需求
- Go 1.21+
- Node.js 18+
- Docker (可選)
- Obsidian
- Claude Desktop 或支援 MCP 的客戶端

### 必要的 API 金鑰
```bash
# 建立 .env 文件
cp .env.example .env

# 編輯 .env 文件，填入以下資訊：
SUPABASE_URL=your_supabase_project_url
SUPABASE_API_KEY=your_supabase_anon_key
LLM_API_KEY=your_openai_api_key
EMBEDDING_API_KEY=your_clip_api_key_or_openai_key
SERVER_PORT=8080
```

## 核心 Ink-Gateway 啟動

### 方法 1: 本地 Go 運行（推薦開發）

```bash
# 1. 安裝依賴
make deps

# 2. 運行資料庫遷移（如果需要）
# 確保 Supabase 專案已設定好相關表格

# 3. 啟動 Ink-Gateway 服務
make run
# 或者
go run cmd/server/main.go

# 4. 驗證服務運行
curl http://localhost:8080/health
# 應該返回: {"status": "ok"}
```

### 方法 2: Docker 運行（推薦生產）

```bash
# 1. 建立 Docker 映像
docker build -t ink-gateway .

# 2. 運行容器
docker run -d \
  --name ink-gateway \
  -p 8080:8080 \
  --env-file .env \
  ink-gateway

# 3. 檢查容器狀態
docker logs ink-gateway

# 4. 驗證服務
curl http://localhost:8080/health
```

### 方法 3: Docker Compose（推薦完整環境）

```bash
# 1. 啟動完整環境
docker-compose up -d

# 2. 檢查所有服務狀態
docker-compose ps

# 3. 查看日誌
docker-compose logs -f ink-gateway
```

### 測試 Ink-Gateway API

```bash
# 測試基本 API 端點
curl -X GET http://localhost:8080/api/v1/chunks

# 測試圖片上傳 API
curl -X POST http://localhost:8080/api/v1/media/upload \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your_api_key" \
  -d '{
    "image_data": "base64_encoded_image_data",
    "filename": "test.jpg",
    "auto_analyze": true,
    "auto_embed": true
  }'

# 測試多模態搜尋
curl -X POST http://localhost:8080/api/v1/search/multimodal \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your_api_key" \
  -d '{
    "text_query": "機器學習",
    "search_type": "hybrid",
    "limit": 10
  }'
```

## Obsidian 插件安裝

### 開發模式安裝

```bash
# 1. 進入 Obsidian 插件目錄
cd obsidian-ink-plugin

# 2. 安裝依賴
npm install

# 3. 建構插件
npm run build

# 4. 複製到 Obsidian 插件目錄
# Windows
cp -r . "C:\Users\YourName\AppData\Roaming\Obsidian\plugins\obsidian-ink-plugin"

# macOS
cp -r . "~/Library/Application Support/obsidian/plugins/obsidian-ink-plugin"

# Linux
cp -r . "~/.config/obsidian/plugins/obsidian-ink-plugin"
```

### 手動安裝步驟

1. **開啟 Obsidian**
2. **進入設定** → **社群插件**
3. **關閉安全模式**
4. **瀏覽** → 選擇插件資料夾
5. **啟用 Obsidian Ink Plugin**
6. **配置插件設定**：
   ```
   Ink Gateway URL: http://localhost:8080
   API Key: your_api_key
   Auto Sync: 啟用
   Sync Interval: 5000ms
   ```

### 驗證插件安裝

```bash
# 1. 檢查插件是否載入
# 在 Obsidian 中按 Ctrl+Shift+I 開啟開發者工具
# 查看 Console 是否有 "Loading Obsidian Ink Plugin..." 訊息

# 2. 測試插件命令
# 按 Ctrl+P 開啟命令面板
# 搜尋 "Ink" 應該看到相關命令：
# - Upload Image
# - Open Image Library
# - Batch Upload Images
# - Upload Image from Clipboard
```

## MCP Server 配置

### 建構 MCP Server

```bash
# 1. 建構 MCP Server
go build -o bin/mcp-server ./cmd/mcp-server

# 2. 測試 MCP Server
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}' | ./bin/mcp-server

# 應該返回初始化回應
```

### Claude Desktop 配置

1. **找到 Claude Desktop 配置文件**：
   ```bash
   # macOS
   ~/Library/Application Support/Claude/claude_desktop_config.json
   
   # Windows
   %APPDATA%\Claude\claude_desktop_config.json
   
   # Linux
   ~/.config/claude/claude_desktop_config.json
   ```

2. **編輯配置文件**：
   ```json
   {
     "mcpServers": {
       "ink-multimodal": {
         "command": "go",
         "args": ["run", "./cmd/mcp-server"],
         "cwd": "/path/to/your/ink-gateway",
         "env": {
           "SUPABASE_URL": "your_supabase_url",
           "SUPABASE_API_KEY": "your_supabase_key",
           "LLM_API_KEY": "your_llm_key",
           "EMBEDDING_API_KEY": "your_embedding_key"
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

3. **重啟 Claude Desktop**

### 驗證 MCP 連接

1. **在 Claude Desktop 中測試**：
   ```
   請使用 ink_search_chunks 工具搜尋 "測試" 相關的內容
   ```

2. **檢查可用工具**：
   ```
   請列出所有可用的 ink 相關工具
   ```

## 整合測試案例

### 測試案例 1: 基本圖片上傳流程

```bash
# 1. 準備測試圖片
mkdir -p test-images
# 放入一些測試圖片: test1.jpg, test2.png, test3.gif

# 2. 在 Obsidian 中測試拖放上傳
# - 開啟一個 Markdown 文件
# - 拖放圖片到編輯器
# - 檢查是否自動插入圖片連結
# - 檢查是否有 AI 分析註解

# 3. 驗證後端處理
curl -X GET http://localhost:8080/api/v1/media/library | jq .

# 4. 在 Claude Desktop 中測試
# "請使用 ink_analyze_image 工具分析剛上傳的圖片"
```

### 測試案例 2: 批次處理流程

```bash
# 1. 準備批次測試圖片
mkdir -p test-batch
# 放入 10-20 張測試圖片

# 2. 在 Obsidian 中測試批次上傳
# - 按 Ctrl+P 開啟命令面板
# - 執行 "Batch Upload Images"
# - 選擇 test-batch 資料夾
# - 配置處理選項
# - 開始處理並觀察進度

# 3. 驗證批次處理結果
curl -X GET "http://localhost:8080/api/v1/media/batch/status" | jq .

# 4. 在 Claude Desktop 中測試批次查詢
# "請使用 ink_search_images 工具搜尋剛批次上傳的圖片"
```

### 測試案例 3: 多模態搜尋流程

```bash
# 1. 在 Obsidian 中建立包含圖片的筆記
# - 建立新筆記 "機器學習概念.md"
# - 上傳相關圖片
# - 添加文字描述

# 2. 測試 Obsidian 搜尋功能
# - 按 Ctrl+P 執行 "Open Semantic Search"
# - 搜尋 "深度學習"
# - 檢查是否返回相關圖片和文字

# 3. 在 Claude Desktop 中測試混合搜尋
# "請使用 ink_hybrid_search 工具，以文字查詢 '神經網路' 和圖片查詢結合搜尋相關內容"

# 4. 驗證搜尋 API
curl -X POST http://localhost:8080/api/v1/search/multimodal \
  -H "Content-Type: application/json" \
  -d '{
    "text_query": "機器學習",
    "search_type": "hybrid",
    "limit": 5
  }' | jq .
```

### 測試案例 4: 投影片圖片推薦

```bash
# 1. 準備投影片內容
# 在 Obsidian 中建立 "AI 簡報.md" 包含：
# - 標題：人工智慧發展趨勢
# - 內容：介紹機器學習、深度學習等概念

# 2. 在 Claude Desktop 中測試推薦
# "請使用 ink_get_images_for_slides 工具，為標題 '人工智慧發展趨勢' 和內容 '介紹機器學習、深度學習等概念' 推薦合適的圖片"

# 3. 驗證推薦 API
curl -X POST http://localhost:8080/api/v1/media/recommend-slides \
  -H "Content-Type: application/json" \
  -d '{
    "slide_title": "人工智慧發展趨勢",
    "slide_content": "介紹機器學習、深度學習等概念",
    "max_suggestions": 5
  }' | jq .
```

### 測試案例 5: 重複圖片檢測

```bash
# 1. 上傳重複或相似圖片
# - 上傳同一張圖片的不同版本
# - 上傳相似的圖片

# 2. 在 Claude Desktop 中測試重複檢測
# "請使用 find_duplicates 工具檢測重複的圖片"

# 3. 驗證重複檢測 API
curl -X POST http://localhost:8080/api/v1/media/find-duplicates \
  -H "Content-Type: application/json" \
  -d '{
    "similarity_threshold": 0.95,
    "min_group_size": 2
  }' | jq .
```

## 效能測試

### 負載測試腳本

```bash
#!/bin/bash
# load_test.sh

echo "開始負載測試..."

# 測試並發圖片上傳
for i in {1..10}; do
  curl -X POST http://localhost:8080/api/v1/media/upload \
    -H "Content-Type: application/json" \
    -d "{\"image_data\":\"$(base64 -i test-images/test$i.jpg)\",\"filename\":\"load_test_$i.jpg\"}" &
done

wait
echo "並發上傳測試完成"

# 測試搜尋效能
time curl -X POST http://localhost:8080/api/v1/search/multimodal \
  -H "Content-Type: application/json" \
  -d '{"text_query":"測試","search_type":"hybrid","limit":50}'

echo "負載測試完成"
```

### 記憶體和 CPU 監控

```bash
# 監控 Ink-Gateway 效能
top -p $(pgrep -f "ink-gateway")

# 監控 MCP Server 效能
top -p $(pgrep -f "mcp-server")

# 檢查記憶體使用
ps aux | grep -E "(ink-gateway|mcp-server)" | awk '{print $2, $4, $6, $11}'
```

## 故障排除

### 常見問題和解決方案

#### 1. Ink-Gateway 啟動失敗
```bash
# 檢查日誌
tail -f logs/ink-gateway.log

# 檢查端口占用
lsof -i :8080

# 檢查環境變數
env | grep -E "(SUPABASE|LLM|EMBEDDING)"
```

#### 2. Obsidian 插件無法載入
```bash
# 檢查插件目錄
ls -la ~/.config/obsidian/plugins/obsidian-ink-plugin/

# 檢查 manifest.json
cat ~/.config/obsidian/plugins/obsidian-ink-plugin/manifest.json

# 重新建構插件
cd obsidian-ink-plugin && npm run build
```

#### 3. MCP Server 連接失敗
```bash
# 測試 MCP Server 獨立運行
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}' | ./bin/mcp-server

# 檢查 Claude Desktop 配置
cat ~/Library/Application\ Support/Claude/claude_desktop_config.json

# 檢查 MCP Server 日誌
./bin/mcp-server 2>&1 | tee mcp-server.log
```

#### 4. API 調用失敗
```bash
# 檢查 API 金鑰
curl -H "Authorization: Bearer $LLM_API_KEY" https://api.openai.com/v1/models

# 檢查 Supabase 連接
curl -H "apikey: $SUPABASE_API_KEY" "$SUPABASE_URL/rest/v1/"

# 測試網路連接
ping api.openai.com
```

### 日誌分析

```bash
# 分析錯誤日誌
grep -i error logs/*.log | tail -20

# 分析效能日誌
grep -i "slow\|timeout\|performance" logs/*.log

# 分析 API 調用日誌
grep -E "(POST|GET|PUT|DELETE)" logs/access.log | tail -20
```

## 自動化測試腳本

### 完整整合測試腳本

```bash
#!/bin/bash
# integration_test.sh

set -e

echo "🚀 開始多模態 MCP 系統整合測試"

# 1. 檢查環境
echo "📋 檢查環境..."
command -v go >/dev/null 2>&1 || { echo "需要安裝 Go"; exit 1; }
command -v node >/dev/null 2>&1 || { echo "需要安裝 Node.js"; exit 1; }
command -v curl >/dev/null 2>&1 || { echo "需要安裝 curl"; exit 1; }

# 2. 啟動 Ink-Gateway
echo "🔧 啟動 Ink-Gateway..."
make run &
GATEWAY_PID=$!
sleep 10

# 3. 測試 API 健康檢查
echo "🏥 測試 API 健康檢查..."
curl -f http://localhost:8080/health || { echo "API 健康檢查失敗"; exit 1; }

# 4. 建構 MCP Server
echo "🔨 建構 MCP Server..."
go build -o bin/mcp-server ./cmd/mcp-server

# 5. 測試 MCP Server
echo "🧪 測試 MCP Server..."
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}' | timeout 10 ./bin/mcp-server > /dev/null || { echo "MCP Server 測試失敗"; exit 1; }

# 6. 建構 Obsidian 插件
echo "🔌 建構 Obsidian 插件..."
cd obsidian-ink-plugin
npm install
npm run build
cd ..

# 7. 執行 API 測試
echo "🌐 執行 API 測試..."
./scripts/api_test.sh

# 8. 清理
echo "🧹 清理..."
kill $GATEWAY_PID 2>/dev/null || true

echo "✅ 整合測試完成！"
```

這個完整的測試指南涵蓋了從環境準備到整合測試的所有步驟，讓你能夠系統性地驗證多模態 MCP 系統的各個組件是否正常運作。