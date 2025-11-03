package services

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// CacheService provides caching functionality
type CacheService interface {
	Get(ctx context.Context, key string, dest interface{}) error
	GetDirect(ctx context.Context, key string) (interface{}, bool) // For direct value retrieval
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	DeletePattern(ctx context.Context, pattern string) error // For pattern-based deletion
	Clear(ctx context.Context) error
	GetStats() CacheStats
}

// CacheStats provides cache performance metrics
type CacheStats struct {
	Hits        int64   `json:"hits"`
	Misses      int64   `json:"misses"`
	HitRate     float64 `json:"hit_rate"`
	Size        int     `json:"size"`
	MaxSize     int     `json:"max_size"`
	Evictions   int64   `json:"evictions"`
	LastCleared time.Time `json:"last_cleared"`
}

// CacheEntry represents a cached item
type CacheEntry struct {
	Value     []byte    `json:"value"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

// InMemoryCache implements CacheService using in-memory storage
type InMemoryCache struct {
	mu       sync.RWMutex
	data     map[string]*CacheEntry
	maxSize  int
	stats    CacheStats
	janitor  *time.Ticker
	stopChan chan struct{}
}

// NewInMemoryCache creates a new in-memory cache
func NewInMemoryCache(maxSize int, cleanupInterval time.Duration) *InMemoryCache {
	cache := &InMemoryCache{
		data:     make(map[string]*CacheEntry),
		maxSize:  maxSize,
		stats:    CacheStats{MaxSize: maxSize, LastCleared: time.Now()},
		janitor:  time.NewTicker(cleanupInterval),
		stopChan: make(chan struct{}),
	}
	
	// Start cleanup goroutine
	go cache.cleanup()
	
	return cache
}

// Get retrieves a value from cache
func (c *InMemoryCache) Get(ctx context.Context, key string, dest interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	entry, exists := c.data[key]
	if !exists {
		c.stats.Misses++
		c.updateHitRate()
		return fmt.Errorf("cache miss: key %s not found", key)
	}
	
	// Check if expired
	if time.Now().After(entry.ExpiresAt) {
		c.stats.Misses++
		// Remove expired entry immediately
		delete(c.data, key)
		c.stats.Size = len(c.data)
		c.updateHitRate()
		return fmt.Errorf("cache miss: key %s expired", key)
	}
	
	c.stats.Hits++
	c.updateHitRate()
	
	// Deserialize value
	return json.Unmarshal(entry.Value, dest)
}

// GetDirect retrieves a value from cache without deserialization
func (c *InMemoryCache) GetDirect(ctx context.Context, key string) (interface{}, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	entry, exists := c.data[key]
	if !exists {
		c.stats.Misses++
		c.updateHitRate()
		return nil, false
	}
	
	// Check if expired
	if time.Now().After(entry.ExpiresAt) {
		c.stats.Misses++
		// Remove expired entry immediately
		delete(c.data, key)
		c.stats.Size = len(c.data)
		c.updateHitRate()
		return nil, false
	}
	
	c.stats.Hits++
	c.updateHitRate()
	
	// Deserialize and return value
	var value interface{}
	if err := json.Unmarshal(entry.Value, &value); err != nil {
		return nil, false
	}
	
	return value, true
}

// Set stores a value in cache
func (c *InMemoryCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	// Serialize value
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to serialize cache value: %w", err)
	}
	
	// Check if we need to evict entries
	if len(c.data) >= c.maxSize {
		c.evictOldest()
	}
	
	entry := &CacheEntry{
		Value:     data,
		ExpiresAt: time.Now().Add(ttl),
		CreatedAt: time.Now(),
	}
	
	c.data[key] = entry
	c.stats.Size = len(c.data)
	
	return nil
}

// Delete removes a key from cache
func (c *InMemoryCache) Delete(ctx context.Context, key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	delete(c.data, key)
	c.stats.Size = len(c.data)
	
	return nil
}

// DeletePattern removes all keys matching a pattern from cache
func (c *InMemoryCache) DeletePattern(ctx context.Context, pattern string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	// Simple pattern matching - supports * wildcard at the end
	if pattern == "*" {
		// Delete all
		c.data = make(map[string]*CacheEntry)
		c.stats.Size = 0
		return nil
	}
	
	// Pattern matching with * at the end
	if len(pattern) > 0 && pattern[len(pattern)-1] == '*' {
		prefix := pattern[:len(pattern)-1]
		keysToDelete := make([]string, 0)
		
		for key := range c.data {
			if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
				keysToDelete = append(keysToDelete, key)
			}
		}
		
		for _, key := range keysToDelete {
			delete(c.data, key)
		}
	} else {
		// Exact match
		delete(c.data, pattern)
	}
	
	c.stats.Size = len(c.data)
	return nil
}

// Clear removes all entries from cache
func (c *InMemoryCache) Clear(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	c.data = make(map[string]*CacheEntry)
	c.stats.Size = 0
	c.stats.LastCleared = time.Now()
	
	return nil
}

// GetStats returns cache statistics
func (c *InMemoryCache) GetStats() CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	stats := c.stats
	stats.Size = len(c.data)
	return stats
}

// Stop stops the cache cleanup goroutine
func (c *InMemoryCache) Stop() {
	close(c.stopChan)
	c.janitor.Stop()
}

// cleanup removes expired entries periodically
func (c *InMemoryCache) cleanup() {
	for {
		select {
		case <-c.janitor.C:
			c.removeExpired()
		case <-c.stopChan:
			return
		}
	}
}

// removeExpired removes all expired entries
func (c *InMemoryCache) removeExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	now := time.Now()
	for key, entry := range c.data {
		if now.After(entry.ExpiresAt) {
			delete(c.data, key)
		}
	}
	c.stats.Size = len(c.data)
}

// evictOldest removes the oldest entry to make room for new ones
func (c *InMemoryCache) evictOldest() {
	var oldestKey string
	var oldestTime time.Time
	
	for key, entry := range c.data {
		if oldestKey == "" || entry.CreatedAt.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.CreatedAt
		}
	}
	
	if oldestKey != "" {
		delete(c.data, oldestKey)
		c.stats.Evictions++
	}
}

// updateHitRate calculates the current hit rate
func (c *InMemoryCache) updateHitRate() {
	total := c.stats.Hits + c.stats.Misses
	if total > 0 {
		c.stats.HitRate = float64(c.stats.Hits) / float64(total)
	}
}

// CacheConfig holds cache configuration
type CacheConfig struct {
	MaxSize         int           `json:"max_size"`
	CleanupInterval time.Duration `json:"cleanup_interval"`
	DefaultTTL      time.Duration `json:"default_ttl"`
	Enabled         bool          `json:"enabled"`
}

// DefaultCacheConfig returns default cache configuration
func DefaultCacheConfig() *CacheConfig {
	return &CacheConfig{
		MaxSize:         1000,
		CleanupInterval: 5 * time.Minute,
		DefaultTTL:      30 * time.Minute,
		Enabled:         true,
	}
}