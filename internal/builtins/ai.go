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
			LongDesc: `The AI effect is AILANG's general-purpose AI oracle - an opaque,
host-provided effect for calling external AI/ML systems.

CONFIGURATION:
  The AI handler must be configured via CLI flags when running AILANG:

  ailang run --ai <model> --caps AI --entry main module.ail

  Supported models (configured via models.yml or guessed from name):
  - Anthropic: claude-sonnet-4-5, claude-haiku-4-5, etc.
  - OpenAI:    gpt-5, gpt-5-mini, etc.
  - Google:    gemini-2-5-pro, gemini-2-5-flash, etc.

  Required environment variables (based on provider):
  - ANTHROPIC_API_KEY for Claude models
  - OPENAI_API_KEY for GPT models
  - GOOGLE_API_KEY for Gemini (or leave unset to use Vertex AI ADC)

TESTING:
  Use --ai-stub for deterministic testing (returns {"kind":"Wait"}):

  ailang run --ai-stub --caps AI --entry main module.ail

PROTOCOL:
  Input/output is string → string. By convention, JSON is used but not enforced.
  The host AI handler interprets the input and generates the response.`,
			Params: []ParamDoc{
				{Name: "input", Description: "Input string (JSON by convention)"},
			},
			Returns: "String response from AI handler",
			Examples: []Example{
				{Code: `AI.call("{\"action\":\"decide\"}")`, Description: "Call AI with JSON input"},
				{Code: `let resp = AI.call(json.encode(ctx))`, Description: "Encode context as JSON"},
			},
			SeeAlso:   []string{"std/json.encode", "std/json.decode"},
			Since:     "v0.5.1",
			Stability: StabilityStable,
			Tags:      []string{"ai", "oracle", "llm", "anthropic", "openai", "gemini"},
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
