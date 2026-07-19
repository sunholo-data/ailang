// Package openrouter provides an OpenRouter API client implementing the ai.Provider interface.
//
// OpenRouter (https://openrouter.ai) is a unified gateway that fronts ~100 LLMs
// from many vendors behind an OpenAI-compatible Chat Completions API. This client
// is a thin Chat Completions adapter — it does not attempt to support OpenAI's
// Responses API or any vendor-specific extensions other than the OpenRouter-specific
// `cached_tokens` and `cost` fields surfaced in the response.
package openrouter

import "encoding/json"

// chatRequest represents the request body for OpenRouter's Chat Completions API.
// OpenRouter normalizes max_tokens across providers, so we use that field
// uniformly (no max_completion_tokens distinction).
type chatRequest struct {
	Model          string              `json:"model"`
	Messages       []chatMessage       `json:"messages"`
	MaxTokens      int                 `json:"max_tokens,omitempty"`
	Temperature    float64             `json:"temperature,omitempty"`
	Seed           *int64              `json:"seed,omitempty"`
	ResponseFormat *chatResponseFormat `json:"response_format,omitempty"`
	// Provider carries OpenRouter's dynamic routing config. Translated from
	// ai.AIRoutingPolicy by translatePolicy; nil when the caller did not
	// supply a routing policy.
	Provider *providerField `json:"provider,omitempty"`
	// Reasoning caps hidden thinking tokens for reasoning models
	// (https://openrouter.ai/docs/use-cases/reasoning-tokens). Nil = provider
	// default. Set from ai.Request.Options["reasoning_max_tokens"] so
	// always-thinking models (e.g. z-ai/glm-5.2) keep content headroom inside
	// max_tokens instead of burning the whole budget on thought.
	Reasoning *reasoningField `json:"reasoning,omitempty"`
}

// reasoningField configures OpenRouter's normalized reasoning controls.
// MaxTokens and Effort are mutually exclusive per the OpenRouter contract;
// when both are requested, Effort wins (it is the vendor-documented dial —
// e.g. Kimi K3's Low/Standard/High/Max — while max_tokens is best-effort).
type reasoningField struct {
	MaxTokens int    `json:"max_tokens,omitempty"`
	Effort    string `json:"effort,omitempty"`
}

// chatResponseFormat configures structured output.
type chatResponseFormat struct {
	Type       string          `json:"type"`                  // "json_schema" or "json_object"
	JSONSchema *chatJSONSchema `json:"json_schema,omitempty"` // Schema definition
}

// chatJSONSchema defines the JSON schema for structured output.
type chatJSONSchema struct {
	Name   string          `json:"name"`
	Schema json.RawMessage `json:"schema"`
	Strict bool            `json:"strict"`
}

// chatMessage represents a message in the Chat Completions API.
type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// chatResponse represents the response from OpenRouter's Chat Completions API.
//
// Provider is an OpenRouter extension reporting which underlying vendor
// served the request when routing is engaged (e.g. "Anthropic", "OpenAI").
// Not all responses include it; absent → empty string.
type chatResponse struct {
	ID       string       `json:"id"`
	Object   string       `json:"object"`
	Created  int64        `json:"created"`
	Model    string       `json:"model"`
	Provider string       `json:"provider,omitempty"`
	Choices  []chatChoice `json:"choices"`
	Usage    chatUsage    `json:"usage"`
}

// chatChoice represents a completion choice.
type chatChoice struct {
	Index        int         `json:"index"`
	Message      chatMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

// chatUsage represents OpenRouter's extended token usage block.
//
// In addition to the OpenAI-shape prompt/completion/total tokens, OpenRouter
// reports:
//   - prompt_tokens_details.cached_tokens — input tokens served from prompt cache
//   - cost — total inference cost in USD as a float (sum of upstream + markup)
//   - cost_details.upstream_inference_cost — upstream-only portion (informational)
type chatUsage struct {
	PromptTokens            int `json:"prompt_tokens"`
	CompletionTokens        int `json:"completion_tokens"`
	TotalTokens             int `json:"total_tokens"`
	CompletionTokensDetails struct {
		ReasoningTokens int `json:"reasoning_tokens"`
	} `json:"completion_tokens_details,omitempty"`
	// OpenRouter extensions:
	PromptTokensDetails struct {
		CachedTokens int `json:"cached_tokens"`
	} `json:"prompt_tokens_details,omitempty"`
	Cost        float64 `json:"cost,omitempty"` // OpenRouter reports total cost as a float
	CostDetails struct {
		UpstreamInferenceCost float64 `json:"upstream_inference_cost,omitempty"`
	} `json:"cost_details,omitempty"`
}

// errorResponse represents an error response from the API.
type errorResponse struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error"`
}
