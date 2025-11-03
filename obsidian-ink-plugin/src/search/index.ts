/**
 * Search Module Exports
 * Centralized exports for all search-related components
 */

export { SearchManager } from './SearchManager';
export { SearchView, SEARCH_VIEW_TYPE } from './SearchView';
export { SearchViewManager } from './SearchViewManager';
export { SearchCache } from '../cache/SearchCache';

// Re-export search-related types
export type {
  SearchQuery,
  SearchResult,
  SearchResultItem,
  SearchFilters,
  SearchHistory
} from '../types';

// Re-export search-related interfaces
export type {
  ISearchManager,
  ISearchView
} from '../interfaces';