package voyage

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/vibino-xyz/commons/ratelimit"
	"github.com/vibino-xyz/synthy/internal/core/llm"
)

type EmbeddingClient struct {
	baseURL   string
	apiKey    string
	model     string
	outputDim int // 0 => model default (1024 for voyage-code-3)
	http      *http.Client
	limiter   *ratelimit.Limiter
}

func NewRateLimitConfig() ratelimit.Config {
	return ratelimit.Config{
		RequestsPerMinute: envInt("VOYAGE_RPM", 2_000),
		TokensPerMinute:   envInt("VOYAGE_TPM", 3_000_000),
	}
}

func NewEmbeddingClient(httpClient *http.Client, limiter *ratelimit.Limiter) llm.EmbeddingClient {
	baseURL := os.Getenv("VOYAGE_BASE_URL")
	if baseURL == "" {
		baseURL = "https://api.voyageai.com/v1"
	}
	model := os.Getenv("VOYAGE_MODEL")
	if model == "" {
		model = "voyage-code-3"
	}
	dim, _ := strconv.Atoi(os.Getenv("VOYAGE_OUTPUT_DIMENSION")) // 0 if unset

	return &EmbeddingClient{
		baseURL:   baseURL,
		apiKey:    os.Getenv("VOYAGE_API_KEY"),
		model:     model,
		outputDim: dim,
		http:      httpClient,
		limiter:   limiter,
	}
}

type embedRequest struct {
	Model           string   `json:"model"`
	Input           []string `json:"input"`
	InputType       string   `json:"input_type,omitempty"`
	OutputDimension int      `json:"output_dimension,omitempty"`
}

type embedResponse struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
	Usage struct {
		TotalTokens int `json:"total_tokens"`
	} `json:"usage"`
}

func (e EmbeddingClient) GenerateEmbeddings(ctx context.Context, input string, inputType llm.InputType) ([]float32, error) {
	if strings.TrimSpace(input) == "" {
		return nil, llm.ErrEmptyInput
	}

	if err := e.limiter.Wait(ctx, ratelimit.EstimateTokens(input)); err != nil {
		return nil, fmt.Errorf("voyage rate limit: %w", err)
	}

	body, err := json.Marshal(embedRequest{
		Model:           e.model,
		Input:           []string{input},
		InputType:       string(inputType),
		OutputDimension: e.outputDim,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.baseURL+"/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e.apiKey)

	resp, err := e.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("voyage embed request failed with status %d: %s", resp.StatusCode, string(b))
	}

	var result embedResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	if len(result.Data) == 0 {
		return nil, fmt.Errorf("no embeddings returned")
	}

	return result.Data[0].Embedding, nil
}

func envInt(key string, def int) int {
	if v, err := strconv.Atoi(os.Getenv(key)); err == nil && v > 0 {
		return v
	}
	return def
}
