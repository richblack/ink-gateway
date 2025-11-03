package testdata

import (
	"context"
	"fmt"
	"math/rand"
	"semantic-text-processor/models"
	"time"
)

// TestDataManager handles test data lifecycle
type TestDataManager interface {
	LoadDataSet(name string) (*DataSet, error)
	GenerateData(schema DataSchema) (*DataSet, error)
	CleanData(pattern string) error
	SeedDatabase(ctx context.Context, dataSet *DataSet) error
	CreateSnapshot(name string) error
	RestoreSnapshot(name string) error
}

// Manager implements TestDataManager interface
type Manager struct {
	config   *Config
	datasets map[string]*DataSet
	snapshots map[string]*Snapshot
}

// Config holds test data manager configuration
type Config struct {
	DataDirectory   string
	SnapshotDir     string
	MaxDataSetSize  int
	CleanupAfterUse bool
}

// DataSet represents a collection of test data
type DataSet struct {
	Name        string                  `json:"name"`
	Version     string                  `json:"version"`
	CreatedAt   time.Time              `json:"created_at"`
	Texts       []models.TextRecord    `json:"texts"`
	Chunks      []models.ChunkRecord   `json:"chunks"`
	Embeddings  []models.EmbeddingRecord `json:"embeddings"`
	Templates   []models.TemplateRecord `json:"templates"`
	Tags        []models.TagRecord     `json:"tags"`
	GraphNodes  []models.GraphNode     `json:"graph_nodes"`
	GraphEdges  []models.GraphEdge     `json:"graph_edges"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// DataSchema defines structure for generated data
type DataSchema struct {
	TextCount       int                    `json:"text_count"`
	ChunksPerText   int                    `json:"chunks_per_text"`
	EmbeddingDim    int                    `json:"embedding_dim"`
	TemplateCount   int                    `json:"template_count"`
	TagCount        int                    `json:"tag_count"`
	GraphNodeCount  int                    `json:"graph_node_count"`
	GraphEdgeCount  int                    `json:"graph_edge_count"`
	TextPatterns    []TextPattern          `json:"text_patterns"`
	RelationTypes   []string               `json:"relation_types"`
	EntityTypes     []string               `json:"entity_types"`
	CustomFields    map[string]interface{} `json:"custom_fields"`
}

// TextPattern defines patterns for generating realistic text
type TextPattern struct {
	Type        string   `json:"type"`
	MinLength   int      `json:"min_length"`
	MaxLength   int      `json:"max_length"`
	Keywords    []string `json:"keywords"`
	Structure   string   `json:"structure"`
	Language    string   `json:"language"`
}

// Snapshot represents a database state snapshot
type Snapshot struct {
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	DataSet   *DataSet  `json:"dataset"`
	Checksum  string    `json:"checksum"`
}

// NewManager creates a new test data manager
func NewManager(config *Config) *Manager {
	return &Manager{
		config:    config,
		datasets:  make(map[string]*DataSet),
		snapshots: make(map[string]*Snapshot),
	}
}

// LoadDataSet loads a predefined dataset
func (m *Manager) LoadDataSet(name string) (*DataSet, error) {
	// Check cache first
	if dataset, exists := m.datasets[name]; exists {
		return dataset, nil
	}

	// Load from file system
	dataset, err := m.loadDataSetFromFile(name)
	if err != nil {
		return nil, fmt.Errorf("failed to load dataset %s: %w", name, err)
	}

	// Cache the dataset
	m.datasets[name] = dataset

	return dataset, nil
}

// GenerateData creates synthetic test data based on schema
func (m *Manager) GenerateData(schema DataSchema) (*DataSet, error) {
	dataset := &DataSet{
		Name:      fmt.Sprintf("generated_%d", time.Now().Unix()),
		Version:   "1.0",
		CreatedAt: time.Now(),
		Metadata:  make(map[string]interface{}),
	}

	// Generate texts
	texts, err := m.generateTexts(schema)
	if err != nil {
		return nil, fmt.Errorf("failed to generate texts: %w", err)
	}
	dataset.Texts = texts

	// Generate chunks
	chunks, err := m.generateChunks(schema, texts)
	if err != nil {
		return nil, fmt.Errorf("failed to generate chunks: %w", err)
	}
	dataset.Chunks = chunks

	// Generate embeddings
	embeddings, err := m.generateEmbeddings(schema, chunks)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embeddings: %w", err)
	}
	dataset.Embeddings = embeddings

	// Generate templates
	templates, err := m.generateTemplates(schema)
	if err != nil {
		return nil, fmt.Errorf("failed to generate templates: %w", err)
	}
	dataset.Templates = templates

	// Generate tags
	tags, err := m.generateTags(schema)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tags: %w", err)
	}
	dataset.Tags = tags

	// Generate knowledge graph
	nodes, edges, err := m.generateKnowledgeGraph(schema, chunks)
	if err != nil {
		return nil, fmt.Errorf("failed to generate knowledge graph: %w", err)
	}
	dataset.GraphNodes = nodes
	dataset.GraphEdges = edges

	// Store metadata
	dataset.Metadata["schema"] = schema
	dataset.Metadata["generation_time"] = time.Now()

	return dataset, nil
}

// generateTexts creates synthetic text data
func (m *Manager) generateTexts(schema DataSchema) ([]models.TextRecord, error) {
	texts := make([]models.TextRecord, schema.TextCount)

	for i := 0; i < schema.TextCount; i++ {
		pattern := schema.TextPatterns[i%len(schema.TextPatterns)]
		content, err := m.generateTextContent(pattern)
		if err != nil {
			return nil, err
		}

		texts[i] = models.TextRecord{
			ID:          generateID("text"),
			Content:     content,
			Title:       fmt.Sprintf("Test Document %d", i+1),
			Source:      "test_generation",
			Language:    pattern.Language,
			ProcessedAt: time.Now(),
			Status:      "active",
			Metadata: map[string]interface{}{
				"test_pattern": pattern.Type,
				"generated":    true,
			},
		}
	}

	return texts, nil
}

// generateTextContent creates realistic text content
func (m *Manager) generateTextContent(pattern TextPattern) (string, error) {
	length := pattern.MinLength + rand.Intn(pattern.MaxLength-pattern.MinLength)

	switch pattern.Type {
	case "article":
		return m.generateArticleContent(pattern, length)
	case "technical":
		return m.generateTechnicalContent(pattern, length)
	case "narrative":
		return m.generateNarrativeContent(pattern, length)
	default:
		return m.generateGenericContent(pattern, length)
	}
}

// generateArticleContent creates article-style content
func (m *Manager) generateArticleContent(pattern TextPattern, length int) (string, error) {
	sections := []string{
		"Introduction: ",
		"Background: ",
		"Analysis: ",
		"Results: ",
		"Conclusion: ",
	}

	content := ""
	sectionLength := length / len(sections)

	for _, section := range sections {
		content += section
		content += m.generateRandomText(sectionLength, pattern.Keywords)
		content += "\n\n"
	}

	return content, nil
}

// generateTechnicalContent creates technical documentation style content
func (m *Manager) generateTechnicalContent(pattern TextPattern, length int) (string, error) {
	content := "Technical Documentation\n\n"
	content += "Overview: " + m.generateRandomText(length/4, pattern.Keywords) + "\n\n"
	content += "Implementation: " + m.generateRandomText(length/4, pattern.Keywords) + "\n\n"
	content += "Configuration: " + m.generateRandomText(length/4, pattern.Keywords) + "\n\n"
	content += "Examples: " + m.generateRandomText(length/4, pattern.Keywords)

	return content, nil
}

// generateNarrativeContent creates story-style content
func (m *Manager) generateNarrativeContent(pattern TextPattern, length int) (string, error) {
	content := m.generateRandomText(length, pattern.Keywords)
	return content, nil
}

// generateGenericContent creates general purpose content
func (m *Manager) generateGenericContent(pattern TextPattern, length int) (string, error) {
	content := m.generateRandomText(length, pattern.Keywords)
	return content, nil
}

// generateRandomText creates random text incorporating keywords
func (m *Manager) generateRandomText(length int, keywords []string) string {
	words := []string{
		"the", "and", "to", "of", "a", "in", "is", "it", "you", "that",
		"he", "was", "for", "on", "are", "as", "with", "his", "they", "i",
		"at", "be", "this", "have", "from", "or", "one", "had", "by", "words",
		"but", "not", "what", "all", "were", "we", "when", "your", "can", "said",
	}

	// Add keywords to word pool
	allWords := append(words, keywords...)

	content := ""
	wordCount := length / 6 // Approximate words per character

	for i := 0; i < wordCount; i++ {
		if i > 0 {
			content += " "
		}
		content += allWords[rand.Intn(len(allWords))]

		// Add punctuation occasionally
		if i > 0 && i%15 == 0 {
			content += "."
		}
	}

	return content
}

// generateChunks creates chunks from texts
func (m *Manager) generateChunks(schema DataSchema, texts []models.TextRecord) ([]models.ChunkRecord, error) {
	var chunks []models.ChunkRecord

	for _, text := range texts {
		textChunks := m.splitTextIntoChunks(text, schema.ChunksPerText)
		chunks = append(chunks, textChunks...)
	}

	return chunks, nil
}

// splitTextIntoChunks divides text into smaller chunks
func (m *Manager) splitTextIntoChunks(text models.TextRecord, chunkCount int) []models.ChunkRecord {
	chunks := make([]models.ChunkRecord, chunkCount)
	contentLength := len(text.Content)
	chunkSize := contentLength / chunkCount

	for i := 0; i < chunkCount; i++ {
		start := i * chunkSize
		end := start + chunkSize
		if i == chunkCount-1 {
			end = contentLength // Last chunk gets remainder
		}

		chunks[i] = models.ChunkRecord{
			ID:        generateID("chunk"),
			TextID:    text.ID,
			Content:   text.Content[start:end],
			Position:  i,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Metadata: map[string]interface{}{
				"chunk_size": end - start,
				"generated":  true,
			},
		}
	}

	return chunks
}

// generateEmbeddings creates embedding vectors for chunks
func (m *Manager) generateEmbeddings(schema DataSchema, chunks []models.ChunkRecord) ([]models.EmbeddingRecord, error) {
	embeddings := make([]models.EmbeddingRecord, len(chunks))

	for i, chunk := range chunks {
		vector := make([]float64, schema.EmbeddingDim)

		// Generate realistic-looking embeddings (normalized random vectors)
		var sum float64
		for j := 0; j < schema.EmbeddingDim; j++ {
			vector[j] = rand.NormFloat64()
			sum += vector[j] * vector[j]
		}

		// Normalize the vector
		norm := 1.0 / (sum + 1e-10) // Avoid division by zero
		for j := 0; j < schema.EmbeddingDim; j++ {
			vector[j] *= norm
		}

		embeddings[i] = models.EmbeddingRecord{
			ID:        generateID("embedding"),
			ChunkID:   chunk.ID,
			Vector:    vector,
			Model:     "test-embedding-model",
			CreatedAt: time.Now(),
			Metadata: map[string]interface{}{
				"dimension": schema.EmbeddingDim,
				"generated": true,
			},
		}
	}

	return embeddings, nil
}

// generateTemplates creates template test data
func (m *Manager) generateTemplates(schema DataSchema) ([]models.TemplateRecord, error) {
	templates := make([]models.TemplateRecord, schema.TemplateCount)

	templateTypes := []string{"form", "document", "report", "analysis"}

	for i := 0; i < schema.TemplateCount; i++ {
		templateType := templateTypes[i%len(templateTypes)]

		templates[i] = models.TemplateRecord{
			ID:          generateID("template"),
			Name:        fmt.Sprintf("Test Template %d", i+1),
			Type:        templateType,
			Content:     m.generateTemplateContent(templateType),
			SlotNames:   m.generateSlotNames(templateType),
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			IsActive:    true,
			Metadata: map[string]interface{}{
				"template_type": templateType,
				"generated":     true,
			},
		}
	}

	return templates, nil
}

// generateTemplateContent creates template content based on type
func (m *Manager) generateTemplateContent(templateType string) string {
	switch templateType {
	case "form":
		return "Name: {{name}}\nEmail: {{email}}\nMessage: {{message}}"
	case "document":
		return "Title: {{title}}\nAuthor: {{author}}\nContent: {{content}}"
	case "report":
		return "Report: {{report_title}}\nDate: {{date}}\nSummary: {{summary}}\nDetails: {{details}}"
	case "analysis":
		return "Analysis of {{subject}}\nMethodology: {{methodology}}\nResults: {{results}}\nConclusion: {{conclusion}}"
	default:
		return "Template: {{title}}\nContent: {{content}}"
	}
}

// generateSlotNames creates slot names for templates
func (m *Manager) generateSlotNames(templateType string) []string {
	switch templateType {
	case "form":
		return []string{"name", "email", "message"}
	case "document":
		return []string{"title", "author", "content"}
	case "report":
		return []string{"report_title", "date", "summary", "details"}
	case "analysis":
		return []string{"subject", "methodology", "results", "conclusion"}
	default:
		return []string{"title", "content"}
	}
}

// generateTags creates tag test data
func (m *Manager) generateTags(schema DataSchema) ([]models.TagRecord, error) {
	tags := make([]models.TagRecord, schema.TagCount)

	tagCategories := []string{"category", "priority", "status", "type", "classification"}
	tagValues := map[string][]string{
		"category":      {"research", "documentation", "analysis", "report"},
		"priority":      {"high", "medium", "low", "urgent"},
		"status":        {"draft", "review", "approved", "published"},
		"type":          {"technical", "business", "academic", "personal"},
		"classification": {"public", "internal", "confidential", "restricted"},
	}

	for i := 0; i < schema.TagCount; i++ {
		category := tagCategories[i%len(tagCategories)]
		values := tagValues[category]
		value := values[rand.Intn(len(values))]

		tags[i] = models.TagRecord{
			ID:        generateID("tag"),
			Content:   fmt.Sprintf("%s:%s", category, value),
			Category:  category,
			Value:     value,
			CreatedAt: time.Now(),
			Metadata: map[string]interface{}{
				"generated": true,
				"category":  category,
			},
		}
	}

	return tags, nil
}

// generateKnowledgeGraph creates graph nodes and edges
func (m *Manager) generateKnowledgeGraph(schema DataSchema, chunks []models.ChunkRecord) ([]models.GraphNode, []models.GraphEdge, error) {
	nodes := make([]models.GraphNode, schema.GraphNodeCount)
	edges := make([]models.GraphEdge, schema.GraphEdgeCount)

	// Generate nodes
	for i := 0; i < schema.GraphNodeCount; i++ {
		entityType := schema.EntityTypes[i%len(schema.EntityTypes)]

		nodes[i] = models.GraphNode{
			ID:         generateID("node"),
			EntityName: fmt.Sprintf("%s_%d", entityType, i+1),
			EntityType: entityType,
			ChunkID:    chunks[i%len(chunks)].ID,
			Properties: map[string]interface{}{
				"generated":     true,
				"entity_type":   entityType,
				"test_node_id":  i,
			},
			CreatedAt: time.Now(),
		}
	}

	// Generate edges
	for i := 0; i < schema.GraphEdgeCount; i++ {
		relationType := schema.RelationTypes[i%len(schema.RelationTypes)]
		sourceNode := nodes[rand.Intn(len(nodes))]
		targetNode := nodes[rand.Intn(len(nodes))]

		// Ensure no self-loops
		for sourceNode.ID == targetNode.ID {
			targetNode = nodes[rand.Intn(len(nodes))]
		}

		edges[i] = models.GraphEdge{
			ID:           generateID("edge"),
			SourceNodeID: sourceNode.ID,
			TargetNodeID: targetNode.ID,
			RelationType: relationType,
			Properties: map[string]interface{}{
				"generated":      true,
				"relation_type":  relationType,
				"test_edge_id":   i,
			},
			CreatedAt: time.Now(),
		}
	}

	return nodes, edges, nil
}

// generateID creates a unique identifier with prefix
func generateID(prefix string) string {
	return fmt.Sprintf("%s_%d_%d", prefix, time.Now().UnixNano(), rand.Intn(10000))
}

// CleanData removes test data matching pattern
func (m *Manager) CleanData(pattern string) error {
	// Implementation for cleaning test data
	// This would connect to the database and remove test data
	return nil
}

// SeedDatabase inserts test data into the database
func (m *Manager) SeedDatabase(ctx context.Context, dataSet *DataSet) error {
	// Implementation for seeding database with test data
	// This would use the SupabaseClient to insert data
	return nil
}

// CreateSnapshot creates a database state snapshot
func (m *Manager) CreateSnapshot(name string) error {
	// Implementation for creating database snapshots
	return nil
}

// RestoreSnapshot restores database from a snapshot
func (m *Manager) RestoreSnapshot(name string) error {
	// Implementation for restoring from snapshots
	return nil
}

// loadDataSetFromFile loads dataset from file system
func (m *Manager) loadDataSetFromFile(name string) (*DataSet, error) {
	// Implementation for loading datasets from files
	return nil, fmt.Errorf("not implemented")
}

// Default test data schemas
func GetDefaultSchemas() map[string]DataSchema {
	return map[string]DataSchema{
		"small": {
			TextCount:      10,
			ChunksPerText:  3,
			EmbeddingDim:   384,
			TemplateCount:  5,
			TagCount:       20,
			GraphNodeCount: 30,
			GraphEdgeCount: 50,
			TextPatterns: []TextPattern{
				{Type: "article", MinLength: 500, MaxLength: 1000, Keywords: []string{"test", "sample", "data"}},
				{Type: "technical", MinLength: 300, MaxLength: 800, Keywords: []string{"system", "process", "method"}},
			},
			RelationTypes: []string{"related_to", "part_of", "describes", "contains"},
			EntityTypes:   []string{"concept", "method", "system", "process"},
		},
		"medium": {
			TextCount:      50,
			ChunksPerText:  5,
			EmbeddingDim:   384,
			TemplateCount:  15,
			TagCount:       100,
			GraphNodeCount: 150,
			GraphEdgeCount: 300,
			TextPatterns: []TextPattern{
				{Type: "article", MinLength: 1000, MaxLength: 2000, Keywords: []string{"analysis", "research", "study"}},
				{Type: "technical", MinLength: 800, MaxLength: 1500, Keywords: []string{"implementation", "architecture", "design"}},
				{Type: "narrative", MinLength: 600, MaxLength: 1200, Keywords: []string{"story", "experience", "case"}},
			},
			RelationTypes: []string{"related_to", "part_of", "describes", "contains", "implements", "uses"},
			EntityTypes:   []string{"concept", "method", "system", "process", "component", "service"},
		},
		"large": {
			TextCount:      200,
			ChunksPerText:  8,
			EmbeddingDim:   768,
			TemplateCount:  50,
			TagCount:       500,
			GraphNodeCount: 1000,
			GraphEdgeCount: 2000,
			TextPatterns: []TextPattern{
				{Type: "article", MinLength: 2000, MaxLength: 4000, Keywords: []string{"comprehensive", "detailed", "analysis"}},
				{Type: "technical", MinLength: 1500, MaxLength: 3000, Keywords: []string{"specification", "documentation", "guide"}},
				{Type: "narrative", MinLength: 1000, MaxLength: 2500, Keywords: []string{"methodology", "approach", "framework"}},
			},
			RelationTypes: []string{"related_to", "part_of", "describes", "contains", "implements", "uses", "depends_on", "extends"},
			EntityTypes:   []string{"concept", "method", "system", "process", "component", "service", "module", "framework"},
		},
	}
}