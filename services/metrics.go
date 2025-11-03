package services

import (
	"context"
	"sync"
	"time"
)

// MetricsService provides application metrics and monitoring
type MetricsService interface {
	IncrementCounter(name string, tags map[string]string)
	RecordDuration(name string, duration time.Duration, tags map[string]string)
	SetGauge(name string, value float64, tags map[string]string)
	GetMetrics() map[string]interface{}
	Reset()
}

// Counter represents a monotonically increasing counter
type Counter struct {
	Value int64            `json:"value"`
	Tags  map[string]string `json:"tags,omitempty"`
}

// Histogram represents duration measurements
type Histogram struct {
	Count    int64         `json:"count"`
	Sum      time.Duration `json:"sum"`
	Min      time.Duration `json:"min"`
	Max      time.Duration `json:"max"`
	Average  time.Duration `json:"average"`
	Tags     map[string]string `json:"tags,omitempty"`
	Buckets  map[string]int64 `json:"buckets"` // Duration buckets for percentiles
}

// Gauge represents a value that can go up and down
type Gauge struct {
	Value float64          `json:"value"`
	Tags  map[string]string `json:"tags,omitempty"`
}

// InMemoryMetrics implements MetricsService using in-memory storage
type InMemoryMetrics struct {
	mu         sync.RWMutex
	counters   map[string]*Counter
	histograms map[string]*Histogram
	gauges     map[string]*Gauge
	startTime  time.Time
}

// NewInMemoryMetrics creates a new in-memory metrics service
func NewInMemoryMetrics() *InMemoryMetrics {
	return &InMemoryMetrics{
		counters:   make(map[string]*Counter),
		histograms: make(map[string]*Histogram),
		gauges:     make(map[string]*Gauge),
		startTime:  time.Now(),
	}
}

// IncrementCounter increments a counter metric
func (m *InMemoryMetrics) IncrementCounter(name string, tags map[string]string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	key := m.generateKey(name, tags)
	if counter, exists := m.counters[key]; exists {
		counter.Value++
	} else {
		m.counters[key] = &Counter{
			Value: 1,
			Tags:  tags,
		}
	}
}

// RecordDuration records a duration measurement
func (m *InMemoryMetrics) RecordDuration(name string, duration time.Duration, tags map[string]string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	key := m.generateKey(name, tags)
	if histogram, exists := m.histograms[key]; exists {
		histogram.Count++
		histogram.Sum += duration
		
		if duration < histogram.Min || histogram.Min == 0 {
			histogram.Min = duration
		}
		if duration > histogram.Max {
			histogram.Max = duration
		}
		
		histogram.Average = histogram.Sum / time.Duration(histogram.Count)
		
		// Update buckets for percentile calculation
		m.updateBuckets(histogram, duration)
	} else {
		buckets := make(map[string]int64)
		m.histograms[key] = &Histogram{
			Count:   1,
			Sum:     duration,
			Min:     duration,
			Max:     duration,
			Average: duration,
			Tags:    tags,
			Buckets: buckets,
		}
		m.updateBuckets(m.histograms[key], duration)
	}
}

// SetGauge sets a gauge value
func (m *InMemoryMetrics) SetGauge(name string, value float64, tags map[string]string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	key := m.generateKey(name, tags)
	m.gauges[key] = &Gauge{
		Value: value,
		Tags:  tags,
	}
}

// GetMetrics returns all collected metrics
func (m *InMemoryMetrics) GetMetrics() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	metrics := make(map[string]interface{})
	
	// Add system info
	metrics["system"] = map[string]interface{}{
		"uptime":     time.Since(m.startTime).String(),
		"start_time": m.startTime.Format(time.RFC3339),
	}
	
	// Add counters
	if len(m.counters) > 0 {
		counters := make(map[string]*Counter)
		for k, v := range m.counters {
			counters[k] = v
		}
		metrics["counters"] = counters
	}
	
	// Add histograms
	if len(m.histograms) > 0 {
		histograms := make(map[string]*Histogram)
		for k, v := range m.histograms {
			histograms[k] = v
		}
		metrics["histograms"] = histograms
	}
	
	// Add gauges
	if len(m.gauges) > 0 {
		gauges := make(map[string]*Gauge)
		for k, v := range m.gauges {
			gauges[k] = v
		}
		metrics["gauges"] = gauges
	}
	
	return metrics
}

// Reset clears all metrics
func (m *InMemoryMetrics) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.counters = make(map[string]*Counter)
	m.histograms = make(map[string]*Histogram)
	m.gauges = make(map[string]*Gauge)
	m.startTime = time.Now()
}

// generateKey creates a unique key for metrics with tags
func (m *InMemoryMetrics) generateKey(name string, tags map[string]string) string {
	key := name
	if tags != nil {
		for k, v := range tags {
			key += "|" + k + ":" + v
		}
	}
	return key
}

// updateBuckets updates histogram buckets for percentile calculation
func (m *InMemoryMetrics) updateBuckets(histogram *Histogram, duration time.Duration) {
	if histogram.Buckets == nil {
		histogram.Buckets = make(map[string]int64)
	}
	
	// Define duration buckets (in milliseconds)
	buckets := []struct {
		name  string
		limit time.Duration
	}{
		{"1ms", time.Millisecond},
		{"5ms", 5 * time.Millisecond},
		{"10ms", 10 * time.Millisecond},
		{"25ms", 25 * time.Millisecond},
		{"50ms", 50 * time.Millisecond},
		{"100ms", 100 * time.Millisecond},
		{"250ms", 250 * time.Millisecond},
		{"500ms", 500 * time.Millisecond},
		{"1s", time.Second},
		{"2.5s", 2500 * time.Millisecond},
		{"5s", 5 * time.Second},
		{"10s", 10 * time.Second},
	}
	
	for _, bucket := range buckets {
		if duration <= bucket.limit {
			histogram.Buckets[bucket.name]++
		}
	}
	
	// Always increment the "+Inf" bucket
	histogram.Buckets["+Inf"]++
}

// PerformanceMonitor wraps services with performance monitoring
type PerformanceMonitor struct {
	metrics MetricsService
}

// NewPerformanceMonitor creates a new performance monitor
func NewPerformanceMonitor(metrics MetricsService) *PerformanceMonitor {
	return &PerformanceMonitor{
		metrics: metrics,
	}
}

// MonitoredSearchService wraps SearchService with performance monitoring
type MonitoredSearchService struct {
	searchService SearchService
	monitor       *PerformanceMonitor
}

// NewMonitoredSearchService creates a monitored search service
func NewMonitoredSearchService(searchService SearchService, monitor *PerformanceMonitor) *MonitoredSearchService {
	return &MonitoredSearchService{
		searchService: searchService,
		monitor:       monitor,
	}
}

// SemanticSearch performs semantic search with monitoring
func (m *MonitoredSearchService) SemanticSearch(ctx context.Context, query string, limit int) ([]SimilarityResult, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		m.monitor.metrics.RecordDuration("search.semantic.duration", duration, map[string]string{
			"operation": "semantic_search",
		})
		m.monitor.metrics.IncrementCounter("search.semantic.requests", map[string]string{
			"operation": "semantic_search",
		})
	}()
	
	results, err := m.searchService.SemanticSearch(ctx, query, limit)
	
	if err != nil {
		m.monitor.metrics.IncrementCounter("search.semantic.errors", map[string]string{
			"operation": "semantic_search",
		})
	} else {
		m.monitor.metrics.SetGauge("search.semantic.results_count", float64(len(results)), map[string]string{
			"operation": "semantic_search",
		})
	}
	
	return results, err
}

// SemanticSearchWithFilters performs filtered semantic search with monitoring
func (m *MonitoredSearchService) SemanticSearchWithFilters(ctx context.Context, req *SemanticSearchRequest) (*SemanticSearchResponse, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		m.monitor.metrics.RecordDuration("search.semantic_filtered.duration", duration, map[string]string{
			"operation": "semantic_search_filtered",
		})
		m.monitor.metrics.IncrementCounter("search.semantic_filtered.requests", map[string]string{
			"operation": "semantic_search_filtered",
		})
	}()
	
	response, err := m.searchService.SemanticSearchWithFilters(ctx, req)
	
	if err != nil {
		m.monitor.metrics.IncrementCounter("search.semantic_filtered.errors", map[string]string{
			"operation": "semantic_search_filtered",
		})
	} else if response != nil {
		m.monitor.metrics.SetGauge("search.semantic_filtered.results_count", float64(len(response.Results)), map[string]string{
			"operation": "semantic_search_filtered",
		})
	}
	
	return response, err
}

// HybridSearch performs hybrid search with monitoring
func (m *MonitoredSearchService) HybridSearch(ctx context.Context, query string, limit int, semanticWeight float64) ([]SimilarityResult, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		m.monitor.metrics.RecordDuration("search.hybrid.duration", duration, map[string]string{
			"operation": "hybrid_search",
		})
		m.monitor.metrics.IncrementCounter("search.hybrid.requests", map[string]string{
			"operation": "hybrid_search",
		})
	}()
	
	results, err := m.searchService.HybridSearch(ctx, query, limit, semanticWeight)
	
	if err != nil {
		m.monitor.metrics.IncrementCounter("search.hybrid.errors", map[string]string{
			"operation": "hybrid_search",
		})
	} else {
		m.monitor.metrics.SetGauge("search.hybrid.results_count", float64(len(results)), map[string]string{
			"operation": "hybrid_search",
		})
	}
	
	return results, err
}

// GraphSearch performs graph search with monitoring
func (m *MonitoredSearchService) GraphSearch(ctx context.Context, query *GraphQuery) (*GraphResult, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		m.monitor.metrics.RecordDuration("search.graph.duration", duration, map[string]string{
			"operation": "graph_search",
		})
		m.monitor.metrics.IncrementCounter("search.graph.requests", map[string]string{
			"operation": "graph_search",
		})
	}()
	
	result, err := m.searchService.GraphSearch(ctx, query)
	
	if err != nil {
		m.monitor.metrics.IncrementCounter("search.graph.errors", map[string]string{
			"operation": "graph_search",
		})
	} else if result != nil {
		m.monitor.metrics.SetGauge("search.graph.nodes_count", float64(len(result.Nodes)), map[string]string{
			"operation": "graph_search",
		})
		m.monitor.metrics.SetGauge("search.graph.edges_count", float64(len(result.Edges)), map[string]string{
			"operation": "graph_search",
		})
	}
	
	return result, err
}

// SearchByTag performs tag search with monitoring
func (m *MonitoredSearchService) SearchByTag(ctx context.Context, tagContent string) ([]ChunkWithTags, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		m.monitor.metrics.RecordDuration("search.tag.duration", duration, map[string]string{
			"operation": "tag_search",
		})
		m.monitor.metrics.IncrementCounter("search.tag.requests", map[string]string{
			"operation": "tag_search",
		})
	}()
	
	results, err := m.searchService.SearchByTag(ctx, tagContent)
	
	if err != nil {
		m.monitor.metrics.IncrementCounter("search.tag.errors", map[string]string{
			"operation": "tag_search",
		})
	} else {
		m.monitor.metrics.SetGauge("search.tag.results_count", float64(len(results)), map[string]string{
			"operation": "tag_search",
		})
	}
	
	return results, err
}

// SearchChunks performs chunk search with monitoring
func (m *MonitoredSearchService) SearchChunks(ctx context.Context, query string, filters map[string]interface{}) ([]ChunkRecord, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		m.monitor.metrics.RecordDuration("search.chunks.duration", duration, map[string]string{
			"operation": "chunk_search",
		})
		m.monitor.metrics.IncrementCounter("search.chunks.requests", map[string]string{
			"operation": "chunk_search",
		})
	}()
	
	results, err := m.searchService.SearchChunks(ctx, query, filters)
	
	if err != nil {
		m.monitor.metrics.IncrementCounter("search.chunks.errors", map[string]string{
			"operation": "chunk_search",
		})
	} else {
		m.monitor.metrics.SetGauge("search.chunks.results_count", float64(len(results)), map[string]string{
			"operation": "chunk_search",
		})
	}
	
	return results, err
}