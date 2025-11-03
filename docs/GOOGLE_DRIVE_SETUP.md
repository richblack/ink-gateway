# Google Drive 儲存設定指南

本指南將幫助你設定 Google Drive 作為圖片儲存後端。

## 設定步驟

### 1. 建立 Google Cloud 專案

1. 前往 [Google Cloud Console](https://console.cloud.google.com/)
2. 建立新專案或選擇現有專案
3. 啟用 Google Drive API

### 2. 建立服務帳戶

1. 在 Google Cloud Console 中，前往 "IAM & Admin" → "Service Accounts"
2. 點擊 "Create Service Account"
3. 輸入服務帳戶名稱和描述
4. 點擊 "Create and Continue"
5. 跳過角色設定（點擊 "Continue"）
6. 點擊 "Done"

### 3. 產生憑證金鑰

1. 在服務帳戶列表中，點擊你剛建立的服務帳戶
2. 前往 "Keys" 標籤
3. 點擊 "Add Key" → "Create new key"
4. 選擇 "JSON" 格式
5. 下載 JSON 檔案並重新命名為 `google-drive-credentials.json`
6. 將檔案放置在 `config/` 目錄中

### 4. 設定 Google Drive 資料夾權限

1. 前往你的 Google Drive 資料夾：
   https://drive.google.com/drive/folders/1Q5rWspN-wqjqnfV0HhfngqhMFVy4QRvl

2. 右鍵點擊資料夾 → "Share"
3. 添加你的服務帳戶電子郵件地址（從 JSON 憑證檔案中的 `client_email` 欄位）
4. 設定權限為 "Editor"
5. 點擊 "Send"

### 5. 更新環境變數

確保你的 `.env` 檔案包含以下設定：

```env
# Google Drive Storage Configuration
GOOGLE_DRIVE_ENABLED=true
GOOGLE_DRIVE_FOLDER_ID=1Q5rWspN-wqjqnfV0HhfngqhMFVy4QRvl
GOOGLE_DRIVE_CREDENTIALS_PATH=./config/google-drive-credentials.json
GOOGLE_DRIVE_BASE_URL=https://drive.google.com/file/d/

# Storage Provider
STORAGE_PROVIDER=google_drive
```

### 6. 測試設定

重啟 Ink-Gateway 服務並嘗試上傳圖片：

```bash
make run
```

## 故障排除

### 常見問題

1. **403 Forbidden 錯誤**
   - 檢查服務帳戶是否有資料夾的編輯權限
   - 確認 Google Drive API 已啟用

2. **憑證檔案錯誤**
   - 確認 JSON 檔案格式正確
   - 檢查檔案路徑是否正確

3. **資料夾 ID 錯誤**
   - 從 Google Drive URL 中正確提取資料夾 ID
   - 確認資料夾存在且可存取

### 取得資料夾 ID

從 Google Drive URL 中提取資料夾 ID：
```
https://drive.google.com/drive/folders/1Q5rWspN-wqjqnfV0HhfngqhMFVy4QRvl
                                        ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
                                        這就是資料夾 ID
```

## 安全注意事項

1. **不要將憑證檔案提交到版本控制**
   - 確保 `google-drive-credentials.json` 在 `.gitignore` 中

2. **定期輪換憑證**
   - 建議每 90 天更新一次服務帳戶金鑰

3. **最小權限原則**
   - 只給予服務帳戶必要的資料夾存取權限