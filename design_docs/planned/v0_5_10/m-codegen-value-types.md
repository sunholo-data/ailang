# M-CODEGEN-VALUE-TYPES: Size-Based Pointer vs Value Strategy

**Status**: Planned
**Target**: v0.5.10
**Priority**: P1 - High (DX improvement, performance)
**Estimated**: 6-8 hours (revised up for stability guarantees)
**Dependencies**: M-CODEGEN-POINTER-RETURN-TYPES (v0.5.9)

## AI-First Alignment Check

**Score this feature against AILANG's core principles:**

| Principle | Impact | Score | Notes |
|-----------|--------|-------|-------|
| Reduce Syntactic Noise | 0 | 0 | No syntax change |
| Preserve Semantic Clarity | + | +1 | Clear value vs pointer semantics |
| Increase Determinism | ++ | +2 | Single source of truth, stable ABI |
| Lower Token Cost | 0 | 0 | No impact on token cost |
| **Net Score** | | **+3** | **Decision: Move forward** |

## Problem Statement

**Feedback Source**: stapledons_voyage project (DX feedback, Dec 9, 2025)

### Current State (v0.5.9)

After M-CODEGEN-POINTER-RETURN-TYPES, ALL user-defined types are now pointers:

```go
// Function signatures - all pointers
func Step(world *World, input *FrameInput) interface{}

// Struct fields - all pointers
type FrameOutput struct {
    Camera *Camera      // small 3-field struct, forced to heap
    Mouse  *MouseState  // small 3-field struct, forced to heap
    Draw   []*DrawCmd   // slice of pointers (OK)
}

// Record literals - all pointers
return &Coord{X: x, Y: y}  // tiny 2-field struct, allocated on heap
```

### Problems

1. **Performance**: Small, frequently-used types like `Coord`, `Camera`, `MouseState` are heap-allocated when they could be stack values
2. **GC Pressure**: 60 FPS game loop creates thousands of tiny pointer allocations
3. **Cache Locality**: Pointer chasing hurts CPU cache efficiency
4. **Breaking Changes**: Existing Go code expects values, not pointers (~15 files affected)

### Why We Made Everything Pointers

The v0.5.9 pointer changes fixed critical type assertion bugs:
- `_impl` functions return `interface{}`
- Record literals are always `&Type{...}` (pointers)
- Type assertions need to match: `.(Type)` vs `.(*Type)`

Making everything pointers created **consistency**. But it's over-aggressive for small types.

## Goals

**Primary Goal:** Generate optimal Go types based on record characteristics with stable, predictable ABI.

**Success Metrics:**
- [ ] Small leaf records (primitives only, ≤threshold) generate as VALUES
- [ ] Large/nested/recursive records generate as POINTERS
- [ ] Type assertions remain correct across interface{} boundaries
- [ ] No runtime panics from type mismatches
- [ ] ABI stability: category persisted in metadata
- [ ] Single source of truth for type categories

---

## Design Review Feedback (Dec 9, 2025)

### What's Solid

1. **Unified type-level decision**: "For each AILANG record, decide once: value vs pointer" is the right granularity. It keeps `_impl` return types, type assertions, and codegen deterministic.

2. **Simple heuristic**: Size-based threshold with `#fields + nested + ADT slices` is predictable for both humans and LLMs. No hidden Go-compiler-ish cost model.

3. **ADTs special-cased as pointers**: Boxing sum variants behind pointers is sane - variant size differences, recursive constructors, and pattern matches all get simpler.

4. **CLI escape hatch**: `--value-threshold` is vital for discovering pathological cases and per-project tuning.

### Main Hazards to Address

#### Hazard 1: ABI Instability ("add one field, everything changes")

**Problem**: Current heuristic means adding a single field to `Camera` could flip:
- All Go signatures from `Camera` to `*Camera`
- All interface assertions
- All existing Go helper code

**Mitigations (MUST IMPLEMENT)**:

1. **Persist category in compiled metadata** (`.aili` or equivalent):
   ```json
   { "name": "sim/protocol.Coord", "go_repr": "value" }
   ```
   Even if heuristic would flip today, existing artifacts retain old decision until explicit regeneration.

2. **Per-project threshold** in `ailang.toml`:
   ```toml
   [codegen]
   value-threshold = 4
   ```
   Prevents accidental rebuild with different flag silently changing everything.

3. **Document as ABI-affecting**: "Changing field count may flip value vs pointer and is treated as an ABI-breaking change."

#### Hazard 2: Type-Assertion Correctness at Interface Boundary

**Critical Invariant**: For each AILANG type T, every place that:
- stores T into an `interface{}`, and
- asserts T out of an `interface{}`

must consult the **same TypeCategory table** and never diverge.

**Solution**: Single source of truth function:
```go
func GoReprForAilangType(name string) (goName string, isPointer bool)
```

**All codegen paths must use this**:
- [ ] Function params and returns (`ailangTypeToGo`)
- [ ] Record literals (`generateTypedRecord`)
- [ ] Struct fields (`mapASTType`)
- [ ] Pattern-match helpers
- [ ] ADT constructors involving records
- [ ] `_impl` wrappers and interface{} boxing/unboxing

#### Hazard 3: Recursive / Mutually-Recursive Records

**Problem**: "Contains nested records → POINTER" is not strong enough.

**Actual requirement**: If a record is **directly or indirectly recursive**, it MUST be pointer.

Examples that must be pointers regardless of field count:
```ailang
type Node = { next: Node }           -- direct recursion
type A = { b: B }
type B = { a: A }                    -- mutual recursion
type NPC = { inventory: [Item] }
type Item = { owner: NPC }           -- via list/ADT
```

**Solution**: Either:
1. Run `AnalyzeRecordType` after type-graph info and mark "is recursive" (use existing cycle machinery)
2. **Conservative approach (recommended)**: "If any field is of a user-defined record/ADT type, treat as pointer. Reserve value only for leaf, non-recursive, primitive-only records."

Given performance concerns are mostly about very small leaf types (`Coord`, `Camera`, `MouseState`), the conservative approach is acceptable and much simpler.

---

## Refined Solution Design

### Two-Step Heuristic

Rather than mixing "size-ish" and "structure-ish" in one blob, split into:

**Step 1: Eligibility for Value Representation (structural constraints)**
```
ELIGIBLE FOR VALUE if and only if:
- No recursive or mutually recursive references
- ALL fields are scalar primitives (int/float/bool/string)
- No ADT fields (sum types)
- No nested record fields
```

Only if Step 1 passes, proceed to Step 2.

**Step 2: Threshold Check**
```
if FieldCount ≤ threshold → Value
else → Pointer
```

**Result**: Value types are truly **leaf, primitive-only, non-recursive "POD-ish" records**. Everything with structural complexity is pointer, regardless of size.

### Type Category as First-Class Concept

Store category in `RecordTypeInfo`, not scattered maps:

```go
type TypeCategory int

const (
    TypeCategoryValue   TypeCategory = iota // Leaf, primitive-only, ≤threshold
    TypeCategoryPointer                      // Large, nested, recursive, or ADT
)

type RecordTypeInfo struct {
    Name        string            // Go type name (PascalCase)
    Category    TypeCategory      // Value or Pointer
    IsLeaf      bool              // All fields are primitives
    IsRecursive bool              // Directly or indirectly recursive
    FieldCount  int               // Number of fields
    Fields      []string          // Go field names
    FieldTypes  map[string]string // Go field types
}
```

### Single Source of Truth

```go
// GoReprForType returns the Go representation for an AILANG type.
// This is THE authority for value vs pointer decisions.
// All codegen paths MUST call this function.
func (g *Generator) GoReprForType(typeName string) (goType string, isPointer bool) {
    if info, ok := g.recordTypes[typeName]; ok {
        if info.Category == TypeCategoryValue {
            return info.Name, false  // Coord
        }
        return info.Name, true  // *World
    }
    // Default: unknown types are pointers (safe fallback)
    return ToGoTypeName(typeName), true
}
```

### Architecture

```
┌─────────────────────────────────────────────────────────────┐
│              Phase 1: Eligibility Analysis                   │
├─────────────────────────────────────────────────────────────┤
│  For each record type:                                       │
│  1. Check if all fields are primitives (int/float/bool/str) │
│  2. Check for recursive references (use type graph)          │
│  3. Check for ADT fields                                     │
│                                                              │
│  Output: IsLeaf, IsRecursive                                 │
└─────────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────┐
│              Phase 2: Category Assignment                    │
├─────────────────────────────────────────────────────────────┤
│  if IsLeaf && !IsRecursive && FieldCount ≤ threshold:       │
│      Category = Value                                        │
│  else:                                                       │
│      Category = Pointer                                      │
│                                                              │
│  Store in RecordTypeInfo.Category                            │
└─────────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────┐
│              Phase 3: Unified Codegen                        │
├─────────────────────────────────────────────────────────────┤
│  ALL codegen paths call GoReprForType():                     │
│  - Function signatures                                       │
│  - Struct field types                                        │
│  - Record literals                                           │
│  - Type assertions                                           │
│  - _impl wrappers                                            │
└─────────────────────────────────────────────────────────────┘
```

### Checklist: All Codegen Paths Using TypeCategory

| Location | Function/File | Before | After |
|----------|---------------|--------|-------|
| Function params | `ailangTypeToGo` (compile.go) | `*TypeName` always | Consult `GoReprForType` |
| Function returns | `ailangTypeToGo` (compile.go) | `*TypeName` always | Consult `GoReprForType` |
| Struct fields | `mapASTType` (adt.go) | `*TypeName` always | Consult `GoReprForType` |
| Record literals | `generateTypedRecord` (codegen_ops.go) | `&Type{...}` always | `Type{}` or `&Type{}` |
| Type assertions | `generateTypedRecord` (codegen_ops.go) | `.(*Type)` always | `.(Type)` or `.(*Type)` |
| Pattern match | Various | `*Type` cases | Consult category |
| ADT with records | Various | Pointer always | Pointer always (ADTs stay pointer) |
| `_impl` wrapper | codegen_decl.go | `interface{}` | Match category for assertions |

### CLI Flag Semantics

```bash
# Default threshold (4 fields)
ailang compile --emit-go src/

# Custom threshold
ailang compile --emit-go --value-threshold 3 src/

# All pointers (v0.5.9 behavior, migration path)
ailang compile --emit-go --value-threshold 0 src/

# Negative values treated as 0 (with warning)
ailang compile --emit-go --value-threshold -1 src/
# Warning: negative threshold treated as 0 (all pointers)
```

**Critical**: Threshold is per-build, not per-module. Don't allow different modules in same build to use different thresholds.

### ABI Stability: Metadata Persistence

Store category decisions in compiled interface metadata:

```json
// gen/types.json (generated alongside Go code)
{
  "version": "0.5.10",
  "threshold": 4,
  "types": {
    "Coord": { "category": "value", "fields": 2 },
    "Camera": { "category": "value", "fields": 3 },
    "World": { "category": "pointer", "fields": 13 },
    "FrameOutput": { "category": "pointer", "fields": 4, "reason": "has_adt_slice" }
  }
}
```

On regeneration:
1. Load existing `types.json` if present
2. Compare new analysis with persisted categories
3. Warn if any type would flip: `Warning: Camera would change from value to pointer (field count increased to 5). Use --force-regenerate to accept ABI change.`
4. Require explicit `--force-regenerate` flag to accept ABI-breaking changes

---

## Semantic Implications

Beyond performance, pointer vs value is a **semantic choice**:

| Representation | Semantics | Use Case |
|----------------|-----------|----------|
| **Value** | Copying is cheap; mutation is local | Small, immutable data: `Coord`, `Camera` |
| **Pointer** | Shared references; mutation affects all holders | Large state graphs: `World`, `BridgeState` |

The heuristic aligns: "small leaf types get value semantics; big graphs get pointer semantics." This is not just a performance hack - it's a meaningful semantic distinction worth documenting.

---

## Examples

### Example 1: Leaf Record (Value Type)

**AILANG:**
```ailang
type Coord = { x: int, y: int }
```

**Analysis:**
- Fields: 2 (≤4 threshold)
- All primitives: YES
- Recursive: NO
- **Category: VALUE**

**Generated Go:**
```go
type Coord struct {
    X int64
    Y int64
}

func GetPosition() Coord { ... }       // VALUE return
type Entity struct { Pos Coord }       // VALUE field
return Coord{X: 10, Y: 20}             // VALUE literal
result.(Coord)                          // VALUE assertion
```

### Example 2: Large Record (Pointer Type)

**AILANG:**
```ailang
type World = {
    width: int, height: int,
    npcs: [NPC], tiles: [[Tile]],
    camera: Camera, player: Entity,
    clock: int, seed: int,
    -- ... 13+ fields
}
```

**Analysis:**
- Fields: 13 (>4 threshold)
- Has nested types: YES (NPC, Tile, Camera, Entity)
- **Category: POINTER** (fails both eligibility and threshold)

**Generated Go:**
```go
func InitWorld(seed int64) *World { ... }  // POINTER return
return &World{...}                          // POINTER literal
result.(*World)                             // POINTER assertion
```

### Example 3: Small But Non-Leaf (Pointer Type)

**AILANG:**
```ailang
type Entity = { pos: Coord, vel: Coord }  -- only 2 fields but nested
```

**Analysis:**
- Fields: 2 (≤4 threshold)
- All primitives: NO (Coord is user-defined)
- **Category: POINTER** (fails eligibility despite small size)

**Generated Go:**
```go
type Entity struct {
    Pos Coord   // Coord is value (leaf)
    Vel Coord   // Coord is value (leaf)
}

func GetEntity() *Entity { ... }  // POINTER return (Entity is not leaf)
return &Entity{...}               // POINTER literal
```

### Example 4: Recursive Record (Always Pointer)

**AILANG:**
```ailang
type Node = { value: int, next: Node }  -- recursive
```

**Analysis:**
- Fields: 2 (≤4 threshold)
- Recursive: YES
- **Category: POINTER** (recursive types must be pointers)

---

## Success Criteria

- [ ] Two-step heuristic: eligibility then threshold
- [ ] `RecordTypeInfo.Category` is single source of truth
- [ ] `GoReprForType()` used by ALL codegen paths
- [ ] Leaf primitive-only records (≤threshold) generate as values
- [ ] Non-leaf, recursive, ADT records generate as pointers
- [ ] Type assertions match generated types exactly
- [ ] `--value-threshold` flag works
- [ ] ABI metadata persisted in `types.json`
- [ ] Warning on category flip without `--force-regenerate`
- [ ] All existing tests pass
- [ ] stapledons_voyage compiles correctly

---

## Implementation Plan

**Phase 1: Type Analyzer** (~2 hours)
- [ ] Create `type_analyzer.go` with two-step analysis
- [ ] Implement `IsLeafRecord()` (all fields primitive)
- [ ] Integrate with existing cycle detection for `IsRecursive`
- [ ] Add `TypeCategory` to `RecordTypeInfo`
- [ ] Unit tests for analysis heuristics

**Phase 2: Single Source of Truth** (~1.5 hours)
- [ ] Implement `GoReprForType()` method on Generator
- [ ] Refactor all codegen paths to use it (see checklist above)
- [ ] Remove scattered `typeCategories` maps

**Phase 3: Compile-Time Integration** (~1.5 hours)
- [ ] Modify compile.go to analyze types before registration
- [ ] Pass category to `RegisterRecordType`
- [ ] Update `ailangTypeToGo` to delegate to `GoReprForType`

**Phase 4: Codegen Updates** (~1.5 hours)
- [ ] Update `generateTypedRecord` for value vs pointer literals
- [ ] Update struct field generation
- [ ] Update type assertions
- [ ] Verify `_impl` wrapper consistency

**Phase 5: CLI Flag & Metadata** (~1 hour)
- [ ] Add `--value-threshold` flag
- [ ] Generate `types.json` metadata file
- [ ] Load existing metadata and compare
- [ ] Add `--force-regenerate` flag
- [ ] Documentation

**Phase 6: Testing** (~0.5 hours)
- [ ] Integration tests for value/pointer generation
- [ ] Tests for ABI stability warnings
- [ ] stapledons_voyage verification

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Type assertion mismatches | High | Single source of truth `GoReprForType()` |
| ABI instability | High | Persist categories in metadata, warn on flip |
| Recursive types as values | High | Always pointer for recursive (checked early) |
| Scattered codegen paths | Medium | Checklist + integration tests |
| Edge cases | Medium | Conservative: non-leaf = pointer |

---

## Future Work (v0.6.0+)

1. **Typed Return Tuples**: Generate `StepResult` structs instead of `[]interface{}`
2. **Explicit Annotations**: `@value type Coord = {...}` for manual override
3. **Value-Based ADTs**: For simple enum-ish sum types (no payload variance)
4. **Escape Analysis Hints**: Let AILANG hints guide Go's escape analysis

---

## Open Questions

1. **Should we recursively check for value-eligible nested types?**
   - E.g., `Entity { pos: Coord }` where Coord is leaf - could Entity be value too?
   - Current answer: NO (conservative). Revisit if needed.

2. **How to handle slices of primitives?**
   - `[int]` is fine in value types (Go slices are reference types)
   - Current: Allowed in leaf records

3. **Plugging cycle detection into analyzer?**
   - Can reuse existing `types.TypeEnv.HasCycle()` machinery
   - Need to call after type graph is built

---

## References

- DX feedback from stapledons_voyage (Dec 9, 2025)
- Design review feedback (Dec 9, 2025)
- M-CODEGEN-POINTER-RETURN-TYPES (v0.5.9 prerequisite)
- Go escape analysis: https://go.dev/doc/faq#stack_or_heap

---

**Document created**: 2025-12-09
**Last updated**: 2025-12-09
**Design review incorporated**: 2025-12-09
