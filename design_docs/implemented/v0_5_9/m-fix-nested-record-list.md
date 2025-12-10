# M-FIX-NESTED-RECORD-LIST: Nested Record List Type Inference Bug

**Status**: ✅ IMPLEMENTED
**Target**: v0.5.9
**Priority**: P1 - Medium (blocks real-world codebase)
**Actual**: ~2 hours
**Dependencies**: M-FIX-RECORD-UPDATE (completed)
**GitHub Issue**: #25

## AI-First Alignment Check

**Score this feature against AILANG's core principles:**

| Principle | Impact | Score | Notes |
|-----------|--------|-------|-------|
| Reduce Syntactic Noise | 0 | 0 | Bug fix, no syntax change |
| Preserve Semantic Clarity | + | +1 | Correct type inference preserves meaning |
| Increase Determinism | + | +2 | Fixes non-deterministic type error |
| Lower Token Cost | 0 | 0 | No token impact |
| **Net Score** | | **+3** | **Decision: Move forward** |

## Problem Statement

**Error Message:**
```
Error: type error in sim/bridge (decl 46): type unification failed at [return type annotation at sim/bridge.ail:347:8]: incompatible closed rows: r1 has extra labels [active hasAlert pos spriteId station], r2 has extra labels [x y]
```

**Reproduction Pattern:**
```ailang
module sim/bridge

export type Coord = { x: int, y: int }
export type ConsoleState = {
    station: Station,
    pos: Coord,           -- Nested record type
    active: bool,
    hasAlert: bool,
    spriteId: string
}

export pure func createConsoles() -> [ConsoleState] {
    let cx = discCenterDefault();
    let cy = discCenterDefault();
    [
        { station: StationHelm, pos: makeCoord(cx, cy - 10), active: true, hasAlert: false, spriteId: spriteConsoleHelm() },
        ...
    ]
}
```

**Current State:**
- Return type `[ConsoleState]` where ConsoleState has field `pos: Coord`
- Type checker confuses ConsoleState fields (`active hasAlert pos spriteId station`) with Coord fields (`x y`)
- Previous bugs #23/#24 (record update) are fixed - npc_ai.ail now passes
- This bug blocks compilation of sim/bridge.ail

**Impact:**
- Real-world codebase (stapledons_voyage) cannot compile
- Pattern is common: list of records with nested record fields

## Root Cause Analysis

### Investigation Findings

**Hypothesis 1 (WRONG): Type Alias Expansion in List Context**
- Investigated but this was not the root cause
- Type alias expansion was working correctly

**Hypothesis 2 (WRONG): Row Unification Path Confusion**
- Investigated RowUnifier creating new Unifier without aliasEnv
- Added parent unifier reference to preserve aliasEnv
- This was a valid improvement but NOT the root cause

**Hypothesis 3 (CORRECT): Hardcoded Row Variable Name Collision**

The actual root cause was in `internal/types/unification_records.go` in the `unifyRecord` function:

```go
// BEFORE (buggy)
if t1.Row != nil || t2Rec.Row != nil {
    row1 := t1.Row
    if row1 == nil {
        row1 = &TVar2{Name: "ρ_empty", Kind: &KRow{ElemKind: &KRecord{}}}  // ← HARDCODED!
    }
    row2 := t2Rec.Row
    if row2 == nil {
        row2 = &TVar2{Name: "ρ_empty", Kind: &KRow{ElemKind: &KRecord{}}}  // ← SAME NAME!
    }
    return u.Unify(row1, row2, sub)
}
```

**Why this caused the bug:**

1. When unifying Entity record (nil Row), creates `TVar2{Name: "ρ_empty"}`
2. This gets bound in substitution: `ρ_empty → Row{pos, name}`
3. When unifying nested Coord record (also nil Row), creates ANOTHER `TVar2{Name: "ρ_empty"}`
4. Unification applies substitution, gets `Row{pos, name}` for Coord's row variable
5. Tries to unify `Row{pos, name}` with `Row{x, y}` → **FAIL!**

The hardcoded "ρ_empty" name caused row variable collision between outer and inner records.

## Solution

### Fix Implementation

**Added fresh row variable name generation in `internal/types/unification_core.go`:**

```go
type Unifier struct {
    rowUnifier    *RowUnifier
    depth         int
    rowVarCounter int // M-FIX-NESTED-RECORD-LIST: Counter for generating unique row variable names
    aliasEnv map[string]Type
}

// freshRowVarName generates a unique row variable name (M-FIX-NESTED-RECORD-LIST)
// This prevents nested records from sharing the same "ρ_empty" variable
func (u *Unifier) freshRowVarName() string {
    u.rowVarCounter++
    return fmt.Sprintf("ρ_empty_%d", u.rowVarCounter)
}
```

**Updated `unifyRecord` in `internal/types/unification_records.go`:**

```go
// AFTER (fixed)
if t1.Row != nil || t2Rec.Row != nil {
    row1 := t1.Row
    if row1 == nil {
        // M-FIX-NESTED-RECORD-LIST: Use fresh name to avoid conflicts with nested records
        row1 = &TVar2{Name: u.freshRowVarName(), Kind: &KRow{ElemKind: &KRecord{}}}
    }
    row2 := t2Rec.Row
    if row2 == nil {
        // M-FIX-NESTED-RECORD-LIST: Use fresh name to avoid conflicts with nested records
        row2 = &TVar2{Name: u.freshRowVarName(), Kind: &KRow{ElemKind: &KRecord{}}}
    }
    return u.Unify(row1, row2, sub)
}
```

### Secondary Fix (RowUnifier Parent Reference)

Also added parent unifier reference to RowUnifier for proper alias expansion:

```go
// internal/types/row_unification.go
type RowUnifier struct {
    freshCounter  int
    parentUnifier *Unifier // M-FIX-NESTED-RECORD-LIST: Reference to parent for alias expansion
}

func (ru *RowUnifier) SetParentUnifier(u *Unifier) {
    ru.parentUnifier = u
}
```

This ensures type aliases are properly expanded when unifying field types in rows.

### Files Modified

| File | Change | LOC |
|------|--------|-----|
| `internal/types/unification_core.go` | Added rowVarCounter, freshRowVarName(), SetParentUnifier call | +15 |
| `internal/types/unification_records.go` | Use fresh row variable names | +8 |
| `internal/types/row_unification.go` | Added parentUnifier field and SetParentUnifier method | +10 |

**Total: ~33 lines changed**

## Test Results

### Test Cases Created

**Same-file nested record list:**
```ailang
module test/nested_record_list

type Coord = { x: int, y: int }
type Entity = { pos: Coord, name: string }

pure func makeCoord(x: int, y: int) -> Coord {
    { x: x, y: y }
}

pure func createEntities() -> [Entity] {
    [
        { pos: makeCoord(1, 2), name: "a" },
        { pos: makeCoord(3, 4), name: "b" }
    ]
}
```
**Result:** ✅ PASS

**Cross-module nested record list:**
```ailang
module lib/types
export type Coord = { x: int, y: int }
export type Entity = { pos: Coord, name: string }

-- In another file:
module app/main
import lib/types (Coord, Entity)
-- Uses Entity with nested Coord
```
**Result:** ✅ PASS

### Regression Tests

| Test | Status |
|------|--------|
| M-FIX-RECORD-UPDATE (record update syntax) | ✅ PASS |
| M-FIX-FLOAT-OP (float operations) | ✅ PASS |
| Full test suite (`make test`) | ✅ PASS |
| Lint (`make lint`) | ✅ PASS |

## Success Criteria

- [x] Minimal test case passes
- [x] Nested record in list literal works
- [x] Cross-module nested records work
- [x] npc_ai.ail still passes (regression check)
- [x] All existing tests passing
- [x] No performance regression

## Lessons Learned

1. **Fresh variable generation is critical** - Never use hardcoded variable names in type inference
2. **Nested structures need unique names** - Each record level needs its own fresh row variable
3. **Test both same-file and cross-module** - Import/export can introduce different code paths

## References

- GitHub Issue #25: Record list type inference confuses nested record types
- M-FIX-RECORD-UPDATE (completed): Cross-module record update fix
- M-DX11: Type inference debugging tools (planned)
- Previous bugs #23/#24: Record update type errors (fixed)

---

**Document created**: 2025-12-10
**Implemented**: 2025-12-10
