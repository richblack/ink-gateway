/**
 * Test setup and configuration
 */

import { jest } from '@jest/globals';

// Mock Obsidian API
const mockObsidian = {
  Plugin: class MockPlugin {
    app: any;
    manifest: any;
    
    constructor() {
      this.app = mockApp;
    }
    
    addCommand() {}
    addRibbonIcon() {}
    addStatusBarItem() { return { setText: jest.fn() }; }
    registerEvent() {}
    loadData() { return Promise.resolve({}); }
    saveData() { return Promise.resolve(); }
  },
  
  TFile: class MockTFile {
    path: string;
    name: string;
    extension: string;
    
    constructor(path: string) {
      this.path = path;
      this.name = path.split('/').pop() || '';
      this.extension = this.name.split('.').pop() || '';
    }
  },
  
  Notice: class MockNotice {
    constructor(message: string) {
      console.log(`Notice: ${message}`);
    }
  }
};

const mockApp = {
  vault: {
    on: jest.fn(),
    getFiles: jest.fn(() => []),
    read: jest.fn(() => Promise.resolve('')),
    modify: jest.fn(() => Promise.resolve()),
    create: jest.fn(() => Promise.resolve()),
    delete: jest.fn(() => Promise.resolve())
  },
  workspace: {
    getActiveFile: jest.fn(() => null),
    openLinkText: jest.fn()
  }
};

// Make mocks available globally
(global as any).obsidian = mockObsidian;
(global as any).mockApp = mockApp;

// Mock navigator.onLine
Object.defineProperty(navigator, 'onLine', {
  writable: true,
  value: true
});

// Setup console for tests
global.console = {
  ...console,
  log: jest.fn(),
  debug: jest.fn(),
  info: jest.fn(),
  warn: jest.fn(),
  error: jest.fn()
};

export { mockObsidian, mockApp };