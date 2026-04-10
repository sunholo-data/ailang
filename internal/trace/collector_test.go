package trace

import (
	"bytes"
	"encoding/hex"
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

func TestOnEventCallback(t *testing.T) {
	c := NewCollector()

	var streamed []TraceEvent
	c.OnEvent = func(evt TraceEvent) {
		streamed = append(streamed, evt)
	}

	c.RecordModuleStart("test", []string{"IO"})
	c.RecordFunctionEnter("main", []string{"5"})
	c.RecordEffect("IO", "println", []string{"\"hello\""}, "()")
	c.RecordContractCheck("requires", true, "x > 0", "test.ail:1:1", "main")
	c.RecordBudgetDelta("IO", 1, 5, 4, 1)
	c.RecordError("oops", "test.ail:2:1")
	c.RecordFunctionExit("main", "()")
	c.RecordModuleEnd("test", 1000)

	// OnEvent should have been called for each event
	if len(streamed) != 8 {
		t.Fatalf("expected 8 streamed events, got %d", len(streamed))
	}

	// Verify event types arrived in order
	expectedTypes := []EventType{
		EventModuleStart, EventFunctionEnter, EventEffect,
		EventContractCheck, EventBudgetDelta, EventError,
		EventFunctionExit, EventModuleEnd,
	}
	for i, et := range expectedTypes {
		if streamed[i].Event != et {
			t.Errorf("streamed[%d]: expected %s, got %s", i, et, streamed[i].Event)
		}
	}

	// Accumulated events should match streamed events
	accumulated := c.Events()
	if len(accumulated) != len(streamed) {
		t.Fatalf("accumulated %d != streamed %d", len(accumulated), len(streamed))
	}
}

func TestOnEventNilIsZeroCost(t *testing.T) {
	c := NewCollector()
	// OnEvent is nil by default — should not panic
	c.RecordModuleStart("test", nil)
	c.RecordFunctionEnter("f", nil)
	c.RecordFunctionExit("f", "")
	c.RecordModuleEnd("test", 0)

	if len(c.Events()) != 4 {
		t.Fatalf("expected 4 events, got %d", len(c.Events()))
	}
}

func TestSpanIDs(t *testing.T) {
	c := NewCollector()

	// Verify trace ID is a valid 32-hex-char string
	if len(c.traceID) != 32 {
		t.Fatalf("traceID length: expected 32, got %d", len(c.traceID))
	}
	if _, err := hex.DecodeString(c.traceID); err != nil {
		t.Fatalf("traceID not valid hex: %v", err)
	}

	// Record nested function calls
	c.RecordModuleStart("app", []string{"IO"})
	c.RecordFunctionEnter("outer", nil)
	c.RecordEffect("IO", "println", nil, "()")
	c.RecordFunctionEnter("inner", nil)
	c.RecordFunctionExit("inner", "42")
	c.RecordFunctionExit("outer", "()")
	c.RecordModuleEnd("app", 1000)

	events := c.Events()
	if len(events) != 7 {
		t.Fatalf("expected 7 events, got %d", len(events))
	}

	// All events should have the same trace ID
	for i, e := range events {
		if e.TraceID != c.traceID {
			t.Errorf("event[%d] traceID mismatch: expected %s, got %s", i, c.traceID, e.TraceID)
		}
	}

	// All span IDs should be valid 16-hex-char strings
	for i, e := range events {
		if e.SpanID == "" {
			t.Errorf("event[%d] (%s) has empty span_id", i, e.Event)
			continue
		}
		if len(e.SpanID) != 16 {
			t.Errorf("event[%d] span_id length: expected 16, got %d", i, len(e.SpanID))
		}
		if _, err := hex.DecodeString(e.SpanID); err != nil {
			t.Errorf("event[%d] span_id not valid hex: %v", i, err)
		}
	}

	// module_start (index 0): has span_id, no parent (root)
	if events[0].ParentSpanID != "" {
		t.Errorf("module_start should have no parent, got %s", events[0].ParentSpanID)
	}
	moduleSpanID := events[0].SpanID

	// function_enter "outer" (index 1): parent = module span
	if events[1].ParentSpanID != moduleSpanID {
		t.Errorf("outer enter parent: expected %s (module), got %s", moduleSpanID, events[1].ParentSpanID)
	}
	outerSpanID := events[1].SpanID

	// effect (index 2): inherits current span (outer)
	if events[2].SpanID != outerSpanID {
		t.Errorf("effect span_id: expected %s (outer), got %s", outerSpanID, events[2].SpanID)
	}

	// function_enter "inner" (index 3): parent = outer
	if events[3].ParentSpanID != outerSpanID {
		t.Errorf("inner enter parent: expected %s (outer), got %s", outerSpanID, events[3].ParentSpanID)
	}
	innerSpanID := events[3].SpanID

	// function_exit "inner" (index 4): same span_id as enter
	if events[4].SpanID != innerSpanID {
		t.Errorf("inner exit span_id: expected %s, got %s", innerSpanID, events[4].SpanID)
	}

	// function_exit "outer" (index 5): same span_id as enter
	if events[5].SpanID != outerSpanID {
		t.Errorf("outer exit span_id: expected %s, got %s", outerSpanID, events[5].SpanID)
	}

	// module_end (index 6): same span_id as module_start
	if events[6].SpanID != moduleSpanID {
		t.Errorf("module_end span_id: expected %s, got %s", moduleSpanID, events[6].SpanID)
	}

	// All span IDs should be unique (except enter/exit pairs)
	seen := map[string]int{}
	for i, e := range events {
		seen[e.SpanID]++
		_ = i
	}
	// module span appears 2x (start+end), outer 3x (enter+effect+exit), inner 2x (enter+exit)
	if seen[moduleSpanID] != 2 {
		t.Errorf("module span count: expected 2, got %d", seen[moduleSpanID])
	}
	if seen[outerSpanID] != 3 {
		t.Errorf("outer span count: expected 3 (enter+effect+exit), got %d", seen[outerSpanID])
	}
	if seen[innerSpanID] != 2 {
		t.Errorf("inner span count: expected 2, got %d", seen[innerSpanID])
	}
}

func TestSpanIDsInJSONL(t *testing.T) {
	c := NewCollector()
	c.RecordFunctionEnter("test", nil)
	c.RecordFunctionExit("test", "ok")

	events := c.Events()
	// Verify span IDs survive JSON round-trip
	for _, e := range events {
		data, err := json.Marshal(e)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		var parsed TraceEvent
		if err := json.Unmarshal(data, &parsed); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if parsed.TraceID != e.TraceID {
			t.Errorf("trace_id round-trip: expected %s, got %s", e.TraceID, parsed.TraceID)
		}
		if parsed.SpanID != e.SpanID {
			t.Errorf("span_id round-trip: expected %s, got %s", e.SpanID, parsed.SpanID)
		}
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
