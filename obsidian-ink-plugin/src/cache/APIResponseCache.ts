import { CacheEntry, CacheStats, CacheConfig } from './SearchCache';

export class APIResponseCache {
    private cache = new Map<string, CacheEntry<any>>();
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
            maxSize: 20 * 1024 * 1024, // 20MB default for API responses
            maxEntries: 200,
            defaultTTL: 2 * 60 * 1000, // 2 minutes (API responses change more frequently)
            cleanupInterval: 60 * 1000, // 1 minute
            enableStats: true,
            ...config
        };

        this.startCleanupTimer();
    }

    private generateKey(endpoint: string, params: any): string {
        const sortedParams = params ? JSON.stringify(params, Object.keys(params).sort()) : '';
        return `${endpoint}:${sortedParams}`;
    }

    private calculateSize(data: any): number {
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

    get<T>(endpoint: string, params: any): T | null {
        const key = this.generateKey(endpoint, params);
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

        return entry.data as T;
    }

    set<T>(endpoint: string, params: any, response: T, ttl?: number): void {
        const key = this.generateKey(endpoint, params);
        const now = Date.now();
        const size = this.calculateSize(response);
        const expiresAt = now + (ttl || this.config.defaultTTL);

        // Remove existing entry if it exists
        if (this.cache.has(key)) {
            this.deleteEntry(key);
        }

        const entry: CacheEntry<T> = {
            data: response,
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

    invalidate(endpoint: string, params?: any): boolean {
        if (params) {
            const key = this.generateKey(endpoint, params);
            const existed = this.cache.has(key);
            this.deleteEntry(key);
            return existed;
        } else {
            // Invalidate all entries for this endpoint
            let invalidated = false;
            const keysToDelete: string[] = [];

            for (const key of this.cache.keys()) {
                if (key.startsWith(`${endpoint}:`)) {
                    keysToDelete.push(key);
                }
            }

            for (const key of keysToDelete) {
                this.deleteEntry(key);
                invalidated = true;
            }

            return invalidated;
        }
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

    // API-specific methods
    invalidateByEndpoint(endpoint: string): number {
        let invalidated = 0;
        const keysToDelete: string[] = [];

        for (const key of this.cache.keys()) {
            if (key.startsWith(`${endpoint}:`)) {
                keysToDelete.push(key);
            }
        }

        for (const key of keysToDelete) {
            this.deleteEntry(key);
            invalidated++;
        }

        return invalidated;
    }

    getEndpoints(): string[] {
        const endpoints = new Set<string>();
        for (const key of this.cache.keys()) {
            const endpoint = key.split(':')[0];
            endpoints.add(endpoint);
        }
        return Array.from(endpoints);
    }

    destroy(): void {
        if (this.cleanupTimer) {
            clearInterval(this.cleanupTimer);
        }
        this.clear();
    }
}