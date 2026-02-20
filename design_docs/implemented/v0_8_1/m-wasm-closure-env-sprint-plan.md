# Sprint Plan: M-WASM-CLOSURE-ENV

## Summary

| Field | Value |
|-------|-------|
| **Sprint ID** | M-WASM-CLOSURE-ENV |
| **Goal** | Fix closure environment resolution when AILANG closures are invoked from JS via ApplyClosure |
| **Design Doc** | [m-wasm-closure-env.md](m-wasm-closure-env.md) |
| **Duration** | 0.5 day (~3-4 hours) |
| **Risk Level** | Medium (diagnosis-first approach) |
| **Estimated LOC** | ~160 |

## Context

After M-WASM-STREAM-BRIDGE Phase 1-2, closures are correctly wrapped as `js.FuncOf` and JS handlers can return ADTs. But closures that reference module-scoped bindings (sibling functions, imports) fail with "undefined variable" when invoked from JS callbacks. This blocks the `onEvent(conn, handler)` real-time streaming pattern.

## Milestones

### M1: Reproduce and Diagnose (~40 LOC test + debug)

**Goal:** Write a minimal reproduction test and determine the exact root cause.

**Tasks:**
- [ ] Write test: load module with `let helper = ...` + `let callback = \x. helper(x)`, invoke callback via `ApplyClosure`
- [ ] Add debug tracing to `CallFunction` to log `fn.Env` bindings
- [ ] Determine which of the 3 hypotheses is correct:
  - (a) Env capture timing: closure created before sibling bindings evaluated
  - (b) Env chain break: Clone() loses needed bindings
  - (c) Resolver absence: REPL evaluator lacks GlobalResolver for module vars
- [ ] Remove debug tracing after diagnosis

**Acceptance Criteria:**
- Test reproduces the "undefined variable" error
- Root cause identified and documented in sprint JSON notes
- `make test` passes (new test expected to fail until M2)
- `make lint` passes

### M2: Fix Closure Environment Resolution (~60 LOC)

**Goal:** Apply the targeted fix based on M1 diagnosis.

**Tasks (one of these paths based on M1):**

**Path A - Env capture timing:**
- [ ] Ensure all module bindings are in env before closures that reference them
- [ ] May need to adjust evaluation order in `module_registry.go`

**Path B - Env chain break:**
- [ ] Fix `Clone()` or `CallFunction` to preserve full parent chain
- [ ] Ensure `fn.Env` parent points to module root env with all bindings

**Path C - Resolver propagation:**
- [ ] Propagate `GlobalResolver` from module evaluator to REPL evaluator
- [ ] Or set `RegistryResolver` on REPL evaluator in `NewWasmREPL()`

**Acceptance Criteria:**
- M1 test now passes (closure resolves sibling binding)
- Test for closure referencing imported function also passes
- No regression in existing `go test ./internal/repl/...` tests
- `make test` passes
- `make lint` passes

### M3: WASM Verification and CHANGELOG (~20 LOC)

**Goal:** Verify WASM build, update CHANGELOG, finalize.

**Tasks:**
- [ ] WASM build: `GOOS=js GOARCH=wasm go build -o web/ailang.wasm ./cmd/wasm/`
- [ ] Update CHANGELOG.md with M-WASM-CLOSURE-ENV entry
- [ ] Run full `make test` and `make lint`
- [ ] Reply to demos inbox with fix summary

**Acceptance Criteria:**
- WASM binary builds successfully
- CHANGELOG.md updated
- All tests pass
- All linting passes

## Dependencies

- M-WASM-STREAM-BRIDGE Phase 1-2 (completed) — provides the `ApplyClosure` mechanism
- Strict sequential: M1 → M2 → M3

## Success Metrics

- Closures referencing module-scoped bindings work via `ApplyClosure`
- No regressions in existing tests
- WASM builds and lints clean
