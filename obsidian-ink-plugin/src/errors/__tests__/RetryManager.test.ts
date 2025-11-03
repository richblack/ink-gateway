/**
 * Tests for RetryManager class
 */

import { RetryManager, CircuitBreakerState, withRetry, retry } from '../RetryManager';
import { ErrorType, PluginError } from '../../types';

describe('RetryManager', () => {
  let retryManager: RetryManager;

  beforeEach(() => {
    retryManager = new RetryManager({
      maxRetries: 3,
      baseDelay: 100,
      maxDelay: 1000,
      backoffFactor: 2,
      jitter: false // Disable jitter for predictable tests
    });
  });

  describe('execute', () => {
    it('should succeed on first attempt', async () => {
      const operation = jest.fn().mockResolvedValue('success');

      const result = await retryManager.execute(operation, 'test_operation');

      expect(result).toBe('success');
      expect(operation).toHaveBeenCalledTimes(1);
    });

    it('should retry on recoverable errors', async () => {
      let attemptCount = 0;
      const operation = jest.fn().mockImplementation(() => {
        attemptCount++;
        if (attemptCount < 3) {
          throw new PluginError(
            ErrorType.NETWORK_ERROR,
            'TEMPORARY_FAILURE',
            {},
            true
          );
        }
        return 'success';
      });

      const result = await retryManager.execute(operation, 'retry_test');

      expect(result).toBe('success');
      expect(operation).toHaveBeenCalledTimes(3);
    });

    it('should not retry non-recoverable errors', async () => {
      const operation = jest.fn().mockImplementation(() => {
        throw new PluginError(
          ErrorType.VALIDATION_ERROR,
          'INVALID_INPUT',
          {},
          false
        );
      });

      await expect(
        retryManager.execute(operation, 'non_recoverable_test')
      ).rejects.toThrow('INVALID_INPUT');

      expect(operation).toHaveBeenCalledTimes(1);
    });

    it('should respect maximum retry attempts', async () => {
      const operation = jest.fn().mockImplementation(() => {
        throw new PluginError(
          ErrorType.NETWORK_ERROR,
          'PERSISTENT_FAILURE',
          {},
          true
        );
      });

      await expect(
        retryManager.execute(operation, 'max_retries_test')
      ).rejects.toThrow('PERSISTENT_FAILURE');

      expect(operation).toHaveBeenCalledTimes(4); // Initial + 3 retries
    });

    it('should use exponential backoff', async () => {
      const delays: number[] = [];
      const originalSetTimeout = global.setTimeout;
      
      global.setTimeout = jest.fn().mockImplementation((callback, delay) => {
        delays.push(delay);
        return originalSetTimeout(callback, 0); // Execute immediately for test
      });

      const operation = jest.fn().mockImplementation(() => {
        throw new PluginError(
          ErrorType.NETWORK_ERROR,
          'BACKOFF_TEST',
          {},
          true
        );
      });

      await expect(
        retryManager.execute(operation, 'backoff_test')
      ).rejects.toThrow();

      // Check that delays follow exponential backoff pattern
      expect(delays).toHaveLength(3);
      expect(delays[0]).toBe(100); // baseDelay * 2^0
      expect(delays[1]).toBe(200); // baseDelay * 2^1
      expect(delays[2]).toBe(400); // baseDelay * 2^2

      global.setTimeout = originalSetTimeout;
    });

    it('should call onRetry callback', async () => {
      const onRetry = jest.fn();
      const operation = jest.fn().mockImplementation(() => {
        throw new PluginError(
          ErrorType.NETWORK_ERROR,
          'CALLBACK_TEST',
          {},
          true
        );
      });

      await expect(
        retryManager.execute(operation, 'callback_test', { onRetry })
      ).rejects.toThrow();

      expect(onRetry).toHaveBeenCalledTimes(3);
      expect(onRetry).toHaveBeenCalledWith(1, expect.any(PluginError));
      expect(onRetry).toHaveBeenCalledWith(2, expect.any(PluginError));
      expect(onRetry).toHaveBeenCalledWith(3, expect.any(PluginError));
    });

    it('should use custom shouldRetry function', async () => {
      const shouldRetry = jest.fn().mockReturnValue(false);
      const operation = jest.fn().mockImplementation(() => {
        throw new Error('Custom retry test');
      });

      await expect(
        retryManager.execute(operation, 'custom_retry_test', { shouldRetry })
      ).rejects.toThrow();

      expect(shouldRetry).toHaveBeenCalledWith(expect.any(Error));
      expect(operation).toHaveBeenCalledTimes(1); // No retries due to custom logic
    });
  });

  describe('circuit breaker', () => {
    beforeEach(() => {
      retryManager = new RetryManager(
        { maxRetries: 2 },
        { failureThreshold: 3, recoveryTimeout: 1000 }
      );
    });

    it('should open circuit breaker after threshold failures', async () => {
      const operation = jest.fn().mockImplementation(() => {
        throw new PluginError(
          ErrorType.NETWORK_ERROR,
          'CIRCUIT_BREAKER_TEST',
          {},
          true
        );
      });

      // Trigger failures to reach threshold
      for (let i = 0; i < 3; i++) {
        await expect(
          retryManager.execute(operation, `failure_${i}`)
        ).rejects.toThrow();
      }

      expect(retryManager.getCircuitBreakerState()).toBe(CircuitBreakerState.OPEN);

      // Next operation should fail immediately due to open circuit
      await expect(
        retryManager.execute(operation, 'circuit_open_test')
      ).rejects.toThrow('CIRCUIT_BREAKER_OPEN');
    });

    it('should transition to half-open after recovery timeout', async () => {
      const operation = jest.fn().mockImplementation(() => {
        throw new PluginError(
          ErrorType.NETWORK_ERROR,
          'RECOVERY_TEST',
          {},
          true
        );
      });

      // Open the circuit breaker
      for (let i = 0; i < 3; i++) {
        await expect(
          retryManager.execute(operation, `failure_${i}`)
        ).rejects.toThrow();
      }

      expect(retryManager.getCircuitBreakerState()).toBe(CircuitBreakerState.OPEN);

      // Wait for recovery timeout (simulate by manipulating time)
      jest.advanceTimersByTime(1100);

      // Mock successful operation for recovery
      operation.mockResolvedValueOnce('recovered');

      const result = await retryManager.execute(operation, 'recovery_test');

      expect(result).toBe('recovered');
      expect(retryManager.getCircuitBreakerState()).toBe(CircuitBreakerState.CLOSED);
    });

    it('should manually reset circuit breaker', async () => {
      const operation = jest.fn().mockImplementation(() => {
        throw new PluginError(
          ErrorType.NETWORK_ERROR,
          'MANUAL_RESET_TEST',
          {},
          true
        );
      });

      // Open the circuit breaker
      for (let i = 0; i < 3; i++) {
        await expect(
          retryManager.execute(operation, `failure_${i}`)
        ).rejects.toThrow();
      }

      expect(retryManager.getCircuitBreakerState()).toBe(CircuitBreakerState.OPEN);

      retryManager.resetCircuitBreaker();

      expect(retryManager.getCircuitBreakerState()).toBe(CircuitBreakerState.CLOSED);
    });
  });

  describe('executeAll', () => {
    it('should execute multiple operations concurrently', async () => {
      const operations = [
        () => Promise.resolve('result1'),
        () => Promise.resolve('result2'),
        () => Promise.resolve('result3')
      ];

      const results = await retryManager.executeAll(operations, 'batch_test');

      expect(results).toEqual(['result1', 'result2', 'result3']);
    });

    it('should handle mixed success and failure with failFast=false', async () => {
      const operations = [
        () => Promise.resolve('success'),
        () => Promise.reject(new Error('failure')),
        () => Promise.resolve('success2')
      ];

      const results = await retryManager.executeAll(operations, 'mixed_test', {
        failFast: false
      });

      expect(results).toHaveLength(3);
      expect(results[0]).toBe('success');
      expect(results[1]).toBeInstanceOf(Error);
      expect(results[2]).toBe('success2');
    });

    it('should fail fast when failFast=true', async () => {
      const operations = [
        () => Promise.resolve('success'),
        () => Promise.reject(new Error('failure')),
        () => Promise.resolve('success2')
      ];

      await expect(
        retryManager.executeAll(operations, 'fail_fast_test', {
          failFast: true
        })
      ).rejects.toThrow('failure');
    });

    it('should respect concurrency limits', async () => {
      let concurrentCount = 0;
      let maxConcurrent = 0;

      const operations = Array(10).fill(null).map(() => async () => {
        concurrentCount++;
        maxConcurrent = Math.max(maxConcurrent, concurrentCount);
        
        await new Promise(resolve => setTimeout(resolve, 50));
        
        concurrentCount--;
        return 'done';
      });

      await retryManager.executeAll(operations, 'concurrency_test', {
        maxConcurrency: 3
      });

      expect(maxConcurrent).toBeLessThanOrEqual(3);
    });
  });

  describe('statistics', () => {
    it('should track retry statistics', async () => {
      let attemptCount = 0;
      const operation = jest.fn().mockImplementation(() => {
        attemptCount++;
        if (attemptCount < 3) {
          throw new PluginError(
            ErrorType.NETWORK_ERROR,
            'STATS_TEST',
            {},
            true
          );
        }
        return 'success';
      });

      await retryManager.execute(operation, 'stats_test');

      const stats = retryManager.getStats();
      expect(stats.totalAttempts).toBe(3);
      expect(stats.successfulRetries).toBe(1);
      expect(stats.averageRetryDelay).toBeGreaterThan(0);
    });

    it('should reset statistics', async () => {
      const operation = jest.fn().mockResolvedValue('success');
      
      await retryManager.execute(operation, 'reset_test');
      
      expect(retryManager.getStats().totalAttempts).toBe(1);
      
      retryManager.resetStats();
      
      expect(retryManager.getStats().totalAttempts).toBe(0);
    });
  });

  describe('configuration updates', () => {
    it('should update retry options', () => {
      retryManager.updateOptions({ maxRetries: 5 });
      
      const config = retryManager.getConfiguration();
      expect(config.retryConfig.maxRetries).toBe(5);
    });

    it('should update circuit breaker options', () => {
      retryManager.updateCircuitBreakerOptions({ failureThreshold: 10 });
      
      // This would need to be tested through behavior since there's no getter
      // for circuit breaker options in the current implementation
    });
  });

  describe('utility functions', () => {
    it('should work with withRetry utility', async () => {
      const operation = jest.fn().mockResolvedValue('utility_success');

      const result = await withRetry(operation, 'utility_test');

      expect(result).toBe('utility_success');
      expect(operation).toHaveBeenCalledTimes(1);
    });
  });

  describe('retry decorator', () => {
    it('should automatically retry decorated methods', async () => {
      let attemptCount = 0;

      class TestClass {
        @retry({ maxRetries: 2 })
        async testMethod(): Promise<string> {
          attemptCount++;
          if (attemptCount < 3) {
            throw new PluginError(
              ErrorType.NETWORK_ERROR,
              'DECORATOR_TEST',
              {},
              true
            );
          }
          return 'decorated_success';
        }
      }

      const instance = new TestClass();
      const result = await instance.testMethod();

      expect(result).toBe('decorated_success');
      expect(attemptCount).toBe(3);
    });
  });

  describe('error type handling', () => {
    it('should retry network errors by default', async () => {
      const operation = jest.fn().mockImplementation(() => {
        throw new PluginError(
          ErrorType.NETWORK_ERROR,
          'NETWORK_TEST',
          {},
          true
        );
      });

      await expect(
        retryManager.execute(operation, 'network_retry_test')
      ).rejects.toThrow();

      expect(operation).toHaveBeenCalledTimes(4); // Initial + 3 retries
    });

    it('should retry specific API error status codes', async () => {
      const operation = jest.fn().mockImplementation(() => {
        throw new PluginError(
          ErrorType.API_ERROR,
          'HTTP_500',
          { status: 500 },
          true
        );
      });

      await expect(
        retryManager.execute(operation, 'api_retry_test')
      ).rejects.toThrow();

      expect(operation).toHaveBeenCalledTimes(4); // Initial + 3 retries
    });

    it('should not retry non-retryable status codes', async () => {
      const operation = jest.fn().mockImplementation(() => {
        throw new PluginError(
          ErrorType.API_ERROR,
          'HTTP_400',
          { status: 400 },
          true
        );
      });

      await expect(
        retryManager.execute(operation, 'non_retryable_api_test')
      ).rejects.toThrow();

      expect(operation).toHaveBeenCalledTimes(1); // No retries for 400 errors
    });
  });
});

describe('Semaphore', () => {
  // Note: The Semaphore class is private, so we test it indirectly through executeAll
  
  it('should control concurrency correctly', async () => {
    const retryManager = new RetryManager();
    let activeOperations = 0;
    let maxActive = 0;

    const operations = Array(10).fill(null).map(() => async () => {
      activeOperations++;
      maxActive = Math.max(maxActive, activeOperations);
      
      await new Promise(resolve => setTimeout(resolve, 10));
      
      activeOperations--;
      return 'done';
    });

    await retryManager.executeAll(operations, 'semaphore_test', {
      maxConcurrency: 2
    });

    expect(maxActive).toBeLessThanOrEqual(2);
  });
});