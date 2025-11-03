package performance

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"semantic-text-processor/models"
	"time"
)

// ReportGenerator generates comprehensive performance reports
type ReportGenerator struct {
	logger        *log.Logger
	reportPath    string
	latestReport  *models.ComprehensivePerformanceReport
}

// NewReportGenerator creates a new report generator
func NewReportGenerator(logger *log.Logger) *ReportGenerator {
	// Create reports directory if it doesn't exist
	reportPath := "./performance_reports"
	os.MkdirAll(reportPath, 0755)

	return &ReportGenerator{
		logger:     logger,
		reportPath: reportPath,
	}
}

// SaveReport saves a comprehensive performance report to file
func (rg *ReportGenerator) SaveReport(report *models.ComprehensivePerformanceReport, customPath string) error {
	rg.latestReport = report

	// Determine output path
	outputPath := customPath
	if outputPath == "" {
		timestamp := report.StartTime.Format("20060102_150405")
		filename := fmt.Sprintf("performance_report_%s.json", timestamp)
		outputPath = filepath.Join(rg.reportPath, filename)
	}

	// Ensure directory exists
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Marshal report to JSON
	jsonData, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal report: %w", err)
	}

	// Write to file
	if err := os.WriteFile(outputPath, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write report file: %w", err)
	}

	rg.logger.Printf("Performance report saved to: %s", outputPath)

	// Also save a summary report
	summaryPath := filepath.Join(dir, "latest_summary.json")
	if err := rg.saveSummaryReport(report, summaryPath); err != nil {
		rg.logger.Printf("Failed to save summary report: %v", err)
	}

	// Save human-readable report
	textPath := filepath.Join(dir, fmt.Sprintf("performance_report_%s.txt",
		report.StartTime.Format("20060102_150405")))
	if err := rg.saveTextReport(report, textPath); err != nil {
		rg.logger.Printf("Failed to save text report: %v", err)
	}

	return nil
}

// saveSummaryReport saves a condensed summary of the performance report
func (rg *ReportGenerator) saveSummaryReport(report *models.ComprehensivePerformanceReport, path string) error {
	summary := rg.generateSummary(report)

	jsonData, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal summary: %w", err)
	}

	return os.WriteFile(path, jsonData, 0644)
}

// saveTextReport saves a human-readable text version of the report
func (rg *ReportGenerator) saveTextReport(report *models.ComprehensivePerformanceReport, path string) error {
	content := rg.generateTextReport(report)
	return os.WriteFile(path, []byte(content), 0644)
}

// generateSummary creates a summary from the comprehensive report
func (rg *ReportGenerator) generateSummary(report *models.ComprehensivePerformanceReport) *models.PerformanceSummary {
	summary := &models.PerformanceSummary{
		TestDate:           report.StartTime,
		OverallScore:       rg.calculateOverallScore(report),
		KeyMetrics:         make(map[string]interface{}),
		CriticalIssues:     []string{},
		TopRecommendations: []string{},
	}

	// Key metrics
	if len(report.LoadTestResults.LoadSteps) > 0 {
		lastStep := report.LoadTestResults.LoadSteps[len(report.LoadTestResults.LoadSteps)-1]
		summary.KeyMetrics["max_concurrent_users"] = lastStep.UserCount
		summary.KeyMetrics["peak_qps"] = lastStep.QPS
		summary.KeyMetrics["error_rate"] = lastStep.ErrorRate
		summary.KeyMetrics["avg_response_time"] = lastStep.AvgResponseTime.String()
	}

	summary.KeyMetrics["total_test_duration"] = report.TotalDuration.String()
	summary.KeyMetrics["data_generated"] = report.DataGeneration.RecordsGenerated
	summary.KeyMetrics["cache_hit_rate"] = report.BaselinePerformance.CacheStats.HitRate

	// Critical issues
	if report.LoadTestResults.OverallStats.ErrorRate > 0.05 {
		summary.CriticalIssues = append(summary.CriticalIssues,
			fmt.Sprintf("High error rate: %.2f%%", report.LoadTestResults.OverallStats.ErrorRate*100))
	}

	if report.BaselinePerformance.CacheStats.HitRate < 0.8 {
		summary.CriticalIssues = append(summary.CriticalIssues,
			fmt.Sprintf("Low cache hit rate: %.2f%%", report.BaselinePerformance.CacheStats.HitRate*100))
	}

	if len(report.OptimizationAnalysis.SlowQueries) > 0 {
		summary.CriticalIssues = append(summary.CriticalIssues,
			fmt.Sprintf("Found %d slow query patterns", len(report.OptimizationAnalysis.SlowQueries)))
	}

	// Top recommendations
	for i, rec := range report.Recommendations {
		if i >= 5 { // Top 5 recommendations
			break
		}
		summary.TopRecommendations = append(summary.TopRecommendations, rec.Title)
	}

	return summary
}

// calculateOverallScore computes an overall performance score
func (rg *ReportGenerator) calculateOverallScore(report *models.ComprehensivePerformanceReport) float64 {
	score := 100.0 // Start with perfect score

	// Deduct for high error rates
	if report.LoadTestResults.OverallStats.ErrorRate > 0 {
		score -= report.LoadTestResults.OverallStats.ErrorRate * 100 * 2 // 2x penalty
	}

	// Deduct for slow response times
	avgResponseMs := float64(report.LoadTestResults.OverallStats.AvgResponseTime.Milliseconds())
	if avgResponseMs > 500 {
		score -= (avgResponseMs - 500) / 50 // Deduct 1 point per 50ms over 500ms
	}

	// Deduct for low cache hit rate
	if report.BaselinePerformance.CacheStats.HitRate < 0.8 {
		score -= (0.8 - report.BaselinePerformance.CacheStats.HitRate) * 50
	}

	// Deduct for slow queries
	score -= float64(len(report.OptimizationAnalysis.SlowQueries)) * 5

	// Ensure score is between 0 and 100
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	return score
}

// generateTextReport creates a human-readable text report
func (rg *ReportGenerator) generateTextReport(report *models.ComprehensivePerformanceReport) string {
	var content string

	content += "=== PERFORMANCE TEST REPORT ===\n\n"
	content += fmt.Sprintf("Test Date: %s\n", report.StartTime.Format("2006-01-02 15:04:05"))
	content += fmt.Sprintf("Test Duration: %s\n", report.TotalDuration.String())
	content += fmt.Sprintf("Overall Score: %.1f/100\n\n", rg.calculateOverallScore(report))

	// Environment Information
	content += "=== ENVIRONMENT ===\n"
	content += fmt.Sprintf("Go Version: %s\n", report.Environment.GoVersion)
	content += fmt.Sprintf("OS/Arch: %s\n", report.Environment.OSArch)
	content += fmt.Sprintf("CPU Count: %d\n", report.Environment.NumCPU)
	content += fmt.Sprintf("Memory Allocated: %s\n", formatBytes(report.Environment.MemoryStats.Alloc))
	content += "\n"

	// Data Generation Results
	if report.DataGeneration.RecordsGenerated > 0 {
		content += "=== DATA GENERATION ===\n"
		content += fmt.Sprintf("Records Generated: %d\n", report.DataGeneration.RecordsGenerated)
		content += fmt.Sprintf("Generation Rate: %.1f records/sec\n", report.DataGeneration.RecordsPerSecond)
		content += fmt.Sprintf("Data Quality Score: %.2f%%\n", report.DataGeneration.DataIntegrityCheck.DataQualityScore*100)
		content += fmt.Sprintf("Generation Errors: %d\n", report.DataGeneration.GenerationErrors)
		content += "\n"
	}

	// Baseline Performance
	content += "=== BASELINE PERFORMANCE ===\n"
	content += fmt.Sprintf("Semantic Search Avg: %s\n", report.BaselinePerformance.BenchmarkResults.SemanticSearch.AverageResponseTime.String())
	content += fmt.Sprintf("Tag Search Avg: %s\n", report.BaselinePerformance.BenchmarkResults.TagSearch.AverageResponseTime.String())
	content += fmt.Sprintf("Cache Hit Rate: %.2f%%\n", report.BaselinePerformance.CacheStats.HitRate*100)
	content += fmt.Sprintf("Overall Success Rate: %.2f%%\n",
		(report.BaselinePerformance.BenchmarkResults.SemanticSearch.SuccessRate+
		report.BaselinePerformance.BenchmarkResults.TagSearch.SuccessRate+
		report.BaselinePerformance.BenchmarkResults.FullTextSearch.SuccessRate)/3*100)
	content += "\n"

	// Load Test Results
	content += "=== LOAD TEST RESULTS ===\n"
	content += fmt.Sprintf("Total Requests: %d\n", report.LoadTestResults.OverallStats.TotalRequests)
	content += fmt.Sprintf("Total Errors: %d (%.2f%%)\n",
		report.LoadTestResults.OverallStats.TotalErrors,
		report.LoadTestResults.OverallStats.ErrorRate*100)
	content += fmt.Sprintf("Average Response Time: %s\n", report.LoadTestResults.OverallStats.AvgResponseTime.String())
	content += fmt.Sprintf("95th Percentile: %s\n", report.LoadTestResults.OverallStats.P95ResponseTime.String())
	content += fmt.Sprintf("99th Percentile: %s\n", report.LoadTestResults.OverallStats.P99ResponseTime.String())

	if len(report.LoadTestResults.LoadSteps) > 0 {
		lastStep := report.LoadTestResults.LoadSteps[len(report.LoadTestResults.LoadSteps)-1]
		content += fmt.Sprintf("Peak Concurrent Users: %d\n", lastStep.UserCount)
		content += fmt.Sprintf("Peak QPS: %.1f\n", lastStep.QPS)
	}
	content += "\n"

	// Resource Utilization
	content += "=== RESOURCE UTILIZATION ===\n"
	content += fmt.Sprintf("Peak Memory Usage: %s\n", formatBytes(report.ResourceUtilization.PeakMemoryUsage))
	content += fmt.Sprintf("Average CPU Usage: %.1f%%\n", report.ResourceUtilization.AverageCPUUsage)
	content += fmt.Sprintf("GC Count: %d\n", report.ResourceUtilization.GarbageCollection.NumGC)
	if report.ResourceUtilization.DatabaseConnections != nil {
		content += fmt.Sprintf("DB Connections: %d/%d\n",
			report.ResourceUtilization.DatabaseConnections.ActiveConnections,
			report.ResourceUtilization.DatabaseConnections.MaxConnections)
	}
	content += "\n"

	// Optimization Analysis
	if len(report.OptimizationAnalysis.SlowQueries) > 0 {
		content += "=== SLOW QUERIES ===\n"
		for i, query := range report.OptimizationAnalysis.SlowQueries {
			if i >= 5 { // Top 5 slow queries
				break
			}
			content += fmt.Sprintf("- %s: %s (count: %d, impact: %.2f)\n",
				query.QueryType, query.AvgDuration.String(), query.Count, query.Impact)
		}
		content += "\n"
	}

	// Recommendations
	if len(report.Recommendations) > 0 {
		content += "=== TOP RECOMMENDATIONS ===\n"
		for i, rec := range report.Recommendations {
			if i >= 10 { // Top 10 recommendations
				break
			}
			content += fmt.Sprintf("%d. [%s] %s\n", i+1, rec.Priority, rec.Title)
			content += fmt.Sprintf("   %s\n", rec.Description)
			if len(rec.Actions) > 0 {
				content += fmt.Sprintf("   Actions: %s\n", rec.Actions[0])
			}
			content += "\n"
		}
	}

	// Summary
	content += "=== SUMMARY ===\n"
	if rg.calculateOverallScore(report) >= 80 {
		content += "✅ System performance is GOOD\n"
	} else if rg.calculateOverallScore(report) >= 60 {
		content += "⚠️  System performance needs IMPROVEMENT\n"
	} else {
		content += "❌ System performance is POOR and needs immediate attention\n"
	}

	criticalCount := 0
	for _, rec := range report.Recommendations {
		if rec.Priority == "critical" || rec.Priority == "high" {
			criticalCount++
		}
	}

	if criticalCount > 0 {
		content += fmt.Sprintf("⚠️  %d critical/high priority issues found\n", criticalCount)
	}

	content += fmt.Sprintf("📊 Overall Performance Score: %.1f/100\n", rg.calculateOverallScore(report))

	return content
}

// GetLatestSummary returns a summary of the most recent performance test
func (rg *ReportGenerator) GetLatestSummary() (*models.PerformanceSummary, error) {
	if rg.latestReport == nil {
		return nil, fmt.Errorf("no performance reports available")
	}

	return rg.generateSummary(rg.latestReport), nil
}

// LoadReport loads a performance report from file
func (rg *ReportGenerator) LoadReport(filePath string) (*models.ComprehensivePerformanceReport, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read report file: %w", err)
	}

	var report models.ComprehensivePerformanceReport
	if err := json.Unmarshal(data, &report); err != nil {
		return nil, fmt.Errorf("failed to unmarshal report: %w", err)
	}

	return &report, nil
}

// ListReports returns a list of available performance reports
func (rg *ReportGenerator) ListReports() ([]string, error) {
	files, err := os.ReadDir(rg.reportPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read reports directory: %w", err)
	}

	var reports []string
	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".json" {
			reports = append(reports, filepath.Join(rg.reportPath, file.Name()))
		}
	}

	return reports, nil
}

// CompareReports compares two performance reports
func (rg *ReportGenerator) CompareReports(report1, report2 *models.ComprehensivePerformanceReport) *models.PerformanceComparison {
	comparison := &models.PerformanceComparison{
		HasRegression:      false,
		ImprovementAreas:   []string{},
		DegradationAreas:   []string{},
		PerformanceDelta:   make(map[string]float64),
	}

	// Compare overall scores
	score1 := rg.calculateOverallScore(report1)
	score2 := rg.calculateOverallScore(report2)
	comparison.PerformanceDelta["overall_score"] = score2 - score1

	// Compare response times
	rt1 := float64(report1.LoadTestResults.OverallStats.AvgResponseTime.Milliseconds())
	rt2 := float64(report2.LoadTestResults.OverallStats.AvgResponseTime.Milliseconds())
	comparison.PerformanceDelta["avg_response_time_ms"] = rt2 - rt1

	if rt2 > rt1*1.1 { // 10% degradation threshold
		comparison.HasRegression = true
		comparison.DegradationAreas = append(comparison.DegradationAreas, "Response time increased")
	} else if rt2 < rt1*0.9 { // 10% improvement threshold
		comparison.ImprovementAreas = append(comparison.ImprovementAreas, "Response time improved")
	}

	// Compare error rates
	er1 := report1.LoadTestResults.OverallStats.ErrorRate
	er2 := report2.LoadTestResults.OverallStats.ErrorRate
	comparison.PerformanceDelta["error_rate"] = er2 - er1

	if er2 > er1+0.01 { // 1% increase threshold
		comparison.HasRegression = true
		comparison.DegradationAreas = append(comparison.DegradationAreas, "Error rate increased")
	} else if er2 < er1-0.01 { // 1% improvement threshold
		comparison.ImprovementAreas = append(comparison.ImprovementAreas, "Error rate improved")
	}

	// Compare cache hit rates
	chr1 := report1.BaselinePerformance.CacheStats.HitRate
	chr2 := report2.BaselinePerformance.CacheStats.HitRate
	comparison.PerformanceDelta["cache_hit_rate"] = chr2 - chr1

	if chr2 < chr1-0.05 { // 5% decrease threshold
		comparison.HasRegression = true
		comparison.DegradationAreas = append(comparison.DegradationAreas, "Cache hit rate decreased")
	} else if chr2 > chr1+0.05 { // 5% improvement threshold
		comparison.ImprovementAreas = append(comparison.ImprovementAreas, "Cache hit rate improved")
	}

	return comparison
}

// Helper function to format bytes in human-readable format
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