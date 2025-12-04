# M-DX26: Typed Wrapper Architecture

**Status:** Planned
**Priority:** High
**Estimated LOC:** ~200
**Dependencies:** M-DX23 (Typed Signatures), M-DX25 (Typed Let Bindings - partial)

---

## Problem Statement

M-DX25 attempted to make function bodies fully typed by adding type assertions and conversions at every expression boundary. This created a "partially typed world" fighting with an `interface{}` runtime, resulting in cascading compile errors:

```
./funcs.go:374:11: cannot use []interface{}{} as []int64 value in return
./funcs.go:380:23: cannot use CallFunc(f, tmp1) (interface{}) as int64
./funcs.go:388:15: cannot use Cons(tmp2, tmp4) as []int64 value in return
./funcs.go:409:20: operator % not defined on idx (interface{})
./funcs.go:430:34: tmp8 (float64) is not an interface
```

**Root cause:** The runtime helpers (`Cons`, `CallFunc`, `NegInt`, `FieldGet`, etc.) all operate in `interface{}` world, but typed function signatures expect concrete types. Every expression boundary becomes a typing war.

---

## Solution: Typed Wrapper Layer

Instead of making the entire function body typed in one shot, we:

1. **Keep the old `interface{}` codegen as an internal implementation** (`_impl` functions)
2. **Generate thin typed wrappers** that bridge between typed Go and the `interface{}` runtime

### Example

For an AILANG function:
```ailang
export pure func add(a: int, b: int) -> int { a + b }
```

Generate TWO Go functions:

```go
// Internal: stays in interface{} world (existing codegen)
func add_impl(a interface{}, b interface{}) interface{} {
    return AddInt(a, b)  // Uses runtime helpers, all interface{}
}

// External: typed wrapper for Go consumers
func Add(a int64, b int64) int64 {
    return add_impl(a, b).(int64)
}
```

For slice-returning functions:
```go
func buildPath_impl(...) interface{} {
    return Cons(head, tail)  // Returns []interface{}
}

func BuildPath(...) []int64 {
    return ToInt64Slice(buildPath_impl(...))
}
```

---

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    Typed Go World                       │
│  (Game code, external consumers, typed APIs)            │
│                                                         │
│   Step(world *World, input FrameInput) FrameOutput      │
│   BuildPath(start Coord, end Coord) []int64             │
└─────────────────────┬───────────────────────────────────┘
                      │ Type assertions / Slice converters
                      ▼
┌─────────────────────────────────────────────────────────┐
│               interface{} VM World                       │
│  (All runtime helpers, Cons, CallFunc, FieldGet, etc.)  │
│                                                         │
│   step_impl(world interface{}, input interface{})       │
│   buildPath_impl(start interface{}, end interface{})    │
└─────────────────────────────────────────────────────────┘
```

**Key insight:** The boundary between worlds is well-defined and narrow (function entry/exit points only), making type conversions tractable.

---

## Implementation Plan

### Phase 1: Generate Dual Functions (~100 LOC)

**File:** `internal/gen/golang/codegen_decl.go`

Modify `generateFuncFromLambda` to generate both versions:

```go
func (g *Generator) generateFuncFromLambda(name string, lam *core.Lambda, exported bool) error {
    // 1. Generate _impl version (interface{} everywhere)
    if err := g.generateImplFunc(name, lam); err != nil {
        return err
    }

    // 2. Generate typed wrapper (if exported or has typed signature)
    paramTypes, returnType := g.getTypedSignature(lam)
    if exported || len(paramTypes) > 0 || returnType != "" {
        return g.generateTypedWrapper(name, lam, paramTypes, returnType, exported)
    }
    return nil
}
```

**`generateImplFunc`:**
- All parameters: `interface{}`
- Return type: `interface{}`
- No CoreTypeInfo lookups
- No type assertions in body
- Uses existing expression codegen unchanged

**`generateTypedWrapper`:**
- Typed parameters from CoreTypeInfo
- Typed return from CoreTypeInfo
- Body is just: `return impl_func(args...)` with conversions

### Phase 2: Slice Conversion Helpers (~50 LOC)

**File:** `internal/gen/golang/codegen_runtime.go`

Add to runtime helpers:

```go
func ToInt64Slice(v interface{}) []int64 {
    if v == nil {
        return nil
    }
    src := v.([]interface{})
    out := make([]int64, len(src))
    for i, e := range src {
        out[i] = e.(int64)
    }
    return out
}

func ToStringSlice(v interface{}) []string { ... }
func ToDrawCmdSlice(v interface{}) []*DrawCmd { ... }
// etc. for each concrete slice type used
```

### Phase 3: Auto-Generate Slice Converters (~50 LOC)

For ADT types like `[]*DrawCmd`, auto-generate converters based on registered types:

```go
// In types.go or codegen.go
func (g *Generator) generateSliceConverters() {
    for typeName := range g.adtSliceTypes {
        g.writef("func To%sSlice(v interface{}) []*%s {\n", typeName, typeName)
        g.writef("    if v == nil { return nil }\n")
        g.writef("    src := v.([]interface{})\n")
        g.writef("    out := make([]*%s, len(src))\n", typeName)
        g.writef("    for i, e := range src { out[i] = e.(*%s) }\n", typeName)
        g.writef("    return out\n")
        g.writef("}\n\n")
    }
}
```

---

## Wrapper Generation Logic

```go
func (g *Generator) generateTypedWrapper(name string, lam *core.Lambda,
    paramTypes []GoType, returnType GoType, exported bool) error {

    funcName := ToGoFuncName(name, exported)
    implName := ToGoVarName(name) + "_impl"

    // Build typed parameter list
    var params []string
    var callArgs []string
    for i, p := range lam.Params {
        pType := "interface{}"
        if i < len(paramTypes) {
            pType = string(paramTypes[i])
        }
        params = append(params, fmt.Sprintf("%s %s", ToGoVarName(p), pType))
        callArgs = append(callArgs, ToGoVarName(p))
    }

    retType := "interface{}"
    if returnType != "" {
        retType = string(returnType)
    }

    // Generate wrapper
    g.writef("func %s(%s) %s {\n", funcName, strings.Join(params, ", "), retType)
    g.indent++

    // Call impl and convert result
    if retType == "interface{}" {
        g.writef("return %s(%s)\n", implName, strings.Join(callArgs, ", "))
    } else if strings.HasPrefix(retType, "[]") {
        // Slice return - use converter
        sliceConv := g.getSliceConverter(retType)
        g.writef("return %s(%s(%s))\n", sliceConv, implName, strings.Join(callArgs, ", "))
    } else {
        // Scalar return - type assertion
        g.writef("return %s(%s).(%s)\n", implName, strings.Join(callArgs, ", "), retType)
    }

    g.indent--
    g.writef("}\n\n")
    return nil
}
```

---

## Error Mapping

| Current Error | Wrapper Solution |
|---------------|------------------|
| `[]interface{}{}` as `[]int64` | `_impl` returns `[]interface{}`, wrapper converts with `ToInt64Slice` |
| `CallFunc(f, tmp1)` as `int64` | `_impl` uses `CallFunc` internally, wrapper asserts result |
| `Cons(...)` as `[]int64` | `_impl` uses `Cons`, wrapper converts slice |
| `operator % not defined on idx` | `_impl` uses `ModInt(idx, width)`, no operators on `interface{}` |
| `tmp8 (float64) is not interface` | `_impl` uses `interface{}` everywhere, no mismatch |

---

## Benefits

1. **Immediate unblocking:** stapledons_voyage compiles today
2. **Clear separation:** `interface{}` runtime vs typed Go API
3. **Incremental migration:** Can collapse layers function-by-function later
4. **Minimal changes:** Most codegen unchanged, just wrapping
5. **Preserves M-DX25 work:** All type mapping infrastructure is reused

## Drawbacks

1. **Performance overhead:** Boxing/unboxing at every function boundary - **NEGLIGIBLE** (see analysis below)
2. **Code size:** Two functions per AILANG function
3. **Delayed full typing:** Body internals stay untyped

---

## Performance Analysis

**The boxing/unboxing overhead is negligible for our use case:**

1. **Function call overhead dominates:** The cost of a function call (stack setup, return) is ~10-50ns. Boxing an int64 to interface{} adds ~1-2ns.

2. **Game loop context:** At 60 FPS, each frame has 16.6ms budget. Even with 1000 function calls per frame, the boxing overhead would be ~2μs - about 0.01% of frame budget.

3. **Go's escape analysis:** The compiler often optimizes interface{} boxing when values don't escape, eliminating heap allocation.

4. **Slice conversion is O(n) but rare:** Slice conversions (`ToInt64Slice`) only happen at function boundaries returning slices, not on every list operation.

**Conclusion:** Performance optimization is not a priority. If profiling later shows hotspots, we can selectively collapse layers for those specific functions. This is a "measure first, optimize later" situation.

---

## Relationship to M-DX25

**M-DX26 builds ON TOP of M-DX25, not replacing it.**

### What We KEEP from M-DX25

| Component | Why We Keep It |
|-----------|----------------|
| List type mapping (`TApp("List", T)` → `[]T`) | Wrappers need correct return types |
| ADT pointer types (`*Direction`) | Wrappers need correct param/return types |
| TypeMapper infrastructure | Used to generate wrapper signatures |
| `getTypedSignature()` | Tells us what the wrapper signature should be |
| Record type inference | Helps generate correct struct converters |
| `funcParamTypes` / `funcReturnTypes` maps | Used for wrapper generation |

### What We DISABLE in `_impl` Functions

| Component | Why We Disable It |
|-----------|-------------------|
| Type assertions in body (`needsReturnTypeAssertion`) | `_impl` returns `interface{}` |
| Slice conversions at returns | Wrapper handles conversion |
| CoreTypeInfo lookups for intermediate variables | Everything is `interface{}` |
| Typed local variable declarations | All locals are `interface{}` |

### Architecture Diagram

```
┌─────────────────────────────────────────────────┐
│  M-DX23/25: Type Mapping Infrastructure (KEEP) │
│  - TypeMapper, getTypedSignature               │
│  - List → []T, ADT → *Type                     │
│  - funcParamTypes, funcReturnTypes             │
└───────────────────────┬─────────────────────────┘
                        │ Used by
        ┌───────────────┴───────────────┐
        ▼                               ▼
┌───────────────────┐    ┌───────────────────────────┐
│ _impl functions   │    │ Typed wrappers            │
│ (interface{} only)│◄───│ (use type mapping for     │
│ NO assertions     │    │  signatures + conversions)│
│ M-DX25 DISABLED   │    │  M-DX25 ENABLED           │
└───────────────────┘    └───────────────────────────┘
```

---

## Future: Collapsing the Layers (M-DX27)

Once wrapper approach is stable, we can incrementally move logic from `_impl` into typed wrappers:

1. For functions where ALL expressions are statically typed:
   - Generate typed body directly (no `_impl`)
   - Use native operators instead of runtime helpers

2. For functions with mixed typing:
   - Keep wrapper pattern
   - Measure performance impact

This is the path to "fully typed bodies" but done safely and incrementally.

---

## Acceptance Criteria

1. [ ] All AILANG functions generate `name_impl` + `Name` pair
2. [ ] `_impl` uses interface{} everywhere (no type assertions in body)
3. [ ] Typed wrapper handles scalar and slice conversions
4. [ ] Slice converters auto-generated for ADT types
5. [ ] stapledons_voyage compiles successfully
6. [ ] All existing codegen tests pass
7. [ ] Performance benchmarks captured for future comparison

---

## Files to Modify

| File | Changes |
|------|---------|
| `internal/gen/golang/codegen_decl.go` | +80 LOC - `generateImplFunc`, `generateTypedWrapper` |
| `internal/gen/golang/codegen_runtime.go` | +50 LOC - Slice converter helpers |
| `internal/gen/golang/codegen.go` | +20 LOC - Slice converter generation |
| Tests | +50 LOC - Wrapper generation tests |

**Total:** ~200 LOC

---

## Related

- M-DX23: Typed Function Signatures (completed)
- M-DX24: Typed Function Bodies (superseded by this approach)
- M-DX25: Typed Let Bindings (partial - wrapper makes this tractable)
- M-DX27: Layer Collapsing (future optimization)
