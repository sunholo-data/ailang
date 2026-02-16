package effects

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sunholo/ailang/internal/eval"
)

// TestStreamIntegration_FullEchoFlow tests the complete WebSocket flow:
// connect → send → receive echo → close
func TestStreamIntegration_FullEchoFlow(t *testing.T) {
	// Create echo server
	server := newTestWSServer(t)
	defer server.Close()

	// Set up effect context with Stream capability
	ctx := NewEffContext(nil)
	ctx.Grant(NewCapability("Stream"))
	ctx.Stream = NewStreamContext()
	ctx.Stream.AllowHTTP = true
	ctx.Stream.AllowLocalhost = true
	ctx.Stream.IdleTimeout = 3 * time.Second
	ctx.Stream.MaxDuration = 10 * time.Second

	// Track events for assertions
	var receivedMessages []string
	var receivedEvents []string
	var eventCount int32

	ctx.FnCaller = func(fn eval.Value, arg eval.Value) (eval.Value, error) {
		tagged, ok := arg.(*eval.TaggedValue)
		if !ok {
			return &eval.BoolValue{Value: true}, nil
		}
		receivedEvents = append(receivedEvents, tagged.CtorName)

		switch tagged.CtorName {
		case "Message":
			if len(tagged.Fields) > 0 {
				if sv, ok := tagged.Fields[0].(*eval.StringValue); ok {
					receivedMessages = append(receivedMessages, sv.Value)
				}
			}
			// Stop after receiving the echo
			return &eval.BoolValue{Value: false}, nil
		case "Opened":
			atomic.AddInt32(&eventCount, 1)
			return &eval.BoolValue{Value: true}, nil
		default:
			return &eval.BoolValue{Value: true}, nil
		}
	}

	url := wsURL(server.URL)

	// Step 1: Connect
	connectResult, err := StreamConnect(ctx, []eval.Value{
		&eval.StringValue{Value: url},
		&eval.RecordValue{Fields: map[string]eval.Value{
			"headers":      &eval.ListValue{Elements: []eval.Value{}},
			"subprotocols": &eval.ListValue{Elements: []eval.Value{}},
		}},
	})
	if err != nil {
		t.Fatalf("connect error: %v", err)
	}

	resultTag := connectResult.(*eval.TaggedValue)
	if resultTag.CtorName != "Ok" {
		t.Fatalf("connect failed: %v", connectResult)
	}
	connVal := resultTag.Fields[0]

	// Step 2: Send message
	sendResult, err := StreamSend(ctx, []eval.Value{
		connVal,
		&eval.TaggedValue{
			CtorName: "Text",
			Fields:   []eval.Value{&eval.StringValue{Value: "integration-test"}},
		},
	})
	if err != nil {
		t.Fatalf("send error: %v", err)
	}
	sendTag := sendResult.(*eval.TaggedValue)
	if sendTag.CtorName != "Ok" {
		t.Fatalf("send failed: %v", sendResult)
	}

	// Step 3: Register handler
	handler := &eval.StringValue{Value: "handler-placeholder"}
	_, err = StreamOnEvent(ctx, []eval.Value{connVal, handler})
	if err != nil {
		t.Fatalf("onEvent error: %v", err)
	}

	// Step 4: Run event loop (should receive Opened + Message then stop)
	done := make(chan error, 1)
	go func() {
		_, err := StreamRunEventLoop(ctx, []eval.Value{connVal})
		done <- err
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("runEventLoop error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("event loop didn't complete in time")
	}

	// Step 5: Verify
	if len(receivedMessages) != 1 {
		t.Fatalf("expected 1 echo message, got %d: %v", len(receivedMessages), receivedMessages)
	}
	if receivedMessages[0] != "integration-test" {
		t.Errorf("echo message = %q, want %q", receivedMessages[0], "integration-test")
	}

	// Should have Opened + Message events
	if len(receivedEvents) < 2 {
		t.Errorf("expected at least 2 events, got %d: %v", len(receivedEvents), receivedEvents)
	}
	if receivedEvents[0] != "Opened" {
		t.Errorf("first event = %s, want Opened", receivedEvents[0])
	}

	// Step 6: Close
	_, err = StreamClose(ctx, []eval.Value{connVal})
	if err != nil {
		t.Fatalf("close error: %v", err)
	}

	// Verify closed
	statusResult, _ := StreamGetStatus(ctx, []eval.Value{connVal})
	if tagged, ok := statusResult.(*eval.TaggedValue); ok {
		if tagged.CtorName != "Closed" {
			t.Errorf("status after close = %s, want Closed", tagged.CtorName)
		}
	}
}

// TestStreamIntegration_MultipleMessages tests sending and receiving multiple messages.
func TestStreamIntegration_MultipleMessages(t *testing.T) {
	server := newTestWSServer(t)
	defer server.Close()

	ctx := NewEffContext(nil)
	ctx.Grant(NewCapability("Stream"))
	ctx.Stream = NewStreamContext()
	ctx.Stream.AllowHTTP = true
	ctx.Stream.AllowLocalhost = true
	ctx.Stream.IdleTimeout = 3 * time.Second
	ctx.Stream.MaxDuration = 10 * time.Second

	var messages []string
	messageTarget := 3

	ctx.FnCaller = func(fn eval.Value, arg eval.Value) (eval.Value, error) {
		tagged, ok := arg.(*eval.TaggedValue)
		if !ok {
			return &eval.BoolValue{Value: true}, nil
		}
		if tagged.CtorName == "Message" && len(tagged.Fields) > 0 {
			if sv, ok := tagged.Fields[0].(*eval.StringValue); ok {
				messages = append(messages, sv.Value)
			}
			if len(messages) >= messageTarget {
				return &eval.BoolValue{Value: false}, nil
			}
		}
		return &eval.BoolValue{Value: true}, nil
	}

	url := wsURL(server.URL)

	// Connect
	result, _ := StreamConnect(ctx, []eval.Value{
		&eval.StringValue{Value: url},
		&eval.RecordValue{Fields: map[string]eval.Value{}},
	})
	connVal := result.(*eval.TaggedValue).Fields[0]

	// Send 3 messages
	for i := 0; i < messageTarget; i++ {
		msg := &eval.TaggedValue{
			CtorName: "Text",
			Fields:   []eval.Value{&eval.StringValue{Value: fmt.Sprintf("msg-%d", i)}},
		}
		StreamSend(ctx, []eval.Value{connVal, msg})
	}

	// Register handler and run
	handler := &eval.StringValue{Value: "handler"}
	StreamOnEvent(ctx, []eval.Value{connVal, handler})

	done := make(chan struct{})
	go func() {
		StreamRunEventLoop(ctx, []eval.Value{connVal})
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("event loop timeout")
	}

	if len(messages) != messageTarget {
		t.Fatalf("expected %d messages, got %d: %v", messageTarget, len(messages), messages)
	}
	for i := 0; i < messageTarget; i++ {
		expected := fmt.Sprintf("msg-%d", i)
		if messages[i] != expected {
			t.Errorf("message[%d] = %q, want %q", i, messages[i], expected)
		}
	}

	StreamClose(ctx, []eval.Value{connVal})
}

// TestStreamIntegration_ConnectionLimit tests that connection limits are enforced.
func TestStreamIntegration_ConnectionLimit(t *testing.T) {
	server := newTestWSServer(t)
	defer server.Close()

	ctx := NewEffContext(nil)
	ctx.Grant(NewCapability("Stream"))
	ctx.Stream = NewStreamContext()
	ctx.Stream.AllowHTTP = true
	ctx.Stream.AllowLocalhost = true
	ctx.Stream.MaxConnections = 2 // Very low limit

	url := wsURL(server.URL)
	config := &eval.RecordValue{Fields: map[string]eval.Value{}}

	// Open 2 connections (should succeed)
	conns := make([]eval.Value, 0)
	for i := 0; i < 2; i++ {
		result, err := StreamConnect(ctx, []eval.Value{
			&eval.StringValue{Value: url},
			config,
		})
		if err != nil {
			t.Fatalf("connect %d error: %v", i, err)
		}
		tag := result.(*eval.TaggedValue)
		if tag.CtorName != "Ok" {
			t.Fatalf("connect %d failed: %v", i, result)
		}
		conns = append(conns, tag.Fields[0])
	}

	// Third connection should fail
	result, err := StreamConnect(ctx, []eval.Value{
		&eval.StringValue{Value: url},
		config,
	})
	if err != nil {
		t.Fatalf("connect 3 error: %v", err)
	}
	tag := result.(*eval.TaggedValue)
	if tag.CtorName != "Err" {
		t.Fatalf("expected Err for 3rd connection, got %s", tag.CtorName)
	}

	// Close one, then try again (should succeed)
	StreamClose(ctx, []eval.Value{conns[0]})

	result, err = StreamConnect(ctx, []eval.Value{
		&eval.StringValue{Value: url},
		config,
	})
	if err != nil {
		t.Fatalf("reconnect error: %v", err)
	}
	tag = result.(*eval.TaggedValue)
	if tag.CtorName != "Ok" {
		t.Fatalf("reconnect failed: %v", result)
	}

	// Clean up
	StreamClose(ctx, []eval.Value{conns[1]})
	StreamClose(ctx, []eval.Value{tag.Fields[0]})
}

// TestStreamIntegration_ServerClose tests handling of server-initiated close.
func TestStreamIntegration_ServerClose(t *testing.T) {
	// Server that closes after first message
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		// Read one message then close
		conn.ReadMessage()
		conn.WriteMessage(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, "server done"))
		conn.Close()
	}))
	defer server.Close()

	ctx := NewEffContext(nil)
	ctx.Grant(NewCapability("Stream"))
	ctx.Stream = NewStreamContext()
	ctx.Stream.AllowHTTP = true
	ctx.Stream.AllowLocalhost = true
	ctx.Stream.IdleTimeout = 3 * time.Second
	ctx.Stream.MaxDuration = 10 * time.Second

	var lastEvent string
	ctx.FnCaller = func(fn eval.Value, arg eval.Value) (eval.Value, error) {
		if tagged, ok := arg.(*eval.TaggedValue); ok {
			lastEvent = tagged.CtorName
			if tagged.CtorName == "Closed" || tagged.CtorName == "Error" {
				return &eval.BoolValue{Value: false}, nil
			}
		}
		return &eval.BoolValue{Value: true}, nil
	}

	url := wsURL(server.URL)

	result, _ := StreamConnect(ctx, []eval.Value{
		&eval.StringValue{Value: url},
		&eval.RecordValue{Fields: map[string]eval.Value{}},
	})
	connVal := result.(*eval.TaggedValue).Fields[0]

	// Send a message to trigger server close
	StreamSend(ctx, []eval.Value{
		connVal,
		&eval.TaggedValue{
			CtorName: "Text",
			Fields:   []eval.Value{&eval.StringValue{Value: "trigger-close"}},
		},
	})

	handler := &eval.StringValue{Value: "handler"}
	StreamOnEvent(ctx, []eval.Value{connVal, handler})

	done := make(chan struct{})
	go func() {
		StreamRunEventLoop(ctx, []eval.Value{connVal})
		close(done)
	}()

	select {
	case <-done:
		// Event loop should have exited due to server close
		if lastEvent != "Closed" && lastEvent != "Error" {
			t.Errorf("last event = %s, want Closed or Error", lastEvent)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("event loop didn't exit after server close")
	}

	StreamClose(ctx, []eval.Value{connVal})
}

// TestStreamIntegration_BudgetEnforcement tests that Stream budgets are enforced.
func TestStreamIntegration_BudgetEnforcement(t *testing.T) {
	server := newTestWSServer(t)
	defer server.Close()

	ctx := NewEffContext(nil)
	ctx.Grant(NewCapability("Stream"))
	ctx.Stream = NewStreamContext()
	ctx.Stream.AllowHTTP = true
	ctx.Stream.AllowLocalhost = true

	// Set budget: only 1 Stream operation allowed
	limit := 1
	ctx.Budget = NewBudgetContext(map[string]*int{"Stream": &limit})

	url := wsURL(server.URL)
	config := &eval.RecordValue{Fields: map[string]eval.Value{}}

	// First connect should succeed (consumes 1 budget unit)
	result, err := StreamConnect(ctx, []eval.Value{
		&eval.StringValue{Value: url},
		config,
	})
	if err != nil {
		t.Fatalf("first connect error: %v", err)
	}
	tag := result.(*eval.TaggedValue)
	if tag.CtorName != "Ok" {
		t.Fatalf("first connect failed: %v", result)
	}

	// Second connect should return BudgetExhausted error (budget = 1, used = 1)
	result2, err := StreamConnect(ctx, []eval.Value{
		&eval.StringValue{Value: url},
		config,
	})
	if err != nil {
		t.Fatalf("second connect error: %v", err)
	}
	tag2 := result2.(*eval.TaggedValue)
	if tag2.CtorName != "Err" {
		t.Fatalf("expected Err for budget-exceeded connect, got %s", tag2.CtorName)
	}
	// Check inner error is BudgetExhausted
	if len(tag2.Fields) > 0 {
		if inner, ok := tag2.Fields[0].(*eval.TaggedValue); ok {
			if inner.CtorName != "BudgetExhausted" {
				t.Errorf("expected BudgetExhausted error, got %s", inner.CtorName)
			}
		}
	}

	// Cleanup
	StreamClose(ctx, []eval.Value{tag.Fields[0]})
}

// TestStreamIntegration_RawResultPassthrough verifies that passing the raw
// Ok(StreamConn(id)) result from connect directly to onEvent/runEventLoop/close
// works. This is the exact code path used by withStream/withSSE in std/stream.ail.
func TestStreamIntegration_RawResultPassthrough(t *testing.T) {
	server := newTestWSServer(t)
	defer server.Close()

	ctx := NewEffContext(nil)
	ctx.Grant(NewCapability("Stream"))
	ctx.Stream = NewStreamContext()
	ctx.Stream.AllowHTTP = true
	ctx.Stream.AllowLocalhost = true
	ctx.Stream.IdleTimeout = 2 * time.Second
	ctx.Stream.MaxDuration = 5 * time.Second

	var eventCount int32
	ctx.FnCaller = func(fn eval.Value, arg eval.Value) (eval.Value, error) {
		atomic.AddInt32(&eventCount, 1)
		return &eval.BoolValue{Value: false}, nil // stop on first event
	}

	url := wsURL(server.URL)

	// Connect — get raw Ok(StreamConn(id)) result
	rawResult, err := StreamConnect(ctx, []eval.Value{
		&eval.StringValue{Value: url},
		&eval.RecordValue{Fields: map[string]eval.Value{}},
	})
	if err != nil {
		t.Fatalf("connect error: %v", err)
	}
	tag, ok := rawResult.(*eval.TaggedValue)
	if !ok || tag.CtorName != "Ok" {
		t.Fatalf("expected Ok(...), got %v", rawResult)
	}

	// Pass raw Ok(StreamConn(id)) directly to onEvent — this is what std/stream.ail does
	handler := &eval.StringValue{Value: "handler-placeholder"}
	_, err = StreamOnEvent(ctx, []eval.Value{rawResult, handler})
	if err != nil {
		t.Fatalf("onEvent with raw Ok(StreamConn) should work: %v", err)
	}

	// Pass raw result to runEventLoop — should dispatch the Opened event and stop
	_, err = StreamRunEventLoop(ctx, []eval.Value{rawResult})
	if err != nil {
		t.Fatalf("runEventLoop with raw Ok(StreamConn) should work: %v", err)
	}

	if atomic.LoadInt32(&eventCount) == 0 {
		t.Error("expected at least one event to be dispatched")
	}

	// Pass raw result to close — should succeed
	_, err = StreamClose(ctx, []eval.Value{rawResult})
	if err != nil {
		t.Fatalf("close with raw Ok(StreamConn) should work: %v", err)
	}
}
