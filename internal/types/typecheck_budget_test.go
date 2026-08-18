package types

import (
	"errors"
	"math"
	"strings"
	"testing"
	"time"
)

// ailang#662 — the WASM type-check budget was a hardcoded wall-clock constant
// living entirely behind `//go:build js && wasm`, so no native test could
// reach it and none existed (`go list` reported zero native Go files carrying
// it). These arms exercise the real state machine: the guard now compiles into
// both targets and is inert until armed, so arming it here runs the same code
// the WASM bridge runs.
//
// The clock is injected rather than slept on. The defect under test IS a
// timing dependence, so a test that measured it by sleeping would inherit the
// flakiness it exists to pin.

// fakeClock drives the budget deterministically.
type fakeClock struct{ t time.Time }

func (c *fakeClock) now() time.Time          { return c.t }
func (c *fakeClock) advance(d time.Duration) { c.t = c.t.Add(d) }

// newTestBudget returns an isolated budget plus its clock, so arms never touch
// the package-level guard the type-checker uses.
func newTestBudget(limit time.Duration) (*typeCheckBudget, *fakeClock) {
	clk := &fakeClock{t: time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)}
	return &typeCheckBudget{now: clk.now, limit: limit}, clk
}

func asBudgetErr(t *testing.T, err error) WasmTypeCheckerBudgetExceededError {
	t.Helper()
	var e WasmTypeCheckerBudgetExceededError
	if !errors.As(err, &e) {
		t.Fatalf("expected WasmTypeCheckerBudgetExceededError, got %T: %v", err, err)
	}
	return e
}

// An unarmed guard must never fire. This is what keeps the native/CLI
// type-checker unbudgeted now that the mechanism compiles into both targets.
func TestBudgetUnarmedIsInert(t *testing.T) {
	b, clk := newTestBudget(time.Second)
	clk.advance(time.Hour)
	for i := 0; i < 10_000; i++ {
		if err := b.check(); err != nil {
			t.Fatalf("unarmed budget fired on step %d: %v", i, err)
		}
	}
	if b.steps != 0 {
		t.Fatalf("unarmed budget counted %d steps; want 0", b.steps)
	}
}

func TestBudgetDoesNotFireBeforeDeadline(t *testing.T) {
	b, clk := newTestBudget(2 * time.Second)
	b.begin("m")
	clk.advance(1999 * time.Millisecond)
	if err := b.check(); err != nil {
		t.Fatalf("budget fired at 1999ms of a 2s limit: %v", err)
	}
}

func TestBudgetFiresAfterDeadline(t *testing.T) {
	b, clk := newTestBudget(2 * time.Second)
	b.begin("docparse/services/docx_parser")
	clk.advance(2001 * time.Millisecond)
	err := b.check()
	if err == nil {
		t.Fatal("budget did not fire at 2001ms of a 2s limit")
	}
	e := asBudgetErr(t, err)
	if e.Module != "docparse/services/docx_parser" {
		t.Fatalf("error names module %q; want the module passed to begin()", e.Module)
	}
	if e.Budget != 2*time.Second {
		t.Fatalf("error reports budget %s; want 2s", e.Budget)
	}
}

// Once tripped the guard stays tripped, even if the clock goes backwards. The
// type-checker's own error-recovery loop keeps trying alternate inference
// paths after Unify returns an error; without stickiness loadModule never
// returns.
func TestBudgetIsSticky(t *testing.T) {
	b, clk := newTestBudget(time.Second)
	b.begin("m")
	clk.advance(2 * time.Second)
	if err := b.check(); err == nil {
		t.Fatal("budget did not fire")
	}
	clk.t = clk.t.Add(-2 * time.Second) // rewind well inside the deadline
	if err := b.check(); err == nil {
		t.Fatal("budget un-tripped when the clock went backwards; stickiness lost")
	}
}

// THE ailang#662 REGRESSION GUARD. Identical work and identical elapsed time,
// two different configured budgets, two different outcomes — which is exactly
// what "the limit is a host setting, not a property of the source" means. If
// the budget ever stops reaching the guard, this arm reds while every other
// arm here still passes.
func TestConfiguredBudgetChangesTheOutcome(t *testing.T) {
	const elapsed = 3 * time.Second

	tight, clkT := newTestBudget(1 * time.Second)
	tight.begin("output_formatter")
	clkT.advance(elapsed)
	tightErr := tight.check()

	roomy, clkR := newTestBudget(10 * time.Second)
	roomy.begin("output_formatter")
	clkR.advance(elapsed)
	roomyErr := roomy.check()

	if tightErr == nil {
		t.Fatal("3s of work passed a 1s budget")
	}
	if roomyErr != nil {
		t.Fatalf("3s of work failed a 10s budget: %v", roomyErr)
	}
	if tight.steps != roomy.steps {
		t.Fatalf("step counts diverged (%d vs %d) for identical work; the count must not depend on the budget",
			tight.steps, roomy.steps)
	}
}

// 0 disables the wall-clock guard entirely — the embedder's escape hatch for
// "I have already downloaded 40 MB, I can wait."
func TestZeroBudgetDisablesTheWallClockGuard(t *testing.T) {
	b, clk := newTestBudget(0)
	b.begin("m")
	clk.advance(24 * time.Hour)
	if err := b.check(); err != nil {
		t.Fatalf("budget of 0 fired after 24h; 0 must disable the wall-clock limit: %v", err)
	}
	if b.steps != 1 {
		t.Fatalf("steps not counted with the wall-clock guard disabled: got %d, want 1", b.steps)
	}
}

// begin() must reset per-check state ITSELF, or a module that blew the budget
// poisons every later load in the same session.
//
// Deliberately re-arms WITHOUT calling end() first. An earlier version of this
// arm did call end(), and end() also clears the sticky flag — so deleting
// begin()'s own `b.tripped = false` left the whole suite green (mutation drill
// M11, iteration 225). The end()-then-begin() path is the happy path the bridge
// takes; this arm exists for the one where end() never ran, which is precisely
// when a leaked trip flag would be permanent.
func TestBeginResetsPerCheckState(t *testing.T) {
	b, clk := newTestBudget(time.Second)
	b.begin("first")
	clk.advance(2 * time.Second)
	if err := b.check(); err == nil {
		t.Fatal("setup: budget did not fire")
	}
	if !b.tripped {
		t.Fatal("setup: budget fired without setting the sticky flag")
	}

	// Re-arm with no intervening end() — e.g. a host whose previous load
	// panicked out before EndWasmTypeCheck.
	b.begin("second")
	if b.steps != 0 {
		t.Fatalf("begin() left %d steps from the previous check", b.steps)
	}
	if b.tripped {
		t.Fatal("begin() left the sticky trip flag set; every later module would fail unconditionally")
	}
	if err := b.check(); err != nil {
		t.Fatalf("second check inherited the first check's failure: %v", err)
	}
	if b.module != "second" {
		t.Fatalf("begin() left module %q; want %q", b.module, "second")
	}
}

// end() must also clear the sticky flag — the bridge's normal path is
// begin/…/end/begin, and a trip surviving end() would fail the next load.
func TestEndClearsTheStickyFlag(t *testing.T) {
	b, clk := newTestBudget(time.Second)
	b.begin("first")
	clk.advance(2 * time.Second)
	if err := b.check(); err == nil {
		t.Fatal("setup: budget did not fire")
	}
	b.end()
	if b.tripped {
		t.Fatal("end() left the sticky trip flag set")
	}
	if b.active {
		t.Fatal("end() left the guard armed")
	}
}

// ailang#662 ask 3: consumption must be readable AFTER the check finishes,
// which is the only moment the bridge can report it.
func TestStatsSurviveEnd(t *testing.T) {
	b, clk := newTestBudget(10 * time.Second)
	b.begin("m")
	for i := 0; i < 7; i++ {
		clk.advance(100 * time.Millisecond)
		if err := b.check(); err != nil {
			t.Fatalf("unexpected budget failure: %v", err)
		}
	}
	b.end()
	if b.lastSteps != 7 {
		t.Fatalf("lastSteps = %d; want 7", b.lastSteps)
	}
	if b.lastElapsed != 700*time.Millisecond {
		t.Fatalf("lastElapsed = %s; want 700ms", b.lastElapsed)
	}
}

// The step count is the hardware-INDEPENDENT half of the report: it must be a
// function of the work alone. Two runs doing the same number of checks under
// wildly different clocks must agree.
func TestStepCountIsIndependentOfTheClock(t *testing.T) {
	run := func(perStep time.Duration) uint64 {
		b, clk := newTestBudget(0) // 0: no wall-clock limit, so neither run trips
		b.begin("m")
		for i := 0; i < 250; i++ {
			clk.advance(perStep)
			if err := b.check(); err != nil {
				t.Fatalf("unexpected failure with the wall-clock guard disabled: %v", err)
			}
		}
		b.end()
		return b.lastSteps
	}
	fast := run(time.Microsecond)
	slow := run(time.Second)
	if fast != slow {
		t.Fatalf("step count depends on the clock: %d (fast) vs %d (slow)", fast, slow)
	}
	if fast != 250 {
		t.Fatalf("step count = %d; want 250 (one per instrumented entry)", fast)
	}
}

// The message must carry the values only this path can produce — the budget
// actually in force and the step count reached — not merely "something went
// wrong". A reader on different hardware can act on the step count; they
// cannot act on ours.
func TestBudgetErrorMessageCarriesBudgetAndSteps(t *testing.T) {
	b, clk := newTestBudget(1500 * time.Millisecond)
	b.begin("docparse/services/docx_parser")
	for i := 0; i < 3; i++ {
		_ = b.check()
	}
	clk.advance(2 * time.Second)
	msg := asBudgetErr(t, b.check()).Error()

	for _, want := range []string{
		"1.5s",                          // the configured budget, not a hardcoded 2s
		"4 type-checker steps",          // the deterministic count reached
		"docparse/services/docx_parser", // the module
		"ailangSetTypeCheckBudget",      // how to change it
		"typeCheckMs / typeCheckSteps",  // how to watch headroom
	} {
		if !strings.Contains(msg, want) {
			t.Errorf("budget error message missing %q\n---\n%s", want, msg)
		}
	}
}

func TestSetWasmTypeCheckBudget(t *testing.T) {
	orig := WasmTypeCheckBudget()
	t.Cleanup(func() { _ = SetWasmTypeCheckBudget(orig) })

	if orig != DefaultWasmTypeCheckBudget {
		t.Fatalf("default budget = %s; want %s", orig, DefaultWasmTypeCheckBudget)
	}
	if err := SetWasmTypeCheckBudget(8 * time.Second); err != nil {
		t.Fatalf("setting a positive budget failed: %v", err)
	}
	if got := WasmTypeCheckBudget(); got != 8*time.Second {
		t.Fatalf("budget = %s after setting 8s", got)
	}
	if err := SetWasmTypeCheckBudget(0); err != nil {
		t.Fatalf("setting 0 (disable) failed: %v", err)
	}
	if got := WasmTypeCheckBudget(); got != 0 {
		t.Fatalf("budget = %s after setting 0", got)
	}
}

// A rejected budget must leave the previous one intact — a host that fat-fingers
// a negative value should not silently lose its guard.
func TestSetWasmTypeCheckBudgetRejectsNegativeAndKeepsPrevious(t *testing.T) {
	orig := WasmTypeCheckBudget()
	t.Cleanup(func() { _ = SetWasmTypeCheckBudget(orig) })

	if err := SetWasmTypeCheckBudget(5 * time.Second); err != nil {
		t.Fatalf("setup: %v", err)
	}
	err := SetWasmTypeCheckBudget(-1 * time.Second)
	if err == nil {
		t.Fatal("a negative budget was accepted")
	}
	if got := WasmTypeCheckBudget(); got != 5*time.Second {
		t.Fatalf("a rejected budget clobbered the previous one: now %s, want 5s", got)
	}
}

func TestParseBudgetMillis(t *testing.T) {
	tests := []struct {
		name    string
		ms      float64
		want    time.Duration
		wantErr string
	}{
		{name: "typical", ms: 8000, want: 8 * time.Second},
		{name: "zero disables", ms: 0, want: 0},
		{name: "fractional", ms: 1500.5, want: 1500500 * time.Microsecond},
		{name: "negative", ms: -1, wantErr: ">= 0 milliseconds"},
		{name: "NaN", ms: math.NaN(), wantErr: "NaN"},
		{name: "absurd", ms: 1e18, wantErr: "<= 86400000 milliseconds"},
		{name: "infinity", ms: math.Inf(1), wantErr: "<= 86400000 milliseconds"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseBudgetMillis(tt.ms)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("ParseBudgetMillis(%v) = %s, want error containing %q", tt.ms, got, tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error %q does not contain %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseBudgetMillis(%v) errored: %v", tt.ms, err)
			}
			if got != tt.want {
				t.Fatalf("ParseBudgetMillis(%v) = %s; want %s", tt.ms, got, tt.want)
			}
		})
	}
}

// The hooks the type-checker actually calls must route to the state machine.
// Without this, the extraction could leave checkWasmBudget/wasmDepthEnter
// wired to nothing and every arm above would still pass.
func TestTypeCheckerHooksRouteToTheBudget(t *testing.T) {
	origNow, origLimit := wasmBudget.now, wasmBudget.limit
	t.Cleanup(func() {
		wasmBudget.now, wasmBudget.limit = origNow, origLimit
		wasmBudget.end()
	})

	clk := &fakeClock{t: time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)}
	wasmBudget.now, wasmBudget.limit = clk.now, time.Second

	// Unarmed — the native/CLI production state.
	if err := checkWasmBudget(); err != nil {
		t.Fatalf("checkWasmBudget fired while unarmed: %v", err)
	}
	tc := &CoreTypeChecker{}
	if err := tc.wasmDepthEnter(nil); err != nil {
		t.Fatalf("wasmDepthEnter fired while unarmed: %v", err)
	}

	wasmBudget.begin("m")
	clk.advance(2 * time.Second)
	if err := checkWasmBudget(); err == nil {
		t.Fatal("checkWasmBudget did not reach the budget: no error past the deadline")
	}
	wasmBudget.begin("m")
	clk.advance(2 * time.Second)
	if err := tc.wasmDepthEnter(nil); err == nil {
		t.Fatal("wasmDepthEnter did not reach the budget: no error past the deadline")
	}
	tc.wasmDepthExit(nil) // retained for the unconditional defer in inferCore
}

// LastWasmTypeCheckStats is the exported accessor the WASM bridge calls to
// build the typeCheckMs / typeCheckSteps fields on every loadModule result;
// exercised here against the package-level guard the bridge actually uses.
func TestLastWasmTypeCheckStatsReportsThePackageGuard(t *testing.T) {
	origNow, origLimit := wasmBudget.now, wasmBudget.limit
	origSteps, origElapsed := wasmBudget.lastSteps, wasmBudget.lastElapsed
	t.Cleanup(func() {
		wasmBudget.now, wasmBudget.limit = origNow, origLimit
		wasmBudget.lastSteps, wasmBudget.lastElapsed = origSteps, origElapsed
		wasmBudget.end()
	})

	clk := &fakeClock{t: time.Date(2026, 8, 19, 0, 0, 0, 0, time.UTC)}
	wasmBudget.now, wasmBudget.limit = clk.now, 10*time.Second

	wasmBudget.begin("docparse/services/docx_parser")
	tc := &CoreTypeChecker{}
	for i := 0; i < 3; i++ {
		clk.advance(50 * time.Millisecond)
		if err := tc.wasmDepthEnter(nil); err != nil {
			t.Fatalf("unexpected budget failure: %v", err)
		}
		tc.wasmDepthExit(nil)
	}
	wasmBudget.end()

	steps, elapsed := LastWasmTypeCheckStats()
	if steps != 3 {
		t.Fatalf("LastWasmTypeCheckStats steps = %d; want 3", steps)
	}
	if elapsed != 150*time.Millisecond {
		t.Fatalf("LastWasmTypeCheckStats elapsed = %s; want 150ms", elapsed)
	}
}

// The bridge names the module, but a guard that trips outside a named check
// must still produce a readable message rather than an empty pair of quotes.
func TestBudgetErrorWithoutAModuleName(t *testing.T) {
	msg := WasmTypeCheckerBudgetExceededError{Budget: time.Second, Steps: 12}.Error()
	if !strings.Contains(msg, "(module name unavailable)") {
		t.Fatalf("unnamed module rendered without a placeholder:\n%s", msg)
	}
	if !strings.Contains(msg, "12 type-checker steps") {
		t.Fatalf("step count missing from the unnamed-module message:\n%s", msg)
	}
}
