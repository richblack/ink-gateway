#!/usr/bin/env node

/**
 * Beta 測試設置腳本
 * 準備 Beta 測試環境和使用者回饋收集機制
 */

const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');

class BetaTestingSetup {
    constructor() {
        this.projectRoot = process.cwd();
        this.betaDir = path.join(this.projectRoot, 'beta');
        this.docsDir = path.join(this.projectRoot, 'docs');
    }

    async setupBetaTesting() {
        console.log('🧪 設置 Beta 測試環境...\n');

        try {
            // 1. 創建 Beta 目錄結構
            this.createBetaDirectories();

            // 2. 生成 Beta 版本
            await this.generateBetaVersion();

            // 3. 創建測試指南
            this.createTestingGuides();

            // 4. 設置回饋收集系統
            this.setupFeedbackCollection();

            // 5. 創建 Beta 測試者文件
            this.createBetaTesterDocumentation();

            // 6. 設置自動化回饋分析
            this.setupFeedbackAnalysis();

            console.log('✅ Beta 測試環境設置完成！');
            this.printBetaInfo();

        } catch (error) {
            console.error('❌ Beta 測試設置失敗:', error.message);
            process.exit(1);
        }
    }

    createBetaDirectories() {
        console.log('📁 創建 Beta 目錄結構...');

        const directories = [
            this.betaDir,
            path.join(this.betaDir, 'releases'),
            path.join(this.betaDir, 'feedback'),
            path.join(this.betaDir, 'testing-guides'),
            path.join(this.betaDir, 'analytics')
        ];

        directories.forEach(dir => {
            if (!fs.existsSync(dir)) {
                fs.mkdirSync(dir, { recursive: true });
                console.log(`  ✓ ${path.relative(this.projectRoot, dir)}`);
            }
        });
    }

    async generateBetaVersion() {
        console.log('🔨 生成 Beta 版本...');

        try {
            // 讀取當前版本
            const packageJson = JSON.parse(fs.readFileSync('package.json', 'utf8'));
            const currentVersion = packageJson.version;
            const betaVersion = `${currentVersion}-beta.${Date.now()}`;

            // 創建 Beta manifest
            const manifest = JSON.parse(fs.readFileSync('manifest.json', 'utf8'));
            manifest.version = betaVersion;
            manifest.name += ' (Beta)';
            manifest.id += '-beta';

            // 添加 Beta 標識
            if (!manifest.description.includes('Beta')) {
                manifest.description = `[BETA] ${manifest.description}`;
            }

            // 保存 Beta manifest
            const betaManifestPath = path.join(this.betaDir, 'releases', 'manifest.json');
            fs.writeFileSync(betaManifestPath, JSON.stringify(manifest, null, 2));

            // 建置 Beta 版本
            console.log('  建置 Beta 插件...');
            execSync('npm run build', { stdio: 'pipe' });

            // 複製建置檔案到 Beta 目錄
            const betaFiles = ['main.js', 'styles.css'];
            betaFiles.forEach(file => {
                if (fs.existsSync(file)) {
                    fs.copyFileSync(file, path.join(this.betaDir, 'releases', file));
                }
            });

            console.log(`  ✓ Beta 版本 ${betaVersion} 生成完成`);

        } catch (error) {
            throw new Error(`Beta 版本生成失敗: ${error.message}`);
        }
    }

    createTestingGuides() {
        console.log('📖 創建測試指南...');

        // Beta 測試計劃
        const testingPlan = this.generateTestingPlan();
        fs.writeFileSync(
            path.join(this.betaDir, 'testing-guides', 'BETA_TESTING_PLAN.md'),
            testingPlan
        );

        // 測試案例
        const testCases = this.generateTestCases();
        fs.writeFileSync(
            path.join(this.betaDir, 'testing-guides', 'TEST_CASES.md'),
            testCases
        );

        // 已知問題列表
        const knownIssues = this.generateKnownIssues();
        fs.writeFileSync(
            path.join(this.betaDir, 'testing-guides', 'KNOWN_ISSUES.md'),
            knownIssues
        );

        console.log('  ✓ 測試指南創建完成');
    }

    setupFeedbackCollection() {
        console.log('📝 設置回饋收集系統...');

        // 回饋表單模板
        const feedbackForm = this.generateFeedbackForm();
        fs.writeFileSync(
            path.join(this.betaDir, 'feedback', 'FEEDBACK_FORM.md'),
            feedbackForm
        );

        // 錯誤報告模板
        const bugReportTemplate = this.generateBugReportTemplate();
        fs.writeFileSync(
            path.join(this.betaDir, 'feedback', 'BUG_REPORT_TEMPLATE.md'),
            bugReportTemplate
        );

        // 功能請求模板
        const featureRequestTemplate = this.generateFeatureRequestTemplate();
        fs.writeFileSync(
            path.join(this.betaDir, 'feedback', 'FEATURE_REQUEST_TEMPLATE.md'),
            featureRequestTemplate
        );

        // 回饋收集腳本
        const feedbackCollector = this.generateFeedbackCollector();
        fs.writeFileSync(
            path.join(this.betaDir, 'feedback', 'collect-feedback.js'),
            feedbackCollector
        );

        console.log('  ✓ 回饋收集系統設置完成');
    }

    createBetaTesterDocumentation() {
        console.log('👥 創建 Beta 測試者文件...');

        // Beta 測試者指南
        const betaTesterGuide = this.generateBetaTesterGuide();
        fs.writeFileSync(
            path.join(this.betaDir, 'BETA_TESTER_GUIDE.md'),
            betaTesterGuide
        );

        // 安裝說明
        const installationGuide = this.generateBetaInstallationGuide();
        fs.writeFileSync(
            path.join(this.betaDir, 'INSTALLATION.md'),
            installationGuide
        );

        // FAQ
        const betaFAQ = this.generateBetaFAQ();
        fs.writeFileSync(
            path.join(this.betaDir, 'BETA_FAQ.md'),
            betaFAQ
        );

        console.log('  ✓ Beta 測試者文件創建完成');
    }

    setupFeedbackAnalysis() {
        console.log('📊 設置回饋分析系統...');

        // 回饋分析腳本
        const analysisScript = this.generateAnalysisScript();
        fs.writeFileSync(
            path.join(this.betaDir, 'analytics', 'analyze-feedback.js'),
            analysisScript
        );

        // 報告生成器
        const reportGenerator = this.generateReportGenerator();
        fs.writeFileSync(
            path.join(this.betaDir, 'analytics', 'generate-report.js'),
            reportGenerator
        );

        console.log('  ✓ 回饋分析系統設置完成');
    }

    generateTestingPlan() {
        return `# Beta 測試計劃

## 測試目標

本 Beta 測試旨在驗證 Obsidian Ink Plugin 的核心功能，收集使用者回饋，並在正式發布前識別和修復問題。

## 測試範圍

### 核心功能測試
- [ ] AI 聊天功能
- [ ] 自動內容處理和同步
- [ ] 語義搜尋功能
- [ ] 模板系統
- [ ] 階層內容解析
- [ ] 離線模式和同步
- [ ] 設定和配置

### 整合測試
- [ ] 與 Obsidian 的整合
- [ ] 與 Ink-Gateway 的通訊
- [ ] 多平台相容性
- [ ] 效能測試

### 使用者體驗測試
- [ ] 介面易用性
- [ ] 錯誤處理
- [ ] 文件完整性
- [ ] 安裝和設置流程

## 測試時程

- **第 1 週**: 核心功能測試
- **第 2 週**: 整合和效能測試
- **第 3 週**: 使用者體驗測試
- **第 4 週**: 回饋整理和修復

## 測試者要求

- 熟悉 Obsidian 的使用
- 願意提供詳細回饋
- 能夠按照測試指南執行測試
- 有時間參與整個測試週期

## 回饋收集

請使用以下方式提供回饋：
- GitHub Issues
- 回饋表單
- 測試報告

## 聯絡資訊

如有問題，請聯絡：
- Email: beta-testing@example.com
- GitHub: [項目頁面](https://github.com/your-username/obsidian-ink-plugin)
`;
    }

    generateTestCases() {
        return `# Beta 測試案例

## TC001: AI 聊天功能測試

### 前置條件
- 插件已安裝並啟用
- Ink-Gateway 連線已配置

### 測試步驟
1. 開啟 AI 聊天視窗
2. 發送測試訊息："Hello, how are you?"
3. 等待 AI 回應
4. 檢查聊天歷史記錄

### 預期結果
- 聊天視窗正常開啟
- AI 回應正常顯示
- 聊天歷史正確保存

---

## TC002: 自動內容處理測試

### 前置條件
- 插件已啟用自動同步
- 有有效的 Ink-Gateway 連線

### 測試步驟
1. 創建新筆記
2. 輸入內容並按 Enter
3. 檢查同步狀態
4. 驗證內容已上傳到 Ink-Gateway

### 預期結果
- 內容自動處理觸發
- 同步狀態正確顯示
- 內容成功上傳

---

## TC003: 語義搜尋測試

### 前置條件
- 已有一些內容同步到系統
- 搜尋功能已啟用

### 測試步驟
1. 開啟搜尋視窗
2. 輸入語義搜尋查詢
3. 檢查搜尋結果
4. 點擊結果項目導航

### 預期結果
- 搜尋結果相關且準確
- 導航功能正常工作
- 搜尋效能可接受

---

## TC004: 模板系統測試

### 前置條件
- 插件已啟用
- 有模板創建權限

### 測試步驟
1. 創建新模板
2. 定義模板插槽
3. 應用模板到新筆記
4. 驗證模板實例

### 預期結果
- 模板創建成功
- 插槽正確映射到 Obsidian 屬性
- 模板應用正常工作

---

## TC005: 離線模式測試

### 前置條件
- 插件已啟用
- 網路連線正常

### 測試步驟
1. 斷開網路連線
2. 修改筆記內容
3. 檢查離線狀態指示
4. 恢復網路連線
5. 驗證自動同步

### 預期結果
- 離線狀態正確檢測
- 變更正確排隊
- 上線後自動同步

---

## 效能測試案例

### PT001: 大型文件處理
- 測試 10MB+ 文件的處理效能
- 記錄處理時間和記憶體使用

### PT002: 並發操作
- 同時處理多個文件
- 測試系統穩定性

### PT003: 長時間運行
- 連續運行 24 小時
- 監控記憶體洩漏和效能衰減
`;
    }

    generateKnownIssues() {
        return `# 已知問題

## 高優先級問題

### KI001: 大型文件同步緩慢
**描述**: 超過 5MB 的文件同步速度較慢
**影響**: 使用者體驗
**狀態**: 正在修復
**預計修復版本**: v1.0.1

### KI002: 某些 Obsidian 主題相容性問題
**描述**: 在某些自定義主題下 UI 顯示異常
**影響**: 視覺效果
**狀態**: 調查中
**預計修復版本**: v1.0.2

## 中優先級問題

### KI003: 搜尋結果排序不一致
**描述**: 相同查詢的搜尋結果順序可能不同
**影響**: 使用者體驗
**狀態**: 已確認
**預計修復版本**: v1.1.0

### KI004: 離線模式下的錯誤訊息不夠清楚
**描述**: 離線時的錯誤提示需要改進
**影響**: 使用者體驗
**狀態**: 計劃中
**預計修復版本**: v1.0.3

## 低優先級問題

### KI005: 某些特殊字符處理問題
**描述**: 包含特殊 Unicode 字符的內容可能處理異常
**影響**: 功能性
**狀態**: 已記錄
**預計修復版本**: v1.2.0

## 限制和注意事項

- 目前不支援超過 100MB 的單個文件
- 同時處理的文件數量建議不超過 50 個
- 某些 Obsidian 插件可能存在衝突

## 回報新問題

如果發現新問題，請：
1. 檢查是否為已知問題
2. 收集詳細的錯誤資訊
3. 提供重現步驟
4. 使用 Bug 報告模板提交
`;
    }

    generateFeedbackForm() {
        return `# Beta 測試回饋表單

## 基本資訊

**測試者姓名/暱稱**: 
**測試日期**: 
**插件版本**: 
**Obsidian 版本**: 
**作業系統**: 

## 功能測試回饋

### AI 聊天功能
- [ ] 功能正常
- [ ] 有小問題
- [ ] 有嚴重問題
- [ ] 未測試

**詳細回饋**:


### 自動內容處理
- [ ] 功能正常
- [ ] 有小問題
- [ ] 有嚴重問題
- [ ] 未測試

**詳細回饋**:


### 語義搜尋
- [ ] 功能正常
- [ ] 有小問題
- [ ] 有嚴重問題
- [ ] 未測試

**詳細回饋**:


### 模板系統
- [ ] 功能正常
- [ ] 有小問題
- [ ] 有嚴重問題
- [ ] 未測試

**詳細回饋**:


## 整體評價

**易用性** (1-5 分): 
**效能** (1-5 分): 
**穩定性** (1-5 分): 
**文件品質** (1-5 分): 

## 建議和改進

**最喜歡的功能**:


**最需要改進的地方**:


**功能建議**:


## 問題報告

**遇到的問題**:


**重現步驟**:


**預期行為**:


**實際行為**:


## 其他回饋

**其他意見或建議**:


---

**提交方式**: 
- 將此表單填寫完成後發送到 beta-testing@example.com
- 或在 GitHub 創建 Issue 並使用 "beta-feedback" 標籤
`;
    }

    generateBugReportTemplate() {
        return `# Bug 報告模板

## Bug 描述
簡潔清楚地描述這個 bug。

## 重現步驟
詳細描述如何重現這個問題：
1. 
2. 
3. 

## 預期行為
描述你預期應該發生什麼。

## 實際行為
描述實際發生了什麼。

## 螢幕截圖
如果適用，請添加螢幕截圖來幫助解釋問題。

## 環境資訊
- **插件版本**: 
- **Obsidian 版本**: 
- **作業系統**: 
- **瀏覽器** (如果相關): 

## 錯誤日誌
如果有錯誤日誌，請貼在這裡：
\`\`\`
錯誤日誌內容
\`\`\`

## 額外資訊
添加任何其他有助於解決問題的資訊。

## 嚴重程度
- [ ] 低 - 小問題，不影響主要功能
- [ ] 中 - 影響某些功能，但有替代方案
- [ ] 高 - 影響主要功能
- [ ] 緊急 - 導致插件無法使用

## 頻率
- [ ] 總是發生
- [ ] 經常發生
- [ ] 偶爾發生
- [ ] 只發生一次
`;
    }

    generateFeatureRequestTemplate() {
        return `# 功能請求模板

## 功能描述
簡潔清楚地描述你想要的功能。

## 問題或需求
這個功能要解決什麼問題？為什麼需要這個功能？

## 建議的解決方案
描述你希望這個功能如何工作。

## 替代方案
描述你考慮過的其他替代解決方案。

## 使用場景
描述這個功能的具體使用場景：
1. 
2. 
3. 

## 優先級
- [ ] 低 - 有了更好，沒有也可以
- [ ] 中 - 會顯著改善使用體驗
- [ ] 高 - 對工作流程很重要
- [ ] 緊急 - 沒有這個功能無法正常使用

## 額外資訊
添加任何其他相關資訊、連結或參考。

## 願意協助
- [ ] 我願意協助測試這個功能
- [ ] 我願意協助撰寫文件
- [ ] 我願意協助開發 (如果是開源項目)
`;
    }

    generateFeedbackCollector() {
        return `#!/usr/bin/env node

/**
 * 回饋收集腳本
 * 自動收集和整理 Beta 測試回饋
 */

const fs = require('fs');
const path = require('path');

class FeedbackCollector {
    constructor() {
        this.feedbackDir = path.join(__dirname);
        this.reportsDir = path.join(this.feedbackDir, 'reports');
        
        if (!fs.existsSync(this.reportsDir)) {
            fs.mkdirSync(this.reportsDir, { recursive: true });
        }
    }

    collectFeedback() {
        console.log('📊 收集 Beta 測試回饋...');
        
        const feedback = {
            timestamp: new Date().toISOString(),
            summary: this.generateSummary(),
            issues: this.collectIssues(),
            suggestions: this.collectSuggestions(),
            ratings: this.collectRatings()
        };
        
        const reportPath = path.join(this.reportsDir, \`feedback-\${Date.now()}.json\`);
        fs.writeFileSync(reportPath, JSON.stringify(feedback, null, 2));
        
        console.log(\`✅ 回饋報告已保存: \${reportPath}\`);
        
        return feedback;
    }

    generateSummary() {
        // 實現回饋摘要生成邏輯
        return {
            totalFeedback: 0,
            averageRating: 0,
            commonIssues: [],
            topSuggestions: []
        };
    }

    collectIssues() {
        // 實現問題收集邏輯
        return [];
    }

    collectSuggestions() {
        // 實現建議收集邏輯
        return [];
    }

    collectRatings() {
        // 實現評分收集邏輯
        return {
            usability: 0,
            performance: 0,
            stability: 0,
            documentation: 0
        };
    }
}

if (require.main === module) {
    const collector = new FeedbackCollector();
    collector.collectFeedback();
}

module.exports = { FeedbackCollector };`;
    }

    generateBetaTesterGuide() {
        return `# Beta 測試者指南

歡迎參與 Obsidian Ink Plugin 的 Beta 測試！

## 開始之前

### 系統要求
- Obsidian v0.15.0 或更高版本
- 穩定的網路連線
- 有效的 Ink-Gateway API 存取權限

### 重要提醒
⚠️ **這是 Beta 版本，可能包含錯誤和不穩定的功能**
- 建議在測試環境中使用
- 定期備份你的 Obsidian 文件庫
- 遇到問題時請及時回報

## 安裝指南

1. 下載 Beta 版本檔案
2. 按照 [安裝說明](INSTALLATION.md) 進行安裝
3. 配置 Ink-Gateway 連線
4. 開始測試

## 測試重點

### 第一週：核心功能
- [ ] AI 聊天功能測試
- [ ] 自動內容處理測試
- [ ] 基本設定配置測試

### 第二週：進階功能
- [ ] 語義搜尋功能測試
- [ ] 模板系統測試
- [ ] 離線模式測試

### 第三週：整合測試
- [ ] 與其他插件的相容性
- [ ] 效能測試
- [ ] 長時間使用測試

### 第四週：使用者體驗
- [ ] 介面易用性評估
- [ ] 文件完整性檢查
- [ ] 整體滿意度評估

## 回饋方式

### 日常回饋
- 使用 [回饋表單](feedback/FEEDBACK_FORM.md)
- 在 GitHub 創建 Issues
- 參與社群討論

### 問題回報
- 使用 [Bug 報告模板](feedback/BUG_REPORT_TEMPLATE.md)
- 提供詳細的重現步驟
- 包含螢幕截圖和錯誤日誌

### 功能建議
- 使用 [功能請求模板](feedback/FEATURE_REQUEST_TEMPLATE.md)
- 描述具體的使用場景
- 說明功能的重要性

## 測試技巧

### 有效測試
1. **系統性測試**: 按照測試計劃逐項進行
2. **邊界測試**: 嘗試極端情況和邊界條件
3. **真實使用**: 在實際工作流程中使用插件
4. **記錄問題**: 詳細記錄遇到的問題和建議

### 回饋品質
- 提供具體的例子和場景
- 包含重現步驟
- 說明問題的影響程度
- 建議可能的解決方案

## 聯絡我們

- **Email**: beta-testing@example.com
- **GitHub**: [項目頁面](https://github.com/your-username/obsidian-ink-plugin)
- **Discord**: [測試者社群](https://discord.gg/example)

## 感謝

感謝你參與 Beta 測試！你的回饋對改善插件品質非常重要。

---

**測試愉快！** 🚀
`;
    }

    generateBetaInstallationGuide() {
        return `# Beta 版本安裝指南

## 自動安裝 (推薦)

### 使用 BRAT 插件
1. 安裝 [BRAT](https://github.com/TfTHacker/obsidian42-brat) 插件
2. 在 BRAT 設定中添加 Beta 插件 URL
3. 啟用插件並重新載入 Obsidian

## 手動安裝

### 步驟 1: 下載檔案
從 Beta 發布頁面下載以下檔案：
- \`main.js\`
- \`manifest.json\`
- \`styles.css\` (如果存在)

### 步驟 2: 創建插件目錄
在你的 Obsidian 文件庫中創建目錄：
\`\`\`
.obsidian/plugins/obsidian-ink-plugin-beta/
\`\`\`

### 步驟 3: 複製檔案
將下載的檔案複製到插件目錄中。

### 步驟 4: 啟用插件
1. 重新載入 Obsidian
2. 前往設定 > 社群插件
3. 找到 "Obsidian Ink Plugin (Beta)" 並啟用

## 配置設定

### 基本配置
1. 開啟插件設定
2. 輸入 Ink-Gateway URL
3. 輸入 API 金鑰
4. 點擊 "測試連線"

### 進階配置
- 調整同步間隔
- 配置快取設定
- 設定除錯模式

## 驗證安裝

### 檢查清單
- [ ] 插件在插件列表中顯示為已啟用
- [ ] 可以開啟插件設定頁面
- [ ] 連線測試成功
- [ ] AI 聊天視窗可以開啟
- [ ] 搜尋視窗可以開啟

### 測試基本功能
1. 開啟 AI 聊天並發送測試訊息
2. 創建新筆記並觸發自動同步
3. 嘗試語義搜尋功能

## 故障排除

### 常見問題

#### 插件無法載入
- 檢查檔案是否正確放置
- 確認 manifest.json 格式正確
- 重新載入 Obsidian

#### 連線失敗
- 檢查網路連線
- 驗證 API 金鑰
- 確認 Ink-Gateway URL 正確

#### 功能異常
- 開啟除錯模式
- 檢查瀏覽器控制台錯誤
- 查看已知問題列表

### 獲取幫助
如果遇到安裝問題：
1. 查看 [故障排除指南](../docs/TROUBLESHOOTING.md)
2. 搜尋已知問題
3. 聯絡 Beta 測試支援團隊

## 卸載

如需卸載 Beta 版本：
1. 在插件設定中停用插件
2. 刪除插件目錄
3. 重新載入 Obsidian

---

**需要幫助？** 聯絡我們：beta-testing@example.com
`;
    }

    generateBetaFAQ() {
        return `# Beta 測試常見問題

## 一般問題

### Q: Beta 測試需要多長時間？
A: 預計 4 週，但可能根據回饋情況調整。

### Q: Beta 版本是否安全？
A: Beta 版本經過基本測試，但可能包含錯誤。建議在測試環境中使用。

### Q: 我的資料會遺失嗎？
A: 雖然我們盡力確保資料安全，但建議定期備份你的 Obsidian 文件庫。

### Q: 可以同時安裝正式版和 Beta 版嗎？
A: 不建議，可能會產生衝突。請選擇其中一個版本。

## 功能問題

### Q: AI 聊天功能需要什麼？
A: 需要有效的 Ink-Gateway API 存取權限和穩定的網路連線。

### Q: 語義搜尋不準確怎麼辦？
A: 這是 Beta 版本的已知問題，我們正在改進演算法。

### Q: 離線模式如何工作？
A: 離線時變更會被排隊，網路恢復後自動同步。

### Q: 模板系統支援哪些類型？
A: 目前支援文字、數字、日期、連結和標籤類型。

## 技術問題

### Q: 插件載入很慢怎麼辦？
A: 這可能是效能問題，請在回饋中報告你的系統配置。

### Q: 與其他插件衝突怎麼辦？
A: 請報告衝突的插件名稱，我們會調查相容性問題。

### Q: 如何開啟除錯模式？
A: 在插件設定中啟用 "除錯模式"，然後檢查瀏覽器控制台。

## 回饋問題

### Q: 如何提供有效的回饋？
A: 提供具體的例子、重現步驟和系統資訊。

### Q: 多久回饋一次？
A: 建議每週至少回饋一次，遇到問題時立即回報。

### Q: 回饋會被採納嗎？
A: 我們會認真考慮所有回饋，但不能保證所有建議都會實現。

## 支援問題

### Q: 遇到問題如何獲得幫助？
A: 可以通過 Email、GitHub Issues 或 Discord 聯絡我們。

### Q: 回應時間是多久？
A: 通常在 24-48 小時內回應，緊急問題會優先處理。

### Q: 可以直接聯絡開發者嗎？
A: 請通過官方管道聯絡，這樣可以確保問題得到適當處理。

## 發布問題

### Q: Beta 測試結束後會怎樣？
A: 會發布正式版本，Beta 測試者會收到升級通知。

### Q: Beta 版本的設定會保留嗎？
A: 大部分設定會保留，但可能需要重新配置某些選項。

### Q: 如何獲得正式版本？
A: 正式版本會在 Obsidian 社群插件商店發布。

---

**還有其他問題？** 
聯絡我們：beta-testing@example.com
`;
    }

    generateAnalysisScript() {
        return `#!/usr/bin/env node

/**
 * 回饋分析腳本
 * 分析 Beta 測試回饋並生成洞察報告
 */

const fs = require('fs');
const path = require('path');

class FeedbackAnalyzer {
    constructor() {
        this.feedbackDir = path.join(__dirname, '..', 'feedback');
        this.reportsDir = path.join(__dirname, 'reports');
        
        if (!fs.existsSync(this.reportsDir)) {
            fs.mkdirSync(this.reportsDir, { recursive: true });
        }
    }

    analyzeFeedback() {
        console.log('📊 分析 Beta 測試回饋...');
        
        const analysis = {
            timestamp: new Date().toISOString(),
            summary: this.generateSummary(),
            issues: this.analyzeIssues(),
            ratings: this.analyzeRatings(),
            suggestions: this.analyzeSuggestions(),
            trends: this.analyzeTrends(),
            recommendations: this.generateRecommendations()
        };
        
        const reportPath = path.join(this.reportsDir, \`analysis-\${Date.now()}.json\`);
        fs.writeFileSync(reportPath, JSON.stringify(analysis, null, 2));
        
        console.log(\`✅ 分析報告已保存: \${reportPath}\`);
        
        return analysis;
    }

    generateSummary() {
        return {
            totalFeedback: 0,
            responseRate: 0,
            averageRating: 0,
            completionRate: 0
        };
    }

    analyzeIssues() {
        return {
            totalIssues: 0,
            criticalIssues: 0,
            commonIssues: [],
            issuesByCategory: {}
        };
    }

    analyzeRatings() {
        return {
            usability: { average: 0, distribution: {} },
            performance: { average: 0, distribution: {} },
            stability: { average: 0, distribution: {} },
            documentation: { average: 0, distribution: {} }
        };
    }

    analyzeSuggestions() {
        return {
            totalSuggestions: 0,
            topSuggestions: [],
            suggestionsByCategory: {}
        };
    }

    analyzeTrends() {
        return {
            ratingTrends: {},
            issueTrends: {},
            engagementTrends: {}
        };
    }

    generateRecommendations() {
        return {
            priorityFixes: [],
            featureRequests: [],
            documentationImprovements: [],
            processImprovements: []
        };
    }
}

if (require.main === module) {
    const analyzer = new FeedbackAnalyzer();
    analyzer.analyzeFeedback();
}

module.exports = { FeedbackAnalyzer };`;
    }

    generateReportGenerator() {
        return `#!/usr/bin/env node

/**
 * 報告生成器
 * 生成 Beta 測試的綜合報告
 */

const fs = require('fs');
const path = require('path');

class ReportGenerator {
    constructor() {
        this.analyticsDir = __dirname;
        this.reportsDir = path.join(this.analyticsDir, 'reports');
    }

    generateReport() {
        console.log('📋 生成 Beta 測試報告...');
        
        const report = this.createComprehensiveReport();
        
        // 生成 Markdown 報告
        const markdownReport = this.generateMarkdownReport(report);
        const markdownPath = path.join(this.reportsDir, \`beta-test-report-\${Date.now()}.md\`);
        fs.writeFileSync(markdownPath, markdownReport);
        
        // 生成 HTML 報告
        const htmlReport = this.generateHTMLReport(report);
        const htmlPath = path.join(this.reportsDir, \`beta-test-report-\${Date.now()}.html\`);
        fs.writeFileSync(htmlPath, htmlReport);
        
        console.log(\`✅ 報告已生成:\`);
        console.log(\`  Markdown: \${markdownPath}\`);
        console.log(\`  HTML: \${htmlPath}\`);
        
        return report;
    }

    createComprehensiveReport() {
        return {
            metadata: {
                generatedAt: new Date().toISOString(),
                reportPeriod: this.getReportPeriod(),
                version: this.getBetaVersion()
            },
            executiveSummary: this.generateExecutiveSummary(),
            testingMetrics: this.generateTestingMetrics(),
            issueAnalysis: this.generateIssueAnalysis(),
            userFeedback: this.generateUserFeedback(),
            recommendations: this.generateRecommendations(),
            nextSteps: this.generateNextSteps()
        };
    }

    generateMarkdownReport(report) {
        return \`# Beta 測試報告

生成時間: \${report.metadata.generatedAt}
測試期間: \${report.metadata.reportPeriod}
Beta 版本: \${report.metadata.version}

## 執行摘要

\${report.executiveSummary}

## 測試指標

\${report.testingMetrics}

## 問題分析

\${report.issueAnalysis}

## 使用者回饋

\${report.userFeedback}

## 建議

\${report.recommendations}

## 下一步

\${report.nextSteps}
\`;
    }

    generateHTMLReport(report) {
        return \`<!DOCTYPE html>
<html>
<head>
    <title>Beta 測試報告</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; }
        h1, h2, h3 { color: #333; }
        .metric { background: #f5f5f5; padding: 10px; margin: 10px 0; }
        .issue { border-left: 4px solid #ff6b6b; padding-left: 10px; }
        .suggestion { border-left: 4px solid #4ecdc4; padding-left: 10px; }
    </style>
</head>
<body>
    <h1>Beta 測試報告</h1>
    <p><strong>生成時間:</strong> \${report.metadata.generatedAt}</p>
    <p><strong>測試期間:</strong> \${report.metadata.reportPeriod}</p>
    <p><strong>Beta 版本:</strong> \${report.metadata.version}</p>
    
    <h2>執行摘要</h2>
    <div>\${report.executiveSummary}</div>
    
    <h2>測試指標</h2>
    <div>\${report.testingMetrics}</div>
    
    <h2>問題分析</h2>
    <div>\${report.issueAnalysis}</div>
    
    <h2>使用者回饋</h2>
    <div>\${report.userFeedback}</div>
    
    <h2>建議</h2>
    <div>\${report.recommendations}</div>
    
    <h2>下一步</h2>
    <div>\${report.nextSteps}</div>
</body>
</html>\`;
    }

    getReportPeriod() {
        return '2024-01-01 to 2024-01-31';
    }

    getBetaVersion() {
        return '1.0.0-beta.1';
    }

    generateExecutiveSummary() {
        return 'Beta 測試執行摘要...';
    }

    generateTestingMetrics() {
        return '測試指標詳情...';
    }

    generateIssueAnalysis() {
        return '問題分析詳情...';
    }

    generateUserFeedback() {
        return '使用者回饋摘要...';
    }

    generateRecommendations() {
        return '改進建議...';
    }

    generateNextSteps() {
        return '後續步驟規劃...';
    }
}

if (require.main === module) {
    const generator = new ReportGenerator();
    generator.generateReport();
}

module.exports = { ReportGenerator };`;
    }

    printBetaInfo() {
        console.log('\n📋 Beta 測試環境資訊');
        console.log('=' .repeat(50));
        console.log(`Beta 目錄: ${this.betaDir}`);
        console.log(`文件目錄: ${this.docsDir}`);
        
        console.log('\n📁 創建的檔案:');
        const files = [
            'beta/releases/manifest.json',
            'beta/BETA_TESTER_GUIDE.md',
            'beta/INSTALLATION.md',
            'beta/testing-guides/BETA_TESTING_PLAN.md',
            'beta/feedback/FEEDBACK_FORM.md'
        ];
        
        files.forEach(file => {
            if (fs.existsSync(file)) {
                console.log(`  ✓ ${file}`);
            }
        });
        
        console.log('\n🚀 下一步:');
        console.log('1. 招募 Beta 測試者');
        console.log('2. 分發 Beta 版本');
        console.log('3. 收集和分析回饋');
        console.log('4. 根據回饋改進插件');
        console.log('5. 準備正式發布');
    }
}

// 主執行邏輯
async function main() {
    try {
        const setup = new BetaTestingSetup();
        await setup.setupBetaTesting();
    } catch (error) {
        console.error('❌ Beta 測試設置失敗:', error.message);
        process.exit(1);
    }
}

if (require.main === module) {
    main();
}

module.exports = { BetaTestingSetup };