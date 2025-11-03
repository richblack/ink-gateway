package errors

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"time"
)

// RetryConfig defines retry behavior configuration
type RetryConfig struct {
	MaxRetries      int           `json:"max_retries"`
	BaseDelay       time.Duration `json:"base_delay"`
	MaxDelay        time.Duration `json:"max_delay"`
	BackoffFactor   float64       `json:"backoff_factor"`
	Jitter          bool          `json:"jitter"`
	RetryableErrors []ErrorType   `json:"retryable_errors"`
}

// DefaultRetryConfig returns a sensible default retry configuration
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxRetries:    3,
		BaseDelay:     100 * time.Millisecond,
		MaxDelay:      30 * time.Second,
		BackoffFactor: 2.0,
		Jitter:        true,
		RetryableErrors: []ErrorType{
			ErrTypeExternal,
			ErrTypeDatabase,
			ErrTypeNetwork,
			ErrTypeTimeout,
			ErrTypeRateLimit,
		},
	}
}

// ExternalServiceRetryConfig returns retry config optimized for external services
func ExternalServiceRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxRetries:    5,
		BaseDelay:     200 * time.Millisecond,
		MaxDelay:      60 * time.Second,
		BackoffFactor: 2.0,
		Jitter:        true,
		RetryableErrors: []ErrorType{
			ErrTypeExternal,
			ErrTypeNetwork,
			ErrTypeTimeout,
			ErrTypeRateLimit,
		},
	}
}

// DatabaseRetryConfig returns retry config optimized for database operations
func DatabaseRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxRetries:    3,
		BaseDelay:     50 * time.Millisecond,
		MaxDelay:      5 * time.Second,
		BackoffFactor: 1.5,
		Jitter:        true,
		RetryableErrors: []ErrorType{
			ErrTypeDatabase,
			ErrTypeNetwork,
			ErrTypeTimeout,
		},
	}
}

// RetryableOperation represents an operation that can be retried
type RetryableOperation func() error

// RetryableOperationWithResult represents an operation that returns a result and can be retried
type RetryableOperationWithResult[T any] func() (T, error)

// Retryer handles retry logic with exponential backoff
type Retryer struct {
	config *RetryConfig
}

// NewRetryer creates a new retryer with the given configuration
func NewRetryer(config *RetryConfig) *Retryer {
	if config == nil {
		config = DefaultRetryConfig()
	}
	return &Retryer{config: config}
}

// Execute executes an operation with retry logic
func (r *Retryer) Execute(ctx context.Context, operation RetryableOperation) error {
	var lastErr error
	
	for attempt := 0; attempt <= r.config.MaxRetries; attempt++ {
		// Add delay before retry (except for first attempt)
		if attempt > 0 {
			delay := r.calculateDelay(attempt)
			
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
				// Continue with retry
			}
		}
		
		// Execute the operation
		err := operation()
		if err == nil {
			return nil // Success
		}
		
		lastErr = err
		
		// Check if we should retry
		if !r.shouldRetry(ctx, err, attempt) {
			break
		}
	}
	
	// All retries exhausted or non-retryable error
	return r.wrapFinalError(lastErr)
}

// ExecuteWithResult executes an operation that returns a result with retry logic
func ExecuteWithResult[T any](ctx context.Context, config *RetryConfig, operation RetryableOperationWithResult[T]) (T, error) {
	retryer := NewRetryer(config)
	var result T
	var lastErr error
	
	for attempt := 0; attempt <= retryer.config.MaxRetries; attempt++ {
		// Add delay before retry (except for first attempt)
		if attempt > 0 {
			delay := retryer.calculateDelay(attempt)
			
			select {
			case <-ctx.Done():
				return result, ctx.Err()
			case <-time.After(delay):
				// Continue with retry
			}
		}
		
		// Execute the operation
		res, err := operation()
		if err == nil {
			return res, nil // Success
		}
		
		result = res
		lastErr = err
		
		// Check if we should retry
		if !retryer.shouldRetry(ctx, err, attempt) {
			break
		}
	}
	
	// All retries exhausted or non-retryable error
	return result, retryer.wrapFinalError(lastErr)
}

// calculateDelay calculates the delay for the given attempt using exponential backoff
func (r *Retryer) calculateDelay(attempt int) time.Duration {
	// Calculate exponential backoff delay
	delay := float64(r.config.BaseDelay) * math.Pow(r.config.BackoffFactor, float64(attempt-1))
	
	// Apply maximum delay limit
	if delay > float64(r.config.MaxDelay) {
		delay = float64(r.config.MaxDelay)
	}
	
	// Add jitter to prevent thundering herd
	if r.config.Jitter {
		jitter := delay * 0.1 * (rand.Float64()*2 - 1) // ±10% jitter
		delay += jitter
	}
	
	return time.Duration(delay)
}

// shouldRetry determines if an operation should be retried
func (r *Retryer) shouldRetry(ctx context.Context, err error, attempt int) bool {
	// Don't retry if context is cancelled
	if ctx.Err() != nil {
		return false
	}
	
	// Don't retry if max attempts reached
	if attempt >= r.config.MaxRetries {
		return false
	}
	
	// Check if error is retryable
	return r.isRetryableError(err)
}

// isRetryableError checks if an error should be retried based on configuration
func (r *Retryer) isRetryableError(err error) bool {
	// Check if it's an AppError with retryable flag
	if appErr, ok := AsAppError(err); ok {
		// First check the explicit retryable flag
		if !appErr.IsRetryable() {
			return false
		}
		
		// Then check if the error type is in the retryable list
		for _, retryableType := range r.config.RetryableErrors {
			if appErr.Type == retryableType {
				return true
			}
		}
		return false
	}
	
	// For non-AppErrors, use default retry logic
	return IsRetryable(err)
}

// wrapFinalError wraps the final error with retry information
func (r *Retryer) wrapFinalError(err error) error {
	if appErr, ok := AsAppError(err); ok {
		appErr.Details = fmt.Sprintf("Failed after %d retries", r.config.MaxRetries)
		return appErr
	}
	
	return WrapError(err, ErrTypeInternal, ErrCodeProcessingError, 
		fmt.Sprintf("Operation failed after %d retries", r.config.MaxRetries))
}

// CircuitBreakerConfig defines circuit breaker behavior
type CircuitBreakerConfig struct {
	FailureThreshold int           `json:"failure_threshold"`
	ResetTimeout     time.Duration `json:"reset_timeout"`
	MaxRequests      int           `json:"max_requests"`
}

// CircuitBreakerState represents the state of a circuit breaker
type CircuitBreakerState int

const (
	CircuitBreakerClosed CircuitBreakerState = iota
	CircuitBreakerOpen
	CircuitBreakerHalfOpen
)

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	config       *CircuitBreakerConfig
	state        CircuitBreakerState
	failures     int
	lastFailTime time.Time
	requests     int
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(config *CircuitBreakerConfig) *CircuitBreaker {
	if config == nil {
		config = &CircuitBreakerConfig{
			FailureThreshold: 5,
			ResetTimeout:     60 * time.Second,
			MaxRequests:      3,
		}
	}
	
	return &CircuitBreaker{
		config: config,
		state:  CircuitBreakerClosed,
	}
}

// Execute executes an operation through the circuit breaker
func (cb *CircuitBreaker) Execute(ctx context.Context, operation RetryableOperation) error {
	if !cb.canExecute() {
		return NewExternalServiceError(
			"CIRCUIT_BREAKER_OPEN",
			"Circuit breaker is open, operation not allowed",
			nil,
		)
	}
	
	err := operation()
	cb.recordResult(err)
	return err
}

// canExecute checks if the operation can be executed based on circuit breaker state
func (cb *CircuitBreaker) canExecute() bool {
	switch cb.state {
	case CircuitBreakerClosed:
		return true
	case CircuitBreakerOpen:
		if time.Since(cb.lastFailTime) > cb.config.ResetTimeout {
			cb.state = CircuitBreakerHalfOpen
			cb.requests = 0
			return true
		}
		return false
	case CircuitBreakerHalfOpen:
		return cb.requests < cb.config.MaxRequests
	default:
		return false
	}
}

// recordResult records the result of an operation
func (cb *CircuitBreaker) recordResult(err error) {
	switch cb.state {
	case CircuitBreakerClosed:
		if err != nil {
			cb.failures++
			if cb.failures >= cb.config.FailureThreshold {
				cb.state = CircuitBreakerOpen
				cb.lastFailTime = time.Now()
			}
		} else {
			cb.failures = 0
		}
	case CircuitBreakerHalfOpen:
		cb.requests++
		if err != nil {
			cb.state = CircuitBreakerOpen
			cb.lastFailTime = time.Now()
			cb.failures = cb.config.FailureThreshold
		} else if cb.requests >= cb.config.MaxRequests {
			cb.state = CircuitBreakerClosed
			cb.failures = 0
		}
	}
}

// GetState returns the current state of the circuit breaker
func (cb *CircuitBreaker) GetState() CircuitBreakerState {
	return cb.state
}