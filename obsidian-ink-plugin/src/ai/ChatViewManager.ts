import { Plugin, WorkspaceLeaf, TFile } from 'obsidian';
import { ChatView, CHAT_VIEW_TYPE } from './ChatView';
import { AIManager } from './AIManager';

export class ChatViewManager {
    private plugin: Plugin;
    private aiManager: AIManager;
    private activeChatView: ChatView | null = null;

    constructor(plugin: Plugin, aiManager: AIManager) {
        this.plugin = plugin;
        this.aiManager = aiManager;
    }

    /**
     * Initialize the chat view manager
     * Requirement 1.1: System SHALL display AI chat interface within Obsidian
     */
    async initialize(): Promise<void> {
        // Register the chat view type
        this.plugin.registerView(
            CHAT_VIEW_TYPE,
            (leaf) => new ChatView(leaf, this.aiManager)
        );

        // Add ribbon icon to open chat
        this.plugin.addRibbonIcon('message-circle', 'Open AI Chat', () => {
            this.openChatView();
        });

        // Add command to open chat
        this.plugin.addCommand({
            id: 'open-ai-chat',
            name: 'Open AI Chat',
            callback: () => {
                this.openChatView();
            }
        });

        // Add command to toggle chat view
        this.plugin.addCommand({
            id: 'toggle-ai-chat',
            name: 'Toggle AI Chat',
            callback: () => {
                this.toggleChatView();
            }
        });

        // Add command to clear chat history
        this.plugin.addCommand({
            id: 'clear-ai-chat',
            name: 'Clear AI Chat History',
            callback: () => {
                this.clearChatHistory();
            }
        });
    }

    /**
     * Open or focus the chat view
     * Requirement 1.1: When user opens plugin, system SHALL display AI chat interface
     */
    async openChatView(): Promise<ChatView> {
        const existing = this.plugin.app.workspace.getLeavesOfType(CHAT_VIEW_TYPE);
        
        if (existing.length > 0) {
            // Focus existing chat view
            this.plugin.app.workspace.revealLeaf(existing[0]);
            this.activeChatView = existing[0].view as ChatView;
        } else {
            // Create new chat view
            const leaf = this.plugin.app.workspace.getRightLeaf(false);
            await leaf?.setViewState({
                type: CHAT_VIEW_TYPE,
                active: true
            });
            this.activeChatView = leaf?.view as ChatView;
        }

        return this.activeChatView;
    }

    /**
     * Toggle chat view visibility
     */
    async toggleChatView(): Promise<void> {
        const existing = this.plugin.app.workspace.getLeavesOfType(CHAT_VIEW_TYPE);
        
        if (existing.length > 0) {
            // Close existing chat view
            existing[0].detach();
            this.activeChatView = null;
        } else {
            // Open new chat view
            await this.openChatView();
        }
    }

    /**
     * Close the chat view
     */
    closeChatView(): void {
        const existing = this.plugin.app.workspace.getLeavesOfType(CHAT_VIEW_TYPE);
        existing.forEach(leaf => leaf.detach());
        this.activeChatView = null;
    }

    /**
     * Get the active chat view
     */
    getActiveChatView(): ChatView | null {
        return this.activeChatView;
    }

    /**
     * Clear chat history in active view
     */
    clearChatHistory(): void {
        if (this.activeChatView) {
            this.aiManager.clearChatHistory();
            // The view will automatically update when it detects the history change
        }
    }

    /**
     * Send a message programmatically
     */
    async sendMessage(message: string): Promise<void> {
        if (!this.activeChatView) {
            await this.openChatView();
        }
        
        // The view handles message sending through its UI
        // This could be extended to allow programmatic message sending
    }

    /**
     * Update chat context with current file
     */
    updateChatContext(file: TFile | null): void {
        if (file) {
            const activeFiles = [file.path];
            this.aiManager.updateContext(activeFiles);
        }
    }

    /**
     * Handle workspace layout change
     */
    onLayoutChange(): void {
        // Update active chat view reference
        const existing = this.plugin.app.workspace.getLeavesOfType(CHAT_VIEW_TYPE);
        if (existing.length > 0) {
            this.activeChatView = existing[0].view as ChatView;
        } else {
            this.activeChatView = null;
        }
    }

    /**
     * Export chat history from active view
     */
    exportChatHistory(): string | null {
        if (this.activeChatView) {
            return this.activeChatView.exportHistory();
        }
        return null;
    }

    /**
     * Import chat history to active view
     */
    async importChatHistory(data: string): Promise<void> {
        if (!this.activeChatView) {
            await this.openChatView();
        }
        
        if (this.activeChatView) {
            this.activeChatView.importHistory(data);
        }
    }

    /**
     * Get session statistics from active view
     */
    getSessionStats(): any {
        if (this.activeChatView) {
            return this.activeChatView.getSessionStats();
        }
        return null;
    }

    /**
     * Cleanup when plugin is disabled
     */
    cleanup(): void {
        this.closeChatView();
        this.activeChatView = null;
    }
}