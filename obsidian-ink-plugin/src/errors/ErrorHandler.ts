/**
 * Comprehensive error handling system for the Obsidian Ink Plugin
 * Provides centralized error management, user-friendly messaging, and debugging capabilities
 */

import { Notice } from 'obsidian';
import { ErrorType, PluginError } from '../types';

// Error severity levels
export enum ErrorSeverity {
  LOW = 'low',
  MEDIUM = 'medium',
  HIGH = 'high',
  CRITICAL = 'critical'
}

// Error context for debugging
export interface ErrorContext {
  operation: string;
  component: string;
  userId?: string;
  sessionId?: string;
  timestamp: Date;
  stackTrace?: string;
  additionalData?: Record<string, any>;
}

// Error log entry
export interface ErrorLogEntry {
  id: string;
  error: PluginError;
  context: ErrorContext;
  severity: ErrorSeverity;
  userMessage: string;
  resolved: boolean;
  resolvedAt?: Date;
  reportedToUser: boolean;
}

// Error handling strategy
export interface ErrorHandlingStrategy {
  showToUser: boolean;
  logError: boolean;
  retryable: boolean;
  maxRetries?: number;
  fallbackAction?: () => Promise<void>;
  customMessage?: string;
}

// Error statistics for monitoring
export interface ErrorStats {
  totalErrors: number;
  errorsByType: Record<ErrorType, number>;
  errorsBySeverity: Record<ErrorSeverity, number>;
  recentErrors: ErrorLogEntry[];
  averageResolutionTime: number;
}

// User-friendly error messages mapping
const ERROR_MESSAGES: Record<string, string> = {
  // Network errors
  'NETWORK_ERROR': 'Unable to connect to Ink-Gateway. Please check your internet connection.',
  'REQUEST_TIMEOUT': 'The request took too long to complete. Please try again.',
  'CONNECTION_REFUSED': 'Cannot connect to Ink-Gateway server. Please check if the server is running.',
  
  // API errors
  'HTTP_401': 'Authentication failed. Please check your API key in settings.',
  'HTTP_403': 'Access denied. Please verify your permissions.',
  'HTTP_404': 'The requested resource was not found.',
  'HTTP_429': 'Too many requests. Please wait a moment before trying again.',
  'HTTP_500': 'Server error occurred. Please try again later.',
  'HTTP_502': 'Gateway error. The server is temporarily unavailable.',
  'HTTP_503': 'Service unavailable. Please try again later.',
  
  // Parsing errors
  'INVALID_MARKDOWN': 'Unable to parse the markdown content. Please check the format.',
  'HIERARCHY_PARSE_ERROR': 'Error parsing content hierarchy. Some relationships may be missing.',
  'METADATA_PARSE_ERROR': 'Error parsing file metadata. Some properties may not be synced.',
  
  // Sync errors
  'SYNC_CONFLICT': 'Content conflict detected. Please resolve manually.',
  'SYNC_FAILED': 'Failed to sync content. Changes will be retried automatically.',
  'BATCH_SYNC_PARTIAL': 'Some content failed to sync. Retrying failed items.',
  
  // Validation errors
  'INVALID_CHUNK_DATA': 'Invalid content data. Please check the content format.',
  'INVALID_TEMPLATE': 'Template format is invalid. Please check the template structure.',
  'MISSING_REQUIRED_FIELD': 'Required field is missing. Please provide all required information.',
  
  // Default fallback
  'UNKNOWN_ERROR': 'An unexpected error occurred. Please try again or contact support.'
};

// Error severity mapping
const ERROR_SEVERITY_MAP: Record<ErrorType, ErrorSeverity> = {
  [ErrorType.NETWORK_ERROR]: ErrorSeverity.MEDIUM,
  [ErrorType.API_ERROR]: ErrorSeverity.MEDIUM,
  [ErrorType.PARSING_ERROR]: ErrorSeverity.LOW,
  [ErrorType.SYNC_ERROR]: ErrorSeverity.HIGH,
  [ErrorType.VALIDATION_ERROR]: ErrorSeverity.LOW
};

export class ErrorHandler {
  private errorLog: ErrorLogEntry[] = [];
  private maxLogSize: number = 1000;
  private debugMode: boolean = false;
  private sessionId: string;
  private errorStrategies: Map<string, ErrorHandlingStrategy> = new Map();

  constructor(debugMode: boolean = false) {
    this.debugMode = debugMode;
    this.sessionId = this.generateSessionId();
    this.initializeDefaultStrategies();
  }

  /**
   * Handle an error with appropriate strategy
   */
  async handleError(
    error: Error | PluginError,
    context: Partial<ErrorContext> = {}
  ): Promise<void> {
    const pluginError = this.normalizeError(error);
    const fullContext = this.buildErrorContext(context);
    const severity = this.determineSeverity(pluginError);
    const userMessage = this.getUserFriendlyMessage(pluginError);
    
    // Create error log entry
    const logEntry: ErrorLogEntry = {
      id: this.generateErrorId(),
      error: pluginError,
      context: fullContext,
      severity,
      userMessage,
      resolved: false,
      reportedToUser: false
    };

    // Log the error
    this.logError(logEntry);

    // Get handling strategy
    const strategy = this.getErrorStrategy(pluginError);

    // Show to user if required
    if (strategy.showToUser) {
      this.showErrorToUser(logEntry, strategy);
      logEntry.reportedToUser = true;
    }

    // Execute fallback action if available
    if (strategy.fallbackAction) {
      try {
        await strategy.fallbackAction();
      } catch (fallbackError) {
        console.error('[ErrorHandler] Fallback action failed:', fallbackError);
      }
    }

    // Debug logging
    if (this.debugMode) {
      this.debugLogError(logEntry);
    }
  }

  /**
   * Handle errors with retry logic
   */
  async handleErrorWithRetry<T>(
    operation: () => Promise<T>,
    context: Partial<ErrorContext> = {},
    maxRetries: number = 3,
    baseDelay: number = 1000
  ): Promise<T> {
    let lastError: Error | null = null;
    
    for (let attempt = 0; attempt <= maxRetries; attempt++) {
      try {
        return await operation();
      } catch (error) {
        lastError = error;
        
        const pluginError = this.normalizeError(error as Error);
        
        // Don't retry if error is not recoverable
        if (!pluginError.recoverable || attempt === maxRetries) {
          await this.handleError(pluginError, {
            ...context,
            operation: context.operation || 'retry_operation',
            additionalData: { attempt, maxRetries }
          });
          throw pluginError;
        }

        // Calculate delay with exponential backoff
        const delay = baseDelay * Math.pow(2, attempt);
        console.warn(`[ErrorHandler] Retrying operation in ${delay}ms (attempt ${attempt + 1}/${maxRetries})`);
        
        await this.sleep(delay);
      }
    }

    throw lastError;
  }

  /**
   * Register custom error handling strategy
   */
  registerErrorStrategy(errorCode: string, strategy: ErrorHandlingStrategy): void {
    this.errorStrategies.set(errorCode, strategy);
  }

  /**
   * Get error statistics for monitoring
   */
  getErrorStats(): ErrorStats {
    const now = Date.now();
    const recentErrors = this.errorLog.filter(
      entry => now - entry.context.timestamp.getTime() < 24 * 60 * 60 * 1000 // Last 24 hours
    );

    const errorsByType: Record<ErrorType, number> = {
      [ErrorType.NETWORK_ERROR]: 0,
      [ErrorType.API_ERROR]: 0,
      [ErrorType.PARSING_ERROR]: 0,
      [ErrorType.SYNC_ERROR]: 0,
      [ErrorType.VALIDATION_ERROR]: 0
    };

    const errorsBySeverity: Record<ErrorSeverity, number> = {
      [ErrorSeverity.LOW]: 0,
      [ErrorSeverity.MEDIUM]: 0,
      [ErrorSeverity.HIGH]: 0,
      [ErrorSeverity.CRITICAL]: 0
    };

    let totalResolutionTime = 0;
    let resolvedCount = 0;

    for (const entry of this.errorLog) {
      errorsByType[entry.error.type]++;
      errorsBySeverity[entry.severity]++;
      
      if (entry.resolved && entry.resolvedAt) {
        totalResolutionTime += entry.resolvedAt.getTime() - entry.context.timestamp.getTime();
        resolvedCount++;
      }
    }

    return {
      totalErrors: this.errorLog.length,
      errorsByType,
      errorsBySeverity,
      recentErrors: recentErrors.slice(-10), // Last 10 recent errors
      averageResolutionTime: resolvedCount > 0 ? totalResolutionTime / resolvedCount : 0
    };
  }

  /**
   * Mark error as resolved
   */
  resolveError(errorId: string): void {
    const entry = this.errorLog.find(e => e.id === errorId);
    if (entry) {
      entry.resolved = true;
      entry.resolvedAt = new Date();
    }
  }

  /**
   * Clear error log
   */
  clearErrorLog(): void {
    this.errorLog = [];
  }

  /**
   * Export error log for debugging
   */
  exportErrorLog(): string {
    return JSON.stringify(this.errorLog, null, 2);
  }

  /**
   * Get recent errors for display
   */
  getRecentErrors(limit: number = 10): ErrorLogEntry[] {
    return this.errorLog
      .sort((a, b) => b.context.timestamp.getTime() - a.context.timestamp.getTime())
      .slice(0, limit);
  }

  // Private methods

  private normalizeError(error: Error | PluginError): PluginError {
    if (error instanceof PluginError) {
      return error;
    }

    // Try to categorize common errors
    if (error.message.includes('fetch')) {
      return new PluginError(
        ErrorType.NETWORK_ERROR,
        'NETWORK_ERROR',
        { originalMessage: error.message },
        true
      );
    }

    if (error.message.includes('JSON')) {
      return new PluginError(
        ErrorType.PARSING_ERROR,
        'JSON_PARSE_ERROR',
        { originalMessage: error.message },
        false
      );
    }

    return new PluginError(
      ErrorType.NETWORK_ERROR,
      'UNKNOWN_ERROR',
      { originalMessage: error.message, stack: error.stack },
      false
    );
  }

  private buildErrorContext(partial: Partial<ErrorContext>): ErrorContext {
    return {
      operation: partial.operation || 'unknown',
      component: partial.component || 'unknown',
      userId: partial.userId,
      sessionId: this.sessionId,
      timestamp: new Date(),
      stackTrace: partial.stackTrace || new Error().stack,
      additionalData: partial.additionalData || {}
    };
  }

  private determineSeverity(error: PluginError): ErrorSeverity {
    // Check for critical conditions
    if (error.code === 'HTTP_401' || error.code === 'HTTP_403') {
      return ErrorSeverity.CRITICAL;
    }

    if (error.type === ErrorType.SYNC_ERROR && error.code === 'SYNC_CONFLICT') {
      return ErrorSeverity.HIGH;
    }

    return ERROR_SEVERITY_MAP[error.type] || ErrorSeverity.MEDIUM;
  }

  private getUserFriendlyMessage(error: PluginError): string {
    const customMessage = ERROR_MESSAGES[error.code];
    if (customMessage) {
      return customMessage;
    }

    const typeMessage = ERROR_MESSAGES[error.type];
    if (typeMessage) {
      return typeMessage;
    }

    return ERROR_MESSAGES['UNKNOWN_ERROR'];
  }

  private getErrorStrategy(error: PluginError): ErrorHandlingStrategy {
    const customStrategy = this.errorStrategies.get(error.code);
    if (customStrategy) {
      return customStrategy;
    }

    // Default strategies based on error type
    switch (error.type) {
      case ErrorType.NETWORK_ERROR:
        return {
          showToUser: true,
          logError: true,
          retryable: error.recoverable,
          maxRetries: 3
        };

      case ErrorType.API_ERROR:
        return {
          showToUser: true,
          logError: true,
          retryable: error.recoverable,
          maxRetries: error.code.startsWith('HTTP_5') ? 2 : 0
        };

      case ErrorType.SYNC_ERROR:
        return {
          showToUser: true,
          logError: true,
          retryable: true,
          maxRetries: 5
        };

      case ErrorType.PARSING_ERROR:
        return {
          showToUser: false,
          logError: true,
          retryable: false
        };

      case ErrorType.VALIDATION_ERROR:
        return {
          showToUser: true,
          logError: true,
          retryable: false
        };

      default:
        return {
          showToUser: true,
          logError: true,
          retryable: false
        };
    }
  }

  private logError(entry: ErrorLogEntry): void {
    this.errorLog.push(entry);

    // Maintain log size limit
    if (this.errorLog.length > this.maxLogSize) {
      this.errorLog.shift();
    }

    // Console logging based on severity
    const logMethod = this.getLogMethod(entry.severity);
    logMethod(`[ErrorHandler] ${entry.error.type}:${entry.error.code}`, {
      message: entry.userMessage,
      context: entry.context,
      details: entry.error.details
    });
  }

  private showErrorToUser(entry: ErrorLogEntry, strategy: ErrorHandlingStrategy): void {
    const message = strategy.customMessage || entry.userMessage;
    
    // Use different notice types based on severity
    switch (entry.severity) {
      case ErrorSeverity.CRITICAL:
        new Notice(`🚨 Critical Error: ${message}`, 10000);
        break;
      case ErrorSeverity.HIGH:
        new Notice(`⚠️ Error: ${message}`, 8000);
        break;
      case ErrorSeverity.MEDIUM:
        new Notice(`⚠️ ${message}`, 5000);
        break;
      case ErrorSeverity.LOW:
        if (this.debugMode) {
          new Notice(`ℹ️ ${message}`, 3000);
        }
        break;
    }
  }

  private debugLogError(entry: ErrorLogEntry): void {
    console.group(`[ErrorHandler Debug] ${entry.error.type}:${entry.error.code}`);
    console.log('Error:', entry.error);
    console.log('Context:', entry.context);
    console.log('Severity:', entry.severity);
    console.log('User Message:', entry.userMessage);
    console.log('Stack Trace:', entry.context.stackTrace);
    console.groupEnd();
  }

  private getLogMethod(severity: ErrorSeverity): typeof console.log {
    switch (severity) {
      case ErrorSeverity.CRITICAL:
      case ErrorSeverity.HIGH:
        return console.error;
      case ErrorSeverity.MEDIUM:
        return console.warn;
      case ErrorSeverity.LOW:
      default:
        return console.log;
    }
  }

  private initializeDefaultStrategies(): void {
    // Network timeout strategy
    this.registerErrorStrategy('REQUEST_TIMEOUT', {
      showToUser: true,
      logError: true,
      retryable: true,
      maxRetries: 2,
      customMessage: 'Request timed out. Retrying with longer timeout...'
    });

    // Authentication error strategy
    this.registerErrorStrategy('HTTP_401', {
      showToUser: true,
      logError: true,
      retryable: false,
      customMessage: 'Authentication failed. Please check your API key in plugin settings.',
      fallbackAction: async () => {
        // Could open settings modal here
        console.log('[ErrorHandler] Consider opening settings for API key configuration');
      }
    });

    // Rate limiting strategy
    this.registerErrorStrategy('HTTP_429', {
      showToUser: true,
      logError: true,
      retryable: true,
      maxRetries: 3,
      customMessage: 'Rate limit exceeded. Waiting before retry...'
    });
  }

  private generateSessionId(): string {
    return `session_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
  }

  private generateErrorId(): string {
    return `error_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
  }

  private sleep(ms: number): Promise<void> {
    return new Promise(resolve => setTimeout(resolve, ms));
  }
}

// Singleton instance for global error handling
export const globalErrorHandler = new ErrorHandler();

// Utility function for easy error handling
export async function withErrorHandling<T>(
  operation: () => Promise<T>,
  context: Partial<ErrorContext> = {}
): Promise<T> {
  try {
    return await operation();
  } catch (error) {
    await globalErrorHandler.handleError(error as Error, context);
    throw error;
  }
}

// Decorator for automatic error handling
export function handleErrors(context: Partial<ErrorContext> = {}) {
  return function (target: any, propertyName: string, descriptor: PropertyDescriptor) {
    const method = descriptor.value;

    descriptor.value = async function (...args: any[]) {
      try {
        return await method.apply(this, args);
      } catch (error) {
        await globalErrorHandler.handleError(error as Error, {
          ...context,
          component: target.constructor.name,
          operation: propertyName
        });
        throw error;
      }
    };
  };
}