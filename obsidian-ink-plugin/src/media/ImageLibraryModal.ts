/**
 * Image Library Modal for browsing and managing uploaded images
 */

import { Modal, App, Setting, ButtonComponent, TextComponent, MarkdownView } from 'obsidian';
import { 
  ImageLibraryItem, 
  ImageLibraryResponse, 
  ImageLibraryFilter,
  ImageSearchRequest,
  ImageSearchResult
} from '../types/media';
import { IInkGatewayClient } from '../interfaces';

export class ImageLibraryModal extends Modal {
  private apiClient: IInkGatewayClient;
  private images: ImageLibraryItem[] = [];
  private filteredImages: ImageLibraryItem[] = [];
  private currentFilter: ImageLibraryFilter = {};
  private selectedImages: Set<string> = new Set();
  private currentPage = 0;
  private pageSize = 20;
  private totalCount = 0;
  private isLoading = false;

  // UI Elements
  private modalContentEl!: HTMLElement;
  private searchInput!: TextComponent;
  private filterContainer!: HTMLElement;
  private imageGrid!: HTMLElement;
  private paginationContainer!: HTMLElement;
  private loadingIndicator!: HTMLElement;

  constructor(app: App, apiClient: IInkGatewayClient) {
    super(app);
    this.apiClient = apiClient;
  }

  onOpen() {
    const { contentEl } = this;
    this.modalContentEl = contentEl;
    
    contentEl.empty();
    contentEl.addClass('ink-image-library-modal');

    this.createHeader();
    this.createSearchAndFilters();
    this.createImageGrid();
    this.createPagination();
    this.createLoadingIndicator();

    // Load initial images
    this.loadImages();
  }

  onClose() {
    this.modalContentEl.empty();
  }

  /**
   * Create modal header
   */
  private createHeader(): void {
    const headerEl = this.modalContentEl.createDiv('ink-library-header');
    
    const titleEl = headerEl.createEl('h2', { text: 'Image Library' });
    titleEl.addClass('ink-library-title');

    const actionsEl = headerEl.createDiv('ink-library-actions');
    
    // Upload button
    new ButtonComponent(actionsEl)
      .setButtonText('Upload Images')
      .setClass('mod-cta')
      .onClick(() => {
        this.openUploadDialog();
      });

    // Batch actions
    new ButtonComponent(actionsEl)
      .setButtonText('Batch Actions')
      .onClick(() => {
        this.openBatchActionsMenu();
      });

    // Refresh button
    new ButtonComponent(actionsEl)
      .setButtonText('Refresh')
      .onClick(() => {
        this.loadImages();
      });
  }

  /**
   * Create search and filter controls
   */
  private createSearchAndFilters(): void {
    const searchContainer = this.modalContentEl.createDiv('ink-library-search');

    // Search input
    const searchSetting = new Setting(searchContainer)
      .setName('Search images')
      .setDesc('Search by filename, tags, or AI description');

    this.searchInput = new TextComponent(searchSetting.controlEl);
    this.searchInput
      .setPlaceholder('Search images...')
      .onChange(async (value) => {
        this.currentFilter.searchQuery = value;
        await this.applyFilters();
      });

    // Filter container
    this.filterContainer = this.modalContentEl.createDiv('ink-library-filters');
    this.createFilterControls();
  }

  /**
   * Create filter controls
   */
  private createFilterControls(): void {
    this.filterContainer.empty();

    // Tags filter
    new Setting(this.filterContainer)
      .setName('Filter by tags')
      .setDesc('Comma-separated list of tags')
      .addText(text => {
        text.setPlaceholder('tag1, tag2, tag3')
          .onChange(async (value) => {
            this.currentFilter.tags = value ? value.split(',').map(t => t.trim()) : undefined;
            await this.applyFilters();
          });
      });

    // Storage type filter
    new Setting(this.filterContainer)
      .setName('Storage type')
      .setDesc('Filter by storage backend')
      .addDropdown(dropdown => {
        dropdown
          .addOption('all', 'All')
          .addOption('local', 'Local')
          .addOption('supabase', 'Supabase')
          .addOption('google_drive', 'Google Drive')
          .onChange(async (value) => {
            this.currentFilter.storageType = value as any;
            await this.applyFilters();
          });
      });

    // Analysis status filter
    new Setting(this.filterContainer)
      .setName('Analysis status')
      .setDesc('Filter by AI analysis status')
      .addDropdown(dropdown => {
        dropdown
          .addOption('all', 'All')
          .addOption('analyzed', 'Analyzed')
          .addOption('not_analyzed', 'Not Analyzed')
          .onChange(async (value) => {
            this.currentFilter.analysisStatus = value as any;
            await this.applyFilters();
          });
      });

    // Date range filter
    const dateContainer = this.filterContainer.createDiv('ink-date-filter');
    new Setting(dateContainer)
      .setName('Date range')
      .setDesc('Filter by upload date')
      .addText(text => {
        text.setPlaceholder('YYYY-MM-DD')
          .onChange(async (value) => {
            if (value) {
              const date = new Date(value);
              if (!isNaN(date.getTime())) {
                this.currentFilter.dateRange = {
                  start: date,
                  end: new Date(date.getTime() + 24 * 60 * 60 * 1000) // Next day
                };
                await this.applyFilters();
              }
            } else {
              this.currentFilter.dateRange = undefined;
              await this.applyFilters();
            }
          });
      });
  }

  /**
   * Create image grid
   */
  private createImageGrid(): void {
    const gridContainer = this.modalContentEl.createDiv('ink-library-grid-container');
    
    // Grid header with view options
    const gridHeader = gridContainer.createDiv('ink-grid-header');
    
    const viewOptions = gridHeader.createDiv('ink-view-options');
    
    // Grid size controls
    new ButtonComponent(viewOptions)
      .setButtonText('Small')
      .onClick(() => this.setGridSize('small'));
    
    new ButtonComponent(viewOptions)
      .setButtonText('Medium')
      .onClick(() => this.setGridSize('medium'));
    
    new ButtonComponent(viewOptions)
      .setButtonText('Large')
      .onClick(() => this.setGridSize('large'));

    // Selection info
    const selectionInfo = gridHeader.createDiv('ink-selection-info');
    this.updateSelectionInfo(selectionInfo);

    // Image grid
    this.imageGrid = gridContainer.createDiv('ink-image-grid');
    this.imageGrid.addClass('ink-grid-medium'); // Default size
  }

  /**
   * Create pagination controls
   */
  private createPagination(): void {
    this.paginationContainer = this.modalContentEl.createDiv('ink-library-pagination');
  }

  /**
   * Create loading indicator
   */
  private createLoadingIndicator(): void {
    this.loadingIndicator = this.modalContentEl.createDiv('ink-loading-indicator');
    this.loadingIndicator.createEl('div', { text: 'Loading images...' });
    this.loadingIndicator.style.display = 'none';
  }

  /**
   * Load images from API
   */
  private async loadImages(): Promise<void> {
    if (this.isLoading) return;

    this.isLoading = true;
    this.showLoading();

    try {
      const response = await this.apiClient.request<ImageLibraryResponse>({
        method: 'GET',
        endpoint: '/api/v1/media/library',
        data: {
          page: this.currentPage,
          limit: this.pageSize,
          filter: this.currentFilter
        }
      });

      this.images = response.items;
      this.totalCount = response.totalCount;
      this.filteredImages = [...this.images];

      this.renderImages();
      this.updatePagination();

    } catch (error) {
      console.error('Failed to load images:', error);
      this.showError('Failed to load images');
    } finally {
      this.isLoading = false;
      this.hideLoading();
    }
  }

  /**
   * Apply current filters
   */
  private async applyFilters(): Promise<void> {
    this.currentPage = 0; // Reset to first page
    await this.loadImages();
  }

  /**
   * Render images in grid
   */
  private renderImages(): void {
    this.imageGrid.empty();

    if (this.filteredImages.length === 0) {
      const emptyState = this.imageGrid.createDiv('ink-empty-state');
      emptyState.createEl('p', { text: 'No images found' });
      return;
    }

    this.filteredImages.forEach(image => {
      const imageCard = this.createImageCard(image);
      this.imageGrid.appendChild(imageCard);
    });
  }

  /**
   * Create individual image card
   */
  private createImageCard(image: ImageLibraryItem): HTMLElement {
    const card = document.createElement('div');
    card.addClass('ink-image-card');
    
    if (this.selectedImages.has(image.chunkId)) {
      card.addClass('selected');
    }

    // Image container
    const imageContainer = card.createDiv('ink-image-container');
    
    const img = imageContainer.createEl('img');
    img.src = image.thumbnailUrl || image.imageUrl;
    img.alt = image.filename;
    img.loading = 'lazy';

    // Selection checkbox
    const checkbox = imageContainer.createEl('input');
    checkbox.type = 'checkbox';
    checkbox.checked = this.selectedImages.has(image.chunkId);
    checkbox.addClass('ink-image-checkbox');
    checkbox.addEventListener('change', () => {
      if (checkbox.checked) {
        this.selectedImages.add(image.chunkId);
        card.addClass('selected');
      } else {
        this.selectedImages.delete(image.chunkId);
        card.removeClass('selected');
      }
      this.updateSelectionInfo();
    });

    // Image info
    const infoContainer = card.createDiv('ink-image-info');
    
    const filename = infoContainer.createEl('div', { 
      text: image.filename,
      cls: 'ink-image-filename'
    });

    if (image.analysis?.description) {
      const description = infoContainer.createEl('div', {
        text: image.analysis.description,
        cls: 'ink-image-description'
      });
    }

    if (image.tags.length > 0) {
      const tagsContainer = infoContainer.createDiv('ink-image-tags');
      image.tags.forEach(tag => {
        const tagEl = tagsContainer.createEl('span', {
          text: tag,
          cls: 'ink-tag'
        });
      });
    }

    const metadata = infoContainer.createDiv('ink-image-metadata');
    metadata.createEl('span', { 
      text: new Date(image.createdAt).toLocaleDateString(),
      cls: 'ink-image-date'
    });
    metadata.createEl('span', { 
      text: image.storageType,
      cls: 'ink-storage-type'
    });

    // Click handlers
    img.addEventListener('click', () => {
      this.openImagePreview(image);
    });

    card.addEventListener('dblclick', () => {
      this.insertImageIntoEditor(image);
    });

    return card;
  }

  /**
   * Set grid size
   */
  private setGridSize(size: 'small' | 'medium' | 'large'): void {
    this.imageGrid.removeClass('ink-grid-small', 'ink-grid-medium', 'ink-grid-large');
    this.imageGrid.addClass(`ink-grid-${size}`);
  }

  /**
   * Update selection info
   */
  private updateSelectionInfo(container?: HTMLElement): void {
    if (!container) {
      const existing = this.modalContentEl.querySelector('.ink-selection-info');
      if (existing) container = existing as HTMLElement;
    }
    
    if (container) {
      container.empty();
      if (this.selectedImages.size > 0) {
        container.createEl('span', { 
          text: `${this.selectedImages.size} selected`,
          cls: 'ink-selection-count'
        });
      }
    }
  }

  /**
   * Update pagination
   */
  private updatePagination(): void {
    this.paginationContainer.empty();

    const totalPages = Math.ceil(this.totalCount / this.pageSize);
    
    if (totalPages <= 1) return;

    const pagination = this.paginationContainer.createDiv('ink-pagination');

    // Previous button
    const prevBtn = new ButtonComponent(pagination);
    prevBtn.setButtonText('Previous')
      .setDisabled(this.currentPage === 0)
      .onClick(() => {
        if (this.currentPage > 0) {
          this.currentPage--;
          this.loadImages();
        }
      });

    // Page info
    pagination.createEl('span', {
      text: `Page ${this.currentPage + 1} of ${totalPages}`,
      cls: 'ink-page-info'
    });

    // Next button
    const nextBtn = new ButtonComponent(pagination);
    nextBtn.setButtonText('Next')
      .setDisabled(this.currentPage >= totalPages - 1)
      .onClick(() => {
        if (this.currentPage < totalPages - 1) {
          this.currentPage++;
          this.loadImages();
        }
      });
  }

  /**
   * Show loading indicator
   */
  private showLoading(): void {
    this.loadingIndicator.style.display = 'block';
    this.imageGrid.style.opacity = '0.5';
  }

  /**
   * Hide loading indicator
   */
  private hideLoading(): void {
    this.loadingIndicator.style.display = 'none';
    this.imageGrid.style.opacity = '1';
  }

  /**
   * Show error message
   */
  private showError(message: string): void {
    this.imageGrid.empty();
    const errorEl = this.imageGrid.createDiv('ink-error-state');
    errorEl.createEl('p', { text: message });
  }

  /**
   * Open upload dialog
   */
  private openUploadDialog(): void {
    // This would open a file picker or drag-drop area
    const input = document.createElement('input');
    input.type = 'file';
    input.multiple = true;
    input.accept = 'image/*';
    input.addEventListener('change', (event) => {
      const files = (event.target as HTMLInputElement).files;
      if (files) {
        // Handle file upload
        console.log('Files selected for upload:', files);
      }
    });
    input.click();
  }

  /**
   * Open batch actions menu
   */
  private openBatchActionsMenu(): void {
    if (this.selectedImages.size === 0) {
      // Show notice that no images are selected
      return;
    }

    // Create context menu for batch actions
    const menu = document.createElement('div');
    menu.addClass('ink-batch-menu');
    
    // Add batch action options
    const actions = [
      { label: 'Add Tags', action: () => this.batchAddTags() },
      { label: 'Remove Tags', action: () => this.batchRemoveTags() },
      { label: 'Delete Images', action: () => this.batchDeleteImages() },
      { label: 'Export URLs', action: () => this.exportImageUrls() }
    ];

    actions.forEach(({ label, action }) => {
      const button = menu.createEl('button', { text: label });
      button.addEventListener('click', action);
    });

    // Position and show menu
    document.body.appendChild(menu);
  }

  /**
   * Open image preview
   */
  private openImagePreview(image: ImageLibraryItem): void {
    // Create full-screen image preview modal
    const previewModal = new Modal(this.app);
    previewModal.onOpen = () => {
      const { contentEl } = previewModal;
      contentEl.addClass('ink-image-preview-modal');
      
      const img = contentEl.createEl('img');
      img.src = image.imageUrl;
      img.alt = image.filename;
      img.style.maxWidth = '100%';
      img.style.maxHeight = '100%';
      
      // Add image details
      const details = contentEl.createDiv('ink-preview-details');
      details.createEl('h3', { text: image.filename });
      
      if (image.analysis?.description) {
        details.createEl('p', { text: image.analysis.description });
      }
      
      if (image.tags.length > 0) {
        const tagsEl = details.createEl('div');
        tagsEl.createEl('strong', { text: 'Tags: ' });
        tagsEl.createSpan(image.tags.join(', '));
      }
    };
    
    previewModal.open();
  }

  /**
   * Insert image into active editor
   */
  private insertImageIntoEditor(image: ImageLibraryItem): void {
    const activeView = this.app.workspace.getActiveViewOfType(MarkdownView);
    if (!activeView) return;

    const editor = activeView.editor;
    const cursor = editor.getCursor();
    
    const imageMarkdown = `![${image.filename}](${image.imageUrl})`;
    editor.replaceRange(imageMarkdown, cursor);
    
    this.close();
  }

  // Batch action methods
  private async batchAddTags(): Promise<void> {
    // Implementation for batch adding tags
  }

  private async batchRemoveTags(): Promise<void> {
    // Implementation for batch removing tags
  }

  private async batchDeleteImages(): Promise<void> {
    // Implementation for batch deleting images
  }

  private exportImageUrls(): void {
    const urls = Array.from(this.selectedImages).map(chunkId => {
      const image = this.images.find(img => img.chunkId === chunkId);
      return image?.imageUrl;
    }).filter(Boolean);

    navigator.clipboard.writeText(urls.join('\n'));
  }
}