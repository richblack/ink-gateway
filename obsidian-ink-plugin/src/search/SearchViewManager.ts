/**
 * Search View Manager - Manages search view lifecycle and integration
 * Handles view registration, navigation, and result interactions
 */

import { 
  App, 
  WorkspaceLeaf, 
  TFile,
  Notice,
  MarkdownView
} from 'obsidian';
import { SearchView, SEARCH_VIEW_TYPE } from './SearchView';
import { SearchManager } from './SearchManager';
import { SearchResultItem, Position } from '../types';

export class SearchViewManager {
  private app: App;
  private searchManager: SearchManager;
  private searchView: SearchView | null = null;

  constructor(app: App, searchManager: SearchManager) {
    this.app = app;
    this.searchManager = searchManager;
  }

  /**
   * Register the search view with Obsidian
   */
  registerView(): void {
    // Register view type with Obsidian
    (this.app.workspace as any).registerViewCreator?.(SEARCH_VIEW_TYPE, (leaf: any) => {
      const view = new SearchView(leaf, this.searchManager);
      this.searchView = view;
      
      // Set up result click handler
      view.onResultClick((result) => this.handleResultClick(result));
      
      return view;
    });
  }

  /**
   * Open or reveal the search view
   */
  async openSearchView(): Promise<SearchView> {
    const existing = this.app.workspace.getLeavesOfType(SEARCH_VIEW_TYPE);
    
    if (existing.length > 0) {
      // Reveal existing view
      this.app.workspace.revealLeaf(existing[0]);
      return existing[0].view as SearchView;
    }
    
    // Create new view
    const leaf = this.app.workspace.getRightLeaf(false);
    await leaf?.setViewState({
      type: SEARCH_VIEW_TYPE,
      active: true
    });
    
    const view = leaf?.view as SearchView;
    this.searchView = view;
    
    // Set up result click handler
    view.onResultClick((result) => this.handleResultClick(result));
    
    return view;
  }

  /**
   * Close the search view
   */
  closeSearchView(): void {
    const leaves = this.app.workspace.getLeavesOfType(SEARCH_VIEW_TYPE);
    leaves.forEach(leaf => leaf.detach());
    this.searchView = null;
  }

  /**
   * Get the current search view instance
   */
  getSearchView(): SearchView | null {
    return this.searchView;
  }

  /**
   * Handle clicking on a search result
   */
  private async handleResultClick(result: SearchResultItem): Promise<void> {
    try {
      await this.navigateToPosition(result.position);
    } catch (error) {
      console.error('Failed to navigate to result:', error);
      new Notice(`Failed to open file: ${result.position.fileName}`);
    }
  }

  /**
   * Navigate to a specific position in a file
   */
  private async navigateToPosition(position: Position): Promise<void> {
    // Find the file
    const file = this.app.vault.getAbstractFileByPath(position.fileName);
    
    if (!file || !(file instanceof TFile)) {
      throw new Error(`File not found: ${position.fileName}`);
    }

    // Open the file
    const leaf = this.app.workspace.getUnpinnedLeaf();
    await leaf.openFile(file);

    // Navigate to the specific position
    const view = leaf.view;
    if (view instanceof MarkdownView) {
      const editor = view.editor;
      
      // Set cursor position
      const cursor = {
        line: position.lineStart - 1, // Editor uses 0-based line numbers
        ch: position.charStart
      };
      
      editor.setCursor(cursor);
      
      // Scroll to the position
      editor.scrollIntoView({
        from: cursor,
        to: cursor
      }, true);
      
      // Highlight the relevant text if we have end position
      if (position.lineEnd > position.lineStart || position.charEnd > position.charStart) {
        const endCursor = {
          line: position.lineEnd - 1,
          ch: position.charEnd
        };
        
        editor.setSelection(cursor, endCursor);
      }
      
      // Focus the editor
      editor.focus();
    }
  }

  /**
   * Search for text and open results view
   */
  async searchAndShow(query: string, searchType?: 'semantic' | 'exact' | 'fuzzy'): Promise<void> {
    const view = await this.openSearchView();
    
    // Set the search query
    if (view.searchInput) {
      view.searchInput.setValue(query);
    }
    
    if (searchType && view.searchTypeDropdown) {
      view.searchTypeDropdown.setValue(searchType);
    }
    
    // Trigger search
    await view.performSearch();
  }

  /**
   * Search for selected text in the current editor
   */
  async searchSelectedText(): Promise<void> {
    const activeView = this.app.workspace.getActiveViewOfType(MarkdownView);
    
    if (!activeView) {
      new Notice('No active markdown editor');
      return;
    }
    
    const editor = activeView.editor;
    const selectedText = editor.getSelection();
    
    if (!selectedText.trim()) {
      new Notice('No text selected');
      return;
    }
    
    await this.searchAndShow(selectedText.trim(), 'semantic');
  }

  /**
   * Search for text under cursor
   */
  async searchWordUnderCursor(): Promise<void> {
    const activeView = this.app.workspace.getActiveViewOfType(MarkdownView);
    
    if (!activeView) {
      new Notice('No active markdown editor');
      return;
    }
    
    const editor = activeView.editor;
    const cursor = editor.getCursor();
    const line = editor.getLine(cursor.line);
    
    // Find word boundaries
    let start = cursor.ch;
    let end = cursor.ch;
    
    // Move start backward to word boundary
    while (start > 0 && /\w/.test(line[start - 1])) {
      start--;
    }
    
    // Move end forward to word boundary
    while (end < line.length && /\w/.test(line[end])) {
      end++;
    }
    
    const word = line.substring(start, end).trim();
    
    if (!word) {
      new Notice('No word under cursor');
      return;
    }
    
    await this.searchAndShow(word, 'semantic');
  }

  /**
   * Search by tags from the current file
   */
  async searchByCurrentFileTags(): Promise<void> {
    const activeView = this.app.workspace.getActiveViewOfType(MarkdownView);
    
    if (!activeView || !activeView.file) {
      new Notice('No active file');
      return;
    }
    
    const file = activeView.file;
    const cache = this.app.metadataCache.getFileCache(file);
    
    if (!cache || !cache.tags || cache.tags.length === 0) {
      new Notice('No tags found in current file');
      return;
    }
    
    const tags = cache.tags.map(tag => tag.tag.replace('#', ''));
    
    const view = await this.openSearchView();
    
    // Set the tags
    if (view.tagInput) {
      view.tagInput.setValue(tags.join(', '));
    }
    
    // Set search type to exact for tag search
    if (view.searchTypeDropdown) {
      view.searchTypeDropdown.setValue('exact');
    }
    
    // Trigger search
    await view.performSearch();
  }

  /**
   * Show search view with empty query (for manual search)
   */
  async showSearchView(): Promise<void> {
    await this.openSearchView();
  }

  /**
   * Toggle search view visibility
   */
  async toggleSearchView(): Promise<void> {
    const existing = this.app.workspace.getLeavesOfType(SEARCH_VIEW_TYPE);
    
    if (existing.length > 0) {
      this.closeSearchView();
    } else {
      await this.openSearchView();
    }
  }

  /**
   * Get search statistics for display
   */
  getSearchStats() {
    return this.searchManager.getSearchStats();
  }

  /**
   * Clear search history
   */
  clearSearchHistory(): void {
    this.searchManager.clearSearchHistory();
    
    if (this.searchView) {
      this.searchView.loadSearchHistory();
      this.searchView.updateHistoryDisplay();
    }
  }

  /**
   * Export search history
   */
  exportSearchHistory(): any {
    return {
      history: this.searchManager.getSearchHistory(),
      stats: this.searchManager.getSearchStats(),
      timestamp: new Date().toISOString()
    };
  }

  /**
   * Import search history
   */
  importSearchHistory(data: any): void {
    // This would require extending SearchManager to support import
    console.log('Import search history:', data);
    new Notice('Search history import not yet implemented');
  }

  /**
   * Cleanup resources
   */
  destroy(): void {
    this.closeSearchView();
    this.searchView = null;
  }
}