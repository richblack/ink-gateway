/**
 * Integration tests for Document ID Pagination functionality
 */

import { InkGatewayClient } from '../InkGatewayClient';
import {
  VirtualDocumentContext,
  VirtualDocument,
  DocumentChunksResult,
  PaginationOptions,
  DocumentScope,
  PluginError,
  ErrorType
} from '../../types';

// Mock fetch for testing
global.fetch = jest.fn();

describe('InkGatewayClient - Document ID Pagination Integration', () => {
  let client: InkGatewayClient;
  const mockFetch = fetch as jest.MockedFunction<typeof fetch>;

  beforeEach(() => {
    client = new InkGatewayClient('http://localhost:8080', 'test-api-key', {
      timeout: 1000, // Shorter timeout for tests
      retryConfig: {
        maxRetries: 1, // Fewer retries for faster tests
        baseDelay: 10,
        maxDelay: 100,
        backoffFactor: 1.5,
        retryableStatusCodes: [408, 429, 500, 502, 503, 504]
      }
    });
    mockFetch.mockClear();
  });

  afterEach(() => {
    jest.clearAllMocks();
  });

  describe('getChunksByDocumentId', () => {
    const mockDocumentId = 'file_abc123_notes_test_md';
    const mockResponse: DocumentChunksResult = {
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
          createdTime: new Date('2024-01-01T00:00:00Z'),
          lastUpdated: new Date('2024-01-01T00:00:00Z')
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
        lastModified: new Date('2024-01-01T00:00:00Z')
      }
    };

    it('should retrieve chunks by document ID without options', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        status: 200,
        statusText: 'OK',
        headers: new Headers({ 'content-type': 'application/json' }),
        json: async () => mockResponse
      } as Response);

      const result = await client.getChunksByDocumentId(mockDocumentId);

      expect(result).toEqual(mockResponse);
      expect(mockFetch).toHaveBeenCalledWith(
        `http://localhost:8080/api/documents/${encodeURIComponent(mockDocumentId)}/chunks`,
        expect.objectContaining({
          method: 'GET',
          headers: expect.objectContaining({
            'Authorization': 'Bearer test-api-key',
            'Content-Type': 'application/json'
          })
        })
      );
    });

    it('should retrieve chunks with pagination options', async () => {
      const options: PaginationOptions = {
        page: 2,
        pageSize: 20,
        includeHierarchy: true,
        sortBy: 'position',
        sortOrder: 'asc'
      };

      mockFetch.mockResolvedValueOnce({
        ok: true,
        status: 200,
        statusText: 'OK',
        headers: new Headers({ 'content-type': 'application/json' }),
        json: async () => mockResponse
      } as Response);

      await client.getChunksByDocumentId(mockDocumentId, options);

      const expectedUrl = `http://localhost:8080/api/documents/${encodeURIComponent(mockDocumentId)}/chunks?page=2&pageSize=20&includeHierarchy=true&sortBy=position&sortOrder=asc`;
      expect(mockFetch).toHaveBeenCalledWith(
        expectedUrl,
        expect.objectContaining({
          method: 'GET'
        })
      );
    });

    it('should handle partial pagination options', async () => {
      const options: PaginationOptions = {
        page: 1,
        sortBy: 'created'
      };

      mockFetch.mockResolvedValueOnce({
        ok: true,
        status: 200,
        statusText: 'OK',
        headers: new Headers({ 'content-type': 'application/json' }),
        json: async () => mockResponse
      } as Response);

      await client.getChunksByDocumentId(mockDocumentId, options);

      const expectedUrl = `http://localhost:8080/api/documents/${encodeURIComponent(mockDocumentId)}/chunks?page=1&sortBy=created`;
      expect(mockFetch).toHaveBeenCalledWith(
        expectedUrl,
        expect.objectContaining({
          method: 'GET'
        })
      );
    });

    it('should validate document ID parameter', async () => {
      await expect(client.getChunksByDocumentId('')).rejects.toThrow(PluginError);
      await expect(client.getChunksByDocumentId(null as any)).rejects.toThrow(PluginError);
      await expect(client.getChunksByDocumentId(undefined as any)).rejects.toThrow(PluginError);
    });

    it('should validate pagination options', async () => {
      const invalidOptions = [
        { page: 0 },
        { page: -1 },
        { page: 1.5 },
        { pageSize: 0 },
        { pageSize: -1 },
        { pageSize: 1001 },
        { sortBy: 'invalid' },
        { sortOrder: 'invalid' }
      ];

      for (const options of invalidOptions) {
        await expect(client.getChunksByDocumentId(mockDocumentId, options as any))
          .rejects.toThrow(PluginError);
      }
    });

    it('should handle HTTP errors', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 404,
        statusText: 'Not Found',
        headers: new Headers()
      } as Response);

      await expect(client.getChunksByDocumentId(mockDocumentId))
        .rejects.toThrow(PluginError);
    });

    it('should handle network errors', async () => {
      mockFetch.mockRejectedValueOnce(new Error('Network error'));

      await expect(client.getChunksByDocumentId(mockDocumentId))
        .rejects.toThrow(PluginError);
    }, 10000);

    it('should retry on retryable errors', async () => {
      // First call fails with 500
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 500,
        statusText: 'Internal Server Error',
        headers: new Headers()
      } as Response);

      // Second call succeeds
      mockFetch.mockResolvedValueOnce({
        ok: true,
        status: 200,
        statusText: 'OK',
        headers: new Headers({ 'content-type': 'application/json' }),
        json: async () => mockResponse
      } as Response);

      const result = await client.getChunksByDocumentId(mockDocumentId);

      expect(result).toEqual(mockResponse);
      expect(mockFetch).toHaveBeenCalledTimes(2);
    });
  });

  describe('createVirtualDocument', () => {
    const mockContext: VirtualDocumentContext = {
      sourceType: 'remnote',
      contextId: 'page123',
      pageTitle: 'Test Page',
      metadata: { category: 'notes' }
    };

    const mockResponse: VirtualDocument = {
      virtualDocumentId: 'virtual_remnote_abc123_page123',
      context: mockContext,
      chunkIds: ['chunk1', 'chunk2'],
      createdAt: new Date('2024-01-01T00:00:00Z'),
      lastUpdated: new Date('2024-01-01T00:00:00Z')
    };

    it('should create virtual document', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        status: 201,
        statusText: 'Created',
        headers: new Headers({ 'content-type': 'application/json' }),
        json: async () => mockResponse
      } as Response);

      const result = await client.createVirtualDocument(mockContext);

      expect(result).toEqual(mockResponse);
      expect(mockFetch).toHaveBeenCalledWith(
        'http://localhost:8080/api/documents/virtual',
        expect.objectContaining({
          method: 'POST',
          headers: expect.objectContaining({
            'Authorization': 'Bearer test-api-key',
            'Content-Type': 'application/json'
          }),
          body: JSON.stringify(mockContext)
        })
      );
    });

    it('should validate virtual document context', async () => {
      const invalidContexts = [
        null,
        undefined,
        {},
        { sourceType: 'remnote' }, // missing contextId
        { contextId: 'page123' }, // missing sourceType
        { sourceType: '', contextId: 'page123' }, // empty sourceType
        { sourceType: 'remnote', contextId: '' }, // empty contextId
        { sourceType: 'invalid', contextId: 'page123' } // invalid sourceType
      ];

      for (const context of invalidContexts) {
        await expect(client.createVirtualDocument(context as any))
          .rejects.toThrow(PluginError);
      }
    });

    it('should handle valid source types', async () => {
      const validSourceTypes = ['remnote', 'logseq', 'obsidian-template'];

      for (const sourceType of validSourceTypes) {
        const context: VirtualDocumentContext = {
          sourceType: sourceType as any,
          contextId: 'test123',
          metadata: {}
        };

        mockFetch.mockResolvedValueOnce({
          ok: true,
          status: 201,
          statusText: 'Created',
          headers: new Headers({ 'content-type': 'application/json' }),
          json: async () => ({ ...mockResponse, context })
        } as Response);

        await expect(client.createVirtualDocument(context)).resolves.toBeDefined();
      }
    });

    it('should handle HTTP errors', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 400,
        statusText: 'Bad Request',
        headers: new Headers()
      } as Response);

      await expect(client.createVirtualDocument(mockContext))
        .rejects.toThrow(PluginError);
    });
  });

  describe('updateDocumentScope', () => {
    const mockChunkId = 'chunk123';
    const mockDocumentId = 'file_abc123_notes_test_md';
    const mockScope: DocumentScope = 'virtual';

    it('should update document scope', async () => {
      mockFetch.mockImplementationOnce(() => 
        Promise.resolve({
          ok: true,
          status: 200,
          statusText: 'OK',
          headers: new Headers(),
          text: () => Promise.resolve(''),
          json: () => Promise.resolve({})
        } as Response)
      );

      await client.updateDocumentScope(mockChunkId, mockDocumentId, mockScope);

      expect(mockFetch).toHaveBeenCalledWith(
        `http://localhost:8080/api/chunks/${encodeURIComponent(mockChunkId)}/document-scope`,
        expect.objectContaining({
          method: 'PUT',
          headers: expect.objectContaining({
            'Authorization': 'Bearer test-api-key',
            'Content-Type': 'application/json'
          }),
          body: JSON.stringify({
            documentId: mockDocumentId,
            scope: mockScope
          })
        })
      );
    }, 5000);

    it('should validate parameters', async () => {
      const invalidParams = [
        ['', mockDocumentId, mockScope], // empty chunkId
        [null, mockDocumentId, mockScope], // null chunkId
        [mockChunkId, '', mockScope], // empty documentId
        [mockChunkId, null, mockScope], // null documentId
        [mockChunkId, mockDocumentId, 'invalid'], // invalid scope
      ];

      for (const [chunkId, documentId, scope] of invalidParams) {
        await expect(client.updateDocumentScope(chunkId as any, documentId as any, scope as any))
          .rejects.toThrow(PluginError);
      }
    });

    it('should handle valid document scopes', async () => {
      const validScopes: DocumentScope[] = ['file', 'virtual', 'page'];

      for (const scope of validScopes) {
        mockFetch.mockImplementationOnce(() => 
          Promise.resolve({
            ok: true,
            status: 200,
            statusText: 'OK',
            headers: new Headers(),
            text: () => Promise.resolve(''),
            json: () => Promise.resolve({})
          } as Response)
        );

        await expect(client.updateDocumentScope(mockChunkId, mockDocumentId, scope))
          .resolves.toBeUndefined();
      }
    }, 5000);

    it('should handle HTTP errors', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 404,
        statusText: 'Not Found',
        headers: new Headers()
      } as Response);

      await expect(client.updateDocumentScope(mockChunkId, mockDocumentId, mockScope))
        .rejects.toThrow(PluginError);
    });
  });

  describe('error handling and retry logic', () => {
    const mockDocumentId = 'test-doc-id';

    it('should retry on network timeout', async () => {
      // First call times out
      mockFetch.mockImplementationOnce(() => 
        new Promise((_, reject) => 
          setTimeout(() => reject(new Error('AbortError')), 100)
        )
      );

      // Second call succeeds
      mockFetch.mockResolvedValueOnce({
        ok: true,
        status: 200,
        statusText: 'OK',
        headers: new Headers({ 'content-type': 'application/json' }),
        json: async () => ({ chunks: [], pagination: {}, documentMetadata: {} })
      } as Response);

      await client.getChunksByDocumentId(mockDocumentId);

      expect(mockFetch).toHaveBeenCalledTimes(2);
    });

    it('should not retry on non-retryable errors', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 400,
        statusText: 'Bad Request',
        headers: new Headers()
      } as Response);

      await expect(client.getChunksByDocumentId(mockDocumentId))
        .rejects.toThrow(PluginError);

      expect(mockFetch).toHaveBeenCalledTimes(1);
    });

    it('should respect maximum retry attempts', async () => {
      // Create a client with more retries for this specific test
      const retryClient = new InkGatewayClient('http://localhost:8080', 'test-api-key', {
        timeout: 1000,
        retryConfig: {
          maxRetries: 3,
          baseDelay: 10,
          maxDelay: 100,
          backoffFactor: 1.5,
          retryableStatusCodes: [408, 429, 500, 502, 503, 504]
        }
      });

      // All calls fail with 500
      mockFetch.mockResolvedValue({
        ok: false,
        status: 500,
        statusText: 'Internal Server Error',
        headers: new Headers()
      } as Response);

      await expect(retryClient.getChunksByDocumentId(mockDocumentId))
        .rejects.toThrow(PluginError);

      // Should be called 4 times (1 initial + 3 retries)
      expect(mockFetch).toHaveBeenCalledTimes(4);
    }, 15000);
  });

  describe('request history and debugging', () => {
    it('should track request history', async () => {
      const mockDocumentId = 'test-doc-id';
      
      mockFetch.mockResolvedValueOnce({
        ok: true,
        status: 200,
        statusText: 'OK',
        headers: new Headers({ 'content-type': 'application/json' }),
        json: async () => ({ chunks: [], pagination: {}, documentMetadata: {} })
      } as Response);

      await client.getChunksByDocumentId(mockDocumentId);

      const history = client.getRequestHistory();
      expect(history).toHaveLength(1);
      expect(history[0].success).toBe(true);
      expect(history[0].config.endpoint).toContain('/api/documents/');
    });

    it('should track failed requests in history', async () => {
      const mockDocumentId = 'test-doc-id';
      
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 404,
        statusText: 'Not Found',
        headers: new Headers()
      } as Response);

      try {
        await client.getChunksByDocumentId(mockDocumentId);
      } catch (error) {
        // Expected to fail
      }

      const history = client.getRequestHistory();
      expect(history).toHaveLength(1);
      expect(history[0].success).toBe(false);
      expect(history[0].error).toBeDefined();
    });

    it('should clear request history', async () => {
      const mockDocumentId = 'test-doc-id';
      
      mockFetch.mockResolvedValueOnce({
        ok: true,
        status: 200,
        statusText: 'OK',
        headers: new Headers({ 'content-type': 'application/json' }),
        json: async () => ({ chunks: [], pagination: {}, documentMetadata: {} })
      } as Response);

      await client.getChunksByDocumentId(mockDocumentId);
      expect(client.getRequestHistory()).toHaveLength(1);

      client.clearRequestHistory();
      expect(client.getRequestHistory()).toHaveLength(0);
    });
  });
});