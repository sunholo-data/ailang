# M-PERF3: Performance Quick Wins for 1.0.0

**Status**: Planned
**Target**: v0.7.0+ (pre-1.0.0)
**Priority**: P2 (Low - planning ahead)
**Estimated**: 2-3 days total (spread across releases)
**Dependencies**: None (independent optimizations)

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | Pure performance optimizations, no semantic changes |
| A2: Replayability | 0 | Traces unchanged |
| A3: Effect Legibility | 0 | No effect system changes |
| A4: Explicit Authority | 0 | No capability changes |
| A5: Bounded Verification | 0 | Type checking unchanged |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | Faster compilation/execution = better AI tooling |
| A8: Minimal Syntax | 0 | No syntax changes |
| A9: Cost Visibility | +1 | Optimizations make costs more predictable |
| A10: Composability | 0 | No semantic changes |
| A11: Structured Failure | 0 | Error handling unchanged |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +2** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No semantic changes, just faster execution
- [x] A3 (Effects): No effect system modifications
- [x] A4 (Authority): No capability changes
- [x] A7 (Machines First): Actively improves machine processing speed

## Problem Statement

**Benchmark Discovery (December 2025):**

Cross-language benchmarking revealed significant performance gaps:

| Implementation | fib(25) Time | vs Native Go |
|----------------|--------------|--------------|
| Native Go | 3ms | baseline |
| Python | 48ms | 16x slower |
| **AILANG interpreted** | **260ms** | **87x slower** |
| AILANG -> Go codegen | ~12ms* | 4x slower |

*Estimated from fib(30) data

**Current State:**
- AILANG interpreter is **~5x slower than Python** for recursive workloads
- High per-call overhead in evaluator (environment cloning, lookup traversal)
- Go codegen uses `interface{}` everywhere, losing type information
- `exprProducesInterface()` called 100,000+ times during codegen without memoization
- Type substitution allocates `visited` maps on every call

**Impact:**
- Large AILANG programs compile/run noticeably slower than expected
- Recursive algorithms hit interpreter overhead hard
- Codegen produces correct but suboptimal Go code

## Goals

**Primary Goal:** Achieve 3-10x speedup for large AILANG programs through targeted optimizations

**Success Metrics:**
- Interpreter: fib(25) from 260ms to <100ms (2.5x+ improvement)
- Codegen compilation: 2x faster for 1000+ line modules
- Generated Go: Reduce interface{} boxing overhead by 50%

## Solution Design

### Overview

Three independent optimization tracks that can be implemented incrementally:

1. **Evaluator Optimizations** - Reduce per-call overhead
2. **Codegen Optimizations** - Memoize hot path calculations
3. **Type System Optimizations** - Pool allocations, cache lookups

### Architecture

**Track 1: Evaluator (internal/eval/)**

| Optimization | Current | Proposed | Impact |
|--------------|---------|----------|--------|
| Environment cloning | Clone on every call | Copy-on-write or lazy flattening | 3-5x for deep closures |
| GetAllBindings() | Recursive traversal | Iterative + caching | 2-3x for function calls |
| Value boxing | All values boxed | Type-specialized fast paths | 1.5-2x for arithmetic |

**Track 2: Codegen (internal/gen/golang/)**

| Optimization | Current | Proposed | Impact |
|--------------|---------|----------|--------|
| exprProducesInterface() | Called per expression | Memoize per codegen phase | 2-3x for large modules |
| CoreTypeInfo lookups | Map lookup per node | Cache locally | 1.5x compilation |
| interface{} elimination | All values interface{} | Type-directed specialization | 2-5x generated code |

**Track 3: Type System (internal/types/)**

| Optimization | Current | Proposed | Impact |
|--------------|---------|----------|--------|
| Substitution visited maps | New map per call | sync.Pool | 10-20% for type checking |
| Type string comparison | Call String() | Direct structural comparison | Already fixed (v0.5.7) |

### Implementation Plan

**Phase 1: Low-Hanging Fruit** (~4 hours)
- [ ] Memoize `exprProducesInterface()` results per codegen phase
- [ ] Add local cache for CoreTypeInfo lookups
- [ ] Pool `visited` maps in type substitution

**Phase 2: Evaluator Improvements** (~8 hours)
- [ ] Replace recursive `GetAllBindings()` with iterative traversal
- [ ] Add binding cache at closure creation time
- [ ] Consider copy-on-write environment strategy

**Phase 3: Codegen Type Specialization** (~12 hours)
- [ ] Track concrete types through codegen
- [ ] Generate type-specialized code for hot paths
- [ ] Reduce interface{} boxing for primitives

### Files to Modify/Create

**Modified files:**
- `internal/gen/golang/codegen_type_analysis.go` - Add memoization (~50 LOC)
- `internal/gen/golang/codegen.go` - Add phase-local caches (~30 LOC)
- `internal/eval/env.go` - Iterative GetAllBindings, caching (~80 LOC)
- `internal/types/unification_substitution.go` - Pool visited maps (~20 LOC)

**New files:**
- `internal/gen/golang/codegen_cache.go` - Shared caching utilities (~100 LOC)

## Examples

### Example 1: exprProducesInterface Memoization

**Before:**
```go
func (g *CodeGen) exprProducesInterface(expr core.Expr) bool {
    // Called on every expression, every time
    if g.coreTypeInfo != nil {
        nodeID := g.getExprNodeID(expr)
        if typ, ok := g.coreTypeInfo[nodeID]; ok {
            // ... expensive computation
        }
    }
    // ... more logic
}
```

**After:**
```go
func (g *CodeGen) exprProducesInterface(expr core.Expr) bool {
    nodeID := g.getExprNodeID(expr)
    if result, cached := g.interfaceCache[nodeID]; cached {
        return result
    }
    // ... compute result
    g.interfaceCache[nodeID] = result
    return result
}
```

### Example 2: Iterative GetAllBindings

**Before:**
```go
func (e *Environment) GetAllBindings() map[string]Value {
    result := make(map[string]Value)
    if e.parent != nil {
        for k, v := range e.parent.GetAllBindings() { // RECURSIVE
            result[k] = v
        }
    }
    for k, v := range e.values {
        result[k] = v
    }
    return result
}
```

**After:**
```go
func (e *Environment) GetAllBindings() map[string]Value {
    // Collect envs iteratively
    envs := make([]*Environment, 0, 8)
    for env := e; env != nil; env = env.parent {
        envs = append(envs, env)
    }
    // Merge from root to leaf (child shadows parent)
    result := make(map[string]Value, len(envs)*4)
    for i := len(envs) - 1; i >= 0; i-- {
        for k, v := range envs[i].values {
            result[k] = v
        }
    }
    return result
}
```

## Success Criteria

- [ ] fib(25) benchmark: AILANG interpreted < 150ms (from 260ms)
- [ ] Codegen compilation time reduced 40%+ for 1000+ line modules
- [ ] No semantic changes (all existing tests pass)
- [ ] Benchmark script updated with new baselines
- [ ] perf-reviewer skill updated with findings

## Testing Strategy

**Unit tests:**
- Environment caching correctness
- Memoization cache invalidation
- Pool reuse safety

**Integration tests:**
- Run full benchmark suite before/after
- Verify identical output (semantics unchanged)

**Manual testing:**
- Profile with pprof before/after
- Verify no regressions on real projects

## Non-Goals

**Not in this feature:**
- JIT compilation - Requires major architecture change
- Native code generation - Out of scope (Go codegen sufficient)
- Parallel evaluation - Would change semantics (A1 violation)
- Garbage collector tuning - Go runtime concern

## Timeline

This is a low-priority feature for 1.0.0 planning. Implementation can be spread across multiple releases:

**v0.7.0**: Phase 1 (codegen memoization) - 4 hours
**v0.8.0**: Phase 2 (evaluator improvements) - 8 hours
**v0.9.0**: Phase 3 (type specialization) - 12 hours

**Total: ~24 hours across 3 releases**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Cache invalidation bugs | Medium | Comprehensive tests, conservative caching |
| Memory overhead from caches | Low | Bounded cache sizes, clear after phase |
| Subtle semantic changes | High | Run all existing tests, add property tests |
| Premature optimization | Low | Profile-driven, focus on proven hot paths |

## Related Documents

**Implemented (informs design):**
- [M-PERF1: Effect checker quadratic traversal](../../../design_docs/implemented/) - Fixed in v0.5.8
- [M-PERF2: Operator lowering cycles](../../../design_docs/implemented/) - Cycle detection added

**Planned (check for overlap):**
- [m-poly-arithmetic-fix](m-poly-arithmetic-fix.md) - May interact with codegen

## References

- [Design Axioms](/docs/references/axioms)
- [Abseil Performance Tips](https://abseil.io/fast/hints.html) - Inspiration for principles
- [perf-reviewer skill](.claude/skills/perf-reviewer/) - Benchmark tooling
- [resources/principles.md](.claude/skills/perf-reviewer/resources/principles.md) - Performance guide

## Future Work

- Investigate bytecode interpreter (major rewrite, v2.0+)
- LLVM backend for native performance (v2.0+)
- Profile-guided optimization for generated Go

---

**Document created**: 2025-12-29
**Last updated**: 2025-12-29
