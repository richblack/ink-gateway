package errors

import (
	"context"
	"fmt"
	"net/http"
)

// ErrorType represents different categories of errors
type ErrorType string

const (
	ErrTypeValidation   ErrorType = "validation"
	ErrTypeExternal     ErrorType = "external_service"
	ErrTypeDatabase     ErrorType = "database"
	ErrTypeInternal     ErrorType = "internal"
	ErrTypeNetwork      ErrorType = "network"
	ErrTypeTimeout      ErrorType = "timeout"
	ErrTypeRateLimit    ErrorType = "rate_limit"
	ErrTypeAuth         ErrorType = "authentication"
	ErrTypeNotFound     ErrorType = "not_found"
	ErrTypeConflict     ErrorType = "conflict"
)

// AppError represents a standardized application error
type AppError struct {
	Type       ErrorType `json:"type"`
	Code       string    `json:"code"`
	Message    string    `json:"message"`
	Details    string    `json:"details,omitempty"`
	Cause      error     `json:"-"`
	StatusCode int       `json:"-"`
	Retryable  bool      `json:"-"`
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (caused by: %v)", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the underlying cause
func (e *AppError) Unwrap() error {
	return e.Cause
}

// IsRetryable returns whether the error should be retried
func (e *AppError) IsRetryable() bool {
	return e.Retryable
}

// GetHTTPStatusCode returns the appropriate HTTP status code
func (e *AppError) GetHTTPStatusCode() int {
	if e.StatusCode != 0 {
		return e.StatusCode
	}
	
	switch e.Type {
	case ErrTypeValidation:
		return http.StatusBadRequest
	case ErrTypeAuth:
		return http.StatusUnauthorized
	case ErrTypeNotFound:
		return http.StatusNotFound
	case ErrTypeConflict:
		return http.StatusConflict
	case ErrTypeRateLimit:
		return http.StatusTooManyRequests
	case ErrTypeTimeout:
		return http.StatusRequestTimeout
	case ErrTypeExternal, ErrTypeDatabase, ErrTypeNetwork:
		return http.StatusBadGateway
	default:
		return http.StatusInternalServerError
	}
}

// Error constructors for common error types

// NewValidationError creates a validation error
func NewValidationError(code, message string, cause error) *AppError {
	return &AppError{
		Type:       ErrTypeValidation,
		Code:       code,
		Message:    message,
		Cause:      cause,
		StatusCode: http.StatusBadRequest,
		Retryable:  false,
	}
}

// NewExternalServiceError creates an external service error
func NewExternalServiceError(code, message string, cause error) *AppError {
	return &AppError{
		Type:       ErrTypeExternal,
		Code:       code,
		Message:    message,
		Cause:      cause,
		StatusCode: http.StatusBadGateway,
		Retryable:  true,
	}
}

// NewDatabaseError creates a database error
func NewDatabaseError(code, message string, cause error) *AppError {
	return &AppError{
		Type:       ErrTypeDatabase,
		Code:       code,
		Message:    message,
		Cause:      cause,
		StatusCode: http.StatusInternalServerError,
		Retryable:  true,
	}
}

// NewInternalError creates an internal error
func NewInternalError(code, message string, cause error) *AppError {
	return &AppError{
		Type:       ErrTypeInternal,
		Code:       code,
		Message:    message,
		Cause:      cause,
		StatusCode: http.StatusInternalServerError,
		Retryable:  false,
	}
}

// NewNetworkError creates a network error
func NewNetworkError(code, message string, cause error) *AppError {
	return &AppError{
		Type:       ErrTypeNetwork,
		Code:       code,
		Message:    message,
		Cause:      cause,
		StatusCode: http.StatusBadGateway,
		Retryable:  true,
	}
}

// NewTimeoutError creates a timeout error
func NewTimeoutError(code, message string, cause error) *AppError {
	return &AppError{
		Type:       ErrTypeTimeout,
		Code:       code,
		Message:    message,
		Cause:      cause,
		StatusCode: http.StatusRequestTimeout,
		Retryable:  true,
	}
}

// NewRateLimitError creates a rate limit error
func NewRateLimitError(code, message string, cause error) *AppError {
	return &AppError{
		Type:       ErrTypeRateLimit,
		Code:       code,
		Message:    message,
		Cause:      cause,
		StatusCode: http.StatusTooManyRequests,
		Retryable:  true,
	}
}

// NewAuthError creates an authentication error
func NewAuthError(code, message string, cause error) *AppError {
	return &AppError{
		Type:       ErrTypeAuth,
		Code:       code,
		Message:    message,
		Cause:      cause,
		StatusCode: http.StatusUnauthorized,
		Retryable:  false,
	}
}

// NewNotFoundError creates a not found error
func NewNotFoundError(code, message string, cause error) *AppError {
	return &AppError{
		Type:       ErrTypeNotFound,
		Code:       code,
		Message:    message,
		Cause:      cause,
		StatusCode: http.StatusNotFound,
		Retryable:  false,
	}
}

// NewConflictError creates a conflict error
func NewConflictError(code, message string, cause error) *AppError {
	return &AppError{
		Type:       ErrTypeConflict,
		Code:       code,
		Message:    message,
		Cause:      cause,
		StatusCode: http.StatusConflict,
		Retryable:  false,
	}
}

// Predefined error codes
const (
	// Validation errors
	ErrCodeInvalidInput     = "INVALID_INPUT"
	ErrCodeMissingField     = "MISSING_FIELD"
	ErrCodeInvalidFormat    = "INVALID_FORMAT"
	ErrCodeInvalidRange     = "INVALID_RANGE"
	
	// External service errors
	ErrCodeLLMServiceFailed      = "LLM_SERVICE_FAILED"
	ErrCodeEmbeddingServiceFailed = "EMBEDDING_SERVICE_FAILED"
	ErrCodeSupabaseAPIFailed     = "SUPABASE_API_FAILED"
	
	// Database errors
	ErrCodeDatabaseConnection = "DATABASE_CONNECTION_FAILED"
	ErrCodeDatabaseQuery      = "DATABASE_QUERY_FAILED"
	ErrCodeDatabaseConstraint = "DATABASE_CONSTRAINT_VIOLATION"
	
	// Internal errors
	ErrCodeConfigurationError = "CONFIGURATION_ERROR"
	ErrCodeSerializationError = "SERIALIZATION_ERROR"
	ErrCodeProcessingError    = "PROCESSING_ERROR"
	
	// Network errors
	ErrCodeNetworkTimeout     = "NETWORK_TIMEOUT"
	ErrCodeNetworkUnavailable = "NETWORK_UNAVAILABLE"
	ErrCodeNetworkConnection  = "NETWORK_CONNECTION_FAILED"
	
	// Resource errors
	ErrCodeResourceNotFound = "RESOURCE_NOT_FOUND"
	ErrCodeResourceConflict = "RESOURCE_CONFLICT"
	ErrCodeResourceLocked   = "RESOURCE_LOCKED"
	
	// Authentication errors
	ErrCodeInvalidCredentials = "INVALID_CREDENTIALS"
	ErrCodeTokenExpired       = "TOKEN_EXPIRED"
	ErrCodeAccessDenied       = "ACCESS_DENIED"
)

// IsAppError checks if an error is an AppError
func IsAppError(err error) bool {
	_, ok := err.(*AppError)
	return ok
}

// AsAppError converts an error to AppError if possible
func AsAppError(err error) (*AppError, bool) {
	if appErr, ok := err.(*AppError); ok {
		return appErr, true
	}
	return nil, false
}

// WrapError wraps an existing error as an AppError
func WrapError(err error, errType ErrorType, code, message string) *AppError {
	if err == nil {
		return nil
	}
	
	// If it's already an AppError, preserve the original type unless explicitly overridden
	if appErr, ok := err.(*AppError); ok {
		return &AppError{
			Type:       errType,
			Code:       code,
			Message:    message,
			Cause:      appErr,
			Retryable:  appErr.Retryable,
		}
	}
	
	return &AppError{
		Type:      errType,
		Code:      code,
		Message:   message,
		Cause:     err,
		Retryable: isRetryableByDefault(errType),
	}
}

// isRetryableByDefault determines default retryability based on error type
func isRetryableByDefault(errType ErrorType) bool {
	switch errType {
	case ErrTypeExternal, ErrTypeDatabase, ErrTypeNetwork, ErrTypeTimeout, ErrTypeRateLimit:
		return true
	default:
		return false
	}
}

// IsRetryable checks if an error should be retried
func IsRetryable(err error) bool {
	if appErr, ok := AsAppError(err); ok {
		return appErr.IsRetryable()
	}
	
	// Check for context errors
	if err == context.Canceled || err == context.DeadlineExceeded {
		return false
	}
	
	// Default to non-retryable for unknown errors
	return false
}