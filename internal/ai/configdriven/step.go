package configdriven

import (
	"context"

	"github.com/sunholo-data/ailang/internal/ai"
)

// Step is the multi-turn / tool-aware completion entry point introduced by
// M-AI-TOOL-LOOP (v0.17.0). Config-driven providers describe their HTTP
// shape via [[ai_provider]] TOML blocks (M-AI-PROVIDER-CONFIG, v0.15.0);
// adding tool-loop support here means extending the spec schema with a
// "supports_tools" bit and an optional translation block. That is out of
// scope for the initial M-AI-TOOL-LOOP sprint — config-driven providers
// stay stub'd until a real consumer needs them.
//
// Until then, any caller invoking Step on a config-driven provider gets a
// typed error pointing at the design doc. Callers can fall back to Generate
// for non-tool single-shot calls.
func (p *Provider) Step(ctx context.Context, req *ai.Request) (*ai.Response, error) {
	if len(req.Tools) > 0 {
		return nil, ai.NewAIError(ai.CodeToolsNotSupported,
			"configdriven: tool calling not supported in v1 of [[ai_provider]] schema (see M-AI-TOOL-LOOP design doc)", false)
	}
	return nil, ai.NewAIError(ai.CodeInternal,
		"configdriven: Step not yet implemented (see M-AI-TOOL-LOOP design doc)", false)
}
