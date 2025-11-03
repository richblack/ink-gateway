/**
 * Drag and Drop Handler for image uploads
 */

import { Editor, MarkdownView, Notice } from 'obsidian';
import { IImageUploadManager } from './ImageUploadManager';
import { ImageUploadRequest, ImageUploadResponse } from '../types/media';

export interface DragDropOptions {
  autoAnalyze?: boolean;
  autoEmbed?: boolean;
  storageType?: 'local' | 'supabase' | 'google_drive';
  insertMode?: 'link' | 'embed' | 'both';
  showProgress?: boolean;
}

export class DragDropHandler {
  private uploadManager: IImageUploadManager;
  private options: DragDropOptions;

  constructor(uploadManager: IImageUploadManager, options: DragDropOptions = {}) {
    this.uploadManager = uploadManager;
    this.options = {
      autoAnalyze: true,
      autoEmbed: true,
      storageType: 'google_drive',
      insertMode: 'embed',
      showProgress: true,
      ...options
    };
  }

  /**
   * Initialize drag and drop handlers
   */
  initialize(): void {
    this.setupDragDropListeners();
    this.setupPasteListener();
  }

  /**
   * Cleanup drag and drop handlers
   */
  cleanup(): void {
    this.removeDragDropListeners();
    this.removePasteListener();
  }

  /**
   * Setup drag and drop event listeners
   */
  private setupDragDropListeners(): void {
    // Add listeners to editor containers
    document.addEventListener('dragover', this.handleDragOver.bind(this));
    document.addEventListener('drop', this.handleDrop.bind(this));
    document.addEventListener('dragenter', this.handleDragEnter.bind(this));
    document.addEventListener('dragleave', this.handleDragLeave.bind(this));
  }

  /**
   * Remove drag and drop event listeners
   */
  private removeDragDropListeners(): void {
    document.removeEventListener('dragover', this.handleDragOver.bind(this));
    document.removeEventListener('drop', this.handleDrop.bind(this));
    document.removeEventListener('dragenter', this.handleDragEnter.bind(this));
    document.removeEventListener('dragleave', this.handleDragLeave.bind(this));
  }

  /**
   * Setup paste event listener
   */
  private setupPasteListener(): void {
    document.addEventListener('paste', this.handlePaste.bind(this));
  }

  /**
   * Remove paste event listener
   */
  private removePasteListener(): void {
    document.removeEventListener('paste', this.handlePaste.bind(this));
  }

  /**
   * Handle drag over event
   */
  private handleDragOver(event: DragEvent): void {
    if (!this.hasImageFiles(event)) return;

    event.preventDefault();
    event.stopPropagation();
    
    if (event.dataTransfer) {
      event.dataTransfer.dropEffect = 'copy';
    }

    this.addDropZoneHighlight();
  }

  /**
   * Handle drag enter event
   */
  private handleDragEnter(event: DragEvent): void {
    if (!this.hasImageFiles(event)) return;

    event.preventDefault();
    event.stopPropagation();
    this.addDropZoneHighlight();
  }

  /**
   * Handle drag leave event
   */
  private handleDragLeave(event: DragEvent): void {
    if (!this.hasImageFiles(event)) return;

    event.preventDefault();
    event.stopPropagation();
    
    // Only remove highlight if we're leaving the document
    if (!event.relatedTarget || !document.contains(event.relatedTarget as Node)) {
      this.removeDropZoneHighlight();
    }
  }

  /**
   * Handle drop event
   */
  private async handleDrop(event: DragEvent): Promise<void> {
    if (!this.hasImageFiles(event)) return;

    event.preventDefault();
    event.stopPropagation();
    this.removeDropZoneHighlight();

    const files = this.getImageFiles(event);
    if (files.length === 0) return;

    const activeView = this.getActiveMarkdownView();
    if (!activeView) {
      new Notice('Please open a markdown file to upload images');
      return;
    }

    await this.processImageFiles(files, activeView);
  }

  /**
   * Handle paste event
   */
  private async handlePaste(event: ClipboardEvent): Promise<void> {
    const activeView = this.getActiveMarkdownView();
    if (!activeView) return;

    const clipboardItems = event.clipboardData?.items;
    if (!clipboardItems) return;

    const imageItems: DataTransferItem[] = [];
    for (let i = 0; i < clipboardItems.length; i++) {
      const item = clipboardItems[i];
      if (item.type.startsWith('image/')) {
        imageItems.push(item);
      }
    }

    if (imageItems.length === 0) return;

    event.preventDefault();
    event.stopPropagation();

    // Process clipboard images
    for (const item of imageItems) {
      const file = item.getAsFile();
      if (file) {
        await this.processImageFiles([file], activeView);
      }
    }
  }

  /**
   * Process uploaded image files
   */
  private async processImageFiles(files: File[], view: MarkdownView): Promise<void> {
    const editor = view.editor;
    const currentFile = view.file;
    
    if (!currentFile) {
      new Notice('No active file to insert images');
      return;
    }

    // Get current page ID (file path without extension)
    const pageId = currentFile.path.replace(/\.[^/.]+$/, '');

    for (const file of files) {
      try {
        if (this.options.showProgress) {
          new Notice(`Uploading ${file.name}...`);
        }

        const uploadRequest: ImageUploadRequest = {
          file,
          filename: file.name,
          pageId,
          autoAnalyze: this.options.autoAnalyze,
          autoEmbed: this.options.autoEmbed,
          storageType: this.options.storageType
        };

        const response = await this.uploadManager.uploadImage(uploadRequest, (progress) => {
          if (this.options.showProgress) {
            // Update progress notice (simplified for now)
            console.log(`Upload progress: ${progress.percentage}%`);
          }
        });

        // Insert image link/embed into editor
        await this.insertImageIntoEditor(editor, response, file.name);

        if (this.options.showProgress) {
          new Notice(`Successfully uploaded ${file.name}`);
        }

      } catch (error) {
        console.error(`Failed to upload ${file.name}:`, error);
        new Notice(`Failed to upload ${file.name}: ${error}`);
      }
    }
  }

  /**
   * Insert image into editor
   */
  private async insertImageIntoEditor(
    editor: Editor, 
    uploadResponse: ImageUploadResponse, 
    originalFilename: string
  ): Promise<void> {
    const cursor = editor.getCursor();
    let insertText = '';

    switch (this.options.insertMode) {
      case 'link':
        insertText = `[${originalFilename}](${uploadResponse.imageUrl})`;
        break;
      case 'embed':
        insertText = `![${originalFilename}](${uploadResponse.imageUrl})`;
        break;
      case 'both':
        insertText = `![${originalFilename}](${uploadResponse.imageUrl})\n[View Image](${uploadResponse.imageUrl})`;
        break;
      default:
        insertText = `![${originalFilename}](${uploadResponse.imageUrl})`;
    }

    // Add AI analysis as comment if available
    if (uploadResponse.analysis && uploadResponse.analysis.description) {
      insertText += `\n<!-- AI Analysis: ${uploadResponse.analysis.description} -->`;
    }

    // Add tags as comment if available
    if (uploadResponse.analysis && uploadResponse.analysis.tags.length > 0) {
      insertText += `\n<!-- Tags: ${uploadResponse.analysis.tags.join(', ')} -->`;
    }

    editor.replaceRange(insertText, cursor);
    
    // Move cursor to end of inserted text
    const newCursor = {
      line: cursor.line,
      ch: cursor.ch + insertText.length
    };
    editor.setCursor(newCursor);
  }

  /**
   * Check if drag event contains image files
   */
  private hasImageFiles(event: DragEvent): boolean {
    if (!event.dataTransfer) return false;

    const types = event.dataTransfer.types;
    return types.includes('Files');
  }

  /**
   * Get image files from drag event
   */
  private getImageFiles(event: DragEvent): File[] {
    if (!event.dataTransfer) return [];

    const files: File[] = [];
    const items = event.dataTransfer.files;

    for (let i = 0; i < items.length; i++) {
      const file = items[i];
      if (file.type.startsWith('image/')) {
        files.push(file);
      }
    }

    return files;
  }

  /**
   * Get active markdown view
   */
  private getActiveMarkdownView(): MarkdownView | null {
    const activeView = (window as any).app.workspace.getActiveViewOfType(MarkdownView);
    return activeView;
  }

  /**
   * Add visual highlight for drop zone
   */
  private addDropZoneHighlight(): void {
    const activeView = this.getActiveMarkdownView();
    if (!activeView) return;

    const editorEl = activeView.contentEl.querySelector('.cm-editor');
    if (editorEl) {
      editorEl.classList.add('ink-drop-zone-active');
    }
  }

  /**
   * Remove visual highlight for drop zone
   */
  private removeDropZoneHighlight(): void {
    const activeView = this.getActiveMarkdownView();
    if (!activeView) return;

    const editorEl = activeView.contentEl.querySelector('.cm-editor');
    if (editorEl) {
      editorEl.classList.remove('ink-drop-zone-active');
    }
  }

  /**
   * Update options
   */
  updateOptions(newOptions: Partial<DragDropOptions>): void {
    this.options = { ...this.options, ...newOptions };
  }
}