#!/usr/bin/env node

const fs = require('fs');

// 修復 OfflineManager.test.ts 中的重複屬性
const offlineManagerPath = 'src/offline/__tests__/OfflineManager.test.ts';
let content = fs.readFileSync(offlineManagerPath, 'utf8');

// 移除重複的 documentId 和 documentScope
content = content.replace(
  /(\s+documentId: 'test-doc-1',\s+documentScope: 'file' as const,[\s\S]*?cssClasses: \[\]\s+},)\s+documentId: '[^']+',\s+documentScope: '[^']+'/g,
  '$1'
);

fs.writeFileSync(offlineManagerPath, content, 'utf8');
console.log('Fixed OfflineManager.test.ts');

console.log('All duplicates fixed!');