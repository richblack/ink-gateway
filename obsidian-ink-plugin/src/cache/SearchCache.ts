import { SearchQuery, SearchResult } from '../types';

export interface CacheEntry<T> {
    data: T;
    timestamp: number;
    expiresAt: number;
    accessCount: number;
    lastAccessed: number;
    size: number;
}

export interface CacheStats {
    totalEntries: number;
    totalSize: number;
    hitCount: number;
    missCount: number;
    evictionCount: number;
    memoryUsage: number;
}

export interface CacheConfig {
    maxSize: number; // Maximum cache size in bytes
    maxEntries: number; // Maximum number of entries
    defaultTTL: number; // Default time to live in milliseconds
    cleanupInterval: number; // Cleanup interval in milliseconds
    enableStats: boolean;
}

export class SearchCache {
    private cache = new Map<string, CacheEntry<SearchResult>>();
    private stats: CacheStats = {
        totalEntries: 0,
        totalSize: 0,
        hitCount: 0,
        missCount: 0,
        evictionCount: 0,
        memoryUsage: 0
    };
    private cleanupTimer?: NodeJS.Timeout;
    private config: CacheConfig;

    constructor(config: Partial<CacheConfig> = {}) {
        this.config = {
            maxSize: 50 * 1024 * 1024, // 50MB default
            maxEntries: 1000,
            defaultTTL: 5 * 60 * 1000, // 5 minutes
            cleanupInterval: 60 * 1000, // 1 minute
            enableStats: true,
            ...config
        };

        this.startCleanupTimer();
    }

    private generateKey(query: SearchQuery): string {
        return JSON.stringify({
            content: query.content,
            tags: query.tags?.sort(),
            tagLogic: query.tagLogic,
            searchType: query.searchType,
            filters: query.filters
        });
    }

    private calculateSize(data: SearchResult): number {
        return JSON.stringify(data).length * 2; // Rough estimate in bytes
    }

    private startCleanupTimer(): void {
        this.cleanupTimer = setInterval(() => {
            this.cleanup();
        }, this.config.cleanupInterval);
    }

    private cleanup(): void {
        const now = Date.now();
        const entriesToDelete: string[] = [];

        // Find expired entries
        for (const [key, entry] of this.cache.entries()) {
            if (entry.expiresAt < now) {
                entriesToDelete.push(key);
            }
        }

        // Delete expired entries
        for (const key of entriesToDelete) {
            this.deleteEntry(key);
        }

        // If still over limits, perform LRU eviction
        this.evictIfNeeded();
    }

    private evictIfNeeded(): void {
        while (this.cache.size > this.config.maxEntries || 
               this.stats.totalSize > this.config.maxSize) {
            
            // Find least recently used entry
            let lruKey: string | null = null;
            let lruTime = Date.now();

            for (const [key, entry] of this.cache.entries()) {
                if (entry.lastAccessed < lruTime) {
                    lruTime = entry.lastAccessed;
                    lruKey = key;
                }
            }

            if (lruKey) {
                this.deleteEntry(lruKey);
                this.stats.evictionCount++;
            } else {
                break;
            }
        }
    }

    private deleteEntry(key: string): void {
        const entry = this.cache.get(key);
        if (entry) {
            this.cache.delete(key);
            this.stats.totalEntries--;
            this.stats.totalSize -= entry.size;
        }
    }

    get(query: SearchQuery): SearchResult | null {
        const key = this.generateKey(query);
        const entry = this.cache.get(key);

        if (!entry) {
            if (this.config.enableStats) {
                this.stats.missCount++;
            }
            return null;
        }

        const now = Date.now();
        if (entry.expiresAt < now) {
            this.deleteEntry(key);
            if (this.config.enableStats) {
                this.stats.missCount++;
            }
            return null;
        }

        // Update access statistics
        entry.lastAccessed = now;
        entry.accessCount++;

        if (this.config.enableStats) {
            this.stats.hitCount++;
        }

        return entry.data;
    }

    set(query: SearchQuery, result: SearchResult, ttl?: number): void {
        const key = this.generateKey(query);
        const now = Date.now();
        const size = this.calculateSize(result);
        const expiresAt = now + (ttl || this.config.defaultTTL);

        // Remove existing entry if it exists
        if (this.cache.has(key)) {
            this.deleteEntry(key);
        }

        const entry: CacheEntry<SearchResult> = {
            data: result,
            timestamp: now,
            expiresAt,
            accessCount: 0,
            lastAccessed: now,
            size
        };

        this.cache.set(key, entry);
        this.stats.totalEntries++;
        this.stats.totalSize += size;

        // Evict if needed
        this.evictIfNeeded();
    }

    clear(): void {
        this.cache.clear();
        this.stats = {
            totalEntries: 0,
            totalSize: 0,
            hitCount: 0,
            missCount: 0,
            evictionCount: 0,
            memoryUsage: 0
        };
    }

    getStats(): CacheStats {
        return {
            ...this.stats,
            memoryUsage: this.stats.totalSize
        };
    }

    getHitRate(): number {
        const total = this.stats.hitCount + this.stats.missCount;
        return total > 0 ? this.stats.hitCount / total : 0;
    }

    destroy(): void {
        if (this.cleanupTimer) {
            clearInterval(this.cleanupTimer);
        }
        this.clear();
    }
}