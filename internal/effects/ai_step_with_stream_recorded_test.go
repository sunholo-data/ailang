package effects

// Tests for aiStepWithStreamRecorded — the recorded-stream variant that keeps
// immediate per-chunk callback delivery AND returns the exact ordered observed
// chunks alongside the final outcome.
//
// The properties under test are the three the consumer depends on:
//
//   1. delivery AND capture, not either — the callback still fires per chunk
//      while the same chunks come back in the result;
//   2. chunks on BOTH outcomes — a stream that fails part-way still returns
//      every chunk observed before the failure;
//   3. identity, not reconstruction — the returned chunks are the values
//      handed to the callback, in order, and concatenating the ContentDelta
//      payloads still equals StepResult.message.content.
//
// The success-path fake is the shared fakeStepHandler from ai_step_test.go.
// The error path needs its own: fakeStepHandler returns before emitting
// anything when Step fails, so "chunks then error" was previously untestable.

import (
	"errors"
	"testing"

	"github.com/sunholo-data/ailang/internal/ai"
	"github.com/sunholo-data/ailang/internal/eval"
)

// partialThenFailHandler emits real chunks and *then* fails, which is the
// case the recorded API exists for and the one a Result[{result, chunks}, ...]
// shape would silently discard.
type partialThenFailHandler struct {
	chunks []string
	err    error
}

func (h *partialThenFailHandler) Call(_ string) (string, error)        { return "", nil }
func (h *partialThenFailHandler) CallJson(_, _ string) (string, error) { return "", nil }
func (h *partialThenFailHandler) CallImage(_, out, _ string) (string, error) {
	return out, nil
}
func (h *partialThenFailHandler) CallImageBase64(_, _ string) (string, error) { return "", nil }
func (h *partialThenFailHandler) Step(_ string, _ []ai.Message, _ []ai.ToolSchema) (*ai.Response, error) {
	return nil, h.err
}
func (h *partialThenFailHandler) StepWithCache(_ string, _ []ai.Message, _ []ai.ToolSchema, _ []ai.CacheBreakpoint) (*ai.Response, error) {
	return nil, h.err
}
func (h *partialThenFailHandler) StepWithStream(_ string, _ []ai.Message, _ []ai.ToolSchema, _ []ai.CacheBreakpoint, onChunk func(ai.StreamChunk)) (*ai.Response, error) {
	for _, c := range h.chunks {
		onChunk(ai.StreamContentDelta{Text: c})
	}
	return nil, h.err
}

// recordedArgs builds the 5-argument call shape shared by these tests.
func recordedArgs() []eval.Value {
	return []eval.Value{
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
		&eval.UnitValue{},
	}
}

// splitRecorded destructures the { chunks, outcome } record.
func splitRecorded(t *testing.T, out eval.Value) ([]eval.Value, *eval.TaggedValue) {
	t.Helper()
	rec, ok := out.(*eval.RecordValue)
	if !ok {
		t.Fatalf("result type = %T, want *eval.RecordValue", out)
	}
	chunks, ok := rec.Fields["chunks"].(*eval.ListValue)
	if !ok {
		t.Fatalf("result.chunks type = %T, want *eval.ListValue", rec.Fields["chunks"])
	}
	outcome, ok := rec.Fields["outcome"].(*eval.TaggedValue)
	if !ok {
		t.Fatalf("result.outcome type = %T, want *eval.TaggedValue", rec.Fields["outcome"])
	}
	return chunks.Elements, outcome
}

func contentDeltaText(t *testing.T, v eval.Value) (string, bool) {
	t.Helper()
	tv, ok := v.(*eval.TaggedValue)
	if !ok || tv.CtorName != "ContentDelta" {
		return "", false
	}
	return tv.Fields[0].(*eval.StringValue).Value, true
}

// TestAIStepWithStreamRecorded_ReturnsDeliveredChunksOnSuccess is property 1
// and property 3: the callback fired, and the identical sequence came back.
func TestAIStepWithStreamRecorded_ReturnsDeliveredChunksOnSuccess(t *testing.T) {
	h := &fakeStepHandler{
		stepResp: &ai.Response{
			Text: "hello world", InputTokens: 42, OutputTokens: 7, FinishReason: "stop",
		},
	}
	var captured []eval.Value
	ctx := &EffContext{
		AI:       NewAIContext(h),
		FnCaller: captureFnCaller(&captured),
		Caps:     map[string]Capability{"AI": NewCapability("AI")},
	}

	out, err := aiStepWithStreamRecorded(ctx, recordedArgs())
	if err != nil {
		t.Fatalf("aiStepWithStreamRecorded returned Go error: %v", err)
	}
	returned, outcome := splitRecorded(t, out)

	// Delivery still happened, and capture did not replace it.
	if len(captured) != 2 {
		t.Fatalf("callback invocations = %d, want 2", len(captured))
	}
	// No duplicate delivery: projected count == returned count.
	if len(returned) != len(captured) {
		t.Fatalf("returned chunk count = %d, want %d (same as delivered)", len(returned), len(captured))
	}
	// Identity, in order: the returned values are the delivered values.
	for i := range captured {
		if captured[i] != returned[i] {
			t.Errorf("chunk %d: returned value is not the delivered value", i)
		}
	}
	if outcome.CtorName != "Ok" {
		t.Fatalf("outcome.CtorName = %q, want Ok", outcome.CtorName)
	}
}

// TestAIStepWithStreamRecorded_ReturnsChunksOnErrorPath is property 2, and the
// reason the return is a record rather than Result[{result, chunks}, AIError]:
// that shape has nowhere to put chunks when the stream fails.
func TestAIStepWithStreamRecorded_ReturnsChunksOnErrorPath(t *testing.T) {
	h := &partialThenFailHandler{
		chunks: []string{"partial-1", "partial-2"},
		err:    errors.New("connection reset by peer"),
	}
	var captured []eval.Value
	ctx := &EffContext{
		AI:       NewAIContext(h),
		FnCaller: captureFnCaller(&captured),
		Caps:     map[string]Capability{"AI": NewCapability("AI")},
	}

	out, err := aiStepWithStreamRecorded(ctx, recordedArgs())
	if err != nil {
		t.Fatalf("aiStepWithStreamRecorded returned Go error: %v", err)
	}
	returned, outcome := splitRecorded(t, out)

	if outcome.CtorName != "Err" {
		t.Fatalf("outcome.CtorName = %q, want Err", outcome.CtorName)
	}
	// Both pre-failure chunks survive.
	if len(returned) != 2 {
		t.Fatalf("returned chunk count on error path = %d, want 2", len(returned))
	}
	want := []string{"partial-1", "partial-2"}
	for i, w := range want {
		got, ok := contentDeltaText(t, returned[i])
		if !ok {
			t.Fatalf("returned[%d] is not a ContentDelta", i)
		}
		if got != w {
			t.Errorf("returned[%d] = %q, want %q", i, got, w)
		}
	}
	if len(captured) != 2 {
		t.Errorf("callback invocations on error path = %d, want 2", len(captured))
	}
}

// TestAIStepWithStreamRecorded_ContentDeltaConcatEqualsMessageContent holds the
// documented StreamChunk invariant across the new surface.
func TestAIStepWithStreamRecorded_ContentDeltaConcatEqualsMessageContent(t *testing.T) {
	h := &fakeStepHandler{
		stepResp: &ai.Response{Text: "hello world", FinishReason: "stop"},
	}
	var captured []eval.Value
	ctx := &EffContext{
		AI:       NewAIContext(h),
		FnCaller: captureFnCaller(&captured),
		Caps:     map[string]Capability{"AI": NewCapability("AI")},
	}

	out, err := aiStepWithStreamRecorded(ctx, recordedArgs())
	if err != nil {
		t.Fatalf("aiStepWithStreamRecorded returned Go error: %v", err)
	}
	returned, outcome := splitRecorded(t, out)

	concat := ""
	for _, c := range returned {
		if text, ok := contentDeltaText(t, c); ok {
			concat += text
		}
	}
	stepResult := outcome.Fields[0].(*eval.RecordValue)
	msg := stepResult.Fields["message"].(*eval.RecordValue)
	content := msg.Fields["content"].(*eval.StringValue).Value
	if concat != content {
		t.Errorf("concat(ContentDelta) = %q, want message.content %q", concat, content)
	}
}

// TestAIStepWithStream_UnchangedByRecordedVariant pins the additive claim: the
// existing entry point still returns Result[StepResult, AIError] directly, not
// a record, so no current caller is affected.
func TestAIStepWithStream_UnchangedByRecordedVariant(t *testing.T) {
	h := &fakeStepHandler{
		stepResp: &ai.Response{Text: "hello world", FinishReason: "stop"},
	}
	var captured []eval.Value
	ctx := &EffContext{
		AI:       NewAIContext(h),
		FnCaller: captureFnCaller(&captured),
		Caps:     map[string]Capability{"AI": NewCapability("AI")},
	}

	out, err := aiStepWithStream(ctx, recordedArgs())
	if err != nil {
		t.Fatalf("aiStepWithStream returned Go error: %v", err)
	}
	tagged, ok := out.(*eval.TaggedValue)
	if !ok {
		t.Fatalf("stepWithStream result type = %T, want *eval.TaggedValue (unchanged)", out)
	}
	if tagged.CtorName != "Ok" {
		t.Fatalf("stepWithStream outcome = %q, want Ok", tagged.CtorName)
	}
	if len(captured) != 2 {
		t.Errorf("stepWithStream callback invocations = %d, want 2", len(captured))
	}
}
