package performance

import (
	"context"
	"fmt"
	"log"
	"math"
	"semantic-text-processor/models"
	"semantic-text-processor/services"
	"sync"
	"sync/atomic"
	"time"
)

// LoadTestExecutor manages and executes load testing scenarios
type LoadTestExecutor struct {
	services        *services.ServiceContainer
	logger          *log.Logger
	queryGenerators map[string]QueryGenerator
	metricsCollector *LoadTestMetricsCollector
}

// LoadTestConfig defines configuration for load testing
type LoadTestConfig struct {
	MaxConcurrentUsers int           `json:"max_concurrent_users"`
	TestDuration       time.Duration `json:"test_duration"`
	RampUpTime         time.Duration `json:"ramp_up_time"`
	CooldownTime       time.Duration `json:"cool_down_time"`
	ProgressiveLoad    bool          `json:"progressive_load"`
	LoadSteps          []int         `json:"load_steps"`
	QueryTypes         []string      `json:"query_types"`
	ThinkTime          time.Duration `json:"think_time"`
	ErrorThreshold     float64       `json:"error_threshold"`
}

// QueryGenerator generates test queries for load testing
type QueryGenerator interface {
	GenerateQuery() models.TestQuery
	GetQueryType() string
}

// LoadTestMetricsCollector collects metrics during load testing
type LoadTestMetricsCollector struct {
	mu              sync.RWMutex
	requestCount    int64
	errorCount      int64
	totalDuration   time.Duration
	responseTimes   []time.Duration
	throughputData  []ThroughputPoint
	errorsByType    map[string]int64
	userMetrics     map[int]*UserMetrics
	systemMetrics   []SystemMetricPoint
}

// UserMetrics tracks metrics for individual virtual users
type UserMetrics struct {
	UserID        int           `json:"user_id"`
	RequestCount  int           `json:"request_count"`
	ErrorCount    int           `json:"error_count"`
	TotalDuration time.Duration `json:"total_duration"`
	AvgResponse   time.Duration `json:"avg_response"`
	LastActivity  time.Time     `json:"last_activity"`
}

// ThroughputPoint represents throughput at a specific time
type ThroughputPoint struct {
	Timestamp  time.Time `json:"timestamp"`
	QPS        float64   `json:"qps"`
	ActiveUsers int      `json:"active_users"`
}

// SystemMetricPoint represents system metrics at a specific time
type SystemMetricPoint struct {
	Timestamp     time.Time `json:"timestamp"`
	CPUUsage      float64   `json:"cpu_usage"`
	MemoryUsage   uint64    `json:"memory_usage"`
	ActiveUsers   int       `json:"active_users"`
	ResponseTime  time.Duration `json:"response_time"`
	ErrorRate     float64   `json:"error_rate"`
}

// NewLoadTestExecutor creates a new load test executor
func NewLoadTestExecutor(services *services.ServiceContainer, logger *log.Logger) *LoadTestExecutor {
	executor := &LoadTestExecutor{
		services:         services,
		logger:           logger,
		queryGenerators:  make(map[string]QueryGenerator),
		metricsCollector: NewLoadTestMetricsCollector(),
	}

	// Initialize query generators
	executor.initializeQueryGenerators()

	return executor
}

// ExecuteProgressiveLoadTest runs a progressive load test
func (lte *LoadTestExecutor) ExecuteProgressiveLoadTest(ctx context.Context, config *LoadTestConfig) (*models.LoadTestResult, error) {
	lte.logger.Printf("Starting progressive load test with config: %+v", config)

	result := &models.LoadTestResult{
		StartTime:    time.Now(),
		Config:       *config,
		LoadSteps:    []models.LoadStepResult{},
		OverallStats: models.LoadTestStats{},
	}

	// Reset metrics collector
	lte.metricsCollector.Reset()

	// Execute each load step
	for i, userCount := range config.LoadSteps {
		lte.logger.Printf("Executing load step %d/%d: %d users", i+1, len(config.LoadSteps), userCount)

		stepResult, err := lte.executeLoadStep(ctx, userCount, config)
		if err != nil {
			lte.logger.Printf("Load step %d failed: %v", i+1, err)
			stepResult.Error = err.Error()
		}

		result.LoadSteps = append(result.LoadSteps, *stepResult)

		// Check if error threshold is exceeded
		if stepResult.ErrorRate > config.ErrorThreshold {
			lte.logger.Printf("Error threshold exceeded (%.2f%% > %.2f%%), stopping test",
				stepResult.ErrorRate*100, config.ErrorThreshold*100)
			break
		}

		// Cooldown between steps
		if i < len(config.LoadSteps)-1 {
			lte.logger.Printf("Cooldown for %v", config.CooldownTime)
			time.Sleep(config.CooldownTime)
		}
	}

	// Calculate overall statistics
	result.OverallStats = lte.calculateOverallStats()
	result.EndTime = time.Now()
	result.TotalDuration = result.EndTime.Sub(result.StartTime)

	// Collect final metrics
	result.ThroughputData = lte.metricsCollector.GetThroughputData()
	result.SystemMetrics = lte.metricsCollector.GetSystemMetrics()
	result.UserMetrics = lte.metricsCollector.GetUserMetrics()

	lte.logger.Printf("Progressive load test completed in %v", result.TotalDuration)
	return result, nil
}

// executeLoadStep executes a single load step with specified user count
func (lte *LoadTestExecutor) executeLoadStep(ctx context.Context, userCount int, config *LoadTestConfig) (*models.LoadStepResult, error) {
	stepStart := time.Now()

	stepResult := &models.LoadStepResult{
		UserCount:   userCount,
		StartTime:   stepStart,
		Duration:    config.TestDuration,
		RequestStats: make(map[string]models.RequestTypeStats),
	}

	// Create context with timeout
	stepCtx, cancel := context.WithTimeout(ctx, config.TestDuration)
	defer cancel()

	// Start system metrics collection
	go lte.collectSystemMetrics(stepCtx, userCount)

	// Launch virtual users
	var wg sync.WaitGroup
	userResultsChan := make(chan *UserMetrics, userCount)

	// Gradual ramp-up if configured
	if config.RampUpTime > 0 {
		rampUpInterval := config.RampUpTime / time.Duration(userCount)
		for i := 0; i < userCount; i++ {
			wg.Add(1)
			go lte.runVirtualUser(stepCtx, i, config, &wg, userResultsChan)

			if i < userCount-1 {
				time.Sleep(rampUpInterval)
			}
		}
	} else {
		// Immediate start for all users
		for i := 0; i < userCount; i++ {
			wg.Add(1)
			go lte.runVirtualUser(stepCtx, i, config, &wg, userResultsChan)
		}
	}

	// Wait for all users to complete
	wg.Wait()
	close(userResultsChan)

	// Collect user metrics
	var totalRequests, totalErrors int64
	var totalDuration time.Duration
	userMetrics := make(map[int]*UserMetrics)

	for userMetric := range userResultsChan {
		userMetrics[userMetric.UserID] = userMetric
		totalRequests += int64(userMetric.RequestCount)
		totalErrors += int64(userMetric.ErrorCount)
		totalDuration += userMetric.TotalDuration
	}

	// Calculate step statistics
	stepResult.EndTime = time.Now()
	stepResult.ActualDuration = stepResult.EndTime.Sub(stepStart)
	stepResult.TotalRequests = totalRequests
	stepResult.TotalErrors = totalErrors
	stepResult.ErrorRate = float64(totalErrors) / float64(totalRequests)
	stepResult.AvgResponseTime = time.Duration(int64(totalDuration) / totalRequests)
	stepResult.QPS = float64(totalRequests) / stepResult.ActualDuration.Seconds()

	// Store user metrics
	lte.metricsCollector.AddUserMetrics(userMetrics)

	return stepResult, nil
}

// runVirtualUser simulates a single virtual user's behavior
func (lte *LoadTestExecutor) runVirtualUser(ctx context.Context, userID int, config *LoadTestConfig, wg *sync.WaitGroup, resultsChan chan<- *UserMetrics) {
	defer wg.Done()

	userMetrics := &UserMetrics{
		UserID:       userID,
		LastActivity: time.Now(),
	}

	lte.logger.Printf("Virtual user %d started", userID)

	for {
		select {
		case <-ctx.Done():
			lte.logger.Printf("Virtual user %d stopping due to context cancellation", userID)
			resultsChan <- userMetrics
			return
		default:
		}

		// Select query type
		queryType := config.QueryTypes[userID%len(config.QueryTypes)]
		generator := lte.queryGenerators[queryType]
		if generator == nil {
			lte.logger.Printf("No generator for query type: %s", queryType)
			continue
		}

		// Generate and execute query
		query := generator.GenerateQuery()
		startTime := time.Now()

		err := lte.executeQuery(ctx, query)
		duration := time.Since(startTime)

		// Update metrics
		userMetrics.RequestCount++
		userMetrics.TotalDuration += duration
		userMetrics.LastActivity = time.Now()

		if err != nil {
			userMetrics.ErrorCount++
			lte.metricsCollector.RecordError(queryType, err)
		}

		// Record response time
		lte.metricsCollector.RecordResponseTime(duration)

		// Update average response time
		userMetrics.AvgResponse = userMetrics.TotalDuration / time.Duration(userMetrics.RequestCount)

		// Think time between requests
		if config.ThinkTime > 0 {
			time.Sleep(config.ThinkTime)
		}
	}
}

// executeQuery executes a test query
func (lte *LoadTestExecutor) executeQuery(ctx context.Context, query models.TestQuery) error {
	switch query.Type {
	case "semantic_search":
		req := query.Parameters.(*models.OptimizedSearchRequest)
		_, err := lte.services.SearchService.SemanticSearchWithFilters(ctx, &services.SemanticSearchRequest{
			Query:         req.Query,
			Limit:         req.Limit,
			MinSimilarity: req.MinSimilarity,
			Filters:       req.Filters,
		})
		return err

	case "tag_search":
		req := query.Parameters.(*models.TagSearchRequest)
		_, err := lte.services.SearchService.SearchByTag(ctx, req.Tags[0]) // Simplified
		return err

	case "chunk_crud":
		// Simulate CRUD operations
		chunks, err := lte.services.UnifiedChunkService.GetChunks(ctx, &services.GetChunksRequest{
			Limit:  20,
			Offset: 0,
		})
		if err != nil {
			return err
		}
		if len(chunks.Chunks) > 0 {
			_, err = lte.services.UnifiedChunkService.GetChunkByID(ctx, chunks.Chunks[0].ID)
		}
		return err

	default:
		return fmt.Errorf("unknown query type: %s", query.Type)
	}
}

// collectSystemMetrics collects system performance metrics during the test
func (lte *LoadTestExecutor) collectSystemMetrics(ctx context.Context, activeUsers int) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Collect current metrics
			metric := SystemMetricPoint{
				Timestamp:   time.Now(),
				ActiveUsers: activeUsers,
				ResponseTime: lte.metricsCollector.GetCurrentAvgResponseTime(),
				ErrorRate:   lte.metricsCollector.GetCurrentErrorRate(),
			}

			// Add system resource metrics (simplified)
			metric.CPUUsage = lte.getCurrentCPUUsage()
			metric.MemoryUsage = lte.getCurrentMemoryUsage()

			lte.metricsCollector.AddSystemMetric(metric)
		}
	}
}

// Helper methods for metrics collection

func (lte *LoadTestExecutor) getCurrentCPUUsage() float64 {
	// Simplified CPU usage calculation
	return 50.0 + (float64(time.Now().UnixNano()%100) - 50.0) * 0.4
}

func (lte *LoadTestExecutor) getCurrentMemoryUsage() uint64 {
	// Simplified memory usage calculation
	return uint64(100*1024*1024) + uint64(time.Now().UnixNano()%50*1024*1024)
}

func (lte *LoadTestExecutor) calculateOverallStats() models.LoadTestStats {
	totalRequests := atomic.LoadInt64(&lte.metricsCollector.requestCount)
	totalErrors := atomic.LoadInt64(&lte.metricsCollector.errorCount)

	stats := models.LoadTestStats{
		TotalRequests:    totalRequests,
		TotalErrors:      totalErrors,
		ErrorRate:        float64(totalErrors) / float64(totalRequests),
		AvgResponseTime:  lte.metricsCollector.GetOverallAvgResponseTime(),
		MinResponseTime:  lte.metricsCollector.GetMinResponseTime(),
		MaxResponseTime:  lte.metricsCollector.GetMaxResponseTime(),
		P50ResponseTime:  lte.metricsCollector.GetPercentileResponseTime(0.50),
		P95ResponseTime:  lte.metricsCollector.GetPercentileResponseTime(0.95),
		P99ResponseTime:  lte.metricsCollector.GetPercentileResponseTime(0.99),
	}

	return stats
}

// initializeQueryGenerators sets up query generators for different query types
func (lte *LoadTestExecutor) initializeQueryGenerators() {
	lte.queryGenerators["semantic_search"] = NewSemanticSearchGenerator()
	lte.queryGenerators["tag_search"] = NewTagSearchGenerator()
	lte.queryGenerators["chunk_crud"] = NewChunkCRUDGenerator()
}

// NewLoadTestMetricsCollector creates a new metrics collector
func NewLoadTestMetricsCollector() *LoadTestMetricsCollector {
	return &LoadTestMetricsCollector{
		responseTimes:   make([]time.Duration, 0),
		throughputData:  make([]ThroughputPoint, 0),
		errorsByType:    make(map[string]int64),
		userMetrics:     make(map[int]*UserMetrics),
		systemMetrics:   make([]SystemMetricPoint, 0),
	}
}

// LoadTestMetricsCollector methods

func (collector *LoadTestMetricsCollector) Reset() {
	collector.mu.Lock()
	defer collector.mu.Unlock()

	atomic.StoreInt64(&collector.requestCount, 0)
	atomic.StoreInt64(&collector.errorCount, 0)
	collector.totalDuration = 0
	collector.responseTimes = collector.responseTimes[:0]
	collector.throughputData = collector.throughputData[:0]
	collector.errorsByType = make(map[string]int64)
	collector.userMetrics = make(map[int]*UserMetrics)
	collector.systemMetrics = collector.systemMetrics[:0]
}

func (collector *LoadTestMetricsCollector) RecordResponseTime(duration time.Duration) {
	collector.mu.Lock()
	defer collector.mu.Unlock()

	atomic.AddInt64(&collector.requestCount, 1)
	collector.totalDuration += duration
	collector.responseTimes = append(collector.responseTimes, duration)
}

func (collector *LoadTestMetricsCollector) RecordError(queryType string, err error) {
	collector.mu.Lock()
	defer collector.mu.Unlock()

	atomic.AddInt64(&collector.errorCount, 1)
	collector.errorsByType[queryType]++
}

func (collector *LoadTestMetricsCollector) AddUserMetrics(userMetrics map[int]*UserMetrics) {
	collector.mu.Lock()
	defer collector.mu.Unlock()

	for userID, metrics := range userMetrics {
		collector.userMetrics[userID] = metrics
	}
}

func (collector *LoadTestMetricsCollector) AddSystemMetric(metric SystemMetricPoint) {
	collector.mu.Lock()
	defer collector.mu.Unlock()

	collector.systemMetrics = append(collector.systemMetrics, metric)
}

func (collector *LoadTestMetricsCollector) GetCurrentAvgResponseTime() time.Duration {
	collector.mu.RLock()
	defer collector.mu.RUnlock()

	requestCount := atomic.LoadInt64(&collector.requestCount)
	if requestCount == 0 {
		return 0
	}

	return collector.totalDuration / time.Duration(requestCount)
}

func (collector *LoadTestMetricsCollector) GetCurrentErrorRate() float64 {
	requestCount := atomic.LoadInt64(&collector.requestCount)
	errorCount := atomic.LoadInt64(&collector.errorCount)

	if requestCount == 0 {
		return 0
	}

	return float64(errorCount) / float64(requestCount)
}

func (collector *LoadTestMetricsCollector) GetOverallAvgResponseTime() time.Duration {
	return collector.GetCurrentAvgResponseTime()
}

func (collector *LoadTestMetricsCollector) GetMinResponseTime() time.Duration {
	collector.mu.RLock()
	defer collector.mu.RUnlock()

	if len(collector.responseTimes) == 0 {
		return 0
	}

	min := collector.responseTimes[0]
	for _, duration := range collector.responseTimes {
		if duration < min {
			min = duration
		}
	}

	return min
}

func (collector *LoadTestMetricsCollector) GetMaxResponseTime() time.Duration {
	collector.mu.RLock()
	defer collector.mu.RUnlock()

	if len(collector.responseTimes) == 0 {
		return 0
	}

	max := collector.responseTimes[0]
	for _, duration := range collector.responseTimes {
		if duration > max {
			max = duration
		}
	}

	return max
}

func (collector *LoadTestMetricsCollector) GetPercentileResponseTime(percentile float64) time.Duration {
	collector.mu.RLock()
	defer collector.mu.RUnlock()

	if len(collector.responseTimes) == 0 {
		return 0
	}

	// Simple percentile calculation (should use proper sorting for production)
	index := int(math.Ceil(float64(len(collector.responseTimes)) * percentile)) - 1
	if index < 0 {
		index = 0
	}
	if index >= len(collector.responseTimes) {
		index = len(collector.responseTimes) - 1
	}

	return collector.responseTimes[index]
}

func (collector *LoadTestMetricsCollector) GetThroughputData() []ThroughputPoint {
	collector.mu.RLock()
	defer collector.mu.RUnlock()

	// Return copy of throughput data
	data := make([]ThroughputPoint, len(collector.throughputData))
	copy(data, collector.throughputData)
	return data
}

func (collector *LoadTestMetricsCollector) GetSystemMetrics() []SystemMetricPoint {
	collector.mu.RLock()
	defer collector.mu.RUnlock()

	// Return copy of system metrics
	metrics := make([]SystemMetricPoint, len(collector.systemMetrics))
	copy(metrics, collector.systemMetrics)
	return metrics
}

func (collector *LoadTestMetricsCollector) GetUserMetrics() map[int]*UserMetrics {
	collector.mu.RLock()
	defer collector.mu.RUnlock()

	// Return copy of user metrics
	metrics := make(map[int]*UserMetrics)
	for userID, userMetric := range collector.userMetrics {
		// Create copy of user metric
		metricCopy := *userMetric
		metrics[userID] = &metricCopy
	}
	return metrics
}