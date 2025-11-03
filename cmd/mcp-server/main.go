package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"semantic-text-processor/config"
	"semantic-text-processor/mcp"
	"semantic-text-processor/services"

	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables from .env file if it exists
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	// Load configuration
	cfg := config.LoadConfig()

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Configuration validation failed: %v", err)
	}

	// Initialize services using service factory
	mcpServices, err := initializeServices(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize services: %v", err)
	}

	// Create MCP server
	server := mcp.NewMCPServer(
		"ink-multimodal-mcp-server",
		"1.0.0",
		"Multimodal MCP Server for Ink Knowledge Base",
		mcpServices,
	)

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Received shutdown signal, stopping server...")
		server.Stop()
		os.Exit(0)
	}()

	// Start MCP server
	log.Println("Starting Ink Multimodal MCP Server...")
	if err := server.Start(); err != nil {
		log.Fatalf("Server error: %v", err)
	}

	log.Println("Server stopped")
}

// initializeServices initializes all services using the service factory
func initializeServices(cfg *config.Config) (*mcp.MCPServices, error) {
	// Create service factory and initialize services
	serviceFactory := services.NewServiceFactory(cfg)
	serviceContainer, err := serviceFactory.CreateServices()
	if err != nil {
		return nil, err
	}

	// Note: MediaProcessor, BatchProcessor, MultimodalSearch etc. need to be added
	// to ServiceContainer. For now, return basic services with nil for unimplemented.
	return &mcp.MCPServices{
		ChunkService:        serviceContainer.UnifiedChunkService,
		MediaProcessor:      nil, // TODO: Initialize when multimodal features are ready
		MultimodalSearch:    nil,
		BatchProcessor:      nil,
		ImageSimilarity:     nil,
		SlideRecommendation: nil,
		StorageService:      nil,
	}, nil
}
