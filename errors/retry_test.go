package errors

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultRetryConfig(t *testing.T) {
	config := DefaultRetryConfig()
	
	assert.Equal(t, 3, config.MaxRetries)
	assert.Equal(t, 100*time.Millisecond, config.BaseDelay)
	assert.Equal(t, 30*time.Second, config.MaxDelay)
	assert.Equal(t, 2.0, config.BackoffFactor)
	assert.True(t, config.Jitter)
	assert.Contains(t, config.RetryableErrors, ErrTypeExternal)
	assert.Contains(t, config.RetryableErrors, ErrTypeDatabase)
	assert.Contains(t, config.RetryableErrors, ErrTypeNetwork)
	assert.Contains(t, config.RetryableErrors, ErrTypeTimeout)
	assert.Contains(t, config.RetryableErrors, ErrTypeRateLimit)
}

func TestExternalServiceRetryConfig(t *testing.T) {
	config := ExternalServiceRetryConfig()
	
	assert.Equal(t, 5, config.MaxRetries)
	assert.Equal(t, 200*time.Millisecond, config.BaseDelay)
	assert.Equal(t, 60*time.Second, config.MaxDelay)
	assert.Equal(t, 2.0, config.BackoffFactor)
	assert.True(t, config.Jitter)
}

func TestDatabaseRetryConfig(t *testing.T) {
	config := DatabaseRetryConfig()
	
	assert.Equal(t, 3, config.MaxRetries)
	assert.Equal(t, 50*time.Millisecond, config.BaseDelay)
	assert.Equal(t, 5*time.Second, config.MaxDelay)
	assert.Equal(t, 1.5, config.BackoffFactor)
	assert.True(t, config.Jitter)
}

func TestRetryer_Execute_Success(t *testing.T) {
	config := &RetryConfig{
		MaxRetries:    3,
		BaseDelay:     10 * time.Millisecond,
		MaxDelay:      1 * time.Second,
		BackoffFactor: 2.0,
		Jitter:        false,
	}
	
	retryer := NewRetryer(config)
	ctx := context.Background()
	
	callCount := 0
	operation := func() error {
		callCount++
		return nil // Success on first try
	}
	
	err := retryer.Execute(ctx, operation)
	
	assert.NoError(t, err)
	assert.Equal(t, 1, callCount)
}

func TestRetryer_Execute_RetryableError(t *testing.T) {
	config := &RetryConfig{
		MaxRetries:    3,
		BaseDelay:     1 * time.Millisecond,
		MaxDelay:      1 * time.Second,
		BackoffFactor: 2.0,
		Jitter:        false,
		RetryableErrors: []ErrorType{ErrTypeExternal},
	}
	
	retryer := NewRetryer(config)
	ctx := context.Background()
	
	callCount := 0
	operation := func() error {
		callCount++
		if callCount < 3 {
			return NewExternalServiceError("TEST", "temporary failure", nil)
		}
		return nil // Success on third try
	}
	
	err := retryer.Execute(ctx, operation)
	
	assert.NoError(t, err)
	assert.Equal(t, 3, callCount)
}

func TestRetryer_Execute_NonRetryableError(t *testing.T) {
	config := &RetryConfig{
		MaxRetries:    3,
		BaseDelay:     1 * time.Millisecond,
		MaxDelay:      1 * time.Second,
		BackoffFactor: 2.0,
		Jitter:        false,
		RetryableErrors: []ErrorType{ErrTypeExternal},
	}
	
	retryer := NewRetryer(config)
	ctx := context.Background()
	
	callCount := 0
	operation := func() error {
		callCount++
		return NewValidationError("TEST", "validation failed", nil)
	}
	
	err := retryer.Execute(ctx, operation)
	
	assert.Error(t, err)
	assert.Equal(t, 1, callCount) // Should not retry
	
	appErr, ok := AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, ErrTypeValidation, appErr.Type)
}

func TestRetryer_Execute_MaxRetriesExceeded(t *testing.T) {
	config := &RetryConfig{
		MaxRetries:    2,
		BaseDelay:     1 * time.Millisecond,
		MaxDelay:      1 * time.Second,
		BackoffFactor: 2.0,
		Jitter:        false,
		RetryableErrors: []ErrorType{ErrTypeExternal},
	}
	
	retryer := NewRetryer(config)
	ctx := context.Background()
	
	callCount := 0
	operation := func() error {
		callCount++
		return NewExternalServiceError("TEST", "persistent failure", nil)
	}
	
	err := retryer.Execute(ctx, operation)
	
	assert.Error(t, err)
	assert.Equal(t, 3, callCount) // Initial attempt + 2 retries
	
	appErr, ok := AsAppError(err)
	require.True(t, ok)
	assert.Contains(t, appErr.Details, "Failed after 2 retries")
}

func TestRetryer_Execute_ContextCanceled(t *testing.T) {
	config := &RetryConfig{
		MaxRetries:    3,
		BaseDelay:     100 * time.Millisecond,
		MaxDelay:      1 * time.Second,
		BackoffFactor: 2.0,
		Jitter:        false,
		RetryableErrors: []ErrorType{ErrTypeExternal},
	}
	
	retryer := NewRetryer(config)
	ctx, cancel := context.WithCancel(context.Background())
	
	callCount := 0
	operation := func() error {
		callCount++
		if callCount == 1 {
			// Cancel context after first failure
			go func() {
				time.Sleep(10 * time.Millisecond)
				cancel()
			}()
			return NewExternalServiceError("TEST", "failure", nil)
		}
		return nil
	}
	
	err := retryer.Execute(ctx, operation)
	
	assert.Error(t, err)
	assert.Equal(t, context.Canceled, err)
	assert.Equal(t, 1, callCount) // Should not retry after context cancellation
}

func TestExecuteWithResult_Success(t *testing.T) {
	config := &RetryConfig{
		MaxRetries:    3,
		BaseDelay:     1 * time.Millisecond,
		MaxDelay:      1 * time.Second,
		BackoffFactor: 2.0,
		Jitter:        false,
		RetryableErrors: []ErrorType{ErrTypeExternal},
	}
	
	ctx := context.Background()
	
	callCount := 0
	operation := func() (string, error) {
		callCount++
		if callCount < 3 {
			return "", NewExternalServiceError("TEST", "temporary failure", nil)
		}
		return "success", nil
	}
	
	result, err := ExecuteWithResult(ctx, config, operation)
	
	assert.NoError(t, err)
	assert.Equal(t, "success", result)
	assert.Equal(t, 3, callCount)
}

func TestExecuteWithResult_Failure(t *testing.T) {
	config := &RetryConfig{
		MaxRetries:    2,
		BaseDelay:     1 * time.Millisecond,
		MaxDelay:      1 * time.Second,
		BackoffFactor: 2.0,
		Jitter:        false,
		RetryableErrors: []ErrorType{ErrTypeExternal},
	}
	
	ctx := context.Background()
	
	callCount := 0
	operation := func() (string, error) {
		callCount++
		return "partial", NewExternalServiceError("TEST", "persistent failure", nil)
	}
	
	result, err := ExecuteWithResult(ctx, config, operation)
	
	assert.Error(t, err)
	assert.Equal(t, "partial", result) // Should return last result
	assert.Equal(t, 3, callCount) // Initial attempt + 2 retries
}

func TestRetryer_calculateDelay(t *testing.T) {
	tests := []struct {
		name     string
		config   *RetryConfig
		attempt  int
		minDelay time.Duration
		maxDelay time.Duration
	}{
		{
			name: "first retry",
			config: &RetryConfig{
				BaseDelay:     100 * time.Millisecond,
				BackoffFactor: 2.0,
				MaxDelay:      10 * time.Second,
				Jitter:        false,
			},
			attempt:  1,
			minDelay: 100 * time.Millisecond,
			maxDelay: 100 * time.Millisecond,
		},
		{
			name: "second retry",
			config: &RetryConfig{
				BaseDelay:     100 * time.Millisecond,
				BackoffFactor: 2.0,
				MaxDelay:      10 * time.Second,
				Jitter:        false,
			},
			attempt:  2,
			minDelay: 200 * time.Millisecond,
			maxDelay: 200 * time.Millisecond,
		},
		{
			name: "max delay reached",
			config: &RetryConfig{
				BaseDelay:     100 * time.Millisecond,
				BackoffFactor: 2.0,
				MaxDelay:      150 * time.Millisecond,
				Jitter:        false,
			},
			attempt:  2,
			minDelay: 150 * time.Millisecond,
			maxDelay: 150 * time.Millisecond,
		},
		{
			name: "with jitter",
			config: &RetryConfig{
				BaseDelay:     100 * time.Millisecond,
				BackoffFactor: 2.0,
				MaxDelay:      10 * time.Second,
				Jitter:        true,
			},
			attempt:  1,
			minDelay: 90 * time.Millisecond,  // 100ms - 10% jitter
			maxDelay: 110 * time.Millisecond, // 100ms + 10% jitter
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			retryer := NewRetryer(tt.config)
			delay := retryer.calculateDelay(tt.attempt)
			
			assert.GreaterOrEqual(t, delay, tt.minDelay)
			assert.LessOrEqual(t, delay, tt.maxDelay)
		})
	}
}

func TestCircuitBreaker_Execute_Closed(t *testing.T) {
	config := &CircuitBreakerConfig{
		FailureThreshold: 3,
		ResetTimeout:     1 * time.Second,
		MaxRequests:      2,
	}
	
	cb := NewCircuitBreaker(config)
	ctx := context.Background()
	
	// Successful operation should keep circuit closed
	err := cb.Execute(ctx, func() error {
		return nil
	})
	
	assert.NoError(t, err)
	assert.Equal(t, CircuitBreakerClosed, cb.GetState())
}

func TestCircuitBreaker_Execute_Open(t *testing.T) {
	config := &CircuitBreakerConfig{
		FailureThreshold: 2,
		ResetTimeout:     1 * time.Second,
		MaxRequests:      2,
	}
	
	cb := NewCircuitBreaker(config)
	ctx := context.Background()
	
	// Fail enough times to open the circuit
	for i := 0; i < 2; i++ {
		cb.Execute(ctx, func() error {
			return fmt.Errorf("failure")
		})
	}
	
	assert.Equal(t, CircuitBreakerOpen, cb.GetState())
	
	// Next operation should be rejected
	err := cb.Execute(ctx, func() error {
		return nil
	})
	
	assert.Error(t, err)
	appErr, ok := AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, "CIRCUIT_BREAKER_OPEN", appErr.Code)
}

func TestCircuitBreaker_Execute_HalfOpen(t *testing.T) {
	config := &CircuitBreakerConfig{
		FailureThreshold: 2,
		ResetTimeout:     10 * time.Millisecond,
		MaxRequests:      2,
	}
	
	cb := NewCircuitBreaker(config)
	ctx := context.Background()
	
	// Fail enough times to open the circuit
	for i := 0; i < 2; i++ {
		cb.Execute(ctx, func() error {
			return fmt.Errorf("failure")
		})
	}
	
	assert.Equal(t, CircuitBreakerOpen, cb.GetState())
	
	// Wait for reset timeout
	time.Sleep(15 * time.Millisecond)
	
	// First operation should transition to half-open
	err := cb.Execute(ctx, func() error {
		return nil
	})
	
	assert.NoError(t, err)
	assert.Equal(t, CircuitBreakerHalfOpen, cb.GetState())
	
	// Second successful operation should close the circuit
	err = cb.Execute(ctx, func() error {
		return nil
	})
	
	assert.NoError(t, err)
	assert.Equal(t, CircuitBreakerClosed, cb.GetState())
}

func TestCircuitBreaker_Execute_HalfOpenFailure(t *testing.T) {
	config := &CircuitBreakerConfig{
		FailureThreshold: 2,
		ResetTimeout:     10 * time.Millisecond,
		MaxRequests:      2,
	}
	
	cb := NewCircuitBreaker(config)
	ctx := context.Background()
	
	// Fail enough times to open the circuit
	for i := 0; i < 2; i++ {
		cb.Execute(ctx, func() error {
			return fmt.Errorf("failure")
		})
	}
	
	// Wait for reset timeout
	time.Sleep(15 * time.Millisecond)
	
	// Failure in half-open state should reopen the circuit
	err := cb.Execute(ctx, func() error {
		return fmt.Errorf("failure")
	})
	
	assert.Error(t, err)
	assert.Equal(t, CircuitBreakerOpen, cb.GetState())
}