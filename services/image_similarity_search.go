package services

import (
	"context"
	"fmt"
	"io"
	"math"
	"sort"
	"time"

	"semantic-text-processor/models"
)

// ImageSimilaritySearch 以圖搜圖服務
type ImageSimilaritySearch struct {
	embeddingService   ImageEmbeddingService
	chunkService       UnifiedChunkService
	mediaProcessor     MediaProcessor
	cacheService       CacheService
	similarityThreshold float64
	maxResults         int
	cacheEnabled       bool
	cacheTTL           time.Duration
}

// NewImageSimilaritySearch 建立新的以圖搜圖服務
func NewImageSimilaritySearch(
	embeddingService ImageEmbeddingService,
	chunkService UnifiedChunkService,
	mediaProcessor MediaProcessor,
	cacheService CacheService,
) *ImageSimilaritySearch {
	return &ImageSimilaritySearch{
		embeddingService:    embeddingService,
		chunkService:        chunkService,
		mediaProcessor:      mediaProcessor,
		cacheService:        cacheService,
		similarityThreshold: 0.7,
		maxResults:          50,
		cacheEnabled:        true,
		cacheTTL:            30 * time.Minute,
	}
}

// 使用 cache.go 中定義的 CacheService interface

// SearchByImageURL 使用圖片 URL 搜尋相似圖片
func (i *ImageSimilaritySearch) SearchByImageURL(ctx context.Context, imageURL string, options *ImageSearchOptions) (*ImageSearchResponse, error) {
	startTime := time.Now()
	
	// 檢查快取
	cacheKey := fmt.Sprintf("image_search:url:%s", imageURL)
	if i.cacheEnabled {
		var cachedResponse ImageSearchResponse
		if err := i.cacheService.Get(ctx, cacheKey, &cachedResponse); err == nil {
			cachedResponse.SearchTime = time.Since(startTime)
			cachedResponse.CacheHit = true
			return &cachedResponse, nil
		}
	}
	
	// 生成查詢向量
	queryVector, err := i.embeddingService.GenerateEmbedding(ctx, imageURL)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding for image URL: %w", err)
	}
	
	// 執行相似度搜尋
	response, err := i.searchByVector(ctx, queryVector, options)
	if err != nil {
		return nil, err
	}
	
	response.SearchTime = time.Since(startTime)
	response.QueryType = "url"
	response.QuerySource = imageURL
	
	// 快取結果
	if i.cacheEnabled {
		i.cacheService.Set(ctx, cacheKey, response, i.cacheTTL)
	}
	
	return response, nil
}

// SearchByImageFile 使用上傳的圖片檔案搜尋相似圖片
func (i *ImageSimilaritySearch) SearchByImageFile(ctx context.Context, imageFile io.Reader, filename string, options *ImageSearchOptions) (*ImageSearchResponse, error) {
	startTime := time.Now()
	
	// 暫時處理圖片檔案
	tempResult, err := i.processTemporaryImage(ctx, imageFile, filename)
	if err != nil {
		return nil, fmt.Errorf("failed to process temporary image: %w", err)
	}
	defer i.cleanupTemporaryImage(ctx, tempResult.URL)
	
	// 生成查詢向量
	queryVector, err := i.embeddingService.GenerateEmbedding(ctx, tempResult.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding for uploaded image: %w", err)
	}
	
	// 執行相似度搜尋
	response, err := i.searchByVector(ctx, queryVector, options)
	if err != nil {
		return nil, err
	}
	
	response.SearchTime = time.Since(startTime)
	response.QueryType = "upload"
	response.QuerySource = filename
	
	return response, nil
}

// SearchByChunkID 使用現有 chunk 的圖片搜尋相似圖片
func (i *ImageSimilaritySearch) SearchByChunkID(ctx context.Context, chunkID string, options *ImageSearchOptions) (*ImageSearchResponse, error) {
	startTime := time.Now()
	
	// 檢查快取
	cacheKey := fmt.Sprintf("image_search:chunk:%s", chunkID)
	if i.cacheEnabled {
		var cachedResponse ImageSearchResponse
		if err := i.cacheService.Get(ctx, cacheKey, &cachedResponse); err == nil {
			cachedResponse.SearchTime = time.Since(startTime)
			cachedResponse.CacheHit = true
			return &cachedResponse, nil
		}
	}
	
	// 取得 chunk
	chunk, err := i.chunkService.GetChunk(ctx, chunkID)
	if err != nil {
		return nil, fmt.Errorf("failed to get chunk: %w", err)
	}
	
	// 檢查是否為圖片 chunk
	if !chunk.IsImageChunk() {
		return nil, fmt.Errorf("chunk %s is not an image chunk", chunkID)
	}
	
	// 使用 chunk 的向量或重新生成
	var queryVector []float64
	if chunk.HasVector() && chunk.GetVectorType() == models.VectorTypeImage {
		queryVector = chunk.Vector
	} else {
		// 重新生成向量
		imageURL := chunk.GetImageURL()
		if imageURL == "" {
			return nil, fmt.Errorf("no image URL found for chunk %s", chunkID)
		}
		
		queryVector, err = i.embeddingService.GenerateEmbedding(ctx, imageURL)
		if err != nil {
			return nil, fmt.Errorf("failed to generate embedding for chunk image: %w", err)
		}
	}
	
	// 執行相似度搜尋
	response, err := i.searchByVector(ctx, queryVector, options)
	if err != nil {
		return nil, err
	}
	
	// 從結果中排除查詢的 chunk 本身
	response.Results = i.excludeChunk(response.Results, chunkID)
	response.TotalCount = len(response.Results)
	
	response.SearchTime = time.Since(startTime)
	response.QueryType = "chunk"
	response.QuerySource = chunkID
	
	// 快取結果
	if i.cacheEnabled {
		i.cacheService.Set(ctx, cacheKey, response, i.cacheTTL)
	}
	
	return response, nil
}

// FindDuplicateImages 尋找重複圖片
func (i *ImageSimilaritySearch) FindDuplicateImages(ctx context.Context, options *DuplicateSearchOptions) (*DuplicateImageResponse, error) {
	startTime := time.Now()
	
	if options == nil {
		options = DefaultDuplicateSearchOptions()
	}
	
	// 取得所有圖片 chunks
	imageChunks, err := i.getAllImageChunks(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get image chunks: %w", err)
	}
	
	var duplicateGroups []DuplicateGroup
	processed := make(map[string]bool)

	// 比較每對圖片
	for idx, chunk1 := range imageChunks {
		if processed[chunk1.ChunkID] {
			continue
		}

		var duplicates []DuplicateImage
		duplicates = append(duplicates, DuplicateImage{
			ChunkID:    chunk1.ChunkID,
			ImageURL:   chunk1.GetImageURL(),
			Similarity: 1.0,
		})

		for j := idx + 1; j < len(imageChunks); j++ {
			chunk2 := imageChunks[j]
			if processed[chunk2.ChunkID] {
				continue
			}

			similarity := i.calculateImageSimilarity(&chunk1, &chunk2)
			if similarity >= options.SimilarityThreshold {
				duplicates = append(duplicates, DuplicateImage{
					ChunkID:    chunk2.ChunkID,
					ImageURL:   chunk2.GetImageURL(),
					Similarity: similarity,
				})
				processed[chunk2.ChunkID] = true
			}
		}
		
		if len(duplicates) > 1 {
			// 按相似度排序
			sort.Slice(duplicates, func(i, j int) bool {
				return duplicates[i].Similarity > duplicates[j].Similarity
			})
			
			duplicateGroups = append(duplicateGroups, DuplicateGroup{
				GroupID:    fmt.Sprintf("group_%d", len(duplicateGroups)+1),
				Images:     duplicates,
				GroupSize:  len(duplicates),
				MaxSimilarity: duplicates[1].Similarity, // 第二高的相似度
			})
		}
		
		processed[chunk1.ChunkID] = true
	}
	
	// 按群組大小排序
	sort.Slice(duplicateGroups, func(i, j int) bool {
		return duplicateGroups[i].GroupSize > duplicateGroups[j].GroupSize
	})
	
	return &DuplicateImageResponse{
		Groups:      duplicateGroups,
		TotalGroups: len(duplicateGroups),
		SearchTime:  time.Since(startTime),
	}, nil
}

// GetSimilarImages 取得相似圖片推薦
func (i *ImageSimilaritySearch) GetSimilarImages(ctx context.Context, chunkID string, count int) (*SimilarImageResponse, error) {
	options := &ImageSearchOptions{
		Limit:               count,
		SimilarityThreshold: 0.6,
		IncludeMetadata:     true,
		SortBy:              "similarity",
		SortOrder:           "desc",
	}
	
	searchResponse, err := i.SearchByChunkID(ctx, chunkID, options)
	if err != nil {
		return nil, err
	}
	
	var recommendations []ImageRecommendation
	for _, result := range searchResponse.Results {
		recommendations = append(recommendations, ImageRecommendation{
			ChunkID:     result.ChunkID,
			ImageURL:    result.ImageURL,
			Similarity:  result.Similarity,
			Description: result.Description,
			Tags:        result.Tags,
			Reason:      fmt.Sprintf("視覺相似度: %.1f%%", result.Similarity*100),
		})
	}
	
	return &SimilarImageResponse{
		Recommendations: recommendations,
		TotalCount:      len(recommendations),
		SearchTime:      searchResponse.SearchTime,
	}, nil
}

// 資料結構

// ImageSearchOptions 圖片搜尋選項
type ImageSearchOptions struct {
	Limit               int     `json:"limit"`
	SimilarityThreshold float64 `json:"similarity_threshold"`
	IncludeMetadata     bool    `json:"include_metadata"`
	ExcludeChunkIDs     []string `json:"exclude_chunk_ids"`
	FilterTags          []string `json:"filter_tags"`
	SortBy              string  `json:"sort_by"` // "similarity", "date", "size"
	SortOrder           string  `json:"sort_order"` // "asc", "desc"
}

// DefaultImageSearchOptions 預設圖片搜尋選項
func DefaultImageSearchOptions() *ImageSearchOptions {
	return &ImageSearchOptions{
		Limit:               20,
		SimilarityThreshold: 0.7,
		IncludeMetadata:     true,
		SortBy:              "similarity",
		SortOrder:           "desc",
	}
}

// ImageSearchResponse 圖片搜尋回應
type ImageSearchResponse struct {
	Results     []ImageSearchResult `json:"results"`
	TotalCount  int                 `json:"total_count"`
	SearchTime  time.Duration       `json:"search_time"`
	QueryType   string              `json:"query_type"`
	QuerySource string              `json:"query_source"`
	CacheHit    bool                `json:"cache_hit"`
}

// ImageSearchResult 圖片搜尋結果
type ImageSearchResult struct {
	ChunkID     string            `json:"chunk_id"`
	ImageURL    string            `json:"image_url"`
	Similarity  float64           `json:"similarity"`
	Description string            `json:"description"`
	Tags        []string          `json:"tags"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
}

// DuplicateSearchOptions 重複圖片搜尋選項
type DuplicateSearchOptions struct {
	SimilarityThreshold float64 `json:"similarity_threshold"`
	MinGroupSize        int     `json:"min_group_size"`
	IncludeMetadata     bool    `json:"include_metadata"`
}

// DefaultDuplicateSearchOptions 預設重複圖片搜尋選項
func DefaultDuplicateSearchOptions() *DuplicateSearchOptions {
	return &DuplicateSearchOptions{
		SimilarityThreshold: 0.95,
		MinGroupSize:        2,
		IncludeMetadata:     true,
	}
}

// DuplicateImageResponse 重複圖片回應
type DuplicateImageResponse struct {
	Groups      []DuplicateGroup `json:"groups"`
	TotalGroups int              `json:"total_groups"`
	SearchTime  time.Duration    `json:"search_time"`
}

// DuplicateGroup 重複圖片群組
type DuplicateGroup struct {
	GroupID       string           `json:"group_id"`
	Images        []DuplicateImage `json:"images"`
	GroupSize     int              `json:"group_size"`
	MaxSimilarity float64          `json:"max_similarity"`
}

// DuplicateImage 重複圖片項目
type DuplicateImage struct {
	ChunkID    string  `json:"chunk_id"`
	ImageURL   string  `json:"image_url"`
	Similarity float64 `json:"similarity"`
}

// SimilarImageResponse 相似圖片回應
type SimilarImageResponse struct {
	Recommendations []ImageRecommendation `json:"recommendations"`
	TotalCount      int                   `json:"total_count"`
	SearchTime      time.Duration         `json:"search_time"`
}

// ImageRecommendation 圖片推薦
type ImageRecommendation struct {
	ChunkID     string   `json:"chunk_id"`
	ImageURL    string   `json:"image_url"`
	Similarity  float64  `json:"similarity"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
	Reason      string   `json:"reason"`
}

// TemporaryImageResult 暫時圖片處理結果
type TemporaryImageResult struct {
	URL      string
	FilePath string
}

// 私有方法

// searchByVector 使用向量執行搜尋
func (i *ImageSimilaritySearch) searchByVector(ctx context.Context, queryVector []float64, options *ImageSearchOptions) (*ImageSearchResponse, error) {
	if options == nil {
		options = DefaultImageSearchOptions()
	}
	
	// 取得所有圖片 chunks
	imageChunks, err := i.getAllImageChunks(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get image chunks: %w", err)
	}
	
	var results []ImageSearchResult
	
	// 計算相似度
	for _, chunk := range imageChunks {
		// 跳過排除的 chunks
		if i.isExcluded(chunk.ChunkID, options.ExcludeChunkIDs) {
			continue
		}
		
		// 檢查標籤過濾
		if !i.matchesTags(chunk.Tags, options.FilterTags) {
			continue
		}
		
		// 計算相似度
		var similarity float64
		if chunk.HasVector() && chunk.GetVectorType() == models.VectorTypeImage {
			similarity = i.calculateCosineSimilarity(queryVector, chunk.Vector)
		} else {
			// 如果沒有向量，跳過
			continue
		}
		
		// 檢查相似度閾值
		if similarity < options.SimilarityThreshold {
			continue
		}
		
		// 建立結果
		result := ImageSearchResult{
			ChunkID:     chunk.ChunkID,
			ImageURL:    chunk.GetImageURL(),
			Similarity:  similarity,
			Description: chunk.Contents,
			Tags:        chunk.Tags,
			CreatedAt:   chunk.CreatedTime,
		}
		
		// 包含 metadata（如果需要）
		if options.IncludeMetadata {
			result.Metadata = chunk.Metadata
		}
		
		results = append(results, result)
	}
	
	// 排序結果
	i.sortResults(results, options.SortBy, options.SortOrder)
	
	// 限制結果數量
	if len(results) > options.Limit {
		results = results[:options.Limit]
	}
	
	return &ImageSearchResponse{
		Results:    results,
		TotalCount: len(results),
	}, nil
}

// processTemporaryImage 處理暫時圖片
func (i *ImageSimilaritySearch) processTemporaryImage(ctx context.Context, imageFile io.Reader, filename string) (*TemporaryImageResult, error) {
	// 這裡應該將圖片暫時儲存到臨時位置
	// 簡化實作，實際應該使用臨時檔案系統

	// 讀取圖片內容
	fileData, err := io.ReadAll(imageFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read image file: %w", err)
	}

	// 使用 MediaProcessor 處理圖片
	req := &models.ProcessImageRequest{
		File:             fileData,
		OriginalFilename: filename,
		AutoAnalyze:      false,
		AutoEmbed:        false,
		StorageType:      models.StorageTypeLocal, // 使用本地暫存
	}

	result, err := i.mediaProcessor.ProcessImage(ctx, req)
	if err != nil {
		return nil, err
	}

	return &TemporaryImageResult{
		URL:      result.URL,
		FilePath: result.StorageID,
	}, nil
}

// cleanupTemporaryImage 清理暫時圖片
func (i *ImageSimilaritySearch) cleanupTemporaryImage(ctx context.Context, imageURL string) {
	// 清理暫時檔案
	// 實際實作中應該刪除暫時檔案
}

// getAllImageChunks 取得所有圖片 chunks
func (i *ImageSimilaritySearch) getAllImageChunks(ctx context.Context) ([]models.UnifiedChunkRecord, error) {
	// 使用搜尋查詢取得所有圖片 chunks
	query := &models.SearchQuery{
		Metadata: map[string]interface{}{
			"media_type": "image",
		},
		Limit: 10000, // 大數量限制
	}
	
	result, err := i.chunkService.SearchChunks(ctx, query)
	if err != nil {
		return nil, err
	}
	
	return result.Chunks, nil
}

// calculateImageSimilarity 計算兩個圖片 chunk 的相似度
func (i *ImageSimilaritySearch) calculateImageSimilarity(chunk1, chunk2 *models.UnifiedChunkRecord) float64 {
	if !chunk1.HasVector() || !chunk2.HasVector() {
		return 0.0
	}
	
	if chunk1.GetVectorType() != models.VectorTypeImage || chunk2.GetVectorType() != models.VectorTypeImage {
		return 0.0
	}
	
	return i.calculateCosineSimilarity(chunk1.Vector, chunk2.Vector)
}

// calculateCosineSimilarity 計算餘弦相似度
func (i *ImageSimilaritySearch) calculateCosineSimilarity(vec1, vec2 []float64) float64 {
	if len(vec1) != len(vec2) {
		return 0.0
	}
	
	var dotProduct, norm1, norm2 float64
	
	for j := 0; j < len(vec1); j++ {
		dotProduct += vec1[j] * vec2[j]
		norm1 += vec1[j] * vec1[j]
		norm2 += vec2[j] * vec2[j]
	}
	
	if norm1 == 0 || norm2 == 0 {
		return 0.0
	}

	return dotProduct / (math.Sqrt(norm1) * math.Sqrt(norm2))
}

// 使用 math.Sqrt 代替自定義 sqrt 函數

// excludeChunk 從結果中排除指定的 chunk
func (i *ImageSimilaritySearch) excludeChunk(results []ImageSearchResult, chunkID string) []ImageSearchResult {
	var filtered []ImageSearchResult
	for _, result := range results {
		if result.ChunkID != chunkID {
			filtered = append(filtered, result)
		}
	}
	return filtered
}

// isExcluded 檢查是否被排除
func (i *ImageSimilaritySearch) isExcluded(chunkID string, excludeList []string) bool {
	for _, excludeID := range excludeList {
		if chunkID == excludeID {
			return true
		}
	}
	return false
}

// matchesTags 檢查標籤是否匹配
func (i *ImageSimilaritySearch) matchesTags(chunkTags, filterTags []string) bool {
	if len(filterTags) == 0 {
		return true // 沒有過濾條件
	}
	
	tagSet := make(map[string]bool)
	for _, tag := range chunkTags {
		tagSet[tag] = true
	}
	
	for _, filterTag := range filterTags {
		if tagSet[filterTag] {
			return true // 至少匹配一個標籤
		}
	}
	
	return false
}

// sortResults 排序結果
func (i *ImageSimilaritySearch) sortResults(results []ImageSearchResult, sortBy, sortOrder string) {
	switch sortBy {
	case "similarity":
		if sortOrder == "desc" {
			sort.Slice(results, func(i, j int) bool {
				return results[i].Similarity > results[j].Similarity
			})
		} else {
			sort.Slice(results, func(i, j int) bool {
				return results[i].Similarity < results[j].Similarity
			})
		}
	case "date":
		if sortOrder == "desc" {
			sort.Slice(results, func(i, j int) bool {
				return results[i].CreatedAt.After(results[j].CreatedAt)
			})
		} else {
			sort.Slice(results, func(i, j int) bool {
				return results[i].CreatedAt.Before(results[j].CreatedAt)
			})
		}
	default:
		// 預設按相似度降序排序
		sort.Slice(results, func(i, j int) bool {
			return results[i].Similarity > results[j].Similarity
		})
	}
}

// SetSimilarityThreshold 設定相似度閾值
func (i *ImageSimilaritySearch) SetSimilarityThreshold(threshold float64) {
	if threshold >= 0.0 && threshold <= 1.0 {
		i.similarityThreshold = threshold
	}
}

// SetMaxResults 設定最大結果數量
func (i *ImageSimilaritySearch) SetMaxResults(maxResults int) {
	if maxResults > 0 {
		i.maxResults = maxResults
	}
}

// EnableCache 啟用快取
func (i *ImageSimilaritySearch) EnableCache(enabled bool) {
	i.cacheEnabled = enabled
}

// SetCacheTTL 設定快取 TTL
func (i *ImageSimilaritySearch) SetCacheTTL(ttl time.Duration) {
	if ttl > 0 {
		i.cacheTTL = ttl
	}
}