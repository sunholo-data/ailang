package openai

import (
	"context"
	"encoding/json"

	"github.com/sunholo-data/ailang/internal/ai"
)

// generateChat uses the Chat Completions API (/v1/chat/completions).
func (c *Client) generateChat(ctx context.Context, req *ai.Request, reasoning ai.ReasoningDecision) (*ai.Response, error) {
	// Build messages
	var messages []ChatMessage

	if req.SystemPrompt != "" {
		messages = append(messages, ChatMessage{
			Role:    "system",
			Content: req.SystemPrompt,
		})
	}

	messages = append(messages, ChatMessage{
		Role:    "user",
		Content: req.FullUserPrompt(),
	})

	// Build request
	apiReq := ChatRequest{
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

	// M-AI-REASONING-EFFORT: apply the resolved qualitative effort as OpenAI
	// Chat's native top-level reasoning_effort field. ReasoningNone leaves it
	// unset (omitempty) => byte-identical body.
	if reasoning.Kind == ai.ReasoningEffortKind {
		apiReq.ReasoningEffort = reasoning.Effort
	}

	// Add structured output configuration
	if req.ResponseFormat == "json" {
		if req.ResponseSchema != "" {
			schema := ensureStrictSchemaCompliance(json.RawMessage(req.ResponseSchema))
			apiReq.ResponseFormat = &ChatResponseFormat{
				Type: "json_schema",
				JSONSchema: &ChatJSONSchema{
					Name:   "response",
					Schema: schema,
					Strict: true,
				},
			}
		} else {
			apiReq.ResponseFormat = &ChatResponseFormat{
				Type: "json_object",
			}
		}
	}

	jsonBody, err := json.Marshal(apiReq)
	if err != nil {
		return nil, ai.NewProviderError("openai", 0, "failed to marshal request", err)
	}

	// Wall time covers request-send → body fully read: the per-call latency
	// datum for route A/Bs (M-LYCEUM-PROVIDER M3). Non-streaming, so TTFT is
	// unobservable client-side and stays 0 (unmeasured, not instant).
	var result ChatResponse
	res, err := ai.DoJSON(ctx, ai.JSONCall{
		Provider: "openai",
		Client:   c.httpClient,
		URL:      c.baseURL + "/chat/completions",
		Headers:  c.authHeader(),
		Body:     jsonBody,
	}, &result)
	if err != nil {
		return nil, err
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
		Text:                 text,
		InputTokens:          result.Usage.PromptTokens,
		CacheReadInputTokens: result.Usage.PromptTokensDetails.CachedTokens,
		OutputTokens:         outputTokens,
		TotalTokens:          result.Usage.TotalTokens,
		ReasonTokens:         reasoningTokens,
		FinishReason:         MapChatFinishReason(result.Choices[0].FinishReason),
		Model:                result.Model,
		WallMS:               res.WallMS,
	}, nil
}
