import { PerformanceMonitor } from '../PerformanceMonitor';

describe('PerformanceMonitor', () => {
    let monitor: PerformanceMonitor;

    beforeEach(() => {
        monitor = new PerformanceMonitor();
    });

    afterEach(() => {
        monitor.destroy();
    });

    describe('operation timing', () => {
        it('should track operation times', () => {
            const operationId = monitor.startOperation('test-operation');
            expect(operationId).toContain('test-operation');
            
            const duration = monitor.endOperation(operationId);
            expect(duration).toBeGreaterThanOrEqual(0);
        });

        it('should return 0 for non-existent operation', () => {
            const duration = monitor.endOperation('non-existent');
            expect(duration).toBe(0);
        });

        it('should provide operation statistics', async () => {
            // Perform multiple operations
            for (let i = 0; i < 5; i++) {
                const operationId = monitor.startOperation('test-op');
                await new Promise(resolve => setTimeout(resolve, 10));
                monitor.endOperation(operationId);
            }

            const stats = monitor.getOperationStats('test-op');
            expect(stats).not.toBeNull();
            expect(stats!.count).toBe(5);
            expect(stats!.average).toBeGreaterThan(0);
            expect(stats!.min).toBeGreaterThanOrEqual(0);
            expect(stats!.max).toBeGreaterThanOrEqual(stats!.min);
        });

        it('should return null for non-existent operation stats', () => {
            const stats = monitor.getOperationStats('non-existent');
            expect(stats).toBeNull();
        });
    });

    describe('API call tracking', () => {
        it('should record successful API calls', () => {
            monitor.recordAPICall(100, true);
            monitor.recordAPICall(200, true);
            
            const metrics = monitor.getMetrics();
            expect(metrics.apiCallStats.totalCalls).toBe(2);
            expect(metrics.apiCallStats.averageResponseTime).toBe(150);
            expect(metrics.apiCallStats.errorRate).toBe(0);
        });

        it('should record failed API calls', () => {
            monitor.recordAPICall(100, false);
            monitor.recordAPICall(200, true);
            
            const metrics = monitor.getMetrics();
            expect(metrics.apiCallStats.totalCalls).toBe(2);
            expect(metrics.apiCallStats.errorRate).toBe(0.5);
        });

        it('should track slow API calls', () => {
            monitor.recordAPICall(1500, true); // Slow call
            monitor.recordAPICall(100, true);  // Fast call
            
            const metrics = monitor.getMetrics();
            expect(metrics.apiCallStats.slowCalls).toBe(1);
        });
    });

    describe('cache statistics', () => {
        it('should update cache stats', () => {
            monitor.updateCacheStats(0.8, 1024 * 1024, 5);
            
            const metrics = monitor.getMetrics();
            expect(metrics.cacheStats.hitRate).toBe(0.8);
            expect(metrics.cacheStats.memoryUsage).toBe(1024 * 1024);
            expect(metrics.cacheStats.evictionCount).toBe(5);
        });
    });

    describe('background task tracking', () => {
        it('should track background task counts', () => {
            monitor.incrementBackgroundTask('active');
            monitor.incrementBackgroundTask('queued');
            monitor.incrementBackgroundTask('completed');
            
            const metrics = monitor.getMetrics();
            expect(metrics.backgroundTasks.active).toBe(1);
            expect(metrics.backgroundTasks.queued).toBe(1);
            expect(metrics.backgroundTasks.completed).toBe(1);
        });

        it('should decrement background task counts', () => {
            monitor.incrementBackgroundTask('active');
            monitor.incrementBackgroundTask('active');
            monitor.decrementBackgroundTask('active');
            
            const metrics = monitor.getMetrics();
            expect(metrics.backgroundTasks.active).toBe(1);
        });

        it('should not decrement below zero', () => {
            monitor.decrementBackgroundTask('active');
            
            const metrics = monitor.getMetrics();
            expect(metrics.backgroundTasks.active).toBe(0);
        });
    });

    describe('alerts', () => {
        it('should generate alerts for slow operations', () => {
            const alertCallback = jest.fn();
            monitor.onAlert(alertCallback);
            
            const operationId = monitor.startOperation('slow-op');
            // Simulate slow operation by manually ending with large duration
            (monitor as any).operationStartTimes.set(operationId, performance.now() - 2000);
            monitor.endOperation(operationId);
            
            expect(alertCallback).toHaveBeenCalledWith(
                expect.objectContaining({
                    type: 'slow_operation',
                    severity: 'medium'
                })
            );
        });

        it('should provide recent alerts', () => {
            // Trigger an alert
            const operationId = monitor.startOperation('slow-op');
            (monitor as any).operationStartTimes.set(operationId, performance.now() - 2000);
            monitor.endOperation(operationId);
            
            const recentAlerts = monitor.getRecentAlerts(1);
            expect(recentAlerts.length).toBeGreaterThan(0);
        });

        it('should clear alerts', () => {
            // Trigger an alert
            const operationId = monitor.startOperation('slow-op');
            (monitor as any).operationStartTimes.set(operationId, performance.now() - 2000);
            monitor.endOperation(operationId);
            
            expect(monitor.getAlerts().length).toBeGreaterThan(0);
            
            monitor.clearAlerts();
            expect(monitor.getAlerts().length).toBe(0);
        });
    });

    describe('optimization suggestions', () => {
        it('should provide optimization suggestions', () => {
            // Set up conditions that would trigger suggestions
            monitor.updateCacheStats(0.2, 1024 * 1024, 150); // Low hit rate, high evictions
            monitor.recordAPICall(600, true); // Slow API call
            
            const suggestions = monitor.getOptimizationSuggestions();
            expect(suggestions.length).toBeGreaterThan(0);
            expect(suggestions.some(s => s.includes('cache'))).toBe(true);
        });
    });
});