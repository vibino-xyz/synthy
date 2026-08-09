package llm

import (
	"context"
	"errors"
)

// ErrEmptyInput is returned instead of calling out to a provider with input
// that has no content. Providers reject it anyway (voyage answers 400), and a
// failed call on a queue message that will never succeed is a poison message.
var ErrEmptyInput = errors.New("llm: input is empty")

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
