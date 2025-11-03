/**
 * Batch Processing Modal for handling folder-based image uploads
 */

import { Modal, App, Setting, ButtonComponent, DropdownComponent, ToggleComponent, Notice } from 'obsidian';
import { 
  BatchUploadRequest, 
  BatchUploadResult, 
  BatchUploadProgress,
  BatchUploadError
} from '../types/media';
import { IImageUploadManager } from './ImageUploadManager';

export class BatchProcessModal extends Modal {
  private uploadManager: IImageUploadManager;
  private selectedFiles: File[] = [];
  private isProcessing = false;
  private currentBatchId: string | null = null;

  // Settings
  private autoAnalyze = true;
  private autoEmbed = true;
  private storageType: 'local' | 'supabase' | 'google_drive' = 'google_drive'; // Will be updated based on plugin settings
  private concurrency = 3;
  private pageId = '';
  private tags: string[] = [];

  // UI Elements
  private fileListEl!: HTMLElement;
  private progressEl!: HTMLElement;
  private settingsEl!: HTMLElement;
  private actionsEl!: HTMLElement;
  private resultsEl!: HTMLElement;

  constructor(app: App, uploadManager: IImageUploadManager, defaultStorageType?: 'local' | 'supabase' | 'google_drive') {
    super(app);
    this.uploadManager = uploadManager;
    if (defaultStorageType) {
      this.storageType = defaultStorageType;
    }
  }

  onOpen() {
    const { contentEl } = this;
    contentEl.empty();
    contentEl.addClass('ink-batch-process-modal');

    this.createHeader();
    this.createFileSelection();
    this.createSettings();
    this.createProgress();
    this.createActions();
    this.createResults();
  }

  onClose() {
    if (this.isProcessing && this.currentBatchId) {
      // Cancel ongoing batch if modal is closed
      this.cancelBatch();
    }
  }

  /**
   * Create modal header
   */
  private createHeader(): void {
    const headerEl = this.contentEl.createDiv('ink-batch-header');
    
    const titleEl = headerEl.createEl('h2', { text: 'Batch Image Processing' });
    titleEl.addClass('ink-batch-title');

    const descEl = headerEl.createEl('p', { 
      text: 'Upload and process multiple images with AI analysis and embedding generation.',
      cls: 'ink-batch-description'
    });
  }

  /**
   * Create file selection area
   */
  private createFileSelection(): void {
    const sectionEl = this.contentEl.createDiv('ink-batch-section');
    sectionEl.createEl('h3', { text: 'Select Images' });

    // File selection buttons
    const buttonsEl = sectionEl.createDiv('ink-file-buttons');
    
    new ButtonComponent(buttonsEl)
      .setButtonText('Select Files')
      .setClass('mod-cta')
      .onClick(() => this.selectFiles());

    new ButtonComponent(buttonsEl)
      .setButtonText('Select Folder')
      .onClick(() => this.selectFolder());

    new ButtonComponent(buttonsEl)
      .setButtonText('Clear Selection')
      .onClick(() => this.clearSelection());

    // Drag and drop area
    const dropZoneEl = sectionEl.createDiv('ink-drop-zone');
    dropZoneEl.createEl('div', { 
      text: 'Or drag and drop images here',
      cls: 'ink-drop-text'
    });

    this.setupDropZone(dropZoneEl);

    // File list
    this.fileListEl = sectionEl.createDiv('ink-file-list');
    this.updateFileList();
  }

  /**
   * Create settings section
   */
  private createSettings(): void {
    this.settingsEl = this.contentEl.createDiv('ink-batch-section');
    this.settingsEl.createEl('h3', { text: 'Processing Settings' });

    // Page ID setting
    new Setting(this.settingsEl)
      .setName('Page ID')
      .setDesc('Associate images with a specific page (optional)')
      .addText(text => {
        text.setPlaceholder('page-id')
          .setValue(this.pageId)
          .onChange(value => {
            this.pageId = value;
          });
      });

    // Tags setting
    new Setting(this.settingsEl)
      .setName('Tags')
      .setDesc('Comma-separated tags to apply to all images')
      .addText(text => {
        text.setPlaceholder('tag1, tag2, tag3')
          .setValue(this.tags.join(', '))
          .onChange(value => {
            this.tags = value ? value.split(',').map(t => t.trim()).filter(t => t) : [];
          });
      });

    // Auto analyze setting
    new Setting(this.settingsEl)
      .setName('Auto Analyze')
      .setDesc('Automatically analyze images with AI')
      .addToggle(toggle => {
        toggle.setValue(this.autoAnalyze)
          .onChange(value => {
            this.autoAnalyze = value;
          });
      });

    // Auto embed setting
    new Setting(this.settingsEl)
      .setName('Auto Embed')
      .setDesc('Automatically generate embeddings for images')
      .addToggle(toggle => {
        toggle.setValue(this.autoEmbed)
          .onChange(value => {
            this.autoEmbed = value;
          });
      });

    // Storage type setting
    new Setting(this.settingsEl)
      .setName('Storage Type')
      .setDesc('Choose storage backend')
      .addDropdown(dropdown => {
        dropdown
          .addOption('google_drive', 'Google Drive')
          .addOption('supabase', 'Supabase')
          .addOption('local', 'Local')
          .setValue(this.storageType)
          .onChange(value => {
            this.storageType = value as 'local' | 'supabase' | 'google_drive';
          });
      });

    // Concurrency setting
    new Setting(this.settingsEl)
      .setName('Concurrency')
      .setDesc('Number of images to process simultaneously')
      .addSlider(slider => {
        slider
          .setLimits(1, 10, 1)
          .setValue(this.concurrency)
          .setDynamicTooltip()
          .onChange(value => {
            this.concurrency = value;
          });
      });
  }

  /**
   * Create progress section
   */
  private createProgress(): void {
    this.progressEl = this.contentEl.createDiv('ink-batch-section');
    this.progressEl.createEl('h3', { text: 'Progress' });
    this.progressEl.style.display = 'none';

    const progressContainer = this.progressEl.createDiv('ink-progress-container');
    
    // Progress bar
    const progressBarContainer = progressContainer.createDiv('ink-progress-bar-container');
    const progressBar = progressBarContainer.createDiv('ink-progress-bar');
    const progressFill = progressBar.createDiv('ink-progress-fill');
    progressFill.style.width = '0%';

    // Progress text
    const progressText = progressContainer.createDiv('ink-progress-text');
    progressText.createEl('span', { cls: 'ink-progress-current' });
    progressText.createEl('span', { text: ' / ', cls: 'ink-progress-separator' });
    progressText.createEl('span', { cls: 'ink-progress-total' });
    progressText.createEl('span', { text: ' images processed', cls: 'ink-progress-label' });

    // Current file
    const currentFile = progressContainer.createDiv('ink-current-file');
    currentFile.createEl('span', { text: 'Processing: ', cls: 'ink-current-label' });
    currentFile.createEl('span', { cls: 'ink-current-filename' });

    // Error list
    const errorList = progressContainer.createDiv('ink-error-list');
    errorList.style.display = 'none';
  }

  /**
   * Create actions section
   */
  private createActions(): void {
    this.actionsEl = this.contentEl.createDiv('ink-batch-actions');

    new ButtonComponent(this.actionsEl)
      .setButtonText('Start Processing')
      .setClass('mod-cta')
      .onClick(() => this.startBatchProcess());

    new ButtonComponent(this.actionsEl)
      .setButtonText('Cancel')
      .onClick(() => this.close());
  }

  /**
   * Create results section
   */
  private createResults(): void {
    this.resultsEl = this.contentEl.createDiv('ink-batch-section');
    this.resultsEl.createEl('h3', { text: 'Results' });
    this.resultsEl.style.display = 'none';
  }

  /**
   * Select files using file picker
   */
  private selectFiles(): void {
    const input = document.createElement('input');
    input.type = 'file';
    input.multiple = true;
    input.accept = 'image/*';

    input.addEventListener('change', (event) => {
      const files = (event.target as HTMLInputElement).files;
      if (files) {
        this.addFiles(Array.from(files));
      }
    });

    input.click();
  }

  /**
   * Select folder using directory picker
   */
  private selectFolder(): void {
    const input = document.createElement('input');
    input.type = 'file';
    input.webkitdirectory = true;
    input.multiple = true;

    input.addEventListener('change', (event) => {
      const files = (event.target as HTMLInputElement).files;
      if (files) {
        const imageFiles = Array.from(files).filter(file => 
          file.type.startsWith('image/')
        );
        this.addFiles(imageFiles);
      }
    });

    input.click();
  }

  /**
   * Setup drag and drop zone
   */
  private setupDropZone(dropZoneEl: HTMLElement): void {
    dropZoneEl.addEventListener('dragover', (e) => {
      e.preventDefault();
      dropZoneEl.addClass('ink-drop-zone-active');
    });

    dropZoneEl.addEventListener('dragleave', (e) => {
      e.preventDefault();
      dropZoneEl.removeClass('ink-drop-zone-active');
    });

    dropZoneEl.addEventListener('drop', (e) => {
      e.preventDefault();
      dropZoneEl.removeClass('ink-drop-zone-active');

      const files = Array.from(e.dataTransfer?.files || [])
        .filter(file => file.type.startsWith('image/'));
      
      if (files.length > 0) {
        this.addFiles(files);
      }
    });
  }

  /**
   * Add files to selection
   */
  private addFiles(files: File[]): void {
    // Remove duplicates based on name and size
    const existingFiles = new Set(
      this.selectedFiles.map(f => `${f.name}-${f.size}`)
    );

    const newFiles = files.filter(file => 
      !existingFiles.has(`${file.name}-${file.size}`)
    );

    this.selectedFiles.push(...newFiles);
    this.updateFileList();
  }

  /**
   * Clear file selection
   */
  private clearSelection(): void {
    this.selectedFiles = [];
    this.updateFileList();
  }

  /**
   * Update file list display
   */
  private updateFileList(): void {
    this.fileListEl.empty();

    if (this.selectedFiles.length === 0) {
      this.fileListEl.createEl('p', { 
        text: 'No images selected',
        cls: 'ink-empty-state'
      });
      return;
    }

    const listEl = this.fileListEl.createEl('ul', { cls: 'ink-file-items' });
    
    this.selectedFiles.forEach((file, index) => {
      const itemEl = listEl.createEl('li', { cls: 'ink-file-item' });
      
      const nameEl = itemEl.createEl('span', { 
        text: file.name,
        cls: 'ink-file-name'
      });
      
      const sizeEl = itemEl.createEl('span', { 
        text: this.formatFileSize(file.size),
        cls: 'ink-file-size'
      });

      const removeBtn = itemEl.createEl('button', { 
        text: '×',
        cls: 'ink-file-remove'
      });
      
      removeBtn.addEventListener('click', () => {
        this.selectedFiles.splice(index, 1);
        this.updateFileList();
      });
    });

    // Summary
    const summaryEl = this.fileListEl.createDiv('ink-file-summary');
    summaryEl.createEl('span', { 
      text: `${this.selectedFiles.length} images selected`,
      cls: 'ink-file-count'
    });

    const totalSize = this.selectedFiles.reduce((sum, file) => sum + file.size, 0);
    summaryEl.createEl('span', { 
      text: `Total size: ${this.formatFileSize(totalSize)}`,
      cls: 'ink-total-size'
    });
  }

  /**
   * Start batch processing
   */
  private async startBatchProcess(): Promise<void> {
    if (this.selectedFiles.length === 0) {
      new Notice('Please select images to process');
      return;
    }

    if (this.isProcessing) {
      return;
    }

    this.isProcessing = true;
    this.showProgress();
    this.hideActions();

    const request: BatchUploadRequest = {
      files: this.selectedFiles,
      pageId: this.pageId || undefined,
      tags: this.tags.length > 0 ? this.tags : undefined,
      autoAnalyze: this.autoAnalyze,
      autoEmbed: this.autoEmbed,
      storageType: this.storageType,
      concurrency: this.concurrency
    };

    try {
      const result = await this.uploadManager.uploadBatch(request, (progress) => {
        this.updateProgress(progress);
      });

      this.showResults(result);
    } catch (error) {
      console.error('Batch processing failed:', error);
      new Notice(`Batch processing failed: ${error}`);
    } finally {
      this.isProcessing = false;
      this.showActions();
    }
  }

  /**
   * Cancel batch processing
   */
  private async cancelBatch(): Promise<void> {
    if (this.currentBatchId) {
      // Implementation would depend on upload manager having cancel capability
      console.log('Cancelling batch:', this.currentBatchId);
    }
  }

  /**
   * Update progress display
   */
  private updateProgress(progress: BatchUploadProgress): void {
    this.currentBatchId = progress.batchId;

    const progressFill = this.progressEl.querySelector('.ink-progress-fill') as HTMLElement;
    const currentEl = this.progressEl.querySelector('.ink-progress-current') as HTMLElement;
    const totalEl = this.progressEl.querySelector('.ink-progress-total') as HTMLElement;
    const filenameEl = this.progressEl.querySelector('.ink-current-filename') as HTMLElement;

    if (progressFill) {
      progressFill.style.width = `${progress.progress}%`;
    }

    if (currentEl) {
      currentEl.textContent = (progress.processedFiles + progress.failedFiles).toString();
    }

    if (totalEl) {
      totalEl.textContent = progress.totalFiles.toString();
    }

    if (filenameEl && progress.currentFile) {
      filenameEl.textContent = progress.currentFile;
    }

    // Show errors if any
    if (progress.errors.length > 0) {
      this.showErrors(progress.errors);
    }
  }

  /**
   * Show processing errors
   */
  private showErrors(errors: BatchUploadError[]): void {
    const errorList = this.progressEl.querySelector('.ink-error-list') as HTMLElement;
    if (!errorList) return;

    errorList.style.display = 'block';
    errorList.empty();

    const errorTitle = errorList.createEl('h4', { text: 'Errors' });
    const errorItems = errorList.createEl('ul');

    errors.forEach(error => {
      const errorItem = errorItems.createEl('li');
      errorItem.createEl('strong', { text: error.filename });
      errorItem.createEl('span', { text: `: ${error.error}` });
    });
  }

  /**
   * Show final results
   */
  private showResults(result: BatchUploadResult): void {
    this.resultsEl.style.display = 'block';
    this.resultsEl.empty();
    this.resultsEl.createEl('h3', { text: 'Processing Complete' });

    const statsEl = this.resultsEl.createDiv('ink-result-stats');
    
    statsEl.createEl('div', { 
      text: `✅ ${result.successCount} successful`,
      cls: 'ink-stat-success'
    });
    
    statsEl.createEl('div', { 
      text: `❌ ${result.failureCount} failed`,
      cls: 'ink-stat-error'
    });
    
    statsEl.createEl('div', { 
      text: `⏱️ ${Math.round(result.duration / 1000)}s total`,
      cls: 'ink-stat-time'
    });

    // Show successful uploads
    if (result.successful.length > 0) {
      const successEl = this.resultsEl.createDiv('ink-result-section');
      successEl.createEl('h4', { text: 'Successfully Processed' });
      
      const successList = successEl.createEl('ul');
      result.successful.forEach(upload => {
        const item = successList.createEl('li');
        item.createEl('span', { text: upload.chunkId });
        if (upload.analysis?.description) {
          item.createEl('small', { 
            text: ` - ${upload.analysis.description}`,
            cls: 'ink-analysis-preview'
          });
        }
      });
    }

    // Show failed uploads
    if (result.failed.length > 0) {
      const failedEl = this.resultsEl.createDiv('ink-result-section');
      failedEl.createEl('h4', { text: 'Failed to Process' });
      
      const failedList = failedEl.createEl('ul');
      result.failed.forEach(error => {
        const item = failedList.createEl('li');
        item.createEl('strong', { text: error.filename });
        item.createEl('span', { text: `: ${error.error}` });
      });
    }

    // Actions
    const actionsEl = this.resultsEl.createDiv('ink-result-actions');
    
    new ButtonComponent(actionsEl)
      .setButtonText('Process More Images')
      .onClick(() => {
        this.resetModal();
      });

    new ButtonComponent(actionsEl)
      .setButtonText('Close')
      .setClass('mod-cta')
      .onClick(() => {
        this.close();
      });
  }

  /**
   * Show progress section
   */
  private showProgress(): void {
    this.progressEl.style.display = 'block';
  }

  /**
   * Hide actions section
   */
  private hideActions(): void {
    this.actionsEl.style.display = 'none';
  }

  /**
   * Show actions section
   */
  private showActions(): void {
    this.actionsEl.style.display = 'block';
  }

  /**
   * Reset modal to initial state
   */
  private resetModal(): void {
    this.selectedFiles = [];
    this.isProcessing = false;
    this.currentBatchId = null;
    
    this.updateFileList();
    this.progressEl.style.display = 'none';
    this.resultsEl.style.display = 'none';
    this.showActions();
  }

  /**
   * Format file size for display
   */
  private formatFileSize(bytes: number): string {
    if (bytes === 0) return '0 B';
    
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    
    return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
  }
}