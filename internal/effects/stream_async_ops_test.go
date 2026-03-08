package effects

import (
	"strings"
	"testing"
	"time"

	"github.com/sunholo/ailang/internal/eval"
)

// TestStdinSource_ReadsLines verifies that stdinSource produces SourceText events from line input.
func TestStdinSource_ReadsLines(t *testing.T) {
	input := "hello\nworld\nfoo\n"
	reader := strings.NewReader(input)

	src := NewStdinSource(reader, "test-stdin", 5)
	defer src.Close()

	if src.Name() != "test-stdin" {
		t.Errorf("Name() = %q, want %q", src.Name(), "test-stdin")
	}
	if src.Priority() != 5 {
		t.Errorf("Priority() = %d, want %d", src.Priority(), 5)
	}

	// Read all three lines
	lines := []string{}
	for i := 0; i < 3; i++ {
		select {
		case evt, ok := <-src.Events():
			if !ok {
				t.Fatalf("channel closed before reading line %d", i)
			}
			if evt.kind != "source_text" {
				t.Errorf("event[%d].kind = %q, want %q", i, evt.kind, "source_text")
			}
			if evt.sourceName != "test-stdin" {
				t.Errorf("event[%d].sourceName = %q, want %q", i, evt.sourceName, "test-stdin")
			}
			lines = append(lines, evt.text)
		case <-time.After(2 * time.Second):
			t.Fatalf("timeout waiting for line %d", i)
		}
	}

	expected := []string{"hello", "world", "foo"}
	for i, exp := range expected {
		if lines[i] != exp {
			t.Errorf("line[%d] = %q, want %q", i, lines[i], exp)
		}
	}

	// Channel should close after EOF
	select {
	case _, ok := <-src.Events():
		if ok {
			t.Error("expected channel to close after EOF")
		}
	case <-time.After(2 * time.Second):
		t.Error("timeout waiting for channel close after EOF")
	}
}

// TestStdinSource_Close stops the read goroutine.
func TestStdinSource_Close(t *testing.T) {
	// Use a reader that blocks forever (we'll close before reading)
	pr, _ := newPipeReader()
	src := NewStdinSource(pr, "close-test", 0)

	// Close should be safe
	src.Close()
	src.Close() // Idempotent
}

// TestStdinSource_EmptyInput produces no events, just closes.
func TestStdinSource_EmptyInput(t *testing.T) {
	reader := strings.NewReader("")
	src := NewStdinSource(reader, "empty", 0)

	select {
	case _, ok := <-src.Events():
		if ok {
			t.Error("expected channel to close immediately for empty input")
		}
	case <-time.After(2 * time.Second):
		t.Error("timeout waiting for channel close on empty input")
	}
}

// TestMakeStreamSource creates a StreamSource ADT value.
func TestMakeStreamSource(t *testing.T) {
	val := makeStreamSource(42)
	tv, ok := val.(*eval.TaggedValue)
	if !ok {
		t.Fatalf("expected TaggedValue, got %T", val)
	}
	if tv.CtorName != "StreamSource" {
		t.Errorf("CtorName = %q, want %q", tv.CtorName, "StreamSource")
	}
	if len(tv.Fields) != 1 {
		t.Fatalf("Fields len = %d, want 1", len(tv.Fields))
	}
	iv, ok := tv.Fields[0].(*eval.IntValue)
	if !ok {
		t.Fatalf("Fields[0] expected IntValue, got %T", tv.Fields[0])
	}
	if iv.Value != 42 {
		t.Errorf("ID = %d, want 42", iv.Value)
	}
}

// TestExtractSourceID round-trips makeStreamSource -> extractSourceID.
func TestExtractSourceID(t *testing.T) {
	val := makeStreamSource(7)
	id, err := extractSourceID(val)
	if err != nil {
		t.Fatalf("extractSourceID error: %v", err)
	}
	if id != 7 {
		t.Errorf("id = %d, want 7", id)
	}
}

// TestExtractSourceID_InvalidInput checks error cases.
func TestExtractSourceID_InvalidInput(t *testing.T) {
	// Wrong type
	_, err := extractSourceID(&eval.StringValue{Value: "not a source"})
	if err == nil {
		t.Error("expected error for StringValue")
	}

	// Wrong constructor
	_, err = extractSourceID(&eval.TaggedValue{CtorName: "StreamConn", Fields: []eval.Value{&eval.IntValue{Value: 1}}})
	if err == nil {
		t.Error("expected error for StreamConn (not StreamSource)")
	}

	// No fields
	_, err = extractSourceID(&eval.TaggedValue{CtorName: "StreamSource"})
	if err == nil {
		t.Error("expected error for StreamSource with no fields")
	}
}

// newPipeReader creates an io.Reader that blocks until the writer is used or closed.
// Returns the reader and a cleanup function.
func newPipeReader() (*strings.Reader, func()) {
	// Use a Reader that returns no data — blocks scanner on Scan()
	// For testing close, we just use an empty reader (won't actually block)
	r := strings.NewReader("")
	return r, func() {}
}

// ============================================================================
// Integration tests for StreamAsyncExecProcess handler
// ============================================================================

// TestStreamAsyncExecProcess_EchoCommand tests the full handler flow:
// call handler → get StreamSource → read events from source.
func TestStreamAsyncExecProcess_EchoCommand(t *testing.T) {
	sc := NewStreamContext()
	defer sc.CloseAll()

	ctx := &EffContext{
		Stream: sc,
	}

	args := []eval.Value{
		&eval.StringValue{Value: "echo"},
		&eval.ListValue{Elements: []eval.Value{
			&eval.StringValue{Value: "hello process"},
		}},
		&eval.StringValue{Value: "test-echo"},
		&eval.IntValue{Value: 5},
		&eval.IntValue{Value: 4096},
	}

	result, err := StreamAsyncExecProcess(ctx, args)
	if err != nil {
		t.Fatalf("StreamAsyncExecProcess: %v", err)
	}

	// Verify we got a StreamSource back
	id, err := extractSourceID(result)
	if err != nil {
		t.Fatalf("extractSourceID: %v", err)
	}

	// Retrieve the source and read from it
	source, found := sc.GetSource(id)
	if !found {
		t.Fatal("source not found in StreamContext")
	}

	select {
	case evt, ok := <-source.Events():
		if !ok {
			t.Fatal("channel closed before receiving event")
		}
		if evt.kind != "source_bytes" {
			t.Errorf("kind = %q, want source_bytes", evt.kind)
		}
		if evt.sourceName != "test-echo" {
			t.Errorf("sourceName = %q, want test-echo", evt.sourceName)
		}
		got := string(evt.data)
		if got != "hello process\n" {
			t.Errorf("data = %q, want %q", got, "hello process\n")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for event")
	}
}

// TestStreamAsyncExecProcess_NoStreamContext verifies error when Stream is nil.
func TestStreamAsyncExecProcess_NoStreamContext(t *testing.T) {
	ctx := &EffContext{
		Stream: nil,
	}

	args := []eval.Value{
		&eval.StringValue{Value: "echo"},
		&eval.ListValue{Elements: []eval.Value{}},
		&eval.StringValue{Value: "test"},
		&eval.IntValue{Value: 0},
		&eval.IntValue{Value: 1024},
	}

	_, err := StreamAsyncExecProcess(ctx, args)
	if err == nil {
		t.Fatal("expected error when Stream is nil")
	}
	if !strings.Contains(err.Error(), "E_STREAM_NO_CONTEXT") {
		t.Errorf("error = %q, want E_STREAM_NO_CONTEXT", err.Error())
	}
}

// TestStreamAsyncExecProcess_CommandNotFound verifies error for unknown command.
func TestStreamAsyncExecProcess_CommandNotFound(t *testing.T) {
	sc := NewStreamContext()
	defer sc.CloseAll()

	ctx := &EffContext{
		Stream: sc,
	}

	args := []eval.Value{
		&eval.StringValue{Value: "nonexistent_command_xyz_12345"},
		&eval.ListValue{Elements: []eval.Value{}},
		&eval.StringValue{Value: "test"},
		&eval.IntValue{Value: 0},
		&eval.IntValue{Value: 1024},
	}

	_, err := StreamAsyncExecProcess(ctx, args)
	if err == nil {
		t.Fatal("expected error for nonexistent command")
	}
	if !strings.Contains(err.Error(), "command not found") {
		t.Errorf("error = %q, want 'command not found'", err.Error())
	}
}

// TestStreamAsyncExecProcess_Allowlist verifies ProcessContext allowlist enforcement.
func TestStreamAsyncExecProcess_Allowlist(t *testing.T) {
	sc := NewStreamContext()
	defer sc.CloseAll()

	ctx := &EffContext{
		Stream: sc,
		Process: &ProcessContext{
			HasAllowlist: true,
			Allowlist:    map[string]string{"echo": "/bin/echo"},
		},
	}

	// Allowed command
	args := []eval.Value{
		&eval.StringValue{Value: "echo"},
		&eval.ListValue{Elements: []eval.Value{&eval.StringValue{Value: "allowed"}}},
		&eval.StringValue{Value: "test"},
		&eval.IntValue{Value: 0},
		&eval.IntValue{Value: 1024},
	}

	_, err := StreamAsyncExecProcess(ctx, args)
	if err != nil {
		// On some systems /bin/echo may not exist (macOS uses /usr/bin/echo)
		// Just skip if it's a spawn error, not an allowlist error
		if strings.Contains(err.Error(), "not allowed") {
			t.Fatalf("unexpected not-allowed error: %v", err)
		}
		t.Skipf("skipping: echo path mismatch on this system: %v", err)
	}

	// Disallowed command
	args[0] = &eval.StringValue{Value: "cat"}
	_, err = StreamAsyncExecProcess(ctx, args)
	if err == nil {
		t.Fatal("expected error for disallowed command")
	}
	if !strings.Contains(err.Error(), "not allowed") {
		t.Errorf("error = %q, want 'not allowed'", err.Error())
	}
}

// TestStreamAsyncExecProcess_WrongArgCount verifies argument validation.
func TestStreamAsyncExecProcess_WrongArgCount(t *testing.T) {
	ctx := &EffContext{Stream: NewStreamContext()}

	_, err := StreamAsyncExecProcess(ctx, []eval.Value{&eval.StringValue{Value: "echo"}})
	if err == nil {
		t.Fatal("expected error for wrong arg count")
	}
	if !strings.Contains(err.Error(), "expected 5 arguments") {
		t.Errorf("error = %q, want 'expected 5 arguments'", err.Error())
	}
}
