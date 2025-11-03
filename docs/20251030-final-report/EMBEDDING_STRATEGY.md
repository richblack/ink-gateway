# Embedding 策略說明

## 📊 你的使用場景分析

### 場景特性
- ✅ **寫筆記時分段變成 chunk 來 embed**
- ✅ **不是上傳 PDF 文件切割**
- ✅ **文字為主的 chunk embedding**

### 結論
**OpenAI text-embedding-3-small 完全適合！** 🎯

---

## 1️⃣ 圖片 Embedding 處理方式

### 當前架構

根據 `services/clip_embedding_service.go`，系統支援三種圖片 embedding 方式：

#### A. **CLIP API 服務**（預設）
```go
type CLIPEmbeddingService struct {
    apiURL     string          // 外部 CLIP API URL
    model      "clip-vit-b-32" // CLIP 模型
    dimensions 512             // 向量維度
}
```

**特點**：
- 🌐 **呼叫外部 API**（不是本地下載處理）
- 📡 透過 HTTP POST `/embeddings` 端點
- 🎯 使用 CLIP ViT-B/32 模型
- 📏 生成 512 維向量

**API 請求格式**：
```json
{
  "images": ["image_url_1", "image_url_2"],
  "model": "clip-vit-b-32"
}
```

#### B. **本地 CLIP 服務**（未實作）
```go
type LocalCLIPService struct {
    modelPath  string // 本地模型路徑
    // TODO: 需要實作
    // 可能方式：CGO 呼叫 Python、ONNX Runtime
}
```

**狀態**：框架已建立，但實作尚未完成

#### C. **Mock 服務**（測試用）
```go
type MockImageEmbeddingService struct {
    // 模擬向量生成（測試用）
    dimensions 512
}
```

### 🎯 圖片 Embedding 流程

```
上傳圖片
   ↓
儲存到 Supabase Storage
   ↓
取得圖片 URL
   ↓
呼叫 CLIP API (外部服務)
   ↓
取得 512 維向量
   ↓
儲存到 PostgreSQL (pgvector)
```

**答案：圖片是透過外部 CLIP API 處理，不是本地下載模型**

---

## 2️⃣ 文字 Chunk Embedding 適用性

### 你的場景：筆記分段 Embedding

#### 典型 Chunk 大小
```
筆記段落範例：
「今天學習了 PostgreSQL 的 pgvector 擴充功能。
它可以儲存向量資料並進行相似度搜尋。
向量維度通常是 512 或 1536。」

Token 數：約 50-100 tokens
```

### OpenAI Small vs Large 比較

| 特性 | text-embedding-3-small | text-embedding-3-large |
|------|------------------------|------------------------|
| 價格 | **$0.02/1M tokens** | $0.13/1M tokens |
| 維度 | 512 或 1536 (可調) | 1536 或 3072 |
| 性能 | ⭐⭐⭐⭐ (優秀) | ⭐⭐⭐⭐⭐ (極佳) |
| 適合場景 | ✅ **筆記 chunk** | 專業搜尋、多語言 |
| 中文支援 | ✅ 良好 | ✅ 優秀 |

### 🎯 為什麼 Small 適合你的場景？

#### 1. **Chunk 大小適中**
```
筆記段落：50-200 tokens
PDF 段落：200-500 tokens (較長)

Small 模型在短文本表現優秀！
```

#### 2. **成本優勢明顯**
```
假設每天寫 100 條筆記：
- 平均每條 100 tokens
- 每月 3,000 條筆記 = 300K tokens

Small: $0.02 × 0.3 = $0.006/月 (6 分錢)
Large: $0.13 × 0.3 = $0.039/月 (4 分錢)

一年成本：
Small: $0.07
Large: $0.47
```

#### 3. **性能足夠**
根據 OpenAI 官方測試：
- Small: MTEB 分數 62.3
- Large: MTEB 分數 64.6
- **差距僅 3.7%**

對於筆記搜尋，這個差距幾乎無感！

#### 4. **速度更快**
- Small: 更快的 embedding 生成
- Large: 略慢但差異不大

---

## 3️⃣ Gemini API 配置

### 更新 .env 配置

我現在幫你配置 Gemini API：

```bash
# Gemini API Configuration
EMBEDDING_API_KEY=AIzaSyCkWbtCuEl-3x3fLn27b7TV8Vjel86rGQ4
EMBEDDING_ENDPOINT=https://generativelanguage.googleapis.com/v1
EMBEDDING_TIMEOUT=30s

# 圖片分析也用 Gemini（有免費額度）
LLM_API_KEY=AIzaSyCkWbtCuEl-3x3fLn27b7TV8Vjel86rGQ4
LLM_ENDPOINT=https://generativelanguage.googleapis.com/v1
LLM_TIMEOUT=60s
```

### ⚠️ 重要提醒

你的 API Key 已經暴露在對話中！建議：

1. **立即重新生成新的 Key**
2. 前往：https://makersuite.google.com/app/apikey
3. 刪除舊 Key，創建新的
4. 更新 .env 文件

---

## 4️⃣ 完整配置策略

### 測試階段（現在）

```bash
# 伺服器
SERVER_PORT=8081

# Supabase (本地)
SUPABASE_URL=http://localhost:8000
SUPABASE_API_KEY=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...

# Gemini (免費額度測試)
LLM_API_KEY=你的新_Gemini_Key
LLM_ENDPOINT=https://generativelanguage.googleapis.com/v1
LLM_TIMEOUT=60s

EMBEDDING_API_KEY=你的新_Gemini_Key
EMBEDDING_ENDPOINT=https://generativelanguage.googleapis.com/v1
EMBEDDING_TIMEOUT=30s

# 圖片 CLIP Embedding (外部服務)
# 需要自行部署或使用第三方 CLIP API
CLIP_API_URL=http://localhost:5000

# 日誌
LOG_LEVEL=debug
LOG_FORMAT=json
```

### 生產環境（建議）

```bash
# 文字 Embedding: OpenAI Small (便宜 7.5 倍)
EMBEDDING_API_KEY=sk-proj-your-openai-key
EMBEDDING_ENDPOINT=https://api.openai.com/v1
EMBEDDING_MODEL=text-embedding-3-small

# 圖片分析: Gemini Vision (多語言好)
LLM_API_KEY=your-gemini-key
LLM_ENDPOINT=https://generativelanguage.googleapis.com/v1

# 圖片 Embedding: 獨立 CLIP 服務
CLIP_API_URL=https://your-clip-service.com
```

---

## 5️⃣ 圖片 CLIP API 部署選項

### 選項 A：使用 Replicate（推薦測試）

```bash
# 免費額度，簡單易用
CLIP_API_URL=https://api.replicate.com/v1
CLIP_API_KEY=your-replicate-token
```

註冊：https://replicate.com/

### 選項 B：自行部署（Docker）

```bash
# 1. 使用 OpenAI CLIP Docker
docker run -d -p 5000:5000 \
  --name clip-server \
  openai/clip-server

# 2. 配置
CLIP_API_URL=http://localhost:5000
```

### 選項 C：使用 Hugging Face Inference API

```bash
CLIP_API_URL=https://api-inference.huggingface.co/models/openai/clip-vit-base-patch32
CLIP_API_KEY=your-hf-token
```

---

## 6️⃣ 成本分析總結

### 你的筆記場景（每月 3000 條）

| 項目 | 服務 | 成本/月 |
|------|------|---------|
| 文字 Embedding | OpenAI Small | **$0.006** |
| 圖片 Embedding | CLIP (自建) | **免費** |
| 圖片分析 (偶爾) | Gemini | **免費** (額度內) |
| **總計** | | **~$0.01/月** |

### 即使用 1 年
- **總成本：約 $0.12**
- 比一杯咖啡還便宜！☕

---

## 📋 行動清單

### 立即執行

1. ✅ **重新生成 Gemini API Key**
   - 網址：https://makersuite.google.com/app/apikey
   - 刪除舊的暴露的 Key
   - 創建新 Key

2. ✅ **更新 .env 配置**
   - 使用新的 Gemini Key
   - 配置測試環境

3. ✅ **啟動測試**
   ```bash
   ./semantic-text-processor
   ```

### 未來規劃

1. **部署 CLIP 服務**（如需圖片搜尋功能）
2. **生產環境切換到 OpenAI Small**（節省成本）
3. **監控使用量和成本**

---

## ❓ 常見問題

### Q1: 為什麼不全用 Gemini？
**A**:
- Gemini 文字 embedding: $0.15/1M
- OpenAI Small: $0.02/1M
- **生產環境 OpenAI 便宜 7.5 倍**

### Q2: CLIP 一定要部署嗎？
**A**:
- 如果**不需要以圖搜圖**功能，可以不部署
- 只用文字搜尋，只需要 OpenAI Small

### Q3: 能混用不同服務嗎？
**A**:
- ✅ 可以！例如：
  - 文字用 OpenAI
  - 圖片用 Gemini
  - CLIP 自己部署

### Q4: 我的場景真的適合 Small 嗎？
**A**:
- ✅ **非常適合！**
- 短文本 (50-200 tokens)
- Small 性能優秀
- 成本低廉
- 速度快

---

## 🎯 最終建議

### 測試階段（2週內）
```
文字 Embedding: Gemini (免費)
圖片分析: Gemini (免費)
圖片 CLIP: 暫不部署（先測試文字功能）
```

### 生產環境
```
文字 Embedding: OpenAI Small ($0.02/1M)
圖片分析: Gemini ($0.01/image)
圖片 CLIP: Replicate 或自建
```

**年成本估算**：約 $1-5（取決於使用量）
