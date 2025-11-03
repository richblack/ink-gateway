# Search API Documentation

This document describes the search API endpoints implemented in the Semantic Text Processor.

## Base URL
```
http://localhost:8080/api/v1
```

## Search Endpoints

### 1. Semantic Search
**Endpoint:** `POST /search/semantic`

Performs vector similarity search using embeddings.

**Request Body:**
```json
{
  "query": "search query text",
  "limit": 10,
  "min_similarity": 0.7,
  "filters": {
    "text_id": "specific-text-id",
    "is_template": false,
    "indent_level": 1
  },
  "include_metadata": true
}
```

**Response:**
```json
{
  "results": [
    {
      "chunk": {
        "id": "chunk-123",
        "text_id": "text-456",
        "content": "matching content",
        "created_at": "2023-12-01T10:00:00Z"
      },
      "similarity": 0.95
    }
  ],
  "total_count": 1,
  "query": "search query text",
  "limit": 10
}
```

### 2. Graph Search
**Endpoint:** `POST /search/graph`

Performs graph traversal search using Apache AGE.

**Request Body:**
```json
{
  "entity_name": "Person Name",
  "max_depth": 3,
  "limit": 50
}
```

**Response:**
```json
{
  "nodes": [
    {
      "id": "node-123",
      "chunk_id": "chunk-456",
      "entity_name": "Person Name",
      "entity_type": "person",
      "properties": {
        "description": "Additional info"
      }
    }
  ],
  "edges": [
    {
      "id": "edge-789",
      "source_node_id": "node-123",
      "target_node_id": "node-456",
      "relationship_type": "knows",
      "properties": {
        "since": "2023"
      }
    }
  ]
}
```

### 3. Tag Search
**Endpoint:** `POST /search/tags`

Searches for chunks by tag content.

**Request Body:**
```json
{
  "tag_content": "important"
}
```

**Response:**
```json
[
  {
    "chunk": {
      "id": "chunk-123",
      "content": "tagged content",
      "created_at": "2023-12-01T10:00:00Z"
    },
    "tags": [
      {
        "id": "tag-456",
        "content": "important"
      }
    ]
  }
]
```

### 4. Chunk Search
**Endpoint:** `POST /search/chunks`

Performs text-based search on chunk content.

**Request Body:**
```json
{
  "query": "search term",
  "filters": {
    "text_id": "text-123",
    "is_template": false
  }
}
```

**Response:**
```json
[
  {
    "id": "chunk-123",
    "text_id": "text-456",
    "content": "content with search term",
    "created_at": "2023-12-01T10:00:00Z"
  }
]
```

### 5. Hybrid Search
**Endpoint:** `POST /search/hybrid`

Combines semantic and text-based search with weighted scoring.

**Request Body:**
```json
{
  "query": "hybrid search query",
  "limit": 10,
  "semantic_weight": 0.7
}
```

**Response:**
```json
[
  {
    "chunk": {
      "id": "chunk-123",
      "content": "matching content",
      "created_at": "2023-12-01T10:00:00Z"
    },
    "similarity": 0.88
  }
]
```

## Error Responses

All endpoints return standardized error responses:

```json
{
  "error": "error message",
  "details": "detailed error information"
}
```

Common HTTP status codes:
- `200 OK` - Success
- `400 Bad Request` - Invalid request body or missing required fields
- `500 Internal Server Error` - Server-side error

## Search Features

### Semantic Search Features
- Vector similarity search using embeddings
- Configurable similarity threshold
- Advanced filtering by chunk properties
- Pagination support

### Graph Search Features
- Entity-based graph traversal
- Configurable traversal depth
- Relationship type filtering
- Breadth-first search algorithm

### Tag Search Features
- Tag-based chunk retrieval
- Returns chunks with all associated tags
- Supports special characters in tag names

### Hybrid Search Features
- Combines semantic and text search
- Weighted scoring system
- Configurable semantic vs text weight
- Merged and ranked results

## Usage Examples

### Basic Semantic Search
```bash
curl -X POST http://localhost:8080/api/v1/search/semantic \
  -H "Content-Type: application/json" \
  -d '{"query": "machine learning", "limit": 5}'
```

### Graph Search for Person
```bash
curl -X POST http://localhost:8080/api/v1/search/graph \
  -H "Content-Type: application/json" \
  -d '{"entity_name": "John Doe", "max_depth": 2}'
```

### Search by Tag
```bash
curl -X POST http://localhost:8080/api/v1/search/tags \
  -H "Content-Type: application/json" \
  -d '{"tag_content": "important"}'
```

### Hybrid Search
```bash
curl -X POST http://localhost:8080/api/v1/search/hybrid \
  -H "Content-Type: application/json" \
  -d '{"query": "AI research", "limit": 10, "semantic_weight": 0.8}'
```