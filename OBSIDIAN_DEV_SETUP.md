# Obsidian 插件開發環境設置

## 🚀 正確的開發實踐

### 1. 設置符號鏈接 (已完成)
```bash
# 刪除舊的插件目錄
rm -rf ~/.obsidian/plugins/obsidian-ink-plugin

# 創建符號鏈接到開發目錄
ln -sf "$(pwd)/obsidian-ink-plugin" ~/.obsidian/plugins/obsidian-ink-plugin
```

### 2. 啟動開發模式 (請手動執行)
```bash
cd obsidian-ink-plugin
npm run dev
```

這會啟動 esbuild 的 watch 模式，自動監控文件變更並重新編譯。

### 3. 開發工作流程

#### 即時重載設置：
1. **啟動 watch 模式**：在終端中運行 `npm run dev`
2. **修改代碼**：任何 TypeScript 文件的變更都會自動重新編譯
3. **重載插件**：在 Obsidian 中按 `Cmd+R` 或使用 Developer Console

#### Developer Console 快捷操作：
打開 Developer Console (`Cmd+Opt+I`) 並執行：

```javascript
// 重載插件
app.plugins.disablePlugin('obsidian-ink-plugin');
app.plugins.enablePlugin('obsidian-ink-plugin');

// 檢查插件狀態
console.log(app.plugins.plugins['obsidian-ink-plugin']);

// 檢查設置
console.log(app.plugins.plugins['obsidian-ink-plugin'].settings);
```

### 4. 調試技巧

#### 添加調試日誌：
```typescript
// 在代碼中添加
console.log('[Ink Plugin Debug]', 'Your debug message', data);
```

#### 檢查編譯狀態：
```bash
# 檢查編譯後的文件時間戳
ls -la obsidian-ink-plugin/main.js

# 搜索特定內容確認修復
grep "API key is recommended" obsidian-ink-plugin/main.js
```

### 5. 版本管理

#### 更新版本號：
```bash
cd obsidian-ink-plugin
npm run version
```

或手動編輯：
- `manifest.json` - 更新 version 字段
- `package.json` - 更新 version 字段

### 6. 當前修復驗證

執行以下命令確認修復已生效：

```bash
# 檢查 API key 驗證修復
grep -A 2 -B 2 "API key is recommended" obsidian-ink-plugin/main.js

# 檢查 URL 設置
grep "localhost:8081" obsidian-ink-plugin/main.js

# 檢查版本號
grep "version" obsidian-ink-plugin/manifest.json
```

## 🔧 現在請執行：

1. **在新終端中啟動開發模式**：
   ```bash
   cd obsidian-ink-plugin
   npm run dev
   ```

2. **在 Obsidian 中重載插件**：
   - 按 `Cmd+R` 重載整個應用
   - 或在 Developer Console 中執行重載命令

3. **測試修復**：
   - 進入插件設置
   - 嘗試保存空的 API key
   - 檢查 Google Drive 連結

## 📝 開發模式的優勢：

- ✅ 文件變更自動重新編譯
- ✅ 包含 source map 便於調試
- ✅ 即時看到代碼變更效果
- ✅ 不需要手動複製文件
- ✅ 支持 TypeScript 錯誤檢查

這樣的開發環境讓你可以快速迭代和測試修改！