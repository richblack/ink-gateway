#!/usr/bin/env node

const fs = require('fs');
const path = require('path');

// 需要修復的文件列表
const filesToFix = [
  'src/offline/__tests__/OfflineManager.test.ts',
  'src/api/__tests__/InkGatewayClient.test.ts', 
  'src/offline/OfflineManager.ts',
  'src/search/__tests__/SearchManager.test.ts',
  'src/content/__tests__/MetadataManager.test.ts',
  'src/template/TemplateParser.ts',
  'src/content/MarkdownParser.ts',
  'src/template/TemplateRenderer.ts'
];

function fixUnifiedChunkInFile(filePath) {
  console.log(`Fixing ${filePath}...`);
  
  let content = fs.readFileSync(filePath, 'utf8');
  
  // 正則表達式來匹配 UnifiedChunk 對象定義
  const chunkRegex = /(const\s+\w+:\s*UnifiedChunk\s*=\s*\{[\s\S]*?)(position:\s*\{[\s\S]*?\})/g;
  
  content = content.replace(chunkRegex, (match, beforePosition, positionPart) => {
    // 檢查是否已經有 documentId 和 documentScope
    if (match.includes('documentId:') && match.includes('documentScope:')) {
      return match; // 已經修復過了
    }
    
    // 在 position 之前添加 documentId 和 documentScope
    const documentIdLine = '      documentId: \'test-doc-1\',\n';
    const documentScopeLine = '      documentScope: \'file\' as const,\n';
    
    return beforePosition + documentIdLine + documentScopeLine + '      ' + positionPart;
  });
  
  fs.writeFileSync(filePath, content, 'utf8');
  console.log(`Fixed ${filePath}`);
}

// 修復所有文件
filesToFix.forEach(file => {
  const fullPath = path.join(__dirname, file);
  if (fs.existsSync(fullPath)) {
    fixUnifiedChunkInFile(fullPath);
  } else {
    console.log(`File not found: ${fullPath}`);
  }
});

console.log('All files fixed!');