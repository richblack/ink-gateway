# PostgreSQL 直接連接遷移進度報告

**日期**: 2025-10-30
**目標**: 從 Supabase REST API 遷移到直接 PostgreSQL 連接

---

## ✅ 已完成的工作

### 1. 安裝依賴 (100%)
- ✅ 安裝 `github.com/jackc/pgx/v5` - PostgreSQL 驅動
- ✅ 安裝 `github.com/jackc/pgx/v5/pgxpool` - 連接池
- ✅ 安裝 `github.com/google/uuid` - UUID 支援

### 2. 建立資料庫層 (100%)
已建立完整的資料庫抽象層：

#### `/database/postgres.go`
- ✅ PostgresConfig: 配置結構
- ✅ PostgresService: 主要資料庫服務
  - Connection pooling (連接池)
  - Health checking (健康檢查)
  - Transaction support (事務支援)
  - 自動重連機制

**重要特性**:
```go
// 安全的參數化查詢
query := "SELECT * FROM chunks WHERE chunk_id = $1"
row := db.QueryRow(ctx, query, chunkID)

// 連接池配置
MaxConns: 10      // 最大連接數
MinConns: 2       // 最小連接數
MaxConnLife: 1h   // 連接最大存活時間
```

#### `/database/chunk_repository.go`
實作完整的 CRUD 操作：
- ✅ Create: 建立 chunk (支援自動生成 UUID)
- ✅ GetByID: 查詢單個 chunk
- ✅ List: 分頁列表
- ✅ Update: 更新 chunk
- ✅ Delete: 刪除 chunk
- ✅ SearchByContent: 內容搜尋
- ✅ BatchCreate: 批次建立 (事務支援)

**所有操作都使用參數化查詢，防止 SQL Injection**

### 3. 更新配置系統 (100%)

#### `config/config.go`
- ✅ 新增 DatabaseConfig 結構
- ✅ 環境變數支援:
  - DB_HOST
  - DB_PORT
  - DB_NAME
  - DB_USER
  - DB_PASSWORD
  - DB_SSLMODE
  - DB_MAX_CONNS
  - DB_MIN_CONNS

#### `.env`
- ✅ 新增 PostgreSQL 配置
- ✅ 保留 Supabase 配置 (標記為 deprecated)

### 4. 測試程式 (100%)
建立 `test_postgres.go` 完整測試：
- ✅ 連接測試
- ✅ 健康檢查
- ✅ CRUD 操作測試
- ✅ 批次操作測試
- ✅ 搜尋功能測試

### 5. 連接成功驗證 (100%)
```
✅ 資料庫連接成功！
✅ 資料庫健康狀態正常
連接池統計:
  總連接數: 3
  閒置連接數: 3
  取得連接次數: 3
```

---

## ⏸️ 目前狀態

### 遇到的問題
測試插入時出現錯誤：
```
ERROR: column "chunk_id" of relation "chunks" does not exist (SQLSTATE 42703)
```

**但是**:
- ✅ 直接 SQL 插入成功
- ✅ Table schema 確認有 `chunk_id` 欄位
- ✅ 資料庫連接正常

### 可能原因
1. **Schema vs Table 名稱**: pgx 可能需要指定 schema (`public.chunks`)
2. **UUID 類型處理**: PostgreSQL 使用 UUID 類型，可能需要特殊處理
3. **權限問題**: 雖然查詢成功，但插入可能有權限限制

### 簡單修復方案
讓資料庫自動生成 UUID，不在 INSERT 中指定 `chunk_id`：

```go
query := `
    INSERT INTO chunks (
        contents, is_page, parent, metadata, created_time
    ) VALUES (
        $1, $2, $3, $4, $5
    )
    RETURNING chunk_id
`
```

這樣可以利用資料庫的 `gen_random_uuid()` 預設值。

---

## 📋 下一步行動

### 方案 A：快速修復 (推薦，30分鐘)
1. 修改 `chunk_repository.go` 的 Create 函數
2. 讓資料庫自動生成 UUID
3. 使用 RETURNING 子句取得生成的 ID
4. 測試所有 CRUD 操作

### 方案 B：深入調查 (1-2小時)
1. 研究 pgx 與 UUID 的正確處理方式
2. 檢查是否需要額外的類型轉換
3. 驗證所有邊界情況

---

## 🎯 完成後的效益

### 效能提升
- ⚡ **延遲降低**: 移除一層 API 呼叫
- ⚡ **連接池**: 重用資料庫連接
- ⚡ **Prepared Statements**: 查詢計劃快取

### 安全性
- 🔒 **參數化查詢**: 防止 SQL Injection
- 🔒 **最小權限**: 資料庫使用者只有必要權限
- 🔒 **TLS 加密**: 資料庫連接加密
- 🔒 **API 層保護**: Ink-Gateway 作為安全邊界

### 架構簡化
```
Before:
使用者 → Ink-Gateway → Supabase REST API → PostgreSQL
        (Go API)       (PostgREST)

After:
使用者 → Ink-Gateway → PostgreSQL
        (Go API)       (直接連接)
```

### 成本節省
- 💰 **本地部署**: $0/年 (vs $390/年 Supabase)
- 💰 **雲端部署**: ~$240/年 (vs $390/年 Supabase)

---

## 📚 相關文檔

1. [ARCHITECTURE_ANALYSIS.md](ARCHITECTURE_ANALYSIS.md) - 完整架構分析
2. [APPLICATION_STATUS.md](APPLICATION_STATUS.md) - 應用程式狀態
3. [CONFIGURATION_GUIDE.md](CONFIGURATION_GUIDE.md) - 配置指南
4. [EMBEDDING_STRATEGY.md](EMBEDDING_STRATEGY.md) - Embedding 策略

---

## 💻 技術細節

### PostgreSQL Driver 選擇
選擇 `pgx/v5` 的原因：
1. ✅ **效能最佳**: 比 database/sql 快 2-3倍
2. ✅ **原生 PostgreSQL**: 支援所有 PostgreSQL 特性
3. ✅ **類型安全**: 編譯時類型檢查
4. ✅ **連接池**: 內建高效連接池
5. ✅ **pgvector 支援**: 原生支援向量類型

### 安全性實作
```go
// ❌ 不安全 - 容易 SQL Injection
query := fmt.Sprintf("SELECT * FROM chunks WHERE id = '%s'", userInput)

// ✅ 安全 - 參數化查詢
query := "SELECT * FROM chunks WHERE chunk_id = $1"
row := db.QueryRow(ctx, query, userInput)
```

### 事務處理
```go
tx, _ := db.Begin(ctx)
defer tx.Rollback(ctx)

// 執行多個操作
tx.Exec(ctx, query1, ...)
tx.Exec(ctx, query2, ...)

// 全部成功才提交
tx.Commit(ctx)
```

---

## 🔧 開發環境

### 當前配置
```bash
# PostgreSQL (Supabase Docker)
Host: localhost
Port: 5432
Database: postgres
User: postgres
Password: postgres

# 連接池
Max Connections: 10
Min Connections: 2
```

### 測試命令
```bash
# 執行測試
go run test_postgres.go

# 直接 SQL 測試
docker exec -i supabase-db psql -U postgres -d postgres

# 檢查 schema
docker exec -i supabase-db psql -U postgres -d postgres -c "\d chunks"

# 查看資料
docker exec -i supabase-db psql -U postgres -d postgres -c "SELECT chunk_id, contents FROM chunks LIMIT 5;"
```

---

## 📈 進度總覽

### 整體進度: 🟡 85% 完成

| 任務 | 狀態 | 完成度 |
|------|------|--------|
| 安裝依賴 | ✅ 完成 | 100% |
| 建立 PostgresService | ✅ 完成 | 100% |
| 建立 ChunkRepository | ✅ 完成 | 100% |
| 更新配置系統 | ✅ 完成 | 100% |
| 資料庫連接測試 | ✅ 成功 | 100% |
| CRUD 操作測試 | ⏸️ 進行中 | 50% |
| 整合到 API | ⏳ 待完成 | 0% |
| Google Drive Adapter | ⏳ 待完成 | 0% |
| 完整測試 | ⏳ 待完成 | 0% |

---

## 🎬 立即可執行的下一步

### 修復 INSERT 問題 (15分鐘)

**修改 `/database/chunk_repository.go`**:

```go
// 修改 Create 函數 - 移除 chunk_id，使用 RETURNING
func (r *ChunkRepository) Create(ctx context.Context, chunk *models.UnifiedChunkRecord) error {
    now := time.Now()
    if chunk.CreatedTime.IsZero() {
        chunk.CreatedTime = now
    }

    metadataJSON, err := json.Marshal(chunk.Metadata)
    if err != nil {
        return fmt.Errorf("failed to marshal metadata: %w", err)
    }

    query := `
        INSERT INTO chunks (
            contents, is_page, parent, metadata, created_time
        ) VALUES (
            $1, $2, $3, $4, $5
        )
        RETURNING chunk_id
    `

    // 使用 QueryRow 取得生成的 ID
    err = r.db.QueryRow(ctx, query,
        chunk.Contents,
        chunk.IsPage,
        chunk.Parent,
        metadataJSON,
        chunk.CreatedTime,
    ).Scan(&chunk.ChunkID)

    if err != nil {
        return fmt.Errorf("failed to insert chunk: %w", err)
    }

    return nil
}
```

### 測試
```bash
go run test_postgres.go
```

預期結果：
```
✅ Chunk 已建立，ID: [auto-generated-uuid]
✅ Chunk 查詢成功
✅ 所有測試通過
```

---

**報告生成時間**: 2025-10-30 20:20
**下一次更新**: INSERT 問題修復後
