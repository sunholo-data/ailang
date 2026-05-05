package openai

import (
	"context"

	"github.com/sunholo-data/ailang/internal/ai"
)

// Step is the multi-turn / tool-aware completion entry point introduced by
// M-AI-TOOL-LOOP (v0.17.0). The OpenAI Chat Completions API natively supports
// tool use via the same /chat/completions endpoint, so a real implementation
// will lift the existing Generate request shape and add tools / tool_choice
// + parse tool_calls out of the response.
//
// This stub returns an *ai.AIError so the Provider interface contract holds.
// M4 (M-AI-TOOL-LOOP) replaces the body with the real translation.
func (c *Client) Step(ctx context.Context, req *ai.Request) (*ai.Response, error) {
	return nil, ai.NewAIError(ai.CodeInternal, "openai: Step not yet implemented (see M-AI-TOOL-LOOP M4)", false)
}
