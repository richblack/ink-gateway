/**
 * Tests for ErrorHandler class
 */

import { ErrorHandler, ErrorSeverity, withErrorHandling, handleErrors } from '../ErrorHandler';
import { ErrorType, PluginError } from '../../types';

// Mock Obsidian Notice
jest.mock('obsidian', () => ({
  Notice: jest.fn().mockImplementation((message: string, duration?: number) => {
    console.log(`Notice: ${message} (duration: ${duration})`);
  })
}));

describe('ErrorHandler', () => {
  let errorHandler: ErrorHandler;

  beforeEach(() => {
    errorHandler = new ErrorHandler(true); // Enable debug mode
    jest.clearAllMocks();
  });

  describe('handleError', () => {
    it('should handle PluginError correctly', async () => {
      const error = new PluginError(
        ErrorType.NETWORK_ERROR,
        'CONNECTION_FAILED',
        { url: 'http://example.com' },
        true
      );

      await errorHandler.handleError(error, {
        component: 'TestComponent',
        operation: 'testOperation'
      });

      const stats = errorHandler.getErrorStats();
      expect(stats.totalErrors).toBe(1);
      expect(stats.errorsByType[ErrorType.NETWORK_ERROR]).toBe(1);
    });

    it('should normalize regular Error to PluginError', async () => {
      const error = new Error('Test error message');

      await errorHandler.handleError(error, {
        component: 'TestComponent',
        operation: 'testOperation'
      });

      const recentErrors = errorHandler.getRecentErrors(1);
      expect(recentErrors).toHaveLength(1);
      expect(recentErrors[0].error).toBeInstanceOf(PluginError);
    });

    it('should determine correct severity levels', async () => {
      const criticalError = new PluginError(
        ErrorType.API_ERROR,
        'HTTP_401',
        {},
        false
      );

      await errorHandler.handleError(criticalError);

      const recentErrors = errorHandler.getRecentErrors(1);
      expect(recentErrors[0].severity).toBe(ErrorSeverity.CRITICAL);
    });

    it('should execute fallback action when provided', async () => {
      const fallbackAction = jest.fn().mockResolvedValue(undefined);
      
      errorHandler.registerErrorStrategy('TEST_ERROR', {
        showToUser: false,
        logError: true,
        retryable: false,
        fallbackAction
      });

      const error = new PluginError(
        ErrorType.VALIDATION_ERROR,
        'TEST_ERROR',
        {},
        false
      );

      await errorHandler.handleError(error);

      expect(fallbackAction).toHaveBeenCalled();
    });
  });

  describe('handleErrorWithRetry', () => {
    it('should retry recoverable errors', async () => {
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

      const result = await errorHandler.handleErrorWithRetry(
        operation,
        { component: 'Test', operation: 'retry_test' },
        3,
        100
      );

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
        errorHandler.handleErrorWithRetry(operation, {}, 3, 100)
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
        errorHandler.handleErrorWithRetry(operation, {}, 2, 50)
      ).rejects.toThrow('PERSISTENT_FAILURE');

      expect(operation).toHaveBeenCalledTimes(3); // Initial + 2 retries
    });
  });

  describe('error statistics', () => {
    it('should track error statistics correctly', async () => {
      const errors = [
        new PluginError(ErrorType.NETWORK_ERROR, 'NET_1', {}, true),
        new PluginError(ErrorType.NETWORK_ERROR, 'NET_2', {}, true),
        new PluginError(ErrorType.API_ERROR, 'API_1', {}, false),
        new PluginError(ErrorType.SYNC_ERROR, 'SYNC_1', {}, true)
      ];

      for (const error of errors) {
        await errorHandler.handleError(error);
      }

      const stats = errorHandler.getErrorStats();
      expect(stats.totalErrors).toBe(4);
      expect(stats.errorsByType[ErrorType.NETWORK_ERROR]).toBe(2);
      expect(stats.errorsByType[ErrorType.API_ERROR]).toBe(1);
      expect(stats.errorsByType[ErrorType.SYNC_ERROR]).toBe(1);
    });

    it('should calculate average resolution time', async () => {
      const error = new PluginError(ErrorType.NETWORK_ERROR, 'TEST', {}, true);
      
      await errorHandler.handleError(error);
      
      const recentErrors = errorHandler.getRecentErrors(1);
      const errorId = recentErrors[0].id;
      
      // Simulate some time passing
      await new Promise(resolve => setTimeout(resolve, 10));
      
      errorHandler.resolveError(errorId);
      
      const stats = errorHandler.getErrorStats();
      expect(stats.averageResolutionTime).toBeGreaterThan(0);
    });
  });

  describe('error strategies', () => {
    it('should use custom error strategies', async () => {
      const customStrategy = {
        showToUser: false,
        logError: true,
        retryable: true,
        maxRetries: 5,
        customMessage: 'Custom error message'
      };

      errorHandler.registerErrorStrategy('CUSTOM_ERROR', customStrategy);

      const error = new PluginError(
        ErrorType.NETWORK_ERROR,
        'CUSTOM_ERROR',
        {},
        true
      );

      await errorHandler.handleError(error);

      // Verify the custom strategy was applied
      const recentErrors = errorHandler.getRecentErrors(1);
      expect(recentErrors).toHaveLength(1);
    });
  });

  describe('utility functions', () => {
    it('should work with withErrorHandling utility', async () => {
      const operation = jest.fn().mockRejectedValue(
        new Error('Test error')
      );

      await expect(
        withErrorHandling(operation, {
          component: 'TestComponent',
          operation: 'testOp'
        })
      ).rejects.toThrow('Test error');

      expect(operation).toHaveBeenCalled();
    });
  });

  describe('handleErrors decorator', () => {
    it('should automatically handle errors in decorated methods', async () => {
      class TestClass {
        @handleErrors({ component: 'TestClass' })
        async testMethod(): Promise<string> {
          throw new Error('Decorated method error');
        }
      }

      const instance = new TestClass();
      
      await expect(instance.testMethod()).rejects.toThrow('Decorated method error');
      
      // Verify error was logged
      const stats = errorHandler.getErrorStats();
      expect(stats.totalErrors).toBeGreaterThan(0);
    });
  });

  describe('log management', () => {
    it('should maintain log size limit', async () => {
      const handler = new ErrorHandler(true);
      
      // Generate more errors than the default limit
      for (let i = 0; i < 1200; i++) {
        await handler.handleError(
          new PluginError(ErrorType.NETWORK_ERROR, `ERROR_${i}`, {}, true)
        );
      }

      const stats = handler.getErrorStats();
      expect(stats.totalErrors).toBeLessThanOrEqual(1000); // Default max log size
    });

    it('should export error log correctly', () => {
      const exportedLog = errorHandler.exportErrorLog();
      expect(exportedLog).toBeDefined();
      expect(() => JSON.parse(exportedLog)).not.toThrow();
    });

    it('should clear error log', async () => {
      await errorHandler.handleError(
        new PluginError(ErrorType.NETWORK_ERROR, 'TEST', {}, true)
      );

      expect(errorHandler.getErrorStats().totalErrors).toBe(1);

      errorHandler.clearErrorLog();

      expect(errorHandler.getErrorStats().totalErrors).toBe(0);
    });
  });

  describe('error resolution', () => {
    it('should mark errors as resolved', async () => {
      const error = new PluginError(ErrorType.NETWORK_ERROR, 'TEST', {}, true);
      
      await errorHandler.handleError(error);
      
      const recentErrors = errorHandler.getRecentErrors(1);
      const errorId = recentErrors[0].id;
      
      expect(recentErrors[0].resolved).toBe(false);
      
      errorHandler.resolveError(errorId);
      
      const updatedErrors = errorHandler.getRecentErrors(1);
      expect(updatedErrors[0].resolved).toBe(true);
      expect(updatedErrors[0].resolvedAt).toBeDefined();
    });
  });
});

describe('Error message mapping', () => {
  let errorHandler: ErrorHandler;

  beforeEach(() => {
    errorHandler = new ErrorHandler(true);
  });

  it('should provide user-friendly messages for common errors', async () => {
    const testCases = [
      {
        error: new PluginError(ErrorType.NETWORK_ERROR, 'NETWORK_ERROR', {}, true),
        expectedMessage: 'Unable to connect to Ink-Gateway. Please check your internet connection.'
      },
      {
        error: new PluginError(ErrorType.API_ERROR, 'HTTP_401', {}, false),
        expectedMessage: 'Authentication failed. Please check your API key in settings.'
      },
      {
        error: new PluginError(ErrorType.SYNC_ERROR, 'SYNC_CONFLICT', {}, true),
        expectedMessage: 'Content conflict detected. Please resolve manually.'
      }
    ];

    for (const testCase of testCases) {
      await errorHandler.handleError(testCase.error);
      
      const recentErrors = errorHandler.getRecentErrors(1);
      expect(recentErrors[0].userMessage).toBe(testCase.expectedMessage);
    }
  });
});