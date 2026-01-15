# Sprint Plan: M-GAP6 - Add maximum/minimum to std/list

## Summary
Add `maximumInt`, `minimumInt`, `maximumFloat`, `minimumFloat`, `maximumString`, and `minimumString` functions to `std/list` with safe `Option` return type for empty list handling.

**Duration:** 0.5 days (3 hours)
**Dependencies:** None (std/option already exports Option type)
**Risk Level:** Low
**Priority:** P2

## Current Status Analysis

### Completed Recently (from CHANGELOG)
- v0.6.3: OpenAI Responses API, OTEL tracing enhancements (~200 LOC/day average)
- v0.6.2: OpenTelemetry integration, coordinator enhancements
- std/list already has: `sortBy`, `nth`, `last`, `any`, `findIndex` (v0.6.5)

### Velocity
- Recent average: 150-200 LOC/day for stdlib additions
- Estimated capacity: ~50 LOC for this sprint (straightforward stdlib addition)

### Existing Infrastructure
- `std/list.ail`: 174 lines with existing functions (length, head, tail, map, filter, foldl, etc.)
- `std/option.ail`: 48 lines with `Option[a] = Some(a) | None` type already exported
- Pattern: Functions follow `export pure func name[a](...)` convention
- Import: `import std/option (Option, Some, None)` already in std/list

## Proposed Milestones

### Milestone 1: Implement Int/Float/String Maximum/Minimum Functions
**Goal:** Add 6 monomorphic functions to std/list for finding extreme values with Option return type
**Estimated:** 40 LOC implementation + 15 LOC exports + 60 LOC test file = ~115 LOC total
**Duration:** 3 hours

**Tasks:**
- Hour 1: Implement `maximumInt`, `minimumInt` functions in std/list.ail
- Hour 2: Implement `maximumFloat`, `minimumFloat`, `maximumString`, `minimumString` functions
- Hour 2.5: Create test example file `examples/docs/list_extremes.ail`
- Hour 3: Run tests, verify all functions work, update exports

**Implementation Pattern (from design doc):**
```ailang
export pure func maximumInt(xs: [int]) -> Option[int] {
  match xs {
    [] => None,
    [x] => Some(x),
    [x, ...rest] => match maximumInt(rest) {
      None => Some(x),
      Some(m) => if x > m then Some(x) else Some(m)
    }
  }
}
```

**Files to Modify:**
| File | Change | LOC |
|------|--------|-----|
| `std/list.ail` | Add maximumInt, minimumInt | ~20 |
| `std/list.ail` | Add maximumFloat, minimumFloat | ~20 |
| `std/list.ail` | Add maximumString, minimumString | ~20 |
| `examples/docs/list_extremes.ail` | Create test/example file | ~55 |

**Acceptance Criteria:**
- [ ] `maximumInt([3, 1, 4, 1, 5])` returns `Some(5)`
- [ ] `minimumInt([3, 1, 4, 1, 5])` returns `Some(1)`
- [ ] `maximumInt([])` returns `None`
- [ ] `minimumInt([])` returns `None`
- [ ] `maximumFloat([1.5, 2.5, 0.5])` returns `Some(2.5)`
- [ ] `minimumFloat([1.5, 2.5, 0.5])` returns `Some(0.5)`
- [ ] `maximumString(["apple", "cherry", "banana"])` returns `Some("cherry")`
- [ ] `minimumString(["apple", "cherry", "banana"])` returns `Some("apple")`
- [ ] Single element lists work: `maximumInt([42])` = `Some(42)`
- [ ] Negative numbers work: `maximumInt([-5, -2, -8])` = `Some(-2)`
- [ ] All 6 functions exported from std/list
- [ ] Example file runs without errors
- [ ] All existing std/list tests continue to pass
- [ ] `make lint` passes

**Risks:**
- Recursive implementation could hit stack limits on very large lists (not a concern for typical use)
- Mitigation: Document that extremely large lists (10000+) may need iterative approach

## Success Metrics
- Test coverage: All 6 functions have test cases in example file
- Examples passing: New `list_extremes.ail` runs successfully
- Documentation: Design doc updated to "Implemented" status
- All tests passing
- All linting passing

## Dependencies
- None - std/option already provides Option type
- std/list already imports Option, Some, None

## Open Questions
- None - design doc is complete and clear

## Notes
- This is a straightforward stdlib addition following existing patterns in std/list
- Monomorphic approach chosen because AILANG's type classes are hardcoded (Ord for int, float, string)
- Polymorphic version (`maximum[a: Ord]`) deferred to v0.7.0+ when type classes are fully supported
- Functions return Option to handle empty list safely (unlike Haskell's partial `maximum`)
