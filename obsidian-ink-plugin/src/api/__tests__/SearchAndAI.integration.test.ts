/**
 * Integration tests for Search and AI Operations API methods
 * Tests search functionality, AI chat, content processing, and caching
 */

import { InkGatewayClient } from '../InkGatewayClient';
import { 
  SearchQuery, 
  SearchResult, 
  SearchResultItem, 
  AIResponse, 
  ProcessingResult,
  UnifiedChunk,
  ErrorType 
} from '../../types';

// Mock fetch globally
global.fetch = jest.fn();
const mockFetch = global.fetch as jest.MockedFunction<typeof fetch>;

describe('Search and AI Operations Integration Tests', () => {
  let client: InkGatewayClient;
  const baseUrl = 'https://api.example.com';
  const apiKey = 'test-api-key';

  beforeEach(() => {
    client = new InkGatewayClient(baseUrl, apiKey);
    mockFetch.mockClear();
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

  const createMockChunk = (id: string, content: string): UnifiedChunk => ({
    chunkId: id,
    contents: content,
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
      charEnd: content.length
    },
    filePath: 'test.md',
    obsidianMetadata: {
      properties: {},
      frontmatter: {},
      aliases: [],
      cssClasses: []
    }
  });

  const createMockSearchResult = (items: SearchResultItem[] = []): SearchResult => ({
    items,
    totalCount: items.length,
    searchTime: 150,
    cacheHit: false
  });

  const createMockSearchResultItem = (chunk: UnifiedChunk, score: number = 0.85): SearchResultItem => ({
    chunk,
    score,
    context: `...${chunk.contents.substring(0, 100)}...`,
    position: chunk.position,
    highlights: [
      { start: 0, end: 10, type: 'match' },
      { start: 20, end: 30, type: 'context' }
    ]
  });

  describe('searchChunks', () => {
    it('should perform semantic search with comprehensive query', async () => {
      const searchQuery: SearchQuery = {
        content: 'artificial intelligence machine learning',
        tags: ['ai', 'ml'],
        tagLogic: 'AND',
        searchType: 'semantic',
        filters: {
          dateRange: {
            start: new Date('2024-01-01'),
            end: new Date('2024-12-31')
          },
          minScore: 0.7,
          excludeTags: ['draft']
        }
      };

      const mockChunks = [
        createMockChunk('ai-chunk-1', 'Introduction to artificial intelligence and machine learning concepts'),
        createMockChunk('ai-chunk-2', 'Deep learning algorithms for neural networks')
      ];

      const mockResult = createMockSearchResult([
        createMockSearchResultItem(mockChunks[0], 0.92),
        createMockSearchResultItem(mockChunks[1], 0.78)
      ]);

      mockFetch.mockResolvedValueOnce(createMockResponse(mockResult));

      const result = await client.searchChunks(searchQuery);

      expect(result).toEqual(mockResult);
      expect(result.items).toHaveLength(2);
      expect(result.items[0].score).toBe(0.92);
      expect(result.totalCount).toBe(2);
      expect(result.searchTime).toBe(150);

      expect(mockFetch).toHaveBeenCalledWith(
        `${baseUrl}/api/search`,
        expect.objectContaining({
          method: 'POST',
          body: JSON.stringify(searchQuery)
        })
      );
    });

    it('should handle exact search queries', async () => {
      const exactQuery: SearchQuery = {
        content: '"exact phrase match"',
        searchType: 'exact'
      };

      const mockChunk = createMockChunk('exact-match', 'This contains the exact phrase match in the text');
      const mockResult = createMockSearchResult([
        createMockSearchResultItem(mockChunk, 1.0)
      ]);

      mockFetch.mockResolvedValueOnce(createMockResponse(mockResult));

      const result = await client.searchChunks(exactQuery);

      expect(result.items[0].score).toBe(1.0);
      expect(result.items[0].chunk.contents).toContain('exact phrase match');
    });

    it('should handle fuzzy search with typos', async () => {
      const fuzzyQuery: SearchQuery = {
        content: 'machien lerning', // Intentional typos
        searchType: 'fuzzy'
      };

      const mockChunk = createMockChunk('fuzzy-match', 'Machine learning algorithms and techniques');
      const mockResult = createMockSearchResult([
        createMockSearchResultItem(mockChunk, 0.75)
      ]);

      mockFetch.mockResolvedValueOnce(createMockResponse(mockResult));

      const result = await client.searchChunks(fuzzyQuery);

      expect(result.items[0].score).toBe(0.75);
      expect(result.items[0].chunk.contents).toContain('Machine learning');
    });

    it('should handle empty search results', async () => {
      const searchQuery: SearchQuery = {
        content: 'nonexistent topic xyz123',
        searchType: 'semantic'
      };

      const emptyResult = createMockSearchResult([]);
      mockFetch.mockResolvedValueOnce(createMockResponse(emptyResult));

      const result = await client.searchChunks(searchQuery);

      expect(result.items).toHaveLength(0);
      expect(result.totalCount).toBe(0);
    });

    it('should handle search errors gracefully', async () => {
      const searchQuery: SearchQuery = {
        content: 'test query',
        searchType: 'semantic'
      };

      mockFetch.mockResolvedValueOnce(createMockResponse(
        { error: 'Search service temporarily unavailable' },
        503
      ));

      await expect(client.searchChunks(searchQuery)).rejects.toMatchObject({
        type: ErrorType.API_ERROR,
        code: 'HTTP_503'
      });
    }, 10000);
  });

  describe('searchSemantic', () => {
    it('should perform semantic search with content only', async () => {
      const searchContent = 'quantum computing and quantum algorithms';
      
      const mockChunk = createMockChunk('quantum-chunk', 'Quantum computing principles and quantum algorithm design');
      const mockResult = createMockSearchResult([
        createMockSearchResultItem(mockChunk, 0.88)
      ]);

      mockFetch.mockResolvedValueOnce(createMockResponse(mockResult));

      const result = await client.searchSemantic(searchContent);

      expect(result).toEqual(mockResult);
      expect(mockFetch).toHaveBeenCalledWith(
        `${baseUrl}/api/search/semantic`,
        expect.objectContaining({
          method: 'POST',
          body: JSON.stringify({ content: searchContent })
        })
      );
    });

    it('should handle long content searches', async () => {
      const longContent = 'A'.repeat(5000); // Very long search content
      
      const mockResult = createMockSearchResult([]);
      mockFetch.mockResolvedValueOnce(createMockResponse(mockResult));

      const result = await client.searchSemantic(longContent);

      expect(result.items).toHaveLength(0);
      
      const callArgs = mockFetch.mock.calls[0][1];
      const sentData = JSON.parse(callArgs?.body as string);
      expect(sentData.content).toHaveLength(5000);
    });

    it('should handle special characters in semantic search', async () => {
      const specialContent = 'Search with émojis 🔍, Chinese 中文, and symbols @#$%';
      
      const mockResult = createMockSearchResult([]);
      mockFetch.mockResolvedValueOnce(createMockResponse(mockResult));

      await client.searchSemantic(specialContent);

      const callArgs = mockFetch.mock.calls[0][1];
      const sentData = JSON.parse(callArgs?.body as string);
      expect(sentData.content).toBe(specialContent);
    });
  });

  describe('searchByTags', () => {
    it('should search by single tag', async () => {
      const tags = ['javascript'];
      
      const mockChunk = createMockChunk('js-chunk', 'JavaScript programming concepts');
      mockChunk.tags = ['javascript', 'programming'];
      
      const mockResult = createMockSearchResult([
        createMockSearchResultItem(mockChunk, 1.0)
      ]);

      mockFetch.mockResolvedValueOnce(createMockResponse(mockResult));

      const result = await client.searchByTags(tags);

      expect(result.items[0].chunk.tags).toContain('javascript');
      expect(mockFetch).toHaveBeenCalledWith(
        `${baseUrl}/api/search/tags`,
        expect.objectContaining({
          method: 'POST',
          body: JSON.stringify({ tags })
        })
      );
    });

    it('should search by multiple tags', async () => {
      const tags = ['react', 'typescript', 'frontend'];
      
      const mockChunks = [
        createMockChunk('react-ts-1', 'React with TypeScript setup'),
        createMockChunk('react-ts-2', 'Frontend development with React and TypeScript')
      ];
      
      mockChunks[0].tags = ['react', 'typescript'];
      mockChunks[1].tags = ['react', 'typescript', 'frontend'];

      const mockResult = createMockSearchResult([
        createMockSearchResultItem(mockChunks[0], 0.9),
        createMockSearchResultItem(mockChunks[1], 1.0)
      ]);

      mockFetch.mockResolvedValueOnce(createMockResponse(mockResult));

      const result = await client.searchByTags(tags);

      expect(result.items).toHaveLength(2);
      expect(result.items[1].score).toBe(1.0); // Perfect match with all tags
    });

    it('should handle empty tag arrays', async () => {
      const emptyTags: string[] = [];
      
      const mockResult = createMockSearchResult([]);
      mockFetch.mockResolvedValueOnce(createMockResponse(mockResult));

      const result = await client.searchByTags(emptyTags);

      expect(result.items).toHaveLength(0);
    });

    it('should handle non-existent tags', async () => {
      const nonExistentTags = ['nonexistent-tag-xyz'];
      
      const mockResult = createMockSearchResult([]);
      mockFetch.mockResolvedValueOnce(createMockResponse(mockResult));

      const result = await client.searchByTags(nonExistentTags);

      expect(result.items).toHaveLength(0);
      expect(result.totalCount).toBe(0);
    });
  });

  describe('chatWithAI', () => {
    it('should send message and receive AI response', async () => {
      const message = 'Explain quantum computing in simple terms';
      const context = ['quantum-physics', 'computing-basics'];

      const mockResponse: AIResponse = {
        message: 'Quantum computing uses quantum mechanical phenomena like superposition and entanglement to process information in ways that classical computers cannot.',
        suggestions: [
          {
            type: 'expansion',
            content: 'Would you like to learn about quantum algorithms?',
            confidence: 0.8
          }
        ],
        actions: [
          {
            type: 'create',
            target: 'quantum-computing-note',
            data: { title: 'Quantum Computing Basics' }
          }
        ],
        metadata: {
          model: 'gpt-4',
          tokens: 150,
          processingTime: 1200,
          confidence: 0.92
        }
      };

      mockFetch.mockResolvedValueOnce(createMockResponse(mockResponse));

      const result = await client.chatWithAI(message, context);

      expect(result).toEqual(mockResponse);
      expect(result.message).toContain('Quantum computing');
      expect(result.suggestions).toHaveLength(1);
      expect(result.actions).toHaveLength(1);
      expect(result.metadata.model).toBe('gpt-4');

      expect(mockFetch).toHaveBeenCalledWith(
        `${baseUrl}/api/ai/chat`,
        expect.objectContaining({
          method: 'POST',
          body: JSON.stringify({ message, context })
        })
      );
    });

    it('should handle chat without context', async () => {
      const message = 'Hello, AI!';

      const mockResponse: AIResponse = {
        message: 'Hello! How can I help you today?',
        metadata: {
          model: 'gpt-3.5-turbo',
          tokens: 25,
          processingTime: 300,
          confidence: 0.95
        }
      };

      mockFetch.mockResolvedValueOnce(createMockResponse(mockResponse));

      const result = await client.chatWithAI(message);

      expect(result.message).toBe('Hello! How can I help you today?');
      expect(mockFetch).toHaveBeenCalledWith(
        `${baseUrl}/api/ai/chat`,
        expect.objectContaining({
          body: JSON.stringify({ message, context: undefined })
        })
      );
    });

    it('should handle long conversations', async () => {
      const longMessage = 'A'.repeat(2000); // Very long message
      const longContext = Array.from({ length: 50 }, (_, i) => `context-item-${i}`);

      const mockResponse: AIResponse = {
        message: 'I understand your detailed question. Here is my comprehensive response...',
        metadata: {
          model: 'gpt-4',
          tokens: 500,
          processingTime: 3000,
          confidence: 0.88
        }
      };

      mockFetch.mockResolvedValueOnce(createMockResponse(mockResponse));

      const result = await client.chatWithAI(longMessage, longContext);

      expect(result.metadata.tokens).toBe(500);
      expect(result.metadata.processingTime).toBe(3000);
    });

    it('should handle AI service errors', async () => {
      const message = 'Test message';

      mockFetch.mockResolvedValueOnce(createMockResponse(
        { error: 'AI service overloaded' },
        429
      ));

      await expect(client.chatWithAI(message)).rejects.toMatchObject({
        type: ErrorType.API_ERROR,
        code: 'HTTP_429'
      });
    }, 10000);

    it('should handle malformed AI responses', async () => {
      const message = 'Test message';

      // Mock a response that's missing required fields
      const malformedResponse = {
        // Missing 'message' field
        metadata: {
          model: 'gpt-4',
          tokens: 100,
          processingTime: 1000,
          confidence: 0.9
        }
      };

      mockFetch.mockResolvedValueOnce(createMockResponse(malformedResponse));

      const result = await client.chatWithAI(message);

      // Client should return the response as-is, validation happens at application level
      expect(result).toEqual(malformedResponse);
    });
  });

  describe('processContent', () => {
    it('should process content and return structured results', async () => {
      const content = `# Machine Learning Basics

Machine learning is a subset of artificial intelligence that focuses on algorithms that can learn from data.

## Types of Machine Learning
- Supervised learning
- Unsupervised learning  
- Reinforcement learning`;

      const mockResult: ProcessingResult = {
        chunks: [
          createMockChunk('ml-intro', 'Machine learning is a subset of artificial intelligence that focuses on algorithms that can learn from data.'),
          createMockChunk('ml-types', 'Types of Machine Learning: Supervised learning, Unsupervised learning, Reinforcement learning')
        ],
        suggestions: [
          {
            type: 'expansion',
            content: 'Consider adding examples for each type of machine learning',
            confidence: 0.85,
            position: {
              fileName: 'ml-basics.md',
              lineStart: 5,
              lineEnd: 8,
              charStart: 100,
              charEnd: 200
            }
          }
        ],
        improvements: [
          {
            type: 'structure',
            original: '- Supervised learning\n- Unsupervised learning\n- Reinforcement learning',
            improved: '1. **Supervised learning** - Learning with labeled data\n2. **Unsupervised learning** - Finding patterns in unlabeled data\n3. **Reinforcement learning** - Learning through interaction and rewards',
            explanation: 'Added descriptions and better formatting for clarity',
            position: {
              fileName: 'ml-basics.md',
              lineStart: 6,
              lineEnd: 8,
              charStart: 150,
              charEnd: 250
            }
          }
        ]
      };

      mockFetch.mockResolvedValueOnce(createMockResponse(mockResult));

      const result = await client.processContent(content);

      expect(result).toEqual(mockResult);
      expect(result.chunks).toHaveLength(2);
      expect(result.suggestions).toHaveLength(1);
      expect(result.improvements).toHaveLength(1);

      expect(mockFetch).toHaveBeenCalledWith(
        `${baseUrl}/api/ai/process`,
        expect.objectContaining({
          method: 'POST',
          body: JSON.stringify({ content })
        })
      );
    });

    it('should handle empty content processing', async () => {
      const emptyContent = '';

      const mockResult: ProcessingResult = {
        chunks: [],
        suggestions: [],
        improvements: []
      };

      mockFetch.mockResolvedValueOnce(createMockResponse(mockResult));

      const result = await client.processContent(emptyContent);

      expect(result.chunks).toHaveLength(0);
      expect(result.suggestions).toHaveLength(0);
      expect(result.improvements).toHaveLength(0);
    });

    it('should handle complex markdown content', async () => {
      const complexContent = `# Project Documentation

## Overview
This project implements a **semantic search** system.

### Features
- [x] Vector embeddings
- [ ] Graph relationships
- [ ] Template system

> **Note**: This is still in development

\`\`\`javascript
const search = new SemanticSearch();
search.query("machine learning");
\`\`\`

![Architecture Diagram](./diagram.png)

| Feature | Status | Priority |
|---------|--------|----------|
| Search  | Done   | High     |
| AI Chat | WIP    | Medium   |`;

      const mockResult: ProcessingResult = {
        chunks: [
          createMockChunk('overview', 'This project implements a semantic search system.'),
          createMockChunk('features', 'Features: Vector embeddings, Graph relationships, Template system'),
          createMockChunk('code-example', 'const search = new SemanticSearch(); search.query("machine learning");')
        ],
        suggestions: [
          {
            type: 'improvement',
            content: 'Consider adding more details about the vector embedding implementation',
            confidence: 0.75
          }
        ],
        improvements: []
      };

      mockFetch.mockResolvedValueOnce(createMockResponse(mockResult));

      const result = await client.processContent(complexContent);

      expect(result.chunks).toHaveLength(3);
      expect(result.chunks.some(chunk => chunk.contents.includes('SemanticSearch'))).toBe(true);
    });

    it('should handle content processing errors', async () => {
      const content = 'Test content';

      mockFetch.mockResolvedValueOnce(createMockResponse(
        { error: 'Content processing failed' },
        500
      ));

      await expect(client.processContent(content)).rejects.toMatchObject({
        type: ErrorType.API_ERROR,
        code: 'HTTP_500'
      });
    }, 10000);
  });

  describe('Search Result Caching', () => {
    it('should indicate cache hits in search results', async () => {
      const searchQuery: SearchQuery = {
        content: 'cached search query',
        searchType: 'semantic'
      };

      const cachedResult = createMockSearchResult([]);
      cachedResult.cacheHit = true;
      cachedResult.searchTime = 5; // Much faster due to cache

      mockFetch.mockResolvedValueOnce(createMockResponse(cachedResult));

      const result = await client.searchChunks(searchQuery);

      expect(result.cacheHit).toBe(true);
      expect(result.searchTime).toBe(5);
    });

    it('should handle cache misses properly', async () => {
      const searchQuery: SearchQuery = {
        content: 'new search query',
        searchType: 'semantic'
      };

      const freshResult = createMockSearchResult([]);
      freshResult.cacheHit = false;
      freshResult.searchTime = 250; // Normal search time

      mockFetch.mockResolvedValueOnce(createMockResponse(freshResult));

      const result = await client.searchChunks(searchQuery);

      expect(result.cacheHit).toBe(false);
      expect(result.searchTime).toBe(250);
    });
  });

  describe('Performance and Scalability', () => {
    it('should handle large search result sets', async () => {
      const searchQuery: SearchQuery = {
        content: 'popular topic',
        searchType: 'semantic'
      };

      // Create 1000 mock search results
      const largeResultSet = Array.from({ length: 1000 }, (_, index) => 
        createMockSearchResultItem(
          createMockChunk(`chunk-${index}`, `Content for chunk ${index}`),
          0.8 - (index * 0.0001) // Decreasing scores
        )
      );

      const largeResult = createMockSearchResult(largeResultSet);
      mockFetch.mockResolvedValueOnce(createMockResponse(largeResult));

      const result = await client.searchChunks(searchQuery);

      expect(result.items).toHaveLength(1000);
      expect(result.totalCount).toBe(1000);
      expect(result.items[0].score).toBeGreaterThan(result.items[999].score);
    });

    it('should handle concurrent search requests', async () => {
      const queries = Array.from({ length: 10 }, (_, index) => ({
        content: `concurrent query ${index}`,
        searchType: 'semantic' as const
      }));

      // Mock responses for all queries
      queries.forEach((_, index) => {
        const mockResult = createMockSearchResult([
          createMockSearchResultItem(
            createMockChunk(`concurrent-${index}`, `Result for query ${index}`),
            0.9
          )
        ]);
        mockFetch.mockResolvedValueOnce(createMockResponse(mockResult));
      });

      // Execute all searches concurrently
      const promises = queries.map(query => client.searchChunks(query));
      const results = await Promise.all(promises);

      expect(results).toHaveLength(10);
      expect(mockFetch).toHaveBeenCalledTimes(10);
      
      results.forEach((result, index) => {
        expect(result.items[0].chunk.contents).toContain(`Result for query ${index}`);
      });
    });
  });
});