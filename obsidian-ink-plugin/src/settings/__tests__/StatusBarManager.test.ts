import { StatusBarManager, StatusBarState } from '../StatusBarManager';
import { DEFAULT_SETTINGS } from '../PluginSettings';
import { DebugLogger } from '../../errors/DebugLogger';

// Mock Obsidian Plugin
const mockPlugin = {
    addStatusBarItem: jest.fn(() => ({
        addClass: jest.fn(),
        style: {},
        addEventListener: jest.fn(),
        empty: jest.fn(),
        createSpan: jest.fn(() => ({
            addClass: jest.fn(),
            removeClass: jest.fn(),
            title: ''
        })),
        remove: jest.fn(),
        title: ''
    }))
};

// Mock DebugLogger
const mockLogger = {
    debug: jest.fn(),
    info: jest.fn(),
    warn: jest.fn(),
    error: jest.fn()
} as unknown as DebugLogger;

describe('StatusBarManager', () => {
    let statusBarManager: StatusBarManager;
    let mockStatusBarItem: any;

    beforeEach(() => {
        jest.clearAllMocks();
        mockStatusBarItem = {
            addClass: jest.fn(),
            style: {},
            addEventListener: jest.fn(),
            empty: jest.fn(),
            createSpan: jest.fn(() => ({
                addClass: jest.fn(),
                removeClass: jest.fn(),
                title: ''
            })),
            remove: jest.fn(),
            title: ''
        };
        mockPlugin.addStatusBarItem.mockReturnValue(mockStatusBarItem);
        
        statusBarManager = new StatusBarManager(
            mockPlugin as any,
            mockLogger,
            DEFAULT_SETTINGS
        );
    });

    describe('initialize', () => {
        it('should create status bar when showStatusBar is true', () => {
            const settings = { ...DEFAULT_SETTINGS, showStatusBar: true };
            statusBarManager = new StatusBarManager(mockPlugin as any, mockLogger, settings);

            statusBarManager.initialize();

            expect(mockPlugin.addStatusBarItem).toHaveBeenCalled();
            expect(mockStatusBarItem.addClass).toHaveBeenCalledWith('ink-plugin-status');
        });

        it('should not create status bar when showStatusBar is false', () => {
            const settings = { ...DEFAULT_SETTINGS, showStatusBar: false };
            statusBarManager = new StatusBarManager(mockPlugin as any, mockLogger, settings);

            statusBarManager.initialize();

            expect(mockPlugin.addStatusBarItem).not.toHaveBeenCalled();
        });
    });

    describe('updateSettings', () => {
        beforeEach(() => {
            statusBarManager.initialize();
        });

        it('should create status bar when showStatusBar changes from false to true', () => {
            const newSettings = { ...DEFAULT_SETTINGS, showStatusBar: true };
            
            statusBarManager.updateSettings(newSettings);

            expect(mockPlugin.addStatusBarItem).toHaveBeenCalled();
        });

        it('should remove status bar when showStatusBar changes from true to false', () => {
            const newSettings = { ...DEFAULT_SETTINGS, showStatusBar: false };
            
            statusBarManager.updateSettings(newSettings);

            expect(mockStatusBarItem.remove).toHaveBeenCalled();
        });

        it('should update display when status bar exists', () => {
            const newSettings = { ...DEFAULT_SETTINGS, showStatusBar: true };
            
            statusBarManager.updateSettings(newSettings);

            expect(mockStatusBarItem.empty).toHaveBeenCalled();
        });
    });

    describe('updateState', () => {
        beforeEach(() => {
            statusBarManager.initialize();
        });

        it('should update connection status', () => {
            statusBarManager.setConnectionStatus('connected');

            expect(mockLogger.debug).toHaveBeenCalledWith(
                'Status bar state updated',
                expect.objectContaining({
                    state: expect.objectContaining({
                        connectionStatus: 'connected'
                    })
                })
            );
        });

        it('should update sync status', () => {
            const syncTime = new Date();
            statusBarManager.setSyncStatus('syncing', syncTime);

            expect(mockLogger.debug).toHaveBeenCalledWith(
                'Status bar state updated',
                expect.objectContaining({
                    state: expect.objectContaining({
                        syncStatus: 'syncing',
                        lastSyncTime: syncTime
                    })
                })
            );
        });

        it('should update pending changes count', () => {
            statusBarManager.setPendingChanges(5);

            expect(mockLogger.debug).toHaveBeenCalledWith(
                'Status bar state updated',
                expect.objectContaining({
                    state: expect.objectContaining({
                        pendingChanges: 5
                    })
                })
            );
        });

        it('should update cache hit rate', () => {
            statusBarManager.setCacheHitRate(0.85);

            expect(mockLogger.debug).toHaveBeenCalledWith(
                'Status bar state updated',
                expect.objectContaining({
                    state: expect.objectContaining({
                        cacheHitRate: 0.85
                    })
                })
            );
        });
    });

    describe('updateState with complex state', () => {
        beforeEach(() => {
            statusBarManager.initialize();
        });

        it('should handle multiple state updates', () => {
            const newState: Partial<StatusBarState> = {
                connectionStatus: 'connected',
                syncStatus: 'syncing',
                pendingChanges: 3,
                cacheHitRate: 0.9
            };

            statusBarManager.updateState(newState);

            expect(mockLogger.debug).toHaveBeenCalledWith(
                'Status bar state updated',
                expect.objectContaining({
                    state: expect.objectContaining(newState)
                })
            );
        });

        it('should preserve existing state when partially updating', () => {
            // 設定初始狀態
            statusBarManager.updateState({
                connectionStatus: 'connected',
                pendingChanges: 5
            });

            // 部分更新
            statusBarManager.updateState({
                syncStatus: 'syncing'
            });

            expect(mockLogger.debug).toHaveBeenLastCalledWith(
                'Status bar state updated',
                expect.objectContaining({
                    state: expect.objectContaining({
                        connectionStatus: 'connected',
                        syncStatus: 'syncing',
                        pendingChanges: 5
                    })
                })
            );
        });
    });

    describe('destroy', () => {
        it('should remove status bar on destroy', () => {
            statusBarManager.initialize();
            
            statusBarManager.destroy();

            expect(mockStatusBarItem.remove).toHaveBeenCalled();
        });

        it('should handle destroy when status bar not created', () => {
            // 不初始化狀態列
            expect(() => statusBarManager.destroy()).not.toThrow();
        });
    });

    describe('status bar display', () => {
        beforeEach(() => {
            statusBarManager.initialize();
        });

        it('should create connection icon span', () => {
            statusBarManager.updateState({ connectionStatus: 'connected' });

            expect(mockStatusBarItem.createSpan).toHaveBeenCalledWith({
                cls: 'ink-connection-icon'
            });
        });

        it('should create sync icon when syncing', () => {
            statusBarManager.updateState({ syncStatus: 'syncing' });

            expect(mockStatusBarItem.createSpan).toHaveBeenCalledWith({
                cls: 'ink-sync-icon'
            });
        });

        it('should not create sync icon when idle', () => {
            statusBarManager.updateState({ syncStatus: 'idle' });

            // 應該只有連線圖示，沒有同步圖示
            expect(mockStatusBarItem.createSpan).toHaveBeenCalledTimes(1);
            expect(mockStatusBarItem.createSpan).toHaveBeenCalledWith({
                cls: 'ink-connection-icon'
            });
        });

        it('should create pending count span when there are pending changes', () => {
            statusBarManager.updateState({ pendingChanges: 3 });

            expect(mockStatusBarItem.createSpan).toHaveBeenCalledWith({
                cls: 'ink-pending-count',
                text: '3'
            });
        });

        it('should not create pending count span when no pending changes', () => {
            statusBarManager.updateState({ pendingChanges: 0 });

            // 應該只有連線圖示
            expect(mockStatusBarItem.createSpan).toHaveBeenCalledTimes(1);
            expect(mockStatusBarItem.createSpan).toHaveBeenCalledWith({
                cls: 'ink-connection-icon'
            });
        });
    });

    describe('tooltip generation', () => {
        beforeEach(() => {
            statusBarManager.initialize();
        });

        it('should generate tooltip with connection status', () => {
            statusBarManager.updateState({ connectionStatus: 'connected' });

            expect(mockStatusBarItem.title).toContain('Connected to Ink-Gateway');
        });

        it('should include sync information in tooltip', () => {
            const lastSyncTime = new Date();
            statusBarManager.updateState({
                connectionStatus: 'connected',
                syncStatus: 'idle',
                lastSyncTime
            });

            expect(mockStatusBarItem.title).toContain('Last sync:');
        });

        it('should include pending changes in tooltip', () => {
            statusBarManager.updateState({
                connectionStatus: 'connected',
                pendingChanges: 5
            });

            expect(mockStatusBarItem.title).toContain('5 pending changes');
        });

        it('should include cache hit rate in tooltip', () => {
            statusBarManager.updateState({
                connectionStatus: 'connected',
                cacheHitRate: 0.85
            });

            expect(mockStatusBarItem.title).toContain('Cache: 85%');
        });
    });

    describe('click handler', () => {
        it('should register click event listener', () => {
            statusBarManager.initialize();

            expect(mockStatusBarItem.addEventListener).toHaveBeenCalledWith(
                'click',
                expect.any(Function)
            );
        });
    });
});