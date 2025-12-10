# M-FIX-NESTED-RECORD-LIST: Nested Record List Type Inference Bug

**Status**: Planned
**Target**: v0.5.9
**Priority**: P1 - Medium (blocks real-world codebase)
**Estimated**: 4-6 hours
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
- Regression from M-FIX-RECORD-UPDATE changes (likely interaction)

## Goals

**Primary Goal:** Fix type inference for lists containing records with nested record type fields

**Success Metrics:**
- [ ] `ailang check sim/bridge.ail` passes
- [ ] Test case with `[ConsoleState]` return type passes
- [ ] No regression in existing tests
- [ ] M-FIX-RECORD-UPDATE still works (record update syntax)

## Diagnosis

### Hypothesis 1: Type Alias Expansion in List Context

The recently added type alias expansion (M-FIX-RECORD-UPDATE) may be expanding nested record types incorrectly in list context:

1. Return type annotation `[ConsoleState]` gets type checked
2. ConsoleState is expanded to `{ station: Station, pos: Coord, ... }`
3. Coord is also expanded to `{ x: int, y: int }`
4. **Bug**: Fields from inner Coord get mixed with outer ConsoleState

### Hypothesis 2: Row Unification Path Confusion

The row unification for records in list element types may be following the wrong path:

1. List literal elements are inferred
2. Each element is a record literal `{ station: ..., pos: makeCoord(...), ... }`
3. `pos` field has type inferred from `makeCoord()` → Coord → `{x: int, y: int}`
4. **Bug**: Unifier tries to unify ConsoleState row with Coord row somewhere

### Hypothesis 3: TRecord vs TRecordOpen Confusion

Similar to M-FIX-RECORD-UPDATE, there may be a closed vs open record confusion:

1. `[ConsoleState]` expects `TRecord{fields: ConsoleState fields}`
2. List literal infers each element as `TRecordOpen{...}` (open row)
3. Nested `pos: Coord` becomes `pos: TRecord{x: int, y: int}`
4. **Bug**: Somewhere the flattening happens

## Solution Design

### Investigation Steps

**Step 1: Create Minimal Reproduction**
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

**Step 2: Enable Debug Tracing**
```bash
DEBUG_ALIAS=1 ailang check test/nested_record_list.ail
```

**Step 3: Trace Unification Path**
- Add logging at row unification entry points
- Track which types are being unified when error occurs
- Identify where Coord fields leak into ConsoleState

### Implementation Plan

**Phase 1: Reproduce and Diagnose** (~2 hours)
- [ ] Create minimal test case that reproduces the bug
- [ ] Add debug tracing to identify unification failure point
- [ ] Understand the exact type flow

**Phase 2: Fix** (~2-3 hours)
- [ ] Identify root cause from Phase 1
- [ ] Implement fix (likely in unification_core.go or unification_types.go)
- [ ] Add unit test for the specific pattern

**Phase 3: Validate** (~1 hour)
- [ ] Run full test suite
- [ ] Test against sim/bridge.ail
- [ ] Verify M-FIX-RECORD-UPDATE still works

### Files to Modify/Create

**Likely modified files (based on similar bugs):**
- `internal/types/unification_core.go` - Main unification logic (~50-100 LOC change)
- `internal/types/unification_types.go` - Type-specific unification (~20-50 LOC change)
- `internal/iface/builder.go` - Type alias handling (~10-20 LOC change)

**New files:**
- `internal/types/nested_record_test.go` - Test case (~50 LOC)

## Examples

### Example 1: Failing Case

**Code:**
```ailang
type Coord = { x: int, y: int }
type Entity = { pos: Coord, name: string }

pure func createEntities() -> [Entity] {
    [{ pos: { x: 1, y: 2 }, name: "test" }]
}
```

**Current Error:**
```
type unification failed at [return type annotation]:
incompatible closed rows: r1 has extra labels [name pos], r2 has extra labels [x y]
```

**Expected:** No error, compiles successfully

### Example 2: Related Working Case (M-FIX-RECORD-UPDATE)

**Code:**
```ailang
type Pos = { x: int, y: int }
type NPC = { pos: Pos, name: string }

pure func updateNpcPos(npc: NPC, newX: int, newY: int) -> NPC {
    { npc | pos: { x: newX, y: newY } }
}
```

**Status:** This now works after M-FIX-RECORD-UPDATE

## Success Criteria

- [ ] Minimal test case passes
- [ ] sim/bridge.ail compiles successfully
- [ ] npc_ai.ail still passes (regression check)
- [ ] All existing tests passing
- [ ] No performance regression

## Testing Strategy

**Unit tests:**
- Test nested record in list literal
- Test nested record with type annotation
- Test multiple levels of nesting

**Integration tests:**
- Test actual sim/bridge.ail file
- Test stapledons_voyage compilation

**Regression tests:**
- Verify M-FIX-RECORD-UPDATE cases still work
- Verify M-FIX-FLOAT-OP cases still work

## Non-Goals

**Not in this fix:**
- General type inference debugging tools (M-DX11 scope)
- Performance optimization of unification
- Additional row polymorphism features

## Timeline

**Session 1** (4-6 hours):
- Phase 1: Reproduce and diagnose
- Phase 2: Implement fix
- Phase 3: Validate

**Total: ~4-6 hours in single session**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Regression in record update | High | Test M-FIX-RECORD-UPDATE cases |
| Complex interaction with type aliases | Medium | Careful tracing and minimal fix |
| May require architectural change | Medium | Start with investigation first |

## References

- GitHub Issue #25: Record list type inference confuses nested record types
- M-FIX-RECORD-UPDATE (completed): Cross-module record update fix
- M-DX11: Type inference debugging tools (planned)
- Previous bugs #23/#24: Record update type errors (fixed)

## Debug Session Notes

*To be filled during investigation:*

```
T+0h: Start investigation
T+Xh: [Add findings here]
```

---

**Document created**: 2025-12-10
**Last updated**: 2025-12-10
