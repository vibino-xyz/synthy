package claude

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/vibino-xyz/synthy/internal/core/llm"
)

type ClaudeCodeClient struct {
	BinaryPath string
	Model      string
}

func NewClaudeCodeClient(binaryPath, model string) llm.LLMClient {
	return &ClaudeCodeClient{
		BinaryPath: binaryPath,
		Model:      model,
	}
}

type cliResult struct {
	Result       string  `json:"result"`
	IsError      bool    `json:"is_error"`
	SessionID    string  `json:"session_id"`
	TotalCostUSD float64 `json:"total_cost_usd"`
}

func (c *ClaudeCodeClient) invoke(ctx context.Context, prompt string) (string, error) {
	bin := c.BinaryPath
	if bin == "" {
		bin = "claude"
	}

	args := []string{
		"-p", prompt,
		"--output-format", "json",
		"--max-turns", "1",
	}

	if c.Model != "" {
		args = append(args, "--model", c.Model)
	}

	cmd := exec.CommandContext(ctx, bin, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("claude CLI invocation failed: %w (stderr: %s)", err, stderr.String())
	}

	var res cliResult
	if err := json.Unmarshal(stdout.Bytes(), &res); err != nil {
		return "", fmt.Errorf("failed to parse claude CLI output: %w (raw: %s)", err, stdout.String())
	}
	if res.IsError {
		return "", fmt.Errorf("claude returned an error: %s", res.Result)
	}
	return res.Result, nil
}

func (c *ClaudeCodeClient) GenerateSummary(ctx context.Context, input string) (string, error) {
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

	return c.invoke(ctx, prompt)
}

func (c *ClaudeCodeClient) GenerateResponse(ctx context.Context, query string, context string) (string, error) {
	prompt := fmt.Sprintf(`You are a coding assistant answering questions about a codebase.

Context:
%s

Question:
%s

Instructions:
- Answer based strictly on the provided context
- Reference specific functions, types, or files when relevant
- Be concise and precise`, context, query)

	return c.invoke(ctx, prompt)
}
