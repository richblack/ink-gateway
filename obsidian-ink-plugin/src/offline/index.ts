/**
 * Offline support system exports
 * Provides centralized access to all offline management components
 */

// Offline manager
export {
  OfflineManager,
  OfflineState,
  SyncStrategy,
  ConflictResolutionStrategy,
  OfflineConfig,
  SyncStats,
  OperationResult,
  SyncBatchResult,
  NetworkStatusListener,
  SyncProgressListener,
  ConflictListener,
  globalOfflineManager
} from './OfflineManager';

// Sync coordinator
export {
  SyncCoordinator,
  SyncOperationType,
  SyncOperationData,
  ConflictDetectionResult,
  SyncCoordinatorConfig
} from './SyncCoordinator';

// Network monitor
export {
  NetworkMonitor,
  NetworkStatus,
  ConnectionQuality,
  NetworkEvent,
  NetworkMonitorConfig,
  NetworkTestResult,
  globalNetworkMonitor
} from './NetworkMonitor';

// Re-export types from main types file for convenience
export {
  OfflineOperation,
  SyncConflict,
  PendingChange
} from '../types';