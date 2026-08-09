package file

import "context"

type FileRepository interface {
	InsertFile(ctx context.Context, file *File) error
	GetFileByRepositoryIDAndPath(ctx context.Context, repositoryID, path string) (*File, error)
	GetFilesByRepositoryID(ctx context.Context, repositoryID string) ([]*File, error)
	DeleteFilesByRepositoryID(ctx context.Context, repositoryID string) error
	DeleteFileByID(ctx context.Context, id string) error
}
