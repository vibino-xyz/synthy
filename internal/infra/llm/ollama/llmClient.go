package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/vibino-xyz/synthy/internal/core/llm"
)

type LLMClient struct {
	baseURL string
	model   string
	http    *http.Client
}

func NewLLMClient(httpClient *http.Client) llm.LLMClient {
	return &LLMClient{
		baseURL: "http://localhost:11434",
		model:   "qwen3-coder:30b",
		http:    httpClient,
	}
}

type generateRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type generateResponse struct {
	Response string `json:"response"`
}

func (c *LLMClient) GenerateSummary(ctx context.Context, input string) (string, error) {
	prompt := "Summarize the following code concisely, focusing on its purpose and key behaviour:\n\n" + input

	body, err := json.Marshal(generateRequest{Model: c.model, Prompt: prompt, Stream: false})
	if err != nil {
		return "", fmt.Errorf("marshal generate request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/generate", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create generate request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("generate request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("generate request failed with status %d", resp.StatusCode)
	}

	var result generateResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode generate response: %w", err)
	}

	return result.Response, nil
}
