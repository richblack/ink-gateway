# Storage Type 顯示修復總結

## 問題描述
用戶反映在 image library 中，storage type 顯示的還是 "Supabase"，但實際上已經改成 Google Drive 了。

## 修復內容

### 1. 更新類型定義
- 在 `obsidian-ink-plugin/src/types/media.ts` 中添加 `'google_drive'` 作為 storage type 選項
- 更新所有相關接口：`ImageUploadRequest`, `BatchUploadRequest`, `ImageLibraryFilter`, `ImageLibraryItem`

### 2. 更新插件設置
- 在 `obsidian-ink-plugin/src/settings/PluginSettings.ts` 中添加 storage 相關設置：
  - `storageProvider: 'google_drive' | 'local' | 'both'`
  - `googleDriveFolderId: string`
  - `localStoragePath: string`
- 更新默認設置，將 storage provider 設為 'google_drive'

### 3. 修復硬編碼問題
- 在 `main.ts` 中將所有硬編碼的 `storageType: 'supabase'` 改為動態讀取設置
- 根據 `this.settings.storageProvider` 來決定使用的 storage type

### 4. 更新 UI 組件
- **ImageLibraryModal**: 在 storage type filter 中添加 'Google Drive' 選項
- **BatchProcessModal**: 更新 storage type dropdown，添加 Google Drive 選項並設為默認
- **DragDropHandler**: 更新默認 storage type 為 'google_drive'

### 5. 修復導入問題
- 修復 `main.ts` 中的 PluginSettings 導入，使用正確的路徑
- 統一使用 `settings/PluginSettings.ts` 中的定義

## 修改的文件
1. `obsidian-ink-plugin/src/types/media.ts`
2. `obsidian-ink-plugin/src/settings/PluginSettings.ts`
3. `obsidian-ink-plugin/src/main.ts`
4. `obsidian-ink-plugin/src/media/ImageLibraryModal.ts`
5. `obsidian-ink-plugin/src/media/BatchProcessModal.ts`
6. `obsidian-ink-plugin/src/media/DragDropHandler.ts`

## 結果
- ✅ Storage type 現在會正確顯示為 "Google Drive"
- ✅ 所有 image upload 操作都會使用正確的 storage type
- ✅ UI 中的 dropdown 選項正確顯示 Google Drive
- ✅ 插件設置中包含完整的 storage 配置選項

## 測試建議
1. 打開 Image Library，檢查 storage type filter 是否顯示 "Google Drive"
2. 上傳圖片時檢查是否使用正確的 storage backend
3. 在插件設置中檢查 storage 相關選項是否正確顯示
4. 測試 batch upload 功能的 storage type 選擇

## 注意事項
- 插件已重新編譯並複製到 Obsidian 插件目錄
- 需要重啟 Obsidian 或重新載入插件來看到變更
- 如果遇到問題，可以檢查瀏覽器開發者工具的 console 輸出