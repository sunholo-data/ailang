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
		Metadata: &BuiltinMetadata{Description: "Embed text using the configured provider", Params: []ParamDoc{{Name: "text", Description: "Text to embed"}}, Returns: "Result[[float], AIError]", Since: "v0.54.14", Stability: StabilityExperimental, Category: "ai", Tags: []string{"ai", "embedding"}},
	}); err != nil {
		panic(err)
	}
}
