export interface PluginSettings {
    // Ink-Gateway 連線設定
    inkGatewayUrl: string;
    apiKey: string;
    connectionTimeout: number;
    retryAttempts: number;
    
    // 同步設定
    autoSync: boolean;
    syncInterval: number;
    syncOnEnter: boolean;
    batchSyncSize: number;
    
    // 快取設定
    cacheEnabled: boolean;
    cacheSize: number;
    cacheTTL: number;
    searchCacheEnabled: boolean;
    
    // AI 設定
    aiChatEnabled: boolean;
    aiAutoProcess: boolean;
    aiContextSize: number;
    
    // 搜尋設定
    semanticSearchEnabled: boolean;
    searchResultLimit: number;
    searchHighlightEnabled: boolean;
    
    // 模板設定
    templateAutoApply: boolean;
    templateValidation: boolean;
    
    // 儲存設定
    storageProvider: 'google_drive' | 'local' | 'both';
    googleDriveFolderId: string;
    localStoragePath: string;
    
    // 除錯和記錄
    debugMode: boolean;
    logLevel: 'error' | 'warn' | 'info' | 'debug';
    performanceMonitoring: boolean;
    
    // UI 設定
    showStatusBar: boolean;
    showNotifications: boolean;
    notificationDuration: number;
    
    // 進階設定
    offlineMode: boolean;
    conflictResolution: 'local' | 'remote' | 'manual';
    dataExportFormat: 'json' | 'csv' | 'markdown';
}

export const DEFAULT_SETTINGS: PluginSettings = {
    // Ink-Gateway 連線設定
    inkGatewayUrl: 'http://localhost:8080',
    apiKey: '',
    connectionTimeout: 30000,
    retryAttempts: 3,
    
    // 同步設定
    autoSync: true,
    syncInterval: 5000,
    syncOnEnter: true,
    batchSyncSize: 50,
    
    // 快取設定
    cacheEnabled: true,
    cacheSize: 100,
    cacheTTL: 300000, // 5 minutes
    searchCacheEnabled: true,
    
    // AI 設定
    aiChatEnabled: true,
    aiAutoProcess: false,
    aiContextSize: 10,
    
    // 搜尋設定
    semanticSearchEnabled: true,
    searchResultLimit: 50,
    searchHighlightEnabled: true,
    
    // 模板設定
    templateAutoApply: false,
    templateValidation: true,
    
    // 儲存設定
    storageProvider: 'google_drive',
    googleDriveFolderId: '1Q5rWspN-wqjqnfV0HhfngqhMFVy4QRvl',
    localStoragePath: './uploads',
    
    // 除錯和記錄
    debugMode: false,
    logLevel: 'info',
    performanceMonitoring: false,
    
    // UI 設定
    showStatusBar: true,
    showNotifications: true,
    notificationDuration: 3000,
    
    // 進階設定
    offlineMode: false,
    conflictResolution: 'manual',
    dataExportFormat: 'json'
};

export interface SettingsValidationResult {
    isValid: boolean;
    errors: SettingsValidationError[];
}

export interface SettingsValidationError {
    field: keyof PluginSettings;
    message: string;
    severity: 'error' | 'warning';
}

export interface SettingsExportData {
    version: string;
    timestamp: string;
    settings: PluginSettings;
    metadata?: Record<string, any>;
}