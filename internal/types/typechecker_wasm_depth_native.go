//go:build !js || !wasm

package types

// M-WASM-TYPECHECK-LIMITS (v0.22.x) — native-build stubs for ARMING the
// type-check budget guard.
//
// On CLI / native targets, type-checker recursion is bound by Go goroutine
// stacks (which grow dynamically up to 1 GiB), so a long type-check is slow
// rather than fatal and no budget is wanted: arming one here would start
// refusing large modules that work today.
//
// The guard's mechanism is NOT build-tagged — it lives in typecheck_budget.go
// and compiles into both targets, so `go test ./internal/types` can exercise
// it. It is inert until armed, and these stubs never arm it, so native
// behaviour is unchanged: checkWasmBudget() takes one predictable branch on a
// package-level bool and returns nil. Tests arm the state machine directly.
//
// The WASM-targeted counterparts live in typechecker_wasm_depth_wasm.go, gated
// by `//go:build js && wasm`. See design_docs/planned/v0_22_0/
// m-wasm-typecheck-limits.md for the full rationale, and ailang#662 for why
// the budget became configurable and observable.

// BeginWasmTypeCheck — no-op on native. The WASM bridge calls this before each
// loadModule to arm the budget.
func BeginWasmTypeCheck(_ string) {
	// Native Go grows goroutine stacks dynamically — no budget needed.
}

// EndWasmTypeCheck — no-op on native: nothing was armed, so nothing to clear.
func EndWasmTypeCheck() {
}
