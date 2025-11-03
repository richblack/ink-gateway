package services

import (
	"context"
	"fmt"
	"semantic-text-processor/models"
)

// tagService implements TagService interface
type tagService struct {
	supabaseClient SupabaseClient
}

// NewTagService creates a new tag service instance
func NewTagService(supabaseClient SupabaseClient) TagService {
	return &tagService{
		supabaseClient: supabaseClient,
	}
}

// AddTag adds a tag to a chunk
func (s *tagService) AddTag(ctx context.Context, chunkID string, tagContent string) error {
	if chunkID == "" {
		return fmt.Errorf("chunk ID is required")
	}
	
	if tagContent == "" {
		return fmt.Errorf("tag content is required")
	}
	
	// Delegate to Supabase client
	return s.supabaseClient.AddTag(ctx, chunkID, tagContent)
}

// RemoveTag removes a tag from a chunk
func (s *tagService) RemoveTag(ctx context.Context, chunkID string, tagChunkID string) error {
	if chunkID == "" {
		return fmt.Errorf("chunk ID is required")
	}
	
	if tagChunkID == "" {
		return fmt.Errorf("tag chunk ID is required")
	}
	
	// Delegate to Supabase client
	return s.supabaseClient.RemoveTag(ctx, chunkID, tagChunkID)
}

// GetChunkTags retrieves all tags for a chunk
func (s *tagService) GetChunkTags(ctx context.Context, chunkID string) ([]models.ChunkRecord, error) {
	if chunkID == "" {
		return nil, fmt.Errorf("chunk ID is required")
	}
	
	// Delegate to Supabase client
	return s.supabaseClient.GetChunkTags(ctx, chunkID)
}

// GetChunksByTag retrieves all chunks with a specific tag
func (s *tagService) GetChunksByTag(ctx context.Context, tagContent string) ([]models.ChunkRecord, error) {
	if tagContent == "" {
		return nil, fmt.Errorf("tag content is required")
	}
	
	// Delegate to Supabase client
	return s.supabaseClient.GetChunksByTag(ctx, tagContent)
}

// AddTagWithInheritance adds a tag to a chunk and propagates it to child chunks
func (s *tagService) AddTagWithInheritance(ctx context.Context, chunkID string, tagContent string) error {
	if chunkID == "" {
		return fmt.Errorf("chunk ID is required")
	}
	
	if tagContent == "" {
		return fmt.Errorf("tag content is required")
	}
	
	// Add tag to the chunk itself
	if err := s.supabaseClient.AddTag(ctx, chunkID, tagContent); err != nil {
		return fmt.Errorf("failed to add tag to chunk: %w", err)
	}
	
	// Get all child chunks and propagate the tag
	if err := s.propagateTagToChildren(ctx, chunkID, tagContent); err != nil {
		return fmt.Errorf("failed to propagate tag to children: %w", err)
	}
	
	return nil
}

// RemoveTagWithInheritance removes a tag from a chunk and its children
func (s *tagService) RemoveTagWithInheritance(ctx context.Context, chunkID string, tagChunkID string) error {
	if chunkID == "" {
		return fmt.Errorf("chunk ID is required")
	}
	
	if tagChunkID == "" {
		return fmt.Errorf("tag chunk ID is required")
	}
	
	// Remove tag from the chunk itself
	if err := s.supabaseClient.RemoveTag(ctx, chunkID, tagChunkID); err != nil {
		return fmt.Errorf("failed to remove tag from chunk: %w", err)
	}
	
	// Remove tag from all child chunks
	if err := s.removeTagFromChildren(ctx, chunkID, tagChunkID); err != nil {
		return fmt.Errorf("failed to remove tag from children: %w", err)
	}
	
	return nil
}

// GetInheritedTags gets all tags for a chunk including inherited ones from parents
func (s *tagService) GetInheritedTags(ctx context.Context, chunkID string) ([]models.ChunkRecord, error) {
	if chunkID == "" {
		return nil, fmt.Errorf("chunk ID is required")
	}
	
	// Get direct tags for this chunk
	directTags, err := s.supabaseClient.GetChunkTags(ctx, chunkID)
	if err != nil {
		return nil, fmt.Errorf("failed to get direct tags: %w", err)
	}
	
	// Get inherited tags from parent hierarchy
	inheritedTags, err := s.getInheritedTagsFromParents(ctx, chunkID)
	if err != nil {
		return nil, fmt.Errorf("failed to get inherited tags: %w", err)
	}
	
	// Combine and deduplicate tags
	allTags := s.deduplicateTags(append(directTags, inheritedTags...))
	
	return allTags, nil
}

// propagateTagToChildren recursively adds a tag to all child chunks
func (s *tagService) propagateTagToChildren(ctx context.Context, parentChunkID string, tagContent string) error {
	// Get direct children
	children, err := s.supabaseClient.GetChildrenChunks(ctx, parentChunkID)
	if err != nil {
		return fmt.Errorf("failed to get children chunks: %w", err)
	}
	
	// Add tag to each child and recursively to their children
	for _, child := range children {
		// Add tag to this child
		if err := s.supabaseClient.AddTag(ctx, child.ID, tagContent); err != nil {
			// Log error but continue with other children
			continue
		}
		
		// Recursively propagate to grandchildren
		if err := s.propagateTagToChildren(ctx, child.ID, tagContent); err != nil {
			// Log error but continue with other children
			continue
		}
	}
	
	return nil
}

// removeTagFromChildren recursively removes a tag from all child chunks
func (s *tagService) removeTagFromChildren(ctx context.Context, parentChunkID string, tagChunkID string) error {
	// Get direct children
	children, err := s.supabaseClient.GetChildrenChunks(ctx, parentChunkID)
	if err != nil {
		return fmt.Errorf("failed to get children chunks: %w", err)
	}
	
	// Remove tag from each child and recursively from their children
	for _, child := range children {
		// Remove tag from this child
		if err := s.supabaseClient.RemoveTag(ctx, child.ID, tagChunkID); err != nil {
			// Log error but continue with other children
			continue
		}
		
		// Recursively remove from grandchildren
		if err := s.removeTagFromChildren(ctx, child.ID, tagChunkID); err != nil {
			// Log error but continue with other children
			continue
		}
	}
	
	return nil
}

// getInheritedTagsFromParents gets all tags inherited from parent chunks
func (s *tagService) getInheritedTagsFromParents(ctx context.Context, chunkID string) ([]models.ChunkRecord, error) {
	var inheritedTags []models.ChunkRecord
	
	// Get the chunk to find its parent
	chunk, err := s.supabaseClient.GetChunkByID(ctx, chunkID)
	if err != nil {
		return nil, fmt.Errorf("failed to get chunk: %w", err)
	}
	
	// If chunk is nil or has no parent, return empty list
	if chunk == nil || chunk.ParentChunkID == nil {
		return inheritedTags, nil
	}
	
	// Get parent's tags
	parentTags, err := s.supabaseClient.GetChunkTags(ctx, *chunk.ParentChunkID)
	if err != nil {
		return nil, fmt.Errorf("failed to get parent tags: %w", err)
	}
	
	inheritedTags = append(inheritedTags, parentTags...)
	
	// Recursively get tags from grandparents
	grandparentTags, err := s.getInheritedTagsFromParents(ctx, *chunk.ParentChunkID)
	if err != nil {
		return nil, fmt.Errorf("failed to get grandparent tags: %w", err)
	}
	
	inheritedTags = append(inheritedTags, grandparentTags...)
	
	return inheritedTags, nil
}

// deduplicateTags removes duplicate tags from a slice
func (s *tagService) deduplicateTags(tags []models.ChunkRecord) []models.ChunkRecord {
	seen := make(map[string]bool)
	var result []models.ChunkRecord
	
	for _, tag := range tags {
		if !seen[tag.ID] {
			seen[tag.ID] = true
			result = append(result, tag)
		}
	}
	
	return result
}