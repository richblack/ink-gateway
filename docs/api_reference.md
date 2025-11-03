# Semantic Text Processor - Complete API Reference

## Table of Contents

1. [Overview](#overview)
2. [Authentication](#authentication)
3. [Base URL and Versioning](#base-url-and-versioning)
4. [Common Request/Response Patterns](#common-requestresponse-patterns)
5. [Health and Monitoring Endpoints](#health-and-monitoring-endpoints)
6. [Text Operations](#text-operations)
7. [Chunk Operations](#chunk-operations)
8. [Template Operations](#template-operations)
9. [Tag Operations](#tag-operations)
10. [Search Operations](#search-operations)
11. [Cache Operations](#cache-operations)
12. [Error Handling](#error-handling)
13. [Rate Limiting and Pagination](#rate-limiting-and-pagination)
14. [SDKs and Client Libraries](#sdks-and-client-libraries)

## Overview

The Semantic Text Processor provides a comprehensive REST API for:
- Text processing and hierarchical chunking
- Template creation and instance management
- Tag-based content organization
- Multi-modal search (semantic, graph, hybrid)
- Performance monitoring and caching

### Key Features
- **Unified Chunk System**: Hierarchical content organization with parent-child relationships
- **Template System**: Reusable content templates with slot-based customization
- **Multi-Database Backend**: PostgreSQL, PGVector, and Apache AGE integration via Supabase
- **Advanced Search**: Semantic similarity, knowledge graph, and hybrid search capabilities
- **Performance Monitoring**: Built-in metrics, caching, and health monitoring

## Authentication

Currently, the API uses Supabase API key authentication. All requests must include appropriate authentication headers.

```bash
# Using Supabase API Key
curl -H "apikey: YOUR_SUPABASE_API_KEY" \
     -H "Authorization: Bearer YOUR_JWT_TOKEN" \
     http://localhost:8080/api/v1/health
```

## Base URL and Versioning

**Base URL**: `http://localhost:8080/api/v1`

All API endpoints are versioned. Current version is `v1`. Future versions will maintain backward compatibility or provide migration paths.

## Common Request/Response Patterns

### Standard Response Format

```json
{
  "data": { /* response data */ },
  "meta": {
    "timestamp": "2024-01-15T10:30:00Z",
    "version": "1.0.0"
  }
}
```

### Pagination Format

```json
{
  "data": [ /* array of items */ ],
  "pagination": {
    "page": 1,
    "page_size": 20,
    "total": 150,
    "total_pages": 8
  }
}
```

### Error Response Format

```json
{
  "error": "error_type",
  "message": "Human readable error message",
  "details": "Additional error details",
  "code": "ERROR_CODE",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

## Health and Monitoring Endpoints

### Health Check

**Endpoint**: `GET /api/v1/health`

Check system health status including all components.

**Response**:
```json
{
  "status": "healthy",
  "timestamp": "2024-01-15T10:30:00Z",
  "uptime": "2h30m15s",
  "version": "1.0.0",
  "components": {
    "database": {
      "name": "database",
      "status": "healthy",
      "message": "Database connection successful",
      "timestamp": "2024-01-15T10:30:00Z",
      "duration": "15ms"
    },
    "cache": {
      "name": "cache",
      "status": "healthy",
      "message": "Cache operations successful",
      "timestamp": "2024-01-15T10:30:00Z",
      "duration": "2ms",
      "details": {
        "hit_rate": 0.85,
        "size": 150,
        "max_size": 1000
      }
    }
  }
}
```

**Status Codes**:
- `200`: System healthy or degraded
- `503`: System unhealthy

### Metrics

**Endpoint**: `GET /api/v1/metrics`

Retrieve system performance metrics.

**Response**:
```json
{
  "counters": {
    "http_requests_total": 1250,
    "http_requests_errors": 15,
    "cache_hits": 890,
    "cache_misses": 110
  },
  "histograms": {
    "http_request_duration": {
      "mean": 125.5,
      "p50": 98.2,
      "p95": 350.1,
      "p99": 1250.8
    }
  },
  "gauges": {
    "active_connections": 12,
    "memory_usage_bytes": 134217728
  }
}
```

## Text Operations

### Create Text

**Endpoint**: `POST /api/v1/texts`

Submit text content for processing and chunking.

**Request Body**:
```json
{
  "content": "Your text content here...",
  "title": "Optional title for the text"
}
```

**Response**:
```json
{
  "id": "text-123e4567-e89b-12d3-a456-426614174000",
  "content": "Your text content here...",
  "title": "Optional title for the text",
  "status": "processing",
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-15T10:30:00Z"
}
```

**Example**:
```bash
curl -X POST http://localhost:8080/api/v1/texts \
  -H "Content-Type: application/json" \
  -d '{
    "content": "This is a sample text for processing. It will be chunked into meaningful segments.",
    "title": "Sample Text Document"
  }'
```

### Get All Texts

**Endpoint**: `GET /api/v1/texts`

Retrieve a paginated list of all texts.

**Query Parameters**:
- `page` (int): Page number (default: 1)
- `page_size` (int): Number of items per page (default: 20, max: 100)
- `status` (string): Filter by status (`processing`, `completed`, `failed`)

**Response**:
```json
{
  "texts": [
    {
      "id": "text-123",
      "title": "Sample Text",
      "status": "completed",
      "created_at": "2024-01-15T10:30:00Z",
      "updated_at": "2024-01-15T10:35:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "page_size": 20,
    "total": 150
  }
}
```

### Get Text by ID

**Endpoint**: `GET /api/v1/texts/{id}`

Retrieve specific text details including processing status.

**Response**:
```json
{
  "text": {
    "id": "text-123",
    "content": "Full text content...",
    "title": "Sample Text",
    "status": "completed",
    "created_at": "2024-01-15T10:30:00Z",
    "updated_at": "2024-01-15T10:35:00Z"
  },
  "chunks": [
    {
      "id": "chunk-456",
      "text_id": "text-123",
      "content": "First chunk content",
      "indent_level": 0,
      "sequence_number": 1,
      "created_at": "2024-01-15T10:35:00Z"
    }
  ]
}
```

### Update Text

**Endpoint**: `PUT /api/v1/texts/{id}`

Update existing text content or title.

**Request Body**:
```json
{
  "content": "Updated text content",
  "title": "Updated title"
}
```

### Delete Text

**Endpoint**: `DELETE /api/v1/texts/{id}`

Delete a text and all associated chunks.

**Response**: `204 No Content`

### Get Text Structure

**Endpoint**: `GET /api/v1/texts/{id}/structure`

Retrieve hierarchical structure of text chunks.

**Response**:
```json
{
  "text_id": "text-123",
  "structure": [
    {
      "chunk_id": "chunk-root",
      "content": "Main content",
      "indent_level": 0,
      "children": [
        {
          "chunk_id": "chunk-child-1",
          "content": "Sub-content 1",
          "indent_level": 1,
          "children": []
        }
      ]
    }
  ]
}
```

### Update Text Structure

**Endpoint**: `PUT /api/v1/texts/{id}/structure`

Update the hierarchical structure of text chunks.

## Chunk Operations

### Get All Chunks

**Endpoint**: `GET /api/v1/chunks`

Retrieve paginated list of chunks with optional filtering.

**Query Parameters**:
- `page` (int): Page number
- `page_size` (int): Items per page
- `text_id` (string): Filter by text ID
- `is_template` (bool): Filter template chunks
- `parent_chunk_id` (string): Filter by parent chunk

**Response**:
```json
{
  "chunks": [
    {
      "id": "chunk-123",
      "text_id": "text-456",
      "content": "Chunk content",
      "is_template": false,
      "is_slot": false,
      "parent_chunk_id": null,
      "indent_level": 0,
      "sequence_number": 1,
      "metadata": {},
      "created_at": "2024-01-15T10:30:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "page_size": 20,
    "total": 85
  }
}
```

### Create Chunk

**Endpoint**: `POST /api/v1/chunks`

Create a new chunk with optional parent relationship.

**Request Body**:
```json
{
  "text_id": "text-123",
  "content": "New chunk content",
  "parent_chunk_id": "chunk-parent-456",
  "indent_level": 1,
  "sequence_number": 2,
  "metadata": {
    "category": "important",
    "tags": ["concept", "definition"]
  }
}
```

### Get Chunk by ID

**Endpoint**: `GET /api/v1/chunks/{id}`

Retrieve specific chunk details.

### Update Chunk

**Endpoint**: `PUT /api/v1/chunks/{id}`

Update chunk content and properties.

**Request Body**:
```json
{
  "content": "Updated chunk content",
  "parent_chunk_id": "new-parent-456",
  "indent_level": 2,
  "metadata": {
    "updated_field": "new_value"
  }
}
```

### Delete Chunk

**Endpoint**: `DELETE /api/v1/chunks/{id}`

Delete a chunk and optionally its children.

### Get Chunk Hierarchy

**Endpoint**: `GET /api/v1/chunks/{id}/hierarchy`

Get full hierarchical tree starting from the specified chunk.

**Response**:
```json
{
  "chunk": {
    "id": "chunk-123",
    "content": "Parent content",
    "indent_level": 0
  },
  "children": [
    {
      "chunk": {
        "id": "chunk-456",
        "content": "Child content",
        "indent_level": 1
      },
      "children": []
    }
  ],
  "depth": 2,
  "total_descendants": 5
}
```

### Get Chunk Children

**Endpoint**: `GET /api/v1/chunks/{id}/children`

Get direct children of a chunk.

### Move Chunk

**Endpoint**: `POST /api/v1/chunks/{id}/move`

Move a chunk to a new position in the hierarchy.

**Request Body**:
```json
{
  "new_parent_id": "chunk-new-parent",
  "new_position": 3,
  "new_indent_level": 2
}
```

### Batch Create Chunks

**Endpoint**: `POST /api/v1/chunks/batch`

Create multiple chunks in a single request (Unified handlers only).

**Request Body**:
```json
{
  "chunks": [
    {
      "text_id": "text-123",
      "content": "Chunk 1 content",
      "indent_level": 0
    },
    {
      "text_id": "text-123",
      "content": "Chunk 2 content",
      "indent_level": 1,
      "parent_chunk_id": "chunk-1-id"
    }
  ]
}
```

### Batch Update Chunks

**Endpoint**: `PUT /api/v1/chunks/batch`

Update multiple chunks in a single request (Unified handlers only).

## Template Operations

### Create Template

**Endpoint**: `POST /api/v1/templates`

Create a reusable content template with slots.

**Request Body**:
```json
{
  "template_name": "Meeting Notes Template",
  "slot_names": ["date", "participants", "topics", "action_items"]
}
```

**Response**:
```json
{
  "template_id": "template-123",
  "template_name": "Meeting Notes Template",
  "chunks": [
    {
      "id": "chunk-template-root",
      "content": "Meeting Notes Template",
      "is_template": true,
      "slots": [
        {
          "id": "slot-date",
          "name": "date",
          "chunk_id": "chunk-slot-date"
        }
      ]
    }
  ],
  "created_at": "2024-01-15T10:30:00Z"
}
```

### Get All Templates

**Endpoint**: `GET /api/v1/templates`

Retrieve all available templates.

### Get Template by Content

**Endpoint**: `GET /api/v1/templates/{content}`

Retrieve template by its content/name.

### Create Template Instance

**Endpoint**: `POST /api/v1/templates/{id}/instances`

Create an instance of a template with filled slots.

**Request Body**:
```json
{
  "instance_name": "Team Meeting - Jan 15",
  "slot_values": {
    "date": "January 15, 2024",
    "participants": "Alice, Bob, Charlie",
    "topics": "Project updates, Budget review",
    "action_items": "1. Complete design review\n2. Schedule follow-up"
  }
}
```

### Update Slot Value

**Endpoint**: `PUT /api/v1/instances/{id}/slots`

Update specific slot values in a template instance.

**Request Body**:
```json
{
  "slot_name": "action_items",
  "value": "Updated action items content"
}
```

## Tag Operations

### Add Tag to Chunk

**Endpoint**: `POST /api/v1/chunks/{id}/tags`

Add a tag to a specific chunk.

**Request Body**:
```json
{
  "tag_content": "important"
}
```

### Remove Tag from Chunk

**Endpoint**: `DELETE /api/v1/chunks/{id}/tags/{tagId}`

Remove a specific tag from a chunk.

### Get Chunk Tags

**Endpoint**: `GET /api/v1/chunks/{id}/tags`

Retrieve all tags associated with a chunk.

**Response**:
```json
{
  "chunk_id": "chunk-123",
  "tags": [
    {
      "id": "tag-456",
      "content": "important",
      "created_at": "2024-01-15T10:30:00Z"
    },
    {
      "id": "tag-789",
      "content": "meeting-notes",
      "created_at": "2024-01-15T10:35:00Z"
    }
  ]
}
```

### Get Chunks by Tag

**Endpoint**: `GET /api/v1/tags/{content}/chunks`

Retrieve all chunks with a specific tag.

### Batch Tag Operations

**Endpoint**: `POST /api/v1/chunks/tags/batch`

Perform multiple tag operations in a single request (Unified handlers only).

**Request Body**:
```json
{
  "operations": [
    {
      "type": "add",
      "chunk_id": "chunk-123",
      "tag_content": "important"
    },
    {
      "type": "remove",
      "chunk_id": "chunk-456",
      "tag_id": "tag-789"
    }
  ]
}
```

### Search Chunks by Multiple Tags

**Endpoint**: `POST /api/v1/tags/search`

Find chunks that match multiple tag criteria (Unified handlers only).

**Request Body**:
```json
{
  "tags": ["important", "meeting-notes"],
  "operator": "AND",
  "limit": 50
}
```

## Search Operations

### Semantic Search

**Endpoint**: `POST /api/v1/search/semantic`

Perform vector similarity search using embeddings.

**Request Body**:
```json
{
  "query": "machine learning algorithms",
  "limit": 10,
  "min_similarity": 0.7,
  "filters": {
    "text_id": "text-123",
    "is_template": false,
    "indent_level": 1
  },
  "include_metadata": true
}
```

**Response**:
```json
{
  "results": [
    {
      "chunk": {
        "id": "chunk-123",
        "text_id": "text-456",
        "content": "Machine learning algorithms are computational methods...",
        "created_at": "2024-01-15T10:30:00Z",
        "metadata": {
          "category": "technical"
        }
      },
      "similarity": 0.95,
      "embedding_id": "emb-789"
    }
  ],
  "total_count": 1,
  "query": "machine learning algorithms",
  "limit": 10,
  "execution_time_ms": 45
}
```

### Graph Search

**Endpoint**: `POST /api/v1/search/graph`

Perform knowledge graph traversal search.

**Request Body**:
```json
{
  "entity_name": "Neural Networks",
  "max_depth": 3,
  "limit": 50,
  "relationship_types": ["relates_to", "depends_on"]
}
```

**Response**:
```json
{
  "nodes": [
    {
      "id": "node-123",
      "chunk_id": "chunk-456",
      "entity_name": "Neural Networks",
      "entity_type": "concept",
      "properties": {
        "description": "Computational model inspired by biological neural networks"
      }
    }
  ],
  "edges": [
    {
      "id": "edge-789",
      "source_node_id": "node-123",
      "target_node_id": "node-456",
      "relationship_type": "relates_to",
      "properties": {
        "strength": 0.85
      }
    }
  ],
  "query_depth": 3,
  "execution_time_ms": 125
}
```

### Tag Search

**Endpoint**: `POST /api/v1/search/tags`

Search for chunks by tag content.

**Request Body**:
```json
{
  "tag_content": "machine-learning"
}
```

### Chunk Search

**Endpoint**: `POST /api/v1/search/chunks`

Perform text-based search on chunk content.

**Request Body**:
```json
{
  "query": "neural network training",
  "filters": {
    "text_id": "text-123",
    "is_template": false,
    "min_indent_level": 0,
    "max_indent_level": 2
  },
  "limit": 20
}
```

### Hybrid Search

**Endpoint**: `POST /api/v1/search/hybrid`

Combine semantic and text-based search with weighted scoring.

**Request Body**:
```json
{
  "query": "deep learning optimization techniques",
  "limit": 15,
  "semantic_weight": 0.7,
  "text_weight": 0.3,
  "filters": {
    "min_similarity": 0.6
  }
}
```

**Response**:
```json
{
  "results": [
    {
      "chunk": {
        "id": "chunk-123",
        "content": "Deep learning optimization involves various techniques...",
        "created_at": "2024-01-15T10:30:00Z"
      },
      "similarity": 0.88,
      "text_score": 0.75,
      "combined_score": 0.835,
      "ranking_factors": {
        "semantic_similarity": 0.88,
        "text_relevance": 0.75,
        "semantic_weight": 0.7,
        "text_weight": 0.3
      }
    }
  ],
  "query": "deep learning optimization techniques",
  "limit": 15,
  "execution_time_ms": 85
}
```

## Cache Operations

### Get Cache Statistics

**Endpoint**: `GET /api/v1/cache/stats`

Retrieve cache performance statistics.

**Response**:
```json
{
  "hit_rate": 0.85,
  "miss_rate": 0.15,
  "size": 1250,
  "max_size": 10000,
  "evictions": 45,
  "memory_usage_bytes": 67108864,
  "avg_item_size_bytes": 53687,
  "oldest_item_age_seconds": 3600,
  "operations": {
    "gets": 5420,
    "sets": 812,
    "deletes": 45,
    "hits": 4607,
    "misses": 813
  }
}
```

### Clear Cache

**Endpoint**: `POST /api/v1/cache/clear`

Clear all cached data.

**Response**:
```json
{
  "message": "Cache cleared successfully",
  "timestamp": "2024-01-15T10:30:00Z",
  "items_cleared": 1250
}
```

## Error Handling

### HTTP Status Codes

- `200 OK`: Request successful
- `201 Created`: Resource created successfully
- `204 No Content`: Request successful, no content to return
- `400 Bad Request`: Invalid request format or parameters
- `401 Unauthorized`: Authentication required
- `403 Forbidden`: Access denied
- `404 Not Found`: Resource not found
- `409 Conflict`: Resource conflict (e.g., duplicate entry)
- `422 Unprocessable Entity`: Valid request format but semantic errors
- `429 Too Many Requests`: Rate limit exceeded
- `500 Internal Server Error`: Server-side error
- `503 Service Unavailable`: Service temporarily unavailable

### Error Response Examples

**Validation Error**:
```json
{
  "error": "validation_error",
  "message": "Invalid request parameters",
  "details": {
    "field_errors": {
      "limit": "must be between 1 and 100",
      "min_similarity": "must be between 0.0 and 1.0"
    }
  },
  "code": "VALIDATION_FAILED",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

**Resource Not Found**:
```json
{
  "error": "not_found",
  "message": "Chunk not found",
  "details": "No chunk found with ID: chunk-nonexistent",
  "code": "CHUNK_NOT_FOUND",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

**Rate Limit Exceeded**:
```json
{
  "error": "rate_limit_exceeded",
  "message": "Too many requests",
  "details": "Rate limit: 100 requests per minute",
  "code": "RATE_LIMIT_EXCEEDED",
  "retry_after_seconds": 45,
  "timestamp": "2024-01-15T10:30:00Z"
}
```

## Rate Limiting and Pagination

### Rate Limiting

Default limits (configurable):
- 100 requests per minute per IP
- 1000 requests per hour per authenticated user
- Special limits for resource-intensive operations:
  - Search operations: 50 per minute
  - Batch operations: 20 per minute

Rate limit headers are included in all responses:
```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 85
X-RateLimit-Reset: 1642248600
```

### Pagination

All list endpoints support pagination:

**Query Parameters**:
- `page`: Page number (starts at 1)
- `page_size`: Items per page (default: 20, max: 100)

**Response Format**:
```json
{
  "data": [ /* items */ ],
  "pagination": {
    "page": 1,
    "page_size": 20,
    "total": 150,
    "total_pages": 8,
    "has_next": true,
    "has_prev": false
  }
}
```

## SDKs and Client Libraries

### cURL Examples

**Basic Authentication**:
```bash
export SUPABASE_URL="https://your-project.supabase.co"
export SUPABASE_API_KEY="your-api-key"
export API_BASE="http://localhost:8080/api/v1"

# Health check
curl -H "apikey: $SUPABASE_API_KEY" "$API_BASE/health"

# Create text
curl -X POST "$API_BASE/texts" \
  -H "Content-Type: application/json" \
  -H "apikey: $SUPABASE_API_KEY" \
  -d '{"content": "Sample text", "title": "Test"}'

# Semantic search
curl -X POST "$API_BASE/search/semantic" \
  -H "Content-Type: application/json" \
  -H "apikey: $SUPABASE_API_KEY" \
  -d '{"query": "machine learning", "limit": 5}'
```

### Python Example

```python
import requests

class SemanticTextProcessorClient:
    def __init__(self, base_url, api_key):
        self.base_url = base_url
        self.headers = {
            'Content-Type': 'application/json',
            'apikey': api_key
        }

    def create_text(self, content, title=None):
        data = {'content': content}
        if title:
            data['title'] = title

        response = requests.post(
            f"{self.base_url}/texts",
            json=data,
            headers=self.headers
        )
        response.raise_for_status()
        return response.json()

    def semantic_search(self, query, limit=10, min_similarity=0.7):
        data = {
            'query': query,
            'limit': limit,
            'min_similarity': min_similarity
        }

        response = requests.post(
            f"{self.base_url}/search/semantic",
            json=data,
            headers=self.headers
        )
        response.raise_for_status()
        return response.json()

# Usage
client = SemanticTextProcessorClient(
    "http://localhost:8080/api/v1",
    "your-api-key"
)

# Create text
text_result = client.create_text(
    "This is sample content for processing",
    "Sample Document"
)

# Search
search_results = client.semantic_search(
    "machine learning algorithms",
    limit=5
)
```

### JavaScript/Node.js Example

```javascript
class SemanticTextProcessorClient {
    constructor(baseUrl, apiKey) {
        this.baseUrl = baseUrl;
        this.headers = {
            'Content-Type': 'application/json',
            'apikey': apiKey
        };
    }

    async createText(content, title = null) {
        const data = { content };
        if (title) data.title = title;

        const response = await fetch(`${this.baseUrl}/texts`, {
            method: 'POST',
            headers: this.headers,
            body: JSON.stringify(data)
        });

        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }

        return await response.json();
    }

    async semanticSearch(query, limit = 10, minSimilarity = 0.7) {
        const data = {
            query,
            limit,
            min_similarity: minSimilarity
        };

        const response = await fetch(`${this.baseUrl}/search/semantic`, {
            method: 'POST',
            headers: this.headers,
            body: JSON.stringify(data)
        });

        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }

        return await response.json();
    }
}

// Usage
const client = new SemanticTextProcessorClient(
    'http://localhost:8080/api/v1',
    'your-api-key'
);

// Example usage with async/await
(async () => {
    try {
        const textResult = await client.createText(
            'This is sample content for processing',
            'Sample Document'
        );
        console.log('Text created:', textResult);

        const searchResults = await client.semanticSearch(
            'machine learning algorithms',
            5
        );
        console.log('Search results:', searchResults);
    } catch (error) {
        console.error('Error:', error);
    }
})();
```

---

For more detailed implementation examples and advanced usage patterns, see the [User Guide](user_guides/getting_started.md) and [Performance Optimization Guide](performance_optimization_guide.md).