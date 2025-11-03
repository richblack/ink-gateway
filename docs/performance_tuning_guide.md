# Performance Tuning and Troubleshooting Guide

## Table of Contents

1. [Overview](#overview)
2. [Performance Baseline Metrics](#performance-baseline-metrics)
3. [System Resource Optimization](#system-resource-optimization)
4. [Database Performance Tuning](#database-performance-tuning)
5. [Cache Optimization](#cache-optimization)
6. [Search Performance Optimization](#search-performance-optimization)
7. [API Response Time Optimization](#api-response-time-optimization)
8. [Memory Management](#memory-management)
9. [Troubleshooting Common Performance Issues](#troubleshooting-common-performance-issues)
10. [Monitoring and Alerting Setup](#monitoring-and-alerting-setup)
11. [Load Testing and Capacity Planning](#load-testing-and-capacity-planning)

## Overview

This guide provides comprehensive performance tuning strategies for the Semantic Text Processor, focusing on the unified chunk system and multi-database architecture. The system's performance depends on several key components:

- **API Layer**: Go HTTP server with routing and middleware
- **Caching Layer**: In-memory and distributed caching
- **Database Layer**: PostgreSQL with PGVector and Apache AGE via Supabase
- **External Services**: LLM and embedding APIs
- **Search Infrastructure**: Semantic, graph, and hybrid search

### Performance Goals

**Target Metrics**:
- API response time: < 200ms (95th percentile)
- Search latency: < 500ms for semantic search
- Cache hit rate: > 80%
- Memory usage: < 2GB under normal load
- Database query time: < 100ms (95th percentile)

## Performance Baseline Metrics

### Establishing Baselines

**1. API Performance Baseline**:
```bash
# Run baseline performance test
ab -n 1000 -c 10 http://localhost:8080/api/v1/health

# Monitor response times
curl -s http://localhost:8080/api/v1/metrics | jq '.histograms.http_request_duration'
```

**2. Search Performance Baseline**:
```bash
# Semantic search baseline
time curl -X POST http://localhost:8080/api/v1/search/semantic \
  -H "Content-Type: application/json" \
  -d '{"query": "test query", "limit": 10}'

# Graph search baseline
time curl -X POST http://localhost:8080/api/v1/search/graph \
  -H "Content-Type: application/json" \
  -d '{"entity_name": "test entity", "max_depth": 2}'
```

**3. Cache Performance Baseline**:
```bash
# Check initial cache stats
curl http://localhost:8080/api/v1/cache/stats | jq '{hit_rate, size, evictions}'
```

### Key Performance Indicators (KPIs)

1. **Throughput**: Requests per second
2. **Latency**: Response time percentiles (50th, 95th, 99th)
3. **Error Rate**: Percentage of failed requests
4. **Resource Utilization**: CPU, memory, disk I/O
5. **Cache Efficiency**: Hit rate, eviction rate
6. **Database Performance**: Query execution time, connection pool usage

## System Resource Optimization

### CPU Optimization

**1. Go Runtime Configuration**:
```bash
# Set optimal GOMAXPROCS (usually equal to CPU cores)
export GOMAXPROCS=4

# Enable CPU profiling for analysis
export ENABLE_CPU_PROFILE=true
export CPU_PROFILE_DURATION=30s
```

**2. Process Priority**:
```bash
# Set higher priority for the main process
nice -n -10 ./semantic-text-processor

# Use taskset for CPU affinity on multi-core systems
taskset -c 0-3 ./semantic-text-processor
```

**3. Monitoring CPU Usage**:
```bash
# Monitor CPU usage per process
top -p $(pgrep semantic-text-processor)

# Check CPU utilization patterns
iostat -x 1 10

# Profile CPU usage with Go tools
go tool pprof http://localhost:8080/debug/pprof/profile?seconds=30
```

### Memory Optimization

**1. Memory Configuration**:
```bash
# Set garbage collection target percentage
export GOGC=100

# Enable memory debugging
export GODEBUG=gctrace=1

# Set memory limits
export GOMEMLIMIT=1GiB
```

**2. Memory Monitoring**:
```bash
# Monitor memory usage
free -h
cat /proc/meminfo

# Go memory profiling
go tool pprof http://localhost:8080/debug/pprof/heap
```

## Database Performance Tuning

### PostgreSQL Optimization via Supabase

**1. Connection Pool Optimization**:
```bash
# Configure connection pool size
export DB_MAX_OPEN_CONNECTIONS=25
export DB_MAX_IDLE_CONNECTIONS=5
export DB_CONNECTION_LIFETIME=5m
```

**2. Query Optimization**:

**Monitor Slow Queries**:
```sql
-- Check slow queries in Supabase dashboard
SELECT query, mean_exec_time, calls, total_exec_time
FROM pg_stat_statements
ORDER BY mean_exec_time DESC
LIMIT 10;
```

**Optimize Common Queries**:
```sql
-- Create indexes for frequent queries
CREATE INDEX CONCURRENTLY idx_chunks_text_id ON chunks(text_id);
CREATE INDEX CONCURRENTLY idx_chunks_parent_id ON chunks(parent_chunk_id);
CREATE INDEX CONCURRENTLY idx_chunks_template_id ON chunks(template_chunk_id);
CREATE INDEX CONCURRENTLY idx_chunks_is_template ON chunks(is_template);

-- Composite indexes for common filter combinations
CREATE INDEX CONCURRENTLY idx_chunks_text_template ON chunks(text_id, is_template);
CREATE INDEX CONCURRENTLY idx_chunks_parent_sequence ON chunks(parent_chunk_id, sequence_number);
```

### PGVector Performance

**1. Vector Index Optimization**:
```sql
-- Create optimal vector indexes
CREATE INDEX CONCURRENTLY embeddings_vector_idx
ON embeddings USING ivfflat (vector vector_cosine_ops)
WITH (lists = 100);

-- For large datasets, consider HNSW index
CREATE INDEX CONCURRENTLY embeddings_hnsw_idx
ON embeddings USING hnsw (vector vector_cosine_ops)
WITH (m = 16, ef_construction = 64);
```

**2. Vector Search Optimization**:
```bash
# Configure vector search parameters
export VECTOR_SEARCH_LISTS=100
export VECTOR_SEARCH_PROBES=10
export VECTOR_SEARCH_EF_SEARCH=40
```

### Apache AGE Graph Performance

**1. Graph Index Optimization**:
```sql
-- Create indexes on graph properties
CREATE INDEX ON ag_label_vertex(properties);
CREATE INDEX ON ag_label_edge(properties);

-- Optimize for common graph traversals
CREATE INDEX ON ag_label_vertex((properties->>'entity_type'));
CREATE INDEX ON ag_label_edge((properties->>'relationship_type'));
```

## Cache Optimization

### Cache Configuration Tuning

**1. Cache Size Optimization**:
```bash
# Calculate optimal cache size (aim for 80% hit rate)
# Monitor current cache statistics
curl http://localhost:8080/api/v1/cache/stats

# Adjust cache size based on memory availability and hit rate
export CACHE_MAX_SIZE=5000  # Increase if hit rate < 80%
export CACHE_DEFAULT_TTL=3600  # 1 hour default
export CACHE_MAX_TTL=86400     # 24 hours maximum
```

**2. Cache Strategy Optimization**:
```bash
# Configure cache strategies by operation type
export CACHE_STRATEGY_SEARCH=LRU
export CACHE_STRATEGY_CHUNKS=LFU
export CACHE_TTL_SEARCH=1800      # 30 minutes
export CACHE_TTL_CHUNKS=7200      # 2 hours
export CACHE_TTL_TEMPLATES=86400  # 24 hours
```

**3. Cache Warming Strategies**:
```bash
# Pre-populate cache with frequently accessed data
curl -X POST http://localhost:8080/api/v1/cache/warm \
  -H "Content-Type: application/json" \
  -d '{"patterns": ["popular_searches", "common_templates"]}'
```

### Cache Performance Monitoring

```bash
# Real-time cache monitoring
watch -n 5 'curl -s http://localhost:8080/api/v1/cache/stats | jq "{hit_rate, size, evictions, memory_usage_mb: (.memory_usage_bytes / 1024 / 1024)}"'

# Cache hit rate analysis
curl -s http://localhost:8080/api/v1/metrics | jq '.gauges | with_entries(select(.key | contains("cache")))'
```

## Search Performance Optimization

### Semantic Search Optimization

**1. Embedding Cache Optimization**:
```bash
# Configure embedding cache
export EMBEDDING_CACHE_SIZE=10000
export EMBEDDING_CACHE_TTL=86400  # 24 hours

# Pre-compute embeddings for common queries
export PRECOMPUTE_EMBEDDINGS=true
export EMBEDDING_BATCH_SIZE=100
```

**2. Vector Search Parameters**:
```bash
# Optimize vector search performance vs accuracy
export VECTOR_SEARCH_QUALITY=balanced  # fast|balanced|accurate
export VECTOR_SIMILARITY_THRESHOLD=0.7
export VECTOR_MAX_RESULTS=100
```

### Graph Search Optimization

**1. Graph Traversal Optimization**:
```bash
# Configure graph search limits
export GRAPH_MAX_DEPTH=5
export GRAPH_MAX_NODES=1000
export GRAPH_TIMEOUT_SECONDS=30

# Enable graph query caching
export GRAPH_CACHE_ENABLED=true
export GRAPH_CACHE_TTL=3600
```

**2. Graph Query Optimization**:
```sql
-- Optimize graph queries with proper indexing
CREATE INDEX ON ag_label_vertex USING gin((properties->'entity_type'));
CREATE INDEX ON ag_label_edge USING gin((properties->'relationship_type'));
```

### Hybrid Search Optimization

**1. Search Weight Optimization**:
```bash
# Configure optimal search weights based on use case
export HYBRID_SEMANTIC_WEIGHT=0.7
export HYBRID_TEXT_WEIGHT=0.3
export HYBRID_GRAPH_WEIGHT=0.0  # Disable if not needed

# Enable search result caching
export SEARCH_RESULT_CACHE_ENABLED=true
export SEARCH_RESULT_CACHE_TTL=1800  # 30 minutes
```

## API Response Time Optimization

### Middleware Optimization

**1. Compression**:
```bash
# Enable response compression
export ENABLE_GZIP_COMPRESSION=true
export GZIP_COMPRESSION_LEVEL=6
```

**2. Connection Keep-Alive**:
```bash
# Optimize HTTP connection settings
export HTTP_READ_TIMEOUT=30s
export HTTP_WRITE_TIMEOUT=30s
export HTTP_IDLE_TIMEOUT=120s
export HTTP_MAX_HEADER_BYTES=1048576
```

**3. Rate Limiting Optimization**:
```bash
# Configure rate limiting for optimal performance
export RATE_LIMIT_ENABLED=true
export RATE_LIMIT_RPS=100
export RATE_LIMIT_BURST=200
export RATE_LIMIT_CLEANUP_INTERVAL=1m
```

### Response Optimization

**1. Pagination Optimization**:
```bash
# Optimize pagination parameters
export DEFAULT_PAGE_SIZE=20
export MAX_PAGE_SIZE=100
export ENABLE_CURSOR_PAGINATION=true
```

**2. Response Caching**:
```bash
# Enable HTTP response caching
export HTTP_CACHE_ENABLED=true
export HTTP_CACHE_MAX_AGE=300  # 5 minutes
export HTTP_CACHE_CONTROL="public, max-age=300"
```

## Memory Management

### Go Memory Optimization

**1. Garbage Collection Tuning**:
```bash
# Optimize GC for lower latency
export GOGC=50      # More frequent GC, lower memory usage
export GOMAXPROCS=4  # Match CPU cores

# For high-throughput scenarios
export GOGC=200     # Less frequent GC, higher throughput
```

**2. Memory Pool Usage**:
```go
// Example optimization in application code
var bufferPool = sync.Pool{
    New: func() interface{} {
        return make([]byte, 0, 1024)
    },
}
```

### Memory Leak Detection

**1. Memory Profiling**:
```bash
# Generate memory profile
go tool pprof http://localhost:8080/debug/pprof/heap

# Monitor memory growth over time
watch -n 30 'ps -p $(pgrep semantic-text-processor) -o pid,vsz,rss,pcpu,pmem'
```

**2. Memory Monitoring Commands**:
```bash
# Check memory usage patterns
cat /proc/$(pgrep semantic-text-processor)/status | grep -E "(VmPeak|VmSize|VmRSS|VmData)"

# Monitor memory allocation rate
go tool pprof -alloc_space http://localhost:8080/debug/pprof/heap
```

## Troubleshooting Common Performance Issues

### High Response Times

**Symptoms**:
- API responses > 1 second
- Timeout errors
- Poor user experience

**Diagnosis Steps**:
```bash
# 1. Check system resources
top -p $(pgrep semantic-text-processor)
iostat -x 1 5

# 2. Analyze request patterns
curl -s http://localhost:8080/api/v1/metrics | jq '.histograms.http_request_duration'

# 3. Check database performance
curl -s http://localhost:8080/api/v1/health | jq '.components.database'

# 4. Verify cache performance
curl -s http://localhost:8080/api/v1/cache/stats | jq '{hit_rate, avg_response_time_ms}'
```

**Resolution Actions**:
1. **Increase cache size** if hit rate < 70%
2. **Optimize database queries** for slow endpoints
3. **Scale horizontally** if CPU/memory constrained
4. **Tune GC settings** for memory-intensive operations

### High Memory Usage

**Symptoms**:
- Memory usage > 2GB
- Out of memory errors
- Frequent garbage collection

**Diagnosis Steps**:
```bash
# 1. Memory profiling
go tool pprof http://localhost:8080/debug/pprof/heap

# 2. Check for memory leaks
go tool pprof -base heap_before.pprof heap_after.pprof

# 3. Monitor GC behavior
export GODEBUG=gctrace=1
tail -f /var/log/semantic-text-processor.log | grep "gc "
```

**Resolution Actions**:
1. **Reduce cache size** temporarily
2. **Optimize data structures** in hot code paths
3. **Implement object pooling** for frequently allocated objects
4. **Tune GOGC** for more aggressive collection

### Database Connection Issues

**Symptoms**:
- Connection pool exhaustion
- Database timeout errors
- Health check failures

**Diagnosis Steps**:
```bash
# 1. Check connection pool status
curl -s http://localhost:8080/api/v1/metrics | jq '.gauges | with_entries(select(.key | contains("db_connections")))'

# 2. Monitor Supabase dashboard
# Check active connections and query performance

# 3. Test direct database connectivity
curl -H "apikey: $SUPABASE_API_KEY" "$SUPABASE_URL/rest/v1/"
```

**Resolution Actions**:
1. **Increase connection pool size** if consistently exhausted
2. **Reduce connection lifetime** to prevent stale connections
3. **Implement connection retry logic** with exponential backoff
4. **Monitor query patterns** for optimization opportunities

## Monitoring and Alerting Setup

### Performance Metrics Collection

**1. Application Metrics**:
```bash
# Configure metrics collection
export METRICS_ENABLED=true
export METRICS_INTERVAL=30s
export METRICS_RETENTION=24h

# Export metrics to external systems
export METRICS_EXPORT_ENABLED=true
export METRICS_EXPORT_URL=http://prometheus:9090/api/v1/write
```

**2. System Metrics**:
```bash
# Install and configure monitoring tools
# Node Exporter for system metrics
wget https://github.com/prometheus/node_exporter/releases/download/v1.6.1/node_exporter-1.6.1.linux-amd64.tar.gz
tar xvfz node_exporter-1.6.1.linux-amd64.tar.gz
./node_exporter &

# Monitor with htop
htop -p $(pgrep semantic-text-processor)
```

### Alert Configuration

**1. Critical Performance Alerts**:
```yaml
# Example Prometheus alert rules
groups:
  - name: semantic-text-processor
    rules:
      - alert: HighResponseTime
        expr: histogram_quantile(0.95, http_request_duration_seconds) > 0.5
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High API response times detected"

      - alert: LowCacheHitRate
        expr: cache_hit_rate < 0.7
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "Cache hit rate below threshold"

      - alert: HighMemoryUsage
        expr: process_resident_memory_bytes > 2147483648  # 2GB
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "Memory usage exceeding limits"
```

**2. Dashboard Configuration**:
```bash
# Key metrics to dashboard:
# - Request rate and response time
# - Error rate by endpoint
# - Cache hit rate and size
# - Memory and CPU usage
# - Database query performance
# - Search operation latency
```

## Load Testing and Capacity Planning

### Load Testing Setup

**1. Basic Load Testing**:
```bash
# Install Apache Bench
sudo apt-get install apache2-utils

# Test API endpoints
ab -n 10000 -c 50 http://localhost:8080/api/v1/health

# Test search endpoints
ab -n 1000 -c 10 -p search_payload.json -T application/json \
   http://localhost:8080/api/v1/search/semantic
```

**2. Advanced Load Testing with wrk**:
```bash
# Install wrk
git clone https://github.com/wg/wrk.git
cd wrk && make

# Run comprehensive load test
wrk -t12 -c400 -d30s --script=search_test.lua http://localhost:8080/api/v1/search/semantic
```

**3. Load Test Scripts**:
```lua
-- search_test.lua
request = function()
   wrk.method = "POST"
   wrk.body   = '{"query": "test query", "limit": 10}'
   wrk.headers["Content-Type"] = "application/json"
   return wrk.format(nil, "/api/v1/search/semantic")
end
```

### Capacity Planning

**1. Resource Requirements Calculation**:
```bash
# Based on load testing results, calculate:
# - Requests per second capacity
# - Memory usage per concurrent user
# - Database connection requirements
# - Cache size requirements

# Example calculation for 1000 concurrent users:
# - CPU: 4 cores minimum (8 cores recommended)
# - Memory: 4GB minimum (8GB recommended)
# - Database connections: 50-100
# - Cache size: 2GB
```

**2. Scaling Recommendations**:
```bash
# Horizontal scaling triggers:
# - CPU usage > 70% consistently
# - Memory usage > 80% consistently
# - Response time > 500ms (95th percentile)
# - Cache hit rate < 70%

# Vertical scaling considerations:
# - Increase memory for larger cache
# - Add CPU cores for better concurrency
# - Use SSD storage for faster I/O
```

### Performance Testing Checklist

- [ ] Baseline performance metrics established
- [ ] Load testing completed for all endpoints
- [ ] Memory usage profiled under load
- [ ] Database performance validated
- [ ] Cache efficiency verified
- [ ] Error rates under load acceptable
- [ ] Recovery time after load tested
- [ ] Monitoring and alerting configured
- [ ] Scaling thresholds defined
- [ ] Performance regression tests automated

---

For additional performance optimization techniques and troubleshooting procedures, refer to the [Operations Manual](operations.md) and [API Reference](api_reference.md).