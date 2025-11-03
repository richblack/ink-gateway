export interface BatchConfig {
    maxBatchSize: number;
    maxWaitTime: number; // Maximum time to wait before processing batch (ms)
    maxConcurrentBatches: number;
    retryAttempts: number;
    retryDelay: number;
}

export interface BatchItem<T, R> {
    id: string;
    data: T;
    resolve: (result: R) => void;
    reject: (error: Error) => void;
    timestamp: number;
    priority: number;
}

export interface BatchResult<R> {
    success: boolean;
    results: Map<string, R>;
    errors: Map<string, Error>;
    processingTime: number;
}

export class BatchProcessor<T, R> {
    private queue: BatchItem<T, R>[] = [];
    private activeBatches = new Set<Promise<void>>();
    private config: BatchConfig;
    private processor: (items: T[]) => Promise<R[]>;
    private batchTimer?: NodeJS.Timeout;
    private isProcessing = false;

    constructor(
        processor: (items: T[]) => Promise<R[]>,
        config: Partial<BatchConfig> = {}
    ) {
        this.processor = processor;
        this.config = {
            maxBatchSize: 10,
            maxWaitTime: 1000, // 1 second
            maxConcurrentBatches: 3,
            retryAttempts: 3,
            retryDelay: 1000,
            ...config
        };
    }

    // Add item to batch queue
    async add(id: string, data: T, priority: number = 0): Promise<R> {
        return new Promise<R>((resolve, reject) => {
            const item: BatchItem<T, R> = {
                id,
                data,
                resolve,
                reject,
                timestamp: Date.now(),
                priority
            };

            this.queue.push(item);
            this.sortQueue();
            this.scheduleProcessing();
        });
    }

    // Add multiple items at once
    async addBatch(items: Array<{ id: string; data: T; priority?: number }>): Promise<Map<string, R>> {
        const promises = items.map(item => 
            this.add(item.id, item.data, item.priority || 0)
        );

        const results = await Promise.allSettled(promises);
        const resultMap = new Map<string, R>();

        items.forEach((item, index) => {
            const result = results[index];
            if (result.status === 'fulfilled') {
                resultMap.set(item.id, result.value);
            }
        });

        return resultMap;
    }

    private sortQueue(): void {
        this.queue.sort((a, b) => {
            // Sort by priority first, then by timestamp
            if (a.priority !== b.priority) {
                return b.priority - a.priority; // Higher priority first
            }
            return a.timestamp - b.timestamp; // Older items first
        });
    }

    private scheduleProcessing(): void {
        if (this.isProcessing) return;

        // Process immediately if batch is full
        if (this.queue.length >= this.config.maxBatchSize) {
            this.processBatch();
            return;
        }

        // Schedule processing after wait time
        if (this.batchTimer) {
            clearTimeout(this.batchTimer);
        }

        this.batchTimer = setTimeout(() => {
            if (this.queue.length > 0) {
                this.processBatch();
            }
        }, this.config.maxWaitTime);
    }

    private async processBatch(): Promise<void> {
        if (this.isProcessing || this.queue.length === 0) return;

        // Wait if too many concurrent batches
        while (this.activeBatches.size >= this.config.maxConcurrentBatches) {
            await new Promise(resolve => setTimeout(resolve, 100));
        }

        this.isProcessing = true;

        // Extract batch from queue
        const batchSize = Math.min(this.queue.length, this.config.maxBatchSize);
        const batch = this.queue.splice(0, batchSize);

        if (batch.length === 0) {
            this.isProcessing = false;
            return;
        }

        // Process batch
        const batchPromise = this.executeBatch(batch);
        this.activeBatches.add(batchPromise);

        try {
            await batchPromise;
        } finally {
            this.activeBatches.delete(batchPromise);
            this.isProcessing = false;

            // Schedule next batch if queue is not empty
            if (this.queue.length > 0) {
                this.scheduleProcessing();
            }
        }
    }

    private async executeBatch(batch: BatchItem<T, R>[]): Promise<void> {
        const startTime = performance.now();
        let attempt = 0;

        while (attempt < this.config.retryAttempts) {
            try {
                const data = batch.map(item => item.data);
                const results = await this.processor(data);

                // Resolve all items in batch
                batch.forEach((item, index) => {
                    if (index < results.length) {
                        item.resolve(results[index]);
                    } else {
                        item.reject(new Error('Result not found for batch item'));
                    }
                });

                return;
            } catch (error) {
                attempt++;
                
                if (attempt >= this.config.retryAttempts) {
                    // Reject all items in batch
                    batch.forEach(item => {
                        item.reject(error as Error);
                    });
                    return;
                }

                // Wait before retry
                await new Promise(resolve => 
                    setTimeout(resolve, this.config.retryDelay * attempt)
                );
            }
        }
    }

    // Get current queue status
    getStatus(): {
        queueSize: number;
        activeBatches: number;
        oldestItemAge: number;
        averageWaitTime: number;
    } {
        const now = Date.now();
        const oldestItemAge = this.queue.length > 0 
            ? now - Math.min(...this.queue.map(item => item.timestamp))
            : 0;

        const totalWaitTime = this.queue.reduce((sum, item) => sum + (now - item.timestamp), 0);
        const averageWaitTime = this.queue.length > 0 ? totalWaitTime / this.queue.length : 0;

        return {
            queueSize: this.queue.length,
            activeBatches: this.activeBatches.size,
            oldestItemAge,
            averageWaitTime
        };
    }

    // Force process current queue
    async flush(): Promise<void> {
        if (this.batchTimer) {
            clearTimeout(this.batchTimer);
            this.batchTimer = undefined;
        }

        while (this.queue.length > 0 || this.activeBatches.size > 0) {
            if (this.queue.length > 0 && !this.isProcessing) {
                await this.processBatch();
            }
            
            // Wait for active batches to complete
            if (this.activeBatches.size > 0) {
                await Promise.all(Array.from(this.activeBatches));
            }
        }
    }

    // Clear queue and reject all pending items
    clear(): void {
        if (this.batchTimer) {
            clearTimeout(this.batchTimer);
            this.batchTimer = undefined;
        }

        // Reject all queued items
        this.queue.forEach(item => {
            item.reject(new Error('Batch processor cleared'));
        });

        this.queue = [];
    }

    // Update configuration
    updateConfig(newConfig: Partial<BatchConfig>): void {
        this.config = { ...this.config, ...newConfig };
    }
}

// Specialized batch processors for common operations

export class APIBatchProcessor extends BatchProcessor<any, any> {
    constructor(
        apiCall: (items: any[]) => Promise<any[]>,
        config: Partial<BatchConfig> = {}
    ) {
        super(apiCall, {
            maxBatchSize: 20,
            maxWaitTime: 500,
            maxConcurrentBatches: 2,
            retryAttempts: 3,
            retryDelay: 1000,
            ...config
        });
    }
}

export class SearchBatchProcessor extends BatchProcessor<string, any> {
    constructor(
        searchFunction: (queries: string[]) => Promise<any[]>,
        config: Partial<BatchConfig> = {}
    ) {
        super(searchFunction, {
            maxBatchSize: 5, // Smaller batches for search
            maxWaitTime: 200, // Faster processing for search
            maxConcurrentBatches: 3,
            retryAttempts: 2,
            retryDelay: 500,
            ...config
        });
    }
}

export class ContentProcessingBatchProcessor extends BatchProcessor<any, any> {
    constructor(
        processingFunction: (content: any[]) => Promise<any[]>,
        config: Partial<BatchConfig> = {}
    ) {
        super(processingFunction, {
            maxBatchSize: 10,
            maxWaitTime: 1000,
            maxConcurrentBatches: 2,
            retryAttempts: 3,
            retryDelay: 2000,
            ...config
        });
    }
}