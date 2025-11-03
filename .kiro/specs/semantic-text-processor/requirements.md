# Requirements Document

## Introduction

這個功能是一個基於 Go 語言的 web 應用程式，專門處理文字內容的語義分析和儲存。系統接收前端傳送的文字內容，透過 LLM 進行語義切割，透過 Supabase API 將結果儲存到多個資料庫系統中，包括 PostgreSQL、PGVector 和 Apache AGE，並提供完整的 web server 功能。

## Requirements

### Requirement 1

**User Story:** 作為一個前端開發者，我希望能夠透過 HTTP API 傳送文字內容到後端，以便系統能夠處理這些文字資料。

#### Acceptance Criteria

1. WHEN 前端透過 HTTP POST 請求傳送文字內容 THEN 系統 SHALL 接收並驗證文字內容格式
2. WHEN 文字內容為空或格式不正確 THEN 系統 SHALL 回傳適當的錯誤訊息
3. WHEN 文字內容超過系統限制大小 THEN 系統 SHALL 回傳檔案大小錯誤訊息

### Requirement 2

**User Story:** 作為系統管理員，我希望系統能夠使用 LLM 將文字內容依照語義切割成有意義的 chunks，以便後續處理和分析。

#### Acceptance Criteria

1. WHEN 系統接收到有效的文字內容 THEN 系統 SHALL 呼叫 LLM API 進行語義分析
2. WHEN LLM 完成語義分析 THEN 系統 SHALL 將文字切割成邏輯相關的 chunks
3. IF LLM API 呼叫失敗 THEN 系統 SHALL 記錄錯誤並回傳適當的錯誤訊息
4. WHEN chunks 產生完成 THEN 系統 SHALL 驗證每個 chunk 的完整性和格式

### Requirement 3

**User Story:** 作為資料分析師，我希望系統能夠透過 Supabase API 將 chunks 儲存到 PostgreSQL 資料庫中，以便進行結構化查詢和分析。

#### Acceptance Criteria

1. WHEN chunks 產生完成 THEN 系統 SHALL 透過 Supabase API 將 chunks 寫入 PostgreSQL 資料庫
2. WHEN 呼叫 Supabase API THEN 系統 SHALL 使用適當的認證和預定義的 table schema
3. IF Supabase API 呼叫失敗 THEN 系統 SHALL 重試請求並記錄錯誤
4. WHEN 資料寫入成功 THEN 系統 SHALL 從 Supabase 回應中取得 chunk ID 供後續參考

### Requirement 4

**User Story:** 作為機器學習工程師，我希望系統能夠為 chunks 產生向量嵌入並透過 Supabase API 儲存到 PGVector 中，以便進行相似性搜尋和向量運算。

#### Acceptance Criteria

1. WHEN chunks 儲存到 PostgreSQL 完成 THEN 系統 SHALL 呼叫 Embeddings model API
2. WHEN Embeddings model 回傳向量資料 THEN 系統 SHALL 透過 Supabase API 將向量寫入 PGVector 擴充功能
3. WHEN 透過 Supabase API 寫入向量 THEN 系統 SHALL 建立 chunk ID 與向量的關聯
4. IF Embeddings API 或 Supabase API 呼叫失敗 THEN 系統 SHALL 記錄錯誤並提供重試機制

### Requirement 5

**User Story:** 作為知識圖譜分析師，我希望系統能夠分析 chunks 並透過 Supabase API 將知識結構寫入 Apache AGE，以便建立知識圖譜和關係分析。

#### Acceptance Criteria

1. WHEN chunks 處理完成 THEN 系統 SHALL 分析 chunks 中的知識結構和實體關係
2. WHEN 知識分析完成 THEN 系統 SHALL 透過 Supabase API 將結構化知識寫入 Apache AGE 圖形資料庫
3. WHEN 透過 Supabase API 寫入 Apache AGE THEN 系統 SHALL 建立節點和邊的關係映射
4. IF Supabase API 呼叫或 Apache AGE 操作失敗 THEN 系統 SHALL 記錄錯誤並提供重試機制

### Requirement 6

**User Story:** 作為資料庫管理員，我希望系統透過 Supabase 為三種不同的資料類型使用獨立的 tables 和 schemas，以確保資料不會互相混雜並維持資料完整性。

#### Acceptance Criteria

1. WHEN 系統初始化 THEN 系統 SHALL 驗證 Supabase 中 PostgreSQL chunks 的專用 table 存在
2. WHEN 系統初始化 THEN 系統 SHALL 驗證 Supabase 中 PGVector 向量資料的專用 table 存在
3. WHEN 系統初始化 THEN 系統 SHALL 驗證 Supabase 中 Apache AGE 知識圖譜的專用 table 存在
4. WHEN 任何資料操作執行 THEN 系統 SHALL 確保透過 Supabase API 資料只寫入對應的 table 中
5. WHEN 透過 Supabase API 操作 THEN 系統 SHALL 遵循預定義的 table schema 和約束條件

### Requirement 7

**User Story:** 作為前端開發者，我希望能夠查詢已儲存的文字內容和相關資料，以便在前端介面中顯示和管理這些資訊。

#### Acceptance Criteria

1. WHEN 前端請求文字列表 THEN 系統 SHALL 透過 Supabase API 從 PostgreSQL 回傳使用者輸入的原始文字和 chunks 列表
2. WHEN 前端請求特定文字詳情 THEN 系統 SHALL 透過 Supabase API 回傳該文字的完整資訊包括所有相關 chunks
3. WHEN 前端提供分頁參數 THEN 系統 SHALL 透過 Supabase API 支援分頁查詢並回傳總數資訊
4. IF 查詢的文字不存在 THEN 系統 SHALL 回傳適當的 404 錯誤訊息

### Requirement 8

**User Story:** 作為使用者，我希望能夠進行語義搜尋，以便根據內容相似性找到相關的文字 chunks。

#### Acceptance Criteria

1. WHEN 前端提供搜尋查詢文字 THEN 系統 SHALL 將查詢文字轉換為向量嵌入
2. WHEN 查詢向量產生完成 THEN 系統 SHALL 透過 Supabase API 在 PGVector 中執行相似性搜尋
3. WHEN 相似性搜尋完成 THEN 系統 SHALL 回傳最相關的 chunks 及其相似度分數
4. WHEN 前端指定搜尋結果數量限制 THEN 系統 SHALL 回傳指定數量的最佳匹配結果
5. IF 沒有找到相似的內容 THEN 系統 SHALL 回傳空結果集和適當訊息

### Requirement 9

**User Story:** 作為研究人員，我希望能夠進行圖形搜尋，以便探索知識實體之間的關係和連結。

#### Acceptance Criteria

1. WHEN 前端提供實體名稱或關鍵字 THEN 系統 SHALL 透過 Supabase API 在 Apache AGE 中搜尋相關的節點
2. WHEN 節點搜尋完成 THEN 系統 SHALL 回傳節點資訊和其直接連結的關係
3. WHEN 前端請求擴展搜尋 THEN 系統 SHALL 透過 Supabase API 支援多層級的關係遍歷查詢
4. WHEN 圖形查詢執行 THEN 系統 SHALL 回傳結構化的圖形資料包括節點和邊的資訊
5. WHEN 前端指定搜尋深度限制 THEN 系統 SHALL 限制遍歷深度避免過度查詢
6. IF 找不到相關的圖形節點 THEN 系統 SHALL 回傳空的圖形結構和適當訊息

### Requirement 10

**User Story:** 作為前端開發者，我希望系統提供完整的 web server 功能，包括 HTTP 路由、中介軟體和錯誤處理，以便與前端應用程式整合。

#### Acceptance Criteria

1. WHEN 系統啟動 THEN 系統 SHALL 啟動 HTTP web server 並監聽指定埠號
2. WHEN HTTP 請求到達 THEN 系統 SHALL 透過適當的路由處理請求
3. WHEN 處理請求 THEN 系統 SHALL 應用必要的中介軟體（如 CORS、日誌記錄、驗證）
4. WHEN 發生錯誤 THEN 系統 SHALL 回傳結構化的錯誤回應
5. WHEN 系統運行 THEN 系統 SHALL 提供健康檢查端點供監控使用