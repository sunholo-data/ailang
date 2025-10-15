package eval_harness

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// OpenAI API structures
type openAIRequest struct {
	Model    string          `json:"model"`
	Messages []openAIMessage `json:"messages"`
	Seed     *int64          `json:"seed,omitempty"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIResponse struct {
	ID      string `json:"id"`
	Choices []struct {
		Message openAIMessage `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
		// GPT-5 reasoning models include reasoning_tokens in completion_tokens
		CompletionTokensDetails struct {
			ReasoningTokens int `json:"reasoning_tokens"`
		} `json:"completion_tokens_details,omitempty"`
	} `json:"usage"`
}

// callOpenAI makes a request to the OpenAI API
func (a *AIAgent) callOpenAI(ctx context.Context, prompt string) (*GenerateResult, error) {
	url := "https://api.openai.com/v1/chat/completions"

	req := openAIRequest{
		Model: a.model,
		Messages: []openAIMessage{
			{
				Role:    "system",
				Content: "You are a programming assistant. Generate ONLY code without explanations or markdown formatting.",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}

	// Add seed if provided
	if a.seed != 0 {
		req.Seed = &a.seed
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+a.apiKey)

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var apiResp openAIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(apiResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	code := apiResp.Choices[0].Message.Content

	// For GPT-5 models, completion_tokens includes reasoning_tokens
	// We need to subtract reasoning tokens to get actual output tokens
	outputTokens := apiResp.Usage.CompletionTokens
	reasoningTokens := apiResp.Usage.CompletionTokensDetails.ReasoningTokens
	if reasoningTokens > 0 {
		outputTokens = outputTokens - reasoningTokens
	}

	return &GenerateResult{
		Code:         extractCodeFromMarkdown(code),
		InputTokens:  apiResp.Usage.PromptTokens,
		OutputTokens: outputTokens,
		TotalTokens:  apiResp.Usage.TotalTokens,
		Model:        a.model,
	}, nil
}

// extractCodeFromMarkdown strips markdown code fences if present
func extractCodeFromMarkdown(text string) string {
	// Trim leading/trailing whitespace first
	text = strings.TrimSpace(text)
	lines := []byte(text)

	// Check if starts with ``` (after trimming)
	if len(lines) > 3 && lines[0] == '`' && lines[1] == '`' && lines[2] == '`' {
		// Find first newline (end of opening fence)
		start := 0
		for i, b := range lines {
			if b == '\n' {
				start = i + 1
				break
			}
		}

		// Find last ``` working backwards
		end := len(lines)
		for i := len(lines) - 1; i >= 2; i-- {
			if lines[i] == '`' && lines[i-1] == '`' && lines[i-2] == '`' {
				// Check if this is at start of line or has newline before it
				end = i - 2
				// Trim trailing newline before closing fence
				if end > 0 && lines[end-1] == '\n' {
					end--
				}
				break
			}
		}

		if start < end {
			return string(lines[start:end])
		}
	}

	return text
}
