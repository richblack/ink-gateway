# 多模態 MCP 系統需求文件

## 簡介

本文件描述將 Ink-Gateway 擴展為全面性多模態知識管理系統的需求規格。該系統將讓使用者能夠透過統一的 MCP (Model Context Protocol) 介面，在多個應用程式中無縫管理、搜尋和使用文字與圖片內容。核心願景是建立一個圖片如文字般可搜尋、儲存方式彈性且雲端優先、與 Obsidian、VSCode、Slide Generator 等工具無縫整合的系統。

## 需求規格

### 需求 1: 統一圖片管理與雲端儲存

**使用者故事:** 作為筆記書寫者，我希望將圖片拖放到 Obsidian 就能自動上傳到雲端儲存並進行 AI 分析，這樣我就能從任何裝置存取圖片而不用擔心本地路徑相依性問題。

#### 接受標準

1. WHEN 使用者拖放圖片到 Obsidian THEN 系統 SHALL 自動上傳圖片到設定的雲端儲存服務
2. WHEN 圖片上傳時 THEN 系統 SHALL 計算 SHA256 雜湊值用於去重和唯一識別
3. WHEN 圖片處理時 THEN 系統 SHALL 記錄本地路徑和雲端 ID 的雙重路徑存取
4. WHEN 相同圖片（相同雜湊值）再次上傳時 THEN 系統 SHALL 重複使用現有雲端檔案而非建立重複檔案
5. WHEN 使用者切換雲端儲存提供商時 THEN 系統 SHALL 只需要設定變更而不影響現有資料參照

### 需求 2: AI 驅動的圖片分析與語義索引

**使用者故事:** 作為筆記書寫者，我希望系統能自動分析我的圖片並生成詳細描述，這樣我就能用「系統架構」等關鍵字搜尋圖片，即使檔名完全不同。

#### 接受標準

1. WHEN 圖片上傳時 THEN 系統 SHALL 使用 GPT-4 Vision 或 Claude 3 生成詳細的 AI 描述
2. WHEN AI 分析完成時 THEN 系統 SHALL 建立 CLIP 向量嵌入（512 維度）用於語義搜尋
3. WHEN AI 分析完成時 THEN 系統 SHALL 基於圖片內容生成相關標籤
4. WHEN 使用者用關鍵字搜尋時 THEN 系統 SHALL 找到 AI 描述包含語義相關概念的圖片
5. WHEN 執行語義搜尋時 THEN 系統 SHALL 回傳相似度分數高於 0.7 閾值的結果

### 需求 3: 批次圖片處理與資料夾監控

**使用者故事:** 作為擁有累積圖片資產的筆記書寫者，我希望能批次處理包含數百張圖片的資料夾，這樣我就能快速整理現有的視覺資源而不需要手動逐一處理。

#### 接受標準

1. WHEN 使用者選擇資料夾進行監控時 THEN 系統 SHALL 掃描並顯示所有支援的圖片格式檔案
2. WHEN 批次處理開始時 THEN 系統 SHALL 顯示上傳、分析和索引階段的進度指示器
3. WHEN 批次處理執行中時 THEN 系統 SHALL 支援暫停和恢復功能
4. WHEN 批次處理完成時 THEN 系統 SHALL 提供詳細報告顯示成功數量、失敗數量和錯誤詳情
5. WHEN 處理大量圖片集時 THEN 系統 SHALL 使用並行處理來優化效能

### 需求 4: MCP 協議整合實現通用存取

**使用者故事:** 作為開發者，我希望在 VSCode 寫程式時能透過 Claude 查詢我的知識庫，這樣我就能存取筆記而不需要切換應用程式和中斷工作流程。

#### 接受標準

1. WHEN MCP 客戶端呼叫 ink_search_chunks 時 THEN 系統 SHALL 支援文字、圖片或混合搜尋模式
2. WHEN MCP 客戶端呼叫 ink_create_chunk 時 THEN 系統 SHALL 建立可包含圖片的知識塊
3. WHEN MCP 客戶端呼叫 ink_analyze_image 時 THEN 系統 SHALL 分析圖片並回傳 AI 描述和向量 ID
4. WHEN VSCode 或 Claude Desktop 連接到 MCP 伺服器時 THEN 系統 SHALL 提供完整的知識庫存取能力
5. WHEN MCP 工具執行時 THEN 系統 SHALL 在 200ms 內回應（P95 百分位數）

### 需求 5: Slide Generator 整合實現自動化簡報

**使用者故事:** 作為簡報製作者，我希望 Slide Generator 能自動從我的知識庫找到相關圖片和文字，這樣我就能生成包含我自己視覺資產的簡報而不需要手動搜尋。

#### 接受標準

1. WHEN Slide Generator 呼叫 ink_get_images_for_slides 時 THEN 系統 SHALL 基於文字內容推薦相關圖片
2. WHEN 推薦圖片時 THEN 系統 SHALL 為每個建議提供相關度分數和推薦理由
3. WHEN Slide Generator 請求內容配對時 THEN 系統 SHALL 同時提供文字內容和相關圖片
4. WHEN 生成簡報時 THEN 系統 SHALL 確保圖片 URL 可被 Slide Generator 存取
5. WHEN 推薦結果不足時 THEN 系統 SHALL 提供替代建議或相似內容

### 需求 6: 進階多模態搜尋功能

**使用者故事:** 作為筆記書寫者，我希望搜尋「資料庫架構」就能找到相關圖表，即使圖片檔名完全無關，這樣我就能用自然語言查詢快速定位視覺資訊。

#### 接受標準

1. WHEN 使用者輸入文字查詢時 THEN 系統 SHALL 同時搜尋文字內容和圖片 AI 描述
2. WHEN 使用者上傳參考圖片時 THEN 系統 SHALL 支援圖片對圖片的相似度搜尋
3. WHEN 執行混合搜尋時 THEN 系統 SHALL 支援文字和圖片組件間的可調整權重
4. WHEN 顯示搜尋結果時 THEN 系統 SHALL 指示匹配類型（文字向量、圖片向量或混合）
5. WHEN 使用者點擊搜尋結果時 THEN 系統 SHALL 導航到包含該圖片的原始筆記

### 需求 7: 可插拔儲存架構實現未來彈性

**使用者故事:** 作為系統管理者，我希望未來能輕鬆切換圖片儲存方式（從 Google Drive 到 NAS 或其他雲端服務），而不需要重新處理所有現有資料。

#### 接受標準

1. WHEN 系統初始化時 THEN 系統 SHALL 根據設定載入適當的 StorageAdapter
2. WHEN 切換儲存服務時 THEN 系統 SHALL 在資料庫中維持一致的 storage_id 格式
3. WHEN 新增儲存服務支援時 THEN 系統 SHALL 只需要實作 StorageAdapter 介面
4. WHEN 儲存服務故障時 THEN 系統 SHALL 提供降級機制（如本地快取）
5. WHEN 在儲存服務間遷移時 THEN 系統 SHALL 提供資料遷移工具

### 需求 8: 跨平台整合與工作流程連續性

**使用者故事:** 作為知識工作者，我希望能在 Obsidian 做筆記、VSCode 開發、Claude Desktop 進行 AI 協助時無縫使用我的知識庫，這樣無論使用哪個工具我的工作流程都不會被中斷。

#### 接受標準

1. WHEN 在任何支援的應用程式中工作時 THEN 系統 SHALL 提供對相同知識庫的一致存取
2. WHEN 在一個應用程式中建立內容時 THEN 系統 SHALL 立即在其他應用程式中提供該內容
3. WHEN 從不同應用程式搜尋時 THEN 系統 SHALL 回傳一致且同步的結果
4. WHEN 應用程式離線時 THEN 系統 SHALL 提供對最近使用內容的快取存取
5. WHEN 在應用程式間切換時 THEN 系統 SHALL 在適當情況下維持上下文和搜尋歷史