/**
 * End-to-End Integration Tests
 * Tests complete workflows from user interaction to Ink-Gateway integration
 */

import { App, TFile, Vault } from 'obsidian';
import ObsidianInkPlugin from '../../src/main';
import { InkGatewayClient } from '../../src/api/InkGatewayClient';
import { ContentManager } from '../../src/content/ContentManager';
import { SearchManager } from '../../src/search/SearchManager';
import { AIManager } from '../../src/ai/AIManager';
import { TemplateManager } from '../../src/template/TemplateManager';

// Mock Obsidian environment
jest.mock('obsidian');

describe('End-to-End Integration Tests', () => {
    let plugin: ObsidianInkPlugin;
    let mockApp: jest.Mocked<App>;
    let mockVault: jest.Mocked<Vault>;
    let mockApiClient: jest.Mocked<InkGatewayClient>;

    beforeEach(async () => {
        // Setup mock Obsidian environment
        mockVault = {
            read: jest.fn(),
            modify: jest.fn(),
            create: jest.fn(),
            delete: jest.fn(),
            getFiles: jest.fn(),
            getMarkdownFiles: jest.fn(),
            on: jest.fn(),
            off: jest.fn(),
        } as any;

        mockApp = {
            vault: mockVault,
            workspace: {
                on: jest.fn(),
                off: jest.fn(),
                getActiveFile: jest.fn(),
                getLeaf: jest.fn(),
            },
            metadataCache: {
                on: jest.fn(),
                off: jest.fn(),
                getFileCache: jest.fn(),
            },
        } as any;

        // Setup mock API client
        mockApiClient = {
            createChunk: jest.fn(),
            updateChunk: jest.fn(),
            deleteChunk: jest.fn(),
            searchChunks: jest.fn(),
            chatWithAI: jest.fn(),
            createTemplate: jest.fn(),
        } as any;

        // Initialize plugin
        plugin = new ObsidianInkPlugin(mockApp, {} as any);
        plugin.apiClient = mockApiClient;
        
        await plugin.onload();
    });

    afterEach(async () => {
        await plugin.onunload();
    });

    describe('Complete Content Processing Workflow', () => {
        it('should process content from creation to search', async () => {
            // Requirement 2.1-2.5: Auto content processing
            const testContent = `# Test Document

This is a test paragraph.

## Section 1
- Item 1
- Item 2
  - Nested item

#tag1 #tag2`;

            const mockFile = {
                path: 'test.md',
                name: 'test.md',
                basename: 'test',
                extension: 'md',
            } as TFile;

            mockVault.read.mockResolvedValue(testContent);
            mockApiClient.createChunk.mockResolvedValue({
                chunkId: 'chunk-1',
                contents: 'Test content',
                documentId: 'doc-1',
                tags: ['tag1', 'tag2'],
                createdTime: new Date(),
                lastUpdated: new Date(),
            } as any);

            // Simulate content change (user presses Enter)
            await plugin.contentManager.handleContentChange(mockFile);

            // Verify content was processed and sent to API
            expect(mockApiClient.createChunk).toHaveBeenCalled();
            
            // Verify chunks were created with proper hierarchy
            const createCalls = mockApiClient.createChunk.mock.calls;
            expect(createCalls.length).toBeGreaterThan(0);
            
            // Check that hierarchy relationships were established
            const chunks = createCalls.map(call => call[0]);
            const headingChunk = chunks.find(chunk => chunk.contents.includes('Test Document'));
            const sectionChunk = chunks.find(chunk => chunk.contents.includes('Section 1'));
            
            expect(headingChunk).toBeDefined();
            expect(sectionChunk).toBeDefined();
            expect(sectionChunk?.parent).toBe(headingChunk?.chunkId);
        });

        it('should handle template creation and application workflow', async () => {
            // Requirement 4.1-4.6: Template system
            const templateContent = `# Contact Template

**Name:** {{name}}
**Email:** {{email}}
**Phone:** {{phone}}
**Notes:** {{notes}}

#contact #template`;

            const mockTemplateFile = {
                path: 'templates/contact.md',
                name: 'contact.md',
            } as TFile;

            mockVault.read.mockResolvedValue(templateContent);
            mockApiClient.createTemplate.mockResolvedValue({
                id: 'template-1',
                name: 'Contact Template',
                slots: [
                    { id: 'name', name: 'name', type: 'text', required: true },
                    { id: 'email', name: 'email', type: 'text', required: true },
                    { id: 'phone', name: 'phone', type: 'text', required: false },
                    { id: 'notes', name: 'notes', type: 'text', required: false },
                ],
            } as any);

            // Create template
            const template = await plugin.templateManager.parseTemplateFromContent(templateContent);
            expect(template.slots).toHaveLength(4);
            expect(template.slots[0].name).toBe('name');

            // Apply template
            const newContactFile = {
                path: 'contacts/john-doe.md',
                name: 'john-doe.md',
            } as TFile;

            await plugin.templateManager.applyTemplate(template.id, newContactFile);
            
            // Verify template was applied and content created
            expect(mockVault.create).toHaveBeenCalled();
        });
    });

    describe('Search Integration Workflow', () => {
        it('should perform semantic search and navigate to results', async () => {
            // Requirement 3.1-3.5: Semantic search
            const searchQuery = {
                content: 'artificial intelligence',
                searchType: 'semantic' as const,
            };

            const mockSearchResults = {
                items: [
                    {
                        chunk: {
                            chunkId: 'chunk-1',
                            contents: 'AI and machine learning concepts',
                            position: {
                                fileName: 'ai-notes.md',
                                lineStart: 5,
                                lineEnd: 7,
                            },
                        },
                        score: 0.95,
                        context: 'This section discusses AI concepts...',
                    },
                ],
                totalCount: 1,
                searchTime: 150,
            };

            mockApiClient.searchChunks.mockResolvedValue(mockSearchResults);

            // Perform search
            const results = await plugin.searchManager.performSearch(searchQuery);
            
            expect(results.items).toHaveLength(1);
            expect(results.items[0].score).toBe(0.95);
            expect(mockApiClient.searchChunks).toHaveBeenCalledWith(searchQuery);

            // Test navigation to result
            const mockWorkspaceLeaf = {
                openFile: jest.fn(),
            };
            mockApp.workspace.getLeaf = jest.fn().mockReturnValue(mockWorkspaceLeaf);

            await plugin.searchManager.navigateToResult(results.items[0]);
            expect(mockWorkspaceLeaf.openFile).toHaveBeenCalled();
        });
    });

    describe('AI Chat Integration Workflow', () => {
        it('should handle AI chat conversation with context', async () => {
            // Requirement 1.1-1.4: AI chat functionality
            const chatMessage = 'Summarize my notes about machine learning';
            
            const mockAIResponse = {
                message: 'Based on your notes, machine learning involves...',
                suggestions: [
                    { type: 'link', content: 'Related: Deep Learning Basics' },
                ],
                metadata: {
                    responseTime: 1200,
                    tokensUsed: 150,
                },
            };

            mockApiClient.chatWithAI.mockResolvedValue(mockAIResponse);

            // Send message
            const response = await plugin.aiManager.sendMessage(chatMessage);
            
            expect(response.message).toContain('machine learning involves');
            expect(response.suggestions).toHaveLength(1);
            expect(mockApiClient.chatWithAI).toHaveBeenCalledWith(
                chatMessage,
                expect.any(Array) // context chunks
            );
        });
    });

    describe('Offline Mode Integration', () => {
        it('should queue operations when offline and sync when online', async () => {
            // Requirement 9.5: Offline support
            const testContent = 'New content created while offline';
            const mockFile = {
                path: 'offline-test.md',
                name: 'offline-test.md',
            } as TFile;

            // Simulate offline mode
            plugin.offlineManager.isOnline = jest.fn().mockReturnValue(false);
            
            // Attempt to sync content while offline
            await plugin.contentManager.handleContentChange(mockFile);
            
            // Verify operation was queued, not sent immediately
            expect(mockApiClient.createChunk).not.toHaveBeenCalled();
            expect(plugin.offlineManager.pendingOperations).toHaveLength(1);

            // Simulate coming back online
            plugin.offlineManager.isOnline = jest.fn().mockReturnValue(true);
            mockApiClient.createChunk.mockResolvedValue({
                chunkId: 'chunk-offline-1',
                contents: testContent,
            } as any);

            // Trigger sync
            await plugin.offlineManager.syncWhenOnline();
            
            // Verify queued operations were processed
            expect(mockApiClient.createChunk).toHaveBeenCalled();
            expect(plugin.offlineManager.pendingOperations).toHaveLength(0);
        });
    });

    describe('Document ID Pagination Workflow', () => {
        it('should manage document IDs and retrieve chunks by document', async () => {
            // Requirement 10.1-10.7: Document ID pagination
            const documentId = 'doc-test-123';
            const mockChunks = [
                {
                    chunkId: 'chunk-1',
                    contents: 'First paragraph',
                    documentId,
                    position: { lineStart: 1, lineEnd: 1 },
                },
                {
                    chunkId: 'chunk-2',
                    contents: 'Second paragraph',
                    documentId,
                    position: { lineStart: 3, lineEnd: 3 },
                },
            ];

            mockApiClient.getChunksByDocumentId = jest.fn().mockResolvedValue({
                chunks: mockChunks,
                pagination: {
                    currentPage: 1,
                    totalPages: 1,
                    totalChunks: 2,
                    pageSize: 10,
                },
                documentMetadata: {
                    documentScope: 'file',
                    totalChunks: 2,
                },
            });

            // Retrieve chunks by document ID
            const result = await plugin.apiClient.getChunksByDocumentId(documentId);
            
            expect(result.chunks).toHaveLength(2);
            expect(result.chunks[0].documentId).toBe(documentId);
            expect(result.pagination.totalChunks).toBe(2);

            // Test document reconstruction
            const reconstructed = await plugin.contentManager.reconstructDocument(documentId);
            expect(reconstructed.chunks).toHaveLength(2);
            expect(reconstructed.documentId).toBe(documentId);
        });
    });

    describe('Error Handling and Recovery', () => {
        it('should handle API errors gracefully and retry', async () => {
            // Requirement 7.5: Error handling
            const testContent = 'Test content for error handling';
            const mockFile = { path: 'error-test.md' } as TFile;

            // Simulate API error on first attempt
            mockApiClient.createChunk
                .mockRejectedValueOnce(new Error('Network error'))
                .mockResolvedValueOnce({
                    chunkId: 'chunk-retry-1',
                    contents: testContent,
                } as any);

            // Attempt content sync
            await plugin.contentManager.handleContentChange(mockFile);

            // Verify retry mechanism worked
            expect(mockApiClient.createChunk).toHaveBeenCalledTimes(2);
        });
    });

    describe('Performance and Caching', () => {
        it('should cache search results and API responses', async () => {
            // Test search result caching
            const searchQuery = {
                content: 'test query',
                searchType: 'semantic' as const,
            };

            const mockResults = {
                items: [{ chunk: { chunkId: 'cached-1' } }],
                totalCount: 1,
                searchTime: 100,
                cacheHit: false,
            };

            mockApiClient.searchChunks.mockResolvedValue(mockResults);

            // First search - should hit API
            const firstResult = await plugin.searchManager.performSearch(searchQuery);
            expect(firstResult.cacheHit).toBe(false);
            expect(mockApiClient.searchChunks).toHaveBeenCalledTimes(1);

            // Second search - should hit cache
            const secondResult = await plugin.searchManager.performSearch(searchQuery);
            expect(secondResult.cacheHit).toBe(true);
            expect(mockApiClient.searchChunks).toHaveBeenCalledTimes(1); // No additional API call
        });
    });
});