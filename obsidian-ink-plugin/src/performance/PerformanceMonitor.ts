export interface PerformanceMetrics {
    memoryUsage: {
        used: number;
        total: number;
        percentage: number;
    };
    operationTimes: Map<string, number[]>;
    apiCallStats: {
        totalCalls: number;
        averageResponseTime: number;
        errorRate: number;
        slowCalls: number; // Calls taking > 1 second
    };
    cacheStats: {
        hitRate: number;
        memoryUsage: number;
        evictionCount: number;
    };
    backgroundTasks: {
        active: number;
        queued: number;
        completed: number;
        failed: number;
    };
}

export interface PerformanceAlert {
    type: 'memory' | 'slow_operation' | 'high_error_rate' | 'cache_thrashing';
    severity: 'low' | 'medium' | 'high' | 'critical';
    message: string;
    timestamp: number;
    metrics: any;
}

export class PerformanceMonitor {
    private metrics: PerformanceMetrics;
    private alerts: PerformanceAlert[] = [];
    private operationStartTimes = new Map<string, number>();
    private monitoringInterval?: NodeJS.Timeout;
    private alertCallbacks: ((alert: PerformanceAlert) => void)[] = [];

    constructor() {
        this.metrics = {
            memoryUsage: { used: 0, total: 0, percentage: 0 },
            operationTimes: new Map(),
            apiCallStats: {
                totalCalls: 0,
                averageResponseTime: 0,
                errorRate: 0,
                slowCalls: 0
            },
            cacheStats: {
                hitRate: 0,
                memoryUsage: 0,
                evictionCount: 0
            },
            backgroundTasks: {
                active: 0,
                queued: 0,
                completed: 0,
                failed: 0
            }
        };

        this.startMonitoring();
    }

    private startMonitoring(): void {
        this.monitoringInterval = setInterval(() => {
            this.updateMemoryMetrics();
            this.checkForAlerts();
        }, 10000); // Check every 10 seconds
    }

    private updateMemoryMetrics(): void {
        if (typeof performance !== 'undefined' && (performance as any).memory) {
            const memory = (performance as any).memory;
            this.metrics.memoryUsage = {
                used: memory.usedJSHeapSize,
                total: memory.totalJSHeapSize,
                percentage: (memory.usedJSHeapSize / memory.totalJSHeapSize) * 100
            };
        }
    }

    private checkForAlerts(): void {
        // Memory usage alert
        if (this.metrics.memoryUsage.percentage > 80) {
            this.createAlert('memory', 'high', 
                `High memory usage: ${this.metrics.memoryUsage.percentage.toFixed(1)}%`,
                { memoryUsage: this.metrics.memoryUsage }
            );
        }

        // API error rate alert
        if (this.metrics.apiCallStats.errorRate > 0.1) { // 10% error rate
            this.createAlert('high_error_rate', 'medium',
                `High API error rate: ${(this.metrics.apiCallStats.errorRate * 100).toFixed(1)}%`,
                { apiCallStats: this.metrics.apiCallStats }
            );
        }

        // Cache thrashing alert
        if (this.metrics.cacheStats.hitRate < 0.3 && this.metrics.cacheStats.evictionCount > 100) {
            this.createAlert('cache_thrashing', 'medium',
                `Low cache hit rate with high evictions: ${(this.metrics.cacheStats.hitRate * 100).toFixed(1)}%`,
                { cacheStats: this.metrics.cacheStats }
            );
        }
    }

    private createAlert(type: PerformanceAlert['type'], severity: PerformanceAlert['severity'], 
                       message: string, metrics: any): void {
        const alert: PerformanceAlert = {
            type,
            severity,
            message,
            timestamp: Date.now(),
            metrics
        };

        this.alerts.push(alert);
        
        // Keep only last 100 alerts
        if (this.alerts.length > 100) {
            this.alerts = this.alerts.slice(-100);
        }

        // Notify callbacks
        this.alertCallbacks.forEach(callback => callback(alert));
    }

    // Operation timing methods
    startOperation(operationName: string): string {
        const operationId = `${operationName}_${Date.now()}_${Math.random()}`;
        this.operationStartTimes.set(operationId, performance.now());
        return operationId;
    }

    endOperation(operationId: string): number {
        const startTime = this.operationStartTimes.get(operationId);
        if (!startTime) return 0;

        const duration = performance.now() - startTime;
        this.operationStartTimes.delete(operationId);

        // Extract operation name from ID
        const operationName = operationId.split('_')[0];
        
        // Store operation time
        if (!this.metrics.operationTimes.has(operationName)) {
            this.metrics.operationTimes.set(operationName, []);
        }
        
        const times = this.metrics.operationTimes.get(operationName)!;
        times.push(duration);
        
        // Keep only last 100 measurements
        if (times.length > 100) {
            times.splice(0, times.length - 100);
        }

        // Check for slow operations
        if (duration > 1000) { // > 1 second
            this.createAlert('slow_operation', 'medium',
                `Slow operation detected: ${operationName} took ${duration.toFixed(0)}ms`,
                { operationName, duration }
            );
        }

        return duration;
    }

    // API call tracking
    recordAPICall(responseTime: number, success: boolean): void {
        this.metrics.apiCallStats.totalCalls++;
        
        // Update average response time
        const currentAvg = this.metrics.apiCallStats.averageResponseTime;
        const totalCalls = this.metrics.apiCallStats.totalCalls;
        this.metrics.apiCallStats.averageResponseTime = 
            (currentAvg * (totalCalls - 1) + responseTime) / totalCalls;

        // Update error rate (simple moving average over last 100 calls)
        const errorCount = success ? 0 : 1;
        const sampleSize = Math.min(totalCalls, 100);
        this.metrics.apiCallStats.errorRate = 
            (this.metrics.apiCallStats.errorRate * (sampleSize - 1) + errorCount) / sampleSize;

        // Track slow calls
        if (responseTime > 1000) {
            this.metrics.apiCallStats.slowCalls++;
        }
    }

    // Cache stats update
    updateCacheStats(hitRate: number, memoryUsage: number, evictionCount: number): void {
        this.metrics.cacheStats = {
            hitRate,
            memoryUsage,
            evictionCount
        };
    }

    // Background task tracking
    incrementBackgroundTask(type: 'active' | 'queued' | 'completed' | 'failed'): void {
        this.metrics.backgroundTasks[type]++;
    }

    decrementBackgroundTask(type: 'active' | 'queued'): void {
        if (this.metrics.backgroundTasks[type] > 0) {
            this.metrics.backgroundTasks[type]--;
        }
    }

    // Getters
    getMetrics(): PerformanceMetrics {
        return { ...this.metrics };
    }

    getAlerts(): PerformanceAlert[] {
        return [...this.alerts];
    }

    getRecentAlerts(minutes: number = 5): PerformanceAlert[] {
        const cutoff = Date.now() - (minutes * 60 * 1000);
        return this.alerts.filter(alert => alert.timestamp > cutoff);
    }

    getOperationStats(operationName: string): {
        count: number;
        average: number;
        min: number;
        max: number;
        p95: number;
    } | null {
        const times = this.metrics.operationTimes.get(operationName);
        if (!times || times.length === 0) return null;

        const sorted = [...times].sort((a, b) => a - b);
        const count = sorted.length;
        const sum = sorted.reduce((a, b) => a + b, 0);
        const average = sum / count;
        const min = sorted[0];
        const max = sorted[count - 1];
        const p95Index = Math.floor(count * 0.95);
        const p95 = sorted[p95Index];

        return { count, average, min, max, p95 };
    }

    // Alert management
    onAlert(callback: (alert: PerformanceAlert) => void): void {
        this.alertCallbacks.push(callback);
    }

    clearAlerts(): void {
        this.alerts = [];
    }

    // Performance optimization suggestions
    getOptimizationSuggestions(): string[] {
        const suggestions: string[] = [];

        if (this.metrics.memoryUsage.percentage > 70) {
            suggestions.push('Consider clearing caches or reducing cache sizes');
        }

        if (this.metrics.cacheStats.hitRate < 0.5) {
            suggestions.push('Cache hit rate is low, consider adjusting cache TTL or size');
        }

        if (this.metrics.apiCallStats.averageResponseTime > 500) {
            suggestions.push('API response times are high, consider implementing request batching');
        }

        if (this.metrics.backgroundTasks.queued > 10) {
            suggestions.push('Background task queue is growing, consider increasing processing capacity');
        }

        const slowOperations = Array.from(this.metrics.operationTimes.entries())
            .filter(([_, times]) => {
                const avg = times.reduce((a, b) => a + b, 0) / times.length;
                return avg > 200; // > 200ms average
            });

        if (slowOperations.length > 0) {
            suggestions.push(`Slow operations detected: ${slowOperations.map(([name]) => name).join(', ')}`);
        }

        return suggestions;
    }

    destroy(): void {
        if (this.monitoringInterval) {
            clearInterval(this.monitoringInterval);
        }
        this.operationStartTimes.clear();
        this.alerts = [];
        this.alertCallbacks = [];
    }
}