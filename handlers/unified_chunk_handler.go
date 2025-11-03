package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"semantic-text-processor/models"
	"semantic-text-processor/services"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

// UnifiedChunkHandler handles chunk-related HTTP requests using the unified service
type UnifiedChunkHandler struct {
	unifiedService     services.UnifiedChunkService
	converter          *ModelConverter
	performanceMonitor *PerformanceMonitor
	cacheService       services.CacheService
	logger             *log.Logger
}

// NewUnifiedChunkHandler creates a new unified chunk handler
func NewUnifiedChunkHandler(
	unifiedService services.UnifiedChunkService,
	cacheService services.CacheService,
	logger *log.Logger,
	slowQueryThreshold time.Duration,
	metricsEnabled bool,
) *UnifiedChunkHandler {
	return &UnifiedChunkHandler{
		unifiedService:     unifiedService,
		converter:          NewModelConverter(),
		performanceMonitor: NewPerformanceMonitor(slowQueryThreshold, logger, metricsEnabled),
		cacheService:       cacheService,
		logger:             logger,
	}
}

// GetChunks handles GET /api/v1/chunks
func (h *UnifiedChunkHandler) GetChunks(w http.ResponseWriter, r *http.Request) {
	h.performanceMonitor.MonitoredHTTPOperation("get_chunks", w, func() (int, error) {
		// Parse query parameters
		query := r.URL.Query()
		searchQuery := query.Get("q")

		// Parse pagination
		limit := 50 // default
		if l := query.Get("limit"); l != "" {
			if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
				limit = parsed
			}
		}

		offset := 0
		if o := query.Get("offset"); o != "" {
			if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
				offset = parsed
			}
		}

		// Build filters from query parameters
		filters := make(map[string]interface{})

		if textID := query.Get("text_id"); textID != "" {
			filters["text_id"] = textID
		}

		if isTemplate := query.Get("is_template"); isTemplate != "" {
			filters["is_template"] = isTemplate == "true"
		}

		if isSlot := query.Get("is_slot"); isSlot != "" {
			filters["is_slot"] = isSlot == "true"
		}

		if parentID := query.Get("parent_id"); parentID != "" {
			filters["parent_chunk_id"] = parentID
		}

		// Convert to unified search query
		unifiedQuery := h.converter.ToUnifiedSearchQuery(searchQuery, filters, limit, offset)

		// Execute search
		result, err := h.unifiedService.SearchChunks(r.Context(), unifiedQuery)
		if err != nil {
			writeErrorResponse(w, http.StatusInternalServerError, "failed to search chunks", err.Error())
			return http.StatusInternalServerError, err
		}

		// Convert results to legacy format
		legacyChunks := h.converter.BatchFromUnified(result.Chunks)

		// Create response with metadata
		response := map[string]interface{}{
			"chunks":      legacyChunks,
			"total_count": result.TotalCount,
			"has_more":    result.HasMore,
			"cache_hit":   result.CacheHit,
		}

		writeJSONResponse(w, http.StatusOK, response)
		return http.StatusOK, nil
	})
}

// CreateChunk handles POST /api/v1/chunks
func (h *UnifiedChunkHandler) CreateChunk(w http.ResponseWriter, r *http.Request) {
	h.performanceMonitor.MonitoredHTTPOperation("create_chunk", w, func() (int, error) {
		var req models.CreateChunkRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErrorResponse(w, http.StatusBadRequest, "invalid request body", err.Error())
			return http.StatusBadRequest, err
		}

		// Validate required fields
		if req.Content == "" {
			writeErrorResponse(w, http.StatusBadRequest, "content is required", "")
			return http.StatusBadRequest, nil
		}

		// Convert to unified format
		unifiedChunk := h.converter.ToCreateChunkRequest(&req)

		// Create chunk using unified service
		if err := h.unifiedService.CreateChunk(r.Context(), unifiedChunk); err != nil {
			writeErrorResponse(w, http.StatusInternalServerError, "failed to create chunk", err.Error())
			return http.StatusInternalServerError, err
		}

		// Convert back to legacy format for response
		legacyChunk := h.converter.FromUnifiedChunk(unifiedChunk)

		writeJSONResponse(w, http.StatusCreated, legacyChunk)
		return http.StatusCreated, nil
	})
}

// GetChunkByID handles GET /api/v1/chunks/{id}
func (h *UnifiedChunkHandler) GetChunkByID(w http.ResponseWriter, r *http.Request) {
	h.performanceMonitor.MonitoredHTTPOperation("get_chunk_by_id", w, func() (int, error) {
		vars := mux.Vars(r)
		chunkID := vars["id"]

		if chunkID == "" {
			writeErrorResponse(w, http.StatusBadRequest, "chunk ID is required", "")
			return http.StatusBadRequest, nil
		}

		// Try cache first
		var chunk *models.UnifiedChunkRecord
		var err error
		cacheHit := false

		if h.cacheService != nil {
			cacheKey := "chunk:" + chunkID
			var cached interface{}
			if h.cacheService.Get(r.Context(), cacheKey, &cached) == nil {
				if cachedChunk, ok := cached.(*models.UnifiedChunkRecord); ok {
					chunk = cachedChunk
					cacheHit = true
				}
			}
		}

		if chunk == nil {
			chunk, err = h.unifiedService.GetChunk(r.Context(), chunkID)
			if err != nil {
				writeErrorResponse(w, http.StatusNotFound, "chunk not found", err.Error())
				return http.StatusNotFound, err
			}

			// Cache the result
			if h.cacheService != nil {
				cacheKey := "chunk:" + chunkID
				h.cacheService.Set(r.Context(), cacheKey, chunk, 15*time.Minute)
			}
		}

		// Convert to legacy format
		legacyChunk := h.converter.FromUnifiedChunk(chunk)

		// Add cache hit header
		if cacheHit {
			w.Header().Set("X-Cache", "HIT")
		} else {
			w.Header().Set("X-Cache", "MISS")
		}

		writeJSONResponse(w, http.StatusOK, legacyChunk)
		return http.StatusOK, nil
	})
}

// UpdateChunk handles PUT /api/v1/chunks/{id}
func (h *UnifiedChunkHandler) UpdateChunk(w http.ResponseWriter, r *http.Request) {
	h.performanceMonitor.MonitoredHTTPOperation("update_chunk", w, func() (int, error) {
		vars := mux.Vars(r)
		chunkID := vars["id"]

		if chunkID == "" {
			writeErrorResponse(w, http.StatusBadRequest, "chunk ID is required", "")
			return http.StatusBadRequest, nil
		}

		var req models.UpdateChunkRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErrorResponse(w, http.StatusBadRequest, "invalid request body", err.Error())
			return http.StatusBadRequest, err
		}

		// Get existing chunk
		chunk, err := h.unifiedService.GetChunk(r.Context(), chunkID)
		if err != nil {
			writeErrorResponse(w, http.StatusNotFound, "chunk not found", err.Error())
			return http.StatusNotFound, err
		}

		// Apply updates
		h.converter.ApplyUpdateRequest(chunk, &req)

		// Update in database
		if err := h.unifiedService.UpdateChunk(r.Context(), chunk); err != nil {
			writeErrorResponse(w, http.StatusInternalServerError, "failed to update chunk", err.Error())
			return http.StatusInternalServerError, err
		}

		// Invalidate cache
		if h.cacheService != nil {
			cacheKey := "chunk:" + chunkID
			h.cacheService.Delete(r.Context(), cacheKey)
		}

		// Convert to legacy format for response
		legacyChunk := h.converter.FromUnifiedChunk(chunk)

		writeJSONResponse(w, http.StatusOK, legacyChunk)
		return http.StatusOK, nil
	})
}

// DeleteChunk handles DELETE /api/v1/chunks/{id}
func (h *UnifiedChunkHandler) DeleteChunk(w http.ResponseWriter, r *http.Request) {
	h.performanceMonitor.MonitoredHTTPOperation("delete_chunk", w, func() (int, error) {
		vars := mux.Vars(r)
		chunkID := vars["id"]

		if chunkID == "" {
			writeErrorResponse(w, http.StatusBadRequest, "chunk ID is required", "")
			return http.StatusBadRequest, nil
		}

		if err := h.unifiedService.DeleteChunk(r.Context(), chunkID); err != nil {
			writeErrorResponse(w, http.StatusInternalServerError, "failed to delete chunk", err.Error())
			return http.StatusInternalServerError, err
		}

		// Invalidate cache
		if h.cacheService != nil {
			cacheKey := "chunk:" + chunkID
			h.cacheService.Delete(r.Context(), cacheKey)
		}

		w.WriteHeader(http.StatusNoContent)
		return http.StatusNoContent, nil
	})
}

// GetChunkChildren handles GET /api/v1/chunks/{id}/children
func (h *UnifiedChunkHandler) GetChunkChildren(w http.ResponseWriter, r *http.Request) {
	h.performanceMonitor.MonitoredHTTPOperation("get_chunk_children", w, func() (int, error) {
		vars := mux.Vars(r)
		chunkID := vars["id"]

		if chunkID == "" {
			writeErrorResponse(w, http.StatusBadRequest, "chunk ID is required", "")
			return http.StatusBadRequest, nil
		}

		children, err := h.unifiedService.GetChildren(r.Context(), chunkID)
		if err != nil {
			writeErrorResponse(w, http.StatusInternalServerError, "failed to get chunk children", err.Error())
			return http.StatusInternalServerError, err
		}

		// Convert to legacy format
		legacyChildren := h.converter.BatchFromUnified(children)

		writeJSONResponse(w, http.StatusOK, legacyChildren)
		return http.StatusOK, nil
	})
}

// GetChunkHierarchy handles GET /api/v1/chunks/{id}/hierarchy
func (h *UnifiedChunkHandler) GetChunkHierarchy(w http.ResponseWriter, r *http.Request) {
	h.performanceMonitor.MonitoredHTTPOperation("get_chunk_hierarchy", w, func() (int, error) {
		vars := mux.Vars(r)
		chunkID := vars["id"]

		if chunkID == "" {
			writeErrorResponse(w, http.StatusBadRequest, "chunk ID is required", "")
			return http.StatusBadRequest, nil
		}

		// Parse max depth parameter
		maxDepth := 10 // default
		if d := r.URL.Query().Get("max_depth"); d != "" {
			if parsed, err := strconv.Atoi(d); err == nil && parsed > 0 {
				maxDepth = parsed
			}
		}

		descendants, err := h.unifiedService.GetDescendants(r.Context(), chunkID, maxDepth)
		if err != nil {
			writeErrorResponse(w, http.StatusInternalServerError, "failed to get chunk hierarchy", err.Error())
			return http.StatusInternalServerError, err
		}

		// Convert to legacy format
		legacyDescendants := h.converter.BatchFromUnified(descendants)

		writeJSONResponse(w, http.StatusOK, legacyDescendants)
		return http.StatusOK, nil
	})
}

// MoveChunk handles POST /api/v1/chunks/{id}/move
func (h *UnifiedChunkHandler) MoveChunk(w http.ResponseWriter, r *http.Request) {
	h.performanceMonitor.MonitoredHTTPOperation("move_chunk", w, func() (int, error) {
		vars := mux.Vars(r)
		chunkID := vars["id"]

		if chunkID == "" {
			writeErrorResponse(w, http.StatusBadRequest, "chunk ID is required", "")
			return http.StatusBadRequest, nil
		}

		var req models.MoveChunkRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErrorResponse(w, http.StatusBadRequest, "invalid request body", err.Error())
			return http.StatusBadRequest, err
		}

		// Set the chunk ID from URL
		req.ChunkID = chunkID

		// Convert to new parent ID format
		newParentID := ""
		if req.NewParentID != nil {
			newParentID = *req.NewParentID
		}

		if err := h.unifiedService.MoveChunk(r.Context(), chunkID, newParentID); err != nil {
			writeErrorResponse(w, http.StatusInternalServerError, "failed to move chunk", err.Error())
			return http.StatusInternalServerError, err
		}

		// Invalidate related caches
		if h.cacheService != nil {
			h.cacheService.Delete(r.Context(), "chunk:"+chunkID)
			if req.NewParentID != nil {
				h.cacheService.Delete(r.Context(), "chunk:"+*req.NewParentID)
			}
		}

		w.WriteHeader(http.StatusNoContent)
		return http.StatusNoContent, nil
	})
}

// BatchCreateChunks handles POST /api/v1/chunks/batch
func (h *UnifiedChunkHandler) BatchCreateChunks(w http.ResponseWriter, r *http.Request) {
	h.performanceMonitor.MonitoredHTTPOperation("batch_create_chunks", w, func() (int, error) {
		var req models.BatchCreateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErrorResponse(w, http.StatusBadRequest, "invalid request body", err.Error())
			return http.StatusBadRequest, err
		}

		if len(req.Chunks) == 0 {
			writeErrorResponse(w, http.StatusBadRequest, "no chunks provided", "")
			return http.StatusBadRequest, nil
		}

		// Convert to unified format
		unifiedChunks := req.Chunks

		// Use batch operation with monitoring
		err := h.performanceMonitor.BatchOperation("batch_create", len(unifiedChunks), func() error {
			return h.unifiedService.BatchCreateChunks(r.Context(), unifiedChunks)
		})

		if err != nil {
			writeErrorResponse(w, http.StatusInternalServerError, "failed to create chunks", err.Error())
			return http.StatusInternalServerError, err
		}

		// Convert back to legacy format for response
		legacyChunks := h.converter.BatchFromUnified(unifiedChunks)

		response := map[string]interface{}{
			"created_count": len(legacyChunks),
			"chunks":        legacyChunks,
		}

		writeJSONResponse(w, http.StatusCreated, response)
		return http.StatusCreated, nil
	})
}

// BatchUpdateChunks handles PUT /api/v1/chunks/batch
func (h *UnifiedChunkHandler) BatchUpdateChunks(w http.ResponseWriter, r *http.Request) {
	h.performanceMonitor.MonitoredHTTPOperation("batch_update_chunks", w, func() (int, error) {
		var req models.BatchUpdateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErrorResponse(w, http.StatusBadRequest, "invalid request body", err.Error())
			return http.StatusBadRequest, err
		}

		if len(req.Chunks) == 0 {
			writeErrorResponse(w, http.StatusBadRequest, "no chunks provided", "")
			return http.StatusBadRequest, nil
		}

		// Convert to unified format
		unifiedChunks := req.Chunks

		// Use batch operation with monitoring
		err := h.performanceMonitor.BatchOperation("batch_update", len(unifiedChunks), func() error {
			return h.unifiedService.BatchUpdateChunks(r.Context(), unifiedChunks)
		})

		if err != nil {
			writeErrorResponse(w, http.StatusInternalServerError, "failed to update chunks", err.Error())
			return http.StatusInternalServerError, err
		}

		// Invalidate caches for updated chunks
		if h.cacheService != nil {
			for _, chunk := range unifiedChunks {
				h.cacheService.Delete(r.Context(), "chunk:"+chunk.ChunkID)
			}
		}

		response := map[string]interface{}{
			"updated_count": len(unifiedChunks),
		}

		writeJSONResponse(w, http.StatusOK, response)
		return http.StatusOK, nil
	})
}