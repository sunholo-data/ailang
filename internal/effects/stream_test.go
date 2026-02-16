package effects

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sunholo/ailang/internal/eval"
)

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
	server := newTestWSServer(t)
	defer server.Close()

	ctx := NewEffContext(nil)
	ctx.Grant(NewCapability("Stream"))
	ctx.Stream = NewStreamContext()
	ctx.Stream.AllowHTTP = true
	ctx.Stream.AllowLocalhost = true

	url := wsURL(server.URL)

	result, err := StreamConnect(ctx, []eval.Value{
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
	StreamClose(ctx, []eval.Value{result.(*eval.TaggedValue).Fields[0]})
}

func TestStreamConnect_SecurityBlocksWS(t *testing.T) {
	server := newTestWSServer(t)
	defer server.Close()

	ctx := NewEffContext(nil)
	ctx.Grant(NewCapability("Stream"))
	ctx.Stream = NewStreamContext()
	// AllowHTTP = false (default)

	url := wsURL(server.URL)

	result, err := StreamConnect(ctx, []eval.Value{
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

func TestStreamConnect_NoStreamContext(t *testing.T) {
	ctx := NewEffContext(nil)
	// No Stream capability or context

	_, err := StreamConnect(ctx, []eval.Value{
		&eval.StringValue{Value: "wss://example.com/ws"},
		&eval.RecordValue{Fields: map[string]eval.Value{}},
	})
	if err == nil {
		t.Fatal("expected error when Stream context is nil")
	}
}

func TestStreamSend_TextMessage(t *testing.T) {
	server := newTestWSServer(t)
	defer server.Close()

	ctx := NewEffContext(nil)
	ctx.Grant(NewCapability("Stream"))
	ctx.Stream = NewStreamContext()
	ctx.Stream.AllowHTTP = true
	ctx.Stream.AllowLocalhost = true

	url := wsURL(server.URL)

	// Connect
	result, _ := StreamConnect(ctx, []eval.Value{
		&eval.StringValue{Value: url},
		&eval.RecordValue{Fields: map[string]eval.Value{}},
	})
	connVal := result.(*eval.TaggedValue).Fields[0]

	// Send text message
	msg := &eval.TaggedValue{
		CtorName: "Text",
		Fields:   []eval.Value{&eval.StringValue{Value: "hello"}},
	}
	sendResult, err := StreamSend(ctx, []eval.Value{connVal, msg})
	if err != nil {
		t.Fatalf("streamSend error: %v", err)
	}

	tagged, ok := sendResult.(*eval.TaggedValue)
	if !ok || tagged.CtorName != "Ok" {
		t.Fatalf("expected Ok, got %v", sendResult)
	}

	// Clean up
	StreamClose(ctx, []eval.Value{connVal})
}

func TestStreamSend_MessageTooLarge(t *testing.T) {
	server := newTestWSServer(t)
	defer server.Close()

	ctx := NewEffContext(nil)
	ctx.Grant(NewCapability("Stream"))
	ctx.Stream = NewStreamContext()
	ctx.Stream.AllowHTTP = true
	ctx.Stream.AllowLocalhost = true
	ctx.Stream.MaxMessageSize = 10 // Very small limit

	url := wsURL(server.URL)

	// Connect
	result, _ := StreamConnect(ctx, []eval.Value{
		&eval.StringValue{Value: url},
		&eval.RecordValue{Fields: map[string]eval.Value{}},
	})
	connVal := result.(*eval.TaggedValue).Fields[0]

	// Send oversized message
	msg := &eval.TaggedValue{
		CtorName: "Text",
		Fields:   []eval.Value{&eval.StringValue{Value: "this message is way too long"}},
	}
	sendResult, err := StreamSend(ctx, []eval.Value{connVal, msg})
	if err != nil {
		t.Fatalf("streamSend error: %v", err)
	}

	tagged, ok := sendResult.(*eval.TaggedValue)
	if !ok || tagged.CtorName != "Err" {
		t.Fatalf("expected Err for oversized message, got %v", sendResult)
	}

	// Clean up
	StreamClose(ctx, []eval.Value{connVal})
}

func TestStreamEventLoop_ReceivesMessages(t *testing.T) {
	server := newTestWSServer(t)
	defer server.Close()

	ctx := NewEffContext(nil)
	ctx.Grant(NewCapability("Stream"))
	ctx.Stream = NewStreamContext()
	ctx.Stream.AllowHTTP = true
	ctx.Stream.AllowLocalhost = true
	ctx.Stream.IdleTimeout = 2 * time.Second
	ctx.Stream.MaxDuration = 5 * time.Second

	url := wsURL(server.URL)

	// Connect
	result, _ := StreamConnect(ctx, []eval.Value{
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
	StreamOnEvent(ctx, []eval.Value{connVal, handler})

	// Send a message that will be echoed back
	msg := &eval.TaggedValue{
		CtorName: "Text",
		Fields:   []eval.Value{&eval.StringValue{Value: "test-echo"}},
	}
	StreamSend(ctx, []eval.Value{connVal, msg})

	// Run event loop (blocks until handler returns false)
	_, err := StreamRunEventLoop(ctx, []eval.Value{connVal})
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
	StreamClose(ctx, []eval.Value{connVal})
}

func TestStreamEventLoop_HandlerReturnsFalseStops(t *testing.T) {
	server := newTestWSServer(t)
	defer server.Close()

	ctx := NewEffContext(nil)
	ctx.Grant(NewCapability("Stream"))
	ctx.Stream = NewStreamContext()
	ctx.Stream.AllowHTTP = true
	ctx.Stream.AllowLocalhost = true
	ctx.Stream.IdleTimeout = 2 * time.Second
	ctx.Stream.MaxDuration = 5 * time.Second

	url := wsURL(server.URL)

	result, _ := StreamConnect(ctx, []eval.Value{
		&eval.StringValue{Value: url},
		&eval.RecordValue{Fields: map[string]eval.Value{}},
	})
	connVal := result.(*eval.TaggedValue).Fields[0]

	// Handler returns false on first event
	ctx.FnCaller = func(fn eval.Value, arg eval.Value) (eval.Value, error) {
		return &eval.BoolValue{Value: false}, nil
	}

	handler := &eval.StringValue{Value: "handler"}
	StreamOnEvent(ctx, []eval.Value{connVal, handler})

	// Should return quickly (handler stops on Opened event)
	done := make(chan struct{})
	go func() {
		StreamRunEventLoop(ctx, []eval.Value{connVal})
		close(done)
	}()

	select {
	case <-done:
		// OK - returned promptly
	case <-time.After(3 * time.Second):
		t.Fatal("event loop didn't stop when handler returned false")
	}

	StreamClose(ctx, []eval.Value{connVal})
}

func TestStreamEventLoop_IdleTimeout(t *testing.T) {
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

	ctx := NewEffContext(nil)
	ctx.Grant(NewCapability("Stream"))
	ctx.Stream = NewStreamContext()
	ctx.Stream.AllowHTTP = true
	ctx.Stream.AllowLocalhost = true
	ctx.Stream.IdleTimeout = 200 * time.Millisecond // Short timeout for test
	ctx.Stream.MaxDuration = 5 * time.Second

	url := wsURL(server.URL)

	result, _ := StreamConnect(ctx, []eval.Value{
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
	StreamOnEvent(ctx, []eval.Value{connVal, handler})

	// Should timeout and deliver Error(Timeout(...))
	done := make(chan struct{})
	go func() {
		StreamRunEventLoop(ctx, []eval.Value{connVal})
		close(done)
	}()

	select {
	case <-done:
		if lastEvent != "Error" {
			t.Errorf("last event should be Error (timeout), got %s", lastEvent)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("event loop didn't stop after idle timeout")
	}

	StreamClose(ctx, []eval.Value{connVal})
}

func TestStreamEventLoop_PanicRecovery(t *testing.T) {
	server := newTestWSServer(t)
	defer server.Close()

	ctx := NewEffContext(nil)
	ctx.Grant(NewCapability("Stream"))
	ctx.Stream = NewStreamContext()
	ctx.Stream.AllowHTTP = true
	ctx.Stream.AllowLocalhost = true
	ctx.Stream.IdleTimeout = 2 * time.Second
	ctx.Stream.MaxDuration = 5 * time.Second

	url := wsURL(server.URL)

	result, _ := StreamConnect(ctx, []eval.Value{
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
	StreamOnEvent(ctx, []eval.Value{connVal, handler})

	done := make(chan struct{})
	go func() {
		StreamRunEventLoop(ctx, []eval.Value{connVal})
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

	StreamClose(ctx, []eval.Value{connVal})
}

func TestStreamClose_GracefulShutdown(t *testing.T) {
	server := newTestWSServer(t)
	defer server.Close()

	ctx := NewEffContext(nil)
	ctx.Grant(NewCapability("Stream"))
	ctx.Stream = NewStreamContext()
	ctx.Stream.AllowHTTP = true
	ctx.Stream.AllowLocalhost = true

	url := wsURL(server.URL)

	result, _ := StreamConnect(ctx, []eval.Value{
		&eval.StringValue{Value: url},
		&eval.RecordValue{Fields: map[string]eval.Value{}},
	})
	connVal := result.(*eval.TaggedValue).Fields[0]

	// Close
	_, err := StreamClose(ctx, []eval.Value{connVal})
	if err != nil {
		t.Fatalf("streamClose error: %v", err)
	}

	// Status should be Closed
	statusResult, _ := StreamGetStatus(ctx, []eval.Value{connVal})
	if tagged, ok := statusResult.(*eval.TaggedValue); ok {
		if tagged.CtorName != "Closed" {
			t.Errorf("status after close = %s, want Closed", tagged.CtorName)
		}
	}
}

func TestStreamStatus(t *testing.T) {
	server := newTestWSServer(t)
	defer server.Close()

	ctx := NewEffContext(nil)
	ctx.Grant(NewCapability("Stream"))
	ctx.Stream = NewStreamContext()
	ctx.Stream.AllowHTTP = true
	ctx.Stream.AllowLocalhost = true

	url := wsURL(server.URL)

	result, _ := StreamConnect(ctx, []eval.Value{
		&eval.StringValue{Value: url},
		&eval.RecordValue{Fields: map[string]eval.Value{}},
	})
	connVal := result.(*eval.TaggedValue).Fields[0]

	// Status should be Open
	statusResult, _ := StreamGetStatus(ctx, []eval.Value{connVal})
	if tagged, ok := statusResult.(*eval.TaggedValue); ok {
		if tagged.CtorName != "Open" {
			t.Errorf("status = %s, want Open", tagged.CtorName)
		}
	}

	StreamClose(ctx, []eval.Value{connVal})
}

func TestEventToADT(t *testing.T) {
	tests := []struct {
		name     string
		evt      streamEvent
		wantCtor string
	}{
		{"message", streamEvent{kind: "message", text: "hello"}, "Message"},
		{"binary", streamEvent{kind: "binary", data: []byte{1, 2, 3}}, "Binary"},
		{"opened", streamEvent{kind: "opened", text: "graphql-ws"}, "Opened"},
		{"closed", streamEvent{kind: "closed", code: 1000, reason: "done"}, "Closed"},
		{"error", streamEvent{kind: "error", errType: "Timeout", text: "timed out"}, "Error"},
		{"ping", streamEvent{kind: "ping", data: []byte{1}}, "Ping"},
		{"unknown", streamEvent{kind: "unknown"}, "Error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := eventToADT(tt.evt)
			tagged, ok := result.(*eval.TaggedValue)
			if !ok {
				t.Fatalf("expected TaggedValue, got %T", result)
			}
			if tagged.CtorName != tt.wantCtor {
				t.Errorf("CtorName = %s, want %s", tagged.CtorName, tt.wantCtor)
			}
		})
	}
}

func TestCallHandlerSafe_PanicRecovery(t *testing.T) {
	fnCaller := func(fn eval.Value, arg eval.Value) (eval.Value, error) {
		panic("test panic")
	}

	handler := &eval.StringValue{Value: "handler"}
	event := &eval.StringValue{Value: "event"}

	shouldContinue, err := callHandlerSafe(fnCaller, handler, event)
	if err == nil {
		t.Fatal("expected error from panic recovery")
	}
	if shouldContinue {
		t.Error("shouldContinue should be false after panic")
	}
	if !strings.Contains(err.Error(), "handler panic") {
		t.Errorf("error should mention 'handler panic', got: %v", err)
	}
}

func TestCallHandlerSafe_ReturnsBool(t *testing.T) {
	fnCallerTrue := func(fn eval.Value, arg eval.Value) (eval.Value, error) {
		return &eval.BoolValue{Value: true}, nil
	}
	fnCallerFalse := func(fn eval.Value, arg eval.Value) (eval.Value, error) {
		return &eval.BoolValue{Value: false}, nil
	}

	handler := &eval.StringValue{Value: "handler"}
	event := &eval.StringValue{Value: "event"}

	cont, err := callHandlerSafe(fnCallerTrue, handler, event)
	if err != nil || !cont {
		t.Errorf("expected (true, nil), got (%v, %v)", cont, err)
	}

	cont, err = callHandlerSafe(fnCallerFalse, handler, event)
	if err != nil || cont {
		t.Errorf("expected (false, nil), got (%v, %v)", cont, err)
	}
}

func TestMakeStreamHelpers(t *testing.T) {
	// Test makeStreamConn
	conn := makeStreamConn(42)
	tagged := conn.(*eval.TaggedValue)
	if tagged.CtorName != "StreamConn" {
		t.Errorf("makeStreamConn CtorName = %s, want StreamConn", tagged.CtorName)
	}
	if tagged.Fields[0].(*eval.IntValue).Value != 42 {
		t.Errorf("makeStreamConn ID = %d, want 42", tagged.Fields[0].(*eval.IntValue).Value)
	}

	// Test makeStreamErr
	errVal := makeStreamErr("Timeout", "timed out")
	errTagged := errVal.(*eval.TaggedValue)
	if errTagged.CtorName != "Err" {
		t.Errorf("makeStreamErr CtorName = %s, want Err", errTagged.CtorName)
	}
	inner := errTagged.Fields[0].(*eval.TaggedValue)
	if inner.CtorName != "Timeout" {
		t.Errorf("inner CtorName = %s, want Timeout", inner.CtorName)
	}

	// Test makeStreamOk
	okVal := makeStreamOk(&eval.IntValue{Value: 1})
	okTagged := okVal.(*eval.TaggedValue)
	if okTagged.CtorName != "Ok" {
		t.Errorf("makeStreamOk CtorName = %s, want Ok", okTagged.CtorName)
	}

	// Test makeStreamOkUnit
	unitVal := makeStreamOkUnit()
	unitTagged := unitVal.(*eval.TaggedValue)
	if unitTagged.CtorName != "Ok" {
		t.Errorf("makeStreamOkUnit CtorName = %s, want Ok", unitTagged.CtorName)
	}
}

func TestExtractConnID(t *testing.T) {
	// Valid
	id, err := extractConnID(&eval.TaggedValue{
		CtorName: "StreamConn",
		Fields:   []eval.Value{&eval.IntValue{Value: 5}},
	})
	if err != nil || id != 5 {
		t.Errorf("extractConnID valid = (%d, %v), want (5, nil)", id, err)
	}

	// Invalid type
	_, err = extractConnID(&eval.StringValue{Value: "not a conn"})
	if err == nil {
		t.Error("expected error for non-TaggedValue")
	}

	// Wrong constructor
	_, err = extractConnID(&eval.TaggedValue{
		CtorName: "NotStreamConn",
		Fields:   []eval.Value{&eval.IntValue{Value: 1}},
	})
	if err == nil {
		t.Error("expected error for wrong constructor name")
	}
}
