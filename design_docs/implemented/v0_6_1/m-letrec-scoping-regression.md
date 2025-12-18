# M-LETREC-SCOPING: Fix Letrec Recursive Binding Scope Regression

**Status**: IMPLEMENTED
**Target**: v0.6.1
**Priority**: P1 (High - breaks documented feature)
**Estimated**: 2-4 hours
**Actual**: ~2 hours
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

### Original Hypothesis (WRONG)

**Suspected locations to investigate:**

1. **Type Checker** (`internal/types/typechecker.go`)
   - `inferLetRec` or similar - does it add binding to environment before checking body?

2. **Elaboration** (`internal/elaborate/elaborate.go`)
   - `case *ast.LetRec:` - does it elaborate in recursive environment?

3. **Parser** (`internal/parser/parser.go`)
   - Less likely - syntax parses, just scoping is broken

### Actual Root Cause: MONOMORPHIZATION CACHE KEY COLLISION

**Discovery process:**
1. Type checking and elaboration were working correctly
2. Bug only manifested at runtime, not during compilation
3. Single letrecs worked; sequential letrecs failed
4. `--no-mono` flag made the bug disappear → monomorphization was the culprit

**The Bug (in `internal/pipeline/specialize_lambda.go`):**

```go
// WRONG - All lambdas shared the same cache key!
key := SpecializationKey{
    DefSym:           "(lambda)",  // <-- PROBLEM: same for ALL lambdas
    TypesFingerprint: fingerprint,
}
```

**What happened:**
1. First letrec: `a = \n. if n == 0 then 1 else n * a(n - 1)` specializes, cached with key `{DefSym: "(lambda)", Type: "int"}`
2. Second letrec: `b = \n. if n == 0 then 1 else n * b(n - 1)` ALSO has key `{DefSym: "(lambda)", Type: "int"}`
3. Cache hit! Returns `a`'s specialized body for `b`
4. `b`'s body now tries to call `a` instead of `b`
5. Runtime error: "undefined variable: a"

**Why this was hard to find:**
- Elaboration output was CORRECT (distinct Var nodes for `a` and `b`)
- Type checking was CORRECT
- Only at runtime, after monomorphization, did wrong code execute
- Single letrecs worked (no cache collision)
- Non-recursive lambdas might produce wrong results silently without crashing

## Solution Design

### Overview

**The fix:** Include lambda's unique NodeID in the specialization cache key to prevent different lambdas with the same type from sharing cached specialized bodies.

### The Fix (in `internal/pipeline/specialize_lambda.go`)

```go
// CORRECT - Each lambda has unique cache key
key := SpecializationKey{
    DefSym:           fmt.Sprintf("(lambda@%d)", lambda.ID()),  // Include NodeID!
    TypesFingerprint: fingerprint,
}
```

### Why This Works

1. Each lambda node in Core AST has a unique NodeID (assigned by `freshNodeID()`)
2. Including NodeID in cache key means `(lambda@42)` ≠ `(lambda@43)` even with same type
3. Different lambdas now get their own cached specializations
4. M-LETREC-SCOPING comment documents why this is critical

### Files Modified

**The actual fix (1 line):**
- `internal/pipeline/specialize_lambda.go:84` - Include lambda ID in cache key

## Broader Impact & Related Issues

### What This Bug Exposed

This was more far-reaching than just the sequential letrecs edge case:

1. **ANY two anonymous lambdas with same type would collide:**
   - `let f = \x. x + 1; let g = \x. x * 2` - both `int -> int`
   - `map(\x. x+1, xs); map(\x. x*2, ys)` - both mapper lambdas
   - Higher-order functions receiving multiple callbacks

2. **Why letrec exposed it:** The recursive case made it obvious because:
   - Lambda body for `b` (should call `b`) got swapped with cached body for `a` (calls `a`)
   - Immediate "undefined variable: a" error
   - Non-recursive lambdas would produce **wrong results silently**

### Related Caching to Audit

Check these for similar cache key collision risks:

| Location | Status | Notes |
|----------|--------|-------|
| `specialize_lambda.go` | ✅ FIXED | Now includes NodeID |
| `specialize.go` named functions | ✅ OK | Uses function name in key |
| `type_cache` in type checker | ⚠️ CHECK | May have similar issue with polymorphic lambdas |

### Recommended Test Coverage

Add regression tests for:
1. ✅ Sequential letrecs with same type
2. 🔲 Sequential let bindings with lambdas of same type
3. 🔲 Higher-order functions with multiple callbacks
4. 🔲 Nested lambdas with same type at different scopes

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

- [x] `letrec factorial = \n. ... factorial(n-1) ...` compiles and runs
- [x] Sequential letrecs work: `let r1 = letrec a = ... in a(3); let r2 = letrec b = ... in b(3)`
- [x] `make test` passes (all tests pass)
- [x] `make verify-examples` passes (61/62 - cli_args_demo.ail fails due to unrelated nullary function issue)
- [x] No regressions in existing functionality

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
**Last updated**: 2025-12-18
**Implemented**: 2025-12-18

## Implementation Notes

**Key learnings:**
1. When debugging "scope" bugs, check ALL compilation phases including monomorphization
2. `--no-mono` flag is a powerful diagnostic tool for isolating monomorphization bugs
3. Cache keys for anonymous constructs MUST include unique identifiers (NodeID, hash, etc.)
4. Bugs that cause "wrong code" silently are much worse than bugs that crash - the recursive case crashed which made debugging possible
