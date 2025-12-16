# M-DX29: Fix Nested ADT Type in Option Pattern Match

**Status**: Implemented
**Target**: v0.5.11
**Priority**: P1 (Medium-High - blocks external projects)
**Estimated**: 3 hours
**Dependencies**: M-DX27 (typedLocalVars infrastructure)

## AI-First Alignment Check

**Score this feature against AILANG's core principles:**

| Principle | Impact | Score | Notes |
|-----------|--------|-------|-------|
| Reduce Syntactic Noise | 0 | 0 | No change to AILANG syntax |
| Preserve Semantic Clarity | + | +1 | Fixes incorrect Go codegen - generated code matches intent |
| Increase Determinism | + | +1 | Same AILANG always produces same valid Go code |
| Lower Token Cost | 0 | 0 | No change to token footprint |
| **Net Score** | | **+2** | **Decision: Move forward** |

**Decision rule:** Net score > +1 -> Move forward | <= 0 -> Reject or redesign

## Problem Statement

When pattern matching on `Option[ADT]` and then matching the inner ADT value, the generated Go code types the extracted ADT value as `interface{}` instead of the concrete ADT pointer type.

**Current State:**
- Nested ADT pattern match on Option generates invalid Go code
- `go build` fails with: `_adt.Kind undefined (type interface{} has no field or method Kind)`
- External projects (stapledons_voyage) blocked by this bug

**Reproduction:**
```ailang
type InteractableID =
  | InteractConsole(BridgeStation)
  | InteractCrew(CrewID)
  | InteractHatch(int)

type State = { hoveredInteractable: Option[InteractableID] }

match state.hoveredInteractable {
    Some(interactable) =>
        match interactable {  -- interactable should be *InteractableID
            InteractConsole(station) => ...
            InteractCrew(crew) => ...
            InteractHatch(h) => ...
        }
    None => ...
}
```

**Generated Go (invalid):**
```go
// interactable extracted as interface{}
interactable := _opt.Value  // type: interface{}

// Then tries to access .Kind on interface{}
_adt := interactable       // still interface{}
switch _adt.Kind {         // ERROR: interface{} has no field Kind
case InteractableIDKindInteractConsole:
    station := _adt.InteractConsole  // ERROR: interface{} has no field InteractConsole
```

**Expected Go (valid):**
```go
interactable := _opt.Value.(*InteractableID)  // type assertion to concrete type

_adt := interactable  // now *InteractableID
switch _adt.Kind {    // works - Kind is a field
case InteractableIDKindInteractConsole:
    station := _adt.InteractConsole.Value0  // works
```

**Impact:**
- External projects using Option[ADT] patterns cannot compile
- Blocks stapledons_voyage project development
- Common pattern in game state management

## Goals

**Primary Goal:** Generate correct type assertions when extracting ADT values from Option types

**Success Metrics:**
- Nested Option[ADT] pattern match compiles to valid Go
- Generated code includes proper `*ADTName` type assertion
- stapledons_voyage builds successfully
- No regression in existing Option handling

## Solution Design

### Overview

Extend the M-DX27 `typedLocalVars` tracking to record ADT pointer types when extracting from Option. When a variable is bound via `Some(x)` where Option's type parameter is an ADT, record that `x` has type `*ADTName`.

### Architecture

**Root Cause Analysis:**
1. Option is generated as `type Option[T] struct { IsSome bool; Value interface{} }`
2. When pattern matching `Some(x)`, codegen extracts `x := _opt.Value` (interface{})
3. If `x` is then used in a nested match, codegen doesn't know it needs `*ADTType`
4. M-DX27's `typedLocalVars` only tracks primitive types (bool, float64, etc.)

**Solution:**
1. When generating Option Some arm extraction, check if the Option's type parameter is an ADT
2. If so, generate `x := _opt.Value.(*ADTName)` instead of `x := _opt.Value`
3. Record in `typedLocalVars` that `x` has type `*ADTName`
4. Subsequent ADT pattern matches on `x` will use correct type

### Implementation Plan

**Phase 1: Investigate Option Codegen** (~1 hour)
- [ ] Find where Option Some extraction is generated
- [ ] Understand how Option type parameter info is available
- [ ] Trace how nested match receives scrutinee type

**Phase 2: Add ADT Type Tracking** (~1.5 hours)
- [ ] Modify Option Some arm generation to add type assertion for ADT values
- [ ] Extend `typedLocalVars` to track ADT pointer types (not just primitives)
- [ ] Update `exprProducesInterface` to handle ADT-typed variables

**Phase 3: Testing** (~0.5 hours)
- [ ] Add unit test for nested Option[ADT] pattern match
- [ ] Verify stapledons_voyage compiles
- [ ] Run full test suite

### Files to Modify/Create

**Modified files:**
- `internal/gen/golang/codegen_match.go` - Add type assertion for ADT extraction from Option (~20 LOC)
- `internal/gen/golang/codegen_match_test.go` - Add test case (~50 LOC)

## Examples

### Example 1: Nested Option[ADT] Match

**Before (invalid Go):**
```go
if _opt.IsSome {
    interactable := _opt.Value  // interface{}
    _adt := interactable        // interface{}
    switch _adt.Kind {          // ERROR!
```

**After (valid Go):**
```go
if _opt.IsSome {
    interactable := _opt.Value.(*InteractableID)  // *InteractableID
    _adt := interactable                           // *InteractableID
    switch _adt.Kind {                             // OK
```

## Success Criteria

- [ ] `Option[ADT]` nested match generates compilable Go code
- [ ] Type assertion `*ADTName` added when extracting ADT from Option
- [ ] stapledons_voyage `make sim && go build` succeeds
- [ ] All existing tests pass
- [ ] New unit test covers this case

## Testing Strategy

**Unit tests:**
- Test nested `Option[ADT]` pattern match codegen
- Verify generated code contains type assertion

**Integration tests:**
- Compile stapledons_voyage with fix
- Verify generated Go code builds

**Manual testing:**
- Build stapledons_voyage game

## Non-Goals

**Not in this feature:**
- Generic Option type parameter tracking - Only ADTs for now
- Deeply nested Option[Option[ADT]] - Single level first

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| May need ADT type info not currently available | Med | Check what info codegen has from type checker |
| Could affect non-ADT Option extractions | Med | Only add assertion when type param is known ADT |

## References

- M-DX27: Bool type assertion fix (established `typedLocalVars` pattern)
- Bug report from stapledons_voyage (msg_20251212_222709_f36f0a4c)

---

**Document created**: 2025-12-12
**Last updated**: 2025-12-12
