package eval_harness

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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

	return &GenerateResult{
		Code:   extractCodeFromMarkdown(code),
		Tokens: apiResp.Usage.TotalTokens,
		Model:  a.model,
	}, nil
}

// extractCodeFromMarkdown strips markdown code fences if present
func extractCodeFromMarkdown(text string) string {
	// Remove ```language\n ... \n``` blocks
	lines := []byte(text)

	// Simple heuristic: if starts with ```, skip first line and remove last ```
	if len(lines) > 3 && lines[0] == '`' && lines[1] == '`' && lines[2] == '`' {
		// Find first newline
		start := 0
		for i, b := range lines {
			if b == '\n' {
				start = i + 1
				break
			}
		}

		// Find last ```
		end := len(lines)
		for i := len(lines) - 1; i >= 0; i-- {
			if i >= 2 && lines[i] == '`' && lines[i-1] == '`' && lines[i-2] == '`' {
				end = i - 2
				// Trim trailing newline before ```
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
