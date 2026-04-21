package llm

import "context"

type EmbeddingClient interface {
	GenerateEmbeddings(ctx context.Context, input string) ([]float32, error)
}

type LLMClient interface {
	GenerateSummary(ctx context.Context, input string) (string, error)
	GenerateResponse(ctx context.Context, query string, context string) (string, error)
}
