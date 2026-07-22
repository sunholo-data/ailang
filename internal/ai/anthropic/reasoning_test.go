package anthropic

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"errors"

	"github.com/sunholo-data/ailang/internal/ai"
)

func captureBody(t *testing.T, captured *string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		*captured = string(b)
		resp := messagesResponse{
			ID: "m", Type: "message", Role: "assistant", Model: "claude",
			Content:    []contentBlock{{Type: "text", Text: "ok"}},
			StopReason: "end_turn",
			Usage:      anthropicUsage{InputTokens: 1, OutputTokens: 1},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
}

// TestAnthropic_Golden_NoReasoning_ByteIdentical (AC14): unset request body
// carries NO thinking key (byte-identical to pre-v0.31.0; omitempty pointer).
func TestAnthropic_Golden_NoReasoning_ByteIdentical(t *testing.T) {
	var body string
	srv := captureBody(t, &body)
	defer srv.Close()

	c := NewClient("k", WithBaseURL(srv.URL))
	_, err := c.Generate(context.Background(), &ai.Request{
		Model: "claude-sonnet-4-5", SystemPrompt: "sys", UserPrompt: "hi", MaxTokens: 2048,
	})
	if err != nil {
		t.Fatalf("Generate error = %v", err)
	}
	if strings.Contains(body, "thinking") {
		t.Fatalf("unset request body leaked thinking block: %s", body)
	}
}

// TestAnthropic_thinkingBlockFor pins the thinking block shape: enabled thinking
// (budget>=1024) emits {type:enabled,budget_tokens:N}; "off"/B==0 and None omit
// the block (exact disablement is expressed by omission).
func TestAnthropic_thinkingBlockFor(t *testing.T) {
	tests := []struct {
		name       string
		dec        ai.ReasoningDecision
		wantBlock  bool
		wantBudget int
	}{
		{"none", ai.ReasoningDecision{Kind: ai.ReasoningNone}, false, 0},
		{"off_zero", ai.ReasoningDecision{Kind: ai.ReasoningEffortKind, Effort: "off", Budget: 0, BudgetSet: true}, false, 0},
		{"high", ai.ReasoningDecision{Kind: ai.ReasoningEffortKind, Effort: "high", Budget: 16384, BudgetSet: true}, true, 16384},
		{"budget_1024", ai.ReasoningDecision{Kind: ai.ReasoningBudgetKind, Budget: 1024, BudgetSet: true}, true, 1024},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tb := thinkingBlockFor(tt.dec)
			if !tt.wantBlock {
				if tb != nil {
					t.Fatalf("expected no thinking block, got %+v", tb)
				}
				return
			}
			if tb == nil || tb.Type != "enabled" || tb.BudgetTokens != tt.wantBudget {
				t.Fatalf("thinking block = %+v, want enabled/%d", tb, tt.wantBudget)
			}
		})
	}
}

// TestAnthropic_StrictBudget_BeforeDefaulting (AC11): an enabled thinking budget
// with MaxTokens unset (the client would otherwise silently substitute 4096) is
// rejected BEFORE that defaulting, and no request is dispatched.
func TestAnthropic_StrictBudget_BeforeDefaulting(t *testing.T) {
	hit := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hit = true }))
	defer srv.Close()

	c := NewClient("k", WithBaseURL(srv.URL))
	// budget 4096 with MaxTokens unset: the 4096 default must NOT rescue it.
	_, err := c.Generate(context.Background(), &ai.Request{
		Model: "claude", UserPrompt: "hi",
		Options: map[string]any{"thinking_budget_tokens": 4096},
	})
	if !errors.Is(err, ai.ErrReasoningBudgetExceedsMaxTokens) {
		t.Fatalf("error = %v, want ErrReasoningBudgetExceedsMaxTokens", err)
	}
	if hit {
		t.Fatalf("request dispatched; validation must precede the MaxTokens=4096 defaulting")
	}

	// budget == MaxTokens is also rejected (strict >).
	_, err = c.Generate(context.Background(), &ai.Request{
		Model: "claude", UserPrompt: "hi", MaxTokens: 4096,
		Options: map[string]any{"thinking_budget_tokens": 4096},
	})
	if !errors.Is(err, ai.ErrReasoningBudgetExceedsMaxTokens) {
		t.Fatalf("budget==maxtokens: error = %v, want ErrReasoningBudgetExceedsMaxTokens", err)
	}
}
