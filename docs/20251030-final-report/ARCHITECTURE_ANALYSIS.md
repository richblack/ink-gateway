# Ink-Gateway 架構分析與建議

**日期**: 2025-10-30
**分析師**: Claude
**目的**: 評估當前架構設計的合理性與優化方案

---

## 📊 當前架構分析

### 架構圖

```
┌─────────────────────────────────────────────────────────────┐
│                       使用者應用程式                          │
│              (Obsidian Plugin / Web App)                    │
└─────────────────────┬───────────────────────────────────────┘
                      │ HTTP/REST API
                      ▼
┌─────────────────────────────────────────────────────────────┐
│                    Ink-Gateway (Go)                         │
│  ┌────────────────────────────────────────────────────┐    │
│  │  HTTP Handlers                                     │    │
│  │  - Chunk Management                                │    │
│  │  - Search (Semantic, Multimodal)                   │    │
│  │  - Media Processing                                │    │
│  │  - Tag Management                                  │    │
│  └────────────────────┬───────────────────────────────┘    │
│                       │                                     │
│  ┌────────────────────▼───────────────────────────────┐    │
│  │  Business Logic Services                           │    │
│  │  - Embedding Service (Gemini)                      │    │
│  │  - Image Analysis (Gemini Vision)                  │    │
│  │  - CLIP Embedding (External API)                   │    │
│  │  - Storage Adapter Pattern                         │    │
│  └────────────────────┬───────────────────────────────┘    │
│                       │                                     │
│  ┌────────────────────▼───────────────────────────────┐    │
│  │  Data Access Layer                                 │    │
│  │  - Supabase Client (目前使用)                      │    │
│  │  - 直接 PostgreSQL Driver (可選)                  │    │
│  └────────────────────┬───────────────────────────────┘    │
└────────────────────────┼───────────────────────────────────┘
                         │
        ┌────────────────┴────────────────┐
        │                                 │
        ▼                                 ▼
┌──────────────────┐          ┌──────────────────────┐
│  Supabase Stack  │          │   External Services  │
│  ┌────────────┐  │          │  ┌────────────────┐  │
│  │ REST API   │  │          │  │ Gemini API     │  │
│  │ (PostgREST)│  │          │  │ - Embedding    │  │
│  └──────┬─────┘  │          │  │ - Vision       │  │
│         │        │          │  └────────────────┘  │
│  ┌──────▼─────┐  │          │  ┌────────────────┐  │
│  │ PostgreSQL │  │          │  │ CLIP API       │  │
│  │ + pgvector │  │          │  │ (External)     │  │
│  └────────────┘  │          │  └────────────────┘  │
│                  │          │                      │
│  ┌────────────┐  │          │  ┌────────────────┐  │
│  │  Storage   │  │          │  │ Google Drive   │  │
│  │  (圖片)     │  │          │  │ (可選)          │  │
│  └────────────┘  │          │  └────────────────┘  │
└──────────────────┘          └──────────────────────┘
```

---

## 🤔 您的核心問題

### 問題 1: Ink-Gateway 是否就是 Supabase 的角色？

**答案：是的，而且更強大！**

#### 當前的「疊床架屋」問題

```
使用者 → Ink-Gateway (API) → Supabase Client → Supabase REST API → PostgreSQL
        ^^^^^^^^^^^^^^^^       ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
        您的 API               實際上只是另一個 API 層
```

這確實是「API 呼叫 API」的架構，存在以下問題：

1. **額外的延遲**: 每個請求都要經過兩層 API
2. **故障點增加**: Supabase REST API 故障會影響整個系統
3. **配置複雜度**: 需要同時維護兩套 API 配置
4. **功能重複**: Ink-Gateway 和 Supabase 都提供類似的功能

#### Ink-Gateway 的**真正價值**

Ink-Gateway **不只是** Supabase 的替代品，它提供了：

1. **領域專用邏輯**
   - ✅ 語義搜尋（pgvector + Gemini embeddings）
   - ✅ 多模態搜尋（文字 + 圖片）
   - ✅ 層級筆記結構（chunk hierarchy）
   - ✅ 標籤系統（chunk tags）
   - ✅ 模板系統（template/slot）
   - ✅ CLIP 圖片 embedding

2. **業務邏輯封裝**
   - Supabase 只是通用的 API Gateway
   - Ink-Gateway 提供**筆記特定的語義**

3. **靈活的儲存策略**
   - 已實作 **Storage Adapter Pattern**
   - 支援多種後端（Local, Supabase, Google Drive, NAS）

---

## 💡 建議方案

### 方案 A：混合架構（推薦）⭐

**資料庫**: 直接連接 PostgreSQL
**圖片儲存**: Google Drive / Google Photos
**Auth**: 自行實作或使用第三方（Clerk, Auth0）

#### 優點

1. ✅ **簡化架構**: 移除中間的 Supabase REST API 層
2. ✅ **降低延遲**: 直接資料庫連接更快
3. ✅ **更好的控制**: 完全掌握資料庫查詢優化
4. ✅ **圖片方便管理**: Google Photos 有完整的查看界面
5. ✅ **成本優化**:
   - Google Drive: 15GB 免費
   - Google Photos: 無限制（壓縮質量）
6. ✅ **安全性**: 透過 Ink-Gateway 的 API 層保護資料庫

#### 實作細節

```go
// 資料庫連接
import "github.com/jackc/pgx/v5/pgxpool"

// 使用 connection pool
pool, err := pgxpool.New(context.Background(),
    "postgres://user:pass@localhost:5432/ink_gateway?pool_max_conns=10")

// 參數化查詢防止 SQL Injection
row := pool.QueryRow(ctx,
    "SELECT chunk_id, contents FROM chunks WHERE chunk_id = $1",
    chunkID)
```

**安全措施**:
- ✅ Connection pooling
- ✅ Prepared statements（參數化查詢）
- ✅ 最小權限原則（資料庫使用者權限）
- ✅ TLS/SSL 加密連接
- ✅ API 層驗證和授權

#### 圖片儲存實作

程式碼中已經預留了 Google Drive 支援：

```go
// models/media.go (已存在)
const (
    StorageTypeLocal        StorageType = "local"
    StorageTypeSupabase     StorageType = "supabase"
    StorageTypeGoogleDrive  StorageType = "google_drive"   // ✅ 已定義
    StorageTypeGooglePhotos StorageType = "google_photos"  // ✅ 已定義
    StorageTypeNAS          StorageType = "nas"
)

// storage_factory.go 第 64 行（註解）
// f.adapters[models.StorageTypeGoogleDrive] = func(...) { ... }
```

**需要實作的部分**:

1. **Google Drive Adapter** (約 200 行程式碼)
   ```go
   // services/google_drive_storage_adapter.go
   type GoogleDriveStorageAdapter struct {
       service *drive.Service
       folderID string
   }

   func (g *GoogleDriveStorageAdapter) Upload(ctx context.Context, file io.Reader, filename string) (string, error) {
       // 上傳到 Google Drive
       // 回傳 file ID
   }

   func (g *GoogleDriveStorageAdapter) GetURL(fileID string) (string, error) {
       // 回傳可分享的 URL
       return fmt.Sprintf("https://drive.google.com/file/d/%s/view", fileID), nil
   }
   ```

2. **Google Photos Adapter** (類似但使用 Photos API)

---

### 方案 B：完全使用 Supabase

**資料庫**: Supabase REST API
**圖片儲存**: Supabase Storage
**Auth**: Supabase Auth

#### 優點

1. ✅ 統一平台
2. ✅ 原設計不需修改
3. ✅ Supabase 處理 Auth 和 Storage

#### 缺點

1. ❌ **疊床架屋**: API → API → DB
2. ❌ **目前無法使用**: REST API 配置問題
3. ❌ **圖片查看不便**: 沒有像 Google Photos 的界面
4. ❌ **未來成本**: Supabase 付費方案
5. ❌ **依賴單一服務**: Supabase 故障影響全部

#### 修復步驟（如果選擇此方案）

需要找出為什麼 PostgREST 返回 404，可能原因：
- Kong 路由配置問題
- PostgREST schema exposure 設定
- 資料庫權限配置
- 需要 Kiro 提供原始 Supabase 配置

---

### 方案 C：本地優先架構

**資料庫**: SQLite + pgvector extension
**圖片儲存**: 本地檔案系統 + 選擇性雲端備份
**Auth**: 本地 token 或不需要（單用戶）

#### 優點

1. ✅ **最快速度**: 所有資料都在本地
2. ✅ **完全離線**: 不依賴網路
3. ✅ **隱私最佳**: 資料不離開裝置
4. ✅ **成本最低**: 無雲端費用

#### 缺點

1. ❌ **單裝置**: 難以跨裝置同步
2. ❌ **無協作**: 單一使用者
3. ❌ **備份責任**: 需要自行處理備份

---

## 🎯 針對您的使用情境的建議

### 使用情境分析

根據您的描述：
- ✅ **筆記應用**: 分段儲存 chunks
- ✅ **語義搜尋**: 需要 embedding
- ✅ **圖片**: 需要儲存和檢視
- ⚠️ **多裝置**: 未明確提及
- ⚠️ **協作**: 未明確提及
- ✅ **未來雲端部署**: 有規劃

### 推薦：**方案 A（混合架構）**

#### 理由

1. **安全性充足**
   ```
   使用者 → Ink-Gateway API (您的控制層) → PostgreSQL
           ^^^^^^^^^^^^^^^^^^^^^^
           這一層就是您的安全保護！
   ```

   - Ink-Gateway 扮演 **API Gateway** 角色
   - 所有查詢都經過您的驗證和授權邏輯
   - 使用參數化查詢防止 SQL Injection
   - 資料庫只允許 Ink-Gateway 連接（firewall rules）

2. **圖片管理更好**
   - Google Photos 有完整的相簿界面
   - 可以在手機、網頁查看
   - 自動備份和同步
   - 無限儲存（壓縮畫質）

3. **架構更清晰**
   ```
   使用者
     ↓
   Ink-Gateway (業務邏輯 + 安全層)
     ↓                    ↓
   PostgreSQL        Google Drive/Photos
   (結構化資料)        (圖片檔案)
   ```

4. **未來擴展性**
   - 輕鬆切換到雲端 PostgreSQL（AWS RDS, Google Cloud SQL）
   - Storage Adapter 可以隨時切換
   - 可以加入 Auth (Clerk, Auth0) 而不依賴 Supabase

5. **效能更好**
   - 少一層 API 呼叫
   - 直接資料庫連接延遲更低
   - 可以使用 prepared statements 和 connection pooling

---

## 📋 實作計畫（方案 A）

### 階段 1: 資料庫連接切換（2-3 小時）

1. **安裝 PostgreSQL Driver**
   ```bash
   go get github.com/jackc/pgx/v5
   go get github.com/jackc/pgx/v5/pgxpool
   ```

2. **建立 Database Service**
   ```go
   // services/postgres_service.go
   type PostgresService struct {
       pool *pgxpool.Pool
   }

   func NewPostgresService(connString string) (*PostgresService, error) {
       pool, err := pgxpool.New(context.Background(), connString)
       if err != nil {
           return nil, err
       }
       return &PostgresService{pool: pool}, nil
   }
   ```

3. **替換 Supabase Client**
   - 修改 `services/chunk_service.go`
   - 修改 `services/search_service.go`
   - 使用參數化查詢

4. **測試**
   - 建立 chunk
   - 查詢 chunk
   - 搜尋功能

### 階段 2: Google Drive Integration（3-4 小時）

1. **設定 Google Cloud Project**
   - 啟用 Google Drive API
   - 建立 OAuth 2.0 credentials
   - 下載 credentials.json

2. **實作 Google Drive Adapter**
   ```go
   // services/google_drive_storage_adapter.go
   type GoogleDriveStorageAdapter struct {
       service  *drive.Service
       folderID string
   }

   func (g *GoogleDriveStorageAdapter) Upload(
       ctx context.Context,
       file io.Reader,
       filename string,
   ) (string, error) {
       f := &drive.File{
           Name:    filename,
           Parents: []string{g.folderID},
       }

       res, err := g.service.Files.
           Create(f).
           Media(file).
           Context(ctx).
           Do()

       if err != nil {
           return "", err
       }

       // 設定為公開或取得分享連結
       return res.Id, nil
   }
   ```

3. **註冊到 Factory**
   ```go
   // services/storage_factory.go
   f.adapters[models.StorageTypeGoogleDrive] = func(config map[string]interface{}) (MediaStorageAdapter, error) {
       credentialsPath := config["credentials_path"].(string)
       folderID := config["folder_id"].(string)

       return NewGoogleDriveStorageAdapter(credentialsPath, folderID)
   }
   ```

4. **測試**
   - 上傳圖片
   - 取得 URL
   - 驗證可存取

### 階段 3: Auth 整合（選配，未來）

1. **選擇 Auth 提供商**
   - Clerk (推薦，簡單)
   - Auth0 (功能強大)
   - 自行實作 JWT

2. **加入 Middleware**
   ```go
   func AuthMiddleware(next http.Handler) http.Handler {
       return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
           token := r.Header.Get("Authorization")

           // 驗證 token
           if !validateToken(token) {
               http.Error(w, "Unauthorized", http.StatusUnauthorized)
               return
           }

           next.ServeHTTP(w, r)
       })
   }
   ```

---

## 🔐 安全性比較

### Supabase Client 安全性

```go
// Supabase Client 其實也是這樣做的：
// 1. 連接 REST API（HTTP 請求）
// 2. REST API 連接 PostgreSQL
// 3. 使用 Row Level Security (RLS) 保護

// 您的顧慮：直接用 SQLAlchemy
// 問題：可能寫出不安全的查詢
```

### 直接 PostgreSQL 安全性

```go
// 使用 pgx driver 是安全的：

// ❌ 不安全（SQL Injection）
query := fmt.Sprintf("SELECT * FROM chunks WHERE id = '%s'", userInput)

// ✅ 安全（參數化查詢）
query := "SELECT * FROM chunks WHERE chunk_id = $1"
row := pool.QueryRow(ctx, query, userInput)

// ✅ 更安全（加上驗證）
if !isValidUUID(userInput) {
    return errors.New("invalid chunk ID")
}
row := pool.QueryRow(ctx, "SELECT * FROM chunks WHERE chunk_id = $1", userInput)
```

**關鍵差異**:
- Python SQLAlchemy: 容易誤用（寫 raw SQL）
- Go pgx: **強制**使用參數化查詢（設計上更安全）
- Ink-Gateway API 層: **就是您的安全邊界**

### 額外保護措施

1. **連接層級**
   ```
   PostgreSQL 設定:
   - 只允許 Ink-Gateway 的 IP 連接
   - 使用 TLS/SSL 加密
   - 設定連接數限制
   ```

2. **應用層級**
   ```go
   // Ink-Gateway 中實作：
   - 輸入驗證
   - Rate limiting
   - API key/JWT 驗證
   - 審計日誌（audit log）
   ```

3. **資料庫層級**
   ```sql
   -- 最小權限原則
   CREATE ROLE ink_gateway WITH LOGIN PASSWORD 'xxx';
   GRANT SELECT, INSERT, UPDATE, DELETE ON chunks TO ink_gateway;
   GRANT SELECT, INSERT, UPDATE, DELETE ON chunk_tags TO ink_gateway;
   -- 不給 DROP, ALTER 等權限
   ```

---

## 💰 成本比較（年度估算）

### 情境：3,000 筆記/月，每筆 1 張圖片

| 項目 | 方案 A (混合) | 方案 B (Supabase) |
|------|---------------|-------------------|
| **資料庫** | 免費（本地）<br/>或 $20/月（雲端 PG） | $25/月（Pro plan） |
| **儲存** | 免費（15GB）<br/>或 $1.99/月（100GB） | $0.021/GB = $7.56/月 |
| **Embedding** | $0.07/年（Gemini） | 同左 |
| **總計（本地）** | **$0.07/年** | **$390/年** |
| **總計（雲端）** | **$264/年** | **$390/年** |

**節省**: $126-390/年

---

## ✅ 結論與建議

### 給您的建議

1. **架構選擇**: **方案 A（混合架構）** ⭐⭐⭐⭐⭐
   - 理由：平衡了安全性、效能、成本和使用體驗

2. **圖片儲存**: **Google Photos** ⭐⭐⭐⭐⭐
   - 理由：無限儲存、完整查看界面、跨裝置同步

3. **資料庫連接**: **直接 PostgreSQL (pgx driver)** ⭐⭐⭐⭐⭐
   - 理由：Go 的參數化查詢設計本身就很安全
   - Ink-Gateway 就是您的安全層，不需要再加一層 Supabase

4. **Auth**: **先不實作，未來再加** ⭐⭐⭐⭐
   - 理由：本地使用不需要，未來雲端部署時再整合 Clerk/Auth0

### 疊床架屋的判斷

**您的直覺是對的！**

```
❌ 疊床架屋:
使用者 → Ink-Gateway → Supabase → PostgreSQL
        (API)         (API)

✅ 合理架構:
使用者 → Ink-Gateway → PostgreSQL
        (業務邏輯 + 安全層)
```

**Ink-Gateway 本身就應該是**:
- ✅ API Gateway
- ✅ 業務邏輯層
- ✅ 安全控制層
- ✅ 資料存取層

**不需要 Supabase 做同樣的事**

### 下一步行動

如果您決定採用**方案 A**，我可以協助：

1. ✅ 實作 PostgreSQL direct connection
2. ✅ 實作 Google Drive Storage Adapter
3. ✅ 移除 Supabase Client 依賴
4. ✅ 測試完整功能

**預估時間**: 4-6 小時即可完成核心功能

---

## 📚 參考資料

### PostgreSQL 安全最佳實踐
- [OWASP SQL Injection Prevention](https://cheatsheetseries.owasp.org/cheatsheets/SQL_Injection_Prevention_Cheat_Sheet.html)
- [pgx Documentation](https://github.com/jackc/pgx)

### Google Drive API
- [Google Drive API Guide](https://developers.google.com/drive/api/guides/about-sdk)
- [Google Photos Library API](https://developers.google.com/photos)

### Go 安全編程
- [Go Security Best Practices](https://github.com/OWASP/Go-SCP)

---

**報告生成時間**: 2025-10-30 19:52
**作者**: Claude
**版本**: 1.0
