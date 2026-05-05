package ollama

import (
	"context"

	"github.com/sunholo-data/ailang/internal/ai"
)

// Step is the multi-turn / tool-aware completion entry point introduced by
// M-AI-TOOL-LOOP (v0.17.0). Ollama's local /api/chat endpoint does not
// uniformly support tool calling across the model catalogue (some models
// support it via Modelfile templating, most don't), so v1 of this adapter
// fundamentally rejects tool requests with a typed error and otherwise
// delegates to Generate.
//
// M4 (M-AI-TOOL-LOOP) finalizes the rejection path; full Step parity may
// arrive once an Ollama tool-calling model becomes universally usable.
func (c *Client) Step(ctx context.Context, req *ai.Request) (*ai.Response, error) {
	if len(req.Tools) > 0 {
		return nil, ai.NewAIError(ai.CodeToolsNotSupported,
			"ollama: tool calling not supported (set req.Tools=nil to fall through to Generate)", false)
	}
	// Stub: M4 wires the no-tools path to the existing Generate call so a
	// caller can use Step() uniformly across providers.
	return nil, ai.NewAIError(ai.CodeInternal, "ollama: Step not yet implemented (see M-AI-TOOL-LOOP M4)", false)
}
