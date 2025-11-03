/**
 * Metadata Manager for handling Obsidian properties, frontmatter, and tags
 * Supports bidirectional synchronization between Obsidian and Ink Gateway
 */

import { TFile, CachedMetadata } from 'obsidian';
import { 
  ContentMetadata, 
  ObsidianMetadata,
  UnifiedChunk,
  PluginError,
  ErrorType
} from '../types';
import { 
  IInkGatewayClient, 
  ILogger, 
  IEventManager, 
  ICacheManager 
} from '../interfaces';

export interface TagSyncOptions {
  bidirectionalSync: boolean;
  autoCreateTags: boolean;
  tagPrefix?: string;
  excludePatterns?: string[];
}

export interface MetadataProcessingOptions {
  syncFrontmatter: boolean;
  syncProperties: boolean;
  syncTags: boolean;
  preserveObsidianMetadata: boolean;
}

export class MetadataManager {
  private apiClient: IInkGatewayClient;
  private logger: ILogger;
  private eventManager: IEventManager;
  private cacheManager: ICacheManager;
  private app: any; // Obsidian App instance
  
  private tagSyncOptions: TagSyncOptions;
  private processingOptions: MetadataProcessingOptions;
  
  // Tag tracking
  private knownTags = new Set<string>();
  private tagSyncInProgress = new Set<string>();
  private cleanupTimers = new Set<NodeJS.Timeout>();

  constructor(
    apiClient: IInkGatewayClient,
    logger: ILogger,
    eventManager: IEventManager,
    cacheManager: ICacheManager,
    app: any,
    tagSyncOptions: TagSyncOptions = {
      bidirectionalSync: true,
      autoCreateTags: true,
      excludePatterns: ['private/', 'temp/']
    },
    processingOptions: MetadataProcessingOptions = {
      syncFrontmatter: true,
      syncProperties: true,
      syncTags: true,
      preserveObsidianMetadata: true
    }
  ) {
    this.apiClient = apiClient;
    this.logger = logger;
    this.eventManager = eventManager;
    this.cacheManager = cacheManager;
    this.app = app;
    this.tagSyncOptions = tagSyncOptions;
    this.processingOptions = processingOptions;
    
    this.setupEventListeners();
    this.initializeKnownTags();
  }

  /**
   * Setup event listeners for metadata changes
   */
  private setupEventListeners(): void {
    // Listen for metadata cache changes
    this.eventManager.on('metadataCacheChanged', (file: TFile) => {
      this.handleMetadataChange(file);
    });
    
    // Listen for tag changes
    this.eventManager.on('tagChanged', (oldTag: string, newTag: string) => {
      this.handleTagChange(oldTag, newTag);
    });
    
    // Listen for tag creation
    this.eventManager.on('tagCreated', (tag: string) => {
      this.handleTagCreation(tag);
    });
    
    // Listen for tag deletion
    this.eventManager.on('tagDeleted', (tag: string) => {
      this.handleTagDeletion(tag);
    });
  }

  /**
   * Initialize known tags from cache or API
   */
  private async initializeKnownTags(): Promise<void> {
    try {
      // Try to load from cache first
      const cachedTags = this.cacheManager.get<string[]>('known_tags');
      if (cachedTags) {
        this.knownTags = new Set(cachedTags);
        this.logger.debug(`Loaded ${cachedTags.length} known tags from cache`);
        return;
      }
      
      // Load from API if not in cache
      const result = await this.apiClient.searchByTags([]);
      const tags = new Set<string>();
      
      result.items.forEach(item => {
        item.chunk.tags.forEach(tag => tags.add(tag));
      });
      
      this.knownTags = tags;
      this.cacheManager.set('known_tags', Array.from(tags), 3600000); // 1 hour TTL
      
      this.logger.info(`Initialized ${tags.size} known tags from API`);
      
    } catch (error) {
      this.logger.error('Failed to initialize known tags:', error as Error);
      // Continue with empty set
      this.knownTags = new Set();
    }
  }

  /**
   * Extract comprehensive metadata from Obsidian file
   */
  public extractMetadata(file: TFile): ContentMetadata {
    try {
      const cachedMetadata = this.app.metadataCache.getFileCache(file);
      
      const metadata: ContentMetadata = {
        title: file.basename,
        tags: [],
        properties: {},
        frontmatter: {},
        aliases: [],
        cssClasses: [],
        createdTime: new Date(file.stat.ctime),
        modifiedTime: new Date(file.stat.mtime)
      };

      if (cachedMetadata) {
        // Extract frontmatter
        if (this.processingOptions.syncFrontmatter && cachedMetadata.frontmatter) {
          metadata.frontmatter = this.processFrontmatter(cachedMetadata.frontmatter);
          
          // Map common frontmatter fields
          this.mapFrontmatterToMetadata(cachedMetadata.frontmatter, metadata);
        }
        
        // Extract properties (Obsidian properties)
        if (this.processingOptions.syncProperties) {
          metadata.properties = this.extractProperties(cachedMetadata);
        }
        
        // Extract tags
        if (this.processingOptions.syncTags) {
          metadata.tags = this.extractTags(cachedMetadata);
        }
        
        // Extract links and embeds
        metadata.properties.links = this.extractLinks(cachedMetadata);
        metadata.properties.embeds = this.extractEmbeds(cachedMetadata);
        
        // Extract headings structure
        metadata.properties.headings = this.extractHeadings(cachedMetadata);
      }
      
      return metadata;
      
    } catch (error) {
      this.logger.error(`Failed to extract metadata for ${file.path}:`, error as Error);
      throw new PluginError(
        ErrorType.PARSING_ERROR,
        'METADATA_EXTRACTION_FAILED',
        { filePath: file.path, error: (error as Error).message },
        true
      );
    }
  }

  /**
   * Process frontmatter data
   */
  private processFrontmatter(frontmatter: Record<string, any>): Record<string, any> {
    const processed: Record<string, any> = {};
    
    for (const [key, value] of Object.entries(frontmatter)) {
      // Skip null or undefined values
      if (value == null) continue;
      
      // Process different value types
      if (Array.isArray(value)) {
        processed[key] = value.map(v => this.processValue(v));
      } else {
        processed[key] = this.processValue(value);
      }
    }
    
    return processed;
  }

  /**
   * Process individual values (handles dates, numbers, etc.)
   */
  private processValue(value: any): any {
    if (typeof value === 'string') {
      // Try to parse as date
      if (value.match(/^\d{4}-\d{2}-\d{2}/) && !isNaN(Date.parse(value))) {
        return new Date(value);
      }
      
      // Try to parse as number
      const num = Number(value);
      if (!isNaN(num) && isFinite(num) && value.trim() === num.toString()) {
        return num;
      }
      
      // Try to parse as boolean
      if (value.toLowerCase() === 'true') return true;
      if (value.toLowerCase() === 'false') return false;
    }
    
    return value;
  }

  /**
   * Map frontmatter fields to metadata structure
   */
  private mapFrontmatterToMetadata(frontmatter: Record<string, any>, metadata: ContentMetadata): void {
    // Title
    if (frontmatter.title && typeof frontmatter.title === 'string') {
      metadata.title = frontmatter.title;
    }
    
    // Tags
    if (frontmatter.tags) {
      const tags = Array.isArray(frontmatter.tags) ? frontmatter.tags : [frontmatter.tags];
      metadata.tags.push(...tags.filter(tag => typeof tag === 'string'));
    }
    
    // Aliases
    if (frontmatter.aliases) {
      const aliases = Array.isArray(frontmatter.aliases) ? frontmatter.aliases : [frontmatter.aliases];
      metadata.aliases = aliases.filter(alias => typeof alias === 'string');
    }
    
    // CSS classes
    if (frontmatter.cssclass || frontmatter.cssclasses) {
      const cssClasses = frontmatter.cssclass || frontmatter.cssclasses;
      const classes = Array.isArray(cssClasses) ? cssClasses : [cssClasses];
      metadata.cssClasses = classes.filter(cls => typeof cls === 'string');
    }
    
    // Created/modified dates
    if (frontmatter.created && !isNaN(Date.parse(frontmatter.created))) {
      metadata.createdTime = new Date(frontmatter.created);
    }
    
    if (frontmatter.modified && !isNaN(Date.parse(frontmatter.modified))) {
      metadata.modifiedTime = new Date(frontmatter.modified);
    }
  }

  /**
   * Extract Obsidian properties
   */
  private extractProperties(cachedMetadata: CachedMetadata): Record<string, any> {
    const properties: Record<string, any> = {};
    
    // Extract from frontmatter (properties are stored there in newer Obsidian versions)
    if (cachedMetadata.frontmatter) {
      for (const [key, value] of Object.entries(cachedMetadata.frontmatter)) {
        // Skip standard frontmatter fields that are handled separately
        if (['title', 'tags', 'aliases', 'cssclass', 'cssclasses', 'created', 'modified'].includes(key)) {
          continue;
        }
        
        properties[key] = this.processValue(value);
      }
    }
    
    return properties;
  }

  /**
   * Extract tags from cached metadata
   */
  private extractTags(cachedMetadata: CachedMetadata): string[] {
    const tags = new Set<string>();
    
    // Extract from frontmatter tags
    if (cachedMetadata.frontmatter?.tags) {
      const frontmatterTags = Array.isArray(cachedMetadata.frontmatter.tags) 
        ? cachedMetadata.frontmatter.tags 
        : [cachedMetadata.frontmatter.tags];
      
      frontmatterTags.forEach(tag => {
        if (typeof tag === 'string') {
          tags.add(this.normalizeTag(tag));
        }
      });
    }
    
    // Extract from inline tags
    if (cachedMetadata.tags) {
      cachedMetadata.tags.forEach(tagRef => {
        const tag = tagRef.tag.replace(/^#/, ''); // Remove leading #
        tags.add(this.normalizeTag(tag));
      });
    }
    
    // Filter out excluded patterns
    const filteredTags = Array.from(tags).filter(tag => {
      return !this.tagSyncOptions.excludePatterns?.some(pattern => 
        tag.startsWith(pattern.replace('/', ''))
      );
    });
    
    return filteredTags;
  }

  /**
   * Extract links from cached metadata
   */
  private extractLinks(cachedMetadata: CachedMetadata): string[] {
    if (!cachedMetadata.links) return [];
    
    return cachedMetadata.links.map(link => link.link);
  }

  /**
   * Extract embeds from cached metadata
   */
  private extractEmbeds(cachedMetadata: CachedMetadata): string[] {
    if (!cachedMetadata.embeds) return [];
    
    return cachedMetadata.embeds.map(embed => embed.link);
  }

  /**
   * Extract headings structure
   */
  private extractHeadings(cachedMetadata: CachedMetadata): Array<{heading: string, level: number}> {
    if (!cachedMetadata.headings) return [];
    
    return cachedMetadata.headings.map(heading => ({
      heading: heading.heading,
      level: heading.level
    }));
  }

  /**
   * Normalize tag format
   */
  private normalizeTag(tag: string): string {
    // Remove leading/trailing whitespace and #
    let normalized = tag.trim().replace(/^#+/, '');
    
    // Apply tag prefix if configured
    if (this.tagSyncOptions.tagPrefix && !normalized.startsWith(this.tagSyncOptions.tagPrefix)) {
      normalized = `${this.tagSyncOptions.tagPrefix}${normalized}`;
    }
    
    return normalized;
  }

  /**
   * Create Obsidian metadata from UnifiedChunk
   */
  public createObsidianMetadata(chunk: UnifiedChunk): ObsidianMetadata {
    const obsidianMetadata: ObsidianMetadata = {
      properties: {},
      frontmatter: {},
      aliases: [],
      cssClasses: []
    };
    
    // Map chunk metadata to Obsidian format
    if (chunk.metadata) {
      obsidianMetadata.properties = { ...chunk.metadata };
    }
    
    // Create frontmatter from chunk data
    if (chunk.tags.length > 0) {
      obsidianMetadata.frontmatter.tags = chunk.tags;
    }
    
    if (chunk.metadata.title) {
      obsidianMetadata.frontmatter.title = chunk.metadata.title;
    }
    
    if (chunk.createdTime) {
      obsidianMetadata.frontmatter.created = chunk.createdTime.toISOString().split('T')[0];
    }
    
    if (chunk.lastUpdated) {
      obsidianMetadata.frontmatter.modified = chunk.lastUpdated.toISOString().split('T')[0];
    }
    
    return obsidianMetadata;
  }

  /**
   * Sync tags bidirectionally
   */
  public async syncTags(localTags: string[], remoteTags: string[]): Promise<{
    tagsToAdd: string[];
    tagsToRemove: string[];
    conflicts: string[];
  }> {
    const tagsToAdd: string[] = [];
    const tagsToRemove: string[] = [];
    const conflicts: string[] = [];
    
    if (!this.tagSyncOptions.bidirectionalSync) {
      // One-way sync: local to remote
      tagsToAdd.push(...localTags.filter(tag => !remoteTags.includes(tag)));
      return { tagsToAdd, tagsToRemove, conflicts };
    }
    
    // Bidirectional sync
    const localSet = new Set(localTags);
    const remoteSet = new Set(remoteTags);
    
    // Tags to add to remote (exist locally but not remotely)
    for (const tag of localTags) {
      if (!remoteSet.has(tag)) {
        if (this.tagSyncOptions.autoCreateTags || this.knownTags.has(tag)) {
          tagsToAdd.push(tag);
        } else {
          conflicts.push(tag);
        }
      }
    }
    
    // Tags to remove from local (exist remotely but not locally)
    // Only if the tag was not recently modified locally
    for (const tag of remoteTags) {
      if (!localSet.has(tag) && !this.tagSyncInProgress.has(tag)) {
        tagsToRemove.push(tag);
      }
    }
    
    return { tagsToAdd, tagsToRemove, conflicts };
  }

  /**
   * Handle metadata change events
   */
  private async handleMetadataChange(file: TFile): Promise<void> {
    try {
      this.logger.debug(`Handling metadata change for: ${file.path}`);
      
      const metadata = this.extractMetadata(file);
      
      // Emit metadata changed event
      this.eventManager.emit('metadataExtracted', {
        file,
        metadata
      });
      
      // Update known tags
      metadata.tags.forEach(tag => this.knownTags.add(tag));
      this.cacheManager.set('known_tags', Array.from(this.knownTags), 3600000);
      
    } catch (error) {
      this.logger.error(`Failed to handle metadata change for ${file.path}:`, error as Error);
    }
  }

  /**
   * Handle tag change events
   */
  private async handleTagChange(oldTag: string, newTag: string): Promise<void> {
    try {
      this.logger.debug(`Handling tag change: ${oldTag} -> ${newTag}`);
      
      this.tagSyncInProgress.add(oldTag);
      this.tagSyncInProgress.add(newTag);
      
      // Update known tags
      this.knownTags.delete(oldTag);
      this.knownTags.add(newTag);
      
      // Emit tag change event
      this.eventManager.emit('tagSyncRequired', {
        type: 'rename',
        oldTag,
        newTag
      });
      
      // Clean up sync tracking after a delay
      const timer = setTimeout(() => {
        this.tagSyncInProgress.delete(oldTag);
        this.tagSyncInProgress.delete(newTag);
        this.cleanupTimers.delete(timer);
      }, 5000);
      this.cleanupTimers.add(timer);
      
    } catch (error) {
      this.logger.error(`Failed to handle tag change ${oldTag} -> ${newTag}:`, error as Error);
    }
  }

  /**
   * Handle tag creation events
   */
  private async handleTagCreation(tag: string): Promise<void> {
    try {
      this.logger.debug(`Handling tag creation: ${tag}`);
      
      const normalizedTag = this.normalizeTag(tag);
      this.knownTags.add(normalizedTag);
      this.tagSyncInProgress.add(normalizedTag);
      
      // Emit tag creation event
      this.eventManager.emit('tagSyncRequired', {
        type: 'create',
        tag: normalizedTag
      });
      
      // Clean up sync tracking after a delay
      const timer = setTimeout(() => {
        this.tagSyncInProgress.delete(normalizedTag);
        this.cleanupTimers.delete(timer);
      }, 5000);
      this.cleanupTimers.add(timer);
      
    } catch (error) {
      this.logger.error(`Failed to handle tag creation ${tag}:`, error as Error);
    }
  }

  /**
   * Handle tag deletion events
   */
  private async handleTagDeletion(tag: string): Promise<void> {
    try {
      this.logger.debug(`Handling tag deletion: ${tag}`);
      
      const normalizedTag = this.normalizeTag(tag);
      this.knownTags.delete(normalizedTag);
      this.tagSyncInProgress.add(normalizedTag);
      
      // Emit tag deletion event
      this.eventManager.emit('tagSyncRequired', {
        type: 'delete',
        tag: normalizedTag
      });
      
      // Clean up sync tracking after a delay
      const timer = setTimeout(() => {
        this.tagSyncInProgress.delete(normalizedTag);
        this.cleanupTimers.delete(timer);
      }, 5000);
      this.cleanupTimers.add(timer);
      
    } catch (error) {
      this.logger.error(`Failed to handle tag deletion ${tag}:`, error as Error);
    }
  }

  /**
   * Update processing options
   */
  public updateProcessingOptions(options: Partial<MetadataProcessingOptions>): void {
    this.processingOptions = { ...this.processingOptions, ...options };
    this.logger.info('Metadata processing options updated', options);
  }

  /**
   * Update tag sync options
   */
  public updateTagSyncOptions(options: Partial<TagSyncOptions>): void {
    this.tagSyncOptions = { ...this.tagSyncOptions, ...options };
    this.logger.info('Tag sync options updated', options);
  }

  /**
   * Get known tags
   */
  public getKnownTags(): string[] {
    return Array.from(this.knownTags);
  }

  /**
   * Add known tag
   */
  public addKnownTag(tag: string): void {
    const normalizedTag = this.normalizeTag(tag);
    this.knownTags.add(normalizedTag);
    this.cacheManager.set('known_tags', Array.from(this.knownTags), 3600000);
  }

  /**
   * Remove known tag
   */
  public removeKnownTag(tag: string): void {
    const normalizedTag = this.normalizeTag(tag);
    this.knownTags.delete(normalizedTag);
    this.cacheManager.set('known_tags', Array.from(this.knownTags), 3600000);
  }

  /**
   * Clear all caches
   */
  public clearCaches(): void {
    this.cacheManager.delete('known_tags');
    this.knownTags.clear();
    this.tagSyncInProgress.clear();
    this.cleanup();
    this.logger.info('Metadata caches cleared');
  }

  /**
   * Cleanup resources and timers
   */
  public cleanup(): void {
    // Clear all cleanup timers
    this.cleanupTimers.forEach(timer => {
      clearTimeout(timer);
    });
    this.cleanupTimers.clear();
    this.logger.debug('MetadataManager cleanup completed');
  }

  /**
   * Get processing statistics
   */
  public getStats(): {
    knownTags: number;
    syncInProgress: number;
    processingOptions: MetadataProcessingOptions;
    tagSyncOptions: TagSyncOptions;
  } {
    return {
      knownTags: this.knownTags.size,
      syncInProgress: this.tagSyncInProgress.size,
      processingOptions: { ...this.processingOptions },
      tagSyncOptions: { ...this.tagSyncOptions }
    };
  }
}