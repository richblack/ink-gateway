package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"semantic-text-processor/config"
	"semantic-text-processor/models"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLLMClient_ChunkText(t *testing.T) {
	tests := []struct {
		name           string
		inputText      string
		mockResponse   LLMResponse
		expectedChunks []string
		expectError    bool
	}{
		{
			name:      "successful chunking",
			inputText: "This is a test document. It has multiple sentences.",
			mockResponse: LLMResponse{
				Success: true,
				Data:    []string{"This is a test document.", "It has multiple sentences."},
			},
			expectedChunks: []string{"This is a test document.", "It has multiple sentences."},
			expectError:    false,
		},
		{
			name:      "empty response returns original text",
			inputText: "Short text",
			mockResponse: LLMResponse{
				Success: true,
				Data:    []string{},
			},
			expectedChunks: []string{"Short text"},
			expectError:    false,
		},
		{
			name:      "API error response",
			inputText: "Test text",
			mockResponse: LLMResponse{
				Success: false,
				Error:   "API rate limit exceeded",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request
				assert.Equal(t, "POST", r.Method)
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
				assert.Equal(t, "Bearer test-api-key", r.Header.Get("Authorization"))

				// Verify request body
				var req LLMRequest
				err := json.NewDecoder(r.Body).Decode(&req)
				require.NoError(t, err)
				assert.Equal(t, tt.inputText, req.Text)
				assert.Equal(t, "chunk_text", req.Operation)

				// Send mock response
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(tt.mockResponse)
			}))
			defer server.Close()

			// Create client
			cfg := &config.LLMConfig{
				APIKey:   "test-api-key",
				Endpoint: server.URL,
				Timeout:  5 * time.Second,
			}
			client := NewLLMClient(cfg)

			// Execute test
			ctx := context.Background()
			chunks, err := client.ChunkText(ctx, tt.inputText)

			// Verify results
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedChunks, chunks)
			}
		})
	}
}

func TestLLMClient_ExtractEntities(t *testing.T) {
	tests := []struct {
		name             string
		inputText        string
		mockResponse     EntityExtractionResponse
		expectedEntities []models.GraphNode
		expectError      bool
	}{
		{
			name:      "successful entity extraction",
			inputText: "John works at Google in California.",
			mockResponse: EntityExtractionResponse{
				Success: true,
				Data: []EntityExtraction{
					{
						Name: "John",
						Type: "PERSON",
						Properties: map[string]interface{}{
							"confidence": 0.95,
						},
					},
					{
						Name: "Google",
						Type: "ORGANIZATION",
						Properties: map[string]interface{}{
							"confidence": 0.90,
						},
					},
					{
						Name: "California",
						Type: "LOCATION",
						Properties: map[string]interface{}{
							"confidence": 0.85,
						},
					},
				},
			},
			expectedEntities: []models.GraphNode{
				{
					EntityName: "John",
					EntityType: "PERSON",
					Properties: map[string]interface{}{
						"confidence": 0.95,
					},
				},
				{
					EntityName: "Google",
					EntityType: "ORGANIZATION",
					Properties: map[string]interface{}{
						"confidence": 0.90,
					},
				},
				{
					EntityName: "California",
					EntityType: "LOCATION",
					Properties: map[string]interface{}{
						"confidence": 0.85,
					},
				},
			},
			expectError: false,
		},
		{
			name:      "API error response",
			inputText: "Test text",
			mockResponse: EntityExtractionResponse{
				Success: false,
				Error:   "Entity extraction failed",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request
				var req LLMRequest
				err := json.NewDecoder(r.Body).Decode(&req)
				require.NoError(t, err)
				assert.Equal(t, tt.inputText, req.Text)
				assert.Equal(t, "extract_entities", req.Operation)

				// Send mock response
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(tt.mockResponse)
			}))
			defer server.Close()

			// Create client
			cfg := &config.LLMConfig{
				APIKey:   "test-api-key",
				Endpoint: server.URL,
				Timeout:  5 * time.Second,
			}
			client := NewLLMClient(cfg)

			// Execute test
			ctx := context.Background()
			entities, err := client.ExtractEntities(ctx, tt.inputText)

			// Verify results
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, entities, len(tt.expectedEntities))
				
				for i, expected := range tt.expectedEntities {
					assert.Equal(t, expected.EntityName, entities[i].EntityName)
					assert.Equal(t, expected.EntityType, entities[i].EntityType)
					assert.Equal(t, expected.Properties, entities[i].Properties)
				}
			}
		})
	}
}

func TestLLMClient_RetryMechanism(t *testing.T) {
	retryCount := 0
	
	// Create server that fails first two requests, succeeds on third
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		retryCount++
		if retryCount < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Server error"))
			return
		}
		
		// Success response
		response := LLMResponse{
			Success: true,
			Data:    []string{"chunk1", "chunk2"},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create client with custom retry config
	cfg := &config.LLMConfig{
		APIKey:   "test-api-key",
		Endpoint: server.URL,
		Timeout:  5 * time.Second,
	}
	client := NewLLMClient(cfg)
	client.SetRetryConfig(&RetryConfig{
		MaxRetries: 3,
		BaseDelay:  10 * time.Millisecond,
		MaxDelay:   100 * time.Millisecond,
	})

	// Execute request
	ctx := context.Background()
	chunks, err := client.ChunkText(ctx, "test text")

	// Verify success after retries
	assert.NoError(t, err)
	assert.Equal(t, []string{"chunk1", "chunk2"}, chunks)
	assert.Equal(t, 3, retryCount)
}

func TestLLMClient_ContextCancellation(t *testing.T) {
	// Create server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		response := LLMResponse{Success: true, Data: []string{"chunk"}}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create client
	cfg := &config.LLMConfig{
		APIKey:   "test-api-key",
		Endpoint: server.URL,
		Timeout:  5 * time.Second,
	}
	client := NewLLMClient(cfg)

	// Create context with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// Execute request
	_, err := client.ChunkText(ctx, "test text")

	// Verify context cancellation error
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context deadline exceeded")
}

func TestMockLLMService_ChunkText(t *testing.T) {
	tests := []struct {
		name           string
		inputText      string
		expectedChunks []string
	}{
		{
			name:      "simple paragraph",
			inputText: "This is a simple paragraph with multiple sentences. It should be chunked appropriately.",
			expectedChunks: []string{
				"This is a simple paragraph with multiple sentences. It should be chunked appropriately.",
			},
		},
		{
			name: "bullet points",
			inputText: `Introduction paragraph

- First bullet point
- Second bullet point
- Third bullet point

Conclusion paragraph`,
			expectedChunks: []string{
				"Introduction paragraph",
				"- First bullet point",
				"- Second bullet point", 
				"- Third bullet point",
				"Conclusion paragraph",
			},
		},
		{
			name: "numbered list",
			inputText: `Steps to follow:

1. First step
2. Second step
3. Third step`,
			expectedChunks: []string{
				"Steps to follow:",
				"1. First step",
				"2. Second step",
				"3. Third step",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := NewMockLLMService()
			ctx := context.Background()
			
			chunks, err := mock.ChunkText(ctx, tt.inputText)
			
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedChunks, chunks)
		})
	}
}

func TestMockLLMService_ExtractEntities(t *testing.T) {
	mock := NewMockLLMService()
	ctx := context.Background()
	
	text := "John Smith works at Microsoft Corporation in Seattle. Contact him at john@microsoft.com."
	
	entities, err := mock.ExtractEntities(ctx, text)
	
	assert.NoError(t, err)
	assert.NotEmpty(t, entities)
	
	// Verify some entities were extracted
	entityNames := make([]string, len(entities))
	for i, entity := range entities {
		entityNames[i] = entity.EntityName
	}
	
	// Should extract capitalized words as entities
	assert.Contains(t, entityNames, "John")
	assert.Contains(t, entityNames, "Smith")
	assert.Contains(t, entityNames, "Microsoft")
	assert.Contains(t, entityNames, "Corporation")
	assert.Contains(t, entityNames, "Seattle")
}

func TestMockLLMService_CustomBehavior(t *testing.T) {
	mock := NewMockLLMService()
	
	// Set custom behavior
	mock.ChunkTextFunc = func(ctx context.Context, text string) ([]string, error) {
		return []string{"custom", "chunks"}, nil
	}
	
	ctx := context.Background()
	chunks, err := mock.ChunkText(ctx, "any text")
	
	assert.NoError(t, err)
	assert.Equal(t, []string{"custom", "chunks"}, chunks)
}