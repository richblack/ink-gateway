package migration

import (
	"context"
	"fmt"
	"log"
	"time"

	"semantic-text-processor/models"
)

// CompatibilityLayer provides backward compatibility with the old API
type CompatibilityLayer struct {
	legacyAdapter    *LegacyAPIAdapter
	unifiedService   UnifiedChunkService
	versionManager   *APIVersionManager
	deprecationMgr   *DeprecationManager
	logger           *log.Logger
}

// UnifiedChunkService interface for the new unified chunk service
type UnifiedChunkService interface {
	CreateChunk(ctx context.Context, chunk *models.UnifiedChunkRecord) error
	GetChunk(ctx context.Context, id string) (*models.UnifiedChunkRecord, error)
	UpdateChunk(ctx context.Context, chunk *models.UnifiedChunkRecord) error
	DeleteChunk(ctx context.Context, id string) error
	SearchChunks(ctx context.Context, query *models.SearchQuery) (*models.SearchResult, error)
	AddTags(ctx context.Context, chunkID string, tags []string) error
	RemoveTags(ctx context.Context, chunkID string, tags []string) error
	GetChunkTags(ctx context.Context, chunkID string) ([]models.UnifiedChunkRecord, error)
	GetChunksByTag(ctx context.Context, tagContent string) ([]models.UnifiedChunkRecord, error)
	MoveChunk(ctx context.Context, req *models.MoveChunkRequest) error
	GetChunkHierarchy(ctx context.Context, rootChunkID string) (*models.UnifiedChunkHierarchy, error)
}

// LegacyAPIAdapter adapts old API calls to new unified service
type LegacyAPIAdapter struct {
	transformers map[string]ResponseTransformer
	validators   map[string]RequestValidator
	rateLimiter  *RateLimiter
	logger       *log.Logger
}

// ResponseTransformer transforms responses between old and new formats
type ResponseTransformer interface {
	TransformToLegacy(unified interface{}) (interface{}, error)
	TransformFromLegacy(legacy interface{}) (interface{}, error)
}

// RequestValidator validates requests for backward compatibility
type RequestValidator interface {
	ValidateRequest(request interface{}) error
	SanitizeRequest(request interface{}) (interface{}, error)
}

// APIVersionManager manages API versioning and deprecation
type APIVersionManager struct {
	supportedVersions   map[string]bool
	defaultVersion      string
	deprecationWarnings map[string]DeprecationInfo
	logger              *log.Logger
}

// DeprecationManager handles deprecation warnings and tracking
type DeprecationManager struct {
	deprecationWarnings map[string]DeprecationInfo
	warningCounts       map[string]int64
	logger              *log.Logger
}

// DeprecationInfo contains information about deprecated features
type DeprecationInfo struct {
	Feature         string    `json:"feature"`
	DeprecatedSince string    `json:"deprecated_since"`
	RemovalVersion  string    `json:"removal_version"`
	Replacement     string    `json:"replacement"`
	WarningMessage  string    `json:"warning_message"`
	DocumentationURL string   `json:"documentation_url"`
}

// RateLimiter provides rate limiting for legacy API calls
type RateLimiter struct {
	// Implementation details would go here
}

// NewCompatibilityLayer creates a new compatibility layer
func NewCompatibilityLayer(unifiedService UnifiedChunkService) *CompatibilityLayer {
	logger := log.New(log.Writer(), "[COMPATIBILITY_LAYER] ", log.LstdFlags|log.Lshortfile)

	return &CompatibilityLayer{
		legacyAdapter:  NewLegacyAPIAdapter(),
		unifiedService: unifiedService,
		versionManager: NewAPIVersionManager(),
		deprecationMgr: NewDeprecationManager(),
		logger:         logger,
	}
}

// NewLegacyAPIAdapter creates a new legacy API adapter
func NewLegacyAPIAdapter() *LegacyAPIAdapter {
	adapter := &LegacyAPIAdapter{
		transformers: make(map[string]ResponseTransformer),
		validators:   make(map[string]RequestValidator),
		logger:       log.New(log.Writer(), "[LEGACY_ADAPTER] ", log.LstdFlags|log.Lshortfile),
	}

	// Register transformers for different entity types
	adapter.transformers["chunk"] = &ChunkTransformer{}
	adapter.transformers["text"] = &TextTransformer{}
	adapter.transformers["tag"] = &TagTransformer{}
	adapter.transformers["template"] = &TemplateTransformer{}

	// Register validators
	adapter.validators["chunk"] = &ChunkValidator{}
	adapter.validators["text"] = &TextValidator{}

	return adapter
}

// NewAPIVersionManager creates a new API version manager
func NewAPIVersionManager() *APIVersionManager {
	return &APIVersionManager{
		supportedVersions: map[string]bool{
			"v1": true,
			"v2": false, // New unified API version
		},
		defaultVersion: "v1",
		deprecationWarnings: map[string]DeprecationInfo{
			"old_chunk_api": {
				Feature:         "Legacy Chunk API",
				DeprecatedSince: "v1.5.0",
				RemovalVersion:  "v2.0.0",
				Replacement:     "Unified Chunk API v2",
				WarningMessage:  "This API version is deprecated. Please migrate to the unified API.",
				DocumentationURL: "https://docs.example.com/migration-guide",
			},
		},
		logger: log.New(log.Writer(), "[API_VERSION_MANAGER] ", log.LstdFlags|log.Lshortfile),
	}
}

// NewDeprecationManager creates a new deprecation manager
func NewDeprecationManager() *DeprecationManager {
	return &DeprecationManager{
		deprecationWarnings: make(map[string]DeprecationInfo),
		warningCounts:       make(map[string]int64),
		logger:              log.New(log.Writer(), "[DEPRECATION_MANAGER] ", log.LstdFlags|log.Lshortfile),
	}
}

// Legacy API Methods - maintaining compatibility with existing SupabaseClient interface

// InsertChunk maintains compatibility with legacy chunk insertion
func (cl *CompatibilityLayer) InsertChunk(ctx context.Context, chunk *models.ChunkRecord) error {
	cl.logger.Printf("Legacy InsertChunk called for chunk: %s", chunk.ID)

	// Add deprecation warning
	cl.deprecationMgr.RecordWarning("insert_chunk", "InsertChunk is deprecated, use CreateChunk instead")

	// Validate legacy request
	if err := cl.legacyAdapter.validators["chunk"].ValidateRequest(chunk); err != nil {
		return fmt.Errorf("legacy chunk validation failed: %w", err)
	}

	// Transform legacy chunk to unified format
	unifiedChunk, err := cl.transformLegacyChunkToUnified(chunk)
	if err != nil {
		return fmt.Errorf("failed to transform legacy chunk: %w", err)
	}

	// Call unified service
	if err := cl.unifiedService.CreateChunk(ctx, unifiedChunk); err != nil {
		return fmt.Errorf("unified service create chunk failed: %w", err)
	}

	// Update the original chunk with any changes (like generated IDs)
	cl.updateLegacyChunkFromUnified(chunk, unifiedChunk)

	return nil
}

// GetChunkByID maintains compatibility with legacy chunk retrieval
func (cl *CompatibilityLayer) GetChunkByID(ctx context.Context, id string) (*models.ChunkRecord, error) {
	cl.logger.Printf("Legacy GetChunkByID called for id: %s", id)

	// Add deprecation warning
	cl.deprecationMgr.RecordWarning("get_chunk_by_id", "GetChunkByID is deprecated, use GetChunk instead")

	// Call unified service
	unifiedChunk, err := cl.unifiedService.GetChunk(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("unified service get chunk failed: %w", err)
	}

	// Transform unified chunk to legacy format
	legacyChunk, err := cl.transformUnifiedChunkToLegacy(unifiedChunk)
	if err != nil {
		return nil, fmt.Errorf("failed to transform unified chunk to legacy: %w", err)
	}

	return legacyChunk, nil
}

// UpdateChunk maintains compatibility with legacy chunk updates
func (cl *CompatibilityLayer) UpdateChunk(ctx context.Context, chunk *models.ChunkRecord) error {
	cl.logger.Printf("Legacy UpdateChunk called for chunk: %s", chunk.ID)

	// Add deprecation warning
	cl.deprecationMgr.RecordWarning("update_chunk", "UpdateChunk is deprecated, use UpdateChunk v2 instead")

	// Transform to unified format
	unifiedChunk, err := cl.transformLegacyChunkToUnified(chunk)
	if err != nil {
		return fmt.Errorf("failed to transform legacy chunk: %w", err)
	}

	// Call unified service
	if err := cl.unifiedService.UpdateChunk(ctx, unifiedChunk); err != nil {
		return fmt.Errorf("unified service update chunk failed: %w", err)
	}

	return nil
}

// DeleteChunk maintains compatibility with legacy chunk deletion
func (cl *CompatibilityLayer) DeleteChunk(ctx context.Context, id string) error {
	cl.logger.Printf("Legacy DeleteChunk called for id: %s", id)

	// Add deprecation warning
	cl.deprecationMgr.RecordWarning("delete_chunk", "DeleteChunk is deprecated, use DeleteChunk v2 instead")

	// Call unified service directly (no transformation needed for deletion)
	return cl.unifiedService.DeleteChunk(ctx, id)
}

// GetChunksByTextID maintains compatibility with text-based chunk retrieval
func (cl *CompatibilityLayer) GetChunksByTextID(ctx context.Context, textID string) ([]models.ChunkRecord, error) {
	cl.logger.Printf("Legacy GetChunksByTextID called for textID: %s", textID)

	// Add deprecation warning
	cl.deprecationMgr.RecordWarning("get_chunks_by_text_id", "GetChunksByTextID is deprecated, use SearchChunks instead")

	// Create search query for unified service
	searchQuery := &models.SearchQuery{
		Metadata: map[string]interface{}{
			"original_text_id": textID,
		},
		IsPage: func() *bool { b := false; return &b }(), // Exclude page chunks to match old behavior
		Limit:  1000, // Reasonable default
	}

	// Call unified service
	searchResult, err := cl.unifiedService.SearchChunks(ctx, searchQuery)
	if err != nil {
		return nil, fmt.Errorf("unified service search failed: %w", err)
	}

	// Transform results to legacy format
	legacyChunks := make([]models.ChunkRecord, 0, len(searchResult.Chunks))
	for _, unifiedChunk := range searchResult.Chunks {
		legacyChunk, err := cl.transformUnifiedChunkToLegacy(&unifiedChunk)
		if err != nil {
			cl.logger.Printf("Failed to transform chunk %s: %v", unifiedChunk.ChunkID, err)
			continue
		}
		legacyChunks = append(legacyChunks, *legacyChunk)
	}

	return legacyChunks, nil
}

// AddTag maintains compatibility with legacy tag operations
func (cl *CompatibilityLayer) AddTag(ctx context.Context, chunkID string, tagContent string) error {
	cl.logger.Printf("Legacy AddTag called for chunk: %s, tag: %s", chunkID, tagContent)

	// Add deprecation warning
	cl.deprecationMgr.RecordWarning("add_tag", "AddTag is deprecated, use AddTags instead")

	// Use unified service
	return cl.unifiedService.AddTags(ctx, chunkID, []string{tagContent})
}

// RemoveTag maintains compatibility with legacy tag removal
func (cl *CompatibilityLayer) RemoveTag(ctx context.Context, chunkID string, tagChunkID string) error {
	cl.logger.Printf("Legacy RemoveTag called for chunk: %s, tagChunk: %s", chunkID, tagChunkID)

	// Add deprecation warning
	cl.deprecationMgr.RecordWarning("remove_tag", "RemoveTag is deprecated, use RemoveTags instead")

	// For legacy compatibility, we need to find the tag content from tag chunk ID
	// This is a simplified approach - in practice, you might need more sophisticated mapping
	return cl.unifiedService.RemoveTags(ctx, chunkID, []string{tagChunkID})
}

// GetChunkTags maintains compatibility with legacy tag retrieval
func (cl *CompatibilityLayer) GetChunkTags(ctx context.Context, chunkID string) ([]models.ChunkRecord, error) {
	cl.logger.Printf("Legacy GetChunkTags called for chunk: %s", chunkID)

	// Add deprecation warning
	cl.deprecationMgr.RecordWarning("get_chunk_tags", "GetChunkTags is deprecated, use GetChunkTags v2 instead")

	// Get unified tags
	unifiedTags, err := cl.unifiedService.GetChunkTags(ctx, chunkID)
	if err != nil {
		return nil, fmt.Errorf("unified service get chunk tags failed: %w", err)
	}

	// Transform to legacy format
	legacyTags := make([]models.ChunkRecord, 0, len(unifiedTags))
	for _, unifiedTag := range unifiedTags {
		legacyTag, err := cl.transformUnifiedChunkToLegacy(&unifiedTag)
		if err != nil {
			cl.logger.Printf("Failed to transform tag %s: %v", unifiedTag.ChunkID, err)
			continue
		}
		legacyTags = append(legacyTags, *legacyTag)
	}

	return legacyTags, nil
}

// GetChunksByTag maintains compatibility with legacy tag-based search
func (cl *CompatibilityLayer) GetChunksByTag(ctx context.Context, tagContent string) ([]models.ChunkRecord, error) {
	cl.logger.Printf("Legacy GetChunksByTag called for tag: %s", tagContent)

	// Add deprecation warning
	cl.deprecationMgr.RecordWarning("get_chunks_by_tag", "GetChunksByTag is deprecated, use SearchChunks instead")

	// Get unified chunks by tag
	unifiedChunks, err := cl.unifiedService.GetChunksByTag(ctx, tagContent)
	if err != nil {
		return nil, fmt.Errorf("unified service get chunks by tag failed: %w", err)
	}

	// Transform to legacy format
	legacyChunks := make([]models.ChunkRecord, 0, len(unifiedChunks))
	for _, unifiedChunk := range unifiedChunks {
		legacyChunk, err := cl.transformUnifiedChunkToLegacy(&unifiedChunk)
		if err != nil {
			cl.logger.Printf("Failed to transform chunk %s: %v", unifiedChunk.ChunkID, err)
			continue
		}
		legacyChunks = append(legacyChunks, *legacyChunk)
	}

	return legacyChunks, nil
}

// Transformation methods

// transformLegacyChunkToUnified converts legacy ChunkRecord to UnifiedChunkRecord
func (cl *CompatibilityLayer) transformLegacyChunkToUnified(legacy *models.ChunkRecord) (*models.UnifiedChunkRecord, error) {
	metadata := make(map[string]interface{})

	// Copy existing metadata
	if legacy.Metadata != nil {
		for k, v := range legacy.Metadata {
			metadata[k] = v
		}
	}

	// Add legacy-specific fields to metadata
	metadata["original_text_id"] = legacy.TextID
	if legacy.TemplateChunkID != nil {
		metadata["template_chunk_id"] = *legacy.TemplateChunkID
	}
	if legacy.SlotValue != nil {
		metadata["slot_value"] = *legacy.SlotValue
	}
	metadata["indent_level"] = legacy.IndentLevel
	if legacy.SequenceNumber != nil {
		metadata["sequence_number"] = *legacy.SequenceNumber
	}

	// Determine type flags
	isPage := legacy.TextID != "" && legacy.ParentChunkID == nil && !legacy.IsTemplate && !legacy.IsSlot
	isTag := cl.determineIfTag(legacy.Content)

	unified := &models.UnifiedChunkRecord{
		ChunkID:     legacy.ID,
		Contents:    legacy.Content,
		Parent:      legacy.ParentChunkID,
		Page:        cl.derivePageFromTextID(legacy.TextID),
		IsPage:      isPage,
		IsTag:       isTag,
		IsTemplate:  legacy.IsTemplate,
		IsSlot:      legacy.IsSlot,
		Ref:         &legacy.TextID,
		Tags:        []string{}, // Will be populated from relationships
		Metadata:    metadata,
		CreatedTime: legacy.CreatedAt,
		LastUpdated: legacy.UpdatedAt,
	}

	return unified, nil
}

// transformUnifiedChunkToLegacy converts UnifiedChunkRecord to legacy ChunkRecord
func (cl *CompatibilityLayer) transformUnifiedChunkToLegacy(unified *models.UnifiedChunkRecord) (*models.ChunkRecord, error) {
	legacy := &models.ChunkRecord{
		ID:         unified.ChunkID,
		Content:    unified.Contents,
		IsTemplate: unified.IsTemplate,
		IsSlot:     unified.IsSlot,
		CreatedAt:  unified.CreatedTime,
		UpdatedAt:  unified.LastUpdated,
		Metadata:   make(map[string]interface{}),
	}

	// Extract legacy fields from metadata
	if unified.Metadata != nil {
		// Copy metadata, excluding our special fields
		for k, v := range unified.Metadata {
			switch k {
			case "original_text_id":
				if textID, ok := v.(string); ok {
					legacy.TextID = textID
				}
			case "template_chunk_id":
				if templateID, ok := v.(string); ok {
					legacy.TemplateChunkID = &templateID
				}
			case "slot_value":
				if slotVal, ok := v.(string); ok {
					legacy.SlotValue = &slotVal
				}
			case "indent_level":
				if indent, ok := v.(float64); ok {
					legacy.IndentLevel = int(indent)
				} else if indent, ok := v.(int); ok {
					legacy.IndentLevel = indent
				}
			case "sequence_number":
				if seq, ok := v.(float64); ok {
					seqInt := int(seq)
					legacy.SequenceNumber = &seqInt
				} else if seq, ok := v.(int); ok {
					legacy.SequenceNumber = &seq
				}
			default:
				legacy.Metadata[k] = v
			}
		}
	}

	// Set parent and text ID
	legacy.ParentChunkID = unified.Parent
	if unified.Ref != nil {
		legacy.TextID = *unified.Ref
	}

	return legacy, nil
}

// Helper methods

func (cl *CompatibilityLayer) determineIfTag(content string) bool {
	// Simple heuristic for tag detection
	return len(content) < 50 && (content[0] == '#' || len(content) < 20)
}

func (cl *CompatibilityLayer) derivePageFromTextID(textID string) *string {
	// In the legacy system, we can use textID as page reference
	// In a real implementation, you might want to look up the actual page chunk
	return &textID
}

func (cl *CompatibilityLayer) updateLegacyChunkFromUnified(legacy *models.ChunkRecord, unified *models.UnifiedChunkRecord) {
	// Update any generated fields
	legacy.UpdatedAt = unified.LastUpdated
	if legacy.ID == "" {
		legacy.ID = unified.ChunkID
	}
}

// RecordWarning records a deprecation warning
func (dm *DeprecationManager) RecordWarning(feature, message string) {
	dm.warningCounts[feature]++

	if dm.warningCounts[feature]%100 == 1 { // Log every 100th usage to avoid spam
		dm.logger.Printf("DEPRECATION WARNING [%s] (usage count: %d): %s",
			feature, dm.warningCounts[feature], message)
	}
}

// GetDeprecationStats returns deprecation usage statistics
func (dm *DeprecationManager) GetDeprecationStats() map[string]int64 {
	stats := make(map[string]int64)
	for feature, count := range dm.warningCounts {
		stats[feature] = count
	}
	return stats
}

// Transformer implementations

type ChunkTransformer struct{}

func (ct *ChunkTransformer) TransformToLegacy(unified interface{}) (interface{}, error) {
	// Implementation would transform unified chunk to legacy format
	return unified, nil
}

func (ct *ChunkTransformer) TransformFromLegacy(legacy interface{}) (interface{}, error) {
	// Implementation would transform legacy chunk to unified format
	return legacy, nil
}

type TextTransformer struct{}

func (tt *TextTransformer) TransformToLegacy(unified interface{}) (interface{}, error) {
	return unified, nil
}

func (tt *TextTransformer) TransformFromLegacy(legacy interface{}) (interface{}, error) {
	return legacy, nil
}

type TagTransformer struct{}

func (tt *TagTransformer) TransformToLegacy(unified interface{}) (interface{}, error) {
	return unified, nil
}

func (tt *TagTransformer) TransformFromLegacy(legacy interface{}) (interface{}, error) {
	return legacy, nil
}

type TemplateTransformer struct{}

func (tt *TemplateTransformer) TransformToLegacy(unified interface{}) (interface{}, error) {
	return unified, nil
}

func (tt *TemplateTransformer) TransformFromLegacy(legacy interface{}) (interface{}, error) {
	return legacy, nil
}

// Validator implementations

type ChunkValidator struct{}

func (cv *ChunkValidator) ValidateRequest(request interface{}) error {
	chunk, ok := request.(*models.ChunkRecord)
	if !ok {
		return fmt.Errorf("invalid chunk type")
	}

	if chunk.Content == "" {
		return fmt.Errorf("chunk content cannot be empty")
	}

	return nil
}

func (cv *ChunkValidator) SanitizeRequest(request interface{}) (interface{}, error) {
	return request, nil
}

type TextValidator struct{}

func (tv *TextValidator) ValidateRequest(request interface{}) error {
	return nil
}

func (tv *TextValidator) SanitizeRequest(request interface{}) (interface{}, error) {
	return request, nil
}

// GetCompatibilityReport generates a report on API compatibility usage
func (cl *CompatibilityLayer) GetCompatibilityReport() *CompatibilityReport {
	return &CompatibilityReport{
		GeneratedAt:        time.Now(),
		DeprecationStats:   cl.deprecationMgr.GetDeprecationStats(),
		SupportedVersions:  cl.versionManager.supportedVersions,
		ActiveWarnings:     cl.versionManager.deprecationWarnings,
		RecommendedActions: cl.generateRecommendedActions(),
	}
}

// CompatibilityReport provides information about API compatibility usage
type CompatibilityReport struct {
	GeneratedAt        time.Time                   `json:"generated_at"`
	DeprecationStats   map[string]int64           `json:"deprecation_stats"`
	SupportedVersions  map[string]bool            `json:"supported_versions"`
	ActiveWarnings     map[string]DeprecationInfo `json:"active_warnings"`
	RecommendedActions []string                   `json:"recommended_actions"`
}

func (cl *CompatibilityLayer) generateRecommendedActions() []string {
	stats := cl.deprecationMgr.GetDeprecationStats()
	actions := []string{}

	for feature, count := range stats {
		if count > 1000 {
			actions = append(actions, fmt.Sprintf("High usage of deprecated feature '%s' (%d calls) - prioritize migration", feature, count))
		} else if count > 100 {
			actions = append(actions, fmt.Sprintf("Moderate usage of deprecated feature '%s' (%d calls) - plan migration", feature, count))
		}
	}

	if len(actions) == 0 {
		actions = append(actions, "No high-usage deprecated features detected")
	}

	return actions
}