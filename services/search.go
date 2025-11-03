package services

import (
	"context"
	"fmt"

	"semantic-text-processor/clients"
	"semantic-text-processor/models"
)

// searchService implements SearchService interface
type searchService struct {
	supabaseClient   clients.SupabaseClient
	embeddingService EmbeddingService
}

// NewSearchService creates a new search service instance
func NewSearchService(supabaseClient clients.SupabaseClient, embeddingService EmbeddingService) SearchService {
	return &searchService{
		supabaseClient:   supabaseClient,
		embeddingService: embeddingService,
	}
}



// SemanticSearch performs vector similarity search with enhanced features
func (s *searchService) SemanticSearch(ctx context.Context, query string, limit int) ([]models.SimilarityResult, error) {
	if query == "" {
		return []models.SimilarityResult{}, nil
	}
	
	if limit <= 0 {
		limit = 10 // Default limit
	}
	
	// Generate embedding for the query
	queryEmbedding, err := s.embeddingService.GenerateEmbedding(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}
	
	// Perform similarity search
	results, err := s.supabaseClient.SearchSimilar(ctx, queryEmbedding, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to perform similarity search: %w", err)
	}
	
	return results, nil
}

// SemanticSearchWithFilters performs semantic search with additional filtering and pagination
func (s *searchService) SemanticSearchWithFilters(ctx context.Context, req *models.SemanticSearchRequest) (*models.SemanticSearchResponse, error) {
	if req.Query == "" {
		return &models.SemanticSearchResponse{
			Results:    []models.SimilarityResult{},
			TotalCount: 0,
			Query:      req.Query,
			Limit:      req.Limit,
		}, nil
	}
	
	if req.Limit <= 0 {
		req.Limit = 10
	}
	
	// Generate embedding for the query
	queryEmbedding, err := s.embeddingService.GenerateEmbedding(ctx, req.Query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}
	
	// Perform similarity search with enhanced parameters
	results, err := s.searchSimilarWithFilters(ctx, queryEmbedding, req)
	if err != nil {
		return nil, fmt.Errorf("failed to perform filtered similarity search: %w", err)
	}
	
	// Filter results by minimum similarity if specified
	if req.MinSimilarity > 0 {
		filteredResults := make([]models.SimilarityResult, 0, len(results))
		for _, result := range results {
			if result.Similarity >= req.MinSimilarity {
				filteredResults = append(filteredResults, result)
			}
		}
		results = filteredResults
	}
	
	return &models.SemanticSearchResponse{
		Results:    results,
		TotalCount: len(results),
		Query:      req.Query,
		Limit:      req.Limit,
	}, nil
}

// searchSimilarWithFilters performs the actual similarity search with filters
func (s *searchService) searchSimilarWithFilters(ctx context.Context, queryVector []float64, req *models.SemanticSearchRequest) ([]models.SimilarityResult, error) {
	// For now, use the basic search and apply filters post-search
	// In a production system, you'd want to push filters to the database level
	results, err := s.supabaseClient.SearchSimilar(ctx, queryVector, req.Limit*2) // Get more results to account for filtering
	if err != nil {
		return nil, err
	}
	
	// Apply filters if specified
	if len(req.Filters) > 0 {
		filteredResults := make([]models.SimilarityResult, 0, len(results))
		for _, result := range results {
			if s.matchesFilters(&result.Chunk, req.Filters) {
				filteredResults = append(filteredResults, result)
			}
		}
		results = filteredResults
	}
	
	// Limit results to requested amount
	if len(results) > req.Limit {
		results = results[:req.Limit]
	}
	
	return results, nil
}

// matchesFilters checks if a chunk matches the specified filters
func (s *searchService) matchesFilters(chunk *models.ChunkRecord, filters map[string]interface{}) bool {
	for key, value := range filters {
		switch key {
		case "text_id":
			if chunk.TextID != value.(string) {
				return false
			}
		case "is_template":
			if chunk.IsTemplate != value.(bool) {
				return false
			}
		case "is_slot":
			if chunk.IsSlot != value.(bool) {
				return false
			}
		case "indent_level":
			if chunk.IndentLevel != value.(int) {
				return false
			}
		case "min_indent_level":
			if chunk.IndentLevel < value.(int) {
				return false
			}
		case "max_indent_level":
			if chunk.IndentLevel > value.(int) {
				return false
			}
		// Add more filter types as needed
		}
	}
	return true
}

// GraphSearch performs graph-based search (placeholder implementation)
func (s *searchService) GraphSearch(ctx context.Context, query *models.GraphQuery) (*models.GraphResult, error) {
	// This will be implemented in task 5.2
	return s.supabaseClient.SearchGraph(ctx, query)
}

// SearchByTag searches chunks by tag content
func (s *searchService) SearchByTag(ctx context.Context, tagContent string) ([]models.ChunkWithTags, error) {
	return s.supabaseClient.SearchByTag(ctx, tagContent)
}

// SearchChunks performs general text-based search on chunks
func (s *searchService) SearchChunks(ctx context.Context, query string, filters map[string]interface{}) ([]models.ChunkRecord, error) {
	return s.supabaseClient.SearchChunks(ctx, query, filters)
}

// HybridSearch combines semantic and text-based search for better results
func (s *searchService) HybridSearch(ctx context.Context, query string, limit int, semanticWeight float64) ([]models.SimilarityResult, error) {
	if semanticWeight < 0 || semanticWeight > 1 {
		return nil, fmt.Errorf("semantic weight must be between 0 and 1")
	}
	
	// Perform semantic search
	semanticResults, err := s.SemanticSearch(ctx, query, limit*2) // Get more results for merging
	if err != nil {
		return nil, fmt.Errorf("semantic search failed: %w", err)
	}
	
	// Perform text-based search
	textResults, err := s.SearchChunks(ctx, query, nil)
	if err != nil {
		return nil, fmt.Errorf("text search failed: %w", err)
	}
	
	// Merge and rank results
	mergedResults := s.mergeSearchResults(semanticResults, textResults, semanticWeight)
	
	// Limit results
	if len(mergedResults) > limit {
		mergedResults = mergedResults[:limit]
	}
	
	return mergedResults, nil
}

// mergeSearchResults combines semantic and text search results with weighted scoring
func (s *searchService) mergeSearchResults(semanticResults []models.SimilarityResult, textResults []models.ChunkRecord, semanticWeight float64) []models.SimilarityResult {
	resultMap := make(map[string]*models.SimilarityResult)
	textWeight := 1.0 - semanticWeight
	
	// Add semantic results
	for _, result := range semanticResults {
		resultMap[result.Chunk.ID] = &models.SimilarityResult{
			Chunk:      result.Chunk,
			Similarity: result.Similarity * semanticWeight,
		}
	}
	
	// Add text results with basic scoring
	for _, chunk := range textResults {
		if existing, exists := resultMap[chunk.ID]; exists {
			// Combine scores
			existing.Similarity += textWeight * 0.8 // Base text match score
		} else {
			// New result from text search
			resultMap[chunk.ID] = &models.SimilarityResult{
				Chunk:      chunk,
				Similarity: textWeight * 0.8,
			}
		}
	}
	
	// Convert map to slice and sort by similarity
	results := make([]models.SimilarityResult, 0, len(resultMap))
	for _, result := range resultMap {
		results = append(results, *result)
	}
	
	// Sort by similarity (descending)
	for i := 0; i < len(results)-1; i++ {
		for j := i + 1; j < len(results); j++ {
			if results[i].Similarity < results[j].Similarity {
				results[i], results[j] = results[j], results[i]
			}
		}
	}
	
	return results
}