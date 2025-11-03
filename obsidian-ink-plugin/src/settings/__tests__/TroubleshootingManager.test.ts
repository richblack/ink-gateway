import { TroubleshootingManager, DiagnosticResult } from '../TroubleshootingManager';
import { SettingsManager } from '../SettingsManager';
import { DEFAULT_SETTINGS } from '../PluginSettings';
import { DebugLogger } from '../../errors/DebugLogger';

// Mock Obsidian App
const mockApp = {
    appVersion: '1.0.0'
};

// Mock SettingsManager
const mockSettingsManager = {
    getSettings: jest.fn(() => DEFAULT_SETTINGS),
    validateSettings: jest.fn(() => ({ isValid: true, errors: [] })),
    testConnection: jest.fn(() => Promise.resolve({ success: true, message: 'Connected', responseTime: 100 })),
    saveSettings: jest.fn(() => Promise.resolve())
};

// Mock DebugLogger
const mockLogger = {
    debug: jest.fn(),
    info: jest.fn(),
    warn: jest.fn(),
    error: jest.fn()
} as unknown as DebugLogger;

// Mock fetch
global.fetch = jest.fn();

// Mock navigator
Object.defineProperty(global.navigator, 'platform', {
    value: 'MacIntel',
    writable: true
});

Object.defineProperty(global.navigator, 'userAgent', {
    value: 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)',
    writable: true
});

// Mock localStorage
const localStorageMock = {
    setItem: jest.fn(),
    removeItem: jest.fn()
};
Object.defineProperty(global, 'localStorage', {
    value: localStorageMock
});

// Mock performance.memory
Object.defineProperty(global.performance, 'memory', {
    value: {
        usedJSHeapSize: 50 * 1024 * 1024, // 50MB
        jsHeapSizeLimit: 100 * 1024 * 1024 // 100MB
    },
    writable: true
});

// Mock navigator.storage
Object.defineProperty(global.navigator, 'storage', {
    value: {
        estimate: jest.fn(() => Promise.resolve({
            usage: 100 * 1024 * 1024, // 100MB
            quota: 1000 * 1024 * 1024 // 1GB
        }))
    }
});

describe('TroubleshootingManager', () => {
    let troubleshootingManager: TroubleshootingManager;

    beforeEach(() => {
        jest.clearAllMocks();
        troubleshootingManager = new TroubleshootingManager(
            mockApp as any,
            mockSettingsManager as any,
            mockLogger
        );
    });

    describe('runFullDiagnostics', () => {
        it('should run all diagnostic categories', async () => {
            (global.fetch as jest.Mock).mockResolvedValue({
                ok: true,
                status: 200,
                statusText: 'OK'
            });

            const results = await troubleshootingManager.runFullDiagnostics();

            expect(results.length).toBeGreaterThan(0);
            
            // 檢查是否包含各個類別
            const categories = [...new Set(results.map(r => r.category))];
            expect(categories).toContain('System');
            expect(categories).toContain('Settings');
            expect(categories).toContain('Network');
            expect(categories).toContain('API');
        });

        it('should handle diagnostic errors gracefully', async () => {
            mockSettingsManager.validateSettings.mockImplementation(() => {
                throw new Error('Validation failed');
            });

            const results = await troubleshootingManager.runFullDiagnostics();

            expect(results).toContainEqual(
                expect.objectContaining({
                    category: 'System',
                    test: 'Diagnostic Execution',
                    status: 'fail'
                })
            );
        });

        it('should log diagnostic completion', async () => {
            await troubleshootingManager.runFullDiagnostics();

            expect(mockLogger.info).toHaveBeenCalledWith(
                'Full diagnostics completed',
                expect.objectContaining({
                    totalTests: expect.any(Number),
                    passed: expect.any(Number),
                    failed: expect.any(Number),
                    warnings: expect.any(Number)
                })
            );
        });
    });

    describe('checkSystemRequirements', () => {
        it('should pass browser support check', async () => {
            const results = await troubleshootingManager.runFullDiagnostics();
            const browserTest = results.find(r => r.test === 'Browser Support');

            expect(browserTest).toEqual(
                expect.objectContaining({
                    category: 'System',
                    test: 'Browser Support',
                    status: 'pass',
                    message: 'Fetch API is supported'
                })
            );
        });

        it('should pass local storage check', async () => {
            const results = await troubleshootingManager.runFullDiagnostics();
            const storageTest = results.find(r => r.test === 'Local Storage');

            expect(storageTest).toEqual(
                expect.objectContaining({
                    category: 'System',
                    test: 'Local Storage',
                    status: 'pass',
                    message: 'Local storage is available'
                })
            );
        });

        it('should handle local storage errors', async () => {
            localStorageMock.setItem.mockImplementation(() => {
                throw new Error('Storage quota exceeded');
            });

            const results = await troubleshootingManager.runFullDiagnostics();
            const storageTest = results.find(r => r.test === 'Local Storage');

            expect(storageTest).toEqual(
                expect.objectContaining({
                    category: 'System',
                    test: 'Local Storage',
                    status: 'fail',
                    message: 'Local storage is not available'
                })
            );
        });

        it('should check memory usage', async () => {
            const results = await troubleshootingManager.runFullDiagnostics();
            const memoryTest = results.find(r => r.test === 'Memory Usage');

            expect(memoryTest).toEqual(
                expect.objectContaining({
                    category: 'System',
                    test: 'Memory Usage',
                    status: 'pass', // 50MB/100MB = 50% < 80%
                    message: expect.stringContaining('Memory usage:')
                })
            );
        });

        it('should warn about high memory usage', async () => {
            // 模擬高記憶體使用
            Object.defineProperty(global.performance, 'memory', {
                value: {
                    usedJSHeapSize: 90 * 1024 * 1024, // 90MB
                    jsHeapSizeLimit: 100 * 1024 * 1024 // 100MB
                }
            });

            const results = await troubleshootingManager.runFullDiagnostics();
            const memoryTest = results.find(r => r.test === 'Memory Usage');

            expect(memoryTest?.status).toBe('warning');
            expect(memoryTest?.fix).toContain('restarting Obsidian');
        });
    });

    describe('validateSettings', () => {
        it('should pass when settings are valid', async () => {
            mockSettingsManager.validateSettings.mockReturnValue({
                isValid: true,
                errors: []
            });

            const results = await troubleshootingManager.runFullDiagnostics();
            const settingsTest = results.find(r => r.test === 'Settings Validation');

            expect(settingsTest).toEqual(
                expect.objectContaining({
                    category: 'Settings',
                    test: 'Settings Validation',
                    status: 'pass',
                    message: 'All settings are valid'
                })
            );
        });

        it('should report validation errors', async () => {
            mockSettingsManager.validateSettings.mockReturnValue({
                isValid: false,
                errors: [
                    {
                        field: 'inkGatewayUrl',
                        message: 'Invalid URL format',
                        severity: 'error'
                    },
                    {
                        field: 'connectionTimeout',
                        message: 'Timeout too short',
                        severity: 'warning'
                    }
                ]
            });

            const results = await troubleshootingManager.runFullDiagnostics();
            const urlTest = results.find(r => r.test === 'Setting: inkGatewayUrl');
            const timeoutTest = results.find(r => r.test === 'Setting: connectionTimeout');

            expect(urlTest?.status).toBe('fail');
            expect(timeoutTest?.status).toBe('warning');
        });

        it('should check for missing gateway URL', async () => {
            mockSettingsManager.getSettings.mockReturnValue({
                ...DEFAULT_SETTINGS,
                inkGatewayUrl: ''
            });

            const results = await troubleshootingManager.runFullDiagnostics();
            const gatewayTest = results.find(r => r.test === 'Gateway URL');

            expect(gatewayTest).toEqual(
                expect.objectContaining({
                    category: 'Settings',
                    test: 'Gateway URL',
                    status: 'fail',
                    message: 'Ink-Gateway URL is not configured'
                })
            );
        });

        it('should check for missing API key', async () => {
            mockSettingsManager.getSettings.mockReturnValue({
                ...DEFAULT_SETTINGS,
                apiKey: ''
            });

            const results = await troubleshootingManager.runFullDiagnostics();
            const apiKeyTest = results.find(r => r.test === 'API Key');

            expect(apiKeyTest).toEqual(
                expect.objectContaining({
                    category: 'Settings',
                    test: 'API Key',
                    status: 'fail',
                    message: 'API key is not configured'
                })
            );
        });
    });

    describe('checkNetworkConnectivity', () => {
        it('should pass when internet connection works', async () => {
            (global.fetch as jest.Mock).mockResolvedValue({
                ok: true,
                status: 200,
                statusText: 'OK'
            });

            const results = await troubleshootingManager.runFullDiagnostics();
            const networkTest = results.find(r => r.test === 'Internet Connectivity');

            expect(networkTest).toEqual(
                expect.objectContaining({
                    category: 'Network',
                    test: 'Internet Connectivity',
                    status: 'pass',
                    message: 'Internet connection is working'
                })
            );
        });

        it('should warn on HTTP errors', async () => {
            (global.fetch as jest.Mock).mockResolvedValue({
                ok: false,
                status: 500,
                statusText: 'Internal Server Error'
            });

            const results = await troubleshootingManager.runFullDiagnostics();
            const networkTest = results.find(r => r.test === 'Internet Connectivity');

            expect(networkTest).toEqual(
                expect.objectContaining({
                    category: 'Network',
                    test: 'Internet Connectivity',
                    status: 'warning',
                    message: 'HTTP 500: Internal Server Error'
                })
            );
        });

        it('should fail when network is unavailable', async () => {
            (global.fetch as jest.Mock).mockRejectedValue(new Error('Network error'));

            const results = await troubleshootingManager.runFullDiagnostics();
            const networkTest = results.find(r => r.test === 'Internet Connectivity');

            expect(networkTest).toEqual(
                expect.objectContaining({
                    category: 'Network',
                    test: 'Internet Connectivity',
                    status: 'fail',
                    message: 'No internet connection'
                })
            );
        });
    });

    describe('checkAPIConnection', () => {
        it('should fail when API configuration is incomplete', async () => {
            mockSettingsManager.getSettings.mockReturnValue({
                ...DEFAULT_SETTINGS,
                inkGatewayUrl: '',
                apiKey: ''
            });

            const results = await troubleshootingManager.runFullDiagnostics();
            const apiTest = results.find(r => r.test === 'API Configuration');

            expect(apiTest).toEqual(
                expect.objectContaining({
                    category: 'API',
                    test: 'API Configuration',
                    status: 'fail',
                    message: 'API configuration is incomplete'
                })
            );
        });

        it('should pass when API connection succeeds', async () => {
            mockSettingsManager.testConnection.mockResolvedValue({
                success: true,
                message: 'Connection successful',
                responseTime: 150
            });

            const results = await troubleshootingManager.runFullDiagnostics();
            const connectionTest = results.find(r => r.test === 'Gateway Connection');

            expect(connectionTest).toEqual(
                expect.objectContaining({
                    category: 'API',
                    test: 'Gateway Connection',
                    status: 'pass',
                    message: 'Connection successful',
                    details: 'Response time: 150ms'
                })
            );
        });

        it('should fail when API connection fails', async () => {
            mockSettingsManager.testConnection.mockResolvedValue({
                success: false,
                message: 'Connection timeout'
            });

            const results = await troubleshootingManager.runFullDiagnostics();
            const connectionTest = results.find(r => r.test === 'Gateway Connection');

            expect(connectionTest).toEqual(
                expect.objectContaining({
                    category: 'API',
                    test: 'Gateway Connection',
                    status: 'fail',
                    message: 'Connection timeout'
                })
            );
        });

        it('should check API version when connection succeeds', async () => {
            mockSettingsManager.testConnection.mockResolvedValue({
                success: true,
                message: 'Connection successful',
                responseTime: 100
            });

            (global.fetch as jest.Mock)
                .mockResolvedValueOnce({ ok: true }) // 網路連線測試
                .mockResolvedValueOnce({ // API 版本檢查
                    ok: true,
                    json: () => Promise.resolve({ version: '2.1.0' })
                });

            const results = await troubleshootingManager.runFullDiagnostics();
            const versionTest = results.find(r => r.test === 'API Version');

            expect(versionTest).toEqual(
                expect.objectContaining({
                    category: 'API',
                    test: 'API Version',
                    status: 'info',
                    message: 'Gateway version: 2.1.0'
                })
            );
        });
    });

    describe('getSystemInfo', () => {
        it('should return system information', () => {
            const systemInfo = troubleshootingManager.getSystemInfo();

            expect(systemInfo).toEqual({
                platform: 'MacIntel',
                userAgent: 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)',
                obsidianVersion: '1.0.0',
                pluginVersion: '1.0.0',
                timestamp: expect.any(String)
            });
        });
    });

    describe('exportDiagnosticReport', () => {
        it('should export diagnostic report with all information', async () => {
            const reportJson = await troubleshootingManager.exportDiagnosticReport();
            const report = JSON.parse(reportJson);

            expect(report).toHaveProperty('systemInfo');
            expect(report).toHaveProperty('diagnostics');
            expect(report).toHaveProperty('settings');
            expect(report).toHaveProperty('timestamp');

            // 檢查敏感資訊是否被排除
            expect(report.settings).not.toHaveProperty('apiKey');
        });
    });

    describe('autoFix', () => {
        it('should fix invalid settings', async () => {
            mockSettingsManager.validateSettings.mockReturnValue({
                isValid: false,
                errors: [
                    {
                        field: 'connectionTimeout',
                        message: 'Invalid timeout',
                        severity: 'error'
                    }
                ]
            });

            const result = await troubleshootingManager.autoFix();

            expect(result.fixed).toBe(1);
            expect(result.messages).toContain('Fixed 1 invalid settings');
            expect(mockSettingsManager.saveSettings).toHaveBeenCalled();
        });

        it('should handle auto-fix errors', async () => {
            mockSettingsManager.saveSettings.mockRejectedValue(new Error('Save failed'));
            mockSettingsManager.validateSettings.mockReturnValue({
                isValid: false,
                errors: [
                    {
                        field: 'connectionTimeout',
                        message: 'Invalid timeout',
                        severity: 'error'
                    }
                ]
            });

            const result = await troubleshootingManager.autoFix();

            expect(result.failed).toBe(1);
            expect(result.messages).toContain(expect.stringContaining('Auto-fix failed'));
        });

        it('should not fix warnings, only errors', async () => {
            mockSettingsManager.validateSettings.mockReturnValue({
                isValid: true, // 整體有效，只有警告
                errors: [
                    {
                        field: 'syncInterval',
                        message: 'Sync interval is short',
                        severity: 'warning'
                    }
                ]
            });

            const result = await troubleshootingManager.autoFix();

            expect(result.fixed).toBe(0);
            expect(mockSettingsManager.saveSettings).not.toHaveBeenCalled();
        });
    });
});