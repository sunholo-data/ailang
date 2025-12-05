# M-CODEGEN-RECORDUPDATE: Go Codegen for Record Update Expressions

**Version**: v0.5.1
**Priority**: High (blocks external project)
**Estimated Effort**: 2-3 hours
**Status**: Planned

## Problem Statement

When running `ailang compile --emit-go` on files using record update syntax (`{ record | field: value }`), code generation fails with:

```
unsupported expression type: *core.RecordUpdate
```

This blocks full Go codegen for projects like `stapledons_voyage` that use record updates extensively for immutable state management in game loops.

**Source**: Bug report from `stapledons_voyage` project (msg_20251202_173307_464213db1ed7)

## Current State

### What Works
- Effect handler interfaces (DebugHandler, RandHandler, ClockHandler)
- Type generation
- Most expression types (Lit, Var, Lambda, App, Let, LetRec, If, Match, BinOp, UnOp, Record, RecordAccess, List, Tuple, Intrinsic, DictRef, DictApp)

### What's Missing
The `generateExpr` function in `internal/gen/golang/codegen.go` doesn't handle `*core.RecordUpdate`:

```go
// Line 175-238: generateExpr switch statement
case *core.RecordAccess:
    return g.generateRecordAccess(e)

// MISSING: case *core.RecordUpdate:
//     return g.generateRecordUpdate(e)

default:
    return fmt.Errorf("unsupported expression type: %T", expr)
```

### Core AST Definition

```go
// internal/core/core.go:234-238
type RecordUpdate struct {
    CoreNode
    Base    CoreExpr            // Must be atomic in ANF
    Updates map[string]CoreExpr // All values must be atomic
}
```

## Proposed Solution

### Implementation

Add a `generateRecordUpdate` function to `internal/gen/golang/codegen.go`:

```go
// generateRecordUpdate generates Go code for a record update expression.
// AILANG: { base | field1: val1, field2: val2 }
// Go: Creates a new struct with updated fields using struct literal syntax
func (g *Generator) generateRecordUpdate(e *core.RecordUpdate) error {
    // Generate: func() interface{} {
    //     _base := <base>
    //     return map[string]interface{}{
    //         "field1": <val1>,
    //         "field2": <val2>,
    //         ...existing fields from _base...
    //     }
    // }()

    // For now, use runtime helper approach:
    g.write("RecordUpdate(")
    if err := g.generateExpr(e.Base); err != nil {
        return err
    }
    g.write(", map[string]interface{}{")

    first := true
    for field, val := range e.Updates {
        if !first {
            g.write(", ")
        }
        first = false
        g.writef("%q: ", field)
        if err := g.generateExpr(val); err != nil {
            return err
        }
    }

    g.write("})")
    return nil
}
```

### Runtime Helper

Add to generated code preamble or runtime library:

```go
// RecordUpdate creates a new record with specified fields updated
func RecordUpdate(base interface{}, updates map[string]interface{}) interface{} {
    baseMap, ok := base.(map[string]interface{})
    if !ok {
        return updates // If base isn't a map, just return updates
    }

    result := make(map[string]interface{}, len(baseMap)+len(updates))
    for k, v := range baseMap {
        result[k] = v
    }
    for k, v := range updates {
        result[k] = v
    }
    return result
}
```

### Switch Case Addition

```go
// In generateExpr switch statement, add after RecordAccess:
case *core.RecordUpdate:
    return g.generateRecordUpdate(e)
```

## Test Cases

### 1. Simple Record Update

```ailang
module test/recordupdate

export func updateAge(person: {name: string, age: int}, newAge: int) -> {name: string, age: int} =
  { person | age: newAge }
```

Expected Go output:
```go
func UpdateAge(person interface{}, newAge interface{}) interface{} {
    return RecordUpdate(person, map[string]interface{}{"age": newAge})
}
```

### 2. Multiple Field Update

```ailang
export func reset(state: {x: int, y: int, score: int}) -> {x: int, y: int, score: int} =
  { state | x: 0, y: 0 }
```

### 3. Nested Update (if supported)

```ailang
export func movePlayer(game: {player: {x: int, y: int}, score: int}, dx: int) -> ... =
  { game | player: { game.player | x: game.player.x + dx } }
```

## Files to Modify

1. **internal/gen/golang/codegen.go**
   - Add `case *core.RecordUpdate` to `generateExpr` switch
   - Add `generateRecordUpdate` function
   - Add `RecordUpdate` runtime helper (or import from runtime package)

2. **internal/gen/golang/codegen_test.go**
   - Add test cases for record update codegen

## Success Criteria

1. `ailang compile --emit-go` succeeds for files with record update syntax
2. Generated Go code compiles without errors
3. Runtime behavior matches interpreter semantics
4. stapledons_voyage sim/step.ail compiles successfully

## Related Work

- Record creation (`*core.Record`) already supported
- Record access (`*core.RecordAccess`) already supported
- This completes the record operations trilogy

## Timeline

- Implementation: 2 hours
- Testing: 1 hour
- Total: 3 hours

## Notes

- Map iteration order is non-deterministic in Go, but record update semantics are well-defined (later updates override earlier ones)
- Consider typed struct generation in future for better performance
