---
sidebar_position: 3
title: Effect System
description: Capability-based side effects in AILANG
---

# Effect System

AILANG uses an **algebraic effect system** with capability-based security. All side effects must be declared in function signatures and granted at runtime.

:::info Academic Foundation
AILANG's effect system builds on foundational research:

- **Plotkin & Pretnar (2009)**: [Handlers of Algebraic Effects](/references/plotkin-pretnar-2009.pdf) — Core theory of algebraic effects
- **Leijen (2014)**: [Koka: Row Polymorphic Effect Types](/references/leijen-koka-2014-arxiv.pdf) — Row polymorphism for effects
- **Lucassen & Gifford (1988)**: [Polymorphic Effect Systems](/references/lucassen-gifford-1988.pdf) — Original effect typing

See [Citations & Bibliography](/docs/references) for complete references.
:::

:::tip See Effects in Action
Browse the [Live Demos](https://www.sunholo.com/ailang-demos/) to see effects working in the browser: AI effect ([Claude Chat](https://www.sunholo.com/ailang-demos/streaming/claude_chat/), [Gemini Live](https://www.sunholo.com/ailang-demos/streaming/gemini_live/)), IO effect ([DocParse](https://www.sunholo.com/ailang-demos/docparse.html)), and contract-verified effect boundaries ([Safe Agent](https://www.sunholo.com/ailang-demos/streaming/safe_agent/)).
:::

## Why Effects?

Traditional languages hide side effects:

```python
# Python - side effects are invisible
def process_data():
    data = read_file("input.txt")  # FS access hidden
    print("Processing...")          # IO hidden
    return transform(data)
```

**AILANG makes effects explicit:**

```typescript
-- AILANG - effects declared in signature
func processData() -> Data ! {IO, FS} {
  println("Processing...");
  let data = readFile("input.txt");
  transform(data)
}
```

**Benefits:**
- **Reasoning**: Know exactly what a function can do from its type
- **Security**: Grant only needed capabilities at runtime
- **Testing**: Mock effects for deterministic testing
- **AI-Friendly**: Models can verify effect safety statically

## Available Effects

### IO Effect

Console input/output operations.

```typescript
import std/io (println, print, readLine)

func greet(name: string) -> () ! {IO} {
  print("Hello, ");
  println(name ++ "!")
}
```

**Builtins:**

| Function | Type | Description |
|----------|------|-------------|
| `println` | `string -> () ! {IO}` | Print with newline |
| `print` | `string -> () ! {IO}` | Print without newline |
| `readLine` | `() -> string ! {IO}` | Read line from stdin |

### FS Effect

File system operations.

```typescript
import std/fs (readFile, writeFile, exists)

func copyFile(src: string, dst: string) -> () ! {FS} {
  let content = readFile(src);
  writeFile(dst, content)
}
```

**Builtins:**

| Function | Type | Description |
|----------|------|-------------|
| `readFile` | `string -> string ! {FS}` | Read file contents |
| `writeFile` | `(string, string) -> () ! {FS}` | Write content to file |
| `fileExists` | `string -> bool ! {FS}` | Check if file exists |
| `listDir` | `string -> [string] ! {FS}` | List directory entries (sorted) |
| `mkdir` | `string -> () ! {FS}` | Create a single directory |
| `mkdirAll` | `string -> () ! {FS}` | Create directory tree (mkdir -p) |
| `isDir` | `string -> bool ! {FS}` | Check if path is a directory |
| `isFile` | `string -> bool ! {FS}` | Check if path is a regular file |
| `removeFile` | `string -> () ! {FS}` | Remove file or empty directory |
| `_zip_listEntries` | `string -> Result[List[string], string] ! {FS}` | List ZIP archive entries |
| `_zip_readEntry` | `(string, string) -> Result[string, string] ! {FS}` | Read text entry from ZIP |
| `_zip_readEntryBytes` | `(string, string) -> Result[string, string] ! {FS}` | Read binary entry from ZIP (base64) |

#### FS Sandbox

Restrict all FS operations to a directory by setting the `AILANG_FS_SANDBOX` environment variable:

```bash
AILANG_FS_SANDBOX=/tmp/sandbox ailang run --caps FS --entry main program.ail
```

When set, all file paths are resolved relative to the sandbox root:

```typescript
-- With AILANG_FS_SANDBOX=/tmp/sandbox:
readFile("data.txt")       -- reads /tmp/sandbox/data.txt
writeFile("out.txt", s)    -- writes /tmp/sandbox/out.txt
listDir("subdir")          -- lists /tmp/sandbox/subdir
```

This applies to **every** FS builtin (`readFile`, `writeFile`, `readFileBytes`, `writeFileBytes`, `appendFile`, `appendFileBytes`, `fileExists`, `listDir`, `mkdir`, `mkdirAll`, `isDir`, `isFile`, `removeFile`) and all `std/zip` operations. It also sets the working directory for `std/process` `exec`.

If unset (the default), paths resolve normally from the process working directory.

:::tip Use Cases
- **Eval benchmarks**: isolate generated code to a temp directory
- **Agent workspaces**: confine AI agents to their workspace
- **Untrusted code**: prevent access to files outside the sandbox
:::

### Clock Effect

Time operations with deterministic mode support.

```typescript
import std/clock (now, sleep)

func timedOperation() -> int ! {Clock, IO} {
  let start = now();
  sleep(1000);  -- Sleep 1 second
  let end = now();
  println("Elapsed: " ++ show(end - start) ++ "ms");
  end - start
}
```

**Builtins:**

| Function | Type | Description |
|----------|------|-------------|
| `now` | `() -> int ! {Clock}` | Current monotonic time (ms) |
| `sleep` | `int -> () ! {Clock}` | Sleep for milliseconds |

**Deterministic Mode:**

```bash
# Fixed time for reproducible tests
ailang run --clock-mode fixed --clock-start 1000 program.ail
```

### Net Effect

HTTP operations with security protections.

```ailang
-- Simple: httpGet/httpPost return body string directly
import std/net (httpGet, httpPost)

func fetchData(url: string) -> string ! {Net} =
  httpGet(url)

func postData(url: string, body: string) -> string ! {Net} =
  httpPost(url, body)
```

```ailang
-- Advanced: httpRequest returns Result[HttpResponse, NetError]
-- HttpResponse = {status: int, headers: List[{name, value}], body: string, ok: bool}
import std/net (httpRequest)
import std/json (decode)

func fetchJson(url: string) -> Result[Json, string] ! {Net} =
  match httpRequest("GET", url, [], "") {
    Ok(resp) => decode(resp.body),   -- resp.body is the body string
    Err(Transport(msg)) => Err("network error: " ++ msg),
    Err(_) => Err("request failed")
  }
```

> **Common mistake:** `Ok(resp)` captures the full `HttpResponse` record, not just the body.
> Use `resp.body` to get the response body string. To parse JSON, do `decode(resp.body)` not `decode(resp)`.

**Builtins:**

| Function | Type | Description |
|----------|------|-------------|
| `httpGet` | `string -> string ! {Net}` | HTTP GET, returns body string directly |
| `httpPost` | `(string, string) -> string ! {Net}` | HTTP POST, returns body string directly |
| `httpRequest` | `(string, string, List[{name, value}], string) -> Result[HttpResponse, NetError] ! {Net}` | Full HTTP with headers, status codes, structured errors |

**Security Features:**
- DNS rebinding prevention
- Private IP blocking (10.x.x.x, 192.168.x.x, etc.)
- HTTPS enforcement (configurable)

### Env Effect

Environment variable access.

```typescript
import std/env (getEnv)

func getConfig() -> string ! {Env} {
  match getEnv("CONFIG_PATH") {
    Some(path) => path,
    None => "/etc/default.conf"
  }
}
```

**Builtins:**

| Function | Type | Description |
|----------|------|-------------|
| `getEnv` | `string -> Option[string] ! {Env}` | Get environment variable |

### Process Effect

Execute external commands with capability-based security.

```typescript
import std/process (exec)
import std/bytes (toString)

func runCommand(cmd: string, args: [string]) -> () ! {IO, Process} {
  match exec(cmd, args) {
    Ok(out) => {
      println("stdout: " ++ toString(out.stdout));
      println("exit code: " ++ show(out.exitCode))
    },
    Err(NotFound(cmd))  => println("not found: " ++ cmd),
    Err(NotAllowed(cmd)) => println("blocked: " ++ cmd),
    Err(Timeout(ms))    => println("timed out after " ++ show(ms) ++ "ms"),
    Err(_)              => println("other error")
  }
}
```

**Builtins:**

| Function | Type | Description |
|----------|------|-------------|
| `exec` | `(string, [string]) -> Result[ProcessOutput, ProcessError] ! {Process}` | Execute command with arguments |
| `spawnProcess` | `(string, [string]) -> ProcessHandle ! {Process}` | Spawn subprocess with writable stdin pipe |
| `writeProcessStdin` | `(ProcessHandle, bytes) -> Result[(), string] ! {Process}` | Write bytes to subprocess stdin |
| `closeProcessStdin` | `(ProcessHandle) -> () ! {Process}` | Close stdin pipe (signals EOF) |

**ProcessHandle:** Opaque ADT `ProcessHandle(int)` — returned by `spawnProcess`, used with `writeProcessStdin` and `closeProcessStdin`.

**Completion Semantics (important):**
- **`Ok`** = process completed (even with non-zero exit code) — check `out.exitCode`
- **`Err`** = infrastructure failure (command not found, timeout, blocked by allowlist)

**ProcessOutput** fields: `stdout: bytes`, `stderr: bytes`, `exitCode: int`, `truncated: bool`, `resolvedPath: string`

**ProcessError** variants: `NotFound`, `NotAllowed`, `PermissionDenied`, `Timeout`, `OutputLimitExceeded`, `SpawnFailed`, `AbnormalExit`

**Security Features:**
- No shell expansion — arguments passed literally (no `sh -c`)
- Command allowlist with path pinning (resolved at startup, prevents TOCTOU)
- Mandatory timeout (default: 30s)
- Output size limits (default: 10MB)

**CLI Flags:**

```bash
ailang run --caps Process --entry main module.ail
ailang run --caps IO,Process --process-timeout 10s --entry main module.ail
ailang run --caps IO,Process --process-allowlist "echo,curl,git" --entry main module.ail
ailang run --caps IO,Process --process-max-output 5242880 --entry main module.ail
```

### Stream Effect

Real-time streaming I/O: WebSocket, SSE, and multi-source event multiplexing.

```typescript
import std/stream (connect, transmit, onEvent, runEventLoop, disconnect,
                   sourceOfConn, asyncReadStdinLines, asyncExecProcess, selectEvents,
                   StreamEvent, Message, Closed, StreamError, SourceText, SourceBytes)
import std/result (Result, Ok, Err)

func main() -> unit ! {Stream, IO} {
  let stdin = asyncReadStdinLines("stdin", 10);
  selectEvents([stdin], \event. match event {
    SourceText(source, text) => { println("[" ++ source ++ "] " ++ text); true },
    _ => true
  })
}
```

**Core Builtins:**

| Function | Type | Description |
|----------|------|-------------|
| `connect` | `(string, StreamConfig) -> Result[StreamConn, StreamErrorKind] ! {Stream}` | Open WebSocket |
| `transmit` | `(StreamConn, string) -> Result[unit, StreamErrorKind] ! {Stream}` | Send text message |
| `transmitBinary` | `(StreamConn, bytes) -> Result[unit, StreamErrorKind] ! {Stream}` | Send binary data |
| `onEvent` | `(StreamConn, (StreamEvent) -> bool) -> unit ! {Stream}` | Register event handler |
| `runEventLoop` | `(StreamConn) -> unit ! {Stream}` | Block until handler returns false |
| `disconnect` | `(StreamConn) -> unit ! {Stream}` | Close connection |
| `sseConnect` | `(string, StreamConfig) -> Result[StreamConn, StreamErrorKind] ! {Stream}` | Open SSE (GET) |
| `ssePost` | `(string, string, StreamConfig) -> Result[StreamConn, StreamErrorKind] ! {Stream}` | Open SSE (POST) |
| `sourceOfConn` | `(StreamConn, string, int) -> StreamSource ! {Stream}` | Wrap connection as event source |
| `asyncReadStdinLines` | `(string, int) -> StreamSource ! {Stream}` | Stdin line reader source |
| `asyncExecProcess` | `(string, [string], string, int, int) -> StreamSource ! {Stream}` | Subprocess stdout source |
| `selectEvents` | `([StreamSource], (StreamEvent) -> bool) -> unit ! {Stream}` | Multi-source event loop |

**Event types:** `Message`, `Binary`, `Opened`, `Closed`, `StreamError`, `Ping`, `SSEData`, `SourceText`, `SourceBytes`

**See [Streaming & Real-Time I/O Guide](/docs/guides/streaming) for complete documentation, examples, and configuration.**

```bash
ailang run --caps Stream,IO --entry main module.ail
ailang run --caps Stream,Process,IO --process-allowlist "echo,ffmpeg" --entry main module.ail
```

## Effect Syntax

### Declaring Effects

Effects are declared after the return type with `!`:

```typescript
-- Single effect
func greet() -> () ! {IO} { ... }

-- Multiple effects
func process() -> Data ! {IO, FS, Net} { ... }

-- Pure function (no effects)
func add(x: int, y: int) -> int { x + y }

-- Explicit pure annotation
pure func factorial(n: int) -> int { ... }
```

### Combining Effects

Effects from called functions are combined:

```typescript
func helper() -> () ! {IO} {
  println("Helper called")
}

func main() -> () ! {IO, FS} {
  helper();           -- IO effect flows up
  let x = readFile("data.txt");  -- FS effect
  println(x)          -- IO effect
}
```

### Effect Subsumption

Functions can be called in contexts with additional effects:

```typescript
func pureCalc(x: int) -> int { x * 2 }

func effectful() -> int ! {IO} {
  println("Calculating...");
  pureCalc(21)  -- Pure function called in effectful context
}
```

## Running with Capabilities

Effects require explicit capability grants at runtime:

```bash
# Grant IO capability
ailang run --caps IO --entry main program.ail

# Grant multiple capabilities
ailang run --caps IO,FS,Net --entry main program.ail

# Grant all capabilities (development only!)
ailang run --caps ALL --entry main program.ail
```

**Missing capability error:**

```
Error: Function 'main' requires capability IO but it was not granted.
Use --caps IO to grant this capability.
```

## Capability Budgets

You can limit how many times a function can perform an effect using **capability budgets**:

```typescript
-- Function limited to 5 IO operations (maximum)
func rateLimited() -> () ! {IO @limit=5} {
  println("Call 1");  -- Uses 1/5 budget
  println("Call 2");  -- Uses 2/5 budget
  -- ...
  println("Call 6");  -- FAILS: BudgetExhaustedError
  ()
}

-- Function must perform at least 1 IO operation (minimum)
func auditRequired() -> () ! {IO @min=1} {
  println("Audit log entry");  -- Satisfies @min=1
  ()
  -- Returning without any IO would FAIL: BudgetUnderrunError
}

-- Combined: between 1 and 3 IO operations
func bounded() -> () ! {IO @min=1 @limit=3} {
  println("Required");  -- Satisfies minimum
  println("Optional");  -- Within limit
  ()
}
```

**Key features:**
- **Per-invocation**: Each function call gets a fresh budget
- **Composable**: Nested calls have their own budgets
- **Maximum limits** (`@limit=N`): Prevent runaway effects
- **Minimum requirements** (`@min=N`): Verify effects actually occurred
- **Bypass for debugging**: Use `--no-budgets` flag

```bash
# Normal mode - budgets enforced
ailang run --caps IO --entry main program.ail

# Debug mode - budgets bypassed
ailang run --caps IO --no-budgets --entry main program.ail
```

**See [Capability Budgets](/docs/reference/capability-budgets) for the complete reference.**

## Effect Safety

### Static Checking

The type checker verifies effect declarations:

```typescript
-- Error: readFile requires FS but not declared
func badFunction() -> string ! {IO} {
  readFile("data.txt")  -- Compile error!
}
```

### Runtime Enforcement

Even if you declare an effect, you must grant it:

```typescript
func main() -> () ! {FS} {
  writeFile("output.txt", "data")
}
```

```bash
# Fails - FS not granted
ailang run --caps IO --entry main program.ail
# Error: Capability FS required but not granted
```

## Effect Patterns

### Effect Polymorphism

Functions can work with any effect set:

```typescript
-- Works with any effects
func twice[e](f: () -> () ! e) -> () ! e {
  f();
  f()
}

-- Usage
twice(\(). println("Hello"))  -- ! {IO}
twice(\(). writeFile("log", "entry"))  -- ! {FS}
```

### Conditional Effects

Use pattern matching to handle effect boundaries:

```typescript
func maybeLog(verbose: bool, msg: string) -> () ! {IO} {
  if verbose then println(msg) else ()
}
```

### Effect-Free Core

Keep business logic pure, effects at boundaries:

```typescript
-- Pure business logic
pure func calculateTax(amount: float) -> float {
  amount * 0.2
}

-- Effectful boundary
func processInvoice(id: string) -> () ! {IO, FS} {
  let data = readFile("invoices/" ++ id ++ ".json");
  let amount = parseAmount(data);
  let tax = calculateTax(amount);  -- Pure!
  println("Tax: " ++ show(tax))
}
```

## Testing with Effects

### Mocking Effects (Planned)

```typescript
-- Future: Mock effects for testing
test "file processing" with {
  FS.mock({
    "input.txt" => "test data"
  })
} {
  let result = processFile("input.txt");
  assert(result == "TEST DATA")
}
```

### Pure Function Testing (Current)

Use inline tests for pure functions:

```typescript
pure func double(x: int) -> int
  tests [
    (0, 0),
    (5, 10),
    ((-3), (-6))
  ]
{
  x * 2
}
```

## Related Resources

- [AI Effect Guide](/docs/guides/ai-effect) - Using the AI effect for LLM calls
- [Streaming Guide](/docs/guides/streaming) - WebSocket, SSE, and multi-source multiplexing
- [Capability Budgets](/docs/reference/capability-budgets) - Fine-grained effect limiting
- [Module Execution](/docs/guides/module_execution) - Running modules with effects
- [Language Syntax](/docs/reference/language-syntax) - Complete syntax reference
- [Module System](/docs/reference/modules) - Imports and exports
- [Testing Guide](/docs/guides/testing) - Writing tests
- [Debugging Guide](/docs/guides/debugging) - Troubleshooting effect-related issues
- [Limitations](/docs/reference/limitations) - Known limitations
