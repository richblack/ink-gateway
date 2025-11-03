export interface BackgroundTask {
    id: string;
    name: string;
    priority: number;
    execute: () => Promise<any>;
    onSuccess?: (result: any) => void;
    onError?: (error: Error) => void;
    retryAttempts: number;
    retryDelay: number;
    timeout?: number;
    createdAt: number;
    startedAt?: number;
    completedAt?: number;
    status: 'queued' | 'running' | 'completed' | 'failed' | 'cancelled';
}

export interface TaskManagerConfig {
    maxConcurrentTasks: number;
    maxQueueSize: number;
    defaultRetryAttempts: number;
    defaultRetryDelay: number;
    defaultTimeout: number;
    cleanupInterval: number; // How often to clean completed tasks (ms)
    maxCompletedTasks: number; // Max completed tasks to keep in memory
}

export interface TaskStats {
    queued: number;
    running: number;
    completed: number;
    failed: number;
    cancelled: number;
    totalProcessed: number;
    averageExecutionTime: number;
    successRate: number;
}

export class BackgroundTaskManager {
    private tasks = new Map<string, BackgroundTask>();
    private queue: string[] = [];
    private running = new Set<string>();
    private config: TaskManagerConfig;
    private cleanupTimer?: NodeJS.Timeout;
    private executionTimes: number[] = [];

    constructor(config: Partial<TaskManagerConfig> = {}) {
        this.config = {
            maxConcurrentTasks: 3,
            maxQueueSize: 100,
            defaultRetryAttempts: 3,
            defaultRetryDelay: 1000,
            defaultTimeout: 30000, // 30 seconds
            cleanupInterval: 5 * 60 * 1000, // 5 minutes
            maxCompletedTasks: 50,
            ...config
        };

        this.startCleanupTimer();
    }

    private startCleanupTimer(): void {
        this.cleanupTimer = setInterval(() => {
            this.cleanup();
        }, this.config.cleanupInterval);
    }

    private cleanup(): void {
        const completedTasks = Array.from(this.tasks.values())
            .filter(task => task.status === 'completed' || task.status === 'failed')
            .sort((a, b) => (b.completedAt || 0) - (a.completedAt || 0));

        // Keep only the most recent completed tasks
        const toRemove = completedTasks.slice(this.config.maxCompletedTasks);
        toRemove.forEach(task => this.tasks.delete(task.id));
    }

    // Add a task to the queue
    addTask(
        name: string,
        execute: () => Promise<any>,
        options: {
            priority?: number;
            retryAttempts?: number;
            retryDelay?: number;
            timeout?: number;
            onSuccess?: (result: any) => void;
            onError?: (error: Error) => void;
        } = {}
    ): string {
        if (this.queue.length >= this.config.maxQueueSize) {
            throw new Error('Task queue is full');
        }

        const taskId = `task_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
        
        const task: BackgroundTask = {
            id: taskId,
            name,
            priority: options.priority || 0,
            execute,
            onSuccess: options.onSuccess,
            onError: options.onError,
            retryAttempts: options.retryAttempts ?? this.config.defaultRetryAttempts,
            retryDelay: options.retryDelay ?? this.config.defaultRetryDelay,
            timeout: options.timeout ?? this.config.defaultTimeout,
            createdAt: Date.now(),
            status: 'queued'
        };

        this.tasks.set(taskId, task);
        this.queue.push(taskId);
        this.sortQueue();
        this.processQueue();

        return taskId;
    }

    private sortQueue(): void {
        this.queue.sort((a, b) => {
            const taskA = this.tasks.get(a);
            const taskB = this.tasks.get(b);
            
            if (!taskA || !taskB) return 0;
            
            // Sort by priority first, then by creation time
            if (taskA.priority !== taskB.priority) {
                return taskB.priority - taskA.priority; // Higher priority first
            }
            return taskA.createdAt - taskB.createdAt; // Older tasks first
        });
    }

    private async processQueue(): Promise<void> {
        while (this.queue.length > 0 && this.running.size < this.config.maxConcurrentTasks) {
            const taskId = this.queue.shift();
            if (!taskId) continue;

            const task = this.tasks.get(taskId);
            if (!task || task.status !== 'queued') continue;

            this.running.add(taskId);
            task.status = 'running';
            task.startedAt = Date.now();

            // Execute task in background
            this.executeTask(task).finally(() => {
                this.running.delete(taskId);
                this.processQueue(); // Process next task
            });
        }
    }

    private async executeTask(task: BackgroundTask): Promise<void> {
        let attempt = 0;
        
        while (attempt < task.retryAttempts) {
            try {
                const startTime = performance.now();
                
                // Create timeout promise
                const timeoutPromise = task.timeout 
                    ? new Promise((_, reject) => 
                        setTimeout(() => reject(new Error('Task timeout')), task.timeout)
                      )
                    : null;

                // Execute task with timeout
                const result = timeoutPromise
                    ? await Promise.race([task.execute(), timeoutPromise])
                    : await task.execute();

                const executionTime = performance.now() - startTime;
                this.executionTimes.push(executionTime);
                
                // Keep only last 100 execution times
                if (this.executionTimes.length > 100) {
                    this.executionTimes = this.executionTimes.slice(-100);
                }

                task.status = 'completed';
                task.completedAt = Date.now();
                
                if (task.onSuccess) {
                    try {
                        task.onSuccess(result);
                    } catch (error) {
                        console.warn('Error in task success callback:', error);
                    }
                }
                
                return;
            } catch (error) {
                attempt++;
                
                if (attempt >= task.retryAttempts) {
                    task.status = 'failed';
                    task.completedAt = Date.now();
                    
                    if (task.onError) {
                        try {
                            task.onError(error as Error);
                        } catch (callbackError) {
                            console.warn('Error in task error callback:', callbackError);
                        }
                    }
                    
                    return;
                }

                // Wait before retry
                await new Promise(resolve => 
                    setTimeout(resolve, task.retryDelay * attempt)
                );
            }
        }
    }

    // Cancel a task
    cancelTask(taskId: string): boolean {
        const task = this.tasks.get(taskId);
        if (!task) return false;

        if (task.status === 'queued') {
            const queueIndex = this.queue.indexOf(taskId);
            if (queueIndex !== -1) {
                this.queue.splice(queueIndex, 1);
            }
            task.status = 'cancelled';
            return true;
        }

        if (task.status === 'running') {
            // Can't cancel running tasks, but mark as cancelled
            task.status = 'cancelled';
            return true;
        }

        return false;
    }

    // Get task status
    getTask(taskId: string): BackgroundTask | null {
        return this.tasks.get(taskId) || null;
    }

    // Get all tasks with optional status filter
    getTasks(status?: BackgroundTask['status']): BackgroundTask[] {
        const tasks = Array.from(this.tasks.values());
        return status ? tasks.filter(task => task.status === status) : tasks;
    }

    // Get task statistics
    getStats(): TaskStats {
        const tasks = Array.from(this.tasks.values());
        const queued = tasks.filter(t => t.status === 'queued').length;
        const running = tasks.filter(t => t.status === 'running').length;
        const completed = tasks.filter(t => t.status === 'completed').length;
        const failed = tasks.filter(t => t.status === 'failed').length;
        const cancelled = tasks.filter(t => t.status === 'cancelled').length;
        
        const totalProcessed = completed + failed;
        const successRate = totalProcessed > 0 ? completed / totalProcessed : 0;
        
        const averageExecutionTime = this.executionTimes.length > 0
            ? this.executionTimes.reduce((sum, time) => sum + time, 0) / this.executionTimes.length
            : 0;

        return {
            queued,
            running,
            completed,
            failed,
            cancelled,
            totalProcessed,
            averageExecutionTime,
            successRate
        };
    }

    // Wait for all running tasks to complete
    async waitForCompletion(): Promise<void> {
        while (this.running.size > 0 || this.queue.length > 0) {
            await new Promise(resolve => setTimeout(resolve, 100));
        }
    }

    // Clear all tasks
    clear(): void {
        // Cancel queued tasks
        this.queue.forEach(taskId => this.cancelTask(taskId));
        this.queue = [];
        
        // Clear completed/failed tasks
        const toRemove = Array.from(this.tasks.entries())
            .filter(([_, task]) => task.status === 'completed' || task.status === 'failed' || task.status === 'cancelled')
            .map(([id]) => id);
        
        toRemove.forEach(id => this.tasks.delete(id));
    }

    // Pause task processing
    pause(): void {
        // Implementation would involve stopping the processQueue method
        // For now, we can clear the queue without cancelling tasks
        const queuedTasks = [...this.queue];
        this.queue = [];
        
        // Store paused tasks for later resume
        (this as any)._pausedTasks = queuedTasks;
    }

    // Resume task processing
    resume(): void {
        const pausedTasks = (this as any)._pausedTasks;
        if (pausedTasks) {
            this.queue.push(...pausedTasks);
            delete (this as any)._pausedTasks;
            this.sortQueue();
            this.processQueue();
        }
    }

    // Update configuration
    updateConfig(newConfig: Partial<TaskManagerConfig>): void {
        this.config = { ...this.config, ...newConfig };
    }

    // Destroy the task manager
    destroy(): void {
        if (this.cleanupTimer) {
            clearInterval(this.cleanupTimer);
        }
        
        this.clear();
        this.tasks.clear();
        this.executionTimes = [];
    }
}