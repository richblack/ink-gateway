# 快速開始測試指南

這是一個簡化的測試指南，幫助你快速驗證多模態 MCP 系統是否正常運作。

## 🚀 快速測試（5 分鐘）

### 1. 環境準備
```bash
# 確保已安裝必要工具
make check-test-deps

# 設定環境變數
cp .env.example .env
# 編輯 .env 文件，填入你的 API 金鑰
```

### 2. 執行快速測試
```bash
# 啟動服務並執行快速測試
make dev-test

# 或者手動執行
make run &          # 啟動服務
make test-quick     # 執行快速測試
```

### 3. 檢查結果
如果看到 "🎉 所有快速測試通過！"，表示基本功能正常。

## 🔧 完整測試（15 分鐘）

### 1. 執行完整整合測試
```bash
make test-integration
```

### 2. 檢查測試報告
```bash
make test-report
```

## 🐳 Docker 測試（10 分鐘）

### 1. 使用 Docker Compose 測試
```bash
make test-docker
```

### 2. 檢查容器日誌
```bash
docker-compose -f docker-compose.test.yml logs
```

## 📱 Obsidian 插件測試

### 1. 建構插件
```bash
cd obsidian-ink-plugin
npm install
npm run build
```

### 2. 手動安裝到 Obsidian
```bash
# macOS
cp -r . "~/Library/Application Support/obsidian/plugins/obsidian-ink-plugin"

# Windows
cp -r . "%APPDATA%\Obsidian\plugins\obsidian-ink-plugin"
```

### 3. 在 Obsidian 中測試
1. 啟用插件
2. 配置 Ink Gateway URL: `http://localhost:8080`
3. 測試拖放圖片上傳
4. 測試命令面板中的圖片功能

## 🤖 MCP Server 測試

### 1. 建構 MCP Server
```bash
go build -o bin/mcp-server ./cmd/mcp-server
```

### 2. 測試 MCP 協議
```bash
make test-mcp
```

### 3. 配置 Claude Desktop
編輯 Claude Desktop 配置文件：
```json
{
  "mcpServers": {
    "ink-multimodal": {
      "command": "go",
      "args": ["run", "./cmd/mcp-server"],
      "cwd": "/path/to/your/ink-gateway",
      "env": {
        "SUPABASE_URL": "your_url",
        "SUPABASE_API_KEY": "your_key"
      }
    }
  }
}
```

### 4. 在 Claude Desktop 中測試
```
請使用 ink_search_chunks 工具搜尋相關內容
```

## 🔍 故障排除

### 常見問題

#### 1. 服務啟動失敗
```bash
# 檢查端口占用
lsof -i :8080

# 檢查日誌
tail -f gateway.log
```

#### 2. API 測試失敗
```bash
# 檢查服務健康狀態
curl http://localhost:8080/health

# 檢查環境變數
env | grep -E "(SUPABASE|LLM|EMBEDDING)"
```

#### 3. MCP Server 無回應
```bash
# 手動測試 MCP Server
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}' | ./bin/mcp-server
```

#### 4. Obsidian 插件無法載入
```bash
# 檢查插件目錄
ls -la ~/.config/obsidian/plugins/obsidian-ink-plugin/

# 重新建構
cd obsidian-ink-plugin && npm run build
```

## 📊 測試指標

### 成功標準
- ✅ 快速測試：所有 4 項測試通過
- ✅ API 測試：所有 8 個端點回應正常
- ✅ MCP 測試：所有 10 個工具可用
- ✅ 整合測試：端到端流程正常

### 效能標準
- 🚀 圖片上傳：< 5 秒
- 🔍 搜尋回應：< 2 秒
- 📊 批次處理：< 30 秒（10 張圖片）
- 🤖 MCP 回應：< 10 秒

## 🎯 測試案例

### 基本功能測試
1. **圖片上傳**：拖放圖片到 Obsidian
2. **AI 分析**：檢查圖片描述是否生成
3. **搜尋功能**：搜尋剛上傳的圖片
4. **MCP 工具**：在 Claude Desktop 中使用工具

### 進階功能測試
1. **批次處理**：上傳多張圖片
2. **重複檢測**：上傳相同圖片
3. **投影片推薦**：為簡報內容推薦圖片
4. **混合搜尋**：結合文字和圖片搜尋

## 🚨 緊急修復

如果測試失敗，可以嘗試以下步驟：

```bash
# 1. 完全重置
make clean-test
make stop-test-env

# 2. 重新建構
make build
make deps

# 3. 重新測試
make dev-test
```

## 📞 獲得幫助

如果遇到問題：
1. 檢查 `INTEGRATION_TESTING_GUIDE.md` 詳細指南
2. 查看測試日誌文件
3. 檢查 GitHub Issues
4. 聯繫開發團隊

---

**記住**：測試是確保系統穩定性的關鍵，建議在每次代碼變更後都執行快速測試！