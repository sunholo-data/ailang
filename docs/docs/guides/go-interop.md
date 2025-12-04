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

### Multi-File Compilation

When your game has multiple AILANG modules, compile them together. You can pass a directory to automatically discover all `.ail` files (v0.5.5+):

```bash
# Pass a directory - discovers all .ail files automatically
ailang compile --emit-go --package-name game sim/

# Or pass multiple files explicitly
ailang compile --emit-go --package-name game step.ail npc_ai.ail camera.ail

# Mix directories and files
ailang compile --emit-go --package-name game sim/ extra.ail
```

This generates one output file per source file (v0.5.5+):

```
gen/game/
├── types.go              # All ADT types (merged from all files)
├── debug_types_debug.go  # Debug effect (//go:build !release)
├── debug_types_release.go# Debug effect no-ops (//go:build release)
├── handlers.go           # Effect handler interfaces
├── runtime.go            # Shared runtime helpers
├── step.go               # Functions from step.ail
├── npc_ai.go             # Functions from npc_ai.ail
└── camera.go             # Functions from camera.ail
```

**Benefits of per-file output:**
- Smaller, more navigable files
- Easier to correlate generated code with source
- Better IDE navigation

**Important**: Compile all your `.ail` files in a single command. Compiling files separately will overwrite previous output.

```bash
# ✅ CORRECT - Compile all files together
ailang compile --emit-go sim/

# ❌ WRONG - Each compile overwrites the previous
ailang compile --emit-go step.ail      # Generates types.go
ailang compile --emit-go npc_ai.ail    # OVERWRITES types.go!
```

### Generated Extern Stubs

The compiler generates `extern_stubs.go` with function signatures to implement:

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

### Effect Handlers

AILANG effects (Rand, Clock, FS, Net, Env, AI) are compiled to Go interface calls. The generated code expects you to implement handler interfaces and initialize them before use.

#### How Effect Handler Codegen Works

When you compile AILANG code with effects:

```ailang
module game/step

import std/rand (rand_int)

export func rollDice() -> int ! {Rand} {
  rand_int(1, 6)
}
```

The compiler generates:

1. **Handler interfaces** in `effects.go`:
```go
// RandHandler provides the Rand effect implementation.
type RandHandler interface {
    RandInt(min, max int64) int64
}
```

2. **Handlers struct** to collect all handlers:
```go
type Handlers struct {
    Debug DebugHandler
    Rand  RandHandler
    Clock ClockHandler
    FS    FSHandler
    Net   NetHandler
    Env   EnvHandler
    AI    AIHandler
}
```

3. **Init function** to initialize the global handlers:
```go
func Init(h Handlers) {
    handlers = h
}

var handlers Handlers
```

4. **Function code** that calls handlers:
```go
func RollDice() int64 {
    return handlers.Rand.RandInt(1, 6)
}
```

#### Implementing Handlers

Game developers implement the handler interfaces for their platform:

```go
package main

import "mygame/gen/game"

// Implement RandHandler
type MyRandHandler struct {
    seed uint64
}

func (r *MyRandHandler) RandInt(min, max int64) int64 {
    // Your deterministic RNG implementation
    r.seed = r.seed*1103515245 + 12345
    return min + int64(r.seed)%(max-min+1)
}

func main() {
    // Initialize handlers before using any AILANG code
    game.Init(game.Handlers{
        Rand: &MyRandHandler{seed: 12345},
        // ... other handlers
    })

    // Now you can call AILANG functions
    result := game.RollDice()
}
```

#### Available Effect Handlers

| Handler | Methods | Purpose |
|---------|---------|---------|
| `DebugHandler` | `Log(msg, loc)`, `Assert(cond, msg, loc)`, `Collect()` | Debugging and tracing |
| `RandHandler` | `RandInt(min, max)`, `RandFloat()` | Deterministic random numbers |
| `ClockHandler` | `Now()`, `DeltaTime()` | Time and game loop timing |
| `FSHandler` | `Exists(path)`, `ReadFile(path)`, `WriteFile(path, content)` | File system operations |
| `NetHandler` | `HttpGet(url)`, `HttpPost(url, body)` | Network requests |
| `EnvHandler` | `GetEnv(key)` | Environment variables |
| `AIHandler` | `Call(prompt, opts)` | AI model calls |

#### Handler Implementation Tips

1. **Keep handlers stateless if possible** - easier to test and reason about
2. **Use deterministic implementations for Rand** - enables replay and testing
3. **Mock handlers for testing** - inject test doubles via Init()
4. **Initialize once at startup** - don't call Init() multiple times

#### Legacy: Extern Functions

For custom operations not covered by the built-in effect handlers, you can use extern functions (see [Extern Functions](#extern-functions) above).

## Debug Effect Host Contract

The Debug effect provides structured tracing for debugging and testing. It is a **ghost effect** - erasable in release mode for zero-cost production builds.

### Generated Files

Running `ailang compile --emit-go` generates:

| File | Purpose | Build Tag |
|------|---------|-----------|
| `debug_types_debug.go` | Full implementation (collects traces) | `//go:build !release` |
| `debug_types_release.go` | No-op implementation (zero cost) | `//go:build release` |

### DebugContext Interface

The generated DebugContext implements this contract:

```go
type DebugContext struct {
    // ... internal state
}

// Host lifecycle methods (HOST-ONLY - not callable from AILANG)
func NewDebugContext() *DebugContext
func (d *DebugContext) SetTimestamp(t int64)  // Host sets logical time
func (d *DebugContext) Collect() DebugOutput  // Host reads accumulated data
func (d *DebugContext) Reset()                // Clear for next step

// Effect operation handlers (called by generated AILANG code)
func (d *DebugContext) Log(msg, location string)
func (d *DebugContext) Assert(cond bool, msg, location string)

// Query methods
func (d *DebugContext) HasFailedAssertions() bool
func (d *DebugContext) FailedAssertions() []AssertionResult
```

### Output Types

```go
type DebugOutput struct {
    Logs       []LogEntry
    Assertions []AssertionResult
}

type LogEntry struct {
    Message   string  // Log message
    Location  string  // Source location (file.ail:42)
    Timestamp int64   // Logical time (host-defined)
}

type AssertionResult struct {
    Passed   bool    // Whether assertion passed
    Message  string  // Assertion message
    Location string  // Source location (file.ail:42)
}
```

### Host Integration Example

```go
func main() {
    debugCtx := game.NewDebugContext()

    for tick := 0; tick < 1000; tick++ {
        // 1. Reset and set timestamp for this step
        debugCtx.Reset()
        debugCtx.SetTimestamp(int64(tick))

        // 2. Run AILANG code (Debug.log/assert calls accumulate)
        world, output, err := game.Step(world, input, debugCtx)
        if err != nil {
            log.Fatal(err)
        }

        // 3. Host collects and handles debug output
        debugData := debugCtx.Collect()

        // Check for assertion failures
        if debugCtx.HasFailedAssertions() {
            for _, a := range debugCtx.FailedAssertions() {
                log.Printf("ASSERTION FAILED at %s: %s", a.Location, a.Message)
            }
        }
    }
}
```

### Building Debug vs Release

```bash
# Debug mode (default) - Debug effect collects traces
go build .

# Release mode - Debug effect is zero-cost no-ops
go build -tags release .
```

### JSON Schema for DebugOutput

The `DebugOutput` structure can be serialized to JSON for external tooling:

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "DebugOutput",
  "type": "object",
  "properties": {
    "logs": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "message": { "type": "string" },
          "location": { "type": "string", "pattern": "^.+:\\d+$" },
          "timestamp": { "type": "integer" }
        },
        "required": ["message", "location", "timestamp"]
      }
    },
    "assertions": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "passed": { "type": "boolean" },
          "message": { "type": "string" },
          "location": { "type": "string", "pattern": "^.+:\\d+$" }
        },
        "required": ["passed", "message", "location"]
      }
    }
  },
  "required": ["logs", "assertions"]
}
```

Example JSON output:

```json
{
  "logs": [
    {"message": "tick=1: delta=4", "location": "impl.go:Step", "timestamp": 1},
    {"message": "tick=2: delta=-2", "location": "impl.go:Step", "timestamp": 2}
  ],
  "assertions": [
    {"passed": true, "message": "tick should increase", "location": "impl.go:Step"},
    {"passed": true, "message": "seed should be preserved", "location": "impl.go:Step"}
  ]
}
```

### Design Principles

1. **Write-only from AILANG**: Code can only write Debug.log/Debug.assert, cannot read its own trace
2. **Host-controlled lifecycle**: Only the host can Collect() and Reset()
3. **Ghost effect**: Erased at build time in release mode, not just no-op at runtime
4. **Auto-injected locations**: Source positions added by compiler, not passed by user
5. **Abstract timestamps**: Host defines what timestamp means (tick, test index, etc.)

## Working Example

See `examples/sim_stub/` for a complete working example demonstrating:
- Type definitions in AILANG
- Extern function declarations
- Generated Go types and stubs
- Go implementation
- Deterministic simulation
- **Debug effect usage** with host-controlled lifecycle

```bash
# Run the example (debug mode)
cd examples/sim_stub
make run

# Build in release mode (Debug operations become no-ops)
go build -tags release .
./sim_stub
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

1. Check the [CHANGELOG](https://github.com/sunholo-data/ailang/blob/dev/CHANGELOG.md) for known issues
2. File an issue at [github.com/sunholo-data/ailang/issues](https://github.com/sunholo-data/ailang/issues)
3. Include: AILANG version, Go version, generated code sample
