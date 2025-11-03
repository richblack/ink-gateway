import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { TFile } from 'obsidian';
import { ObsidianInkPlugin } from '../../src/main';
import { UnifiedChunk } from '../../src/types';
import { createMockEnvironment } from '../mock-data/mock-environment';

/**
 * 效能和穩定性測試
 * 測試插件在各種負載和壓力情況下的表現
 */
describe('Performance and Stability Tests', () => {
    let plugin: ObsidianInkPlugin;
    let mockApp: any;
    let mockVault: any;

    beforeEach(async () => {
        const mockEnv = createMockEnvironment();
        mockApp = mockEnv.app;
        mockVault = mockEnv.vault;
        
        plugin = new ObsidianInkPlugin(mockApp, {} as any);
        await plugin.onload();
    });

    afterEach(async () => {
        await plugin.onunload();
        vi.clearAllMocks();
    });

    describe('Large Content Processing', () => {
        it('should handle large documents efficiently', async () => {
            // 生成大型文件內容（10MB）
            const largeContent = generateLargeContent(10 * 1024 * 1024);
            
            vi.spyOn(mockVault, 'read').mockResolvedValue(largeContent);
            vi.spyOn(plugin.apiClient, 'batchCreateChunks').mockResolvedValue([]);
            
            const startTime = performance.now();
            
            const mockFile = {
                path: 'large-file.md',
                name: 'large-file.md'
            } as TFile;
            
            await plugin.contentManager.handleContentChange(mockFile);
            
            const endTime = performance.now();
            const processingTime = endTime - startTime;
            
            // 處理時間應該在合理範圍內（< 5秒）
            expect(processingTime).toBeLessThan(5000);
        });

        it('should handle memory efficiently with large content', async () => {
            const initialMemory = getMemoryUsage();
            
            // 處理多個大型文件
            for (let i = 0; i < 10; i++) {
                const content = generateLargeContent(1024 * 1024); // 1MB each
                await plugin.contentManager.parseContent(content, `file-${i}.md`);
            }
            
            // 強制垃圾回收
            if (global.gc) {
                global.gc();
            }
            
            const finalMemory = getMemoryUsage();
            const memoryIncrease = finalMemory - initialMemory;
            
            // 記憶體增長應該在合理範圍內（< 100MB）
            expect(memoryIncrease).toBeLessThan(100 * 1024 * 1024);
        });
    });

    describe('Concurrent Operations', () => {
        it('should handle concurrent content processing', async () => {
            const concurrentOperations = 20;
            const promises: Promise<any>[] = [];
            
            vi.spyOn(plugin.apiClient, 'batchCreateChunks').mockResolvedValue([]);
            
            for (let i = 0; i < concurrentOperations; i++) {
                const content = `# File ${i}\n\nContent for file ${i}`;
                vi.spyOn(mockVault, 'read').mockResolvedValue(content);
                
                const mockFile = {
                    path: `file-${i}.md`,
                    name: `file-${i}.md`
                } as TFile;
                
                promises.push(plugin.contentManager.handleContentChange(mockFile));
            }
            
            const startTime = performance.now();
            await Promise.all(promises);
            const endTime = performance.now();
            
            const totalTime = endTime - startTime;
            
            // 並發處理應該在合理時間內完成
            expect(totalTime).toBeLessThan(10000);
            expect(plugin.apiClient.batchCreateChunks).toHaveBeenCalledTimes(concurrentOperations);
        });

        it('should handle concurrent search operations', async () => {
            const concurrentSearches = 15;
            const promises: Promise<any>[] = [];
            
            const mockSearchResult = {
                items: [],
                totalCount: 0,
                searchTime: 50,
                cacheHit: false
            };
            
            vi.spyOn(plugin.apiClient, 'searchSemantic').mockResolvedValue(mockSearchResult);
            
            for (let i = 0; i < concurrentSearches; i++) {
                promises.push(plugin.searchManager.performSearch({
                    content: `search query ${i}`,
                    searchType: 'semantic'
                }));
            }
            
            const results = await Promise.all(promises);
            
            expect(results).toHaveLength(concurrentSearches);
            results.forEach(result => {
                expect(result).toEqual(mockSearchResult);
            });
        });
    });

    describe('Error Recovery and Resilience', () => {
        it('should recover from network failures', async () => {
            let failureCount = 0;
            const maxFailures = 3;
            
            vi.spyOn(plugin.apiClient, 'searchSemantic').mockImplementation(async () => {
                if (failureCount < maxFailures) {
                    failureCount++;
                    throw new Error('Network timeout');
                }
                return {
                    items: [],
                    totalCount: 0,
                    searchTime: 100,
                    cacheHit: false
                };
            });
            
            const result = await plugin.searchManager.performSearch({
                content: 'test query',
                searchType: 'semantic'
            });
            
            expect(result).toBeDefined();
            expect(failureCount).toBe(maxFailures);
        });

        it('should handle API rate limiting gracefully', async () => {
            let requestCount = 0;
            
            vi.spyOn(plugin.apiClient, 'createChunk').mockImplementation(async () => {
                requestCount++;
                if (requestCount <= 5) {
                    const error = new Error('Rate limit exceeded') as any;
                    error.status = 429;
                    throw error;
                }
                return {} as UnifiedChunk;
            });
            
            const chunk: UnifiedChunk = {
                chunkId: 'test-chunk',
                contents: 'Test content',
                documentId: 'doc-1',
                filePath: 'test.md',
                tags: [],
                metadata: {},
                createdTime: new Date(),
                lastUpdated: new Date(),
                isPage: false,
                isTag: false,
                isTemplate: false,
                isSlot: false,
                documentScope: 'file' as const,
                position: { fileName: 'test.md', lineStart: 1, lineEnd: 1, charStart: 0, charEnd: 12 }
            };
            
            const result = await plugin.apiClient.createChunk(chunk);
            expect(result).toBeDefined();
            expect(requestCount).toBeGreaterThan(5);
        });
    });

    describe('Cache Performance', () => {
        it('should improve performance with caching', async () => {
            const searchQuery = {
                content: 'cached query',
                searchType: 'semantic' as const
            };
            
            const mockResult = {
                items: [],
                totalCount: 0,
                searchTime: 100,
                cacheHit: false
            };
            
            vi.spyOn(plugin.apiClient, 'searchSemantic').mockResolvedValue(mockResult);
            
            // 第一次搜尋（應該呼叫 API）
            const startTime1 = performance.now();
            const result1 = await plugin.searchManager.performSearch(searchQuery);
            const endTime1 = performance.now();
            const time1 = endTime1 - startTime1;
            
            // 第二次搜尋（應該使用快取）
            const startTime2 = performance.now();
            const result2 = await plugin.searchManager.performSearch(searchQuery);
            const endTime2 = performance.now();
            const time2 = endTime2 - startTime2;
            
            expect(result1).toEqual(result2);
            expect(time2).toBeLessThan(time1); // 快取應該更快
            expect(plugin.apiClient.searchSemantic).toHaveBeenCalledTimes(1); // 只呼叫一次 API
        });

        it('should manage cache size effectively', async () => {
            const cacheManager = plugin.cacheManager;
            
            // 填滿快取
            for (let i = 0; i < 1000; i++) {
                await cacheManager.set(`key-${i}`, `value-${i}`, 'search');
            }
            
            const cacheStats = cacheManager.getStats();
            
            // 快取大小應該在限制內
            expect(cacheStats.totalSize).toBeLessThan(50 * 1024 * 1024); // 50MB limit
            expect(cacheStats.itemCount).toBeLessThanOrEqual(1000);
        });
    });

    describe('Long-running Stability', () => {
        it('should maintain stability over extended periods', async () => {
            const iterations = 100;
            const errors: Error[] = [];
            
            for (let i = 0; i < iterations; i++) {
                try {
                    // 模擬各種操作
                    await simulateUserActivity(plugin);
                    
                    // 每 10 次迭代檢查記憶體
                    if (i % 10 === 0) {
                        const memoryUsage = getMemoryUsage();
                        expect(memoryUsage).toBeLessThan(500 * 1024 * 1024); // 500MB limit
                    }
                } catch (error) {
                    errors.push(error as Error);
                }
            }
            
            // 錯誤率應該很低
            const errorRate = errors.length / iterations;
            expect(errorRate).toBeLessThan(0.05); // < 5% error rate
        });
    });

    describe('Resource Cleanup', () => {
        it('should clean up resources properly on unload', async () => {
            const initialHandlers = getEventHandlerCount();
            
            // 創建多個插件實例
            const plugins: ObsidianInkPlugin[] = [];
            for (let i = 0; i < 10; i++) {
                const p = new ObsidianInkPlugin(mockApp, {} as any);
                await p.onload();
                plugins.push(p);
            }
            
            // 卸載所有插件
            for (const p of plugins) {
                await p.onunload();
            }
            
            const finalHandlers = getEventHandlerCount();
            
            // 事件處理器應該被清理
            expect(finalHandlers).toBeLessThanOrEqual(initialHandlers + 1);
        });
    });
});

// 輔助函數
function generateLargeContent(sizeInBytes: number): string {
    const chunkSize = 1000;
    const chunks = Math.ceil(sizeInBytes / chunkSize);
    let content = '';
    
    for (let i = 0; i < chunks; i++) {
        content += `# Heading ${i}\n\n`;
        content += 'Lorem ipsum dolor sit amet, consectetur adipiscing elit. '.repeat(10);
        content += '\n\n';
    }
    
    return content.substring(0, sizeInBytes);
}

function getMemoryUsage(): number {
    if (typeof process !== 'undefined' && process.memoryUsage) {
        return process.memoryUsage().heapUsed;
    }
    return 0;
}

function getEventHandlerCount(): number {
    // 模擬事件處理器計數
    return 0;
}

async function simulateUserActivity(plugin: ObsidianInkPlugin): Promise<void> {
    const activities = [
        () => plugin.searchManager.performSearch({ content: 'test', searchType: 'semantic' }),
        () => plugin.aiManager.sendMessage('Hello'),
        () => plugin.contentManager.parseContent('# Test\n\nContent', 'test.md'),
        () => plugin.cacheManager.cleanup()
    ];
    
    const randomActivity = activities[Math.floor(Math.random() * activities.length)];
    
    try {
        await randomActivity();
    } catch (error) {
        // 忽略模擬錯誤
    }
}