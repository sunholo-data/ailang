package openrouter

import (
	"context"

	"github.com/sunholo-data/ailang/internal/ai"
)

// Step is the multi-turn / tool-aware completion entry point introduced by
// M-AI-TOOL-LOOP (v0.17.0). OpenRouter speaks OpenAI Chat Completions
// natively, so the real implementation will pass tools / messages straight
// through to its existing Chat endpoint and verify it composes with the
// Routing field added in M-AI-OPENROUTER (v0.16.x) — i.e. a routed-model
// tool call works end-to-end against e.g. anthropic/claude-sonnet-4.5
// behind the OpenRouter URL.
//
// This stub returns an *ai.AIError so the Provider interface contract holds.
// M4 (M-AI-TOOL-LOOP) replaces the body with the real translation.
func (c *Client) Step(ctx context.Context, req *ai.Request) (*ai.Response, error) {
	return nil, ai.NewAIError(ai.CodeInternal, "openrouter: Step not yet implemented (see M-AI-TOOL-LOOP M4)", false)
}
