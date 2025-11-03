package services

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// QueryPerformanceMonitor provides query performance monitoring (legacy interface)
type QueryPerformanceMonitor interface {
	RecordQuery(queryType string, duration time.Duration, rowCount int)
	RecordSlowQuery(query string, duration time.Duration, params map[string]interface{})
	GetQueryStats() QueryStatistics
	GetSlowQueries(limit int) []SlowQueryRecord
}

// EnhancedPerformanceMonitor provides comprehensive performance monitoring with alerting
type EnhancedPerformanceMonitor interface {
	QueryPerformanceMonitor
	
	// Alert functionality
	SetAlertThresholds(config *AlertConfig)
	GetAlerts(limit int) []AlertRecord
	ClearAlerts()
	
	// Health monitoring
	IsHealthy() bool
	GetPerformanceHealth() PerformanceHealthStatus
	
	// Error tracking
	RecordError(queryType string, err error)
	
	// Reset and cleanup
	Reset()
	Stop()
}

// AlertConfig defines thresholds for performance alerts
type AlertConfig struct {
	SlowQueryThreshold    time.Duration `json:"slow_query_threshold"`
	VerySlowQueryThreshold time.Duration `json:"very_slow_query_threshold"`
	HighErrorRateThreshold float64       `json:"high_error_rate_threshold"`
	MaxQueriesPerSecond   int           `json:"max_queries_per_second"`
	AlertCooldown         time.Duration `json:"alert_cooldown"`
}

// AlertRecord represents a performance alert
type AlertRecord struct {
	Type        string                 `json:"type"`
	Message     string                 `json:"message"`
	Severity    string                 `json:"severity"`
	Timestamp   time.Time              `json:"timestamp"`
	QueryType   string                 `json:"query_type,omitempty"`
	Duration    time.Duration          `json:"duration,omitempty"`
	Details     map[string]interface{} `json:"details,omitempty"`
}

// PerformanceHealthStatus represents the performance health of the system
type PerformanceHealthStatus struct {
	IsHealthy           bool          `json:"is_healthy"`
	AverageResponseTime time.Duration `json:"average_response_time"`
	SlowQueryRate       float64       `json:"slow_query_rate"`
	ErrorRate           float64       `json:"error_rate"`
	QueriesPerSecond    float64       `json:"queries_per_second"`
	LastCheck           time.Time     `json:"last_check"`
}

// QueryStatistics holds performance statistics
type QueryStatistics struct {
	TotalQueries    int64         `json:"total_queries"`
	AverageTime     time.Duration `json:"average_time"`
	SlowQueries     int64         `json:"slow_queries"`
	QueryTypes      map[string]QueryTypeStats `json:"query_types"`
	LastReset       time.Time     `json:"last_reset"`
}

// QueryTypeStats holds statistics for a specific query type
type QueryTypeStats struct {
	Count       int64         `json:"count"`
	TotalTime   time.Duration `json:"total_time"`
	AverageTime time.Duration `json:"average_time"`
	MinTime     time.Duration `json:"min_time"`
	MaxTime     time.Duration `json:"max_time"`
	TotalRows   int64         `json:"total_rows"`
}

// SlowQueryRecord represents a slow query record
type SlowQueryRecord struct {
	Query     string                 `json:"query"`
	Duration  time.Duration          `json:"duration"`
	Params    map[string]interface{} `json:"params"`
	Timestamp time.Time              `json:"timestamp"`
}

// InMemoryPerformanceMonitor implements EnhancedPerformanceMonitor using in-memory storage
type InMemoryPerformanceMonitor struct {
	mu             sync.RWMutex
	stats          QueryStatistics
	slowQueries    []SlowQueryRecord
	alerts         []AlertRecord
	slowThreshold  time.Duration
	maxSlowQueries int
	maxAlerts      int
	alertConfig    *AlertConfig
	lastAlertTime  map[string]time.Time
	startTime      time.Time
	errorCount     int64
	ctx            context.Context
	cancel         context.CancelFunc
	stopped        bool
}

// NewInMemoryPerformanceMonitor creates a new in-memory performance monitor
func NewInMemoryPerformanceMonitor(slowThreshold time.Duration, maxSlowQueries int) *InMemoryPerformanceMonitor {
	ctx, cancel := context.WithCancel(context.Background())
	
	monitor := &InMemoryPerformanceMonitor{
		stats: QueryStatistics{
			QueryTypes: make(map[string]QueryTypeStats),
			LastReset:  time.Now(),
		},
		slowQueries:    make([]SlowQueryRecord, 0),
		alerts:         make([]AlertRecord, 0),
		slowThreshold:  slowThreshold,
		maxSlowQueries: maxSlowQueries,
		maxAlerts:      1000,
		alertConfig:    DefaultAlertConfig(),
		lastAlertTime:  make(map[string]time.Time),
		startTime:      time.Now(),
		ctx:            ctx,
		cancel:         cancel,
	}
	
	// Start background monitoring
	go monitor.backgroundMonitoring()
	
	return monitor
}

// DefaultAlertConfig returns default alert configuration
func DefaultAlertConfig() *AlertConfig {
	return &AlertConfig{
		SlowQueryThreshold:     100 * time.Millisecond,
		VerySlowQueryThreshold: 1 * time.Second,
		HighErrorRateThreshold: 0.05, // 5% error rate
		MaxQueriesPerSecond:    1000,
		AlertCooldown:          5 * time.Minute,
	}
}

// RecordQuery records a query execution
func (m *InMemoryPerformanceMonitor) RecordQuery(queryType string, duration time.Duration, rowCount int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Update overall stats
	m.stats.TotalQueries++

	// Update query type stats
	typeStats, exists := m.stats.QueryTypes[queryType]
	if !exists {
		typeStats = QueryTypeStats{
			MinTime: duration,
			MaxTime: duration,
		}
	}

	typeStats.Count++
	typeStats.TotalTime += duration
	typeStats.AverageTime = typeStats.TotalTime / time.Duration(typeStats.Count)
	typeStats.TotalRows += int64(rowCount)

	if duration < typeStats.MinTime {
		typeStats.MinTime = duration
	}
	if duration > typeStats.MaxTime {
		typeStats.MaxTime = duration
	}

	m.stats.QueryTypes[queryType] = typeStats

	// Calculate overall average
	totalTime := time.Duration(0)
	for _, stats := range m.stats.QueryTypes {
		totalTime += stats.TotalTime
	}
	if m.stats.TotalQueries > 0 {
		m.stats.AverageTime = totalTime / time.Duration(m.stats.TotalQueries)
	}

	// Check if it's a slow query
	if duration >= m.slowThreshold {
		m.stats.SlowQueries++
	}

	// Check for alerts
	m.checkQueryAlerts(queryType, duration, rowCount)
}

// checkQueryAlerts checks if the query should trigger any alerts
func (m *InMemoryPerformanceMonitor) checkQueryAlerts(queryType string, duration time.Duration, rowCount int) {
	// Very slow query alert
	if duration >= m.alertConfig.VerySlowQueryThreshold {
		m.addAlert("very_slow_query", 
			fmt.Sprintf("Very slow %s query detected: %v", queryType, duration), 
			"critical", map[string]interface{}{
				"query_type": queryType,
				"duration": duration.String(),
				"row_count": rowCount,
				"threshold": m.alertConfig.VerySlowQueryThreshold.String(),
			})
	} else if duration >= m.alertConfig.SlowQueryThreshold {
		// Regular slow query alert (less frequent)
		if time.Since(m.lastAlertTime["slow_query"]) > m.alertConfig.AlertCooldown {
			m.addAlert("slow_query", 
				fmt.Sprintf("Slow %s query detected: %v", queryType, duration), 
				"warning", map[string]interface{}{
					"query_type": queryType,
					"duration": duration.String(),
					"row_count": rowCount,
					"threshold": m.alertConfig.SlowQueryThreshold.String(),
				})
		}
	}
}

// RecordSlowQuery records a slow query with details
func (m *InMemoryPerformanceMonitor) RecordSlowQuery(query string, duration time.Duration, params map[string]interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()

	record := SlowQueryRecord{
		Query:     query,
		Duration:  duration,
		Params:    params,
		Timestamp: time.Now(),
	}

	// Add to slow queries list
	m.slowQueries = append(m.slowQueries, record)

	// Keep only the most recent slow queries
	if len(m.slowQueries) > m.maxSlowQueries {
		m.slowQueries = m.slowQueries[len(m.slowQueries)-m.maxSlowQueries:]
	}

	m.stats.SlowQueries++

	// Generate alert for slow query with full details
	m.addAlert("slow_query_detailed", 
		fmt.Sprintf("Slow query recorded: %v", duration), 
		"warning", map[string]interface{}{
			"query": query,
			"duration": duration.String(),
			"params": params,
			"timestamp": record.Timestamp,
		})
}

// GetQueryStats returns current query statistics
func (m *InMemoryPerformanceMonitor) GetQueryStats() QueryStatistics {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Create a deep copy to avoid race conditions
	stats := QueryStatistics{
		TotalQueries: m.stats.TotalQueries,
		AverageTime:  m.stats.AverageTime,
		SlowQueries:  m.stats.SlowQueries,
		QueryTypes:   make(map[string]QueryTypeStats),
		LastReset:    m.stats.LastReset,
	}

	for k, v := range m.stats.QueryTypes {
		stats.QueryTypes[k] = v
	}

	return stats
}

// GetSlowQueries returns the most recent slow queries
func (m *InMemoryPerformanceMonitor) GetSlowQueries(limit int) []SlowQueryRecord {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if limit <= 0 || limit > len(m.slowQueries) {
		limit = len(m.slowQueries)
	}

	// Return the most recent queries
	start := len(m.slowQueries) - limit
	if start < 0 {
		start = 0
	}

	result := make([]SlowQueryRecord, limit)
	copy(result, m.slowQueries[start:])

	return result
}

// SetAlertThresholds updates the alert configuration
func (m *InMemoryPerformanceMonitor) SetAlertThresholds(config *AlertConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.alertConfig = config
}

// GetAlerts returns recent alerts
func (m *InMemoryPerformanceMonitor) GetAlerts(limit int) []AlertRecord {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if limit <= 0 || limit > len(m.alerts) {
		limit = len(m.alerts)
	}

	// Return the most recent alerts
	start := len(m.alerts) - limit
	if start < 0 {
		start = 0
	}

	result := make([]AlertRecord, limit)
	copy(result, m.alerts[start:])

	return result
}

// ClearAlerts removes all alerts
func (m *InMemoryPerformanceMonitor) ClearAlerts() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.alerts = make([]AlertRecord, 0)
}

// IsHealthy returns true if the system is performing within acceptable parameters
func (m *InMemoryPerformanceMonitor) IsHealthy() bool {
	status := m.GetPerformanceHealth()
	return status.IsHealthy
}

// GetPerformanceHealth returns detailed performance health information
func (m *InMemoryPerformanceMonitor) GetPerformanceHealth() PerformanceHealthStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	now := time.Now()
	uptime := now.Sub(m.startTime)
	
	// Calculate queries per second
	var qps float64
	if uptime.Seconds() > 0 {
		qps = float64(m.stats.TotalQueries) / uptime.Seconds()
	}

	// Calculate slow query rate
	var slowQueryRate float64
	if m.stats.TotalQueries > 0 {
		slowQueryRate = float64(m.stats.SlowQueries) / float64(m.stats.TotalQueries)
	}

	// Calculate error rate
	var errorRate float64
	if m.stats.TotalQueries > 0 {
		errorRate = float64(m.errorCount) / float64(m.stats.TotalQueries)
	}

	// Determine if system is healthy
	isHealthy := true
	if m.stats.AverageTime > m.alertConfig.VerySlowQueryThreshold {
		isHealthy = false
	}
	if slowQueryRate > 0.1 { // More than 10% slow queries
		isHealthy = false
	}
	if errorRate > m.alertConfig.HighErrorRateThreshold {
		isHealthy = false
	}
	if qps > float64(m.alertConfig.MaxQueriesPerSecond) {
		isHealthy = false
	}

	return PerformanceHealthStatus{
		IsHealthy:           isHealthy,
		AverageResponseTime: m.stats.AverageTime,
		SlowQueryRate:       slowQueryRate,
		ErrorRate:           errorRate,
		QueriesPerSecond:    qps,
		LastCheck:           now,
	}
}

// RecordError records a query error for error rate calculation
func (m *InMemoryPerformanceMonitor) RecordError(queryType string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.errorCount++
	
	// Check if we should trigger an error rate alert
	if m.stats.TotalQueries > 0 {
		errorRate := float64(m.errorCount) / float64(m.stats.TotalQueries)
		if errorRate > m.alertConfig.HighErrorRateThreshold {
			m.addAlert("high_error_rate", fmt.Sprintf("High error rate detected: %.2f%%", errorRate*100), "warning", map[string]interface{}{
				"error_rate": errorRate,
				"error_count": m.errorCount,
				"total_queries": m.stats.TotalQueries,
				"query_type": queryType,
				"error": err.Error(),
			})
		}
	}
}

// Reset clears all statistics
func (m *InMemoryPerformanceMonitor) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.stats = QueryStatistics{
		QueryTypes: make(map[string]QueryTypeStats),
		LastReset:  time.Now(),
	}
	m.slowQueries = make([]SlowQueryRecord, 0)
	m.alerts = make([]AlertRecord, 0)
	m.errorCount = 0
	m.lastAlertTime = make(map[string]time.Time)
}

// Stop gracefully shuts down the performance monitor
func (m *InMemoryPerformanceMonitor) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if !m.stopped {
		m.cancel()
		m.stopped = true
	}
}

// addAlert adds a new alert (internal method, must be called with lock held)
func (m *InMemoryPerformanceMonitor) addAlert(alertType, message, severity string, details map[string]interface{}) {
	// Check cooldown
	if lastAlert, exists := m.lastAlertTime[alertType]; exists {
		if time.Since(lastAlert) < m.alertConfig.AlertCooldown {
			return // Still in cooldown period
		}
	}

	alert := AlertRecord{
		Type:      alertType,
		Message:   message,
		Severity:  severity,
		Timestamp: time.Now(),
		Details:   details,
	}

	m.alerts = append(m.alerts, alert)
	m.lastAlertTime[alertType] = alert.Timestamp

	// Keep only the most recent alerts
	if len(m.alerts) > m.maxAlerts {
		m.alerts = m.alerts[len(m.alerts)-m.maxAlerts:]
	}

	// Log the alert
	log.Printf("PERFORMANCE ALERT [%s]: %s - %s", severity, alertType, message)
}

// backgroundMonitoring runs periodic health checks and alerts
func (m *InMemoryPerformanceMonitor) backgroundMonitoring() {
	ticker := time.NewTicker(30 * time.Second) // Check every 30 seconds
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.performHealthCheck()
		}
	}
}

// performHealthCheck performs periodic health monitoring
func (m *InMemoryPerformanceMonitor) performHealthCheck() {
	m.mu.Lock()
	defer m.mu.Unlock()

	status := m.getHealthStatusLocked()
	
	// Check for performance issues and generate alerts
	if !status.IsHealthy {
		details := map[string]interface{}{
			"average_response_time": status.AverageResponseTime.String(),
			"slow_query_rate": status.SlowQueryRate,
			"error_rate": status.ErrorRate,
			"queries_per_second": status.QueriesPerSecond,
		}

		if status.AverageResponseTime > m.alertConfig.VerySlowQueryThreshold {
			m.addAlert("high_average_response_time", 
				fmt.Sprintf("High average response time: %v", status.AverageResponseTime), 
				"critical", details)
		}

		if status.SlowQueryRate > 0.1 {
			m.addAlert("high_slow_query_rate", 
				fmt.Sprintf("High slow query rate: %.2f%%", status.SlowQueryRate*100), 
				"warning", details)
		}

		if status.QueriesPerSecond > float64(m.alertConfig.MaxQueriesPerSecond) {
			m.addAlert("high_query_rate", 
				fmt.Sprintf("High query rate: %.2f queries/second", status.QueriesPerSecond), 
				"warning", details)
		}
	}
}

// getHealthStatusLocked returns health status (must be called with lock held)
func (m *InMemoryPerformanceMonitor) getHealthStatusLocked() PerformanceHealthStatus {
	now := time.Now()
	uptime := now.Sub(m.startTime)
	
	var qps float64
	if uptime.Seconds() > 0 {
		qps = float64(m.stats.TotalQueries) / uptime.Seconds()
	}

	var slowQueryRate float64
	if m.stats.TotalQueries > 0 {
		slowQueryRate = float64(m.stats.SlowQueries) / float64(m.stats.TotalQueries)
	}

	var errorRate float64
	if m.stats.TotalQueries > 0 {
		errorRate = float64(m.errorCount) / float64(m.stats.TotalQueries)
	}

	isHealthy := true
	if m.stats.AverageTime > m.alertConfig.VerySlowQueryThreshold {
		isHealthy = false
	}
	if slowQueryRate > 0.1 {
		isHealthy = false
	}
	if errorRate > m.alertConfig.HighErrorRateThreshold {
		isHealthy = false
	}
	if qps > float64(m.alertConfig.MaxQueriesPerSecond) {
		isHealthy = false
	}

	return PerformanceHealthStatus{
		IsHealthy:           isHealthy,
		AverageResponseTime: m.stats.AverageTime,
		SlowQueryRate:       slowQueryRate,
		ErrorRate:           errorRate,
		QueriesPerSecond:    qps,
		LastCheck:           now,
	}
}

// PerformanceConfig holds performance monitoring configuration
type PerformanceConfig struct {
	SlowQueryThreshold time.Duration `json:"slow_query_threshold"`
	MaxSlowQueries     int           `json:"max_slow_queries"`
	Enabled            bool          `json:"enabled"`
}

// DefaultPerformanceConfig returns default performance monitoring configuration
func DefaultPerformanceConfig() *PerformanceConfig {
	return &PerformanceConfig{
		SlowQueryThreshold: 100 * time.Millisecond,
		MaxSlowQueries:     100,
		Enabled:            true,
	}
}