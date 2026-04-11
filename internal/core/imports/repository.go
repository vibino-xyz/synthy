package imports

import "context"

type ImportEdgeRepository interface {
	InsertImportEdges(ctx context.Context, edges []*ImportEdge) error
	GetImportEdgesByRepositoryID(ctx context.Context, repositoryID string) ([]*ImportEdge, error)
	GetImportEdgesByFromFileID(ctx context.Context, fromFileID string) ([]*ImportEdge, error)
	DeleteImportEdgesByRepositoryID(ctx context.Context, repositoryID string) error
}
