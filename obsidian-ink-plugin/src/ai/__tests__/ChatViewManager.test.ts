import { Plugin, WorkspaceLeaf, TFile } from 'obsidian';
import { ChatViewManager } from '../ChatViewManager';
import { ChatView, CHAT_VIEW_TYPE } from '../ChatView';
import { AIManager } from '../AIManager';
import { InkGatewayClient } from '../../api/InkGatewayClient';
import { AIManagerSettings } from '../../types';

// Mock Obsidian components
const mockWorkspace = {
    getLeavesOfType: jest.fn(),
    revealLeaf: jest.fn(),
    getRightLeaf: jest.fn()
};

const mockApp = {
    workspace: mockWorkspace
};

jest.mock('obsidian', () => ({
    Plugin: jest.fn().mockImplementation(() => ({
        app: mockApp,
        registerView: jest.fn(),
        addRibbonIcon: jest.fn(),
        addCommand: jest.fn()
    })),
    WorkspaceLeaf: jest.fn().mockImplementation(() => ({
        setViewState: jest.fn(),
        detach: jest.fn(),
        view: null
    })),
    TFile: jest.fn().mockImplementation((path?: string) => ({
        path: path || 'test.md'
    })),
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
    }
}));

// Mock ChatView and AIManager
jest.mock('../ChatView');
jest.mock('../AIManager');
jest.mock('../../api/InkGatewayClient');

describe('ChatViewManager', () => {
    let chatViewManager: ChatViewManager;
    let mockPlugin: jest.Mocked<Plugin>;
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
        
        // Create mock plugin with proper typing
        mockPlugin = {
            app: mockApp,
            registerView: jest.fn(),
            addRibbonIcon: jest.fn(),
            addCommand: jest.fn()
        } as any;

        // Mock AIManager methods
        mockAIManager.clearChatHistory = jest.fn();
        mockAIManager.updateContext = jest.fn();

        // Reset workspace mocks
        mockWorkspace.getLeavesOfType.mockReset();
        mockWorkspace.revealLeaf.mockReset();
        mockWorkspace.getRightLeaf.mockReset();

        // Create ChatViewManager instance
        chatViewManager = new ChatViewManager(mockPlugin, mockAIManager);
    });

    afterEach(() => {
        jest.clearAllMocks();
    });

    describe('initialization', () => {
        it('should register view type and commands', async () => {
            // Act
            await chatViewManager.initialize();

            // Assert
            expect(mockPlugin.registerView).toHaveBeenCalledWith(
                CHAT_VIEW_TYPE,
                expect.any(Function)
            );
            expect(mockPlugin.addRibbonIcon).toHaveBeenCalledWith(
                'message-circle',
                'Open AI Chat',
                expect.any(Function)
            );
            expect(mockPlugin.addCommand).toHaveBeenCalledTimes(3);
        });

        it('should register correct commands', async () => {
            // Act
            await chatViewManager.initialize();

            // Assert
            const commandCalls = mockPlugin.addCommand.mock.calls;
            expect(commandCalls[0][0]).toEqual({
                id: 'open-ai-chat',
                name: 'Open AI Chat',
                callback: expect.any(Function)
            });
            expect(commandCalls[1][0]).toEqual({
                id: 'toggle-ai-chat',
                name: 'Toggle AI Chat',
                callback: expect.any(Function)
            });
            expect(commandCalls[2][0]).toEqual({
                id: 'clear-ai-chat',
                name: 'Clear AI Chat History',
                callback: expect.any(Function)
            });
        });
    });

    describe('chat view management', () => {
        beforeEach(async () => {
            await chatViewManager.initialize();
        });

        it('should open new chat view when none exists', async () => {
            // Arrange
            const mockLeaf = {
                setViewState: jest.fn().mockResolvedValue(undefined),
                detach: jest.fn(),
                view: null
            };
            const mockChatView = new ChatView(mockLeaf as any, mockAIManager) as jest.Mocked<ChatView>;
            mockLeaf.view = mockChatView;

            mockWorkspace.getLeavesOfType.mockReturnValue([]);
            mockWorkspace.getRightLeaf.mockReturnValue(mockLeaf as any);

            // Act
            const result = await chatViewManager.openChatView();

            // Assert
            expect(mockWorkspace.getLeavesOfType).toHaveBeenCalledWith(CHAT_VIEW_TYPE);
            expect(mockWorkspace.getRightLeaf).toHaveBeenCalledWith(false);
            expect(mockLeaf.setViewState).toHaveBeenCalledWith({
                type: CHAT_VIEW_TYPE,
                active: true
            });
            expect(result).toBe(mockChatView);
        });

        it('should focus existing chat view when it exists', async () => {
            // Arrange
            const mockLeaf = {
                setViewState: jest.fn(),
                detach: jest.fn(),
                view: null
            };
            const mockChatView = new ChatView(mockLeaf as any, mockAIManager) as jest.Mocked<ChatView>;
            mockLeaf.view = mockChatView;

            mockWorkspace.getLeavesOfType.mockReturnValue([mockLeaf as any]);

            // Act
            const result = await chatViewManager.openChatView();

            // Assert
            expect(mockWorkspace.revealLeaf).toHaveBeenCalledWith(mockLeaf);
            expect(result).toBe(mockChatView);
        });

        it('should toggle chat view - close when open', async () => {
            // Arrange
            const mockLeaf = {
                setViewState: jest.fn(),
                detach: jest.fn(),
                view: null
            };
            mockWorkspace.getLeavesOfType.mockReturnValue([mockLeaf as any]);

            // Act
            await chatViewManager.toggleChatView();

            // Assert
            expect(mockLeaf.detach).toHaveBeenCalled();
        });

        it('should toggle chat view - open when closed', async () => {
            // Arrange
            const mockLeaf = {
                setViewState: jest.fn().mockResolvedValue(undefined),
                detach: jest.fn(),
                view: null
            };
            const mockChatView = new ChatView(mockLeaf as any, mockAIManager) as jest.Mocked<ChatView>;
            mockLeaf.view = mockChatView;

            // First call returns empty (closed), second call for opening
            mockWorkspace.getLeavesOfType
                .mockReturnValueOnce([])
                .mockReturnValueOnce([]);
            mockWorkspace.getRightLeaf.mockReturnValue(mockLeaf as any);

            // Act
            await chatViewManager.toggleChatView();

            // Assert
            expect(mockWorkspace.getRightLeaf).toHaveBeenCalledWith(false);
            expect(mockLeaf.setViewState).toHaveBeenCalledWith({
                type: CHAT_VIEW_TYPE,
                active: true
            });
        });

        it('should close chat view', () => {
            // Arrange
            const mockLeaf1 = {
                setViewState: jest.fn(),
                detach: jest.fn(),
                view: null
            };
            const mockLeaf2 = {
                setViewState: jest.fn(),
                detach: jest.fn(),
                view: null
            };
            mockWorkspace.getLeavesOfType.mockReturnValue([mockLeaf1 as any, mockLeaf2 as any]);

            // Act
            chatViewManager.closeChatView();

            // Assert
            expect(mockLeaf1.detach).toHaveBeenCalled();
            expect(mockLeaf2.detach).toHaveBeenCalled();
            expect(chatViewManager.getActiveChatView()).toBeNull();
        });
    });

    describe('chat history management', () => {
        it('should clear chat history', () => {
            // Arrange
            const mockLeaf = {
                setViewState: jest.fn(),
                detach: jest.fn(),
                view: null
            };
            const mockChatView = new ChatView(mockLeaf as any, mockAIManager) as jest.Mocked<ChatView>;
            chatViewManager['activeChatView'] = mockChatView;

            // Act
            chatViewManager.clearChatHistory();

            // Assert
            expect(mockAIManager.clearChatHistory).toHaveBeenCalled();
        });

        it('should not clear history when no active view', () => {
            // Arrange
            chatViewManager['activeChatView'] = null;

            // Act
            chatViewManager.clearChatHistory();

            // Assert
            expect(mockAIManager.clearChatHistory).not.toHaveBeenCalled();
        });
    });

    describe('context management', () => {
        it('should update chat context with file', () => {
            // Arrange
            const mockFile = { path: 'test.md' } as any;

            // Act
            chatViewManager.updateChatContext(mockFile);

            // Assert
            expect(mockAIManager.updateContext).toHaveBeenCalledWith(['test.md']);
        });

        it('should not update context with null file', () => {
            // Act
            chatViewManager.updateChatContext(null);

            // Assert
            expect(mockAIManager.updateContext).not.toHaveBeenCalled();
        });
    });

    describe('layout change handling', () => {
        it('should update active view reference on layout change', () => {
            // Arrange
            const mockLeaf = {
                setViewState: jest.fn(),
                detach: jest.fn(),
                view: null
            };
            const mockChatView = new ChatView(mockLeaf as any, mockAIManager) as jest.Mocked<ChatView>;
            mockLeaf.view = mockChatView;

            mockWorkspace.getLeavesOfType.mockReturnValue([mockLeaf as any]);

            // Act
            chatViewManager.onLayoutChange();

            // Assert
            expect(chatViewManager.getActiveChatView()).toBe(mockChatView);
        });

        it('should clear active view reference when no views exist', () => {
            // Arrange
            mockWorkspace.getLeavesOfType.mockReturnValue([]);

            // Act
            chatViewManager.onLayoutChange();

            // Assert
            expect(chatViewManager.getActiveChatView()).toBeNull();
        });
    });

    describe('history import/export', () => {
        it('should export chat history from active view', () => {
            // Arrange
            const mockLeaf = {
                setViewState: jest.fn(),
                detach: jest.fn(),
                view: null
            };
            const mockChatView = new ChatView(mockLeaf as any, mockAIManager) as jest.Mocked<ChatView>;
            const mockExportData = JSON.stringify({ test: 'data' });
            
            mockChatView.exportHistory = jest.fn().mockReturnValue(mockExportData);
            chatViewManager['activeChatView'] = mockChatView;

            // Act
            const result = chatViewManager.exportChatHistory();

            // Assert
            expect(result).toBe(mockExportData);
            expect(mockChatView.exportHistory).toHaveBeenCalled();
        });

        it('should return null when no active view for export', () => {
            // Arrange
            chatViewManager['activeChatView'] = null;

            // Act
            const result = chatViewManager.exportChatHistory();

            // Assert
            expect(result).toBeNull();
        });

        it('should import chat history to active view', async () => {
            // Arrange
            const mockLeaf = {
                setViewState: jest.fn(),
                detach: jest.fn(),
                view: null
            };
            const mockChatView = new ChatView(mockLeaf as any, mockAIManager) as jest.Mocked<ChatView>;
            const mockImportData = JSON.stringify({ test: 'imported' });
            
            mockChatView.importHistory = jest.fn();
            chatViewManager['activeChatView'] = mockChatView;

            // Act
            await chatViewManager.importChatHistory(mockImportData);

            // Assert
            expect(mockChatView.importHistory).toHaveBeenCalledWith(mockImportData);
        });

        it('should open view before importing when no active view', async () => {
            // Arrange
            const mockLeaf = {
                setViewState: jest.fn().mockResolvedValue(undefined),
                detach: jest.fn(),
                view: null
            };
            const mockChatView = new ChatView(mockLeaf as any, mockAIManager) as jest.Mocked<ChatView>;
            const mockImportData = JSON.stringify({ test: 'imported' });
            
            mockLeaf.view = mockChatView;
            mockChatView.importHistory = jest.fn();
            
            chatViewManager['activeChatView'] = null;
            mockWorkspace.getLeavesOfType.mockReturnValue([]);
            mockWorkspace.getRightLeaf.mockReturnValue(mockLeaf as any);

            // Act
            await chatViewManager.importChatHistory(mockImportData);

            // Assert
            expect(mockLeaf.setViewState).toHaveBeenCalled();
            expect(mockChatView.importHistory).toHaveBeenCalledWith(mockImportData);
        });
    });

    describe('session statistics', () => {
        it('should get session stats from active view', () => {
            // Arrange
            const mockLeaf = {
                setViewState: jest.fn(),
                detach: jest.fn(),
                view: null
            };
            const mockChatView = new ChatView(mockLeaf as any, mockAIManager) as jest.Mocked<ChatView>;
            const mockStats = {
                messageCount: 5,
                sessionDuration: 300000,
                lastActivity: new Date()
            };
            
            mockChatView.getSessionStats = jest.fn().mockReturnValue(mockStats);
            chatViewManager['activeChatView'] = mockChatView;

            // Act
            const result = chatViewManager.getSessionStats();

            // Assert
            expect(result).toBe(mockStats);
            expect(mockChatView.getSessionStats).toHaveBeenCalled();
        });

        it('should return null when no active view for stats', () => {
            // Arrange
            chatViewManager['activeChatView'] = null;

            // Act
            const result = chatViewManager.getSessionStats();

            // Assert
            expect(result).toBeNull();
        });
    });

    describe('cleanup', () => {
        it('should cleanup properly', () => {
            // Arrange
            const mockLeaf = {
                setViewState: jest.fn(),
                detach: jest.fn(),
                view: null
            };
            mockWorkspace.getLeavesOfType.mockReturnValue([mockLeaf as any]);

            // Act
            chatViewManager.cleanup();

            // Assert
            expect(mockLeaf.detach).toHaveBeenCalled();
            expect(chatViewManager.getActiveChatView()).toBeNull();
        });
    });
});