package chunker

import "context"

type ChunkRepository interface {
	InsertChunks(ctx context.Context, chunks []*CodeChunk) error
	GetChunksByFileID(ctx context.Context, fileID string) ([]*CodeChunk, error)
	GetChunksByRepositoryID(ctx context.Context, repositoryID string) ([]*CodeChunk, error)
	UpdateChunkEmbeddingID(ctx context.Context, chunkID, embeddingID string) error
	DeleteChunksByRepositoryID(ctx context.Context, repositoryID string) error
}
