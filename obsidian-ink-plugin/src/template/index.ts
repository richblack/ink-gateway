/**
 * Template module exports
 * Provides template management, parsing, rendering, and synchronization functionality
 */

export { TemplateManager } from './TemplateManager';
export { TemplateParser } from './TemplateParser';
export { TemplateRenderer } from './TemplateRenderer';
export { PropertyMapper } from './PropertyMapper';
export { TemplateValidator } from './TemplateValidator';
export { TemplateSyncManager } from './TemplateSyncManager';

// Export additional interfaces
export type { PropertyMapping, PropertySyncResult } from './PropertyMapper';
export type { 
  ValidationResult, 
  ValidationError, 
  ValidationWarning, 
  ValidationSuggestion,
  AutoFillResult 
} from './TemplateValidator';
export type { TemplateSyncOptions, TemplateSyncResult } from './TemplateSyncManager';

// Re-export template-related types for convenience
export type {
  Template,
  TemplateInstance,
  TemplateSlot,
  TemplateStructure,
  TemplateSection,
  TemplateMetadata,
  ValidationRule
} from '../types';