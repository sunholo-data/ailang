package openrouter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/sunholo-data/ailang/internal/ai"
	"github.com/sunholo-data/ailang/internal/ai/openai"
)

// generateChat uses OpenRouter's Chat Completions API (/v1/chat/completions).
//
// OpenRouter's API is OpenAI-compatible. The wire format matches OpenAI Chat
// Completions exactly, with two notable extensions on the response:
//   - usage.prompt_tokens_details.cached_tokens — input tokens served from cache
//   - usage.cost — total inference cost in USD
//
// The optional HTTP-Referer and X-Title headers are an OpenRouter convention
// (used for app-leaderboard attribution). They are sent only when the
// OPENROUTER_HTTP_REFERER and OPENROUTER_X_TITLE environment variables are set.
func (c *Client) generateChat(ctx context.Context, req *ai.Request, reasoning ai.ReasoningDecision) (*ai.Response, error) {
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

	// Build request. OpenRouter normalizes max_tokens for upstream providers,
	// so we always use MaxTokens (no max_completion_tokens distinction).
	apiReq := chatRequest{
		Model:    req.Model,
		Messages: messages,
	}

	maxTokens := req.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 4096
	}
	apiReq.MaxTokens = maxTokens

	if req.Temperature > 0 {
		apiReq.Temperature = req.Temperature
	}

	// Check for seed in options
	if req.Options != nil {
		if seed, ok := req.Options["seed"].(int64); ok {
			apiReq.Seed = &seed
		}
	}

	// M-AI-REASONING-EFFORT (M5): the previous untyped "Effort-wins" branch
	// (Options["reasoning_effort"] silently beating reasoning_max_tokens, with
	// no validation) is REPLACED by the shared fail-loud resolver. All four
	// reasoning inputs are validated with deterministic precedence/conflict
	// rules before dispatch; effort maps to reasoning.effort, the deprecated
	// reasoning_max_tokens alone maps to reasoning.max_tokens (today's body),
	// and any conflicting combination fails loudly.
	switch reasoning.Kind {
	case ai.ReasoningEffortKind:
		apiReq.Reasoning = &reasoningField{Effort: reasoning.Effort}
	case ai.ReasoningMaxTokensKind:
		apiReq.Reasoning = &reasoningField{MaxTokens: reasoning.MaxTokensReasoning}
	}

	// Translate optional routing policy. Nil when no policy or zero policy.
	// A malformed max-price cap fails loud rather than shipping a request that
	// silently ignores the caller's cost guard.
	provider, rerr := translatePolicy(req.Routing)
	if rerr != nil {
		return nil, ai.NewAIError(ai.CodeSchemaValidation,
			fmt.Sprintf("openrouter: invalid routing policy: %v", rerr), false)
	}
	apiReq.Provider = provider

	// Add structured output configuration
	if req.ResponseFormat == "json" {
		if req.ResponseSchema != "" {
			apiReq.ResponseFormat = &chatResponseFormat{
				Type: "json_schema",
				JSONSchema: &chatJSONSchema{
					Name:   "response",
					Schema: json.RawMessage(req.ResponseSchema),
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
		return nil, ai.NewProviderError("openrouter", 0, "failed to marshal request", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat/completions", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, ai.NewProviderError("openrouter", 0, "failed to create request", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	setAttributionHeaders(httpReq, req.Attribution)

	// Execute request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, ai.NewProviderError("openrouter", 0, "request failed", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, ai.NewProviderError("openrouter", resp.StatusCode, "failed to read response", err)
	}

	// Handle errors
	if resp.StatusCode != http.StatusOK {
		var errResp errorResponse
		if json.Unmarshal(body, &errResp) == nil && errResp.Error.Message != "" {
			return nil, ai.NewProviderError("openrouter", resp.StatusCode, errResp.Error.Message, nil)
		}
		return nil, ai.NewProviderError("openrouter", resp.StatusCode, string(body), nil)
	}

	// Parse successful response
	var result chatResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, ai.NewProviderError("openrouter", 0, "failed to parse response", err)
	}

	if len(result.Choices) == 0 {
		return nil, ai.NewProviderError("openrouter", 0, "no choices in response", nil)
	}

	text := result.Choices[0].Message.Content

	// Calculate output tokens. For reasoning models, completion_tokens
	// includes reasoning_tokens — split them out the same way openai does.
	// Some upstreams report reasoning_tokens > completion_tokens (observed on
	// z-ai via OpenRouter); clamp at 0 rather than banking a negative count.
	outputTokens := result.Usage.CompletionTokens
	reasoningTokens := result.Usage.CompletionTokensDetails.ReasoningTokens
	if reasoningTokens > 0 {
		outputTokens = max(outputTokens-reasoningTokens, 0)
	}

	// Cost is reported as a float by OpenRouter; preserve precision via
	// strconv.FormatFloat. Empty string when the upstream did not report cost.
	costUSD := ""
	if result.Usage.Cost != 0 {
		costUSD = strconv.FormatFloat(result.Usage.Cost, 'f', -1, 64)
	}

	// FallbackChain: OpenRouter doesn't reliably surface a true fallback
	// chain in standard responses. For successful calls we report the
	// resolved model as a single-element chain so consumers always see a
	// non-empty list when routing was active. If we adopt a richer signal
	// later, this is the place to populate it.
	var fallbackChain []string
	if result.Model != "" {
		fallbackChain = []string{result.Model}
	}

	return &ai.Response{
		Text:             text,
		InputTokens:      result.Usage.PromptTokens,
		OutputTokens:     outputTokens,
		TotalTokens:      result.Usage.TotalTokens,
		ReasonTokens:     reasoningTokens,
		FinishReason:     openai.MapChatFinishReason(result.Choices[0].FinishReason),
		CachedTokens:     result.Usage.PromptTokensDetails.CachedTokens,
		CostUSD:          costUSD,
		Model:            result.Model,
		RequestedModel:   req.Model,
		ResolvedProvider: result.Provider,
		FallbackChain:    fallbackChain,
	}, nil
}
