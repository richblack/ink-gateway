/**
 * SearchManager Tests
 * Comprehensive test suite for search functionality
 */

import { SearchManager } from '../SearchManager';
import { SearchCache } from '../../cache/SearchCache';
import { 
  SearchQuery, 
  SearchResult, 
  SearchResultItem, 
  UnifiedChunk, 
  Position,
  PluginError,
  ErrorType
} from '../../types';
import { IInkGatewayClient } from '../../interfaces';

// Mock API Client
class MockInkGatewayClient implements Partial<IInkGatewayClient> {
  private mockResponses = new Map<string, any>();
  private mockErrors = new Map<string, Error>();
  private requestHistory: any[] = [];

  setMockResponse(endpoint: string, response: any): void {
    this.mockResponses.set(endpoint, response);
  }

  setMockError(endpoint: string, error: Error): void {
    this.mockErrors.set(endpoint, error);
  }

  getRequestHistory(): any[] {
    return this.requestHistory;
  }

  clearHistory(): void {
    this.requestHistory = [];
  }

  async searchChunks(query: SearchQuery): Promise<SearchResult> {
    this.requestHistory.push({ method: 'searchChunks', query });
    
    if (this.mockErrors.has('searchChunks')) {
      throw this.mockErrors.get('searchChunks');
    }
    
    // Add small delay to simulate network request
    await new Promise(resolve => setTimeout(resolve, 1));
    
    return this.mockResponses.get('searchChunks') || this.createMockSearchResult();
  }

  async searchSemantic(content: string): Promise<SearchResult> {
    this.requestHistory.push({ method: 'searchSemantic', content });
    
    if (this.mockErrors.has('searchSemantic')) {
      throw this.mockErrors.get('searchSemantic');
    }
    
    // Add small delay to simulate network request
    await new Promise(resolve => setTimeout(resolve, 1));
    
    return this.mockResponses.get('searchSemantic') || this.createMockSearchResult();
  }

  async searchByTags(tags: string[]): Promise<SearchResult> {
    this.requestHistory.push({ method: 'searchByTags', tags });
    
    if (this.mockErrors.has('searchByTags')) {
      throw this.mockErrors.get('searchByTags');
    }
    
    // Add small delay to simulate network request
    await new Promise(resolve => setTimeout(resolve, 1));
    
    return this.mockResponses.get('searchByTags') || this.createMockSearchResult();
  }

  private createMockSearchResult(): SearchResult {
    const mockChunk: UnifiedChunk = {
      chunkId: 'test-chunk-1',
      contents: 'This is a test chunk content',
      parent: undefined,
      page: 'test-page',
      isPage: false,
      isTag: false,
      isTemplate: false,
      isSlot: false,
      ref: undefined,
      tags: ['test', 'mock'],
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
        charEnd: 28
      },
      filePath: 'test.md',
      obsidianMetadata: {
        properties: {},
        frontmatter: {},
        aliases: [],
        cssClasses: []
      }
    };

    const mockResultItem: SearchResultItem = {
      chunk: mockChunk,
      score: 0.85,
      context: 'This is a test chunk content',
      position: mockChunk.position,
      highlights: []
    };

    return {
      items: [mockResultItem],
      totalCount: 1,
      searchTime: 100,
      cacheHit: false
    };
  }

  // Implement other required methods as no-ops for testing
  async createChunk(): Promise<UnifiedChunk> { throw new Error('Not implemented'); }
  async updateChunk(): Promise<UnifiedChunk> { throw new Error('Not implemented'); }
  async deleteChunk(): Promise<void> { throw new Error('Not implemented'); }
  async getChunk(): Promise<UnifiedChunk> { throw new Error('Not implemented'); }
  async batchCreateChunks(): Promise<UnifiedChunk[]> { throw new Error('Not implemented'); }
  async getHierarchy(): Promise<any[]> { throw new Error('Not implemented'); }
  async updateHierarchy(): Promise<void> { throw new Error('Not implemented'); }
  async chatWithAI(): Promise<any> { throw new Error('Not implemented'); }
  async processContent(): Promise<any> { throw new Error('Not implemented'); }
  async createTemplate(): Promise<any> { throw new Error('Not implemented'); }
  async getTemplateInstances(): Promise<any[]> { throw new Error('Not implemented'); }
  async healthCheck(): Promise<boolean> { return true; }
}

describe('SearchManager', () => {
  let searchManager: SearchManager;
  let mockApiClient: MockInkGatewayClient;
  let mockCache: SearchCache;

  beforeEach(() => {
    mockApiClient = new MockInkGatewayClient();
    mockCache = new SearchCache({ defaultTTL: 1000, maxEntries: 10 });
    searchManager = new SearchManager(mockApiClient as any, mockCache);
  });

  afterEach(() => {
    searchManager.destroy();
  });

  describe('Basic Search Operations', () => {
    test('should perform semantic search successfully', async () => {
      const query: SearchQuery = {
        content: 'test content',
        searchType: 'semantic'
      };

      const result = await searchManager.performSearch(query);

      expect(result).toBeDefined();
      expect(result.items).toHaveLength(1);
      expect(result.items[0].chunk.contents).toBe('This is a test chunk content');
      expect(mockApiClient.getRequestHistory()).toHaveLength(1);
      expect(mockApiClient.getRequestHistory()[0].method).toBe('searchSemantic');
    });

    test('should perform exact search successfully', async () => {
      const query: SearchQuery = {
        content: 'test content',
        searchType: 'exact'
      };

      const result = await searchManager.performSearch(query);

      expect(result).toBeDefined();
      expect(result.items).toHaveLength(1);
      expect(mockApiClient.getRequestHistory()).toHaveLength(1);
      expect(mockApiClient.getRequestHistory()[0].method).toBe('searchChunks');
    });

    test('should perform fuzzy search successfully', async () => {
      const query: SearchQuery = {
        content: 'test content',
        searchType: 'fuzzy'
      };

      const result = await searchManager.performSearch(query);

      expect(result).toBeDefined();
      expect(result.items).toHaveLength(1);
      expect(mockApiClient.getRequestHistory()).toHaveLength(1);
      expect(mockApiClient.getRequestHistory()[0].method).toBe('searchChunks');
    });

    test('should perform tag search successfully', async () => {
      const tags = ['test', 'mock'];
      const result = await searchManager.searchByTags(tags);

      expect(result).toBeDefined();
      expect(result.items).toHaveLength(1);
      expect(mockApiClient.getRequestHistory()).toHaveLength(1);
      expect(mockApiClient.getRequestHistory()[0].method).toBe('searchByTags');
    });
  });

  describe('Query Validation', () => {
    test('should throw error for empty query', async () => {
      const query: SearchQuery = {
        searchType: 'semantic'
      };

      await expect(searchManager.performSearch(query)).rejects.toThrow(PluginError);
      await expect(searchManager.performSearch(query)).rejects.toThrow('EMPTY_QUERY');
    });

    test('should throw error for too short content', async () => {
      const query: SearchQuery = {
        content: 'a',
        searchType: 'semantic'
      };

      await expect(searchManager.performSearch(query)).rejects.toThrow(PluginError);
      await expect(searchManager.performSearch(query)).rejects.toThrow('QUERY_TOO_SHORT');
    });

    test('should accept valid query with tags only', async () => {
      const query: SearchQuery = {
        tags: ['test'],
        searchType: 'exact'
      };

      const result = await searchManager.performSearch(query);
      expect(result).toBeDefined();
    });
  });

  describe('Caching', () => {
    test('should cache search results', async () => {
      const query: SearchQuery = {
        content: 'test content',
        searchType: 'semantic'
      };

      // First search - should hit API
      const result1 = await searchManager.performSearch(query);
      expect(result1.cacheHit).toBe(false);
      expect(mockApiClient.getRequestHistory()).toHaveLength(1);

      // Second search - should hit cache
      const result2 = await searchManager.performSearch(query);
      expect(result2.cacheHit).toBe(true);
      expect(mockApiClient.getRequestHistory()).toHaveLength(1); // No additional API call
    });

    test('should respect cache disabled option', async () => {
      const query: SearchQuery = {
        content: 'test content',
        searchType: 'semantic'
      };

      // First search with cache disabled
      await searchManager.performSearch(query, { enableCache: false });
      expect(mockApiClient.getRequestHistory()).toHaveLength(1);

      // Second search with cache disabled - should hit API again
      await searchManager.performSearch(query, { enableCache: false });
      expect(mockApiClient.getRequestHistory()).toHaveLength(2);
    });
  });

  describe('Result Processing', () => {
    test('should limit results based on maxResults option', async () => {
      // Mock multiple results
      const mockResult: SearchResult = {
        items: Array(10).fill(null).map((_, i) => ({
          chunk: {
            chunkId: `chunk-${i}`,
            contents: `Content ${i}`,
            parent: undefined,
            page: 'test-page',
            isPage: false,
            isTag: false,
            isTemplate: false,
            isSlot: false,
            ref: undefined,
            tags: [],
            metadata: {},
            createdTime: new Date(),
            lastUpdated: new Date(),
            position: {
              fileName: 'test.md',
              lineStart: i,
              lineEnd: i,
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
          },
          score: 0.9 - i * 0.1,
          context: `Content ${i}`,
          position: {
            fileName: 'test.md',
            lineStart: i,
            lineEnd: i,
            charStart: 0,
            charEnd: 10
          },
          highlights: []
        })),
        totalCount: 10,
        searchTime: 100,
        cacheHit: false
      };

      mockApiClient.setMockResponse('searchSemantic', mockResult);

      const query: SearchQuery = {
        content: 'test content',
        searchType: 'semantic'
      };

      const result = await searchManager.performSearch(query, { maxResults: 5 });
      expect(result.items).toHaveLength(5);
    });

    test('should sort results by relevance by default', async () => {
      const mockResult: SearchResult = {
        items: [
          {
            chunk: { chunkId: '1', contents: 'Content 1' } as UnifiedChunk,
            score: 0.5,
            context: 'Content 1',
            position: {} as Position,
            highlights: []
          },
          {
            chunk: { chunkId: '2', contents: 'Content 2' } as UnifiedChunk,
            score: 0.9,
            context: 'Content 2',
            position: {} as Position,
            highlights: []
          },
          {
            chunk: { chunkId: '3', contents: 'Content 3' } as UnifiedChunk,
            score: 0.7,
            context: 'Content 3',
            position: {} as Position,
            highlights: []
          }
        ],
        totalCount: 3,
        searchTime: 100,
        cacheHit: false
      };

      mockApiClient.setMockResponse('searchSemantic', mockResult);

      const query: SearchQuery = {
        content: 'test content',
        searchType: 'semantic'
      };

      const result = await searchManager.performSearch(query);
      expect(result.items[0].score).toBe(0.9);
      expect(result.items[1].score).toBe(0.7);
      expect(result.items[2].score).toBe(0.5);
    });
  });

  describe('Search History', () => {
    test('should track search history', async () => {
      const query: SearchQuery = {
        content: 'test content',
        searchType: 'semantic'
      };

      await searchManager.performSearch(query);

      const history = searchManager.getSearchHistory();
      expect(history).toHaveLength(1);
      expect(history[0].query.content).toBe('test content');
      expect(history[0].resultCount).toBe(1);
    });

    test('should provide search suggestions from history', async () => {
      const queries = [
        { content: 'javascript tutorial', searchType: 'semantic' as const },
        { content: 'javascript functions', searchType: 'semantic' as const },
        { content: 'python basics', searchType: 'semantic' as const }
      ];

      for (const query of queries) {
        await searchManager.performSearch(query);
      }

      const suggestions = await searchManager.getSearchSuggestions('javascript');
      expect(suggestions).toHaveLength(2);
      expect(suggestions).toContain('javascript tutorial');
      expect(suggestions).toContain('javascript functions');
    });

    test('should clear search history', async () => {
      const query: SearchQuery = {
        content: 'test content',
        searchType: 'semantic'
      };

      await searchManager.performSearch(query);
      expect(searchManager.getSearchHistory()).toHaveLength(1);

      searchManager.clearSearchHistory();
      expect(searchManager.getSearchHistory()).toHaveLength(0);
    });
  });

  describe('Statistics', () => {
    test('should track search statistics', async () => {
      const query: SearchQuery = {
        content: 'test content',
        searchType: 'semantic'
      };

      await searchManager.performSearch(query);

      const stats = searchManager.getSearchStats();
      expect(stats.totalSearches).toBe(1);
      expect(stats.averageResponseTime).toBeGreaterThan(0);
      expect(stats.cacheHitRate).toBe(0); // First search is not cached
    });

    test('should calculate cache hit rate correctly', async () => {
      const query: SearchQuery = {
        content: 'test content',
        searchType: 'semantic'
      };

      // First search - cache miss
      await searchManager.performSearch(query);
      let stats = searchManager.getSearchStats();
      expect(stats.cacheHitRate).toBe(0);

      // Second search - cache hit
      await searchManager.performSearch(query);
      stats = searchManager.getSearchStats();
      expect(stats.cacheHitRate).toBe(0.5); // 1 hit out of 2 searches
    });

    test('should reset statistics', async () => {
      const query: SearchQuery = {
        content: 'test content',
        searchType: 'semantic'
      };

      await searchManager.performSearch(query);
      expect(searchManager.getSearchStats().totalSearches).toBe(1);

      searchManager.resetStats();
      expect(searchManager.getSearchStats().totalSearches).toBe(0);
    });
  });

  describe('Error Handling', () => {
    test('should handle API errors gracefully', async () => {
      const apiError = new Error('API Error');
      mockApiClient.setMockError('searchSemantic', apiError);

      const query: SearchQuery = {
        content: 'test content',
        searchType: 'semantic'
      };

      await expect(searchManager.performSearch(query)).rejects.toThrow(PluginError);
      await expect(searchManager.performSearch(query)).rejects.toThrow('SEARCH_FAILED');
    });

    test('should handle network timeouts', async () => {
      const timeoutError = new PluginError(
        ErrorType.NETWORK_ERROR,
        'REQUEST_TIMEOUT',
        { timeout: 5000 },
        true
      );
      mockApiClient.setMockError('searchSemantic', timeoutError);

      const query: SearchQuery = {
        content: 'test content',
        searchType: 'semantic'
      };

      await expect(searchManager.performSearch(query)).rejects.toThrow(PluginError);
      await expect(searchManager.performSearch(query)).rejects.toThrow('REQUEST_TIMEOUT');
    });
  });

  describe('Hybrid Search', () => {
    test('should combine multiple search types', async () => {
      const query: SearchQuery = {
        content: 'test content',
        tags: ['test']
        // No searchType specified - should trigger hybrid search
      };

      // Mock different responses for different search types
      const mockSemanticResult = {
        items: [{
          chunk: { 
            chunkId: 'semantic-1',
            contents: 'Semantic result',
            parent: undefined,
            page: 'test-page',
            isPage: false,
            isTag: false,
            isTemplate: false,
            isSlot: false,
            ref: undefined,
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
          } as UnifiedChunk,
          score: 0.9,
          context: 'Semantic result',
          position: {} as Position,
          highlights: []
        }],
        totalCount: 1,
        searchTime: 100,
        cacheHit: false
      };

      const mockTagResult = {
        items: [{
          chunk: { 
            chunkId: 'tag-1',
            contents: 'Tag result',
            parent: undefined,
            page: 'test-page',
            isPage: false,
            isTag: false,
            isTemplate: false,
            isSlot: false,
            ref: undefined,
            tags: ['test'],
            metadata: {},
            createdTime: new Date(),
            lastUpdated: new Date(),
            position: {
              fileName: 'test.md',
              lineStart: 2,
              lineEnd: 2,
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
          } as UnifiedChunk,
          score: 0.8,
          context: 'Tag result',
          position: {} as Position,
          highlights: []
        }],
        totalCount: 1,
        searchTime: 50,
        cacheHit: false
      };

      mockApiClient.setMockResponse('searchSemantic', mockSemanticResult);
      mockApiClient.setMockResponse('searchByTags', mockTagResult);

      const result = await searchManager.performSearch(query);

      // Should have results from multiple search types
      expect(result.items.length).toBeGreaterThan(0);
      expect(mockApiClient.getRequestHistory().length).toBeGreaterThan(1);
    });
  });

  describe('Search Filters', () => {
    test('should apply search filters', async () => {
      const filters = {
        dateRange: {
          start: new Date('2023-01-01'),
          end: new Date('2023-12-31')
        },
        fileTypes: ['md', 'txt'],
        excludeTags: ['draft'],
        minScore: 0.5
      };

      const result = await searchManager.searchWithFilters('test content', filters);

      expect(result).toBeDefined();
      const history = mockApiClient.getRequestHistory();
      expect(history[0].method).toBe('searchSemantic');
      expect(history[0].content).toBe('test content');
    });
  });
});