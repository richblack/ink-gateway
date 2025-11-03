/**
 * Unit tests for ContentManager
 */

import { ContentManager } from '../ContentManager';
import { MarkdownParser } from '../MarkdownParser';
import { 
  IInkGatewayClient, 
  ILogger, 
  IEventManager, 
  ICacheManager,
  IOfflineManager
} from '../../interfaces';
import { 
  UnifiedChunk, 
  SyncResult, 
  ParsedContent,
  PluginError,
  ErrorType 
} from '../../types';

// Mock implementations
class MockInkGatewayClient implements IInkGatewayClient {
  private shouldFail = false;
  private isHealthy = true;
  
  setHealthy(healthy: boolean) { this.isHealthy = healthy; }
  setShouldFail(fail: boolean) { this.shouldFail = fail; }

  async healthCheck(): Promise<boolean> {
    return this.isHealthy;
  }

  async batchCreateChunks(chunks: UnifiedChunk[]): Promise<UnifiedChunk[]> {
    if (this.shouldFail) {
      throw new Error('API Error');
    }
    return chunks.map(chunk => ({ ...chunk, chunkId: `synced_${chunk.chunkId}` }));
  }

  // Other required methods (simplified for testing)
  async createChunk(chunk: UnifiedChunk): Promise<UnifiedChunk> { return chunk; }
  async updateChunk(id: string, chunk: Partial<UnifiedChunk>): Promise<UnifiedChunk> { return chunk as UnifiedChunk; }
  async deleteChunk(id: string): Promise<void> {}
  async getChunk(id: string): Promise<UnifiedChunk> { return {} as UnifiedChunk; }
  async searchChunks(query: any): Promise<any> { return { items: [], totalCount: 0, searchTime: 0, cacheHit: false }; }
  async searchSemantic(content: string): Promise<any> { return { items: [], totalCount: 0, searchTime: 0, cacheHit: false }; }
  async searchByTags(tags: string[]): Promise<any> { return { items: [], totalCount: 0, searchTime: 0, cacheHit: false }; }
  async getHierarchy(rootId: string): Promise<any[]> { return []; }
  async updateHierarchy(relations: any[]): Promise<void> {}
  async chatWithAI(message: string, context?: string[]): Promise<any> { return { message: 'Mock response', suggestions: [], actions: [], metadata: { model: 'mock', processingTime: 100, confidence: 0.9 } }; }
  async processContent(content: string): Promise<any> { return { chunks: [], suggestions: [], improvements: [] }; }
  async createTemplate(template: any): Promise<any> { return template; }
  async getTemplateInstances(templateId: string): Promise<any[]> { return []; }
  async getChunksByDocumentId(documentId: string, options?: any): Promise<any> { return { chunks: [], pagination: { currentPage: 1, totalPages: 1, totalChunks: 0, pageSize: 10 }, documentMetadata: { documentScope: 'file', totalChunks: 0, lastModified: new Date() } }; }
  async createVirtualDocument(context: any): Promise<any> { return { virtualDocumentId: 'mock-virtual-doc', context, chunkIds: [], createdAt: new Date(), lastUpdated: new Date() }; }
  async updateDocumentScope(chunkId: string, documentId: string, scope: any): Promise<void> {}
  async request<T = any>(config: any): Promise<any> { return { data: {} as T, status: 200, statusText: 'OK', headers: {} }; }
}

class MockLogger implements ILogger {
  debug = jest.fn();
  info = jest.fn();
  warn = jest.fn();
  error = jest.fn();
}

class MockEventManager implements IEventManager {
  private events = new Map<string, Function[]>();
  
  on(event: string, callback: Function): void {
    if (!this.events.has(event)) this.events.set(event, []);
    this.events.get(event)!.push(callback);
  }
  
  off(event: string, callback: Function): void {
    const callbacks = this.events.get(event);
    if (callbacks) {
      const index = callbacks.indexOf(callback);
      if (index > -1) callbacks.splice(index, 1);
    }
  }
  
  emit(event: string, ...args: any[]): void {
    const callbacks = this.events.get(event);
    if (callbacks) {
      callbacks.forEach(callback => callback(...args));
    }
  }
}

class MockCacheManager implements ICacheManager {
  private cache = new Map<string, { value: any; expires?: number }>();
  
  get<T>(key: string): T | null {
    const item = this.cache.get(key);
    if (!item) return null;
    if (item.expires && Date.now() > item.expires) {
      this.cache.delete(key);
      return null;
    }
    return item.value;
  }
  
  set<T>(key: string, value: T, ttl?: number): void {
    const expires = ttl ? Date.now() + ttl : undefined;
    this.cache.set(key, { value, expires });
  }
  
  delete(key: string): void {
    this.cache.delete(key);
  }
  
  clear(): void {
    this.cache.clear();
  }
  
  size(): number {
    return this.cache.size;
  }
}

class MockTFile {
  constructor(
    public path: string,
    public basename: string,
    public extension: string = 'md',
    public stat = { ctime: Date.now(), mtime: Date.now() }
  ) {}
}

class MockApp {
  vault = {
    read: jest.fn().mockResolvedValue('# Test Content\n\nThis is test content.'),
    getAbstractFileByPath: jest.fn()
  };
  
  metadataCache = {
    getFileCache: jest.fn().mockReturnValue({
      frontmatter: { title: 'Test', tags: ['test'] },
      tags: [{ tag: '#important' }],
      links: [{ link: 'other-file' }],
      embeds: []
    })
  };
}

// Mock MarkdownParser
jest.mock('../MarkdownParser', () => ({
  MarkdownParser: {
    parseContent: jest.fn(),
    validateParsedContent: jest.fn(() => true)
  }
}));

describe('ContentManager', () => {
  let contentManager: ContentManager;
  let mockApiClient: MockInkGatewayClient;
  let mockLogger: MockLogger;
  let mockEventManager: MockEventManager;
  let mockCacheManager: MockCacheManager;
  let mockApp: MockApp;

  beforeEach(() => {
    mockApiClient = new MockInkGatewayClient();
    mockLogger = new MockLogger();
    mockEventManager = new MockEventManager();
    mockCacheManager = new MockCacheManager();
    mockApp = new MockApp();

    const mockOfflineManager: IOfflineManager = {
      isOnline: () => true,
      queueOperation: jest.fn(),
      syncWhenOnline: jest.fn(),
      handleConflicts: jest.fn()
    };

    contentManager = new ContentManager(
      mockApiClient,
      mockLogger,
      mockEventManager,
      mockCacheManager,
      mockOfflineManager,
      mockApp
    );

    // Setup default mock return values
    (MarkdownParser.parseContent as jest.Mock).mockReturnValue({
      chunks: [
        {
          chunkId: 'test_chunk_1',
          contents: 'Test content',
          isPage: false,
          isTag: false,
          isTemplate: false,
          isSlot: false,
          tags: [],
          metadata: {},
          createdTime: new Date(),
          lastUpdated: new Date(),
          position: {
            fileName: 'test.md',
            lineStart: 1,
            lineEnd: 1,
            charStart: 0,
            charEnd: 10
          },
          filePath: 'test.md',
          obsidianMetadata: {
            properties: {},
            frontmatter: {},
            aliases: [],
            cssClasses: []
          }
        }
      ],
      hierarchy: [
        {
          id: 'node_1',
          content: 'Main',
          level: 1,
          type: 'heading' as const,
          children: ['node_2'],
          position: {
            fileName: 'test.md',
            lineStart: 1,
            lineEnd: 1,
            charStart: 0,
            charEnd: 4
          }
        },
        {
          id: 'node_2',
          content: 'Sub',
          level: 2,
          type: 'heading' as const,
          parent: 'node_1',
          children: [],
          position: {
            fileName: 'test.md',
            lineStart: 2,
            lineEnd: 2,
            charStart: 5,
            charEnd: 8
          }
        }
      ],
      metadata: {
        title: 'Test',
        tags: ['test'],
        properties: {},
        frontmatter: {},
        aliases: [],
        cssClasses: [],
        createdTime: new Date(),
        modifiedTime: new Date()
      },
      positions: new Map()
    });
  });

  describe('parseContent', () => {
    it('should parse content successfully', async () => {
      const content = `# Test Document

This is test content with #tags.

## Section

More content here.`;

      const result = await contentManager.parseContent(content, 'test.md');

      expect(result).toBeDefined();
      expect(result.chunks).toBeDefined();
      expect(result.hierarchy).toBeDefined();
      expect(result.metadata).toBeDefined();
      expect(result.positions).toBeDefined();
      expect(result.chunks.length).toBeGreaterThan(0);
    });

    it('should use cached content when available', async () => {
      const content = 'Test content';
      const filePath = 'test.md';

      // First call
      const result1 = await contentManager.parseContent(content, filePath);
      
      // Second call should use cache
      const result2 = await contentManager.parseContent(content, filePath);

      expect(result1).toEqual(result2);
      expect(mockLogger.debug).toHaveBeenCalledWith(
        expect.stringContaining('Using cached parsed content')
      );
    });

    it('should handle parsing errors gracefully', async () => {
      // Mock MarkdownParser to throw error
      (MarkdownParser.parseContent as jest.Mock).mockImplementation(() => {
        throw new Error('Parse error');
      });

      await expect(
        contentManager.parseContent('content', 'test.md')
      ).rejects.toThrow(PluginError);

      expect(mockLogger.error).toHaveBeenCalled();
    });
  });

  describe('syncToInkGateway', () => {
    const createTestChunk = (id: string): UnifiedChunk => ({
      chunkId: id,
      contents: `Content ${id}`,
      isPage: false,
      isTag: false,
      isTemplate: false,
      isSlot: false,
      tags: [],
      metadata: {},
      createdTime: new Date(),
      lastUpdated: new Date(),
      documentId: 'test-doc-1',
      documentScope: 'file' as const,
      position: {
        fileName: 'test.md',
        lineStart: 1,
        lineEnd: 1,
        charStart: 0,
        charEnd: 10
      },
      filePath: 'test.md',
      obsidianMetadata: {
        properties: {},
        frontmatter: {},
        aliases: [],
        cssClasses: []
      }
    });

    it('should sync chunks successfully', async () => {
      const chunks = [createTestChunk('1'), createTestChunk('2')];

      const result = await contentManager.syncToInkGateway(chunks);

      expect(result.success).toBe(true);
      expect(result.syncedChunks).toBe(2);
      expect(result.errors).toHaveLength(0);
      expect(mockLogger.info).toHaveBeenCalledWith(
        'Sync completed',
        expect.objectContaining({ success: true, syncedChunks: 2 })
      );
    });

    it('should handle API unavailable', async () => {
      mockApiClient.setHealthy(false);
      const chunks = [createTestChunk('1')];

      const result = await contentManager.syncToInkGateway(chunks);

      expect(result.success).toBe(false);
      expect(result.syncedChunks).toBe(0);
      expect(result.errors).toHaveLength(1);
    });

    it('should handle batch sync failures', async () => {
      mockApiClient.setShouldFail(true);
      const chunks = [createTestChunk('1'), createTestChunk('2')];

      const result = await contentManager.syncToInkGateway(chunks);

      expect(result.success).toBe(false);
      expect(result.errors.length).toBeGreaterThan(0);
      expect(mockLogger.error).toHaveBeenCalled();
    });

    it('should process chunks in batches', async () => {
      // Create more chunks than batch size
      const chunks = Array.from({ length: 25 }, (_, i) => createTestChunk(`chunk_${i}`));

      const result = await contentManager.syncToInkGateway(chunks);

      expect(result.success).toBe(true);
      expect(result.syncedChunks).toBe(25);
    });

    it('should emit sync completed event', async () => {
      const chunks = [createTestChunk('1')];
      const eventSpy = jest.fn();
      mockEventManager.on('syncCompleted', eventSpy);

      await contentManager.syncToInkGateway(chunks);

      expect(eventSpy).toHaveBeenCalledWith(
        expect.objectContaining({ success: true })
      );
    });
  });

  describe('handleContentChange', () => {
    it('should handle content change successfully', async () => {
      const file = new MockTFile('test.md', 'test');
      mockApp.vault.read.mockResolvedValue('# New Content\n\nUpdated content.');

      await contentManager.handleContentChange(file as any);

      expect(mockApp.vault.read).toHaveBeenCalledWith(file);
      expect(mockLogger.info).toHaveBeenCalledWith(
        expect.stringContaining('Successfully processed content change')
      );
    });

    it('should skip processing if content unchanged', async () => {
      const file = new MockTFile('test.md', 'test');
      const content = '# Same Content';
      mockApp.vault.read.mockResolvedValue(content);

      // First call
      await contentManager.handleContentChange(file as any);
      
      // Reset mocks
      jest.clearAllMocks();
      
      // Second call with same content
      await contentManager.handleContentChange(file as any);

      expect(mockLogger.debug).toHaveBeenCalledWith(
        expect.stringContaining('Content unchanged')
      );
    });

    it('should prevent duplicate processing', async () => {
      const file = new MockTFile('test.md', 'test');
      mockApp.vault.read.mockResolvedValue('# Content');

      // Start two concurrent processes
      const promise1 = contentManager.handleContentChange(file as any);
      const promise2 = contentManager.handleContentChange(file as any);

      await Promise.all([promise1, promise2]);

      // Second call should be skipped
      expect(mockLogger.debug).toHaveBeenCalledWith(
        expect.stringContaining('Already processing file')
      );
    });

    it('should handle errors gracefully', async () => {
      const file = new MockTFile('test.md', 'test');
      mockApp.vault.read.mockRejectedValue(new Error('Read error'));

      await expect(
        contentManager.handleContentChange(file as any)
      ).rejects.toThrow('Read error');

      expect(mockLogger.error).toHaveBeenCalled();
    });
  });

  describe('parseHierarchy', () => {
    it('should parse hierarchy from content', () => {
      const content = `# Main
## Sub
### Deep`;

      const hierarchy = contentManager.parseHierarchy(content);

      expect(hierarchy).toBeDefined();
      expect(hierarchy.length).toBeGreaterThan(0);
      
      const mainNode = hierarchy.find(n => n.content === 'Main');
      const subNode = hierarchy.find(n => n.content === 'Sub');

      expect(mainNode).toBeDefined();
      expect(subNode).toBeDefined();
      expect(subNode?.parent).toBe(mainNode?.id);
    });

    it('should handle parsing errors gracefully', () => {
      // Mock MarkdownParser to throw error
      (MarkdownParser.parseContent as jest.Mock).mockImplementation(() => {
        throw new Error('Parse error');
      });

      const hierarchy = contentManager.parseHierarchy('content');

      expect(hierarchy).toEqual([]);
      expect(mockLogger.error).toHaveBeenCalled();
    });
  });

  describe('extractMetadata', () => {
    it('should extract metadata from file', () => {
      const file = new MockTFile('test.md', 'test');

      const metadata = contentManager.extractMetadata(file as any);

      expect(metadata).toBeDefined();
      expect(metadata.title).toBe('Test'); // From mock frontmatter
      expect(metadata.tags).toContain('test');
      expect(metadata.tags).toContain('important');
      expect(metadata.properties.links).toContain('other-file');
    });

    it('should handle missing cached metadata', () => {
      const file = new MockTFile('test.md', 'test');
      mockApp.metadataCache.getFileCache.mockReturnValue(null);

      const metadata = contentManager.extractMetadata(file as any);

      expect(metadata).toBeDefined();
      expect(metadata.title).toBe('test'); // Falls back to basename
      expect(metadata.tags).toEqual([]);
    });

    it('should handle extraction errors gracefully', () => {
      const file = new MockTFile('test.md', 'test');
      mockApp.metadataCache.getFileCache.mockImplementation(() => {
        throw new Error('Cache error');
      });

      expect(() => {
        contentManager.extractMetadata(file as any);
      }).toThrow('METADATA_EXTRACTION_FAILED');

      expect(mockLogger.error).toHaveBeenCalled();
    });
  });

  describe('utility methods', () => {
    it('should track processing status', async () => {
      const file = new MockTFile('test.md', 'test');
      mockApp.vault.read.mockImplementation(() => 
        new Promise(resolve => setTimeout(() => resolve('content'), 100))
      );

      const processingPromise = contentManager.handleContentChange(file as any);
      
      expect(contentManager.isProcessing('test.md')).toBe(true);
      
      await processingPromise;
      
      expect(contentManager.isProcessing('test.md')).toBe(false);
    });

    it('should force sync file', async () => {
      const file = new MockTFile('test.md', 'test');
      // Mock the file as a TFile instance
      Object.setPrototypeOf(file, { constructor: { name: 'TFile' } });
      mockApp.vault.getAbstractFileByPath.mockReturnValue(file);
      mockApp.vault.read.mockResolvedValue('# Content');

      await contentManager.forceSyncFile('test.md');

      expect(mockApp.vault.getAbstractFileByPath).toHaveBeenCalledWith('test.md');
      expect(mockApp.vault.read).toHaveBeenCalledWith(file);
    });

    it('should handle force sync for non-existent file', async () => {
      mockApp.vault.getAbstractFileByPath.mockReturnValue(null);

      await expect(
        contentManager.forceSyncFile('nonexistent.md')
      ).rejects.toThrow(PluginError);
    });

    afterEach(() => {
      // Clean up any resources
      if (contentManager && typeof contentManager.cleanup === 'function') {
        contentManager.cleanup();
      }
    });
  });

  describe('event handling', () => {
    it('should handle clear cache event', () => {
      const initialSize = mockCacheManager.size();
      mockCacheManager.set('test', 'value');
      expect(mockCacheManager.size()).toBe(initialSize + 1);

      mockEventManager.emit('clearCache');

      expect(mockCacheManager.size()).toBe(0);
    });

    it('should handle force sync file event', async () => {
      const file = new MockTFile('test.md', 'test');
      mockApp.vault.getAbstractFileByPath.mockReturnValue(file);
      mockApp.vault.read.mockResolvedValue('# Content');

      mockEventManager.emit('forceSyncFile', 'test.md');

      // Wait for async operation
      await new Promise(resolve => setTimeout(resolve, 10));

      expect(mockApp.vault.getAbstractFileByPath).toHaveBeenCalledWith('test.md');
    });
  });
});