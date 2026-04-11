package symbol

import "context"

type SymbolRepository interface {
	InsertSymbols(ctx context.Context, symbols []*Symbol) error
	GetSymbolsByFileID(ctx context.Context, fileID string) ([]*Symbol, error)
	GetSymbolsByRepositoryID(ctx context.Context, repositoryID string) ([]*Symbol, error)
	DeleteSymbolsByRepositoryID(ctx context.Context, repositoryID string) error
}
