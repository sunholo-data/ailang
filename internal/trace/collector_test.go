package trace

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestNewCollector(t *testing.T) {
	c := NewCollector()
	if c == nil {
		t.Fatal("NewCollector returned nil")
	}
	if !c.Enabled() {
		t.Error("new collector should be enabled")
	}
	if len(c.Events()) != 0 {
		t.Errorf("new collector should have 0 events, got %d", len(c.Events()))
	}
}

func TestNilCollector(t *testing.T) {
	var c *Collector
	if c.Enabled() {
		t.Error("nil collector should not be enabled")
	}
	if c.Events() != nil {
		t.Error("nil collector Events() should return nil")
	}
	// These should not panic on nil
	c.RecordModuleStart("test", nil)
	c.RecordModuleEnd("test", 0)
	c.RecordFunctionEnter("f", nil)
	c.RecordFunctionExit("f", "")
	c.RecordEffect("IO", "println", nil, "")
	c.RecordContractCheck("requires", true, "", "", "")
	c.RecordBudgetDelta("IO", 1, 5, 4, 1)
	c.RecordError("msg", "loc")
}

func TestRecordModuleStartEnd(t *testing.T) {
	c := NewCollector()
	c.RecordModuleStart("examples/hello", []string{"IO", "FS"})
	c.RecordModuleEnd("examples/hello", 1000000)

	events := c.Events()
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}

	start := events[0]
	if start.Event != EventModuleStart {
		t.Errorf("expected module_start, got %s", start.Event)
	}
	if start.Module == nil {
		t.Fatal("module payload is nil")
	}
	if start.Module.Name != "examples/hello" {
		t.Errorf("expected module name 'examples/hello', got %q", start.Module.Name)
	}
	if len(start.Module.Caps) != 2 {
		t.Errorf("expected 2 caps, got %d", len(start.Module.Caps))
	}
	if start.Version != "1.0" {
		t.Errorf("expected version 1.0, got %s", start.Version)
	}

	end := events[1]
	if end.Event != EventModuleEnd {
		t.Errorf("expected module_end, got %s", end.Event)
	}
	if end.Module.DurationNS != 1000000 {
		t.Errorf("expected duration 1000000, got %d", end.Module.DurationNS)
	}
}

func TestRecordFunctionEnterExit(t *testing.T) {
	c := NewCollector()
	c.RecordFunctionEnter("factorial", []string{"5"})
	c.RecordFunctionEnter("factorial", []string{"4"})
	c.RecordFunctionExit("factorial", "24")
	c.RecordFunctionExit("factorial", "120")

	events := c.Events()
	if len(events) != 4 {
		t.Fatalf("expected 4 events, got %d", len(events))
	}

	// Depth tracking: enter increments, exit decrements
	if events[0].Depth != 1 {
		t.Errorf("first enter depth: expected 1, got %d", events[0].Depth)
	}
	if events[1].Depth != 2 {
		t.Errorf("second enter depth: expected 2, got %d", events[1].Depth)
	}
	if events[2].Depth != 2 {
		t.Errorf("first exit depth: expected 2, got %d", events[2].Depth)
	}
	if events[3].Depth != 1 {
		t.Errorf("second exit depth: expected 1, got %d", events[3].Depth)
	}

	// Check function name and args
	if events[0].Function.Name != "factorial" {
		t.Errorf("expected function name 'factorial', got %q", events[0].Function.Name)
	}
	if len(events[0].Function.Args) != 1 || events[0].Function.Args[0] != "5" {
		t.Errorf("expected args [5], got %v", events[0].Function.Args)
	}

	// Check exit has result
	if events[2].Function.Result != "24" {
		t.Errorf("expected result '24', got %q", events[2].Function.Result)
	}

	// Duration should be non-negative
	if events[2].Function.DurationNS < 0 {
		t.Errorf("expected non-negative duration, got %d", events[2].Function.DurationNS)
	}
}

func TestRecordEffect(t *testing.T) {
	c := NewCollector()
	c.RecordEffect("IO", "println", []string{"\"hello\""}, "()")

	events := c.Events()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	e := events[0]
	if e.Event != EventEffect {
		t.Errorf("expected effect event, got %s", e.Event)
	}
	if e.Effect == nil {
		t.Fatal("effect payload nil")
	}
	if e.Effect.EffectName != "IO" {
		t.Errorf("expected effect IO, got %s", e.Effect.EffectName)
	}
	if e.Effect.OpName != "println" {
		t.Errorf("expected op println, got %s", e.Effect.OpName)
	}
	if e.Effect.Result != "()" {
		t.Errorf("expected result (), got %s", e.Effect.Result)
	}
}

func TestRecordEffect_NonDeterministicFlag(t *testing.T) {
	c := NewCollector()

	// Deterministic effect (IO.println) should have nil Deterministic
	c.RecordEffect("IO", "println", []string{"\"hello\""}, "()")
	// Non-deterministic effect (Clock.now) should have Deterministic=false
	c.RecordEffect("Clock", "now", nil, "1234567890")

	events := c.Events()
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}

	// IO.println: Deterministic should be nil (not flagged)
	if events[0].Effect.Deterministic != nil {
		t.Errorf("IO.println should have nil Deterministic, got %v", *events[0].Effect.Deterministic)
	}

	// Clock.now: Deterministic should be false
	if events[1].Effect.Deterministic == nil {
		t.Fatal("Clock.now should have non-nil Deterministic flag")
	}
	if *events[1].Effect.Deterministic != false {
		t.Errorf("Clock.now Deterministic should be false, got %v", *events[1].Effect.Deterministic)
	}
}

func TestRecordContractCheck(t *testing.T) {
	c := NewCollector()
	c.RecordContractCheck("requires", true, "x > 0", "main.ail:5:3", "factorial")
	c.RecordContractCheck("ensures", false, "result > 0", "main.ail:8:3", "factorial")

	events := c.Events()
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}

	req := events[0]
	if req.Contract.Kind != "requires" {
		t.Errorf("expected requires, got %s", req.Contract.Kind)
	}
	if !req.Contract.Passed {
		t.Error("expected passed=true")
	}
	if req.Contract.Function != "factorial" {
		t.Errorf("expected function factorial, got %s", req.Contract.Function)
	}

	ens := events[1]
	if ens.Contract.Kind != "ensures" {
		t.Errorf("expected ensures, got %s", ens.Contract.Kind)
	}
	if ens.Contract.Passed {
		t.Error("expected passed=false")
	}
}

func TestRecordBudgetDelta(t *testing.T) {
	c := NewCollector()
	c.RecordBudgetDelta("IO", 3, 5, 2, 3)

	events := c.Events()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	b := events[0]
	if b.Event != EventBudgetDelta {
		t.Errorf("expected budget_delta, got %s", b.Event)
	}
	if b.Budget.Used != 3 || b.Budget.Limit != 5 || b.Budget.Remaining != 2 || b.Budget.Physical != 3 {
		t.Errorf("unexpected budget values: %+v", b.Budget)
	}
}

func TestRecordError(t *testing.T) {
	c := NewCollector()
	c.RecordError("division by zero", "math.ail:12:5")

	events := c.Events()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	e := events[0]
	if e.Event != EventError {
		t.Errorf("expected error, got %s", e.Event)
	}
	if e.Error.Message != "division by zero" {
		t.Errorf("expected message 'division by zero', got %q", e.Error.Message)
	}
}

func TestTimestampsAreMonotonic(t *testing.T) {
	c := NewCollector()
	c.RecordFunctionEnter("a", nil)
	c.RecordFunctionEnter("b", nil)
	c.RecordFunctionExit("b", "")
	c.RecordFunctionExit("a", "")

	events := c.Events()
	for i := 1; i < len(events); i++ {
		if events[i].TimestampNS < events[i-1].TimestampNS {
			t.Errorf("timestamps not monotonic: event[%d]=%d < event[%d]=%d",
				i, events[i].TimestampNS, i-1, events[i-1].TimestampNS)
		}
	}
}

func TestDepthNeverNegative(t *testing.T) {
	c := NewCollector()
	// Exit without enter should not go negative
	c.RecordFunctionExit("orphan", "")
	c.RecordFunctionExit("orphan2", "")

	events := c.Events()
	for i, e := range events {
		if e.Depth < 0 {
			t.Errorf("event[%d] has negative depth %d", i, e.Depth)
		}
	}
}

func TestJSONLRoundTrip(t *testing.T) {
	c := NewCollector()
	c.RecordModuleStart("test/mod", []string{"IO"})
	c.RecordFunctionEnter("main", nil)
	c.RecordEffect("IO", "println", []string{"\"hello\""}, "()")
	c.RecordBudgetDelta("IO", 1, 5, 4, 1)
	c.RecordContractCheck("requires", true, "x > 0", "test:1:1", "main")
	c.RecordFunctionExit("main", "()")
	c.RecordModuleEnd("test/mod", 5000000)

	// Write JSONL
	var buf bytes.Buffer
	if err := WriteJSONL(&buf, c.Events()); err != nil {
		t.Fatalf("WriteJSONL failed: %v", err)
	}

	// Parse each line back
	decoder := json.NewDecoder(&buf)
	var parsed []TraceEvent
	for decoder.More() {
		var event TraceEvent
		if err := decoder.Decode(&event); err != nil {
			t.Fatalf("decode failed: %v", err)
		}
		parsed = append(parsed, event)
	}

	if len(parsed) != 7 {
		t.Fatalf("expected 7 parsed events, got %d", len(parsed))
	}

	// Verify event types round-trip
	expectedTypes := []EventType{
		EventModuleStart, EventFunctionEnter, EventEffect,
		EventBudgetDelta, EventContractCheck, EventFunctionExit, EventModuleEnd,
	}
	for i, expected := range expectedTypes {
		if parsed[i].Event != expected {
			t.Errorf("event[%d]: expected %s, got %s", i, expected, parsed[i].Event)
		}
		if parsed[i].Version != "1.0" {
			t.Errorf("event[%d]: expected version 1.0, got %s", i, parsed[i].Version)
		}
	}

	// Verify specific payload round-trips
	if parsed[0].Module.Name != "test/mod" {
		t.Errorf("module name didn't round-trip: %q", parsed[0].Module.Name)
	}
	if parsed[2].Effect.EffectName != "IO" || parsed[2].Effect.OpName != "println" {
		t.Errorf("effect didn't round-trip: %+v", parsed[2].Effect)
	}
	if parsed[4].Contract.Kind != "requires" || !parsed[4].Contract.Passed {
		t.Errorf("contract didn't round-trip: %+v", parsed[4].Contract)
	}
}

func TestMixedEventSequence(t *testing.T) {
	c := NewCollector()

	// Simulate a realistic execution trace
	c.RecordModuleStart("app", []string{"IO", "FS"})
	c.RecordFunctionEnter("processFile", []string{"\"input.txt\""})
	c.RecordEffect("FS", "readFile", []string{"\"input.txt\""}, "\"data...\"")
	c.RecordBudgetDelta("FS", 1, 3, 2, 1)
	c.RecordContractCheck("requires", true, "file exists", "app.ail:10:3", "processFile")
	c.RecordFunctionEnter("transform", []string{"\"data...\""})
	c.RecordFunctionExit("transform", "\"result\"")
	c.RecordEffect("IO", "println", []string{"\"result\""}, "()")
	c.RecordBudgetDelta("IO", 1, 5, 4, 1)
	c.RecordFunctionExit("processFile", "()")
	c.RecordModuleEnd("app", 10000000)

	events := c.Events()
	if len(events) != 11 {
		t.Fatalf("expected 11 events, got %d", len(events))
	}

	// All events should have version and timestamp
	for i, e := range events {
		if e.Version == "" {
			t.Errorf("event[%d] has no version", i)
		}
		if e.TimestampNS < 0 {
			t.Errorf("event[%d] has negative timestamp", i)
		}
	}
}
