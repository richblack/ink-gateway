package services

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"semantic-text-processor/models"
	"sort"
	"strings"
	"time"
)

// QueryCacheManager manages query result caching with intelligent cache key generation
type QueryCacheManager struct {
	cache   CacheService
	monitor QueryPerformanceMonitor
	config  *QueryCacheConfig
}

// QueryCacheConfig holds configuration for query caching
type QueryCacheConfig struct {
	DefaultTTL      time.Duration `json:"default_ttl"`
	TagQueryTTL     time.Duration `json:"tag_query_ttl"`
	HierarchyTTL    time.Duration `json:"hierarchy_ttl"`
	SearchTTL       time.Duration `json:"search_ttl"`
	MaxCacheSize    int           `json:"max_cache_size"`
	Enabled         bool          `json:"enabled"`
}

// DefaultQueryCacheConfig returns default query cache configuration
func DefaultQueryCacheConfig() *QueryCacheConfig {
	return &QueryCacheConfig{
		DefaultTTL:   5 * time.Minute,
		TagQueryTTL:  10 * time.Minute,
		HierarchyTTL: 15 * time.Minute,
		SearchTTL:    3 * time.Minute,
		MaxCacheSize: 10000,
		Enabled:      true,
	}
}

// NewQueryCacheManager creates a new query cache manager
func NewQueryCacheManager(cache CacheService, monitor QueryPerformanceMonitor, config *QueryCacheConfig) *QueryCacheManager {
	if config == nil {
		config = DefaultQueryCacheConfig()
	}
	
	return &QueryCacheManager{
		cache:   cache,
		monitor: monitor,
		config:  config,
	}
}

// CacheKey represents a structured cache key
type CacheKey struct {
	Type       string                 `json:"type"`
	Identifier string                 `json:"identifier"`
	Params     map[string]interface{} `json:"params,omitempty"`
	Version    string                 `json:"version,omitempty"`
}

// GenerateCacheKey creates a deterministic cache key from parameters
func (qcm *QueryCacheManager) GenerateCacheKey(keyType, identifier string, params map[string]interface{}) string {
	if !qcm.config.Enabled {
		return ""
	}

	key := CacheKey{
		Type:       keyType,
		Identifier: identifier,
		Params:     params,
		Version:    "v1", // For future cache invalidation when schema changes
	}

	// Sort parameters for consistent key generation
	if params != nil {
		sortedParams := make(map[string]interface{})
		keys := make([]string, 0, len(params))
		for k := range params {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		
		for _, k := range keys {
			sortedParams[k] = params[k]
		}
		key.Params = sortedParams
	}

	// Serialize to JSON for hashing
	jsonBytes, err := json.Marshal(key)
	if err != nil {
		// Fallback to simple key if JSON marshaling fails
		return fmt.Sprintf("%s:%s", keyType, identifier)
	}

	// Generate SHA256 hash for consistent, collision-resistant keys
	hash := sha256.Sum256(jsonBytes)
	return fmt.Sprintf("qcache:%x", hash[:16]) // Use first 16 bytes for shorter keys
}

// GetCachedResult retrieves a cached result if available
func (qcm *QueryCacheManager) GetCachedResult(ctx context.Context, cacheKey string, dest interface{}) (bool, error) {
	if !qcm.config.Enabled || cacheKey == "" {
		return false, nil
	}

	start := time.Now()
	err := qcm.cache.Get(ctx, cacheKey, dest)
	duration := time.Since(start)

	if err != nil {
		qcm.monitor.RecordQuery("cache_miss", duration, 0)
		return false, nil
	}

	qcm.monitor.RecordQuery("cache_hit", duration, 1)
	return true, nil
}

// SetCachedResult stores a result in cache with appropriate TTL
func (qcm *QueryCacheManager) SetCachedResult(ctx context.Context, cacheKey string, result interface{}, queryType string) error {
	if !qcm.config.Enabled || cacheKey == "" {
		return nil
	}

	ttl := qcm.getTTLForQueryType(queryType)
	return qcm.cache.Set(ctx, cacheKey, result, ttl)
}

// InvalidateCachePatterns invalidates cache entries matching patterns
func (qcm *QueryCacheManager) InvalidateCachePatterns(ctx context.Context, patterns []string) error {
	if !qcm.config.Enabled {
		return nil
	}

	for _, pattern := range patterns {
		if err := qcm.cache.DeletePattern(ctx, pattern); err != nil {
			return fmt.Errorf("failed to invalidate cache pattern %s: %w", pattern, err)
		}
	}
	return nil
}

// ExecuteWithCache executes a query function with caching
func (qcm *QueryCacheManager) ExecuteWithCache(
	ctx context.Context,
	cacheKey string,
	queryType string,
	queryFunc func() (interface{}, error),
	dest interface{},
) error {
	// Try to get from cache first
	if hit, err := qcm.GetCachedResult(ctx, cacheKey, dest); err == nil && hit {
		return nil
	}

	// Execute the query
	start := time.Now()
	result, err := queryFunc()
	duration := time.Since(start)

	if err != nil {
		qcm.monitor.RecordQuery(queryType+"_error", duration, 0)
		return err
	}

	// Record successful query
	qcm.monitor.RecordQuery(queryType, duration, qcm.getResultCount(result))

	// Cache the result
	if err := qcm.SetCachedResult(ctx, cacheKey, result, queryType); err != nil {
		// Log cache error but don't fail the query
		qcm.monitor.RecordQuery("cache_set_error", 0, 0)
	}

	// Copy result to destination
	return qcm.copyResult(result, dest)
}

// getTTLForQueryType returns appropriate TTL based on query type
func (qcm *QueryCacheManager) getTTLForQueryType(queryType string) time.Duration {
	switch {
	case strings.Contains(queryType, "tag"):
		return qcm.config.TagQueryTTL
	case strings.Contains(queryType, "hierarchy") || strings.Contains(queryType, "children") || strings.Contains(queryType, "ancestors"):
		return qcm.config.HierarchyTTL
	case strings.Contains(queryType, "search"):
		return qcm.config.SearchTTL
	default:
		return qcm.config.DefaultTTL
	}
}

// getResultCount extracts count from various result types for monitoring
func (qcm *QueryCacheManager) getResultCount(result interface{}) int {
	switch r := result.(type) {
	case []models.UnifiedChunkRecord:
		return len(r)
	case *models.SearchResult:
		return len(r.Chunks)
	case []models.ChunkTagRelation:
		return len(r)
	case []models.ChunkHierarchyRelation:
		return len(r)
	default:
		return 1
	}
}

// copyResult copies the result to the destination interface
func (qcm *QueryCacheManager) copyResult(src, dest interface{}) error {
	// Use JSON marshaling/unmarshaling for deep copy
	jsonBytes, err := json.Marshal(src)
	if err != nil {
		return fmt.Errorf("failed to marshal result: %w", err)
	}
	
	return json.Unmarshal(jsonBytes, dest)
}

// CachedUnifiedChunkService wraps UnifiedChunkService with caching
type CachedUnifiedChunkService struct {
	base        UnifiedChunkService
	cacheManager *QueryCacheManager
}

// NewCachedUnifiedChunkService creates a cached wrapper for UnifiedChunkService
func NewCachedUnifiedChunkService(base UnifiedChunkService, cacheManager *QueryCacheManager) UnifiedChunkService {
	return &CachedUnifiedChunkService{
		base:         base,
		cacheManager: cacheManager,
	}
}

// GetChunk retrieves a chunk with caching
func (s *CachedUnifiedChunkService) GetChunk(ctx context.Context, chunkID string) (*models.UnifiedChunkRecord, error) {
	cacheKey := s.cacheManager.GenerateCacheKey("chunk", chunkID, nil)
	
	var result *models.UnifiedChunkRecord
	err := s.cacheManager.ExecuteWithCache(ctx, cacheKey, "get_chunk", func() (interface{}, error) {
		return s.base.GetChunk(ctx, chunkID)
	}, &result)
	
	return result, err
}

// GetChunksByTag retrieves chunks by tag with caching
func (s *CachedUnifiedChunkService) GetChunksByTag(ctx context.Context, tagChunkID string) ([]models.UnifiedChunkRecord, error) {
	cacheKey := s.cacheManager.GenerateCacheKey("chunks_by_tag", tagChunkID, nil)
	
	var result []models.UnifiedChunkRecord
	err := s.cacheManager.ExecuteWithCache(ctx, cacheKey, "get_chunks_by_tag", func() (interface{}, error) {
		return s.base.GetChunksByTag(ctx, tagChunkID)
	}, &result)
	
	return result, err
}

// GetChunksByTags retrieves chunks by multiple tags with caching
func (s *CachedUnifiedChunkService) GetChunksByTags(ctx context.Context, tagChunkIDs []string, matchType string) ([]models.UnifiedChunkRecord, error) {
	params := map[string]interface{}{
		"tags":       tagChunkIDs,
		"match_type": matchType,
	}
	cacheKey := s.cacheManager.GenerateCacheKey("chunks_by_tags", "", params)
	
	var result []models.UnifiedChunkRecord
	err := s.cacheManager.ExecuteWithCache(ctx, cacheKey, "get_chunks_by_tags", func() (interface{}, error) {
		return s.base.GetChunksByTags(ctx, tagChunkIDs, matchType)
	}, &result)
	
	return result, err
}

// GetChunkTags retrieves chunk tags with caching
func (s *CachedUnifiedChunkService) GetChunkTags(ctx context.Context, chunkID string) ([]models.UnifiedChunkRecord, error) {
	cacheKey := s.cacheManager.GenerateCacheKey("chunk_tags", chunkID, nil)
	
	var result []models.UnifiedChunkRecord
	err := s.cacheManager.ExecuteWithCache(ctx, cacheKey, "get_chunk_tags", func() (interface{}, error) {
		return s.base.GetChunkTags(ctx, chunkID)
	}, &result)
	
	return result, err
}

// GetChildren retrieves children with caching
func (s *CachedUnifiedChunkService) GetChildren(ctx context.Context, parentChunkID string) ([]models.UnifiedChunkRecord, error) {
	cacheKey := s.cacheManager.GenerateCacheKey("chunk_children", parentChunkID, nil)
	
	var result []models.UnifiedChunkRecord
	err := s.cacheManager.ExecuteWithCache(ctx, cacheKey, "get_children", func() (interface{}, error) {
		return s.base.GetChildren(ctx, parentChunkID)
	}, &result)
	
	return result, err
}

// GetDescendants retrieves descendants with caching
func (s *CachedUnifiedChunkService) GetDescendants(ctx context.Context, ancestorChunkID string, maxDepth int) ([]models.UnifiedChunkRecord, error) {
	params := map[string]interface{}{
		"max_depth": maxDepth,
	}
	cacheKey := s.cacheManager.GenerateCacheKey("chunk_descendants", ancestorChunkID, params)
	
	var result []models.UnifiedChunkRecord
	err := s.cacheManager.ExecuteWithCache(ctx, cacheKey, "get_descendants", func() (interface{}, error) {
		return s.base.GetDescendants(ctx, ancestorChunkID, maxDepth)
	}, &result)
	
	return result, err
}

// GetAncestors retrieves ancestors with caching
func (s *CachedUnifiedChunkService) GetAncestors(ctx context.Context, chunkID string) ([]models.UnifiedChunkRecord, error) {
	cacheKey := s.cacheManager.GenerateCacheKey("chunk_ancestors", chunkID, nil)
	
	var result []models.UnifiedChunkRecord
	err := s.cacheManager.ExecuteWithCache(ctx, cacheKey, "get_ancestors", func() (interface{}, error) {
		return s.base.GetAncestors(ctx, chunkID)
	}, &result)
	
	return result, err
}

// SearchChunks performs search with caching
func (s *CachedUnifiedChunkService) SearchChunks(ctx context.Context, query *models.SearchQuery) (*models.SearchResult, error) {
	params := map[string]interface{}{
		"content":     query.Content,
		"tags":        query.Tags,
		"tag_logic":   query.TagLogic,
		"is_page":     query.IsPage,
		"is_tag":      query.IsTag,
		"is_template": query.IsTemplate,
		"is_slot":     query.IsSlot,
		"parent":      query.Parent,
		"page":        query.Page,
		"metadata":    query.Metadata,
		"limit":       query.Limit,
		"offset":      query.Offset,
	}
	cacheKey := s.cacheManager.GenerateCacheKey("search_chunks", "", params)
	
	var result *models.SearchResult
	err := s.cacheManager.ExecuteWithCache(ctx, cacheKey, "search_chunks", func() (interface{}, error) {
		return s.base.SearchChunks(ctx, query)
	}, &result)
	
	if result != nil {
		result.CacheHit = true // Mark as cache hit if we got here via cache
	}
	
	return result, err
}

// SearchByContent performs content search with caching
func (s *CachedUnifiedChunkService) SearchByContent(ctx context.Context, content string, filters map[string]interface{}) ([]models.UnifiedChunkRecord, error) {
	params := map[string]interface{}{
		"content": content,
		"filters": filters,
	}
	cacheKey := s.cacheManager.GenerateCacheKey("search_by_content", "", params)
	
	var result []models.UnifiedChunkRecord
	err := s.cacheManager.ExecuteWithCache(ctx, cacheKey, "search_by_content", func() (interface{}, error) {
		return s.base.SearchByContent(ctx, content, filters)
	}, &result)
	
	return result, err
}

// Write operations - these invalidate caches and delegate to base service

// CreateChunk creates a chunk and invalidates related caches
func (s *CachedUnifiedChunkService) CreateChunk(ctx context.Context, chunk *models.UnifiedChunkRecord) error {
	err := s.base.CreateChunk(ctx, chunk)
	if err != nil {
		return err
	}
	
	// Invalidate related caches
	patterns := s.getInvalidationPatterns(chunk.ChunkID, chunk.Tags, chunk.Parent)
	s.cacheManager.InvalidateCachePatterns(ctx, patterns)
	
	return nil
}

// UpdateChunk updates a chunk and invalidates related caches
func (s *CachedUnifiedChunkService) UpdateChunk(ctx context.Context, chunk *models.UnifiedChunkRecord) error {
	err := s.base.UpdateChunk(ctx, chunk)
	if err != nil {
		return err
	}
	
	// Invalidate related caches
	patterns := s.getInvalidationPatterns(chunk.ChunkID, chunk.Tags, chunk.Parent)
	s.cacheManager.InvalidateCachePatterns(ctx, patterns)
	
	return nil
}

// DeleteChunk deletes a chunk and invalidates related caches
func (s *CachedUnifiedChunkService) DeleteChunk(ctx context.Context, chunkID string) error {
	err := s.base.DeleteChunk(ctx, chunkID)
	if err != nil {
		return err
	}
	
	// Invalidate related caches
	patterns := s.getInvalidationPatterns(chunkID, nil, nil)
	s.cacheManager.InvalidateCachePatterns(ctx, patterns)
	
	return nil
}

// BatchCreateChunks creates multiple chunks and invalidates related caches
func (s *CachedUnifiedChunkService) BatchCreateChunks(ctx context.Context, chunks []models.UnifiedChunkRecord) error {
	err := s.base.BatchCreateChunks(ctx, chunks)
	if err != nil {
		return err
	}
	
	// Invalidate caches for all chunks
	patterns := []string{"qcache:*"} // Invalidate all query caches for batch operations
	s.cacheManager.InvalidateCachePatterns(ctx, patterns)
	
	return nil
}

// BatchUpdateChunks updates multiple chunks and invalidates related caches
func (s *CachedUnifiedChunkService) BatchUpdateChunks(ctx context.Context, chunks []models.UnifiedChunkRecord) error {
	err := s.base.BatchUpdateChunks(ctx, chunks)
	if err != nil {
		return err
	}
	
	// Invalidate caches for all chunks
	patterns := []string{"qcache:*"} // Invalidate all query caches for batch operations
	s.cacheManager.InvalidateCachePatterns(ctx, patterns)
	
	return nil
}

// AddTags adds tags and invalidates related caches
func (s *CachedUnifiedChunkService) AddTags(ctx context.Context, chunkID string, tagChunkIDs []string) error {
	err := s.base.AddTags(ctx, chunkID, tagChunkIDs)
	if err != nil {
		return err
	}
	
	// Invalidate tag-related caches
	patterns := s.getTagInvalidationPatterns(chunkID, tagChunkIDs)
	s.cacheManager.InvalidateCachePatterns(ctx, patterns)
	
	return nil
}

// RemoveTags removes tags and invalidates related caches
func (s *CachedUnifiedChunkService) RemoveTags(ctx context.Context, chunkID string, tagChunkIDs []string) error {
	err := s.base.RemoveTags(ctx, chunkID, tagChunkIDs)
	if err != nil {
		return err
	}
	
	// Invalidate tag-related caches
	patterns := s.getTagInvalidationPatterns(chunkID, tagChunkIDs)
	s.cacheManager.InvalidateCachePatterns(ctx, patterns)
	
	return nil
}

// MoveChunk moves a chunk and invalidates related caches
func (s *CachedUnifiedChunkService) MoveChunk(ctx context.Context, chunkID, newParentID string) error {
	err := s.base.MoveChunk(ctx, chunkID, newParentID)
	if err != nil {
		return err
	}
	
	// Invalidate hierarchy-related caches
	patterns := s.getHierarchyInvalidationPatterns(chunkID, newParentID)
	s.cacheManager.InvalidateCachePatterns(ctx, patterns)
	
	return nil
}

// Helper methods for cache invalidation patterns

func (s *CachedUnifiedChunkService) getInvalidationPatterns(chunkID string, tags []string, parent *string) []string {
	patterns := []string{
		"qcache:*", // For now, invalidate all cache entries to ensure correctness
	}
	
	return patterns
}

func (s *CachedUnifiedChunkService) getTagInvalidationPatterns(chunkID string, tagChunkIDs []string) []string {
	patterns := []string{
		"qcache:*chunk*" + chunkID + "*",
		"qcache:*tag*",
		"qcache:*search*",
	}
	
	for _, tagID := range tagChunkIDs {
		patterns = append(patterns, "qcache:*"+tagID+"*")
	}
	
	return patterns
}

func (s *CachedUnifiedChunkService) getHierarchyInvalidationPatterns(chunkID, newParentID string) []string {
	patterns := []string{
		"qcache:*chunk*" + chunkID + "*",
		"qcache:*hierarchy*",
		"qcache:*children*",
		"qcache:*ancestors*",
		"qcache:*descendants*",
		"qcache:*search*",
	}
	
	if newParentID != "" {
		patterns = append(patterns, "qcache:*"+newParentID+"*")
	}
	
	return patterns
}