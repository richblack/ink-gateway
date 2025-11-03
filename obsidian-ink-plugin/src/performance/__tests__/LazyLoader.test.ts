import { LazyLoader } from '../LazyLoader';

describe('LazyLoader', () => {
    let loader: LazyLoader<string>;
    let mockLoader: jest.Mock;

    beforeEach(() => {
        loader = new LazyLoader<string>({
            batchSize: 3,
            loadDelay: 50,
            preloadThreshold: 2,
            maxConcurrentLoads: 2
        });
        
        mockLoader = jest.fn();
    });

    afterEach(() => {
        loader.clear();
    });

    describe('basic loading', () => {
        it('should register and load items', async () => {
            mockLoader.mockResolvedValue('test-data');
            loader.register('item1', mockLoader, 1);
            
            const result = await loader.get('item1');
            expect(result).toBe('test-data');
            expect(mockLoader).toHaveBeenCalledTimes(1);
        });

        it('should return null for non-existent items', async () => {
            const result = await loader.get('non-existent');
            expect(result).toBeNull();
        });

        it('should cache loaded items', async () => {
            mockLoader.mockResolvedValue('test-data');
            loader.register('item1', mockLoader);
            
            const result1 = await loader.get('item1');
            const result2 = await loader.get('item1');
            
            expect(result1).toBe('test-data');
            expect(result2).toBe('test-data');
            expect(mockLoader).toHaveBeenCalledTimes(1); // Should only load once
        });

        it('should handle loading errors', async () => {
            mockLoader.mockRejectedValue(new Error('Load failed'));
            loader.register('item1', mockLoader);
            
            const result = await loader.get('item1');
            expect(result).toBeNull();
        });
    });

    describe('batch loading', () => {
        it('should load multiple items in batch', async () => {
            const loaders = [
                jest.fn().mockResolvedValue('data1'),
                jest.fn().mockResolvedValue('data2'),
                jest.fn().mockResolvedValue('data3')
            ];
            
            loader.register('item1', loaders[0]);
            loader.register('item2', loaders[1]);
            loader.register('item3', loaders[2]);
            
            const results = await loader.getBatch(['item1', 'item2', 'item3']);
            
            expect(results.get('item1')).toBe('data1');
            expect(results.get('item2')).toBe('data2');
            expect(results.get('item3')).toBe('data3');
        });

        it('should handle mixed loaded and unloaded items', async () => {
            const loader1 = jest.fn().mockResolvedValue('data1');
            const loader2 = jest.fn().mockResolvedValue('data2');
            
            loader.register('item1', loader1);
            loader.register('item2', loader2);
            
            // Pre-load item1
            await loader.get('item1');
            
            const results = await loader.getBatch(['item1', 'item2']);
            
            expect(results.get('item1')).toBe('data1');
            expect(results.get('item2')).toBe('data2');
            expect(loader1).toHaveBeenCalledTimes(1);
            expect(loader2).toHaveBeenCalledTimes(1);
        });
    });

    describe('status checking', () => {
        it('should report loading status correctly', async () => {
            mockLoader.mockImplementation(() => new Promise(resolve => 
                setTimeout(() => resolve('data'), 100)
            ));
            
            loader.register('item1', mockLoader);
            
            expect(loader.isLoaded('item1')).toBe(false);
            expect(loader.isLoading('item1')).toBe(false);
            
            const loadPromise = loader.get('item1');
            expect(loader.isLoading('item1')).toBe(true);
            
            await loadPromise;
            expect(loader.isLoaded('item1')).toBe(true);
            expect(loader.isLoading('item1')).toBe(false);
        });

        it('should provide batch loading status', () => {
            loader.register('item1', jest.fn());
            loader.register('item2', jest.fn());
            
            const status = loader.getLoadingStatus(['item1', 'item2', 'item3']);
            
            expect(status.get('item1')).toBe('not_loaded');
            expect(status.get('item2')).toBe('not_loaded');
            expect(status.get('item3')).toBe('not_found');
        });
    });

    describe('preloading', () => {
        it('should preload specified items', async () => {
            const loaders = [
                jest.fn().mockResolvedValue('data1'),
                jest.fn().mockResolvedValue('data2')
            ];
            
            loader.register('item1', loaders[0]);
            loader.register('item2', loaders[1]);
            
            await loader.preload(['item1', 'item2']);
            
            expect(loader.isLoaded('item1')).toBe(true);
            expect(loader.isLoaded('item2')).toBe(true);
        });
    });

    describe('memory management', () => {
        it('should unload items to free memory', async () => {
            mockLoader.mockResolvedValue('test-data');
            loader.register('item1', mockLoader);
            
            await loader.get('item1');
            expect(loader.isLoaded('item1')).toBe(true);
            
            loader.unload('item1');
            expect(loader.isLoaded('item1')).toBe(false);
        });

        it('should unload LRU items', async () => {
            const loaders = [
                jest.fn().mockResolvedValue('data1'),
                jest.fn().mockResolvedValue('data2'),
                jest.fn().mockResolvedValue('data3')
            ];
            
            loader.register('item1', loaders[0]);
            loader.register('item2', loaders[1]);
            loader.register('item3', loaders[2]);
            
            // Load all items with delays to establish access order
            await loader.get('item1');
            await new Promise(resolve => setTimeout(resolve, 10));
            await loader.get('item2');
            await new Promise(resolve => setTimeout(resolve, 10));
            await loader.get('item3');
            
            // Unload 2 LRU items (should be item1 and item2)
            loader.unloadLRU(2);
            
            expect(loader.isLoaded('item1')).toBe(false);
            expect(loader.isLoaded('item2')).toBe(false);
            expect(loader.isLoaded('item3')).toBe(true);
        });
    });

    describe('statistics', () => {
        it('should provide loading statistics', async () => {
            const loaders = [
                jest.fn().mockResolvedValue('data1'),
                jest.fn().mockRejectedValue(new Error('Failed')),
                jest.fn().mockResolvedValue('data3')
            ];
            
            loader.register('item1', loaders[0]);
            loader.register('item2', loaders[1]);
            loader.register('item3', loaders[2]);
            
            await loader.get('item1');
            await loader.get('item2'); // This will fail
            // item3 is registered but not loaded
            
            const stats = loader.getStats();
            expect(stats.total).toBe(3);
            expect(stats.loaded).toBe(1);
            expect(stats.errors).toBe(1);
        });
    });
});