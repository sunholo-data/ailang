// Package ai provides a unified interface for AI provider implementations.
//
// This package is designed to be used by:
//   - Eval harness (internal/eval_harness/)
//   - CLI --ai flag (cmd/ailang/)
//   - std/ai effect (internal/effects/)
//
// Each provider (openai, gemini, anthropic) implements the Provider interface
// and can be wrapped with Handler for use with the effects system.
package ai

import (
	"context"
	"fmt"
)

// Request represents a generic AI request.
//
// Most fields map directly onto provider-native parameters. The optional
// Routing field carries an AIRoutingPolicy for dynamic provider selection
// — currently consumed only by OpenRouter; other providers reject a
// non-zero policy with ErrRoutingNotSupported.
type Request struct {
	// Model is the model name (e.g., "gemini-2.5-flash", "gpt-5", "claude-sonnet-4-5")
	Model string

	// SystemPrompt contains system/developer instructions
	SystemPrompt string

	// UserPrompt contains the user message
	UserPrompt string

	// MaxTokens is the maximum number of response tokens (0 = provider default)
	MaxTokens int

	// Temperature controls randomness (0.0-2.0, 0 = provider default)
	Temperature float64

	// ResponseFormat controls structured output. Values: "json" or "" (text).
	// When set to "json", providers configure their native structured output.
	ResponseFormat string

	// ResponseSchema is an optional JSON Schema string for structured output.
	// When provided with ResponseFormat="json", providers enforce this schema.
	// When empty with ResponseFormat="json", providers return valid JSON without schema.
	ResponseSchema string

	// ResponseModalities controls output types. Values: ["TEXT"], ["IMAGE"], ["TEXT", "IMAGE"].
	// When set to ["IMAGE"], providers that support image generation will return image data.
	ResponseModalities []string

	// ImageOptions configures image generation parameters (used with ResponseModalities containing "IMAGE").
	ImageOptions *ImageOptions

	// Options contains provider-specific options
	Options map[string]any

	// Routing is an optional dynamic-routing policy. When set with HasRouting()
	// returning true, only providers that support routing (currently: openrouter)
	// will accept the request. Other providers return ErrRoutingNotSupported.
	Routing *AIRoutingPolicy

	// Messages, when non-empty, supersedes SystemPrompt + UserPrompt.
	// Used for multi-turn conversations and Provider.Step tool dispatch.
	// Adapters that see a non-empty Messages MUST use it; legacy single-shot
	// callers that only set SystemPrompt + UserPrompt continue to work
	// unchanged on the Generate path.
	Messages []Message

	// Tools advertises tool schemas the model may call. Empty = no tools.
	// Only consulted by Provider.Step. Providers that do not support tools
	// (currently: ollama) return AIError{Code: CodeToolsNotSupported,
	// Retryable: false} when len(Tools) > 0.
	Tools []ToolSchema
}

// Message is one entry in a multi-turn AI conversation.
//
//	Role           string ("user" | "assistant" | "tool" | "system")
//	Content        prose content (may be empty for assistant messages
//	               whose ToolCalls is non-empty)
//	ToolCalls      assistant only — tool invocations the model emitted
//	ToolCallID     "tool" role only — references a prior ToolCall.ID
type Message struct {
	Role       string
	Content    string
	ToolCalls  []ToolCall
	ToolCallID string
}

// ToolSchema is a JSON-Schema-described tool the model may call.
// Parameters is the raw JSON Schema string (same shape as Request.ResponseSchema).
type ToolSchema struct {
	Name        string
	Description string
	Parameters  string // JSON Schema as a string
}

// ToolCall is a single tool invocation emitted by the model in a Step
// response. The host dispatches it and feeds the result back as a Message
// with Role="tool" and ToolCallID matching this ID.
type ToolCall struct {
	ID        string // provider-assigned (Anthropic, OpenAI) or adapter-generated (Gemini)
	Name      string // tool name from the advertised ToolSchema
	Arguments string // JSON-encoded; caller decodes per the advertised parameters schema
}

// ImageOptions configures image generation parameters.
type ImageOptions struct {
	// AspectRatio controls the image aspect ratio (e.g., "1:1", "16:9", "9:16").
	AspectRatio string

	// MIMEType controls the output format (e.g., "image/png", "image/jpeg").
	MIMEType string
}

// Response represents a generic AI response.
type Response struct {
	// Text is the generated text content
	Text string

	// ImageData contains raw image bytes (PNG/JPEG) when the response includes an image.
	// Nil for text-only responses.
	ImageData []byte

	// ImageMIME is the MIME type of ImageData (e.g., "image/png").
	// Empty for text-only responses.
	ImageMIME string

	// InputTokens is the number of prompt/input tokens
	InputTokens int

	// OutputTokens is the number of completion/output tokens
	OutputTokens int

	// TotalTokens is InputTokens + OutputTokens
	TotalTokens int

	// ReasonTokens is the number of reasoning tokens (for o1/codex models)
	ReasonTokens int

	// CachedTokens is the number of cached input tokens (provider-specific).
	// OpenRouter reports this in usage.prompt_tokens_details.cached_tokens.
	// Other providers may leave this as 0.
	//
	// DEPRECATED: prefer CacheReadInputTokens (same data, normalized name).
	// Kept for back-compat with the OpenRouter cost-normalization path.
	CachedTokens int

	// CacheReadInputTokens is the number of input tokens served from the
	// provider's prompt cache (a "cache hit"). Anthropic surfaces this as
	// usage.cache_read_input_tokens; OpenAI as usage.prompt_tokens_details.cached_tokens;
	// Gemini as usageMetadata.cachedContentTokenCount. Providers that don't
	// support prompt caching (or didn't this turn) report 0.
	CacheReadInputTokens int

	// CacheCreationInputTokens is the number of input tokens written into
	// the provider's prompt cache this turn (a "cache write"). Anthropic-
	// specific (usage.cache_creation_input_tokens); other providers leave 0.
	// Useful for distinguishing "cold cache" vs "warm cache" cost profiles
	// in eval-harness telemetry.
	CacheCreationInputTokens int

	// CostUSD is the inference cost in USD as reported by the provider.
	// Stored as a string to preserve provider-reported precision.
	// OpenRouter reports this in usage.cost. Empty string if not reported.
	CostUSD string

	// Model is the model that was actually used (may differ from request)
	Model string

	// RequestedModel is the model the caller asked for. May differ from
	// Model (which is what the provider actually used) when routing is in
	// effect. OpenRouter is the only provider where these can differ today;
	// direct providers leave this empty (in which case Model is the
	// requested-and-used model).
	RequestedModel string

	// ResolvedProvider is the underlying provider that ultimately served
	// the request (e.g., "anthropic" when OpenRouter routed to
	// claude-sonnet-4.5). Empty when not applicable or not reported.
	ResolvedProvider string

	// FallbackChain lists the models tried in order before settling on
	// Model. Empty for direct providers; usually [Model] for successful
	// first-try OpenRouter calls. Reserved for future use when richer
	// fallback signals become available.
	FallbackChain []string

	// ToolCalls, when non-empty, indicates the model wants the host to
	// dispatch these tools and feed results back via a follow-up Step call.
	// Only populated by Provider.Step responses; Generate leaves this nil.
	ToolCalls []ToolCall

	// FinishReason is the normalized stop reason. One of:
	//   "stop"        — natural end of model turn (no tool calls)
	//   "tool_calls"  — model emitted tool calls; loop driver should dispatch
	//   "length"      — truncated by max_tokens
	//   "error"       — provider-side error after partial response
	// Empty on legacy Generate responses (back-compat).
	FinishReason string
}

// RequestsImage returns true if the request asks for image generation.
func RequestsImage(req *Request) bool {
	for _, m := range req.ResponseModalities {
		if m == "IMAGE" {
			return true
		}
	}
	return false
}

// Provider interface for AI providers.
// Each provider (openai, gemini, anthropic) implements this interface.
type Provider interface {
	// Generate makes a single completion request.
	// The context can be used for cancellation and timeouts.
	Generate(ctx context.Context, req *Request) (*Response, error)

	// Step is the multi-turn / tool-aware variant of Generate, added by
	// M-AI-TOOL-LOOP (v0.17.0). It uses req.Messages and req.Tools, and
	// populates resp.ToolCalls + resp.FinishReason. Adapters that have
	// not yet implemented Step return an *AIError with Code = CodeInternal
	// and a "not yet implemented" message; adapters that fundamentally
	// cannot support tools (e.g. ollama) return CodeToolsNotSupported when
	// len(req.Tools) > 0 and otherwise fall through to Generate.
	Step(ctx context.Context, req *Request) (*Response, error)

	// Name returns the provider name (e.g., "openai", "gemini", "anthropic")
	Name() string
}

// ProviderError represents an error from an AI provider.
type ProviderError struct {
	Provider   string // Provider name
	StatusCode int    // HTTP status code (0 if not applicable)
	Message    string // Error message
	Err        error  // Underlying error (may be nil)
}

func (e *ProviderError) Error() string {
	if e.StatusCode > 0 {
		return fmt.Sprintf("%s error (%d): %s", e.Provider, e.StatusCode, e.Message)
	}
	return fmt.Sprintf("%s error: %s", e.Provider, e.Message)
}

func (e *ProviderError) Unwrap() error {
	return e.Err
}

// NewProviderError creates a new ProviderError.
func NewProviderError(provider string, statusCode int, message string, err error) *ProviderError {
	return &ProviderError{
		Provider:   provider,
		StatusCode: statusCode,
		Message:    message,
		Err:        err,
	}
}
