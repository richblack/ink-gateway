package handlers

import (
	"encoding/json"
	"net/http"
	"semantic-text-processor/models"
	"semantic-text-processor/services"

	"github.com/gorilla/mux"
)

// TagHandler handles tag-related HTTP requests
type TagHandler struct {
	tagService services.TagService
}

// NewTagHandler creates a new tag handler
func NewTagHandler(tagService services.TagService) *TagHandler {
	return &TagHandler{
		tagService: tagService,
	}
}

// AddTag handles POST /api/v1/chunks/{id}/tags
func (h *TagHandler) AddTag(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	chunkID := vars["id"]

	if chunkID == "" {
		writeErrorResponse(w, http.StatusBadRequest, "chunk ID is required", "")
		return
	}

	var req models.AddTagRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	// Set chunk ID from URL
	req.ChunkID = chunkID

	// Validate request
	if req.TagContent == "" {
		writeErrorResponse(w, http.StatusBadRequest, "tag content is required", "")
		return
	}

	// Add tag
	if err := h.tagService.AddTag(r.Context(), req.ChunkID, req.TagContent); err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "failed to add tag", err.Error())
		return
	}

	w.WriteHeader(http.StatusCreated)
}

// RemoveTag handles DELETE /api/v1/chunks/{id}/tags/{tagId}
func (h *TagHandler) RemoveTag(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	chunkID := vars["id"]
	tagID := vars["tagId"]

	if chunkID == "" {
		writeErrorResponse(w, http.StatusBadRequest, "chunk ID is required", "")
		return
	}

	if tagID == "" {
		writeErrorResponse(w, http.StatusBadRequest, "tag ID is required", "")
		return
	}

	// Remove tag
	if err := h.tagService.RemoveTag(r.Context(), chunkID, tagID); err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "failed to remove tag", err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetChunkTags handles GET /api/v1/chunks/{id}/tags
func (h *TagHandler) GetChunkTags(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	chunkID := vars["id"]

	if chunkID == "" {
		writeErrorResponse(w, http.StatusBadRequest, "chunk ID is required", "")
		return
	}

	// Get chunk tags
	tags, err := h.tagService.GetChunkTags(r.Context(), chunkID)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "failed to get chunk tags", err.Error())
		return
	}

	writeJSONResponse(w, http.StatusOK, tags)
}

// GetChunksByTag handles GET /api/v1/tags/{content}/chunks
func (h *TagHandler) GetChunksByTag(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	tagContent := vars["content"]

	if tagContent == "" {
		writeErrorResponse(w, http.StatusBadRequest, "tag content is required", "")
		return
	}

	// Get chunks by tag
	chunks, err := h.tagService.GetChunksByTag(r.Context(), tagContent)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "failed to get chunks by tag", err.Error())
		return
	}

	writeJSONResponse(w, http.StatusOK, chunks)
}

// AddTagWithInheritance handles POST /api/v1/chunks/{id}/tags/inherit
func (h *TagHandler) AddTagWithInheritance(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	chunkID := vars["id"]

	if chunkID == "" {
		writeErrorResponse(w, http.StatusBadRequest, "chunk ID is required", "")
		return
	}

	var req models.AddTagRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	// Set chunk ID from URL
	req.ChunkID = chunkID

	// Validate request
	if req.TagContent == "" {
		writeErrorResponse(w, http.StatusBadRequest, "tag content is required", "")
		return
	}

	// Add tag with inheritance
	if err := h.tagService.AddTagWithInheritance(r.Context(), req.ChunkID, req.TagContent); err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "failed to add tag with inheritance", err.Error())
		return
	}

	w.WriteHeader(http.StatusCreated)
}

// RemoveTagWithInheritance handles DELETE /api/v1/chunks/{id}/tags/{tagId}/inherit
func (h *TagHandler) RemoveTagWithInheritance(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	chunkID := vars["id"]
	tagID := vars["tagId"]

	if chunkID == "" {
		writeErrorResponse(w, http.StatusBadRequest, "chunk ID is required", "")
		return
	}

	if tagID == "" {
		writeErrorResponse(w, http.StatusBadRequest, "tag ID is required", "")
		return
	}

	// Remove tag with inheritance
	if err := h.tagService.RemoveTagWithInheritance(r.Context(), chunkID, tagID); err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "failed to remove tag with inheritance", err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetInheritedTags handles GET /api/v1/chunks/{id}/tags/inherited
func (h *TagHandler) GetInheritedTags(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	chunkID := vars["id"]

	if chunkID == "" {
		writeErrorResponse(w, http.StatusBadRequest, "chunk ID is required", "")
		return
	}

	// Get inherited tags
	tags, err := h.tagService.GetInheritedTags(r.Context(), chunkID)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "failed to get inherited tags", err.Error())
		return
	}

	writeJSONResponse(w, http.StatusOK, tags)
}