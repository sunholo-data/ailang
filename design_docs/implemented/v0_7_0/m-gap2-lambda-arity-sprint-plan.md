# Sprint Plan: M-GAP2 Fix Path-Dependent Lambda Arity Bug

## Summary

Fix a critical P0 bug where lambda syntax with `foldl` works in `examples/runnable/` but fails with "arity mismatch: 2 vs 1" when identical code is placed in different directories. This violates AILANG's determinism guarantee (Axiom A1).

**Duration:** 1 day (4-8 hours)
**Dependencies:** None
**Risk Level:** Medium (investigation phase may reveal unexpected root cause)
**Priority:** P0 (Critical)

## Current Status Analysis

### Completed Recently
- v0.6.3: OpenAI Responses API (~190 LOC)
- v0.6.3: Deprecate ailang-agent (~1,760 LOC removed)
- v0.6.3: Enhanced tracing DX (~260 LOC)
- v0.6.2: OpenTelemetry integration (~500 LOC)

### Velocity
- Recent average: ~400-600 LOC/day for focused work
- This is primarily investigation/fix work, not new features
- Estimated capacity: 150-300 LOC for fix + tests

### Bug Context
- **Symptom:** `\acc x. expr` lambda works in `examples/runnable/` but fails elsewhere
- **Error:** "arity mismatch: 2 vs 1"
- **Workaround:** Use verbose `func(acc: int, x: int) -> int { ... }` syntax

### Key Files Identified
| File | LOC | Role |
|------|-----|------|
| `internal/parser/parser_expr.go` | ~600 | `parseBackslashLambda()` - curried sugar expansion |
| `internal/types/unification_types.go` | 263 | `unifyFunctions()` - arity mismatch error origin |
| `internal/elaborate/file.go` | 762 | Lambda elaboration to Core |
| `internal/loader/loader.go` | 635 | Module path resolution |

### Root Cause Hypothesis

The parser converts `\acc x. body` to nested lambdas `\acc. \x. body` (curried form). This means:
- `\acc x. expr` becomes type `a -> (b -> c)` (1-arity returning function)
- But `foldl` expects `(b, a) -> b` (2-arity function)

This should ALWAYS fail with this error, not just in certain paths. The path-dependent behavior suggests:
1. **Cache interference** - Some compilation artifact differs by path
2. **Import resolution** - `std/list.foldl` resolves differently
3. **Module context** - Type checking context differs

## Proposed Milestones

### Milestone 1: Diagnosis and Minimal Reproduction
**Goal:** Identify the exact root cause of path-dependent behavior
**Estimated:** 50 LOC diagnostic scripts + 50 LOC test cases = 100 LOC
**Duration:** 2-4 hours

**Tasks:**
- Hour 1: Create minimal reproduction test
  - Copy exact code to `/tmp/`, project root, `tests/`
  - Verify which locations fail vs succeed
  - Clear any caches: `rm -rf ~/.ailang/cache/`

- Hour 2: Enable debug output and compare
  ```bash
  DEBUG_PARSER=1 ailang check examples/runnable/no_loops_fold.ail 2>&1 > working.log
  DEBUG_PARSER=1 ailang check internal/dashboard_transforms/test_fold.ail 2>&1 > failing.log
  diff working.log failing.log
  ```

- Hour 3: Compare AST and type output
  ```bash
  ailang debug ast working.ail --show-types > working_ast.txt
  ailang debug ast failing.ail --show-types > failing_ast.txt
  diff working_ast.txt failing_ast.txt
  ```

- Hour 4: Check stdlib import resolution
  - Verify `foldl` type is identical in both contexts
  - Check module path canonicalization
  - Test with explicit type annotations

**Acceptance Criteria:**
- [ ] Minimal reproduction documented
- [ ] Root cause identified (cache/import/module/parser)
- [ ] Hypothesis confirmed with debug output
- [ ] Ready to implement fix

**Risks:**
- Investigation may reveal multiple contributing factors
- Mitigation: Time-box to 4 hours, document findings even if incomplete

### Milestone 2: Implement Fix
**Goal:** Fix the path-dependent behavior so lambda syntax works everywhere
**Estimated:** 100 LOC fix + 100 LOC tests = 200 LOC
**Duration:** 2-3 hours

**Tasks (based on diagnosis - one of):**

**If Cache Issue:**
- Clear path-dependent caching in `internal/pipeline/`
- Ensure cache keys include full context (module path, version)
- Add cache invalidation tests

**If Import Resolution:**
- Fix `internal/loader/loader.go` path normalization
- Ensure stdlib paths resolve consistently
- Add import resolution tests

**If Module Context:**
- Fix type checker context in `internal/types/`
- Ensure canonical path handling
- Add module context tests

**If Parser Bug:**
- Fix `parseBackslashLambda()` in `internal/parser/parser_expr.go`
- Ensure lambda arity is computed correctly
- Add parser unit tests

**Acceptance Criteria:**
- [ ] Root cause fixed in appropriate file
- [ ] `\acc x. expr` works in all file locations
- [ ] Existing tests continue to pass
- [ ] New regression tests added

**Risks:**
- Fix may have unintended side effects
- Mitigation: Run full test suite, verify all examples

### Milestone 3: Regression Tests and Documentation
**Goal:** Ensure bug cannot recur and document the fix
**Estimated:** 100 LOC tests + 50 LOC docs = 150 LOC
**Duration:** 1-2 hours

**Tasks:**
- Create comprehensive regression test suite
  ```bash
  mkdir -p tests/regression/lambda_arity/{root,nested/deep,absolute}
  # Each contains identical lambda test
  ```
- Test edge cases:
  - [ ] Lambda in REPL vs file
  - [ ] Lambda in imported module vs main module
  - [ ] Lambda with 1, 2, 3+ parameters
  - [ ] Nested lambdas `\x. \y. \z. ...`
- Update CHANGELOG.md with fix details
- Update GAPS_DISCOVERED.md to mark GAP-2 as resolved
- Create example file `examples/runnable/lambda_foldl.ail`

**Acceptance Criteria:**
- [ ] Regression test suite passes
- [ ] All edge cases covered
- [ ] CHANGELOG.md updated
- [ ] GAP-2 marked as resolved
- [ ] Example file created and working

**Risks:**
- None significant - documentation phase

## Success Metrics

- Test coverage: All new regression tests passing
- Examples passing: All 48+ existing examples + new lambda_foldl.ail
- Documentation:
  - [ ] CHANGELOG.md updated
  - [ ] GAPS_DISCOVERED.md updated
  - [ ] Design doc moved to `implemented/`
- All tests passing: `make test` clean
- All linting passing: `make lint` clean

## Dependencies

- None - this is a standalone critical bug fix

## Open Questions

1. **Is `examples/runnable/no_loops_fold.ail` actually working?**
   - Design doc claims it works but file doesn't exist in current worktree
   - Need to verify actual behavior first

2. **What is the exact code that works vs fails?**
   - `internal/dashboard_transforms/event_formatter.ail` uses workaround
   - Need to create actual failing test case

3. **Is the curried lambda expansion the real issue?**
   - `\acc x. expr` -> `\acc. \x. expr` is correct
   - Issue may be in type unification, not parsing

## Notes

- This is the root cause of GAP-3 documented in GAPS_DISCOVERED.md
- Fixing this eliminates need for verbose `func` workaround
- Critical for AILANG's determinism guarantee (Axiom A1)
- Priority P0 - should be fixed before v0.6.4 release

## Investigation Quick Reference

```bash
# Clear caches
rm -rf ~/.ailang/cache/ /tmp/ailang*

# Debug parser
DEBUG_PARSER=1 ailang check file.ail

# Debug type checking
DEBUG_STRICT=1 DEBUG_MONO_VERBOSE=1 ailang check file.ail

# Compare ASTs
ailang debug ast file.ail --show-types --compact

# Check foldl type
ailang repl
:type foldl
```

## Timeline

| Phase | Duration | Deliverable |
|-------|----------|-------------|
| M1: Diagnosis | 2-4 hours | Root cause identified |
| M2: Fix | 2-3 hours | Bug fixed, tests passing |
| M3: Docs | 1-2 hours | Regression tests, documentation |
| **Total** | **4-8 hours** | **Path-independent lambda behavior** |
