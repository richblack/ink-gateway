package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds all configuration for the application
type Config struct {
	Server      ServerConfig
	Database    DatabaseConfig
	Supabase    SupabaseConfig // Deprecated: Use Database instead
	LLM         LLMConfig
	Embedding   EmbeddingConfig
	Logging     LoggingConfig
	Cache       CacheConfig
	Performance PerformanceConfig
	Features    FeaturesConfig
	Storage     StorageConfig
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

// DatabaseConfig holds PostgreSQL database configuration
type DatabaseConfig struct {
	Host     string
	Port     int
	Database string
	User     string
	Password string
	SSLMode  string

	// Connection pool settings
	MaxConns int
	MinConns int
}

// SupabaseConfig holds Supabase client configuration
// Deprecated: Use DatabaseConfig for direct PostgreSQL connection
type SupabaseConfig struct {
	URL    string
	APIKey string
}

// LLMConfig holds LLM service configuration
type LLMConfig struct {
	APIKey   string
	Endpoint string
	Timeout  time.Duration
}

// EmbeddingConfig holds embedding service configuration
type EmbeddingConfig struct {
	APIKey   string
	Endpoint string
	Timeout  time.Duration
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level  string
	Format string
}

// CacheConfig holds cache configuration
type CacheConfig struct {
	Enabled         bool
	MaxSize         int
	CleanupInterval time.Duration
	DefaultTTL      time.Duration
}

// PerformanceConfig holds performance monitoring configuration
type PerformanceConfig struct {
	MetricsEnabled     bool
	MetricsEndpoint    string
	MonitoringEnabled  bool
	SlowQueryThreshold time.Duration
}

// FeaturesConfig holds feature flag configuration
type FeaturesConfig struct {
	UseUnifiedHandlers bool
}

// StorageConfig holds storage configuration
type StorageConfig struct {
	Provider string // "google_drive", "local", or "both"
	
	// Google Drive configuration
	GoogleDrive GoogleDriveConfig
	
	// Local storage configuration
	Local LocalStorageConfig
}

// GoogleDriveConfig holds Google Drive storage configuration
type GoogleDriveConfig struct {
	Enabled         bool
	FolderID        string
	CredentialsPath string
	BaseURL         string
}

// LocalStorageConfig holds local storage configuration
type LocalStorageConfig struct {
	Path    string
	BaseURL string
}

// LoadConfig loads configuration from environment variables
func LoadConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Port:         getEnv("SERVER_PORT", "8080"),
			ReadTimeout:  getDurationEnv("SERVER_READ_TIMEOUT", 30*time.Second),
			WriteTimeout: getDurationEnv("SERVER_WRITE_TIMEOUT", 30*time.Second),
			IdleTimeout:  getDurationEnv("SERVER_IDLE_TIMEOUT", 60*time.Second),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getIntEnv("DB_PORT", 5432),
			Database: getEnv("DB_NAME", "postgres"),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", ""),
			SSLMode:  getEnv("DB_SSLMODE", "prefer"),
			MaxConns: getIntEnv("DB_MAX_CONNS", 10),
			MinConns: getIntEnv("DB_MIN_CONNS", 2),
		},
		Supabase: SupabaseConfig{
			URL:    getEnv("SUPABASE_URL", ""),
			APIKey: getEnv("SUPABASE_API_KEY", ""),
		},
		LLM: LLMConfig{
			APIKey:   getEnv("LLM_API_KEY", ""),
			Endpoint: getEnv("LLM_ENDPOINT", ""),
			Timeout:  getDurationEnv("LLM_TIMEOUT", 60*time.Second),
		},
		Embedding: EmbeddingConfig{
			APIKey:   getEnv("EMBEDDING_API_KEY", ""),
			Endpoint: getEnv("EMBEDDING_ENDPOINT", ""),
			Timeout:  getDurationEnv("EMBEDDING_TIMEOUT", 30*time.Second),
		},
		Logging: LoggingConfig{
			Level:  getEnv("LOG_LEVEL", "info"),
			Format: getEnv("LOG_FORMAT", "json"),
		},
		Cache: CacheConfig{
			Enabled:         getBoolEnv("CACHE_ENABLED", true),
			MaxSize:         getIntEnv("CACHE_MAX_SIZE", 1000),
			CleanupInterval: getDurationEnv("CACHE_CLEANUP_INTERVAL", 5*time.Minute),
			DefaultTTL:      getDurationEnv("CACHE_DEFAULT_TTL", 30*time.Minute),
		},
		Performance: PerformanceConfig{
			MetricsEnabled:     getBoolEnv("METRICS_ENABLED", true),
			MetricsEndpoint:    getEnv("METRICS_ENDPOINT", "/metrics"),
			MonitoringEnabled:  getBoolEnv("MONITORING_ENABLED", true),
			SlowQueryThreshold: getDurationEnv("SLOW_QUERY_THRESHOLD", 500*time.Millisecond),
		},
		Features: FeaturesConfig{
			UseUnifiedHandlers: getBoolEnv("USE_UNIFIED_HANDLERS", false),
		},
		Storage: StorageConfig{
			Provider: getEnv("STORAGE_PROVIDER", "local"),
			GoogleDrive: GoogleDriveConfig{
				Enabled:         getBoolEnv("GOOGLE_DRIVE_ENABLED", false),
				FolderID:        getEnv("GOOGLE_DRIVE_FOLDER_ID", ""),
				CredentialsPath: getEnv("GOOGLE_DRIVE_CREDENTIALS_PATH", "./config/google-drive-credentials.json"),
				BaseURL:         getEnv("GOOGLE_DRIVE_BASE_URL", "https://drive.google.com/file/d/"),
			},
			Local: LocalStorageConfig{
				Path:    getEnv("LOCAL_STORAGE_PATH", "./uploads"),
				BaseURL: getEnv("LOCAL_STORAGE_BASE_URL", "http://localhost:8081/uploads/"),
			},
		},
	}
}

// getEnv gets environment variable with default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getDurationEnv gets duration from environment variable with default value
func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

// getIntEnv gets integer from environment variable with default value
func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// getBoolEnv gets boolean from environment variable with default value
func getBoolEnv(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Supabase.URL == "" {
		return &ConfigError{Field: "SUPABASE_URL", Message: "Supabase URL is required"}
	}
	if c.Supabase.APIKey == "" {
		return &ConfigError{Field: "SUPABASE_API_KEY", Message: "Supabase API key is required"}
	}
	return nil
}

// ConfigError represents configuration validation error
type ConfigError struct {
	Field   string
	Message string
}

func (e *ConfigError) Error() string {
	return e.Field + ": " + e.Message
}