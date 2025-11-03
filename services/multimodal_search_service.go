package services

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"semantic-text-processor/models"
)

// multimodalSearchService 多模態搜尋服務實作
type multimodalSearchService struct {
	chunkService     UnifiedChunkService
	embeddingService ImageEmbeddingService
	visionService    VisionAIService
	searchCache      SearchCacheService
	performanceMonitor QueryPerformanceMonitor
}

// NewMultimodalSearchService 建立新的多模態搜尋服務
func NewMultimodalSearchService(
	chunkService UnifiedChunkService,
	embeddingService ImageEmbeddingService,
	visionService VisionAIService,
	searchCache SearchCacheService,
	performanceMonitor QueryPerformanceMonitor,
) MultimodalSearchService {
	return &multimodalSearchService{
		chunkService:       chunkService,
		embeddingService:   embeddingService,
		visionService:      visionService,
		searchCache:        searchCache,
		performanceMonitor: performanceMonitor,
	}
}

// SearchText 文字搜尋（包含圖片 AI 描述）
func (m *multimodalSearchService) SearchText(ctx context.Context, req *models.MultimodalSearchRequest) (*models.MultimodalSearchResponse, error) {
	startTime := time.Now()
	
	// 檢查快取
	cacheKey := m.generateCacheKey("text", req)
	if cachedResult, err := m.searchCache.GetCachedSearch(ctx, cacheKey); err == nil && cachedResult != nil {
		return m.buildResponseFromCache(cachedResult, "text", time.Since(startTime))
	}
	
	// 建立搜尋查詢
	searchQuery := &models.SearchQuery{
		Content: req.TextQuery,
		Limit:   req.Limit,
	}
	
	// 如果有過濾器，加入到查詢中
	if req.Filters != nil {
		m.applyFiltersToQuery(searchQuery, req.Filters)
	}
	
	// 執行搜尋
	searchResult, err := m.chunkService.SearchChunks(ctx, searchQuery)
	if err != nil {
		return nil, fmt.Errorf("text search failed: %w", err)
	}
	
	// 轉換結果
	response := m.convertToMultimodalResponse(searchResult, "text", req.MinSimilarity)
	response.SearchTime = time.Since(startTime)
	response.Query = req.TextQuery
	
	// 快取結果
	m.cacheSearchResult(ctx, cacheKey, response)
	
	// 記錄效能
	m.performanceMonitor.RecordQuery("text_search", response.SearchTime, len(response.Results))
	
	return response, nil
}

// SearchImages 圖片搜尋（向量相似度）
func (m *multimodalSearchService) SearchImages(ctx context.Context, req *models.MultimodalSearchRequest) (*models.MultimodalSearchResponse, error) {
	startTime := time.Now()
	
	// 檢查快取
	cacheKey := m.generateCacheKey("image", req)
	if cachedResult, err := m.searchCache.GetCachedSearch(ctx, cacheKey); err == nil && cachedResult != nil {
		return m.buildResponseFromCache(cachedResult, "image", time.Since(startTime))
	}
	
	// 生成查詢向量
	var queryVector []float64
	var err error
	
	if req.ImageQuery != "" {
		// 如果提供了圖片查詢，生成圖片向量
		queryVector, err = m.embeddingService.GenerateEmbedding(ctx, req.ImageQuery)
		if err != nil {
			return nil, fmt.Errorf("failed to generate image embedding: %w", err)
		}
	} else if req.TextQuery != "" {
		// 如果只有文字查詢，搜尋圖片的 AI 描述
		return m.searchImagesByDescription(ctx, req, startTime)
	} else {
		return nil, fmt.Errorf("either image_query or text_query is required for image search")
	}
	
	// 執行向量搜尋
	results, err := m.searchByVector(ctx, queryVector, "image", req)
	if err != nil {
		return nil, fmt.Errorf("image vector search failed: %w", err)
	}
	
	// 建立回應
	response := &models.MultimodalSearchResponse{
		Results:     results,
		TotalCount:  len(results),
		SearchTime:  time.Since(startTime),
		Query:       req.ImageQuery,
		MatchTypes:  []string{"image_vector"},
	}
	
	// 快取結果
	m.cacheSearchResult(ctx, cacheKey, response)
	
	// 記錄效能
	m.performanceMonitor.RecordQuery("image_search", response.SearchTime, len(response.Results))
	
	return response, nil
}

// HybridSearch 混合搜尋（文字+圖片）
func (m *multimodalSearchService) HybridSearch(ctx context.Context, req *models.MultimodalSearchRequest) (*models.MultimodalSearchResponse, error) {
	startTime := time.Now()
	
	// 檢查快取
	cacheKey := m.generateCacheKey("hybrid", req)
	if cachedResult, err := m.searchCache.GetCachedSearch(ctx, cacheKey); err == nil && cachedResult != nil {
		return m.buildResponseFromCache(cachedResult, "hybrid", time.Since(startTime))
	}
	
	var allResults []models.MultimodalSearchResult
	var matchTypes []string
	
	// 執行文字搜尋
	if req.TextQuery != "" {
		textReq := &models.MultimodalSearchRequest{
			TextQuery:     req.TextQuery,
			Filters:       req.Filters,
			Limit:         req.Limit * 2, // 取更多結果用於合併
			MinSimilarity: req.MinSimilarity,
		}
		
		textResults, err := m.SearchText(ctx, textReq)
		if err == nil {
			for _, result := range textResults.Results {
				result.MatchType = "text"
				allResults = append(allResults, result)
			}
			matchTypes = append(matchTypes, "text")
		}
	}
	
	// 執行圖片搜尋
	if req.ImageQuery != "" {
		imageReq := &models.MultimodalSearchRequest{
			ImageQuery:    req.ImageQuery,
			Filters:       req.Filters,
			Limit:         req.Limit * 2, // 取更多結果用於合併
			MinSimilarity: req.MinSimilarity,
		}
		
		imageResults, err := m.SearchImages(ctx, imageReq)
		if err == nil {
			for _, result := range imageResults.Results {
				result.MatchType = "image"
				allResults = append(allResults, result)
			}
			matchTypes = append(matchTypes, "image")
		}
	}
	
	// 合併和排序結果
	mergedResults := m.mergeAndRankResults(allResults, req.Weights, req.Limit)
	
	// 建立回應
	response := &models.MultimodalSearchResponse{
		Results:     mergedResults,
		TotalCount:  len(mergedResults),
		SearchTime:  time.Since(startTime),
		Query:       fmt.Sprintf("text:%s image:%s", req.TextQuery, req.ImageQuery),
		MatchTypes:  matchTypes,
	}
	
	// 快取結果
	m.cacheSearchResult(ctx, cacheKey, response)
	
	// 記錄效能
	m.performanceMonitor.RecordQuery("hybrid_search", response.SearchTime, len(response.Results))
	
	return response, nil
}

// SearchByImage 以圖搜圖
func (m *multimodalSearchService) SearchByImage(ctx context.Context, imageURL string, limit int, minSimilarity float64) (*models.MultimodalSearchResponse, error) {
	req := &models.MultimodalSearchRequest{
		ImageQuery:    imageURL,
		Limit:         limit,
		MinSimilarity: minSimilarity,
	}
	
	return m.SearchImages(ctx, req)
}

// RecommendImagesForSlides 為 Slide Generator 推薦圖片
func (m *multimodalSearchService) RecommendImagesForSlides(ctx context.Context, req *models.SlideImageRequest) (*models.ImageRecommendationResponse, error) {
	startTime := time.Now()
	
	// 分析文字內容，提取關鍵概念
	keywords := m.extractKeywords(req.TextContent)
	
	var allRecommendations []models.ImageRecommendation
	
	// 對每個關鍵字進行搜尋
	for _, keyword := range keywords {
		searchReq := &models.MultimodalSearchRequest{
			TextQuery:     keyword,
			VectorType:    "all",
			Limit:         req.MaxSuggestions * 2, // 取更多結果用於篩選
			MinSimilarity: req.MinRelevance,
			Filters: map[string]interface{}{
				"media_type": "image", // 只搜尋圖片
			},
		}
		
		searchResults, err := m.SearchText(ctx, searchReq)
		if err != nil {
			continue // 跳過失敗的搜尋
		}
		
		// 轉換為推薦格式
		for _, result := range searchResults.Results {
			if result.Chunk.IsImageChunk() {
				recommendation := models.ImageRecommendation{
					ChunkID:        result.Chunk.ChunkID,
					ImageURL:       result.Chunk.GetImageURL(),
					Description:    result.Chunk.Contents,
					RelevanceScore: result.Similarity,
					Reason:         fmt.Sprintf("與關鍵字 '%s' 相關", keyword),
					Tags:           result.Chunk.Tags,
				}
				allRecommendations = append(allRecommendations, recommendation)
			}
		}
	}
	
	// 去重和排序
	uniqueRecommendations := m.deduplicateRecommendations(allRecommendations)
	sortedRecommendations := m.sortRecommendationsByRelevance(uniqueRecommendations)
	
	// 限制結果數量
	if len(sortedRecommendations) > req.MaxSuggestions {
		sortedRecommendations = sortedRecommendations[:req.MaxSuggestions]
	}
	
	return &models.ImageRecommendationResponse{
		Suggestions: sortedRecommendations,
		TotalCount:  len(sortedRecommendations),
		SearchTime:  time.Since(startTime),
	}, nil
}

// 私有方法

// searchImagesByDescription 透過描述搜尋圖片
func (m *multimodalSearchService) searchImagesByDescription(ctx context.Context, req *models.MultimodalSearchRequest, startTime time.Time) (*models.MultimodalSearchResponse, error) {
	// 建立搜尋查詢，專門搜尋圖片的 AI 描述
	searchQuery := &models.SearchQuery{
		Content: req.TextQuery,
		Limit:   req.Limit,
		Metadata: map[string]interface{}{
			"media_type": "image",
		},
	}
	
	// 執行搜尋
	searchResult, err := m.chunkService.SearchChunks(ctx, searchQuery)
	if err != nil {
		return nil, fmt.Errorf("image description search failed: %w", err)
	}
	
	// 轉換結果
	response := m.convertToMultimodalResponse(searchResult, "image_description", req.MinSimilarity)
	response.SearchTime = time.Since(startTime)
	response.Query = req.TextQuery
	response.MatchTypes = []string{"image_description"}
	
	return response, nil
}

// searchByVector 執行向量搜尋
func (m *multimodalSearchService) searchByVector(ctx context.Context, queryVector []float64, vectorType string, req *models.MultimodalSearchRequest) ([]models.MultimodalSearchResult, error) {
	// 這裡需要實作向量搜尋邏輯
	// 由於我們使用 UnifiedChunk 中的向量欄位，需要透過 chunkService 進行向量搜尋
	
	// 暫時使用文字搜尋作為替代方案
	// 在實際實作中，這裡應該呼叫向量資料庫的相似度搜尋
	
	searchQuery := &models.SearchQuery{
		Limit: req.Limit,
		Metadata: map[string]interface{}{
			"media_type": "image",
		},
	}
	
	if req.Filters != nil {
		m.applyFiltersToQuery(searchQuery, req.Filters)
	}
	
	searchResult, err := m.chunkService.SearchChunks(ctx, searchQuery)
	if err != nil {
		return nil, err
	}
	
	// 轉換結果並計算相似度
	var results []models.MultimodalSearchResult
	for _, chunk := range searchResult.Chunks {
		if chunk.HasVector() && chunk.GetVectorType() == models.VectorType(vectorType) {
			similarity := m.calculateCosineSimilarity(queryVector, chunk.Vector)
			
			if similarity >= req.MinSimilarity {
				result := models.MultimodalSearchResult{
					Chunk:       &chunk,
					Similarity:  similarity,
					MatchType:   vectorType + "_vector",
					Explanation: fmt.Sprintf("向量相似度: %.3f", similarity),
				}
				results = append(results, result)
			}
		}
	}
	
	// 按相似度排序
	sort.Slice(results, func(i, j int) bool {
		return results[i].Similarity > results[j].Similarity
	})
	
	return results, nil
}

// mergeAndRankResults 合併和排序結果
func (m *multimodalSearchService) mergeAndRankResults(results []models.MultimodalSearchResult, weights *models.SearchWeights, limit int) []models.MultimodalSearchResult {
	if weights == nil {
		weights = &models.SearchWeights{Text: 0.5, Image: 0.5}
	}
	
	// 去重（基於 chunk ID）
	uniqueResults := make(map[string]models.MultimodalSearchResult)
	
	for _, result := range results {
		chunkID := result.Chunk.ChunkID
		
		if existing, exists := uniqueResults[chunkID]; exists {
			// 合併相似度分數
			var combinedSimilarity float64
			if result.MatchType == "text" {
				combinedSimilarity = existing.Similarity*weights.Image + result.Similarity*weights.Text
			} else {
				combinedSimilarity = existing.Similarity*weights.Text + result.Similarity*weights.Image
			}
			
			// 更新結果
			existing.Similarity = combinedSimilarity
			existing.MatchType = "hybrid"
			existing.Explanation = fmt.Sprintf("混合匹配 (文字: %.3f, 圖片: %.3f)", 
				existing.Similarity, result.Similarity)
			uniqueResults[chunkID] = existing
		} else {
			uniqueResults[chunkID] = result
		}
	}
	
	// 轉換為切片並排序
	var mergedResults []models.MultimodalSearchResult
	for _, result := range uniqueResults {
		mergedResults = append(mergedResults, result)
	}
	
	sort.Slice(mergedResults, func(i, j int) bool {
		return mergedResults[i].Similarity > mergedResults[j].Similarity
	})
	
	// 限制結果數量
	if len(mergedResults) > limit {
		mergedResults = mergedResults[:limit]
	}
	
	return mergedResults
}

// convertToMultimodalResponse 轉換為多模態回應
func (m *multimodalSearchService) convertToMultimodalResponse(searchResult *models.SearchResult, matchType string, minSimilarity float64) *models.MultimodalSearchResponse {
	var results []models.MultimodalSearchResult
	
	for _, chunk := range searchResult.Chunks {
		// 這裡應該有實際的相似度計算，暫時使用固定值
		similarity := 0.8 // 實際實作中應該從搜尋結果中取得
		
		if similarity >= minSimilarity {
			result := models.MultimodalSearchResult{
				Chunk:       &chunk,
				Similarity:  similarity,
				MatchType:   matchType,
				Explanation: m.generateExplanation(matchType, similarity),
			}
			results = append(results, result)
		}
	}
	
	return &models.MultimodalSearchResponse{
		Results:    results,
		TotalCount: len(results),
		MatchTypes: []string{matchType},
	}
}

// generateExplanation 生成匹配解釋
func (m *multimodalSearchService) generateExplanation(matchType string, similarity float64) string {
	switch matchType {
	case "text":
		return fmt.Sprintf("文字內容匹配，相似度: %.3f", similarity)
	case "image":
		return fmt.Sprintf("圖片向量匹配，相似度: %.3f", similarity)
	case "image_description":
		return fmt.Sprintf("圖片描述匹配，相似度: %.3f", similarity)
	case "hybrid":
		return fmt.Sprintf("混合匹配，綜合相似度: %.3f", similarity)
	default:
		return fmt.Sprintf("匹配，相似度: %.3f", similarity)
	}
}

// calculateCosineSimilarity 計算餘弦相似度
func (m *multimodalSearchService) calculateCosineSimilarity(vec1, vec2 []float64) float64 {
	if len(vec1) != len(vec2) {
		return 0.0
	}
	
	var dotProduct, norm1, norm2 float64
	
	for i := 0; i < len(vec1); i++ {
		dotProduct += vec1[i] * vec2[i]
		norm1 += vec1[i] * vec1[i]
		norm2 += vec2[i] * vec2[i]
	}
	
	if norm1 == 0 || norm2 == 0 {
		return 0.0
	}

	return dotProduct / (math.Sqrt(norm1) * math.Sqrt(norm2))
}

// extractKeywords 提取關鍵字
func (m *multimodalSearchService) extractKeywords(text string) []string {
	// 簡單的關鍵字提取邏輯
	words := strings.Fields(strings.ToLower(text))
	
	// 過濾停用詞
	stopWords := map[string]bool{
		"的": true, "是": true, "在": true, "有": true, "和": true,
		"與": true, "或": true, "但": true, "如果": true, "因為": true,
		"the": true, "is": true, "in": true, "and": true, "or": true,
		"but": true, "if": true, "because": true, "a": true, "an": true,
	}
	
	var keywords []string
	for _, word := range words {
		if len(word) > 2 && !stopWords[word] {
			keywords = append(keywords, word)
		}
	}
	
	// 限制關鍵字數量
	if len(keywords) > 10 {
		keywords = keywords[:10]
	}
	
	return keywords
}

// deduplicateRecommendations 去重推薦
func (m *multimodalSearchService) deduplicateRecommendations(recommendations []models.ImageRecommendation) []models.ImageRecommendation {
	seen := make(map[string]bool)
	var unique []models.ImageRecommendation
	
	for _, rec := range recommendations {
		if !seen[rec.ChunkID] {
			seen[rec.ChunkID] = true
			unique = append(unique, rec)
		}
	}
	
	return unique
}

// sortRecommendationsByRelevance 按相關度排序推薦
func (m *multimodalSearchService) sortRecommendationsByRelevance(recommendations []models.ImageRecommendation) []models.ImageRecommendation {
	sort.Slice(recommendations, func(i, j int) bool {
		return recommendations[i].RelevanceScore > recommendations[j].RelevanceScore
	})
	return recommendations
}

// applyFiltersToQuery 將過濾器應用到查詢
func (m *multimodalSearchService) applyFiltersToQuery(query *models.SearchQuery, filters map[string]interface{}) {
	if query.Metadata == nil {
		query.Metadata = make(map[string]interface{})
	}
	
	for key, value := range filters {
		query.Metadata[key] = value
	}
}

// generateCacheKey 生成快取鍵
func (m *multimodalSearchService) generateCacheKey(searchType string, req *models.MultimodalSearchRequest) map[string]interface{} {
	return map[string]interface{}{
		"type":           searchType,
		"text_query":     req.TextQuery,
		"image_query":    req.ImageQuery,
		"vector_type":    req.VectorType,
		"limit":          req.Limit,
		"min_similarity": req.MinSimilarity,
		"filters":        req.Filters,
	}
}

// buildResponseFromCache 從快取建立回應
func (m *multimodalSearchService) buildResponseFromCache(cached *models.SearchCacheEntry, matchType string, searchTime time.Duration) (*models.MultimodalSearchResponse, error) {
	// 這裡需要從快取的 chunk IDs 重建完整的回應
	// 暫時返回空結果
	return &models.MultimodalSearchResponse{
		Results:    []models.MultimodalSearchResult{},
		TotalCount: 0,
		SearchTime: searchTime,
		MatchTypes: []string{matchType},
	}, nil
}

// cacheSearchResult 快取搜尋結果
func (m *multimodalSearchService) cacheSearchResult(ctx context.Context, cacheKey map[string]interface{}, response *models.MultimodalSearchResponse) {
	// 提取 chunk IDs 用於快取
	var chunkIDs []string
	for _, result := range response.Results {
		chunkIDs = append(chunkIDs, result.Chunk.ChunkID)
	}
	
	// 快取結果
	m.searchCache.SetCachedSearch(ctx, cacheKey, chunkIDs, 10*time.Minute)
}