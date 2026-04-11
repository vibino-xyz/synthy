package psql

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vibino-xyz/synthy/internal/core/symbol"
)

type symbolRepository struct {
	db *pgxpool.Pool
}

func NewSymbolRepository(db *pgxpool.Pool) symbol.SymbolRepository {
	return &symbolRepository{db: db}
}

func (r *symbolRepository) InsertSymbols(ctx context.Context, symbols []*symbol.Symbol) error {
	if len(symbols) == 0 {
		return nil
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	const query = `
		INSERT INTO symbol (
			id, file_id, repository_id, fq_name, type, language, signature,
			receiver_type, visibility, parent_symbol_id,
			start_line, end_line, start_column, end_column,
			created_at, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)`

	for _, sym := range symbols {
		if _, err := tx.Exec(ctx, query,
			sym.ID, sym.FileID, sym.RepositoryID, sym.FqName, sym.Type, sym.Language, sym.Signature,
			sym.ReceiverType, sym.Visibility, sym.ParentSymbolId,
			sym.StartLine, sym.EndLine, sym.StartColumn, sym.EndColumn,
			sym.CreatedAt, sym.UpdatedAt,
		); err != nil {
			return fmt.Errorf("insert symbol %s: %w", sym.FqName, err)
		}
	}

	return tx.Commit(ctx)
}

func (r *symbolRepository) GetSymbolsByFileID(ctx context.Context, fileID string) ([]*symbol.Symbol, error) {
	const query = `
		SELECT id, file_id, repository_id, fq_name, type, language, signature,
		       receiver_type, visibility, parent_symbol_id,
		       start_line, end_line, start_column, end_column,
		       created_at, updated_at
		FROM symbol
		WHERE file_id = $1`

	return r.querySymbols(ctx, query, fileID)
}

func (r *symbolRepository) GetSymbolsByRepositoryID(ctx context.Context, repositoryID string) ([]*symbol.Symbol, error) {
	const query = `
		SELECT id, file_id, repository_id, fq_name, type, language, signature,
		       receiver_type, visibility, parent_symbol_id,
		       start_line, end_line, start_column, end_column,
		       created_at, updated_at
		FROM symbol
		WHERE repository_id = $1`

	return r.querySymbols(ctx, query, repositoryID)
}

func (r *symbolRepository) DeleteSymbolsByRepositoryID(ctx context.Context, repositoryID string) error {
	const query = `DELETE FROM symbol WHERE repository_id = $1`
	_, err := r.db.Exec(ctx, query, repositoryID)
	return err
}

func (r *symbolRepository) querySymbols(ctx context.Context, query string, arg interface{}) ([]*symbol.Symbol, error) {
	rows, err := r.db.Query(ctx, query, arg)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	defer rows.Close()

	var symbols []*symbol.Symbol
	for rows.Next() {
		var sym symbol.Symbol
		if err := rows.Scan(
			&sym.ID, &sym.FileID, &sym.RepositoryID, &sym.FqName, &sym.Type, &sym.Language, &sym.Signature,
			&sym.ReceiverType, &sym.Visibility, &sym.ParentSymbolId,
			&sym.StartLine, &sym.EndLine, &sym.StartColumn, &sym.EndColumn,
			&sym.CreatedAt, &sym.UpdatedAt,
		); err != nil {
			return nil, err
		}
		symbols = append(symbols, &sym)
	}

	return symbols, rows.Err()
}
