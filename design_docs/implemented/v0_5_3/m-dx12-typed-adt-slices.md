# M-DX12: Typed World Boundary Marshalling

## Status
**Planned** - Ready for v0.5.x implementation

**Last Updated**: 2025-12-03 (incorporated DX feedback from stapledons_voyage)

## Problem Statement

When AILANG generates Go code for profile-exported types (World, FrameOutput, etc.), list fields containing ADTs become `interface{}` instead of typed slices. This leaks `interface{}` into the "profile surface" where host code (game engines, etc.) expects precise types.

**AILANG source:**
```ailang
type FrameOutput = {
    draw: [DrawCmd],
    sounds: [int],
    debug: [string],
    camera: Camera
}
```

**Current generated Go:**
```go
type FrameOutput struct {
    Draw   interface{}  // DX footgun - host code must type assert
    Sounds []int64      // Correct!
    Debug  []string     // Correct!
    Camera Camera
}
```

**Desired:**
```go
type FrameOutput struct {
    Draw   []*DrawCmd  // Typed slice - clean host API
    Sounds []int64
    Debug  []string
    Camera Camera
}
```

## Strategic Framing

This is really about the friction between two worlds:

1. **AILANG semantic world**: Lists are homogeneous, but at runtime represented as `[]eval.Value` → effectively `[]interface{}` inside the evaluator.

2. **Go host world**: Profile-facing types should be precise (`[]*DrawCmd`), even if internally we shuttle through `[]interface{}`.

**M-DX12 should be framed as:**

> "Add typed world boundary marshalling so host types can be precise, even if evaluator & IR use `[]interface{}`."

This is more general than "ADT slices" and dovetails with execution profiles (SimProfile, GameProfile).

## Source

DX feedback from `stapledons_voyage` agent (2025-12-03).

## Root Cause Analysis

The issue is in `adt.go` lines 273-280:

```go
case *ast.ListType:
    elemType := g.mapASTType(typ.Element)
    // For ADT/user-defined element types, use interface{} to avoid runtime type mismatch
    // AILANG runtime passes []interface{}, not []*ADTType
    if isUserDefinedType(elemType) {
        return "interface{}"
    }
    return fmt.Sprintf("[]%s", elemType)
```

**The fundamental issue:** AILANG's evaluator uses `[]interface{}` internally for all lists. When passing data to generated Go code, there's a type mismatch:
- Evaluator produces: `[]interface{}{*DrawCmd{...}, *DrawCmd{...}}`
- Go expects: `[]*DrawCmd{...}`

## Scope

### In Scope
- Record fields of type `[ADT]`
- ADT constructors that take `[ADT]` parameters (e.g., `PatternPatrol([Direction])`)
- Any profile-exported type (World, FrameOutput, Request, Response) containing `[ADT]` fields

### Out of Scope
- Internal lists inside compiled funcs; they stay `[]interface{}` for now
- Evaluator internal representation changes (deferred to v0.7+)

## Solution Options

### Option A: World Boundary Marshalling (Recommended)

Generate conversion helpers and use them **in the world/record marshalling code**, so user-facing Go types are fully typed and conversions are invisible to game code.

**Key refinement**: Tie conversion to specific record fields / ADT fields in the generated marshalling code, rather than sprinkling generic helpers into user code.

**Implementation shape:**

```go
// Marshal from eval.Value to Go-facing struct (generated in marshalling layer)
func toFrameOutput(v eval.Value) FrameOutput {
    rec := v.(*eval.RecordValue) // or whatever your record repr is
    return FrameOutput{
        Draw:   convertToDrawCmdSlice(rec.Fields["draw"]),
        Sounds: convertToIntSlice(rec.Fields["sounds"]),
        Debug:  convertToStringSlice(rec.Fields["debug"]),
        Camera: toCamera(rec.Fields["camera"]),
    }
}
```

**Converter implementation (fail-fast):**

```go
func convertToDrawCmdSlice(v interface{}) []*DrawCmd {
    if v == nil {
        return nil
    }
    src, ok := v.([]interface{})
    if !ok {
        panic(fmt.Sprintf("convertToDrawCmdSlice: expected []interface{}, got %T", v))
    }
    out := make([]*DrawCmd, len(src))
    for i, e := range src {
        dc, ok := e.(*DrawCmd)
        if !ok {
            panic(fmt.Sprintf("convertToDrawCmdSlice: element %d: expected *DrawCmd, got %T", i, e))
        }
        out[i] = dc
    }
    return out
}
```

**Pros:**
- Localizes risk and complexity to one marshalling layer
- Keeps public API surface purely typed
- One choke point to optimize later if evaluator representation changes
- No evaluator changes needed
- Game code never sees `interface{}`

**Cons:**
- Runtime conversion overhead (acceptable for profile boundaries)
- Need to generate per-ADT type converters
- Converters generated alongside types

### Option B: Change Evaluator Internals
Modify the AILANG evaluator to use typed slices instead of `[]interface{}`.

**Status**: Explicitly parked as "Possible v0.7+ performance pass"

**Blockers:**
- Lists are everywhere (pattern-matching, stdlib list ops, tests)
- Changing from `[]eval.Value` to typed slices is basically a partial re-write
- Strategy needed for polymorphic lists `[a]` – impossible to represent as compile-time typed Go slice without monomorphization

**Pros:**
- Clean solution
- No conversion needed

**Cons:**
- Major breaking change
- Affects all of internal/eval
- Requires extensive testing
- May break existing code

### Option C: Go Generics

Use Go generics to DRY up the converters:

```go
func toSlice[T any](v interface{}) []T {
    if v == nil { return nil }
    src, ok := v.([]interface{})
    if !ok {
        panic(fmt.Sprintf("toSlice: expected []interface{}, got %T", v))
    }
    out := make([]T, len(src))
    for i, e := range src {
        out[i] = e.(T)
    }
    return out
}

// Usage: toSlice[*DrawCmd](v)
```

**Trade-offs:**
- **Pros**: DRYs up codegen; less LOC in generated code
- **Cons**: Requires go1.18+; still pays runtime conversion; generics don't give free cast

**Recommendation**: Treat generics as a code-size/maintainability optimization, not a semantic change. Start with monomorphic helpers per ADT kind and refactor to generic helper later if it gets noisy.

## Design Decisions

### Decision 1: Pointer vs Value ADTs

**Invariant to document:**

> All ADT constructors are represented as pointers to struct types in generated Go (`*DrawCmd`, `*Camera`), and lists are `[]*DrawCmd` etc.

This ensures:
- Clean interop with `interface{}`
- Avoid copying large structs
- Consistent nil representation

**Future consideration**: Single-constructor no-sum ADTs might become value types, but that's a separate design decision.

### Decision 2: Failure Mode

**Conversion helpers will panic on type mismatch** because this indicates a **compiler bug**, not user error.

- In dev/debug builds: panic with clear message showing expected vs actual type
- In production builds: keep the panic (bug = crash is fine for compiler invariant violations)

```go
panic(fmt.Sprintf("convertToDrawCmdSlice: element %d: expected *DrawCmd, got %T", i, e))
```

### Decision 3: Zero vs Nil Semantics

| AILANG | Go |
|--------|-----|
| Empty list `[]` | Empty slice `[]T{}` |
| Null (if present) | `nil` |

**Converter behavior:**
- Pass through `nil` unchanged
- Treat empty `[]interface{}{}` as empty slice `[]T{}`

## Alignment with Existing Roadmap

This feature aligns with:

1. **Slice conversions in M-GAME-B Phase 2** (for constructor arguments)
2. **SimProfile / GameProfile** using World/FrameOutput structs as host-facing contract
3. **ADT → Go type mapping rules** (pointer vs value, record vs enum)

**Profile-level requirement for SimProfile:**
- Host-facing world and output types must not contain `interface{}` fields unless marked "opaque"
- List-of-ADT fields like `[DrawCmd]` must map to `[]*DrawCmd`

**Single boundary concept:**
All evaluator ↔ Go conversions go through a single "world marshaller" layer that handles:
- ADT constructors / tags
- Record field types
- Lists vs arrays vs primitives

## Recommendation

**v0.5.x Implementation (Option A)**:
1. Implement world/record marshalling helpers
2. Cover ADT list fields + ADT lists in ADT constructors using a common mechanism
3. Make stapledon profiles fully typed (no public `interface{}`)

**v0.6+**:
- If profiling shows conversions are hot, consider:
  - Small optimizations (avoid conversion if already `[]*DrawCmd`)
  - Revisiting internal evaluator representation

**v0.7+**:
- Option B (evaluator changes) only if performance becomes critical

## Estimated Effort

| Approach | LOC | Time | Risk |
|----------|-----|------|------|
| Option A (Recommended) | ~250 | 4-6h | Low |
| Option B | ~500+ | 2-3 days | High |
| Option C (Generics) | ~150 | 3-4h | Medium |

## Dependencies

- **M-DX11 (Named ADT Fields)** - COMPLETED
- Need to understand full evaluator→codegen data flow

## Files to Modify

1. `internal/gen/golang/adt.go` - Change type mapping for `[ADT]` fields
2. `internal/gen/golang/codegen_runtime.go` - Add per-ADT conversion helpers
3. `cmd/ailang/compile.go` - Generate marshalling layer for profile types
4. `internal/gen/golang/marshal.go` (NEW) - World boundary marshalling code

## Acceptance Criteria

1. [ ] `[ADT]` fields in profile types generate as typed slices (`[]*DrawCmd`)
2. [ ] Conversion happens automatically in marshalling layer
3. [ ] Game code never sees `interface{}` in public API
4. [ ] Converters panic with clear message on type mismatch
5. [ ] Empty lists map to empty slices (not nil)
6. [ ] Nil maps to nil
7. [ ] Runtime performance acceptable at profile boundaries
8. [ ] Backwards compatible with existing code
9. [ ] All tests pass

## Implementation Plan

### Phase 1: Converter Generation (~2h)
- [ ] Add `generateADTSliceConverter()` to codegen
- [ ] Generate one converter per ADT type used in lists
- [ ] Include fail-fast panic with type info

### Phase 2: Type Mapping Update (~2h)
- [ ] Change `adt.go` to emit typed slices for `[ADT]` fields
- [ ] Only for profile-exported types initially
- [ ] Keep `interface{}` for internal compiled code

### Phase 3: Marshalling Layer (~2h)
- [ ] Create `marshal.go` for world boundary conversions
- [ ] Generate `toFrameOutput()` style functions
- [ ] Wire up converters in record field marshalling

### Phase 4: Testing & Validation (~1h)
- [ ] Test with stapledon's FrameOutput type
- [ ] Verify no public `interface{}` in generated API
- [ ] Verify panic messages are helpful
