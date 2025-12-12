package ai

import (
	"context"
)

// Handler wraps a Provider for use with the effects.AIHandler interface.
// This bridges the unified AI package with AILANG's effect system.
type Handler struct {
	provider     Provider
	model        string
	systemPrompt string
	maxTokens    int
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
	})
	if err != nil {
		return "", err
	}
	return resp.Text, nil
}

// CallWithContext is like Call but accepts a context for cancellation/timeout.
func (h *Handler) CallWithContext(ctx context.Context, input string) (string, error) {
	resp, err := h.provider.Generate(ctx, &Request{
		Model:        h.model,
		SystemPrompt: h.systemPrompt,
		UserPrompt:   input,
		MaxTokens:    h.maxTokens,
	})
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
	})
}

// Provider returns the underlying provider.
func (h *Handler) Provider() Provider {
	return h.provider
}

// Model returns the model name.
func (h *Handler) Model() string {
	return h.model
}
