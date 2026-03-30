# Sprint Plan: M-PKG-INTERREF

## Summary

Fix package inter-function references that fail when loaded as a dependency, and enhance `check --package` to catch this class of bug. Root cause: resolver evaluates sequential Let bindings without accumulating them in the evaluator's environment.

**Duration:** 1 day (3 milestones, ~4h total)
**Dependencies:** None — all changes are self-contained
**Risk Level:** Low — surgical fix with clear root cause, existing test infrastructure

## Current Status Analysis

### Completed Recently
- ✅ M-PERF5 M1-M3: Data-intensive workload optimizations
- ✅ Parser DX fixes: @nowrap, effect annotations, record key typos
- ✅ WASM build fix + broken examples for CI

### Velocity
- Recent average: ~200-300 LOC/day (bug fix area)
- Estimated capacity: ~150 LOC for this sprint (small, targeted fixes)

### Remaining from Design Doc
- ⏳ Phase 1: Resolver env accumulation (~8 LOC)
- ⏳ Phase 2: Regression tests (~60 LOC)
- ⏳ Phase 3: `check --package` DX enhancement (~40 LOC)

## Proposed Milestones

### M1: FIX_RESOLVER_ENV — Accumulate Let bindings in resolver evaluator

**Goal:** After evaluating each Let/LetRec in resolver.go, bind results in the evaluator's local environment so subsequent declarations can reference earlier ones.

**Estimated:** 8 LOC implementation + 40 LOC tests = 48 LOC
**Duration:** ~1h

**Tasks:**
1. Add `evaluator.Env().Set(d.Name, val)` after Let evaluation in `resolver.go:130`
2. Add `evaluator.Env().Set(name, val)` after each LetRec binding in `resolver.go:122`
3. Write unit test: module with 3 sequential Lets where f3→f2→f1
4. Write unit test: LetRec followed by Let that references a LetRec binding
5. Run `make test`

**Acceptance Criteria:**
- [ ] `resolver.go` accumulates Let bindings in evaluator env
- [ ] `resolver.go` accumulates LetRec bindings in evaluator env
- [ ] Unit test: sequential Let chain resolves correctly
- [ ] Unit test: LetRec+Let mix resolves correctly
- [ ] `make test` passes
- [ ] `make lint` passes

**Risks:**
- Env accumulation could change semantics for existing modules → mitigated by running full test suite

### M2: INTEGRATION_TESTS — Cross-package inter-function reference tests

**Goal:** Create end-to-end test packages that exercise the fixed resolver path, covering all 4 reported cases.

**Estimated:** 60 LOC (test package + consumer + test harness)
**Duration:** ~1h

**Tasks:**
1. Create test package `tests/runtime_integration/cross_pkg_interref/testpkg/helpers.ail` with sequential helper functions
2. Create consumer `tests/runtime_integration/cross_pkg_interref/main.ail` that imports and calls the exported function
3. Add test case to integration test runner (or standalone `go test`)
4. Create example file `examples/tests/test_package_interref.ail` demonstrating the pattern
5. Run `make test` and `make verify-examples`

**Acceptance Criteria:**
- [ ] Test package with helper functions (non-exported) calling each other
- [ ] Consumer successfully imports and calls exported function that uses helpers
- [ ] Example file demonstrates the pattern
- [ ] `make test` passes
- [ ] `make verify-examples` passes

**Risks:**
- Test package structure may need ailang.toml/ailang.lock setup → check existing cross_pkg tests for pattern

### M3: CHECK_PACKAGE_DX — Enhance `check --package` to catch resolver bugs

**Goal:** Add a link-simulation phase to `check --package` that evaluates module declarations, catching "undefined variable" errors that currently only surface at consumer load time.

**Estimated:** 40 LOC implementation + 20 LOC tests = 60 LOC
**Duration:** ~1.5h

**Tasks:**
1. After successful pipeline.Run per file, add eval pass: iterate Core program decls and evaluate Let/LetRec values
2. Accumulate evaluator env between decls (same pattern as resolver fix)
3. Report errors as warnings: "symbol X in function Y would fail at consumer load time"
4. Add test: package with broken inter-ref → `check --package` now catches it
5. Run `make test`

**Acceptance Criteria:**
- [ ] `check --package` evaluates module declarations after type checking
- [ ] Catches "undefined variable" that would fail at consumer load time
- [ ] Error message is actionable (names the symbol and function)
- [ ] Does NOT break existing packages that pass check
- [ ] `make test` passes
- [ ] `make lint` passes

**Risks:**
- Evaluation during check may be slow for large packages → mitigate with timeout (already exists in check_package.go)
- Some packages may use effects in top-level code that can't be evaluated during check → only evaluate pure Let/LetRec values, skip effectful code

## Success Metrics
- All 4 reported cases from sunholo/gemini_live would work (verified by pattern match in tests)
- `make test` passes
- `make verify-examples` passes
- `make lint` passes
- CHANGELOG.md updated
- Example file added

## Dependencies
- None — all changes are in internal/link/resolver.go, cmd/ailang/check_package.go, and test files

## Open Questions
- None — root cause confirmed, fix verified (Env() accessor already exists)

## Notes
- This is the third instance of the "resolver context leak" bug class
- M-DX-XPKG-RESOLVE (v0.9.5) fixed VarGlobal resolution with FallbackResolver
- This fix addresses core.Var (local variable) resolution within on-demand module eval
- The fixes are complementary, not conflicting
