/**
 * Integration tests for Template System
 * Tests the complete template workflow including property mapping and synchronization
 */

import { TFile, CachedMetadata, Vault, MetadataCache } from 'obsidian';
import { TemplateManager } from '../TemplateManager';
import { PropertyMapper } from '../PropertyMapper';
import { TemplateValidator } from '../TemplateValidator';
import { TemplateSyncManager } from '../TemplateSyncManager';
import { IInkGatewayClient, ILogger } from '../../interfaces';
import {
  Template,
  TemplateInstance,
  TemplateStructure,
  UnifiedChunk
} from '../../types';

// Mock implementations
class MockInkGatewayClient implements Partial<IInkGatewayClient> {
  private chunks: Map<string, UnifiedChunk> = new Map();
  private templates: Map<string, Template> = new Map();
  private instances: Map<string, TemplateInstance[]> = new Map();

  async createChunk(chunk: UnifiedChunk): Promise<UnifiedChunk> {
    this.chunks.set(chunk.chunkId, chunk);
    return chunk;
  }

  async updateChunk(id: string, chunk: Partial<UnifiedChunk>): Promise<UnifiedChunk> {
    const existing = this.chunks.get(id);
    if (!existing) throw new Error('Chunk not found');
    const updated = { ...existing, ...chunk };
    this.chunks.set(id, updated);
    return updated;
  }

  async getChunk(id: string): Promise<UnifiedChunk> {
    const chunk = this.chunks.get(id);
    if (!chunk) throw new Error('Chunk not found');
    return chunk;
  }

  async createTemplate(template: Template): Promise<Template> {
    this.templates.set(template.id, template);
    this.instances.set(template.id, []);
    return template;
  }

  async getTemplateInstances(templateId: string): Promise<TemplateInstance[]> {
    return this.instances.get(templateId) || [];
  }

  // Helper methods for testing
  getChunkSync(id: string): UnifiedChunk | undefined {
    return this.chunks.get(id);
  }

  getAllChunks(): UnifiedChunk[] {
    return Array.from(this.chunks.values());
  }
}

class MockLogger implements ILogger {
  debug = jest.fn();
  info = jest.fn();
  warn = jest.fn();
  error = jest.fn();
}

class MockVault implements Partial<Vault> {
  private files: Map<string, string> = new Map();

  async read(file: TFile): Promise<string> {
    return this.files.get(file.path) || '';
  }

  async modify(file: TFile, data: string): Promise<void> {
    this.files.set(file.path, data);
  }

  getAbstractFileByPath(path: string): TFile | null {
    if (this.files.has(path)) {
      return {
        path,
        name: path.split('/').pop() || '',
        basename: path.split('/').pop()?.replace(/\.[^.]+$/, '') || '',
        stat: { ctime: Date.now(), mtime: Date.now(), size: 0 }
      } as TFile;
    }
    return null;
  }

  setFileContent(path: string, content: string): void {
    this.files.set(path, content);
  }
}

class MockMetadataCache implements Partial<MetadataCache> {
  private metadata: Map<string, CachedMetadata> = new Map();

  getFileCache(file: TFile): CachedMetadata | null {
    return this.metadata.get(file.path) || null;
  }

  setFileMetadata(path: string, metadata: CachedMetadata): void {
    this.metadata.set(path, metadata);
  }
}

describe('Template System Integration', () => {
  let templateManager: TemplateManager;
  let propertyMapper: PropertyMapper;
  let validator: TemplateValidator;
  let syncManager: TemplateSyncManager;
  let mockApiClient: MockInkGatewayClient;
  let mockLogger: MockLogger;
  let mockVault: MockVault;
  let mockMetadataCache: MockMetadataCache;

  beforeEach(() => {
    mockApiClient = new MockInkGatewayClient();
    mockLogger = new MockLogger();
    mockVault = new MockVault();
    mockMetadataCache = new MockMetadataCache();
    
    templateManager = new TemplateManager(mockApiClient as any, mockLogger);
    propertyMapper = new PropertyMapper(mockLogger);
    validator = new TemplateValidator(mockLogger);
    syncManager = new TemplateSyncManager(mockApiClient as any, mockLogger, mockVault as any, mockMetadataCache as any);
  });

  describe('Complete Template Workflow', () => {
    it('should create, apply, and synchronize a template', async () => {
      // Step 1: Create a template
      const structure: TemplateStructure = {
        layout: 'standard',
        sections: [
          {
            id: 'contact-info',
            title: 'Contact Information',
            content: 'Name: {{name:text|required}}\nEmail: {{email:text|pattern:^[^@]+@[^@]+\\.[^@]+$}}\nPhone: {{phone:text|default:Not provided}}',
            slots: ['name', 'email', 'phone']
          },
          {
            id: 'notes',
            title: 'Notes',
            content: 'Additional notes: {{notes:text}}',
            slots: ['notes']
          }
        ]
      };

      const template = await templateManager.createTemplate('Contact Template', structure);
      expect(template).toBeDefined();
      expect(template.slots).toHaveLength(4);

      // Step 2: Create property mappings
      const mappings = propertyMapper.createPropertyMappings(template);
      expect(mappings).toHaveLength(4);
      expect(mappings.find(m => m.slotName === 'name')?.propertyName).toBe('name');
      expect(mappings.find(m => m.slotName === 'email')?.propertyName).toBe('email');

      // Step 3: Apply template to a file
      const filePath = '/contacts/john-doe.md';
      const mockFile = {
        path: filePath,
        name: 'john-doe.md',
        basename: 'john-doe',
        stat: { ctime: Date.now(), mtime: Date.now(), size: 0 }
      } as TFile;

      await templateManager.applyTemplate(template.id, mockFile);

      // Step 4: Get template instance and populate with values
      const instances = await templateManager.getTemplateInstances(template.id);
      expect(instances).toHaveLength(1);

      const instance = instances[0];
      instance.slotValues = {
        'slot-name': 'John Doe',
        'slot-email': 'john@example.com',
        'slot-phone': '123-456-7890',
        'slot-notes': 'Important client'
      };

      // Step 5: Validate the instance
      const validationResult = validator.validateTemplateInstance(template, instance);
      expect(validationResult.valid).toBe(true);
      expect(validationResult.errors).toHaveLength(0);

      // Step 6: Apply slot values to properties
      const initialContent = `# Contact Template

## Contact Information
Name: {{name:text|required}}
Email: {{email:text}}
Phone: {{phone:text|default:Not provided}}

## Notes
Additional notes: {{notes:text}}`;

      mockVault.setFileContent(filePath, initialContent);

      const { content: updatedContent, metadata } = await propertyMapper.applySlotValuesToProperties(
        template,
        instance,
        initialContent
      );

      expect(updatedContent).toContain('name: "John Doe"');
      expect(updatedContent).toContain('email: "john@example.com"');
      expect(updatedContent).toContain('phone: "123-456-7890"');
      expect(updatedContent).toContain('template: "Contact Template"');
      expect(metadata.properties.name).toBe('John Doe');

      // Step 7: Synchronize with Ink-Gateway
      mockVault.setFileContent(filePath, updatedContent);

      const syncResult = await syncManager.synchronizeTemplateInstance(
        template,
        instance,
        mockFile,
        {
          validateBeforeSync: true,
          autoFillMissingSlots: false,
          preserveUserContent: true,
          syncToInkGateway: true
        }
      );

      expect(syncResult.success).toBe(true);
      expect(syncResult.syncedChunks).toBeGreaterThan(0);
      expect(syncResult.validationResult?.valid).toBe(true);

      // Verify chunks were created in mock API client
      const chunks = mockApiClient.getAllChunks();
      expect(chunks.length).toBeGreaterThan(0);
      
      const mainChunk = chunks.find(c => c.chunkId === instance.id);
      expect(mainChunk).toBeDefined();
      expect(mainChunk?.metadata.templateId).toBe(template.id);
    });

    it('should handle template updates and instance synchronization', async () => {
      // Create initial template
      const structure: TemplateStructure = {
        layout: 'standard',
        sections: [
          {
            id: 'basic',
            title: 'Basic Info',
            content: 'Name: {{name:text|required}}',
            slots: ['name']
          }
        ]
      };

      const template = await templateManager.createTemplate('Simple Template', structure);
      
      // Create instance
      const filePath = '/test/simple.md';
      const mockFile = {
        path: filePath,
        name: 'simple.md',
        basename: 'simple',
        stat: { ctime: Date.now(), mtime: Date.now(), size: 0 }
      } as TFile;

      await templateManager.applyTemplate(template.id, mockFile);
      const instances = await templateManager.getTemplateInstances(template.id);
      const instance = instances[0];
      instance.slotValues = { 'slot-name': 'Test User' };

      // Update template with new slot
      const updatedStructure: TemplateStructure = {
        layout: 'standard',
        sections: [
          {
            id: 'basic',
            title: 'Basic Info',
            content: 'Name: {{name:text|required}}\nAge: {{age:number}}',
            slots: ['name', 'age']
          }
        ]
      };

      const updatedTemplate = await templateManager.updateTemplate(template.id, {
        structure: updatedStructure,
        slots: [
          ...template.slots,
          {
            id: 'slot-age',
            name: 'age',
            type: 'number',
            required: false,
            defaultValue: undefined,
            validation: undefined
          }
        ]
      });

      // Auto-fill new slot
      const autoFillResult = await validator.autoFillTemplate(updatedTemplate, instance);
      expect(autoFillResult.success).toBe(true);

      // Update instance synchronization
      mockVault.setFileContent(filePath, '# Simple Template\n\n## Basic Info\nName: Test User\nAge: 25');

      const syncResults = await syncManager.updateTemplateInstances(
        updatedTemplate,
        [instance],
        {
          validateBeforeSync: true,
          autoFillMissingSlots: true,
          preserveUserContent: true,
          syncToInkGateway: true
        }
      );

      expect(syncResults).toHaveLength(1);
      expect(syncResults[0].success).toBe(true);
    });

    it('should handle property synchronization conflicts', async () => {
      // Create template
      const structure: TemplateStructure = {
        layout: 'standard',
        sections: [
          {
            id: 'info',
            title: 'Information',
            content: 'Title: {{title:text|required}}\nStatus: {{status:text}}',
            slots: ['title', 'status']
          }
        ]
      };

      const template = await templateManager.createTemplate('Document Template', structure);
      
      // Create instance with values
      const filePath = '/docs/document.md';
      const mockFile = {
        path: filePath,
        name: 'document.md',
        basename: 'document',
        stat: { ctime: Date.now(), mtime: Date.now(), size: 0 }
      } as TFile;

      await templateManager.applyTemplate(template.id, mockFile);
      const instances = await templateManager.getTemplateInstances(template.id);
      const instance = instances[0];
      
      // Set instance values
      instance.slotValues = {
        'slot-title': 'Original Title',
        'slot-status': 'Draft'
      };

      // Set conflicting file metadata
      const conflictingMetadata: CachedMetadata = {
        frontmatter: {
          title: 'Modified Title',
          status: 'Published'
        }
      };

      mockMetadataCache.setFileMetadata(filePath, conflictingMetadata);
      mockVault.setFileContent(filePath, `---
title: Modified Title
status: Published
---

# Document Template

## Information
Title: Modified Title
Status: Published`);

      // Resolve conflicts with merge strategy
      const conflictResult = await syncManager.resolveTemplateConflicts(
        template,
        instance,
        mockFile,
        'merge'
      );

      expect(conflictResult.success).toBe(true);
      expect(conflictResult.conflicts.length).toBeGreaterThan(0);
      
      // Instance should now have merged values
      expect(instance.slotValues['slot-title']).toBe('Original Title'); // Keep instance value
      expect(instance.slotValues['slot-status']).toBe('Draft'); // Keep instance value
    });

    it('should validate template content and provide suggestions', async () => {
      // Create template with validation rules
      const structure: TemplateStructure = {
        layout: 'standard',
        sections: [
          {
            id: 'contact',
            title: 'Contact Details',
            content: 'Email: {{email:text|pattern:^[^@]+@[^@]+\\.[^@]+$,required}}\nWebsite: {{website:link}}',
            slots: ['email', 'website']
          }
        ]
      };

      const template = await templateManager.createTemplate('Contact Form', structure);
      
      // Create instance with invalid values
      const instance: TemplateInstance = {
        id: 'test-instance',
        templateId: template.id,
        filePath: '/test/contact.md',
        slotValues: {
          'slot-email': 'invalid-email', // Invalid email format
          'slot-website': 'example.com' // Missing link formatting
        },
        createdAt: new Date(),
        updatedAt: new Date()
      };

      // Validate instance
      const validationResult = validator.validateTemplateInstance(template, instance);
      
      expect(validationResult.valid).toBe(false);
      expect(validationResult.errors.some(e => e.slotName === 'email')).toBe(true);
      expect(validationResult.suggestions.some(s => s.slotName === 'website')).toBe(true);

      // Get improvement suggestions
      const suggestions = validator.suggestImprovements(template, instance);
      expect(suggestions.length).toBeGreaterThan(0);
      
      const websiteSuggestion = suggestions.find(s => s.slotName === 'website');
      expect(websiteSuggestion?.suggestedValue).toBe('[[example.com]]');
    });

    it('should handle batch synchronization of multiple instances', async () => {
      // Create multiple templates and instances
      const templates = new Map<string, Template>();
      const instances: TemplateInstance[] = [];

      // Template 1
      const structure1: TemplateStructure = {
        layout: 'standard',
        sections: [
          {
            id: 'info',
            title: 'Info',
            content: 'Name: {{name:text}}',
            slots: ['name']
          }
        ]
      };

      const template1 = await templateManager.createTemplate('Template 1', structure1);
      templates.set(template1.id, template1);

      // Template 2
      const structure2: TemplateStructure = {
        layout: 'standard',
        sections: [
          {
            id: 'details',
            title: 'Details',
            content: 'Title: {{title:text}}',
            slots: ['title']
          }
        ]
      };

      const template2 = await templateManager.createTemplate('Template 2', structure2);
      templates.set(template2.id, template2);

      // Create instances
      for (let i = 1; i <= 5; i++) {
        const template = i <= 3 ? template1 : template2;
        const filePath = `/batch/file${i}.md`;
        
        mockVault.setFileContent(filePath, `# Test File ${i}`);
        
        instances.push({
          id: `instance-${i}`,
          templateId: template.id,
          filePath,
          slotValues: {
            [`slot-${i <= 3 ? 'name' : 'title'}`]: `Value ${i}`
          },
          createdAt: new Date(),
          updatedAt: new Date()
        });
      }

      // Batch synchronize
      const results = await syncManager.batchSynchronizeInstances(
        templates,
        instances,
        {
          validateBeforeSync: true,
          autoFillMissingSlots: true,
          preserveUserContent: true,
          syncToInkGateway: true
        }
      );

      expect(results).toHaveLength(5);
      expect(results.filter(r => r.success)).toHaveLength(5);
      
      // Verify all instances were synchronized
      const chunks = mockApiClient.getAllChunks();
      expect(chunks.length).toBeGreaterThan(5); // Main chunks + slot chunks
    });
  });
});