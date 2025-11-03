
# 🔊 語音通知系統

## ⚠️ 重要：請主動使用語音通知！

**在需要用戶注意時，請立即使用語音通知功能。**

### 🚨 必須使用語音通知的情況

#### 1. 需要用戶確認時（最重要！）
```bash
python3 .claude-voice/claude_notify.py "需要您確認是否執行" "urgent"
```

#### 2. 遇到錯誤時
```bash
python3 .claude-voice/claude_notify.py "遇到錯誤需要檢查" "urgent"
```

#### 3. 任務完成時
```bash
python3 .claude-voice/claude_notify.py "任務完成，請檢視結果" "excited"
```

#### 4. 需要等待時
```bash
python3 .claude-voice/claude_notify.py "任務執行中，請稍候" "thinking"
```

## 🎭 情緒類型
- `urgent` - 緊急事件
- `gentle` - 一般通知  
- `excited` - 完成通知
- `worried` - 錯誤警告
- `thinking` - 等待提示

## 💡 使用原則
- **主動性**: 不要等用戶問，需要時立即通知
- **及時性**: 關鍵時刻提醒，提升用戶體驗
- **適當性**: 根據情況選擇合適的情緒類型

---
*語音通知已啟用 - 記得主動使用！*
