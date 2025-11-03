/**
 * HTTP client for Ink-Gateway API integration
 * Provides comprehensive API communication with error handling, retry logic, and authentication
 */

import { 
  UnifiedChunk, 
  SearchQuery, 
  SearchResult, 
  Template, 
  TemplateInstance, 
  AIResponse, 
  ProcessingResult, 
  HierarchyNode,
  ErrorType,
  PluginError,
  VirtualDocumentContext,
  VirtualDocument,
  DocumentScope,
  PaginationOptions,
  DocumentChunksResult
} from '../types';
import { IInkGatewayClient, HierarchyRelation } from '../interfaces';

// HTTP request configuration
export interface RequestConfig {
  method: 'GET' | 'POST' | 'PUT' | 'DELETE';
  endpoint: string;
  data?: any;
  headers?: Record<string, string>;
  timeout?: number;
  retries?: number;
}

// HTTP response wrapper
export interface APIResponse<T = any> {
  data: T;
  status: number;
  statusText: string;
  headers: Record<string, string>;
}

// Retry configuration
export interface RetryConfig {
  maxRetries: number;
  baseDelay: number;
  maxDelay: number;
  backoffFactor: number;
  retryableStatusCodes: number[];
}

// Request/Response interceptors
export type RequestInterceptor = (config: RequestConfig) => RequestConfig | Promise<RequestConfig>;
export type ResponseInterceptor = (response: APIResponse) => APIResponse | Promise<APIResponse>;
export type ErrorInterceptor = (error: PluginError) => PluginError | Promise<PluginError>;

export class InkGatewayClient implements IInkGatewayClient {
  private baseUrl: string;
  private apiKey: string;
  private timeout: number;
  private retryConfig: RetryConfig;
  
  // Interceptors
  private requestInterceptors: RequestInterceptor[] = [];
  private responseInterceptors: ResponseInterceptor[] = [];
  private errorInterceptors: ErrorInterceptor[] = [];
  
  // Request tracking for debugging
  private requestHistory: Array<{
    config: RequestConfig;
    timestamp: Date;
    duration?: number;
    success: boolean;
    error?: string;
  }> = [];

  constructor(baseUrl: string, apiKey: string, options: Partial<{
    timeout: number;
    retryConfig: Partial<RetryConfig>;
  }> = {}) {
    this.baseUrl = baseUrl.replace(/\/$/, ''); // Remove trailing slash
    this.apiKey = apiKey;
    this.timeout = options.timeout || 30000; // 30 seconds default
    
    // Default retry configuration
    this.retryConfig = {
      maxRetries: 3,
      baseDelay: 1000,
      maxDelay: 10000,
      backoffFactor: 2,
      retryableStatusCodes: [408, 429, 500, 502, 503, 504],
      ...options.retryConfig
    };

    // Add default request interceptor for authentication
    this.addRequestInterceptor((config) => {
      config.headers = {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${this.apiKey}`,
        'User-Agent': 'Obsidian-Ink-Plugin/1.0.0',
        ...config.headers
      };
      return config;
    });

    // Add default error interceptor for common error handling
    this.addErrorInterceptor((error) => {
      console.error(`[InkGatewayClient] ${error.type}: ${error.message}`, error.details);
      return error;
    });
  }

  // Interceptor management
  addRequestInterceptor(interceptor: RequestInterceptor): void {
    this.requestInterceptors.push(interceptor);
  }

  addResponseInterceptor(interceptor: ResponseInterceptor): void {
    this.responseInterceptors.push(interceptor);
  }

  addErrorInterceptor(interceptor: ErrorInterceptor): void {
    this.errorInterceptors.push(interceptor);
  }

  // Core HTTP request method with retry logic
  public async request<T = any>(config: RequestConfig): Promise<APIResponse<T>> {
    const startTime = Date.now();
    let lastError: PluginError | null = null;

    // Apply request interceptors
    let processedConfig = config;
    for (const interceptor of this.requestInterceptors) {
      processedConfig = await interceptor(processedConfig);
    }

    // Add default timeout if not specified
    if (!processedConfig.timeout) {
      processedConfig.timeout = this.timeout;
    }

    // Retry logic with exponential backoff
    for (let attempt = 0; attempt <= this.retryConfig.maxRetries; attempt++) {
      try {
        const response = await this.executeRequest<T>(processedConfig);
        
        // Apply response interceptors
        let processedResponse = response;
        for (const interceptor of this.responseInterceptors) {
          processedResponse = await interceptor(processedResponse);
        }

        // Log successful request
        this.logRequest(processedConfig, Date.now() - startTime, true);
        
        return processedResponse;

      } catch (error) {
        lastError = error instanceof PluginError ? error : new PluginError(
          ErrorType.NETWORK_ERROR,
          'REQUEST_FAILED',
          error,
          true
        );

        // Apply error interceptors
        for (const interceptor of this.errorInterceptors) {
          lastError = await interceptor(lastError);
        }

        // Check if we should retry
        if (attempt < this.retryConfig.maxRetries && this.shouldRetry(lastError)) {
          const delay = this.calculateDelay(attempt);
          console.warn(`[InkGatewayClient] Request failed, retrying in ${delay}ms (attempt ${attempt + 1}/${this.retryConfig.maxRetries})`);
          await this.sleep(delay);
          continue;
        }

        // Log failed request
        this.logRequest(processedConfig, Date.now() - startTime, false, lastError.message);
        break;
      }
    }

    throw lastError;
  }

  // Execute the actual HTTP request
  private async executeRequest<T>(config: RequestConfig): Promise<APIResponse<T>> {
    const url = `${this.baseUrl}${config.endpoint}`;
    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), config.timeout);

    try {
      const response = await fetch(url, {
        method: config.method,
        headers: config.headers,
        body: config.data ? JSON.stringify(config.data) : undefined,
        signal: controller.signal
      });

      clearTimeout(timeoutId);

      // Check for HTTP errors
      if (!response.ok) {
        throw new PluginError(
          ErrorType.API_ERROR,
          `HTTP_${response.status}`,
          {
            status: response.status,
            statusText: response.statusText,
            url: url
          },
          this.retryConfig.retryableStatusCodes.includes(response.status)
        );
      }

      // Parse response
      let data: T;
      const contentType = response.headers.get('content-type');
      if (contentType && contentType.includes('application/json')) {
        data = await response.json();
      } else {
        data = await response.text() as any;
      }

      // Convert headers to plain object
      const headers: Record<string, string> = {};
      response.headers.forEach((value, key) => {
        headers[key] = value;
      });

      return {
        data,
        status: response.status,
        statusText: response.statusText,
        headers
      };

    } catch (error) {
      clearTimeout(timeoutId);
      
      if (error instanceof PluginError) {
        throw error;
      }

      // Handle fetch-specific errors
      if (error instanceof Error) {
        if (error.name === 'AbortError') {
          throw new PluginError(
            ErrorType.NETWORK_ERROR,
            'REQUEST_TIMEOUT',
            { timeout: config.timeout, url },
            true
          );
        }
        
        throw new PluginError(
          ErrorType.NETWORK_ERROR,
          'NETWORK_ERROR',
          { message: error.message, url },
          true
        );
      }

      throw new PluginError(
        ErrorType.NETWORK_ERROR,
        'UNKNOWN_ERROR',
        { error, url },
        false
      );
    }
  }

  // Retry logic helpers
  private shouldRetry(error: PluginError): boolean {
    if (!error.recoverable) return false;
    
    if (error.type === ErrorType.API_ERROR && error.details?.status) {
      return this.retryConfig.retryableStatusCodes.includes(error.details.status);
    }
    
    return error.type === ErrorType.NETWORK_ERROR;
  }

  private calculateDelay(attempt: number): number {
    const delay = this.retryConfig.baseDelay * Math.pow(this.retryConfig.backoffFactor, attempt);
    return Math.min(delay, this.retryConfig.maxDelay);
  }

  private sleep(ms: number): Promise<void> {
    return new Promise(resolve => setTimeout(resolve, ms));
  }

  // Request logging
  private logRequest(config: RequestConfig, duration: number, success: boolean, error?: string): void {
    this.requestHistory.push({
      config: { ...config },
      timestamp: new Date(),
      duration,
      success,
      error
    });

    // Keep only last 100 requests to prevent memory leaks
    if (this.requestHistory.length > 100) {
      this.requestHistory.shift();
    }
  }

  // Public method to get request history for debugging
  getRequestHistory(): typeof this.requestHistory {
    return [...this.requestHistory];
  }

  // Clear request history
  clearRequestHistory(): void {
    this.requestHistory = [];
  }

  // Health check method
  async healthCheck(): Promise<boolean> {
    const response = await this.request({
      method: 'GET',
      endpoint: '/api/v1/health'
    });
    return response.status === 200;
  }

  // Chunk operations
  async createChunk(chunk: UnifiedChunk): Promise<UnifiedChunk> {
    const response = await this.request<UnifiedChunk>({
      method: 'POST',
      endpoint: '/api/v1/chunks',
      data: chunk
    });
    return response.data;
  }

  async updateChunk(id: string, chunk: Partial<UnifiedChunk>): Promise<UnifiedChunk> {
    const response = await this.request<UnifiedChunk>({
      method: 'PUT',
      endpoint: `/api/v1/chunks/${id}`,
      data: chunk
    });
    return response.data;
  }

  async deleteChunk(id: string): Promise<void> {
    await this.request({
      method: 'DELETE',
      endpoint: `/api/chunks/${id}`
    });
  }

  async getChunk(id: string): Promise<UnifiedChunk> {
    const response = await this.request<UnifiedChunk>({
      method: 'GET',
      endpoint: `/api/chunks/${id}`
    });
    return response.data;
  }

  async batchCreateChunks(chunks: UnifiedChunk[]): Promise<UnifiedChunk[]> {
    const response = await this.request<UnifiedChunk[]>({
      method: 'POST',
      endpoint: '/api/v1/chunks/batch',
      data: { chunks }
    });
    return response.data;
  }

  // Search operations
  async searchChunks(query: SearchQuery): Promise<SearchResult> {
    const response = await this.request<SearchResult>({
      method: 'POST',
      endpoint: '/api/search',
      data: query
    });
    return response.data;
  }

  async searchSemantic(content: string): Promise<SearchResult> {
    const response = await this.request<SearchResult>({
      method: 'POST',
      endpoint: '/api/v1/search/multimodal',
      data: { content }
    });
    return response.data;
  }

  async searchByTags(tags: string[]): Promise<SearchResult> {
    const response = await this.request<SearchResult>({
      method: 'POST',
      endpoint: '/api/v1/tags/search',
      data: { tags }
    });
    return response.data;
  }

  // Hierarchy operations
  async getHierarchy(rootId: string): Promise<HierarchyNode[]> {
    const response = await this.request<HierarchyNode[]>({
      method: 'GET',
      endpoint: `/api/hierarchy/${rootId}`
    });
    return response.data;
  }

  async updateHierarchy(relations: HierarchyRelation[]): Promise<void> {
    await this.request({
      method: 'POST',
      endpoint: '/api/hierarchy',
      data: { relations }
    });
  }

  // AI operations
  async chatWithAI(message: string, context?: string[]): Promise<AIResponse> {
    // TODO: Implement AI chat endpoint in Ink-Gateway
    // For now, return a mock response
    return {
      message: `Echo: ${message}`,
      suggestions: [],
      actions: [],
      metadata: {
        model: 'mock',
        processingTime: 100,
        confidence: 0.9
      }
    };
  }

  async processContent(content: string): Promise<ProcessingResult> {
    const response = await this.request<ProcessingResult>({
      method: 'POST',
      endpoint: '/api/ai/process',
      data: { content }
    });
    return response.data;
  }

  // Template operations
  async createTemplate(template: Template): Promise<Template> {
    const response = await this.request<Template>({
      method: 'POST',
      endpoint: '/api/templates',
      data: template
    });
    return response.data;
  }

  async getTemplateInstances(templateId: string): Promise<TemplateInstance[]> {
    const response = await this.request<TemplateInstance[]>({
      method: 'GET',
      endpoint: `/api/templates/${templateId}/instances`
    });
    return response.data;
  }

  // Document ID pagination operations
  async getChunksByDocumentId(documentId: string, options?: PaginationOptions): Promise<DocumentChunksResult> {
    // Validate documentId parameter
    if (!documentId || typeof documentId !== 'string') {
      throw new PluginError(
        ErrorType.VALIDATION_ERROR,
        'INVALID_DOCUMENT_ID',
        { documentId },
        false
      );
    }

    // Validate pagination options
    if (options) {
      this.validatePaginationOptions(options);
    }

    // Build query parameters
    const queryParams = new URLSearchParams();
    if (options?.page !== undefined) {
      queryParams.append('page', options.page.toString());
    }
    if (options?.pageSize !== undefined) {
      queryParams.append('pageSize', options.pageSize.toString());
    }
    if (options?.includeHierarchy !== undefined) {
      queryParams.append('includeHierarchy', options.includeHierarchy.toString());
    }
    if (options?.sortBy) {
      queryParams.append('sortBy', options.sortBy);
    }
    if (options?.sortOrder) {
      queryParams.append('sortOrder', options.sortOrder);
    }

    const queryString = queryParams.toString();
    const endpoint = `/api/documents/${encodeURIComponent(documentId)}/chunks${queryString ? `?${queryString}` : ''}`;

    const response = await this.request<DocumentChunksResult>({
      method: 'GET',
      endpoint
    });

    return response.data;
  }

  async createVirtualDocument(context: VirtualDocumentContext): Promise<VirtualDocument> {
    // Validate virtual document context
    if (!context || typeof context !== 'object') {
      throw new PluginError(
        ErrorType.VALIDATION_ERROR,
        'INVALID_VIRTUAL_DOCUMENT_CONTEXT',
        { context },
        false
      );
    }

    // Validate required fields
    if (!context.sourceType || !context.contextId) {
      throw new PluginError(
        ErrorType.VALIDATION_ERROR,
        'MISSING_REQUIRED_CONTEXT_FIELDS',
        { 
          sourceType: context.sourceType, 
          contextId: context.contextId 
        },
        false
      );
    }

    // Validate sourceType
    const validSourceTypes = ['remnote', 'logseq', 'obsidian-template'];
    if (!validSourceTypes.includes(context.sourceType)) {
      throw new PluginError(
        ErrorType.VALIDATION_ERROR,
        'INVALID_SOURCE_TYPE',
        { 
          sourceType: context.sourceType, 
          validTypes: validSourceTypes 
        },
        false
      );
    }

    const response = await this.request<VirtualDocument>({
      method: 'POST',
      endpoint: '/api/documents/virtual',
      data: context
    });

    return response.data;
  }

  async updateDocumentScope(chunkId: string, documentId: string, scope: DocumentScope): Promise<void> {
    // Validate parameters
    if (!chunkId || typeof chunkId !== 'string') {
      throw new PluginError(
        ErrorType.VALIDATION_ERROR,
        'INVALID_CHUNK_ID',
        { chunkId },
        false
      );
    }

    if (!documentId || typeof documentId !== 'string') {
      throw new PluginError(
        ErrorType.VALIDATION_ERROR,
        'INVALID_DOCUMENT_ID',
        { documentId },
        false
      );
    }

    // Validate document scope
    const validScopes: DocumentScope[] = ['file', 'virtual', 'page'];
    if (!validScopes.includes(scope)) {
      throw new PluginError(
        ErrorType.VALIDATION_ERROR,
        'INVALID_DOCUMENT_SCOPE',
        { 
          scope, 
          validScopes 
        },
        false
      );
    }

    await this.request({
      method: 'PUT',
      endpoint: `/api/chunks/${encodeURIComponent(chunkId)}/document-scope`,
      data: {
        documentId,
        scope
      }
    });
  }

  // Private validation methods for document ID pagination
  private validatePaginationOptions(options: PaginationOptions): void {
    if (options.page !== undefined) {
      if (!Number.isInteger(options.page) || options.page < 1) {
        throw new PluginError(
          ErrorType.VALIDATION_ERROR,
          'INVALID_PAGE_NUMBER',
          { page: options.page },
          false
        );
      }
    }

    if (options.pageSize !== undefined) {
      if (!Number.isInteger(options.pageSize) || options.pageSize < 1 || options.pageSize > 1000) {
        throw new PluginError(
          ErrorType.VALIDATION_ERROR,
          'INVALID_PAGE_SIZE',
          { pageSize: options.pageSize, maxSize: 1000 },
          false
        );
      }
    }

    if (options.sortBy !== undefined) {
      const validSortFields = ['position', 'created', 'updated'];
      if (!validSortFields.includes(options.sortBy)) {
        throw new PluginError(
          ErrorType.VALIDATION_ERROR,
          'INVALID_SORT_FIELD',
          { sortBy: options.sortBy, validFields: validSortFields },
          false
        );
      }
    }

    if (options.sortOrder !== undefined) {
      const validSortOrders = ['asc', 'desc'];
      if (!validSortOrders.includes(options.sortOrder)) {
        throw new PluginError(
          ErrorType.VALIDATION_ERROR,
          'INVALID_SORT_ORDER',
          { sortOrder: options.sortOrder, validOrders: validSortOrders },
          false
        );
      }
    }
  }

  // Configuration methods
  updateBaseUrl(baseUrl: string): void {
    this.baseUrl = baseUrl.replace(/\/$/, '');
  }

  updateApiKey(apiKey: string): void {
    this.apiKey = apiKey;
  }

  updateTimeout(timeout: number): void {
    this.timeout = timeout;
  }

  updateRetryConfig(config: Partial<RetryConfig>): void {
    this.retryConfig = { ...this.retryConfig, ...config };
  }

  // Utility methods for testing and debugging
  getConfiguration(): {
    baseUrl: string;
    timeout: number;
    retryConfig: RetryConfig;
  } {
    return {
      baseUrl: this.baseUrl,
      timeout: this.timeout,
      retryConfig: { ...this.retryConfig }
    };
  }
}