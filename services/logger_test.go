package services

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStructuredLogger_LogLevels(t *testing.T) {
	var buf bytes.Buffer
	logger := NewStructuredLogger(LogLevelInfo, &buf)
	
	// Debug should not be logged (below info level)
	logger.Debug("debug message")
	assert.Empty(t, buf.String())
	
	// Info should be logged
	buf.Reset()
	logger.Info("info message")
	assert.NotEmpty(t, buf.String())
	
	var entry LogEntry
	err := json.Unmarshal(buf.Bytes(), &entry)
	require.NoError(t, err)
	assert.Equal(t, LogLevelInfo, entry.Level)
	assert.Equal(t, "info message", entry.Message)
	
	// Warn should be logged
	buf.Reset()
	logger.Warn("warn message")
	assert.NotEmpty(t, buf.String())
	
	err = json.Unmarshal(buf.Bytes(), &entry)
	require.NoError(t, err)
	assert.Equal(t, LogLevelWarn, entry.Level)
	assert.Equal(t, "warn message", entry.Message)
	
	// Error should be logged
	buf.Reset()
	testErr := errors.New("test error")
	logger.Error("error message", testErr)
	assert.NotEmpty(t, buf.String())
	
	err = json.Unmarshal(buf.Bytes(), &entry)
	require.NoError(t, err)
	assert.Equal(t, LogLevelError, entry.Level)
	assert.Equal(t, "error message", entry.Message)
	assert.Equal(t, "test error", entry.Error)
}

func TestStructuredLogger_WithFields(t *testing.T) {
	var buf bytes.Buffer
	logger := NewStructuredLogger(LogLevelInfo, &buf)
	
	logger.Info("test message", 
		String("user_id", "123"),
		Int("count", 42),
		Bool("active", true))
	
	var entry LogEntry
	err := json.Unmarshal(buf.Bytes(), &entry)
	require.NoError(t, err)
	
	assert.Equal(t, "test message", entry.Message)
	assert.Equal(t, "123", entry.Fields["user_id"])
	assert.Equal(t, float64(42), entry.Fields["count"]) // JSON unmarshals numbers as float64
	assert.Equal(t, true, entry.Fields["active"])
}

func TestStructuredLogger_With(t *testing.T) {
	var buf bytes.Buffer
	logger := NewStructuredLogger(LogLevelInfo, &buf)
	
	// Create logger with base fields
	contextLogger := logger.With(
		String("service", "test-service"),
		String("version", "1.0.0"))
	
	contextLogger.Info("test message", String("request_id", "req-123"))
	
	var entry LogEntry
	err := json.Unmarshal(buf.Bytes(), &entry)
	require.NoError(t, err)
	
	assert.Equal(t, "test message", entry.Message)
	assert.Equal(t, "test-service", entry.Fields["service"])
	assert.Equal(t, "1.0.0", entry.Fields["version"])
	assert.Equal(t, "req-123", entry.Fields["request_id"])
}

func TestStructuredLogger_DebugLevel(t *testing.T) {
	var buf bytes.Buffer
	logger := NewStructuredLogger(LogLevelDebug, &buf)
	
	// Debug should now be logged
	logger.Debug("debug message", String("debug_info", "test"))
	assert.NotEmpty(t, buf.String())
	
	var entry LogEntry
	err := json.Unmarshal(buf.Bytes(), &entry)
	require.NoError(t, err)
	assert.Equal(t, LogLevelDebug, entry.Level)
	assert.Equal(t, "debug message", entry.Message)
	assert.Equal(t, "test", entry.Fields["debug_info"])
}

func TestLogFieldHelpers(t *testing.T) {
	var buf bytes.Buffer
	logger := NewStructuredLogger(LogLevelInfo, &buf)
	
	logger.Info("field test",
		String("string_field", "test"),
		Int("int_field", 123),
		Int64("int64_field", 456),
		Float64("float_field", 3.14),
		Bool("bool_field", true),
		Duration("duration_field", 5*1000000000), // 5 seconds in nanoseconds
		Any("any_field", map[string]string{"key": "value"}))
	
	var entry LogEntry
	err := json.Unmarshal(buf.Bytes(), &entry)
	require.NoError(t, err)
	
	assert.Equal(t, "test", entry.Fields["string_field"])
	assert.Equal(t, float64(123), entry.Fields["int_field"])
	assert.Equal(t, float64(456), entry.Fields["int64_field"])
	assert.Equal(t, 3.14, entry.Fields["float_field"])
	assert.Equal(t, true, entry.Fields["bool_field"])
	assert.Equal(t, "5s", entry.Fields["duration_field"])
	
	anyField, ok := entry.Fields["any_field"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "value", anyField["key"])
}

func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected LogLevel
	}{
		{"debug", LogLevelDebug},
		{"info", LogLevelInfo},
		{"warn", LogLevelWarn},
		{"warning", LogLevelWarn},
		{"error", LogLevelError},
		{"invalid", LogLevelInfo}, // defaults to info
		{"", LogLevelInfo},        // defaults to info
	}
	
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ParseLogLevel(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNewLoggerFromConfig(t *testing.T) {
	var buf bytes.Buffer
	
	config := &LoggerConfig{
		Level:  LogLevelWarn,
		Format: "json",
		Output: &buf,
	}
	
	logger := NewLoggerFromConfig(config)
	
	// Info should not be logged (below warn level)
	logger.Info("info message")
	assert.Empty(t, buf.String())
	
	// Warn should be logged
	logger.Warn("warn message")
	assert.NotEmpty(t, buf.String())
	assert.True(t, strings.Contains(buf.String(), "warn message"))
}

func TestNewLoggerFromConfig_Defaults(t *testing.T) {
	// Test with nil config
	logger := NewLoggerFromConfig(nil)
	assert.NotNil(t, logger)
	
	// Test with empty config
	logger = NewLoggerFromConfig(&LoggerConfig{})
	assert.NotNil(t, logger)
}

func TestStructuredLogger_EmptyFields(t *testing.T) {
	var buf bytes.Buffer
	logger := NewStructuredLogger(LogLevelInfo, &buf)
	
	// Log without fields
	logger.Info("message without fields")
	
	var entry LogEntry
	err := json.Unmarshal(buf.Bytes(), &entry)
	require.NoError(t, err)
	
	assert.Equal(t, "message without fields", entry.Message)
	assert.Nil(t, entry.Fields) // Should be nil when empty
}

func BenchmarkStructuredLogger_Info(b *testing.B) {
	var buf bytes.Buffer
	logger := NewStructuredLogger(LogLevelInfo, &buf)
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		logger.Info("benchmark message",
			String("request_id", "req-123"),
			Int("iteration", i),
			Bool("success", true))
		buf.Reset() // Reset buffer to avoid memory growth
	}
}

func BenchmarkStructuredLogger_Debug_Filtered(b *testing.B) {
	var buf bytes.Buffer
	logger := NewStructuredLogger(LogLevelInfo, &buf) // Debug messages will be filtered out
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		logger.Debug("debug message",
			String("request_id", "req-123"),
			Int("iteration", i))
	}
}