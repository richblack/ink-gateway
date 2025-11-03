package handlers

import (
	"encoding/json"
	"net/http"
	"semantic-text-processor/models"
	"semantic-text-processor/services"
	"strconv"

	"github.com/gorilla/mux"
)

// TextHandler handles text-related HTTP requests
type TextHandler struct {
	textProcessor  services.TextProcessor
	supabaseClient services.SupabaseClient
}

// NewTextHandler creates a new text handler
func NewTextHandler(textProcessor services.TextProcessor, supabaseClient services.SupabaseClient) *TextHandler {
	return &TextHandler{
		textProcessor:  textProcessor,
		supabaseClient: supabaseClient,
	}
}

// CreateText handles POST /api/v1/texts
func (h *TextHandler) CreateText(w http.ResponseWriter, r *http.Request) {
	var req models.CreateTextRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	// Validate request
	if req.Content == "" {
		writeErrorResponse(w, http.StatusBadRequest, "content is required", "")
		return
	}

	// Process the text
	result, err := h.textProcessor.ProcessText(r.Context(), req.Content)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "failed to process text", err.Error())
		return
	}

	// Create text record
	textRecord := &models.TextRecord{
		ID:      result.TextID,
		Content: req.Content,
		Title:   req.Title,
		Status:  result.Status,
	}

	// Insert text record
	if err := h.supabaseClient.InsertText(r.Context(), textRecord); err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "failed to save text", err.Error())
		return
	}

	// Insert chunks
	if err := h.supabaseClient.InsertChunks(r.Context(), result.Chunks); err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "failed to save chunks", err.Error())
		return
	}

	// Generate and save embeddings
	embeddings, err := h.textProcessor.GenerateEmbeddings(r.Context(), result.Chunks)
	if err != nil {
		// Log error but don't fail the request
		// Embeddings can be generated later
		writeWarningLog("failed to generate embeddings", err)
	} else {
		if err := h.supabaseClient.InsertEmbeddings(r.Context(), embeddings); err != nil {
			writeWarningLog("failed to save embeddings", err)
		}
	}

	// Extract and save knowledge graph
	graphResult, err := h.textProcessor.ExtractKnowledge(r.Context(), result.Chunks)
	if err != nil {
		writeWarningLog("failed to extract knowledge", err)
	} else {
		if err := h.supabaseClient.InsertGraphNodes(r.Context(), graphResult.Nodes); err != nil {
			writeWarningLog("failed to save graph nodes", err)
		}
		if err := h.supabaseClient.InsertGraphEdges(r.Context(), graphResult.Edges); err != nil {
			writeWarningLog("failed to save graph edges", err)
		}
	}

	// Return success response
	response := models.CreateTextResponse{
		ID:          result.TextID,
		Status:      result.Status,
		ChunkCount:  len(result.Chunks),
		ProcessedAt: result.ProcessedAt,
	}

	writeJSONResponse(w, http.StatusCreated, response)
}

// GetTexts handles GET /api/v1/texts
func (h *TextHandler) GetTexts(w http.ResponseWriter, r *http.Request) {
	// Parse pagination parameters
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	pagination := &models.Pagination{
		Page:     page,
		PageSize: pageSize,
	}

	// Get texts from database
	textList, err := h.supabaseClient.GetTexts(r.Context(), pagination)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "failed to get texts", err.Error())
		return
	}

	writeJSONResponse(w, http.StatusOK, textList)
}

// GetTextByID handles GET /api/v1/texts/{id}
func (h *TextHandler) GetTextByID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	textID := vars["id"]

	if textID == "" {
		writeErrorResponse(w, http.StatusBadRequest, "text ID is required", "")
		return
	}

	// Get text details
	textDetail, err := h.supabaseClient.GetTextByID(r.Context(), textID)
	if err != nil {
		writeErrorResponse(w, http.StatusNotFound, "text not found", err.Error())
		return
	}

	writeJSONResponse(w, http.StatusOK, textDetail)
}

// UpdateText handles PUT /api/v1/texts/{id}
func (h *TextHandler) UpdateText(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	textID := vars["id"]

	if textID == "" {
		writeErrorResponse(w, http.StatusBadRequest, "text ID is required", "")
		return
	}

	var req models.UpdateTextRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	// Get existing text
	existingText, err := h.supabaseClient.GetTextByID(r.Context(), textID)
	if err != nil {
		writeErrorResponse(w, http.StatusNotFound, "text not found", err.Error())
		return
	}

	// Update fields
	textRecord := &existingText.Text
	if req.Title != nil {
		textRecord.Title = *req.Title
	}
	if req.Content != nil {
		textRecord.Content = *req.Content
		// If content changed, mark for reprocessing
		textRecord.Status = "pending_reprocess"
	}

	// Update in database
	if err := h.supabaseClient.UpdateText(r.Context(), textRecord); err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "failed to update text", err.Error())
		return
	}

	writeJSONResponse(w, http.StatusOK, textRecord)
}

// DeleteText handles DELETE /api/v1/texts/{id}
func (h *TextHandler) DeleteText(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	textID := vars["id"]

	if textID == "" {
		writeErrorResponse(w, http.StatusBadRequest, "text ID is required", "")
		return
	}

	// Delete text (cascades to chunks)
	if err := h.supabaseClient.DeleteText(r.Context(), textID); err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "failed to delete text", err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetTextStructure handles GET /api/v1/texts/{id}/structure
func (h *TextHandler) GetTextStructure(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	textID := vars["id"]

	if textID == "" {
		writeErrorResponse(w, http.StatusBadRequest, "text ID is required", "")
		return
	}

	// Get chunks for the text
	chunks, err := h.supabaseClient.GetChunksByTextID(r.Context(), textID)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "failed to get text chunks", err.Error())
		return
	}

	// Build hierarchical structure
	structure := buildBulletStructure(chunks)

	writeJSONResponse(w, http.StatusOK, structure)
}

// UpdateTextStructure handles PUT /api/v1/texts/{id}/structure
func (h *TextHandler) UpdateTextStructure(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	textID := vars["id"]

	if textID == "" {
		writeErrorResponse(w, http.StatusBadRequest, "text ID is required", "")
		return
	}

	var req models.BulkUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	// Perform bulk update
	if err := h.supabaseClient.BulkUpdateChunks(r.Context(), &req); err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "failed to update text structure", err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// buildBulletStructure converts flat chunks to hierarchical structure
func buildBulletStructure(chunks []models.ChunkRecord) *models.BulletStructure {
	// Create a map for quick lookup
	chunkMap := make(map[string]*models.ChunkRecord)
	for i := range chunks {
		chunkMap[chunks[i].ID] = &chunks[i]
	}

	// Build hierarchy
	var rootChunks []models.ChunkHierarchy
	maxDepth := 0

	for i := range chunks {
		chunk := &chunks[i]
		if chunk.ParentChunkID == nil {
			// This is a root chunk
			hierarchy := buildChunkHierarchy(chunk, chunkMap, 0)
			rootChunks = append(rootChunks, hierarchy)
			
			// Calculate max depth for this hierarchy
			hierarchyDepth := calculateMaxDepth(hierarchy)
			if hierarchyDepth > maxDepth {
				maxDepth = hierarchyDepth
			}
		}
	}

	return &models.BulletStructure{
		RootChunks: rootChunks,
		MaxDepth:   maxDepth,
	}
}

// buildChunkHierarchy recursively builds chunk hierarchy
func buildChunkHierarchy(chunk *models.ChunkRecord, chunkMap map[string]*models.ChunkRecord, level int) models.ChunkHierarchy {
	hierarchy := models.ChunkHierarchy{
		Chunk:    chunk,
		Children: []models.ChunkHierarchy{},
		Level:    level,
	}

	// Find children
	for _, otherChunk := range chunkMap {
		if otherChunk.ParentChunkID != nil && *otherChunk.ParentChunkID == chunk.ID {
			childHierarchy := buildChunkHierarchy(otherChunk, chunkMap, level+1)
			hierarchy.Children = append(hierarchy.Children, childHierarchy)
		}
	}

	return hierarchy
}

// calculateMaxDepth recursively calculates the maximum depth of a hierarchy
func calculateMaxDepth(hierarchy models.ChunkHierarchy) int {
	maxChildDepth := hierarchy.Level
	for _, child := range hierarchy.Children {
		childDepth := calculateMaxDepth(child)
		if childDepth > maxChildDepth {
			maxChildDepth = childDepth
		}
	}
	return maxChildDepth
}