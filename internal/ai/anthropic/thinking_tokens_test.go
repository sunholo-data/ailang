package anthropic

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sunholo-data/ailang/internal/ai"
)

// TestAnthropicUsage_ParsesThinkingTokens pins the wire shape against a real
// response body.
//
// History worth keeping: the mapping was deliberately omitted with the comment
// "Anthropic reports no separate thinking-token count". That was true when
// written and false later — the API gained
// usage.output_tokens_details.thinking_tokens — and because nothing tested the
// claim, every Anthropic row silently banked 0 reasoning tokens. The Fable 5.1
// core+frontier run (2026-09-02) reported 0 for an ALWAYS-ON thinking model
// before this was fixed. This test exists so the next such change fails loudly.
func TestAnthropicUsage_ParsesThinkingTokens(t *testing.T) {
	// Verbatim shape from a live claude-fable-5-1 response, 2026-09-02.
	body := `{
      "input_tokens": 20,
      "cache_creation_input_tokens": 0,
      "cache_read_input_tokens": 0,
      "output_tokens": 1160,
      "output_tokens_details": {"thinking_tokens": 298},
      "service_tier": "standard"
    }`

	var u anthropicUsage
	if err := json.Unmarshal([]byte(body), &u); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got := u.OutputTokensDetails.ThinkingTokens; got != 298 {
		t.Errorf("ThinkingTokens = %d, want 298", got)
	}
	if u.OutputTokens != 1160 {
		t.Errorf("OutputTokens = %d, want 1160", u.OutputTokens)
	}
	// Thinking is INCLUDED in output_tokens, so it must never exceed it —
	// and callers must never sum the two into a total.
	if u.OutputTokensDetails.ThinkingTokens > u.OutputTokens {
		t.Error("thinking tokens must be a subset of output tokens, not additional to them")
	}
}

// A model that does not think, or a thinking model that chose not to, omits the
// object entirely. That must decode as 0, not as an error.
func TestAnthropicUsage_AbsentDetailsIsZero(t *testing.T) {
	var u anthropicUsage
	if err := json.Unmarshal([]byte(`{"input_tokens":5,"output_tokens":9}`), &u); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if u.OutputTokensDetails.ThinkingTokens != 0 {
		t.Errorf("absent output_tokens_details must yield 0, got %d", u.OutputTokensDetails.ThinkingTokens)
	}
}

// TestGenerate_OutputAndReasonAreDisjoint pins the ai.Response contract: the
// wire's output_tokens INCLUDES thinking, but OutputTokens must not — every cost
// caller bills OutputTokens+ReasonTokens (the OpenAI/OpenRouter convention).
// Passing the merged figure through billed each thinking token twice on every
// Anthropic row from 2026-09-02 (when ReasonTokens started populating) until
// 2026-11-09.
func TestGenerate_OutputAndReasonAreDisjoint(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"msg_1","type":"message","role":"assistant",
			"content":[{"type":"thinking","thinking":""},{"type":"text","text":"ok"}],
			"model":"claude-fable-5-2","stop_reason":"end_turn",
			"usage":{"input_tokens":20,"output_tokens":1160,
			         "output_tokens_details":{"thinking_tokens":298}}}`))
	}))
	defer srv.Close()

	c := NewClient("test-key", WithBaseURL(srv.URL))
	resp, err := c.Generate(context.Background(), &ai.Request{Model: "claude-fable-5-2", UserPrompt: "x", MaxTokens: 2048})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if resp.OutputTokens != 862 || resp.ReasonTokens != 298 {
		t.Errorf("OutputTokens=%d ReasonTokens=%d, want 862 + 298 (disjoint)", resp.OutputTokens, resp.ReasonTokens)
	}
	if billed := resp.OutputTokens + resp.ReasonTokens; billed != 1160 {
		t.Errorf("OutputTokens+ReasonTokens = %d, want the wire's 1160 output tokens", billed)
	}
	if resp.TotalTokens != 1180 {
		t.Errorf("TotalTokens = %d, want 1180 (input + wire output)", resp.TotalTokens)
	}
}
