# M-PERF1: Effect Checker Performance for Large Array Literals

**Status**: Implemented
**Target**: v0.5.8
**Priority**: P0 (High - DX blocker for game development)
**Estimated**: 4-6 hours
**Dependencies**: None
**Reporter**: stapledons_voyage project

## AI-First Alignment Check

| Principle | Impact | Score | Notes |
|-----------|--------|-------|-------|
| Reduce Syntactic Noise | 0 | 0 | No syntax changes |
| Preserve Semantic Clarity | 0 | 0 | Semantics unchanged |
| Increase Determinism | 0 | 0 | Behavior unchanged |
| Lower Token Cost | + | +1 | Faster feedback loop = less iteration tokens |
| **Net Score** | | **+1** | **Decision: Move forward** |

## Problem Statement

When running `ailang check sim/bridge.ail`, the effect checker hangs indefinitely (30+ seconds, requires kill).

**Reproducer:**
```ailang
module sim/bridge

import std/array as A

-- 16x12 bridge tile layout (192 elements)
let tileLayout = A.fromList([
  1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
  -- ... 192 total int elements
])

-- 16x12 walkable map (192 elements)
let walkableMap = A.fromList([
  true, true, true, true, true, true, true, true,
  -- ... 192 total bool elements
])

-- More definitions using these arrays...
export func main() -> () ! IO = ...
```

**Current State:**
- Type checking completes successfully (~2 seconds)
- Effect checking hangs indefinitely (killed after 30+ seconds)
- Other modules without large array literals compile in <2 seconds

**Impact:**
- **Who**: Game developers using AILANG for tile-based games
- **Severity**: DX blocker - cannot use inline array literals for level data
- **Workarounds**: Move data to Go layer or build arrays incrementally (both break AILANG's value proposition)

## Root Cause Analysis

The hang is in `ValidateEffects` (`internal/pipeline/validate_effects.go`), specifically due to **quadratic traversal** in the interaction between `validateDecl` and `collectRequiredEffects`.

### The Bug

When processing nested `Let` expressions (how top-level bindings are desugared), we have redundant traversals:

```go
// validateDecl (line 60-78)
if let, ok := decl.(*core.Let); ok {
    // 1. Collect effects from RHS
    required := collectRequiredEffects(let.Value, typeInfo)
    // ... check subsumption ...

    // 2. Recurse on body (which contains remaining Lets)
    return validateDecl(let.Body, declaredEffects, typeInfo)
}

// collectRequiredEffects (line 183-188)
case *core.Let:
    // Traverse RHS AND body
    rhsEffects := collectRequiredEffects(e.Value, typeInfo)
    bodyEffects := collectRequiredEffects(e.Body, typeInfo)  // ← REDUNDANT!
    return types.UnionEffectRows(rhsEffects, bodyEffects)
```

### Complexity Analysis

For a module with `m` top-level Let bindings:

1. `validateDecl(Let1)` calls:
   - `collectRequiredEffects(value1)` → traverses array1
   - `collectRequiredEffects(body1)` → traverses Let2, Let3, ..., Letm (including all arrays!)
   - `validateDecl(body1)` → recurses on Let2

2. `validateDecl(Let2)` calls:
   - `collectRequiredEffects(value2)` → traverses array2 **AGAIN**
   - `collectRequiredEffects(body2)` → traverses Let3, ..., Letm **AGAIN**
   - `validateDecl(body2)` → recurses on Let3

**Result**: Each array is traversed multiple times:
- Array in Letk is traversed by validateDecl(Let1), validateDecl(Let2), ..., validateDecl(Letk)
- Total work: O(m²) where m = number of Let bindings

With 10+ bindings and 192-element arrays, this can mean 10,000+ redundant literal traversals.

### Why It Hangs

The O(m²) behavior combines with:
1. Each `collectRequiredEffects` call does type lookups via `typeInfo.Get(e.ID())`
2. Each literal element creates allocations in `UnionEffectRows`
3. GC pressure from repeated map/slice allocations

## Goals

**Primary Goal:** Effect checking completes in O(n) time where n = total AST size.

**Success Metrics:**
- `ailang check` on 192-element array modules completes in <5 seconds
- No observable slowdown for small files (<100ms overhead)
- Effect semantics unchanged (same pass/fail behavior)

## Solution Design

### Overview

**Fix the quadratic traversal by either:**

**Option A: Don't traverse Let bodies in collectRequiredEffects** (Recommended)
- `validateDecl` already recursively handles Let bodies
- `collectRequiredEffects` should only collect effects from the immediate value, not the body

**Option B: Cache effect results per NodeID**
- Memoize `collectRequiredEffects` results
- Higher memory overhead but handles all cases

### Option A: Fix Let Traversal (Recommended)

**Change in `collectRequiredEffects`:**

```go
case *core.Let:
    // BEFORE (quadratic):
    rhsEffects := collectRequiredEffects(e.Value, typeInfo)
    bodyEffects := collectRequiredEffects(e.Body, typeInfo)
    return types.UnionEffectRows(rhsEffects, bodyEffects)

    // AFTER (linear):
    // Only collect from RHS - validateDecl handles body recursion
    return collectRequiredEffects(e.Value, typeInfo)
```

**Why this is correct:**
- `validateDecl` already calls `collectRequiredEffects` on each Let's value
- `validateDecl` then recurses on the body
- Each binding's effects are checked exactly once
- The body's effects are validated when `validateDecl` processes them

**Same fix needed for LetRec:**
```go
case *core.LetRec:
    // Only collect from bindings, not body
    var effects *types.Row
    for _, binding := range e.Bindings {
        bindingEffects := collectRequiredEffects(binding.Value, typeInfo)
        effects = types.UnionEffectRows(effects, bindingEffects)
    }
    return effects  // Don't traverse body here
```

### Implementation Plan

**Phase 1: Fix Quadratic Traversal** (~2 hours)
- [ ] Modify `collectRequiredEffects` for `*core.Let` case
- [ ] Modify `collectRequiredEffects` for `*core.LetRec` case
- [ ] Add regression test with large array literals
- [ ] Verify existing effect checking tests still pass

**Phase 2: Add Performance Tests** (~2 hours)
- [ ] Add benchmark: `BenchmarkEffectCheck_LargeArrays`
- [ ] Add benchmark: `BenchmarkEffectCheck_DeepNesting`
- [ ] Add CI gate: effect checking must complete in <10s for test corpus

**Phase 3: Optimization (if needed)** (~2 hours)
- [ ] Profile with `pprof` to identify remaining hotspots
- [ ] Consider caching type lookups if still slow
- [ ] Consider pooling Row allocations if GC is bottleneck

### Files to Modify

**Modified files:**
- `internal/pipeline/validate_effects.go` - Fix Let/LetRec traversal (~10 LOC change)
- `internal/pipeline/validate_effects_test.go` - Add performance tests (~100 LOC)

**No new files needed.**

## Examples

### Example 1: Effect Checking Large Arrays

**Before (hangs):**
```bash
$ time ailang check sim/bridge.ail
→ Type checking sim/bridge.ail...
→ Effect checking...
^C  # Killed after 30+ seconds
```

**After (fast):**
```bash
$ time ailang check sim/bridge.ail
→ Type checking sim/bridge.ail...
→ Effect checking...
✓ No errors found!

real    0m1.234s  # <2 seconds total
```

### Example 2: Complexity Comparison

**Input:** Module with 10 Let bindings, each with a 100-element array.

| Metric | Before | After |
|--------|--------|-------|
| Let traversals | 55 (1+2+...+10) | 10 |
| Total nodes visited | ~55,000 | ~1,000 |
| Time | 30+ seconds | <1 second |

## Success Criteria

- [x] `ailang check` on 192-element array module completes in <5 seconds (172µs!)
- [x] `BenchmarkEffectCheck_LargeArrays` shows O(n) scaling (ratio 6.4 for 10x input)
- [x] All existing effect checking tests pass
- [ ] Manual test with stapledons_voyage sim/bridge.ail works (pending user verification)
- [x] No performance regression on small files

## Testing Strategy

**Unit tests:**
- Test effect checking on modules with nested Let bindings
- Test effect checking on modules with large array literals
- Test effect checking on modules with deep nesting (20+ Let levels)

**Benchmarks:**
```go
func BenchmarkEffectCheck_LargeArrays(b *testing.B) {
    // Generate module with N-element arrays
    for _, n := range []int{100, 500, 1000} {
        b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
            src := generateLargeArrayModule(n)
            for i := 0; i < b.N; i++ {
                pipeline.Run(cfg, src)
            }
        })
    }
}
```

**Manual testing:**
- Test with actual stapledons_voyage sim/bridge.ail file
- Verify type errors and effect errors still reported correctly

## Non-Goals

**Not in this feature:**
- General compiler performance optimization - only effect checking
- Memory usage optimization - focus on time complexity first
- Parallelization - not needed for O(n) algorithm

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Fix breaks effect semantics | High | Comprehensive test suite + manual verification |
| Other quadratic patterns exist | Medium | Profile after fix, address incrementally |
| Fix insufficient for 1000+ element arrays | Low | Option B (caching) as fallback |

## References

- Original bug report: Agent message `6d5bfdda-6461-4acb-88aa-ff24a20e224e`
- Effect checking code: `internal/pipeline/validate_effects.go`
- Effect types: `internal/types/effects.go`

## Future Work

- Profile full compilation pipeline for other O(n²) patterns
- Consider lazy effect collection (only compute when needed)
- Add `--no-effect-check` flag for development iteration speed

---

**Document created**: 2025-12-08
**Last updated**: 2025-12-08
