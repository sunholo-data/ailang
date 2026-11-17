---
reviewBy: 2026-11-16
---

# Go Interop Guide

This guide explains how to integrate AILANG code with Go applications.

> **The Go emitter has been removed.** `ailang compile --emit-go` (and `--emit-go-v2`) used to
> translate AILANG to Go source. It was removed after v0.60.5: AILANG has two engines, the
> evaluator and the bytecode VM, and they agree. Run programs with `ailang run`, and add
> `--bytecode` for the VM (`--strict-bytecode` fails instead of falling back to the evaluator).
> See [Go code generation (removed)](#go-code-generation-removed).
>
> `internal/embed` is an `internal/` package, so Go only lets code inside the `ailang`
> module import it. A separate Go project cannot import it today. Use a
> [child process](#child-process-ailang-run---bytecode) instead.

## Integration Options

AILANG offers two ways to integrate with Go:

| Approach | Use Case | Performance | Complexity |
|----------|----------|-------------|------------|
| **Child process** | Simulations, game logic, any host language | Bytecode VM | Low |
| **Runtime Embedding** | Dynamic transforms, config DSL, scripting (inside the `ailang` module) | Interpreter (cached) | Low |

- **Child process**: start `ailang run --bytecode` and exchange JSON over stdin/stdout
- **Runtime Embedding** (v0.6.4+): Embed AILANG interpreter, call functions at runtime

---

## Child process (`ailang run --bytecode`)

The host starts the AILANG program as a child process and exchanges newline-delimited JSON
over stdin/stdout, one request and one reply per step. The program runs on the bytecode VM,
and the host can be written in any language. Stapledon's Voyage runs its simulation this way
under Godot.

```bash
ailang run --quiet --bytecode --package-dir sim --caps IO --entry main sim/ship.ail
```

Use `--strict-bytecode` in CI, so that code which cannot run on the VM fails instead of
silently falling back to the slower interpreter.

---

## Runtime Embedding (v0.6.4+)

The `internal/embed` package lets you embed AILANG as a scripting/extension language in Go applications.

### Quick Start

```go
package main

import (
    "fmt"
    "log"

    "github.com/sunholo/ailang/internal/embed"
)

func main() {
    // Create engine with base path for module resolution
    engine := embed.New("/path/to/project")
    defer engine.Close()

    // Call an AILANG function
    result, err := engine.Call(
        "transforms/formatter",  // module path
        "truncate",              // function name
        "Hello, World!",         // args...
        10,
    )
    if err != nil {
        log.Fatal(err)
    }

    // Extract typed result
    text, _ := embed.ToString(result)
    fmt.Println(text)  // "Hello, Wor..."
}
```

### Engine API

```go
// Create a new engine
engine := embed.New(basePath string) *Engine

// Load a module (optional - Call auto-loads)
engine.Load(modulePath string) error

// Call an exported function
engine.Call(modulePath, funcName string, args ...interface{}) (eval.Value, error)

// JSON-based calling (for language-agnostic integration)
engine.CallJSON(modulePath, funcName string, inputJSON []byte) ([]byte, error)

// Evaluate an expression (requires module context)
engine.Eval(code string) (eval.Value, error)

// Introspection
engine.ListExports(modulePath string) ([]string, error)
engine.HasExport(modulePath, name string) bool

// Cleanup
engine.Close() error
```

### Value Conversion

The embed package provides bidirectional conversion between Go and AILANG values:

**Go to AILANG (`FromGo`):**

| Go Type | AILANG Type |
|---------|-------------|
| `nil` | `()` (unit) |
| `bool` | `bool` |
| `int`, `int8`...`int64` | `int` |
| `uint`, `uint8`...`uint64` | `int` |
| `float32`, `float64` | `float` |
| `string` | `string` |
| `[]byte` | `bytes` |
| `[]T` | `[T]` (list) |
| `[N]T` | array |
| `map[string]T` | `{...}` (record) |
| `struct` | `{...}` (record, exported fields) |

**AILANG to Go (`ToGo`):**

| AILANG Type | Go Type |
|-------------|---------|
| `()` (unit) | `nil` |
| `bool` | `bool` |
| `int` | `int` |
| `float` | `float64` |
| `string` | `string` |
| `bytes` | `[]byte` |
| `[T]` (list) | `[]interface{}` |
| `{...}` (record) | `map[string]interface{}` |

### Type-Safe Extractors

For type safety, use the specific extractors:

```go
// Extract specific types
value, err := embed.ToInt(result)
value, err := embed.ToFloat(result)
value, err := embed.ToString(result)
value, err := embed.ToBool(result)
value, err := embed.ToBytes(result)
value, err := embed.ToList(result)
value, err := embed.ToRecord(result)

// Check for unit
if embed.IsUnit(result) {
    // result is ()
}
```

### JSON Integration

For language-agnostic use or HTTP APIs:

```go
// JSON input → AILANG → JSON output
inputJSON := []byte(`{"events": [...], "maxLen": 200}`)
outputJSON, err := engine.CallJSON(
    "transforms/formatter",
    "processEvents",
    inputJSON,
)
// outputJSON ready for HTTP response

// Manual JSON conversion
ailangVal, err := embed.FromJSON(jsonBytes)
jsonBytes, err := embed.ToJSON(ailangVal)
```

### Complete Example

Given this AILANG module:

```ailang
-- transforms/event_formatter.ail
module transforms/event_formatter

import std/string (length as strlen, substring)

export pure func truncate(text: string, maxLen: int) -> string =
  if maxLen == 0 || strlen(text) <= maxLen
  then text
  else "${substring(text, 0, maxLen)}..."

export pure func countWords(text: string) -> int =
  length(split(text, " "))
```

Call it from Go:

```go
package main

import (
    "fmt"
    "github.com/sunholo/ailang/internal/embed"
)

func main() {
    engine := embed.New(".")
    defer engine.Close()

    // Call truncate
    result, _ := engine.Call(
        "transforms/event_formatter",
        "truncate",
        "The quick brown fox jumps over the lazy dog",
        20,
    )
    text, _ := embed.ToString(result)
    fmt.Println(text)  // "The quick brown fox ..."

    // Call countWords
    result, _ = engine.Call(
        "transforms/event_formatter",
        "countWords",
        "hello world foo bar",
    )
    count, _ := embed.ToInt(result)
    fmt.Println(count)  // 4

    // Using structs
    type Event struct {
        TurnNum    int    `json:"turnNum"`
        StreamType string `json:"streamType"`
        Text       string `json:"text"`
    }

    event := Event{TurnNum: 1, StreamType: "text", Text: "Hello"}
    ailangVal, _ := embed.FromGo(event)
    // ailangVal is now an AILANG record
}
```

### Limitations

1. **Pure functions recommended**: Embedded modules should be pure for predictability
2. **No hot reloading**: Restart engine to pick up module changes
3. **Standalone expressions**: `Eval("1+2")` requires let binding: `Eval("let x = 1+2 in x")`
4. **Thread safety**: Engine is mutex-protected; for parallel calls, use separate engines

### When to Use Runtime Embedding

**Good for:**
- Configuration DSLs
- Data transformation pipelines
- Plugin/extension systems
- Scripting in applications
- Dynamic rule engines

For performance-critical loops, use a [child process](#child-process-ailang-run---bytecode) on the bytecode VM.

---

## Go code generation (removed)

`ailang compile --emit-go` and `--emit-go-v2` generated Go source (types, effect-handler
interfaces, extern stubs and a Debug-effect host contract). The emitter was removed after
v0.60.5, and `ailang compile` now fails with a message naming `ailang run`. It boxed every
value as `interface{}`, had no tail-call elimination, and gave answers that differed from the
evaluator's in several known cases. A Go host now runs AILANG as a
[child process](#child-process-ailang-run---bytecode) on the bytecode VM, or, inside the
`ailang` module, through [runtime embedding](#runtime-embedding-v064).
