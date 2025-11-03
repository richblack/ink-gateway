package services

import (
	"context"
	"semantic-text-processor/models"
	"strings"
	"time"
)

// MockLLMService provides a mock implementation for testing
type MockLLMService struct {
	ChunkTextFunc      func(ctx context.Context, text string) ([]string, error)
	ExtractEntitiesFunc func(ctx context.Context, text string) ([]models.GraphNode, error)
}

// NewMockLLMService creates a new mock LLM service
func NewMockLLMService() *MockLLMService {
	return &MockLLMService{
		ChunkTextFunc:      defaultChunkText,
		ExtractEntitiesFunc: defaultExtractEntities,
	}
}

// ChunkText implements LLMService.ChunkText with mock behavior
func (m *MockLLMService) ChunkText(ctx context.Context, text string) ([]string, error) {
	if m.ChunkTextFunc != nil {
		return m.ChunkTextFunc(ctx, text)
	}
	return defaultChunkText(ctx, text)
}

// ExtractEntities implements LLMService.ExtractEntities with mock behavior
func (m *MockLLMService) ExtractEntities(ctx context.Context, text string) ([]models.GraphNode, error) {
	if m.ExtractEntitiesFunc != nil {
		return m.ExtractEntitiesFunc(ctx, text)
	}
	return defaultExtractEntities(ctx, text)
}

// defaultChunkText provides simple text chunking for testing
func defaultChunkText(ctx context.Context, text string) ([]string, error) {
	// Simple chunking by paragraphs and bullet points
	lines := strings.Split(text, "\n")
	var chunks []string
	var currentChunk strings.Builder
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			if currentChunk.Len() > 0 {
				chunks = append(chunks, strings.TrimSpace(currentChunk.String()))
				currentChunk.Reset()
			}
			continue
		}
		
		// Check if line starts with bullet point or number
		if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") || 
		   strings.HasPrefix(line, "+ ") || isNumberedList(line) {
			// Start new chunk for bullet points
			if currentChunk.Len() > 0 {
				chunks = append(chunks, strings.TrimSpace(currentChunk.String()))
				currentChunk.Reset()
			}
			currentChunk.WriteString(line)
		} else {
			// Continue current chunk
			if currentChunk.Len() > 0 {
				currentChunk.WriteString(" ")
			}
			currentChunk.WriteString(line)
		}
	}
	
	// Add final chunk
	if currentChunk.Len() > 0 {
		chunks = append(chunks, strings.TrimSpace(currentChunk.String()))
	}
	
	// If no chunks found, return original text
	if len(chunks) == 0 {
		chunks = []string{text}
	}
	
	return chunks, nil
}

// defaultExtractEntities provides simple entity extraction for testing
func defaultExtractEntities(ctx context.Context, text string) ([]models.GraphNode, error) {
	// Simple mock entity extraction
	words := strings.Fields(text)
	entities := make(map[string]string)
	
	for _, word := range words {
		word = strings.Trim(word, ".,!?;:")
		
		// Simple heuristics for entity detection
		if len(word) > 2 {
			if isCapitalized(word) {
				if len(word) > 6 {
					entities[word] = "ORGANIZATION"
				} else {
					entities[word] = "PERSON"
				}
			} else if strings.Contains(word, "@") {
				entities[word] = "EMAIL"
			} else if len(word) > 8 {
				entities[word] = "CONCEPT"
			}
		}
	}
	
	nodes := make([]models.GraphNode, 0, len(entities))
	for name, entityType := range entities {
		nodes = append(nodes, models.GraphNode{
			EntityName: name,
			EntityType: entityType,
			Properties: map[string]interface{}{
				"confidence": 0.8,
				"source":     "mock",
			},
			CreatedAt: time.Now(),
		})
	}
	
	return nodes, nil
}

// isNumberedList checks if line starts with a number
func isNumberedList(line string) bool {
	if len(line) < 3 {
		return false
	}
	
	for i, char := range line {
		if char >= '0' && char <= '9' {
			continue
		}
		if char == '.' || char == ')' {
			return i > 0 && i < len(line)-1 && line[i+1] == ' '
		}
		return false
	}
	return false
}

// isCapitalized checks if word starts with capital letter
func isCapitalized(word string) bool {
	if len(word) == 0 {
		return false
	}
	return word[0] >= 'A' && word[0] <= 'Z'
}