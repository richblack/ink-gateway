#!/usr/bin/env node

/**
 * Obsidian Plugin Development Helper
 * 提供更好的開發體驗和調試信息
 */

import { spawn } from 'child_process';
import { watch } from 'fs';
import path from 'path';

console.log('🚀 Obsidian Ink Plugin - Development Mode');
console.log('=====================================');
console.log('');
console.log('📁 Monitoring: src/ directory');
console.log('🔄 Auto-rebuild: Enabled');
console.log('🔗 Plugin location: ~/.obsidian/plugins/obsidian-ink-plugin');
console.log('');
console.log('💡 Development Tips:');
console.log('   • Files will auto-rebuild when you save changes');
console.log('   • In Obsidian: Press Cmd+R to reload the app');
console.log('   • Or use Developer Console: app.plugins.disablePlugin("obsidian-ink-plugin"); app.plugins.enablePlugin("obsidian-ink-plugin");');
console.log('   • Check console for plugin logs and errors');
console.log('');
console.log('🛠️  Current fixes applied:');
console.log('   ✅ API key validation (warning instead of error)');
console.log('   ✅ URL default changed to localhost:8081');
console.log('   ✅ Google Drive folder link dynamic update');
console.log('   ✅ Storage type display fixes');
console.log('');
console.log('---');

// 啟動 esbuild watch
const buildProcess = spawn('npm', ['run', 'dev'], {
  stdio: 'inherit',
  shell: true
});

buildProcess.on('error', (error) => {
  console.error('❌ Failed to start build process:', error);
});

buildProcess.on('exit', (code) => {
  console.log(`\n📦 Build process exited with code ${code}`);
});

// 監控額外的配置文件變更
const configFiles = ['manifest.json', 'package.json'];
configFiles.forEach(file => {
  watch(file, (eventType, filename) => {
    if (filename) {
      console.log(`📝 Config file changed: ${filename}`);
      console.log('💡 You may need to restart Obsidian to see manifest changes');
    }
  });
});

// 優雅退出
process.on('SIGINT', () => {
  console.log('\n🛑 Stopping development server...');
  buildProcess.kill();
  process.exit(0);
});