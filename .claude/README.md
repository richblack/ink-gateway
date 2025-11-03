# ink-gateway 專案 MCP 設定

此專案的 MCP (Model Context Protocol) 伺服器已安裝到 Claude Code 全域設定中。

## ✅ 已安裝的 MCP 伺服器

### 1. **filesystem-ink-gateway** 📁
- **用途**: ink-gateway 專案的檔案系統操作
- **範圍**: `/Users/youlinhsieh/Documents/ink-gateway`
- **狀態**: 已加入全域 MCP 設定 (`~/.claude.json`)
- **使用**: 在 Claude Code 中自動啟用

### 2. **Context7** 📚 (全域)
- **用途**: 查詢最新程式庫文件（Go、PostgreSQL、pgx 等）
- **已存在**: 全域 MCP，所有專案可用

### 3. **Chrome DevTools** 🌐 (全域)
- **用途**: 瀏覽器自動化測試
- **已存在**: 全域 MCP，所有專案可用

## 🔍 查看 MCP 狀態

在終端機中執行：

```bash
claude mcp list
```

你應該會看到 `filesystem-ink-gateway` 在列表中。

## 🚀 如何使用

MCP 會在 Claude Code 對話中自動啟用：

1. **查詢文件**: "使用 context7 查 pgx/v5 的 JSONB 處理方法"
2. **檔案操作**: "搜尋專案中所有 PostgreSQL 相關檔案"
3. **瀏覽器測試**: "用 chrome-devtools 測試 localhost:8081"

## 📝 說明

### 為什麼不用專案級 MCP？

Claude Code 目前**只支援全域 MCP 設定**（`~/.claude.json`），不支援專案級設定（`.clauderc`）。

因此我們：
1. 保留 `.clauderc` 作為備份/參考
2. 將 ink-gateway 專用的 MCP 加入全域設定
3. 使用 `filesystem-ink-gateway` 名稱以區分不同專案

### 設定檔位置

- **全域設定**: `~/.claude.json` (實際使用的設定)
- **專案參考**: `.clauderc` (備份，未來可能支援)
- **說明文件**: `.claude/README.md` (本檔案)

## 🔄 管理 MCP

### 檢視所有 MCP
```bash
claude mcp list
```

### 啟用/停用 MCP
在 Claude Code 對話中 @-mention MCP 名稱來切換：
```
@filesystem-ink-gateway  # 切換 ink-gateway 檔案系統
```

### 編輯全域設定
```bash
code ~/.claude.json
```

## ⚠️ 注意事項

- **全域 vs 專案級**: 目前只有全域 MCP 會生效
- **已加入 .gitignore**: `.clauderc` 和 `.claude/` 不會提交到 Git
- **名稱衝突**: 使用 `filesystem-ink-gateway` 而非 `filesystem` 避免與其他專案衝突
- **自動載入**: MCP 在每次 Claude Code 啟動時自動載入

## 🔗 相關連結

- [Claude Code MCP 文件](https://docs.claude.com/en/docs/claude-code/mcp)
- [PostgreSQL 直連遷移](./docs/postgresql-migration.md)
- [專案 CLAUDE.md](../CLAUDE.md)
