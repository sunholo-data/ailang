package main

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/eval_harness"
)

// A harness passes --ai a wire name ("openrouter/deepseek/deepseek-v4-flash-0731")
// rather than a models.yml key. The direct path must still apply that model's
// declared output budget: at the handler default of 4096 a reasoning model spent
// the whole budget thinking and returned finish_reason=length with no text and
// no tool call (motoko A/B, 2026-09-27).
func TestDeclaredMaxOutputTokens_WireNameResolves(t *testing.T) {
	if err := eval_harness.InitModelsConfig(); err != nil {
		t.Fatalf("load models config: %v", err)
	}
	if got := declaredMaxOutputTokens("openrouter/deepseek/deepseek-v4-flash-0731"); got <= 4096 {
		t.Fatalf("declared max output tokens for motoko's wire name = %d, want the models.yml value (> 4096)", got)
	}
	if got := declaredMaxOutputTokens("nobody/unknown-model-xyz"); got != 0 {
		t.Fatalf("unknown model = %d, want 0 (handler default)", got)
	}
}
