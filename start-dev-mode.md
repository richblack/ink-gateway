# 🚀 啟動開發模式

## ✅ 設置完成
- 符號鏈接已設置到正確的 Google Drive vault 位置
- 插件現在會自動同步你的代碼變更

## 🔄 現在請執行：

### 1. 啟動 watch 模式
```bash
cd obsidian-ink-plugin
npm run dev
```

### 2. 在 Obsidian 中重載
- 完全重啟 Obsidian
- 或按 `Cmd+R` 重載
- 或在 Developer Console (`Cmd+Opt+I`) 中執行：
```javascript
app.plugins.disablePlugin('obsidian-ink-plugin');
app.plugins.enablePlugin('obsidian-ink-plugin');
```

## 📝 開發工作流程

1. **修改代碼** → `src/` 目錄中的 TypeScript 文件
2. **自動重建** → esbuild watch 會自動重新編譯
3. **重載插件** → 在 Obsidian 中重載插件看到變更

## 🔍 驗證修復

進入插件設置，測試：
- ✅ API Key 可以留空並保存
- ✅ Google Drive 資料夾連結正確顯示
- ✅ 連接測試使用 localhost:8081

## 💡 調試技巧

在代碼中添加調試日誌：
```typescript
console.log('[Ink Plugin Debug]', 'Your message', data);
```

然後在 Obsidian 的 Developer Console 中查看輸出。