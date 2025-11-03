/**
 * Test Data Generator
 * Utilities for generating mock data for testing
 */

import { UnifiedChunk, Template, TemplateSlot, SearchResultItem, AIResponse } from '../../src/types';

export class TestDataGenerator {
    private static chunkIdCounter = 0;
    private static templateIdCounter = 0;
    private static documentIdCounter = 0;

    /**
     * Generate a mock UnifiedChunk
     */
    static generateChunk(overrides: Partial<UnifiedChunk> = {}): UnifiedChunk {
        const id = ++this.chunkIdCounter;
        return {
            chunkId: `test-chunk-${id}`,
            contents: `Test chunk content ${id}`,
            parent: undefined,
            page: undefined,
            isPage: false,
            isTag: false,
            isTemplate: false,
            isSlot: false,
            ref: undefined,
            tags: [`tag${id % 5}`, 'test'],
            metadata: {
                testData: true,
                generatedAt: new Date().toISOString(),
            },
            createdTime: new Date(),
            lastUpdated: new Date(),
            position: {
                fileName: `test-${id}.md`,
                lineStart: id,
                lineEnd: id + 1,
                charStart: 0,
                charEnd: 50,
            },
            filePath: `test-${id}.md`,
            obsidianMetadata: {
                properties: {},
                frontmatter: {},
                aliases: [],
                cssClasses: [],
            },
            documentId: `doc-${Math.floor(id / 10)}`,
            documentScope: 'file',
            ...overrides,
        };
    }

    /**
     * Generate multiple chunks with hierarchy
     */
    static generateChunkHierarchy(levels: number = 3, childrenPerLevel: number = 3): UnifiedChunk[] {
        const chunks: UnifiedChunk[] = [];
        
        // Root chunk
        const rootChunk = this.generateChunk({
            contents: 'Root Document',
            isPage: true,
        });
        chunks.push(rootChunk);
        
        this.generateChildChunks(rootChunk, chunks, levels - 1, childrenPerLevel, 1);
        
        return chunks;
    }

    private static generateChildChunks(
        parent: UnifiedChunk,
        chunks: UnifiedChunk[],
        remainingLevels: number,
        childrenPerLevel: number,
        currentLevel: number
    ): void {
        if (remainingLevels <= 0) return;
        
        for (let i = 0; i < childrenPerLevel; i++) {
            const childChunk = this.generateChunk({
                contents: `Level ${currentLevel} Child ${i + 1}`,
                parent: parent.chunkId,
                documentId: parent.documentId,
            });
            chunks.push(childChunk);
            
            // Recursively generate children
            this.generateChildChunks(childChunk, chunks, remainingLevels - 1, childrenPerLevel, currentLevel + 1);
        }
    }

    /**
     * Generate a mock Template
     */
    static generateTemplate(overrides: Partial<Template> = {}): Template {
        const id = ++this.templateIdCounter;
        return {
            id: `template-${id}`,
            name: `Test Template ${id}`,
            slots: [
                {
                    id: `slot-${id}-1`,
                    name: 'title',
                    type: 'text',
                    required: true,
                    defaultValue: `Default Title ${id}`,
                },
                {
                    id: `slot-${id}-2`,
                    name: 'description',
                    type: 'text',
                    required: false,
                    defaultValue: '',
                },
                {
                    id: `slot-${id}-3`,
                    name: 'date',
                    type: 'date',
                    required: true,
                    defaultValue: new Date().toISOString().split('T')[0],
                },
            ],
            structure: {
                layout: 'standard',
                sections: ['header', 'content', 'footer'],
            },
            metadata: {
                category: 'test',
                version: '1.0',
                createdBy: 'test-generator',
            },
            ...overrides,
        };
    }

    /**
     * Generate template slots
     */
    static generateTemplateSlots(count: number = 5): TemplateSlot[] {
        const slots: TemplateSlot[] = [];
        const slotTypes: Array<'text' | 'number' | 'date' | 'link' | 'tag'> = ['text', 'number', 'date', 'link', 'tag'];
        
        for (let i = 0; i < count; i++) {
            slots.push({
                id: `slot-${i}`,
                name: `field_${i}`,
                type: slotTypes[i % slotTypes.length],
                required: i % 2 === 0,
                defaultValue: this.getDefaultValueForType(slotTypes[i % slotTypes.length]),
            });
        }
        
        return slots;
    }

    private static getDefaultValueForType(type: string): any {
        switch (type) {
            case 'text':
                return 'Default text';
            case 'number':
                return 0;
            case 'date':
                return new Date().toISOString().split('T')[0];
            case 'link':
                return '[[Default Link]]';
            case 'tag':
                return '#default';
            default:
                return '';
        }
    }

    /**
     * Generate search results
     */
    static generateSearchResults(count: number = 10, query: string = 'test'): SearchResultItem[] {
        const results: SearchResultItem[] = [];
        
        for (let i = 0; i < count; i++) {
            const chunk = this.generateChunk({
                contents: `Search result ${i} containing ${query} in the content`,
            });
            
            results.push({
                chunk,
                score: Math.random() * 0.5 + 0.5, // 0.5 to 1.0
                context: `...context before ${query} context after...`,
                position: chunk.position,
                highlights: [
                    {
                        start: chunk.contents.indexOf(query),
                        end: chunk.contents.indexOf(query) + query.length,
                        text: query,
                    },
                ],
            });
        }
        
        // Sort by score descending
        results.sort((a, b) => b.score - a.score);
        
        return results;
    }

    /**
     * Generate AI response
     */
    static generateAIResponse(overrides: Partial<AIResponse> = {}): AIResponse {
        return {
            message: 'This is a mock AI response generated for testing purposes.',
            suggestions: [
                {
                    type: 'link',
                    content: 'Related: Test Document',
                    action: 'navigate',
                },
                {
                    type: 'tag',
                    content: '#suggested-tag',
                    action: 'add',
                },
            ],
            actions: [
                {
                    type: 'create_note',
                    title: 'Suggested Note',
                    content: 'AI suggested content',
                },
            ],
            metadata: {
                responseTime: Math.floor(Math.random() * 2000) + 500, // 500-2500ms
                tokensUsed: Math.floor(Math.random() * 500) + 100, // 100-600 tokens
                model: 'test-model-v1',
                confidence: Math.random() * 0.3 + 0.7, // 0.7-1.0
            },
            ...overrides,
        };
    }

    /**
     * Generate markdown content with various elements
     */
    static generateMarkdownContent(options: {
        headings?: number;
        paragraphs?: number;
        lists?: number;
        codeBlocks?: number;
        tags?: string[];
        links?: number;
    } = {}): string {
        const {
            headings = 3,
            paragraphs = 5,
            lists = 2,
            codeBlocks = 1,
            tags = ['test', 'generated'],
            links = 3,
        } = options;
        
        const content: string[] = [];
        
        // Add title
        content.push('# Generated Test Document\n');
        
        // Add headings and content
        for (let h = 0; h < headings; h++) {
            content.push(`## Section ${h + 1}\n`);
            
            // Add paragraphs
            for (let p = 0; p < Math.ceil(paragraphs / headings); p++) {
                content.push(`This is paragraph ${p + 1} in section ${h + 1}. It contains some sample text with **bold** and *italic* formatting.`);
                
                // Add links
                if (p < links) {
                    content.push(`Here is a [link to example ${p + 1}](http://example.com/${p + 1}).`);
                }
                
                content.push('');
            }
            
            // Add lists
            if (h < lists) {
                content.push('### List Items\n');
                content.push(`- Item ${h + 1}.1`);
                content.push(`- Item ${h + 1}.2`);
                content.push(`  - Nested item ${h + 1}.2.1`);
                content.push(`  - Nested item ${h + 1}.2.2`);
                content.push(`- Item ${h + 1}.3\n`);
            }
            
            // Add code blocks
            if (h < codeBlocks) {
                content.push('### Code Example\n');
                content.push('```javascript');
                content.push(`// Example code block ${h + 1}`);
                content.push(`function example${h + 1}() {`);
                content.push(`  return "Hello from section ${h + 1}";`);
                content.push('}');
                content.push('```\n');
            }
        }
        
        // Add tags
        if (tags.length > 0) {
            content.push(`Tags: ${tags.map(tag => `#${tag}`).join(' ')}\n`);
        }
        
        return content.join('\n');
    }

    /**
     * Generate large document for performance testing
     */
    static generateLargeDocument(sizeKB: number): string {
        const targetSize = sizeKB * 1024;
        const baseContent = this.generateMarkdownContent({
            headings: 10,
            paragraphs: 20,
            lists: 5,
            codeBlocks: 3,
            tags: ['performance', 'large', 'test'],
            links: 10,
        });
        
        // Repeat content to reach target size
        let content = baseContent;
        let counter = 1;
        
        while (content.length < targetSize) {
            content += `\n\n## Additional Section ${counter}\n\n`;
            content += `This is additional content to reach the target size of ${sizeKB}KB. `;
            content += 'Lorem ipsum dolor sit amet, consectetur adipiscing elit. '.repeat(10);
            counter++;
        }
        
        return content.substring(0, targetSize);
    }

    /**
     * Generate test file structure
     */
    static generateFileStructure(depth: number = 3, filesPerLevel: number = 5): any[] {
        const files: any[] = [];
        
        function generateLevel(currentDepth: number, parentPath: string = '') {
            if (currentDepth <= 0) return;
            
            for (let i = 0; i < filesPerLevel; i++) {
                const fileName = `${parentPath}file-${currentDepth}-${i}.md`;
                const content = this.generateMarkdownContent({
                    headings: currentDepth,
                    paragraphs: currentDepth * 2,
                });
                
                files.push({
                    path: fileName,
                    name: `file-${currentDepth}-${i}.md`,
                    content,
                    size: content.length,
                });
                
                // Generate subdirectory
                if (currentDepth > 1 && i === 0) {
                    generateLevel(currentDepth - 1, `${parentPath}subdir-${currentDepth}/`);
                }
            }
        }
        
        generateLevel.call(this, depth);
        return files;
    }

    /**
     * Generate performance test scenarios
     */
    static generatePerformanceScenarios(): Array<{
        name: string;
        description: string;
        data: any;
        expectedTime: number;
    }> {
        return [
            {
                name: 'Small Document Processing',
                description: 'Process a small document (1KB)',
                data: this.generateLargeDocument(1),
                expectedTime: 100, // 100ms
            },
            {
                name: 'Medium Document Processing',
                description: 'Process a medium document (10KB)',
                data: this.generateLargeDocument(10),
                expectedTime: 500, // 500ms
            },
            {
                name: 'Large Document Processing',
                description: 'Process a large document (100KB)',
                data: this.generateLargeDocument(100),
                expectedTime: 2000, // 2 seconds
            },
            {
                name: 'Batch Chunk Creation',
                description: 'Create 100 chunks in batch',
                data: Array.from({ length: 100 }, () => this.generateChunk()),
                expectedTime: 1000, // 1 second
            },
            {
                name: 'Complex Hierarchy Processing',
                description: 'Process document with deep hierarchy',
                data: this.generateChunkHierarchy(5, 5), // 5 levels, 5 children each
                expectedTime: 1500, // 1.5 seconds
            },
            {
                name: 'Search Result Processing',
                description: 'Process 1000 search results',
                data: this.generateSearchResults(1000),
                expectedTime: 300, // 300ms
            },
        ];
    }

    /**
     * Reset counters (useful for test isolation)
     */
    static resetCounters(): void {
        this.chunkIdCounter = 0;
        this.templateIdCounter = 0;
        this.documentIdCounter = 0;
    }
}