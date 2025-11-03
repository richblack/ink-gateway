/**
 * Main plugin class for Obsidian Ink Plugin
 */

import { Plugin, TFile, Notice, MarkdownView } from 'obsidian';
import { 
  SyncState,
  ErrorType,
  PluginError
} from './types';
import { PluginSettings, DEFAULT_SETTINGS } from './settings/PluginSettings';
import {
  IContentManager,
  ISearchManager,
  ITemplateManager,
  IAIManager,
  IInkGatewayClient,
  IOfflineManager,
  IMemoryManager,
  ICacheManager,
  IEventManager,
  ILogger
} from './interfaces';
import { InkGatewayClient } from './api/InkGatewayClient';
import { ImageUploadManager, IImageUploadManager } from './media/ImageUploadManager';
import { DragDropHandler } from './media/DragDropHandler';
import { ImageLibraryModal } from './media/ImageLibraryModal';
import { BatchProcessModal } from './media/BatchProcessModal';
import { InkPluginSettingsTab } from './settings/SettingsTab';
import { SettingsManager } from './settings/SettingsManager';
import { DebugLogger } from './errors/DebugLogger';

export default class ObsidianInkPlugin extends Plugin {
  settings!: PluginSettings;
  
  // Core managers (will be initialized in onload)
  contentManager!: IContentManager;
  searchManager!: ISearchManager;
  templateManager!: ITemplateManager;
  aiManager!: IAIManager;
  apiClient!: IInkGatewayClient;
  offlineManager!: IOfflineManager;
  memoryManager!: IMemoryManager;
  cacheManager!: ICacheManager;
  eventManager!: IEventManager;
  logger!: ILogger;
  
  // Media managers
  imageUploadManager!: IImageUploadManager;
  dragDropHandler!: DragDropHandler;  
  // Settings and logging
  settingsManager!: SettingsManager;
  logger!: DebugLogger;
  
  // Plugin state
  private syncState!: SyncState;
  private isInitialized: boolean = false;
  private syncInterval: NodeJS.Timeout | null = null;

  /**
   * Plugin lifecycle: Load
   */
  async onload() {
    console.log('Loading Obsidian Ink Plugin...');
    
    try {
      // Load settings
      await this.loadSettings();
      
      // Initialize core components
      await this.initializeComponents();
      
      // Setup event listeners
      this.setupEventListeners();
      
      // Setup commands
      this.setupCommands();
      
      // Setup ribbon icons
      this.setupRibbonIcons();
      
      // Setup status bar
      this.setupStatusBar();
      // Add settings tab
      this.addSettingTab(new InkPluginSettingsTab(
        this.app, 
        this, 
        this.settingsManager, 
        this.logger
      ));
      
      // Start auto-sync if enabled
      if (this.settings.autoSync) {
        this.startAutoSync();
      }
      
      // Initialize sync state
      this.initializeSyncState();
      
      this.isInitialized = true;
      console.log('Obsidian Ink Plugin loaded successfully');
      
      // Show welcome notice
      new Notice('Ink Gateway Plugin loaded successfully!');
      
    } catch (error) {
      console.error('Failed to load Obsidian Ink Plugin:', error);
      new Notice('Failed to load Ink Gateway Plugin. Check console for details.');
      throw new PluginError(
        ErrorType.API_ERROR,
        'PLUGIN_LOAD_FAILED',
        error,
        false
      );
    }
  }

  /**
   * Plugin lifecycle: Unload
   */
  onunload() {
    console.log('Unloading Obsidian Ink Plugin...');
    
    try {
      // Stop auto-sync
      this.stopAutoSync();
      
      // Cleanup drag and drop handler
      if (this.dragDropHandler) {
        this.dragDropHandler.cleanup();
      }
      
      // Clear sync interval
      if (this.syncInterval) {
        clearInterval(this.syncInterval);
      }
      
      // Cleanup managers
      this.cleanupComponents();
      
      // Clear event listeners
      this.clearEventListeners();
      
      // Final memory cleanup
      if (this.memoryManager) {
        this.memoryManager.cleanupCache();
      }
      
      console.log('Obsidian Ink Plugin unloaded successfully');
      
    } catch (error) {
      console.error('Error during plugin unload:', error);
    }
  }

  /**
   * Load plugin settings
   */
  async loadSettings() {
    this.settings = Object.assign({}, DEFAULT_SETTINGS, await this.loadData());
  }

  /**
   * Save plugin settings
   */
  async saveSettings() {
    await this.saveData(this.settings);
  }

  /**
   * Initialize core components
   * Note: Actual implementations will be created in subsequent tasks
   */
  private async initializeComponents() {
    // TODO: Initialize actual implementations in subsequent tasks
    // For now, we'll create placeholder implementations
    
    // Initialize logger first
    this.logger = this.createLogger();
    
    // Initialize settings manager
    this.settingsManager = this.createSettingsManager();
    
    // Initialize event manager
    this.eventManager = this.createEventManager();
    
    // Initialize cache manager
    this.cacheManager = this.createCacheManager();
    
    // Initialize memory manager
    this.memoryManager = this.createMemoryManager();
    
    // Initialize offline manager
    this.offlineManager = this.createOfflineManager();
    
    // Initialize API client
    this.apiClient = this.createAPIClient();
    
    // Initialize content manager
    this.contentManager = this.createContentManager();
    
    // Initialize search manager
    this.searchManager = this.createSearchManager();
    
    // Initialize template manager
    this.templateManager = this.createTemplateManager();
    
    // Initialize AI manager
    this.aiManager = this.createAIManager();
    
    // Initialize image upload manager
    this.imageUploadManager = this.createImageUploadManager();
    
    // Initialize drag and drop handler
    this.dragDropHandler = this.createDragDropHandler();
    
    this.logger.info('ObsidianInkPlugin', 'initializeComponents', 'All components initialized successfully');
  }

  /**
   * Setup event listeners
   */
  private setupEventListeners() {
    // File modification events
    this.registerEvent(
      this.app.vault.on('modify', (file) => {
        if (file instanceof TFile && file.extension === 'md') {
          this.handleFileModification(file);
        }
      })
    );

    // File creation events
    this.registerEvent(
      this.app.vault.on('create', (file) => {
        if (file instanceof TFile && file.extension === 'md') {
          this.handleFileCreation(file);
        }
      })
    );

    // File deletion events
    this.registerEvent(
      this.app.vault.on('delete', (file) => {
        if (file instanceof TFile && file.extension === 'md') {
          this.handleFileDeletion(file);
        }
      })
    );
  }

  /**
   * Setup plugin commands
   */
  private setupCommands() {
    // Open AI Chat command
    this.addCommand({
      id: 'open-ai-chat',
      name: 'Open AI Chat',
      callback: () => {
        this.aiManager.createChatView();
      }
    });

    // Open Search command
    this.addCommand({
      id: 'open-search',
      name: 'Open Semantic Search',
      callback: () => {
        this.searchManager.createSearchView();
      }
    });

    // Sync now command
    this.addCommand({
      id: 'sync-now',
      name: 'Sync with Ink Gateway',
      callback: async () => {
        await this.performManualSync();
      }
    });

    // Toggle auto-sync command
    this.addCommand({
      id: 'toggle-auto-sync',
      name: 'Toggle Auto Sync',
      callback: () => {
        this.toggleAutoSync();
      }
    });

    // Image upload commands
    this.addCommand({
      id: 'upload-image',
      name: 'Upload Image',
      callback: () => {
        this.openImageUploadDialog();
      }
    });

    this.addCommand({
      id: 'upload-from-clipboard',
      name: 'Upload Image from Clipboard',
      callback: async () => {
        await this.uploadFromClipboard();
      }
    });

    this.addCommand({
      id: 'open-image-library',
      name: 'Open Image Library',
      callback: () => {
        this.openImageLibrary();
      }
    });

    this.addCommand({
      id: 'batch-upload-images',
      name: 'Batch Upload Images',
      callback: () => {
        this.openBatchUploadDialog();
      }
    });
  }

  /**
   * Setup ribbon icons
   */
  private setupRibbonIcons() {
    // AI Chat ribbon icon
    this.addRibbonIcon('message-circle', 'Open AI Chat', () => {
      this.aiManager.createChatView();
    });

    // Search ribbon icon
    this.addRibbonIcon('search', 'Open Semantic Search', () => {
      this.searchManager.createSearchView();
    });

    // Sync ribbon icon
    this.addRibbonIcon('refresh-cw', 'Sync with Ink Gateway', async () => {
      await this.performManualSync();
    });

    // Image library ribbon icon
    this.addRibbonIcon('image', 'Open Image Library', () => {
      this.openImageLibrary();
    });

    // Upload image ribbon icon
    this.addRibbonIcon('upload', 'Upload Image', () => {
      this.openImageUploadDialog();
    });
  }

  /**
   * Setup status bar
   */
  private setupStatusBar() {
    const statusBarItem = this.addStatusBarItem();
    statusBarItem.setText('Ink Gateway: Ready');
    
    // Update status bar based on sync state
    this.eventManager.on('syncStateChanged', (state: SyncState) => {
      switch (state.syncStatus) {
        case 'idle':
          statusBarItem.setText('Ink Gateway: Ready');
          break;
        case 'syncing':
          statusBarItem.setText('Ink Gateway: Syncing...');
          break;
        case 'error':
          statusBarItem.setText('Ink Gateway: Error');
          break;
        case 'offline':
          statusBarItem.setText('Ink Gateway: Offline');
          break;
      }
    });
  }

  /**
   * Initialize sync state
   */
  private initializeSyncState() {
    this.syncState = {
      lastSyncTime: new Date(),
      pendingChanges: [],
      conflictResolution: {
        strategy: 'local',
        conflicts: []
      },
      syncStatus: 'idle'
    };
  }

  /**
   * Start auto-sync timer
   */
  private startAutoSync() {
    if (this.syncInterval) {
      clearInterval(this.syncInterval);
    }
    
    this.syncInterval = setInterval(async () => {
      if (this.offlineManager.isOnline() && this.syncState.pendingChanges.length > 0) {
        await this.performAutoSync();
      }
    }, this.settings.syncInterval);
  }

  /**
   * Stop auto-sync timer
   */
  private stopAutoSync() {
    if (this.syncInterval) {
      clearInterval(this.syncInterval);
      this.syncInterval = null;
    }
  }

  /**
   * Toggle auto-sync
   */
  private toggleAutoSync() {
    this.settings.autoSync = !this.settings.autoSync;
    this.saveSettings();
    
    if (this.settings.autoSync) {
      this.startAutoSync();
      new Notice('Auto-sync enabled');
    } else {
      this.stopAutoSync();
      new Notice('Auto-sync disabled');
    }
  }

  /**
   * Handle file modification
   */
  private async handleFileModification(file: TFile) {
    try {
      if (this.contentManager) {
        await this.contentManager.handleContentChange(file);
      }
    } catch (error) {
      this.logger.error('Error handling file modification:', error as Error);
    }
  }

  /**
   * Handle file creation
   */
  private async handleFileCreation(file: TFile) {
    try {
      if (this.contentManager) {
        await this.contentManager.handleContentChange(file);
      }
    } catch (error) {
      this.logger.error('Error handling file creation:', error as Error);
    }
  }

  /**
   * Handle file deletion
   */
  private async handleFileDeletion(file: TFile) {
    try {
      // TODO: Implement file deletion handling
      this.logger.info(`File deleted: ${file.path}`);
    } catch (error) {
      this.logger.error('Error handling file deletion:', error as Error);
    }
  }

  /**
   * Perform manual sync
   */
  private async performManualSync() {
    try {
      new Notice('Starting manual sync...');
      
      if (!this.offlineManager.isOnline()) {
        new Notice('Cannot sync: offline');
        return;
      }
      
      this.syncState.syncStatus = 'syncing';
      this.eventManager.emit('syncStateChanged', this.syncState);
      
      // TODO: Implement actual sync logic
      await new Promise(resolve => setTimeout(resolve, 1000)); // Placeholder
      
      this.syncState.syncStatus = 'idle';
      this.syncState.lastSyncTime = new Date();
      this.eventManager.emit('syncStateChanged', this.syncState);
      
      new Notice('Sync completed successfully');
      
    } catch (error) {
      this.syncState.syncStatus = 'error';
      this.eventManager.emit('syncStateChanged', this.syncState);
      this.logger.error('Manual sync failed:', error as Error);
      new Notice('Sync failed. Check console for details.');
    }
  }

  /**
   * Perform auto sync
   */
  private async performAutoSync() {
    try {
      this.syncState.syncStatus = 'syncing';
      this.eventManager.emit('syncStateChanged', this.syncState);
      
      // TODO: Implement actual auto-sync logic
      await new Promise(resolve => setTimeout(resolve, 500)); // Placeholder
      
      this.syncState.syncStatus = 'idle';
      this.syncState.lastSyncTime = new Date();
      this.eventManager.emit('syncStateChanged', this.syncState);
      
    } catch (error) {
      this.syncState.syncStatus = 'error';
      this.eventManager.emit('syncStateChanged', this.syncState);
      this.logger.error('Auto sync failed:', error as Error);
    }
  }

  /**
   * Cleanup components
   */
  private cleanupComponents() {
    // TODO: Implement cleanup logic for each component
    this.logger?.info('Components cleaned up');
  }

  /**
   * Clear event listeners
   */
  private clearEventListeners() {
    // Event listeners are automatically cleared by Obsidian when plugin unloads
    this.logger?.info('Event listeners cleared');
  }

  // Placeholder factory methods - will be implemented in subsequent tasks
  private createLogger(): ILogger {
    return {
      debug: (message: string, ...args: any[]) => console.debug(`[Ink Plugin] ${message}`, ...args),
      info: (message: string, ...args: any[]) => console.info(`[Ink Plugin] ${message}`, ...args),
      warn: (message: string, ...args: any[]) => console.warn(`[Ink Plugin] ${message}`, ...args),
      error: (message: string, error?: Error, ...args: any[]) => console.error(`[Ink Plugin] ${message}`, error, ...args)
    };
  }

  private createSettingsManager(): SettingsManager {
    return new SettingsManager(this, this.logger);
  }

  private createEventManager(): IEventManager {
    const events = new Map<string, Function[]>();
    return {
      on: (event: string, callback: Function) => {
        if (!events.has(event)) events.set(event, []);
        events.get(event)!.push(callback);
      },
      off: (event: string, callback: Function) => {
        const callbacks = events.get(event);
        if (callbacks) {
          const index = callbacks.indexOf(callback);
          if (index > -1) callbacks.splice(index, 1);
        }
      },
      emit: (event: string, ...args: any[]) => {
        const callbacks = events.get(event);
        if (callbacks) {
          callbacks.forEach(callback => callback(...args));
        }
      }
    };
  }

  private createCacheManager(): ICacheManager {
    const cache = new Map<string, { value: any; expires?: number }>();
    return {
      get: <T>(key: string): T | null => {
        const item = cache.get(key);
        if (!item) return null;
        if (item.expires && Date.now() > item.expires) {
          cache.delete(key);
          return null;
        }
        return item.value;
      },
      set: <T>(key: string, value: T, ttl?: number) => {
        const expires = ttl ? Date.now() + ttl : undefined;
        cache.set(key, { value, expires });
      },
      delete: (key: string) => cache.delete(key),
      clear: () => cache.clear(),
      size: () => cache.size
    };
  }

  private createMemoryManager(): IMemoryManager {
    return {
      cleanupCache: () => this.cacheManager.clear(),
      monitorMemoryUsage: () => ({
        totalMemory: 0,
        usedMemory: 0,
        cacheSize: this.cacheManager.size(),
        pendingOperations: this.syncState?.pendingChanges?.length || 0
      }),
      optimizePerformance: () => {
        // TODO: Implement performance optimization
      }
    };
  }

  private createOfflineManager(): IOfflineManager {
    return {
      isOnline: () => navigator.onLine,
      queueOperation: (operation) => {
        // TODO: Implement operation queuing
      },
      syncWhenOnline: async () => {
        // TODO: Implement sync when online
      },
      handleConflicts: async (conflicts) => {
        // TODO: Implement conflict handling
      }
    };
  }

  private createAPIClient(): IInkGatewayClient {
    return new InkGatewayClient(
      this.settings.inkGatewayUrl,
      this.settings.apiKey,
      {
        timeout: 30000,
        retryConfig: {
          maxRetries: 3,
          baseDelay: 1000,
          maxDelay: 10000,
          backoffFactor: 2,
          retryableStatusCodes: [408, 429, 500, 502, 503, 504]
        }
      }
    );
  }

  private createContentManager(): IContentManager {
    const { ContentManager } = require('./content/ContentManager');
    return new ContentManager(
      this.apiClient,
      this.logger,
      this.eventManager,
      this.cacheManager,
      this.offlineManager,
      this.app
    );
  }

  private createSearchManager(): ISearchManager {
    // TODO: Implement actual search manager in subsequent tasks
    return {} as ISearchManager;
  }

  private createTemplateManager(): ITemplateManager {
    // TODO: Implement actual template manager in subsequent tasks
    return {} as ITemplateManager;
  }

  private createAIManager(): IAIManager {
    // TODO: Implement actual AI manager in subsequent tasks
    return {} as IAIManager;
  }

  private createImageUploadManager(): IImageUploadManager {
    return new ImageUploadManager(this.apiClient);
  }

  private createDragDropHandler(): DragDropHandler {
    const handler = new DragDropHandler(this.imageUploadManager, {
      autoAnalyze: true,
      autoEmbed: true,
      storageType: this.settings.storageProvider === 'google_drive' ? 'google_drive' : 'local',
      insertMode: 'embed',
      showProgress: true
    });
    
    handler.initialize();
    return handler;
  }

  // Image-related command handlers
  private openImageUploadDialog(): void {
    const input = document.createElement('input');
    input.type = 'file';
    input.multiple = true;
    input.accept = 'image/*';
    
    input.addEventListener('change', async (event) => {
      const files = (event.target as HTMLInputElement).files;
      if (!files || files.length === 0) return;

      const activeView = this.app.workspace.getActiveViewOfType(MarkdownView);
      if (!activeView) {
        new Notice('Please open a markdown file to upload images');
        return;
      }

      for (const file of Array.from(files)) {
        try {
          new Notice(`Uploading ${file.name}...`);
          
          const response = await this.imageUploadManager.uploadImage({
            file,
            filename: file.name,
            pageId: activeView.file?.path.replace(/\.[^/.]+$/, ''),
            autoAnalyze: true,
            autoEmbed: true,
            storageType: this.settings.storageProvider === 'google_drive' ? 'google_drive' : 'local'
          });

          // Insert image into editor
          const editor = activeView.editor;
          const cursor = editor.getCursor();
          const imageMarkdown = `![${file.name}](${response.imageUrl})`;
          
          if (response.analysis?.description) {
            const fullMarkdown = `${imageMarkdown}\n<!-- AI Analysis: ${response.analysis.description} -->`;
            editor.replaceRange(fullMarkdown, cursor);
          } else {
            editor.replaceRange(imageMarkdown, cursor);
          }

          new Notice(`Successfully uploaded ${file.name}`);
        } catch (error) {
          console.error(`Failed to upload ${file.name}:`, error);
          new Notice(`Failed to upload ${file.name}: ${error}`);
        }
      }
    });

    input.click();
  }

  private async uploadFromClipboard(): Promise<void> {
    try {
      const activeView = this.app.workspace.getActiveViewOfType(MarkdownView);
      if (!activeView) {
        new Notice('Please open a markdown file to upload images');
        return;
      }

      const response = await this.imageUploadManager.uploadFromClipboard({
        pageId: activeView.file?.path.replace(/\.[^/.]+$/, ''),
        autoAnalyze: true,
        autoEmbed: true,
        storageType: this.settings.storageProvider === 'google_drive' ? 'google_drive' : 'local'
      });

      // Insert image into editor
      const editor = activeView.editor;
      const cursor = editor.getCursor();
      const imageMarkdown = `![Clipboard Image](${response.imageUrl})`;
      
      if (response.analysis?.description) {
        const fullMarkdown = `${imageMarkdown}\n<!-- AI Analysis: ${response.analysis.description} -->`;
        editor.replaceRange(fullMarkdown, cursor);
      } else {
        editor.replaceRange(imageMarkdown, cursor);
      }

      new Notice('Successfully uploaded image from clipboard');
    } catch (error) {
      console.error('Failed to upload from clipboard:', error);
      new Notice(`Failed to upload from clipboard: ${error}`);
    }
  }

  private openImageLibrary(): void {
    const modal = new ImageLibraryModal(this.app, this.apiClient);
    modal.open();
  }

  private openBatchUploadDialog(): void {
    const storageType = this.settings.storageProvider === 'google_drive' ? 'google_drive' : 'local';
    const modal = new BatchProcessModal(this.app, this.imageUploadManager, storageType);
    modal.open();
  }


}