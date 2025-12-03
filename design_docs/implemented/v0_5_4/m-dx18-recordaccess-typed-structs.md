# M-DX18: RecordAccess Uses Map Syntax on Typed Structs

**Version**: v0.5.4
**Priority**: High (causes runtime panics)
**Estimated Effort**: 1-2 hours
**Status**: Planned

## Problem Statement

When accessing fields on records, the codegen generates map access syntax (`record.(map[string]interface{})["field"]`), but after M-DX13/M-DX16, records are typed structs (`*World`), causing runtime panics.

**Source**: Bug report from `stapledons_voyage` project (msg_20251203_175923_722f8c4c738d)

## Current Behavior

AILANG code:
```ailang
type World = { tick: int, name: string }

pure func step(world: World) -> World =
  { world | tick: world.tick + 1 }  -- Accesses world.tick
```

Generated Go (WRONG):
```go
func step(world interface{}) interface{} {
    // M-DX16: RecordUpdate now preserves *World type ✓
    return RecordUpdate(world, map[string]interface{}{
        "tick": AddInt(
            world.(map[string]interface{})["tick"],  // PANIC! world is *World, not map!
            int64(1),
        ),
    })
}
```

## Expected Behavior

Generated Go (CORRECT):
```go
func step(world interface{}) interface{} {
    return RecordUpdate(world, map[string]interface{}{
        "tick": AddInt(
            FieldGet(world, "tick"),  // Works with both maps and structs
            int64(1),
        ),
    })
}

// Runtime helper
func FieldGet(record interface{}, field string) interface{} {
    // Handle map
    if m, ok := record.(map[string]interface{}); ok {
        return m[field]
    }
    // Handle typed struct using reflection
    val := reflect.ValueOf(record)
    if val.Kind() == reflect.Ptr {
        val = val.Elem()
    }
    if val.Kind() == reflect.Struct {
        goField := strings.ToUpper(field[:1]) + field[1:]  // tick -> Tick
        f := val.FieldByName(goField)
        if f.IsValid() {
            return f.Interface()
        }
    }
    return nil
}
```

## Root Cause

1. M-DX13 introduced typed record literals (`*World` instead of `map[string]interface{}`)
2. M-DX16 made RecordUpdate preserve typed structs
3. BUT `generateRecordAccess` still generates map access syntax
4. Result: Type mismatch at runtime

## Proposed Solution

### Option A: FieldGet Helper (Recommended)
Add a `FieldGet` runtime helper that handles both maps and structs:
- Check if value is `map[string]interface{}` → use map access
- Check if value is struct pointer → use reflection
- Convert field name to PascalCase for Go struct access

### Option B: Typed Code Generation
Generate different access patterns based on whether record type is known:
- Known record type → `record.(*World).Tick`
- Unknown type → `FieldGet(record, "tick")`

Option A is simpler and more consistent with RecordUpdate's approach.

## Implementation

### 1. Add FieldGet Runtime Helper

In `internal/gen/golang/codegen_runtime.go`:

```go
g.writef("// FieldGet retrieves a field from a record (map or typed struct).\n")
g.writef("// M-DX18: Handles both map[string]interface{} and typed structs.\n")
g.writef("func FieldGet(record interface{}, field string) interface{} {\n")
// ... implementation using reflection
```

### 2. Update generateRecordAccess

In `internal/gen/golang/codegen_ops.go`:

```go
func (g *Generator) generateRecordAccess(e *core.RecordAccess) error {
    // M-DX18: Use FieldGet helper instead of map access
    g.write("FieldGet(")
    if err := g.generateExpr(e.Record); err != nil {
        return err
    }
    g.writef(", %q)", e.Field)
    return nil
}
```

## Files to Modify

1. **internal/gen/golang/codegen_runtime.go**
   - Add `FieldGet` runtime helper (~30 LOC)

2. **internal/gen/golang/codegen_ops.go**
   - Update `generateRecordAccess` to use `FieldGet` (~10 LOC)

## Test Cases

1. Field access on typed struct works
2. Field access on map still works (backwards compatibility)
3. Nested field access works (`world.player.position.x`)
4. Field access in expressions works (`world.tick + 1`)

## Related Issues

- M-DX13: Typed record literals
- M-DX16: RecordUpdate preserves typed structs
- M-DX17: Integer literals as int64

## Success Criteria

1. `step(InitWorld())` works without panic
2. Multiple `step` calls work in a loop
3. All existing tests pass
