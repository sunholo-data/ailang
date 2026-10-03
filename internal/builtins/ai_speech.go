package builtins

// _ai_call_speech: text-to-speech for std/ai.callSpeech (#1495). A thin
// pass-through to the AI.callSpeech effect op (internal/effects/ai_speech.go),
// which builds the Result[bytes, AIError]. Registered through
// RegisterEffectBuiltin like every std/ai builtin, so the evaluator and the
// bytecode VM resolve it from the same registry.

import (
	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/types"
)

func init() {
	registerAICallSpeech()
}

func registerAICallSpeech() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/ai",
		Name:    "_ai_call_speech",
		NumArgs: 4, // text, voice, style, options
		Effect:  "AI",
		Type:    makeAICallSpeechType,
		Impl:    aiCallSpeechImpl,
		Metadata: &BuiltinMetadata{
			Description: "Text-to-speech returning Result[bytes, AIError]: s16le mono PCM at 24 kHz",
			LongDesc: `Synthesises speech through the --ai provider. The bytes are raw PCM:
signed 16-bit little-endian, mono, 24000 Hz — the format std/audio.wavFromPcm
and std/audio.encode take (wavFromPcm(pcm, 24000, 1, 16)).

Providers: Gemini (AI Studio or Vertex) via its TTS models. The model is
options.model, else the bound --ai model when its name contains "tts", else
gemini-2.5-flash-preview-tts. Every other provider returns
Err(AIError{code: "CapabilityNotSupported"}).

voice is a provider voice name (Gemini prebuilt voices, e.g. "Kore", "Puck");
"" uses the provider default. style is a natural-language delivery
instruction ("Say cheerfully"); Gemini receives it as "<style>: <text>".
options is JSON with optional keys model and language_code; any other key
is Err(SchemaValidation).

--ai-stub returns a fixed 250 ms 400 Hz tone; --ai-stub-fixtures replays
"speech" fixtures keyed by {style, text, voice}.`,
			Params: []ParamDoc{
				{Name: "text", Description: "What to say"},
				{Name: "voice", Description: "Provider voice name, or \"\" for the default"},
				{Name: "style", Description: "Delivery instruction, or \"\""},
				{Name: "options", Description: "JSON: {\"model\": ..., \"language_code\": ...}, or \"\""},
			},
			Returns:   "Result[bytes, AIError]",
			SeeAlso:   []string{"std/ai.callSpeech", "std/audio.wavFromPcm", "std/audio.encode"},
			Since:     "v0.52.0",
			Stability: StabilityExperimental,
			Tags:      []string{"ai", "result", "typed-errors", "speech", "tts", "audio"},
			Category:  "ai",
		},
	})
	if err != nil {
		panic("failed to register _ai_call_speech builtin: " + err.Error())
	}
}

func makeAICallSpeechType() types.Type {
	T := types.NewBuilder()
	// (text, voice, style, options: string) -> Result[bytes, AIError] ! {AI}
	return T.Func(T.String(), T.String(), T.String(), T.String()).
		Returns(T.App("Result", T.Bytes(), aiErrorRecordType(T))).
		Effects("AI")
}

func aiCallSpeechImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	if err := ctx.RequireCapWithBudget("AI", ""); err != nil {
		return nil, err
	}
	return effects.Call(ctx, "AI", "callSpeech", args)
}
