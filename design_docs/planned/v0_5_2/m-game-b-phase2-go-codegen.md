# M-GAME-B Phase 2: Complete Go Codegen for Game Modules

**Status**: Planned
**Target**: v0.5.2
**Priority**: P1 - Medium
**Estimated**: 5 hours
**Dependencies**: M-GAME-B Phase 1 (completed in v0.5.2)

## AI-First Alignment Check

| Principle | Impact | Score | Notes |
|-----------|--------|-------|-------|
| Reduce Syntactic Noise | 0 | 0 | Backend codegen, no syntax changes |
| Preserve Semantic Clarity | + | +1 | Generated Go matches AILANG semantics exactly |
| Increase Determinism | + | +1 | Consistent codegen for all module configurations |
| Lower Token Cost | 0 | 0 | No change to source token count |
| **Net Score** | | **+2** | **Decision: Move forward** |

## Strategic Context

### Scope Declaration

> **The Go backend in v0.5.x is a Tier-1 target for:**
> - Pure or effect-bounded game/sim modules
> - With a restricted set of stdlib features (prelude, io, array, debug, ai)
> - Compiled via `ailang compile --emit-go` with a documented contract
>
> **Everything outside that profile is "best effort, may fail, interpreter is canonical."**

### Why Go Codegen is Worth It

We're past the "toy" phase with:
- ADTs, records, effects, lists, arrays implemented
- Cross-module codegen working
- Real external consumer (stapledon) depending on it

Backing out would mean:
- Only interpreter execution → worse perf, worse tooling
- Losing the sales pitch: "deterministic AI language with escape hatch into boring Go"
- Throwing away hard-won lowering insights

### Control Mechanisms

1. **Go is a projection of AILANG semantics** - Interpreter remains the spec
2. **Shared runtime package** - One place for helpers, not per-module generation
3. **Regression testing** - Every Go test also runs through interpreter to verify equality

## Problem Statement

The Go codegen for AILANG game modules works for simple cases but fails on three patterns:

**Issue 1: Slice Type Assertions**

When an ADT constructor expects a typed slice, current codegen generates invalid Go:
```go
// Generated (WRONG):
NewMovementPatternPatternPatrol(path.([]map[string]interface{}))
// Runtime panic: interface conversion failed
```

Go slices are invariant - `[]interface{}` cannot be directly asserted to `[]T`.

**Issue 2: Missing Runtime Helpers**

Builtins like `show`, `concat_string`, and `log` compile but generated Go doesn't include implementations.

**Issue 3: Cross-Module Function Generation**

Functions defined in imported modules are not generated in funcs.go.

## Solution Design

### Overview

Three targeted fixes with architectural guardrails:

1. **Slice Type Conversion**: Generate type-aware conversion loops from AILANG type info
2. **Runtime Package**: Shared `github.com/sunholo/ailang/runtime` package for helpers
3. **Reachable Functions**: Only generate functions reachable from entry module

### Architecture

#### Component 1: Slice Type Conversion (`codegen_expr.go`)

**Key insight**: Don't hard-code "map" conversions; generate from AILANG type.

The conversion generator should:
1. Look at the field's internal type in Core type system
2. Emit the relevant conversion based on element type

```go
// For [int] → []int64
func convertToInt64Slice(v any) []int64 {
    if v == nil { return nil }
    src := v.([]any)
    dst := make([]int64, len(src))
    for i, elem := range src {
        dst[i] = elem.(int64)
    }
    return dst
}

// For [{x: int, y: int}] → []map[string]any
func convertToRecordSlice(v any) []map[string]any {
    if v == nil { return nil }
    src := v.([]any)
    dst := make([]map[string]any, len(src))
    for i, elem := range src {
        dst[i] = elem.(map[string]any)
    }
    return dst
}
```

**Panic semantics**: If AILANG would fail (wrong type), Go panics too. No gentle errors needed - matches interpreter behavior.

#### Component 2: Runtime Package (`runtime/`)

**Create a shared Go runtime package** instead of generating helpers per module:

```
github.com/sunholo/ailang/runtime/
├── show.go       // Show(v any) string
├── string.go     // ConcatString(a, b any) string
├── io.go         // Log(msg any) any
└── convert.go    // Type conversion helpers
```

Generated code imports this:
```go
import "github.com/sunholo/ailang/runtime"

runtime.Show(v)
runtime.ConcatString(a, b)
```

**Advantages:**
- One place to evolve helpers
- Consistent semantics across all generated code
- "Go backend is real and supported" feels solid
- Types are consistent (AILANG `int` = Go `int64`)

#### Component 3: Reachable Function Generation

**Only generate what's actually reachable**, not every function in every imported module.

Reachability analysis:
1. Start from exported functions used by entry module
2. Walk call graph within Core IR → collect referenced function IDs
3. Generate code for that closure only

**Module prefixing contract** (deterministic, hard to change later):
- `sim/protocol.directionToString` → `SimProtocol_DirectionToString`
- Transform: `ModulePath_FunctionName` with path segments PascalCased

```go
// From sim/protocol (reachable)
func SimProtocol_DirectionToString(d any) any {
    // match implementation
}

// From sim/step (entry module)
func LogDirection(d any) any {
    return runtime.Log(SimProtocol_DirectionToString(d))
}
```

### Implementation Plan

**Phase 1: Runtime Package** (~1.5 hours)
- [ ] Create `runtime/` package structure
- [ ] Implement `Show(v any) string` with type switch
- [ ] Implement `ConcatString(a, b any) string`
- [ ] Implement `Log(msg any) any`
- [ ] Add basic type conversion helpers
- [ ] Unit tests for each helper

**Phase 2: Slice Type Conversion** (~1.5 hours)
- [ ] Add `getSliceElementType()` to extract element type from AILANG type
- [ ] Generate type-aware conversion functions based on element type
- [ ] Modify `generateApp()` to detect slice fields and apply conversion
- [ ] Support: `[]int64`, `[]string`, `[]map[string]any`, `[]*ADTType`
- [ ] Tests for each slice type

**Phase 3: Reachable Cross-Module Functions** (~2 hours)
- [ ] Implement call graph walker in Core IR
- [ ] Collect reachable function set from entry module
- [ ] Generate only reachable functions with module prefixes
- [ ] Update function calls to use prefixed names
- [ ] Test with stapledon's multi-module setup

### Files to Modify/Create

**New files:**
- `runtime/show.go` - Show implementation (~30 LOC)
- `runtime/string.go` - String helpers (~15 LOC)
- `runtime/io.go` - IO helpers (~20 LOC)
- `runtime/convert.go` - Type conversion helpers (~40 LOC)
- `runtime/runtime_test.go` - Tests (~100 LOC)

**Modified files:**
- `internal/gen/golang/codegen_expr.go` - Slice conversion logic (~50 LOC)
- `internal/gen/golang/codegen.go` - Reachability + prefixing (~80 LOC)
- `cmd/ailang/compile.go` - Pass reachable set to codegen (~20 LOC)

## Examples

### Example 1: Slice Type Conversion (Type-Aware)

**AILANG source:**
```ailang
type MovementPattern
  = PatternPatrol([{x: int, y: int}])

pure func patrol(points: [{x: int, y: int}]) -> MovementPattern {
  PatternPatrol(points)
}
```

**Generated Go (type-aware conversion):**
```go
// Generated based on AILANG type [{x: int, y: int}]
func convertToRecordSlice(v any) []map[string]any {
    if v == nil { return nil }
    src := v.([]any)
    dst := make([]map[string]any, len(src))
    for i, elem := range src {
        dst[i] = elem.(map[string]any)
    }
    return dst
}

NewMovementPatternPatternPatrol(convertToRecordSlice(points))
```

### Example 2: Runtime Package Import

**AILANG source:**
```ailang
import std/prelude (show, log)

func main() -> ! {IO} () {
  log(show(42) ++ " is the answer")
}
```

**Generated Go:**
```go
import "github.com/sunholo/ailang/runtime"

func Main() any {
    return runtime.Log(runtime.ConcatString(runtime.Show(int64(42)), " is the answer"))
}
```

**runtime/show.go:**
```go
package runtime

import "fmt"

// Show converts any AILANG value to its string representation.
// Semantics defined here are canonical for Go backend.
func Show(v any) string {
    switch x := v.(type) {
    case string:
        return x
    case int64:
        return fmt.Sprintf("%d", x)
    case float64:
        return fmt.Sprintf("%g", x)
    case bool:
        if x { return "true" }
        return "false"
    default:
        return fmt.Sprintf("%v", x)
    }
}
```

### Example 3: Reachable Cross-Module Functions

**sim/protocol.ail:**
```ailang
module sim/protocol

pure func directionToString(d: Direction) -> string { ... }
pure func unusedHelper() -> int { 42 }  -- NOT generated (unreachable)
```

**sim/step.ail (entry module):**
```ailang
module sim/step
import sim/protocol (Direction, directionToString)

func logDirection(d: Direction) -> ! {IO} () {
  log(directionToString(d))
}
```

**Generated funcs.go (only reachable functions):**
```go
import "github.com/sunholo/ailang/runtime"

// From sim/protocol (reachable via logDirection)
func SimProtocol_DirectionToString(d any) any {
    // match implementation
}
// NOTE: unusedHelper NOT generated - not reachable

// From sim/step (entry module)
func LogDirection(d any) any {
    return runtime.Log(SimProtocol_DirectionToString(d))
}
```

## Success Criteria

- [ ] `runtime/` package exists with Show, ConcatString, Log
- [ ] Slice arguments to ADT constructors compile without errors
- [ ] Generated code imports runtime package, not inline helpers
- [ ] Only reachable functions generated (no dead code)
- [ ] Module-prefixed function names are deterministic
- [ ] stapledon `go build ./...` succeeds
- [ ] All existing codegen tests still pass
- [ ] Documentation updated

## Testing Strategy

**Unit tests:**
- Test runtime.Show for all primitive types + edge cases
- Test runtime.ConcatString with various inputs
- Test slice conversion for `[]int64`, `[]string`, `[]map[string]any`
- Test reachability analysis on sample Core IR

**Integration tests:**
- Compile stapledon test_ctor.ail with protocol.ail
- Verify generated Go compiles and runs
- Test runtime behavior matches AILANG semantics

**Interpreter Oracle Tests (KEY CONTROL MECHANISM):**
- For each generated Go test case, also run same input via AILANG interpreter
- Assert equality of results
- Interpreter remains the semantic oracle

**Manual testing:**
- Build full stapledon game project
- Run game loop to verify runtime stability

## Non-Goals

**Not in this feature:**
- Optimized slice conversion (avoid allocations) - Later if profiling shows need
- All builtins - Only Show, ConcatString, Log initially
- Recursive/nested type handling - v0.5.3 if needed
- ADT-aware Show (pretty-printing ADTs) - Future enhancement
- Full call graph optimization (DCE) - Just reachability for now

## Timeline

**Day 1** (5 hours):
- Phase 1: Runtime package (1.5h)
- Phase 2: Slice type conversion (1.5h)
- Phase 3: Reachable functions (2h)
- Testing with stapledon

**Total: ~5 hours in one session**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Runtime package becomes bloated | Medium | Start minimal, add helpers only when needed |
| Slice conversion performance | Low | Profile later; current approach is correct first |
| Module prefix collisions | Low | Deterministic transform: `ModPath_FuncName` |
| Reachability misses edge cases | Medium | Fall back to generating all if analysis fails |
| Go/Interpreter semantic drift | High | **Interpreter oracle tests enforce equality** |

## References

- M-GAME-B Phase 1 fixes (4 commits in v0.5.2)
- [internal/gen/golang/codegen_expr.go](../../../internal/gen/golang/codegen_expr.go)
- [internal/gen/golang/codegen.go](../../../internal/gen/golang/codegen.go)
- stapledon game project bug reports via agent inbox

## Future Work

- **Performance optimizations**: Pool slice allocations, avoid conversion when types match
- **Complete builtin coverage**: All std/prelude builtins in runtime package
- **Type-specific Show**: ADT-aware pretty printing
- **Dead code elimination**: Full DCE pass on generated code
- **Runtime ABI documentation**: Lock down interface between Core IR and runtime

---

**Document created**: 2025-12-03
**Last updated**: 2025-12-03
