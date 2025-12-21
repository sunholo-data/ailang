# M-DX22: Go Codegen - ADT Constructor Resolution Ambiguity

**Status**: IMPLEMENTED
**Target**: v0.6.2
**Priority**: P0 (Blocking - breaks multi-ADT projects)
**Estimated**: 3 hours
**Actual**: ~1 hour (140 LOC)
**Dependencies**: None
**Reporter**: stapledons_voyage (agent message)

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No change to execution semantics |
| A2: Replayability | 0 | No impact on traces |
| A3: Effect Legibility | 0 | No effect handling changes |
| A4: Explicit Authority | 0 | No capability changes |
| A5: Bounded Verification | +1 | Enables correct resolution without manual workarounds |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | AI agents can compile multi-ADT projects correctly |
| A8: Minimal Syntax | 0 | No syntax changes |
| A9: Cost Visibility | 0 | Resource costs unchanged |
| A10: Composability | +1 | Multiple ADT types with same-named constructors compose correctly |
| A11: Structured Failure | 0 | No error handling changes |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +3** → **Decision: Move forward**

### Hard Violation Check

**These axioms cannot have −1 scores (automatic rejection):**

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Improves machine (AI agent) experience

## Problem Statement

When multiple ADT types have constructors with the same name, the Go codegen incorrectly resolves constructor calls to the wrong ADT type.

**Current State:**

The `adtConstructors` map in `codegen.go` uses constructor names as keys:
```go
adtConstructors map[string]*ADTConstructorInfo  // Key: "Viewport", not "DrawCmd.Viewport"
```

When two ADT types have constructors with the same name, only one can be stored in the map. The codegen then generates calls to the wrong constructor.

**Bug Report (stapledons_voyage):**

```
interior.ail imports: Viewport from sim/protocol (DrawCmd.Viewport)
interior.ail does NOT import sim/window (which has WindowType.Viewport)

Error: sim_gen/interior.go:907:32: too many arguments in call to NewWindowTypeViewport
    have (string, int64, interface{}, ...)
    want ()
```

The codegen called `NewWindowTypeViewport(...)` with `DrawCmd.Viewport` parameters instead of `NewDrawCmdViewport(...)`.

**Impact:**
- Breaks any project with multiple ADT types having same-named constructors
- Common patterns like `Result.Ok/Error` or `Option.Some/None` across different types won't work
- Forces users to rename constructors to avoid collisions

## Goals

**Primary Goal:** Resolve ADT constructors using type-qualified names to handle same-named constructors in different ADT types.

**Success Metrics:**
- Multiple ADT types with same-named constructors compile correctly
- Import context determines which constructor is used
- No regressions for single-ADT projects
- stapledons_voyage compiles successfully

## Solution Design

### Overview

Change the `adtConstructors` map to key by fully-qualified name `TypeName.CtorName` instead of just `CtorName`. Update all lookup sites to use qualified names.

### Architecture

**Current (broken):**
```go
// Registration
g.adtConstructors["Viewport"] = &ADTConstructorInfo{...}  // Only one Viewport!

// Lookup
if info, ok := g.adtConstructors["Viewport"]; ok { ... }  // Wrong one might be returned
```

**Fixed:**
```go
// Registration
g.adtConstructors["DrawCmd.Viewport"] = &ADTConstructorInfo{...}
g.adtConstructors["WindowType.Viewport"] = &ADTConstructorInfo{...}

// Lookup - use type context from elaborator
if info, ok := g.adtConstructors["DrawCmd.Viewport"]; ok { ... }
```

**Components:**
1. **Registry Key Change**: Use `TypeName.CtorName` as map key
2. **Lookup Enhancement**: Extract type name from VarGlobal module/context
3. **Fallback for Compatibility**: If unqualified lookup fails, try qualified lookup

### Implementation Plan

**Phase 1: Change Registry Keys** (~1 hour)
- [ ] Modify `registerADTConstructor` to use `TypeName.CtorName` key
- [ ] Update `RegisterADTConstructorWithFieldTypes` similarly
- [ ] Ensure ADT type name is passed through registration

**Phase 2: Update Lookup Sites** (~1.5 hours)
- [ ] `codegen_expr_simple.go:95,105` - VarGlobal ADT constructor lookup
- [ ] `codegen_expr_app.go:261,267` - App expression constructor calls
- [ ] `codegen_match.go:78,513` - Pattern matching constructor checks
- [ ] `codegen_ops.go:448,453,463,468` - Operator lowering
- [ ] `codegen_decl.go:502,521,529,546` - Declaration processing

**Phase 3: Testing & Validation** (~30 min)
- [ ] Add test case with two ADTs having same-named constructors
- [ ] Verify stapledons_voyage compiles
- [ ] Run full test suite

### Files to Modify

**Modified files:**
- `internal/gen/golang/codegen.go` - Change `adtConstructors` key format (~20 LOC)
- `internal/gen/golang/codegen_expr_simple.go` - Update lookups (~10 LOC)
- `internal/gen/golang/codegen_expr_app.go` - Update lookups (~10 LOC)
- `internal/gen/golang/codegen_match.go` - Update lookups (~10 LOC)
- `internal/gen/golang/codegen_ops.go` - Update lookups (~10 LOC)
- `internal/gen/golang/codegen_decl.go` - Update lookups (~15 LOC)

**Total estimated LOC:** ~75

## Examples

### Example 1: Same-Named Constructors in Different ADTs

**AILANG Source:**
```ailang
module sim/protocol

type DrawCmd =
  | Clear
  | Viewport(name: string, x: int, y: int, w: int, h: int)
  | Text(content: string)

export DrawCmd, Clear, Viewport, Text
```

```ailang
module sim/window

type WindowType =
  | Fullscreen
  | Viewport  -- Nullary, no fields!
  | Windowed(w: int, h: int)

export WindowType, Fullscreen, Viewport, Windowed
```

```ailang
module sim/interior

import sim/protocol (Viewport)  -- Only import DrawCmd.Viewport

pure func makeViewport() -> DrawCmd =
    Viewport("main", 0, 0, 800, 600)  -- Should use DrawCmd.Viewport
```

**Before (broken):**
```go
// Wrong constructor called - WindowType.Viewport (nullary) instead of DrawCmd.Viewport (5 args)
return NewWindowTypeViewport("main", int64(0), int64(0), int64(800), int64(600))
// Error: too many arguments
```

**After (fixed):**
```go
// Correct constructor called based on import context
return NewDrawCmdViewport("main", int64(0), int64(0), int64(800), int64(600))
```

## Success Criteria

- [ ] Two ADTs with same-named constructors compile without errors
- [ ] Import context determines which constructor is used
- [ ] Unimported constructors are not accessible
- [ ] All existing codegen tests pass
- [ ] `make test` passes
- [ ] stapledons_voyage sim/*.ail compiles successfully

## Testing Strategy

**Unit tests:**
- Test ADT constructor registration with qualified names
- Test lookup with same-named constructors in different types
- Test that unimported constructors are not resolved

**Integration tests:**
- Compile two modules with same-named constructors
- Verify correct constructor is called based on imports
- Verify generated Go compiles and runs correctly

**Manual testing:**
- Compile stapledons_voyage and verify it works

## Non-Goals

**Not in this feature:**
- Cross-module constructor re-exporting - out of scope
- Constructor overloading (same name, different arities in same ADT) - not supported

## Timeline

**Single session** (~3 hours):
- Phase 1: 1 hour - Registry key changes
- Phase 2: 1.5 hours - Lookup updates
- Phase 3: 30 min - Testing

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Breaking existing codegen tests | Medium | Run test suite after each phase |
| Type context not available at lookup | High | Fall back to unqualified lookup if needed |
| Performance impact from longer keys | Low | Map lookup still O(1) |

## Related Documents

**Implemented (may inform design):**
- [M-DX11: Named ADT Constructor Fields](../../implemented/v0_5_3/m-dx11-named-adt-fields.md) - ADT codegen architecture
- [M-DX22: ADT Slice Converters](../../implemented/v0_5_4/m-dx22-adt-slice-converters.md) - ADT type handling

## References

- [Design Axioms](/docs/references/axioms) - The 12 non-negotiable principles
- stapledons_voyage bug report (agent message)

## Future Work

- Consider warning when same-named constructors exist across ADTs (for clarity)
- Consider import aliasing for constructors: `import sim/protocol (Viewport as DrawViewport)`

---

**Document created**: 2025-12-21
**Last updated**: 2025-12-21
