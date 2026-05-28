package builtins

// AI multi-turn / Result-returning builtins introduced by M-AI-TOOL-LOOP
// (v0.17.0). Wires the AILANG-side surface (std/ai.callResult,
// std/ai.callJsonResult, std/ai.step) to the corresponding effect ops in
// internal/effects/ai_step.go.
//
// All three return AILANG `Result[..., AIError]` — the effect ops do the
// AILANG ADT construction, so these builtins are thin pass-throughs over
// effects.Call.

import (
	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/types"
)

// aiErrorRecordType returns the canonical AIError record shape.
// Matches std/ai/streaming.AIError byte-for-byte.
func aiErrorRecordType(T *types.Builder) types.Type {
	return T.Record(
		types.Field("code", T.String()),
		types.Field("message", T.String()),
		types.Field("retryable", T.Bool()),
	)
}

// toolCallRecordType returns the AILANG ToolCall record shape.
func toolCallRecordType(T *types.Builder) types.Type {
	return T.Record(
		types.Field("id", T.String()),
		types.Field("name", T.String()),
		types.Field("arguments", T.String()),
	)
}

// messageRecordType returns the AILANG Message record shape.
func messageRecordType(T *types.Builder) types.Type {
	return T.Record(
		types.Field("role", T.String()),
		types.Field("content", T.String()),
		types.Field("tool_calls", T.List(toolCallRecordType(T))),
		types.Field("tool_call_id", T.String()),
	)
}

// toolSchemaRecordType returns the AILANG ToolSchema record shape.
func toolSchemaRecordType(T *types.Builder) types.Type {
	return T.Record(
		types.Field("name", T.String()),
		types.Field("description", T.String()),
		types.Field("parameters", T.String()),
	)
}

// cacheBreakpointRecordType returns the AILANG CacheBreakpoint record shape.
// Mirrors ai.CacheBreakpoint byte-for-byte. Introduced by M-AI-PROMPT-CACHING
// (v0.18.4).
func cacheBreakpointRecordType(T *types.Builder) types.Type {
	return T.Record(
		types.Field("position", T.String()),
		types.Field("ttl", T.String()),
	)
}

// stepResultRecordType returns the AILANG StepResult record shape.
//
// Cache-token fields (cache_read_input_tokens, cache_creation_input_tokens)
// surface upstream prompt-cache telemetry: Anthropic populates both;
// OpenAI + Gemini populate cache_read only (no separate cache-write count
// in their APIs); other providers leave both at 0. Useful for eval-harness
// cost telemetry that needs to distinguish "cold cache" vs "warm cache"
// per-step costs.
func stepResultRecordType(T *types.Builder) types.Type {
	return T.Record(
		types.Field("message", messageRecordType(T)),
		types.Field("tool_calls", T.List(toolCallRecordType(T))),
		types.Field("finish_reason", T.String()),
		types.Field("input_tokens", T.Int()),
		types.Field("output_tokens", T.Int()),
		types.Field("cache_read_input_tokens", T.Int()),
		types.Field("cache_creation_input_tokens", T.Int()),
	)
}

// ============================================================================
// _ai_call_result: Result-returning variant of _ai_call
// ============================================================================

func registerAICallResult() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/ai",
		Name:    "_ai_call_result",
		NumArgs: 1, // input: string
		Effect:  "AI",
		Type:    makeAICallResultType,
		Impl:    aiCallResultImpl,
		Metadata: &BuiltinMetadata{
			Description: "Call the AI oracle returning Result[string, AIError] (typed errors)",
			LongDesc: `Result-returning variant of _ai_call. Same wire path as the legacy
single-shot call, but typed errors flow back as Err(AIError record) instead
of crashing the host with a Go error. AIError shape mirrors
std/ai/streaming.AIError exactly: { code, message, retryable }.

Use this in agent loops where retry-vs-surface decisions need structured
error data (retryable bool, code string for telemetry routing) without
parsing string error messages.`,
			Params: []ParamDoc{
				{Name: "input", Description: "Prompt string"},
			},
			Returns: "Result[string, AIError]",
			SeeAlso: []string{
				"_ai_call",
				"_ai_call_json_result",
				"std/ai.callResult",
			},
			Since:     "v0.17.0",
			Stability: StabilityExperimental,
			Tags:      []string{"ai", "result", "typed-errors"},
			Category:  "ai",
		},
	})
	if err != nil {
		panic("failed to register _ai_call_result builtin: " + err.Error())
	}
}

func makeAICallResultType() types.Type {
	T := types.NewBuilder()
	// (input: string) -> Result[string, AIError] ! {AI}
	return T.Func(T.String()).
		Returns(T.App("Result", T.String(), aiErrorRecordType(T))).
		Effects("AI")
}

func aiCallResultImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	if err := ctx.RequireCapWithBudget("AI", ""); err != nil {
		return nil, err
	}
	return effects.Call(ctx, "AI", "callResult", args)
}

// ============================================================================
// _ai_call_json_result: Result-returning variant of _ai_call_json
// ============================================================================

func registerAICallJsonResult() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/ai",
		Name:    "_ai_call_json_result",
		NumArgs: 2, // input: string, schema: string
		Effect:  "AI",
		Type:    makeAICallJsonResultType,
		Impl:    aiCallJsonResultImpl,
		Metadata: &BuiltinMetadata{
			Description: "Call the AI oracle requesting structured JSON output, returning Result[string, AIError]",
			LongDesc: `Result-returning variant of _ai_call_json. Same schema-enforced JSON
output as the legacy call, but typed errors flow back as Err(AIError record)
instead of crashing the host with a Go error.

Particularly useful for agent extractors that retry on schema-validation
failures: AIError.code = "SchemaValidation" identifies a non-retryable
caller-side issue (model output didn't match the schema), distinct from
transient codes like "RateLimit" or "Timeout".`,
			Params: []ParamDoc{
				{Name: "input", Description: "Prompt string"},
				{Name: "schema", Description: "JSON Schema string for response validation"},
			},
			Returns: "Result[string, AIError]",
			SeeAlso: []string{
				"_ai_call_json",
				"_ai_call_result",
				"std/ai.callJsonResult",
			},
			Since:     "v0.17.0",
			Stability: StabilityExperimental,
			Tags:      []string{"ai", "result", "typed-errors", "json", "structured-output"},
			Category:  "ai",
		},
	})
	if err != nil {
		panic("failed to register _ai_call_json_result builtin: " + err.Error())
	}
}

func makeAICallJsonResultType() types.Type {
	T := types.NewBuilder()
	// (input: string, schema: string) -> Result[string, AIError] ! {AI}
	return T.Func(T.String(), T.String()).
		Returns(T.App("Result", T.String(), aiErrorRecordType(T))).
		Effects("AI")
}

func aiCallJsonResultImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	if err := ctx.RequireCapWithBudget("AI", ""); err != nil {
		return nil, err
	}
	return effects.Call(ctx, "AI", "callJsonResult", args)
}

// ============================================================================
// _ai_call_json_simple_result: Result-returning variant of _ai_call_json_simple
// ============================================================================

func registerAICallJsonSimpleResult() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/ai",
		Name:    "_ai_call_json_simple_result",
		NumArgs: 1, // input: string  (no schema, unlike _ai_call_json_result)
		Effect:  "AI",
		Type:    makeAICallResultType, // (string) -> Result[string, AIError] ! {AI} — same shape as _ai_call_result
		Impl:    aiCallJsonSimpleResultImpl,
		Metadata: &BuiltinMetadata{
			Description: "Call the AI oracle requesting JSON (no schema), returning Result[string, AIError]",
			LongDesc: `Result-returning variant of _ai_call_json_simple. Same no-schema raw-JSON
output as the legacy callJsonSimple, but typed errors flow back as
Err(AIError record) instead of crashing the host with a Go error.

Exists separately from _ai_call_json_result because callJsonSimple is the
no-schema path that high-volume extractors must use (callJson has a known
large-response corruption bug). Without this Result variant, a transient
provider 5xx on the extraction path escaped as an uncaught {AI} effect
error; this surfaces it as a catchable Err(AIError{retryable}).`,
			Params: []ParamDoc{
				{Name: "input", Description: "Prompt string"},
			},
			Returns: "Result[string, AIError]",
			SeeAlso: []string{
				"_ai_call_json_simple",
				"_ai_call_json_result",
				"std/ai.callJsonSimpleResult",
			},
			Since:     "v0.23.0",
			Stability: StabilityExperimental,
			Tags:      []string{"ai", "result", "typed-errors", "json"},
			Category:  "ai",
		},
	})
	if err != nil {
		panic("failed to register _ai_call_json_simple_result builtin: " + err.Error())
	}
}

func aiCallJsonSimpleResultImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	if err := ctx.RequireCapWithBudget("AI", ""); err != nil {
		return nil, err
	}
	return effects.Call(ctx, "AI", "callJsonSimpleResult", args)
}

// ============================================================================
// _ai_step: multi-turn / tool-aware completion
// ============================================================================

func registerAIStep() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/ai",
		Name:    "_ai_step",
		NumArgs: 3, // model: string, messages: list[Message], tools: list[ToolSchema]
		Effect:  "AI",
		Type:    makeAIStepType,
		Impl:    aiStepImpl,
		Metadata: &BuiltinMetadata{
			Description: "Multi-turn / tool-aware AI completion returning Result[StepResult, AIError]",
			LongDesc: `One agent turn. Takes a conversation (messages) plus an optional tool
catalog (tools), returns the assistant's response carrying text + any tool
calls the model wants the host to dispatch + a normalized finish_reason.

When finish_reason is "tool_calls", the loop driver (std/ai.runTools, M6)
dispatches each ToolCall via the user-provided callback, appends the
results as tool-role Messages, and calls _ai_step again. When finish_reason
is "stop", the loop terminates.

Provider parity:
  - Anthropic: uses Messages API tool_use content blocks
  - Gemini:    uses functionCall parts (adapter-generated stable IDs)
  - OpenAI / OpenRouter: uses Chat Completions tool_calls
  - Ollama:   rejects tools with AIError{ToolsNotSupported}
  - Configdriven: same as Ollama (v1 schema lacks tool support)

The model parameter is per-call routable — overrides the handler-bound
model. Pass "" to use the handler's bound model.`,
			Params: []ParamDoc{
				{Name: "model", Description: "Model ID (or empty for handler default)"},
				{Name: "messages", Description: "Conversation as list[Message]"},
				{Name: "tools", Description: "Tool catalog as list[ToolSchema]"},
			},
			Returns: "Result[StepResult, AIError]",
			SeeAlso: []string{
				"_ai_call_result",
				"_ai_call_json_result",
				"std/ai.step",
				"std/ai.runTools",
			},
			Since:     "v0.17.0",
			Stability: StabilityExperimental,
			Tags:      []string{"ai", "result", "tool-use", "multi-turn", "agent"},
			Category:  "ai",
		},
	})
	if err != nil {
		panic("failed to register _ai_step builtin: " + err.Error())
	}
}

func makeAIStepType() types.Type {
	T := types.NewBuilder()
	// (model: string, messages: list[Message], tools: list[ToolSchema])
	//   -> Result[StepResult, AIError] ! {AI}
	return T.Func(
		T.String(),
		T.List(messageRecordType(T)),
		T.List(toolSchemaRecordType(T)),
	).
		Returns(T.App("Result", stepResultRecordType(T), aiErrorRecordType(T))).
		Effects("AI")
}

func aiStepImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	if err := ctx.RequireCapWithBudget("AI", ""); err != nil {
		return nil, err
	}
	return effects.Call(ctx, "AI", "step", args)
}

// ============================================================================
// _ai_step_with_cache: cache-aware variant of _ai_step
// (M-AI-PROMPT-CACHING, v0.18.4)
// ============================================================================

func registerAIStepWithCache() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/ai",
		Name:    "_ai_step_with_cache",
		NumArgs: 4, // model, messages, tools, cache_breakpoints
		Effect:  "AI",
		Type:    makeAIStepWithCacheType,
		Impl:    aiStepWithCacheImpl,
		Metadata: &BuiltinMetadata{
			Description: "Cache-aware multi-turn AI completion returning Result[StepResult, AIError]",
			LongDesc: `Identical to _ai_step except it accepts a 4th argument: a list of
opt-in CacheBreakpoint hints that providers interpret per their own caching
contract.

Per-provider behavior:
  - Anthropic (direct/Bedrock/Vertex): stamps cache_control:{type:"ephemeral"}
    on the matching content block. Phase 1 supports position="system" only.
  - OpenAI: NO-OP (auto-caches prompts >=1024 tokens). Emits one-shot
    session warning so callers know hints were ignored.
  - Gemini: NO-OP for v0.18.4. Emits one-shot warning. Explicit
    CachedContent API integration deferred.
  - OpenRouter: dispatches based on model-string prefix (anthropic/...
    -> Anthropic shape; openai/google/other -> NO-OP + warning).
  - Ollama: silent NO-OP (local model, no caching API).

Empty cache_breakpoints produces bit-for-bit identical wire bytes vs
_ai_step. Inspect StepResult.cache_read_input_tokens to verify cache hits.`,
			Params: []ParamDoc{
				{Name: "model", Description: "Model ID (or empty for handler default)"},
				{Name: "messages", Description: "Conversation as list[Message]"},
				{Name: "tools", Description: "Tool catalog as list[ToolSchema]"},
				{Name: "cache_breakpoints", Description: "Opt-in cache hints as list[CacheBreakpoint]"},
			},
			Returns: "Result[StepResult, AIError]",
			SeeAlso: []string{
				"_ai_step",
				"std/ai.stepWithCache",
			},
			Since:     "v0.18.4",
			Stability: StabilityExperimental,
			Tags:      []string{"ai", "result", "tool-use", "multi-turn", "agent", "cache"},
			Category:  "ai",
		},
	})
	if err != nil {
		panic("failed to register _ai_step_with_cache builtin: " + err.Error())
	}
}

func makeAIStepWithCacheType() types.Type {
	T := types.NewBuilder()
	// (model: string, messages: list[Message], tools: list[ToolSchema],
	//  cache_breakpoints: list[CacheBreakpoint])
	//   -> Result[StepResult, AIError] ! {AI}
	return T.Func(
		T.String(),
		T.List(messageRecordType(T)),
		T.List(toolSchemaRecordType(T)),
		T.List(cacheBreakpointRecordType(T)),
	).
		Returns(T.App("Result", stepResultRecordType(T), aiErrorRecordType(T))).
		Effects("AI")
}

func aiStepWithCacheImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	if err := ctx.RequireCapWithBudget("AI", ""); err != nil {
		return nil, err
	}
	return effects.Call(ctx, "AI", "stepWithCache", args)
}

// ============================================================================
// _ai_step_with_stream: streaming variant of _ai_step_with_cache
// (M-AI-STEP-STREAMING, v0.18.7)
// ============================================================================

// streamChunkType returns the AILANG StreamChunk ADT shape.
//
// Mirrors the type definition in std/ai.ail:
//
//	type StreamChunk =
//	  | ContentDelta(string)
//	  | Usage({ input_tokens: int, output_tokens: int,
//	           cache_read_input_tokens: int, cache_creation_input_tokens: int })
//
// Phase 1 (v0.18.7) ships ContentDelta + Usage only. ToolCallDelta and
// ThinkingDelta are reserved for Phase 2 once per-provider JSON-fragment
// buffering is in place.
func streamChunkType(T *types.Builder) types.Type {
	// StreamChunk is exposed as an ADT in AILANG; on the Go-side type-checker
	// surface we treat it as a named type App so callers can pattern-match it.
	return T.App("StreamChunk")
}

// streamUsageRecordType returns the record carried by StreamChunk.Usage.
// Kept as a separate helper so the type checker sees the same shape the
// effect op encodes via encodeStreamChunk.
func streamUsageRecordType(T *types.Builder) types.Type {
	return T.Record(
		types.Field("input_tokens", T.Int()),
		types.Field("output_tokens", T.Int()),
		types.Field("cache_read_input_tokens", T.Int()),
		types.Field("cache_creation_input_tokens", T.Int()),
	)
}

func registerAIStepWithStream() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/ai",
		Name:    "_ai_step_with_stream",
		NumArgs: 5, // model, messages, tools, cache_breakpoints, on_chunk
		Effect:  "AI",
		Type:    makeAIStepWithStreamType,
		Impl:    aiStepWithStreamImpl,
		Metadata: &BuiltinMetadata{
			Description: "Streaming multi-turn AI completion with per-chunk callback returning Result[StepResult, AIError]",
			LongDesc: `Identical to _ai_step_with_cache except for a 5th argument: an AILANG
closure invoked once per StreamChunk as the SSE stream drains. The final
return value is the same Ok(StepResult)/Err(AIError) shape as
_ai_step_with_cache — the callback is purely a side-channel for incremental
rendering (e.g. token-by-token TUI output).

StreamChunk variants (Phase 1):
  ContentDelta(string)
    -- one chunk of assistant text content as it arrives
  Usage({input_tokens, output_tokens, cache_read_input_tokens,
        cache_creation_input_tokens})
    -- final usage block (per-provider timing varies)

Per-provider behavior:
  - Anthropic (direct/Bedrock/Vertex): native SSE, ContentDelta per chunk
    + final Usage from message_delta.
  - OpenAI / OpenRouter (anthropic/* prefix routes through Anthropic shape):
    native SSE, ContentDelta per delta + final Usage from chat.completion.chunk.
  - Gemini / Ollama / config-driven providers: NO-OP fallback — calls
    StepWithCache and fires one synthetic ContentDelta(full_text) + one
    final Usage chunk. Same StepResult shape, no incremental visibility.

The callback's effects propagate via row polymorphism — typically users
will pass an IO-effect closure for stdout streaming, but the row is open.

Empty cache_breakpoints + non-streaming providers produces bit-for-bit
identical StepResult to stepWithCache.`,
			Params: []ParamDoc{
				{Name: "model", Description: "Model ID (or empty for handler default)"},
				{Name: "messages", Description: "Conversation as list[Message]"},
				{Name: "tools", Description: "Tool catalog as list[ToolSchema]"},
				{Name: "cache_breakpoints", Description: "Opt-in cache hints as list[CacheBreakpoint]"},
				{Name: "on_chunk", Description: "Callback (StreamChunk) -> () invoked per chunk"},
			},
			Returns: "Result[StepResult, AIError]",
			SeeAlso: []string{
				"_ai_step_with_cache",
				"_ai_call_stream",
				"std/ai.stepWithStream",
			},
			Since:     "v0.18.7",
			Stability: StabilityExperimental,
			Tags:      []string{"ai", "result", "tool-use", "multi-turn", "agent", "cache", "streaming"},
			Category:  "ai",
		},
	})
	if err != nil {
		panic("failed to register _ai_step_with_stream builtin: " + err.Error())
	}
}

func makeAIStepWithStreamType() types.Type {
	T := types.NewBuilder()
	// (model: string, messages: list[Message], tools: list[ToolSchema],
	//  cache_breakpoints: list[CacheBreakpoint],
	//  on_chunk: (StreamChunk) -> ())
	//   -> Result[StepResult, AIError] ! {AI}
	//
	// Note on callback effects: the callback's effect row is intentionally
	// left open in the Go-side type so AILANG's row polymorphism can flow
	// the user's effects through (typically {IO}). The std/ai.stepWithStream
	// wrapper in std/ai.ail surfaces this as `(StreamChunk) -> () \! {IO}`.
	onChunkType := T.Func(streamChunkType(T)).Returns(T.Unit()).Build()
	return T.Func(
		T.String(),
		T.List(messageRecordType(T)),
		T.List(toolSchemaRecordType(T)),
		T.List(cacheBreakpointRecordType(T)),
		onChunkType,
	).
		Returns(T.App("Result", stepResultRecordType(T), aiErrorRecordType(T))).
		Effects("AI")
}

func aiStepWithStreamImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	if err := ctx.RequireCapWithBudget("AI", ""); err != nil {
		return nil, err
	}
	return effects.Call(ctx, "AI", "stepWithStream", args)
}

// streamUsageRecordType is referenced for documentation symmetry with
// stepResultRecordType / cacheBreakpointRecordType. It is currently unused
// at the Go-side type surface (StreamChunk is exposed as an opaque ADT App
// rather than a record-of-fields) but retained so future tooling that
// inspects builtin metadata can find the canonical shape in one place.
var _ = streamUsageRecordType
