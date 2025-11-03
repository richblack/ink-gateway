/**
 * Search View - User interface for search functionality
 * Provides search input, results display, and navigation
 */

import { 
  ItemView, 
  WorkspaceLeaf, 
  Setting, 
  ButtonComponent, 
  TextComponent,
  DropdownComponent,
  ToggleComponent,
  debounce
} from 'obsidian';
import { 
  SearchQuery, 
  SearchResult, 
  SearchResultItem, 
  SearchFilters,
  SearchHistory
} from '../types';
import { ISearchView } from '../interfaces';
import { SearchManager } from './SearchManager';

export const SEARCH_VIEW_TYPE = 'ink-search-view';

export class SearchView extends ItemView implements ISearchView {
  private searchManager: SearchManager;
  private currentResults: SearchResult | null = null;
  private searchHistory: SearchHistory[] = [];
  private onResultClickCallback?: (result: SearchResultItem) => void;
  
  // UI Elements
  public searchInput!: TextComponent;
  public searchTypeDropdown!: DropdownComponent;
  public tagInput!: TextComponent;
  private tagLogicToggle!: ToggleComponent;
  private searchButton!: ButtonComponent;
  private clearButton!: ButtonComponent;
  private resultsContainer!: HTMLElement;
  private statusContainer!: HTMLElement;
  private historyContainer!: HTMLElement;
  private filtersContainer!: HTMLElement;
  
  // Search state
  public currentQuery: SearchQuery = {};
  private isSearching = false;
  private showFilters = false;
  private showHistory = false;

  constructor(leaf: WorkspaceLeaf, searchManager: SearchManager) {
    super(leaf);
    this.searchManager = searchManager;
  }

  getViewType(): string {
    return SEARCH_VIEW_TYPE;
  }

  getDisplayText(): string {
    return 'Ink Search';
  }

  getIcon(): string {
    return 'search';
  }

  async onOpen(): Promise<void> {
    const container = this.containerEl.children[1];
    container.empty();
    container.addClass('ink-search-view');

    this.createSearchInterface(container);
    this.loadSearchHistory();
  }

  async onClose(): Promise<void> {
    // Cleanup if needed
  }

  /**
   * Create the main search interface
   */
  private createSearchInterface(container: Element): void {
    // Header with title and controls
    const header = container.createDiv('search-header');
    header.createEl('h3', { text: 'Semantic Search' });
    
    const headerControls = header.createDiv('header-controls');
    
    // Toggle filters button
    const filtersButton = new ButtonComponent(headerControls)
      .setButtonText('Filters')
      .setClass('mod-cta')
      .onClick(() => this.toggleFilters());
    
    // Toggle history button
    const historyButton = new ButtonComponent(headerControls)
      .setButtonText('History')
      .onClick(() => this.toggleHistory());

    // Search form
    const searchForm = container.createDiv('search-form');
    this.createSearchForm(searchForm);

    // Filters section (initially hidden)
    this.filtersContainer = container.createDiv('search-filters');
    this.filtersContainer.style.display = 'none';
    this.createFiltersSection(this.filtersContainer);

    // History section (initially hidden)
    this.historyContainer = container.createDiv('search-history');
    this.historyContainer.style.display = 'none';
    this.createHistorySection(this.historyContainer);

    // Status container
    this.statusContainer = container.createDiv('search-status');

    // Results container
    this.resultsContainer = container.createDiv('search-results');
  }

  /**
   * Create the main search form
   */
  private createSearchForm(container: Element): void {
    // Search input
    const searchInputContainer = container.createDiv('search-input-container');
    
    new Setting(searchInputContainer as HTMLElement)
      .setName('Search content')
      .setDesc('Enter text to search for')
      .addText(text => {
        this.searchInput = text;
        text.setPlaceholder('Search your notes...')
          .onChange(debounce((value: string) => {
            this.currentQuery.content = value;
            this.showSearchSuggestions(value);
          }, 300));
      });

    // Search type dropdown
    new Setting(container as HTMLElement)
      .setName('Search type')
      .setDesc('Choose the type of search to perform')
      .addDropdown(dropdown => {
        this.searchTypeDropdown = dropdown;
        dropdown.addOption('semantic', 'Semantic (AI-powered)')
          .addOption('exact', 'Exact match')
          .addOption('fuzzy', 'Fuzzy match')
          .addOption('', 'Hybrid (all types)')
          .setValue('')
          .onChange((value: string) => {
            this.currentQuery.searchType = value as any || undefined;
          });
      });

    // Tags input
    new Setting(container as HTMLElement)
      .setName('Tags')
      .setDesc('Search by tags (comma-separated)')
      .addText(text => {
        this.tagInput = text;
        text.setPlaceholder('tag1, tag2, tag3')
          .onChange((value: string) => {
            this.currentQuery.tags = value
              .split(',')
              .map(tag => tag.trim())
              .filter(tag => tag.length > 0);
          });
      });

    // Tag logic toggle
    new Setting(container as HTMLElement)
      .setName('Tag logic')
      .setDesc('Match ALL tags (AND) or ANY tags (OR)')
      .addToggle(toggle => {
        this.tagLogicToggle = toggle;
        toggle.setValue(false) // false = OR, true = AND
          .onChange((value: boolean) => {
            this.currentQuery.tagLogic = value ? 'AND' : 'OR';
          });
      });

    // Search buttons
    const buttonContainer = container.createDiv('search-buttons');
    
    this.searchButton = new ButtonComponent(buttonContainer)
      .setButtonText('Search')
      .setClass('mod-cta')
      .onClick(() => this.performSearch());

    this.clearButton = new ButtonComponent(buttonContainer)
      .setButtonText('Clear')
      .onClick(() => this.clearSearch());

    // Enter key support
    this.searchInput.inputEl.addEventListener('keydown', (event) => {
      if (event.key === 'Enter') {
        this.performSearch();
      }
    });
  }

  /**
   * Create the filters section
   */
  private createFiltersSection(container: Element): void {
    container.createEl('h4', { text: 'Advanced Filters' });

    // Date range filter
    const dateSection = container.createDiv('filter-section');
    dateSection.createEl('h5', { text: 'Date Range' });
    
    new Setting(dateSection as HTMLElement)
      .setName('From date')
      .addText(text => {
        text.inputEl.type = 'date';
        text.onChange((value: string) => {
          if (!this.currentQuery.filters) this.currentQuery.filters = {};
          if (!this.currentQuery.filters.dateRange) this.currentQuery.filters.dateRange = {} as any;
          this.currentQuery.filters.dateRange.start = new Date(value);
        });
      });

    new Setting(dateSection as HTMLElement)
      .setName('To date')
      .addText(text => {
        text.inputEl.type = 'date';
        text.onChange((value: string) => {
          if (!this.currentQuery.filters) this.currentQuery.filters = {};
          if (!this.currentQuery.filters.dateRange) this.currentQuery.filters.dateRange = {} as any;
          this.currentQuery.filters.dateRange.end = new Date(value);
        });
      });

    // File types filter
    new Setting(container as HTMLElement)
      .setName('File types')
      .setDesc('Filter by file extensions (comma-separated)')
      .addText(text => {
        text.setPlaceholder('md, txt, pdf')
          .onChange((value: string) => {
            if (!this.currentQuery.filters) this.currentQuery.filters = {};
            this.currentQuery.filters.fileTypes = value
              .split(',')
              .map(type => type.trim())
              .filter(type => type.length > 0);
          });
      });

    // Exclude tags filter
    new Setting(container as HTMLElement)
      .setName('Exclude tags')
      .setDesc('Exclude results with these tags')
      .addText(text => {
        text.setPlaceholder('draft, private, archive')
          .onChange((value: string) => {
            if (!this.currentQuery.filters) this.currentQuery.filters = {};
            this.currentQuery.filters.excludeTags = value
              .split(',')
              .map(tag => tag.trim())
              .filter(tag => tag.length > 0);
          });
      });

    // Minimum score filter
    new Setting(container as HTMLElement)
      .setName('Minimum relevance score')
      .setDesc('Only show results above this score (0-1)')
      .addSlider(slider => {
        slider.setLimits(0, 1, 0.1)
          .setValue(0)
          .setDynamicTooltip()
          .onChange((value: number) => {
            if (!this.currentQuery.filters) this.currentQuery.filters = {};
            this.currentQuery.filters.minScore = value;
          });
      });
  }

  /**
   * Create the history section
   */
  private createHistorySection(container: Element): void {
    container.createEl('h4', { text: 'Search History' });
    
    const historyList = container.createDiv('history-list');
    
    // Clear history button
    new ButtonComponent(container as HTMLElement)
      .setButtonText('Clear History')
      .setClass('mod-warning')
      .onClick(() => {
        this.searchManager.clearSearchHistory();
        this.loadSearchHistory();
        this.updateHistoryDisplay();
      });

    this.updateHistoryDisplay();
  }

  /**
   * Update the history display
   */
  public updateHistoryDisplay(): void {
    const historyList = this.historyContainer.querySelector('.history-list');
    if (!historyList) return;

    historyList.empty();
    
    this.searchHistory.slice(0, 10).forEach((entry, index) => {
      const historyItem = historyList.createDiv('history-item');
      
      const queryText = entry.query.content || 
        (entry.query.tags ? `Tags: ${entry.query.tags.join(', ')}` : 'Unknown query');
      
      const itemContent = historyItem.createDiv('history-item-content');
      itemContent.createSpan('history-query').textContent = queryText;
      itemContent.createSpan('history-meta').textContent = 
        `${entry.resultCount} results • ${entry.timestamp.toLocaleDateString()}`;
      
      historyItem.addEventListener('click', () => {
        this.loadQueryFromHistory(entry.query);
      });
    });
  }

  /**
   * Load a query from history
   */
  private loadQueryFromHistory(query: SearchQuery): void {
    this.currentQuery = { ...query };
    
    // Update UI elements
    if (query.content) {
      this.searchInput.setValue(query.content);
    }
    
    if (query.searchType) {
      this.searchTypeDropdown.setValue(query.searchType);
    }
    
    if (query.tags) {
      this.tagInput.setValue(query.tags.join(', '));
    }
    
    if (query.tagLogic) {
      this.tagLogicToggle.setValue(query.tagLogic === 'AND');
    }
  }

  /**
   * Show search suggestions based on input
   */
  private async showSearchSuggestions(input: string): Promise<void> {
    if (input.length < 2) return;
    
    try {
      const suggestions = await this.searchManager.getSearchSuggestions(input);
      // TODO: Display suggestions dropdown
      console.log('Search suggestions:', suggestions);
    } catch (error) {
      console.error('Failed to get search suggestions:', error);
    }
  }

  /**
   * Perform the search
   */
  public async performSearch(): Promise<void> {
    if (this.isSearching) return;
    
    try {
      this.isSearching = true;
      this.updateSearchStatus('Searching...', 'loading');
      this.searchButton.setDisabled(true);
      
      const result = await this.searchManager.performSearch(this.currentQuery);
      this.currentResults = result;
      
      this.displayResults(result);
      this.updateSearchStatus(
        `Found ${result.totalCount} results in ${result.searchTime}ms`,
        'success'
      );
      
    } catch (error) {
      console.error('Search failed:', error);
      this.updateSearchStatus(
        `Search failed: ${error instanceof Error ? error.message : 'Unknown error'}`,
        'error'
      );
    } finally {
      this.isSearching = false;
      this.searchButton.setDisabled(false);
    }
  }

  /**
   * Clear the search
   */
  public clearSearch(): void {
    this.currentQuery = {};
    this.currentResults = null;
    
    // Clear UI elements
    this.searchInput.setValue('');
    this.searchTypeDropdown.setValue('');
    this.tagInput.setValue('');
    this.tagLogicToggle.setValue(false);
    
    // Clear results
    this.resultsContainer.empty();
    this.statusContainer.empty();
  }

  /**
   * Display search results
   */
  displayResults(results: SearchResult): void {
    this.resultsContainer.empty();
    
    if (results.items.length === 0) {
      this.resultsContainer.createDiv('no-results').textContent = 'No results found';
      return;
    }
    
    // Results header
    const resultsHeader = this.resultsContainer.createDiv('results-header');
    resultsHeader.createSpan('results-count').textContent = 
      `${results.items.length} of ${results.totalCount} results`;
    
    if (results.cacheHit) {
      resultsHeader.createSpan('cache-indicator').textContent = '(cached)';
    }
    
    // Results list
    const resultsList = this.resultsContainer.createDiv('results-list');
    
    results.items.forEach((item, index) => {
      this.createResultItem(resultsList, item, index);
    });
  }

  /**
   * Create a single result item
   */
  private createResultItem(container: Element, item: SearchResultItem, index: number): void {
    const resultItem = container.createDiv('result-item');
    resultItem.setAttribute('data-index', index.toString());
    
    // Result header
    const resultHeader = resultItem.createDiv('result-header');
    
    const titleSpan = resultHeader.createSpan('result-title');
    titleSpan.textContent = this.getResultTitle(item);
    
    const scoreSpan = resultHeader.createSpan('result-score');
    scoreSpan.textContent = `${Math.round(item.score * 100)}%`;
    
    // Result content
    const resultContent = resultItem.createDiv('result-content');
    resultContent.textContent = item.context;
    
    // Result metadata
    const resultMeta = resultItem.createDiv('result-meta');
    resultMeta.createSpan('result-file').textContent = item.position.fileName;
    resultMeta.createSpan('result-line').textContent = `Line ${item.position.lineStart}`;
    
    if (item.chunk.tags.length > 0) {
      const tagsSpan = resultMeta.createSpan('result-tags');
      tagsSpan.textContent = item.chunk.tags.map(tag => `#${tag}`).join(' ');
    }
    
    // Click handler
    resultItem.addEventListener('click', () => {
      this.navigateToResult(item);
    });
    
    // Hover effects
    resultItem.addEventListener('mouseenter', () => {
      resultItem.addClass('result-item-hover');
    });
    
    resultItem.addEventListener('mouseleave', () => {
      resultItem.removeClass('result-item-hover');
    });
  }

  /**
   * Get a display title for the result
   */
  private getResultTitle(item: SearchResultItem): string {
    const content = item.chunk.contents;
    
    // Try to extract a title from the content
    const lines = content.split('\n');
    const firstLine = lines[0].trim();
    
    // If first line looks like a heading, use it
    if (firstLine.startsWith('#')) {
      return firstLine.replace(/^#+\s*/, '');
    }
    
    // Otherwise, use first 50 characters
    return firstLine.length > 50 ? firstLine.substring(0, 50) + '...' : firstLine;
  }

  /**
   * Navigate to a search result
   */
  navigateToResult(result: SearchResultItem): void {
    if (this.onResultClickCallback) {
      this.onResultClickCallback(result);
    } else {
      // Default navigation behavior
      console.log('Navigate to result:', result.position);
      // TODO: Implement actual navigation to file position
    }
  }

  /**
   * Set callback for result clicks
   */
  onResultClick(callback: (result: SearchResultItem) => void): void {
    this.onResultClickCallback = callback;
  }

  /**
   * Update search status
   */
  private updateSearchStatus(message: string, type: 'loading' | 'success' | 'error'): void {
    this.statusContainer.empty();
    
    const statusEl = this.statusContainer.createDiv(`search-status-${type}`);
    statusEl.textContent = message;
    
    if (type === 'loading') {
      statusEl.addClass('search-loading');
    }
  }

  /**
   * Toggle filters visibility
   */
  private toggleFilters(): void {
    this.showFilters = !this.showFilters;
    this.filtersContainer.style.display = this.showFilters ? 'block' : 'none';
  }

  /**
   * Toggle history visibility
   */
  private toggleHistory(): void {
    this.showHistory = !this.showHistory;
    this.historyContainer.style.display = this.showHistory ? 'block' : 'none';
    
    if (this.showHistory) {
      this.loadSearchHistory();
      this.updateHistoryDisplay();
    }
  }

  /**
   * Load search history from manager
   */
  public loadSearchHistory(): void {
    this.searchHistory = this.searchManager.getSearchHistory();
  }

  /**
   * Update results with new data
   */
  updateResults(results: SearchResult): void {
    this.currentResults = results;
    this.displayResults(results);
  }

  /**
   * Show the search view
   */
  show(): void {
    this.app.workspace.revealLeaf(this.leaf);
  }

  /**
   * Hide the search view
   */
  hide(): void {
    this.leaf.detach();
  }

  /**
   * Get current search results
   */
  getCurrentResults(): SearchResult | null {
    return this.currentResults;
  }

  /**
   * Get current search query
   */
  getCurrentQuery(): SearchQuery {
    return { ...this.currentQuery };
  }
}