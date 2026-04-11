package psql

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vibino-xyz/synthy/internal/core/file"
)

type fileRepository struct {
	db *pgxpool.Pool
}

func NewFileRepository(db *pgxpool.Pool) file.FileRepository {
	return &fileRepository{db: db}
}

func (r *fileRepository) InsertFile(ctx context.Context, f *file.File) error {
	const query = `
		INSERT INTO file (id, repository_id, path, name, extension, checksum, language, is_binary, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

	_, err := r.db.Exec(ctx, query,
		f.ID, f.RepositoryID, f.Path, f.Name, f.Extension,
		f.Checksum, f.Language, f.IsBinary, f.CreatedAt, f.UpdatedAt,
	)
	return err
}

func (r *fileRepository) GetFileByRepositoryIDAndPath(ctx context.Context, repositoryID, path string) (*file.File, error) {
	const query = `
		SELECT id, repository_id, path, name, extension, checksum, language, is_binary, created_at, updated_at
		FROM file
		WHERE repository_id = $1 AND path = $2`

	var f file.File
	err := r.db.QueryRow(ctx, query, repositoryID, path).Scan(
		&f.ID, &f.RepositoryID, &f.Path, &f.Name, &f.Extension,
		&f.Checksum, &f.Language, &f.IsBinary, &f.CreatedAt, &f.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &f, nil
}

func (r *fileRepository) GetFilesByRepositoryID(ctx context.Context, repositoryID string) ([]*file.File, error) {
	const query = `
		SELECT id, repository_id, path, name, extension, checksum, language, is_binary, created_at, updated_at
		FROM file
		WHERE repository_id = $1`

	rows, err := r.db.Query(ctx, query, repositoryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []*file.File
	for rows.Next() {
		var f file.File
		if err := rows.Scan(
			&f.ID, &f.RepositoryID, &f.Path, &f.Name, &f.Extension,
			&f.Checksum, &f.Language, &f.IsBinary, &f.CreatedAt, &f.UpdatedAt,
		); err != nil {
			return nil, err
		}
		files = append(files, &f)
	}

	return files, rows.Err()
}

func (r *fileRepository) DeleteFilesByRepositoryID(ctx context.Context, repositoryID string) error {
	const query = `DELETE FROM file WHERE repository_id = $1`
	_, err := r.db.Exec(ctx, query, repositoryID)
	return err
}
