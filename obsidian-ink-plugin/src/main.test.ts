/**
 * Main Plugin Tests
 * Tests for the main plugin class and lifecycle
 */

import ObsidianInkPlugin from './main';
import { App, Plugin } from 'obsidian';
import { mockEnvironment } from '../tests/mock-data/mock-environment';

// Mock Obsidian
jest.mock('obsidian');

describe('ObsidianInkPlugin', () => {
    let plugin: ObsidianInkPlugin;
    let mockApp: jest.Mocked<App>;

    beforeEach(() => {
        mockApp = mockEnvironment.mockApp;
        plugin = new ObsidianInkPlugin(mockApp, {} as any);
    });

    afterEach(() => {
        mockEnvironment.reset();
    });

    describe('Plugin Lifecycle', () => {
        it('should initialize plugin correctly', () => {
            expect(plugin).toBeInstanceOf(Plugin);
            expect(plugin.app).toBe(mockApp);
        });

        it('should load plugin successfully', async () => {
            await plugin.onload();
            
            // Verify core managers are initialized
            expect(plugin.contentManager).toBeDefined();
            expect(plugin.searchManager).toBeDefined();
            expect(plugin.aiManager).toBeDefined();
            expect(plugin.templateManager).toBeDefined();
            expect(plugin.apiClient).toBeDefined();
            expect(plugin.cacheManager).toBeDefined();
            expect(plugin.offlineManager).toBeDefined();
            expect(plugin.performanceMonitor).toBeDefined();
        });

        it('should unload plugin cleanly', async () => {
            await plugin.onload();
            await plugin.onunload();
            
            // Verify cleanup was performed
            expect(plugin.contentManager.cleanup).toHaveBeenCalled();
            expect(plugin.cacheManager.clearAll).toHaveBeenCalled();
        });

        it('should handle load errors gracefully', async () => {
            // Mock a component that fails to initialize
            const originalConsoleError = console.error;
            console.error = jest.fn();
            
            // Force an error during initialization
            plugin.loadSettings = jest.fn().mockRejectedValue(new Error('Settings load failed'));
            
            await expect(plugin.onload()).rejects.toThrow('Settings load failed');
            
            console.error = originalConsoleError;
        });
    });

    describe('Settings Management', () => {
        it('should load default settings', async () => {
            await plugin.loadSettings();
            
            expect(plugin.settings).toBeDefined();
            expect(plugin.settings.inkGatewayUrl).toBe('http://localhost:8080');
            expect(plugin.settings.autoSync).toBe(true);
            expect(plugin.settings.cacheEnabled).toBe(true);
        });

        it('should save settings correctly', async () => {
            plugin.settings = {
                inkGatewayUrl: 'https://custom-gateway.com',
                apiKey: 'test-key',
                autoSync: false,
                syncInterval: 30000,
                cacheEnabled: false,
                debugMode: true,
            };
            
            await plugin.saveSettings();
            
            expect(mockApp.vault.adapter.write).toHaveBeenCalledWith(
                expect.stringContaining('data.json'),
                expect.stringContaining('custom-gateway.com')
            );
        });

        it('should merge saved settings with defaults', async () => {
            // Mock existing settings file
            mockApp.vault.adapter.read = jest.fn().mockResolvedValue(
                JSON.stringify({ inkGatewayUrl: 'https://saved-url.com' })
            );
            
            await plugin.loadSettings();
            
            expect(plugin.settings.inkGatewayUrl).toBe('https://saved-url.com');
            expect(plugin.settings.autoSync).toBe(true); // Default value
        });
    });

    describe('Command Registration', () => {
        it('should register all plugin commands', async () => {
            await plugin.onload();
            
            const expectedCommands = [
                'open-ai-chat',
                'open-search-view',
                'sync-content',
                'create-template',
                'toggle-auto-sync',
                'clear-cache',
                'show-performance-stats',
            ];
            
            expectedCommands.forEach(commandId => {
                expect(plugin.addCommand).toHaveBeenCalledWith(
                    expect.objectContaining({ id: commandId })
                );
            });
        });

        it('should handle command execution', async () => {
            await plugin.onload();
            
            // Find the sync command
            const syncCommand = (plugin.addCommand as jest.Mock).mock.calls
                .find(call => call[0].id === 'sync-content');
            
            expect(syncCommand).toBeDefined();
            
            // Execute the command
            await syncCommand[0].callback();
            
            // Verify sync was triggered
            expect(plugin.contentManager.syncAllContent).toHaveBeenCalled();
        });
    });

    describe('Event Handling', () => {
        it('should register file change listeners', async () => {
            await plugin.onload();
            
            expect(mockApp.vault.on).toHaveBeenCalledWith(
                'modify',
                expect.any(Function)
            );
            expect(mockApp.vault.on).toHaveBeenCalledWith(
                'create',
                expect.any(Function)
            );
            expect(mockApp.vault.on).toHaveBeenCalledWith(
                'delete',
                expect.any(Function)
            );
        });

        it('should handle file modification events', async () => {
            await plugin.onload();
            
            // Get the modify event handler
            const modifyHandler = (mockApp.vault.on as jest.Mock).mock.calls
                .find(call => call[0] === 'modify')[1];
            
            const mockFile = mockEnvironment.addFile('test.md', '# Test Content').file;
            
            // Trigger the event
            await modifyHandler(mockFile);
            
            // Verify content processing was triggered
            expect(plugin.contentManager.handleContentChange).toHaveBeenCalledWith(mockFile);
        });

        it('should handle workspace events', async () => {
            await plugin.onload();
            
            expect(mockApp.workspace.on).toHaveBeenCalledWith(
                'active-leaf-change',
                expect.any(Function)
            );
        });
    });

    describe('View Management', () => {
        it('should register custom views', async () => {
            await plugin.onload();
            
            expect(plugin.registerView).toHaveBeenCalledWith(
                'ink-search-view',
                expect.any(Function)
            );
            expect(plugin.registerView).toHaveBeenCalledWith(
                'ink-chat-view',
                expect.any(Function)
            );
        });

        it('should open search view', async () => {
            await plugin.onload();
            
            const leaf = { setViewState: jest.fn() };
            mockApp.workspace.getLeaf = jest.fn().mockReturnValue(leaf);
            
            await plugin.openSearchView();
            
            expect(leaf.setViewState).toHaveBeenCalledWith({
                type: 'ink-search-view',
                active: true,
            });
        });

        it('should open AI chat view', async () => {
            await plugin.onload();
            
            const leaf = { setViewState: jest.fn() };
            mockApp.workspace.getLeaf = jest.fn().mockReturnValue(leaf);
            
            await plugin.openAIChatView();
            
            expect(leaf.setViewState).toHaveBeenCalledWith({
                type: 'ink-chat-view',
                active: true,
            });
        });
    });

    describe('Status Bar Integration', () => {
        it('should add status bar item', async () => {
            await plugin.onload();
            
            expect(plugin.addStatusBarItem).toHaveBeenCalled();
            expect(plugin.statusBarManager).toBeDefined();
        });

        it('should update status bar on sync events', async () => {
            await plugin.onload();
            
            // Trigger sync event
            plugin.contentManager.trigger('sync-start');
            
            expect(plugin.statusBarManager.updateSyncStatus).toHaveBeenCalledWith('syncing');
        });
    });

    describe('Settings Tab Integration', () => {
        it('should add settings tab', async () => {
            await plugin.onload();
            
            expect(plugin.addSettingTab).toHaveBeenCalledWith(
                expect.any(Object)
            );
        });
    });

    describe('Error Handling', () => {
        it('should handle initialization errors', async () => {
            const consoleSpy = jest.spyOn(console, 'error').mockImplementation();
            
            // Mock component initialization failure
            plugin.contentManager = null as any;
            
            await plugin.onload();
            
            expect(consoleSpy).toHaveBeenCalledWith(
                expect.stringContaining('Failed to initialize')
            );
            
            consoleSpy.mockRestore();
        });

        it('should handle API connection errors', async () => {
            mockEnvironment.setNetworkCondition(false);
            
            await plugin.onload();
            
            // Verify error handling was set up
            expect(plugin.errorHandler).toBeDefined();
            expect(plugin.errorHandler.handleError).toBeDefined();
        });
    });

    describe('Performance Monitoring', () => {
        it('should initialize performance monitoring', async () => {
            await plugin.onload();
            
            expect(plugin.performanceMonitor).toBeDefined();
            expect(plugin.performanceMonitor.startMonitoring).toHaveBeenCalled();
        });

        it('should track plugin metrics', async () => {
            await plugin.onload();
            
            // Simulate some operations
            await plugin.contentManager.handleContentChange(
                mockEnvironment.addFile('test.md', '# Test').file
            );
            
            const metrics = plugin.performanceMonitor.getMetrics();
            expect(metrics.counters['content-processed']).toBeGreaterThan(0);
        });
    });

    describe('Cache Management', () => {
        it('should initialize cache with correct settings', async () => {
            plugin.settings.cacheEnabled = true;
            
            await plugin.onload();
            
            expect(plugin.cacheManager.isEnabled()).toBe(true);
        });

        it('should disable cache when setting is false', async () => {
            plugin.settings.cacheEnabled = false;
            
            await plugin.onload();
            
            expect(plugin.cacheManager.isEnabled()).toBe(false);
        });
    });

    describe('Offline Support', () => {
        it('should initialize offline manager', async () => {
            await plugin.onload();
            
            expect(plugin.offlineManager).toBeDefined();
            expect(plugin.offlineManager.startMonitoring).toHaveBeenCalled();
        });

        it('should handle offline state changes', async () => {
            await plugin.onload();
            
            // Simulate going offline
            plugin.offlineManager.trigger('offline');
            
            expect(plugin.statusBarManager.updateConnectionStatus).toHaveBeenCalledWith('offline');
        });
    });
});