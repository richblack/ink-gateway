# 配置指南 Configuration Guide

## 📋 必要配置項目

根據 `config/config.go`，以下是應用程式所需的環境變數配置：

### 1️⃣ 基礎配置（必須）

#### Supabase 連線
```bash
SUPABASE_URL=http://localhost:8000
SUPABASE_API_KEY=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyAgCiAgICAicm9sZSI6ICJzZXJ2aWNlX3JvbGUiLAogICAgImlzcyI6ICJzdXBhYmFzZS1kZW1vIiwKICAgICJpYXQiOiAxNjQxNzY5MjAwLAogICAgImV4cCI6IDE3OTk1MzU2MDAKfQ.DaYlNEoUrrEn2Ig7tqibS-PHK5vgusbcbo7X36XVt4Q
```
✅ **狀態**: 已配置（本地 Docker Supabase）

#### 伺服器設定
```bash
SERVER_PORT=8081
```
✅ **狀態**: 已配置

---

### 2️⃣ AI 服務配置（選用 - 測試階段可用假值）

#### LLM 服務（大語言模型）
```bash
LLM_API_KEY=test_key           # ⚠️ 測試用假值
LLM_ENDPOINT=http://localhost:8000
LLM_TIMEOUT=60s
```

**用途**：
- 圖片內容分析（AI 描述圖片內容）
- 圖片推薦（根據投影片內容推薦圖片）
- 語義理解

**真實 API 選項**：
1. **OpenAI GPT-4 Vision**
   - API Key: 從 [OpenAI Platform](https://platform.openai.com/) 獲取
   - Endpoint: `https://api.openai.com/v1`

2. **Google Gemini Vision**
   - API Key: 從 [Google AI Studio](https://makersuite.google.com/app/apikey) 獲取
   - Endpoint: `https://generativelanguage.googleapis.com/v1`

3. **Anthropic Claude**
   - API Key: 從 [Anthropic Console](https://console.anthropic.com/) 獲取
   - Endpoint: `https://api.anthropic.com/v1`

#### Embedding 服務（向量嵌入）
```bash
EMBEDDING_API_KEY=test_key     # ⚠️ 測試用假值
EMBEDDING_ENDPOINT=http://localhost:8001
EMBEDDING_TIMEOUT=30s
```

**用途**：
- 圖片向量化（以圖搜圖功能）
- 文字向量化（語義搜索）
- 相似度計算

**真實 API 選項**：
1. **OpenAI Embeddings**
   - 模型: `text-embedding-3-large` 或 `text-embedding-3-small`
   - API Key: 同 LLM API Key
   - Endpoint: `https://api.openai.com/v1`

2. **Voyage AI**
   - 專門的 embedding 服務
   - API Key: 從 [Voyage AI](https://www.voyageai.com/) 獲取
   - Endpoint: `https://api.voyageai.com/v1`

3. **本地部署選項**:
   - [CLIP](https://github.com/openai/CLIP)（圖片+文字）
   - [Sentence Transformers](https://www.sbert.net/)（純文字）

---

### 3️⃣ 選用配置（有預設值）

#### 日誌設定
```bash
LOG_LEVEL=info              # debug, info, warn, error
LOG_FORMAT=json             # json, text
```

#### 快取設定
```bash
CACHE_ENABLED=true
CACHE_MAX_SIZE=1000
CACHE_CLEANUP_INTERVAL=5m
CACHE_DEFAULT_TTL=30m
```

#### 效能監控
```bash
METRICS_ENABLED=true
METRICS_ENDPOINT=/metrics
MONITORING_ENABLED=true
SLOW_QUERY_THRESHOLD=500ms
```

#### 功能開關
```bash
USE_UNIFIED_HANDLERS=false
```

---

## 🎯 測試階段建議配置

### 方案 A：純測試模式（不需要真實 API）
保持當前的 `.env` 配置即可：
```bash
LLM_API_KEY=test_key
EMBEDDING_API_KEY=test_key
```

**限制**：
- ❌ 無法使用 AI 圖片分析功能
- ❌ 無法使用向量搜索（以圖搜圖）
- ✅ 可以測試圖片上傳和存儲
- ✅ 可以測試批次處理功能
- ✅ 可以測試資料庫 CRUD 操作

### 方案 B：部分 AI 功能（推薦）
只配置 OpenAI API（一個 key 兩用）：

1. **取得 OpenAI API Key**:
   - 前往 https://platform.openai.com/api-keys
   - 登入並創建新的 API key
   - 複製 key（格式：`sk-...`）

2. **更新 .env**:
   ```bash
   # 改用真實 OpenAI API
   LLM_API_KEY=sk-your-actual-openai-key-here
   LLM_ENDPOINT=https://api.openai.com/v1

   EMBEDDING_API_KEY=sk-your-actual-openai-key-here
   EMBEDDING_ENDPOINT=https://api.openai.com/v1
   ```

**優點**：
- ✅ 只需一個 API key
- ✅ 完整功能可用
- ✅ 價格相對便宜（約 $0.01/1K tokens）

### 方案 C：完整本地部署（進階）
使用 Ollama 本地運行 AI 模型：

1. **安裝 Ollama**:
   ```bash
   brew install ollama
   ollama pull llava        # 視覺模型
   ollama pull nomic-embed  # embedding 模型
   ```

2. **更新 .env**:
   ```bash
   LLM_API_KEY=not-needed
   LLM_ENDPOINT=http://localhost:11434

   EMBEDDING_API_KEY=not-needed
   EMBEDDING_ENDPOINT=http://localhost:11434
   ```

**優點**：
- ✅ 完全免費
- ✅ 數據隱私
- ❌ 需要較強的硬體（建議 16GB+ RAM）

---

## 📝 完整 .env 範例

### 測試環境（最小配置）
```bash
# Server
SERVER_PORT=8081

# Supabase
SUPABASE_URL=http://localhost:8000
SUPABASE_API_KEY=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyAgCiAgICAicm9sZSI6ICJzZXJ2aWNlX3JvbGUiLAogICAgImlzcyI6ICJzdXBhYmFzZS1kZW1vIiwKICAgICJpYXQiOiAxNjQxNzY5MjAwLAogICAgImV4cCI6IDE3OTk1MzU2MDAKfQ.DaYlNEoUrrEn2Ig7tqibS-PHK5vgusbcbo7X36XVt4Q

# AI Services (測試用假值)
LLM_API_KEY=test_key
LLM_ENDPOINT=http://localhost:8000
LLM_TIMEOUT=60s

EMBEDDING_API_KEY=test_key
EMBEDDING_ENDPOINT=http://localhost:8001
EMBEDDING_TIMEOUT=30s

# Logging
LOG_LEVEL=info
LOG_FORMAT=json
```

### 開發環境（使用 OpenAI）
```bash
# Server
SERVER_PORT=8081

# Supabase
SUPABASE_URL=http://localhost:8000
SUPABASE_API_KEY=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyAgCiAgICAicm9sZSI6ICJzZXJ2aWNlX3JvbGUiLAogICAgImlzcyI6ICJzdXBhYmFzZS1kZW1vIiwKICAgICJpYXQiOiAxNjQxNzY5MjAwLAogICAgImV4cCI6IDE3OTk1MzU2MDAKfQ.DaYlNEoUrrEn2Ig7tqibS-PHK5vgusbcbo7X36XVt4Q

# OpenAI (真實 API)
LLM_API_KEY=sk-proj-your-actual-key-here
LLM_ENDPOINT=https://api.openai.com/v1
LLM_TIMEOUT=60s

EMBEDDING_API_KEY=sk-proj-your-actual-key-here
EMBEDDING_ENDPOINT=https://api.openai.com/v1
EMBEDDING_TIMEOUT=30s

# Logging
LOG_LEVEL=debug
LOG_FORMAT=json

# Cache
CACHE_ENABLED=true
CACHE_MAX_SIZE=1000

# Performance
METRICS_ENABLED=true
MONITORING_ENABLED=true
```

---

## 🚀 啟動測試

### 1. 檢查配置
```bash
# 顯示當前配置
cat .env

# 驗證 Supabase 連線
curl http://localhost:8000/rest/v1/ \
  -H "apikey: YOUR_SUPABASE_KEY"
```

### 2. 啟動應用
```bash
./semantic-text-processor
```

### 3. 測試健康檢查
```bash
curl http://localhost:8081/health
```

### 4. 測試 API 端點
```bash
# 圖片上傳
curl -X POST http://localhost:8081/api/v1/media/upload \
  -F "image=@test.jpg" \
  -F "page_id=test-page" \
  -F "auto_analyze=true"

# 多模態搜索
curl -X POST http://localhost:8081/api/v1/search/multimodal \
  -H "Content-Type: application/json" \
  -d '{"text_query": "測試圖片", "limit": 10}'
```

---

## ❓ 常見問題

### Q1: 我需要立即取得真實 API key 嗎？
**A**: 不需要。測試階段可以使用假值，只要不觸發 AI 功能即可。

### Q2: OpenAI API 收費如何？
**A**:
- GPT-4 Vision: ~$0.01/image
- Embeddings: ~$0.0001/1K tokens
- 小規模測試約 $1-5 即可

### Q3: 可以混用不同服務嗎？
**A**: 可以。例如：
- LLM 用 OpenAI
- Embedding 用 Voyage AI 或本地 CLIP

### Q4: 如何檢查配置是否正確？
**A**:
```bash
# 應用啟動時會驗證必要配置
./semantic-text-processor

# 查看日誌中的配置加載訊息
# 如果缺少必要配置，會顯示錯誤
```

---

## 📊 配置優先級總結

| 配置項 | 必要性 | 測試階段 | 生產環境 |
|--------|--------|----------|----------|
| SUPABASE_URL | ✅ 必須 | localhost | 生產 URL |
| SUPABASE_API_KEY | ✅ 必須 | demo key | 生產 key |
| LLM_API_KEY | ⚠️ 選用 | 假值可 | 真實 key |
| EMBEDDING_API_KEY | ⚠️ 選用 | 假值可 | 真實 key |
| SERVER_PORT | ✅ 必須 | 8081 | 依需求 |
| LOG_LEVEL | ⭕ 選用 | debug | info/warn |

---

**當前狀態**:
- ✅ Supabase 已配置且運行中
- ✅ 伺服器端口已設定
- ⚠️ AI 服務使用測試假值（功能受限）
- ✅ 應用已成功編譯

**建議**: 先用當前配置測試基本功能，如果需要完整 AI 功能再取得 OpenAI API key。
