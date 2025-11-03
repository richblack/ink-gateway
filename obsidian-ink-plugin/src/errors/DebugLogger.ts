/**
 * Advanced debugging and logging system
 * Provides comprehensive logging, debugging information collection, and diagnostic tools
 */

import { ErrorLogEntry } from './ErrorHandler';
import { PluginError, ErrorType } from '../types';

// Log levels
export enum LogLevel {
  TRACE = 0,
  DEBUG = 1,
  INFO = 2,
  WARN = 3,
  ERROR = 4,
  FATAL = 5
}

// Log entry structure
export interface LogEntry {
  id: string;
  timestamp: Date;
  level: LogLevel;
  component: string;
  operation: string;
  message: string;
  data?: any;
  error?: Error;
  stackTrace?: string;
  sessionId: string;
  userId?: string;
}

// Debug session information
export interface DebugSession {
  sessionId: string;
  startTime: Date;
  endTime?: Date;
  userAgent: string;
  pluginVersion: string;
  obsidianVersion: string;
  totalLogs: number;
  errorCount: number;
  warningCount: number;
}

// Performance metrics
export interface PerformanceMetrics {
  operationName: string;
  startTime: number;
  endTime: number;
  duration: number;
  memoryUsage?: {
    used: number;
    total: number;
  };
  additionalMetrics?: Record<string, number>;
}

// Debug configuration
export interface DebugConfig {
  enabled: boolean;
  logLevel: LogLevel;
  maxLogEntries: number;
  includeStackTrace: boolean;
  collectPerformanceMetrics: boolean;
  persistLogs: boolean;
  logToConsole: boolean;
  logToFile: boolean;
}

// Default debug configuration
const DEFAULT_DEBUG_CONFIG: DebugConfig = {
  enabled: false,
  logLevel: LogLevel.INFO,
  maxLogEntries: 1000,
  includeStackTrace: true,
  collectPerformanceMetrics: true,
  persistLogs: false,
  logToConsole: true,
  logToFile: false
};

export class DebugLogger {
  private config: DebugConfig;
  private logs: LogEntry[] = [];
  private session: DebugSession;
  private performanceMetrics: PerformanceMetrics[] = [];
  private activeTimers: Map<string, number> = new Map();

  constructor(config: Partial<DebugConfig> = {}) {
    this.config = { ...DEFAULT_DEBUG_CONFIG, ...config };
    this.session = this.createSession();
    
    // Set up global error handlers if debugging is enabled
    if (this.config.enabled) {
      this.setupGlobalErrorHandlers();
    }
  }

  /**
   * Log a trace message
   */
  trace(component: string, operation: string, message: string, data?: any): void {
    this.log(LogLevel.TRACE, component, operation, message, data);
  }

  /**
   * Log a debug message
   */
  debug(component: string, operation: string, message: string, data?: any): void {
    this.log(LogLevel.DEBUG, component, operation, message, data);
  }

  /**
   * Log an info message
   */
  info(component: string, operation: string, message: string, data?: any): void {
    this.log(LogLevel.INFO, component, operation, message, data);
  }

  /**
   * Log a warning message
   */
  warn(component: string, operation: string, message: string, data?: any): void {
    this.log(LogLevel.WARN, component, operation, message, data);
  }

  /**
   * Log an error message
   */
  error(component: string, operation: string, message: string, error?: Error, data?: any): void {
    this.log(LogLevel.ERROR, component, operation, message, data, error);
  }

  /**
   * Log a fatal error message
   */
  fatal(component: string, operation: string, message: string, error?: Error, data?: any): void {
    this.log(LogLevel.FATAL, component, operation, message, data, error);
  }

  /**
   * Start performance timing for an operation
   */
  startTimer(operationName: string): void {
    if (!this.config.collectPerformanceMetrics) return;
    
    this.activeTimers.set(operationName, performance.now());
    this.debug('PerformanceTimer', 'start', `Started timer for ${operationName}`);
  }

  /**
   * End performance timing and record metrics
   */
  endTimer(operationName: string, additionalMetrics?: Record<string, number>): PerformanceMetrics | null {
    if (!this.config.collectPerformanceMetrics) return null;
    
    const startTime = this.activeTimers.get(operationName);
    if (!startTime) {
      this.warn('PerformanceTimer', 'end', `No start time found for ${operationName}`);
      return null;
    }

    const endTime = performance.now();
    const duration = endTime - startTime;
    
    const metrics: PerformanceMetrics = {
      operationName,
      startTime,
      endTime,
      duration,
      additionalMetrics
    };

    // Add memory usage if available
    if ('memory' in performance) {
      const memory = (performance as any).memory;
      metrics.memoryUsage = {
        used: memory.usedJSHeapSize,
        total: memory.totalJSHeapSize
      };
    }

    this.performanceMetrics.push(metrics);
    this.activeTimers.delete(operationName);
    
    this.debug('PerformanceTimer', 'end', 
      `Completed ${operationName} in ${duration.toFixed(2)}ms`, metrics);
    
    return metrics;
  }

  /**
   * Log an error entry from the error handler
   */
  logErrorEntry(entry: ErrorLogEntry): void {
    this.log(
      LogLevel.ERROR,
      entry.context.component,
      entry.context.operation,
      entry.userMessage,
      {
        errorId: entry.id,
        errorType: entry.error.type,
        errorCode: entry.error.code,
        severity: entry.severity,
        details: entry.error.details
      },
      entry.error
    );
  }

  /**
   * Get recent logs
   */
  getRecentLogs(limit: number = 100, level?: LogLevel): LogEntry[] {
    let filteredLogs = this.logs;
    
    if (level !== undefined) {
      filteredLogs = this.logs.filter(log => log.level >= level);
    }
    
    return filteredLogs
      .sort((a, b) => b.timestamp.getTime() - a.timestamp.getTime())
      .slice(0, limit);
  }

  /**
   * Get performance metrics
   */
  getPerformanceMetrics(): PerformanceMetrics[] {
    return [...this.performanceMetrics];
  }

  /**
   * Get performance summary
   */
  getPerformanceSummary(): Record<string, {
    count: number;
    averageDuration: number;
    minDuration: number;
    maxDuration: number;
    totalDuration: number;
  }> {
    const summary: Record<string, any> = {};
    
    for (const metric of this.performanceMetrics) {
      if (!summary[metric.operationName]) {
        summary[metric.operationName] = {
          count: 0,
          totalDuration: 0,
          minDuration: Infinity,
          maxDuration: 0
        };
      }
      
      const op = summary[metric.operationName];
      op.count++;
      op.totalDuration += metric.duration;
      op.minDuration = Math.min(op.minDuration, metric.duration);
      op.maxDuration = Math.max(op.maxDuration, metric.duration);
    }
    
    // Calculate averages
    for (const op of Object.values(summary)) {
      (op as any).averageDuration = (op as any).totalDuration / (op as any).count;
    }
    
    return summary;
  }

  /**
   * Get debug session information
   */
  getSession(): DebugSession {
    return {
      ...this.session,
      totalLogs: this.logs.length,
      errorCount: this.logs.filter(log => log.level >= LogLevel.ERROR).length,
      warningCount: this.logs.filter(log => log.level === LogLevel.WARN).length
    };
  }

  /**
   * Export logs for debugging
   */
  exportLogs(format: 'json' | 'csv' | 'text' = 'json'): string {
    switch (format) {
      case 'json':
        return JSON.stringify({
          session: this.getSession(),
          logs: this.logs,
          performanceMetrics: this.performanceMetrics
        }, null, 2);
      
      case 'csv':
        return this.exportLogsAsCsv();
      
      case 'text':
        return this.exportLogsAsText();
      
      default:
        throw new Error(`Unsupported export format: ${format}`);
    }
  }

  /**
   * Clear all logs and metrics
   */
  clear(): void {
    this.logs = [];
    this.performanceMetrics = [];
    this.activeTimers.clear();
    this.info('DebugLogger', 'clear', 'Cleared all logs and metrics');
  }

  /**
   * Update debug configuration
   */
  updateConfig(config: Partial<DebugConfig>): void {
    const oldEnabled = this.config.enabled;
    this.config = { ...this.config, ...config };
    
    // Set up or tear down global error handlers based on enabled state
    if (this.config.enabled && !oldEnabled) {
      this.setupGlobalErrorHandlers();
    }
    
    this.info('DebugLogger', 'updateConfig', 'Debug configuration updated', config);
  }

  /**
   * Get current configuration
   */
  getConfig(): DebugConfig {
    return { ...this.config };
  }

  // Private methods

  private log(
    level: LogLevel,
    component: string,
    operation: string,
    message: string,
    data?: any,
    error?: Error
  ): void {
    if (!this.config.enabled || level < this.config.logLevel) {
      return;
    }

    const entry: LogEntry = {
      id: this.generateLogId(),
      timestamp: new Date(),
      level,
      component,
      operation,
      message,
      data,
      error,
      sessionId: this.session.sessionId
    };

    // Add stack trace if enabled and error is present
    if (this.config.includeStackTrace && error) {
      entry.stackTrace = error.stack;
    }

    this.logs.push(entry);

    // Maintain log size limit
    if (this.logs.length > this.config.maxLogEntries) {
      this.logs.shift();
    }

    // Console logging
    if (this.config.logToConsole) {
      this.logToConsole(entry);
    }

    // File logging (would need to be implemented with file system access)
    if (this.config.logToFile) {
      this.logToFile(entry);
    }
  }

  private logToConsole(entry: LogEntry): void {
    const timestamp = entry.timestamp.toISOString();
    const prefix = `[${timestamp}] [${LogLevel[entry.level]}] [${entry.component}:${entry.operation}]`;
    const message = `${prefix} ${entry.message}`;

    switch (entry.level) {
      case LogLevel.TRACE:
      case LogLevel.DEBUG:
        console.debug(message, entry.data);
        break;
      case LogLevel.INFO:
        console.info(message, entry.data);
        break;
      case LogLevel.WARN:
        console.warn(message, entry.data);
        break;
      case LogLevel.ERROR:
      case LogLevel.FATAL:
        console.error(message, entry.error || entry.data);
        if (entry.stackTrace) {
          console.error(entry.stackTrace);
        }
        break;
    }
  }

  private logToFile(entry: LogEntry): void {
    // File logging would be implemented here
    // This would require file system access which may not be available in all contexts
    console.log('File logging not implemented yet');
  }

  private exportLogsAsCsv(): string {
    const headers = ['Timestamp', 'Level', 'Component', 'Operation', 'Message', 'Error'];
    const rows = [headers.join(',')];

    for (const log of this.logs) {
      const row = [
        log.timestamp.toISOString(),
        LogLevel[log.level],
        log.component,
        log.operation,
        `"${log.message.replace(/"/g, '""')}"`,
        log.error ? `"${log.error.message.replace(/"/g, '""')}"` : ''
      ];
      rows.push(row.join(','));
    }

    return rows.join('\n');
  }

  private exportLogsAsText(): string {
    const lines = [];
    lines.push(`Debug Session: ${this.session.sessionId}`);
    lines.push(`Start Time: ${this.session.startTime.toISOString()}`);
    lines.push(`Plugin Version: ${this.session.pluginVersion}`);
    lines.push(`Total Logs: ${this.logs.length}`);
    lines.push('');

    for (const log of this.logs) {
      const timestamp = log.timestamp.toISOString();
      const level = LogLevel[log.level].padEnd(5);
      const component = log.component.padEnd(15);
      
      lines.push(`${timestamp} [${level}] [${component}:${log.operation}] ${log.message}`);
      
      if (log.data) {
        lines.push(`  Data: ${JSON.stringify(log.data)}`);
      }
      
      if (log.error) {
        lines.push(`  Error: ${log.error.message}`);
        if (log.stackTrace) {
          lines.push(`  Stack: ${log.stackTrace}`);
        }
      }
      
      lines.push('');
    }

    return lines.join('\n');
  }

  private createSession(): DebugSession {
    return {
      sessionId: this.generateSessionId(),
      startTime: new Date(),
      userAgent: navigator.userAgent,
      pluginVersion: '1.0.0', // This should come from plugin manifest
      obsidianVersion: (window as any).app?.vault?.adapter?.version || 'unknown',
      totalLogs: 0,
      errorCount: 0,
      warningCount: 0
    };
  }

  private setupGlobalErrorHandlers(): void {
    // Capture unhandled promise rejections
    window.addEventListener('unhandledrejection', (event) => {
      this.error(
        'GlobalHandler',
        'unhandledRejection',
        'Unhandled promise rejection',
        event.reason,
        { reason: event.reason }
      );
    });

    // Capture global errors
    window.addEventListener('error', (event) => {
      this.error(
        'GlobalHandler',
        'globalError',
        'Global error occurred',
        event.error,
        {
          filename: event.filename,
          lineno: event.lineno,
          colno: event.colno,
          message: event.message
        }
      );
    });
  }

  private generateLogId(): string {
    return `log_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
  }

  private generateSessionId(): string {
    return `session_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
  }
}

// Global debug logger instance
export const globalDebugLogger = new DebugLogger();

// Utility functions for easy logging
export const logger = {
  trace: (component: string, operation: string, message: string, data?: any) =>
    globalDebugLogger.trace(component, operation, message, data),
  
  debug: (component: string, operation: string, message: string, data?: any) =>
    globalDebugLogger.debug(component, operation, message, data),
  
  info: (component: string, operation: string, message: string, data?: any) =>
    globalDebugLogger.info(component, operation, message, data),
  
  warn: (component: string, operation: string, message: string, data?: any) =>
    globalDebugLogger.warn(component, operation, message, data),
  
  error: (component: string, operation: string, message: string, error?: Error, data?: any) =>
    globalDebugLogger.error(component, operation, message, error, data),
  
  fatal: (component: string, operation: string, message: string, error?: Error, data?: any) =>
    globalDebugLogger.fatal(component, operation, message, error, data)
};

// Performance timing utilities
export const perf = {
  start: (operationName: string) => globalDebugLogger.startTimer(operationName),
  end: (operationName: string, additionalMetrics?: Record<string, number>) =>
    globalDebugLogger.endTimer(operationName, additionalMetrics)
};

// Decorator for automatic performance timing
export function timed(operationName?: string) {
  return function (target: any, propertyName: string, descriptor: PropertyDescriptor) {
    const method = descriptor.value;
    const opName = operationName || `${target.constructor.name}.${propertyName}`;

    descriptor.value = async function (...args: any[]) {
      perf.start(opName);
      try {
        const result = await method.apply(this, args);
        perf.end(opName);
        return result;
      } catch (error) {
        perf.end(opName);
        throw error;
      }
    };
  };
}

// Decorator for automatic logging
export function logged(component?: string) {
  return function (target: any, propertyName: string, descriptor: PropertyDescriptor) {
    const method = descriptor.value;
    const comp = component || target.constructor.name;

    descriptor.value = async function (...args: any[]) {
      logger.debug(comp, propertyName, `Starting ${propertyName}`, { args });
      try {
        const result = await method.apply(this, args);
        logger.debug(comp, propertyName, `Completed ${propertyName}`, { result });
        return result;
      } catch (error) {
        logger.error(comp, propertyName, `Failed ${propertyName}`, error, { args });
        throw error;
      }
    };
  };
}