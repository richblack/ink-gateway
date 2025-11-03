/**
 * Tests for DebugLogger class
 */

import { DebugLogger, LogLevel, timed, logged } from '../DebugLogger';

describe('DebugLogger', () => {
  let debugLogger: DebugLogger;

  beforeEach(() => {
    debugLogger = new DebugLogger({
      enabled: true,
      logLevel: LogLevel.DEBUG,
      logToConsole: false // Disable console logging for tests
    });
  });

  describe('basic logging', () => {
    it('should log messages at different levels', () => {
      debugLogger.trace('TestComponent', 'testOp', 'Trace message');
      debugLogger.debug('TestComponent', 'testOp', 'Debug message');
      debugLogger.info('TestComponent', 'testOp', 'Info message');
      debugLogger.warn('TestComponent', 'testOp', 'Warning message');
      debugLogger.error('TestComponent', 'testOp', 'Error message');
      debugLogger.fatal('TestComponent', 'testOp', 'Fatal message');

      const logs = debugLogger.getRecentLogs();
      expect(logs).toHaveLength(5); // Trace is below DEBUG level, so not logged
    });

    it('should respect log level filtering', () => {
      const logger = new DebugLogger({
        enabled: true,
        logLevel: LogLevel.WARN,
        logToConsole: false
      });

      logger.debug('Test', 'op', 'Debug message');
      logger.info('Test', 'op', 'Info message');
      logger.warn('Test', 'op', 'Warning message');
      logger.error('Test', 'op', 'Error message');

      const logs = logger.getRecentLogs();
      expect(logs).toHaveLength(2); // Only WARN and ERROR
      expect(logs.every(log => log.level >= LogLevel.WARN)).toBe(true);
    });

    it('should include error objects and stack traces', () => {
      const error = new Error('Test error');
      
      debugLogger.error('TestComponent', 'errorOp', 'Error occurred', error);

      const logs = debugLogger.getRecentLogs(1);
      expect(logs[0].error).toBe(error);
      expect(logs[0].stackTrace).toBeDefined();
    });

    it('should include additional data', () => {
      const data = { userId: '123', action: 'test' };
      
      debugLogger.info('TestComponent', 'dataOp', 'Operation with data', data);

      const logs = debugLogger.getRecentLogs(1);
      expect(logs[0].data).toEqual(data);
    });
  });

  describe('performance timing', () => {
    it('should track operation timing', () => {
      debugLogger.startTimer('testOperation');
      
      // Simulate some work
      const start = performance.now();
      while (performance.now() - start < 10) {
        // Busy wait for 10ms
      }
      
      const metrics = debugLogger.endTimer('testOperation');
      
      expect(metrics).toBeDefined();
      expect(metrics!.operationName).toBe('testOperation');
      expect(metrics!.duration).toBeGreaterThan(5);
    });

    it('should handle missing start timer', () => {
      const metrics = debugLogger.endTimer('nonExistentOperation');
      
      expect(metrics).toBeNull();
      
      const logs = debugLogger.getRecentLogs();
      const warningLog = logs.find(log => 
        log.level === LogLevel.WARN && 
        log.message.includes('No start time found')
      );
      expect(warningLog).toBeDefined();
    });

    it('should include additional metrics', () => {
      debugLogger.startTimer('metricsTest');
      
      const additionalMetrics = { itemsProcessed: 100, cacheHits: 50 };
      const metrics = debugLogger.endTimer('metricsTest', additionalMetrics);
      
      expect(metrics!.additionalMetrics).toEqual(additionalMetrics);
    });

    it('should collect performance summary', () => {
      // Run multiple operations
      for (let i = 0; i < 3; i++) {
        debugLogger.startTimer('repeatedOp');
        debugLogger.endTimer('repeatedOp');
      }
      
      debugLogger.startTimer('singleOp');
      debugLogger.endTimer('singleOp');
      
      const summary = debugLogger.getPerformanceSummary();
      
      expect(summary.repeatedOp.count).toBe(3);
      expect(summary.singleOp.count).toBe(1);
      expect(summary.repeatedOp.averageDuration).toBeGreaterThan(0);
    });
  });

  describe('session management', () => {
    it('should create session with metadata', () => {
      const session = debugLogger.getSession();
      
      expect(session.sessionId).toBeDefined();
      expect(session.startTime).toBeInstanceOf(Date);
      expect(session.userAgent).toBe(navigator.userAgent);
      expect(session.pluginVersion).toBeDefined();
    });

    it('should track session statistics', () => {
      debugLogger.info('Test', 'op1', 'Info message');
      debugLogger.warn('Test', 'op2', 'Warning message');
      debugLogger.error('Test', 'op3', 'Error message');
      
      const session = debugLogger.getSession();
      
      expect(session.totalLogs).toBe(3);
      expect(session.errorCount).toBe(1);
      expect(session.warningCount).toBe(1);
    });
  });

  describe('log management', () => {
    it('should maintain log size limit', () => {
      const logger = new DebugLogger({
        enabled: true,
        maxLogEntries: 5,
        logToConsole: false
      });

      // Add more logs than the limit
      for (let i = 0; i < 10; i++) {
        logger.info('Test', 'op', `Message ${i}`);
      }

      const logs = logger.getRecentLogs();
      expect(logs.length).toBeLessThanOrEqual(5);
    });

    it('should filter logs by level', () => {
      debugLogger.debug('Test', 'op', 'Debug message');
      debugLogger.info('Test', 'op', 'Info message');
      debugLogger.warn('Test', 'op', 'Warning message');
      debugLogger.error('Test', 'op', 'Error message');

      const errorLogs = debugLogger.getRecentLogs(10, LogLevel.ERROR);
      expect(errorLogs).toHaveLength(1);
      expect(errorLogs[0].level).toBe(LogLevel.ERROR);

      const warnAndAbove = debugLogger.getRecentLogs(10, LogLevel.WARN);
      expect(warnAndAbove).toHaveLength(2);
    });

    it('should clear logs', () => {
      debugLogger.info('Test', 'op', 'Test message');
      expect(debugLogger.getRecentLogs()).toHaveLength(1);

      debugLogger.clear();
      expect(debugLogger.getRecentLogs()).toHaveLength(0);
    });
  });

  describe('log export', () => {
    beforeEach(() => {
      debugLogger.info('Test', 'op1', 'First message', { data: 'test1' });
      debugLogger.warn('Test', 'op2', 'Warning message');
      debugLogger.error('Test', 'op3', 'Error message', new Error('Test error'));
    });

    it('should export logs as JSON', () => {
      const exported = debugLogger.exportLogs('json');
      
      expect(() => JSON.parse(exported)).not.toThrow();
      
      const parsed = JSON.parse(exported);
      expect(parsed.session).toBeDefined();
      expect(parsed.logs).toHaveLength(3);
      expect(parsed.performanceMetrics).toBeDefined();
    });

    it('should export logs as CSV', () => {
      const exported = debugLogger.exportLogs('csv');
      
      const lines = exported.split('\n');
      expect(lines[0]).toContain('Timestamp,Level,Component,Operation,Message,Error');
      expect(lines.length).toBeGreaterThan(3); // Header + 3 log entries
    });

    it('should export logs as text', () => {
      const exported = debugLogger.exportLogs('text');
      
      expect(exported).toContain('Debug Session:');
      expect(exported).toContain('First message');
      expect(exported).toContain('Warning message');
      expect(exported).toContain('Error message');
    });

    it('should throw error for unsupported format', () => {
      expect(() => {
        debugLogger.exportLogs('xml' as any);
      }).toThrow('Unsupported export format: xml');
    });
  });

  describe('configuration', () => {
    it('should update configuration', () => {
      const newConfig = {
        logLevel: LogLevel.ERROR,
        maxLogEntries: 500
      };

      debugLogger.updateConfig(newConfig);

      const config = debugLogger.getConfig();
      expect(config.logLevel).toBe(LogLevel.ERROR);
      expect(config.maxLogEntries).toBe(500);
    });

    it('should respect disabled state', () => {
      const logger = new DebugLogger({
        enabled: false,
        logToConsole: false
      });

      logger.info('Test', 'op', 'This should not be logged');

      const logs = logger.getRecentLogs();
      expect(logs).toHaveLength(0);
    });
  });

  describe('global error handling', () => {
    it('should set up global error handlers when enabled', () => {
      const addEventListenerSpy = jest.spyOn(window, 'addEventListener');

      new DebugLogger({ enabled: true });

      expect(addEventListenerSpy).toHaveBeenCalledWith(
        'unhandledrejection',
        expect.any(Function)
      );
      expect(addEventListenerSpy).toHaveBeenCalledWith(
        'error',
        expect.any(Function)
      );

      addEventListenerSpy.mockRestore();
    });
  });

  describe('decorators', () => {
    it('should work with timed decorator', async () => {
      class TestClass {
        @timed('customTimedOp')
        async timedMethod(): Promise<string> {
          await new Promise(resolve => setTimeout(resolve, 10));
          return 'timed_result';
        }
      }

      const instance = new TestClass();
      const result = await instance.timedMethod();

      expect(result).toBe('timed_result');

      const metrics = debugLogger.getPerformanceMetrics();
      const timedMetric = metrics.find(m => m.operationName === 'customTimedOp');
      expect(timedMetric).toBeDefined();
      expect(timedMetric!.duration).toBeGreaterThan(5);
    });

    it('should work with logged decorator', async () => {
      class TestClass {
        @logged('TestClass')
        async loggedMethod(param: string): Promise<string> {
          return `processed_${param}`;
        }
      }

      const instance = new TestClass();
      const result = await instance.loggedMethod('test');

      expect(result).toBe('processed_test');

      const logs = debugLogger.getRecentLogs();
      const startLog = logs.find(log => 
        log.message.includes('Starting loggedMethod')
      );
      const endLog = logs.find(log => 
        log.message.includes('Completed loggedMethod')
      );

      expect(startLog).toBeDefined();
      expect(endLog).toBeDefined();
    });

    it('should handle errors in logged decorator', async () => {
      class TestClass {
        @logged('TestClass')
        async errorMethod(): Promise<string> {
          throw new Error('Test error in logged method');
        }
      }

      const instance = new TestClass();

      await expect(instance.errorMethod()).rejects.toThrow('Test error in logged method');

      const logs = debugLogger.getRecentLogs();
      const errorLog = logs.find(log => 
        log.message.includes('Failed errorMethod')
      );

      expect(errorLog).toBeDefined();
      expect(errorLog!.level).toBe(LogLevel.ERROR);
    });
  });

  describe('memory usage tracking', () => {
    it('should include memory usage in performance metrics when available', () => {
      // Mock performance.memory
      const originalMemory = (performance as any).memory;
      (performance as any).memory = {
        usedJSHeapSize: 1000000,
        totalJSHeapSize: 2000000
      };

      debugLogger.startTimer('memoryTest');
      const metrics = debugLogger.endTimer('memoryTest');

      expect(metrics!.memoryUsage).toBeDefined();
      expect(metrics!.memoryUsage!.used).toBe(1000000);
      expect(metrics!.memoryUsage!.total).toBe(2000000);

      // Restore original memory object
      (performance as any).memory = originalMemory;
    });
  });
});