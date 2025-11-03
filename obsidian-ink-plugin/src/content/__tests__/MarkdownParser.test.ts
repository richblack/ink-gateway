/**
 * Unit tests for MarkdownParser
 */

import { MarkdownParser } from '../MarkdownParser';
import { ParsedContent, HierarchyNode, ContentMetadata } from '../../types';

describe('MarkdownParser', () => {
  describe('parseContent', () => {
    it('should parse basic markdown content', () => {
      const content = `# Main Title

This is a paragraph.

## Subtitle

Another paragraph with some text.

- List item 1
- List item 2
  - Nested item
    - Deep nested item

### Sub-subtitle

Final paragraph.`;

      const filePath = 'test.md';
      const result = MarkdownParser.parseContent(content, filePath);

      expect(result).toBeDefined();
      expect(result.chunks).toBeDefined();
      expect(result.hierarchy).toBeDefined();
      expect(result.metadata).toBeDefined();
      expect(result.positions).toBeDefined();
    });

    it('should parse frontmatter correctly', () => {
      const content = `---
title: Test Document
tags: [test, markdown, parser]
aliases: [test-doc, sample]
cssclass: custom-style
created: 2024-01-01
---

# Content

This is the main content.`;

      const result = MarkdownParser.parseContent(content, 'test.md');

      expect(result.metadata.title).toBe('Test Document');
      expect(result.metadata.tags).toEqual(['test', 'markdown', 'parser']);
      expect(result.metadata.frontmatter.title).toBe('Test Document');
      expect(result.metadata.frontmatter.created).toEqual(new Date('2024-01-01'));
    });

    it('should parse inline properties', () => {
      const content = `# Document

author:: John Doe
status:: draft
priority:: high

Content here.`;

      const result = MarkdownParser.parseContent(content, 'test.md');

      expect(result.metadata.properties.author).toBe('John Doe');
      expect(result.metadata.properties.status).toBe('draft');
      expect(result.metadata.properties.priority).toBe('high');
    });

    it('should extract tags from content', () => {
      const content = `# Document

This document has #important and #urgent tags.

Also has #project/work and #status/active tags.`;

      const result = MarkdownParser.parseContent(content, 'test.md');

      expect(result.metadata.tags).toContain('important');
      expect(result.metadata.tags).toContain('urgent');
      expect(result.metadata.tags).toContain('project/work');
      expect(result.metadata.tags).toContain('status/active');
    });

    it('should build correct hierarchy for headings', () => {
      const content = `# Level 1

Content under level 1.

## Level 2A

Content under level 2A.

### Level 3

Content under level 3.

## Level 2B

Content under level 2B.`;

      const result = MarkdownParser.parseContent(content, 'test.md');
      const hierarchy = result.hierarchy;

      // Find nodes
      const level1 = hierarchy.find(n => n.content === 'Level 1' && n.level === 1);
      const level2A = hierarchy.find(n => n.content === 'Level 2A' && n.level === 2);
      const level3 = hierarchy.find(n => n.content === 'Level 3' && n.level === 3);
      const level2B = hierarchy.find(n => n.content === 'Level 2B' && n.level === 2);

      expect(level1).toBeDefined();
      expect(level2A).toBeDefined();
      expect(level3).toBeDefined();
      expect(level2B).toBeDefined();

      // Check parent-child relationships
      expect(level2A?.parent).toBe(level1?.id);
      expect(level3?.parent).toBe(level2A?.id);
      expect(level2B?.parent).toBe(level1?.id);

      // Check children
      expect(level1?.children).toContain(level2A?.id);
      expect(level1?.children).toContain(level2B?.id);
      expect(level2A?.children).toContain(level3?.id);
    });

    it('should build correct hierarchy for lists', () => {
      const content = `# Document

- Item 1
  - Sub item 1.1
    - Deep item 1.1.1
  - Sub item 1.2
- Item 2
  - Sub item 2.1`;

      const result = MarkdownParser.parseContent(content, 'test.md');
      const hierarchy = result.hierarchy.filter(n => n.type === 'bullet');

      // Find nodes
      const item1 = hierarchy.find(n => n.content === 'Item 1');
      const subItem11 = hierarchy.find(n => n.content === 'Sub item 1.1');
      const deepItem111 = hierarchy.find(n => n.content === 'Deep item 1.1.1');
      const subItem12 = hierarchy.find(n => n.content === 'Sub item 1.2');
      const item2 = hierarchy.find(n => n.content === 'Item 2');
      const subItem21 = hierarchy.find(n => n.content === 'Sub item 2.1');

      expect(item1).toBeDefined();
      expect(subItem11).toBeDefined();
      expect(deepItem111).toBeDefined();

      // Check levels
      expect(item1?.level).toBe(1);
      expect(subItem11?.level).toBe(2);
      expect(deepItem111?.level).toBe(3);

      // Check parent-child relationships
      expect(subItem11?.parent).toBe(item1?.id);
      expect(deepItem111?.parent).toBe(subItem11?.id);
      expect(subItem12?.parent).toBe(item1?.id);
      expect(subItem21?.parent).toBe(item2?.id);
    });

    it('should track positions correctly', () => {
      const content = `# Title
Paragraph 1
## Subtitle
Paragraph 2`;

      const result = MarkdownParser.parseContent(content, 'test.md');

      expect(result.positions.size).toBeGreaterThan(0);
      
      // Check that positions are tracked
      const titlePosition = result.positions.get('Title');
      expect(titlePosition).toBeDefined();
      expect(titlePosition?.fileName).toBe('test.md');
      expect(titlePosition?.lineStart).toBe(1);
    });

    it('should generate chunks correctly', () => {
      const content = `# Document Title

This is content.

## Section

More content here.`;

      const result = MarkdownParser.parseContent(content, 'test.md');
      const chunks = result.chunks;

      // Should have page chunk plus content chunks
      expect(chunks.length).toBeGreaterThan(1);

      // Find page chunk
      const pageChunk = chunks.find(c => c.isPage);
      expect(pageChunk).toBeDefined();
      expect(pageChunk?.contents).toBe('test'); // Uses filename when no title in frontmatter

      // Find content chunks
      const contentChunks = chunks.filter(c => !c.isPage);
      expect(contentChunks.length).toBeGreaterThan(0);

      // Check chunk properties
      for (const chunk of chunks) {
        expect(chunk.chunkId).toBeDefined();
        expect(chunk.contents).toBeDefined();
        expect(chunk.position).toBeDefined();
        expect(chunk.filePath).toBe('test.md');
        expect(chunk.createdTime).toBeDefined();
        expect(chunk.lastUpdated).toBeDefined();
      }
    });

    it('should handle empty content gracefully', () => {
      const content = '';
      const result = MarkdownParser.parseContent(content, 'empty.md');

      expect(result.chunks).toBeDefined();
      expect(result.hierarchy).toBeDefined();
      expect(result.metadata).toBeDefined();
      expect(result.positions).toBeDefined();

      // Should still have a page chunk
      const pageChunk = result.chunks.find(c => c.isPage);
      expect(pageChunk).toBeDefined();
    });

    it('should handle malformed frontmatter gracefully', () => {
      const content = `---
title: Test
invalid yaml: [unclosed array
tags: test
---

# Content`;

      const result = MarkdownParser.parseContent(content, 'test.md');

      // Should still parse what it can
      expect(result.metadata).toBeDefined();
      // May or may not have parsed the malformed parts, but shouldn't crash
    });
  });

  describe('parseLinks', () => {
    it('should extract links correctly', () => {
      const content = 'Check out [Google](https://google.com) and [GitHub](https://github.com).';
      const links = MarkdownParser.parseLinks(content);

      expect(links).toHaveLength(2);
      expect(links[0]).toEqual({
        text: 'Google',
        url: 'https://google.com',
        position: expect.any(Number)
      });
      expect(links[1]).toEqual({
        text: 'GitHub',
        url: 'https://github.com',
        position: expect.any(Number)
      });
    });

    it('should handle empty content', () => {
      const links = MarkdownParser.parseLinks('');
      expect(links).toHaveLength(0);
    });
  });

  describe('parseImages', () => {
    it('should extract images correctly', () => {
      const content = 'Here is an image: ![Alt text](image.jpg) and another ![](image2.png).';
      const images = MarkdownParser.parseImages(content);

      expect(images).toHaveLength(2);
      expect(images[0]).toEqual({
        alt: 'Alt text',
        src: 'image.jpg',
        position: expect.any(Number)
      });
      expect(images[1]).toEqual({
        alt: '',
        src: 'image2.png',
        position: expect.any(Number)
      });
    });
  });

  describe('validateParsedContent', () => {
    it('should validate correct parsed content', () => {
      const content = `# Test
Content here.`;
      
      const parsed = MarkdownParser.parseContent(content, 'test.md');
      const isValid = MarkdownParser.validateParsedContent(parsed);

      expect(isValid).toBe(true);
    });

    it('should reject invalid parsed content', () => {
      const invalidParsed = {
        chunks: [],
        hierarchy: [],
        metadata: null, // Invalid - should be object
        positions: new Map()
      } as any;

      const isValid = MarkdownParser.validateParsedContent(invalidParsed);
      expect(isValid).toBe(false);
    });

    it('should reject parsed content with invalid chunks', () => {
      const invalidParsed = {
        chunks: [{ chunkId: '', contents: 'test' }], // Invalid - missing required fields
        hierarchy: [],
        metadata: { tags: [], properties: {}, frontmatter: {}, aliases: [], cssClasses: [], createdTime: new Date(), modifiedTime: new Date() },
        positions: new Map()
      } as any;

      const isValid = MarkdownParser.validateParsedContent(invalidParsed);
      expect(isValid).toBe(false);
    });
  });

  describe('edge cases', () => {
    it('should handle mixed heading and list hierarchy', () => {
      const content = `# Main Heading

- List under heading
  - Nested list item

## Sub Heading

- Another list
  - With nested items`;

      const result = MarkdownParser.parseContent(content, 'test.md');
      
      expect(result.hierarchy.length).toBeGreaterThan(0);
      
      // Should have both heading and bullet type nodes
      const headingNodes = result.hierarchy.filter(n => n.type === 'heading');
      const bulletNodes = result.hierarchy.filter(n => n.type === 'bullet');
      
      expect(headingNodes.length).toBeGreaterThan(0);
      expect(bulletNodes.length).toBeGreaterThan(0);
    });

    it('should handle code blocks', () => {
      const content = `# Document

\`\`\`javascript
function test() {
  return true;
}
\`\`\`

Regular content.`;

      const result = MarkdownParser.parseContent(content, 'test.md');
      
      // Should parse without errors
      expect(result.chunks.length).toBeGreaterThan(0);
    });

    it('should handle blockquotes', () => {
      const content = `# Document

> This is a blockquote
> with multiple lines

Regular content.`;

      const result = MarkdownParser.parseContent(content, 'test.md');
      
      // Should parse without errors
      expect(result.chunks.length).toBeGreaterThan(0);
    });

    it('should handle various tag formats', () => {
      const content = `# Document

Tags: #simple #multi-word #with/slash #with_underscore

#at-start and #at-end#`;

      const result = MarkdownParser.parseContent(content, 'test.md');
      
      expect(result.metadata.tags).toContain('simple');
      expect(result.metadata.tags).toContain('multi-word');
      expect(result.metadata.tags).toContain('with/slash');
      expect(result.metadata.tags).toContain('with_underscore');
      expect(result.metadata.tags).toContain('at-start');
      expect(result.metadata.tags).toContain('at-end');
    });

    it('should handle numbered lists', () => {
      const content = `# Document

1. First item
2. Second item
   1. Nested numbered item
   2. Another nested item
3. Third item`;

      const result = MarkdownParser.parseContent(content, 'test.md');
      
      const bulletNodes = result.hierarchy.filter(n => n.type === 'bullet');
      expect(bulletNodes.length).toBeGreaterThan(0);
      
      // Check that nested items have correct levels
      const nestedItems = bulletNodes.filter(n => n.level > 1);
      expect(nestedItems.length).toBeGreaterThan(0);
    });
  });
});