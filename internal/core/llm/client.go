package llm

import "context"

type InputType string

const (
	InputTypeDocument InputType = "document"
	InputTypeQuery    InputType = "query"
)

type EmbeddingClient interface {
	GenerateEmbeddings(ctx context.Context, input string, inputType InputType) ([]float32, error)
}

type LLMClient interface {
	GenerateSummary(ctx context.Context, input string) (string, error)
	GenerateResponse(ctx context.Context, query string, context string) (string, error)
}
