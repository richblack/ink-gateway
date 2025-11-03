/**
 * Unit tests for Document ID Management functionality
 */

import { ContentManager } from '../ContentManager';
import { InkGatewayClient } from '../../api/InkGatewayClient';
import { 
  VirtualDocumentContext, 
  VirtualDocument, 
  DocumentChunksResult, 
  PaginationOptions,
  ReconstructedDocument,
  DocumentScope,
  UnifiedChunk,
  HierarchyNode,
  Position,
  PluginError,
  ErrorType
} from '../../types';
import { 
  ILogger, 
  IEventManager, 
  ICacheManager, 
  IOfflineManager 
} from '../../interfaces';

// Mock implementations
class MockLogger implements ILogger {
  debug = jest.fn();
  info = jest.fn();
  warn = jest.fn();
  error = jest.fn();
}

class MockEventManager implements IEventManager {
  private listeners = new Map<string, Function[]>();
  
  on(event: string, callback: Function): void {
    if (!this.listeners.has(event)) {
      this.listeners.set(event, []);
    }
    this.listeners.get(event)!.push(callback);
  }
  
  off(event: string, callback: Function): void {
    const callbacks = this.listeners.get(event);
    if (callbacks) {
      const index = callbacks.indexOf(callback);
      if (index > -1) {
        callbacks.splice(index, 1);
      }
    }
  }
  
  emit(event: string, ...args: any[]): void {
    const callbacks = this.listeners.get(event);
    if (callbacks) {
      callbacks.forEach(callback => callback(...args));
    }
  }
}

class MockCacheManager implements ICacheManager {
  private cache = new Map<string, { value: any; expiry?: number }>();
  
  get<T>(key: string): T | null {
    const item = this.cache.get(key);
    if (!item) return null;
    
    if (item.expiry && Date.now() > item.expiry) {
      this.cache.delete(key);
      return null;
    }
    
    return item.value;
  }
  
  set<T>(key: string, value: T, ttl?: number): void {
    const expiry = ttl ? Date.now() + ttl : undefined;
    this.cache.set(key, { value, expiry });
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

class MockOfflineManager implements IOfflineManager {
  isOnline = jest.fn().mockReturnValue(true);
  queueOperation = jest.fn();
  syncWhenOnline = jest.fn();
  handleConflicts = jest.fn();
}

class MockApp {
  vault = {
    read: jest.fn(),
    getAbstractFileByPath: jest.fn()
  };
}

describe('ContentManager - Document ID Management', () => {
  let contentManager: ContentManager;
  let mockApiClient: jest.Mocked<InkGatewayClient>;
  let mockLogger: MockLogger;
  let mockEventManager: MockEventManager;
  let mockCacheManager: MockCacheManager;
  let mockOfflineManager: MockOfflineManager;
  let mockApp: MockApp;

  beforeEach(() => {
    // Create mocks
    mockApiClient = {
      getChunksByDocumentId: jest.fn(),
      createVirtualDocument: jest.fn(),
      updateDocumentScope: jest.fn(),
    } as any;

    mockLogger = new MockLogger();
    mockEventManager = new MockEventManager();
    mockCacheManager = new MockCacheManager();
    mockOfflineManager = new MockOfflineManager();
    mockApp = new MockApp();

    // Create ContentManager instance
    contentManager = new ContentManager(
      mockApiClient,
      mockLogger,
      mockEventManager,
      mockCacheManager,
      mockOfflineManager,
      mockApp
    );
  });

  describe('generateDocumentId', () => {
    it('should generate consistent document ID for file path', () => {
      const filePath = 'notes/project/readme.md';
      const documentId1 = contentManager.generateDocumentId(filePath);
      const documentId2 = contentManager.generateDocumentId(filePath);
      
      expect(documentId1).toBe(documentId2);
      expect(documentId1).toMatch(/^file_[a-z0-9]+_notes_project_readme_md$/);
    });

    it('should handle different file paths', () => {
      const filePath1 = 'notes/project1/readme.md';
      const filePath2 = 'notes/project2/readme.md';
      
      const documentId1 = contentManager.generateDocumentId(filePath1);
      const documentId2 = contentManager.generateDocumentId(filePath2);
      
      expect(documentId1).not.toBe(documentId2);
    });

    it('should normalize file paths', () => {
      const filePath1 = '/notes/project/readme.md';
      const filePath2 = 'notes\\project\\readme.md';
      const filePath3 = 'notes/project/readme.md';
      
      const documentId1 = contentManager.generateDocumentId(filePath1);
      const documentId2 = contentManager.generateDocumentId(filePath2);
      const documentId3 = contentManager.generateDocumentId(filePath3);
      
      expect(documentId1).toBe(documentId2);
      expect(documentId2).toBe(documentId3);
    });

    it('should throw error for invalid file path', () => {
      expect(() => contentManager.generateDocumentId('')).toThrow(PluginError);
      expect(() => contentManager.generateDocumentId(null as any)).toThrow(PluginError);
      expect(() => contentManager.generateDocumentId(undefined as any)).toThrow(PluginError);
    });
  });

  describe('generateVirtualDocumentId', () => {
    it('should generate consistent virtual document ID', () => {
      const context: VirtualDocumentContext = {
        sourceType: 'remnote',
        contextId: 'page123',
        pageTitle: 'Test Page',
        metadata: { category: 'notes' }
      };
      
      const virtualId1 = contentManager.generateVirtualDocumentId(context);
      const virtualId2 = contentManager.generateVirtualDocumentId(context);
      
      expect(virtualId1).toBe(virtualId2);
      expect(virtualId1).toMatch(/^virtual_remnote_[a-z0-9]+_page123$/);
    });

    it('should handle different source types', () => {
      const context1: VirtualDocumentContext = {
        sourceType: 'remnote',
        contextId: 'page123',
        metadata: {}
      };
      
      const context2: VirtualDocumentContext = {
        sourceType: 'logseq',
        contextId: 'page123',
        metadata: {}
      };
      
      const virtualId1 = contentManager.generateVirtualDocumentId(context1);
      const virtualId2 = contentManager.generateVirtualDocumentId(context2);
      
      expect(virtualId1).not.toBe(virtualId2);
      expect(virtualId1).toContain('virtual_remnote_');
      expect(virtualId2).toContain('virtual_logseq_');
    });

    it('should throw error for invalid context', () => {
      expect(() => contentManager.generateVirtualDocumentId(null as any)).toThrow(PluginError);
      expect(() => contentManager.generateVirtualDocumentId({} as any)).toThrow(PluginError);
      expect(() => contentManager.generateVirtualDocumentId({
        sourceType: 'remnote',
        contextId: '',
        metadata: {}
      })).toThrow(PluginError);
    });
  });

  describe('getChunksByDocumentId', () => {
    const mockDocumentId = 'file_abc123_notes_test_md';
    const mockChunksResult: DocumentChunksResult = {
      chunks: [
        {
          chunkId: 'chunk1',
          contents: 'Test content 1',
          documentId: mockDocumentId,
          documentScope: 'file',
          position: { fileName: 'test.md', lineStart: 1, lineEnd: 1, charStart: 0, charEnd: 10 },
          filePath: 'notes/test.md',
          obsidianMetadata: { properties: {}, frontmatter: {}, aliases: [], cssClasses: [] },
          isPage: false,
          isTag: false,
          isTemplate: false,
          isSlot: false,
          tags: [],
          metadata: {},
          createdTime: new Date(),
          lastUpdated: new Date()
        }
      ],
      pagination: {
        currentPage: 1,
        totalPages: 1,
        totalChunks: 1,
        pageSize: 10
      },
      documentMetadata: {
        totalChunks: 1,
        documentScope: 'file',
        lastModified: new Date()
      }
    };

    it('should retrieve chunks by document ID', async () => {
      mockApiClient.getChunksByDocumentId.mockResolvedValue(mockChunksResult);
      
      const result = await contentManager.getChunksByDocumentId(mockDocumentId);
      
      expect(result).toBe(mockChunksResult);
      expect(mockApiClient.getChunksByDocumentId).toHaveBeenCalledWith(mockDocumentId, undefined);
    });

    it('should pass pagination options to API', async () => {
      const options: PaginationOptions = {
        page: 2,
        pageSize: 20,
        includeHierarchy: true,
        sortBy: 'position',
        sortOrder: 'asc'
      };
      
      mockApiClient.getChunksByDocumentId.mockResolvedValue(mockChunksResult);
      
      await contentManager.getChunksByDocumentId(mockDocumentId, options);
      
      expect(mockApiClient.getChunksByDocumentId).toHaveBeenCalledWith(mockDocumentId, options);
    });

    it('should use cache when available', async () => {
      mockApiClient.getChunksByDocumentId.mockResolvedValue(mockChunksResult);
      
      // First call
      await contentManager.getChunksByDocumentId(mockDocumentId);
      
      // Second call should use cache
      const result = await contentManager.getChunksByDocumentId(mockDocumentId);
      
      expect(result).toBe(mockChunksResult);
      expect(mockApiClient.getChunksByDocumentId).toHaveBeenCalledTimes(1);
    });

    it('should handle API errors', async () => {
      const apiError = new Error('API Error');
      mockApiClient.getChunksByDocumentId.mockRejectedValue(apiError);
      
      await expect(contentManager.getChunksByDocumentId(mockDocumentId))
        .rejects.toThrow(PluginError);
    });
  });

  describe('createVirtualDocument', () => {
    const mockContext: VirtualDocumentContext = {
      sourceType: 'remnote',
      contextId: 'page123',
      pageTitle: 'Test Page',
      metadata: { category: 'notes' }
    };

    const mockVirtualDoc: VirtualDocument = {
      virtualDocumentId: 'virtual_remnote_abc123_page123',
      context: mockContext,
      chunkIds: ['chunk1', 'chunk2'],
      createdAt: new Date(),
      lastUpdated: new Date()
    };

    it('should create virtual document', async () => {
      mockApiClient.createVirtualDocument.mockResolvedValue(mockVirtualDoc);
      
      const result = await contentManager.createVirtualDocument(mockContext);
      
      expect(result).toBe(mockVirtualDoc);
      expect(mockApiClient.createVirtualDocument).toHaveBeenCalledWith(mockContext);
    });

    it('should use cache for existing virtual documents', async () => {
      mockApiClient.createVirtualDocument.mockResolvedValue(mockVirtualDoc);
      
      // First call
      await contentManager.createVirtualDocument(mockContext);
      
      // Second call should use cache
      const result = await contentManager.createVirtualDocument(mockContext);
      
      expect(result).toBe(mockVirtualDoc);
      expect(mockApiClient.createVirtualDocument).toHaveBeenCalledTimes(1);
    });

    it('should handle API errors', async () => {
      const apiError = new Error('API Error');
      mockApiClient.createVirtualDocument.mockRejectedValue(apiError);
      
      await expect(contentManager.createVirtualDocument(mockContext))
        .rejects.toThrow(PluginError);
    });
  });

  describe('reconstructDocument', () => {
    const mockDocumentId = 'file_abc123_notes_test_md';
    const mockPosition: Position = {
      fileName: 'test.md',
      lineStart: 1,
      lineEnd: 1,
      charStart: 0,
      charEnd: 10
    };

    const mockChunks: UnifiedChunk[] = [
      {
        chunkId: 'chunk1',
        contents: '# Header 1',
        parent: undefined,
        documentId: mockDocumentId,
        documentScope: 'file',
        position: mockPosition,
        filePath: 'notes/test.md',
        obsidianMetadata: { properties: {}, frontmatter: {}, aliases: [], cssClasses: [] },
        isPage: true,
        isTag: false,
        isTemplate: false,
        isSlot: false,
        tags: [],
        metadata: {},
        createdTime: new Date(),
        lastUpdated: new Date()
      },
      {
        chunkId: 'chunk2',
        contents: 'Content under header 1',
        parent: 'chunk1',
        documentId: mockDocumentId,
        documentScope: 'file',
        position: mockPosition,
        filePath: 'notes/test.md',
        obsidianMetadata: { properties: {}, frontmatter: {}, aliases: [], cssClasses: [] },
        isPage: false,
        isTag: false,
        isTemplate: false,
        isSlot: false,
        tags: [],
        metadata: {},
        createdTime: new Date(),
        lastUpdated: new Date()
      }
    ];

    const mockChunksResult: DocumentChunksResult = {
      chunks: mockChunks,
      pagination: {
        currentPage: 1,
        totalPages: 1,
        totalChunks: 2,
        pageSize: 1000
      },
      documentMetadata: {
        totalChunks: 2,
        documentScope: 'file',
        lastModified: new Date()
      }
    };

    it('should reconstruct document from chunks', async () => {
      mockApiClient.getChunksByDocumentId.mockResolvedValue(mockChunksResult);
      
      const result = await contentManager.reconstructDocument(mockDocumentId);
      
      expect(result.documentId).toBe(mockDocumentId);
      expect(result.chunks).toHaveLength(2);
      expect(result.hierarchy).toHaveLength(1); // One root node
      expect(result.metadata.totalChunks).toBe(2);
    });

    it('should use cache when available', async () => {
      mockApiClient.getChunksByDocumentId.mockResolvedValue(mockChunksResult);
      
      // First call
      await contentManager.reconstructDocument(mockDocumentId);
      
      // Second call should use cache
      const result = await contentManager.reconstructDocument(mockDocumentId);
      
      expect(result.documentId).toBe(mockDocumentId);
      expect(mockApiClient.getChunksByDocumentId).toHaveBeenCalledTimes(1);
    });

    it('should handle empty document', async () => {
      const emptyResult: DocumentChunksResult = {
        chunks: [],
        pagination: { currentPage: 1, totalPages: 0, totalChunks: 0, pageSize: 1000 },
        documentMetadata: { totalChunks: 0, documentScope: 'file', lastModified: new Date() }
      };
      
      mockApiClient.getChunksByDocumentId.mockResolvedValue(emptyResult);
      
      await expect(contentManager.reconstructDocument(mockDocumentId))
        .rejects.toThrow(PluginError);
    });
  });

  describe('updateDocumentScope', () => {
    const mockChunkId = 'chunk123';
    const mockDocumentId = 'file_abc123_notes_test_md';
    const mockScope: DocumentScope = 'virtual';

    it('should update document scope', async () => {
      mockApiClient.updateDocumentScope.mockResolvedValue(undefined);
      
      await contentManager.updateDocumentScope(mockChunkId, mockDocumentId, mockScope);
      
      expect(mockApiClient.updateDocumentScope).toHaveBeenCalledWith(
        mockChunkId,
        mockDocumentId,
        mockScope
      );
    });

    it('should clear related caches after update', async () => {
      mockApiClient.updateDocumentScope.mockResolvedValue(undefined);
      
      // Set some cache entries
      mockCacheManager.set(`doc_chunks_${mockDocumentId}_{}`, { test: 'data' });
      mockCacheManager.set(`chunk_${mockChunkId}`, { test: 'data' });
      
      await contentManager.updateDocumentScope(mockChunkId, mockDocumentId, mockScope);
      
      // Cache should be cleared
      expect(mockCacheManager.get(`chunk_${mockChunkId}`)).toBeNull();
    });

    it('should handle API errors', async () => {
      const apiError = new Error('API Error');
      mockApiClient.updateDocumentScope.mockRejectedValue(apiError);
      
      await expect(contentManager.updateDocumentScope(mockChunkId, mockDocumentId, mockScope))
        .rejects.toThrow(PluginError);
    });
  });

  describe('utility methods', () => {
    it('should identify virtual document IDs', () => {
      expect(contentManager.isVirtualDocumentId('virtual_remnote_abc123_page123')).toBe(true);
      expect(contentManager.isVirtualDocumentId('file_abc123_notes_test_md')).toBe(false);
    });

    it('should extract file path from document ID', () => {
      const documentId = 'file_abc123_notes_project_readme_md';
      const filePath = contentManager.extractFilePathFromDocumentId(documentId);
      
      expect(filePath).toBe('notes/project/readme.md');
    });

    it('should return null for virtual document ID file path extraction', () => {
      const virtualDocId = 'virtual_remnote_abc123_page123';
      const filePath = contentManager.extractFilePathFromDocumentId(virtualDocId);
      
      expect(filePath).toBeNull();
    });

    it('should get document ID from file', () => {
      const mockFile = {
        path: 'notes/test.md'
      } as any;
      
      const documentId = contentManager.getDocumentIdFromFile(mockFile);
      
      expect(documentId).toMatch(/^file_[a-z0-9]+_notes_test_md$/);
    });
  });

  describe('error handling', () => {
    it('should handle validation errors properly', () => {
      expect(() => contentManager.generateDocumentId('')).toThrow(PluginError);
      expect(() => contentManager.generateVirtualDocumentId({} as any)).toThrow(PluginError);
    });

    it('should handle API errors and convert to PluginError', async () => {
      const apiError = new Error('Network error');
      mockApiClient.getChunksByDocumentId.mockRejectedValue(apiError);
      
      try {
        await contentManager.getChunksByDocumentId('test-doc-id');
      } catch (error) {
        expect(error).toBeInstanceOf(PluginError);
        expect((error as PluginError).type).toBe(ErrorType.API_ERROR);
      }
    });

    it('should preserve PluginError instances', async () => {
      const pluginError = new PluginError(
        ErrorType.VALIDATION_ERROR,
        'TEST_ERROR',
        { test: 'data' },
        false
      );
      
      mockApiClient.getChunksByDocumentId.mockRejectedValue(pluginError);
      
      try {
        await contentManager.getChunksByDocumentId('test-doc-id');
      } catch (error) {
        expect(error).toBe(pluginError);
      }
    });
  });
});