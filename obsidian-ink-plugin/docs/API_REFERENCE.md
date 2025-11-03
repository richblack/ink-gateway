# API 參考文件

## 目錄

1. [概述](#概述)
2. [核心介面](#核心介面)
3. [資料類型](#資料類型)
4. [事件系統](#事件系統)
5. [錯誤處理](#錯誤處理)
6. [擴展開發](#擴展開發)
7. [範例程式碼](#範例程式碼)

## 概述

Obsidian Ink Plugin 提供了一套完整的 API，讓開發者可以：

- 整合自訂功能
- 擴展插件能力
- 建立第三方工具
- 自動化工作流程

### API 版本

- **當前版本**: 1.0.0
- **相容性**: 向後相容
- **更新頻率**: 跟隨插件版本

### 存取方式

```typescript
// 透過插件實例存取 API
const plugin = app.plugins.plugins['obsidian-ink-plugin'];
const api = plugin.api;

// 或透過全域物件
const api = window.InkPluginAPI;
```

## 核心介面

### IContentManager

內容管理器負責處理 Obsidian 內容的解析、同步和管理。

```typescript
interface IContentManager {
    // 內容解析
    parseContent(content: string, filePath: string): Promise<ParsedContent>;
    parseHierarchy(content: string): HierarchyNode[];
    extractMetadata(file: TFile): ContentMetadata;
    
    // 同步管理
    syncToInkGateway(chunks: UnifiedChunk[]): Promise<SyncResult>;
    handleContentChange(file: TFile): Promise<void>;
    syncAllContent(): Promise<void>;
    
    // 文件 ID 管理
    generateDocumentId(filePath: string): string;
    generateVirtualDocumentId(context: VirtualDocumentContext): string;
    getChunksByDocumentId(documentId: string): Promise<UnifiedChunk[]>;
    reconstructDocument(documentId: string): Promise<ReconstructedDocument>;
    
    // 事件
    on(event: ContentEvent, callback: Function): void;
    off(event: ContentEvent, callback: Function): void;
    trigger(event: ContentEvent, ...args: any[]): void;
}
```

#### 方法詳細說明

##### parseContent()

解析 Markdown 內容並建立統一區塊。

```typescript
async parseContent(content: string, filePath: string): Promise<ParsedContent>
```

**參數**:
- `content`: 要解析的 Markdown 內容
- `filePath`: 檔案路徑，用於生成位置資訊

**回傳值**:
```typescript
interface ParsedContent {
    chunks: UnifiedChunk[];
    hierarchy: HierarchyNode[];
    metadata: ContentMetadata;
    positions: PositionMap;
}
```

**範例**:
```typescript
const content = `# 標題\n\n這是內容段落。`;
const result = await contentManager.parseContent(content, 'test.md');
console.log(`解析出 ${result.chunks.length} 個區塊`);
```

##### syncToInkGateway()

將區塊同步到 Ink-Gateway。

```typescript
async syncToInkGateway(chunks: UnifiedChunk[]): Promise<SyncResult>
```

**參數**:
- `chunks`: 要同步的統一區塊陣列

**回傳值**:
```typescript
interface SyncResult {
    success: boolean;
    syncedChunks: number;
    failedChunks: number;
    errors: SyncError[];
    duration: number;
}
```

### ISearchManager

搜尋管理器提供語義搜尋和結果展示功能。

```typescript
interface ISearchManager {
    // 搜尋操作
    performSearch(query: SearchQuery): Promise<SearchResult>;
    searchSemantic(content: string): Promise<SearchResult>;
    searchByTags(tags: string[], logic?: 'AND' | 'OR'): Promise<SearchResult>;
    
    // 結果處理
    displayResults(results: SearchResult): void;
    navigateToResult(result: SearchResultItem): void;
    exportResults(results: SearchResult, format: 'json' | 'csv' | 'md'): string;
    
    // 快取管理
    clearSearchCache(): void;
    getSearchStats(): SearchStats;
    
    // UI 管理
    createSearchView(): SearchView;
    showSearchView(): void;
    hideSearchView(): void;
}
```

#### 方法詳細說明

##### performSearch()

執行搜尋操作。

```typescript
async performSearch(query: SearchQuery): Promise<SearchResult>
```

**參數**:
```typescript
interface SearchQuery {
    content?: string;
    tags?: string[];
    tagLogic?: 'AND' | 'OR';
    filters?: SearchFilters;
    searchType: 'semantic' | 'exact' | 'fuzzy';
    pagination?: {
        page: number;
        pageSize: number;
    };
}
```

**範例**:
```typescript
const query: SearchQuery = {
    content: '機器學習',
    searchType: 'semantic',
    tags: ['AI', 'technology'],
    tagLogic: 'AND'
};

const results = await searchManager.performSearch(query);
console.log(`找到 ${results.totalCount} 個結果`);
```

### IAIManager

AI 管理器處理 AI 聊天功能和智能內容處理。

```typescript
interface IAIManager {
    // 聊天功能
    sendMessage(message: string, context?: string[]): Promise<AIResponse>;
    processContent(content: string): Promise<ProcessingResult>;
    
    // 聊天歷史
    getChatHistory(): ChatMessage[];
    clearChatHistory(): void;
    exportChatHistory(format: 'json' | 'md'): string;
    importChatHistory(data: string): void;
    
    // 設定管理
    updateAISettings(settings: AISettings): void;
    getAISettings(): AISettings;
    
    // UI 管理
    createChatView(): ChatView;
    showChatView(): void;
    hideChatView(): void;
}
```

#### 方法詳細說明

##### sendMessage()

發送訊息給 AI 助手。

```typescript
async sendMessage(message: string, context?: string[]): Promise<AIResponse>
```

**參數**:
- `message`: 要發送的訊息
- `context`: 可選的上下文區塊 ID 陣列

**回傳值**:
```typescript
interface AIResponse {
    message: string;
    suggestions?: ContentSuggestion[];
    actions?: AIAction[];
    metadata: ResponseMetadata;
}
```

**範例**:
```typescript
const response = await aiManager.sendMessage(
    '總結我關於機器學習的筆記',
    ['chunk-id-1', 'chunk-id-2']
);
console.log('AI 回應:', response.message);
```

### ITemplateManager

模板管理器處理模板的建立、應用和管理。

```typescript
interface ITemplateManager {
    // 模板操作
    createTemplate(name: string, structure: TemplateStructure): Promise<Template>;
    updateTemplate(id: string, updates: Partial<Template>): Promise<Template>;
    deleteTemplate(id: string): Promise<void>;
    getTemplate(id: string): Promise<Template>;
    listTemplates(): Promise<Template[]>;
    
    // 模板應用
    applyTemplate(templateId: string, targetFile: TFile): Promise<void>;
    parseTemplateFromContent(content: string): Template;
    validateTemplate(template: Template): ValidationResult;
    
    // 實例管理
    getTemplateInstances(templateId: string): Promise<TemplateInstance[]>;
    updateTemplateInstance(instanceId: string, values: Record<string, any>): Promise<void>;
    
    // 匯入匯出
    exportTemplate(templateId: string): string;
    importTemplate(data: string): Promise<Template>;
}
```

### IInkGatewayClient

API 客戶端負責與 Ink-Gateway 後端服務通訊。

```typescript
interface IInkGatewayClient {
    // 連線管理
    connect(): Promise<void>;
    disconnect(): void;
    isConnected(): boolean;
    healthCheck(): Promise<boolean>;
    
    // 區塊操作
    createChunk(chunk: UnifiedChunk): Promise<UnifiedChunk>;
    updateChunk(id: string, chunk: Partial<UnifiedChunk>): Promise<UnifiedChunk>;
    deleteChunk(id: string): Promise<void>;
    getChunk(id: string): Promise<UnifiedChunk>;
    batchCreateChunks(chunks: UnifiedChunk[]): Promise<UnifiedChunk[]>;
    
    // 搜尋操作
    searchChunks(query: SearchQuery): Promise<SearchResult>;
    searchSemantic(content: string): Promise<SearchResult>;
    searchByTags(tags: string[]): Promise<SearchResult>;
    
    // AI 操作
    chatWithAI(message: string, context?: string[]): Promise<AIResponse>;
    processContent(content: string): Promise<ProcessingResult>;
    
    // 模板操作
    createTemplate(template: Template): Promise<Template>;
    getTemplateInstances(templateId: string): Promise<TemplateInstance[]>;
    
    // 文件 ID 分頁操作
    getChunksByDocumentId(documentId: string, options?: PaginationOptions): Promise<DocumentChunksResult>;
    createVirtualDocument(context: VirtualDocumentContext): Promise<VirtualDocument>;
    updateDocumentScope(chunkId: string, documentId: string, scope: DocumentScope): Promise<void>;
    
    // 階層操作
    getHierarchy(rootId: string): Promise<HierarchyNode[]>;
    updateHierarchy(relations: HierarchyRelation[]): Promise<void>;
    
    // 設定管理
    updateSettings(settings: ClientSettings): void;
    getSettings(): ClientSettings;
}
```

## 資料類型

### UnifiedChunk

統一區塊是系統中的核心資料結構。

```typescript
interface UnifiedChunk {
    // 基本資訊
    chunkId: string;
    contents: string;
    parent?: string;
    page?: string;
    
    // 類型標記
    isPage: boolean;
    isTag: boolean;
    isTemplate: boolean;
    isSlot: boolean;
    
    // 關聯資訊
    ref?: string;
    tags: string[];
    metadata: Record<string, any>;
    
    // 時間戳記
    createdTime: Date;
    lastUpdated: Date;
    
    // Obsidian 特定資訊
    position: Position;
    filePath: string;
    obsidianMetadata: ObsidianMetadata;
    
    // 文件 ID 分頁功能
    documentId: string;
    virtualDocumentId?: string;
    documentScope: DocumentScope;
}
```

### Position

位置資訊用於記錄內容在原始檔案中的位置。

```typescript
interface Position {
    fileName: string;
    lineStart: number;
    lineEnd: number;
    charStart: number;
    charEnd: number;
}
```

### SearchQuery

搜尋查詢參數。

```typescript
interface SearchQuery {
    content?: string;
    tags?: string[];
    tagLogic?: 'AND' | 'OR';
    filters?: SearchFilters;
    searchType: 'semantic' | 'exact' | 'fuzzy';
    pagination?: PaginationOptions;
    sortBy?: 'relevance' | 'date' | 'title';
    sortOrder?: 'asc' | 'desc';
}

interface SearchFilters {
    dateRange?: {
        start: Date;
        end: Date;
    };
    fileTypes?: string[];
    minScore?: number;
    maxResults?: number;
}
```

### SearchResult

搜尋結果。

```typescript
interface SearchResult {
    items: SearchResultItem[];
    totalCount: number;
    searchTime: number;
    cacheHit: boolean;
    pagination?: {
        currentPage: number;
        totalPages: number;
        pageSize: number;
    };
}

interface SearchResultItem {
    chunk: UnifiedChunk;
    score: number;
    context: string;
    position: Position;
    highlights: TextHighlight[];
}
```

### Template

模板定義。

```typescript
interface Template {
    id: string;
    name: string;
    description?: string;
    slots: TemplateSlot[];
    structure: TemplateStructure;
    metadata: TemplateMetadata;
    createdAt: Date;
    updatedAt: Date;
}

interface TemplateSlot {
    id: string;
    name: string;
    type: 'text' | 'number' | 'date' | 'link' | 'tag';
    required: boolean;
    defaultValue?: any;
    validation?: ValidationRule;
    description?: string;
}
```

### AIResponse

AI 回應資料。

```typescript
interface AIResponse {
    message: string;
    suggestions?: ContentSuggestion[];
    actions?: AIAction[];
    metadata: ResponseMetadata;
}

interface ContentSuggestion {
    type: 'link' | 'tag' | 'content' | 'template';
    content: string;
    confidence: number;
    action?: string;
}

interface ResponseMetadata {
    responseTime: number;
    tokensUsed: number;
    model: string;
    confidence: number;
}
```

## 事件系統

### 事件類型

```typescript
type PluginEvent = 
    // 內容事件
    | 'content-changed'
    | 'content-parsed'
    | 'content-synced'
    
    // 同步事件
    | 'sync-start'
    | 'sync-progress'
    | 'sync-complete'
    | 'sync-error'
    
    // 搜尋事件
    | 'search-performed'
    | 'search-results-updated'
    
    // AI 事件
    | 'ai-message-sent'
    | 'ai-response-received'
    | 'ai-error'
    
    // 模板事件
    | 'template-created'
    | 'template-applied'
    | 'template-updated'
    
    // 系統事件
    | 'plugin-loaded'
    | 'plugin-unloaded'
    | 'settings-changed'
    | 'offline-mode-changed';
```

### 事件監聽

```typescript
// 監聽事件
plugin.on('content-changed', (file: TFile, changes: ContentChange[]) => {
    console.log(`檔案 ${file.path} 已變更`);
});

// 一次性監聽
plugin.once('sync-complete', (result: SyncResult) => {
    console.log('同步完成');
});

// 移除監聽器
const handler = (data: any) => console.log(data);
plugin.on('search-performed', handler);
plugin.off('search-performed', handler);

// 觸發事件
plugin.trigger('custom-event', { data: 'example' });
```

### 事件資料

```typescript
interface ContentChangeEvent {
    file: TFile;
    changes: ContentChange[];
    timestamp: Date;
}

interface SyncProgressEvent {
    total: number;
    completed: number;
    current: string;
    errors: SyncError[];
}

interface SearchPerformedEvent {
    query: SearchQuery;
    results: SearchResult;
    duration: number;
}
```

## 錯誤處理

### 錯誤類型

```typescript
class PluginError extends Error {
    constructor(
        message: string,
        public code: string,
        public type: ErrorType,
        public recoverable: boolean = true,
        public details?: any
    ) {
        super(message);
        this.name = 'PluginError';
    }
}

enum ErrorType {
    NETWORK_ERROR = 'network_error',
    API_ERROR = 'api_error',
    PARSING_ERROR = 'parsing_error',
    SYNC_ERROR = 'sync_error',
    VALIDATION_ERROR = 'validation_error',
    TEMPLATE_ERROR = 'template_error',
    AI_ERROR = 'ai_error'
}
```

### 錯誤處理器

```typescript
interface IErrorHandler {
    handleError(error: Error, context?: string): void;
    registerErrorHandler(type: ErrorType, handler: ErrorHandlerFunction): void;
    getErrorHistory(): ErrorRecord[];
    clearErrorHistory(): void;
}

type ErrorHandlerFunction = (error: PluginError) => void;

interface ErrorRecord {
    error: PluginError;
    timestamp: Date;
    context?: string;
    resolved: boolean;
}
```

### 使用範例

```typescript
try {
    await contentManager.syncToInkGateway(chunks);
} catch (error) {
    if (error instanceof PluginError) {
        switch (error.type) {
            case ErrorType.NETWORK_ERROR:
                // 處理網路錯誤
                if (error.recoverable) {
                    // 重試邏輯
                }
                break;
            case ErrorType.API_ERROR:
                // 處理 API 錯誤
                break;
            default:
                // 其他錯誤
        }
    }
}
```

## 擴展開發

### 建立自訂功能

```typescript
// 擴展內容管理器
class CustomContentManager extends ContentManager {
    async customParseMethod(content: string): Promise<CustomResult> {
        // 自訂解析邏輯
        const baseResult = await super.parseContent(content, 'custom.md');
        
        // 添加自訂處理
        return {
            ...baseResult,
            customData: this.processCustomData(content)
        };
    }
}

// 註冊自訂管理器
plugin.registerContentManager(new CustomContentManager(plugin));
```

### 建立自訂搜尋提供者

```typescript
interface ISearchProvider {
    name: string;
    search(query: SearchQuery): Promise<SearchResult>;
    supports(searchType: string): boolean;
}

class CustomSearchProvider implements ISearchProvider {
    name = 'custom-search';
    
    async search(query: SearchQuery): Promise<SearchResult> {
        // 自訂搜尋邏輯
        return {
            items: [],
            totalCount: 0,
            searchTime: 0,
            cacheHit: false
        };
    }
    
    supports(searchType: string): boolean {
        return searchType === 'custom';
    }
}

// 註冊搜尋提供者
plugin.searchManager.registerProvider(new CustomSearchProvider());
```

### 建立自訂 AI 處理器

```typescript
interface IAIProcessor {
    name: string;
    process(message: string, context: string[]): Promise<AIResponse>;
    supports(messageType: string): boolean;
}

class CustomAIProcessor implements IAIProcessor {
    name = 'custom-ai';
    
    async process(message: string, context: string[]): Promise<AIResponse> {
        // 自訂 AI 處理邏輯
        return {
            message: `自訂回應: ${message}`,
            suggestions: [],
            actions: [],
            metadata: {
                responseTime: 100,
                tokensUsed: 50,
                model: 'custom-model',
                confidence: 0.9
            }
        };
    }
    
    supports(messageType: string): boolean {
        return messageType.startsWith('custom:');
    }
}

// 註冊 AI 處理器
plugin.aiManager.registerProcessor(new CustomAIProcessor());
```

## 範例程式碼

### 基本使用範例

```typescript
// 獲取插件 API
const plugin = app.plugins.plugins['obsidian-ink-plugin'];
const api = plugin.api;

// 解析內容
const content = `# 我的筆記\n\n這是一個範例。`;
const parsed = await api.contentManager.parseContent(content, 'example.md');
console.log(`解析出 ${parsed.chunks.length} 個區塊`);

// 執行搜尋
const searchResult = await api.searchManager.performSearch({
    content: '範例',
    searchType: 'semantic'
});
console.log(`找到 ${searchResult.totalCount} 個結果`);

// AI 對話
const aiResponse = await api.aiManager.sendMessage('總結這個筆記');
console.log('AI 回應:', aiResponse.message);
```

### 進階整合範例

```typescript
// 建立自訂工作流程
class CustomWorkflow {
    constructor(private plugin: ObsidianInkPlugin) {}
    
    async processNewNote(file: TFile): Promise<void> {
        // 1. 解析內容
        const content = await this.plugin.app.vault.read(file);
        const parsed = await this.plugin.contentManager.parseContent(content, file.path);
        
        // 2. 自動標籤
        const tags = await this.generateTags(parsed.chunks);
        
        // 3. 套用模板（如果適用）
        const template = await this.findSuitableTemplate(parsed);
        if (template) {
            await this.plugin.templateManager.applyTemplate(template.id, file);
        }
        
        // 4. 同步到 Gateway
        await this.plugin.contentManager.syncToInkGateway(parsed.chunks);
        
        // 5. 觸發自訂事件
        this.plugin.trigger('custom-note-processed', { file, parsed, tags });
    }
    
    private async generateTags(chunks: UnifiedChunk[]): Promise<string[]> {
        // 使用 AI 生成標籤
        const content = chunks.map(c => c.contents).join('\n');
        const response = await this.plugin.aiManager.sendMessage(
            `為以下內容生成適當的標籤：\n${content}`
        );
        
        // 解析 AI 回應中的標籤
        return this.extractTagsFromResponse(response.message);
    }
    
    private async findSuitableTemplate(parsed: ParsedContent): Promise<Template | null> {
        const templates = await this.plugin.templateManager.listTemplates();
        
        // 根據內容結構找到最適合的模板
        for (const template of templates) {
            if (this.matchesTemplate(parsed, template)) {
                return template;
            }
        }
        
        return null;
    }
}

// 使用自訂工作流程
const workflow = new CustomWorkflow(plugin);

// 監聽檔案建立事件
plugin.app.vault.on('create', async (file) => {
    if (file instanceof TFile && file.extension === 'md') {
        await workflow.processNewNote(file);
    }
});
```

### 批次處理範例

```typescript
// 批次處理多個檔案
async function batchProcessFiles(files: TFile[]): Promise<void> {
    const batchSize = 10;
    const results: ProcessResult[] = [];
    
    for (let i = 0; i < files.length; i += batchSize) {
        const batch = files.slice(i, i + batchSize);
        const batchPromises = batch.map(async (file) => {
            try {
                const content = await app.vault.read(file);
                const parsed = await api.contentManager.parseContent(content, file.path);
                await api.contentManager.syncToInkGateway(parsed.chunks);
                
                return { file: file.path, success: true };
            } catch (error) {
                return { file: file.path, success: false, error: error.message };
            }
        });
        
        const batchResults = await Promise.all(batchPromises);
        results.push(...batchResults);
        
        // 顯示進度
        console.log(`已處理 ${Math.min(i + batchSize, files.length)}/${files.length} 個檔案`);
    }
    
    // 顯示結果摘要
    const successful = results.filter(r => r.success).length;
    const failed = results.filter(r => !r.success).length;
    console.log(`批次處理完成: ${successful} 成功, ${failed} 失敗`);
}
```

### 自訂 UI 元件範例

```typescript
// 建立自訂搜尋面板
class CustomSearchPanel extends ItemView {
    constructor(leaf: WorkspaceLeaf, private plugin: ObsidianInkPlugin) {
        super(leaf);
    }
    
    getViewType(): string {
        return 'custom-search-panel';
    }
    
    getDisplayText(): string {
        return '自訂搜尋';
    }
    
    async onOpen(): Promise<void> {
        const container = this.containerEl.children[1];
        container.empty();
        
        // 建立搜尋介面
        const searchInput = container.createEl('input', {
            type: 'text',
            placeholder: '輸入搜尋關鍵字...'
        });
        
        const searchButton = container.createEl('button', {
            text: '搜尋'
        });
        
        const resultsContainer = container.createEl('div', {
            cls: 'search-results'
        });
        
        // 綁定搜尋事件
        searchButton.onclick = async () => {
            const query = searchInput.value;
            if (!query) return;
            
            const results = await this.plugin.searchManager.performSearch({
                content: query,
                searchType: 'semantic'
            });
            
            this.displayResults(results, resultsContainer);
        };
    }
    
    private displayResults(results: SearchResult, container: HTMLElement): void {
        container.empty();
        
        results.items.forEach(item => {
            const resultEl = container.createEl('div', {
                cls: 'search-result-item'
            });
            
            resultEl.createEl('h4', {
                text: item.chunk.contents.substring(0, 50) + '...'
            });
            
            resultEl.createEl('p', {
                text: `相關性: ${(item.score * 100).toFixed(1)}%`
            });
            
            resultEl.onclick = () => {
                this.plugin.searchManager.navigateToResult(item);
            };
        });
    }
}

// 註冊自訂視圖
plugin.registerView(
    'custom-search-panel',
    (leaf) => new CustomSearchPanel(leaf, plugin)
);
```

---

**版本**: 1.0.0  
**最後更新**: 2024年1月  
**文件語言**: 繁體中文