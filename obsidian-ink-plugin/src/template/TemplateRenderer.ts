/**
 * Template Renderer for applying templates and rendering content with slot values
 * Handles template instantiation and content generation
 */

import {
  Template,
  TemplateInstance,
  TemplateSlot,
  UnifiedChunk,
  Position,
  ObsidianMetadata
} from '../types';
import { ILogger } from '../interfaces';

export class TemplateRenderer {
  private logger: ILogger;

  constructor(logger: ILogger) {
    this.logger = logger;
  }

  /**
   * Render template with slot values to create final content
   */
  renderTemplate(template: Template, slotValues: Record<string, any>): string {
    try {
      this.logger.debug(`Rendering template: ${template.name}`);

      let content = '';

      // Add template header
      content += `# ${template.name}\n\n`;

      // Add description if available
      if (template.metadata.description) {
        content += `${template.metadata.description}\n\n`;
      }

      // Render each section
      for (const section of template.structure.sections) {
        content += `## ${section.title}\n\n`;
        
        // Apply slot values to section content
        const renderedContent = this.applySlotValues(section.content, slotValues);
        content += `${renderedContent}\n\n`;
      }

      // Add metadata section
      content += this.renderMetadataSection(template, slotValues);

      this.logger.debug(`Template rendered successfully: ${template.name}`);
      return content;

    } catch (error) {
      this.logger.error(`Failed to render template: ${template.name}`, error instanceof Error ? error : new Error(String(error)));
      throw error;
    }
  }

  /**
   * Create template instance with populated values
   */
  createTemplateInstance(
    template: Template,
    filePath: string,
    slotValues: Record<string, any>
  ): TemplateInstance {
    const instance: TemplateInstance = {
      id: this.generateInstanceId(),
      templateId: template.id,
      filePath,
      slotValues: { ...slotValues },
      createdAt: new Date(),
      updatedAt: new Date()
    };

    this.logger.debug(`Created template instance: ${instance.id} for template: ${template.name}`);
    return instance;
  }

  /**
   * Update template instance with new slot values
   */
  updateTemplateInstance(
    instance: TemplateInstance,
    newSlotValues: Record<string, any>
  ): TemplateInstance {
    const updatedInstance: TemplateInstance = {
      ...instance,
      slotValues: { ...instance.slotValues, ...newSlotValues },
      updatedAt: new Date()
    };

    this.logger.debug(`Updated template instance: ${instance.id}`);
    return updatedInstance;
  }

  /**
   * Convert template instance to UnifiedChunks
   */
  instanceToChunks(
    template: Template,
    instance: TemplateInstance
  ): UnifiedChunk[] {
    const chunks: UnifiedChunk[] = [];
    const basePosition: Position = {
      fileName: instance.filePath,
      lineStart: 1,
      lineEnd: 1,
      charStart: 0,
      charEnd: 0
    };

    // Create main instance chunk
    const mainChunk: UnifiedChunk = {
      chunkId: instance.id,
      contents: this.renderTemplate(template, instance.slotValues),
      parent: undefined,
      page: instance.filePath,
      isPage: false,
      isTag: false,
      isTemplate: false,
      isSlot: false,
      ref: template.id,
      tags: template.metadata.tags,
      metadata: {
        templateId: template.id,
        templateName: template.name,
        instanceId: instance.id,
        slotValues: instance.slotValues,
        createdAt: instance.createdAt,
        updatedAt: instance.updatedAt
      },
      createdTime: instance.createdAt,
      lastUpdated: instance.updatedAt,
      position: basePosition,
      filePath: instance.filePath,
      obsidianMetadata: this.createObsidianMetadata(template, instance)
    };

    chunks.push(mainChunk);

    // Create chunks for populated slots
    template.slots.forEach((slot, index) => {
      const slotValue = instance.slotValues[slot.id];
      if (slotValue !== undefined && slotValue !== null && slotValue !== '') {
        const slotChunk: UnifiedChunk = {
          chunkId: `${instance.id}-${slot.id}`,
          contents: this.formatSlotValue(slotValue, slot.type),
          parent: instance.id,
          page: instance.filePath,
          isPage: false,
          isTag: slot.type === 'tag',
          isTemplate: false,
          isSlot: true,
          ref: slot.id,
          tags: slot.type === 'tag' ? [String(slotValue)] : [],
          metadata: {
            slotName: slot.name,
            slotType: slot.type,
            slotValue: slotValue,
            templateId: template.id,
            instanceId: instance.id
          },
          createdTime: instance.createdAt,
          lastUpdated: instance.updatedAt,
                documentId: 'test-doc-1',
      documentScope: 'file' as const,
      position: {
            ...basePosition,
            lineStart: index + 5, // Approximate line position
            lineEnd: index + 5
          },
          filePath: instance.filePath,
          obsidianMetadata: {
            properties: {
              slotName: slot.name,
              slotType: slot.type,
              slotValue: slotValue
            },
            frontmatter: {},
            aliases: [],
            cssClasses: ['template-slot-value']
          }
        };

        chunks.push(slotChunk);
      }
    });

    return chunks;
  }

  /**
   * Generate template preview with placeholder values
   */
  generateTemplatePreview(template: Template): string {
    const placeholderValues: Record<string, any> = {};

    // Generate placeholder values for each slot
    template.slots.forEach(slot => {
      placeholderValues[slot.id] = this.generatePlaceholderValue(slot);
    });

    return this.renderTemplate(template, placeholderValues);
  }

  /**
   * Extract slot values from rendered content
   */
  extractSlotValuesFromContent(
    template: Template,
    content: string
  ): Record<string, any> {
    const slotValues: Record<string, any> = {};

    // This is a simplified extraction - in practice, you'd need more sophisticated parsing
    template.slots.forEach(slot => {
      const pattern = new RegExp(`<!-- slot:${slot.name} -->(.*?)<!-- /slot:${slot.name} -->`, 's');
      const match = content.match(pattern);
      
      if (match) {
        slotValues[slot.id] = this.parseSlotValue(match[1].trim(), slot.type);
      }
    });

    return slotValues;
  }

  /**
   * Validate rendered content against template structure
   */
  validateRenderedContent(template: Template, content: string): {
    valid: boolean;
    errors: string[];
    warnings: string[];
  } {
    const errors: string[] = [];
    const warnings: string[] = [];

    try {
      // Check if all required sections are present
      for (const section of template.structure.sections) {
        const sectionPattern = new RegExp(`^##\\s+${section.title}`, 'm');
        if (!sectionPattern.test(content)) {
          errors.push(`Missing required section: ${section.title}`);
        }
      }

      // Check if all required slots have values
      for (const slot of template.slots) {
        if (slot.required) {
          const slotPattern = new RegExp(`\\{\\{${slot.name}(?::[^}]+)?\\}\\}`, 'g');
          if (slotPattern.test(content)) {
            errors.push(`Required slot not filled: ${slot.name}`);
          }
        }
      }

      // Check for unfilled optional slots
      const unfilledSlots = content.match(/\{\{\w+(?::[^}]+)?\}\}/g);
      if (unfilledSlots) {
        warnings.push(`Unfilled optional slots: ${unfilledSlots.join(', ')}`);
      }

      return {
        valid: errors.length === 0,
        errors,
        warnings
      };

    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : String(error);
      errors.push(`Validation error: ${errorMessage}`);
      return { valid: false, errors, warnings };
    }
  }

  // Private helper methods

  private applySlotValues(content: string, slotValues: Record<string, any>): string {
    const slotPattern = /\{\{(\w+)(?::(\w+))?(?:\|(.*?))?\}\}/g;

    return content.replace(slotPattern, (match, name, type, options) => {
      const slotId = `slot-${name.toLowerCase().replace(/[^a-z0-9]/g, '-')}`;
      const value = slotValues[slotId];

      if (value !== undefined && value !== null && value !== '') {
        return this.formatSlotValue(value, type);
      }

      // Check for default value in options
      const defaultValue = this.extractDefaultValue(options);
      if (defaultValue !== undefined) {
        return this.formatSlotValue(defaultValue, type);
      }

      // Return placeholder or empty string
      return `[${name}]`;
    });
  }

  private formatSlotValue(value: any, type?: string): string {
    switch (type) {
      case 'date':
        if (value instanceof Date) {
          return value.toISOString().split('T')[0];
        }
        if (typeof value === 'string') {
          const date = new Date(value);
          return isNaN(date.getTime()) ? String(value) : date.toISOString().split('T')[0];
        }
        return String(value);

      case 'number':
        const num = Number(value);
        return isNaN(num) ? String(value) : String(num);

      case 'link':
        const linkValue = String(value);
        if (linkValue.startsWith('[[') && linkValue.endsWith(']]')) {
          return linkValue;
        }
        return `[[${linkValue}]]`;

      case 'tag':
        const tagValue = String(value);
        if (tagValue.startsWith('#')) {
          return tagValue;
        }
        return `#${tagValue}`;

      case 'text':
      default:
        return String(value);
    }
  }

  private generatePlaceholderValue(slot: TemplateSlot): any {
    if (slot.defaultValue !== undefined) {
      return slot.defaultValue;
    }

    switch (slot.type) {
      case 'text':
        return `[${slot.name}]`;
      case 'number':
        return 0;
      case 'date':
        return new Date().toISOString().split('T')[0];
      case 'link':
        return `[[${slot.name}]]`;
      case 'tag':
        return `#${slot.name.toLowerCase()}`;
      default:
        return `[${slot.name}]`;
    }
  }

  private parseSlotValue(value: string, type: string): any {
    switch (type) {
      case 'number':
        const num = Number(value);
        return isNaN(num) ? value : num;
      case 'date':
        const date = new Date(value);
        return isNaN(date.getTime()) ? value : date;
      case 'link':
        return value.replace(/^\[\[|\]\]$/g, '');
      case 'tag':
        return value.replace(/^#/, '');
      default:
        return value;
    }
  }

  private extractDefaultValue(options: string): any {
    if (!options) return undefined;

    const defaultMatch = options.match(/default:([^,]+)/);
    if (defaultMatch) {
      const value = defaultMatch[1].trim();
      // Remove quotes if present
      return value.replace(/^["']|["']$/g, '');
    }

    return undefined;
  }

  private createObsidianMetadata(template: Template, instance: TemplateInstance): ObsidianMetadata {
    const properties: Record<string, any> = {
      templateId: template.id,
      templateName: template.name,
      instanceId: instance.id,
      createdAt: instance.createdAt.toISOString(),
      updatedAt: instance.updatedAt.toISOString()
    };

    // Add slot values as properties
    template.slots.forEach(slot => {
      const value = instance.slotValues[slot.id];
      if (value !== undefined && value !== null && value !== '') {
        properties[slot.name] = value;
      }
    });

    return {
      properties,
      frontmatter: {
        template: template.name,
        templateId: template.id,
        instanceId: instance.id
      },
      aliases: [template.name],
      cssClasses: ['template-instance', `template-${template.name.toLowerCase().replace(/\s+/g, '-')}`]
    };
  }

  private renderMetadataSection(template: Template, slotValues: Record<string, any>): string {
    let metadata = '---\n';
    metadata += `template: ${template.name}\n`;
    metadata += `templateId: ${template.id}\n`;
    metadata += `createdAt: ${new Date().toISOString()}\n`;

    // Add filled slot values as metadata
    template.slots.forEach(slot => {
      const value = slotValues[slot.id];
      if (value !== undefined && value !== null && value !== '') {
        metadata += `${slot.name}: ${JSON.stringify(value)}\n`;
      }
    });

    metadata += '---\n\n';
    return metadata;
  }

  private generateInstanceId(): string {
    return `instance-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
  }
}