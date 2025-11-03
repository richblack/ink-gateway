# Requirements Document

## Introduction

這個功能是對現有 Semantic Text Processor 的重大重構，實現真正的統一資料表設計。所有內容（文字、標籤、模板、slots、頁面）都儲存在單一 chunks 表中，透過 boolean 欄位區分類型，並使用輔助表優化多對多關係的查詢效能。系統需要在大資料量（百萬級）下仍能保持毫秒級查詢響應時間。

## Requirements

### Requirement 1

**User Story:** 作為系統架構師，我希望所有內容都儲存在統一的 chunks 表中，以便系統能夠容納任何新的資料類型而無需修改資料表結構。

#### Acceptance Criteria

1. WHEN 系統初始化 THEN 系統 SHALL 創建單一 chunks 主表儲存所有內容
2. WHEN 新增任何類型的內容 THEN 系統 SHALL 將其作為 chunk 儲存在同一張表中
3. WHEN 區分不同類型的 chunk THEN 系統 SHALL 使用 boolean 欄位（is_page, is_tag, is_template, is_slot）
4. WHEN 需要擴展新的內容類型 THEN 系統 SHALL 支援在不修改表結構的情況下添加新類型

### Requirement 2

**User Story:** 作為前端開發者，我希望系統支援多對多的標籤關係，以便一個 chunk 可以有多個標籤，一個標籤也可以被多個 chunks 使用。

#### Acceptance Criteria

1. WHEN 為 chunk 添加標籤 THEN 系統 SHALL 支援添加多個標籤 chunk_id
2. WHEN 查詢某個標籤的使用情況 THEN 系統 SHALL 回傳所有使用該標籤的 chunks
3. WHEN 查詢某個 chunk 的標籤 THEN 系統 SHALL 回傳該 chunk 的所有標籤
4. WHEN 標籤關係發生變化 THEN 系統 SHALL 同時更新主表和輔助表以保持一致性

### Requirement 3

**User Story:** 作為系統管理員，我希望系統在大資料量下仍能保持高效能查詢，以便支援生產環境的使用需求。

#### Acceptance Criteria

1. WHEN 資料量達到百萬級 THEN 系統 SHALL 在標籤查詢中保持毫秒級響應時間
2. WHEN 執行複雜的多標籤查詢 THEN 系統 SHALL 使用輔助表優化查詢效能
3. WHEN 查詢層級結構 THEN 系統 SHALL 使用專用輔助表避免遞迴查詢效能問題
4. WHEN 系統負載增加 THEN 系統 SHALL 透過索引和快取策略維持查詢效能

### Requirement 4

**User Story:** 作為資料分析師，我希望系統提供快速的標籤查詢功能，以便能夠高效地根據標籤篩選和分析內容。

#### Acceptance Criteria

1. WHEN 根據單一標籤查詢 chunks THEN 系統 SHALL 使用 chunk_tags 輔助表提供毫秒級查詢
2. WHEN 根據多個標籤查詢 chunks THEN 系統 SHALL 支援 AND 和 OR 邏輯操作
3. WHEN 查詢標籤的統計資訊 THEN 系統 SHALL 回傳標籤的使用次數和相關 chunks 數量
4. WHEN 執行標籤相關的聚合查詢 THEN 系統 SHALL 使用預建索引優化查詢效能

### Requirement 5

**User Story:** 作為前端開發者，我希望系統支援高效的層級結構查詢，以便快速渲染 bullet-point 風格的內容層級。

#### Acceptance Criteria

1. WHEN 查詢某個 chunk 的所有子項目 THEN 系統 SHALL 使用 chunk_hierarchy 輔助表避免遞迴查詢
2. WHEN 查詢某個 chunk 的完整路徑 THEN 系統 SHALL 回傳從根節點到該 chunk 的完整層級路徑
3. WHEN 移動 chunk 的層級位置 THEN 系統 SHALL 同時更新主表和層級輔助表
4. WHEN 查詢特定深度的層級結構 THEN 系統 SHALL 支援深度限制和分頁查詢

### Requirement 6

**User Story:** 作為系統開發者，我希望系統提供查詢快取機制，以便減少重複查詢的資料庫負載並提升響應速度。

#### Acceptance Criteria

1. WHEN 執行常用查詢 THEN 系統 SHALL 將結果快取在記憶體中
2. WHEN 資料發生變更 THEN 系統 SHALL 自動清除相關的快取項目
3. WHEN 快取命中 THEN 系統 SHALL 直接回傳快取結果而不查詢資料庫
4. WHEN 快取過期 THEN 系統 SHALL 自動重新查詢並更新快取

### Requirement 7

**User Story:** 作為資料庫管理員，我希望系統使用適當的索引策略，以便在大資料量下維持查詢效能。

#### Acceptance Criteria

1. WHEN 系統初始化 THEN 系統 SHALL 創建所有必要的資料庫索引
2. WHEN 執行標籤查詢 THEN 系統 SHALL 使用 GIN 索引優化 JSONB 欄位查詢
3. WHEN 執行全文搜尋 THEN 系統 SHALL 使用全文搜尋索引提升查詢效能
4. WHEN 監控查詢效能 THEN 系統 SHALL 記錄慢查詢並提供效能分析資訊

### Requirement 8

**User Story:** 作為 API 使用者，我希望系統提供批量操作功能，以便高效地處理大量資料的插入和更新。

#### Acceptance Criteria

1. WHEN 批量插入 chunks THEN 系統 SHALL 使用批量 INSERT 操作提升效能
2. WHEN 批量更新標籤關係 THEN 系統 SHALL 同時更新主表和輔助表
3. WHEN 批量操作失敗 THEN 系統 SHALL 使用事務確保資料一致性
4. WHEN 批量操作完成 THEN 系統 SHALL 清除相關的快取項目

### Requirement 9

**User Story:** 作為系統監控人員，我希望系統提供效能監控功能，以便及時發現和解決效能問題。

#### Acceptance Criteria

1. WHEN 查詢執行時間超過閾值 THEN 系統 SHALL 記錄慢查詢日誌
2. WHEN 監控系統效能 THEN 系統 SHALL 提供查詢統計和效能指標
3. WHEN 資料庫負載過高 THEN 系統 SHALL 提供告警機制
4. WHEN 分析效能問題 THEN 系統 SHALL 提供詳細的查詢執行計劃

### Requirement 10

**User Story:** 作為系統維護人員，我希望系統支援資料一致性檢查，以便確保主表和輔助表之間的資料同步。

#### Acceptance Criteria

1. WHEN 資料不一致時 THEN 系統 SHALL 提供資料一致性檢查工具
2. WHEN 發現資料不一致 THEN 系統 SHALL 提供自動修復機制
3. WHEN 執行資料遷移 THEN 系統 SHALL 確保主表和輔助表的同步更新
4. WHEN 系統啟動時 THEN 系統 SHALL 驗證資料完整性並報告任何問題
