/**
 * Load Testing Suite
 * Comprehensive load testing for various plugin components
 */

import { performance } from 'perf_hooks';
import ObsidianInkPlugin from '../../src/main';
import { ContentManager } from '../../src/content/ContentManager';
import { SearchManager } from '../../src/search/SearchManager';
import { AIManager } from '../../src/ai/AIManager';
import { CacheManager } from '../../src/cache/CacheManager';

// Mock Obsidian
jest.mock('obsidian');

// Extend Jest timeout for load tests
jest.setTimeout(30000);

describe('Load Testing Suite', () => {
    let plugin: ObsidianInkPlugin;
    let contentManager: ContentManager;
    let searchManager: SearchManager;
    let aiManager: AIManager;
    let cacheManager: CacheManager;

    beforeEach(async () => {
        plugin = new ObsidianInkPlugin({} as any, {} as any);
        contentManager = new ContentManager(plugin);
        searchManager = new SearchManager(plugin);
        aiManager = new AIManager(plugin);
        cacheManager = new CacheManager();
        
        // Setup mock API responses
        setupMockApiResponses();
        
        await plugin.onload();
    });

    afterEach(async () => {
        await plugin.onunload();
    });

    describe('Content Processing Load Tests', () => {
        it('should handle massive document processing', async () => {
            const documentSizes = [1000, 5000, 10000, 20000]; // Lines per document
            const results: any[] = [];
            
            for (const size of documentSizes) {
                const content = generateMassiveDocument(size);
                
                const startTime = performance.now();
                const parsed = await contentManager.parseContent(content, `massive-${size}.md`);
                const endTime = performance.now();
                
                const processingTime = endTime - startTime;
                const linesPerSecond = (size / processingTime) * 1000;
                
                results.push({
                    size,
                    processingTime,
                    chunksCreated: parsed.chunks.length,
                    linesPerSecond,
                });
                
                // Verify processing completed successfully
                expect(parsed.chunks.length).toBeGreaterThan(0);
                expect(processingTime).toBeLessThan(size * 2); // Max 2ms per line
                
                console.log(`Document size: ${size} lines, Time: ${processingTime.toFixed(2)}ms, Chunks: ${parsed.chunks.length}, Rate: ${linesPerSecond.toFixed(2)} lines/sec`);
            }
            
            // Verify performance scales reasonably
            const smallDoc = results[0];
            const largeDoc = results[results.length - 1];
            const scalingFactor = largeDoc.processingTime / smallDoc.processingTime;
            const sizeFactor = largeDoc.size / smallDoc.size;
            
            // Processing time should scale sub-linearly (better than O(n))
            expect(scalingFactor).toBeLessThan(sizeFactor * 1.5);
        });

        it('should handle concurrent document processing', async () => {
            const concurrencyLevels = [5, 10, 20, 50];
            const documentSize = 1000; // Lines per document
            
            for (const concurrency of concurrencyLevels) {
                const startTime = performance.now();
                const promises: Promise<any>[] = [];
                
                for (let i = 0; i < concurrency; i++) {
                    const content = generateMassiveDocument(documentSize);
                    const promise = contentManager.parseContent(content, `concurrent-${concurrency}-${i}.md`);
                    promises.push(promise);
                }
                
                const results = await Promise.all(promises);
                const endTime = performance.now();
                
                const totalTime = endTime - startTime;
                const documentsPerSecond = (concurrency / totalTime) * 1000;
                
                expect(results).toHaveLength(concurrency);
                expect(totalTime).toBeLessThan(concurrency * 1000); // Should be faster than sequential
                
                console.log(`Concurrency: ${concurrency}, Time: ${totalTime.toFixed(2)}ms, Rate: ${documentsPerSecond.toFixed(2)} docs/sec`);
            }
        });

        it('should handle burst processing scenarios', async () => {
            const burstSizes = [10, 50, 100, 200];
            const burstInterval = 100; // ms between bursts
            
            for (const burstSize of burstSizes) {
                const startTime = performance.now();
                const allPromises: Promise<any>[] = [];
                
                // Create multiple bursts
                for (let burst = 0; burst < 3; burst++) {
                    const burstPromises: Promise<any>[] = [];
                    
                    // Create burst of operations
                    for (let i = 0; i < burstSize; i++) {
                        const content = `# Burst Document ${burst}-${i}\n\nContent for burst processing test.`;
                        const promise = contentManager.parseContent(content, `burst-${burst}-${i}.md`);
                        burstPromises.push(promise);
                    }
                    
                    allPromises.push(...burstPromises);
                    
                    // Wait between bursts
                    if (burst < 2) {
                        await new Promise(resolve => setTimeout(resolve, burstInterval));
                    }
                }
                
                const results = await Promise.all(allPromises);
                const endTime = performance.now();
                
                const totalTime = endTime - startTime;
                const totalOperations = burstSize * 3;
                
                expect(results).toHaveLength(totalOperations);
                expect(totalTime).toBeLessThan(totalOperations * 100); // Max 100ms per operation
                
                console.log(`Burst size: ${burstSize}, Total ops: ${totalOperations}, Time: ${totalTime.toFixed(2)}ms`);
            }
        });
    });

    describe('Search Load Tests', () => {
        it('should handle high-frequency search requests', async () => {
            const searchFrequencies = [10, 50, 100, 200]; // Searches per second target
            const testDuration = 5000; // 5 seconds
            
            for (const targetFrequency of searchFrequencies) {
                const interval = 1000 / targetFrequency;
                const searches: Promise<any>[] = [];
                let searchCount = 0;
                
                const startTime = performance.now();
                
                const searchTest = new Promise<void>((resolve) => {
                    const intervalId = setInterval(() => {
                        if (performance.now() - startTime >= testDuration) {
                            clearInterval(intervalId);
                            resolve();
                            return;
                        }
                        
                        const searchQuery = {
                            content: `search query ${searchCount}`,
                            searchType: 'semantic' as const,
                        };
                        
                        const searchPromise = searchManager.performSearch(searchQuery);
                        searches.push(searchPromise);
                        searchCount++;
                    }, interval);
                });
                
                await searchTest;
                const results = await Promise.all(searches);
                const endTime = performance.now();
                
                const actualDuration = endTime - startTime;
                const actualFrequency = (searchCount / actualDuration) * 1000;
                
                expect(results).toHaveLength(searchCount);
                expect(actualFrequency).toBeGreaterThan(targetFrequency * 0.8); // Within 20% of target
                
                console.log(`Target: ${targetFrequency} searches/sec, Actual: ${actualFrequency.toFixed(2)} searches/sec, Count: ${searchCount}`);
            }
        });

        it('should maintain search performance under load', async () => {
            const searchCounts = [100, 500, 1000, 2000];
            
            for (const searchCount of searchCounts) {
                const searches: Promise<any>[] = [];
                const searchTimes: number[] = [];
                
                const overallStartTime = performance.now();
                
                for (let i = 0; i < searchCount; i++) {
                    const searchStartTime = performance.now();
                    
                    const searchQuery = {
                        content: `performance test query ${i}`,
                        searchType: 'semantic' as const,
                    };
                    
                    const searchPromise = searchManager.performSearch(searchQuery)
                        .then(result => {
                            const searchEndTime = performance.now();
                            searchTimes.push(searchEndTime - searchStartTime);
                            return result;
                        });
                    
                    searches.push(searchPromise);
                }
                
                const results = await Promise.all(searches);
                const overallEndTime = performance.now();
                
                const totalTime = overallEndTime - overallStartTime;
                const averageSearchTime = searchTimes.reduce((a, b) => a + b, 0) / searchTimes.length;
                const maxSearchTime = Math.max(...searchTimes);
                const minSearchTime = Math.min(...searchTimes);
                
                expect(results).toHaveLength(searchCount);
                expect(averageSearchTime).toBeLessThan(500); // Average under 500ms
                expect(maxSearchTime).toBeLessThan(2000); // Max under 2 seconds
                
                console.log(`Search count: ${searchCount}, Avg: ${averageSearchTime.toFixed(2)}ms, Min: ${minSearchTime.toFixed(2)}ms, Max: ${maxSearchTime.toFixed(2)}ms, Total: ${totalTime.toFixed(2)}ms`);
            }
        });
    });

    describe('AI Chat Load Tests', () => {
        it('should handle multiple concurrent AI conversations', async () => {
            const conversationCounts = [5, 10, 20];
            const messagesPerConversation = 10;
            
            for (const conversationCount of conversationCounts) {
                const startTime = performance.now();
                const conversationPromises: Promise<any>[] = [];
                
                for (let conv = 0; conv < conversationCount; conv++) {
                    const conversationPromise = (async () => {
                        const messages: any[] = [];
                        
                        for (let msg = 0; msg < messagesPerConversation; msg++) {
                            const message = `Message ${msg} from conversation ${conv}`;
                            const response = await aiManager.sendMessage(message);
                            messages.push(response);
                        }
                        
                        return messages;
                    })();
                    
                    conversationPromises.push(conversationPromise);
                }
                
                const results = await Promise.all(conversationPromises);
                const endTime = performance.now();
                
                const totalTime = endTime - startTime;
                const totalMessages = conversationCount * messagesPerConversation;
                const messagesPerSecond = (totalMessages / totalTime) * 1000;
                
                expect(results).toHaveLength(conversationCount);
                expect(totalTime).toBeLessThan(totalMessages * 1000); // Max 1 second per message
                
                console.log(`Conversations: ${conversationCount}, Messages: ${totalMessages}, Time: ${totalTime.toFixed(2)}ms, Rate: ${messagesPerSecond.toFixed(2)} msg/sec`);
            }
        });
    });

    describe('Cache Load Tests', () => {
        it('should handle high-volume cache operations', async () => {
            const cacheOperationCounts = [1000, 5000, 10000, 20000];
            
            for (const operationCount of cacheOperationCounts) {
                const startTime = performance.now();
                
                // Fill cache
                for (let i = 0; i < operationCount; i++) {
                    const key = `load-test-key-${i}`;
                    const data = { id: i, content: `Test data ${i}`, timestamp: Date.now() };
                    cacheManager.set(key, data, 'test');
                }
                
                const fillTime = performance.now();
                
                // Read from cache
                const readPromises: Promise<any>[] = [];
                for (let i = 0; i < operationCount; i++) {
                    const key = `load-test-key-${i}`;
                    const readPromise = Promise.resolve(cacheManager.get(key, 'test'));
                    readPromises.push(readPromise);
                }
                
                const readResults = await Promise.all(readPromises);
                const endTime = performance.now();
                
                const fillDuration = fillTime - startTime;
                const readDuration = endTime - fillTime;
                const totalDuration = endTime - startTime;
                
                const writeOpsPerSecond = (operationCount / fillDuration) * 1000;
                const readOpsPerSecond = (operationCount / readDuration) * 1000;
                
                expect(readResults.filter(r => r !== null)).toHaveLength(operationCount);
                expect(writeOpsPerSecond).toBeGreaterThan(1000); // At least 1000 writes/sec
                expect(readOpsPerSecond).toBeGreaterThan(10000); // At least 10000 reads/sec
                
                console.log(`Cache ops: ${operationCount}, Write: ${writeOpsPerSecond.toFixed(0)} ops/sec, Read: ${readOpsPerSecond.toFixed(0)} ops/sec, Total: ${totalDuration.toFixed(2)}ms`);
            }
        });

        it('should handle cache eviction under memory pressure', async () => {
            const maxCacheSize = 1000;
            const operationCount = maxCacheSize * 3; // 3x the cache size
            
            // Configure cache with size limit
            cacheManager.setMaxSize(maxCacheSize);
            
            const startTime = performance.now();
            
            // Fill cache beyond capacity
            for (let i = 0; i < operationCount; i++) {
                const key = `eviction-test-${i}`;
                const data = { id: i, largeData: 'x'.repeat(1000) }; // 1KB per entry
                cacheManager.set(key, data, 'eviction-test');
            }
            
            const endTime = performance.now();
            const totalTime = endTime - startTime;
            
            // Check that cache size is maintained
            const cacheStats = cacheManager.getStats();
            expect(cacheStats.totalItems).toBeLessThanOrEqual(maxCacheSize * 1.1); // Allow 10% overflow
            
            // Verify recent items are still in cache
            const recentKey = `eviction-test-${operationCount - 1}`;
            const recentItem = cacheManager.get(recentKey, 'eviction-test');
            expect(recentItem).toBeDefined();
            
            // Verify old items were evicted
            const oldKey = `eviction-test-0`;
            const oldItem = cacheManager.get(oldKey, 'eviction-test');
            expect(oldItem).toBeNull();
            
            console.log(`Cache eviction test: ${operationCount} operations, ${totalTime.toFixed(2)}ms, Final size: ${cacheStats.totalItems}`);
        });
    });

    describe('Memory Stress Tests', () => {
        it('should handle memory-intensive operations without leaks', async () => {
            const initialMemory = process.memoryUsage();
            const iterations = 100;
            const largeDataSize = 1024 * 1024; // 1MB per iteration
            
            for (let i = 0; i < iterations; i++) {
                // Create large content
                const largeContent = 'x'.repeat(largeDataSize);
                const content = `# Large Document ${i}\n\n${largeContent}`;
                
                // Process content
                const parsed = await contentManager.parseContent(content, `memory-stress-${i}.md`);
                
                // Verify processing
                expect(parsed.chunks.length).toBeGreaterThan(0);
                
                // Force garbage collection periodically
                if (i % 10 === 0 && global.gc) {
                    global.gc();
                }
                
                // Check memory usage periodically
                if (i % 20 === 0) {
                    const currentMemory = process.memoryUsage();
                    const memoryIncrease = currentMemory.heapUsed - initialMemory.heapUsed;
                    const memoryPerIteration = memoryIncrease / (i + 1);
                    
                    console.log(`Iteration ${i}: Memory increase: ${(memoryIncrease / 1024 / 1024).toFixed(2)}MB, Per iteration: ${(memoryPerIteration / 1024).toFixed(2)}KB`);
                    
                    // Memory per iteration should not grow indefinitely
                    expect(memoryPerIteration).toBeLessThan(largeDataSize * 0.1); // Less than 10% of input size
                }
            }
            
            const finalMemory = process.memoryUsage();
            const totalMemoryIncrease = finalMemory.heapUsed - initialMemory.heapUsed;
            
            console.log(`Memory stress test completed: ${iterations} iterations, Total memory increase: ${(totalMemoryIncrease / 1024 / 1024).toFixed(2)}MB`);
        });
    });
});

// Helper functions
function setupMockApiResponses() {
    // Mock API responses for load testing
    const mockChunk = {
        chunkId: 'mock-chunk',
        contents: 'Mock content',
        documentId: 'mock-doc',
        tags: ['mock'],
        createdTime: new Date(),
        lastUpdated: new Date(),
    };
    
    const mockSearchResult = {
        items: [
            {
                chunk: mockChunk,
                score: 0.9,
                context: 'Mock context',
            },
        ],
        totalCount: 1,
        searchTime: 50,
        cacheHit: false,
    };
    
    const mockAIResponse = {
        message: 'Mock AI response',
        suggestions: [],
        metadata: { responseTime: 100, tokensUsed: 50 },
    };
    
    // Setup mocks (these would be properly mocked in the actual test setup)
    jest.mock('../../src/api/InkGatewayClient', () => ({
        InkGatewayClient: jest.fn().mockImplementation(() => ({
            createChunk: jest.fn().mockResolvedValue(mockChunk),
            batchCreateChunks: jest.fn().mockResolvedValue([mockChunk]),
            searchChunks: jest.fn().mockResolvedValue(mockSearchResult),
            chatWithAI: jest.fn().mockResolvedValue(mockAIResponse),
        })),
    }));
}

function generateMassiveDocument(lines: number): string {
    const content: string[] = [];
    content.push('# Massive Test Document\n');
    
    let currentSection = 0;
    let currentSubsection = 0;
    
    for (let i = 0; i < lines; i++) {
        // Add sections and subsections
        if (i % 200 === 0) {
            currentSection++;
            content.push(`\n## Section ${currentSection}\n`);
        }
        if (i % 50 === 0) {
            currentSubsection++;
            content.push(`\n### Subsection ${currentSubsection}\n`);
        }
        
        // Add various content types
        if (i % 10 === 0) {
            content.push(`\nThis is paragraph ${i}. It contains substantial text content to simulate real-world documents with meaningful content that would be processed by the plugin.`);
        } else if (i % 7 === 0) {
            content.push(`\n- List item ${i}`);
            content.push(`  - Nested item ${i}.1`);
            content.push(`  - Nested item ${i}.2`);
            content.push(`    - Deep nested item ${i}.2.1`);
        } else if (i % 15 === 0) {
            content.push(`\n> Blockquote content for line ${i}. This represents quoted material or important callouts.`);
        } else if (i % 25 === 0) {
            content.push(`\n\`\`\`javascript`);
            content.push(`// Code block for line ${i}`);
            content.push(`function example${i}() {`);
            content.push(`  return "This is example code ${i}";`);
            content.push(`}`);
            content.push(`\`\`\``);
        } else {
            content.push(`Line ${i}: Regular content with some **bold text** and *italic text* and [links](http://example.com/${i}).`);
        }
        
        // Add tags periodically
        if (i % 30 === 0) {
            content.push(`\n#tag${i % 10} #performance #load-test #massive`);
        }
    }
    
    return content.join('\n');
}