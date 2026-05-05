package gemini

import (
	"context"

	"github.com/sunholo-data/ailang/internal/ai"
)

// Step is the multi-turn / tool-aware completion entry point introduced by
// M-AI-TOOL-LOOP (v0.17.0). Gemini's generateContent API supports function
// calling via `functionDeclarations` in `tools` and emits `functionCall` /
// `functionResponse` parts in `contents[].parts[]`. A real implementation
// will translate req.Messages + req.Tools to that shape and parse
// functionCall parts into resp.ToolCalls. Gemini does NOT assign tool-call
// IDs natively, so the adapter generates deterministic IDs of the form
// "<turn_index>_<call_index>" so the round-trip back to the model is stable.
//
// This stub returns an *ai.AIError so the Provider interface contract holds.
// M3 (M-AI-TOOL-LOOP) replaces the body with the real translation.
func (c *Client) Step(ctx context.Context, req *ai.Request) (*ai.Response, error) {
	return nil, ai.NewAIError(ai.CodeInternal, "gemini: Step not yet implemented (see M-AI-TOOL-LOOP M3)", false)
}
