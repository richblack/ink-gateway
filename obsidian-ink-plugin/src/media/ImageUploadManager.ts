/**
 * Image Upload Manager for handling image uploads and processing
 */

import { Notice, TFile } from 'obsidian';
import { 
  ImageUploadRequest, 
  ImageUploadResponse, 
  BatchUploadRequest, 
  BatchUploadResult, 
  BatchUploadProgress,
  UploadProgressCallback,
  BatchProgressCallback,
  ImageProcessingEvent,
  ImageProcessingEventCallback
} from '../types/media';
import { IInkGatewayClient } from '../interfaces';
import { PluginError, ErrorType } from '../types';

export interface IImageUploadManager {
  uploadImage(request: ImageUploadRequest, onProgress?: UploadProgressCallback): Promise<ImageUploadResponse>;
  uploadBatch(request: BatchUploadRequest, onProgress?: BatchProgressCallback): Promise<BatchUploadResult>;
  uploadFromFile(file: TFile, options?: Partial<ImageUploadRequest>): Promise<ImageUploadResponse>;
  uploadFromPath(filePath: string, options?: Partial<ImageUploadRequest>): Promise<ImageUploadResponse>;
  uploadFromClipboard(options?: Partial<ImageUploadRequest>): Promise<ImageUploadResponse>;
  cancelUpload(uploadId: string): Promise<void>;
  getUploadProgress(uploadId: string): BatchUploadProgress | null;
  addEventListener(callback: ImageProcessingEventCallback): void;
  removeEventListener(callback: ImageProcessingEventCallback): void;
}

export class ImageUploadManager implements IImageUploadManager {
  private apiClient: IInkGatewayClient;
  private activeUploads: Map<string, AbortController> = new Map();
  private uploadProgress: Map<string, BatchUploadProgress> = new Map();
  private eventListeners: Set<ImageProcessingEventCallback> = new Set();

  constructor(apiClient: IInkGatewayClient) {
    this.apiClient = apiClient;
  }

  /**
   * Upload a single image
   */
  async uploadImage(
    request: ImageUploadRequest, 
    onProgress?: UploadProgressCallback
  ): Promise<ImageUploadResponse> {
    const uploadId = this.generateUploadId();
    const abortController = new AbortController();
    this.activeUploads.set(uploadId, abortController);

    try {
      this.emitEvent({
        type: 'upload_start',
        filename: request.filename,
        timestamp: new Date()
      });

      // Convert file to base64 if needed
      let imageData: string;
      if (request.file instanceof File) {
        imageData = await this.fileToBase64(request.file);
      } else {
        imageData = this.arrayBufferToBase64(request.file);
      }

      // Prepare API request
      const apiRequest = {
        image_data: imageData,
        filename: request.filename,
        page_id: request.pageId,
        tags: request.tags || [],
        auto_analyze: request.autoAnalyze ?? true,
        auto_embed: request.autoEmbed ?? true,
        storage_type: request.storageType || 'supabase'
      };

      // Make API call with progress tracking
      const response = await this.apiClient.request<ImageUploadResponse>({
        method: 'POST',
        endpoint: '/api/v1/media/upload',
        data: apiRequest,
        headers: {
          'Content-Type': 'application/json'
        }
      });

      this.emitEvent({
        type: 'upload_complete',
        filename: request.filename,
        data: response,
        timestamp: new Date()
      });

      return response;

    } catch (error) {
      this.emitEvent({
        type: 'upload_error',
        filename: request.filename,
        error: error instanceof Error ? error.message : String(error),
        timestamp: new Date()
      });

      throw new PluginError(
        ErrorType.API_ERROR,
        `Failed to upload image ${request.filename}: ${error}`,
        { originalError: error }
      );
    } finally {
      this.activeUploads.delete(uploadId);
    }
  }

  /**
   * Upload multiple images in batch
   */
  async uploadBatch(
    request: BatchUploadRequest, 
    onProgress?: BatchProgressCallback
  ): Promise<BatchUploadResult> {
    const batchId = this.generateUploadId();
    const startTime = Date.now();
    
    const progress: BatchUploadProgress = {
      batchId,
      totalFiles: request.files.length,
      processedFiles: 0,
      failedFiles: 0,
      progress: 0,
      errors: []
    };

    this.uploadProgress.set(batchId, progress);

    const successful: ImageUploadResponse[] = [];
    const concurrency = request.concurrency || 3;

    try {
      // Process files in batches with concurrency control
      for (let i = 0; i < request.files.length; i += concurrency) {
        const batch = request.files.slice(i, i + concurrency);
        
        const batchPromises = batch.map(async (file) => {
          try {
            progress.currentFile = file.name;
            onProgress?.(progress);

            const uploadRequest: ImageUploadRequest = {
              file,
              filename: file.name,
              pageId: request.pageId,
              tags: request.tags,
              autoAnalyze: request.autoAnalyze,
              autoEmbed: request.autoEmbed,
              storageType: request.storageType
            };

            const result = await this.uploadImage(uploadRequest);
            successful.push(result);
            progress.processedFiles++;

          } catch (error) {
            progress.failedFiles++;
            progress.errors.push({
              filename: file.name,
              error: error instanceof Error ? error.message : String(error),
              timestamp: new Date()
            });
          }

          progress.progress = ((progress.processedFiles + progress.failedFiles) / progress.totalFiles) * 100;
          onProgress?.(progress);
        });

        await Promise.all(batchPromises);
      }

      const result: BatchUploadResult = {
        batchId,
        successful,
        failed: progress.errors,
        totalFiles: request.files.length,
        successCount: successful.length,
        failureCount: progress.errors.length,
        duration: Date.now() - startTime
      };

      return result;

    } finally {
      this.uploadProgress.delete(batchId);
    }
  }

  /**
   * Upload image from Obsidian TFile
   */
  async uploadFromFile(
    file: TFile, 
    options: Partial<ImageUploadRequest> = {}
  ): Promise<ImageUploadResponse> {
    if (!this.isImageFile(file)) {
      throw new PluginError(
        ErrorType.VALIDATION_ERROR,
        `File ${file.name} is not a supported image format`
      );
    }

    const arrayBuffer = await this.app.vault.readBinary(file);
    
    const request: ImageUploadRequest = {
      file: arrayBuffer,
      filename: file.name,
      ...options
    };

    return this.uploadImage(request);
  }

  /**
   * Upload image from file path
   */
  async uploadFromPath(
    filePath: string, 
    options: Partial<ImageUploadRequest> = {}
  ): Promise<ImageUploadResponse> {
    const file = this.app.vault.getAbstractFileByPath(filePath);
    
    if (!file || !(file instanceof TFile)) {
      throw new PluginError(
        ErrorType.VALIDATION_ERROR,
        `File not found: ${filePath}`
      );
    }

    return this.uploadFromFile(file, options);
  }

  /**
   * Upload image from clipboard
   */
  async uploadFromClipboard(
    options: Partial<ImageUploadRequest> = {}
  ): Promise<ImageUploadResponse> {
    try {
      const clipboardItems = await navigator.clipboard.read();
      
      for (const item of clipboardItems) {
        for (const type of item.types) {
          if (type.startsWith('image/')) {
            const blob = await item.getType(type);
            const arrayBuffer = await blob.arrayBuffer();
            
            const filename = options.filename || `clipboard-image-${Date.now()}.${this.getExtensionFromMimeType(type)}`;
            
            const request: ImageUploadRequest = {
              file: arrayBuffer,
              filename,
              ...options
            };

            return this.uploadImage(request);
          }
        }
      }

      throw new Error('No image found in clipboard');

    } catch (error) {
      throw new PluginError(
        ErrorType.VALIDATION_ERROR,
        `Failed to read image from clipboard: ${error}`
      );
    }
  }

  /**
   * Cancel an ongoing upload
   */
  async cancelUpload(uploadId: string): Promise<void> {
    const controller = this.activeUploads.get(uploadId);
    if (controller) {
      controller.abort();
      this.activeUploads.delete(uploadId);
    }
  }

  /**
   * Get upload progress for a batch
   */
  getUploadProgress(uploadId: string): BatchUploadProgress | null {
    return this.uploadProgress.get(uploadId) || null;
  }

  /**
   * Add event listener
   */
  addEventListener(callback: ImageProcessingEventCallback): void {
    this.eventListeners.add(callback);
  }

  /**
   * Remove event listener
   */
  removeEventListener(callback: ImageProcessingEventCallback): void {
    this.eventListeners.delete(callback);
  }

  // Private helper methods

  private generateUploadId(): string {
    return `upload_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
  }

  private async fileToBase64(file: File): Promise<string> {
    return new Promise((resolve, reject) => {
      const reader = new FileReader();
      reader.onload = () => {
        const result = reader.result as string;
        // Remove data URL prefix (data:image/jpeg;base64,)
        const base64 = result.split(',')[1];
        resolve(base64);
      };
      reader.onerror = reject;
      reader.readAsDataURL(file);
    });
  }

  private arrayBufferToBase64(buffer: ArrayBuffer): string {
    const bytes = new Uint8Array(buffer);
    let binary = '';
    for (let i = 0; i < bytes.byteLength; i++) {
      binary += String.fromCharCode(bytes[i]);
    }
    return btoa(binary);
  }

  private isImageFile(file: TFile): boolean {
    const imageExtensions = ['.jpg', '.jpeg', '.png', '.gif', '.webp', '.bmp', '.tiff', '.svg'];
    return imageExtensions.some(ext => file.name.toLowerCase().endsWith(ext));
  }

  private getExtensionFromMimeType(mimeType: string): string {
    const mimeToExt: Record<string, string> = {
      'image/jpeg': 'jpg',
      'image/png': 'png',
      'image/gif': 'gif',
      'image/webp': 'webp',
      'image/bmp': 'bmp',
      'image/tiff': 'tiff',
      'image/svg+xml': 'svg'
    };
    return mimeToExt[mimeType] || 'jpg';
  }

  private emitEvent(event: ImageProcessingEvent): void {
    this.eventListeners.forEach(callback => {
      try {
        callback(event);
      } catch (error) {
        console.error('Error in image processing event listener:', error);
      }
    });
  }

  // Note: app property would be injected by the plugin
  private get app() {
    return (window as any).app;
  }
}