package effects

// Tests for the M-AI-TOOL-LOOP (v0.17.0) effect ops:
//   aiCallResult, aiCallJsonResult, aiStep
// plus the AILANG record encoders/decoders.

import (
	"errors"
	"testing"

	"github.com/sunholo-data/ailang/internal/ai"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/trace"
)

// fakeStepHandler is a controllable AIHandler for Step-path testing.
// Lets each test pin a Step response, error, and capture the most-recent
// arguments for assertion.
type fakeStepHandler struct {
	stepResp             *ai.Response
	stepErr              error
	lastModel            string
	lastMessages         []ai.Message
	lastTools            []ai.ToolSchema
	lastCacheBreakpoints []ai.CacheBreakpoint
	lastStreamChunkCount int
	// Call/CallJson responses for callResult/callJsonResult tests.
	callResp     string
	callErr      error
	callJsonResp string
	callJsonErr  error
}

func (h *fakeStepHandler) Call(_ string) (string, error) {
	return h.callResp, h.callErr
}
func (h *fakeStepHandler) CallJson(_, _ string) (string, error) {
	return h.callJsonResp, h.callJsonErr
}
func (h *fakeStepHandler) CallImage(_, outputPath, _ string) (string, error) {
	return outputPath, nil
}
func (h *fakeStepHandler) CallImageBase64(_, _ string) (string, error) {
	return "", nil
}
func (h *fakeStepHandler) Step(model string, messages []ai.Message, tools []ai.ToolSchema) (*ai.Response, error) {
	h.lastModel = model
	h.lastMessages = messages
	h.lastTools = tools
	return h.stepResp, h.stepErr
}
func (h *fakeStepHandler) StepWithStream(model string, messages []ai.Message, tools []ai.ToolSchema, cacheBreakpoints []ai.CacheBreakpoint, onChunk func(ai.StreamChunk)) (*ai.Response, error) {
	h.lastCacheBreakpoints = cacheBreakpoints
	h.lastStreamChunkCount = 0
	resp, err := h.Step(model, messages, tools)
	if err != nil {
		return nil, err
	}
	if onChunk != nil {
		if resp.Text != "" {
			onChunk(ai.StreamContentDelta{Text: resp.Text})
			h.lastStreamChunkCount++
		}
		onChunk(ai.StreamUsage{InputTokens: resp.InputTokens, OutputTokens: resp.OutputTokens})
		h.lastStreamChunkCount++
	}
	return resp, nil
}
func (h *fakeStepHandler) StepWithCache(model string, messages []ai.Message, tools []ai.ToolSchema, cacheBreakpoints []ai.CacheBreakpoint) (*ai.Response, error) {
	h.lastCacheBreakpoints = cacheBreakpoints
	return h.Step(model, messages, tools)
}

// ============================================================================
// callResult / callJsonResult
// ============================================================================

func TestAICallResult_HappyPath_ReturnsOk(t *testing.T) {
	h := &fakeStepHandler{callResp: "hello"}
	ctx := &EffContext{AI: NewAIContext(h)}

	out, err := aiCallResult(ctx, []eval.Value{&eval.StringValue{Value: "hi"}})
	if err != nil {
		t.Fatalf("unexpected Go error: %v", err)
	}
	tagged := out.(*eval.TaggedValue)
	if tagged.CtorName != "Ok" {
		t.Fatalf("expected Ok, got %s", tagged.CtorName)
	}
	if got := tagged.Fields[0].(*eval.StringValue).Value; got != "hello" {
		t.Errorf("Ok payload = %q, want \"hello\"", got)
	}
}

func TestAICallResult_HandlerError_ReturnsErr(t *testing.T) {
	h := &fakeStepHandler{callErr: errors.New("rate limit hit")}
	ctx := &EffContext{AI: NewAIContext(h)}

	out, err := aiCallResult(ctx, []eval.Value{&eval.StringValue{Value: "hi"}})
	if err != nil {
		t.Fatalf("aiCallResult should not surface Go error; got %v", err)
	}
	tagged := out.(*eval.TaggedValue)
	if tagged.CtorName != "Err" {
		t.Fatalf("expected Err, got %s", tagged.CtorName)
	}
	rec := tagged.Fields[0].(*eval.RecordValue)
	if code := rec.Fields["code"].(*eval.StringValue).Value; code == "" {
		t.Error("AIError.code is empty")
	}
}

func TestAICallResult_TypedAIErrorPassesThrough(t *testing.T) {
	// Handler returns an *ai.AIError directly — ClassifyError should pass
	// it through verbatim rather than re-classifying.
	original := ai.NewAIError(ai.CodeAuthFailed, "bad key", false)
	h := &fakeStepHandler{callErr: original}
	ctx := &EffContext{AI: NewAIContext(h)}

	out, _ := aiCallResult(ctx, []eval.Value{&eval.StringValue{Value: "hi"}})
	rec := out.(*eval.TaggedValue).Fields[0].(*eval.RecordValue)
	if got := rec.Fields["code"].(*eval.StringValue).Value; got != ai.CodeAuthFailed {
		t.Errorf("code = %q, want %q (verbatim pass-through)", got, ai.CodeAuthFailed)
	}
	if rec.Fields["retryable"].(*eval.BoolValue).Value {
		t.Error("retryable = true, want false (AuthFailed)")
	}
}

func TestAICallResult_NilHandler_ReturnsErr(t *testing.T) {
	ctx := &EffContext{AI: NewAIContext(nil)}

	out, err := aiCallResult(ctx, []eval.Value{&eval.StringValue{Value: "hi"}})
	if err != nil {
		t.Fatalf("nil handler should produce typed Err, not Go error; got %v", err)
	}
	tagged := out.(*eval.TaggedValue)
	if tagged.CtorName != "Err" {
		t.Fatalf("expected Err, got %s", tagged.CtorName)
	}
	rec := tagged.Fields[0].(*eval.RecordValue)
	if code := rec.Fields["code"].(*eval.StringValue).Value; code != ai.CodeProviderNotFound {
		t.Errorf("code = %q, want %q", code, ai.CodeProviderNotFound)
	}
}

func TestAICallJsonResult_HappyPath(t *testing.T) {
	h := &fakeStepHandler{callJsonResp: `{"name":"foo"}`}
	ctx := &EffContext{AI: NewAIContext(h)}

	out, _ := aiCallJsonResult(ctx, []eval.Value{
		&eval.StringValue{Value: "give me json"},
		&eval.StringValue{Value: `{"type":"object"}`},
	})
	tagged := out.(*eval.TaggedValue)
	if tagged.CtorName != "Ok" {
		t.Fatalf("expected Ok, got %s", tagged.CtorName)
	}
	if got := tagged.Fields[0].(*eval.StringValue).Value; got != `{"name":"foo"}` {
		t.Errorf("Ok payload = %q", got)
	}
}

// ============================================================================
// callJsonSimpleResult (M-DOCPARSE-RESILIENCE-FIXES) — no-schema Result variant
// ============================================================================

func TestAICallJsonSimpleResult_HappyPath_ReturnsOk(t *testing.T) {
	// callJsonSimple routes through CallJson(input, "") — fakeStepHandler
	// returns callJsonResp regardless of schema, so this exercises the Ok path.
	h := &fakeStepHandler{callJsonResp: `[1,2,3]`}
	ctx := &EffContext{AI: NewAIContext(h)}

	out, err := aiCallJsonSimpleResult(ctx, []eval.Value{&eval.StringValue{Value: "give me json"}})
	if err != nil {
		t.Fatalf("unexpected Go error: %v", err)
	}
	tagged := out.(*eval.TaggedValue)
	if tagged.CtorName != "Ok" {
		t.Fatalf("expected Ok, got %s", tagged.CtorName)
	}
	if got := tagged.Fields[0].(*eval.StringValue).Value; got != `[1,2,3]` {
		t.Errorf("Ok payload = %q, want \"[1,2,3]\"", got)
	}
}

func TestAICallJsonSimpleResult_HandlerError_ReturnsErr(t *testing.T) {
	h := &fakeStepHandler{callJsonErr: errors.New("Internal error encountered")}
	ctx := &EffContext{AI: NewAIContext(h)}

	out, err := aiCallJsonSimpleResult(ctx, []eval.Value{&eval.StringValue{Value: "hi"}})
	if err != nil {
		t.Fatalf("aiCallJsonSimpleResult should not surface Go error; got %v", err)
	}
	tagged := out.(*eval.TaggedValue)
	if tagged.CtorName != "Err" {
		t.Fatalf("expected Err, got %s", tagged.CtorName)
	}
	rec := tagged.Fields[0].(*eval.RecordValue)
	if code := rec.Fields["code"].(*eval.StringValue).Value; code == "" {
		t.Error("AIError.code is empty")
	}
}

func TestAICallJsonSimpleResult_TypedAIErrorPassesThrough(t *testing.T) {
	// A transient failure classified as retryable must pass through with
	// retryable=true — this is the whole point of the variant (docparse retry).
	original := ai.NewAIError(ai.CodeTimeout, "500 Internal error encountered", true)
	h := &fakeStepHandler{callJsonErr: original}
	ctx := &EffContext{AI: NewAIContext(h)}

	out, _ := aiCallJsonSimpleResult(ctx, []eval.Value{&eval.StringValue{Value: "hi"}})
	rec := out.(*eval.TaggedValue).Fields[0].(*eval.RecordValue)
	if got := rec.Fields["code"].(*eval.StringValue).Value; got != ai.CodeTimeout {
		t.Errorf("code = %q, want %q (verbatim pass-through)", got, ai.CodeTimeout)
	}
	if !rec.Fields["retryable"].(*eval.BoolValue).Value {
		t.Error("retryable = false, want true (transient 5xx) — retry path would be broken")
	}
}

func TestAICallJsonSimpleResult_NilHandler_ReturnsErr(t *testing.T) {
	ctx := &EffContext{AI: NewAIContext(nil)}

	out, err := aiCallJsonSimpleResult(ctx, []eval.Value{&eval.StringValue{Value: "hi"}})
	if err != nil {
		t.Fatalf("nil handler should produce typed Err, not Go error; got %v", err)
	}
	tagged := out.(*eval.TaggedValue)
	if tagged.CtorName != "Err" {
		t.Fatalf("expected Err, got %s", tagged.CtorName)
	}
	rec := tagged.Fields[0].(*eval.RecordValue)
	if code := rec.Fields["code"].(*eval.StringValue).Value; code != ai.CodeProviderNotFound {
		t.Errorf("code = %q, want %q", code, ai.CodeProviderNotFound)
	}
}

// ============================================================================
// aiStep
// ============================================================================

func TestAIStep_HappyPath_ReturnsOkStepResult(t *testing.T) {
	h := &fakeStepHandler{
		stepResp: &ai.Response{
			Text:         "I'll read the doc",
			ToolCalls:    []ai.ToolCall{{ID: "call_1", Name: "read_doc", Arguments: `{"name":"nda.docx"}`}},
			FinishReason: "tool_calls",
			InputTokens:  50,
			OutputTokens: 25,
		},
	}
	ctx := &EffContext{AI: NewAIContext(h)}

	args := []eval.Value{
		&eval.StringValue{Value: "claude-sonnet-4-5"},
		&eval.ListValue{Elements: []eval.Value{
			messageRecordVal("user", "Read nda.docx", nil, ""),
		}},
		&eval.ListValue{Elements: []eval.Value{
			toolSchemaRecordVal("read_doc", "Read a doc", `{"type":"object"}`),
		}},
	}
	out, err := aiStep(ctx, args)
	if err != nil {
		t.Fatalf("unexpected Go error: %v", err)
	}
	tagged := out.(*eval.TaggedValue)
	if tagged.CtorName != "Ok" {
		t.Fatalf("expected Ok, got %s", tagged.CtorName)
	}
	stepResult := tagged.Fields[0].(*eval.RecordValue)

	if got := stepResult.Fields["finish_reason"].(*eval.StringValue).Value; got != "tool_calls" {
		t.Errorf("finish_reason = %q, want \"tool_calls\"", got)
	}
	if got := stepResult.Fields["input_tokens"].(*eval.IntValue).Value; got != 50 {
		t.Errorf("input_tokens = %d, want 50", got)
	}
	if got := stepResult.Fields["output_tokens"].(*eval.IntValue).Value; got != 25 {
		t.Errorf("output_tokens = %d, want 25", got)
	}

	// Tool calls round-tripped to AILANG list[ToolCall].
	tcList := stepResult.Fields["tool_calls"].(*eval.ListValue)
	if len(tcList.Elements) != 1 {
		t.Fatalf("tool_calls len = %d, want 1", len(tcList.Elements))
	}
	tc := tcList.Elements[0].(*eval.RecordValue)
	if tc.Fields["id"].(*eval.StringValue).Value != "call_1" ||
		tc.Fields["name"].(*eval.StringValue).Value != "read_doc" ||
		tc.Fields["arguments"].(*eval.StringValue).Value != `{"name":"nda.docx"}` {
		t.Errorf("tool_call mismatch: %+v", tc.Fields)
	}

	// Handler received the model + messages + tools verbatim.
	if h.lastModel != "claude-sonnet-4-5" {
		t.Errorf("handler.lastModel = %q", h.lastModel)
	}
	if len(h.lastMessages) != 1 || h.lastMessages[0].Role != "user" {
		t.Errorf("handler.lastMessages = %+v", h.lastMessages)
	}
	if len(h.lastTools) != 1 || h.lastTools[0].Name != "read_doc" {
		t.Errorf("handler.lastTools = %+v", h.lastTools)
	}
}

func TestAIStep_HandlerError_ReturnsErr(t *testing.T) {
	h := &fakeStepHandler{stepErr: ai.NewAIError(ai.CodeRateLimit, "throttled", true)}
	ctx := &EffContext{AI: NewAIContext(h)}

	args := []eval.Value{
		&eval.StringValue{Value: "any-model"},
		&eval.ListValue{Elements: []eval.Value{}},
		&eval.ListValue{Elements: []eval.Value{}},
	}
	out, _ := aiStep(ctx, args)
	tagged := out.(*eval.TaggedValue)
	if tagged.CtorName != "Err" {
		t.Fatalf("expected Err, got %s", tagged.CtorName)
	}
	rec := tagged.Fields[0].(*eval.RecordValue)
	if rec.Fields["code"].(*eval.StringValue).Value != ai.CodeRateLimit {
		t.Errorf("code = %q, want %q", rec.Fields["code"].(*eval.StringValue).Value, ai.CodeRateLimit)
	}
	if !rec.Fields["retryable"].(*eval.BoolValue).Value {
		t.Error("retryable = false, want true (RateLimit)")
	}
}

func TestAIStep_ToolResultMessage_RoundTrip(t *testing.T) {
	// Caller passes a Message with role="tool" + tool_call_id + content.
	// The handler should receive it intact (so Anthropic / Gemini etc.
	// can build their tool_result content blocks).
	h := &fakeStepHandler{
		stepResp: &ai.Response{Text: "ok", FinishReason: "stop"},
	}
	ctx := &EffContext{AI: NewAIContext(h)}

	args := []eval.Value{
		&eval.StringValue{Value: ""},
		&eval.ListValue{Elements: []eval.Value{
			messageRecordVal("user", "Read nda.docx", nil, ""),
			messageRecordVal("assistant", "I'll read it",
				[]eval.Value{toolCallRecordVal("call_1", "read_doc", `{"name":"nda.docx"}`)}, ""),
			messageRecordVal("tool", "<doc body>", nil, "call_1"),
		}},
		&eval.ListValue{Elements: []eval.Value{}},
	}
	_, err := aiStep(ctx, args)
	if err != nil {
		t.Fatalf("aiStep: %v", err)
	}
	if len(h.lastMessages) != 3 {
		t.Fatalf("messages = %d, want 3", len(h.lastMessages))
	}
	tool := h.lastMessages[2]
	if tool.Role != "tool" || tool.ToolCallID != "call_1" || tool.Content != "<doc body>" {
		t.Errorf("tool message round-trip failed: %+v", tool)
	}
	// And the assistant turn's tool_calls came through.
	asst := h.lastMessages[1]
	if len(asst.ToolCalls) != 1 || asst.ToolCalls[0].ID != "call_1" {
		t.Errorf("assistant tool_calls round-trip failed: %+v", asst.ToolCalls)
	}
}

func TestAIStep_NilHandler_ReturnsErr(t *testing.T) {
	ctx := &EffContext{AI: NewAIContext(nil)}
	out, _ := aiStep(ctx, []eval.Value{
		&eval.StringValue{Value: ""},
		&eval.ListValue{Elements: []eval.Value{}},
		&eval.ListValue{Elements: []eval.Value{}},
	})
	tagged := out.(*eval.TaggedValue)
	if tagged.CtorName != "Err" {
		t.Fatalf("expected Err, got %s", tagged.CtorName)
	}
	rec := tagged.Fields[0].(*eval.RecordValue)
	if code := rec.Fields["code"].(*eval.StringValue).Value; code != ai.CodeProviderNotFound {
		t.Errorf("code = %q, want %q", code, ai.CodeProviderNotFound)
	}
}

// TestAIStep_RecordsTraceEvent verifies the new ops emit AI/step events
// via the same RecordAIEffect machinery as call/callJson — so dashboards
// and replay get a uniform view across single-shot and tool-loop calls.
// Acceptance criterion from M7: ailang trace list shows ai.step events.
func TestAIStep_RecordsTraceEvent(t *testing.T) {
	h := &fakeStepHandler{
		stepResp: &ai.Response{
			Text:         "ok",
			FinishReason: "stop",
			InputTokens:  10,
			OutputTokens: 5,
		},
	}
	collector := traceCollector(t)
	ctx := NewEffContext(nil)
	ctx.Grant(NewCapability("AI"))
	ctx.AI = NewAIContext(h)
	ctx.Trace = collector

	args := []eval.Value{
		&eval.StringValue{Value: "claude-sonnet-4-5"},
		&eval.ListValue{Elements: []eval.Value{}},
		&eval.ListValue{Elements: []eval.Value{}},
	}
	if _, err := Call(ctx, "AI", "step", args); err != nil {
		t.Fatalf("Call: %v", err)
	}

	stepEvent := findAIEvent(collector, "step")
	if stepEvent == nil {
		t.Fatal("no AI/step trace event recorded")
	}
	if !contains(stepEvent.Result, "tool_calls:0") {
		t.Errorf("event Result = %q; want it to mention tool_calls:0", stepEvent.Result)
	}
	if !contains(stepEvent.Result, "finish:stop") {
		t.Errorf("event Result = %q; want it to mention finish:stop", stepEvent.Result)
	}
}

// TestAIStep_RecordsErrorEvent verifies the trace event on failure carries
// the AIError code so telemetry consumers can route on it (e.g. count
// rate-limit failures separately from auth failures).
func TestAIStep_RecordsErrorEvent(t *testing.T) {
	h := &fakeStepHandler{stepErr: ai.NewAIError(ai.CodeRateLimit, "throttled", true)}
	collector := traceCollector(t)
	ctx := NewEffContext(nil)
	ctx.Grant(NewCapability("AI"))
	ctx.AI = NewAIContext(h)
	ctx.Trace = collector

	args := []eval.Value{
		&eval.StringValue{Value: "any"},
		&eval.ListValue{Elements: []eval.Value{}},
		&eval.ListValue{Elements: []eval.Value{}},
	}
	if _, err := Call(ctx, "AI", "step", args); err != nil {
		t.Fatalf("Call: %v", err)
	}

	stepEvent := findAIEvent(collector, "step")
	if stepEvent == nil {
		t.Fatal("no AI/step trace event recorded on failure")
	}
	if !contains(stepEvent.Result, "err:RateLimit") {
		t.Errorf("event Result = %q; want \"err:RateLimit\"", stepEvent.Result)
	}
}

func TestAIStep_BadMessageType_ReturnsSchemaValidationErr(t *testing.T) {
	// Pass a non-record (string) where a Message record is expected.
	// The decoder should produce a SchemaValidation typed error.
	h := &fakeStepHandler{}
	ctx := &EffContext{AI: NewAIContext(h)}
	out, _ := aiStep(ctx, []eval.Value{
		&eval.StringValue{Value: ""},
		&eval.ListValue{Elements: []eval.Value{
			&eval.StringValue{Value: "not a record"},
		}},
		&eval.ListValue{Elements: []eval.Value{}},
	})
	tagged := out.(*eval.TaggedValue)
	if tagged.CtorName != "Err" {
		t.Fatalf("expected Err, got %s", tagged.CtorName)
	}
	rec := tagged.Fields[0].(*eval.RecordValue)
	if code := rec.Fields["code"].(*eval.StringValue).Value; code != ai.CodeSchemaValidation {
		t.Errorf("code = %q, want %q", code, ai.CodeSchemaValidation)
	}
}

// ============================================================================
// Helpers — build AILANG record values for tests
// ============================================================================

func messageRecordVal(role, content string, toolCalls []eval.Value, toolCallID string) eval.Value {
	tcList := &eval.ListValue{Elements: toolCalls}
	if toolCalls == nil {
		tcList = &eval.ListValue{Elements: []eval.Value{}}
	}
	return &eval.RecordValue{
		Fields: map[string]eval.Value{
			"role":         &eval.StringValue{Value: role},
			"content":      &eval.StringValue{Value: content},
			"tool_calls":   tcList,
			"tool_call_id": &eval.StringValue{Value: toolCallID},
		},
	}
}

func toolCallRecordVal(id, name, args string) eval.Value {
	return &eval.RecordValue{
		Fields: map[string]eval.Value{
			"id":        &eval.StringValue{Value: id},
			"name":      &eval.StringValue{Value: name},
			"arguments": &eval.StringValue{Value: args},
		},
	}
}

func toolSchemaRecordVal(name, desc, params string) eval.Value {
	return &eval.RecordValue{
		Fields: map[string]eval.Value{
			"name":        &eval.StringValue{Value: name},
			"description": &eval.StringValue{Value: desc},
			"parameters":  &eval.StringValue{Value: params},
		},
	}
}

// traceCollector returns a fresh trace collector for one test.
func traceCollector(_ *testing.T) *trace.Collector {
	return trace.NewCollector()
}

// findAIEvent walks the collector's events and returns the first AI event
// whose OpName matches op (e.g. "step", "callResult"). Returns nil if not
// found, so callers can write `if ev == nil { t.Fatal(...) }`.
func findAIEvent(c *trace.Collector, op string) *trace.EffectEvent {
	for _, ev := range c.Events() {
		if ev.Effect != nil && ev.Effect.EffectName == "AI" && ev.Effect.OpName == op {
			return ev.Effect
		}
	}
	return nil
}
