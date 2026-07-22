package openrouter

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/ai"
)

const okChatJSON = `{"id":"g","object":"chat.completion","model":"m","choices":[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}],"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}}`

func captureBody(t *testing.T, captured *string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		*captured = string(b)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(okChatJSON))
	}))
}

// TestOpenRouter_Golden_NoReasoning_ByteIdentical (AC14): unset Generate body
// carries no reasoning key.
func TestOpenRouter_Golden_NoReasoning_ByteIdentical(t *testing.T) {
	var body string
	srv := captureBody(t, &body)
	defer srv.Close()

	c := NewClient("k", WithBaseURL(srv.URL))
	_, err := c.Generate(context.Background(), &ai.Request{Model: "m", UserPrompt: "hi", MaxTokens: 100})
	if err != nil {
		t.Fatalf("Generate error = %v", err)
	}
	if strings.Contains(body, "reasoning") {
		t.Fatalf("unset body leaked reasoning: %s", body)
	}
}

// TestOpenRouter_ReasoningMaxTokens_Alone_Preserved (AC7/AC14): reasoning_max_tokens
// alone preserves today's reasoning.max_tokens body exactly (Generate path).
func TestOpenRouter_ReasoningMaxTokens_Alone_Preserved(t *testing.T) {
	var body string
	srv := captureBody(t, &body)
	defer srv.Close()

	c := NewClient("k", WithBaseURL(srv.URL))
	_, err := c.Generate(context.Background(), &ai.Request{
		Model: "m", UserPrompt: "hi", MaxTokens: 100,
		Options: map[string]any{"reasoning_max_tokens": 2000},
	})
	if err != nil {
		t.Fatalf("Generate error = %v", err)
	}
	var got chatRequest
	if err := json.Unmarshal([]byte(body), &got); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if got.Reasoning == nil || got.Reasoning.MaxTokens != 2000 || got.Reasoning.Effort != "" {
		t.Fatalf("reasoning block = %+v, want {max_tokens:2000}", got.Reasoning)
	}
}

// TestOpenRouter_Effort_Plus_MaxTokens_Conflict (AC7): the replaced Effort-wins
// branch no longer silently picks effort; combining them fails loudly with no
// dispatch.
func TestOpenRouter_Effort_Plus_MaxTokens_Conflict(t *testing.T) {
	hit := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hit = true }))
	defer srv.Close()

	c := NewClient("k", WithBaseURL(srv.URL))
	_, err := c.Generate(context.Background(), &ai.Request{
		Model: "m", UserPrompt: "hi", MaxTokens: 100,
		Options: map[string]any{"reasoning_effort": "high", "reasoning_max_tokens": 2000},
	})
	if !errors.Is(err, ai.ErrConflictingReasoningConfig) {
		t.Fatalf("error = %v, want ErrConflictingReasoningConfig", err)
	}
	if hit {
		t.Fatalf("request dispatched for conflicting reasoning config")
	}
}

// TestOpenRouter_reasoningExtras pins the Step/StreamStep reasoning{} splice
// shape for each decision kind (synthesized directly; table ships empty).
func TestOpenRouter_reasoningExtras(t *testing.T) {
	// None => no fragment.
	frags, err := reasoningExtras(ai.ReasoningDecision{Kind: ai.ReasoningNone})
	if err != nil || frags != nil {
		t.Fatalf("none: frags=%v err=%v, want nil/nil", frags, err)
	}
	// Effort => "reasoning":{"effort":"low"}
	frags, err = reasoningExtras(ai.ReasoningDecision{Kind: ai.ReasoningEffortKind, Effort: "low"})
	if err != nil || len(frags) != 1 || !strings.Contains(string(frags[0]), `"reasoning":{"effort":"low"}`) {
		t.Fatalf("effort: frags=%q err=%v", frags, err)
	}
	// Max-tokens => "reasoning":{"max_tokens":2000}
	frags, err = reasoningExtras(ai.ReasoningDecision{Kind: ai.ReasoningMaxTokensKind, MaxTokensReasoning: 2000})
	if err != nil || len(frags) != 1 || !strings.Contains(string(frags[0]), `"reasoning":{"max_tokens":2000}`) {
		t.Fatalf("maxtokens: frags=%q err=%v", frags, err)
	}
}

// TestOpenRouter_ReasoningMaxTokens_Alone_Step_Preserved: the Step path (which
// pre-sprint emitted NO reasoning at all) now splices reasoning.max_tokens when
// reasoning_max_tokens is supplied alone.
func TestOpenRouter_ReasoningMaxTokens_Alone_Step_Preserved(t *testing.T) {
	var body string
	srv := captureBody(t, &body)
	defer srv.Close()

	c := NewClient("k", WithBaseURL(srv.URL))
	_, err := c.Step(context.Background(), &ai.Request{
		Model:    "m",
		Messages: []ai.Message{{Role: "user", Content: "hi"}},
		Options:  map[string]any{"reasoning_max_tokens": 1500},
	})
	if err != nil {
		t.Fatalf("Step error = %v", err)
	}
	if !strings.Contains(body, `"reasoning":{"max_tokens":1500}`) {
		t.Fatalf("Step body missing reasoning.max_tokens: %s", body)
	}
}
