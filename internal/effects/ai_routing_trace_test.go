package effects

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/ai"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/trace"
)

// routingStubHandler implements both AIHandler and AIHandlerWithRouting,
// returning a fixed response and a fixed routing metadata. Used to verify
// that AI effect ops attach routing metadata to trace events.
type routingStubHandler struct {
	response string
	route    *trace.ResolvedRoute
}

func (h *routingStubHandler) Call(_ string) (string, error) {
	return h.response, nil
}

func (h *routingStubHandler) CallJson(_, _ string) (string, error) {
	return h.response, nil
}

func (h *routingStubHandler) CallImage(_, outputPath, _ string) (string, error) {
	return outputPath, nil
}

func (h *routingStubHandler) CallImageBase64(_, _ string) (string, error) {
	return `{"base64":"","mime_type":"image/png"}`, nil
}

func (h *routingStubHandler) LastRoutingMetadata() *trace.ResolvedRoute {
	return h.route
}

// Step satisfies the M-AI-TOOL-LOOP extension to AIHandler. These tests
// only exercise Call/CallJson, so the stub returns a fixed response.
func (h *routingStubHandler) Step(model string, _ []ai.Message, _ []ai.ToolSchema) (*ai.Response, error) {
	return &ai.Response{Text: h.response, FinishReason: "stop", Model: model}, nil
}
func (h *routingStubHandler) StepWithCache(model string, m []ai.Message, t []ai.ToolSchema, _ []ai.CacheBreakpoint) (*ai.Response, error) {
	return h.Step(model, m, t)
}
func (h *routingStubHandler) StepWithStream(model string, m []ai.Message, t []ai.ToolSchema, _ []ai.CacheBreakpoint, _ func(ai.StreamChunk)) (*ai.Response, error) {
	return h.Step(model, m, t)
}

// TestAICall_RecordsTraceEventWithRoute verifies that aiCall emits an
// effect trace event tagged with the routing metadata returned by an
// AIHandlerWithRouting handler.
func TestAICall_RecordsTraceEventWithRoute(t *testing.T) {
	collector := trace.NewCollector()
	expectedRoute := &trace.ResolvedRoute{
		RequestedModel:   "openrouter/auto",
		ResolvedModel:    "anthropic/claude-sonnet-4.5",
		ResolvedProvider: "Anthropic",
		PromptTokens:     50,
		CostUSD:          "0.0002",
	}
	handler := &routingStubHandler{
		response: "hello world",
		route:    expectedRoute,
	}

	ctx := NewEffContext(nil)
	ctx.Grant(NewCapability("AI"))
	ctx.AI = NewAIContext(handler)
	ctx.Trace = collector

	args := []eval.Value{&eval.StringValue{Value: "summarize this please"}}
	result, err := Call(ctx, "AI", "call", args)
	if err != nil {
		t.Fatalf("Call() error = %v", err)
	}
	if sv, ok := result.(*eval.StringValue); !ok || sv.Value != "hello world" {
		t.Errorf("result = %v, want StringValue(hello world)", result)
	}

	events := collector.Events()
	if len(events) == 0 {
		t.Fatal("expected at least one trace event, got none")
	}

	var aiEvent *trace.EffectEvent
	for i := range events {
		if events[i].Effect != nil && events[i].Effect.EffectName == "AI" {
			aiEvent = events[i].Effect
			break
		}
	}
	if aiEvent == nil {
		t.Fatal("no AI effect event recorded")
	}
	if aiEvent.OpName != "call" {
		t.Errorf("OpName = %q, want call", aiEvent.OpName)
	}
	if aiEvent.Result != "hello world" {
		t.Errorf("Result = %q", aiEvent.Result)
	}
	if aiEvent.Route == nil {
		t.Fatal("Route is nil — handler routing metadata was not recorded")
	}
	if aiEvent.Route.ResolvedProvider != "Anthropic" {
		t.Errorf("Route.ResolvedProvider = %q", aiEvent.Route.ResolvedProvider)
	}
	if aiEvent.Route.RequestedModel != "openrouter/auto" {
		t.Errorf("Route.RequestedModel = %q", aiEvent.Route.RequestedModel)
	}
}

// TestAICall_RecordsTraceEventWithoutRoute verifies that AI ops backed by
// a plain (non-routing) handler emit a trace event with Route == nil — the
// stub handler doesn't implement AIHandlerWithRouting, so no routing
// metadata should be fabricated.
func TestAICall_RecordsTraceEventWithoutRoute(t *testing.T) {
	collector := trace.NewCollector()
	ctx := NewEffContext(nil)
	ctx.Grant(NewCapability("AI"))
	ctx.AI = NewAIContext(NewStubAIHandler())
	ctx.Trace = collector

	args := []eval.Value{&eval.StringValue{Value: "decide"}}
	if _, err := Call(ctx, "AI", "call", args); err != nil {
		t.Fatalf("Call() error = %v", err)
	}

	events := collector.Events()
	var aiEvent *trace.EffectEvent
	for i := range events {
		if events[i].Effect != nil && events[i].Effect.EffectName == "AI" {
			aiEvent = events[i].Effect
			break
		}
	}
	if aiEvent == nil {
		t.Fatal("no AI effect event recorded")
	}
	if aiEvent.Route != nil {
		t.Errorf("Route = %+v, want nil for non-routing handler", aiEvent.Route)
	}
}

// TestAICall_TruncatesLongArgsAndResult verifies that long AI inputs and
// outputs are clipped before being placed in the trace event so a 100k
// prompt doesn't blow up the trace stream.
func TestAICall_TruncatesLongArgsAndResult(t *testing.T) {
	long := strings.Repeat("a", 1000)
	collector := trace.NewCollector()
	handler := &routingStubHandler{response: long}

	ctx := NewEffContext(nil)
	ctx.Grant(NewCapability("AI"))
	ctx.AI = NewAIContext(handler)
	ctx.Trace = collector

	args := []eval.Value{&eval.StringValue{Value: long}}
	if _, err := Call(ctx, "AI", "call", args); err != nil {
		t.Fatalf("Call() error = %v", err)
	}

	events := collector.Events()
	var aiEvent *trace.EffectEvent
	for i := range events {
		if events[i].Effect != nil && events[i].Effect.EffectName == "AI" {
			aiEvent = events[i].Effect
			break
		}
	}
	if aiEvent == nil {
		t.Fatal("no AI effect event recorded")
	}
	// Args/result should be truncated below the original 1000-char length.
	for _, a := range aiEvent.Args {
		if len(a) > traceArgMaxLen+len("...[truncated]") {
			t.Errorf("trace arg length %d exceeds max + ellipsis", len(a))
		}
	}
	if len(aiEvent.Result) > traceArgMaxLen+len("...[truncated]") {
		t.Errorf("trace result length %d exceeds max + ellipsis", len(aiEvent.Result))
	}
}
