package effects

// Tests for the M-AI-PROMPT-CACHING (v0.18.4) effect op:
//   aiStepWithCache + decodeCacheBreakpoints

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/ai"
	"github.com/sunholo-data/ailang/internal/eval"
)

func cacheBreakpointRecordVal(position, ttl string) *eval.RecordValue {
	return &eval.RecordValue{
		Fields: map[string]eval.Value{
			"position": &eval.StringValue{Value: position},
			"ttl":      &eval.StringValue{Value: ttl},
		},
	}
}

func TestAIStepWithCache_RoundTripsBreakpoints(t *testing.T) {
	h := &fakeStepHandler{
		stepResp: &ai.Response{
			Text:                     "ok",
			FinishReason:             "stop",
			InputTokens:              30,
			OutputTokens:             5,
			CacheReadInputTokens:     20,
			CacheCreationInputTokens: 0,
		},
	}
	ctx := &EffContext{AI: NewAIContext(h)}

	args := []eval.Value{
		&eval.StringValue{Value: "claude-3-5-haiku"},
		&eval.ListValue{Elements: []eval.Value{
			messageRecordVal("user", "hi", nil, ""),
		}},
		&eval.ListValue{Elements: []eval.Value{}},
		&eval.ListValue{Elements: []eval.Value{
			cacheBreakpointRecordVal("system", "ephemeral"),
			cacheBreakpointRecordVal("last_user", "ephemeral"),
		}},
	}
	out, err := aiStepWithCache(ctx, args)
	if err != nil {
		t.Fatalf("unexpected Go error: %v", err)
	}
	tagged := out.(*eval.TaggedValue)
	if tagged.CtorName != "Ok" {
		t.Fatalf("expected Ok, got %s", tagged.CtorName)
	}
	stepResult := tagged.Fields[0].(*eval.RecordValue)

	// Cache-token telemetry round-tripped.
	if got := stepResult.Fields["cache_read_input_tokens"].(*eval.IntValue).Value; got != 20 {
		t.Errorf("cache_read_input_tokens = %d, want 20", got)
	}

	// Handler received the breakpoints verbatim.
	if len(h.lastCacheBreakpoints) != 2 {
		t.Fatalf("handler.lastCacheBreakpoints len = %d, want 2", len(h.lastCacheBreakpoints))
	}
	if h.lastCacheBreakpoints[0].Position != "system" || h.lastCacheBreakpoints[0].TTL != "ephemeral" {
		t.Errorf("breakpoint[0] = %+v, want {system, ephemeral}", h.lastCacheBreakpoints[0])
	}
	if h.lastCacheBreakpoints[1].Position != "last_user" {
		t.Errorf("breakpoint[1].Position = %q, want last_user", h.lastCacheBreakpoints[1].Position)
	}
}

func TestAIStepWithCache_EmptyBreakpointsBehavesLikeStep(t *testing.T) {
	h := &fakeStepHandler{
		stepResp: &ai.Response{Text: "ok", FinishReason: "stop"},
	}
	ctx := &EffContext{AI: NewAIContext(h)}

	args := []eval.Value{
		&eval.StringValue{Value: "any-model"},
		&eval.ListValue{Elements: []eval.Value{}},
		&eval.ListValue{Elements: []eval.Value{}},
		&eval.ListValue{Elements: []eval.Value{}},
	}
	out, err := aiStepWithCache(ctx, args)
	if err != nil {
		t.Fatalf("unexpected Go error: %v", err)
	}
	if out.(*eval.TaggedValue).CtorName != "Ok" {
		t.Fatalf("expected Ok with empty breakpoints")
	}
	if len(h.lastCacheBreakpoints) != 0 {
		t.Errorf("empty breakpoints arg should produce nil/empty handler arg, got %+v", h.lastCacheBreakpoints)
	}
}

func TestAIStepWithCache_MissingBreakpointsArg_TypeError(t *testing.T) {
	ctx := &EffContext{AI: NewAIContext(&fakeStepHandler{})}
	// Only 3 args — should reject with a Go type error (not Result).
	args := []eval.Value{
		&eval.StringValue{Value: "m"},
		&eval.ListValue{},
		&eval.ListValue{},
	}
	_, err := aiStepWithCache(ctx, args)
	if err == nil {
		t.Fatal("expected type error for arg-count mismatch, got nil")
	}
}

func TestAIStepWithCache_NilHandler_ReturnsErr(t *testing.T) {
	ctx := &EffContext{AI: NewAIContext(nil)}
	args := []eval.Value{
		&eval.StringValue{Value: "m"},
		&eval.ListValue{},
		&eval.ListValue{},
		&eval.ListValue{},
	}
	out, err := aiStepWithCache(ctx, args)
	if err != nil {
		t.Fatalf("nil handler should produce typed Err, not Go error; got %v", err)
	}
	if out.(*eval.TaggedValue).CtorName != "Err" {
		t.Fatal("expected Err for nil handler")
	}
}

func TestDecodeCacheBreakpoints_Empty(t *testing.T) {
	// nil and empty list both produce nil/empty slice.
	out, err := decodeCacheBreakpoints(&eval.ListValue{Elements: []eval.Value{}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) != 0 {
		t.Errorf("expected empty result, got %+v", out)
	}
}

func TestDecodeCacheBreakpoints_NonRecordRejected(t *testing.T) {
	_, err := decodeCacheBreakpoints(&eval.ListValue{Elements: []eval.Value{
		&eval.StringValue{Value: "not a record"},
	}})
	if err == nil {
		t.Fatal("expected error for non-record element")
	}
}

func TestDecodeCacheBreakpoints_MissingFieldsAreEmptyStrings(t *testing.T) {
	// Tolerant decoder: missing fields default to "".
	out, err := decodeCacheBreakpoints(&eval.ListValue{Elements: []eval.Value{
		&eval.RecordValue{Fields: map[string]eval.Value{
			"position": &eval.StringValue{Value: "system"},
			// ttl missing
		}},
	}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) != 1 || out[0].Position != "system" || out[0].TTL != "" {
		t.Errorf("expected {system, ''}, got %+v", out)
	}
}
