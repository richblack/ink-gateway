# Operations Runbook - Updated for Unified Chunk System

This document provides operational procedures and troubleshooting guides for the Semantic Text Processor application with the new unified chunk system.

## Quick Reference

### Service Endpoints

- **Health Check**: `GET /api/v1/health`
- **Metrics**: `GET /api/v1/metrics`
- **Cache Stats**: `GET /api/v1/cache/stats`
- **Cache Clear**: `POST /api/v1/cache/clear`
- **Text Operations**: `GET|POST|PUT|DELETE /api/v1/texts/*`
- **Chunk Operations**: `GET|POST|PUT|DELETE /api/v1/chunks/*`
- **Template Operations**: `GET|POST /api/v1/templates/*`
- **Tag Operations**: `GET|POST|DELETE /api/v1/chunks/*/tags/*`
- **Search Operations**: `POST /api/v1/search/*`

### Key Configuration

```bash
# Critical Environment Variables
SUPABASE_URL=https://your-project.supabase.co
SUPABASE_API_KEY=your-api-key
LLM_API_KEY=your-llm-key
EMBEDDING_API_KEY=your-embedding-key
LOG_LEVEL=info

# Cache Configuration
CACHE_ENABLED=true
CACHE_MAX_SIZE=1000
CACHE_DEFAULT_TTL=3600

# Performance Monitoring
METRICS_ENABLED=true
MONITORING_ENABLED=true
SLOW_QUERY_THRESHOLD=500ms

# Feature Flags
USE_UNIFIED_HANDLERS=true
USE_ENHANCED_SEARCH=true
```

## Health Monitoring

### Health Check Response

**Healthy System**:
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

**Unhealthy System**:
```json
{
  "status": "unhealthy",
  "timestamp": "2024-01-15T10:30:00Z",
  "uptime": "2h30m15s",
  "version": "1.0.0",
  "components": {
    "database": {
      "name": "database",
      "status": "unhealthy",
      "message": "Connection timeout",
      "timestamp": "2024-01-15T10:30:00Z",
      "duration": "5s"
    }
  }
}
```

### Health Check Interpretation

| Status | HTTP Code | Action Required |
|--------|-----------|-----------------|
| `healthy` | 200 | None |
| `degraded` | 200 | Monitor closely |
| `unhealthy` | 503 | Immediate action |

## Troubleshooting Procedures

### 1. Service Not Responding

**Symptoms**:
- HTTP requests timeout
- Health check fails
- No response from service

**Investigation Steps**:
1. Check if process is running:
   ```bash
   ps aux | grep semantic-text-processor
   ```

2. Check port availability:
   ```bash
   netstat -tlnp | grep :8080
   ```

3. Check system resources:
   ```bash
   top
   free -h
   df -h
   ```

4. Check application logs:
   ```bash
   tail -f /var/log/semantic-text-processor.log
   ```

**Resolution**:
1. Restart the service:
   ```bash
   systemctl restart semantic-text-processor
   ```

2. If restart fails, check configuration:
   ```bash
   semantic-text-processor --config-check
   ```

### 2. Database Connection Issues

**Symptoms**:
- Database component shows `unhealthy`
- Errors in logs: "database connection failed"
- API requests return 500 errors

**Investigation Steps**:
1. Check Supabase status:
   ```bash
   curl -H "apikey: $SUPABASE_API_KEY" "$SUPABASE_URL/rest/v1/"
   ```

2. Verify environment variables:
   ```bash
   echo $SUPABASE_URL
   echo $SUPABASE_API_KEY
   ```

3. Check network connectivity:
   ```bash
   ping your-project.supabase.co
   nslookup your-project.supabase.co
   ```

**Resolution**:
1. Verify API key is valid and has correct permissions
2. Check Supabase project status in dashboard
3. Restart service if configuration was corrected

### 3. High Memory Usage

**Symptoms**:
- Memory usage > 80%
- Service becomes slow
- Out of memory errors

**Investigation Steps**:
1. Check memory usage:
   ```bash
   free -h
   top -p $(pgrep semantic-text-processor)
   ```

2. Check cache statistics:
   ```bash
   curl http://localhost:8080/api/v1/cache/stats
   ```

3. Review cache configuration:
   ```bash
   echo $CACHE_MAX_SIZE
   echo $CACHE_DEFAULT_TTL
   ```

**Resolution**:
1. Clear cache if necessary:
   ```bash
   curl -X POST http://localhost:8080/api/v1/cache/clear
   ```

2. Adjust cache size:
   ```bash
   export CACHE_MAX_SIZE=500
   systemctl restart semantic-text-processor
   ```

3. Monitor for memory leaks in logs

### 4. High Response Times

**Symptoms**:
- API responses > 5 seconds
- Timeout errors
- Poor user experience

**Investigation Steps**:
1. Check metrics:
   ```bash
   curl http://localhost:8080/api/v1/metrics | jq '.histograms'
   ```

2. Check cache hit rate:
   ```bash
   curl http://localhost:8080/api/v1/cache/stats | jq '.hit_rate'
   ```

3. Check external service response times in logs

**Resolution**:
1. If cache hit rate is low, investigate cache configuration
2. Check external service status (LLM, Embedding APIs)
3. Consider scaling if load is high

### 5. External Service Failures

**Symptoms**:
- LLM or Embedding API errors
- Text processing failures
- Search functionality not working

**Investigation Steps**:
1. Check API key validity:
   ```bash
   curl -H "Authorization: Bearer $LLM_API_KEY" https://api.openai.com/v1/models
   ```

2. Check service status pages
3. Review timeout configurations

**Resolution**:
1. Verify API keys are current
2. Check rate limits and quotas
3. Implement fallback mechanisms if available

## Performance Optimization

### Cache Optimization

**Monitor Cache Performance**:
```bash
# Check cache statistics
curl http://localhost:8080/api/v1/cache/stats

# Expected good performance:
# - hit_rate > 0.7
# - size < max_size * 0.8
# - evictions should be minimal
```

**Optimize Cache Settings**:
```bash
# Increase cache size for better hit rates
export CACHE_MAX_SIZE=2000

# Adjust TTL based on data freshness requirements
export CACHE_DEFAULT_TTL=60m

# Restart service
systemctl restart semantic-text-processor
```

### Database Query Optimization

**Monitor Query Performance**:
- Review slow query logs
- Check database metrics in Supabase dashboard
- Monitor connection pool usage

**Optimization Actions**:
- Add database indexes for frequently queried fields
- Optimize query patterns
- Consider read replicas for high read loads

### Unified Chunk System Operations

**Key Table Structure**:
The unified chunk system consolidates all content into a single `chunks` table with the following key fields:
- `id`: Unique chunk identifier
- `text_id`: Parent text document ID
- `content`: Chunk content
- `is_template`: Boolean flag for template chunks
- `is_slot`: Boolean flag for template slots
- `parent_chunk_id`: Hierarchical parent reference
- `template_chunk_id`: Template reference for instances
- `slot_value`: Value for template slot instances
- `indent_level`: Hierarchical indentation level
- `sequence_number`: Ordering within siblings
- `metadata`: JSON metadata storage

**Monitoring Unified Chunk Operations**:
```bash
# Check chunk distribution
curl "$API_BASE/chunks?page_size=1" | jq '.pagination.total'

# Monitor hierarchical integrity
curl "$API_BASE/chunks/{id}/hierarchy" | jq '.depth, .total_descendants'

# Check template usage
curl "$API_BASE/chunks?is_template=true&page_size=1" | jq '.pagination.total'

# Monitor tag distribution
curl "$API_BASE/chunks/{id}/tags" | jq 'length'
```

**Common Unified Chunk Issues**:

1. **Orphaned Chunks**:
   ```bash
   # Find chunks with invalid parent references
   # This requires database-level query through Supabase
   ```

2. **Template Consistency**:
   ```bash
   # Verify template instances have valid template references
   curl "$API_BASE/templates" | jq '.[] | select(.instances > 0)'
   ```

3. **Hierarchy Depth Issues**:
   ```bash
   # Monitor for excessive nesting (>10 levels usually indicates issues)
   curl "$API_BASE/chunks/{id}/hierarchy" | jq 'select(.depth > 10)'
   ```

## Alerting and Monitoring

### Critical Alerts

Set up alerts for:

1. **Service Down**:
   - Health check returns 503
   - Process not running
   - Port not responding

2. **High Error Rate**:
   - HTTP 5xx errors > 5%
   - Database connection failures
   - External service failures

3. **Performance Degradation**:
   - Response time > 5 seconds
   - Memory usage > 80%
   - Cache hit rate < 50%

### Monitoring Commands

```bash
# Service status
systemctl status semantic-text-processor

# Resource usage
htop
iostat 1
vmstat 1

# Network connections
netstat -an | grep :8080

# Application metrics
curl -s http://localhost:8080/api/v1/metrics | jq '.counters'
curl -s http://localhost:8080/api/v1/health | jq '.status'
```

## Maintenance Procedures

### Regular Maintenance

**Daily**:
- Check service health
- Review error logs
- Monitor resource usage

**Weekly**:
- Review performance metrics
- Check cache efficiency
- Update dependencies if needed

**Monthly**:
- Review and rotate logs
- Performance optimization review
- Security updates

### Log Management

**Log Rotation**:
```bash
# Configure logrotate
cat > /etc/logrotate.d/semantic-text-processor << EOF
/var/log/semantic-text-processor.log {
    daily
    rotate 30
    compress
    delaycompress
    missingok
    notifempty
    postrotate
        systemctl reload semantic-text-processor
    endscript
}
EOF
```

**Log Analysis**:
```bash
# Error analysis
grep -i error /var/log/semantic-text-processor.log | tail -20

# Performance analysis
grep "duration" /var/log/semantic-text-processor.log | \
  jq -r '.fields.duration' | sort -n | tail -10

# Request analysis
grep "HTTP request" /var/log/semantic-text-processor.log | \
  jq -r '.fields.path' | sort | uniq -c | sort -nr
```

## Emergency Procedures

### Service Recovery

1. **Immediate Actions**:
   ```bash
   # Stop service
   systemctl stop semantic-text-processor
   
   # Check for core dumps
   ls -la /var/crash/
   
   # Clear cache if corrupted
   rm -rf /tmp/semantic-text-processor-cache/*
   
   # Start service
   systemctl start semantic-text-processor
   ```

2. **Rollback Procedure**:
   ```bash
   # Stop current version
   systemctl stop semantic-text-processor
   
   # Restore previous version
   cp /backup/semantic-text-processor-previous /usr/local/bin/semantic-text-processor
   
   # Start service
   systemctl start semantic-text-processor
   ```

### Data Recovery

1. **Cache Recovery**:
   - Cache is ephemeral, will rebuild automatically
   - Clear cache if corrupted: `POST /api/v1/cache/clear`

2. **Configuration Recovery**:
   - Restore from backup: `/backup/config/`
   - Verify environment variables
   - Restart service

## Contact Information

### Escalation Path

1. **Level 1**: Application logs and basic troubleshooting
2. **Level 2**: System administrator for infrastructure issues
3. **Level 3**: Development team for application bugs

### External Dependencies

- **Supabase Support**: [Supabase Support](https://supabase.com/support)
- **OpenAI Status**: [OpenAI Status Page](https://status.openai.com/)
- **Infrastructure Provider**: Contact your cloud provider

## Useful Commands Reference

```bash
# Service management
systemctl status semantic-text-processor
systemctl start semantic-text-processor
systemctl stop semantic-text-processor
systemctl restart semantic-text-processor

# Log viewing
journalctl -u semantic-text-processor -f
tail -f /var/log/semantic-text-processor.log

# Health checks
curl http://localhost:8080/api/v1/health
curl http://localhost:8080/api/v1/metrics

# Process monitoring
ps aux | grep semantic-text-processor
pgrep -f semantic-text-processor

# Resource monitoring
top -p $(pgrep semantic-text-processor)
lsof -p $(pgrep semantic-text-processor)
```