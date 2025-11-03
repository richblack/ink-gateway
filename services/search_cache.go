package services

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"semantic-text-processor/models"
	"time"

	"github.com/lib/pq"
)

// SearchCacheService provides database-backed search result caching
type SearchCacheService interface {
	// Cache operations
	GetCachedSearch(ctx context.Context, queryParams map[string]interface{}) (*models.SearchCacheEntry, error)
	SetCachedSearch(ctx context.Context, queryParams map[string]interface{}, chunkIDs []string, ttl time.Duration) error
	InvalidateSearchCache(ctx context.Context, patterns []string) error
	CleanupExpiredEntries(ctx context.Context) (int, error)
	
	// Statistics and optimization
	GetCacheStats(ctx context.Context) (*SearchCacheStats, error)
	GetOptimizationSuggestions(ctx context.Context) ([]OptimizationSuggestion, error)
	UpdateHitCount(ctx context.Context, searchHash string) error
}

// SearchCacheStats provides comprehensive cache performance metrics
type SearchCacheStats struct {
	TotalEntries    int     `json:"total_entries"`
	ExpiredEntries  int     `json:"expired_entries"`
	AverageHitCount float64 `json:"average_hit_count"`
	HitRate         float64 `json:"hit_rate"`
	CacheSize       int64   `json:"cache_size_bytes"`
	TopQueries      []QueryStats `json:"top_queries"`
	ExpirationStats ExpirationStats `json:"expiration_stats"`
}

// QueryStats represents statistics for individual queries
type QueryStats struct {
	SearchHash   string                 `json:"search_hash"`
	QueryParams  map[string]interface{} `json:"query_params"`
	HitCount     int                    `json:"hit_count"`
	ResultCount  int                    `json:"result_count"`
	LastAccessed time.Time              `json:"last_accessed"`
	CreatedAt    time.Time              `json:"created_at"`
}

// ExpirationStats provides insights into cache expiration patterns
type ExpirationStats struct {
	EntriesExpiringSoon int `json:"entries_expiring_soon"` // Within next hour
	EntriesExpiredToday int `json:"entries_expired_today"`
	AverageTTL          time.Duration `json:"average_ttl"`
}

// OptimizationSuggestion provides actionable cache optimization recommendations
type OptimizationSuggestion struct {
	Type        string                 `json:"type"`
	Priority    string                 `json:"priority"` // "high", "medium", "low"
	Description string                 `json:"description"`
	Action      string                 `json:"action"`
	Impact      string                 `json:"impact"`
	Data        map[string]interface{} `json:"data,omitempty"`
}

// DatabaseSearchCache implements SearchCacheService using PostgreSQL
type DatabaseSearchCache struct {
	db      *sql.DB
	config  *SearchCacheConfig
	monitor QueryPerformanceMonitor
}

// SearchCacheConfig holds configuration for database search cache
type SearchCacheConfig struct {
	DefaultTTL          time.Duration `json:"default_ttl"`
	MaxCacheEntries     int           `json:"max_cache_entries"`
	CleanupInterval     time.Duration `json:"cleanup_interval"`
	HitCountThreshold   int           `json:"hit_count_threshold"`
	OptimizationEnabled bool          `json:"optimization_enabled"`
	StatsEnabled        bool          `json:"stats_enabled"`
}

// DefaultSearchCacheConfig returns default search cache configuration
func DefaultSearchCacheConfig() *SearchCacheConfig {
	return &SearchCacheConfig{
		DefaultTTL:          15 * time.Minute,
		MaxCacheEntries:     50000,
		CleanupInterval:     10 * time.Minute,
		HitCountThreshold:   5,
		OptimizationEnabled: true,
		StatsEnabled:        true,
	}
}

// NewDatabaseSearchCache creates a new database search cache
func NewDatabaseSearchCache(db *sql.DB, config *SearchCacheConfig, monitor QueryPerformanceMonitor) SearchCacheService {
	if config == nil {
		config = DefaultSearchCacheConfig()
	}
	
	cache := &DatabaseSearchCache{
		db:      db,
		config:  config,
		monitor: monitor,
	}
	
	// Start background cleanup if enabled
	if config.CleanupInterval > 0 {
		go cache.startCleanupRoutine()
	}
	
	return cache
}

// generateSearchHash creates a deterministic hash for search parameters
func (dsc *DatabaseSearchCache) generateSearchHash(queryParams map[string]interface{}) string {
	// Normalize and sort parameters for consistent hashing
	normalized := make(map[string]interface{})
	for k, v := range queryParams {
		if v != nil {
			normalized[k] = v
		}
	}
	
	jsonBytes, err := json.Marshal(normalized)
	if err != nil {
		// Fallback to timestamp-based hash if JSON marshaling fails
		return fmt.Sprintf("fallback_%d", time.Now().UnixNano())
	}
	
	hash := sha256.Sum256(jsonBytes)
	return fmt.Sprintf("%x", hash[:16]) // Use first 16 bytes for shorter hash
}

// GetCachedSearch retrieves a cached search result
func (dsc *DatabaseSearchCache) GetCachedSearch(ctx context.Context, queryParams map[string]interface{}) (*models.SearchCacheEntry, error) {
	start := time.Now()
	searchHash := dsc.generateSearchHash(queryParams)
	
	query := `
		SELECT search_hash, query_params, chunk_ids, result_count, created_at, expires_at, hit_count
		FROM chunk_search_cache 
		WHERE search_hash = $1 AND expires_at > NOW()
	`
	
	var entry models.SearchCacheEntry
	var queryParamsJSON []byte
	var chunkIDsArray pq.StringArray
	
	err := dsc.db.QueryRowContext(ctx, query, searchHash).Scan(
		&entry.SearchHash,
		&queryParamsJSON,
		&chunkIDsArray,
		&entry.ResultCount,
		&entry.CreatedAt,
		&entry.ExpiresAt,
		&entry.HitCount,
	)
	
	duration := time.Since(start)
	
	if err != nil {
		if err == sql.ErrNoRows {
			dsc.monitor.RecordQuery("search_cache_miss", duration, 0)
			return nil, nil // Cache miss, not an error
		}
		dsc.monitor.RecordQuery("search_cache_error", duration, 0)
		return nil, fmt.Errorf("failed to get cached search: %w", err)
	}
	
	// Deserialize query parameters
	if err := json.Unmarshal(queryParamsJSON, &entry.QueryParams); err != nil {
		return nil, fmt.Errorf("failed to deserialize query params: %w", err)
	}
	
	entry.ChunkIDs = []string(chunkIDsArray)
	
	// Update hit count asynchronously
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		dsc.UpdateHitCount(ctx, searchHash)
	}()
	
	dsc.monitor.RecordQuery("search_cache_hit", duration, entry.ResultCount)
	return &entry, nil
}

// SetCachedSearch stores a search result in cache
func (dsc *DatabaseSearchCache) SetCachedSearch(ctx context.Context, queryParams map[string]interface{}, chunkIDs []string, ttl time.Duration) error {
	start := time.Now()
	searchHash := dsc.generateSearchHash(queryParams)
	
	// Serialize query parameters
	queryParamsJSON, err := json.Marshal(queryParams)
	if err != nil {
		return fmt.Errorf("failed to serialize query params: %w", err)
	}
	
	expiresAt := time.Now().Add(ttl)
	
	query := `
		INSERT INTO chunk_search_cache (search_hash, query_params, chunk_ids, result_count, expires_at, hit_count)
		VALUES ($1, $2, $3, $4, $5, 0)
		ON CONFLICT (search_hash) 
		DO UPDATE SET 
			query_params = EXCLUDED.query_params,
			chunk_ids = EXCLUDED.chunk_ids,
			result_count = EXCLUDED.result_count,
			expires_at = EXCLUDED.expires_at,
			created_at = NOW()
	`
	
	_, err = dsc.db.ExecContext(ctx, query, 
		searchHash, 
		queryParamsJSON, 
		pq.Array(chunkIDs), 
		len(chunkIDs), 
		expiresAt,
	)
	
	duration := time.Since(start)
	
	if err != nil {
		dsc.monitor.RecordQuery("search_cache_set_error", duration, 0)
		return fmt.Errorf("failed to set cached search: %w", err)
	}
	
	dsc.monitor.RecordQuery("search_cache_set", duration, len(chunkIDs))
	
	// Check if we need to cleanup old entries
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		dsc.cleanupIfNeeded(ctx)
	}()
	
	return nil
}

// UpdateHitCount increments the hit count for a cached entry
func (dsc *DatabaseSearchCache) UpdateHitCount(ctx context.Context, searchHash string) error {
	query := `UPDATE chunk_search_cache SET hit_count = hit_count + 1 WHERE search_hash = $1`
	_, err := dsc.db.ExecContext(ctx, query, searchHash)
	return err
}

// InvalidateSearchCache removes cache entries matching patterns
func (dsc *DatabaseSearchCache) InvalidateSearchCache(ctx context.Context, patterns []string) error {
	start := time.Now()
	
	for _, pattern := range patterns {
		var query string
		var args []interface{}
		
		if pattern == "*" {
			// Delete all entries
			query = "DELETE FROM chunk_search_cache"
		} else if len(pattern) > 0 && pattern[len(pattern)-1] == '*' {
			// Pattern matching with LIKE
			prefix := pattern[:len(pattern)-1]
			query = "DELETE FROM chunk_search_cache WHERE search_hash LIKE $1"
			args = []interface{}{prefix + "%"}
		} else {
			// Exact match
			query = "DELETE FROM chunk_search_cache WHERE search_hash = $1"
			args = []interface{}{pattern}
		}
		
		result, err := dsc.db.ExecContext(ctx, query, args...)
		if err != nil {
			return fmt.Errorf("failed to invalidate cache pattern %s: %w", pattern, err)
		}
		
		if rowsAffected, err := result.RowsAffected(); err == nil {
			dsc.monitor.RecordQuery("search_cache_invalidate", time.Since(start), int(rowsAffected))
		}
	}
	
	return nil
}

// CleanupExpiredEntries removes expired cache entries
func (dsc *DatabaseSearchCache) CleanupExpiredEntries(ctx context.Context) (int, error) {
	start := time.Now()
	
	query := "DELETE FROM chunk_search_cache WHERE expires_at < NOW()"
	result, err := dsc.db.ExecContext(ctx, query)
	if err != nil {
		dsc.monitor.RecordQuery("search_cache_cleanup_error", time.Since(start), 0)
		return 0, fmt.Errorf("failed to cleanup expired entries: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get affected rows: %w", err)
	}
	
	dsc.monitor.RecordQuery("search_cache_cleanup", time.Since(start), int(rowsAffected))
	return int(rowsAffected), nil
}

// GetCacheStats retrieves comprehensive cache statistics
func (dsc *DatabaseSearchCache) GetCacheStats(ctx context.Context) (*SearchCacheStats, error) {
	if !dsc.config.StatsEnabled {
		return &SearchCacheStats{}, nil
	}
	
	stats := &SearchCacheStats{}
	
	// Get basic stats
	basicQuery := `
		SELECT 
			COUNT(*) as total_entries,
			COUNT(*) FILTER (WHERE expires_at < NOW()) as expired_entries,
			AVG(hit_count) as avg_hit_count,
			SUM(octet_length(query_params::text) + array_length(chunk_ids, 1) * 36) as cache_size
		FROM chunk_search_cache
	`
	
	var avgHitCount sql.NullFloat64
	var cacheSize sql.NullInt64
	
	err := dsc.db.QueryRowContext(ctx, basicQuery).Scan(
		&stats.TotalEntries,
		&stats.ExpiredEntries,
		&avgHitCount,
		&cacheSize,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get basic cache stats: %w", err)
	}
	
	if avgHitCount.Valid {
		stats.AverageHitCount = avgHitCount.Float64
	}
	if cacheSize.Valid {
		stats.CacheSize = cacheSize.Int64
	}
	
	// Calculate hit rate (simplified - would need more sophisticated tracking in production)
	if stats.TotalEntries > 0 {
		stats.HitRate = float64(stats.TotalEntries-stats.ExpiredEntries) / float64(stats.TotalEntries)
	}
	
	// Get top queries
	topQueriesQuery := `
		SELECT search_hash, query_params, hit_count, result_count, created_at
		FROM chunk_search_cache 
		WHERE expires_at > NOW()
		ORDER BY hit_count DESC 
		LIMIT 10
	`
	
	rows, err := dsc.db.QueryContext(ctx, topQueriesQuery)
	if err != nil {
		return stats, nil // Return partial stats if top queries fail
	}
	defer rows.Close()
	
	for rows.Next() {
		var queryStats QueryStats
		var queryParamsJSON []byte
		
		err := rows.Scan(
			&queryStats.SearchHash,
			&queryParamsJSON,
			&queryStats.HitCount,
			&queryStats.ResultCount,
			&queryStats.CreatedAt,
		)
		if err != nil {
			continue
		}
		
		if err := json.Unmarshal(queryParamsJSON, &queryStats.QueryParams); err != nil {
			continue
		}
		
		queryStats.LastAccessed = queryStats.CreatedAt // Simplified
		stats.TopQueries = append(stats.TopQueries, queryStats)
	}
	
	// Get expiration stats
	expirationQuery := `
		SELECT 
			COUNT(*) FILTER (WHERE expires_at BETWEEN NOW() AND NOW() + INTERVAL '1 hour') as expiring_soon,
			COUNT(*) FILTER (WHERE expires_at BETWEEN NOW() - INTERVAL '1 day' AND NOW()) as expired_today,
			AVG(EXTRACT(EPOCH FROM (expires_at - created_at))) as avg_ttl_seconds
		FROM chunk_search_cache
	`
	
	var avgTTLSeconds sql.NullFloat64
	err = dsc.db.QueryRowContext(ctx, expirationQuery).Scan(
		&stats.ExpirationStats.EntriesExpiringSoon,
		&stats.ExpirationStats.EntriesExpiredToday,
		&avgTTLSeconds,
	)
	if err == nil && avgTTLSeconds.Valid {
		stats.ExpirationStats.AverageTTL = time.Duration(avgTTLSeconds.Float64) * time.Second
	}
	
	return stats, nil
}

// GetOptimizationSuggestions provides actionable optimization recommendations
func (dsc *DatabaseSearchCache) GetOptimizationSuggestions(ctx context.Context) ([]OptimizationSuggestion, error) {
	if !dsc.config.OptimizationEnabled {
		return []OptimizationSuggestion{}, nil
	}
	
	suggestions := []OptimizationSuggestion{}
	
	// Get cache stats for analysis
	stats, err := dsc.GetCacheStats(ctx)
	if err != nil {
		return suggestions, err
	}
	
	// Suggestion 1: High cache miss rate
	if stats.HitRate < 0.5 {
		suggestions = append(suggestions, OptimizationSuggestion{
			Type:        "hit_rate",
			Priority:    "high",
			Description: fmt.Sprintf("Cache hit rate is low (%.1f%%)", stats.HitRate*100),
			Action:      "Consider increasing TTL for frequently accessed queries or reviewing query patterns",
			Impact:      "Improving hit rate can significantly reduce database load",
			Data: map[string]interface{}{
				"current_hit_rate": stats.HitRate,
				"target_hit_rate":  0.7,
			},
		})
	}
	
	// Suggestion 2: Too many expired entries
	if stats.ExpiredEntries > stats.TotalEntries/4 {
		suggestions = append(suggestions, OptimizationSuggestion{
			Type:        "expiration",
			Priority:    "medium",
			Description: fmt.Sprintf("High number of expired entries (%d out of %d)", stats.ExpiredEntries, stats.TotalEntries),
			Action:      "Run cleanup more frequently or adjust TTL values",
			Impact:      "Reducing expired entries improves cache efficiency",
			Data: map[string]interface{}{
				"expired_entries": stats.ExpiredEntries,
				"total_entries":   stats.TotalEntries,
			},
		})
	}
	
	// Suggestion 3: Cache size optimization
	if stats.CacheSize > 100*1024*1024 { // 100MB
		suggestions = append(suggestions, OptimizationSuggestion{
			Type:        "size",
			Priority:    "medium",
			Description: fmt.Sprintf("Cache size is large (%.1f MB)", float64(stats.CacheSize)/(1024*1024)),
			Action:      "Consider reducing TTL or implementing more aggressive cleanup",
			Impact:      "Smaller cache size improves memory usage and query performance",
			Data: map[string]interface{}{
				"cache_size_mb": float64(stats.CacheSize) / (1024 * 1024),
				"max_recommended_mb": 50,
			},
		})
	}
	
	// Suggestion 4: Underutilized cache entries
	lowHitQueries := 0
	for _, query := range stats.TopQueries {
		if query.HitCount < dsc.config.HitCountThreshold {
			lowHitQueries++
		}
	}
	
	if lowHitQueries > len(stats.TopQueries)/2 {
		suggestions = append(suggestions, OptimizationSuggestion{
			Type:        "utilization",
			Priority:    "low",
			Description: fmt.Sprintf("%d queries have low hit counts", lowHitQueries),
			Action:      "Review query patterns and consider shorter TTL for infrequently accessed queries",
			Impact:      "Better cache utilization improves overall performance",
			Data: map[string]interface{}{
				"low_hit_queries":    lowHitQueries,
				"hit_count_threshold": dsc.config.HitCountThreshold,
			},
		})
	}
	
	return suggestions, nil
}

// cleanupIfNeeded performs cleanup if cache size exceeds limits
func (dsc *DatabaseSearchCache) cleanupIfNeeded(ctx context.Context) {
	// Check current entry count
	var count int
	err := dsc.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM chunk_search_cache").Scan(&count)
	if err != nil {
		return
	}
	
	if count > dsc.config.MaxCacheEntries {
		// Remove oldest entries beyond the limit
		query := `
			DELETE FROM chunk_search_cache 
			WHERE search_hash IN (
				SELECT search_hash FROM chunk_search_cache 
				ORDER BY created_at ASC 
				LIMIT $1
			)
		`
		excess := count - dsc.config.MaxCacheEntries
		dsc.db.ExecContext(ctx, query, excess)
	}
}

// startCleanupRoutine runs periodic cleanup of expired entries
func (dsc *DatabaseSearchCache) startCleanupRoutine() {
	ticker := time.NewTicker(dsc.config.CleanupInterval)
	defer ticker.Stop()
	
	for range ticker.C {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		dsc.CleanupExpiredEntries(ctx)
		cancel()
	}
}