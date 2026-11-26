---
reviewed: 2026-11-26
reviewBy: 2027-02-26
---

# Go Interop Guide

Integrate AILANG code with Go applications either via a child process running on the bytecode VM (`ailang run --bytecode`) or by embedding `internal/embed` inside the `ailang` Go module.

> **The Go emitter (`ailang compile --emit-go`) has been removed.** Use `ailang run --bytecode` (`--strict-bytecode` in CI) for external Go hosts, or `internal/embed` inside this repository.

## Integration Options

| Approach | Use Case | Engine | Scope |
|---|---|---|---|
| **Child process (`ailang run --bytecode`)** | Simulations, game logic, external Go repos | Bytecode VM | Any host language |
| **Runtime Embedding (`internal/embed`)** | Transforms, config DSLs, scripting | Reference evaluator (cached) | Inside `ailang` module only |

## Child process (`ailang run --bytecode`)

Start the AILANG program as a child process and exchange newline-delimited JSON over stdin/stdout:

```bash
ailang run --quiet --bytecode --package-dir sim --caps IO --entry main sim/ship.ail
```

Pass `--strict-bytecode` in CI so any construct unsupported by the VM fails rather than falling back to the evaluator.

## Runtime Embedding (`internal/embed`)

### Quick Start

```go
package main

import (
    "fmt"
    "log"

    "github.com/sunholo/ailang/internal/embed"
)

func main() {
    engine := embed.New("/path/to/project")
    defer engine.Close()

    result, err := engine.Call("transforms/formatter", "truncate", "Hello, World!", 10)
    if err != nil {
        log.Fatal(err)
    }

    text, _ := embed.ToString(result)
    fmt.Println(text) // "Hello, Wor..."
}
```

### Engine API

```go
engine := embed.New(basePath string) *Engine
engine.Load(modulePath string) error
engine.Call(modulePath, funcName string, args ...interface{}) (eval.Value, error)
engine.CallJSON(modulePath, funcName string, inputJSON []byte) ([]byte, error)
engine.Eval(code string) (eval.Value, error)
engine.ListExports(modulePath string) ([]string, error)
engine.HasExport(modulePath, name string) bool
engine.Close() error
```

### Value Conversion (`FromGo` / `ToGo`) & Extractors

| Go Type | AILANG Type | Extractor (`embed.To*`) |
|---|---|---|
| `nil` | `()` (unit) | `embed.IsUnit(v)` |
| `bool` | `bool` | `embed.ToBool(v)` |
| `int*` / `uint*` | `int` | `embed.ToInt(v)` |
| `float32` / `float64` | `float` | `embed.ToFloat(v)` |
| `string` | `string` | `embed.ToString(v)` |
| `[]byte` | `bytes` | `embed.ToBytes(v)` |
| `[]T` / `[N]T` | `[T]` / `array` | `embed.ToList(v)` |
| `map[string]T` / `struct` | `{...}` (record) | `embed.ToRecord(v)` |

For HTTP/JSON pipelines, `engine.CallJSON(modulePath, funcName, inputJSON)` handles `FromJSON`/`ToJSON` marshaling directly.

### Complete Example

```ailang
-- transforms/event_formatter.ail
module transforms/event_formatter

import std/list (length)
import std/string (length as strlen, substring, split)

export pure func truncate(text: string, maxLen: int) -> string =
  if maxLen == 0 || strlen(text) <= maxLen
  then text
  else "${substring(text, 0, maxLen)}..."

export pure func countWords(text: string) -> int =
  length(split(text, " "))
```

```go
package main

import (
    "fmt"
    "github.com/sunholo/ailang/internal/embed"
)

func main() {
    engine := embed.New(".")
    defer engine.Close()

    result, _ := engine.Call("transforms/event_formatter", "truncate",
        "The quick brown fox jumps over the lazy dog", 20)
    text, _ := embed.ToString(result)
    fmt.Println(text) // "The quick brown fox ..."

    result, _ = engine.Call("transforms/event_formatter", "countWords", "hello world foo bar")
    count, _ := embed.ToInt(result)
    fmt.Println(count) // 4
}
```

### Limitations

1. **Pure functions recommended**: Embedded modules should avoid unprovided effects.
2. **No hot reloading**: Restart `Engine` to pick up module edits.
3. **Standalone expressions**: `Eval("1+2")` requires let-binding (`Eval("let x = 1+2 in x")`).
4. **Concurrency**: `Engine` is mutex-protected; instantiate separate engines for parallel calls.
