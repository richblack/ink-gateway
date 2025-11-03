package services

import (
	"context"
	"fmt"
	"math/rand"
	"time"
)

// TestEmbeddingService provides a mock implementation for testing
type TestEmbeddingService struct {
	embeddings map[string][]float64
	shouldFail bool
	delay      time.Duration
}

// NewTestEmbeddingService creates a new test embedding service
func NewTestEmbeddingService() *TestEmbeddingService {
	return &TestEmbeddingService{
		embeddings: make(map[string][]float64),
		shouldFail: false,
		delay:      0,
	}
}

// SetShouldFail configures the mock to return errors
func (m *TestEmbeddingService) SetShouldFail(shouldFail bool) {
	m.shouldFail = shouldFail
}

// SetDelay configures artificial delay for testing
func (m *TestEmbeddingService) SetDelay(delay time.Duration) {
	m.delay = delay
}

// SetEmbedding sets a predefined embedding for a specific text
func (m *TestEmbeddingService) SetEmbedding(text string, embedding []float64) {
	m.embeddings[text] = embedding
}

// GenerateEmbedding generates a mock embedding for a single text
func (m *TestEmbeddingService) GenerateEmbedding(ctx context.Context, text string) ([]float64, error) {
	if m.delay > 0 {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(m.delay):
		}
	}
	
	if m.shouldFail {
		return nil, fmt.Errorf("mock embedding service error")
	}
	
	// Return predefined embedding if exists
	if embedding, exists := m.embeddings[text]; exists {
		return embedding, nil
	}
	
	// Generate deterministic mock embedding based on text hash
	return m.generateMockEmbedding(text), nil
}

// GenerateBatchEmbeddings generates mock embeddings for multiple texts
func (m *TestEmbeddingService) GenerateBatchEmbeddings(ctx context.Context, texts []string) ([][]float64, error) {
	if m.delay > 0 {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(m.delay):
		}
	}
	
	if m.shouldFail {
		return nil, fmt.Errorf("mock embedding service error")
	}
	
	embeddings := make([][]float64, len(texts))
	for i, text := range texts {
		// Return predefined embedding if exists
		if embedding, exists := m.embeddings[text]; exists {
			embeddings[i] = embedding
		} else {
			embeddings[i] = m.generateMockEmbedding(text)
		}
	}
	
	return embeddings, nil
}

// generateMockEmbedding creates a deterministic mock embedding
func (m *TestEmbeddingService) generateMockEmbedding(text string) []float64 {
	// Use text hash as seed for deterministic results
	seed := int64(0)
	for _, char := range text {
		seed += int64(char)
	}
	
	rng := rand.New(rand.NewSource(seed))
	
	// Generate 1536-dimensional embedding (OpenAI ada-002 size)
	embedding := make([]float64, 1536)
	for i := range embedding {
		embedding[i] = rng.Float64()*2 - 1 // Values between -1 and 1
	}
	
	// Normalize the vector
	var magnitude float64
	for _, val := range embedding {
		magnitude += val * val
	}
	magnitude = 1.0 / (magnitude + 1e-8) // Add small epsilon to avoid division by zero
	
	for i := range embedding {
		embedding[i] *= magnitude
	}
	
	return embedding
}