package eval

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/core"
)

// #1317: deep evaluation continues on fresh goroutines every evalSegmentLevels
// nested evalCore levels. These tests shrink the segment so a small program hops
// hundreds of times, and check that hopping is invisible: same result, same
// errors, panics re-raised on the caller's goroutine, hook paired with cleanup.

func withSegmentLevels(t *testing.T, n int) {
	t.Helper()
	saved := evalSegmentLevels
	evalSegmentLevels = n
	t.Cleanup(func() { evalSegmentLevels = saved })
}

func TestStackHopPreservesResult(t *testing.T) {
	withSegmentLevels(t, 16)
	ev := newSumEvaluator(10000)
	var enters, exits int
	ev.SetStackHopHook(func() func() {
		enters++
		return func() { exits++ }
	})
	v, err := ev.evalCore(sumProgram(2000))
	if err != nil {
		t.Fatal(err)
	}
	if got := v.(*IntValue).Value; got != 2001000 {
		t.Fatalf("sum(2000) = %d across hops, want 2001000", got)
	}
	if enters < 100 {
		t.Fatalf("instrument failure: only %d hops at segment 16 — the test is not exercising hopping", enters)
	}
	if enters != exits {
		t.Fatalf("hook entered %d times but cleaned up %d", enters, exits)
	}
	if ev.evalDepth != 0 || ev.segmentBase != 0 {
		t.Fatalf("depth state leaked: evalDepth=%d segmentBase=%d", ev.evalDepth, ev.segmentBase)
	}
}

// RT_REC_003 raised on a continuation goroutine must reach the caller as the
// same ordinary error, and leave the counters balanced for the next call.
func TestStackHopPropagatesRecursionError(t *testing.T) {
	withSegmentLevels(t, 16)
	ev := newSumEvaluator(500)
	_, err := ev.evalCore(sumProgram(2000))
	if err == nil || !strings.Contains(err.Error(), "RT_REC_003") {
		t.Fatalf("want RT_REC_003 across hops, got %v", err)
	}
	if ev.recursionDepth != 0 || ev.evalDepth != 0 || ev.segmentBase != 0 {
		t.Fatalf("counters leaked after error: rec=%d eval=%d base=%d", ev.recursionDepth, ev.evalDepth, ev.segmentBase)
	}
}

// A panic deep inside a hopped segment must surface on the caller's goroutine
// with its original value, so recover() above evalCore sees it exactly as it
// would without hopping (an unrecovered panic on a child goroutine would kill
// the process instead).
func TestStackHopReRaisesPanicOnCaller(t *testing.T) {
	withSegmentLevels(t, 4)
	ev := newSumEvaluator(10000)
	ev.env.Set("boom", &BuiltinFunction{Name: "boom", Fn: func([]Value) (Value, error) {
		panic("boom-sentinel")
	}})
	// Nest the panicking call under enough lets to force several hops.
	var expr core.CoreExpr = &core.App{Func: &core.Var{Name: "boom"}, Args: []core.CoreExpr{&core.Lit{Kind: core.IntLit, Value: 1}}}
	for i := 0; i < 40; i++ {
		expr = &core.Let{Name: "x", Value: &core.Lit{Kind: core.IntLit, Value: i}, Body: expr}
	}
	defer func() {
		if p := recover(); p != "boom-sentinel" {
			t.Fatalf("recovered %v on the caller, want boom-sentinel", p)
		}
	}()
	_, _ = ev.evalCore(expr)
	t.Fatal("panic did not propagate")
}
