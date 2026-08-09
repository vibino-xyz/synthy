package repository

import "context"

type RepositoryRepository interface {
	InsertRepository(ctx context.Context, repository *Repository) (string, error)
	ListRepositoriesByOrganizationID(ctx context.Context, organizationID string) ([]*Repository, error)
	GetRepositoryById(ctx context.Context, repositoryId string) (*Repository, error)
	GetRepositoryByExternalID(ctx context.Context, externalID string) (*Repository, error)
	// UpdateRepository refreshes the mutable fields of an existing row. An
	// incremental index must not delete and reinsert the repository: that would
	// cascade away every file, chunk and embedding it is trying to preserve.
	UpdateRepository(ctx context.Context, repo *Repository) error
	DeleteRepositoryByID(ctx context.Context, id string) error
}
