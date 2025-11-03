package services

import (
	"context"
	"fmt"
	"time"
)

// HealthStatus represents the health status of a component
type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
	HealthStatusDegraded  HealthStatus = "degraded"
)

// ComponentHealth represents the health of a single component
type ComponentHealth struct {
	Name      string                 `json:"name"`
	Status    HealthStatus           `json:"status"`
	Message   string                 `json:"message,omitempty"`
	Details   map[string]interface{} `json:"details,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Duration  time.Duration          `json:"duration"`
}

// SystemHealth represents the overall system health
type SystemHealth struct {
	Status     HealthStatus               `json:"status"`
	Timestamp  time.Time                  `json:"timestamp"`
	Uptime     time.Duration              `json:"uptime"`
	Version    string                     `json:"version,omitempty"`
	Components map[string]ComponentHealth `json:"components"`
}

// HealthChecker interface for health checking
type HealthChecker interface {
	Name() string
	Check(ctx context.Context) ComponentHealth
}

// HealthService manages health checks for the system
type HealthService interface {
	RegisterChecker(checker HealthChecker)
	CheckHealth(ctx context.Context) SystemHealth
	CheckComponent(ctx context.Context, name string) (ComponentHealth, error)
	GetSystemInfo() map[string]interface{}
}

// DefaultHealthService implements HealthService
type DefaultHealthService struct {
	checkers  map[string]HealthChecker
	startTime time.Time
	version   string
	logger    Logger
}

// NewHealthService creates a new health service
func NewHealthService(version string, logger Logger) *DefaultHealthService {
	if logger == nil {
		logger = NewDefaultLogger()
	}
	
	return &DefaultHealthService{
		checkers:  make(map[string]HealthChecker),
		startTime: time.Now(),
		version:   version,
		logger:    logger,
	}
}

// RegisterChecker registers a health checker
func (h *DefaultHealthService) RegisterChecker(checker HealthChecker) {
	h.checkers[checker.Name()] = checker
	h.logger.Info("Health checker registered", String("component", checker.Name()))
}

// CheckHealth performs health checks on all registered components
func (h *DefaultHealthService) CheckHealth(ctx context.Context) SystemHealth {
	start := time.Now()
	components := make(map[string]ComponentHealth)
	overallStatus := HealthStatusHealthy
	
	// Check each component
	for name, checker := range h.checkers {
		componentHealth := h.checkComponentWithTimeout(ctx, checker, 5*time.Second)
		components[name] = componentHealth
		
		// Determine overall status
		switch componentHealth.Status {
		case HealthStatusUnhealthy:
			overallStatus = HealthStatusUnhealthy
		case HealthStatusDegraded:
			if overallStatus == HealthStatusHealthy {
				overallStatus = HealthStatusDegraded
			}
		}
	}
	
	systemHealth := SystemHealth{
		Status:     overallStatus,
		Timestamp:  time.Now(),
		Uptime:     time.Since(h.startTime),
		Version:    h.version,
		Components: components,
	}
	
	duration := time.Since(start)
	h.logger.Info("Health check completed",
		String("status", string(overallStatus)),
		Duration("duration", duration),
		Int("components_checked", len(components)))
	
	return systemHealth
}

// CheckComponent checks the health of a specific component
func (h *DefaultHealthService) CheckComponent(ctx context.Context, name string) (ComponentHealth, error) {
	checker, exists := h.checkers[name]
	if !exists {
		return ComponentHealth{}, fmt.Errorf("component %s not found", name)
	}
	
	return h.checkComponentWithTimeout(ctx, checker, 5*time.Second), nil
}

// GetSystemInfo returns general system information
func (h *DefaultHealthService) GetSystemInfo() map[string]interface{} {
	return map[string]interface{}{
		"version":    h.version,
		"uptime":     time.Since(h.startTime).String(),
		"start_time": h.startTime.Format(time.RFC3339),
		"components": len(h.checkers),
	}
}

// checkComponentWithTimeout checks a component with a timeout
func (h *DefaultHealthService) checkComponentWithTimeout(ctx context.Context, checker HealthChecker, timeout time.Duration) ComponentHealth {
	start := time.Now()
	
	// Create context with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	
	// Channel to receive the result
	resultChan := make(chan ComponentHealth, 1)
	
	// Run the health check in a goroutine
	go func() {
		defer func() {
			if r := recover(); r != nil {
				resultChan <- ComponentHealth{
					Name:      checker.Name(),
					Status:    HealthStatusUnhealthy,
					Message:   fmt.Sprintf("Health check panicked: %v", r),
					Timestamp: time.Now(),
					Duration:  time.Since(start),
				}
			}
		}()
		
		result := checker.Check(timeoutCtx)
		result.Duration = time.Since(start)
		resultChan <- result
	}()
	
	// Wait for result or timeout
	select {
	case result := <-resultChan:
		return result
	case <-timeoutCtx.Done():
		return ComponentHealth{
			Name:      checker.Name(),
			Status:    HealthStatusUnhealthy,
			Message:   "Health check timed out",
			Timestamp: time.Now(),
			Duration:  timeout,
		}
	}
}

// DatabaseHealthChecker checks database connectivity
type DatabaseHealthChecker struct {
	name   string
	client SupabaseClient
}

// NewDatabaseHealthChecker creates a database health checker
func NewDatabaseHealthChecker(name string, client SupabaseClient) *DatabaseHealthChecker {
	return &DatabaseHealthChecker{
		name:   name,
		client: client,
	}
}

// Name returns the checker name
func (d *DatabaseHealthChecker) Name() string {
	return d.name
}

// Check performs the database health check
func (d *DatabaseHealthChecker) Check(ctx context.Context) ComponentHealth {
	start := time.Now()
	
	err := d.client.HealthCheck(ctx)
	
	health := ComponentHealth{
		Name:      d.name,
		Timestamp: time.Now(),
		Duration:  time.Since(start),
	}
	
	if err != nil {
		health.Status = HealthStatusUnhealthy
		health.Message = err.Error()
	} else {
		health.Status = HealthStatusHealthy
		health.Message = "Database connection successful"
	}
	
	return health
}

// CacheHealthChecker checks cache service health
type CacheHealthChecker struct {
	name  string
	cache CacheService
}

// NewCacheHealthChecker creates a cache health checker
func NewCacheHealthChecker(name string, cache CacheService) *CacheHealthChecker {
	return &CacheHealthChecker{
		name:  name,
		cache: cache,
	}
}

// Name returns the checker name
func (c *CacheHealthChecker) Name() string {
	return c.name
}

// Check performs the cache health check
func (c *CacheHealthChecker) Check(ctx context.Context) ComponentHealth {
	start := time.Now()
	
	health := ComponentHealth{
		Name:      c.name,
		Timestamp: time.Now(),
	}
	
	// Test cache operations
	testKey := "health_check_test"
	testValue := "test_value"
	
	// Try to set a value
	if err := c.cache.Set(ctx, testKey, testValue, time.Minute); err != nil {
		health.Status = HealthStatusUnhealthy
		health.Message = fmt.Sprintf("Cache set failed: %v", err)
		health.Duration = time.Since(start)
		return health
	}
	
	// Try to get the value
	var result string
	if err := c.cache.Get(ctx, testKey, &result); err != nil {
		health.Status = HealthStatusUnhealthy
		health.Message = fmt.Sprintf("Cache get failed: %v", err)
		health.Duration = time.Since(start)
		return health
	}
	
	// Verify the value
	if result != testValue {
		health.Status = HealthStatusUnhealthy
		health.Message = "Cache value mismatch"
		health.Duration = time.Since(start)
		return health
	}
	
	// Clean up
	c.cache.Delete(ctx, testKey)
	
	// Get cache stats
	stats := c.cache.GetStats()
	
	health.Status = HealthStatusHealthy
	health.Message = "Cache operations successful"
	health.Duration = time.Since(start)
	health.Details = map[string]interface{}{
		"hit_rate": stats.HitRate,
		"size":     stats.Size,
		"max_size": stats.MaxSize,
	}
	
	return health
}

// MetricsHealthChecker checks metrics service health
type MetricsHealthChecker struct {
	name    string
	metrics MetricsService
}

// NewMetricsHealthChecker creates a metrics health checker
func NewMetricsHealthChecker(name string, metrics MetricsService) *MetricsHealthChecker {
	return &MetricsHealthChecker{
		name:    name,
		metrics: metrics,
	}
}

// Name returns the checker name
func (m *MetricsHealthChecker) Name() string {
	return m.name
}

// Check performs the metrics health check
func (m *MetricsHealthChecker) Check(ctx context.Context) ComponentHealth {
	start := time.Now()
	
	health := ComponentHealth{
		Name:      m.name,
		Timestamp: time.Now(),
	}
	
	// Test metrics operations
	testCounter := "health_check_counter"
	testGauge := "health_check_gauge"
	
	// Test counter
	m.metrics.IncrementCounter(testCounter, map[string]string{"test": "true"})
	
	// Test gauge
	m.metrics.SetGauge(testGauge, 42.0, map[string]string{"test": "true"})
	
	// Test duration recording
	m.metrics.RecordDuration("health_check_duration", time.Millisecond*100, map[string]string{"test": "true"})
	
	// Get metrics
	allMetrics := m.metrics.GetMetrics()
	
	health.Status = HealthStatusHealthy
	health.Message = "Metrics operations successful"
	health.Duration = time.Since(start)
	health.Details = map[string]interface{}{
		"counters":   len(allMetrics),
		"has_system": allMetrics["system"] != nil,
	}
	
	return health
}