package builtins

import (
	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/types"
)

func init() {
	registerAICall()
	registerAICallJson()
	registerAICallJsonSimple()
	registerAICallImage()
	registerAICallImageBase64()
	registerAIStreamCall()
	registerAICallStream()
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
  - Anthropic: claude-sonnet-4-6, claude-haiku-4-5, etc.
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
	if err := ctx.RequireCapWithBudget("AI", ""); err != nil {
		return nil, err
	}

	// Call through to the effect operation
	return effects.Call(ctx, "AI", "call", args)
}

// _ai_call_json: Call the AI oracle requesting structured JSON output with schema
func registerAICallJson() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/ai",
		Name:    "_ai_call_json",
		NumArgs: 2, // input: string, schema: string
		Effect:  "AI",
		Type:    makeAICallJsonType,
		Impl:    aiCallJsonImpl,
		Metadata: &BuiltinMetadata{
			Description: "Call the AI oracle requesting structured JSON output with schema enforcement",
			LongDesc: `Sends a request to the AI provider with structured output configuration.
The provider enforces the given JSON Schema on its response.
Returns raw JSON string — the stdlib wrapper parses it to Json ADT.

Each provider implements this differently:
- Gemini: responseMimeType + responseSchema in generationConfig
- OpenAI: response_format with json_schema
- Anthropic: tool_use pattern with schema as input_schema
- Ollama: format field with schema (no enforcement guarantee)`,
			Params: []ParamDoc{
				{Name: "input", Description: "Prompt string"},
				{Name: "schema", Description: "JSON Schema string for response validation"},
			},
			Returns:   "Raw JSON string matching the schema",
			SeeAlso:   []string{"std/ai.callJson", "std/json.decode", "_ai_call_json_simple"},
			Since:     "v0.7.3",
			Stability: StabilityStable,
			Tags:      []string{"ai", "json", "structured-output"},
			Category:  "ai",
		},
	})
	if err != nil {
		panic("failed to register _ai_call_json builtin: " + err.Error())
	}
}

func makeAICallJsonType() types.Type {
	T := types.NewBuilder()
	// (input: string, schema: string) -> string ! {AI}
	return T.Func(T.String(), T.String()).Returns(T.String()).Effects("AI")
}

func aiCallJsonImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	if err := ctx.RequireCapWithBudget("AI", ""); err != nil {
		return nil, err
	}
	return effects.Call(ctx, "AI", "callJson", args)
}

// _ai_call_json_simple: Call the AI oracle requesting JSON output without schema
func registerAICallJsonSimple() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/ai",
		Name:    "_ai_call_json_simple",
		NumArgs: 1, // input: string
		Effect:  "AI",
		Type:    makeAICallJsonSimpleType,
		Impl:    aiCallJsonSimpleImpl,
		Metadata: &BuiltinMetadata{
			Description: "Call the AI oracle requesting valid JSON output without schema enforcement",
			LongDesc: `Sends a request to the AI provider configured for JSON output mode.
No schema is enforced — the provider guarantees valid JSON but not a specific shape.
Returns raw JSON string — the stdlib wrapper parses it to Json ADT.`,
			Params: []ParamDoc{
				{Name: "input", Description: "Prompt string"},
			},
			Returns:   "Raw JSON string (valid JSON, no schema enforcement)",
			SeeAlso:   []string{"std/ai.callJsonSimple", "std/json.decode", "_ai_call_json"},
			Since:     "v0.7.3",
			Stability: StabilityStable,
			Tags:      []string{"ai", "json", "structured-output"},
			Category:  "ai",
		},
	})
	if err != nil {
		panic("failed to register _ai_call_json_simple builtin: " + err.Error())
	}
}

func makeAICallJsonSimpleType() types.Type {
	T := types.NewBuilder()
	// (input: string) -> string ! {AI}
	return T.Func(T.String()).Returns(T.String()).Effects("AI")
}

func aiCallJsonSimpleImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	if err := ctx.RequireCapWithBudget("AI", ""); err != nil {
		return nil, err
	}
	return effects.Call(ctx, "AI", "callJsonSimple", args)
}

// _ai_call_image: Generate an image and save to file
func registerAICallImage() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/ai",
		Name:    "_ai_call_image",
		NumArgs: 3, // prompt, output_path, options
		Effect:  "AI",
		Type:    makeAICallImageType,
		Impl:    aiCallImageImpl,
		Metadata: &BuiltinMetadata{
			Description: "Generate an image via AI and save to file",
			LongDesc: `Calls an AI image generation model (e.g., Gemini gemini-2.5-flash-image)
and writes the resulting image to the specified output path.

Options is a JSON string with optional fields:
  - aspect_ratio: "1:1", "16:9", "9:16", etc.
  - mime_type: "image/png" (default), "image/jpeg"

Returns the output path on success. Requires both AI and FS capabilities.`,
			Params: []ParamDoc{
				{Name: "prompt", Description: "Image generation prompt"},
				{Name: "output_path", Description: "File path to write the image"},
				{Name: "options", Description: "JSON options string (aspect_ratio, mime_type)"},
			},
			Returns: "The output file path",
			Examples: []Example{
				{Code: `AI.callImage("A sunset over mountains", "output.png", "{}")`, Description: "Generate image with defaults"},
				{Code: `AI.callImage("Banner", "banner.png", "{\"aspect_ratio\": \"16:9\"}")`, Description: "Generate with aspect ratio"},
			},
			SeeAlso:   []string{"std/ai.callImageBase64", "std/ai.call"},
			Since:     "v0.10.0",
			Stability: StabilityStable,
			Tags:      []string{"ai", "image", "generation", "gemini"},
			Category:  "ai",
		},
	})
	if err != nil {
		panic("failed to register _ai_call_image builtin: " + err.Error())
	}
}

func makeAICallImageType() types.Type {
	T := types.NewBuilder()
	// (prompt: string, output_path: string, options: string) -> string ! {AI}
	return T.Func(T.String(), T.String(), T.String()).Returns(T.String()).Effects("AI")
}

func aiCallImageImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	if err := ctx.RequireCapWithBudget("AI", ""); err != nil {
		return nil, err
	}
	return effects.Call(ctx, "AI", "callImage", args)
}

// _ai_call_image_base64: Generate an image and return as base64 JSON
func registerAICallImageBase64() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/ai",
		Name:    "_ai_call_image_base64",
		NumArgs: 2, // prompt, options
		Effect:  "AI",
		Type:    makeAICallImageBase64Type,
		Impl:    aiCallImageBase64Impl,
		Metadata: &BuiltinMetadata{
			Description: "Generate an image via AI and return as base64 JSON",
			LongDesc: `Calls an AI image generation model and returns the image as a JSON string:
{"base64": "<base64-encoded-data>", "mime_type": "image/png"}

Use std/bytes.fromBase64 to decode the base64 data to bytes if needed.
Only requires AI capability (no file system access).`,
			Params: []ParamDoc{
				{Name: "prompt", Description: "Image generation prompt"},
				{Name: "options", Description: "JSON options string (aspect_ratio, mime_type)"},
			},
			Returns: "JSON string with base64 and mime_type fields",
			Examples: []Example{
				{Code: `AI.callImageBase64("A logo", "{}")`, Description: "Generate image as base64"},
			},
			SeeAlso:   []string{"std/ai.callImage", "std/bytes.fromBase64"},
			Since:     "v0.10.0",
			Stability: StabilityStable,
			Tags:      []string{"ai", "image", "generation", "base64"},
			Category:  "ai",
		},
	})
	if err != nil {
		panic("failed to register _ai_call_image_base64 builtin: " + err.Error())
	}
}

func makeAICallImageBase64Type() types.Type {
	T := types.NewBuilder()
	// (prompt: string, options: string) -> string ! {AI}
	return T.Func(T.String(), T.String()).Returns(T.String()).Effects("AI")
}

func aiCallImageBase64Impl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	if err := ctx.RequireCapWithBudget("AI", ""); err != nil {
		return nil, err
	}
	return effects.Call(ctx, "AI", "callImageBase64", args)
}

// _ai_stream_call: Open an SSE token stream from a config-driven AI provider
// (M-AI-STREAMING-HELPER, v0.15.0).
//
// The AI.streamCall effect op is registered in cmd/ailang/configdriven_streaming.go
// (not in internal/effects/) to break the import cycle between
// internal/ai/configdriven and internal/telemetry/internal/effects.
//
// Routes through the M-AI-PROVIDER-CONFIG registry (D11): caller passes a
// provider name + model + serialised messages, never URL or API key. Returns
// Result[StreamConn, StreamError] so the AILANG side handles connection
// failures via the existing pattern.
func registerAIStreamCall() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/ai",
		Name:    "_ai_stream_call",
		NumArgs: 3, // provider: string, model: string, messages_json: string
		Effect:  "AI",
		Type:    makeAIStreamCallType,
		Impl:    aiStreamCallImpl,
		Metadata: &BuiltinMetadata{
			Description: "Open an SSE token stream from a config-driven AI provider",
			LongDesc: `Streams token deltas from an AI provider declared via [[ai_provider]]
in ailang.toml. The provider must declare [ai_provider.streaming] enabled = true
and capabilities.streaming = true.

Caller passes:
  - provider: registered provider name (e.g. "vllm", "openrouter-via-config")
  - model:    model identifier within that provider (e.g. "llama-3.1-70b")
  - messages_json: JSON array of {role, content} records

Returns Result[StreamConn, StreamError]. Use std/stream.onEvent +
runEventLoop to consume the SSE event stream; use std/ai/streaming.parseDelta
to extract TokenDelta records from each event.

EFFECT SIGNATURE: ! {AI, Stream, Net} — AI cap gates spend, Stream provides
the event-loop machinery, Net underlies the HTTP POST. Built-in providers
(openai, anthropic, gemini, ollama, openrouter) are NOT routable here in v1
— they have their own streaming code paths in future milestones.

INTEGRATION: budget tracking, AI cap, and trace span emission all flow
through the standard AI effect machinery — same shape as non-streaming
_ai_call.`,
			Params: []ParamDoc{
				{Name: "provider", Description: "Registered config-driven provider name"},
				{Name: "model", Description: "Model identifier within the provider"},
				{Name: "messages_json", Description: "JSON array of {role, content} records"},
			},
			Returns: "Result[StreamConn, StreamError]",
			SeeAlso: []string{
				"std/ai/streaming.openaiCompatStream",
				"std/ai/streaming.anthropicStream",
				"std/stream.onEvent",
				"_stream_sse_post",
			},
			Since:     "v0.15.0",
			Stability: StabilityExperimental,
			Tags:      []string{"ai", "streaming", "sse", "config-driven"},
			Category:  "ai",
		},
	})
	if err != nil {
		panic("failed to register _ai_stream_call builtin: " + err.Error())
	}
}

func makeAIStreamCallType() types.Type {
	T := types.NewBuilder()
	// (provider: string, model: string, messages_json: string) -> Result[StreamConn, StreamErrorKind] ! {AI, Stream, Net}
	// Matches the existing _stream_sse_post return shape so the AILANG-side
	// std/ai/streaming wrapper can pattern-match on Ok/Err uniformly.
	return T.Func(
		T.String(), // provider
		T.String(), // model
		T.String(), // messages_json
	).Returns(
		T.App("Result", T.Con("StreamConn"), T.Con("StreamErrorKind")),
	).Effects("AI", "Stream", "Net")
}

func aiStreamCallImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	if err := ctx.RequireCapWithBudget("AI", ""); err != nil {
		return nil, err
	}
	if err := ctx.RequireCapWithBudget("Stream", "stream.sse_post"); err != nil {
		return nil, err
	}
	return effects.Call(ctx, "AI", "streamCall", args)
}

// _ai_call_stream: Synchronous accumulator wrapper around _ai_stream_call
// (M-AI-CALL-STREAM-HELPER, v0.15.1).
//
// Opens an SSE token stream from a config-driven AI provider, drives the
// event loop in Go, accumulates content deltas, and returns a single
// Result[string, AIError]. Eliminates the user-side event-loop +
// accumulator boilerplate that consumers like motoko_agent had to build
// by hand against the lower-level _ai_stream_call primitive.
//
// The AI.callStream effect op is registered in cmd/ailang/configdriven_streaming.go
// alongside streamCall, for the same import-cycle reason.
func registerAICallStream() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/ai",
		Name:    "_ai_call_stream",
		NumArgs: 3, // provider: string, model: string, messages_json: string
		Effect:  "AI",
		Type:    makeAICallStreamType,
		Impl:    aiCallStreamImpl,
		Metadata: &BuiltinMetadata{
			Description: "Stream tokens from a config-driven AI provider and return the accumulated string",
			LongDesc: `Synchronous accumulator over _ai_stream_call: opens the SSE
connection, drains events in Go, accumulates content deltas via the provider's
[ai_provider.streaming] delta_path, and returns Result[string, AIError].

Use this when you want the streaming-friendly transport (token-by-token
network framing, configurable timeouts, structured failure) without writing
the event loop yourself. Functionally equivalent to a non-streaming _ai_call,
but flows through the streaming code path so providers that ONLY support
streaming (e.g. some OpenRouter-routed deployments) work as drop-in.

ERROR MAPPING:
  StreamErrorKind variants are flattened into AIError records:
  - ConnectionFailed (retryable=true): network failure opening the stream
  - AuthFailed (retryable=false):       4xx auth errors from upstream
  - Timeout (retryable=true):           idle timeout drained the connection
  - BudgetExhausted (retryable=false):  Stream/AI cap budget overflow
  - ProviderNotFound (retryable=false): provider name not registered
  - CapabilityNotSupported (retryable=false): streaming.enabled = false
  - ProtocolError (retryable=false):    malformed upstream response

REASONING FIELDS:
  In v0.15.1, reasoning_content / thinking deltas are READ from upstream
  events but NOT included in the accumulated string. Only visible content
  appears. v0.15.2's callStreamWithReasoning will surface both.

TRACE SPANS:
  Exactly ONE AI/streamCall span is emitted per _ai_call_stream invocation
  — the underlying streamCall op records the span; this accumulator does NOT
  emit a separate span.`,
			Params: []ParamDoc{
				{Name: "provider", Description: "Registered config-driven provider name"},
				{Name: "model", Description: "Model identifier within the provider"},
				{Name: "messages_json", Description: "JSON array of {role, content} records"},
			},
			Returns: "Result[string, AIError]",
			SeeAlso: []string{
				"_ai_stream_call",
				"std/ai/streaming.callStream",
				"std/ai/streaming.openaiCompatStream",
			},
			Since:     "v0.15.1",
			Stability: StabilityExperimental,
			Tags:      []string{"ai", "streaming", "sse", "config-driven", "accumulator"},
			Category:  "ai",
		},
	})
	if err != nil {
		panic("failed to register _ai_call_stream builtin: " + err.Error())
	}
}

func makeAICallStreamType() types.Type {
	T := types.NewBuilder()
	// (provider: string, model: string, messages_json: string)
	//   -> Result[string, AIError] ! {AI, Stream, Net}
	// AIError is a record { code: string, message: string, retryable: bool }.
	aiErrorType := T.Record(
		types.Field("code", T.String()),
		types.Field("message", T.String()),
		types.Field("retryable", T.Bool()),
	)
	return T.Func(
		T.String(), // provider
		T.String(), // model
		T.String(), // messages_json
	).Returns(
		T.App("Result", T.String(), aiErrorType),
	).Effects("AI", "Stream", "Net")
}

func aiCallStreamImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	if err := ctx.RequireCapWithBudget("AI", ""); err != nil {
		return nil, err
	}
	if err := ctx.RequireCapWithBudget("Stream", "stream.sse_post"); err != nil {
		return nil, err
	}
	return effects.Call(ctx, "AI", "callStream", args)
}
