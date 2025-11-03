/**
 * Performance Tests
 * Tests for performance benchmarks, load testing, and optimization validation
 */

import { performance } from 'perf_hooks';
import ObsidianInkPlugin from '../../src/main';
import { ContentManager } from '../../src/content/ContentManager';
import { SearchManager } from '../../src/search/SearchManager';
import { CacheManager } from '../../src/cache/CacheManager';
import { PerformanceMonitor } from '../../src/performance/PerformanceMonitor';

// Mock Obsidian
jest.mock('obsidian');

describe('Performance Tests', () => {
    let plugin: ObsidianInkPlugin;
    let contentManager: ContentManager;
    let searchManager: SearchManager;
    let cacheManager: CacheManager;
    let performanceMonitor: PerformanceMonitor;

    beforeEach(async () => {
        // Setup test environment
        plugin = new ObsidianInkPlugin({} as any, {} as any);
        contentManager = new ContentManager(plugin);
        searchManager = new SearchManager(plugin);
        cacheManager = new CacheManager();
        performanceMonitor = new PerformanceMonitor();
        
        await plugin.onload();
    });

    afterEach(async () => {
        await plugin.onunload();
    });

    describe('Content Processing Performance', () => {
        it('should process large documents within acceptable time limits', async () => {
            // Generate large test document
            const largeContent = generateLargeMarkdownContent(10000); // 10k lines
            
            const startTime = performance.now();
            
            const parsed = await contentManager.parseContent(largeContent, 'large-test.md');
            
            const endTime = performance.now();
            const processingTime = endTime - startTime;
            
            // Should process 10k lines in under 5 seconds
            expect(processingTime).toBeLessThan(5000);
            expect(parsed.chunks.length).toBeGreaterThan(0);
            
            console.log(`Large document processing: ${processingTime.toFixed(2)}ms for ${parsed.chunks.length} chunks`);
        });

        it('should handle batch operations efficiently', async () => {
            const batchSize = 100;
            const chunks = generateTestChunks(batchSize);
            
            const startTime = performance.now();
            
            // Mock API client for batch operations
            plugin.apiClient.batchCreateChunks = jest.fn().mockResolvedValue(chunks);
            
            await plugin.apiClient.batchCreateChunks(chunks);
            
            const endTime = performance.now();
            const batchTime = endTime - startTime;
            
            // Batch operations should be faster than individual operations
            expect(batchTime).toBeLessThan(1000); // Under 1 second for 100 chunks
            
            console.log(`Batch processing: ${batchTime.toFixed(2)}ms for ${batchSize} chunks`);
        });

        it('should maintain performance with concurrent operations', async () => {
            const concurrentOperations = 10;
            const operationPromises: Promise<any>[] = [];
            
            const startTime = performance.now();
            
            // Create multiple concurrent content processing operations
            for (let i = 0; i < concurrentOperations; i++) {
                const content = `# Document ${i}\n\nContent for document ${i}`;
                const promise = contentManager.parseContent(content, `doc-${i}.md`);
                operationPromises.push(promise);
            }
            
            const results = await Promise.all(operationPromises);
            
            const endTime = performance.now();
            const totalTime = endTime - startTime;
            
            expect(results).toHaveLength(concurrentOperations);
            expect(totalTime).toBeLessThan(3000); // Under 3 seconds for 10 concurrent operations
            
            console.log(`Concurrent operations: ${totalTime.toFixed(2)}ms for ${concurrentOperations} operations`);
        });
    });

    describe('Search Performance', () => {
        it('should perform searches within acceptable response times', async () => {
            // Setup mock search results
            const mockResults = {
                items: generateSearchResults(50),
                totalCount: 50,
                searchTime: 0,
                cacheHit: false,
            };
            
            plugin.apiClient.searchChunks = jest.fn().mockResolvedValue(mockResults);
            
            const searchQuery = {
                content: 'test search query',
                searchType: 'semantic' as const,
            };
            
            const startTime = performance.now();
            
            const results = await searchManager.performSearch(searchQuery);
            
            const endTime = performance.now();
            const searchTime = endTime - startTime;
            
            expect(searchTime).toBeLessThan(500); // Under 500ms for search
            expect(results.items).toHaveLength(50);
            
            console.log(`Search performance: ${searchTime.toFixed(2)}ms for ${results.items.length} results`);
        });

        it('should demonstrate cache performance improvement', async () => {
            const searchQuery = {
                content: 'cached search query',
                searchType: 'semantic' as const,
            };
            
            const mockResults = {
                items: generateSearchResults(20),
                totalCount: 20,
                searchTime: 0,
                cacheHit: false,
            };
            
            plugin.apiClient.searchChunks = jest.fn().mockResolvedValue(mockResults);
            
            // First search (no cache)
            const startTime1 = performance.now();
            const firstResult = await searchManager.performSearch(searchQuery);
            const endTime1 = performance.now();
            const firstSearchTime = endTime1 - startTime1;
            
            // Second search (with cache)
            const startTime2 = performance.now();
            const secondResult = await searchManager.performSearch(searchQuery);
            const endTime2 = performance.now();
            const secondSearchTime = endTime2 - startTime2;
            
            // Cached search should be significantly faster
            expect(secondSearchTime).toBeLessThan(firstSearchTime * 0.1); // At least 10x faster
            expect(secondResult.cacheHit).toBe(true);
            
            console.log(`Cache improvement: ${firstSearchTime.toFixed(2)}ms -> ${secondSearchTime.toFixed(2)}ms`);
        });
    });

    describe('Memory Performance', () => {
        it('should manage memory efficiently with large datasets', async () => {
            const initialMemory = process.memoryUsage();
            
            // Process multiple large documents
            const documentCount = 50;
            const documentsProcessed: any[] = [];
            
            for (let i = 0; i < documentCount; i++) {
                const content = generateLargeMarkdownContent(1000); // 1k lines each
                const parsed = await contentManager.parseContent(content, `memory-test-${i}.md`);
                documentsProcessed.push(parsed);
                
                // Trigger garbage collection periodically
                if (i % 10 === 0 && global.gc) {
                    global.gc();
                }
            }
            
            const finalMemory = process.memoryUsage();
            const memoryIncrease = finalMemory.heapUsed - initialMemory.heapUsed;
            const memoryPerDocument = memoryIncrease / documentCount;
            
            // Memory increase should be reasonable (less than 1MB per document)
            expect(memoryPerDocument).toBeLessThan(1024 * 1024);
            
            console.log(`Memory usage: ${(memoryIncrease / 1024 / 1024).toFixed(2)}MB total, ${(memoryPerDocument / 1024).toFixed(2)}KB per document`);
        });

        it('should clean up cache memory appropriately', async () => {
            const initialMemory = process.memoryUsage();
            
            // Fill cache with data
            for (let i = 0; i < 100; i++) {
                const key = `test-key-${i}`;
                const data = generateLargeTestData(1000); // 1KB each
                cacheManager.set(key, data, 'search');
            }
            
            const cacheFilledMemory = process.memoryUsage();
            
            // Clear cache
            cacheManager.clearAll();
            
            // Force garbage collection if available
            if (global.gc) {
                global.gc();
            }
            
            const clearedMemory = process.memoryUsage();
            
            const memoryAfterCache = cacheFilledMemory.heapUsed - initialMemory.heapUsed;
            const memoryAfterClear = clearedMemory.heapUsed - initialMemory.heapUsed;
            
            // Memory should be significantly reduced after cache clear
            expect(memoryAfterClear).toBeLessThan(memoryAfterCache * 0.5);
            
            console.log(`Cache memory management: +${(memoryAfterCache / 1024).toFixed(2)}KB -> +${(memoryAfterClear / 1024).toFixed(2)}KB`);
        });
    });

    describe('Performance Monitoring', () => {
        it('should track performance metrics accurately', async () => {
            const metrics = performanceMonitor.getMetrics();
            
            // Simulate various operations
            performanceMonitor.startTimer('test-operation');
            await new Promise(resolve => setTimeout(resolve, 100)); // 100ms operation
            performanceMonitor.endTimer('test-operation');
            
            performanceMonitor.recordMetric('api-calls', 1);
            performanceMonitor.recordMetric('cache-hits', 1);
            performanceMonitor.recordMetric('memory-usage', process.memoryUsage().heapUsed);
            
            const updatedMetrics = performanceMonitor.getMetrics();
            
            expect(updatedMetrics.timers['test-operation']).toBeDefined();
            expect(updatedMetrics.timers['test-operation'].count).toBe(1);
            expect(updatedMetrics.timers['test-operation'].average).toBeGreaterThan(90);
            expect(updatedMetrics.timers['test-operation'].average).toBeLessThan(150);
            
            expect(updatedMetrics.counters['api-calls']).toBe(1);
            expect(updatedMetrics.counters['cache-hits']).toBe(1);
            expect(updatedMetrics.gauges['memory-usage']).toBeGreaterThan(0);
        });

        it('should detect performance bottlenecks', async () => {
            // Simulate slow operation
            performanceMonitor.startTimer('slow-operation');
            await new Promise(resolve => setTimeout(resolve, 1000)); // 1 second operation
            performanceMonitor.endTimer('slow-operation');
            
            // Simulate fast operation
            performanceMonitor.startTimer('fast-operation');
            await new Promise(resolve => setTimeout(resolve, 10)); // 10ms operation
            performanceMonitor.endTimer('fast-operation');
            
            const bottlenecks = performanceMonitor.detectBottlenecks();
            
            expect(bottlenecks).toContain('slow-operation');
            expect(bottlenecks).not.toContain('fast-operation');
        });
    });

    describe('Load Testing', () => {
        it('should handle high-frequency operations', async () => {
            const operationCount = 1000;
            const operations: Promise<any>[] = [];
            
            const startTime = performance.now();
            
            // Create many small operations
            for (let i = 0; i < operationCount; i++) {
                const operation = contentManager.parseContent(
                    `# Quick Test ${i}\nContent ${i}`,
                    `quick-${i}.md`
                );
                operations.push(operation);
            }
            
            const results = await Promise.all(operations);
            
            const endTime = performance.now();
            const totalTime = endTime - startTime;
            const operationsPerSecond = (operationCount / totalTime) * 1000;
            
            expect(results).toHaveLength(operationCount);
            expect(operationsPerSecond).toBeGreaterThan(100); // At least 100 ops/sec
            
            console.log(`Load test: ${operationCount} operations in ${totalTime.toFixed(2)}ms (${operationsPerSecond.toFixed(2)} ops/sec)`);
        });

        it('should maintain stability under sustained load', async () => {
            const duration = 5000; // 5 seconds
            const interval = 50; // Every 50ms
            const operations: Promise<any>[] = [];
            
            const startTime = performance.now();
            let operationCount = 0;
            
            const loadTest = new Promise<void>((resolve) => {
                const intervalId = setInterval(async () => {
                    if (performance.now() - startTime >= duration) {
                        clearInterval(intervalId);
                        resolve();
                        return;
                    }
                    
                    const operation = contentManager.parseContent(
                        `# Sustained Load ${operationCount}\nContent for sustained load test`,
                        `sustained-${operationCount}.md`
                    );
                    operations.push(operation);
                    operationCount++;
                }, interval);
            });
            
            await loadTest;
            const results = await Promise.all(operations);
            
            const endTime = performance.now();
            const actualDuration = endTime - startTime;
            const averageOpsPerSecond = (operationCount / actualDuration) * 1000;
            
            expect(results).toHaveLength(operationCount);
            expect(operationCount).toBeGreaterThan(50); // Should have processed many operations
            expect(averageOpsPerSecond).toBeGreaterThan(10); // Sustained rate
            
            console.log(`Sustained load: ${operationCount} operations over ${actualDuration.toFixed(2)}ms (${averageOpsPerSecond.toFixed(2)} avg ops/sec)`);
        });
    });
});

// Helper functions for generating test data
function generateLargeMarkdownContent(lines: number): string {
    const content: string[] = [];
    content.push('# Large Test Document\n');
    
    for (let i = 0; i < lines; i++) {
        if (i % 100 === 0) {
            content.push(`\n## Section ${Math.floor(i / 100)}\n`);
        }
        if (i % 20 === 0) {
            content.push(`\n### Subsection ${Math.floor(i / 20)}\n`);
        }
        content.push(`This is line ${i} of the large test document. It contains some sample text to simulate real content.`);
        
        if (i % 10 === 0) {
            content.push(`\n- List item ${i}`);
            content.push(`  - Nested item ${i}.1`);
            content.push(`  - Nested item ${i}.2`);
        }
        
        if (i % 50 === 0) {
            content.push(`\n#tag${i} #performance #test`);
        }
    }
    
    return content.join('\n');
}

function generateTestChunks(count: number): any[] {
    const chunks = [];
    for (let i = 0; i < count; i++) {
        chunks.push({
            chunkId: `chunk-${i}`,
            contents: `Test chunk content ${i}`,
            documentId: `doc-${Math.floor(i / 10)}`,
            tags: [`tag${i % 5}`, 'test'],
            createdTime: new Date(),
            lastUpdated: new Date(),
        });
    }
    return chunks;
}

function generateSearchResults(count: number): any[] {
    const results = [];
    for (let i = 0; i < count; i++) {
        results.push({
            chunk: {
                chunkId: `result-${i}`,
                contents: `Search result content ${i}`,
                position: {
                    fileName: `result-${i}.md`,
                    lineStart: i,
                    lineEnd: i + 1,
                },
            },
            score: Math.random() * 0.5 + 0.5, // 0.5 to 1.0
            context: `Context for result ${i}`,
        });
    }
    return results;
}

function generateLargeTestData(sizeKB: number): string {
    const targetSize = sizeKB * 1024;
    const chunk = 'A'.repeat(1024); // 1KB chunk
    const chunks = Math.ceil(targetSize / 1024);
    return chunk.repeat(chunks).substring(0, targetSize);
}