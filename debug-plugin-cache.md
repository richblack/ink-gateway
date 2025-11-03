# Obsidian 插件緩存清除指南

## 已完成的步驟：

1. ✅ 更新版本號從 1.0.0 → 1.0.1
2. ✅ 完全刪除舊的插件目錄
3. ✅ 重新複製新的編譯文件
4. ✅ 驗證修復內容已包含在編譯文件中

## 現在請執行以下步驟：

### 步驟 1: 完全重啟 Obsidian
- 完全關閉 Obsidian 應用程式
- 等待 5 秒
- 重新打開 Obsidian

### 步驟 2: 檢查插件狀態
- 進入 Settings → Community plugins
- 確認 "Ink Gateway Plugin" 顯示版本 1.0.1
- 如果插件被禁用，重新啟用它

### 步驟 3: 測試設置
- 進入插件設置
- API Key 留空
- 嘗試修改任何設置（如 Google Drive Folder ID）
- 應該不會再出現 "API key is required" 錯誤

### 步驟 4: 如果仍有問題
如果還是有緩存問題，可以嘗試：

1. **清除 Obsidian 應用緩存**：
   - 關閉 Obsidian
   - 刪除 `~/Library/Application Support/obsidian/` 中的緩存文件
   - 重新打開 Obsidian

2. **手動重新載入插件**：
   - 在 Developer Console (Cmd+Opt+I) 中執行：
   ```javascript
   app.plugins.disablePlugin('obsidian-ink-plugin');
   app.plugins.enablePlugin('obsidian-ink-plugin');
   ```

## 驗證修復是否生效：

檢查這些項目：
- [ ] API Key 可以留空並保存設置
- [ ] 連接測試使用 localhost:8081 而不是 8080
- [ ] Google Drive 資料夾連結正確顯示資料夾 ID
- [ ] 版本號顯示為 1.0.1

## 技術細節：

修復內容已確認包含在編譯文件中：
- API key 驗證改為 warning: "API key is recommended for full functionality"
- 默認 URL 設為 localhost:8081
- Google Drive 連結動態更新機制已添加