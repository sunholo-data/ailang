# M-CODEGEN-RECORD-UPDATE-FIELDS: Record Update Syntax Missing Fields

**Status**: RESOLVED (No Changes Needed)
**Target**: v0.5.11
**Priority**: Closed
**Estimated**: N/A
**Dependencies**: None
**GitHub Issue**: Bug report from stapledons_voyage

## Resolution

**Date**: 2025-12-12

**Root Cause**: The bug report was based on stale generated code in `/tmp/stap_debug/`. Investigation revealed:

1. The AILANG source (`sim/celestial.ail` lines 224-235) does NOT use record update syntax `{base | field: value}`
2. Instead, it uses explicit record literals with field accesses:
   ```ailang
   {
       id: planet.id,
       name: planet.name,
       ...
       currentAngle: newAngle,
       ...
   }
   ```
3. The actual generated code in `sim_gen/celestial.go` is correct and includes all 10 fields with proper type assertions
4. `stapledons_voyage` builds successfully

**Action Taken**: None - no code changes needed. The `/tmp` copy was from an older compilation.

## AI-First Alignment Check

**Score this feature against AILANG's core principles:**

| Principle | Impact | Score | Notes |
|-----------|--------|-------|-------|
| Reduce Syntactic Noise | 0 | 0 | Bug fix, no syntax change |
| Preserve Semantic Clarity | + | +1 | Record update should preserve all fields |
| Increase Determinism | + | +1 | Correct codegen produces consistent output |
| Lower Token Cost | 0 | 0 | No token impact |
| **Net Score** | | **+2** | **Decision: Move forward (critical bug fix)** |

**Decision rule:** Net score > +1 → Move forward | ≤ 0 → Reject or redesign

## Problem Statement

Record update syntax `{ record | field: newValue }` generates Go code that only includes the updated fields, missing all other fields from the original record.

**Current State:**
- AILANG code: `{ planet | currentAngle: newAngle }` should copy all planet fields and update currentAngle
- Generated Go: Only includes 2 of 10 fields (Name, Radius)
- Variables for other fields are declared but never used
- Go compiler fails with "declared and not used" errors

**Example from stapledons_voyage `celestial.go:250-274`:**

```go
// Generated (WRONG):
func stepPlanetOrbit_impl(...) interface{} {
    // ... compute newAngle ...
    var newAngle interface{} = ...  // Computed but UNUSED!
    var tmp37 interface{} = ...      // id - UNUSED!
    var tmp38 interface{} = ...      // name
    var tmp39 interface{} = ...      // planetType - UNUSED!
    var tmp40 interface{} = ...      // orbitDistance - UNUSED!
    var tmp41 interface{} = ...      // radius
    // ... more unused variables ...

    // Only 2 of 10 fields emitted!
    return &Planet{Name: tmp38.(string), Radius: tmp41.(float64)}
}
```

**Expected:**
```go
// Should be:
return &Planet{
    Id: tmp37.(int64),
    Name: tmp38.(string),
    PlanetType: tmp39.(*PlanetType),
    OrbitDistance: tmp40.(float64),
    OrbitalPeriod: tmp42.(float64),
    CurrentAngle: newAngle.(float64),  // The updated field
    HasRings: tmp44.(bool),
    RingColor: tmp45.(*RGBA),
    AtmosphereColor: tmp46.(*RGBA),
    Radius: tmp41.(float64),
}
```

**Impact:**
- **Build failure**: Go compiler rejects unused variables
- **Blocking**: stapledons_voyage sprint cannot proceed
- **Data loss**: If build succeeded, updated records would lose most fields

## Goals

**Primary Goal:** Fix record update codegen to emit all fields from the original record plus the updated field.

**Success Metrics:**
- `{ planet | currentAngle: newAngle }` generates all 10 Planet fields
- No unused variable errors in generated Go code
- stapledons_voyage builds successfully
- All existing tests pass

## Root Cause Analysis

### Likely Location

The bug is likely in `internal/gen/golang/codegen_ops.go` or `codegen_expr_let.go` where record update expressions are generated.

**Core IR for Record Update:**
- `core.RecordUpdate` has fields: `Record` (base), `Field` (name), `Value` (new value)
- Codegen needs to:
  1. Get all fields from the base record's type
  2. Copy each field from base record
  3. Override the updated field with new value

**Hypothesis:**
The codegen is only emitting fields that are explicitly mentioned in the update expression, not copying all fields from the base record type.

## Solution Design

### Overview

1. Find the record update codegen in `codegen_ops.go` or `codegen_expr_let.go`
2. When generating a RecordUpdate:
   - Look up the record's type from CoreTypeInfo
   - Get all field names from the TRecord type
   - For each field:
     - If it's the updated field: use the new value
     - Otherwise: access the field from the base record

### Implementation Plan

**Phase 1: Investigation** (~30 min)
- [ ] Find RecordUpdate handling in codegen
- [ ] Trace what fields are being emitted
- [ ] Verify hypothesis about missing field copying

**Phase 2: Fix** (~1-2 hours)
- [ ] Modify codegen to emit all fields from record type
- [ ] Handle field access from base record
- [ ] Ensure updated field uses new value

**Phase 3: Verification** (~1 hour)
- [ ] Run existing tests
- [ ] Compile stapledons_voyage
- [ ] Verify generated code is correct

### Files to Modify

**Modified files:**
- `internal/gen/golang/codegen_ops.go` - Record update handling (~+30 LOC)
- OR `internal/gen/golang/codegen_expr_let.go` - If handled in Let bindings

## Examples

### Example 1: Planet Orbit Update

**AILANG Source:**
```ailang
export pure func stepPlanetOrbit(planet: Planet, dt: float) -> Planet {
    let newAngle = planet.currentAngle + (2.0 * PI() / planet.orbitalPeriod) * dt
    { planet | currentAngle: newAngle }
}
```

**Before (Wrong):**
```go
return &Planet{Name: tmp38.(string), Radius: tmp41.(float64)}
// Missing: Id, PlanetType, OrbitDistance, OrbitalPeriod, CurrentAngle, HasRings, RingColor, AtmosphereColor
```

**After (Correct):**
```go
return &Planet{
    Id:              FieldGet(planet, "id").(int64),
    Name:            FieldGet(planet, "name").(string),
    PlanetType:      FieldGet(planet, "planetType").(*PlanetType),
    OrbitDistance:   FieldGet(planet, "orbitDistance").(float64),
    OrbitalPeriod:   FieldGet(planet, "orbitalPeriod").(float64),
    CurrentAngle:    newAngle.(float64),  // Updated field
    HasRings:        FieldGet(planet, "hasRings").(bool),
    RingColor:       FieldGet(planet, "ringColor").(*RGBA),
    AtmosphereColor: FieldGet(planet, "atmosphereColor").(*RGBA),
    Radius:          FieldGet(planet, "radius").(float64),
}
```

### Example 2: Simple Record Update

**AILANG Source:**
```ailang
type Point = {x: int, y: int}
let p = {x: 1, y: 2}
let p2 = {p | x: 10}  -- Should be {x: 10, y: 2}
```

**Expected Go:**
```go
p2 := &Point{
    X: int64(10),              // Updated
    Y: FieldGet(p, "y").(int64), // Copied from original
}
```

## Success Criteria

- [ ] `{ record | field: value }` emits all fields from record type
- [ ] Updated field uses new value
- [ ] Non-updated fields access base record
- [ ] No unused variable warnings in generated code
- [ ] stapledons_voyage `celestial.go` compiles without errors
- [ ] All existing tests pass
- [ ] Planet orbit stepping preserves all planet data

## Testing Strategy

**Unit tests:**
- Test record update with 1 field type (existing tests)
- Test record update with multi-field type (new test)
- Test that all fields are present in output

**Integration tests:**
- Compile stapledons_voyage
- Run generated Go code
- Verify planet data is preserved through orbit stepping

**Manual testing:**
- `DEBUG_CODEGEN=1` to inspect generated code
- Compare before/after for stepPlanetOrbit

## Non-Goals

**Not in this feature:**
- **Nested record updates**: `{ a | b.c: value }` - more complex syntax
- **Multiple field updates in one expression**: Already supported via chaining

## Timeline

**Single Session** (~2-4 hours):
- Investigation: 30 min
- Implementation: 1-2 hours
- Testing: 1 hour
- Documentation: 30 min

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| RecordUpdate might be in unexpected location | Low | Search codebase for RecordUpdate handling |
| May need type info not currently available | Medium | CoreTypeInfo should have TRecord with all fields |
| Fix might break other record operations | Medium | Run full test suite before/after |

## References

- Bug report from stapledons_voyage (message ID: msg_20251212_185921_f61c4073)
- `internal/gen/golang/codegen_ops.go` - Record codegen
- `internal/core/core.go` - RecordUpdate definition
- Related: M-TYPENAME-NESTED-PROPAGATION (just completed)

## Future Work

- **Nested field updates**: `{ a | b.c: value }` syntax
- **Spread syntax**: `{ ...a, x: 1, y: 2 }` for multiple updates

---

**Document created**: 2025-12-12
**Last updated**: 2025-12-12
