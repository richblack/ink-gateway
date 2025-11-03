package errors

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAppError_Error(t *testing.T) {
	tests := []struct {
		name     string
		appError *AppError
		expected string
	}{
		{
			name: "error without cause",
			appError: &AppError{
				Code:    "TEST_ERROR",
				Message: "Test error message",
			},
			expected: "TEST_ERROR: Test error message",
		},
		{
			name: "error with cause",
			appError: &AppError{
				Code:    "TEST_ERROR",
				Message: "Test error message",
				Cause:   fmt.Errorf("underlying error"),
			},
			expected: "TEST_ERROR: Test error message (caused by: underlying error)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.appError.Error())
		})
	}
}

func TestAppError_GetHTTPStatusCode(t *testing.T) {
	tests := []struct {
		name       string
		appError   *AppError
		expected   int
	}{
		{
			name: "validation error",
			appError: &AppError{Type: ErrTypeValidation},
			expected: http.StatusBadRequest,
		},
		{
			name: "auth error",
			appError: &AppError{Type: ErrTypeAuth},
			expected: http.StatusUnauthorized,
		},
		{
			name: "not found error",
			appError: &AppError{Type: ErrTypeNotFound},
			expected: http.StatusNotFound,
		},
		{
			name: "conflict error",
			appError: &AppError{Type: ErrTypeConflict},
			expected: http.StatusConflict,
		},
		{
			name: "rate limit error",
			appError: &AppError{Type: ErrTypeRateLimit},
			expected: http.StatusTooManyRequests,
		},
		{
			name: "timeout error",
			appError: &AppError{Type: ErrTypeTimeout},
			expected: http.StatusRequestTimeout,
		},
		{
			name: "external service error",
			appError: &AppError{Type: ErrTypeExternal},
			expected: http.StatusBadGateway,
		},
		{
			name: "database error",
			appError: &AppError{Type: ErrTypeDatabase},
			expected: http.StatusBadGateway,
		},
		{
			name: "network error",
			appError: &AppError{Type: ErrTypeNetwork},
			expected: http.StatusBadGateway,
		},
		{
			name: "internal error",
			appError: &AppError{Type: ErrTypeInternal},
			expected: http.StatusInternalServerError,
		},
		{
			name: "custom status code",
			appError: &AppError{
				Type:       ErrTypeValidation,
				StatusCode: http.StatusTeapot,
			},
			expected: http.StatusTeapot,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.appError.GetHTTPStatusCode())
		})
	}
}

func TestErrorConstructors(t *testing.T) {
	cause := fmt.Errorf("underlying error")

	tests := []struct {
		name        string
		constructor func() *AppError
		expectedType ErrorType
		expectedRetryable bool
		expectedStatusCode int
	}{
		{
			name: "validation error",
			constructor: func() *AppError {
				return NewValidationError("TEST_CODE", "test message", cause)
			},
			expectedType: ErrTypeValidation,
			expectedRetryable: false,
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name: "external service error",
			constructor: func() *AppError {
				return NewExternalServiceError("TEST_CODE", "test message", cause)
			},
			expectedType: ErrTypeExternal,
			expectedRetryable: true,
			expectedStatusCode: http.StatusBadGateway,
		},
		{
			name: "database error",
			constructor: func() *AppError {
				return NewDatabaseError("TEST_CODE", "test message", cause)
			},
			expectedType: ErrTypeDatabase,
			expectedRetryable: true,
			expectedStatusCode: http.StatusInternalServerError,
		},
		{
			name: "internal error",
			constructor: func() *AppError {
				return NewInternalError("TEST_CODE", "test message", cause)
			},
			expectedType: ErrTypeInternal,
			expectedRetryable: false,
			expectedStatusCode: http.StatusInternalServerError,
		},
		{
			name: "network error",
			constructor: func() *AppError {
				return NewNetworkError("TEST_CODE", "test message", cause)
			},
			expectedType: ErrTypeNetwork,
			expectedRetryable: true,
			expectedStatusCode: http.StatusBadGateway,
		},
		{
			name: "timeout error",
			constructor: func() *AppError {
				return NewTimeoutError("TEST_CODE", "test message", cause)
			},
			expectedType: ErrTypeTimeout,
			expectedRetryable: true,
			expectedStatusCode: http.StatusRequestTimeout,
		},
		{
			name: "rate limit error",
			constructor: func() *AppError {
				return NewRateLimitError("TEST_CODE", "test message", cause)
			},
			expectedType: ErrTypeRateLimit,
			expectedRetryable: true,
			expectedStatusCode: http.StatusTooManyRequests,
		},
		{
			name: "auth error",
			constructor: func() *AppError {
				return NewAuthError("TEST_CODE", "test message", cause)
			},
			expectedType: ErrTypeAuth,
			expectedRetryable: false,
			expectedStatusCode: http.StatusUnauthorized,
		},
		{
			name: "not found error",
			constructor: func() *AppError {
				return NewNotFoundError("TEST_CODE", "test message", cause)
			},
			expectedType: ErrTypeNotFound,
			expectedRetryable: false,
			expectedStatusCode: http.StatusNotFound,
		},
		{
			name: "conflict error",
			constructor: func() *AppError {
				return NewConflictError("TEST_CODE", "test message", cause)
			},
			expectedType: ErrTypeConflict,
			expectedRetryable: false,
			expectedStatusCode: http.StatusConflict,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.constructor()
			
			assert.Equal(t, tt.expectedType, err.Type)
			assert.Equal(t, "TEST_CODE", err.Code)
			assert.Equal(t, "test message", err.Message)
			assert.Equal(t, cause, err.Cause)
			assert.Equal(t, tt.expectedRetryable, err.Retryable)
			assert.Equal(t, tt.expectedStatusCode, err.StatusCode)
		})
	}
}

func TestIsAppError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "app error",
			err:      NewValidationError("TEST", "test", nil),
			expected: true,
		},
		{
			name:     "regular error",
			err:      fmt.Errorf("regular error"),
			expected: false,
		},
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, IsAppError(tt.err))
		})
	}
}

func TestAsAppError(t *testing.T) {
	appErr := NewValidationError("TEST", "test", nil)
	regularErr := fmt.Errorf("regular error")

	tests := []struct {
		name        string
		err         error
		expectedErr *AppError
		expectedOk  bool
	}{
		{
			name:        "app error",
			err:         appErr,
			expectedErr: appErr,
			expectedOk:  true,
		},
		{
			name:        "regular error",
			err:         regularErr,
			expectedErr: nil,
			expectedOk:  false,
		},
		{
			name:        "nil error",
			err:         nil,
			expectedErr: nil,
			expectedOk:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err, ok := AsAppError(tt.err)
			assert.Equal(t, tt.expectedOk, ok)
			assert.Equal(t, tt.expectedErr, err)
		})
	}
}

func TestWrapError(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		errType      ErrorType
		code         string
		message      string
		expectedType ErrorType
		expectedCode string
	}{
		{
			name:         "wrap regular error",
			err:          fmt.Errorf("regular error"),
			errType:      ErrTypeValidation,
			code:         "TEST_CODE",
			message:      "test message",
			expectedType: ErrTypeValidation,
			expectedCode: "TEST_CODE",
		},
		{
			name:         "wrap app error",
			err:          NewDatabaseError("DB_ERROR", "database failed", nil),
			errType:      ErrTypeExternal,
			code:         "EXTERNAL_ERROR",
			message:      "external service failed",
			expectedType: ErrTypeExternal,
			expectedCode: "EXTERNAL_ERROR",
		},
		{
			name:         "wrap nil error",
			err:          nil,
			errType:      ErrTypeValidation,
			code:         "TEST_CODE",
			message:      "test message",
			expectedType: "",
			expectedCode: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := WrapError(tt.err, tt.errType, tt.code, tt.message)
			
			if tt.err == nil {
				assert.Nil(t, result)
				return
			}
			
			require.NotNil(t, result)
			assert.Equal(t, tt.expectedType, result.Type)
			assert.Equal(t, tt.expectedCode, result.Code)
			assert.Equal(t, tt.message, result.Message)
			assert.Equal(t, tt.err, result.Cause)
		})
	}
}

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "retryable app error",
			err:      NewExternalServiceError("TEST", "test", nil),
			expected: true,
		},
		{
			name:     "non-retryable app error",
			err:      NewValidationError("TEST", "test", nil),
			expected: false,
		},
		{
			name:     "context canceled",
			err:      context.Canceled,
			expected: false,
		},
		{
			name:     "context deadline exceeded",
			err:      context.DeadlineExceeded,
			expected: false,
		},
		{
			name:     "regular error",
			err:      fmt.Errorf("regular error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, IsRetryable(tt.err))
		})
	}
}

func TestAppError_Unwrap(t *testing.T) {
	cause := fmt.Errorf("underlying error")
	appErr := NewValidationError("TEST", "test", cause)
	
	assert.Equal(t, cause, appErr.Unwrap())
}

func TestAppError_IsRetryable(t *testing.T) {
	retryableErr := NewExternalServiceError("TEST", "test", nil)
	nonRetryableErr := NewValidationError("TEST", "test", nil)
	
	assert.True(t, retryableErr.IsRetryable())
	assert.False(t, nonRetryableErr.IsRetryable())
}