package ai

import (
	"context"
	"errors"
	"testing"
)

// stubProvider is a minimal Provider for unit-testing handler.go's routing
// capture without needing the openrouter HTTP machinery.
type stubProvider struct {
	resp *Response
	err  error
}

func (s *stubProvider) Generate(_ context.Context, _ *Request) (*Response, error) {
	return s.resp, s.err
}

func (s *stubProvider) Name() string { return "stub" }

// TestHandler_LastRoutingMetadata_RoutedResponse verifies that a response
// with RequestedModel != Model populates lastRoute on the handler so the
// AI effect ops can attach it to a trace event.
func TestHandler_LastRoutingMetadata_RoutedResponse(t *testing.T) {
	provider := &stubProvider{
		resp: &Response{
			Text:             "hi",
			InputTokens:      100,
			OutputTokens:     25,
			CachedTokens:     60,
			CostUSD:          "0.000345",
			Model:            "anthropic/claude-sonnet-4.5",
			RequestedModel:   "openrouter/auto",
			ResolvedProvider: "Anthropic",
			FallbackChain:    []string{"anthropic/claude-sonnet-4.5"},
		},
	}
	h := NewHandler(provider, "openrouter/auto")
	if _, err := h.Call("ping"); err != nil {
		t.Fatalf("Call() error = %v", err)
	}
	got := h.LastRoutingMetadata()
	if got == nil {
		t.Fatal("LastRoutingMetadata() = nil, want populated for routed response")
	}
	if got.RequestedModel != "openrouter/auto" {
		t.Errorf("RequestedModel = %q", got.RequestedModel)
	}
	if got.ResolvedModel != "anthropic/claude-sonnet-4.5" {
		t.Errorf("ResolvedModel = %q", got.ResolvedModel)
	}
	if got.ResolvedProvider != "Anthropic" {
		t.Errorf("ResolvedProvider = %q", got.ResolvedProvider)
	}
	if got.CachedTokens != 60 {
		t.Errorf("CachedTokens = %d", got.CachedTokens)
	}
	if got.CostUSD != "0.000345" {
		t.Errorf("CostUSD = %q", got.CostUSD)
	}
}

// TestHandler_LastRoutingMetadata_DirectProviderResponse verifies that a
// response with no routing-distinct fields leaves lastRoute nil — direct
// providers (anthropic/openai/gemini) don't pollute trace events with
// empty ResolvedRoute payloads.
func TestHandler_LastRoutingMetadata_DirectProviderResponse(t *testing.T) {
	provider := &stubProvider{
		resp: &Response{
			Text:         "hi",
			InputTokens:  10,
			OutputTokens: 5,
			Model:        "claude-sonnet-4-5",
			// No RequestedModel, ResolvedProvider, CachedTokens, CostUSD,
			// or FallbackChain — direct provider with bare token counts.
		},
	}
	h := NewHandler(provider, "claude-sonnet-4-5")
	if _, err := h.Call("ping"); err != nil {
		t.Fatalf("Call() error = %v", err)
	}
	if got := h.LastRoutingMetadata(); got != nil {
		t.Errorf("LastRoutingMetadata() = %+v, want nil for direct-provider response", got)
	}
}

// TestHandler_LastRoutingMetadata_ErrorClearsRoute verifies that an error
// from Generate clears any prior routing metadata.
func TestHandler_LastRoutingMetadata_ErrorClearsRoute(t *testing.T) {
	provider := &stubProvider{
		resp: &Response{
			Model:          "anthropic/claude-sonnet-4.5",
			RequestedModel: "openrouter/auto",
		},
	}
	h := NewHandler(provider, "openrouter/auto")
	if _, err := h.Call("ping"); err != nil {
		t.Fatalf("Call() error = %v", err)
	}
	if h.LastRoutingMetadata() == nil {
		t.Fatal("setup: expected routing metadata after first call")
	}

	// Now flip the provider to error and verify cleanup.
	provider.resp = nil
	provider.err = errors.New("upstream unavailable")
	if _, err := h.Call("ping"); err == nil {
		t.Fatal("Call() expected error, got nil")
	}
	if got := h.LastRoutingMetadata(); got != nil {
		t.Errorf("LastRoutingMetadata() = %+v after error, want nil", got)
	}
}

// TestHandler_LastRoutingMetadata_OnlyCostUSDPopulates verifies that a
// response with just CostUSD set is enough to mark the call as routed
// (matches OpenRouter's most common shape: model unchanged but cost
// reported).
func TestHandler_LastRoutingMetadata_OnlyCostUSDPopulates(t *testing.T) {
	provider := &stubProvider{
		resp: &Response{
			Text:           "hi",
			Model:          "anthropic/claude-sonnet-4.5",
			RequestedModel: "anthropic/claude-sonnet-4.5", // same — common case
			CostUSD:        "0.0001",
		},
	}
	h := NewHandler(provider, "anthropic/claude-sonnet-4.5")
	if _, err := h.Call("ping"); err != nil {
		t.Fatalf("Call() error = %v", err)
	}
	got := h.LastRoutingMetadata()
	if got == nil {
		t.Fatal("LastRoutingMetadata() = nil, want populated when CostUSD is set")
	}
	if got.CostUSD != "0.0001" {
		t.Errorf("CostUSD = %q", got.CostUSD)
	}
}
