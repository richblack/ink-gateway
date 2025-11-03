package handlers

import (
	"encoding/json"
	"net/http"
	"semantic-text-processor/models"
	"semantic-text-processor/services"

	"github.com/gorilla/mux"
)

// TemplateHandler handles template-related HTTP requests
type TemplateHandler struct {
	templateService services.TemplateService
}

// NewTemplateHandler creates a new template handler
func NewTemplateHandler(templateService services.TemplateService) *TemplateHandler {
	return &TemplateHandler{
		templateService: templateService,
	}
}

// CreateTemplate handles POST /api/v1/templates
func (h *TemplateHandler) CreateTemplate(w http.ResponseWriter, r *http.Request) {
	var req models.CreateTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	// Validate request
	if req.TemplateName == "" {
		writeErrorResponse(w, http.StatusBadRequest, "template name is required", "")
		return
	}

	if len(req.SlotNames) == 0 {
		writeErrorResponse(w, http.StatusBadRequest, "at least one slot name is required", "")
		return
	}

	// Create template
	template, err := h.templateService.CreateTemplate(r.Context(), &req)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "failed to create template", err.Error())
		return
	}

	writeJSONResponse(w, http.StatusCreated, template)
}

// GetAllTemplates handles GET /api/v1/templates
func (h *TemplateHandler) GetAllTemplates(w http.ResponseWriter, r *http.Request) {
	templates, err := h.templateService.GetAllTemplates(r.Context())
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "failed to get templates", err.Error())
		return
	}

	writeJSONResponse(w, http.StatusOK, templates)
}

// GetTemplateByContent handles GET /api/v1/templates/{content}
func (h *TemplateHandler) GetTemplateByContent(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	templateContent := vars["content"]

	if templateContent == "" {
		writeErrorResponse(w, http.StatusBadRequest, "template content is required", "")
		return
	}

	template, err := h.templateService.GetTemplate(r.Context(), templateContent)
	if err != nil {
		writeErrorResponse(w, http.StatusNotFound, "template not found", err.Error())
		return
	}

	writeJSONResponse(w, http.StatusOK, template)
}

// CreateTemplateInstance handles POST /api/v1/templates/{id}/instances
func (h *TemplateHandler) CreateTemplateInstance(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	templateID := vars["id"]

	if templateID == "" {
		writeErrorResponse(w, http.StatusBadRequest, "template ID is required", "")
		return
	}

	var req models.CreateInstanceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	// Set template ID from URL
	req.TemplateChunkID = templateID

	// Validate request
	if req.InstanceName == "" {
		writeErrorResponse(w, http.StatusBadRequest, "instance name is required", "")
		return
	}

	// Create instance
	instance, err := h.templateService.CreateInstance(r.Context(), &req)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "failed to create template instance", err.Error())
		return
	}

	writeJSONResponse(w, http.StatusCreated, instance)
}

// UpdateSlotValue handles PUT /api/v1/instances/{id}/slots
func (h *TemplateHandler) UpdateSlotValue(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	instanceID := vars["id"]

	if instanceID == "" {
		writeErrorResponse(w, http.StatusBadRequest, "instance ID is required", "")
		return
	}

	var req models.UpdateSlotValueRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	// Validate request
	if req.SlotName == "" {
		writeErrorResponse(w, http.StatusBadRequest, "slot name is required", "")
		return
	}

	// Update slot value
	if err := h.templateService.UpdateSlotValue(r.Context(), instanceID, req.SlotName, req.Value); err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "failed to update slot value", err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}