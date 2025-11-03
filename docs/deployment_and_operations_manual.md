# Semantic Text Processor - Deployment and Operations Manual

## Table of Contents

1. [System Overview](#system-overview)
2. [Prerequisites](#prerequisites)
3. [Production Deployment](#production-deployment)
4. [Configuration Management](#configuration-management)
5. [Monitoring and Alerting](#monitoring-and-alerting)
6. [Backup and Recovery](#backup-and-recovery)
7. [Security Considerations](#security-considerations)
8. [Troubleshooting](#troubleshooting)
9. [Maintenance Procedures](#maintenance-procedures)
10. [Performance Optimization](#performance-optimization)

## System Overview

The Semantic Text Processor is a Go-based microservice that provides:
- Text processing and chunking capabilities
- Embedding generation and storage
- Knowledge graph extraction
- Semantic search functionality
- Template and tag management

### Architecture Components

```
┌─────────────────────────────────────────────────────────────────┐
│                        Load Balancer (Nginx)                   │
└─────────────────────────┬───────────────────────────────────────┘
                          │
┌─────────────────────────┴───────────────────────────────────────┐
│                 Semantic Text Processor                        │
│                      (Go Application)                          │
└─────────────────┬───────────────────────┬─────────────────────┘
                  │                       │
┌─────────────────┴─────────────┐  ┌─────┴──────────────────────┐
│        PostgreSQL            │  │        Redis Cache         │
│     (with pgvector)          │  │                            │
└───────────────────────────────┘  └────────────────────────────┘
```

## Prerequisites

### Hardware Requirements

**Minimum Production Environment:**
- CPU: 4 cores
- Memory: 8GB RAM
- Storage: 100GB SSD
- Network: 1Gbps

**Recommended Production Environment:**
- CPU: 8 cores
- Memory: 16GB RAM
- Storage: 500GB SSD (with backup storage)
- Network: 10Gbps

### Software Requirements

- Docker Engine 20.10+
- Docker Compose 2.0+
- Git
- curl
- jq
- OpenSSL (for SSL certificates)

### External Services

- OpenAI API or compatible LLM service
- Embedding service (OpenAI or local)
- SMTP server (for alerts)
- AWS S3 (for backups, optional)

## Production Deployment

### Initial Setup

1. **Clone the repository:**
```bash
git clone https://github.com/your-org/semantic-text-processor.git
cd semantic-text-processor
```

2. **Set up environment:**
```bash
cp .env.example .env.production
# Edit .env.production with your configuration
```

3. **Generate SSL certificates:**
```bash
# Self-signed (for testing)
openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
  -keyout deployments/nginx/ssl/private.key \
  -out deployments/nginx/ssl/certificate.crt

# Or use Let's Encrypt for production
certbot certonly --standalone -d your-domain.com
```

4. **Initialize database:**
```bash
# Create database schema
docker-compose -f deployments/production/docker-compose.prod.yml \
  run --rm semantic-processor ./scripts/migrate.sh
```

5. **Start services:**
```bash
docker-compose -f deployments/production/docker-compose.prod.yml up -d
```

### Environment Configuration

Create `/deployments/production/.env.production`:

```bash
# Application Configuration
VERSION=latest
ENV=production
LOG_LEVEL=info

# Database Configuration
DB_HOST=postgres
DB_PORT=5432
DB_NAME=semantic_processor_prod
DB_USER=semantic_user
DB_PASSWORD=your_secure_password

# Service URLs
LLM_SERVICE_URL=https://api.openai.com/v1
EMBEDDING_SERVICE_URL=https://api.openai.com/v1

# Security
ENCRYPTION_KEY=your_encryption_key_32_chars
JWT_SECRET=your_jwt_secret

# Monitoring
GRAFANA_PASSWORD=your_grafana_password

# Backup Configuration
S3_BUCKET=your-backup-bucket
AWS_REGION=us-west-2
RETENTION_DAYS=30

# Performance Tuning
MAX_REQUEST_SIZE=10MB
CONNECTION_TIMEOUT=30s
READ_TIMEOUT=60s
WRITE_TIMEOUT=60s
```

### Docker Compose Override

Create `docker-compose.override.yml` for environment-specific customizations:

```yaml
version: '3.8'

services:
  semantic-processor:
    environment:
      - CUSTOM_ENV_VAR=value
    volumes:
      - ./custom-config:/app/custom-config:ro

  postgres:
    volumes:
      - ./custom-init:/docker-entrypoint-initdb.d:ro
```

### Deployment Verification

After deployment, verify all components:

```bash
# Check service status
docker-compose -f deployments/production/docker-compose.prod.yml ps

# Verify health endpoints
curl http://localhost/health
curl http://localhost/ready

# Check logs
docker-compose -f deployments/production/docker-compose.prod.yml logs -f
```

## Configuration Management

### Application Configuration

Main configuration file: `config/production.yaml`

```yaml
server:
  host: "0.0.0.0"
  port: 8080
  read_timeout: "60s"
  write_timeout: "60s"
  idle_timeout: "120s"

database:
  host: "postgres"
  port: 5432
  name: "semantic_processor_prod"
  ssl_mode: "prefer"
  max_connections: 25
  max_idle_connections: 5
  connection_lifetime: "1h"

cache:
  redis_url: "redis://redis:6379"
  default_ttl: "1h"
  max_connections: 10

llm:
  provider: "openai"
  model: "gpt-3.5-turbo"
  max_tokens: 4096
  temperature: 0.1
  timeout: "30s"

embedding:
  provider: "openai"
  model: "text-embedding-ada-002"
  dimensions: 1536
  timeout: "30s"

processing:
  chunk_size: 1000
  chunk_overlap: 200
  max_concurrent_jobs: 5
  job_timeout: "5m"

search:
  max_results: 100
  similarity_threshold: 0.7
  enable_hybrid_search: true

metrics:
  enabled: true
  path: "/metrics"
  namespace: "semantic_processor"

logging:
  level: "info"
  format: "json"
  output: "/app/logs/app.log"
```

### Nginx Configuration

File: `deployments/nginx/nginx.conf`

```nginx
events {
    worker_connections 1024;
}

http {
    upstream semantic_processor {
        server semantic-processor:8080;
    }

    # Rate limiting
    limit_req_zone $binary_remote_addr zone=api:10m rate=10r/s;

    # SSL Configuration
    ssl_certificate /etc/nginx/ssl/certificate.crt;
    ssl_certificate_key /etc/nginx/ssl/private.key;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers ECDHE-RSA-AES256-GCM-SHA512:DHE-RSA-AES256-GCM-SHA512;

    server {
        listen 80;
        server_name _;
        return 301 https://$host$request_uri;
    }

    server {
        listen 443 ssl;
        server_name _;

        # Security headers
        add_header X-Frame-Options DENY;
        add_header X-Content-Type-Options nosniff;
        add_header X-XSS-Protection "1; mode=block";

        # API endpoints
        location /api/ {
            limit_req zone=api burst=20 nodelay;
            proxy_pass http://semantic_processor;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;
            proxy_timeout 60s;
        }

        # Health checks
        location /health {
            proxy_pass http://semantic_processor;
            access_log off;
        }

        location /ready {
            proxy_pass http://semantic_processor;
            access_log off;
        }

        # Metrics (internal only)
        location /metrics {
            proxy_pass http://semantic_processor;
            allow 172.20.0.0/16;  # Docker network
            deny all;
        }
    }
}
```

## Monitoring and Alerting

### Prometheus Configuration

File: `deployments/monitoring/prometheus.yml`

```yaml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

rule_files:
  - "alert_rules.yml"

alerting:
  alertmanagers:
    - static_configs:
        - targets:
          - alertmanager:9093

scrape_configs:
  - job_name: 'semantic-processor'
    static_configs:
      - targets: ['semantic-processor:8080']
    metrics_path: '/metrics'
    scrape_interval: 10s

  - job_name: 'postgres'
    static_configs:
      - targets: ['postgres_exporter:9187']

  - job_name: 'redis'
    static_configs:
      - targets: ['redis_exporter:9121']

  - job_name: 'nginx'
    static_configs:
      - targets: ['nginx_exporter:9113']

  - job_name: 'node'
    static_configs:
      - targets: ['node_exporter:9100']
```

### Alert Rules

File: `deployments/monitoring/alert_rules.yml`

```yaml
groups:
  - name: semantic_processor_alerts
    rules:
      - alert: HighErrorRate
        expr: rate(http_requests_total{status=~"5.."}[5m]) > 0.1
        for: 2m
        labels:
          severity: critical
        annotations:
          summary: "High error rate detected"
          description: "Error rate is {{ $value }} errors per second"

      - alert: HighResponseTime
        expr: histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m])) > 1
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High response time"
          description: "95th percentile response time is {{ $value }}s"

      - alert: DatabaseConnectionFailure
        expr: up{job="postgres"} == 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "Database connection failure"
          description: "PostgreSQL database is not reachable"

      - alert: HighMemoryUsage
        expr: (container_memory_usage_bytes / container_spec_memory_limit_bytes) > 0.9
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High memory usage"
          description: "Memory usage is {{ $value | humanizePercentage }}"

      - alert: DiskSpaceLow
        expr: (node_filesystem_free_bytes / node_filesystem_size_bytes) < 0.1
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "Disk space low"
          description: "Free disk space is {{ $value | humanizePercentage }}"
```

### Grafana Dashboards

Key dashboards to import:
- Application Performance Monitoring
- Database Performance
- Infrastructure Monitoring
- Error Tracking
- Business Metrics

## Backup and Recovery

### Automated Backup Setup

1. **Configure backup script:**
```bash
cp config/backup.conf.example config/backup.conf
# Edit backup configuration
```

2. **Set up cron job:**
```bash
# Add to crontab
0 2 * * * /app/scripts/backup_and_recovery.sh backup
0 3 * * 0 /app/scripts/backup_and_recovery.sh cleanup
```

3. **Test backup and restore:**
```bash
# Create test backup
./scripts/backup_and_recovery.sh backup

# Verify backup
./scripts/backup_and_recovery.sh verify /backups/latest

# Test restore (to test database)
./scripts/backup_and_recovery.sh restore /backups/latest test_db
```

### Disaster Recovery Procedures

#### Complete System Recovery

1. **Prepare new environment:**
```bash
# Install Docker and dependencies
# Clone repository
# Copy configuration files
```

2. **Restore from backup:**
```bash
# Download backup from S3
aws s3 cp s3://your-backup-bucket/semantic-processor/backups/latest.tar.gz.gpg ./

# Restore system
./scripts/backup_and_recovery.sh restore latest.tar.gz.gpg full
```

3. **Verify recovery:**
```bash
# Check all services are running
docker-compose ps

# Verify application health
curl http://localhost/health

# Run integration tests
./scripts/run_integration_tests.sh
```

#### Database-Only Recovery

```bash
# Restore only database
./scripts/backup_and_recovery.sh restore backup.dump database

# Verify database integrity
./scripts/verify_database.sh
```

### Recovery Time Objectives (RTO/RPO)

- **RTO (Recovery Time Objective):** 4 hours
- **RPO (Recovery Point Objective):** 1 hour
- **Backup Frequency:** Daily full, hourly incremental
- **Backup Retention:** 30 days local, 90 days in S3

## Security Considerations

### Network Security

1. **Firewall Rules:**
```bash
# Allow only necessary ports
ufw allow 22/tcp    # SSH
ufw allow 80/tcp    # HTTP
ufw allow 443/tcp   # HTTPS
ufw deny by default incoming
```

2. **Internal Network Isolation:**
- Use Docker networks for service isolation
- Restrict database access to application containers only
- Implement network segmentation

### Application Security

1. **Environment Variables:**
- Store secrets in environment variables
- Use Docker secrets for sensitive data
- Rotate credentials regularly

2. **API Security:**
- Implement rate limiting
- Use API authentication/authorization
- Input validation and sanitization
- SQL injection prevention

### Data Security

1. **Encryption:**
- Enable TLS/SSL for all communications
- Encrypt data at rest
- Encrypt backup files

2. **Access Control:**
- Implement role-based access control
- Use least privilege principle
- Regular security audits

## Troubleshooting

### Common Issues

#### 1. Application Won't Start

**Symptoms:**
- Container exits immediately
- Error in logs about configuration

**Diagnosis:**
```bash
# Check container logs
docker-compose logs semantic-processor

# Check configuration
docker-compose config

# Verify environment variables
docker-compose run --rm semantic-processor env
```

**Resolution:**
- Verify all required environment variables are set
- Check configuration file syntax
- Ensure database is accessible

#### 2. Database Connection Issues

**Symptoms:**
- "Connection refused" errors
- Timeouts connecting to database

**Diagnosis:**
```bash
# Check database status
docker-compose ps postgres

# Test database connection
docker-compose exec postgres psql -U $DB_USER -d $DB_NAME -c "SELECT 1;"

# Check network connectivity
docker-compose exec semantic-processor nc -zv postgres 5432
```

**Resolution:**
- Verify database credentials
- Check network configuration
- Ensure database is fully initialized

#### 3. High Memory Usage

**Symptoms:**
- Container restarts frequently
- Out of memory errors

**Diagnosis:**
```bash
# Check memory usage
docker stats

# Analyze memory allocation
docker-compose exec semantic-processor go tool pprof http://localhost:8080/debug/pprof/heap
```

**Resolution:**
- Increase memory limits
- Optimize application code
- Implement memory profiling

#### 4. Slow Performance

**Symptoms:**
- High response times
- Timeout errors

**Diagnosis:**
```bash
# Check CPU and memory usage
docker stats

# Analyze database performance
docker-compose exec postgres psql -U $DB_USER -d $DB_NAME -c "
SELECT query, mean_time, calls
FROM pg_stat_statements
ORDER BY mean_time DESC
LIMIT 10;"

# Check application metrics
curl http://localhost/metrics | grep -E "(response_time|request_duration)"
```

**Resolution:**
- Add database indexes
- Optimize queries
- Increase resource allocation
- Implement caching

### Log Analysis

#### Application Logs
```bash
# Real-time logs
docker-compose logs -f semantic-processor

# Filter by log level
docker-compose logs semantic-processor | grep ERROR

# Export logs for analysis
docker-compose logs --no-color semantic-processor > app.log
```

#### Database Logs
```bash
# PostgreSQL query logs
docker-compose exec postgres tail -f /var/log/postgresql/postgresql.log

# Slow query analysis
docker-compose exec postgres psql -U $DB_USER -d $DB_NAME -c "
SELECT query, total_time, mean_time, calls
FROM pg_stat_statements
WHERE mean_time > 1000
ORDER BY total_time DESC;"
```

### Performance Monitoring

#### Key Metrics to Monitor

1. **Application Metrics:**
   - Request rate (requests/second)
   - Response time (95th percentile)
   - Error rate (%)
   - Active connections

2. **Database Metrics:**
   - Connection count
   - Query execution time
   - Cache hit ratio
   - Lock waits

3. **System Metrics:**
   - CPU utilization
   - Memory usage
   - Disk I/O
   - Network throughput

#### Performance Baselines

- Response time: < 200ms (95th percentile)
- Throughput: > 100 requests/second
- Error rate: < 1%
- Database connections: < 80% of max

## Maintenance Procedures

### Regular Maintenance Tasks

#### Daily
- Monitor system health and alerts
- Check backup completion
- Review error logs

#### Weekly
- Analyze performance metrics
- Update security patches
- Review capacity planning

#### Monthly
- Test backup and recovery procedures
- Update dependencies
- Security audit
- Performance optimization review

### Update Procedures

#### Application Updates

1. **Prepare update:**
```bash
# Create backup
./scripts/backup_and_recovery.sh backup

# Download new version
docker pull semantic-processor:new-version
```

2. **Deploy update:**
```bash
# Update docker-compose file
sed -i 's/semantic-processor:old-version/semantic-processor:new-version/' docker-compose.prod.yml

# Deploy with zero downtime
docker-compose up -d --no-deps semantic-processor
```

3. **Verify update:**
```bash
# Check version
curl http://localhost/health | jq '.version'

# Run health checks
./scripts/health_check.sh

# Monitor for issues
docker-compose logs -f semantic-processor
```

#### Database Migrations

```bash
# Run migrations
docker-compose run --rm semantic-processor ./scripts/migrate.sh

# Verify migration
docker-compose exec postgres psql -U $DB_USER -d $DB_NAME -c "
SELECT version FROM schema_migrations ORDER BY version DESC LIMIT 1;"
```

### Scaling Procedures

#### Horizontal Scaling

```bash
# Scale application instances
docker-compose up -d --scale semantic-processor=3

# Update load balancer configuration
# Add health checks for new instances
```

#### Vertical Scaling

```bash
# Update resource limits in docker-compose.yml
# Restart services with new limits
docker-compose up -d --force-recreate semantic-processor
```

## Performance Optimization

### Database Optimization

1. **Index Optimization:**
```sql
-- Create indexes for common queries
CREATE INDEX CONCURRENTLY idx_chunks_text_id ON chunks(text_id);
CREATE INDEX CONCURRENTLY idx_embeddings_vector ON embeddings USING ivfflat (vector vector_cosine_ops);

-- Monitor index usage
SELECT schemaname, tablename, indexname, idx_scan, idx_tup_read, idx_tup_fetch
FROM pg_stat_user_indexes
ORDER BY idx_scan DESC;
```

2. **Connection Pooling:**
```yaml
# Configure in production.yaml
database:
  max_connections: 25
  max_idle_connections: 5
  connection_lifetime: "1h"
```

3. **Query Optimization:**
```sql
-- Enable query statistics
ALTER SYSTEM SET shared_preload_libraries = 'pg_stat_statements';
ALTER SYSTEM SET pg_stat_statements.track = 'all';

-- Analyze slow queries
SELECT query, total_time, mean_time, calls
FROM pg_stat_statements
ORDER BY total_time DESC
LIMIT 10;
```

### Application Optimization

1. **Caching Configuration:**
```yaml
cache:
  redis_url: "redis://redis:6379"
  default_ttl: "1h"
  embedding_cache_ttl: "24h"
  search_cache_ttl: "30m"
```

2. **Concurrent Processing:**
```yaml
processing:
  max_concurrent_jobs: 5
  chunk_batch_size: 100
  embedding_batch_size: 50
```

3. **Memory Management:**
```yaml
# Set appropriate memory limits
deploy:
  resources:
    limits:
      memory: 2G
    reservations:
      memory: 1G
```

### Infrastructure Optimization

1. **Resource Allocation:**
- Monitor CPU and memory usage
- Adjust container resource limits
- Implement auto-scaling policies

2. **Network Optimization:**
- Use CDN for static content
- Implement connection keep-alive
- Optimize DNS resolution

3. **Storage Optimization:**
- Use SSD storage for database
- Implement database partitioning
- Regular vacuum and analyze operations

This operations manual provides comprehensive guidance for deploying, monitoring, and maintaining the Semantic Text Processor in production environments. Regular review and updates of these procedures ensure continued system reliability and performance.