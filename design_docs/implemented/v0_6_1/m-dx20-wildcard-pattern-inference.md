# M-DX20: Wildcard Pattern Type Inference in List Patterns

**Status**: Implemented
**Target**: v0.6.1
**Priority**: P2 (Low - workaround exists)
**Estimated**: 2-3 hours
**Actual**: 30 minutes
**Dependencies**: None
**Reporter**: stapledons_voyage (agent message)

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | Type inference is deterministic |
| A2: Replayability | 0 | No impact on traces |
| A3: Effect Legibility | 0 | No effect changes |
| A4: Explicit Authority | 0 | No capability changes |
| A5: Bounded Verification | +1 | Clearer code without unused bindings |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | AI agents can use idiomatic patterns |
| A8: Minimal Syntax | +1 | _ is cleaner than unused named binding |
| A9: Cost Visibility | 0 | No cost changes |
| A10: Composability | 0 | No composability changes |
| A11: Structured Failure | 0 | No error handling changes |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +3** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Improves machine (AI agent) experience

## Problem Statement

Using wildcard `_` in list cons patterns causes type inference errors.

**Current State:**
```ailang
pure func hasElements(xs: [int]) -> bool =
    match xs {
        [] => false
        _ :: _ => true  -- Type error: "cannot unify int with *types.TList"
    }
```

**Workaround:**
```ailang
pure func hasElements(xs: [int]) -> bool =
    match xs {
        [] => false
        head :: tail => true  -- Works, but head and tail are unused
    }
```

**Impact:**
- Minor - workaround is straightforward
- Forces unused variable bindings
- Less idiomatic pattern matching

## Goals

**Primary Goal:** Allow wildcard `_` in list cons patterns.

**Success Metrics:**
- [x] `_ :: _` pattern works in match expressions
- [x] `_ :: rest` pattern works (wildcard head, named tail)
- [x] `head :: _` pattern works (named head, wildcard tail)
- [x] No unused variable warnings needed

---

## Implementation Report

### Root Cause Found

The issue was **NOT** in the type checker's pattern inference (which was already bidirectional). The bug was in the **elaboration phase**.

**Discovery:**
1. `_` is tokenized as `lexer.IDENT` (not a special token)
2. Parser's `parseBasePattern()` has a check for `curToken.Literal == "_"` but it's in the `default` case
3. Since `_` is an IDENT, it matches `case lexer.IDENT` first, never reaching the wildcard check
4. Result: `_` becomes `ast.Identifier{Name: "_"}` instead of `ast.WildcardPattern`
5. In the elaborator, `ast.Identifier` becomes `core.VarPattern{Name: "_"}`
6. When `_ :: _` is elaborated, both wildcards become `VarPattern("_")`
7. Type checker binds first `_` to element type (`int`), tries to bind second `_` to list type (`[int]`)
8. Unification fails: "cannot unify int with *types.TList"

### Fix Applied

Added a check in `internal/elaborate/patterns.go` to recognize `_` as a wildcard:

```go
case *ast.Identifier:
    // M-DX20: Check for wildcard pattern first
    // "_" is a wildcard that matches anything but binds nothing
    if p.Name == "_" {
        return &core.WildcardPattern{}, nil
    }
    // ... rest of identifier handling
```

### Files Modified

| File | Changes | LOC |
|------|---------|-----|
| `internal/elaborate/patterns.go` | Added wildcard check for `Identifier.Name == "_"` | +5 |

**Total: 5 LOC**

### Verification

All test cases from design doc verified working:

```ailang
-- Example 1: Basic
_ :: _           -- ✅ Works

-- Example 2: Mixed - Wildcard Head
_ :: rest        -- ✅ Works

-- Example 3: Mixed - Wildcard Tail
head :: _        -- ✅ Works

-- Example 4: Nested Wildcards
_ :: (_ :: _)    -- ✅ Works
```

### Why Initial Analysis Was Partially Wrong

The design doc hypothesized the issue was in **type inference** (bottom-up vs bidirectional). Investigation showed:

1. Type checker already uses bidirectional pattern checking (`checkPattern(pat, scrutType, ctx)`)
2. `WildcardPattern` handler correctly returns empty bindings
3. The bug was that `_` never became `WildcardPattern` in the first place

The fix was much simpler than expected - just 5 lines in the elaborator.

---

## Success Criteria

- [x] `_ :: _` pattern type-checks correctly
- [x] `_ :: rest` pattern type-checks correctly
- [x] `head :: _` pattern type-checks correctly
- [x] Nested wildcards work (`_ :: (_ :: _)`)
- [x] `_` still forces scrutinee to be list type (sound behavior)
- [x] No regressions in existing pattern matching tests
- [x] All existing tests pass
- [x] `make test` passes

## References

- [Design Axioms](/docs/references/axioms) - The 12 non-negotiable principles
- stapledons_voyage DX Feedback (agent message)

---

**Document created**: 2025-12-20
**Last updated**: 2025-12-21
**Implemented**: 2025-12-21
