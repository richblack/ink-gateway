package services

import (
	"context"
	"fmt"
	"testing"
	"time"

	"semantic-text-processor/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockSupabaseClientForTemplate for testing template service
type MockSupabaseClientForTemplate struct {
	mock.Mock
}

func (m *MockSupabaseClientForTemplate) CreateTemplate(ctx context.Context, templateName string, slotNames []string) (*models.TemplateWithInstances, error) {
	args := m.Called(ctx, templateName, slotNames)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.TemplateWithInstances), args.Error(1)
}

func (m *MockSupabaseClientForTemplate) GetTemplateByContent(ctx context.Context, templateContent string) (*models.TemplateWithInstances, error) {
	args := m.Called(ctx, templateContent)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.TemplateWithInstances), args.Error(1)
}

func (m *MockSupabaseClientForTemplate) GetAllTemplates(ctx context.Context) ([]models.TemplateWithInstances, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.TemplateWithInstances), args.Error(1)
}

func (m *MockSupabaseClientForTemplate) CreateTemplateInstance(ctx context.Context, req *models.CreateInstanceRequest) (*models.TemplateInstance, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.TemplateInstance), args.Error(1)
}

func (m *MockSupabaseClientForTemplate) UpdateSlotValue(ctx context.Context, instanceChunkID, slotName, value string) error {
	args := m.Called(ctx, instanceChunkID, slotName, value)
	return args.Error(0)
}

// Implement other required methods as no-ops for this test
func (m *MockSupabaseClientForTemplate) InsertText(ctx context.Context, text *models.TextRecord) error { return nil }
func (m *MockSupabaseClientForTemplate) GetTexts(ctx context.Context, pagination *models.Pagination) (*models.TextList, error) { return nil, nil }
func (m *MockSupabaseClientForTemplate) GetTextByID(ctx context.Context, id string) (*models.TextDetail, error) { return nil, nil }
func (m *MockSupabaseClientForTemplate) UpdateText(ctx context.Context, text *models.TextRecord) error { return nil }
func (m *MockSupabaseClientForTemplate) DeleteText(ctx context.Context, id string) error { return nil }
func (m *MockSupabaseClientForTemplate) InsertChunk(ctx context.Context, chunk *models.ChunkRecord) error { return nil }
func (m *MockSupabaseClientForTemplate) InsertChunks(ctx context.Context, chunks []models.ChunkRecord) error { return nil }
func (m *MockSupabaseClientForTemplate) GetChunkByID(ctx context.Context, id string) (*models.ChunkRecord, error) { return nil, nil }
func (m *MockSupabaseClientForTemplate) GetChunkByContent(ctx context.Context, content string) (*models.ChunkRecord, error) { return nil, nil }
func (m *MockSupabaseClientForTemplate) UpdateChunk(ctx context.Context, chunk *models.ChunkRecord) error { return nil }
func (m *MockSupabaseClientForTemplate) DeleteChunk(ctx context.Context, id string) error { return nil }
func (m *MockSupabaseClientForTemplate) GetChunksByTextID(ctx context.Context, textID string) ([]models.ChunkRecord, error) { return nil, nil }
func (m *MockSupabaseClientForTemplate) GetTemplateInstances(ctx context.Context, templateChunkID string) ([]models.TemplateInstance, error) { return nil, nil }
func (m *MockSupabaseClientForTemplate) AddTag(ctx context.Context, chunkID string, tagContent string) error { return nil }
func (m *MockSupabaseClientForTemplate) RemoveTag(ctx context.Context, chunkID string, tagChunkID string) error { return nil }
func (m *MockSupabaseClientForTemplate) GetChunkTags(ctx context.Context, chunkID string) ([]models.ChunkRecord, error) { return nil, nil }
func (m *MockSupabaseClientForTemplate) GetChunksByTag(ctx context.Context, tagContent string) ([]models.ChunkRecord, error) { return nil, nil }
func (m *MockSupabaseClientForTemplate) GetChunkHierarchy(ctx context.Context, rootChunkID string) (*models.ChunkHierarchy, error) { return nil, nil }
func (m *MockSupabaseClientForTemplate) GetChildrenChunks(ctx context.Context, parentChunkID string) ([]models.ChunkRecord, error) { return nil, nil }
func (m *MockSupabaseClientForTemplate) GetSiblingChunks(ctx context.Context, chunkID string) ([]models.ChunkRecord, error) { return nil, nil }
func (m *MockSupabaseClientForTemplate) MoveChunk(ctx context.Context, req *models.MoveChunkRequest) error { return nil }
func (m *MockSupabaseClientForTemplate) BulkUpdateChunks(ctx context.Context, req *models.BulkUpdateRequest) error { return nil }
func (m *MockSupabaseClientForTemplate) SearchChunks(ctx context.Context, query string, filters map[string]interface{}) ([]models.ChunkRecord, error) { return nil, nil }
func (m *MockSupabaseClientForTemplate) SearchByTag(ctx context.Context, tagContent string) ([]models.ChunkWithTags, error) { return nil, nil }
func (m *MockSupabaseClientForTemplate) InsertEmbeddings(ctx context.Context, embeddings []models.EmbeddingRecord) error { return nil }
func (m *MockSupabaseClientForTemplate) SearchSimilar(ctx context.Context, queryVector []float64, limit int) ([]models.SimilarityResult, error) { return nil, nil }
func (m *MockSupabaseClientForTemplate) InsertGraphNodes(ctx context.Context, nodes []models.GraphNode) error { return nil }
func (m *MockSupabaseClientForTemplate) InsertGraphEdges(ctx context.Context, edges []models.GraphEdge) error { return nil }
func (m *MockSupabaseClientForTemplate) SearchGraph(ctx context.Context, query *models.GraphQuery) (*models.GraphResult, error) { return nil, nil }
func (m *MockSupabaseClientForTemplate) GetNodesByEntity(ctx context.Context, entityName string) ([]models.GraphNode, error) { return nil, nil }
func (m *MockSupabaseClientForTemplate) GetNodeNeighbors(ctx context.Context, nodeID string, maxDepth int) (*models.GraphResult, error) { return nil, nil }
func (m *MockSupabaseClientForTemplate) FindPathBetweenNodes(ctx context.Context, sourceNodeID, targetNodeID string, maxDepth int) (*models.GraphResult, error) { return nil, nil }
func (m *MockSupabaseClientForTemplate) GetNodesByChunk(ctx context.Context, chunkID string) ([]models.GraphNode, error) { return nil, nil }
func (m *MockSupabaseClientForTemplate) GetEdgesByRelationType(ctx context.Context, relationType string) ([]models.GraphEdge, error) { return nil, nil }
func (m *MockSupabaseClientForTemplate) HealthCheck(ctx context.Context) error { return nil }

func TestTemplateService_CreateTemplate(t *testing.T) {
	tests := []struct {
		name           string
		request        *models.CreateTemplateRequest
		mockResponse   *models.TemplateWithInstances
		mockError      error
		expectedError  string
	}{
		{
			name: "successful template creation",
			request: &models.CreateTemplateRequest{
				TemplateName: "Contact Template",
				SlotNames:    []string{"name", "email", "phone"},
			},
			mockResponse: &models.TemplateWithInstances{
				Template: &models.ChunkRecord{
					ID:         "template-123",
					Content:    "Contact Template#template",
					IsTemplate: true,
					CreatedAt:  time.Now(),
				},
				Slots: []models.ChunkRecord{
					{ID: "slot-1", Content: "#name", IsSlot: true},
					{ID: "slot-2", Content: "#email", IsSlot: true},
					{ID: "slot-3", Content: "#phone", IsSlot: true},
				},
				Instances: []models.TemplateInstance{},
			},
			mockError:     nil,
			expectedError: "",
		},
		{
			name: "empty template name",
			request: &models.CreateTemplateRequest{
				TemplateName: "",
				SlotNames:    []string{"name"},
			},
			mockResponse:  nil,
			mockError:     nil,
			expectedError: "template name is required",
		},
		{
			name: "empty slot names",
			request: &models.CreateTemplateRequest{
				TemplateName: "Test Template",
				SlotNames:    []string{},
			},
			mockResponse:  nil,
			mockError:     nil,
			expectedError: "at least one slot name is required",
		},
		{
			name: "supabase client error",
			request: &models.CreateTemplateRequest{
				TemplateName: "Test Template",
				SlotNames:    []string{"slot1"},
			},
			mockResponse:  nil,
			mockError:     fmt.Errorf("database error"),
			expectedError: "database error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockClient := new(MockSupabaseClientForTemplate)
			service := NewTemplateService(mockClient)

			// Setup mock expectations
			if tt.request.TemplateName != "" && len(tt.request.SlotNames) > 0 {
				mockClient.On("CreateTemplate", mock.Anything, tt.request.TemplateName, tt.request.SlotNames).Return(tt.mockResponse, tt.mockError)
			}

			// Execute
			result, err := service.CreateTemplate(context.Background(), tt.request)

			// Assert
			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.mockResponse.Template.ID, result.Template.ID)
				assert.Equal(t, len(tt.mockResponse.Slots), len(result.Slots))
			}

			mockClient.AssertExpectations(t)
		})
	}
}

func TestTemplateService_GetTemplate(t *testing.T) {
	tests := []struct {
		name            string
		templateContent string
		mockResponse    *models.TemplateWithInstances
		mockError       error
		expectedError   string
	}{
		{
			name:            "successful get template",
			templateContent: "Contact Template",
			mockResponse: &models.TemplateWithInstances{
				Template: &models.ChunkRecord{
					ID:         "template-123",
					Content:    "Contact Template#template",
					IsTemplate: true,
				},
				Slots:     []models.ChunkRecord{{ID: "slot-1", Content: "#name", IsSlot: true}},
				Instances: []models.TemplateInstance{},
			},
			mockError:     nil,
			expectedError: "",
		},
		{
			name:            "empty template content",
			templateContent: "",
			mockResponse:    nil,
			mockError:       nil,
			expectedError:   "template content is required",
		},
		{
			name:            "supabase client error",
			templateContent: "Test Template",
			mockResponse:    nil,
			mockError:       fmt.Errorf("template not found"),
			expectedError:   "template not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockClient := new(MockSupabaseClientForTemplate)
			service := NewTemplateService(mockClient)

			// Setup mock expectations
			if tt.templateContent != "" {
				mockClient.On("GetTemplateByContent", mock.Anything, tt.templateContent).Return(tt.mockResponse, tt.mockError)
			}

			// Execute
			result, err := service.GetTemplate(context.Background(), tt.templateContent)

			// Assert
			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.mockResponse.Template.ID, result.Template.ID)
			}

			mockClient.AssertExpectations(t)
		})
	}
}

func TestTemplateService_GetAllTemplates(t *testing.T) {
	tests := []struct {
		name          string
		mockResponse  []models.TemplateWithInstances
		mockError     error
		expectedError string
		expectedCount int
	}{
		{
			name: "successful get all templates",
			mockResponse: []models.TemplateWithInstances{
				{
					Template: &models.ChunkRecord{ID: "template-1", Content: "Template 1#template", IsTemplate: true},
					Slots:    []models.ChunkRecord{{ID: "slot-1", Content: "#name", IsSlot: true}},
				},
				{
					Template: &models.ChunkRecord{ID: "template-2", Content: "Template 2#template", IsTemplate: true},
					Slots:    []models.ChunkRecord{{ID: "slot-2", Content: "#email", IsSlot: true}},
				},
			},
			mockError:     nil,
			expectedError: "",
			expectedCount: 2,
		},
		{
			name:          "empty templates list",
			mockResponse:  []models.TemplateWithInstances{},
			mockError:     nil,
			expectedError: "",
			expectedCount: 0,
		},
		{
			name:          "supabase client error",
			mockResponse:  nil,
			mockError:     fmt.Errorf("database error"),
			expectedError: "database error",
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockClient := new(MockSupabaseClientForTemplate)
			service := NewTemplateService(mockClient)

			// Setup mock expectations
			mockClient.On("GetAllTemplates", mock.Anything).Return(tt.mockResponse, tt.mockError)

			// Execute
			result, err := service.GetAllTemplates(context.Background())

			// Assert
			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedCount, len(result))
			}

			mockClient.AssertExpectations(t)
		})
	}
}

func TestTemplateService_CreateInstance(t *testing.T) {
	tests := []struct {
		name          string
		request       *models.CreateInstanceRequest
		mockResponse  *models.TemplateInstance
		mockError     error
		expectedError string
	}{
		{
			name: "successful instance creation",
			request: &models.CreateInstanceRequest{
				TemplateChunkID: "template-123",
				InstanceName:    "John Contact",
				SlotValues: map[string]string{
					"name":  "John Doe",
					"email": "john@example.com",
				},
			},
			mockResponse: &models.TemplateInstance{
				Instance: &models.ChunkRecord{
					ID:              "instance-456",
					Content:         "John Contact#Contact Template",
					TemplateChunkID: stringPtr("template-123"),
				},
				SlotValues: map[string]*models.ChunkRecord{
					"name":  {ID: "value-1", Content: "John Doe", SlotValue: stringPtr("John Doe")},
					"email": {ID: "value-2", Content: "john@example.com", SlotValue: stringPtr("john@example.com")},
				},
			},
			mockError:     nil,
			expectedError: "",
		},
		{
			name: "empty template chunk ID",
			request: &models.CreateInstanceRequest{
				TemplateChunkID: "",
				InstanceName:    "Test Instance",
			},
			mockResponse:  nil,
			mockError:     nil,
			expectedError: "template chunk ID is required",
		},
		{
			name: "empty instance name",
			request: &models.CreateInstanceRequest{
				TemplateChunkID: "template-123",
				InstanceName:    "",
			},
			mockResponse:  nil,
			mockError:     nil,
			expectedError: "instance name is required",
		},
		{
			name: "supabase client error",
			request: &models.CreateInstanceRequest{
				TemplateChunkID: "template-123",
				InstanceName:    "Test Instance",
			},
			mockResponse:  nil,
			mockError:     fmt.Errorf("database error"),
			expectedError: "database error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockClient := new(MockSupabaseClientForTemplate)
			service := NewTemplateService(mockClient)

			// Setup mock expectations
			if tt.request.TemplateChunkID != "" && tt.request.InstanceName != "" {
				mockClient.On("CreateTemplateInstance", mock.Anything, tt.request).Return(tt.mockResponse, tt.mockError)
			}

			// Execute
			result, err := service.CreateInstance(context.Background(), tt.request)

			// Assert
			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.mockResponse.Instance.ID, result.Instance.ID)
			}

			mockClient.AssertExpectations(t)
		})
	}
}

func TestTemplateService_UpdateSlotValue(t *testing.T) {
	tests := []struct {
		name            string
		instanceChunkID string
		slotName        string
		value           string
		mockError       error
		expectedError   string
	}{
		{
			name:            "successful slot value update",
			instanceChunkID: "instance-123",
			slotName:        "name",
			value:           "Updated Name",
			mockError:       nil,
			expectedError:   "",
		},
		{
			name:            "empty instance chunk ID",
			instanceChunkID: "",
			slotName:        "name",
			value:           "Updated Name",
			mockError:       nil,
			expectedError:   "instance chunk ID is required",
		},
		{
			name:            "empty slot name",
			instanceChunkID: "instance-123",
			slotName:        "",
			value:           "Updated Name",
			mockError:       nil,
			expectedError:   "slot name is required",
		},
		{
			name:            "supabase client error",
			instanceChunkID: "instance-123",
			slotName:        "name",
			value:           "Updated Name",
			mockError:       fmt.Errorf("database error"),
			expectedError:   "database error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockClient := new(MockSupabaseClientForTemplate)
			service := NewTemplateService(mockClient)

			// Setup mock expectations
			if tt.instanceChunkID != "" && tt.slotName != "" {
				mockClient.On("UpdateSlotValue", mock.Anything, tt.instanceChunkID, tt.slotName, tt.value).Return(tt.mockError)
			}

			// Execute
			err := service.UpdateSlotValue(context.Background(), tt.instanceChunkID, tt.slotName, tt.value)

			// Assert
			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
			}

			mockClient.AssertExpectations(t)
		})
	}
}

// Helper function to create string pointers
func stringPtr(s string) *string {
	return &s
}