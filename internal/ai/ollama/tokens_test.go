package ollama

import (
	"testing"

	ollamaapi "github.com/ollama/ollama/api"
	"github.com/sunholo-data/ailang/internal/ai"
)

// TestTokenTallyStreamedResponse pins the shape Ollama actually streams:
// metric-free chunks followed by one terminal chunk carrying the counts.
// Before M-OLLAMA-CLOUD AC9 both native call sites hardcoded 0 here, which made
// cost_usd 0 for any row with real pricing and left the
// budgets.max_tokens_per_bench WORK gate with nothing to count.
func TestTokenTallyStreamedResponse(t *testing.T) {
	var tally tokenTally

	// Content chunks carry no metrics — the zero Metrics must not clobber.
	tally.observe(ollamaapi.Metrics{})
	tally.observe(ollamaapi.Metrics{})
	// Terminal chunk carries the counts.
	tally.observe(ollamaapi.Metrics{PromptEvalCount: 282048, EvalCount: 1971})

	var resp ai.Response
	tally.apply(&resp)

	if resp.InputTokens != 282048 {
		t.Errorf("InputTokens = %d, want 282048", resp.InputTokens)
	}
	if resp.OutputTokens != 1971 {
		t.Errorf("OutputTokens = %d, want 1971", resp.OutputTokens)
	}
	// TotalTokens is derived: Ollama has no total field.
	if resp.TotalTokens != 284019 {
		t.Errorf("TotalTokens = %d, want 284019 (in+out)", resp.TotalTokens)
	}
}

// TestTokenTallyIgnoresZeroChunks is the regression guard for the reason
// observe() keeps the last non-zero value instead of summing: a repeated or
// trailing metric-free chunk must not zero out or double a settled count.
func TestTokenTallyIgnoresZeroChunks(t *testing.T) {
	var tally tokenTally
	tally.observe(ollamaapi.Metrics{PromptEvalCount: 100, EvalCount: 20})
	tally.observe(ollamaapi.Metrics{}) // trailing empty chunk

	var resp ai.Response
	tally.apply(&resp)

	if resp.InputTokens != 100 || resp.OutputTokens != 20 || resp.TotalTokens != 120 {
		t.Errorf("got in=%d out=%d total=%d, want 100/20/120",
			resp.InputTokens, resp.OutputTokens, resp.TotalTokens)
	}
}

// TestTokenTallyNoMetricsStaysZero: a provider or version that genuinely
// reports nothing must still yield 0, not a fabricated count. Zero here means
// "not reported" and is the honest answer — the bug was hardcoding it when the
// counts WERE available, not the zero itself.
func TestTokenTallyNoMetricsStaysZero(t *testing.T) {
	var tally tokenTally
	tally.observe(ollamaapi.Metrics{})

	var resp ai.Response
	tally.apply(&resp)

	if resp.InputTokens != 0 || resp.OutputTokens != 0 || resp.TotalTokens != 0 {
		t.Errorf("got in=%d out=%d total=%d, want all 0",
			resp.InputTokens, resp.OutputTokens, resp.TotalTokens)
	}
}
