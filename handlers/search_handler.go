package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"semantic-text-processor/models"
	"semantic-text-processor/services"
)

// SearchHandler 搜尋處理器
type SearchHandler struct {
	multimodalSearch    services.MultimodalSearchService
	imageSimilarity     *services.ImageSimilaritySearch
	slideRecommendation *services.SlideImageRecommendationService
	cacheEnabled        bool
}

// ErrorResponse 錯誤回應結構
type ErrorResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
	Details string `json:"details,omitempty"`
}

// NewSearchHandler 建立新的搜尋處理器
func NewSearchHandler(
	multimodalSearch services.MultimodalSearchService,
	imageSimilarity *services.ImageSimilaritySearch,
	slideRecommendation *services.SlideImageRecommendationService,
) *SearchHandler {
	return &SearchHandler{
		multimodalSearch:    multimodalSearch,
		imageSimilarity:     imageSimilarity,
		slideRecommendation: slideRecommendation,
		cacheEnabled:        true,
	}
}

// MultimodalSearch 多模態搜尋
// POST /api/v1/search/multimodal
func (s *SearchHandler) MultimodalSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed", nil)
		return
	}

	var searchReq MultimodalSearchRequest
	if err := json.NewDecoder(r.Body).Decode(&searchReq); err != nil {
		s.writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON request", err)
		return
	}

	// 驗證請求
	if searchReq.TextQuery == "" && searchReq.ImageQuery == "" {
		s.writeErrorResponse(w, http.StatusBadRequest, "Either text_query or image_query is required", nil)
		return
	}

	// 建立搜尋請求
	req := &models.MultimodalSearchRequest{
		TextQuery:     searchReq.TextQuery,
		ImageQuery:    searchReq.ImageQuery,
		VectorType:    searchReq.VectorType,
		Weights:       searchReq.Weights,
		Filters:       searchReq.Filters,
		Limit:         searchReq.Limit,
		MinSimilarity: searchReq.MinSimilarity,
	}

	// 設定預設值
	if req.Limit <= 0 {
		req.Limit = 20
	}
	if req.MinSimilarity <= 0 {
		req.MinSimilarity = 0.7
	}
	if req.VectorType == "" {
		req.VectorType = "all"
	}

	// 執行搜尋
	var searchResponse *models.MultimodalSearchResponse
	var err error

	switch searchReq.SearchType {
	case "text":
		searchResponse, err = s.multimodalSearch.SearchText(r.Context(), req)
	case "image":
		searchResponse, err = s.multimodalSearch.SearchImages(r.Context(), req)
	case "hybrid", "":
		searchResponse, err = s.multimodalSearch.HybridSearch(r.Context(), req)
	default:
		s.writeErrorResponse(w, http.StatusBadRequest, "Invalid search_type", nil)
		return
	}

	if err != nil {
		s.writeErrorResponse(w, http.StatusInternalServerError, "Search failed", err)
		return
	}

	// 轉換回應格式
	response := s.convertSearchResponse(searchResponse)
	s.writeJSONResponse(w, http.StatusOK, response)
}

// SearchByImage 以圖搜圖
// POST /api/v1/search/image-similarity
func (s *SearchHandler) SearchByImage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed", nil)
		return
	}

	var searchReq ImageSimilaritySearchRequest
	if err := json.NewDecoder(r.Body).Decode(&searchReq); err != nil {
		s.writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON request", err)
		return
	}

	// 驗證請求
	if searchReq.ImageURL == "" && searchReq.ChunkID == "" {
		s.writeErrorResponse(w, http.StatusBadRequest, "Either image_url or chunk_id is required", nil)
		return
	}

	// 建立搜尋選項
	options := &services.ImageSearchOptions{
		Limit:               searchReq.Limit,
		SimilarityThreshold: searchReq.MinSimilarity,
		IncludeMetadata:     searchReq.IncludeMetadata,
		ExcludeChunkIDs:     searchReq.ExcludeChunkIDs,
		FilterTags:          searchReq.FilterTags,
		SortBy:              searchReq.SortBy,
		SortOrder:           searchReq.SortOrder,
	}

	// 設定預設值
	if options.Limit <= 0 {
		options.Limit = 20
	}
	if options.SimilarityThreshold <= 0 {
		options.SimilarityThreshold = 0.7
	}
	if options.SortBy == "" {
		options.SortBy = "similarity"
	}
	if options.SortOrder == "" {
		options.SortOrder = "desc"
	}

	// 執行搜尋
	var searchResponse *services.ImageSearchResponse
	var err error

	if searchReq.ChunkID != "" {
		searchResponse, err = s.imageSimilarity.SearchByChunkID(r.Context(), searchReq.ChunkID, options)
	} else {
		searchResponse, err = s.imageSimilarity.SearchByImageURL(r.Context(), searchReq.ImageURL, options)
	}

	if err != nil {
		s.writeErrorResponse(w, http.StatusInternalServerError, "Image similarity search failed", err)
		return
	}

	// 建立回應
	response := &ImageSimilaritySearchResponse{
		Success:     true,
		Results:     searchResponse.Results,
		TotalCount:  searchResponse.TotalCount,
		SearchTime:  searchResponse.SearchTime.String(),
		QueryType:   searchResponse.QueryType,
		QuerySource: searchResponse.QuerySource,
		CacheHit:    searchResponse.CacheHit,
		Message:     "Image similarity search completed successfully",
	}

	s.writeJSONResponse(w, http.StatusOK, response)
}

// RecommendImagesForSlide 為投影片推薦圖片
// POST /api/v1/search/slide-recommendations
func (s *SearchHandler) RecommendImagesForSlide(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed", nil)
		return
	}

	var slideReq SlideRecommendationRequest
	if err := json.NewDecoder(r.Body).Decode(&slideReq); err != nil {
		s.writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON request", err)
		return
	}

	// 驗證請求
	if slideReq.SlideContent == "" && slideReq.SlideTitle == "" {
		s.writeErrorResponse(w, http.StatusBadRequest, "Either slide_title or slide_content is required", nil)
		return
	}

	// 建立推薦請求
	req := &services.SlideImageRecommendationRequest{
		SlideTitle:      slideReq.SlideTitle,
		SlideContent:    slideReq.SlideContent,
		SlideContext:    slideReq.SlideContext,
		MaxSuggestions:  slideReq.MaxSuggestions,
		MinRelevance:    slideReq.MinRelevance,
		PreferredStyles: slideReq.PreferredStyles,
		ExcludeImageIDs: slideReq.ExcludeImageIDs,
	}

	// 設定預設值
	if req.MaxSuggestions <= 0 {
		req.MaxSuggestions = 10
	}
	if req.MinRelevance <= 0 {
		req.MinRelevance = 0.6
	}

	// 執行推薦
	recommendationResponse, err := s.slideRecommendation.RecommendImagesForSlide(r.Context(), req)
	if err != nil {
		s.writeErrorResponse(w, http.StatusInternalServerError, "Slide recommendation failed", err)
		return
	}

	// 建立回應
	response := &SlideRecommendationResponse{
		Success:         true,
		Recommendations: recommendationResponse.Recommendations,
		TotalCount:      recommendationResponse.TotalCount,
		SearchTime:      recommendationResponse.SearchTime.String(),
		Analysis:        recommendationResponse.Analysis,
		CacheHit:        recommendationResponse.CacheHit,
		Message:         "Slide image recommendations generated successfully",
	}

	s.writeJSONResponse(w, http.StatusOK, response)
}

// RecommendImagesForPresentation 為整個簡報推薦圖片
// POST /api/v1/search/presentation-recommendations
func (s *SearchHandler) RecommendImagesForPresentation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed", nil)
		return
	}

	var presentationReq PresentationRecommendationRequest
	if err := json.NewDecoder(r.Body).Decode(&presentationReq); err != nil {
		s.writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON request", err)
		return
	}

	// 驗證請求
	if len(presentationReq.Slides) == 0 {
		s.writeErrorResponse(w, http.StatusBadRequest, "At least one slide is required", nil)
		return
	}

	// 建立推薦請求
	req := &services.PresentationImageRecommendationRequest{
		Slides:              presentationReq.Slides,
		PresentationContext: presentationReq.PresentationContext,
		ImagesPerSlide:      presentationReq.ImagesPerSlide,
		MinRelevance:        presentationReq.MinRelevance,
		PreferredStyles:     presentationReq.PreferredStyles,
		ExcludeImageIDs:     presentationReq.ExcludeImageIDs,
	}

	// 設定預設值
	if req.ImagesPerSlide <= 0 {
		req.ImagesPerSlide = 5
	}
	if req.MinRelevance <= 0 {
		req.MinRelevance = 0.6
	}

	// 執行推薦
	recommendationResponse, err := s.slideRecommendation.RecommendImagesForPresentation(r.Context(), req)
	if err != nil {
		s.writeErrorResponse(w, http.StatusInternalServerError, "Presentation recommendation failed", err)
		return
	}

	// 建立回應
	response := &PresentationRecommendationResponse{
		Success:              true,
		SlideRecommendations: recommendationResponse.SlideRecommendations,
		TotalSlides:          recommendationResponse.TotalSlides,
		SearchTime:           recommendationResponse.SearchTime.String(),
		Statistics:           recommendationResponse.Statistics,
		Message:              "Presentation image recommendations generated successfully",
	}

	s.writeJSONResponse(w, http.StatusOK, response)
}

// FindDuplicateImages 尋找重複圖片
// POST /api/v1/search/duplicates
func (s *SearchHandler) FindDuplicateImages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed", nil)
		return
	}

	var duplicateReq DuplicateSearchRequest
	if err := json.NewDecoder(r.Body).Decode(&duplicateReq); err != nil {
		s.writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON request", err)
		return
	}

	// 建立搜尋選項
	options := &services.DuplicateSearchOptions{
		SimilarityThreshold: duplicateReq.SimilarityThreshold,
		MinGroupSize:        duplicateReq.MinGroupSize,
		IncludeMetadata:     duplicateReq.IncludeMetadata,
	}

	// 設定預設值
	if options.SimilarityThreshold <= 0 {
		options.SimilarityThreshold = 0.95
	}
	if options.MinGroupSize <= 0 {
		options.MinGroupSize = 2
	}

	// 執行重複圖片搜尋
	duplicateResponse, err := s.imageSimilarity.FindDuplicateImages(r.Context(), options)
	if err != nil {
		s.writeErrorResponse(w, http.StatusInternalServerError, "Duplicate search failed", err)
		return
	}

	// 建立回應
	response := &DuplicateSearchResponse{
		Success:     true,
		Groups:      duplicateResponse.Groups,
		TotalGroups: duplicateResponse.TotalGroups,
		SearchTime:  duplicateResponse.SearchTime.String(),
		Message:     "Duplicate image search completed successfully",
	}

	s.writeJSONResponse(w, http.StatusOK, response)
}

// GetSimilarImages 取得相似圖片
// GET /api/v1/search/similar/{chunk_id}
func (s *SearchHandler) GetSimilarImages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed", nil)
		return
	}

	chunkID := s.extractChunkIDFromPath(r.URL.Path)
	if chunkID == "" {
		s.writeErrorResponse(w, http.StatusBadRequest, "Invalid chunk ID", nil)
		return
	}

	// 解析查詢參數
	countStr := r.URL.Query().Get("count")
	count := 10 // 預設值
	if countStr != "" {
		if parsedCount, err := strconv.Atoi(countStr); err == nil && parsedCount > 0 {
			count = parsedCount
		}
	}

	// 取得相似圖片
	similarResponse, err := s.imageSimilarity.GetSimilarImages(r.Context(), chunkID, count)
	if err != nil {
		s.writeErrorResponse(w, http.StatusInternalServerError, "Failed to get similar images", err)
		return
	}

	// 建立回應
	response := &SimilarImagesResponse{
		Success:         true,
		ChunkID:         chunkID,
		Recommendations: similarResponse.Recommendations,
		TotalCount:      similarResponse.TotalCount,
		SearchTime:      similarResponse.SearchTime.String(),
		Message:         "Similar images retrieved successfully",
	}

	s.writeJSONResponse(w, http.StatusOK, response)
}

// 資料結構

// MultimodalSearchRequest 多模態搜尋請求
type MultimodalSearchRequest struct {
	TextQuery     string                     `json:"text_query,omitempty"`
	ImageQuery    string                     `json:"image_query,omitempty"`
	SearchType    string                     `json:"search_type,omitempty"` // text, image, hybrid
	VectorType    string                     `json:"vector_type,omitempty"`
	Weights       *models.SearchWeights      `json:"weights,omitempty"`
	Filters       map[string]interface{}     `json:"filters,omitempty"`
	Limit         int                        `json:"limit,omitempty"`
	MinSimilarity float64                    `json:"min_similarity,omitempty"`
}

// MultimodalSearchResponse 多模態搜尋回應
type MultimodalSearchResponse struct {
	Success     bool                              `json:"success"`
	Results     []MultimodalSearchResultResponse  `json:"results"`
	TotalCount  int                               `json:"total_count"`
	SearchTime  string                            `json:"search_time"`
	Query       string                            `json:"query"`
	MatchTypes  []string                          `json:"match_types"`
	Message     string                            `json:"message"`
}

// MultimodalSearchResultResponse 多模態搜尋結果回應
type MultimodalSearchResultResponse struct {
	ChunkID     string                 `json:"chunk_id"`
	Content     string                 `json:"content"`
	ImageURL    string                 `json:"image_url,omitempty"`
	Similarity  float64                `json:"similarity"`
	MatchType   string                 `json:"match_type"`
	Explanation string                 `json:"explanation,omitempty"`
	Tags        []string               `json:"tags,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt   string                 `json:"created_at,omitempty"`
}

// ImageSimilaritySearchRequest 圖片相似度搜尋請求
type ImageSimilaritySearchRequest struct {
	ImageURL        string   `json:"image_url,omitempty"`
	ChunkID         string   `json:"chunk_id,omitempty"`
	Limit           int      `json:"limit,omitempty"`
	MinSimilarity   float64  `json:"min_similarity,omitempty"`
	IncludeMetadata bool     `json:"include_metadata,omitempty"`
	ExcludeChunkIDs []string `json:"exclude_chunk_ids,omitempty"`
	FilterTags      []string `json:"filter_tags,omitempty"`
	SortBy          string   `json:"sort_by,omitempty"`
	SortOrder       string   `json:"sort_order,omitempty"`
}

// ImageSimilaritySearchResponse 圖片相似度搜尋回應
type ImageSimilaritySearchResponse struct {
	Success     bool                            `json:"success"`
	Results     []services.ImageSearchResult    `json:"results"`
	TotalCount  int                             `json:"total_count"`
	SearchTime  string                          `json:"search_time"`
	QueryType   string                          `json:"query_type"`
	QuerySource string                          `json:"query_source"`
	CacheHit    bool                            `json:"cache_hit"`
	Message     string                          `json:"message"`
}

// SlideRecommendationRequest 投影片推薦請求
type SlideRecommendationRequest struct {
	SlideTitle      string   `json:"slide_title,omitempty"`
	SlideContent    string   `json:"slide_content,omitempty"`
	SlideContext    string   `json:"slide_context,omitempty"`
	MaxSuggestions  int      `json:"max_suggestions,omitempty"`
	MinRelevance    float64  `json:"min_relevance,omitempty"`
	PreferredStyles []string `json:"preferred_styles,omitempty"`
	ExcludeImageIDs []string `json:"exclude_image_ids,omitempty"`
}

// SlideRecommendationResponse 投影片推薦回應
type SlideRecommendationResponse struct {
	Success         bool                                    `json:"success"`
	Recommendations []services.SlideImageRecommendation     `json:"recommendations"`
	TotalCount      int                                     `json:"total_count"`
	SearchTime      string                                  `json:"search_time"`
	Analysis        *services.SlideContentAnalysis          `json:"analysis,omitempty"`
	CacheHit        bool                                    `json:"cache_hit"`
	Message         string                                  `json:"message"`
}

// PresentationRecommendationRequest 簡報推薦請求
type PresentationRecommendationRequest struct {
	Slides              []services.SlideInfo `json:"slides"`
	PresentationContext string               `json:"presentation_context,omitempty"`
	ImagesPerSlide      int                  `json:"images_per_slide,omitempty"`
	MinRelevance        float64              `json:"min_relevance,omitempty"`
	PreferredStyles     []string             `json:"preferred_styles,omitempty"`
	ExcludeImageIDs     []string             `json:"exclude_image_ids,omitempty"`
}

// PresentationRecommendationResponse 簡報推薦回應
type PresentationRecommendationResponse struct {
	Success              bool                                `json:"success"`
	SlideRecommendations []services.SlideRecommendations     `json:"slide_recommendations"`
	TotalSlides          int                                 `json:"total_slides"`
	SearchTime           string                              `json:"search_time"`
	Statistics           *services.PresentationStatistics   `json:"statistics,omitempty"`
	Message              string                              `json:"message"`
}

// DuplicateSearchRequest 重複搜尋請求
type DuplicateSearchRequest struct {
	SimilarityThreshold float64 `json:"similarity_threshold,omitempty"`
	MinGroupSize        int     `json:"min_group_size,omitempty"`
	IncludeMetadata     bool    `json:"include_metadata,omitempty"`
}

// DuplicateSearchResponse 重複搜尋回應
type DuplicateSearchResponse struct {
	Success     bool                        `json:"success"`
	Groups      []services.DuplicateGroup   `json:"groups"`
	TotalGroups int                         `json:"total_groups"`
	SearchTime  string                      `json:"search_time"`
	Message     string                      `json:"message"`
}

// SimilarImagesResponse 相似圖片回應
type SimilarImagesResponse struct {
	Success         bool                              `json:"success"`
	ChunkID         string                            `json:"chunk_id"`
	Recommendations []services.ImageRecommendation    `json:"recommendations"`
	TotalCount      int                               `json:"total_count"`
	SearchTime      string                            `json:"search_time"`
	Message         string                            `json:"message"`
}

// 私有方法

// convertSearchResponse 轉換搜尋回應格式
func (s *SearchHandler) convertSearchResponse(searchResponse *models.MultimodalSearchResponse) *MultimodalSearchResponse {
	var results []MultimodalSearchResultResponse
	
	for _, result := range searchResponse.Results {
		resultResponse := MultimodalSearchResultResponse{
			ChunkID:     result.Chunk.ChunkID,
			Content:     result.Chunk.Contents,
			Similarity:  result.Similarity,
			MatchType:   result.MatchType,
			Explanation: result.Explanation,
			Tags:        result.Chunk.Tags,
			Metadata:    result.Chunk.Metadata,
			CreatedAt:   result.Chunk.CreatedTime.Format("2006-01-02T15:04:05Z07:00"),
		}
		
		// 如果是圖片 chunk，加入圖片 URL
		if result.Chunk.IsImageChunk() {
			resultResponse.ImageURL = result.Chunk.GetImageURL()
		}
		
		results = append(results, resultResponse)
	}
	
	return &MultimodalSearchResponse{
		Success:     true,
		Results:     results,
		TotalCount:  searchResponse.TotalCount,
		SearchTime:  searchResponse.SearchTime.String(),
		Query:       searchResponse.Query,
		MatchTypes:  searchResponse.MatchTypes,
		Message:     "Search completed successfully",
	}
}

// extractChunkIDFromPath 從 URL 路徑提取 chunk ID
func (s *SearchHandler) extractChunkIDFromPath(path string) string {
	// 假設路徑格式為 /api/v1/search/similar/{chunk_id}
	parts := strings.Split(path, "/")
	if len(parts) >= 6 {
		return parts[5] // chunk_id 在第 6 個位置
	}
	return ""
}

// writeJSONResponse 寫入 JSON 回應
func (s *SearchHandler) writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	
	if err := json.NewEncoder(w).Encode(data); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"success": false, "error": "Failed to encode JSON response"}`))
	}
}

// writeErrorResponse 寫入錯誤回應
func (s *SearchHandler) writeErrorResponse(w http.ResponseWriter, statusCode int, message string, err error) {
	response := &ErrorResponse{
		Success: false,
		Error:   message,
	}
	
	if err != nil {
		response.Details = err.Error()
	}
	
	s.writeJSONResponse(w, statusCode, response)
}

// SearchByTags 根據標籤搜尋
func (s *SearchHandler) SearchByTags(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With, Accept, Origin")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	
	// Handle preflight OPTIONS request
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	
	var req struct {
		Tags []string `json:"tags"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON request", err)
		return
	}
	
	// TODO: Implement actual tag search logic
	// For now, return mock response
	response := map[string]interface{}{
		"items":       []interface{}{},
		"totalCount":  0,
		"searchTime":  50,
		"cacheHit":    false,
	}
	
	s.writeJSONResponse(w, http.StatusOK, response)
}

// SetCacheEnabled 設定快取啟用狀態
func (s *SearchHandler) SetCacheEnabled(enabled bool) {
	s.cacheEnabled = enabled
}