package models

import (
	"time"
)

// Enhanced search request/response models for Task 6 optimizations

// OptimizedSearchRequest represents an enhanced semantic search request
type OptimizedSearchRequest struct {
	Query           string                 `json:"query"`
	Limit           int                    `json:"limit"`
	MinSimilarity   float64                `json:"min_similarity"`
	Filters         map[string]interface{} `json:"filters"`
	IncludeMetadata bool                   `json:"include_metadata"`
	UseCache        bool                   `json:"use_cache"`
	PreloadHints    []string               `json:"preload_hints,omitempty"`
}

// OptimizedSearchResponse represents an enhanced search response with optimization metadata
type OptimizedSearchResponse struct {
	Results       []OptimizedSearchResult `json:"results"`
	TotalCount    int                     `json:"total_count"`
	Duration      time.Duration           `json:"duration"`
	CacheHit      bool                    `json:"cache_hit"`
	Optimizations []string                `json:"optimizations"`
	Metadata      SearchMetadata          `json:"metadata"`
}

// OptimizedSearchResult represents a single search result with enhanced scoring
type OptimizedSearchResult struct {
	ChunkID     string                 `json:"chunk_id"`
	Content     string                 `json:"content"`
	Similarity  float64                `json:"similarity"`
	Relevance   float64                `json:"relevance"`
	Metadata    map[string]interface{} `json:"metadata"`
	Tags        []string               `json:"tags"`
	Snippet     string                 `json:"snippet,omitempty"`
	Highlights  []TextHighlight        `json:"highlights,omitempty"`
}

// SearchMetadata provides additional information about the search operation
type SearchMetadata struct {
	QueryHash         string        `json:"query_hash"`
	IndexesUsed       []string      `json:"indexes_used"`
	DatabaseQueries   int           `json:"database_queries"`
	CacheOperations   int           `json:"cache_operations"`
	OptimizationLevel string        `json:"optimization_level"`
	ProcessingSteps   []string      `json:"processing_steps"`
}

// TextHighlight represents highlighted text segments
type TextHighlight struct {
	Text      string `json:"text"`
	StartPos  int    `json:"start_pos"`
	EndPos    int    `json:"end_pos"`
	MatchType string `json:"match_type"`
}

// OptimizedTagSearchRequest represents an optimized tag search request
type OptimizedTagSearchRequest struct {
	Tags            []string               `json:"tags"`
	CombinationMode string                 `json:"combination_mode"` // "AND", "OR", "NOT"
	Limit           int                    `json:"limit"`
	SortByRelevance bool                   `json:"sort_by_relevance"`
	IncludeStats    bool                   `json:"include_stats"`
	Filters         map[string]interface{} `json:"filters"`
}

// TagSearchResponse represents an optimized tag search response
type TagSearchResponse struct {
	Results       []TaggedChunk          `json:"results"`
	TagStats      map[string]int         `json:"tag_stats"`
	Duration      time.Duration          `json:"duration"`
	TotalCount    int                    `json:"total_count"`
	Optimizations []string               `json:"optimizations"`
	Suggestions   []TagSuggestion        `json:"suggestions,omitempty"`
}

// TaggedChunk represents a chunk with its associated tags and relevance
type TaggedChunk struct {
	ChunkID      string                 `json:"chunk_id"`
	Content      string                 `json:"content"`
	Tags         []string               `json:"tags"`
	TagRelevance map[string]float64     `json:"tag_relevance"`
	Metadata     map[string]interface{} `json:"metadata"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

// TagSuggestion represents suggested tags based on search patterns
type TagSuggestion struct {
	Tag         string  `json:"tag"`
	Relevance   float64 `json:"relevance"`
	Frequency   int     `json:"frequency"`
	RelatedTags []string `json:"related_tags"`
}

// FullTextRequest represents an enhanced full-text search request
type FullTextRequest struct {
	Query           string `json:"query"`
	Limit           int    `json:"limit"`
	IncludeSnippets bool   `json:"include_snippets"`
	IncludeHighlights bool `json:"include_highlights"`
	RankingMode     string `json:"ranking_mode"` // "bm25", "tfidf", "hybrid"
	BoostFields     map[string]float64 `json:"boost_fields,omitempty"`
}

// FullTextResponse represents an enhanced full-text search response
type FullTextResponse struct {
	Results       []FullTextResult `json:"results"`
	Suggestions   []string         `json:"suggestions"`
	Duration      time.Duration    `json:"duration"`
	TotalCount    int              `json:"total_count"`
	CacheHit      bool             `json:"cache_hit"`
	Optimizations []string         `json:"optimizations"`
	QueryAnalysis QueryAnalysis    `json:"query_analysis"`
}

// FullTextResult represents a single full-text search result
type FullTextResult struct {
	ChunkID       string                 `json:"chunk_id"`
	Content       string                 `json:"content"`
	Title         string                 `json:"title,omitempty"`
	Snippet       string                 `json:"snippet"`
	Highlights    []TextHighlight        `json:"highlights"`
	RelevanceScore float64               `json:"relevance_score"`
	BM25Score     float64                `json:"bm25_score,omitempty"`
	TFIDFScore    float64                `json:"tfidf_score,omitempty"`
	Metadata      map[string]interface{} `json:"metadata"`
	MatchedTerms  []string               `json:"matched_terms"`
}

// QueryAnalysis provides insights into the search query processing
type QueryAnalysis struct {
	OriginalQuery   string   `json:"original_query"`
	ProcessedQuery  string   `json:"processed_query"`
	ExtractedTerms  []string `json:"extracted_terms"`
	QueryType       string   `json:"query_type"`
	EstimatedResults int     `json:"estimated_results"`
	ProcessingTime  time.Duration `json:"processing_time"`
}

// UnifiedSearchQuery represents a query for the unified search system
type UnifiedSearchQuery struct {
	QueryText       string                 `json:"query_text"`
	QueryVector     []float64              `json:"query_vector,omitempty"`
	Filters         map[string]interface{} `json:"filters"`
	Limit           int                    `json:"limit"`
	Offset          int                    `json:"offset"`
	MinSimilarity   float64                `json:"min_similarity"`
	UseVectorIndex  bool                   `json:"use_vector_index"`
	BatchSize       int                    `json:"batch_size"`
	IncludeMetadata bool                   `json:"include_metadata"`
	OptimizeForSpeed bool                  `json:"optimize_for_speed"`
}

// FullTextSearchQuery represents a full-text search query for the unified system
type FullTextSearchQuery struct {
	Query               string `json:"query"`
	OriginalQuery       string `json:"original_query"`
	Limit               int    `json:"limit"`
	UseOptimizedIndexes bool   `json:"use_optimized_indexes"`
	RankingFunction     string `json:"ranking_function"`
	IncludeSnippets     bool   `json:"include_snippets"`
}

// Benchmark and performance models

// BenchmarkResults represents comprehensive benchmark results
type BenchmarkResults struct {
	Timestamp        time.Time                  `json:"timestamp"`
	SemanticSearch   SemanticSearchBenchmark    `json:"semantic_search"`
	TagSearch        TagSearchBenchmark         `json:"tag_search"`
	FullTextSearch   FullTextSearchBenchmark    `json:"full_text_search"`
	CachePerformance CachePerformanceBenchmark  `json:"cache_performance"`
	StressTest       *StressTestResults         `json:"stress_test,omitempty"`
	OverallScore     float64                    `json:"overall_score"`
	Duration         time.Duration              `json:"duration"`
}

// SemanticSearchBenchmark represents semantic search performance metrics
type SemanticSearchBenchmark struct {
	AverageResponseTime time.Duration `json:"average_response_time"`
	ThroughputQPS       float64       `json:"throughput_qps"`
	P95ResponseTime     time.Duration `json:"p95_response_time"`
	P99ResponseTime     time.Duration `json:"p99_response_time"`
	SuccessRate         float64       `json:"success_rate"`
	CacheHitRate        float64       `json:"cache_hit_rate"`
	AccuracyScore       float64       `json:"accuracy_score"`
}

// TagSearchBenchmark represents tag search performance metrics
type TagSearchBenchmark struct {
	AverageResponseTime   time.Duration `json:"average_response_time"`
	ThroughputQPS         float64       `json:"throughput_qps"`
	SingleTagPerformance  time.Duration `json:"single_tag_performance"`
	MultiTagPerformance   time.Duration `json:"multi_tag_performance"`
	TagCacheEffectiveness float64       `json:"tag_cache_effectiveness"`
	SuccessRate           float64       `json:"success_rate"`
}

// FullTextSearchBenchmark represents full-text search performance metrics
type FullTextSearchBenchmark struct {
	AverageResponseTime time.Duration `json:"average_response_time"`
	ThroughputQPS       float64       `json:"throughput_qps"`
	IndexingEfficiency  float64       `json:"indexing_efficiency"`
	RelevanceScore      float64       `json:"relevance_score"`
	SuggestionQuality   float64       `json:"suggestion_quality"`
	SuccessRate         float64       `json:"success_rate"`
}

// CachePerformanceBenchmark represents cache performance metrics
type CachePerformanceBenchmark struct {
	OverallHitRate       float64       `json:"overall_hit_rate"`
	SemanticCacheHitRate float64       `json:"semantic_cache_hit_rate"`
	TagCacheHitRate      float64       `json:"tag_cache_hit_rate"`
	FullTextCacheHitRate float64       `json:"full_text_cache_hit_rate"`
	CacheWarmingTime     time.Duration `json:"cache_warming_time"`
	CacheEffectiveness   float64       `json:"cache_effectiveness"`
}

// StressTestResults represents stress test performance metrics
type StressTestResults struct {
	ConcurrentUsers     int           `json:"concurrent_users"`
	TestDuration        time.Duration `json:"test_duration"`
	TotalRequests       int           `json:"total_requests"`
	FailedRequests      int           `json:"failed_requests"`
	SuccessRate         float64       `json:"success_rate"`
	AverageResponseTime time.Duration `json:"average_response_time"`
	ThroughputQPS       float64       `json:"throughput_qps"`
	MaxResponseTime     time.Duration `json:"max_response_time"`
	MinResponseTime     time.Duration `json:"min_response_time"`
}

// SearchStatistics represents comprehensive search system statistics
type SearchStatistics struct {
	TotalSearches       int                    `json:"total_searches"`
	AverageResponseTime time.Duration          `json:"average_response_time"`
	CacheHitRate        float64                `json:"cache_hit_rate"`
	PopularSearches     []SearchPattern        `json:"popular_searches"`
	SlowQueries         []SlowQuery            `json:"slow_queries"`
	PerformanceMetrics  map[string]interface{} `json:"performance_metrics"`
}

// SearchPattern represents an analyzed search pattern
type SearchPattern struct {
	Pattern      string        `json:"pattern"`
	Frequency    int           `json:"frequency"`
	LastUsed     time.Time     `json:"last_used"`
	AvgDuration  time.Duration `json:"avg_duration"`
	CacheHitRate float64       `json:"cache_hit_rate"`
}

// SlowQuery represents a query that exceeds performance thresholds
type SlowQuery struct {
	Query       string        `json:"query"`
	Duration    time.Duration `json:"duration"`
	Timestamp   time.Time     `json:"timestamp"`
	QueryType   string        `json:"query_type"`
	Suggestions []string      `json:"suggestions"`
}

// CacheOptimizationSuggestion represents a cache optimization recommendation
type CacheOptimizationSuggestion struct {
	Type              string   `json:"type"`
	Pattern           string   `json:"pattern"`
	Priority          string   `json:"priority"`
	Description       string   `json:"description"`
	Actions           []string `json:"actions"`
	EstimatedImpact   string   `json:"estimated_impact"`
	Implementation    string   `json:"implementation"`
}

// Additional models for Task 8 performance testing

// EnvironmentInfo captures the test environment information
type EnvironmentInfo struct {
	GoVersion     string      `json:"go_version"`
	NumCPU        int         `json:"num_cpu"`
	NumGoroutines int         `json:"num_goroutines"`
	MemoryStats   MemoryStats `json:"memory_stats"`
	OSArch        string      `json:"os_arch"`
	Timestamp     time.Time   `json:"timestamp"`
}

// MemoryStats captures memory usage statistics
type MemoryStats struct {
	Alloc        uint64 `json:"alloc"`
	TotalAlloc   uint64 `json:"total_alloc"`
	Sys          uint64 `json:"sys"`
	Lookups      uint64 `json:"lookups"`
	Mallocs      uint64 `json:"mallocs"`
	Frees        uint64 `json:"frees"`
	HeapAlloc    uint64 `json:"heap_alloc"`
	HeapSys      uint64 `json:"heap_sys"`
	HeapIdle     uint64 `json:"heap_idle"`
	HeapInuse    uint64 `json:"heap_inuse"`
	HeapReleased uint64 `json:"heap_released"`
	HeapObjects  uint64 `json:"heap_objects"`
	StackInuse   uint64 `json:"stack_inuse"`
	StackSys     uint64 `json:"stack_sys"`
	NumGC        uint32 `json:"num_gc"`
	PauseTotalNs uint64 `json:"pause_total_ns"`
}

// SystemInfo captures system information
type SystemInfo struct {
	CPUCount   int `json:"cpu_count"`
	MaxProcs   int `json:"max_procs"`
	Goroutines int `json:"goroutines"`
	CGOCalls   int64 `json:"cgo_calls"`
}

// DataGenerationResult represents the result of data generation
type DataGenerationResult struct {
	StartTime           time.Time           `json:"start_time"`
	EndTime             time.Time           `json:"end_time"`
	Duration            time.Duration       `json:"duration"`
	TargetSize          int                 `json:"target_size"`
	RecordsGenerated    int                 `json:"records_generated"`
	GenerationErrors    int                 `json:"generation_errors"`
	RecordsPerSecond    float64             `json:"records_per_second"`
	DataIntegrityCheck  DataIntegrityCheck  `json:"data_integrity_check"`
}

// DataIntegrityCheck represents data integrity verification results
type DataIntegrityCheck struct {
	TotalRecords     int                    `json:"total_records"`
	ValidRecords     int                    `json:"valid_records"`
	InvalidRecords   int                    `json:"invalid_records"`
	MissingFields    map[string]int         `json:"missing_fields"`
	DataQualityScore float64                `json:"data_quality_score"`
	Issues           []DataQualityIssue     `json:"issues"`
	SampleValidation map[string]interface{} `json:"sample_validation"`
}

// DataQualityIssue represents a specific data quality problem
type DataQualityIssue struct {
	Type        string `json:"type"`
	Field       string `json:"field"`
	Count       int    `json:"count"`
	Description string `json:"description"`
	Severity    string `json:"severity"`
}

// BaselinePerformanceResult represents baseline performance test results
type BaselinePerformanceResult struct {
	BenchmarkResults BenchmarkResults `json:"benchmark_results"`
	DatabaseStats    DatabaseStats    `json:"database_stats"`
	CacheStats       CacheStats       `json:"cache_stats"`
	MemoryUsage      MemoryStats      `json:"memory_usage"`
	SystemInfo       SystemInfo       `json:"system_info"`
}

// DatabaseStats represents database performance statistics
type DatabaseStats struct {
	ActiveConnections    int           `json:"active_connections"`
	IdleConnections      int           `json:"idle_connections"`
	TotalConnections     int           `json:"total_connections"`
	MaxConnections       int           `json:"max_connections"`
	QueryCount           int64         `json:"query_count"`
	SlowQueryCount       int64         `json:"slow_query_count"`
	AverageQueryDuration time.Duration `json:"average_query_duration"`
}

// CacheStats represents cache performance statistics
type CacheStats struct {
	HitRate     float64 `json:"hit_rate"`
	MissRate    float64 `json:"miss_rate"`
	Size        int64   `json:"size"`
	MaxSize     int64   `json:"max_size"`
	Evictions   int64   `json:"evictions"`
	Hits        int64   `json:"hits"`
	Misses      int64   `json:"misses"`
}

// LoadTestResult represents load test execution results
type LoadTestResult struct {
	StartTime       time.Time                      `json:"start_time"`
	EndTime         time.Time                      `json:"end_time"`
	TotalDuration   time.Duration                  `json:"total_duration"`
	Config          interface{}                    `json:"config"`
	LoadSteps       []LoadStepResult               `json:"load_steps"`
	OverallStats    LoadTestStats                  `json:"overall_stats"`
	ThroughputData  []ThroughputPoint              `json:"throughput_data"`
	SystemMetrics   []SystemMetricPoint            `json:"system_metrics"`
	UserMetrics     map[int]*UserMetrics           `json:"user_metrics"`
}

// LoadStepResult represents results for a single load step
type LoadStepResult struct {
	UserCount        int                              `json:"user_count"`
	StartTime        time.Time                        `json:"start_time"`
	EndTime          time.Time                        `json:"end_time"`
	Duration         time.Duration                    `json:"duration"`
	ActualDuration   time.Duration                    `json:"actual_duration"`
	TotalRequests    int64                            `json:"total_requests"`
	TotalErrors      int64                            `json:"total_errors"`
	ErrorRate        float64                          `json:"error_rate"`
	AvgResponseTime  time.Duration                    `json:"avg_response_time"`
	QPS              float64                          `json:"qps"`
	RequestStats     map[string]RequestTypeStats      `json:"request_stats"`
	Error            string                           `json:"error,omitempty"`
}

// LoadTestStats represents overall load test statistics
type LoadTestStats struct {
	TotalRequests    int64         `json:"total_requests"`
	TotalErrors      int64         `json:"total_errors"`
	ErrorRate        float64       `json:"error_rate"`
	AvgResponseTime  time.Duration `json:"avg_response_time"`
	MinResponseTime  time.Duration `json:"min_response_time"`
	MaxResponseTime  time.Duration `json:"max_response_time"`
	P50ResponseTime  time.Duration `json:"p50_response_time"`
	P95ResponseTime  time.Duration `json:"p95_response_time"`
	P99ResponseTime  time.Duration `json:"p99_response_time"`
}

// RequestTypeStats represents statistics for a specific request type
type RequestTypeStats struct {
	Count           int64         `json:"count"`
	Errors          int64         `json:"errors"`
	AvgResponseTime time.Duration `json:"avg_response_time"`
	MinResponseTime time.Duration `json:"min_response_time"`
	MaxResponseTime time.Duration `json:"max_response_time"`
}

// ThroughputPoint represents throughput at a specific time
type ThroughputPoint struct {
	Timestamp   time.Time `json:"timestamp"`
	QPS         float64   `json:"qps"`
	ActiveUsers int       `json:"active_users"`
}

// SystemMetricPoint represents system metrics at a specific time
type SystemMetricPoint struct {
	Timestamp    time.Time     `json:"timestamp"`
	CPUUsage     float64       `json:"cpu_usage"`
	MemoryUsage  uint64        `json:"memory_usage"`
	ActiveUsers  int           `json:"active_users"`
	ResponseTime time.Duration `json:"response_time"`
	ErrorRate    float64       `json:"error_rate"`
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

// OptimizationAnalysisResult represents performance optimization analysis results
type OptimizationAnalysisResult struct {
	AnalysisTimestamp   time.Time                      `json:"analysis_timestamp"`
	Recommendations     []OptimizationRecommendation   `json:"recommendations"`
	SlowQueries         []SlowQueryAnalysis            `json:"slow_queries"`
	CacheOptimizations  []CacheOptimizationSuggestion  `json:"cache_optimizations"`
	IndexSuggestions    []IndexSuggestion              `json:"index_suggestions"`
	ConfigurationTuning []ConfigurationTuning          `json:"configuration_tuning"`
	PerformanceMetrics  map[string]interface{}         `json:"performance_metrics"`
}

// OptimizationRecommendation represents a performance optimization recommendation
type OptimizationRecommendation struct {
	Category        string   `json:"category"`
	Priority        string   `json:"priority"`
	Title           string   `json:"title"`
	Description     string   `json:"description"`
	Actions         []string `json:"actions"`
	EstimatedImpact string   `json:"estimated_impact"`
	Implementation  string   `json:"implementation"`
}

// SlowQueryAnalysis represents analysis of slow queries
type SlowQueryAnalysis struct {
	QueryPattern            string        `json:"query_pattern"`
	Count                   int           `json:"count"`
	AvgDuration            time.Duration `json:"avg_duration"`
	MaxDuration            time.Duration `json:"max_duration"`
	QueryType              string        `json:"query_type"`
	AffectedTables         []string      `json:"affected_tables"`
	OptimizationSuggestions []string      `json:"optimization_suggestions"`
	Impact                 float64       `json:"impact"`
}

// IndexSuggestion represents a database index optimization suggestion
type IndexSuggestion struct {
	TableName            string   `json:"table_name"`
	IndexName            string   `json:"index_name"`
	IndexType            string   `json:"index_type"`
	Columns              []string `json:"columns"`
	Reasoning            string   `json:"reasoning"`
	EstimatedImprovement string   `json:"estimated_improvement"`
	Priority             string   `json:"priority"`
	SQLCommand           string   `json:"sql_command"`
}

// ConfigurationTuning represents a configuration optimization suggestion
type ConfigurationTuning struct {
	Component        string `json:"component"`
	Setting          string `json:"setting"`
	CurrentValue     string `json:"current_value"`
	RecommendedValue string `json:"recommended_value"`
	Reasoning        string `json:"reasoning"`
	Impact           string `json:"impact"`
	ConfigKey        string `json:"config_key"`
}

// RegressionTestResult represents regression test results
type RegressionTestResult struct {
	HistoricalBaseline BaselinePerformanceResult `json:"historical_baseline"`
	CurrentResults     BaselinePerformanceResult `json:"current_results"`
	Comparison         PerformanceComparison     `json:"comparison"`
	RegressionDetected bool                      `json:"regression_detected"`
	ImprovementAreas   []string                  `json:"improvement_areas"`
}

// PerformanceComparison represents comparison between performance results
type PerformanceComparison struct {
	HasRegression      bool                `json:"has_regression"`
	ImprovementAreas   []string            `json:"improvement_areas"`
	DegradationAreas   []string            `json:"degradation_areas"`
	PerformanceDelta   map[string]float64  `json:"performance_delta"`
}

// ResourceUtilizationResult represents system resource utilization analysis
type ResourceUtilizationResult struct {
	PeakMemoryUsage     uint64                     `json:"peak_memory_usage"`
	AverageCPUUsage     float64                    `json:"average_cpu_usage"`
	DiskIOStats         DiskIOStats                `json:"disk_io_stats"`
	NetworkIOStats      NetworkIOStats             `json:"network_io_stats"`
	GarbageCollection   GCStats                    `json:"garbage_collection"`
	DatabaseConnections *DatabaseStats             `json:"database_connections"`
	ResourceThresholds  map[string]interface{}     `json:"resource_thresholds"`
}

// DiskIOStats represents disk I/O statistics
type DiskIOStats struct {
	ReadBytes  uint64 `json:"read_bytes"`
	WriteBytes uint64 `json:"write_bytes"`
	ReadOps    uint64 `json:"read_ops"`
	WriteOps   uint64 `json:"write_ops"`
}

// NetworkIOStats represents network I/O statistics
type NetworkIOStats struct {
	BytesSent     uint64 `json:"bytes_sent"`
	BytesReceived uint64 `json:"bytes_received"`
	PacketsSent   uint64 `json:"packets_sent"`
	PacketsReceived uint64 `json:"packets_received"`
}

// GCStats represents garbage collection statistics
type GCStats struct {
	NumGC        uint32        `json:"num_gc"`
	PauseTotalNs uint64        `json:"pause_total_ns"`
	PauseNs      []uint64      `json:"pause_ns"`
	LastGC       time.Time     `json:"last_gc"`
}

// PerformanceRecommendation represents a final performance recommendation
type PerformanceRecommendation struct {
	Category    string   `json:"category"`
	Priority    string   `json:"priority"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Actions     []string `json:"actions"`
}

// PerformanceSummary represents a summary of performance test results
type PerformanceSummary struct {
	TestDate            time.Time              `json:"test_date"`
	OverallScore        float64                `json:"overall_score"`
	KeyMetrics          map[string]interface{} `json:"key_metrics"`
	CriticalIssues      []string               `json:"critical_issues"`
	TopRecommendations  []string               `json:"top_recommendations"`
}

// TestQuery represents a test query for load testing
type TestQuery struct {
	Type       string      `json:"type"`
	Parameters interface{} `json:"parameters"`
	Expected   interface{} `json:"expected,omitempty"`
}

// ChunkContent represents generated chunk content
type ChunkContent struct {
	ID       int    `json:"id"`
	Content  string `json:"content"`
	Category string `json:"category"`
	Length   int    `json:"length"`
}

// GeneratedChunkRecord represents a complete generated test record
type GeneratedChunkRecord struct {
	ID        int                    `json:"id"`
	Content   string                 `json:"content"`
	Category  string                 `json:"category"`
	Embedding []float64              `json:"embedding"`
	Tags      []string               `json:"tags"`
	Metadata  map[string]interface{} `json:"metadata"`
	CreatedAt time.Time              `json:"created_at"`
}

// CacheAnalysisResult represents cache performance analysis
type CacheAnalysisResult struct {
	HitRate           float64                    `json:"hit_rate"`
	MissRate          float64                    `json:"miss_rate"`
	OptimalCacheSize  int64                      `json:"optimal_cache_size"`
	EvictionAnalysis  EvictionAnalysis           `json:"eviction_analysis"`
	HotDataPatterns   []HotDataPattern           `json:"hot_data_patterns"`
}

// EvictionAnalysis represents cache eviction analysis
type EvictionAnalysis struct {
	EvictionRate      float64 `json:"eviction_rate"`
	PrematureEvictions int64   `json:"premature_evictions"`
	RecommendedTTL    time.Duration `json:"recommended_ttl"`
}

// HotDataPattern represents frequently accessed data patterns
type HotDataPattern struct {
	Pattern     string  `json:"pattern"`
	AccessCount int64   `json:"access_count"`
	LastAccess  time.Time `json:"last_access"`
	CacheWeight float64 `json:"cache_weight"`
}

// IndexAnalysisResult represents database index analysis
type IndexAnalysisResult struct {
	CurrentIndexes    []CurrentIndex    `json:"current_indexes"`
	SuggestedIndexes  []IndexSuggestion `json:"suggested_indexes"`
	UnusedIndexes     []UnusedIndex     `json:"unused_indexes"`
	IndexEfficiency   IndexEfficiency   `json:"index_efficiency"`
}

// CurrentIndex represents an existing database index
type CurrentIndex struct {
	Name       string   `json:"name"`
	Table      string   `json:"table"`
	Columns    []string `json:"columns"`
	Type       string   `json:"type"`
	Size       int64    `json:"size"`
	UsageCount int64    `json:"usage_count"`
}

// UnusedIndex represents an unused database index
type UnusedIndex struct {
	Name          string    `json:"name"`
	Table         string    `json:"table"`
	Size          int64     `json:"size"`
	LastUsed      time.Time `json:"last_used"`
	RemovalSafe   bool      `json:"removal_safe"`
}

// IndexEfficiency represents overall index efficiency metrics
type IndexEfficiency struct {
	OverallEfficiency  float64 `json:"overall_efficiency"`
	IndexUtilization   float64 `json:"index_utilization"`
	MaintenanceOverhead float64 `json:"maintenance_overhead"`
	Recommendation string                 `json:"recommendation"`
	Metrics        map[string]interface{} `json:"metrics"`
}

// Search index and optimization models

// SearchIndex represents metadata about search indexes
type SearchIndex struct {
	Name        string            `json:"name"`
	Type        string            `json:"type"` // "vector", "fulltext", "btree", "gin", "gist"
	Table       string            `json:"table"`
	Columns     []string          `json:"columns"`
	Size        int64             `json:"size"`
	Usage       SearchIndexUsage  `json:"usage"`
	Performance IndexPerformance  `json:"performance"`
	Config      map[string]interface{} `json:"config"`
}

// SearchIndexUsage represents index usage statistics
type SearchIndexUsage struct {
	QueriesUsed    int       `json:"queries_used"`
	LastUsed       time.Time `json:"last_used"`
	HitRate        float64   `json:"hit_rate"`
	Effectiveness  float64   `json:"effectiveness"`
}

// IndexPerformance represents index performance metrics
type IndexPerformance struct {
	AverageSeekTime  time.Duration `json:"average_seek_time"`
	AverageScanTime  time.Duration `json:"average_scan_time"`
	IndexSelectivity float64       `json:"index_selectivity"`
	MaintenanceCost  float64       `json:"maintenance_cost"`
}

// SearchOptimizationPlan represents a plan for optimizing search performance
type SearchOptimizationPlan struct {
	PlanID          string                    `json:"plan_id"`
	CreatedAt       time.Time                 `json:"created_at"`
	TargetMetrics   SearchPerformanceTargets  `json:"target_metrics"`
	Optimizations   []OptimizationAction      `json:"optimizations"`
	EstimatedImpact OptimizationImpact        `json:"estimated_impact"`
	Status          string                    `json:"status"`
}

// SearchPerformanceTargets defines target performance metrics
type SearchPerformanceTargets struct {
	MaxResponseTime    time.Duration `json:"max_response_time"`
	MinThroughput      float64       `json:"min_throughput"`
	MinCacheHitRate    float64       `json:"min_cache_hit_rate"`
	MinSuccessRate     float64       `json:"min_success_rate"`
	MaxErrorRate       float64       `json:"max_error_rate"`
}

// OptimizationAction represents a specific optimization to perform
type OptimizationAction struct {
	ActionID     string                 `json:"action_id"`
	Type         string                 `json:"type"` // "index", "cache", "query", "config"
	Description  string                 `json:"description"`
	Priority     int                    `json:"priority"`
	Effort       string                 `json:"effort"` // "low", "medium", "high"
	Risk         string                 `json:"risk"`   // "low", "medium", "high"
	Parameters   map[string]interface{} `json:"parameters"`
	Dependencies []string               `json:"dependencies"`
}

// OptimizationImpact represents the expected impact of optimizations
type OptimizationImpact struct {
	ResponseTimeImprovement float64 `json:"response_time_improvement"`
	ThroughputImprovement   float64 `json:"throughput_improvement"`
	CacheHitRateImprovement float64 `json:"cache_hit_rate_improvement"`
	ResourceUsageChange     float64 `json:"resource_usage_change"`
	ConfidenceLevel         float64 `json:"confidence_level"`
}