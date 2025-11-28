# M-BUG-RECORD-UPDATE-INFERENCE: Fix Record Update Type Inference

**Status**: Planned
**Target**: v0.4.9
**Priority**: P1 (Medium) - Affects immutable update patterns
**Estimated**: 1-2 days
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

**Reproduction:**
```ailang
type World = { tick: int, entities: [Entity] }

func step(world: World) -> World {
  let newTick = world.tick + 1
  {world | tick: newTick}  -- ERROR: record update requires base to be a record type, got *types.TVar2
}
```

**Current Workaround:** Construct new record explicitly:
```ailang
func step(world: World) -> World {
  { tick: world.tick + 1, entities: world.entities }
}
```

**Current State:**
- Record update syntax parses correctly
- Type inference fails when base expression type not yet resolved
- The ailang prompt documents `{base | field: val}` syntax
- Works when constructing new record explicitly

**Impact:**
- Verbose code for immutable updates
- Extra maintenance burden (must update all fields)
- Breaks common functional programming patterns

## Root Cause Analysis

The error message "got *types.TVar2" indicates the type checker is seeing an unresolved type variable instead of a concrete record type.

**Likely cause:** Type inference order issue
1. `world` has type `World` in function signature
2. When checking `{world | tick: newTick}`, the checker gets `world`'s type
3. But the type hasn't been substituted yet (still a TVar)
4. Record update check fails because TVar is not a record type

**Expected behavior:**
1. Look up `world` in environment → type is `World`
2. Substitute any type variables → `World` is already concrete
3. Check that `World` is a record type → yes
4. Check that `tick` is a field of `World` → yes
5. Proceed with update

## Goals

**Primary Goal:** Record update syntax works when base expression has a known record type.

**Success Metrics:**
- `{world | field: val}` works for typed function parameters
- `{world | field: val}` works for let-bound records
- Clear error if base type is genuinely not a record
- Row polymorphism preserved (can update open records)

## Solution Design

### Overview

Fix type inference to resolve the base expression's type before checking if it's a record type.

### Architecture

**Investigation points:**
1. Where record update is type-checked
2. Why substitution isn't applied before the record check
3. Whether this is a unification ordering issue

**Proposed Fix:**

In the type checker for record update expressions:
1. Infer type of base expression
2. Apply current substitution to resolve type variables
3. Then check if result is a record type

```go
// Before (pseudocode):
func (tc *TypeChecker) checkRecordUpdate(expr *ast.RecordUpdate) (types.Type, error) {
    baseType := tc.infer(expr.Base)
    if !isRecordType(baseType) {
        return nil, fmt.Errorf("requires record type, got %v", baseType)
    }
    // ...
}

// After:
func (tc *TypeChecker) checkRecordUpdate(expr *ast.RecordUpdate) (types.Type, error) {
    baseType := tc.infer(expr.Base)
    baseType = tc.substitute(baseType)  // Resolve type variables
    if !isRecordType(baseType) {
        return nil, fmt.Errorf("requires record type, got %v", baseType)
    }
    // ...
}
```

### Implementation Plan

**Phase 1: Diagnosis** (~2 hours)
- [ ] Find record update type checking code
- [ ] Add debug logging to see actual types
- [ ] Identify exactly where substitution is missing

**Phase 2: Fix** (~3 hours)
- [ ] Apply substitution before record type check
- [ ] Handle row polymorphism correctly
- [ ] Add test cases

**Phase 3: Testing** (~2 hours)
- [ ] Test with typed parameters
- [ ] Test with let-bound records
- [ ] Test with nested records
- [ ] Test error case (updating non-record)

### Files to Modify

**Modified files:**
- `internal/types/checker.go` - Record update inference (~20 LOC)
- `internal/types/unify.go` - May need substitution fix (~10 LOC)

**New files:**
- `tests/record_update_test.go` - Additional test cases (~50 LOC)

## Examples

### Example 1: Basic Record Update (Reported Case)

**Current (Fails):**
```ailang
type World = { tick: int, entities: [Entity] }

func step(world: World) -> World {
  {world | tick: world.tick + 1}
}
-- ERROR: record update requires base to be a record type, got *types.TVar2
```

**After Fix:**
```ailang
-- Same code works
func step(world: World) -> World {
  {world | tick: world.tick + 1}
}
-- Returns World with updated tick
```

### Example 2: Multiple Field Update

```ailang
type State = { x: int, y: int, active: bool }

func move(s: State, dx: int, dy: int) -> State {
  {s | x: s.x + dx, y: s.y + dy}
}
```

### Example 3: Nested Record Update

```ailang
type Player = { pos: { x: int, y: int }, health: int }

func heal(p: Player, amount: int) -> Player {
  {p | health: min(100, p.health + amount)}
}
```

### Example 4: Row Polymorphic Update

```ailang
func setName[r](rec: { name: string | r }) -> { name: string | r } {
  {rec | name: "updated"}
}
```

## Success Criteria

- [ ] `{base | field: val}` works with typed function parameters
- [ ] `{base | field: val}` works with let-bound records
- [ ] Multiple field updates work
- [ ] Nested field access works in update
- [ ] Row polymorphism preserved
- [ ] Clear error for non-record base types
- [ ] All existing tests pass
- [ ] Examples added

## Testing Strategy

**Unit tests:**
- Record update with concrete types
- Record update with type inference
- Multiple field updates
- Nested record updates

**Integration tests:**
- Game state update patterns
- Complex record hierarchies

**Manual testing:**
- Run stapledons_voyage code after fix

## Non-Goals

- **Not in this fix:**
  - Nested field update syntax (`{rec | pos.x: newX}`)
  - Lens-based updates
  - Partial record construction

## Timeline

**Day 1** (4 hours):
- Diagnosis and fix implementation

**Day 2** (3 hours):
- Testing and edge cases

**Total: ~7 hours across 2 days**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Row polymorphism interaction | Medium | Test with open records |
| Substitution in wrong place | Low | Check similar patterns in codebase |
| Breaking other type inference | High | Run full test suite |

## References

- stapledons_voyage bug report (agent inbox, 2025-11-28)
- `internal/types/checker.go` - Type checker
- `internal/types/unify.go` - Unification and substitution
- Row polymorphism documentation

## Future Work

- Nested field update syntax
- Lens-style deep updates
- Record spread operator

---

**Document created**: 2025-11-28
**Last updated**: 2025-11-28
