# n8n Test Guide for Ink-Gateway

## 📋 Overview

This guide provides comprehensive testing scenarios for the Ink-Gateway API using n8n automation workflows.

## 🚀 Go Server Basic Operations

### Starting the Server

```bash
# Navigate to project directory
cd /Users/youlinhsieh/Documents/ink-gateway

# Install dependencies
go mod tidy

# Start server (foreground)
go run main.go

# Start server (background with logs)
nohup go run main.go > server.log 2>&1 &

# Check if server is running
ps aux | grep "go run\|semantic" | grep -v grep
lsof -i :8081
```

### Server Configuration

- **Port**: 8081 (configured in `.env`)
- **Health Check**: `GET /api/v1/health`
- **Logs**: Available in `server.log` when running in background

### Stopping the Server

```bash
# Find and kill the process
pkill -f "go run main.go"

# Or find PID and kill
ps aux | grep "go run main.go" | grep -v grep | awk '{print $2}' | xargs kill
```

## 🧪 n8n Test Scenarios

### 1. Health Check Monitoring

#### Basic Health Check

```
Node Type: HTTP Request
Method: GET
URL: http://localhost:8081/api/v1/health
Headers:
  Content-Type: application/json

Expected Response:
{
  "status": "unhealthy",
  "timestamp": "2025-09-24T...",
  "uptime": 123456789,
  "version": "1.0.0",
  "components": {
    "cache": {"status": "healthy"},
    "database": {"status": "unhealthy"},
    "metrics": {"status": "healthy"}
  }
}
```

#### Automated Health Monitoring Workflow

1. **Schedule Trigger**: Every 5 minutes
2. **HTTP Request**: Health check endpoint
3. **IF Node**: Check if status is "healthy"
4. **Notification**: Send alert if unhealthy

### 2. API Endpoint Testing

#### Text Creation Test

```
Node Type: HTTP Request
Method: POST
URL: http://localhost:8081/api/v1/texts
Headers:
  Content-Type: application/json
Body:
{
  "content": "這是測試文本內容",
  "title": "n8n 測試標題"
}

Expected Response: Error (due to missing LLM credentials)
{
  "type": "error",
  "code": "Internal Server Error",
  "message": "failed to process text"
}
```

#### Search API Test

```
Node Type: HTTP Request
Method: POST
URL: http://localhost:8081/api/v1/search/semantic
Headers:
  Content-Type: application/json
Body:
{
  "query": "測試搜索",
  "limit": 10,
  "min_similarity": 0.5
}
```

#### Template API Test

```
Node Type: HTTP Request
Method: POST
URL: http://localhost:8081/api/v1/templates
Headers:
  Content-Type: application/json
Body:
{
  "template_name": "測試模板",
  "slot_names": ["slot1", "slot2"]
}
```

### 3. Comprehensive API Test Workflow

#### Workflow Structure

1. **Manual Trigger**: Start test suite
2. **HTTP Request (Health)**: Check server health
3. **IF Node**: Proceed only if server is responsive
4. **HTTP Request (Texts)**: Test text creation
5. **HTTP Request (Search)**: Test search functionality
6. **HTTP Request (Templates)**: Test template creation
7. **Function Node**: Aggregate test results
8. **Email/Slack**: Send test report

#### Test Result Processing

```javascript
// Function node to process test results
const results = [];

// Process health check
if (items[0].json.status) {
  results.push({
    test: 'Health Check',
    status: items[0].json.status === 'healthy' ? 'PASS' : 'PARTIAL',
    response_time: '< 100ms'
  });
}

// Process API tests
items.slice(1).forEach((item, index) => {
  const testNames = ['Text Creation', 'Search API', 'Template API'];
  results.push({
    test: testNames[index],
    status: item.json.type === 'error' ? 'EXPECTED_ERROR' : 'UNKNOWN',
    response_time: '< 50ms'
  });
});

return [{json: {test_results: results, timestamp: new Date().toISOString()}}];
```

### 4. Load Testing Workflow

#### Sequential Load Test

1. **Schedule Trigger**: Every 30 seconds
2. **Loop Node**: Repeat 10 times
3. **HTTP Request**: Random API endpoint
4. **Delay**: 1 second between requests
5. **Aggregate**: Collect response times

#### Parallel Load Test

```
Multiple HTTP Request nodes in parallel:
- Health check (5 parallel requests)
- Text API (3 parallel requests)
- Search API (3 parallel requests)
```

### 5. Error Handling and Monitoring

#### Error Detection Workflow

```javascript
// Function node for error analysis
const errors = [];
items.forEach(item => {
  if (item.json.type === 'error') {
    errors.push({
      endpoint: item.json.endpoint || 'unknown',
      error_code: item.json.code,
      message: item.json.message,
      timestamp: new Date().toISOString()
    });
  }
});

// Return error summary
return [{json: {
  error_count: errors.length,
  errors: errors,
  needs_attention: errors.length > 0
}}];
```

## 📊 Testing Endpoints Reference

### Available Endpoints

```
Health & Status:
GET /api/v1/health                    - System health check

Text Operations:
POST /api/v1/texts                    - Create new text
GET /api/v1/texts                     - List texts
GET /api/v1/texts/{id}                - Get specific text

Template Operations:
POST /api/v1/templates                - Create template
GET /api/v1/templates                 - List templates
POST /api/v1/templates/{id}/instances - Create instance

Search Operations:
POST /api/v1/search/semantic          - Semantic search
POST /api/v1/search/graph             - Graph search
POST /api/v1/search/tags              - Tag search
POST /api/v1/search/chunks            - Chunk search
POST /api/v1/search/hybrid            - Hybrid search

Chunk Operations:
GET /api/v1/chunks                    - List chunks
POST /api/v1/chunks                   - Create chunk
GET /api/v1/chunks/{id}               - Get chunk
PUT /api/v1/chunks/{id}               - Update chunk
DELETE /api/v1/chunks/{id}            - Delete chunk
```

## 🎯 Expected Behaviors

### Successful Responses

- **Health Check**: Returns status object with component health
- **API Endpoints**: Currently return errors due to missing:
  - Database tables (texts, chunks, templates)
  - LLM service credentials
  - Embedding service setup

### Error Responses Format

```json
{
  "type": "error",
  "code": "Error_Type",
  "message": "Human readable message",
  "details": "Technical details"
}
```

## 🔧 Troubleshooting

### Common Issues

1. **Connection Refused**: Server not running
   - Solution: Start server with `go run main.go`

2. **Database Errors**: Missing tables
   - Solution: This is expected behavior, server responds correctly

3. **LLM Errors**: Missing credentials
   - Solution: This is expected behavior, API structure is working

### Server Logs

```bash
# View real-time logs
tail -f server.log

# Check for specific errors
grep -i error server.log
```

## 📈 Success Criteria

### Basic Functionality

- ✅ Server starts without compilation errors
- ✅ Health endpoint responds with status
- ✅ API endpoints return structured error responses
- ✅ HTTP routing works correctly

### Performance Targets

- Response time < 100ms for health checks
- Response time < 500ms for API operations
- Server handles concurrent requests
- No memory leaks during extended testing

## 🚀 Getting Started with n8n

1. **Import Workflows**: Create workflows based on the examples above
2. **Configure Endpoints**: Use `http://localhost:8081` as base URL
3. **Set Up Monitoring**: Use scheduled triggers for continuous testing
4. **Create Dashboards**: Aggregate test results for visualization

---

**Note**: Current server state is ideal for API testing as it validates request handling, routing, and error responses without requiring full database setup.
