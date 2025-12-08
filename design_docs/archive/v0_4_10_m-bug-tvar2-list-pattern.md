# M-BUG-TVAR2-LIST-PATTERN: Record Types in List Pattern Bindings

## Status: Planned (v0.5.0)

## Problem Statement

When matching on a list of records (`[{...}]`), accessing fields on the match-bound variable fails with a TVar2 unification error. This affects all record types in list patterns, not just nested field access.

**Reported by:** stapledons_voyage (via agent inbox, 2025-12-01)

## Error Message

```
type error in module_name (decl N): type unification failed at [list pattern]: cannot unify open record with *types.TVar2
```

## Reproduction

### Minimal Failing Case
```ailang
module test/list_pattern_bug

type Position = { x: int, y: int }

pure func getFirstX(positions: [Position]) -> int {
    match positions {
        pos :: rest => pos.x,   -- FAILS: TVar2 error
        [] => 0
    }
}

let ps = [{ x: 10, y: 20 }]
getFirstX(ps)
```

### Nested Records Also Fail
```ailang
type NPC = { pos: Position, name: string }

pure func getNpcX(npcs: [NPC]) -> int {
    match npcs {
        npc :: rest => npc.pos.x,  -- FAILS: same TVar2 error
        [] => 0
    }
}
```

### Works: Non-Record Types
```ailang
pure func getFirst(nums: [int]) -> int {
    match nums {
        n :: rest => n,  -- WORKS
        [] => 0
    }
}
```

### Works: Direct Function Parameters
```ailang
-- Interestingly, record access through function params works
pure func getX(pos: Position) -> int {
    pos.x  -- WORKS
}

pure func getNpcX(npc: NPC) -> int {
    npc.pos.x  -- WORKS (confirmed in v0.4.9 testing)
}
```

## Analysis

### What Works
- Record field access through function parameters: `npc.pos.x`
- Record field access through let bindings: `let npc = {...}; npc.pos.x`
- List patterns with primitive types: `n :: rest => n`
- Inline nested record literals (fixed in v0.4.9)

### What Fails
- Record field access through list pattern bindings: `pos :: rest => pos.x`
- Any record type in list patterns, regardless of nesting depth

### Suspected Root Cause

The issue likely occurs during pattern type inference in `internal/types/` when:

1. The list pattern `pos :: rest` is typed
2. The element type is inferred as an open record type (row-polymorphic)
3. When `pos.x` is type-checked, the unifier tries to match the open record against a TVar2
4. The unification fails because TVar2 handling doesn't properly account for row polymorphism

The difference between function parameters and match bindings:
- Function params have explicit type annotations that guide inference
- Match bindings rely on pattern inference which may not properly instantiate record types

## Workarounds

**None confirmed working.** Even intermediate let bindings fail:

```ailang
match npcs {
    npc :: rest => {
        let pos = npc.pos;  -- Still triggers the error
        pos.x
    },
    [] => 0
}
```

The only workaround is to avoid records in list patterns entirely, which is severely limiting.

## Investigation Areas

1. **`internal/types/infer.go`** - Pattern type inference
2. **`internal/types/unify.go`** - TVar2 unification with open records
3. **`internal/types/pattern.go`** - List pattern handling
4. **`internal/types/row.go`** - Row polymorphism implementation

### Key Questions

1. How does list pattern typing handle element types?
2. Is the element type being properly instantiated before field access?
3. Is there a missing substitution step between pattern binding and body type checking?
4. Does the unifier correctly handle TVar2 vs open record types?

## Proposed Fix

### Phase 1: Diagnosis
- Add debug logging to pattern type inference
- Trace the type of match-bound variables through elaboration
- Identify where TVar2 is introduced vs where record type is expected

### Phase 2: Fix
TBD based on diagnosis. Likely candidates:
- Missing type instantiation in pattern bindings
- Incorrect TVar2 creation for record types in patterns
- Row unification bug when element type comes from pattern

## Impact

**High** - This blocks any match expression over lists of records, which is a fundamental programming pattern. Stapledons_voyage reports this blocks their game engine development.

## Test Cases

```ailang
-- test_list_pattern_record.ail
module test/list_pattern_record

-- Case 1: Simple record in list pattern
type Point = { x: int, y: int }

pure func sumX(points: [Point]) -> int {
    match points {
        p :: rest => p.x + sumX(rest),
        [] => 0
    }
}

-- Case 2: Nested record in list pattern
type Entity = { pos: Point, name: string }

pure func sumEntityX(entities: [Entity]) -> int {
    match entities {
        e :: rest => e.pos.x + sumEntityX(rest),
        [] => 0
    }
}

-- Case 3: Record update in list pattern
pure func moveAll(points: [Point], dx: int) -> [Point] {
    match points {
        p :: rest => { p | x: p.x + dx } :: moveAll(rest, dx),
        [] => []
    }
}

-- All should pass
sumX([{x: 1, y: 2}, {x: 3, y: 4}])  -- Expected: 4
sumEntityX([{pos: {x: 10, y: 20}, name: "a"}])  -- Expected: 10
moveAll([{x: 0, y: 0}], 5)  -- Expected: [{x: 5, y: 0}]
```

## Related Issues

- M-BUG-NESTED-RECORD-ANF (fixed in v0.4.9) - Different issue (ANF verification)
- M-BUG-RECORD-UPDATE-INFERENCE (v0.4.8) - Related to record type unification
- Row polymorphism implementation

## References

- Stapledons message: msg_20251201_145005_2827a1f4db2b
- v0.4.9 release: Fixed ANF issues but not this type unification bug
