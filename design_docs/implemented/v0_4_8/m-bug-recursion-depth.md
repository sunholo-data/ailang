# M-BUG-RECURSION-DEPTH: Fix Unexpected Recursion Depth Overflow

**Status**: Planned
**Target**: v0.4.9
**Priority**: P1 (Medium) - Affects functional programming patterns
**Estimated**: 2-3 days
**Dependencies**: None
**Reported by**: stapledons_voyage (agent inbox)

## AI-First Alignment Check

| Principle | Impact | Score | Notes |
|-----------|--------|-------|-------|
| Reduce Syntactic Noise | + | +1 | Enables idiomatic recursive patterns |
| Preserve Semantic Clarity | 0 | 0 | Recursion semantics unchanged |
| Increase Determinism | + | +1 | Predictable stack behavior |
| Lower Token Cost | 0 | 0 | No impact on token usage |
| **Net Score** | | **+2** | **Decision: Move forward** |

## Problem Statement

Simple recursive list generation hits 10000 recursion limit even for small inputs like n=64.

**Reproduction:**
```ailang
pure func generate(n: int, f: int -> Tile) -> [Tile] {
  match n {
    0 => [],
    _ => f(n-1) :: generate(n-1, f)
  }
}

-- Expected: ~64 recursive calls for 8x8 grid
-- Actual: "recursion depth exceeded: 10000"
```

**Current State:**
- Recursion limit is 10000 (reasonable for deep recursion protection)
- Simple list generation for n=64 should only need ~64 calls
- Something is causing exponential stack growth or lazy evaluation issues

**Impact:**
- Blocks game development (can't generate even small grids)
- Forces workarounds like batching or iteration
- Makes functional idioms impractical

## Root Cause Analysis

Possible causes:

### 1. Lazy Evaluation Stack Explosion
If `::` (cons) is lazy and `generate(n-1, f)` is evaluated before returning:
- Each `::` creates a thunk
- Evaluating the list forces all thunks at once
- Stack grows to n * (overhead per thunk)

### 2. Non-Tail-Recursive Pattern
The pattern `f(n-1) :: generate(n-1, f)` is NOT tail-recursive:
- `generate(n-1, f)` must complete before `::` can run
- Stack frame held for each level

### 3. Closure Capture Overhead
If `f` captures environment, each call may allocate extra frames.

### 4. Interpreter Stack Counting Bug
The recursion counter may be incrementing incorrectly:
- Counting per-expression instead of per-call
- Counting thunk creation as recursion

## Goals

**Primary Goal:** Allow recursive list generation for reasonable sizes (n≤1000) without hitting recursion limit.

**Success Metrics:**
- `generate(64, f)` completes successfully
- `generate(1000, f)` completes successfully
- No regression in stack overflow protection for actual deep recursion
- Clear error message when limit legitimately exceeded

## Solution Design

### Overview

Investigate the actual stack behavior and implement one of:
1. **Tail-call optimization (TCO)** for tail-recursive functions
2. **Fix lazy evaluation** to not consume stack
3. **Trampoline transformation** for non-tail recursion
4. **Adjust recursion counter** if it's counting wrong

### Investigation Plan

**Step 1: Reproduce and Measure**
```ailang
-- Minimal test case
pure func count(n: int) -> int {
  match n {
    0 => 0,
    _ => 1 + count(n-1)
  }
}

-- Test at various sizes
count(10)    -- Should work
count(100)   -- Should work
count(1000)  -- Should work
count(10000) -- Should hit limit
```

**Step 2: Trace Stack Usage**
Add debug output to evaluator showing:
- Current recursion depth
- Function being called
- Where depth increments

**Step 3: Identify Fix**
Based on findings, choose appropriate solution.

### Implementation Options

**Option A: Tail-Call Optimization**
- Detect tail-recursive patterns in elaboration
- Transform to iterative loop in Core
- Estimated: 200-300 LOC

**Option B: Increase Limit Intelligently**
- If limit is per-thunk instead of per-call, fix counting
- Or make limit configurable via flag
- Estimated: 20-50 LOC

**Option C: Accumulator Pattern in Stdlib**
- Provide `std/list.generate` with tail-recursive implementation
- Document pattern for users
- Estimated: 50 LOC

### Implementation Plan

**Phase 1: Investigation** (~4 hours)
- [ ] Create minimal reproduction
- [ ] Add debug tracing to evaluator
- [ ] Identify where stack is consumed
- [ ] Determine root cause

**Phase 2: Fix** (~6 hours)
- [ ] Implement chosen solution
- [ ] Add tests for boundary cases
- [ ] Ensure stack protection still works

**Phase 3: Documentation** (~2 hours)
- [ ] Document recursion limits
- [ ] Add idiomatic patterns to teaching prompt
- [ ] Update examples

### Files to Modify

**Investigation:**
- `internal/eval/eval.go` - Add debug tracing

**Potential fixes:**
- `internal/eval/eval.go` - Fix recursion counting (~30 LOC)
- `internal/elaborate/elaborate.go` - TCO detection (~100 LOC)
- `internal/core/core.go` - Loop construct for TCO (~50 LOC)
- `stdlib/std/list.ail` - Tail-recursive generate (~20 LOC)

## Examples

### Example 1: List Generation (Reported Case)

**Current (Fails):**
```ailang
pure func generate(n: int, f: int -> Tile) -> [Tile] {
  match n {
    0 => [],
    _ => f(n-1) :: generate(n-1, f)
  }
}

let grid = generate(64, \i. Tile(i))  -- ERROR: recursion depth exceeded
```

**After Fix:**
```ailang
-- Same code works
let grid = generate(64, \i. Tile(i))  -- Returns [Tile] with 64 elements
```

### Example 2: Tail-Recursive Alternative

**Workaround (Accumulator Pattern):**
```ailang
pure func generate_acc(n: int, f: int -> Tile, acc: [Tile]) -> [Tile] {
  match n {
    0 => acc,
    _ => generate_acc(n-1, f, f(n-1) :: acc)
  }
}

pure func generate(n: int, f: int -> Tile) -> [Tile] {
  generate_acc(n, f, [])
}
```

This pattern IS tail-recursive and should work even without TCO.

## Success Criteria

- [ ] `generate(64, f)` completes without error
- [ ] `generate(1000, f)` completes without error
- [ ] Deep recursion (10000+) still triggers limit
- [ ] Clear error message shows actual depth
- [ ] All existing tests pass
- [ ] Example added to examples/

## Testing Strategy

**Unit tests:**
- Recursion at various depths (10, 100, 1000, 5000, 10001)
- Tail-recursive vs non-tail-recursive patterns
- List generation with different element types

**Integration tests:**
- Game grid generation scenario
- Nested recursive calls

**Manual testing:**
- Run stapledons_voyage code after fix

## Non-Goals

- **Not in this fix:**
  - Full TCO for all tail-recursive functions (if scope is just counting fix)
  - Arbitrary stack size configuration
  - Trampolining for mutually recursive functions

## Timeline

**Day 1** (4 hours):
- Investigation and root cause identification

**Day 2** (6 hours):
- Implement fix based on findings

**Day 3** (2 hours):
- Testing and documentation

**Total: ~12 hours across 3 days**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Root cause unclear | High | Thorough investigation before coding |
| TCO scope creep | Medium | Start with minimal fix, defer full TCO |
| Breaking stack protection | High | Preserve limit for actual deep recursion |

## References

- stapledons_voyage bug report (agent inbox, 2025-11-28)
- `internal/eval/eval.go` - Evaluator with recursion limit
- Tail-call optimization literature

## Future Work

- Full tail-call optimization
- Configurable stack limits
- Better stack traces on overflow

---

**Document created**: 2025-11-28
**Last updated**: 2025-11-28
