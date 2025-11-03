/**
 * Unit tests for MetadataManager
 */

import { MetadataManager, TagSyncOptions, MetadataProcessingOptions } from '../MetadataManager';
import { 
  IInkGatewayClient, 
  ILogger, 
  IEventManager, 
  ICacheManager 
} from '../../interfaces';
import { 
  ContentMetadata,
  UnifiedChunk
} from '../../types';

// Mock implementations
class MockInkGatewayClient implements IInkGatewayClient {
  private tags = new Set(['existing-tag', 'remote-tag']);
  
  async searchByTags(tags: string[]): Promise<any> {
    return {
      items: [
        {
          chunk: {
            tags: Array.from(this.tags)
          }
        }
      ]
    };
  }

  // Other required methods (simplified for testing)
  async healthCheck(): Promise<boolean> { return true; }
  async createChunk(chunk: UnifiedChunk): Promise<UnifiedChunk> { return chunk; }
  async updateChunk(id: string, chunk: Partial<UnifiedChunk>): Promise<UnifiedChunk> { return chunk as UnifiedChunk; }
  async deleteChunk(id: string): Promise<void> {}
  async getChunk(id: string): Promise<UnifiedChunk> { return {} as UnifiedChunk; }
  async batchCreateChunks(chunks: UnifiedChunk[]): Promise<UnifiedChunk[]> { return chunks; }
  async searchChunks(query: any): Promise<any> { return { items: [], totalCount: 0, searchTime: 0, cacheHit: false }; }
  async searchSemantic(content: string): Promise<any> { return { items: [], totalCount: 0, searchTime: 0, cacheHit: false }; }
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
  metadataCache = {
    getFileCache: jest.fn()
  };
}

describe('MetadataManager', () => {
  let metadataManager: MetadataManager;
  let mockApiClient: MockInkGatewayClient;
  let mockLogger: MockLogger;
  let mockEventManager: MockEventManager;
  let mockCacheManager: MockCacheManager;
  let mockApp: MockApp;

  const defaultTagSyncOptions: TagSyncOptions = {
    bidirectionalSync: true,
    autoCreateTags: true,
    excludePatterns: ['private/', 'temp/']
  };

  const defaultProcessingOptions: MetadataProcessingOptions = {
    syncFrontmatter: true,
    syncProperties: true,
    syncTags: true,
    preserveObsidianMetadata: true
  };

  beforeEach(() => {
    mockApiClient = new MockInkGatewayClient();
    mockLogger = new MockLogger();
    mockEventManager = new MockEventManager();
    mockCacheManager = new MockCacheManager();
    mockApp = new MockApp();

    metadataManager = new MetadataManager(
      mockApiClient,
      mockLogger,
      mockEventManager,
      mockCacheManager,
      mockApp,
      defaultTagSyncOptions,
      defaultProcessingOptions
    );
  });

  afterEach(() => {
    if (metadataManager) {
      metadataManager.cleanup();
    }
  });

  describe('extractMetadata', () => {
    it('should extract basic metadata from file', () => {
      const file = new MockTFile('test.md', 'test');
      mockApp.metadataCache.getFileCache.mockReturnValue({
        frontmatter: {
          title: 'Test Document',
          tags: ['test', 'document'],
          created: '2024-01-01'
        },
        tags: [
          { tag: '#important' },
          { tag: '#work' }
        ],
        links: [
          { link: 'other-file' }
        ],
        embeds: [
          { link: 'image.png' }
        ],
        headings: [
          { heading: 'Main Section', level: 1 },
          { heading: 'Subsection', level: 2 }
        ]
      });

      const metadata = metadataManager.extractMetadata(file as any);

      expect(metadata.title).toBe('Test Document');
      expect(metadata.tags).toContain('test');
      expect(metadata.tags).toContain('document');
      expect(metadata.tags).toContain('important');
      expect(metadata.tags).toContain('work');
      expect(metadata.properties.links).toContain('other-file');
      expect(metadata.properties.embeds).toContain('image.png');
      expect(metadata.properties.headings).toHaveLength(2);
      expect(metadata.createdTime).toEqual(new Date('2024-01-01'));
    });

    it('should handle missing cached metadata', () => {
      const file = new MockTFile('test.md', 'test');
      mockApp.metadataCache.getFileCache.mockReturnValue(null);

      const metadata = metadataManager.extractMetadata(file as any);

      expect(metadata.title).toBe('test');
      expect(metadata.tags).toHaveLength(0);
      expect(metadata.properties).toEqual({});
    });

    it('should process different value types in frontmatter', () => {
      const file = new MockTFile('test.md', 'test');
      mockApp.metadataCache.getFileCache.mockReturnValue({
        frontmatter: {
          title: 'Test',
          priority: '5',
          completed: 'true',
          due_date: '2024-12-31',
          tags: ['work', 'important']
        }
      });

      const metadata = metadataManager.extractMetadata(file as any);

      expect(metadata.frontmatter.priority).toBe(5);
      expect(metadata.frontmatter.completed).toBe(true);
      expect(metadata.frontmatter.due_date).toEqual(new Date('2024-12-31'));
      expect(metadata.tags).toEqual(['work', 'important']);
    });

    it('should handle aliases and CSS classes', () => {
      const file = new MockTFile('test.md', 'test');
      mockApp.metadataCache.getFileCache.mockReturnValue({
        frontmatter: {
          aliases: ['alias1', 'alias2'],
          cssclass: 'custom-style'
        }
      });

      const metadata = metadataManager.extractMetadata(file as any);

      expect(metadata.aliases).toEqual(['alias1', 'alias2']);
      expect(metadata.cssClasses).toEqual(['custom-style']);
    });

    it('should filter excluded tag patterns', () => {
      const file = new MockTFile('test.md', 'test');
      mockApp.metadataCache.getFileCache.mockReturnValue({
        tags: [
          { tag: '#public-tag' },
          { tag: '#private/secret' },
          { tag: '#temp/draft' },
          { tag: '#work' }
        ]
      });

      const metadata = metadataManager.extractMetadata(file as any);

      expect(metadata.tags).toContain('public-tag');
      expect(metadata.tags).toContain('work');
      expect(metadata.tags).not.toContain('private/secret');
      expect(metadata.tags).not.toContain('temp/draft');
    });

    it('should handle extraction errors gracefully', () => {
      const file = new MockTFile('test.md', 'test');
      mockApp.metadataCache.getFileCache.mockImplementation(() => {
        throw new Error('Cache error');
      });

      expect(() => {
        metadataManager.extractMetadata(file as any);
      }).toThrow('METADATA_EXTRACTION_FAILED');
    });
  });

  describe('createObsidianMetadata', () => {
    it('should create Obsidian metadata from UnifiedChunk', () => {
      const chunk: UnifiedChunk = {
        chunkId: 'test-chunk',
        contents: 'Test content',
        isPage: false,
        isTag: false,
        isTemplate: false,
        isSlot: false,
        tags: ['work', 'important'],
        metadata: {
          title: 'Test Document',
          priority: 5,
          completed: true
        },
        createdTime: new Date('2024-01-01'),
        lastUpdated: new Date('2024-01-02'),
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
      };

      const obsidianMetadata = metadataManager.createObsidianMetadata(chunk);

      expect(obsidianMetadata.frontmatter.tags).toEqual(['work', 'important']);
      expect(obsidianMetadata.frontmatter.title).toBe('Test Document');
      expect(obsidianMetadata.frontmatter.created).toBe('2024-01-01');
      expect(obsidianMetadata.frontmatter.modified).toBe('2024-01-02');
      expect(obsidianMetadata.properties.priority).toBe(5);
      expect(obsidianMetadata.properties.completed).toBe(true);
    });
  });

  describe('syncTags', () => {
    it('should sync tags bidirectionally', async () => {
      const localTags = ['local-tag', 'shared-tag'];
      const remoteTags = ['remote-tag', 'shared-tag'];

      const result = await metadataManager.syncTags(localTags, remoteTags);

      expect(result.tagsToAdd).toContain('local-tag');
      expect(result.tagsToRemove).toContain('remote-tag');
      expect(result.conflicts).toHaveLength(0);
    });

    it('should handle one-way sync', async () => {
      // Update options for one-way sync
      metadataManager.updateTagSyncOptions({ bidirectionalSync: false });

      const localTags = ['local-tag', 'shared-tag'];
      const remoteTags = ['remote-tag', 'shared-tag'];

      const result = await metadataManager.syncTags(localTags, remoteTags);

      expect(result.tagsToAdd).toContain('local-tag');
      expect(result.tagsToRemove).toHaveLength(0); // No removal in one-way sync
      expect(result.conflicts).toHaveLength(0);
    });

    it('should handle conflicts when auto-create is disabled', async () => {
      // Update options to disable auto-create
      metadataManager.updateTagSyncOptions({ autoCreateTags: false });

      const localTags = ['unknown-tag', 'existing-tag'];
      const remoteTags = ['remote-tag'];

      // Add existing-tag to known tags
      metadataManager.addKnownTag('existing-tag');

      const result = await metadataManager.syncTags(localTags, remoteTags);

      expect(result.tagsToAdd).toContain('existing-tag');
      expect(result.conflicts).toContain('unknown-tag');
    });
  });

  describe('tag normalization', () => {
    it('should normalize tags correctly', () => {
      // Test through tag extraction
      const file = new MockTFile('test.md', 'test');
      mockApp.metadataCache.getFileCache.mockReturnValue({
        tags: [
          { tag: '#  spaced-tag  ' },
          { tag: '###multiple-hash' },
          { tag: '#normal-tag' }
        ]
      });

      const metadata = metadataManager.extractMetadata(file as any);

      expect(metadata.tags).toContain('spaced-tag');
      expect(metadata.tags).toContain('multiple-hash');
      expect(metadata.tags).toContain('normal-tag');
    });

    it('should apply tag prefix when configured', () => {
      // Create new manager with tag prefix
      const managerWithPrefix = new MetadataManager(
        mockApiClient,
        mockLogger,
        mockEventManager,
        mockCacheManager,
        mockApp,
        { ...defaultTagSyncOptions, tagPrefix: 'obsidian/' },
        defaultProcessingOptions
      );

      const file = new MockTFile('test.md', 'test');
      mockApp.metadataCache.getFileCache.mockReturnValue({
        tags: [
          { tag: '#work' },
          { tag: '#obsidian/existing' } // Already has prefix
        ]
      });

      const metadata = managerWithPrefix.extractMetadata(file as any);

      expect(metadata.tags).toContain('obsidian/work');
      expect(metadata.tags).toContain('obsidian/existing');
    });
  });

  describe('event handling', () => {
    it('should handle metadata change events', async () => {
      const file = new MockTFile('test.md', 'test');
      mockApp.metadataCache.getFileCache.mockReturnValue({
        frontmatter: { title: 'Test' },
        tags: [{ tag: '#new-tag' }]
      });

      const eventSpy = jest.fn();
      mockEventManager.on('metadataExtracted', eventSpy);

      mockEventManager.emit('metadataCacheChanged', file);

      // Wait for async processing
      await new Promise(resolve => setTimeout(resolve, 10));

      expect(eventSpy).toHaveBeenCalledWith({
        file,
        metadata: expect.objectContaining({
          title: 'Test',
          tags: expect.arrayContaining(['new-tag'])
        })
      });

      expect(metadataManager.getKnownTags()).toContain('new-tag');
    });

    it('should handle tag change events', async () => {
      const eventSpy = jest.fn();
      mockEventManager.on('tagSyncRequired', eventSpy);

      mockEventManager.emit('tagChanged', 'old-tag', 'new-tag');

      // Wait for async processing
      await new Promise(resolve => setTimeout(resolve, 10));

      expect(eventSpy).toHaveBeenCalledWith({
        type: 'rename',
        oldTag: 'old-tag',
        newTag: 'new-tag'
      });

      expect(metadataManager.getKnownTags()).toContain('new-tag');
      expect(metadataManager.getKnownTags()).not.toContain('old-tag');
    });

    it('should handle tag creation events', async () => {
      const eventSpy = jest.fn();
      mockEventManager.on('tagSyncRequired', eventSpy);

      mockEventManager.emit('tagCreated', 'new-tag');

      // Wait for async processing
      await new Promise(resolve => setTimeout(resolve, 10));

      expect(eventSpy).toHaveBeenCalledWith({
        type: 'create',
        tag: 'new-tag'
      });

      expect(metadataManager.getKnownTags()).toContain('new-tag');
    });

    it('should handle tag deletion events', async () => {
      // Add tag first
      metadataManager.addKnownTag('tag-to-delete');
      expect(metadataManager.getKnownTags()).toContain('tag-to-delete');

      const eventSpy = jest.fn();
      mockEventManager.on('tagSyncRequired', eventSpy);

      mockEventManager.emit('tagDeleted', 'tag-to-delete');

      // Wait for async processing
      await new Promise(resolve => setTimeout(resolve, 10));

      expect(eventSpy).toHaveBeenCalledWith({
        type: 'delete',
        tag: 'tag-to-delete'
      });

      expect(metadataManager.getKnownTags()).not.toContain('tag-to-delete');
    });
  });

  describe('configuration management', () => {
    it('should update processing options', () => {
      const newOptions = {
        syncFrontmatter: false,
        syncTags: false
      };

      metadataManager.updateProcessingOptions(newOptions);

      const stats = metadataManager.getStats();
      expect(stats.processingOptions.syncFrontmatter).toBe(false);
      expect(stats.processingOptions.syncTags).toBe(false);
      expect(stats.processingOptions.syncProperties).toBe(true); // Unchanged
    });

    it('should update tag sync options', () => {
      const newOptions = {
        bidirectionalSync: false,
        autoCreateTags: false
      };

      metadataManager.updateTagSyncOptions(newOptions);

      const stats = metadataManager.getStats();
      expect(stats.tagSyncOptions.bidirectionalSync).toBe(false);
      expect(stats.tagSyncOptions.autoCreateTags).toBe(false);
    });
  });

  describe('cache management', () => {
    it('should manage known tags cache', () => {
      metadataManager.addKnownTag('cached-tag');
      expect(metadataManager.getKnownTags()).toContain('cached-tag');

      metadataManager.removeKnownTag('cached-tag');
      expect(metadataManager.getKnownTags()).not.toContain('cached-tag');
    });

    it('should clear all caches', () => {
      metadataManager.addKnownTag('tag-to-clear');
      expect(metadataManager.getKnownTags()).toContain('tag-to-clear');

      metadataManager.clearCaches();
      expect(metadataManager.getKnownTags()).toHaveLength(0);
    });

    it('should restore known tags from cache on initialization', () => {
      const cachedTags = ['cached-tag-1', 'cached-tag-2'];
      mockCacheManager.set('known_tags', cachedTags);

      const newManager = new MetadataManager(
        mockApiClient,
        mockLogger,
        mockEventManager,
        mockCacheManager,
        mockApp,
        defaultTagSyncOptions,
        defaultProcessingOptions
      );

      expect(newManager.getKnownTags()).toEqual(expect.arrayContaining(cachedTags));
    });
  });

  describe('statistics', () => {
    it('should provide processing statistics', () => {
      metadataManager.addKnownTag('stat-tag');

      const stats = metadataManager.getStats();

      expect(stats.knownTags).toBeGreaterThan(0);
      expect(stats.syncInProgress).toBe(0);
      expect(stats.processingOptions).toEqual(defaultProcessingOptions);
      expect(stats.tagSyncOptions).toEqual(defaultTagSyncOptions);
    });
  });

  describe('error handling', () => {
    it('should handle API errors during initialization', async () => {
      const failingApiClient = {
        ...mockApiClient,
        searchByTags: jest.fn().mockRejectedValue(new Error('API Error'))
      };

      // Clear cache to force API call
      mockCacheManager.delete('known_tags');

      const managerWithFailingApi = new MetadataManager(
        failingApiClient as any,
        mockLogger,
        mockEventManager,
        mockCacheManager,
        mockApp,
        defaultTagSyncOptions,
        defaultProcessingOptions
      );

      // Wait for async initialization to complete
      await new Promise(resolve => setTimeout(resolve, 50));

      // Should not throw, should continue with empty known tags
      expect(managerWithFailingApi.getKnownTags()).toHaveLength(0);
      expect(mockLogger.error).toHaveBeenCalledWith(
        'Failed to initialize known tags:',
        expect.any(Error)
      );
    });

    it('should handle metadata extraction errors', () => {
      const file = new MockTFile('test.md', 'test');
      mockApp.metadataCache.getFileCache.mockImplementation(() => {
        throw new Error('Metadata cache error');
      });

      expect(() => {
        metadataManager.extractMetadata(file as any);
      }).toThrow('METADATA_EXTRACTION_FAILED');

      expect(mockLogger.error).toHaveBeenCalled();
    });
  });
});