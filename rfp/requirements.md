# 需求

你已經完成 Ink-gateway 開發，這裡是下一段需求。

## 確定是否符合需求

這個計劃要使用單一表來記錄用戶掃描或用筆記輸入的資料，會把所有輸入內容解析為可以放進單一資料表，例如別的資料庫會把內文、標籤、參照、模板...每種建立單獨的表，這將造成每種服務都要人工增加新表，如果全部記錄在單一表，就可以容納任何內容，但原本的標籤、參照、模板...，因為都平行放在同一張表中，需要思考如何聰明的存取。

### 前端呈現部分說明

查看 rfp/ink-gateway-article-demo.md，這是計劃中的前端輸入筆記界面想法及 demo，下面註解對照內文的註腳符號。
- [^1]: 版型在頁面呈現，不記錄
- [^2]: 1）打字定義筆記 chunk，屬「20250920 筆記」頁；2）「h1」是排版標籤與層級無關
- [^3]: Tab 內縮定義這個 chunk 是上一個的子 chunk,需記錄 parent chunk、屬於「20250920 筆記」頁
- [^4]: 同一層級定義這個chunk與上一個chunk有同一個 parent chunk，記錄 parent chunk、屬於「20250920 筆記」頁
- [^5]:  Leo：indent 定義與上一個 chunk 同一個 parent，「##」是排版標籤定義標題字形，indent（tab）定義層級，與 md heading 無關
- [^6]: 因為都用 List 呈現，排版標籤「ol」標註用途是輸出 md 恢復 list
- [^7]:  1）排版標籤「>」系統解析為 quote block，供輸出時呈現；2）當 outdent 層級高於 List items，表示它上個 chunk 是 List 結尾
- [^8]: 圖片及連結記錄它的連結，標註為連結
- [^9]: 定義一個標籤及定義它有 template，在另外頁面設計 template 及 slot
- [^10]: Tagged Contact 後，template 會自動產生 Slots
- [^11]: Field 產生後，當使用者填入值，這個值的 chunk 與 slot 產生關聯
- [^12]: 自定義標籤只是文字，當 1）第一次被其他文字做為標籤使用時具有標籤的特徵；2）手動選取後標示這個 chunk 是標籤

### 資料表部分說明

#### 說明

- 所有資料儲存在同一張表中，目的是 RAG 時會輸入各種資料，這種結構，不論輸入任何資料，都無需修改資料表結構
- 每筆 entry 都是一個 chunk，如果在 LLM 讀取輸入資料發現一個段落的含義差別大，可以切割成多個 chunks，但 return 一個 chunk 給前端，確定 display 時與原本相同
- 前端解析 Markdown 後這些版型是 default 的 tags
- 版型標籤：由於前端用 bullets, indents 來表示層級關係，有些真正的層級會無法顯示，例如 ul, ol；也因為用層級代替 Heading 表示法，所以 H1, H2... 標籤需自行加上
- 沒有任何版型標籤時，就是段落文字<p>
- 定義不完整之處需要增加或建議

#### 欄位

- chunk_id|GUID：GUID 或 UUID，輸入時自動產生
- contents|text：chunk 的內容
- parent|text：前端按下 tab 後產生 indent，記錄上下級關係
- page|text：用來恢復原本版面，page 含有標題及內容，所有標示它的會被 load 回來（如何記錄 page 上每個元素之間的上下關係？）
- is_page|BL：這個 chunk 是 page
- is_tag|BL: 這個 chunk 用作 tag
- is_template|BL: 這個 chunk 是某個 tag 的 template
- is_slot|BL: 這個 chunk 是某個 template 下的 slot，就是一個固定輸入欄位
- ref|URI: 連結到對外網址或另一個 chunk
- Tags|text: 這個 chunk 被 tagged 的 tags list，一個 chunk 可以加上多個 tags，每個 tag 有不同影響，例如 todo 是 tag，在 chunk 前加上 checkbox、有 template 的 tag 會加上它的 slots、有的 tags 可改變 css
- created_time|datetime: chunk 創建時間
- last_updated|datetime: chunk 最新更新時間

### 功能是否完備？

為了要存取這些資料，要有一些特殊的邏輯，例如可以從標籤獲取 chunk。

因為 chunk 在關聯、向量、圖形資料庫是一對一的關係，可用一個 key 獲取 3 個資料庫的內容。

上述功能是我想到最基本要求，是否能滿足？

## 提供操作方式

需要操作手冊，包括測試方式，Go 如何編譯、編譯後檔案是哪個、如何執行？常用的指令。

現有的 API 文件，就算用 localhost，我可以用 n8n、postman，或其他方式測試是否正確。
