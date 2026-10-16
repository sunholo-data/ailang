package builtins

import (
	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/types"
)

func init() {
	if err := RegisterEffectBuiltin(BuiltinSpec{
		Module: "std/ai", Name: "_ai_embed", NumArgs: 1, Effect: "AI",
		Type: func() types.Type {
			T := types.NewBuilder()
			return T.Func(T.String()).Returns(T.App("Result", T.List(T.Float()), aiErrorRecordType(T))).Effects("AI")
		},
		Impl: func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
			if err := ctx.RequireCapWithBudget("AI", ""); err != nil {
				return nil, err
			}
			return effects.Call(ctx, "AI", "embed", args)
		},
		Metadata: &BuiltinMetadata{Description: "Embed text using the configured provider", LongDesc: "Returns typed AI errors. Uses the configured Gemini, OpenAI or Ollama embedding provider and content-hash caching. Explicit --ai-stub uses a stable 32-dimensional SHA256 vector (ailang/stub-sha256-v1) for keyless tests; use a real embedding provider for semantic similarity. Pin the embedding model for each index namespace.", Params: []ParamDoc{{Name: "text", Description: "Text to embed"}}, Returns: "Result[[float], AIError]", Since: "v0.54.14", Stability: StabilityExperimental, Category: "ai", Tags: []string{"ai", "embedding"}},
	}); err != nil {
		panic(err)
	}
}
