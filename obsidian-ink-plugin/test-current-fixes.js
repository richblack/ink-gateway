#!/usr/bin/env node

/**
 * 測試當前修復是否已正確應用
 */

import fs from 'fs';
import path from 'path';

console.log('🔍 Testing Current Fixes...');
console.log('==========================');

// 檢查編譯後的文件
const mainJsPath = './main.js';
if (!fs.existsSync(mainJsPath)) {
  console.log('❌ main.js not found. Run npm run build first.');
  process.exit(1);
}

const mainJsContent = fs.readFileSync(mainJsPath, 'utf8');

// 測試 1: API key 驗證修復
const hasApiKeyFix = mainJsContent.includes('API key is recommended for full functionality');
console.log(`${hasApiKeyFix ? '✅' : '❌'} API Key Fix: ${hasApiKeyFix ? 'Applied' : 'Missing'}`);

// 測試 2: URL 設置
const hasCorrectUrl = mainJsContent.includes('localhost:8081');
const hasOldUrl = mainJsContent.includes('localhost:8080');
console.log(`${hasCorrectUrl ? '✅' : '❌'} URL Fix: ${hasCorrectUrl ? 'localhost:8081 found' : 'localhost:8081 missing'}`);
if (hasOldUrl) {
  console.log(`⚠️  Warning: Still contains localhost:8080 references`);
}

// 測試 3: 檢查 manifest 版本
const manifestPath = './manifest.json';
if (fs.existsSync(manifestPath)) {
  const manifest = JSON.parse(fs.readFileSync(manifestPath, 'utf8'));
  console.log(`📦 Plugin Version: ${manifest.version}`);
} else {
  console.log('❌ manifest.json not found');
}

// 測試 4: 檢查符號鏈接
const pluginPath = path.join(process.env.HOME, '.obsidian/plugins/obsidian-ink-plugin');
try {
  const stats = fs.lstatSync(pluginPath);
  const isSymlink = stats.isSymbolicLink();
  console.log(`${isSymlink ? '✅' : '❌'} Development Setup: ${isSymlink ? 'Symlink active' : 'Not using symlink'}`);
  
  if (isSymlink) {
    const target = fs.readlinkSync(pluginPath);
    console.log(`🔗 Symlink target: ${target}`);
  }
} catch (error) {
  console.log('❌ Plugin directory not found in Obsidian');
}

console.log('');
console.log('💡 Next Steps:');
console.log('1. If fixes are applied: Start dev mode with `npm run dev:enhanced`');
console.log('2. In Obsidian: Press Cmd+R to reload');
console.log('3. Test the plugin settings');
console.log('4. Check Developer Console for any errors');