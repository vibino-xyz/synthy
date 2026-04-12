package embedding

import "context"

type PineconeRepository interface {
	UpsertEmbeddings(ctx context.Context, embeddings []float32, metadata map[string]any, namespace string) (string, error)
}
