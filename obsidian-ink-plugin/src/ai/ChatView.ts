import { ItemView, WorkspaceLeaf, Setting, ButtonComponent, TextAreaComponent } from 'obsidian';
import { AIManager } from './AIManager';
import { ChatMessage, ChatHistory } from '../types';

export const CHAT_VIEW_TYPE = 'ink-ai-chat-view';

export class ChatView extends ItemView {
    private aiManager: AIManager;
    private chatContainer!: HTMLElement;
    private inputContainer!: HTMLElement;
    private messageInput!: TextAreaComponent;
    private sendButton!: ButtonComponent;
    private loadingIndicator!: HTMLElement;
    private errorContainer!: HTMLElement;
    private isLoading: boolean = false;

    constructor(leaf: WorkspaceLeaf, aiManager: AIManager) {
        super(leaf);
        this.aiManager = aiManager;
    }

    getViewType(): string {
        return CHAT_VIEW_TYPE;
    }

    getDisplayText(): string {
        return 'AI Chat';
    }

    getIcon(): string {
        return 'message-circle';
    }

    /**
     * Initialize the chat view UI
     * Requirement 1.1: System SHALL display AI chat interface within Obsidian
     */
    async onOpen(): Promise<void> {
        const container = this.containerEl.children[1] as HTMLElement;
        container.empty();
        container.addClass('ink-chat-view');

        // Create main layout
        this.createHeader(container);
        this.createChatContainer(container);
        this.createInputContainer(container);
        this.createErrorContainer(container);

        // Load existing chat history
        this.loadChatHistory();

        // Set up event listeners
        this.setupEventListeners();
    }

    /**
     * Clean up when view is closed
     */
    async onClose(): Promise<void> {
        // Clean up any event listeners or resources
        this.messageInput?.inputEl.removeEventListener('keydown', this.handleKeyDown.bind(this));
    }

    /**
     * Create the header section with title and controls
     */
    private createHeader(container: HTMLElement): void {
        const header = container.createDiv('ink-chat-header');
        
        const title = header.createDiv('ink-chat-title');
        title.setText('AI Assistant');

        const controls = header.createDiv('ink-chat-controls');
        
        // Clear history button
        const clearButton = new ButtonComponent(controls);
        clearButton
            .setButtonText('Clear')
            .setTooltip('Clear chat history')
            .onClick(() => this.clearChat());

        // Settings button
        const settingsButton = new ButtonComponent(controls);
        settingsButton
            .setButtonText('⚙️')
            .setTooltip('Chat settings')
            .onClick(() => this.openSettings());
    }

    /**
     * Create the chat messages container
     */
    private createChatContainer(container: HTMLElement): void {
        this.chatContainer = container.createDiv('ink-chat-container');
        this.chatContainer.addClass('ink-scrollable');
    }

    /**
     * Create the input container with message input and send button
     * Requirement 1.2: System SHALL send request to Ink-Gateway AI service when user inputs message
     */
    private createInputContainer(container: HTMLElement): void {
        this.inputContainer = container.createDiv('ink-input-container');

        // Message input area
        const inputWrapper = this.inputContainer.createDiv('ink-input-wrapper');
        
        this.messageInput = new TextAreaComponent(inputWrapper);
        this.messageInput
            .setPlaceholder('Type your message here...')
            .onChange((value) => {
                // Enable/disable send button based on input
                this.sendButton.setDisabled(!value.trim() || this.isLoading);
            });

        // Send button
        const buttonWrapper = this.inputContainer.createDiv('ink-button-wrapper');
        this.sendButton = new ButtonComponent(buttonWrapper);
        this.sendButton
            .setButtonText('Send')
            .setDisabled(true)
            .onClick(() => this.sendMessage());

        // Loading indicator
        this.loadingIndicator = this.inputContainer.createDiv('ink-loading-indicator');
        this.loadingIndicator.setText('AI is thinking...');
        this.loadingIndicator.hide();
    }

    /**
     * Create error message container
     */
    private createErrorContainer(container: HTMLElement): void {
        this.errorContainer = container.createDiv('ink-error-container');
        this.errorContainer.hide();
    }

    /**
     * Set up event listeners
     */
    private setupEventListeners(): void {
        // Handle Enter key to send message (Shift+Enter for new line)
        this.messageInput.inputEl.addEventListener('keydown', this.handleKeyDown.bind(this));
    }

    /**
     * Handle keyboard input in message area
     */
    private handleKeyDown(event: KeyboardEvent): void {
        if (event.key === 'Enter' && !event.shiftKey) {
            event.preventDefault();
            if (!this.isLoading && this.messageInput.getValue().trim()) {
                this.sendMessage();
            }
        }
    }

    /**
     * Send message to AI
     * Requirement 1.2: When user inputs message, system SHALL send request to Ink-Gateway
     */
    private async sendMessage(): Promise<void> {
        const message = this.messageInput.getValue().trim();
        if (!message || this.isLoading) return;

        try {
            this.setLoading(true);
            this.hideError();

            // Add user message to UI immediately
            this.addMessageToUI({
                id: `user_${Date.now()}`,
                content: message,
                role: 'user',
                timestamp: new Date(),
                metadata: {}
            });

            // Clear input
            this.messageInput.setValue('');

            // Send to AI
            const response = await this.aiManager.sendMessage(message);

            // Add AI response to UI
            this.addMessageToUI({
                id: `ai_${Date.now()}`,
                content: response.message,
                role: 'assistant',
                timestamp: new Date(),
                metadata: {
                    suggestions: response.suggestions,
                    actions: response.actions,
                    responseMetadata: response.metadata
                }
            });

            // Handle suggestions and actions if present
            if (response.suggestions && response.suggestions.length > 0) {
                this.displaySuggestions(response.suggestions);
            }

        } catch (error) {
            console.error('Error sending message:', error);
            this.showError(`Failed to send message: ${(error as Error).message}`);
        } finally {
            this.setLoading(false);
        }
    }

    /**
     * Add a message to the chat UI
     * Requirement 1.3: System SHALL display AI response in chat interface
     */
    private addMessageToUI(message: ChatMessage): void {
        const messageEl = this.chatContainer.createDiv('ink-chat-message');
        messageEl.addClass(`ink-message-${message.role}`);

        // Message header with timestamp
        const header = messageEl.createDiv('ink-message-header');
        const roleSpan = header.createSpan('ink-message-role');
        roleSpan.setText(message.role === 'user' ? 'You' : 'AI Assistant');
        
        const timeSpan = header.createSpan('ink-message-time');
        timeSpan.setText(this.formatTime(message.timestamp));

        // Message content
        const content = messageEl.createDiv('ink-message-content');
        content.setText(message.content);

        // Add metadata if present (for AI messages)
        if (message.role === 'assistant' && message.metadata.responseMetadata) {
            const metadata = messageEl.createDiv('ink-message-metadata');
            const metaInfo = message.metadata.responseMetadata;
            metadata.setText(`Model: ${metaInfo.model} | Confidence: ${(metaInfo.confidence * 100).toFixed(1)}%`);
        }

        // Scroll to bottom
        this.scrollToBottom();
    }

    /**
     * Display content suggestions
     */
    private displaySuggestions(suggestions: any[]): void {
        if (suggestions.length === 0) return;

        const suggestionsEl = this.chatContainer.createDiv('ink-suggestions');
        const title = suggestionsEl.createDiv('ink-suggestions-title');
        title.setText('💡 Suggestions:');

        suggestions.forEach((suggestion, index) => {
            const suggestionEl = suggestionsEl.createDiv('ink-suggestion');
            suggestionEl.setText(`${index + 1}. ${suggestion.content}`);
            suggestionEl.addClass('ink-clickable');
            
            suggestionEl.addEventListener('click', () => {
                this.applySuggestion(suggestion);
            });
        });

        this.scrollToBottom();
    }

    /**
     * Apply a suggestion
     */
    private applySuggestion(suggestion: any): void {
        // This could be extended to apply suggestions to the current document
        this.messageInput.setValue(suggestion.content);
        this.messageInput.inputEl.focus();
    }

    /**
     * Load existing chat history
     * Requirement 1.4: System SHALL maintain chat history during session
     */
    private loadChatHistory(): void {
        const history = this.aiManager.getChatHistory();
        
        if (history.messages.length === 0) {
            // Show welcome message
            this.showWelcomeMessage();
            return;
        }

        // Display existing messages
        history.messages.forEach(message => {
            this.addMessageToUI(message);
        });
    }

    /**
     * Show welcome message for new sessions
     */
    private showWelcomeMessage(): void {
        const welcomeEl = this.chatContainer.createDiv('ink-welcome-message');
        welcomeEl.innerHTML = `
            <div class="ink-welcome-title">👋 Welcome to AI Assistant</div>
            <div class="ink-welcome-text">
                I can help you with:
                <ul>
                    <li>Analyzing your notes and content</li>
                    <li>Answering questions about your knowledge base</li>
                    <li>Suggesting improvements and connections</li>
                    <li>Processing and organizing information</li>
                </ul>
                Start by typing a message below!
            </div>
        `;
    }

    /**
     * Clear chat history
     */
    private clearChat(): void {
        this.aiManager.clearChatHistory();
        this.chatContainer.empty();
        this.showWelcomeMessage();
    }

    /**
     * Set loading state
     */
    private setLoading(loading: boolean): void {
        this.isLoading = loading;
        
        if (loading) {
            this.loadingIndicator.show();
            this.sendButton.setDisabled(true);
            this.messageInput.setDisabled(true);
        } else {
            this.loadingIndicator.hide();
            this.sendButton.setDisabled(!this.messageInput.getValue().trim());
            this.messageInput.setDisabled(false);
            this.messageInput.inputEl.focus();
        }
    }

    /**
     * Show error message
     */
    private showError(message: string): void {
        this.errorContainer.empty();
        this.errorContainer.setText(`❌ ${message}`);
        this.errorContainer.show();
        
        // Auto-hide after 5 seconds
        setTimeout(() => {
            this.hideError();
        }, 5000);
    }

    /**
     * Hide error message
     */
    private hideError(): void {
        this.errorContainer.hide();
    }

    /**
     * Scroll chat to bottom
     */
    private scrollToBottom(): void {
        this.chatContainer.scrollTop = this.chatContainer.scrollHeight;
    }

    /**
     * Format timestamp for display
     */
    private formatTime(date: Date): string {
        return date.toLocaleTimeString([], { 
            hour: '2-digit', 
            minute: '2-digit' 
        });
    }

    /**
     * Open chat settings (placeholder)
     */
    private openSettings(): void {
        // This could open a modal with chat-specific settings
        console.log('Chat settings not implemented yet');
    }

    /**
     * Get current session statistics for display
     */
    getSessionStats(): {
        messageCount: number;
        sessionDuration: number;
        lastActivity: Date;
    } {
        return this.aiManager.getSessionStats();
    }

    /**
     * Export chat history
     */
    exportHistory(): string {
        return this.aiManager.exportChatHistory();
    }

    /**
     * Import chat history
     */
    importHistory(data: string): void {
        this.aiManager.importChatHistory(data);
        this.chatContainer.empty();
        this.loadChatHistory();
    }
}