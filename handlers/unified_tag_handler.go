package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"semantic-text-processor/models"
	"semantic-text-processor/services"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

// UnifiedTagHandler handles tag-related HTTP requests using the unified service
type UnifiedTagHandler struct {
	unifiedService     services.UnifiedChunkService
	converter          *ModelConverter
	performanceMonitor *PerformanceMonitor
	cacheService       services.CacheService
	logger             *log.Logger
}

// NewUnifiedTagHandler creates a new unified tag handler
func NewUnifiedTagHandler(
	unifiedService services.UnifiedChunkService,
	cacheService services.CacheService,
	logger *log.Logger,
	slowQueryThreshold time.Duration,
	metricsEnabled bool,
) *UnifiedTagHandler {
	return &UnifiedTagHandler{
		unifiedService:     unifiedService,
		converter:          NewModelConverter(),
		performanceMonitor: NewPerformanceMonitor(slowQueryThreshold, logger, metricsEnabled),
		cacheService:       cacheService,
		logger:             logger,
	}
}

// BatchAddTagsRequest represents a request to add multiple tags to multiple chunks
type BatchAddTagsRequest struct {
	Operations []TagOperation `json:"operations"`
}

// TagOperation represents a single tag operation
type TagOperation struct {
	ChunkID    string   `json:"chunk_id"`
	TagContent string   `json:"tag_content,omitempty"`
	TagIDs     []string `json:"tag_ids,omitempty"`
	Operation  string   `json:"operation"` // "add" or "remove"
}

// AddTag handles POST /api/v1/chunks/{id}/tags
func (h *UnifiedTagHandler) AddTag(w http.ResponseWriter, r *http.Request) {
	h.performanceMonitor.MonitoredHTTPOperation("add_tag", w, func() (int, error) {
		vars := mux.Vars(r)
		chunkID := vars["id"]

		if chunkID == "" {
			writeErrorResponse(w, http.StatusBadRequest, "chunk ID is required", "")
			return http.StatusBadRequest, nil
		}

		var req models.AddTagRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErrorResponse(w, http.StatusBadRequest, "invalid request body", err.Error())
			return http.StatusBadRequest, err
		}

		// Set chunk ID from URL
		req.ChunkID = chunkID

		// Validate request
		if req.TagContent == "" {
			writeErrorResponse(w, http.StatusBadRequest, "tag content is required", "")
			return http.StatusBadRequest, nil
		}

		// First, find or create the tag chunk
		tagChunkID, err := h.findOrCreateTagChunk(r.Context(), req.TagContent)
		if err != nil {
			writeErrorResponse(w, http.StatusInternalServerError, "failed to find or create tag", err.Error())
			return http.StatusInternalServerError, err
		}

		// Add tag relationship
		if err := h.unifiedService.AddTags(r.Context(), chunkID, []string{tagChunkID}); err != nil {
			writeErrorResponse(w, http.StatusInternalServerError, "failed to add tag", err.Error())
			return http.StatusInternalServerError, err
		}

		// Invalidate caches
		if h.cacheService != nil {
			h.cacheService.Delete(r.Context(), "chunk_tags:"+chunkID)
			h.cacheService.Delete(r.Context(), "tag_chunks:"+tagChunkID)
		}

		w.WriteHeader(http.StatusCreated)
		return http.StatusCreated, nil
	})
}

// RemoveTag handles DELETE /api/v1/chunks/{id}/tags/{tagId}
func (h *UnifiedTagHandler) RemoveTag(w http.ResponseWriter, r *http.Request) {
	h.performanceMonitor.MonitoredHTTPOperation("remove_tag", w, func() (int, error) {
		vars := mux.Vars(r)
		chunkID := vars["id"]
		tagID := vars["tagId"]

		if chunkID == "" {
			writeErrorResponse(w, http.StatusBadRequest, "chunk ID is required", "")
			return http.StatusBadRequest, nil
		}

		if tagID == "" {
			writeErrorResponse(w, http.StatusBadRequest, "tag ID is required", "")
			return http.StatusBadRequest, nil
		}

		// Remove tag relationship
		if err := h.unifiedService.RemoveTags(r.Context(), chunkID, []string{tagID}); err != nil {
			writeErrorResponse(w, http.StatusInternalServerError, "failed to remove tag", err.Error())
			return http.StatusInternalServerError, err
		}

		// Invalidate caches
		if h.cacheService != nil {
			h.cacheService.Delete(r.Context(), "chunk_tags:"+chunkID)
			h.cacheService.Delete(r.Context(), "tag_chunks:"+tagID)
		}

		w.WriteHeader(http.StatusNoContent)
		return http.StatusNoContent, nil
	})
}

// GetChunkTags handles GET /api/v1/chunks/{id}/tags
func (h *UnifiedTagHandler) GetChunkTags(w http.ResponseWriter, r *http.Request) {
	h.performanceMonitor.MonitoredHTTPOperation("get_chunk_tags", w, func() (int, error) {
		vars := mux.Vars(r)
		chunkID := vars["id"]

		if chunkID == "" {
			writeErrorResponse(w, http.StatusBadRequest, "chunk ID is required", "")
			return http.StatusBadRequest, nil
		}

		// Try cache first
		var tags []models.UnifiedChunkRecord
		var err error
		cacheHit := false

		if h.cacheService != nil {
			cacheKey := "chunk_tags:" + chunkID
			var cached interface{}
		if h.cacheService.Get(r.Context(), cacheKey, &cached) == nil {
				if cachedTags, ok := cached.([]models.UnifiedChunkRecord); ok {
					tags = cachedTags
					cacheHit = true
				}
			}
		}

		if tags == nil {
			tags, err = h.unifiedService.GetChunkTags(r.Context(), chunkID)
			if err != nil {
				writeErrorResponse(w, http.StatusInternalServerError, "failed to get chunk tags", err.Error())
				return http.StatusInternalServerError, err
			}

			// Cache the result
			if h.cacheService != nil {
				cacheKey := "chunk_tags:" + chunkID
				h.cacheService.Set(r.Context(), cacheKey, tags, 10*time.Minute)
			}
		}

		// Convert to legacy format
		legacyTags := h.converter.BatchFromUnified(tags)

		// Add cache hit header
		if cacheHit {
			w.Header().Set("X-Cache", "HIT")
		} else {
			w.Header().Set("X-Cache", "MISS")
		}

		writeJSONResponse(w, http.StatusOK, legacyTags)
		return http.StatusOK, nil
	})
}

// GetChunksByTag handles GET /api/v1/tags/{content}/chunks
func (h *UnifiedTagHandler) GetChunksByTag(w http.ResponseWriter, r *http.Request) {
	h.performanceMonitor.MonitoredHTTPOperation("get_chunks_by_tag", w, func() (int, error) {
		vars := mux.Vars(r)
		tagContent := vars["content"]

		if tagContent == "" {
			writeErrorResponse(w, http.StatusBadRequest, "tag content is required", "")
			return http.StatusBadRequest, nil
		}

		// Find the tag chunk by content
		tagChunkID, err := h.findTagChunkByContent(r.Context(), tagContent)
		if err != nil {
			writeErrorResponse(w, http.StatusNotFound, "tag not found", err.Error())
			return http.StatusNotFound, err
		}

		// Try cache first
		var chunks []models.UnifiedChunkRecord
		cacheHit := false

		if h.cacheService != nil {
			cacheKey := "tag_chunks:" + tagChunkID
			var cached interface{}
		if h.cacheService.Get(r.Context(), cacheKey, &cached) == nil {
				if cachedChunks, ok := cached.([]models.UnifiedChunkRecord); ok {
					chunks = cachedChunks
					cacheHit = true
				}
			}
		}

		if chunks == nil {
			chunks, err = h.unifiedService.GetChunksByTag(r.Context(), tagChunkID)
			if err != nil {
				writeErrorResponse(w, http.StatusInternalServerError, "failed to get chunks by tag", err.Error())
				return http.StatusInternalServerError, err
			}

			// Cache the result
			if h.cacheService != nil {
				cacheKey := "tag_chunks:" + tagChunkID
				h.cacheService.Set(r.Context(), cacheKey, chunks, 10*time.Minute)
			}
		}

		// Convert to legacy format
		legacyChunks := h.converter.BatchFromUnified(chunks)

		// Add cache hit header
		if cacheHit {
			w.Header().Set("X-Cache", "HIT")
		} else {
			w.Header().Set("X-Cache", "MISS")
		}

		writeJSONResponse(w, http.StatusOK, legacyChunks)
		return http.StatusOK, nil
	})
}

// GetChunksByTags handles POST /api/v1/tags/search
func (h *UnifiedTagHandler) GetChunksByTags(w http.ResponseWriter, r *http.Request) {
	h.performanceMonitor.MonitoredHTTPOperation("get_chunks_by_tags", w, func() (int, error) {
		var req struct {
			TagContents []string `json:"tag_contents"`
			Logic       string   `json:"logic"` // "AND" or "OR"
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErrorResponse(w, http.StatusBadRequest, "invalid request body", err.Error())
			return http.StatusBadRequest, err
		}

		if len(req.TagContents) == 0 {
			writeErrorResponse(w, http.StatusBadRequest, "tag contents are required", "")
			return http.StatusBadRequest, nil
		}

		// Default to AND logic
		logic := "AND"
		if req.Logic != "" {
			logic = strings.ToUpper(req.Logic)
		}

		// Find tag chunk IDs for all tag contents
		tagChunkIDs := make([]string, 0, len(req.TagContents))
		for _, tagContent := range req.TagContents {
			tagChunkID, err := h.findTagChunkByContent(r.Context(), tagContent)
			if err != nil {
				// If any tag is not found and using AND logic, return empty result
				if logic == "AND" {
					writeJSONResponse(w, http.StatusOK, []models.ChunkRecord{})
					return http.StatusOK, nil
				}
				// For OR logic, skip missing tags
				continue
			}
			tagChunkIDs = append(tagChunkIDs, tagChunkID)
		}

		if len(tagChunkIDs) == 0 {
			writeJSONResponse(w, http.StatusOK, []models.ChunkRecord{})
			return http.StatusOK, nil
		}

		// Get chunks by tags
		chunks, err := h.unifiedService.GetChunksByTags(r.Context(), tagChunkIDs, logic)
		if err != nil {
			writeErrorResponse(w, http.StatusInternalServerError, "failed to get chunks by tags", err.Error())
			return http.StatusInternalServerError, err
		}

		// Convert to legacy format
		legacyChunks := h.converter.BatchFromUnified(chunks)

		writeJSONResponse(w, http.StatusOK, legacyChunks)
		return http.StatusOK, nil
	})
}

// BatchTagOperations handles POST /api/v1/chunks/tags/batch
func (h *UnifiedTagHandler) BatchTagOperations(w http.ResponseWriter, r *http.Request) {
	h.performanceMonitor.MonitoredHTTPOperation("batch_tag_operations", w, func() (int, error) {
		var req BatchAddTagsRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErrorResponse(w, http.StatusBadRequest, "invalid request body", err.Error())
			return http.StatusBadRequest, err
		}

		if len(req.Operations) == 0 {
			writeErrorResponse(w, http.StatusBadRequest, "no operations provided", "")
			return http.StatusBadRequest, nil
		}

		// Process operations with monitoring
		err := h.performanceMonitor.BatchOperation("batch_tag_operations", len(req.Operations), func() error {
			return h.processBatchTagOperations(r.Context(), req.Operations)
		})

		if err != nil {
			writeErrorResponse(w, http.StatusInternalServerError, "failed to process batch tag operations", err.Error())
			return http.StatusInternalServerError, err
		}

		response := map[string]interface{}{
			"processed_count": len(req.Operations),
			"message":         "batch tag operations completed successfully",
		}

		writeJSONResponse(w, http.StatusOK, response)
		return http.StatusOK, nil
	})
}

// Helper function to find or create a tag chunk
func (h *UnifiedTagHandler) findOrCreateTagChunk(ctx context.Context, tagContent string) (string, error) {
	// First try to find existing tag
	query := &models.SearchQuery{
		Content: tagContent,
		IsTag:   &[]bool{true}[0],
		Limit:   1,
	}

	result, err := h.unifiedService.SearchChunks(ctx, query)
	if err != nil {
		return "", err
	}

	if len(result.Chunks) > 0 {
		return result.Chunks[0].ChunkID, nil
	}

	// Create new tag chunk
	tagChunk := &models.UnifiedChunkRecord{
		Contents:    tagContent,
		IsTag:       true,
		IsPage:      false,
		IsTemplate:  false,
		IsSlot:      false,
		Tags:        []string{},
		Metadata:    map[string]interface{}{},
		CreatedTime: time.Now(),
		LastUpdated: time.Now(),
	}

	if err := h.unifiedService.CreateChunk(ctx, tagChunk); err != nil {
		return "", err
	}

	return tagChunk.ChunkID, nil
}

// Helper function to find tag chunk by content
func (h *UnifiedTagHandler) findTagChunkByContent(ctx context.Context, tagContent string) (string, error) {
	query := &models.SearchQuery{
		Content: tagContent,
		IsTag:   &[]bool{true}[0],
		Limit:   1,
	}

	result, err := h.unifiedService.SearchChunks(ctx, query)
	if err != nil {
		return "", err
	}

	if len(result.Chunks) == 0 {
		return "", models.ErrNotFound
	}

	return result.Chunks[0].ChunkID, nil
}

// Helper function to process batch tag operations
func (h *UnifiedTagHandler) processBatchTagOperations(ctx context.Context, operations []TagOperation) error {
	for _, op := range operations {
		switch strings.ToLower(op.Operation) {
		case "add":
			if op.TagContent != "" {
				tagChunkID, err := h.findOrCreateTagChunk(ctx, op.TagContent)
				if err != nil {
					return err
				}
				if err := h.unifiedService.AddTags(ctx, op.ChunkID, []string{tagChunkID}); err != nil {
					return err
				}
			} else if len(op.TagIDs) > 0 {
				if err := h.unifiedService.AddTags(ctx, op.ChunkID, op.TagIDs); err != nil {
					return err
				}
			}

		case "remove":
			if len(op.TagIDs) > 0 {
				if err := h.unifiedService.RemoveTags(ctx, op.ChunkID, op.TagIDs); err != nil {
					return err
				}
			} else if op.TagContent != "" {
				// Find tag by content and remove
				tagChunkID, err := h.findTagChunkByContent(ctx, op.TagContent)
				if err == nil { // Don't fail if tag doesn't exist
					if err := h.unifiedService.RemoveTags(ctx, op.ChunkID, []string{tagChunkID}); err != nil {
						return err
					}
				}
			}

		default:
			return models.ErrInvalidOperation
		}

		// Invalidate related caches
		if h.cacheService != nil {
			h.cacheService.Delete(ctx, "chunk_tags:"+op.ChunkID)
		}
	}

	return nil
}