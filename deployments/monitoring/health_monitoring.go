package monitoring

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"semantic-text-processor/database"
	"semantic-text-processor/services"
	"sync"
	"time"
)

// HealthMonitor manages system health checks and monitoring
type HealthMonitor struct {
	config            *HealthConfig
	dbClient          database.Client
	serviceFactory    *services.Factory
	checks            map[string]HealthChecker
	metrics           *MetricsCollector
	alertManager      *AlertManager
	lastHealthReport  *HealthReport
	mutex             sync.RWMutex
}

// HealthConfig contains health monitoring configuration
type HealthConfig struct {
	CheckInterval        time.Duration `json:"check_interval"`
	Timeout              time.Duration `json:"timeout"`
	FailureThreshold     int           `json:"failure_threshold"`
	RecoveryThreshold    int           `json:"recovery_threshold"`
	AlertOnFailure       bool          `json:"alert_on_failure"`
	EnableDetailedChecks bool          `json:"enable_detailed_checks"`
	EndpointsToCheck     []string      `json:"endpoints_to_check"`
}

// HealthChecker defines interface for individual health checks
type HealthChecker interface {
	Name() string
	Check(ctx context.Context) *ComponentHealth
	Priority() Priority
	Timeout() time.Duration
}

// ComponentHealth represents the health status of a component
type ComponentHealth struct {
	Component   string                 `json:"component"`
	Status      HealthStatus           `json:"status"`
	Message     string                 `json:"message"`
	Timestamp   time.Time              `json:"timestamp"`
	Duration    time.Duration          `json:"duration"`
	Details     map[string]interface{} `json:"details,omitempty"`
	Error       string                 `json:"error,omitempty"`
	Metrics     map[string]float64     `json:"metrics,omitempty"`
}

// HealthReport aggregates all component health statuses
type HealthReport struct {
	OverallStatus string                      `json:"overall_status"`
	Timestamp     time.Time                   `json:"timestamp"`
	Version       string                      `json:"version"`
	Uptime        time.Duration               `json:"uptime"`
	Components    map[string]*ComponentHealth `json:"components"`
	Summary       *HealthSummary              `json:"summary"`
	Alerts        []Alert                     `json:"alerts,omitempty"`
}

// HealthSummary provides aggregated health information
type HealthSummary struct {
	TotalComponents int                        `json:"total_components"`
	HealthyCount    int                        `json:"healthy_count"`
	WarningCount    int                        `json:"warning_count"`
	CriticalCount   int                        `json:"critical_count"`
	FailedCount     int                        `json:"failed_count"`
	Scores          map[string]float64         `json:"scores"`
	Trends          map[string]HealthTrend     `json:"trends"`
}

// HealthStatus represents component status
type HealthStatus string

const (
	StatusHealthy  HealthStatus = "healthy"
	StatusWarning  HealthStatus = "warning"
	StatusCritical HealthStatus = "critical"
	StatusFailed   HealthStatus = "failed"
	StatusUnknown  HealthStatus = "unknown"
)

// Priority represents check priority
type Priority int

const (
	PriorityLow Priority = iota
	PriorityMedium
	PriorityHigh
	PriorityCritical
)

// HealthTrend represents health trend over time
type HealthTrend struct {
	Direction string  `json:"direction"` // "improving", "stable", "degrading"
	Change    float64 `json:"change"`    // Percentage change
	Period    string  `json:"period"`    // Time period for trend
}

// NewHealthMonitor creates a new health monitor
func NewHealthMonitor(config *HealthConfig, dbClient database.Client, serviceFactory *services.Factory) *HealthMonitor {
	monitor := &HealthMonitor{
		config:         config,
		dbClient:       dbClient,
		serviceFactory: serviceFactory,
		checks:         make(map[string]HealthChecker),
		metrics:        NewMetricsCollector(),
		alertManager:   NewAlertManager(),
	}

	// Register default health checkers
	monitor.registerDefaultCheckers()

	return monitor
}

// registerDefaultCheckers registers built-in health checkers
func (h *HealthMonitor) registerDefaultCheckers() {
	h.RegisterChecker(NewDatabaseHealthChecker(h.dbClient))
	h.RegisterChecker(NewMemoryHealthChecker())
	h.RegisterChecker(NewCPUHealthChecker())
	h.RegisterChecker(NewDiskHealthChecker())
	h.RegisterChecker(NewNetworkHealthChecker())
	h.RegisterChecker(NewServiceHealthChecker(h.serviceFactory))

	if h.config.EnableDetailedChecks {
		h.RegisterChecker(NewEmbeddingServiceChecker())
		h.RegisterChecker(NewLLMServiceChecker())
		h.RegisterChecker(NewCacheHealthChecker())
	}
}

// RegisterChecker adds a new health checker
func (h *HealthMonitor) RegisterChecker(checker HealthChecker) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	h.checks[checker.Name()] = checker
}

// StartMonitoring begins continuous health monitoring
func (h *HealthMonitor) StartMonitoring(ctx context.Context) error {
	ticker := time.NewTicker(h.config.CheckInterval)
	defer ticker.Stop()

	// Perform initial health check
	if err := h.performHealthCheck(ctx); err != nil {
		return fmt.Errorf("initial health check failed: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := h.performHealthCheck(ctx); err != nil {
				fmt.Printf("Health check error: %v\n", err)
			}
		}
	}
}

// performHealthCheck executes all registered health checks
func (h *HealthMonitor) performHealthCheck(ctx context.Context) error {
	startTime := time.Now()

	// Create context with timeout
	checkCtx, cancel := context.WithTimeout(ctx, h.config.Timeout)
	defer cancel()

	// Execute health checks concurrently
	results := h.runHealthChecks(checkCtx)

	// Generate health report
	report := h.generateHealthReport(results, startTime)

	// Update last health report
	h.mutex.Lock()
	h.lastHealthReport = report
	h.mutex.Unlock()

	// Record metrics
	h.recordHealthMetrics(report)

	// Check for alerts
	if h.config.AlertOnFailure {
		h.checkAndSendAlerts(report)
	}

	return nil
}

// runHealthChecks executes all health checks concurrently
func (h *HealthMonitor) runHealthChecks(ctx context.Context) map[string]*ComponentHealth {
	results := make(map[string]*ComponentHealth)
	resultsChan := make(chan *ComponentHealth, len(h.checks))

	var wg sync.WaitGroup

	// Start all health checks
	h.mutex.RLock()
	for _, checker := range h.checks {
		wg.Add(1)
		go func(c HealthChecker) {
			defer wg.Done()

			// Create context with checker-specific timeout
			checkTimeout := c.Timeout()
			if checkTimeout == 0 {
				checkTimeout = h.config.Timeout
			}

			checkCtx, cancel := context.WithTimeout(ctx, checkTimeout)
			defer cancel()

			// Execute health check
			result := c.Check(checkCtx)
			resultsChan <- result
		}(checker)
	}
	h.mutex.RUnlock()

	// Wait for all checks to complete
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// Collect results
	for result := range resultsChan {
		results[result.Component] = result
	}

	return results
}

// generateHealthReport creates a comprehensive health report
func (h *HealthMonitor) generateHealthReport(results map[string]*ComponentHealth, startTime time.Time) *HealthReport {
	report := &HealthReport{
		Timestamp:  time.Now(),
		Version:    "1.0.0", // Should come from build info
		Uptime:     time.Since(startTime),
		Components: results,
		Summary:    h.calculateHealthSummary(results),
	}

	// Determine overall status
	report.OverallStatus = h.determineOverallStatus(results)

	// Generate alerts if needed
	if h.config.AlertOnFailure {
		report.Alerts = h.generateAlerts(results)
	}

	return report
}

// calculateHealthSummary calculates summary statistics
func (h *HealthMonitor) calculateHealthSummary(results map[string]*ComponentHealth) *HealthSummary {
	summary := &HealthSummary{
		TotalComponents: len(results),
		Scores:         make(map[string]float64),
		Trends:         make(map[string]HealthTrend),
	}

	// Count components by status
	for _, result := range results {
		switch result.Status {
		case StatusHealthy:
			summary.HealthyCount++
		case StatusWarning:
			summary.WarningCount++
		case StatusCritical:
			summary.CriticalCount++
		case StatusFailed:
			summary.FailedCount++
		}
	}

	// Calculate health scores
	if summary.TotalComponents > 0 {
		summary.Scores["overall"] = float64(summary.HealthyCount) / float64(summary.TotalComponents) * 100
		summary.Scores["availability"] = float64(summary.TotalComponents-summary.FailedCount) / float64(summary.TotalComponents) * 100
	}

	return summary
}

// determineOverallStatus determines the overall system status
func (h *HealthMonitor) determineOverallStatus(results map[string]*ComponentHealth) string {
	hasCritical := false
	hasWarning := false
	hasFailed := false

	for _, result := range results {
		switch result.Status {
		case StatusFailed:
			hasFailed = true
		case StatusCritical:
			hasCritical = true
		case StatusWarning:
			hasWarning = true
		}
	}

	if hasFailed {
		return string(StatusFailed)
	}
	if hasCritical {
		return string(StatusCritical)
	}
	if hasWarning {
		return string(StatusWarning)
	}
	return string(StatusHealthy)
}

// GetCurrentHealth returns the latest health report
func (h *HealthMonitor) GetCurrentHealth() *HealthReport {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	return h.lastHealthReport
}

// GetHealthHandler returns an HTTP handler for health checks
func (h *HealthMonitor) GetHealthHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		report := h.GetCurrentHealth()
		if report == nil {
			http.Error(w, "Health check not available", http.StatusServiceUnavailable)
			return
		}

		w.Header().Set("Content-Type", "application/json")

		// Set status code based on overall health
		switch report.OverallStatus {
		case string(StatusHealthy):
			w.WriteHeader(http.StatusOK)
		case string(StatusWarning):
			w.WriteHeader(http.StatusOK) // Warning still returns 200
		case string(StatusCritical):
			w.WriteHeader(http.StatusServiceUnavailable)
		case string(StatusFailed):
			w.WriteHeader(http.StatusServiceUnavailable)
		default:
			w.WriteHeader(http.StatusServiceUnavailable)
		}

		if err := json.NewEncoder(w).Encode(report); err != nil {
			http.Error(w, "Failed to encode health report", http.StatusInternalServerError)
		}
	}
}

// GetReadinessHandler returns an HTTP handler for readiness checks
func (h *HealthMonitor) GetReadinessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		report := h.GetCurrentHealth()
		if report == nil {
			http.Error(w, "Readiness check not available", http.StatusServiceUnavailable)
			return
		}

		// Check critical components for readiness
		ready := true
		criticalComponents := []string{"database", "memory", "cpu"}

		for _, component := range criticalComponents {
			if health, exists := report.Components[component]; exists {
				if health.Status == StatusFailed || health.Status == StatusCritical {
					ready = false
					break
				}
			}
		}

		if ready {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"ready"}`))
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"status":"not_ready"}`))
		}
	}
}

// GetLivenessHandler returns an HTTP handler for liveness checks
func (h *HealthMonitor) GetLivenessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Simple liveness check - just verify the service is responding
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"alive"}`))
	}
}

// recordHealthMetrics records health metrics for monitoring
func (h *HealthMonitor) recordHealthMetrics(report *HealthReport) {
	// Record overall metrics
	h.metrics.RecordGauge("system.health.overall_score", report.Summary.Scores["overall"])
	h.metrics.RecordGauge("system.health.availability_score", report.Summary.Scores["availability"])
	h.metrics.RecordGauge("system.health.healthy_components", float64(report.Summary.HealthyCount))
	h.metrics.RecordGauge("system.health.warning_components", float64(report.Summary.WarningCount))
	h.metrics.RecordGauge("system.health.critical_components", float64(report.Summary.CriticalCount))
	h.metrics.RecordGauge("system.health.failed_components", float64(report.Summary.FailedCount))

	// Record component-specific metrics
	for name, health := range report.Components {
		labels := map[string]string{"component": name}

		// Convert status to numeric value for metrics
		var statusValue float64
		switch health.Status {
		case StatusHealthy:
			statusValue = 1
		case StatusWarning:
			statusValue = 0.75
		case StatusCritical:
			statusValue = 0.5
		case StatusFailed:
			statusValue = 0
		default:
			statusValue = 0
		}

		h.metrics.RecordGaugeWithLabels("component.health.status", statusValue, labels)
		h.metrics.RecordGaugeWithLabels("component.health.duration_ms", float64(health.Duration.Milliseconds()), labels)

		// Record component-specific metrics
		for metricName, value := range health.Metrics {
			metricLabels := map[string]string{"component": name, "metric": metricName}
			h.metrics.RecordGaugeWithLabels("component.metric", value, metricLabels)
		}
	}
}

// checkAndSendAlerts checks for alert conditions and sends notifications
func (h *HealthMonitor) checkAndSendAlerts(report *HealthReport) {
	alerts := h.generateAlerts(report.Components)

	for _, alert := range alerts {
		if err := h.alertManager.SendAlert(alert); err != nil {
			fmt.Printf("Failed to send alert: %v\n", err)
		}
	}
}

// generateAlerts creates alerts based on health status
func (h *HealthMonitor) generateAlerts(results map[string]*ComponentHealth) []Alert {
	var alerts []Alert

	for _, health := range results {
		if health.Status == StatusCritical || health.Status == StatusFailed {
			alert := Alert{
				Level:     AlertLevelCritical,
				Component: health.Component,
				Message:   fmt.Sprintf("Component %s is %s: %s", health.Component, health.Status, health.Message),
				Timestamp: health.Timestamp,
				Details:   health.Details,
			}

			if health.Error != "" {
				alert.Details["error"] = health.Error
			}

			alerts = append(alerts, alert)
		}
	}

	return alerts
}

// Default health checker implementations would be in separate files:
// - database_health_checker.go
// - memory_health_checker.go
// - cpu_health_checker.go
// - disk_health_checker.go
// - network_health_checker.go
// - service_health_checker.go
// - embedding_service_checker.go
// - llm_service_checker.go
// - cache_health_checker.go

// Alert represents a system alert
type Alert struct {
	Level     AlertLevel             `json:"level"`
	Component string                 `json:"component"`
	Message   string                 `json:"message"`
	Timestamp time.Time              `json:"timestamp"`
	Details   map[string]interface{} `json:"details,omitempty"`
}

// AlertLevel represents alert severity
type AlertLevel string

const (
	AlertLevelInfo     AlertLevel = "info"
	AlertLevelWarning  AlertLevel = "warning"
	AlertLevelCritical AlertLevel = "critical"
)

// AlertManager handles alert notifications
type AlertManager struct {
	config    *AlertConfig
	notifiers map[string]AlertNotifier
}

// AlertConfig contains alert configuration
type AlertConfig struct {
	EnabledNotifiers []string          `json:"enabled_notifiers"`
	RateLimits       map[string]string `json:"rate_limits"`
	Channels         map[string]string `json:"channels"`
}

// AlertNotifier defines interface for alert notifications
type AlertNotifier interface {
	SendAlert(alert Alert) error
	Name() string
}

// NewAlertManager creates a new alert manager
func NewAlertManager() *AlertManager {
	return &AlertManager{
		notifiers: make(map[string]AlertNotifier),
	}
}

// SendAlert sends an alert through configured notifiers
func (a *AlertManager) SendAlert(alert Alert) error {
	for _, notifier := range a.notifiers {
		if err := notifier.SendAlert(alert); err != nil {
			return fmt.Errorf("failed to send alert via %s: %w", notifier.Name(), err)
		}
	}
	return nil
}

// MetricsCollector handles metrics collection
type MetricsCollector struct {
	// Implementation would integrate with Prometheus or similar
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{}
}

// RecordGauge records a gauge metric
func (m *MetricsCollector) RecordGauge(name string, value float64) {
	// Implementation would record to metrics backend
}

// RecordGaugeWithLabels records a gauge metric with labels
func (m *MetricsCollector) RecordGaugeWithLabels(name string, value float64, labels map[string]string) {
	// Implementation would record to metrics backend with labels
}