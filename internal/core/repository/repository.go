package repository

import "context"

type RepositoryRepository interface {
	InsertRepository(ctx context.Context, repository *Repository) (string, error)
	GetRepositoryById(ctx context.Context, repositoryId string) (*Repository, error)
	GetRepositoryByExternalID(ctx context.Context, externalID string) (*Repository, error)
	DeleteRepositoryByID(ctx context.Context, id string) error
}
