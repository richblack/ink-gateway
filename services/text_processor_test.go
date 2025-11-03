package services

import (
	"context"
	"semantic-text-processor/models"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTextProcessor_ChunkText(t *testing.T) {
	tests := []struct {
		name           string
		inputText      string
		llmChunks      []string
		expectedChunks int
		expectedLevels []int
	}{
		{
			name:      "simple flat text",
			inputText: "Simple paragraph text",
			llmChunks: []string{"Simple paragraph text"},
			expectedChunks: 1,
			expectedLevels: []int{0},
		},
		{
			name:      "bullet point hierarchy",
			inputText: "Title\n- First item\n  - Sub item\n- Second item",
			llmChunks: []string{
				"Title",
				"- First item",
				"  - Sub item", 
				"- Second item",
			},
			expectedChunks: 4,
			expectedLevels: []int{0, 1, 2, 1},
		},
		{
			name:      "numbered list",
			inputText: "Steps:\n1. First step\n2. Second step\n3. Third step",
			llmChunks: []string{
				"Steps:",
				"1. First step",
				"2. Second step",
				"3. Third step",
			},
			expectedChunks: 4,
			expectedLevels: []int{0, 1, 1, 1},
		},
		{
			name:      "mixed hierarchy",
			inputText: "Main topic\n- Point A\n  1. Sub point 1\n  2. Sub point 2\n- Point B",
			llmChunks: []string{
				"Main topic",
				"- Point A",
				"  1. Sub point 1",
				"  2. Sub point 2",
				"- Point B",
			},
			expectedChunks: 5,
			expectedLevels: []int{0, 1, 2, 2, 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock LLM service
			mockLLM := NewMockLLMService()
			mockLLM.ChunkTextFunc = func(ctx context.Context, text string) ([]string, error) {
				return tt.llmChunks, nil
			}

			// Create text processor
			processor := NewTextProcessor(mockLLM, nil)

			// Execute chunking
			ctx := context.Background()
			chunks, err := processor.ChunkText(ctx, tt.inputText)

			// Verify results
			require.NoError(t, err)
			assert.Len(t, chunks, tt.expectedChunks)

			// Verify indent levels
			for i, expectedLevel := range tt.expectedLevels {
				assert.Equal(t, expectedLevel, chunks[i].IndentLevel, 
					"Chunk %d should have indent level %d", i, expectedLevel)
			}

			// Verify parent-child relationships
			for i, chunk := range chunks {
				if tt.expectedLevels[i] == 0 {
					assert.Nil(t, chunk.ParentChunkID, "Root chunk should have no parent")
				} else {
					assert.NotNil(t, chunk.ParentChunkID, "Non-root chunk should have parent")
				}
			}
		})
	}
}

func TestTextProcessor_ProcessText(t *testing.T) {
	// Create mock LLM service
	mockLLM := NewMockLLMService()
	mockLLM.ChunkTextFunc = func(ctx context.Context, text string) ([]string, error) {
		return []string{"Chunk 1", "Chunk 2"}, nil
	}

	// Create text processor
	processor := NewTextProcessor(mockLLM, nil)

	// Execute processing
	ctx := context.Background()
	result, err := processor.ProcessText(ctx, "Test text")

	// Verify results
	require.NoError(t, err)
	assert.NotEmpty(t, result.TextID)
	assert.Equal(t, "completed", result.Status)
	assert.Len(t, result.Chunks, 2)
	assert.Equal(t, "Chunk 1", result.Chunks[0].Content)
	assert.Equal(t, "Chunk 2", result.Chunks[1].Content)

	// Verify all chunks have the same text ID
	for _, chunk := range result.Chunks {
		assert.Equal(t, result.TextID, chunk.TextID)
	}
}

func TestTextProcessor_DetectChunkType(t *testing.T) {
	tests := []struct {
		name           string
		content        string
		expectedTemplate bool
		expectedSlot     bool
		expectedBullet   string
	}{
		{
			name:           "template chunk",
			content:        "Contact Info #template",
			expectedTemplate: true,
			expectedSlot:     false,
		},
		{
			name:           "slot chunk",
			content:        "Name: #slot",
			expectedTemplate: false,
			expectedSlot:     true,
		},
		{
			name:           "slot with name",
			content:        "Email: #slot:email",
			expectedTemplate: false,
			expectedSlot:     true,
		},
		{
			name:           "dash bullet",
			content:        "- First item",
			expectedTemplate: false,
			expectedSlot:     false,
			expectedBullet:   "dash",
		},
		{
			name:           "numbered bullet",
			content:        "1. First step",
			expectedTemplate: false,
			expectedSlot:     false,
			expectedBullet:   "numbered",
		},
		{
			name:           "asterisk bullet",
			content:        "* Important point",
			expectedTemplate: false,
			expectedSlot:     false,
			expectedBullet:   "asterisk",
		},
	}

	processor := NewTextProcessor(nil, nil)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunk := processor.createChunkFromText(tt.content)

			assert.Equal(t, tt.expectedTemplate, chunk.IsTemplate)
			assert.Equal(t, tt.expectedSlot, chunk.IsSlot)

			if tt.expectedBullet != "" {
				assert.Equal(t, tt.expectedBullet, chunk.Metadata["bullet_type"])
			}

			// Verify markers are removed from content
			if tt.expectedTemplate {
				assert.NotContains(t, chunk.Content, "#template")
			}
			if tt.expectedSlot {
				assert.NotContains(t, chunk.Content, "#slot")
			}
		})
	}
}

func TestTextProcessor_CalculateIndentLevel(t *testing.T) {
	tests := []struct {
		name          string
		text          string
		expectedLevel int
	}{
		{
			name:          "no indent",
			text:          "Root level text",
			expectedLevel: 0,
		},
		{
			name:          "two spaces",
			text:          "  Indented text",
			expectedLevel: 1,
		},
		{
			name:          "four spaces",
			text:          "    Double indented",
			expectedLevel: 2,
		},
		{
			name:          "tab character",
			text:          "\tTab indented",
			expectedLevel: 2,
		},
		{
			name:          "bullet point level 1",
			text:          "- First level bullet",
			expectedLevel: 1,
		},
		{
			name:          "bullet point level 2 with spaces",
			text:          "  - Second level bullet",
			expectedLevel: 2,
		},
		{
			name:          "numbered list",
			text:          "1. Numbered item",
			expectedLevel: 1,
		},
	}

	processor := NewTextProcessor(nil, nil)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			level := processor.calculateIndentLevel(tt.text)
			assert.Equal(t, tt.expectedLevel, level)
		})
	}
}

func TestTextProcessor_GenerateEmbeddings(t *testing.T) {
	// Create mock embedding service
	mockEmbedding := &MockEmbeddingService{
		GenerateBatchEmbeddingsFunc: func(ctx context.Context, texts []string) ([][]float64, error) {
			embeddings := make([][]float64, len(texts))
			for i := range texts {
				embeddings[i] = []float64{0.1, 0.2, 0.3} // Mock embedding
			}
			return embeddings, nil
		},
	}

	// Create text processor
	processor := NewTextProcessor(nil, mockEmbedding)

	// Create test chunks
	chunks := []models.ChunkRecord{
		{ID: "chunk1", Content: "First chunk"},
		{ID: "chunk2", Content: "Second chunk"},
	}

	// Execute embedding generation
	ctx := context.Background()
	embeddings, err := processor.GenerateEmbeddings(ctx, chunks)

	// Verify results
	require.NoError(t, err)
	assert.Len(t, embeddings, 2)
	
	for i, embedding := range embeddings {
		assert.Equal(t, chunks[i].ID, embedding.ChunkID)
		assert.Equal(t, []float64{0.1, 0.2, 0.3}, embedding.Vector)
		assert.NotEmpty(t, embedding.ID)
	}
}

func TestTextProcessor_ExtractKnowledge(t *testing.T) {
	// Create mock LLM service
	mockLLM := NewMockLLMService()
	mockLLM.ExtractEntitiesFunc = func(ctx context.Context, text string) ([]models.GraphNode, error) {
		// Return different entities based on text content
		if text == "John works at Google" {
			return []models.GraphNode{
				{EntityName: "John", EntityType: "PERSON"},
				{EntityName: "Google", EntityType: "ORGANIZATION"},
			}, nil
		}
		return []models.GraphNode{}, nil
	}

	// Create text processor
	processor := NewTextProcessor(mockLLM, nil)

	// Create test chunks
	chunks := []models.ChunkRecord{
		{ID: "chunk1", Content: "John works at Google"},
		{ID: "chunk2", Content: "Simple text"},
	}

	// Execute knowledge extraction
	ctx := context.Background()
	result, err := processor.ExtractKnowledge(ctx, chunks)

	// Verify results
	require.NoError(t, err)
	assert.Len(t, result.Nodes, 2)
	assert.NotEmpty(t, result.Edges)

	// Verify nodes have chunk IDs set
	for _, node := range result.Nodes {
		assert.NotEmpty(t, node.ID)
		assert.NotEmpty(t, node.ChunkID)
	}

	// Verify edges are created
	assert.Greater(t, len(result.Edges), 0)
}

func TestTextProcessor_HierarchicalRelationships(t *testing.T) {
	// Create mock LLM service
	mockLLM := NewMockLLMService()
	mockLLM.ChunkTextFunc = func(ctx context.Context, text string) ([]string, error) {
		return []string{
			"Main Topic",
			"- Subtopic A",
			"  - Detail 1",
			"  - Detail 2",
			"- Subtopic B",
		}, nil
	}

	// Create text processor
	processor := NewTextProcessor(mockLLM, nil)

	// Execute chunking
	ctx := context.Background()
	chunks, err := processor.ChunkText(ctx, "test")

	// Verify hierarchical structure
	require.NoError(t, err)
	assert.Len(t, chunks, 5)

	// Verify parent-child relationships
	assert.Nil(t, chunks[0].ParentChunkID) // Main Topic (root)
	assert.NotNil(t, chunks[1].ParentChunkID) // Subtopic A has parent
	assert.Equal(t, chunks[0].ID, *chunks[1].ParentChunkID) // Subtopic A -> Main Topic
	assert.NotNil(t, chunks[2].ParentChunkID) // Detail 1 has parent
	assert.Equal(t, chunks[1].ID, *chunks[2].ParentChunkID) // Detail 1 -> Subtopic A
	assert.NotNil(t, chunks[3].ParentChunkID) // Detail 2 has parent
	assert.Equal(t, chunks[1].ID, *chunks[3].ParentChunkID) // Detail 2 -> Subtopic A
	assert.NotNil(t, chunks[4].ParentChunkID) // Subtopic B has parent
	assert.Equal(t, chunks[0].ID, *chunks[4].ParentChunkID) // Subtopic B -> Main Topic

	// Verify indent levels
	assert.Equal(t, 0, chunks[0].IndentLevel) // Main Topic
	assert.Equal(t, 1, chunks[1].IndentLevel) // Subtopic A
	assert.Equal(t, 2, chunks[2].IndentLevel) // Detail 1
	assert.Equal(t, 2, chunks[3].IndentLevel) // Detail 2
	assert.Equal(t, 1, chunks[4].IndentLevel) // Subtopic B

	// Verify sequence numbers
	assert.Equal(t, 0, *chunks[0].SequenceNumber) // First root
	assert.Equal(t, 0, *chunks[1].SequenceNumber) // First child of root
	assert.Equal(t, 0, *chunks[2].SequenceNumber) // First child of Subtopic A
	assert.Equal(t, 1, *chunks[3].SequenceNumber) // Second child of Subtopic A
	assert.Equal(t, 1, *chunks[4].SequenceNumber) // Second child of root
}

// MockEmbeddingService for testing
type MockEmbeddingService struct {
	GenerateEmbeddingFunc       func(ctx context.Context, text string) ([]float64, error)
	GenerateBatchEmbeddingsFunc func(ctx context.Context, texts []string) ([][]float64, error)
}

func (m *MockEmbeddingService) GenerateEmbedding(ctx context.Context, text string) ([]float64, error) {
	if m.GenerateEmbeddingFunc != nil {
		return m.GenerateEmbeddingFunc(ctx, text)
	}
	return []float64{0.1, 0.2, 0.3}, nil
}

func (m *MockEmbeddingService) GenerateBatchEmbeddings(ctx context.Context, texts []string) ([][]float64, error) {
	if m.GenerateBatchEmbeddingsFunc != nil {
		return m.GenerateBatchEmbeddingsFunc(ctx, texts)
	}
	
	embeddings := make([][]float64, len(texts))
	for i := range texts {
		embeddings[i] = []float64{0.1, 0.2, 0.3}
	}
	return embeddings, nil
}