#!/usr/bin/env node

/**
 * 版本管理腳本
 * 自動化版本號更新和發布流程
 */

const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');

class VersionManager {
    constructor() {
        this.projectRoot = process.cwd();
        this.packageJsonPath = path.join(this.projectRoot, 'package.json');
        this.manifestPath = path.join(this.projectRoot, 'manifest.json');
        this.versionsPath = path.join(this.projectRoot, 'versions.json');
    }

    async updateVersion(versionType = 'patch') {
        console.log(`🔄 更新版本 (${versionType})...`);

        try {
            // 1. 讀取當前版本
            const currentVersion = this.getCurrentVersion();
            console.log(`當前版本: ${currentVersion}`);

            // 2. 計算新版本
            const newVersion = this.calculateNewVersion(currentVersion, versionType);
            console.log(`新版本: ${newVersion}`);

            // 3. 驗證版本格式
            this.validateVersion(newVersion);

            // 4. 更新所有版本檔案
            this.updateVersionFiles(newVersion);

            // 5. 更新 versions.json
            this.updateVersionsJson(newVersion);

            // 6. 創建 Git 標籤
            this.createGitTag(newVersion);

            // 7. 生成更新日誌
            this.generateChangelog(currentVersion, newVersion);

            console.log(`✅ 版本更新完成: ${currentVersion} → ${newVersion}`);
            
            return newVersion;

        } catch (error) {
            console.error('❌ 版本更新失敗:', error.message);
            throw error;
        }
    }

    getCurrentVersion() {
        if (!fs.existsSync(this.packageJsonPath)) {
            throw new Error('package.json 檔案不存在');
        }

        const packageJson = JSON.parse(fs.readFileSync(this.packageJsonPath, 'utf8'));
        return packageJson.version;
    }

    calculateNewVersion(currentVersion, versionType) {
        const versionParts = currentVersion.split('.').map(Number);
        
        if (versionParts.length !== 3) {
            throw new Error(`無效的版本格式: ${currentVersion}`);
        }

        let [major, minor, patch] = versionParts;

        switch (versionType) {
            case 'major':
                major += 1;
                minor = 0;
                patch = 0;
                break;
            case 'minor':
                minor += 1;
                patch = 0;
                break;
            case 'patch':
                patch += 1;
                break;
            default:
                // 直接指定版本號
                if (!/^\d+\.\d+\.\d+$/.test(versionType)) {
                    throw new Error(`無效的版本類型或格式: ${versionType}`);
                }
                return versionType;
        }

        return `${major}.${minor}.${patch}`;
    }

    validateVersion(version) {
        if (!/^\d+\.\d+\.\d+$/.test(version)) {
            throw new Error(`無效的版本格式: ${version}`);
        }

        const currentVersion = this.getCurrentVersion();
        if (this.compareVersions(version, currentVersion) <= 0) {
            throw new Error(`新版本 ${version} 必須大於當前版本 ${currentVersion}`);
        }
    }

    updateVersionFiles(newVersion) {
        console.log('📝 更新版本檔案...');

        // 更新 package.json
        const packageJson = JSON.parse(fs.readFileSync(this.packageJsonPath, 'utf8'));
        packageJson.version = newVersion;
        fs.writeFileSync(this.packageJsonPath, JSON.stringify(packageJson, null, 2) + '\n');
        console.log('  ✓ package.json');

        // 更新 manifest.json
        if (fs.existsSync(this.manifestPath)) {
            const manifest = JSON.parse(fs.readFileSync(this.manifestPath, 'utf8'));
            manifest.version = newVersion;
            fs.writeFileSync(this.manifestPath, JSON.stringify(manifest, null, 2) + '\n');
            console.log('  ✓ manifest.json');
        }

        // 更新 version-bump.mjs 中的版本資訊
        const versionBumpPath = path.join(this.projectRoot, 'version-bump.mjs');
        if (fs.existsSync(versionBumpPath)) {
            let content = fs.readFileSync(versionBumpPath, 'utf8');
            content = content.replace(/const targetVersion = ['"][\d.]+['"]/, `const targetVersion = "${newVersion}"`);
            fs.writeFileSync(versionBumpPath, content);
            console.log('  ✓ version-bump.mjs');
        }
    }

    updateVersionsJson(newVersion) {
        console.log('📋 更新版本歷史...');

        let versions = {};
        
        if (fs.existsSync(this.versionsPath)) {
            versions = JSON.parse(fs.readFileSync(this.versionsPath, 'utf8'));
        }

        // 添加新版本
        versions[newVersion] = this.getMinAppVersion();

        // 保持版本排序
        const sortedVersions = {};
        Object.keys(versions)
            .sort((a, b) => this.compareVersions(b, a)) // 降序排列
            .forEach(version => {
                sortedVersions[version] = versions[version];
            });

        fs.writeFileSync(this.versionsPath, JSON.stringify(sortedVersions, null, 2) + '\n');
        console.log('  ✓ versions.json');
    }

    getMinAppVersion() {
        if (fs.existsSync(this.manifestPath)) {
            const manifest = JSON.parse(fs.readFileSync(this.manifestPath, 'utf8'));
            return manifest.minAppVersion || '0.15.0';
        }
        return '0.15.0';
    }

    createGitTag(version) {
        console.log('🏷️  創建 Git 標籤...');

        try {
            // 檢查是否有未提交的變更
            const status = execSync('git status --porcelain', { encoding: 'utf8' });
            if (status.trim()) {
                console.log('  提交版本變更...');
                execSync('git add package.json manifest.json versions.json version-bump.mjs', { stdio: 'pipe' });
                execSync(`git commit -m "chore: bump version to ${version}"`, { stdio: 'pipe' });
            }

            // 創建標籤
            execSync(`git tag -a v${version} -m "Release version ${version}"`, { stdio: 'pipe' });
            console.log(`  ✓ 創建標籤 v${version}`);

        } catch (error) {
            console.warn('⚠️  Git 操作失敗:', error.message);
        }
    }

    generateChangelog(oldVersion, newVersion) {
        console.log('📝 生成更新日誌...');

        const changelogPath = path.join(this.projectRoot, 'CHANGELOG.md');
        const date = new Date().toISOString().split('T')[0];
        
        let changelog = '';
        
        if (fs.existsSync(changelogPath)) {
            changelog = fs.readFileSync(changelogPath, 'utf8');
        } else {
            changelog = '# 更新日誌\n\n所有重要變更都會記錄在此檔案中。\n\n';
        }

        // 生成新版本條目
        const newEntry = this.generateChangelogEntry(newVersion, date);
        
        // 插入到檔案開頭
        const lines = changelog.split('\n');
        const insertIndex = lines.findIndex(line => line.startsWith('## ')) || 2;
        lines.splice(insertIndex, 0, newEntry);
        
        fs.writeFileSync(changelogPath, lines.join('\n'));
        console.log('  ✓ CHANGELOG.md');
    }

    generateChangelogEntry(version, date) {
        return `## [${version}] - ${date}

### 新增
- 新功能和改進

### 變更
- 現有功能的變更

### 修復
- 錯誤修復

### 移除
- 移除的功能

`;
    }

    async createRelease(version) {
        console.log(`🚀 創建發布 v${version}...`);

        try {
            // 1. 建置發布版本
            console.log('  建置插件...');
            execSync('node scripts/build-release.js', { stdio: 'inherit' });

            // 2. 推送到遠端
            console.log('  推送到 Git 遠端...');
            execSync('git push origin main', { stdio: 'pipe' });
            execSync(`git push origin v${version}`, { stdio: 'pipe' });

            // 3. 創建 GitHub Release (如果有 GitHub CLI)
            try {
                const releaseNotes = this.generateReleaseNotes(version);
                const releaseNotesFile = path.join('release', 'RELEASE_NOTES.md');
                
                execSync(`gh release create v${version} release/*.zip release/main.js release/manifest.json --title "Release v${version}" --notes-file "${releaseNotesFile}"`, { stdio: 'pipe' });
                console.log('  ✓ GitHub Release 創建成功');
            } catch (error) {
                console.log('  ⚠️  GitHub Release 創建失敗 (可能需要手動創建)');
            }

            console.log(`✅ 發布 v${version} 創建完成`);

        } catch (error) {
            console.error('❌ 發布創建失敗:', error.message);
            throw error;
        }
    }

    generateReleaseNotes(version) {
        const changelogPath = path.join(this.projectRoot, 'CHANGELOG.md');
        
        if (!fs.existsSync(changelogPath)) {
            return `Release v${version}`;
        }

        const changelog = fs.readFileSync(changelogPath, 'utf8');
        const lines = changelog.split('\n');
        
        // 找到當前版本的開始和結束
        const startIndex = lines.findIndex(line => line.includes(`[${version}]`));
        if (startIndex === -1) {
            return `Release v${version}`;
        }

        const endIndex = lines.findIndex((line, index) => 
            index > startIndex && line.startsWith('## [')
        );

        const versionLines = lines.slice(startIndex + 1, endIndex === -1 ? undefined : endIndex);
        return versionLines.join('\n').trim();
    }

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

    listVersions() {
        console.log('📋 版本歷史:');
        
        if (fs.existsSync(this.versionsPath)) {
            const versions = JSON.parse(fs.readFileSync(this.versionsPath, 'utf8'));
            
            Object.entries(versions).forEach(([version, minAppVersion]) => {
                console.log(`  ${version} (Obsidian >= ${minAppVersion})`);
            });
        } else {
            console.log('  無版本歷史記錄');
        }
    }
}

// CLI 介面
async function main() {
    const args = process.argv.slice(2);
    const command = args[0];
    const versionType = args[1] || 'patch';

    const versionManager = new VersionManager();

    try {
        switch (command) {
            case 'update':
            case 'bump':
                await versionManager.updateVersion(versionType);
                break;
                
            case 'release':
                const version = versionManager.getCurrentVersion();
                await versionManager.createRelease(version);
                break;
                
            case 'list':
                versionManager.listVersions();
                break;
                
            case 'current':
                console.log(versionManager.getCurrentVersion());
                break;
                
            default:
                console.log(`
使用方式:
  node scripts/version-manager.js update [patch|minor|major|x.y.z]  # 更新版本
  node scripts/version-manager.js release                            # 創建發布
  node scripts/version-manager.js list                               # 列出版本歷史
  node scripts/version-manager.js current                            # 顯示當前版本

範例:
  node scripts/version-manager.js update patch    # 1.0.0 → 1.0.1
  node scripts/version-manager.js update minor    # 1.0.0 → 1.1.0
  node scripts/version-manager.js update major    # 1.0.0 → 2.0.0
  node scripts/version-manager.js update 1.2.3    # 直接設定版本
`);
                process.exit(1);
        }
    } catch (error) {
        console.error('❌ 操作失敗:', error.message);
        process.exit(1);
    }
}

if (require.main === module) {
    main();
}

module.exports = { VersionManager };