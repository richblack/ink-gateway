package services

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockHealthChecker for testing
type MockHealthChecker struct {
	mock.Mock
	name string
}

func (m *MockHealthChecker) Name() string {
	return m.name
}

func (m *MockHealthChecker) Check(ctx context.Context) ComponentHealth {
	args := m.Called(ctx)
	return args.Get(0).(ComponentHealth)
}

func TestHealthService_RegisterChecker(t *testing.T) {
	logger := NewDefaultLogger()
	healthService := NewHealthService("1.0.0", logger)
	
	checker := &MockHealthChecker{name: "test-component"}
	healthService.RegisterChecker(checker)
	
	assert.Len(t, healthService.checkers, 1)
	assert.Contains(t, healthService.checkers, "test-component")
}

func TestHealthService_CheckHealth_AllHealthy(t *testing.T) {
	logger := NewDefaultLogger()
	healthService := NewHealthService("1.0.0", logger)
	
	// Register healthy checkers
	checker1 := &MockHealthChecker{name: "component1"}
	checker1.On("Check", mock.Anything).Return(ComponentHealth{
		Name:      "component1",
		Status:    HealthStatusHealthy,
		Message:   "All good",
		Timestamp: time.Now(),
	})
	
	checker2 := &MockHealthChecker{name: "component2"}
	checker2.On("Check", mock.Anything).Return(ComponentHealth{
		Name:      "component2",
		Status:    HealthStatusHealthy,
		Message:   "Working fine",
		Timestamp: time.Now(),
	})
	
	healthService.RegisterChecker(checker1)
	healthService.RegisterChecker(checker2)
	
	ctx := context.Background()
	systemHealth := healthService.CheckHealth(ctx)
	
	assert.Equal(t, HealthStatusHealthy, systemHealth.Status)
	assert.Equal(t, "1.0.0", systemHealth.Version)
	assert.Len(t, systemHealth.Components, 2)
	assert.Contains(t, systemHealth.Components, "component1")
	assert.Contains(t, systemHealth.Components, "component2")
	
	checker1.AssertExpectations(t)
	checker2.AssertExpectations(t)
}

func TestHealthService_CheckHealth_OneUnhealthy(t *testing.T) {
	logger := NewDefaultLogger()
	healthService := NewHealthService("1.0.0", logger)
	
	// Register one healthy and one unhealthy checker
	healthyChecker := &MockHealthChecker{name: "healthy"}
	healthyChecker.On("Check", mock.Anything).Return(ComponentHealth{
		Name:   "healthy",
		Status: HealthStatusHealthy,
	})
	
	unhealthyChecker := &MockHealthChecker{name: "unhealthy"}
	unhealthyChecker.On("Check", mock.Anything).Return(ComponentHealth{
		Name:    "unhealthy",
		Status:  HealthStatusUnhealthy,
		Message: "Something is wrong",
	})
	
	healthService.RegisterChecker(healthyChecker)
	healthService.RegisterChecker(unhealthyChecker)
	
	ctx := context.Background()
	systemHealth := healthService.CheckHealth(ctx)
	
	// Overall status should be unhealthy
	assert.Equal(t, HealthStatusUnhealthy, systemHealth.Status)
	assert.Len(t, systemHealth.Components, 2)
	
	healthyChecker.AssertExpectations(t)
	unhealthyChecker.AssertExpectations(t)
}

func TestHealthService_CheckHealth_OneDegraded(t *testing.T) {
	logger := NewDefaultLogger()
	healthService := NewHealthService("1.0.0", logger)
	
	// Register one healthy and one degraded checker
	healthyChecker := &MockHealthChecker{name: "healthy"}
	healthyChecker.On("Check", mock.Anything).Return(ComponentHealth{
		Name:   "healthy",
		Status: HealthStatusHealthy,
	})
	
	degradedChecker := &MockHealthChecker{name: "degraded"}
	degradedChecker.On("Check", mock.Anything).Return(ComponentHealth{
		Name:    "degraded",
		Status:  HealthStatusDegraded,
		Message: "Running slowly",
	})
	
	healthService.RegisterChecker(healthyChecker)
	healthService.RegisterChecker(degradedChecker)
	
	ctx := context.Background()
	systemHealth := healthService.CheckHealth(ctx)
	
	// Overall status should be degraded
	assert.Equal(t, HealthStatusDegraded, systemHealth.Status)
	assert.Len(t, systemHealth.Components, 2)
	
	healthyChecker.AssertExpectations(t)
	degradedChecker.AssertExpectations(t)
}

func TestHealthService_CheckComponent(t *testing.T) {
	logger := NewDefaultLogger()
	healthService := NewHealthService("1.0.0", logger)
	
	checker := &MockHealthChecker{name: "test-component"}
	expectedHealth := ComponentHealth{
		Name:    "test-component",
		Status:  HealthStatusHealthy,
		Message: "Component is healthy",
	}
	checker.On("Check", mock.Anything).Return(expectedHealth)
	
	healthService.RegisterChecker(checker)
	
	ctx := context.Background()
	health, err := healthService.CheckComponent(ctx, "test-component")
	
	require.NoError(t, err)
	assert.Equal(t, expectedHealth.Name, health.Name)
	assert.Equal(t, expectedHealth.Status, health.Status)
	assert.Equal(t, expectedHealth.Message, health.Message)
	
	checker.AssertExpectations(t)
}

func TestHealthService_CheckComponent_NotFound(t *testing.T) {
	logger := NewDefaultLogger()
	healthService := NewHealthService("1.0.0", logger)
	
	ctx := context.Background()
	_, err := healthService.CheckComponent(ctx, "nonexistent")
	
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "component nonexistent not found")
}

func TestHealthService_GetSystemInfo(t *testing.T) {
	logger := NewDefaultLogger()
	healthService := NewHealthService("1.0.0", logger)
	
	// Add some checkers
	checker1 := &MockHealthChecker{name: "component1"}
	checker2 := &MockHealthChecker{name: "component2"}
	healthService.RegisterChecker(checker1)
	healthService.RegisterChecker(checker2)
	
	info := healthService.GetSystemInfo()
	
	assert.Equal(t, "1.0.0", info["version"])
	assert.Equal(t, 2, info["components"])
	assert.Contains(t, info, "uptime")
	assert.Contains(t, info, "start_time")
}

func TestCacheHealthChecker(t *testing.T) {
	cache := NewInMemoryCache(10, time.Minute)
	defer cache.Stop()
	
	checker := NewCacheHealthChecker("cache", cache)
	
	ctx := context.Background()
	health := checker.Check(ctx)
	
	assert.Equal(t, "cache", health.Name)
	assert.Equal(t, HealthStatusHealthy, health.Status)
	assert.Contains(t, health.Message, "successful")
	assert.Contains(t, health.Details, "hit_rate")
	assert.Contains(t, health.Details, "size")
	assert.Contains(t, health.Details, "max_size")
}

func TestMetricsHealthChecker(t *testing.T) {
	metrics := NewInMemoryMetrics()
	checker := NewMetricsHealthChecker("metrics", metrics)
	
	ctx := context.Background()
	health := checker.Check(ctx)
	
	assert.Equal(t, "metrics", health.Name)
	assert.Equal(t, HealthStatusHealthy, health.Status)
	assert.Contains(t, health.Message, "successful")
	assert.Contains(t, health.Details, "has_system")
}

func TestHealthService_CheckTimeout(t *testing.T) {
	logger := NewDefaultLogger()
	healthService := NewHealthService("1.0.0", logger)
	
	// Create a checker that will timeout
	slowChecker := &MockHealthChecker{name: "slow"}
	slowChecker.On("Check", mock.Anything).Run(func(args mock.Arguments) {
		// Sleep longer than the timeout
		time.Sleep(6 * time.Second)
	}).Return(ComponentHealth{
		Name:   "slow",
		Status: HealthStatusHealthy,
	})
	
	healthService.RegisterChecker(slowChecker)
	
	ctx := context.Background()
	start := time.Now()
	systemHealth := healthService.CheckHealth(ctx)
	duration := time.Since(start)
	
	// Should complete within reasonable time (not wait for the full 6 seconds)
	assert.True(t, duration < 6*time.Second)
	
	// The slow component should be marked as unhealthy due to timeout
	slowHealth := systemHealth.Components["slow"]
	assert.Equal(t, HealthStatusUnhealthy, slowHealth.Status)
	assert.Contains(t, slowHealth.Message, "timed out")
}

func BenchmarkHealthService_CheckHealth(b *testing.B) {
	logger := NewDefaultLogger()
	healthService := NewHealthService("1.0.0", logger)
	
	// Register multiple checkers
	for i := 0; i < 5; i++ {
		checker := &MockHealthChecker{name: fmt.Sprintf("component%d", i)}
		checker.On("Check", mock.Anything).Return(ComponentHealth{
			Name:   fmt.Sprintf("component%d", i),
			Status: HealthStatusHealthy,
		})
		healthService.RegisterChecker(checker)
	}
	
	ctx := context.Background()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		healthService.CheckHealth(ctx)
	}
}