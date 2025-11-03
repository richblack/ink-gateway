#!/usr/bin/env node

/**
 * 強制重新載入插件腳本
 */

import fs from 'fs';
import path from 'path';

console.log('🔄 Force Reloading Obsidian Plugin...');
console.log('====================================');

// 1. 更新版本號以強制重載
const manifestPath = './manifest.json';
const manifest = JSON.parse(fs.readFileSync(manifestPath, 'utf8'));
const currentVersion = manifest.version;
const versionParts = currentVersion.split('.');
versionParts[2] = (parseInt(versionParts[2]) + 1).toString();
const newVersion = versionParts.join('.');

manifest.version = newVersion;
fs.writeFileSync(manifestPath, JSON.stringify(manifest, null, 2));

console.log(`📦 Version updated: ${currentVersion} → ${newVersion}`);

// 2. 重新編譯
console.log('🔨 Rebuilding plugin...');
import('./esbuild.config.mjs');

console.log('');
console.log('✅ Plugin force-reloaded!');
console.log('');
console.log('🔄 Next steps:');
console.log('1. In Obsidian: Go to Settings → Community plugins');
console.log('2. Disable "Ink Gateway Plugin"');
console.log('3. Enable "Ink Gateway Plugin" again');
console.log('4. Check that version shows as', newVersion);
console.log('5. Test the settings - API key should now be optional');
console.log('');
console.log('💡 Alternative: Restart Obsidian completely');