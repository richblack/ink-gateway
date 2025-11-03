/**
 * Unit tests for TemplateRenderer
 * Tests template rendering, instance creation, and content generation
 */

import { TemplateRenderer } from '../TemplateRenderer';
import { ILogger } from '../../interfaces';
import { Template, TemplateInstance } from '../../types';

class MockLogger implements ILogger {
  debug = jest.fn();
  info = jest.fn();
  warn = jest.fn();
  error = jest.fn();
}

describe('TemplateRenderer', () => {
  let renderer: TemplateRenderer;
  let mockLogger: MockLogger;

  beforeEach(() => {
    mockLogger = new MockLogger();
    renderer = new TemplateRenderer(mockLogger);
  });

  const createSampleTemplate = (): Template => ({
    id: 'template-1',
    name: 'Contact Template',
    slots: [
      {
        id: 'slot-name',
        name: 'name',
        type: 'text',
        required: true,
        defaultValue: undefined,
        validation: undefined
      },
      {
        id: 'slot-email',
        name: 'email',
        type: 'text',
        required: false,
        defaultValue: undefined,
        validation: { pattern: '^[^@]+@[^@]+\\.[^@]+$' }
      },
      {
        id: 'slot-phone',
        name: 'phone',
        type: 'text',
        required: false,
        defaultValue: 'Not provided',
        validation: undefined
      }
    ],
    structure: {
      layout: 'standard',
      sections: [
        {
          id: 'basic-info',
          title: 'Basic Information',
          content: 'Name: {{name:text|required}}\nEmail: {{email:text}}',
          slots: ['name', 'email']
        },
        {
          id: 'contact-details',
          title: 'Contact Details',
          content: 'Phone: {{phone:text|default:Not provided}}',
          slots: ['phone']
        }
      ]
    },
    metadata: {
      description: 'A template for contact information',
      category: 'personal',
      tags: ['contact'],
      createdTime: new Date('2023-01-01'),
      lastUpdated: new Date('2023-01-01')
    }
  });

  describe('renderTemplate', () => {
    it('should render template with slot values', () => {
      const template = createSampleTemplate();
      const slotValues = {
        'slot-name': 'John Doe',
        'slot-email': 'john@example.com',
        'slot-phone': '123-456-7890'
      };

      const result = renderer.renderTemplate(template, slotValues);

      expect(result).toContain('# Contact Template');
      expect(result).toContain('A template for contact information');
      expect(result).toContain('## Basic Information');
      expect(result).toContain('Name: John Doe');
      expect(result).toContain('Email: john@example.com');
      expect(result).toContain('## Contact Details');
      expect(result).toContain('Phone: 123-456-7890');
      expect(result).toContain('template: Contact Template');
      expect(result).toContain('templateId: template-1');
    });

    it('should use default values for missing slots', () => {
      const template = createSampleTemplate();
      const slotValues = {
        'slot-name': 'Jane Doe'
        // email and phone missing
      };

      const result = renderer.renderTemplate(template, slotValues);

      expect(result).toContain('Name: Jane Doe');
      expect(result).toContain('Email: [email]'); // Placeholder for missing value
      expect(result).toContain('Phone: Not provided'); // Default value
    });

    it('should handle template without description', () => {
      const template = createSampleTemplate();
      template.metadata.description = undefined;

      const slotValues = {
        'slot-name': 'Test User'
      };

      const result = renderer.renderTemplate(template, slotValues);

      expect(result).toContain('# Contact Template');
      expect(result).not.toContain('A template for contact information');
      expect(result).toContain('Name: Test User');
    });

    it('should handle rendering errors', () => {
      const template = createSampleTemplate();
      // Simulate an error by making structure invalid
      template.structure.sections = null as any;

      expect(() => {
        renderer.renderTemplate(template, {});
      }).toThrow();

      expect(mockLogger.error).toHaveBeenCalled();
    });
  });

  describe('createTemplateInstance', () => {
    it('should create template instance', () => {
      const template = createSampleTemplate();
      const slotValues = {
        'slot-name': 'John Doe',
        'slot-email': 'john@example.com'
      };

      const instance = renderer.createTemplateInstance(
        template,
        '/notes/contact.md',
        slotValues
      );

      expect(instance.id).toBeDefined();
      expect(instance.templateId).toBe('template-1');
      expect(instance.filePath).toBe('/notes/contact.md');
      expect(instance.slotValues).toEqual(slotValues);
      expect(instance.createdAt).toBeInstanceOf(Date);
      expect(instance.updatedAt).toBeInstanceOf(Date);
    });

    it('should create instance with empty slot values', () => {
      const template = createSampleTemplate();
      const instance = renderer.createTemplateInstance(
        template,
        '/notes/empty.md',
        {}
      );

      expect(instance.slotValues).toEqual({});
      expect(instance.templateId).toBe('template-1');
    });
  });

  describe('updateTemplateInstance', () => {
    it('should update instance with new slot values', async () => {
      const template = createSampleTemplate();
      const originalInstance = renderer.createTemplateInstance(
        template,
        '/notes/contact.md',
        { 'slot-name': 'John Doe' }
      );

      // Add a small delay to ensure different timestamps
      await new Promise(resolve => setTimeout(resolve, 1));
      
      const updatedInstance = renderer.updateTemplateInstance(
        originalInstance,
        { 'slot-email': 'john@example.com' }
      );

      expect(updatedInstance.id).toBe(originalInstance.id);
      expect(updatedInstance.slotValues).toEqual({
        'slot-name': 'John Doe',
        'slot-email': 'john@example.com'
      });
      expect(updatedInstance.updatedAt.getTime()).toBeGreaterThanOrEqual(
        originalInstance.updatedAt.getTime()
      );
    });

    it('should overwrite existing slot values', () => {
      const template = createSampleTemplate();
      const originalInstance = renderer.createTemplateInstance(
        template,
        '/notes/contact.md',
        { 'slot-name': 'John Doe', 'slot-email': 'old@example.com' }
      );

      const updatedInstance = renderer.updateTemplateInstance(
        originalInstance,
        { 'slot-email': 'new@example.com' }
      );

      expect(updatedInstance.slotValues).toEqual({
        'slot-name': 'John Doe',
        'slot-email': 'new@example.com'
      });
    });
  });

  describe('instanceToChunks', () => {
    it('should convert instance to unified chunks', () => {
      const template = createSampleTemplate();
      const instance = renderer.createTemplateInstance(
        template,
        '/notes/contact.md',
        {
          'slot-name': 'John Doe',
          'slot-email': 'john@example.com',
          'slot-phone': '123-456-7890'
        }
      );

      const chunks = renderer.instanceToChunks(template, instance);

      expect(chunks).toHaveLength(4); // 1 main + 3 populated slots
      
      const mainChunk = chunks[0];
      expect(mainChunk.chunkId).toBe(instance.id);
      expect(mainChunk.isTemplate).toBe(false);
      expect(mainChunk.isSlot).toBe(false);
      expect(mainChunk.ref).toBe(template.id);
      
      const nameSlotChunk = chunks.find(c => c.chunkId === `${instance.id}-slot-name`);
      expect(nameSlotChunk?.isSlot).toBe(true);
      expect(nameSlotChunk?.contents).toBe('John Doe');
      expect(nameSlotChunk?.parent).toBe(instance.id);
      
      const emailSlotChunk = chunks.find(c => c.chunkId === `${instance.id}-slot-email`);
      expect(emailSlotChunk?.contents).toBe('john@example.com');
      
      const phoneSlotChunk = chunks.find(c => c.chunkId === `${instance.id}-slot-phone`);
      expect(phoneSlotChunk?.contents).toBe('123-456-7890');
    });

    it('should only create chunks for populated slots', () => {
      const template = createSampleTemplate();
      const instance = renderer.createTemplateInstance(
        template,
        '/notes/contact.md',
        {
          'slot-name': 'John Doe'
          // email and phone not provided
        }
      );

      const chunks = renderer.instanceToChunks(template, instance);

      expect(chunks).toHaveLength(2); // 1 main + 1 populated slot
      
      const slotChunks = chunks.filter(c => c.isSlot);
      expect(slotChunks).toHaveLength(1);
      expect(slotChunks[0].chunkId).toBe(`${instance.id}-slot-name`);
    });

    it('should handle tag slots correctly', () => {
      const template = createSampleTemplate();
      template.slots.push({
        id: 'slot-category',
        name: 'category',
        type: 'tag',
        required: false,
        defaultValue: undefined,
        validation: undefined
      });

      const instance = renderer.createTemplateInstance(
        template,
        '/notes/contact.md',
        {
          'slot-name': 'John Doe',
          'slot-category': 'work'
        }
      );

      const chunks = renderer.instanceToChunks(template, instance);
      
      const categoryChunk = chunks.find(c => c.chunkId === `${instance.id}-slot-category`);
      expect(categoryChunk?.isTag).toBe(true);
      expect(categoryChunk?.tags).toContain('work');
    });
  });

  describe('generateTemplatePreview', () => {
    it('should generate preview with placeholder values', () => {
      const template = createSampleTemplate();
      const preview = renderer.generateTemplatePreview(template);

      expect(preview).toContain('# Contact Template');
      expect(preview).toContain('Name: [name]');
      expect(preview).toContain('Email: [email]');
      expect(preview).toContain('Phone: Not provided'); // Default value
    });

    it('should use default values in preview', () => {
      const template = createSampleTemplate();
      template.slots[0].defaultValue = 'Sample Name';

      const preview = renderer.generateTemplatePreview(template);

      expect(preview).toContain('Name: Sample Name');
    });
  });

  describe('validateRenderedContent', () => {
    it('should validate complete rendered content', () => {
      const template = createSampleTemplate();
      const content = `# Contact Template

## Basic Information
Name: John Doe
Email: john@example.com

## Contact Details
Phone: 123-456-7890`;

      const result = renderer.validateRenderedContent(template, content);

      expect(result.valid).toBe(true);
      expect(result.errors).toHaveLength(0);
      expect(result.warnings).toHaveLength(0);
    });

    it('should detect missing required sections', () => {
      const template = createSampleTemplate();
      const content = `# Contact Template

## Basic Information
Name: John Doe
Email: john@example.com

// Missing Contact Details section`;

      const result = renderer.validateRenderedContent(template, content);

      expect(result.valid).toBe(false);
      expect(result.errors.some(e => e.includes('Missing required section: Contact Details'))).toBe(true);
    });

    it('should detect unfilled required slots', () => {
      const template = createSampleTemplate();
      const content = `# Contact Template

## Basic Information
Name: {{name:text|required}}
Email: john@example.com

## Contact Details
Phone: 123-456-7890`;

      const result = renderer.validateRenderedContent(template, content);

      expect(result.valid).toBe(false);
      expect(result.errors.some(e => e.includes('Required slot not filled: name'))).toBe(true);
    });

    it('should warn about unfilled optional slots', () => {
      const template = createSampleTemplate();
      const content = `# Contact Template

## Basic Information
Name: John Doe
Email: {{email:text}}

## Contact Details
Phone: 123-456-7890`;

      const result = renderer.validateRenderedContent(template, content);

      expect(result.valid).toBe(true);
      expect(result.warnings.some(w => w.includes('Unfilled optional slots'))).toBe(true);
    });

    it('should handle validation errors', () => {
      const template = createSampleTemplate();
      // Simulate an error by making sections invalid
      template.structure.sections = null as any;

      const result = renderer.validateRenderedContent(template, 'content');

      expect(result.valid).toBe(false);
      expect(result.errors.some(e => e.includes('Validation error'))).toBe(true);
    });
  });

  describe('extractSlotValuesFromContent', () => {
    it('should extract slot values from content with slot markers', () => {
      const template = createSampleTemplate();
      const content = `# Contact Template

<!-- slot:name -->John Doe<!-- /slot:name -->
<!-- slot:email -->john@example.com<!-- /slot:email -->
<!-- slot:phone -->123-456-7890<!-- /slot:phone -->`;

      const slotValues = renderer.extractSlotValuesFromContent(template, content);

      expect(slotValues['slot-name']).toBe('John Doe');
      expect(slotValues['slot-email']).toBe('john@example.com');
      expect(slotValues['slot-phone']).toBe('123-456-7890');
    });

    it('should handle missing slot markers', () => {
      const template = createSampleTemplate();
      const content = `# Contact Template

Name: John Doe
Email: john@example.com`;

      const slotValues = renderer.extractSlotValuesFromContent(template, content);

      expect(Object.keys(slotValues)).toHaveLength(0);
    });

    it('should parse values according to slot type', () => {
      const template = createSampleTemplate();
      template.slots.push({
        id: 'slot-age',
        name: 'age',
        type: 'number',
        required: false,
        defaultValue: undefined,
        validation: undefined
      });

      const content = `<!-- slot:age -->25<!-- /slot:age -->`;
      const slotValues = renderer.extractSlotValuesFromContent(template, content);

      expect(slotValues['slot-age']).toBe(25);
      expect(typeof slotValues['slot-age']).toBe('number');
    });
  });
});