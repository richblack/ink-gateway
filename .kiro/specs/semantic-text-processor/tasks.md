# Implementation Plan

- [x] 1. 建立專案基礎架構和核心介面
  - 建立 Go 專案結構，包含 models、services、handlers 和 clients 目錄
  - 定義核心資料結構和介面，建立系統邊界
  - 設定環境變數配置和基本的 HTTP server
  - _Requirements: 10.1, 10.2_

- [x] 2. 實作 Supabase 客戶端和資料模型
  - [x] 2.1 建立 Supabase 客戶端連線和基本操作
    - 實作 Supabase HTTP 客戶端，包含認證和錯誤處理
    - 建立資料庫連線測試和健康檢查功能
    - _Requirements: 3.1, 4.1, 5.1_

  - [x] 2.2 實作 chunks 表的 CRUD 操作
    - 建立 ChunkRecord 的插入、查詢、更新和刪除操作
    - 實作層級結構查詢，包含父子關係和縮排層級
    - 撰寫單元測試驗證資料操作的正確性
    - _Requirements: 3.1, 7.1, 7.2_

  - [x] 2.3 實作標籤系統的資料操作
    - 建立 chunk_tags 表的操作，支援標籤的新增和移除
    - 實作標籤查詢功能，包含根據標籤內容搜尋 chunks
    - 撰寫標籤關係的單元測試
    - _Requirements: 7.1, 7.2_

- [x] 3. 建立文字處理和 LLM 整合服務
  - [x] 3.1 實作 LLM 服務客戶端
    - 建立 LLM API 客戶端，支援文字語義切割功能
    - 實作重試機制和錯誤處理，確保服務穩定性
    - 撰寫 mock 測試驗證 LLM 整合邏輯
    - _Requirements: 2.1, 2.2, 2.3_

  - [x] 3.2 實作文字切割和處理邏輯
    - 建立文字處理服務，將 LLM 回應轉換為 chunks
    - 實作層級結構解析，支援 bullet-point 格式的文字
    - 撰寫文字處理的單元測試和整合測試
    - _Requirements: 2.1, 2.4_

- [x] 4. 實作 Embeddings 和向量搜尋功能
  - [x] 4.1 建立 Embeddings 服務整合
    - 實作 Embeddings API 客戶端，支援批量向量生成
    - 建立向量資料的 Supabase 儲存操作
    - 撰寫向量生成和儲存的測試
    - _Requirements: 4.1, 4.2, 4.3_

  - [x] 4.2 實作向量相似性搜尋
    - 建立 PGVector 相似性搜尋功能，透過 Supabase API
    - 實作搜尋結果排序和分頁功能
    - 撰寫語義搜尋的端到端測試
    - _Requirements: 8.1, 8.2, 8.3, 8.4, 8.5_

- [x] 5. 實作知識圖譜和 Apache AGE 整合
  - [x] 5.1 建立知識實體抽取功能
    - 實作從 chunks 中抽取實體和關係的邏輯
    - 建立圖形節點和邊的資料結構
    - 撰寫實體抽取的單元測試
    - _Requirements: 5.1, 5.2_

  - [x] 5.2 實作 Apache AGE 圖形資料庫整合
    - 完成 Supabase 客戶端中的圖形操作方法實作
    - 實作圖形查詢和遍歷功能
    - 撰寫圖形搜尋的測試案例
    - _Requirements: 5.2, 5.3, 9.1, 9.2, 9.3, 9.4, 9.5, 9.6_

  - [x] 5.3 完成 Supabase 客戶端缺失的方法實作
    - 完成 SearchChunks 方法的實作
    - 完成 SearchSimilar 方法中被截斷的程式碼
    - 實作向量格式轉換和 RPC 呼叫邏輯
    - _Requirements: 8.1, 8.2, 8.3_

- [x] 6. 建立模板系統和動態結構
  - [x] 6.1 實作模板創建和管理
    - 完成 Supabase 客戶端中的模板相關方法實作
    - 實作 slot 系統，支援 #slot 標記和自動結構生成
    - 撰寫模板系統的單元測試
    - _Requirements: 6.1, 6.2, 6.3, 6.4, 6.5_

  - [x] 6.2 實作模板實例化功能
    - 建立基於模板創建實例的邏輯
    - 實作 slot 值的填入和更新功能
    - 撰寫模板實例化的整合測試
    - _Requirements: 6.4, 6.5_

- [x] 7. 實作 HTTP API 和路由處理
  - [x] 7.1 建立基本的 HTTP 路由和中介軟體
    - 實作 HTTP server 和基本路由結構
    - 建立 CORS、日誌記錄和錯誤處理中介軟體
    - 撰寫 HTTP 中介軟體的測試
    - _Requirements: 10.1, 10.2, 10.3, 10.4_

  - [x] 7.1.1 建立服務層整合和依賴注入
    - 建立服務實例並注入到 HTTP server 中
    - 整合 Supabase 客戶端、LLM 服務、Embedding 服務和搜尋服務
    - 建立服務工廠和配置管理
    - _Requirements: 10.1, 10.2_

  - [x] 7.2 實作文字處理相關的 API 端點
    - 建立實際的 HTTP handlers 取代 placeholder 方法
    - 整合 TextProcessor 服務到 HTTP handlers 中
    - 實作文字提交和處理的完整流程
    - 實作文字列表和詳情查詢的 API
    - 撰寫文字 API 的整合測試
    - _Requirements: 1.1, 1.2, 1.3, 7.1, 7.2, 7.3, 7.4_

  - [x] 7.3 實作 chunks 和層級結構的 API
    - 建立實際的 chunks CRUD 操作 HTTP handlers
    - 整合 Supabase 客戶端到 HTTP handlers 中
    - 實作層級結構查詢和移動操作的 API
    - 撰寫層級結構 API 的測試
    - _Requirements: 7.1, 7.2, 7.3, 7.4_

- [x] 8. 實作搜尋和查詢 API
  - [x] 8.1 建立語義搜尋 API
    - 實作語義搜尋的 HTTP handler
    - 整合向量搜尋功能到 API 層
    - 撰寫語義搜尋 API 的測試
    - _Requirements: 8.1, 8.2, 8.3, 8.4, 8.5_

  - [x] 8.2 實作圖形搜尋和標籤查詢 API
    - 建立圖形搜尋的 HTTP handler
    - 實作標籤相關的查詢 API
    - 撰寫搜尋 API 的整合測試
    - _Requirements: 9.1, 9.2, 9.3, 9.4, 9.5, 9.6_

- [x] 9. 實作模板和標籤管理 API
  - [x] 9.1 建立模板管理的 HTTP API
    - 實作模板創建、查詢和管理的 handlers
    - 建立模板實例化的 API 端點
    - 撰寫模板 API 的測試
    - _Requirements: 6.1, 6.2, 6.3, 6.4, 6.5_

  - [x] 9.2 實作標籤系統的 HTTP API
    - 建立標籤新增、移除和查詢的 handlers
    - 實作標籤繼承和特性傳播的邏輯
    - 撰寫標籤 API 的測試
    - _Requirements: 7.1, 7.2, 7.3, 7.4_

- [x] 10. 整合測試和錯誤處理完善
  - [x] 10.1 建立完整的錯誤處理機制
    - 實作統一的錯誤類型和處理邏輯
    - 建立重試機制和降級策略
    - 撰寫錯誤處理的測試案例
    - _Requirements: 2.3, 4.4, 5.4, 10.4_

  - [x] 10.2 實作端到端整合測試
    - 建立完整的工作流程測試，從文字輸入到查詢輸出
    - 測試模板系統和標籤系統的整合功能
    - 驗證所有 API 端點的正確性和效能
    - _Requirements: 1.1, 1.2, 1.3, 2.1, 2.2, 2.3, 2.4_

- [x] 11. 效能優化和部署準備
  - [x] 11.1 實作快取和效能優化
    - 建立查詢結果的快取機制
    - 優化資料庫查詢和 API 回應時間
    - 撰寫效能測試和基準測試
    - _Requirements: 8.4, 9.5, 10.2_

  - [x] 11.2 完善日誌記錄和監控
    - 實作結構化日誌記錄系統
    - 建立健康檢查和監控指標
    - 撰寫部署和運維相關的文件
    - _Requirements: 10.3, 10.5_