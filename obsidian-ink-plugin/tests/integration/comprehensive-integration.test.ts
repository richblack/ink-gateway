import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { TFile, Vault, MetadataCache } from 'obsidian';
import { ObsidianInkPlugin } from '../../src/main';
import { InkGatewayClient } from '../../src/api/InkGatewayClient';
import { ContentManager } from '../../src/content/ContentManager';
import { SearchManager } from '../../src/search/SearchManager';
import { AIManager } from '../../src/ai/AIManager';
import { TemplateManager } from '../../src/template/TemplateManager';
import { UnifiedChunk, SearchQuery, Template } from '../../src/types';
import { createMockEnvironment, MockObsidianApp } from '../mock-data/mock-environment';

/**
 * 完整系統整合測試
 * 驗證所有需求 1.1-10.7 的整合功能
 */
describe('Comprehensive System Integration Tests', () => {
    let plugin: ObsidianInkPlugin;
    let mockApp: MockObsidianApp;
    let mockVault: Vault;
    let mockFile: TFile;
    let apiClient: InkGatewayClient;

    beforeEach(async () => {
        const mockEnv = createMockEnvironment();
        mockApp = mockEnv.app;
        mockVault = mockEnv.vault;
        
        // 創建測試檔案
        mockFile = {
            path: 'test-note.md',
            name: 'test-note.md',
            basename: 'test-note',
            extension: 'md',
            stat: { ctime: Date.now(), mtime: Date.now(), size: 1000 },
            vault: mockVault
        } as TFile;

        // 初始化插件
        plugin = new ObsidianInkPlugin(mockApp, {} as any);
        await plugin.onload();
        
        apiClient = plugin.apiClient;
    });

    afterEach(async () => {
        await plugin.onunload();
        vi.clearAllMocks();
    });

    /**
     * 需求 1: AI 聊天功能整合測試
     */
    describe('AI Chat Integration (Requirements 1.1-1.4)', () => {
        it('should provide complete AI chat functionality', async () => {
            const aiManager = plugin.aiManager;
            
            // 測試聊天視窗創建
            const chatView = aiManager.createChatView();
            expect(chatView).toBeDefined();
            
            // 測試 AI 訊息發送
            const mockResponse = {
                message: 'Hello! How can I help you?',
                suggestions: [],
                actions: [],
                metadata: { timestamp: new Date(), model: 'gpt-4' }
            };
            
            vi.spyOn(apiClient, 'chatWithAI').mockResolvedValue(mockResponse);
            
            const response = await aiManager.sendMessage('Hello');
            expect(response).toEqual(mockResponse);
            expect(apiClient.chatWithAI).toHaveBeenCalledWith('Hello', undefined);
            
            // 測試聊天歷史維護
            aiManager.maintainChatHistory();
            expect(chatView.getHistory()).toContain('Hello');
        });

        it('should handle AI content processing', async () => {
            const content = '# Test Note\n\nThis is a test note with some content.';
            const mockResult = {
                chunks: [{
                    chunkId: 'chunk-1',
                    contents: 'This is a test note with some content.',
                    documentId: 'doc-1',
                    filePath: 'test-note.md',
                    tags: [],
                    metadata: {},
                    createdTime: new Date(),
                    lastUpdated: new Date(),
                    isPage: false,
                    isTag: false,
                    isTemplate: false,
                    isSlot: false,
                    documentScope: 'file' as const,
                    position: { fileName: 'test-note.md', lineStart: 1, lineEnd: 1, charStart: 0, charEnd: 50 }
                }],
                suggestions: [],
                improvements: []
            };
            
            vi.spyOn(apiClient, 'processContent').mockResolvedValue(mockResult);
            
            const result = await plugin.aiManager.processContent(content);
            expect(result).toEqual(mockResult);
        });
    });

    /**
     * 需求 2: 自動內容處理整合測試
     */
    describe('Auto Content Processing Integration (Requirements 2.1-2.5)', () => {
        it('should automatically process content on Enter key', async () => {
            const content = '# Test Heading\n\nTest paragraph content.\n\n- Bullet point 1\n- Bullet point 2';
            
            // 模擬檔案內容
            vi.spyOn(mockVault, 'read').mockResolvedValue(content);
            
            // 模擬 API 回應
            const mockChunks: UnifiedChunk[] = [
                {
                    chunkId: 'chunk-1',
                    contents: 'Test Heading',
                    documentId: 'doc-1',
                    filePath: 'test-note.md',
                    tags: [],
                    metadata: {},
                    createdTime: new Date(),
                    lastUpdated: new Date(),
                    isPage: false,
                    isTag: false,
                    isTemplate: false,
                    isSlot: false,
                    documentScope: 'file' as const,
                    position: { fileName: 'test-note.md', lineStart: 1, lineEnd: 1, charStart: 0, charEnd: 12 }
                }
            ];
            
            vi.spyOn(apiClient, 'batchCreateChunks').mockResolvedValue(mockChunks);
            
            // 觸發內容變更處理
            await plugin.contentManager.handleContentChange(mockFile);
            
            expect(apiClient.batchCreateChunks).toHaveBeenCalled();
        });

        it('should handle sync failures with retry mechanism', async () => {
            const content = 'Test content';
            vi.spyOn(mockVault, 'read').mockResolvedValue(content);
            
            // 第一次失敗，第二次成功
            vi.spyOn(apiClient, 'batchCreateChunks')
                .mockRejectedValueOnce(new Error('Network error'))
                .mockResolvedValueOnce([]);
            
            await plugin.contentManager.handleContentChange(mockFile);
            
            // 應該重試
            expect(apiClient.batchCreateChunks).toHaveBeenCalledTimes(2);
        });
    });

    /**
     * 需求 3: 語義搜尋整合測試
     */
    describe('Semantic Search Integration (Requirements 3.1-3.5)', () => {
        it('should perform comprehensive search operations', async () => {
            const searchManager = plugin.searchManager;
            
            const mockSearchResult = {
                items: [{
                    chunk: {
                        chunkId: 'chunk-1',
                        contents: 'Test content',
                        documentId: 'doc-1',
                        filePath: 'test.md',
                        tags: ['test'],
                        metadata: {},
                        createdTime: new Date(),
                        lastUpdated: new Date(),
                        isPage: false,
                        isTag: false,
                        isTemplate: false,
                        isSlot: false,
                        documentScope: 'file' as const,
                        position: { fileName: 'test.md', lineStart: 1, lineEnd: 1, charStart: 0, charEnd: 12 }
                    },
                    score: 0.95,
                    context: 'Test content context',
                    position: { fileName: 'test.md', lineStart: 1, lineEnd: 1, charStart: 0, charEnd: 12 },
                    highlights: []
                }],
                totalCount: 1,
                searchTime: 100,
                cacheHit: false
            };
            
            // 測試語義搜尋
            vi.spyOn(apiClient, 'searchSemantic').mockResolvedValue(mockSearchResult);
            
            const semanticQuery: SearchQuery = {
                content: 'test query',
                searchType: 'semantic'
            };
            
            const result = await searchManager.performSearch(semanticQuery);
            expect(result).toEqual(mockSearchResult);
            
            // 測試標籤搜尋
            vi.spyOn(apiClient, 'searchByTags').mockResolvedValue(mockSearchResult);
            
            const tagQuery: SearchQuery = {
                tags: ['test'],
                tagLogic: 'AND',
                searchType: 'exact'
            };
            
            const tagResult = await searchManager.performSearch(tagQuery);
            expect(tagResult).toEqual(mockSearchResult);
        });

        it('should navigate to search results', async () => {
            const searchManager = plugin.searchManager;
            const mockWorkspace = mockApp.workspace;
            
            const resultItem = {
                chunk: {
                    chunkId: 'chunk-1',
                    contents: 'Test content',
                    documentId: 'doc-1',
                    filePath: 'test.md',
                    tags: [],
                    metadata: {},
                    createdTime: new Date(),
                    lastUpdated: new Date(),
                    isPage: false,
                    isTag: false,
                    isTemplate: false,
                    isSlot: false,
                    documentScope: 'file' as const,
                    position: { fileName: 'test.md', lineStart: 5, lineEnd: 5, charStart: 0, charEnd: 12 }
                },
                score: 0.95,
                context: 'Test context',
                position: { fileName: 'test.md', lineStart: 5, lineEnd: 5, charStart: 0, charEnd: 12 },
                highlights: []
            };
            
            vi.spyOn(mockWorkspace, 'openLinkText');
            
            await searchManager.navigateToResult(resultItem);
            
            expect(mockWorkspace.openLinkText).toHaveBeenCalledWith(
                'test.md',
                '',
                false,
                { line: 5 }
            );
        });
    });

    /**
     * 需求 4: 模板系統整合測試
     */
    describe('Template System Integration (Requirements 4.1-4.6)', () => {
        it('should create and apply templates with Obsidian properties', async () => {
            const templateManager = plugin.templateManager;
            
            const template: Template = {
                id: 'template-1',
                name: 'Contact Template',
                slots: [
                    { id: 'name', name: 'Name', type: 'text', required: true },
                    { id: 'email', name: 'Email', type: 'text', required: false }
                ],
                structure: {
                    layout: 'vertical',
                    sections: []
                },
                metadata: {
                    createdAt: new Date(),
                    updatedAt: new Date(),
                    version: '1.0'
                }
            };
            
            vi.spyOn(apiClient, 'createTemplate').mockResolvedValue(template);
            
            const createdTemplate = await templateManager.createTemplate(
                'Contact Template',
                template.structure
            );
            
            expect(createdTemplate).toEqual(template);
            
            // 測試模板應用
            const templateContent = `---
name: John Doe
email: john@example.com
---

# Contact: {{name}}

Email: {{email}}`;
            
            vi.spyOn(mockVault, 'modify').mockResolvedValue();
            
            await templateManager.applyTemplate('template-1', mockFile);
            
            expect(mockVault.modify).toHaveBeenCalled();
        });
    });

    /**
     * 需求 5-6: 階層解析和位置追蹤整合測試
     */
    describe('Hierarchy and Position Tracking Integration (Requirements 5.1-6.4)', () => {
        it('should parse hierarchical content with position tracking', async () => {
            const content = `# Main Heading

## Sub Heading 1

Content under sub heading 1.

- Bullet point 1
  - Nested bullet 1
  - Nested bullet 2
- Bullet point 2

## Sub Heading 2

Content under sub heading 2.`;

            vi.spyOn(mockVault, 'read').mockResolvedValue(content);
            
            const parsedContent = await plugin.contentManager.parseContent(content, 'test.md');
            
            // 驗證階層結構
            expect(parsedContent.hierarchy).toBeDefined();
            expect(parsedContent.hierarchy.length).toBeGreaterThan(0);
            
            // 驗證位置追蹤
            parsedContent.chunks.forEach(chunk => {
                expect(chunk.position).toBeDefined();
                expect(chunk.position.fileName).toBe('test.md');
                expect(chunk.position.lineStart).toBeGreaterThan(0);
            });
            
            // 驗證父子關係
            const hierarchyNodes = parsedContent.hierarchy;
            const mainHeading = hierarchyNodes.find(node => node.content.includes('Main Heading'));
            expect(mainHeading?.children.length).toBeGreaterThan(0);
        });
    });

    /**
     * 需求 7: 解耦架構整合測試
     */
    describe('Decoupled Architecture Integration (Requirements 7.1-7.5)', () => {
        it('should handle API unavailability gracefully', async () => {
            // 模擬 API 不可用
            vi.spyOn(apiClient, 'searchChunks').mockRejectedValue(new Error('Service unavailable'));
            
            const searchQuery: SearchQuery = {
                content: 'test',
                searchType: 'semantic'
            };
            
            // 應該優雅處理錯誤
            await expect(plugin.searchManager.performSearch(searchQuery)).rejects.toThrow();
            
            // 驗證錯誤處理
            expect(plugin.errorHandler.getLastError()).toBeDefined();
        });

        it('should maintain clean API interface', () => {
            // 驗證 API 客戶端介面
            expect(typeof apiClient.createChunk).toBe('function');
            expect(typeof apiClient.searchChunks).toBe('function');
            expect(typeof apiClient.chatWithAI).toBe('function');
            expect(typeof apiClient.createTemplate).toBe('function');
        });
    });

    /**
     * 需求 8: 標籤和元資料整合測試
     */
    describe('Tags and Metadata Integration (Requirements 8.1-8.5)', () => {
        it('should sync tags and metadata bidirectionally', async () => {
            const contentWithTags = `---
tags: [project, important]
category: work
---

# Project Notes

This is a #tagged note with #multiple-tags.`;

            vi.spyOn(mockVault, 'read').mockResolvedValue(contentWithTags);
            
            const parsedContent = await plugin.contentManager.parseContent(contentWithTags, 'test.md');
            
            // 驗證標籤提取
            const chunk = parsedContent.chunks[0];
            expect(chunk.tags).toContain('project');
            expect(chunk.tags).toContain('important');
            expect(chunk.tags).toContain('tagged');
            expect(chunk.tags).toContain('multiple-tags');
            
            // 驗證元資料
            expect(chunk.metadata.category).toBe('work');
        });
    });

    /**
     * 需求 9: 即時同步整合測試
     */
    describe('Real-time Sync Integration (Requirements 9.1-9.5)', () => {
        it('should handle offline mode and sync when online', async () => {
            const offlineManager = plugin.offlineManager;
            
            // 模擬離線狀態
            vi.spyOn(offlineManager, 'isOnline').mockReturnValue(false);
            
            const content = 'Offline content';
            vi.spyOn(mockVault, 'read').mockResolvedValue(content);
            
            // 觸發內容變更（應該排隊）
            await plugin.contentManager.handleContentChange(mockFile);
            
            // 驗證操作被排隊
            expect(offlineManager.getPendingOperations().length).toBeGreaterThan(0);
            
            // 模擬上線
            vi.spyOn(offlineManager, 'isOnline').mockReturnValue(true);
            vi.spyOn(apiClient, 'batchCreateChunks').mockResolvedValue([]);
            
            // 觸發同步
            await offlineManager.syncWhenOnline();
            
            // 驗證同步完成
            expect(offlineManager.getPendingOperations().length).toBe(0);
        });
    });

    /**
     * 需求 10: 文件 ID 分頁整合測試
     */
    describe('Document ID Pagination Integration (Requirements 10.1-10.7)', () => {
        it('should manage document IDs and pagination', async () => {
            const documentId = 'doc-123';
            const mockChunks: UnifiedChunk[] = [
                {
                    chunkId: 'chunk-1',
                    contents: 'First chunk',
                    documentId,
                    filePath: 'test.md',
                    tags: [],
                    metadata: {},
                    createdTime: new Date(),
                    lastUpdated: new Date(),
                    isPage: false,
                    isTag: false,
                    isTemplate: false,
                    isSlot: false,
                    documentScope: 'file' as const,
                    position: { fileName: 'test.md', lineStart: 1, lineEnd: 1, charStart: 0, charEnd: 11 }
                }
            ];
            
            const mockResult = {
                chunks: mockChunks,
                pagination: {
                    currentPage: 1,
                    totalPages: 1,
                    totalChunks: 1,
                    pageSize: 10
                },
                documentMetadata: {
                    originalFilePath: 'test.md',
                    totalChunks: 1,
                    documentScope: 'file' as const,
                    lastModified: new Date()
                }
            };
            
            vi.spyOn(apiClient, 'getChunksByDocumentId').mockResolvedValue(mockResult);
            
            const result = await plugin.contentManager.getChunksByDocumentId(documentId);
            expect(result).toEqual(mockResult);
            
            // 測試文件重建
            const reconstructed = await plugin.contentManager.reconstructDocument(documentId);
            expect(reconstructed.chunks).toEqual(mockChunks);
        });
    });
});