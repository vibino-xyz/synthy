package repository

import "context"

type RepositoryRepository interface {
	InsertRepository(ctx context.Context, repository *Repository) (string, error)
	GetRepositoryById(ctx context.Context, repositoryId string) (*Repository, error)
}
