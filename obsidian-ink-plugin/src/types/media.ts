/**
 * Media and image upload related types
 */

export interface ImageUploadRequest {
  file: File | ArrayBuffer;
  filename: string;
  pageId?: string;
  tags?: string[];
  autoAnalyze?: boolean;
  autoEmbed?: boolean;
  storageType?: 'local' | 'supabase' | 'google_drive';
}

export interface ImageUploadResponse {
  chunkId: string;
  imageUrl: string;
  storageId: string;
  fileHash: string;
  analysis?: ImageAnalysis;
  embeddingIds?: string[];
}

export interface ImageAnalysis {
  description: string;
  tags: string[];
  confidence: number;
  model: string;
  analyzedAt: Date;
}

export interface ImageMetadata {
  width?: number;
  height?: number;
  format?: string;
  size?: number;
  colorSpace?: string;
  hasAlpha?: boolean;
}

export interface BatchUploadRequest {
  files: File[];
  pageId?: string;
  tags?: string[];
  autoAnalyze?: boolean;
  autoEmbed?: boolean;
  storageType?: 'local' | 'supabase' | 'google_drive';
  concurrency?: number;
}

export interface BatchUploadProgress {
  batchId: string;
  totalFiles: number;
  processedFiles: number;
  failedFiles: number;
  currentFile?: string;
  progress: number; // 0-100
  errors: BatchUploadError[];
}

export interface BatchUploadError {
  filename: string;
  error: string;
  timestamp: Date;
}

export interface BatchUploadResult {
  batchId: string;
  successful: ImageUploadResponse[];
  failed: BatchUploadError[];
  totalFiles: number;
  successCount: number;
  failureCount: number;
  duration: number;
}

export interface ImageSearchRequest {
  query?: string;
  imageUrl?: string;
  searchType: 'text' | 'image' | 'hybrid';
  limit?: number;
  minSimilarity?: number;
  filters?: Record<string, any>;
}

export interface ImageSearchResult {
  chunkId: string;
  imageUrl: string;
  similarity: number;
  matchType: string;
  explanation: string;
  tags: string[];
  createdAt: Date;
}

export interface SlideImageRecommendation {
  chunkId: string;
  imageUrl: string;
  title: string;
  description: string;
  relevanceScore: number;
  matchReason: string;
  tags: string[];
  imageMetadata?: ImageMetadata;
}

export interface SlideRecommendationRequest {
  slideTitle?: string;
  slideContent?: string;
  slideContext?: string;
  maxSuggestions?: number;
  minRelevance?: number;
  preferredStyles?: string[];
  excludeImageIds?: string[];
}

export interface DuplicateImageGroup {
  groupId: string;
  images: Array<{
    chunkId: string;
    imageUrl: string;
    similarity: number;
  }>;
  groupSize: number;
  maxSimilarity: number;
}

export interface DuplicateSearchOptions {
  similarityThreshold?: number;
  minGroupSize?: number;
  includeMetadata?: boolean;
}

export interface ImageLibraryFilter {
  tags?: string[];
  dateRange?: {
    start: Date;
    end: Date;
  };
  analysisStatus?: 'analyzed' | 'not_analyzed' | 'all';
  storageType?: 'local' | 'supabase' | 'google_drive' | 'all';
  minSimilarity?: number;
  searchQuery?: string;
}

export interface ImageLibraryItem {
  chunkId: string;
  imageUrl: string;
  thumbnailUrl?: string;
  filename: string;
  tags: string[];
  analysis?: ImageAnalysis;
  metadata?: ImageMetadata;
  createdAt: Date;
  updatedAt: Date;
  storageType: 'local' | 'supabase' | 'google_drive';
}

export interface ImageLibraryResponse {
  items: ImageLibraryItem[];
  totalCount: number;
  hasMore: boolean;
  nextCursor?: string;
}

// Upload progress callback type
export type UploadProgressCallback = (progress: {
  filename: string;
  loaded: number;
  total: number;
  percentage: number;
}) => void;

// Batch upload progress callback type
export type BatchProgressCallback = (progress: BatchUploadProgress) => void;

// Image processing events
export interface ImageProcessingEvent {
  type: 'upload_start' | 'upload_progress' | 'upload_complete' | 'upload_error' | 
        'analysis_start' | 'analysis_complete' | 'analysis_error' |
        'embedding_start' | 'embedding_complete' | 'embedding_error';
  filename: string;
  data?: any;
  error?: string;
  timestamp: Date;
}

export type ImageProcessingEventCallback = (event: ImageProcessingEvent) => void;