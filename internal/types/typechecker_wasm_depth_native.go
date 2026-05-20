//go:build !js || !wasm

package types

// M-WASM-TYPECHECK-LIMITS (v0.22.x) — native-build stubs for the WASM-only
// wall-clock budget guard. On CLI / native targets, type-checker recursion
// is bound by Go goroutine stacks (which grow dynamically up to 1 GiB), so
// we have no budget. These stubs compile away — callers can invoke them
// unconditionally without build-tag conditionals.
//
// The WASM-targeted counterparts live in typechecker_wasm_depth_wasm.go,
// gated by `//go:build js && wasm`. See design_docs/planned/v0_22_0/
// m-wasm-typecheck-limits.md for the full rationale.

// BeginWasmTypeCheck — no-op on native. WASM bridge calls this before each
// loadModule to start the wall-clock budget.
func BeginWasmTypeCheck(_ string) {
	// Native Go grows goroutine stacks dynamically — no budget needed.
}

// EndWasmTypeCheck — no-op on native.
func EndWasmTypeCheck() {
	// Native: nothing to clean up.
}

// checkWasmBudget — no-op on native. Always returns nil so the caller's
// `if err != nil` branch is always taken on the negative side.
func checkWasmBudget() error {
	return nil
}

func (tc *CoreTypeChecker) wasmDepthEnter(_ *InferenceContext) error {
	// Native: no budget. Caller's err check is always nil.
	return nil
}

func (tc *CoreTypeChecker) wasmDepthExit(_ *InferenceContext) {
	// Native: nothing to track or clean up.
}
