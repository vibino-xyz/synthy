package embedding

import "context"

type PineconeRepository interface {
	UpsertEmbeddings(ctx context.Context, embeddings []float32, metadata map[string]any, namespace string) (string, error)
	QuerySimilarVectors(ctx context.Context, queryEmbedding []float32, topK int, namespace string) ([]string, error)
}
