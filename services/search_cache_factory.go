package services

import (
	"context"
	"database/sql"
	"fmt"
	"semantic-text-processor/models"
	"time"
)

// SearchCacheFactory creates and configures search cache related services
type SearchCacheFactory struct {
	db      *sql.DB
	config  *SearchCacheFactoryConfig
}

// SearchCacheFactoryConfig holds configuration for search cache factory
type SearchCacheFactoryConfig struct {
	// Database search cache configuration
	SearchCacheEnabled      bool          `json:"search_cache_enabled"`
	SearchCacheDefaultTTL   time.Duration `json:"search_cache_default_ttl"`
	SearchCacheMaxEntries   int           `json:"search_cache_max_entries"`
	SearchCacheCleanupInterval time.Duration `json:"search_cache_cleanup_interval"`
	
	// In-memory cache configuration
	InMemoryCacheEnabled    bool          `json:"in_memory_cache_enabled"`
	InMemoryCacheMaxSize    int           `json:"in_memory_cache_max_size"`
	InMemoryCacheCleanupInterval time.Duration `json:"in_memory_cache_cleanup_interval"`
	
	// Performance monitoring
	MonitoringEnabled       bool          `json:"monitoring_enabled"`
	StatsEnabled           bool          `json:"stats_enabled"`
	OptimizationEnabled    bool          `json:"optimization_enabled"`
}

// DefaultSearchCacheFactoryConfig returns default configuration
func DefaultSearchCacheFactoryConfig() *SearchCacheFactoryConfig {
	return &SearchCacheFactoryConfig{
		SearchCacheEnabled:      true,
		SearchCacheDefaultTTL:   15 * time.Minute,
		SearchCacheMaxEntries:   50000,
		SearchCacheCleanupInterval: 10 * time.Minute,
		
		InMemoryCacheEnabled:    true,
		InMemoryCacheMaxSize:    1000,
		InMemoryCacheCleanupInterval: 5 * time.Minute,
		
		MonitoringEnabled:       true,
		StatsEnabled:           true,
		OptimizationEnabled:    true,
	}
}

// NewSearchCacheFactory creates a new search cache factory
func NewSearchCacheFactory(db *sql.DB, config *SearchCacheFactoryConfig) *SearchCacheFactory {
	if config == nil {
		config = DefaultSearchCacheFactoryConfig()
	}
	
	return &SearchCacheFactory{
		db:     db,
		config: config,
	}
}

// CreateSearchCacheEnhancedService creates a fully configured search cache enhanced service
func (f *SearchCacheFactory) CreateSearchCacheEnhancedService(baseService UnifiedChunkService) UnifiedChunkService {
	// Create performance monitor
	var monitor QueryPerformanceMonitor
	if f.config.MonitoringEnabled {
		monitor = NewInMemoryPerformanceMonitor(100*time.Millisecond, 100)
	} else {
		monitor = &NoOpQueryPerformanceMonitor{}
	}
	
	// Create in-memory cache service
	var cacheService CacheService
	if f.config.InMemoryCacheEnabled {
		cacheService = NewInMemoryCache(
			f.config.InMemoryCacheMaxSize,
			f.config.InMemoryCacheCleanupInterval,
		)
	} else {
		cacheService = &NoOpCacheService{}
	}
	
	// Create database search cache
	var searchCache SearchCacheService
	if f.config.SearchCacheEnabled {
		searchCacheConfig := &SearchCacheConfig{
			DefaultTTL:          f.config.SearchCacheDefaultTTL,
			MaxCacheEntries:     f.config.SearchCacheMaxEntries,
			CleanupInterval:     f.config.SearchCacheCleanupInterval,
			HitCountThreshold:   5,
			OptimizationEnabled: f.config.OptimizationEnabled,
			StatsEnabled:        f.config.StatsEnabled,
		}
		searchCache = NewDatabaseSearchCache(f.db, searchCacheConfig, monitor)
	} else {
		searchCache = &NoOpSearchCacheService{}
	}
	
	// Create the enhanced service
	enhancedService := NewSearchCacheEnhancedUnifiedChunkService(
		baseService,
		searchCache,
		f.db,
		monitor,
	)
	
	// Wrap with in-memory caching if enabled
	if f.config.InMemoryCacheEnabled {
		queryCacheConfig := DefaultQueryCacheConfig()
		queryCacheManager := NewQueryCacheManager(cacheService, monitor, queryCacheConfig)
		enhancedService = NewCachedUnifiedChunkService(enhancedService, queryCacheManager)
	}
	
	return enhancedService
}

// GetSearchCacheService returns the database search cache service
func (f *SearchCacheFactory) GetSearchCacheService() SearchCacheService {
	if !f.config.SearchCacheEnabled {
		return &NoOpSearchCacheService{}
	}
	
	var monitor QueryPerformanceMonitor
	if f.config.MonitoringEnabled {
		monitor = NewInMemoryPerformanceMonitor(100*time.Millisecond, 100)
	} else {
		monitor = &NoOpQueryPerformanceMonitor{}
	}
	
	searchCacheConfig := &SearchCacheConfig{
		DefaultTTL:          f.config.SearchCacheDefaultTTL,
		MaxCacheEntries:     f.config.SearchCacheMaxEntries,
		CleanupInterval:     f.config.SearchCacheCleanupInterval,
		HitCountThreshold:   5,
		OptimizationEnabled: f.config.OptimizationEnabled,
		StatsEnabled:        f.config.StatsEnabled,
	}
	
	return NewDatabaseSearchCache(f.db, searchCacheConfig, monitor)
}

// NoOpSearchCacheService provides a no-op implementation for when search cache is disabled
type NoOpSearchCacheService struct{}

func (n *NoOpSearchCacheService) GetCachedSearch(ctx context.Context, queryParams map[string]interface{}) (*models.SearchCacheEntry, error) {
	return nil, nil // Always cache miss
}

func (n *NoOpSearchCacheService) SetCachedSearch(ctx context.Context, queryParams map[string]interface{}, chunkIDs []string, ttl time.Duration) error {
	return nil // No-op
}

func (n *NoOpSearchCacheService) InvalidateSearchCache(ctx context.Context, patterns []string) error {
	return nil // No-op
}

func (n *NoOpSearchCacheService) CleanupExpiredEntries(ctx context.Context) (int, error) {
	return 0, nil // No-op
}

func (n *NoOpSearchCacheService) GetCacheStats(ctx context.Context) (*SearchCacheStats, error) {
	return &SearchCacheStats{}, nil // Empty stats
}

func (n *NoOpSearchCacheService) GetOptimizationSuggestions(ctx context.Context) ([]OptimizationSuggestion, error) {
	return []OptimizationSuggestion{}, nil // No suggestions
}

func (n *NoOpSearchCacheService) UpdateHitCount(ctx context.Context, searchHash string) error {
	return nil // No-op
}

// NoOpQueryPerformanceMonitor provides a no-op implementation for when monitoring is disabled
type NoOpQueryPerformanceMonitor struct{}

func (n *NoOpQueryPerformanceMonitor) RecordQuery(queryType string, duration time.Duration, rowCount int) {
	// No-op
}

func (n *NoOpQueryPerformanceMonitor) GetQueryStats() QueryStatistics {
	return QueryStatistics{}
}

func (n *NoOpQueryPerformanceMonitor) GetSlowQueries(limit int) []SlowQueryRecord {
	return []SlowQueryRecord{}
}

func (n *NoOpQueryPerformanceMonitor) RecordSlowQuery(query string, duration time.Duration, params map[string]interface{}) {
	// No-op
}

// NoOpCacheService provides a no-op implementation for when caching is disabled
type NoOpCacheService struct{}

func (n *NoOpCacheService) Get(ctx context.Context, key string, dest interface{}) error {
	return fmt.Errorf("cache disabled")
}

func (n *NoOpCacheService) GetDirect(ctx context.Context, key string) (interface{}, bool) {
	return nil, false
}

func (n *NoOpCacheService) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	return nil // No-op
}

func (n *NoOpCacheService) Delete(ctx context.Context, key string) error {
	return nil // No-op
}

func (n *NoOpCacheService) DeletePattern(ctx context.Context, pattern string) error {
	return nil // No-op
}

func (n *NoOpCacheService) Clear(ctx context.Context) error {
	return nil // No-op
}

func (n *NoOpCacheService) GetStats() CacheStats {
	return CacheStats{}
}