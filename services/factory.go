package services

import (
	"context"
	"fmt"
	"semantic-text-processor/clients"
	"semantic-text-processor/config"
	"semantic-text-processor/database"
)

// ServiceContainer holds all service instances
type ServiceContainer struct {
	// Core services
	TextProcessor      TextProcessor
	LLMService         LLMService
	EmbeddingService   EmbeddingService
	SearchService      SearchService
	TemplateService    TemplateService
	TagService         TagService
	UnifiedChunkService UnifiedChunkService

	// Database
	PostgresService *database.PostgresService

	// Clients (deprecated - use PostgresService)
	SupabaseClient SupabaseClient

	// Performance and monitoring
	CacheService   CacheService
	MetricsService MetricsService
	Logger         Logger
	HealthService  HealthService
}

// ServiceFactory creates and configures all services
type ServiceFactory struct {
	config *config.Config
}

// NewServiceFactory creates a new service factory
func NewServiceFactory(cfg *config.Config) *ServiceFactory {
	return &ServiceFactory{
		config: cfg,
	}
}

// CreateServices creates and wires all services together
func (f *ServiceFactory) CreateServices() (*ServiceContainer, error) {
	// Create logger
	logLevel := ParseLogLevel(f.config.Logging.Level)
	logger := NewStructuredLogger(logLevel, nil)
	
	// Create performance and monitoring services
	var cacheService CacheService
	var metricsService MetricsService
	
	if f.config.Cache.Enabled {
		cacheService = NewInMemoryCache(
			f.config.Cache.MaxSize,
			f.config.Cache.CleanupInterval,
		)
	}
	
	if f.config.Performance.MetricsEnabled {
		metricsService = NewInMemoryMetrics()
	}
	
	// Create health service
	healthService := NewHealthService("1.0.0", logger)

	// Create PostgreSQL service
	pgConfig := &database.PostgresConfig{
		Host:     f.config.Database.Host,
		Port:     int(f.config.Database.Port),
		Database: f.config.Database.Database,
		User:     f.config.Database.User,
		Password: f.config.Database.Password,
		SSLMode:  f.config.Database.SSLMode,
		MaxConns: int32(f.config.Database.MaxConns),
		MinConns: int32(f.config.Database.MinConns),
	}

	postgresService, err := database.NewPostgresService(pgConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create PostgreSQL service: %w", err)
	}

	// Create Supabase client (deprecated)
	supabaseClient := clients.NewSupabaseClient(&f.config.Supabase)
	
	// Wrap with caching if enabled
	var wrappedSupabaseClient SupabaseClient = supabaseClient
	// TODO: Implement NewCachedSupabaseClient when needed
	// if cacheService != nil {
	// 	cacheConfig := &CacheConfig{
	// 		MaxSize:         f.config.Cache.MaxSize,
	// 		CleanupInterval: f.config.Cache.CleanupInterval,
	// 		DefaultTTL:      f.config.Cache.DefaultTTL,
	// 		Enabled:         f.config.Cache.Enabled,
	// 	}
	// 	wrappedSupabaseClient = NewCachedSupabaseClient(supabaseClient, cacheService, cacheConfig)
	// }
	
	// Create external service clients
	llmService := NewLLMClient(&f.config.LLM)
	embeddingService := NewEmbeddingService(&f.config.Embedding)
	
	// Create core services with dependencies
	textProcessor := NewTextProcessor(llmService, embeddingService)
	searchService := NewSearchService(wrappedSupabaseClient, embeddingService)
	templateService := NewTemplateService(wrappedSupabaseClient)
	tagService := NewTagService(wrappedSupabaseClient)

	// Create unified chunk service with PostgreSQL
	// Use no-op monitor for now
	monitor := NewNoOpMonitor()
	stdlibDB, err := postgresService.StdlibDB()
	if err != nil {
		return nil, fmt.Errorf("failed to get stdlib DB: %w", err)
	}
	unifiedChunkService := NewUnifiedChunkService(stdlibDB, cacheService, monitor)
	
	// TODO: Implement NewCachedSearchService when needed
	// Wrap search service with caching and monitoring
	// if cacheService != nil {
	// 	cacheConfig := &CacheConfig{
	// 		MaxSize:         f.config.Cache.MaxSize,
	// 		CleanupInterval: f.config.Cache.CleanupInterval,
	// 		DefaultTTL:      f.config.Cache.DefaultTTL,
	// 		Enabled:         f.config.Cache.Enabled,
	// 	}
	// 	searchService = NewCachedSearchService(searchService, cacheService, cacheConfig)
	// }
	
	if metricsService != nil {
		monitor := NewPerformanceMonitor(metricsService)
		searchService = NewMonitoredSearchService(searchService, monitor)
	}
	
	// Register health checkers
	if wrappedSupabaseClient != nil {
		healthService.RegisterChecker(NewDatabaseHealthChecker("database", wrappedSupabaseClient))
	}
	if cacheService != nil {
		healthService.RegisterChecker(NewCacheHealthChecker("cache", cacheService))
	}
	if metricsService != nil {
		healthService.RegisterChecker(NewMetricsHealthChecker("metrics", metricsService))
	}
	
	container := &ServiceContainer{
		TextProcessor:       textProcessor,
		LLMService:          llmService,
		EmbeddingService:    embeddingService,
		SearchService:       searchService,
		TemplateService:     templateService,
		TagService:          tagService,
		UnifiedChunkService: unifiedChunkService,
		PostgresService:     postgresService,
		SupabaseClient:      wrappedSupabaseClient,
		CacheService:        cacheService,
		MetricsService:      metricsService,
		Logger:              logger,
		HealthService:       healthService,
	}
	
	return container, nil
}

// HealthCheck verifies all services are healthy
func (c *ServiceContainer) HealthCheck() error {
	// Check Supabase connection
	if err := c.SupabaseClient.HealthCheck(context.Background()); err != nil {
		return fmt.Errorf("supabase client health check failed: %w", err)
	}
	
	// Additional health checks can be added here for other services
	// For now, we only check the critical Supabase connection
	
	return nil
}