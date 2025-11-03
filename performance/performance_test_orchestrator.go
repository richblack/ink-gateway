package performance

import (
	"context"
	"fmt"
	"log"
	"runtime"
	"semantic-text-processor/config"
	"semantic-text-processor/models"
	"semantic-text-processor/services"
	"sync"
	"time"
)

// PerformanceTestOrchestrator coordinates all performance testing activities
type PerformanceTestOrchestrator struct {
	config           *config.Config
	logger           *log.Logger
	dataGenerator    *DataGenerationService
	loadExecutor     *LoadTestExecutor
	metricsCollector *MetricsCollector
	optimizer        *OptimizationAnalyzer
	monitor          *ContinuousMonitor
	reportGenerator  *ReportGenerator
	services         *services.ServiceContainer
}

// PerformanceTestConfig holds configuration for performance tests
type PerformanceTestConfig struct {
	DatasetSize            int           `json:"dataset_size"`
	MaxConcurrentUsers     int           `json:"max_concurrent_users"`
	TestDuration           time.Duration `json:"test_duration"`
	RampUpTime             time.Duration `json:"ramp_up_time"`
	CooldownTime           time.Duration `json:"cool_down_time"`
	EnableRegression       bool          `json:"enable_regression"`
	EnableResourceMonitor  bool          `json:"enable_resource_monitor"`
	SlowQueryThreshold     time.Duration `json:"slow_query_threshold"`
	MemoryLimitMB          int           `json:"memory_limit_mb"`
	CPUUsageThreshold      float64       `json:"cpu_usage_threshold"`
	GenerateMillionRecords bool          `json:"generate_million_records"`
}

// NewPerformanceTestOrchestrator creates a new performance test orchestrator
func NewPerformanceTestOrchestrator(cfg *config.Config, services *services.ServiceContainer, logger *log.Logger) *PerformanceTestOrchestrator {
	return &PerformanceTestOrchestrator{
		config:           cfg,
		logger:           logger,
		services:         services,
		dataGenerator:    NewDataGenerationService(logger),
		loadExecutor:     NewLoadTestExecutor(services, logger),
		metricsCollector: NewMetricsCollector(logger),
		optimizer:        NewOptimizationAnalyzer(logger),
		monitor:          NewContinuousMonitor(services, logger),
		reportGenerator:  NewReportGenerator(logger),
	}
}

// ExecuteComprehensivePerformanceTest runs the complete performance test suite
func (pto *PerformanceTestOrchestrator) ExecuteComprehensivePerformanceTest(ctx context.Context, testConfig *PerformanceTestConfig) (*models.ComprehensivePerformanceReport, error) {
	pto.logger.Printf("Starting comprehensive performance test with config: %+v", testConfig)
	startTime := time.Now()

	report := &models.ComprehensivePerformanceReport{
		StartTime:   startTime,
		TestConfig:  *testConfig,
		Environment: pto.captureEnvironmentInfo(),
	}

	// Phase 1: Generate large-scale test data
	if testConfig.GenerateMillionRecords {
		pto.logger.Printf("Phase 1: Generating million-level test dataset...")
		dataGenResult, err := pto.generateLargeScaleData(ctx, testConfig.DatasetSize)
		if err != nil {
			return nil, fmt.Errorf("data generation failed: %w", err)
		}
		report.DataGeneration = *dataGenResult
	}

	// Phase 2: Execute baseline performance tests
	pto.logger.Printf("Phase 2: Running baseline performance tests...")
	baselineResult, err := pto.runBaselineTests(ctx, testConfig)
	if err != nil {
		return nil, fmt.Errorf("baseline tests failed: %w", err)
	}
	report.BaselinePerformance = *baselineResult

	// Phase 3: Execute load and stress tests
	pto.logger.Printf("Phase 3: Running load and stress tests...")
	loadTestResult, err := pto.runLoadTests(ctx, testConfig)
	if err != nil {
		return nil, fmt.Errorf("load tests failed: %w", err)
	}
	report.LoadTestResults = *loadTestResult

	// Phase 4: Performance optimization analysis
	pto.logger.Printf("Phase 4: Analyzing performance and generating optimizations...")
	optimizationResult, err := pto.analyzeAndOptimize(ctx, report)
	if err != nil {
		pto.logger.Printf("Optimization analysis failed: %v", err)
	} else {
		report.OptimizationAnalysis = *optimizationResult
	}

	// Phase 5: Regression testing if enabled
	if testConfig.EnableRegression {
		pto.logger.Printf("Phase 5: Running regression tests...")
		regressionResult, err := pto.runRegressionTests(ctx, testConfig)
		if err != nil {
			pto.logger.Printf("Regression tests failed: %v", err)
		} else {
			report.RegressionResults = regressionResult
		}
	}

	// Phase 6: Resource utilization analysis
	if testConfig.EnableResourceMonitor {
		pto.logger.Printf("Phase 6: Analyzing resource utilization...")
		resourceResult := pto.analyzeResourceUtilization(ctx, testConfig)
		report.ResourceUtilization = *resourceResult
	}

	report.EndTime = time.Now()
	report.TotalDuration = report.EndTime.Sub(startTime)

	// Generate final recommendations
	report.Recommendations = pto.generateFinalRecommendations(report)

	pto.logger.Printf("Performance test completed in %v", report.TotalDuration)
	return report, nil
}

// generateLargeScaleData creates million-level test datasets
func (pto *PerformanceTestOrchestrator) generateLargeScaleData(ctx context.Context, targetSize int) (*models.DataGenerationResult, error) {
	startTime := time.Now()

	result := &models.DataGenerationResult{
		StartTime:  startTime,
		TargetSize: targetSize,
	}

	// Generate chunks in batches to manage memory
	batchSize := 10000
	batches := targetSize / batchSize
	if targetSize%batchSize != 0 {
		batches++
	}

	var totalGenerated int
	var totalErrors int

	pto.logger.Printf("Generating %d records in %d batches of %d", targetSize, batches, batchSize)

	for batch := 0; batch < batches; batch++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		currentBatchSize := batchSize
		if batch == batches-1 && targetSize%batchSize != 0 {
			currentBatchSize = targetSize % batchSize
		}

		batchStart := time.Now()
		generated, errors := pto.dataGenerator.GenerateBatchData(ctx, currentBatchSize, batch*batchSize)
		batchDuration := time.Since(batchStart)

		totalGenerated += generated
		totalErrors += errors

		pto.logger.Printf("Batch %d/%d: Generated %d records in %v (errors: %d)",
			batch+1, batches, generated, batchDuration, errors)

		// Add some delay between batches to prevent overwhelming the system
		if batch < batches-1 {
			time.Sleep(100 * time.Millisecond)
		}
	}

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(startTime)
	result.RecordsGenerated = totalGenerated
	result.GenerationErrors = totalErrors
	result.RecordsPerSecond = float64(totalGenerated) / result.Duration.Seconds()

	// Verify data integrity
	verificationResult := pto.dataGenerator.VerifyDataIntegrity(ctx, targetSize)
	result.DataIntegrityCheck = verificationResult

	return result, nil
}

// runBaselineTests executes baseline performance measurements
func (pto *PerformanceTestOrchestrator) runBaselineTests(ctx context.Context, testConfig *PerformanceTestConfig) (*models.BaselinePerformanceResult, error) {
	// Use existing benchmark suite but with enhanced measurements
	benchmarkSuite := services.NewSearchBenchmarkSuite(pto.services.SearchService, pto.logger)

	// Run comprehensive benchmarks
	benchmarkResults, err := benchmarkSuite.RunComprehensiveBenchmarks(ctx)
	if err != nil {
		return nil, fmt.Errorf("benchmark execution failed: %w", err)
	}

	// Collect additional baseline metrics
	dbStats, err := pto.collectDatabaseStats(ctx)
	if err != nil {
		pto.logger.Printf("Failed to collect database stats: %v", err)
	}

	cacheStats := pto.collectCacheStats(ctx)

	return &models.BaselinePerformanceResult{
		BenchmarkResults: *benchmarkResults,
		DatabaseStats:    dbStats,
		CacheStats:       cacheStats,
		MemoryUsage:      pto.getCurrentMemoryUsage(),
		SystemInfo:       pto.getSystemInfo(),
	}, nil
}

// runLoadTests executes progressive load testing
func (pto *PerformanceTestOrchestrator) runLoadTests(ctx context.Context, testConfig *PerformanceTestConfig) (*models.LoadTestResult, error) {
	loadTestConfig := &LoadTestConfig{
		MaxConcurrentUsers: testConfig.MaxConcurrentUsers,
		TestDuration:       testConfig.TestDuration,
		RampUpTime:         testConfig.RampUpTime,
		CooldownTime:       testConfig.CooldownTime,
		ProgressiveLoad:    true,
		LoadSteps:          []int{1, 5, 10, 25, 50, 100, testConfig.MaxConcurrentUsers},
	}

	return pto.loadExecutor.ExecuteProgressiveLoadTest(ctx, loadTestConfig)
}

// analyzeAndOptimize performs performance analysis and generates optimization recommendations
func (pto *PerformanceTestOrchestrator) analyzeAndOptimize(ctx context.Context, report *models.ComprehensivePerformanceReport) (*models.OptimizationAnalysisResult, error) {
	analysisData := &OptimizationAnalysisData{
		BaselineResults: &report.BaselinePerformance,
		LoadTestResults: &report.LoadTestResults,
		ResourceUsage:   &report.ResourceUtilization,
	}

	return pto.optimizer.AnalyzePerformanceAndOptimize(ctx, analysisData)
}

// runRegressionTests compares performance against previous benchmarks
func (pto *PerformanceTestOrchestrator) runRegressionTests(ctx context.Context, testConfig *PerformanceTestConfig) (*models.RegressionTestResult, error) {
	// Load historical performance data
	historicalData, err := pto.loadHistoricalPerformanceData()
	if err != nil {
		return nil, fmt.Errorf("failed to load historical data: %w", err)
	}

	// Run current tests with same parameters
	currentResults, err := pto.runBaselineTests(ctx, testConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to run current baseline: %w", err)
	}

	// Compare results
	comparison := pto.comparePerformanceResults(historicalData, currentResults)

	return &models.RegressionTestResult{
		HistoricalBaseline: historicalData,
		CurrentResults:     *currentResults,
		Comparison:         comparison,
		RegressionDetected: comparison.HasRegression,
		ImprovementAreas:   comparison.ImprovementAreas,
	}, nil
}

// analyzeResourceUtilization monitors system resource usage during tests
func (pto *PerformanceTestOrchestrator) analyzeResourceUtilization(ctx context.Context, testConfig *PerformanceTestConfig) *models.ResourceUtilizationResult {
	return &models.ResourceUtilizationResult{
		PeakMemoryUsage:    pto.monitor.GetPeakMemoryUsage(),
		AverageCPUUsage:    pto.monitor.GetAverageCPUUsage(),
		DiskIOStats:        pto.monitor.GetDiskIOStats(),
		NetworkIOStats:     pto.monitor.GetNetworkIOStats(),
		GarbageCollection:  pto.monitor.GetGCStats(),
		DatabaseConnections: pto.monitor.GetDatabaseConnectionStats(),
		ResourceThresholds: map[string]interface{}{
			"memory_limit_mb":      testConfig.MemoryLimitMB,
			"cpu_usage_threshold":  testConfig.CPUUsageThreshold,
			"slow_query_threshold": testConfig.SlowQueryThreshold,
		},
	}
}

// Helper methods

func (pto *PerformanceTestOrchestrator) captureEnvironmentInfo() models.EnvironmentInfo {
	return models.EnvironmentInfo{
		GoVersion:       runtime.Version(),
		NumCPU:          runtime.NumCPU(),
		NumGoroutines:   runtime.NumGoroutine(),
		MemoryStats:     pto.getCurrentMemoryUsage(),
		OSArch:          runtime.GOOS + "/" + runtime.GOARCH,
		Timestamp:       time.Now(),
	}
}

func (pto *PerformanceTestOrchestrator) getCurrentMemoryUsage() models.MemoryStats {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return models.MemoryStats{
		Alloc:        m.Alloc,
		TotalAlloc:   m.TotalAlloc,
		Sys:          m.Sys,
		Lookups:      m.Lookups,
		Mallocs:      m.Mallocs,
		Frees:        m.Frees,
		HeapAlloc:    m.HeapAlloc,
		HeapSys:      m.HeapSys,
		HeapIdle:     m.HeapIdle,
		HeapInuse:    m.HeapInuse,
		HeapReleased: m.HeapReleased,
		HeapObjects:  m.HeapObjects,
		StackInuse:   m.StackInuse,
		StackSys:     m.StackSys,
		NumGC:        m.NumGC,
		PauseTotalNs: m.PauseTotalNs,
	}
}

func (pto *PerformanceTestOrchestrator) getSystemInfo() models.SystemInfo {
	return models.SystemInfo{
		CPUCount:      runtime.NumCPU(),
		MaxProcs:      runtime.GOMAXPROCS(0),
		Goroutines:    runtime.NumGoroutine(),
		CGOCalls:      runtime.NumCgoCall(),
	}
}

func (pto *PerformanceTestOrchestrator) collectDatabaseStats(ctx context.Context) (models.DatabaseStats, error) {
	// Implement database statistics collection
	// This would include connection pool stats, query counts, etc.
	return models.DatabaseStats{
		ActiveConnections:    0, // Placeholder - implement actual collection
		IdleConnections:      0,
		TotalConnections:     0,
		QueryCount:           0,
		SlowQueryCount:       0,
		AverageQueryDuration: 0,
	}, nil
}

func (pto *PerformanceTestOrchestrator) collectCacheStats(ctx context.Context) models.CacheStats {
	if pto.services.CacheService != nil {
		return pto.services.CacheService.GetStats()
	}
	return models.CacheStats{}
}

func (pto *PerformanceTestOrchestrator) loadHistoricalPerformanceData() (*models.BaselinePerformanceResult, error) {
	// Implement loading historical performance data from storage
	// This would typically load from a file or database
	return nil, fmt.Errorf("historical data loading not implemented")
}

func (pto *PerformanceTestOrchestrator) comparePerformanceResults(historical, current *models.BaselinePerformanceResult) models.PerformanceComparison {
	// Implement performance comparison logic
	return models.PerformanceComparison{
		HasRegression:      false,
		ImprovementAreas:   []string{},
		DegradationAreas:   []string{},
		PerformanceDelta:   map[string]float64{},
	}
}

func (pto *PerformanceTestOrchestrator) generateFinalRecommendations(report *models.ComprehensivePerformanceReport) []models.PerformanceRecommendation {
	recommendations := []models.PerformanceRecommendation{}

	// Add recommendations based on performance analysis
	if report.OptimizationAnalysis.SlowQueries != nil && len(report.OptimizationAnalysis.SlowQueries) > 0 {
		recommendations = append(recommendations, models.PerformanceRecommendation{
			Category:    "Database",
			Priority:    "High",
			Title:       "Optimize Slow Queries",
			Description: fmt.Sprintf("Found %d slow queries that exceed threshold", len(report.OptimizationAnalysis.SlowQueries)),
			Actions:     []string{"Add database indexes", "Query optimization", "Connection pooling review"},
		})
	}

	if report.BaselinePerformance.CacheStats.HitRate < 0.8 {
		recommendations = append(recommendations, models.PerformanceRecommendation{
			Category:    "Cache",
			Priority:    "Medium",
			Title:       "Improve Cache Hit Rate",
			Description: fmt.Sprintf("Current cache hit rate: %.2f%%, target: >80%%", report.BaselinePerformance.CacheStats.HitRate*100),
			Actions:     []string{"Increase cache size", "Optimize cache TTL", "Implement cache warming"},
		})
	}

	return recommendations
}

// SavePerformanceReport saves the comprehensive performance report
func (pto *PerformanceTestOrchestrator) SavePerformanceReport(report *models.ComprehensivePerformanceReport, filepath string) error {
	return pto.reportGenerator.SaveReport(report, filepath)
}

// GetPerformanceSummary returns a summary of the most recent performance test
func (pto *PerformanceTestOrchestrator) GetPerformanceSummary() (*models.PerformanceSummary, error) {
	return pto.reportGenerator.GetLatestSummary()
}