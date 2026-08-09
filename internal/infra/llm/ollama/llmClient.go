package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/vibino-xyz/synthy/internal/core/llm"
)

type LLMClient struct {
	baseURL string
	model   string
	http    *http.Client
}

func NewLLMClient(httpClient *http.Client) llm.LLMClient {
	return &LLMClient{
		baseURL: os.Getenv("LLM_CLIENT_BASE_URL"),
		model:   os.Getenv("LLM_CLIENT_MODEL"),
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
	if strings.TrimSpace(input) == "" {
		return "", llm.ErrEmptyInput
	}

	prompt := fmt.Sprintf(`You are analyzing a code chunk.

Goal:
Generate a concise, retrieval-optimized summary.

Instructions:
- Describe what the code does in terms of behavior and purpose
- Include specific identifiers when relevant:
  - table names, struct/class names, key functions, external APIs
- Mention key operations (e.g., SELECT, INSERT, HTTP call, validation, transformation)
- Focus on what makes this code unique

Strict rules:
- Max 60 words
- Single paragraph
- NO generic phrases like:
  - "this function"
  - "this code"
  - "key concepts include"
- NO fluff, explanations, or repetition
- Do NOT restate obvious things like "handles database operations"

Output:
Return only the summary text.

Code:
%s`, input)

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

func (c *LLMClient) GenerateResponse(ctx context.Context, query string, context string) (string, error) {
	prompt := fmt.Sprintf(`You are a coding assistant answering questions about a codebase.

Context:
%s

Question:
%s

Instructions:
- Answer based strictly on the provided context
- Reference specific functions, types, or files when relevant
- Be concise and precise`, context, query)

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
