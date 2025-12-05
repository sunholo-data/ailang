---
sidebar_position: 3
title: Effect System
description: Capability-based side effects in AILANG
---

# Effect System

AILANG uses an **algebraic effect system** with capability-based security. All side effects must be declared in function signatures and granted at runtime.

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
| `exists` | `string -> bool ! {FS}` | Check if file exists |

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

```typescript
import std/net (httpGet, httpPost)

func fetchData(url: string) -> string ! {Net} {
  httpGet(url)
}

func postData(url: string, body: string) -> string ! {Net} {
  httpPost(url, body)
}
```

**Builtins:**

| Function | Type | Description |
|----------|------|-------------|
| `httpGet` | `string -> string ! {Net}` | HTTP GET request |
| `httpPost` | `(string, string) -> string ! {Net}` | HTTP POST request |

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

- [Language Syntax](/docs/reference/language-syntax) - Complete syntax reference
- [Module System](/docs/reference/modules) - Imports and exports
- [Testing Guide](/docs/guides/testing) - Writing tests
- [Limitations](/docs/reference/limitations) - Known limitations
