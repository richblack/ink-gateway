#!/usr/bin/env node

const fs = require('fs');

const mockMethods = `
  async chatWithAI(message: string, context?: string[]): Promise<any> { return { message: 'Mock response', suggestions: [], actions: [], metadata: { model: 'mock', processingTime: 100, confidence: 0.9 } }; }
  async processContent(content: string): Promise<any> { return { chunks: [], suggestions: [], improvements: [] }; }
  async createTemplate(template: any): Promise<any> { return template; }
  async getTemplateInstances(templateId: string): Promise<any[]> { return []; }
  async getChunksByDocumentId(documentId: string, options?: any): Promise<any> { return { chunks: [], pagination: { currentPage: 1, totalPages: 1, totalChunks: 0, pageSize: 10 }, documentMetadata: { documentScope: 'file', totalChunks: 0, lastModified: new Date() } }; }
  async createVirtualDocument(context: any): Promise<any> { return { virtualDocumentId: 'mock-virtual-doc', context, chunkIds: [], createdAt: new Date(), lastUpdated: new Date() }; }
  async updateDocumentScope(chunkId: string, documentId: string, scope: any): Promise<void> {}
  async request<T = any>(config: any): Promise<any> { return { data: {} as T, status: 200, statusText: 'OK', headers: {} }; }`;

const filesToFix = [
  'src/content/__tests__/MetadataManager.test.ts',
  'src/content/__tests__/SyncManager.test.ts'
];

filesToFix.forEach(filePath => {
  if (fs.existsSync(filePath)) {
    let content = fs.readFileSync(filePath, 'utf8');
    
    // 在 updateHierarchy 方法後添加缺少的方法
    content = content.replace(
      /async updateHierarchy\(relations: any\[\]\): Promise<void> \{\}/,
      `async updateHierarchy(relations: any[]): Promise<void> {}${mockMethods}`
    );
    
    fs.writeFileSync(filePath, content, 'utf8');
    console.log(`Fixed ${filePath}`);
  }
});

console.log('All mock clients fixed!');