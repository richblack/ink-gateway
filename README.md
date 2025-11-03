# Semantic Text Processor

A Go-based web application for semantic text analysis and multi-database storage using Supabase API.

## ⚠️ 重要約束條件

**🚨 只能使用 Supabase API 進行數據庫操作，絕對不可直接連接數據庫**

詳細約束條件請參閱 [docs/CONSTRAINTS.md](docs/CONSTRAINTS.md)

## Project Structure

```
semantic-text-processor/
├── main.go                 # Application entry point
├── go.mod                  # Go module definition
├── .env.example           # Environment variables template
├── README.md              # Project documentation
├── config/                # Configuration management
│   └── config.go
├── models/                # Data models and structures
│   ├── types.go          # Core data types
│   ├── requests.go       # API request structures
│   └── responses.go      # API response structures
├── services/              # Business logic interfaces
│   └── interfaces.go     # Service interface definitions
├── clients/               # External service clients
│   └── supabase.go       # Supabase client interface
├── handlers/              # HTTP request handlers
│   └── interfaces.go     # Handler interface definitions
└── server/                # HTTP server setup
    ├── server.go         # Server configuration and routing
    └── middleware.go     # HTTP middleware
```

## Features

- **Text Processing**: LLM-based semantic text chunking
- **Multi-Database Storage**: PostgreSQL, PGVector, and Apache AGE via Supabase
- **Template System**: Dynamic content templates with slots
- **Hierarchical Structure**: Bullet-point style content organization
- **Tag System**: Flexible content tagging and categorization
- **Search Capabilities**: Semantic, graph, and tag-based search
- **RESTful API**: Complete HTTP API for all operations

## Configuration

Copy `.env.example` to `.env` and configure the following:

- `SUPABASE_URL`: Your Supabase project URL
- `SUPABASE_API_KEY`: Your Supabase API key
- `LLM_API_KEY`: API key for LLM service
- `EMBEDDING_API_KEY`: API key for embedding service

## Getting Started

1. Install dependencies:
   ```bash
   go mod tidy
   ```

2. Configure environment variables:
   ```bash
   cp .env.example .env
   # Edit .env with your configuration
   ```

3. Run the application:
   ```bash
   go run main.go
   ```

The server will start on port 8080 (or the port specified in SERVER_PORT).

## API Endpoints

### Health Check
- `GET /api/v1/health` - Health check endpoint

### Text Operations
- `POST /api/v1/texts` - Submit text for processing
- `GET /api/v1/texts` - List texts (with pagination)
- `GET /api/v1/texts/{id}` - Get specific text details

### Template Operations
- `POST /api/v1/templates` - Create new template
- `GET /api/v1/templates` - List all templates
- `POST /api/v1/templates/{id}/instances` - Create template instance

### Search Operations
- `POST /api/v1/search/semantic` - Semantic similarity search
- `POST /api/v1/search/graph` - Knowledge graph search
- `POST /api/v1/search/tags` - Tag-based search

## Architecture

The application follows a layered architecture:

1. **API Layer**: HTTP routing and middleware
2. **Service Layer**: Business logic processing
3. **Integration Layer**: External service integration
4. **Data Access Layer**: Supabase API client

All data operations go through Supabase API to interact with PostgreSQL, PGVector, and Apache AGE databases.