# M-GAP4: Record Width Subtyping via Row Polymorphism

## Status
- **Status:** Planned
- **Target:** v0.6.4+
- **Priority:** P1 (High)
- **Estimated:** 2-3 days
- **Dependencies:** None (extends existing row polymorphism)

## Problem Statement

Records in AILANG must match exactly - there is no width subtyping. You cannot pass a record with extra fields to a function expecting fewer fields.

### Current Behavior (Broken)
```ailang
pure func countTurns(events: [{turnNum: int}]) -> int =
  length(events)

let events = [{turnNum: 1, streamType: "text", text: "hello"}]
countTurns(events)
-- ✗ Error: field count mismatch: 1 vs 3
```

### Expected Behavior
```ailang
-- Should work: {turnNum: int, streamType: string, text: string}
-- is a subtype of {turnNum: int}
countTurns(events)  -- ✓ Returns 1
```

### Impact
- Cannot write generic functions that operate on record subsets
- Must repeat full record types in every function signature
- Breaks modularity and code reuse
- Discovered during dogfooding (event processing functions)

## Goals

**Primary Goal:** Enable functions to accept records with additional fields beyond what they require

**Success Metrics:**
- `{a: T, b: U, ...}` unifies with `{a: T | r}` where `r` captures extra fields
- Functions can specify minimum required fields
- No breaking changes to existing code

## Solution Design

### Overview

AILANG already has row polymorphism infrastructure (row variables `| r` in types). The issue is that the unification algorithm enforces exact field count matching instead of allowing width subtyping.

### Current Type System Behavior

```ailang
-- Row polymorphism syntax exists
type HasName = {name: string | r}

-- But unification enforces exact match
func greet(x: {name: string}) = "Hello, " ++ x.name
greet({name: "Alice", age: 30})  -- ✗ Fails (should work)
```

### Proposed Changes

**1. Unification: Allow record widening**

When unifying `{a: T, b: U, c: V}` with `{a: T}`:
- Current: FAIL (field count mismatch)
- Proposed: SUCCEED, treating as `{a: T | {b: U, c: V}}`

**2. Type inference: Infer row variables for record parameters**

```ailang
-- Inferred type: {turnNum: int | r} -> int
pure func countTurns(events: [{turnNum: int}]) -> int = ...

-- Now accepts any record with at least turnNum field
```

**3. Explicit row variable syntax (already exists)**

```ailang
-- Explicit row polymorphism
pure func getName[r](person: {name: string | r}) -> string =
  person.name

getName({name: "Alice"})                    -- ✓
getName({name: "Bob", age: 30})             -- ✓
getName({name: "Carol", role: "admin"})     -- ✓
```

### Implementation Plan

#### Phase 1: Unification Changes (~4 hours)

**File:** `internal/types/unify.go`

```go
// Current behavior (unifyRecord):
if len(fields1) != len(fields2) {
    return fmt.Errorf("field count mismatch: %d vs %d", ...)
}

// Proposed behavior:
// If one record has more fields, create row extension
if len(fields1) > len(fields2) {
    // fields2 is smaller - check it's a subset of fields1
    extraFields := findExtraFields(fields1, fields2)
    // Create row variable for extra fields
    rowVar := freshRowVar()
    // Unify fields2 with {common fields | rowVar}
}
```

#### Phase 2: Type Inference (~4 hours)

**File:** `internal/types/infer.go`

- When inferring record parameter types, add implicit row variable
- `{a: T}` in parameter position → `{a: T | r}` with fresh `r`

#### Phase 3: Error Messages (~2 hours)

Improve error messages for record type mismatches:
- Show which fields are missing
- Suggest using row polymorphism explicitly

### Files to Modify

| File | Change | LOC |
|------|--------|-----|
| `internal/types/unify.go` | Width subtyping in unification | ~50 |
| `internal/types/infer.go` | Implicit row variables | ~30 |
| `internal/types/types.go` | Helper functions | ~20 |
| `internal/errors/type_errors.go` | Better error messages | ~20 |

### Design Decisions

**Q: Should width subtyping be implicit or explicit?**

A: **Implicit for function parameters, explicit for type aliases.**
- Function parameters: `{name: string}` implicitly means "at least name"
- Type aliases: `type Person = {name: string}` means exactly those fields
- Explicit row syntax: `{name: string | r}` for explicit polymorphism

**Q: Does this affect record literals?**

A: No. Record literals have exact types. Width subtyping only applies at function boundaries.

**Q: Performance impact?**

A: Minimal. Row variables are already supported; this just changes when they're introduced.

## Examples

### Before (doesn't work)
```ailang
pure func summarize(events: [{turnNum: int}]) -> string =
  "Total turns: " ++ intToString(length(events))

let fullEvents = [
  {turnNum: 1, text: "hello", streamType: "text"},
  {turnNum: 2, text: "world", streamType: "text"}
]

summarize(fullEvents)  -- ✗ Error: field count mismatch
```

### After (works)
```ailang
-- Same function definition
pure func summarize(events: [{turnNum: int}]) -> string =
  "Total turns: " ++ intToString(length(events))

-- Works because {turnNum, text, streamType} ⊇ {turnNum}
summarize(fullEvents)  -- ✓ Returns "Total turns: 2"
```

### Explicit Row Polymorphism (already works, still works)
```ailang
pure func extractField[r](records: [{id: int | r}]) -> [int] =
  map(\rec. rec.id, records)
```

## Testing

### New Test Cases
```ailang
-- test_record_width_subtyping.ail

-- Basic width subtyping
pure func getName(x: {name: string}) -> string = x.name
let _ = getName({name: "Alice", age: 30})  -- Should work

-- List of records
pure func countItems(xs: [{id: int}]) -> int = length(xs)
let _ = countItems([{id: 1, data: "x"}, {id: 2, data: "y"}])

-- Nested records
pure func getCity(x: {address: {city: string}}) -> string =
  x.address.city
let _ = getCity({address: {city: "NYC", zip: "10001"}, name: "Alice"})
```

### Regression Tests
- [ ] Existing record tests still pass
- [ ] Exact-match type aliases still enforce exact match
- [ ] Row polymorphism tests still pass
- [ ] Pattern matching on records still works

## Success Criteria

- [ ] Functions accept records with extra fields
- [ ] No breaking changes to existing code
- [ ] Clear error messages for actual type mismatches
- [ ] Documentation updated

## Timeline

**Day 1:** Unification changes + basic tests (4 hours)
**Day 2:** Inference changes + comprehensive tests (4 hours)
**Day 3:** Error messages + documentation (2 hours)

## Axiom Alignment

| Axiom | Score | Rationale |
|-------|-------|-----------|
| A7: Machines First | +1 | More flexible API design for AI-generated code |
| A10: Composability | +1 | Functions compose better with record subtyping |
| A5: Bounded Verification | 0 | Local type checking preserved |

**Net Score:** +2 (Accept)

## Related Documents

- [internal/types/unify.go](../../../internal/types/unify.go) - Unification implementation
- [internal/types/types.go](../../../internal/types/types.go) - Row type definitions
- Haskell's row polymorphism (PureScript, Elm records)

## Risks

**Risk:** Breaking change for code relying on exact record matching
- **Mitigation:** Only apply width subtyping at function boundaries, not type aliases

**Risk:** Inference becomes less predictable
- **Mitigation:** Clear documentation + explicit syntax available
