package psql

import (
	"context"

	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vibino-xyz/synthy/internal/core/repository"
)

type repositoryRepository struct {
	db *pgxpool.Pool
}

func NewRepositoryRepository(db *pgxpool.Pool) repository.RepositoryRepository {
	return &repositoryRepository{
		db: db,
	}
}

func (r *repositoryRepository) GetRepositoryById(ctx context.Context, repositoryId string) (*repository.Repository, error) {
	const query = `SELECT id, organization_id, name, description, default_branch, repository_url, storage_url, provider, external_id, created_at, updated_at from repository WHERE id = $1`

	var result repository.Repository

	err := r.db.QueryRow(ctx, query, repositoryId).Scan(
		&result.ID,
		&result.OrganizationID,
		&result.Name,
		&result.Description,
		&result.DefaultBranch,
		&result.RepositoryUrl,
		&result.StorageUrl,
		&result.Provider,
		&result.ExternalId,
		&result.CreatedAt,
		&result.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &result, nil
}

func (r *repositoryRepository) InsertRepository(ctx context.Context, repo *repository.Repository) (string, error) {
	const query = `
		INSERT INTO repository (id, organization_id, name, description, default_branch, repository_url, storage_url, provider, external_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id`

	var repositoryId string
	err := r.db.QueryRow(ctx, query,
		repo.ID,
		repo.OrganizationID,
		repo.Name,
		repo.Description,
		repo.DefaultBranch,
		repo.RepositoryUrl,
		repo.StorageUrl,
		repo.Provider,
		repo.ExternalId,
		repo.CreatedAt,
		repo.UpdatedAt,
	).Scan(&repositoryId)

	return repositoryId, err
}
