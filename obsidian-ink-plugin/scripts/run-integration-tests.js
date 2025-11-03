#!/usr/bin/env node

/**
 * 整合測試執行腳本
 * 執行完整的系統整合測試套件
 */

const { execSync } = require('child_process');
const fs = require('fs');
const path = require('path');

// 測試配置
const TEST_CONFIG = {
    timeout: 60000, // 60秒超時
    retries: 2,
    coverage: true,
    verbose: true
};

// 測試套件
const TEST_SUITES = [
    {
        name: '完整系統整合測試',
        path: 'tests/integration/comprehensive-integration.test.ts',
        description: '驗證所有需求 1.1-10.7 的整合功能'
    },
    {
        name: '效能和穩定性測試',
        path: 'tests/integration/performance-stability.test.ts',
        description: '測試插件在各種負載和壓力情況下的表現'
    },
    {
        name: '端到端使用場景測試',
        path: 'tests/integration/end-to-end-scenarios.test.ts',
        description: '模擬真實使用者工作流程'
    },
    {
        name: '現有端到端測試',
        path: 'tests/integration/end-to-end.test.ts',
        description: '原有的端到端測試'
    }
];

class IntegrationTestRunner {
    constructor() {
        this.results = [];
        this.startTime = Date.now();
    }

    async runAllTests() {
        console.log('🚀 開始執行整合測試套件...\n');
        
        // 檢查環境
        this.checkEnvironment();
        
        // 執行測試前準備
        await this.setupTestEnvironment();
        
        // 執行所有測試套件
        for (const suite of TEST_SUITES) {
            await this.runTestSuite(suite);
        }
        
        // 生成報告
        this.generateReport();
        
        // 清理
        await this.cleanup();
    }

    checkEnvironment() {
        console.log('🔍 檢查測試環境...');
        
        // 檢查必要的檔案
        const requiredFiles = [
            'package.json',
            'vitest.config.ts',
            'tsconfig.json'
        ];
        
        for (const file of requiredFiles) {
            if (!fs.existsSync(file)) {
                throw new Error(`缺少必要檔案: ${file}`);
            }
        }
        
        // 檢查測試檔案
        for (const suite of TEST_SUITES) {
            if (!fs.existsSync(suite.path)) {
                console.warn(`⚠️  測試檔案不存在: ${suite.path}`);
            }
        }
        
        console.log('✅ 環境檢查完成\n');
    }

    async setupTestEnvironment() {
        console.log('⚙️  設置測試環境...');
        
        try {
            // 安裝依賴
            console.log('  📦 檢查依賴...');
            execSync('npm ci', { stdio: 'pipe' });
            
            // 編譯 TypeScript
            console.log('  🔨 編譯 TypeScript...');
            execSync('npm run build', { stdio: 'pipe' });
            
            console.log('✅ 測試環境設置完成\n');
        } catch (error) {
            console.error('❌ 測試環境設置失敗:', error.message);
            process.exit(1);
        }
    }

    async runTestSuite(suite) {
        console.log(`📋 執行測試套件: ${suite.name}`);
        console.log(`   描述: ${suite.description}`);
        console.log(`   檔案: ${suite.path}`);
        
        const result = {
            name: suite.name,
            path: suite.path,
            startTime: Date.now(),
            success: false,
            output: '',
            error: null,
            coverage: null
        };
        
        try {
            // 構建測試命令
            const testCommand = this.buildTestCommand(suite.path);
            
            console.log(`   🏃 執行命令: ${testCommand}`);
            
            // 執行測試
            const output = execSync(testCommand, { 
                encoding: 'utf8',
                timeout: TEST_CONFIG.timeout,
                maxBuffer: 1024 * 1024 * 10 // 10MB buffer
            });
            
            result.output = output;
            result.success = true;
            
            // 解析覆蓋率
            if (TEST_CONFIG.coverage) {
                result.coverage = this.parseCoverage(output);
            }
            
            console.log('   ✅ 測試通過');
            
        } catch (error) {
            result.error = error.message;
            result.output = error.stdout || error.message;
            
            console.log('   ❌ 測試失敗');
            console.log(`   錯誤: ${error.message}`);
            
            // 重試機制
            if (TEST_CONFIG.retries > 0) {
                console.log(`   🔄 重試測試 (剩餘 ${TEST_CONFIG.retries} 次)...`);
                TEST_CONFIG.retries--;
                return this.runTestSuite(suite);
            }
        }
        
        result.endTime = Date.now();
        result.duration = result.endTime - result.startTime;
        
        this.results.push(result);
        console.log(`   ⏱️  執行時間: ${result.duration}ms\n`);
    }

    buildTestCommand(testPath) {
        let command = 'npx vitest run';
        
        // 添加測試檔案
        command += ` "${testPath}"`;
        
        // 添加選項
        if (TEST_CONFIG.coverage) {
            command += ' --coverage';
        }
        
        if (TEST_CONFIG.verbose) {
            command += ' --reporter=verbose';
        }
        
        // 設置超時
        command += ` --testTimeout=${TEST_CONFIG.timeout}`;
        
        return command;
    }

    parseCoverage(output) {
        try {
            // 嘗試從輸出中解析覆蓋率資訊
            const coverageMatch = output.match(/All files\s+\|\s+([\d.]+)\s+\|\s+([\d.]+)\s+\|\s+([\d.]+)\s+\|\s+([\d.]+)/);
            
            if (coverageMatch) {
                return {
                    statements: parseFloat(coverageMatch[1]),
                    branches: parseFloat(coverageMatch[2]),
                    functions: parseFloat(coverageMatch[3]),
                    lines: parseFloat(coverageMatch[4])
                };
            }
        } catch (error) {
            console.warn('無法解析覆蓋率資訊:', error.message);
        }
        
        return null;
    }

    generateReport() {
        const endTime = Date.now();
        const totalDuration = endTime - this.startTime;
        
        console.log('📊 測試報告');
        console.log('=' .repeat(50));
        
        // 總體統計
        const totalTests = this.results.length;
        const passedTests = this.results.filter(r => r.success).length;
        const failedTests = totalTests - passedTests;
        
        console.log(`總測試套件: ${totalTests}`);
        console.log(`通過: ${passedTests}`);
        console.log(`失敗: ${failedTests}`);
        console.log(`總執行時間: ${totalDuration}ms`);
        console.log('');
        
        // 詳細結果
        console.log('詳細結果:');
        console.log('-'.repeat(50));
        
        for (const result of this.results) {
            const status = result.success ? '✅ PASS' : '❌ FAIL';
            console.log(`${status} ${result.name} (${result.duration}ms)`);
            
            if (result.coverage) {
                console.log(`     覆蓋率: ${result.coverage.lines}% 行, ${result.coverage.functions}% 函數`);
            }
            
            if (!result.success && result.error) {
                console.log(`     錯誤: ${result.error}`);
            }
        }
        
        console.log('');
        
        // 覆蓋率摘要
        if (TEST_CONFIG.coverage) {
            this.generateCoverageSummary();
        }
        
        // 建議
        this.generateRecommendations();
        
        // 保存報告到檔案
        this.saveReportToFile();
    }

    generateCoverageSummary() {
        const coverageResults = this.results
            .filter(r => r.coverage)
            .map(r => r.coverage);
        
        if (coverageResults.length === 0) {
            return;
        }
        
        const avgCoverage = {
            statements: coverageResults.reduce((sum, c) => sum + c.statements, 0) / coverageResults.length,
            branches: coverageResults.reduce((sum, c) => sum + c.branches, 0) / coverageResults.length,
            functions: coverageResults.reduce((sum, c) => sum + c.functions, 0) / coverageResults.length,
            lines: coverageResults.reduce((sum, c) => sum + c.lines, 0) / coverageResults.length
        };
        
        console.log('覆蓋率摘要:');
        console.log(`  語句: ${avgCoverage.statements.toFixed(1)}%`);
        console.log(`  分支: ${avgCoverage.branches.toFixed(1)}%`);
        console.log(`  函數: ${avgCoverage.functions.toFixed(1)}%`);
        console.log(`  行數: ${avgCoverage.lines.toFixed(1)}%`);
        console.log('');
    }

    generateRecommendations() {
        console.log('建議:');
        
        const failedTests = this.results.filter(r => !r.success);
        
        if (failedTests.length === 0) {
            console.log('🎉 所有測試都通過了！插件已準備好進行部署。');
        } else {
            console.log('⚠️  有測試失敗，建議在部署前修復以下問題:');
            
            failedTests.forEach(test => {
                console.log(`  - ${test.name}: ${test.error}`);
            });
        }
        
        // 效能建議
        const slowTests = this.results.filter(r => r.duration > 10000); // 超過 10 秒
        if (slowTests.length > 0) {
            console.log('⏱️  以下測試執行較慢，可能需要最佳化:');
            slowTests.forEach(test => {
                console.log(`  - ${test.name}: ${test.duration}ms`);
            });
        }
        
        console.log('');
    }

    saveReportToFile() {
        const reportData = {
            timestamp: new Date().toISOString(),
            summary: {
                total: this.results.length,
                passed: this.results.filter(r => r.success).length,
                failed: this.results.filter(r => !r.success).length,
                duration: Date.now() - this.startTime
            },
            results: this.results
        };
        
        const reportPath = path.join('coverage', 'integration-test-report.json');
        
        // 確保目錄存在
        const reportDir = path.dirname(reportPath);
        if (!fs.existsSync(reportDir)) {
            fs.mkdirSync(reportDir, { recursive: true });
        }
        
        fs.writeFileSync(reportPath, JSON.stringify(reportData, null, 2));
        console.log(`📄 測試報告已保存到: ${reportPath}`);
    }

    async cleanup() {
        console.log('🧹 清理測試環境...');
        
        try {
            // 清理臨時檔案
            // 這裡可以添加清理邏輯
            
            console.log('✅ 清理完成');
        } catch (error) {
            console.warn('⚠️  清理時發生錯誤:', error.message);
        }
    }
}

// 主執行邏輯
async function main() {
    try {
        const runner = new IntegrationTestRunner();
        await runner.runAllTests();
        
        // 根據測試結果設置退出碼
        const failedTests = runner.results.filter(r => !r.success).length;
        process.exit(failedTests > 0 ? 1 : 0);
        
    } catch (error) {
        console.error('❌ 測試執行失敗:', error.message);
        console.error(error.stack);
        process.exit(1);
    }
}

// 處理未捕獲的異常
process.on('uncaughtException', (error) => {
    console.error('❌ 未捕獲的異常:', error.message);
    process.exit(1);
});

process.on('unhandledRejection', (reason, promise) => {
    console.error('❌ 未處理的 Promise 拒絕:', reason);
    process.exit(1);
});

// 執行主函數
if (require.main === module) {
    main();
}

module.exports = { IntegrationTestRunner };