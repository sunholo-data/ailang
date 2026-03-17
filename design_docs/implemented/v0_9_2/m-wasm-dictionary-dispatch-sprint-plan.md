# Sprint Plan: M-WASM-DICTIONARY-DISPATCH

## Summary
Fix two WASM-only bugs (import shadowing + eq_Int dispatch in helper lambdas) by adding monomorphization and var type resolution phases to ModuleRegistry, achieving compilation parity with the native CLI pipeline.

**Sprint ID**: M-WASM-DICTIONARY-DISPATCH
**Duration**: 1 day (6-8 hours)
**Dependencies**: None (all infrastructure exists in native pipeline)
**Risk Level**: Low (reusing proven code from native pipeline)
**Design Doc**: [m-wasm-dictionary-dispatch.md](m-wasm-dictionary-dispatch.md)

## Current Status Analysis

### Completed Recently
- M-PERF7: foldChars/charAt builtins + batch mode (~200 LOC)
- M-ITERATIVE-LIST: Iterative Go builtins for map/filter/foldl (~150 LOC)
- M-PERF6: Content-addressed compilation cache (~400 LOC)
- M-GIT-GUARDRAILS: Per-agent git_mode (~100 LOC)
- M-HARNESS-COMMIT-CONTRACT: siteSlug/briefId dispatch (~80 LOC)

### Velocity
- Recent average: ~150-200 LOC/day
- This sprint is small: ~80-180 LOC total (implementation + tests)
- Well within single-day capacity

### Remaining from Design Doc
- Phase 1: Add monomorphization + var resolution to ModuleRegistry (~30 LOC)
- Phase 2: Write regression tests for both bugs (~100-150 LOC)
- Phase 3: Remove workarounds and verify (~minimal LOC, mostly deletions)

## Proposed Milestones

### M1: Add Monomorphization + Var Resolution to ModuleRegistry
**Goal:** Insert the two missing compilation phases between Step 4 (dictionary elaboration) and Step 5 (op lowering) in `module_registry.go`, matching the native pipeline.
**Estimated:** 30 LOC implementation
**Duration:** 1-2 hours

**Files:**
| File | LOC | Description |
|------|-----|-------------|
| `internal/repl/module_registry.go` | ~30 | Add Specialize + VarResolve between Steps 4-5 |

**Tasks:**
- [ ] After Step 4 (line 511), add `ValidateCoreTypeInfo(coreProg, typeChecker.CoreTI)`
- [ ] Add `pipeline.NewSpecializer(&typeChecker.CoreTI)` + `specializer.Specialize(elaboratedProg)`
- [ ] Add `pipeline.NewVarResolver(typeChecker.CoreTI)` + `resolver.Resolve(specializedProg)`
- [ ] Update Step 5 to use the specialized program instead of `elaboratedProg`
- [ ] Run `make test` to verify no regressions

**Acceptance Criteria:**
- [ ] ModuleRegistry pipeline now includes monomorphization phase
- [ ] ModuleRegistry pipeline now includes var type resolution phase
- [ ] Existing `module_registry_test.go` tests all pass
- [ ] `make test` passes
- [ ] `make lint` passes

**Risks:**
- CoreTypeInfo may not be fully populated in ModuleRegistry context — Mitigation: native pipeline uses identical typeChecker.CoreTI, so should work

---

### M2: Regression Tests for Both Bugs
**Goal:** Write test cases that reproduce both WASM bugs and verify they're fixed.
**Estimated:** 100-150 LOC tests
**Duration:** 2-3 hours

**Files:**
| File | LOC | Description |
|------|-----|-------------|
| `internal/repl/module_registry_dispatch_test.go` | ~100-150 | Bug reproduction tests |

**Tasks:**
- [ ] Test: Load module importing `std/string (length)` + `std/list (length as listLength)`, verify `listLength` dispatches correctly on a list
- [ ] Test: Load module with non-exported helper function using `any()` with lambda containing `s == slug` (string ==), verify `eq_String` dispatch
- [ ] Test: Verify inlined version also works (control case)
- [ ] Test: Verify helper with non-string `==` (e.g., int) also works
- [ ] Run `make test` and `make lint`

**Acceptance Criteria:**
- [ ] Bug 1 reproduction test passes (import shadowing resolved)
- [ ] Bug 2 reproduction test passes (eq_String dispatched correctly)
- [ ] Control tests pass (inlined versions, int ==)
- [ ] All existing tests pass
- [ ] WASM builds: `GOOS=js GOARCH=wasm go build ./cmd/wasm/`

**Risks:**
- Stdlib modules may not load in test environment — Mitigation: check existing test patterns in `module_registry_test.go` and `wasm_effects_test.go`

---

### M3: Verify and Clean Up
**Goal:** Verify fix in WASM build, remove workarounds from validator.ail, update CHANGELOG.
**Estimated:** ~20 LOC changes (mostly deletions)
**Duration:** 1 hour

**Tasks:**
- [ ] Build WASM: `GOOS=js GOARCH=wasm go build ./cmd/wasm/`
- [ ] Send message to demos inbox confirming fix
- [ ] Update CHANGELOG.md with bug fix entry
- [ ] Move design doc to `design_docs/implemented/v0_9_2/`

**Acceptance Criteria:**
- [ ] WASM binary builds successfully
- [ ] CHANGELOG updated
- [ ] Design doc moved to implemented

**Risks:**
- None significant

## Day-by-Day Schedule

### Day 1 (6-8 hours)

| Time | Task | Milestone |
|------|------|-----------|
| 0-1h | Add monomorphization + var resolution to `module_registry.go` | M1 |
| 1-2h | Run existing tests, fix any issues | M1 |
| 2-4h | Write bug reproduction tests | M2 |
| 4-5h | Run full test suite, fix regressions | M2 |
| 5-6h | Build WASM, update CHANGELOG, send confirmation | M3 |

**End of Day 1**: Both bugs fixed, tested, WASM verified.

## Success Metrics
- All existing `module_registry_test.go` tests pass
- 4+ new tests for the two bug scenarios
- `make test` passes
- `make lint` passes
- WASM builds successfully
- CHANGELOG updated

## Dependencies
- None — all code (Specializer, VarResolver, ValidateCoreTypeInfo) exists in `internal/pipeline/`

## Open Questions
- None — approach is clear (copy pattern from native pipeline)

## Notes
- The fix is intentionally minimal: ~30 LOC insertion reusing existing infrastructure
- Both bugs have identical root cause (missing phases), so one fix addresses both
- Workaround removal in validator.ail is deferred to the demos repo maintainer
- Consider future work: unify ModuleRegistry and Pipeline to prevent future phase-gap bugs

---

**Created**: 2026-03-17
**Approved**: Pending user approval
