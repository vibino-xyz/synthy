package psql

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vibino-xyz/synthy/internal/core/chunker"
)

type chunkRepository struct {
	db *pgxpool.Pool
}

func NewChunkRepository(db *pgxpool.Pool) chunker.ChunkRepository {
	return &chunkRepository{db: db}
}

func (r *chunkRepository) InsertChunks(ctx context.Context, chunks []*chunker.CodeChunk) error {
	if len(chunks) == 0 {
		return nil
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	const query = `
		INSERT INTO code_chunk (
			id, repository_id, file_id, symbol_id,
			content, content_hash, type, language,
			start_line, end_line, embedding_id, summary,
			created_at, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`

	for _, c := range chunks {
		if _, err := tx.Exec(ctx, query,
			c.ID, c.RepositoryID, c.FileID, c.SymbolID,
			c.Content, c.ContentHash, c.Type, c.Language,
			c.StartLine, c.EndLine, c.EmbeddingID, c.Summary,
			c.CreatedAt, c.UpdatedAt,
		); err != nil {
			return fmt.Errorf("insert chunk: %w", err)
		}
	}

	return tx.Commit(ctx)
}

func (r *chunkRepository) GetChunksByFileID(ctx context.Context, fileID string) ([]*chunker.CodeChunk, error) {
	const query = `
		SELECT id, repository_id, file_id, symbol_id,
		       content, content_hash, type, language,
		       start_line, end_line, embedding_id, summary,
		       created_at, updated_at
		FROM code_chunk
		WHERE file_id = $1`

	return r.queryChunks(ctx, query, fileID)
}

func (r *chunkRepository) GetChunksByRepositoryID(ctx context.Context, repositoryID string) ([]*chunker.CodeChunk, error) {
	const query = `
		SELECT id, repository_id, file_id, symbol_id,
		       content, content_hash, type, language,
		       start_line, end_line, embedding_id, summary,
		       created_at, updated_at
		FROM code_chunk
		WHERE repository_id = $1`

	return r.queryChunks(ctx, query, repositoryID)
}

func (r *chunkRepository) UpdateChunkEmbeddingID(ctx context.Context, chunkID, embeddingID string) error {
	const query = `UPDATE code_chunk SET embedding_id = $1 WHERE id = $2`
	_, err := r.db.Exec(ctx, query, embeddingID, chunkID)
	return err
}

func (r *chunkRepository) DeleteChunksByRepositoryID(ctx context.Context, repositoryID string) error {
	const query = `DELETE FROM code_chunk WHERE repository_id = $1`
	_, err := r.db.Exec(ctx, query, repositoryID)
	return err
}

func (r *chunkRepository) queryChunks(ctx context.Context, query string, arg interface{}) ([]*chunker.CodeChunk, error) {
	rows, err := r.db.Query(ctx, query, arg)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	defer rows.Close()

	var chunks []*chunker.CodeChunk
	for rows.Next() {
		var c chunker.CodeChunk
		if err := rows.Scan(
			&c.ID, &c.RepositoryID, &c.FileID, &c.SymbolID,
			&c.Content, &c.ContentHash, &c.Type, &c.Language,
			&c.StartLine, &c.EndLine, &c.EmbeddingID, &c.Summary,
			&c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, err
		}
		chunks = append(chunks, &c)
	}

	return chunks, rows.Err()
}
