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

// HybridSearchAlgorithm 混合搜尋演算法服務
type HybridSearchAlgorithm struct {
	textWeight       float64
	imageWeight      float64
	similarityThreshold float64
	maxResults       int
	boostFactors     map[string]float64
	penaltyFactors   map[string]float64
}

// NewHybridSearchAlgorithm 建立新的混合搜尋演算法
func NewHybridSearchAlgorithm() *HybridSearchAlgorithm {
	return &HybridSearchAlgorithm{
		textWeight:          0.6,
		imageWeight:         0.4,
		similarityThreshold: 0.3,
		maxResults:          100,
		boostFactors: map[string]float64{
			"exact_match":    1.5,
			"title_match":    1.3,
			"tag_match":      1.2,
			"recent_content": 1.1,
		},
		penaltyFactors: map[string]float64{
			"old_content":     0.9,
			"low_quality":     0.8,
			"duplicate_type":  0.7,
		},
	}
}

// HybridSearchConfig 混合搜尋配置
type HybridSearchConfig struct {
	TextWeight          float64            `json:"text_weight"`
	ImageWeight         float64            `json:"image_weight"`
	SimilarityThreshold float64            `json:"similarity_threshold"`
	MaxResults          int                `json:"max_results"`
	BoostFactors        map[string]float64 `json:"boost_factors"`
	PenaltyFactors      map[string]float64 `json:"penalty_factors"`
	EnableReranking     bool               `json:"enable_reranking"`
	EnableExplanation   bool               `json:"enable_explanation"`
}

// SearchResult 搜尋結果項目
type SearchResult struct {
	Chunk           *models.UnifiedChunkRecord
	TextSimilarity  float64
	ImageSimilarity float64
	CombinedScore   float64
	MatchType       string
	Explanation     string
	BoostFactors    []string
	PenaltyFactors  []string
}

// MergeResults 合併文字和圖片搜尋結果
func (h *HybridSearchAlgorithm) MergeResults(
	ctx context.Context,
	textResults []models.MultimodalSearchResult,
	imageResults []models.MultimodalSearchResult,
	weights *models.SearchWeights,
) ([]models.MultimodalSearchResult, error) {
	
	// 使用提供的權重或預設權重
	textWeight := h.textWeight
	imageWeight := h.imageWeight
	
	if weights != nil {
		textWeight = weights.Text
		imageWeight = weights.Image
		
		// 正規化權重
		total := textWeight + imageWeight
		if total > 0 {
			textWeight = textWeight / total
			imageWeight = imageWeight / total
		}
	}
	
	// 建立結果映射
	resultMap := make(map[string]*SearchResult)
	
	// 處理文字搜尋結果
	for _, result := range textResults {
		chunkID := result.Chunk.ChunkID
		
		searchResult := &SearchResult{
			Chunk:           result.Chunk,
			TextSimilarity:  result.Similarity,
			ImageSimilarity: 0.0,
			MatchType:       "text",
		}
		
		resultMap[chunkID] = searchResult
	}
	
	// 處理圖片搜尋結果
	for _, result := range imageResults {
		chunkID := result.Chunk.ChunkID
		
		if existing, exists := resultMap[chunkID]; exists {
			// 合併結果
			existing.ImageSimilarity = result.Similarity
			existing.MatchType = "hybrid"
		} else {
			// 新的圖片結果
			searchResult := &SearchResult{
				Chunk:           result.Chunk,
				TextSimilarity:  0.0,
				ImageSimilarity: result.Similarity,
				MatchType:       "image",
			}
			
			resultMap[chunkID] = searchResult
		}
	}
	
	// 計算組合分數
	var results []SearchResult
	for _, result := range resultMap {
		result.CombinedScore = h.calculateCombinedScore(result, textWeight, imageWeight)
		
		// 應用提升和懲罰因子
		h.applyBoostAndPenalty(result)
		
		// 生成解釋
		result.Explanation = h.generateExplanation(result, textWeight, imageWeight)
		
		// 過濾低分結果
		if result.CombinedScore >= h.similarityThreshold {
			results = append(results, *result)
		}
	}
	
	// 排序結果
	sort.Slice(results, func(i, j int) bool {
		return results[i].CombinedScore > results[j].CombinedScore
	})
	
	// 限制結果數量
	if len(results) > h.maxResults {
		results = results[:h.maxResults]
	}
	
	// 轉換為 MultimodalSearchResult
	return h.convertToMultimodalResults(results), nil
}

// RankResults 重新排序搜尋結果
func (h *HybridSearchAlgorithm) RankResults(
	ctx context.Context,
	results []models.MultimodalSearchResult,
	query string,
	options *RankingOptions,
) ([]models.MultimodalSearchResult, error) {
	
	if options == nil {
		options = DefaultRankingOptions()
	}
	
	// 轉換為內部結果格式
	searchResults := h.convertFromMultimodalResults(results)
	
	// 應用重新排序演算法
	for i := range searchResults {
		result := &searchResults[i]
		
		// 計算查詢相關性
		queryRelevance := h.calculateQueryRelevance(result.Chunk, query)
		result.CombinedScore = result.CombinedScore * (1.0 + queryRelevance*options.QueryRelevanceWeight)
		
		// 計算內容品質分數
		qualityScore := h.calculateContentQuality(result.Chunk)
		result.CombinedScore = result.CombinedScore * (1.0 + qualityScore*options.QualityWeight)
		
		// 計算時間衰減
		if options.EnableTimeDecay {
			timeDecay := h.calculateTimeDecay(result.Chunk.CreatedTime, options.TimeDecayFactor)
			result.CombinedScore = result.CombinedScore * timeDecay
		}
		
		// 計算多樣性分數
		if options.EnableDiversification {
			diversityScore := h.calculateDiversityScore(result, searchResults, i)
			result.CombinedScore = result.CombinedScore * (1.0 + diversityScore*options.DiversityWeight)
		}
		
		// 更新解釋
		result.Explanation = h.generateDetailedExplanation(result, queryRelevance, qualityScore)
	}
	
	// 重新排序
	sort.Slice(searchResults, func(i, j int) bool {
		return searchResults[i].CombinedScore > searchResults[j].CombinedScore
	})
	
	return h.convertToMultimodalResults(searchResults), nil
}

// RankingOptions 排序選項
type RankingOptions struct {
	QueryRelevanceWeight float64 `json:"query_relevance_weight"`
	QualityWeight        float64 `json:"quality_weight"`
	TimeDecayFactor      float64 `json:"time_decay_factor"`
	DiversityWeight      float64 `json:"diversity_weight"`
	EnableTimeDecay      bool    `json:"enable_time_decay"`
	EnableDiversification bool    `json:"enable_diversification"`
}

// DefaultRankingOptions 預設排序選項
func DefaultRankingOptions() *RankingOptions {
	return &RankingOptions{
		QueryRelevanceWeight:  0.3,
		QualityWeight:         0.2,
		TimeDecayFactor:       0.1,
		DiversityWeight:       0.1,
		EnableTimeDecay:       true,
		EnableDiversification: true,
	}
}

// FilterResults 過濾搜尋結果
func (h *HybridSearchAlgorithm) FilterResults(
	ctx context.Context,
	results []models.MultimodalSearchResult,
	filters *ResultFilters,
) ([]models.MultimodalSearchResult, error) {
	
	if filters == nil {
		return results, nil
	}
	
	var filtered []models.MultimodalSearchResult
	
	for _, result := range results {
		if h.matchesFilters(result, filters) {
			filtered = append(filtered, result)
		}
	}
	
	return filtered, nil
}

// ResultFilters 結果過濾器
type ResultFilters struct {
	MinSimilarity    float64   `json:"min_similarity"`
	MaxSimilarity    float64   `json:"max_similarity"`
	MatchTypes       []string  `json:"match_types"`
	ContentTypes     []string  `json:"content_types"`
	Tags             []string  `json:"tags"`
	ExcludeTags      []string  `json:"exclude_tags"`
	DateRange        *DateRange `json:"date_range"`
	MinContentLength int       `json:"min_content_length"`
	MaxContentLength int       `json:"max_content_length"`
}

// DateRange 日期範圍
type DateRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// DeduplicateResults 去重搜尋結果
func (h *HybridSearchAlgorithm) DeduplicateResults(
	ctx context.Context,
	results []models.MultimodalSearchResult,
	strategy string,
) ([]models.MultimodalSearchResult, error) {
	
	switch strategy {
	case "chunk_id":
		return h.deduplicateByChunkID(results), nil
	case "content_hash":
		return h.deduplicateByContentHash(results), nil
	case "similarity":
		return h.deduplicateBySimilarity(results, 0.95), nil
	default:
		return h.deduplicateByChunkID(results), nil
	}
}

// 私有方法

// calculateCombinedScore 計算組合分數
func (h *HybridSearchAlgorithm) calculateCombinedScore(result *SearchResult, textWeight, imageWeight float64) float64 {
	// 基本加權平均
	combinedScore := result.TextSimilarity*textWeight + result.ImageSimilarity*imageWeight
	
	// 如果同時有文字和圖片匹配，給予額外獎勵
	if result.TextSimilarity > 0 && result.ImageSimilarity > 0 {
		combinedScore *= 1.1 // 10% 獎勵
	}
	
	return combinedScore
}

// applyBoostAndPenalty 應用提升和懲罰因子
func (h *HybridSearchAlgorithm) applyBoostAndPenalty(result *SearchResult) {
	chunk := result.Chunk
	
	// 檢查提升因子
	if h.isExactMatch(chunk) {
		result.CombinedScore *= h.boostFactors["exact_match"]
		result.BoostFactors = append(result.BoostFactors, "exact_match")
	}
	
	if h.isTitleMatch(chunk) {
		result.CombinedScore *= h.boostFactors["title_match"]
		result.BoostFactors = append(result.BoostFactors, "title_match")
	}
	
	if h.isTagMatch(chunk) {
		result.CombinedScore *= h.boostFactors["tag_match"]
		result.BoostFactors = append(result.BoostFactors, "tag_match")
	}
	
	if h.isRecentContent(chunk) {
		result.CombinedScore *= h.boostFactors["recent_content"]
		result.BoostFactors = append(result.BoostFactors, "recent_content")
	}
	
	// 檢查懲罰因子
	if h.isOldContent(chunk) {
		result.CombinedScore *= h.penaltyFactors["old_content"]
		result.PenaltyFactors = append(result.PenaltyFactors, "old_content")
	}
	
	if h.isLowQuality(chunk) {
		result.CombinedScore *= h.penaltyFactors["low_quality"]
		result.PenaltyFactors = append(result.PenaltyFactors, "low_quality")
	}
}

// generateExplanation 生成解釋
func (h *HybridSearchAlgorithm) generateExplanation(result *SearchResult, textWeight, imageWeight float64) string {
	var parts []string
	
	if result.TextSimilarity > 0 {
		parts = append(parts, fmt.Sprintf("文字相似度: %.3f (權重: %.2f)", result.TextSimilarity, textWeight))
	}
	
	if result.ImageSimilarity > 0 {
		parts = append(parts, fmt.Sprintf("圖片相似度: %.3f (權重: %.2f)", result.ImageSimilarity, imageWeight))
	}
	
	parts = append(parts, fmt.Sprintf("綜合分數: %.3f", result.CombinedScore))
	
	if len(result.BoostFactors) > 0 {
		parts = append(parts, fmt.Sprintf("提升因子: %s", strings.Join(result.BoostFactors, ", ")))
	}
	
	if len(result.PenaltyFactors) > 0 {
		parts = append(parts, fmt.Sprintf("懲罰因子: %s", strings.Join(result.PenaltyFactors, ", ")))
	}
	
	return strings.Join(parts, "; ")
}

// generateDetailedExplanation 生成詳細解釋
func (h *HybridSearchAlgorithm) generateDetailedExplanation(result *SearchResult, queryRelevance, qualityScore float64) string {
	explanation := result.Explanation
	
	if queryRelevance > 0 {
		explanation += fmt.Sprintf("; 查詢相關性: %.3f", queryRelevance)
	}
	
	if qualityScore > 0 {
		explanation += fmt.Sprintf("; 內容品質: %.3f", qualityScore)
	}
	
	return explanation
}

// calculateQueryRelevance 計算查詢相關性
func (h *HybridSearchAlgorithm) calculateQueryRelevance(chunk *models.UnifiedChunkRecord, query string) float64 {
	if query == "" {
		return 0.0
	}
	
	queryWords := strings.Fields(strings.ToLower(query))
	content := strings.ToLower(chunk.Contents)
	
	var matches int
	for _, word := range queryWords {
		if strings.Contains(content, word) {
			matches++
		}
	}
	
	if len(queryWords) == 0 {
		return 0.0
	}
	
	return float64(matches) / float64(len(queryWords))
}

// calculateContentQuality 計算內容品質
func (h *HybridSearchAlgorithm) calculateContentQuality(chunk *models.UnifiedChunkRecord) float64 {
	score := 0.5 // 基礎分數
	
	// 內容長度
	contentLength := len(chunk.Contents)
	if contentLength > 100 {
		score += 0.1
	}
	if contentLength > 500 {
		score += 0.1
	}
	
	// 標籤數量
	if len(chunk.Tags) > 0 {
		score += 0.1
	}
	if len(chunk.Tags) > 3 {
		score += 0.1
	}
	
	// 是否有 metadata
	if chunk.Metadata != nil && len(chunk.Metadata) > 0 {
		score += 0.1
	}
	
	// 是否有向量
	if chunk.HasVector() {
		score += 0.1
	}
	
	return math.Min(score, 1.0)
}

// calculateTimeDecay 計算時間衰減
func (h *HybridSearchAlgorithm) calculateTimeDecay(createdTime time.Time, decayFactor float64) float64 {
	daysSinceCreation := time.Since(createdTime).Hours() / 24
	
	// 使用指數衰減
	decay := math.Exp(-decayFactor * daysSinceCreation / 365) // 以年為單位
	
	return math.Max(decay, 0.1) // 最小保持 10%
}

// calculateDiversityScore 計算多樣性分數
func (h *HybridSearchAlgorithm) calculateDiversityScore(result *SearchResult, allResults []SearchResult, currentIndex int) float64 {
	if currentIndex == 0 {
		return 0.0 // 第一個結果不需要多樣性調整
	}
	
	// 檢查與前面結果的相似性
	var similaritySum float64
	for i := 0; i < currentIndex; i++ {
		similarity := h.calculateContentSimilarity(result.Chunk, allResults[i].Chunk)
		similaritySum += similarity
	}
	
	averageSimilarity := similaritySum / float64(currentIndex)
	
	// 相似性越高，多樣性分數越低
	return 1.0 - averageSimilarity
}

// calculateContentSimilarity 計算內容相似性
func (h *HybridSearchAlgorithm) calculateContentSimilarity(chunk1, chunk2 *models.UnifiedChunkRecord) float64 {
	// 簡單的基於標籤的相似性計算
	if len(chunk1.Tags) == 0 || len(chunk2.Tags) == 0 {
		return 0.0
	}
	
	tagSet1 := make(map[string]bool)
	for _, tag := range chunk1.Tags {
		tagSet1[tag] = true
	}
	
	var commonTags int
	for _, tag := range chunk2.Tags {
		if tagSet1[tag] {
			commonTags++
		}
	}
	
	totalTags := len(chunk1.Tags) + len(chunk2.Tags) - commonTags
	if totalTags == 0 {
		return 0.0
	}
	
	return float64(commonTags) / float64(totalTags)
}

// matchesFilters 檢查是否符合過濾條件
func (h *HybridSearchAlgorithm) matchesFilters(result models.MultimodalSearchResult, filters *ResultFilters) bool {
	// 相似度過濾
	if filters.MinSimilarity > 0 && result.Similarity < filters.MinSimilarity {
		return false
	}
	
	if filters.MaxSimilarity > 0 && result.Similarity > filters.MaxSimilarity {
		return false
	}
	
	// 匹配類型過濾
	if len(filters.MatchTypes) > 0 {
		found := false
		for _, matchType := range filters.MatchTypes {
			if result.MatchType == matchType {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	
	// 標籤過濾
	if len(filters.Tags) > 0 {
		chunkTags := make(map[string]bool)
		for _, tag := range result.Chunk.Tags {
			chunkTags[tag] = true
		}
		
		found := false
		for _, requiredTag := range filters.Tags {
			if chunkTags[requiredTag] {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	
	// 排除標籤過濾
	if len(filters.ExcludeTags) > 0 {
		for _, excludeTag := range filters.ExcludeTags {
			for _, chunkTag := range result.Chunk.Tags {
				if chunkTag == excludeTag {
					return false
				}
			}
		}
	}
	
	// 日期範圍過濾
	if filters.DateRange != nil {
		if result.Chunk.CreatedTime.Before(filters.DateRange.Start) ||
			result.Chunk.CreatedTime.After(filters.DateRange.End) {
			return false
		}
	}
	
	// 內容長度過濾
	contentLength := len(result.Chunk.Contents)
	if filters.MinContentLength > 0 && contentLength < filters.MinContentLength {
		return false
	}
	
	if filters.MaxContentLength > 0 && contentLength > filters.MaxContentLength {
		return false
	}
	
	return true
}

// deduplicateByChunkID 根據 chunk ID 去重
func (h *HybridSearchAlgorithm) deduplicateByChunkID(results []models.MultimodalSearchResult) []models.MultimodalSearchResult {
	seen := make(map[string]bool)
	var unique []models.MultimodalSearchResult
	
	for _, result := range results {
		if !seen[result.Chunk.ChunkID] {
			seen[result.Chunk.ChunkID] = true
			unique = append(unique, result)
		}
	}
	
	return unique
}

// deduplicateByContentHash 根據內容雜湊去重
func (h *HybridSearchAlgorithm) deduplicateByContentHash(results []models.MultimodalSearchResult) []models.MultimodalSearchResult {
	seen := make(map[string]bool)
	var unique []models.MultimodalSearchResult
	
	for _, result := range results {
		contentHash := h.calculateContentHash(result.Chunk.Contents)
		if !seen[contentHash] {
			seen[contentHash] = true
			unique = append(unique, result)
		}
	}
	
	return unique
}

// deduplicateBySimilarity 根據相似度去重
func (h *HybridSearchAlgorithm) deduplicateBySimilarity(results []models.MultimodalSearchResult, threshold float64) []models.MultimodalSearchResult {
	var unique []models.MultimodalSearchResult
	
	for _, result := range results {
		isDuplicate := false
		
		for _, existing := range unique {
			similarity := h.calculateContentSimilarity(result.Chunk, existing.Chunk)
			if similarity >= threshold {
				isDuplicate = true
				break
			}
		}
		
		if !isDuplicate {
			unique = append(unique, result)
		}
	}
	
	return unique
}

// calculateContentHash 計算內容雜湊
func (h *HybridSearchAlgorithm) calculateContentHash(content string) string {
	// 簡單的字串雜湊
	hash := uint32(5381)
	for _, c := range content {
		hash = ((hash << 5) + hash) + uint32(c)
	}
	return fmt.Sprintf("%x", hash)
}

// convertToMultimodalResults 轉換為 MultimodalSearchResult
func (h *HybridSearchAlgorithm) convertToMultimodalResults(results []SearchResult) []models.MultimodalSearchResult {
	var converted []models.MultimodalSearchResult
	
	for _, result := range results {
		converted = append(converted, models.MultimodalSearchResult{
			Chunk:       result.Chunk,
			Similarity:  result.CombinedScore,
			MatchType:   result.MatchType,
			Explanation: result.Explanation,
		})
	}
	
	return converted
}

// convertFromMultimodalResults 從 MultimodalSearchResult 轉換
func (h *HybridSearchAlgorithm) convertFromMultimodalResults(results []models.MultimodalSearchResult) []SearchResult {
	var converted []SearchResult
	
	for _, result := range results {
		searchResult := SearchResult{
			Chunk:         result.Chunk,
			CombinedScore: result.Similarity,
			MatchType:     result.MatchType,
			Explanation:   result.Explanation,
		}
		
		// 根據匹配類型設定相似度
		switch result.MatchType {
		case "text":
			searchResult.TextSimilarity = result.Similarity
		case "image":
			searchResult.ImageSimilarity = result.Similarity
		case "hybrid":
			// 假設各佔一半
			searchResult.TextSimilarity = result.Similarity * 0.6
			searchResult.ImageSimilarity = result.Similarity * 0.4
		}
		
		converted = append(converted, searchResult)
	}
	
	return converted
}

// 輔助方法

// isExactMatch 檢查是否為精確匹配
func (h *HybridSearchAlgorithm) isExactMatch(chunk *models.UnifiedChunkRecord) bool {
	// 簡化實作，實際應該根據查詢內容判斷
	return false
}

// isTitleMatch 檢查是否為標題匹配
func (h *HybridSearchAlgorithm) isTitleMatch(chunk *models.UnifiedChunkRecord) bool {
	// 簡化實作
	return chunk.IsPage
}

// isTagMatch 檢查是否為標籤匹配
func (h *HybridSearchAlgorithm) isTagMatch(chunk *models.UnifiedChunkRecord) bool {
	return len(chunk.Tags) > 0
}

// isRecentContent 檢查是否為最近內容
func (h *HybridSearchAlgorithm) isRecentContent(chunk *models.UnifiedChunkRecord) bool {
	return time.Since(chunk.CreatedTime) < 30*24*time.Hour // 30 天內
}

// isOldContent 檢查是否為舊內容
func (h *HybridSearchAlgorithm) isOldContent(chunk *models.UnifiedChunkRecord) bool {
	return time.Since(chunk.CreatedTime) > 365*24*time.Hour // 1 年前
}

// isLowQuality 檢查是否為低品質內容
func (h *HybridSearchAlgorithm) isLowQuality(chunk *models.UnifiedChunkRecord) bool {
	return len(chunk.Contents) < 50 // 內容太短
}