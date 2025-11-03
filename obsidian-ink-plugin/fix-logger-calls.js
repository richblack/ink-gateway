#!/usr/bin/env node

const fs = require('fs');
const path = require('path');
const glob = require('glob');

// 查找所有 TypeScript 文件
const files = glob.sync('src/**/*.ts', { cwd: __dirname });

function fixLoggerCalls(filePath) {
  console.log(`Fixing logger calls in ${filePath}...`);
  
  let content = fs.readFileSync(filePath, 'utf8');
  let modified = false;
  
  // 修復 debug 調用 - 缺少 operation 參數
  content = content.replace(
    /this\.logger\.debug\('([^']+)',\s*\{([^}]+)\}\)/g,
    "this.logger.debug('$1', 'operation', '$1', { $2 })"
  );
  
  // 修復 debug 調用 - 只有一個字符串參數
  content = content.replace(
    /this\.logger\.debug\('([^']+)'\)/g,
    "this.logger.debug('component', 'operation', '$1')"
  );
  
  // 修復 info 調用 - 缺少 operation 參數
  content = content.replace(
    /this\.logger\.info\('([^']+)',\s*\{([^}]+)\}\)/g,
    "this.logger.info('component', 'operation', '$1', { $2 })"
  );
  
  // 修復 info 調用 - 只有一個字符串參數
  content = content.replace(
    /this\.logger\.info\('([^']+)'\)/g,
    "this.logger.info('component', 'operation', '$1')"
  );
  
  // 修復 warn 調用 - 缺少 operation 參數
  content = content.replace(
    /this\.logger\.warn\('([^']+)',\s*\{([^}]+)\}\)/g,
    "this.logger.warn('component', 'operation', '$1', { $2 })"
  );
  
  // 修復 error 調用 - 缺少 operation 參數
  content = content.replace(
    /this\.logger\.error\('([^']+)',\s*\{([^}]+)\}\)/g,
    "this.logger.error('component', 'operation', '$1', undefined, { $2 })"
  );
  
  if (content !== fs.readFileSync(filePath, 'utf8')) {
    fs.writeFileSync(filePath, content, 'utf8');
    console.log(`Fixed ${filePath}`);
    modified = true;
  }
  
  return modified;
}

// 修復所有文件
let totalFixed = 0;
files.forEach(file => {
  const fullPath = path.join(__dirname, file);
  if (fs.existsSync(fullPath)) {
    if (fixLoggerCalls(fullPath)) {
      totalFixed++;
    }
  }
});

console.log(`Fixed logger calls in ${totalFixed} files!`);