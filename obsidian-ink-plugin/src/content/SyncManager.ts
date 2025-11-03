/**
 * Sync Manager for handling content synchronization with Ink Gateway
 * Manages sync state, conflict resolution, and offline operations
 */

import { 
  SyncState, 
  PendingChange, 
  SyncConflict, 
  ConflictResolution,
  UnifiedChunk,
  SyncResult,
  PluginError,
  ErrorType
} from '../types';
import { 
  IInkGatewayClient, 
  ILogger, 
  IEventManager, 
  ICacheManager,
  IOfflineManager 
} from '../interfaces';

export interface SyncManagerOptions {
  autoSyncEnabled: boolean;
  syncInterval: number;
  maxRetries: number;
  batchSize: number;
  conflictResolutionStrategy: 'local' | 'remote' | 'merge' | 'manual';
}

export class SyncManager {
  private apiClient: IInkGatewayClient;
  private logger: ILogger;
  private eventManager: IEventManager;
  private cacheManager: ICacheManager;
  private offlineManager: IOfflineManager;
  private options: SyncManagerOptions;
  
  private syncState!: SyncState;
  private syncTimer: NodeJS.Timeout | null = null;
  private isSyncing = false;
  
  constructor(
    apiClient: IInkGatewayClient,
    logger: ILogger,
    eventManager: IEventManager,
    cacheManager: ICacheManager,
    offlineManager: IOfflineManager,
    options: SyncManagerOptions
  ) {
    this.apiClient = apiClient;
    this.logger = logger;
    this.eventManager = eventManager;
    this.cacheManager = cacheManager;
    this.offlineManager = offlineManager;
    this.options = options;
    
    this.initializeSyncState();
    this.setupEventListeners();
    
    if (options.autoSyncEnabled) {
      this.startAutoSync();
    }
  }

  /**
   * Initialize sync state
   */
  private initializeSyncState(): void {
    this.syncState = {
      lastSyncTime: new Date(0), // Start with epoch
      pendingChanges: [],
      conflictResolution: {
        strategy: this.options.conflictResolutionStrategy,
        conflicts: []
      },
      syncStatus: 'idle'
    };
    
    // Try to restore sync state from cache
    const cachedState = this.cacheManager.get<SyncState>('sync_state');
    if (cachedState) {
      this.syncState = { ...this.syncState, ...cachedState };
      this.logger.debug('Restored sync state from cache', {
        pendingChanges: this.syncState.pendingChanges.length,
        lastSyncTime: this.syncState.lastSyncTime
      });
    }
  }

  /**
   * Setup event listeners
   */
  private setupEventListeners(): void {
    // Listen for content changes
    this.eventManager.on('contentChanged', (chunk: UnifiedChunk) => {
      this.queueChange('update', chunk);
    });
    
    // Listen for content creation
    this.eventManager.on('contentCreated', (chunk: UnifiedChunk) => {
      this.queueChange('create', chunk);
    });
    
    // Listen for content deletion
    this.eventManager.on('contentDeleted', (chunkId: string) => {
      this.queueChange('delete', { chunkId } as UnifiedChunk);
    });
    
    // Listen for online/offline status changes
    this.eventManager.on('onlineStatusChanged', (isOnline: boolean) => {
      if (isOnline && this.syncState.pendingChanges.length > 0) {
        this.performSync();
      }
    });
    
    // Listen for manual sync requests
    this.eventManager.on('manualSyncRequested', () => {
      this.performSync();
    });
  }

  /**
   * Queue a change for synchronization
   */
  public queueChange(type: 'create' | 'update' | 'delete', chunk: UnifiedChunk): void {
    const change: PendingChange = {
      id: `${type}_${chunk.chunkId}_${Date.now()}`,
      type,
      chunk,
      timestamp: new Date(),
      retryCount: 0
    };
    
    // Remove any existing changes for the same chunk
    this.syncState.pendingChanges = this.syncState.pendingChanges.filter(
      c => c.chunk.chunkId !== chunk.chunkId
    );
    
    // Add new change
    this.syncState.pendingChanges.push(change);
    
    this.logger.debug(`Queued ${type} change for chunk: ${chunk.chunkId}`);
    this.saveSyncState();
    
    // Emit event
    this.eventManager.emit('syncStateChanged', this.syncState);
    
    // Trigger immediate sync if online and auto-sync is enabled
    if (this.options.autoSyncEnabled && this.offlineManager.isOnline()) {
      this.scheduleSync();
    }
  }

  /**
   * Perform synchronization
   */
  public async performSync(): Promise<SyncResult> {
    if (this.isSyncing) {
      this.logger.debug('Sync already in progress, skipping');
      return {
        success: false,
        syncedChunks: 0,
        errors: [{ chunkId: 'sync_in_progress', error: 'Sync already in progress', recoverable: true }],
        conflicts: [],
        duration: 0
      };
    }
    
    if (!this.offlineManager.isOnline()) {
      this.logger.debug('Offline, queueing changes for later sync');
      this.syncState.syncStatus = 'offline';
      this.eventManager.emit('syncStateChanged', this.syncState);
      return {
        success: false,
        syncedChunks: 0,
        errors: [{ chunkId: 'offline', error: 'Device is offline', recoverable: true }],
        conflicts: [],
        duration: 0
      };
    }
    
    if (this.syncState.pendingChanges.length === 0) {
      this.logger.debug('No pending changes to sync');
      return {
        success: true,
        syncedChunks: 0,
        errors: [],
        conflicts: [],
        duration: 0
      };
    }
    
    this.isSyncing = true;
    this.syncState.syncStatus = 'syncing';
    this.eventManager.emit('syncStateChanged', this.syncState);
    
    const startTime = Date.now();
    let syncedChunks = 0;
    const errors: any[] = [];
    const conflicts: SyncConflict[] = [];
    
    try {
      this.logger.info(`Starting sync of ${this.syncState.pendingChanges.length} pending changes`);
      
      // Check API health
      const isHealthy = await this.apiClient.healthCheck();
      if (!isHealthy) {
        throw new PluginError(
          ErrorType.API_ERROR,
          'GATEWAY_UNAVAILABLE',
          'Ink Gateway is not available',
          true
        );
      }
      
      // Process changes in batches
      const batches = this.createBatches(this.syncState.pendingChanges, this.options.batchSize);
      
      for (const batch of batches) {
        try {
          const batchResult = await this.processBatch(batch);
          syncedChunks += batchResult.syncedChunks;
          errors.push(...batchResult.errors);
          conflicts.push(...batchResult.conflicts);
          
          // Remove successfully synced changes
          this.syncState.pendingChanges = this.syncState.pendingChanges.filter(
            change => !batchResult.syncedChangeIds.includes(change.id)
          );
          
        } catch (error) {
          this.logger.error('Batch processing failed:', error as Error);
          
          // Increment retry count for failed changes
          const changesToRemove: string[] = [];
          batch.forEach(change => {
            // Find the change in pendingChanges and increment retry count
            const pendingChange = this.syncState.pendingChanges.find(c => c.id === change.id);
            if (pendingChange) {
              pendingChange.retryCount++;
              if (pendingChange.retryCount >= this.options.maxRetries) {
                errors.push({
                  chunkId: change.chunk.chunkId,
                  error: `Max retries exceeded: ${(error as Error).message}`,
                  recoverable: false
                });
                
                changesToRemove.push(change.id);
              }
            }
          });
          
          // Remove changes that exceeded max retries
          this.syncState.pendingChanges = this.syncState.pendingChanges.filter(
            c => !changesToRemove.includes(c.id)
          );
        }
      }
      
      // Update sync state
      this.syncState.lastSyncTime = new Date();
      this.syncState.syncStatus = 'idle';
      
      const duration = Date.now() - startTime;
      const success = errors.length === 0;
      
      const result: SyncResult = {
        success,
        syncedChunks,
        errors,
        conflicts,
        duration
      };
      
      this.logger.info('Sync completed', {
        success,
        syncedChunks,
        totalErrors: errors.length,
        totalConflicts: conflicts.length,
        duration,
        remainingPendingChanges: this.syncState.pendingChanges.length
      });
      
      // Save sync state and emit events
      this.saveSyncState();
      this.eventManager.emit('syncCompleted', result);
      this.eventManager.emit('syncStateChanged', this.syncState);
      
      return result;
      
    } catch (error) {
      this.syncState.syncStatus = 'error';
      this.eventManager.emit('syncStateChanged', this.syncState);
      
      const duration = Date.now() - startTime;
      this.logger.error('Sync failed:', error as Error);
      
      return {
        success: false,
        syncedChunks,
        errors: [{
          chunkId: 'sync_error',
          error: (error as Error).message,
          recoverable: true
        }],
        conflicts,
        duration
      };
    } finally {
      this.isSyncing = false;
    }
  }

  /**
   * Process a batch of changes
   */
  private async processBatch(batch: PendingChange[]): Promise<{
    syncedChunks: number;
    errors: any[];
    conflicts: SyncConflict[];
    syncedChangeIds: string[];
  }> {
    const syncedChangeIds: string[] = [];
    const errors: any[] = [];
    const conflicts: SyncConflict[] = [];
    let syncedChunks = 0;
    
    // Group changes by type
    const creates = batch.filter(c => c.type === 'create');
    const updates = batch.filter(c => c.type === 'update');
    const deletes = batch.filter(c => c.type === 'delete');
    
    // Process creates
    if (creates.length > 0) {
      try {
        const chunks = creates.map(c => c.chunk);
        const createdChunks = await this.apiClient.batchCreateChunks(chunks);
        
        syncedChunks += createdChunks.length;
        syncedChangeIds.push(...creates.map(c => c.id));
        
        // Update cache
        createdChunks.forEach(chunk => {
          this.cacheManager.set(`chunk_${chunk.chunkId}`, chunk, 600000);
        });
        
      } catch (error) {
        this.logger.error('Failed to create chunks:', error as Error);
        
        // Handle retry logic for failed creates
        const changesToRemove: string[] = [];
        creates.forEach(change => {
          // Find the change in pendingChanges and increment retry count
          const pendingChange = this.syncState.pendingChanges.find(c => c.id === change.id);
          if (pendingChange) {
            pendingChange.retryCount++;
            if (pendingChange.retryCount >= this.options.maxRetries) {
              errors.push({
                chunkId: change.chunk.chunkId,
                error: `Max retries exceeded: ${(error as Error).message}`,
                recoverable: false
              });
              changesToRemove.push(change.id);
            } else {
              errors.push({
                chunkId: change.chunk.chunkId,
                error: (error as Error).message,
                recoverable: true
              });
            }
          }
        });
        
        // Remove changes that exceeded max retries
        this.syncState.pendingChanges = this.syncState.pendingChanges.filter(
          c => !changesToRemove.includes(c.id)
        );
      }
    }
    
    // Process updates
    for (const update of updates) {
      try {
        // Check for conflicts by getting the current version
        const currentChunk = await this.apiClient.getChunk(update.chunk.chunkId);
        
        if (this.hasConflict(update.chunk, currentChunk)) {
          const conflict: SyncConflict = {
            chunkId: update.chunk.chunkId,
            localVersion: update.chunk,
            remoteVersion: currentChunk,
            conflictType: 'content'
          };
          
          conflicts.push(conflict);
          this.syncState.conflictResolution.conflicts.push(conflict);
          
          // Handle conflict based on strategy
          const resolved = await this.resolveConflict(conflict);
          if (resolved) {
            syncedChangeIds.push(update.id);
            syncedChunks++;
          }
        } else {
          // No conflict, proceed with update
          const updatedChunk = await this.apiClient.updateChunk(
            update.chunk.chunkId, 
            update.chunk
          );
          
          syncedChunks++;
          syncedChangeIds.push(update.id);
          
          // Update cache
          this.cacheManager.set(`chunk_${updatedChunk.chunkId}`, updatedChunk, 600000);
        }
        
      } catch (error) {
        this.logger.error(`Failed to update chunk ${update.chunk.chunkId}:`, error as Error);
        errors.push({
          chunkId: update.chunk.chunkId,
          error: (error as Error).message,
          recoverable: true
        });
      }
    }
    
    // Process deletes
    for (const deleteChange of deletes) {
      try {
        await this.apiClient.deleteChunk(deleteChange.chunk.chunkId);
        syncedChunks++;
        syncedChangeIds.push(deleteChange.id);
        
        // Remove from cache
        this.cacheManager.delete(`chunk_${deleteChange.chunk.chunkId}`);
        
      } catch (error) {
        this.logger.error(`Failed to delete chunk ${deleteChange.chunk.chunkId}:`, error as Error);
        errors.push({
          chunkId: deleteChange.chunk.chunkId,
          error: (error as Error).message,
          recoverable: true
        });
      }
    }
    
    return { syncedChunks, errors, conflicts, syncedChangeIds };
  }

  /**
   * Check if there's a conflict between local and remote versions
   */
  private hasConflict(localChunk: UnifiedChunk, remoteChunk: UnifiedChunk): boolean {
    // Simple conflict detection based on lastUpdated timestamp
    return localChunk.lastUpdated < remoteChunk.lastUpdated &&
           localChunk.contents !== remoteChunk.contents;
  }

  /**
   * Resolve a sync conflict
   */
  private async resolveConflict(conflict: SyncConflict): Promise<boolean> {
    switch (this.syncState.conflictResolution.strategy) {
      case 'local':
        // Use local version
        try {
          await this.apiClient.updateChunk(conflict.chunkId, conflict.localVersion);
          this.logger.info(`Resolved conflict using local version: ${conflict.chunkId}`);
          return true;
        } catch (error) {
          this.logger.error(`Failed to resolve conflict with local version: ${conflict.chunkId}`, error as Error);
          return false;
        }
        
      case 'remote':
        // Use remote version (no action needed, just mark as resolved)
        this.logger.info(`Resolved conflict using remote version: ${conflict.chunkId}`);
        return true;
        
      case 'merge':
        // Attempt to merge (simplified merge strategy)
        try {
          const mergedChunk = this.mergeChunks(conflict.localVersion, conflict.remoteVersion);
          await this.apiClient.updateChunk(conflict.chunkId, mergedChunk);
          this.logger.info(`Resolved conflict using merge strategy: ${conflict.chunkId}`);
          return true;
        } catch (error) {
          this.logger.error(`Failed to resolve conflict with merge strategy: ${conflict.chunkId}`, error as Error);
          return false;
        }
        
      case 'manual':
        // Emit event for manual resolution
        this.eventManager.emit('conflictRequiresManualResolution', conflict);
        this.logger.info(`Conflict requires manual resolution: ${conflict.chunkId}`);
        return false;
        
      default:
        this.logger.warn(`Unknown conflict resolution strategy: ${this.syncState.conflictResolution.strategy}`);
        return false;
    }
  }

  /**
   * Merge two chunks (simplified merge strategy)
   */
  private mergeChunks(localChunk: UnifiedChunk, remoteChunk: UnifiedChunk): UnifiedChunk {
    // Simple merge: combine content and use latest metadata
    const mergedContent = `${localChunk.contents}\n\n---\n\n${remoteChunk.contents}`;
    
    return {
      ...remoteChunk, // Use remote as base
      contents: mergedContent,
      tags: [...new Set([...localChunk.tags, ...remoteChunk.tags])], // Merge tags
      lastUpdated: new Date()
    };
  }

  /**
   * Create batches from changes
   */
  private createBatches<T>(items: T[], batchSize: number): T[][] {
    const batches: T[][] = [];
    for (let i = 0; i < items.length; i += batchSize) {
      batches.push(items.slice(i, i + batchSize));
    }
    return batches;
  }

  /**
   * Schedule a sync operation
   */
  private scheduleSync(): void {
    if (this.syncTimer) {
      clearTimeout(this.syncTimer);
      this.syncTimer = null;
    }
    
    this.syncTimer = setTimeout(() => {
      this.performSync();
    }, 1000); // 1 second delay to batch multiple changes
  }

  /**
   * Start auto-sync timer
   */
  public startAutoSync(): void {
    this.stopAutoSync(); // Ensure any existing timer is cleared
    
    this.syncTimer = setInterval(() => {
      if (this.syncState.pendingChanges.length > 0 && this.offlineManager.isOnline()) {
        this.performSync();
      }
    }, this.options.syncInterval);
    
    this.logger.info(`Auto-sync started with interval: ${this.options.syncInterval}ms`);
  }

  /**
   * Stop auto-sync timer
   */
  public stopAutoSync(): void {
    if (this.syncTimer) {
      clearInterval(this.syncTimer);
      this.syncTimer = null;
    }
    
    this.logger.info('Auto-sync stopped');
  }

  /**
   * Save sync state to cache
   */
  private saveSyncState(): void {
    this.cacheManager.set('sync_state', this.syncState, 86400000); // 24 hours TTL
  }

  /**
   * Get current sync state
   */
  public getSyncState(): SyncState {
    return { ...this.syncState };
  }

  /**
   * Clear all pending changes
   */
  public clearPendingChanges(): void {
    this.syncState.pendingChanges = [];
    this.saveSyncState();
    this.eventManager.emit('syncStateChanged', this.syncState);
    this.logger.info('Cleared all pending changes');
  }

  /**
   * Get pending changes count
   */
  public getPendingChangesCount(): number {
    return this.syncState.pendingChanges.length;
  }

  /**
   * Force sync now
   */
  public async forceSyncNow(): Promise<SyncResult> {
    this.logger.info('Force sync requested');
    return await this.performSync();
  }

  /**
   * Update sync options
   */
  public updateOptions(options: Partial<SyncManagerOptions>): void {
    this.options = { ...this.options, ...options };
    
    if (options.autoSyncEnabled !== undefined) {
      if (options.autoSyncEnabled) {
        this.startAutoSync();
      } else {
        this.stopAutoSync();
      }
    }
    
    if (options.conflictResolutionStrategy) {
      this.syncState.conflictResolution.strategy = options.conflictResolutionStrategy;
      this.saveSyncState();
    }
    
    this.logger.info('Sync options updated', options);
  }

  /**
   * Cleanup resources
   */
  public cleanup(): void {
    this.stopAutoSync();
    
    // Clear any scheduled sync timer
    if (this.syncTimer) {
      clearTimeout(this.syncTimer);
      this.syncTimer = null;
    }
    
    this.saveSyncState();
    this.logger.info('SyncManager cleanup completed');
  }
}