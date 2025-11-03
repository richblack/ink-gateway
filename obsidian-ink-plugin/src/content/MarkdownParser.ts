/**
 * Markdown content parser for Obsidian Ink Plugin
 * Handles parsing of markdown content including hierarchy, metadata, and position tracking
 */

import { 
  ParsedContent, 
  HierarchyNode, 
  ContentMetadata, 
  Position, 
  PositionMap,
  UnifiedChunk,
  ObsidianMetadata
} from '../types';

export interface MarkdownElement {
  type: 'heading' | 'paragraph' | 'list' | 'listItem' | 'link' | 'image' | 'code' | 'blockquote';
  content: string;
  level?: number;
  position: Position;
  metadata?: Record<string, any>;
}

export interface ParseOptions {
  trackPositions: boolean;
  parseHierarchy: boolean;
  extractMetadata: boolean;
  generateChunks: boolean;
}

export class MarkdownParser {
  private static readonly HEADING_REGEX = /^(#{1,6})\s+(.+)$/gm;
  private static readonly LIST_ITEM_REGEX = /^(\s*)([-*+]|\d+\.)\s+(.+)$/gm;
  private static readonly LINK_REGEX = /\[([^\]]+)\]\(([^)]+)\)/g;
  private static readonly IMAGE_REGEX = /!\[([^\]]*)\]\(([^)]+)\)/g;
  private static readonly TAG_REGEX = /#([a-zA-Z0-9_/-]+)/g;
  private static readonly FRONTMATTER_REGEX = /^---\n([\s\S]*?)\n---/;
  private static readonly PROPERTY_REGEX = /^([a-zA-Z0-9_-]+)::\s*(.+)$/gm;

  /**
   * Parse markdown content into structured format
   */
  public static parseContent(
    content: string, 
    filePath: string, 
    options: ParseOptions = {
      trackPositions: true,
      parseHierarchy: true,
      extractMetadata: true,
      generateChunks: true
    }
  ): ParsedContent {
    const lines = content.split('\n');
    const elements: MarkdownElement[] = [];
    const positions: PositionMap = new Map();
    
    // Parse frontmatter and properties
    const metadata = options.extractMetadata ? 
      this.extractMetadata(content, filePath) : 
      this.createEmptyMetadata(filePath);
    
    // Parse content elements
    let currentPosition = 0;
    for (let i = 0; i < lines.length; i++) {
      const line = lines[i];
      const lineStart = currentPosition;
      const lineEnd = currentPosition + line.length;
      
      const position: Position = {
        fileName: filePath,
        lineStart: i + 1,
        lineEnd: i + 1,
        charStart: lineStart,
        charEnd: lineEnd
      };
      
      // Parse different element types
      const element = this.parseLine(line, position, i);
      if (element) {
        elements.push(element);
        if (options.trackPositions) {
          positions.set(element.content, position);
        }
      }
      
      currentPosition = lineEnd + 1; // +1 for newline character
    }
    
    // Build hierarchy
    const hierarchy = options.parseHierarchy ? 
      this.buildHierarchy(elements) : [];
    
    // Generate chunks
    const chunks = options.generateChunks ? 
      this.generateChunks(elements, metadata, filePath) : [];
    
    return {
      chunks,
      hierarchy,
      metadata,
      positions
    };
  }

  /**
   * Parse a single line into a markdown element
   */
  private static parseLine(line: string, position: Position, lineNumber: number): MarkdownElement | null {
    const trimmedLine = line.trim();
    
    if (!trimmedLine) {
      return null; // Skip empty lines
    }
    
    // Check for heading
    const headingMatch = trimmedLine.match(/^(#{1,6})\s+(.+)$/);
    if (headingMatch) {
      return {
        type: 'heading',
        content: headingMatch[2],
        level: headingMatch[1].length,
        position
      };
    }
    
    // Check for list item
    const listMatch = line.match(/^(\s*)([-*+]|\d+\.)\s+(.+)$/);
    if (listMatch) {
      const indentLevel = Math.floor(listMatch[1].length / 2) + 1; // 2 spaces = 1 level
      return {
        type: 'listItem',
        content: listMatch[3],
        level: indentLevel,
        position,
        metadata: {
          bullet: listMatch[2],
          indent: listMatch[1].length
        }
      };
    }
    
    // Check for blockquote
    if (trimmedLine.startsWith('>')) {
      return {
        type: 'blockquote',
        content: trimmedLine.substring(1).trim(),
        position
      };
    }
    
    // Check for code block
    if (trimmedLine.startsWith('```')) {
      return {
        type: 'code',
        content: trimmedLine,
        position,
        metadata: {
          language: trimmedLine.substring(3).trim()
        }
      };
    }
    
    // Default to paragraph
    return {
      type: 'paragraph',
      content: trimmedLine,
      position
    };
  }

  /**
   * Extract metadata from content including frontmatter and properties
   */
  private static extractMetadata(content: string, filePath: string): ContentMetadata {
    const metadata: ContentMetadata = this.createEmptyMetadata(filePath);
    
    // Extract frontmatter
    const frontmatterMatch = content.match(this.FRONTMATTER_REGEX);
    if (frontmatterMatch) {
      try {
        const frontmatterContent = frontmatterMatch[1];
        const frontmatterLines = frontmatterContent.split('\n');
        
        for (const line of frontmatterLines) {
          const [key, ...valueParts] = line.split(':');
          if (key && valueParts.length > 0) {
            const value = valueParts.join(':').trim();
            metadata.frontmatter[key.trim()] = this.parseValue(value);
            
            // Map common frontmatter fields
            switch (key.trim().toLowerCase()) {
              case 'title':
                metadata.title = value;
                break;
              case 'tags':
                metadata.tags = this.parseTags(value);
                break;
              case 'aliases':
                metadata.properties.aliases = this.parseArray(value);
                break;
              case 'cssclass':
              case 'cssclasses':
                metadata.properties.cssClasses = this.parseArray(value);
                break;
            }
          }
        }
      } catch (error) {
        console.warn('Failed to parse frontmatter:', error);
      }
    }
    
    // Extract inline properties (Obsidian format: key:: value)
    let propertyMatch;
    this.PROPERTY_REGEX.lastIndex = 0;
    while ((propertyMatch = this.PROPERTY_REGEX.exec(content)) !== null) {
      const key = propertyMatch[1];
      const value = propertyMatch[2];
      metadata.properties[key] = this.parseValue(value);
    }
    
    // Extract tags from content
    const contentTags = this.extractTags(content);
    metadata.tags = [...new Set([...metadata.tags, ...contentTags])];
    
    return metadata;
  }

  /**
   * Create empty metadata structure
   */
  private static createEmptyMetadata(filePath: string): ContentMetadata {
    const now = new Date();
    return {
      tags: [],
      properties: {},
      frontmatter: {},
      aliases: [],
      cssClasses: [],
      createdTime: now,
      modifiedTime: now
    };
  }

  /**
   * Build hierarchical structure from parsed elements
   */
  private static buildHierarchy(elements: MarkdownElement[]): HierarchyNode[] {
    const nodes: HierarchyNode[] = [];
    const stack: HierarchyNode[] = []; // Stack to track parent nodes
    
    for (const element of elements) {
      if (element.type === 'heading' || element.type === 'listItem') {
        const node: HierarchyNode = {
          id: this.generateNodeId(element),
          content: element.content,
          level: element.level || 1,
          type: element.type === 'heading' ? 'heading' : 'bullet',
          children: [],
          position: element.position
        };
        
        // Find parent based on level
        let parent: HierarchyNode | undefined;
        
        if (element.type === 'heading') {
          // For headings, find the closest parent heading with lower level
          while (stack.length > 0) {
            const potential = stack[stack.length - 1];
            if (potential.type === 'heading' && potential.level < node.level) {
              parent = potential;
              break;
            }
            stack.pop();
          }
        } else if (element.type === 'listItem') {
          // For list items, find parent with lower level
          while (stack.length > 0) {
            const potential = stack[stack.length - 1];
            if (potential.level < node.level) {
              parent = potential;
              break;
            }
            stack.pop();
          }
        }
        
        // Set parent relationship
        if (parent) {
          node.parent = parent.id;
          parent.children.push(node.id);
        }
        
        nodes.push(node);
        stack.push(node);
      }
    }
    
    return nodes;
  }

  /**
   * Generate chunks from parsed elements
   */
  private static generateChunks(
    elements: MarkdownElement[], 
    metadata: ContentMetadata, 
    filePath: string
  ): UnifiedChunk[] {
    const chunks: UnifiedChunk[] = [];
    const now = new Date();
    
    // Generate document ID for this file
    const documentId = this.generateDocumentId(filePath);
    
    // Create page chunk
    const pageChunk: UnifiedChunk = {
      chunkId: this.generateChunkId(filePath, 'page'),
      contents: metadata.title || this.extractFileName(filePath),
      isPage: true,
      isTag: false,
      isTemplate: false,
      isSlot: false,
      tags: metadata.tags,
      metadata: metadata.properties,
      createdTime: metadata.createdTime,
      lastUpdated: now,
      documentId,
      documentScope: 'file' as const,
      position: {
        fileName: filePath,
        lineStart: 1,
        lineEnd: 1,
        charStart: 0,
        charEnd: 0
      },
      filePath,
      obsidianMetadata: {
        properties: metadata.properties,
        frontmatter: metadata.frontmatter,
        aliases: metadata.aliases,
        cssClasses: metadata.cssClasses
      }
    };
    chunks.push(pageChunk);
    
    // Create chunks for content elements
    for (const element of elements) {
      if (element.content.trim()) {
        const chunk: UnifiedChunk = {
          chunkId: this.generateChunkId(filePath, element.content),
          contents: element.content,
          parent: element.type === 'heading' ? undefined : pageChunk.chunkId,
          page: pageChunk.chunkId,
          isPage: false,
          isTag: false,
          isTemplate: false,
          isSlot: false,
          tags: this.extractTags(element.content),
          metadata: {
            type: element.type,
            level: element.level,
            ...element.metadata
          },
          createdTime: now,
          lastUpdated: now,
          position: element.position,
          filePath,
          obsidianMetadata: {
            properties: {},
            frontmatter: {},
            aliases: [],
            cssClasses: []
          },
          documentId,
          documentScope: 'file'
        };
        
        chunks.push(chunk);
      }
    }
    
    return chunks;
  }

  /**
   * Extract tags from content
   */
  private static extractTags(content: string): string[] {
    const tags: string[] = [];
    let match;
    
    this.TAG_REGEX.lastIndex = 0;
    while ((match = this.TAG_REGEX.exec(content)) !== null) {
      tags.push(match[1]);
    }
    
    return [...new Set(tags)]; // Remove duplicates
  }

  /**
   * Parse value from string (handles arrays, booleans, numbers)
   */
  private static parseValue(value: string): any {
    const trimmed = value.trim();
    
    // Handle arrays
    if (trimmed.startsWith('[') && trimmed.endsWith(']')) {
      return this.parseArray(trimmed);
    }
    
    // Handle booleans
    if (trimmed.toLowerCase() === 'true') return true;
    if (trimmed.toLowerCase() === 'false') return false;
    
    // Handle numbers
    const num = Number(trimmed);
    if (!isNaN(num) && isFinite(num)) return num;
    
    // Handle dates
    const date = new Date(trimmed);
    if (!isNaN(date.getTime()) && trimmed.match(/^\d{4}-\d{2}-\d{2}/)) {
      return date;
    }
    
    // Default to string
    return trimmed;
  }

  /**
   * Parse array from string
   */
  private static parseArray(value: string): string[] {
    const trimmed = value.trim();
    
    if (trimmed.startsWith('[') && trimmed.endsWith(']')) {
      const content = trimmed.slice(1, -1);
      return content.split(',').map(item => item.trim().replace(/^["']|["']$/g, ''));
    }
    
    // Handle comma-separated values
    return value.split(',').map(item => item.trim());
  }

  /**
   * Parse tags from various formats
   */
  private static parseTags(value: string): string[] {
    const trimmed = value.trim();
    
    if (trimmed.startsWith('[') && trimmed.endsWith(']')) {
      return this.parseArray(trimmed);
    }
    
    // Handle space or comma separated tags
    return trimmed.split(/[,\s]+/).filter(tag => tag.length > 0);
  }

  /**
   * Generate unique node ID
   */
  private static generateNodeId(element: MarkdownElement): string {
    const content = element.content.substring(0, 50).replace(/[^a-zA-Z0-9]/g, '_');
    const type = element.type;
    const level = element.level || 0;
    return `${type}_${level}_${content}_${Date.now()}`;
  }

  /**
   * Generate unique chunk ID
   */
  private static generateChunkId(filePath: string, content: string): string {
    const fileName = this.extractFileName(filePath);
    const contentHash = this.simpleHash(content);
    return `${fileName}_${contentHash}_${Date.now()}`;
  }

  /**
   * Extract file name from path
   */
  private static extractFileName(filePath: string): string {
    return filePath.split('/').pop()?.replace('.md', '') || 'unknown';
  }

  /**
   * Simple hash function for content
   */
  private static simpleHash(str: string): string {
    let hash = 0;
    for (let i = 0; i < str.length; i++) {
      const char = str.charCodeAt(i);
      hash = ((hash << 5) - hash) + char;
      hash = hash & hash; // Convert to 32-bit integer
    }
    return Math.abs(hash).toString(36);
  }

  /**
   * Generate document ID for a file path
   */
  private static generateDocumentId(filePath: string): string {
    // Normalize file path and create a consistent document ID
    const normalizedPath = filePath.replace(/\\/g, '/').replace(/^\/+/, '');
    
    // Use a combination of file path and a hash for uniqueness
    const pathHash = this.generatePathHash(normalizedPath);
    return `file_${pathHash}_${normalizedPath.replace(/[^a-zA-Z0-9]/g, '_')}`;
  }

  /**
   * Generate hash for path/context
   */
  private static generatePathHash(input: string): string {
    let hash = 0;
    for (let i = 0; i < input.length; i++) {
      const char = input.charCodeAt(i);
      hash = ((hash << 5) - hash) + char;
      hash = hash & hash;
    }
    return Math.abs(hash).toString(36).padStart(6, '0');
  }

  /**
   * Parse links from content
   */
  public static parseLinks(content: string): Array<{text: string, url: string, position: number}> {
    const links: Array<{text: string, url: string, position: number}> = [];
    let match;
    
    this.LINK_REGEX.lastIndex = 0;
    while ((match = this.LINK_REGEX.exec(content)) !== null) {
      links.push({
        text: match[1],
        url: match[2],
        position: match.index
      });
    }
    
    return links;
  }

  /**
   * Parse images from content
   */
  public static parseImages(content: string): Array<{alt: string, src: string, position: number}> {
    const images: Array<{alt: string, src: string, position: number}> = [];
    let match;
    
    this.IMAGE_REGEX.lastIndex = 0;
    while ((match = this.IMAGE_REGEX.exec(content)) !== null) {
      images.push({
        alt: match[1],
        src: match[2],
        position: match.index
      });
    }
    
    return images;
  }

  /**
   * Validate parsed content
   */
  public static validateParsedContent(parsed: ParsedContent): boolean {
    try {
      // Check required fields
      if (!parsed.chunks || !parsed.hierarchy || !parsed.metadata || !parsed.positions) {
        return false;
      }
      
      // Validate chunks
      for (const chunk of parsed.chunks) {
        if (!chunk.chunkId || !chunk.contents || !chunk.position) {
          return false;
        }
      }
      
      // Validate hierarchy
      for (const node of parsed.hierarchy) {
        if (!node.id || !node.content || !node.position) {
          return false;
        }
      }
      
      return true;
    } catch (error) {
      console.error('Content validation failed:', error);
      return false;
    }
  }
}