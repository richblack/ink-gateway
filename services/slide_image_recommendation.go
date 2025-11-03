package services

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"semantic-text-processor/models"
)

// SlideImageRecommendationService Slide Generator 圖片推薦服務
type SlideImageRecommendationService struct {
	multimodalSearch MultimodalSearchService
	nlpService       NLPService
	cacheService     CacheService
	config           *SlideRecommendationConfig
}

// NewSlideImageRecommendationService 建立新的 Slide 圖片推薦服務
func NewSlideImageRecommendationService(
	multimodalSearch MultimodalSearchService,
	nlpService NLPService,
	cacheService CacheService,
	config *SlideRecommendationConfig,
) *SlideImageRecommendationService {
	if config == nil {
		config = DefaultSlideRecommendationConfig()
	}
	
	return &SlideImageRecommendationService{
		multimodalSearch: multimodalSearch,
		nlpService:       nlpService,
		cacheService:     cacheService,
		config:           config,
	}
}

// NLPService 自然語言處理服務介面
type NLPService interface {
	ExtractKeywords(text string) ([]string, error)
	ExtractEntities(text string) ([]Entity, error)
	AnalyzeSentiment(text string) (*SentimentAnalysis, error)
	SummarizeText(text string, maxLength int) (string, error)
}

// Entity 實體
type Entity struct {
	Text       string  `json:"text"`
	Type       string  `json:"type"`
	Confidence float64 `json:"confidence"`
}

// SentimentAnalysis 情感分析
type SentimentAnalysis struct {
	Sentiment  string  `json:"sentiment"` // positive, negative, neutral
	Confidence float64 `json:"confidence"`
	Score      float64 `json:"score"`
}

// SlideRecommendationConfig Slide 推薦配置
type SlideRecommendationConfig struct {
	MaxRecommendations   int     `json:"max_recommendations"`
	MinRelevanceScore    float64 `json:"min_relevance_score"`
	KeywordWeight        float64 `json:"keyword_weight"`
	EntityWeight         float64 `json:"entity_weight"`
	ContextWeight        float64 `json:"context_weight"`
	SemanticWeight       float64 `json:"semantic_weight"`
	EnableCaching        bool    `json:"enable_caching"`
	CacheTTL             time.Duration `json:"cache_ttl"`
	EnableDiversification bool    `json:"enable_diversification"`
	DiversityThreshold   float64 `json:"diversity_threshold"`
}

// DefaultSlideRecommendationConfig 預設 Slide 推薦配置
func DefaultSlideRecommendationConfig() *SlideRecommendationConfig {
	return &SlideRecommendationConfig{
		MaxRecommendations:   10,
		MinRelevanceScore:    0.6,
		KeywordWeight:        0.3,
		EntityWeight:         0.3,
		ContextWeight:        0.2,
		SemanticWeight:       0.2,
		EnableCaching:        true,
		CacheTTL:             15 * time.Minute,
		EnableDiversification: true,
		DiversityThreshold:   0.8,
	}
}

// RecommendImagesForSlide 為單張投影片推薦圖片
func (s *SlideImageRecommendationService) RecommendImagesForSlide(ctx context.Context, req *SlideImageRecommendationRequest) (*SlideImageRecommendationResponse, error) {
	startTime := time.Now()
	
	// 檢查快取
	cacheKey := s.generateCacheKey(req)
	if s.config.EnableCaching {
		var cachedResponse SlideImageRecommendationResponse
		if err := s.cacheService.Get(ctx, cacheKey, &cachedResponse); err == nil {
			cachedResponse.SearchTime = time.Since(startTime)
			cachedResponse.CacheHit = true
			return &cachedResponse, nil
		}
	}
	
	// 分析文字內容
	analysis, err := s.analyzeSlideContent(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze slide content: %w", err)
	}
	
	// 生成搜尋查詢
	searchQueries := s.generateSearchQueries(analysis, req)
	
	// 執行多重搜尋
	var allRecommendations []SlideImageRecommendation
	for _, query := range searchQueries {
		recommendations, err := s.executeSearch(ctx, query, req)
		if err != nil {
			continue // 跳過失敗的搜尋
		}
		allRecommendations = append(allRecommendations, recommendations...)
	}
	
	// 合併和排序推薦
	finalRecommendations := s.mergeAndRankRecommendations(allRecommendations, analysis, req)
	
	// 多樣化處理
	if s.config.EnableDiversification {
		finalRecommendations = s.diversifyRecommendations(finalRecommendations)
	}
	
	// 限制結果數量
	if len(finalRecommendations) > s.config.MaxRecommendations {
		finalRecommendations = finalRecommendations[:s.config.MaxRecommendations]
	}
	
	response := &SlideImageRecommendationResponse{
		Recommendations: finalRecommendations,
		TotalCount:      len(finalRecommendations),
		SearchTime:      time.Since(startTime),
		Analysis:        analysis,
	}
	
	// 快取結果
	if s.config.EnableCaching {
		s.cacheService.Set(ctx, cacheKey, response, s.config.CacheTTL)
	}
	
	return response, nil
}

// RecommendImagesForPresentation 為整個簡報推薦圖片
func (s *SlideImageRecommendationService) RecommendImagesForPresentation(ctx context.Context, req *PresentationImageRecommendationRequest) (*PresentationImageRecommendationResponse, error) {
	startTime := time.Now()
	
	var slideRecommendations []SlideRecommendations
	
	// 為每張投影片生成推薦
	for i, slide := range req.Slides {
		slideReq := &SlideImageRecommendationRequest{
			SlideTitle:       slide.Title,
			SlideContent:     slide.Content,
			SlideContext:     req.PresentationContext,
			MaxSuggestions:   req.ImagesPerSlide,
			MinRelevance:     req.MinRelevance,
			PreferredStyles:  req.PreferredStyles,
			ExcludeImageIDs:  req.ExcludeImageIDs,
		}
		
		slideResponse, err := s.RecommendImagesForSlide(ctx, slideReq)
		if err != nil {
			// 記錄錯誤但繼續處理其他投影片
			slideRecommendations = append(slideRecommendations, SlideRecommendations{
				SlideIndex:      i,
				SlideTitle:      slide.Title,
				Recommendations: []SlideImageRecommendation{},
				Error:           err.Error(),
			})
			continue
		}
		
		slideRecommendations = append(slideRecommendations, SlideRecommendations{
			SlideIndex:      i,
			SlideTitle:      slide.Title,
			Recommendations: slideResponse.Recommendations,
		})
	}
	
	// 生成整體統計
	stats := s.generatePresentationStats(slideRecommendations)
	
	return &PresentationImageRecommendationResponse{
		SlideRecommendations: slideRecommendations,
		TotalSlides:          len(req.Slides),
		SearchTime:           time.Since(startTime),
		Statistics:           stats,
	}, nil
}

// GetImageRecommendationsByTopic 根據主題推薦圖片
func (s *SlideImageRecommendationService) GetImageRecommendationsByTopic(ctx context.Context, req *TopicImageRecommendationRequest) (*TopicImageRecommendationResponse, error) {
	startTime := time.Now()
	
	var topicRecommendations []TopicRecommendations
	
	for _, topic := range req.Topics {
		// 為每個主題生成搜尋查詢
		searchReq := &models.MultimodalSearchRequest{
			TextQuery:     topic.Name,
			VectorType:    "all",
			Limit:         req.ImagesPerTopic * 2, // 取更多結果用於篩選
			MinSimilarity: req.MinRelevance,
			Filters: map[string]interface{}{
				"media_type": "image",
			},
		}
		
		// 如果有描述，加入搜尋
		if topic.Description != "" {
			searchReq.TextQuery += " " + topic.Description
		}
		
		searchResponse, err := s.multimodalSearch.SearchText(ctx, searchReq)
		if err != nil {
			topicRecommendations = append(topicRecommendations, TopicRecommendations{
				Topic:           topic,
				Recommendations: []SlideImageRecommendation{},
				Error:           err.Error(),
			})
			continue
		}
		
		// 轉換為推薦格式
		var recommendations []SlideImageRecommendation
		for _, result := range searchResponse.Results {
			if result.Chunk.IsImageChunk() {
				recommendation := SlideImageRecommendation{
					ChunkID:        result.Chunk.ChunkID,
					ImageURL:       result.Chunk.GetImageURL(),
					Title:          s.extractImageTitle(result.Chunk),
					Description:    result.Chunk.Contents,
					RelevanceScore: result.Similarity,
					MatchReason:    fmt.Sprintf("與主題 '%s' 相關", topic.Name),
					Tags:           result.Chunk.Tags,
					ImageMetadata:  s.extractImageMetadata(result.Chunk),
				}
				recommendations = append(recommendations, recommendation)
			}
		}
		
		// 限制數量
		if len(recommendations) > req.ImagesPerTopic {
			recommendations = recommendations[:req.ImagesPerTopic]
		}
		
		topicRecommendations = append(topicRecommendations, TopicRecommendations{
			Topic:           topic,
			Recommendations: recommendations,
		})
	}
	
	return &TopicImageRecommendationResponse{
		TopicRecommendations: topicRecommendations,
		TotalTopics:          len(req.Topics),
		SearchTime:           time.Since(startTime),
	}, nil
}

// 資料結構

// SlideImageRecommendationRequest Slide 圖片推薦請求
type SlideImageRecommendationRequest struct {
	SlideTitle       string   `json:"slide_title"`
	SlideContent     string   `json:"slide_content"`
	SlideContext     string   `json:"slide_context"`
	MaxSuggestions   int      `json:"max_suggestions"`
	MinRelevance     float64  `json:"min_relevance"`
	PreferredStyles  []string `json:"preferred_styles"`
	ExcludeImageIDs  []string `json:"exclude_image_ids"`
}

// SlideImageRecommendationResponse Slide 圖片推薦回應
type SlideImageRecommendationResponse struct {
	Recommendations []SlideImageRecommendation `json:"recommendations"`
	TotalCount      int                        `json:"total_count"`
	SearchTime      time.Duration              `json:"search_time"`
	Analysis        *SlideContentAnalysis      `json:"analysis,omitempty"`
	CacheHit        bool                       `json:"cache_hit"`
}

// SlideImageRecommendation Slide 圖片推薦
type SlideImageRecommendation struct {
	ChunkID        string                 `json:"chunk_id"`
	ImageURL       string                 `json:"image_url"`
	Title          string                 `json:"title"`
	Description    string                 `json:"description"`
	RelevanceScore float64                `json:"relevance_score"`
	MatchReason    string                 `json:"match_reason"`
	Tags           []string               `json:"tags"`
	ImageMetadata  map[string]interface{} `json:"image_metadata,omitempty"`
}

// SlideContentAnalysis Slide 內容分析
type SlideContentAnalysis struct {
	Keywords   []string           `json:"keywords"`
	Entities   []Entity           `json:"entities"`
	Sentiment  *SentimentAnalysis `json:"sentiment,omitempty"`
	MainTopic  string             `json:"main_topic"`
	Concepts   []string           `json:"concepts"`
}

// PresentationImageRecommendationRequest 簡報圖片推薦請求
type PresentationImageRecommendationRequest struct {
	Slides              []SlideInfo `json:"slides"`
	PresentationContext string      `json:"presentation_context"`
	ImagesPerSlide      int         `json:"images_per_slide"`
	MinRelevance        float64     `json:"min_relevance"`
	PreferredStyles     []string    `json:"preferred_styles"`
	ExcludeImageIDs     []string    `json:"exclude_image_ids"`
}

// SlideInfo 投影片資訊
type SlideInfo struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

// PresentationImageRecommendationResponse 簡報圖片推薦回應
type PresentationImageRecommendationResponse struct {
	SlideRecommendations []SlideRecommendations    `json:"slide_recommendations"`
	TotalSlides          int                       `json:"total_slides"`
	SearchTime           time.Duration             `json:"search_time"`
	Statistics           *PresentationStatistics   `json:"statistics"`
}

// SlideRecommendations 投影片推薦
type SlideRecommendations struct {
	SlideIndex      int                        `json:"slide_index"`
	SlideTitle      string                     `json:"slide_title"`
	Recommendations []SlideImageRecommendation `json:"recommendations"`
	Error           string                     `json:"error,omitempty"`
}

// PresentationStatistics 簡報統計
type PresentationStatistics struct {
	TotalRecommendations int                    `json:"total_recommendations"`
	AverageRelevance     float64                `json:"average_relevance"`
	TopTags              []TagStatistic         `json:"top_tags"`
	CoverageRate         float64                `json:"coverage_rate"`
}

// TagStatistic 標籤統計
type TagStatistic struct {
	Tag   string `json:"tag"`
	Count int    `json:"count"`
}

// TopicImageRecommendationRequest 主題圖片推薦請求
type TopicImageRecommendationRequest struct {
	Topics          []Topic `json:"topics"`
	ImagesPerTopic  int     `json:"images_per_topic"`
	MinRelevance    float64 `json:"min_relevance"`
	ExcludeImageIDs []string `json:"exclude_image_ids"`
}

// Topic 主題
type Topic struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// TopicImageRecommendationResponse 主題圖片推薦回應
type TopicImageRecommendationResponse struct {
	TopicRecommendations []TopicRecommendations `json:"topic_recommendations"`
	TotalTopics          int                    `json:"total_topics"`
	SearchTime           time.Duration          `json:"search_time"`
}

// TopicRecommendations 主題推薦
type TopicRecommendations struct {
	Topic           Topic                      `json:"topic"`
	Recommendations []SlideImageRecommendation `json:"recommendations"`
	Error           string                     `json:"error,omitempty"`
}

// SearchQuery 搜尋查詢
type SearchQuery struct {
	Query      string  `json:"query"`
	Weight     float64 `json:"weight"`
	QueryType  string  `json:"query_type"` // keyword, entity, semantic
}

// 私有方法

// analyzeSlideContent 分析投影片內容
func (s *SlideImageRecommendationService) analyzeSlideContent(ctx context.Context, req *SlideImageRecommendationRequest) (*SlideContentAnalysis, error) {
	fullText := req.SlideTitle + " " + req.SlideContent + " " + req.SlideContext
	
	analysis := &SlideContentAnalysis{}
	
	// 提取關鍵字
	if s.nlpService != nil {
		keywords, err := s.nlpService.ExtractKeywords(fullText)
		if err == nil {
			analysis.Keywords = keywords
		}
		
		// 提取實體
		entities, err := s.nlpService.ExtractEntities(fullText)
		if err == nil {
			analysis.Entities = entities
		}
		
		// 情感分析
		sentiment, err := s.nlpService.AnalyzeSentiment(fullText)
		if err == nil {
			analysis.Sentiment = sentiment
		}
	} else {
		// 簡單的關鍵字提取
		analysis.Keywords = s.simpleKeywordExtraction(fullText)
	}
	
	// 確定主要主題
	analysis.MainTopic = s.determineMainTopic(req.SlideTitle, analysis.Keywords)
	
	// 提取概念
	analysis.Concepts = s.extractConcepts(fullText, analysis.Keywords)
	
	return analysis, nil
}

// generateSearchQueries 生成搜尋查詢
func (s *SlideImageRecommendationService) generateSearchQueries(analysis *SlideContentAnalysis, req *SlideImageRecommendationRequest) []SearchQuery {
	var queries []SearchQuery
	
	// 基於關鍵字的查詢
	for _, keyword := range analysis.Keywords {
		queries = append(queries, SearchQuery{
			Query:     keyword,
			Weight:    s.config.KeywordWeight,
			QueryType: "keyword",
		})
	}
	
	// 基於實體的查詢
	for _, entity := range analysis.Entities {
		queries = append(queries, SearchQuery{
			Query:     entity.Text,
			Weight:    s.config.EntityWeight * entity.Confidence,
			QueryType: "entity",
		})
	}
	
	// 基於主題的查詢
	if analysis.MainTopic != "" {
		queries = append(queries, SearchQuery{
			Query:     analysis.MainTopic,
			Weight:    s.config.ContextWeight,
			QueryType: "topic",
		})
	}
	
	// 語義查詢（使用完整內容）
	if req.SlideContent != "" {
		queries = append(queries, SearchQuery{
			Query:     req.SlideContent,
			Weight:    s.config.SemanticWeight,
			QueryType: "semantic",
		})
	}
	
	return queries
}

// executeSearch 執行搜尋
func (s *SlideImageRecommendationService) executeSearch(ctx context.Context, query SearchQuery, req *SlideImageRecommendationRequest) ([]SlideImageRecommendation, error) {
	searchReq := &models.MultimodalSearchRequest{
		TextQuery:     query.Query,
		VectorType:    "all",
		Limit:         req.MaxSuggestions * 3, // 取更多結果用於篩選
		MinSimilarity: req.MinRelevance * 0.8, // 稍微降低閾值
		Filters: map[string]interface{}{
			"media_type": "image",
		},
	}
	
	searchResponse, err := s.multimodalSearch.SearchText(ctx, searchReq)
	if err != nil {
		return nil, err
	}
	
	var recommendations []SlideImageRecommendation
	for _, result := range searchResponse.Results {
		if result.Chunk.IsImageChunk() {
			// 檢查是否在排除列表中
			if s.isExcluded(result.Chunk.ChunkID, req.ExcludeImageIDs) {
				continue
			}
			
			recommendation := SlideImageRecommendation{
				ChunkID:        result.Chunk.ChunkID,
				ImageURL:       result.Chunk.GetImageURL(),
				Title:          s.extractImageTitle(result.Chunk),
				Description:    result.Chunk.Contents,
				RelevanceScore: result.Similarity * query.Weight,
				MatchReason:    s.generateMatchReason(query, result.Similarity),
				Tags:           result.Chunk.Tags,
				ImageMetadata:  s.extractImageMetadata(result.Chunk),
			}
			recommendations = append(recommendations, recommendation)
		}
	}
	
	return recommendations, nil
}

// mergeAndRankRecommendations 合併和排序推薦
func (s *SlideImageRecommendationService) mergeAndRankRecommendations(recommendations []SlideImageRecommendation, analysis *SlideContentAnalysis, req *SlideImageRecommendationRequest) []SlideImageRecommendation {
	// 去重（基於 ChunkID）
	uniqueRecommendations := make(map[string]SlideImageRecommendation)
	
	for _, rec := range recommendations {
		if existing, exists := uniqueRecommendations[rec.ChunkID]; exists {
			// 合併分數（取較高者）
			if rec.RelevanceScore > existing.RelevanceScore {
				uniqueRecommendations[rec.ChunkID] = rec
			}
		} else {
			uniqueRecommendations[rec.ChunkID] = rec
		}
	}
	
	// 轉換為切片
	var finalRecommendations []SlideImageRecommendation
	for _, rec := range uniqueRecommendations {
		finalRecommendations = append(finalRecommendations, rec)
	}
	
	// 應用額外的評分因子
	for i := range finalRecommendations {
		s.applyAdditionalScoring(&finalRecommendations[i], analysis, req)
	}
	
	// 按相關度排序
	sort.Slice(finalRecommendations, func(i, j int) bool {
		return finalRecommendations[i].RelevanceScore > finalRecommendations[j].RelevanceScore
	})
	
	return finalRecommendations
}

// diversifyRecommendations 多樣化推薦
func (s *SlideImageRecommendationService) diversifyRecommendations(recommendations []SlideImageRecommendation) []SlideImageRecommendation {
	if len(recommendations) <= 1 {
		return recommendations
	}
	
	var diversified []SlideImageRecommendation
	diversified = append(diversified, recommendations[0]) // 保留最高分的
	
	for _, candidate := range recommendations[1:] {
		isDiverse := true
		
		for _, selected := range diversified {
			similarity := s.calculateTagSimilarity(candidate.Tags, selected.Tags)
			if similarity > s.config.DiversityThreshold {
				isDiverse = false
				break
			}
		}
		
		if isDiverse {
			diversified = append(diversified, candidate)
		}
	}
	
	return diversified
}

// 輔助方法

// simpleKeywordExtraction 簡單關鍵字提取
func (s *SlideImageRecommendationService) simpleKeywordExtraction(text string) []string {
	words := strings.Fields(strings.ToLower(text))
	
	// 停用詞
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

// determineMainTopic 確定主要主題
func (s *SlideImageRecommendationService) determineMainTopic(title string, keywords []string) string {
	if title != "" {
		return title
	}
	
	if len(keywords) > 0 {
		return keywords[0]
	}
	
	return ""
}

// extractConcepts 提取概念
func (s *SlideImageRecommendationService) extractConcepts(text string, keywords []string) []string {
	// 簡化實作，實際應該使用更複雜的 NLP 技術
	return keywords
}

// extractImageTitle 提取圖片標題
func (s *SlideImageRecommendationService) extractImageTitle(chunk *models.UnifiedChunkRecord) string {
	if chunk.Metadata != nil {
		if storage, ok := chunk.Metadata["storage"].(map[string]interface{}); ok {
			if filename, ok := storage["original_filename"].(string); ok {
				return filename
			}
		}
	}
	
	// 使用內容的前幾個字作為標題
	if len(chunk.Contents) > 50 {
		return chunk.Contents[:50] + "..."
	}
	
	return chunk.Contents
}

// extractImageMetadata 提取圖片元資料
func (s *SlideImageRecommendationService) extractImageMetadata(chunk *models.UnifiedChunkRecord) map[string]interface{} {
	if chunk.Metadata == nil {
		return nil
	}
	
	metadata := make(map[string]interface{})
	
	// 提取圖片屬性
	if imageProps, ok := chunk.Metadata["image_properties"].(map[string]interface{}); ok {
		metadata["format"] = imageProps["format"]
		metadata["width"] = imageProps["width"]
		metadata["height"] = imageProps["height"]
		metadata["size_bytes"] = imageProps["size_bytes"]
	}
	
	// 提取 AI 分析
	if aiAnalysis, ok := chunk.Metadata["ai_analysis"].(map[string]interface{}); ok {
		metadata["ai_description"] = aiAnalysis["description"]
		metadata["ai_tags"] = aiAnalysis["tags"]
		metadata["ai_confidence"] = aiAnalysis["confidence"]
	}
	
	return metadata
}

// generateMatchReason 生成匹配原因
func (s *SlideImageRecommendationService) generateMatchReason(query SearchQuery, similarity float64) string {
	switch query.QueryType {
	case "keyword":
		return fmt.Sprintf("關鍵字 '%s' 匹配 (%.1f%%)", query.Query, similarity*100)
	case "entity":
		return fmt.Sprintf("實體 '%s' 匹配 (%.1f%%)", query.Query, similarity*100)
	case "topic":
		return fmt.Sprintf("主題 '%s' 相關 (%.1f%%)", query.Query, similarity*100)
	case "semantic":
		return fmt.Sprintf("語義相關 (%.1f%%)", similarity*100)
	default:
		return fmt.Sprintf("內容相關 (%.1f%%)", similarity*100)
	}
}

// applyAdditionalScoring 應用額外評分因子
func (s *SlideImageRecommendationService) applyAdditionalScoring(rec *SlideImageRecommendation, analysis *SlideContentAnalysis, req *SlideImageRecommendationRequest) {
	// 標籤匹配獎勵
	tagBonus := s.calculateTagBonus(rec.Tags, analysis.Keywords)
	rec.RelevanceScore += tagBonus * 0.1
	
	// 最近內容獎勵（如果有時間資訊）
	// 這裡可以根據圖片的建立時間給予獎勵
	
	// 品質獎勵（基於描述長度和標籤數量）
	qualityBonus := s.calculateQualityBonus(rec)
	rec.RelevanceScore += qualityBonus * 0.05
}

// calculateTagBonus 計算標籤獎勵
func (s *SlideImageRecommendationService) calculateTagBonus(imageTags, keywords []string) float64 {
	if len(imageTags) == 0 || len(keywords) == 0 {
		return 0.0
	}
	
	keywordSet := make(map[string]bool)
	for _, keyword := range keywords {
		keywordSet[strings.ToLower(keyword)] = true
	}
	
	var matches int
	for _, tag := range imageTags {
		if keywordSet[strings.ToLower(tag)] {
			matches++
		}
	}
	
	return float64(matches) / float64(len(keywords))
}

// calculateQualityBonus 計算品質獎勵
func (s *SlideImageRecommendationService) calculateQualityBonus(rec *SlideImageRecommendation) float64 {
	bonus := 0.0
	
	// 描述長度獎勵
	if len(rec.Description) > 100 {
		bonus += 0.3
	} else if len(rec.Description) > 50 {
		bonus += 0.1
	}
	
	// 標籤數量獎勵
	if len(rec.Tags) > 3 {
		bonus += 0.2
	} else if len(rec.Tags) > 0 {
		bonus += 0.1
	}
	
	return bonus
}

// calculateTagSimilarity 計算標籤相似度
func (s *SlideImageRecommendationService) calculateTagSimilarity(tags1, tags2 []string) float64 {
	if len(tags1) == 0 || len(tags2) == 0 {
		return 0.0
	}
	
	tagSet1 := make(map[string]bool)
	for _, tag := range tags1 {
		tagSet1[tag] = true
	}
	
	var commonTags int
	for _, tag := range tags2 {
		if tagSet1[tag] {
			commonTags++
		}
	}
	
	totalTags := len(tags1) + len(tags2) - commonTags
	if totalTags == 0 {
		return 0.0
	}
	
	return float64(commonTags) / float64(totalTags)
}

// isExcluded 檢查是否被排除
func (s *SlideImageRecommendationService) isExcluded(chunkID string, excludeList []string) bool {
	for _, excludeID := range excludeList {
		if chunkID == excludeID {
			return true
		}
	}
	return false
}

// generateCacheKey 生成快取鍵
func (s *SlideImageRecommendationService) generateCacheKey(req *SlideImageRecommendationRequest) string {
	return fmt.Sprintf("slide_rec:%s:%s:%d:%.2f", 
		req.SlideTitle, req.SlideContent, req.MaxSuggestions, req.MinRelevance)
}

// generatePresentationStats 生成簡報統計
func (s *SlideImageRecommendationService) generatePresentationStats(slideRecommendations []SlideRecommendations) *PresentationStatistics {
	stats := &PresentationStatistics{
		TopTags: make([]TagStatistic, 0),
	}
	
	var totalRecommendations int
	var totalRelevance float64
	var slidesWithRecommendations int
	tagCounts := make(map[string]int)
	
	for _, slide := range slideRecommendations {
		if len(slide.Recommendations) > 0 {
			slidesWithRecommendations++
			totalRecommendations += len(slide.Recommendations)
			
			for _, rec := range slide.Recommendations {
				totalRelevance += rec.RelevanceScore
				
				for _, tag := range rec.Tags {
					tagCounts[tag]++
				}
			}
		}
	}
	
	stats.TotalRecommendations = totalRecommendations
	
	if totalRecommendations > 0 {
		stats.AverageRelevance = totalRelevance / float64(totalRecommendations)
	}
	
	if len(slideRecommendations) > 0 {
		stats.CoverageRate = float64(slidesWithRecommendations) / float64(len(slideRecommendations))
	}
	
	// 生成 Top 標籤
	type tagCount struct {
		tag   string
		count int
	}
	
	var tagList []tagCount
	for tag, count := range tagCounts {
		tagList = append(tagList, tagCount{tag: tag, count: count})
	}
	
	sort.Slice(tagList, func(i, j int) bool {
		return tagList[i].count > tagList[j].count
	})
	
	maxTags := 10
	if len(tagList) < maxTags {
		maxTags = len(tagList)
	}
	
	for i := 0; i < maxTags; i++ {
		stats.TopTags = append(stats.TopTags, TagStatistic{
			Tag:   tagList[i].tag,
			Count: tagList[i].count,
		})
	}
	
	return stats
}