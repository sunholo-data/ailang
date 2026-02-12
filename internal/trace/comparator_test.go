package trace

import (
	"testing"
)

func TestCompareTraces_Identical(t *testing.T) {
	events := []TraceEvent{
		{Version: "1.0", Event: EventFunctionEnter, TimestampNS: 1000, Depth: 1,
			Function: &FunctionEvent{Name: "main", Args: []string{"5"}}},
		{Version: "1.0", Event: EventFunctionExit, TimestampNS: 2000, Depth: 1,
			Function: &FunctionEvent{Name: "main", Result: "120"}},
	}

	result := CompareTraces(events, events)
	if !result.Match {
		t.Errorf("expected match, got %d mismatches: %+v", len(result.Mismatches), result.Mismatches)
	}
}

func TestCompareTraces_TimestampsDiffer(t *testing.T) {
	baseline := []TraceEvent{
		{Version: "1.0", Event: EventFunctionEnter, TimestampNS: 1000, Depth: 1,
			Function: &FunctionEvent{Name: "main"}},
		{Version: "1.0", Event: EventFunctionExit, TimestampNS: 5000, Depth: 1,
			Function: &FunctionEvent{Name: "main", Result: "()", DurationNS: 4000}},
	}
	replay := []TraceEvent{
		{Version: "1.0", Event: EventFunctionEnter, TimestampNS: 9999, Depth: 1,
			Function: &FunctionEvent{Name: "main"}},
		{Version: "1.0", Event: EventFunctionExit, TimestampNS: 88888, Depth: 1,
			Function: &FunctionEvent{Name: "main", Result: "()", DurationNS: 77777}},
	}

	result := CompareTraces(baseline, replay)
	if !result.Match {
		t.Errorf("timestamps/durations should be ignored, got %d mismatches: %+v",
			len(result.Mismatches), result.Mismatches)
	}
}

func TestCompareTraces_DifferentResult(t *testing.T) {
	baseline := []TraceEvent{
		{Version: "1.0", Event: EventFunctionExit, Depth: 1,
			Function: &FunctionEvent{Name: "factorial", Result: "120"}},
	}
	replay := []TraceEvent{
		{Version: "1.0", Event: EventFunctionExit, Depth: 1,
			Function: &FunctionEvent{Name: "factorial", Result: "119"}},
	}

	result := CompareTraces(baseline, replay)
	if result.Match {
		t.Fatal("expected mismatch for different result")
	}
	if len(result.Mismatches) != 1 {
		t.Fatalf("expected 1 mismatch, got %d", len(result.Mismatches))
	}
	mm := result.Mismatches[0]
	if mm.Field != "function.result" {
		t.Errorf("expected field 'function.result', got %q", mm.Field)
	}
	if mm.Expected != "120" || mm.Actual != "119" {
		t.Errorf("expected 120/119, got %s/%s", mm.Expected, mm.Actual)
	}
}

func TestCompareTraces_DifferentEventType(t *testing.T) {
	baseline := []TraceEvent{
		{Version: "1.0", Event: EventFunctionEnter, Depth: 1,
			Function: &FunctionEvent{Name: "main"}},
	}
	replay := []TraceEvent{
		{Version: "1.0", Event: EventEffect, Depth: 1,
			Effect: &EffectEvent{EffectName: "IO", OpName: "println"}},
	}

	result := CompareTraces(baseline, replay)
	if result.Match {
		t.Fatal("expected mismatch for different event type")
	}
	if result.Mismatches[0].Field != "event" {
		t.Errorf("expected field 'event', got %q", result.Mismatches[0].Field)
	}
}

func TestCompareTraces_DifferentLength(t *testing.T) {
	baseline := []TraceEvent{
		{Version: "1.0", Event: EventFunctionEnter, Depth: 1,
			Function: &FunctionEvent{Name: "main"}},
		{Version: "1.0", Event: EventFunctionExit, Depth: 1,
			Function: &FunctionEvent{Name: "main"}},
	}
	replay := []TraceEvent{
		{Version: "1.0", Event: EventFunctionEnter, Depth: 1,
			Function: &FunctionEvent{Name: "main"}},
	}

	result := CompareTraces(baseline, replay)
	if result.Match {
		t.Fatal("expected mismatch for different length")
	}
	if result.BaselineN != 2 || result.ReplayN != 1 {
		t.Errorf("expected counts 2/1, got %d/%d", result.BaselineN, result.ReplayN)
	}

	// Should have an event_count mismatch
	found := false
	for _, mm := range result.Mismatches {
		if mm.Field == "event_count" {
			found = true
		}
	}
	if !found {
		t.Error("expected event_count mismatch")
	}
}

func TestCompareTraces_Empty(t *testing.T) {
	result := CompareTraces(nil, nil)
	if !result.Match {
		t.Error("two empty traces should match")
	}

	result2 := CompareTraces([]TraceEvent{}, []TraceEvent{})
	if !result2.Match {
		t.Error("two empty slice traces should match")
	}
}

func TestCompareTraces_EffectMismatch(t *testing.T) {
	baseline := []TraceEvent{
		{Version: "1.0", Event: EventEffect, Depth: 1,
			Effect: &EffectEvent{EffectName: "IO", OpName: "println", Result: "()"}},
	}
	replay := []TraceEvent{
		{Version: "1.0", Event: EventEffect, Depth: 1,
			Effect: &EffectEvent{EffectName: "IO", OpName: "println", Result: "error: budget exceeded"}},
	}

	result := CompareTraces(baseline, replay)
	if result.Match {
		t.Fatal("expected mismatch for different effect result")
	}
	if result.Mismatches[0].Field != "effect.result" {
		t.Errorf("expected field 'effect.result', got %q", result.Mismatches[0].Field)
	}
}

func TestCompareTraces_ContractMismatch(t *testing.T) {
	baseline := []TraceEvent{
		{Version: "1.0", Event: EventContractCheck, Depth: 1,
			Contract: &ContractEvent{Kind: "requires", Passed: true, Message: "x > 0"}},
	}
	replay := []TraceEvent{
		{Version: "1.0", Event: EventContractCheck, Depth: 1,
			Contract: &ContractEvent{Kind: "requires", Passed: false, Message: "x > 0"}},
	}

	result := CompareTraces(baseline, replay)
	if result.Match {
		t.Fatal("expected mismatch for different contract result")
	}
	if result.Mismatches[0].Field != "contract.passed" {
		t.Errorf("expected field 'contract.passed', got %q", result.Mismatches[0].Field)
	}
}

func TestCompareTraces_BudgetMismatch(t *testing.T) {
	baseline := []TraceEvent{
		{Version: "1.0", Event: EventBudgetDelta, Depth: 1,
			Budget: &BudgetEvent{Effect: "IO", Used: 3, Limit: 5, Remaining: 2}},
	}
	replay := []TraceEvent{
		{Version: "1.0", Event: EventBudgetDelta, Depth: 1,
			Budget: &BudgetEvent{Effect: "IO", Used: 4, Limit: 5, Remaining: 1}},
	}

	result := CompareTraces(baseline, replay)
	if result.Match {
		t.Fatal("expected mismatch for different budget")
	}

	fields := make(map[string]bool)
	for _, mm := range result.Mismatches {
		fields[mm.Field] = true
	}
	if !fields["budget.used"] {
		t.Error("expected budget.used mismatch")
	}
	if !fields["budget.remaining"] {
		t.Error("expected budget.remaining mismatch")
	}
}

func TestCompareTraces_DepthMismatch(t *testing.T) {
	baseline := []TraceEvent{
		{Version: "1.0", Event: EventFunctionEnter, Depth: 1,
			Function: &FunctionEvent{Name: "main"}},
	}
	replay := []TraceEvent{
		{Version: "1.0", Event: EventFunctionEnter, Depth: 2,
			Function: &FunctionEvent{Name: "main"}},
	}

	result := CompareTraces(baseline, replay)
	if result.Match {
		t.Fatal("expected mismatch for different depth")
	}
	if result.Mismatches[0].Field != "depth" {
		t.Errorf("expected field 'depth', got %q", result.Mismatches[0].Field)
	}
}

func TestCompareTraces_MultipleMismatches(t *testing.T) {
	baseline := []TraceEvent{
		{Version: "1.0", Event: EventFunctionEnter, Depth: 1,
			Function: &FunctionEvent{Name: "a"}},
		{Version: "1.0", Event: EventFunctionExit, Depth: 1,
			Function: &FunctionEvent{Name: "a", Result: "1"}},
	}
	replay := []TraceEvent{
		{Version: "1.0", Event: EventFunctionEnter, Depth: 1,
			Function: &FunctionEvent{Name: "b"}},
		{Version: "1.0", Event: EventFunctionExit, Depth: 1,
			Function: &FunctionEvent{Name: "b", Result: "2"}},
	}

	result := CompareTraces(baseline, replay)
	if result.Match {
		t.Fatal("expected mismatches")
	}
	// Should have mismatches for both events (name + result diffs)
	if len(result.Mismatches) < 2 {
		t.Errorf("expected at least 2 mismatches, got %d", len(result.Mismatches))
	}
}

func TestCompareTraces_NonDeterministicEffectSkipped(t *testing.T) {
	nondet := false
	baseline := []TraceEvent{
		{Version: "1.0", Event: EventEffect, Depth: 1,
			Effect: &EffectEvent{
				EffectName:    "Clock",
				OpName:        "now",
				Result:        "1000",
				Deterministic: &nondet, // flagged non-deterministic
			}},
	}
	replay := []TraceEvent{
		{Version: "1.0", Event: EventEffect, Depth: 1,
			Effect: &EffectEvent{
				EffectName:    "Clock",
				OpName:        "now",
				Result:        "9999", // different result — should be tolerated
				Deterministic: &nondet,
			}},
	}

	result := CompareTraces(baseline, replay)
	if !result.Match {
		t.Fatalf("expected match (non-deterministic effect result should be skipped), got %d mismatches: %v", len(result.Mismatches), result.Mismatches)
	}
}

func TestCompareTraces_DeterministicEffectNotSkipped(t *testing.T) {
	baseline := []TraceEvent{
		{Version: "1.0", Event: EventEffect, Depth: 1,
			Effect: &EffectEvent{EffectName: "IO", OpName: "println", Result: "()"}},
	}
	replay := []TraceEvent{
		{Version: "1.0", Event: EventEffect, Depth: 1,
			Effect: &EffectEvent{EffectName: "IO", OpName: "println", Result: "error"}},
	}

	result := CompareTraces(baseline, replay)
	if result.Match {
		t.Fatal("expected mismatch for deterministic effect with different result")
	}
}

func TestIsNonDeterministic(t *testing.T) {
	cases := []struct {
		effect, op string
		want       bool
	}{
		{"Clock", "now", true},
		{"Clock", "sleep", true},
		{"IO", "readLine", true},
		{"Net", "httpGet", true},
		{"IO", "println", false},
		{"FS", "readFile", false},
	}
	for _, tc := range cases {
		got := IsNonDeterministic(tc.effect, tc.op)
		if got != tc.want {
			t.Errorf("IsNonDeterministic(%q, %q) = %v, want %v", tc.effect, tc.op, got, tc.want)
		}
	}
}
