package performance

import (
	"context"
	"fmt"
	"log"
	"math"
	"semantic-text-processor/models"
	"sort"
	"time"
)

// OptimizationAnalyzer analyzes performance data and generates optimization recommendations
type OptimizationAnalyzer struct {
	logger           *log.Logger
	thresholds       *PerformanceThresholds
	optimizationRules []OptimizationRule
}

// PerformanceThresholds defines thresholds for performance analysis
type PerformanceThresholds struct {
	SlowQueryThreshold        time.Duration `json:"slow_query_threshold"`
	LowCacheHitRateThreshold  float64       `json:"low_cache_hit_rate_threshold"`
	HighErrorRateThreshold    float64       `json:"high_error_rate_threshold"`
	LowThroughputThreshold    float64       `json:"low_throughput_threshold"`
	HighMemoryUsageThreshold  uint64        `json:"high_memory_usage_threshold"`
	HighCPUUsageThreshold     float64       `json:"high_cpu_usage_threshold"`
	DatabaseConnectionLimit   int           `json:"database_connection_limit"`
}

// OptimizationRule defines a rule for generating optimization recommendations
type OptimizationRule struct {
	Name        string                                                       `json:"name"`
	Category    string                                                       `json:"category"`
	Priority    string                                                       `json:"priority"`
	Condition   func(data *OptimizationAnalysisData) bool                    `json:"-"`
	Generate    func(data *OptimizationAnalysisData) models.OptimizationRecommendation `json:"-"`
}

// OptimizationAnalysisData contains all data needed for optimization analysis
type OptimizationAnalysisData struct {
	BaselineResults *models.BaselinePerformanceResult
	LoadTestResults *models.LoadTestResult
	ResourceUsage   *models.ResourceUtilizationResult
	SlowQueries     []models.SlowQueryAnalysis
	CacheAnalysis   *models.CacheAnalysisResult
	IndexAnalysis   *models.IndexAnalysisResult
}

// SlowQueryPattern represents a pattern of slow queries
type SlowQueryPattern struct {
	Pattern         string        `json:"pattern"`
	Count           int           `json:"count"`
	AvgDuration     time.Duration `json:"avg_duration"`
	MaxDuration     time.Duration `json:"max_duration"`
	QueryType       string        `json:"query_type"`
	Tables          []string      `json:"tables"`
	SuggestedIndexes []string     `json:"suggested_indexes"`
}

// NewOptimizationAnalyzer creates a new optimization analyzer
func NewOptimizationAnalyzer(logger *log.Logger) *OptimizationAnalyzer {
	analyzer := &OptimizationAnalyzer{
		logger:     logger,
		thresholds: getDefaultPerformanceThresholds(),
	}

	analyzer.initializeOptimizationRules()
	return analyzer
}

// AnalyzePerformanceAndOptimize performs comprehensive performance analysis
func (oa *OptimizationAnalyzer) AnalyzePerformanceAndOptimize(ctx context.Context, data *OptimizationAnalysisData) (*models.OptimizationAnalysisResult, error) {
	oa.logger.Printf("Starting performance optimization analysis...")

	result := &models.OptimizationAnalysisResult{
		AnalysisTimestamp: time.Now(),
		Recommendations:   []models.OptimizationRecommendation{},
		SlowQueries:       []models.SlowQueryAnalysis{},
		CacheOptimizations: []models.CacheOptimizationSuggestion{},
		IndexSuggestions:  []models.IndexSuggestion{},
		ConfigurationTuning: []models.ConfigurationTuning{},
		PerformanceMetrics: make(map[string]interface{}),
	}

	// Analyze slow queries
	oa.logger.Printf("Analyzing slow queries...")
	slowQueryAnalysis, err := oa.analyzeSlowQueries(ctx, data)
	if err != nil {
		oa.logger.Printf("Slow query analysis failed: %v", err)
	} else {
		result.SlowQueries = slowQueryAnalysis
	}

	// Analyze cache performance
	oa.logger.Printf("Analyzing cache performance...")
	cacheOptimizations, err := oa.analyzeCachePerformance(ctx, data)
	if err != nil {
		oa.logger.Printf("Cache analysis failed: %v", err)
	} else {
		result.CacheOptimizations = cacheOptimizations
	}

	// Analyze database indexes
	oa.logger.Printf("Analyzing database indexes...")
	indexSuggestions, err := oa.analyzeIndexOptimizations(ctx, data)
	if err != nil {
		oa.logger.Printf("Index analysis failed: %v", err)
	} else {
		result.IndexSuggestions = indexSuggestions
	}

	// Analyze configuration tuning
	oa.logger.Printf("Analyzing configuration tuning...")
	configTuning, err := oa.analyzeConfigurationTuning(ctx, data)
	if err != nil {
		oa.logger.Printf("Configuration analysis failed: %v", err)
	} else {
		result.ConfigurationTuning = configTuning
	}

	// Generate optimization recommendations
	oa.logger.Printf("Generating optimization recommendations...")
	recommendations := oa.generateOptimizationRecommendations(data)
	result.Recommendations = recommendations

	// Calculate performance metrics
	result.PerformanceMetrics = oa.calculatePerformanceMetrics(data)

	// Prioritize recommendations
	result.Recommendations = oa.prioritizeRecommendations(result.Recommendations)

	oa.logger.Printf("Optimization analysis completed with %d recommendations", len(result.Recommendations))
	return result, nil
}

// analyzeSlowQueries identifies and analyzes slow queries
func (oa *OptimizationAnalyzer) analyzeSlowQueries(ctx context.Context, data *OptimizationAnalysisData) ([]models.SlowQueryAnalysis, error) {
	var slowQueries []models.SlowQueryAnalysis

	// Analyze baseline performance for slow queries
	if data.BaselineResults != nil {
		baselineSlowQueries := oa.extractSlowQueriesFromBaseline(data.BaselineResults)
		slowQueries = append(slowQueries, baselineSlowQueries...)
	}

	// Analyze load test results for slow queries
	if data.LoadTestResults != nil {
		loadTestSlowQueries := oa.extractSlowQueriesFromLoadTest(data.LoadTestResults)
		slowQueries = append(slowQueries, loadTestSlowQueries...)
	}

	// Group similar queries and identify patterns
	patterns := oa.identifySlowQueryPatterns(slowQueries)

	// Convert patterns to analysis results
	var analysisResults []models.SlowQueryAnalysis
	for _, pattern := range patterns {
		analysis := models.SlowQueryAnalysis{
			QueryPattern:     pattern.Pattern,
			Count:           pattern.Count,
			AvgDuration:     pattern.AvgDuration,
			MaxDuration:     pattern.MaxDuration,
			QueryType:       pattern.QueryType,
			AffectedTables:  pattern.Tables,
			OptimizationSuggestions: oa.generateQueryOptimizationSuggestions(pattern),
			Impact:          oa.calculateQueryImpact(pattern),
		}
		analysisResults = append(analysisResults, analysis)
	}

	// Sort by impact (highest first)
	sort.Slice(analysisResults, func(i, j int) bool {
		return analysisResults[i].Impact > analysisResults[j].Impact
	})

	return analysisResults, nil
}

// analyzeCachePerformance analyzes cache performance and suggests optimizations
func (oa *OptimizationAnalyzer) analyzeCachePerformance(ctx context.Context, data *OptimizationAnalysisData) ([]models.CacheOptimizationSuggestion, error) {
	var suggestions []models.CacheOptimizationSuggestion

	if data.BaselineResults == nil || data.BaselineResults.CacheStats.HitRate == 0 {
		return suggestions, nil
	}

	cacheStats := data.BaselineResults.CacheStats

	// Low cache hit rate analysis
	if cacheStats.HitRate < oa.thresholds.LowCacheHitRateThreshold {
		suggestions = append(suggestions, models.CacheOptimizationSuggestion{
			Type:        "hit_rate_improvement",
			Priority:    "high",
			Description: fmt.Sprintf("Cache hit rate is %.2f%%, below threshold of %.2f%%",
				cacheStats.HitRate*100, oa.thresholds.LowCacheHitRateThreshold*100),
			Actions: []string{
				"Increase cache size",
				"Optimize cache TTL settings",
				"Implement cache warming strategies",
				"Review cache eviction policies",
			},
			EstimatedImpact: "20-40% performance improvement",
			Implementation:  "Update cache configuration in config files",
		})
	}

	// Cache size optimization
	if cacheStats.Size > 0 && cacheStats.MaxSize > 0 {
		utilizationRate := float64(cacheStats.Size) / float64(cacheStats.MaxSize)
		if utilizationRate > 0.9 {
			suggestions = append(suggestions, models.CacheOptimizationSuggestion{
				Type:        "size_optimization",
				Priority:    "medium",
				Description: fmt.Sprintf("Cache utilization is %.2f%%, consider increasing cache size", utilizationRate*100),
				Actions: []string{
					"Increase maximum cache size",
					"Implement cache partitioning",
					"Optimize data structures in cache",
				},
				EstimatedImpact: "10-20% performance improvement",
				Implementation:  "Adjust CACHE_MAX_SIZE environment variable",
			})
		}
	}

	// Cache warming suggestions
	suggestions = append(suggestions, models.CacheOptimizationSuggestion{
		Type:        "cache_warming",
		Priority:    "medium",
		Description: "Implement cache warming to improve cold start performance",
		Actions: []string{
			"Pre-load frequently accessed data",
			"Implement background cache warming jobs",
			"Use predictive caching based on access patterns",
		},
		EstimatedImpact: "15-25% improvement in cold start performance",
		Implementation:  "Add cache warming logic to application startup",
	})

	return suggestions, nil
}

// analyzeIndexOptimizations suggests database index optimizations
func (oa *OptimizationAnalyzer) analyzeIndexOptimizations(ctx context.Context, data *OptimizationAnalysisData) ([]models.IndexSuggestion, error) {
	var suggestions []models.IndexSuggestion

	// Analyze slow queries for index opportunities
	for _, slowQuery := range data.SlowQueries {
		if slowQuery.QueryType == "semantic_search" {
			suggestions = append(suggestions, models.IndexSuggestion{
				TableName:    "chunks",
				IndexName:    "idx_chunks_embedding_cosine",
				IndexType:    "vector",
				Columns:      []string{"embedding"},
				Reasoning:    "Optimize vector similarity searches",
				EstimatedImprovement: "50-70% improvement in semantic search performance",
				Priority:     "high",
				SQLCommand:   "CREATE INDEX idx_chunks_embedding_cosine ON chunks USING ivfflat (embedding vector_cosine_ops);",
			})
		}

		if slowQuery.QueryType == "tag_search" {
			suggestions = append(suggestions, models.IndexSuggestion{
				TableName:    "chunk_tags",
				IndexName:    "idx_chunk_tags_content_chunk",
				IndexType:    "btree",
				Columns:      []string{"tag_content", "chunk_id"},
				Reasoning:    "Optimize tag-based searches",
				EstimatedImprovement: "40-60% improvement in tag search performance",
				Priority:     "high",
				SQLCommand:   "CREATE INDEX idx_chunk_tags_content_chunk ON chunk_tags (tag_content, chunk_id);",
			})
		}

		if slowQuery.QueryType == "hierarchy_search" {
			suggestions = append(suggestions, models.IndexSuggestion{
				TableName:    "chunks",
				IndexName:    "idx_chunks_parent_level",
				IndexType:    "btree",
				Columns:      []string{"parent_id", "level"},
				Reasoning:    "Optimize hierarchical queries",
				EstimatedImprovement: "30-50% improvement in hierarchy traversal",
				Priority:     "medium",
				SQLCommand:   "CREATE INDEX idx_chunks_parent_level ON chunks (parent_id, level);",
			})
		}
	}

	// Add general optimization indexes
	suggestions = append(suggestions, models.IndexSuggestion{
		TableName:    "chunks",
		IndexName:    "idx_chunks_created_at",
		IndexType:    "btree",
		Columns:      []string{"created_at"},
		Reasoning:    "Optimize time-based queries and sorting",
		EstimatedImprovement: "20-30% improvement in time-based filtering",
		Priority:     "low",
		SQLCommand:   "CREATE INDEX idx_chunks_created_at ON chunks (created_at DESC);",
	})

	return suggestions, nil
}

// analyzeConfigurationTuning suggests configuration optimizations
func (oa *OptimizationAnalyzer) analyzeConfigurationTuning(ctx context.Context, data *OptimizationAnalysisData) ([]models.ConfigurationTuning, error) {
	var tuning []models.ConfigurationTuning

	// Database connection pool optimization
	if data.ResourceUsage != nil && data.ResourceUsage.DatabaseConnections != nil {
		dbStats := data.ResourceUsage.DatabaseConnections
		if dbStats.ActiveConnections > dbStats.MaxConnections*0.8 {
			tuning = append(tuning, models.ConfigurationTuning{
				Component:   "database",
				Setting:     "connection_pool_size",
				CurrentValue: fmt.Sprintf("%d", dbStats.MaxConnections),
				RecommendedValue: fmt.Sprintf("%d", int(float64(dbStats.MaxConnections)*1.5)),
				Reasoning:   "High connection pool utilization detected",
				Impact:      "Reduce connection contention and improve throughput",
				ConfigKey:   "DB_MAX_CONNECTIONS",
			})
		}
	}

	// Memory optimization
	if data.ResourceUsage != nil && data.ResourceUsage.PeakMemoryUsage > oa.thresholds.HighMemoryUsageThreshold {
		tuning = append(tuning, models.ConfigurationTuning{
			Component:   "application",
			Setting:     "memory_limits",
			CurrentValue: fmt.Sprintf("%d MB", data.ResourceUsage.PeakMemoryUsage/(1024*1024)),
			RecommendedValue: fmt.Sprintf("%d MB", oa.thresholds.HighMemoryUsageThreshold/(1024*1024)),
			Reasoning:   "High memory usage detected",
			Impact:      "Prevent out-of-memory errors and improve stability",
			ConfigKey:   "MEMORY_LIMIT",
		})
	}

	// Timeout optimization
	if data.LoadTestResults != nil {
		avgResponseTime := data.LoadTestResults.OverallStats.AvgResponseTime
		if avgResponseTime > 5*time.Second {
			tuning = append(tuning, models.ConfigurationTuning{
				Component:   "http_server",
				Setting:     "read_timeout",
				CurrentValue: "30s",
				RecommendedValue: fmt.Sprintf("%ds", int(avgResponseTime.Seconds()*2)),
				Reasoning:   "High response times require longer timeouts",
				Impact:      "Prevent premature timeout errors",
				ConfigKey:   "SERVER_READ_TIMEOUT",
			})
		}
	}

	return tuning, nil
}

// generateOptimizationRecommendations generates recommendations using rules engine
func (oa *OptimizationAnalyzer) generateOptimizationRecommendations(data *OptimizationAnalysisData) []models.OptimizationRecommendation {
	var recommendations []models.OptimizationRecommendation

	for _, rule := range oa.optimizationRules {
		if rule.Condition(data) {
			recommendation := rule.Generate(data)
			recommendations = append(recommendations, recommendation)
		}
	}

	return recommendations
}

// Helper methods

func (oa *OptimizationAnalyzer) extractSlowQueriesFromBaseline(baseline *models.BaselinePerformanceResult) []models.SlowQueryAnalysis {
	var slowQueries []models.SlowQueryAnalysis

	// Extract from benchmark results
	if baseline.BenchmarkResults.SemanticSearch.AverageResponseTime > oa.thresholds.SlowQueryThreshold {
		slowQueries = append(slowQueries, models.SlowQueryAnalysis{
			QueryPattern: "semantic_search",
			QueryType:    "semantic_search",
			AvgDuration:  baseline.BenchmarkResults.SemanticSearch.AverageResponseTime,
			Count:        100, // Estimated from benchmark
		})
	}

	if baseline.BenchmarkResults.TagSearch.AverageResponseTime > oa.thresholds.SlowQueryThreshold {
		slowQueries = append(slowQueries, models.SlowQueryAnalysis{
			QueryPattern: "tag_search",
			QueryType:    "tag_search",
			AvgDuration:  baseline.BenchmarkResults.TagSearch.AverageResponseTime,
			Count:        100, // Estimated from benchmark
		})
	}

	return slowQueries
}

func (oa *OptimizationAnalyzer) extractSlowQueriesFromLoadTest(loadTest *models.LoadTestResult) []models.SlowQueryAnalysis {
	var slowQueries []models.SlowQueryAnalysis

	// Analyze load test statistics
	if loadTest.OverallStats.AvgResponseTime > oa.thresholds.SlowQueryThreshold {
		slowQueries = append(slowQueries, models.SlowQueryAnalysis{
			QueryPattern: "load_test_queries",
			QueryType:    "mixed",
			AvgDuration:  loadTest.OverallStats.AvgResponseTime,
			MaxDuration:  loadTest.OverallStats.MaxResponseTime,
			Count:        int(loadTest.OverallStats.TotalRequests),
		})
	}

	return slowQueries
}

func (oa *OptimizationAnalyzer) identifySlowQueryPatterns(slowQueries []models.SlowQueryAnalysis) []SlowQueryPattern {
	patterns := make(map[string]*SlowQueryPattern)

	for _, query := range slowQueries {
		pattern, exists := patterns[query.QueryType]
		if !exists {
			pattern = &SlowQueryPattern{
				Pattern:     query.QueryPattern,
				QueryType:   query.QueryType,
				Count:       0,
				AvgDuration: 0,
				MaxDuration: 0,
			}
			patterns[query.QueryType] = pattern
		}

		pattern.Count += query.Count
		pattern.AvgDuration = (pattern.AvgDuration*time.Duration(pattern.Count-query.Count) + query.AvgDuration*time.Duration(query.Count)) / time.Duration(pattern.Count)
		if query.MaxDuration > pattern.MaxDuration {
			pattern.MaxDuration = query.MaxDuration
		}
	}

	var result []SlowQueryPattern
	for _, pattern := range patterns {
		result = append(result, *pattern)
	}

	return result
}

func (oa *OptimizationAnalyzer) generateQueryOptimizationSuggestions(pattern SlowQueryPattern) []string {
	var suggestions []string

	switch pattern.QueryType {
	case "semantic_search":
		suggestions = []string{
			"Add vector index on embedding column",
			"Optimize embedding dimension and model",
			"Implement query result caching",
			"Use approximate nearest neighbor algorithms",
		}
	case "tag_search":
		suggestions = []string{
			"Add composite index on tag_content and chunk_id",
			"Optimize tag normalization and storage",
			"Implement tag hierarchy for faster filtering",
			"Use tag frequency optimization",
		}
	case "hierarchy_search":
		suggestions = []string{
			"Add index on parent_id and level columns",
			"Implement path enumeration for faster traversal",
			"Use materialized path for hierarchy queries",
			"Cache frequently accessed hierarchy paths",
		}
	default:
		suggestions = []string{
			"Analyze query execution plan",
			"Add appropriate database indexes",
			"Optimize query structure and joins",
			"Consider query result caching",
		}
	}

	return suggestions
}

func (oa *OptimizationAnalyzer) calculateQueryImpact(pattern SlowQueryPattern) float64 {
	// Calculate impact based on count and duration
	durationScore := math.Min(pattern.AvgDuration.Seconds()/10.0, 1.0) // Normalize to 0-1
	countScore := math.Min(float64(pattern.Count)/1000.0, 1.0)         // Normalize to 0-1

	return (durationScore + countScore) / 2.0
}

func (oa *OptimizationAnalyzer) calculatePerformanceMetrics(data *OptimizationAnalysisData) map[string]interface{} {
	metrics := make(map[string]interface{})

	if data.BaselineResults != nil {
		metrics["baseline_semantic_avg_time"] = data.BaselineResults.BenchmarkResults.SemanticSearch.AverageResponseTime
		metrics["baseline_tag_avg_time"] = data.BaselineResults.BenchmarkResults.TagSearch.AverageResponseTime
		metrics["baseline_cache_hit_rate"] = data.BaselineResults.CacheStats.HitRate
	}

	if data.LoadTestResults != nil {
		metrics["load_test_avg_time"] = data.LoadTestResults.OverallStats.AvgResponseTime
		metrics["load_test_max_time"] = data.LoadTestResults.OverallStats.MaxResponseTime
		metrics["load_test_error_rate"] = data.LoadTestResults.OverallStats.ErrorRate
		metrics["load_test_total_requests"] = data.LoadTestResults.OverallStats.TotalRequests
	}

	if data.ResourceUsage != nil {
		metrics["peak_memory_usage"] = data.ResourceUsage.PeakMemoryUsage
		metrics["avg_cpu_usage"] = data.ResourceUsage.AverageCPUUsage
	}

	return metrics
}

func (oa *OptimizationAnalyzer) prioritizeRecommendations(recommendations []models.OptimizationRecommendation) []models.OptimizationRecommendation {
	// Define priority weights
	priorityWeights := map[string]int{
		"critical": 4,
		"high":     3,
		"medium":   2,
		"low":      1,
	}

	// Sort by priority
	sort.Slice(recommendations, func(i, j int) bool {
		weightI := priorityWeights[recommendations[i].Priority]
		weightJ := priorityWeights[recommendations[j].Priority]
		return weightI > weightJ
	})

	return recommendations
}

// initializeOptimizationRules sets up the rules engine
func (oa *OptimizationAnalyzer) initializeOptimizationRules() {
	oa.optimizationRules = []OptimizationRule{
		{
			Name:     "slow_semantic_search",
			Category: "database",
			Priority: "high",
			Condition: func(data *OptimizationAnalysisData) bool {
				return data.BaselineResults != nil &&
					data.BaselineResults.BenchmarkResults.SemanticSearch.AverageResponseTime > oa.thresholds.SlowQueryThreshold
			},
			Generate: func(data *OptimizationAnalysisData) models.OptimizationRecommendation {
				return models.OptimizationRecommendation{
					Category:    "database",
					Priority:    "high",
					Title:       "Optimize Semantic Search Performance",
					Description: "Semantic search queries are exceeding performance thresholds",
					Actions: []string{
						"Add vector index using ivfflat",
						"Optimize embedding dimension",
						"Implement query result caching",
						"Use approximate nearest neighbor",
					},
					EstimatedImpact: "50-70% improvement in semantic search performance",
					Implementation:  "Add vector indexes and optimize query structure",
				}
			},
		},
		{
			Name:     "low_cache_hit_rate",
			Category: "cache",
			Priority: "medium",
			Condition: func(data *OptimizationAnalysisData) bool {
				return data.BaselineResults != nil &&
					data.BaselineResults.CacheStats.HitRate < oa.thresholds.LowCacheHitRateThreshold
			},
			Generate: func(data *OptimizationAnalysisData) models.OptimizationRecommendation {
				return models.OptimizationRecommendation{
					Category:    "cache",
					Priority:    "medium",
					Title:       "Improve Cache Hit Rate",
					Description: fmt.Sprintf("Cache hit rate is %.2f%%, below optimal threshold", data.BaselineResults.CacheStats.HitRate*100),
					Actions: []string{
						"Increase cache size",
						"Optimize cache TTL settings",
						"Implement cache warming",
						"Review cache eviction policies",
					},
					EstimatedImpact: "20-40% performance improvement",
					Implementation:  "Update cache configuration and implement warming strategies",
				}
			},
		},
		{
			Name:     "high_error_rate",
			Category: "reliability",
			Priority: "critical",
			Condition: func(data *OptimizationAnalysisData) bool {
				return data.LoadTestResults != nil &&
					data.LoadTestResults.OverallStats.ErrorRate > oa.thresholds.HighErrorRateThreshold
			},
			Generate: func(data *OptimizationAnalysisData) models.OptimizationRecommendation {
				return models.OptimizationRecommendation{
					Category:    "reliability",
					Priority:    "critical",
					Title:       "Address High Error Rate",
					Description: fmt.Sprintf("Error rate is %.2f%%, exceeding acceptable threshold", data.LoadTestResults.OverallStats.ErrorRate*100),
					Actions: []string{
						"Investigate error root causes",
						"Implement better error handling",
						"Add circuit breakers",
						"Improve timeout configurations",
					},
					EstimatedImpact: "Critical for system reliability",
					Implementation:  "Immediate investigation and error handling improvements required",
				}
			},
		},
	}
}

func getDefaultPerformanceThresholds() *PerformanceThresholds {
	return &PerformanceThresholds{
		SlowQueryThreshold:        500 * time.Millisecond,
		LowCacheHitRateThreshold:  0.8,
		HighErrorRateThreshold:    0.05, // 5%
		LowThroughputThreshold:    100.0,
		HighMemoryUsageThreshold:  1024 * 1024 * 1024, // 1GB
		HighCPUUsageThreshold:     80.0,                // 80%
		DatabaseConnectionLimit:   100,
	}
}