package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"semantic-text-processor/config"
	"semantic-text-processor/errors"
	"semantic-text-processor/models"
	"time"
)

// LLMClient implements LLMService interface
type LLMClient struct {
	config     *config.LLMConfig
	httpClient *http.Client
	retryConfig *RetryConfig
}

// RetryConfig defines retry behavior
type RetryConfig struct {
	MaxRetries int
	BaseDelay  time.Duration
	MaxDelay   time.Duration
}

// NewLLMClient creates a new LLM service client
func NewLLMClient(cfg *config.LLMConfig) *LLMClient {
	return &LLMClient{
		config: cfg,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
		retryConfig: &RetryConfig{
			MaxRetries: 3,
			BaseDelay:  time.Second,
			MaxDelay:   30 * time.Second,
		},
	}
}

// LLMRequest represents the request structure for LLM API
type LLMRequest struct {
	Text      string `json:"text"`
	Operation string `json:"operation"`
	Options   map[string]interface{} `json:"options,omitempty"`
}

// LLMResponse represents the response structure from LLM API
type LLMResponse struct {
	Success bool     `json:"success"`
	Data    []string `json:"data"`
	Error   string   `json:"error,omitempty"`
}

// EntityExtractionResponse represents entity extraction response
type EntityExtractionResponse struct {
	Success bool                `json:"success"`
	Data    []EntityExtraction  `json:"data"`
	Error   string              `json:"error,omitempty"`
}

// EntityExtraction represents extracted entity information
type EntityExtraction struct {
	Name       string                 `json:"name"`
	Type       string                 `json:"type"`
	Properties map[string]interface{} `json:"properties,omitempty"`
}

// ChunkText implements LLMService.ChunkText
func (c *LLMClient) ChunkText(ctx context.Context, text string) ([]string, error) {
	if text == "" {
		return nil, errors.NewValidationError(
			errors.ErrCodeInvalidInput,
			"Text cannot be empty",
			nil,
		)
	}

	request := LLMRequest{
		Text:      text,
		Operation: "chunk_text",
		Options: map[string]interface{}{
			"preserve_structure": true,
			"semantic_chunking":  true,
		},
	}

	var response LLMResponse
	err := c.executeWithRetry(ctx, request, &response)
	if err != nil {
		return nil, errors.WrapError(err, errors.ErrTypeExternal, 
			errors.ErrCodeLLMServiceFailed, "Failed to chunk text")
	}

	if !response.Success {
		return nil, errors.NewExternalServiceError(
			errors.ErrCodeLLMServiceFailed,
			"LLM API returned error: "+response.Error,
			nil,
		)
	}

	if len(response.Data) == 0 {
		return []string{text}, nil // Return original text if no chunks
	}

	return response.Data, nil
}

// ExtractEntities implements LLMService.ExtractEntities
func (c *LLMClient) ExtractEntities(ctx context.Context, text string) ([]models.GraphNode, error) {
	if text == "" {
		return nil, errors.NewValidationError(
			errors.ErrCodeInvalidInput,
			"Text cannot be empty",
			nil,
		)
	}

	request := LLMRequest{
		Text:      text,
		Operation: "extract_entities",
		Options: map[string]interface{}{
			"include_relationships": true,
			"entity_types": []string{"PERSON", "ORGANIZATION", "LOCATION", "CONCEPT", "EVENT"},
		},
	}

	var response EntityExtractionResponse
	err := c.executeWithRetry(ctx, request, &response)
	if err != nil {
		return nil, errors.WrapError(err, errors.ErrTypeExternal,
			errors.ErrCodeLLMServiceFailed, "Failed to extract entities")
	}

	if !response.Success {
		return nil, errors.NewExternalServiceError(
			errors.ErrCodeLLMServiceFailed,
			"LLM API returned error: "+response.Error,
			nil,
		)
	}

	// Convert EntityExtraction to GraphNode
	nodes := make([]models.GraphNode, len(response.Data))
	for i, entity := range response.Data {
		nodes[i] = models.GraphNode{
			EntityName: entity.Name,
			EntityType: entity.Type,
			Properties: entity.Properties,
			CreatedAt:  time.Now(),
		}
	}

	return nodes, nil
}

// executeWithRetry executes HTTP request with retry logic using the new error system
func (c *LLMClient) executeWithRetry(ctx context.Context, request LLMRequest, response interface{}) error {
	retryer := errors.NewRetryer(errors.ExternalServiceRetryConfig())
	
	operation := func() error {
		return c.makeHTTPRequest(ctx, request, response)
	}
	
	return retryer.Execute(ctx, operation)
}

// makeHTTPRequest makes the actual HTTP request to LLM API
func (c *LLMClient) makeHTTPRequest(ctx context.Context, request LLMRequest, response interface{}) error {
	// Marshal request body
	requestBody, err := json.Marshal(request)
	if err != nil {
		return errors.NewInternalError(
			errors.ErrCodeSerializationError,
			"Failed to marshal LLM request",
			err,
		)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", c.config.Endpoint, bytes.NewBuffer(requestBody))
	if err != nil {
		return errors.NewInternalError(
			errors.ErrCodeProcessingError,
			"Failed to create HTTP request",
			err,
		)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.config.APIKey)

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return errors.NewNetworkError(
			errors.ErrCodeNetworkConnection,
			"LLM API request failed",
			err,
		)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return errors.NewNetworkError(
			errors.ErrCodeNetworkConnection,
			"Failed to read LLM API response",
			err,
		)
	}

	// Check HTTP status
	if resp.StatusCode >= 400 {
		return c.handleHTTPError(resp.StatusCode, string(body))
	}

	// Unmarshal response
	if err := json.Unmarshal(body, response); err != nil {
		return errors.NewInternalError(
			errors.ErrCodeSerializationError,
			"Failed to unmarshal LLM API response",
			err,
		)
	}

	return nil
}

// handleHTTPError converts HTTP errors to appropriate AppErrors
func (c *LLMClient) handleHTTPError(statusCode int, body string) error {
	switch {
	case statusCode == 401:
		return errors.NewAuthError(
			errors.ErrCodeInvalidCredentials,
			"LLM API authentication failed",
			fmt.Errorf("HTTP %d: %s", statusCode, body),
		)
	case statusCode == 429:
		return errors.NewRateLimitError(
			"LLM_RATE_LIMIT",
			"LLM API rate limit exceeded",
			fmt.Errorf("HTTP %d: %s", statusCode, body),
		)
	case statusCode >= 500:
		return errors.NewExternalServiceError(
			errors.ErrCodeLLMServiceFailed,
			"LLM API server error",
			fmt.Errorf("HTTP %d: %s", statusCode, body),
		)
	case statusCode >= 400:
		return errors.NewValidationError(
			errors.ErrCodeInvalidInput,
			"LLM API client error",
			fmt.Errorf("HTTP %d: %s", statusCode, body),
		)
	default:
		return errors.NewExternalServiceError(
			errors.ErrCodeLLMServiceFailed,
			"Unexpected LLM API error",
			fmt.Errorf("HTTP %d: %s", statusCode, body),
		)
	}
}

// HTTPError represents HTTP-specific errors (kept for backward compatibility)
type HTTPError struct {
	StatusCode int
	Message    string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("HTTP %d: %s", e.StatusCode, e.Message)
}

// SetRetryConfig allows customizing retry behavior
func (c *LLMClient) SetRetryConfig(config *RetryConfig) {
	c.retryConfig = config
}