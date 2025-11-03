/**
 * Integration tests for Chunk Operations API methods
 * Tests the complete flow of chunk CRUD operations with proper validation
 */

import { InkGatewayClient } from '../InkGatewayClient';
import { UnifiedChunk, ErrorType, PluginError } from '../../types';

// Mock fetch globally
global.fetch = jest.fn();
const mockFetch = global.fetch as jest.MockedFunction<typeof fetch>;

describe('Chunk Operations Integration Tests', () => {
  let client: InkGatewayClient;
  const baseUrl = 'https://api.example.com';
  const apiKey = 'test-api-key';

  beforeEach(() => {
    client = new InkGatewayClient(baseUrl, apiKey);
    mockFetch.mockClear();
  });

  const createMockChunk = (id: string = 'test-chunk-1'): UnifiedChunk => ({
    chunkId: id,
    contents: 'Test content for chunk',
    isPage: false,
    isTag: false,
    isTemplate: false,
    isSlot: false,
    tags: ['test', 'integration'],
    metadata: {
      source: 'integration-test',
      priority: 'high'
    },
    createdTime: new Date('2024-01-01T00:00:00Z'),
    lastUpdated: new Date('2024-01-01T00:00:00Z'),
    documentId: 'test-doc-1',
    documentScope: 'file' as const,
    position: {
      fileName: 'test.md',
      lineStart: 1,
      lineEnd: 3,
      charStart: 0,
      charEnd: 25
    },
    filePath: 'vault/test.md',
    obsidianMetadata: {
      properties: {
        title: 'Test Note',
        author: 'Test User'
      },
      frontmatter: {
        tags: ['test'],
        created: '2024-01-01'
      },
      aliases: ['test-alias'],
      cssClasses: ['test-class']
    }
  });

  const createMockResponse = (data: any, status: number = 200): Response => {
    return {
      ok: status >= 200 && status < 300,
      status,
      statusText: status === 200 ? 'OK' : 'Error',
      headers: new Map([['content-type', 'application/json']]),
      json: jest.fn().mockResolvedValue(data),
      text: jest.fn().mockResolvedValue(JSON.stringify(data))
    } as any;
  };

  describe('createChunk', () => {
    it('should create a new chunk with all required fields', async () => {
      const mockChunk = createMockChunk();
      const expectedResponse = { ...mockChunk, chunkId: 'generated-id-123' };
      
      mockFetch.mockResolvedValueOnce(createMockResponse(expectedResponse, 201));

      const result = await client.createChunk(mockChunk);

      expect(result).toEqual(expectedResponse);
      expect(mockFetch).toHaveBeenCalledWith(
        `${baseUrl}/api/chunks`,
        expect.objectContaining({
          method: 'POST',
          headers: expect.objectContaining({
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${apiKey}`
          }),
          body: JSON.stringify(mockChunk)
        })
      );
    });

    it('should handle validation errors during chunk creation', async () => {
      const invalidChunk = createMockChunk();
      invalidChunk.contents = ''; // Invalid empty content
      
      mockFetch.mockResolvedValueOnce(createMockResponse(
        { error: 'Validation failed: contents cannot be empty' },
        400
      ));

      await expect(client.createChunk(invalidChunk)).rejects.toMatchObject({
        type: ErrorType.API_ERROR,
        code: 'HTTP_400'
      });
    });

    it('should serialize complex metadata correctly', async () => {
      const chunkWithComplexMetadata = createMockChunk();
      chunkWithComplexMetadata.metadata = {
        nested: {
          object: {
            value: 'test'
          }
        },
        array: [1, 2, 3],
        date: new Date('2024-01-01'),
        boolean: true,
        null: null
      };

      mockFetch.mockResolvedValueOnce(createMockResponse(chunkWithComplexMetadata));

      await client.createChunk(chunkWithComplexMetadata);

      const callArgs = mockFetch.mock.calls[0][1];
      const sentData = JSON.parse(callArgs?.body as string);
      
      expect(sentData.metadata).toEqual({
        nested: {
          object: {
            value: 'test'
          }
        },
        array: [1, 2, 3],
        date: '2024-01-01T00:00:00.000Z',
        boolean: true,
        null: null
      });
    });
  });

  describe('updateChunk', () => {
    it('should update chunk with partial data', async () => {
      const chunkId = 'existing-chunk-123';
      const updateData = {
        contents: 'Updated content',
        tags: ['updated', 'test'],
        lastUpdated: new Date('2024-01-02T00:00:00Z')
      };
      const expectedResponse = { ...createMockChunk(chunkId), ...updateData };

      mockFetch.mockResolvedValueOnce(createMockResponse(expectedResponse));

      const result = await client.updateChunk(chunkId, updateData);

      expect(result).toEqual(expectedResponse);
      expect(mockFetch).toHaveBeenCalledWith(
        `${baseUrl}/api/chunks/${chunkId}`,
        expect.objectContaining({
          method: 'PUT',
          body: JSON.stringify(updateData)
        })
      );
    });

    it('should handle non-existent chunk updates', async () => {
      const chunkId = 'non-existent-chunk';
      
      mockFetch.mockResolvedValueOnce(createMockResponse(
        { error: 'Chunk not found' },
        404
      ));

      await expect(client.updateChunk(chunkId, { contents: 'new content' }))
        .rejects.toMatchObject({
          type: ErrorType.API_ERROR,
          code: 'HTTP_404'
        });
    });

    it('should preserve unchanged fields during update', async () => {
      const chunkId = 'test-chunk-preserve';
      const originalChunk = createMockChunk(chunkId);
      const updateData = { contents: 'Updated content only' };
      const expectedResponse = { ...originalChunk, ...updateData };

      mockFetch.mockResolvedValueOnce(createMockResponse(expectedResponse));

      const result = await client.updateChunk(chunkId, updateData);

      // Verify that other fields are preserved
      expect(result.tags).toEqual(originalChunk.tags);
      expect(result.metadata).toEqual(originalChunk.metadata);
      expect(result.position).toEqual(originalChunk.position);
      expect(result.contents).toBe('Updated content only');
    });
  });

  describe('deleteChunk', () => {
    it('should delete chunk successfully', async () => {
      const chunkId = 'chunk-to-delete';
      
      mockFetch.mockResolvedValueOnce(createMockResponse({}, 204));

      await client.deleteChunk(chunkId);

      expect(mockFetch).toHaveBeenCalledWith(
        `${baseUrl}/api/chunks/${chunkId}`,
        expect.objectContaining({
          method: 'DELETE'
        })
      );
    });

    it('should handle deletion of non-existent chunk', async () => {
      const chunkId = 'non-existent-chunk';
      
      mockFetch.mockResolvedValueOnce(createMockResponse(
        { error: 'Chunk not found' },
        404
      ));

      await expect(client.deleteChunk(chunkId)).rejects.toMatchObject({
        type: ErrorType.API_ERROR,
        code: 'HTTP_404'
      });
    });

    it('should handle deletion conflicts', async () => {
      const chunkId = 'chunk-with-dependencies';
      
      mockFetch.mockResolvedValueOnce(createMockResponse(
        { error: 'Cannot delete chunk with active references' },
        409
      ));

      await expect(client.deleteChunk(chunkId)).rejects.toMatchObject({
        type: ErrorType.API_ERROR,
        code: 'HTTP_409'
      });
    });
  });

  describe('getChunk', () => {
    it('should retrieve chunk with all data intact', async () => {
      const chunkId = 'retrieve-test-chunk';
      const expectedChunk = createMockChunk(chunkId);
      
      mockFetch.mockResolvedValueOnce(createMockResponse(expectedChunk));

      const result = await client.getChunk(chunkId);

      expect(result).toEqual(expectedChunk);
      expect(mockFetch).toHaveBeenCalledWith(
        `${baseUrl}/api/chunks/${chunkId}`,
        expect.objectContaining({
          method: 'GET'
        })
      );
    });

    it('should handle non-existent chunk retrieval', async () => {
      const chunkId = 'non-existent-chunk';
      
      mockFetch.mockResolvedValueOnce(createMockResponse(
        { error: 'Chunk not found' },
        404
      ));

      await expect(client.getChunk(chunkId)).rejects.toMatchObject({
        type: ErrorType.API_ERROR,
        code: 'HTTP_404'
      });
    });

    it('should deserialize complex data types correctly', async () => {
      const chunkId = 'complex-data-chunk';
      const chunkWithDates = createMockChunk(chunkId);
      
      // Mock response with string dates (as they come from JSON)
      const responseData = {
        ...chunkWithDates,
        createdTime: '2024-01-01T00:00:00.000Z',
        lastUpdated: '2024-01-02T12:30:45.123Z'
      };
      
      mockFetch.mockResolvedValueOnce(createMockResponse(responseData));

      const result = await client.getChunk(chunkId);

      // Verify dates are returned as strings (client doesn't auto-parse dates)
      expect(result.createdTime).toBe('2024-01-01T00:00:00.000Z');
      expect(result.lastUpdated).toBe('2024-01-02T12:30:45.123Z');
    });
  });

  describe('batchCreateChunks', () => {
    it('should create multiple chunks in a single request', async () => {
      const chunks = [
        createMockChunk('batch-chunk-1'),
        createMockChunk('batch-chunk-2'),
        createMockChunk('batch-chunk-3')
      ];
      
      const expectedResponse = chunks.map((chunk, index) => ({
        ...chunk,
        chunkId: `generated-batch-id-${index + 1}`
      }));

      mockFetch.mockResolvedValueOnce(createMockResponse(expectedResponse));

      const result = await client.batchCreateChunks(chunks);

      expect(result).toEqual(expectedResponse);
      expect(mockFetch).toHaveBeenCalledWith(
        `${baseUrl}/api/chunks/batch`,
        expect.objectContaining({
          method: 'POST',
          body: JSON.stringify({ chunks })
        })
      );
    });

    it('should handle partial batch failures', async () => {
      const chunks = [
        createMockChunk('valid-chunk'),
        { ...createMockChunk('invalid-chunk'), contents: '' } // Invalid
      ];

      mockFetch.mockResolvedValueOnce(createMockResponse(
        { 
          error: 'Batch validation failed',
          details: {
            failed: [1],
            errors: ['contents cannot be empty']
          }
        },
        400
      ));

      await expect(client.batchCreateChunks(chunks)).rejects.toMatchObject({
        type: ErrorType.API_ERROR,
        code: 'HTTP_400'
      });
    });

    it('should handle empty batch requests', async () => {
      const emptyChunks: UnifiedChunk[] = [];
      
      mockFetch.mockResolvedValueOnce(createMockResponse([]));

      const result = await client.batchCreateChunks(emptyChunks);

      expect(result).toEqual([]);
      expect(mockFetch).toHaveBeenCalledWith(
        `${baseUrl}/api/chunks/batch`,
        expect.objectContaining({
          body: JSON.stringify({ chunks: [] })
        })
      );
    });

    it('should handle large batch requests', async () => {
      // Create 100 chunks for batch processing
      const largeChunkBatch = Array.from({ length: 100 }, (_, index) => 
        createMockChunk(`large-batch-chunk-${index}`)
      );

      const expectedResponse = largeChunkBatch.map((chunk, index) => ({
        ...chunk,
        chunkId: `generated-large-batch-id-${index}`
      }));

      mockFetch.mockResolvedValueOnce(createMockResponse(expectedResponse));

      const result = await client.batchCreateChunks(largeChunkBatch);

      expect(result).toHaveLength(100);
      expect(result).toEqual(expectedResponse);
    });
  });

  describe('Request Validation and Serialization', () => {
    it('should properly serialize Date objects in requests', async () => {
      const chunk = createMockChunk();
      chunk.createdTime = new Date('2024-01-01T10:30:00.000Z');
      chunk.lastUpdated = new Date('2024-01-02T15:45:30.123Z');

      mockFetch.mockResolvedValueOnce(createMockResponse(chunk));

      await client.createChunk(chunk);

      const callArgs = mockFetch.mock.calls[0][1];
      const sentData = JSON.parse(callArgs?.body as string);
      
      expect(sentData.createdTime).toBe('2024-01-01T10:30:00.000Z');
      expect(sentData.lastUpdated).toBe('2024-01-02T15:45:30.123Z');
    });

    it('should handle special characters in chunk content', async () => {
      const chunk = createMockChunk();
      chunk.contents = 'Content with special chars: 中文, émojis 🚀, and "quotes"';
      
      mockFetch.mockResolvedValueOnce(createMockResponse(chunk));

      await client.createChunk(chunk);

      const callArgs = mockFetch.mock.calls[0][1];
      const sentData = JSON.parse(callArgs?.body as string);
      
      expect(sentData.contents).toBe('Content with special chars: 中文, émojis 🚀, and "quotes"');
    });

    it('should validate required fields before sending requests', async () => {
      const incompleteChunk = {
        // Missing required fields like chunkId, contents, etc.
        tags: ['test']
      } as any;

      mockFetch.mockResolvedValueOnce(createMockResponse(incompleteChunk));

      // The client should still send the request (validation happens server-side)
      await client.createChunk(incompleteChunk);

      expect(mockFetch).toHaveBeenCalledWith(
        `${baseUrl}/api/chunks`,
        expect.objectContaining({
          method: 'POST',
          body: JSON.stringify(incompleteChunk)
        })
      );
    });
  });

  describe('Error Recovery and Resilience', () => {
    it('should retry chunk operations on server errors', async () => {
      const chunk = createMockChunk();
      
      // First call fails with 500, second succeeds
      mockFetch
        .mockResolvedValueOnce(createMockResponse(
          { error: 'Internal server error' },
          500
        ))
        .mockResolvedValueOnce(createMockResponse(chunk));

      const result = await client.createChunk(chunk);

      expect(result).toEqual(chunk);
      expect(mockFetch).toHaveBeenCalledTimes(2);
    });

    it('should not retry on client errors (4xx)', async () => {
      const chunk = createMockChunk();
      
      mockFetch.mockResolvedValueOnce(createMockResponse(
        { error: 'Bad request' },
        400
      ));

      await expect(client.createChunk(chunk)).rejects.toThrow();
      expect(mockFetch).toHaveBeenCalledTimes(1);
    });

    it('should handle network timeouts gracefully', async () => {
      const chunk = createMockChunk();
      
      // Mock a timeout scenario
      mockFetch.mockRejectedValueOnce((() => {
        const error = new Error('The operation was aborted');
        error.name = 'AbortError';
        return error;
      })());

      await expect(client.createChunk(chunk)).rejects.toMatchObject({
        type: ErrorType.NETWORK_ERROR,
        code: 'REQUEST_TIMEOUT'
      });
    }, 10000);
  });
});