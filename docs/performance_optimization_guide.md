# Performance Optimization Guide

## Overview

This guide provides comprehensive information about performance testing, optimization, and monitoring for the semantic-text-processor system. It covers the implementation of Task 8 requirements including large-scale data testing and query optimization.

## Performance Testing Framework

### Architecture

The performance testing framework consists of the following components:

1. **PerformanceTestOrchestrator**: Central coordinator for all performance tests
2. **DataGenerationService**: Creates large-scale test datasets (up to million-level)
3. **LoadTestExecutor**: Manages concurrent test execution with progressive load
4. **OptimizationAnalyzer**: Analyzes performance and generates recommendations
5. **ContinuousMonitor**: Real-time performance monitoring
6. **ReportGenerator**: Comprehensive performance reporting

### Usage

#### Basic Performance Test

```bash
# Run basic performance test
go run cmd/performance-test/main.go

# Million-level dataset testing
go run cmd/performance-test/main.go -generate-million -max-users 100 -duration 10m

# Custom configuration
go run cmd/performance-test/main.go \
  -dataset-size 500000 \
  -max-users 50 \
  -duration 5m \
  -slow-threshold 200ms \
  -cpu-threshold 70
```

#### Configuration Options

| Flag | Description | Default |
|------|-------------|---------|
| `-dataset-size` | Number of test records | 100,000 |
| `-max-users` | Maximum concurrent users | 50 |
| `-duration` | Test duration per step | 5m |
| `-generate-million` | Generate million-level dataset | false |
| `-regression` | Enable regression testing | false |
| `-monitoring` | Enable resource monitoring | true |
| `-slow-threshold` | Slow query threshold | 500ms |
| `-memory-limit` | Memory limit in MB | 1024 |
| `-cpu-threshold` | CPU usage threshold % | 80.0 |

## Performance Optimization Strategies

### Database Optimization

#### Vector Index Optimization

```sql
-- Create optimal vector index for semantic search
CREATE INDEX idx_chunks_embedding_cosine
ON chunks USING ivfflat (embedding vector_cosine_ops)
WITH (lists = 1000);

-- Optimize for different similarity metrics
CREATE INDEX idx_chunks_embedding_ip
ON chunks USING ivfflat (embedding vector_ip_ops);

-- For exact search when needed
CREATE INDEX idx_chunks_embedding_l2
ON chunks USING ivfflat (embedding vector_l2_ops);
```

#### Tag Search Optimization

```sql
-- Composite index for tag searches
CREATE INDEX idx_chunk_tags_content_chunk
ON chunk_tags (tag_content, chunk_id);

-- Partial index for active tags
CREATE INDEX idx_chunk_tags_active
ON chunk_tags (tag_content)
WHERE active = true;

-- GIN index for multiple tag queries
CREATE INDEX idx_chunk_tags_gin
ON chunk_tags USING gin (tag_content gin_trgm_ops);
```

#### Hierarchy Optimization

```sql
-- Optimize hierarchical queries
CREATE INDEX idx_chunks_parent_level
ON chunks (parent_id, level);

-- Materialized path for fast hierarchy traversal
CREATE INDEX idx_chunks_path
ON chunks USING gin (path);
```

### Cache Optimization

#### Configuration Tuning

```bash
# Environment variables for cache optimization
export CACHE_ENABLED=true
export CACHE_MAX_SIZE=10000
export CACHE_DEFAULT_TTL=30m
export CACHE_CLEANUP_INTERVAL=5m
```

#### Cache Strategy

1. **Semantic Search Caching**
   - Cache embedding results for 30 minutes
   - Use query hash as cache key
   - Implement cache warming for popular queries

2. **Tag Search Caching**
   - Cache tag combinations for 1 hour
   - Use LRU eviction policy
   - Pre-warm frequently used tag patterns

3. **Result Set Caching**
   - Cache final result sets for 15 minutes
   - Implement cache invalidation on data updates
   - Use compression for large result sets

### Query Optimization

#### Semantic Search Optimization

```go
// Optimized semantic search configuration
req := &models.OptimizedSearchRequest{
    Query:         query,
    Limit:         20,
    MinSimilarity: 0.7,
    UseCache:      true,
    PreloadHints:  []string{"tags", "metadata"},
}

// Use appropriate similarity thresholds
// - High precision: > 0.8
// - Balanced: 0.6-0.8
// - High recall: < 0.6
```

#### Tag Search Optimization

```go
// Efficient tag query structure
req := &models.TagSearchRequest{
    Tags:            []string{"programming", "go"},
    CombinationMode: "AND",
    Limit:          50,
    SortByRelevance: true,
    Filters: map[string]interface{}{
        "created_after": time.Now().AddDate(0, -1, 0),
    },
}
```

### Memory Optimization

#### Configuration

```bash
# Memory management settings
export MEMORY_LIMIT=2048  # MB
export GOGC=100           # GC percentage
export GOMEMLIMIT=2GiB    # Go memory limit
```

#### Best Practices

1. **Batch Processing**
   - Process data in batches of 1000-10000 records
   - Use streaming for large result sets
   - Implement proper connection pooling

2. **Memory Monitoring**
   - Monitor heap usage patterns
   - Track GC frequency and pause times
   - Set appropriate memory limits

3. **Resource Cleanup**
   - Close database connections promptly
   - Clear large slices and maps
   - Use object pooling for frequently allocated objects

### Connection Pool Optimization

```bash
# Database connection settings
export DB_MAX_CONNECTIONS=50
export DB_MAX_IDLE_CONNECTIONS=10
export DB_MAX_LIFETIME=1h
export DB_MAX_IDLE_TIME=10m
```

## Performance Monitoring

### Real-time Monitoring

The system provides continuous monitoring of:

- Memory usage (peak and average)
- CPU utilization
- Database connection pool status
- Cache hit rates
- Query response times
- Error rates
- Garbage collection statistics

### Metrics Collection

```go
// Enable metrics collection
cfg.Performance.MetricsEnabled = true
cfg.Performance.MonitoringEnabled = true

// Access metrics
metrics := service.GetMetrics()
```

### Alerting Thresholds

| Metric | Warning | Critical |
|--------|---------|----------|
| Response Time | > 500ms | > 2s |
| Error Rate | > 1% | > 5% |
| Memory Usage | > 80% | > 95% |
| CPU Usage | > 70% | > 90% |
| Cache Hit Rate | < 80% | < 60% |

## Performance Testing Scenarios

### Load Testing Patterns

#### Progressive Load Test

```go
loadSteps := []int{1, 5, 10, 25, 50, 100, 200}
```

#### Spike Testing

```go
// Sudden load increase
loadSteps := []int{1, 1, 1, 100, 100, 100, 1}
```

#### Soak Testing

```go
// Extended duration testing
duration := 4 * time.Hour
users := 50
```

### Test Data Generation

#### Realistic Data Patterns

1. **Content Variety**
   - Technical documentation (30%)
   - Business content (40%)
   - Scientific papers (30%)

2. **Tag Distribution**
   - Popular tags: 20% frequency
   - Common tags: 60% frequency
   - Rare tags: 20% frequency

3. **Size Distribution**
   - Short content: 100-500 characters (40%)
   - Medium content: 500-2000 characters (50%)
   - Long content: 2000+ characters (10%)

## Optimization Analysis

### Automated Recommendations

The system automatically generates optimization recommendations based on:

1. **Slow Query Analysis**
   - Query pattern identification
   - Index usage analysis
   - Execution plan optimization

2. **Cache Performance Analysis**
   - Hit rate optimization
   - TTL tuning recommendations
   - Size optimization

3. **Resource Utilization Analysis**
   - Memory usage patterns
   - CPU utilization optimization
   - Connection pool tuning

### Performance Scoring

The system calculates an overall performance score (0-100) based on:

- Response time (30% weight)
- Throughput (20% weight)
- Success rate (20% weight)
- Cache hit rate (15% weight)
- Resource efficiency (15% weight)

## Troubleshooting

### Common Performance Issues

#### High Response Times

1. **Symptoms**: Average response time > 1s
2. **Diagnosis**: Check slow query log, analyze query plans
3. **Solutions**:
   - Add appropriate indexes
   - Optimize query structure
   - Implement result caching

#### Low Cache Hit Rate

1. **Symptoms**: Hit rate < 70%
2. **Diagnosis**: Analyze cache access patterns
3. **Solutions**:
   - Increase cache size
   - Optimize TTL settings
   - Implement cache warming

#### Memory Leaks

1. **Symptoms**: Continuously increasing memory usage
2. **Diagnosis**: Monitor heap growth, analyze GC patterns
3. **Solutions**:
   - Fix resource cleanup
   - Optimize data structures
   - Implement proper connection management

#### Database Connection Issues

1. **Symptoms**: Connection timeout errors
2. **Diagnosis**: Monitor connection pool metrics
3. **Solutions**:
   - Increase pool size
   - Optimize connection lifecycle
   - Implement connection retry logic

## Best Practices

### Development

1. **Performance Testing**
   - Run performance tests on every major change
   - Maintain performance baselines
   - Set up automated performance regression detection

2. **Code Optimization**
   - Use appropriate data structures
   - Minimize database queries
   - Implement proper caching strategies

3. **Monitoring**
   - Monitor key performance metrics
   - Set up alerting for performance degradation
   - Regular performance reviews

### Deployment

1. **Environment Configuration**
   - Tune database parameters
   - Optimize cache settings
   - Set appropriate resource limits

2. **Scaling Strategy**
   - Horizontal scaling for read operations
   - Vertical scaling for write operations
   - Database sharding for large datasets

3. **Monitoring and Alerting**
   - Production performance monitoring
   - Real-time alerting system
   - Performance dashboard

## Continuous Improvement

### Performance Review Process

1. **Weekly Reviews**
   - Analyze performance trends
   - Review slow query reports
   - Update optimization recommendations

2. **Monthly Analysis**
   - Comprehensive performance analysis
   - Capacity planning review
   - Architecture optimization planning

3. **Quarterly Optimization**
   - Major performance improvements
   - Infrastructure upgrades
   - Performance testing strategy updates

### Metrics Tracking

Track the following metrics over time:

- Performance test scores
- Response time percentiles
- Error rates
- Resource utilization
- User satisfaction metrics

This comprehensive guide ensures optimal performance for the semantic-text-processor system and provides the foundation for continuous performance improvement.