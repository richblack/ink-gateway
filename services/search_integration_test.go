package services

import (
	"context"
	"fmt"
	"testing"
	"time"

	"semantic-text-processor/clients"
	"semantic-text-processor/config"
	"semantic-text-processor/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSemanticSearchEndToEnd tests the complete semantic search workflow
func TestSemanticSearchEndToEnd(t *testing.T) {
	// Skip if no Supabase configuration
	cfg := config.LoadConfig()
	if cfg.Supabase.URL == "" || cfg.Supabase.APIKey == "" {
		t.Skip("Supabase configuration not available")
	}

	// Create clients and services
	supabaseClient := clients.NewSupabaseClient(&cfg.Supabase)
	embeddingService := NewTestEmbeddingService() // Use test service for predictable results
	searchService := NewSearchService(supabaseClient, embeddingService)

	ctx := context.Background()

	// Test data
	testText := &models.TextRecord{
		Content: "Complete semantic search integration test document",
		Title:   "Search Test Document",
		Status:  "processing",
	}

	testChunks := []models.ChunkRecord{
		{
			Content:     "Machine learning algorithms for natural language processing",
			IndentLevel: 0,
		},
		{
			Content:     "Deep learning neural networks and transformers",
			IndentLevel: 0,
		},
		{
			Content:     "Vector embeddings and semantic similarity search",
			IndentLevel: 1,
		},
		{
			Content:     "Information retrieval and document ranking",
			IndentLevel: 0,
		},
		{
			Content:     "Database indexing and query optimization",
			IndentLevel: 1,
		},
	}

	t.Run("complete semantic search workflow", func(t *testing.T) {
		// 1. Setup test data
		err := supabaseClient.InsertText(ctx, testText)
		require.NoError(t, err)

		for i := range testChunks {
			testChunks[i].TextID = testText.ID
		}
		err = supabaseClient.InsertChunks(ctx, testChunks)
		require.NoError(t, err)

		// 2. Generate and store embeddings
		texts := make([]string, len(testChunks))
		for i, chunk := range testChunks {
			texts[i] = chunk.Content
		}

		embeddings, err := embeddingService.GenerateBatchEmbeddings(ctx, texts)
		require.NoError(t, err)

		embeddingRecords := make([]models.EmbeddingRecord, len(testChunks))
		for i, chunk := range testChunks {
			embeddingRecords[i] = models.EmbeddingRecord{
				ChunkID: chunk.ID,
				Vector:  embeddings[i],
			}
		}

		err = supabaseClient.InsertEmbeddings(ctx, embeddingRecords)
		require.NoError(t, err)

		// 3. Test basic semantic search
		results, err := searchService.SemanticSearch(ctx, "machine learning algorithms", 3)
		require.NoError(t, err)
		assert.LessOrEqual(t, len(results), 3)

		// Verify results are sorted by similarity
		for i := 1; i < len(results); i++ {
			assert.GreaterOrEqual(t, results[i-1].Similarity, results[i].Similarity)
		}

		// 4. Test semantic search with filters
		filterReq := &models.SemanticSearchRequest{
			Query: "neural networks",
			Limit: 5,
			Filters: map[string]interface{}{
				"text_id": testText.ID,
			},
			MinSimilarity: 0.1, // Low threshold for test data
		}

		filterResponse, err := searchService.SemanticSearchWithFilters(ctx, filterReq)
		require.NoError(t, err)
		assert.LessOrEqual(t, len(filterResponse.Results), 5)
		assert.Equal(t, filterReq.Query, filterResponse.Query)

		// Verify all results match the text_id filter
		for _, result := range filterResponse.Results {
			assert.Equal(t, testText.ID, result.Chunk.TextID)
		}

		// 5. Test hybrid search (if text search is implemented)
		hybridResults, err := searchService.HybridSearch(ctx, "learning", 3, 0.7)
		if err == nil { // Only test if text search is available
			assert.LessOrEqual(t, len(hybridResults), 3)
		}

		// Cleanup
		err = supabaseClient.DeleteText(ctx, testText.ID)
		assert.NoError(t, err)
	})

	t.Run("search performance test", func(t *testing.T) {
		// Create larger dataset for performance testing
		largeText := &models.TextRecord{
			Content: "Performance test document with many chunks",
			Title:   "Performance Test",
			Status:  "processing",
		}

		err := supabaseClient.InsertText(ctx, largeText)
		require.NoError(t, err)

		// Create 50 test chunks
		largeChunks := make([]models.ChunkRecord, 50)
		for i := range largeChunks {
			largeChunks[i] = models.ChunkRecord{
				TextID:      largeText.ID,
				Content:     fmt.Sprintf("Performance test chunk number %d with unique content", i),
				IndentLevel: i % 3, // Vary indent levels
			}
		}

		err = supabaseClient.InsertChunks(ctx, largeChunks)
		require.NoError(t, err)

		// Generate embeddings
		texts := make([]string, len(largeChunks))
		for i, chunk := range largeChunks {
			texts[i] = chunk.Content
		}

		embeddings, err := embeddingService.GenerateBatchEmbeddings(ctx, texts)
		require.NoError(t, err)

		embeddingRecords := make([]models.EmbeddingRecord, len(largeChunks))
		for i, chunk := range largeChunks {
			embeddingRecords[i] = models.EmbeddingRecord{
				ChunkID: chunk.ID,
				Vector:  embeddings[i],
			}
		}

		err = supabaseClient.InsertEmbeddings(ctx, embeddingRecords)
		require.NoError(t, err)

		// Test search performance
		start := time.Now()
		results, err := searchService.SemanticSearch(ctx, "performance test content", 10)
		duration := time.Since(start)

		require.NoError(t, err)
		assert.LessOrEqual(t, len(results), 10)
		assert.Less(t, duration, 5*time.Second) // Should complete within 5 seconds

		// Test filtered search performance
		start = time.Now()
		filterReq := &models.SemanticSearchRequest{
			Query: "unique content",
			Limit: 20,
			Filters: map[string]interface{}{
				"text_id":         largeText.ID,
				"max_indent_level": 2,
			},
		}

		filterResponse, err := searchService.SemanticSearchWithFilters(ctx, filterReq)
		duration = time.Since(start)

		require.NoError(t, err)
		assert.LessOrEqual(t, len(filterResponse.Results), 20)
		assert.Less(t, duration, 5*time.Second)

		// Cleanup
		err = supabaseClient.DeleteText(ctx, largeText.ID)
		assert.NoError(t, err)
	})
}

// TestSearchServiceConcurrency tests concurrent search operations
func TestSearchServiceConcurrency(t *testing.T) {
	cfg := config.LoadConfig()
	if cfg.Supabase.URL == "" || cfg.Supabase.APIKey == "" {
		t.Skip("Supabase configuration not available")
	}

	supabaseClient := clients.NewSupabaseClient(&cfg.Supabase)
	embeddingService := NewTestEmbeddingService()
	searchService := NewSearchService(supabaseClient, embeddingService)

	ctx := context.Background()
	const numGoroutines = 5
	const numSearches = 3

	results := make(chan error, numGoroutines*numSearches)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			for j := 0; j < numSearches; j++ {
				query := fmt.Sprintf("concurrent search test %d-%d", id, j)
				_, err := searchService.SemanticSearch(ctx, query, 5)
				results <- err
			}
		}(i)
	}

	// Collect results
	for i := 0; i < numGoroutines*numSearches; i++ {
		err := <-results
		// Errors are expected since we don't have test data, but should not panic
		if err != nil {
			t.Logf("Expected error in concurrent test: %v", err)
		}
	}
}

// BenchmarkSemanticSearch benchmarks search performance
func BenchmarkSemanticSearch(b *testing.B) {
	// Use mock services for consistent benchmarking
	mockSupabase := &MockSupabaseClient{
		searchSimilarFunc: func(ctx context.Context, queryVector []float64, limit int) ([]models.SimilarityResult, error) {
			// Simulate realistic search results
			results := make([]models.SimilarityResult, limit)
			for i := 0; i < limit; i++ {
				results[i] = models.SimilarityResult{
					Chunk: models.ChunkRecord{
						ID:      fmt.Sprintf("chunk%d", i),
						Content: fmt.Sprintf("Benchmark chunk content %d", i),
					},
					Similarity: 0.9 - float64(i)*0.1,
				}
			}
			return results, nil
		},
	}

	embeddingService := NewTestEmbeddingService()
	searchService := NewSearchService(mockSupabase, embeddingService)
	ctx := context.Background()

	b.Run("basic semantic search", func(b *testing.B) {
		query := "benchmark test query"
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, err := searchService.SemanticSearch(ctx, query, 10)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("filtered semantic search", func(b *testing.B) {
		req := &models.SemanticSearchRequest{
			Query: "benchmark test query",
			Limit: 10,
			Filters: map[string]interface{}{
				"text_id": "test-text-id",
			},
			MinSimilarity: 0.5,
		}
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, err := searchService.SemanticSearchWithFilters(ctx, req)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("hybrid search", func(b *testing.B) {
		mockSupabase.searchChunksFunc = func(ctx context.Context, query string, filters map[string]interface{}) ([]models.ChunkRecord, error) {
			return []models.ChunkRecord{
				{
					ID:      "text-chunk1",
					Content: "Text search result",
				},
			}, nil
		}

		query := "benchmark hybrid query"
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, err := searchService.HybridSearch(ctx, query, 10, 0.7)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}