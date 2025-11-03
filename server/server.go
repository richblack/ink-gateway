package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"semantic-text-processor/config"
	"semantic-text-processor/handlers"
	"semantic-text-processor/services"
	"syscall"
	"time"

	"github.com/gorilla/mux"
)

// Server represents the HTTP server
type Server struct {
	config      *config.Config
	router      *mux.Router
	httpServer  *http.Server
	services    *services.ServiceContainer

	// Handlers
	textHandler     *handlers.TextHandler
	chunkHandler    handlers.ChunkHandlerInterface
	searchHandler   *handlers.SearchHandler
	templateHandler *handlers.TemplateHandler
	tagHandler      handlers.TagHandlerInterface
	simpleMediaHandler    *handlers.SimpleMediaHandler
	aiHandler       *handlers.AIHandler
}

// NewServer creates a new server instance
func NewServer(cfg *config.Config) *Server {
	// Create service factory and initialize services
	serviceFactory := services.NewServiceFactory(cfg)
	serviceContainer, err := serviceFactory.CreateServices()
	if err != nil {
		log.Fatalf("Failed to create services: %v", err)
	}
	
	router := mux.NewRouter()

	// Create handler factory
	slowQueryThreshold := cfg.Performance.SlowQueryThreshold
	if slowQueryThreshold == 0 {
		slowQueryThreshold = 500 * time.Millisecond // default
	}

	handlerFactory := handlers.NewHandlerFactory(
		serviceContainer.UnifiedChunkService,
		serviceContainer.SupabaseClient,
		serviceContainer.TagService,
		serviceContainer.CacheService,
		log.New(os.Stderr, "[handler] ", log.LstdFlags),
		slowQueryThreshold,
		cfg.Performance.MetricsEnabled,
		cfg.Features.UseUnifiedHandlers, // feature flag to control rollout
	)

	// Create handlers
	textHandler := handlers.NewTextHandler(serviceContainer.TextProcessor, serviceContainer.SupabaseClient)
	chunkHandler := handlerFactory.CreateChunkHandler()
	// TODO: Add multimodal search and image similarity services to ServiceContainer
	searchHandler := handlers.NewSearchHandler(nil, nil, nil)
	templateHandler := handlers.NewTemplateHandler(serviceContainer.TemplateService)
	tagHandler := handlerFactory.CreateTagHandler()
	simpleMediaHandler := handlers.NewSimpleMediaHandler(cfg)
	aiHandler := handlers.NewAIHandler()
	
	server := &Server{
		config:          cfg,
		router:          router,
		services:        serviceContainer,
		textHandler:     textHandler,
		chunkHandler:    chunkHandler,
		searchHandler:   searchHandler,
		templateHandler: templateHandler,
		tagHandler:      tagHandler,
		simpleMediaHandler:    simpleMediaHandler,
		aiHandler:       aiHandler,
		httpServer: &http.Server{
			Addr:         ":" + cfg.Server.Port,
			Handler:      router,
			ReadTimeout:  cfg.Server.ReadTimeout,
			WriteTimeout: cfg.Server.WriteTimeout,
			IdleTimeout:  cfg.Server.IdleTimeout,
		},
	}

	server.setupRoutes()
	server.setupMiddleware()
	
	return server
}

// setupRoutes configures all HTTP routes
func (s *Server) setupRoutes() {

	// API version prefix
	api := s.router.PathPrefix("/api/v1").Subrouter()

	// Health check
	api.HandleFunc("/health", s.healthCheck).Methods("GET", "OPTIONS")
	
	// Performance and monitoring endpoints
	if s.config.Performance.MetricsEnabled && s.services.MetricsService != nil {
		api.HandleFunc(s.config.Performance.MetricsEndpoint, s.metricsHandler).Methods("GET")
	}
	api.HandleFunc("/cache/stats", s.cacheStatsHandler).Methods("GET")
	api.HandleFunc("/cache/clear", s.cacheClearHandler).Methods("POST")

	// Text routes
	api.HandleFunc("/texts", s.textHandler.CreateText).Methods("POST")
	api.HandleFunc("/texts", s.textHandler.GetTexts).Methods("GET")
	api.HandleFunc("/texts/{id}", s.textHandler.GetTextByID).Methods("GET")
	api.HandleFunc("/texts/{id}", s.textHandler.UpdateText).Methods("PUT")
	api.HandleFunc("/texts/{id}", s.textHandler.DeleteText).Methods("DELETE")
	api.HandleFunc("/texts/{id}/structure", s.textHandler.GetTextStructure).Methods("GET")
	api.HandleFunc("/texts/{id}/structure", s.textHandler.UpdateTextStructure).Methods("PUT")

	// Template routes
	api.HandleFunc("/templates", s.templateHandler.CreateTemplate).Methods("POST")
	api.HandleFunc("/templates", s.templateHandler.GetAllTemplates).Methods("GET")
	api.HandleFunc("/templates/{content}", s.templateHandler.GetTemplateByContent).Methods("GET")
	api.HandleFunc("/templates/{id}/instances", s.templateHandler.CreateTemplateInstance).Methods("POST")
	api.HandleFunc("/instances/{id}/slots", s.templateHandler.UpdateSlotValue).Methods("PUT")

	// Tag routes
	api.HandleFunc("/chunks/{id}/tags", s.tagHandler.AddTag).Methods("POST")
	api.HandleFunc("/chunks/{id}/tags/{tagId}", s.tagHandler.RemoveTag).Methods("DELETE")
	api.HandleFunc("/chunks/{id}/tags", s.tagHandler.GetChunkTags).Methods("GET")
	api.HandleFunc("/tags/{content}/chunks", s.tagHandler.GetChunksByTag).Methods("GET")
	
	// Batch tag operations and advanced search (only available with unified handlers)
	if unifiedTagHandler, ok := s.tagHandler.(*handlers.UnifiedTagHandler); ok {
		api.HandleFunc("/chunks/tags/batch", unifiedTagHandler.BatchTagOperations).Methods("POST")
		api.HandleFunc("/tags/search", unifiedTagHandler.GetChunksByTags).Methods("POST")
	}

	// Legacy tag inheritance routes (only for legacy handlers)
	if legacyTagWrapper, ok := s.tagHandler.(*handlers.LegacyTagHandlerWrapper); ok {
		api.HandleFunc("/chunks/{id}/tags/inherit", func(w http.ResponseWriter, r *http.Request) {
			legacyTagWrapper.Handler.AddTagWithInheritance(w, r)
		}).Methods("POST")
		api.HandleFunc("/chunks/{id}/tags/{tagId}/inherit", func(w http.ResponseWriter, r *http.Request) {
			legacyTagWrapper.Handler.RemoveTagWithInheritance(w, r)
		}).Methods("DELETE")
		api.HandleFunc("/chunks/{id}/tags/inherited", func(w http.ResponseWriter, r *http.Request) {
			legacyTagWrapper.Handler.GetInheritedTags(w, r)
		}).Methods("GET")
	}

	// Chunk routes
	api.HandleFunc("/chunks", s.chunkHandler.GetChunks).Methods("GET")
	api.HandleFunc("/chunks", s.chunkHandler.CreateChunk).Methods("POST")
	api.HandleFunc("/chunks/{id}", s.chunkHandler.GetChunkByID).Methods("GET")
	api.HandleFunc("/chunks/{id}", s.chunkHandler.UpdateChunk).Methods("PUT")
	api.HandleFunc("/chunks/{id}", s.chunkHandler.DeleteChunk).Methods("DELETE")
	api.HandleFunc("/chunks/{id}/hierarchy", s.chunkHandler.GetChunkHierarchy).Methods("GET")
	api.HandleFunc("/chunks/{id}/children", s.chunkHandler.GetChunkChildren).Methods("GET")
	api.HandleFunc("/chunks/{id}/move", s.chunkHandler.MoveChunk).Methods("POST")

	// Batch chunk operations (only available with unified handlers)
	if unifiedHandler, ok := s.chunkHandler.(*handlers.UnifiedChunkHandler); ok {
		api.HandleFunc("/chunks/batch", unifiedHandler.BatchCreateChunks).Methods("POST")
		api.HandleFunc("/chunks/batch", unifiedHandler.BatchUpdateChunks).Methods("PUT")
	}

	// Legacy bulk update route and siblings route for backward compatibility
	if legacyWrapper, ok := s.chunkHandler.(*handlers.LegacyChunkHandlerWrapper); ok {
		api.HandleFunc("/chunks/bulk-update", func(w http.ResponseWriter, r *http.Request) {
			legacyWrapper.Handler.BulkUpdateChunks(w, r)
		}).Methods("PUT")
		api.HandleFunc("/chunks/{id}/siblings", func(w http.ResponseWriter, r *http.Request) {
			legacyWrapper.Handler.GetChunkSiblings(w, r)
		}).Methods("GET")
	}

	// Search routes
	// TODO: Update these to use new multimodal search endpoints
	// Old endpoints commented out - need to map to new multimodal search handler
	// api.HandleFunc("/search/semantic", s.searchHandler.SemanticSearch).Methods("POST")
	// api.HandleFunc("/search/graph", s.searchHandler.GraphSearch).Methods("POST")
	// api.HandleFunc("/search/tags", s.searchHandler.SearchByTag).Methods("POST")
	// api.HandleFunc("/search/chunks", s.searchHandler.SearchChunks).Methods("POST")
	// api.HandleFunc("/search/hybrid", s.searchHandler.HybridSearch).Methods("POST")

	// New multimodal search endpoints
	api.HandleFunc("/search/multimodal", s.searchHandler.MultimodalSearch).Methods("POST")
	api.HandleFunc("/search/image-similarity", s.searchHandler.SearchByImage).Methods("POST")
	api.HandleFunc("/search/slide-recommendations", s.searchHandler.RecommendImagesForSlide).Methods("POST")
	api.HandleFunc("/search/presentation-recommendations", s.searchHandler.RecommendImagesForPresentation).Methods("POST")
	api.HandleFunc("/search/duplicates", s.searchHandler.FindDuplicateImages).Methods("POST")
	api.HandleFunc("/search/similar/{chunk_id}", s.searchHandler.GetSimilarImages).Methods("GET")

	// Media routes
	api.HandleFunc("/media/upload", s.simpleMediaHandler.UploadImage).Methods("POST", "OPTIONS")
	api.HandleFunc("/media/library", s.simpleMediaHandler.GetImageLibrary).Methods("GET", "OPTIONS")

	// AI routes
	api.HandleFunc("/ai/chat", s.aiHandler.ChatWithAI).Methods("POST", "OPTIONS")
	api.HandleFunc("/ai/process", s.aiHandler.ProcessContent).Methods("POST", "OPTIONS")
	
	// Additional search routes for plugin compatibility
	api.HandleFunc("/search/tags", s.searchHandler.SearchByTags).Methods("POST")
	api.HandleFunc("/search/tags", s.corsHandler).Methods("OPTIONS")
	api.HandleFunc("/tags/search", s.searchHandler.SearchByTags).Methods("POST")
	api.HandleFunc("/tags/search", s.corsHandler).Methods("OPTIONS")
}

// setupMiddleware configures middleware
func (s *Server) setupMiddleware() {
	// CORS must be first to handle preflight requests
	s.router.Use(s.corsMiddleware)
	s.router.Use(s.loggingMiddleware)
	s.router.Use(s.contentTypeMiddleware)
	
	// Add performance monitoring middleware if enabled
	if s.config.Performance.MonitoringEnabled && s.services.MetricsService != nil {
		s.router.Use(s.performanceMiddleware)
	}
}

// Start starts the HTTP server
func (s *Server) Start() error {
	log.Printf("Starting server on port %s", s.config.Server.Port)
	
	// Start server in a goroutine
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	return s.Shutdown()
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return s.httpServer.Shutdown(ctx)
}

// healthCheck handles health check requests
func (s *Server) corsHandler(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers for preflight requests
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With, Accept, Origin")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Max-Age", "86400")
	w.WriteHeader(http.StatusOK)
}

func (s *Server) healthCheck(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers for Obsidian compatibility
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With, Accept, Origin")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	
	// Handle preflight OPTIONS request
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	
	if s.services.HealthService == nil {
		// Fallback to simple health check
		if err := s.services.HealthCheck(); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprintf(w, `{"status":"unhealthy","error":"%s","timestamp":"%s"}`, 
				err.Error(), time.Now().Format(time.RFC3339))
			return
		}
		
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"healthy","timestamp":"%s"}`, time.Now().Format(time.RFC3339))
		return
	}
	
	// Use comprehensive health service
	systemHealth := s.services.HealthService.CheckHealth(r.Context())
	
	statusCode := http.StatusOK
	if systemHealth.Status == services.HealthStatusUnhealthy {
		statusCode = http.StatusServiceUnavailable
	} else if systemHealth.Status == services.HealthStatusDegraded {
		statusCode = http.StatusOK // Still return 200 for degraded
	}
	
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(systemHealth); err != nil {
		log.Printf("Failed to encode health response: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"error":"failed to encode health response"}`)
	}
}

// metricsHandler handles metrics requests
func (s *Server) metricsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	if s.services.MetricsService == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprintf(w, `{"error":"metrics service not available"}`)
		return
	}
	
	metrics := s.services.MetricsService.GetMetrics()
	
	// Add cache stats if available
	if s.services.CacheService != nil {
		cacheStats := s.services.CacheService.GetStats()
		metrics["cache"] = cacheStats
	}
	
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(metrics); err != nil {
		log.Printf("Failed to encode metrics: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"error":"failed to encode metrics"}`)
	}
}

// cacheStatsHandler handles cache statistics requests
func (s *Server) cacheStatsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	if s.services.CacheService == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprintf(w, `{"error":"cache service not available"}`)
		return
	}
	
	stats := s.services.CacheService.GetStats()
	
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(stats); err != nil {
		log.Printf("Failed to encode cache stats: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"error":"failed to encode cache stats"}`)
	}
}

// cacheClearHandler handles cache clear requests
func (s *Server) cacheClearHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	if s.services.CacheService == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprintf(w, `{"error":"cache service not available"}`)
		return
	}
	
	if err := s.services.CacheService.Clear(r.Context()); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"error":"failed to clear cache","details":"%s"}`, err.Error())
		return
	}
	
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"message":"cache cleared successfully","timestamp":"%s"}`, time.Now().Format(time.RFC3339))
}

// placeholder handler for routes not yet implemented
func (s *Server) placeholder(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	fmt.Fprintf(w, `{"error":"endpoint not implemented","method":"%s","path":"%s"}`, r.Method, r.URL.Path)
}