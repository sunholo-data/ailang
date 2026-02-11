package trace

import (
	"bytes"
	"strings"
	"testing"
)

func TestReadJSONL_RoundTrip(t *testing.T) {
	// Create events, write as JSONL, read back
	c := NewCollector()
	c.RecordModuleStart("test/mod", []string{"IO"})
	c.RecordFunctionEnter("main", []string{"5"})
	c.RecordEffect("IO", "println", []string{"\"hello\""}, "()")
	c.RecordFunctionExit("main", "()")
	c.RecordModuleEnd("test/mod", 5000)

	var buf bytes.Buffer
	if err := WriteJSONL(&buf, c.Events()); err != nil {
		t.Fatalf("WriteJSONL: %v", err)
	}

	events, err := ReadJSONL(&buf)
	if err != nil {
		t.Fatalf("ReadJSONL: %v", err)
	}

	if len(events) != 5 {
		t.Fatalf("expected 5 events, got %d", len(events))
	}

	if events[0].Event != EventModuleStart {
		t.Errorf("event[0]: expected module_start, got %s", events[0].Event)
	}
	if events[0].Module.Name != "test/mod" {
		t.Errorf("event[0] module name: expected test/mod, got %s", events[0].Module.Name)
	}
	if events[1].Function.Name != "main" {
		t.Errorf("event[1] function name: expected main, got %s", events[1].Function.Name)
	}
	if events[2].Effect.OpName != "println" {
		t.Errorf("event[2] effect op: expected println, got %s", events[2].Effect.OpName)
	}
}

func TestReadJSONL_Empty(t *testing.T) {
	events, err := ReadJSONL(strings.NewReader(""))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 0 {
		t.Errorf("expected 0 events, got %d", len(events))
	}
}

func TestReadJSONL_BlankLines(t *testing.T) {
	input := `{"version":"1.0","event":"function_enter","timestamp_ns":100,"function":{"name":"f"}}

{"version":"1.0","event":"function_exit","timestamp_ns":200,"function":{"name":"f"}}

`
	events, err := ReadJSONL(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 2 {
		t.Errorf("expected 2 events (blank lines skipped), got %d", len(events))
	}
}

func TestReadJSONL_MalformedJSON(t *testing.T) {
	input := `{"version":"1.0","event":"function_enter"}
{not valid json}
`
	_, err := ReadJSONL(strings.NewReader(input))
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
	if !strings.Contains(err.Error(), "line 2") {
		t.Errorf("expected error to mention line 2, got: %v", err)
	}
}

func TestReadJSONL_MissingVersion(t *testing.T) {
	input := `{"event":"function_enter","timestamp_ns":100}`
	_, err := ReadJSONL(strings.NewReader(input))
	if err == nil {
		t.Fatal("expected error for missing version")
	}
	if !strings.Contains(err.Error(), "missing version") {
		t.Errorf("expected 'missing version' error, got: %v", err)
	}
}

func TestReadJSONL_MissingEvent(t *testing.T) {
	input := `{"version":"1.0","timestamp_ns":100}`
	_, err := ReadJSONL(strings.NewReader(input))
	if err == nil {
		t.Fatal("expected error for missing event")
	}
	if !strings.Contains(err.Error(), "missing event") {
		t.Errorf("expected 'missing event' error, got: %v", err)
	}
}

func TestTraceMetadata(t *testing.T) {
	events := []TraceEvent{
		{Event: EventModuleStart, Module: &ModuleEvent{Name: "examples/hello", Caps: []string{"IO", "FS"}}},
		{Event: EventFunctionEnter, Function: &FunctionEvent{Name: "main"}},
	}

	name, caps := TraceMetadata(events)
	if name != "examples/hello" {
		t.Errorf("expected module name 'examples/hello', got %q", name)
	}
	if len(caps) != 2 || caps[0] != "IO" || caps[1] != "FS" {
		t.Errorf("expected caps [IO FS], got %v", caps)
	}
}

func TestTraceMetadata_NoModule(t *testing.T) {
	events := []TraceEvent{
		{Event: EventFunctionEnter, Function: &FunctionEvent{Name: "main"}},
	}

	name, caps := TraceMetadata(events)
	if name != "" {
		t.Errorf("expected empty module name, got %q", name)
	}
	if caps != nil {
		t.Errorf("expected nil caps, got %v", caps)
	}
}
