package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"semantic-text-processor/config"
	"semantic-text-processor/performance"
	"semantic-text-processor/services"
	"time"
)

func main() {
	// Command line flags
	var (
		configFile         = flag.String("config", "", "Configuration file path")
		datasetSize        = flag.Int("dataset-size", 100000, "Number of test records to generate")
		maxUsers           = flag.Int("max-users", 50, "Maximum concurrent users for load testing")
		testDuration       = flag.Duration("duration", 5*time.Minute, "Duration for each load test step")
		generateMillion    = flag.Bool("generate-million", false, "Generate million-level test dataset")
		enableRegression   = flag.Bool("regression", false, "Enable regression testing")
		enableMonitoring   = flag.Bool("monitoring", true, "Enable resource monitoring")
		outputPath         = flag.String("output", "", "Custom output path for reports")
		slowQueryThreshold = flag.Duration("slow-threshold", 500*time.Millisecond, "Slow query threshold")
		memoryLimitMB      = flag.Int("memory-limit", 1024, "Memory limit in MB")
		cpuThreshold       = flag.Float64("cpu-threshold", 80.0, "CPU usage threshold percentage")
		help               = flag.Bool("help", false, "Show help message")
	)

	flag.Parse()

	if *help {
		showHelp()
		return
	}

	// Load configuration
	cfg := config.LoadConfig()
	if *configFile != "" {
		// Load from custom config file if specified
		log.Printf("Loading configuration from: %s", *configFile)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Configuration validation failed: %v", err)
	}

	// Set up logger
	logger := log.New(os.Stdout, "[PERF-TEST] ", log.LstdFlags|log.Lshortfile)
	logger.Printf("Starting performance testing suite...")

	// Create service container
	serviceFactory := services.NewServiceFactory(cfg)
	serviceContainer, err := serviceFactory.CreateServices()
	if err != nil {
		log.Fatalf("Failed to create services: %v", err)
	}

	// Create performance test orchestrator
	orchestrator := performance.NewPerformanceTestOrchestrator(cfg, serviceContainer, logger)

	// Create test configuration
	testConfig := &performance.PerformanceTestConfig{
		DatasetSize:            *datasetSize,
		MaxConcurrentUsers:     *maxUsers,
		TestDuration:           *testDuration,
		RampUpTime:             *testDuration / 10, // 10% of test duration for ramp-up
		CooldownTime:           30 * time.Second,
		EnableRegression:       *enableRegression,
		EnableResourceMonitor:  *enableMonitoring,
		SlowQueryThreshold:     *slowQueryThreshold,
		MemoryLimitMB:          *memoryLimitMB,
		CPUUsageThreshold:      *cpuThreshold,
		GenerateMillionRecords: *generateMillion,
	}

	// Adjust dataset size for million-level testing
	if *generateMillion {
		testConfig.DatasetSize = 1000000
		logger.Printf("Million-level testing enabled: generating %d records", testConfig.DatasetSize)
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Hour)
	defer cancel()

	// Execute comprehensive performance test
	logger.Printf("Executing comprehensive performance test with config: %+v", testConfig)

	startTime := time.Now()
	report, err := orchestrator.ExecuteComprehensivePerformanceTest(ctx, testConfig)
	if err != nil {
		log.Fatalf("Performance test failed: %v", err)
	}

	totalTime := time.Since(startTime)
	logger.Printf("Performance test completed successfully in %v", totalTime)

	// Save the report
	if err := orchestrator.SavePerformanceReport(report, *outputPath); err != nil {
		log.Fatalf("Failed to save performance report: %v", err)
	}

	// Print summary to console
	printSummary(report, logger)

	// Generate final exit code based on results
	exitCode := generateExitCode(report)
	if exitCode != 0 {
		logger.Printf("Performance test completed with issues (exit code: %d)", exitCode)
	} else {
		logger.Printf("Performance test completed successfully")
	}

	os.Exit(exitCode)
}

func showHelp() {
	fmt.Println("Performance Testing Suite for Semantic Text Processor")
	fmt.Println("====================================================")
	fmt.Println()
	fmt.Println("This tool performs comprehensive performance testing including:")
	fmt.Println("• Large-scale data generation and testing")
	fmt.Println("• Progressive load testing with concurrent users")
	fmt.Println("• Performance optimization analysis")
	fmt.Println("• Resource utilization monitoring")
	fmt.Println("• Regression testing")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  performance-test [flags]")
	fmt.Println()
	fmt.Println("Flags:")
	flag.PrintDefaults()
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  # Basic performance test")
	fmt.Println("  performance-test")
	fmt.Println()
	fmt.Println("  # Million-level dataset with high load")
	fmt.Println("  performance-test -generate-million -max-users 100 -duration 10m")
	fmt.Println()
	fmt.Println("  # Regression testing with custom thresholds")
	fmt.Println("  performance-test -regression -slow-threshold 200ms -cpu-threshold 70")
	fmt.Println()
	fmt.Println("  # Save report to custom location")
	fmt.Println("  performance-test -output /path/to/report.json")
	fmt.Println()
}

func printSummary(report *performance.ComprehensivePerformanceReport, logger *log.Logger) {
	logger.Printf("=== PERFORMANCE TEST SUMMARY ===")
	logger.Printf("Test Duration: %v", report.TotalDuration)
	logger.Printf("Test Configuration:")
	logger.Printf("  - Dataset Size: %d records", report.TestConfig.DatasetSize)
	logger.Printf("  - Max Concurrent Users: %d", report.TestConfig.MaxConcurrentUsers)
	logger.Printf("  - Test Duration per Step: %v", report.TestConfig.TestDuration)

	if report.DataGeneration.RecordsGenerated > 0 {
		logger.Printf("Data Generation:")
		logger.Printf("  - Records Generated: %d", report.DataGeneration.RecordsGenerated)
		logger.Printf("  - Generation Rate: %.1f records/sec", report.DataGeneration.RecordsPerSecond)
		logger.Printf("  - Data Quality: %.2f%%", report.DataGeneration.DataIntegrityCheck.DataQualityScore*100)
	}

	logger.Printf("Baseline Performance:")
	logger.Printf("  - Semantic Search: %v", report.BaselinePerformance.BenchmarkResults.SemanticSearch.AverageResponseTime)
	logger.Printf("  - Tag Search: %v", report.BaselinePerformance.BenchmarkResults.TagSearch.AverageResponseTime)
	logger.Printf("  - Cache Hit Rate: %.2f%%", report.BaselinePerformance.CacheStats.HitRate*100)

	logger.Printf("Load Test Results:")
	logger.Printf("  - Total Requests: %d", report.LoadTestResults.OverallStats.TotalRequests)
	logger.Printf("  - Error Rate: %.2f%%", report.LoadTestResults.OverallStats.ErrorRate*100)
	logger.Printf("  - Avg Response Time: %v", report.LoadTestResults.OverallStats.AvgResponseTime)
	logger.Printf("  - 95th Percentile: %v", report.LoadTestResults.OverallStats.P95ResponseTime)

	if len(report.LoadTestResults.LoadSteps) > 0 {
		lastStep := report.LoadTestResults.LoadSteps[len(report.LoadTestResults.LoadSteps)-1]
		logger.Printf("  - Peak Users: %d", lastStep.UserCount)
		logger.Printf("  - Peak QPS: %.1f", lastStep.QPS)
	}

	logger.Printf("Resource Utilization:")
	logger.Printf("  - Peak Memory: %s", formatBytes(report.ResourceUtilization.PeakMemoryUsage))
	logger.Printf("  - Avg CPU: %.1f%%", report.ResourceUtilization.AverageCPUUsage)
	logger.Printf("  - GC Count: %d", report.ResourceUtilization.GarbageCollection.NumGC)

	if len(report.OptimizationAnalysis.SlowQueries) > 0 {
		logger.Printf("Performance Issues:")
		logger.Printf("  - Slow Query Patterns: %d", len(report.OptimizationAnalysis.SlowQueries))
		for i, query := range report.OptimizationAnalysis.SlowQueries {
			if i >= 3 { // Show top 3
				break
			}
			logger.Printf("    • %s: %v (impact: %.2f)", query.QueryType, query.AvgDuration, query.Impact)
		}
	}

	if len(report.Recommendations) > 0 {
		logger.Printf("Top Recommendations:")
		for i, rec := range report.Recommendations {
			if i >= 5 { // Show top 5
				break
			}
			logger.Printf("  %d. [%s] %s", i+1, rec.Priority, rec.Title)
		}
	}

	// Calculate overall score
	score := calculateOverallScore(report)
	logger.Printf("Overall Performance Score: %.1f/100", score)

	if score >= 80 {
		logger.Printf("✅ Performance Status: EXCELLENT")
	} else if score >= 60 {
		logger.Printf("⚠️  Performance Status: NEEDS IMPROVEMENT")
	} else {
		logger.Printf("❌ Performance Status: POOR - Immediate attention required")
	}
}

func calculateOverallScore(report *performance.ComprehensivePerformanceReport) float64 {
	score := 100.0

	// Deduct for high error rates
	if report.LoadTestResults.OverallStats.ErrorRate > 0 {
		score -= report.LoadTestResults.OverallStats.ErrorRate * 100 * 2
	}

	// Deduct for slow response times
	avgResponseMs := float64(report.LoadTestResults.OverallStats.AvgResponseTime.Milliseconds())
	if avgResponseMs > 500 {
		score -= (avgResponseMs - 500) / 50
	}

	// Deduct for low cache hit rate
	if report.BaselinePerformance.CacheStats.HitRate < 0.8 {
		score -= (0.8 - report.BaselinePerformance.CacheStats.HitRate) * 50
	}

	// Deduct for slow queries
	score -= float64(len(report.OptimizationAnalysis.SlowQueries)) * 5

	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	return score
}

func generateExitCode(report *performance.ComprehensivePerformanceReport) int {
	// Exit code 0: Success
	// Exit code 1: Performance issues found
	// Exit code 2: Critical performance problems

	score := calculateOverallScore(report)

	if score < 40 {
		return 2 // Critical issues
	} else if score < 70 {
		return 1 // Performance issues
	}

	// Check for critical issues
	if report.LoadTestResults.OverallStats.ErrorRate > 0.1 { // 10% error rate
		return 2
	}

	criticalRecommendations := 0
	for _, rec := range report.Recommendations {
		if rec.Priority == "critical" {
			criticalRecommendations++
		}
	}

	if criticalRecommendations > 0 {
		return 2
	}

	highPriorityRecommendations := 0
	for _, rec := range report.Recommendations {
		if rec.Priority == "high" {
			highPriorityRecommendations++
		}
	}

	if highPriorityRecommendations > 5 {
		return 1
	}

	return 0 // Success
}

func formatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}