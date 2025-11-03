import { Plugin } from 'obsidian';
import { 
    PluginSettings, 
    DEFAULT_SETTINGS, 
    SettingsValidationResult, 
    SettingsValidationError,
    SettingsExportData 
} from './PluginSettings';
import { DebugLogger } from '../errors/DebugLogger';

export class SettingsManager {
    private plugin: Plugin;
    private logger: DebugLogger;
    private settings: PluginSettings;
    private changeListeners: Array<(settings: PluginSettings) => void> = [];

    constructor(plugin: Plugin, logger: DebugLogger) {
        this.plugin = plugin;
        this.logger = logger;
        this.settings = { ...DEFAULT_SETTINGS };
    }

    /**
     * 載入設定
     */
    async loadSettings(): Promise<PluginSettings> {
        try {
            const data = await this.plugin.loadData();
            if (data) {
                this.settings = { ...DEFAULT_SETTINGS, ...data };
                this.logger.debug('SettingsManager', 'loadSettings', 'Settings loaded successfully', { settings: this.settings });
            } else {
                this.settings = { ...DEFAULT_SETTINGS };
                this.logger.info('SettingsManager', 'loadSettings', 'No existing settings found, using defaults');
            }
            
            // 驗證載入的設定
            const validation = this.validateSettings(this.settings);
            if (!validation.isValid) {
                this.logger.warn('SettingsManager', 'loadSettings', 'Settings validation failed', { errors: validation.errors });
                // 修復無效設定
                this.settings = this.repairSettings(this.settings, validation.errors);
                await this.saveSettings();
            }
            
            return this.settings;
        } catch (error) {
            this.logger.error('SettingsManager', 'loadSettings', 'Failed to load settings', error, { error });
            this.settings = { ...DEFAULT_SETTINGS };
            return this.settings;
        }
    }

    /**
     * 儲存設定
     */
    async saveSettings(newSettings?: Partial<PluginSettings>): Promise<void> {
        try {
            if (newSettings) {
                const updatedSettings = { ...this.settings, ...newSettings };
                const validation = this.validateSettings(updatedSettings);
                
                if (!validation.isValid) {
                    const errors = validation.errors.filter(e => e.severity === 'error');
                    if (errors.length > 0) {
                        throw new Error(`Settings validation failed: ${errors.map(e => e.message).join(', ')}`);
                    }
                }
                
                this.settings = updatedSettings;
            }
            
            await this.plugin.saveData(this.settings);
            this.logger.debug('SettingsManager', 'saveSettings', 'Settings saved successfully');
            
            // 通知變更監聽器
            this.notifyChangeListeners();
            
        } catch (error) {
            this.logger.error('Failed to save settings', { error });
            throw error;
        }
    }

    /**
     * 取得目前設定
     */
    getSettings(): PluginSettings {
        return { ...this.settings };
    }

    /**
     * 更新特定設定
     */
    async updateSetting<K extends keyof PluginSettings>(
        key: K, 
        value: PluginSettings[K]
    ): Promise<void> {
        const newSettings = { [key]: value } as Partial<PluginSettings>;
        await this.saveSettings(newSettings);
    }

    /**
     * 重置設定為預設值
     */
    async resetSettings(): Promise<void> {
        this.settings = { ...DEFAULT_SETTINGS };
        await this.plugin.saveData(this.settings);
        this.logger.info('SettingsManager', 'resetToDefaults', 'Settings reset to defaults');
        this.notifyChangeListeners();
    }

    /**
     * 驗證設定
     */
    validateSettings(settings: PluginSettings): SettingsValidationResult {
        const errors: SettingsValidationError[] = [];

        // 驗證 Ink-Gateway URL
        if (!settings.inkGatewayUrl || !this.isValidUrl(settings.inkGatewayUrl)) {
            errors.push({
                field: 'inkGatewayUrl',
                message: 'Invalid Ink-Gateway URL format',
                severity: 'error'
            });
        }

        // 驗證 API 金鑰 (改為警告而不是錯誤，允許空值用於初次設置)
        if (!settings.apiKey || settings.apiKey.trim().length === 0) {
            errors.push({
                field: 'apiKey',
                message: 'API key is recommended for full functionality',
                severity: 'warning'
            });
        }

        // 驗證數值範圍
        if (settings.connectionTimeout < 1000 || settings.connectionTimeout > 300000) {
            errors.push({
                field: 'connectionTimeout',
                message: 'Connection timeout must be between 1-300 seconds',
                severity: 'warning'
            });
        }

        if (settings.retryAttempts < 0 || settings.retryAttempts > 10) {
            errors.push({
                field: 'retryAttempts',
                message: 'Retry attempts must be between 0-10',
                severity: 'warning'
            });
        }

        if (settings.syncInterval < 1000) {
            errors.push({
                field: 'syncInterval',
                message: 'Sync interval must be at least 1 second',
                severity: 'warning'
            });
        }

        if (settings.cacheSize < 10 || settings.cacheSize > 1000) {
            errors.push({
                field: 'cacheSize',
                message: 'Cache size must be between 10-1000',
                severity: 'warning'
            });
        }

        if (settings.searchResultLimit < 1 || settings.searchResultLimit > 500) {
            errors.push({
                field: 'searchResultLimit',
                message: 'Search result limit must be between 1-500',
                severity: 'warning'
            });
        }

        return {
            isValid: errors.filter(e => e.severity === 'error').length === 0,
            errors
        };
    }

    /**
     * 修復無效設定
     */
    private repairSettings(
        settings: PluginSettings, 
        errors: SettingsValidationError[]
    ): PluginSettings {
        const repairedSettings = { ...settings };

        for (const error of errors) {
            if (error.severity === 'error') {
                // 使用預設值修復錯誤設定
                (repairedSettings as any)[error.field] = (DEFAULT_SETTINGS as any)[error.field];
                this.logger.warn(`Repaired setting: ${error.field}`, { 
                    oldValue: (settings as any)[error.field],
                    newValue: (DEFAULT_SETTINGS as any)[error.field]
                });
            }
        }

        return repairedSettings;
    }

    /**
     * 匯出設定
     */
    exportSettings(): SettingsExportData {
        return {
            version: '1.0.0',
            timestamp: new Date().toISOString(),
            settings: this.settings,
            metadata: {
                exportedBy: 'Obsidian Ink Plugin',
                platform: navigator.platform
            }
        };
    }

    /**
     * 匯入設定
     */
    async importSettings(exportData: SettingsExportData): Promise<void> {
        try {
            if (!exportData.settings) {
                throw new Error('Invalid export data: missing settings');
            }

            const validation = this.validateSettings(exportData.settings);
            if (!validation.isValid) {
                const errors = validation.errors.filter(e => e.severity === 'error');
                if (errors.length > 0) {
                    throw new Error(`Invalid settings: ${errors.map(e => e.message).join(', ')}`);
                }
            }

            await this.saveSettings(exportData.settings);
            this.logger.info('Settings imported successfully', { 
                version: exportData.version,
                timestamp: exportData.timestamp 
            });

        } catch (error) {
            this.logger.error('Failed to import settings', { error });
            throw error;
        }
    }

    /**
     * 註冊設定變更監聽器
     */
    onSettingsChange(listener: (settings: PluginSettings) => void): void {
        this.changeListeners.push(listener);
    }

    /**
     * 移除設定變更監聽器
     */
    removeSettingsChangeListener(listener: (settings: PluginSettings) => void): void {
        const index = this.changeListeners.indexOf(listener);
        if (index > -1) {
            this.changeListeners.splice(index, 1);
        }
    }

    /**
     * 通知所有變更監聽器
     */
    private notifyChangeListeners(): void {
        for (const listener of this.changeListeners) {
            try {
                listener(this.settings);
            } catch (error) {
                this.logger.error('Settings change listener error', { error });
            }
        }
    }

    /**
     * 驗證 URL 格式
     */
    private isValidUrl(url: string): boolean {
        try {
            new URL(url);
            return true;
        } catch {
            return false;
        }
    }

    /**
     * 測試連線設定
     */
    async testConnection(): Promise<{ success: boolean; message: string; responseTime?: number }> {
        try {
            const startTime = Date.now();
            const controller = new AbortController();
            const timeoutId = setTimeout(() => controller.abort(), this.settings.connectionTimeout);

            const response = await fetch(`${this.settings.inkGatewayUrl}/api/v1/health`, {
                method: 'GET',
                headers: {
                    'Authorization': `Bearer ${this.settings.apiKey}`,
                    'Content-Type': 'application/json'
                },
                signal: controller.signal
            });

            clearTimeout(timeoutId);
            const responseTime = Date.now() - startTime;

            if (response.ok) {
                return {
                    success: true,
                    message: 'Connection successful',
                    responseTime
                };
            } else {
                return {
                    success: false,
                    message: `Connection failed: ${response.status} ${response.statusText}`
                };
            }

        } catch (error) {
            if (error instanceof Error) {
                if (error.name === 'AbortError') {
                    return {
                        success: false,
                        message: 'Connection timeout'
                    };
                }
                return {
                    success: false,
                    message: `Connection error: ${error.message}`
                };
            }
            return {
                success: false,
                message: 'Unknown connection error'
            };
        }
    }
}