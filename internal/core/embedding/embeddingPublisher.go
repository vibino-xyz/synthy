package embedding

import (
	"context"
)

type EmbeddingPublisher interface {
	PublishEmbedding(ctx context.Context, req EmbeddingPublisherRequest) error
}

type EmbeddingPublisherRequest struct {
	ChunkId      string `json:"chunk_id"`
	RepositoryId string `json:"repository_id"`
}
