/**
 * Content Manager for Obsidian Ink Plugin
 * Handles content parsing, synchronization, and management
 */

import { TFile, CachedMetadata } from 'obsidian';
import { 
  IContentManager, 
  IInkGatewayClient, 
  ILogger, 
  IEventManager,
  ICacheManager,
  IOfflineManager
} from '../interfaces';
import {
  ParsedContent,
  HierarchyNode,
  ContentMetadata,
  SyncResult,
  UnifiedChunk,
  Position,
  SyncError,
  PluginError,
  ErrorType,
  VirtualDocumentContext,
  VirtualDocument,
  DocumentChunksResult,
  PaginationOptions,
  ReconstructedDocument,
  DocumentMetadata,
  DocumentScope
} from '../types';
import { MarkdownParser, ParseOptions } from './MarkdownParser';
import { SyncManager, SyncManagerOptions } from './SyncManager';
import { MetadataManager, TagSyncOptions, MetadataProcessingOptions } from './MetadataManager';

export class ContentManager implements IContentManager {
  private apiClient: IInkGatewayClient;
  private logger: ILogger;
  private eventManager: IEventManager;
  private cacheManager: ICacheManager;
  private app: any; // Obsidian App instance
  private syncManager: SyncManager;
  private metadataManager: MetadataManager;
  
  // Processing state
  private processingFiles = new Set<string>();
  private lastProcessedContent = new Map<string, string>();
  
  constructor(
    apiClient: IInkGatewayClient,
    logger: ILogger,
    eventManager: IEventManager,
    cacheManager: ICacheManager,
    offlineManager: IOfflineManager,
    app: any,
    syncOptions?: Partial<SyncManagerOptions>,
    tagSyncOptions?: Partial<TagSyncOptions>,
    metadataOptions?: Partial<MetadataProcessingOptions>
  ) {
    this.apiClient = apiClient;
    this.logger = logger;
    this.eventManager = eventManager;
    this.cacheManager = cacheManager;
    this.app = app;
    
    // Initialize sync manager
    const defaultSyncOptions: SyncManagerOptions = {
      autoSyncEnabled: true,
      syncInterval: 5000,
      maxRetries: 3,
      batchSize: 10,
      conflictResolutionStrategy: 'local'
    };
    
    this.syncManager = new SyncManager(
      apiClient,
      logger,
      eventManager,
      cacheManager,
      offlineManager,
      { ...defaultSyncOptions, ...syncOptions }
    );
    
    // Initialize metadata manager
    const defaultTagSyncOptions: TagSyncOptions = {
      bidirectionalSync: true,
      autoCreateTags: true,
      excludePatterns: ['private/', 'temp/']
    };
    
    const defaultMetadataOptions: MetadataProcessingOptions = {
      syncFrontmatter: true,
      syncProperties: true,
      syncTags: true,
      preserveObsidianMetadata: true
    };
    
    this.metadataManager = new MetadataManager(
      apiClient,
      logger,
      eventManager,
      cacheManager,
      app,
      { ...defaultTagSyncOptions, ...tagSyncOptions },
      { ...defaultMetadataOptions, ...metadataOptions }
    );
    
    this.setupEventListeners();
  }

  /**
   * Parse markdown content into structured format
   */
  async parseContent(content: string, filePath: string): Promise<ParsedContent> {
    try {
      this.logger.debug(`Parsing content for file: ${filePath}`);
      
      // Check cache first
      const cacheKey = `parsed_${filePath}_${this.getContentHash(content)}`;
      const cached = this.cacheManager.get<ParsedContent>(cacheKey);
      if (cached) {
        this.logger.debug(`Using cached parsed content for: ${filePath}`);
        return cached;
      }
      
      // Parse options
      const options: ParseOptions = {
        trackPositions: true,
        parseHierarchy: true,
        extractMetadata: true,
        generateChunks: true
      };
      
      // Parse content
      const parsed = MarkdownParser.parseContent(content, filePath, options);
      
      // Validate parsed content
      if (!MarkdownParser.validateParsedContent(parsed)) {
        throw new PluginError(
          ErrorType.PARSING_ERROR,
          'INVALID_PARSED_CONTENT',
          { filePath },
          true
        );
      }
      
      // Cache the result
      this.cacheManager.set(cacheKey, parsed, 300000); // 5 minutes TTL
      
      this.logger.debug(`Successfully parsed content for: ${filePath}`, {
        chunks: parsed.chunks.length,
        hierarchyNodes: parsed.hierarchy.length,
        tags: parsed.metadata.tags.length
      });
      
      return parsed;
      
    } catch (error) {
      this.logger.error(`Failed to parse content for ${filePath}:`, error as Error);
      throw new PluginError(
        ErrorType.PARSING_ERROR,
        'CONTENT_PARSE_FAILED',
        { filePath, error: (error as Error).message },
        true
      );
    }
  }

  /**
   * Synchronize chunks to Ink Gateway using SyncManager
   */
  async syncToInkGateway(chunks: UnifiedChunk[]): Promise<SyncResult> {
    try {
      this.logger.debug(`Queueing ${chunks.length} chunks for sync`);
      
      // Queue chunks for synchronization
      for (const chunk of chunks) {
        // Determine if this is a new chunk or an update
        const existingChunk = this.cacheManager.get<UnifiedChunk>(`chunk_${chunk.chunkId}`);
        const changeType = existingChunk ? 'update' : 'create';
        
        this.syncManager.queueChange(changeType, chunk);
        
        // Emit content change event
        this.eventManager.emit(changeType === 'create' ? 'contentCreated' : 'contentChanged', chunk);
      }
      
      // Perform immediate sync
      return await this.syncManager.performSync();
      
    } catch (error) {
      this.logger.error('Failed to sync chunks:', error as Error);
      
      return {
        success: false,
        syncedChunks: 0,
        errors: [{
          chunkId: 'sync_queue_error',
          error: (error as Error).message,
          recoverable: true
        }],
        conflicts: [],
        duration: 0
      };
    }
  }

  /**
   * Handle content change events from Obsidian
   */
  async handleContentChange(file: TFile): Promise<void> {
    try {
      // Prevent duplicate processing
      if (this.processingFiles.has(file.path)) {
        this.logger.debug(`Already processing file: ${file.path}`);
        return;
      }
      
      this.processingFiles.add(file.path);
      this.logger.debug(`Handling content change for: ${file.path}`);
      
      // Read file content
      const content = await this.app.vault.read(file);
      
      // Check if content actually changed
      const lastContent = this.lastProcessedContent.get(file.path);
      if (lastContent === content) {
        this.logger.debug(`Content unchanged for: ${file.path}`);
        return;
      }
      
      // Parse content
      const parsed = await this.parseContent(content, file.path);
      
      // Sync to Ink Gateway
      const syncResult = await this.syncToInkGateway(parsed.chunks);
      
      if (syncResult.success) {
        // Update last processed content
        this.lastProcessedContent.set(file.path, content);
        
        // Clear related caches
        this.clearFileCache(file.path);
        
        this.logger.info(`Successfully processed content change for: ${file.path}`);
      } else {
        this.logger.warn(`Sync failed for: ${file.path}`, syncResult.errors);
      }
      
    } catch (error) {
      this.logger.error(`Failed to handle content change for ${file.path}:`, error as Error);
      throw error;
    } finally {
      this.processingFiles.delete(file.path);
    }
  }

  /**
   * Parse hierarchy from content
   */
  parseHierarchy(content: string): HierarchyNode[] {
    try {
      const parsed = MarkdownParser.parseContent(content, 'temp', {
        trackPositions: false,
        parseHierarchy: true,
        extractMetadata: false,
        generateChunks: false
      });
      
      return parsed.hierarchy;
      
    } catch (error) {
      this.logger.error('Failed to parse hierarchy:', error as Error);
      return [];
    }
  }

  /**
   * Extract metadata from file using MetadataManager
   */
  extractMetadata(file: TFile): ContentMetadata {
    return this.metadataManager.extractMetadata(file);
  }

  /**
   * Setup event listeners
   */
  private setupEventListeners(): void {
    // Listen for cache clear events
    this.eventManager.on('clearCache', () => {
      this.clearAllCaches();
    });
    
    // Listen for sync events
    this.eventManager.on('forceSyncFile', async (filePath: string) => {
      const file = this.app.vault.getAbstractFileByPath(filePath);
      if (file instanceof TFile) {
        await this.handleContentChange(file);
      }
    });
  }



  /**
   * Generate simple hash for content
   */
  private getContentHash(content: string): string {
    let hash = 0;
    for (let i = 0; i < content.length; i++) {
      const char = content.charCodeAt(i);
      hash = ((hash << 5) - hash) + char;
      hash = hash & hash;
    }
    return Math.abs(hash).toString(36);
  }

  /**
   * Clear cache for specific file
   */
  private clearFileCache(filePath: string): void {
    // Clear all cache entries related to this file
    const keysToDelete: string[] = [];
    
    // This is a simplified approach - in a real implementation,
    // you might want to track cache keys more systematically
    for (let i = 0; i < 1000; i++) { // Arbitrary limit for safety
      const key = `parsed_${filePath}_${i}`;
      if (this.cacheManager.get(key)) {
        keysToDelete.push(key);
      }
    }
    
    keysToDelete.forEach(key => this.cacheManager.delete(key));
    this.logger.debug(`Cleared ${keysToDelete.length} cache entries for: ${filePath}`);
  }

  /**
   * Clear all caches
   */
  private clearAllCaches(): void {
    this.cacheManager.clear();
    this.lastProcessedContent.clear();
    this.logger.info('All caches cleared');
  }

  /**
   * Get processing status for file
   */
  public isProcessing(filePath: string): boolean {
    return this.processingFiles.has(filePath);
  }

  /**
   * Get last processed content hash
   */
  public getLastProcessedHash(filePath: string): string | undefined {
    const content = this.lastProcessedContent.get(filePath);
    return content ? this.getContentHash(content) : undefined;
  }

  /**
   * Force sync file
   */
  public async forceSyncFile(filePath: string): Promise<void> {
    const file = this.app.vault.getAbstractFileByPath(filePath);
    if (file && file.extension === 'md') {
      // Clear last processed content to force sync
      this.lastProcessedContent.delete(filePath);
      await this.handleContentChange(file);
    } else {
      throw new PluginError(
        ErrorType.VALIDATION_ERROR,
        'FILE_NOT_FOUND',
        { filePath },
        false
      );
    }
  }

  /**
   * Get sync state
   */
  public getSyncState() {
    return this.syncManager.getSyncState();
  }

  /**
   * Get pending changes count
   */
  public getPendingChangesCount(): number {
    return this.syncManager.getPendingChangesCount();
  }

  /**
   * Force sync now
   */
  public async forceSyncNow(): Promise<SyncResult> {
    return await this.syncManager.forceSyncNow();
  }

  /**
   * Update sync options
   */
  public updateSyncOptions(options: Partial<SyncManagerOptions>): void {
    this.syncManager.updateOptions(options);
  }

  /**
   * Start auto sync
   */
  public startAutoSync(): void {
    this.syncManager.startAutoSync();
  }

  /**
   * Stop auto sync
   */
  public stopAutoSync(): void {
    this.syncManager.stopAutoSync();
  }

  /**
   * Clear pending changes
   */
  public clearPendingChanges(): void {
    this.syncManager.clearPendingChanges();
  }

  /**
   * Sync tags between local and remote
   */
  public async syncTags(localTags: string[], remoteTags: string[]): Promise<{
    tagsToAdd: string[];
    tagsToRemove: string[];
    conflicts: string[];
  }> {
    return await this.metadataManager.syncTags(localTags, remoteTags);
  }

  /**
   * Get known tags
   */
  public getKnownTags(): string[] {
    return this.metadataManager.getKnownTags();
  }

  /**
   * Add known tag
   */
  public addKnownTag(tag: string): void {
    this.metadataManager.addKnownTag(tag);
  }

  /**
   * Remove known tag
   */
  public removeKnownTag(tag: string): void {
    this.metadataManager.removeKnownTag(tag);
  }

  /**
   * Update metadata processing options
   */
  public updateMetadataOptions(options: Partial<MetadataProcessingOptions>): void {
    this.metadataManager.updateProcessingOptions(options);
  }

  /**
   * Update tag sync options
   */
  public updateTagSyncOptions(options: Partial<TagSyncOptions>): void {
    this.metadataManager.updateTagSyncOptions(options);
  }

  /**
   * Get metadata processing statistics
   */
  public getMetadataStats() {
    return this.metadataManager.getStats();
  }

  /**
   * Create Obsidian metadata from UnifiedChunk
   */
  public createObsidianMetadata(chunk: UnifiedChunk) {
    return this.metadataManager.createObsidianMetadata(chunk);
  }

  /**
   * Clear metadata caches
   */
  public clearMetadataCaches(): void {
    this.metadataManager.clearCaches();
  }

  // Document ID Management Methods

  /**
   * Generate document ID for a physical file
   */
  generateDocumentId(filePath: string): string {
    if (!filePath || typeof filePath !== 'string') {
      throw new PluginError(
        ErrorType.VALIDATION_ERROR,
        'INVALID_FILE_PATH',
        { filePath },
        false
      );
    }

    // Normalize file path and create a consistent document ID
    const normalizedPath = filePath.replace(/\\/g, '/').replace(/^\/+/, '');
    
    // Use a combination of file path and a hash for uniqueness
    const pathHash = this.generatePathHash(normalizedPath);
    return `file_${pathHash}_${normalizedPath.replace(/[^a-zA-Z0-9]/g, '_')}`;
  }

  /**
   * Generate virtual document ID for non-file content
   */
  generateVirtualDocumentId(context: VirtualDocumentContext): string {
    if (!context || typeof context !== 'object') {
      throw new PluginError(
        ErrorType.VALIDATION_ERROR,
        'INVALID_VIRTUAL_CONTEXT',
        { context },
        false
      );
    }

    if (!context.sourceType || !context.contextId) {
      throw new PluginError(
        ErrorType.VALIDATION_ERROR,
        'MISSING_VIRTUAL_CONTEXT_FIELDS',
        { sourceType: context.sourceType, contextId: context.contextId },
        false
      );
    }

    // Create virtual document ID based on source type and context
    const contextHash = this.generatePathHash(context.contextId);
    const sourcePrefix = context.sourceType.toLowerCase();
    
    return `virtual_${sourcePrefix}_${contextHash}_${context.contextId.replace(/[^a-zA-Z0-9]/g, '_')}`;
  }

  /**
   * Get chunks by document ID with pagination
   */
  async getChunksByDocumentId(documentId: string, options?: PaginationOptions): Promise<DocumentChunksResult> {
    try {
      this.logger.debug(`Retrieving chunks for document ID: ${documentId}`, options);

      // Check cache first
      const cacheKey = `doc_chunks_${documentId}_${JSON.stringify(options || {})}`;
      const cached = this.cacheManager.get<DocumentChunksResult>(cacheKey);
      if (cached) {
        this.logger.debug(`Using cached chunks for document: ${documentId}`);
        return cached;
      }

      // Fetch from API
      const result = await this.apiClient.getChunksByDocumentId(documentId, options);

      // Validate result
      if (!result || !result.chunks || !Array.isArray(result.chunks)) {
        throw new PluginError(
          ErrorType.API_ERROR,
          'INVALID_DOCUMENT_CHUNKS_RESULT',
          { documentId, result },
          true
        );
      }

      // Cache the result (shorter TTL for paginated results)
      this.cacheManager.set(cacheKey, result, 60000); // 1 minute TTL

      this.logger.debug(`Retrieved ${result.chunks.length} chunks for document: ${documentId}`);
      return result;

    } catch (error) {
      this.logger.error(`Failed to get chunks for document ${documentId}:`, error as Error);
      
      if (error instanceof PluginError) {
        throw error;
      }
      
      throw new PluginError(
        ErrorType.API_ERROR,
        'DOCUMENT_CHUNKS_RETRIEVAL_FAILED',
        { documentId, error: (error as Error).message },
        true
      );
    }
  }

  /**
   * Create virtual document
   */
  async createVirtualDocument(context: VirtualDocumentContext): Promise<VirtualDocument> {
    try {
      this.logger.debug('Creating virtual document', context);

      // Validate context
      const virtualDocId = this.generateVirtualDocumentId(context);
      
      // Check if virtual document already exists in cache
      const cacheKey = `virtual_doc_${virtualDocId}`;
      const cached = this.cacheManager.get<VirtualDocument>(cacheKey);
      if (cached) {
        this.logger.debug(`Virtual document already exists: ${virtualDocId}`);
        return cached;
      }

      // Create via API
      const virtualDoc = await this.apiClient.createVirtualDocument(context);

      // Validate result
      if (!virtualDoc || !virtualDoc.virtualDocumentId) {
        throw new PluginError(
          ErrorType.API_ERROR,
          'INVALID_VIRTUAL_DOCUMENT_RESULT',
          { context, result: virtualDoc },
          true
        );
      }

      // Cache the result
      this.cacheManager.set(cacheKey, virtualDoc, 300000); // 5 minutes TTL

      this.logger.info(`Created virtual document: ${virtualDoc.virtualDocumentId}`);
      return virtualDoc;

    } catch (error) {
      this.logger.error('Failed to create virtual document:', error as Error);
      
      if (error instanceof PluginError) {
        throw error;
      }
      
      throw new PluginError(
        ErrorType.API_ERROR,
        'VIRTUAL_DOCUMENT_CREATION_FAILED',
        { context, error: (error as Error).message },
        true
      );
    }
  }

  /**
   * Reconstruct complete document from chunks
   */
  async reconstructDocument(documentId: string): Promise<ReconstructedDocument> {
    try {
      this.logger.debug(`Reconstructing document: ${documentId}`);

      // Check cache first
      const cacheKey = `reconstructed_${documentId}`;
      const cached = this.cacheManager.get<ReconstructedDocument>(cacheKey);
      if (cached) {
        this.logger.debug(`Using cached reconstructed document: ${documentId}`);
        return cached;
      }

      // Get all chunks for the document (without pagination to get complete document)
      const chunksResult = await this.getChunksByDocumentId(documentId, {
        pageSize: 1000, // Large page size to get all chunks
        includeHierarchy: true,
        sortBy: 'position',
        sortOrder: 'asc'
      });

      if (!chunksResult.chunks || chunksResult.chunks.length === 0) {
        throw new PluginError(
          ErrorType.API_ERROR,
          'NO_CHUNKS_FOUND_FOR_DOCUMENT',
          { documentId },
          true
        );
      }

      // Build hierarchy from chunks
      const hierarchy = this.buildHierarchyFromChunks(chunksResult.chunks);

      // Create document metadata
      const documentMetadata: DocumentMetadata = {
        ...chunksResult.documentMetadata,
        totalChunks: chunksResult.chunks.length,
        lastModified: new Date()
      };

      // Create reconstructed document
      const reconstructed: ReconstructedDocument = {
        documentId,
        chunks: chunksResult.chunks,
        hierarchy,
        metadata: documentMetadata,
        reconstructionTime: new Date()
      };

      // Cache the result
      this.cacheManager.set(cacheKey, reconstructed, 300000); // 5 minutes TTL

      this.logger.info(`Successfully reconstructed document: ${documentId}`, {
        chunks: reconstructed.chunks.length,
        hierarchyNodes: reconstructed.hierarchy.length
      });

      return reconstructed;

    } catch (error) {
      this.logger.error(`Failed to reconstruct document ${documentId}:`, error as Error);
      
      if (error instanceof PluginError) {
        throw error;
      }
      
      throw new PluginError(
        ErrorType.API_ERROR,
        'DOCUMENT_RECONSTRUCTION_FAILED',
        { documentId, error: (error as Error).message },
        true
      );
    }
  }

  /**
   * Update document scope for a chunk
   */
  async updateDocumentScope(chunkId: string, documentId: string, scope: DocumentScope): Promise<void> {
    try {
      this.logger.debug(`Updating document scope for chunk ${chunkId}`, { documentId, scope });

      // Update via API
      await this.apiClient.updateDocumentScope(chunkId, documentId, scope);

      // Clear related caches
      this.clearDocumentCaches(documentId);
      this.cacheManager.delete(`chunk_${chunkId}`);

      this.logger.info(`Updated document scope for chunk: ${chunkId}`);

    } catch (error) {
      this.logger.error(`Failed to update document scope for chunk ${chunkId}:`, error as Error);
      
      if (error instanceof PluginError) {
        throw error;
      }
      
      throw new PluginError(
        ErrorType.API_ERROR,
        'DOCUMENT_SCOPE_UPDATE_FAILED',
        { chunkId, documentId, scope, error: (error as Error).message },
        true
      );
    }
  }

  /**
   * Get document ID from file path
   */
  getDocumentIdFromFile(file: TFile): string {
    return this.generateDocumentId(file.path);
  }

  /**
   * Check if document ID is virtual
   */
  isVirtualDocumentId(documentId: string): boolean {
    return documentId.startsWith('virtual_');
  }

  /**
   * Extract file path from document ID (for file-based documents)
   */
  extractFilePathFromDocumentId(documentId: string): string | null {
    if (this.isVirtualDocumentId(documentId)) {
      return null;
    }

    // Extract file path from file-based document ID
    const match = documentId.match(/^file_[^_]+_(.+)$/);
    if (match) {
      // Replace underscores with appropriate characters
      let filePath = match[1];
      
      // Handle the .md extension specially - replace _md at the end with .md
      if (filePath.endsWith('_md')) {
        filePath = filePath.slice(0, -3) + '.md';
      }
      
      // Replace remaining underscores with forward slashes
      filePath = filePath.replace(/_/g, '/');
      
      return filePath;
    }

    return null;
  }

  // Private helper methods

  /**
   * Generate hash for path/context
   */
  private generatePathHash(input: string): string {
    let hash = 0;
    for (let i = 0; i < input.length; i++) {
      const char = input.charCodeAt(i);
      hash = ((hash << 5) - hash) + char;
      hash = hash & hash;
    }
    return Math.abs(hash).toString(36).padStart(6, '0');
  }

  /**
   * Build hierarchy from chunks
   */
  private buildHierarchyFromChunks(chunks: UnifiedChunk[]): HierarchyNode[] {
    const hierarchyMap = new Map<string, HierarchyNode>();
    const rootNodes: HierarchyNode[] = [];

    // First pass: create hierarchy nodes
    for (const chunk of chunks) {
      if (chunk.parent || chunk.isPage) {
        const node: HierarchyNode = {
          id: chunk.chunkId,
          content: chunk.contents,
          level: this.calculateHierarchyLevel(chunk, chunks),
          type: this.determineHierarchyType(chunk),
          parent: chunk.parent,
          children: [],
          position: chunk.position
        };

        hierarchyMap.set(chunk.chunkId, node);
      }
    }

    // Second pass: build parent-child relationships
    for (const node of hierarchyMap.values()) {
      if (node.parent && hierarchyMap.has(node.parent)) {
        const parentNode = hierarchyMap.get(node.parent)!;
        parentNode.children.push(node.id);
      } else {
        rootNodes.push(node);
      }
    }

    return rootNodes;
  }

  /**
   * Calculate hierarchy level for a chunk
   */
  private calculateHierarchyLevel(chunk: UnifiedChunk, allChunks: UnifiedChunk[]): number {
    let level = 0;
    let currentParent = chunk.parent;

    while (currentParent) {
      level++;
      const parentChunk = allChunks.find(c => c.chunkId === currentParent);
      currentParent = parentChunk?.parent;
      
      // Prevent infinite loops
      if (level > 10) break;
    }

    return level;
  }

  /**
   * Determine hierarchy type from chunk
   */
  private determineHierarchyType(chunk: UnifiedChunk): 'heading' | 'bullet' {
    // This is a simplified implementation
    // In a real scenario, you might store this information in chunk metadata
    if (chunk.contents.startsWith('#')) {
      return 'heading';
    }
    return 'bullet';
  }

  /**
   * Clear document-related caches
   */
  private clearDocumentCaches(documentId: string): void {
    const keysToDelete: string[] = [];
    
    // Clear document chunks cache
    for (let page = 1; page <= 100; page++) {
      for (const pageSize of [10, 20, 50, 100]) {
        for (const includeHierarchy of [true, false]) {
          for (const sortBy of ['position', 'created', 'updated']) {
            for (const sortOrder of ['asc', 'desc']) {
              const options = { page, pageSize, includeHierarchy, sortBy, sortOrder };
              const key = `doc_chunks_${documentId}_${JSON.stringify(options)}`;
              keysToDelete.push(key);
            }
          }
        }
      }
    }

    // Clear reconstructed document cache
    keysToDelete.push(`reconstructed_${documentId}`);
    
    // Clear virtual document cache if applicable
    if (this.isVirtualDocumentId(documentId)) {
      keysToDelete.push(`virtual_doc_${documentId}`);
    }

    // Delete cache entries
    keysToDelete.forEach(key => this.cacheManager.delete(key));
    
    this.logger.debug(`Cleared document caches for: ${documentId}`);
  }

  /**
   * Cleanup resources
   */
  public cleanup(): void {
    this.syncManager.cleanup();
    this.metadataManager.cleanup();
  }
}