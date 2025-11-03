/**
 * Advanced retry mechanism with exponential backoff and circuit breaker pattern
 * Provides intelligent retry logic for network operations and API calls
 */

import { ErrorType, PluginError } from '../types';
import { globalErrorHandler } from './ErrorHandler';

// Retry configuration options
export interface RetryOptions {
  maxRetries: number;
  baseDelay: number;
  maxDelay: number;
  backoffFactor: number;
  jitter: boolean;
  retryableErrors: ErrorType[];
  retryableStatusCodes: number[];
  onRetry?: (attempt: number, error: Error) => void;
  shouldRetry?: (error: Error) => boolean;
}

// Circuit breaker configuration
export interface CircuitBreakerOptions {
  failureThreshold: number;
  recoveryTimeout: number;
  monitoringPeriod: number;
}

// Circuit breaker states
export enum CircuitBreakerState {
  CLOSED = 'closed',
  OPEN = 'open',
  HALF_OPEN = 'half_open'
}

// Retry statistics
export interface RetryStats {
  totalAttempts: number;
  successfulRetries: number;
  failedRetries: number;
  averageRetryDelay: number;
  circuitBreakerTrips: number;
}

// Default retry configuration
const DEFAULT_RETRY_OPTIONS: RetryOptions = {
  maxRetries: 3,
  baseDelay: 1000,
  maxDelay: 30000,
  backoffFactor: 2,
  jitter: true,
  retryableErrors: [ErrorType.NETWORK_ERROR, ErrorType.API_ERROR],
  retryableStatusCodes: [408, 429, 500, 502, 503, 504]
};

// Default circuit breaker configuration
const DEFAULT_CIRCUIT_BREAKER_OPTIONS: CircuitBreakerOptions = {
  failureThreshold: 5,
  recoveryTimeout: 60000, // 1 minute
  monitoringPeriod: 300000 // 5 minutes
};

export class RetryManager {
  private options: RetryOptions;
  private circuitBreakerOptions: CircuitBreakerOptions;
  private circuitBreakerState: CircuitBreakerState = CircuitBreakerState.CLOSED;
  private failureCount: number = 0;
  private lastFailureTime: number = 0;
  private stats: RetryStats = {
    totalAttempts: 0,
    successfulRetries: 0,
    failedRetries: 0,
    averageRetryDelay: 0,
    circuitBreakerTrips: 0
  };
  private delayHistory: number[] = [];

  constructor(
    options: Partial<RetryOptions> = {},
    circuitBreakerOptions: Partial<CircuitBreakerOptions> = {}
  ) {
    this.options = { ...DEFAULT_RETRY_OPTIONS, ...options };
    this.circuitBreakerOptions = { ...DEFAULT_CIRCUIT_BREAKER_OPTIONS, ...circuitBreakerOptions };
  }

  /**
   * Execute operation with retry logic and circuit breaker
   */
  async execute<T>(
    operation: () => Promise<T>,
    operationName: string = 'unknown',
    customOptions?: Partial<RetryOptions>
  ): Promise<T> {
    const effectiveOptions = { ...this.options, ...customOptions };
    
    // Check circuit breaker state
    if (this.circuitBreakerState === CircuitBreakerState.OPEN) {
      if (this.shouldAttemptRecovery()) {
        this.circuitBreakerState = CircuitBreakerState.HALF_OPEN;
        console.log(`[RetryManager] Circuit breaker moving to HALF_OPEN state for ${operationName}`);
      } else {
        throw new PluginError(
          ErrorType.NETWORK_ERROR,
          'CIRCUIT_BREAKER_OPEN',
          { 
            operationName,
            failureCount: this.failureCount,
            lastFailureTime: this.lastFailureTime
          },
          false
        );
      }
    }

    let lastError: Error | null = null;
    
    for (let attempt = 0; attempt <= effectiveOptions.maxRetries; attempt++) {
      this.stats.totalAttempts++;
      
      try {
        const result = await operation();
        
        // Success - reset circuit breaker if needed
        if (this.circuitBreakerState === CircuitBreakerState.HALF_OPEN) {
          this.circuitBreakerState = CircuitBreakerState.CLOSED;
          this.failureCount = 0;
          console.log(`[RetryManager] Circuit breaker closed for ${operationName}`);
        }
        
        if (attempt > 0) {
          this.stats.successfulRetries++;
          console.log(`[RetryManager] Operation ${operationName} succeeded after ${attempt} retries`);
        }
        
        return result;

      } catch (error) {
        lastError = error as Error;
        
        // Check if we should retry
        if (attempt === effectiveOptions.maxRetries || !this.shouldRetry(error as Error, effectiveOptions)) {
          this.handleFailure(operationName);
          this.stats.failedRetries++;
          break;
        }

        // Calculate delay for next retry
        const delay = this.calculateDelay(attempt, effectiveOptions);
        this.delayHistory.push(delay);
        
        // Update average delay
        this.stats.averageRetryDelay = this.delayHistory.reduce((a, b) => a + b, 0) / this.delayHistory.length;
        
        // Call retry callback if provided
        if (effectiveOptions.onRetry) {
          effectiveOptions.onRetry(attempt + 1, error as Error);
        }

        console.warn(
          `[RetryManager] ${operationName} failed (attempt ${attempt + 1}/${effectiveOptions.maxRetries + 1}), retrying in ${delay}ms`,
          error
        );

        // Wait before retry
        await this.sleep(delay);
      }
    }

    // All retries exhausted
    await globalErrorHandler.handleError(lastError!, {
      operation: operationName,
      component: 'RetryManager',
      additionalData: {
        maxRetries: effectiveOptions.maxRetries,
        circuitBreakerState: this.circuitBreakerState
      }
    });

    throw lastError;
  }

  /**
   * Execute multiple operations with retry logic
   */
  async executeAll<T>(
    operations: Array<() => Promise<T>>,
    operationName: string = 'batch_operation',
    options?: {
      failFast?: boolean;
      maxConcurrency?: number;
      retryOptions?: Partial<RetryOptions>;
    }
  ): Promise<Array<T | Error>> {
    const { failFast = false, maxConcurrency = 5, retryOptions } = options || {};
    
    const results: Array<T | Error> = [];
    const semaphore = new Semaphore(maxConcurrency);
    
    const executeWithSemaphore = async (operation: () => Promise<T>, index: number): Promise<T | Error> => {
      await semaphore.acquire();
      try {
        return await this.execute(operation, `${operationName}_${index}`, retryOptions);
      } catch (error) {
        if (failFast) {
          throw error;
        }
        return error instanceof Error ? error : new Error(String(error));
      } finally {
        semaphore.release();
      }
    };

    try {
      const promises = operations.map((operation, index) => 
        executeWithSemaphore(operation, index)
      );
      
      const batchResults = await Promise.all(promises);
      results.push(...batchResults);
      
    } catch (error) {
      if (failFast) {
        throw error;
      }
    }

    return results;
  }

  /**
   * Get retry statistics
   */
  getStats(): RetryStats {
    return { ...this.stats };
  }

  /**
   * Reset statistics
   */
  resetStats(): void {
    this.stats = {
      totalAttempts: 0,
      successfulRetries: 0,
      failedRetries: 0,
      averageRetryDelay: 0,
      circuitBreakerTrips: 0
    };
    this.delayHistory = [];
  }

  /**
   * Get circuit breaker state
   */
  getCircuitBreakerState(): CircuitBreakerState {
    return this.circuitBreakerState;
  }

  /**
   * Manually reset circuit breaker
   */
  resetCircuitBreaker(): void {
    this.circuitBreakerState = CircuitBreakerState.CLOSED;
    this.failureCount = 0;
    this.lastFailureTime = 0;
    console.log('[RetryManager] Circuit breaker manually reset');
  }

  /**
   * Update retry options
   */
  updateOptions(options: Partial<RetryOptions>): void {
    this.options = { ...this.options, ...options };
  }

  /**
   * Update circuit breaker options
   */
  updateCircuitBreakerOptions(options: Partial<CircuitBreakerOptions>): void {
    this.circuitBreakerOptions = { ...this.circuitBreakerOptions, ...options };
  }

  // Private methods

  private shouldRetry(error: Error, options: RetryOptions): boolean {
    // Custom retry logic if provided
    if (options.shouldRetry) {
      return options.shouldRetry(error);
    }

    // Check if error is a PluginError
    if (error instanceof PluginError) {
      // Don't retry if explicitly marked as non-recoverable
      if (!error.recoverable) {
        return false;
      }

      // Check if error type is retryable
      if (!options.retryableErrors.includes(error.type)) {
        return false;
      }

      // Check specific status codes for API errors
      if (error.type === ErrorType.API_ERROR && error.details?.status) {
        return options.retryableStatusCodes.includes(error.details.status);
      }

      return true;
    }

    // For non-PluginError, check common patterns
    const errorMessage = error.message.toLowerCase();
    
    // Network-related errors are usually retryable
    if (errorMessage.includes('network') || 
        errorMessage.includes('timeout') || 
        errorMessage.includes('connection') ||
        errorMessage.includes('fetch')) {
      return true;
    }

    return false;
  }

  private calculateDelay(attempt: number, options: RetryOptions): number {
    // Base exponential backoff
    let delay = options.baseDelay * Math.pow(options.backoffFactor, attempt);
    
    // Apply jitter to prevent thundering herd
    if (options.jitter) {
      delay = delay * (0.5 + Math.random() * 0.5);
    }
    
    // Cap at maximum delay
    return Math.min(delay, options.maxDelay);
  }

  private handleFailure(operationName: string): void {
    this.failureCount++;
    this.lastFailureTime = Date.now();
    
    // Check if we should trip the circuit breaker
    if (this.failureCount >= this.circuitBreakerOptions.failureThreshold) {
      this.circuitBreakerState = CircuitBreakerState.OPEN;
      this.stats.circuitBreakerTrips++;
      console.warn(`[RetryManager] Circuit breaker opened for ${operationName} after ${this.failureCount} failures`);
    }
  }

  private shouldAttemptRecovery(): boolean {
    const timeSinceLastFailure = Date.now() - this.lastFailureTime;
    return timeSinceLastFailure >= this.circuitBreakerOptions.recoveryTimeout;
  }

  private sleep(ms: number): Promise<void> {
    return new Promise(resolve => setTimeout(resolve, ms));
  }
}

// Semaphore for controlling concurrency
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

// Global retry manager instance
export const globalRetryManager = new RetryManager();

// Utility function for easy retry execution
export async function withRetry<T>(
  operation: () => Promise<T>,
  operationName: string = 'operation',
  options?: Partial<RetryOptions>
): Promise<T> {
  return globalRetryManager.execute(operation, operationName, options);
}

// Decorator for automatic retry
export function retry(options: Partial<RetryOptions> = {}) {
  return function (target: any, propertyName: string, descriptor: PropertyDescriptor) {
    const method = descriptor.value;

    descriptor.value = async function (...args: any[]) {
      const operationName = `${target.constructor.name}.${propertyName}`;
      return globalRetryManager.execute(
        () => method.apply(this, args),
        operationName,
        options
      );
    };
  };
}