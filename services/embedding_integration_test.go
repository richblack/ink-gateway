package services

import (
	"context"
	"fmt"
	"testing"

	"semantic-text-processor/clients"
	"semantic-text-processor/config"
	"semantic-text-processor/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEmbeddingIntegration tests the complete embedding workflow
func TestEmbeddingIntegration(t *testing.T) {
	// Skip if no Supabase configuration
	cfg := config.LoadConfig()
	if cfg.Supabase.URL == "" || cfg.Supabase.APIKey == "" {
		t.Skip("Supabase configuration not available")
	}

	// Create clients
	supabaseClient := clients.NewSupabaseClient(&cfg.Supabase)
	embeddingService := NewTestEmbeddingService() // Use mock for predictable tests

	ctx := context.Background()

	// Test data
	testText := &models.TextRecord{
		Content: "This is a test document for embedding integration",
		Title:   "Test Document",
		Status:  "processing",
	}

	testChunks := []models.ChunkRecord{
		{
			Content:     "First chunk of text content",
			IndentLevel: 0,
		},
		{
			Content:     "Second chunk with different content",
			IndentLevel: 0,
		},
		{
			Content:     "Third chunk for comprehensive testing",
			IndentLevel: 1,
		},
	}

	t.Run("complete embedding workflow", func(t *testing.T) {
		// 1. Insert test text
		err := supabaseClient.InsertText(ctx, testText)
		require.NoError(t, err)
		require.NotEmpty(t, testText.ID)

		// 2. Insert test chunks
		for i := range testChunks {
			testChunks[i].TextID = testText.ID
		}
		err = supabaseClient.InsertChunks(ctx, testChunks)
		require.NoError(t, err)

		// 3. Generate embeddings for chunks
		texts := make([]string, len(testChunks))
		for i, chunk := range testChunks {
			texts[i] = chunk.Content
		}

		embeddings, err := embeddingService.GenerateBatchEmbeddings(ctx, texts)
		require.NoError(t, err)
		require.Len(t, embeddings, len(testChunks))

		// 4. Create embedding records
		embeddingRecords := make([]models.EmbeddingRecord, len(testChunks))
		for i, chunk := range testChunks {
			embeddingRecords[i] = models.EmbeddingRecord{
				ChunkID: chunk.ID,
				Vector:  embeddings[i],
			}
		}

		// 5. Store embeddings in Supabase
		err = supabaseClient.InsertEmbeddings(ctx, embeddingRecords)
		require.NoError(t, err)

		// Verify embeddings were stored
		for _, record := range embeddingRecords {
			assert.NotEmpty(t, record.ID)
			assert.NotZero(t, record.CreatedAt)
		}

		// 6. Test similarity search
		queryText := "chunk content testing"
		queryEmbedding, err := embeddingService.GenerateEmbedding(ctx, queryText)
		require.NoError(t, err)

		results, err := supabaseClient.SearchSimilar(ctx, queryEmbedding, 5)
		require.NoError(t, err)

		// Verify search results
		assert.LessOrEqual(t, len(results), 5)
		for _, result := range results {
			assert.NotEmpty(t, result.Chunk.ID)
			assert.NotEmpty(t, result.Chunk.Content)
			assert.GreaterOrEqual(t, result.Similarity, 0.0)
			assert.LessOrEqual(t, result.Similarity, 1.0)
		}

		// Cleanup
		err = supabaseClient.DeleteText(ctx, testText.ID)
		assert.NoError(t, err)
	})

	t.Run("embedding storage error handling", func(t *testing.T) {
		// Test with invalid chunk ID
		invalidEmbedding := []models.EmbeddingRecord{
			{
				ChunkID: "invalid-chunk-id",
				Vector:  []float64{0.1, 0.2, 0.3},
			},
		}

		err := supabaseClient.InsertEmbeddings(ctx, invalidEmbedding)
		assert.Error(t, err)
	})

	t.Run("similarity search edge cases", func(t *testing.T) {
		// Test with empty vector
		results, err := supabaseClient.SearchSimilar(ctx, []float64{}, 5)
		assert.Error(t, err)
		assert.Nil(t, results)

		// Test with zero limit
		queryEmbedding := make([]float64, 1536)
		for i := range queryEmbedding {
			queryEmbedding[i] = 0.1
		}

		results, err = supabaseClient.SearchSimilar(ctx, queryEmbedding, 0)
		assert.NoError(t, err)
		assert.LessOrEqual(t, len(results), 10) // Should use default limit
	})
}

// TestEmbeddingServiceIntegration tests embedding service with real API
func TestEmbeddingServiceIntegration(t *testing.T) {
	cfg := config.LoadConfig()
	if cfg.Embedding.APIKey == "" || cfg.Embedding.Endpoint == "" {
		t.Skip("Embedding service configuration not available")
	}

	service := NewEmbeddingService(&cfg.Embedding)
	ctx := context.Background()

	t.Run("real API single embedding", func(t *testing.T) {
		text := "This is a test sentence for embedding generation."
		
		embedding, err := service.GenerateEmbedding(ctx, text)
		require.NoError(t, err)
		assert.Greater(t, len(embedding), 0)
		
		// Verify embedding is normalized (approximately)
		var magnitude float64
		for _, val := range embedding {
			magnitude += val * val
		}
		assert.InDelta(t, 1.0, magnitude, 0.1) // Should be close to 1.0 for normalized vectors
	})

	t.Run("real API batch embeddings", func(t *testing.T) {
		texts := []string{
			"First test sentence.",
			"Second different sentence.",
			"Third unique text content.",
		}
		
		embeddings, err := service.GenerateBatchEmbeddings(ctx, texts)
		require.NoError(t, err)
		assert.Len(t, embeddings, len(texts))
		
		// Verify all embeddings have same dimensions
		expectedDim := len(embeddings[0])
		for i, embedding := range embeddings {
			assert.Len(t, embedding, expectedDim, "embedding %d has wrong dimensions", i)
		}
		
		// Verify embeddings are different
		assert.NotEqual(t, embeddings[0], embeddings[1])
		assert.NotEqual(t, embeddings[1], embeddings[2])
	})

	t.Run("real API error handling", func(t *testing.T) {
		// Test with very long text that might exceed limits
		longText := ""
		for i := 0; i < 10000; i++ {
			longText += "word "
		}
		
		_, err := service.GenerateEmbedding(ctx, longText)
		// Should either succeed or fail gracefully
		if err != nil {
			assert.Contains(t, err.Error(), "embedding")
		}
	})
}

// BenchmarkEmbeddingService benchmarks embedding generation performance
func BenchmarkEmbeddingService(b *testing.B) {
	mock := NewTestEmbeddingService()
	ctx := context.Background()
	
	b.Run("single embedding", func(b *testing.B) {
		text := "Benchmark test text for embedding generation performance"
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := mock.GenerateEmbedding(ctx, text)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
	
	b.Run("batch embeddings", func(b *testing.B) {
		texts := []string{
			"First benchmark text",
			"Second benchmark text", 
			"Third benchmark text",
			"Fourth benchmark text",
			"Fifth benchmark text",
		}
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := mock.GenerateBatchEmbeddings(ctx, texts)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// TestEmbeddingConcurrency tests concurrent embedding generation
func TestEmbeddingConcurrency(t *testing.T) {
	mock := NewTestEmbeddingService()
	ctx := context.Background()
	
	const numGoroutines = 10
	const numRequests = 5
	
	results := make(chan error, numGoroutines*numRequests)
	
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			for j := 0; j < numRequests; j++ {
				text := fmt.Sprintf("concurrent test text %d-%d", id, j)
				_, err := mock.GenerateEmbedding(ctx, text)
				results <- err
			}
		}(i)
	}
	
	// Collect results
	for i := 0; i < numGoroutines*numRequests; i++ {
		err := <-results
		assert.NoError(t, err)
	}
}