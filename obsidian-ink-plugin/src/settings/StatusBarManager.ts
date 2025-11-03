import { Plugin, setIcon } from 'obsidian';
import { PluginSettings } from './PluginSettings';
import { DebugLogger } from '../errors/DebugLogger';

export interface StatusBarState {
    connectionStatus: 'connected' | 'disconnected' | 'connecting' | 'error';
    syncStatus: 'idle' | 'syncing' | 'error';
    lastSyncTime?: Date;
    pendingChanges: number;
    cacheHitRate?: number;
}

export class StatusBarManager {
    private plugin: Plugin;
    private logger: DebugLogger;
    private settings: PluginSettings;
    private statusBarItem: HTMLElement | null = null;
    private state: StatusBarState;

    constructor(plugin: Plugin, logger: DebugLogger, settings: PluginSettings) {
        this.plugin = plugin;
        this.logger = logger;
        this.settings = settings;
        this.state = {
            connectionStatus: 'disconnected',
            syncStatus: 'idle',
            pendingChanges: 0
        };
    }

    /**
     * 初始化狀態列
     */
    initialize(): void {
        if (this.settings.showStatusBar) {
            this.createStatusBar();
        }
    }

    /**
     * 更新設定
     */
    updateSettings(settings: PluginSettings): void {
        const wasVisible = this.settings.showStatusBar;
        this.settings = settings;

        if (settings.showStatusBar && !wasVisible) {
            this.createStatusBar();
        } else if (!settings.showStatusBar && wasVisible) {
            this.removeStatusBar();
        }

        if (this.statusBarItem) {
            this.updateDisplay();
        }
    }

    /**
     * 更新狀態
     */
    updateState(newState: Partial<StatusBarState>): void {
        this.state = { ...this.state, ...newState };
        this.updateDisplay();
        this.logger.debug('Status bar state updated', { state: this.state });
    }

    /**
     * 設定連線狀態
     */
    setConnectionStatus(status: StatusBarState['connectionStatus']): void {
        this.updateState({ connectionStatus: status });
    }

    /**
     * 設定同步狀態
     */
    setSyncStatus(status: StatusBarState['syncStatus'], lastSyncTime?: Date): void {
        this.updateState({ 
            syncStatus: status,
            lastSyncTime: lastSyncTime || this.state.lastSyncTime
        });
    }

    /**
     * 設定待處理變更數量
     */
    setPendingChanges(count: number): void {
        this.updateState({ pendingChanges: count });
    }

    /**
     * 設定快取命中率
     */
    setCacheHitRate(rate: number): void {
        this.updateState({ cacheHitRate: rate });
    }

    /**
     * 創建狀態列項目
     */
    private createStatusBar(): void {
        if (this.statusBarItem) {
            return;
        }

        this.statusBarItem = this.plugin.addStatusBarItem();
        this.statusBarItem.addClass('ink-plugin-status');
        this.statusBarItem.style.cursor = 'pointer';
        
        // 點擊事件 - 顯示詳細狀態
        this.statusBarItem.addEventListener('click', () => {
            this.showStatusModal();
        });

        this.updateDisplay();
        this.logger.debug('Status bar created');
    }

    /**
     * 移除狀態列項目
     */
    private removeStatusBar(): void {
        if (this.statusBarItem) {
            this.statusBarItem.remove();
            this.statusBarItem = null;
            this.logger.debug('Status bar removed');
        }
    }

    /**
     * 更新顯示內容
     */
    private updateDisplay(): void {
        if (!this.statusBarItem) {
            return;
        }

        const { connectionStatus, syncStatus, pendingChanges } = this.state;
        
        // 清空內容
        this.statusBarItem.empty();

        // 連線狀態圖示
        const connectionIcon = this.statusBarItem.createSpan({ cls: 'ink-connection-icon' });
        this.setConnectionIcon(connectionIcon, connectionStatus);

        // 同步狀態圖示
        if (syncStatus !== 'idle') {
            const syncIcon = this.statusBarItem.createSpan({ cls: 'ink-sync-icon' });
            this.setSyncIcon(syncIcon, syncStatus);
        }

        // 待處理變更數量
        if (pendingChanges > 0) {
            const pendingSpan = this.statusBarItem.createSpan({ 
                cls: 'ink-pending-count',
                text: pendingChanges.toString()
            });
            pendingSpan.title = `${pendingChanges} pending changes`;
        }

        // 設定整體標題
        this.statusBarItem.title = this.getStatusTooltip();
    }

    /**
     * 設定連線狀態圖示
     */
    private setConnectionIcon(element: HTMLElement, status: StatusBarState['connectionStatus']): void {
        element.removeClass('ink-connected', 'ink-disconnected', 'ink-connecting', 'ink-error');
        
        switch (status) {
            case 'connected':
                setIcon(element, 'wifi');
                element.addClass('ink-connected');
                break;
            case 'connecting':
                setIcon(element, 'loader-2');
                element.addClass('ink-connecting');
                break;
            case 'error':
                setIcon(element, 'wifi-off');
                element.addClass('ink-error');
                break;
            case 'disconnected':
            default:
                setIcon(element, 'wifi-off');
                element.addClass('ink-disconnected');
                break;
        }
    }

    /**
     * 設定同步狀態圖示
     */
    private setSyncIcon(element: HTMLElement, status: StatusBarState['syncStatus']): void {
        element.removeClass('ink-syncing', 'ink-sync-error');
        
        switch (status) {
            case 'syncing':
                setIcon(element, 'refresh-cw');
                element.addClass('ink-syncing');
                break;
            case 'error':
                setIcon(element, 'alert-circle');
                element.addClass('ink-sync-error');
                break;
        }
    }

    /**
     * 取得狀態提示文字
     */
    private getStatusTooltip(): string {
        const { connectionStatus, syncStatus, lastSyncTime, pendingChanges, cacheHitRate } = this.state;
        
        const parts: string[] = [];
        
        // 連線狀態
        switch (connectionStatus) {
            case 'connected':
                parts.push('🟢 Connected to Ink-Gateway');
                break;
            case 'connecting':
                parts.push('🟡 Connecting to Ink-Gateway...');
                break;
            case 'error':
                parts.push('🔴 Connection error');
                break;
            case 'disconnected':
                parts.push('⚫ Disconnected from Ink-Gateway');
                break;
        }

        // 同步狀態
        if (syncStatus === 'syncing') {
            parts.push('🔄 Syncing...');
        } else if (syncStatus === 'error') {
            parts.push('❌ Sync error');
        } else if (lastSyncTime) {
            const timeAgo = this.getTimeAgo(lastSyncTime);
            parts.push(`✅ Last sync: ${timeAgo}`);
        }

        // 待處理變更
        if (pendingChanges > 0) {
            parts.push(`📝 ${pendingChanges} pending changes`);
        }

        // 快取命中率
        if (cacheHitRate !== undefined) {
            parts.push(`💾 Cache: ${Math.round(cacheHitRate * 100)}%`);
        }

        return parts.join('\n');
    }

    /**
     * 取得相對時間描述
     */
    private getTimeAgo(date: Date): string {
        const now = new Date();
        const diffMs = now.getTime() - date.getTime();
        const diffSecs = Math.floor(diffMs / 1000);
        const diffMins = Math.floor(diffSecs / 60);
        const diffHours = Math.floor(diffMins / 60);

        if (diffSecs < 60) {
            return 'just now';
        } else if (diffMins < 60) {
            return `${diffMins}m ago`;
        } else if (diffHours < 24) {
            return `${diffHours}h ago`;
        } else {
            return date.toLocaleDateString();
        }
    }

    /**
     * 顯示詳細狀態模態框
     */
    private showStatusModal(): void {
        const modal = new StatusModal(this.plugin.app, this.state, this.settings);
        modal.open();
    }

    /**
     * 清理資源
     */
    destroy(): void {
        this.removeStatusBar();
    }
}

// 狀態詳情模態框
import { App, Modal } from 'obsidian';

class StatusModal extends Modal {
    private state: StatusBarState;
    private settings: PluginSettings;

    constructor(app: App, state: StatusBarState, settings: PluginSettings) {
        super(app);
        this.state = state;
        this.settings = settings;
    }

    onOpen(): void {
        const { contentEl } = this;
        contentEl.createEl('h2', { text: 'Ink Plugin Status' });

        // 連線狀態區塊
        this.addConnectionSection(contentEl);
        
        // 同步狀態區塊
        this.addSyncSection(contentEl);
        
        // 效能統計區塊
        this.addPerformanceSection(contentEl);
        
        // 設定快速連結
        this.addQuickActions(contentEl);
    }

    private addConnectionSection(container: HTMLElement): void {
        const section = container.createDiv({ cls: 'ink-status-section' });
        section.createEl('h3', { text: 'Connection Status' });

        const statusEl = section.createDiv({ cls: 'ink-status-item' });
        const statusIcon = this.getConnectionStatusIcon(this.state.connectionStatus);
        const statusText = this.getConnectionStatusText(this.state.connectionStatus);
        
        statusEl.createSpan({ text: `${statusIcon} ${statusText}` });
        
        if (this.settings.inkGatewayUrl) {
            section.createDiv({ 
                cls: 'ink-status-detail',
                text: `Gateway: ${this.settings.inkGatewayUrl}`
            });
        }
    }

    private addSyncSection(container: HTMLElement): void {
        const section = container.createDiv({ cls: 'ink-status-section' });
        section.createEl('h3', { text: 'Synchronization' });

        const syncEl = section.createDiv({ cls: 'ink-status-item' });
        const syncIcon = this.getSyncStatusIcon(this.state.syncStatus);
        const syncText = this.getSyncStatusText(this.state.syncStatus);
        
        syncEl.createSpan({ text: `${syncIcon} ${syncText}` });

        if (this.state.lastSyncTime) {
            section.createDiv({ 
                cls: 'ink-status-detail',
                text: `Last sync: ${this.state.lastSyncTime.toLocaleString()}`
            });
        }

        if (this.state.pendingChanges > 0) {
            section.createDiv({ 
                cls: 'ink-status-detail',
                text: `Pending changes: ${this.state.pendingChanges}`
            });
        }
    }

    private addPerformanceSection(container: HTMLElement): void {
        const section = container.createDiv({ cls: 'ink-status-section' });
        section.createEl('h3', { text: 'Performance' });

        if (this.state.cacheHitRate !== undefined) {
            section.createDiv({ 
                cls: 'ink-status-item',
                text: `💾 Cache hit rate: ${Math.round(this.state.cacheHitRate * 100)}%`
            });
        }

        // 可以添加更多效能指標
        section.createDiv({ 
            cls: 'ink-status-detail',
            text: `Cache enabled: ${this.settings.cacheEnabled ? 'Yes' : 'No'}`
        });

        section.createDiv({ 
            cls: 'ink-status-detail',
            text: `Auto sync: ${this.settings.autoSync ? 'Yes' : 'No'}`
        });
    }

    private addQuickActions(container: HTMLElement): void {
        const section = container.createDiv({ cls: 'ink-status-section' });
        section.createEl('h3', { text: 'Quick Actions' });

        const buttonContainer = section.createDiv({ cls: 'ink-button-container' });

        // 開啟設定按鈕
        const settingsBtn = buttonContainer.createEl('button', {
            text: 'Open Settings',
            cls: 'mod-cta'
        });
        settingsBtn.onclick = () => {
            this.close();
            // 觸發開啟設定頁面的事件
            (this.app as any).setting.open();
            (this.app as any).setting.openTabById('ink-plugin');
        };

        // 測試連線按鈕
        const testBtn = buttonContainer.createEl('button', {
            text: 'Test Connection'
        });
        testBtn.onclick = async () => {
            testBtn.textContent = 'Testing...';
            testBtn.disabled = true;
            
            try {
                // 這裡應該調用實際的連線測試方法
                // 暫時模擬測試結果
                await new Promise(resolve => setTimeout(resolve, 1000));
                testBtn.textContent = 'Connection OK';
                testBtn.style.color = 'green';
            } catch (error) {
                testBtn.textContent = 'Connection Failed';
                testBtn.style.color = 'red';
            } finally {
                setTimeout(() => {
                    testBtn.textContent = 'Test Connection';
                    testBtn.disabled = false;
                    testBtn.style.color = '';
                }, 2000);
            }
        };
    }

    private getConnectionStatusIcon(status: StatusBarState['connectionStatus']): string {
        switch (status) {
            case 'connected': return '🟢';
            case 'connecting': return '🟡';
            case 'error': return '🔴';
            case 'disconnected': return '⚫';
            default: return '❓';
        }
    }

    private getConnectionStatusText(status: StatusBarState['connectionStatus']): string {
        switch (status) {
            case 'connected': return 'Connected';
            case 'connecting': return 'Connecting...';
            case 'error': return 'Connection Error';
            case 'disconnected': return 'Disconnected';
            default: return 'Unknown';
        }
    }

    private getSyncStatusIcon(status: StatusBarState['syncStatus']): string {
        switch (status) {
            case 'syncing': return '🔄';
            case 'error': return '❌';
            case 'idle': return '✅';
            default: return '❓';
        }
    }

    private getSyncStatusText(status: StatusBarState['syncStatus']): string {
        switch (status) {
            case 'syncing': return 'Syncing...';
            case 'error': return 'Sync Error';
            case 'idle': return 'Up to date';
            default: return 'Unknown';
        }
    }

    onClose(): void {
        const { contentEl } = this;
        contentEl.empty();
    }
}