import { ParsedContent } from '../types';
import { CacheEntry, CacheStats, CacheConfig } from './SearchCache';

export class ContentCache {
    private cache = new Map<string, CacheEntry<ParsedContent>>();
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
            maxSize: 30 * 1024 * 1024, // 30MB default for content
            maxEntries: 500,
            defaultTTL: 10 * 60 * 1000, // 10 minutes (content changes less frequently)
            cleanupInterval: 2 * 60 * 1000, // 2 minutes
            enableStats: true,
            ...config
        };

        this.startCleanupTimer();
    }

    private calculateSize(data: ParsedContent): number {
        // More accurate size calculation for parsed content
        let size = 0;
        
        // Size of chunks
        size += JSON.stringify(data.chunks).length * 2;
        
        // Size of hierarchy
        size += JSON.stringify(data.hierarchy).length * 2;
        
        // Size of metadata
        size += JSON.stringify(data.metadata).length * 2;
        
        // Size of positions
        size += JSON.stringify(data.positions).length * 2;
        
        return size;
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

    get(filePath: string): ParsedContent | null {
        const entry = this.cache.get(filePath);

        if (!entry) {
            if (this.config.enableStats) {
                this.stats.missCount++;
            }
            return null;
        }

        const now = Date.now();
        if (entry.expiresAt < now) {
            this.deleteEntry(filePath);
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

    set(filePath: string, content: ParsedContent, ttl?: number): void {
        const now = Date.now();
        const size = this.calculateSize(content);
        const expiresAt = now + (ttl || this.config.defaultTTL);

        // Remove existing entry if it exists
        if (this.cache.has(filePath)) {
            this.deleteEntry(filePath);
        }

        const entry: CacheEntry<ParsedContent> = {
            data: content,
            timestamp: now,
            expiresAt,
            accessCount: 0,
            lastAccessed: now,
            size
        };

        this.cache.set(filePath, entry);
        this.stats.totalEntries++;
        this.stats.totalSize += size;

        // Evict if needed
        this.evictIfNeeded();
    }

    delete(filePath: string): boolean {
        const existed = this.cache.has(filePath);
        this.deleteEntry(filePath);
        return existed;
    }

    has(filePath: string): boolean {
        const entry = this.cache.get(filePath);
        if (!entry) return false;
        
        const now = Date.now();
        if (entry.expiresAt < now) {
            this.deleteEntry(filePath);
            return false;
        }
        
        return true;
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

    // Content-specific methods
    invalidateByPattern(pattern: RegExp): number {
        let invalidated = 0;
        const keysToDelete: string[] = [];

        for (const key of this.cache.keys()) {
            if (pattern.test(key)) {
                keysToDelete.push(key);
            }
        }

        for (const key of keysToDelete) {
            this.deleteEntry(key);
            invalidated++;
        }

        return invalidated;
    }

    getContentPaths(): string[] {
        return Array.from(this.cache.keys());
    }

    destroy(): void {
        if (this.cleanupTimer) {
            clearInterval(this.cleanupTimer);
        }
        this.clear();
    }
}