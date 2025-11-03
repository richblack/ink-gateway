/**
 * Sync coordinator for integrating offline operations with the Ink-Gateway API
 * Handles the actual synchronization of operations with the backend
 */

import { UnifiedChunk, SyncConflict, OfflineOperation } from '../types';
import { IInkGatewayClient } from '../interfaces';
import { OperationResult, SyncBatchResult } from './OfflineManager';
import { globalErrorHandler } from '../errors/ErrorHandler';
import { globalRetryManager } from '../errors/RetryManager';
import { logger, perf } from '../errors/DebugLogger';

// Sync operation types
export type SyncOperationType = 'create' | 'update' | 'delete';

// Sync operation data
export interface SyncOperationData {
  chunk?: UnifiedChunk;
  chunkId?: string;
  metadata?: Record<string, any>;
}

// Conflict detection result
export interface ConflictDetectionResult {
  hasConflict: boolean;
  localVersion?: UnifiedChunk;
  remoteVersion?: UnifiedChunk;
  conflictType?: 'content' | 'metadata' | 'hierarchy';
}

// Sync coordinator configuration
export interface SyncCoordinatorConfig {
  enableConflictDetection: boolean;
  conflictDetectionStrategy: 'timestamp' | 'checksum' | 'version';
  batchTimeout: number;
  maxConcurrentOperations: number;
  enableOptimisticLocking: boolean;
}

// Default configuration
const DEFAULT_SYNC_CONFIG: SyncCoordinatorConfig = {
  enableConflictDetection: true,
  conflictDetectionStrategy: 'timestamp',
  batchTimeout: 30000, // 30 seconds
  maxConcurrentOperations: 5,
  enableOptimisticLocking: true
};

export class SyncCoordinator {
  private apiClient: IInkGatewayClient;
  private config: SyncCoordinatorConfig;
  private activeSyncs: Map<string, Promise<OperationResult>> = new Map();

  constructor(apiClient: IInkGatewayClient, config: Partial<SyncCoordinatorConfig> = {}) {
    this.apiClient = apiClient;
    this.config = { ...DEFAULT_SYNC_CONFIG, ...config };
  }

  /**
   * Sync a batch of operations
   */
  async syncOperations(operations: OfflineOperation[]): Promise<OperationResult[]> {
    if (operations.length === 0) {
      return [];
    }

    perf.start('syncOperations');
    logger.info('SyncCoordinator', 'syncOperations', `Starting sync of ${operations.length} operations`);

    try {
      // Group operations by type for optimal processing
      const groupedOperations = this.groupOperationsByType(operations);
      const results: OperationResult[] = [];

      // Process each group
      for (const [type, ops] of groupedOperations.entries()) {
        logger.debug('SyncCoordinator', 'syncOperations', `Processing ${ops.length} ${type} operations`);
        
        const groupResults = await this.processOperationGroup(type, ops);
        results.push(...groupResults);
      }

      logger.info('SyncCoordinator', 'syncOperations', 'Sync completed', {
        total: operations.length,
        successful: results.filter(r => r.success).length,
        failed: results.filter(r => !r.success).length
      });

      return results;

    } catch (error) {
      logger.error('SyncCoordinator', 'syncOperations', 'Sync failed', error);
      
      // Return failed results for all operations
      return operations.map(op => ({
        success: false,
        operation: op,
        error: error instanceof Error ? error : new Error(String(error)),
        retryCount: 0
      }));
    } finally {
      perf.end('syncOperations');
    }
  }

  /**
   * Sync a single operation
   */
  async syncSingleOperation(operation: OfflineOperation): Promise<OperationResult> {
    const operationKey = `${operation.type}_${operation.id}`;
    
    // Check if this operation is already being synced
    if (this.activeSyncs.has(operationKey)) {
      logger.debug('SyncCoordinator', 'syncSingleOperation', 'Operation already in progress', {
        operationId: operation.id
      });
      return await this.activeSyncs.get(operationKey)!;
    }

    // Start sync
    const syncPromise = this.executeSingleOperation(operation);
    this.activeSyncs.set(operationKey, syncPromise);

    try {
      const result = await syncPromise;
      return result;
    } finally {
      this.activeSyncs.delete(operationKey);
    }
  }

  /**
   * Check for conflicts before syncing
   */
  async detectConflicts(operations: OfflineOperation[]): Promise<Map<string, ConflictDetectionResult>> {
    if (!this.config.enableConflictDetection) {
      return new Map();
    }

    const conflicts = new Map<string, ConflictDetectionResult>();

    for (const operation of operations) {
      if (operation.type === 'update' && operation.data?.chunkId) {
        try {
          const conflictResult = await this.checkForConflict(operation);
          if (conflictResult.hasConflict) {
            conflicts.set(operation.id, conflictResult);
          }
        } catch (error) {
          logger.warn('SyncCoordinator', 'detectConflicts', 'Conflict detection failed', error, {
            operationId: operation.id
          });
        }
      }
    }

    return conflicts;
  }

  /**
   * Update configuration
   */
  updateConfig(config: Partial<SyncCoordinatorConfig>): void {
    this.config = { ...this.config, ...config };
    logger.debug('SyncCoordinator', 'updateConfig', 'Configuration updated', config);
  }

  /**
   * Get current configuration
   */
  getConfig(): SyncCoordinatorConfig {
    return { ...this.config };
  }

  // Private methods

  private groupOperationsByType(operations: OfflineOperation[]): Map<SyncOperationType, OfflineOperation[]> {
    const groups = new Map<SyncOperationType, OfflineOperation[]>();

    for (const operation of operations) {
      const type = operation.type as SyncOperationType;
      if (!groups.has(type)) {
        groups.set(type, []);
      }
      groups.get(type)!.push(operation);
    }

    return groups;
  }

  private async processOperationGroup(
    type: SyncOperationType,
    operations: OfflineOperation[]
  ): Promise<OperationResult[]> {
    // Use different strategies based on operation type
    switch (type) {
      case 'create':
        return await this.processBatchCreate(operations);
      case 'update':
        return await this.processBatchUpdate(operations);
      case 'delete':
        return await this.processBatchDelete(operations);
      default:
        throw new Error(`Unsupported operation type: ${type}`);
    }
  }

  private async processBatchCreate(operations: OfflineOperation[]): Promise<OperationResult[]> {
    // For create operations, we can use batch API if available
    const chunks = operations
      .map(op => op.data as UnifiedChunk)
      .filter(chunk => chunk != null);

    if (chunks.length === 0) {
      return operations.map(op => ({
        success: false,
        operation: op,
        error: new Error('Invalid chunk data for create operation'),
        retryCount: 0
      }));
    }

    try {
      // Use batch create if available
      const createdChunks = await globalRetryManager.execute(
        () => this.apiClient.batchCreateChunks(chunks),
        'batchCreateChunks'
      );

      // Map results back to operations
      return operations.map((op, index) => ({
        success: true,
        operation: op,
        retryCount: 0
      }));

    } catch (error) {
      logger.error('SyncCoordinator', 'processBatchCreate', 'Batch create failed', error);
      
      // Fall back to individual creates
      return await this.processIndividualOperations(operations);
    }
  }

  private async processBatchUpdate(operations: OfflineOperation[]): Promise<OperationResult[]> {
    // Check for conflicts first
    const conflicts = await this.detectConflicts(operations);
    const results: OperationResult[] = [];

    for (const operation of operations) {
      const conflict = conflicts.get(operation.id);
      
      if (conflict?.hasConflict) {
        // Handle conflict
        const syncConflict: SyncConflict = {
          chunkId: operation.data.chunkId,
          localVersion: conflict.localVersion!,
          remoteVersion: conflict.remoteVersion!,
          conflictType: conflict.conflictType!
        };

        results.push({
          success: false,
          operation,
          conflict: syncConflict,
          retryCount: 0
        });
      } else {
        // No conflict, proceed with update
        const result = await this.executeSingleOperation(operation);
        results.push(result);
      }
    }

    return results;
  }

  private async processBatchDelete(operations: OfflineOperation[]): Promise<OperationResult[]> {
    // Process deletes individually to handle errors gracefully
    return await this.processIndividualOperations(operations);
  }

  private async processIndividualOperations(operations: OfflineOperation[]): Promise<OperationResult[]> {
    const results: OperationResult[] = [];
    const semaphore = new Semaphore(this.config.maxConcurrentOperations);

    const promises = operations.map(async (operation) => {
      await semaphore.acquire();
      try {
        return await this.executeSingleOperation(operation);
      } finally {
        semaphore.release();
      }
    });

    const operationResults = await Promise.all(promises);
    results.push(...operationResults);

    return results;
  }

  private async executeSingleOperation(operation: OfflineOperation): Promise<OperationResult> {
    const operationName = `${operation.type}_${operation.id}`;
    perf.start(operationName);

    try {
      let success = false;
      let error: Error | undefined;

      switch (operation.type) {
        case 'create':
          await this.executeCreate(operation);
          success = true;
          break;

        case 'update':
          await this.executeUpdate(operation);
          success = true;
          break;

        case 'delete':
          await this.executeDelete(operation);
          success = true;
          break;

        default:
          throw new Error(`Unsupported operation type: ${operation.type}`);
      }

      return {
        success,
        operation,
        error,
        retryCount: 0
      };

    } catch (err) {
      const error = err instanceof Error ? err : new Error(String(err));
      
      logger.error('SyncCoordinator', 'executeSingleOperation', 'Operation failed', error, {
        operationId: operation.id,
        operationType: operation.type
      });

      return {
        success: false,
        operation,
        error,
        retryCount: 0
      };
    } finally {
      perf.end(operationName);
    }
  }

  private async executeCreate(operation: OfflineOperation): Promise<void> {
    const chunk = operation.data as UnifiedChunk;
    if (!chunk) {
      throw new Error('Invalid chunk data for create operation');
    }

    await globalRetryManager.execute(
      () => this.apiClient.createChunk(chunk),
      `createChunk_${operation.id}`
    );
  }

  private async executeUpdate(operation: OfflineOperation): Promise<void> {
    const chunk = operation.data as UnifiedChunk;
    if (!chunk || !chunk.chunkId) {
      throw new Error('Invalid chunk data for update operation');
    }

    await globalRetryManager.execute(
      () => this.apiClient.updateChunk(chunk.chunkId, chunk),
      `updateChunk_${operation.id}`
    );
  }

  private async executeDelete(operation: OfflineOperation): Promise<void> {
    const chunkId = operation.data?.chunkId || operation.data;
    if (!chunkId) {
      throw new Error('Invalid chunk ID for delete operation');
    }

    await globalRetryManager.execute(
      () => this.apiClient.deleteChunk(chunkId),
      `deleteChunk_${operation.id}`
    );
  }

  private async checkForConflict(operation: OfflineOperation): Promise<ConflictDetectionResult> {
    const chunk = operation.data as UnifiedChunk;
    if (!chunk?.chunkId) {
      return { hasConflict: false };
    }

    try {
      // Fetch current version from server
      const remoteChunk = await this.apiClient.getChunk(chunk.chunkId);
      
      // Compare versions based on strategy
      const hasConflict = await this.compareVersions(chunk, remoteChunk);
      
      if (hasConflict) {
        return {
          hasConflict: true,
          localVersion: chunk,
          remoteVersion: remoteChunk,
          conflictType: this.determineConflictType(chunk, remoteChunk)
        };
      }

      return { hasConflict: false };

    } catch (error) {
      // If we can't fetch the remote version, assume no conflict
      logger.warn('SyncCoordinator', 'checkForConflict', 'Could not fetch remote version', error);
      return { hasConflict: false };
    }
  }

  private async compareVersions(localChunk: UnifiedChunk, remoteChunk: UnifiedChunk): Promise<boolean> {
    switch (this.config.conflictDetectionStrategy) {
      case 'timestamp':
        return this.compareByTimestamp(localChunk, remoteChunk);
      
      case 'checksum':
        return await this.compareByChecksum(localChunk, remoteChunk);
      
      case 'version':
        return this.compareByVersion(localChunk, remoteChunk);
      
      default:
        return false;
    }
  }

  private compareByTimestamp(localChunk: UnifiedChunk, remoteChunk: UnifiedChunk): boolean {
    const localTime = new Date(localChunk.lastUpdated).getTime();
    const remoteTime = new Date(remoteChunk.lastUpdated).getTime();
    
    // Conflict if remote was updated after local
    return remoteTime > localTime;
  }

  private async compareByChecksum(localChunk: UnifiedChunk, remoteChunk: UnifiedChunk): Promise<boolean> {
    const localChecksum = await this.calculateChecksum(localChunk.contents);
    const remoteChecksum = await this.calculateChecksum(remoteChunk.contents);
    
    return localChecksum !== remoteChecksum;
  }

  private compareByVersion(localChunk: UnifiedChunk, remoteChunk: UnifiedChunk): boolean {
    // Assuming version is stored in metadata
    const localVersion = localChunk.metadata?.version || 0;
    const remoteVersion = remoteChunk.metadata?.version || 0;
    
    return remoteVersion > localVersion;
  }

  private determineConflictType(localChunk: UnifiedChunk, remoteChunk: UnifiedChunk): 'content' | 'metadata' | 'hierarchy' {
    // Simple heuristic to determine conflict type
    if (localChunk.contents !== remoteChunk.contents) {
      return 'content';
    }
    
    if (localChunk.parent !== remoteChunk.parent) {
      return 'hierarchy';
    }
    
    return 'metadata';
  }

  private async calculateChecksum(content: string): Promise<string> {
    // Simple checksum calculation using built-in crypto API
    const encoder = new TextEncoder();
    const data = encoder.encode(content);
    const hashBuffer = await crypto.subtle.digest('SHA-256', data);
    const hashArray = Array.from(new Uint8Array(hashBuffer));
    return hashArray.map(b => b.toString(16).padStart(2, '0')).join('');
  }
}

// Simple semaphore implementation for concurrency control
class Semaphore {
  private permits: number;
  private waitQueue: Array<() => void> = [];

  constructor(permits: number) {
    this.permits = permits;
  }

  async acquire(): Promise<void> {
    if (this.permits > 0) {
      this.permits--;
      return;
    }

    return new Promise<void>(resolve => {
      this.waitQueue.push(resolve);
    });
  }

  release(): void {
    if (this.waitQueue.length > 0) {
      const resolve = this.waitQueue.shift()!;
      resolve();
    } else {
      this.permits++;
    }
  }
}