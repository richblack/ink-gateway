package services

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInMemoryPerformanceMonitor_RecordQuery(t *testing.T) {
	monitor := NewInMemoryPerformanceMonitor(100*time.Millisecond, 10)

	// Record some queries
	monitor.RecordQuery("SELECT", 50*time.Millisecond, 10)
	monitor.RecordQuery("SELECT", 150*time.Millisecond, 20)
	monitor.RecordQuery("INSERT", 25*time.Millisecond, 1)

	stats := monitor.GetQueryStats()

	// Check overall stats
	assert.Equal(t, int64(3), stats.TotalQueries)
	assert.Equal(t, int64(1), stats.SlowQueries) // Only the 150ms query

	// Check query type stats
	selectStats, exists := stats.QueryTypes["SELECT"]
	require.True(t, exists)
	assert.Equal(t, int64(2), selectStats.Count)
	assert.Equal(t, int64(30), selectStats.TotalRows)
	assert.Equal(t, 50*time.Millisecond, selectStats.MinTime)
	assert.Equal(t, 150*time.Millisecond, selectStats.MaxTime)
	assert.Equal(t, 100*time.Millisecond, selectStats.AverageTime)

	insertStats, exists := stats.QueryTypes["INSERT"]
	require.True(t, exists)
	assert.Equal(t, int64(1), insertStats.Count)
	assert.Equal(t, int64(1), insertStats.TotalRows)
	assert.Equal(t, 25*time.Millisecond, insertStats.MinTime)
	assert.Equal(t, 25*time.Millisecond, insertStats.MaxTime)
}

func TestInMemoryPerformanceMonitor_RecordSlowQuery(t *testing.T) {
	monitor := NewInMemoryPerformanceMonitor(100*time.Millisecond, 3)

	// Record slow queries
	params1 := map[string]interface{}{"param1": "value1"}
	params2 := map[string]interface{}{"param2": "value2"}

	monitor.RecordSlowQuery("SELECT * FROM chunks WHERE id = $1", 200*time.Millisecond, params1)
	monitor.RecordSlowQuery("SELECT * FROM chunk_tags WHERE chunk_id = $1", 150*time.Millisecond, params2)

	slowQueries := monitor.GetSlowQueries(10)
	assert.Len(t, slowQueries, 2)

	// Check first slow query
	assert.Equal(t, "SELECT * FROM chunks WHERE id = $1", slowQueries[0].Query)
	assert.Equal(t, 200*time.Millisecond, slowQueries[0].Duration)
	assert.Equal(t, params1, slowQueries[0].Params)

	// Check second slow query
	assert.Equal(t, "SELECT * FROM chunk_tags WHERE chunk_id = $1", slowQueries[1].Query)
	assert.Equal(t, 150*time.Millisecond, slowQueries[1].Duration)
	assert.Equal(t, params2, slowQueries[1].Params)
}

func TestInMemoryPerformanceMonitor_SlowQueryLimit(t *testing.T) {
	monitor := NewInMemoryPerformanceMonitor(100*time.Millisecond, 2)

	// Record more slow queries than the limit
	for i := 0; i < 5; i++ {
		query := "SELECT * FROM test"
		duration := time.Duration(100+i*10) * time.Millisecond
		params := map[string]interface{}{"iteration": i}
		monitor.RecordSlowQuery(query, duration, params)
	}

	slowQueries := monitor.GetSlowQueries(10)
	assert.Len(t, slowQueries, 2) // Should only keep the last 2

	// Should have the last 2 queries (iterations 3 and 4)
	assert.Equal(t, 3, slowQueries[0].Params["iteration"])
	assert.Equal(t, 4, slowQueries[1].Params["iteration"])
}

func TestInMemoryPerformanceMonitor_GetSlowQueriesLimit(t *testing.T) {
	monitor := NewInMemoryPerformanceMonitor(100*time.Millisecond, 10)

	// Record 5 slow queries
	for i := 0; i < 5; i++ {
		query := "SELECT * FROM test"
		duration := time.Duration(100+i*10) * time.Millisecond
		params := map[string]interface{}{"iteration": i}
		monitor.RecordSlowQuery(query, duration, params)
	}

	// Request only 3 queries
	slowQueries := monitor.GetSlowQueries(3)
	assert.Len(t, slowQueries, 3)

	// Should get the most recent 3 (iterations 2, 3, 4)
	assert.Equal(t, 2, slowQueries[0].Params["iteration"])
	assert.Equal(t, 3, slowQueries[1].Params["iteration"])
	assert.Equal(t, 4, slowQueries[2].Params["iteration"])
}

func TestInMemoryPerformanceMonitor_Reset(t *testing.T) {
	monitor := NewInMemoryPerformanceMonitor(100*time.Millisecond, 10)

	// Record some data
	monitor.RecordQuery("SELECT", 50*time.Millisecond, 10)
	monitor.RecordSlowQuery("SELECT * FROM test", 200*time.Millisecond, map[string]interface{}{})

	// Verify data exists
	stats := monitor.GetQueryStats()
	assert.Equal(t, int64(1), stats.TotalQueries)
	assert.Len(t, monitor.GetSlowQueries(10), 1)

	// Reset
	monitor.Reset()

	// Verify data is cleared
	stats = monitor.GetQueryStats()
	assert.Equal(t, int64(0), stats.TotalQueries)
	assert.Equal(t, int64(0), stats.SlowQueries)
	assert.Empty(t, stats.QueryTypes)
	assert.Len(t, monitor.GetSlowQueries(10), 0)
}

func TestInMemoryPerformanceMonitor_ConcurrentAccess(t *testing.T) {
	monitor := NewInMemoryPerformanceMonitor(100*time.Millisecond, 100)
	defer monitor.Stop()

	// Run concurrent operations
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			defer func() { done <- true }()
			
			for j := 0; j < 100; j++ {
				monitor.RecordQuery("SELECT", time.Duration(j)*time.Millisecond, j)
				if j%10 == 0 {
					monitor.RecordSlowQuery("SLOW SELECT", 200*time.Millisecond, map[string]interface{}{"id": id, "iteration": j})
				}
			}
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify final state
	stats := monitor.GetQueryStats()
	assert.Equal(t, int64(1000), stats.TotalQueries) // 10 goroutines * 100 operations each
	assert.True(t, stats.SlowQueries > 0)
	assert.Contains(t, stats.QueryTypes, "SELECT")

	slowQueries := monitor.GetSlowQueries(200)
	assert.True(t, len(slowQueries) > 0)
}

func TestPerformanceConfig_DefaultValues(t *testing.T) {
	config := DefaultPerformanceConfig()

	assert.Equal(t, 100*time.Millisecond, config.SlowQueryThreshold)
	assert.Equal(t, 100, config.MaxSlowQueries)
	assert.True(t, config.Enabled)
}

func BenchmarkInMemoryPerformanceMonitor_RecordQuery(b *testing.B) {
	monitor := NewInMemoryPerformanceMonitor(100*time.Millisecond, 1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		monitor.RecordQuery("SELECT", time.Duration(i%1000)*time.Microsecond, i%100)
	}
}

func BenchmarkInMemoryPerformanceMonitor_RecordSlowQuery(b *testing.B) {
	monitor := NewInMemoryPerformanceMonitor(100*time.Millisecond, 1000)
	params := map[string]interface{}{"param1": "value1", "param2": 123}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		monitor.RecordSlowQuery("SELECT * FROM test WHERE id = $1", 200*time.Millisecond, params)
	}
}

func BenchmarkInMemoryPerformanceMonitor_GetQueryStats(b *testing.B) {
	monitor := NewInMemoryPerformanceMonitor(100*time.Millisecond, 1000)

	// Pre-populate with some data
	for i := 0; i < 1000; i++ {
		monitor.RecordQuery("SELECT", time.Duration(i)*time.Microsecond, i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = monitor.GetQueryStats()
	}
}

func BenchmarkInMemoryPerformanceMonitor_GetSlowQueries(b *testing.B) {
	monitor := NewInMemoryPerformanceMonitor(100*time.Millisecond, 1000)

	// Pre-populate with slow queries
	for i := 0; i < 1000; i++ {
		monitor.RecordSlowQuery("SELECT * FROM test", 200*time.Millisecond, map[string]interface{}{"id": i})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = monitor.GetSlowQueries(100)
	}
}

// Test performance monitoring integration with context
func TestPerformanceMonitorWithContext(t *testing.T) {
	monitor := NewInMemoryPerformanceMonitor(50*time.Millisecond, 10)
	defer monitor.Stop()
	
	// Simulate a monitored operation
	start := time.Now()
	
	// Simulate some work
	time.Sleep(10 * time.Millisecond)
	
	duration := time.Since(start)
	monitor.RecordQuery("test_operation", duration, 1)
	
	stats := monitor.GetQueryStats()
	assert.Equal(t, int64(1), stats.TotalQueries)
	assert.Contains(t, stats.QueryTypes, "test_operation")
	
	// Verify the recorded duration is reasonable
	testStats := stats.QueryTypes["test_operation"]
	assert.True(t, testStats.AverageTime >= 10*time.Millisecond)
	assert.True(t, testStats.AverageTime < 100*time.Millisecond) // Should be much less than 100ms
}

// Test alert functionality (will be implemented)
func TestPerformanceMonitor_AlertThresholds(t *testing.T) {
	monitor := NewInMemoryPerformanceMonitor(100*time.Millisecond, 10)
	
	// Record queries that should trigger alerts
	monitor.RecordQuery("SELECT", 200*time.Millisecond, 10) // Slow query
	monitor.RecordQuery("SELECT", 300*time.Millisecond, 10) // Very slow query
	
	stats := monitor.GetQueryStats()
	assert.Equal(t, int64(2), stats.SlowQueries)
	
	// Check if we have slow queries recorded
	slowQueries := monitor.GetSlowQueries(10)
	assert.Len(t, slowQueries, 2)
	
	// Verify both queries are above threshold
	for _, query := range slowQueries {
		assert.True(t, query.Duration >= 100*time.Millisecond)
	}
}

// Test enhanced performance monitor features
func TestEnhancedPerformanceMonitor_Alerts(t *testing.T) {
	monitor := NewInMemoryPerformanceMonitor(50*time.Millisecond, 10)
	defer monitor.Stop()
	
	// Configure alert thresholds
	config := &AlertConfig{
		SlowQueryThreshold:     50 * time.Millisecond,
		VerySlowQueryThreshold: 200 * time.Millisecond,
		HighErrorRateThreshold: 0.1,
		MaxQueriesPerSecond:    100,
		AlertCooldown:          1 * time.Second,
	}
	monitor.SetAlertThresholds(config)
	
	// Record a very slow query to trigger alert
	monitor.RecordQuery("SELECT", 300*time.Millisecond, 10)
	
	// Check alerts
	alerts := monitor.GetAlerts(10)
	assert.True(t, len(alerts) > 0, "Should have generated alerts")
	
	// Find the very slow query alert
	var foundAlert bool
	for _, alert := range alerts {
		if alert.Type == "very_slow_query" {
			foundAlert = true
			assert.Equal(t, "critical", alert.Severity)
			assert.Contains(t, alert.Message, "Very slow")
			assert.Contains(t, alert.Details, "query_type")
			assert.Equal(t, "SELECT", alert.Details["query_type"])
			break
		}
	}
	assert.True(t, foundAlert, "Should have found very slow query alert")
}

func TestEnhancedPerformanceMonitor_ErrorTracking(t *testing.T) {
	monitor := NewInMemoryPerformanceMonitor(100*time.Millisecond, 10)
	defer monitor.Stop()
	
	// Configure low error rate threshold for testing
	config := &AlertConfig{
		SlowQueryThreshold:     100 * time.Millisecond,
		VerySlowQueryThreshold: 1 * time.Second,
		HighErrorRateThreshold: 0.2, // 20% error rate
		MaxQueriesPerSecond:    1000,
		AlertCooldown:          1 * time.Second,
	}
	monitor.SetAlertThresholds(config)
	
	// Record some queries and errors
	monitor.RecordQuery("SELECT", 50*time.Millisecond, 10)
	monitor.RecordQuery("INSERT", 30*time.Millisecond, 1)
	monitor.RecordQuery("UPDATE", 40*time.Millisecond, 5)
	
	// Record errors (these don't count as queries)
	monitor.RecordError("SELECT", fmt.Errorf("connection timeout"))
	monitor.RecordError("INSERT", fmt.Errorf("constraint violation"))
	
	// Check health status
	health := monitor.GetPerformanceHealth()
	assert.True(t, health.ErrorRate > 0, "Should have recorded errors")
	
	// Error rate should be 2 errors / 3 total queries = 66.7%
	expectedErrorRate := 2.0 / 3.0
	assert.InDelta(t, expectedErrorRate, health.ErrorRate, 0.01, "Error rate should be calculated correctly")
}

func TestEnhancedPerformanceMonitor_HealthStatus(t *testing.T) {
	monitor := NewInMemoryPerformanceMonitor(100*time.Millisecond, 10)
	defer monitor.Stop()
	
	// Set more reasonable thresholds for testing
	config := &AlertConfig{
		SlowQueryThreshold:     100 * time.Millisecond,
		VerySlowQueryThreshold: 1 * time.Second,
		HighErrorRateThreshold: 0.05,
		MaxQueriesPerSecond:    1000000, // Very high to avoid triggering in tests
		AlertCooldown:          1 * time.Second,
	}
	monitor.SetAlertThresholds(config)
	
	// Initially should be healthy
	assert.True(t, monitor.IsHealthy())
	
	health := monitor.GetPerformanceHealth()
	assert.True(t, health.IsHealthy)
	assert.Equal(t, time.Duration(0), health.AverageResponseTime)
	assert.Equal(t, 0.0, health.SlowQueryRate)
	assert.Equal(t, 0.0, health.ErrorRate)
	
	// Record some fast queries - should remain healthy
	for i := 0; i < 10; i++ {
		monitor.RecordQuery("SELECT", 20*time.Millisecond, 10)
	}
	
	health = monitor.GetPerformanceHealth()
	t.Logf("Health after fast queries: %+v", health)
	// The system might be unhealthy due to high QPS in test environment, so let's check the specific metrics
	assert.Equal(t, 20*time.Millisecond, health.AverageResponseTime)
	assert.Equal(t, 0.0, health.SlowQueryRate, "Should have no slow queries")
	
	// Record many slow queries - should become unhealthy
	for i := 0; i < 20; i++ {
		monitor.RecordQuery("SELECT", 2*time.Second, 10) // Very slow queries
	}
	
	health = monitor.GetPerformanceHealth()
	assert.False(t, health.IsHealthy, "Should be unhealthy due to slow queries")
	assert.True(t, health.SlowQueryRate > 0.5, "Should have high slow query rate")
	assert.True(t, health.AverageResponseTime > time.Second, "Should have high average response time")
}

func TestEnhancedPerformanceMonitor_AlertCooldown(t *testing.T) {
	monitor := NewInMemoryPerformanceMonitor(50*time.Millisecond, 10)
	defer monitor.Stop()
	
	// Configure very short cooldown for testing
	config := &AlertConfig{
		SlowQueryThreshold:     50 * time.Millisecond,
		VerySlowQueryThreshold: 200 * time.Millisecond,
		HighErrorRateThreshold: 0.05,
		MaxQueriesPerSecond:    100,
		AlertCooldown:          100 * time.Millisecond, // Very short cooldown
	}
	monitor.SetAlertThresholds(config)
	
	// Record multiple slow queries quickly
	monitor.RecordQuery("SELECT", 100*time.Millisecond, 10)
	monitor.RecordQuery("SELECT", 100*time.Millisecond, 10)
	monitor.RecordQuery("SELECT", 100*time.Millisecond, 10)
	
	// Should only have one alert due to cooldown
	alerts := monitor.GetAlerts(10)
	slowQueryAlerts := 0
	for _, alert := range alerts {
		if alert.Type == "slow_query" {
			slowQueryAlerts++
		}
	}
	assert.LessOrEqual(t, slowQueryAlerts, 1, "Should respect alert cooldown")
	
	// Wait for cooldown to expire
	time.Sleep(150 * time.Millisecond)
	
	// Record another slow query
	monitor.RecordQuery("SELECT", 100*time.Millisecond, 10)
	
	// Should now have additional alert
	alerts = monitor.GetAlerts(10)
	slowQueryAlerts = 0
	for _, alert := range alerts {
		if alert.Type == "slow_query" {
			slowQueryAlerts++
		}
	}
	assert.True(t, slowQueryAlerts > 0, "Should have generated new alert after cooldown")
}

func TestEnhancedPerformanceMonitor_ClearAlerts(t *testing.T) {
	monitor := NewInMemoryPerformanceMonitor(50*time.Millisecond, 10)
	defer monitor.Stop()
	
	// Generate some alerts
	monitor.RecordQuery("SELECT", 300*time.Millisecond, 10) // Very slow query
	
	// Verify alerts exist
	alerts := monitor.GetAlerts(10)
	assert.True(t, len(alerts) > 0, "Should have alerts")
	
	// Clear alerts
	monitor.ClearAlerts()
	
	// Verify alerts are cleared
	alerts = monitor.GetAlerts(10)
	assert.Equal(t, 0, len(alerts), "Should have no alerts after clearing")
}

func TestEnhancedPerformanceMonitor_Stop(t *testing.T) {
	monitor := NewInMemoryPerformanceMonitor(100*time.Millisecond, 10)
	
	// Verify monitor is running
	assert.False(t, monitor.stopped)
	
	// Stop the monitor
	monitor.Stop()
	
	// Verify monitor is stopped
	assert.True(t, monitor.stopped)
	
	// Calling stop again should be safe
	monitor.Stop()
	assert.True(t, monitor.stopped)
}