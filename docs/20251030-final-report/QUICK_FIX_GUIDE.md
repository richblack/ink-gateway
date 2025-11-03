# Ink-Gateway 快速修復指南

## 📌 當前狀況

✅ **已完成**:
- Supabase Docker 環境運行正常
- 資料庫 schema 已設置完成
- 環境變數已配置

⚠️ **待修復**: 有幾個 Go 程式碼編譯錯誤需要修復

## 🔧 修復步驟（5分鐘內完成）

### 步驟 1: 刪除重複的 sqrt 函數

**檔案**: `services/multimodal_search_service.go` 第 463-475 行

**刪除這段程式碼**:
```go
// sqrt 計算平方根（簡單實作）
func sqrt(x float64) float64 {
	if x == 0 {
		return 0
	}

	// 使用牛頓法近似計算
	guess := x / 2
	for i := 0; i < 10; i++ {
		guess = (guess + x/guess) / 2
	}
	return guess
}
```

**原因**: 這個函數在 `services/image_similarity_search.go` 中已有定義

---

### 步驟 2: 修復 BatchProcessStatus 結構

**檔案**: `models/media.go`

找到 `BatchProcessStatus` 結構（大約第 107 行），確認是否包含以下欄位：

```go
type BatchProcessStatus struct {
	BatchID      string                 `json:"batch_id"`
	Status       string                 `json:"status"`  // ← 需要這個欄位
	TotalFiles   int                    `json:"total_files"`
	Processed    int                    `json:"processed"`
	Successful   int                    `json:"successful"`
	Failed       int                    `json:"failed"`
	Errors       []BatchError           `json:"errors"`  // ← 需要這個欄位
	StartTime    time.Time              `json:"start_time"`
	EndTime      *time.Time             `json:"end_time,omitempty"`
	Results      []BatchProcessResult   `json:"results"`
}
```

如果缺少 `Status` 或 `Errors` 欄位，請添加。

---

### 步驟 3: 修復 BatchProcessRequest

**檔案**: `models/media.go`

找到 `BatchProcessRequest` 結構（大約第 82 行），確認是否包含 `Files` 欄位：

```go
type BatchProcessRequest struct {
	PageID      string   `json:"page_id"`
	FolderPath  string   `json:"folder_path"`
	Files       []string `json:"files"`         // ← 需要這個欄位
	Tags        []string `json:"tags"`
	AutoAnalyze bool     `json:"auto_analyze"`
	AutoEmbed   bool     `json:"auto_embed"`
	Concurrency int      `json:"concurrency"`
}
```

---

### 步驟 4: 編譯測試

執行以下命令：

```bash
cd /Users/youlinhsieh/Documents/ink-gateway
go build -o bin/ink-gateway main.go
```

如果成功，您會看到沒有錯誤訊息，並且 `bin/ink-gateway` 檔案會被創建。

---

## 🚀 啟動服務

編譯成功後，啟動服務：

```bash
./bin/ink-gateway
```

您應該會看到類似的輸出：
```
No .env file found, using system environment variables
Semantic Text Processor starting...
Server starting on :8081...
```

---

## 🧪 測試連接

### 測試 1: Health Check

```bash
curl http://localhost:8081/health
```

預期輸出:
```json
{"status":"ok"}
```

### 測試 2: 資料庫連接

```bash
curl -X GET http://localhost:8081/api/v1/chunks
```

預期輸出: JSON 格式的 chunks 列表（可能為空）

---

## 💾 快速資料庫測試

插入測試資料：

```bash
docker exec -i supabase-db psql -U postgres -d ink_gateway << 'EOF'
-- 插入一個測試頁面
INSERT INTO chunks (contents, is_page)
VALUES ('測試頁面', true)
RETURNING chunk_id, contents, is_page, created_time;

-- 查看所有資料
SELECT chunk_id, contents, is_page, is_tag, created_time
FROM chunks
ORDER BY created_time DESC
LIMIT 10;
EOF
```

---

## 📝 完整編譯錯誤清單

如果您不想手動修復，以下是完整的錯誤列表：

1. **重複定義錯誤**:
   - `sqrt` 函數在 `services/multimodal_search_service.go` 和 `services/image_similarity_search.go`

2. **結構欄位缺失錯誤**:
   - `BatchProcessStatus` 缺少 `Status` 和 `Errors` 欄位
   - `BatchProcessRequest` 缺少 `Files` 欄位

---

## 🆘 如果還是無法編譯

### 選項 A: 檢查是否有預編譯檔案

```bash
# 查找可能的預編譯版本
ls -lh semantic-text-processor
ls -lh bin/

# 如果存在，直接運行
./semantic-text-processor
```

### 選項 B: 使用 Docker（如果有 Dockerfile）

```bash
docker build -t ink-gateway .
docker run -d --name ink-gateway --env-file .env -p 8081:8081 ink-gateway
```

### 選項 C: 跳過編譯，直接測試資料庫

您可以使用 Supabase REST API 直接與資料庫互動，不需要 Go 程式：

```bash
# 插入資料
curl -X POST http://localhost:8000/rest/v1/chunks \
  -H "apikey: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..." \
  -H "Content-Type: application/json" \
  -d '{"contents": "測試內容", "is_page": false}'

# 查詢資料
curl -X GET http://localhost:8000/rest/v1/chunks \
  -H "apikey: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
```

---

## 📞 需要幫助？

如果遇到問題，可以：

1. 查看 [TESTING_STATUS.md](TESTING_STATUS.md) 了解詳細狀況
2. 查看 [INTEGRATION_TESTING_GUIDE.md](INTEGRATION_TESTING_GUIDE.md) 了解完整測試流程
3. 檢查 `.env` 檔案確認配置正確
4. 查看 Docker logs: `docker logs supabase-db`

---

**估計修復時間**: 5-10 分鐘
**難度**: 簡單（複製貼上即可）
