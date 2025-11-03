/**
 * Unit tests for TemplateManager
 * Tests template creation, application, parsing, and management functionality
 */

import { TemplateManager } from '../TemplateManager';
import { IInkGatewayClient, ILogger } from '../../interfaces';
import {
  Template,
  TemplateInstance,
  TemplateStructure,
  TemplateSlot,
  PluginError,
  ErrorType
} from '../../types';

// Mock implementations
class MockInkGatewayClient implements Partial<IInkGatewayClient> {
  private templates: Map<string, Template> = new Map();
  private instances: Map<string, TemplateInstance[]> = new Map();

  async createTemplate(template: Template): Promise<Template> {
    this.templates.set(template.id, template);
    this.instances.set(template.id, []);
    return template;
  }

  async getTemplateInstances(templateId: string): Promise<TemplateInstance[]> {
    return this.instances.get(templateId) || [];
  }

  // Add methods to control mock behavior
  setMockError(error: Error) {
    this.createTemplate = async () => { throw error; };
  }

  clearMockError() {
    this.createTemplate = async (template: Template) => {
      this.templates.set(template.id, template);
      return template;
    };
  }
}

class MockLogger implements ILogger {
  debug = jest.fn();
  info = jest.fn();
  warn = jest.fn();
  error = jest.fn();
}

describe('TemplateManager', () => {
  let templateManager: TemplateManager;
  let mockApiClient: MockInkGatewayClient;
  let mockLogger: MockLogger;

  beforeEach(() => {
    mockApiClient = new MockInkGatewayClient();
    mockLogger = new MockLogger();
    templateManager = new TemplateManager(mockApiClient as any, mockLogger);
  });

  describe('createTemplate', () => {
    it('should create a template successfully', async () => {
      const structure: TemplateStructure = {
        layout: 'standard',
        sections: [
          {
            id: 'section-1',
            title: 'Basic Info',
            content: 'Name: {{name:text|required}}\nAge: {{age:number}}',
            slots: ['name', 'age']
          }
        ]
      };

      const template = await templateManager.createTemplate('Contact Template', structure);

      expect(template).toBeDefined();
      expect(template.name).toBe('Contact Template');
      expect(template.slots).toHaveLength(2);
      expect(template.slots[0].name).toBe('name');
      expect(template.slots[0].required).toBe(true);
      expect(template.slots[1].name).toBe('age');
      expect(template.slots[1].type).toBe('number');
    });

    it('should validate template structure', async () => {
      const invalidStructure: TemplateStructure = {
        layout: '',
        sections: []
      };

      await expect(
        templateManager.createTemplate('Invalid Template', invalidStructure)
      ).rejects.toThrow(PluginError);
    });

    it('should handle API errors gracefully', async () => {
      const structure: TemplateStructure = {
        layout: 'standard',
        sections: [
          {
            id: 'section-1',
            title: 'Test',
            content: 'Test content',
            slots: []
          }
        ]
      };

      mockApiClient.setMockError(new Error('API Error'));

      await expect(
        templateManager.createTemplate('Test Template', structure)
      ).rejects.toThrow(PluginError);

      expect(mockLogger.error).toHaveBeenCalled();
    });
  });

  describe('parseTemplateFromContent', () => {
    it('should parse template from markdown content', () => {
      const content = `---
name: Contact Template
description: A template for contact information
category: personal
---

# Contact Template

## Basic Information
Name: {{name:text|required}}
Email: {{email:text|pattern:^[^@]+@[^@]+\\.[^@]+$}}

## Details
Phone: {{phone:text}}
Notes: {{notes:text|default:No notes}}`;

      const template = templateManager.parseTemplateFromContent(content);

      expect(template.name).toBe('Contact Template');
      expect(template.metadata.description).toBe('A template for contact information');
      expect(template.metadata.category).toBe('personal');
      expect(template.slots).toHaveLength(4);
      
      const nameSlot = template.slots.find(s => s.name === 'name');
      expect(nameSlot?.required).toBe(true);
      
      const emailSlot = template.slots.find(s => s.name === 'email');
      expect(emailSlot?.validation?.pattern).toBe('^[^@]+@[^@]+\\.[^@]+$');
      
      const notesSlot = template.slots.find(s => s.name === 'notes');
      expect(notesSlot?.defaultValue).toBe('No notes');
    });

    it('should handle content without frontmatter', () => {
      const content = `# Simple Template

Name: {{name:text}}
Description: {{description:text}}`;

      const template = templateManager.parseTemplateFromContent(content);

      expect(template.name).toBe('Parsed Template');
      expect(template.slots).toHaveLength(2);
    });

    it('should handle parsing errors', () => {
      // Mock the parseTemplateStructure method to throw an error
      const originalParseTemplateStructure = (templateManager as any).parseTemplateStructure;
      (templateManager as any).parseTemplateStructure = jest.fn().mockImplementation(() => {
        throw new Error('Parsing failed');
      });

      const invalidContent = `# Invalid Template

Unclosed slot: {{name`;

      expect(() => {
        templateManager.parseTemplateFromContent(invalidContent);
      }).toThrow(PluginError);

      // Restore original method
      (templateManager as any).parseTemplateStructure = originalParseTemplateStructure;
    });
  });

  describe('applyTemplate', () => {
    it('should apply template to target file', async () => {
      // First create a template
      const structure: TemplateStructure = {
        layout: 'standard',
        sections: [
          {
            id: 'section-1',
            title: 'Info',
            content: 'Name: {{name:text}}',
            slots: ['name']
          }
        ]
      };

      const template = await templateManager.createTemplate('Test Template', structure);

      const mockFile = {
        path: '/test/file.md',
        name: 'file.md'
      } as any;

      await templateManager.applyTemplate(template.id, mockFile);

      // Verify template was applied (instance created)
      const instances = await templateManager.getTemplateInstances(template.id);
      expect(instances).toHaveLength(1);
      expect(instances[0].filePath).toBe('/test/file.md');
    });

    it('should throw error for non-existent template', async () => {
      const mockFile = { path: '/test/file.md', name: 'file.md' } as any;

      await expect(
        templateManager.applyTemplate('non-existent-id', mockFile)
      ).rejects.toThrow(PluginError);
    });
  });

  describe('getTemplateInstances', () => {
    it('should return template instances', async () => {
      const structure: TemplateStructure = {
        layout: 'standard',
        sections: [
          {
            id: 'section-1',
            title: 'Test',
            content: 'Content',
            slots: []
          }
        ]
      };

      const template = await templateManager.createTemplate('Test Template', structure);
      const instances = await templateManager.getTemplateInstances(template.id);

      expect(instances).toEqual([]);
    });

    it('should handle API errors when fetching instances', async () => {
      mockApiClient.getTemplateInstances = async () => {
        throw new Error('API Error');
      };

      await expect(
        templateManager.getTemplateInstances('test-id')
      ).rejects.toThrow(PluginError);
    });
  });

  describe('updateTemplate', () => {
    it('should update existing template', async () => {
      const structure: TemplateStructure = {
        layout: 'standard',
        sections: [
          {
            id: 'section-1',
            title: 'Test',
            content: 'Content',
            slots: []
          }
        ]
      };

      const template = await templateManager.createTemplate('Test Template', structure);
      
      const updates = {
        name: 'Updated Template',
        metadata: {
          ...template.metadata,
          description: 'Updated description'
        }
      };

      const updatedTemplate = await templateManager.updateTemplate(template.id, updates);

      expect(updatedTemplate.name).toBe('Updated Template');
      expect(updatedTemplate.metadata.description).toBe('Updated description');
      expect(updatedTemplate.metadata.lastUpdated).toBeInstanceOf(Date);
    });

    it('should throw error for non-existent template', async () => {
      await expect(
        templateManager.updateTemplate('non-existent-id', { name: 'Updated' })
      ).rejects.toThrow(PluginError);
    });
  });

  describe('deleteTemplate', () => {
    it('should delete existing template', async () => {
      const structure: TemplateStructure = {
        layout: 'standard',
        sections: [
          {
            id: 'section-1',
            title: 'Test',
            content: 'Content',
            slots: []
          }
        ]
      };

      const template = await templateManager.createTemplate('Test Template', structure);
      
      await templateManager.deleteTemplate(template.id);

      expect(templateManager.getTemplate(template.id)).toBeUndefined();
    });

    it('should throw error for non-existent template', async () => {
      await expect(
        templateManager.deleteTemplate('non-existent-id')
      ).rejects.toThrow(PluginError);
    });
  });

  describe('validateSlotValues', () => {
    it('should validate required slots', async () => {
      const structure: TemplateStructure = {
        layout: 'standard',
        sections: [
          {
            id: 'section-1',
            title: 'Test',
            content: 'Name: {{name:text|required}}',
            slots: ['name']
          }
        ]
      };

      const template = await templateManager.createTemplate('Test Template', structure);
      
      // Should pass with required value
      expect(
        templateManager.validateSlotValues(template, { 'slot-name': 'John Doe' })
      ).toBe(true);

      // Should fail without required value
      expect(() => {
        templateManager.validateSlotValues(template, {});
      }).toThrow(PluginError);
    });

    it('should validate slot patterns', async () => {
      const structure: TemplateStructure = {
        layout: 'standard',
        sections: [
          {
            id: 'section-1',
            title: 'Test',
            content: 'Email: {{email:text|pattern:^[^@]+@[^@]+\\.[^@]+$}}',
            slots: ['email']
          }
        ]
      };

      const template = await templateManager.createTemplate('Test Template', structure);
      
      // Should pass with valid email
      expect(
        templateManager.validateSlotValues(template, { 'slot-email': 'test@example.com' })
      ).toBe(true);

      // Should fail with invalid email
      expect(() => {
        templateManager.validateSlotValues(template, { 'slot-email': 'invalid-email' });
      }).toThrow(PluginError);
    });

    it('should validate slot length constraints', async () => {
      const structure: TemplateStructure = {
        layout: 'standard',
        sections: [
          {
            id: 'section-1',
            title: 'Test',
            content: 'Name: {{name:text|minLength:2,maxLength:50}}',
            slots: ['name']
          }
        ]
      };

      const template = await templateManager.createTemplate('Test Template', structure);
      
      // Should pass with valid length
      expect(
        templateManager.validateSlotValues(template, { 'slot-name': 'John' })
      ).toBe(true);

      // Should fail with too short value
      expect(() => {
        templateManager.validateSlotValues(template, { 'slot-name': 'J' });
      }).toThrow(PluginError);

      // Should fail with too long value
      expect(() => {
        templateManager.validateSlotValues(template, { 
          'slot-name': 'A'.repeat(51) 
        });
      }).toThrow(PluginError);
    });
  });

  describe('getTemplates', () => {
    it('should return all templates', async () => {
      const structure: TemplateStructure = {
        layout: 'standard',
        sections: [
          {
            id: 'section-1',
            title: 'Test',
            content: 'Content',
            slots: []
          }
        ]
      };

      await templateManager.createTemplate('Template 1', structure);
      await templateManager.createTemplate('Template 2', structure);

      const templates = templateManager.getTemplates();
      expect(templates).toHaveLength(2);
    });
  });

  describe('getTemplate', () => {
    it('should return specific template', async () => {
      const structure: TemplateStructure = {
        layout: 'standard',
        sections: [
          {
            id: 'section-1',
            title: 'Test',
            content: 'Content',
            slots: []
          }
        ]
      };

      const template = await templateManager.createTemplate('Test Template', structure);
      const retrieved = templateManager.getTemplate(template.id);

      expect(retrieved).toBeDefined();
      expect(retrieved?.name).toBe('Test Template');
    });

    it('should return undefined for non-existent template', () => {
      const retrieved = templateManager.getTemplate('non-existent-id');
      expect(retrieved).toBeUndefined();
    });
  });
});