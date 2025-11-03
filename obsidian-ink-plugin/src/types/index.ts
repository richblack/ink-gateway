/**
 * Core type definitions for the Obsidian Ink Plugin
 */

// Re-export media types
export * from './media';

// Plugin Settings Interface
export interface PluginSettings {
  inkGatewayUrl: string;
  apiKey: string;
  autoSync: boolean;
  syncInterval: number;
  cacheEnabled: boolean;
  debugMode: boolean;
  
  // Storage settings
  storageProvider: 'google_drive' | 'local' | 'both';
  googleDriveFolderId: string;
  localStoragePath: string;
}

// Note: DEFAULT_SETTINGS is defined in ./settings/PluginSettings.ts

// Position tracking for content location
export interface Position {
  fileName: string;
  lineStart: number;
  lineEnd: number;
  charStart: number;
  charEnd: number;
}

// Obsidian-specific metadata
export interface ObsidianMetadata {
  properties: Record<string, any>;
  frontmatter: Record<string, any>;
  aliases: string[];
  cssClasses: string[];
}

// Unified Chunk interface matching the Ink-Gateway system
export interface UnifiedChunk {
  chunkId: string;
  contents: string;
  parent?: string;
  page?: string;
  isPage: boolean;
  isTag: boolean;
  isTemplate: boolean;
  isSlot: boolean;
  ref?: string;
  tags: string[];
  metadata: Record<string, any>;
  createdTime: Date;
  lastUpdated: Date;
  
  // Obsidian-specific fields
  position: Position;
  filePath: string;
  obsidianMetadata: ObsidianMetadata;
  
  // Document ID pagination fields
  documentId: string;
  virtualDocumentId?: string;
  documentScope: DocumentScope;
}

// Hierarchy node for content structure
export interface HierarchyNode {
  id: string;
  content: string;
  level: number;
  type: 'heading' | 'bullet';
  parent?: string;
  children: string[];
  position: Position;
}

// Content metadata
export interface ContentMetadata {
  title?: string;
  tags: string[];
  properties: Record<string, any>;
  frontmatter: Record<string, any>;
  aliases: string[];
  cssClasses: string[];
  createdTime: Date;
  modifiedTime: Date;
}

// Parsed content structure
export interface ParsedContent {
  chunks: UnifiedChunk[];
  hierarchy: HierarchyNode[];
  metadata: ContentMetadata;
  positions: PositionMap;
}

// Position mapping for content tracking
export type PositionMap = Map<string, Position>;

// Sync-related types
export interface SyncState {
  lastSyncTime: Date;
  pendingChanges: PendingChange[];
  conflictResolution: ConflictResolution;
  syncStatus: 'idle' | 'syncing' | 'error' | 'offline';
}

export interface PendingChange {
  id: string;
  type: 'create' | 'update' | 'delete';
  chunk: UnifiedChunk;
  timestamp: Date;
  retryCount: number;
}

export interface ConflictResolution {
  strategy: 'local' | 'remote' | 'merge' | 'manual';
  conflicts: SyncConflict[];
}

export interface SyncConflict {
  chunkId: string;
  localVersion: UnifiedChunk;
  remoteVersion: UnifiedChunk;
  conflictType: 'content' | 'metadata' | 'hierarchy';
}

// Sync result
export interface SyncResult {
  success: boolean;
  syncedChunks: number;
  errors: SyncError[];
  conflicts: SyncConflict[];
  duration: number;
}

export interface SyncError {
  chunkId: string;
  error: string;
  recoverable: boolean;
}

// Search-related types
export interface SearchQuery {
  content?: string;
  tags?: string[];
  tagLogic?: 'AND' | 'OR';
  filters?: SearchFilters;
  searchType?: 'semantic' | 'exact' | 'fuzzy';
}

export interface SearchFilters {
  dateRange?: {
    start: Date;
    end: Date;
  };
  fileTypes?: string[];
  excludeTags?: string[];
  minScore?: number;
}

export interface SearchResult {
  items: SearchResultItem[];
  totalCount: number;
  searchTime: number;
  cacheHit: boolean;
}

export interface SearchResultItem {
  chunk: UnifiedChunk;
  score: number;
  context: string;
  position: Position;
  highlights: TextHighlight[];
}

export interface TextHighlight {
  start: number;
  end: number;
  type: 'match' | 'context';
}

// Template-related types
export interface Template {
  id: string;
  name: string;
  slots: TemplateSlot[];
  structure: TemplateStructure;
  metadata: TemplateMetadata;
}

export interface TemplateSlot {
  id: string;
  name: string;
  type: 'text' | 'number' | 'date' | 'link' | 'tag';
  required: boolean;
  defaultValue?: any;
  validation?: ValidationRule;
}

export interface TemplateStructure {
  layout: string;
  sections: TemplateSection[];
}

export interface TemplateSection {
  id: string;
  title: string;
  content: string;
  slots: string[];
}

export interface TemplateMetadata {
  description?: string;
  category?: string;
  tags: string[];
  createdTime: Date;
  lastUpdated: Date;
}

export interface ValidationRule {
  pattern?: string;
  minLength?: number;
  maxLength?: number;
  required?: boolean;
  customValidator?: (value: any) => boolean;
}

export interface TemplateInstance {
  id: string;
  templateId: string;
  filePath: string;
  slotValues: Record<string, any>;
  createdAt: Date;
  updatedAt: Date;
}

// AI-related types
export interface AIResponse {
  message: string;
  suggestions?: ContentSuggestion[];
  actions?: AIAction[];
  metadata: ResponseMetadata;
}

export interface ContentSuggestion {
  type: 'improvement' | 'expansion' | 'correction';
  content: string;
  confidence: number;
  position?: Position;
}

export interface AIAction {
  type: 'create' | 'update' | 'delete' | 'tag' | 'link';
  target: string;
  data: any;
}

export interface ResponseMetadata {
  model: string;
  tokens?: number;
  tokensUsed?: number;
  processingTime: number;
  confidence: number;
}

export interface ProcessingResult {
  chunks: UnifiedChunk[];
  suggestions: ContentSuggestion[];
  improvements: ContentImprovement[];
}

export interface ContentImprovement {
  type: 'grammar' | 'style' | 'structure' | 'clarity';
  original: string;
  improved: string;
  explanation: string;
  position: Position;
}

// Error handling types
export enum ErrorType {
  NETWORK_ERROR = 'network_error',
  API_ERROR = 'api_error',
  PARSING_ERROR = 'parsing_error',
  SYNC_ERROR = 'sync_error',
  VALIDATION_ERROR = 'validation_error'
}

export class PluginError extends Error {
  type: ErrorType;
  code: string;
  details?: any;
  recoverable: boolean;

  constructor(type: ErrorType, code: string, details?: any, recoverable: boolean = true) {
    super(`${type}: ${code}`);
    this.name = 'PluginError';
    this.type = type;
    this.code = code;
    this.details = details;
    this.recoverable = recoverable;
  }
}

// Offline operation types
export interface OfflineOperation {
  id: string;
  type: 'create' | 'update' | 'delete';
  data: any;
  timestamp: Date;
  priority: number;
}

// Memory management types
export interface MemoryStats {
  totalMemory: number;
  usedMemory: number;
  cacheSize: number;
  pendingOperations: number;
}

// User action types for testing
export interface UserAction {
  type: 'click' | 'type' | 'scroll' | 'key';
  target: string;
  data?: any;
}

// Search history types
export interface SearchHistory {
  query: SearchQuery;
  timestamp: Date;
  resultCount: number;
}

// Document ID pagination types
export type DocumentScope = 'file' | 'virtual' | 'page';

export interface VirtualDocumentContext {
  sourceType: 'remnote' | 'logseq' | 'obsidian-template';
  contextId: string;
  pageTitle?: string;
  metadata: Record<string, any>;
}

export interface VirtualDocument {
  virtualDocumentId: string;
  context: VirtualDocumentContext;
  chunkIds: string[];
  createdAt: Date;
  lastUpdated: Date;
}

export interface DocumentMetadata {
  originalFilePath?: string;
  virtualContext?: VirtualDocumentContext;
  totalChunks: number;
  documentScope: DocumentScope;
  lastModified: Date;
}

export interface ReconstructedDocument {
  documentId: string;
  chunks: UnifiedChunk[];
  hierarchy: HierarchyNode[];
  metadata: DocumentMetadata;
  reconstructionTime: Date;
}

export interface PaginationOptions {
  page?: number;
  pageSize?: number;
  includeHierarchy?: boolean;
  sortBy?: 'position' | 'created' | 'updated';
  sortOrder?: 'asc' | 'desc';
}

export interface DocumentChunksResult {
  chunks: UnifiedChunk[];
  pagination: {
    currentPage: number;
    totalPages: number;
    totalChunks: number;
    pageSize: number;
  };
  documentMetadata: DocumentMetadata;
}

export interface DocumentScopeUpdate {
  chunkId: string;
  documentId: string;
  scope: DocumentScope;
  metadata?: Record<string, any>;
}

// Additional AI types for chat functionality
export interface ChatMessage {
  id: string;
  content: string;
  role: 'user' | 'assistant' | 'system';
  timestamp: Date;
  metadata: Record<string, any>;
}

export interface ChatHistory {
  messages: ChatMessage[];
  sessionId: string;
  startTime: Date;
  lastActivity: Date;
}

export interface ConversationContext {
  activeFiles: string[];
  relevantChunks: UnifiedChunk[];
  userPreferences: Record<string, any>;
}

export interface AIManagerSettings {
  maxHistorySize?: number;
  contextWindowSize?: number;
  autoSaveHistory?: boolean;
  enableSuggestions?: boolean;
}