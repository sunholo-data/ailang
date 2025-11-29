# M-BUG-RECORD-UPDATE-INFERENCE: Fix Record Update Type Inference

**Status**: Planned
**Target**: v0.4.9
**Priority**: P1 (Medium) - Affects immutable update patterns
**Estimated**: 1-2 days (~8-10 hours)
**Dependencies**: None
**Reported by**: stapledons_voyage (agent inbox)

## AI-First Alignment Check

| Principle | Impact | Score | Notes |
|-----------|--------|-------|-------|
| Reduce Syntactic Noise | + | +1 | Enables concise immutable updates |
| Preserve Semantic Clarity | 0 | 0 | Record update semantics unchanged |
| Increase Determinism | + | +1 | Type inference works consistently |
| Lower Token Cost | + | +1 | Avoids verbose explicit construction |
| **Net Score** | | **+3** | **Decision: Move forward** |

## Problem Statement

Record update syntax `{base | field: value}` fails with error: "record update requires base to be a record type, got *types.TVar2".

**Verified Reproduction (2025-11-29):**
```ailang
-- FAILS: Lambda parameter (even with type annotation)
type World = { tick: int }
let step: World -> World = \world. {world | tick: world.tick + 1}
-- ERROR: record update requires base to be a record type, got *types.TVar2

-- FAILS: Lambda without type annotation
let step = \world. {world | tick: world.tick + 1}
-- ERROR: record update requires base to be a record type, got *types.TVar2

-- WORKS: Let-bound record with inferred type
let person = { name: "Alice", age: 25 }
let updated = {person | age: 30}  -- ✓ Works
```

**Pattern:** Fails whenever the base expression's type flows through a type variable (lambda parameters), works when the type is fully known at binding time (let-bound records).

**Current Workaround:** Construct new record explicitly:
```ailang
let step: World -> World = \world. { tick: world.tick + 1 }
```

**Impact:**
- Verbose code for immutable updates (must list ALL fields)
- Extra maintenance burden when adding record fields
- Breaks common functional programming patterns

## Root Cause Analysis

**Location:** `internal/types/typechecker_data.go:99-122` (`inferRecordUpdate` function)

**The bug (verified by code reading):**
```go
// Line 101-106: Infer base type
baseNode, _, err := tc.inferCore(ctx, upd.Base)
baseType := getType(baseNode)

// Line 110-122: Switch on base type - FAILS for TVar2
switch t := baseType.(type) {
case *TRecord:   // ...
case *TRecord2:  // ...
case *TRecordOpen: // ...
default:
    // FAILS HERE: TVar2 is not a record type yet
    return nil, ctx.env, fmt.Errorf("record update requires base to be a record type, got %T", baseType)
}
```

**Why the original design doc's fix won't work:**
The proposed fix suggested applying substitution before the check:
```go
baseType = tc.substitute(baseType)  // ❌ Won't work
```

This doesn't work because:
1. During inference, constraints are **accumulated** but not yet **solved**
2. `SolveConstraints()` is called **after** inference completes
3. There is no partial substitution available during `inferRecordUpdate`
4. The type variable hasn't been unified with anything yet

**The real fix needs to be constraint-based** (like `inferRecordAccess` at lines 67-82).

## Goals

**Primary Goal:** Record update syntax works when base expression's type flows through type variables.

**Success Metrics:**
- `{world | field: val}` works for lambda parameters
- `{world | field: val}` works for let-bound records (already works)
- Clear error if base type is genuinely not a record
- Row polymorphism preserved (can update open records)

## Solution Design

### Overview

Refactor `inferRecordUpdate` to use **constraint-based type checking** instead of eager type inspection, following the pattern established in `inferRecordAccess`.

### Architecture

**The correct approach** (based on `inferRecordAccess` pattern at lines 67-82):

```go
func (tc *CoreTypeChecker) inferRecordUpdate(ctx *InferenceContext, upd *core.RecordUpdate) (typedast.TypedNode, *TypeEnv, error) {
    // 1. Infer base record type
    baseNode, _, err := tc.inferCore(ctx, upd.Base)
    if err != nil {
        return nil, ctx.env, err
    }
    baseType := getType(baseNode)

    // 2. Create fresh type variables for updated fields
    fieldTypes := make(map[string]Type)
    for fieldName := range upd.Updates {
        fieldTypes[fieldName] = ctx.freshTypeVar()
    }

    // 3. Create a row variable for other fields (row polymorphism)
    rowVar := &RowVar{Name: ctx.freshRowVarName(), Kind: RecordRow}

    // 4. Create constraint: base must be record with at least these fields
    expectedRecord := &TRecordOpen{
        Fields: fieldTypes,
        Row:    rowVar,  // Open for additional fields
    }
    ctx.addConstraint(TypeEq{
        Left:  baseType,
        Right: expectedRecord,
        Path:  []string{"record update base at " + upd.Span().String()},
    })

    // 5. Type check updated field values
    updatedFields := make(map[string]typedast.TypedNode)
    for fieldName, fieldValue := range upd.Updates {
        valueNode, _, err := tc.inferCore(ctx, fieldValue)
        if err != nil {
            return nil, ctx.env, err
        }

        // Unify value type with expected field type
        ctx.addConstraint(TypeEq{
            Left:  getType(valueNode),
            Right: fieldTypes[fieldName],
            Path:  []string{"record update field '" + fieldName + "'"},
        })

        updatedFields[fieldName] = valueNode
    }

    // 6. Result type is same as base (record update preserves structure)
    return &typedast.TypedRecord{
        TypedExpr: typedast.TypedExpr{
            NodeID:    upd.ID(),
            Span:      upd.Span(),
            Type:      baseType,  // Same type as base
            EffectRow: getEffectRow(baseNode),
            Core:      upd,
        },
        Fields: updatedFields,
    }, ctx.env, nil
}
```

**Key insight:** By using `ctx.addConstraint(TypeEq{...})` instead of checking the type immediately, we defer resolution to the constraint solver which runs AFTER inference completes.

### Implementation Plan

**Phase 1: Implement Constraint-Based Approach** (~4 hours)
- [ ] Modify `inferRecordUpdate` in `internal/types/typechecker_data.go`
- [ ] Replace eager type check with constraint generation
- [ ] Ensure row variable handling is correct
- [ ] Preserve existing behavior for concrete record types

**Phase 2: Testing** (~3 hours)
- [ ] Add test: lambda parameter with type annotation
- [ ] Add test: lambda parameter without type annotation
- [ ] Add test: let-bound record (regression test)
- [ ] Add test: multiple field updates
- [ ] Add test: error case (updating non-record)
- [ ] Add test: row polymorphic record

**Phase 3: Examples & Documentation** (~1 hour)
- [ ] Create `examples/record_update.ail`
- [ ] Update CHANGELOG.md
- [ ] Respond to stapledons_voyage

### Files to Modify

**Modified files:**
- `internal/types/typechecker_data.go:99-162` - Refactor `inferRecordUpdate` (~40 LOC net change)

**New files:**
- `examples/record_update.ail` - Working example (~20 LOC)
- `tests/record_update_inference_test.ail` - Test cases (~40 LOC)

**Total estimated LOC:** ~100 LOC (including tests)

## Examples

### Example 1: Basic Record Update (Reported Case)

**Current (Fails):**
```ailang
type World = { tick: int }

-- Lambda with type annotation - FAILS
let step: World -> World = \world. {world | tick: world.tick + 1}
-- ERROR: record update requires base to be a record type, got *types.TVar2
```

**After Fix:**
```ailang
type World = { tick: int }

-- Lambda with type annotation - WORKS
let step: World -> World = \world. {world | tick: world.tick + 1}

let w = { tick: 0 }
step(w)  -- Returns { tick: 1 }
```

### Example 2: Multiple Field Update

```ailang
type State = { x: int, y: int, active: bool }

-- Currently FAILS, should work after fix
let move: State -> int -> int -> State = \s dx dy. {s | x: s.x + dx, y: s.y + dy}

let s = { x: 0, y: 0, active: true }
move(s)(5)(10)  -- Returns { x: 5, y: 10, active: true }
```

### Example 3: Let-Bound Record (Already Works)

```ailang
-- This already works - no lambda parameter involved
let person = { name: "Alice", age: 25 }
let updated = {person | age: 30}
updated.age  -- Returns 30
```

### Example 4: Row Polymorphic Update (Future Enhancement)

```ailang
-- Row polymorphism - may need additional work beyond this fix
let setName: { name: string | r } -> { name: string | r } = \rec. {rec | name: "updated"}

let user = { name: "Alice", email: "alice@example.com" }
setName(user)  -- Should return { name: "updated", email: "alice@example.com" }
```

## Success Criteria

- [ ] `{base | field: val}` works with lambda parameters (main bug fix)
- [ ] `{base | field: val}` continues to work with let-bound records (regression)
- [ ] Multiple field updates work: `{s | x: 1, y: 2}`
- [ ] Field access in update value works: `{s | x: s.x + 1}`
- [ ] Clear error for genuinely non-record base types
- [ ] All existing tests pass
- [ ] Example file added and verified

## Testing Strategy

**Unit tests** (`tests/record_update_inference_test.ail`):
```ailang
-- Test 1: Lambda with type annotation
type T1 = { x: int }
let f1: T1 -> T1 = \t. {t | x: t.x + 1}
assert f1({ x: 5 }).x == 6

-- Test 2: Lambda without type annotation (type flows from usage)
let f2 = \t. {t | x: t.x + 1}
assert f2({ x: 5 }).x == 6

-- Test 3: Let-bound record (regression)
let r3 = { x: 10, y: 20 }
let r3b = {r3 | x: 15}
assert r3b.x == 15
assert r3b.y == 20

-- Test 4: Multiple field update
type T4 = { a: int, b: int, c: int }
let f4: T4 -> T4 = \t. {t | a: 1, b: 2}
assert f4({ a: 0, b: 0, c: 3 }).c == 3

-- Test 5: Error case (should fail type check)
-- let bad = \x. {x | name: "test"}  -- x is not known to be a record
```

**Manual testing:**
- Verify stapledons_voyage use case works after fix

## Non-Goals

- **Not in this fix:**
  - Nested field update syntax (`{rec | pos.x: newX}`)
  - Lens-based updates
  - Partial record construction
  - Full row polymorphism support (may need follow-up)

## Timeline

**Day 1** (~5 hours):
- Implement constraint-based fix in `inferRecordUpdate`
- Add basic test cases
- Verify existing tests pass

**Day 2** (~3 hours):
- Edge case testing
- Create example file
- Update CHANGELOG.md
- Respond to stapledons_voyage

**Total: ~8 hours across 2 days**

## Risks & Mitigations

| Risk | Impact | Probability | Mitigation |
|------|--------|-------------|-----------|
| Row polymorphism edge cases | Medium | Medium | Test open records, may defer complex cases |
| Unification order issues | High | Low | Follow `inferRecordAccess` pattern exactly |
| Breaking let-bound record updates | High | Low | Add regression test first |
| Result type inference | Medium | Medium | Use base type as result type initially |

## References

- stapledons_voyage bug report (agent inbox, 2025-11-28)
- `internal/types/typechecker_data.go:99-162` - `inferRecordUpdate` function (bug location)
- `internal/types/typechecker_data.go:59-95` - `inferRecordAccess` function (pattern to follow)
- `internal/types/inference.go:564-615` - Constraint solving mechanism

## Future Work

- Row polymorphic record updates
- Nested field update syntax (`{rec | pos.x: newX}`)
- Lens-style deep updates
- Record spread operator

---

**Document created**: 2025-11-28
**Last updated**: 2025-11-29 (verified root cause, updated fix approach)
