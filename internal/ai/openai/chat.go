package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/sunholo/ailang/internal/ai"
)

// generateChat uses the Chat Completions API (/v1/chat/completions).
func (c *Client) generateChat(ctx context.Context, req *ai.Request) (*ai.Response, error) {
	// Build messages
	var messages []chatMessage

	if req.SystemPrompt != "" {
		messages = append(messages, chatMessage{
			Role:    "system",
			Content: req.SystemPrompt,
		})
	}

	messages = append(messages, chatMessage{
		Role:    "user",
		Content: req.UserPrompt,
	})

	// Build request
	apiReq := chatRequest{
		Model:    req.Model,
		Messages: messages,
	}

	// Set max tokens based on model type
	maxTokens := req.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 4096
	}

	if usesMaxCompletionTokens(req.Model) {
		apiReq.MaxCompletionTokens = maxTokens
	} else {
		apiReq.MaxTokens = maxTokens
	}

	if req.Temperature > 0 {
		apiReq.Temperature = req.Temperature
	}

	// Check for seed in options
	if req.Options != nil {
		if seed, ok := req.Options["seed"].(int64); ok {
			apiReq.Seed = &seed
		}
	}

	// Add structured output configuration
	if req.ResponseFormat == "json" {
		if req.ResponseSchema != "" {
			schema := ensureStrictSchemaCompliance(json.RawMessage(req.ResponseSchema))
			apiReq.ResponseFormat = &chatResponseFormat{
				Type: "json_schema",
				JSONSchema: &chatJSONSchema{
					Name:   "response",
					Schema: schema,
					Strict: true,
				},
			}
		} else {
			apiReq.ResponseFormat = &chatResponseFormat{
				Type: "json_object",
			}
		}
	}

	// Marshal request
	jsonBody, err := json.Marshal(apiReq)
	if err != nil {
		return nil, ai.NewProviderError("openai", 0, "failed to marshal request", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat/completions", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, ai.NewProviderError("openai", 0, "failed to create request", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	// Execute request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, ai.NewProviderError("openai", 0, "request failed", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, ai.NewProviderError("openai", resp.StatusCode, "failed to read response", err)
	}

	// Handle errors
	if resp.StatusCode != http.StatusOK {
		var errResp errorResponse
		if json.Unmarshal(body, &errResp) == nil && errResp.Error.Message != "" {
			return nil, ai.NewProviderError("openai", resp.StatusCode, errResp.Error.Message, nil)
		}
		return nil, ai.NewProviderError("openai", resp.StatusCode, string(body), nil)
	}

	// Parse successful response
	var result chatResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, ai.NewProviderError("openai", 0, "failed to parse response", err)
	}

	if len(result.Choices) == 0 {
		return nil, ai.NewProviderError("openai", 0, "no choices in response", nil)
	}

	text := result.Choices[0].Message.Content

	// Calculate output tokens
	// For GPT-5+ reasoning models, completion_tokens includes reasoning_tokens
	outputTokens := result.Usage.CompletionTokens
	reasoningTokens := result.Usage.CompletionTokensDetails.ReasoningTokens
	if reasoningTokens > 0 {
		outputTokens = outputTokens - reasoningTokens
	}

	return &ai.Response{
		Text:         text,
		InputTokens:  result.Usage.PromptTokens,
		OutputTokens: outputTokens,
		TotalTokens:  result.Usage.TotalTokens,
		ReasonTokens: reasoningTokens,
		Model:        result.Model,
	}, nil
}
