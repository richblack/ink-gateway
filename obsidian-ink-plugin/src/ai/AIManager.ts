import { InkGatewayClient } from '../api/InkGatewayClient';
import { 
    AIResponse, 
    ProcessingResult, 
    ChatMessage, 
    ChatHistory, 
    ConversationContext,
    AIManagerSettings
} from '../types';

export class AIManager {
    private apiClient: InkGatewayClient;
    private chatHistory: ChatHistory;
    private conversationContext: ConversationContext;
    private settings: AIManagerSettings;

    constructor(apiClient: InkGatewayClient, settings: AIManagerSettings) {
        this.apiClient = apiClient;
        this.settings = settings;
        this.chatHistory = {
            messages: [],
            sessionId: this.generateSessionId(),
            startTime: new Date(),
            lastActivity: new Date()
        };
        this.conversationContext = {
            activeFiles: [],
            relevantChunks: [],
            userPreferences: {}
        };
    }

    /**
     * Send a message to AI and get response
     * Requirement 1.2: When user inputs message in chat window, system SHALL send request to Ink-Gateway AI service
     */
    async sendMessage(message: string, includeContext: boolean = true): Promise<AIResponse> {
        try {
            // Add user message to history
            const userMessage: ChatMessage = {
                id: this.generateMessageId(),
                content: message,
                role: 'user',
                timestamp: new Date(),
                metadata: {}
            };
            
            this.addMessageToHistory(userMessage);

            // Prepare context if requested
            let contextChunks: string[] = [];
            if (includeContext) {
                contextChunks = this.conversationContext.relevantChunks.map(chunk => chunk.chunkId);
            }

            // Send to Ink-Gateway
            const response = await this.apiClient.chatWithAI(message, contextChunks);

            // Add AI response to history
            const aiMessage: ChatMessage = {
                id: this.generateMessageId(),
                content: response.message,
                role: 'assistant',
                timestamp: new Date(),
                metadata: {
                    suggestions: response.suggestions,
                    actions: response.actions,
                    responseMetadata: response.metadata
                }
            };

            this.addMessageToHistory(aiMessage);
            this.updateLastActivity();

            return response;
        } catch (error) {
            console.error('Error sending message to AI:', error);
            throw new Error(`Failed to send message: ${(error as Error).message}`);
        }
    }

    /**
     * Process content using AI
     * Requirement 1.4: System SHALL maintain chat history during session
     */
    async processContent(content: string): Promise<ProcessingResult> {
        try {
            const result = await this.apiClient.processContent(content);
            
            // Add processing activity to context
            this.conversationContext.relevantChunks.push(...result.chunks);
            this.updateLastActivity();

            return result;
        } catch (error) {
            console.error('Error processing content:', error);
            throw new Error(`Failed to process content: ${(error as Error).message}`);
        }
    }

    /**
     * Get chat history
     * Requirement 1.4: System SHALL maintain chat history during session
     */
    getChatHistory(): ChatHistory {
        return { ...this.chatHistory };
    }

    /**
     * Clear chat history
     */
    clearChatHistory(): void {
        this.chatHistory = {
            messages: [],
            sessionId: this.generateSessionId(),
            startTime: new Date(),
            lastActivity: new Date()
        };
        this.conversationContext.relevantChunks = [];
    }

    /**
     * Add message to chat history
     */
    private addMessageToHistory(message: ChatMessage): void {
        this.chatHistory.messages.push(message);
        
        // Limit history size if configured
        if (this.settings.maxHistorySize && 
            this.chatHistory.messages.length > this.settings.maxHistorySize) {
            this.chatHistory.messages = this.chatHistory.messages.slice(-this.settings.maxHistorySize);
        }
    }

    /**
     * Update conversation context with relevant files
     */
    updateContext(activeFiles: string[], relevantChunks: any[] = []): void {
        this.conversationContext.activeFiles = activeFiles;
        if (relevantChunks.length > 0) {
            this.conversationContext.relevantChunks = relevantChunks;
        }
        this.updateLastActivity();
    }

    /**
     * Get conversation context
     */
    getContext(): ConversationContext {
        return { ...this.conversationContext };
    }

    /**
     * Update last activity timestamp
     */
    private updateLastActivity(): void {
        this.chatHistory.lastActivity = new Date();
    }

    /**
     * Generate unique session ID
     */
    private generateSessionId(): string {
        return `session_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
    }

    /**
     * Generate unique message ID
     */
    private generateMessageId(): string {
        return `msg_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
    }

    /**
     * Get session statistics
     */
    getSessionStats(): {
        messageCount: number;
        sessionDuration: number;
        lastActivity: Date;
        contextSize: number;
    } {
        const now = new Date();
        const sessionDuration = now.getTime() - this.chatHistory.startTime.getTime();
        
        return {
            messageCount: this.chatHistory.messages.length,
            sessionDuration,
            lastActivity: this.chatHistory.lastActivity,
            contextSize: this.conversationContext.relevantChunks.length
        };
    }

    /**
     * Export chat history for persistence
     */
    exportChatHistory(): string {
        return JSON.stringify({
            history: this.chatHistory,
            context: this.conversationContext,
            exportTime: new Date()
        }, null, 2);
    }

    /**
     * Import chat history from persistence
     */
    importChatHistory(data: string): void {
        try {
            const imported = JSON.parse(data);
            if (imported.history && imported.context) {
                this.chatHistory = imported.history;
                this.conversationContext = imported.context;
            }
        } catch (error) {
            console.error('Error importing chat history:', error);
        }
    }
}