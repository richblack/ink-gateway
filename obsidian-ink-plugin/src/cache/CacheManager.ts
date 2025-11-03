import { SearchCache, CacheConfig, CacheStats } from './SearchCache';
import { ContentCache } from './ContentCache';
import { APIResponseCache } from './APIResponseCache';
import { UnifiedChunk, SearchQuery, SearchResult, ParsedContent } from '../types';

export interface CacheManagerConfig {
    search: Partial<CacheConfig>;
    content: Partial<CacheConfig>;
    apiResponse: Partial<CacheConfig>;
    globalMemoryLimit: number; // Total memory limit in bytes
    monitoringEnabled: boolean;
}

export interface GlobalCacheStats {
    search: CacheStats;
    content: CacheStats;
    apiResponse: CacheStats;
    totalMemoryUsage: number;
    globalHitRate: number;
}

export class CacheManager {
    private searchCache: SearchCache;
    private contentCache: ContentCache;
    private apiResponseCache: APIResponseCache;
    private config: CacheManagerConfig;
    private monitoringTimer?: NodeJS.Timeout;

    constructor(config: Partial<CacheManagerConfig> = {}) {
        this.config = {
            search: {},
            content: {},
            apiResponse: {},
            globalMemoryLimit: 100 * 1024 * 1024, // 100MB default
            monitoringEnabled: true,
            ...config
        };

        this.searchCache = new SearchCache(this.config.search);
        this.contentCache = new ContentCache(this.config.content);
        this.apiResponseCache = new APIResponseCache(this.config.apiResponse);

        if (this.config.monitoringEnabled) {
            this.startMonitoring();
        }
    }

    private startMonitoring(): void {
        this.monitoringTimer = setInterval(() => {
            this.monitorMemoryUsage();
        }, 30000); // Check every 30 seconds
    }

    private monitorMemoryUsage(): void {
        const stats = this.getGlobalStats();
        
        if (stats.totalMemoryUsage > this.config.globalMemoryLimit) {
            console.warn(`Cache memory usage (${stats.totalMemoryUsage} bytes) exceeds limit (${this.config.globalMemoryLimit} bytes)`);
            this.performGlobalCleanup();
        }
    }

    private performGlobalCleanup(): void {
        // Clear least important caches first
        const stats = this.getGlobalStats();
        
        // If API response cache is largest, clear it first
        if (stats.apiResponse.memoryUsage > stats.search.memoryUsage && 
            stats.apiResponse.memoryUsage > stats.content.memoryUsage) {
            this.apiResponseCache.clear();
        }
        // Otherwise clear search cache if it's larger than content cache
        else if (stats.search.memoryUsage > stats.content.memoryUsage) {
            this.searchCache.clear();
        }
        // Finally clear content cache if needed
        else {
            this.contentCache.clear();
        }
    }

    // Search Cache Methods
    getSearchResult(query: SearchQuery): SearchResult | null {
        return this.searchCache.get(query);
    }

    setSearchResult(query: SearchQuery, result: SearchResult, ttl?: number): void {
        this.searchCache.set(query, result, ttl);
    }

    // Content Cache Methods
    getParsedContent(filePath: string): ParsedContent | null {
        return this.contentCache.get(filePath);
    }

    setParsedContent(filePath: string, content: ParsedContent, ttl?: number): void {
        this.contentCache.set(filePath, content, ttl);
    }

    invalidateContent(filePath: string): void {
        this.contentCache.delete(filePath);
    }

    // API Response Cache Methods
    getAPIResponse<T>(endpoint: string, params: any): T | null {
        return this.apiResponseCache.get<T>(endpoint, params);
    }

    setAPIResponse<T>(endpoint: string, params: any, response: T, ttl?: number): void {
        this.apiResponseCache.set(endpoint, params, response, ttl);
    }

    // Global Cache Management
    clearAll(): void {
        this.searchCache.clear();
        this.contentCache.clear();
        this.apiResponseCache.clear();
    }

    getGlobalStats(): GlobalCacheStats {
        const searchStats = this.searchCache.getStats();
        const contentStats = this.contentCache.getStats();
        const apiStats = this.apiResponseCache.getStats();

        const totalHits = searchStats.hitCount + contentStats.hitCount + apiStats.hitCount;
        const totalMisses = searchStats.missCount + contentStats.missCount + apiStats.missCount;
        const globalHitRate = (totalHits + totalMisses) > 0 ? totalHits / (totalHits + totalMisses) : 0;

        return {
            search: searchStats,
            content: contentStats,
            apiResponse: apiStats,
            totalMemoryUsage: searchStats.memoryUsage + contentStats.memoryUsage + apiStats.memoryUsage,
            globalHitRate
        };
    }

    getMemoryUsage(): number {
        return this.getGlobalStats().totalMemoryUsage;
    }

    isMemoryLimitExceeded(): boolean {
        return this.getMemoryUsage() > this.config.globalMemoryLimit;
    }

    optimizeMemory(): void {
        if (this.isMemoryLimitExceeded()) {
            this.performGlobalCleanup();
        }
    }

    destroy(): void {
        if (this.monitoringTimer) {
            clearInterval(this.monitoringTimer);
        }
        
        this.searchCache.destroy();
        this.contentCache.destroy();
        this.apiResponseCache.destroy();
    }
}