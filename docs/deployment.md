# Deployment Guide

This document provides comprehensive guidance for deploying the Semantic Text Processor application.

## Prerequisites

### System Requirements
- Go 1.21 or higher
- Docker (optional, for containerized deployment)
- Access to Supabase instance
- LLM API access (OpenAI, Anthropic, etc.)
- Embedding service API access

### Environment Variables

Create a `.env` file based on `.env.example`:

```bash
# Server Configuration
SERVER_PORT=8080
SERVER_READ_TIMEOUT=30s
SERVER_WRITE_TIMEOUT=30s
SERVER_IDLE_TIMEOUT=60s

# Database Configuration
SUPABASE_URL=https://your-project.supabase.co
SUPABASE_API_KEY=your-supabase-api-key

# LLM Service Configuration
LLM_API_KEY=your-llm-api-key
LLM_ENDPOINT=https://api.openai.com/v1
LLM_TIMEOUT=60s

# Embedding Service Configuration
EMBEDDING_API_KEY=your-embedding-api-key
EMBEDDING_ENDPOINT=https://api.openai.com/v1
EMBEDDING_TIMEOUT=30s

# Logging Configuration
LOG_LEVEL=info
LOG_FORMAT=json

# Cache Configuration
CACHE_ENABLED=true
CACHE_MAX_SIZE=1000
CACHE_CLEANUP_INTERVAL=5m
CACHE_DEFAULT_TTL=30m

# Performance Monitoring
METRICS_ENABLED=true
METRICS_ENDPOINT=/metrics
MONITORING_ENABLED=true
```

## Build and Deployment

### Local Development

1. **Install Dependencies**
   ```bash
   make deps
   ```

2. **Run Tests**
   ```bash
   make test
   ```

3. **Start the Application**
   ```bash
   make run
   ```

### Production Build

1. **Build for Production**
   ```bash
   make build-prod
   ```

2. **Run the Binary**
   ```bash
   ./bin/semantic-text-processor-linux
   ```

### Docker Deployment

1. **Create Dockerfile**
   ```dockerfile
   FROM golang:1.21-alpine AS builder
   
   WORKDIR /app
   COPY go.mod go.sum ./
   RUN go mod download
   
   COPY . .
   RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .
   
   FROM alpine:latest
   RUN apk --no-cache add ca-certificates
   WORKDIR /root/
   
   COPY --from=builder /app/main .
   COPY --from=builder /app/.env.example .env
   
   EXPOSE 8080
   CMD ["./main"]
   ```

2. **Build and Run**
   ```bash
   docker build -t semantic-text-processor .
   docker run -p 8080:8080 --env-file .env semantic-text-processor
   ```

### Docker Compose

```yaml
version: '3.8'

services:
  semantic-text-processor:
    build: .
    ports:
      - "8080:8080"
    environment:
      - SERVER_PORT=8080
      - SUPABASE_URL=${SUPABASE_URL}
      - SUPABASE_API_KEY=${SUPABASE_API_KEY}
      - LLM_API_KEY=${LLM_API_KEY}
      - EMBEDDING_API_KEY=${EMBEDDING_API_KEY}
      - LOG_LEVEL=info
      - CACHE_ENABLED=true
      - METRICS_ENABLED=true
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/api/v1/health"]
      interval: 30s
      timeout: 10s
      retries: 3
```

## Monitoring and Observability

### Health Checks

The application provides comprehensive health checks:

- **Basic Health**: `GET /api/v1/health`
- **Component Health**: Individual component status
- **System Metrics**: `GET /api/v1/metrics`
- **Cache Statistics**: `GET /api/v1/cache/stats`

### Logging

The application uses structured JSON logging:

```json
{
  "timestamp": "2024-01-15T10:30:00Z",
  "level": "info",
  "message": "HTTP request",
  "fields": {
    "method": "POST",
    "path": "/api/v1/texts",
    "status_code": 200,
    "duration": "150ms",
    "remote_addr": "192.168.1.100"
  }
}
```

### Metrics

Available metrics include:

- **HTTP Metrics**:
  - `http.requests.total` - Total HTTP requests
  - `http.request.duration` - Request duration histogram
  - `http.requests.errors` - Error count
  - `http.requests.slow` - Slow requests (>1s)

- **Search Metrics**:
  - `search.semantic.requests` - Semantic search requests
  - `search.semantic.duration` - Search duration
  - `search.semantic.errors` - Search errors

- **Cache Metrics**:
  - Cache hit rate
  - Cache size and utilization
  - Cache evictions

### Performance Monitoring

Monitor these key performance indicators:

1. **Response Times**
   - API endpoint response times
   - Database query times
   - External service call times

2. **Throughput**
   - Requests per second
   - Text processing rate
   - Search queries per second

3. **Error Rates**
   - HTTP error rates
   - Service failure rates
   - Timeout rates

4. **Resource Usage**
   - Memory usage
   - CPU utilization
   - Cache hit rates

## Security Considerations

### API Security

1. **Authentication**: Implement API key authentication
2. **Rate Limiting**: Configure rate limits for API endpoints
3. **Input Validation**: Validate all input data
4. **CORS**: Configure CORS policies appropriately

### Environment Security

1. **Secrets Management**: Use secure secret management
2. **Network Security**: Configure firewalls and network policies
3. **TLS/SSL**: Use HTTPS in production
4. **Access Control**: Implement proper access controls

## Scaling and Performance

### Horizontal Scaling

The application is designed to be stateless and can be scaled horizontally:

1. **Load Balancer**: Use a load balancer to distribute traffic
2. **Multiple Instances**: Run multiple application instances
3. **Database Scaling**: Scale Supabase as needed
4. **Cache Scaling**: Consider distributed caching for high load

### Performance Optimization

1. **Caching Strategy**:
   - Enable caching for frequently accessed data
   - Configure appropriate TTL values
   - Monitor cache hit rates

2. **Database Optimization**:
   - Use database indexes effectively
   - Optimize query patterns
   - Consider read replicas for high read loads

3. **External Service Optimization**:
   - Implement connection pooling
   - Use appropriate timeouts
   - Implement circuit breakers

## Troubleshooting

### Common Issues

1. **Database Connection Issues**
   - Check Supabase URL and API key
   - Verify network connectivity
   - Check database health

2. **External Service Failures**
   - Verify API keys and endpoints
   - Check service status
   - Review timeout configurations

3. **Performance Issues**
   - Monitor resource usage
   - Check cache hit rates
   - Review slow query logs

### Debugging

1. **Enable Debug Logging**:
   ```bash
   LOG_LEVEL=debug
   ```

2. **Check Health Endpoints**:
   ```bash
   curl http://localhost:8080/api/v1/health
   curl http://localhost:8080/api/v1/metrics
   ```

3. **Monitor Application Logs**:
   ```bash
   tail -f application.log | jq '.'
   ```

## Backup and Recovery

### Data Backup

1. **Database Backup**: Use Supabase backup features
2. **Configuration Backup**: Backup environment configurations
3. **Application State**: Consider any persistent application state

### Disaster Recovery

1. **Recovery Procedures**: Document recovery procedures
2. **Backup Testing**: Regularly test backup restoration
3. **Monitoring**: Monitor backup processes

## Maintenance

### Regular Maintenance Tasks

1. **Log Rotation**: Configure log rotation
2. **Cache Cleanup**: Monitor cache usage
3. **Metrics Cleanup**: Archive old metrics data
4. **Security Updates**: Keep dependencies updated

### Monitoring Checklist

- [ ] Health checks are responding
- [ ] Error rates are within acceptable limits
- [ ] Response times are acceptable
- [ ] Cache hit rates are optimal
- [ ] Resource usage is normal
- [ ] External services are healthy
- [ ] Logs are being generated correctly
- [ ] Metrics are being collected

## Support and Maintenance

For ongoing support and maintenance:

1. **Documentation**: Keep deployment documentation updated
2. **Runbooks**: Create operational runbooks
3. **Alerting**: Set up appropriate alerts
4. **On-call Procedures**: Establish on-call procedures