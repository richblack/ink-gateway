/**
 * Tests for the main plugin class
 */

import { jest, describe, it, expect, beforeEach, afterEach } from '@jest/globals';

// Use fake timers
jest.useFakeTimers();
import './setup';
import ObsidianInkPlugin from '../src/main';
import { DEFAULT_SETTINGS } from '../src/types';

// Mock the Obsidian module
jest.mock('obsidian', () => require('./setup').mockObsidian, { virtual: true });

describe('ObsidianInkPlugin', () => {
  let plugin: ObsidianInkPlugin;
  let mockApp: any;

  beforeEach(() => {
    mockApp = (global as any).mockApp;
    plugin = new ObsidianInkPlugin(mockApp, {
      id: 'obsidian-ink-plugin',
      name: 'Ink Gateway Plugin',
      version: '1.0.0',
      minAppVersion: '0.15.0',
      description: 'Test plugin',
      author: 'Test',
      authorUrl: '',
      isDesktopOnly: false
    });
  });

  afterEach(() => {
    jest.clearAllMocks();
    // Clean up any timers
    if (plugin) {
      plugin.onunload();
    }
    jest.clearAllTimers();
  });

  describe('Plugin Lifecycle', () => {
    it('should load successfully', async () => {
      await expect(plugin.onload()).resolves.not.toThrow();
      expect(plugin.settings).toEqual(DEFAULT_SETTINGS);
    });

    it('should unload successfully', () => {
      expect(() => plugin.onunload()).not.toThrow();
    });

    it('should initialize with default settings', async () => {
      await plugin.onload();
      expect(plugin.settings).toEqual(DEFAULT_SETTINGS);
    });
  });

  describe('Settings Management', () => {
    it('should load settings', async () => {
      const mockSettings = { ...DEFAULT_SETTINGS, apiKey: 'test-key' };
      jest.spyOn(plugin, 'loadData').mockResolvedValue(mockSettings);
      
      await plugin.loadSettings();
      expect(plugin.settings.apiKey).toBe('test-key');
    });

    it('should save settings', async () => {
      jest.spyOn(plugin, 'saveData').mockResolvedValue();
      plugin.settings = { ...DEFAULT_SETTINGS, apiKey: 'new-key' };
      
      await plugin.saveSettings();
      expect(plugin.saveData).toHaveBeenCalledWith(plugin.settings);
    });
  });

  describe('Component Initialization', () => {
    it('should initialize all components on load', async () => {
      await plugin.onload();
      
      expect(plugin.logger).toBeDefined();
      expect(plugin.eventManager).toBeDefined();
      expect(plugin.cacheManager).toBeDefined();
      expect(plugin.memoryManager).toBeDefined();
      expect(plugin.offlineManager).toBeDefined();
    });

    it('should create logger with correct methods', async () => {
      await plugin.onload();
      
      expect(plugin.logger.debug).toBeDefined();
      expect(plugin.logger.info).toBeDefined();
      expect(plugin.logger.warn).toBeDefined();
      expect(plugin.logger.error).toBeDefined();
    });

    it('should create event manager with correct methods', async () => {
      await plugin.onload();
      
      expect(plugin.eventManager.on).toBeDefined();
      expect(plugin.eventManager.off).toBeDefined();
      expect(plugin.eventManager.emit).toBeDefined();
    });

    it('should create cache manager with correct methods', async () => {
      await plugin.onload();
      
      expect(plugin.cacheManager.get).toBeDefined();
      expect(plugin.cacheManager.set).toBeDefined();
      expect(plugin.cacheManager.delete).toBeDefined();
      expect(plugin.cacheManager.clear).toBeDefined();
      expect(plugin.cacheManager.size).toBeDefined();
    });
  });

  describe('Cache Manager', () => {
    beforeEach(async () => {
      await plugin.onload();
    });

    it('should store and retrieve values', () => {
      plugin.cacheManager.set('test-key', 'test-value');
      expect(plugin.cacheManager.get('test-key')).toBe('test-value');
    });

    it('should return null for non-existent keys', () => {
      expect(plugin.cacheManager.get('non-existent')).toBeNull();
    });

    it('should delete values', () => {
      plugin.cacheManager.set('test-key', 'test-value');
      plugin.cacheManager.delete('test-key');
      expect(plugin.cacheManager.get('test-key')).toBeNull();
    });

    it('should clear all values', () => {
      plugin.cacheManager.set('key1', 'value1');
      plugin.cacheManager.set('key2', 'value2');
      plugin.cacheManager.clear();
      expect(plugin.cacheManager.size()).toBe(0);
    });

    it('should handle TTL expiration', () => {
      plugin.cacheManager.set('ttl-key', 'ttl-value', 10); // 10ms TTL
      
      // Fast-forward time
      jest.advanceTimersByTime(20);
      
      expect(plugin.cacheManager.get('ttl-key')).toBeNull();
    });
  });

  describe('Event Manager', () => {
    beforeEach(async () => {
      await plugin.onload();
    });

    it('should register and emit events', () => {
      const callback = jest.fn();
      plugin.eventManager.on('test-event', callback);
      plugin.eventManager.emit('test-event', 'test-data');
      
      expect(callback).toHaveBeenCalledWith('test-data');
    });

    it('should remove event listeners', () => {
      const callback = jest.fn();
      plugin.eventManager.on('test-event', callback);
      plugin.eventManager.off('test-event', callback);
      plugin.eventManager.emit('test-event', 'test-data');
      
      expect(callback).not.toHaveBeenCalled();
    });

    it('should handle multiple listeners for same event', () => {
      const callback1 = jest.fn();
      const callback2 = jest.fn();
      
      plugin.eventManager.on('test-event', callback1);
      plugin.eventManager.on('test-event', callback2);
      plugin.eventManager.emit('test-event', 'test-data');
      
      expect(callback1).toHaveBeenCalledWith('test-data');
      expect(callback2).toHaveBeenCalledWith('test-data');
    });
  });

  describe('Memory Manager', () => {
    beforeEach(async () => {
      await plugin.onload();
    });

    it('should monitor memory usage', () => {
      const stats = plugin.memoryManager.monitorMemoryUsage();
      
      expect(stats).toHaveProperty('totalMemory');
      expect(stats).toHaveProperty('usedMemory');
      expect(stats).toHaveProperty('cacheSize');
      expect(stats).toHaveProperty('pendingOperations');
    });

    it('should cleanup cache', () => {
      plugin.cacheManager.set('test', 'value');
      plugin.memoryManager.cleanupCache();
      
      expect(plugin.cacheManager.size()).toBe(0);
    });
  });

  describe('Offline Manager', () => {
    beforeEach(async () => {
      await plugin.onload();
    });

    it('should detect online status', () => {
      expect(plugin.offlineManager.isOnline()).toBe(true);
    });

    it('should detect offline status', () => {
      Object.defineProperty(navigator, 'onLine', { value: false });
      expect(plugin.offlineManager.isOnline()).toBe(false);
      Object.defineProperty(navigator, 'onLine', { value: true }); // Reset
    });
  });

  describe('Error Handling', () => {
    it('should handle errors gracefully', () => {
      // Basic error handling test
      expect(() => plugin.onunload()).not.toThrow();
    });
  });
});