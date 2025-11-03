/**
 * Template Manager for handling template creation, application, and management
 * Implements requirements 4.1, 4.2, 4.3, 4.4, 4.5, 4.6
 */

import { TFile, Notice } from 'obsidian';
import {
  Template,
  TemplateInstance,
  TemplateStructure,
  TemplateSlot,
  TemplateMetadata,
  ValidationRule,
  UnifiedChunk,
  Position,
  PluginError,
  ErrorType
} from '../types';
import { ITemplateManager, IInkGatewayClient, ILogger } from '../interfaces';

export class TemplateManager implements ITemplateManager {
  private templates: Map<string, Template> = new Map();
  private instances: Map<string, TemplateInstance[]> = new Map();
  private apiClient: IInkGatewayClient;
  private logger: ILogger;

  constructor(apiClient: IInkGatewayClient, logger: ILogger) {
    this.apiClient = apiClient;
    this.logger = logger;
  }

  /**
   * Create a new template with the specified structure
   * Requirement 4.1: Template creation and management
   */
  async createTemplate(name: string, structure: TemplateStructure): Promise<Template> {
    try {
      this.logger.debug(`Creating template: ${name}`);

      // Validate template structure
      this.validateTemplateStructure(structure);

      // Generate template ID
      const templateId = this.generateTemplateId(name);

      // Create template object
      const template: Template = {
        id: templateId,
        name,
        slots: this.extractSlotsFromStructure(structure),
        structure,
        metadata: {
          description: `Template for ${name}`,
          category: 'user-defined',
          tags: [],
          createdTime: new Date(),
          lastUpdated: new Date()
        }
      };

      // Store template locally
      this.templates.set(templateId, template);
      this.instances.set(templateId, []);

      // Sync to Ink-Gateway
      const savedTemplate = await this.apiClient.createTemplate(template);
      
      this.logger.info(`Template created successfully: ${name} (${templateId})`);
      new Notice(`Template "${name}" created successfully`);

      return savedTemplate;
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : String(error);
      this.logger.error(`Failed to create template: ${name}`, error instanceof Error ? error : new Error(String(error)));
      throw new PluginError(
        ErrorType.API_ERROR,
        'TEMPLATE_CREATION_FAILED',
        { name, error: errorMessage },
        true
      );
    }
  }

  /**
   * Apply a template to a target file
   * Requirement 4.2: Template application logic
   */
  async applyTemplate(templateId: string, targetFile: TFile): Promise<void> {
    try {
      this.logger.debug(`Applying template ${templateId} to file: ${targetFile.path}`);

      const template = this.templates.get(templateId);
      if (!template) {
        throw new PluginError(
          ErrorType.VALIDATION_ERROR,
          'TEMPLATE_NOT_FOUND',
          { templateId },
          false
        );
      }

      // Generate template content
      const templateContent = this.generateTemplateContent(template);

      // Create template instance
      const instance: TemplateInstance = {
        id: this.generateInstanceId(),
        templateId,
        filePath: targetFile.path,
        slotValues: this.getDefaultSlotValues(template.slots),
        createdAt: new Date(),
        updatedAt: new Date()
      };

      // Store instance
      const instances = this.instances.get(templateId) || [];
      instances.push(instance);
      this.instances.set(templateId, instances);

      // Apply template to file (this would be handled by the calling code)
      this.logger.info(`Template applied successfully: ${templateId} to ${targetFile.path}`);
      new Notice(`Template "${template.name}" applied to ${targetFile.name}`);

    } catch (error) {
      this.logger.error(`Failed to apply template: ${templateId}`, error instanceof Error ? error : new Error(String(error)));
      throw error;
    }
  }

  /**
   * Parse template from existing content
   * Requirement 4.3: Template structure definition and slot system
   */
  parseTemplateFromContent(content: string): Template {
    try {
      this.logger.debug('Parsing template from content');

      // Extract template metadata from frontmatter or content
      const metadata = this.extractTemplateMetadata(content);
      
      // Parse template structure
      const structure = this.parseTemplateStructure(content);
      
      // Extract slots from content
      const slots = this.extractSlotsFromContent(content);

      const template: Template = {
        id: this.generateTemplateId(metadata.name || 'parsed-template'),
        name: metadata.name || 'Parsed Template',
        slots,
        structure,
        metadata: {
          description: metadata.description || 'Template parsed from content',
          category: metadata.category || 'parsed',
          tags: metadata.tags || [],
          createdTime: new Date(),
          lastUpdated: new Date()
        }
      };

      this.logger.info(`Template parsed successfully: ${template.name}`);
      return template;

    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : String(error);
      this.logger.error('Failed to parse template from content', error instanceof Error ? error : new Error(String(error)));
      throw new PluginError(
        ErrorType.PARSING_ERROR,
        'TEMPLATE_PARSING_FAILED',
        { error: errorMessage },
        true
      );
    }
  }

  /**
   * Get all instances of a specific template
   * Requirement 4.4: Template instance tracking and query functionality
   */
  async getTemplateInstances(templateId: string): Promise<TemplateInstance[]> {
    try {
      this.logger.debug(`Getting instances for template: ${templateId}`);

      // Try to get from local cache first
      let instances = this.instances.get(templateId);

      if (!instances) {
        // Fetch from Ink-Gateway if not in cache
        instances = await this.apiClient.getTemplateInstances(templateId);
        this.instances.set(templateId, instances);
      }

      this.logger.debug(`Found ${instances.length} instances for template: ${templateId}`);
      return instances;

    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : String(error);
      this.logger.error(`Failed to get template instances: ${templateId}`, error instanceof Error ? error : new Error(String(error)));
      throw new PluginError(
        ErrorType.API_ERROR,
        'TEMPLATE_INSTANCES_FETCH_FAILED',
        { templateId, error: errorMessage },
        true
      );
    }
  }

  /**
   * Update an existing template
   * Requirement 4.5: Template management
   */
  async updateTemplate(templateId: string, updates: Partial<Template>): Promise<Template> {
    try {
      this.logger.debug(`Updating template: ${templateId}`);

      const existingTemplate = this.templates.get(templateId);
      if (!existingTemplate) {
        throw new PluginError(
          ErrorType.VALIDATION_ERROR,
          'TEMPLATE_NOT_FOUND',
          { templateId },
          false
        );
      }

      // Merge updates with existing template
      const updatedTemplate: Template = {
        ...existingTemplate,
        ...updates,
        metadata: {
          ...existingTemplate.metadata,
          ...updates.metadata,
          lastUpdated: new Date()
        }
      };

      // Validate updated structure
      if (updates.structure) {
        this.validateTemplateStructure(updates.structure);
      }

      // Update local cache
      this.templates.set(templateId, updatedTemplate);

      this.logger.info(`Template updated successfully: ${templateId}`);
      return updatedTemplate;

    } catch (error) {
      this.logger.error(`Failed to update template: ${templateId}`, error instanceof Error ? error : new Error(String(error)));
      throw error;
    }
  }

  /**
   * Delete a template and all its instances
   * Requirement 4.6: Template management
   */
  async deleteTemplate(templateId: string): Promise<void> {
    try {
      this.logger.debug(`Deleting template: ${templateId}`);

      const template = this.templates.get(templateId);
      if (!template) {
        throw new PluginError(
          ErrorType.VALIDATION_ERROR,
          'TEMPLATE_NOT_FOUND',
          { templateId },
          false
        );
      }

      // Remove from local cache
      this.templates.delete(templateId);
      this.instances.delete(templateId);

      this.logger.info(`Template deleted successfully: ${templateId}`);
      new Notice(`Template "${template.name}" deleted`);

    } catch (error) {
      this.logger.error(`Failed to delete template: ${templateId}`, error instanceof Error ? error : new Error(String(error)));
      throw error;
    }
  }

  /**
   * Get all available templates
   */
  getTemplates(): Template[] {
    return Array.from(this.templates.values());
  }

  /**
   * Get a specific template by ID
   */
  getTemplate(templateId: string): Template | undefined {
    return this.templates.get(templateId);
  }

  /**
   * Validate slot values against template definition
   */
  validateSlotValues(template: Template, slotValues: Record<string, any>): boolean {
    try {
      for (const slot of template.slots) {
        const value = slotValues[slot.id];

        // Check required slots
        if (slot.required && (value === undefined || value === null || value === '')) {
          throw new PluginError(
            ErrorType.VALIDATION_ERROR,
            'REQUIRED_SLOT_MISSING',
            { slotId: slot.id, slotName: slot.name },
            false
          );
        }

        // Validate slot value if present
        if (value !== undefined && slot.validation) {
          if (!this.validateSlotValue(value, slot.validation)) {
            throw new PluginError(
              ErrorType.VALIDATION_ERROR,
              'SLOT_VALIDATION_FAILED',
              { slotId: slot.id, value, validation: slot.validation },
              false
            );
          }
        }
      }

      return true;
    } catch (error) {
      this.logger.error('Slot validation failed', error instanceof Error ? error : new Error(String(error)));
      throw error;
    }
  }

  // Private helper methods

  private validateTemplateStructure(structure: TemplateStructure): void {
    if (!structure.layout || !structure.sections) {
      throw new PluginError(
        ErrorType.VALIDATION_ERROR,
        'INVALID_TEMPLATE_STRUCTURE',
        { structure },
        false
      );
    }

    // Validate sections
    for (const section of structure.sections) {
      if (!section.id || !section.title || !section.content) {
        throw new PluginError(
          ErrorType.VALIDATION_ERROR,
          'INVALID_TEMPLATE_SECTION',
          { section },
          false
        );
      }
    }
  }

  private extractSlotsFromStructure(structure: TemplateStructure): TemplateSlot[] {
    const slots: TemplateSlot[] = [];
    const slotPattern = /\{\{(\w+)(?::(\w+))?(?:\|(.*?))?\}\}/g;

    for (const section of structure.sections) {
      let match;
      while ((match = slotPattern.exec(section.content)) !== null) {
        const [, name, type = 'text', options = ''] = match;
        
        const slot: TemplateSlot = {
          id: this.generateSlotId(name),
          name,
          type: type as any,
          required: options.includes('required'),
          defaultValue: this.extractDefaultValue(options),
          validation: this.parseValidationRules(options)
        };

        slots.push(slot);
      }
    }

    return slots;
  }

  private extractSlotsFromContent(content: string): TemplateSlot[] {
    const slots: TemplateSlot[] = [];
    const slotPattern = /\{\{(\w+)(?::(\w+))?(?:\|(.*?))?\}\}/g;

    let match;
    while ((match = slotPattern.exec(content)) !== null) {
      const [, name, type = 'text', options = ''] = match;
      
      const slot: TemplateSlot = {
        id: this.generateSlotId(name),
        name,
        type: type as any,
        required: options.includes('required'),
        defaultValue: this.extractDefaultValue(options),
        validation: this.parseValidationRules(options)
      };

      slots.push(slot);
    }

    return slots;
  }

  private parseTemplateStructure(content: string): TemplateStructure {
    const sections: any[] = [];
    const lines = content.split('\n');
    let currentSection: any = null;

    for (let i = 0; i < lines.length; i++) {
      const line = lines[i];
      
      // Check for section headers (## Section Name)
      const headerMatch = line.match(/^##\s+(.+)$/);
      if (headerMatch) {
        if (currentSection) {
          sections.push(currentSection);
        }
        
        currentSection = {
          id: this.generateSectionId(headerMatch[1]),
          title: headerMatch[1],
          content: '',
          slots: []
        };
      } else if (currentSection) {
        currentSection.content += line + '\n';
      }
    }

    if (currentSection) {
      sections.push(currentSection);
    }

    return {
      layout: 'standard',
      sections
    };
  }

  private extractTemplateMetadata(content: string): any {
    const frontmatterMatch = content.match(/^---\n(.*?)\n---/s);
    if (frontmatterMatch) {
      try {
        // Simple YAML parsing for basic metadata
        const frontmatter = frontmatterMatch[1];
        const metadata: any = {};
        
        const lines = frontmatter.split('\n');
        for (const line of lines) {
          const match = line.match(/^(\w+):\s*(.+)$/);
          if (match) {
            const [, key, value] = match;
            metadata[key] = value.replace(/^["']|["']$/g, ''); // Remove quotes
          }
        }
        
        return metadata;
      } catch (error) {
        this.logger.warn('Failed to parse frontmatter', error);
      }
    }

    return {};
  }

  private generateTemplateContent(template: Template): string {
    let content = '';

    // Add template header
    content += `# ${template.name}\n\n`;
    
    if (template.metadata.description) {
      content += `${template.metadata.description}\n\n`;
    }

    // Add sections
    for (const section of template.structure.sections) {
      content += `## ${section.title}\n\n`;
      content += `${section.content}\n\n`;
    }

    return content;
  }

  private getDefaultSlotValues(slots: TemplateSlot[]): Record<string, any> {
    const values: Record<string, any> = {};
    
    for (const slot of slots) {
      if (slot.defaultValue !== undefined) {
        values[slot.id] = slot.defaultValue;
      }
    }

    return values;
  }

  private validateSlotValue(value: any, validation: ValidationRule): boolean {
    if (validation.pattern && typeof value === 'string') {
      const regex = new RegExp(validation.pattern);
      if (!regex.test(value)) return false;
    }

    if (validation.minLength && typeof value === 'string') {
      if (value.length < validation.minLength) return false;
    }

    if (validation.maxLength && typeof value === 'string') {
      if (value.length > validation.maxLength) return false;
    }

    if (validation.customValidator) {
      return validation.customValidator(value);
    }

    return true;
  }

  private generateTemplateId(name: string): string {
    const timestamp = Date.now();
    const sanitized = name.toLowerCase().replace(/[^a-z0-9]/g, '-');
    return `template-${sanitized}-${timestamp}`;
  }

  private generateInstanceId(): string {
    return `instance-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
  }

  private generateSlotId(name: string): string {
    return `slot-${name.toLowerCase().replace(/[^a-z0-9]/g, '-')}`;
  }

  private generateSectionId(title: string): string {
    return `section-${title.toLowerCase().replace(/[^a-z0-9]/g, '-')}`;
  }

  private extractDefaultValue(options: string): any {
    const defaultMatch = options.match(/default:([^,]+)/);
    return defaultMatch ? defaultMatch[1].trim() : undefined;
  }

  private parseValidationRules(options: string): ValidationRule | undefined {
    if (!options) return undefined;

    const validation: ValidationRule = {};

    const patternMatch = options.match(/pattern:([^,]+)/);
    if (patternMatch) {
      validation.pattern = patternMatch[1].trim();
    }

    const minLengthMatch = options.match(/minLength:(\d+)/);
    if (minLengthMatch) {
      validation.minLength = parseInt(minLengthMatch[1]);
    }

    const maxLengthMatch = options.match(/maxLength:(\d+)/);
    if (maxLengthMatch) {
      validation.maxLength = parseInt(maxLengthMatch[1]);
    }

    validation.required = options.includes('required');

    return Object.keys(validation).length > 0 ? validation : undefined;
  }
}