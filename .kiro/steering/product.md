# Product Overview

Semantic Text Processor is a Go-based web application for semantic text analysis and multi-database storage using Supabase API.

## Core Features

- **Text Processing**: LLM-based semantic text chunking and analysis
- **Multi-Database Storage**: PostgreSQL, PGVector, and Apache AGE via Supabase
- **Template System**: Dynamic content templates with slots for reusable structures
- **Hierarchical Structure**: Bullet-point style content organization with parent-child relationships
- **Tag System**: Flexible content tagging and categorization
- **Search Capabilities**: Semantic, graph, and tag-based search operations
- **RESTful API**: Complete HTTP API for all operations

## Architecture

The application follows a layered architecture:

1. **API Layer**: HTTP routing and middleware
2. **Service Layer**: Business logic processing  
3. **Integration Layer**: External service integration
4. **Data Access Layer**: Supabase API client

## Database Schemas

- `content_db` schema: Relational database (texts and chunks)
- `vector_db` schema: Vector database (embeddings)
- `graph_db` schema: Graph database (nodes and edges)

This separation enables future scalability and prevents data mixing.
