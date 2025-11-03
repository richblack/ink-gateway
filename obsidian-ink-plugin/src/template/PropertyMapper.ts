/**
 * Property Mapper for integrating template slots with Obsidian properties
 * Handles mapping between template slots and Obsidian frontmatter/properties
 * Implements requirements 4.1, 4.2, 4.3, 4.6
 */

import { CachedMetadata } from 'obsidian';
import {
  Template,
  TemplateInstance,
  TemplateSlot,
  ObsidianMetadata,
  PluginError,
  ErrorType
} from '../types';
import { ILogger } from '../interfaces';

export interface PropertyMapping {
  slotId: string;
  slotName: string;
  propertyName: string;
  propertyType: 'frontmatter' | 'property';
  bidirectional: boolean;
  transform?: (value: any) => any;
}

export interface PropertySyncResult {
  success: boolean;
  updatedSlots: string[];
  updatedProperties: string[];
  errors: string[];
}

export class PropertyMapper {
  private logger: ILogger;
  private mappings: Map<string, PropertyMapping[]> = new Map();

  constructor(logger: ILogger) {
    this.logger = logger;
  }

  /**
   * Create property mappings for a template
   * Maps template slots to Obsidian properties based on slot configuration
   */
  createPropertyMappings(template: Template): PropertyMapping[] {
    try {
      this.logger.debug(`Creating property mappings for template: ${template.name}`);

      const mappings: PropertyMapping[] = [];

      template.slots.forEach(slot => {
        const mapping: PropertyMapping = {
          slotId: slot.id,
          slotName: slot.name,
          propertyName: this.generatePropertyName(slot.name),
          propertyType: this.determinePropertyType(slot),
          bidirectional: true,
          transform: this.createTransformFunction(slot)
        };

        mappings.push(mapping);
      });

      // Store mappings for the template
      this.mappings.set(template.id, mappings);

      this.logger.debug(`Created ${mappings.length} property mappings for template: ${template.name}`);
      return mappings;

    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : String(error);
      this.logger.error(`Failed to create property mappings for template: ${template.name}`, error instanceof Error ? error : new Error(String(error)));
      throw new PluginError(
        ErrorType.VALIDATION_ERROR,
        'PROPERTY_MAPPING_CREATION_FAILED',
        { templateId: template.id, error: errorMessage },
        true
      );
    }
  }

  /**
   * Apply template slot values to Obsidian properties
   * Updates file frontmatter and properties based on template instance
   */
  async applySlotValuesToProperties(
    template: Template,
    instance: TemplateInstance,
    fileContent: string
  ): Promise<{ content: string; metadata: ObsidianMetadata }> {
    try {
      this.logger.debug(`Applying slot values to properties for instance: ${instance.id}`);

      const mappings = this.mappings.get(template.id) || this.createPropertyMappings(template);
      const { frontmatter, properties } = this.parseFileMetadata(fileContent);
      
      // Apply slot values to properties
      mappings.forEach(mapping => {
        const slotValue = instance.slotValues[mapping.slotId];
        if (slotValue !== undefined && slotValue !== null && slotValue !== '') {
          const transformedValue = mapping.transform ? mapping.transform(slotValue) : slotValue;
          
          if (mapping.propertyType === 'frontmatter') {
            frontmatter[mapping.propertyName] = transformedValue;
          } else {
            properties[mapping.propertyName] = transformedValue;
          }
        }
      });

      // Add template metadata
      frontmatter.template = template.name;
      frontmatter.templateId = template.id;
      frontmatter.instanceId = instance.id;
      frontmatter.createdAt = instance.createdAt.toISOString();
      frontmatter.updatedAt = instance.updatedAt.toISOString();

      // Generate updated content
      const updatedContent = this.generateContentWithMetadata(fileContent, frontmatter, properties);
      
      const obsidianMetadata: ObsidianMetadata = {
        properties: { ...properties, ...frontmatter }, // Merge properties and frontmatter
        frontmatter,
        aliases: frontmatter.aliases || [],
        cssClasses: frontmatter.cssClasses || [`template-${template.name.toLowerCase().replace(/\s+/g, '-')}`]
      };

      this.logger.debug(`Applied slot values to properties for instance: ${instance.id}`);
      return { content: updatedContent, metadata: obsidianMetadata };

    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : String(error);
      this.logger.error(`Failed to apply slot values to properties: ${instance.id}`, error instanceof Error ? error : new Error(String(error)));
      throw new PluginError(
        ErrorType.SYNC_ERROR,
        'PROPERTY_APPLICATION_FAILED',
        { instanceId: instance.id, error: errorMessage },
        true
      );
    }
  }

  /**
   * Extract slot values from Obsidian properties
   * Reads file properties and maps them back to template slot values
   */
  extractSlotValuesFromProperties(
    template: Template,
    metadata: CachedMetadata
  ): Record<string, any> {
    try {
      this.logger.debug(`Extracting slot values from properties for template: ${template.name}`);

      const mappings = this.mappings.get(template.id) || this.createPropertyMappings(template);
      const slotValues: Record<string, any> = {};

      const frontmatter = metadata.frontmatter || {};
      // Note: CachedMetadata doesn't have a properties field in Obsidian API
      // Properties are typically stored in frontmatter
      const properties = frontmatter;

      mappings.forEach(mapping => {
        let propertyValue: any;

        if (mapping.propertyType === 'frontmatter') {
          propertyValue = frontmatter[mapping.propertyName];
        } else {
          propertyValue = properties[mapping.propertyName];
        }

        if (propertyValue !== undefined && propertyValue !== null) {
          // Apply reverse transform if needed
          const slotValue = mapping.transform ? this.reverseTransform(propertyValue, mapping) : propertyValue;
          slotValues[mapping.slotId] = slotValue;
        }
      });

      this.logger.debug(`Extracted ${Object.keys(slotValues).length} slot values from properties`);
      return slotValues;

    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : String(error);
      this.logger.error(`Failed to extract slot values from properties: ${template.name}`, error instanceof Error ? error : new Error(String(error)));
      throw new PluginError(
        ErrorType.PARSING_ERROR,
        'PROPERTY_EXTRACTION_FAILED',
        { templateId: template.id, error: errorMessage },
        true
      );
    }
  }

  /**
   * Synchronize template instance with file properties
   * Bidirectional sync between template slots and Obsidian properties
   */
  async synchronizeProperties(
    template: Template,
    instance: TemplateInstance,
    fileContent: string,
    metadata: CachedMetadata
  ): Promise<PropertySyncResult> {
    try {
      this.logger.debug(`Synchronizing properties for instance: ${instance.id}`);

      const mappings = this.mappings.get(template.id) || this.createPropertyMappings(template);
      const result: PropertySyncResult = {
        success: true,
        updatedSlots: [],
        updatedProperties: [],
        errors: []
      };

      // Extract current property values
      const propertySlotValues = this.extractSlotValuesFromProperties(template, metadata);
      
      // Compare with instance slot values and sync
      mappings.forEach(mapping => {
        if (!mapping.bidirectional) return;

        const instanceValue = instance.slotValues[mapping.slotId];
        const propertyValue = propertySlotValues[mapping.slotId];

        // Determine which value is more recent or should take precedence
        if (instanceValue !== propertyValue) {
          // For now, prioritize instance values (could be made configurable)
          if (instanceValue !== undefined && instanceValue !== null && instanceValue !== '') {
            result.updatedProperties.push(mapping.propertyName);
          } else if (propertyValue !== undefined && propertyValue !== null && propertyValue !== '') {
            instance.slotValues[mapping.slotId] = propertyValue;
            result.updatedSlots.push(mapping.slotName);
          }
        }
      });

      this.logger.debug(`Synchronized properties for instance: ${instance.id}`);
      return result;

    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : String(error);
      this.logger.error(`Failed to synchronize properties: ${instance.id}`, error instanceof Error ? error : new Error(String(error)));
      return {
        success: false,
        updatedSlots: [],
        updatedProperties: [],
        errors: [errorMessage]
      };
    }
  }

  /**
   * Validate property mappings for a template
   */
  validatePropertyMappings(template: Template): { valid: boolean; errors: string[] } {
    const errors: string[] = [];

    try {
      const mappings = this.mappings.get(template.id) || this.createPropertyMappings(template);

      // Check for duplicate property names
      const propertyNames = mappings.map(m => m.propertyName);
      const duplicates = propertyNames.filter((name, index) => propertyNames.indexOf(name) !== index);
      
      if (duplicates.length > 0) {
        const uniqueDuplicates = Array.from(new Set(duplicates));
        errors.push(`Duplicate property names: ${uniqueDuplicates.join(', ')}`);
      }

      // Check for invalid property names
      mappings.forEach(mapping => {
        if (!this.isValidPropertyName(mapping.propertyName)) {
          errors.push(`Invalid property name: ${mapping.propertyName}`);
        }
      });

      return { valid: errors.length === 0, errors };

    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : String(error);
      errors.push(`Validation error: ${errorMessage}`);
      return { valid: false, errors };
    }
  }

  /**
   * Update property mappings for a template
   */
  updatePropertyMappings(templateId: string, mappings: PropertyMapping[]): void {
    try {
      this.logger.debug(`Updating property mappings for template: ${templateId}`);

      // Validate mappings
      const validation = this.validateMappings(mappings);
      if (!validation.valid) {
        throw new PluginError(
          ErrorType.VALIDATION_ERROR,
          'INVALID_PROPERTY_MAPPINGS',
          { errors: validation.errors },
          false
        );
      }

      this.mappings.set(templateId, mappings);
      this.logger.debug(`Updated property mappings for template: ${templateId}`);

    } catch (error) {
      this.logger.error(`Failed to update property mappings: ${templateId}`, error instanceof Error ? error : new Error(String(error)));
      throw error;
    }
  }

  /**
   * Get property mappings for a template
   */
  getPropertyMappings(templateId: string): PropertyMapping[] {
    return this.mappings.get(templateId) || [];
  }

  /**
   * Remove property mappings for a template
   */
  removePropertyMappings(templateId: string): void {
    this.mappings.delete(templateId);
    this.logger.debug(`Removed property mappings for template: ${templateId}`);
  }

  // Private helper methods

  private generatePropertyName(slotName: string): string {
    // Convert slot name to valid property name
    return slotName.toLowerCase()
      .replace(/[^a-z0-9]/g, '_')
      .replace(/^_+|_+$/g, '')
      .replace(/_+/g, '_');
  }

  private determinePropertyType(slot: TemplateSlot): 'frontmatter' | 'property' {
    // Determine whether to use frontmatter or properties based on slot type
    switch (slot.type) {
      case 'tag':
        return 'frontmatter'; // Tags are typically in frontmatter
      case 'date':
        return 'property'; // Dates work well as properties
      case 'number':
        return 'property'; // Numbers work well as properties
      default:
        return 'frontmatter'; // Default to frontmatter for text and links
    }
  }

  private createTransformFunction(slot: TemplateSlot): ((value: any) => any) | undefined {
    switch (slot.type) {
      case 'date':
        return (value: any) => {
          if (value instanceof Date) {
            return value.toISOString().split('T')[0];
          }
          if (typeof value === 'string') {
            const date = new Date(value);
            return isNaN(date.getTime()) ? value : date.toISOString().split('T')[0];
          }
          return value;
        };

      case 'number':
        return (value: any) => {
          const num = Number(value);
          return isNaN(num) ? value : num;
        };

      case 'tag':
        return (value: any) => {
          const tagValue = String(value);
          if (Array.isArray(value)) {
            return value.map(v => String(v).replace(/^#/, ''));
          }
          return tagValue.replace(/^#/, '');
        };

      case 'link':
        return (value: any) => {
          const linkValue = String(value);
          return linkValue.replace(/^\[\[|\]\]$/g, '');
        };

      default:
        return undefined;
    }
  }

  private reverseTransform(value: any, _mapping: PropertyMapping): any {
    // This is a simplified reverse transform - in practice, you'd need to know the slot type
    if (typeof value === 'string' && value.match(/^\d{4}-\d{2}-\d{2}$/)) {
      return new Date(value);
    }
    
    return value;
  }

  private parseFileMetadata(content: string): { frontmatter: Record<string, any>; properties: Record<string, any> } {
    const frontmatter: Record<string, any> = {};
    const properties: Record<string, any> = {};

    // Parse frontmatter
    const frontmatterMatch = content.match(/^---\n([\s\S]*?)\n---/);
    if (frontmatterMatch) {
      const frontmatterContent = frontmatterMatch[1];
      const lines = frontmatterContent.split('\n');
      
      for (const line of lines) {
        const match = line.match(/^(\w+):\s*(.+)$/);
        if (match) {
          const [, key, value] = match;
          try {
            // Try to parse as JSON, otherwise use as string
            frontmatter[key] = JSON.parse(value);
          } catch {
            frontmatter[key] = value.replace(/^["']|["']$/g, '');
          }
        }
      }
    }

    return { frontmatter, properties };
  }

  private generateContentWithMetadata(
    content: string,
    frontmatter: Record<string, any>,
    _properties: Record<string, any>
  ): string {
    // Remove existing frontmatter
    const contentWithoutFrontmatter = content.replace(/^---\n[\s\S]*?\n---\n/, '');
    
    // Generate new frontmatter
    let newFrontmatter = '---\n';
    Object.entries(frontmatter).forEach(([key, value]) => {
      if (value !== undefined && value !== null) {
        newFrontmatter += `${key}: ${JSON.stringify(value)}\n`;
      }
    });
    newFrontmatter += '---\n\n';

    return newFrontmatter + contentWithoutFrontmatter;
  }

  private isValidPropertyName(name: string): boolean {
    // Check if property name is valid for Obsidian
    return /^[a-z][a-z0-9_]*$/i.test(name) && name.length <= 50;
  }

  private validateMappings(mappings: PropertyMapping[]): { valid: boolean; errors: string[] } {
    const errors: string[] = [];

    mappings.forEach((mapping, index) => {
      if (!mapping.slotId || !mapping.slotName || !mapping.propertyName) {
        errors.push(`Mapping ${index}: Missing required fields`);
      }

      if (!this.isValidPropertyName(mapping.propertyName)) {
        errors.push(`Mapping ${index}: Invalid property name: ${mapping.propertyName}`);
      }

      if (!['frontmatter', 'property'].includes(mapping.propertyType)) {
        errors.push(`Mapping ${index}: Invalid property type: ${mapping.propertyType}`);
      }
    });

    return { valid: errors.length === 0, errors };
  }
}