package effects

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/eval"
)

// newTestNDJSONServer creates a mock server that returns the given lines as an
// NDJSON stream. Each line is written with a trailing newline and flushed
// immediately so the client observes one event per line.
func newTestNDJSONServer(t *testing.T, lines []string, contentType string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if contentType != "" {
			w.Header().Set("Content-Type", contentType)
		}
		w.WriteHeader(http.StatusOK)

		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("server does not support flushing")
			return
		}

		for _, line := range lines {
			fmt.Fprintln(w, line)
			flusher.Flush()
		}
	}))
}

func newNDJSONTestContext(t *testing.T) *EffContext {
	t.Helper()
	ctx := NewEffContext(nil)
	ctx.Grant(NewCapability("Stream"))
	ctx.Stream = NewStreamContext()
	ctx.Stream.AllowHTTP = true
	ctx.Stream.AllowLocalhost = true
	ctx.Stream.IdleTimeout = 3 * time.Second
	ctx.Stream.MaxDuration = 10 * time.Second
	return ctx
}

// emptyConfig returns the equivalent of AILANG `{headers: []}`.
func emptyConfig() eval.Value {
	return &eval.RecordValue{
		Fields: map[string]eval.Value{
			"headers": &eval.ListValue{Elements: []eval.Value{}},
		},
	}
}

// TestStreamNDJSONPost_Success exercises the happy path: three JSON lines
// arriving over the wire produce three SSEData events whose data field is the
// raw JSON line (no `data:` prefix stripped, no rejoining).
func TestStreamNDJSONPost_Success(t *testing.T) {
	lines := []string{
		`{"response":"hello","done":false}`,
		`{"response":" world","done":false}`,
		`{"response":"","done":true,"eval_count":2}`,
	}
	server := newTestNDJSONServer(t, lines, "application/x-ndjson")
	defer server.Close()

	ctx := newNDJSONTestContext(t)

	var receivedEvents []string
	var receivedData []string

	ctx.FnCaller = func(fn eval.Value, arg eval.Value) (eval.Value, error) {
		tagged, ok := arg.(*eval.TaggedValue)
		if !ok {
			return &eval.BoolValue{Value: true}, nil
		}
		receivedEvents = append(receivedEvents, tagged.CtorName)
		if tagged.CtorName == "SSEData" && len(tagged.Fields) >= 2 {
			if sv, ok := tagged.Fields[1].(*eval.StringValue); ok {
				receivedData = append(receivedData, sv.Value)
			}
		}
		// Stop after we have all three lines (don't wait for Closed).
		if len(receivedData) >= 3 {
			return &eval.BoolValue{Value: false}, nil
		}
		return &eval.BoolValue{Value: true}, nil
	}

	result, err := StreamNDJSONPost(ctx, []eval.Value{
		&eval.StringValue{Value: server.URL},
		&eval.StringValue{Value: `{"model":"test","prompt":"hi","stream":true}`},
		emptyConfig(),
	})
	if err != nil {
		t.Fatalf("NDJSON post error: %v", err)
	}

	resultTag, ok := result.(*eval.TaggedValue)
	if !ok || resultTag.CtorName != "Ok" {
		t.Fatalf("NDJSON post did not return Ok: %v", result)
	}
	connVal := resultTag.Fields[0]

	if _, err := StreamOnEvent(ctx, []eval.Value{connVal, &eval.UnitValue{}}); err != nil {
		t.Fatalf("onEvent error: %v", err)
	}
	if _, err := StreamRunEventLoop(ctx, []eval.Value{connVal}); err != nil {
		t.Fatalf("runEventLoop error: %v", err)
	}

	if len(receivedEvents) == 0 || receivedEvents[0] != "Opened" {
		t.Fatalf("expected first event Opened, got events: %v", receivedEvents)
	}

	if len(receivedData) != 3 {
		t.Fatalf("expected 3 SSEData events, got %d: %v", len(receivedData), receivedData)
	}
	for i, want := range lines {
		if receivedData[i] != want {
			t.Errorf("data[%d]: got %q, want %q", i, receivedData[i], want)
		}
	}

	StreamClose(ctx, []eval.Value{connVal})
}

// TestStreamNDJSONPost_NoContentTypeCheck confirms that the response
// content-type is NOT validated — callers have opted into NDJSON parsing by
// choosing this builtin. ssePost rejects anything that isn't
// text/event-stream; ndjsonPost must not.
func TestStreamNDJSONPost_NoContentTypeCheck(t *testing.T) {
	server := newTestNDJSONServer(t,
		[]string{`{"chunk":1}`, `{"chunk":2}`},
		"application/json", // Ollama actually uses this for /api/generate
	)
	defer server.Close()

	ctx := newNDJSONTestContext(t)
	var receivedData []string

	ctx.FnCaller = func(fn eval.Value, arg eval.Value) (eval.Value, error) {
		tagged, ok := arg.(*eval.TaggedValue)
		if !ok {
			return &eval.BoolValue{Value: true}, nil
		}
		if tagged.CtorName == "SSEData" && len(tagged.Fields) >= 2 {
			if sv, ok := tagged.Fields[1].(*eval.StringValue); ok {
				receivedData = append(receivedData, sv.Value)
			}
		}
		if len(receivedData) >= 2 {
			return &eval.BoolValue{Value: false}, nil
		}
		return &eval.BoolValue{Value: true}, nil
	}

	result, err := StreamNDJSONPost(ctx, []eval.Value{
		&eval.StringValue{Value: server.URL},
		&eval.StringValue{Value: `{}`},
		emptyConfig(),
	})
	if err != nil {
		t.Fatalf("ndjson post error: %v", err)
	}
	resultTag := result.(*eval.TaggedValue)
	if resultTag.CtorName != "Ok" {
		t.Fatalf("expected Ok (no content-type check), got %v", result)
	}
	connVal := resultTag.Fields[0]
	StreamOnEvent(ctx, []eval.Value{connVal, &eval.UnitValue{}})
	StreamRunEventLoop(ctx, []eval.Value{connVal})

	if len(receivedData) != 2 {
		t.Errorf("expected 2 events, got %d", len(receivedData))
	}
	StreamClose(ctx, []eval.Value{connVal})
}

// TestStreamNDJSONPost_RequestBodyAndHeaders verifies that the body is sent
// verbatim and that custom headers reach the server.
func TestStreamNDJSONPost_RequestBodyAndHeaders(t *testing.T) {
	wantBody := `{"model":"gemma4:26b","prompt":"hi"}`
	var gotBody string
	var gotAuth string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf, _ := io.ReadAll(r.Body)
		gotBody = string(buf)
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/x-ndjson")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, `{"ok":true}`)
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	}))
	defer server.Close()

	ctx := newNDJSONTestContext(t)
	ctx.FnCaller = func(fn eval.Value, arg eval.Value) (eval.Value, error) {
		tagged, ok := arg.(*eval.TaggedValue)
		if ok && tagged.CtorName == "SSEData" {
			return &eval.BoolValue{Value: false}, nil
		}
		return &eval.BoolValue{Value: true}, nil
	}

	cfg := &eval.RecordValue{
		Fields: map[string]eval.Value{
			"headers": &eval.ListValue{Elements: []eval.Value{
				&eval.RecordValue{Fields: map[string]eval.Value{
					"name":  &eval.StringValue{Value: "Authorization"},
					"value": &eval.StringValue{Value: "Bearer test-token"},
				}},
			}},
		},
	}

	result, err := StreamNDJSONPost(ctx, []eval.Value{
		&eval.StringValue{Value: server.URL},
		&eval.StringValue{Value: wantBody},
		cfg,
	})
	if err != nil {
		t.Fatalf("post error: %v", err)
	}
	connVal := result.(*eval.TaggedValue).Fields[0]
	StreamOnEvent(ctx, []eval.Value{connVal, &eval.UnitValue{}})
	StreamRunEventLoop(ctx, []eval.Value{connVal})

	if gotBody != wantBody {
		t.Errorf("body: got %q, want %q", gotBody, wantBody)
	}
	if gotAuth != "Bearer test-token" {
		t.Errorf("auth header: got %q, want %q", gotAuth, "Bearer test-token")
	}
	StreamClose(ctx, []eval.Value{connVal})
}

// TestStreamNDJSONPost_HTTPError ensures non-200 responses surface as an Err
// with ConnectionFailed, matching ssePost behaviour.
func TestStreamNDJSONPost_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "model not found", http.StatusNotFound)
	}))
	defer server.Close()

	ctx := newNDJSONTestContext(t)
	result, err := StreamNDJSONPost(ctx, []eval.Value{
		&eval.StringValue{Value: server.URL},
		&eval.StringValue{Value: `{}`},
		emptyConfig(),
	})
	if err != nil {
		t.Fatalf("post error: %v", err)
	}
	tag, ok := result.(*eval.TaggedValue)
	if !ok || tag.CtorName != "Err" {
		t.Fatalf("expected Err on 404, got %v", result)
	}
	// Optional: peek at error message to confirm ConnectionFailed wraps it
	if rec, ok := tag.Fields[0].(*eval.RecordValue); ok {
		if msgVal, ok := rec.Fields["message"].(*eval.StringValue); ok {
			if !strings.Contains(msgVal.Value, "404") {
				t.Errorf("expected error mentioning 404, got %q", msgVal.Value)
			}
		}
	}
}
