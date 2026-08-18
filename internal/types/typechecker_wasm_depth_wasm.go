//go:build js && wasm

package types

// M-WASM-TYPECHECK-LIMITS (v0.22.x) — WASM-only ARMING of the type-check
// budget guard. The guard itself (state machine, configuration, step counting,
// error type, and the checkWasmBudget / wasmDepthEnter hooks) lives in the
// build-tag-free typecheck_budget.go so it is reachable from native tests;
// only the decision to arm it is target-specific.
//
// Why the guard exists:
//
// The browser-freeze symptom is silent unresponsiveness for 80–120 seconds
// before the JS engine eventually throws "Maximum call stack size exceeded"
// (or the user force-quits the tab). Whether the cause is true stack overflow
// OR pathological type-checker slowness (e.g. quadratic isTaggedUnion
// analysis, deeply-nested unify recursion), the user experience is the same:
// browser is frozen. A budget catches BOTH causes with one structured error.
//
// First hit 2026-05-20: cognitive_commons/services/citizen.ail loaded in 18ms
// on CLI but hung WASM for 80+ seconds. Full diagnostic trail:
// demos/debug-notes/wasm-citizen-stack-overflow.md.
//
// Why native does NOT arm it (see typechecker_wasm_depth_native.go): native Go
// has dynamically-growable per-goroutine stacks and no host call-stack limit,
// so a CLI type-check that takes a long time is slow, not fatal — and applying
// a wall-clock limit there would refuse large modules that work fine today.
//
// Honest scope of this guard: it catches type-checker pathologies whose hot
// loop goes through Unify or inferCore, the two instrumented entry points.
// citizen.ail's specific shape does most of its 80+s in pattern resolution /
// constraint substitution, which those call sites miss; the demos harness has
// its own 15s per-module timeout as a fallback. Plumbing the check into
// inferRecordAccess / checkPattern / ApplySubstitution would need signature
// changes or panic-based unwinding, and remains deferred.

// BeginWasmTypeCheck arms the budget for the next type-check. Called by the
// WASM bridge (cmd/wasm/main.go::loadModule) before invoking the type-checker.
// The native build has a no-op stub.
func BeginWasmTypeCheck(moduleName string) {
	wasmBudget.begin(moduleName)
}

// EndWasmTypeCheck disarms the budget and records consumption, which the
// bridge then reports via LastWasmTypeCheckStats. Called after loadModule
// returns, on success or failure. The native build has a no-op stub.
func EndWasmTypeCheck() {
	wasmBudget.end()
}
