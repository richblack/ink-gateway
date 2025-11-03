package services

import (
	"context"
	"fmt"
	"regexp"
	"semantic-text-processor/models"
	"strings"
	"time"

	"github.com/google/uuid"
)

// TextProcessorImpl implements TextProcessor interface
type TextProcessorImpl struct {
	llmService       LLMService
	embeddingService EmbeddingService
}

// NewTextProcessor creates a new text processor
func NewTextProcessor(llmService LLMService, embeddingService EmbeddingService) *TextProcessorImpl {
	return &TextProcessorImpl{
		llmService:       llmService,
		embeddingService: embeddingService,
	}
}

// ProcessText implements TextProcessor.ProcessText
func (tp *TextProcessorImpl) ProcessText(ctx context.Context, text string) (*models.ProcessResult, error) {
	// Generate text ID
	textID := uuid.New().String()
	
	// Chunk the text using LLM
	chunks, err := tp.ChunkText(ctx, text)
	if err != nil {
		return nil, fmt.Errorf("failed to chunk text: %w", err)
	}
	
	// Set text ID for all chunks
	for i := range chunks {
		chunks[i].TextID = textID
	}
	
	return &models.ProcessResult{
		TextID:      textID,
		Chunks:      chunks,
		Status:      "completed",
		ProcessedAt: time.Now(),
	}, nil
}

// ChunkText implements TextProcessor.ChunkText
func (tp *TextProcessorImpl) ChunkText(ctx context.Context, text string) ([]models.ChunkRecord, error) {
	// Use LLM to get raw chunks
	rawChunks, err := tp.llmService.ChunkText(ctx, text)
	if err != nil {
		return nil, fmt.Errorf("LLM chunking failed: %w", err)
	}
	
	// Parse hierarchical structure and convert to ChunkRecord
	chunks := tp.parseHierarchicalStructure(rawChunks)
	
	return chunks, nil
}

// GenerateEmbeddings implements TextProcessor.GenerateEmbeddings
func (tp *TextProcessorImpl) GenerateEmbeddings(ctx context.Context, chunks []models.ChunkRecord) ([]models.EmbeddingRecord, error) {
	if tp.embeddingService == nil {
		return nil, fmt.Errorf("embedding service not available")
	}
	
	// Extract text content from chunks
	texts := make([]string, len(chunks))
	for i, chunk := range chunks {
		texts[i] = chunk.Content
	}
	
	// Generate embeddings in batch
	vectors, err := tp.embeddingService.GenerateBatchEmbeddings(ctx, texts)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embeddings: %w", err)
	}
	
	// Create embedding records
	embeddings := make([]models.EmbeddingRecord, len(chunks))
	for i, chunk := range chunks {
		embeddings[i] = models.EmbeddingRecord{
			ID:        uuid.New().String(),
			ChunkID:   chunk.ID,
			Vector:    vectors[i],
			CreatedAt: time.Now(),
		}
	}
	
	return embeddings, nil
}

// ExtractKnowledge implements TextProcessor.ExtractKnowledge
func (tp *TextProcessorImpl) ExtractKnowledge(ctx context.Context, chunks []models.ChunkRecord) (*models.GraphResult, error) {
	var allNodes []models.GraphNode
	var allEdges []models.GraphEdge
	
	// Extract entities from each chunk
	for _, chunk := range chunks {
		nodes, err := tp.llmService.ExtractEntities(ctx, chunk.Content)
		if err != nil {
			return nil, fmt.Errorf("failed to extract entities from chunk %s: %w", chunk.ID, err)
		}
		
		// Set chunk ID for nodes
		for i := range nodes {
			nodes[i].ID = uuid.New().String()
			nodes[i].ChunkID = chunk.ID
		}
		
		allNodes = append(allNodes, nodes...)
	}
	
	// Generate edges based on co-occurrence and relationships
	edges := tp.generateKnowledgeEdges(allNodes, chunks)
	allEdges = append(allEdges, edges...)
	
	return &models.GraphResult{
		Nodes: allNodes,
		Edges: allEdges,
	}, nil
}

// parseHierarchicalStructure parses text chunks and creates hierarchical structure
func (tp *TextProcessorImpl) parseHierarchicalStructure(rawChunks []string) []models.ChunkRecord {
	var chunks []models.ChunkRecord
	var parentStack []*models.ChunkRecord
	
	for _, rawChunk := range rawChunks {
		chunk := tp.createChunkFromText(rawChunk)
		
		// Determine indent level and hierarchy
		indentLevel := tp.calculateIndentLevel(rawChunk)
		chunk.IndentLevel = indentLevel
		
		// Handle parent-child relationships
		if indentLevel == 0 {
			// Root level chunk
			parentStack = []*models.ChunkRecord{&chunk}
			chunk.ParentChunkID = nil
		} else {
			// Find appropriate parent
			if indentLevel <= len(parentStack) {
				// Adjust stack to current level
				parentStack = parentStack[:indentLevel]
			}
			
			if len(parentStack) > 0 {
				parent := parentStack[len(parentStack)-1]
				chunk.ParentChunkID = &parent.ID
			}
			
			// Add to stack
			parentStack = append(parentStack, &chunk)
		}
		
		// Set sequence number within the same parent
		chunk.SequenceNumber = tp.calculateSequenceNumber(chunks, chunk.ParentChunkID)
		
		chunks = append(chunks, chunk)
	}
	
	return chunks
}

// createChunkFromText creates a ChunkRecord from raw text
func (tp *TextProcessorImpl) createChunkFromText(text string) models.ChunkRecord {
	chunk := models.ChunkRecord{
		ID:        uuid.New().String(),
		Content:   strings.TrimSpace(text),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Metadata:  make(map[string]interface{}),
	}
	
	// Detect special chunk types
	tp.detectChunkType(&chunk)
	
	return chunk
}

// detectChunkType detects if chunk is template, slot, etc.
func (tp *TextProcessorImpl) detectChunkType(chunk *models.ChunkRecord) {
	content := chunk.Content
	
	// Check for template marker
	if strings.Contains(content, "#template") {
		chunk.IsTemplate = true
		chunk.Metadata["template_marker"] = true
		// Remove template marker from content
		chunk.Content = strings.ReplaceAll(content, "#template", "")
		chunk.Content = strings.TrimSpace(chunk.Content)
	}
	
	// Check for slot marker
	if strings.Contains(content, "#slot") {
		chunk.IsSlot = true
		chunk.Metadata["slot_marker"] = true
		// Extract slot name if present
		slotName := tp.extractSlotName(content)
		if slotName != "" {
			chunk.Metadata["slot_name"] = slotName
		}
		// Remove slot marker from content
		chunk.Content = strings.ReplaceAll(content, "#slot", "")
		chunk.Content = strings.TrimSpace(chunk.Content)
	}
	
	// Detect bullet point type
	bulletType := tp.detectBulletType(content)
	if bulletType != "" {
		chunk.Metadata["bullet_type"] = bulletType
	}
}

// calculateIndentLevel determines the indentation level of text
func (tp *TextProcessorImpl) calculateIndentLevel(text string) int {
	// Count leading spaces/tabs
	leadingSpaces := 0
	for _, char := range text {
		if char == ' ' {
			leadingSpaces++
		} else if char == '\t' {
			leadingSpaces += 4 // Treat tab as 4 spaces
		} else {
			break
		}
	}
	
	// Convert to indent level (every 2 spaces = 1 level)
	spaceLevel := leadingSpaces / 2
	
	// Check for bullet point patterns
	bulletLevel := tp.detectBulletIndentLevel(text)
	
	// If it's a bullet point, add the bullet level to the space level
	if bulletLevel > 0 {
		return spaceLevel + bulletLevel
	}
	
	return spaceLevel
}

// detectBulletIndentLevel detects indent level from bullet patterns
func (tp *TextProcessorImpl) detectBulletIndentLevel(text string) int {
	trimmed := strings.TrimSpace(text)
	
	// Check if it's a bullet point at all
	patterns := []string{
		`^[-*+]\s`,           // - * +
		`^\d+\.\s`,           // 1. 2. 3.
		`^[a-z]\)\s`,         // a) b) c)
		`^[ivx]+\.\s`,        // i. ii. iii.
		`^[A-Z]\.\s`,         // A. B. C.
		`^[IVX]+\.\s`,        // I. II. III.
	}
	
	isBullet := false
	for _, pattern := range patterns {
		matched, _ := regexp.MatchString(pattern, trimmed)
		if matched {
			isBullet = true
			break
		}
	}
	
	if !isBullet {
		return 0
	}
	
	// If it's a bullet, return 1 (bullets are always at least level 1)
	return 1
}

// detectBulletType identifies the type of bullet point
func (tp *TextProcessorImpl) detectBulletType(text string) string {
	trimmed := strings.TrimSpace(text)
	
	patterns := map[string]string{
		`^[-]\s`:      "dash",
		`^[*]\s`:      "asterisk", 
		`^[+]\s`:      "plus",
		`^\d+\.\s`:    "numbered",
		`^[a-z]\)\s`:  "letter_lower",
		`^[A-Z]\.\s`:  "letter_upper",
		`^[ivx]+\.\s`: "roman_lower",
		`^[IVX]+\.\s`: "roman_upper",
	}
	
	for pattern, bulletType := range patterns {
		matched, _ := regexp.MatchString(pattern, trimmed)
		if matched {
			return bulletType
		}
	}
	
	return ""
}

// extractSlotName extracts slot name from slot marker
func (tp *TextProcessorImpl) extractSlotName(content string) string {
	// Look for patterns like "#slot:name" or "#slot name"
	patterns := []string{
		`#slot:(\w+)`,
		`#slot\s+(\w+)`,
	}
	
	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(content)
		if len(matches) > 1 {
			return matches[1]
		}
	}
	
	return ""
}

// calculateSequenceNumber calculates sequence number within parent
func (tp *TextProcessorImpl) calculateSequenceNumber(existingChunks []models.ChunkRecord, parentID *string) *int {
	count := 0
	
	for _, chunk := range existingChunks {
		// Count chunks with same parent
		if (parentID == nil && chunk.ParentChunkID == nil) ||
		   (parentID != nil && chunk.ParentChunkID != nil && *chunk.ParentChunkID == *parentID) {
			count++
		}
	}
	
	return &count
}

// generateKnowledgeEdges creates edges between entities based on relationships
func (tp *TextProcessorImpl) generateKnowledgeEdges(nodes []models.GraphNode, chunks []models.ChunkRecord) []models.GraphEdge {
	var edges []models.GraphEdge
	
	// Group nodes by chunk for co-occurrence analysis
	chunkNodes := make(map[string][]models.GraphNode)
	for _, node := range nodes {
		chunkNodes[node.ChunkID] = append(chunkNodes[node.ChunkID], node)
	}
	
	// Create co-occurrence edges within each chunk
	for chunkID, chunkNodeList := range chunkNodes {
		for i, node1 := range chunkNodeList {
			for j, node2 := range chunkNodeList {
				if i >= j {
					continue // Avoid duplicate and self-edges
				}
				
				edge := models.GraphEdge{
					ID:               uuid.New().String(),
					SourceNodeID:     node1.ID,
					TargetNodeID:     node2.ID,
					RelationshipType: "co_occurs_with",
					Properties: map[string]interface{}{
						"chunk_id":   chunkID,
						"confidence": 0.7,
						"source":     "co_occurrence",
					},
					CreatedAt: time.Now(),
				}
				
				edges = append(edges, edge)
			}
		}
	}
	
	// Create hierarchical relationships based on chunk hierarchy
	for _, chunk := range chunks {
		if chunk.ParentChunkID != nil {
			// Find nodes in parent and child chunks
			parentNodes := chunkNodes[*chunk.ParentChunkID]
			childNodes := chunkNodes[chunk.ID]
			
			// Create hierarchical edges
			for _, parentNode := range parentNodes {
				for _, childNode := range childNodes {
					edge := models.GraphEdge{
						ID:               uuid.New().String(),
						SourceNodeID:     parentNode.ID,
						TargetNodeID:     childNode.ID,
						RelationshipType: "contains",
						Properties: map[string]interface{}{
							"parent_chunk_id": *chunk.ParentChunkID,
							"child_chunk_id":  chunk.ID,
							"confidence":      0.8,
							"source":          "hierarchy",
						},
						CreatedAt: time.Now(),
					}
					
					edges = append(edges, edge)
				}
			}
		}
	}
	
	return edges
}