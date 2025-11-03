/**
 * Offline mode support and synchronization management
 * Handles offline state detection, operation queuing, and conflict resolution
 */

import { UnifiedChunk, SyncConflict, PendingChange, OfflineOperation } from '../types';
import { globalErrorHandler, ErrorSeverity } from '../errors/ErrorHandler';
import { globalRetryManager } from '../errors/RetryManager';
import { logger } from '../errors/DebugLogger';

// Offline state
export enum OfflineState {
  ONLINE = 'online',
  OFFLINE = 'offline',
  SYNCING = 'syncing',
  SYNC_ERROR = 'sync_error'
}

// Sync strategy options
export enum SyncStrategy {
  IMMEDIATE = 'immediate',
  BATCHED = 'batched',
  SCHEDULED = 'scheduled'
}

// Conflict resolution strategies
export enum ConflictResolutionStrategy {
  LOCAL_WINS = 'local_wins',
  REMOTE_WINS = 'remote_wins',
  MERGE = 'merge',
  MANUAL = 'manual'
}

// Offline configuration
export interface OfflineConfig {
  enabled: boolean;
  maxQueueSize: number;
  syncStrategy: SyncStrategy;
  batchSize: number;
  syncInterval: number;
  conflictResolution: ConflictResolutionStrategy;
  retryFailedOperations: boolean;
  maxRetryAttempts: number;
  persistQueue: boolean;
}

// Sync statistics
export interface SyncStats {
  totalOperations: number;
  successfulOperations: number;
  failedOperations: number;
  conflictedOperations: number;
  lastSyncTime: Date | null;
  averageSyncTime: number;
  queueSize: number;
}

// Operation result
export interface OperationResult {
  success: boolean;
  operation: OfflineOperation;
  error?: Error;
  conflict?: SyncConflict;
  retryCount: number;
}

// Sync batch result
export interface SyncBatchResult {
  totalOperations: number;
  successfulOperations: number;
  failedOperations: number;
  conflicts: SyncConflict[];
  errors: Error[];
  duration: number;
}

// Network status listener
export type NetworkStatusListener = (isOnline: boolean) => void;

// Sync progress listener
export type SyncProgressListener = (progress: {
  current: number;
  total: number;
  operation: OfflineOperation;
}) => void;

// Conflict listener
export type ConflictListener = (conflict: SyncConflict) => void;

// Default offline configuration
const DEFAULT_OFFLINE_CONFIG: OfflineConfig = {
  enabled: true,
  maxQueueSize: 1000,
  syncStrategy: SyncStrategy.BATCHED,
  batchSize: 10,
  syncInterval: 30000, // 30 seconds
  conflictResolution: ConflictResolutionStrategy.MANUAL,
  retryFailedOperations: true,
  maxRetryAttempts: 3,
  persistQueue: true
};

export class OfflineManager {
  private config: OfflineConfig;
  private state: OfflineState = OfflineState.ONLINE;
  private operationQueue: OfflineOperation[] = [];
  private failedOperations: OfflineOperation[] = [];
  private conflicts: SyncConflict[] = [];
  private syncTimer: NodeJS.Timeout | null = null;
  private stats: SyncStats = {
    totalOperations: 0,
    successfulOperations: 0,
    failedOperations: 0,
    conflictedOperations: 0,
    lastSyncTime: null,
    averageSyncTime: 0,
    queueSize: 0
  };

  // Event listeners
  private networkStatusListeners: NetworkStatusListener[] = [];
  private syncProgressListeners: SyncProgressListener[] = [];
  private conflictListeners: ConflictListener[] = [];

  // Sync function (to be provided by the calling component)
  private syncFunction: ((operations: OfflineOperation[]) => Promise<OperationResult[]>) | null = null;

  constructor(config: Partial<OfflineConfig> = {}) {
    this.config = { ...DEFAULT_OFFLINE_CONFIG, ...config };
    
    if (this.config.enabled) {
      this.initialize();
    }
  }

  /**
   * Initialize offline manager
   */
  private initialize(): void {
    this.setupNetworkMonitoring();
    this.loadPersistedQueue();
    this.startSyncTimer();
    
    logger.info('OfflineManager', 'initialize', 'Offline manager initialized', {
      config: this.config
    });
  }

  /**
   * Check if currently online
   */
  isOnline(): boolean {
    return navigator.onLine && this.state !== OfflineState.OFFLINE;
  }

  /**
   * Get current offline state
   */
  getState(): OfflineState {
    return this.state;
  }

  /**
   * Queue an operation for offline execution
   */
  queueOperation(operation: OfflineOperation): void {
    if (!this.config.enabled) {
      logger.warn('OfflineManager', 'queueOperation', 'Offline manager is disabled');
      return;
    }

    // Check queue size limit
    if (this.operationQueue.length >= this.config.maxQueueSize) {
      logger.warn('OfflineManager', 'queueOperation', 'Queue size limit reached, removing oldest operation');
      this.operationQueue.shift();
    }

    // Add operation to queue
    this.operationQueue.push(operation);
    this.stats.queueSize = this.operationQueue.length;

    logger.debug('OfflineManager', 'queueOperation', 'Operation queued', {
      operationId: operation.id,
      type: operation.type,
      queueSize: this.operationQueue.length
    });

    // Persist queue if enabled
    if (this.config.persistQueue) {
      this.persistQueue();
    }

    // Try immediate sync if online and strategy allows
    if (this.isOnline() && this.config.syncStrategy === SyncStrategy.IMMEDIATE) {
      this.syncWhenOnline();
    }
  }

  /**
   * Sync queued operations when online
   */
  async syncWhenOnline(): Promise<SyncBatchResult> {
    if (!this.isOnline() || this.state === OfflineState.SYNCING) {
      logger.debug('OfflineManager', 'syncWhenOnline', 'Skipping sync - offline or already syncing');
      return this.createEmptySyncResult();
    }

    if (this.operationQueue.length === 0 && this.failedOperations.length === 0) {
      logger.debug('OfflineManager', 'syncWhenOnline', 'No operations to sync');
      return this.createEmptySyncResult();
    }

    this.setState(OfflineState.SYNCING);
    const startTime = Date.now();

    try {
      // Combine queued and failed operations
      const operationsToSync = [
        ...this.operationQueue,
        ...this.failedOperations
      ].sort((a, b) => {
        // Sort by priority (higher first), then by timestamp (older first)
        if (a.priority !== b.priority) {
          return b.priority - a.priority;
        }
        return a.timestamp.getTime() - b.timestamp.getTime();
      });

      const batchSize = this.config.batchSize;
      const batches = this.createBatches(operationsToSync, batchSize);
      
      let totalSuccessful = 0;
      let totalFailed = 0;
      const allConflicts: SyncConflict[] = [];
      const allErrors: Error[] = [];

      // Process batches
      for (let i = 0; i < batches.length; i++) {
        const batch = batches[i];
        
        logger.debug('OfflineManager', 'syncWhenOnline', `Processing batch ${i + 1}/${batches.length}`, {
          batchSize: batch.length
        });

        try {
          const batchResult = await this.processBatch(batch, i, batches.length);
          
          totalSuccessful += batchResult.successfulOperations;
          totalFailed += batchResult.failedOperations;
          allConflicts.push(...batchResult.conflicts);
          allErrors.push(...batchResult.errors);

        } catch (error) {
          logger.error('OfflineManager', 'syncWhenOnline', `Batch ${i + 1} failed completely`, error);
          totalFailed += batch.length;
          allErrors.push(error instanceof Error ? error : new Error(String(error)));
        }
      }

      // Update statistics
      const duration = Date.now() - startTime;
      this.updateSyncStats(totalSuccessful, totalFailed, allConflicts.length, duration);

      // Handle conflicts
      if (allConflicts.length > 0) {
        await this.handleConflicts(allConflicts);
      }

      const result: SyncBatchResult = {
        totalOperations: operationsToSync.length,
        successfulOperations: totalSuccessful,
        failedOperations: totalFailed,
        conflicts: allConflicts,
        errors: allErrors,
        duration
      };

      logger.info('OfflineManager', 'syncWhenOnline', 'Sync completed', result);

      this.setState(totalFailed > 0 ? OfflineState.SYNC_ERROR : OfflineState.ONLINE);
      return result;

    } catch (error) {
      logger.error('OfflineManager', 'syncWhenOnline', 'Sync failed', error);
      this.setState(OfflineState.SYNC_ERROR);
      
      await globalErrorHandler.handleError(error, {
        component: 'OfflineManager',
        operation: 'syncWhenOnline'
      });

      throw error;
    }
  }

  /**
   * Handle sync conflicts
   */
  async handleConflicts(conflicts: SyncConflict[]): Promise<void> {
    this.conflicts.push(...conflicts);

    for (const conflict of conflicts) {
      // Notify conflict listeners
      this.conflictListeners.forEach(listener => {
        try {
          listener(conflict);
        } catch (error) {
          logger.error('OfflineManager', 'handleConflicts', 'Conflict listener error', error);
        }
      });

      // Apply resolution strategy
      try {
        await this.resolveConflict(conflict);
      } catch (error) {
        logger.error('OfflineManager', 'handleConflicts', 'Conflict resolution failed', error, {
          conflictId: conflict.chunkId
        });
      }
    }
  }

  /**
   * Resolve a single conflict based on strategy
   */
  private async resolveConflict(conflict: SyncConflict): Promise<void> {
    switch (this.config.conflictResolution) {
      case ConflictResolutionStrategy.LOCAL_WINS:
        // Keep local version, no action needed
        logger.info('OfflineManager', 'resolveConflict', 'Conflict resolved: local wins', {
          chunkId: conflict.chunkId
        });
        break;

      case ConflictResolutionStrategy.REMOTE_WINS:
        // Accept remote version
        await this.acceptRemoteVersion(conflict);
        break;

      case ConflictResolutionStrategy.MERGE:
        // Attempt to merge versions
        await this.mergeVersions(conflict);
        break;

      case ConflictResolutionStrategy.MANUAL:
        // Leave for manual resolution
        logger.info('OfflineManager', 'resolveConflict', 'Conflict marked for manual resolution', {
          chunkId: conflict.chunkId
        });
        break;
    }
  }

  /**
   * Accept remote version in conflict
   */
  private async acceptRemoteVersion(conflict: SyncConflict): Promise<void> {
    // This would typically update the local content with the remote version
    logger.info('OfflineManager', 'acceptRemoteVersion', 'Accepting remote version', {
      chunkId: conflict.chunkId
    });
    
    // Remove from conflicts list
    this.conflicts = this.conflicts.filter(c => c.chunkId !== conflict.chunkId);
  }

  /**
   * Merge conflicting versions
   */
  private async mergeVersions(conflict: SyncConflict): Promise<void> {
    // Simple merge strategy - this could be more sophisticated
    const merged: UnifiedChunk = {
      ...conflict.localVersion,
      contents: this.mergeContent(
        conflict.localVersion.contents,
        conflict.remoteVersion.contents
      ),
      lastUpdated: new Date()
    };

    logger.info('OfflineManager', 'mergeVersions', 'Versions merged', {
      chunkId: conflict.chunkId
    });

    // Queue the merged version for sync
    this.queueOperation({
      id: `merge_${conflict.chunkId}_${Date.now()}`,
      type: 'update',
      data: merged,
      timestamp: new Date(),
      priority: 10 // High priority for merged content
    });

    // Remove from conflicts list
    this.conflicts = this.conflicts.filter(c => c.chunkId !== conflict.chunkId);
  }

  /**
   * Simple content merge (could be enhanced with more sophisticated algorithms)
   */
  private mergeContent(localContent: string, remoteContent: string): string {
    // Very basic merge - in practice, you'd want a more sophisticated algorithm
    if (localContent === remoteContent) {
      return localContent;
    }

    // For now, just concatenate with a separator
    return `${localContent}\n\n--- MERGED CONTENT ---\n\n${remoteContent}`;
  }

  /**
   * Set sync function to be used for operations
   */
  setSyncFunction(syncFn: (operations: OfflineOperation[]) => Promise<OperationResult[]>): void {
    this.syncFunction = syncFn;
    logger.debug('OfflineManager', 'setSyncFunction', 'Sync function set');
  }

  /**
   * Get sync statistics
   */
  getStats(): SyncStats {
    return {
      ...this.stats,
      queueSize: this.operationQueue.length
    };
  }

  /**
   * Get pending operations
   */
  getPendingOperations(): OfflineOperation[] {
    return [...this.operationQueue];
  }

  /**
   * Get failed operations
   */
  getFailedOperations(): OfflineOperation[] {
    return [...this.failedOperations];
  }

  /**
   * Get unresolved conflicts
   */
  getConflicts(): SyncConflict[] {
    return [...this.conflicts];
  }

  /**
   * Clear all queued operations
   */
  clearQueue(): void {
    this.operationQueue = [];
    this.stats.queueSize = 0;
    
    if (this.config.persistQueue) {
      this.persistQueue();
    }
    
    logger.info('OfflineManager', 'clearQueue', 'Operation queue cleared');
  }

  /**
   * Clear failed operations
   */
  clearFailedOperations(): void {
    this.failedOperations = [];
    logger.info('OfflineManager', 'clearFailedOperations', 'Failed operations cleared');
  }

  /**
   * Manually resolve a conflict
   */
  resolveConflictManually(chunkId: string, resolution: UnifiedChunk): void {
    const conflictIndex = this.conflicts.findIndex(c => c.chunkId === chunkId);
    if (conflictIndex === -1) {
      logger.warn('OfflineManager', 'resolveConflictManually', 'Conflict not found', { chunkId });
      return;
    }

    // Remove the conflict
    this.conflicts.splice(conflictIndex, 1);

    // Queue the resolution for sync
    this.queueOperation({
      id: `manual_resolve_${chunkId}_${Date.now()}`,
      type: 'update',
      data: resolution,
      timestamp: new Date(),
      priority: 10
    });

    logger.info('OfflineManager', 'resolveConflictManually', 'Conflict resolved manually', { chunkId });
  }

  /**
   * Add network status listener
   */
  addNetworkStatusListener(listener: NetworkStatusListener): void {
    this.networkStatusListeners.push(listener);
  }

  /**
   * Remove network status listener
   */
  removeNetworkStatusListener(listener: NetworkStatusListener): void {
    const index = this.networkStatusListeners.indexOf(listener);
    if (index > -1) {
      this.networkStatusListeners.splice(index, 1);
    }
  }

  /**
   * Add sync progress listener
   */
  addSyncProgressListener(listener: SyncProgressListener): void {
    this.syncProgressListeners.push(listener);
  }

  /**
   * Remove sync progress listener
   */
  removeSyncProgressListener(listener: SyncProgressListener): void {
    const index = this.syncProgressListeners.indexOf(listener);
    if (index > -1) {
      this.syncProgressListeners.splice(index, 1);
    }
  }

  /**
   * Add conflict listener
   */
  addConflictListener(listener: ConflictListener): void {
    this.conflictListeners.push(listener);
  }

  /**
   * Remove conflict listener
   */
  removeConflictListener(listener: ConflictListener): void {
    const index = this.conflictListeners.indexOf(listener);
    if (index > -1) {
      this.conflictListeners.splice(index, 1);
    }
  }

  /**
   * Update configuration
   */
  updateConfig(config: Partial<OfflineConfig>): void {
    const oldEnabled = this.config.enabled;
    this.config = { ...this.config, ...config };

    if (this.config.enabled && !oldEnabled) {
      this.initialize();
    } else if (!this.config.enabled && oldEnabled) {
      this.cleanup();
    }

    logger.info('OfflineManager', 'updateConfig', 'Configuration updated', config);
  }

  /**
   * Cleanup resources
   */
  cleanup(): void {
    if (this.syncTimer) {
      clearInterval(this.syncTimer);
      this.syncTimer = null;
    }

    // Remove event listeners
    window.removeEventListener('online', this.handleOnline);
    window.removeEventListener('offline', this.handleOffline);

    logger.info('OfflineManager', 'cleanup', 'Offline manager cleaned up');
  }

  // Private methods

  private setState(newState: OfflineState): void {
    if (this.state !== newState) {
      const oldState = this.state;
      this.state = newState;
      
      logger.debug('OfflineManager', 'setState', 'State changed', {
        from: oldState,
        to: newState
      });
    }
  }

  private setupNetworkMonitoring(): void {
    // Initial state
    this.setState(navigator.onLine ? OfflineState.ONLINE : OfflineState.OFFLINE);

    // Listen for network changes
    window.addEventListener('online', this.handleOnline);
    window.addEventListener('offline', this.handleOffline);
  }

  private handleOnline = (): void => {
    logger.info('OfflineManager', 'handleOnline', 'Network connection restored');
    this.setState(OfflineState.ONLINE);
    
    // Notify listeners
    this.networkStatusListeners.forEach(listener => {
      try {
        listener(true);
      } catch (error) {
        logger.error('OfflineManager', 'handleOnline', 'Network status listener error', error);
      }
    });

    // Trigger sync if there are pending operations
    if (this.operationQueue.length > 0 || this.failedOperations.length > 0) {
      this.syncWhenOnline().catch(error => {
        logger.error('OfflineManager', 'handleOnline', 'Auto-sync failed', error);
      });
    }
  };

  private handleOffline = (): void => {
    logger.info('OfflineManager', 'handleOffline', 'Network connection lost');
    this.setState(OfflineState.OFFLINE);
    
    // Notify listeners
    this.networkStatusListeners.forEach(listener => {
      try {
        listener(false);
      } catch (error) {
        logger.error('OfflineManager', 'handleOffline', 'Network status listener error', error);
      }
    });
  };

  private startSyncTimer(): void {
    if (this.config.syncStrategy === SyncStrategy.SCHEDULED && this.config.syncInterval > 0) {
      this.syncTimer = setInterval(() => {
        if (this.isOnline() && this.operationQueue.length > 0) {
          this.syncWhenOnline().catch(error => {
            logger.error('OfflineManager', 'startSyncTimer', 'Scheduled sync failed', error);
          });
        }
      }, this.config.syncInterval);
    }
  }

  private createBatches<T>(items: T[], batchSize: number): T[][] {
    const batches: T[][] = [];
    for (let i = 0; i < items.length; i += batchSize) {
      batches.push(items.slice(i, i + batchSize));
    }
    return batches;
  }

  private async processBatch(
    batch: OfflineOperation[],
    batchIndex: number,
    totalBatches: number
  ): Promise<SyncBatchResult> {
    if (!this.syncFunction) {
      throw new Error('Sync function not set');
    }

    // Notify progress listeners
    this.syncProgressListeners.forEach(listener => {
      try {
        batch.forEach((operation, index) => {
          listener({
            current: batchIndex * this.config.batchSize + index + 1,
            total: totalBatches * this.config.batchSize,
            operation
          });
        });
      } catch (error) {
        logger.error('OfflineManager', 'processBatch', 'Progress listener error', error);
      }
    });

    const startTime = Date.now();
    const results = await this.syncFunction(batch);
    const duration = Date.now() - startTime;

    let successfulOperations = 0;
    let failedOperations = 0;
    const conflicts: SyncConflict[] = [];
    const errors: Error[] = [];

    for (const result of results) {
      if (result.success) {
        successfulOperations++;
        // Remove from queue
        this.removeOperationFromQueue(result.operation);
      } else {
        failedOperations++;
        
        if (result.conflict) {
          conflicts.push(result.conflict);
        }
        
        if (result.error) {
          errors.push(result.error);
        }

        // Handle failed operation
        await this.handleFailedOperation(result);
      }
    }

    return {
      totalOperations: batch.length,
      successfulOperations,
      failedOperations,
      conflicts,
      errors,
      duration
    };
  }

  private removeOperationFromQueue(operation: OfflineOperation): void {
    // Remove from main queue
    const queueIndex = this.operationQueue.findIndex(op => op.id === operation.id);
    if (queueIndex > -1) {
      this.operationQueue.splice(queueIndex, 1);
    }

    // Remove from failed operations
    const failedIndex = this.failedOperations.findIndex(op => op.id === operation.id);
    if (failedIndex > -1) {
      this.failedOperations.splice(failedIndex, 1);
    }

    this.stats.queueSize = this.operationQueue.length;
  }

  private async handleFailedOperation(result: OperationResult): Promise<void> {
    const { operation, error } = result;

    if (this.config.retryFailedOperations && result.retryCount < this.config.maxRetryAttempts) {
      // Add to failed operations for retry
      const existingIndex = this.failedOperations.findIndex(op => op.id === operation.id);
      if (existingIndex === -1) {
        this.failedOperations.push(operation);
      }
      
      logger.warn('OfflineManager', 'handleFailedOperation', 'Operation will be retried', {
        operationId: operation.id,
        retryCount: result.retryCount,
        error: error?.message
      });
    } else {
      // Remove from all queues - max retries reached
      this.removeOperationFromQueue(operation);
      
      logger.error('OfflineManager', 'handleFailedOperation', 'Operation failed permanently', error, {
        operationId: operation.id,
        retryCount: result.retryCount
      });
    }
  }

  private updateSyncStats(
    successful: number,
    failed: number,
    conflicts: number,
    duration: number
  ): void {
    this.stats.totalOperations += successful + failed;
    this.stats.successfulOperations += successful;
    this.stats.failedOperations += failed;
    this.stats.conflictedOperations += conflicts;
    this.stats.lastSyncTime = new Date();
    
    // Update average sync time
    if (this.stats.averageSyncTime === 0) {
      this.stats.averageSyncTime = duration;
    } else {
      this.stats.averageSyncTime = (this.stats.averageSyncTime + duration) / 2;
    }
  }

  private createEmptySyncResult(): SyncBatchResult {
    return {
      totalOperations: 0,
      successfulOperations: 0,
      failedOperations: 0,
      conflicts: [],
      errors: [],
      duration: 0
    };
  }

  private persistQueue(): void {
    try {
      const queueData = {
        operations: this.operationQueue,
        failedOperations: this.failedOperations,
        timestamp: new Date().toISOString()
      };
      
      localStorage.setItem('obsidian-ink-plugin-offline-queue', JSON.stringify(queueData));
    } catch (error) {
      logger.error('OfflineManager', 'persistQueue', 'Failed to persist queue', error);
    }
  }

  private loadPersistedQueue(): void {
    if (!this.config.persistQueue) return;

    try {
      const queueData = localStorage.getItem('obsidian-ink-plugin-offline-queue');
      if (queueData) {
        const parsed = JSON.parse(queueData);
        
        // Restore operations with proper Date objects
        this.operationQueue = parsed.operations.map((op: any) => ({
          ...op,
          timestamp: new Date(op.timestamp)
        }));
        
        this.failedOperations = parsed.failedOperations.map((op: any) => ({
          ...op,
          timestamp: new Date(op.timestamp)
        }));

        this.stats.queueSize = this.operationQueue.length;
        
        logger.info('OfflineManager', 'loadPersistedQueue', 'Queue restored from storage', {
          queueSize: this.operationQueue.length,
          failedSize: this.failedOperations.length
        });
      }
    } catch (error) {
      logger.error('OfflineManager', 'loadPersistedQueue', 'Failed to load persisted queue', error);
    }
  }
}

// Global offline manager instance
export const globalOfflineManager = new OfflineManager();