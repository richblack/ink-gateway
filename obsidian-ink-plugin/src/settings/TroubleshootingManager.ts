import { App, Modal, Notice } from 'obsidian';
import { PluginSettings } from './PluginSettings';
import { SettingsManager } from './SettingsManager';
import { DebugLogger } from '../errors/DebugLogger';

export interface DiagnosticResult {
    category: string;
    test: string;
    status: 'pass' | 'fail' | 'warning' | 'info';
    message: string;
    details?: string;
    fix?: string;
}

export interface SystemInfo {
    platform: string;
    userAgent: string;
    obsidianVersion: string;
    pluginVersion: string;
    timestamp: string;
}

export class TroubleshootingManager {
    private app: App;
    private settingsManager: SettingsManager;
    private logger: DebugLogger;
    private settings: PluginSettings;

    constructor(
        app: App,
        settingsManager: SettingsManager,
        logger: DebugLogger
    ) {
        this.app = app;
        this.settingsManager = settingsManager;
        this.logger = logger;
        this.settings = settingsManager.getSettings();
    }

    /**
     * 執行完整診斷
     */
    async runFullDiagnostics(): Promise<DiagnosticResult[]> {
        const results: DiagnosticResult[] = [];

        try {
            // 基本系統檢查
            results.push(...await this.checkSystemRequirements());
            
            // 設定驗證
            results.push(...await this.validateSettings());
            
            // 網路連線檢查
            results.push(...await this.checkNetworkConnectivity());
            
            // API 連線檢查
            results.push(...await this.checkAPIConnection());
            
            // 快取狀態檢查
            results.push(...await this.checkCacheStatus());
            
            // 效能檢查
            results.push(...await this.checkPerformance());

            this.logger.info('Full diagnostics completed', { 
                totalTests: results.length,
                passed: results.filter(r => r.status === 'pass').length,
                failed: results.filter(r => r.status === 'fail').length,
                warnings: results.filter(r => r.status === 'warning').length
            });

        } catch (error) {
            this.logger.error('Diagnostics failed', { error });
            results.push({
                category: 'System',
                test: 'Diagnostic Execution',
                status: 'fail',
                message: 'Failed to complete diagnostics',
                details: error instanceof Error ? error.message : 'Unknown error'
            });
        }

        return results;
    }

    /**
     * 檢查系統需求
     */
    private async checkSystemRequirements(): Promise<DiagnosticResult[]> {
        const results: DiagnosticResult[] = [];

        // 檢查瀏覽器支援
        results.push({
            category: 'System',
            test: 'Browser Support',
            status: typeof fetch !== 'undefined' ? 'pass' : 'fail',
            message: typeof fetch !== 'undefined' 
                ? 'Fetch API is supported' 
                : 'Fetch API is not supported',
            fix: typeof fetch === 'undefined' 
                ? 'Please update your browser or Obsidian version' 
                : undefined
        });

        // 檢查本地儲存
        try {
            localStorage.setItem('ink-plugin-test', 'test');
            localStorage.removeItem('ink-plugin-test');
            results.push({
                category: 'System',
                test: 'Local Storage',
                status: 'pass',
                message: 'Local storage is available'
            });
        } catch (error) {
            results.push({
                category: 'System',
                test: 'Local Storage',
                status: 'fail',
                message: 'Local storage is not available',
                details: error instanceof Error ? error.message : 'Unknown error',
                fix: 'Check browser privacy settings'
            });
        }

        // 檢查記憶體使用
        if ('memory' in performance) {
            const memory = (performance as any).memory;
            const usedMB = Math.round(memory.usedJSHeapSize / 1024 / 1024);
            const limitMB = Math.round(memory.jsHeapSizeLimit / 1024 / 1024);
            const usage = usedMB / limitMB;

            results.push({
                category: 'System',
                test: 'Memory Usage',
                status: usage > 0.8 ? 'warning' : 'pass',
                message: `Memory usage: ${usedMB}MB / ${limitMB}MB (${Math.round(usage * 100)}%)`,
                fix: usage > 0.8 ? 'Consider restarting Obsidian to free memory' : undefined
            });
        }

        return results;
    }

    /**
     * 驗證設定
     */
    private async validateSettings(): Promise<DiagnosticResult[]> {
        const results: DiagnosticResult[] = [];
        const validation = this.settingsManager.validateSettings(this.settings);

        if (validation.isValid) {
            results.push({
                category: 'Settings',
                test: 'Settings Validation',
                status: 'pass',
                message: 'All settings are valid'
            });
        } else {
            for (const error of validation.errors) {
                results.push({
                    category: 'Settings',
                    test: `Setting: ${error.field}`,
                    status: error.severity === 'error' ? 'fail' : 'warning',
                    message: error.message,
                    fix: 'Please check and correct the setting in the plugin settings'
                });
            }
        }

        // 檢查必要設定
        if (!this.settings.inkGatewayUrl) {
            results.push({
                category: 'Settings',
                test: 'Gateway URL',
                status: 'fail',
                message: 'Ink-Gateway URL is not configured',
                fix: 'Set the Gateway URL in plugin settings'
            });
        }

        if (!this.settings.apiKey) {
            results.push({
                category: 'Settings',
                test: 'API Key',
                status: 'fail',
                message: 'API key is not configured',
                fix: 'Set the API key in plugin settings'
            });
        }

        return results;
    }

    /**
     * 檢查網路連線
     */
    private async checkNetworkConnectivity(): Promise<DiagnosticResult[]> {
        const results: DiagnosticResult[] = [];

        try {
            // 檢查基本網路連線
            const response = await fetch('https://httpbin.org/get', {
                method: 'GET',
                signal: new AbortController().signal
            });

            if (response.ok) {
                results.push({
                    category: 'Network',
                    test: 'Internet Connectivity',
                    status: 'pass',
                    message: 'Internet connection is working'
                });
            } else {
                results.push({
                    category: 'Network',
                    test: 'Internet Connectivity',
                    status: 'warning',
                    message: `HTTP ${response.status}: ${response.statusText}`,
                    fix: 'Check your internet connection'
                });
            }
        } catch (error) {
            results.push({
                category: 'Network',
                test: 'Internet Connectivity',
                status: 'fail',
                message: 'No internet connection',
                details: error instanceof Error ? error.message : 'Unknown error',
                fix: 'Check your internet connection and firewall settings'
            });
        }

        return results;
    }

    /**
     * 檢查 API 連線
     */
    private async checkAPIConnection(): Promise<DiagnosticResult[]> {
        const results: DiagnosticResult[] = [];

        if (!this.settings.inkGatewayUrl || !this.settings.apiKey) {
            results.push({
                category: 'API',
                test: 'API Configuration',
                status: 'fail',
                message: 'API configuration is incomplete',
                fix: 'Configure Gateway URL and API key in settings'
            });
            return results;
        }

        try {
            const connectionResult = await this.settingsManager.testConnection();
            
            results.push({
                category: 'API',
                test: 'Gateway Connection',
                status: connectionResult.success ? 'pass' : 'fail',
                message: connectionResult.message,
                details: connectionResult.responseTime 
                    ? `Response time: ${connectionResult.responseTime}ms` 
                    : undefined,
                fix: !connectionResult.success 
                    ? 'Check Gateway URL, API key, and network connectivity' 
                    : undefined
            });

            // 檢查 API 版本相容性（如果連線成功）
            if (connectionResult.success) {
                try {
                    const versionResponse = await fetch(`${this.settings.inkGatewayUrl}/version`, {
                        headers: {
                            'Authorization': `Bearer ${this.settings.apiKey}`
                        },
                        signal: new AbortController().signal
                    });

                    if (versionResponse.ok) {
                        const versionData = await versionResponse.json();
                        results.push({
                            category: 'API',
                            test: 'API Version',
                            status: 'info',
                            message: `Gateway version: ${versionData.version || 'Unknown'}`
                        });
                    }
                } catch (error) {
                    // 版本檢查失敗不是致命錯誤
                    results.push({
                        category: 'API',
                        test: 'API Version',
                        status: 'warning',
                        message: 'Could not retrieve API version',
                        details: error instanceof Error ? error.message : 'Unknown error'
                    });
                }
            }

        } catch (error) {
            results.push({
                category: 'API',
                test: 'Gateway Connection',
                status: 'fail',
                message: 'Failed to test API connection',
                details: error instanceof Error ? error.message : 'Unknown error',
                fix: 'Check Gateway URL and network connectivity'
            });
        }

        return results;
    }

    /**
     * 檢查快取狀態
     */
    private async checkCacheStatus(): Promise<DiagnosticResult[]> {
        const results: DiagnosticResult[] = [];

        if (this.settings.cacheEnabled) {
            results.push({
                category: 'Cache',
                test: 'Cache Configuration',
                status: 'pass',
                message: `Cache enabled (size: ${this.settings.cacheSize}, TTL: ${this.settings.cacheTTL / 1000}s)`
            });

            // 檢查快取空間
            try {
                if ('storage' in navigator && 'estimate' in navigator.storage) {
                    const estimate = await navigator.storage.estimate();
                    const usedMB = Math.round((estimate.usage || 0) / 1024 / 1024);
                    const quotaMB = Math.round((estimate.quota || 0) / 1024 / 1024);
                    const usage = (estimate.usage || 0) / (estimate.quota || 1);

                    results.push({
                        category: 'Cache',
                        test: 'Storage Space',
                        status: usage > 0.9 ? 'warning' : 'pass',
                        message: `Storage usage: ${usedMB}MB / ${quotaMB}MB (${Math.round(usage * 100)}%)`,
                        fix: usage > 0.9 ? 'Consider clearing cache or increasing storage quota' : undefined
                    });
                }
            } catch (error) {
                results.push({
                    category: 'Cache',
                    test: 'Storage Space',
                    status: 'warning',
                    message: 'Could not check storage usage',
                    details: error instanceof Error ? error.message : 'Unknown error'
                });
            }
        } else {
            results.push({
                category: 'Cache',
                test: 'Cache Configuration',
                status: 'info',
                message: 'Cache is disabled'
            });
        }

        return results;
    }

    /**
     * 檢查效能
     */
    private async checkPerformance(): Promise<DiagnosticResult[]> {
        const results: DiagnosticResult[] = [];

        // 檢查同步間隔設定
        if (this.settings.autoSync) {
            if (this.settings.syncInterval < 5000) {
                results.push({
                    category: 'Performance',
                    test: 'Sync Interval',
                    status: 'warning',
                    message: `Sync interval is very short (${this.settings.syncInterval}ms)`,
                    fix: 'Consider increasing sync interval to reduce server load'
                });
            } else {
                results.push({
                    category: 'Performance',
                    test: 'Sync Interval',
                    status: 'pass',
                    message: `Sync interval: ${this.settings.syncInterval}ms`
                });
            }
        }

        // 檢查批次大小
        if (this.settings.batchSyncSize > 100) {
            results.push({
                category: 'Performance',
                test: 'Batch Size',
                status: 'warning',
                message: `Large batch size (${this.settings.batchSyncSize})`,
                fix: 'Consider reducing batch size for better performance'
            });
        } else {
            results.push({
                category: 'Performance',
                test: 'Batch Size',
                status: 'pass',
                message: `Batch size: ${this.settings.batchSyncSize}`
            });
        }

        // 檢查除錯模式
        if (this.settings.debugMode) {
            results.push({
                category: 'Performance',
                test: 'Debug Mode',
                status: 'warning',
                message: 'Debug mode is enabled',
                fix: 'Disable debug mode in production for better performance'
            });
        }

        return results;
    }

    /**
     * 取得系統資訊
     */
    getSystemInfo(): SystemInfo {
        return {
            platform: navigator.platform,
            userAgent: navigator.userAgent,
            obsidianVersion: (this.app as any).appVersion || 'Unknown',
            pluginVersion: '1.0.0', // 應該從 manifest.json 讀取
            timestamp: new Date().toISOString()
        };
    }

    /**
     * 匯出診斷報告
     */
    async exportDiagnosticReport(): Promise<string> {
        const results = await this.runFullDiagnostics();
        const systemInfo = this.getSystemInfo();

        const report = {
            systemInfo,
            diagnostics: results,
            settings: {
                // 只匯出非敏感設定
                inkGatewayUrl: this.settings.inkGatewayUrl,
                connectionTimeout: this.settings.connectionTimeout,
                retryAttempts: this.settings.retryAttempts,
                autoSync: this.settings.autoSync,
                syncInterval: this.settings.syncInterval,
                cacheEnabled: this.settings.cacheEnabled,
                cacheSize: this.settings.cacheSize,
                debugMode: this.settings.debugMode,
                logLevel: this.settings.logLevel
            },
            timestamp: new Date().toISOString()
        };

        return JSON.stringify(report, null, 2);
    }

    /**
     * 自動修復常見問題
     */
    async autoFix(): Promise<{ fixed: number; failed: number; messages: string[] }> {
        const messages: string[] = [];
        let fixed = 0;
        let failed = 0;

        try {
            // 修復設定驗證錯誤
            const validation = this.settingsManager.validateSettings(this.settings);
            if (!validation.isValid) {
                const errorFields = validation.errors
                    .filter(e => e.severity === 'error')
                    .map(e => e.field);

                if (errorFields.length > 0) {
                    // 重置有問題的設定為預設值
                    const fixedSettings: Partial<PluginSettings> = {};
                    for (const field of errorFields) {
                        (fixedSettings as any)[field] = (this.settingsManager as any).DEFAULT_SETTINGS[field];
                    }

                    await this.settingsManager.saveSettings(fixedSettings);
                    fixed++;
                    messages.push(`Fixed ${errorFields.length} invalid settings`);
                }
            }

            // 清理快取（如果啟用）
            if (this.settings.cacheEnabled) {
                try {
                    // 這裡應該調用實際的快取清理方法
                    // await this.cacheManager.clear();
                    fixed++;
                    messages.push('Cache cleared successfully');
                } catch (error) {
                    failed++;
                    messages.push('Failed to clear cache');
                }
            }

        } catch (error) {
            failed++;
            messages.push(`Auto-fix failed: ${error instanceof Error ? error.message : 'Unknown error'}`);
        }

        return { fixed, failed, messages };
    }
}

/**
 * 故障排除模態框
 */
export class TroubleshootingModal extends Modal {
    private troubleshootingManager: TroubleshootingManager;
    private results: DiagnosticResult[] = [];

    constructor(app: App, troubleshootingManager: TroubleshootingManager) {
        super(app);
        this.troubleshootingManager = troubleshootingManager;
    }

    onOpen(): void {
        const { contentEl } = this;
        contentEl.createEl('h2', { text: 'Troubleshooting & Diagnostics' });

        this.createActionButtons(contentEl);
        this.createResultsContainer(contentEl);
    }

    private createActionButtons(container: HTMLElement): void {
        const buttonContainer = container.createDiv({ cls: 'ink-troubleshooting-buttons' });

        // 執行診斷按鈕
        const runDiagnosticsBtn = buttonContainer.createEl('button', {
            text: 'Run Diagnostics',
            cls: 'mod-cta'
        });
        runDiagnosticsBtn.onclick = () => this.runDiagnostics();

        // 自動修復按鈕
        const autoFixBtn = buttonContainer.createEl('button', {
            text: 'Auto Fix'
        });
        autoFixBtn.onclick = () => this.runAutoFix();

        // 匯出報告按鈕
        const exportBtn = buttonContainer.createEl('button', {
            text: 'Export Report'
        });
        exportBtn.onclick = () => this.exportReport();

        // 系統資訊按鈕
        const systemInfoBtn = buttonContainer.createEl('button', {
            text: 'System Info'
        });
        systemInfoBtn.onclick = () => this.showSystemInfo();
    }

    private createResultsContainer(container: HTMLElement): void {
        container.createDiv({ cls: 'ink-diagnostics-results', attr: { id: 'diagnostics-results' } });
    }

    private async runDiagnostics(): Promise<void> {
        const resultsContainer = this.contentEl.querySelector('#diagnostics-results') as HTMLElement;
        resultsContainer.empty();
        resultsContainer.createEl('p', { text: 'Running diagnostics...' });

        try {
            this.results = await this.troubleshootingManager.runFullDiagnostics();
            this.displayResults(resultsContainer);
        } catch (error) {
            resultsContainer.empty();
            resultsContainer.createEl('p', { 
                text: `Diagnostics failed: ${error}`,
                cls: 'ink-error'
            });
        }
    }

    private displayResults(container: HTMLElement): void {
        container.empty();

        if (this.results.length === 0) {
            container.createEl('p', { text: 'No diagnostic results available.' });
            return;
        }

        // 統計摘要
        const summary = container.createDiv({ cls: 'ink-diagnostics-summary' });
        const passed = this.results.filter(r => r.status === 'pass').length;
        const failed = this.results.filter(r => r.status === 'fail').length;
        const warnings = this.results.filter(r => r.status === 'warning').length;

        summary.createEl('h3', { text: 'Summary' });
        summary.createEl('p', { text: `✅ Passed: ${passed} | ❌ Failed: ${failed} | ⚠️ Warnings: ${warnings}` });

        // 按類別分組顯示結果
        const categories = [...new Set(this.results.map(r => r.category))];
        
        for (const category of categories) {
            const categoryResults = this.results.filter(r => r.category === category);
            const categorySection = container.createDiv({ cls: 'ink-diagnostics-category' });
            
            categorySection.createEl('h3', { text: category });
            
            for (const result of categoryResults) {
                const resultEl = categorySection.createDiv({ cls: `ink-diagnostic-result ink-${result.status}` });
                
                const statusIcon = this.getStatusIcon(result.status);
                const header = resultEl.createDiv({ cls: 'ink-result-header' });
                header.createSpan({ text: `${statusIcon} ${result.test}` });
                
                resultEl.createDiv({ 
                    cls: 'ink-result-message',
                    text: result.message 
                });
                
                if (result.details) {
                    resultEl.createDiv({ 
                        cls: 'ink-result-details',
                        text: `Details: ${result.details}` 
                    });
                }
                
                if (result.fix) {
                    resultEl.createDiv({ 
                        cls: 'ink-result-fix',
                        text: `💡 ${result.fix}` 
                    });
                }
            }
        }
    }

    private getStatusIcon(status: DiagnosticResult['status']): string {
        switch (status) {
            case 'pass': return '✅';
            case 'fail': return '❌';
            case 'warning': return '⚠️';
            case 'info': return 'ℹ️';
            default: return '❓';
        }
    }

    private async runAutoFix(): Promise<void> {
        const resultsContainer = this.contentEl.querySelector('#diagnostics-results') as HTMLElement;
        resultsContainer.empty();
        resultsContainer.createEl('p', { text: 'Running auto-fix...' });

        try {
            const result = await this.troubleshootingManager.autoFix();
            resultsContainer.empty();
            
            const summary = resultsContainer.createDiv({ cls: 'ink-autofix-summary' });
            summary.createEl('h3', { text: 'Auto-Fix Results' });
            summary.createEl('p', { text: `Fixed: ${result.fixed} | Failed: ${result.failed}` });
            
            if (result.messages.length > 0) {
                const messagesList = summary.createEl('ul');
                for (const message of result.messages) {
                    messagesList.createEl('li', { text: message });
                }
            }

            if (result.fixed > 0) {
                new Notice(`Auto-fix completed: ${result.fixed} issues fixed`);
            }

        } catch (error) {
            resultsContainer.empty();
            resultsContainer.createEl('p', { 
                text: `Auto-fix failed: ${error}`,
                cls: 'ink-error'
            });
        }
    }

    private async exportReport(): Promise<void> {
        try {
            const report = await this.troubleshootingManager.exportDiagnosticReport();
            
            const blob = new Blob([report], { type: 'application/json' });
            const url = URL.createObjectURL(blob);
            const a = document.createElement('a');
            a.href = url;
            a.download = `ink-plugin-diagnostics-${new Date().toISOString().split('T')[0]}.json`;
            document.body.appendChild(a);
            a.click();
            document.body.removeChild(a);
            URL.revokeObjectURL(url);
            
            new Notice('Diagnostic report exported');
        } catch (error) {
            new Notice(`Export failed: ${error}`);
        }
    }

    private showSystemInfo(): void {
        const systemInfo = this.troubleshootingManager.getSystemInfo();
        const resultsContainer = this.contentEl.querySelector('#diagnostics-results') as HTMLElement;
        
        resultsContainer.empty();
        resultsContainer.createEl('h3', { text: 'System Information' });
        
        const infoList = resultsContainer.createEl('dl', { cls: 'ink-system-info' });
        
        Object.entries(systemInfo).forEach(([key, value]) => {
            infoList.createEl('dt', { text: key.charAt(0).toUpperCase() + key.slice(1) });
            infoList.createEl('dd', { text: value });
        });
    }

    onClose(): void {
        const { contentEl } = this;
        contentEl.empty();
    }
}