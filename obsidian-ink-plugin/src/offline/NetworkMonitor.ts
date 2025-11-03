/**
 * Network status monitoring and connectivity management
 * Provides advanced network detection and quality assessment
 */

import { logger } from '../errors/DebugLogger';

// Network status
export enum NetworkStatus {
  ONLINE = 'online',
  OFFLINE = 'offline',
  SLOW = 'slow',
  UNSTABLE = 'unstable',
  UNKNOWN = 'unknown'
}

// Connection quality metrics
export interface ConnectionQuality {
  status: NetworkStatus;
  latency: number; // in milliseconds
  bandwidth: number; // estimated in Mbps
  stability: number; // 0-1 score
  lastChecked: Date;
}

// Network event
export interface NetworkEvent {
  type: 'status_change' | 'quality_change' | 'connection_test';
  timestamp: Date;
  previousStatus?: NetworkStatus;
  currentStatus: NetworkStatus;
  quality?: ConnectionQuality;
}

// Network monitor configuration
export interface NetworkMonitorConfig {
  enabled: boolean;
  checkInterval: number; // milliseconds
  timeoutDuration: number; // milliseconds
  testEndpoints: string[];
  qualityThresholds: {
    slowLatency: number; // ms
    unstableLatency: number; // ms
    minBandwidth: number; // Mbps
  };
  maxRetries: number;
  enableQualityMonitoring: boolean;
}

// Network test result
export interface NetworkTestResult {
  success: boolean;
  latency: number;
  timestamp: Date;
  endpoint: string;
  error?: string;
}

// Network listener
export type NetworkStatusListener = (event: NetworkEvent) => void;

// Default configuration
const DEFAULT_NETWORK_CONFIG: NetworkMonitorConfig = {
  enabled: true,
  checkInterval: 30000, // 30 seconds
  timeoutDuration: 5000, // 5 seconds
  testEndpoints: [
    'https://www.google.com/favicon.ico',
    'https://httpbin.org/status/200',
    'https://jsonplaceholder.typicode.com/posts/1'
  ],
  qualityThresholds: {
    slowLatency: 1000, // 1 second
    unstableLatency: 2000, // 2 seconds
    minBandwidth: 0.5 // 0.5 Mbps
  },
  maxRetries: 3,
  enableQualityMonitoring: true
};

export class NetworkMonitor {
  private config: NetworkMonitorConfig;
  private currentStatus: NetworkStatus = NetworkStatus.UNKNOWN;
  private currentQuality: ConnectionQuality | null = null;
  private listeners: NetworkStatusListener[] = [];
  private monitorTimer: NodeJS.Timeout | null = null;
  private testHistory: NetworkTestResult[] = [];
  private maxHistorySize = 100;

  constructor(config: Partial<NetworkMonitorConfig> = {}) {
    this.config = { ...DEFAULT_NETWORK_CONFIG, ...config };
    
    if (this.config.enabled) {
      this.initialize();
    }
  }

  /**
   * Initialize network monitoring
   */
  private initialize(): void {
    // Set initial status based on navigator.onLine
    this.currentStatus = navigator.onLine ? NetworkStatus.ONLINE : NetworkStatus.OFFLINE;

    // Set up browser event listeners
    window.addEventListener('online', this.handleBrowserOnline);
    window.addEventListener('offline', this.handleBrowserOffline);

    // Start periodic monitoring
    this.startMonitoring();

    // Perform initial connectivity test
    this.performConnectivityTest();

    logger.info('NetworkMonitor', 'initialize', 'Network monitoring initialized', {
      config: this.config,
      initialStatus: this.currentStatus
    });
  }

  /**
   * Get current network status
   */
  getStatus(): NetworkStatus {
    return this.currentStatus;
  }

  /**
   * Get current connection quality
   */
  getQuality(): ConnectionQuality | null {
    return this.currentQuality;
  }

  /**
   * Check if currently online
   */
  isOnline(): boolean {
    return this.currentStatus === NetworkStatus.ONLINE || 
           this.currentStatus === NetworkStatus.SLOW;
  }

  /**
   * Check if connection is stable
   */
  isStable(): boolean {
    return this.currentStatus === NetworkStatus.ONLINE;
  }

  /**
   * Perform immediate connectivity test
   */
  async testConnectivity(): Promise<ConnectionQuality> {
    logger.debug('NetworkMonitor', 'testConnectivity', 'Starting connectivity test');

    const testResults: NetworkTestResult[] = [];
    
    // Test multiple endpoints
    for (const endpoint of this.config.testEndpoints) {
      try {
        const result = await this.testEndpoint(endpoint);
        testResults.push(result);
        
        // Add to history
        this.addToHistory(result);
        
      } catch (error) {
        logger.warn('NetworkMonitor', 'testConnectivity', 'Endpoint test failed', error, {
          endpoint
        });
        
        testResults.push({
          success: false,
          latency: -1,
          timestamp: new Date(),
          endpoint,
          error: error instanceof Error ? error.message : String(error)
        });
      }
    }

    // Analyze results
    const quality = this.analyzeTestResults(testResults);
    this.updateQuality(quality);

    return quality;
  }

  /**
   * Add network status listener
   */
  addListener(listener: NetworkStatusListener): void {
    this.listeners.push(listener);
  }

  /**
   * Remove network status listener
   */
  removeListener(listener: NetworkStatusListener): void {
    const index = this.listeners.indexOf(listener);
    if (index > -1) {
      this.listeners.splice(index, 1);
    }
  }

  /**
   * Get network test history
   */
  getTestHistory(): NetworkTestResult[] {
    return [...this.testHistory];
  }

  /**
   * Get network statistics
   */
  getStatistics(): {
    averageLatency: number;
    successRate: number;
    totalTests: number;
    recentTests: NetworkTestResult[];
  } {
    const recentTests = this.testHistory.slice(-20); // Last 20 tests
    const successfulTests = recentTests.filter(test => test.success);
    
    const averageLatency = successfulTests.length > 0
      ? successfulTests.reduce((sum, test) => sum + test.latency, 0) / successfulTests.length
      : 0;
    
    const successRate = recentTests.length > 0
      ? successfulTests.length / recentTests.length
      : 0;

    return {
      averageLatency,
      successRate,
      totalTests: this.testHistory.length,
      recentTests
    };
  }

  /**
   * Update configuration
   */
  updateConfig(config: Partial<NetworkMonitorConfig>): void {
    const wasEnabled = this.config.enabled;
    this.config = { ...this.config, ...config };

    if (this.config.enabled && !wasEnabled) {
      this.initialize();
    } else if (!this.config.enabled && wasEnabled) {
      this.cleanup();
    } else if (this.config.enabled) {
      // Restart monitoring with new config
      this.stopMonitoring();
      this.startMonitoring();
    }

    logger.info('NetworkMonitor', 'updateConfig', 'Configuration updated', config);
  }

  /**
   * Cleanup resources
   */
  cleanup(): void {
    this.stopMonitoring();
    
    window.removeEventListener('online', this.handleBrowserOnline);
    window.removeEventListener('offline', this.handleBrowserOffline);
    
    this.listeners = [];
    
    logger.info('NetworkMonitor', 'cleanup', 'Network monitor cleaned up');
  }

  // Private methods

  private startMonitoring(): void {
    if (this.monitorTimer) {
      clearInterval(this.monitorTimer);
    }

    this.monitorTimer = setInterval(() => {
      this.performConnectivityTest();
    }, this.config.checkInterval);
  }

  private stopMonitoring(): void {
    if (this.monitorTimer) {
      clearInterval(this.monitorTimer);
      this.monitorTimer = null;
    }
  }

  private async performConnectivityTest(): Promise<void> {
    try {
      await this.testConnectivity();
    } catch (error) {
      logger.error('NetworkMonitor', 'performConnectivityTest', 'Connectivity test failed', error);
    }
  }

  private async testEndpoint(endpoint: string): Promise<NetworkTestResult> {
    const startTime = performance.now();
    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), this.config.timeoutDuration);

    try {
      const response = await fetch(endpoint, {
        method: 'HEAD',
        mode: 'no-cors',
        signal: controller.signal,
        cache: 'no-cache'
      });

      const latency = performance.now() - startTime;
      clearTimeout(timeoutId);

      return {
        success: true,
        latency,
        timestamp: new Date(),
        endpoint
      };

    } catch (error) {
      clearTimeout(timeoutId);
      
      const latency = performance.now() - startTime;
      
      return {
        success: false,
        latency,
        timestamp: new Date(),
        endpoint,
        error: error instanceof Error ? error.message : String(error)
      };
    }
  }

  private analyzeTestResults(results: NetworkTestResult[]): ConnectionQuality {
    const successfulResults = results.filter(result => result.success);
    const successRate = results.length > 0 ? successfulResults.length / results.length : 0;

    if (successRate === 0) {
      return {
        status: NetworkStatus.OFFLINE,
        latency: -1,
        bandwidth: 0,
        stability: 0,
        lastChecked: new Date()
      };
    }

    // Calculate average latency from successful tests
    const averageLatency = successfulResults.length > 0
      ? successfulResults.reduce((sum, result) => sum + result.latency, 0) / successfulResults.length
      : -1;

    // Estimate bandwidth (very rough estimation)
    const estimatedBandwidth = this.estimateBandwidth(averageLatency);

    // Calculate stability score
    const stability = this.calculateStability(results);

    // Determine status based on metrics
    let status: NetworkStatus;
    if (successRate < 0.5) {
      status = NetworkStatus.UNSTABLE;
    } else if (averageLatency > this.config.qualityThresholds.unstableLatency) {
      status = NetworkStatus.UNSTABLE;
    } else if (averageLatency > this.config.qualityThresholds.slowLatency) {
      status = NetworkStatus.SLOW;
    } else {
      status = NetworkStatus.ONLINE;
    }

    return {
      status,
      latency: averageLatency,
      bandwidth: estimatedBandwidth,
      stability,
      lastChecked: new Date()
    };
  }

  private estimateBandwidth(latency: number): number {
    // Very rough bandwidth estimation based on latency
    // This is a simplified heuristic and not accurate
    if (latency < 100) return 10; // Fast connection
    if (latency < 300) return 5;  // Good connection
    if (latency < 1000) return 2; // Slow connection
    return 0.5; // Very slow connection
  }

  private calculateStability(results: NetworkTestResult[]): number {
    if (results.length < 2) return 1;

    const successfulResults = results.filter(r => r.success);
    if (successfulResults.length === 0) return 0;

    // Calculate variance in latency
    const latencies = successfulResults.map(r => r.latency);
    const mean = latencies.reduce((sum, lat) => sum + lat, 0) / latencies.length;
    const variance = latencies.reduce((sum, lat) => sum + Math.pow(lat - mean, 2), 0) / latencies.length;
    const standardDeviation = Math.sqrt(variance);

    // Convert to stability score (0-1, where 1 is most stable)
    const maxAcceptableDeviation = 500; // ms
    const stability = Math.max(0, 1 - (standardDeviation / maxAcceptableDeviation));

    return Math.min(1, stability);
  }

  private updateStatus(newStatus: NetworkStatus): void {
    if (this.currentStatus !== newStatus) {
      const previousStatus = this.currentStatus;
      this.currentStatus = newStatus;

      const event: NetworkEvent = {
        type: 'status_change',
        timestamp: new Date(),
        previousStatus,
        currentStatus: newStatus,
        quality: this.currentQuality
      };

      this.notifyListeners(event);

      logger.info('NetworkMonitor', 'updateStatus', 'Network status changed', {
        from: previousStatus,
        to: newStatus
      });
    }
  }

  private updateQuality(quality: ConnectionQuality): void {
    const previousStatus = this.currentStatus;
    this.currentQuality = quality;
    
    // Update status based on quality
    this.updateStatus(quality.status);

    // Notify quality change if status didn't change but quality metrics did
    if (previousStatus === quality.status) {
      const event: NetworkEvent = {
        type: 'quality_change',
        timestamp: new Date(),
        currentStatus: quality.status,
        quality
      };

      this.notifyListeners(event);
    }
  }

  private notifyListeners(event: NetworkEvent): void {
    this.listeners.forEach(listener => {
      try {
        listener(event);
      } catch (error) {
        logger.error('NetworkMonitor', 'notifyListeners', 'Listener error', error);
      }
    });
  }

  private addToHistory(result: NetworkTestResult): void {
    this.testHistory.push(result);
    
    // Maintain history size limit
    if (this.testHistory.length > this.maxHistorySize) {
      this.testHistory.shift();
    }
  }

  private handleBrowserOnline = (): void => {
    logger.debug('NetworkMonitor', 'handleBrowserOnline', 'Browser online event');
    
    // Don't immediately trust browser event, perform test
    this.performConnectivityTest();
  };

  private handleBrowserOffline = (): void => {
    logger.debug('NetworkMonitor', 'handleBrowserOffline', 'Browser offline event');
    
    this.updateStatus(NetworkStatus.OFFLINE);
    this.currentQuality = {
      status: NetworkStatus.OFFLINE,
      latency: -1,
      bandwidth: 0,
      stability: 0,
      lastChecked: new Date()
    };
  };
}

// Global network monitor instance
export const globalNetworkMonitor = new NetworkMonitor();