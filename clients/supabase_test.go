package clients

import (
	"context"
	"strings"
	"testing"
	"time"

	"semantic-text-processor/config"
	"semantic-text-processor/models"
)



func (m *MockSupabaseClient) HealthCheck(ctx context.Context) error {
	return nil
}

func (m *MockSupabaseClient) InsertText(ctx context.Context, text *models.TextRecord) error {
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
	
	m.texts[text.ID] = text
	return nil
}

func (m *MockSupabaseClient) GetTexts(ctx context.Context, pagination *models.Pagination) (*models.TextList, error) {
	if pagination == nil {
		pagination = &models.Pagination{Page: 1, PageSize: 20}
	}
	
	var texts []models.TextRecord
	for _, text := range m.texts {
		texts = append(texts, *text)
	}
	
	pagination.Total = len(texts)
	
	return &models.TextList{
		Texts:      texts,
		Pagination: *pagination,
	}, nil
}

func (m *MockSupabaseClient) GetTextByID(ctx context.Context, id string) (*models.TextDetail, error) {
	text, exists := m.texts[id]
	if !exists {
		return nil, &SupabaseError{Code: "404", Message: "Text not found"}
	}
	
	var chunks []models.ChunkRecord
	for _, chunk := range m.chunks {
		if chunk.TextID == id {
			chunks = append(chunks, *chunk)
		}
	}
	
	return &models.TextDetail{
		Text:   *text,
		Chunks: chunks,
	}, nil
}

func (m *MockSupabaseClient) UpdateText(ctx context.Context, text *models.TextRecord) error {
	if _, exists := m.texts[text.ID]; !exists {
		return &SupabaseError{Code: "404", Message: "Text not found"}
	}
	
	text.UpdatedAt = time.Now()
	m.texts[text.ID] = text
	return nil
}

func (m *MockSupabaseClient) DeleteText(ctx context.Context, id string) error {
	if _, exists := m.texts[id]; !exists {
		return &SupabaseError{Code: "404", Message: "Text not found"}
	}
	
	delete(m.texts, id)
	
	// Delete associated chunks
	for chunkID, chunk := range m.chunks {
		if chunk.TextID == id {
			delete(m.chunks, chunkID)
		}
	}
	
	return nil
}

func (m *MockSupabaseClient) InsertChunk(ctx context.Context, chunk *models.ChunkRecord) error {
	if chunk.ID == "" {
		chunk.ID = generateUUID()
	}
	if chunk.CreatedAt.IsZero() {
		chunk.CreatedAt = time.Now()
	}
	if chunk.UpdatedAt.IsZero() {
		chunk.UpdatedAt = time.Now()
	}
	
	m.chunks[chunk.ID] = chunk
	return nil
}

func (m *MockSupabaseClient) InsertChunks(ctx context.Context, chunks []models.ChunkRecord) error {
	for i := range chunks {
		if err := m.InsertChunk(ctx, &chunks[i]); err != nil {
			return err
		}
	}
	return nil
}

func (m *MockSupabaseClient) GetChunkByID(ctx context.Context, id string) (*models.ChunkRecord, error) {
	chunk, exists := m.chunks[id]
	if !exists {
		return nil, &SupabaseError{Code: "404", Message: "Chunk not found"}
	}
	return chunk, nil
}

func (m *MockSupabaseClient) GetChunkByContent(ctx context.Context, content string) (*models.ChunkRecord, error) {
	for _, chunk := range m.chunks {
		if chunk.Content == content {
			return chunk, nil
		}
	}
	return nil, &SupabaseError{Code: "404", Message: "Chunk not found"}
}

func (m *MockSupabaseClient) UpdateChunk(ctx context.Context, chunk *models.ChunkRecord) error {
	if _, exists := m.chunks[chunk.ID]; !exists {
		return &SupabaseError{Code: "404", Message: "Chunk not found"}
	}
	
	chunk.UpdatedAt = time.Now()
	m.chunks[chunk.ID] = chunk
	return nil
}

func (m *MockSupabaseClient) DeleteChunk(ctx context.Context, id string) error {
	if _, exists := m.chunks[id]; !exists {
		return &SupabaseError{Code: "404", Message: "Chunk not found"}
	}
	
	delete(m.chunks, id)
	return nil
}

func (m *MockSupabaseClient) GetChunksByTextID(ctx context.Context, textID string) ([]models.ChunkRecord, error) {
	var chunks []models.ChunkRecord
	for _, chunk := range m.chunks {
		if chunk.TextID == textID {
			chunks = append(chunks, *chunk)
		}
	}
	return chunks, nil
}

// Template system implementations for mock
func (m *MockSupabaseClient) CreateTemplate(ctx context.Context, templateName string, slotNames []string) (*models.TemplateWithInstances, error) {
	templateID := "template-" + templateName
	templateContent := templateName + "#template"
	
	// Create template chunk
	templateChunk := &models.ChunkRecord{
		ID:         templateID,
		Content:    templateContent,
		IsTemplate: true,
		IsSlot:     false,
	}
	
	// Create slot chunks
	var slots []models.ChunkRecord
	for i, slotName := range slotNames {
		slotID := templateID + "-slot-" + slotName
		seqNum := i
		slot := models.ChunkRecord{
			ID:              slotID,
			Content:         "#" + slotName,
			IsTemplate:      false,
			IsSlot:          true,
			ParentChunkID:   &templateID,
			TemplateChunkID: &templateID,
			SequenceNumber:  &seqNum,
		}
		slots = append(slots, slot)
		m.chunks[slotID] = &slot
	}
	
	template := &models.TemplateWithInstances{
		Template:  templateChunk,
		Slots:     slots,
		Instances: []models.TemplateInstance{},
	}
	
	m.chunks[templateID] = templateChunk
	
	return template, nil
}

func (m *MockSupabaseClient) GetTemplateByContent(ctx context.Context, templateContent string) (*models.TemplateWithInstances, error) {
	// Find template chunk by content
	for _, chunk := range m.chunks {
		if chunk.Content == templateContent && chunk.IsTemplate {
			// Get slots for this template
			var slots []models.ChunkRecord
			for _, slotChunk := range m.chunks {
				if slotChunk.TemplateChunkID != nil && *slotChunk.TemplateChunkID == chunk.ID && slotChunk.IsSlot {
					slots = append(slots, *slotChunk)
				}
			}
			
			// Get instances for this template
			var instances []models.TemplateInstance
			for _, instanceChunk := range m.chunks {
				if instanceChunk.TemplateChunkID != nil && *instanceChunk.TemplateChunkID == chunk.ID && 
				   !instanceChunk.IsTemplate && !instanceChunk.IsSlot && instanceChunk.ParentChunkID == nil {
					
					// Get slot values for this instance
					slotValues := make(map[string]*models.ChunkRecord)
					for _, slotValueChunk := range m.chunks {
						if slotValueChunk.ParentChunkID != nil && *slotValueChunk.ParentChunkID == instanceChunk.ID {
							// Extract slot name from sequence number
							if slotValueChunk.SequenceNumber != nil {
								for _, slot := range slots {
									if slot.SequenceNumber != nil && *slot.SequenceNumber == *slotValueChunk.SequenceNumber {
										slotName := strings.TrimPrefix(slot.Content, "#")
										slotValues[slotName] = slotValueChunk
										break
									}
								}
							}
						}
					}
					
					instances = append(instances, models.TemplateInstance{
						Instance:   instanceChunk,
						SlotValues: slotValues,
					})
				}
			}
			
			return &models.TemplateWithInstances{
				Template:  chunk,
				Slots:     slots,
				Instances: instances,
			}, nil
		}
	}
	
	return nil, &SupabaseError{Code: "404", Message: "Template not found"}
}

func (m *MockSupabaseClient) GetAllTemplates(ctx context.Context) ([]models.TemplateWithInstances, error) {
	var templates []models.TemplateWithInstances
	
	for _, chunk := range m.chunks {
		if chunk.IsTemplate {
			template, err := m.GetTemplateByContent(ctx, chunk.Content)
			if err == nil {
				templates = append(templates, *template)
			}
		}
	}
	
	return templates, nil
}

func (m *MockSupabaseClient) CreateTemplateInstance(ctx context.Context, req *models.CreateInstanceRequest) (*models.TemplateInstance, error) {
	// Check if template exists
	templateChunk, exists := m.chunks[req.TemplateChunkID]
	if !exists || !templateChunk.IsTemplate {
		return nil, &SupabaseError{Code: "404", Message: "Template not found"}
	}
	
	instanceID := req.TemplateChunkID + "-instance-" + req.InstanceName
	
	instanceChunk := &models.ChunkRecord{
		ID:              instanceID,
		Content:         req.InstanceName,
		TemplateChunkID: &req.TemplateChunkID,
		IsTemplate:      false,
		IsSlot:          false,
	}
	
	// Get template slots to create slot values for all slots
	templateSlots := make([]models.ChunkRecord, 0)
	for _, chunk := range m.chunks {
		if chunk.TemplateChunkID != nil && *chunk.TemplateChunkID == req.TemplateChunkID && chunk.IsSlot {
			templateSlots = append(templateSlots, *chunk)
		}
	}

	slotValues := make(map[string]*models.ChunkRecord)
	for i, templateSlot := range templateSlots {
		slotName := strings.TrimPrefix(templateSlot.Content, "#")
		value, hasValue := req.SlotValues[slotName]
		if !hasValue {
			value = "" // Empty value if not provided
		}
		
		slotValueID := instanceID + "-slot-" + slotName
		seqNum := i
		slotValue := &models.ChunkRecord{
			ID:              slotValueID,
			Content:         value,
			ParentChunkID:   &instanceID,
			TemplateChunkID: &req.TemplateChunkID,
			SlotValue:       &value,
			SequenceNumber:  &seqNum,
		}
		slotValues[slotName] = slotValue
		m.chunks[slotValueID] = slotValue
	}
	
	instance := &models.TemplateInstance{
		Instance:   instanceChunk,
		SlotValues: slotValues,
	}
	
	m.chunks[instanceID] = instanceChunk
	
	return instance, nil
}

func (m *MockSupabaseClient) GetTemplateInstances(ctx context.Context, templateChunkID string) ([]models.TemplateInstance, error) {
	var instances []models.TemplateInstance
	
	for _, chunk := range m.chunks {
		if chunk.TemplateChunkID != nil && *chunk.TemplateChunkID == templateChunkID && 
		   !chunk.IsTemplate && !chunk.IsSlot && chunk.ParentChunkID == nil {
			
			// Get slot values for this instance
			slotValues := make(map[string]*models.ChunkRecord)
			for _, slotValueChunk := range m.chunks {
				if slotValueChunk.ParentChunkID != nil && *slotValueChunk.ParentChunkID == chunk.ID {
					// For mock, we'll use a simple naming convention
					slotName := strings.TrimSuffix(strings.TrimPrefix(slotValueChunk.ID, chunk.ID+"-slot-"), "")
					slotValues[slotName] = slotValueChunk
				}
			}
			
			instances = append(instances, models.TemplateInstance{
				Instance:   chunk,
				SlotValues: slotValues,
			})
		}
	}
	
	return instances, nil
}

func (m *MockSupabaseClient) UpdateSlotValue(ctx context.Context, instanceChunkID, slotName, value string) error {
	// Find the slot value chunk and update it
	slotValueID := instanceChunkID + "-slot-" + slotName
	if slotValue, exists := m.chunks[slotValueID]; exists {
		slotValue.Content = value
		slotValue.SlotValue = &value
		return nil
	}
	return &SupabaseError{Code: "404", Message: "Slot value not found"}
}

// MockSupabaseClient for testing without database
type MockSupabaseClient struct {
	texts    map[string]*models.TextRecord
	chunks   map[string]*models.ChunkRecord
	tagRels  map[string]*models.ChunkTag // key: chunkID_tagChunkID
	nodes    []models.GraphNode
	edges    []models.GraphEdge
}

func NewMockSupabaseClient() *MockSupabaseClient {
	return &MockSupabaseClient{
		texts:   make(map[string]*models.TextRecord),
		chunks:  make(map[string]*models.ChunkRecord),
		tagRels: make(map[string]*models.ChunkTag),
		nodes:   make([]models.GraphNode, 0),
		edges:   make([]models.GraphEdge, 0),
	}
}

func (m *MockSupabaseClient) AddTag(ctx context.Context, chunkID string, tagContent string) error {
	// Check if chunk exists
	if _, exists := m.chunks[chunkID]; !exists {
		return &SupabaseError{Code: "404", Message: "Chunk not found"}
	}
	
	// Find or create tag chunk
	var tagChunk *models.ChunkRecord
	for _, chunk := range m.chunks {
		if chunk.Content == tagContent {
			tagChunk = chunk
			break
		}
	}
	
	if tagChunk == nil {
		// Create new tag chunk
		targetChunk := m.chunks[chunkID]
		tagChunk = &models.ChunkRecord{
			ID:          generateUUID(),
			TextID:      targetChunk.TextID,
			Content:     tagContent,
			IsTemplate:  false,
			IsSlot:      false,
			IndentLevel: 0,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		m.chunks[tagChunk.ID] = tagChunk
	}
	
	// Create tag relationship
	relKey := chunkID + "_" + tagChunk.ID
	m.tagRels[relKey] = &models.ChunkTag{
		ID:         generateUUID(),
		ChunkID:    chunkID,
		TagChunkID: tagChunk.ID,
		CreatedAt:  time.Now(),
	}
	
	return nil
}

func (m *MockSupabaseClient) RemoveTag(ctx context.Context, chunkID string, tagChunkID string) error {
	relKey := chunkID + "_" + tagChunkID
	if _, exists := m.tagRels[relKey]; !exists {
		return &SupabaseError{Code: "404", Message: "Tag relationship not found"}
	}
	
	delete(m.tagRels, relKey)
	return nil
}

func (m *MockSupabaseClient) GetChunkTags(ctx context.Context, chunkID string) ([]models.ChunkRecord, error) {
	var tags []models.ChunkRecord
	
	for _, tagRel := range m.tagRels {
		if tagRel.ChunkID == chunkID {
			if tagChunk, exists := m.chunks[tagRel.TagChunkID]; exists {
				tags = append(tags, *tagChunk)
			}
		}
	}
	
	return tags, nil
}

func (m *MockSupabaseClient) GetChunksByTag(ctx context.Context, tagContent string) ([]models.ChunkRecord, error) {
	// Find tag chunk by content
	var tagChunkID string
	for _, chunk := range m.chunks {
		if chunk.Content == tagContent {
			tagChunkID = chunk.ID
			break
		}
	}
	
	if tagChunkID == "" {
		return []models.ChunkRecord{}, nil
	}
	
	// Find chunks with this tag
	var chunks []models.ChunkRecord
	for _, tagRel := range m.tagRels {
		if tagRel.TagChunkID == tagChunkID {
			if chunk, exists := m.chunks[tagRel.ChunkID]; exists {
				chunks = append(chunks, *chunk)
			}
		}
	}
	
	return chunks, nil
}

func (m *MockSupabaseClient) GetChunkHierarchy(ctx context.Context, rootChunkID string) (*models.ChunkHierarchy, error) {
	rootChunk, err := m.GetChunkByID(ctx, rootChunkID)
	if err != nil {
		return nil, err
	}
	
	return m.buildMockHierarchy(ctx, rootChunk, 0)
}

func (m *MockSupabaseClient) buildMockHierarchy(ctx context.Context, chunk *models.ChunkRecord, level int) (*models.ChunkHierarchy, error) {
	hierarchy := &models.ChunkHierarchy{
		Chunk: chunk,
		Level: level,
	}
	
	children, err := m.GetChildrenChunks(ctx, chunk.ID)
	if err != nil {
		return nil, err
	}
	
	for _, child := range children {
		childHierarchy, err := m.buildMockHierarchy(ctx, &child, level+1)
		if err != nil {
			return nil, err
		}
		hierarchy.Children = append(hierarchy.Children, *childHierarchy)
	}
	
	return hierarchy, nil
}

func (m *MockSupabaseClient) GetChildrenChunks(ctx context.Context, parentChunkID string) ([]models.ChunkRecord, error) {
	var children []models.ChunkRecord
	for _, chunk := range m.chunks {
		if chunk.ParentChunkID != nil && *chunk.ParentChunkID == parentChunkID {
			children = append(children, *chunk)
		}
	}
	return children, nil
}

func (m *MockSupabaseClient) GetSiblingChunks(ctx context.Context, chunkID string) ([]models.ChunkRecord, error) {
	chunk, exists := m.chunks[chunkID]
	if !exists {
		return nil, &SupabaseError{Code: "404", Message: "Chunk not found"}
	}
	
	var siblings []models.ChunkRecord
	for _, otherChunk := range m.chunks {
		if otherChunk.ID == chunkID {
			continue // Skip the chunk itself
		}
		
		// Check if they have the same parent
		if chunk.ParentChunkID == nil && otherChunk.ParentChunkID == nil {
			// Both are root level in the same text
			if chunk.TextID == otherChunk.TextID {
				siblings = append(siblings, *otherChunk)
			}
		} else if chunk.ParentChunkID != nil && otherChunk.ParentChunkID != nil {
			// Both have parents, check if same parent
			if *chunk.ParentChunkID == *otherChunk.ParentChunkID {
				siblings = append(siblings, *otherChunk)
			}
		}
	}
	
	return siblings, nil
}

func (m *MockSupabaseClient) MoveChunk(ctx context.Context, req *models.MoveChunkRequest) error {
	chunk, exists := m.chunks[req.ChunkID]
	if !exists {
		return &SupabaseError{Code: "404", Message: "Chunk not found"}
	}
	
	chunk.ParentChunkID = req.NewParentID
	chunk.IndentLevel = req.NewIndentLevel
	if req.NewPosition >= 0 {
		chunk.SequenceNumber = &req.NewPosition
	}
	
	m.chunks[req.ChunkID] = chunk
	return nil
}

func (m *MockSupabaseClient) BulkUpdateChunks(ctx context.Context, req *models.BulkUpdateRequest) error {
	for _, update := range req.Updates {
		chunk, exists := m.chunks[update.ChunkID]
		if !exists {
			return &SupabaseError{Code: "404", Message: "Chunk not found"}
		}
		
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
		
		chunk.UpdatedAt = time.Now()
		m.chunks[update.ChunkID] = chunk
	}
	
	return nil
}

func (m *MockSupabaseClient) SearchChunks(ctx context.Context, query string, filters map[string]interface{}) ([]models.ChunkRecord, error) {
	return nil, nil
}

func (m *MockSupabaseClient) SearchByTag(ctx context.Context, tagContent string) ([]models.ChunkWithTags, error) {
	chunks, err := m.GetChunksByTag(ctx, tagContent)
	if err != nil {
		return nil, err
	}
	
	var results []models.ChunkWithTags
	for _, chunk := range chunks {
		tags, err := m.GetChunkTags(ctx, chunk.ID)
		if err != nil {
			tags = []models.ChunkRecord{}
		}
		
		results = append(results, models.ChunkWithTags{
			Chunk: &chunk,
			Tags:  tags,
		})
	}
	
	return results, nil
}

func (m *MockSupabaseClient) InsertEmbeddings(ctx context.Context, embeddings []models.EmbeddingRecord) error {
	return nil
}

func (m *MockSupabaseClient) SearchSimilar(ctx context.Context, queryVector []float64, limit int) ([]models.SimilarityResult, error) {
	return nil, nil
}

// Graph operations are implemented in supabase_graph_mock_test.go

// Test functions
func TestSupabaseClient_HealthCheck(t *testing.T) {
	client := NewMockSupabaseClient()
	ctx := context.Background()
	
	err := client.HealthCheck(ctx)
	if err != nil {
		t.Errorf("HealthCheck failed: %v", err)
	}
}

func TestSupabaseClient_TextOperations(t *testing.T) {
	client := NewMockSupabaseClient()
	ctx := context.Background()
	
	// Test InsertText
	text := &models.TextRecord{
		Content: "Test content",
		Title:   "Test title",
	}
	
	err := client.InsertText(ctx, text)
	if err != nil {
		t.Fatalf("InsertText failed: %v", err)
	}
	
	if text.ID == "" {
		t.Error("Text ID should be generated")
	}
	
	if text.Status != "processing" {
		t.Errorf("Expected status 'processing', got '%s'", text.Status)
	}
	
	// Test GetTextByID
	retrieved, err := client.GetTextByID(ctx, text.ID)
	if err != nil {
		t.Fatalf("GetTextByID failed: %v", err)
	}
	
	if retrieved.Text.Content != text.Content {
		t.Errorf("Expected content '%s', got '%s'", text.Content, retrieved.Text.Content)
	}
	
	// Test UpdateText
	text.Title = "Updated title"
	err = client.UpdateText(ctx, text)
	if err != nil {
		t.Fatalf("UpdateText failed: %v", err)
	}
	
	// Test GetTexts
	textList, err := client.GetTexts(ctx, nil)
	if err != nil {
		t.Fatalf("GetTexts failed: %v", err)
	}
	
	if len(textList.Texts) != 1 {
		t.Errorf("Expected 1 text, got %d", len(textList.Texts))
	}
	
	// Test DeleteText
	err = client.DeleteText(ctx, text.ID)
	if err != nil {
		t.Fatalf("DeleteText failed: %v", err)
	}
	
	// Verify deletion
	_, err = client.GetTextByID(ctx, text.ID)
	if err == nil {
		t.Error("Expected error when getting deleted text")
	}
}

func TestSupabaseClient_ChunkOperations(t *testing.T) {
	client := NewMockSupabaseClient()
	ctx := context.Background()
	
	// Create a text first
	text := &models.TextRecord{
		Content: "Test content",
		Title:   "Test title",
	}
	err := client.InsertText(ctx, text)
	if err != nil {
		t.Fatalf("InsertText failed: %v", err)
	}
	
	// Test InsertChunk
	chunk := &models.ChunkRecord{
		TextID:      text.ID,
		Content:     "Test chunk content",
		IndentLevel: 0,
	}
	
	err = client.InsertChunk(ctx, chunk)
	if err != nil {
		t.Fatalf("InsertChunk failed: %v", err)
	}
	
	if chunk.ID == "" {
		t.Error("Chunk ID should be generated")
	}
	
	// Test GetChunkByID
	retrieved, err := client.GetChunkByID(ctx, chunk.ID)
	if err != nil {
		t.Fatalf("GetChunkByID failed: %v", err)
	}
	
	if retrieved.Content != chunk.Content {
		t.Errorf("Expected content '%s', got '%s'", chunk.Content, retrieved.Content)
	}
	
	// Test GetChunkByContent
	retrieved, err = client.GetChunkByContent(ctx, chunk.Content)
	if err != nil {
		t.Fatalf("GetChunkByContent failed: %v", err)
	}
	
	if retrieved.ID != chunk.ID {
		t.Errorf("Expected ID '%s', got '%s'", chunk.ID, retrieved.ID)
	}
	
	// Test GetChunksByTextID
	chunks, err := client.GetChunksByTextID(ctx, text.ID)
	if err != nil {
		t.Fatalf("GetChunksByTextID failed: %v", err)
	}
	
	if len(chunks) != 1 {
		t.Errorf("Expected 1 chunk, got %d", len(chunks))
	}
	
	// Test UpdateChunk
	chunk.Content = "Updated chunk content"
	err = client.UpdateChunk(ctx, chunk)
	if err != nil {
		t.Fatalf("UpdateChunk failed: %v", err)
	}
	
	// Test InsertChunks (batch)
	chunks = []models.ChunkRecord{
		{TextID: text.ID, Content: "Chunk 1", IndentLevel: 1},
		{TextID: text.ID, Content: "Chunk 2", IndentLevel: 1},
	}
	
	err = client.InsertChunks(ctx, chunks)
	if err != nil {
		t.Fatalf("InsertChunks failed: %v", err)
	}
	
	// Verify batch insert
	allChunks, err := client.GetChunksByTextID(ctx, text.ID)
	if err != nil {
		t.Fatalf("GetChunksByTextID failed: %v", err)
	}
	
	if len(allChunks) != 3 { // 1 original + 2 batch inserted
		t.Errorf("Expected 3 chunks, got %d", len(allChunks))
	}
	
	// Test DeleteChunk
	err = client.DeleteChunk(ctx, chunk.ID)
	if err != nil {
		t.Fatalf("DeleteChunk failed: %v", err)
	}
	
	// Verify deletion
	_, err = client.GetChunkByID(ctx, chunk.ID)
	if err == nil {
		t.Error("Expected error when getting deleted chunk")
	}
}

func TestNewSupabaseClient(t *testing.T) {
	cfg := &config.SupabaseConfig{
		URL:    "https://test.supabase.co",
		APIKey: "test-api-key",
	}
	
	client := NewSupabaseClient(cfg)
	if client == nil {
		t.Error("NewSupabaseClient should return a client instance")
	}
	
	// Type assertion to check if it's the correct implementation
	if _, ok := client.(*supabaseHTTPClient); !ok {
		t.Error("NewSupabaseClient should return supabaseHTTPClient instance")
	}
}

func TestSupabaseClient_HierarchyOperations(t *testing.T) {
	client := NewMockSupabaseClient()
	ctx := context.Background()
	
	// Create a text first
	text := &models.TextRecord{
		Content: "Test content",
		Title:   "Test title",
	}
	err := client.InsertText(ctx, text)
	if err != nil {
		t.Fatalf("InsertText failed: %v", err)
	}
	
	// Create a hierarchical structure
	// Root chunk
	rootChunk := &models.ChunkRecord{
		TextID:         text.ID,
		Content:        "Root chunk",
		IndentLevel:    0,
		SequenceNumber: intPtr(0),
	}
	err = client.InsertChunk(ctx, rootChunk)
	if err != nil {
		t.Fatalf("InsertChunk failed: %v", err)
	}
	
	// Child chunks
	child1 := &models.ChunkRecord{
		TextID:         text.ID,
		Content:        "Child 1",
		ParentChunkID:  &rootChunk.ID,
		IndentLevel:    1,
		SequenceNumber: intPtr(0),
	}
	err = client.InsertChunk(ctx, child1)
	if err != nil {
		t.Fatalf("InsertChunk failed: %v", err)
	}
	
	child2 := &models.ChunkRecord{
		TextID:         text.ID,
		Content:        "Child 2",
		ParentChunkID:  &rootChunk.ID,
		IndentLevel:    1,
		SequenceNumber: intPtr(1),
	}
	err = client.InsertChunk(ctx, child2)
	if err != nil {
		t.Fatalf("InsertChunk failed: %v", err)
	}
	
	// Test GetChildrenChunks
	children, err := client.GetChildrenChunks(ctx, rootChunk.ID)
	if err != nil {
		t.Fatalf("GetChildrenChunks failed: %v", err)
	}
	
	if len(children) != 2 {
		t.Errorf("Expected 2 children, got %d", len(children))
	}
	
	// Test GetSiblingChunks
	siblings, err := client.GetSiblingChunks(ctx, child1.ID)
	if err != nil {
		t.Fatalf("GetSiblingChunks failed: %v", err)
	}
	
	if len(siblings) != 1 {
		t.Errorf("Expected 1 sibling, got %d", len(siblings))
	}
	
	if siblings[0].ID != child2.ID {
		t.Errorf("Expected sibling to be child2, got %s", siblings[0].ID)
	}
}

// Helper function to create int pointer
func intPtr(i int) *int {
	return &i
}

func TestSupabaseClient_TagOperations(t *testing.T) {
	client := NewMockSupabaseClient()
	ctx := context.Background()
	
	// Create a text first
	text := &models.TextRecord{
		Content: "Test content",
		Title:   "Test title",
	}
	err := client.InsertText(ctx, text)
	if err != nil {
		t.Fatalf("InsertText failed: %v", err)
	}
	
	// Create a chunk
	chunk := &models.ChunkRecord{
		TextID:      text.ID,
		Content:     "Test chunk content",
		IndentLevel: 0,
	}
	err = client.InsertChunk(ctx, chunk)
	if err != nil {
		t.Fatalf("InsertChunk failed: %v", err)
	}
	
	// Test AddTag
	tagContent := "important"
	err = client.AddTag(ctx, chunk.ID, tagContent)
	if err != nil {
		t.Fatalf("AddTag failed: %v", err)
	}
	
	// Test GetChunkTags
	tags, err := client.GetChunkTags(ctx, chunk.ID)
	if err != nil {
		t.Fatalf("GetChunkTags failed: %v", err)
	}
	
	if len(tags) != 1 {
		t.Errorf("Expected 1 tag, got %d", len(tags))
	}
	
	if tags[0].Content != tagContent {
		t.Errorf("Expected tag content '%s', got '%s'", tagContent, tags[0].Content)
	}
	
	// Test GetChunksByTag
	chunksWithTag, err := client.GetChunksByTag(ctx, tagContent)
	if err != nil {
		t.Fatalf("GetChunksByTag failed: %v", err)
	}
	
	if len(chunksWithTag) != 1 {
		t.Errorf("Expected 1 chunk with tag, got %d", len(chunksWithTag))
	}
	
	if chunksWithTag[0].ID != chunk.ID {
		t.Errorf("Expected chunk ID '%s', got '%s'", chunk.ID, chunksWithTag[0].ID)
	}
	
	// Test SearchByTag
	searchResults, err := client.SearchByTag(ctx, tagContent)
	if err != nil {
		t.Fatalf("SearchByTag failed: %v", err)
	}
	
	if len(searchResults) != 1 {
		t.Errorf("Expected 1 search result, got %d", len(searchResults))
	}
	
	if searchResults[0].Chunk.ID != chunk.ID {
		t.Errorf("Expected chunk ID '%s', got '%s'", chunk.ID, searchResults[0].Chunk.ID)
	}
	
	if len(searchResults[0].Tags) != 1 {
		t.Errorf("Expected 1 tag in search result, got %d", len(searchResults[0].Tags))
	}
	
	// Test RemoveTag
	tagChunkID := tags[0].ID
	err = client.RemoveTag(ctx, chunk.ID, tagChunkID)
	if err != nil {
		t.Fatalf("RemoveTag failed: %v", err)
	}
	
	// Verify tag removal
	tagsAfterRemoval, err := client.GetChunkTags(ctx, chunk.ID)
	if err != nil {
		t.Fatalf("GetChunkTags after removal failed: %v", err)
	}
	
	if len(tagsAfterRemoval) != 0 {
		t.Errorf("Expected 0 tags after removal, got %d", len(tagsAfterRemoval))
	}
}