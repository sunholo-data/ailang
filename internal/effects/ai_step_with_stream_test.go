package effects

// Integration tests for aiStepWithStream (M-AI-STEP-STREAMING, v0.18.7).
//
// These exercise the WHOLE round-trip path that an AILANG caller of
// std/ai.stepWithStream goes through:
//
//   AILANG closure
//     → ctx.FnCaller (the test mock here stands in for the real evaluator)
//     → encodeStreamChunk (Go ai.StreamChunk → AILANG TaggedValue)
//     → AIHandler.StepWithStream
//     → fakeStepHandler fires the synthetic ContentDelta + Usage chunks
//     → typed StepResult returned as Ok(record)
//
// The fakeStepHandler from ai_step_test.go is reused so we don't need a
// separate mock — its StepWithStream method already calls the onChunk
// callback after a Step round-trip.

import (
	"errors"
	"testing"

	"github.com/sunholo-data/ailang/internal/ai"
	"github.com/sunholo-data/ailang/internal/eval"
)

// captureFnCaller is a test mock of EffContext.FnCaller. It records each
// invocation's argument value (the encoded StreamChunk) into the slice
// passed in, then returns Unit (matching the AILANG `() -> ()` callback
// signature we ship in std/ai.ail).
func captureFnCaller(captured *[]eval.Value) func(eval.Value, eval.Value) (eval.Value, error) {
	return func(_ eval.Value, arg eval.Value) (eval.Value, error) {
		*captured = append(*captured, arg)
		return &eval.UnitValue{}, nil
	}
}

// TestAIStepWithStream_FiresContentDeltaAndUsageChunks verifies that the
// per-chunk callback is invoked with properly encoded AILANG StreamChunk
// ADT values for both ContentDelta and Usage variants.
func TestAIStepWithStream_FiresContentDeltaAndUsageChunks(t *testing.T) {
	h := &fakeStepHandler{
		stepResp: &ai.Response{
			Text:                     "hello world",
			InputTokens:              42,
			OutputTokens:             7,
			CacheReadInputTokens:     5,
			CacheCreationInputTokens: 3,
			FinishReason:             "stop",
		},
	}

	var captured []eval.Value
	ctx := &EffContext{
		AI:       NewAIContext(h),
		FnCaller: captureFnCaller(&captured),
		Caps:     map[string]Capability{"AI": NewCapability("AI")},
	}

	// 5 args: model, messages, tools, cache_breakpoints, on_chunk_closure.
	// The closure value can be anything — captureFnCaller ignores it.
	out, err := aiStepWithStream(ctx, []eval.Value{
		&eval.StringValue{Value: "gpt-4o"},
		&eval.ListValue{Elements: []eval.Value{
			&eval.RecordValue{Fields: map[string]eval.Value{
				"role":       &eval.StringValue{Value: "user"},
				"content":    &eval.StringValue{Value: "hi"},
				"tool_calls": &eval.ListValue{Elements: []eval.Value{}},
			}},
		}},
		&eval.ListValue{Elements: []eval.Value{}},
		&eval.ListValue{Elements: []eval.Value{}},
		&eval.UnitValue{}, // closure stand-in
	})
	if err != nil {
		t.Fatalf("aiStepWithStream returned Go error: %v", err)
	}

	// fakeStepHandler.StepWithStream fires one ContentDelta(text) plus one
	// Usage(input/output) — so we expect exactly 2 captured chunks.
	if len(captured) != 2 {
		t.Fatalf("captured chunk count = %d, want 2", len(captured))
	}

	// First chunk: ContentDelta("hello world").
	delta, ok := captured[0].(*eval.TaggedValue)
	if !ok {
		t.Fatalf("captured[0] type = %T, want *eval.TaggedValue", captured[0])
	}
	if delta.CtorName != "ContentDelta" {
		t.Errorf("captured[0].CtorName = %q, want ContentDelta", delta.CtorName)
	}
	if len(delta.Fields) != 1 {
		t.Fatalf("ContentDelta field count = %d, want 1", len(delta.Fields))
	}
	if got := delta.Fields[0].(*eval.StringValue).Value; got != "hello world" {
		t.Errorf("ContentDelta payload = %q, want \"hello world\"", got)
	}

	// Second chunk: Usage(record).
	usage, ok := captured[1].(*eval.TaggedValue)
	if !ok {
		t.Fatalf("captured[1] type = %T, want *eval.TaggedValue", captured[1])
	}
	if usage.CtorName != "Usage" {
		t.Errorf("captured[1].CtorName = %q, want Usage", usage.CtorName)
	}
	usageRec := usage.Fields[0].(*eval.RecordValue)
	if got := usageRec.Fields["input_tokens"].(*eval.IntValue).Value; got != 42 {
		t.Errorf("Usage.input_tokens = %d, want 42", got)
	}
	if got := usageRec.Fields["output_tokens"].(*eval.IntValue).Value; got != 7 {
		t.Errorf("Usage.output_tokens = %d, want 7", got)
	}
	// Note: fakeStepHandler.StepWithStream only forwards InputTokens +
	// OutputTokens in its synthetic Usage chunk (it doesn't know about
	// cache fields), so cache_read/creation chunks come from the StepResult
	// record, not the Usage callback. The cache assertions live in the
	// StepResult check below.

	// Result must be Ok(StepResult record) with the same fields the
	// non-streaming stepWithCache produces.
	tagged, ok := out.(*eval.TaggedValue)
	if !ok {
		t.Fatalf("out type = %T, want *eval.TaggedValue", out)
	}
	if tagged.CtorName != "Ok" {
		t.Fatalf("out.CtorName = %q, want Ok", tagged.CtorName)
	}
	stepResult := tagged.Fields[0].(*eval.RecordValue)
	if got := stepResult.Fields["finish_reason"].(*eval.StringValue).Value; got != "stop" {
		t.Errorf("StepResult.finish_reason = %q, want stop", got)
	}
	if got := stepResult.Fields["input_tokens"].(*eval.IntValue).Value; got != 42 {
		t.Errorf("StepResult.input_tokens = %d, want 42", got)
	}
	if got := stepResult.Fields["cache_read_input_tokens"].(*eval.IntValue).Value; got != 5 {
		t.Errorf("StepResult.cache_read_input_tokens = %d, want 5", got)
	}
}

// TestAIStepWithStream_HandlerErrorReturnsErrResult verifies that a
// handler-level error surfaces as a typed Err(AIError) — the same shape
// stepWithCache produces — rather than aborting the Go-side call.
func TestAIStepWithStream_HandlerErrorReturnsErrResult(t *testing.T) {
	h := &fakeStepHandler{
		stepErr: errors.New("rate limit hit"),
	}

	var captured []eval.Value
	ctx := &EffContext{
		AI:       NewAIContext(h),
		FnCaller: captureFnCaller(&captured),
		Caps:     map[string]Capability{"AI": NewCapability("AI")},
	}

	out, err := aiStepWithStream(ctx, []eval.Value{
		&eval.StringValue{Value: "gpt-4o"},
		&eval.ListValue{Elements: []eval.Value{}},
		&eval.ListValue{Elements: []eval.Value{}},
		&eval.ListValue{Elements: []eval.Value{}},
		&eval.UnitValue{},
	})
	if err != nil {
		t.Fatalf("aiStepWithStream returned Go error (expected nil — error should be in Result): %v", err)
	}

	tagged, ok := out.(*eval.TaggedValue)
	if !ok {
		t.Fatalf("out type = %T, want *eval.TaggedValue", out)
	}
	if tagged.CtorName != "Err" {
		t.Errorf("out.CtorName = %q, want Err", tagged.CtorName)
	}
	errRec := tagged.Fields[0].(*eval.RecordValue)
	if msg := errRec.Fields["message"].(*eval.StringValue).Value; msg == "" {
		t.Error("Err.message is empty")
	}

	// On error, no chunks should fire (the handler aborted before producing any).
	if len(captured) != 0 {
		t.Errorf("captured chunk count = %d, want 0 on handler error", len(captured))
	}
}

// TestAIStepWithStream_NoFnCallerWiredReturnsTypedErr verifies the
// defensive guard for the case where the evaluator hasn't wired
// EffContext.FnCaller (e.g. embed.Engine without the closure callback set).
// Should surface as Err(AIError{Internal}) rather than panic.
func TestAIStepWithStream_NoFnCallerWiredReturnsTypedErr(t *testing.T) {
	h := &fakeStepHandler{stepResp: &ai.Response{Text: "ok"}}
	ctx := &EffContext{
		AI:   NewAIContext(h),
		Caps: map[string]Capability{"AI": NewCapability("AI")},
		// FnCaller intentionally nil
	}

	out, err := aiStepWithStream(ctx, []eval.Value{
		&eval.StringValue{Value: "m"},
		&eval.ListValue{Elements: []eval.Value{}},
		&eval.ListValue{Elements: []eval.Value{}},
		&eval.ListValue{Elements: []eval.Value{}},
		&eval.UnitValue{},
	})
	if err != nil {
		t.Fatalf("aiStepWithStream returned Go error: %v", err)
	}
	tagged, ok := out.(*eval.TaggedValue)
	if !ok || tagged.CtorName != "Err" {
		t.Fatalf("out = %v, want Err(...)", out)
	}
	errRec := tagged.Fields[0].(*eval.RecordValue)
	code := errRec.Fields["code"].(*eval.StringValue).Value
	if code != ai.CodeInternal {
		t.Errorf("Err.code = %q, want %q", code, ai.CodeInternal)
	}
}

// TestAIStepWithStream_PassesCacheBreakpointsToHandler verifies that the
// cache_breakpoints arg is decoded and threaded through to the handler's
// StepWithStream call (so anthropic-style cache_control hints work for
// streaming clients).
func TestAIStepWithStream_PassesCacheBreakpointsToHandler(t *testing.T) {
	h := &fakeStepHandler{stepResp: &ai.Response{Text: "ok"}}

	var captured []eval.Value
	ctx := &EffContext{
		AI:       NewAIContext(h),
		FnCaller: captureFnCaller(&captured),
		Caps:     map[string]Capability{"AI": NewCapability("AI")},
	}

	breakpoints := &eval.ListValue{Elements: []eval.Value{
		&eval.RecordValue{Fields: map[string]eval.Value{
			"position": &eval.StringValue{Value: "system"},
			"ttl":      &eval.StringValue{Value: "ephemeral"},
		}},
	}}

	_, err := aiStepWithStream(ctx, []eval.Value{
		&eval.StringValue{Value: "claude-sonnet-4-5"},
		&eval.ListValue{Elements: []eval.Value{}},
		&eval.ListValue{Elements: []eval.Value{}},
		breakpoints,
		&eval.UnitValue{},
	})
	if err != nil {
		t.Fatalf("aiStepWithStream returned error: %v", err)
	}

	if len(h.lastCacheBreakpoints) != 1 {
		t.Fatalf("handler saw %d cache breakpoints, want 1", len(h.lastCacheBreakpoints))
	}
	if h.lastCacheBreakpoints[0].Position != "system" {
		t.Errorf("breakpoint position = %q, want system", h.lastCacheBreakpoints[0].Position)
	}
	if h.lastCacheBreakpoints[0].TTL != "ephemeral" {
		t.Errorf("breakpoint ttl = %q, want ephemeral", h.lastCacheBreakpoints[0].TTL)
	}
}
