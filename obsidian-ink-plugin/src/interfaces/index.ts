/**
 * Core interfaces for the Obsidian Ink Plugin
 */

import { TFile } from 'obsidian';
import {
  UnifiedChunk,
  ParsedContent,
  HierarchyNode,
  ContentMetadata,
  SyncResult,
  SearchQuery,
  SearchResult,
  SearchResultItem,
  Template,
  TemplateInstance,
  TemplateStructure,
  AIResponse,
  ProcessingResult,
  OfflineOperation,
  SyncConflict,
  MemoryStats,
  UserAction,
  VirtualDocumentContext,
  VirtualDocument,
  DocumentScope,
  PaginationOptions,
  DocumentChunksResult,
  ReconstructedDocument
} from '../types';

// Content Manager Interface
export interface IContentManager {
  parseContent(content: string, filePath: string): Promise<ParsedContent>;
  syncToInkGateway(chunks: UnifiedChunk[]): Promise<SyncResult>;
  handleContentChange(file: TFile): Promise<void>;
  parseHierarchy(content: string): HierarchyNode[];
  extractMetadata(file: TFile): ContentMetadata;
  
  // Document ID management methods
  generateDocumentId(filePath: string): string;
  generateVirtualDocumentId(context: VirtualDocumentContext): string;
  getChunksByDocumentId(documentId: string, options?: PaginationOptions): Promise<DocumentChunksResult>;
  createVirtualDocument(context: VirtualDocumentContext): Promise<VirtualDocument>;
  reconstructDocument(documentId: string): Promise<ReconstructedDocument>;
  updateDocumentScope(chunkId: string, documentId: string, scope: DocumentScope): Promise<void>;
  getDocumentIdFromFile(file: TFile): string;
  isVirtualDocumentId(documentId: string): boolean;
  extractFilePathFromDocumentId(documentId: string): string | null;
}

// Search Manager Interface
export interface ISearchManager {
  performSearch(query: SearchQuery): Promise<SearchResult>;
  displayResults(results: SearchResult): void;
  navigateToResult(result: SearchResultItem): void;
  createSearchView(): void;
}

// Template Manager Interface
export interface ITemplateManager {
  createTemplate(name: string, structure: TemplateStructure): Promise<Template>;
  applyTemplate(templateId: string, targetFile: TFile): Promise<void>;
  parseTemplateFromContent(content: string): Template;
  getTemplateInstances(templateId: string): Promise<TemplateInstance[]>;
}

// AI Manager Interface
export interface IAIManager {
  sendMessage(message: string): Promise<AIResponse>;
  processContent(content: string): Promise<ProcessingResult>;
  createChatView(): void;
  maintainChatHistory(): void;
}

// API Client Interface
export interface IInkGatewayClient {
  // Chunk operations
  createChunk(chunk: UnifiedChunk): Promise<UnifiedChunk>;
  updateChunk(id: string, chunk: Partial<UnifiedChunk>): Promise<UnifiedChunk>;
  deleteChunk(id: string): Promise<void>;
  getChunk(id: string): Promise<UnifiedChunk>;
  batchCreateChunks(chunks: UnifiedChunk[]): Promise<UnifiedChunk[]>;
  
  // Search operations
  searchChunks(query: SearchQuery): Promise<SearchResult>;
  searchSemantic(content: string): Promise<SearchResult>;
  searchByTags(tags: string[]): Promise<SearchResult>;
  
  // Hierarchy operations
  getHierarchy(rootId: string): Promise<HierarchyNode[]>;
  updateHierarchy(relations: HierarchyRelation[]): Promise<void>;
  
  // AI operations
  chatWithAI(message: string, context?: string[]): Promise<AIResponse>;
  processContent(content: string): Promise<ProcessingResult>;
  
  // Template operations
  createTemplate(template: Template): Promise<Template>;
  getTemplateInstances(templateId: string): Promise<TemplateInstance[]>;
  
  // Document ID pagination operations
  getChunksByDocumentId(documentId: string, options?: PaginationOptions): Promise<DocumentChunksResult>;
  createVirtualDocument(context: VirtualDocumentContext): Promise<VirtualDocument>;
  updateDocumentScope(chunkId: string, documentId: string, scope: DocumentScope): Promise<void>;
  
  // Health check
  healthCheck(): Promise<boolean>;
  
  // Generic request method for custom API calls
  request<T = any>(config: {
    method: 'GET' | 'POST' | 'PUT' | 'DELETE';
    endpoint: string;
    data?: any;
    headers?: Record<string, string>;
    timeout?: number;
    retries?: number;
  }): Promise<{ data: T; status: number; statusText: string; headers: Record<string, string> }>;
}

// Hierarchy relation for API
export interface HierarchyRelation {
  parentId: string;
  childId: string;
  relationType: 'heading' | 'bullet' | 'template';
}

// Offline Manager Interface
export interface IOfflineManager {
  isOnline(): boolean;
  queueOperation(operation: OfflineOperation): void;
  syncWhenOnline(): Promise<void>;
  handleConflicts(conflicts: SyncConflict[]): Promise<void>;
}

// Memory Manager Interface
export interface IMemoryManager {
  cleanupCache(): void;
  monitorMemoryUsage(): MemoryStats;
  optimizePerformance(): void;
}

// Cache Manager Interface
export interface ICacheManager {
  get<T>(key: string): T | null;
  set<T>(key: string, value: T, ttl?: number): void;
  delete(key: string): void;
  clear(): void;
  size(): number;
}

// Event Manager Interface
export interface IEventManager {
  on(event: string, callback: Function): void;
  off(event: string, callback: Function): void;
  emit(event: string, ...args: any[]): void;
}

// Logger Interface
export interface ILogger {
  debug(message: string, ...args: any[]): void;
  info(message: string, ...args: any[]): void;
  warn(message: string, ...args: any[]): void;
  error(message: string, error?: Error, ...args: any[]): void;
}

// Test Utilities Interface
export interface ITestUtils {
  createMockVault(): MockVault;
  createMockFile(content: string): MockFile;
  mockInkGatewayAPI(): MockAPIClient;
  simulateUserInteraction(action: UserAction): Promise<void>;
}

// Mock interfaces for testing
export interface MockVault {
  files: MockFile[];
  getFiles(): MockFile[];
  getFileByPath(path: string): MockFile | null;
  create(path: string, content: string): MockFile;
  modify(file: MockFile, content: string): void;
  delete(file: MockFile): void;
}

export interface MockFile {
  path: string;
  name: string;
  content: string;
  stat: {
    ctime: number;
    mtime: number;
    size: number;
  };
}

export interface MockAPIClient extends IInkGatewayClient {
  setMockResponse(endpoint: string, response: any): void;
  setMockError(endpoint: string, error: Error): void;
  getRequestHistory(): APIRequest[];
  clearHistory(): void;
}

export interface APIRequest {
  method: string;
  endpoint: string;
  data?: any;
  timestamp: Date;
}

// View interfaces for UI components
export interface ISearchView {
  show(): void;
  hide(): void;
  updateResults(results: SearchResult): void;
  onResultClick(callback: (result: SearchResultItem) => void): void;
}

export interface IChatView {
  show(): void;
  hide(): void;
  addMessage(message: string, isUser: boolean): void;
  onMessageSend(callback: (message: string) => void): void;
  showTyping(): void;
  hideTyping(): void;
}

export interface ISettingsView {
  show(): void;
  hide(): void;
  getSettings(): any;
  updateSettings(settings: any): void;
  onSettingsChange(callback: (settings: any) => void): void;
}