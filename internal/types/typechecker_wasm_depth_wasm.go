//go:build js && wasm

package types

import (
	"fmt"
	"time"
)

// M-WASM-TYPECHECK-LIMITS (v0.22.x) — wall-clock budget guard for the
// AILANG WASM type-checker.
//
// Why this exists:
//
// The browser-freeze symptom is silent unresponsiveness for 80–120 seconds
// before the JS engine eventually throws "Maximum call stack size exceeded"
// (or the user force-quits the tab). Whether the cause is true stack
// overflow OR pathological type-checker slowness (e.g. quadratic isTaggedUnion
// analysis, deeply-nested unify recursion), the user experience is the same:
// browser is frozen. A wall-clock budget catches BOTH causes with the same
// structured error.
//
// First hit 2026-05-20: cognitive_commons/services/citizen.ail loaded in 18ms
// on CLI but hung WASM for 80+ seconds. Full diagnostic trail:
// demos/debug-notes/wasm-citizen-stack-overflow.md.
//
// On CLI the corresponding stubs in typechecker_wasm_depth_native.go are
// no-ops. Native Go has dynamically-growable per-goroutine stacks AND no
// host call-stack limit, so we don't need a budget there.
//
// Implementation strategy:
//
// WASM Go is single-threaded, so we use a package-level deadline variable
// instead of plumbing context through every recursive function. The WASM
// bridge (cmd/wasm/main.go::loadModule) sets the deadline before invoking
// the type-checker via beginWasmTypeCheck(); helpers check it on every
// recursive call (inferCore, Unify) and return a structured error when
// exceeded. endWasmTypeCheck() clears the deadline.

// wasmTypeCheckBudget — chosen empirically by probing citizen.ail. The
// guard fires when checkWasmBudget() is called from instrumented sites
// (currently Unify + inferCore entry). Empirical results:
//
//	Budget   citizen.ail (pathological)        commons_browser (legit, 7.5KB)
//	100ms    ✓ fires in 110ms                  ✗ false-fire at 102ms
//	500ms    ✗ Unify-free phase hides it       ✓ passes (102ms)
//	2000ms   ✗ same as 500ms                   ✓ passes
//
// **Honest scope of this guard**: catches type-checker pathologies whose
// hot loop goes through Unify or inferCore. citizen.ail's specific shape
// happens to do most of its 80+s in pattern resolution / constraint
// substitution, which our current call sites miss. The harness in the
// demos repo has its own 15s per-module timeout that catches citizen as
// a fallback.
//
// 2000ms ships as the production budget — gives 10-20x headroom over the
// slowest legitimate module (commons_browser at 102-225ms) and still
// catches pathologies that DO go through Unify. If a future module hits
// the limit, the user sees the same clear error pointing at the standard
// workarounds (flatten matches, extract helpers).
//
// Future improvement (deferred): plumb checkWasmBudget into more entry
// points (inferRecordAccess, checkPattern, ApplySubstitution) so the
// guard catches citizen-shape recursions too. Currently this would
// require signature changes (ApplySubstitution returns Type, not error)
// or panic-based unwinding. Both are non-trivial and deferred.
const wasmTypeCheckBudget = 2 * time.Second

// wasmDeadline is the current type-check's wall-clock deadline. Zero means
// no active check. Package-level (rather than ctx-plumbed) so it's
// accessible from Unify without requiring an InferenceContext parameter.
// Safe because WASM Go is single-threaded.
var wasmDeadline time.Time

// wasmActiveModule is the module name being checked, for inclusion in the
// budget-exceeded error message. Set alongside wasmDeadline.
var wasmActiveModule string

// wasmBudgetTripped is "sticky" — once the budget is exceeded once, every
// subsequent checkWasmBudget() call returns the same error until
// EndWasmTypeCheck() resets the state. Without this, the type-checker's
// own error-recovery loop kept trying alternate inference paths after we
// returned an error from Unify, and the loadModule call never returned.
// Sticky semantics: budget exceeded → abort everything in the current
// loadModule, surface the error to the bridge, let the bridge re-arm.
var wasmBudgetTripped bool

// BeginWasmTypeCheck starts the wall-clock budget for the next type-check.
// Called by the WASM bridge before invoking loadModule. The CLI build has
// a no-op stub of this function.
func BeginWasmTypeCheck(moduleName string) {
	wasmDeadline = time.Now().Add(wasmTypeCheckBudget)
	wasmActiveModule = moduleName
	wasmBudgetTripped = false
}

// EndWasmTypeCheck clears the deadline. Called by the WASM bridge after
// loadModule returns (success or failure). The CLI build has a no-op stub.
func EndWasmTypeCheck() {
	wasmDeadline = time.Time{}
	wasmActiveModule = ""
	wasmBudgetTripped = false
}

// checkWasmBudget returns a structured error if the budget has been
// exceeded. Sticky: once tripped, every subsequent call returns the same
// error until EndWasmTypeCheck() resets. Called at the top of every
// recursive type-checker entry (inferCore, Unify). Fast-path zero-check
// means uninstrumented calls (e.g. during REPL evaluation before
// BeginWasmTypeCheck) skip the time.Now() syscall.
func checkWasmBudget() error {
	if wasmDeadline.IsZero() {
		return nil
	}
	if wasmBudgetTripped {
		return WasmTypeCheckerBudgetExceededError{
			Budget: wasmTypeCheckBudget,
			Module: wasmActiveModule,
		}
	}
	if time.Now().After(wasmDeadline) {
		wasmBudgetTripped = true
		return WasmTypeCheckerBudgetExceededError{
			Budget: wasmTypeCheckBudget,
			Module: wasmActiveModule,
		}
	}
	return nil
}

// wasmDepthEnter is called at the top of inferCore. Delegates to the
// shared wall-clock check.
func (tc *CoreTypeChecker) wasmDepthEnter(ctx *InferenceContext) error {
	return checkWasmBudget()
}

// wasmDepthExit is intentionally empty for the wall-clock budget design.
// The depth-counter design had bookkeeping here (decrement on exit) but
// the wall-clock variant doesn't need to track per-call state — checks
// happen on entry against a package-level deadline. The function is
// retained so the native + WASM stub signatures stay aligned and the
// inferCore caller can `defer tc.wasmDepthExit(ctx)` unconditionally.
func (tc *CoreTypeChecker) wasmDepthExit(_ *InferenceContext) {
	// no-op — see comment above
}

// WasmTypeCheckerBudgetExceededError is returned when type-checking exceeds
// the wall-clock budget on WASM. The error message lists common triggers
// and workarounds so the user can flatten their module without needing to
// read AILANG internals.
//
// Defined here so it compiles into the WASM binary only.
type WasmTypeCheckerBudgetExceededError struct {
	Budget time.Duration
	Module string
}

func (e WasmTypeCheckerBudgetExceededError) Error() string {
	mod := e.Module
	if mod == "" {
		mod = "(module name unavailable)"
	}
	return fmt.Sprintf(
		"WASM type-checker budget exceeded (%s) while checking module %q.\n\n"+
			"This module's type structure recurses too deeply OR triggers pathologically\n"+
			"slow analysis for the WASM runtime. The same source likely works on the\n"+
			"AILANG CLI — this limit is specific to browser execution.\n\n"+
			"Common triggers:\n"+
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
		e.Budget, mod,
	)
}
