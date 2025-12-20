# M-DX20: Wildcard Pattern Type Inference in List Patterns

**Status**: Planned
**Target**: v0.6.2
**Priority**: P2 (Low - workaround exists)
**Estimated**: 2-3 hours
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
- `_ :: _` pattern works in match expressions
- `_ :: rest` pattern works (wildcard head, named tail)
- `head :: _` pattern works (named head, wildcard tail)
- No unused variable warnings needed

## Solution Design

### Overview

The type checker needs to handle wildcard patterns specially in cons patterns, not attempting to unify them with specific types.

### Root Cause Analysis

The issue is likely in pattern type inference where `_` is being given a concrete type that conflicts with the expected element type. In a cons pattern `head :: tail`:
- `head` should have type `T` (element type)
- `tail` should have type `[T]` (list type)
- `_` should accept any type without constraint

### Implementation Plan

**Phase 1: Investigation** (~30 min)
- [ ] Create minimal repro case
- [ ] Trace type inference for `_ :: _` pattern
- [ ] Identify where unification fails

**Phase 2: Fix** (~1-2 hours)
- [ ] Modify pattern type inference to handle `_` specially
- [ ] Ensure `_` doesn't add constraints to type environment
- [ ] Test with various combinations (`_ :: _`, `_ :: rest`, `head :: _`)

**Phase 3: Testing** (~30 min)
- [ ] Add unit tests for wildcard patterns
- [ ] Verify existing pattern tests still pass
- [ ] Test with stapledons_voyage code

### Files to Modify

**Likely modified files:**
- `internal/types/infer_pattern.go` - Pattern type inference (~20 LOC)
- `internal/elaborate/pattern.go` - Pattern elaboration (~10 LOC)

## Examples

### Example 1: Check Non-Empty

**Before (fails):**
```ailang
pure func nonEmpty(xs: [int]) -> bool =
    match xs {
        [] => false
        _ :: _ => true  -- ERROR
    }
```

**After (works):**
```ailang
pure func nonEmpty(xs: [int]) -> bool =
    match xs {
        [] => false
        _ :: _ => true  -- OK
    }
```

### Example 2: Get Tail Only

```ailang
pure func getTail(xs: [int]) -> [int] =
    match xs {
        [] => []
        _ :: rest => rest  -- Discard head, keep tail
    }
```

## Success Criteria

- [ ] `_ :: _` pattern type-checks correctly
- [ ] `_ :: rest` pattern type-checks correctly
- [ ] `head :: _` pattern type-checks correctly
- [ ] No regressions in existing pattern matching tests
- [ ] All existing tests pass
- [ ] `make test` passes

## Testing Strategy

**Unit tests:**
- Test `_ :: _` with various list element types
- Test mixed wildcard/named patterns
- Test nested list patterns with wildcards

**Integration tests:**
- Compile stapledons_voyage code with wildcard patterns

## Non-Goals

**Not in this feature:**
- Wildcard patterns in other contexts (already working)
- Multiple wildcards with different inferred types

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Breaking existing pattern matching | High | Comprehensive test suite |
| Subtle type inference bugs | Medium | Add logging, careful testing |

## Related Documents

- Pattern matching implementation
- Type inference documentation

## References

- [Design Axioms](/docs/references/axioms) - The 12 non-negotiable principles
- stapledons_voyage DX Feedback (agent message)

---

**Document created**: 2025-12-20
**Last updated**: 2025-12-20
