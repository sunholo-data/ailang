package builtins

import (
	"github.com/sunholo/ailang/internal/effects"
	"github.com/sunholo/ailang/internal/eval"
	"github.com/sunholo/ailang/internal/types"
)

func init() {
	registerAICall()
}

// _ai_call: Call the AI oracle with a string input
func registerAICall() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/ai",
		Name:    "_ai_call",
		NumArgs: 1, // input: string
		Effect:  "AI",
		Type:    makeAICallType,
		Impl:    aiCallImpl,
		Metadata: &BuiltinMetadata{
			Description: "Call the AI oracle with a string input",
			Params: []ParamDoc{
				{Name: "input", Description: "Input string (JSON by convention)"},
			},
			Returns: "String response from AI handler",
			Examples: []Example{
				{Code: `AI.call("{\"action\":\"decide\"}")`, Description: "Call AI with JSON input"},
				{Code: `let resp = AI.call(json.encode(ctx))`, Description: "Encode context as JSON"},
			},
			Since:     "v0.5.1",
			Stability: StabilityStable,
			Tags:      []string{"ai", "oracle", "llm"},
			Category:  "ai",
		},
	})
	if err != nil {
		panic("failed to register _ai_call builtin: " + err.Error())
	}
}

func makeAICallType() types.Type {
	T := types.NewBuilder()
	// (input: string) -> string ! {AI}
	return T.Func(T.String()).Returns(T.String()).Effects("AI")
}

func aiCallImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	// Check AI capability
	if err := ctx.RequireCap("AI"); err != nil {
		return nil, err
	}

	// Call through to the effect operation
	return effects.Call(ctx, "AI", "call", args)
}
