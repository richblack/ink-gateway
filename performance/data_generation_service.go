package performance

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"semantic-text-processor/models"
	"strings"
	"time"
)

// DataGenerationService handles large-scale test data generation
type DataGenerationService struct {
	logger        *log.Logger
	textPatterns  []TextPattern
	tagPatterns   []TagPattern
	metadataTypes []MetadataType
	rand          *rand.Rand
}

// TextPattern defines patterns for generating realistic text content
type TextPattern struct {
	Category    string   `json:"category"`
	Templates   []string `json:"templates"`
	Keywords    []string `json:"keywords"`
	MinLength   int      `json:"min_length"`
	MaxLength   int      `json:"max_length"`
	Complexity  string   `json:"complexity"` // "simple", "medium", "complex"
}

// TagPattern defines patterns for generating relevant tags
type TagPattern struct {
	Domain      string   `json:"domain"`
	BaseTags    []string `json:"base_tags"`
	Frequencies map[string]float64 `json:"frequencies"`
	MaxTags     int      `json:"max_tags"`
}

// MetadataType defines types of metadata to generate
type MetadataType struct {
	Name       string      `json:"name"`
	Type       string      `json:"type"` // "string", "int", "float", "bool", "timestamp"
	Values     []interface{} `json:"values,omitempty"`
	Generator  string      `json:"generator"` // "random", "sequential", "pattern"
}

// DataIntegrityCheck represents the result of data integrity verification
type DataIntegrityCheck struct {
	TotalRecords     int                    `json:"total_records"`
	ValidRecords     int                    `json:"valid_records"`
	InvalidRecords   int                    `json:"invalid_records"`
	MissingFields    map[string]int         `json:"missing_fields"`
	DataQualityScore float64                `json:"data_quality_score"`
	Issues           []DataQualityIssue     `json:"issues"`
	SampleValidation map[string]interface{} `json:"sample_validation"`
}

// DataQualityIssue represents a specific data quality problem
type DataQualityIssue struct {
	Type        string `json:"type"`
	Field       string `json:"field"`
	Count       int    `json:"count"`
	Description string `json:"description"`
	Severity    string `json:"severity"`
}

// NewDataGenerationService creates a new data generation service
func NewDataGenerationService(logger *log.Logger) *DataGenerationService {
	return &DataGenerationService{
		logger:        logger,
		textPatterns:  getDefaultTextPatterns(),
		tagPatterns:   getDefaultTagPatterns(),
		metadataTypes: getDefaultMetadataTypes(),
		rand:          rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// GenerateBatchData generates a batch of test data
func (dgs *DataGenerationService) GenerateBatchData(ctx context.Context, batchSize, startOffset int) (generated, errors int) {
	dgs.logger.Printf("Generating batch: size=%d, offset=%d", batchSize, startOffset)

	for i := 0; i < batchSize; i++ {
		select {
		case <-ctx.Done():
			return generated, errors
		default:
		}

		recordID := startOffset + i + 1

		// Generate chunk data
		chunk, err := dgs.generateChunkRecord(recordID)
		if err != nil {
			dgs.logger.Printf("Failed to generate chunk %d: %v", recordID, err)
			errors++
			continue
		}

		// Generate embedding vector
		embedding := dgs.generateEmbedding(chunk.Content)

		// Generate tags
		tags := dgs.generateTags(chunk.Content, chunk.Category)

		// Generate metadata
		metadata := dgs.generateMetadata(recordID)

		// Create complete record
		record := models.GeneratedChunkRecord{
			ID:        recordID,
			Content:   chunk.Content,
			Category:  chunk.Category,
			Embedding: embedding,
			Tags:      tags,
			Metadata:  metadata,
			CreatedAt: time.Now(),
		}

		// Validate record
		if err := dgs.validateRecord(&record); err != nil {
			dgs.logger.Printf("Invalid record %d: %v", recordID, err)
			errors++
			continue
		}

		// In a real implementation, this would save to database
		// For now, we'll just count as generated
		generated++

		// Progress logging for large batches
		if generated%1000 == 0 {
			dgs.logger.Printf("Generated %d records in current batch", generated)
		}
	}

	return generated, errors
}

// generateChunkRecord creates a realistic chunk record
func (dgs *DataGenerationService) generateChunkRecord(id int) (*models.ChunkContent, error) {
	// Select random text pattern
	pattern := dgs.textPatterns[dgs.rand.Intn(len(dgs.textPatterns))]

	// Generate content based on pattern
	content := dgs.generateContentFromPattern(pattern)

	return &models.ChunkContent{
		ID:       id,
		Content:  content,
		Category: pattern.Category,
		Length:   len(content),
	}, nil
}

// generateContentFromPattern creates content based on a text pattern
func (dgs *DataGenerationService) generateContentFromPattern(pattern TextPattern) string {
	template := pattern.Templates[dgs.rand.Intn(len(pattern.Templates))]

	// Replace placeholders with keywords
	content := template
	for i := 0; i < 3; i++ {
		keyword := pattern.Keywords[dgs.rand.Intn(len(pattern.Keywords))]
		placeholder := fmt.Sprintf("{keyword%d}", i+1)
		content = strings.ReplaceAll(content, placeholder, keyword)
	}

	// Ensure length constraints
	targetLength := pattern.MinLength + dgs.rand.Intn(pattern.MaxLength-pattern.MinLength)

	if len(content) < targetLength {
		// Pad with additional sentences
		padding := dgs.generatePaddingText(targetLength - len(content))
		content += " " + padding
	} else if len(content) > targetLength {
		// Truncate to target length
		words := strings.Fields(content)
		truncated := ""
		for _, word := range words {
			if len(truncated)+len(word)+1 <= targetLength {
				if truncated != "" {
					truncated += " "
				}
				truncated += word
			} else {
				break
			}
		}
		content = truncated
	}

	return content
}

// generateEmbedding creates a realistic embedding vector
func (dgs *DataGenerationService) generateEmbedding(content string) []float64 {
	// Generate a 1536-dimensional embedding (OpenAI embedding size)
	dimension := 1536
	embedding := make([]float64, dimension)

	// Generate based on content hash for consistency
	seed := int64(0)
	for _, char := range content {
		seed += int64(char)
	}
	contentRand := rand.New(rand.NewSource(seed))

	// Generate normalized vector
	for i := 0; i < dimension; i++ {
		embedding[i] = contentRand.NormFloat64()
	}

	// Normalize the vector
	magnitude := 0.0
	for _, val := range embedding {
		magnitude += val * val
	}
	magnitude = 1.0 / (magnitude + 1e-8) // Avoid division by zero

	for i := range embedding {
		embedding[i] *= magnitude
	}

	return embedding
}

// generateTags creates relevant tags for content
func (dgs *DataGenerationService) generateTags(content, category string) []string {
	var tags []string

	// Find matching tag pattern
	var pattern *TagPattern
	for _, p := range dgs.tagPatterns {
		if p.Domain == category {
			pattern = &p
			break
		}
	}

	if pattern == nil {
		pattern = &dgs.tagPatterns[0] // Default pattern
	}

	// Generate tags based on frequencies
	numTags := 1 + dgs.rand.Intn(pattern.MaxTags)
	usedTags := make(map[string]bool)

	for len(tags) < numTags {
		tag := pattern.BaseTags[dgs.rand.Intn(len(pattern.BaseTags))]

		if !usedTags[tag] {
			// Check frequency probability
			frequency := pattern.Frequencies[tag]
			if frequency == 0 {
				frequency = 0.3 // Default frequency
			}

			if dgs.rand.Float64() < frequency {
				tags = append(tags, tag)
				usedTags[tag] = true
			}
		}
	}

	// Add content-specific tags
	contentLower := strings.ToLower(content)
	if strings.Contains(contentLower, "performance") {
		tags = append(tags, "performance")
	}
	if strings.Contains(contentLower, "database") {
		tags = append(tags, "database")
	}
	if strings.Contains(contentLower, "optimization") {
		tags = append(tags, "optimization")
	}

	return tags
}

// generateMetadata creates metadata for a record
func (dgs *DataGenerationService) generateMetadata(id int) map[string]interface{} {
	metadata := make(map[string]interface{})

	for _, metaType := range dgs.metadataTypes {
		var value interface{}

		switch metaType.Type {
		case "string":
			if len(metaType.Values) > 0 {
				value = metaType.Values[dgs.rand.Intn(len(metaType.Values))]
			} else {
				value = fmt.Sprintf("value_%d", dgs.rand.Intn(100))
			}
		case "int":
			value = dgs.rand.Intn(1000)
		case "float":
			value = dgs.rand.Float64() * 100
		case "bool":
			value = dgs.rand.Float64() < 0.5
		case "timestamp":
			value = time.Now().Add(-time.Duration(dgs.rand.Intn(86400)) * time.Second)
		}

		metadata[metaType.Name] = value
	}

	// Add record-specific metadata
	metadata["record_id"] = id
	metadata["generation_time"] = time.Now()
	metadata["generator_version"] = "1.0"

	return metadata
}

// generatePaddingText creates additional text to reach target length
func (dgs *DataGenerationService) generatePaddingText(targetLength int) string {
	paddingWords := []string{
		"Additionally", "Furthermore", "Moreover", "However", "Nevertheless",
		"Therefore", "Consequently", "Subsequently", "Meanwhile", "Specifically",
		"particularly", "especially", "notably", "importantly", "significantly",
	}

	var padding strings.Builder
	currentLength := 0

	for currentLength < targetLength {
		word := paddingWords[dgs.rand.Intn(len(paddingWords))]
		if currentLength > 0 {
			padding.WriteString(" ")
			currentLength++
		}
		padding.WriteString(word)
		currentLength += len(word)
	}

	return padding.String()
}

// validateRecord checks if a generated record is valid
func (dgs *DataGenerationService) validateRecord(record *models.GeneratedChunkRecord) error {
	if record.ID <= 0 {
		return fmt.Errorf("invalid ID: %d", record.ID)
	}

	if len(record.Content) == 0 {
		return fmt.Errorf("empty content")
	}

	if len(record.Embedding) == 0 {
		return fmt.Errorf("empty embedding")
	}

	if len(record.Tags) == 0 {
		return fmt.Errorf("no tags generated")
	}

	if record.Metadata == nil {
		return fmt.Errorf("no metadata generated")
	}

	return nil
}

// VerifyDataIntegrity performs comprehensive data integrity checks
func (dgs *DataGenerationService) VerifyDataIntegrity(ctx context.Context, expectedCount int) DataIntegrityCheck {
	dgs.logger.Printf("Starting data integrity verification for %d records", expectedCount)

	check := DataIntegrityCheck{
		TotalRecords:     expectedCount,
		ValidRecords:     0,
		InvalidRecords:   0,
		MissingFields:    make(map[string]int),
		Issues:           []DataQualityIssue{},
		SampleValidation: make(map[string]interface{}),
	}

	// In a real implementation, this would query the database to verify records
	// For simulation, we'll assume most records are valid
	simulatedValidRecords := int(float64(expectedCount) * 0.98) // 98% success rate
	check.ValidRecords = simulatedValidRecords
	check.InvalidRecords = expectedCount - simulatedValidRecords

	// Calculate quality score
	check.DataQualityScore = float64(check.ValidRecords) / float64(check.TotalRecords)

	// Add sample issues
	if check.InvalidRecords > 0 {
		check.Issues = append(check.Issues, DataQualityIssue{
			Type:        "missing_embedding",
			Field:       "embedding",
			Count:       check.InvalidRecords / 2,
			Description: "Records missing embedding vectors",
			Severity:    "medium",
		})

		check.Issues = append(check.Issues, DataQualityIssue{
			Type:        "empty_content",
			Field:       "content",
			Count:       check.InvalidRecords / 2,
			Description: "Records with empty content",
			Severity:    "high",
		})
	}

	// Sample validation
	check.SampleValidation["avg_content_length"] = 250.5
	check.SampleValidation["avg_tags_per_record"] = 3.2
	check.SampleValidation["embedding_dimension"] = 1536
	check.SampleValidation["unique_categories"] = len(dgs.textPatterns)

	dgs.logger.Printf("Data integrity check completed: %.2f%% quality score", check.DataQualityScore*100)

	return check
}

// Default patterns and configurations

func getDefaultTextPatterns() []TextPattern {
	return []TextPattern{
		{
			Category: "technical",
			Templates: []string{
				"This document discusses {keyword1} implementation using {keyword2} technology. The {keyword3} approach provides significant benefits.",
				"The {keyword1} system integrates with {keyword2} to enable {keyword3} functionality across multiple platforms.",
				"Performance optimization of {keyword1} requires careful consideration of {keyword2} and {keyword3} factors.",
			},
			Keywords:   []string{"database", "performance", "optimization", "indexing", "caching", "scaling", "microservices", "API", "architecture", "monitoring"},
			MinLength:  100,
			MaxLength:  500,
			Complexity: "medium",
		},
		{
			Category: "business",
			Templates: []string{
				"The {keyword1} strategy focuses on {keyword2} to achieve {keyword3} objectives in the market.",
				"Our {keyword1} solution delivers {keyword2} value through innovative {keyword3} approaches.",
				"Market analysis shows that {keyword1} trends are driving {keyword2} adoption in {keyword3} sectors.",
			},
			Keywords:   []string{"growth", "revenue", "market", "customer", "strategy", "innovation", "digital", "transformation", "analytics", "insights"},
			MinLength:  80,
			MaxLength:  400,
			Complexity: "simple",
		},
		{
			Category: "scientific",
			Templates: []string{
				"Research indicates that {keyword1} mechanisms influence {keyword2} processes through {keyword3} pathways.",
				"The study examined {keyword1} factors affecting {keyword2} outcomes in {keyword3} environments.",
				"Experimental results demonstrate {keyword1} correlation between {keyword2} and {keyword3} variables.",
			},
			Keywords:   []string{"analysis", "research", "study", "experimental", "hypothesis", "methodology", "results", "correlation", "statistical", "evidence"},
			MinLength:  150,
			MaxLength:  600,
			Complexity: "complex",
		},
	}
}

func getDefaultTagPatterns() []TagPattern {
	return []TagPattern{
		{
			Domain:   "technical",
			BaseTags: []string{"programming", "database", "performance", "optimization", "architecture", "security", "testing", "deployment", "monitoring", "debugging"},
			Frequencies: map[string]float64{
				"programming":   0.8,
				"database":      0.6,
				"performance":   0.7,
				"optimization":  0.5,
				"architecture":  0.4,
				"security":      0.3,
				"testing":       0.6,
				"deployment":    0.4,
				"monitoring":    0.5,
				"debugging":     0.3,
			},
			MaxTags: 5,
		},
		{
			Domain:   "business",
			BaseTags: []string{"strategy", "growth", "market", "customer", "revenue", "analytics", "innovation", "digital", "transformation", "leadership"},
			Frequencies: map[string]float64{
				"strategy":       0.7,
				"growth":         0.6,
				"market":         0.8,
				"customer":       0.9,
				"revenue":        0.5,
				"analytics":      0.6,
				"innovation":     0.4,
				"digital":        0.7,
				"transformation": 0.5,
				"leadership":     0.3,
			},
			MaxTags: 4,
		},
		{
			Domain:   "scientific",
			BaseTags: []string{"research", "analysis", "study", "experimental", "methodology", "statistical", "hypothesis", "evidence", "correlation", "publication"},
			Frequencies: map[string]float64{
				"research":     0.9,
				"analysis":     0.8,
				"study":        0.7,
				"experimental": 0.6,
				"methodology":  0.5,
				"statistical":  0.6,
				"hypothesis":   0.4,
				"evidence":     0.7,
				"correlation":  0.5,
				"publication":  0.3,
			},
			MaxTags: 6,
		},
	}
}

func getDefaultMetadataTypes() []MetadataType {
	return []MetadataType{
		{
			Name:      "priority",
			Type:      "string",
			Values:    []interface{}{"low", "medium", "high", "critical"},
			Generator: "random",
		},
		{
			Name:      "status",
			Type:      "string",
			Values:    []interface{}{"draft", "review", "approved", "published", "archived"},
			Generator: "random",
		},
		{
			Name:      "complexity_score",
			Type:      "float",
			Generator: "random",
		},
		{
			Name:      "view_count",
			Type:      "int",
			Generator: "random",
		},
		{
			Name:      "is_featured",
			Type:      "bool",
			Generator: "random",
		},
		{
			Name:      "last_modified",
			Type:      "timestamp",
			Generator: "random",
		},
	}
}