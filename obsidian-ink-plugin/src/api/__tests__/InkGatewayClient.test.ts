/**
 * Unit tests for InkGatewayClient
 * Tests HTTP client functionality, error handling, retry logic, and API methods
 */

import { InkGatewayClient } from '../InkGatewayClient';
import { UnifiedChunk, SearchQuery, ErrorType, PluginError } from '../../types';

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

describe('InkGatewayClient', () => {
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

  describe('Constructor and Configuration', () => {
    it('should initialize with correct default values', () => {
      const config = client.getConfiguration();
      expect(config.baseUrl).toBe(baseUrl);
      expect(config.timeout).toBe(30000);
      expect(config.retryConfig.maxRetries).toBe(3);
    });

    it('should remove trailing slash from baseUrl', () => {
      const clientWithSlash = new InkGatewayClient('https://api.example.com/', apiKey);
      const config = clientWithSlash.getConfiguration();
      expect(config.baseUrl).toBe('https://api.example.com');
    });

    it('should accept custom configuration options', () => {
      const customClient = new InkGatewayClient(baseUrl, apiKey, {
        timeout: 60000,
        retryConfig: {
          maxRetries: 5,
          baseDelay: 2000
        }
      });
      
      const config = customClient.getConfiguration();
      expect(config.timeout).toBe(60000);
      expect(config.retryConfig.maxRetries).toBe(5);
      expect(config.retryConfig.baseDelay).toBe(2000);
    });
  });

  describe('Request Interceptors', () => {
    it('should apply request interceptors', async () => {
      let interceptorCalled = false;
      client.addRequestInterceptor((config) => {
        interceptorCalled = true;
        config.headers = { ...config.headers, 'X-Custom': 'test' };
        return config;
      });

      mockFetch.mockResolvedValueOnce(createMockResponse({ success: true }));

      await client.healthCheck();

      expect(interceptorCalled).toBe(true);
      expect(mockFetch).toHaveBeenCalledWith(
        expect.any(String),
        expect.objectContaining({
          headers: expect.objectContaining({
            'X-Custom': 'test',
            'Authorization': `Bearer ${apiKey}`
          })
        })
      );
    });

    it('should apply multiple request interceptors in order', async () => {
      const order: number[] = [];
      
      client.addRequestInterceptor((config) => {
        order.push(1);
        return config;
      });
      
      client.addRequestInterceptor((config) => {
        order.push(2);
        return config;
      });

      mockFetch.mockResolvedValueOnce(createMockResponse({ success: true }));

      await client.healthCheck();

      expect(order).toEqual([1, 2]);
    });
  });

  describe('Response Interceptors', () => {
    it('should apply response interceptors', async () => {
      let interceptorCalled = false;
      client.addResponseInterceptor((response) => {
        interceptorCalled = true;
        return response;
      });

      mockFetch.mockResolvedValueOnce(createMockResponse({ success: true }));

      await client.healthCheck();

      expect(interceptorCalled).toBe(true);
    });
  });

  describe('Error Handling', () => {
    it('should handle network errors', async () => {
      mockFetch.mockRejectedValueOnce(new Error('Network error'));

      await expect(client.healthCheck()).rejects.toThrow(PluginError);
      
      // Reset mock for second assertion
      mockFetch.mockRejectedValueOnce(new Error('Network error'));
      await expect(client.healthCheck()).rejects.toMatchObject({
        type: ErrorType.NETWORK_ERROR,
        code: 'NETWORK_ERROR'
      });
    }, 10000);

    it('should handle HTTP errors', async () => {
      mockFetch.mockResolvedValueOnce(createMockResponse(
        { error: 'Not found' },
        { status: 404, statusText: 'Not Found' }
      ));

      await expect(client.healthCheck()).rejects.toThrow(PluginError);
      
      // Reset mock for second assertion
      mockFetch.mockResolvedValueOnce(createMockResponse(
        { error: 'Not found' },
        { status: 404, statusText: 'Not Found' }
      ));
      await expect(client.healthCheck()).rejects.toMatchObject({
        type: ErrorType.API_ERROR,
        code: 'HTTP_404'
      });
    });

    it('should handle timeout errors', async () => {
      // Mock AbortController for timeout simulation
      const mockAbortController = {
        abort: jest.fn(),
        signal: { aborted: false }
      };
      global.AbortController = jest.fn(() => mockAbortController) as any;

      mockFetch.mockImplementationOnce(() => {
        return new Promise((_, reject) => {
          setTimeout(() => {
            const error = new Error('The operation was aborted');
            error.name = 'AbortError';
            reject(error);
          }, 100);
        });
      });

      const promise = client.healthCheck();
      
      // Fast-forward time to trigger timeout
      jest.advanceTimersByTime(31000);
      
      await expect(promise).rejects.toMatchObject({
        type: ErrorType.NETWORK_ERROR,
        code: 'REQUEST_TIMEOUT'
      });
    }, 10000);

    it('should apply error interceptors', async () => {
      let interceptorCalled = false;
      client.addErrorInterceptor((error) => {
        interceptorCalled = true;
        return error;
      });

      mockFetch.mockRejectedValueOnce(new Error('Network error'));

      await expect(client.healthCheck()).rejects.toThrow();
      expect(interceptorCalled).toBe(true);
    }, 10000);
  });

  describe('Retry Logic', () => {
    it('should retry on retryable errors', async () => {
      // First call fails with 500, second succeeds
      mockFetch
        .mockResolvedValueOnce(createMockResponse(
          { error: 'Server error' },
          { status: 500, statusText: 'Internal Server Error' }
        ))
        .mockResolvedValueOnce(createMockResponse({ success: true }));

      const promise = client.healthCheck();
      
      // Fast-forward through retry delay
      jest.advanceTimersByTime(1000);
      
      const result = await promise;

      expect(result).toBe(true);
      expect(mockFetch).toHaveBeenCalledTimes(2);
    }, 10000);

    it('should not retry on non-retryable errors', async () => {
      mockFetch.mockResolvedValueOnce(createMockResponse(
        { error: 'Bad request' },
        { status: 400, statusText: 'Bad Request' }
      ));

      await expect(client.healthCheck()).rejects.toThrow();
      expect(mockFetch).toHaveBeenCalledTimes(1);
    });

    it('should respect maximum retry attempts', async () => {
      // All calls fail with 500
      mockFetch.mockResolvedValue(createMockResponse(
        { error: 'Server error' },
        { status: 500, statusText: 'Internal Server Error' }
      ));

      const promise = client.healthCheck();
      
      // Fast-forward through all retry delays
      jest.advanceTimersByTime(1000); // First retry
      jest.advanceTimersByTime(2000); // Second retry  
      jest.advanceTimersByTime(4000); // Third retry

      await expect(promise).rejects.toThrow();
      
      // Should be called 4 times: initial + 3 retries
      expect(mockFetch).toHaveBeenCalledTimes(4);
    });

    it('should use exponential backoff for retries', async () => {
      mockFetch.mockResolvedValue(createMockResponse(
        { error: 'Server error' },
        { status: 500, statusText: 'Internal Server Error' }
      ));

      const promise = client.healthCheck();
      
      // Fast-forward through retry delays
      jest.advanceTimersByTime(1000); // First retry delay
      jest.advanceTimersByTime(2000); // Second retry delay
      jest.advanceTimersByTime(4000); // Third retry delay
      
      await expect(promise).rejects.toThrow();
    }, 10000);
  });

  describe('Request History', () => {
    it('should track request history', async () => {
      mockFetch.mockResolvedValueOnce(createMockResponse({ success: true }));

      await client.healthCheck();

      const history = client.getRequestHistory();
      expect(history).toHaveLength(1);
      expect(history[0]).toMatchObject({
        success: true,
        config: expect.objectContaining({
          method: 'GET',
          endpoint: '/health'
        })
      });
    });

    it('should clear request history', async () => {
      mockFetch.mockResolvedValueOnce(createMockResponse({ success: true }));

      await client.healthCheck();
      expect(client.getRequestHistory()).toHaveLength(1);

      client.clearRequestHistory();
      expect(client.getRequestHistory()).toHaveLength(0);
    });

    it('should limit request history size', async () => {
      mockFetch.mockResolvedValue(createMockResponse({ success: true }));

      // Make 150 requests (more than the 100 limit)
      for (let i = 0; i < 150; i++) {
        await client.healthCheck();
      }

      const history = client.getRequestHistory();
      expect(history).toHaveLength(100);
    });
  });

  describe('Chunk Operations', () => {
    const mockChunk: UnifiedChunk = {
      chunkId: 'test-chunk-1',
      contents: 'Test content',
      isPage: false,
      isTag: false,
      isTemplate: false,
      isSlot: false,
      tags: ['test'],
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
        charEnd: 12
      },
      filePath: 'test.md',
      obsidianMetadata: {
        properties: {},
        frontmatter: {},
        aliases: [],
        cssClasses: []
      }
    };

    it('should create chunk', async () => {
      mockFetch.mockResolvedValueOnce(createMockResponse(mockChunk));

      const result = await client.createChunk(mockChunk);

      expect(result).toEqual(mockChunk);
      expect(mockFetch).toHaveBeenCalledWith(
        `${baseUrl}/api/chunks`,
        expect.objectContaining({
          method: 'POST',
          body: JSON.stringify(mockChunk)
        })
      );
    });

    it('should update chunk', async () => {
      const updatedChunk = { ...mockChunk, contents: 'Updated content' };
      mockFetch.mockResolvedValueOnce(createMockResponse(updatedChunk));

      const result = await client.updateChunk('test-chunk-1', { contents: 'Updated content' });

      expect(result).toEqual(updatedChunk);
      expect(mockFetch).toHaveBeenCalledWith(
        `${baseUrl}/api/chunks/test-chunk-1`,
        expect.objectContaining({
          method: 'PUT',
          body: JSON.stringify({ contents: 'Updated content' })
        })
      );
    });

    it('should delete chunk', async () => {
      mockFetch.mockResolvedValueOnce(createMockResponse({}));

      await client.deleteChunk('test-chunk-1');

      expect(mockFetch).toHaveBeenCalledWith(
        `${baseUrl}/api/chunks/test-chunk-1`,
        expect.objectContaining({
          method: 'DELETE'
        })
      );
    });

    it('should get chunk', async () => {
      mockFetch.mockResolvedValueOnce(createMockResponse(mockChunk));

      const result = await client.getChunk('test-chunk-1');

      expect(result).toEqual(mockChunk);
      expect(mockFetch).toHaveBeenCalledWith(
        `${baseUrl}/api/chunks/test-chunk-1`,
        expect.objectContaining({
          method: 'GET'
        })
      );
    });

    it('should batch create chunks', async () => {
      const chunks = [mockChunk, { ...mockChunk, chunkId: 'test-chunk-2' }];
      mockFetch.mockResolvedValueOnce(createMockResponse(chunks));

      const result = await client.batchCreateChunks(chunks);

      expect(result).toEqual(chunks);
      expect(mockFetch).toHaveBeenCalledWith(
        `${baseUrl}/api/chunks/batch`,
        expect.objectContaining({
          method: 'POST',
          body: JSON.stringify({ chunks })
        })
      );
    });
  });

  describe('Search Operations', () => {
    const mockSearchQuery: SearchQuery = {
      content: 'test query',
      searchType: 'semantic'
    };

    const mockSearchResult = {
      items: [],
      totalCount: 0,
      searchTime: 100,
      cacheHit: false
    };

    it('should search chunks', async () => {
      mockFetch.mockResolvedValueOnce(createMockResponse(mockSearchResult));

      const result = await client.searchChunks(mockSearchQuery);

      expect(result).toEqual(mockSearchResult);
      expect(mockFetch).toHaveBeenCalledWith(
        `${baseUrl}/api/search`,
        expect.objectContaining({
          method: 'POST',
          body: JSON.stringify(mockSearchQuery)
        })
      );
    });

    it('should search semantic', async () => {
      mockFetch.mockResolvedValueOnce(createMockResponse(mockSearchResult));

      const result = await client.searchSemantic('test content');

      expect(result).toEqual(mockSearchResult);
      expect(mockFetch).toHaveBeenCalledWith(
        `${baseUrl}/api/search/semantic`,
        expect.objectContaining({
          method: 'POST',
          body: JSON.stringify({ content: 'test content' })
        })
      );
    });

    it('should search by tags', async () => {
      mockFetch.mockResolvedValueOnce(createMockResponse(mockSearchResult));

      const result = await client.searchByTags(['tag1', 'tag2']);

      expect(result).toEqual(mockSearchResult);
      expect(mockFetch).toHaveBeenCalledWith(
        `${baseUrl}/api/search/tags`,
        expect.objectContaining({
          method: 'POST',
          body: JSON.stringify({ tags: ['tag1', 'tag2'] })
        })
      );
    });
  });

  describe('AI Operations', () => {
    it('should chat with AI', async () => {
      const mockResponse = {
        message: 'AI response',
        suggestions: [],
        actions: [],
        metadata: {
          model: 'gpt-4',
          tokens: 100,
          processingTime: 500,
          confidence: 0.9
        }
      };

      mockFetch.mockResolvedValueOnce(createMockResponse(mockResponse));

      const result = await client.chatWithAI('Hello AI', ['context1']);

      expect(result).toEqual(mockResponse);
      expect(mockFetch).toHaveBeenCalledWith(
        `${baseUrl}/api/ai/chat`,
        expect.objectContaining({
          method: 'POST',
          body: JSON.stringify({ message: 'Hello AI', context: ['context1'] })
        })
      );
    });

    it('should process content', async () => {
      const mockResult = {
        chunks: [],
        suggestions: [],
        improvements: []
      };

      mockFetch.mockResolvedValueOnce(createMockResponse(mockResult));

      const result = await client.processContent('Content to process');

      expect(result).toEqual(mockResult);
      expect(mockFetch).toHaveBeenCalledWith(
        `${baseUrl}/api/ai/process`,
        expect.objectContaining({
          method: 'POST',
          body: JSON.stringify({ content: 'Content to process' })
        })
      );
    });
  });

  describe('Configuration Updates', () => {
    it('should update base URL', () => {
      client.updateBaseUrl('https://new-api.example.com/');
      const config = client.getConfiguration();
      expect(config.baseUrl).toBe('https://new-api.example.com');
    });

    it('should update API key', () => {
      client.updateApiKey('new-api-key');
      
      // Verify by checking if the new key is used in requests
      client.addRequestInterceptor((config) => {
        expect(config.headers?.['Authorization']).toBe('Bearer new-api-key');
        return config;
      });

      mockFetch.mockResolvedValueOnce(createMockResponse({ success: true }));
      client.healthCheck();
    });

    it('should update timeout', () => {
      client.updateTimeout(60000);
      const config = client.getConfiguration();
      expect(config.timeout).toBe(60000);
    });

    it('should update retry config', () => {
      client.updateRetryConfig({ maxRetries: 5, baseDelay: 2000 });
      const config = client.getConfiguration();
      expect(config.retryConfig.maxRetries).toBe(5);
      expect(config.retryConfig.baseDelay).toBe(2000);
    });
  });
});

// Helper function to create mock fetch responses
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