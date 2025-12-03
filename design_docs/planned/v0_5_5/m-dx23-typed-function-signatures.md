# M-DX23: Typed Function Signatures

**Status**: Planned
**Target**: v0.5.5
**Priority**: P1 (Medium)
**Estimated**: 6 hours (revised from 4 - infrastructure changes needed)
**Dependencies**: None
**Source**: DX feedback from `stapledons_voyage` agent

## AI-First Alignment Check

| Principle | Impact | Score | Notes |
|-----------|--------|-------|-------|
| Reduce Syntactic Noise | + | +1 | Removes type assertions at every call site |
| Preserve Semantic Clarity | + | +1 | Explicit return types in signatures |
| Increase Determinism | 0 | 0 | No change to determinism |
| Lower Token Cost | + | +1 | Fewer casts = fewer tokens |
| **Net Score** | | **+3** | **Decision: Move forward** |

## Problem Statement

Generated AILANG functions return `interface{}`, requiring type assertions at every call site:

```go
// Current: InitWorld and Step return interface{}
var world interface{} = InitWorld(seed)
w := world.(*World)  // Required cast at every use

for tick := 0; tick < 100; tick++ {
    result := Step(w)
    w = result.(*World)  // Cast again
}
```

**Current State:**
- All generated functions return `interface{}`
- Type assertions required at every call site
- Boilerplate accumulates in game loops
- Type info exists in `CoreTypeInfo` but isn't passed to codegen

**Impact:**
- ~2 extra lines per function call (cast + error handling)
- Potential runtime panics if types mismatch
- AI must generate and maintain cast boilerplate

## Goals

**Primary Goal:** Generate typed function signatures with concrete return types.

**Success Metrics:**
- [ ] InitWorld returns `*World` not `interface{}`
- [ ] Step returns `*World` not `interface{}`
- [ ] Zero type assertions needed at call sites
- [ ] Type safety enforced at compile time

## Solution Design

### Overview

Pipe `CoreTypeInfo` from the type checker through to the Go code generator, then use it to generate typed function signatures.

### Architecture

**Current data flow:**
```
pipeline.go → typeChecker.CoreTI → (discarded)
                                     ↓
compile.go  → gen.New() → Generate() → interface{} signatures
```

**Proposed data flow:**
```
pipeline.go → typeChecker.CoreTI → Artifacts.CoreTI
                                         ↓
compile.go  → gen.New() → SetCoreTypeInfo(cti) → Generate() → TYPED signatures
```

### Key Components

1. **CoreTypeInfo** (`internal/types/typeinfo.go:46`)
   - Maps Core NodeID (`uint64`) → `Type`
   - Populated during type checking
   - Available via `typeChecker.CoreTI`

2. **Type Mapping** (new)
   - `AilangTypeToGo(t types.Type) string`
   - Converts AILANG types to Go type strings
   - Handles: primitives, records, ADTs, functions

3. **Generator Enhancement** (`internal/gen/golang/codegen.go`)
   - Add `coreTypeInfo types.CoreTypeInfo` field
   - Add `SetCoreTypeInfo(cti types.CoreTypeInfo)` method
   - Use in `generateFuncFromLambda`

### Implementation Plan

**Phase 0: Infrastructure** (~1.5 hours)
- [ ] Add `CoreTI types.CoreTypeInfo` to `Artifacts` struct in `internal/pipeline/pipeline.go`
- [ ] Populate `Artifacts.CoreTI` in `pipeline_single.go` after type checking
- [ ] Populate `Artifacts.CoreTI` in `pipeline_module.go` for multi-file compilation
- [ ] Update compile.go to access `result.Artifacts.CoreTI`

**Phase 1: Type Mapping** (~1.5 hours)
- [ ] Create `internal/gen/golang/type_mapper.go`
- [ ] Implement `AilangTypeToGo(t types.Type, recordTypes map[string]bool) string`
- [ ] Handle primitives: `Int` → `int64`, `Float` → `float64`, `String` → `string`, `Bool` → `bool`
- [ ] Handle records: Named → `*RecordName`
- [ ] Handle ADTs: Named → `*ADTName`
- [ ] Handle lists: `[T]` → `[]T` (recursive)
- [ ] Handle functions: Return `interface{}` (polymorphic functions stay dynamic)
- [ ] Handle type variables: Return `interface{}` (unresolved generics)

**Phase 2: Generator Changes** (~2 hours)
- [ ] Add `coreTypeInfo types.CoreTypeInfo` field to Generator
- [ ] Add `SetCoreTypeInfo(cti types.CoreTypeInfo)` method
- [ ] Import `github.com/sunholo/ailang/internal/types` in codegen.go
- [ ] Modify `generateFuncFromLambda`:
  - Look up Lambda's type from CoreTI using `g.coreTypeInfo.GetForExpr(lam)`
  - Extract parameter types from function type
  - Extract return type from function type
  - Generate typed parameter list
  - Generate typed return type
  - Keep `interface{}` as fallback for missing type info

**Phase 3: Testing** (~1 hour)
- [ ] Add unit tests for type mapping
- [ ] Add codegen test for typed function signatures
- [ ] Test with step.ail example
- [ ] Verify stapledons_voyage generates typed signatures
- [ ] Verify backward compatibility (functions without type info)

### Files to Modify/Create

**New files:**
- `internal/gen/golang/type_mapper.go` - Type conversion logic (~80 LOC)

**Modified files:**
- `internal/pipeline/pipeline.go` - Add CoreTI to Artifacts (~3 LOC)
- `internal/pipeline/pipeline_single.go` - Populate CoreTI (~5 LOC)
- `internal/pipeline/pipeline_module.go` - Populate CoreTI (~5 LOC)
- `internal/gen/golang/codegen.go` - Add CoreTI field and setter (~15 LOC)
- `internal/gen/golang/codegen_decl.go` - Use CoreTI in generateFuncFromLambda (~30 LOC)
- `cmd/ailang/compile.go` - Pass CoreTI to generator (~5 LOC)
- `internal/gen/golang/codegen_test.go` - Add tests (~40 LOC)

**Total:** ~180 LOC

## Examples

### Example 1: Game Loop

**Before (interface{} everywhere):**
```go
func main() {
    world := InitWorld(int64(12345))
    w := world.(*World)  // Required cast

    for tick := 0; tick < 100; tick++ {
        result := Step(w)
        w = result.(*World)  // Cast every iteration
    }
}
```

**After (typed signatures):**
```go
func main() {
    w := InitWorld(int64(12345))  // Returns *World directly

    for tick := 0; tick < 100; tick++ {
        w = Step(w)  // No casts needed!
    }
}
```

### Example 2: Type Mapping

| AILANG Type | Go Type |
|-------------|---------|
| `int` | `int64` |
| `float` | `float64` |
| `string` | `string` |
| `bool` | `bool` |
| `World` (record) | `*World` |
| `Selection` (ADT) | `*Selection` |
| `[int]` | `[]int64` |
| `[World]` | `[]*World` |
| `int -> int` | `interface{}` (polymorphic fallback) |
| `α` (type var) | `interface{}` (unresolved generic) |

### Example 3: Generated Signatures

**AILANG source:**
```ailang
pure func step(world: World) -> World =
  { world | tick: world.tick + 1 }

let initWorld: World = { tick: 0, name: "game" }
```

**Generated Go (current):**
```go
func Step(world interface{}) interface{} { ... }
var initWorld interface{} = &World{...}
```

**Generated Go (proposed):**
```go
func Step(world *World) *World {
    return RecordUpdate(world, map[string]interface{}{
        "tick": AddInt(FieldGet(world, "tick"), int64(1)),
    }).(*World)  // Cast internal helper, but signature is typed
}
var initWorld *World = &World{...}
```

## Success Criteria

- [ ] Functions with typed annotations generate typed Go signatures
- [ ] Type assertions only needed for runtime helpers (RecordUpdate, FieldGet)
- [ ] stapledons_voyage game loop works without casts at call sites
- [ ] All existing tests pass
- [ ] Backward compatibility: functions without type info use interface{}
- [ ] Variables with typed record values get explicit type annotation

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Breaking existing code | High | Keep interface{} for untyped/polymorphic funcs |
| CoreTI not always available | Med | Graceful fallback to interface{} |
| Complex generic types | Med | Start with monomorphic, expand later |
| Internal casts add overhead | Low | Go compiler optimizes these away |

## Non-Goals

**Not in this feature:**
- Polymorphic function generation (future M-POLY work)
- Automatic interface boxing/unboxing
- Higher-kinded type support
- Effect types in signatures

## Testing Strategy

**Unit tests:**
- Type mapping for all primitive types
- Type mapping for records and ADTs
- Type mapping for lists (nested)
- Fallback to interface{} for type variables

**Integration tests:**
- Generate step.ail with typed signatures
- Compile and run generated code
- Verify no casts needed at call sites

**Manual testing:**
- Build stapledons_voyage with new codegen
- Verify game loop compiles without manual casts

## References

- [typeinfo.go](../../../internal/types/typeinfo.go) - CoreTypeInfo definition
- [pipeline_single.go](../../../internal/pipeline/pipeline_single.go) - Where CoreTI is populated
- [codegen_decl.go](../../../internal/gen/golang/codegen_decl.go) - Current function generation
- M-DX16/17/18/19/20/21/22 - Related typed struct work
- stapledons_voyage feedback message (2025-12-03)

## Future Work

- Support for polymorphic functions (`[T] -> T`)
- Effect-typed function signatures (`! {IO}`)
- Optimized calling conventions for hot paths
- Generic type instantiation in generated code

---

**Document created**: 2025-12-03
**Last updated**: 2025-12-03
