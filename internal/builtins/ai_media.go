package builtins

// _ai_call_media_result: single-shot typed media call returning
// Result[MediaCallResult, AIError] (M-STD-AI-GEMINI-FILE-SLICING, #2458).
// Thin pass-through to the AI.callMediaResult effect op
// (internal/effects/ai_media.go).

import (
	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/types"
)

func init() {
	registerAICallMediaResult()
}

// mediaCallResultRecordType returns the AILANG MediaCallResult record shape:
// { text: string, input_tokens: int, output_tokens: int }.
func mediaCallResultRecordType(T *types.Builder) types.Type {
	return T.Record(
		types.Field("text", T.String()),
		types.Field("input_tokens", T.Int()),
		types.Field("output_tokens", T.Int()),
	)
}

func registerAICallMediaResult() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/ai",
		Name:    "_ai_call_media_result",
		NumArgs: 3, // prompt, media, options
		Effect:  "AI",
		Type:    makeAICallMediaResultType,
		Impl:    aiCallMediaResultImpl,
		Metadata: &BuiltinMetadata{
			Description: "Single-shot AI call with typed media attachments returning Result[MediaCallResult, AIError]",
			LongDesc: `Calls the configured --ai provider with a text prompt and an ordered list of
typed MediaPart attachments (Image or File, with optional per-part resolution
and server-side slicing options), returning both the response text and token
usage ({ text, input_tokens, output_tokens }).

options is a JSON object with optional call-level keys (currently: "model" for
a per-call model override); "" or "{}" means defaults. Slicing controls live
per-part on File.options and are validated by MIME class at the effect boundary.
Unknown keys or invalid ranges return Err(AIError{code: "SchemaValidation"}).
Unsupported providers/endpoints return Err(AIError{code: "CapabilityNotSupported"}).`,
			Params: []ParamDoc{
				{Name: "prompt", Description: "Prompt text"},
				{Name: "media", Description: "Ordered list of MediaPart attachments (Image | File)"},
				{Name: "options", Description: "Call-level JSON options (e.g. {\"model\": \"...\"}), or \"\""},
			},
			Returns:   "Result[MediaCallResult, AIError]",
			SeeAlso:   []string{"std/ai.callMediaResult", "std/ai.step", "_ai_call_result"},
			Since:     "v0.60.7",
			Stability: StabilityExperimental,
			Tags:      []string{"ai", "result", "typed-errors", "media", "slicing"},
			Category:  "ai",
		},
	})
	if err != nil {
		panic("failed to register _ai_call_media_result builtin: " + err.Error())
	}
}

func makeAICallMediaResultType() types.Type {
	T := types.NewBuilder()
	// (prompt: string, media: [MediaPart], options: string) -> Result[MediaCallResult, AIError] ! {AI}
	return T.Func(T.String(), T.List(mediaPartType(T)), T.String()).
		Returns(T.App("Result", mediaCallResultRecordType(T), aiErrorRecordType(T))).
		Effects("AI")
}

func aiCallMediaResultImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	if err := ctx.RequireCapWithBudget("AI", ""); err != nil {
		return nil, err
	}
	return effects.Call(ctx, "AI", "callMediaResult", args)
}
