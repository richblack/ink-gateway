# Technology Stack

## Language & Runtime

- **Go 1.21+** - Primary programming language
- **Gorilla Mux** - HTTP routing and middleware
- **UUID** - Unique identifier generation

## External Services

- **Supabase** - Database operations via REST API
- **LLM Service** - Text chunking and entity extraction
- **Embedding Service** - Vector embedding generation

## Development Tools

- **godotenv** - Environment variable management
- **testify** - Testing framework with mocks and assertions

## Build System

### Common Commands

```bash
# Development setup
make dev-setup          # Install deps, format, and vet code
make deps               # Install and tidy dependencies

# Development
make run                # Run the application locally
make fmt                # Format all Go code
make vet                # Run Go vet for static analysis

# Testing
make test               # Run all tests
make test-coverage      # Run tests with coverage report
make check              # Format, vet, and test (quality check)

# Building
make build              # Build for development
make build-prod         # Build for production (Linux, static)
make clean              # Remove build artifacts
```

### Configuration

- Environment variables via `.env` file (copy from `.env.example`)
- Required: `SUPABASE_URL`, `SUPABASE_API_KEY`, `LLM_API_KEY`, `EMBEDDING_API_KEY`
- Optional: `SERVER_PORT` (defaults to 8080)

### Project Structure

- Interface-driven design with clear separation of concerns
- Mock implementations for testing
- Retry logic with exponential backoff for external API calls
