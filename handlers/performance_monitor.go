package handlers

import (
	"context"
	"log"
	"net/http"
	"time"
)

// PerformanceMonitor handles performance monitoring for handler operations
type PerformanceMonitor struct {
	slowQueryThreshold time.Duration
	logger            *log.Logger
	metricsEnabled    bool
}

// NewPerformanceMonitor creates a new performance monitor
func NewPerformanceMonitor(slowQueryThreshold time.Duration, logger *log.Logger, metricsEnabled bool) *PerformanceMonitor {
	return &PerformanceMonitor{
		slowQueryThreshold: slowQueryThreshold,
		logger:            logger,
		metricsEnabled:    metricsEnabled,
	}
}

// OperationMetrics contains metrics for a single operation
type OperationMetrics struct {
	Operation   string
	Duration    time.Duration
	Success     bool
	StatusCode  int
	ErrorMsg    string
	Timestamp   time.Time
}

// MonitoredOperation wraps an operation with performance monitoring
func (pm *PerformanceMonitor) MonitoredOperation(operation string, fn func() error) error {
	start := time.Now()
	err := fn()
	duration := time.Since(start)

	success := err == nil

	// Record metrics
	if pm.metricsEnabled {
		pm.recordMetrics(OperationMetrics{
			Operation: operation,
			Duration:  duration,
			Success:   success,
			Timestamp: time.Now(),
			ErrorMsg:  pm.getErrorMsg(err),
		})
	}

	// Log slow queries
	if duration > pm.slowQueryThreshold {
		pm.logSlowOperation(operation, duration, err)
	}

	return err
}

// MonitoredHTTPOperation wraps an HTTP handler operation with monitoring
func (pm *PerformanceMonitor) MonitoredHTTPOperation(operation string, w http.ResponseWriter, fn func() (int, error)) {
	start := time.Now()
	statusCode, err := fn()
	duration := time.Since(start)

	success := err == nil && statusCode < 400

	// Record metrics
	if pm.metricsEnabled {
		pm.recordMetrics(OperationMetrics{
			Operation:  operation,
			Duration:   duration,
			Success:    success,
			StatusCode: statusCode,
			Timestamp:  time.Now(),
			ErrorMsg:   pm.getErrorMsg(err),
		})
	}

	// Log slow operations
	if duration > pm.slowQueryThreshold {
		pm.logSlowOperation(operation, duration, err)
	}

	// Add performance headers
	w.Header().Set("X-Response-Time", duration.String())
	if duration > pm.slowQueryThreshold {
		w.Header().Set("X-Slow-Query", "true")
	}
}

// MonitoredContextOperation wraps a context-aware operation with monitoring
func (pm *PerformanceMonitor) MonitoredContextOperation(ctx context.Context, operation string, fn func(context.Context) error) error {
	start := time.Now()
	err := fn(ctx)
	duration := time.Since(start)

	success := err == nil

	// Record metrics
	if pm.metricsEnabled {
		pm.recordMetrics(OperationMetrics{
			Operation: operation,
			Duration:  duration,
			Success:   success,
			Timestamp: time.Now(),
			ErrorMsg:  pm.getErrorMsg(err),
		})
	}

	// Log slow operations
	if duration > pm.slowQueryThreshold {
		pm.logSlowOperation(operation, duration, err)
	}

	return err
}

// recordMetrics records operation metrics (placeholder for actual metrics system)
func (pm *PerformanceMonitor) recordMetrics(metrics OperationMetrics) {
	// In a real implementation, this would send metrics to a monitoring system
	// like Prometheus, DataDog, or CloudWatch
	if pm.logger != nil {
		pm.logger.Printf("METRICS: operation=%s duration=%v success=%v status=%d",
			metrics.Operation, metrics.Duration, metrics.Success, metrics.StatusCode)
	}
}

// logSlowOperation logs slow operations for debugging
func (pm *PerformanceMonitor) logSlowOperation(operation string, duration time.Duration, err error) {
	if pm.logger != nil {
		if err != nil {
			pm.logger.Printf("SLOW_OPERATION_ERROR: operation=%s duration=%v error=%v",
				operation, duration, err)
		} else {
			pm.logger.Printf("SLOW_OPERATION: operation=%s duration=%v",
				operation, duration)
		}
	}
}

// getErrorMsg safely extracts error message
func (pm *PerformanceMonitor) getErrorMsg(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// GetSlowQueryThreshold returns the current slow query threshold
func (pm *PerformanceMonitor) GetSlowQueryThreshold() time.Duration {
	return pm.slowQueryThreshold
}

// SetSlowQueryThreshold updates the slow query threshold
func (pm *PerformanceMonitor) SetSlowQueryThreshold(threshold time.Duration) {
	pm.slowQueryThreshold = threshold
}

// CacheAwareOperation monitors operations that may hit cache
func (pm *PerformanceMonitor) CacheAwareOperation(operation string, cacheHit bool, fn func() error) error {
	cacheStatus := "miss"
	if cacheHit {
		cacheStatus = "hit"
	}

	operationWithCache := operation + "_cache_" + cacheStatus
	return pm.MonitoredOperation(operationWithCache, fn)
}

// BatchOperation monitors batch operations with item counts
func (pm *PerformanceMonitor) BatchOperation(operation string, itemCount int, fn func() error) error {
	start := time.Now()
	err := fn()
	duration := time.Since(start)

	success := err == nil

	// Record batch-specific metrics
	if pm.metricsEnabled {
		pm.recordBatchMetrics(operation, itemCount, duration, success, err)
	}

	// Log slow batch operations (adjusted threshold based on item count)
	adjustedThreshold := pm.slowQueryThreshold
	if itemCount > 10 {
		adjustedThreshold = pm.slowQueryThreshold * time.Duration(itemCount/10)
	}

	if duration > adjustedThreshold {
		pm.logSlowBatchOperation(operation, itemCount, duration, err)
	}

	return err
}

// recordBatchMetrics records metrics for batch operations
func (pm *PerformanceMonitor) recordBatchMetrics(operation string, itemCount int, duration time.Duration, success bool, err error) {
	if pm.logger != nil {
		pm.logger.Printf("BATCH_METRICS: operation=%s items=%d duration=%v success=%v avg_per_item=%v",
			operation, itemCount, duration, success, duration/time.Duration(max(itemCount, 1)))
	}
}

// logSlowBatchOperation logs slow batch operations
func (pm *PerformanceMonitor) logSlowBatchOperation(operation string, itemCount int, duration time.Duration, err error) {
	if pm.logger != nil {
		if err != nil {
			pm.logger.Printf("SLOW_BATCH_ERROR: operation=%s items=%d duration=%v error=%v",
				operation, itemCount, duration, err)
		} else {
			pm.logger.Printf("SLOW_BATCH: operation=%s items=%d duration=%v",
				operation, itemCount, duration)
		}
	}
}

// max returns the maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}