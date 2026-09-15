package effects_test

// WebSocket-backed Stream tests. External test package on purpose: the
// transport lives in internal/platform/streamws (which imports effects), so
// an internal test could not register it — and test imports do not count
// toward the language-core closure, which is the point of the seam.

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/platform/streamws"
)

// useWSTransport registers the platform websocket transport for one test and
// removes it afterwards so the unregistered-path test sees a clean registry.
func useWSTransport(t *testing.T) {
	t.Helper()
	streamws.Register()
	t.Cleanup(func() { effects.RegisterStreamTransport("ws", nil) })
}

// newTestWSServer creates a test WebSocket server that echoes messages.
func newTestWSServer(t *testing.T) *httptest.Server {
	t.Helper()
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Logf("test server upgrade error: %v", err)
			return
		}
		defer conn.Close()

		for {
			msgType, data, err := conn.ReadMessage()
			if err != nil {
				return
			}
			if err := conn.WriteMessage(msgType, data); err != nil {
				return
			}
		}
	}))
}

// wsURL converts http://... to ws://...
func wsURL(s string) string {
	return "ws" + strings.TrimPrefix(s, "http")
}

func TestStreamConnect_Success(t *testing.T) {
	useWSTransport(t)
	server := newTestWSServer(t)
	defer server.Close()

	ctx := effects.NewEffContext(nil)
	ctx.Grant(effects.NewCapability("Stream"))
	ctx.Stream = effects.NewStreamContext()
	ctx.Stream.AllowHTTP = true
	ctx.Stream.AllowLocalhost = true

	url := wsURL(server.URL)

	result, err := effects.StreamConnect(ctx, []eval.Value{
		&eval.StringValue{Value: url},
		&eval.RecordValue{Fields: map[string]eval.Value{}},
	})
	if err != nil {
		t.Fatalf("streamConnect error: %v", err)
	}

	// Should be Ok(StreamConn(id))
	tagged, ok := result.(*eval.TaggedValue)
	if !ok {
		t.Fatalf("expected TaggedValue, got %T", result)
	}
	if tagged.CtorName != "Ok" {
		t.Fatalf("expected Ok, got %s: %v", tagged.CtorName, result)
	}

	// Extract StreamConn(id)
	connVal, ok := tagged.Fields[0].(*eval.TaggedValue)
	if !ok || connVal.CtorName != "StreamConn" {
		t.Fatalf("expected StreamConn, got %v", tagged.Fields[0])
	}

	connID := connVal.Fields[0].(*eval.IntValue).Value
	if connID < 1 {
		t.Errorf("connection ID should be >= 1, got %d", connID)
	}

	// Clean up
	effects.StreamClose(ctx, []eval.Value{result.(*eval.TaggedValue).Fields[0]})
}

func TestStreamConnect_SecurityBlocksWS(t *testing.T) {
	useWSTransport(t)
	server := newTestWSServer(t)
	defer server.Close()

	ctx := effects.NewEffContext(nil)
	ctx.Grant(effects.NewCapability("Stream"))
	ctx.Stream = effects.NewStreamContext()
	// AllowHTTP = false (default)

	url := wsURL(server.URL)

	result, err := effects.StreamConnect(ctx, []eval.Value{
		&eval.StringValue{Value: url},
		&eval.RecordValue{Fields: map[string]eval.Value{}},
	})
	if err != nil {
		t.Fatalf("streamConnect should not return Go error: %v", err)
	}

	// Should be Err(ConnectionFailed(...))
	tagged, ok := result.(*eval.TaggedValue)
	if !ok || tagged.CtorName != "Err" {
		t.Fatalf("expected Err result, got %v", result)
	}
}

func TestStreamSend_TextMessage(t *testing.T) {
	useWSTransport(t)
	server := newTestWSServer(t)
	defer server.Close()

	ctx := effects.NewEffContext(nil)
	ctx.Grant(effects.NewCapability("Stream"))
	ctx.Stream = effects.NewStreamContext()
	ctx.Stream.AllowHTTP = true
	ctx.Stream.AllowLocalhost = true

	url := wsURL(server.URL)

	// Connect
	result, _ := effects.StreamConnect(ctx, []eval.Value{
		&eval.StringValue{Value: url},
		&eval.RecordValue{Fields: map[string]eval.Value{}},
	})
	connVal := result.(*eval.TaggedValue).Fields[0]

	// Send text message
	msg := &eval.TaggedValue{
		CtorName: "Text",
		Fields:   []eval.Value{&eval.StringValue{Value: "hello"}},
	}
	sendResult, err := effects.StreamSend(ctx, []eval.Value{connVal, msg})
	if err != nil {
		t.Fatalf("streamSend error: %v", err)
	}

	tagged, ok := sendResult.(*eval.TaggedValue)
	if !ok || tagged.CtorName != "Ok" {
		t.Fatalf("expected Ok, got %v", sendResult)
	}

	// Clean up
	effects.StreamClose(ctx, []eval.Value{connVal})
}

func TestStreamSend_MessageTooLarge(t *testing.T) {
	useWSTransport(t)
	server := newTestWSServer(t)
	defer server.Close()

	ctx := effects.NewEffContext(nil)
	ctx.Grant(effects.NewCapability("Stream"))
	ctx.Stream = effects.NewStreamContext()
	ctx.Stream.AllowHTTP = true
	ctx.Stream.AllowLocalhost = true
	ctx.Stream.MaxMessageSize = 10 // Very small limit

	url := wsURL(server.URL)

	// Connect
	result, _ := effects.StreamConnect(ctx, []eval.Value{
		&eval.StringValue{Value: url},
		&eval.RecordValue{Fields: map[string]eval.Value{}},
	})
	connVal := result.(*eval.TaggedValue).Fields[0]

	// Send oversized message
	msg := &eval.TaggedValue{
		CtorName: "Text",
		Fields:   []eval.Value{&eval.StringValue{Value: "this message is way too long"}},
	}
	sendResult, err := effects.StreamSend(ctx, []eval.Value{connVal, msg})
	if err != nil {
		t.Fatalf("streamSend error: %v", err)
	}

	tagged, ok := sendResult.(*eval.TaggedValue)
	if !ok || tagged.CtorName != "Err" {
		t.Fatalf("expected Err for oversized message, got %v", sendResult)
	}

	// Clean up
	effects.StreamClose(ctx, []eval.Value{connVal})
}

func TestStreamEventLoop_ReceivesMessages(t *testing.T) {
	useWSTransport(t)
	server := newTestWSServer(t)
	defer server.Close()

	ctx := effects.NewEffContext(nil)
	ctx.Grant(effects.NewCapability("Stream"))
	ctx.Stream = effects.NewStreamContext()
	ctx.Stream.AllowHTTP = true
	ctx.Stream.AllowLocalhost = true
	ctx.Stream.IdleTimeout = 2 * time.Second
	ctx.Stream.MaxDuration = 5 * time.Second

	url := wsURL(server.URL)

	// Connect
	result, _ := effects.StreamConnect(ctx, []eval.Value{
		&eval.StringValue{Value: url},
		&eval.RecordValue{Fields: map[string]eval.Value{}},
	})
	connVal := result.(*eval.TaggedValue).Fields[0]

	// Track received events
	var mu sync.Mutex
	var events []string

	// Set up FnCaller
	eventCount := 0
	ctx.FnCaller = func(fn eval.Value, arg eval.Value) (eval.Value, error) {
		mu.Lock()
		defer mu.Unlock()
		if tagged, ok := arg.(*eval.TaggedValue); ok {
			events = append(events, tagged.CtorName)
		}
		eventCount++
		// Stop after receiving Opened + echo Message
		if eventCount >= 3 {
			return &eval.BoolValue{Value: false}, nil
		}
		return &eval.BoolValue{Value: true}, nil
	}

	// Register handler (a dummy function value; FnCaller will be called instead)
	handler := &eval.StringValue{Value: "handler-placeholder"}
	effects.StreamOnEvent(ctx, []eval.Value{connVal, handler})

	// Send a message that will be echoed back
	msg := &eval.TaggedValue{
		CtorName: "Text",
		Fields:   []eval.Value{&eval.StringValue{Value: "test-echo"}},
	}
	effects.StreamSend(ctx, []eval.Value{connVal, msg})

	// Run event loop (blocks until handler returns false)
	_, err := effects.StreamRunEventLoop(ctx, []eval.Value{connVal})
	if err != nil {
		t.Fatalf("runEventLoop error: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	// Should have received Opened event + at least one Message
	if len(events) < 2 {
		t.Errorf("expected at least 2 events, got %d: %v", len(events), events)
	}
	if events[0] != "Opened" {
		t.Errorf("first event should be Opened, got %s", events[0])
	}

	// Clean up
	effects.StreamClose(ctx, []eval.Value{connVal})
}

func TestStreamEventLoop_HandlerReturnsFalseStops(t *testing.T) {
	useWSTransport(t)
	server := newTestWSServer(t)
	defer server.Close()

	ctx := effects.NewEffContext(nil)
	ctx.Grant(effects.NewCapability("Stream"))
	ctx.Stream = effects.NewStreamContext()
	ctx.Stream.AllowHTTP = true
	ctx.Stream.AllowLocalhost = true
	ctx.Stream.IdleTimeout = 2 * time.Second
	ctx.Stream.MaxDuration = 5 * time.Second

	url := wsURL(server.URL)

	result, _ := effects.StreamConnect(ctx, []eval.Value{
		&eval.StringValue{Value: url},
		&eval.RecordValue{Fields: map[string]eval.Value{}},
	})
	connVal := result.(*eval.TaggedValue).Fields[0]

	// Handler returns false on first event
	ctx.FnCaller = func(fn eval.Value, arg eval.Value) (eval.Value, error) {
		return &eval.BoolValue{Value: false}, nil
	}

	handler := &eval.StringValue{Value: "handler"}
	effects.StreamOnEvent(ctx, []eval.Value{connVal, handler})

	// Should return quickly (handler stops on Opened event)
	done := make(chan struct{})
	go func() {
		effects.StreamRunEventLoop(ctx, []eval.Value{connVal})
		close(done)
	}()

	select {
	case <-done:
		// OK - returned promptly
	case <-time.After(3 * time.Second):
		t.Fatal("event loop didn't stop when handler returned false")
	}

	effects.StreamClose(ctx, []eval.Value{connVal})
}

func TestStreamEventLoop_IdleTimeout(t *testing.T) {
	useWSTransport(t)
	// Server that doesn't send anything after upgrade
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		// Just hold the connection open without sending
		select {}
	}))
	defer server.Close()

	ctx := effects.NewEffContext(nil)
	ctx.Grant(effects.NewCapability("Stream"))
	ctx.Stream = effects.NewStreamContext()
	ctx.Stream.AllowHTTP = true
	ctx.Stream.AllowLocalhost = true
	ctx.Stream.IdleTimeout = 200 * time.Millisecond // Short timeout for test
	ctx.Stream.MaxDuration = 5 * time.Second

	url := wsURL(server.URL)

	result, _ := effects.StreamConnect(ctx, []eval.Value{
		&eval.StringValue{Value: url},
		&eval.RecordValue{Fields: map[string]eval.Value{}},
	})
	connVal := result.(*eval.TaggedValue).Fields[0]

	var lastEvent string
	ctx.FnCaller = func(fn eval.Value, arg eval.Value) (eval.Value, error) {
		if tagged, ok := arg.(*eval.TaggedValue); ok {
			lastEvent = tagged.CtorName
		}
		return &eval.BoolValue{Value: true}, nil // keep going
	}

	handler := &eval.StringValue{Value: "handler"}
	effects.StreamOnEvent(ctx, []eval.Value{connVal, handler})

	// Should timeout and deliver StreamError(Timeout(...))
	done := make(chan struct{})
	go func() {
		effects.StreamRunEventLoop(ctx, []eval.Value{connVal})
		close(done)
	}()

	select {
	case <-done:
		if lastEvent != "StreamError" {
			t.Errorf("last event should be StreamError (timeout), got %s", lastEvent)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("event loop didn't stop after idle timeout")
	}

	effects.StreamClose(ctx, []eval.Value{connVal})
}

func TestStreamEventLoop_PanicRecovery(t *testing.T) {
	useWSTransport(t)
	server := newTestWSServer(t)
	defer server.Close()

	ctx := effects.NewEffContext(nil)
	ctx.Grant(effects.NewCapability("Stream"))
	ctx.Stream = effects.NewStreamContext()
	ctx.Stream.AllowHTTP = true
	ctx.Stream.AllowLocalhost = true
	ctx.Stream.IdleTimeout = 2 * time.Second
	ctx.Stream.MaxDuration = 5 * time.Second

	url := wsURL(server.URL)

	result, _ := effects.StreamConnect(ctx, []eval.Value{
		&eval.StringValue{Value: url},
		&eval.RecordValue{Fields: map[string]eval.Value{}},
	})
	connVal := result.(*eval.TaggedValue).Fields[0]

	callCount := 0
	ctx.FnCaller = func(fn eval.Value, arg eval.Value) (eval.Value, error) {
		callCount++
		if callCount == 1 {
			panic("test panic in handler")
		}
		// Second call will be the Error event from the panic
		return &eval.BoolValue{Value: false}, nil
	}

	handler := &eval.StringValue{Value: "handler"}
	effects.StreamOnEvent(ctx, []eval.Value{connVal, handler})

	done := make(chan struct{})
	go func() {
		effects.StreamRunEventLoop(ctx, []eval.Value{connVal})
		close(done)
	}()

	select {
	case <-done:
		// Should have recovered from panic and delivered error event
		if callCount < 2 {
			t.Errorf("expected at least 2 handler calls (original + error), got %d", callCount)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("event loop didn't recover from panic")
	}

	effects.StreamClose(ctx, []eval.Value{connVal})
}

func TestStreamClose_GracefulShutdown(t *testing.T) {
	useWSTransport(t)
	server := newTestWSServer(t)
	defer server.Close()

	ctx := effects.NewEffContext(nil)
	ctx.Grant(effects.NewCapability("Stream"))
	ctx.Stream = effects.NewStreamContext()
	ctx.Stream.AllowHTTP = true
	ctx.Stream.AllowLocalhost = true

	url := wsURL(server.URL)

	result, _ := effects.StreamConnect(ctx, []eval.Value{
		&eval.StringValue{Value: url},
		&eval.RecordValue{Fields: map[string]eval.Value{}},
	})
	connVal := result.(*eval.TaggedValue).Fields[0]

	// Close
	_, err := effects.StreamClose(ctx, []eval.Value{connVal})
	if err != nil {
		t.Fatalf("streamClose error: %v", err)
	}

	// Status should be Closed
	statusResult, _ := effects.StreamGetStatus(ctx, []eval.Value{connVal})
	if tagged, ok := statusResult.(*eval.TaggedValue); ok {
		if tagged.CtorName != "StreamClosed" {
			t.Errorf("status after close = %s, want StreamClosed", tagged.CtorName)
		}
	}
}

func TestStreamStatus(t *testing.T) {
	useWSTransport(t)
	server := newTestWSServer(t)
	defer server.Close()

	ctx := effects.NewEffContext(nil)
	ctx.Grant(effects.NewCapability("Stream"))
	ctx.Stream = effects.NewStreamContext()
	ctx.Stream.AllowHTTP = true
	ctx.Stream.AllowLocalhost = true

	url := wsURL(server.URL)

	result, _ := effects.StreamConnect(ctx, []eval.Value{
		&eval.StringValue{Value: url},
		&eval.RecordValue{Fields: map[string]eval.Value{}},
	})
	connVal := result.(*eval.TaggedValue).Fields[0]

	// Status should be Open
	statusResult, _ := effects.StreamGetStatus(ctx, []eval.Value{connVal})
	if tagged, ok := statusResult.(*eval.TaggedValue); ok {
		if tagged.CtorName != "Open" {
			t.Errorf("status = %s, want Open", tagged.CtorName)
		}
	}

	effects.StreamClose(ctx, []eval.Value{connVal})
}
