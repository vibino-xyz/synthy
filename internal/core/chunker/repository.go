package chunker

import "context"

type UpdateChunkRequest struct {
	EmbeddingID *string
	Summary     *string
}

type ChunkRepository interface {
	InsertChunks(ctx context.Context, chunks []*CodeChunk) error
	GetChunksByFileID(ctx context.Context, fileID string) ([]*CodeChunk, error)
	GetChunkByID(ctx context.Context, chunkID string) (*CodeChunk, error)
	GetChunksByRepositoryID(ctx context.Context, repositoryID string) ([]*CodeChunk, error)
	GetChunksWithoutEmbedding(ctx context.Context, limit int) ([]*CodeChunk, error)
	UpdateChunk(ctx context.Context, chunkID string, req UpdateChunkRequest) error
	DeleteChunksByRepositoryID(ctx context.Context, repositoryID string) error
	GetChunksByEmbeddingIDs(ctx context.Context, embeddingIDs []string) ([]*CodeChunk, error)
}
