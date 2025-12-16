# M-CODEGEN-UNIFIED-SLICE-CONVERTERS: Complete Typed Slice Conversion

**Status**: Planned
**Target**: v0.5.11
**Priority**: P0 - Critical (Blocks stapledons_voyage)
**Estimated**: 3 hours
**Dependencies**: None
**Parent**: M-DX26 (Typed Wrapper Architecture)

## Problem Statement

The typed slice conversion system has been built incrementally with gaps:

| Slice Type | Converter? | `getSliceConversion`? | Status |
|------------|------------|----------------------|--------|
| `[]int64` | ✅ | ✅ | Working |
| `[]string` | ✅ | ✅ | Working |
| `[]bool` | ✅ | ✅ | Working |
| `[]float64` | ❌ **NO** | ❌ falls through | **BUG** |
| `[]*ADTType` | ✅ (M-DX22) | ⚠️ only if in `adtSliceTypes` | Partial |
| `[]*RecordType` | ❌ **NO** | ❌ falls through | **BUG** |

### The Symptom

When a function returns `[SolarPlanet]` or `[float]`, the wrapper does a direct type assertion that panics:

```go
// Generated (BROKEN):
func GetSolarPlanets() []*SolarPlanet {
    return getSolarPlanets_impl().([]*SolarPlanet)  // PANIC!
}

func GetScores() []float64 {
    return getScores_impl().([]float64)  // PANIC!
}
```

### Root Cause: Fragmented Design

The converter system evolved incrementally without a unified approach:

1. **M-DX12**: Added `adtSliceTypes` map for ADT slices
2. **M-DX22**: Extended converter generation to all ADTs via `adtConstructors`
3. **But**: `getSliceConversion()` still only checks `adtSliceTypes`
4. **Records**: Never had converters
5. **`[]float64`**: Simply forgotten

The check in `getSliceConversion()`:
```go
if strings.HasPrefix(goType, "[]*") {
    typeName := goType[3:]
    if g.adtSliceTypes[typeName] {  // <-- Only checks ADT types!
        return "ConvertTo" + typeName + "Slice"
    }
}
return ""  // Falls through for float64, records, etc.
```

---

## Solution: Unified Slice Converter Registry

### Design Principle

**One registry, one generator, one lookup function.**

```go
// sliceConverterTypes tracks ALL types needing slice converters
// Keys are the Go element types (e.g., "int64", "float64", "*SolarPlanet")
sliceConverterTypes map[string]SliceConverterKind

type SliceConverterKind int
const (
    SliceConverterPrimitive SliceConverterKind = iota  // int64, float64, string, bool
    SliceConverterPointer                               // *ADTType, *RecordType
)
```

### Implementation Plan

#### Fix 1: Add `[]float64` converter (~10 LOC)

In `codegen_runtime_collections.go`, add to `writeRuntimeSliceConverters()`:

```go
g.writef("// ConvertToFloat64Slice converts []interface{} to []float64.\n")
g.writef("func ConvertToFloat64Slice(v interface{}) []float64 {\n")
// ... same pattern as ConvertToInt64Slice
```

In `getSliceConversion()`:
```go
case "[]float64":
    return "ConvertToFloat64Slice"
```

#### Fix 2: Unify pointer type checking (~15 LOC)

Update `getSliceConversion()` to check ALL pointer types:

```go
if strings.HasPrefix(goType, "[]*") {
    typeName := goType[3:]

    // Check ADT types (from adtConstructors, not just adtSliceTypes)
    if _, ok := g.adtConstructors[typeName]; ok {
        return "ConvertTo" + typeName + "Slice"
    }
    // Also check if typeName is an ADT type name in any constructor
    for _, info := range g.adtConstructors {
        if info.TypeName == typeName {
            return "ConvertTo" + typeName + "Slice"
        }
    }

    // Check record types
    if _, ok := g.recordTypes[typeName]; ok {
        return "ConvertTo" + typeName + "Slice"
    }

    // Legacy: check adtSliceTypes for backward compatibility
    if g.adtSliceTypes[typeName] {
        return "ConvertTo" + typeName + "Slice"
    }
}
```

#### Fix 3: Generate record slice converters (~40 LOC)

Add `writeRecordSliceConverters()` in `codegen_runtime_collections.go`:

```go
func (g *Generator) writeRecordSliceConverters() {
    for typeName := range g.recordTypes {
        goTypeName := ToGoTypeName(typeName)
        funcName := "ConvertTo" + goTypeName + "Slice"

        g.writef("// %s converts []interface{} to []*%s.\n", funcName, goTypeName)
        g.writef("func %s(v interface{}) []*%s {\n", funcName, goTypeName)
        g.indent++

        g.writef("if v == nil { return nil }\n")
        g.writef("src, ok := v.([]interface{})\n")
        g.writef("if !ok { panic(fmt.Sprintf(\"%s: expected []interface{}, got %%T\", v)) }\n", funcName)
        g.writef("if len(src) == 0 { return []*%s{} }\n", goTypeName)
        g.writef("out := make([]*%s, len(src))\n", goTypeName)
        g.writef("for i, e := range src {\n")
        g.indent++
        g.writef("out[i] = e.(*%s)\n", goTypeName)
        g.indent--
        g.writef("}\n")
        g.writef("return out\n")

        g.indent--
        g.writef("}\n\n")
    }
}
```

#### Fix 4: Call converters in Generate() (~2 LOC)

```go
g.writeRuntimeSliceConverters()  // int64, string, bool, float64
g.writeADTSliceConverters()       // []*ADTType
g.writeRecordSliceConverters()    // []*RecordType  <-- ADD
```

---

## Files to Modify

| File | Changes |
|------|---------|
| `internal/gen/golang/codegen.go` | +15 LOC - Update `getSliceConversion()` |
| `internal/gen/golang/codegen_runtime_collections.go` | +60 LOC - Add `ConvertToFloat64Slice`, `writeRecordSliceConverters()` |
| `internal/gen/golang/codegen_datastructures_test.go` | +40 LOC - Tests |

**Total:** ~115 LOC

---

## Test Cases

```go
// Test 1: Float slice conversion
func TestFloatSliceConversion(t *testing.T) {
    g := NewGenerator(...)
    conv := g.getSliceConversion("[]float64")
    if conv != "ConvertToFloat64Slice" {
        t.Errorf("expected ConvertToFloat64Slice, got %s", conv)
    }
}

// Test 2: Record slice conversion
func TestRecordSliceConversion(t *testing.T) {
    g := NewGenerator(...)
    g.RegisterRecordType("Planet", ...)
    conv := g.getSliceConversion("[]*Planet")
    if conv != "ConvertToPlanetSlice" {
        t.Errorf("expected ConvertToPlanetSlice, got %s", conv)
    }
}

// Test 3: ADT slice via adtConstructors (not adtSliceTypes)
func TestADTSliceViaConstructors(t *testing.T) {
    g := NewGenerator(...)
    g.RegisterADTConstructor("Star", "G", 0)  // Registers via adtConstructors
    // Note: NOT calling RegisterADTSliceType
    conv := g.getSliceConversion("[]*Star")
    if conv != "ConvertToStarSlice" {
        t.Errorf("expected ConvertToStarSlice, got %s", conv)
    }
}
```

---

## Why This Keeps Happening

The converter system grew organically:

1. **v0.5.3 (M-DX12)**: Added ADT slice converters, tracked in `adtSliceTypes`
2. **v0.5.4 (M-DX22)**: Extended generation to all ADTs via `adtConstructors`, but forgot to update `getSliceConversion` check
3. **v0.5.7**: Added `[]bool` converter
4. **v0.5.8**: Various fixes but no unified approach
5. **Now**: Records and `[]float64` are missing

**Pattern**: Each fix adds a new special case instead of unifying the system.

### Prevention

After this fix, the system should have:

1. **Single source of truth**: `getSliceConversion()` checks all type registries
2. **Complete primitive coverage**: int64, float64, string, bool
3. **Automatic pointer type coverage**: Any `[]*T` where T is in `adtConstructors` OR `recordTypes`
4. **Tests**: Verify each type category has working conversion

---

## Acceptance Criteria

- [ ] `ConvertToFloat64Slice` function generated
- [ ] `getSliceConversion("[]float64")` returns converter name
- [ ] `ConvertTo<RecordType>Slice` functions generated for all record types
- [ ] `getSliceConversion("[]*RecordType")` returns converter name
- [ ] `getSliceConversion("[]*ADTType")` works even if not in `adtSliceTypes`
- [ ] stapledons_voyage compiles and runs without panic
- [ ] All codegen tests pass

---

## References

- M-DX26: [Typed Wrapper Architecture](../../implemented/v0_5_5/m-dx26-typed-wrapper-architecture.md)
- M-DX12: [Typed ADT Slices](../../implemented/v0_5_3/m-dx12-typed-adt-slices.md)
- M-DX22: [ADT Slice Converters](../../implemented/v0_5_4/m-dx22-adt-slice-converters.md)

---

**Document created**: 2025-12-13
**Bug report**: stapledons_voyage message `msg_20251213_204115_a3b27208`
