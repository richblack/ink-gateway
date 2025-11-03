package handlers

import (
	"encoding/json"
	"net/http"
	"semantic-text-processor/models"
	"semantic-text-processor/services"
	"strconv"

	"github.com/gorilla/mux"
)

// ChunkHandler handles chunk-related HTTP requests
type ChunkHandler struct {
	supabaseClient services.SupabaseClient
}

// NewChunkHandler creates a new chunk handler
func NewChunkHandler(supabaseClient services.SupabaseClient) *ChunkHandler {
	return &ChunkHandler{
		supabaseClient: supabaseClient,
	}
}

// GetChunks handles GET /api/v1/chunks
func (h *ChunkHandler) GetChunks(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	query := r.URL.Query()
	
	// Build filters
	filters := make(map[string]interface{})
	
	if textID := query.Get("text_id"); textID != "" {
		filters["text_id"] = textID
	}
	
	if isTemplate := query.Get("is_template"); isTemplate != "" {
		if isTemplate == "true" {
			filters["is_template"] = true
		} else if isTemplate == "false" {
			filters["is_template"] = false
		}
	}
	
	if isSlot := query.Get("is_slot"); isSlot != "" {
		if isSlot == "true" {
			filters["is_slot"] = true
		} else if isSlot == "false" {
			filters["is_slot"] = false
		}
	}
	
	if indentLevel := query.Get("indent_level"); indentLevel != "" {
		if level, err := strconv.Atoi(indentLevel); err == nil {
			filters["indent_level"] = level
		}
	}

	// Perform search
	searchQuery := query.Get("q")
	chunks, err := h.supabaseClient.SearchChunks(r.Context(), searchQuery, filters)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "failed to search chunks", err.Error())
		return
	}

	writeJSONResponse(w, http.StatusOK, chunks)
}

// CreateChunk handles POST /api/v1/chunks
func (h *ChunkHandler) CreateChunk(w http.ResponseWriter, r *http.Request) {
	var req models.CreateChunkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	// Validate required fields
	if req.Content == "" {
		writeErrorResponse(w, http.StatusBadRequest, "content is required", "")
		return
	}

	// Create chunk record
	chunk := &models.ChunkRecord{
		TextID:          req.TextID,
		Content:         req.Content,
		IsTemplate:      req.IsTemplate,
		IsSlot:          req.IsSlot,
		ParentChunkID:   req.ParentChunkID,
		TemplateChunkID: req.TemplateChunkID,
		SlotValue:       req.SlotValue,
		IndentLevel:     req.IndentLevel,
		SequenceNumber:  req.SequenceNumber,
		Metadata:        req.Metadata,
	}

	// Insert chunk
	if err := h.supabaseClient.InsertChunk(r.Context(), chunk); err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "failed to create chunk", err.Error())
		return
	}

	writeJSONResponse(w, http.StatusCreated, chunk)
}

// GetChunkByID handles GET /api/v1/chunks/{id}
func (h *ChunkHandler) GetChunkByID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	chunkID := vars["id"]

	if chunkID == "" {
		writeErrorResponse(w, http.StatusBadRequest, "chunk ID is required", "")
		return
	}

	chunk, err := h.supabaseClient.GetChunkByID(r.Context(), chunkID)
	if err != nil {
		writeErrorResponse(w, http.StatusNotFound, "chunk not found", err.Error())
		return
	}

	writeJSONResponse(w, http.StatusOK, chunk)
}

// UpdateChunk handles PUT /api/v1/chunks/{id}
func (h *ChunkHandler) UpdateChunk(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	chunkID := vars["id"]

	if chunkID == "" {
		writeErrorResponse(w, http.StatusBadRequest, "chunk ID is required", "")
		return
	}

	var req models.UpdateChunkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	// Get existing chunk
	chunk, err := h.supabaseClient.GetChunkByID(r.Context(), chunkID)
	if err != nil {
		writeErrorResponse(w, http.StatusNotFound, "chunk not found", err.Error())
		return
	}

	// Update fields
	if req.Content != nil {
		chunk.Content = *req.Content
	}
	if req.ParentChunkID != nil {
		chunk.ParentChunkID = req.ParentChunkID
	}
	if req.IndentLevel != nil {
		chunk.IndentLevel = *req.IndentLevel
	}
	if req.SequenceNumber != nil {
		chunk.SequenceNumber = req.SequenceNumber
	}
	if req.Metadata != nil {
		chunk.Metadata = req.Metadata
	}

	// Update in database
	if err := h.supabaseClient.UpdateChunk(r.Context(), chunk); err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "failed to update chunk", err.Error())
		return
	}

	writeJSONResponse(w, http.StatusOK, chunk)
}

// DeleteChunk handles DELETE /api/v1/chunks/{id}
func (h *ChunkHandler) DeleteChunk(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	chunkID := vars["id"]

	if chunkID == "" {
		writeErrorResponse(w, http.StatusBadRequest, "chunk ID is required", "")
		return
	}

	if err := h.supabaseClient.DeleteChunk(r.Context(), chunkID); err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "failed to delete chunk", err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetChunkHierarchy handles GET /api/v1/chunks/{id}/hierarchy
func (h *ChunkHandler) GetChunkHierarchy(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	chunkID := vars["id"]

	if chunkID == "" {
		writeErrorResponse(w, http.StatusBadRequest, "chunk ID is required", "")
		return
	}

	hierarchy, err := h.supabaseClient.GetChunkHierarchy(r.Context(), chunkID)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "failed to get chunk hierarchy", err.Error())
		return
	}

	writeJSONResponse(w, http.StatusOK, hierarchy)
}

// GetChunkChildren handles GET /api/v1/chunks/{id}/children
func (h *ChunkHandler) GetChunkChildren(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	chunkID := vars["id"]

	if chunkID == "" {
		writeErrorResponse(w, http.StatusBadRequest, "chunk ID is required", "")
		return
	}

	children, err := h.supabaseClient.GetChildrenChunks(r.Context(), chunkID)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "failed to get chunk children", err.Error())
		return
	}

	writeJSONResponse(w, http.StatusOK, children)
}

// GetChunkSiblings handles GET /api/v1/chunks/{id}/siblings
func (h *ChunkHandler) GetChunkSiblings(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	chunkID := vars["id"]

	if chunkID == "" {
		writeErrorResponse(w, http.StatusBadRequest, "chunk ID is required", "")
		return
	}

	siblings, err := h.supabaseClient.GetSiblingChunks(r.Context(), chunkID)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "failed to get chunk siblings", err.Error())
		return
	}

	writeJSONResponse(w, http.StatusOK, siblings)
}

// MoveChunk handles POST /api/v1/chunks/{id}/move
func (h *ChunkHandler) MoveChunk(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	chunkID := vars["id"]

	if chunkID == "" {
		writeErrorResponse(w, http.StatusBadRequest, "chunk ID is required", "")
		return
	}

	var req models.MoveChunkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	// Set the chunk ID from URL
	req.ChunkID = chunkID

	if err := h.supabaseClient.MoveChunk(r.Context(), &req); err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "failed to move chunk", err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// BulkUpdateChunks handles PUT /api/v1/chunks/bulk-update
func (h *ChunkHandler) BulkUpdateChunks(w http.ResponseWriter, r *http.Request) {
	var req models.BulkUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	if len(req.Updates) == 0 {
		writeErrorResponse(w, http.StatusBadRequest, "no updates provided", "")
		return
	}

	if err := h.supabaseClient.BulkUpdateChunks(r.Context(), &req); err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "failed to bulk update chunks", err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}