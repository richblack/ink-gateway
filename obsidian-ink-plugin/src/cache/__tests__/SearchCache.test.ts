import { SearchCache } from '../SearchCache';
import { SearchQuery, SearchResult } from '../../types';

describe('SearchCache', () => {
    let cache: SearchCache;
    
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

    beforeEach(() => {
        cache = new SearchCache({
            maxSize: 1024 * 1024, // 1MB for testing
            maxEntries: 10,
            defaultTTL: 1000, // 1 second for testing
            cleanupInterval: 500,
            enableStats: true
        });
    });

    afterEach(() => {
        cache.destroy();
    });

    describe('basic operations', () => {
        it('should store and retrieve search results', () => {
            cache.set(mockQuery, mockResult);
            const retrieved = cache.get(mockQuery);
            
            expect(retrieved).toEqual(mockResult);
        });

        it('should return null for non-existent queries', () => {
            const result = cache.get(mockQuery);
            expect(result).toBeNull();
        });

        it('should handle query variations correctly', () => {
            const query1: SearchQuery = { content: 'test', searchType: 'semantic' };
            const query2: SearchQuery = { content: 'test', searchType: 'exact' };
            
            cache.set(query1, mockResult);
            
            expect(cache.get(query1)).toEqual(mockResult);
            expect(cache.get(query2)).toBeNull();
        });
    });

    describe('expiration', () => {
        it('should expire entries after TTL', async () => {
            cache.set(mockQuery, mockResult, 100); // 100ms TTL
            
            expect(cache.get(mockQuery)).toEqual(mockResult);
            
            await new Promise(resolve => setTimeout(resolve, 150));
            
            expect(cache.get(mockQuery)).toBeNull();
        });

        it('should use default TTL when not specified', async () => {
            cache.set(mockQuery, mockResult);
            
            expect(cache.get(mockQuery)).toEqual(mockResult);
            
            await new Promise(resolve => setTimeout(resolve, 1100));
            
            expect(cache.get(mockQuery)).toBeNull();
        });
    });

    describe('size limits', () => {
        it('should evict entries when max entries exceeded', () => {
            // Fill cache to max entries
            for (let i = 0; i < 10; i++) {
                const query: SearchQuery = { content: `test ${i}`, searchType: 'semantic' };
                cache.set(query, mockResult);
            }
            
            const stats = cache.getStats();
            expect(stats.totalEntries).toBe(10);
            
            // Add one more to trigger eviction
            const newQuery: SearchQuery = { content: 'test new', searchType: 'semantic' };
            cache.set(newQuery, mockResult);
            
            const newStats = cache.getStats();
            expect(newStats.totalEntries).toBe(10);
            expect(newStats.evictionCount).toBe(1);
        });
    });

    describe('statistics', () => {
        it('should track hit and miss counts', () => {
            // Miss
            cache.get(mockQuery);
            let stats = cache.getStats();
            expect(stats.missCount).toBe(1);
            expect(stats.hitCount).toBe(0);
            
            // Hit
            cache.set(mockQuery, mockResult);
            cache.get(mockQuery);
            stats = cache.getStats();
            expect(stats.hitCount).toBe(1);
            expect(stats.missCount).toBe(1);
        });

        it('should calculate hit rate correctly', () => {
            cache.set(mockQuery, mockResult);
            
            // 2 hits, 1 miss
            cache.get(mockQuery);
            cache.get(mockQuery);
            cache.get({ content: 'non-existent', searchType: 'semantic' });
            
            expect(cache.getHitRate()).toBeCloseTo(2/3);
        });

        it('should track memory usage', () => {
            const stats1 = cache.getStats();
            expect(stats1.totalSize).toBe(0);
            
            cache.set(mockQuery, mockResult);
            
            const stats2 = cache.getStats();
            expect(stats2.totalSize).toBeGreaterThan(0);
        });
    });

    describe('cleanup', () => {
        it('should clear all entries', () => {
            cache.set(mockQuery, mockResult);
            expect(cache.getStats().totalEntries).toBe(1);
            
            cache.clear();
            expect(cache.getStats().totalEntries).toBe(0);
        });
    });

    describe('key generation', () => {
        it('should generate consistent keys for same queries', () => {
            const query1: SearchQuery = {
                content: 'test',
                tags: ['tag1', 'tag2'],
                searchType: 'semantic'
            };
            
            const query2: SearchQuery = {
                content: 'test',
                tags: ['tag2', 'tag1'], // Different order
                searchType: 'semantic'
            };
            
            cache.set(query1, mockResult);
            expect(cache.get(query2)).toEqual(mockResult); // Should find it despite tag order
        });
    });
});