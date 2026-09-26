package telemetry

import (
	"context"
	"testing"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/sunholo-data/ailang/internal/eval"
	ailtrace "github.com/sunholo-data/ailang/internal/trace"
)

// recordEffectSpans runs one op of each kind through the wrapper at the given
// tier and returns the span names it exported.
func recordEffectSpans(t *testing.T, tier ailtrace.Tier) []string {
	t.Helper()
	exp := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exp))
	t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })
	wrap := newEffectSpanWrapper(tp.Tracer("test-effects"), tier)

	ok := func() (eval.Value, error) { return &eval.UnitValue{}, nil }
	for _, op := range [][2]string{
		{"Debug", "log"}, {"IO", "println"}, {"IO", "eprint"}, {"IO", "flush"},
		{"IO", "readLine"}, {"Process", "exec"}, {"FS", "readFile"},
	} {
		if _, err := wrap(context.Background(), op[0], op[1], nil, ok); err != nil {
			t.Fatalf("%s.%s: %v", op[0], op[1], err)
		}
	}
	var names []string
	for _, s := range exp.GetSpans() {
		names = append(names, s.Name)
	}
	return names
}

func TestEffectSpanWrapper_StandardTierSkipsConsoleChatter(t *testing.T) {
	got := recordEffectSpans(t, ailtrace.TierStandard)
	want := []string{"effect.IO.readLine", "effect.Process.exec", "effect.FS.readFile"}
	if len(got) != len(want) {
		t.Fatalf("standard tier exported %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("standard tier exported %v, want %v", got, want)
		}
	}
}

func TestEffectSpanWrapper_DeepTierKeepsEverything(t *testing.T) {
	got := recordEffectSpans(t, ailtrace.TierDeep)
	if len(got) != 7 {
		t.Fatalf("deep tier exported %d spans (%v), want all 7", len(got), got)
	}
	if got[0] != "effect.Debug.log" || got[1] != "effect.IO.println" {
		t.Fatalf("deep tier dropped console spans: %v", got)
	}
}

// The skip must still run the effect: a println that is not traced is still
// printed.
func TestEffectSpanWrapper_SkippedOpStillExecutes(t *testing.T) {
	tp := sdktrace.NewTracerProvider()
	wrap := newEffectSpanWrapper(tp.Tracer("t"), ailtrace.TierStandard)
	ran := false
	_, err := wrap(context.Background(), "Debug", "log", nil, func() (eval.Value, error) {
		ran = true
		return &eval.UnitValue{}, nil
	})
	if err != nil || !ran {
		t.Fatalf("skipped op did not execute (ran=%v err=%v)", ran, err)
	}
}
