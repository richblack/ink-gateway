package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
)

// PostgresConfig holds PostgreSQL connection configuration
type PostgresConfig struct {
	Host     string
	Port     int
	Database string
	User     string
	Password string
	SSLMode  string

	// Connection pool settings
	MaxConns     int32
	MinConns     int32
	MaxConnLife  time.Duration
	MaxConnIdle  time.Duration
	HealthCheck  time.Duration
}

// DefaultPostgresConfig returns sensible defaults
func DefaultPostgresConfig() *PostgresConfig {
	return &PostgresConfig{
		Host:         "localhost",
		Port:         5432,
		Database:     "postgres",
		User:         "postgres",
		Password:     "",
		SSLMode:      "prefer",
		MaxConns:     10,
		MinConns:     2,
		MaxConnLife:  time.Hour,
		MaxConnIdle:  30 * time.Minute,
		HealthCheck:  time.Minute,
	}
}

// BuildConnectionString builds PostgreSQL connection string
func (c *PostgresConfig) BuildConnectionString() string {
	return fmt.Sprintf(
		"host=%s port=%d dbname=%s user=%s password=%s sslmode=%s pool_max_conns=%d pool_min_conns=%d search_path=public",
		c.Host, c.Port, c.Database, c.User, c.Password, c.SSLMode,
		c.MaxConns, c.MinConns,
	)
}

// PostgresService provides PostgreSQL database operations
type PostgresService struct {
	pool *pgxpool.Pool
	cfg  *PostgresConfig
}

// NewPostgresService creates a new PostgreSQL service with connection pooling
func NewPostgresService(cfg *PostgresConfig) (*PostgresService, error) {
	if cfg == nil {
		cfg = DefaultPostgresConfig()
	}

	// Build pool config
	poolConfig, err := pgxpool.ParseConfig(cfg.BuildConnectionString())
	if err != nil {
		return nil, fmt.Errorf("failed to parse connection string: %w", err)
	}

		// Configure connection pool
	poolConfig.MaxConns = cfg.MaxConns
	poolConfig.MinConns = cfg.MinConns
	poolConfig.MaxConnLifetime = cfg.MaxConnLife
	poolConfig.MaxConnIdleTime = cfg.MaxConnIdle
	if cfg.HealthCheck > 0 {
		poolConfig.HealthCheckPeriod = cfg.HealthCheck
	}

	// Set AfterConnect to ensure search_path is set for each connection
	poolConfig.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		_, err := conn.Exec(ctx, "SET search_path TO public")
		return err
	}

	// Create connection pool
	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &PostgresService{
		pool: pool,
		cfg:  cfg,
	}, nil
}

// Close closes the connection pool
func (s *PostgresService) Close() {
	if s.pool != nil {
		s.pool.Close()
	}
}

// Pool returns the underlying connection pool
func (s *PostgresService) Pool() *pgxpool.Pool {
	return s.pool
}

// StdlibDB returns a database/sql.DB interface for compatibility
// with code that uses database/sql instead of pgx directly
func (s *PostgresService) StdlibDB() (*sql.DB, error) {
	// Build connection string without pool parameters (which are pgxpool-specific)
	connStr := fmt.Sprintf(
		"host=%s port=%d dbname=%s user=%s password=%s sslmode=%s search_path=public",
		s.cfg.Host, s.cfg.Port, s.cfg.Database, s.cfg.User, s.cfg.Password, s.cfg.SSLMode,
	)

	connConfig, err := pgx.ParseConfig(connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse connection config: %w", err)
	}

	return stdlib.OpenDB(*connConfig), nil
}

// Ping checks database connectivity
func (s *PostgresService) Ping(ctx context.Context) error {
	return s.pool.Ping(ctx)
}

// Begin starts a new transaction
func (s *PostgresService) Begin(ctx context.Context) (pgx.Tx, error) {
	return s.pool.Begin(ctx)
}

// QueryRow executes a query that returns at most one row
func (s *PostgresService) QueryRow(ctx context.Context, query string, args ...interface{}) pgx.Row {
	return s.pool.QueryRow(ctx, query, args...)
}

// Query executes a query that returns rows
func (s *PostgresService) Query(ctx context.Context, query string, args ...interface{}) (pgx.Rows, error) {
	return s.pool.Query(ctx, query, args...)
}

// Exec executes a query without returning rows
func (s *PostgresService) Exec(ctx context.Context, query string, args ...interface{}) (pgconn.CommandTag, error) {
	return s.pool.Exec(ctx, query, args...)
}

// Stats returns connection pool statistics
func (s *PostgresService) Stats() *pgxpool.Stat {
	return s.pool.Stat()
}

// Health checks database health
func (s *PostgresService) Health(ctx context.Context) error {
	// Check connection
	if err := s.Ping(ctx); err != nil {
		return fmt.Errorf("ping failed: %w", err)
	}

	// Check pool stats
	stats := s.Stats()
	if stats.AcquireCount() == 0 && stats.TotalConns() == 0 {
		return fmt.Errorf("no active connections")
	}

	// Execute a simple query
	var result int
	err := s.QueryRow(ctx, "SELECT 1").Scan(&result)
	if err != nil {
		return fmt.Errorf("query failed: %w", err)
	}

	return nil
}
