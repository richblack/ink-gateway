/**
 * Template Sync Manager for handling template instance synchronization
 * Manages synchronization between template instances and file content
 * Implements requirements 4.1, 4.2, 4.3, 4.6
 */

import { TFile, CachedMetadata, Vault, MetadataCache } from 'obsidian';
import {
  Template,
  TemplateInstance,
  UnifiedChunk,
  SyncResult,
  SyncError,
  PluginError,
  ErrorType
} from '../types';
import { IInkGatewayClient, ILogger } from '../interfaces';
import { PropertyMapper, PropertySyncResult } from './PropertyMapper';
import { TemplateValidator, ValidationResult } from './TemplateValidator';
import { TemplateRenderer } from './TemplateRenderer';

export interface TemplateSyncOptions {
  validateBeforeSync: boolean;
  autoFillMissingSlots: boolean;
  preserveUserContent: boolean;
  syncToInkGateway: boolean;
}

export interface TemplateSyncResult extends SyncResult {
  templateId: string;
  instanceId: string;
  validationResult?: ValidationResult;
  propertySync?: PropertySyncResult;
  updatedContent?: string;
}

export class TemplateSyncManager {
  private logger: ILogger;
  private apiClient: IInkGatewayClient;
  private propertyMapper: PropertyMapper;
  private validator: TemplateValidator;
  private renderer: TemplateRenderer;
  private vault: Vault;
  private metadataCache?: MetadataCache;

  constructor(
    apiClient: IInkGatewayClient,
    logger: ILogger,
    vault: Vault,
    metadataCache?: MetadataCache
  ) {
    this.apiClient = apiClient;
    this.logger = logger;
    this.vault = vault;
    this.metadataCache = metadataCache;
    this.propertyMapper = new PropertyMapper(logger);
    this.validator = new TemplateValidator(logger);
    this.renderer = new TemplateRenderer(logger);
  }

  /**
   * Synchronize template instance with file content and Ink-Gateway
   */
  async synchronizeTemplateInstance(
    template: Template,
    instance: TemplateInstance,
    file: TFile,
    options: TemplateSyncOptions = {
      validateBeforeSync: true,
      autoFillMissingSlots: true,
      preserveUserContent: true,
      syncToInkGateway: true
    }
  ): Promise<TemplateSyncResult> {
    try {
      this.logger.debug(`Synchronizing template instance: ${instance.id}`);

      const result: TemplateSyncResult = {
        success: true,
        syncedChunks: 0,
        errors: [],
        conflicts: [],
        duration: 0,
        templateId: template.id,
        instanceId: instance.id
      };

      const startTime = Date.now();

      // Read current file content
      const fileContent = await this.vault.read(file);
      const metadata = this.metadataCache?.getFileCache(file);

      // Validate template instance if requested
      if (options.validateBeforeSync) {
        result.validationResult = this.validator.validateTemplateInstance(template, instance);
        if (!result.validationResult.valid) {
          result.success = false;
          result.errors = result.validationResult.errors.map(e => ({
            chunkId: instance.id,
            error: e.message,
            recoverable: e.severity === 'warning'
          }));
          return result;
        }
      }

      // Auto-fill missing slots if requested
      if (options.autoFillMissingSlots) {
        const autoFillResult = await this.validator.autoFillTemplate(template, instance, {
          file,
          metadata: metadata || undefined
        });
        
        if (autoFillResult.success) {
          instance.slotValues = autoFillResult.filledSlots;
          instance.updatedAt = new Date();
        }
      }

      // Synchronize with file properties
      if (metadata) {
        result.propertySync = await this.propertyMapper.synchronizeProperties(
          template,
          instance,
          fileContent,
          metadata
        );
      }

      // Generate updated content
      const { content: updatedContent, metadata: obsidianMetadata } = 
        await this.propertyMapper.applySlotValuesToProperties(template, instance, fileContent);

      result.updatedContent = updatedContent;

      // Update file content if it has changed
      if (updatedContent !== fileContent) {
        await this.vault.modify(file, updatedContent);
        this.logger.debug(`Updated file content for: ${file.path}`);
      }

      // Synchronize with Ink-Gateway if requested
      if (options.syncToInkGateway) {
        const chunks = this.renderer.instanceToChunks(template, instance);
        const syncResult = await this.syncChunksToInkGateway(chunks);
        
        result.syncedChunks = syncResult.syncedChunks;
        result.errors.push(...syncResult.errors);
        result.conflicts.push(...syncResult.conflicts);
        
        if (!syncResult.success) {
          result.success = false;
        }
      }

      result.duration = Date.now() - startTime;
      this.logger.debug(`Template synchronization completed: ${instance.id}, duration: ${result.duration}ms`);

      return result;

    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : String(error);
      this.logger.error(`Failed to synchronize template instance: ${instance.id}`, error instanceof Error ? error : new Error(String(error)));
      return {
        success: false,
        syncedChunks: 0,
        errors: [{
          chunkId: instance.id,
          error: errorMessage,
          recoverable: true
        }],
        conflicts: [],
        duration: Date.now() - Date.now(),
        templateId: template.id,
        instanceId: instance.id
      };
    }
  }

  /**
   * Update template instances when template definition changes
   */
  async updateTemplateInstances(
    template: Template,
    instances: TemplateInstance[],
    options: TemplateSyncOptions = {
      validateBeforeSync: true,
      autoFillMissingSlots: false,
      preserveUserContent: true,
      syncToInkGateway: true
    }
  ): Promise<TemplateSyncResult[]> {
    try {
      this.logger.debug(`Updating ${instances.length} template instances for template: ${template.name}`);

      const results: TemplateSyncResult[] = [];

      for (const instance of instances) {
        try {
          // Find the file for this instance
          const file = this.vault.getAbstractFileByPath(instance.filePath);
          if (!file || !(file instanceof TFile)) {
            results.push({
              success: false,
              syncedChunks: 0,
              errors: [{
                chunkId: instance.id,
                error: `File not found: ${instance.filePath}`,
                recoverable: false
              }],
              conflicts: [],
              duration: 0,
              templateId: template.id,
              instanceId: instance.id
            });
            continue;
          }

          // Update property mappings for the template
          this.propertyMapper.createPropertyMappings(template);

          // Synchronize the instance
          const syncResult = await this.synchronizeTemplateInstance(
            template,
            instance,
            file,
            options
          );

          results.push(syncResult);

        } catch (error) {
          const errorMessage = error instanceof Error ? error.message : String(error);
          this.logger.error(`Failed to update template instance: ${instance.id}`, error instanceof Error ? error : new Error(String(error)));
          results.push({
            success: false,
            syncedChunks: 0,
            errors: [{
              chunkId: instance.id,
              error: errorMessage,
              recoverable: true
            }],
            conflicts: [],
            duration: 0,
            templateId: template.id,
            instanceId: instance.id
          });
        }
      }

      const successCount = results.filter(r => r.success).length;
      this.logger.info(`Updated ${successCount}/${instances.length} template instances for template: ${template.name}`);

      return results;

    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : String(error);
      this.logger.error(`Failed to update template instances for template: ${template.name}`, error instanceof Error ? error : new Error(String(error)));
      throw new PluginError(
        ErrorType.SYNC_ERROR,
        'TEMPLATE_INSTANCES_UPDATE_FAILED',
        { templateId: template.id, error: errorMessage },
        true
      );
    }
  }

  /**
   * Detect and resolve conflicts between template instances and file content
   */
  async resolveTemplateConflicts(
    template: Template,
    instance: TemplateInstance,
    file: TFile,
    strategy: 'prefer_template' | 'prefer_file' | 'merge' | 'manual' = 'merge'
  ): Promise<TemplateSyncResult> {
    try {
      this.logger.debug(`Resolving template conflicts for instance: ${instance.id}`);

      const fileContent = await this.vault.read(file);
      const metadata = this.metadataCache?.getFileCache(file);

      // Extract slot values from file properties
      const fileSlotValues = metadata ? 
        this.propertyMapper.extractSlotValuesFromProperties(template, metadata) : {};

      // Compare with instance slot values
      const conflicts: string[] = [];
      const resolvedValues: Record<string, any> = { ...instance.slotValues };

      Object.keys(fileSlotValues).forEach(slotId => {
        const instanceValue = instance.slotValues[slotId];
        const fileValue = fileSlotValues[slotId];

        if (instanceValue !== fileValue && fileValue !== undefined) {
          conflicts.push(slotId);

          switch (strategy) {
            case 'prefer_template':
              // Keep instance value
              break;
            case 'prefer_file':
              resolvedValues[slotId] = fileValue;
              break;
            case 'merge':
              // Use file value if instance value is empty
              if (!instanceValue || instanceValue === '') {
                resolvedValues[slotId] = fileValue;
              }
              break;
            case 'manual':
              // Leave both values for manual resolution
              break;
          }
        }
      });

      // Update instance with resolved values
      if (strategy !== 'manual') {
        instance.slotValues = resolvedValues;
        instance.updatedAt = new Date();
      }

      // Synchronize with resolved values
      const syncResult = await this.synchronizeTemplateInstance(template, instance, file, {
        validateBeforeSync: false,
        autoFillMissingSlots: false,
        preserveUserContent: true,
        syncToInkGateway: true
      });

      syncResult.conflicts = conflicts.map(slotId => ({
        chunkId: instance.id,
        localVersion: {
          chunkId: instance.id,
          contents: String(instance.slotValues[slotId] || ''),
          // ... other UnifiedChunk properties would be filled in
        } as UnifiedChunk,
        remoteVersion: {
          chunkId: instance.id,
          contents: String(fileSlotValues[slotId] || ''),
          // ... other UnifiedChunk properties would be filled in
        } as UnifiedChunk,
        conflictType: 'content'
      }));

      this.logger.debug(`Resolved ${conflicts.length} conflicts for instance: ${instance.id}`);
      return syncResult;

    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : String(error);
      this.logger.error(`Failed to resolve template conflicts: ${instance.id}`, error instanceof Error ? error : new Error(String(error)));
      throw new PluginError(
        ErrorType.SYNC_ERROR,
        'TEMPLATE_CONFLICT_RESOLUTION_FAILED',
        { instanceId: instance.id, error: errorMessage },
        true
      );
    }
  }

  /**
   * Batch synchronize multiple template instances
   */
  async batchSynchronizeInstances(
    templates: Map<string, Template>,
    instances: TemplateInstance[],
    options: TemplateSyncOptions = {
      validateBeforeSync: true,
      autoFillMissingSlots: true,
      preserveUserContent: true,
      syncToInkGateway: true
    }
  ): Promise<TemplateSyncResult[]> {
    try {
      this.logger.debug(`Batch synchronizing ${instances.length} template instances`);

      const results: TemplateSyncResult[] = [];
      const batchSize = 10; // Process in batches to avoid overwhelming the system

      for (let i = 0; i < instances.length; i += batchSize) {
        const batch = instances.slice(i, i + batchSize);
        const batchPromises = batch.map(async (instance) => {
          const template = templates.get(instance.templateId);
          if (!template) {
            return {
              success: false,
              syncedChunks: 0,
              errors: [{
                chunkId: instance.id,
                error: `Template not found: ${instance.templateId}`,
                recoverable: false
              }],
              conflicts: [],
              duration: 0,
              templateId: instance.templateId,
              instanceId: instance.id
            } as TemplateSyncResult;
          }

          const file = this.vault.getAbstractFileByPath(instance.filePath);
          if (!file || !(file instanceof TFile)) {
            return {
              success: false,
              syncedChunks: 0,
              errors: [{
                chunkId: instance.id,
                error: `File not found: ${instance.filePath}`,
                recoverable: false
              }],
              conflicts: [],
              duration: 0,
              templateId: template.id,
              instanceId: instance.id
            } as TemplateSyncResult;
          }

          return this.synchronizeTemplateInstance(template, instance, file, options);
        });

        const batchResults = await Promise.allSettled(batchPromises);
        
        batchResults.forEach((result, index) => {
          if (result.status === 'fulfilled') {
            results.push(result.value);
          } else {
            const instance = batch[index];
            results.push({
              success: false,
              syncedChunks: 0,
              errors: [{
                chunkId: instance.id,
                error: result.reason?.message || 'Unknown error',
                recoverable: true
              }],
              conflicts: [],
              duration: 0,
              templateId: instance.templateId,
              instanceId: instance.id
            });
          }
        });

        // Small delay between batches
        if (i + batchSize < instances.length) {
          await new Promise(resolve => setTimeout(resolve, 100));
        }
      }

      const successCount = results.filter(r => r.success).length;
      this.logger.info(`Batch synchronized ${successCount}/${instances.length} template instances`);

      return results;

    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : String(error);
      this.logger.error('Failed to batch synchronize template instances', error instanceof Error ? error : new Error(String(error)));
      throw new PluginError(
        ErrorType.SYNC_ERROR,
        'BATCH_TEMPLATE_SYNC_FAILED',
        { error: errorMessage },
        true
      );
    }
  }

  // Private helper methods

  private async syncChunksToInkGateway(chunks: UnifiedChunk[]): Promise<SyncResult> {
    try {
      const startTime = Date.now();
      const errors: SyncError[] = [];
      let syncedCount = 0;

      // Batch create/update chunks
      for (const chunk of chunks) {
        try {
          // Try to update first, then create if not found
          try {
            await this.apiClient.updateChunk(chunk.chunkId, chunk);
          } catch (updateError) {
            // If update fails, try to create
            await this.apiClient.createChunk(chunk);
          }
          syncedCount++;
        } catch (error) {
          const errorMessage = error instanceof Error ? error.message : String(error);
          errors.push({
            chunkId: chunk.chunkId,
            error: errorMessage,
            recoverable: true
          });
        }
      }

      return {
        success: errors.length === 0,
        syncedChunks: syncedCount,
        errors,
        conflicts: [],
        duration: Date.now() - startTime
      };

    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : String(error);
      this.logger.error('Failed to sync chunks to Ink-Gateway', error instanceof Error ? error : new Error(String(error)));
      return {
        success: false,
        syncedChunks: 0,
        errors: [{
          chunkId: 'batch',
          error: errorMessage,
          recoverable: true
        }],
        conflicts: [],
        duration: 0
      };
    }
  }
}