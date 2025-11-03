#!/usr/bin/env node

/**
 * 插件發布建置腳本
 * 自動化插件打包和發布準備流程
 */

const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');
const crypto = require('crypto');

class ReleaseBuilder {
    constructor() {
        this.projectRoot = process.cwd();
        this.packageJson = require(path.join(this.projectRoot, 'package.json'));
        this.manifest = require(path.join(this.projectRoot, 'manifest.json'));
        this.buildDir = path.join(this.projectRoot, 'build');
        this.releaseDir = path.join(this.projectRoot, 'release');
    }

    async buildRelease() {
        console.log('🚀 開始建置插件發布版本...\n');

        try {
            // 1. 驗證環境和版本
            this.validateEnvironment();
            
            // 2. 清理舊的建置檔案
            this.cleanBuildDirectories();
            
            // 3. 執行測試
            await this.runTests();
            
            // 4. 建置插件
            await this.buildPlugin();
            
            // 5. 驗證建置結果
            this.validateBuild();
            
            // 6. 創建發布包
            this.createReleasePackage();
            
            // 7. 生成校驗和
            this.generateChecksums();
            
            // 8. 創建發布說明
            this.createReleaseNotes();
            
            // 9. 驗證 Obsidian 社群插件要求
            this.validateCommunityPluginRequirements();
            
            console.log('✅ 插件發布版本建置完成！');
            this.printReleaseInfo();
            
        } catch (error) {
            console.error('❌ 建置失敗:', error.message);
            process.exit(1);
        }
    }

    validateEnvironment() {
        console.log('🔍 驗證建置環境...');
        
        // 檢查必要檔案
        const requiredFiles = [
            'package.json',
            'manifest.json',
            'esbuild.config.mjs',
            'tsconfig.json'
        ];
        
        for (const file of requiredFiles) {
            if (!fs.existsSync(file)) {
                throw new Error(`缺少必要檔案: ${file}`);
            }
        }
        
        // 檢查版本一致性
        if (this.packageJson.version !== this.manifest.version) {
            throw new Error(`版本不一致: package.json (${this.packageJson.version}) vs manifest.json (${this.manifest.version})`);
        }
        
        // 檢查 Node.js 版本
        const nodeVersion = process.version;
        const requiredNodeVersion = '16.0.0';
        if (this.compareVersions(nodeVersion.slice(1), requiredNodeVersion) < 0) {
            throw new Error(`需要 Node.js ${requiredNodeVersion} 或更高版本，當前版本: ${nodeVersion}`);
        }
        
        console.log('✅ 環境驗證通過');
    }

    cleanBuildDirectories() {
        console.log('🧹 清理建置目錄...');
        
        const dirsToClean = [this.buildDir, this.releaseDir];
        
        for (const dir of dirsToClean) {
            if (fs.existsSync(dir)) {
                fs.rmSync(dir, { recursive: true, force: true });
            }
            fs.mkdirSync(dir, { recursive: true });
        }
        
        console.log('✅ 建置目錄清理完成');
    }

    async runTests() {
        console.log('🧪 執行測試套件...');
        
        try {
            // 執行單元測試
            console.log('  執行單元測試...');
            execSync('npm run test', { stdio: 'pipe' });
            
            // 執行整合測試
            console.log('  執行整合測試...');
            execSync('node scripts/run-integration-tests.js', { stdio: 'pipe' });
            
            // 執行覆蓋率檢查
            console.log('  檢查測試覆蓋率...');
            const coverageOutput = execSync('npm run test:coverage', { encoding: 'utf8' });
            
            // 解析覆蓋率
            const coverage = this.parseCoverageOutput(coverageOutput);
            if (coverage.lines < 80) {
                console.warn(`⚠️  測試覆蓋率較低: ${coverage.lines}% (建議 > 80%)`);
            }
            
            console.log('✅ 所有測試通過');
            
        } catch (error) {
            throw new Error(`測試失敗: ${error.message}`);
        }
    }

    async buildPlugin() {
        console.log('🔨 建置插件...');
        
        try {
            // 執行 TypeScript 編譯
            console.log('  編譯 TypeScript...');
            execSync('npm run build', { stdio: 'pipe' });
            
            // 檢查建置輸出
            const mainJsPath = path.join(this.projectRoot, 'main.js');
            if (!fs.existsSync(mainJsPath)) {
                throw new Error('建置失敗: main.js 檔案未生成');
            }
            
            // 檢查檔案大小
            const stats = fs.statSync(mainJsPath);
            const fileSizeMB = stats.size / (1024 * 1024);
            
            if (fileSizeMB > 10) {
                console.warn(`⚠️  建置檔案較大: ${fileSizeMB.toFixed(2)}MB`);
            }
            
            console.log(`✅ 插件建置完成 (${fileSizeMB.toFixed(2)}MB)`);
            
        } catch (error) {
            throw new Error(`建置失敗: ${error.message}`);
        }
    }

    validateBuild() {
        console.log('🔍 驗證建置結果...');
        
        const requiredFiles = ['main.js', 'manifest.json'];
        
        for (const file of requiredFiles) {
            const filePath = path.join(this.projectRoot, file);
            if (!fs.existsSync(filePath)) {
                throw new Error(`建置檔案缺失: ${file}`);
            }
        }
        
        // 驗證 manifest.json 格式
        try {
            const manifest = JSON.parse(fs.readFileSync('manifest.json', 'utf8'));
            
            const requiredFields = ['id', 'name', 'version', 'minAppVersion', 'description', 'author'];
            for (const field of requiredFields) {
                if (!manifest[field]) {
                    throw new Error(`manifest.json 缺少必要欄位: ${field}`);
                }
            }
            
        } catch (error) {
            throw new Error(`manifest.json 格式錯誤: ${error.message}`);
        }
        
        // 驗證 main.js 可以載入
        try {
            const mainJs = fs.readFileSync('main.js', 'utf8');
            if (!mainJs.includes('Plugin')) {
                throw new Error('main.js 似乎不包含有效的插件代碼');
            }
        } catch (error) {
            throw new Error(`main.js 驗證失敗: ${error.message}`);
        }
        
        console.log('✅ 建置結果驗證通過');
    }

    createReleasePackage() {
        console.log('📦 創建發布包...');
        
        const version = this.manifest.version;
        const releaseFiles = [
            'main.js',
            'manifest.json',
            'styles.css'
        ];
        
        // 複製發布檔案
        for (const file of releaseFiles) {
            const srcPath = path.join(this.projectRoot, file);
            const destPath = path.join(this.releaseDir, file);
            
            if (fs.existsSync(srcPath)) {
                fs.copyFileSync(srcPath, destPath);
                console.log(`  ✓ ${file}`);
            } else if (file === 'styles.css') {
                // styles.css 是可選的
                console.log(`  - ${file} (可選，跳過)`);
            } else {
                throw new Error(`發布檔案缺失: ${file}`);
            }
        }
        
        // 創建版本化的 ZIP 檔案
        const zipFileName = `obsidian-ink-plugin-${version}.zip`;
        const zipPath = path.join(this.releaseDir, zipFileName);
        
        try {
            execSync(`cd "${this.releaseDir}" && zip -r "${zipFileName}" main.js manifest.json ${fs.existsSync(path.join(this.releaseDir, 'styles.css')) ? 'styles.css' : ''}`, { stdio: 'pipe' });
            console.log(`  ✓ ${zipFileName}`);
        } catch (error) {
            throw new Error(`創建 ZIP 檔案失敗: ${error.message}`);
        }
        
        console.log('✅ 發布包創建完成');
    }

    generateChecksums() {
        console.log('🔐 生成校驗和...');
        
        const files = fs.readdirSync(this.releaseDir);
        const checksums = {};
        
        for (const file of files) {
            if (file.endsWith('.zip') || file.endsWith('.js') || file.endsWith('.json')) {
                const filePath = path.join(this.releaseDir, file);
                const content = fs.readFileSync(filePath);
                
                checksums[file] = {
                    md5: crypto.createHash('md5').update(content).digest('hex'),
                    sha256: crypto.createHash('sha256').update(content).digest('hex')
                };
                
                console.log(`  ✓ ${file}`);
            }
        }
        
        // 保存校驗和檔案
        const checksumPath = path.join(this.releaseDir, 'checksums.json');
        fs.writeFileSync(checksumPath, JSON.stringify(checksums, null, 2));
        
        console.log('✅ 校驗和生成完成');
    }

    createReleaseNotes() {
        console.log('📝 創建發布說明...');
        
        const version = this.manifest.version;
        const releaseNotes = this.generateReleaseNotes(version);
        
        const releaseNotesPath = path.join(this.releaseDir, 'RELEASE_NOTES.md');
        fs.writeFileSync(releaseNotesPath, releaseNotes);
        
        console.log('✅ 發布說明創建完成');
    }

    validateCommunityPluginRequirements() {
        console.log('🔍 驗證 Obsidian 社群插件要求...');
        
        const manifest = this.manifest;
        
        // 檢查必要欄位
        const requiredFields = {
            'id': '插件 ID',
            'name': '插件名稱',
            'version': '版本號',
            'minAppVersion': '最低 Obsidian 版本',
            'description': '插件描述',
            'author': '作者'
        };
        
        for (const [field, description] of Object.entries(requiredFields)) {
            if (!manifest[field]) {
                throw new Error(`缺少必要欄位 ${field} (${description})`);
            }
        }
        
        // 檢查版本格式
        if (!/^\d+\.\d+\.\d+$/.test(manifest.version)) {
            throw new Error(`版本號格式錯誤: ${manifest.version} (應為 x.y.z 格式)`);
        }
        
        // 檢查 ID 格式
        if (!/^[a-z0-9-]+$/.test(manifest.id)) {
            throw new Error(`插件 ID 格式錯誤: ${manifest.id} (只能包含小寫字母、數字和連字符)`);
        }
        
        // 檢查描述長度
        if (manifest.description.length < 10 || manifest.description.length > 250) {
            throw new Error(`描述長度不當: ${manifest.description.length} 字符 (應為 10-250 字符)`);
        }
        
        // 檢查檔案大小限制
        const mainJsSize = fs.statSync('main.js').size;
        const maxSize = 10 * 1024 * 1024; // 10MB
        
        if (mainJsSize > maxSize) {
            throw new Error(`main.js 檔案過大: ${(mainJsSize / 1024 / 1024).toFixed(2)}MB (限制: 10MB)`);
        }
        
        console.log('✅ Obsidian 社群插件要求驗證通過');
    }

    generateReleaseNotes(version) {
        const date = new Date().toISOString().split('T')[0];
        
        return `# Obsidian Ink Plugin v${version}

發布日期: ${date}

## 新功能

- ✨ 完整的 AI 聊天功能，支援與 Ink-Gateway 系統整合
- 🔍 強大的語義搜尋功能，支援向量、圖形和標籤搜尋
- 📝 自動內容處理和同步到三個資料庫
- 🎯 模板系統，支援結構化內容管理
- 📊 階層內容解析，支援標題和項目符號層級
- 🔄 即時同步功能，支援離線模式
- 📍 精確的位置追蹤和導航功能
- 📄 文件 ID 分頁管理系統

## 技術特性

- 🏗️ 解耦架構設計，易於擴展到其他筆記應用
- ⚡ 高效能快取系統
- 🛡️ 完善的錯誤處理和重試機制
- 🔧 豐富的設定選項和故障排除功能
- 📱 響應式使用者介面
- 🧪 完整的測試覆蓋率

## 系統要求

- Obsidian v${this.manifest.minAppVersion} 或更高版本
- 有效的 Ink-Gateway API 連線

## 安裝方式

### 方式 1: Obsidian 社群插件商店 (推薦)
1. 開啟 Obsidian 設定
2. 前往「社群插件」
3. 搜尋「Ink Plugin」
4. 點擊安裝並啟用

### 方式 2: 手動安裝
1. 下載最新版本的 \`main.js\` 和 \`manifest.json\`
2. 在 Obsidian 文件庫中創建 \`.obsidian/plugins/obsidian-ink-plugin/\` 目錄
3. 將檔案複製到該目錄
4. 重新載入 Obsidian 並啟用插件

## 配置

1. 在插件設定中配置 Ink-Gateway URL 和 API 金鑰
2. 測試連線確保正常運作
3. 根據需要調整同步和快取設定

## 支援

- 📖 [使用者指南](docs/USER_GUIDE.md)
- 🔧 [故障排除](docs/TROUBLESHOOTING.md)
- 💬 [GitHub Issues](https://github.com/your-username/obsidian-ink-plugin/issues)

## 更新日誌

詳細的更新日誌請參閱 [CHANGELOG.md](CHANGELOG.md)

---

感謝使用 Obsidian Ink Plugin！`;
    }

    printReleaseInfo() {
        console.log('\n📋 發布資訊');
        console.log('=' .repeat(50));
        console.log(`版本: ${this.manifest.version}`);
        console.log(`插件名稱: ${this.manifest.name}`);
        console.log(`作者: ${this.manifest.author}`);
        console.log(`發布目錄: ${this.releaseDir}`);
        
        // 列出發布檔案
        console.log('\n📁 發布檔案:');
        const files = fs.readdirSync(this.releaseDir);
        files.forEach(file => {
            const filePath = path.join(this.releaseDir, file);
            const stats = fs.statSync(filePath);
            const size = (stats.size / 1024).toFixed(1);
            console.log(`  ${file} (${size} KB)`);
        });
        
        console.log('\n🚀 下一步:');
        console.log('1. 檢查發布檔案');
        console.log('2. 測試插件安裝');
        console.log('3. 提交到 Obsidian 社群插件商店');
        console.log('4. 創建 GitHub Release');
    }

    // 輔助方法
    compareVersions(version1, version2) {
        const v1parts = version1.split('.').map(Number);
        const v2parts = version2.split('.').map(Number);
        
        for (let i = 0; i < Math.max(v1parts.length, v2parts.length); i++) {
            const v1part = v1parts[i] || 0;
            const v2part = v2parts[i] || 0;
            
            if (v1part < v2part) return -1;
            if (v1part > v2part) return 1;
        }
        
        return 0;
    }

    parseCoverageOutput(output) {
        try {
            const match = output.match(/All files\s+\|\s+([\d.]+)/);
            return {
                lines: match ? parseFloat(match[1]) : 0
            };
        } catch (error) {
            return { lines: 0 };
        }
    }
}

// 主執行邏輯
async function main() {
    try {
        const builder = new ReleaseBuilder();
        await builder.buildRelease();
    } catch (error) {
        console.error('❌ 發布建置失敗:', error.message);
        process.exit(1);
    }
}

if (require.main === module) {
    main();
}

module.exports = { ReleaseBuilder };