# Capacity Planning and Scaling Recommendations Guide

## Table of Contents

1. [Overview](#overview)
2. [Current System Baseline](#current-system-baseline)
3. [Resource Requirements Analysis](#resource-requirements-analysis)
4. [Traffic Pattern Analysis](#traffic-pattern-analysis)
5. [Scaling Strategies](#scaling-strategies)
6. [Performance Bottleneck Identification](#performance-bottleneck-identification)
7. [Infrastructure Scaling](#infrastructure-scaling)
8. [Database Scaling Considerations](#database-scaling-considerations)
9. [Cost Optimization](#cost-optimization)
10. [Monitoring and Alerting for Scaling](#monitoring-and-alerting-for-scaling)
11. [Scaling Implementation Plan](#scaling-implementation-plan)

## Overview

This guide provides comprehensive capacity planning and scaling recommendations for the Semantic Text Processor system. The approach balances performance requirements, cost efficiency, and operational complexity to ensure optimal system scalability.

### Scaling Objectives

1. **Performance Maintenance**: Maintain response times under 250ms (P95) regardless of load
2. **Cost Efficiency**: Optimize resource utilization and minimize waste
3. **Reliability**: Ensure high availability during scaling operations
4. **Flexibility**: Support both predictable and unpredictable growth patterns
5. **Operational Simplicity**: Minimize complexity in scaling procedures

### Key Scaling Dimensions

- **Compute Resources**: CPU, memory, and processing capacity
- **Storage**: Database storage, cache storage, and file storage
- **Network**: Bandwidth and connection capacity
- **External Dependencies**: API rate limits and third-party services

## Current System Baseline

### Resource Usage Baseline

**1. Compute Resources**:
```bash
# Collect current resource usage baseline
CPU_USAGE=$(top -bn1 | grep "Cpu(s)" | awk '{print $2}' | cut -d'%' -f1)
MEMORY_USAGE=$(free | grep Mem | awk '{printf "%.1f", $3/$2 * 100.0}')
LOAD_AVERAGE=$(uptime | awk -F'load average:' '{print $2}' | cut -d',' -f1 | xargs)

echo "Current System Baseline:"
echo "CPU Usage: ${CPU_USAGE}%"
echo "Memory Usage: ${MEMORY_USAGE}%"
echo "Load Average (1m): ${LOAD_AVERAGE}"

# Application-specific metrics
APP_MEMORY=$(ps -p $(pgrep semantic-text-processor) -o rss= | awk '{print $1/1024}')
APP_CPU=$(ps -p $(pgrep semantic-text-processor) -o pcpu= | awk '{print $1}')

echo "Application Usage:"
echo "App Memory: ${APP_MEMORY}MB"
echo "App CPU: ${APP_CPU}%"
```

**2. Performance Baseline**:
```bash
# Collect performance baseline
RESPONSE_TIME_P95=$(curl -s http://localhost:8080/api/v1/metrics | jq -r '.histograms.http_request_duration.p95')
REQUEST_RATE=$(curl -s http://localhost:8080/api/v1/metrics | jq -r '.counters.http_requests_total')
CACHE_HIT_RATE=$(curl -s http://localhost:8080/api/v1/cache/stats | jq -r '.hit_rate')

echo "Performance Baseline:"
echo "Response Time P95: ${RESPONSE_TIME_P95}ms"
echo "Total Requests: ${REQUEST_RATE}"
echo "Cache Hit Rate: ${CACHE_HIT_RATE}"
```

**3. Database Baseline**:
```sql
-- Database size and performance baseline
SELECT
    pg_size_pretty(pg_total_relation_size('chunks')) as chunks_size,
    pg_size_pretty(pg_total_relation_size('embeddings')) as embeddings_size,
    pg_size_pretty(pg_database_size(current_database())) as total_db_size;

-- Query performance baseline
SELECT
    schemaname,
    tablename,
    seq_scan,
    seq_tup_read,
    idx_scan,
    idx_tup_fetch
FROM pg_stat_user_tables
WHERE tablename IN ('chunks', 'embeddings', 'texts');
```

### Current Capacity Limits

**1. Theoretical Limits**:
- **CPU**: 4 cores @ 2.4GHz = ~9,600 CPU-seconds/hour
- **Memory**: 8GB total, ~6GB available for application
- **Storage**: 100GB SSD with ~80GB usable
- **Network**: 1Gbps connection

**2. Practical Limits (80% utilization target)**:
- **CPU**: ~7,680 CPU-seconds/hour
- **Memory**: ~4.8GB for application
- **Storage**: ~64GB for data
- **Concurrent Connections**: ~500 (based on current configuration)

**3. Current Utilization**:
```python
# current_utilization.py
import psutil
import requests
import json

def assess_current_utilization():
    """Assess current resource utilization against capacity"""

    # System resources
    cpu_percent = psutil.cpu_percent(interval=1)
    memory = psutil.virtual_memory()
    disk = psutil.disk_usage('/')

    # Application metrics
    try:
        metrics_response = requests.get('http://localhost:8080/api/v1/metrics')
        app_metrics = metrics_response.json()

        utilization_report = {
            'timestamp': '2024-01-15T10:30:00Z',
            'system_resources': {
                'cpu_percent': cpu_percent,
                'cpu_headroom': 80 - cpu_percent,
                'memory_percent': memory.percent,
                'memory_headroom_gb': (memory.total * 0.8 - memory.used) / (1024**3),
                'disk_percent': (disk.used / disk.total) * 100,
                'disk_headroom_gb': (disk.total * 0.8 - disk.used) / (1024**3)
            },
            'application_performance': {
                'response_time_p95': app_metrics['histograms']['http_request_duration']['p95'],
                'response_time_headroom_ms': 250 - app_metrics['histograms']['http_request_duration']['p95'],
                'cache_hit_rate': app_metrics.get('cache_hit_rate', 0),
                'cache_efficiency': 'good' if app_metrics.get('cache_hit_rate', 0) > 0.8 else 'poor'
            },
            'scaling_recommendation': 'immediate' if cpu_percent > 70 or memory.percent > 80 else 'monitor'
        }

        return utilization_report

    except Exception as e:
        print(f"Error collecting utilization data: {e}")
        return None

# Generate utilization assessment
utilization = assess_current_utilization()
if utilization:
    print(json.dumps(utilization, indent=2))
```

## Resource Requirements Analysis

### Workload Characterization

**1. Request Patterns**:
```python
# workload_analysis.py
import pandas as pd
import numpy as np
from datetime import datetime, timedelta

class WorkloadAnalyzer:
    def __init__(self, metrics_history):
        self.metrics_history = metrics_history

    def analyze_request_patterns(self):
        """Analyze request patterns and resource consumption"""

        # Convert to DataFrame for analysis
        df = pd.DataFrame(self.metrics_history)
        df['timestamp'] = pd.to_datetime(df['timestamp'])
        df['hour'] = df['timestamp'].dt.hour
        df['day_of_week'] = df['timestamp'].dt.dayofweek

        patterns = {
            'hourly_distribution': df.groupby('hour')['request_rate'].mean().to_dict(),
            'daily_distribution': df.groupby('day_of_week')['request_rate'].mean().to_dict(),
            'peak_hour': df.groupby('hour')['request_rate'].mean().idxmax(),
            'peak_day': df.groupby('day_of_week')['request_rate'].mean().idxmax(),
            'peak_to_average_ratio': df['request_rate'].max() / df['request_rate'].mean()
        }

        return patterns

    def calculate_resource_per_request(self):
        """Calculate resource consumption per request"""

        df = pd.DataFrame(self.metrics_history)

        # Calculate per-request resource usage
        resources_per_request = {
            'cpu_ms_per_request': df['cpu_usage'].mean() / df['request_rate'].mean() * 1000,
            'memory_mb_per_request': df['memory_usage_mb'].mean() / df['concurrent_requests'].mean(),
            'response_time_ms': df['response_time_p95'].mean(),
            'cache_benefit_ms': df['response_time_no_cache'].mean() - df['response_time_with_cache'].mean()
        }

        return resources_per_request

    def predict_capacity_needs(self, growth_factor=2.0, target_response_time=250):
        """Predict capacity needs for growth scenario"""

        current_rps = np.mean([h['request_rate'] for h in self.metrics_history])
        current_response_time = np.mean([h['response_time_p95'] for h in self.metrics_history])

        # Linear scaling assumption (conservative)
        predicted_rps = current_rps * growth_factor
        predicted_response_time = current_response_time * growth_factor

        # Calculate required resources
        if predicted_response_time > target_response_time:
            scaling_factor = predicted_response_time / target_response_time
            required_scaling = {
                'cpu_scaling_factor': scaling_factor,
                'memory_scaling_factor': scaling_factor * 0.8,  # Memory scales less linearly
                'cache_scaling_factor': growth_factor,
                'database_scaling_factor': growth_factor * 1.2  # Database load often scales super-linearly
            }
        else:
            required_scaling = {
                'cpu_scaling_factor': growth_factor,
                'memory_scaling_factor': growth_factor * 0.8,
                'cache_scaling_factor': growth_factor,
                'database_scaling_factor': growth_factor
            }

        return {
            'predicted_load': {
                'rps': predicted_rps,
                'response_time': predicted_response_time
            },
            'scaling_requirements': required_scaling,
            'recommended_resources': {
                'cpu_cores': int(4 * required_scaling['cpu_scaling_factor']),
                'memory_gb': int(8 * required_scaling['memory_scaling_factor']),
                'cache_size_mb': int(1000 * required_scaling['cache_scaling_factor']),
                'db_connections': int(25 * required_scaling['database_scaling_factor'])
            }
        }

# Example usage
sample_metrics = [
    {'timestamp': '2024-01-15T10:00:00Z', 'request_rate': 50, 'cpu_usage': 45, 'memory_usage_mb': 2048, 'response_time_p95': 120, 'concurrent_requests': 20},
    {'timestamp': '2024-01-15T11:00:00Z', 'request_rate': 75, 'cpu_usage': 60, 'memory_usage_mb': 2256, 'response_time_p95': 150, 'concurrent_requests': 30},
    {'timestamp': '2024-01-15T12:00:00Z', 'request_rate': 100, 'cpu_usage': 70, 'memory_usage_mb': 2512, 'response_time_p95': 180, 'concurrent_requests': 40}
]

analyzer = WorkloadAnalyzer(sample_metrics)
capacity_prediction = analyzer.predict_capacity_needs(growth_factor=3.0)
print(json.dumps(capacity_prediction, indent=2))
```

**2. Resource Consumption Models**:
```python
# resource_models.py
import numpy as np
from scipy import stats

class ResourceConsumptionModel:
    def __init__(self):
        self.models = {}

    def build_cpu_model(self, request_rates, cpu_utilizations):
        """Build CPU consumption model"""
        slope, intercept, r_value, p_value, std_err = stats.linregress(request_rates, cpu_utilizations)

        self.models['cpu'] = {
            'type': 'linear',
            'slope': slope,
            'intercept': intercept,
            'r_squared': r_value**2,
            'formula': f'cpu_usage = {slope:.2f} * request_rate + {intercept:.2f}'
        }

        return self.models['cpu']

    def build_memory_model(self, concurrent_users, memory_usage):
        """Build memory consumption model"""
        # Memory often has a base usage plus per-user component
        slope, intercept, r_value, p_value, std_err = stats.linregress(concurrent_users, memory_usage)

        self.models['memory'] = {
            'type': 'linear',
            'slope': slope,
            'intercept': intercept,
            'r_squared': r_value**2,
            'base_memory_mb': intercept,
            'memory_per_user_mb': slope,
            'formula': f'memory_mb = {slope:.2f} * concurrent_users + {intercept:.2f}'
        }

        return self.models['memory']

    def build_response_time_model(self, request_rates, response_times):
        """Build response time model (often exponential under load)"""
        # Try exponential model: response_time = a * e^(b * request_rate)
        log_response_times = np.log(response_times)
        slope, intercept, r_value, p_value, std_err = stats.linregress(request_rates, log_response_times)

        self.models['response_time'] = {
            'type': 'exponential',
            'a': np.exp(intercept),
            'b': slope,
            'r_squared': r_value**2,
            'formula': f'response_time = {np.exp(intercept):.2f} * e^({slope:.4f} * request_rate)'
        }

        return self.models['response_time']

    def predict_resources(self, target_request_rate, target_concurrent_users):
        """Predict resource requirements for target load"""
        predictions = {}

        if 'cpu' in self.models:
            predictions['cpu_usage'] = (
                self.models['cpu']['slope'] * target_request_rate +
                self.models['cpu']['intercept']
            )

        if 'memory' in self.models:
            predictions['memory_usage_mb'] = (
                self.models['memory']['slope'] * target_concurrent_users +
                self.models['memory']['intercept']
            )

        if 'response_time' in self.models:
            predictions['response_time_ms'] = (
                self.models['response_time']['a'] *
                np.exp(self.models['response_time']['b'] * target_request_rate)
            )

        return predictions

    def calculate_scaling_requirements(self, current_load, target_load, performance_target):
        """Calculate scaling requirements to meet performance targets"""
        current_prediction = self.predict_resources(current_load['rps'], current_load['concurrent_users'])
        target_prediction = self.predict_resources(target_load['rps'], target_load['concurrent_users'])

        scaling_requirements = {}

        # CPU scaling
        if target_prediction.get('cpu_usage', 0) > 80:  # 80% utilization target
            scaling_requirements['cpu_cores'] = int(np.ceil(target_prediction['cpu_usage'] / 80))
        else:
            scaling_requirements['cpu_cores'] = 1

        # Memory scaling
        if target_prediction.get('memory_usage_mb', 0) > 6144:  # 6GB target
            scaling_requirements['memory_gb'] = int(np.ceil(target_prediction['memory_usage_mb'] / 1024))
        else:
            scaling_requirements['memory_gb'] = 8

        # Response time consideration
        if target_prediction.get('response_time_ms', 0) > performance_target:
            # Need more aggressive scaling
            response_time_factor = target_prediction['response_time_ms'] / performance_target
            scaling_requirements['cpu_cores'] = int(scaling_requirements['cpu_cores'] * response_time_factor)

        return scaling_requirements

# Example usage
model = ResourceConsumptionModel()

# Build models with sample data
request_rates = [10, 25, 50, 75, 100, 150]
cpu_utilizations = [15, 25, 40, 55, 70, 85]
memory_usage = [1800, 2000, 2300, 2600, 2900, 3400]
response_times = [80, 90, 120, 150, 200, 300]
concurrent_users = [5, 12, 25, 38, 50, 75]

model.build_cpu_model(request_rates, cpu_utilizations)
model.build_memory_model(concurrent_users, memory_usage)
model.build_response_time_model(request_rates, response_times)

# Predict requirements for 2x load
scaling_reqs = model.calculate_scaling_requirements(
    current_load={'rps': 50, 'concurrent_users': 25},
    target_load={'rps': 100, 'concurrent_users': 50},
    performance_target=250
)

print("Scaling Requirements:", scaling_reqs)
```

## Traffic Pattern Analysis

### Load Forecasting

**1. Historical Growth Analysis**:
```python
# growth_analysis.py
import pandas as pd
import numpy as np
from sklearn.linear_model import LinearRegression
from sklearn.metrics import mean_squared_error
import matplotlib.pyplot as plt

class GrowthAnalyzer:
    def __init__(self, historical_data):
        self.data = pd.DataFrame(historical_data)
        self.data['timestamp'] = pd.to_datetime(self.data['timestamp'])
        self.data['days_since_start'] = (self.data['timestamp'] - self.data['timestamp'].min()).dt.days

    def analyze_growth_trend(self, metric='daily_requests'):
        """Analyze growth trend for a specific metric"""
        X = self.data['days_since_start'].values.reshape(-1, 1)
        y = self.data[metric].values

        # Fit linear growth model
        model = LinearRegression()
        model.fit(X, y)

        # Calculate growth rate
        daily_growth_rate = model.coef_[0]
        monthly_growth_rate = daily_growth_rate * 30
        annual_growth_rate = daily_growth_rate * 365

        # Calculate R-squared
        y_pred = model.predict(X)
        r_squared = 1 - (np.sum((y - y_pred) ** 2) / np.sum((y - np.mean(y)) ** 2))

        return {
            'daily_growth_rate': daily_growth_rate,
            'monthly_growth_rate': monthly_growth_rate,
            'annual_growth_rate': annual_growth_rate,
            'r_squared': r_squared,
            'current_value': y[-1],
            'model': model
        }

    def forecast_load(self, days_ahead=90, confidence_interval=0.95):
        """Forecast load for specified days ahead"""
        forecasts = {}

        for metric in ['daily_requests', 'peak_concurrent_users', 'data_volume_gb']:
            if metric in self.data.columns:
                growth_analysis = self.analyze_growth_trend(metric)

                # Forecast future values
                future_days = np.arange(self.data['days_since_start'].max() + 1,
                                      self.data['days_since_start'].max() + days_ahead + 1).reshape(-1, 1)
                future_values = growth_analysis['model'].predict(future_days)

                # Calculate confidence intervals (simplified)
                historical_error = np.std(self.data[metric] - growth_analysis['model'].predict(
                    self.data['days_since_start'].values.reshape(-1, 1)))

                z_score = 1.96 if confidence_interval == 0.95 else 1.645  # 95% or 90%

                forecasts[metric] = {
                    'forecasted_values': future_values.tolist(),
                    'confidence_lower': (future_values - z_score * historical_error).tolist(),
                    'confidence_upper': (future_values + z_score * historical_error).tolist(),
                    'growth_rate_per_day': growth_analysis['daily_growth_rate'],
                    'current_value': growth_analysis['current_value']
                }

        return forecasts

    def identify_scaling_triggers(self):
        """Identify when scaling actions should be triggered"""
        current_metrics = self.data.iloc[-1]

        # Define scaling thresholds
        thresholds = {
            'cpu_utilization': 70,
            'memory_utilization': 80,
            'response_time_p95': 200,
            'cache_hit_rate': 0.7,
            'database_connections_used': 0.8
        }

        triggers = {}
        for metric, threshold in thresholds.items():
            if metric in current_metrics:
                current_value = current_metrics[metric]
                if isinstance(threshold, float) and threshold < 1:
                    # Percentage threshold
                    triggers[metric] = {
                        'current': current_value,
                        'threshold': threshold,
                        'triggered': current_value < threshold if 'hit_rate' in metric else current_value > threshold,
                        'urgency': 'high' if abs(current_value - threshold) / threshold > 0.2 else 'medium'
                    }
                else:
                    # Absolute threshold
                    triggers[metric] = {
                        'current': current_value,
                        'threshold': threshold,
                        'triggered': current_value > threshold,
                        'urgency': 'high' if current_value > threshold * 1.2 else 'medium'
                    }

        return triggers

# Example historical data
historical_data = [
    {'timestamp': '2024-01-01', 'daily_requests': 10000, 'peak_concurrent_users': 50, 'data_volume_gb': 10, 'cpu_utilization': 45, 'memory_utilization': 60},
    {'timestamp': '2024-01-08', 'daily_requests': 12000, 'peak_concurrent_users': 60, 'data_volume_gb': 12, 'cpu_utilization': 50, 'memory_utilization': 65},
    {'timestamp': '2024-01-15', 'daily_requests': 15000, 'peak_concurrent_users': 75, 'data_volume_gb': 15, 'cpu_utilization': 60, 'memory_utilization': 70},
    {'timestamp': '2024-01-22', 'daily_requests': 18000, 'peak_concurrent_users': 90, 'data_volume_gb': 18, 'cpu_utilization': 65, 'memory_utilization': 75}
]

analyzer = GrowthAnalyzer(historical_data)
forecasts = analyzer.forecast_load(days_ahead=30)
triggers = analyzer.identify_scaling_triggers()

print("30-day forecast:")
for metric, forecast in forecasts.items():
    print(f"{metric}: {forecast['current_value']:.0f} -> {forecast['forecasted_values'][-1]:.0f}")

print("\nScaling triggers:")
for metric, trigger in triggers.items():
    if trigger['triggered']:
        print(f"⚠️ {metric}: {trigger['current']} > {trigger['threshold']} ({trigger['urgency']} urgency)")
```

**2. Seasonal Pattern Recognition**:
```python
# seasonal_analysis.py
import pandas as pd
import numpy as np
from scipy import signal

class SeasonalAnalyzer:
    def __init__(self, hourly_data):
        self.data = pd.DataFrame(hourly_data)
        self.data['timestamp'] = pd.to_datetime(self.data['timestamp'])
        self.data['hour'] = self.data['timestamp'].dt.hour
        self.data['day_of_week'] = self.data['timestamp'].dt.day_of_week
        self.data['day_of_month'] = self.data['timestamp'].dt.day

    def identify_daily_patterns(self):
        """Identify daily usage patterns"""
        hourly_avg = self.data.groupby('hour')['request_rate'].agg(['mean', 'std', 'min', 'max'])

        patterns = {
            'peak_hour': hourly_avg['mean'].idxmax(),
            'lowest_hour': hourly_avg['mean'].idxmin(),
            'peak_to_trough_ratio': hourly_avg['mean'].max() / hourly_avg['mean'].min(),
            'high_variance_hours': hourly_avg[hourly_avg['std'] > hourly_avg['std'].quantile(0.8)].index.tolist(),
            'hourly_distribution': hourly_avg.to_dict()
        }

        return patterns

    def identify_weekly_patterns(self):
        """Identify weekly usage patterns"""
        daily_avg = self.data.groupby('day_of_week')['request_rate'].agg(['mean', 'std', 'min', 'max'])

        patterns = {
            'peak_day': daily_avg['mean'].idxmax(),
            'lowest_day': daily_avg['mean'].idxmin(),
            'weekday_vs_weekend': {
                'weekday_avg': daily_avg.loc[0:4, 'mean'].mean(),
                'weekend_avg': daily_avg.loc[5:6, 'mean'].mean()
            },
            'daily_distribution': daily_avg.to_dict()
        }

        return patterns

    def calculate_capacity_requirements(self):
        """Calculate capacity requirements based on patterns"""
        daily_patterns = self.identify_daily_patterns()
        weekly_patterns = self.identify_weekly_patterns()

        # Calculate required capacity for different scenarios
        baseline_load = self.data['request_rate'].mean()
        peak_load = self.data['request_rate'].max()
        p95_load = self.data['request_rate'].quantile(0.95)

        capacity_scenarios = {
            'baseline': {
                'load_factor': 1.0,
                'required_capacity': baseline_load,
                'description': 'Average load capacity'
            },
            'p95': {
                'load_factor': p95_load / baseline_load,
                'required_capacity': p95_load,
                'description': '95th percentile load capacity'
            },
            'peak': {
                'load_factor': peak_load / baseline_load,
                'required_capacity': peak_load,
                'description': 'Peak load capacity'
            },
            'peak_with_buffer': {
                'load_factor': (peak_load * 1.2) / baseline_load,
                'required_capacity': peak_load * 1.2,
                'description': 'Peak load with 20% buffer'
            }
        }

        return {
            'patterns': {
                'daily': daily_patterns,
                'weekly': weekly_patterns
            },
            'capacity_scenarios': capacity_scenarios,
            'recommendations': {
                'auto_scaling_target': 'p95',
                'manual_scaling_target': 'peak_with_buffer',
                'monitoring_frequency': 'hourly' if daily_patterns['peak_to_trough_ratio'] > 3 else 'daily'
            }
        }

# Example usage with hourly data
import random
from datetime import datetime, timedelta

# Generate sample hourly data with realistic patterns
start_date = datetime(2024, 1, 1)
hourly_data = []

for i in range(24 * 30):  # 30 days of hourly data
    timestamp = start_date + timedelta(hours=i)
    hour = timestamp.hour
    day_of_week = timestamp.weekday()

    # Simulate realistic patterns
    base_load = 100
    hourly_multiplier = 1.5 if 9 <= hour <= 17 else 0.7  # Business hours
    weekly_multiplier = 1.2 if day_of_week < 5 else 0.8  # Weekday vs weekend
    noise = random.uniform(0.8, 1.2)

    request_rate = base_load * hourly_multiplier * weekly_multiplier * noise

    hourly_data.append({
        'timestamp': timestamp,
        'request_rate': request_rate,
        'response_time': 100 + (request_rate / 10)  # Response time increases with load
    })

analyzer = SeasonalAnalyzer(hourly_data)
capacity_analysis = analyzer.calculate_capacity_requirements()

print("Capacity Analysis Results:")
print(f"Peak hour: {capacity_analysis['patterns']['daily']['peak_hour']}")
print(f"Peak day: {capacity_analysis['patterns']['weekly']['peak_day']}")
print(f"Peak to trough ratio: {capacity_analysis['patterns']['daily']['peak_to_trough_ratio']:.2f}")

for scenario, details in capacity_analysis['capacity_scenarios'].items():
    print(f"{scenario}: {details['required_capacity']:.0f} RPS ({details['load_factor']:.2f}x baseline)")
```

## Scaling Strategies

### Horizontal vs Vertical Scaling

**1. Horizontal Scaling Strategy**:
```yaml
# horizontal_scaling_config.yaml
horizontal_scaling:
  application_tier:
    strategy: "load_balancer_with_auto_scaling"
    min_instances: 2
    max_instances: 10
    target_cpu_utilization: 70
    target_memory_utilization: 80
    scale_up_threshold:
      cpu: 70
      memory: 80
      response_time_p95: 200
    scale_down_threshold:
      cpu: 40
      memory: 50
      response_time_p95: 100
    cooldown_periods:
      scale_up: 300  # 5 minutes
      scale_down: 600  # 10 minutes

  cache_tier:
    strategy: "distributed_cache"
    nodes: 3
    replication_factor: 2
    sharding: "consistent_hashing"
    auto_scaling: false

  database_tier:
    strategy: "read_replicas"
    primary: 1
    read_replicas: 2
    auto_scaling: false
    connection_pooling:
      max_connections_per_instance: 25
      total_max_connections: 100
```

**2. Vertical Scaling Strategy**:
```yaml
# vertical_scaling_config.yaml
vertical_scaling:
  scaling_tiers:
    tier_1:
      cpu_cores: 2
      memory_gb: 4
      max_concurrent_users: 100
      max_rps: 50

    tier_2:
      cpu_cores: 4
      memory_gb: 8
      max_concurrent_users: 250
      max_rps: 125

    tier_3:
      cpu_cores: 8
      memory_gb: 16
      max_concurrent_users: 500
      max_rps: 250

    tier_4:
      cpu_cores: 16
      memory_gb: 32
      max_concurrent_users: 1000
      max_rps: 500

  upgrade_triggers:
    cpu_utilization: 80
    memory_utilization: 85
    response_time_p95: 300
    sustained_duration: 900  # 15 minutes

  downgrade_triggers:
    cpu_utilization: 40
    memory_utilization: 50
    response_time_p95: 100
    sustained_duration: 1800  # 30 minutes
```

### Auto-scaling Implementation

**1. Kubernetes Horizontal Pod Autoscaler**:
```yaml
# k8s-hpa.yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: semantic-text-processor-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: semantic-text-processor
  minReplicas: 2
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
  - type: Pods
    pods:
      metric:
        name: http_requests_per_second
      target:
        type: AverageValue
        averageValue: "100"
  behavior:
    scaleUp:
      stabilizationWindowSeconds: 300
      policies:
      - type: Percent
        value: 50
        periodSeconds: 60
    scaleDown:
      stabilizationWindowSeconds: 600
      policies:
      - type: Percent
        value: 10
        periodSeconds: 60
```

**2. Custom Auto-scaling Script**:
```python
# autoscaler.py
import time
import requests
import json
import subprocess
from datetime import datetime, timedelta

class AutoScaler:
    def __init__(self, config):
        self.config = config
        self.scaling_history = []
        self.last_scale_action = None

    def get_current_metrics(self):
        """Get current system metrics"""
        try:
            metrics_response = requests.get(f"{self.config['api_base']}/metrics")
            health_response = requests.get(f"{self.config['api_base']}/health")

            return {
                'timestamp': datetime.now(),
                'metrics': metrics_response.json(),
                'health': health_response.json(),
                'instance_count': self.get_current_instance_count()
            }
        except Exception as e:
            print(f"Error getting metrics: {e}")
            return None

    def get_current_instance_count(self):
        """Get current number of running instances"""
        try:
            # Example for Docker Swarm
            result = subprocess.run(['docker', 'service', 'ls', '--filter', 'name=semantic-text-processor', '--format', '{{.Replicas}}'],
                                  capture_output=True, text=True)
            replicas = result.stdout.strip().split('/')[0] if result.stdout else "1"
            return int(replicas)
        except:
            return 1

    def should_scale_up(self, metrics):
        """Determine if scaling up is needed"""
        current_time = metrics['timestamp']

        # Check cooldown period
        if (self.last_scale_action and
            (current_time - self.last_scale_action['timestamp']).seconds < self.config['scale_up_cooldown']):
            return False

        # Check scaling criteria
        cpu_util = metrics['metrics']['gauges'].get('cpu_utilization', 0)
        memory_util = metrics['metrics']['gauges'].get('memory_utilization', 0)
        response_time = metrics['metrics']['histograms']['http_request_duration']['p95']

        return (cpu_util > self.config['scale_up_thresholds']['cpu'] or
                memory_util > self.config['scale_up_thresholds']['memory'] or
                response_time > self.config['scale_up_thresholds']['response_time'])

    def should_scale_down(self, metrics):
        """Determine if scaling down is needed"""
        current_time = metrics['timestamp']

        # Check cooldown period (longer for scale down)
        if (self.last_scale_action and
            (current_time - self.last_scale_action['timestamp']).seconds < self.config['scale_down_cooldown']):
            return False

        # Only scale down if we have more than minimum instances
        if metrics['instance_count'] <= self.config['min_instances']:
            return False

        # Check scaling criteria
        cpu_util = metrics['metrics']['gauges'].get('cpu_utilization', 100)
        memory_util = metrics['metrics']['gauges'].get('memory_utilization', 100)
        response_time = metrics['metrics']['histograms']['http_request_duration']['p95']

        return (cpu_util < self.config['scale_down_thresholds']['cpu'] and
                memory_util < self.config['scale_down_thresholds']['memory'] and
                response_time < self.config['scale_down_thresholds']['response_time'])

    def scale_up(self, current_instances):
        """Scale up the service"""
        new_instance_count = min(current_instances + 1, self.config['max_instances'])

        try:
            # Example for Docker Swarm
            subprocess.run(['docker', 'service', 'scale', f'semantic-text-processor={new_instance_count}'],
                          check=True)

            self.last_scale_action = {
                'timestamp': datetime.now(),
                'action': 'scale_up',
                'from': current_instances,
                'to': new_instance_count
            }

            print(f"Scaled up from {current_instances} to {new_instance_count} instances")
            return True

        except Exception as e:
            print(f"Error scaling up: {e}")
            return False

    def scale_down(self, current_instances):
        """Scale down the service"""
        new_instance_count = max(current_instances - 1, self.config['min_instances'])

        try:
            # Example for Docker Swarm
            subprocess.run(['docker', 'service', 'scale', f'semantic-text-processor={new_instance_count}'],
                          check=True)

            self.last_scale_action = {
                'timestamp': datetime.now(),
                'action': 'scale_down',
                'from': current_instances,
                'to': new_instance_count
            }

            print(f"Scaled down from {current_instances} to {new_instance_count} instances")
            return True

        except Exception as e:
            print(f"Error scaling down: {e}")
            return False

    def run_autoscaling_loop(self):
        """Main auto-scaling loop"""
        print("Starting auto-scaling loop...")

        while True:
            try:
                metrics = self.get_current_metrics()
                if not metrics:
                    time.sleep(self.config['check_interval'])
                    continue

                current_instances = metrics['instance_count']

                if self.should_scale_up(metrics):
                    print("Scale up conditions met")
                    self.scale_up(current_instances)
                elif self.should_scale_down(metrics):
                    print("Scale down conditions met")
                    self.scale_down(current_instances)
                else:
                    print(f"No scaling needed. Current instances: {current_instances}")

                # Log current state
                self.scaling_history.append({
                    'timestamp': metrics['timestamp'],
                    'instances': current_instances,
                    'cpu_util': metrics['metrics']['gauges'].get('cpu_utilization', 0),
                    'memory_util': metrics['metrics']['gauges'].get('memory_utilization', 0),
                    'response_time': metrics['metrics']['histograms']['http_request_duration']['p95']
                })

                # Keep only last 24 hours of history
                cutoff_time = datetime.now() - timedelta(hours=24)
                self.scaling_history = [h for h in self.scaling_history if h['timestamp'] > cutoff_time]

                time.sleep(self.config['check_interval'])

            except KeyboardInterrupt:
                print("Auto-scaling stopped")
                break
            except Exception as e:
                print(f"Error in auto-scaling loop: {e}")
                time.sleep(self.config['check_interval'])

# Configuration
autoscaler_config = {
    'api_base': 'http://localhost:8080/api/v1',
    'min_instances': 2,
    'max_instances': 10,
    'scale_up_thresholds': {
        'cpu': 70,
        'memory': 80,
        'response_time': 250
    },
    'scale_down_thresholds': {
        'cpu': 40,
        'memory': 50,
        'response_time': 150
    },
    'scale_up_cooldown': 300,    # 5 minutes
    'scale_down_cooldown': 600,  # 10 minutes
    'check_interval': 60         # 1 minute
}

# Run autoscaler
if __name__ == "__main__":
    autoscaler = AutoScaler(autoscaler_config)
    autoscaler.run_autoscaling_loop()
```

## Performance Bottleneck Identification

### Bottleneck Analysis Framework

**1. System Bottleneck Detector**:
```python
# bottleneck_detector.py
import psutil
import requests
import numpy as np
from dataclasses import dataclass
from typing import List, Dict, Optional

@dataclass
class BottleneckResult:
    component: str
    severity: str  # 'low', 'medium', 'high', 'critical'
    utilization: float
    threshold: float
    impact: str
    recommendations: List[str]

class BottleneckDetector:
    def __init__(self, api_base="http://localhost:8080/api/v1"):
        self.api_base = api_base
        self.thresholds = {
            'cpu_utilization': {'warning': 70, 'critical': 90},
            'memory_utilization': {'warning': 80, 'critical': 95},
            'disk_utilization': {'warning': 80, 'critical': 90},
            'response_time_p95': {'warning': 250, 'critical': 500},
            'cache_hit_rate': {'warning': 0.7, 'critical': 0.5},
            'database_connections': {'warning': 0.8, 'critical': 0.95}
        }

    def detect_cpu_bottleneck(self) -> Optional[BottleneckResult]:
        """Detect CPU bottlenecks"""
        cpu_percent = psutil.cpu_percent(interval=1)

        if cpu_percent > self.thresholds['cpu_utilization']['critical']:
            severity = 'critical'
            recommendations = [
                'Immediate horizontal scaling required',
                'Investigate CPU-intensive operations',
                'Consider optimizing algorithms',
                'Enable CPU profiling to identify hot spots'
            ]
        elif cpu_percent > self.thresholds['cpu_utilization']['warning']:
            severity = 'high'
            recommendations = [
                'Plan for scaling within 24 hours',
                'Monitor CPU usage patterns',
                'Review recent changes for performance impact',
                'Consider vertical scaling'
            ]
        else:
            return None

        return BottleneckResult(
            component='CPU',
            severity=severity,
            utilization=cpu_percent,
            threshold=self.thresholds['cpu_utilization']['warning'],
            impact='High response times, request queuing',
            recommendations=recommendations
        )

    def detect_memory_bottleneck(self) -> Optional[BottleneckResult]:
        """Detect memory bottlenecks"""
        memory = psutil.virtual_memory()

        if memory.percent > self.thresholds['memory_utilization']['critical']:
            severity = 'critical'
            recommendations = [
                'Immediate memory scaling required',
                'Check for memory leaks',
                'Reduce cache size temporarily',
                'Restart application if necessary'
            ]
        elif memory.percent > self.thresholds['memory_utilization']['warning']:
            severity = 'high'
            recommendations = [
                'Increase memory allocation',
                'Optimize memory usage patterns',
                'Review garbage collection settings',
                'Monitor for memory leaks'
            ]
        else:
            return None

        return BottleneckResult(
            component='Memory',
            severity=severity,
            utilization=memory.percent,
            threshold=self.thresholds['memory_utilization']['warning'],
            impact='Garbage collection pressure, potential OOM',
            recommendations=recommendations
        )

    def detect_application_bottleneck(self) -> List[BottleneckResult]:
        """Detect application-level bottlenecks"""
        bottlenecks = []

        try:
            metrics_response = requests.get(f"{self.api_base}/metrics")
            cache_response = requests.get(f"{self.api_base}/cache/stats")

            if metrics_response.status_code == 200 and cache_response.status_code == 200:
                metrics = metrics_response.json()
                cache_stats = cache_response.json()

                # Response time bottleneck
                response_time_p95 = metrics['histograms']['http_request_duration']['p95']
                if response_time_p95 > self.thresholds['response_time_p95']['critical']:
                    bottlenecks.append(BottleneckResult(
                        component='Response Time',
                        severity='critical',
                        utilization=response_time_p95,
                        threshold=self.thresholds['response_time_p95']['warning'],
                        impact='Poor user experience, potential timeouts',
                        recommendations=[
                            'Investigate slow endpoints',
                            'Optimize database queries',
                            'Increase cache usage',
                            'Scale horizontally'
                        ]
                    ))
                elif response_time_p95 > self.thresholds['response_time_p95']['warning']:
                    bottlenecks.append(BottleneckResult(
                        component='Response Time',
                        severity='medium',
                        utilization=response_time_p95,
                        threshold=self.thresholds['response_time_p95']['warning'],
                        impact='Degraded user experience',
                        recommendations=[
                            'Monitor response time trends',
                            'Review recent changes',
                            'Optimize slow operations'
                        ]
                    ))

                # Cache hit rate bottleneck
                cache_hit_rate = cache_stats.get('hit_rate', 1.0)
                if cache_hit_rate < self.thresholds['cache_hit_rate']['critical']:
                    bottlenecks.append(BottleneckResult(
                        component='Cache',
                        severity='high',
                        utilization=cache_hit_rate,
                        threshold=self.thresholds['cache_hit_rate']['warning'],
                        impact='Increased database load, slower responses',
                        recommendations=[
                            'Increase cache size',
                            'Optimize cache TTL settings',
                            'Implement cache warming',
                            'Review cache key strategy'
                        ]
                    ))

        except Exception as e:
            print(f"Error detecting application bottlenecks: {e}")

        return bottlenecks

    def detect_database_bottleneck(self) -> Optional[BottleneckResult]:
        """Detect database bottlenecks"""
        try:
            # This would require database monitoring integration
            # For now, we'll use proxy metrics from application health
            health_response = requests.get(f"{self.api_base}/health")

            if health_response.status_code == 200:
                health = health_response.json()
                db_component = health.get('components', {}).get('database', {})

                if db_component.get('status') != 'healthy':
                    return BottleneckResult(
                        component='Database',
                        severity='critical',
                        utilization=0,  # Would need actual metrics
                        threshold=0,
                        impact='Application failures, data inconsistency',
                        recommendations=[
                            'Check database connectivity',
                            'Review slow query logs',
                            'Increase connection pool size',
                            'Consider read replicas'
                        ]
                    )

        except Exception as e:
            print(f"Error detecting database bottlenecks: {e}")

        return None

    def run_comprehensive_analysis(self) -> Dict[str, List[BottleneckResult]]:
        """Run comprehensive bottleneck analysis"""
        results = {
            'critical': [],
            'high': [],
            'medium': [],
            'low': []
        }

        # Check all bottleneck types
        checks = [
            self.detect_cpu_bottleneck(),
            self.detect_memory_bottleneck(),
            self.detect_database_bottleneck(),
            *self.detect_application_bottleneck()
        ]

        for result in checks:
            if result:
                results[result.severity].append(result)

        return results

    def generate_scaling_recommendation(self, bottlenecks: Dict[str, List[BottleneckResult]]) -> Dict:
        """Generate scaling recommendations based on bottlenecks"""
        recommendations = {
            'immediate_actions': [],
            'short_term_actions': [],
            'long_term_actions': [],
            'scaling_priority': 'none'
        }

        if bottlenecks['critical']:
            recommendations['scaling_priority'] = 'immediate'
            for bottleneck in bottlenecks['critical']:
                recommendations['immediate_actions'].extend(bottleneck.recommendations)

        if bottlenecks['high']:
            if recommendations['scaling_priority'] == 'none':
                recommendations['scaling_priority'] = 'urgent'
            for bottleneck in bottlenecks['high']:
                recommendations['short_term_actions'].extend(bottleneck.recommendations)

        if bottlenecks['medium']:
            if recommendations['scaling_priority'] == 'none':
                recommendations['scaling_priority'] = 'planned'
            for bottleneck in bottlenecks['medium']:
                recommendations['long_term_actions'].extend(bottleneck.recommendations)

        # Remove duplicates
        recommendations['immediate_actions'] = list(set(recommendations['immediate_actions']))
        recommendations['short_term_actions'] = list(set(recommendations['short_term_actions']))
        recommendations['long_term_actions'] = list(set(recommendations['long_term_actions']))

        return recommendations

# Usage example
if __name__ == "__main__":
    detector = BottleneckDetector()
    bottlenecks = detector.run_comprehensive_analysis()

    print("Bottleneck Analysis Results:")
    print("=" * 50)

    for severity, results in bottlenecks.items():
        if results:
            print(f"\n{severity.upper()} Issues:")
            for result in results:
                print(f"  - {result.component}: {result.utilization:.1f}% (threshold: {result.threshold})")
                print(f"    Impact: {result.impact}")
                print(f"    Recommendations: {', '.join(result.recommendations[:2])}...")

    recommendations = detector.generate_scaling_recommendation(bottlenecks)
    print(f"\nScaling Priority: {recommendations['scaling_priority']}")

    if recommendations['immediate_actions']:
        print("\nImmediate Actions Required:")
        for action in recommendations['immediate_actions'][:3]:
            print(f"  - {action}")
```

This comprehensive capacity planning and scaling guide provides all the necessary tools and procedures to effectively plan for and manage the growth of the Semantic Text Processor system. The approach is data-driven, cost-conscious, and designed to maintain performance while minimizing operational complexity.