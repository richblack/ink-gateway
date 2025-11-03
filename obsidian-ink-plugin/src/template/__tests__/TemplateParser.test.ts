/**
 * Unit tests for TemplateParser
 * Tests template parsing, slot extraction, and content processing
 */

import { TemplateParser } from '../TemplateParser';
import { ILogger } from '../../interfaces';
import { TemplateSlot, TemplateStructure } from '../../types';

class MockLogger implements ILogger {
  debug = jest.fn();
  info = jest.fn();
  warn = jest.fn();
  error = jest.fn();
}

describe('TemplateParser', () => {
  let parser: TemplateParser;
  let mockLogger: MockLogger;

  beforeEach(() => {
    mockLogger = new MockLogger();
    parser = new TemplateParser(mockLogger);
  });

  describe('parseTemplateContent', () => {
    it('should parse template content with sections and slots', () => {
      const content = `# Contact Template

## Basic Information
Name: {{name:text|required}}
Email: {{email:text|pattern:^[^@]+@[^@]+\\.[^@]+$}}

## Additional Details
Phone: {{phone:text}}
Notes: {{notes:text|default:No notes}}`;

      const result = parser.parseTemplateContent(content, '/templates/contact.md');

      expect(result.structure.sections).toHaveLength(2);
      expect(result.structure.sections[0].title).toBe('Basic Information');
      expect(result.structure.sections[1].title).toBe('Additional Details');
      
      expect(result.slots).toHaveLength(4);
      
      const nameSlot = result.slots.find(s => s.name === 'name');
      expect(nameSlot?.required).toBe(true);
      expect(nameSlot?.type).toBe('text');
      
      const emailSlot = result.slots.find(s => s.name === 'email');
      expect(emailSlot?.validation?.pattern).toBe('^[^@]+@[^@]+\\.[^@]+$');
      
      const notesSlot = result.slots.find(s => s.name === 'notes');
      expect(notesSlot?.defaultValue).toBe('No notes');

      expect(result.chunks).toHaveLength(2); // One chunk per section
    });

    it('should handle content without sections', () => {
      const content = `Name: {{name:text}}
Age: {{age:number}}
Email: {{email:text}}`;

      const result = parser.parseTemplateContent(content, '/templates/simple.md');

      expect(result.structure.sections).toHaveLength(0);
      expect(result.slots).toHaveLength(3);
      expect(result.chunks).toHaveLength(0);
    });

    it('should handle empty content', () => {
      const content = '';

      const result = parser.parseTemplateContent(content, '/templates/empty.md');

      expect(result.structure.sections).toHaveLength(0);
      expect(result.slots).toHaveLength(0);
      expect(result.chunks).toHaveLength(0);
    });
  });

  describe('extractSlotDefinitions', () => {
    it('should extract basic slots', () => {
      const content = 'Name: {{name:text}} Age: {{age:number}}';
      const slots = parser.extractSlotDefinitions(content);

      expect(slots).toHaveLength(2);
      expect(slots[0].name).toBe('name');
      expect(slots[0].type).toBe('text');
      expect(slots[1].name).toBe('age');
      expect(slots[1].type).toBe('number');
    });

    it('should extract slots with validation rules', () => {
      const content = `Email: {{email:text|pattern:^[^@]+@[^@]+\\.[^@]+$,required}}
Password: {{password:text|minLength:8,maxLength:50,required}}
Age: {{age:number|default:25}}`;

      const slots = parser.extractSlotDefinitions(content);

      expect(slots).toHaveLength(3);
      
      const emailSlot = slots.find(s => s.name === 'email');
      expect(emailSlot?.validation?.pattern).toBe('^[^@]+@[^@]+\\.[^@]+$');
      expect(emailSlot?.required).toBe(true);
      
      const passwordSlot = slots.find(s => s.name === 'password');
      expect(passwordSlot?.validation?.minLength).toBe(8);
      expect(passwordSlot?.validation?.maxLength).toBe(50);
      expect(passwordSlot?.required).toBe(true);
      
      const ageSlot = slots.find(s => s.name === 'age');
      expect(ageSlot?.defaultValue).toBe(25);
      expect(ageSlot?.type).toBe('number');
    });

    it('should handle different slot types', () => {
      const content = `Name: {{name:text}}
Birthday: {{birthday:date}}
Website: {{website:link}}
Tags: {{tags:tag}}
Count: {{count:number}}`;

      const slots = parser.extractSlotDefinitions(content);

      expect(slots).toHaveLength(5);
      expect(slots.find(s => s.name === 'name')?.type).toBe('text');
      expect(slots.find(s => s.name === 'birthday')?.type).toBe('date');
      expect(slots.find(s => s.name === 'website')?.type).toBe('link');
      expect(slots.find(s => s.name === 'tags')?.type).toBe('tag');
      expect(slots.find(s => s.name === 'count')?.type).toBe('number');
    });

    it('should avoid duplicate slots', () => {
      const content = `Name: {{name:text}}
Full Name: {{name:text}}
Display Name: {{name:text}}`;

      const slots = parser.extractSlotDefinitions(content);

      expect(slots).toHaveLength(1);
      expect(slots[0].name).toBe('name');
    });

    it('should handle invalid slot types', () => {
      const content = 'Invalid: {{invalid:unknown}}';
      const slots = parser.extractSlotDefinitions(content);

      expect(slots).toHaveLength(1);
      expect(slots[0].type).toBe('text'); // Should default to text
    });
  });

  describe('applySlotValues', () => {
    it('should replace slots with values', () => {
      const content = 'Name: {{name:text}} Age: {{age:number}}';
      const slotValues = {
        'slot-name': 'John Doe',
        'slot-age': 30
      };

      const result = parser.applySlotValues(content, slotValues);

      expect(result).toBe('Name: John Doe Age: 30');
    });

    it('should use default values when slot value is missing', () => {
      const content = 'Name: {{name:text|default:Unknown}} Age: {{age:number}}';
      const slotValues = {
        'slot-age': 25
      };

      const result = parser.applySlotValues(content, slotValues);

      expect(result).toBe('Name: Unknown Age: 25');
    });

    it('should format values according to type', () => {
      const content = `Date: {{date:date}}
Link: {{link:link}}
Tag: {{tag:tag}}
Number: {{number:number}}`;

      const slotValues = {
        'slot-date': new Date('2023-12-25'),
        'slot-link': 'My Page',
        'slot-tag': 'important',
        'slot-number': 42.5
      };

      const result = parser.applySlotValues(content, slotValues);

      expect(result).toContain('Date: 2023-12-25');
      expect(result).toContain('Link: [[My Page]]');
      expect(result).toContain('Tag: #important');
      expect(result).toContain('Number: 42.5');
    });

    it('should handle missing values with placeholders', () => {
      const content = 'Name: {{name:text}} Age: {{age:number}}';
      const slotValues = {
        'slot-name': 'John Doe'
      };

      const result = parser.applySlotValues(content, slotValues);

      expect(result).toBe('Name: John Doe Age: [age]');
    });
  });

  describe('validateTemplateSyntax', () => {
    it('should validate correct syntax', () => {
      const content = 'Name: {{name:text}} Age: {{age:number|required}}';
      const result = parser.validateTemplateSyntax(content);

      expect(result.valid).toBe(true);
      expect(result.errors).toHaveLength(0);
    });

    it('should detect mismatched tags', () => {
      const content = 'Name: {{name:text} Age: {{age:number}}';
      const result = parser.validateTemplateSyntax(content);

      expect(result.valid).toBe(false);
      expect(result.errors).toContain('Mismatched slot tags: unclosed {{ or }}');
    });

    it('should detect invalid slot syntax', () => {
      const content = 'Name: {{}} Age: {{age:number}}';
      const result = parser.validateTemplateSyntax(content);

      expect(result.valid).toBe(false);
      expect(result.errors.some(e => e.includes('Invalid slot syntax'))).toBe(true);
    });

    it('should detect duplicate slot names', () => {
      const content = 'Name: {{name:text}} Full Name: {{name:text}}';
      const result = parser.validateTemplateSyntax(content);

      expect(result.valid).toBe(false);
      expect(result.errors.some(e => e.includes('Duplicate slot names'))).toBe(true);
    });

    it('should handle syntax validation errors', () => {
      // Create content that will cause a regex error
      const content = 'Name: {{name:text}} Invalid: {{';
      const result = parser.validateTemplateSyntax(content);

      expect(result.valid).toBe(false);
      expect(result.errors.some(e => e.includes('Mismatched slot tags'))).toBe(true);
    });
  });

  describe('templateToChunks', () => {
    it('should convert template to unified chunks', () => {
      const template = {
        id: 'template-1',
        name: 'Contact Template',
        slots: [
          {
            id: 'slot-name',
            name: 'name',
            type: 'text' as const,
            required: true,
            defaultValue: undefined,
            validation: undefined
          },
          {
            id: 'slot-email',
            name: 'email',
            type: 'text' as const,
            required: false,
            defaultValue: undefined,
            validation: { pattern: '^[^@]+@[^@]+\\.[^@]+$' }
          }
        ],
        structure: {
          layout: 'standard',
          sections: []
        },
        metadata: {
          description: 'A contact template',
          category: 'personal',
          tags: ['contact'],
          createdTime: new Date('2023-01-01'),
          lastUpdated: new Date('2023-01-01')
        }
      };

      const chunks = parser.templateToChunks(template, '/templates/contact.md');

      expect(chunks).toHaveLength(3); // 1 main + 2 slots
      
      const mainChunk = chunks[0];
      expect(mainChunk.chunkId).toBe('template-1');
      expect(mainChunk.isTemplate).toBe(true);
      expect(mainChunk.contents).toBe('Contact Template');
      
      const nameSlotChunk = chunks.find(c => c.chunkId === 'template-1-slot-name');
      expect(nameSlotChunk?.isSlot).toBe(true);
      expect(nameSlotChunk?.parent).toBe('template-1');
      
      const emailSlotChunk = chunks.find(c => c.chunkId === 'template-1-slot-email');
      expect(emailSlotChunk?.isSlot).toBe(true);
      expect(emailSlotChunk?.metadata.slotValidation).toEqual({ pattern: '^[^@]+@[^@]+\\.[^@]+$' });
    });

    it('should handle template with no slots', () => {
      const template = {
        id: 'template-2',
        name: 'Simple Template',
        slots: [],
        structure: {
          layout: 'standard',
          sections: []
        },
        metadata: {
          description: 'A simple template',
          category: 'basic',
          tags: [],
          createdTime: new Date(),
          lastUpdated: new Date()
        }
      };

      const chunks = parser.templateToChunks(template, '/templates/simple.md');

      expect(chunks).toHaveLength(1); // Only main chunk
      expect(chunks[0].isTemplate).toBe(true);
      expect(chunks[0].contents).toBe('Simple Template');
    });
  });
});