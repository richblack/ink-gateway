/**
 * Tests for OfflineManager class
 */

import { OfflineManager, OfflineState, SyncStrategy, ConflictResolutionStrategy } from '../OfflineManager';
import { UnifiedChunk, SyncConflict, OfflineOperation } from '../../types';

// Mock localStorage
const localStorageMock = {
  getItem: jest.fn(),
  setItem: jest.fn(),
  removeItem: jest.fn(),
  clear: jest.fn(),
};
Object.defineProperty(window, 'localStorage', { value: localStorageMock });

// Mock navigator.onLine
Object.defineProperty(navigator, 'onLine', {
  writable: true,
  value: true,
});

describe('OfflineManager', () => {
  let offlineManager: OfflineManager;
  let mockSyncFunction: jest.Mock;

  beforeEach(() => {
    jest.clearAllMocks();
    localStorageMock.getItem.mockReturnValue(null);
    
    mockSyncFunction = jest.fn();
    
    offlineManager = new OfflineManager({
      enabled: true,
      maxQueueSize: 10,
      syncStrategy: SyncStrategy.IMMEDIATE,
      persistQueue: false // Disable for tests
    });
    
    offlineManager.setSyncFunction(mockSyncFunction);
  });

  afterEach(() => {
    offlineManager.cleanup();
  });

  describe('initialization', () => {
    it('should initialize with correct default state', () => {
      expect(offlineManager.getState()).toBe(OfflineState.ONLINE);
      expect(offlineManager.isOnline()).toBe(true);
    });

    it('should load persisted queue on initialization', () => {
      const persistedData = {
        operations: [
          {
            id: 'test1',
            type: 'create',
            data: { test: 'data' },
            timestamp: '2023-01-01T00:00:00.000Z',
            priority: 1
          }
        ],
        failedOperations: [],
        timestamp: '2023-01-01T00:00:00.000Z'
      };

      localStorageMock.getItem.mockReturnValue(JSON.stringify(persistedData));

      const manager = new OfflineManager({
        enabled: true,
        persistQueue: true
      });

      const pendingOps = manager.getPendingOperations();
      expect(pendingOps).toHaveLength(1);
      expect(pendingOps[0].id).toBe('test1');

      manager.cleanup();
    });
  });

  describe('operation queuing', () => {
    it('should queue operations correctly', () => {
      const operation: OfflineOperation = {
        id: 'test1',
        type: 'create',
        data: { test: 'data' },
        timestamp: new Date(),
        priority: 1
      };

      offlineManager.queueOperation(operation);

      const pendingOps = offlineManager.getPendingOperations();
      expect(pendingOps).toHaveLength(1);
      expect(pendingOps[0]).toEqual(operation);
    });

    it('should respect queue size limit', () => {
      // Queue more operations than the limit
      for (let i = 0; i < 15; i++) {
        offlineManager.queueOperation({
          id: `test${i}`,
          type: 'create',
          data: { test: `data${i}` },
          timestamp: new Date(),
          priority: 1
        });
      }

      const pendingOps = offlineManager.getPendingOperations();
      expect(pendingOps.length).toBeLessThanOrEqual(10);
    });

    it('should not queue operations when disabled', () => {
      const disabledManager = new OfflineManager({ enabled: false });
      
      disabledManager.queueOperation({
        id: 'test1',
        type: 'create',
        data: { test: 'data' },
        timestamp: new Date(),
        priority: 1
      });

      expect(disabledManager.getPendingOperations()).toHaveLength(0);
      disabledManager.cleanup();
    });
  });

  describe('synchronization', () => {
    it('should sync operations successfully', async () => {
      const operation: OfflineOperation = {
        id: 'test1',
        type: 'create',
        data: { test: 'data' },
        timestamp: new Date(),
        priority: 1
      };

      mockSyncFunction.mockResolvedValue([{
        success: true,
        operation,
        retryCount: 0
      }]);

      offlineManager.queueOperation(operation);

      const result = await offlineManager.syncWhenOnline();

      expect(result.totalOperations).toBe(1);
      expect(result.successfulOperations).toBe(1);
      expect(result.failedOperations).toBe(0);
      expect(offlineManager.getPendingOperations()).toHaveLength(0);
    });

    it('should handle sync failures', async () => {
      const operation: OfflineOperation = {
        id: 'test1',
        type: 'create',
        data: { test: 'data' },
        timestamp: new Date(),
        priority: 1
      };

      mockSyncFunction.mockResolvedValue([{
        success: false,
        operation,
        error: new Error('Sync failed'),
        retryCount: 0
      }]);

      offlineManager.queueOperation(operation);

      const result = await offlineManager.syncWhenOnline();

      expect(result.totalOperations).toBe(1);
      expect(result.successfulOperations).toBe(0);
      expect(result.failedOperations).toBe(1);
      expect(offlineManager.getFailedOperations()).toHaveLength(1);
    });

    it('should not sync when offline', async () => {
      // Mock offline state
      Object.defineProperty(navigator, 'onLine', { value: false });
      
      const operation: OfflineOperation = {
        id: 'test1',
        type: 'create',
        data: { test: 'data' },
        timestamp: new Date(),
        priority: 1
      };

      offlineManager.queueOperation(operation);

      const result = await offlineManager.syncWhenOnline();

      expect(result.totalOperations).toBe(0);
      expect(mockSyncFunction).not.toHaveBeenCalled();
    });

    it('should process operations in priority order', async () => {
      const operations: OfflineOperation[] = [
        {
          id: 'low',
          type: 'create',
          data: { priority: 'low' },
          timestamp: new Date(Date.now() - 1000),
          priority: 1
        },
        {
          id: 'high',
          type: 'create',
          data: { priority: 'high' },
          timestamp: new Date(),
          priority: 10
        },
        {
          id: 'medium',
          type: 'create',
          data: { priority: 'medium' },
          timestamp: new Date(Date.now() - 500),
          priority: 5
        }
      ];

      mockSyncFunction.mockImplementation((ops) => {
        return Promise.resolve(ops.map(op => ({
          success: true,
          operation: op,
          retryCount: 0
        })));
      });

      operations.forEach(op => offlineManager.queueOperation(op));

      await offlineManager.syncWhenOnline();

      // Verify operations were processed in priority order (high to low)
      const callArgs = mockSyncFunction.mock.calls[0][0];
      expect(callArgs[0].id).toBe('high');
      expect(callArgs[1].id).toBe('medium');
      expect(callArgs[2].id).toBe('low');
    });
  });

  describe('conflict handling', () => {
    it('should handle conflicts correctly', async () => {
      const localChunk: UnifiedChunk = {
        chunkId: 'chunk1',
        contents: 'local content',
        isPage: false,
        isTag: false,
        isTemplate: false,
        isSlot: false,
        tags: [],
        metadata: {},
        createdTime: new Date(),
        lastUpdated: new Date(),
              documentId: 'test-doc-1',
      documentScope: 'file' as const,
      position: {
          fileName: 'test.md',
          lineStart: 1,
          lineEnd: 1,
          charStart: 0,
          charEnd: 10
        },
        filePath: 'test.md',
        obsidianMetadata: {
          properties: {},
          frontmatter: {},
          aliases: [],
          cssClasses: []
        },
      };

      const remoteChunk: UnifiedChunk = {
        ...localChunk,
        contents: 'remote content',
        lastUpdated: new Date(Date.now() + 1000)
      };

      const conflict: SyncConflict = {
        chunkId: 'chunk1',
        localVersion: localChunk,
        remoteVersion: remoteChunk,
        conflictType: 'content'
      };

      const operation: OfflineOperation = {
        id: 'test1',
        type: 'update',
        data: localChunk,
        timestamp: new Date(),
        priority: 1
      };

      mockSyncFunction.mockResolvedValue([{
        success: false,
        operation,
        conflict,
        retryCount: 0
      }]);

      offlineManager.queueOperation(operation);

      const result = await offlineManager.syncWhenOnline();

      expect(result.conflicts).toHaveLength(1);
      expect(offlineManager.getConflicts()).toHaveLength(1);
    });

    it('should resolve conflicts manually', () => {
      const localChunk: UnifiedChunk = {
        chunkId: 'chunk1',
        contents: 'local content',
        isPage: false,
        isTag: false,
        isTemplate: false,
        isSlot: false,
        tags: [],
        metadata: {},
        createdTime: new Date(),
        lastUpdated: new Date(),
              documentId: 'test-doc-1',
      documentScope: 'file' as const,
      position: {
          fileName: 'test.md',
          lineStart: 1,
          lineEnd: 1,
          charStart: 0,
          charEnd: 10
        },
        filePath: 'test.md',
        obsidianMetadata: {
          properties: {},
          frontmatter: {},
          aliases: [],
          cssClasses: []
        },
      };

      const conflict: SyncConflict = {
        chunkId: 'chunk1',
        localVersion: localChunk,
        remoteVersion: { ...localChunk, contents: 'remote content' },
        conflictType: 'content'
      };

      // Simulate conflict
      offlineManager['conflicts'] = [conflict];

      const resolution: UnifiedChunk = {
        ...localChunk,
        contents: 'resolved content'
      };

      offlineManager.resolveConflictManually('chunk1', resolution);

      expect(offlineManager.getConflicts()).toHaveLength(0);
      expect(offlineManager.getPendingOperations()).toHaveLength(1);
    });
  });

  describe('statistics', () => {
    it('should track sync statistics', async () => {
      const operation: OfflineOperation = {
        id: 'test1',
        type: 'create',
        data: { test: 'data' },
        timestamp: new Date(),
        priority: 1
      };

      mockSyncFunction.mockResolvedValue([{
        success: true,
        operation,
        retryCount: 0
      }]);

      offlineManager.queueOperation(operation);
      await offlineManager.syncWhenOnline();

      const stats = offlineManager.getStats();
      expect(stats.totalOperations).toBe(1);
      expect(stats.successfulOperations).toBe(1);
      expect(stats.failedOperations).toBe(0);
      expect(stats.lastSyncTime).toBeDefined();
    });
  });

  describe('event listeners', () => {
    it('should notify network status listeners', () => {
      const listener = jest.fn();
      offlineManager.addNetworkStatusListener(listener);

      // Simulate network change
      const event = new Event('offline');
      window.dispatchEvent(event);

      expect(listener).toHaveBeenCalledWith(false);
    });

    it('should notify sync progress listeners', async () => {
      const progressListener = jest.fn();
      offlineManager.addSyncProgressListener(progressListener);

      const operation: OfflineOperation = {
        id: 'test1',
        type: 'create',
        data: { test: 'data' },
        timestamp: new Date(),
        priority: 1
      };

      mockSyncFunction.mockResolvedValue([{
        success: true,
        operation,
        retryCount: 0
      }]);

      offlineManager.queueOperation(operation);
      await offlineManager.syncWhenOnline();

      expect(progressListener).toHaveBeenCalled();
    });

    it('should notify conflict listeners', async () => {
      const conflictListener = jest.fn();
      offlineManager.addConflictListener(conflictListener);

      const conflict: SyncConflict = {
        chunkId: 'chunk1',
        localVersion: {} as UnifiedChunk,
        remoteVersion: {} as UnifiedChunk,
        conflictType: 'content'
      };

      const operation: OfflineOperation = {
        id: 'test1',
        type: 'update',
        data: { chunkId: 'chunk1' },
        timestamp: new Date(),
        priority: 1
      };

      mockSyncFunction.mockResolvedValue([{
        success: false,
        operation,
        conflict,
        retryCount: 0
      }]);

      offlineManager.queueOperation(operation);
      await offlineManager.syncWhenOnline();

      expect(conflictListener).toHaveBeenCalledWith(conflict);
    });
  });

  describe('configuration', () => {
    it('should update configuration', () => {
      const newConfig = {
        maxQueueSize: 20,
        syncStrategy: SyncStrategy.BATCHED
      };

      offlineManager.updateConfig(newConfig);

      // Verify configuration was updated by testing behavior
      for (let i = 0; i < 15; i++) {
        offlineManager.queueOperation({
          id: `test${i}`,
          type: 'create',
          data: { test: `data${i}` },
          timestamp: new Date(),
          priority: 1
        });
      }

      expect(offlineManager.getPendingOperations().length).toBe(15);
    });
  });

  describe('queue management', () => {
    it('should clear queue', () => {
      offlineManager.queueOperation({
        id: 'test1',
        type: 'create',
        data: { test: 'data' },
        timestamp: new Date(),
        priority: 1
      });

      expect(offlineManager.getPendingOperations()).toHaveLength(1);

      offlineManager.clearQueue();

      expect(offlineManager.getPendingOperations()).toHaveLength(0);
    });

    it('should clear failed operations', async () => {
      const operation: OfflineOperation = {
        id: 'test1',
        type: 'create',
        data: { test: 'data' },
        timestamp: new Date(),
        priority: 1
      };

      mockSyncFunction.mockResolvedValue([{
        success: false,
        operation,
        error: new Error('Sync failed'),
        retryCount: 0
      }]);

      offlineManager.queueOperation(operation);
      await offlineManager.syncWhenOnline();

      expect(offlineManager.getFailedOperations()).toHaveLength(1);

      offlineManager.clearFailedOperations();

      expect(offlineManager.getFailedOperations()).toHaveLength(0);
    });
  });

  describe('content merging', () => {
    it('should merge conflicting content', async () => {
      const manager = new OfflineManager({
        enabled: true,
        conflictResolution: ConflictResolutionStrategy.MERGE
      });

      const localChunk: UnifiedChunk = {
        chunkId: 'chunk1',
        contents: 'local content',
        isPage: false,
        isTag: false,
        isTemplate: false,
        isSlot: false,
        tags: [],
        metadata: {},
        createdTime: new Date(),
        lastUpdated: new Date(),
              documentId: 'test-doc-1',
      documentScope: 'file' as const,
      position: {
          fileName: 'test.md',
          lineStart: 1,
          lineEnd: 1,
          charStart: 0,
          charEnd: 10
        },
        filePath: 'test.md',
        obsidianMetadata: {
          properties: {},
          frontmatter: {},
          aliases: [],
          cssClasses: []
        },
      };

      const conflict: SyncConflict = {
        chunkId: 'chunk1',
        localVersion: localChunk,
        remoteVersion: { ...localChunk, contents: 'remote content' },
        conflictType: 'content'
      };

      await manager['handleConflicts']([conflict]);

      // Should queue a merged version
      expect(manager.getPendingOperations()).toHaveLength(1);
      expect(manager.getConflicts()).toHaveLength(0);

      manager.cleanup();
    });
  });
});