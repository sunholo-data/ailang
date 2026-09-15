package effects

import (
	"errors"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/eval"
)

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

// TestStreamConnect_UnregisteredTransport: a ws:// URL with no transport
// registered is a harness error — the typed sentinel as a Go error, not a
// program-visible Err(ConnectionFailed) and not a panic.
func TestStreamConnect_UnregisteredTransport(t *testing.T) {
	RegisterStreamTransport("ws", nil)

	ctx := NewEffContext(nil)
	ctx.Grant(NewCapability("Stream"))
	ctx.Stream = NewStreamContext()
	ctx.Stream.AllowHTTP = true
	ctx.Stream.AllowLocalhost = true

	result, err := StreamConnect(ctx, []eval.Value{
		&eval.StringValue{Value: "ws://127.0.0.1:1/ws"},
		&eval.RecordValue{Fields: map[string]eval.Value{}},
	})
	if err == nil {
		t.Fatalf("expected a Go error with no transport registered, got result %v", result)
	}
	if !errors.Is(err, ErrBackendNotRegistered) {
		t.Fatalf("errors.Is(err, ErrBackendNotRegistered) = false; err = %v", err)
	}
	if result != nil {
		t.Errorf("result should be nil on a harness error, got %v", result)
	}
}

// TestStreamConnect_NonWSScheme: http(s) is the SSE path; Stream.connect
// reports it as a program-visible ConnectionFailed, never a registry error.
func TestStreamConnect_NonWSScheme(t *testing.T) {
	ctx := NewEffContext(nil)
	ctx.Grant(NewCapability("Stream"))
	ctx.Stream = NewStreamContext()
	ctx.Stream.AllowHTTP = true
	ctx.Stream.AllowLocalhost = true

	result, err := StreamConnect(ctx, []eval.Value{
		&eval.StringValue{Value: "https://127.0.0.1:1/events"},
		&eval.RecordValue{Fields: map[string]eval.Value{}},
	})
	if err != nil {
		t.Fatalf("non-ws scheme should not be a Go error: %v", err)
	}
	tagged, ok := result.(*eval.TaggedValue)
	if !ok || tagged.CtorName != "Err" {
		t.Fatalf("expected Err result, got %v", result)
	}
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
		{"error", streamEvent{kind: "error", errType: "Timeout", text: "timed out"}, "StreamError"},
		{"ping", streamEvent{kind: "ping", data: []byte{1}}, "Ping"},
		{"unknown", streamEvent{kind: "unknown"}, "StreamError"},
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

	// Ok-wrapped StreamConn (the withStream/withSSE bug fix)
	id, err = extractConnID(&eval.TaggedValue{
		CtorName: "Ok",
		Fields: []eval.Value{
			&eval.TaggedValue{
				CtorName: "StreamConn",
				Fields:   []eval.Value{&eval.IntValue{Value: 7}},
			},
		},
	})
	if err != nil {
		t.Errorf("extractConnID(Ok(StreamConn(7))): unexpected error: %v", err)
	}
	if id != 7 {
		t.Errorf("extractConnID(Ok(StreamConn(7))) = %d, want 7", id)
	}

	// Err gives descriptive error
	_, err = extractConnID(&eval.TaggedValue{
		CtorName: "Err",
		Fields: []eval.Value{
			&eval.TaggedValue{
				CtorName: "ConnectionFailed",
				Fields:   []eval.Value{&eval.StringValue{Value: "timeout"}},
			},
		},
	})
	if err == nil {
		t.Fatal("expected error for Err(...)")
	}
	if !strings.Contains(err.Error(), "stream connection failed") {
		t.Errorf("error should contain 'stream connection failed', got: %v", err)
	}
	if !strings.Contains(err.Error(), "ConnectionFailed") {
		t.Errorf("error should contain 'ConnectionFailed', got: %v", err)
	}

	// Ok with wrong inner type
	_, err = extractConnID(&eval.TaggedValue{
		CtorName: "Ok",
		Fields:   []eval.Value{&eval.StringValue{Value: "not a StreamConn"}},
	})
	if err == nil {
		t.Error("expected error for Ok(string)")
	}

	// Ok with wrong constructor
	_, err = extractConnID(&eval.TaggedValue{
		CtorName: "Ok",
		Fields: []eval.Value{
			&eval.TaggedValue{
				CtorName: "SomeOtherADT",
				Fields:   []eval.Value{&eval.IntValue{Value: 42}},
			},
		},
	})
	if err == nil {
		t.Error("expected error for Ok(SomeOtherADT)")
	}
}
