/**
 * Template Parser utility for parsing and processing template content
 * Supports slot system and template structure definition
 */

import {
  Template,
  TemplateSlot,
  TemplateStructure,
  TemplateSection,
  ValidationRule,
  UnifiedChunk,
  Position
} from '../types';
import { ILogger } from '../interfaces';

export class TemplateParser {
  private logger: ILogger;

  constructor(logger: ILogger) {
    this.logger = logger;
  }

  /**
   * Parse template content and extract slots
   */
  parseTemplateContent(content: string, filePath: string): {
    structure: TemplateStructure;
    slots: TemplateSlot[];
    chunks: UnifiedChunk[];
  } {
    try {
      this.logger.debug(`Parsing template content from: ${filePath}`);

      // Parse structure
      const structure = this.parseStructure(content);
      
      // Extract slots
      const slots = this.extractSlots(content);
      
      // Create chunks for template parts
      const chunks = this.createTemplateChunks(content, filePath, structure);

      return { structure, slots, chunks };

    } catch (error) {
      this.logger.error('Failed to parse template content', error instanceof Error ? error : new Error(String(error)));
      throw error;
    }
  }

  /**
   * Apply slot values to template content
   */
  applySlotValues(templateContent: string, slotValues: Record<string, any>): string {
    let result = templateContent;

    // Replace slot placeholders with actual values
    const slotPattern = /\{\{(\w+)(?::(\w+))?(?:\|(.*?))?\}\}/g;
    
    result = result.replace(slotPattern, (match, name, type, options) => {
      const slotId = `slot-${name.toLowerCase().replace(/[^a-z0-9]/g, '-')}`;
      const value = slotValues[slotId];
      
      if (value !== undefined && value !== null) {
        return this.formatSlotValue(value, type);
      }
      
      // Return default value or placeholder
      const defaultValue = this.extractDefaultFromOptions(options);
      return defaultValue || `[${name}]`;
    });

    return result;
  }

  /**
   * Extract slot definitions from template content
   */
  extractSlotDefinitions(content: string): TemplateSlot[] {
    const slots: TemplateSlot[] = [];
    const slotPattern = /\{\{(\w+)(?::(\w+))?(?:\|(.*?))?\}\}/g;
    const seenSlots = new Set<string>();

    let match;
    while ((match = slotPattern.exec(content)) !== null) {
      const [, name, type = 'text', options = ''] = match;
      const slotId = `slot-${name.toLowerCase().replace(/[^a-z0-9]/g, '-')}`;
      
      // Avoid duplicate slots
      if (seenSlots.has(slotId)) continue;
      seenSlots.add(slotId);

      const slot: TemplateSlot = {
        id: slotId,
        name,
        type: this.parseSlotType(type),
        required: options.includes('required'),
        defaultValue: this.extractDefaultFromOptions(options),
        validation: this.parseValidationFromOptions(options)
      };

      slots.push(slot);
    }

    return slots;
  }

  /**
   * Validate template syntax
   */
  validateTemplateSyntax(content: string): { valid: boolean; errors: string[] } {
    const errors: string[] = [];

    try {
      // Check for unclosed slot tags
      const openTags = (content.match(/\{\{/g) || []).length;
      const closeTags = (content.match(/\}\}/g) || []).length;
      
      if (openTags !== closeTags) {
        errors.push('Mismatched slot tags: unclosed {{ or }}');
      }

      // Check for invalid slot syntax - find all potential slots
      const potentialSlots = content.match(/\{\{[^}]*\}\}/g) || [];
      const slotPattern = /\{\{(\w+)(?::(\w+))?(?:\|(.*?))?\}\}/;
      
      for (const slot of potentialSlots) {
        if (!slotPattern.test(slot)) {
          errors.push(`Invalid slot syntax: ${slot}`);
        }
      }

      // Check for duplicate slot names by parsing all slots without deduplication
      const allSlotMatches = content.match(/\{\{(\w+)(?::(\w+))?(?:\|(.*?))?\}\}/g) || [];
      const allSlotNames = allSlotMatches.map(match => {
        const nameMatch = match.match(/\{\{(\w+)/);
        return nameMatch ? nameMatch[1] : '';
      }).filter(name => name);
      
      const uniqueNames = new Set(allSlotNames);
      if (allSlotNames.length !== uniqueNames.size) {
        const duplicates = allSlotNames.filter((name, index) => allSlotNames.indexOf(name) !== index);
        const uniqueDuplicates = Array.from(new Set(duplicates));
        errors.push(`Duplicate slot names: ${uniqueDuplicates.join(', ')}`);
      }

      return { valid: errors.length === 0, errors };

    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : String(error);
      errors.push(`Syntax validation error: ${errorMessage}`);
      return { valid: false, errors };
    }
  }

  /**
   * Convert template to UnifiedChunk format
   */
  templateToChunks(template: Template, filePath: string): UnifiedChunk[] {
    const chunks: UnifiedChunk[] = [];
    const basePosition: Position = {
      fileName: filePath,
      lineStart: 1,
      lineEnd: 1,
      charStart: 0,
      charEnd: 0
    };

    // Create main template chunk
    const mainChunk: UnifiedChunk = {
      chunkId: template.id,
      contents: template.name,
      parent: undefined,
      page: filePath,
      isPage: false,
      isTag: false,
      isTemplate: true,
      isSlot: false,
      ref: template.id,
      tags: template.metadata.tags,
      metadata: {
        templateName: template.name,
        templateDescription: template.metadata.description,
        templateCategory: template.metadata.category,
        slotCount: template.slots.length,
        sectionCount: template.structure.sections.length
      },
      createdTime: template.metadata.createdTime,
      lastUpdated: template.metadata.lastUpdated,
      position: basePosition,
      filePath,
      obsidianMetadata: {
        properties: {},
        frontmatter: {},
        aliases: [],
        cssClasses: ['template']
      }
    };

    chunks.push(mainChunk);

    // Create chunks for each slot
    template.slots.forEach((slot, index) => {
      const slotChunk: UnifiedChunk = {
        chunkId: `${template.id}-${slot.id}`,
        contents: slot.name,
        parent: template.id,
        page: filePath,
        isPage: false,
        isTag: false,
        isTemplate: false,
        isSlot: true,
        ref: slot.id,
        tags: [],
        metadata: {
          slotType: slot.type,
          slotRequired: slot.required,
          slotDefaultValue: slot.defaultValue,
          slotValidation: slot.validation
        },
        createdTime: template.metadata.createdTime,
        lastUpdated: template.metadata.lastUpdated,
              documentId: 'test-doc-1',
      documentScope: 'file' as const,
      position: {
          ...basePosition,
          lineStart: index + 2,
          lineEnd: index + 2
        },
        filePath,
        obsidianMetadata: {
          properties: {
            slotType: slot.type,
            required: slot.required
          },
          frontmatter: {},
          aliases: [],
          cssClasses: ['template-slot']
        }
      };

      chunks.push(slotChunk);
    });

    return chunks;
  }

  // Private helper methods

  private parseStructure(content: string): TemplateStructure {
    const sections: TemplateSection[] = [];
    const lines = content.split('\n');
    let currentSection: Partial<TemplateSection> | null = null;
    let lineNumber = 0;

    for (const line of lines) {
      lineNumber++;
      
      // Check for section headers (## Section Name)
      const headerMatch = line.match(/^##\s+(.+)$/);
      if (headerMatch) {
        // Save previous section
        if (currentSection && currentSection.title) {
          sections.push({
            id: this.generateSectionId(currentSection.title),
            title: currentSection.title,
            content: currentSection.content || '',
            slots: this.extractSlotsFromText(currentSection.content || '')
          });
        }
        
        // Start new section
        currentSection = {
          title: headerMatch[1].trim(),
          content: ''
        };
      } else if (currentSection) {
        // Add line to current section
        currentSection.content = (currentSection.content || '') + line + '\n';
      }
    }

    // Add final section
    if (currentSection && currentSection.title) {
      sections.push({
        id: this.generateSectionId(currentSection.title),
        title: currentSection.title,
        content: currentSection.content || '',
        slots: this.extractSlotsFromText(currentSection.content || '')
      });
    }

    return {
      layout: 'standard',
      sections
    };
  }

  private extractSlots(content: string): TemplateSlot[] {
    return this.extractSlotDefinitions(content);
  }

  private createTemplateChunks(content: string, filePath: string, structure: TemplateStructure): UnifiedChunk[] {
    const chunks: UnifiedChunk[] = [];
    let lineNumber = 1;

    structure.sections.forEach((section, index) => {
      const chunk: UnifiedChunk = {
        chunkId: `section-${section.id}`,
        contents: section.content,
        parent: undefined,
        page: filePath,
        isPage: false,
        isTag: false,
        isTemplate: true,
        isSlot: false,
        ref: section.id,
        tags: [],
        metadata: {
          sectionTitle: section.title,
          sectionIndex: index,
          slotCount: section.slots.length
        },
        createdTime: new Date(),
        lastUpdated: new Date(),
              documentId: 'test-doc-1',
      documentScope: 'file' as const,
      position: {
          fileName: filePath,
          lineStart: lineNumber,
          lineEnd: lineNumber + section.content.split('\n').length,
          charStart: 0,
          charEnd: section.content.length
        },
        filePath,
        obsidianMetadata: {
          properties: {
            sectionTitle: section.title
          },
          frontmatter: {},
          aliases: [],
          cssClasses: ['template-section']
        }
      };

      chunks.push(chunk);
      lineNumber += section.content.split('\n').length + 2; // +2 for header and spacing
    });

    return chunks;
  }

  private parseSlotType(type: string): 'text' | 'number' | 'date' | 'link' | 'tag' {
    const validTypes = ['text', 'number', 'date', 'link', 'tag'];
    return validTypes.includes(type) ? type as any : 'text';
  }

  private extractDefaultFromOptions(options: string): any {
    if (!options) return undefined;
    
    const defaultMatch = options.match(/default:([^,]+)/);
    if (defaultMatch) {
      const value = defaultMatch[1].trim();
      // Try to parse as number if it looks like one
      if (/^\d+$/.test(value)) return parseInt(value);
      if (/^\d+\.\d+$/.test(value)) return parseFloat(value);
      return value;
    }
    
    return undefined;
  }

  private parseValidationFromOptions(options: string): ValidationRule | undefined {
    if (!options) return undefined;

    const validation: ValidationRule = {};

    // Extract pattern
    const patternMatch = options.match(/pattern:([^,]+)/);
    if (patternMatch) {
      validation.pattern = patternMatch[1].trim();
    }

    // Extract length constraints
    const minLengthMatch = options.match(/minLength:(\d+)/);
    if (minLengthMatch) {
      validation.minLength = parseInt(minLengthMatch[1]);
    }

    const maxLengthMatch = options.match(/maxLength:(\d+)/);
    if (maxLengthMatch) {
      validation.maxLength = parseInt(maxLengthMatch[1]);
    }

    // Check required flag
    validation.required = options.includes('required');

    return Object.keys(validation).length > 0 ? validation : undefined;
  }

  private formatSlotValue(value: any, type?: string): string {
    switch (type) {
      case 'date':
        if (value instanceof Date) {
          return value.toISOString().split('T')[0];
        }
        return String(value);
      
      case 'number':
        return String(Number(value));
      
      case 'link':
        if (typeof value === 'string' && !value.startsWith('[[')) {
          return `[[${value}]]`;
        }
        return String(value);
      
      case 'tag':
        if (typeof value === 'string' && !value.startsWith('#')) {
          return `#${value}`;
        }
        return String(value);
      
      default:
        return String(value);
    }
  }

  private extractSlotsFromText(text: string): string[] {
    const slots: string[] = [];
    const slotPattern = /\{\{(\w+)(?::(\w+))?(?:\|(.*?))?\}\}/g;

    let match;
    while ((match = slotPattern.exec(text)) !== null) {
      slots.push(match[1]);
    }

    return slots;
  }

  private generateSectionId(title: string): string {
    return title.toLowerCase()
      .replace(/[^a-z0-9\s]/g, '')
      .replace(/\s+/g, '-')
      .replace(/^-+|-+$/g, '');
  }
}