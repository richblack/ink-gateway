/**
 * Search Manager - Core search functionality implementation
 * Integrates multiple search types with caching and performance optimization
 */

import { 
  SearchQuery, 
  SearchResult, 
  SearchResultItem, 
  SearchFilters,
  PluginError,
  ErrorType,
  Position
} from '../types';
import { ISearchManager, IInkGatewayClient, ICacheManager } from '../interfaces';
import { SearchCache } from '../cache/SearchCache';

export interface SearchOptions {
  enableCache?: boolean;
  maxResults?: number;
  timeout?: number;
  sortBy?: 'relevance' | 'date' | 'title';
  sortOrder?: 'asc' | 'desc';
}

export interface SearchHistory {
  query: SearchQuery;
  timestamp: Date;
  resultCount: number;
}

export interface SearchStats {
  totalSearches: number;
  averageResponseTime: number;
  cacheHitRate: number;
  popularQueries: SearchHistory[];
}

export class SearchManager implements ISearchManager {
  private apiClient: IInkGatewayClient;
  private cache: SearchCache;
  private searchHistory: SearchHistory[] = [];
  private stats: SearchStats = {
    totalSearches: 0,
    averageResponseTime: 0,
    cacheHitRate: 0,
    popularQueries: []
  };
  
  private readonly maxHistorySize = 100;
  private readonly defaultOptions: SearchOptions = {
    enableCache: true,
    maxResults: 50,
    timeout: 10000,
    sortBy: 'relevance',
    sortOrder: 'desc'
  };

  constructor(apiClient: IInkGatewayClient, cache?: SearchCache) {
    this.apiClient = apiClient;
    this.cache = cache || new SearchCache();
  }

  /**
   * Perform search with multiple types and optimization
   */
  async performSearch(query: SearchQuery, options: SearchOptions = {}): Promise<SearchResult> {
    const startTime = Date.now();
    const searchOptions = { ...this.defaultOptions, ...options };
    
    try {
      // Validate query
      this.validateQuery(query);
      
      // Check cache first if enabled
      if (searchOptions.enableCache) {
        const cachedResult = this.cache.getCachedSearchResult(query);
        if (cachedResult) {
          this.updateStats(startTime, true);
          return this.processSearchResult(cachedResult, searchOptions);
        }
      }

      // Perform search based on type
      let result: SearchResult;
      
      switch (query.searchType) {
        case 'semantic':
          result = await this.performSemanticSearch(query, searchOptions);
          break;
        case 'exact':
          result = await this.performExactSearch(query, searchOptions);
          break;
        case 'fuzzy':
          result = await this.performFuzzySearch(query, searchOptions);
          break;
        default:
          // If no searchType specified, perform hybrid search
          result = await this.performHybridSearch(query, searchOptions);
      }

      // Cache result if enabled
      if (searchOptions.enableCache) {
        this.cache.cacheSearchResult(query, result);
      }

      // Process and optimize result
      const processedResult = this.processSearchResult(result, searchOptions);
      
      // Update statistics and history
      this.updateStats(startTime, false);
      this.addToHistory(query, processedResult.items.length);
      
      return processedResult;

    } catch (error) {
      this.updateStats(startTime, false);
      
      if (error instanceof PluginError) {
        throw error;
      }
      
      throw new PluginError(
        ErrorType.API_ERROR,
        'SEARCH_FAILED',
        { query, error: error instanceof Error ? error.message : error },
        true
      );
    }
  }

  /**
   * Perform semantic search using vector embeddings
   */
  private async performSemanticSearch(query: SearchQuery, options: SearchOptions): Promise<SearchResult> {
    if (!query.content) {
      throw new PluginError(
        ErrorType.VALIDATION_ERROR,
        'MISSING_CONTENT',
        { message: 'Semantic search requires content' },
        false
      );
    }

    return await this.apiClient.searchSemantic(query.content);
  }

  /**
   * Perform exact text search
   */
  private async performExactSearch(query: SearchQuery, options: SearchOptions): Promise<SearchResult> {
    const exactQuery: SearchQuery = {
      ...query,
      searchType: 'exact'
    };
    
    return await this.apiClient.searchChunks(exactQuery);
  }

  /**
   * Perform fuzzy search with tolerance for typos
   */
  private async performFuzzySearch(query: SearchQuery, options: SearchOptions): Promise<SearchResult> {
    const fuzzyQuery: SearchQuery = {
      ...query,
      searchType: 'fuzzy'
    };
    
    return await this.apiClient.searchChunks(fuzzyQuery);
  }

  /**
   * Perform hybrid search combining multiple approaches
   */
  private async performHybridSearch(query: SearchQuery, options: SearchOptions): Promise<SearchResult> {
    const promises: Promise<SearchResult>[] = [];
    
    // Semantic search if content is provided
    if (query.content) {
      promises.push(this.performSemanticSearch({ ...query, searchType: 'semantic' }, options));
    }
    
    // Tag search if tags are provided
    if (query.tags && query.tags.length > 0) {
      promises.push(this.apiClient.searchByTags(query.tags));
    }
    
    // If no specific searches were added, do exact search as fallback
    if (promises.length === 0) {
      promises.push(this.performExactSearch(query, options));
    }
    
    const results = await Promise.allSettled(promises);
    
    // Combine and deduplicate results
    return this.combineSearchResults(results, options);
  }

  /**
   * Search by tags with logic operators
   */
  async searchByTags(tags: string[], logic: 'AND' | 'OR' = 'OR'): Promise<SearchResult> {
    return await this.apiClient.searchByTags(tags);
  }

  /**
   * Search with filters
   */
  /**
   * Search with filters
   */
  async searchWithFilters(content: string, filters: SearchFilters): Promise<SearchResult> {
    const query: SearchQuery = {
      content,
      filters,
      searchType: 'semantic'
    };
    
    return this.performSearch(query);
  }

  /**
   * Get search suggestions based on partial input
   */
  async getSearchSuggestions(partialQuery: string): Promise<string[]> {
    // Get suggestions from search history
    const historySuggestions = this.searchHistory
      .filter(h => h.query.content?.toLowerCase().includes(partialQuery.toLowerCase()))
      .map(h => h.query.content!)
      .filter((value, index, self) => self.indexOf(value) === index)
      .slice(0, 5);

    // TODO: Could also get suggestions from API if available
    return historySuggestions;
  }

  /**
   * Process and optimize search results
   */
  private processSearchResult(result: SearchResult, options: SearchOptions): SearchResult {
    let items = [...result.items];
    
    // Apply result limit
    if (options.maxResults && items.length > options.maxResults) {
      items = items.slice(0, options.maxResults);
    }
    
    // Apply sorting
    items = this.sortResults(items, options.sortBy!, options.sortOrder!);
    
    // Enhance with additional context
    items = this.enhanceResults(items);
    
    return {
      ...result,
      items,
      totalCount: Math.min(result.totalCount, items.length)
    };
  }

  /**
   * Sort search results
   */
  private sortResults(items: SearchResultItem[], sortBy: string, sortOrder: string): SearchResultItem[] {
    return items.sort((a, b) => {
      let comparison = 0;
      
      switch (sortBy) {
        case 'relevance':
          comparison = b.score - a.score;
          break;
        case 'date':
          comparison = new Date(b.chunk.lastUpdated).getTime() - new Date(a.chunk.lastUpdated).getTime();
          break;
        case 'title':
          comparison = a.chunk.contents.localeCompare(b.chunk.contents);
          break;
      }
      
      return sortOrder === 'asc' ? -comparison : comparison;
    });
  }

  /**
   * Enhance results with additional context and metadata
   */
  private enhanceResults(items: SearchResultItem[]): SearchResultItem[] {
    return items.map(item => ({
      ...item,
      context: this.generateContext(item),
      highlights: this.generateHighlights(item)
    }));
  }

  /**
   * Generate context snippet for search result
   */
  private generateContext(item: SearchResultItem): string {
    const content = item.chunk.contents;
    const maxLength = 200;
    
    if (content.length <= maxLength) {
      return content;
    }
    
    // Try to center around the best match
    const midPoint = Math.floor(content.length / 2);
    const start = Math.max(0, midPoint - maxLength / 2);
    const end = Math.min(content.length, start + maxLength);
    
    let context = content.substring(start, end);
    
    // Add ellipsis if truncated
    if (start > 0) context = '...' + context;
    if (end < content.length) context = context + '...';
    
    return context;
  }

  /**
   * Generate text highlights for search matches
   */
  private generateHighlights(item: SearchResultItem): any[] {
    // This would typically be provided by the search API
    // For now, return empty array - could be enhanced with client-side highlighting
    return [];
  }

  /**
   * Combine multiple search results
   */
  private combineSearchResults(results: PromiseSettledResult<SearchResult>[], options: SearchOptions): SearchResult {
    const successfulResults = results
      .filter((result): result is PromiseFulfilledResult<SearchResult> => result.status === 'fulfilled')
      .map(result => result.value);
    
    if (successfulResults.length === 0) {
      return {
        items: [],
        totalCount: 0,
        searchTime: 0,
        cacheHit: false
      };
    }
    
    // Combine all items
    const allItems: SearchResultItem[] = [];
    let totalSearchTime = 0;
    
    successfulResults.forEach(result => {
      allItems.push(...result.items);
      totalSearchTime += result.searchTime;
    });
    
    // Remove duplicates based on chunk ID
    const uniqueItems = allItems.filter((item, index, self) => 
      index === self.findIndex(i => i.chunk.chunkId === item.chunk.chunkId)
    );
    
    // Sort by relevance score
    uniqueItems.sort((a, b) => b.score - a.score);
    
    return {
      items: uniqueItems.slice(0, options.maxResults || 50),
      totalCount: uniqueItems.length,
      searchTime: totalSearchTime / successfulResults.length,
      cacheHit: false
    };
  }

  /**
   * Validate search query
   */
  private validateQuery(query: SearchQuery): void {
    if (!query.content && (!query.tags || query.tags.length === 0)) {
      throw new PluginError(
        ErrorType.VALIDATION_ERROR,
        'EMPTY_QUERY',
        { message: 'Search query must contain content or tags' },
        false
      );
    }
    
    if (query.content && query.content.trim().length < 2) {
      throw new PluginError(
        ErrorType.VALIDATION_ERROR,
        'QUERY_TOO_SHORT',
        { message: 'Search content must be at least 2 characters' },
        false
      );
    }
  }

  /**
   * Update search statistics
   */
  private updateStats(startTime: number, cacheHit: boolean): void {
    const duration = Date.now() - startTime;
    
    this.stats.totalSearches++;
    
    // Update average response time
    if (this.stats.totalSearches === 1) {
      this.stats.averageResponseTime = duration;
    } else {
      this.stats.averageResponseTime = 
        (this.stats.averageResponseTime * (this.stats.totalSearches - 1) + duration) / this.stats.totalSearches;
    }
    
    // Update cache hit rate
    if (this.stats.totalSearches === 1) {
      this.stats.cacheHitRate = cacheHit ? 1 : 0;
    } else {
      const previousHits = this.stats.cacheHitRate * (this.stats.totalSearches - 1);
      const currentHits = previousHits + (cacheHit ? 1 : 0);
      this.stats.cacheHitRate = currentHits / this.stats.totalSearches;
    }
  }

  /**
   * Add search to history
   */
  private addToHistory(query: SearchQuery, resultCount: number): void {
    const historyEntry: SearchHistory = {
      query,
      timestamp: new Date(),
      resultCount
    };
    
    this.searchHistory.unshift(historyEntry);
    
    // Limit history size
    if (this.searchHistory.length > this.maxHistorySize) {
      this.searchHistory = this.searchHistory.slice(0, this.maxHistorySize);
    }
    
    // Update popular queries
    this.updatePopularQueries();
  }

  /**
   * Update popular queries based on frequency
   */
  private updatePopularQueries(): void {
    const queryFrequency = new Map<string, { count: number; lastUsed: Date; query: SearchQuery }>();
    
    this.searchHistory.forEach(entry => {
      const key = JSON.stringify(entry.query);
      const existing = queryFrequency.get(key);
      
      if (existing) {
        existing.count++;
        existing.lastUsed = entry.timestamp;
      } else {
        queryFrequency.set(key, {
          count: 1,
          lastUsed: entry.timestamp,
          query: entry.query
        });
      }
    });
    
    // Sort by frequency and recency
    this.stats.popularQueries = Array.from(queryFrequency.values())
      .sort((a, b) => {
        const scoreA = a.count + (Date.now() - a.lastUsed.getTime()) / (1000 * 60 * 60 * 24); // Decay over days
        const scoreB = b.count + (Date.now() - b.lastUsed.getTime()) / (1000 * 60 * 60 * 24);
        return scoreB - scoreA;
      })
      .slice(0, 10)
      .map(item => ({
        query: item.query,
        timestamp: item.lastUsed,
        resultCount: 0 // This would need to be tracked separately
      }));
  }

  /**
   * Get search history
   */
  getSearchHistory(): SearchHistory[] {
    return [...this.searchHistory];
  }

  /**
   * Clear search history
   */
  clearSearchHistory(): void {
    this.searchHistory = [];
    this.stats.popularQueries = [];
  }

  /**
   * Get search statistics
   */
  getSearchStats(): SearchStats {
    return { ...this.stats };
  }

  /**
   * Reset search statistics
   */
  resetStats(): void {
    this.stats = {
      totalSearches: 0,
      averageResponseTime: 0,
      cacheHitRate: 0,
      popularQueries: []
    };
  }

  /**
   * Display results - placeholder for UI integration
   */
  displayResults(results: SearchResult): void {
    // This will be implemented when we create the UI components
    console.log('Search results:', results);
  }

  /**
   * Navigate to result - placeholder for UI integration
   */
  navigateToResult(result: SearchResultItem): void {
    // This will be implemented when we create the UI components
    console.log('Navigate to:', result.position);
  }

  /**
   * Create search view - placeholder for UI integration
   */
  createSearchView(): void {
    // This will be implemented in task 4.2
    console.log('Creating search view...');
  }

  /**
   * Cleanup resources
   */
  destroy(): void {
    this.cache.destroy();
    this.searchHistory = [];
  }
}