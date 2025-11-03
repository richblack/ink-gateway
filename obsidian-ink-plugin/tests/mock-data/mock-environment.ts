/**
 * Mock Environment Setup
 * Provides mock implementations for Obsidian API and external services
 */

import { App, TFile, Vault, Workspace, MetadataCache, Plugin } from 'obsidian';
import { InkGatewayClient } from '../../src/api/InkGatewayClient';
import { TestDataGenerator } from './test-data-generator';

export class MockEnvironment {
    public mockApp!: jest.Mocked<App>;
    public mockVault!: jest.Mocked<Vault>;
    public mockWorkspace!: jest.Mocked<Workspace>;
    public mockMetadataCache!: jest.Mocked<MetadataCache>;
    public mockApiClient!: jest.Mocked<InkGatewayClient>;
    public mockFiles: Map<string, MockFile> = new Map();

    constructor() {
        this.setupMockVault();
        this.setupMockWorkspace();
        this.setupMockMetadataCache();
        this.setupMockApp();
        this.setupMockApiClient();
    }

    private setupMockVault(): void {
        this.mockVault = {
            // File operations
            read: jest.fn().mockImplementation((file: TFile) => {
                const mockFile = this.mockFiles.get(file.path);
                return Promise.resolve(mockFile?.content || '');
            }),
            
            modify: jest.fn().mockImplementation((file: TFile, content: string) => {
                const mockFile = this.mockFiles.get(file.path);
                if (mockFile) {
                    mockFile.content = content;
                    mockFile.stat.mtime = Date.now();
                }
                return Promise.resolve();
            }),
            
            create: jest.fn().mockImplementation((path: string, content: string) => {
                const mockFile = new MockFile(path, content);
                this.mockFiles.set(path, mockFile);
                return Promise.resolve(mockFile.file);
            }),
            
            delete: jest.fn().mockImplementation((file: TFile) => {
                this.mockFiles.delete(file.path);
                return Promise.resolve();
            }),
            
            rename: jest.fn().mockImplementation((file: TFile, newPath: string) => {
                const mockFile = this.mockFiles.get(file.path);
                if (mockFile) {
                    this.mockFiles.delete(file.path);
                    mockFile.file.path = newPath;
                    mockFile.file.name = newPath.split('/').pop() || newPath;
                    this.mockFiles.set(newPath, mockFile);
                }
                return Promise.resolve();
            }),
            
            // File listing
            getFiles: jest.fn().mockImplementation(() => {
                return Array.from(this.mockFiles.values()).map(f => f.file);
            }),
            
            getMarkdownFiles: jest.fn().mockImplementation(() => {
                return Array.from(this.mockFiles.values())
                    .filter(f => f.file.extension === 'md')
                    .map(f => f.file);
            }),
            
            getAbstractFileByPath: jest.fn().mockImplementation((path: string) => {
                return this.mockFiles.get(path)?.file || null;
            }),
            
            // Event handling
            on: jest.fn(),
            off: jest.fn(),
            trigger: jest.fn(),
            
            // Other properties
            adapter: {} as any,
            configDir: '.obsidian',
        } as any;
    }

    private setupMockWorkspace(): void {
        this.mockWorkspace = {
            // Active file management
            getActiveFile: jest.fn().mockReturnValue(null),
            
            // Leaf management
            getLeaf: jest.fn().mockImplementation(() => ({
                openFile: jest.fn().mockResolvedValue(undefined),
                setViewState: jest.fn().mockResolvedValue(undefined),
                view: null,
            })),
            
            getUnpinnedLeaf: jest.fn().mockImplementation(() => this.mockWorkspace.getLeaf()),
            
            // Layout
            leftSplit: {
                children: [],
                getRoot: jest.fn(),
            } as any,
            
            rightSplit: {
                children: [],
                getRoot: jest.fn(),
            } as any,
            
            rootSplit: {
                children: [],
                getRoot: jest.fn(),
            } as any,
            
            // Event handling
            on: jest.fn(),
            off: jest.fn(),
            trigger: jest.fn(),
            
            // View management
            detachLeavesOfType: jest.fn(),
            revealLeaf: jest.fn(),
            
            // Other properties
            layoutReady: true,
        } as any;
    }

    private setupMockMetadataCache(): void {
        this.mockMetadataCache = {
            // File cache
            getFileCache: jest.fn().mockImplementation((file: TFile) => {
                const mockFile = this.mockFiles.get(file.path);
                return mockFile?.cache || null;
            }),
            
            getCache: jest.fn().mockImplementation((path: string) => {
                const mockFile = this.mockFiles.get(path);
                return mockFile?.cache || null;
            }),
            
            // Event handling
            on: jest.fn(),
            off: jest.fn(),
            trigger: jest.fn(),
            
            // Other methods
            resolvedLinks: {},
            unresolvedLinks: {},
        } as any;
    }

    private setupMockApp(): void {
        this.mockApp = {
            vault: this.mockVault,
            workspace: this.mockWorkspace,
            metadataCache: this.mockMetadataCache,
            
            // Plugin management
            plugins: {
                plugins: {},
                enablePlugin: jest.fn(),
                disablePlugin: jest.fn(),
            } as any,
            
            // Settings
            setting: {
                open: jest.fn(),
                close: jest.fn(),
            } as any,
            
            // Other properties
            lastOpenFiles: [],
        } as any;
    }

    private setupMockApiClient(): void {
        this.mockApiClient = {
            // Chunk operations
            createChunk: jest.fn().mockImplementation((chunk) => {
                return Promise.resolve({
                    ...chunk,
                    chunkId: chunk.chunkId || `mock-chunk-${Date.now()}`,
                    createdTime: new Date(),
                    lastUpdated: new Date(),
                });
            }),
            
            updateChunk: jest.fn().mockImplementation((id, updates) => {
                return Promise.resolve({
                    chunkId: id,
                    ...updates,
                    lastUpdated: new Date(),
                });
            }),
            
            deleteChunk: jest.fn().mockResolvedValue(undefined),
            
            getChunk: jest.fn().mockImplementation((id) => {
                return Promise.resolve(TestDataGenerator.generateChunk({ chunkId: id }));
            }),
            
            batchCreateChunks: jest.fn().mockImplementation((chunks) => {
                return Promise.resolve(chunks.map(chunk => ({
                    ...chunk,
                    chunkId: chunk.chunkId || `mock-chunk-${Date.now()}-${Math.random()}`,
                    createdTime: new Date(),
                    lastUpdated: new Date(),
                })));
            }),
            
            // Search operations
            searchChunks: jest.fn().mockImplementation((query) => {
                const results = TestDataGenerator.generateSearchResults(10, query.content || 'test');
                return Promise.resolve({
                    items: results,
                    totalCount: results.length,
                    searchTime: Math.random() * 200 + 50, // 50-250ms
                    cacheHit: false,
                });
            }),
            
            searchSemantic: jest.fn().mockImplementation((content) => {
                const results = TestDataGenerator.generateSearchResults(5, content);
                return Promise.resolve({
                    items: results,
                    totalCount: results.length,
                    searchTime: Math.random() * 300 + 100, // 100-400ms
                    cacheHit: false,
                });
            }),
            
            searchByTags: jest.fn().mockImplementation((tags) => {
                const results = TestDataGenerator.generateSearchResults(8, tags[0] || 'tag');
                return Promise.resolve({
                    items: results,
                    totalCount: results.length,
                    searchTime: Math.random() * 150 + 30, // 30-180ms
                    cacheHit: false,
                });
            }),
            
            // AI operations
            chatWithAI: jest.fn().mockImplementation((message, context) => {
                return Promise.resolve(TestDataGenerator.generateAIResponse({
                    message: `AI response to: ${message}`,
                }));
            }),
            
            processContent: jest.fn().mockImplementation((content) => {
                const chunks = [TestDataGenerator.generateChunk({ contents: content })];
                return Promise.resolve({
                    chunks,
                    suggestions: [],
                    improvements: [],
                });
            }),
            
            // Template operations
            createTemplate: jest.fn().mockImplementation((template) => {
                return Promise.resolve({
                    ...template,
                    id: template.id || `mock-template-${Date.now()}`,
                });
            }),
            
            getTemplateInstances: jest.fn().mockImplementation((templateId) => {
                return Promise.resolve([
                    {
                        id: `instance-1-${templateId}`,
                        templateId,
                        filePath: 'test-instance.md',
                        slotValues: { title: 'Test Instance' },
                        createdAt: new Date(),
                        updatedAt: new Date(),
                    },
                ]);
            }),
            
            // Document ID operations
            getChunksByDocumentId: jest.fn().mockImplementation((documentId, options) => {
                const chunks = TestDataGenerator.generateChunkHierarchy(3, 3)
                    .map(chunk => ({ ...chunk, documentId }));
                
                return Promise.resolve({
                    chunks,
                    pagination: {
                        currentPage: options?.page || 1,
                        totalPages: 1,
                        totalChunks: chunks.length,
                        pageSize: options?.pageSize || 50,
                    },
                    documentMetadata: {
                        documentScope: 'file' as const,
                        totalChunks: chunks.length,
                        lastModified: new Date(),
                    },
                });
            }),
            
            createVirtualDocument: jest.fn().mockImplementation((context) => {
                return Promise.resolve({
                    virtualDocumentId: `virtual-${Date.now()}`,
                    context,
                    chunkIds: [],
                    createdAt: new Date(),
                    lastUpdated: new Date(),
                });
            }),
            
            updateDocumentScope: jest.fn().mockResolvedValue(undefined),
            
            // Hierarchy operations
            getHierarchy: jest.fn().mockImplementation((rootId) => {
                return Promise.resolve(TestDataGenerator.generateChunkHierarchy(3, 3));
            }),
            
            updateHierarchy: jest.fn().mockResolvedValue(undefined),
        } as any;
    }

    /**
     * Add a mock file to the environment
     */
    addFile(path: string, content: string, metadata: any = {}): MockFile {
        const mockFile = new MockFile(path, content, metadata);
        this.mockFiles.set(path, mockFile);
        return mockFile;
    }

    /**
     * Remove a mock file from the environment
     */
    removeFile(path: string): void {
        this.mockFiles.delete(path);
    }

    /**
     * Get a mock file
     */
    getFile(path: string): MockFile | undefined {
        return this.mockFiles.get(path);
    }

    /**
     * Clear all mock files
     */
    clearFiles(): void {
        this.mockFiles.clear();
    }

    /**
     * Setup network conditions (for offline testing)
     */
    setNetworkCondition(online: boolean): void {
        if (!online) {
            // Make API calls fail with network errors
            Object.keys(this.mockApiClient).forEach(method => {
                if (typeof this.mockApiClient[method as keyof InkGatewayClient] === 'function') {
                    (this.mockApiClient[method as keyof InkGatewayClient] as jest.Mock)
                        .mockRejectedValue(new Error('Network error: offline'));
                }
            });
        } else {
            // Restore normal API behavior
            this.setupMockApiClient();
        }
    }

    /**
     * Simulate API latency
     */
    setApiLatency(minMs: number, maxMs: number): void {
        const originalMethods = { ...this.mockApiClient };
        
        Object.keys(this.mockApiClient).forEach(method => {
            if (typeof this.mockApiClient[method as keyof InkGatewayClient] === 'function') {
                (this.mockApiClient[method as keyof InkGatewayClient] as jest.Mock)
                    .mockImplementation(async (...args: any[]) => {
                        const delay = Math.random() * (maxMs - minMs) + minMs;
                        await new Promise(resolve => setTimeout(resolve, delay));
                        return (originalMethods[method as keyof InkGatewayClient] as any)(...args);
                    });
            }
        });
    }

    /**
     * Reset all mocks
     */
    reset(): void {
        this.clearFiles();
        this.setupMockApiClient();
        jest.clearAllMocks();
    }
}

/**
 * Mock file implementation
 */
export class MockFile {
    public file: TFile;
    public content: string;
    public cache: any;
    public stat: { mtime: number; ctime: number; size: number };

    constructor(path: string, content: string, metadata: any = {}) {
        this.content = content;
        this.stat = {
            mtime: Date.now(),
            ctime: Date.now(),
            size: content.length,
        };
        
        const pathParts = path.split('/');
        const name = pathParts[pathParts.length - 1];
        const extension = name.split('.').pop() || '';
        const basename = name.replace(`.${extension}`, '');
        
        this.file = {
            path,
            name,
            basename,
            extension,
            stat: this.stat,
            vault: null as any,
            parent: null,
        } as TFile;
        
        // Generate cache metadata
        this.cache = {
            headings: this.extractHeadings(content),
            tags: this.extractTags(content),
            links: this.extractLinks(content),
            frontmatter: this.extractFrontmatter(content),
            ...metadata,
        };
    }

    private extractHeadings(content: string): any[] {
        const headings: any[] = [];
        const lines = content.split('\n');
        
        lines.forEach((line, index) => {
            const match = line.match(/^(#{1,6})\s+(.+)$/);
            if (match) {
                headings.push({
                    heading: match[2],
                    level: match[1].length,
                    position: {
                        start: { line: index, col: 0, offset: 0 },
                        end: { line: index, col: line.length, offset: 0 },
                    },
                });
            }
        });
        
        return headings;
    }

    private extractTags(content: string): any[] {
        const tags: any[] = [];
        const tagRegex = /#[\w-]+/g;
        let match;
        
        while ((match = tagRegex.exec(content)) !== null) {
            tags.push({
                tag: match[0].substring(1), // Remove #
                position: {
                    start: { offset: match.index },
                    end: { offset: match.index + match[0].length },
                },
            });
        }
        
        return tags;
    }

    private extractLinks(content: string): any[] {
        const links: any[] = [];
        const linkRegex = /\[\[([^\]]+)\]\]/g;
        let match;
        
        while ((match = linkRegex.exec(content)) !== null) {
            links.push({
                link: match[1],
                original: match[0],
                position: {
                    start: { offset: match.index },
                    end: { offset: match.index + match[0].length },
                },
            });
        }
        
        return links;
    }

    private extractFrontmatter(content: string): any {
        const frontmatterRegex = /^---\n([\s\S]*?)\n---/;
        const match = content.match(frontmatterRegex);
        
        if (match) {
            try {
                // Simple YAML parsing for basic key-value pairs
                const frontmatter: any = {};
                const lines = match[1].split('\n');
                
                lines.forEach(line => {
                    const colonIndex = line.indexOf(':');
                    if (colonIndex > 0) {
                        const key = line.substring(0, colonIndex).trim();
                        const value = line.substring(colonIndex + 1).trim();
                        frontmatter[key] = value.replace(/^["']|["']$/g, ''); // Remove quotes
                    }
                });
                
                return frontmatter;
            } catch (error) {
                return {};
            }
        }
        
        return {};
    }
}

// Global mock environment instance for tests
export const mockEnvironment = new MockEnvironment();