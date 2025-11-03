# FAQ and Solution Knowledge Base

## Table of Contents

1. [General Questions](#general-questions)
2. [Installation and Setup](#installation-and-setup)
3. [API Usage](#api-usage)
4. [Performance and Optimization](#performance-and-optimization)
5. [Troubleshooting](#troubleshooting)
6. [Data Management](#data-management)
7. [Security](#security)
8. [Integrations](#integrations)
9. [Common Error Solutions](#common-error-solutions)
10. [Best Practices](#best-practices)

## General Questions

### Q: What is the Semantic Text Processor?

**A:** The Semantic Text Processor is a Go-based microservice that provides text processing, chunking, embedding generation, knowledge graph extraction, and semantic search capabilities. It uses a unified chunk system to organize content hierarchically and supports templates, tags, and multi-modal search.

**Key Features:**
- Hierarchical text chunking
- Template system with slot-based customization
- Semantic, graph, and hybrid search
- Multi-database backend (PostgreSQL, PGVector, Apache AGE)
- Performance monitoring and caching
- RESTful API

### Q: What databases does the system support?

**A:** The system uses multiple database technologies through Supabase:
- **PostgreSQL**: Primary data storage for texts, chunks, and metadata
- **PGVector**: Vector storage and similarity search for embeddings
- **Apache AGE**: Graph database for knowledge graph relationships

**Important:** All database operations must go through the Supabase API. Direct database connections are not supported.

### Q: What are the system requirements?

**A:**
**Minimum Requirements:**
- CPU: 2 cores
- Memory: 4GB RAM
- Storage: 50GB SSD
- Network: 100Mbps

**Recommended Production:**
- CPU: 4+ cores
- Memory: 8GB+ RAM
- Storage: 100GB+ SSD
- Network: 1Gbps

### Q: Can the system be deployed in the cloud?

**A:** Yes, the system is designed for cloud deployment and supports:
- Docker containers
- Kubernetes orchestration
- AWS, GCP, Azure cloud platforms
- Auto-scaling capabilities
- Load balancing
- Distributed caching

## Installation and Setup

### Q: How do I install and configure the system?

**A:**
1. **Prerequisites:**
   ```bash
   # Install Go 1.19+
   go version

   # Install Docker (optional)
   docker --version
   ```

2. **Configuration:**
   ```bash
   # Copy environment template
   cp .env.example .env

   # Edit configuration
   nano .env
   ```

3. **Required Environment Variables:**
   ```bash
   SUPABASE_URL=https://your-project.supabase.co
   SUPABASE_API_KEY=your-api-key
   LLM_API_KEY=your-llm-key
   EMBEDDING_API_KEY=your-embedding-key
   ```

4. **Start the service:**
   ```bash
   go mod tidy
   go run main.go
   ```

### Q: How do I set up Supabase integration?

**A:**
1. **Create Supabase Project:**
   - Go to [supabase.com](https://supabase.com)
   - Create new project
   - Note the project URL and API key

2. **Enable Required Extensions:**
   ```sql
   -- Enable vector extension
   CREATE EXTENSION IF NOT EXISTS vector;

   -- Enable Apache AGE
   CREATE EXTENSION IF NOT EXISTS age;
   ```

3. **Configure Environment:**
   ```bash
   export SUPABASE_URL="https://your-project.supabase.co"
   export SUPABASE_API_KEY="your-anon-key"
   ```

### Q: What external services are required?

**A:**
**Required Services:**
- **Supabase**: Database and API backend
- **OpenAI API** (or compatible): LLM and embedding services

**Optional Services:**
- **Redis**: Distributed caching (fallback to in-memory)
- **Prometheus**: Metrics collection
- **Grafana**: Monitoring dashboards

## API Usage

### Q: How do I create and process text?

**A:**
```bash
# Create text
curl -X POST http://localhost:8080/api/v1/texts \
  -H "Content-Type: application/json" \
  -d '{
    "content": "Your text content here",
    "title": "Document Title"
  }'

# Response
{
  "id": "text-123",
  "status": "processing",
  "created_at": "2024-01-15T10:30:00Z"
}

# Check processing status
curl http://localhost:8080/api/v1/texts/text-123

# Get processed chunks
curl http://localhost:8080/api/v1/texts/text-123 | jq '.chunks'
```

### Q: How do I perform semantic search?

**A:**
```bash
# Basic semantic search
curl -X POST http://localhost:8080/api/v1/search/semantic \
  -H "Content-Type: application/json" \
  -d '{
    "query": "machine learning algorithms",
    "limit": 10,
    "min_similarity": 0.7
  }'

# Advanced search with filters
curl -X POST http://localhost:8080/api/v1/search/semantic \
  -H "Content-Type: application/json" \
  -d '{
    "query": "neural networks",
    "limit": 5,
    "min_similarity": 0.8,
    "filters": {
      "text_id": "specific-text-id",
      "is_template": false
    },
    "include_metadata": true
  }'
```

### Q: How do I create and use templates?

**A:**
```bash
# Create template
curl -X POST http://localhost:8080/api/v1/templates \
  -H "Content-Type: application/json" \
  -d '{
    "template_name": "Meeting Notes",
    "slot_names": ["date", "participants", "topics", "action_items"]
  }'

# Create template instance
curl -X POST http://localhost:8080/api/v1/templates/template-123/instances \
  -H "Content-Type: application/json" \
  -d '{
    "instance_name": "Weekly Team Meeting",
    "slot_values": {
      "date": "2024-01-15",
      "participants": "Alice, Bob, Charlie",
      "topics": "Project updates, Budget review",
      "action_items": "Complete design review by Friday"
    }
  }'
```

### Q: How do I manage tags?

**A:**
```bash
# Add tag to chunk
curl -X POST http://localhost:8080/api/v1/chunks/chunk-123/tags \
  -H "Content-Type: application/json" \
  -d '{"tag_content": "important"}'

# Get chunks by tag
curl http://localhost:8080/api/v1/tags/important/chunks

# Remove tag
curl -X DELETE http://localhost:8080/api/v1/chunks/chunk-123/tags/tag-456

# Batch tag operations (unified handlers)
curl -X POST http://localhost:8080/api/v1/chunks/tags/batch \
  -H "Content-Type: application/json" \
  -d '{
    "operations": [
      {"type": "add", "chunk_id": "chunk-123", "tag_content": "urgent"},
      {"type": "remove", "chunk_id": "chunk-456", "tag_id": "tag-789"}
    ]
  }'
```

## Performance and Optimization

### Q: How can I improve API response times?

**A:**
**Immediate Actions:**
1. **Enable and optimize caching:**
   ```bash
   export CACHE_ENABLED=true
   export CACHE_MAX_SIZE=2000
   export CACHE_DEFAULT_TTL=3600
   ```

2. **Check cache performance:**
   ```bash
   curl http://localhost:8080/api/v1/cache/stats
   # Target: hit_rate > 0.8
   ```

3. **Optimize database queries:**
   ```sql
   -- Create indexes for frequent queries
   CREATE INDEX CONCURRENTLY idx_chunks_text_id ON chunks(text_id);
   CREATE INDEX CONCURRENTLY idx_chunks_parent_id ON chunks(parent_chunk_id);
   ```

**Long-term Solutions:**
- Implement horizontal scaling
- Use read replicas for database
- Optimize vector search parameters
- Enable response compression

### Q: Why is semantic search slow?

**A:**
**Common Causes and Solutions:**

1. **Large vector space:**
   ```bash
   # Optimize vector search parameters
   export VECTOR_SEARCH_LISTS=100
   export VECTOR_SEARCH_PROBES=10
   ```

2. **Missing vector indexes:**
   ```sql
   -- Create optimized vector index
   CREATE INDEX CONCURRENTLY embeddings_vector_idx
   ON embeddings USING ivfflat (vector vector_cosine_ops)
   WITH (lists = 100);
   ```

3. **High similarity threshold:**
   ```bash
   # Lower similarity threshold for faster results
   curl -X POST /api/v1/search/semantic \
     -d '{"query": "...", "min_similarity": 0.6}'  # Instead of 0.8
   ```

4. **Large result sets:**
   ```bash
   # Limit results for better performance
   curl -X POST /api/v1/search/semantic \
     -d '{"query": "...", "limit": 10}'  # Instead of 100
   ```

### Q: How do I monitor system performance?

**A:**
**Built-in Monitoring:**
```bash
# Health check
curl http://localhost:8080/api/v1/health

# Performance metrics
curl http://localhost:8080/api/v1/metrics

# Cache statistics
curl http://localhost:8080/api/v1/cache/stats
```

**Key Metrics to Monitor:**
- Response time P95 < 250ms
- Cache hit rate > 80%
- Memory usage < 80%
- CPU utilization < 70%
- Error rate < 1%

**Alerting Thresholds:**
```yaml
alerts:
  response_time_p95: 500ms
  cache_hit_rate: 0.7
  memory_usage: 85%
  cpu_usage: 80%
  error_rate: 5%
```

## Troubleshooting

### Q: The service won't start. What should I check?

**A:**
**Systematic Troubleshooting:**

1. **Check configuration:**
   ```bash
   # Verify environment variables
   echo $SUPABASE_URL
   echo $SUPABASE_API_KEY

   # Test configuration
   semantic-text-processor --config-check
   ```

2. **Test external dependencies:**
   ```bash
   # Test Supabase connectivity
   curl -H "apikey: $SUPABASE_API_KEY" "$SUPABASE_URL/rest/v1/"

   # Test LLM API
   curl -H "Authorization: Bearer $LLM_API_KEY" \
        https://api.openai.com/v1/models
   ```

3. **Check system resources:**
   ```bash
   # Memory and disk space
   free -h
   df -h

   # Port availability
   netstat -tlnp | grep :8080
   ```

4. **Review logs:**
   ```bash
   # Application logs
   tail -f /var/log/semantic-text-processor.log

   # System logs
   journalctl -u semantic-text-processor -f
   ```

### Q: API requests are returning 500 errors. How do I debug?

**A:**
**Error Investigation Steps:**

1. **Check health status:**
   ```bash
   curl http://localhost:8080/api/v1/health
   ```

2. **Review error logs:**
   ```bash
   # Find recent errors
   grep -i error /var/log/semantic-text-processor.log | tail -20

   # Check for database errors
   grep -i "database\|supabase" /var/log/semantic-text-processor.log | tail -10
   ```

3. **Test individual components:**
   ```bash
   # Test database connectivity
   curl -H "apikey: $SUPABASE_API_KEY" \
        "$SUPABASE_URL/rest/v1/chunks?limit=1"

   # Test cache functionality
   curl http://localhost:8080/api/v1/cache/stats
   ```

4. **Enable debug logging:**
   ```bash
   export LOG_LEVEL=debug
   systemctl restart semantic-text-processor
   ```

### Q: Embeddings are not being generated. What's wrong?

**A:**
**Common Issues and Solutions:**

1. **API key problems:**
   ```bash
   # Test embedding API
   curl -H "Authorization: Bearer $EMBEDDING_API_KEY" \
        -H "Content-Type: application/json" \
        -d '{"input": "test text", "model": "text-embedding-ada-002"}' \
        https://api.openai.com/v1/embeddings
   ```

2. **Rate limiting:**
   ```bash
   # Check for rate limit errors in logs
   grep -i "rate limit\|429" /var/log/semantic-text-processor.log

   # Reduce embedding batch size
   export EMBEDDING_BATCH_SIZE=50  # Default might be too high
   ```

3. **Content issues:**
   ```bash
   # Check chunk content length
   curl http://localhost:8080/api/v1/chunks | \
     jq '.chunks[] | select(.content | length < 10 or length > 8000)'
   ```

4. **Processing queue:**
   ```bash
   # Check processing status
   curl http://localhost:8080/api/v1/admin/processing/status
   ```

### Q: Search results are inconsistent or poor quality. How do I improve them?

**A:**
**Search Quality Improvement:**

1. **Check embedding quality:**
   ```bash
   # Verify embeddings exist
   curl -X POST http://localhost:8080/api/v1/admin/query \
     -d '{"query": "SELECT COUNT(*) FROM chunks c LEFT JOIN embeddings e ON c.id = e.chunk_id WHERE e.chunk_id IS NULL"}'
   ```

2. **Adjust search parameters:**
   ```bash
   # Lower similarity threshold for more results
   curl -X POST /api/v1/search/semantic \
     -d '{"query": "...", "min_similarity": 0.6}'

   # Use hybrid search for better results
   curl -X POST /api/v1/search/hybrid \
     -d '{"query": "...", "semantic_weight": 0.7, "text_weight": 0.3}'
   ```

3. **Improve chunking:**
   ```bash
   # Check chunk size distribution
   curl http://localhost:8080/api/v1/chunks | \
     jq '.chunks[] | .content | length' | sort -n
   ```

4. **Re-process problematic content:**
   ```bash
   # Re-generate embeddings
   curl -X POST http://localhost:8080/api/v1/admin/embeddings/regenerate \
     -d '{"text_id": "problematic-text-id"}'
   ```

## Data Management

### Q: How do I backup and restore data?

**A:**
**Backup Procedures:**

1. **API-based backup:**
   ```bash
   # Export all data
   mkdir backup_$(date +%Y%m%d)
   cd backup_$(date +%Y%m%d)

   # Export texts
   curl http://localhost:8080/api/v1/texts > texts.json

   # Export chunks
   curl http://localhost:8080/api/v1/chunks > chunks.json

   # Export templates
   curl http://localhost:8080/api/v1/templates > templates.json
   ```

2. **Database-level backup (via Supabase):**
   ```bash
   # Using Supabase CLI
   supabase db dump --db-url "$SUPABASE_URL" > backup.sql
   ```

**Restore Procedures:**
```bash
# Restore from API backup
python3 restore_from_backup.py --backup-dir backup_20240115

# Restore from database dump
supabase db reset --db-url "$SUPABASE_URL"
psql "$SUPABASE_URL" < backup.sql
```

### Q: How do I migrate to the unified chunk system?

**A:**
**Migration Steps:**

1. **Pre-migration preparation:**
   ```bash
   # Create backup
   ./backup_system.sh

   # Test migration in staging
   ./test_migration.sh
   ```

2. **Run migration:**
   ```bash
   # Enable maintenance mode
   export MAINTENANCE_MODE=true

   # Run migration script
   python3 migrate_to_unified_chunks.py --verify --no-dry-run

   # Verify migration
   python3 verify_migration.py
   ```

3. **Post-migration validation:**
   ```bash
   # Check data consistency
   ./consistency_check.sh

   # Performance validation
   ./performance_test.sh
   ```

### Q: How do I clean up orphaned data?

**A:**
**Data Cleanup Procedures:**

1. **Find orphaned data:**
   ```bash
   # Orphaned chunks
   curl -X POST http://localhost:8080/api/v1/admin/query \
     -d '{"query": "SELECT COUNT(*) FROM chunks WHERE parent_chunk_id IS NOT NULL AND parent_chunk_id NOT IN (SELECT id FROM chunks)"}'

   # Orphaned embeddings
   curl -X POST http://localhost:8080/api/v1/admin/query \
     -d '{"query": "SELECT COUNT(*) FROM embeddings WHERE chunk_id NOT IN (SELECT id FROM chunks)"}'
   ```

2. **Clean up orphaned data:**
   ```bash
   # Run data repair (dry run first)
   python3 data_repair.py --dry-run

   # Run actual cleanup
   python3 data_repair.py --fix-orphaned-chunks --fix-orphaned-embeddings
   ```

3. **Verify cleanup:**
   ```bash
   # Run consistency check
   ./consistency_check.sh
   ```

## Security

### Q: How do I secure API access?

**A:**
**Security Best Practices:**

1. **API Key Management:**
   ```bash
   # Use environment variables, never hardcode
   export SUPABASE_API_KEY="your-secure-key"

   # Rotate keys regularly
   # Use different keys for different environments
   ```

2. **Network Security:**
   ```bash
   # Firewall configuration
   ufw allow 8080/tcp from 10.0.0.0/8  # Internal network only
   ufw deny 8080/tcp  # Block external access

   # Use reverse proxy with SSL
   # Configure rate limiting
   ```

3. **Input Validation:**
   ```bash
   # The API automatically validates:
   # - Request size limits
   # - Input sanitization
   # - SQL injection prevention
   ```

### Q: What data is stored and how is it protected?

**A:**
**Data Storage and Protection:**

1. **Data Types Stored:**
   - Original text content
   - Processed chunks and hierarchy
   - Vector embeddings (mathematical representations)
   - Metadata and tags
   - Processing logs (no sensitive content)

2. **Protection Measures:**
   - All data encrypted in transit (HTTPS)
   - Database encryption at rest (via Supabase)
   - API key authentication required
   - Input validation and sanitization
   - No direct database access allowed

3. **Data Retention:**
   ```bash
   # Configure data retention policies
   export DATA_RETENTION_DAYS=365

   # Automatic cleanup of old data
   ./cleanup_old_data.sh
   ```

## Integrations

### Q: How do I integrate with my existing application?

**A:**
**Integration Approaches:**

1. **REST API Integration:**
   ```python
   # Python example
   import requests

   class SemanticProcessor:
       def __init__(self, api_base, api_key):
           self.api_base = api_base
           self.headers = {'Authorization': f'Bearer {api_key}'}

       def process_text(self, content, title=None):
           response = requests.post(
               f"{self.api_base}/texts",
               json={'content': content, 'title': title},
               headers=self.headers
           )
           return response.json()

       def search(self, query, limit=10):
           response = requests.post(
               f"{self.api_base}/search/semantic",
               json={'query': query, 'limit': limit},
               headers=self.headers
           )
           return response.json()
   ```

2. **Webhook Integration:**
   ```bash
   # Configure webhooks for processing completion
   export WEBHOOK_URL="https://your-app.com/webhook"
   export WEBHOOK_EVENTS="text.processed,search.completed"
   ```

3. **Batch Processing:**
   ```python
   # Batch process multiple documents
   def batch_process_documents(documents):
       results = []
       for doc in documents:
           result = semantic_processor.process_text(
               doc['content'],
               doc['title']
           )
           results.append(result)
       return results
   ```

### Q: Can I use custom embedding models?

**A:**
**Custom Embedding Integration:**

1. **Configure custom endpoint:**
   ```bash
   export EMBEDDING_API_URL="https://your-custom-api.com/embeddings"
   export EMBEDDING_MODEL="your-custom-model"
   export EMBEDDING_DIMENSIONS=768  # Adjust based on your model
   ```

2. **API compatibility requirements:**
   ```json
   {
     "input": "text to embed",
     "model": "your-model-name"
   }
   ```

3. **Response format:**
   ```json
   {
     "data": [
       {
         "embedding": [0.1, 0.2, ...],
         "index": 0
       }
     ]
   }
   ```

## Common Error Solutions

### Error: "Database connection failed"

**Solution:**
```bash
# Check Supabase configuration
echo $SUPABASE_URL
echo $SUPABASE_API_KEY

# Test connectivity
curl -H "apikey: $SUPABASE_API_KEY" "$SUPABASE_URL/rest/v1/"

# Check network connectivity
ping your-project.supabase.co

# Verify API key permissions in Supabase dashboard
```

### Error: "Cache operation failed"

**Solution:**
```bash
# Check cache configuration
curl http://localhost:8080/api/v1/cache/stats

# Clear cache if corrupted
curl -X POST http://localhost:8080/api/v1/cache/clear

# Restart service if persistent
systemctl restart semantic-text-processor

# Check memory availability
free -h
```

### Error: "Embedding generation timeout"

**Solution:**
```bash
# Check API rate limits
grep -i "rate limit" /var/log/semantic-text-processor.log

# Increase timeout
export EMBEDDING_TIMEOUT=60s

# Reduce batch size
export EMBEDDING_BATCH_SIZE=10

# Check API key quota
curl -H "Authorization: Bearer $EMBEDDING_API_KEY" \
     https://api.openai.com/v1/usage
```

### Error: "Search returned no results"

**Solution:**
```bash
# Check if embeddings exist
curl -X POST http://localhost:8080/api/v1/admin/query \
  -d '{"query": "SELECT COUNT(*) FROM embeddings"}'

# Lower similarity threshold
curl -X POST /api/v1/search/semantic \
  -d '{"query": "your query", "min_similarity": 0.5}'

# Try different search types
curl -X POST /api/v1/search/hybrid \
  -d '{"query": "your query", "limit": 10}'

# Check chunk content
curl http://localhost:8080/api/v1/chunks | head -20
```

### Error: "Template instance creation failed"

**Solution:**
```bash
# Verify template exists
curl http://localhost:8080/api/v1/templates

# Check slot names match
curl http://localhost:8080/api/v1/templates/template-id

# Validate slot values format
curl -X POST /api/v1/templates/template-id/instances \
  -H "Content-Type: application/json" \
  -d '{
    "instance_name": "Test Instance",
    "slot_values": {
      "slot1": "value1",
      "slot2": "value2"
    }
  }'
```

## Best Practices

### Performance Best Practices

1. **Caching Strategy:**
   ```bash
   # Enable caching for all environments
   export CACHE_ENABLED=true
   export CACHE_MAX_SIZE=2000
   export CACHE_DEFAULT_TTL=3600

   # Monitor cache hit rate
   curl http://localhost:8080/api/v1/cache/stats
   # Target: hit_rate > 0.8
   ```

2. **Search Optimization:**
   ```bash
   # Use appropriate similarity thresholds
   # - High precision: min_similarity >= 0.8
   # - Balanced: min_similarity = 0.7
   # - High recall: min_similarity <= 0.6

   # Limit result sets
   # - Interactive use: limit = 10-20
   # - Batch processing: limit = 50-100
   ```

3. **Resource Management:**
   ```bash
   # Monitor resource usage
   curl http://localhost:8080/api/v1/metrics

   # Set appropriate limits
   export MAX_CONCURRENT_REQUESTS=100
   export REQUEST_TIMEOUT=30s
   export MAX_REQUEST_SIZE=10MB
   ```

### Data Management Best Practices

1. **Content Organization:**
   ```bash
   # Use descriptive titles
   # Organize by topic/project
   # Apply consistent tagging
   # Maintain chunk hierarchy
   ```

2. **Template Design:**
   ```bash
   # Keep templates focused and specific
   # Use clear slot names
   # Provide default values
   # Document template purpose
   ```

3. **Backup Strategy:**
   ```bash
   # Daily automated backups
   # Test restore procedures
   # Monitor backup integrity
   # Keep multiple backup generations
   ```

### Security Best Practices

1. **Access Control:**
   ```bash
   # Use environment-specific API keys
   # Implement rate limiting
   # Monitor access patterns
   # Regular security audits
   ```

2. **Data Protection:**
   ```bash
   # Encrypt sensitive data before processing
   # Use HTTPS for all communications
   # Implement data retention policies
   # Regular security updates
   ```

### Monitoring Best Practices

1. **Key Metrics:**
   ```bash
   # Performance: Response time P95 < 250ms
   # Reliability: Error rate < 1%
   # Efficiency: Cache hit rate > 80%
   # Capacity: CPU/Memory usage < 80%
   ```

2. **Alerting:**
   ```bash
   # Set up alerts for critical thresholds
   # Use escalation procedures
   # Document incident response
   # Regular alert testing
   ```

---

**Need additional help?**
- Check the [API Reference](api_reference.md) for detailed endpoint documentation
- Review the [Operations Manual](operations.md) for operational procedures
- Consult the [Performance Guide](performance_tuning_guide.md) for optimization techniques

**Contact:**
- Create an issue in the project repository
- Check community forums
- Review documentation updates