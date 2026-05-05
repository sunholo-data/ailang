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

// stepResultRecordType returns the AILANG StepResult record shape.
func stepResultRecordType(T *types.Builder) types.Type {
	return T.Record(
		types.Field("message", messageRecordType(T)),
		types.Field("tool_calls", T.List(toolCallRecordType(T))),
		types.Field("finish_reason", T.String()),
		types.Field("input_tokens", T.Int()),
		types.Field("output_tokens", T.Int()),
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
