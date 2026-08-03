package effects

// AI multi-turn / tool-aware ops introduced by M-AI-TOOL-LOOP (v0.17.0).
//
// This file defines:
//   - aiCallResult        — Result-returning variant of AI.call
//   - aiCallJsonResult    — Result-returning variant of AI.callJson
//   - aiStep              — multi-turn step with optional tool dispatch
//
// All three return AILANG `Result[..., AIError]` values constructed via
// makeAIErrorResult / makeOkResult so callers don't need to wrap in user
// code. Errors from the underlying handler / provider are converted to
// the canonical *ai.AIError via ai.ClassifyError before being marshalled
// into the AILANG record shape `{code, message, retryable}`.
//
// Tracing: each op records a trace event with the same shape as the
// existing Call/CallJson ops, so dashboards and replay get a uniform
// view across single-shot and tool-loop calls.

import (
	"errors"
	"fmt"

	"github.com/sunholo-data/ailang/internal/ai"
	"github.com/sunholo-data/ailang/internal/eval"
)

// errResultPrefix is the trace-event result-string prefix for op failures.
// One constant so the format string isn't duplicated across the three ops.
const errResultPrefix = "err:%s"

// classifyOpError converts a Go error from an AIHandler call into a typed
// *ai.AIError, with one extra signal beyond ai.ClassifyError: the
// "no AI model configured" sentinel maps to CodeProviderNotFound (so AILANG
// callers see the same code shape `callStream` already produces) instead
// of falling through to CodeInternal.
func classifyOpError(err error) *ai.AIError {
	if errors.Is(err, ErrNoAIHandler) {
		return ai.NewAIError(ai.CodeProviderNotFound, err.Error(), false)
	}
	return ai.ClassifyError(err)
}

// init wires the new ops alongside the existing call/callJson/etc.
// Defined as a separate init function (Go runs all init() in a package
// in source order) so this file can be lifted into a sub-package later
// without disturbing the existing registrations.
func init() {
	RegisterOp("AI", "callResult", aiCallResult)
	RegisterOp("AI", "callJsonResult", aiCallJsonResult)
	RegisterOp("AI", "callJsonSimpleResult", aiCallJsonSimpleResult)
	RegisterOp("AI", "step", aiStep)
	RegisterOp("AI", "stepWithCache", aiStepWithCache)
	RegisterOp("AI", "stepWithStream", aiStepWithStream)
	RegisterOp("AI", "stepWithStreamRecorded", aiStepWithStreamRecorded)
}

// ============================================================================
// aiCallResult — Result-returning AI.call
// ============================================================================

// aiCallResult implements AI.callResult(input: string) -> Result[string, AIError].
// On success returns Ok(text); on failure converts the Go error to *ai.AIError
// via ai.ClassifyError (which passes through pre-typed AIErrors) and returns
// Err(record).
func aiCallResult(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("E_AI_TYPE_ERROR: callResult: expected 1 argument, got %d", len(args))
	}
	input, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("E_AI_TYPE_ERROR: callResult: expected string input, got %T", args[0])
	}
	if ctx.AI == nil {
		// Surface the no-handler condition as a typed AIError so callers
		// using callResult get a uniform Result-shape (rather than a
		// raw Go error from the dispatch layer).
		return makeAIErrorResultRecord(ai.NewAIError(ai.CodeProviderNotFound, ErrNoAIHandler.Error(), false)), nil
	}

	output, err := ctx.AI.Call(input.Value)
	if err != nil {
		aiErr := classifyOpError(err)
		ctx.RecordAIEffect("callResult",
			[]string{truncateForTrace(input.Value)},
			fmt.Sprintf(errResultPrefix, aiErr.Code),
			ctx.AI.LastRoutingMetadata(),
		)
		return makeAIErrorResultRecord(aiErr), nil
	}

	ctx.RecordAIEffect("callResult",
		[]string{truncateForTrace(input.Value)},
		truncateForTrace(output),
		ctx.AI.LastRoutingMetadata(),
	)
	return makeOkStringResult(output), nil
}

// ============================================================================
// aiCallJsonResult — Result-returning AI.callJson
// ============================================================================

// aiCallJsonResult implements AI.callJsonResult(input: string, schema: string)
// -> Result[string, AIError].
func aiCallJsonResult(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("E_AI_TYPE_ERROR: callJsonResult: expected 2 arguments, got %d", len(args))
	}
	input, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("E_AI_TYPE_ERROR: callJsonResult: expected string input, got %T", args[0])
	}
	schema, ok := args[1].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("E_AI_TYPE_ERROR: callJsonResult: expected string schema, got %T", args[1])
	}
	if ctx.AI == nil {
		return makeAIErrorResultRecord(ai.NewAIError(ai.CodeProviderNotFound, ErrNoAIHandler.Error(), false)), nil
	}

	output, err := ctx.AI.CallJson(input.Value, schema.Value)
	if err != nil {
		aiErr := classifyOpError(err)
		ctx.RecordAIEffect("callJsonResult",
			[]string{truncateForTrace(input.Value), truncateForTrace(schema.Value)},
			fmt.Sprintf(errResultPrefix, aiErr.Code),
			ctx.AI.LastRoutingMetadata(),
		)
		return makeAIErrorResultRecord(aiErr), nil
	}

	ctx.RecordAIEffect("callJsonResult",
		[]string{truncateForTrace(input.Value), truncateForTrace(schema.Value)},
		truncateForTrace(output),
		ctx.AI.LastRoutingMetadata(),
	)
	return makeOkStringResult(output), nil
}

// ============================================================================
// aiCallJsonSimpleResult — Result-returning AI.callJsonSimple
// ============================================================================

// aiCallJsonSimpleResult implements AI.callJsonSimpleResult(input: string)
// -> Result[string, AIError]. The no-schema sibling of aiCallJsonResult:
// same raw-JSON-string output as the legacy callJsonSimple (which is
// CallJson with an empty schema), but transient provider failures return
// Err(AIError{retryable}) instead of crashing the host. M-DOCPARSE-RESILIENCE-FIXES.
func aiCallJsonSimpleResult(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("E_AI_TYPE_ERROR: callJsonSimpleResult: expected 1 argument, got %d", len(args))
	}
	input, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("E_AI_TYPE_ERROR: callJsonSimpleResult: expected string input, got %T", args[0])
	}
	if ctx.AI == nil {
		return makeAIErrorResultRecord(ai.NewAIError(ai.CodeProviderNotFound, ErrNoAIHandler.Error(), false)), nil
	}

	// callJsonSimple == CallJson with empty schema (no schema enforcement).
	output, err := ctx.AI.CallJson(input.Value, "")
	if err != nil {
		aiErr := classifyOpError(err)
		ctx.RecordAIEffect("callJsonSimpleResult",
			[]string{truncateForTrace(input.Value)},
			fmt.Sprintf(errResultPrefix, aiErr.Code),
			ctx.AI.LastRoutingMetadata(),
		)
		return makeAIErrorResultRecord(aiErr), nil
	}

	ctx.RecordAIEffect("callJsonSimpleResult",
		[]string{truncateForTrace(input.Value)},
		truncateForTrace(output),
		ctx.AI.LastRoutingMetadata(),
	)
	return makeOkStringResult(output), nil
}

// ============================================================================
// aiStep — multi-turn / tool-aware
// ============================================================================

// aiStep implements AI.step(model: string, messages: list[Message],
// tools: list[ToolSchema]) -> Result[StepResult, AIError].
//
// The args layout matches the AILANG-side signature:
//
//	args[0] = StringValue(model)
//	args[1] = ListValue([RecordValue(Message), ...])
//	args[2] = ListValue([RecordValue(ToolSchema), ...])
//
// Each AILANG record is decoded into the matching ai.Message / ai.ToolSchema
// struct, dispatched to the handler's Step, and the *ai.Response is
// re-encoded as an AILANG StepResult record wrapped in Ok(). On failure
// the *ai.AIError is wrapped in Err().
func aiStep(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	if len(args) < 3 {
		return nil, fmt.Errorf("E_AI_TYPE_ERROR: step: expected 3 arguments, got %d", len(args))
	}
	model, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("E_AI_TYPE_ERROR: step: expected string model, got %T", args[0])
	}
	messagesArg, ok := args[1].(*eval.ListValue)
	if !ok {
		return nil, fmt.Errorf("E_AI_TYPE_ERROR: step: expected list[Message] messages, got %T", args[1])
	}
	toolsArg, ok := args[2].(*eval.ListValue)
	if !ok {
		return nil, fmt.Errorf("E_AI_TYPE_ERROR: step: expected list[ToolSchema] tools, got %T", args[2])
	}
	if ctx.AI == nil {
		return makeAIErrorResultRecord(ai.NewAIError(ai.CodeProviderNotFound, ErrNoAIHandler.Error(), false)), nil
	}

	messages, conversionErr := decodeMessages(messagesArg)
	if conversionErr != nil {
		// Schema-level error in caller-supplied AILANG records — return as
		// a typed AIError rather than propagating a raw Go error.
		return makeAIErrorResultRecord(ai.NewAIError(ai.CodeSchemaValidation, conversionErr.Error(), false)), nil
	}
	tools, conversionErr := decodeToolSchemas(toolsArg)
	if conversionErr != nil {
		return makeAIErrorResultRecord(ai.NewAIError(ai.CodeSchemaValidation, conversionErr.Error(), false)), nil
	}

	resp, err := ctx.AI.Step(model.Value, messages, tools)
	if err != nil {
		aiErr := classifyOpError(err)
		ctx.RecordAIEffect("step",
			[]string{truncateForTrace(model.Value), fmt.Sprintf("messages:%d tools:%d", len(messages), len(tools))},
			fmt.Sprintf(errResultPrefix, aiErr.Code),
			ctx.AI.LastRoutingMetadata(),
		)
		return makeAIErrorResultRecord(aiErr), nil
	}

	ctx.RecordAIEffect("step",
		[]string{truncateForTrace(model.Value), fmt.Sprintf("messages:%d tools:%d", len(messages), len(tools))},
		fmt.Sprintf("text:%s tool_calls:%d finish:%s", truncateForTrace(resp.Text), len(resp.ToolCalls), resp.FinishReason),
		ctx.AI.LastRoutingMetadata(),
	)
	return makeOkStepResult(resp), nil
}

// ============================================================================
// aiStepWithCache — multi-turn / tool-aware with opt-in prompt-cache hints
// (M-AI-PROMPT-CACHING, v0.18.4)
// ============================================================================

// aiStepWithCache implements AI.stepWithCache(model, messages, tools,
// cache_breakpoints) -> Result[StepResult, AIError]. Identical to aiStep
// except it accepts a 4th argument: a list[CacheBreakpoint] of opt-in
// prompt-cache hints, threaded onto the underlying provider Request.
//
// Empty cache_breakpoints produces bit-for-bit identical behavior to aiStep.
func aiStepWithCache(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	if len(args) < 4 {
		return nil, fmt.Errorf("E_AI_TYPE_ERROR: stepWithCache: expected 4 arguments, got %d", len(args))
	}
	model, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("E_AI_TYPE_ERROR: stepWithCache: expected string model, got %T", args[0])
	}
	messagesArg, ok := args[1].(*eval.ListValue)
	if !ok {
		return nil, fmt.Errorf("E_AI_TYPE_ERROR: stepWithCache: expected list[Message] messages, got %T", args[1])
	}
	toolsArg, ok := args[2].(*eval.ListValue)
	if !ok {
		return nil, fmt.Errorf("E_AI_TYPE_ERROR: stepWithCache: expected list[ToolSchema] tools, got %T", args[2])
	}
	breakpointsArg, ok := args[3].(*eval.ListValue)
	if !ok {
		return nil, fmt.Errorf("E_AI_TYPE_ERROR: stepWithCache: expected list[CacheBreakpoint] cache_breakpoints, got %T", args[3])
	}
	if ctx.AI == nil {
		return makeAIErrorResultRecord(ai.NewAIError(ai.CodeProviderNotFound, ErrNoAIHandler.Error(), false)), nil
	}

	messages, conversionErr := decodeMessages(messagesArg)
	if conversionErr != nil {
		return makeAIErrorResultRecord(ai.NewAIError(ai.CodeSchemaValidation, conversionErr.Error(), false)), nil
	}
	tools, conversionErr := decodeToolSchemas(toolsArg)
	if conversionErr != nil {
		return makeAIErrorResultRecord(ai.NewAIError(ai.CodeSchemaValidation, conversionErr.Error(), false)), nil
	}
	breakpoints, conversionErr := decodeCacheBreakpoints(breakpointsArg)
	if conversionErr != nil {
		return makeAIErrorResultRecord(ai.NewAIError(ai.CodeSchemaValidation, conversionErr.Error(), false)), nil
	}

	resp, err := ctx.AI.StepWithCache(model.Value, messages, tools, breakpoints)
	if err != nil {
		aiErr := classifyOpError(err)
		ctx.RecordAIEffect("stepWithCache",
			[]string{truncateForTrace(model.Value), fmt.Sprintf("messages:%d tools:%d cache:%d", len(messages), len(tools), len(breakpoints))},
			fmt.Sprintf(errResultPrefix, aiErr.Code),
			ctx.AI.LastRoutingMetadata(),
		)
		return makeAIErrorResultRecord(aiErr), nil
	}

	ctx.RecordAIEffect("stepWithCache",
		[]string{truncateForTrace(model.Value), fmt.Sprintf("messages:%d tools:%d cache:%d", len(messages), len(tools), len(breakpoints))},
		fmt.Sprintf("text:%s tool_calls:%d finish:%s cache_read:%d cache_create:%d", truncateForTrace(resp.Text), len(resp.ToolCalls), resp.FinishReason, resp.CacheReadInputTokens, resp.CacheCreationInputTokens),
		ctx.AI.LastRoutingMetadata(),
	)
	return makeOkStepResult(resp), nil
}

// ============================================================================
// aiStepWithStream — multi-turn / tool-aware with per-chunk callback
// (M-AI-STEP-STREAMING, v0.18.7)
// ============================================================================

// aiStepWithStream implements AI.stepWithStream(model, messages, tools,
// cache_breakpoints, on_chunk) -> Result[StepResult, AIError].
//
// Identical to aiStepWithCache except for a 5th argument: an AILANG closure
// invoked once per ai.StreamChunk as the SSE stream drains. The final return
// is the same Ok(StepResult)/Err(AIError) shape as stepWithCache — the
// callback is purely a side-channel for incremental rendering. Providers
// without native streaming (Gemini, Ollama, configdriven) NO-OP fall back to
// StepWithCache and fire one synthetic ContentDelta + Usage at the end.
func aiStepWithStream(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	_, resp, aiErr, err := aiStreamCore(ctx, args, "stepWithStream", streamRecordPolicy{})
	if err != nil {
		return nil, err
	}
	if aiErr != nil {
		return makeAIErrorResultRecord(aiErr), nil
	}
	return makeOkStepResult(resp), nil
}

// ============================================================================
// aiStepWithStreamRecorded — the recorded-stream API (#546), adopted from
// @arniwesth's prototype (arniwesth/ailang#2) and productionized in v0.32.0
// with the fail-loud + bounded-inert-drain policy (see ai_stream_core.go).
//
// Identical to aiStepWithStream except the return shape: it preserves
// immediate per-chunk callbacks AND returns the exact ordered list of observed
// chunks alongside the final outcome:
//
//   { chunks: [StreamChunk], outcome: Result[StepResult, AIError] }
//
// The chunks are returned on BOTH outcomes. That is the point of the shape: a
// Result[{result, chunks}, AIError] discards every chunk observed before a
// mid-stream failure, which is the case a deterministic replay most depends on.
// ============================================================================

func aiStepWithStreamRecorded(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	recorded, resp, aiErr, err := aiStreamCore(ctx, args, "stepWithStreamRecorded", streamRecordPolicy{record: true, failLoud: true})
	if err != nil {
		return nil, err
	}
	if aiErr != nil {
		return makeRecordedStream(recorded, makeAIErrorResultRecord(aiErr)), nil
	}
	return makeRecordedStream(recorded, makeOkStepResult(resp)), nil
}

// makeRecordedStream builds { chunks: [StreamChunk], outcome: Result[...] }.
func makeRecordedStream(chunks []eval.Value, outcome eval.Value) eval.Value {
	if chunks == nil {
		chunks = []eval.Value{}
	}
	return &eval.RecordValue{
		Fields: map[string]eval.Value{
			"chunks":  &eval.ListValue{Elements: chunks},
			"outcome": outcome,
		},
	}
}
