package anthropic

import (
	"context"

	"github.com/sunholo-data/ailang/internal/ai"
)

// Step is the multi-turn / tool-aware completion entry point introduced by
// M-AI-TOOL-LOOP (v0.17.0). Anthropic's Messages API supports tool use via
// `tool_use` content blocks; a real implementation will translate
// req.Messages + req.Tools to Anthropic's messages + tools shape and parse
// tool_use content blocks into resp.ToolCalls.
//
// This stub returns an *ai.AIError so the Provider interface contract holds.
// M2 (M-AI-TOOL-LOOP) replaces the body with the real translation.
func (c *Client) Step(ctx context.Context, req *ai.Request) (*ai.Response, error) {
	return nil, ai.NewAIError(ai.CodeInternal, "anthropic: Step not yet implemented (see M-AI-TOOL-LOOP M2)", false)
}
