/**
 * Tests for NetworkMonitor class
 */

import { NetworkMonitor, NetworkStatus } from '../NetworkMonitor';

// Mock fetch
global.fetch = jest.fn();

// Mock navigator.onLine
Object.defineProperty(navigator, 'onLine', {
  writable: true,
  value: true,
});

// Mock performance.now
Object.defineProperty(performance, 'now', {
  writable: true,
  value: jest.fn(() => Date.now()),
});

describe('NetworkMonitor', () => {
  let networkMonitor: NetworkMonitor;
  let mockFetch: jest.MockedFunction<typeof fetch>;

  beforeEach(() => {
    jest.clearAllMocks();
    mockFetch = fetch as jest.MockedFunction<typeof fetch>;
    
    networkMonitor = new NetworkMonitor({
      enabled: true,
      checkInterval: 1000, // 1 second for tests
      timeoutDuration: 500,
      testEndpoints: ['https://test.com/ping']
    });
  });

  afterEach(() => {
    networkMonitor.cleanup();
  });

  describe('initialization', () => {
    it('should initialize with correct default status', () => {
      expect(networkMonitor.getStatus()).toBe(NetworkStatus.ONLINE);
      expect(networkMonitor.isOnline()).toBe(true);
    });

    it('should initialize as offline when navigator.onLine is false', () => {
      Object.defineProperty(navigator, 'onLine', { value: false });
      
      const monitor = new NetworkMonitor({ enabled: true });
      expect(monitor.getStatus()).toBe(NetworkStatus.OFFLINE);
      expect(monitor.isOnline()).toBe(false);
      
      monitor.cleanup();
    });
  });

  describe('connectivity testing', () => {
    it('should detect online status with successful requests', async () => {
      mockFetch.mockResolvedValue(new Response('', { status: 200 }));

      const quality = await networkMonitor.testConnectivity();

      expect(quality.status).toBe(NetworkStatus.ONLINE);
      expect(quality.latency).toBeGreaterThan(0);
      expect(quality.stability).toBeGreaterThan(0);
    });

    it('should detect offline status with failed requests', async () => {
      mockFetch.mockRejectedValue(new Error('Network error'));

      const quality = await networkMonitor.testConnectivity();

      expect(quality.status).toBe(NetworkStatus.OFFLINE);
      expect(quality.latency).toBe(-1);
      expect(quality.stability).toBe(0);
    });

    it('should detect slow connection with high latency', async () => {
      // Mock slow response
      mockFetch.mockImplementation(() => {
        return new Promise(resolve => {
          setTimeout(() => {
            resolve(new Response('', { status: 200 }));
          }, 1200); // 1.2 seconds
        });
      });

      const quality = await networkMonitor.testConnectivity();

      expect(quality.status).toBe(NetworkStatus.SLOW);
      expect(quality.latency).toBeGreaterThan(1000);
    });

    it('should handle mixed success/failure results', async () => {
      const monitor = new NetworkMonitor({
        enabled: true,
        testEndpoints: ['https://test1.com', 'https://test2.com', 'https://test3.com']
      });

      mockFetch
        .mockResolvedValueOnce(new Response('', { status: 200 }))
        .mockRejectedValueOnce(new Error('Network error'))
        .mockResolvedValueOnce(new Response('', { status: 200 }));

      const quality = await monitor.testConnectivity();

      // Should be online with 2/3 success rate
      expect(quality.status).toBe(NetworkStatus.ONLINE);
      expect(quality.stability).toBeLessThan(1); // Not perfect stability

      monitor.cleanup();
    });

    it('should timeout long requests', async () => {
      const monitor = new NetworkMonitor({
        enabled: true,
        timeoutDuration: 100 // Very short timeout
      });

      mockFetch.mockImplementation(() => {
        return new Promise(resolve => {
          setTimeout(() => {
            resolve(new Response('', { status: 200 }));
          }, 200); // Longer than timeout
        });
      });

      const quality = await monitor.testConnectivity();

      expect(quality.status).toBe(NetworkStatus.OFFLINE);

      monitor.cleanup();
    });
  });

  describe('event listeners', () => {
    it('should notify listeners of status changes', async () => {
      const listener = jest.fn();
      networkMonitor.addListener(listener);

      // Mock failed request to trigger status change
      mockFetch.mockRejectedValue(new Error('Network error'));

      await networkMonitor.testConnectivity();

      expect(listener).toHaveBeenCalledWith(
        expect.objectContaining({
          type: 'status_change',
          currentStatus: NetworkStatus.OFFLINE
        })
      );
    });

    it('should notify listeners of quality changes', async () => {
      const listener = jest.fn();
      networkMonitor.addListener(listener);

      // First test - fast connection
      mockFetch.mockResolvedValueOnce(new Response('', { status: 200 }));
      await networkMonitor.testConnectivity();

      // Second test - slow connection (same status but different quality)
      mockFetch.mockImplementation(() => {
        return new Promise(resolve => {
          setTimeout(() => {
            resolve(new Response('', { status: 200 }));
          }, 800);
        });
      });
      await networkMonitor.testConnectivity();

      // Should have been called for quality change
      expect(listener).toHaveBeenCalledWith(
        expect.objectContaining({
          type: 'quality_change'
        })
      );
    });

    it('should remove listeners correctly', async () => {
      const listener = jest.fn();
      networkMonitor.addListener(listener);
      networkMonitor.removeListener(listener);

      mockFetch.mockRejectedValue(new Error('Network error'));
      await networkMonitor.testConnectivity();

      expect(listener).not.toHaveBeenCalled();
    });
  });

  describe('browser events', () => {
    it('should handle browser online event', () => {
      const listener = jest.fn();
      networkMonitor.addListener(listener);

      // Mock successful connectivity test
      mockFetch.mockResolvedValue(new Response('', { status: 200 }));

      // Simulate browser online event
      const event = new Event('online');
      window.dispatchEvent(event);

      // Should trigger connectivity test
      expect(mockFetch).toHaveBeenCalled();
    });

    it('should handle browser offline event', () => {
      const listener = jest.fn();
      networkMonitor.addListener(listener);

      // Simulate browser offline event
      const event = new Event('offline');
      window.dispatchEvent(event);

      expect(networkMonitor.getStatus()).toBe(NetworkStatus.OFFLINE);
      expect(listener).toHaveBeenCalledWith(
        expect.objectContaining({
          type: 'status_change',
          currentStatus: NetworkStatus.OFFLINE
        })
      );
    });
  });

  describe('statistics', () => {
    it('should track test history', async () => {
      mockFetch.mockResolvedValue(new Response('', { status: 200 }));

      await networkMonitor.testConnectivity();
      await networkMonitor.testConnectivity();

      const history = networkMonitor.getTestHistory();
      expect(history.length).toBeGreaterThan(0);
      expect(history[0]).toHaveProperty('success');
      expect(history[0]).toHaveProperty('latency');
      expect(history[0]).toHaveProperty('timestamp');
    });

    it('should calculate statistics correctly', async () => {
      // Mock mixed results
      mockFetch
        .mockResolvedValueOnce(new Response('', { status: 200 }))
        .mockRejectedValueOnce(new Error('Network error'))
        .mockResolvedValueOnce(new Response('', { status: 200 }));

      await networkMonitor.testConnectivity();
      await networkMonitor.testConnectivity();
      await networkMonitor.testConnectivity();

      const stats = networkMonitor.getStatistics();
      expect(stats.totalTests).toBeGreaterThan(0);
      expect(stats.successRate).toBeGreaterThan(0);
      expect(stats.successRate).toBeLessThanOrEqual(1);
      expect(stats.averageLatency).toBeGreaterThanOrEqual(0);
    });

    it('should maintain history size limit', async () => {
      const monitor = new NetworkMonitor({
        enabled: true,
        testEndpoints: ['https://test.com']
      });

      // Set a small history size for testing
      monitor['maxHistorySize'] = 5;

      mockFetch.mockResolvedValue(new Response('', { status: 200 }));

      // Perform more tests than the limit
      for (let i = 0; i < 10; i++) {
        await monitor.testConnectivity();
      }

      const history = monitor.getTestHistory();
      expect(history.length).toBeLessThanOrEqual(5);

      monitor.cleanup();
    });
  });

  describe('configuration', () => {
    it('should update configuration', () => {
      const newConfig = {
        checkInterval: 5000,
        timeoutDuration: 1000
      };

      networkMonitor.updateConfig(newConfig);

      const config = networkMonitor['config'];
      expect(config.checkInterval).toBe(5000);
      expect(config.timeoutDuration).toBe(1000);
    });

    it('should enable/disable monitoring', () => {
      networkMonitor.updateConfig({ enabled: false });
      
      // Should not perform tests when disabled
      const spy = jest.spyOn(networkMonitor, 'testConnectivity');
      
      // Trigger what would normally cause a test
      const event = new Event('online');
      window.dispatchEvent(event);
      
      expect(spy).not.toHaveBeenCalled();
    });
  });

  describe('stability calculation', () => {
    it('should calculate stability based on latency variance', async () => {
      const monitor = new NetworkMonitor({
        enabled: true,
        testEndpoints: ['https://test.com']
      });

      // Mock consistent latency (stable)
      let callCount = 0;
      mockFetch.mockImplementation(() => {
        const latency = 100 + (callCount++ * 5); // Gradually increasing but consistent
        return new Promise(resolve => {
          setTimeout(() => {
            resolve(new Response('', { status: 200 }));
          }, latency);
        });
      });

      await monitor.testConnectivity();
      await monitor.testConnectivity();
      await monitor.testConnectivity();

      const quality = monitor.getQuality();
      expect(quality?.stability).toBeGreaterThan(0.5);

      monitor.cleanup();
    });

    it('should detect unstable connections', async () => {
      const monitor = new NetworkMonitor({
        enabled: true,
        testEndpoints: ['https://test.com']
      });

      // Mock highly variable latency (unstable)
      let callCount = 0;
      mockFetch.mockImplementation(() => {
        const latencies = [50, 2000, 100, 1800, 75]; // Highly variable
        const latency = latencies[callCount++ % latencies.length];
        return new Promise(resolve => {
          setTimeout(() => {
            resolve(new Response('', { status: 200 }));
          }, latency);
        });
      });

      for (let i = 0; i < 5; i++) {
        await monitor.testConnectivity();
      }

      const quality = monitor.getQuality();
      expect(quality?.status).toBe(NetworkStatus.UNSTABLE);

      monitor.cleanup();
    });
  });

  describe('error handling', () => {
    it('should handle listener errors gracefully', async () => {
      const faultyListener = jest.fn(() => {
        throw new Error('Listener error');
      });
      
      networkMonitor.addListener(faultyListener);

      mockFetch.mockRejectedValue(new Error('Network error'));

      // Should not throw despite listener error
      await expect(networkMonitor.testConnectivity()).resolves.toBeDefined();
    });

    it('should handle fetch errors gracefully', async () => {
      mockFetch.mockImplementation(() => {
        throw new Error('Fetch failed');
      });

      const quality = await networkMonitor.testConnectivity();

      expect(quality.status).toBe(NetworkStatus.OFFLINE);
      expect(quality.latency).toBe(-1);
    });
  });

  describe('cleanup', () => {
    it('should cleanup resources properly', () => {
      const removeEventListenerSpy = jest.spyOn(window, 'removeEventListener');
      
      networkMonitor.cleanup();

      expect(removeEventListenerSpy).toHaveBeenCalledWith('online', expect.any(Function));
      expect(removeEventListenerSpy).toHaveBeenCalledWith('offline', expect.any(Function));
    });
  });
});