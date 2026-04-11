package psql

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vibino-xyz/synthy/internal/core/calls"
)

type callEdgeRepository struct {
	db *pgxpool.Pool
}

func NewCallEdgeRepository(db *pgxpool.Pool) calls.CallEdgeRepository {
	return &callEdgeRepository{db: db}
}

func (r *callEdgeRepository) InsertCallEdges(ctx context.Context, edges []*calls.CallEdge) error {
	if len(edges) == 0 {
		return nil
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	const query = `
		INSERT INTO call_edge (id, repository_id, caller_symbol_id, callee_symbol_id, file_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`

	for _, edge := range edges {
		if _, err := tx.Exec(ctx, query,
			edge.ID, edge.RepositoryID, edge.CallerSymbolID, edge.CalleeSymbolID,
			edge.FileID, edge.CreatedAt, edge.UpdatedAt,
		); err != nil {
			return fmt.Errorf("insert call edge: %w", err)
		}
	}

	return tx.Commit(ctx)
}

func (r *callEdgeRepository) GetCallEdgesByRepositoryID(ctx context.Context, repositoryID string) ([]*calls.CallEdge, error) {
	const query = `
		SELECT id, repository_id, caller_symbol_id, callee_symbol_id, file_id, created_at, updated_at
		FROM call_edge
		WHERE repository_id = $1`

	return r.queryEdges(ctx, query, repositoryID)
}

func (r *callEdgeRepository) GetCallEdgesByCallerSymbolID(ctx context.Context, callerSymbolID string) ([]*calls.CallEdge, error) {
	const query = `
		SELECT id, repository_id, caller_symbol_id, callee_symbol_id, file_id, created_at, updated_at
		FROM call_edge
		WHERE caller_symbol_id = $1`

	return r.queryEdges(ctx, query, callerSymbolID)
}

func (r *callEdgeRepository) DeleteCallEdgesByRepositoryID(ctx context.Context, repositoryID string) error {
	const query = `DELETE FROM call_edge WHERE repository_id = $1`
	_, err := r.db.Exec(ctx, query, repositoryID)
	return err
}

func (r *callEdgeRepository) queryEdges(ctx context.Context, query string, arg interface{}) ([]*calls.CallEdge, error) {
	rows, err := r.db.Query(ctx, query, arg)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	defer rows.Close()

	var edges []*calls.CallEdge
	for rows.Next() {
		var edge calls.CallEdge
		if err := rows.Scan(
			&edge.ID, &edge.RepositoryID, &edge.CallerSymbolID, &edge.CalleeSymbolID,
			&edge.FileID, &edge.CreatedAt, &edge.UpdatedAt,
		); err != nil {
			return nil, err
		}
		edges = append(edges, &edge)
	}

	return edges, rows.Err()
}
