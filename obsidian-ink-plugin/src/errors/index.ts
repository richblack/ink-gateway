/**
 * Error handling system exports
 * Provides centralized access to all error handling components
 */

// Core error handling
export {
  ErrorHandler,
  ErrorSeverity,
  ErrorContext,
  ErrorLogEntry,
  ErrorHandlingStrategy,
  ErrorStats,
  globalErrorHandler,
  withErrorHandling,
  handleErrors
} from './ErrorHandler';

// Retry management
export {
  RetryManager,
  RetryOptions,
  CircuitBreakerOptions,
  CircuitBreakerState,
  RetryStats,
  globalRetryManager,
  withRetry,
  retry
} from './RetryManager';

// Error display
export {
  ErrorDisplayManager,
  ErrorDisplayOptions,
  ErrorDisplayConfig,
  RecoveryAction,
  ERROR_DISPLAY_STYLES,
  initializeErrorDisplay,
  getErrorDisplayManager
} from './ErrorDisplay';

// Debug logging
export {
  DebugLogger,
  LogLevel,
  LogEntry,
  DebugSession,
  PerformanceMetrics,
  DebugConfig,
  globalDebugLogger,
  logger,
  perf,
  timed,
  logged
} from './DebugLogger';

// Re-export types from main types file for convenience
export {
  ErrorType,
  PluginError
} from '../types';