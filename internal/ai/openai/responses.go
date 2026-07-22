package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/sunholo-data/ailang/internal/ai"
)

// generateResponses uses the Responses API (/v1/responses).
// This is used for codex models that support autonomous operation and reasoning.
func (c *Client) generateResponses(ctx context.Context, req *ai.Request, reasoning ai.ReasoningDecision) (*ai.Response, error) {
	// Build input array with developer/user roles
	var input []responsesInput

	// Map SystemPrompt to "developer" role (Responses API equivalent of "system")
	if req.SystemPrompt != "" {
		input = append(input, responsesInput{
			Role:    "developer",
			Content: req.SystemPrompt,
		})
	}

	input = append(input, responsesInput{
		Role:    "user",
		Content: req.UserPrompt,
	})

	// Build request
	apiReq := responsesRequest{
		Model: req.Model,
		Input: input,
	}

	// Set max tokens if specified
	maxTokens := req.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 16384 // Codex models support larger outputs
	}
	apiReq.MaxTokens = maxTokens

	// Set reasoning effort. The legacy Options["reasoning_effort"] parsing has
	// moved into the shared resolver (ai.ResolveReasoning), which validates the
	// value before we reach here. When no reasoning control is supplied
	// (ReasoningNone) we preserve the historical implicit "medium" block exactly
	// — the sole compatibility default. A resolved qualitative effort replaces
	// it with the requested value.
	effort := "medium"
	if reasoning.Kind == ai.ReasoningEffortKind {
		effort = reasoning.Effort
	}
	apiReq.Reasoning = &responsesReasoning{Effort: effort}

	// Add structured output configuration
	if req.ResponseFormat == "json" {
		if req.ResponseSchema != "" {
			schema := ensureStrictSchemaCompliance(json.RawMessage(req.ResponseSchema))
			apiReq.Text = &responsesText{
				Format: responsesTextFormat{
					Type:   "json_schema",
					Name:   "response",
					Schema: schema,
					Strict: true,
				},
			}
		} else {
			apiReq.Text = &responsesText{
				Format: responsesTextFormat{
					Type: "json_object",
				},
			}
		}
	}

	// Marshal request
	jsonBody, err := json.Marshal(apiReq)
	if err != nil {
		return nil, ai.NewProviderError("openai", 0, "failed to marshal request", err)
	}

	// Create HTTP request to /v1/responses endpoint
	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/responses", bytes.NewReader(jsonBody))
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
	var result responsesResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, ai.NewProviderError("openai", 0, "failed to parse response", err)
	}

	// Extract text from polymorphic output items
	// Output can contain "message", "reasoning", "function_call" types
	var textBuilder strings.Builder
	for _, item := range result.Output {
		if item.Type == "message" && item.Role == "assistant" {
			for _, content := range item.Content {
				if content.Type == "output_text" {
					if textBuilder.Len() > 0 {
						textBuilder.WriteString("\n")
					}
					textBuilder.WriteString(content.Text)
				}
			}
		}
	}

	text := textBuilder.String()
	if text == "" {
		return nil, ai.NewProviderError("openai", 0, "no text output in response", nil)
	}

	// Calculate output tokens (subtract reasoning tokens from total output)
	outputTokens := result.Usage.OutputTokens
	reasoningTokens := result.Usage.OutputDetails.ReasoningTokens
	if reasoningTokens > 0 {
		outputTokens = outputTokens - reasoningTokens
	}

	return &ai.Response{
		Text:         text,
		InputTokens:  result.Usage.InputTokens,
		OutputTokens: outputTokens,
		TotalTokens:  result.Usage.TotalTokens,
		ReasonTokens: reasoningTokens,
		Model:        result.Model,
	}, nil
}
