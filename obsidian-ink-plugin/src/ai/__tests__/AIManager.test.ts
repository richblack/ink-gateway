import { AIManager } from '../AIManager';
import { InkGatewayClient } from '../../api/InkGatewayClient';
import { 
    AIResponse, 
    ProcessingResult, 
    ChatMessage, 
    AIManagerSettings,
    UnifiedChunk 
} from '../../types';

// Mock the InkGatewayClient
jest.mock('../../api/InkGatewayClient');

describe('AIManager', () => {
    let aiManager: AIManager;
    let mockApiClient: jest.Mocked<InkGatewayClient>;
    let settings: AIManagerSettings;

    beforeEach(() => {
        mockApiClient = new InkGatewayClient('http://test.com', 'test-key') as jest.Mocked<InkGatewayClient>;
        settings = {
            maxHistorySize: 100,
            contextWindowSize: 10,
            autoSaveHistory: true,
            enableSuggestions: true
        };
        aiManager = new AIManager(mockApiClient, settings);
    });

    afterEach(() => {
        jest.clearAllMocks();
    });

    describe('sendMessage', () => {
        it('should send message to AI and return response', async () => {
            // Arrange
            const message = 'Hello AI';
            const mockResponse: AIResponse = {
                message: 'Hello! How can I help you?',
                suggestions: [],
                actions: [],
                metadata: {
                    processingTime: 100,
                    model: 'gpt-4',
                    confidence: 0.95,
                    tokensUsed: 50
                }
            };

            mockApiClient.chatWithAI.mockResolvedValue(mockResponse);

            // Act
            const result = await aiManager.sendMessage(message);

            // Assert
            expect(mockApiClient.chatWithAI).toHaveBeenCalledWith(message, []);
            expect(result).toEqual(mockResponse);
            
            // Check that messages were added to history
            const history = aiManager.getChatHistory();
            expect(history.messages).toHaveLength(2);
            expect(history.messages[0].content).toBe(message);
            expect(history.messages[0].role).toBe('user');
            expect(history.messages[1].content).toBe(mockResponse.message);
            expect(history.messages[1].role).toBe('assistant');
        });

        it('should include context when requested', async () => {
            // Arrange
            const message = 'Analyze this content';
            const mockChunks: UnifiedChunk[] = [{
                chunkId: 'chunk1',
                contents: 'Test content',
                parent: undefined,
                page: undefined,
                isPage: false,
                isTag: false,
                isTemplate: false,
                isSlot: false,
                ref: undefined,
                tags: [],
                metadata: {},
                createdTime: new Date(),
                lastUpdated: new Date(),
                position: {
                    fileName: 'test.md',
                    lineStart: 1,
                    lineEnd: 1,
                    charStart: 0,
                    charEnd: 12
                },
                filePath: 'test.md',
                obsidianMetadata: {
                    properties: {},
                    frontmatter: {},
                    aliases: [],
                    cssClasses: []
                },
                documentId: 'doc1',
                documentScope: 'file'
            }];

            const mockResponse: AIResponse = {
                message: 'Analysis complete',
                suggestions: [],
                actions: [],
                metadata: {
                    processingTime: 200,
                    model: 'gpt-4',
                    confidence: 0.9,
                    tokensUsed: 100
                }
            };

            aiManager.updateContext(['test.md'], mockChunks);
            mockApiClient.chatWithAI.mockResolvedValue(mockResponse);

            // Act
            const result = await aiManager.sendMessage(message, true);

            // Assert
            expect(mockApiClient.chatWithAI).toHaveBeenCalledWith(message, ['chunk1']);
            expect(result).toEqual(mockResponse);
        });

        it('should handle API errors gracefully', async () => {
            // Arrange
            const message = 'Test message';
            const error = new Error('API Error');
            mockApiClient.chatWithAI.mockRejectedValue(error);

            // Act & Assert
            await expect(aiManager.sendMessage(message)).rejects.toThrow('Failed to send message: API Error');
        });
    });

    describe('processContent', () => {
        it('should process content and update context', async () => {
            // Arrange
            const content = 'This is test content to process';
            const mockResult: ProcessingResult = {
                chunks: [{
                    chunkId: 'processed1',
                    contents: content,
                    parent: undefined,
                    page: undefined,
                    isPage: false,
                    isTag: false,
                    isTemplate: false,
                    isSlot: false,
                    ref: undefined,
                    tags: ['processed'],
                    metadata: {},
                    createdTime: new Date(),
                    lastUpdated: new Date(),
                    position: {
                        fileName: 'processed.md',
                        lineStart: 1,
                        lineEnd: 1,
                        charStart: 0,
                        charEnd: content.length
                    },
                    filePath: 'processed.md',
                    obsidianMetadata: {
                        properties: {},
                        frontmatter: {},
                        aliases: [],
                        cssClasses: []
                    },
                    documentId: 'doc2',
                    documentScope: 'file'
                }],
                suggestions: [],
                improvements: []
            };

            mockApiClient.processContent.mockResolvedValue(mockResult);

            // Act
            const result = await aiManager.processContent(content);

            // Assert
            expect(mockApiClient.processContent).toHaveBeenCalledWith(content);
            expect(result).toEqual(mockResult);
            
            // Check that context was updated
            const context = aiManager.getContext();
            expect(context.relevantChunks).toHaveLength(1);
            expect(context.relevantChunks[0].chunkId).toBe('processed1');
        });

        it('should handle processing errors', async () => {
            // Arrange
            const content = 'Test content';
            const error = new Error('Processing failed');
            mockApiClient.processContent.mockRejectedValue(error);

            // Act & Assert
            await expect(aiManager.processContent(content)).rejects.toThrow('Failed to process content: Processing failed');
        });
    });

    describe('chat history management', () => {
        it('should maintain chat history', async () => {
            // Arrange
            const messages = ['Hello', 'How are you?', 'Goodbye'];
            const mockResponse: AIResponse = {
                message: 'Response',
                suggestions: [],
                actions: [],
                metadata: {
                    processingTime: 100,
                    model: 'gpt-4',
                    confidence: 0.9,
                    tokensUsed: 25
                }
            };

            mockApiClient.chatWithAI.mockResolvedValue(mockResponse);

            // Act
            for (const message of messages) {
                await aiManager.sendMessage(message);
            }

            // Assert
            const history = aiManager.getChatHistory();
            expect(history.messages).toHaveLength(6); // 3 user + 3 assistant messages
            expect(history.sessionId).toBeDefined();
            expect(history.startTime).toBeInstanceOf(Date);
            expect(history.lastActivity).toBeInstanceOf(Date);
        });

        it('should limit history size when configured', async () => {
            // Arrange
            const limitedSettings: AIManagerSettings = {
                maxHistorySize: 2
            };
            const limitedAIManager = new AIManager(mockApiClient, limitedSettings);
            
            const mockResponse: AIResponse = {
                message: 'Response',
                suggestions: [],
                actions: [],
                metadata: {
                    processingTime: 100,
                    model: 'gpt-4',
                    confidence: 0.9,
                    tokensUsed: 25
                }
            };

            mockApiClient.chatWithAI.mockResolvedValue(mockResponse);

            // Act
            await limitedAIManager.sendMessage('Message 1');
            await limitedAIManager.sendMessage('Message 2');
            await limitedAIManager.sendMessage('Message 3');

            // Assert
            const history = limitedAIManager.getChatHistory();
            expect(history.messages).toHaveLength(2); // Limited to 2 messages
        });

        it('should clear chat history', () => {
            // Arrange
            const initialHistory = aiManager.getChatHistory();
            const initialSessionId = initialHistory.sessionId;

            // Act
            aiManager.clearChatHistory();

            // Assert
            const clearedHistory = aiManager.getChatHistory();
            expect(clearedHistory.messages).toHaveLength(0);
            expect(clearedHistory.sessionId).not.toBe(initialSessionId);
            expect(clearedHistory.startTime).toBeInstanceOf(Date);
        });
    });

    describe('context management', () => {
        it('should update conversation context', () => {
            // Arrange
            const activeFiles = ['file1.md', 'file2.md'];
            const relevantChunks: UnifiedChunk[] = [{
                chunkId: 'chunk1',
                contents: 'Context content',
                parent: undefined,
                page: undefined,
                isPage: false,
                isTag: false,
                isTemplate: false,
                isSlot: false,
                ref: undefined,
                tags: [],
                metadata: {},
                createdTime: new Date(),
                lastUpdated: new Date(),
                position: {
                    fileName: 'context.md',
                    lineStart: 1,
                    lineEnd: 1,
                    charStart: 0,
                    charEnd: 15
                },
                filePath: 'context.md',
                obsidianMetadata: {
                    properties: {},
                    frontmatter: {},
                    aliases: [],
                    cssClasses: []
                },
                documentId: 'doc3',
                documentScope: 'file'
            }];

            // Act
            aiManager.updateContext(activeFiles, relevantChunks);

            // Assert
            const context = aiManager.getContext();
            expect(context.activeFiles).toEqual(activeFiles);
            expect(context.relevantChunks).toEqual(relevantChunks);
        });

        it('should get session statistics', async () => {
            // Arrange
            const mockResponse: AIResponse = {
                message: 'Test response',
                suggestions: [],
                actions: [],
                metadata: {
                    processingTime: 100,
                    model: 'gpt-4',
                    confidence: 0.9,
                    tokensUsed: 25
                }
            };

            mockApiClient.chatWithAI.mockResolvedValue(mockResponse);
            
            // Add a small delay to ensure session duration > 0
            await new Promise(resolve => setTimeout(resolve, 1));
            await aiManager.sendMessage('Test message');

            // Act
            const stats = aiManager.getSessionStats();

            // Assert
            expect(stats.messageCount).toBe(2); // user + assistant
            expect(stats.sessionDuration).toBeGreaterThanOrEqual(0);
            expect(stats.lastActivity).toBeInstanceOf(Date);
            expect(stats.contextSize).toBe(0);
        });
    });

    describe('history persistence', () => {
        it('should export chat history', async () => {
            // Arrange
            const mockResponse: AIResponse = {
                message: 'Test response',
                suggestions: [],
                actions: [],
                metadata: {
                    processingTime: 100,
                    model: 'gpt-4',
                    confidence: 0.9,
                    tokensUsed: 25
                }
            };

            mockApiClient.chatWithAI.mockResolvedValue(mockResponse);
            await aiManager.sendMessage('Test message');

            // Act
            const exported = aiManager.exportChatHistory();

            // Assert
            expect(exported).toBeDefined();
            const parsed = JSON.parse(exported);
            expect(parsed.history).toBeDefined();
            expect(parsed.context).toBeDefined();
            expect(parsed.exportTime).toBeDefined();
        });

        it('should import chat history', () => {
            // Arrange
            const mockHistory = {
                history: {
                    messages: [{
                        id: 'msg1',
                        content: 'Imported message',
                        role: 'user',
                        timestamp: new Date(),
                        metadata: {}
                    }],
                    sessionId: 'imported-session',
                    startTime: new Date(),
                    lastActivity: new Date()
                },
                context: {
                    activeFiles: ['imported.md'],
                    relevantChunks: [],
                    userPreferences: {}
                },
                exportTime: new Date()
            };

            // Act
            aiManager.importChatHistory(JSON.stringify(mockHistory));

            // Assert
            const history = aiManager.getChatHistory();
            expect(history.messages).toHaveLength(1);
            expect(history.messages[0].content).toBe('Imported message');
            expect(history.sessionId).toBe('imported-session');
        });

        it('should handle invalid import data gracefully', () => {
            // Arrange
            const invalidData = 'invalid json';

            // Act & Assert
            expect(() => aiManager.importChatHistory(invalidData)).not.toThrow();
            
            // History should remain unchanged
            const history = aiManager.getChatHistory();
            expect(history.messages).toHaveLength(0);
        });
    });
});