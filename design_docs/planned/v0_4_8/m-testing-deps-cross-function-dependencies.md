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

### Overview

Add static dependency analysis that walks function bodies to identify referenced user-defined functions, then include those dependencies in the test harness.

### Architecture

**Components:**
1. **Dependency Analyzer** (`internal/testing/deps.go`): Walks Core AST to find function references
2. **Transitive Closure**: Computes all dependencies recursively
3. **Enhanced Harness Builder**: Includes dependencies in synthetic test program
4. **Mutual Recursion Detector**: Groups mutually recursive functions together

### Implementation Plan

**Phase 1: Dependency Analyzer** (~4 hours)
- [ ] Create `internal/testing/deps.go`
- [ ] Implement `FindDependencies` using Core AST walker
- [ ] Implement `TransitiveDependencies` with cycle detection
- [ ] Add unit tests for dependency analysis

**Phase 2: Enhanced Harness** (~4 hours)
- [ ] Modify `BuildTestHarness` signature to accept dependencies
- [ ] Update harness building to include all dependencies
- [ ] Handle ordering (dependencies before dependents)
- [ ] Add tests for multi-function harness

**Phase 3: Mutual Recursion** (~4 hours)
- [ ] Implement `FindMutuallyRecursive` detection
- [ ] Group mutually recursive functions in single LetRec
- [ ] Add tests for isEven/isOdd pattern

**Phase 4: Integration & Testing** (~4 hours)
- [ ] Update `TestExecutor` to use dependency analyzer
- [ ] Enable `lcm` tests in examples/snippets/v3_3/math/gcd.ail
- [ ] Add comprehensive test suite
- [ ] Update documentation

### Files to Modify/Create

**New files:**
- `internal/testing/deps.go` - Dependency analyzer (~150 LOC)
- `internal/testing/deps_test.go` - Unit tests (~200 LOC)

**Modified files:**
- `internal/testing/harness.go` - Accept dependencies (~30 LOC changes)
- `internal/testing/executor.go` - Integrate analyzer (~50 LOC changes)
- `examples/snippets/v3_3/math/gcd.ail` - Enable lcm tests (~5 LOC)

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
- Phase 1: Dependency Analyzer
- Phase 2: Enhanced Harness

**Day 2** (8 hours):
- Phase 3: Mutual Recursion
- Phase 4: Integration & Testing
- Documentation

**Total: ~16 hours across 2 days**

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

- **M-TESTING-IMPORTS**: Cross-module dependency support
- **Dependency visualization**: `ailang test --show-deps` to display dependency graph
- **Selective dependency inclusion**: Allow tests to exclude certain dependencies

---

**Document created**: 2025-11-27
**Last updated**: 2025-11-27
