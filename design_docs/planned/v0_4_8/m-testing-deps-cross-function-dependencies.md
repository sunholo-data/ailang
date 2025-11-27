# M-TESTING-DEPS: Cross-Function Dependency Support for Inline Tests

**Status**: Planned
**Target**: v0.4.8
**Priority**: P1 (Medium)
**Estimated**: 2 days
**Dependencies**: M-TESTING-INLINE-CORE (v0.4.5) - COMPLETE

## AI-First Alignment Check

**Score this feature against AILANG's core principles:**

| Principle | Impact | Score | Notes |
|-----------|--------|-------|-------|
| Reduce Syntactic Noise | + | +1 | Enables testing more functions without workarounds |
| Preserve Semantic Clarity | 0 | 0 | Test syntax unchanged, dependency resolution automatic |
| Increase Determinism | + | +1 | Deterministic dependency analysis at compile time |
| Lower Token Cost | + | +1 | AI can test lcm without manually setting up gcd dependency |
| **Net Score** | | **+3** | **Decision: Move forward** |

**Decision rule:** Net score > +1 → Move forward | ≤ 0 → Reject or redesign

**Reference:** See [AI-first DX philosophy](../v0_3_15/example-parity-vision-alignment.md#-design-principle-ai-first-dx)

## Problem Statement

Inline tests currently only work for **self-contained pure functions**. Functions that call other user-defined functions fail with "undefined variable" errors because the test harness only extracts the single function being tested.

**Current State:**
- Test harness extracts only the function under test
- Functions with dependencies on other user-defined functions cannot be tested
- 22 functions testable in M-TESTING-INLINE-CORE, but functions like `lcm` (calls `gcd`) are excluded
- Workaround requires manual testing via `main` function

**Impact:**
- AI code generators cannot test functions with dependencies inline
- Reduces test coverage completeness
- Forces awkward workarounds for common patterns (helper functions, composition)

### Error Example

```typescript
-- This works: gcd is self-contained
export pure func gcd(a: int, b: int) -> int
  tests [((12, 8), 4), ((54, 24), 6)]
{
  if b == 0 then a else gcd(b, a % b)
}

-- This FAILS: lcm calls gcd
export pure func lcm(a: int, b: int) -> int
  tests [((4, 6), 12)]  -- Error: undefined variable: gcd
{
  (a * b) / gcd(a, b)
}
```

## Goals

**Primary Goal:** Enable inline tests for functions that depend on other user-defined functions within the same module.

**Success Metrics:**
- lcm tests pass when gcd is in same module
- Chain dependencies (3+ levels) work correctly
- Mutually recursive functions (isEven/isOdd) can be tested
- No performance regression (dependency analysis is O(n))
- Clear error messages for external dependencies (different modules)

## Solution Design

### Target UX (Mental Model)

For the AI/user mental model, the end-state should be:

> "Inline tests can call any pure function in the same module, including helpers and mutually recursive groups. If it compiles and is pure, you can test it inline."

### Overview

Build a call-graph over Core AST, compute **Strongly Connected Components (SCCs)** for mutual recursion detection, then synthesize a "cluster harness" containing the function under test plus its entire pure dependency closure.

### Architecture

**Components:**
1. **Call Graph Builder** (`internal/testing/callgraph.go`): Build dependency graph over Core bindings
2. **SCC Computer**: Tarjan's algorithm to find mutually recursive groups
3. **Pure Cluster Extractor**: Collect SCC + transitive pure dependencies
4. **Cluster Harness Builder**: LetRec over entire cluster with test body

### Key Concepts

**Call Graph:**
- Nodes: top-level function/value bindings in a module
- Edges: `f → g` if the Core body of `f` references `g`
- Built on elaborated Core AST (all sugar resolved, actual `Var` references visible)

**Strongly Connected Components (SCCs):**
- SCCs naturally capture mutual recursion: `{isEven, isOdd}` form one SCC
- A single self-recursive function is its own SCC
- Non-recursive functions are singleton SCCs

**Pure Cluster:**
- For function `f` under test, the pure cluster is:
  1. The SCC containing `f`
  2. All SCCs reachable from that SCC via the call graph
  3. **Gated on purity**: every function in the cluster must have effect row `{}` (pure)
- If any dependency has effects, the cluster is not pure → test rejected with clear error

**Cluster Harness:**
Instead of the current single-binding harness:
```
LetRec(f, λ_f, test_body(f))
```

We synthesize a multi-binding harness:
```
LetRec(
  [ f  ↦ λ_f,
    g1 ↦ λ_g1,
    g2 ↦ λ_g2,
    ...
  ],
  test_body(f)
)
```

Where `test_body(f)` is the existing nested-Let harness (calls to `f` with test inputs, returning results).

### Implementation Plan

**Phase 1: Call Graph & SCC** (~6 hours)
- [ ] Create `internal/testing/callgraph.go`
- [ ] Implement `BuildCallGraph(module) -> Graph[string, string]`
- [ ] Walk Core AST, collect `Var` references per binding
- [ ] Implement Tarjan's SCC algorithm (or use existing graph lib)
- [ ] Unit tests: single function, chain, mutual recursion

**Phase 2: Pure Cluster Extraction** (~4 hours)
- [ ] Implement `ExtractPureCluster(funcName, callGraph, sccs, effectInfo) -> []Binding`
- [ ] Start from function's SCC
- [ ] BFS/DFS to collect reachable bindings
- [ ] Gate on effect row `{}` for each binding
- [ ] Return error if cluster contains effectful function
- [ ] Unit tests: pure cluster, effectful rejection

**Phase 3: Cluster Harness Builder** (~4 hours)
- [ ] Modify `BuildTestHarness` to accept `[]core.RecBinding` cluster
- [ ] Synthesize single `LetRec` with all bindings
- [ ] Test body references function under test
- [ ] Integration tests with lcm/gcd, isEven/isOdd

**Phase 4: Integration & Polish** (~4 hours)
- [ ] Wire into `TestExecutor`
- [ ] Add `--dump-inline-harness` flag for debugging (optional)
- [ ] Enable lcm, isEven/isOdd tests in examples
- [ ] Update documentation

### Files to Modify/Create

**New files:**
- `internal/testing/callgraph.go` - Call graph + SCC (~200 LOC)
- `internal/testing/callgraph_test.go` - Unit tests (~250 LOC)
- `internal/testing/cluster.go` - Pure cluster extraction (~100 LOC)

**Modified files:**
- `internal/testing/harness.go` - Cluster harness builder (~50 LOC changes)
- `internal/testing/executor.go` - Integrate call graph (~30 LOC changes)
- `examples/snippets/v3_3/math/gcd.ail` - Enable lcm tests (~5 LOC)

### Edge Cases

**Mutual recursion across module "sections":**
```typescript
func isEven(n) = if n == 0 then true else isOdd(n - 1)
func isOdd(n)  = if n == 0 then false else isEven(n - 1)
```
Both end up in same SCC; both go into the LetRec binding vector; tests on either work.

**Polymorphic helpers:**
Locally defined polymorphic helpers (e.g., `map`) are just another binding. As long as effect row is pure, they join the cluster.

**Kitchen-sink modules:**
Large modules where "everything depends on everything" result in big clusters. Fine for v1 - harness includes more bindings but still works. Optimize later if perf becomes issue.

**Dependency on effectful function:**
```typescript
pure func foo() = bar()   -- bar has effects
```
Cluster extraction fails with clear error: "Cannot test foo: dependency bar has effect row {IO}"

## Examples

### Example 1: Simple Dependency (lcm → gcd)

**Before:**
```typescript
export pure func lcm(a: int, b: int) -> int
  -- Tests not supported: cross-function dependency
{
  (a * b) / gcd(a, b)
}
```

**After:**
```typescript
export pure func lcm(a: int, b: int) -> int
  tests [((4, 6), 12), ((3, 5), 15), ((12, 8), 24)]
{
  (a * b) / gcd(a, b)
}
-- Tests pass because gcd is automatically included in harness
```

### Example 2: Chain Dependency (octuple → quadruple → double)

```typescript
pure func double(x: int) -> int { x * 2 }
pure func quadruple(x: int) -> int { double(double(x)) }
pure func octuple(x: int) -> int
  tests [(1, 8), (2, 16), (10, 80)]
{
  double(quadruple(x))
}
-- Harness includes: double, quadruple, octuple
```

### Example 3: Mutual Recursion (isEven ↔ isOdd)

```typescript
pure func isEven(n: int) -> bool
  tests [(0, true), (1, false), (4, true), (7, false)]
{ if n == 0 then true else isOdd(n - 1) }

pure func isOdd(n: int) -> bool
  tests [(0, false), (1, true), (5, true), (6, false)]
{ if n == 0 then false else isEven(n - 1) }
-- Both functions included in single LetRec binding group
```

## Success Criteria

- [ ] lcm tests pass when gcd is in same module
- [ ] Chain dependencies (3+ levels) work correctly
- [ ] Mutually recursive functions can be tested
- [ ] Dependency analysis runs in O(n) time
- [ ] Clear error messages for external dependencies
- [ ] All tests passing
- [ ] Documentation updated
- [ ] Examples added

## Testing Strategy

**Unit tests:**
- `FindDependencies` returns correct direct dependencies
- `TransitiveDependencies` handles cycles correctly
- `FindMutuallyRecursive` detects mutual recursion

**Integration tests:**
- lcm/gcd example passes
- Chain dependency example passes
- isEven/isOdd example passes
- Error case: external module dependency

**Manual testing:**
- Run `ailang test examples/snippets/v3_3/math/gcd.ail`
- Verify all tests pass

## Non-Goals

**Not in this feature:**
- Cross-module dependencies - Requires module loader integration (future M-TESTING-IMPORTS)
- Effect dependencies - Functions calling effectful functions (see M-TESTING-EFFECTS)
- Dynamic dependencies - Functions stored in variables/data structures

## Timeline

**Day 1** (8 hours):
- Phase 1: Call Graph & SCC (6 hours)
- Phase 2: Pure Cluster Extraction (start)

**Day 2** (8 hours):
- Phase 2: Pure Cluster Extraction (complete, 2 hours)
- Phase 3: Cluster Harness Builder (4 hours)
- Phase 4: Integration & Polish (2 hours)

**Day 3** (4 hours - buffer):
- Documentation
- Additional edge case testing
- `--dump-inline-harness` debug flag

**Total: ~18-20 hours across 2-3 days**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Circular dependencies cause infinite loops | High | Implement cycle detection with visited set |
| Performance regression for large modules | Medium | O(n) algorithm, benchmark before/after |
| Complex mutual recursion patterns | Low | Start with simple pairs, extend if needed |

## References

- [M-TESTING-INLINE-CORE design doc](../v0_4_7/m-testing-inline-core-evaluation.md)
- [Migration summary](../../../M-TESTING-INLINE-MIGRATION-SUMMARY.md)
- [Core AST walker](../../../internal/core/walk.go)

## Future Work

**Next milestone after DEPS:**
- **M-TESTING-PROPERTY**: Property-based / inline generators on top of cluster harness
  - Same cluster logic applies
  - Different harness body (quantified cases instead of enumerated)
  - Executor drives property harness N times with random seeds
  - DEPS pays dividends here - getting cluster harness right enables property tests

**Later:**
- **M-TESTING-IMPORTS**: Cross-module dependency support
- **Dependency visualization**: `ailang test --show-deps` to display dependency graph
- **Selective dependency inclusion**: Allow tests to exclude certain dependencies

## Open Questions

1. **Harness debugging**: Should we add `--dump-inline-harness` flag to emit synthetic Core for inspection?
2. **Property tests location**: Same `tests` block, separate `properties` keyword, or separate file?

---

**Document created**: 2025-11-27
**Last updated**: 2025-11-27
