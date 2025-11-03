package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"semantic-text-processor/config"
)

// embeddingService implements EmbeddingService interface
type embeddingService struct {
	apiKey     string
	endpoint   string
	httpClient *http.Client
}

// NewEmbeddingService creates a new embedding service instance
func NewEmbeddingService(cfg *config.EmbeddingConfig) EmbeddingService {
	return &embeddingService{
		apiKey:   cfg.APIKey,
		endpoint: cfg.Endpoint,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
	}
}

// EmbeddingRequest represents the request structure for embedding API
type EmbeddingRequest struct {
	Input []string `json:"input"`
	Model string   `json:"model"`
}

// EmbeddingResponse represents the response structure from embedding API
type EmbeddingResponse struct {
	Data []struct {
		Embedding []float64 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
	Model string `json:"model"`
	Usage struct {
		PromptTokens int `json:"prompt_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
}

// EmbeddingError represents errors from embedding API
type EmbeddingError struct {
	ErrorInfo struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error"`
}

func (e *EmbeddingError) Error() string {
	return fmt.Sprintf("embedding API error [%s]: %s", e.ErrorInfo.Type, e.ErrorInfo.Message)
}

// GenerateEmbedding generates vector embedding for a single text
func (s *embeddingService) GenerateEmbedding(ctx context.Context, text string) ([]float64, error) {
	embeddings, err := s.GenerateBatchEmbeddings(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	
	if len(embeddings) == 0 {
		return nil, fmt.Errorf("no embedding returned for text")
	}
	
	return embeddings[0], nil
}

// GenerateBatchEmbeddings generates vector embeddings for multiple texts
func (s *embeddingService) GenerateBatchEmbeddings(ctx context.Context, texts []string) ([][]float64, error) {
	if len(texts) == 0 {
		return [][]float64{}, nil
	}
	
	// Prepare request
	request := EmbeddingRequest{
		Input: texts,
		Model: "text-embedding-ada-002", // Default OpenAI model
	}
	
	// Execute with retry
	var response EmbeddingResponse
	err := s.executeWithRetry(ctx, func() error {
		return s.makeRequest(ctx, request, &response)
	})
	
	if err != nil {
		return nil, fmt.Errorf("failed to generate embeddings: %w", err)
	}
	
	// Extract embeddings in correct order
	embeddings := make([][]float64, len(texts))
	for _, data := range response.Data {
		if data.Index < len(embeddings) {
			embeddings[data.Index] = data.Embedding
		}
	}
	
	// Validate all embeddings were returned
	for i, embedding := range embeddings {
		if embedding == nil {
			return nil, fmt.Errorf("missing embedding for text at index %d", i)
		}
	}
	
	return embeddings, nil
}

// makeRequest performs HTTP request to embedding API
func (s *embeddingService) makeRequest(ctx context.Context, request EmbeddingRequest, response *EmbeddingResponse) error {
	// Marshal request body
	jsonData, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}
	
	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", s.endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	
	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	
	// Execute request
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	
	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}
	
	// Handle error responses
	if resp.StatusCode >= 400 {
		var embeddingErr EmbeddingError
		if err := json.Unmarshal(respBody, &embeddingErr); err != nil {
			return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
		}
		return &embeddingErr
	}
	
	// Parse successful response
	if err := json.Unmarshal(respBody, response); err != nil {
		return fmt.Errorf("failed to unmarshal response: %w", err)
	}
	
	return nil
}

// executeWithRetry executes an operation with exponential backoff retry
func (s *embeddingService) executeWithRetry(ctx context.Context, operation func() error) error {
	maxRetries := 3
	baseDelay := 100 * time.Millisecond
	maxDelay := 5 * time.Second
	
	var lastErr error
	
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			// Calculate delay with exponential backoff
			delay := time.Duration(attempt) * baseDelay
			if delay > maxDelay {
				delay = maxDelay
			}
			
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}
		
		if err := operation(); err != nil {
			lastErr = err
			// Check if error is retryable
			if !s.isRetryableError(err) {
				return err
			}
			continue
		}
		
		return nil
	}
	
	return fmt.Errorf("operation failed after %d retries: %w", maxRetries, lastErr)
}

// isRetryableError determines if an error should trigger a retry
func (s *embeddingService) isRetryableError(err error) bool {
	if embeddingErr, ok := err.(*EmbeddingError); ok {
		// Don't retry client errors (4xx), but retry server errors (5xx) and rate limits
		return embeddingErr.ErrorInfo.Type == "server_error" || 
			   embeddingErr.ErrorInfo.Type == "rate_limit_exceeded"
	}
	return true // Retry network errors and other unknown errors
}