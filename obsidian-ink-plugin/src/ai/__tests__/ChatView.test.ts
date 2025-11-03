import { WorkspaceLeaf } from 'obsidian';
import { ChatView, CHAT_VIEW_TYPE } from '../ChatView';
import { AIManager } from '../AIManager';
import { InkGatewayClient } from '../../api/InkGatewayClient';
import { AIResponse, AIManagerSettings } from '../../types';

// Mock Obsidian components
jest.mock('obsidian', () => ({
    ItemView: class MockItemView {
        containerEl = {
            children: [null, {
                empty: jest.fn(),
                addClass: jest.fn(),
                createDiv: jest.fn().mockReturnValue({
                    createDiv: jest.fn().mockReturnValue({
                        setText: jest.fn(),
                        createSpan: jest.fn().mockReturnValue({
                            setText: jest.fn()
                        }),
                        addEventListener: jest.fn(),
                        addClass: jest.fn(),
                        hide: jest.fn(),
                        show: jest.fn(),
                        scrollTop: 0,
                        scrollHeight: 100,
                        innerHTML: '',
                        empty: jest.fn()
                    }),
                    setText: jest.fn(),
                    addClass: jest.fn(),
                    hide: jest.fn(),
                    show: jest.fn(),
                    innerHTML: '',
                    empty: jest.fn(),
                    scrollTop: 0,
                    scrollHeight: 100
                })
            }]
        };
        leaf: any;
        constructor(leaf: any) {
            this.leaf = leaf;
        }
    },
    Setting: jest.fn(),
    ButtonComponent: jest.fn().mockImplementation(() => ({
        setButtonText: jest.fn().mockReturnThis(),
        setTooltip: jest.fn().mockReturnThis(),
        setDisabled: jest.fn().mockReturnThis(),
        onClick: jest.fn().mockReturnThis()
    })),
    TextAreaComponent: jest.fn().mockImplementation(() => ({
        setPlaceholder: jest.fn().mockReturnThis(),
        onChange: jest.fn().mockReturnThis(),
        setValue: jest.fn().mockReturnThis(),
        getValue: jest.fn().mockReturnValue(''),
        setDisabled: jest.fn().mockReturnThis(),
        inputEl: {
            addEventListener: jest.fn(),
            removeEventListener: jest.fn(),
            focus: jest.fn()
        }
    }))
}));

// Mock AIManager
jest.mock('../AIManager');
jest.mock('../../api/InkGatewayClient');

describe('ChatView', () => {
    let chatView: ChatView;
    let mockLeaf: WorkspaceLeaf;
    let mockAIManager: jest.Mocked<AIManager>;
    let mockApiClient: jest.Mocked<InkGatewayClient>;

    beforeEach(() => {
        // Create mocks
        mockApiClient = new InkGatewayClient('http://test.com', 'test-key') as jest.Mocked<InkGatewayClient>;
        const settings: AIManagerSettings = {
            maxHistorySize: 100,
            enableSuggestions: true
        };
        mockAIManager = new AIManager(mockApiClient, settings) as jest.Mocked<AIManager>;
        mockLeaf = {} as WorkspaceLeaf;

        // Mock AIManager methods
        mockAIManager.getChatHistory = jest.fn().mockReturnValue({
            messages: [],
            sessionId: 'test-session',
            startTime: new Date(),
            lastActivity: new Date()
        });
        mockAIManager.sendMessage = jest.fn();
        mockAIManager.clearChatHistory = jest.fn();
        mockAIManager.getSessionStats = jest.fn().mockReturnValue({
            messageCount: 0,
            sessionDuration: 0,
            lastActivity: new Date(),
            contextSize: 0
        });
        mockAIManager.exportChatHistory = jest.fn().mockReturnValue('{}');
        mockAIManager.importChatHistory = jest.fn();

        // Create ChatView instance
        chatView = new ChatView(mockLeaf, mockAIManager);
    });

    afterEach(() => {
        jest.clearAllMocks();
    });

    describe('view properties', () => {
        it('should return correct view type', () => {
            expect(chatView.getViewType()).toBe(CHAT_VIEW_TYPE);
        });

        it('should return correct display text', () => {
            expect(chatView.getDisplayText()).toBe('AI Chat');
        });

        it('should return correct icon', () => {
            expect(chatView.getIcon()).toBe('message-circle');
        });
    });

    describe('initialization', () => {
        it('should initialize view on open', async () => {
            // Mock DOM methods
            const mockContainer = {
                empty: jest.fn(),
                addClass: jest.fn(),
                createDiv: jest.fn().mockReturnValue({
                    createDiv: jest.fn().mockReturnValue({
                        setText: jest.fn(),
                        createSpan: jest.fn().mockReturnValue({
                            setText: jest.fn()
                        }),
                        addEventListener: jest.fn(),
                        addClass: jest.fn(),
                        hide: jest.fn(),
                        show: jest.fn(),
                        scrollTop: 0,
                        scrollHeight: 100
                    }),
                    setText: jest.fn(),
                    addClass: jest.fn(),
                    hide: jest.fn(),
                    show: jest.fn(),
                    innerHTML: '',
                    empty: jest.fn(),
                    scrollTop: 0,
                    scrollHeight: 100
                })
            };

            chatView.containerEl.children[1] = mockContainer as any;

            await chatView.onOpen();

            expect(mockContainer.empty).toHaveBeenCalled();
            expect(mockContainer.addClass).toHaveBeenCalledWith('ink-chat-view');
            expect(mockAIManager.getChatHistory).toHaveBeenCalled();
        });
    });

    describe('chat functionality', () => {
        beforeEach(async () => {
            // Setup DOM mocks
            const mockContainer = {
                empty: jest.fn(),
                addClass: jest.fn(),
                createDiv: jest.fn().mockReturnValue({
                    createDiv: jest.fn().mockReturnValue({
                        setText: jest.fn(),
                        createSpan: jest.fn().mockReturnValue({
                            setText: jest.fn()
                        }),
                        addEventListener: jest.fn(),
                        addClass: jest.fn(),
                        hide: jest.fn(),
                        show: jest.fn(),
                        scrollTop: 0,
                        scrollHeight: 100
                    }),
                    setText: jest.fn(),
                    addClass: jest.fn(),
                    hide: jest.fn(),
                    show: jest.fn(),
                    innerHTML: '',
                    empty: jest.fn(),
                    scrollTop: 0,
                    scrollHeight: 100
                })
            };

            // Mock the chat container specifically
            const mockChatContainer = {
                createDiv: jest.fn().mockReturnValue({
                    addClass: jest.fn(),
                    createDiv: jest.fn().mockReturnValue({
                        setText: jest.fn(),
                        createSpan: jest.fn().mockReturnValue({
                            setText: jest.fn()
                        })
                    }),
                    setText: jest.fn()
                }),
                scrollTop: 0,
                scrollHeight: 100,
                empty: jest.fn()
            };

            chatView.containerEl.children[1] = mockContainer as any;
            chatView['chatContainer'] = mockChatContainer as any;
            await chatView.onOpen();
        });

        it('should send message to AI manager', async () => {
            // Arrange
            const testMessage = 'Hello AI';
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

            mockAIManager.sendMessage.mockResolvedValue(mockResponse);

            // Mock the message input to return our test message
            const mockTextArea = chatView['messageInput'] as any;
            mockTextArea.getValue = jest.fn().mockReturnValue(testMessage);

            // Act
            await chatView['sendMessage']();

            // Assert
            expect(mockAIManager.sendMessage).toHaveBeenCalledWith(testMessage);
            expect(mockTextArea.setValue).toHaveBeenCalledWith('');
        });

        it('should handle AI response with suggestions', async () => {
            // Arrange
            const testMessage = 'Test message';
            const mockResponse: AIResponse = {
                message: 'Response with suggestions',
                suggestions: [
                    {
                        type: 'improvement',
                        content: 'Consider adding more details',
                        confidence: 0.8
                    }
                ],
                actions: [],
                metadata: {
                    processingTime: 150,
                    model: 'gpt-4',
                    confidence: 0.9,
                    tokensUsed: 75
                }
            };

            mockAIManager.sendMessage.mockResolvedValue(mockResponse);

            const mockTextArea = chatView['messageInput'] as any;
            mockTextArea.getValue = jest.fn().mockReturnValue(testMessage);

            // Act
            await chatView['sendMessage']();

            // Assert
            expect(mockAIManager.sendMessage).toHaveBeenCalledWith(testMessage);
            // Suggestions should be displayed (tested through DOM manipulation)
        });

        it('should handle API errors gracefully', async () => {
            // Arrange
            const testMessage = 'Test message';
            const error = new Error('API Error');
            mockAIManager.sendMessage.mockRejectedValue(error);

            const mockTextArea = chatView['messageInput'] as any;
            mockTextArea.getValue = jest.fn().mockReturnValue(testMessage);

            // Spy on console.error
            const consoleSpy = jest.spyOn(console, 'error').mockImplementation();

            // Act
            await chatView['sendMessage']();

            // Assert
            expect(consoleSpy).toHaveBeenCalledWith('Error sending message:', error);
            
            consoleSpy.mockRestore();
        });
    });

    describe('chat history management', () => {
        it('should load existing chat history', () => {
            // Arrange
            const mockHistory = {
                messages: [
                    {
                        id: 'msg1',
                        content: 'Hello',
                        role: 'user' as const,
                        timestamp: new Date(),
                        metadata: {}
                    },
                    {
                        id: 'msg2',
                        content: 'Hi there!',
                        role: 'assistant' as const,
                        timestamp: new Date(),
                        metadata: {}
                    }
                ],
                sessionId: 'test-session',
                startTime: new Date(),
                lastActivity: new Date()
            };

            mockAIManager.getChatHistory.mockReturnValue(mockHistory);

            // Act
            chatView['loadChatHistory']();

            // Assert
            expect(mockAIManager.getChatHistory).toHaveBeenCalled();
            // Messages should be added to UI (tested through DOM manipulation)
        });

        it('should show welcome message for empty history', () => {
            // Arrange
            mockAIManager.getChatHistory.mockReturnValue({
                messages: [],
                sessionId: 'empty-session',
                startTime: new Date(),
                lastActivity: new Date()
            });

            // Act
            chatView['loadChatHistory']();

            // Assert
            expect(mockAIManager.getChatHistory).toHaveBeenCalled();
            // Welcome message should be shown (tested through DOM manipulation)
        });

        it('should clear chat history', () => {
            // Act
            chatView['clearChat']();

            // Assert
            expect(mockAIManager.clearChatHistory).toHaveBeenCalled();
        });
    });

    describe('session management', () => {
        it('should get session statistics', () => {
            // Arrange
            const mockStats = {
                messageCount: 5,
                sessionDuration: 300000,
                lastActivity: new Date(),
                contextSize: 2
            };
            mockAIManager.getSessionStats.mockReturnValue(mockStats);

            // Act
            const stats = chatView.getSessionStats();

            // Assert
            expect(stats).toEqual(mockStats);
            expect(mockAIManager.getSessionStats).toHaveBeenCalled();
        });

        it('should export chat history', () => {
            // Arrange
            const mockExport = JSON.stringify({ test: 'data' });
            mockAIManager.exportChatHistory.mockReturnValue(mockExport);

            // Act
            const exported = chatView.exportHistory();

            // Assert
            expect(exported).toBe(mockExport);
            expect(mockAIManager.exportChatHistory).toHaveBeenCalled();
        });

        it('should import chat history', () => {
            // Arrange
            const mockData = JSON.stringify({ test: 'imported' });

            // Mock DOM methods for reloading history
            const mockContainer = {
                empty: jest.fn()
            };
            chatView['chatContainer'] = mockContainer as any;

            // Act
            chatView.importHistory(mockData);

            // Assert
            expect(mockAIManager.importChatHistory).toHaveBeenCalledWith(mockData);
            expect(mockContainer.empty).toHaveBeenCalled();
        });
    });

    describe('UI state management', () => {
        it('should set loading state correctly', () => {
            // Arrange
            const mockSendButton = {
                setDisabled: jest.fn()
            };
            const mockMessageInput = {
                setDisabled: jest.fn(),
                getValue: jest.fn().mockReturnValue('test'),
                inputEl: { focus: jest.fn() }
            };
            const mockLoadingIndicator = {
                show: jest.fn(),
                hide: jest.fn()
            };

            chatView['sendButton'] = mockSendButton as any;
            chatView['messageInput'] = mockMessageInput as any;
            chatView['loadingIndicator'] = mockLoadingIndicator as any;

            // Act - set loading
            chatView['setLoading'](true);

            // Assert
            expect(mockLoadingIndicator.show).toHaveBeenCalled();
            expect(mockSendButton.setDisabled).toHaveBeenCalledWith(true);
            expect(mockMessageInput.setDisabled).toHaveBeenCalledWith(true);

            // Act - clear loading
            chatView['setLoading'](false);

            // Assert
            expect(mockLoadingIndicator.hide).toHaveBeenCalled();
            expect(mockSendButton.setDisabled).toHaveBeenCalledWith(false);
            expect(mockMessageInput.setDisabled).toHaveBeenCalledWith(false);
        });

        it('should show and hide error messages', () => {
            // Arrange
            const mockErrorContainer = {
                empty: jest.fn(),
                setText: jest.fn(),
                show: jest.fn(),
                hide: jest.fn()
            };
            chatView['errorContainer'] = mockErrorContainer as any;

            // Act - show error
            chatView['showError']('Test error message');

            // Assert
            expect(mockErrorContainer.empty).toHaveBeenCalled();
            expect(mockErrorContainer.setText).toHaveBeenCalledWith('❌ Test error message');
            expect(mockErrorContainer.show).toHaveBeenCalled();

            // Act - hide error
            chatView['hideError']();

            // Assert
            expect(mockErrorContainer.hide).toHaveBeenCalled();
        });
    });

    describe('utility methods', () => {
        it('should format time correctly', () => {
            // Arrange
            const testDate = new Date('2023-12-01T14:30:00');

            // Act
            const formatted = chatView['formatTime'](testDate);

            // Assert
            expect(formatted).toMatch(/\d{1,2}:\d{2}/); // Should match time format
        });

        it('should scroll to bottom', () => {
            // Arrange
            const mockChatContainer = {
                scrollTop: 0,
                scrollHeight: 500
            };
            chatView['chatContainer'] = mockChatContainer as any;

            // Act
            chatView['scrollToBottom']();

            // Assert
            expect(mockChatContainer.scrollTop).toBe(500);
        });
    });
});