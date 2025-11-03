#!/usr/bin/env node

const fs = require('fs');

// 讀取 main.ts 文件
let content = fs.readFileSync('src/main.ts', 'utf8');

// 1. 添加 SettingsTab 導入
const importToAdd = `import { InkPluginSettingsTab } from './settings/SettingsTab';
import { SettingsManager } from './settings/SettingsManager';
import { DebugLogger } from './errors/DebugLogger';`;

// 在 BatchProcessModal 導入後添加新的導入
content = content.replace(
  /import { BatchProcessModal } from '\.\/media\/BatchProcessModal';/,
  `import { BatchProcessModal } from './media/BatchProcessModal';
${importToAdd}`
);

// 2. 添加 settingsManager 和 logger 屬性
const propertiesToAdd = `  
  // Settings and logging
  settingsManager!: SettingsManager;
  logger!: DebugLogger;`;

// 在 dragDropHandler 屬性後添加
content = content.replace(
  /dragDropHandler!: DragDropHandler;/,
  `dragDropHandler!: DragDropHandler;${propertiesToAdd}`
);

// 3. 在 onload 方法中添加設定標籤註冊
const settingsTabRegistration = `
      // Add settings tab
      this.addSettingTab(new InkPluginSettingsTab(
        this.app, 
        this, 
        this.settingsManager, 
        this.logger
      ));`;

// 在 setupStatusBar() 後添加
content = content.replace(
  /this\.setupStatusBar\(\);/,
  `this.setupStatusBar();${settingsTabRegistration}`
);

// 寫回文件
fs.writeFileSync('src/main.ts', content, 'utf8');
console.log('已修復 main.ts 中的設定標籤註冊');