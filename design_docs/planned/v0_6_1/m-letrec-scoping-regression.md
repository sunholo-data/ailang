# M-LETREC-SCOPING: Fix Letrec Recursive Binding Scope Regression

**Status**: Planned
**Target**: v0.6.1
**Priority**: P1 (High - breaks documented feature)
**Estimated**: 2-4 hours
**Dependencies**: None

## AI-First Alignment Check

**Score this feature against AILANG's core principles:**

| Principle | Impact | Score | Notes |
|-----------|--------|-------|-------|
| Reduce Syntactic Noise | 0 | 0 | Bug fix - no change |
| Preserve Semantic Clarity | + | +1 | Restores expected recursive semantics |
| Increase Determinism | + | +1 | Fixes non-deterministic failure mode |
| Lower Token Cost | 0 | 0 | Bug fix - no change |
| **Net Score** | | **+2** | **Decision: Move forward** |

**Decision rule:** Net score > +1 → Move forward

## Problem Statement

**Regression Bug**: `letrec` bindings no longer make the recursive variable visible within its own body.

**Symptom:**
```
$ ./bin/ailang run --caps IO --entry main examples/runnable/letrec_recursion.ail
Error: undefined variable: factorial at examples/runnable/letrec_recursion.ail:16:67
```

**Failing Code:**
```ailang
-- This should work but fails with "undefined variable: factorial"
let fact_result = letrec factorial = \n. if n == 0 then 1 else n * factorial(n - 1) in factorial(5);
```

**Current State:**
- `letrec` was implemented in v0.3.5 per design doc [20251013_letrec_surface_syntax.md](../../implemented/v0_3/20251013_letrec_surface_syntax.md)
- At some point between v0.3.5 and v0.6.0, a regression broke recursive scoping
- The `letrec` keyword parses correctly, but elaboration/type-checking doesn't make the binding visible in its own body
- Example file `examples/runnable/letrec_recursion.ail` documents the expected behavior

**Impact:**
- **Who**: All users wanting recursive lambdas
- **Severity**: High - documented feature completely broken
- **Workaround**: Must use module-level `func` declarations instead

## Goals

**Primary Goal:** Restore `letrec` functionality so recursive bindings are visible in their own body.

**Success Metrics:**
- `letrec factorial = \n. ... factorial(n-1) ...` works
- `examples/runnable/letrec_recursion.ail` passes (currently fails)
- All 6 test cases in the example file pass:
  - Fibonacci(10) = 55
  - Factorial(5) = 120
  - Sum(1..100) = 5050
  - GCD(48, 18) = 6
  - 2^10 = 1024
  - Is 42 even? true

## Root Cause Analysis

**Suspected locations to investigate:**

1. **Type Checker** (`internal/types/typechecker.go`)
   - `inferLetRec` or similar - does it add binding to environment before checking body?
   - Compare with how module `func` declarations handle recursion

2. **Elaboration** (`internal/elaborate/elaborate.go`)
   - `case *ast.LetRec:` - does it elaborate in recursive environment?
   - Should create `core.LetRec` with binding visible in value

3. **Parser** (`internal/parser/parser.go`)
   - Less likely - syntax parses, just scoping is broken

**Investigation steps:**
```bash
# Debug elaboration
DEBUG_ELAB=1 ./bin/ailang run --caps IO --entry main examples/runnable/letrec_recursion.ail

# Compare with working module func recursion
./bin/ailang run --caps IO --entry main examples/runnable/test_fizzbuzz.ail  # Uses module func
```

## Solution Design

### Overview

Restore recursive scoping by ensuring the `letrec` binding name is added to the type environment BEFORE type-checking the value expression.

### Expected Code Flow

**Correct behavior (how `func` works):**
1. Register function name in environment with fresh type variable
2. Type-check function body (can reference own name)
3. Unify inferred type with registered type variable
4. Type-check usage sites

**Bug behavior (what `letrec` is doing):**
1. Type-check value expression (WRONG - name not in env yet!)
2. Register binding
3. Fail with "undefined variable"

### Implementation Plan

**Phase 1: Diagnosis** (~1 hour)
- [ ] Add debug logging to `inferLetRec` in type checker
- [ ] Trace environment before/after binding registration
- [ ] Compare with `inferFunc` or module function handling
- [ ] Identify exact line where scoping breaks

**Phase 2: Fix** (~1-2 hours)
- [ ] Modify type checker to add binding BEFORE checking value
- [ ] Use RefCell pattern (or equivalent) for deferred type
- [ ] Ensure elaboration creates correct `core.LetRec` node

**Phase 3: Testing** (~1 hour)
- [ ] Verify `examples/runnable/letrec_recursion.ail` passes
- [ ] Add unit test for letrec scoping
- [ ] Run full test suite to catch regressions
- [ ] Update example verification (`make verify-examples`)

### Files to Modify

**Primary suspects:**
- `internal/types/typechecker.go` - Fix recursive environment setup (~20 LOC change)
- `internal/elaborate/elaborate.go` - Ensure correct Core node generation (~10 LOC)

**Testing:**
- `internal/types/typechecker_test.go` - Add letrec scoping test (~30 LOC)

## Examples

### Example 1: Basic Recursive Lambda

**Currently fails:**
```ailang
letrec factorial = \n. if n == 0 then 1 else n * factorial(n - 1) in factorial(5)
-- Error: undefined variable: factorial
```

**Should produce:**
```
120
```

### Example 2: Full Example File

**File:** `examples/runnable/letrec_recursion.ail`

**Expected output:**
```
=== Letrec Recursive Lambdas (v0.3.5) ===
Fibonacci(10) = 55
Factorial(5) = 120
Sum(1..100) = 5050
GCD(48, 18) = 6
2^10 = 1024
Is 42 even? true
```

## Success Criteria

- [ ] `letrec factorial = \n. ... factorial(n-1) ...` compiles and runs
- [ ] `examples/runnable/letrec_recursion.ail` passes verification
- [ ] Unit test added for letrec recursive scoping
- [ ] `make test` passes
- [ ] `make verify-examples` shows improved pass rate (54/56 → 55/56)
- [ ] No regressions in existing functionality

## Testing Strategy

**Unit tests:**
- `TestLetRecScope` - Verify binding visible in own body
- `TestLetRecTypeInference` - Verify types inferred correctly
- `TestLetRecNestedLetrec` - Verify nested letrec works

**Integration tests:**
- Run `examples/runnable/letrec_recursion.ail` end-to-end

**Manual testing:**
```bash
# Quick smoke test
echo 'letrec fac = \n. if n == 0 then 1 else n * fac(n-1) in fac(5)' | ./bin/ailang repl
# Should print: 120
```

## Non-Goals

**Not in this fix:**
- Mutual recursion (`letrec f = ... and g = ...`) - separate feature
- Performance optimization of recursive calls - out of scope
- Better error messages for infinite recursion - separate issue

## Timeline

**Day 1** (2-4 hours):
- Diagnosis and root cause identification
- Implement fix
- Test with examples
- Run full test suite
- Document fix

**Total: ~2-4 hours (single session fix)**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Fix breaks module func recursion | High | Test both letrec and func after fix |
| Type inference becomes unsound | High | Add comprehensive type tests |
| Performance regression | Low | Unlikely for scoping fix |

## References

- [Original letrec design doc](../../implemented/v0_3/20251013_letrec_surface_syntax.md) - v0.3.5 implementation
- [M-R4 recursion implementation](../../implemented/v0_3_0/M-R4_recursion.md) - Core recursion mechanics
- Example file: `examples/runnable/letrec_recursion.ail`

## Future Work

- Mutual recursion: `letrec f = ... and g = ... in body`
- Better error messages for non-terminating recursion
- Tail call optimization for recursive lambdas

---

**Document created**: 2025-12-17
**Last updated**: 2025-12-17
