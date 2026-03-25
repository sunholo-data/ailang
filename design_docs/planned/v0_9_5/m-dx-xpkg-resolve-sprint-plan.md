# Sprint Plan: M-DX-XPKG-RESOLVE — Cross-Package Stdlib Function Resolution Bug

## Summary

Fix the runtime bug where pure AILANG stdlib functions (e.g., `asString` from `std/json`) silently return wrong results when called cross-package. Investigation-first approach: reproduce, identify root cause, fix, verify.

**Duration:** 2 days
**Dependencies:** None
**Risk Level:** Medium (root cause not yet confirmed — investigation may reveal unexpected complexity)

## Current Status Analysis

### Completed Recently
- Debug ghost effect (auto-grant, ceiling bypass, stderr output): ~400 LOC in 2 days
- Package install @latest support: ~200 LOC in 1 day
- --log-level flag: ~150 LOC in 0.5 days

### Velocity
- Recent average: ~200 LOC/day
- Estimated capacity: ~400 LOC for this 2-day sprint
- Bug fix sprints tend to be lower LOC but higher investigation time

### Remaining from Design Doc
- M1: Reproduction + root cause identification (~100 LOC tests + debug tracing)
- M2: Fix implementation (~100-200 LOC depending on root cause)
- M3: Regression tests + verification (~100 LOC)

## Proposed Milestones

### Milestone 1: M1_REPRODUCE — Reproduce and Identify Root Cause
**Goal:** Create a minimal failing test and identify which of the three hypotheses is correct
**Estimated:** 100 LOC tests + 50 LOC debug tracing = 150 LOC
**Duration:** 0.5 days

**Tasks:**
1. Create `internal/runtime/cross_package_resolve_test.go` with minimal two-module test:
   - Module A defines ADT `Wrapper = Wrap(string) | Empty`, plus `unwrap(w)` using pattern match, plus `getAndUnwrap(obj, key)` calling `unwrap` internally
   - Module B imports `unwrap` from A and calls it directly
   - Assert both return same result
2. If test passes (can't reproduce with simple case), escalate to std/json-specific test:
   - Load actual `std/json` module, create a second module that imports `asString`
   - Call `asString(JString("hello"))` cross-module
3. Add `DEBUG_RESOLVER=1` env-gated tracing to `moduleGlobalResolver.ResolveValue()` to log resolution path
4. Run test with tracing to identify which resolution path is taken and where it fails

**Acceptance Criteria:**
- [ ] Minimal test that reproduces the bug (test FAILS, proving the bug exists)
- [ ] Root cause identified and documented (which hypothesis was correct)
- [ ] Debug tracing available via `DEBUG_RESOLVER=1`
- [ ] All existing tests still pass (`make test`)

**Risks:**
- Bug may not reproduce in isolated test → Mitigation: use actual std/json module loading, escalate to serve-api integration test
- Root cause may be none of the three hypotheses → Mitigation: debug tracing will reveal actual resolution path

**Pause Point:** After M1, review findings with user before proceeding. The fix approach depends on the root cause.

### Milestone 2: M2_FIX — Implement Fix
**Goal:** Fix the root cause so cross-package stdlib function calls work correctly
**Estimated:** 100-200 LOC (depends on root cause)
**Duration:** 0.5-1 day

**Tasks (contingent on M1 findings):**

**If Hypothesis A (resolver context leak):**
1. Add `Resolver` field to `FunctionValue` in `internal/eval/value.go`
2. Capture resolver during closure creation in `evalCoreLambda` and `buildClosure`
3. Swap resolver in `evalCoreApp` when calling a function with a stored resolver
4. Restore caller's resolver after function body returns

**If Hypothesis C (stale export binding):**
1. Trace `GetExport` path in `internal/runtime/module.go`
2. Ensure exported values are the same references as closure-captured values
3. Fix any copy/separate-storage that breaks reference identity

**Acceptance Criteria:**
- [ ] Reproduction test from M1 now PASSES
- [ ] `getString` and direct `asString` calls return identical results cross-module
- [ ] All existing tests pass (`make test`)
- [ ] `make lint` clean

**Risks:**
- Resolver-in-closure approach may break existing behavior → Mitigation: extensive test run, only apply resolver when non-nil
- Performance impact of storing resolver in every FunctionValue → Mitigation: resolver is a lightweight pointer, negligible cost

### Milestone 3: M3_VERIFY — Regression Tests and Verification
**Goal:** Comprehensive tests covering the fix, verify examples, update docs
**Estimated:** 100 LOC tests + docs
**Duration:** 0.5 days

**Tasks:**
1. Add test cases covering multiple cross-package patterns:
   - Package function calling stdlib ADT matcher (the original bug)
   - Package function calling stdlib function that calls another stdlib function (transitive)
   - Package function using `match` on stdlib ADT constructors
   - Multiple packages calling same stdlib function
2. Run `make verify-examples` — all examples pass
3. Run `make ci` — full CI verification
4. Update design doc status from "Planned" to "Implemented" with root cause findings
5. Update CHANGELOG.md with bug fix entry

**Acceptance Criteria:**
- [ ] At least 4 cross-package resolution test cases passing
- [ ] `make verify-examples` passes
- [ ] `make ci` passes
- [ ] Design doc updated with actual root cause and fix description
- [ ] CHANGELOG.md updated

## Success Metrics
- Test coverage: New tests cover the cross-package resolution path
- Examples passing: All existing examples still work
- Documentation: Design doc updated, CHANGELOG entry added
- All tests passing
- All linting passing

## Dependencies
- None — this is a standalone bug fix

## Open Questions
- Will the reproduction test catch the bug in isolation, or does it only manifest under serve-api's module loading path? (Answered in M1)
- Is the fix applicable to all three hypotheses, or do we need hypothesis-specific approaches? (Answered in M1)

## Notes
- The design doc identifies three hypotheses. M1's primary job is to narrow to ONE confirmed root cause before implementing.
- Hypothesis B (constructor ModulePath mismatch) has been **eliminated** — pattern matching only checks `CtorName`, not `ModulePath` (eval_patterns.go:186).
- This sprint has a deliberate pause point after M1 to review findings before committing to a fix approach.

## Key Files

| File | Purpose |
|------|---------|
| `internal/runtime/resolver.go` | Module global resolver — primary investigation target |
| `internal/runtime/runtime.go:227-383` | Module evaluation and environment setup |
| `internal/eval/eval_operations.go:37-171` | Function application — resolver context during body eval |
| `internal/eval/eval_expressions.go:86-115` | Var and VarGlobal resolution |
| `internal/eval/eval_patterns.go:181-188` | Constructor pattern matching |
| `internal/eval/value.go:196-206` | FunctionValue definition — may need Resolver field |
| `std/json.ail:102-180` | asString and getString definitions |
