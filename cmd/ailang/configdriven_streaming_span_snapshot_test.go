package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/ai"
	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/trace"
)

// TestStreamingAISpan_SameShapeAsNonStreaming verifies M-AI-STREAMING-HELPER
// M1 acceptance criterion #6: "Snapshot test: streaming AI span attributes
// match non-streaming AI span attributes (modulo stream_id)".
//
// The contract is that both `_ai_call` and `_ai_stream_call` go through
// `ctx.RecordAIEffect(...)`, producing trace events with the same shape:
//   - EffectName: "AI"
//   - OpName: differs ("call" vs "streamCall") — that's the modulo
//   - Args: present (3 strings vs 1 string — also modulo)
//   - Same TraceEvent envelope (Version, Event="effect", TraceID, SpanID, etc.)
//
// This is the load-bearing test for design-doc A4 (Explicit Authority) and
// A9 (Cost Visibility) compliance — if streaming bypassed RecordAIEffect,
// observability for streaming AI calls would silently degrade.
//
// The test mocks the AI handler interface to avoid setting up full provider
// HTTP clients; the recording call is the only thing being verified.
func TestStreamingAISpan_SameShapeAsNonStreaming(t *testing.T) {
	// Capture trace events from a non-streaming AI call simulation.
	nonStreamingEvent := captureAIEffectEvent(t, func(ctx *effects.EffContext) {
		// Simulate what aiCall in internal/effects/ai.go does at line 243.
		ctx.RecordAIEffect("call",
			[]string{"Hello, AI"},
			"Hello, human!",
			nil, // no routing for direct providers
		)
	})

	// Capture trace events from a streaming AI call simulation.
	streamingEvent := captureAIEffectEvent(t, func(ctx *effects.EffContext) {
		// Simulate what aiStreamCall in cmd/ailang/configdriven_streaming.go does.
		ctx.RecordAIEffect("streamCall",
			[]string{"openai", "gpt-4o", `[{"role":"user","content":"Hello"}]`},
			"<streaming>",
			nil, // routing not applicable to config-driven providers (D11)
		)
	})

	// Snapshot: both events must use the same TraceEvent envelope.
	if nonStreamingEvent.Version != streamingEvent.Version {
		t.Errorf("Version mismatch: non-streaming=%q streaming=%q",
			nonStreamingEvent.Version, streamingEvent.Version)
	}
	if nonStreamingEvent.Event != streamingEvent.Event {
		t.Errorf("Event type mismatch: non-streaming=%q streaming=%q",
			nonStreamingEvent.Event, streamingEvent.Event)
	}

	// Both must populate the Effect field with the same EffectName.
	if nonStreamingEvent.Effect == nil {
		t.Fatal("non-streaming event has no Effect payload")
	}
	if streamingEvent.Effect == nil {
		t.Fatal("streaming event has no Effect payload")
	}
	if nonStreamingEvent.Effect.EffectName != "AI" {
		t.Errorf("non-streaming EffectName = %q, want AI", nonStreamingEvent.Effect.EffectName)
	}
	if streamingEvent.Effect.EffectName != "AI" {
		t.Errorf("streaming EffectName = %q, want AI", streamingEvent.Effect.EffectName)
	}

	// OpName MUST differ ("call" vs "streamCall") — that's the documented modulo.
	if nonStreamingEvent.Effect.OpName == streamingEvent.Effect.OpName {
		t.Errorf("OpName should differ between streaming and non-streaming: both = %q",
			nonStreamingEvent.Effect.OpName)
	}
	if nonStreamingEvent.Effect.OpName != "call" {
		t.Errorf("non-streaming OpName = %q, want call", nonStreamingEvent.Effect.OpName)
	}
	if streamingEvent.Effect.OpName != "streamCall" {
		t.Errorf("streaming OpName = %q, want streamCall", streamingEvent.Effect.OpName)
	}

	// Args populated for both (count differs but presence is shared).
	if len(nonStreamingEvent.Effect.Args) == 0 {
		t.Errorf("non-streaming Args empty; should record at least the prompt")
	}
	if len(streamingEvent.Effect.Args) == 0 {
		t.Errorf("streaming Args empty; should record provider/model/messages")
	}

	// Result populated for both (string contents differ; presence is shared).
	if nonStreamingEvent.Effect.Result == "" {
		t.Errorf("non-streaming Result empty")
	}
	if streamingEvent.Effect.Result == "" {
		t.Errorf("streaming Result empty (expected '<streaming>' marker)")
	}

	// Route field is nil for both — config-driven providers reject
	// AIRoutingPolicy (D11), and the simulated non-streaming call also has
	// no routing. If a future refactor adds routing to config-driven
	// streaming, this assertion will catch it and the design doc must be
	// updated first.
	if nonStreamingEvent.Effect.Route != nil {
		t.Errorf("non-streaming Route should be nil for direct providers, got %+v", nonStreamingEvent.Effect.Route)
	}
	if streamingEvent.Effect.Route != nil {
		t.Errorf("streaming Route should be nil for config-driven providers (D11), got %+v", streamingEvent.Effect.Route)
	}

	// TraceID format identical (both 32-hex per W3C trace-context).
	if len(nonStreamingEvent.TraceID) != 32 || len(streamingEvent.TraceID) != 32 {
		t.Errorf("TraceID should be 32-hex chars, got non-streaming=%d streaming=%d",
			len(nonStreamingEvent.TraceID), len(streamingEvent.TraceID))
	}

	// Both events should have a non-zero TimestampNS.
	if nonStreamingEvent.TimestampNS == 0 || streamingEvent.TimestampNS == 0 {
		t.Errorf("TimestampNS missing: non-streaming=%d streaming=%d",
			nonStreamingEvent.TimestampNS, streamingEvent.TimestampNS)
	}
}

// captureAIEffectEvent runs the supplied callback against a fresh EffContext
// with an enabled trace.Collector, and returns the single recorded event.
// Fails the test if 0 or >1 events were recorded.
func captureAIEffectEvent(t *testing.T, doCall func(ctx *effects.EffContext)) trace.TraceEvent {
	t.Helper()
	ctx := effects.NewEffContext(nil)
	ctx.Trace = trace.NewCollector()

	doCall(ctx)

	events := ctx.Trace.Events()
	if len(events) != 1 {
		t.Fatalf("expected exactly 1 trace event, got %d", len(events))
	}
	return events[0]
}

// TestStreamingAISpan_RecordedFromAIStreamCallEndToEnd verifies that the real
// aiStreamCall function (not just RecordAIEffect directly) emits the trace
// span when invoked end-to-end against a mock server. Belt-and-suspenders
// confirmation that the dispatch path actually reaches the recording call.
func TestStreamingAISpan_RecordedFromAIStreamCallEndToEnd(t *testing.T) {
	ai.GlobalProviderRegistry.Reset()
	defer ai.GlobalProviderRegistry.Reset()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"x\"}}]}\n\ndata: [DONE]\n\n"))
		flusher.Flush()
	}))
	defer server.Close()

	registerOpenAITestProvider(t, "span-end2end-test", server.URL, "")

	ctx := streamTestContext(t)
	ctx.Trace = trace.NewCollector()
	ctx.Stream.IdleTimeout = 1 * time.Second
	ctx.Stream.MaxDuration = 2 * time.Second

	_, err := aiStreamCall(ctx, []eval.Value{
		&eval.StringValue{Value: "span-end2end-test"},
		&eval.StringValue{Value: "any-model"},
		&eval.StringValue{Value: `[{"role":"user","content":"hi"}]`},
	})
	if err != nil {
		t.Fatalf("aiStreamCall error: %v", err)
	}

	// Find the AI streamCall trace event.
	events := ctx.Trace.Events()
	var found *trace.TraceEvent
	for i := range events {
		if events[i].Effect != nil && events[i].Effect.EffectName == "AI" && events[i].Effect.OpName == "streamCall" {
			found = &events[i]
			break
		}
	}
	if found == nil {
		t.Fatalf("expected AI/streamCall event in trace, got %d events", len(events))
	}
	if len(found.Effect.Args) != 3 {
		t.Errorf("expected 3 args (provider, model, messages_json), got %d", len(found.Effect.Args))
	}
	if found.Effect.Result != "<streaming>" {
		t.Errorf("Result = %q, want <streaming> marker", found.Effect.Result)
	}
}
