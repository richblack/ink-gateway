# 故障排除指南

## 目錄

1. [常見問題快速診斷](#常見問題快速診斷)
2. [安裝和設定問題](#安裝和設定問題)
3. [連線和同步問題](#連線和同步問題)
4. [效能問題](#效能問題)
5. [功能異常問題](#功能異常問題)
6. [錯誤代碼參考](#錯誤代碼參考)
7. [診斷工具](#診斷工具)
8. [聯繫技術支援](#聯繫技術支援)

## 常見問題快速診斷

### 問題檢查清單

在深入診斷之前，請先檢查以下基本項目：

- [ ] Obsidian 版本是否符合需求（≥ 1.0.0）
- [ ] 插件是否已正確安裝並啟用
- [ ] 網路連線是否正常
- [ ] Ink-Gateway 伺服器是否運行中
- [ ] API Key 是否正確設定
- [ ] 是否有其他插件衝突

### 快速修復方法

| 問題症狀 | 快速修復 |
|---------|----------|
| 插件無法載入 | 重新啟動 Obsidian |
| 同步失敗 | 檢查網路連線和 API Key |
| 搜尋無結果 | 等待內容索引完成 |
| AI 聊天無回應 | 檢查 Gateway 連線狀態 |
| 效能緩慢 | 清除快取並重新啟動 |

## 安裝和設定問題

### 插件安裝失敗

#### 症狀
- 無法從社群插件商店安裝
- 手動安裝後插件不出現在列表中
- 安裝過程中出現錯誤訊息

#### 可能原因
1. Obsidian 版本過舊
2. 插件檔案損壞
3. 權限不足
4. 磁碟空間不足

#### 解決步驟

**步驟 1: 檢查 Obsidian 版本**
```
設定 → 關於 → 檢查版本號
```
確保版本 ≥ 1.0.0

**步驟 2: 手動安裝**
```bash
# 下載最新版本
wget https://github.com/your-repo/obsidian-ink-plugin/releases/latest/download/obsidian-ink-plugin.zip

# 解壓到插件目錄
unzip obsidian-ink-plugin.zip -d /path/to/vault/.obsidian/plugins/
```

**步驟 3: 檢查檔案權限**
```bash
# 確保檔案可讀取
chmod -R 755 /path/to/vault/.obsidian/plugins/obsidian-ink-plugin/
```

### 插件啟用失敗

#### 症狀
- 插件出現在列表中但無法啟用
- 啟用後立即停用
- 出現 "Failed to load plugin" 錯誤

#### 解決步驟

**步驟 1: 檢查依賴**
確認所有必要檔案都存在：
- `main.js`
- `manifest.json`
- `styles.css`

**步驟 2: 檢查 manifest.json**
```json
{
  "id": "obsidian-ink-plugin",
  "name": "Ink Plugin",
  "version": "1.0.0",
  "minAppVersion": "1.0.0",
  "description": "Obsidian plugin for Ink-Gateway integration"
}
```

**步驟 3: 查看控制台錯誤**
1. 開啟開發者工具 (`Ctrl+Shift+I`)
2. 查看 Console 標籤中的錯誤訊息
3. 根據錯誤訊息進行相應處理

### 設定載入失敗

#### 症狀
- 設定頁面空白或無法開啟
- 設定變更無法儲存
- 出現 "Settings validation failed" 錯誤

#### 解決步驟

**步驟 1: 重置設定**
```bash
# 刪除設定檔案
rm /path/to/vault/.obsidian/plugins/obsidian-ink-plugin/data.json

# 重新啟動 Obsidian
```

**步驟 2: 手動修復設定**
建立基本設定檔案：
```json
{
  "inkGatewayUrl": "http://localhost:8080",
  "apiKey": "",
  "autoSync": true,
  "syncInterval": 5000,
  "cacheEnabled": true,
  "debugMode": false
}
```

## 連線和同步問題

### 無法連接到 Ink-Gateway

#### 症狀
- 連線測試失敗
- 同步操作超時
- 出現 "Network error" 或 "Connection refused" 錯誤

#### 診斷步驟

**步驟 1: 檢查 Gateway 狀態**
```bash
# 測試 Gateway 是否運行
curl -I http://your-gateway-url/health

# 預期回應: HTTP/1.1 200 OK
```

**步驟 2: 檢查網路連線**
```bash
# 測試網路連通性
ping your-gateway-host

# 測試埠號是否開放
telnet your-gateway-host 8080
```

**步驟 3: 檢查防火牆設定**
- 確認防火牆允許對應埠號的連線
- 檢查企業網路是否有代理伺服器設定

#### 解決方法

**方法 1: 更新 Gateway URL**
```
設定 → Ink Plugin → Gateway URL
確保格式正確: http://hostname:port
```

**方法 2: 配置代理伺服器**
如果在企業網路環境：
```json
{
  "inkGatewayUrl": "http://proxy-server:port/gateway",
  "proxySettings": {
    "enabled": true,
    "host": "proxy.company.com",
    "port": 8080
  }
}
```

### API 認證失敗

#### 症狀
- 出現 "Unauthorized" 或 "Invalid API Key" 錯誤
- 連線測試顯示認證失敗
- 同步操作被拒絕

#### 解決步驟

**步驟 1: 驗證 API Key**
```bash
# 測試 API Key 有效性
curl -H "Authorization: Bearer YOUR_API_KEY" \
     http://your-gateway-url/api/health
```

**步驟 2: 重新生成 API Key**
1. 登入 Ink-Gateway 管理介面
2. 前往 API Keys 頁面
3. 撤銷舊的 API Key
4. 生成新的 API Key
5. 更新插件設定

**步驟 3: 檢查 Key 格式**
確保 API Key：
- 沒有多餘的空格
- 完整複製（通常為 32-64 字元）
- 沒有特殊字元編碼問題

### 同步衝突

#### 症狀
- 出現 "Sync conflict detected" 警告
- 內容版本不一致
- 部分變更遺失

#### 解決策略

**策略 1: 自動解決**
```typescript
// 插件會自動嘗試合併非衝突變更
// 檢查自動解決結果
```

**策略 2: 手動解決**
1. 開啟衝突解決介面
2. 比較不同版本的內容
3. 選擇要保留的版本
4. 確認合併結果

**策略 3: 強制同步**
```
命令面板 → "Force Sync All Content"
注意：這會覆蓋本地變更
```

## 效能問題

### Obsidian 運行緩慢

#### 症狀
- 開啟檔案延遲
- 輸入文字卡頓
- 搜尋回應緩慢

#### 診斷方法

**檢查記憶體使用**
```javascript
// 在開發者控制台執行
console.log('Memory usage:', performance.memory);
```

**檢查快取大小**
```
設定 → Ink Plugin → 診斷 → 快取統計
```

#### 最佳化步驟

**步驟 1: 調整快取設定**
```json
{
  "cacheEnabled": true,
  "cacheMaxSize": 50,  // 減少快取大小
  "cacheTTL": 300000   // 5 分鐘過期
}
```

**步驟 2: 調整同步頻率**
```json
{
  "autoSync": true,
  "syncInterval": 10000  // 增加到 10 秒
}
```

**步驟 3: 清理快取**
```
命令面板 → "Clear All Cache"
```

### 搜尋效能問題

#### 症狀
- 搜尋結果載入緩慢
- 搜尋介面無回應
- 記憶體使用量持續增加

#### 最佳化方法

**方法 1: 限制搜尋範圍**
```typescript
// 使用日期範圍限制
searchQuery.dateRange = {
  start: new Date('2024-01-01'),
  end: new Date()
};

// 使用標籤篩選
searchQuery.tags = ['specific-tag'];
```

**方法 2: 啟用搜尋快取**
```json
{
  "searchCacheEnabled": true,
  "searchCacheSize": 100,
  "searchCacheTTL": 600000  // 10 分鐘
}
```

**方法 3: 分頁載入結果**
```typescript
// 設定分頁大小
searchQuery.pageSize = 20;
searchQuery.page = 1;
```

## 功能異常問題

### AI 聊天功能異常

#### 症狀
- AI 無回應或回應緩慢
- 聊天歷史遺失
- 出現 "AI service unavailable" 錯誤

#### 診斷步驟

**步驟 1: 檢查 AI 服務狀態**
```bash
curl http://your-gateway-url/api/ai/health
```

**步驟 2: 檢查聊天歷史**
```
設定 → Ink Plugin → 診斷 → 聊天統計
```

**步驟 3: 測試 AI 連線**
```
AI 聊天視窗 → 設定 → 測試連線
```

#### 解決方法

**重置聊天狀態**
```
命令面板 → "Reset AI Chat State"
```

**清除聊天歷史**
```
AI 聊天視窗 → 設定 → 清除歷史
```

### 模板功能異常

#### 症狀
- 模板無法套用
- 插槽值無法填入
- 模板解析錯誤

#### 常見問題和解決

**問題 1: 模板語法錯誤**
```markdown
<!-- 錯誤語法 -->
{title}  <!-- 單大括號 -->
{{title  <!-- 缺少結束括號 -->

<!-- 正確語法 -->
{{title}}
{{date:YYYY-MM-DD}}
{{content:required}}
```

**問題 2: 插槽名稱衝突**
```markdown
<!-- 避免使用保留字 -->
{{class}}    <!-- 錯誤：class 是保留字 -->
{{category}} <!-- 正確 -->
```

**問題 3: 模板實例損壞**
```
命令面板 → "Repair Template Instances"
```

### 搜尋功能異常

#### 症狀
- 搜尋無結果或結果不準確
- 搜尋介面載入失敗
- 語義搜尋效果差

#### 解決步驟

**步驟 1: 重建搜尋索引**
```
命令面板 → "Rebuild Search Index"
```

**步驟 2: 檢查內容索引狀態**
```
設定 → Ink Plugin → 診斷 → 索引統計
```

**步驟 3: 調整搜尋參數**
```json
{
  "searchSettings": {
    "semanticThreshold": 0.7,  // 降低語義搜尋門檻
    "maxResults": 50,          // 增加結果數量
    "includeContent": true     // 包含內容預覽
  }
}
```

## 錯誤代碼參考

### 連線錯誤 (1xxx)

| 代碼 | 說明 | 解決方法 |
|------|------|----------|
| 1001 | 網路連線超時 | 檢查網路連線和 Gateway 狀態 |
| 1002 | DNS 解析失敗 | 檢查 Gateway URL 是否正確 |
| 1003 | 連線被拒絕 | 檢查防火牆和埠號設定 |
| 1004 | SSL/TLS 錯誤 | 檢查憑證設定 |

### 認證錯誤 (2xxx)

| 代碼 | 說明 | 解決方法 |
|------|------|----------|
| 2001 | API Key 無效 | 重新生成並設定 API Key |
| 2002 | API Key 過期 | 更新 API Key |
| 2003 | 權限不足 | 檢查 API Key 權限設定 |
| 2004 | 認證格式錯誤 | 檢查 API Key 格式 |

### 同步錯誤 (3xxx)

| 代碼 | 說明 | 解決方法 |
|------|------|----------|
| 3001 | 同步衝突 | 使用衝突解決工具 |
| 3002 | 資料格式錯誤 | 檢查內容格式 |
| 3003 | 同步超時 | 調整同步間隔 |
| 3004 | 儲存空間不足 | 清理快取或聯繫管理員 |

### 功能錯誤 (4xxx)

| 代碼 | 說明 | 解決方法 |
|------|------|----------|
| 4001 | AI 服務不可用 | 檢查 AI 服務狀態 |
| 4002 | 搜尋索引損壞 | 重建搜尋索引 |
| 4003 | 模板解析失敗 | 檢查模板語法 |
| 4004 | 快取錯誤 | 清除快取 |

### 系統錯誤 (5xxx)

| 代碼 | 說明 | 解決方法 |
|------|------|----------|
| 5001 | 記憶體不足 | 重新啟動 Obsidian |
| 5002 | 檔案系統錯誤 | 檢查檔案權限 |
| 5003 | 插件初始化失敗 | 重新安裝插件 |
| 5004 | 設定檔案損壞 | 重置設定 |

## 診斷工具

### 內建診斷功能

#### 系統資訊檢查
```
命令面板 → "Show System Information"
```
顯示：
- Obsidian 版本
- 插件版本
- 作業系統資訊
- 記憶體使用狀況

#### 連線診斷
```
設定 → Ink Plugin → 診斷 → 連線測試
```
測試項目：
- Gateway 連通性
- API 認證狀態
- 網路延遲
- 服務可用性

#### 效能分析
```
命令面板 → "Performance Analysis"
```
分析內容：
- 記憶體使用趨勢
- 同步效能統計
- 搜尋回應時間
- 快取命中率

### 日誌收集

#### 啟用詳細日誌
```json
{
  "debugMode": true,
  "logLevel": "debug",
  "logToFile": true
}
```

#### 查看日誌
```
設定 → Ink Plugin → 診斷 → 查看日誌
```

#### 匯出診斷報告
```
命令面板 → "Export Diagnostic Report"
```
報告包含：
- 系統資訊
- 錯誤日誌
- 效能統計
- 設定資訊（已脫敏）

### 外部診斷工具

#### 網路診斷
```bash
# 測試連線
ping gateway-host

# 測試埠號
telnet gateway-host 8080

# 測試 HTTP 回應
curl -v http://gateway-host:8080/health
```

#### 效能監控
```bash
# 監控 Obsidian 程序
top -p $(pgrep Obsidian)

# 監控記憶體使用
ps aux | grep Obsidian
```

## 聯繫技術支援

### 提交問題前的準備

#### 收集必要資訊
1. **系統資訊**
   - 作業系統版本
   - Obsidian 版本
   - 插件版本

2. **問題描述**
   - 具體症狀
   - 重現步驟
   - 預期行為
   - 實際行為

3. **診斷資料**
   - 錯誤訊息
   - 日誌檔案
   - 診斷報告
   - 螢幕截圖

#### 問題報告模板

```markdown
## 問題描述
[簡要描述問題]

## 環境資訊
- 作業系統: [Windows/macOS/Linux] [版本]
- Obsidian 版本: [版本號]
- 插件版本: [版本號]
- Gateway 版本: [版本號]

## 重現步驟
1. [步驟 1]
2. [步驟 2]
3. [步驟 3]

## 預期行為
[描述預期的正確行為]

## 實際行為
[描述實際發生的行為]

## 錯誤訊息
```
[貼上完整的錯誤訊息]
```

## 附加資訊
- 診斷報告: [附加檔案]
- 螢幕截圖: [附加圖片]
- 相關設定: [相關設定資訊]
```

### 支援管道

#### GitHub Issues
- **URL**: https://github.com/your-repo/obsidian-ink-plugin/issues
- **適用**: 錯誤報告、功能請求
- **回應時間**: 1-3 工作日

#### 社群論壇
- **URL**: https://forum.obsidian.md
- **適用**: 使用問題、經驗分享
- **回應時間**: 數小時到 1 天

#### 電子郵件支援
- **Email**: support@ink-gateway.com
- **適用**: 緊急問題、私人資訊
- **回應時間**: 24 小時內

#### 即時聊天
- **平台**: Discord/Slack
- **適用**: 快速問題、即時協助
- **服務時間**: 工作日 9:00-18:00 (UTC+8)

### 緊急問題處理

#### 定義緊急問題
- 插件完全無法使用
- 資料遺失風險
- 安全性問題
- 影響生產環境

#### 緊急聯繫方式
1. **電子郵件**: urgent@ink-gateway.com
2. **電話**: +886-xxx-xxx-xxx
3. **即時訊息**: 標註 @urgent 在社群頻道

#### 緊急回應承諾
- **確認回應**: 2 小時內
- **初步診斷**: 4 小時內
- **解決方案**: 24 小時內

---

**版本**: 1.0.0  
**最後更新**: 2024年1月  
**文件語言**: 繁體中文