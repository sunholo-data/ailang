# M-TESTING-ADT: ADT Constructors as Test Expected Values

**Status**: Planned
**Target**: v0.5.0
**Priority**: P1 (Medium)
**Estimated**: 4-6 hours
**Dependencies**: M2 (ADT Test Harness Scope) - completed in v0.4.10

## AI-First Alignment Check

**Score this feature against AILANG's core principles:**

| Principle | Impact | Score | Notes |
|-----------|--------|-------|-------|
| Reduce Syntactic Noise | + | +1 | Eliminates workaround of encoding ADTs as ints for test expected values |
| Preserve Semantic Clarity | + | +1 | Tests express intent directly: `(Alive, 2, Alive)` not `(1, 2, 1)` |
| Increase Determinism | 0 | 0 | No change - evaluation remains deterministic |
| Lower Token Cost | + | +1 | Tests are more concise and self-documenting |
| **Net Score** | | **+3** | **Decision: Move forward** |

**Decision rule:** Net score > +1 → Move forward | ≤ 0 → Reject or redesign

**Reference:** See [AI-first DX philosophy](../v0_3_15/example-parity-vision-alignment.md#-design-principle-ai-first-dx)

## Problem Statement

The inline test harness (M-TESTING) partially supports ADT constructors:
- ✅ **Inputs work**: `tests [(North, 0), (South, 1)]` - constructors in input position
- ❌ **Expected values fail**: `tests [(Alive, 2, Alive)]` - constructors in expected position

**Current Error:**
```
test 0: failed to evaluate expected: expected literal expression, got *ast.Identifier
```

**Root Cause:**
The test harness evaluates expected values using `evalLiteralExpr()` in `internal/testing/executor.go`, which only handles primitive literals (int, float, string, bool). ADT constructor identifiers like `Alive`, `Dead`, `North` are `*ast.Identifier` nodes, not literal expressions.

**Current State:**
- M2 fix injects ADT constructors into evaluator environment for **inputs**
- Expected value evaluation path does NOT have access to constructor bindings
- Workaround: Encode ADTs as integers (`1` for Alive, `0` for Dead)

**Impact:**
- Tests are less readable and self-documenting
- Mismatch between test inputs (ADTs allowed) and expected values (ADTs forbidden)
- Users must maintain mental mapping between ints and ADT variants

## Goals

**Primary Goal:** Enable ADT constructors as expected values in inline tests, achieving symmetry with input handling.

**Success Metrics:**
- `tests [(Alive, true), (Dead, false)]` compiles and runs correctly
- `tests [((Alive, 0), Dead), ((Alive, 2), Alive)]` works for multi-arg functions
- Nullary constructors (North, South, Alive, Dead) work as expected values
- Constructor application `Some(5)` works as expected values (stretch goal)
- All existing tests continue to pass

## Solution Design

### Overview

Extend the expected value evaluation in `testing/executor.go` to:
1. Recognize ADT constructor identifiers
2. Look them up in the injected constructor environment
3. Evaluate them to `TaggedValue` for comparison

### Architecture

**Components:**
1. **evalExpectedExpr**: New/modified function to evaluate expected expressions (literals + constructors)
2. **Constructor lookup**: Reuse the same injection mechanism from M2
3. **Value comparison**: Existing `compareValues()` already handles `TaggedValue`

### Implementation Plan

**Phase 1: Extend Expected Expression Evaluation** (~2 hours)
- [ ] Add `evalExpectedExpr()` function that handles both literals and identifiers
- [ ] For identifiers, check if name exists in constructor bindings
- [ ] Return appropriate `eval.Value` (TaggedValue for constructors)
- [ ] Handle nullary constructors (no arguments)

**Phase 2: Handle Constructor Application** (~2 hours)
- [ ] Support `Some(5)` syntax in expected values
- [ ] Parse as `*ast.CallExpr` with constructor as function
- [ ] Evaluate arguments recursively
- [ ] Apply constructor to create TaggedValue with fields

**Phase 3: Testing and Documentation** (~2 hours)
- [ ] Add unit tests for expected value evaluation
- [ ] Create example file `examples/adt_test_expected.ail`
- [ ] Update existing Conway example to use ADT expected values
- [ ] Document in AILANG teaching prompt

### Files to Modify/Create

**Modified files:**
- `internal/testing/executor.go` - Add `evalExpectedExpr()`, modify test comparison (~80 LOC)

**New files:**
- `internal/testing/expected_eval.go` - Expected expression evaluator (~100 LOC) [optional, could inline]
- `examples/adt_test_expected.ail` - Example demonstrating feature (~30 LOC)

## Examples

### Example 1: Simple ADT Expected Values

**Before (workaround):**
```ailang
type Cell = Alive | Dead

pure func cellIsAlive(cell: Cell) -> int
  tests [(Alive, 1), (Dead, 0)]  -- Must use ints
{ match cell { Alive => 1, Dead => 0 } }
```

**After:**
```ailang
type Cell = Alive | Dead

pure func cellIsAlive(cell: Cell) -> bool
  tests [(Alive, true), (Dead, false)]  -- Direct bools
{ match cell { Alive => true, Dead => false } }

pure func nextState(current: Cell, neighbors: int) -> Cell
  tests [
    ((Alive, 2), Alive),   -- ADT constructor as expected value!
    ((Alive, 4), Dead),
    ((Dead, 3), Alive)
  ]
{ ... }
```

### Example 2: Constructor Application (Stretch Goal)

**After:**
```ailang
type Option[a] = Some(a) | None

pure func safeDiv(a: int, b: int) -> Option[int]
  tests [
    ((10, 2), Some(5)),    -- Constructor with argument
    ((10, 0), None)        -- Nullary constructor
  ]
{ if b == 0 then None else Some(a / b) }
```

## Success Criteria

- [ ] `tests [(Alive, true), (Dead, false)]` works with local ADTs
- [ ] `tests [(North, 0)]` works with imported ADT constructors
- [ ] `tests [((Alive, 2), Alive)]` works for multi-arg functions returning ADTs
- [ ] Nested ADTs work: `tests [(Some(5), true)]` with Option[int]
- [ ] All existing tests pass (no regression)
- [ ] Example file created and verified
- [ ] Conway example updated to use ADT expected values

## Testing Strategy

**Unit tests:**
- `TestEvalExpectedExpr_NullaryConstructor` - Evaluate `Alive`, `North`
- `TestEvalExpectedExpr_ConstructorWithArgs` - Evaluate `Some(5)`
- `TestEvalExpectedExpr_Literals` - Existing literal handling unchanged
- `TestEvalExpectedExpr_UnknownIdent` - Error for undefined identifiers

**Integration tests:**
- Full test run on `examples/adt_test_expected.ail`
- Existing test suite passes

**Manual testing:**
- Run Conway example with ADT expected values
- Verify error messages for malformed tests

## Non-Goals

**Not in this feature:**
- **Polymorphic constructor comparison**: `Some(x) == Some(y)` requires x == y; not implementing custom equality
- **Pattern matching in expected values**: `Some(_)` to match any Some - use explicit values
- **Effect-based expected values**: No IO or effectful computation in expected positions
- **Record expected values**: `{ x: 1, y: 2 }` - separate feature if needed

## Timeline

**Day 1** (4 hours):
- Phase 1: evalExpectedExpr for nullary constructors
- Phase 2: Constructor application handling
- Unit tests

**Day 2** (2 hours):
- Phase 3: Integration testing
- Example file creation
- Documentation updates

**Total: ~6 hours across 1-2 days**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Constructor not in scope | Medium | Clear error message: "Unknown constructor 'Foo' in test expected value" |
| Type mismatch at comparison | Low | Existing compareValues handles TaggedValue correctly |
| Performance of constructor lookup | Low | O(1) map lookup, already done for inputs |
| Nested constructor evaluation | Medium | Recursive evalExpectedExpr with depth limit |

## References

- [M2 ADT Test Harness Fix](../../implemented/v0_5_0/m-bug-fixes-sprint-plan.md) - Completed input-side handling
- [M-TESTING-INLINE Design Doc](../v0_3_15/m-testing-inline.md) - Original inline test design
- `internal/testing/executor.go` - Current implementation
- `internal/eval/value.go` - TaggedValue definition

## Future Work

- **Record literals as expected values**: `tests [(point, { x: 1, y: 2 })]`
- **Wildcard patterns**: `tests [(input, Some(_))]` to match any Some
- **Property-based testing integration**: Generate test cases automatically
- **Test coverage reporting**: Track which ADT variants are tested

---

**Document created**: 2025-12-01
**Last updated**: 2025-12-01
