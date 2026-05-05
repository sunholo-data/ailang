package main

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/ai"
	"github.com/sunholo-data/ailang/internal/ai/configdriven"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/pkg"
	"github.com/sunholo-data/ailang/internal/trace"
)

// M-AI-CALL-STREAM-HELPER (v0.15.1) integration tests.
//
// These tests cover the synchronous accumulator wrapper that turns a
// streamCall + event-loop into a single Result[string, AIError]. They
// share the streamTestContext + registerOpenAITestProvider helpers
// from configdriven_streaming_test.go.

// TestAICallStream_OpenAIShape_AccumulatesContent: end-to-end happy path.
//
// Sends two delta events ("Hello" + " world") followed by [DONE] and
// verifies the accumulator returns Ok("Hello world").
func TestAICallStream_OpenAIShape_AccumulatesContent(t *testing.T) {
	ai.GlobalProviderRegistry.Reset()
	defer ai.GlobalProviderRegistry.Reset()

	server := newOpenAISSEServer(t, nil)
	defer server.Close()

	registerOpenAITestProvider(t, "callstream-openai", server.URL, "")

	ctx := streamTestContext(t)
	result, err := aiCallStream(ctx, []eval.Value{
		&eval.StringValue{Value: "callstream-openai"},
		&eval.StringValue{Value: "gpt-4o"},
		&eval.StringValue{Value: `[{"role":"user","content":"hi"}]`},
	})
	if err != nil {
		t.Fatalf("aiCallStream returned Go error: %v", err)
	}
	tagged, ok := result.(*eval.TaggedValue)
	if !ok || tagged.CtorName != "Ok" {
		t.Fatalf("expected Ok(string), got %+v", result)
	}
	str, ok := tagged.Fields[0].(*eval.StringValue)
	if !ok {
		t.Fatalf("expected StringValue inside Ok, got %T", tagged.Fields[0])
	}
	if str.Value != "Hello world" {
		t.Errorf("accumulated string = %q, want %q", str.Value, "Hello world")
	}
}

// newAnthropicSSEServer mocks an Anthropic-shape stream that terminates on
// `message_stop` (no [DONE] sentinel). Two content_block_delta events plus
// the terminator.
func newAnthropicSSEServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		flusher, _ := w.(http.Flusher)

		// message_start (carries no content)
		_, _ = fmt.Fprintln(w, "event: message_start")
		_, _ = fmt.Fprintln(w, `data: {"type":"message_start","message":{"id":"msg_1"}}`)
		_, _ = fmt.Fprintln(w, "")
		flusher.Flush()

		// content_block_delta x2
		_, _ = fmt.Fprintln(w, "event: content_block_delta")
		_, _ = fmt.Fprintln(w, `data: {"type":"content_block_delta","delta":{"text":"Hi"}}`)
		_, _ = fmt.Fprintln(w, "")
		flusher.Flush()

		_, _ = fmt.Fprintln(w, "event: content_block_delta")
		_, _ = fmt.Fprintln(w, `data: {"type":"content_block_delta","delta":{"text":" there"}}`)
		_, _ = fmt.Fprintln(w, "")
		flusher.Flush()

		// message_stop terminates the stream (no [DONE])
		_, _ = fmt.Fprintln(w, "event: message_stop")
		_, _ = fmt.Fprintln(w, `data: {"type":"message_stop"}`)
		_, _ = fmt.Fprintln(w, "")
		flusher.Flush()
	}))
}

// registerAnthropicTestProvider creates a config-driven provider that
// matches the Anthropic shape: no done_sentinel; uses message_stop event
// type to terminate; delta_path traverses delta.text.
func registerAnthropicTestProvider(t *testing.T, name, url string) {
	t.Helper()
	spec := &pkg.AIProviderSpec{
		SchemaVersion: 1,
		Name:          name,
		Endpoint:      url,
		RequestShape:  "anthropic_messages",
		ResponsePath:  "$.content[0].text",
		Auth:          pkg.AIProviderAuth{Type: "none"},
		Capabilities:  pkg.AIProviderCapabilities{Streaming: true},
		Streaming: pkg.AIProviderStreaming{
			Enabled:   true,
			DeltaPath: "$.delta.text",
			// No DoneSentinel — termination relies on message_stop event.
		},
	}
	if err := ai.GlobalProviderRegistry.Register(name, configdriven.New(spec), "test://"+name); err != nil {
		t.Fatalf("register provider: %v", err)
	}
}

// TestAICallStream_AnthropicShape_TerminatesOnMessageStop: Anthropic-shape
// streams have no [DONE] sentinel — termination must come from the
// message_stop event type. Verifies the accumulator returns the deltas
// concatenated and stops cleanly.
func TestAICallStream_AnthropicShape_TerminatesOnMessageStop(t *testing.T) {
	ai.GlobalProviderRegistry.Reset()
	defer ai.GlobalProviderRegistry.Reset()

	server := newAnthropicSSEServer(t)
	defer server.Close()

	registerAnthropicTestProvider(t, "callstream-anthropic", server.URL)

	ctx := streamTestContext(t)
	result, err := aiCallStream(ctx, []eval.Value{
		&eval.StringValue{Value: "callstream-anthropic"},
		&eval.StringValue{Value: "claude-sonnet-4-5"},
		&eval.StringValue{Value: `[{"role":"user","content":"hi"}]`},
	})
	if err != nil {
		t.Fatalf("aiCallStream returned Go error: %v", err)
	}
	tagged := result.(*eval.TaggedValue)
	if tagged.CtorName != "Ok" {
		t.Fatalf("expected Ok, got %s; %+v", tagged.CtorName, tagged.Fields)
	}
	str := tagged.Fields[0].(*eval.StringValue).Value
	if str != "Hi there" {
		t.Errorf("accumulated string = %q, want %q", str, "Hi there")
	}
}

// TestAICallStream_ProviderNotFound: provider name not registered.
//
// In this code path the underlying streamCall returns Err(ProtocolError("[ProviderNotFound] ..."))
// and the wrapper unpacks the [code] prefix into AIError.code.
func TestAICallStream_ProviderNotFound(t *testing.T) {
	ai.GlobalProviderRegistry.Reset()
	defer ai.GlobalProviderRegistry.Reset()

	ctx := streamTestContext(t)
	result, err := aiCallStream(ctx, []eval.Value{
		&eval.StringValue{Value: "no-such-provider"},
		&eval.StringValue{Value: "any"},
		&eval.StringValue{Value: `[{"role":"user","content":"hi"}]`},
	})
	if err != nil {
		t.Fatalf("expected structured Err, got Go error: %v", err)
	}
	assertAIErrorCode(t, result, "ProviderNotFound", false)
}

// TestAICallStream_StreamingNotEnabled: provider with streaming.enabled=false.
//
// streamCall rejects the call with ProtocolError("[CapabilityNotSupported] ...");
// the wrapper unpacks that into AIError.code = "CapabilityNotSupported".
func TestAICallStream_StreamingNotEnabled(t *testing.T) {
	ai.GlobalProviderRegistry.Reset()
	defer ai.GlobalProviderRegistry.Reset()

	spec := &pkg.AIProviderSpec{
		SchemaVersion: 1,
		Name:          "no-stream-cs",
		Endpoint:      "http://localhost:1",
		RequestShape:  "openai_chat",
		ResponsePath:  "$.choices[0].message.content",
		Auth:          pkg.AIProviderAuth{Type: "none"},
		Capabilities:  pkg.AIProviderCapabilities{Streaming: false},
	}
	if err := ai.GlobalProviderRegistry.Register("no-stream-cs", configdriven.New(spec), "test://no-stream-cs"); err != nil {
		t.Fatal(err)
	}

	ctx := streamTestContext(t)
	result, _ := aiCallStream(ctx, []eval.Value{
		&eval.StringValue{Value: "no-stream-cs"},
		&eval.StringValue{Value: "x"},
		&eval.StringValue{Value: `[{"role":"user","content":"hi"}]`},
	})
	assertAIErrorCode(t, result, "CapabilityNotSupported", false)
}

// TestAICallStream_ReasoningFieldsDiscarded: reasoning_content / thinking
// deltas in the upstream JSON are READ but NOT included in the accumulated
// string — only visible content fields appear. v0.15.1 ignores reasoning;
// v0.15.2 will surface it via callStreamWithReasoning.
//
// The mock server emits events that include BOTH delta.content (visible)
// and delta.reasoning_content (hidden). Accumulator must return only
// content concatenation.
func TestAICallStream_ReasoningFieldsDiscarded(t *testing.T) {
	ai.GlobalProviderRegistry.Reset()
	defer ai.GlobalProviderRegistry.Reset()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)
		// Event 1: only reasoning_content (no visible content) — must be discarded.
		_, _ = fmt.Fprintln(w, `data: {"choices":[{"delta":{"reasoning_content":"thinking..."}}]}`)
		_, _ = fmt.Fprintln(w, "")
		flusher.Flush()
		// Event 2: only content (visible) — must be accumulated.
		_, _ = fmt.Fprintln(w, `data: {"choices":[{"delta":{"content":"answer"}}]}`)
		_, _ = fmt.Fprintln(w, "")
		flusher.Flush()
		// Event 3: BOTH content + reasoning_content — only content accumulates.
		_, _ = fmt.Fprintln(w, `data: {"choices":[{"delta":{"content":" here","reasoning_content":"more thinking"}}]}`)
		_, _ = fmt.Fprintln(w, "")
		flusher.Flush()
		// Sentinel.
		_, _ = fmt.Fprintln(w, "data: [DONE]")
		_, _ = fmt.Fprintln(w, "")
		flusher.Flush()
	}))
	defer server.Close()

	registerOpenAITestProvider(t, "callstream-reasoning", server.URL, "")

	ctx := streamTestContext(t)
	result, err := aiCallStream(ctx, []eval.Value{
		&eval.StringValue{Value: "callstream-reasoning"},
		&eval.StringValue{Value: "deepseek-reasoner"},
		&eval.StringValue{Value: `[{"role":"user","content":"hi"}]`},
	})
	if err != nil {
		t.Fatalf("aiCallStream returned Go error: %v", err)
	}
	tagged := result.(*eval.TaggedValue)
	if tagged.CtorName != "Ok" {
		t.Fatalf("expected Ok, got %s", tagged.CtorName)
	}
	str := tagged.Fields[0].(*eval.StringValue).Value
	// Only visible content should appear; reasoning fields are ignored.
	if str != "answer here" {
		t.Errorf("accumulated string = %q, want %q (reasoning fields must be discarded)", str, "answer here")
	}
	if strings.Contains(str, "thinking") {
		t.Errorf("reasoning content leaked into accumulator: %q", str)
	}
}

// TestAICallStream_SkipsMalformedAndPathMissEvents: events whose JSON is
// malformed OR whose delta_path doesn't resolve must be silently skipped,
// not error the whole stream. This matches Anthropic's message_start
// (no delta_path field) and OpenAI's role-only first chunk.
func TestAICallStream_SkipsMalformedAndPathMissEvents(t *testing.T) {
	ai.GlobalProviderRegistry.Reset()
	defer ai.GlobalProviderRegistry.Reset()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)
		// Path-miss: delta has no .content field (role-only first chunk in OpenAI shape).
		_, _ = fmt.Fprintln(w, `data: {"choices":[{"delta":{"role":"assistant"}}]}`)
		_, _ = fmt.Fprintln(w, "")
		flusher.Flush()
		// Malformed JSON.
		_, _ = fmt.Fprintln(w, `data: {not valid json`)
		_, _ = fmt.Fprintln(w, "")
		flusher.Flush()
		// Real content.
		_, _ = fmt.Fprintln(w, `data: {"choices":[{"delta":{"content":"OK"}}]}`)
		_, _ = fmt.Fprintln(w, "")
		flusher.Flush()
		_, _ = fmt.Fprintln(w, "data: [DONE]")
		_, _ = fmt.Fprintln(w, "")
		flusher.Flush()
	}))
	defer server.Close()

	registerOpenAITestProvider(t, "callstream-skipping", server.URL, "")

	ctx := streamTestContext(t)
	result, err := aiCallStream(ctx, []eval.Value{
		&eval.StringValue{Value: "callstream-skipping"},
		&eval.StringValue{Value: "gpt-4o"},
		&eval.StringValue{Value: `[{"role":"user","content":"hi"}]`},
	})
	if err != nil {
		t.Fatalf("aiCallStream returned Go error: %v", err)
	}
	tagged := result.(*eval.TaggedValue)
	if tagged.CtorName != "Ok" {
		t.Fatalf("expected Ok despite skippable events, got %s", tagged.CtorName)
	}
	str := tagged.Fields[0].(*eval.StringValue).Value
	if str != "OK" {
		t.Errorf("accumulated = %q, want %q", str, "OK")
	}
}

// TestAICallStream_NoSeparateCallStreamSpan: the accumulator MUST NOT emit
// a separate AI/callStream trace span — it relies on the underlying
// streamCall op's existing span. A double-tagged span would double-count
// the effect for cost-visibility purposes (A9).
//
// Note: effects.Call itself records a generic effect event in addition to
// the AI op handler's RecordAIEffect — that is pre-existing dispatcher
// behaviour and produces multiple "streamCall" events per call. The
// load-bearing assertion here is the OpName: there must be NO event with
// op=callStream, only op=streamCall (which the underlying primitive owns).
func TestAICallStream_NoSeparateCallStreamSpan(t *testing.T) {
	ai.GlobalProviderRegistry.Reset()
	defer ai.GlobalProviderRegistry.Reset()

	server := newOpenAISSEServer(t, nil)
	defer server.Close()

	registerOpenAITestProvider(t, "callstream-span-test", server.URL, "")

	ctx := streamTestContext(t)
	ctx.Trace = trace.NewCollector()
	ctx.Stream.IdleTimeout = 1 * time.Second
	ctx.Stream.MaxDuration = 2 * time.Second

	_, err := aiCallStream(ctx, []eval.Value{
		&eval.StringValue{Value: "callstream-span-test"},
		&eval.StringValue{Value: "gpt-4o"},
		&eval.StringValue{Value: `[{"role":"user","content":"hi"}]`},
	})
	if err != nil {
		t.Fatalf("aiCallStream returned Go error: %v", err)
	}

	// Inspect every AI event's OpName. None must be "callStream" — that is
	// the explicit invariant: aiCallStream MUST NOT emit its own AI span.
	var sawStreamCall bool
	for _, evt := range ctx.Trace.Events() {
		if evt.Effect == nil || evt.Effect.EffectName != "AI" {
			continue
		}
		if evt.Effect.OpName == "callStream" {
			t.Errorf("aiCallStream emitted a forbidden AI/callStream span — accumulator must reuse the underlying streamCall span; got %+v",
				evt.Effect)
		}
		if evt.Effect.OpName == "streamCall" {
			sawStreamCall = true
		}
	}
	if !sawStreamCall {
		t.Errorf("expected at least one AI/streamCall span (from the underlying primitive), got none")
	}
}

// TestAICallStream_DoneSentinelTerminatesCleanly: the [DONE] sentinel must
// terminate the accumulator with no spurious empty-delta append.
func TestAICallStream_DoneSentinelTerminatesCleanly(t *testing.T) {
	ai.GlobalProviderRegistry.Reset()
	defer ai.GlobalProviderRegistry.Reset()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)
		_, _ = fmt.Fprintln(w, `data: {"choices":[{"delta":{"content":"single"}}]}`)
		_, _ = fmt.Fprintln(w, "")
		flusher.Flush()
		_, _ = fmt.Fprintln(w, "data: [DONE]")
		_, _ = fmt.Fprintln(w, "")
		flusher.Flush()
		// Bytes after [DONE] should be ignored — the loop must have already
		// returned. If termination is buggy, this delta would slip through.
		_, _ = fmt.Fprintln(w, `data: {"choices":[{"delta":{"content":"AFTER_DONE"}}]}`)
		_, _ = fmt.Fprintln(w, "")
		flusher.Flush()
	}))
	defer server.Close()

	registerOpenAITestProvider(t, "callstream-done", server.URL, "")

	ctx := streamTestContext(t)
	result, err := aiCallStream(ctx, []eval.Value{
		&eval.StringValue{Value: "callstream-done"},
		&eval.StringValue{Value: "gpt-4o"},
		&eval.StringValue{Value: `[{"role":"user","content":"hi"}]`},
	})
	if err != nil {
		t.Fatalf("aiCallStream returned Go error: %v", err)
	}
	tagged := result.(*eval.TaggedValue)
	if tagged.CtorName != "Ok" {
		t.Fatalf("expected Ok, got %s", tagged.CtorName)
	}
	str := tagged.Fields[0].(*eval.StringValue).Value
	if str != "single" {
		t.Errorf("accumulated = %q, want %q ([DONE] sentinel may not terminate cleanly)", str, "single")
	}
}

// assertAIErrorCode unwraps an Err(AIError record) and verifies the code
// + retryable fields. Fails the test if the result isn't Err or doesn't
// contain a record-shaped inner.
func assertAIErrorCode(t *testing.T, result eval.Value, wantCode string, wantRetryable bool) {
	t.Helper()
	tagged, ok := result.(*eval.TaggedValue)
	if !ok || tagged.CtorName != "Err" {
		t.Fatalf("expected Err(AIError), got %+v", result)
	}
	rec, ok := tagged.Fields[0].(*eval.RecordValue)
	if !ok {
		t.Fatalf("expected RecordValue inside Err, got %T", tagged.Fields[0])
	}
	codeField, ok := rec.Fields["code"].(*eval.StringValue)
	if !ok {
		t.Fatalf("AIError.code missing or wrong type, got %+v", rec.Fields)
	}
	if codeField.Value != wantCode {
		t.Errorf("AIError.code = %q, want %q", codeField.Value, wantCode)
	}
	retryField, ok := rec.Fields["retryable"].(*eval.BoolValue)
	if !ok {
		t.Fatalf("AIError.retryable missing or wrong type")
	}
	if retryField.Value != wantRetryable {
		t.Errorf("AIError.retryable = %v, want %v", retryField.Value, wantRetryable)
	}
}
