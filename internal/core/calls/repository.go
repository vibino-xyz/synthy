package calls

import "context"

type CallEdgeRepository interface {
	InsertCallEdges(ctx context.Context, edges []*CallEdge) error
	GetCallEdgesByRepositoryID(ctx context.Context, repositoryID string) ([]*CallEdge, error)
	GetCallEdgesByCallerSymbolID(ctx context.Context, callerSymbolID string) ([]*CallEdge, error)
	DeleteCallEdgesByRepositoryID(ctx context.Context, repositoryID string) error
}
