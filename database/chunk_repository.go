package database

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"semantic-text-processor/models"
)

// ChunkRepository provides database operations for chunks
type ChunkRepository struct {
	db *PostgresService
}

// NewChunkRepository creates a new chunk repository
func NewChunkRepository(db *PostgresService) *ChunkRepository {
	return &ChunkRepository{db: db}
}

// Create inserts a new chunk into the database
func (r *ChunkRepository) Create(ctx context.Context, chunk *models.UnifiedChunkRecord) error {
	// Set timestamps
	now := time.Now()
	if chunk.CreatedTime.IsZero() {
		chunk.CreatedTime = now
	}

	// Convert metadata to JSON
	metadataJSON, err := json.Marshal(chunk.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// SQL query - let database generate UUID with RETURNING clause
	query := `
		INSERT INTO public.chunks (
			contents, is_page, parent, metadata, created_time
		) VALUES (
			$1, $2, $3, $4, $5
		)
		RETURNING chunk_id
	`

	// DEBUG: Print query and parameters
	fmt.Printf("DEBUG - Query: %s\n", query)
	fmt.Printf("DEBUG - Contents: %v\n", chunk.Contents)
	fmt.Printf("DEBUG - IsPage: %v\n", chunk.IsPage)
	fmt.Printf("DEBUG - Parent: %v\n", chunk.Parent)
	fmt.Printf("DEBUG - Metadata: %s\n", string(metadataJSON))
	fmt.Printf("DEBUG - CreatedTime: %v\n", chunk.CreatedTime)

	// Explicitly set search_path before query
	_, err = r.db.Exec(ctx, "SET search_path TO public")
	if err != nil {
		return fmt.Errorf("failed to set search_path: %w", err)
	}

	// Use QueryRow to get the generated UUID
	err = r.db.QueryRow(ctx, query,
		chunk.Contents,
		chunk.IsPage,
		chunk.Parent,
		metadataJSON,
		chunk.CreatedTime,
	).Scan(&chunk.ChunkID)

	if err != nil {
		return fmt.Errorf("failed to insert chunk: %w", err)
	}

	return nil
}

// GetByID retrieves a chunk by its ID
func (r *ChunkRepository) GetByID(ctx context.Context, chunkID string) (*models.UnifiedChunkRecord, error) {
	query := `
		SELECT
			chunk_id, contents, is_page, parent, metadata, created_time
		FROM public.chunks
		WHERE chunk_id = $1
	`

	var chunk models.UnifiedChunkRecord
	var metadataJSON []byte

	err := r.db.QueryRow(ctx, query, chunkID).Scan(
		&chunk.ChunkID,
		&chunk.Contents,
		&chunk.IsPage,
		&chunk.Parent,
		&metadataJSON,
		&chunk.CreatedTime,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("chunk not found: %s", chunkID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query chunk: %w", err)
	}

	// Parse metadata
	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &chunk.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return &chunk, nil
}

// List retrieves all chunks with optional pagination
func (r *ChunkRepository) List(ctx context.Context, limit, offset int) ([]*models.UnifiedChunkRecord, error) {
	query := `
		SELECT
			chunk_id, contents, is_page, parent, metadata, created_time
		FROM public.chunks
		ORDER BY created_time DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query chunks: %w", err)
	}
	defer rows.Close()

	var chunks []*models.UnifiedChunkRecord

	for rows.Next() {
		var chunk models.UnifiedChunkRecord
		var metadataJSON []byte

		err := rows.Scan(
			&chunk.ChunkID,
			&chunk.Contents,
			&chunk.IsPage,
			&chunk.Parent,
			&metadataJSON,
			&chunk.CreatedTime,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan chunk: %w", err)
		}

		// Parse metadata
		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &chunk.Metadata); err != nil {
				return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
			}
		}

		chunks = append(chunks, &chunk)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return chunks, nil
}

// Update updates an existing chunk
func (r *ChunkRepository) Update(ctx context.Context, chunk *models.UnifiedChunkRecord) error {
	// Convert metadata to JSON
	metadataJSON, err := json.Marshal(chunk.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		UPDATE public.chunks SET
			contents = $2,
			is_page = $3,
			parent = $4,
			metadata = $5
		WHERE chunk_id = $1
	`

	cmdTag, err := r.db.Exec(ctx, query,
		chunk.ChunkID,
		chunk.Contents,
		chunk.IsPage,
		chunk.Parent,
		metadataJSON,
	)

	if err != nil {
		return fmt.Errorf("failed to update chunk: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("chunk not found: %s", chunk.ChunkID)
	}

	return nil
}

// Delete removes a chunk from the database
func (r *ChunkRepository) Delete(ctx context.Context, chunkID string) error {
	query := `DELETE FROM public.chunks WHERE chunk_id = $1`

	cmdTag, err := r.db.Exec(ctx, query, chunkID)
	if err != nil {
		return fmt.Errorf("failed to delete chunk: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("chunk not found: %s", chunkID)
	}

	return nil
}

// SearchByContent searches chunks by content using full-text search
func (r *ChunkRepository) SearchByContent(ctx context.Context, searchText string, limit int) ([]*models.UnifiedChunkRecord, error) {
	query := `
		SELECT
			chunk_id, contents, is_page, parent, metadata, created_time
		FROM public.chunks
		WHERE contents ILIKE $1
		ORDER BY created_time DESC
		LIMIT $2
	`

	rows, err := r.db.Query(ctx, query, "%"+searchText+"%", limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search chunks: %w", err)
	}
	defer rows.Close()

	var chunks []*models.UnifiedChunkRecord

	for rows.Next() {
		var chunk models.UnifiedChunkRecord
		var metadataJSON []byte

		err := rows.Scan(
			&chunk.ChunkID,
			&chunk.Contents,
			&chunk.IsPage,
			&chunk.Parent,
			&metadataJSON,
			&chunk.CreatedTime,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan chunk: %w", err)
		}

		// Parse metadata
		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &chunk.Metadata); err != nil {
				return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
			}
		}

		chunks = append(chunks, &chunk)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return chunks, nil
}

// BatchCreate inserts multiple chunks in a transaction
func (r *ChunkRepository) BatchCreate(ctx context.Context, chunks []models.UnifiedChunkRecord) error {
	// Start transaction
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO public.chunks (
			contents, is_page, parent, metadata, created_time
		) VALUES (
			$1, $2, $3, $4, $5
		)
		RETURNING chunk_id
	`

	now := time.Now()

	for i := range chunks {
		chunk := &chunks[i]

		// Set timestamps
		if chunk.CreatedTime.IsZero() {
			chunk.CreatedTime = now
		}

		// Convert metadata to JSON
		metadataJSON, err := json.Marshal(chunk.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata for chunk %d: %w", i, err)
		}

		// Use QueryRow to get the generated UUID
		err = tx.QueryRow(ctx, query,
			chunk.Contents,
			chunk.IsPage,
			chunk.Parent,
			metadataJSON,
			chunk.CreatedTime,
		).Scan(&chunk.ChunkID)

		if err != nil {
			return fmt.Errorf("failed to insert chunk %d: %w", i, err)
		}
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
