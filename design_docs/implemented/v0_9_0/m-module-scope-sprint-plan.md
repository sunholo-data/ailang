# Sprint Plan: M-MODULE-SCOPE — Per-module Environment Isolation

## Summary
Fix the module scope collision bug where non-exported functions with the same name in different modules overwrite each other in the shared evaluator environment. Implement Option A from the design doc: per-module environment isolation in ModuleRuntime.

**Duration:** Half day (~3-4 hours)
**Dependencies:** None (regression test already exists and FAILS)
**Risk Level:** Low — well-understood root cause, clear fix, existing test infrastructure

## Current Status Analysis

### Completed (investigation phase)
- Root cause identified: `runtime.go:extractBindings()` adds ALL bindings to shared `rt.evaluator.Env()`
- Regression test written: `TestModuleScopeCollision_Runtime` — currently FAILS (proves bug)
- Test fixtures created: `scope_a.ail`, `scope_b.ail`, `scope_main.ail`
- Design doc written: `m-docparse-table-regression.md`

### Key Code Context
- `Environment` already supports parent-chain scoping (`NewChildEnvironment`, `Clone`)
- `CoreEvaluator` has `Env()` accessor but no `SetEnv()` — needs adding
- `evaluateModule()` at runtime.go:227 is the single entry point to fix
- `extractBindings()` at runtime.go:290 calls `rt.evaluator.Env().Set(name, val)` — the bug site

## Proposed Milestones

### Milestone 1: Add SetEnv to CoreEvaluator
**Goal:** Enable the runtime to save/restore evaluator environments
**Estimated:** 10 LOC implementation + 20 LOC tests = 30 LOC
**Duration:** 15 minutes

**Tasks:**
- Add `SetEnv(env *Environment)` method to `CoreEvaluator`
- Add unit test for SetEnv

**Acceptance Criteria:**
- [ ] `SetEnv` method exists and works
- [ ] Unit test passes
- [ ] No other evaluator tests break

### Milestone 2: Per-module Environment Isolation in evaluateModule
**Goal:** Each module evaluates in a child environment; closures capture module-scoped env
**Estimated:** 30 LOC implementation + 10 LOC comments = 40 LOC
**Duration:** 30 minutes

**Tasks:**
- In `evaluateModule()`: save current env, create child env, evaluate, restore parent env
- Closures will naturally capture the child env (since `evalCoreLambda` captures `e.env`)
- Keep builtins accessible via parent chain (child → parent with builtins)

**Implementation approach:**
```go
func (rt *ModuleRuntime) evaluateModule(inst *ModuleInstance) error {
    // Save parent environment (contains builtins + prior module exports)
    parentEnv := rt.evaluator.Env()
    // Create isolated child scope for this module's internal bindings
    moduleEnv := parentEnv.NewChildEnvironment()
    rt.evaluator.SetEnv(moduleEnv)
    // After evaluation, restore parent so next module starts fresh
    defer rt.evaluator.SetEnv(parentEnv)

    // ... rest of evaluateModule unchanged ...
}
```

**Acceptance Criteria:**
- [ ] `TestModuleScopeCollision_Runtime` PASSES (was FAILING)
- [ ] All existing `TestIntegration_*` tests still pass
- [ ] Cross-module imports still resolve (VarGlobal path unaffected)

### Milestone 3: Verify Closure Capture Correctness
**Goal:** Ensure closures capture their module's env, not the parent
**Estimated:** 60 LOC tests = 60 LOC
**Duration:** 30 minutes

**Tasks:**
- Add test: load 3 modules with same internal name, verify all resolve correctly
- Add test: closure called AFTER other modules load still uses correct internal
- Verify the `evalCoreLambda` capture path works with child environments

**Acceptance Criteria:**
- [ ] Three-module collision test passes
- [ ] Closure capture test passes
- [ ] `go test ./internal/runtime/ ./internal/eval/` all pass

### Milestone 4: Full Test Suite + Cleanup
**Goal:** Run full test suite, update docs, clean up temp files
**Estimated:** 20 LOC doc updates = 20 LOC
**Duration:** 30 minutes

**Tasks:**
- Run `make test` (full suite with `-tags '!stream'`)
- Run `make lint`
- Update design doc status to "implemented"
- Update CHANGELOG.md
- Clean up `/tmp/scope_bug/` temp files
- Remove `internal/repl/module_scope_collision_test.go` (tested wrong path)

**Acceptance Criteria:**
- [ ] `make test` passes (with `-tags '!stream'`)
- [ ] `make lint` passes
- [ ] Design doc moved to `implemented/v0_9_0/`
- [ ] CHANGELOG updated

## Success Metrics
- Regression test `TestModuleScopeCollision_Runtime` PASSES
- All existing runtime integration tests PASS
- All existing eval tests PASS
- `make test` clean (with `-tags '!stream'`)
- Design doc marked implemented

## Risks
- **Child env parent chain lookup**: Internal functions in a module need to reference builtins (e.g., `+` operator). Since `NewChildEnvironment` chains to parent, builtins remain accessible. **Low risk.**
- **Cross-module export resolution**: Exported functions resolve via `VarGlobal` → `moduleGlobalResolver`, which is independent of the evaluator env. **No risk.**
- **Closure re-entry**: When a closure from module_a is called after module_b loads, the closure's captured env must still point to module_a's scope. Since `evalCoreLambda` captures `e.env` at closure creation time, and we create a child env per module, this is correct. **Low risk.**

## Notes
- This is a targeted fix — we're NOT changing elaboration (Option B) or env key namespacing (Option C)
- ModuleRegistry (WASM) is already correct — it creates fresh evaluator per module
- The docparse demo workaround (rename internal functions) is no longer needed after this fix
