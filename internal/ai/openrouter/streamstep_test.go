package openrouter

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/ai"
)

// TestStreamStep_BasicContentAndUsage verifies that OpenRouter StreamStep
// parses an OpenAI-shape SSE stream end-to-end and produces a typed
// StepResult identical in shape to openai.StreamStep.
//
// OpenRouter speaks the OpenAI Chat Completions wire format regardless of
// the routed-to provider — anthropic/* models still emit OpenAI-shape
// chunks via OpenRouter's translation layer.
func TestStreamStep_BasicContentAndUsage(t *testing.T) {
	sseBody := strings.Join([]string{
		`data: {"id":"orchat-1","object":"chat.completion.chunk","model":"anthropic/claude-sonnet-4.5","choices":[{"index":0,"delta":{"role":"assistant","content":"Hi"},"finish_reason":null}]}`,
		``,
		`data: {"id":"orchat-1","object":"chat.completion.chunk","model":"anthropic/claude-sonnet-4.5","choices":[{"index":0,"delta":{"content":" there"},"finish_reason":null}]}`,
		``,
		`data: {"id":"orchat-1","object":"chat.completion.chunk","model":"anthropic/claude-sonnet-4.5","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`,
		``,
		`data: {"id":"orchat-1","object":"chat.completion.chunk","model":"anthropic/claude-sonnet-4.5","choices":[],"usage":{"prompt_tokens":30,"completion_tokens":2,"total_tokens":32}}`,
		``,
		`data: [DONE]`,
		``,
	}, "\n")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprint(w, sseBody)
	}))
	defer srv.Close()

	client := NewClient("test-key", WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))

	var contentChunks []string
	var usageChunks []ai.StreamUsage
	resp, err := client.StreamStep(context.Background(), &ai.Request{
		Model:    "anthropic/claude-sonnet-4.5",
		Messages: []ai.Message{{Role: "user", Content: "say hi"}},
	}, func(chunk ai.StreamChunk) {
		switch c := chunk.(type) {
		case ai.StreamContentDelta:
			contentChunks = append(contentChunks, c.Text)
		case ai.StreamUsage:
			usageChunks = append(usageChunks, c)
		}
	})
	if err != nil {
		t.Fatalf("StreamStep returned error: %v", err)
	}

	if resp.Text != "Hi there" {
		t.Errorf("resp.Text = %q, want %q", resp.Text, "Hi there")
	}
	if len(contentChunks) != 2 {
		t.Errorf("ContentDelta count = %d, want 2", len(contentChunks))
	}
	if len(usageChunks) != 1 {
		t.Fatalf("Usage chunk count = %d, want 1", len(usageChunks))
	}
	if usageChunks[0].InputTokens != 30 || usageChunks[0].OutputTokens != 2 {
		t.Errorf("Usage tokens = %d/%d, want 30/2", usageChunks[0].InputTokens, usageChunks[0].OutputTokens)
	}
	if resp.RequestedModel != "anthropic/claude-sonnet-4.5" {
		t.Errorf("resp.RequestedModel = %q, want anthropic/claude-sonnet-4.5", resp.RequestedModel)
	}
	if resp.FinishReason != "stop" {
		t.Errorf("resp.FinishReason = %q, want stop", resp.FinishReason)
	}
}

// TestStreamStep_RoutingPolicyOnTheWire verifies that req.Routing translates
// to a top-level "provider" field in the wire body — the same composition
// the non-streaming Step uses, ensuring routing + streaming compose cleanly.
func TestStreamStep_RoutingPolicyOnTheWire(t *testing.T) {
	var capturedBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(buf)
		capturedBody = buf
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprint(w, `data: {"choices":[{"delta":{},"finish_reason":"stop"}]}`+"\n\n"+`data: [DONE]`+"\n\n")
	}))
	defer srv.Close()

	client := NewClient("k", WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))
	_, err := client.StreamStep(context.Background(), &ai.Request{
		Model:    "openrouter/auto",
		Messages: []ai.Message{{Role: "user", Content: "x"}},
		Routing: &ai.AIRoutingPolicy{
			Order: []string{"Anthropic", "OpenAI"},
		},
	}, func(_ ai.StreamChunk) {})
	if err != nil {
		t.Fatalf("StreamStep returned error: %v", err)
	}
	body := string(capturedBody)
	if !strings.Contains(body, `"stream":true`) {
		t.Errorf("missing stream:true: %s", body)
	}
	if !strings.Contains(body, `"include_usage":true`) {
		t.Errorf("missing include_usage:true: %s", body)
	}
	if !strings.Contains(body, `"provider":`) {
		t.Errorf("missing provider field (routing policy not translated): %s", body)
	}
	if !strings.Contains(body, `"Anthropic"`) {
		t.Errorf("provider order not preserved: %s", body)
	}
}

// TestStreamStep_IncludeReasoningOnTheWire verifies that StreamStep
// always sends "include_reasoning":true so OpenRouter-routed thinking
// models surface reasoning chunks via delta.reasoning. Without this
// flag, deepseek-r1 / anthropic-via-OR / qwen-thinking all silently
// drop reasoning regardless of the underlying provider's native field.
// (M-AI-OPENROUTER-REASONING-FIELD, v0.18.9)
func TestStreamStep_IncludeReasoningOnTheWire(t *testing.T) {
	var capturedBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(buf)
		capturedBody = buf
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprint(w, `data: {"choices":[{"delta":{},"finish_reason":"stop"}]}`+"\n\n"+`data: [DONE]`+"\n\n")
	}))
	defer srv.Close()

	client := NewClient("k", WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))
	_, err := client.StreamStep(context.Background(), &ai.Request{
		Model:    "deepseek/deepseek-r1",
		Messages: []ai.Message{{Role: "user", Content: "x"}},
	}, func(_ ai.StreamChunk) {})
	if err != nil {
		t.Fatalf("StreamStep returned error: %v", err)
	}
	body := string(capturedBody)
	if !strings.Contains(body, `"include_reasoning":true`) {
		t.Errorf("missing include_reasoning:true (OR drops reasoning silently without it): %s", body)
	}
	if !strings.Contains(body, `"stream":true`) {
		t.Errorf("missing stream:true: %s", body)
	}
}

// TestStreamStep_HTTPError verifies the error envelope is hoisted.
func TestStreamStep_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_, _ = fmt.Fprint(w, `{"error":{"message":"insufficient credits","type":"forbidden"}}`)
	}))
	defer srv.Close()

	client := NewClient("k", WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))
	_, err := client.StreamStep(context.Background(), &ai.Request{
		Model:    "openrouter/auto",
		Messages: []ai.Message{{Role: "user", Content: "x"}},
	}, func(_ ai.StreamChunk) {})
	if err == nil {
		t.Fatal("StreamStep returned nil error for 403 response")
	}
	aiErr, ok := err.(*ai.AIError)
	if !ok {
		t.Fatalf("error type = %T, want *ai.AIError", err)
	}
	if !strings.Contains(aiErr.Message, "insufficient credits") {
		t.Errorf("AIError.Message = %q, want it to contain 'insufficient credits'", aiErr.Message)
	}
}
