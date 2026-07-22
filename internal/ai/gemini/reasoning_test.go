package gemini

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

// captureBodyServer records the raw request body of the first request.
func captureBodyServer(t *testing.T, captured *string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		*captured = string(b)
		resp := generateResponse{
			Candidates: []candidate{{
				Content:      content{Role: "model", Parts: []part{{Text: "ok"}}},
				FinishReason: "STOP",
			}},
			UsageMetadata: usageMetadata{PromptTokenCount: 1, CandidatesTokenCount: 1, TotalTokenCount: 2},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
}

// TestGemini_Golden_NoReasoning_ByteIdentical (AC14): with ReasoningEffort==""
// and no legacy reasoning option, the wire body carries NO thinkingConfig key,
// i.e. it is byte-identical to the pre-v0.31.0 body (the only additive field is
// the omitempty *thinkingConfig pointer).
func TestGemini_Golden_NoReasoning_ByteIdentical(t *testing.T) {
	var body string
	srv := captureBodyServer(t, &body)
	defer srv.Close()

	c := NewClient("k", WithBaseURL(srv.URL))
	_, err := c.Generate(context.Background(), &ai.Request{
		Model:        "gemini-2.5-flash",
		SystemPrompt: "sys",
		UserPrompt:   "hi",
		MaxTokens:    100,
		Temperature:  0.5,
	})
	if err != nil {
		t.Fatalf("Generate error = %v", err)
	}
	if strings.Contains(body, "thinkingConfig") || strings.Contains(body, "thinkingBudget") {
		t.Fatalf("unset request body leaked reasoning fields: %s", body)
	}
}

// TestGemini_applyReasoning_Budgets pins the thinkingConfig wire shape for each
// decision kind without needing the (empty) capability table: the decision is
// synthesized directly, exactly as the resolver would hand it to buildStepRequest.
func TestGemini_applyReasoning_Budgets(t *testing.T) {
	tests := []struct {
		name       string
		dec        ai.ReasoningDecision
		wantConfig bool
		wantBudget int
	}{
		{"none_omits", ai.ReasoningDecision{Kind: ai.ReasoningNone}, false, 0},
		{"budget_high", ai.ReasoningDecision{Kind: ai.ReasoningEffortKind, Effort: "high", Budget: 16384, BudgetSet: true}, true, 16384},
		{"budget_low", ai.ReasoningDecision{Kind: ai.ReasoningBudgetKind, Budget: 1024, BudgetSet: true}, true, 1024},
		{"off_zero", ai.ReasoningDecision{Kind: ai.ReasoningEffortKind, Effort: "off", Budget: 0, BudgetSet: true}, true, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &generationConfig{}
			applyReasoning(cfg, tt.dec)
			if !tt.wantConfig {
				if cfg.ThinkingConfig != nil {
					t.Fatalf("expected no thinkingConfig, got %+v", cfg.ThinkingConfig)
				}
				return
			}
			if cfg.ThinkingConfig == nil || cfg.ThinkingConfig.ThinkingBudget == nil {
				t.Fatalf("expected thinkingConfig with budget, got nil")
			}
			if *cfg.ThinkingConfig.ThinkingBudget != tt.wantBudget {
				t.Fatalf("budget = %d, want %d", *cfg.ThinkingConfig.ThinkingBudget, tt.wantBudget)
			}
			// Marshal and confirm thinkingBudget:0 survives for the off case
			// (non-omitempty pointer) — exact disablement must reach the wire.
			raw, _ := json.Marshal(cfg)
			if tt.wantBudget == 0 && !strings.Contains(string(raw), `"thinkingBudget":0`) {
				t.Fatalf("off case must emit thinkingBudget:0, got %s", raw)
			}
		})
	}
}

// TestGemini_Reasoning_MaxTokensConflict_FailLoud (AC10): enabled thinking via
// thinking_budget_tokens with MaxTokens unset is rejected before dispatch.
func TestGemini_Reasoning_MaxTokensConflict_FailLoud(t *testing.T) {
	hit := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hit = true
	}))
	defer srv.Close()

	c := NewClient("k", WithBaseURL(srv.URL))
	_, err := c.Generate(context.Background(), &ai.Request{
		Model:      "gemini-2.5-flash",
		UserPrompt: "hi",
		Options:    map[string]any{"thinking_budget_tokens": 1024}, // MaxTokens unset
	})
	if !errors.Is(err, ai.ErrReasoningBudgetExceedsMaxTokens) {
		t.Fatalf("error = %v, want ErrReasoningBudgetExceedsMaxTokens", err)
	}
	if hit {
		t.Fatalf("request dispatched; validation must precede dispatch")
	}
}
