package psql

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vibino-xyz/synthy/internal/core/imports"
)

type importEdgeRepository struct {
	db *pgxpool.Pool
}

func NewImportEdgeRepository(db *pgxpool.Pool) imports.ImportEdgeRepository {
	return &importEdgeRepository{db: db}
}

func (r *importEdgeRepository) InsertImportEdges(ctx context.Context, edges []*imports.ImportEdge) error {
	if len(edges) == 0 {
		return nil
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	const query = `
		INSERT INTO import_edge (id, repository_id, from_file_id, to_file_id, import_path, is_external, alias, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	for _, edge := range edges {
		if _, err := tx.Exec(ctx, query,
			edge.ID, edge.RepositoryID, edge.FromFileID, edge.ToFileID,
			edge.ImportPath, edge.IsExternal, edge.Alias,
			edge.CreatedAt, edge.UpdatedAt,
		); err != nil {
			return fmt.Errorf("insert import edge: %w", err)
		}
	}

	return tx.Commit(ctx)
}

func (r *importEdgeRepository) GetImportEdgesByRepositoryID(ctx context.Context, repositoryID string) ([]*imports.ImportEdge, error) {
	const query = `
		SELECT id, repository_id, from_file_id, to_file_id, import_path, is_external, alias, created_at, updated_at
		FROM import_edge
		WHERE repository_id = $1`

	return r.queryEdges(ctx, query, repositoryID)
}

func (r *importEdgeRepository) GetImportEdgesByFromFileID(ctx context.Context, fromFileID string) ([]*imports.ImportEdge, error) {
	const query = `
		SELECT id, repository_id, from_file_id, to_file_id, import_path, is_external, alias, created_at, updated_at
		FROM import_edge
		WHERE from_file_id = $1`

	return r.queryEdges(ctx, query, fromFileID)
}

func (r *importEdgeRepository) DeleteImportEdgesByRepositoryID(ctx context.Context, repositoryID string) error {
	const query = `DELETE FROM import_edge WHERE repository_id = $1`
	_, err := r.db.Exec(ctx, query, repositoryID)
	return err
}

func (r *importEdgeRepository) queryEdges(ctx context.Context, query string, arg interface{}) ([]*imports.ImportEdge, error) {
	rows, err := r.db.Query(ctx, query, arg)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	defer rows.Close()

	var edges []*imports.ImportEdge
	for rows.Next() {
		var edge imports.ImportEdge
		if err := rows.Scan(
			&edge.ID, &edge.RepositoryID, &edge.FromFileID, &edge.ToFileID,
			&edge.ImportPath, &edge.IsExternal, &edge.Alias,
			&edge.CreatedAt, &edge.UpdatedAt,
		); err != nil {
			return nil, err
		}
		edges = append(edges, &edge)
	}

	return edges, rows.Err()
}
