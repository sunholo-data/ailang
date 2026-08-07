package effects

import (
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/ai"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/trace"
)

type scriptedStreamHandler struct {
	fakeStepHandler
	chunks []ai.StreamChunk
	resp   *ai.Response
	err    error
	route  *trace.ResolvedRoute
}

func (h *scriptedStreamHandler) StepWithStream(_ string, _ []ai.Message, _ []ai.ToolSchema, _ []ai.CacheBreakpoint, onChunk func(ai.StreamChunk)) (*ai.Response, error) {
	for _, chunk := range h.chunks {
		onChunk(chunk)
	}
	return h.resp, h.err
}

func (h *scriptedStreamHandler) LastRoutingMetadata() *trace.ResolvedRoute { return h.route }

func streamTestContext(h AIHandler, captured *[]eval.Value) *EffContext {
	return &EffContext{
		AI: NewAIContext(h),
		FnCaller: func(_ eval.Value, arg eval.Value) (eval.Value, error) {
			*captured = append(*captured, arg)
			return &eval.UnitValue{}, nil
		},
		Trace: trace.NewCollector(),
	}
}

func streamErrRecord(t *testing.T, outcome *eval.TaggedValue) *eval.RecordValue {
	t.Helper()
	if outcome.CtorName != "Err" {
		t.Fatalf("outcome = %s, want Err", outcome.CtorName)
	}
	return outcome.Fields[0].(*eval.RecordValue)
}

func streamSuccessResponse() *ai.Response {
	return &ai.Response{Text: "done", FinishReason: "stop", InputTokens: 7, OutputTokens: 3}
}

func terminalAIEvent(t *testing.T, ctx *EffContext) *trace.EffectEvent {
	t.Helper()
	events := ctx.Trace.Events()
	for i := len(events) - 1; i >= 0; i-- {
		if events[i].Effect != nil && !strings.HasSuffix(events[i].Effect.OpName, ".callback") {
			return events[i].Effect
		}
	}
	t.Fatal("terminal AI trace event not found")
	return nil
}

func TestAIStreamCoreMatrix(t *testing.T) {
	rows := []struct {
		name string
		run  func(*testing.T)
	}{
		{"01_argument_type_decode_failure_parity", testStreamArgumentParity},
		{"02_handler_and_fncaller_typed_errors", testStreamMissingIntegration},
		{"03_callback_failure_is_fail_soft", testStreamCallbackFailure},
		{"04_all_chunk_variants_full_usage_order", testStreamChunkVariants},
		{"05_unencodable_first_and_middle", testStreamUnencodable},
		{"06_independent_drain_budgets", testStreamDrainBudgets},
		{"07_latched_error_not_overwritten", testStreamLatchedError},
		{"08_empty_stream_success_and_error", testStreamEmpty},
		{"09_stable_order_identity_no_duplicates", testStreamIdentity},
		{"10_capability_and_budget_layer_parity", testStreamCapabilityContract},
		{"11_trace_contract", testStreamTrace},
		{"12_registry_public_type_and_metadata", testStreamSurfaceContract},
		{"13_nested_recorded_stream_vm_shape", testStreamNestedValueShape},
		{"14_adr009_ordering_gate", testStreamADR009Ordering},
	}
	if len(rows) != 14 {
		t.Fatalf("matrix rows = %d, want 14", len(rows))
	}
	for _, row := range rows {
		t.Run(row.name, row.run)
	}
}

func testStreamArgumentParity(t *testing.T) {
	cases := []struct {
		name string
		args []eval.Value
	}{
		{"arity", nil},
		{"model", []eval.Value{&eval.IntValue{Value: 1}, &eval.ListValue{}, &eval.ListValue{}, &eval.ListValue{}, &eval.UnitValue{}}},
		{"messages", []eval.Value{&eval.StringValue{}, &eval.IntValue{}, &eval.ListValue{}, &eval.ListValue{}, &eval.UnitValue{}}},
		{"tools", []eval.Value{&eval.StringValue{}, &eval.ListValue{}, &eval.IntValue{}, &eval.ListValue{}, &eval.UnitValue{}}},
		{"breakpoints", []eval.Value{&eval.StringValue{}, &eval.ListValue{}, &eval.ListValue{}, &eval.IntValue{}, &eval.UnitValue{}}},
		{"callback", []eval.Value{&eval.StringValue{}, &eval.ListValue{}, &eval.ListValue{}, &eval.ListValue{}, nil}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for _, call := range []func(*EffContext, []eval.Value) (eval.Value, error){aiStepWithStream, aiStepWithStreamRecorded} {
				if _, err := call(&EffContext{}, tc.args); err == nil || !strings.Contains(err.Error(), "E_AI_TYPE_ERROR") {
					t.Fatalf("error = %v, want E_AI_TYPE_ERROR", err)
				}
			}
		})
	}
	bad := recordedArgs()
	bad[1] = &eval.ListValue{Elements: []eval.Value{&eval.IntValue{Value: 1}}}
	for _, call := range []func(*EffContext, []eval.Value) (eval.Value, error){aiStepWithStream, aiStepWithStreamRecorded} {
		out, err := call(&EffContext{AI: NewAIContext(&scriptedStreamHandler{resp: streamSuccessResponse()}), FnCaller: captureFnCaller(&[]eval.Value{})}, bad)
		if err != nil {
			t.Fatal(err)
		}
		if _, ok := out.(*eval.RecordValue); ok {
			_, outcome := splitRecorded(t, out)
			streamErrRecord(t, outcome)
		} else if out.(*eval.TaggedValue).CtorName != "Err" {
			t.Fatal("legacy decode failure did not return Err")
		}
	}
}

func testStreamMissingIntegration(t *testing.T) {
	legacy, _ := aiStepWithStream(&EffContext{}, recordedArgs())
	if streamErrRecord(t, legacy.(*eval.TaggedValue)).Fields["code"].(*eval.StringValue).Value != ai.CodeProviderNotFound {
		t.Fatal("legacy missing handler code mismatch")
	}
	recorded, _ := aiStepWithStreamRecorded(&EffContext{}, recordedArgs())
	chunks, outcome := splitRecorded(t, recorded)
	if len(chunks) != 0 || streamErrRecord(t, outcome).Fields["code"].(*eval.StringValue).Value != ai.CodeProviderNotFound {
		t.Fatal("recorded missing handler contract mismatch")
	}
	ctx := &EffContext{AI: NewAIContext(&scriptedStreamHandler{})}
	recorded, _ = aiStepWithStreamRecorded(ctx, recordedArgs())
	chunks, outcome = splitRecorded(t, recorded)
	if len(chunks) != 0 || streamErrRecord(t, outcome).Fields["code"].(*eval.StringValue).Value != ai.CodeInternal {
		t.Fatal("recorded missing FnCaller contract mismatch")
	}
}

func testStreamCallbackFailure(t *testing.T) {
	h := &scriptedStreamHandler{chunks: []ai.StreamChunk{ai.StreamContentDelta{Text: "a"}, ai.StreamContentDelta{Text: "b"}}, resp: streamSuccessResponse()}
	ctx := &EffContext{AI: NewAIContext(h), Trace: trace.NewCollector(), FnCaller: func(eval.Value, eval.Value) (eval.Value, error) { return nil, errors.New("callback broke") }}
	out, err := aiStepWithStreamRecorded(ctx, recordedArgs())
	if err != nil {
		t.Fatal(err)
	}
	chunks, outcome := splitRecorded(t, out)
	if len(chunks) != 2 || outcome.CtorName != "Ok" || len(ctx.Trace.Events()) != 3 {
		t.Fatalf("chunks=%d outcome=%s events=%d", len(chunks), outcome.CtorName, len(ctx.Trace.Events()))
	}
}

func testStreamChunkVariants(t *testing.T) {
	usage := ai.StreamUsage{InputTokens: 1, OutputTokens: 2, CacheReadInputTokens: 3, CacheCreationInputTokens: 4}
	h := &scriptedStreamHandler{chunks: []ai.StreamChunk{ai.StreamContentDelta{Text: "c"}, ai.StreamThinkingDelta{Text: "r"}, usage}, resp: streamSuccessResponse()}
	var captured []eval.Value
	out, _ := aiStepWithStreamRecorded(streamTestContext(h, &captured), recordedArgs())
	chunks, _ := splitRecorded(t, out)
	wantNames := []string{"ContentDelta", "ThinkingDelta", "Usage"}
	for i, want := range wantNames {
		if chunks[i].(*eval.TaggedValue).CtorName != want {
			t.Fatalf("chunk %d ctor = %s", i, chunks[i].(*eval.TaggedValue).CtorName)
		}
	}
	usageRec := chunks[2].(*eval.TaggedValue).Fields[0].(*eval.RecordValue)
	for field, want := range map[string]int{"input_tokens": 1, "output_tokens": 2, "cache_read_input_tokens": 3, "cache_creation_input_tokens": 4} {
		if usageRec.Fields[field].(*eval.IntValue).Value != want {
			t.Errorf("%s mismatch", field)
		}
	}
}

func testStreamUnencodable(t *testing.T) {
	for _, chunks := range [][]ai.StreamChunk{{nil, ai.StreamContentDelta{Text: "later"}}, {ai.StreamContentDelta{Text: "prefix"}, nil, ai.StreamContentDelta{Text: "later"}}} {
		h := &scriptedStreamHandler{chunks: chunks, resp: streamSuccessResponse()}
		var captured []eval.Value
		out, _ := aiStepWithStreamRecorded(streamTestContext(h, &captured), recordedArgs())
		returned, outcome := splitRecorded(t, out)
		if len(returned) != len(captured) || len(returned) != len(chunks)-2 {
			t.Fatalf("returned=%d captured=%d", len(returned), len(captured))
		}
		msg := streamErrRecord(t, outcome).Fields["message"].(*eval.StringValue).Value
		if !strings.HasPrefix(msg, unencodableStreamChunkErrorPrefix) || !strings.Contains(msg, "incomplete prefix") {
			t.Fatalf("message = %q", msg)
		}
	}
}

func testStreamDrainBudgets(t *testing.T) {
	// The budget values are public contract (sprint plan M3): a silent change
	// must fail here, not ship. Self-referential feeding alone cannot catch a
	// budget regression (a 256->2 mutation survived it — iter-135).
	if recordedDrainMaxChunks != 256 || recordedDrainMaxBytes != 1<<20 {
		t.Fatalf("drain budget contract changed: chunks=%d bytes=%d", recordedDrainMaxChunks, recordedDrainMaxBytes)
	}
	chunkBudget := make([]ai.StreamChunk, recordedDrainMaxChunks+10)
	chunkBudget[0] = nil
	for i := 1; i < len(chunkBudget); i++ {
		chunkBudget[i] = ai.StreamUsage{}
	}
	byteBudget := []ai.StreamChunk{nil, ai.StreamContentDelta{Text: strings.Repeat("x", recordedDrainMaxBytes/2)}, ai.StreamThinkingDelta{Text: strings.Repeat("y", recordedDrainMaxBytes/2)}}
	for _, chunks := range [][]ai.StreamChunk{chunkBudget, byteBudget} {
		ctx := streamTestContext(&scriptedStreamHandler{chunks: chunks, resp: streamSuccessResponse()}, &[]eval.Value{})
		out, _ := aiStepWithStreamRecorded(ctx, recordedArgs())
		_, outcome := splitRecorded(t, out)
		streamErrRecord(t, outcome)
		if event := terminalAIEvent(t, ctx); !strings.Contains(event.Args[1], "drain_exhausted:true") {
			t.Fatalf("trace args = %q", event.Args)
		}
	}
	// Under-budget control: a drain that stays inside both budgets must NOT
	// report exhaustion (proves the exhaustion assertions above are informative).
	under := []ai.StreamChunk{nil, ai.StreamUsage{}, ai.StreamUsage{}}
	ctx := streamTestContext(&scriptedStreamHandler{chunks: under, resp: streamSuccessResponse()}, &[]eval.Value{})
	out, _ := aiStepWithStreamRecorded(ctx, recordedArgs())
	_, outcome := splitRecorded(t, out)
	streamErrRecord(t, outcome)
	if event := terminalAIEvent(t, ctx); strings.Contains(event.Args[1], "drain_exhausted:true") {
		t.Fatalf("under-budget drain reported exhaustion: %q", event.Args)
	}
}

func testStreamLatchedError(t *testing.T) {
	for _, providerErr := range []error{nil, ai.NewAIError(ai.CodeAuthFailed, "later", false)} {
		h := &scriptedStreamHandler{chunks: []ai.StreamChunk{nil}, resp: streamSuccessResponse(), err: providerErr}
		out, _ := aiStepWithStreamRecorded(streamTestContext(h, &[]eval.Value{}), recordedArgs())
		_, outcome := splitRecorded(t, out)
		rec := streamErrRecord(t, outcome)
		if rec.Fields["code"].(*eval.StringValue).Value != ai.CodeInternal || !strings.HasPrefix(rec.Fields["message"].(*eval.StringValue).Value, unencodableStreamChunkErrorPrefix) {
			t.Fatal("latched representation error was overwritten")
		}
	}
}

func testStreamEmpty(t *testing.T) {
	for _, providerErr := range []error{nil, errors.New("empty failed")} {
		h := &scriptedStreamHandler{resp: streamSuccessResponse(), err: providerErr}
		out, _ := aiStepWithStreamRecorded(streamTestContext(h, &[]eval.Value{}), recordedArgs())
		chunks, outcome := splitRecorded(t, out)
		if len(chunks) != 0 || (providerErr == nil) != (outcome.CtorName == "Ok") {
			t.Fatalf("chunks=%d outcome=%s", len(chunks), outcome.CtorName)
		}
	}
}

func testStreamIdentity(t *testing.T) {
	h := &scriptedStreamHandler{chunks: []ai.StreamChunk{ai.StreamContentDelta{Text: "1"}, ai.StreamThinkingDelta{Text: "2"}, ai.StreamUsage{}}, resp: streamSuccessResponse()}
	var captured []eval.Value
	out, _ := aiStepWithStreamRecorded(streamTestContext(h, &captured), recordedArgs())
	returned, _ := splitRecorded(t, out)
	for i := range returned {
		if returned[i] != captured[i] {
			t.Fatalf("chunk %d was encoded twice", i)
		}
	}
}

func testStreamCapabilityContract(t *testing.T) {
	source, err := os.ReadFile("../builtins/ai_step.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(source)
	for _, impl := range []string{"aiStepWithStreamImpl", "aiStepWithStreamRecordedImpl"} {
		start := strings.Index(text, "func "+impl)
		if start < 0 || !strings.Contains(text[start:start+220], `RequireCapWithBudget("AI", "")`) {
			t.Fatalf("%s does not share AI capability/budget gate", impl)
		}
	}
}

func testStreamTrace(t *testing.T) {
	ctx := streamTestContext(&scriptedStreamHandler{chunks: []ai.StreamChunk{ai.StreamContentDelta{Text: "ok"}, nil, ai.StreamUsage{}}, resp: streamSuccessResponse()}, &[]eval.Value{})
	out, _ := aiStepWithStreamRecorded(ctx, recordedArgs())
	returned, _ := splitRecorded(t, out)
	event := terminalAIEvent(t, ctx)
	if event.OpName != "stepWithStreamRecorded" || event.Result != "err:Internal" {
		t.Fatalf("trace op/result = %s/%s", event.OpName, event.Result)
	}
	for _, part := range []string{"provider_chunks:3", "delivered_chunks:1", "fatal_provider_index:2"} {
		if !strings.Contains(event.Args[1], part) {
			t.Errorf("missing %s in %q", part, event.Args[1])
		}
	}
	if len(returned) != 1 {
		t.Fatal("delivered_chunks != len(recorded)")
	}
}

func testStreamSurfaceContract(t *testing.T) {
	for path, needles := range map[string][]string{
		"../builtins/ai_step.go": {"_ai_step_with_stream_recorded", `Since:     "v0.32.0"`, "StabilityExperimental", "makeAIStepWithStreamRecordedType"},
		"../../std/ai.ail":       {"export type RecordedStream", "export func stepWithStreamRecorded"},
	} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		for _, needle := range needles {
			if !strings.Contains(string(data), needle) {
				t.Errorf("%s missing %q", path, needle)
			}
		}
	}
}

func testStreamNestedValueShape(t *testing.T) {
	h := &scriptedStreamHandler{chunks: []ai.StreamChunk{ai.StreamContentDelta{Text: "nested"}}, resp: streamSuccessResponse()}
	out, _ := aiStepWithStreamRecorded(streamTestContext(h, &[]eval.Value{}), recordedArgs())
	rec := out.(*eval.RecordValue)
	if _, ok := rec.Fields["chunks"].(*eval.ListValue); !ok {
		t.Fatal("chunks is not VM-convertible list shape")
	}
	if _, ok := rec.Fields["outcome"].(*eval.TaggedValue); !ok {
		t.Fatal("outcome is not VM-convertible ADT shape")
	}
}

func testStreamADR009Ordering(t *testing.T) {
	for _, providerErr := range []error{nil, errors.New("partial terminal error")} {
		h := &scriptedStreamHandler{chunks: []ai.StreamChunk{ai.StreamContentDelta{Text: "a"}, ai.StreamThinkingDelta{Text: "b"}, ai.StreamUsage{}}, resp: streamSuccessResponse(), err: providerErr}
		var projected []eval.Value
		out, _ := aiStepWithStreamRecorded(streamTestContext(h, &projected), recordedArgs())
		returned, outcome := splitRecorded(t, out)
		if !reflect.DeepEqual(returned, projected) || len(returned) != 3 {
			t.Fatal("returned log differs from immediate ordered projection")
		}
		if (providerErr == nil) != (outcome.CtorName == "Ok") {
			t.Fatalf("outcome = %s", outcome.CtorName)
		}
	}
}
