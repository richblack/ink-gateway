# Supabase Client Implementation

This package implements a comprehensive Supabase HTTP client for the semantic text processor application.

## Features Implemented

### Task 2.1: Supabase Client Connection and Basic Operations
- ✅ HTTP client with authentication and error handling
- ✅ Retry mechanism with exponential backoff
- ✅ Health check functionality
- ✅ Comprehensive error handling with custom error types
- ✅ Request/response marshaling and unmarshaling

### Task 2.2: Chunks Table CRUD Operations
- ✅ Basic CRUD operations for chunks (Create, Read, Update, Delete)
- ✅ Batch insert operations for multiple chunks
- ✅ Hierarchical structure support with parent-child relationships
- ✅ Sequence numbering for proper ordering
- ✅ Hierarchy traversal operations (GetChildrenChunks, GetSiblingChunks)
- ✅ Chunk movement and reordering functionality
- ✅ Bulk update operations for batch modifications

### Task 2.3: Tag System Data Operations
- ✅ Tag creation and management (AddTag, RemoveTag)
- ✅ Tag relationship storage in chunk_tags table
- ✅ Tag retrieval operations (GetChunkTags, GetChunksByTag)
- ✅ Advanced tag search functionality (SearchByTag)
- ✅ Automatic tag chunk creation when needed

## Architecture

### Client Structure
```go
type supabaseHTTPClient struct {
    baseURL    string
    apiKey     string
    httpClient *http.Client
}
```

### Key Components

1. **Authentication**: Bearer token authentication with API key
2. **Error Handling**: Custom SupabaseError type with retry logic
3. **Request Management**: Centralized HTTP request handling with context support
4. **Retry Logic**: Exponential backoff for transient failures

### Supported Operations

#### Text Operations
- InsertText, GetTexts, GetTextByID, UpdateText, DeleteText

#### Chunk Operations
- InsertChunk, InsertChunks (batch), GetChunkByID, GetChunkByContent
- UpdateChunk, DeleteChunk, GetChunksByTextID

#### Hierarchy Operations
- GetChunkHierarchy, GetChildrenChunks, GetSiblingChunks
- MoveChunk, BulkUpdateChunks

#### Tag Operations
- AddTag, RemoveTag, GetChunkTags, GetChunksByTag, SearchByTag

## Testing

### Test Coverage
- Unit tests with mock client implementation
- Integration tests for real Supabase connections
- Comprehensive test cases for all CRUD operations
- Hierarchy and tag system testing

### Running Tests
```bash
# Run all tests
go test ./clients -v

# Run with coverage
go test ./clients -v -cover

# Run integration tests (requires Supabase credentials)
SUPABASE_URL=your_url SUPABASE_API_KEY=your_key go test ./clients -v -run Integration
```

## Configuration

The client requires Supabase configuration:
```go
cfg := &config.SupabaseConfig{
    URL:    "https://your-project.supabase.co",
    APIKey: "your-api-key",
}
client := NewSupabaseClient(cfg)
```

## Error Handling

The client implements comprehensive error handling:
- Network errors with retry logic
- Supabase API errors with proper error codes
- Context cancellation support
- Timeout handling

## Future Enhancements

The following methods are stubbed for future implementation:
- Template operations (CreateTemplate, GetTemplateByContent, etc.)
- Vector operations (InsertEmbeddings, SearchSimilar)
- Graph operations (InsertGraphNodes, InsertGraphEdges, SearchGraph)
- Advanced search operations

## Requirements Satisfied

This implementation satisfies the following requirements from the specification:
- **Requirement 3.1**: Supabase API integration for PostgreSQL operations
- **Requirement 4.1**: Database connection and health check functionality  
- **Requirement 5.1**: Error handling and retry mechanisms
- **Requirement 7.1**: Chunk CRUD operations and hierarchy support
- **Requirement 7.2**: Tag system implementation and search functionality