package types

import (
	"fmt"
	"math"
	"time"
)

// M-WASM-TYPECHECK-LIMITS — the type-check budget state machine.
//
// This file carries NO build tag on purpose. The guard it implements only
// ever ARMS on WASM (BeginWasmTypeCheck is a no-op on native — see
// typechecker_wasm_depth_native.go), but the mechanism itself compiles and
// runs identically on both targets so it can be exercised by ordinary
// `go test ./internal/types`.
//
// Before ailang#662 the whole mechanism lived behind `//go:build js && wasm`,
// which put it outside the native build by construction: `go list` reports
// zero native Go files containing it, so no native test could reach it and
// none existed. A guard nothing can red when you remove it is not a guard.
//
// ailang#662 — the budget was also a hardcoded wall-clock constant, which
// makes shipped correctness hardware-dependent: the same bytes load on a fast
// desktop and fail on a slower laptop or a slower browser engine, and CI on
// fast runners cannot see it. Two things change here:
//
//  1. the limit is CONFIGURABLE by the embedder (SetWasmTypeCheckBudget), so a
//     host that knows its own tolerance is not held to ours; and
//  2. consumption is REPORTED — both the wall-clock elapsed and a deterministic
//     STEP count — so an embedder can watch headroom instead of discovering the
//     limit in production.
//
// The step count is deliberately NOT a gate yet. It is a reproducible measure
// of type-checker work (identical source ⇒ identical count, on every machine),
// and it exists so the step ceiling that would REPLACE the wall-clock limit can
// be chosen from measurements on real module corpora rather than guessed. See
// the queue row m-wasm-deterministic-typecheck-budget.

// DefaultWasmTypeCheckBudget is the wall-clock limit applied when an embedder
// has not chosen one. Kept at the pre-ailang#662 value so behaviour is
// unchanged for hosts that never call SetWasmTypeCheckBudget.
//
// Chosen empirically in v0.22.x by probing citizen.ail: 100ms false-fires on a
// legitimate 7.5 KB module, 500ms and 2000ms both pass it. 2000ms shipped for
// the headroom. ailang#662 is the report that the headroom is illusory on
// slower hardware.
const DefaultWasmTypeCheckBudget = 2 * time.Second

// typeCheckBudget is the guard's state. Package-level rather than plumbed
// through every recursive type-checker call because WASM Go is single-threaded
// (the same reason the pre-#662 implementation used package-level vars).
//
// The clock is injectable so tests can drive the deadline deterministically
// instead of sleeping — the defect ailang#662 reports IS a timing dependence,
// so a test that measures it by sleeping would inherit the flakiness it is
// meant to pin.
type typeCheckBudget struct {
	now   func() time.Time
	limit time.Duration // 0 == no wall-clock limit

	active   bool
	tripped  bool
	deadline time.Time
	started  time.Time
	module   string
	steps    uint64

	// Retained after end() so the bridge can report consumption for the
	// check that just finished, on success as well as on failure.
	lastSteps   uint64
	lastElapsed time.Duration
}

var wasmBudget = &typeCheckBudget{
	now:   time.Now,
	limit: DefaultWasmTypeCheckBudget,
}

// begin arms the guard for one type-check and resets per-check state.
func (b *typeCheckBudget) begin(module string) {
	b.active = true
	b.tripped = false
	b.module = module
	b.steps = 0
	b.started = b.now()
	if b.limit > 0 {
		b.deadline = b.started.Add(b.limit)
	} else {
		b.deadline = time.Time{}
	}
}

// end disarms the guard, capturing consumption for later reporting.
func (b *typeCheckBudget) end() {
	if b.active {
		b.lastSteps = b.steps
		b.lastElapsed = b.now().Sub(b.started)
	}
	b.active = false
	b.tripped = false
	b.module = ""
	b.deadline = time.Time{}
}

// check is called from the instrumented type-checker entry points (inferCore
// and Unify). It counts one step and reports whether the wall-clock limit has
// been passed.
//
// Sticky: once tripped, every later call returns the same error until end().
// Without this the type-checker's own error-recovery loop keeps trying
// alternate inference paths after Unify returns an error, and the enclosing
// loadModule never returns.
func (b *typeCheckBudget) check() error {
	if !b.active {
		return nil
	}
	b.steps++
	if b.tripped {
		return b.err()
	}
	// deadline is zero exactly when limit <= 0, i.e. the embedder disabled
	// the wall-clock guard. Steps are still counted.
	if b.deadline.IsZero() {
		return nil
	}
	if b.now().After(b.deadline) {
		b.tripped = true
		return b.err()
	}
	return nil
}

func (b *typeCheckBudget) err() WasmTypeCheckerBudgetExceededError {
	return WasmTypeCheckerBudgetExceededError{
		Budget: b.limit,
		Module: b.module,
		Steps:  b.steps,
	}
}

// SetWasmTypeCheckBudget sets the wall-clock limit for subsequent
// type-checks. A duration of 0 disables the wall-clock guard entirely (steps
// are still counted and reported). A negative duration is rejected and leaves
// the current limit unchanged.
//
// ailang#662: embedders know their own tolerance — a host that has already
// downloaded a 40 MB WASM binary may well prefer a slow load to a refused one,
// and a host targeting slow hardware needs a bigger number than a CI runner
// does. Takes effect at the next BeginWasmTypeCheck; an in-flight check keeps
// the deadline it was armed with.
func SetWasmTypeCheckBudget(d time.Duration) error {
	if d < 0 {
		return fmt.Errorf("type-check budget must be >= 0 (0 disables the wall-clock limit), got %s", d)
	}
	wasmBudget.limit = d
	return nil
}

// WasmTypeCheckBudget reports the configured wall-clock limit; 0 means the
// wall-clock guard is disabled.
func WasmTypeCheckBudget() time.Duration {
	return wasmBudget.limit
}

// LastWasmTypeCheckStats reports consumption for the most recently completed
// type-check: the deterministic step count and the wall-clock elapsed.
//
// ailang#662 ask 3 — "expose the consumed budget on success, so embedders can
// monitor headroom instead of discovering the limit in production". steps is
// the reproducible half: identical source yields an identical count on every
// machine, unlike elapsed.
func LastWasmTypeCheckStats() (steps uint64, elapsed time.Duration) {
	return wasmBudget.lastSteps, wasmBudget.lastElapsed
}

// ParseBudgetMillis converts an embedder-supplied millisecond value into a
// duration, rejecting the shapes a JS host can hand us that a time.Duration
// cannot represent meaningfully.
//
// Lives here rather than in the WASM bridge so it is reachable from an
// ordinary native test: everything inside cmd/wasm is behind `js && wasm` and
// therefore untestable by `go test`.
func ParseBudgetMillis(ms float64) (time.Duration, error) {
	if math.IsNaN(ms) {
		return 0, fmt.Errorf("type-check budget must be a number, got NaN")
	}
	if ms < 0 {
		return 0, fmt.Errorf("type-check budget must be >= 0 milliseconds (0 disables the wall-clock limit), got %v", ms)
	}
	const maxBudgetMillis = 24 * 60 * 60 * 1000 // one day; well inside int64 nanoseconds
	if ms > maxBudgetMillis {
		return 0, fmt.Errorf("type-check budget must be <= %d milliseconds, got %v", int64(maxBudgetMillis), ms)
	}
	return time.Duration(ms * float64(time.Millisecond)), nil
}

// WasmTypeCheckerBudgetExceededError is returned when type-checking exceeds
// the wall-clock budget. The message lists common triggers and workarounds so
// the user can act without reading AILANG internals, and — since ailang#662 —
// names the budget that was actually in force plus how to change it, because
// the previous message described a fixed limit that is in fact a host setting.
type WasmTypeCheckerBudgetExceededError struct {
	Budget time.Duration
	Module string
	// Steps is the deterministic work count reached when the budget blew.
	// Reproducible across machines for identical source, so it is the number
	// worth quoting in a bug report.
	Steps uint64
}

func (e WasmTypeCheckerBudgetExceededError) Error() string {
	mod := e.Module
	if mod == "" {
		mod = "(module name unavailable)"
	}
	return fmt.Sprintf(
		"WASM type-checker budget exceeded (%s, %d type-checker steps) while checking module %q.\n\n"+
			"This limit is WALL-CLOCK, so it depends on how fast this machine and browser\n"+
			"are: the same source may load elsewhere and fail here. The step count above is\n"+
			"hardware-independent — quote it if you report this.\n\n"+
			"If the module is simply large (rather than pathological), raise the limit from\n"+
			"the host and reload:\n"+
			"    ailangSetTypeCheckBudget(8000);   // milliseconds; 0 disables the limit\n"+
			"ailangLoadModule() reports typeCheckMs / typeCheckSteps on success, so you can\n"+
			"watch headroom instead of discovering this in production.\n\n"+
			"Common triggers for genuinely pathological checking:\n"+
			"  - Triple-nested match patterns (match inside match inside match)\n"+
			"  - Multiple back-to-back matches on the same tagged-union value\n"+
			"      (e.g. Ok(s) => s.x, then Ok(s) => s.y, then Ok(s) => s.z)\n"+
			"  - Long chains of intra-package imports with destructured constructors\n"+
			"  - Deeply-extended record rows used inside match arms\n\n"+
			"Workarounds (in order of simplicity):\n"+
			"  1. Flatten nested matches into sequential let-bindings:\n"+
			"       let x = match foo { Ok(s) => s.x, Err(_) => 0.0 };\n"+
			"  2. Extract a helper function that does one match and returns a record:\n"+
			"       pure func unpack(r: Result[T, E]) -> { ok: bool, x: T } = match r { ... }\n"+
			"  3. Split the function into smaller top-level functions.\n\n"+
			"See: https://ailang.sunholo.com/docs/reference/limitations#wasm-type-checker-depth-limit-by-design-wasm-only\n"+
			"Headless smoke harness: demos/scripts/wasm-loadmodule-harness.js (exit code 4 = this error)",
		e.Budget, e.Steps, mod,
	)
}

// checkWasmBudget is the instrumented hook called from Unify. It is inert
// until BeginWasmTypeCheck arms the guard, which only happens on WASM — so on
// native builds this is one predictable branch on a package-level bool, and
// tests can arm it explicitly to exercise the real code path rather than a
// build-tag twin of it.
func checkWasmBudget() error {
	return wasmBudget.check()
}

// wasmDepthEnter is called at the top of inferCore. Same hook, same inertness.
func (tc *CoreTypeChecker) wasmDepthEnter(_ *InferenceContext) error {
	return wasmBudget.check()
}

// wasmDepthExit is intentionally empty: the budget is checked on entry against
// package-level state, so there is no per-call bookkeeping to unwind. Retained
// so inferCore can `defer tc.wasmDepthExit(ctx)` unconditionally.
func (tc *CoreTypeChecker) wasmDepthExit(_ *InferenceContext) {
	// no-op — see comment above
}
