package services

import (
	"context"
	"fmt"
	"semantic-text-processor/models"
)

// templateService implements TemplateService interface
type templateService struct {
	supabaseClient SupabaseClient
}

// NewTemplateService creates a new template service instance
func NewTemplateService(supabaseClient SupabaseClient) TemplateService {
	return &templateService{
		supabaseClient: supabaseClient,
	}
}

// CreateTemplate creates a new template with slots
func (s *templateService) CreateTemplate(ctx context.Context, req *models.CreateTemplateRequest) (*models.TemplateWithInstances, error) {
	if req.TemplateName == "" {
		return nil, fmt.Errorf("template name is required")
	}
	
	if len(req.SlotNames) == 0 {
		return nil, fmt.Errorf("at least one slot name is required")
	}
	
	// Delegate to Supabase client
	return s.supabaseClient.CreateTemplate(ctx, req.TemplateName, req.SlotNames)
}

// GetTemplate retrieves a template by content
func (s *templateService) GetTemplate(ctx context.Context, templateContent string) (*models.TemplateWithInstances, error) {
	if templateContent == "" {
		return nil, fmt.Errorf("template content is required")
	}
	
	// Delegate to Supabase client
	return s.supabaseClient.GetTemplateByContent(ctx, templateContent)
}

// CreateInstance creates a new instance of a template
func (s *templateService) CreateInstance(ctx context.Context, req *models.CreateInstanceRequest) (*models.TemplateInstance, error) {
	if req.TemplateChunkID == "" {
		return nil, fmt.Errorf("template chunk ID is required")
	}
	
	if req.InstanceName == "" {
		return nil, fmt.Errorf("instance name is required")
	}
	
	// Delegate to Supabase client
	return s.supabaseClient.CreateTemplateInstance(ctx, req)
}

// GetAllTemplates retrieves all templates
func (s *templateService) GetAllTemplates(ctx context.Context) ([]models.TemplateWithInstances, error) {
	// Delegate to Supabase client
	return s.supabaseClient.GetAllTemplates(ctx)
}

// UpdateSlotValue updates a slot value in a template instance
func (s *templateService) UpdateSlotValue(ctx context.Context, instanceChunkID, slotName, value string) error {
	if instanceChunkID == "" {
		return fmt.Errorf("instance chunk ID is required")
	}
	
	if slotName == "" {
		return fmt.Errorf("slot name is required")
	}
	
	// Delegate to Supabase client
	return s.supabaseClient.UpdateSlotValue(ctx, instanceChunkID, slotName, value)
}