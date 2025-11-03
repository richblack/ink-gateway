20250920 筆記[^1]
- # 在 n8n 自動化使用 Google 服務詳解 | 從申請 Google Service Account 到 n8n Credential 設定 #h1 [^2] 
    - 目錄
        - [第一段：Google 多種認證方式解析](#google-多種認證方式解析-h2-5)
        - [第二段：開始申請 Service Account](開始申請-service-account-h2)
    - 網站服務用「帳密」認證，n8n 代替你整合各種服務，但它不會輸入帳密，要把網站的「API Key」（API金鑰），存在 n8n 的「證書」（Credential）中，就可以代表你整合各種服務了。但非常有用的 Google services 認證方式卻很繁雜，導致每次上課總有學員卡關，本文詳解取得 Service Account 的細節。「^3]
    - 你可以把本文做為小抄，忘了再回來查！我會持續更新，保持在最新狀態。[^4]
    - ## Google 多種認證方式解析 #h2 [^5]
        - Google 提供了「GCP，Google Cloud Console」，剛進去會覺得是天書，久了以後，還是天書~~，沒關係，不求什麼都懂，我們先懂認證方式即可，這段搞定，Google 服務本身還是比較易懂的。
            - Google 有 3 種認證方式：#ol [^6]
                1. API 金鑰（API Key）：這是最容易使用的，跟大部分服務類似，就是一長串文字貼進去就好了！可惜 Google 只有少部分服務提供，例如 Google Gemini。
                1. 開放授權協定 2.0（OAuth2）：這是好幾家科技公司合作制定的「授權標準協定」，包括 Google 和 Facebook 等，目的是讓人可以用大廠的帳號登入其他網站。例如「使用Google 認證登入 Line」就省下每個網站登錄的麻煩。但如果你發現你的工作流突然沒工作，很大機率是 OAuth2 被 Google 踢掉了。雖然 Google 說在少數狀況下才會把你登出，但 n8n 社群回報常莫名其妙被踢掉，就要再次手動認證，造成不少麻煩。
                1. 服務帳戶（Service Account）：顧名思義，這個帳戶（Account）不是提供人類使用，而是提供「服務」（Service）使用的，就是「機器對機器」認證，它難在要預先設好機器使用的權限，因為一台機器不應該像你一樣可以 100% 控制所有功能（至少別讓它幫你付錢！）。
        - > 😁 第一次看到這個名稱是不是以為用這個帳戶就可獲得 Google 特別的服務？它說人話我都配合，但它技術太高有說人話障礙，第一次使用 Google Cloud 時，發票上除了帳單金額都看不懂！[^7]
        - 因為 n8n 是機器上的一個服務，所以我們應該使用 Service Account，讓服務有自己的帳號。
        - 重點就在「不會過期」，我們睡覺時，n8n 仍然在默默地工作，如果你有多個工作流在不同時間觸發，過期了就要一一重新登入，真要命！更麻煩是你搞不清楚是誰出的問題，被老闆罵！
    - ## 開始申請 Service Account #h2
        - ![Tux, the Linux mascot](https://encrypted-tbn0.gstatic.com/images?q=tbn:ANd9GcTAx6camOf_G96Zcs2Zj34Yjmy0ysPd-p0oJQ&s) [^8]
- Contact [^9]
    - Phone #slot
    - Email #slot
    - Title #slot
- Leo #Contact [^10]
    - Phone|: 0987654321 [^11]
    - Email|: leo@abc.com
    - Title|: Manager
- 排版標籤設計，只在輸出一般文件時有效，避免使用 list 表達層級時忽略輸出後實際有層級的版式：如下請查看
    - h1 [^12]
    - h2
    - ul
