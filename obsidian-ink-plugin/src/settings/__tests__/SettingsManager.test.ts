import { SettingsManager } from '../SettingsManager';
import { DEFAULT_SETTINGS, PluginSettings } from '../PluginSettings';
import { DebugLogger } from '../../errors/DebugLogger';

// Mock Obsidian Plugin
const mockPlugin = {
    loadData: jest.fn(),
    saveData: jest.fn()
};

// Mock DebugLogger
const mockLogger = {
    debug: jest.fn(),
    info: jest.fn(),
    warn: jest.fn(),
    error: jest.fn()
} as unknown as DebugLogger;

// Mock fetch for connection testing
global.fetch = jest.fn();

describe('SettingsManager', () => {
    let settingsManager: SettingsManager;

    beforeEach(() => {
        jest.clearAllMocks();
        settingsManager = new SettingsManager(mockPlugin as any, mockLogger);
    });

    describe('loadSettings', () => {
        it('should load settings from plugin data', async () => {
            const savedSettings = { ...DEFAULT_SETTINGS, inkGatewayUrl: 'http://test.com' };
            mockPlugin.loadData.mockResolvedValue(savedSettings);

            const settings = await settingsManager.loadSettings();

            expect(settings.inkGatewayUrl).toBe('http://test.com');
            expect(mockPlugin.loadData).toHaveBeenCalled();
        });

        it('should use default settings when no data exists', async () => {
            mockPlugin.loadData.mockResolvedValue(null);

            const settings = await settingsManager.loadSettings();

            expect(settings).toEqual(DEFAULT_SETTINGS);
        });

        it('should repair invalid settings', async () => {
            const invalidSettings = { 
                ...DEFAULT_SETTINGS, 
                inkGatewayUrl: 'invalid-url',
                apiKey: ''
            };
            mockPlugin.loadData.mockResolvedValue(invalidSettings);
            mockPlugin.saveData.mockResolvedValue(undefined);

            const settings = await settingsManager.loadSettings();

            expect(settings.inkGatewayUrl).toBe(DEFAULT_SETTINGS.inkGatewayUrl);
            expect(settings.apiKey).toBe(DEFAULT_SETTINGS.apiKey);
            expect(mockPlugin.saveData).toHaveBeenCalled();
        });

        it('should handle load errors gracefully', async () => {
            mockPlugin.loadData.mockRejectedValue(new Error('Load failed'));

            const settings = await settingsManager.loadSettings();

            expect(settings).toEqual(DEFAULT_SETTINGS);
            expect(mockLogger.error).toHaveBeenCalled();
        });
    });

    describe('saveSettings', () => {
        beforeEach(async () => {
            mockPlugin.loadData.mockResolvedValue(DEFAULT_SETTINGS);
            await settingsManager.loadSettings();
        });

        it('should save valid settings', async () => {
            const newSettings = { inkGatewayUrl: 'http://new-url.com' };
            mockPlugin.saveData.mockResolvedValue(undefined);

            await settingsManager.saveSettings(newSettings);

            expect(mockPlugin.saveData).toHaveBeenCalledWith(
                expect.objectContaining(newSettings)
            );
        });

        it('should reject invalid settings', async () => {
            const invalidSettings = { inkGatewayUrl: 'invalid-url' };

            await expect(settingsManager.saveSettings(invalidSettings))
                .rejects.toThrow('Settings validation failed');
        });

        it('should handle save errors', async () => {
            mockPlugin.saveData.mockRejectedValue(new Error('Save failed'));

            await expect(settingsManager.saveSettings({ autoSync: false }))
                .rejects.toThrow('Save failed');
        });
    });

    describe('validateSettings', () => {
        it('should validate correct settings', () => {
            const validSettings: PluginSettings = {
                ...DEFAULT_SETTINGS,
                inkGatewayUrl: 'http://localhost:8080',
                apiKey: 'valid-key'
            };

            const result = settingsManager.validateSettings(validSettings);

            expect(result.isValid).toBe(true);
            expect(result.errors).toHaveLength(0);
        });

        it('should detect invalid URL', () => {
            const invalidSettings = {
                ...DEFAULT_SETTINGS,
                inkGatewayUrl: 'not-a-url'
            };

            const result = settingsManager.validateSettings(invalidSettings);

            expect(result.isValid).toBe(false);
            expect(result.errors).toContainEqual(
                expect.objectContaining({
                    field: 'inkGatewayUrl',
                    severity: 'error'
                })
            );
        });

        it('should detect missing API key', () => {
            const invalidSettings = {
                ...DEFAULT_SETTINGS,
                apiKey: ''
            };

            const result = settingsManager.validateSettings(invalidSettings);

            expect(result.isValid).toBe(false);
            expect(result.errors).toContainEqual(
                expect.objectContaining({
                    field: 'apiKey',
                    severity: 'error'
                })
            );
        });

        it('should detect out-of-range values', () => {
            const invalidSettings = {
                ...DEFAULT_SETTINGS,
                connectionTimeout: 500, // Too low
                retryAttempts: 15, // Too high
                syncInterval: 500, // Too low
                cacheSize: 5000 // Too high
            };

            const result = settingsManager.validateSettings(invalidSettings);

            expect(result.errors.length).toBeGreaterThan(0);
            expect(result.errors).toContainEqual(
                expect.objectContaining({ field: 'connectionTimeout' })
            );
            expect(result.errors).toContainEqual(
                expect.objectContaining({ field: 'retryAttempts' })
            );
            expect(result.errors).toContainEqual(
                expect.objectContaining({ field: 'syncInterval' })
            );
            expect(result.errors).toContainEqual(
                expect.objectContaining({ field: 'cacheSize' })
            );
        });
    });

    describe('updateSetting', () => {
        beforeEach(async () => {
            mockPlugin.loadData.mockResolvedValue(DEFAULT_SETTINGS);
            mockPlugin.saveData.mockResolvedValue(undefined);
            await settingsManager.loadSettings();
        });

        it('should update a single setting', async () => {
            await settingsManager.updateSetting('autoSync', false);

            const settings = settingsManager.getSettings();
            expect(settings.autoSync).toBe(false);
            expect(mockPlugin.saveData).toHaveBeenCalled();
        });

        it('should validate single setting updates', async () => {
            await expect(settingsManager.updateSetting('connectionTimeout', 500))
                .rejects.toThrow();
        });
    });

    describe('resetSettings', () => {
        it('should reset to default settings', async () => {
            mockPlugin.saveData.mockResolvedValue(undefined);

            await settingsManager.resetSettings();

            const settings = settingsManager.getSettings();
            expect(settings).toEqual(DEFAULT_SETTINGS);
            expect(mockPlugin.saveData).toHaveBeenCalledWith(DEFAULT_SETTINGS);
        });
    });

    describe('exportSettings', () => {
        beforeEach(async () => {
            mockPlugin.loadData.mockResolvedValue(DEFAULT_SETTINGS);
            await settingsManager.loadSettings();
        });

        it('should export settings with metadata', () => {
            const exportData = settingsManager.exportSettings();

            expect(exportData.version).toBe('1.0.0');
            expect(exportData.settings).toEqual(DEFAULT_SETTINGS);
            expect(exportData.timestamp).toBeDefined();
            expect(exportData.metadata).toBeDefined();
        });
    });

    describe('importSettings', () => {
        beforeEach(async () => {
            mockPlugin.loadData.mockResolvedValue(DEFAULT_SETTINGS);
            mockPlugin.saveData.mockResolvedValue(undefined);
            await settingsManager.loadSettings();
        });

        it('should import valid settings', async () => {
            const importData = {
                version: '1.0.0',
                timestamp: new Date().toISOString(),
                settings: { ...DEFAULT_SETTINGS, autoSync: false }
            };

            await settingsManager.importSettings(importData);

            const settings = settingsManager.getSettings();
            expect(settings.autoSync).toBe(false);
        });

        it('should reject invalid import data', async () => {
            const invalidImportData = {
                version: '1.0.0',
                timestamp: new Date().toISOString(),
                settings: { ...DEFAULT_SETTINGS, inkGatewayUrl: 'invalid' }
            };

            await expect(settingsManager.importSettings(invalidImportData))
                .rejects.toThrow('Invalid settings');
        });

        it('should reject malformed import data', async () => {
            const malformedData = {
                version: '1.0.0',
                timestamp: new Date().toISOString()
                // Missing settings
            };

            await expect(settingsManager.importSettings(malformedData as any))
                .rejects.toThrow('Invalid export data');
        });
    });

    describe('testConnection', () => {
        beforeEach(async () => {
            const settings = {
                ...DEFAULT_SETTINGS,
                inkGatewayUrl: 'http://localhost:8080',
                apiKey: 'test-key',
                connectionTimeout: 5000
            };
            mockPlugin.loadData.mockResolvedValue(settings);
            await settingsManager.loadSettings();
        });

        it('should test successful connection', async () => {
            const mockResponse = {
                ok: true,
                status: 200,
                statusText: 'OK'
            };
            (global.fetch as jest.Mock).mockResolvedValue(mockResponse);

            const result = await settingsManager.testConnection();

            expect(result.success).toBe(true);
            expect(result.message).toBe('Connection successful');
            expect(result.responseTime).toBeDefined();
            expect(global.fetch).toHaveBeenCalledWith(
                'http://localhost:8080/health',
                expect.objectContaining({
                    method: 'GET',
                    headers: expect.objectContaining({
                        'Authorization': 'Bearer test-key'
                    })
                })
            );
        });

        it('should handle connection failure', async () => {
            const mockResponse = {
                ok: false,
                status: 500,
                statusText: 'Internal Server Error'
            };
            (global.fetch as jest.Mock).mockResolvedValue(mockResponse);

            const result = await settingsManager.testConnection();

            expect(result.success).toBe(false);
            expect(result.message).toContain('Connection failed');
        });

        it('should handle network errors', async () => {
            (global.fetch as jest.Mock).mockRejectedValue(new Error('Network error'));

            const result = await settingsManager.testConnection();

            expect(result.success).toBe(false);
            expect(result.message).toContain('Connection error');
        });

        it('should handle timeout', async () => {
            (global.fetch as jest.Mock).mockImplementation(() => 
                new Promise((resolve) => {
                    setTimeout(() => resolve({ ok: true }), 10000);
                })
            );

            const result = await settingsManager.testConnection();

            expect(result.success).toBe(false);
            expect(result.message).toBe('Connection timeout');
        });
    });

    describe('change listeners', () => {
        let changeListener: jest.Mock;

        beforeEach(async () => {
            changeListener = jest.fn();
            mockPlugin.loadData.mockResolvedValue(DEFAULT_SETTINGS);
            mockPlugin.saveData.mockResolvedValue(undefined);
            await settingsManager.loadSettings();
            settingsManager.onSettingsChange(changeListener);
        });

        it('should notify listeners on settings change', async () => {
            await settingsManager.saveSettings({ autoSync: false });

            expect(changeListener).toHaveBeenCalledWith(
                expect.objectContaining({ autoSync: false })
            );
        });

        it('should remove listeners', async () => {
            settingsManager.removeSettingsChangeListener(changeListener);
            await settingsManager.saveSettings({ autoSync: false });

            expect(changeListener).not.toHaveBeenCalled();
        });

        it('should handle listener errors gracefully', async () => {
            const errorListener = jest.fn().mockImplementation(() => {
                throw new Error('Listener error');
            });
            settingsManager.onSettingsChange(errorListener);

            await settingsManager.saveSettings({ autoSync: false });

            expect(mockLogger.error).toHaveBeenCalled();
            expect(changeListener).toHaveBeenCalled(); // Other listeners should still work
        });
    });
});