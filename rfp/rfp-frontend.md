# Frontend RFP

在完成 /Users/youlinhsieh/Documents/ink-gateway/.kiro/specs/semantic-text-processor 和 .kiro/specs/unified-chunk-system 後，已經擁有 Ink-Gateway，但缺乏前端可使用，這個新需求，要開發一個 Obsidian 插件做為前端。

1. AI 聊天視窗：有一個聊天視窗，可以向 AI 提出需求。
2. 打字時輸入：在打完一段按下 Enter 時啓動，用 Ink-gateway 讀取並儲存到 3 種資料庫。
3. 搜尋視窗：既然已經幫他把檔案結構化存入資料庫，就可以搜尋，Obsidian 的搜尋是搜檔案，這裡的搜尋是搜資料庫
4. 用模板存取資料：從 Obsidian 的模板取出規範的模板，轉成 chunks，這樣當內容以模板形式儲存或取出就很容易，例如一個 contact 是 Leo.md，含有 電話、Email、地址、職稱幾個 Slots，這個模板在 Ink-gateway 是一個 chunk，可以查詢有多少個 contact，每個 contact 的 id，可以取出相關的數個資料，這個 slot 在 Obsidian 叫屬性
5. Parse 層級：Obsidian 是用標題層級和內文來決定層級，也可以用 bullet + indent 來決定層級，兩者可以共用，在讀取 md 檔時可以把標題層級和 indent 層級都 parse 成 parent, son 的關聯
6. 記錄原文位置：在讀取 RemNote 時原文是個 ID，但在 Obsidian 是檔名 + 座標（第幾行），Parse 時記錄下來，這樣用戶找到段落後可以用來找到 Obsidian 的原文
7. 解耦結構：Ink-gateway 會跟不同的軟體串接，例如開發 Obsidian, Logseq, RemNote 的 Plugin，與這些軟體串接部分產生接口不影響 ink-gateway 運作。

## 20251002
8. 文件或虛擬文件 ID：如果從一篇 Obsidian 的 MD 檔抓下數個段落，這些段落屬於同一份文件，它們有共同的 Parent，但如果只用 Parent 來搜尋它的 son 文件，由於可以有無限多層級，可能會輪詢到非常多段，超過原本文件的段落數量。為了要從資料庫取出段落重現存入的文件，它們應該有個類似「文件 id」，如果在 RemNote 這種全部由資料庫撈取沒有文件長度的軟體，為了重現當初輸入的頁面，也要有個虛擬的「文件 id」，這樣可以用文件 id 來取出 chunks。