package anthropic

import (
	"encoding/json"
	"testing"
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
