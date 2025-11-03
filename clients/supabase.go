package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"semantic-text-processor/config"
	"semantic-text-processor/models"

	"github.com/google/uuid"
)

// SupabaseClient defines the interface for Supabase operations
type SupabaseClient interface {
	// Basic Text operations
	InsertText(ctx context.Context, text *models.TextRecord) error
	GetTexts(ctx context.Context, pagination *models.Pagination) (*models.TextList, error)
	GetTextByID(ctx context.Context, id string) (*models.TextDetail, error)
	UpdateText(ctx context.Context, text *models.TextRecord) error
	DeleteText(ctx context.Context, id string) error

	// Chunk operations
	InsertChunk(ctx context.Context, chunk *models.ChunkRecord) error
	InsertChunks(ctx context.Context, chunks []models.ChunkRecord) error
	GetChunkByID(ctx context.Context, id string) (*models.ChunkRecord, error)
	GetChunkByContent(ctx context.Context, content string) (*models.ChunkRecord, error)
	UpdateChunk(ctx context.Context, chunk *models.ChunkRecord) error
	DeleteChunk(ctx context.Context, id string) error
	GetChunksByTextID(ctx context.Context, textID string) ([]models.ChunkRecord, error)

	// Template operations
	CreateTemplate(ctx context.Context, templateName string, slotNames []string) (*models.TemplateWithInstances, error)
	GetTemplateByContent(ctx context.Context, templateContent string) (*models.TemplateWithInstances, error)
	GetAllTemplates(ctx context.Context) ([]models.TemplateWithInstances, error)

	// Template instance operations
	CreateTemplateInstance(ctx context.Context, req *models.CreateInstanceRequest) (*models.TemplateInstance, error)
	GetTemplateInstances(ctx context.Context, templateChunkID string) ([]models.TemplateInstance, error)
	UpdateSlotValue(ctx context.Context, instanceChunkID, slotName, value string) error

	// Tag operations
	AddTag(ctx context.Context, chunkID string, tagContent string) error
	RemoveTag(ctx context.Context, chunkID string, tagChunkID string) error
	GetChunkTags(ctx context.Context, chunkID string) ([]models.ChunkRecord, error)
	GetChunksByTag(ctx context.Context, tagContent string) ([]models.ChunkRecord, error)

	// Hierarchy operations
	GetChunkHierarchy(ctx context.Context, rootChunkID string) (*models.ChunkHierarchy, error)
	GetChildrenChunks(ctx context.Context, parentChunkID string) ([]models.ChunkRecord, error)
	GetSiblingChunks(ctx context.Context, chunkID string) ([]models.ChunkRecord, error)
	MoveChunk(ctx context.Context, req *models.MoveChunkRequest) error
	BulkUpdateChunks(ctx context.Context, req *models.BulkUpdateRequest) error

	// Search operations
	SearchChunks(ctx context.Context, query string, filters map[string]interface{}) ([]models.ChunkRecord, error)
	SearchByTag(ctx context.Context, tagContent string) ([]models.ChunkWithTags, error)

	// Vector operations
	InsertEmbeddings(ctx context.Context, embeddings []models.EmbeddingRecord) error
	SearchSimilar(ctx context.Context, queryVector []float64, limit int) ([]models.SimilarityResult, error)

	// Graph operations
	InsertGraphNodes(ctx context.Context, nodes []models.GraphNode) error
	InsertGraphEdges(ctx context.Context, edges []models.GraphEdge) error
	SearchGraph(ctx context.Context, query *models.GraphQuery) (*models.GraphResult, error)
	GetNodesByEntity(ctx context.Context, entityName string) ([]models.GraphNode, error)
	GetNodeNeighbors(ctx context.Context, nodeID string, maxDepth int) (*models.GraphResult, error)
	FindPathBetweenNodes(ctx context.Context, sourceNodeID, targetNodeID string, maxDepth int) (*models.GraphResult, error)
	GetNodesByChunk(ctx context.Context, chunkID string) ([]models.GraphNode, error)
	GetEdgesByRelationType(ctx context.Context, relationType string) ([]models.GraphEdge, error)

	// Health check
	HealthCheck(ctx context.Context) error
}

// supabaseHTTPClient implements SupabaseClient using HTTP REST API
type supabaseHTTPClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewSupabaseClient creates a new Supabase HTTP client
func NewSupabaseClient(cfg *config.SupabaseConfig) SupabaseClient {
	return &supabaseHTTPClient{
		baseURL: strings.TrimSuffix(cfg.URL, "/") + "/rest/v1",
		apiKey:  cfg.APIKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SupabaseError represents errors from Supabase API
type SupabaseError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details"`
	Hint    string `json:"hint"`
}

func (e *SupabaseError) Error() string {
	return fmt.Sprintf("supabase error [%s]: %s", e.Code, e.Message)
}

// RetryConfig defines retry behavior for failed requests
type RetryConfig struct {
	MaxRetries int
	BaseDelay  time.Duration
	MaxDelay   time.Duration
}

// DefaultRetryConfig provides sensible defaults for retry behavior
var DefaultRetryConfig = &RetryConfig{
	MaxRetries: 3,
	BaseDelay:  100 * time.Millisecond,
	MaxDelay:   5 * time.Second,
}

// executeWithRetry executes an operation with exponential backoff retry
func (c *supabaseHTTPClient) executeWithRetry(ctx context.Context, operation func() error) error {
	var lastErr error
	
	for attempt := 0; attempt <= DefaultRetryConfig.MaxRetries; attempt++ {
		if attempt > 0 {
			// Calculate delay with exponential backoff
			delay := time.Duration(attempt) * DefaultRetryConfig.BaseDelay
			if delay > DefaultRetryConfig.MaxDelay {
				delay = DefaultRetryConfig.MaxDelay
			}
			
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}
		
		if err := operation(); err != nil {
			lastErr = err
			// Check if error is retryable
			if !isRetryableError(err) {
				return err
			}
			continue
		}
		
		return nil
	}
	
	return fmt.Errorf("operation failed after %d retries: %w", DefaultRetryConfig.MaxRetries, lastErr)
}

// isRetryableError determines if an error should trigger a retry
func isRetryableError(err error) bool {
	if supabaseErr, ok := err.(*SupabaseError); ok {
		// Don't retry client errors (4xx), but retry server errors (5xx)
		return strings.HasPrefix(supabaseErr.Code, "5")
	}
	return true // Retry network errors and other unknown errors
}

// makeRequest performs HTTP request to Supabase with authentication
func (c *supabaseHTTPClient) makeRequest(ctx context.Context, method, endpoint string, body interface{}, result interface{}) error {
	return c.executeWithRetry(ctx, func() error {
		return c.doRequest(ctx, method, endpoint, body, result)
	})
}

// doRequest performs the actual HTTP request
func (c *supabaseHTTPClient) doRequest(ctx context.Context, method, endpoint string, body interface{}, result interface{}) error {
	var reqBody io.Reader
	
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonData)
	}
	
	url := c.baseURL + endpoint
	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	
	// Set required headers
	req.Header.Set("apikey", c.apiKey)
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Prefer", "return=representation")
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}
	
	// Handle error responses
	if resp.StatusCode >= 400 {
		var supabaseErr SupabaseError
		if err := json.Unmarshal(respBody, &supabaseErr); err != nil {
			return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
		}
		return &supabaseErr
	}
	
	// Parse successful response
	if result != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("failed to unmarshal response: %w", err)
		}
	}
	
	return nil
}

// HealthCheck verifies connection to Supabase
func (c *supabaseHTTPClient) HealthCheck(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	
	// Simple health check by querying a system table
	endpoint := "/texts?select=count&limit=1"
	var result []map[string]interface{}
	
	return c.makeRequest(ctx, "GET", endpoint, nil, &result)
}

// Helper function to generate UUID
func generateUUID() string {
	return uuid.New().String()
}

// Helper function to build query parameters
func buildQueryParams(params map[string]string) string {
	if len(params) == 0 {
		return ""
	}
	
	values := url.Values{}
	for key, value := range params {
		values.Add(key, value)
	}
	
	return "?" + values.Encode()
}

// InsertText creates a new text record in Supabase
func (c *supabaseHTTPClient) InsertText(ctx context.Context, text *models.TextRecord) error {
	if text.ID == "" {
		text.ID = generateUUID()
	}
	if text.CreatedAt.IsZero() {
		text.CreatedAt = time.Now()
	}
	if text.UpdatedAt.IsZero() {
		text.UpdatedAt = time.Now()
	}
	if text.Status == "" {
		text.Status = "processing"
	}
	
	var result []models.TextRecord
	err := c.makeRequest(ctx, "POST", "/texts", text, &result)
	if err != nil {
		return fmt.Errorf("failed to insert text: %w", err)
	}
	
	if len(result) > 0 {
		*text = result[0] // Update with returned data including server-generated fields
	}
	
	return nil
}

// GetTexts retrieves paginated list of texts
func (c *supabaseHTTPClient) GetTexts(ctx context.Context, pagination *models.Pagination) (*models.TextList, error) {
	if pagination == nil {
		pagination = &models.Pagination{Page: 1, PageSize: 20}
	}
	
	offset := (pagination.Page - 1) * pagination.PageSize
	params := map[string]string{
		"select": "*",
		"limit":  strconv.Itoa(pagination.PageSize),
		"offset": strconv.Itoa(offset),
		"order":  "created_at.desc",
	}
	
	endpoint := "/texts" + buildQueryParams(params)
	var texts []models.TextRecord
	
	err := c.makeRequest(ctx, "GET", endpoint, nil, &texts)
	if err != nil {
		return nil, fmt.Errorf("failed to get texts: %w", err)
	}
	
	// Get total count for pagination
	countParams := map[string]string{
		"select": "count",
	}
	countEndpoint := "/texts" + buildQueryParams(countParams)
	var countResult []map[string]interface{}
	
	err = c.makeRequest(ctx, "GET", countEndpoint, nil, &countResult)
	if err != nil {
		return nil, fmt.Errorf("failed to get text count: %w", err)
	}
	
	total := 0
	if len(countResult) > 0 {
		if count, ok := countResult[0]["count"].(float64); ok {
			total = int(count)
		}
	}
	
	pagination.Total = total
	
	return &models.TextList{
		Texts:      texts,
		Pagination: *pagination,
	}, nil
}

// GetTextByID retrieves a specific text with its chunks
func (c *supabaseHTTPClient) GetTextByID(ctx context.Context, id string) (*models.TextDetail, error) {
	// Get text record
	params := map[string]string{
		"select": "*",
		"id":     "eq." + id,
	}
	endpoint := "/texts" + buildQueryParams(params)
	
	var texts []models.TextRecord
	err := c.makeRequest(ctx, "GET", endpoint, nil, &texts)
	if err != nil {
		return nil, fmt.Errorf("failed to get text: %w", err)
	}
	
	if len(texts) == 0 {
		return nil, fmt.Errorf("text not found: %s", id)
	}
	
	// Get associated chunks
	chunks, err := c.GetChunksByTextID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get chunks for text: %w", err)
	}
	
	return &models.TextDetail{
		Text:   texts[0],
		Chunks: chunks,
	}, nil
}

// UpdateText updates an existing text record
func (c *supabaseHTTPClient) UpdateText(ctx context.Context, text *models.TextRecord) error {
	text.UpdatedAt = time.Now()
	
	params := map[string]string{
		"id": "eq." + text.ID,
	}
	endpoint := "/texts" + buildQueryParams(params)
	
	var result []models.TextRecord
	err := c.makeRequest(ctx, "PATCH", endpoint, text, &result)
	if err != nil {
		return fmt.Errorf("failed to update text: %w", err)
	}
	
	if len(result) > 0 {
		*text = result[0]
	}
	
	return nil
}

// DeleteText removes a text record and its associated chunks
func (c *supabaseHTTPClient) DeleteText(ctx context.Context, id string) error {
	params := map[string]string{
		"id": "eq." + id,
	}
	endpoint := "/texts" + buildQueryParams(params)
	
	err := c.makeRequest(ctx, "DELETE", endpoint, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to delete text: %w", err)
	}
	
	return nil
}

// InsertChunk creates a new chunk record
func (c *supabaseHTTPClient) InsertChunk(ctx context.Context, chunk *models.ChunkRecord) error {
	if chunk.ID == "" {
		chunk.ID = generateUUID()
	}
	if chunk.CreatedAt.IsZero() {
		chunk.CreatedAt = time.Now()
	}
	if chunk.UpdatedAt.IsZero() {
		chunk.UpdatedAt = time.Now()
	}
	
	var result []models.ChunkRecord
	err := c.makeRequest(ctx, "POST", "/chunks", chunk, &result)
	if err != nil {
		return fmt.Errorf("failed to insert chunk: %w", err)
	}
	
	if len(result) > 0 {
		*chunk = result[0]
	}
	
	return nil
}

// InsertChunks creates multiple chunk records in batch
func (c *supabaseHTTPClient) InsertChunks(ctx context.Context, chunks []models.ChunkRecord) error {
	if len(chunks) == 0 {
		return nil
	}
	
	// Prepare chunks with required fields
	for i := range chunks {
		if chunks[i].ID == "" {
			chunks[i].ID = generateUUID()
		}
		if chunks[i].CreatedAt.IsZero() {
			chunks[i].CreatedAt = time.Now()
		}
		if chunks[i].UpdatedAt.IsZero() {
			chunks[i].UpdatedAt = time.Now()
		}
	}
	
	var result []models.ChunkRecord
	err := c.makeRequest(ctx, "POST", "/chunks", chunks, &result)
	if err != nil {
		return fmt.Errorf("failed to insert chunks: %w", err)
	}
	
	// Update chunks with returned data
	if len(result) == len(chunks) {
		copy(chunks, result)
	}
	
	return nil
}

// GetChunkByID retrieves a specific chunk by ID
func (c *supabaseHTTPClient) GetChunkByID(ctx context.Context, id string) (*models.ChunkRecord, error) {
	params := map[string]string{
		"select": "*",
		"id":     "eq." + id,
	}
	endpoint := "/chunks" + buildQueryParams(params)
	
	var chunks []models.ChunkRecord
	err := c.makeRequest(ctx, "GET", endpoint, nil, &chunks)
	if err != nil {
		return nil, fmt.Errorf("failed to get chunk: %w", err)
	}
	
	if len(chunks) == 0 {
		return nil, fmt.Errorf("chunk not found: %s", id)
	}
	
	return &chunks[0], nil
}

// GetChunkByContent retrieves a chunk by its content
func (c *supabaseHTTPClient) GetChunkByContent(ctx context.Context, content string) (*models.ChunkRecord, error) {
	params := map[string]string{
		"select":  "*",
		"content": "eq." + content,
		"limit":   "1",
	}
	endpoint := "/chunks" + buildQueryParams(params)
	
	var chunks []models.ChunkRecord
	err := c.makeRequest(ctx, "GET", endpoint, nil, &chunks)
	if err != nil {
		return nil, fmt.Errorf("failed to get chunk by content: %w", err)
	}
	
	if len(chunks) == 0 {
		return nil, fmt.Errorf("chunk not found with content: %s", content)
	}
	
	return &chunks[0], nil
}

// UpdateChunk updates an existing chunk record
func (c *supabaseHTTPClient) UpdateChunk(ctx context.Context, chunk *models.ChunkRecord) error {
	chunk.UpdatedAt = time.Now()
	
	params := map[string]string{
		"id": "eq." + chunk.ID,
	}
	endpoint := "/chunks" + buildQueryParams(params)
	
	var result []models.ChunkRecord
	err := c.makeRequest(ctx, "PATCH", endpoint, chunk, &result)
	if err != nil {
		return fmt.Errorf("failed to update chunk: %w", err)
	}
	
	if len(result) > 0 {
		*chunk = result[0]
	}
	
	return nil
}

// DeleteChunk removes a chunk record
func (c *supabaseHTTPClient) DeleteChunk(ctx context.Context, id string) error {
	params := map[string]string{
		"id": "eq." + id,
	}
	endpoint := "/chunks" + buildQueryParams(params)
	
	err := c.makeRequest(ctx, "DELETE", endpoint, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to delete chunk: %w", err)
	}
	
	return nil
}

// GetChunksByTextID retrieves all chunks for a specific text
func (c *supabaseHTTPClient) GetChunksByTextID(ctx context.Context, textID string) ([]models.ChunkRecord, error) {
	params := map[string]string{
		"select":  "*",
		"text_id": "eq." + textID,
		"order":   "sequence_number.asc,created_at.asc",
	}
	endpoint := "/chunks" + buildQueryParams(params)
	
	var chunks []models.ChunkRecord
	err := c.makeRequest(ctx, "GET", endpoint, nil, &chunks)
	if err != nil {
		return nil, fmt.Errorf("failed to get chunks by text ID: %w", err)
	}
	
	return chunks, nil
}

// Stub implementations for remaining interface methods (to be implemented in later tasks)

// CreateTemplate creates a new template with slots
func (c *supabaseHTTPClient) CreateTemplate(ctx context.Context, templateName string, slotNames []string) (*models.TemplateWithInstances, error) {
	// Create the main template chunk
	templateChunk := &models.ChunkRecord{
		ID:          generateUUID(),
		Content:     templateName + "#template",
		IsTemplate:  true,
		IsSlot:      false,
		IndentLevel: 0,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	
	// For templates, we need to create a dummy text record or associate with existing one
	// For now, create a dedicated text for templates
	templateText := &models.TextRecord{
		ID:        generateUUID(),
		Content:   "Template: " + templateName,
		Title:     "Template: " + templateName,
		Status:    "completed",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	
	// Insert the template text
	err := c.InsertText(ctx, templateText)
	if err != nil {
		return nil, fmt.Errorf("failed to create template text: %w", err)
	}
	
	templateChunk.TextID = templateText.ID
	
	// Insert the template chunk
	err = c.InsertChunk(ctx, templateChunk)
	if err != nil {
		return nil, fmt.Errorf("failed to create template chunk: %w", err)
	}
	
	// Create slot chunks
	var slotChunks []models.ChunkRecord
	for i, slotName := range slotNames {
		slotChunk := models.ChunkRecord{
			ID:              generateUUID(),
			TextID:          templateText.ID,
			Content:         "#" + slotName,
			IsTemplate:      false,
			IsSlot:          true,
			ParentChunkID:   &templateChunk.ID,
			TemplateChunkID: &templateChunk.ID,
			IndentLevel:     1,
			SequenceNumber:  &i,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}
		slotChunks = append(slotChunks, slotChunk)
	}
	
	// Insert slot chunks
	if len(slotChunks) > 0 {
		err = c.InsertChunks(ctx, slotChunks)
		if err != nil {
			return nil, fmt.Errorf("failed to create slot chunks: %w", err)
		}
	}
	
	// Create template_slots relationships
	for i, slotChunk := range slotChunks {
		templateSlot := models.TemplateSlot{
			ID:              generateUUID(),
			TemplateChunkID: templateChunk.ID,
			SlotChunkID:     slotChunk.ID,
			SlotOrder:       i,
			CreatedAt:       time.Now(),
		}
		
		err = c.makeRequest(ctx, "POST", "/template_slots", templateSlot, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create template slot relationship: %w", err)
		}
	}
	
	// Return the complete template structure
	return &models.TemplateWithInstances{
		Template:  templateChunk,
		Slots:     slotChunks,
		Instances: []models.TemplateInstance{}, // No instances yet
	}, nil
}

// GetTemplateByContent retrieves a template by its content with all instances
func (c *supabaseHTTPClient) GetTemplateByContent(ctx context.Context, templateContent string) (*models.TemplateWithInstances, error) {
	// Find the template chunk by content
	templateChunk, err := c.GetChunkByContent(ctx, templateContent)
	if err != nil {
		return nil, fmt.Errorf("template not found: %w", err)
	}
	
	if !templateChunk.IsTemplate {
		return nil, fmt.Errorf("chunk is not a template: %s", templateContent)
	}
	
	// Get template slots
	slots, err := c.getTemplateSlots(ctx, templateChunk.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get template slots: %w", err)
	}
	
	// Get template instances
	instances, err := c.GetTemplateInstances(ctx, templateChunk.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get template instances: %w", err)
	}
	
	return &models.TemplateWithInstances{
		Template:  templateChunk,
		Slots:     slots,
		Instances: instances,
	}, nil
}

// GetAllTemplates retrieves all templates in the system
func (c *supabaseHTTPClient) GetAllTemplates(ctx context.Context) ([]models.TemplateWithInstances, error) {
	// Get all template chunks
	params := map[string]string{
		"select":      "*",
		"is_template": "eq.true",
		"order":       "created_at.desc",
	}
	endpoint := "/chunks" + buildQueryParams(params)
	
	var templateChunks []models.ChunkRecord
	err := c.makeRequest(ctx, "GET", endpoint, nil, &templateChunks)
	if err != nil {
		return nil, fmt.Errorf("failed to get template chunks: %w", err)
	}
	
	// For each template, get its slots and instances
	var templates []models.TemplateWithInstances
	for _, templateChunk := range templateChunks {
		slots, err := c.getTemplateSlots(ctx, templateChunk.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get slots for template %s: %w", templateChunk.ID, err)
		}
		
		instances, err := c.GetTemplateInstances(ctx, templateChunk.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get instances for template %s: %w", templateChunk.ID, err)
		}
		
		templates = append(templates, models.TemplateWithInstances{
			Template:  &templateChunk,
			Slots:     slots,
			Instances: instances,
		})
	}
	
	return templates, nil
}

// getTemplateSlots retrieves slots for a specific template
func (c *supabaseHTTPClient) getTemplateSlots(ctx context.Context, templateChunkID string) ([]models.ChunkRecord, error) {
	// Get template slot relationships
	params := map[string]string{
		"select":            "slot_chunk_id",
		"template_chunk_id": "eq." + templateChunkID,
		"order":             "slot_order.asc",
	}
	endpoint := "/template_slots" + buildQueryParams(params)
	
	var slotRelations []map[string]interface{}
	err := c.makeRequest(ctx, "GET", endpoint, nil, &slotRelations)
	if err != nil {
		return nil, fmt.Errorf("failed to get template slot relationships: %w", err)
	}
	
	// Get the actual slot chunks
	var slots []models.ChunkRecord
	for _, relation := range slotRelations {
		if slotChunkID, ok := relation["slot_chunk_id"].(string); ok {
			slotChunk, err := c.GetChunkByID(ctx, slotChunkID)
			if err != nil {
				return nil, fmt.Errorf("failed to get slot chunk %s: %w", slotChunkID, err)
			}
			slots = append(slots, *slotChunk)
		}
	}
	
	return slots, nil
}

// CreateTemplateInstance creates a new instance of a template
func (c *supabaseHTTPClient) CreateTemplateInstance(ctx context.Context, req *models.CreateInstanceRequest) (*models.TemplateInstance, error) {
	// Get the template chunk
	templateChunk, err := c.GetChunkByID(ctx, req.TemplateChunkID)
	if err != nil {
		return nil, fmt.Errorf("failed to get template chunk: %w", err)
	}
	
	if !templateChunk.IsTemplate {
		return nil, fmt.Errorf("chunk is not a template: %s", req.TemplateChunkID)
	}
	
	// Create the instance chunk
	instanceChunk := &models.ChunkRecord{
		ID:              generateUUID(),
		TextID:          templateChunk.TextID, // Same text as template for now
		Content:         req.InstanceName + "#" + strings.TrimSuffix(templateChunk.Content, "#template"),
		IsTemplate:      false,
		IsSlot:          false,
		TemplateChunkID: &templateChunk.ID,
		IndentLevel:     templateChunk.IndentLevel,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
	
	err = c.InsertChunk(ctx, instanceChunk)
	if err != nil {
		return nil, fmt.Errorf("failed to create instance chunk: %w", err)
	}
	
	// Get template slots
	slots, err := c.getTemplateSlots(ctx, templateChunk.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get template slots: %w", err)
	}
	
	// Create slot value chunks for the instance
	slotValues := make(map[string]*models.ChunkRecord)
	for i, slot := range slots {
		slotName := strings.TrimPrefix(slot.Content, "#")
		
		// Get the value for this slot from the request
		value, hasValue := req.SlotValues[slotName]
		if !hasValue {
			value = "" // Empty value if not provided
		}
		
		// Create slot value chunk
		slotValueChunk := &models.ChunkRecord{
			ID:              generateUUID(),
			TextID:          templateChunk.TextID,
			Content:         value,
			IsTemplate:      false,
			IsSlot:          false,
			ParentChunkID:   &instanceChunk.ID,
			TemplateChunkID: &templateChunk.ID,
			SlotValue:       &value,
			IndentLevel:     slot.IndentLevel,
			SequenceNumber:  &i,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}
		
		err = c.InsertChunk(ctx, slotValueChunk)
		if err != nil {
			return nil, fmt.Errorf("failed to create slot value chunk for %s: %w", slotName, err)
		}
		
		slotValues[slotName] = slotValueChunk
	}
	
	return &models.TemplateInstance{
		Instance:   instanceChunk,
		SlotValues: slotValues,
	}, nil
}

// GetTemplateInstances retrieves all instances of a specific template
func (c *supabaseHTTPClient) GetTemplateInstances(ctx context.Context, templateChunkID string) ([]models.TemplateInstance, error) {
	// Get all chunks that reference this template
	params := map[string]string{
		"select":            "*",
		"template_chunk_id": "eq." + templateChunkID,
		"is_template":       "eq.false",
		"is_slot":           "eq.false",
		"order":             "created_at.desc",
	}
	endpoint := "/chunks" + buildQueryParams(params)
	
	var instanceChunks []models.ChunkRecord
	err := c.makeRequest(ctx, "GET", endpoint, nil, &instanceChunks)
	if err != nil {
		return nil, fmt.Errorf("failed to get template instances: %w", err)
	}
	
	// For each instance, get its slot values
	var instances []models.TemplateInstance
	for _, instanceChunk := range instanceChunks {
		// Skip slot value chunks (they have parent_chunk_id set to instance)
		if instanceChunk.ParentChunkID != nil {
			continue
		}
		
		slotValues, err := c.getInstanceSlotValues(ctx, instanceChunk.ID, templateChunkID)
		if err != nil {
			return nil, fmt.Errorf("failed to get slot values for instance %s: %w", instanceChunk.ID, err)
		}
		
		instances = append(instances, models.TemplateInstance{
			Instance:   &instanceChunk,
			SlotValues: slotValues,
		})
	}
	
	return instances, nil
}

// getInstanceSlotValues retrieves slot values for a specific template instance
func (c *supabaseHTTPClient) getInstanceSlotValues(ctx context.Context, instanceChunkID, templateChunkID string) (map[string]*models.ChunkRecord, error) {
	// Get slot value chunks for this instance
	params := map[string]string{
		"select":            "*",
		"parent_chunk_id":   "eq." + instanceChunkID,
		"template_chunk_id": "eq." + templateChunkID,
		"order":             "sequence_number.asc",
	}
	endpoint := "/chunks" + buildQueryParams(params)
	
	var slotValueChunks []models.ChunkRecord
	err := c.makeRequest(ctx, "GET", endpoint, nil, &slotValueChunks)
	if err != nil {
		return nil, fmt.Errorf("failed to get slot value chunks: %w", err)
	}
	
	// Get template slots to map slot names
	templateSlots, err := c.getTemplateSlots(ctx, templateChunkID)
	if err != nil {
		return nil, fmt.Errorf("failed to get template slots: %w", err)
	}
	
	// Create mapping from sequence number to slot name
	slotNames := make(map[int]string)
	for i, slot := range templateSlots {
		slotName := strings.TrimPrefix(slot.Content, "#")
		slotNames[i] = slotName
	}
	
	// Build slot values map
	slotValues := make(map[string]*models.ChunkRecord)
	for _, slotValueChunk := range slotValueChunks {
		if slotValueChunk.SequenceNumber != nil {
			if slotName, exists := slotNames[*slotValueChunk.SequenceNumber]; exists {
				slotValues[slotName] = &slotValueChunk
			}
		}
	}
	
	return slotValues, nil
}

// UpdateSlotValue updates the value of a specific slot in a template instance
func (c *supabaseHTTPClient) UpdateSlotValue(ctx context.Context, instanceChunkID, slotName, value string) error {
	// Get the instance chunk to verify it exists and get template info
	instanceChunk, err := c.GetChunkByID(ctx, instanceChunkID)
	if err != nil {
		return fmt.Errorf("failed to get instance chunk: %w", err)
	}
	
	if instanceChunk.TemplateChunkID == nil {
		return fmt.Errorf("chunk is not a template instance: %s", instanceChunkID)
	}
	
	// Get template slots to find the slot sequence number
	templateSlots, err := c.getTemplateSlots(ctx, *instanceChunk.TemplateChunkID)
	if err != nil {
		return fmt.Errorf("failed to get template slots: %w", err)
	}
	
	// Find the slot sequence number
	var targetSequenceNumber *int
	for i, slot := range templateSlots {
		if strings.TrimPrefix(slot.Content, "#") == slotName {
			targetSequenceNumber = &i
			break
		}
	}
	
	if targetSequenceNumber == nil {
		return fmt.Errorf("slot not found in template: %s", slotName)
	}
	
	// Find the existing slot value chunk
	params := map[string]string{
		"select":           "*",
		"parent_chunk_id":  "eq." + instanceChunkID,
		"sequence_number":  "eq." + strconv.Itoa(*targetSequenceNumber),
		"limit":            "1",
	}
	endpoint := "/chunks" + buildQueryParams(params)
	
	var slotValueChunks []models.ChunkRecord
	err = c.makeRequest(ctx, "GET", endpoint, nil, &slotValueChunks)
	if err != nil {
		return fmt.Errorf("failed to find slot value chunk: %w", err)
	}
	
	if len(slotValueChunks) == 0 {
		return fmt.Errorf("slot value chunk not found for slot: %s", slotName)
	}
	
	// Update the slot value
	slotValueChunk := &slotValueChunks[0]
	slotValueChunk.Content = value
	slotValueChunk.SlotValue = &value
	
	err = c.UpdateChunk(ctx, slotValueChunk)
	if err != nil {
		return fmt.Errorf("failed to update slot value: %w", err)
	}
	
	return nil
}

// Note: The new graph methods are implemented above in the main implementation section

// AddTag adds a tag to a chunk by creating or finding a tag chunk and establishing the relationship
func (c *supabaseHTTPClient) AddTag(ctx context.Context, chunkID string, tagContent string) error {
	// First, try to find an existing chunk with the tag content
	var tagChunk *models.ChunkRecord
	existingTag, err := c.GetChunkByContent(ctx, tagContent)
	if err != nil {
		// If tag doesn't exist, create a new chunk for the tag
		tagChunk = &models.ChunkRecord{
			ID:          generateUUID(),
			Content:     tagContent,
			IsTemplate:  false,
			IsSlot:      false,
			IndentLevel: 0,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		
		// We need to associate it with a text - for now, get the text from the target chunk
		targetChunk, err := c.GetChunkByID(ctx, chunkID)
		if err != nil {
			return fmt.Errorf("failed to get target chunk: %w", err)
		}
		tagChunk.TextID = targetChunk.TextID
		
		err = c.InsertChunk(ctx, tagChunk)
		if err != nil {
			return fmt.Errorf("failed to create tag chunk: %w", err)
		}
	} else {
		tagChunk = existingTag
	}
	
	// Create the tag relationship
	tagRelation := models.ChunkTag{
		ID:         generateUUID(),
		ChunkID:    chunkID,
		TagChunkID: tagChunk.ID,
		CreatedAt:  time.Now(),
	}
	
	err = c.makeRequest(ctx, "POST", "/chunk_tags", tagRelation, nil)
	if err != nil {
		return fmt.Errorf("failed to create tag relationship: %w", err)
	}
	
	return nil
}

// RemoveTag removes a tag relationship between a chunk and a tag chunk
func (c *supabaseHTTPClient) RemoveTag(ctx context.Context, chunkID string, tagChunkID string) error {
	params := map[string]string{
		"chunk_id":     "eq." + chunkID,
		"tag_chunk_id": "eq." + tagChunkID,
	}
	endpoint := "/chunk_tags" + buildQueryParams(params)
	
	err := c.makeRequest(ctx, "DELETE", endpoint, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to remove tag relationship: %w", err)
	}
	
	return nil
}

// GetChunkTags retrieves all tag chunks associated with a specific chunk
func (c *supabaseHTTPClient) GetChunkTags(ctx context.Context, chunkID string) ([]models.ChunkRecord, error) {
	// First get the tag relationships
	params := map[string]string{
		"select":   "tag_chunk_id",
		"chunk_id": "eq." + chunkID,
	}
	endpoint := "/chunk_tags" + buildQueryParams(params)
	
	var tagRelations []map[string]interface{}
	err := c.makeRequest(ctx, "GET", endpoint, nil, &tagRelations)
	if err != nil {
		return nil, fmt.Errorf("failed to get tag relationships: %w", err)
	}
	
	// Get the actual tag chunks
	var tags []models.ChunkRecord
	for _, relation := range tagRelations {
		if tagChunkID, ok := relation["tag_chunk_id"].(string); ok {
			tagChunk, err := c.GetChunkByID(ctx, tagChunkID)
			if err != nil {
				// Log error but continue with other tags
				continue
			}
			tags = append(tags, *tagChunk)
		}
	}
	
	return tags, nil
}

// GetChunksByTag retrieves all chunks that have a specific tag content
func (c *supabaseHTTPClient) GetChunksByTag(ctx context.Context, tagContent string) ([]models.ChunkRecord, error) {
	// First find the tag chunk by content
	tagChunk, err := c.GetChunkByContent(ctx, tagContent)
	if err != nil {
		return []models.ChunkRecord{}, nil // Return empty slice if tag doesn't exist
	}
	
	// Get all chunk relationships for this tag
	params := map[string]string{
		"select":       "chunk_id",
		"tag_chunk_id": "eq." + tagChunk.ID,
	}
	endpoint := "/chunk_tags" + buildQueryParams(params)
	
	var tagRelations []map[string]interface{}
	err = c.makeRequest(ctx, "GET", endpoint, nil, &tagRelations)
	if err != nil {
		return nil, fmt.Errorf("failed to get tag relationships: %w", err)
	}
	
	// Get the actual chunks
	var chunks []models.ChunkRecord
	for _, relation := range tagRelations {
		if chunkID, ok := relation["chunk_id"].(string); ok {
			chunk, err := c.GetChunkByID(ctx, chunkID)
			if err != nil {
				// Log error but continue with other chunks
				continue
			}
			chunks = append(chunks, *chunk)
		}
	}
	
	return chunks, nil
}

// GetChunkHierarchy retrieves the complete hierarchical structure starting from a root chunk
func (c *supabaseHTTPClient) GetChunkHierarchy(ctx context.Context, rootChunkID string) (*models.ChunkHierarchy, error) {
	// Get the root chunk
	rootChunk, err := c.GetChunkByID(ctx, rootChunkID)
	if err != nil {
		return nil, fmt.Errorf("failed to get root chunk: %w", err)
	}
	
	// Build hierarchy recursively
	hierarchy, err := c.buildHierarchy(ctx, rootChunk, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to build hierarchy: %w", err)
	}
	
	return hierarchy, nil
}

// buildHierarchy recursively builds the chunk hierarchy
func (c *supabaseHTTPClient) buildHierarchy(ctx context.Context, chunk *models.ChunkRecord, level int) (*models.ChunkHierarchy, error) {
	hierarchy := &models.ChunkHierarchy{
		Chunk: chunk,
		Level: level,
	}
	
	// Get children chunks
	children, err := c.GetChildrenChunks(ctx, chunk.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get children for chunk %s: %w", chunk.ID, err)
	}
	
	// Recursively build children hierarchies
	for _, child := range children {
		childHierarchy, err := c.buildHierarchy(ctx, &child, level+1)
		if err != nil {
			return nil, err
		}
		hierarchy.Children = append(hierarchy.Children, *childHierarchy)
	}
	
	return hierarchy, nil
}

// GetChildrenChunks retrieves direct children of a parent chunk
func (c *supabaseHTTPClient) GetChildrenChunks(ctx context.Context, parentChunkID string) ([]models.ChunkRecord, error) {
	params := map[string]string{
		"select":           "*",
		"parent_chunk_id":  "eq." + parentChunkID,
		"order":            "sequence_number.asc,created_at.asc",
	}
	endpoint := "/chunks" + buildQueryParams(params)
	
	var chunks []models.ChunkRecord
	err := c.makeRequest(ctx, "GET", endpoint, nil, &chunks)
	if err != nil {
		return nil, fmt.Errorf("failed to get children chunks: %w", err)
	}
	
	return chunks, nil
}

// GetSiblingChunks retrieves chunks at the same level as the given chunk
func (c *supabaseHTTPClient) GetSiblingChunks(ctx context.Context, chunkID string) ([]models.ChunkRecord, error) {
	// First get the chunk to find its parent
	chunk, err := c.GetChunkByID(ctx, chunkID)
	if err != nil {
		return nil, fmt.Errorf("failed to get chunk: %w", err)
	}
	
	var params map[string]string
	if chunk.ParentChunkID != nil {
		// Get siblings with the same parent
		params = map[string]string{
			"select":          "*",
			"parent_chunk_id": "eq." + *chunk.ParentChunkID,
			"id":              "neq." + chunkID, // Exclude the chunk itself
			"order":           "sequence_number.asc,created_at.asc",
		}
	} else {
		// Get root-level siblings (chunks with no parent in the same text)
		params = map[string]string{
			"select":          "*",
			"text_id":         "eq." + chunk.TextID,
			"parent_chunk_id": "is.null",
			"id":              "neq." + chunkID,
			"order":           "sequence_number.asc,created_at.asc",
		}
	}
	
	endpoint := "/chunks" + buildQueryParams(params)
	
	var chunks []models.ChunkRecord
	err = c.makeRequest(ctx, "GET", endpoint, nil, &chunks)
	if err != nil {
		return nil, fmt.Errorf("failed to get sibling chunks: %w", err)
	}
	
	return chunks, nil
}

// MoveChunk moves a chunk to a new position in the hierarchy
func (c *supabaseHTTPClient) MoveChunk(ctx context.Context, req *models.MoveChunkRequest) error {
	// Get the chunk to move
	chunk, err := c.GetChunkByID(ctx, req.ChunkID)
	if err != nil {
		return fmt.Errorf("failed to get chunk to move: %w", err)
	}
	
	// Update chunk with new position
	chunk.ParentChunkID = req.NewParentID
	chunk.IndentLevel = req.NewIndentLevel
	
	// If sequence number is provided, update it
	if req.NewPosition >= 0 {
		chunk.SequenceNumber = &req.NewPosition
	}
	
	// Update the chunk
	err = c.UpdateChunk(ctx, chunk)
	if err != nil {
		return fmt.Errorf("failed to update chunk position: %w", err)
	}
	
	// Reorder siblings if necessary
	err = c.reorderSiblings(ctx, chunk)
	if err != nil {
		return fmt.Errorf("failed to reorder siblings: %w", err)
	}
	
	return nil
}

// reorderSiblings ensures proper sequence numbering for siblings
func (c *supabaseHTTPClient) reorderSiblings(ctx context.Context, movedChunk *models.ChunkRecord) error {
	// Get all siblings including the moved chunk
	var siblings []models.ChunkRecord
	var err error
	
	if movedChunk.ParentChunkID != nil {
		siblings, err = c.GetChildrenChunks(ctx, *movedChunk.ParentChunkID)
	} else {
		// Get root-level chunks for the same text
		params := map[string]string{
			"select":          "*",
			"text_id":         "eq." + movedChunk.TextID,
			"parent_chunk_id": "is.null",
			"order":           "sequence_number.asc,created_at.asc",
		}
		endpoint := "/chunks" + buildQueryParams(params)
		err = c.makeRequest(ctx, "GET", endpoint, nil, &siblings)
	}
	
	if err != nil {
		return fmt.Errorf("failed to get siblings for reordering: %w", err)
	}
	
	// Update sequence numbers
	for i, sibling := range siblings {
		newSeq := i
		if sibling.SequenceNumber == nil || *sibling.SequenceNumber != newSeq {
			sibling.SequenceNumber = &newSeq
			if err := c.UpdateChunk(ctx, &sibling); err != nil {
				return fmt.Errorf("failed to update sibling sequence: %w", err)
			}
		}
	}
	
	return nil
}

// BulkUpdateChunks performs batch updates on multiple chunks
func (c *supabaseHTTPClient) BulkUpdateChunks(ctx context.Context, req *models.BulkUpdateRequest) error {
	for _, update := range req.Updates {
		// Get the chunk to update
		chunk, err := c.GetChunkByID(ctx, update.ChunkID)
		if err != nil {
			return fmt.Errorf("failed to get chunk %s: %w", update.ChunkID, err)
		}
		
		// Apply updates
		if update.Content != nil {
			chunk.Content = *update.Content
		}
		if update.ParentChunkID != nil {
			chunk.ParentChunkID = update.ParentChunkID
		}
		if update.SequenceNumber != nil {
			chunk.SequenceNumber = update.SequenceNumber
		}
		if update.IndentLevel != nil {
			chunk.IndentLevel = *update.IndentLevel
		}
		
		// Update the chunk
		err = c.UpdateChunk(ctx, chunk)
		if err != nil {
			return fmt.Errorf("failed to update chunk %s: %w", update.ChunkID, err)
		}
	}
	
	return nil
}

// SearchChunks performs text-based search on chunks with optional filters
func (c *supabaseHTTPClient) SearchChunks(ctx context.Context, query string, filters map[string]interface{}) ([]models.ChunkRecord, error) {
	if query == "" {
		return []models.ChunkRecord{}, nil
	}
	
	// Build query parameters
	params := map[string]string{
		"select": "*",
		"order":  "created_at.desc",
	}
	
	// Add text search using PostgreSQL full-text search
	params["content"] = "ilike.*" + query + "*"
	
	// Apply filters
	if filters != nil {
		if textID, ok := filters["text_id"].(string); ok && textID != "" {
			params["text_id"] = "eq." + textID
		}
		if isTemplate, ok := filters["is_template"].(bool); ok {
			params["is_template"] = "eq." + strconv.FormatBool(isTemplate)
		}
		if isSlot, ok := filters["is_slot"].(bool); ok {
			params["is_slot"] = "eq." + strconv.FormatBool(isSlot)
		}
		if minIndentLevel, ok := filters["min_indent_level"].(int); ok {
			params["indent_level"] = "gte." + strconv.Itoa(minIndentLevel)
		}
		if maxIndentLevel, ok := filters["max_indent_level"].(int); ok {
			params["indent_level"] = "lte." + strconv.Itoa(maxIndentLevel)
		}
		if limit, ok := filters["limit"].(int); ok && limit > 0 {
			params["limit"] = strconv.Itoa(limit)
		}
	}
	
	// Set default limit if not specified
	if _, hasLimit := params["limit"]; !hasLimit {
		params["limit"] = "100" // Default search limit
	}
	
	endpoint := "/chunks" + buildQueryParams(params)
	
	var chunks []models.ChunkRecord
	err := c.makeRequest(ctx, "GET", endpoint, nil, &chunks)
	if err != nil {
		return nil, fmt.Errorf("failed to search chunks: %w", err)
	}
	
	return chunks, nil
}

// SearchByTag retrieves chunks with their associated tags based on tag content
func (c *supabaseHTTPClient) SearchByTag(ctx context.Context, tagContent string) ([]models.ChunkWithTags, error) {
	// Get chunks that have the specified tag
	chunks, err := c.GetChunksByTag(ctx, tagContent)
	if err != nil {
		return nil, fmt.Errorf("failed to get chunks by tag: %w", err)
	}
	
	// For each chunk, get all its tags
	var results []models.ChunkWithTags
	for _, chunk := range chunks {
		tags, err := c.GetChunkTags(ctx, chunk.ID)
		if err != nil {
			// Log error but continue with other chunks
			tags = []models.ChunkRecord{}
		}
		
		results = append(results, models.ChunkWithTags{
			Chunk: &chunk,
			Tags:  tags,
		})
	}
	
	return results, nil
}

// InsertEmbeddings stores vector embeddings in Supabase
func (c *supabaseHTTPClient) InsertEmbeddings(ctx context.Context, embeddings []models.EmbeddingRecord) error {
	if len(embeddings) == 0 {
		return nil
	}
	
	// Prepare embeddings with required fields
	for i := range embeddings {
		if embeddings[i].ID == "" {
			embeddings[i].ID = generateUUID()
		}
		if embeddings[i].CreatedAt.IsZero() {
			embeddings[i].CreatedAt = time.Now()
		}
	}
	
	var result []models.EmbeddingRecord
	err := c.makeRequest(ctx, "POST", "/embeddings", embeddings, &result)
	if err != nil {
		return fmt.Errorf("failed to insert embeddings: %w", err)
	}
	
	// Update embeddings with returned data
	if len(result) == len(embeddings) {
		copy(embeddings, result)
	}
	
	return nil
}

// SearchSimilar performs vector similarity search using PGVector
func (c *supabaseHTTPClient) SearchSimilar(ctx context.Context, queryVector []float64, limit int) ([]models.SimilarityResult, error) {
	if limit <= 0 {
		limit = 10 // Default limit
	}
	
	// Convert vector to string format for Supabase RPC call
	vectorStr, err := vectorToString(queryVector)
	if err != nil {
		return nil, fmt.Errorf("failed to convert vector: %w", err)
	}
	
	// Use Supabase RPC function for similarity search
	rpcRequest := map[string]interface{}{
		"query_embedding": vectorStr,
		"match_threshold": 0.0, // Minimum similarity threshold
		"match_count":     limit,
	}
	
	var rpcResult []map[string]interface{}
	err = c.makeRequest(ctx, "POST", "/rpc/match_chunks", rpcRequest, &rpcResult)
	if err != nil {
		return nil, fmt.Errorf("failed to execute similarity search: %w", err)
	}
	
	// Convert results to SimilarityResult
	var results []models.SimilarityResult
	for _, row := range rpcResult {
		// Extract chunk data
		chunkData, ok := row["chunk"].(map[string]interface{})
		if !ok {
			continue
		}
		
		chunk, err := mapToChunkRecord(chunkData)
		if err != nil {
			continue // Skip invalid chunks
		}
		
		// Extract similarity score
		similarity, ok := row["similarity"].(float64)
		if !ok {
			similarity = 0.0
		}
		
		results = append(results, models.SimilarityResult{
			Chunk:      *chunk,
			Similarity: similarity,
		})
	}
	
	return results, nil
}

// vectorToString converts float64 slice to PostgreSQL vector format
func vectorToString(vector []float64) (string, error) {
	if len(vector) == 0 {
		return "[]", nil
	}
	
	// Convert to JSON array format
	jsonData, err := json.Marshal(vector)
	if err != nil {
		return "", fmt.Errorf("failed to marshal vector: %w", err)
	}
	
	return string(jsonData), nil
}

// mapToChunkRecord converts map to ChunkRecord
func mapToChunkRecord(data map[string]interface{}) (*models.ChunkRecord, error) {
	// Convert map to JSON and back to struct for type safety
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal chunk data: %w", err)
	}
	
	var chunk models.ChunkRecord
	err = json.Unmarshal(jsonData, &chunk)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal chunk data: %w", err)
	}
	
	return &chunk, nil
}

// InsertGraphNodes stores knowledge graph nodes in Supabase
func (c *supabaseHTTPClient) InsertGraphNodes(ctx context.Context, nodes []models.GraphNode) error {
	if len(nodes) == 0 {
		return nil
	}
	
	// Prepare nodes with required fields
	for i := range nodes {
		if nodes[i].ID == "" {
			nodes[i].ID = generateUUID()
		}
		if nodes[i].CreatedAt.IsZero() {
			nodes[i].CreatedAt = time.Now()
		}
		if nodes[i].Properties == nil {
			nodes[i].Properties = make(map[string]interface{})
		}
	}
	
	var result []models.GraphNode
	err := c.makeRequest(ctx, "POST", "/graph_nodes", nodes, &result)
	if err != nil {
		return fmt.Errorf("failed to insert graph nodes: %w", err)
	}
	
	// Update nodes with returned data
	if len(result) == len(nodes) {
		copy(nodes, result)
	}
	
	return nil
}

// InsertGraphEdges stores knowledge graph edges in Supabase
func (c *supabaseHTTPClient) InsertGraphEdges(ctx context.Context, edges []models.GraphEdge) error {
	if len(edges) == 0 {
		return nil
	}
	
	// Prepare edges with required fields
	for i := range edges {
		if edges[i].ID == "" {
			edges[i].ID = generateUUID()
		}
		if edges[i].CreatedAt.IsZero() {
			edges[i].CreatedAt = time.Now()
		}
		if edges[i].Properties == nil {
			edges[i].Properties = make(map[string]interface{})
		}
	}
	
	var result []models.GraphEdge
	err := c.makeRequest(ctx, "POST", "/graph_edges", edges, &result)
	if err != nil {
		return fmt.Errorf("failed to insert graph edges: %w", err)
	}
	
	// Update edges with returned data
	if len(result) == len(edges) {
		copy(edges, result)
	}
	
	return nil
}

// SearchGraph performs graph traversal and search using Apache AGE through Supabase
func (c *supabaseHTTPClient) SearchGraph(ctx context.Context, query *models.GraphQuery) (*models.GraphResult, error) {
	if query == nil {
		return nil, fmt.Errorf("graph query cannot be nil")
	}
	
	// Set default values
	if query.MaxDepth <= 0 {
		query.MaxDepth = 3 // Default traversal depth
	}
	if query.Limit <= 0 {
		query.Limit = 50 // Default result limit
	}
	
	// Try RPC function first, fallback to manual traversal if not available
	rpcRequest := map[string]interface{}{
		"entity_name":  query.EntityName,
		"max_depth":    query.MaxDepth,
		"result_limit": query.Limit,
	}
	
	var rpcResult map[string]interface{}
	err := c.makeRequest(ctx, "POST", "/rpc/search_graph", rpcRequest, &rpcResult)
	if err != nil {
		// Fallback to manual graph traversal if RPC function doesn't exist
		return c.manualGraphSearch(ctx, query)
	}
	
	// Parse nodes from result
	var nodes []models.GraphNode
	if nodesData, ok := rpcResult["nodes"].([]interface{}); ok {
		for _, nodeData := range nodesData {
			if nodeMap, ok := nodeData.(map[string]interface{}); ok {
				node, err := mapToGraphNode(nodeMap)
				if err != nil {
					continue // Skip invalid nodes
				}
				nodes = append(nodes, *node)
			}
		}
	}
	
	// Parse edges from result
	var edges []models.GraphEdge
	if edgesData, ok := rpcResult["edges"].([]interface{}); ok {
		for _, edgeData := range edgesData {
			if edgeMap, ok := edgeData.(map[string]interface{}); ok {
				edge, err := mapToGraphEdge(edgeMap)
				if err != nil {
					continue // Skip invalid edges
				}
				edges = append(edges, *edge)
			}
		}
	}
	
	return &models.GraphResult{
		Nodes: nodes,
		Edges: edges,
	}, nil
}

// manualGraphSearch performs graph search using manual traversal when RPC is not available
func (c *supabaseHTTPClient) manualGraphSearch(ctx context.Context, query *models.GraphQuery) (*models.GraphResult, error) {
	// Find starting nodes by entity name
	startNodes, err := c.GetNodesByEntity(ctx, query.EntityName)
	if err != nil {
		return nil, fmt.Errorf("failed to find starting nodes: %w", err)
	}
	
	if len(startNodes) == 0 {
		return &models.GraphResult{
			Nodes: []models.GraphNode{},
			Edges: []models.GraphEdge{},
		}, nil
	}
	
	// Perform breadth-first traversal from starting nodes
	visited := make(map[string]bool)
	var allNodes []models.GraphNode
	var allEdges []models.GraphEdge
	
	// Use a queue for BFS
	queue := make([]models.GraphNode, 0)
	depthMap := make(map[string]int)
	
	// Initialize with starting nodes
	for _, node := range startNodes {
		queue = append(queue, node)
		depthMap[node.ID] = 0
		visited[node.ID] = true
		allNodes = append(allNodes, node)
	}
	
	// BFS traversal
	for len(queue) > 0 && len(allNodes) < query.Limit {
		current := queue[0]
		queue = queue[1:]
		
		currentDepth := depthMap[current.ID]
		if currentDepth >= query.MaxDepth {
			continue
		}
		
		// Get neighbors of current node
		neighbors, edges, err := c.getNodeNeighborsWithEdges(ctx, current.ID)
		if err != nil {
			continue // Skip on error, continue with other nodes
		}
		
		// Add edges to result
		for _, edge := range edges {
			allEdges = append(allEdges, edge)
		}
		
		// Add unvisited neighbors to queue
		for _, neighbor := range neighbors {
			if !visited[neighbor.ID] && len(allNodes) < query.Limit {
				visited[neighbor.ID] = true
				depthMap[neighbor.ID] = currentDepth + 1
				queue = append(queue, neighbor)
				allNodes = append(allNodes, neighbor)
			}
		}
	}
	
	return &models.GraphResult{
		Nodes: allNodes,
		Edges: allEdges,
	}, nil
}

// GetNodesByEntity retrieves all nodes with a specific entity name
func (c *supabaseHTTPClient) GetNodesByEntity(ctx context.Context, entityName string) ([]models.GraphNode, error) {
	params := map[string]string{
		"select":      "*",
		"entity_name": "eq." + entityName,
		"order":       "created_at.desc",
	}
	endpoint := "/graph_nodes" + buildQueryParams(params)
	
	var nodes []models.GraphNode
	err := c.makeRequest(ctx, "GET", endpoint, nil, &nodes)
	if err != nil {
		return nil, fmt.Errorf("failed to get nodes by entity: %w", err)
	}
	
	return nodes, nil
}

// GetNodeNeighbors retrieves neighboring nodes within specified depth
func (c *supabaseHTTPClient) GetNodeNeighbors(ctx context.Context, nodeID string, maxDepth int) (*models.GraphResult, error) {
	if maxDepth <= 0 {
		maxDepth = 1
	}
	
	visited := make(map[string]bool)
	var allNodes []models.GraphNode
	var allEdges []models.GraphEdge
	
	// Get the starting node
	startNode, err := c.getNodeByID(ctx, nodeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get starting node: %w", err)
	}
	
	// BFS traversal
	queue := []models.GraphNode{*startNode}
	depthMap := map[string]int{nodeID: 0}
	visited[nodeID] = true
	allNodes = append(allNodes, *startNode)
	
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		
		currentDepth := depthMap[current.ID]
		if currentDepth >= maxDepth {
			continue
		}
		
		neighbors, edges, err := c.getNodeNeighborsWithEdges(ctx, current.ID)
		if err != nil {
			continue
		}
		
		// Add edges
		for _, edge := range edges {
			allEdges = append(allEdges, edge)
		}
		
		// Add unvisited neighbors
		for _, neighbor := range neighbors {
			if !visited[neighbor.ID] {
				visited[neighbor.ID] = true
				depthMap[neighbor.ID] = currentDepth + 1
				queue = append(queue, neighbor)
				allNodes = append(allNodes, neighbor)
			}
		}
	}
	
	return &models.GraphResult{
		Nodes: allNodes,
		Edges: allEdges,
	}, nil
}

// FindPathBetweenNodes finds a path between two nodes using BFS
func (c *supabaseHTTPClient) FindPathBetweenNodes(ctx context.Context, sourceNodeID, targetNodeID string, maxDepth int) (*models.GraphResult, error) {
	if maxDepth <= 0 {
		maxDepth = 5 // Default max depth for path finding
	}
	
	// BFS to find shortest path
	queue := []string{sourceNodeID}
	visited := make(map[string]bool)
	parent := make(map[string]string)
	depth := make(map[string]int)
	
	visited[sourceNodeID] = true
	depth[sourceNodeID] = 0
	
	found := false
	
	for len(queue) > 0 && !found {
		current := queue[0]
		queue = queue[1:]
		
		if depth[current] >= maxDepth {
			continue
		}
		
		// Get neighbors
		neighbors, _, err := c.getNodeNeighborsWithEdges(ctx, current)
		if err != nil {
			continue
		}
		
		for _, neighbor := range neighbors {
			if !visited[neighbor.ID] {
				visited[neighbor.ID] = true
				parent[neighbor.ID] = current
				depth[neighbor.ID] = depth[current] + 1
				queue = append(queue, neighbor.ID)
				
				if neighbor.ID == targetNodeID {
					found = true
					break
				}
			}
		}
	}
	
	if !found {
		return &models.GraphResult{
			Nodes: []models.GraphNode{},
			Edges: []models.GraphEdge{},
		}, nil
	}
	
	// Reconstruct path
	var pathNodes []models.GraphNode
	var pathEdges []models.GraphEdge
	
	// Build path from target back to source
	current := targetNodeID
	for current != "" {
		node, err := c.getNodeByID(ctx, current)
		if err != nil {
			break
		}
		pathNodes = append([]models.GraphNode{*node}, pathNodes...)
		
		if parent[current] != "" {
			// Find edge between parent and current
			edge, err := c.findEdgeBetweenNodes(ctx, parent[current], current)
			if err == nil {
				pathEdges = append([]models.GraphEdge{*edge}, pathEdges...)
			}
		}
		
		current = parent[current]
	}
	
	return &models.GraphResult{
		Nodes: pathNodes,
		Edges: pathEdges,
	}, nil
}

// GetNodesByChunk retrieves all graph nodes associated with a specific chunk
func (c *supabaseHTTPClient) GetNodesByChunk(ctx context.Context, chunkID string) ([]models.GraphNode, error) {
	params := map[string]string{
		"select":   "*",
		"chunk_id": "eq." + chunkID,
		"order":    "created_at.desc",
	}
	endpoint := "/graph_nodes" + buildQueryParams(params)
	
	var nodes []models.GraphNode
	err := c.makeRequest(ctx, "GET", endpoint, nil, &nodes)
	if err != nil {
		return nil, fmt.Errorf("failed to get nodes by chunk: %w", err)
	}
	
	return nodes, nil
}

// GetEdgesByRelationType retrieves all edges of a specific relationship type
func (c *supabaseHTTPClient) GetEdgesByRelationType(ctx context.Context, relationType string) ([]models.GraphEdge, error) {
	params := map[string]string{
		"select":            "*",
		"relationship_type": "eq." + relationType,
		"order":             "created_at.desc",
	}
	endpoint := "/graph_edges" + buildQueryParams(params)
	
	var edges []models.GraphEdge
	err := c.makeRequest(ctx, "GET", endpoint, nil, &edges)
	if err != nil {
		return nil, fmt.Errorf("failed to get edges by relationship type: %w", err)
	}
	
	return edges, nil
}

// Helper methods for graph operations

// getNodeByID retrieves a single node by ID
func (c *supabaseHTTPClient) getNodeByID(ctx context.Context, nodeID string) (*models.GraphNode, error) {
	params := map[string]string{
		"select": "*",
		"id":     "eq." + nodeID,
	}
	endpoint := "/graph_nodes" + buildQueryParams(params)
	
	var nodes []models.GraphNode
	err := c.makeRequest(ctx, "GET", endpoint, nil, &nodes)
	if err != nil {
		return nil, fmt.Errorf("failed to get node by ID: %w", err)
	}
	
	if len(nodes) == 0 {
		return nil, fmt.Errorf("node not found: %s", nodeID)
	}
	
	return &nodes[0], nil
}

// getNodeNeighborsWithEdges retrieves direct neighbors and connecting edges
func (c *supabaseHTTPClient) getNodeNeighborsWithEdges(ctx context.Context, nodeID string) ([]models.GraphNode, []models.GraphEdge, error) {
	// Get all edges connected to this node
	params := map[string]string{
		"select": "*",
		"or":     "(source_node_id.eq." + nodeID + ",target_node_id.eq." + nodeID + ")",
	}
	endpoint := "/graph_edges" + buildQueryParams(params)
	
	var edges []models.GraphEdge
	err := c.makeRequest(ctx, "GET", endpoint, nil, &edges)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get edges for node: %w", err)
	}
	
	// Collect neighbor node IDs
	neighborIDs := make(map[string]bool)
	for _, edge := range edges {
		if edge.SourceNodeID == nodeID {
			neighborIDs[edge.TargetNodeID] = true
		} else if edge.TargetNodeID == nodeID {
			neighborIDs[edge.SourceNodeID] = true
		}
	}
	
	// Get neighbor nodes
	var neighbors []models.GraphNode
	for neighborID := range neighborIDs {
		node, err := c.getNodeByID(ctx, neighborID)
		if err != nil {
			continue // Skip invalid nodes
		}
		neighbors = append(neighbors, *node)
	}
	
	return neighbors, edges, nil
}

// findEdgeBetweenNodes finds an edge between two specific nodes
func (c *supabaseHTTPClient) findEdgeBetweenNodes(ctx context.Context, sourceID, targetID string) (*models.GraphEdge, error) {
	params := map[string]string{
		"select": "*",
		"or":     "(and(source_node_id.eq." + sourceID + ",target_node_id.eq." + targetID + "),and(source_node_id.eq." + targetID + ",target_node_id.eq." + sourceID + "))",
		"limit":  "1",
	}
	endpoint := "/graph_edges" + buildQueryParams(params)
	
	var edges []models.GraphEdge
	err := c.makeRequest(ctx, "GET", endpoint, nil, &edges)
	if err != nil {
		return nil, fmt.Errorf("failed to find edge between nodes: %w", err)
	}
	
	if len(edges) == 0 {
		return nil, fmt.Errorf("no edge found between nodes %s and %s", sourceID, targetID)
	}
	
	return &edges[0], nil
}

// mapToGraphNode converts map to GraphNode
func mapToGraphNode(data map[string]interface{}) (*models.GraphNode, error) {
	// Convert map to JSON and back to struct for type safety
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal graph node data: %w", err)
	}
	
	var node models.GraphNode
	err = json.Unmarshal(jsonData, &node)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal graph node data: %w", err)
	}
	
	return &node, nil
}

// mapToGraphEdge converts map to GraphEdge
func mapToGraphEdge(data map[string]interface{}) (*models.GraphEdge, error) {
	// Convert map to JSON and back to struct for type safety
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal graph edge data: %w", err)
	}
	
	var edge models.GraphEdge
	err = json.Unmarshal(jsonData, &edge)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal graph edge data: %w", err)
	}
	
	return &edge, nil
}