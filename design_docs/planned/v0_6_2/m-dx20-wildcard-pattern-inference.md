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

## Key Insight: Typed Hole, Not "No Constraints"

**Important:** The fix is NOT "wildcard adds no constraints" (that can be unsound/too permissive).

**Correct behavior:**
- `_` does **not bind a name** into the environment
- `_` still **participates in unification** with the expected type (it just discards the value)

In other words: `_` should be a **typed hole** that accepts the type it's placed in, without introducing a variable.

## Root Cause Analysis

**The error message is revealing:**
```
cannot unify int with *types.TList
```

In a cons pattern `pHead :: pTail`, the checker should enforce:
- scrutinee type = `[T]`
- `pHead` has type `T`
- `pTail` has type `[T]`

The error strongly suggests pattern inference is accidentally inferring `_` in `_ :: _` as a list pattern (giving the head the list type instead of element type).

**Likely cause:** Pattern inference is **bottom-up** ("infer pattern type, then unify with scrutinee") rather than **bidirectional** ("given expected type, typecheck the pattern").

## Solution Design

### Best Approach: Bidirectional Pattern Typing

Instead of special-casing wildcard only in cons patterns, make pattern typing take an expected type:

```go
// Bidirectional pattern checking
func (tc *TypeChecker) checkPattern(p Pattern, expected Type) (env Env, err error)
```

**Then for each pattern kind:**

**Wildcard `_`:**
```go
case *WildcardPattern:
    // Return empty env, unify nothing beyond "it fits expected"
    return Env{}, nil
```

**Cons pattern `p1 :: p2`:**
```go
case *ConsPattern:
    // 1. Create fresh type variable T
    elemT := tc.freshTypeVar()
    // 2. Unify expected with [T]
    if err := tc.unify(expected, ListType(elemT)); err != nil {
        return nil, err
    }
    // 3. Check head against element type
    env1, err := tc.checkPattern(p.Head, elemT)
    if err != nil { return nil, err }
    // 4. Check tail against list type
    env2, err := tc.checkPattern(p.Tail, ListType(elemT))
    if err != nil { return nil, err }
    // 5. Merge environments
    return merge(env1, env2), nil
```

**Why this is better than "wildcard adds no constraints":**

`_` should still be consistent with context. Example:
```ailang
match xs {
    _ :: _ => true
}
```

This branch should **force `xs` to be a list type**. If `_` adds "no constraints" in the wrong way, you lose that signal and end up with ambiguous/incorrect inference.

### Minimal Patch (If No Refactor)

If you want the 2-3 hour patch without full bidirectional refactor:

In `inferConsPattern` (or whatever handles `::`):
1. Always derive `elemT` and `tailT = [elemT]` from the scrutinee/list context
2. When head/tail is wildcard, don't infer independently; just accept `elemT` / `tailT` and don't bind

**Prerequisite:** This requires having the scrutinee type available when typing the pattern. If not available, you need bidirectional checking.

### Implementation Plan

**Phase 1: Investigation** (~30 min)
- [ ] Create minimal repro case
- [ ] Trace type inference for `_ :: _` pattern
- [ ] Confirm whether pattern typing is bottom-up or bidirectional
- [ ] Identify exact location where unification fails

**Phase 2: Fix** (~1-2 hours)
- [ ] If bidirectional: ensure `checkPattern` passes expected type down
- [ ] If bottom-up: add scrutinee type context to cons pattern handler
- [ ] Wildcard should accept expected type without binding
- [ ] Test with various combinations (`_ :: _`, `_ :: rest`, `head :: _`)

**Phase 3: Testing** (~30 min)
- [ ] Add unit tests for wildcard patterns
- [ ] Verify existing pattern tests still pass
- [ ] Test with stapledons_voyage code

### Files to Modify

**Likely modified files:**
- `internal/types/infer_pattern.go` - Pattern type inference (~30 LOC)
- `internal/elaborate/pattern.go` - Pattern elaboration (~10 LOC)

## Examples

### Example 1: Basic - Check Non-Empty

```ailang
pure func nonEmpty(xs: [int]) -> bool =
    match xs {
        [] => false
        _ :: _ => true  -- Should work: wildcards accept int and [int]
    }
```

### Example 2: Mixed - Wildcard Head, Named Tail

```ailang
pure func getTail(xs: [int]) -> [int] =
    match xs {
        [] => []
        _ :: rest => rest  -- Discard head, keep tail
    }
```

### Example 3: Mixed - Named Head, Wildcard Tail

```ailang
pure func getHead(xs: [int]) -> int =
    match xs {
        [] => 0
        head :: _ => head  -- Keep head, discard tail
    }
```

### Example 4: Nested Wildcards

```ailang
pure func hasAtLeastTwo(xss: [[int]]) -> bool =
    match xss {
        [] => false
        _ :: (_ :: _) => true  -- Nested cons pattern with wildcards
        _ => false
    }
```

### Example 5: Polymorphic Scenario

```ailang
-- Should infer xs: [a] -> bool (if polymorphism is supported)
pure func nonEmpty(xs) =
    match xs {
        [] => false
        _ :: _ => true
    }
```

## Success Criteria

- [ ] `_ :: _` pattern type-checks correctly
- [ ] `_ :: rest` pattern type-checks correctly
- [ ] `head :: _` pattern type-checks correctly
- [ ] Nested wildcards work (`_ :: (_ :: _)`)
- [ ] `_` still forces scrutinee to be list type (sound behavior)
- [ ] No regressions in existing pattern matching tests
- [ ] All existing tests pass
- [ ] `make test` passes

## Testing Strategy

**Unit tests (prevent regressions):**

1. **Basic:**
```ailang
match xs { [] => false, _ :: _ => true }
```

2. **Mixed:**
```ailang
match xs { [] => [], _ :: rest => rest }
match xs { [] => 0, head :: _ => head }
```

3. **Nested:**
```ailang
match xss { [] => false, _ :: (_ :: _) => true, _ => false }
```

4. **Polymorphic (if supported):**
```ailang
pure func nonEmpty(xs) = match xs { [] => false, _ :: _ => true }
-- Should infer xs: [a] -> bool
```

**Integration tests:**
- Compile stapledons_voyage code with wildcard patterns

## Open Questions

1. **Is expected type available in pattern typing?**
   - If yes: Pass it through and use bidirectional checking
   - If no: Need to add scrutinee type to pattern typing context

2. **Should `_` work in constructor field patterns too?**
   - Example: `Some(_)` to match Some but ignore value
   - Recommendation: Yes, same principle applies (typed hole)

## Non-Goals

**Not in this feature:**
- Wildcard patterns in other contexts (already working)
- Multiple wildcards with different inferred types (already correct with expected-type approach)

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Breaking existing pattern matching | High | Comprehensive test suite |
| Subtle type inference bugs | Medium | Add logging, careful testing |
| Unsound "no constraints" approach | High | Use typed-hole approach instead |

## Related Documents

- Pattern matching implementation
- Type inference documentation

## References

- [Design Axioms](/docs/references/axioms) - The 12 non-negotiable principles
- stapledons_voyage DX Feedback (agent message)

---

**Document created**: 2025-12-20
**Last updated**: 2025-12-20
