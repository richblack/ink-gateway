package services

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"time"
)

// LogLevel represents the severity level of a log entry
type LogLevel string

const (
	LogLevelDebug LogLevel = "debug"
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
)

// LogField represents a structured log field
type LogField struct {
	Key   string      `json:"key"`
	Value interface{} `json:"value"`
}

// LogEntry represents a structured log entry
type LogEntry struct {
	Timestamp string                 `json:"timestamp"`
	Level     LogLevel               `json:"level"`
	Message   string                 `json:"message"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
	Error     string                 `json:"error,omitempty"`
}

// Logger interface for structured logging
type Logger interface {
	Debug(msg string, fields ...LogField)
	Info(msg string, fields ...LogField)
	Warn(msg string, fields ...LogField)
	Error(msg string, err error, fields ...LogField)
	With(fields ...LogField) Logger
}

// StructuredLogger implements Logger interface
type StructuredLogger struct {
	level      LogLevel
	output     io.Writer
	baseFields map[string]interface{}
}

// NewStructuredLogger creates a new structured logger
func NewStructuredLogger(level LogLevel, output io.Writer) *StructuredLogger {
	if output == nil {
		output = os.Stdout
	}
	
	return &StructuredLogger{
		level:      level,
		output:     output,
		baseFields: make(map[string]interface{}),
	}
}

// NewDefaultLogger creates a logger with default settings
func NewDefaultLogger() *StructuredLogger {
	return NewStructuredLogger(LogLevelInfo, os.Stdout)
}

// Debug logs a debug message
func (l *StructuredLogger) Debug(msg string, fields ...LogField) {
	if l.shouldLog(LogLevelDebug) {
		l.log(LogLevelDebug, msg, nil, fields...)
	}
}

// Info logs an info message
func (l *StructuredLogger) Info(msg string, fields ...LogField) {
	if l.shouldLog(LogLevelInfo) {
		l.log(LogLevelInfo, msg, nil, fields...)
	}
}

// Warn logs a warning message
func (l *StructuredLogger) Warn(msg string, fields ...LogField) {
	if l.shouldLog(LogLevelWarn) {
		l.log(LogLevelWarn, msg, nil, fields...)
	}
}

// Error logs an error message
func (l *StructuredLogger) Error(msg string, err error, fields ...LogField) {
	if l.shouldLog(LogLevelError) {
		l.log(LogLevelError, msg, err, fields...)
	}
}

// With creates a new logger with additional base fields
func (l *StructuredLogger) With(fields ...LogField) Logger {
	newFields := make(map[string]interface{})
	
	// Copy existing base fields
	for k, v := range l.baseFields {
		newFields[k] = v
	}
	
	// Add new fields
	for _, field := range fields {
		newFields[field.Key] = field.Value
	}
	
	return &StructuredLogger{
		level:      l.level,
		output:     l.output,
		baseFields: newFields,
	}
}

// log writes a log entry
func (l *StructuredLogger) log(level LogLevel, msg string, err error, fields ...LogField) {
	entry := LogEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		Level:     level,
		Message:   msg,
		Fields:    make(map[string]interface{}),
	}
	
	// Add base fields
	for k, v := range l.baseFields {
		entry.Fields[k] = v
	}
	
	// Add provided fields
	for _, field := range fields {
		entry.Fields[field.Key] = field.Value
	}
	
	// Add error if provided
	if err != nil {
		entry.Error = err.Error()
	}
	
	// Remove fields if empty
	if len(entry.Fields) == 0 {
		entry.Fields = nil
	}
	
	// Marshal to JSON and write
	data, marshalErr := json.Marshal(entry)
	if marshalErr != nil {
		// Fallback to standard logging if JSON marshaling fails
		log.Printf("Failed to marshal log entry: %v", marshalErr)
		log.Printf("[%s] %s: %s", level, msg, err)
		return
	}
	
	fmt.Fprintln(l.output, string(data))
}

// shouldLog determines if a message should be logged based on level
func (l *StructuredLogger) shouldLog(level LogLevel) bool {
	levelOrder := map[LogLevel]int{
		LogLevelDebug: 0,
		LogLevelInfo:  1,
		LogLevelWarn:  2,
		LogLevelError: 3,
	}
	
	currentLevel, exists := levelOrder[l.level]
	if !exists {
		currentLevel = levelOrder[LogLevelInfo]
	}
	
	messageLevel, exists := levelOrder[level]
	if !exists {
		messageLevel = levelOrder[LogLevelInfo]
	}
	
	return messageLevel >= currentLevel
}

// Field creates a log field
func Field(key string, value interface{}) LogField {
	return LogField{Key: key, Value: value}
}

// String field helper
func String(key, value string) LogField {
	return LogField{Key: key, Value: value}
}

// Int field helper
func Int(key string, value int) LogField {
	return LogField{Key: key, Value: value}
}

// Int64 field helper
func Int64(key string, value int64) LogField {
	return LogField{Key: key, Value: value}
}

// Float64 field helper
func Float64(key string, value float64) LogField {
	return LogField{Key: key, Value: value}
}

// Bool field helper
func Bool(key string, value bool) LogField {
	return LogField{Key: key, Value: value}
}

// Duration field helper
func Duration(key string, value time.Duration) LogField {
	return LogField{Key: key, Value: value.String()}
}

// Any field helper for arbitrary values
func Any(key string, value interface{}) LogField {
	return LogField{Key: key, Value: value}
}

// LoggerConfig holds logger configuration
type LoggerConfig struct {
	Level  LogLevel
	Format string // "json" or "text"
	Output io.Writer
}

// NewLoggerFromConfig creates a logger from configuration
func NewLoggerFromConfig(config *LoggerConfig) Logger {
	if config == nil {
		return NewDefaultLogger()
	}
	
	level := config.Level
	if level == "" {
		level = LogLevelInfo
	}
	
	output := config.Output
	if output == nil {
		output = os.Stdout
	}
	
	return NewStructuredLogger(level, output)
}

// ParseLogLevel parses a log level string
func ParseLogLevel(level string) LogLevel {
	switch level {
	case "debug":
		return LogLevelDebug
	case "info":
		return LogLevelInfo
	case "warn", "warning":
		return LogLevelWarn
	case "error":
		return LogLevelError
	default:
		return LogLevelInfo
	}
}