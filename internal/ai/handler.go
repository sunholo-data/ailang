package ai

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sunholo-data/ailang/internal/trace"
)

// Handler wraps a Provider for use with the effects.AIHandler interface.
// This bridges the unified AI package with AILANG's effect system.
//
// Thread-safety: single-threaded use within one evaluation. lastRoute is
// updated after every call and read immediately afterwards by the AI
// effect ops via LastRoutingMetadata().
type Handler struct {
	provider     Provider
	model        string
	systemPrompt string
	maxTokens    int

	// routingPolicy is attached to every outgoing Request when non-nil. Set
	// via WithRoutingPolicy. Only OpenRouter consumes it; other providers
	// reject a non-zero policy with ErrRoutingNotSupported. nil = no policy.
	routingPolicy *AIRoutingPolicy

	// lastRoute caches routing metadata from the most recent successful
	// Generate() call. nil when the response had no routing-distinct
	// metadata (direct providers, OR calls that didn't engage routing).
	lastRoute *trace.ResolvedRoute
}

// HandlerOption configures a Handler.
type HandlerOption func(*Handler)

// WithSystemPrompt sets the system prompt for all requests.
func WithSystemPrompt(prompt string) HandlerOption {
	return func(h *Handler) {
		h.systemPrompt = prompt
	}
}

// WithMaxTokens sets the maximum response tokens.
func WithMaxTokens(tokens int) HandlerOption {
	return func(h *Handler) {
		h.maxTokens = tokens
	}
}

// WithRoutingPolicy attaches an AIRoutingPolicy to every outgoing Request
// from this handler. The policy is consumed only by the OpenRouter provider;
// other providers will reject a non-zero policy with ErrRoutingNotSupported.
//
// Pass nil (or omit the option) for no policy.
func WithRoutingPolicy(p *AIRoutingPolicy) HandlerOption {
	return func(h *Handler) {
		h.routingPolicy = p
	}
}

// NewHandler creates a new Handler that wraps a Provider.
//
// The Handler implements effects.AIHandler, allowing any Provider to be
// used with AILANG's AI effect system.
//
// Example:
//
//	client := anthropic.NewClient(apiKey)
//	handler := ai.NewHandler(client, "claude-sonnet-4-5",
//	    ai.WithSystemPrompt("You are a helpful assistant."),
//	    ai.WithMaxTokens(4096),
//	)
//	effCtx.AI = effects.NewAIContext(handler)
func NewHandler(provider Provider, model string, opts ...HandlerOption) *Handler {
	h := &Handler{
		provider:  provider,
		model:     model,
		maxTokens: 4096, // Default
	}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

// Call implements effects.AIHandler.
// It sends the input to the provider and returns the generated text.
func (h *Handler) Call(input string) (string, error) {
	resp, err := h.provider.Generate(context.Background(), &Request{
		Model:        h.model,
		SystemPrompt: h.systemPrompt,
		UserPrompt:   input,
		MaxTokens:    h.maxTokens,
		Routing:      h.routingPolicy,
	})
	h.captureRoute(resp, err)
	if err != nil {
		return "", err
	}
	return resp.Text, nil
}

// captureRoute updates h.lastRoute from a Response. Called after every
// provider.Generate so the AI effect ops can attach routing metadata to
// trace events. We only populate lastRoute when the response has
// routing-distinct metadata (RequestedModel != Model OR CachedTokens > 0
// OR CostUSD != "" OR ResolvedProvider != ""); otherwise leave it nil so
// trace events for direct providers stay clean.
//
// On error we clear the previous metadata so a subsequent successful call
// from a non-routing provider doesn't inherit stale routing info.
func (h *Handler) captureRoute(resp *Response, err error) {
	if err != nil || resp == nil {
		h.lastRoute = nil
		return
	}

	hasRouting := resp.RequestedModel != "" && resp.RequestedModel != resp.Model
	hasRouting = hasRouting || resp.ResolvedProvider != ""
	hasRouting = hasRouting || resp.CachedTokens > 0
	hasRouting = hasRouting || resp.CostUSD != ""
	hasRouting = hasRouting || len(resp.FallbackChain) > 0

	if !hasRouting {
		h.lastRoute = nil
		return
	}

	h.lastRoute = &trace.ResolvedRoute{
		RequestedModel:   resp.RequestedModel,
		ResolvedModel:    resp.Model,
		ResolvedProvider: resp.ResolvedProvider,
		FallbackChain:    resp.FallbackChain,
		PromptTokens:     resp.InputTokens,
		CompletionTokens: resp.OutputTokens,
		CachedTokens:     resp.CachedTokens,
		ReasoningTokens:  resp.ReasonTokens,
		CostUSD:          resp.CostUSD,
	}
}

// LastRoutingMetadata returns routing info for the most recent Call,
// CallJson, or related operation, or nil if the call did not engage
// routing.
//
// Thread-safety: caller must invoke immediately after the matching
// Call/CallJson returns, before any other handler operation.
// Single-threaded use only — matches the AI handler contract.
func (h *Handler) LastRoutingMetadata() *trace.ResolvedRoute {
	return h.lastRoute
}

// jsonMaxTokensMinimum is the minimum max_tokens for JSON structured output.
// JSON responses are often larger than freeform text (keys, braces, quotes),
// so we use a higher floor to avoid truncation.
const jsonMaxTokensMinimum = 8192

// CallJson sends a request configured for JSON structured output.
// If schema is non-empty, providers enforce the schema on the response.
// If schema is empty, providers return valid JSON without schema enforcement.
//
// Uses at least 8192 max tokens (JSON responses need more room than freeform text).
// Trims whitespace from the response (some providers pad output to token boundary).
func (h *Handler) CallJson(input string, schema string) (string, error) {
	// JSON structured output needs more tokens than freeform text
	maxTokens := h.maxTokens
	if maxTokens < jsonMaxTokensMinimum {
		maxTokens = jsonMaxTokensMinimum
	}

	resp, err := h.provider.Generate(context.Background(), &Request{
		Model:          h.model,
		SystemPrompt:   h.systemPrompt,
		UserPrompt:     input,
		MaxTokens:      maxTokens,
		ResponseFormat: "json",
		ResponseSchema: schema,
		Routing:        h.routingPolicy,
	})
	h.captureRoute(resp, err)
	if err != nil {
		return "", err
	}
	// Trim whitespace: some providers (notably Gemini) pad structured output
	// with trailing spaces when approaching the token limit.
	return strings.TrimSpace(resp.Text), nil
}

// CallWithContext is like Call but accepts a context for cancellation/timeout.
func (h *Handler) CallWithContext(ctx context.Context, input string) (string, error) {
	resp, err := h.provider.Generate(ctx, &Request{
		Model:        h.model,
		SystemPrompt: h.systemPrompt,
		UserPrompt:   input,
		MaxTokens:    h.maxTokens,
		Routing:      h.routingPolicy,
	})
	h.captureRoute(resp, err)
	if err != nil {
		return "", err
	}
	return resp.Text, nil
}

// GenerateWithDetails returns the full response including token counts.
// This is useful for eval harness and cost tracking.
func (h *Handler) GenerateWithDetails(ctx context.Context, input string) (*Response, error) {
	return h.provider.Generate(ctx, &Request{
		Model:        h.model,
		SystemPrompt: h.systemPrompt,
		UserPrompt:   input,
		MaxTokens:    h.maxTokens,
		Routing:      h.routingPolicy,
	})
}

// CallImage generates an image and writes it to outputPath.
// Options is a JSON string: {"aspect_ratio": "16:9", "mime_type": "image/png"}.
func (h *Handler) CallImage(prompt, outputPath, options string) (string, error) {
	opts := parseImageOptions(options)
	resp, err := h.provider.Generate(context.Background(), &Request{
		Model:              h.model,
		SystemPrompt:       h.systemPrompt,
		UserPrompt:         prompt,
		ResponseModalities: []string{"IMAGE"},
		ImageOptions:       opts,
		Routing:            h.routingPolicy,
	})
	if err != nil {
		return "", err
	}
	if resp.ImageData == nil {
		return "", fmt.Errorf("provider returned no image data for prompt: %s", prompt)
	}
	// Ensure parent directory exists
	if dir := filepath.Dir(outputPath); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return "", fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}
	if err := os.WriteFile(outputPath, resp.ImageData, 0o644); err != nil {
		return "", fmt.Errorf("failed to write image to %s: %w", outputPath, err)
	}
	return outputPath, nil
}

// CallImageBase64 generates an image and returns JSON with base64 data.
// Returns: {"base64": "...", "mime_type": "image/png"}
func (h *Handler) CallImageBase64(prompt, options string) (string, error) {
	opts := parseImageOptions(options)
	resp, err := h.provider.Generate(context.Background(), &Request{
		Model:              h.model,
		SystemPrompt:       h.systemPrompt,
		UserPrompt:         prompt,
		ResponseModalities: []string{"IMAGE"},
		ImageOptions:       opts,
		Routing:            h.routingPolicy,
	})
	if err != nil {
		return "", err
	}
	if resp.ImageData == nil {
		return "", fmt.Errorf("provider returned no image data for prompt: %s", prompt)
	}
	b64 := base64.StdEncoding.EncodeToString(resp.ImageData)
	mime := resp.ImageMIME
	if mime == "" {
		mime = "image/png"
	}
	return fmt.Sprintf(`{"base64":"%s","mime_type":"%s"}`, b64, mime), nil
}

// parseImageOptions parses a JSON options string into ImageOptions.
func parseImageOptions(optionsJSON string) *ImageOptions {
	if optionsJSON == "" || optionsJSON == "{}" {
		return nil
	}
	var raw struct {
		AspectRatio string `json:"aspect_ratio"`
		MIMEType    string `json:"mime_type"`
	}
	if err := json.Unmarshal([]byte(optionsJSON), &raw); err != nil {
		return nil
	}
	if raw.AspectRatio == "" && raw.MIMEType == "" {
		return nil
	}
	return &ImageOptions{
		AspectRatio: raw.AspectRatio,
		MIMEType:    raw.MIMEType,
	}
}

// Provider returns the underlying provider.
func (h *Handler) Provider() Provider {
	return h.provider
}

// Model returns the model name.
func (h *Handler) Model() string {
	return h.model
}

// Step implements effects.AIHandler.Step (M-AI-TOOL-LOOP, v0.17.0). It
// dispatches to the bound provider's Step method, wiring the per-call
// model + handler-bound system prompt + max-tokens + routing policy onto
// the request, and forwards Messages + Tools verbatim. Errors flow back
// unchanged — the effect-op layer wraps them into *AIError before
// returning Err(AIError record) to AILANG.
//
// Note: the model parameter is per-call routable; the handler-bound model
// (h.model) is used only as a default when model == "".
func (h *Handler) Step(model string, messages []Message, tools []ToolSchema) (*Response, error) {
	chosenModel := model
	if chosenModel == "" {
		chosenModel = h.model
	}
	resp, err := h.provider.Step(context.Background(), &Request{
		Model:        chosenModel,
		SystemPrompt: h.systemPrompt,
		MaxTokens:    h.maxTokens,
		Routing:      h.routingPolicy,
		Messages:     messages,
		Tools:        tools,
	})
	h.captureRoute(resp, err)
	if err != nil {
		return nil, err
	}
	return resp, nil
}
