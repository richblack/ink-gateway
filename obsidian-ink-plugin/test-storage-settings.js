// 簡單測試腳本來驗證 storage 設置
const { DEFAULT_SETTINGS } = require('./lib/settings/PluginSettings.js');

console.log('Testing storage settings...');
console.log('Default storage provider:', DEFAULT_SETTINGS.storageProvider);
console.log('Default Google Drive folder ID:', DEFAULT_SETTINGS.googleDriveFolderId);
console.log('Default local storage path:', DEFAULT_SETTINGS.localStoragePath);

// 驗證設置值
if (DEFAULT_SETTINGS.storageProvider === 'google_drive') {
    console.log('✅ Storage provider correctly set to Google Drive');
} else {
    console.log('❌ Storage provider not set correctly');
}

if (DEFAULT_SETTINGS.googleDriveFolderId) {
    console.log('✅ Google Drive folder ID is set');
} else {
    console.log('❌ Google Drive folder ID is missing');
}

if (DEFAULT_SETTINGS.localStoragePath) {
    console.log('✅ Local storage path is set');
} else {
    console.log('❌ Local storage path is missing');
}

console.log('Storage settings test completed!');