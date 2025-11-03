# 🔄 MCP Server 重啟後測試指南

**編譯時間**: 2025-10-31 08:33:11
**狀態**: ✅ 編譯成功，等待重啟測試

---

## ✅ 已完成的修復

### 1. Metadata NULL 處理問題
**檔案**: [services/unified_chunk_impl.go](../services/unified_chunk_impl.go)

**問題**:
- 資料庫 metadata 欄位為 NULL 時，無法掃描到 `map[string]interface{}` 類型
- 錯誤: `sql: Scan error... unsupported Scan, storing driver.Value type <nil>`

**修復**:
```go
// 使用 []byte 來掃描 JSONB（可處理 NULL）
var metadataBytes []byte
err := s.db.QueryRowContext(ctx, query, chunkID).Scan(
    // ... 其他欄位
    &metadataBytes,  // ✅ 可以處理 NULL
    // ...
)

// 安全地解析或初始化
if len(metadataBytes) > 0 {
    json.Unmarshal(metadataBytes, &chunk.Metadata)
} else {
    chunk.Metadata = make(map[string]interface{})
}
```

---

## 🧪 重啟後測試步驟

### 測試 1: 驗證 GetChunk 修復
```
使用工具: ink_get_chunk
參數: chunk_id = "15510996-34a7-433c-9695-aba935a33dc3"
期望結果: 成功返回 chunk 資料，不再有 SQL scan 錯誤
```

### 測試 2: 測試其他已創建的 chunks
```
chunk_id: "910a90df-0bc4-48b6-bc9d-6d448bef2398" (MCP Server 架構)
chunk_id: "363459a6-e18f-4266-a5b9-ca392b8cb781" (PostgreSQL 配置)
```

### 測試 3: 測試搜尋功能
```
使用工具: ink_search_text
參數: query = "PostgreSQL"
已知問題: SearchChunks 方法尚未實作
狀態: ⏳ 待實作
```

---

## ⏳ 待實作項目

### 1. SearchChunks 方法實作
**檔案**: [services/unified_chunk_impl.go](../services/unified_chunk_impl.go)
**方法**: `func (s *unifiedChunkService) SearchChunks(ctx context.Context, query *models.SearchQuery) (*models.SearchResult, error)`

**目前狀態**: 返回 "not implemented" 錯誤

**需求**:
- 實作全文搜尋功能
- 支援 PostgreSQL 的 `to_tsvector` 和 `to_tsquery`
- 或使用 `ILIKE` 進行簡單搜尋
- 支援分頁和限制

**優先級**: 高（影響 `ink_search_text` 工具）

---

## 📊 目前 MCP 工具狀態

| 工具名稱 | 狀態 | 說明 |
|---------|------|------|
| `ink_create_text_chunk` | ✅ 正常 | 已測試，可創建 chunks |
| `ink_get_chunk` | 🔄 待測試 | 修復完成，等待重啟驗證 |
| `ink_search_text` | ❌ 未實作 | 需要 SearchChunks 方法 |

---

## 📝 已創建的測試數據

1. **Page Chunk** (15510996-34a7-433c-9695-aba935a33dc3)
   - 內容: ink-gateway 專案介紹
   - is_page: true

2. **Sub-chunk 1** (910a90df-0bc4-48b6-bc9d-6d448bef2398)
   - 內容: MCP Server 架構說明
   - parent: 15510996-34a7-433c-9695-aba935a33dc3

3. **Sub-chunk 2** (363459a6-e18f-4266-a5b9-ca392b8cb781)
   - 內容: PostgreSQL 配置資訊
   - parent: 15510996-34a7-433c-9695-aba935a33dc3

---

## 🎯 下一步建議

1. **立即**: 重啟 Claude Code
2. **然後**: 執行測試 1-2 驗證 GetChunk 修復
3. **接著**: 實作 SearchChunks 方法
4. **最後**: 完整測試搜尋功能

---

*最後更新: 2025-10-31 08:33*
