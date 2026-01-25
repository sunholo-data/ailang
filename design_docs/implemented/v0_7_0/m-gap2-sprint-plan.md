# Sprint Plan: M-GAP2 Final Expression Lambda Arity Bug

## Sprint Summary

| Field | Value |
|-------|-------|
| **Sprint ID** | M-GAP2 |
| **Goal** | Fix type inference bug where bare identifier final expressions fail with multi-param lambdas |
| **Duration** | 1 day (~6 hours) |
| **Risk Level** | Medium (type system change, requires careful testing) |
| **Design Doc** | [m-gap2-lambda-arity-path-dependent-bug.md](m-gap2-lambda-arity-path-dependent-bug.md) |

## Problem Summary

**Root cause identified:** Multi-param lambda type inference fails when a module's final expression is a bare identifier referencing a computed value, but works when parenthesized.

| Expression | Result |
|------------|--------|
| `sum` | ❌ FAILS: "arity mismatch: 2 vs 1" |
| `(sum)` | ✅ WORKS |
| `(sum, x)` | ✅ WORKS |

## Technical Analysis

### Key Code Locations

1. **`internal/types/typechecker_functions.go:143-222`** - `inferLet()` handles let bindings with generalization
2. **`internal/types/typechecker_literals.go:59-105`** - `inferVar()` handles variable lookups with instantiation
3. **`internal/types/typechecker_core.go:308-394`** - `InferWithConstraints()` main entry point
4. **`internal/pipeline/pipeline_module.go:588-594`** - Module-level declaration processing

### Hypothesis

The issue is in how constraints are solved when the let body is a bare `Var` vs a compound expression:

1. **Bare `Var` path:**
   - `inferLet` generalizes `sum`'s type (with the lambda type)
   - `inferVar` instantiates the scheme when looking up `sum`
   - Constraints may not be fully resolved before generalization

2. **Parenthesized/Tuple path:**
   - Same generalization happens
   - But the tuple/parens creates additional constraint context
   - This forces proper constraint solving before the final expression is typed

### Investigation Strategy

```bash
# Add debug logging to track the difference
DEBUG_MONO_VERBOSE=1 ailang check ~/test_ailang/test_bare.ail 2>&1 | tee bare.log
DEBUG_MONO_VERBOSE=1 ailang check ~/test_ailang/test_parens.ail 2>&1 | tee parens.log
diff bare.log parens.log
```

## Milestones

### M1: Diagnosis & Minimal Reproduction (~2 hours)

**Goal:** Identify exactly where the type inference diverges

**Tasks:**
1. Add debug logging to `inferLet` showing:
   - Binding type before/after generalization
   - Constraint state at generalization boundary
   - Environment after binding is added
2. Add debug logging to `inferVar` showing:
   - Scheme being instantiated
   - Resulting monotype
3. Compare logs between bare vs parenthesized cases
4. Document the exact divergence point

**Acceptance Criteria:**
- [ ] Debug output shows where typing differs
- [ ] Root cause is documented with code line numbers
- [ ] Hypothesis confirmed or revised

**Files to Modify:**
| File | Change | Est. LOC |
|------|--------|----------|
| `internal/types/typechecker_functions.go` | Add debug logging | ~20 |
| `internal/types/typechecker_literals.go` | Add debug logging | ~10 |

### M2: Implement Fix (~2 hours)

**Goal:** Make bare identifier final expressions work identically to parenthesized

**Tasks:**
1. Based on M1 findings, implement fix (likely one of):
   - Ensure constraints are fully solved before `inferVar` returns for bare identifiers
   - Ensure proper instantiation of polymorphic bindings in let body position
   - Add missing substitution application step
2. Write unit test that captures the bug
3. Verify fix doesn't break existing tests

**Acceptance Criteria:**
- [ ] `test_bare.ail` passes
- [ ] `test_parens.ail` still passes
- [ ] `make test` passes (all existing tests)

**Likely Fix Location:**
```go
// internal/types/typechecker_functions.go - inferLet()
// After line 203: bodyNode, finalEnv, err := tc.inferCore(ctx, let.Body)

// Add: Ensure substitution is applied when body is just a Var
if _, isVar := let.Body.(*core.Var); isVar {
    // Force constraint solving for the body expression
    // ... (exact fix TBD from M1 diagnosis)
}
```

**Files to Modify:**
| File | Change | Est. LOC |
|------|--------|----------|
| `internal/types/typechecker_functions.go` | Fix generalization/instantiation | ~30 |
| `internal/types/typechecker_functions_test.go` | Add regression test | ~50 |

### M3: Regression Tests & Cleanup (~1.5 hours)

**Goal:** Ensure comprehensive test coverage and clean up debug code

**Tasks:**
1. Create regression test file `examples/runnable/bare_identifier_return.ail`
2. Add edge case tests:
   - Bare identifier with non-lambda binding (should work)
   - Bare identifier with 1-param lambda
   - Bare identifier with 2+ param lambda
   - Multiple let bindings with bare identifier return
3. Remove debug logging (or gate behind DEBUG flag)
4. Update CHANGELOG.md

**Acceptance Criteria:**
- [ ] New example file runs successfully
- [ ] All edge case tests pass
- [ ] `make test` passes
- [ ] `make lint` passes
- [ ] Debug output removed or gated

**Files to Create/Modify:**
| File | Change | Est. LOC |
|------|--------|----------|
| `examples/runnable/bare_identifier_return.ail` | New test file | ~30 |
| `internal/types/typechecker_functions.go` | Cleanup debug | -20 |
| `CHANGELOG.md` | Document fix | ~10 |

### M4: Documentation & Verification (~0.5 hours)

**Goal:** Document the fix and verify with real-world examples

**Tasks:**
1. Update design doc with implementation notes
2. Test with the original `no_loops_fold.ail` modified to use bare identifier
3. Test workaround removal (users no longer need parentheses)
4. Commit changes

**Acceptance Criteria:**
- [ ] Design doc updated with "Implemented" status
- [ ] Original example works with bare identifier
- [ ] Clean commit with all changes

## Estimated LOC

| Component | LOC |
|-----------|-----|
| Fix implementation | ~30 |
| Unit tests | ~50 |
| Example file | ~30 |
| Debug code (temp) | ~30 |
| Cleanup | -20 |
| **Total Net** | **~120** |

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Fix breaks other type inference | Medium | High | Comprehensive test suite, careful review |
| Root cause different than expected | Low | Medium | M1 diagnosis before implementation |
| Performance impact | Low | Low | Change is in constraint solving, not hot path |

## Success Criteria

- [ ] `sum` works identically to `(sum)` in final expression position
- [ ] All existing tests pass
- [ ] New regression tests added
- [ ] No "arity mismatch" errors for correct code
- [ ] CHANGELOG updated
- [ ] Design doc moved to implemented/

## Dependencies

- None (standalone bug fix)

## Workaround (Current)

Until fixed, users can wrap final expressions in parentheses:
```ailang
-- Instead of:
sum

-- Use:
(sum)
```

## Velocity Context

Based on recent work:
- v0.6.4 stdlib gaps: ~4 hours for 3 features
- Typical bug fix: 2-4 hours
- This estimate: 6 hours (conservative due to type system complexity)
