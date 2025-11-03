#!/usr/bin/env node

/**
 * Obsidian 社群插件商店提交準備腳本
 * 自動化準備插件提交到 Obsidian 社群插件商店的流程
 */

const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');

class CommunityPluginPrep {
    constructor() {
        this.projectRoot = process.cwd();
        this.submissionDir = path.join(this.projectRoot, 'community-plugin-submission');
        this.packageJson = require(path.join(this.projectRoot, 'package.json'));
        this.manifest = require(path.join(this.projectRoot, 'manifest.json'));
    }

    async prepareSubmission() {
        console.log('🏪 準備 Obsidian 社群插件商店提交...\n');

        try {
            // 1. 驗證插件要求
            this.validatePluginRequirements();

            // 2. 創建提交目錄
            this.createSubmissionDirectory();

            // 3. 準備必要檔案
            this.prepareRequiredFiles();

            // 4. 生成提交文件
            this.generateSubmissionDocuments();

            // 5. 驗證提交內容
            this.validateSubmission();

            // 6. 創建提交檢查清單
            this.createSubmissionChecklist();

            // 7. 生成 PR 模板
            this.generatePRTemplate();

            console.log('✅ 社群插件商店提交準備完成！');
            this.printSubmissionInfo();

        } catch (error) {
            console.error('❌ 提交準備失敗:', error.message);
            process.exit(1);
        }
    }

    validatePluginRequirements() {
        console.log('🔍 驗證插件要求...');

        // 檢查必要檔案
        const requiredFiles = [
            'main.js',
            'manifest.json',
            'README.md'
        ];

        for (const file of requiredFiles) {
            if (!fs.existsSync(file)) {
                throw new Error(`缺少必要檔案: ${file}`);
            }
        }

        // 驗證 manifest.json
        this.validateManifest();

        // 檢查檔案大小
        this.validateFileSize();

        // 驗證插件 ID 唯一性
        this.validatePluginId();

        console.log('✅ 插件要求驗證通過');
    }

    validateManifest() {
        const manifest = this.manifest;

        // 必要欄位檢查
        const requiredFields = {
            'id': '插件 ID',
            'name': '插件名稱',
            'version': '版本號',
            'minAppVersion': '最低 Obsidian 版本',
            'description': '插件描述',
            'author': '作者',
            'authorUrl': '作者 URL'
        };

        for (const [field, description] of Object.entries(requiredFields)) {
            if (!manifest[field]) {
                throw new Error(`manifest.json 缺少必要欄位: ${field} (${description})`);
            }
        }

        // 格式驗證
        if (!/^[a-z0-9-]+$/.test(manifest.id)) {
            throw new Error(`插件 ID 格式錯誤: ${manifest.id} (只能包含小寫字母、數字和連字符)`);
        }

        if (!/^\d+\.\d+\.\d+$/.test(manifest.version)) {
            throw new Error(`版本號格式錯誤: ${manifest.version} (應為 x.y.z 格式)`);
        }

        if (manifest.description.length < 10 || manifest.description.length > 250) {
            throw new Error(`描述長度不當: ${manifest.description.length} 字符 (應為 10-250 字符)`);
        }

        // 檢查 minAppVersion
        if (!/^\d+\.\d+\.\d+$/.test(manifest.minAppVersion)) {
            throw new Error(`最低 Obsidian 版本格式錯誤: ${manifest.minAppVersion}`);
        }
    }

    validateFileSize() {
        const mainJsPath = path.join(this.projectRoot, 'main.js');
        const stats = fs.statSync(mainJsPath);
        const fileSizeMB = stats.size / (1024 * 1024);

        if (fileSizeMB > 10) {
            throw new Error(`main.js 檔案過大: ${fileSizeMB.toFixed(2)}MB (限制: 10MB)`);
        }

        console.log(`  ✓ main.js 大小: ${fileSizeMB.toFixed(2)}MB`);
    }

    validatePluginId() {
        const pluginId = this.manifest.id;
        
        // 檢查是否與知名插件衝突
        const reservedIds = [
            'obsidian-git',
            'dataview',
            'templater-obsidian',
            'calendar',
            'advanced-tables-obsidian'
        ];

        if (reservedIds.includes(pluginId)) {
            throw new Error(`插件 ID 與現有插件衝突: ${pluginId}`);
        }

        console.log(`  ✓ 插件 ID: ${pluginId}`);
    }

    createSubmissionDirectory() {
        console.log('📁 創建提交目錄...');

        if (fs.existsSync(this.submissionDir)) {
            fs.rmSync(this.submissionDir, { recursive: true, force: true });
        }

        fs.mkdirSync(this.submissionDir, { recursive: true });

        const subdirs = [
            'plugin-files',
            'documentation',
            'assets',
            'submission-docs'
        ];

        subdirs.forEach(dir => {
            fs.mkdirSync(path.join(this.submissionDir, dir), { recursive: true });
            console.log(`  ✓ ${dir}`);
        });
    }

    prepareRequiredFiles() {
        console.log('📋 準備必要檔案...');

        // 複製插件檔案
        const pluginFiles = ['main.js', 'manifest.json', 'styles.css'];
        const pluginDir = path.join(this.submissionDir, 'plugin-files');

        pluginFiles.forEach(file => {
            const srcPath = path.join(this.projectRoot, file);
            const destPath = path.join(pluginDir, file);

            if (fs.existsSync(srcPath)) {
                fs.copyFileSync(srcPath, destPath);
                console.log(`  ✓ ${file}`);
            } else if (file === 'styles.css') {
                console.log(`  - ${file} (可選，跳過)`);
            }
        });

        // 複製文件
        const docFiles = ['README.md', 'LICENSE'];
        const docDir = path.join(this.submissionDir, 'documentation');

        docFiles.forEach(file => {
            const srcPath = path.join(this.projectRoot, file);
            const destPath = path.join(docDir, file);

            if (fs.existsSync(srcPath)) {
                fs.copyFileSync(srcPath, destPath);
                console.log(`  ✓ ${file}`);
            }
        });

        // 複製資源檔案
        this.prepareAssets();
    }

    prepareAssets() {
        const assetsDir = path.join(this.submissionDir, 'assets');

        // 創建插件圖示 (如果不存在)
        const iconPath = path.join(this.projectRoot, 'icon.png');
        if (!fs.existsSync(iconPath)) {
            this.generateDefaultIcon(iconPath);
        }

        if (fs.existsSync(iconPath)) {
            fs.copyFileSync(iconPath, path.join(assetsDir, 'icon.png'));
            console.log('  ✓ icon.png');
        }

        // 複製螢幕截圖
        const screenshotsDir = path.join(this.projectRoot, 'screenshots');
        if (fs.existsSync(screenshotsDir)) {
            const screenshots = fs.readdirSync(screenshotsDir);
            screenshots.forEach(screenshot => {
                fs.copyFileSync(
                    path.join(screenshotsDir, screenshot),
                    path.join(assetsDir, screenshot)
                );
            });
            console.log(`  ✓ ${screenshots.length} 個螢幕截圖`);
        }
    }

    generateDefaultIcon(iconPath) {
        // 創建簡單的 SVG 圖示並轉換為 PNG
        const svgIcon = `<svg width="100" height="100" xmlns="http://www.w3.org/2000/svg">
            <rect width="100" height="100" fill="#6366f1"/>
            <text x="50" y="60" font-family="Arial" font-size="40" fill="white" text-anchor="middle">I</text>
        </svg>`;

        // 注意：實際實現需要 SVG 到 PNG 的轉換庫
        console.log('  ⚠️  請手動創建 icon.png (100x100 像素)');
    }

    generateSubmissionDocuments() {
        console.log('📝 生成提交文件...');

        // 插件資訊摘要
        this.generatePluginSummary();

        // 功能列表
        this.generateFeatureList();

        // 安裝說明
        this.generateInstallationInstructions();

        // 使用指南
        this.generateUsageGuide();

        // 開發者資訊
        this.generateDeveloperInfo();
    }

    generatePluginSummary() {
        const summary = `# ${this.manifest.name}

## 基本資訊

- **插件 ID**: ${this.manifest.id}
- **版本**: ${this.manifest.version}
- **作者**: ${this.manifest.author}
- **最低 Obsidian 版本**: ${this.manifest.minAppVersion}

## 描述

${this.manifest.description}

## 主要功能

- 🤖 AI 聊天功能，與 Ink-Gateway 系統整合
- 🔍 強大的語義搜尋，支援向量、圖形和標籤搜尋
- 📝 自動內容處理和同步到多個資料庫
- 🎯 模板系統，支援結構化內容管理
- 📊 階層內容解析，支援標題和項目符號層級
- 🔄 即時同步功能，支援離線模式
- 📍 精確的位置追蹤和導航功能

## 技術特性

- 解耦架構設計，易於擴展
- 高效能快取系統
- 完善的錯誤處理機制
- 豐富的設定選項
- 完整的測試覆蓋率

## 系統要求

- Obsidian ${this.manifest.minAppVersion} 或更高版本
- 有效的 Ink-Gateway API 連線
- 穩定的網路連線

## 授權

${this.getLicenseInfo()}

## 支援

- GitHub: ${this.getRepositoryUrl()}
- 文件: ${this.getDocumentationUrl()}
- 問題回報: ${this.getIssuesUrl()}
`;

        fs.writeFileSync(
            path.join(this.submissionDir, 'submission-docs', 'PLUGIN_SUMMARY.md'),
            summary
        );

        console.log('  ✓ 插件摘要');
    }

    generateFeatureList() {
        const features = `# 功能列表

## 核心功能

### 1. AI 聊天系統
- 與 Ink-Gateway AI 服務整合
- 支援上下文感知對話
- 聊天歷史記錄管理
- 智能內容建議

### 2. 語義搜尋
- 向量相似性搜尋
- 圖形關係搜尋
- 標籤組合搜尋
- 搜尋結果快取

### 3. 自動內容處理
- 即時內容同步
- 多資料庫儲存 (PostgreSQL, PGVector, Amazon AGE)
- 智能內容分塊
- 元資料提取

### 4. 模板系統
- 結構化內容模板
- 動態插槽系統
- 與 Obsidian 屬性整合
- 模板實例管理

### 5. 階層解析
- 標題層級識別
- 項目符號縮排解析
- 父子關係建立
- 位置追蹤

## 進階功能

### 6. 離線支援
- 離線狀態檢測
- 變更佇列管理
- 自動同步恢復
- 衝突解決

### 7. 效能最佳化
- 多層快取系統
- 延遲載入
- 批次處理
- 記憶體管理

### 8. 使用者介面
- 直觀的設定介面
- 即時狀態顯示
- 錯誤訊息提示
- 故障排除工具

## 開發者功能

### 9. 除錯支援
- 詳細日誌記錄
- 效能監控
- 錯誤追蹤
- 診斷工具

### 10. 擴展性
- 模組化架構
- 插件 API
- 自定義配置
- 第三方整合
`;

        fs.writeFileSync(
            path.join(this.submissionDir, 'submission-docs', 'FEATURE_LIST.md'),
            features
        );

        console.log('  ✓ 功能列表');
    }

    generateInstallationInstructions() {
        const instructions = `# 安裝說明

## 自動安裝 (推薦)

### 從 Obsidian 社群插件商店安裝

1. 開啟 Obsidian
2. 前往 設定 → 社群插件
3. 點擊 "瀏覽" 按鈕
4. 搜尋 "${this.manifest.name}"
5. 點擊 "安裝" 按鈕
6. 安裝完成後，點擊 "啟用" 按鈕

## 手動安裝

### 從 GitHub Releases 安裝

1. 前往 [Releases 頁面](${this.getRepositoryUrl()}/releases)
2. 下載最新版本的 \`main.js\`、\`manifest.json\` 和 \`styles.css\`
3. 在你的 Obsidian 文件庫中創建資料夾：\`.obsidian/plugins/${this.manifest.id}/\`
4. 將下載的檔案放入該資料夾
5. 重新載入 Obsidian
6. 在設定中啟用插件

### 使用 BRAT 安裝 (Beta 版本)

1. 安裝 [BRAT 插件](https://github.com/TfTHacker/obsidian42-brat)
2. 在 BRAT 設定中添加：\`${this.getRepositoryUrl().replace('https://github.com/', '')}\`
3. 啟用插件

## 初始設定

### 1. 配置 Ink-Gateway 連線

1. 開啟插件設定
2. 輸入 Ink-Gateway URL
3. 輸入 API 金鑰
4. 點擊 "測試連線" 驗證設定

### 2. 調整同步設定

1. 設定自動同步間隔
2. 選擇同步範圍
3. 配置離線模式選項

### 3. 自定義介面

1. 選擇顯示選項
2. 設定快捷鍵
3. 調整視窗佈局

## 驗證安裝

安裝完成後，你應該能夠：

- [ ] 在插件列表中看到 "${this.manifest.name}"
- [ ] 開啟插件設定頁面
- [ ] 看到 AI 聊天和搜尋按鈕
- [ ] 成功連線到 Ink-Gateway

## 故障排除

如果遇到安裝問題：

1. 確認 Obsidian 版本 ≥ ${this.manifest.minAppVersion}
2. 檢查插件檔案是否正確放置
3. 重新載入 Obsidian
4. 查看控制台錯誤訊息
5. 參考 [故障排除指南](${this.getDocumentationUrl()}/TROUBLESHOOTING.md)

## 卸載

如需卸載插件：

1. 在設定中停用插件
2. 刪除插件資料夾：\`.obsidian/plugins/${this.manifest.id}/\`
3. 重新載入 Obsidian
`;

        fs.writeFileSync(
            path.join(this.submissionDir, 'submission-docs', 'INSTALLATION.md'),
            instructions
        );

        console.log('  ✓ 安裝說明');
    }

    generateUsageGuide() {
        const guide = `# 使用指南

## 快速開始

### 第一次使用

1. **設定連線**
   - 開啟插件設定
   - 配置 Ink-Gateway URL 和 API 金鑰
   - 測試連線確保正常

2. **開始使用**
   - 創建或開啟筆記
   - 內容會自動同步到 Ink-Gateway
   - 使用 AI 聊天和搜尋功能

## 主要功能使用

### AI 聊天

1. 點擊側邊欄的 AI 聊天圖示
2. 在聊天視窗中輸入問題
3. AI 會基於你的筆記內容回答
4. 聊天歷史會自動保存

### 語義搜尋

1. 點擊搜尋圖示開啟搜尋視窗
2. 輸入搜尋關鍵字或問題
3. 選擇搜尋類型（語義/精確/標籤）
4. 點擊結果可直接跳轉到原文

### 模板使用

1. 創建模板筆記
2. 使用 \`{{slot_name}}\` 定義插槽
3. 在前置資料中定義插槽值
4. 系統會自動識別並管理模板

### 自動同步

- 內容會在你按下 Enter 後自動同步
- 可在設定中調整同步頻率
- 離線時變更會排隊，上線後自動同步

## 進階使用

### 自定義設定

- **同步選項**: 調整同步頻率和範圍
- **快取設定**: 配置快取大小和過期時間
- **介面選項**: 自定義視窗佈局和顯示
- **除錯模式**: 開啟詳細日誌記錄

### 快捷鍵

- \`Ctrl/Cmd + Shift + A\`: 開啟 AI 聊天
- \`Ctrl/Cmd + Shift + S\`: 開啟語義搜尋
- \`Ctrl/Cmd + Shift + T\`: 應用模板
- \`Ctrl/Cmd + Shift + R\`: 手動同步

### 標籤管理

- 使用 \`#tag\` 格式添加標籤
- 標籤會自動同步到 Ink-Gateway
- 支援標籤搜尋和過濾

## 最佳實踐

### 內容組織

1. **使用階層結構**
   - 利用標題層級組織內容
   - 使用項目符號建立關係

2. **善用標籤**
   - 為內容添加相關標籤
   - 使用一致的標籤命名

3. **模板應用**
   - 為重複性內容創建模板
   - 使用屬性系統管理結構化資料

### 效能最佳化

1. **合理使用同步**
   - 避免過於頻繁的同步
   - 大型文件可分段處理

2. **快取管理**
   - 定期清理快取
   - 調整快取大小限制

## 常見問題

### Q: 為什麼同步很慢？
A: 檢查網路連線和 Ink-Gateway 狀態，考慮調整同步設定。

### Q: AI 聊天沒有回應？
A: 確認 API 金鑰正確，檢查 Ink-Gateway 服務狀態。

### Q: 搜尋結果不準確？
A: 嘗試不同的搜尋類型，確保內容已正確同步。

### Q: 插件影響 Obsidian 效能？
A: 調整快取設定，關閉不需要的功能，檢查除錯日誌。

## 獲得幫助

- 📖 [完整文件](${this.getDocumentationUrl()})
- 🐛 [問題回報](${this.getIssuesUrl()})
- 💬 [社群討論](${this.getDiscussionUrl()})
- 📧 [聯絡支援](mailto:support@example.com)
`;

        fs.writeFileSync(
            path.join(this.submissionDir, 'submission-docs', 'USAGE_GUIDE.md'),
            guide
        );

        console.log('  ✓ 使用指南');
    }

    generateDeveloperInfo() {
        const devInfo = `# 開發者資訊

## 專案資訊

- **專案名稱**: ${this.manifest.name}
- **開發者**: ${this.manifest.author}
- **授權**: ${this.getLicenseInfo()}
- **程式語言**: TypeScript
- **建置工具**: esbuild
- **測試框架**: Vitest

## 技術架構

### 核心技術棧

- **前端**: TypeScript, Obsidian API
- **建置**: esbuild, npm scripts
- **測試**: Vitest, Jest
- **程式碼品質**: ESLint, Prettier
- **版本控制**: Git, GitHub

### 專案結構

\`\`\`
src/
├── api/           # API 客戶端
├── ai/            # AI 功能
├── cache/         # 快取系統
├── content/       # 內容管理
├── errors/        # 錯誤處理
├── offline/       # 離線支援
├── performance/   # 效能最佳化
├── search/        # 搜尋功能
├── settings/      # 設定管理
├── template/      # 模板系統
└── types/         # 類型定義
\`\`\`

## 開發環境設定

### 前置要求

- Node.js 16.0.0+
- npm 7.0.0+
- Git

### 安裝步驟

1. Clone 專案
\`\`\`bash
git clone ${this.getRepositoryUrl()}
cd ${this.manifest.id}
\`\`\`

2. 安裝依賴
\`\`\`bash
npm install
\`\`\`

3. 開發建置
\`\`\`bash
npm run dev
\`\`\`

4. 執行測試
\`\`\`bash
npm test
\`\`\`

## 建置和發布

### 開發建置
\`\`\`bash
npm run build
\`\`\`

### 生產建置
\`\`\`bash
npm run build:prod
\`\`\`

### 版本管理
\`\`\`bash
# 更新版本
node scripts/version-manager.js update patch

# 創建發布
node scripts/build-release.js
\`\`\`

## 測試

### 單元測試
\`\`\`bash
npm run test:unit
\`\`\`

### 整合測試
\`\`\`bash
npm run test:integration
\`\`\`

### 覆蓋率測試
\`\`\`bash
npm run test:coverage
\`\`\`

## 程式碼品質

### Linting
\`\`\`bash
npm run lint
npm run lint:fix
\`\`\`

### 格式化
\`\`\`bash
npm run format
\`\`\`

### 類型檢查
\`\`\`bash
npm run type-check
\`\`\`

## 貢獻指南

### 開發流程

1. Fork 專案
2. 創建功能分支
3. 開發和測試
4. 提交 Pull Request

### 程式碼規範

- 使用 TypeScript
- 遵循 ESLint 規則
- 撰寫單元測試
- 更新文件

### 提交訊息格式

\`\`\`
type(scope): description

feat(search): add semantic search functionality
fix(sync): resolve offline sync issue
docs(readme): update installation instructions
\`\`\`

## API 文件

### 核心介面

詳細的 API 文件請參考：
- [API 參考](${this.getDocumentationUrl()}/API_REFERENCE.md)
- [開發者指南](${this.getDocumentationUrl()}/DEVELOPER_GUIDE.md)

## 支援和社群

- **GitHub**: ${this.getRepositoryUrl()}
- **Issues**: ${this.getIssuesUrl()}
- **Discussions**: ${this.getDiscussionUrl()}
- **Email**: ${this.getContactEmail()}

## 授權

本專案採用 ${this.getLicenseInfo()} 授權。
詳細資訊請參考 [LICENSE](${this.getRepositoryUrl()}/blob/main/LICENSE) 檔案。
`;

        fs.writeFileSync(
            path.join(this.submissionDir, 'submission-docs', 'DEVELOPER_INFO.md'),
            devInfo
        );

        console.log('  ✓ 開發者資訊');
    }

    validateSubmission() {
        console.log('🔍 驗證提交內容...');

        // 檢查必要檔案
        const requiredFiles = [
            'plugin-files/main.js',
            'plugin-files/manifest.json',
            'documentation/README.md',
            'submission-docs/PLUGIN_SUMMARY.md'
        ];

        for (const file of requiredFiles) {
            const filePath = path.join(this.submissionDir, file);
            if (!fs.existsSync(filePath)) {
                throw new Error(`提交檔案缺失: ${file}`);
            }
        }

        // 驗證檔案內容
        this.validateSubmissionContent();

        console.log('✅ 提交內容驗證通過');
    }

    validateSubmissionContent() {
        // 驗證 manifest.json
        const manifestPath = path.join(this.submissionDir, 'plugin-files', 'manifest.json');
        const manifest = JSON.parse(fs.readFileSync(manifestPath, 'utf8'));

        if (manifest.version !== this.manifest.version) {
            throw new Error('提交的 manifest.json 版本不一致');
        }

        // 驗證 README.md
        const readmePath = path.join(this.submissionDir, 'documentation', 'README.md');
        const readme = fs.readFileSync(readmePath, 'utf8');

        if (readme.length < 100) {
            throw new Error('README.md 內容過短');
        }

        if (!readme.includes(manifest.name)) {
            throw new Error('README.md 未包含插件名稱');
        }
    }

    createSubmissionChecklist() {
        console.log('📋 創建提交檢查清單...');

        const checklist = `# Obsidian 社群插件提交檢查清單

## 提交前檢查

### 必要檔案
- [ ] \`main.js\` - 插件主程式
- [ ] \`manifest.json\` - 插件清單
- [ ] \`README.md\` - 專案說明
- [ ] \`LICENSE\` - 授權檔案

### 可選檔案
- [ ] \`styles.css\` - 樣式檔案
- [ ] \`icon.png\` - 插件圖示 (100x100)
- [ ] 螢幕截圖

### 程式碼品質
- [ ] 通過所有測試
- [ ] 程式碼覆蓋率 > 80%
- [ ] 無 ESLint 錯誤
- [ ] TypeScript 編譯無錯誤

### 文件完整性
- [ ] README.md 包含完整說明
- [ ] 安裝說明清楚
- [ ] 使用指南詳細
- [ ] API 文件完整

### 功能驗證
- [ ] 所有核心功能正常運作
- [ ] 與 Obsidian 整合良好
- [ ] 無明顯效能問題
- [ ] 錯誤處理完善

### 相容性測試
- [ ] 支援最低 Obsidian 版本
- [ ] 多平台測試 (Windows, macOS, Linux)
- [ ] 與常用插件相容
- [ ] 不同主題下正常顯示

## 提交資訊

### 基本資訊
- **插件名稱**: ${this.manifest.name}
- **插件 ID**: ${this.manifest.id}
- **版本**: ${this.manifest.version}
- **作者**: ${this.manifest.author}
- **最低 Obsidian 版本**: ${this.manifest.minAppVersion}

### 專案連結
- **GitHub 倉庫**: ${this.getRepositoryUrl()}
- **Issues 頁面**: ${this.getIssuesUrl()}
- **文件網站**: ${this.getDocumentationUrl()}

### 提交說明
- [ ] 插件功能描述清楚
- [ ] 安裝說明詳細
- [ ] 使用範例充足
- [ ] 已知問題列出

## 社群插件商店要求

### 技術要求
- [ ] 插件 ID 唯一且符合規範
- [ ] 版本號遵循 SemVer
- [ ] main.js 檔案 < 10MB
- [ ] 無惡意程式碼

### 內容要求
- [ ] 描述長度 10-250 字符
- [ ] 功能說明清楚
- [ ] 不侵犯版權
- [ ] 遵循社群準則

### 維護承諾
- [ ] 承諾持續維護
- [ ] 及時回應問題
- [ ] 定期更新
- [ ] 社群支援

## 提交步驟

1. **準備 Pull Request**
   - Fork [obsidian-releases](https://github.com/obsidianmd/obsidian-releases)
   - 在 \`community-plugins.json\` 中添加插件資訊
   - 創建 Pull Request

2. **填寫 PR 模板**
   - 使用提供的 PR 模板
   - 填寫所有必要資訊
   - 附上檢查清單

3. **等待審核**
   - 回應審核意見
   - 修復發現的問題
   - 保持耐心等待

## 審核後續

### 如果被接受
- [ ] 插件會出現在社群插件商店
- [ ] 設定自動發布流程
- [ ] 監控使用者回饋

### 如果被拒絕
- [ ] 仔細閱讀拒絕原因
- [ ] 修復指出的問題
- [ ] 重新提交

## 注意事項

- 提交後無法修改插件 ID
- 審核過程可能需要數週時間
- 保持專業和耐心的態度
- 遵循社群準則和最佳實踐

---

**檢查完成日期**: ___________
**提交者簽名**: ___________
`;

        fs.writeFileSync(
            path.join(this.submissionDir, 'SUBMISSION_CHECKLIST.md'),
            checklist
        );

        console.log('  ✓ 提交檢查清單');
    }

    generatePRTemplate() {
        console.log('📝 生成 PR 模板...');

        const prTemplate = `# Add ${this.manifest.name} to Community Plugins

## Plugin Information

- **Plugin Name**: ${this.manifest.name}
- **Plugin ID**: ${this.manifest.id}
- **Version**: ${this.manifest.version}
- **Author**: ${this.manifest.author}
- **Repository**: ${this.getRepositoryUrl()}
- **Minimum Obsidian Version**: ${this.manifest.minAppVersion}

## Description

${this.manifest.description}

## Key Features

- 🤖 AI chat functionality with Ink-Gateway integration
- 🔍 Powerful semantic search with vector, graph, and tag search
- 📝 Automatic content processing and sync to multiple databases
- 🎯 Template system for structured content management
- 📊 Hierarchical content parsing with heading and bullet levels
- 🔄 Real-time sync with offline mode support
- 📍 Precise location tracking and navigation

## Technical Details

- **Language**: TypeScript
- **Build Tool**: esbuild
- **Testing**: Vitest with >80% coverage
- **Architecture**: Modular, decoupled design
- **Performance**: Optimized with multi-layer caching

## Quality Assurance

- [ ] All tests pass
- [ ] Code coverage >80%
- [ ] No ESLint errors
- [ ] TypeScript compilation successful
- [ ] Manual testing completed
- [ ] Documentation complete

## Compatibility

- [ ] Tested on Windows
- [ ] Tested on macOS
- [ ] Tested on Linux
- [ ] Compatible with popular plugins
- [ ] Works with different themes

## Files Included

- [ ] \`main.js\` (${this.getFileSize('main.js')})
- [ ] \`manifest.json\`
- [ ] \`styles.css\` ${fs.existsSync('styles.css') ? '✓' : '(not included)'}
- [ ] \`README.md\`
- [ ] \`LICENSE\`

## Community Plugin Store Entry

\`\`\`json
{
  "id": "${this.manifest.id}",
  "name": "${this.manifest.name}",
  "author": "${this.manifest.author}",
  "description": "${this.manifest.description}",
  "repo": "${this.getRepositoryUrl().replace('https://github.com/', '')}"
}
\`\`\`

## Additional Information

### Installation

The plugin can be installed through the Obsidian Community Plugin store or manually from the GitHub releases.

### Configuration

Users need to configure their Ink-Gateway connection in the plugin settings.

### Support

- Documentation: ${this.getDocumentationUrl()}
- Issues: ${this.getIssuesUrl()}
- Discussions: ${this.getDiscussionUrl()}

## Checklist

- [ ] Plugin follows Obsidian plugin guidelines
- [ ] No malicious code
- [ ] Respects user privacy
- [ ] Proper error handling
- [ ] Good user experience
- [ ] Comprehensive documentation
- [ ] Responsive to community feedback

## Screenshots

${this.getScreenshotsList()}

---

I confirm that this plugin meets all the requirements for the Obsidian Community Plugin store and I commit to maintaining it according to community standards.
`;

        fs.writeFileSync(
            path.join(this.submissionDir, 'PR_TEMPLATE.md'),
            prTemplate
        );

        console.log('  ✓ PR 模板');
    }

    // 輔助方法
    getRepositoryUrl() {
        return this.packageJson.repository?.url?.replace('git+', '').replace('.git', '') || 
               `https://github.com/${this.manifest.author}/${this.manifest.id}`;
    }

    getDocumentationUrl() {
        return `${this.getRepositoryUrl()}/blob/main/docs`;
    }

    getIssuesUrl() {
        return `${this.getRepositoryUrl()}/issues`;
    }

    getDiscussionUrl() {
        return `${this.getRepositoryUrl()}/discussions`;
    }

    getContactEmail() {
        return this.packageJson.author?.email || 'support@example.com';
    }

    getLicenseInfo() {
        return this.packageJson.license || 'MIT';
    }

    getFileSize(filename) {
        try {
            const stats = fs.statSync(filename);
            const sizeKB = (stats.size / 1024).toFixed(1);
            return `${sizeKB} KB`;
        } catch (error) {
            return 'Unknown';
        }
    }

    getScreenshotsList() {
        const screenshotsDir = path.join(this.projectRoot, 'screenshots');
        if (fs.existsSync(screenshotsDir)) {
            const screenshots = fs.readdirSync(screenshotsDir);
            return screenshots.map(file => `![Screenshot](screenshots/${file})`).join('\n');
        }
        return '(No screenshots available)';
    }

    printSubmissionInfo() {
        console.log('\n📋 提交準備資訊');
        console.log('=' .repeat(50));
        console.log(`插件名稱: ${this.manifest.name}`);
        console.log(`插件 ID: ${this.manifest.id}`);
        console.log(`版本: ${this.manifest.version}`);
        console.log(`提交目錄: ${this.submissionDir}`);
        
        console.log('\n📁 準備的檔案:');
        const files = [
            'plugin-files/main.js',
            'plugin-files/manifest.json',
            'documentation/README.md',
            'SUBMISSION_CHECKLIST.md',
            'PR_TEMPLATE.md'
        ];
        
        files.forEach(file => {
            const filePath = path.join(this.submissionDir, file);
            if (fs.existsSync(filePath)) {
                console.log(`  ✓ ${file}`);
            }
        });
        
        console.log('\n🚀 下一步:');
        console.log('1. 檢查提交檢查清單');
        console.log('2. Fork obsidian-releases 倉庫');
        console.log('3. 添加插件到 community-plugins.json');
        console.log('4. 使用 PR 模板創建 Pull Request');
        console.log('5. 等待社群審核');
        
        console.log('\n📖 相關連結:');
        console.log(`- 倉庫: ${this.getRepositoryUrl()}`);
        console.log(`- 文件: ${this.getDocumentationUrl()}`);
        console.log(`- 問題: ${this.getIssuesUrl()}`);
        console.log('- 社群插件商店: https://github.com/obsidianmd/obsidian-releases');
    }
}

// 主執行邏輯
async function main() {
    try {
        const prep = new CommunityPluginPrep();
        await prep.prepareSubmission();
    } catch (error) {
        console.error('❌ 提交準備失敗:', error.message);
        process.exit(1);
    }
}

if (require.main === module) {
    main();
}

module.exports = { CommunityPluginPrep };