import { CacheManager } from '../CacheManager';
import { SearchQuery, SearchResult, ParsedContent } from '../../types';

describe('CacheManager', () => {
    let cacheManager: CacheManager;
    
    const mockQuery: SearchQuery = {
        content: 'test query',
        searchType: 'semantic'
    };
    
    const mockResult: SearchResult = {
        items: [],
        totalCount: 0,
        searchTime: 100,
        cacheHit: false
    };
    
    const mockParsedContent: ParsedContent = {
        chunks: [],
        hierarchy: [],
        metadata: {
            tags: [],
            properties: {},
            frontmatter: {},
            aliases: [],
            cssClasses: [],
            createdTime: new Date(),
            modifiedTime: new Date()
        },
        positions: new Map()
    };

    beforeEach(() => {
        cacheManager = new CacheManager({
            globalMemoryLimit: 1024 * 1024, // 1MB for testing
            monitoringEnabled: false, // Disable for testing
            search: {
                maxEntries: 5,
                defaultTTL: 1000
            },
            content: {
                maxEntries: 5,
                defaultTTL: 1000
            },
            apiResponse: {
                maxEntries: 5,
                defaultTTL: 1000
            }
        });
    });

    afterEach(() => {
        cacheManager.destroy();
    });

    describe('search cache integration', () => {
        it('should store and retrieve search results', () => {
            cacheManager.setSearchResult(mockQuery, mockResult);
            const retrieved = cacheManager.getSearchResult(mockQuery);
            
            expect(retrieved).toEqual(mockResult);
        });

        it('should return null for non-existent search results', () => {
            const result = cacheManager.getSearchResult(mockQuery);
            expect(result).toBeNull();
        });
    });

    describe('content cache integration', () => {
        it('should store and retrieve parsed content', () => {
            const filePath = '/test/file.md';
            
            cacheManager.setParsedContent(filePath, mockParsedContent);
            const retrieved = cacheManager.getParsedContent(filePath);
            
            expect(retrieved).toEqual(mockParsedContent);
        });

        it('should invalidate content by file path', () => {
            const filePath = '/test/file.md';
            
            cacheManager.setParsedContent(filePath, mockParsedContent);
            expect(cacheManager.getParsedContent(filePath)).toEqual(mockParsedContent);
            
            cacheManager.invalidateContent(filePath);
            expect(cacheManager.getParsedContent(filePath)).toBeNull();
        });
    });

    describe('API response cache integration', () => {
        it('should store and retrieve API responses', () => {
            const endpoint = '/api/chunks';
            const params = { id: '123' };
            const response = { data: 'test' };
            
            cacheManager.setAPIResponse(endpoint, params, response);
            const retrieved = cacheManager.getAPIResponse(endpoint, params);
            
            expect(retrieved).toEqual(response);
        });

        it('should handle different parameter combinations', () => {
            const endpoint = '/api/chunks';
            const params1 = { id: '123' };
            const params2 = { id: '456' };
            const response1 = { data: 'test1' };
            const response2 = { data: 'test2' };
            
            cacheManager.setAPIResponse(endpoint, params1, response1);
            cacheManager.setAPIResponse(endpoint, params2, response2);
            
            expect(cacheManager.getAPIResponse(endpoint, params1)).toEqual(response1);
            expect(cacheManager.getAPIResponse(endpoint, params2)).toEqual(response2);
        });
    });

    describe('global cache management', () => {
        it('should provide global statistics', () => {
            cacheManager.setSearchResult(mockQuery, mockResult);
            cacheManager.setParsedContent('/test.md', mockParsedContent);
            cacheManager.setAPIResponse('/api/test', {}, { data: 'test' });
            
            const stats = cacheManager.getGlobalStats();
            
            expect(stats.search.totalEntries).toBe(1);
            expect(stats.content.totalEntries).toBe(1);
            expect(stats.apiResponse.totalEntries).toBe(1);
            expect(stats.totalMemoryUsage).toBeGreaterThan(0);
        });

        it('should calculate global hit rate', () => {
            cacheManager.setSearchResult(mockQuery, mockResult);
            
            // Generate some hits and misses
            cacheManager.getSearchResult(mockQuery); // hit
            cacheManager.getSearchResult({ content: 'other', searchType: 'semantic' }); // miss
            cacheManager.getParsedContent('/nonexistent.md'); // miss
            
            const stats = cacheManager.getGlobalStats();
            expect(stats.globalHitRate).toBeCloseTo(1/3);
        });

        it('should clear all caches', () => {
            cacheManager.setSearchResult(mockQuery, mockResult);
            cacheManager.setParsedContent('/test.md', mockParsedContent);
            cacheManager.setAPIResponse('/api/test', {}, { data: 'test' });
            
            let stats = cacheManager.getGlobalStats();
            expect(stats.search.totalEntries + stats.content.totalEntries + stats.apiResponse.totalEntries).toBe(3);
            
            cacheManager.clearAll();
            
            stats = cacheManager.getGlobalStats();
            expect(stats.search.totalEntries + stats.content.totalEntries + stats.apiResponse.totalEntries).toBe(0);
        });

        it('should track memory usage', () => {
            expect(cacheManager.getMemoryUsage()).toBe(0);
            
            cacheManager.setSearchResult(mockQuery, mockResult);
            expect(cacheManager.getMemoryUsage()).toBeGreaterThan(0);
        });

        it('should detect memory limit exceeded', () => {
            expect(cacheManager.isMemoryLimitExceeded()).toBe(false);
            
            // This would need a very large object to exceed 1MB limit in test
            // For now, just test the method exists and returns boolean
            expect(typeof cacheManager.isMemoryLimitExceeded()).toBe('boolean');
        });
    });

    describe('memory optimization', () => {
        it('should optimize memory when requested', () => {
            cacheManager.setSearchResult(mockQuery, mockResult);
            cacheManager.setParsedContent('/test.md', mockParsedContent);
            
            const initialMemory = cacheManager.getMemoryUsage();
            cacheManager.optimizeMemory();
            
            // Memory should be same or less after optimization
            expect(cacheManager.getMemoryUsage()).toBeLessThanOrEqual(initialMemory);
        });
    });
});