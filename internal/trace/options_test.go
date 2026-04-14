package trace

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestParseTier(t *testing.T) {
	tests := []struct {
		in      string
		want    Tier
		wantErr bool
	}{
		{"off", TierOff, false},
		{"OFF", TierOff, false},
		{"none", TierOff, false},
		{"disabled", TierOff, false},
		{"standard", TierStandard, false},
		{"default", TierStandard, false},
		{"", TierStandard, false},
		{"deep", TierDeep, false},
		{"full", TierDeep, false},
		{"all", TierDeep, false},
		{"bogus", TierStandard, true},
	}
	for _, tc := range tests {
		got, err := ParseTier(tc.in)
		if (err != nil) != tc.wantErr {
			t.Errorf("ParseTier(%q) err=%v, wantErr=%v", tc.in, err, tc.wantErr)
		}
		if got != tc.want {
			t.Errorf("ParseTier(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestTierString(t *testing.T) {
	cases := map[Tier]string{
		TierOff:      "off",
		TierStandard: "standard",
		TierDeep:     "deep",
	}
	for tier, want := range cases {
		if got := tier.String(); got != want {
			t.Errorf("Tier(%d).String() = %q, want %q", int(tier), got, want)
		}
	}
}

func TestTierFromEnv(t *testing.T) {
	// Snapshot and restore env.
	oldTrace := os.Getenv("AILANG_TRACE")
	oldNoTrace := os.Getenv("AILANG_NO_TRACE")
	defer func() {
		os.Setenv("AILANG_TRACE", oldTrace)
		os.Setenv("AILANG_NO_TRACE", oldNoTrace)
	}()

	os.Unsetenv("AILANG_TRACE")
	os.Unsetenv("AILANG_NO_TRACE")
	if got, _ := TierFromEnv(); got != TierStandard {
		t.Errorf("default: got %v, want standard", got)
	}

	os.Setenv("AILANG_NO_TRACE", "1")
	if got, _ := TierFromEnv(); got != TierOff {
		t.Errorf("AILANG_NO_TRACE=1: got %v, want off", got)
	}

	// AILANG_TRACE wins over AILANG_NO_TRACE
	os.Setenv("AILANG_TRACE", "deep")
	if got, _ := TierFromEnv(); got != TierDeep {
		t.Errorf("AILANG_TRACE=deep (with AILANG_NO_TRACE=1): got %v, want deep", got)
	}

	os.Setenv("AILANG_TRACE", "standard")
	if got, _ := TierFromEnv(); got != TierStandard {
		t.Errorf("AILANG_TRACE=standard: got %v, want standard", got)
	}
}

func TestResolveTier_CLIWinsOverEnv(t *testing.T) {
	oldTrace := os.Getenv("AILANG_TRACE")
	defer os.Setenv("AILANG_TRACE", oldTrace)

	os.Setenv("AILANG_TRACE", "off")
	got, err := ResolveTier("deep")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != TierDeep {
		t.Errorf("CLI=deep, env=off: got %v, want deep", got)
	}
}

// makeFixture returns a canned event stream covering module start, a nested
// function call with a top-level effect and a nested effect, and a module end.
func makeFixture() []TraceEvent {
	return []TraceEvent{
		{Version: "1.0", Event: EventModuleStart, TimestampNS: 100, Depth: 0,
			Module: &ModuleEvent{Name: "test", Caps: []string{}}},
		// Top-level effect (Depth=1): kept in standard
		{Version: "1.0", Event: EventEffect, TimestampNS: 200, Depth: 1,
			Effect: &EffectEvent{EffectName: "IO", OpName: "println", Args: []string{"hi"}}},
		// Function call
		{Version: "1.0", Event: EventFunctionEnter, TimestampNS: 300, Depth: 1,
			Function: &FunctionEvent{Name: "helper", Args: []string{"1"}}},
		// Nested effect (Depth=2): dropped in standard
		{Version: "1.0", Event: EventEffect, TimestampNS: 400, Depth: 2,
			Effect: &EffectEvent{EffectName: "IO", OpName: "println", Args: []string{"inner"}}},
		{Version: "1.0", Event: EventFunctionExit, TimestampNS: 500, Depth: 1,
			Function: &FunctionEvent{Name: "helper", Result: "()", DurationNS: 200}},
		{Version: "1.0", Event: EventModuleEnd, TimestampNS: 600, Depth: 0,
			Module: &ModuleEvent{Name: "test", DurationNS: 500}},
	}
}

func TestEmitOTELSpansWithOptions_TierOff(t *testing.T) {
	exporter, tp := setupTestTracer()
	defer tp.Shutdown(context.Background())
	tracer := tp.Tracer("test")

	err := EmitOTELSpansWithOptions(context.Background(), tracer, makeFixture(), time.Now(),
		TracingOptions{Tier: TierOff})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := len(exporter.GetSpans()); got != 0 {
		t.Errorf("TierOff emitted %d spans, want 0", got)
	}
}

func TestEmitOTELSpansWithOptions_TierStandard(t *testing.T) {
	exporter, tp := setupTestTracer()
	defer tp.Shutdown(context.Background())
	tracer := tp.Tracer("test")

	err := EmitOTELSpansWithOptions(context.Background(), tracer, makeFixture(), time.Now(),
		TracingOptions{Tier: TierStandard})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	spans := exporter.GetSpans()
	// Expect: module span + 1 top-level effect = 2. No function span, no nested effect.
	gotNames := make(map[string]int)
	for _, s := range spans {
		gotNames[s.Name]++
	}
	if gotNames["eval.module.test"] != 1 {
		t.Errorf("want 1 eval.module.test span, got %d", gotNames["eval.module.test"])
	}
	if gotNames["eval.effect.IO.println"] != 1 {
		t.Errorf("want 1 top-level eval.effect.IO.println span, got %d (nested effect leaked?)", gotNames["eval.effect.IO.println"])
	}
	for name := range gotNames {
		if name == "eval.function.helper" {
			t.Errorf("TierStandard leaked function span %q", name)
		}
	}
}

func TestEmitOTELSpansWithOptions_TierDeep(t *testing.T) {
	exporter, tp := setupTestTracer()
	defer tp.Shutdown(context.Background())
	tracer := tp.Tracer("test")

	err := EmitOTELSpansWithOptions(context.Background(), tracer, makeFixture(), time.Now(),
		TracingOptions{Tier: TierDeep})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	gotNames := make(map[string]int)
	for _, s := range exporter.GetSpans() {
		gotNames[s.Name]++
	}
	if gotNames["eval.module.test"] != 1 {
		t.Errorf("want 1 module span, got %d", gotNames["eval.module.test"])
	}
	if gotNames["eval.function.helper"] != 1 {
		t.Errorf("want 1 function span, got %d", gotNames["eval.function.helper"])
	}
	if gotNames["eval.effect.IO.println"] != 2 {
		t.Errorf("want 2 effect spans (top-level + nested), got %d", gotNames["eval.effect.IO.println"])
	}
}
