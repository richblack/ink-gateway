package models

import (
	"errors"
	"time"
)

// TextRecord represents a text document in the system
type TextRecord struct {
	ID        string    `json:"id" db:"id"`
	Content   string    `json:"content" db:"content"`
	Title     string    `json:"title" db:"title"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
	Status    string    `json:"status" db:"status"`
}

// ChunkRecord represents a text chunk with hierarchical structure
type ChunkRecord struct {
	ID              string                 `json:"id" db:"id"`
	TextID          string                 `json:"text_id" db:"text_id"`
	Content         string                 `json:"content" db:"content"`
	IsTemplate      bool                   `json:"is_template" db:"is_template"`
	IsSlot          bool                   `json:"is_slot" db:"is_slot"`
	ParentChunkID   *string                `json:"parent_chunk_id" db:"parent_chunk_id"`
	TemplateChunkID *string                `json:"template_chunk_id" db:"template_chunk_id"`
	SlotValue       *string                `json:"slot_value" db:"slot_value"`
	IndentLevel     int                    `json:"indent_level" db:"indent_level"`
	SequenceNumber  *int                   `json:"sequence_number" db:"sequence_number"`
	Metadata        map[string]interface{} `json:"metadata" db:"metadata"`
	CreatedAt       time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at" db:"updated_at"`
}

// ChunkTag represents the relationship between chunks and tags
type ChunkTag struct {
	ID         string    `json:"id" db:"id"`
	ChunkID    string    `json:"chunk_id" db:"chunk_id"`
	TagChunkID string    `json:"tag_chunk_id" db:"tag_chunk_id"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

// TemplateSlot represents slot definitions in templates
type TemplateSlot struct {
	ID              string    `json:"id" db:"id"`
	TemplateChunkID string    `json:"template_chunk_id" db:"template_chunk_id"`
	SlotChunkID     string    `json:"slot_chunk_id" db:"slot_chunk_id"`
	SlotOrder       int       `json:"slot_order" db:"slot_order"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
}

// EmbeddingRecord represents vector embeddings for chunks
type EmbeddingRecord struct {
	ID        string    `json:"id" db:"id"`
	ChunkID   string    `json:"chunk_id" db:"chunk_id"`
	Vector    []float64 `json:"vector" db:"vector"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// GraphNode represents knowledge graph nodes
type GraphNode struct {
	ID         string                 `json:"id" db:"id"`
	ChunkID    string                 `json:"chunk_id" db:"chunk_id"`
	EntityName string                 `json:"entity_name" db:"entity_name"`
	EntityType string                 `json:"entity_type" db:"entity_type"`
	Properties map[string]interface{} `json:"properties" db:"properties"`
	CreatedAt  time.Time              `json:"created_at" db:"created_at"`
}

// GraphEdge represents relationships in knowledge graph
type GraphEdge struct {
	ID               string                 `json:"id" db:"id"`
	SourceNodeID     string                 `json:"source_node_id" db:"source_node_id"`
	TargetNodeID     string                 `json:"target_node_id" db:"target_node_id"`
	RelationshipType string                 `json:"relationship_type" db:"relationship_type"`
	Properties       map[string]interface{} `json:"properties" db:"properties"`
	CreatedAt        time.Time              `json:"created_at" db:"created_at"`
}

// Common errors
var (
	ErrNotFound         = errors.New("resource not found")
	ErrInvalidOperation = errors.New("invalid operation")
	ErrDuplicateEntry   = errors.New("duplicate entry")
	ErrInvalidInput     = errors.New("invalid input")
)