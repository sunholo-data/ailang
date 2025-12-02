# Go Interop Guide

This guide explains how to integrate AILANG code with Go applications.

> **ABI Stability Notice (v0.5.x)**: The Go interop ABI is considered "stable preview".
> Breaking changes are allowed until v0.6.0 but will be announced in the CHANGELOG.
> See [ABI Stability](#abi-stability) for details.

## Type Mapping

AILANG types map to Go types as follows:

| AILANG Type | Go Type | Notes |
|-------------|---------|-------|
| `int` | `int64` | 64-bit signed integer |
| `float` | `float64` | 64-bit floating point |
| `string` | `string` | UTF-8 string |
| `bool` | `bool` | Boolean |
| `()` | `struct{}` | Unit type (empty struct) |
| `[T]` | `[]T` | Slice of element type |
| `{ field: T }` | `*TypeName` | Pointer to generated struct |
| ADT variants | `*TypeName` | Pointer to discriminated union |

## Extern Functions

Extern functions allow you to implement performance-critical code in Go while maintaining type safety with AILANG.

### Declaring Extern Functions

In your AILANG file:

```ailang
-- Declare types
type Coord = { x: int, y: int }
type Path = [Coord]

-- Declare extern function (implemented in Go)
extern func find_path(world: World, from: Coord, to: Coord) -> Path
```

### Generating Stubs

Run the compiler to generate Go stubs:

```bash
ailang compile --emit-go world.ail
```

This generates `extern_stubs.go` with function signatures to implement:

```go
// Find_path is an extern function declared in AILANG.
//
// AILANG signature:
//   extern func find_path(world: World, from: Coord, to: Coord) -> Path
//
// Implement this function to provide the behavior.
func Find_path(world *World, from *Coord, to *Coord) []*Coord {
    panic("not implemented: find_path")
}
```

### Implementing Extern Functions

Replace the panic with your implementation:

```go
func Find_path(world *World, from *Coord, to *Coord) []*Coord {
    // Your A* pathfinding implementation here
    return aStarSearch(world, from, to)
}
```

## Restrictions

1. **Monomorphic only**: Extern functions cannot use type parameters (generics)
   ```ailang
   -- ERROR: extern functions cannot be polymorphic
   extern func identity[T](x: T) -> T
   ```

2. **No underscore prefix**: Extern function names cannot start with `_`
   ```ailang
   -- ERROR: underscore prefix reserved for builtins
   extern func _internal_helper(x: int) -> int
   ```

3. **Explicit return type**: Extern functions must declare their return type
   ```ailang
   -- ERROR: must have explicit return type
   extern func do_something(x: int)

   -- OK
   extern func do_something(x: int) -> ()
   ```

## Error Messages

Common errors and their solutions:

### EXT001: Underscore prefix not allowed
```
EXT001: extern function '_helper' cannot have underscore-prefix (reserved for builtins)
Suggestion: Use a public name without leading underscore
```
**Solution**: Remove the leading underscore from the function name.

### EXT002: Polymorphic not supported
```
EXT002: extern functions cannot be polymorphic (no type parameters)
Suggestion: Extern functions must use concrete types like int, float, string
```
**Solution**: Use concrete types instead of type parameters.

### EXT003: Missing return type
```
EXT003: extern functions must have explicit return type
Suggestion: Add '-> ReturnType' after parameters
```
**Solution**: Add an explicit return type annotation.

## Best Practices

1. **Keep extern functions focused**: Extern functions should do one thing well
2. **Document type expectations**: Add comments explaining any type constraints
3. **Handle errors explicitly**: Return error values rather than panic
4. **Test thoroughly**: Write Go tests for your extern implementations

## ADT to Go Mapping Rules

When AILANG Algebraic Data Types (ADTs) are compiled to Go, the following rules apply:

### Record Types

AILANG record types become Go structs with pointer semantics:

```ailang
type Player = { name: string, health: int, position: Coord }
```

Generates:

```go
type Player struct {
    Name     string
    Health   int64
    Position *Coord
}
```

**Rules:**
- Field names are capitalized (exported)
- Nested records use pointer types
- All record type function parameters/returns use pointers

### Sum Types (Variants)

AILANG sum types (ADTs with multiple constructors) become Go structs with a discriminator:

```ailang
type Option[T] =
  | Some(T)
  | None
```

Generates:

```go
type Option struct {
    Tag   OptionTag
    Some  *SomeData  // nil if Tag != OptionTagSome
}

type OptionTag int

const (
    OptionTagNone OptionTag = iota
    OptionTagSome
)

type SomeData struct {
    Value interface{}  // Generic types use interface{}
}
```

**Rules:**
- Discriminator field is named `Tag`
- Constructor data stored in nullable fields
- Use type switch on `Tag` for pattern matching

### Effect Handlers (Experimental)

Effect handlers are not yet fully supported in Go codegen. Current workaround:

1. Use extern functions for effectful operations
2. Implement handlers in Go

```ailang
-- Declare intent, implement in Go
extern func read_file(path: string) -> string
extern func write_file(path: string, content: string) -> ()
```

## Working Example

See `examples/sim_stub/` for a complete working example demonstrating:
- Type definitions in AILANG
- Extern function declarations
- Generated Go types and stubs
- Go implementation
- Deterministic simulation

```bash
# Run the example
cd examples/sim_stub
make run
```

## ABI Stability

### Stability Promise (v0.5.x)

The Go interop ABI is considered **"stable preview"** for v0.5.x:

| Component | Stability | Notes |
|-----------|-----------|-------|
| Type mapping (primitives) | Stable | int→int64, float→float64, etc. |
| Record type generation | Stable | Struct field ordering preserved |
| Extern function signatures | Stable | Generated stubs won't break |
| ADT discriminator format | Preview | May change before v0.6.0 |
| Generic type handling | Preview | Currently uses interface{} |

### What "Stable Preview" Means

- **Safe to use in production** for non-generic, non-ADT code
- **Breaking changes announced** in CHANGELOG with migration path
- **Full stability guaranteed** starting v0.6.0

### Migration Guide: v0.4.x to v0.5.x

If upgrading from v0.4.x:

1. **New `compile` subcommand**: Use `ailang compile --emit-go` instead of any previous codegen method
2. **Extern functions**: New feature - no migration needed
3. **Type mapping unchanged**: Existing type correspondences still apply
4. **Generated code location**: Now outputs to `gen/<package>/` by default

### Reporting Issues

If you encounter ABI-related issues:

1. Check the [CHANGELOG](../../../CHANGELOG.md) for known issues
2. File an issue at [github.com/sunholo-data/ailang/issues](https://github.com/sunholo-data/ailang/issues)
3. Include: AILANG version, Go version, generated code sample
