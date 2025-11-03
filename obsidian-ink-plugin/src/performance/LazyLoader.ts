export interface LazyLoadConfig {
    batchSize: number;
    loadDelay: number; // Delay between batches in ms
    preloadThreshold: number; // How many items ahead to preload
    maxConcurrentLoads: number;
}

export interface LazyLoadItem<T> {
    id: string;
    loader: () => Promise<T>;
    priority: number;
    loaded: boolean;
    loading: boolean;
    data?: T;
    error?: Error;
    lastAccessed: number;
}

export class LazyLoader<T> {
    private items = new Map<string, LazyLoadItem<T>>();
    private loadQueue: string[] = [];
    private activeLoads = new Set<string>();
    private config: LazyLoadConfig;
    private loadingPromises = new Map<string, Promise<T>>();

    constructor(config: Partial<LazyLoadConfig> = {}) {
        this.config = {
            batchSize: 5,
            loadDelay: 100,
            preloadThreshold: 3,
            maxConcurrentLoads: 3,
            ...config
        };
    }

    // Register an item for lazy loading
    register(id: string, loader: () => Promise<T>, priority: number = 0): void {
        this.items.set(id, {
            id,
            loader,
            priority,
            loaded: false,
            loading: false,
            lastAccessed: 0
        });
    }

    // Get an item, loading it if necessary
    async get(id: string): Promise<T | null> {
        const item = this.items.get(id);
        if (!item) return null;

        item.lastAccessed = Date.now();

        // If already loaded, return data
        if (item.loaded && item.data) {
            this.preloadNearbyItems(id);
            return item.data;
        }

        // If currently loading, wait for it
        if (item.loading) {
            const promise = this.loadingPromises.get(id);
            if (promise) {
                try {
                    return await promise;
                } catch (error) {
                    return null;
                }
            }
        }

        // Start loading
        return this.loadItem(id);
    }

    // Load multiple items in batch
    async getBatch(ids: string[]): Promise<Map<string, T | null>> {
        const results = new Map<string, T | null>();
        const toLoad: string[] = [];

        // Separate already loaded from needs loading
        for (const id of ids) {
            const item = this.items.get(id);
            if (!item) {
                results.set(id, null);
                continue;
            }

            item.lastAccessed = Date.now();

            if (item.loaded && item.data) {
                results.set(id, item.data);
            } else {
                toLoad.push(id);
            }
        }

        // Load remaining items in batches
        const batches = this.createBatches(toLoad);
        for (const batch of batches) {
            const batchPromises = batch.map(id => this.loadItem(id));
            const batchResults = await Promise.allSettled(batchPromises);
            
            batch.forEach((id, index) => {
                const result = batchResults[index];
                if (result.status === 'fulfilled') {
                    results.set(id, result.value);
                } else {
                    results.set(id, null);
                }
            });

            // Delay between batches
            if (batches.indexOf(batch) < batches.length - 1) {
                await new Promise(resolve => setTimeout(resolve, this.config.loadDelay));
            }
        }

        return results;
    }

    private async loadItem(id: string): Promise<T | null> {
        const item = this.items.get(id);
        if (!item || item.loaded) return item?.data || null;

        // Check if already loading
        if (item.loading) {
            const promise = this.loadingPromises.get(id);
            if (promise) {
                try {
                    return await promise;
                } catch (error) {
                    return null;
                }
            }
        }

        // Wait if too many concurrent loads
        while (this.activeLoads.size >= this.config.maxConcurrentLoads) {
            await new Promise(resolve => setTimeout(resolve, 50));
        }

        item.loading = true;
        this.activeLoads.add(id);

        const loadPromise = this.executeLoad(item);
        this.loadingPromises.set(id, loadPromise);

        try {
            const data = await loadPromise;
            item.data = data;
            item.loaded = true;
            item.error = undefined;
            return data;
        } catch (error) {
            item.error = error as Error;
            return null;
        } finally {
            item.loading = false;
            this.activeLoads.delete(id);
            this.loadingPromises.delete(id);
        }
    }

    private async executeLoad(item: LazyLoadItem<T>): Promise<T> {
        try {
            return await item.loader();
        } catch (error) {
            throw error;
        }
    }

    private preloadNearbyItems(currentId: string): void {
        const itemIds = Array.from(this.items.keys());
        const currentIndex = itemIds.indexOf(currentId);
        
        if (currentIndex === -1) return;

        // Preload items ahead
        const preloadStart = currentIndex + 1;
        const preloadEnd = Math.min(preloadStart + this.config.preloadThreshold, itemIds.length);
        
        for (let i = preloadStart; i < preloadEnd; i++) {
            const id = itemIds[i];
            const item = this.items.get(id);
            
            if (item && !item.loaded && !item.loading) {
                // Add to queue for background loading
                if (!this.loadQueue.includes(id)) {
                    this.loadQueue.push(id);
                }
            }
        }

        // Process queue in background
        this.processLoadQueue();
    }

    private async processLoadQueue(): Promise<void> {
        if (this.loadQueue.length === 0) return;

        // Sort by priority
        this.loadQueue.sort((a, b) => {
            const itemA = this.items.get(a);
            const itemB = this.items.get(b);
            return (itemB?.priority || 0) - (itemA?.priority || 0);
        });

        const batch = this.loadQueue.splice(0, this.config.batchSize);
        
        // Load batch in background (don't await)
        Promise.all(batch.map(id => this.loadItem(id))).catch(() => {
            // Ignore errors in background loading
        });
    }

    private createBatches(ids: string[]): string[][] {
        const batches: string[][] = [];
        for (let i = 0; i < ids.length; i += this.config.batchSize) {
            batches.push(ids.slice(i, i + this.config.batchSize));
        }
        return batches;
    }

    // Check if item is loaded
    isLoaded(id: string): boolean {
        const item = this.items.get(id);
        return item?.loaded || false;
    }

    // Check if item is loading
    isLoading(id: string): boolean {
        const item = this.items.get(id);
        return item?.loading || false;
    }

    // Get loading status for multiple items
    getLoadingStatus(ids: string[]): Map<string, 'loaded' | 'loading' | 'not_loaded' | 'not_found'> {
        const status = new Map<string, 'loaded' | 'loading' | 'not_loaded' | 'not_found'>();
        
        for (const id of ids) {
            const item = this.items.get(id);
            if (!item) {
                status.set(id, 'not_found');
            } else if (item.loaded) {
                status.set(id, 'loaded');
            } else if (item.loading) {
                status.set(id, 'loading');
            } else {
                status.set(id, 'not_loaded');
            }
        }
        
        return status;
    }

    // Preload specific items
    async preload(ids: string[]): Promise<void> {
        const toLoad = ids.filter(id => {
            const item = this.items.get(id);
            return item && !item.loaded && !item.loading;
        });

        if (toLoad.length === 0) return;

        const batches = this.createBatches(toLoad);
        for (const batch of batches) {
            await Promise.all(batch.map(id => this.loadItem(id)));
            
            if (batches.indexOf(batch) < batches.length - 1) {
                await new Promise(resolve => setTimeout(resolve, this.config.loadDelay));
            }
        }
    }

    // Clear loaded data to free memory
    unload(id: string): void {
        const item = this.items.get(id);
        if (item) {
            item.loaded = false;
            item.data = undefined;
            item.error = undefined;
        }
    }

    // Unload least recently used items
    unloadLRU(count: number): void {
        const loadedItems = Array.from(this.items.values())
            .filter(item => item.loaded)
            .sort((a, b) => a.lastAccessed - b.lastAccessed);

        const toUnload = loadedItems.slice(0, count);
        toUnload.forEach(item => this.unload(item.id));
    }

    // Get statistics
    getStats(): {
        total: number;
        loaded: number;
        loading: number;
        errors: number;
        queueSize: number;
        activeLoads: number;
    } {
        let loaded = 0;
        let loading = 0;
        let errors = 0;

        for (const item of this.items.values()) {
            if (item.loaded) loaded++;
            if (item.loading) loading++;
            if (item.error) errors++;
        }

        return {
            total: this.items.size,
            loaded,
            loading,
            errors,
            queueSize: this.loadQueue.length,
            activeLoads: this.activeLoads.size
        };
    }

    // Clear all data
    clear(): void {
        this.items.clear();
        this.loadQueue = [];
        this.activeLoads.clear();
        this.loadingPromises.clear();
    }
}