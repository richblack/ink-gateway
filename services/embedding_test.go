package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"semantic-text-processor/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEmbeddingService_GenerateEmbedding(t *testing.T) {
	tests := []struct {
		name           string
		text           string
		serverResponse string
		serverStatus   int
		expectedError  bool
		expectedDim    int
	}{
		{
			name: "successful single embedding",
			text: "Hello world",
			serverResponse: `{
				"data": [
					{
						"embedding": [0.1, 0.2, 0.3],
						"index": 0
					}
				],
				"model": "text-embedding-ada-002",
				"usage": {
					"prompt_tokens": 2,
					"total_tokens": 2
				}
			}`,
			serverStatus:  200,
			expectedError: false,
			expectedDim:   3,
		},
		{
			name:           "empty text",
			text:           "",
			serverResponse: `{"data": [{"embedding": [], "index": 0}]}`,
			serverStatus:   200,
			expectedError:  false,
			expectedDim:    0,
		},
		{
			name:           "server error",
			text:           "test",
			serverResponse: `{"error": {"message": "Server error", "type": "server_error"}}`,
			serverStatus:   500,
			expectedError:  true,
		},
		{
			name:           "rate limit error",
			text:           "test",
			serverResponse: `{"error": {"message": "Rate limit exceeded", "type": "rate_limit_exceeded"}}`,
			serverStatus:   429,
			expectedError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "POST", r.Method)
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
				assert.Contains(t, r.Header.Get("Authorization"), "Bearer")

				// Verify request body
				var req EmbeddingRequest
				err := json.NewDecoder(r.Body).Decode(&req)
				require.NoError(t, err)
				assert.Equal(t, []string{tt.text}, req.Input)
				assert.Equal(t, "text-embedding-ada-002", req.Model)

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.serverStatus)
				w.Write([]byte(tt.serverResponse))
			}))
			defer server.Close()

			// Create service
			cfg := &config.EmbeddingConfig{
				APIKey:   "test-key",
				Endpoint: server.URL,
				Timeout:  5 * time.Second,
			}
			service := NewEmbeddingService(cfg)

			// Test
			ctx := context.Background()
			embedding, err := service.GenerateEmbedding(ctx, tt.text)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, embedding)
			} else {
				assert.NoError(t, err)
				assert.Len(t, embedding, tt.expectedDim)
			}
		})
	}
}

func TestEmbeddingService_GenerateBatchEmbeddings(t *testing.T) {
	tests := []struct {
		name           string
		texts          []string
		serverResponse string
		serverStatus   int
		expectedError  bool
		expectedCount  int
	}{
		{
			name:  "successful batch embedding",
			texts: []string{"Hello", "World"},
			serverResponse: `{
				"data": [
					{
						"embedding": [0.1, 0.2],
						"index": 0
					},
					{
						"embedding": [0.3, 0.4],
						"index": 1
					}
				],
				"model": "text-embedding-ada-002"
			}`,
			serverStatus:  200,
			expectedError: false,
			expectedCount: 2,
		},
		{
			name:           "empty input",
			texts:          []string{},
			serverResponse: "",
			serverStatus:   200,
			expectedError:  false,
			expectedCount:  0,
		},
		{
			name:  "missing embedding in response",
			texts: []string{"Hello", "World"},
			serverResponse: `{
				"data": [
					{
						"embedding": [0.1, 0.2],
						"index": 0
					}
				]
			}`,
			serverStatus:  200,
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requestCount := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requestCount++
				
				if len(tt.texts) > 0 {
					var req EmbeddingRequest
					err := json.NewDecoder(r.Body).Decode(&req)
					require.NoError(t, err)
					assert.Equal(t, tt.texts, req.Input)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.serverStatus)
				if tt.serverResponse != "" {
					w.Write([]byte(tt.serverResponse))
				}
			}))
			defer server.Close()

			cfg := &config.EmbeddingConfig{
				APIKey:   "test-key",
				Endpoint: server.URL,
				Timeout:  5 * time.Second,
			}
			service := NewEmbeddingService(cfg)

			ctx := context.Background()
			embeddings, err := service.GenerateBatchEmbeddings(ctx, tt.texts)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, embeddings, tt.expectedCount)
				
				// Verify each embedding has correct dimensions
				for i, embedding := range embeddings {
					assert.NotNil(t, embedding, "embedding %d should not be nil", i)
				}
			}

			// Verify request was made only if texts were provided
			if len(tt.texts) > 0 {
				assert.Equal(t, 1, requestCount)
			} else {
				assert.Equal(t, 0, requestCount)
			}
		})
	}
}

func TestEmbeddingService_Retry(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		
		if requestCount < 3 {
			// Fail first two requests
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(500)
			w.Write([]byte(`{"error": {"message": "Server error", "type": "server_error"}}`))
			return
		}
		
		// Succeed on third request
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write([]byte(`{
			"data": [{"embedding": [0.1, 0.2], "index": 0}],
			"model": "text-embedding-ada-002"
		}`))
	}))
	defer server.Close()

	cfg := &config.EmbeddingConfig{
		APIKey:   "test-key",
		Endpoint: server.URL,
		Timeout:  5 * time.Second,
	}
	service := NewEmbeddingService(cfg)

	ctx := context.Background()
	embedding, err := service.GenerateEmbedding(ctx, "test")

	assert.NoError(t, err)
	assert.Len(t, embedding, 2)
	assert.Equal(t, 3, requestCount) // Should have retried twice
}

func TestEmbeddingService_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow response
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(200)
		w.Write([]byte(`{"data": [{"embedding": [0.1], "index": 0}]}`))
	}))
	defer server.Close()

	cfg := &config.EmbeddingConfig{
		APIKey:   "test-key",
		Endpoint: server.URL,
		Timeout:  5 * time.Second,
	}
	service := NewEmbeddingService(cfg)

	// Create context that cancels quickly
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := service.GenerateEmbedding(ctx, "test")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context deadline exceeded")
}

func TestMockEmbeddingService(t *testing.T) {
	mock := NewTestEmbeddingService()

	t.Run("generate deterministic embedding", func(t *testing.T) {
		ctx := context.Background()
		text := "test text"

		// Generate embedding twice
		embedding1, err1 := mock.GenerateEmbedding(ctx, text)
		embedding2, err2 := mock.GenerateEmbedding(ctx, text)

		assert.NoError(t, err1)
		assert.NoError(t, err2)
		assert.Equal(t, embedding1, embedding2) // Should be deterministic
		assert.Len(t, embedding1, 1536)        // OpenAI ada-002 dimensions
	})

	t.Run("predefined embedding", func(t *testing.T) {
		ctx := context.Background()
		text := "custom text"
		expectedEmbedding := []float64{0.1, 0.2, 0.3}

		mock.SetEmbedding(text, expectedEmbedding)
		embedding, err := mock.GenerateEmbedding(ctx, text)

		assert.NoError(t, err)
		assert.Equal(t, expectedEmbedding, embedding)
	})

	t.Run("batch embeddings", func(t *testing.T) {
		ctx := context.Background()
		texts := []string{"text1", "text2"}

		embeddings, err := mock.GenerateBatchEmbeddings(ctx, texts)

		assert.NoError(t, err)
		assert.Len(t, embeddings, 2)
		assert.Len(t, embeddings[0], 1536)
		assert.Len(t, embeddings[1], 1536)
		assert.NotEqual(t, embeddings[0], embeddings[1]) // Should be different
	})

	t.Run("failure mode", func(t *testing.T) {
		ctx := context.Background()
		mock.SetShouldFail(true)

		_, err := mock.GenerateEmbedding(ctx, "test")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "mock embedding service error")
	})

	t.Run("delay simulation", func(t *testing.T) {
		ctx := context.Background()
		mock.SetShouldFail(false)
		mock.SetDelay(50 * time.Millisecond)

		start := time.Now()
		_, err := mock.GenerateEmbedding(ctx, "test")
		duration := time.Since(start)

		assert.NoError(t, err)
		assert.GreaterOrEqual(t, duration, 50*time.Millisecond)
	})

	t.Run("context cancellation with delay", func(t *testing.T) {
		mock.SetDelay(100 * time.Millisecond)
		
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		_, err := mock.GenerateEmbedding(ctx, "test")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "context deadline exceeded")
	})
}