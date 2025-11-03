package main

import (
	"log"
	"semantic-text-processor/config"
	"semantic-text-processor/server"

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

	// Create and start server
	srv := server.NewServer(cfg)
	
	log.Println("Semantic Text Processor starting...")
	if err := srv.Start(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}