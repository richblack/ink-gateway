/**
 * Mock implementation of Obsidian API for testing
 */

export class Plugin {
  app: any;
  manifest: any;
  
  constructor() {
    this.app = mockApp;
  }
  
  addCommand() {}
  addRibbonIcon() { return {}; }
  addStatusBarItem() { return { setText: jest.fn() }; }
  registerEvent() {}
  loadData() { return Promise.resolve({}); }
  saveData() { return Promise.resolve(); }
  onload() {}
  onunload() {}
}

export class TFile {
  path: string;
  name: string;
  basename: string;
  extension: string;
  stat: { ctime: number; mtime: number; size: number };
  
  constructor(path: string, basename?: string) {
    this.path = path;
    this.name = path.split('/').pop() || '';
    this.basename = basename || this.name.replace(/\.[^/.]+$/, '');
    this.extension = this.name.split('.').pop() || '';
    this.stat = {
      ctime: Date.now(),
      mtime: Date.now(),
      size: 1000
    };
  }
}

export class Notice {
  constructor(message: string) {
    console.log(`Notice: ${message}`);
  }
}

export interface CachedMetadata {
  frontmatter?: Record<string, any>;
  tags?: Array<{ tag: string; position: any }>;
  links?: Array<{ link: string; position: any }>;
  embeds?: Array<{ link: string; position: any }>;
  headings?: Array<{ heading: string; level: number; position: any }>;
}

const mockApp = {
  vault: {
    on: jest.fn(),
    getFiles: jest.fn(() => []),
    read: jest.fn(() => Promise.resolve('')),
    modify: jest.fn(() => Promise.resolve()),
    create: jest.fn(() => Promise.resolve()),
    delete: jest.fn(() => Promise.resolve()),
    getAbstractFileByPath: jest.fn()
  },
  workspace: {
    getActiveFile: jest.fn(() => null),
    openLinkText: jest.fn()
  },
  metadataCache: {
    getFileCache: jest.fn(() => null)
  }
};

export { mockApp };