package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"semantic-text-processor/models"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockTemplateService for testing
type MockTemplateService struct {
	mock.Mock
}

func (m *MockTemplateService) CreateTemplate(ctx context.Context, req *models.CreateTemplateRequest) (*models.TemplateWithInstances, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.TemplateWithInstances), args.Error(1)
}

func (m *MockTemplateService) GetTemplate(ctx context.Context, templateContent string) (*models.TemplateWithInstances, error) {
	args := m.Called(ctx, templateContent)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.TemplateWithInstances), args.Error(1)
}

func (m *MockTemplateService) GetAllTemplates(ctx context.Context) ([]models.TemplateWithInstances, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.TemplateWithInstances), args.Error(1)
}

func (m *MockTemplateService) CreateInstance(ctx context.Context, req *models.CreateInstanceRequest) (*models.TemplateInstance, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.TemplateInstance), args.Error(1)
}

func (m *MockTemplateService) UpdateSlotValue(ctx context.Context, instanceChunkID, slotName, value string) error {
	args := m.Called(ctx, instanceChunkID, slotName, value)
	return args.Error(0)
}

func TestTemplateHandler_CreateTemplate(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    models.CreateTemplateRequest
		mockResponse   *models.TemplateWithInstances
		mockError      error
		expectedStatus int
	}{
		{
			name: "successful template creation",
			requestBody: models.CreateTemplateRequest{
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
			mockError:      nil,
			expectedStatus: http.StatusCreated,
		},
		{
			name: "missing template name",
			requestBody: models.CreateTemplateRequest{
				TemplateName: "",
				SlotNames:    []string{"name"},
			},
			mockResponse:   nil,
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "missing slot names",
			requestBody: models.CreateTemplateRequest{
				TemplateName: "Test Template",
				SlotNames:    []string{},
			},
			mockResponse:   nil,
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "service error",
			requestBody: models.CreateTemplateRequest{
				TemplateName: "Test Template",
				SlotNames:    []string{"slot1"},
			},
			mockResponse:   nil,
			mockError:      fmt.Errorf("service error"),
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockService := new(MockTemplateService)
			handler := NewTemplateHandler(mockService)

			// Setup mock expectations
			if tt.requestBody.TemplateName != "" && len(tt.requestBody.SlotNames) > 0 {
				mockService.On("CreateTemplate", mock.Anything, &tt.requestBody).Return(tt.mockResponse, tt.mockError)
			}

			// Create request
			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest("POST", "/api/v1/templates", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			// Execute
			handler.CreateTemplate(w, req)

			// Assert
			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusCreated {
				var response models.TemplateWithInstances
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, tt.mockResponse.Template.ID, response.Template.ID)
				assert.Equal(t, len(tt.mockResponse.Slots), len(response.Slots))
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestTemplateHandler_GetAllTemplates(t *testing.T) {
	tests := []struct {
		name           string
		mockResponse   []models.TemplateWithInstances
		mockError      error
		expectedStatus int
		expectedCount  int
	}{
		{
			name: "successful get all templates",
			mockResponse: []models.TemplateWithInstances{
				{
					Template: &models.ChunkRecord{
						ID:         "template-1",
						Content:    "Template 1#template",
						IsTemplate: true,
					},
					Slots:     []models.ChunkRecord{{ID: "slot-1", Content: "#name", IsSlot: true}},
					Instances: []models.TemplateInstance{},
				},
				{
					Template: &models.ChunkRecord{
						ID:         "template-2",
						Content:    "Template 2#template",
						IsTemplate: true,
					},
					Slots:     []models.ChunkRecord{{ID: "slot-2", Content: "#email", IsSlot: true}},
					Instances: []models.TemplateInstance{},
				},
			},
			mockError:      nil,
			expectedStatus: http.StatusOK,
			expectedCount:  2,
		},
		{
			name:           "empty templates list",
			mockResponse:   []models.TemplateWithInstances{},
			mockError:      nil,
			expectedStatus: http.StatusOK,
			expectedCount:  0,
		},
		{
			name:           "service error",
			mockResponse:   nil,
			mockError:      fmt.Errorf("service error"),
			expectedStatus: http.StatusInternalServerError,
			expectedCount:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockService := new(MockTemplateService)
			handler := NewTemplateHandler(mockService)

			// Setup mock expectations
			mockService.On("GetAllTemplates", mock.Anything).Return(tt.mockResponse, tt.mockError)

			// Create request
			req := httptest.NewRequest("GET", "/api/v1/templates", nil)
			w := httptest.NewRecorder()

			// Execute
			handler.GetAllTemplates(w, req)

			// Assert
			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusOK {
				var response []models.TemplateWithInstances
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedCount, len(response))
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestTemplateHandler_GetTemplateByContent(t *testing.T) {
	tests := []struct {
		name           string
		templateContent string
		mockResponse   *models.TemplateWithInstances
		mockError      error
		expectedStatus int
	}{
		{
			name:            "successful get template by content",
			templateContent: "ContactTemplate",
			mockResponse: &models.TemplateWithInstances{
				Template: &models.ChunkRecord{
					ID:         "template-123",
					Content:    "Contact Template#template",
					IsTemplate: true,
				},
				Slots: []models.ChunkRecord{
					{ID: "slot-1", Content: "#name", IsSlot: true},
				},
				Instances: []models.TemplateInstance{},
			},
			mockError:      nil,
			expectedStatus: http.StatusOK,
		},
		{
			name:            "empty template content",
			templateContent: "",
			mockResponse:    nil,
			mockError:       nil,
			expectedStatus:  http.StatusBadRequest,
		},
		{
			name:            "template not found",
			templateContent: "NonExistentTemplate",
			mockResponse:    nil,
			mockError:       fmt.Errorf("template not found"),
			expectedStatus:  http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockService := new(MockTemplateService)
			handler := NewTemplateHandler(mockService)

			// Setup mock expectations
			if tt.templateContent != "" {
				mockService.On("GetTemplate", mock.Anything, tt.templateContent).Return(tt.mockResponse, tt.mockError)
			}

			// Create request with mux vars
			req := httptest.NewRequest("GET", "/api/v1/templates/"+tt.templateContent, nil)
			w := httptest.NewRecorder()

			// Setup mux vars
			req = mux.SetURLVars(req, map[string]string{"content": tt.templateContent})

			// Execute
			handler.GetTemplateByContent(w, req)

			// Assert
			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusOK {
				var response models.TemplateWithInstances
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, tt.mockResponse.Template.ID, response.Template.ID)
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestTemplateHandler_CreateTemplateInstance(t *testing.T) {
	tests := []struct {
		name           string
		templateID     string
		requestBody    models.CreateInstanceRequest
		mockResponse   *models.TemplateInstance
		mockError      error
		expectedStatus int
	}{
		{
			name:       "successful instance creation",
			templateID: "template-123",
			requestBody: models.CreateInstanceRequest{
				InstanceName: "John Contact",
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
			mockError:      nil,
			expectedStatus: http.StatusCreated,
		},
		{
			name:       "empty template ID",
			templateID: "",
			requestBody: models.CreateInstanceRequest{
				InstanceName: "Test Instance",
			},
			mockResponse:   nil,
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:       "empty instance name",
			templateID: "template-123",
			requestBody: models.CreateInstanceRequest{
				InstanceName: "",
			},
			mockResponse:   nil,
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:       "service error",
			templateID: "template-123",
			requestBody: models.CreateInstanceRequest{
				InstanceName: "Test Instance",
			},
			mockResponse:   nil,
			mockError:      fmt.Errorf("service error"),
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockService := new(MockTemplateService)
			handler := NewTemplateHandler(mockService)

			// Setup mock expectations
			if tt.templateID != "" && tt.requestBody.InstanceName != "" {
				expectedReq := tt.requestBody
				expectedReq.TemplateChunkID = tt.templateID
				mockService.On("CreateInstance", mock.Anything, &expectedReq).Return(tt.mockResponse, tt.mockError)
			}

			// Create request
			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest("POST", "/api/v1/templates/"+tt.templateID+"/instances", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			// Setup mux vars
			req = mux.SetURLVars(req, map[string]string{"id": tt.templateID})

			// Execute
			handler.CreateTemplateInstance(w, req)

			// Assert
			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusCreated {
				var response models.TemplateInstance
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, tt.mockResponse.Instance.ID, response.Instance.ID)
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestTemplateHandler_UpdateSlotValue(t *testing.T) {
	tests := []struct {
		name           string
		instanceID     string
		requestBody    models.UpdateSlotValueRequest
		mockError      error
		expectedStatus int
	}{
		{
			name:       "successful slot value update",
			instanceID: "instance-123",
			requestBody: models.UpdateSlotValueRequest{
				SlotName: "name",
				Value:    "Updated Name",
			},
			mockError:      nil,
			expectedStatus: http.StatusNoContent,
		},
		{
			name:       "empty instance ID",
			instanceID: "",
			requestBody: models.UpdateSlotValueRequest{
				SlotName: "name",
				Value:    "Updated Name",
			},
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:       "empty slot name",
			instanceID: "instance-123",
			requestBody: models.UpdateSlotValueRequest{
				SlotName: "",
				Value:    "Updated Name",
			},
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:       "service error",
			instanceID: "instance-123",
			requestBody: models.UpdateSlotValueRequest{
				SlotName: "name",
				Value:    "Updated Name",
			},
			mockError:      fmt.Errorf("service error"),
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockService := new(MockTemplateService)
			handler := NewTemplateHandler(mockService)

			// Setup mock expectations
			if tt.instanceID != "" && tt.requestBody.SlotName != "" {
				mockService.On("UpdateSlotValue", mock.Anything, tt.instanceID, tt.requestBody.SlotName, tt.requestBody.Value).Return(tt.mockError)
			}

			// Create request
			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest("PUT", "/api/v1/instances/"+tt.instanceID+"/slots", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			// Setup mux vars
			req = mux.SetURLVars(req, map[string]string{"id": tt.instanceID})

			// Execute
			handler.UpdateSlotValue(w, req)

			// Assert
			assert.Equal(t, tt.expectedStatus, w.Code)

			mockService.AssertExpectations(t)
		})
	}
}

