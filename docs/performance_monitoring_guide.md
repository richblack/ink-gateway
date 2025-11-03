# Performance Monitoring and Metrics Interpretation Guide

## Table of Contents

1. [Overview](#overview)
2. [Monitoring Architecture](#monitoring-architecture)
3. [Core Performance Metrics](#core-performance-metrics)
4. [Metrics Collection and Storage](#metrics-collection-and-storage)
5. [Dashboard Setup and Configuration](#dashboard-setup-and-configuration)
6. [Metrics Interpretation and Analysis](#metrics-interpretation-and-analysis)
7. [Alerting and Thresholds](#alerting-and-thresholds)
8. [Performance Baseline Establishment](#performance-baseline-establishment)
9. [Troubleshooting with Metrics](#troubleshooting-with-metrics)
10. [Automated Performance Analysis](#automated-performance-analysis)

## Overview

This guide provides comprehensive instructions for monitoring and interpreting performance metrics in the Semantic Text Processor system. The monitoring approach focuses on proactive performance management and early issue detection.

### Monitoring Objectives

1. **Early Warning**: Detect performance degradation before user impact
2. **Root Cause Analysis**: Identify bottlenecks and optimization opportunities
3. **Capacity Planning**: Understand resource usage patterns and growth trends
4. **SLA Compliance**: Ensure service level agreements are met
5. **Optimization Guidance**: Data-driven performance improvements

### Key Performance Areas

- **API Response Times**: Request/response latency across all endpoints
- **Search Performance**: Semantic, graph, and hybrid search latencies
- **Resource Utilization**: CPU, memory, disk, and network usage
- **Database Performance**: Query execution times and connection health
- **Cache Efficiency**: Hit rates, eviction patterns, and memory usage
- **External Dependencies**: LLM and embedding API performance

## Monitoring Architecture

### Monitoring Stack Components

```
┌─────────────────────────────────────────────────────────────┐
│                    Monitoring Architecture                  │
├─────────────────────────────────────────────────────────────┤
│  Application (Semantic Text Processor)                     │
│  ├── Metrics Endpoint (/api/v1/metrics)                    │
│  ├── Health Endpoint (/api/v1/health)                      │
│  └── Profiling Endpoints (/debug/pprof/*)                  │
├─────────────────────────────────────────────────────────────┤
│  Metrics Collection                                         │
│  ├── Prometheus (Time-series DB)                           │
│  ├── Node Exporter (System metrics)                        │
│  └── Custom Exporters (Business metrics)                   │
├─────────────────────────────────────────────────────────────┤
│  Visualization & Alerting                                  │
│  ├── Grafana (Dashboards)                                  │
│  ├── Alertmanager (Alert routing)                          │
│  └── Custom Dashboards (Application-specific)              │
├─────────────────────────────────────────────────────────────┤
│  Log Management                                             │
│  ├── Application Logs                                      │
│  ├── System Logs                                           │
│  └── Access Logs                                           │
└─────────────────────────────────────────────────────────────┘
```

### Built-in Monitoring Features

**1. Application Metrics Endpoint**:
```bash
# Built-in metrics endpoint
curl http://localhost:8080/api/v1/metrics | jq '.'

# Expected response structure:
{
  "counters": {
    "http_requests_total": 12450,
    "http_requests_errors": 23,
    "cache_hits": 8900,
    "cache_misses": 1200,
    "search_operations_total": 450,
    "database_operations_total": 3200
  },
  "histograms": {
    "http_request_duration": {
      "count": 12450,
      "sum": 1245000,
      "mean": 100.0,
      "p50": 85.2,
      "p95": 250.1,
      "p99": 500.8,
      "max": 2500.0
    },
    "search_duration": {
      "count": 450,
      "sum": 225000,
      "mean": 500.0,
      "p50": 420.5,
      "p95": 850.2,
      "p99": 1200.0,
      "max": 3000.0
    }
  },
  "gauges": {
    "active_connections": 15,
    "memory_usage_bytes": 268435456,
    "cache_size": 1250,
    "goroutines": 45
  }
}
```

**2. Health Check Monitoring**:
```bash
# Comprehensive health check
curl http://localhost:8080/api/v1/health | jq '.'

# Response includes component-level health
{
  "status": "healthy",
  "timestamp": "2024-01-15T10:30:00Z",
  "uptime": "72h15m30s",
  "version": "1.2.0",
  "components": {
    "database": {
      "status": "healthy",
      "response_time_ms": 15,
      "last_check": "2024-01-15T10:30:00Z"
    },
    "cache": {
      "status": "healthy",
      "hit_rate": 0.85,
      "size": 1250,
      "response_time_ms": 2
    },
    "external_apis": {
      "llm_api": {
        "status": "healthy",
        "response_time_ms": 120
      },
      "embedding_api": {
        "status": "healthy",
        "response_time_ms": 85
      }
    }
  }
}
```

## Core Performance Metrics

### Application Performance Metrics

**1. Request/Response Metrics**:
```yaml
# Key metrics to monitor
http_requests_total:
  description: "Total number of HTTP requests"
  labels: [method, endpoint, status_code]
  type: counter

http_request_duration_seconds:
  description: "HTTP request duration in seconds"
  labels: [method, endpoint]
  type: histogram
  buckets: [0.001, 0.01, 0.1, 0.5, 1.0, 2.5, 5.0, 10.0]

http_request_size_bytes:
  description: "HTTP request size in bytes"
  labels: [method, endpoint]
  type: histogram

http_response_size_bytes:
  description: "HTTP response size in bytes"
  labels: [method, endpoint]
  type: histogram
```

**2. Search Performance Metrics**:
```yaml
search_operations_total:
  description: "Total number of search operations"
  labels: [search_type, status]
  type: counter

search_duration_seconds:
  description: "Search operation duration"
  labels: [search_type]
  type: histogram

search_results_count:
  description: "Number of search results returned"
  labels: [search_type]
  type: histogram

semantic_search_similarity_score:
  description: "Average similarity score for semantic searches"
  type: gauge
```

**3. Cache Performance Metrics**:
```yaml
cache_operations_total:
  description: "Total cache operations"
  labels: [operation_type, result]
  type: counter

cache_hit_rate:
  description: "Cache hit rate percentage"
  type: gauge

cache_size:
  description: "Current cache size"
  type: gauge

cache_evictions_total:
  description: "Total cache evictions"
  labels: [reason]
  type: counter

cache_memory_usage_bytes:
  description: "Cache memory usage in bytes"
  type: gauge
```

### System Resource Metrics

**1. CPU and Memory Metrics**:
```yaml
process_cpu_seconds_total:
  description: "Total user and system CPU time"
  type: counter

process_resident_memory_bytes:
  description: "Resident memory size in bytes"
  type: gauge

process_virtual_memory_bytes:
  description: "Virtual memory size in bytes"
  type: gauge

go_memstats_alloc_bytes:
  description: "Number of bytes allocated and still in use"
  type: gauge

go_memstats_sys_bytes:
  description: "Number of bytes obtained from system"
  type: gauge

go_gc_duration_seconds:
  description: "Time spent in garbage collection"
  type: summary
```

**2. Database Performance Metrics**:
```yaml
database_connections_active:
  description: "Number of active database connections"
  type: gauge

database_connections_idle:
  description: "Number of idle database connections"
  type: gauge

database_query_duration_seconds:
  description: "Database query duration"
  labels: [query_type]
  type: histogram

database_operations_total:
  description: "Total database operations"
  labels: [operation_type, status]
  type: counter
```

## Metrics Collection and Storage

### Prometheus Configuration

**1. Prometheus Setup**:
```yaml
# prometheus.yml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  - job_name: 'semantic-text-processor'
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: '/api/v1/metrics'
    scrape_interval: 15s
    scrape_timeout: 10s

  - job_name: 'node-exporter'
    static_configs:
      - targets: ['localhost:9100']

rule_files:
  - "semantic_text_processor_rules.yml"

alerting:
  alertmanagers:
    - static_configs:
        - targets:
          - alertmanager:9093
```

**2. Custom Metrics Export**:
```bash
#!/bin/bash
# metrics_exporter.sh

# Export application metrics to Prometheus format
curl -s http://localhost:8080/api/v1/metrics | \
    jq -r '
    .counters | to_entries[] | "# HELP \(.key) Application counter metric\n# TYPE \(.key) counter\n\(.key) \(.value)",
    .gauges | to_entries[] | "# HELP \(.key) Application gauge metric\n# TYPE \(.key) gauge\n\(.key) \(.value)",
    (.histograms | to_entries[] |
        "# HELP \(.key) Application histogram metric\n# TYPE \(.key) histogram\n" +
        "\(.key)_sum \(.value.sum)\n" +
        "\(.key)_count \(.value.count)\n" +
        "\(.key)_bucket{le=\"0.1\"} \(.value.p50)\n" +
        "\(.key)_bucket{le=\"1.0\"} \(.value.p95)\n" +
        "\(.key)_bucket{le=\"+Inf\"} \(.value.count)"
    )' > /tmp/semantic_text_processor_metrics.prom
```

### InfluxDB Integration (Alternative)

**1. InfluxDB Configuration**:
```bash
# Configure InfluxDB for time-series storage
export INFLUXDB_URL="http://localhost:8086"
export INFLUXDB_TOKEN="your-token"
export INFLUXDB_ORG="semantic-text-processor"
export INFLUXDB_BUCKET="metrics"

# Send metrics to InfluxDB
curl -s http://localhost:8080/api/v1/metrics | \
    jq -r '
    .counters | to_entries[] |
    "counters,metric=\(.key) value=\(.value) \(now)"
    ' | influx write --bucket metrics --org semantic-text-processor
```

## Dashboard Setup and Configuration

### Grafana Dashboard Configuration

**1. Main Performance Dashboard**:
```json
{
  "dashboard": {
    "title": "Semantic Text Processor - Performance Overview",
    "panels": [
      {
        "title": "Request Rate",
        "type": "stat",
        "targets": [
          {
            "expr": "rate(http_requests_total[5m])",
            "legendFormat": "{{method}} {{endpoint}}"
          }
        ]
      },
      {
        "title": "Response Time Percentiles",
        "type": "graph",
        "targets": [
          {
            "expr": "histogram_quantile(0.50, rate(http_request_duration_seconds_bucket[5m]))",
            "legendFormat": "p50"
          },
          {
            "expr": "histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))",
            "legendFormat": "p95"
          },
          {
            "expr": "histogram_quantile(0.99, rate(http_request_duration_seconds_bucket[5m]))",
            "legendFormat": "p99"
          }
        ]
      },
      {
        "title": "Error Rate",
        "type": "stat",
        "targets": [
          {
            "expr": "rate(http_requests_total{status_code=~\"5..\"}[5m]) / rate(http_requests_total[5m]) * 100",
            "legendFormat": "Error Rate %"
          }
        ]
      },
      {
        "title": "Cache Performance",
        "type": "graph",
        "targets": [
          {
            "expr": "cache_hit_rate",
            "legendFormat": "Hit Rate"
          },
          {
            "expr": "cache_size",
            "legendFormat": "Cache Size"
          }
        ]
      }
    ]
  }
}
```

**2. Search Performance Dashboard**:
```json
{
  "dashboard": {
    "title": "Search Performance Metrics",
    "panels": [
      {
        "title": "Search Operations Rate",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(search_operations_total[5m])",
            "legendFormat": "{{search_type}}"
          }
        ]
      },
      {
        "title": "Search Duration by Type",
        "type": "graph",
        "targets": [
          {
            "expr": "histogram_quantile(0.95, rate(search_duration_seconds_bucket[5m]))",
            "legendFormat": "{{search_type}} p95"
          }
        ]
      },
      {
        "title": "Search Results Distribution",
        "type": "heatmap",
        "targets": [
          {
            "expr": "rate(search_results_count_bucket[5m])",
            "legendFormat": "{{le}}"
          }
        ]
      }
    ]
  }
}
```

### Custom Dashboard Creation

**1. Dashboard as Code**:
```python
# dashboard_generator.py
import json

def create_performance_dashboard():
    dashboard = {
        "dashboard": {
            "title": "Semantic Text Processor Performance",
            "tags": ["semantic-text-processor", "performance"],
            "time": {
                "from": "now-1h",
                "to": "now"
            },
            "panels": []
        }
    }

    # Add request rate panel
    dashboard["dashboard"]["panels"].append({
        "id": 1,
        "title": "Request Rate (req/sec)",
        "type": "stat",
        "targets": [{
            "expr": "rate(http_requests_total[5m])",
            "refId": "A"
        }],
        "fieldConfig": {
            "defaults": {
                "unit": "reqps",
                "min": 0
            }
        }
    })

    # Add response time panel
    dashboard["dashboard"]["panels"].append({
        "id": 2,
        "title": "Response Time Distribution",
        "type": "graph",
        "targets": [
            {
                "expr": "histogram_quantile(0.50, rate(http_request_duration_seconds_bucket[5m]))",
                "legendFormat": "p50",
                "refId": "A"
            },
            {
                "expr": "histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))",
                "legendFormat": "p95",
                "refId": "B"
            }
        ]
    })

    return json.dumps(dashboard, indent=2)

# Generate and save dashboard
dashboard_json = create_performance_dashboard()
with open('semantic_text_processor_dashboard.json', 'w') as f:
    f.write(dashboard_json)
```

## Metrics Interpretation and Analysis

### Performance Threshold Analysis

**1. Response Time Analysis**:
```bash
#!/bin/bash
# response_time_analysis.sh

# Get current response time metrics
METRICS=$(curl -s http://localhost:8080/api/v1/metrics)

# Extract key percentiles
P50=$(echo "$METRICS" | jq -r '.histograms.http_request_duration.p50')
P95=$(echo "$METRICS" | jq -r '.histograms.http_request_duration.p95')
P99=$(echo "$METRICS" | jq -r '.histograms.http_request_duration.p99')

echo "Response Time Analysis:"
echo "P50: ${P50}ms (Target: <100ms)"
echo "P95: ${P95}ms (Target: <250ms)"
echo "P99: ${P99}ms (Target: <500ms)"

# Performance classification
if (( $(echo "$P95 > 500" | bc -l) )); then
    echo "STATUS: CRITICAL - Response times exceed acceptable thresholds"
elif (( $(echo "$P95 > 250" | bc -l) )); then
    echo "STATUS: WARNING - Response times approaching limits"
else
    echo "STATUS: HEALTHY - Response times within acceptable range"
fi
```

**2. Search Performance Analysis**:
```bash
#!/bin/bash
# search_performance_analysis.sh

# Analyze search performance by type
SEARCH_METRICS=$(curl -s http://localhost:8080/api/v1/metrics)

# Extract search-specific metrics
SEMANTIC_P95=$(echo "$SEARCH_METRICS" | jq -r '.histograms.search_duration.p95 // 0')
SEARCH_RATE=$(echo "$SEARCH_METRICS" | jq -r '.counters.search_operations_total // 0')

echo "Search Performance Analysis:"
echo "Semantic Search P95: ${SEMANTIC_P95}ms (Target: <500ms)"
echo "Search Operations Rate: ${SEARCH_RATE} operations"

# Performance recommendations
if (( $(echo "$SEMANTIC_P95 > 1000" | bc -l) )); then
    echo "RECOMMENDATION: Consider optimizing vector indexes or increasing cache size"
fi
```

### Trend Analysis

**1. Performance Trend Monitoring**:
```python
# trend_analysis.py
import requests
import json
from datetime import datetime, timedelta
import numpy as np

class PerformanceTrendAnalyzer:
    def __init__(self, metrics_url):
        self.metrics_url = metrics_url
        self.historical_data = []

    def collect_metrics(self):
        """Collect current metrics"""
        try:
            response = requests.get(f"{self.metrics_url}/api/v1/metrics")
            data = response.json()

            timestamp = datetime.now()
            metrics_point = {
                'timestamp': timestamp,
                'response_time_p95': data['histograms']['http_request_duration']['p95'],
                'cache_hit_rate': data['gauges']['cache_hit_rate'],
                'memory_usage': data['gauges']['memory_usage_bytes'],
                'error_rate': data['counters']['http_requests_errors'] / data['counters']['http_requests_total']
            }

            self.historical_data.append(metrics_point)
            return metrics_point

        except Exception as e:
            print(f"Error collecting metrics: {e}")
            return None

    def analyze_trends(self, metric_name, hours=24):
        """Analyze trends for specific metric"""
        cutoff_time = datetime.now() - timedelta(hours=hours)
        recent_data = [
            point[metric_name] for point in self.historical_data
            if point['timestamp'] > cutoff_time
        ]

        if len(recent_data) < 2:
            return "Insufficient data for trend analysis"

        # Calculate trend
        values = np.array(recent_data)
        trend_slope = np.polyfit(range(len(values)), values, 1)[0]

        if trend_slope > 0:
            return f"INCREASING - {metric_name} trending upward"
        elif trend_slope < 0:
            return f"DECREASING - {metric_name} trending downward"
        else:
            return f"STABLE - {metric_name} showing stable pattern"

    def performance_summary(self):
        """Generate performance summary"""
        if not self.historical_data:
            return "No data available"

        latest = self.historical_data[-1]

        summary = {
            'timestamp': latest['timestamp'].isoformat(),
            'response_time_status': 'HEALTHY' if latest['response_time_p95'] < 250 else 'WARNING',
            'cache_performance': 'GOOD' if latest['cache_hit_rate'] > 0.8 else 'POOR',
            'memory_status': 'NORMAL' if latest['memory_usage'] < 2*1024*1024*1024 else 'HIGH',
            'error_rate_status': 'ACCEPTABLE' if latest['error_rate'] < 0.01 else 'HIGH'
        }

        return summary

# Usage example
analyzer = PerformanceTrendAnalyzer("http://localhost:8080")
metrics = analyzer.collect_metrics()
summary = analyzer.performance_summary()
print(json.dumps(summary, indent=2))
```

### Correlation Analysis

**1. Performance Correlation Detection**:
```python
# correlation_analysis.py
import pandas as pd
import numpy as np
from scipy.stats import pearsonr

def analyze_performance_correlations(metrics_data):
    """Analyze correlations between different performance metrics"""

    # Convert to DataFrame
    df = pd.DataFrame(metrics_data)

    # Calculate correlations
    correlations = {}

    # Response time vs cache hit rate
    corr_resp_cache, p_val = pearsonr(df['response_time_p95'], df['cache_hit_rate'])
    correlations['response_time_cache_correlation'] = {
        'correlation': corr_resp_cache,
        'p_value': p_val,
        'interpretation': 'Strong negative correlation indicates cache effectiveness' if corr_resp_cache < -0.7 else 'Weak correlation'
    }

    # Memory usage vs response time
    corr_mem_resp, p_val = pearsonr(df['memory_usage'], df['response_time_p95'])
    correlations['memory_response_correlation'] = {
        'correlation': corr_mem_resp,
        'p_value': p_val,
        'interpretation': 'High memory usage may impact response times' if corr_mem_resp > 0.5 else 'Memory usage not significantly impacting response times'
    }

    # Error rate vs load
    if 'request_rate' in df.columns:
        corr_error_load, p_val = pearsonr(df['error_rate'], df['request_rate'])
        correlations['error_load_correlation'] = {
            'correlation': corr_error_load,
            'p_value': p_val,
            'interpretation': 'Errors increase with load' if corr_error_load > 0.3 else 'Errors not significantly correlated with load'
        }

    return correlations

# Example usage
correlations = analyze_performance_correlations(historical_metrics_data)
for metric, data in correlations.items():
    print(f"{metric}: {data['interpretation']} (r={data['correlation']:.3f})")
```

## Alerting and Thresholds

### Alert Rule Configuration

**1. Prometheus Alert Rules**:
```yaml
# semantic_text_processor_rules.yml
groups:
  - name: semantic_text_processor_alerts
    rules:
      # High response time alert
      - alert: HighResponseTime
        expr: histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m])) > 0.5
        for: 5m
        labels:
          severity: warning
          service: semantic-text-processor
        annotations:
          summary: "High response time detected"
          description: "95th percentile response time is {{ $value }}s for the last 5 minutes"

      # High error rate alert
      - alert: HighErrorRate
        expr: rate(http_requests_total{status_code=~"5.."}[5m]) / rate(http_requests_total[5m]) > 0.05
        for: 2m
        labels:
          severity: critical
          service: semantic-text-processor
        annotations:
          summary: "High error rate detected"
          description: "Error rate is {{ $value | humanizePercentage }} for the last 5 minutes"

      # Low cache hit rate alert
      - alert: LowCacheHitRate
        expr: cache_hit_rate < 0.7
        for: 10m
        labels:
          severity: warning
          service: semantic-text-processor
        annotations:
          summary: "Low cache hit rate"
          description: "Cache hit rate is {{ $value | humanizePercentage }}"

      # High memory usage alert
      - alert: HighMemoryUsage
        expr: process_resident_memory_bytes > 2147483648  # 2GB
        for: 5m
        labels:
          severity: warning
          service: semantic-text-processor
        annotations:
          summary: "High memory usage"
          description: "Memory usage is {{ $value | humanizeBytes }}"

      # Search performance degradation
      - alert: SlowSearchPerformance
        expr: histogram_quantile(0.95, rate(search_duration_seconds_bucket[5m])) > 1.0
        for: 5m
        labels:
          severity: warning
          service: semantic-text-processor
        annotations:
          summary: "Search performance degraded"
          description: "95th percentile search time is {{ $value }}s"

      # Database connection issues
      - alert: DatabaseConnectionIssues
        expr: database_connections_active / database_connections_max > 0.8
        for: 2m
        labels:
          severity: critical
          service: semantic-text-processor
        annotations:
          summary: "Database connection pool nearly exhausted"
          description: "{{ $value | humanizePercentage }} of database connections in use"
```

**2. Dynamic Threshold Calculation**:
```python
# dynamic_thresholds.py
import numpy as np
from datetime import datetime, timedelta

class DynamicThresholdCalculator:
    def __init__(self, historical_data, std_multiplier=2):
        self.historical_data = historical_data
        self.std_multiplier = std_multiplier

    def calculate_threshold(self, metric_name, lookback_hours=168):  # 1 week
        """Calculate dynamic threshold based on historical data"""
        cutoff_time = datetime.now() - timedelta(hours=lookback_hours)

        recent_values = [
            point[metric_name] for point in self.historical_data
            if point['timestamp'] > cutoff_time and metric_name in point
        ]

        if len(recent_values) < 50:  # Minimum data points
            return None

        mean = np.mean(recent_values)
        std = np.std(recent_values)

        # Calculate upper and lower thresholds
        upper_threshold = mean + (self.std_multiplier * std)
        lower_threshold = max(0, mean - (self.std_multiplier * std))

        return {
            'metric': metric_name,
            'mean': mean,
            'std': std,
            'upper_threshold': upper_threshold,
            'lower_threshold': lower_threshold,
            'data_points': len(recent_values)
        }

    def is_anomaly(self, current_value, metric_name):
        """Check if current value is anomalous"""
        threshold_data = self.calculate_threshold(metric_name)

        if not threshold_data:
            return False, "Insufficient data"

        if current_value > threshold_data['upper_threshold']:
            return True, f"Value {current_value} exceeds upper threshold {threshold_data['upper_threshold']}"
        elif current_value < threshold_data['lower_threshold']:
            return True, f"Value {current_value} below lower threshold {threshold_data['lower_threshold']}"

        return False, "Within normal range"

# Usage example
threshold_calc = DynamicThresholdCalculator(historical_performance_data)
current_response_time = 450  # milliseconds

is_anomaly, message = threshold_calc.is_anomaly(current_response_time, 'response_time_p95')
if is_anomaly:
    print(f"ALERT: {message}")
else:
    print(f"Normal: {message}")
```

### Alert Notification Configuration

**1. Alertmanager Configuration**:
```yaml
# alertmanager.yml
global:
  smtp_smarthost: 'localhost:587'
  smtp_from: 'alerts@yourdomain.com'

route:
  group_by: ['alertname']
  group_wait: 10s
  group_interval: 10s
  repeat_interval: 1h
  receiver: 'web.hook'
  routes:
    - match:
        severity: critical
      receiver: 'critical-alerts'
    - match:
        severity: warning
      receiver: 'warning-alerts'

receivers:
  - name: 'web.hook'
    webhook_configs:
      - url: 'http://localhost:5000/webhook'

  - name: 'critical-alerts'
    email_configs:
      - to: 'oncall@yourdomain.com'
        subject: 'CRITICAL: {{ range .Alerts }}{{ .Annotations.summary }}{{ end }}'
        body: |
          {{ range .Alerts }}
          Alert: {{ .Annotations.summary }}
          Description: {{ .Annotations.description }}
          Labels: {{ range .Labels.SortedPairs }}{{ .Name }}={{ .Value }} {{ end }}
          {{ end }}
    slack_configs:
      - api_url: 'YOUR_SLACK_WEBHOOK_URL'
        channel: '#alerts'
        title: 'CRITICAL Alert'
        text: '{{ range .Alerts }}{{ .Annotations.summary }}{{ end }}'

  - name: 'warning-alerts'
    email_configs:
      - to: 'team@yourdomain.com'
        subject: 'Warning: {{ range .Alerts }}{{ .Annotations.summary }}{{ end }}'
```

## Performance Baseline Establishment

### Baseline Collection Process

**1. Automated Baseline Collection**:
```bash
#!/bin/bash
# baseline_collection.sh

echo "Starting performance baseline collection..."

BASELINE_DIR="/var/lib/semantic-text-processor/baselines"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BASELINE_FILE="$BASELINE_DIR/baseline_$TIMESTAMP.json"

mkdir -p "$BASELINE_DIR"

# Collect baseline over 1 hour with 1-minute intervals
for i in {1..60}; do
    echo "Collecting sample $i/60..."

    # Get current metrics
    METRICS=$(curl -s http://localhost:8080/api/v1/metrics)
    HEALTH=$(curl -s http://localhost:8080/api/v1/health)
    CACHE_STATS=$(curl -s http://localhost:8080/api/v1/cache/stats)

    # Combine into single baseline point
    BASELINE_POINT=$(jq -n \
        --argjson metrics "$METRICS" \
        --argjson health "$HEALTH" \
        --argjson cache "$CACHE_STATS" \
        --arg timestamp "$(date -Iseconds)" \
        '{
            timestamp: $timestamp,
            metrics: $metrics,
            health: $health,
            cache: $cache
        }')

    # Append to baseline file
    echo "$BASELINE_POINT" >> "$BASELINE_FILE"

    sleep 60
done

# Calculate baseline statistics
python3 << EOF
import json
import numpy as np

baseline_data = []
with open('$BASELINE_FILE', 'r') as f:
    for line in f:
        baseline_data.append(json.loads(line))

# Calculate statistics for key metrics
response_times = [point['metrics']['histograms']['http_request_duration']['p95'] for point in baseline_data]
cache_hit_rates = [point['cache']['hit_rate'] for point in baseline_data]
memory_usage = [point['metrics']['gauges']['memory_usage_bytes'] for point in baseline_data]

baseline_stats = {
    'collection_period': {
        'start': baseline_data[0]['timestamp'],
        'end': baseline_data[-1]['timestamp'],
        'samples': len(baseline_data)
    },
    'response_time_p95': {
        'mean': np.mean(response_times),
        'std': np.std(response_times),
        'min': np.min(response_times),
        'max': np.max(response_times),
        'p50': np.percentile(response_times, 50),
        'p95': np.percentile(response_times, 95)
    },
    'cache_hit_rate': {
        'mean': np.mean(cache_hit_rates),
        'std': np.std(cache_hit_rates),
        'min': np.min(cache_hit_rates),
        'max': np.max(cache_hit_rates)
    },
    'memory_usage': {
        'mean': np.mean(memory_usage),
        'std': np.std(memory_usage),
        'min': np.min(memory_usage),
        'max': np.max(memory_usage)
    }
}

with open('$BASELINE_DIR/baseline_stats_$TIMESTAMP.json', 'w') as f:
    json.dump(baseline_stats, f, indent=2)

print("Baseline collection completed!")
print(f"Statistics saved to: $BASELINE_DIR/baseline_stats_$TIMESTAMP.json")
EOF
```

## Troubleshooting with Metrics

### Performance Issue Investigation

**1. Response Time Investigation**:
```bash
#!/bin/bash
# response_time_investigation.sh

echo "Investigating response time issues..."

# Get detailed metrics
METRICS=$(curl -s http://localhost:8080/api/v1/metrics)
CURRENT_P95=$(echo "$METRICS" | jq -r '.histograms.http_request_duration.p95')

echo "Current P95 response time: ${CURRENT_P95}ms"

# Check by endpoint
curl -s "http://localhost:8080/api/v1/metrics" | \
    jq -r '.histograms | to_entries[] | select(.key | contains("endpoint")) | "\(.key): \(.value.p95)ms"'

# Identify potential causes
CACHE_HIT_RATE=$(curl -s http://localhost:8080/api/v1/cache/stats | jq -r '.hit_rate')
MEMORY_USAGE=$(echo "$METRICS" | jq -r '.gauges.memory_usage_bytes')
ACTIVE_CONNECTIONS=$(echo "$METRICS" | jq -r '.gauges.active_connections')

echo ""
echo "Potential causes analysis:"

if (( $(echo "$CACHE_HIT_RATE < 0.7" | bc -l) )); then
    echo "❌ Low cache hit rate: $CACHE_HIT_RATE (target: >0.8)"
fi

if (( $(echo "$MEMORY_USAGE > 1073741824" | bc -l) )); then  # 1GB
    echo "❌ High memory usage: $(($MEMORY_USAGE / 1024 / 1024))MB"
fi

if (( $(echo "$ACTIVE_CONNECTIONS > 50" | bc -l) )); then
    echo "❌ High connection count: $ACTIVE_CONNECTIONS"
fi
```

**2. Memory Leak Detection**:
```bash
#!/bin/bash
# memory_leak_detection.sh

echo "Starting memory leak detection..."

# Collect memory usage over time
MEMORY_LOG="/tmp/memory_usage.log"
echo "timestamp,memory_bytes,memory_mb" > "$MEMORY_LOG"

for i in {1..30}; do  # Monitor for 30 minutes
    TIMESTAMP=$(date -Iseconds)
    MEMORY_BYTES=$(curl -s http://localhost:8080/api/v1/metrics | jq -r '.gauges.memory_usage_bytes')
    MEMORY_MB=$((MEMORY_BYTES / 1024 / 1024))

    echo "$TIMESTAMP,$MEMORY_BYTES,$MEMORY_MB" >> "$MEMORY_LOG"
    echo "Sample $i: ${MEMORY_MB}MB"

    sleep 60
done

# Analyze for memory growth trend
python3 << EOF
import pandas as pd
import numpy as np

df = pd.read_csv('$MEMORY_LOG')
df['timestamp'] = pd.to_datetime(df['timestamp'])

# Calculate trend
x = np.arange(len(df))
slope, intercept = np.polyfit(x, df['memory_mb'], 1)

print(f"Memory usage trend: {slope:.2f} MB/minute")

if slope > 5:  # Growing more than 5MB per minute
    print("❌ POTENTIAL MEMORY LEAK DETECTED")
    print("Recommended actions:")
    print("1. Check for goroutine leaks")
    print("2. Review cache size settings")
    print("3. Analyze GC metrics")
elif slope > 1:
    print("⚠️  Memory usage growing - monitor closely")
else:
    print("✅ Memory usage stable")

# Calculate memory growth over observation period
growth = df['memory_mb'].iloc[-1] - df['memory_mb'].iloc[0]
print(f"Total memory growth: {growth:.1f}MB over {len(df)} minutes")
EOF
```

## Automated Performance Analysis

### Performance Report Generation

**1. Automated Performance Report**:
```python
# performance_report.py
import requests
import json
import pandas as pd
from datetime import datetime, timedelta
import matplotlib.pyplot as plt
import numpy as np

class PerformanceReportGenerator:
    def __init__(self, metrics_url, baseline_file=None):
        self.metrics_url = metrics_url
        self.baseline_file = baseline_file
        self.report_data = {}

    def collect_current_metrics(self):
        """Collect current system metrics"""
        try:
            metrics_response = requests.get(f"{self.metrics_url}/api/v1/metrics")
            health_response = requests.get(f"{self.metrics_url}/api/v1/health")
            cache_response = requests.get(f"{self.metrics_url}/api/v1/cache/stats")

            self.report_data['current_metrics'] = {
                'timestamp': datetime.now().isoformat(),
                'metrics': metrics_response.json(),
                'health': health_response.json(),
                'cache': cache_response.json()
            }

            return True
        except Exception as e:
            print(f"Error collecting metrics: {e}")
            return False

    def compare_with_baseline(self):
        """Compare current metrics with baseline"""
        if not self.baseline_file:
            return {}

        try:
            with open(self.baseline_file, 'r') as f:
                baseline_stats = json.load(f)

            current = self.report_data['current_metrics']['metrics']

            comparison = {
                'response_time_p95': {
                    'current': current['histograms']['http_request_duration']['p95'],
                    'baseline_mean': baseline_stats['response_time_p95']['mean'],
                    'variance_percent': ((current['histograms']['http_request_duration']['p95'] -
                                        baseline_stats['response_time_p95']['mean']) /
                                       baseline_stats['response_time_p95']['mean'] * 100)
                },
                'cache_hit_rate': {
                    'current': self.report_data['current_metrics']['cache']['hit_rate'],
                    'baseline_mean': baseline_stats['cache_hit_rate']['mean'],
                    'variance_percent': ((self.report_data['current_metrics']['cache']['hit_rate'] -
                                        baseline_stats['cache_hit_rate']['mean']) /
                                       baseline_stats['cache_hit_rate']['mean'] * 100)
                }
            }

            return comparison
        except Exception as e:
            print(f"Error comparing with baseline: {e}")
            return {}

    def generate_recommendations(self):
        """Generate performance optimization recommendations"""
        recommendations = []

        current = self.report_data['current_metrics']

        # Response time recommendations
        p95_response_time = current['metrics']['histograms']['http_request_duration']['p95']
        if p95_response_time > 500:
            recommendations.append({
                'type': 'critical',
                'area': 'response_time',
                'issue': f'P95 response time is {p95_response_time}ms (target: <250ms)',
                'recommendations': [
                    'Check database query performance',
                    'Increase cache size or TTL',
                    'Review and optimize slow endpoints',
                    'Consider horizontal scaling'
                ]
            })
        elif p95_response_time > 250:
            recommendations.append({
                'type': 'warning',
                'area': 'response_time',
                'issue': f'P95 response time is {p95_response_time}ms approaching limits',
                'recommendations': [
                    'Monitor trend closely',
                    'Review cache hit rates',
                    'Check for resource constraints'
                ]
            })

        # Cache recommendations
        cache_hit_rate = current['cache']['hit_rate']
        if cache_hit_rate < 0.7:
            recommendations.append({
                'type': 'warning',
                'area': 'cache',
                'issue': f'Cache hit rate is {cache_hit_rate:.1%} (target: >80%)',
                'recommendations': [
                    'Increase cache size',
                    'Optimize cache TTL settings',
                    'Review cache key strategy',
                    'Implement cache warming for popular queries'
                ]
            })

        # Memory recommendations
        memory_usage = current['metrics']['gauges']['memory_usage_bytes']
        memory_mb = memory_usage / 1024 / 1024
        if memory_mb > 2048:  # 2GB
            recommendations.append({
                'type': 'warning',
                'area': 'memory',
                'issue': f'Memory usage is {memory_mb:.0f}MB (high)',
                'recommendations': [
                    'Check for memory leaks',
                    'Reduce cache size if necessary',
                    'Optimize data structures',
                    'Consider garbage collection tuning'
                ]
            })

        return recommendations

    def generate_html_report(self, output_file='performance_report.html'):
        """Generate HTML performance report"""
        html_template = """
        <!DOCTYPE html>
        <html>
        <head>
            <title>Performance Report - {timestamp}</title>
            <style>
                body {{ font-family: Arial, sans-serif; margin: 40px; }}
                .header {{ background-color: #f4f4f4; padding: 20px; border-radius: 5px; }}
                .metric-card {{ background-color: #fff; border: 1px solid #ddd; margin: 10px 0; padding: 15px; border-radius: 5px; }}
                .critical {{ border-left: 5px solid #d32f2f; }}
                .warning {{ border-left: 5px solid #f57c00; }}
                .good {{ border-left: 5px solid #388e3c; }}
                .recommendations {{ background-color: #e3f2fd; padding: 15px; border-radius: 5px; margin-top: 10px; }}
                table {{ border-collapse: collapse; width: 100%; }}
                th, td {{ border: 1px solid #ddd; padding: 8px; text-align: left; }}
                th {{ background-color: #f2f2f2; }}
            </style>
        </head>
        <body>
            <div class="header">
                <h1>Semantic Text Processor Performance Report</h1>
                <p>Generated: {timestamp}</p>
            </div>

            <h2>Current Performance Metrics</h2>
            {metrics_html}

            <h2>Baseline Comparison</h2>
            {comparison_html}

            <h2>Performance Recommendations</h2>
            {recommendations_html}
        </body>
        </html>
        """

        # Generate metrics HTML
        current = self.report_data['current_metrics']
        metrics_html = f"""
        <div class="metric-card">
            <h3>Response Time</h3>
            <table>
                <tr><th>Metric</th><th>Value</th><th>Status</th></tr>
                <tr><td>P50</td><td>{current['metrics']['histograms']['http_request_duration']['p50']:.1f}ms</td><td>{'✅' if current['metrics']['histograms']['http_request_duration']['p50'] < 100 else '⚠️'}</td></tr>
                <tr><td>P95</td><td>{current['metrics']['histograms']['http_request_duration']['p95']:.1f}ms</td><td>{'✅' if current['metrics']['histograms']['http_request_duration']['p95'] < 250 else '⚠️'}</td></tr>
                <tr><td>P99</td><td>{current['metrics']['histograms']['http_request_duration']['p99']:.1f}ms</td><td>{'✅' if current['metrics']['histograms']['http_request_duration']['p99'] < 500 else '⚠️'}</td></tr>
            </table>
        </div>

        <div class="metric-card">
            <h3>Cache Performance</h3>
            <table>
                <tr><th>Metric</th><th>Value</th><th>Status</th></tr>
                <tr><td>Hit Rate</td><td>{current['cache']['hit_rate']:.1%}</td><td>{'✅' if current['cache']['hit_rate'] > 0.8 else '⚠️'}</td></tr>
                <tr><td>Size</td><td>{current['cache']['size']}</td><td>-</td></tr>
                <tr><td>Memory Usage</td><td>{current['cache'].get('memory_usage_bytes', 0) / 1024 / 1024:.1f}MB</td><td>-</td></tr>
            </table>
        </div>
        """

        # Generate recommendations HTML
        recommendations = self.generate_recommendations()
        recommendations_html = ""
        for rec in recommendations:
            css_class = rec['type']
            recommendations_html += f"""
            <div class="metric-card {css_class}">
                <h3>{rec['area'].title()} - {rec['type'].title()}</h3>
                <p><strong>Issue:</strong> {rec['issue']}</p>
                <div class="recommendations">
                    <strong>Recommendations:</strong>
                    <ul>
                        {''.join([f'<li>{r}</li>' for r in rec['recommendations']])}
                    </ul>
                </div>
            </div>
            """

        # Fill template
        html_content = html_template.format(
            timestamp=current['timestamp'],
            metrics_html=metrics_html,
            comparison_html="Baseline comparison not available" if not self.baseline_file else "Comparison data available",
            recommendations_html=recommendations_html if recommendations_html else "<p>No specific recommendations at this time. System performance is within acceptable ranges.</p>"
        )

        # Write to file
        with open(output_file, 'w') as f:
            f.write(html_content)

        print(f"Performance report generated: {output_file}")
        return output_file

# Usage example
if __name__ == "__main__":
    report_generator = PerformanceReportGenerator(
        "http://localhost:8080",
        "/var/lib/semantic-text-processor/baselines/baseline_stats_latest.json"
    )

    if report_generator.collect_current_metrics():
        report_file = report_generator.generate_html_report()
        print(f"Performance report available at: {report_file}")
    else:
        print("Failed to collect metrics for report generation")
```

This comprehensive performance monitoring guide provides all the tools and procedures needed to effectively monitor, analyze, and optimize the Semantic Text Processor's performance. The monitoring approach is designed to be proactive, helping identify and resolve performance issues before they impact users.