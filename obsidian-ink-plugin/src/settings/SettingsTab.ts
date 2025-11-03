import { App, PluginSettingTab, Setting, Notice, Modal } from 'obsidian';
import { PluginSettings, SettingsExportData } from './PluginSettings';
import { SettingsManager } from './SettingsManager';
import { DebugLogger } from '../errors/DebugLogger';

export class InkPluginSettingsTab extends PluginSettingTab {
    private settingsManager: SettingsManager;
    private logger: DebugLogger;
    private settings: PluginSettings;
    private folderLinkSetting: Setting | null = null;

    constructor(
        app: App, 
        plugin: any, 
        settingsManager: SettingsManager,
        logger: DebugLogger
    ) {
        super(app, plugin);
        this.settingsManager = settingsManager;
        this.logger = logger;
        this.settings = settingsManager.getSettings();
    }

    display(): void {
        const { containerEl } = this;
        containerEl.empty();

        containerEl.createEl('h2', { text: 'Obsidian Ink Plugin Settings' });

        // Ink-Gateway 連線設定
        this.addConnectionSettings(containerEl);
        
        // 儲存設定
        this.addStorageSettings(containerEl);
        
        // 同步設定
        this.addSyncSettings(containerEl);
        
        // 快取設定
        this.addCacheSettings(containerEl);
        
        // AI 設定
        this.addAISettings(containerEl);
        
        // 搜尋設定
        this.addSearchSettings(containerEl);
        
        // 模板設定
        this.addTemplateSettings(containerEl);
        
        // UI 設定
        this.addUISettings(containerEl);
        
        // 進階設定
        this.addAdvancedSettings(containerEl);
        
        // 除錯設定
        this.addDebugSettings(containerEl);
        
        // 匯入/匯出和重置
        this.addManagementSettings(containerEl);
    }

    private addConnectionSettings(containerEl: HTMLElement): void {
        containerEl.createEl('h3', { text: 'Ink-Gateway Connection' });

        new Setting(containerEl)
            .setName('Gateway URL')
            .setDesc('The URL of your Ink-Gateway server')
            .addText(text => text
                .setPlaceholder('http://localhost:8081')
                .setValue(this.settings.inkGatewayUrl)
                .onChange(async (value) => {
                    await this.settingsManager.updateSetting('inkGatewayUrl', value);
                    this.settings = this.settingsManager.getSettings();
                }));

        new Setting(containerEl)
            .setName('API Key')
            .setDesc('Your Ink-Gateway API key')
            .addText(text => {
                text.inputEl.type = 'password';
                text.setPlaceholder('Enter your API key')
                    .setValue(this.settings.apiKey)
                    .onChange(async (value) => {
                        await this.settingsManager.updateSetting('apiKey', value);
                        this.settings = this.settingsManager.getSettings();
                    });
            });

        new Setting(containerEl)
            .setName('Connection Timeout')
            .setDesc('Timeout for API requests (milliseconds)')
            .addSlider(slider => slider
                .setLimits(5000, 60000, 5000)
                .setValue(this.settings.connectionTimeout)
                .setDynamicTooltip()
                .onChange(async (value) => {
                    await this.settingsManager.updateSetting('connectionTimeout', value);
                    this.settings = this.settingsManager.getSettings();
                }));

        new Setting(containerEl)
            .setName('Retry Attempts')
            .setDesc('Number of retry attempts for failed requests')
            .addSlider(slider => slider
                .setLimits(0, 10, 1)
                .setValue(this.settings.retryAttempts)
                .setDynamicTooltip()
                .onChange(async (value) => {
                    await this.settingsManager.updateSetting('retryAttempts', value);
                    this.settings = this.settingsManager.getSettings();
                }));

        new Setting(containerEl)
            .setName('Test Connection')
            .setDesc('Test the connection to Ink-Gateway')
            .addButton(button => button
                .setButtonText('Test')
                .setCta()
                .onClick(async () => {
                    button.setButtonText('Testing...');
                    button.setDisabled(true);
                    
                    try {
                        const result = await this.settingsManager.testConnection();
                        if (result.success) {
                            new Notice(`✅ ${result.message} (${result.responseTime}ms)`);
                        } else {
                            new Notice(`❌ ${result.message}`);
                        }
                    } catch (error) {
                        new Notice(`❌ Connection test failed: ${error}`);
                    } finally {
                        button.setButtonText('Test');
                        button.setDisabled(false);
                    }
                }));
    }

    private addStorageSettings(containerEl: HTMLElement): void {
        containerEl.createEl('h3', { text: 'Storage Configuration' });

        new Setting(containerEl)
            .setName('Storage Provider')
            .setDesc('Choose where to store uploaded images')
            .addDropdown(dropdown => dropdown
                .addOption('google_drive', 'Google Drive')
                .addOption('local', 'Local Storage')
                .addOption('both', 'Both (Google Drive + Local)')
                .setValue(this.settings.storageProvider)
                .onChange(async (value: 'google_drive' | 'local' | 'both') => {
                    await this.settingsManager.updateSetting('storageProvider', value);
                    this.settings = this.settingsManager.getSettings();
                }));

        new Setting(containerEl)
            .setName('Google Drive Folder ID')
            .setDesc('The Google Drive folder ID where images will be stored')
            .addText(text => text
                .setPlaceholder('1Q5rWspN-wqjqnfV0HhfngqhMFVy4QRvl')
                .setValue(this.settings.googleDriveFolderId)
                .onChange(async (value) => {
                    await this.settingsManager.updateSetting('googleDriveFolderId', value);
                    this.settings = this.settingsManager.getSettings();
                    // 更新 Google Drive 連結
                    this.updateGoogleDriveLink();
                }));

        new Setting(containerEl)
            .setName('Local Storage Path')
            .setDesc('Local directory path for image storage (fallback)')
            .addText(text => text
                .setPlaceholder('./uploads')
                .setValue(this.settings.localStoragePath)
                .onChange(async (value) => {
                    await this.settingsManager.updateSetting('localStoragePath', value);
                    this.settings = this.settingsManager.getSettings();
                }));

        // 添加 Google Drive 資料夾連結
        this.folderLinkSetting = new Setting(containerEl)
            .setName('Google Drive Folder')
            .setDesc('Click to open your Google Drive storage folder');
        
        this.updateGoogleDriveLink();
    }

    private addSyncSettings(containerEl: HTMLElement): void {
        containerEl.createEl('h3', { text: 'Synchronization' });

        new Setting(containerEl)
            .setName('Auto Sync')
            .setDesc('Automatically sync content changes')
            .addToggle(toggle => toggle
                .setValue(this.settings.autoSync)
                .onChange(async (value) => {
                    await this.settingsManager.updateSetting('autoSync', value);
                    this.settings = this.settingsManager.getSettings();
                }));

        new Setting(containerEl)
            .setName('Sync on Enter')
            .setDesc('Trigger sync when pressing Enter')
            .addToggle(toggle => toggle
                .setValue(this.settings.syncOnEnter)
                .onChange(async (value) => {
                    await this.settingsManager.updateSetting('syncOnEnter', value);
                    this.settings = this.settingsManager.getSettings();
                }));

        new Setting(containerEl)
            .setName('Sync Interval')
            .setDesc('Interval between automatic syncs (milliseconds)')
            .addSlider(slider => slider
                .setLimits(1000, 60000, 1000)
                .setValue(this.settings.syncInterval)
                .setDynamicTooltip()
                .onChange(async (value) => {
                    await this.settingsManager.updateSetting('syncInterval', value);
                    this.settings = this.settingsManager.getSettings();
                }));

        new Setting(containerEl)
            .setName('Batch Sync Size')
            .setDesc('Number of chunks to sync in one batch')
            .addSlider(slider => slider
                .setLimits(10, 200, 10)
                .setValue(this.settings.batchSyncSize)
                .setDynamicTooltip()
                .onChange(async (value) => {
                    await this.settingsManager.updateSetting('batchSyncSize', value);
                    this.settings = this.settingsManager.getSettings();
                }));
    }

    private addCacheSettings(containerEl: HTMLElement): void {
        containerEl.createEl('h3', { text: 'Cache Settings' });

        new Setting(containerEl)
            .setName('Enable Cache')
            .setDesc('Enable local caching for better performance')
            .addToggle(toggle => toggle
                .setValue(this.settings.cacheEnabled)
                .onChange(async (value) => {
                    await this.settingsManager.updateSetting('cacheEnabled', value);
                    this.settings = this.settingsManager.getSettings();
                }));

        new Setting(containerEl)
            .setName('Cache Size')
            .setDesc('Maximum number of items to cache')
            .addSlider(slider => slider
                .setLimits(10, 1000, 10)
                .setValue(this.settings.cacheSize)
                .setDynamicTooltip()
                .onChange(async (value) => {
                    await this.settingsManager.updateSetting('cacheSize', value);
                    this.settings = this.settingsManager.getSettings();
                }));

        new Setting(containerEl)
            .setName('Cache TTL')
            .setDesc('Cache time-to-live in minutes')
            .addSlider(slider => slider
                .setLimits(1, 60, 1)
                .setValue(this.settings.cacheTTL / 60000)
                .setDynamicTooltip()
                .onChange(async (value) => {
                    await this.settingsManager.updateSetting('cacheTTL', value * 60000);
                    this.settings = this.settingsManager.getSettings();
                }));

        new Setting(containerEl)
            .setName('Search Cache')
            .setDesc('Enable caching for search results')
            .addToggle(toggle => toggle
                .setValue(this.settings.searchCacheEnabled)
                .onChange(async (value) => {
                    await this.settingsManager.updateSetting('searchCacheEnabled', value);
                    this.settings = this.settingsManager.getSettings();
                }));
    }

    private addAISettings(containerEl: HTMLElement): void {
        containerEl.createEl('h3', { text: 'AI Features' });

        new Setting(containerEl)
            .setName('Enable AI Chat')
            .setDesc('Enable AI chat functionality')
            .addToggle(toggle => toggle
                .setValue(this.settings.aiChatEnabled)
                .onChange(async (value) => {
                    await this.settingsManager.updateSetting('aiChatEnabled', value);
                    this.settings = this.settingsManager.getSettings();
                }));

        new Setting(containerEl)
            .setName('Auto Process Content')
            .setDesc('Automatically process content with AI')
            .addToggle(toggle => toggle
                .setValue(this.settings.aiAutoProcess)
                .onChange(async (value) => {
                    await this.settingsManager.updateSetting('aiAutoProcess', value);
                    this.settings = this.settingsManager.getSettings();
                }));

        new Setting(containerEl)
            .setName('AI Context Size')
            .setDesc('Number of previous messages to include in AI context')
            .addSlider(slider => slider
                .setLimits(1, 50, 1)
                .setValue(this.settings.aiContextSize)
                .setDynamicTooltip()
                .onChange(async (value) => {
                    await this.settingsManager.updateSetting('aiContextSize', value);
                    this.settings = this.settingsManager.getSettings();
                }));
    }

    private addSearchSettings(containerEl: HTMLElement): void {
        containerEl.createEl('h3', { text: 'Search Settings' });

        new Setting(containerEl)
            .setName('Enable Semantic Search')
            .setDesc('Enable semantic search functionality')
            .addToggle(toggle => toggle
                .setValue(this.settings.semanticSearchEnabled)
                .onChange(async (value) => {
                    await this.settingsManager.updateSetting('semanticSearchEnabled', value);
                    this.settings = this.settingsManager.getSettings();
                }));

        new Setting(containerEl)
            .setName('Search Result Limit')
            .setDesc('Maximum number of search results to display')
            .addSlider(slider => slider
                .setLimits(10, 500, 10)
                .setValue(this.settings.searchResultLimit)
                .setDynamicTooltip()
                .onChange(async (value) => {
                    await this.settingsManager.updateSetting('searchResultLimit', value);
                    this.settings = this.settingsManager.getSettings();
                }));

        new Setting(containerEl)
            .setName('Highlight Search Results')
            .setDesc('Highlight matching text in search results')
            .addToggle(toggle => toggle
                .setValue(this.settings.searchHighlightEnabled)
                .onChange(async (value) => {
                    await this.settingsManager.updateSetting('searchHighlightEnabled', value);
                    this.settings = this.settingsManager.getSettings();
                }));
    }

    private addTemplateSettings(containerEl: HTMLElement): void {
        containerEl.createEl('h3', { text: 'Template Settings' });

        new Setting(containerEl)
            .setName('Auto Apply Templates')
            .setDesc('Automatically apply templates when creating new content')
            .addToggle(toggle => toggle
                .setValue(this.settings.templateAutoApply)
                .onChange(async (value) => {
                    await this.settingsManager.updateSetting('templateAutoApply', value);
                    this.settings = this.settingsManager.getSettings();
                }));

        new Setting(containerEl)
            .setName('Template Validation')
            .setDesc('Validate template content before saving')
            .addToggle(toggle => toggle
                .setValue(this.settings.templateValidation)
                .onChange(async (value) => {
                    await this.settingsManager.updateSetting('templateValidation', value);
                    this.settings = this.settingsManager.getSettings();
                }));
    }

    private addUISettings(containerEl: HTMLElement): void {
        containerEl.createEl('h3', { text: 'User Interface' });

        new Setting(containerEl)
            .setName('Show Status Bar')
            .setDesc('Show plugin status in the status bar')
            .addToggle(toggle => toggle
                .setValue(this.settings.showStatusBar)
                .onChange(async (value) => {
                    await this.settingsManager.updateSetting('showStatusBar', value);
                    this.settings = this.settingsManager.getSettings();
                }));

        new Setting(containerEl)
            .setName('Show Notifications')
            .setDesc('Show notifications for plugin actions')
            .addToggle(toggle => toggle
                .setValue(this.settings.showNotifications)
                .onChange(async (value) => {
                    await this.settingsManager.updateSetting('showNotifications', value);
                    this.settings = this.settingsManager.getSettings();
                }));

        new Setting(containerEl)
            .setName('Notification Duration')
            .setDesc('How long notifications are displayed (seconds)')
            .addSlider(slider => slider
                .setLimits(1, 10, 1)
                .setValue(this.settings.notificationDuration / 1000)
                .setDynamicTooltip()
                .onChange(async (value) => {
                    await this.settingsManager.updateSetting('notificationDuration', value * 1000);
                    this.settings = this.settingsManager.getSettings();
                }));
    }

    private addAdvancedSettings(containerEl: HTMLElement): void {
        containerEl.createEl('h3', { text: 'Advanced Settings' });

        new Setting(containerEl)
            .setName('Offline Mode')
            .setDesc('Enable offline functionality')
            .addToggle(toggle => toggle
                .setValue(this.settings.offlineMode)
                .onChange(async (value) => {
                    await this.settingsManager.updateSetting('offlineMode', value);
                    this.settings = this.settingsManager.getSettings();
                }));

        new Setting(containerEl)
            .setName('Conflict Resolution')
            .setDesc('How to handle sync conflicts')
            .addDropdown(dropdown => dropdown
                .addOption('local', 'Prefer Local Changes')
                .addOption('remote', 'Prefer Remote Changes')
                .addOption('manual', 'Manual Resolution')
                .setValue(this.settings.conflictResolution)
                .onChange(async (value: 'local' | 'remote' | 'manual') => {
                    await this.settingsManager.updateSetting('conflictResolution', value);
                    this.settings = this.settingsManager.getSettings();
                }));

        new Setting(containerEl)
            .setName('Data Export Format')
            .setDesc('Default format for data exports')
            .addDropdown(dropdown => dropdown
                .addOption('json', 'JSON')
                .addOption('csv', 'CSV')
                .addOption('markdown', 'Markdown')
                .setValue(this.settings.dataExportFormat)
                .onChange(async (value: 'json' | 'csv' | 'markdown') => {
                    await this.settingsManager.updateSetting('dataExportFormat', value);
                    this.settings = this.settingsManager.getSettings();
                }));
    }

    private addDebugSettings(containerEl: HTMLElement): void {
        containerEl.createEl('h3', { text: 'Debug & Monitoring' });

        new Setting(containerEl)
            .setName('Debug Mode')
            .setDesc('Enable debug mode for troubleshooting')
            .addToggle(toggle => toggle
                .setValue(this.settings.debugMode)
                .onChange(async (value) => {
                    await this.settingsManager.updateSetting('debugMode', value);
                    this.settings = this.settingsManager.getSettings();
                }));

        new Setting(containerEl)
            .setName('Log Level')
            .setDesc('Minimum log level to record')
            .addDropdown(dropdown => dropdown
                .addOption('error', 'Error')
                .addOption('warn', 'Warning')
                .addOption('info', 'Info')
                .addOption('debug', 'Debug')
                .setValue(this.settings.logLevel)
                .onChange(async (value: 'error' | 'warn' | 'info' | 'debug') => {
                    await this.settingsManager.updateSetting('logLevel', value);
                    this.settings = this.settingsManager.getSettings();
                }));

        new Setting(containerEl)
            .setName('Performance Monitoring')
            .setDesc('Enable performance monitoring and metrics')
            .addToggle(toggle => toggle
                .setValue(this.settings.performanceMonitoring)
                .onChange(async (value) => {
                    await this.settingsManager.updateSetting('performanceMonitoring', value);
                    this.settings = this.settingsManager.getSettings();
                }));
    }

    private addManagementSettings(containerEl: HTMLElement): void {
        containerEl.createEl('h3', { text: 'Settings Management' });

        new Setting(containerEl)
            .setName('Export Settings')
            .setDesc('Export current settings to a file')
            .addButton(button => button
                .setButtonText('Export')
                .onClick(() => {
                    const exportData = this.settingsManager.exportSettings();
                    this.downloadSettings(exportData);
                    new Notice('Settings exported successfully');
                }));

        new Setting(containerEl)
            .setName('Import Settings')
            .setDesc('Import settings from a file')
            .addButton(button => button
                .setButtonText('Import')
                .onClick(() => {
                    this.showImportModal();
                }));

        new Setting(containerEl)
            .setName('Reset Settings')
            .setDesc('Reset all settings to default values')
            .addButton(button => button
                .setButtonText('Reset')
                .setWarning()
                .onClick(async () => {
                    const confirmed = await this.showConfirmModal(
                        'Reset Settings',
                        'Are you sure you want to reset all settings to default values? This action cannot be undone.'
                    );
                    
                    if (confirmed) {
                        await this.settingsManager.resetSettings();
                        this.settings = this.settingsManager.getSettings();
                        this.display(); // Refresh the settings display
                        new Notice('Settings reset to defaults');
                    }
                }));
    }

    private downloadSettings(exportData: SettingsExportData): void {
        const blob = new Blob([JSON.stringify(exportData, null, 2)], { 
            type: 'application/json' 
        });
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = `obsidian-ink-settings-${new Date().toISOString().split('T')[0]}.json`;
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
        URL.revokeObjectURL(url);
    }

    private showImportModal(): void {
        new ImportSettingsModal(this.app, this.settingsManager, (success) => {
            if (success) {
                this.settings = this.settingsManager.getSettings();
                this.display(); // Refresh the settings display
                new Notice('Settings imported successfully');
            }
        }).open();
    }

    private updateGoogleDriveLink(): void {
        if (!this.folderLinkSetting) return;
        
        // 清除現有的控制元素
        this.folderLinkSetting.controlEl.empty();
        
        // 創建新的連結
        if (this.settings.googleDriveFolderId && this.settings.googleDriveFolderId.trim()) {
            const link = this.folderLinkSetting.controlEl.createEl('a', {
                text: 'Open Google Drive Folder',
                href: `https://drive.google.com/drive/folders/${this.settings.googleDriveFolderId}`,
                cls: 'mod-cta'
            });
            link.setAttribute('target', '_blank');
        } else {
            this.folderLinkSetting.controlEl.createEl('span', {
                text: 'Enter Google Drive Folder ID above',
                cls: 'setting-item-description'
            });
        }
    }

    private showConfirmModal(title: string, message: string): Promise<boolean> {
        return new Promise((resolve) => {
            new ConfirmModal(this.app, title, message, resolve).open();
        });
    }
}

class ImportSettingsModal extends Modal {
    private settingsManager: SettingsManager;
    private onComplete: (success: boolean) => void;

    constructor(
        app: App, 
        settingsManager: SettingsManager, 
        onComplete: (success: boolean) => void
    ) {
        super(app);
        this.settingsManager = settingsManager;
        this.onComplete = onComplete;
    }

    onOpen(): void {
        const { contentEl } = this;
        contentEl.createEl('h2', { text: 'Import Settings' });

        const fileInput = contentEl.createEl('input', {
            type: 'file',
            attr: { accept: '.json' }
        });

        const buttonContainer = contentEl.createDiv({ cls: 'modal-button-container' });
        
        const importButton = buttonContainer.createEl('button', {
            text: 'Import',
            cls: 'mod-cta'
        });

        const cancelButton = buttonContainer.createEl('button', {
            text: 'Cancel'
        });

        importButton.onclick = async () => {
            const file = fileInput.files?.[0];
            if (!file) {
                new Notice('Please select a file');
                return;
            }

            try {
                const text = await file.text();
                const exportData: SettingsExportData = JSON.parse(text);
                await this.settingsManager.importSettings(exportData);
                this.close();
                this.onComplete(true);
            } catch (error) {
                new Notice(`Import failed: ${error}`);
                this.onComplete(false);
            }
        };

        cancelButton.onclick = () => {
            this.close();
            this.onComplete(false);
        };
    }

    onClose(): void {
        const { contentEl } = this;
        contentEl.empty();
    }
}

class ConfirmModal extends Modal {
    private title: string;
    private message: string;
    private onConfirm: (confirmed: boolean) => void;

    constructor(
        app: App, 
        title: string, 
        message: string, 
        onConfirm: (confirmed: boolean) => void
    ) {
        super(app);
        this.title = title;
        this.message = message;
        this.onConfirm = onConfirm;
    }

    onOpen(): void {
        const { contentEl } = this;
        contentEl.createEl('h2', { text: this.title });
        contentEl.createEl('p', { text: this.message });

        const buttonContainer = contentEl.createDiv({ cls: 'modal-button-container' });
        
        const confirmButton = buttonContainer.createEl('button', {
            text: 'Confirm',
            cls: 'mod-warning'
        });

        const cancelButton = buttonContainer.createEl('button', {
            text: 'Cancel'
        });

        confirmButton.onclick = () => {
            this.close();
            this.onConfirm(true);
        };

        cancelButton.onclick = () => {
            this.close();
            this.onConfirm(false);
        };
    }

    onClose(): void {
        const { contentEl } = this;
        contentEl.empty();
    }
}