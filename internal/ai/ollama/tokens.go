package ollama

import (
	ollamaapi "github.com/ollama/ollama/api"
	"github.com/sunholo-data/ailang/internal/ai"
)

// tokenTally accumulates the token counts Ollama reports over a streamed
// response.
//
// Both native call sites previously hardcoded InputTokens/OutputTokens/
// TotalTokens to 0, with a comment claiming "Ollama doesn't report tokens the
// same way". That was stale: ollamaapi.ChatResponse embeds Metrics, which
// carries PromptEvalCount and EvalCount (api/types.go:557-564), and the
// OpenAI-compatible /v1 path returns a standard `usage` block.
//
// The zeros were invisible for as long as every ollama row in models.yml was
// priced 0/0 — cost came out 0 whether the price or the token count was the
// zero, so neither masked the other. The moment a row carries a real imputed
// price (Ollama Cloud, M-OLLAMA-CLOUD-PROVIDER D1) the gap shows up as
// cost_usd = 0 on a run that genuinely spent tokens. It also left
// budgets.max_tokens_per_bench — the WORK gate that exists so a slow box is
// not charged as model failure — with nothing to count on this path.
//
// Ollama emits the counts only on the terminal chunk, so observe() keeps the
// last non-zero value rather than summing: a mid-stream chunk carries no
// metrics, and summing would double-count if a future version repeated them.
type tokenTally struct {
	in  int
	out int
}

// observe records the token counts from one streamed chunk. Zero-valued
// metrics (every chunk before the last) are ignored.
func (t *tokenTally) observe(m ollamaapi.Metrics) {
	if m.PromptEvalCount > 0 {
		t.in = m.PromptEvalCount
	}
	if m.EvalCount > 0 {
		t.out = m.EvalCount
	}
}

// apply writes the tallied counts onto a response. TotalTokens is derived
// rather than read, since Ollama reports the two counts separately and has no
// total field.
func (t tokenTally) apply(r *ai.Response) {
	r.InputTokens = t.in
	r.OutputTokens = t.out
	r.TotalTokens = t.in + t.out
}
