/**
 * Integration tests for Document ID Pagination API methods
 * Tests real-world scenarios and end-to-end workflows
 */

import { InkGatewayClient } from '../InkGatewayClient';
import { 
  VirtualDocumentContext, 
  VirtualDocument, 
  DocumentScope, 
  PaginationOptions, 
  DocumentChunksResult,
  UnifiedChunk,
  DocumentMetadata
} from '../../types';

// Mock fetch globally
global.fetch = jest.fn();
const mockFetch = global.fetch as jest.MockedFunction<typeof fetch>;

// Mock console methods to avoid noise in tests
const originalConsole = console;
beforeAll(() => {
  console.error = jest.fn();
  console.warn = jest.fn();
});

afterAll(() => {
  console.error = originalConsole.error;
  console.warn = originalConsole.warn;
});

describe('InkGatewayClient - Document ID Pagination Integration', () => {
  let client: InkGatewayClient;
  const baseUrl = 'https://api.example.com';
  const apiKey = 'test-api-key';

  beforeEach(() => {
    client = new InkGatewayClient(baseUrl, apiKey);
    mockFetch.mockClear();
    jest.clearAllTimers();
    jest.useFakeTimers();
  });

  afterEach(() => {
    jest.useRealTimers();
  });

  describe('Complete Document Workflow', () => {
    it('should handle complete document creation and retrieval workflow', async () => {
      // Step 1: Create a virtual document
      const virtualContext: VirtualDocumentContext = {
        sourceType: 'remnote',
        contextId: 'workflow-test-context',
        pageTitle: 'Integration Test Page',
        metadata: {
          author: 'Test User',
          category: 'integration-test'
        }
      };

      const mockVirtualDocument: VirtualDocument = {
        virtualDocumentId: 'virtual-doc-workflow-123',
        context: virtualContext,
        chunkIds: ['chunk-1', 'chunk-2', 'chunk-3'],
        createdAt: new Date('2023-01-01'),
        lastUpdated: new Date('2023-01-02')
      };

      mockFetch.mockResolvedValueOnce(createMockResponse(mockVirtualDocument));

      const createdDocument = await client.createVirtualDocument(virtualContext);
      expect(createdDocument.virtualDocumentId).toBe('virtual-doc-workflow-123');
      expect(createdDocument.chunkIds).toHaveLength(3);

      // Step 2: Retrieve chunks by document ID
      const mockChunks: UnifiedChunk[] = [
        createMockChunk('chunk-1', 'First chunk content', createdDocument.virtualDocumentId),
        createMockChunk('chunk-2', 'Second chunk content', createdDocument.virtualDocumentId),
        createMockChunk('chunk-3', 'Third chunk content', createdDocument.virtualDocumentId)
      ];

      const mockDocumentResult: DocumentChunksResult = {
        chunks: mockChunks,
        pagination: {
          currentPage: 1,
          totalPages: 1,
          totalChunks: 3,
          pageSize: 20
        },
        documentMetadata: {
          virtualContext: virtualContext,
          totalChunks: 3,
          documentScope: 'virtual' as DocumentScope,
          lastModified: new Date('2023-01-02')
        }
      };

      mockFetch.mockResolvedValueOnce(createMockResponse(mockDocumentResult));

      const retrievedChunks = await client.getChunksByDocumentId(createdDocument.virtualDocumentId);
      expect(retrievedChunks.chunks).toHaveLength(3);
      expect(retrievedChunks.documentMetadata.documentScope).toBe('virtual');

      // Step 3: Update document scope for one of the chunks
      mockFetch.mockResolvedValueOnce(createMockResponse({}));

      await client.updateDocumentScope('chunk-1', createdDocument.virtualDocumentId, 'page');

      // Verify all API calls were made correctly
      expect(mockFetch).toHaveBeenCalledTimes(3);
      
      // Verify create virtual document call
      expect(mockFetch).toHaveBeenNthCalledWith(1,
        `${baseUrl}/api/documents/virtual`,
        expect.objectContaining({
          method: 'POST',
          body: JSON.stringify(virtualContext)
        })
      );

      // Verify get chunks call
      expect(mockFetch).toHaveBeenNthCalledWith(2,
        `${baseUrl}/api/documents/${encodeURIComponent(createdDocument.virtualDocumentId)}/chunks`,
        expect.objectContaining({
          method: 'GET'
        })
      );

      // Verify update document scope call
      expect(mockFetch).toHaveBeenNthCalledWith(3,
        `${baseUrl}/api/chunks/chunk-1/document-scope`,
        expect.objectContaining({
          method: 'PUT',
          body: JSON.stringify({
            documentId: createdDocument.virtualDocumentId,
            scope: 'page'
          })
        })
      );
    });
  });

  describe('Pagination Scenarios', () => {
    it('should handle large document with multiple pages', async () => {
      const documentId = 'large-doc-123';
      const totalChunks = 150;
      const pageSize = 20;

      // Test first page
      const firstPageChunks = Array.from({ length: pageSize }, (_, i) => 
        createMockChunk(`chunk-${i + 1}`, `Content ${i + 1}`, documentId)
      );

      const firstPageResult: DocumentChunksResult = {
        chunks: firstPageChunks,
        pagination: {
          currentPage: 1,
          totalPages: Math.ceil(totalChunks / pageSize),
          totalChunks,
          pageSize
        },
        documentMetadata: {
          originalFilePath: 'large-document.md',
          totalChunks,
          documentScope: 'file' as DocumentScope,
          lastModified: new Date('2023-01-02')
        }
      };

      mockFetch.mockResolvedValueOnce(createMockResponse(firstPageResult));

      const firstPage = await client.getChunksByDocumentId(documentId, { page: 1, pageSize });
      expect(firstPage.chunks).toHaveLength(pageSize);
      expect(firstPage.pagination.currentPage).toBe(1);
      expect(firstPage.pagination.totalPages).toBe(8);

      // Test middle page
      const middlePageChunks = Array.from({ length: pageSize }, (_, i) => 
        createMockChunk(`chunk-${(4 - 1) * pageSize + i + 1}`, `Content ${(4 - 1) * pageSize + i + 1}`, documentId)
      );

      const middlePageResult: DocumentChunksResult = {
        chunks: middlePageChunks,
        pagination: {
          currentPage: 4,
          totalPages: Math.ceil(totalChunks / pageSize),
          totalChunks,
          pageSize
        },
        documentMetadata: firstPageResult.documentMetadata
      };

      mockFetch.mockResolvedValueOnce(createMockResponse(middlePageResult));

      const middlePage = await client.getChunksByDocumentId(documentId, { page: 4, pageSize });
      expect(middlePage.chunks).toHaveLength(pageSize);
      expect(middlePage.pagination.currentPage).toBe(4);

      // Test last page (partial)
      const lastPageSize = totalChunks % pageSize || pageSize;
      const lastPageChunks = Array.from({ length: lastPageSize }, (_, i) => 
        createMockChunk(`chunk-${(8 - 1) * pageSize + i + 1}`, `Content ${(8 - 1) * pageSize + i + 1}`, documentId)
      );

      const lastPageResult: DocumentChunksResult = {
        chunks: lastPageChunks,
        pagination: {
          currentPage: 8,
          totalPages: Math.ceil(totalChunks / pageSize),
          totalChunks,
          pageSize
        },
        documentMetadata: firstPageResult.documentMetadata
      };

      mockFetch.mockResolvedValueOnce(createMockResponse(lastPageResult));

      const lastPage = await client.getChunksByDocumentId(documentId, { page: 8, pageSize });
      expect(lastPage.chunks).toHaveLength(lastPageSize);
      expect(lastPage.pagination.currentPage).toBe(8);
    });

    it('should handle sorting and filtering options', async () => {
      const documentId = 'sorted-doc-123';
      
      // Test sorting by creation date descending
      const sortedChunks = [
        createMockChunk('chunk-3', 'Latest content', documentId, new Date('2023-01-03')),
        createMockChunk('chunk-2', 'Middle content', documentId, new Date('2023-01-02')),
        createMockChunk('chunk-1', 'Oldest content', documentId, new Date('2023-01-01'))
      ];

      const sortedResult: DocumentChunksResult = {
        chunks: sortedChunks,
        pagination: {
          currentPage: 1,
          totalPages: 1,
          totalChunks: 3,
          pageSize: 20
        },
        documentMetadata: {
          originalFilePath: 'sorted-document.md',
          totalChunks: 3,
          documentScope: 'file' as DocumentScope,
          lastModified: new Date('2023-01-03')
        }
      };

      mockFetch.mockResolvedValueOnce(createMockResponse(sortedResult));

      const result = await client.getChunksByDocumentId(documentId, {
        sortBy: 'created',
        sortOrder: 'desc',
        includeHierarchy: true
      });

      expect(result.chunks).toHaveLength(3);
      expect(new Date(result.chunks[0].createdTime).getTime()).toBeGreaterThan(
        new Date(result.chunks[1].createdTime).getTime()
      );

      // Check that the URL contains all expected parameters (order may vary)
      const lastCall = mockFetch.mock.calls[mockFetch.mock.calls.length - 1];
      const calledUrl = lastCall[0] as string;
      
      expect(calledUrl).toContain(`/api/documents/${encodeURIComponent(documentId)}/chunks`);
      expect(calledUrl).toContain('sortBy=created');
      expect(calledUrl).toContain('sortOrder=desc');
      expect(calledUrl).toContain('includeHierarchy=true');
      
      expect(lastCall[1]).toEqual(expect.objectContaining({
        method: 'GET'
      }));
    });
  });

  describe('Multi-Source Virtual Documents', () => {
    it('should handle different source types correctly', async () => {
      const sourceTypes = ['remnote', 'logseq', 'obsidian-template'] as const;
      
      for (const sourceType of sourceTypes) {
        const context: VirtualDocumentContext = {
          sourceType,
          contextId: `${sourceType}-context-123`,
          pageTitle: `${sourceType} Test Page`,
          metadata: {
            sourceSpecific: `${sourceType}-specific-data`
          }
        };

        const mockDocument: VirtualDocument = {
          virtualDocumentId: `virtual-${sourceType}-doc-123`,
          context,
          chunkIds: [`${sourceType}-chunk-1`, `${sourceType}-chunk-2`],
          createdAt: new Date('2023-01-01'),
          lastUpdated: new Date('2023-01-02')
        };

        mockFetch.mockResolvedValueOnce(createMockResponse(mockDocument));

        const result = await client.createVirtualDocument(context);
        
        expect(result.virtualDocumentId).toBe(`virtual-${sourceType}-doc-123`);
        expect(result.context.sourceType).toBe(sourceType);
        expect(result.chunkIds).toHaveLength(2);
      }
    });
  });

  describe('Document Scope Management', () => {
    it('should handle document scope transitions', async () => {
      const chunkId = 'transition-chunk-123';
      const documentId = 'transition-doc-456';
      const scopes: DocumentScope[] = ['file', 'virtual', 'page'];

      for (const scope of scopes) {
        mockFetch.mockResolvedValueOnce(createMockResponse({}));
        
        await client.updateDocumentScope(chunkId, documentId, scope);
        
        expect(mockFetch).toHaveBeenLastCalledWith(
          `${baseUrl}/api/chunks/${encodeURIComponent(chunkId)}/document-scope`,
          expect.objectContaining({
            method: 'PUT',
            body: JSON.stringify({
              documentId,
              scope
            })
          })
        );
      }
    });
  });

  describe('Error Recovery and Resilience', () => {
    it('should handle API errors gracefully', async () => {
      const documentId = 'error-test-doc';
      
      mockFetch.mockResolvedValueOnce(createMockResponse(
        { error: 'Document not found' },
        { status: 404, statusText: 'Not Found' }
      ));

      await expect(client.getChunksByDocumentId(documentId)).rejects.toThrow();
    });
  });

  describe('Performance and Scalability', () => {
    it('should handle concurrent requests efficiently', async () => {
      const documentIds = Array.from({ length: 5 }, (_, i) => `concurrent-doc-${i}`);
      
      // Mock responses for all concurrent requests
      documentIds.forEach((docId, index) => {
        const mockResult: DocumentChunksResult = {
          chunks: [createMockChunk(`chunk-${index}`, `Content ${index}`, docId)],
          pagination: {
            currentPage: 1,
            totalPages: 1,
            totalChunks: 1,
            pageSize: 20
          },
          documentMetadata: {
            originalFilePath: `doc-${index}.md`,
            totalChunks: 1,
            documentScope: 'file' as DocumentScope,
            lastModified: new Date()
          }
        };
        mockFetch.mockResolvedValueOnce(createMockResponse(mockResult));
      });

      // Execute all requests concurrently
      const promises = documentIds.map(docId => 
        client.getChunksByDocumentId(docId)
      );

      const results = await Promise.all(promises);
      
      expect(results).toHaveLength(5);
      expect(mockFetch).toHaveBeenCalledTimes(5);
      
      // Verify each result corresponds to the correct document
      results.forEach((result, index) => {
        expect(result.chunks[0].chunkId).toBe(`chunk-${index}`);
        expect(result.chunks[0].documentId).toBe(`concurrent-doc-${index}`);
      });
    }, 10000);
  });
});

// Helper functions
function createMockChunk(
  chunkId: string, 
  content: string, 
  documentId: string, 
  createdTime: Date = new Date('2023-01-01')
): UnifiedChunk {
  return {
    chunkId,
    contents: content,
    isPage: false,
    isTag: false,
    isTemplate: false,
    isSlot: false,
    tags: ['test'],
    metadata: {},
    createdTime,
    lastUpdated: new Date('2023-01-02'),
    position: {
      fileName: 'test.md',
      lineStart: 1,
      lineEnd: 1,
      charStart: 0,
      charEnd: content.length
    },
    filePath: 'test.md',
    obsidianMetadata: {
      properties: {},
      frontmatter: {},
      aliases: [],
      cssClasses: []
    },
    documentId,
    documentScope: 'file' as DocumentScope
  };
}

function createMockResponse(
  data: any,
  options: { status?: number; statusText?: string; headers?: Record<string, string> } = {}
): Response {
  const { status = 200, statusText = 'OK', headers = {} } = options;
  
  const response = {
    ok: status >= 200 && status < 300,
    status,
    statusText,
    headers: new Map(Object.entries({
      'content-type': 'application/json',
      ...headers
    })),
    json: jest.fn().mockResolvedValue(data),
    text: jest.fn().mockResolvedValue(JSON.stringify(data))
  } as any;

  // Add forEach method to headers for compatibility
  response.headers.forEach = function(callback: (value: string, key: string) => void) {
    for (const [key, value] of this.entries()) {
      callback(value, key);
    }
  };

  return response;
}