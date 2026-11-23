---
title: Language Syntax
sidebar_position: 1
reviewed: 2026-11-23
reviewBy: 2027-05-23
---

# AILANG Language Syntax Reference

This reference documents the current AILANG language syntax. Every code block below is checked against the compiler (`make check-doc-samples`). For features not yet implemented, see [Known Limitations](/docs/reference/limitations).

## Basic Constructs

```ailang
-- Comments use double dash
let x = 5
let incremented = let y = x in y + 1           -- Expression-style let
let double = \n. n * 2                         -- Lambda function
let label = if x > 0 then "pos" else "neg"     -- Conditional expression
let nums = [1, 2, 3]                           -- List literal
let person = {name: "Alice", age: 30}          -- Record literal
let triple = (1, "hello", true)                -- Tuple literal
```

## Module System

```ailang
-- Define a module
module examples/math

-- Import functions from standard library
import std/io (println)
import std/fs (readFile, writeFile)

-- Export a recursive function
export func factorial(n: int) -> int {
  if n <= 1 then 1 else n * factorial(n - 1)
}

-- Effectful function with multiple capabilities
export func main() -> () ! {IO, FS} {
  println("Factorial of 5:");
  println(show(factorial(5)))
}
```

Run with:
```bash
ailang run --caps IO,FS --entry main examples/math.ail
```

### Module-Level Bindings

Besides functions, a module can declare top-level `let` bindings — constants shared
across the module's functions. Both annotated and inferred forms work:

```ailang
module examples/config

let maxRetries : int = 3         -- with type annotation
let greeting = "hello"           -- inferred; may reference earlier bindings

export func banner() -> string = "${greeting} (retries=${show(maxRetries)})"
```

This is distinct from the expression form `let x = e in body`, which binds only within a
single expression. A top-level `let name = expr` is in scope for every declaration in the
module — use it for shared constants instead of zero-argument functions.

### Function Declarations

```ailang
import std/io (println)

-- Pure function (no effects)
export func add(x: int, y: int) -> int {
  x + y
}

-- Function with effects
export func greet(name: string) -> () ! {IO} {
  println("Hello, ${name}!")
}

-- Recursive function
export func factorial(n: int) -> int {
  if n <= 1 then 1 else n * factorial(n - 1)
}

-- Mutual recursion
export func isEven(n: int) -> bool {
  if n == 0 then true else isOdd(n - 1)
}

export func isOdd(n: int) -> bool {
  if n == 0 then false else isEven(n - 1)
}
```

#### Two body styles — the rule (read this once)

A function body is written in **either** block style or expression style:

| Style | Shape | `=`? | `;` between statements? |
|-------|-------|------|--------------------------|
| **Block body** | `func f(...) -> T { s1; s2; result }` | **No `=`** | Semicolon or newline between statements; last expression is the return value |
| **Expression body** | `func f(...) -> T = expr` | **`=` required** | Single expression (or `;`-separated sequence) |

```ailang
import std/io (print, println)

-- Block body: multiple statements, braces, NO '='
export func render() -> () ! {IO} {
  print("frame");
  println(" done")
}

-- Expression body: single expression, '=' REQUIRED
export pure func clamp(n: int) -> int = if n < 0 then 0 else n
```

*(Note: inside a `{ ... }` block, newlines separate statements except when the next line starts with `(` — put a `;` before a trailing tuple expression like `(a, b)` so it isn't parsed as a curried call.)*

### Inline Tests

```ailang
-- Inline tests for pure functions
pure func factorial(n: int) -> int
  tests [
    (0, 1),
    (1, 1),
    (5, 120),
    (10, 3628800)
  ]
{
  if n <= 1 then 1 else n * factorial(n - 1)
}

-- Multi-parameter functions use tuple syntax
pure func add(x: int, y: int) -> int
  tests [
    ((1, 2), 3),
    ((5, 7), 12),
    ((0, 0), 0)
  ]
{
  x + y
}

-- List operations
pure func sum(xs: [int]) -> int
  tests [
    ([], 0),
    ([1], 1),
    ([1, 2, 3], 6)
  ]
{
  match xs {
    [] => 0,
    x :: rest => x + sum(rest)
  }
}
```

Inline row inputs and expected values support scalar literals (including negative
numbers), lists, tuples, records, local ADT constructors, and nested combinations.
For example, a row may expect `[2, 3]`, `{value: 2}`, `Allow`, or `Deny("nope")`.
Binary operators, lambdas, and other expressions requiring elaboration are rejected by
`check` and `test` with `TST001` (unsupported syntax) or `TST002` (binary operator).
Write the computed value directly, or use a named `test "..." { ... }` block for those cases.

Run tests with:
```bash
ailang test examples/factorial.ail
```

## Lambda Expressions

```ailang
func lambdaExamples() -> (int, int, int) {
  -- Basic curried lambda
  let add = \x. \y. x + y;
  let add5 = add(5);
  let r1 = add5(3);  -- 8

  -- Higher-order functions
  let compose = \f. \g. \x. f(g(x));
  let double = \x. x * 2;
  let inc = \x. x + 1;
  let doubleThenInc = compose(inc)(double);
  let r2 = doubleThenInc(5);  -- 11

  -- Closures capture environment
  let makeAdder = \x. \y. x + y;
  let add10 = makeAdder(10);
  let r3 = add10(5);  -- 15

  (r1, r2, r3)
}
```

## Pattern Matching

`Option` (`Some`, `None`) and `Result` (`Ok`, `Err`) are in scope in every non-`std` module automatically.

```ailang
func matchExamples(value: Option[int]) -> (int, string, int, string) {
  -- Pattern matching on constructors
  let r1 = match Some(42) {
    Some(x) => x * 2,
    None => 0
  };

  -- Pattern matching on lists
  let r2 = match [1, 2, 3] {
    [] => "empty",
    [x] => "single",
    _ => "multiple"
  };

  -- Pattern matching with guards
  let r3 = match value {
    Some(x) if x > 0 => x * 2,
    Some(x) if x < 0 => 0,
    Some(x) => x,
    None => 0
  };

  -- Tuple patterns
  let r4 = match (1, "hello") {
    (0, _) => "zero",
    (_, "hello") => "greeting",
    (x, y) => "other"
  };

  (r1, r2, r3, r4)
}
```

### Exhaustiveness

A `match` must cover every value of what it matches; `ailang check` rejects one
that misses a case with `TC_MATCH_002` and names the missing pattern
(`no arm matches: Blue`). Arms with an `if` guard do not count toward coverage.
`int`, `float` and `string` matches need a catch-all arm. For a case that cannot
happen, say so with `unreachable("why")`: it has any type and stops the program
with `RT010` if it is ever reached. An arm that no value can reach is warning
`TC_MATCH_003`.

```ailang
module docs/exhaustive

type Light = Red | Amber | Green

pure func next(l: Light) -> Light = match l {
  Red => Green,
  Green => Amber,
  Amber => Red
}

pure func head(xs: [int]) -> int = match xs {
  [] => unreachable("callers pass a non-empty list"),
  [h, ..._] => h
}
```

Full runnable example: `examples/runnable/match_exhaustiveness.ail`.

## Records & Row Polymorphism

Closed record types require an exact field match; use **row polymorphism** (`{name: string | r}`) for functions that accept any record containing a `name` field (enabled by default, no env var needed):

```ailang
-- Row-polymorphic function accepts any record with a `name: string` field
export func getName[r](obj: {name: string | r}) -> string {
  obj.name
}

func recordExamples() -> (string, string, string) {
  let person = {name: "Alice", age: 30};
  let user = {
    profile: {name: "Bob", email: "bob@example.com"},
    admin: true
  };
  let n1 = getName(person);          -- "Alice"
  let n2 = getName({name: "Bob", id: 123});  -- "Bob"
  let n3 = user.profile.name;        -- "Bob"
  (n1, n2, n3)
}
```

## Block Expressions

```ailang
module docs/blocks

import std/io (println)

export func sumBlock() -> int {
  let x = 5;
  let y = 10;
  x + y
}

-- Multi-statement if/else branches require braces { ... }
export func describe(x: int) -> int ! {IO} {
  if x > 0 then {
    println("Positive");
    x * 2
  } else {
    println("Non-positive");
    0
  }
}

export func main() -> () ! {IO} {
  println("Line 1");
  println("Line 2");
  println(show(describe(sumBlock())))
}
```

## Effects and Capabilities

```ailang
module examples/demo

import std/io (println)
import std/fs (readFile, writeFile)
import std/clock (now, sleep)
import std/net (httpGet)

-- Multiple effects in function signature
export func processData() -> () ! {IO, FS, Clock, Net} {
  println("Starting...");

  let content = readFile("input.txt");
  let response = httpGet("https://api.example.com/data");

  sleep(1000);  -- Sleep for 1 second
  let timestamp = now();

  writeFile("output.txt", "${content}\n${response}");
  println("Done at ${show(timestamp)}")
}
```

Run with capabilities:
```bash
ailang run --caps IO,FS,Clock,Net --entry processData demo.ail
```

### Core Effects

| Effect | Key Stdlib Functions | Description |
|--------|----------------------|-------------|
| **IO** | `println`, `print`, `readLine` | Console I/O |
| **FS** | `readFile`, `writeFile`, `fileExists`, `listDir`, `mkdir`, `mkdirAll`, `isDir`, `isFile`, `removeFile` | File system access |
| **Clock** | `now`, `sleep` | Time operations (monotonic, deterministic mode available) |
| **Net** | `httpGet`, `httpPost`, `httpRequest` | HTTP requests with security (DNS rebinding prevention, IP blocking) |
| **Env** | `getEnv`, `getArgs` | Environment variables and CLI arguments |
| **AI** | `call`, `callJson` | Typed LLM calls as a first-class effect |
| **Debug** | `log`, `check` | Ghost effect for structured tracing |

## Planned Syntax (Not Yet Implemented)

The following features are **not yet implemented** in the parser (see [Known Limitations](/docs/reference/limitations) for current workarounds):

| Feature | Current Workaround |
|---------|--------------------|
| **`?` error propagation** | Explicit `match` on `Result` / `Option` |
| **Typed quasiquotes** | `"${expr}"` string interpolation |
| **CSP concurrency (channels / session types)** | Deferred |

## Type Inference & Type Classes

AILANG uses **Hindley-Milner type inference** with built-in type classes (`Num`, `Fractional`, `Eq`, `Ord`, `Show`):

```ailang
-- Types are inferred automatically
let x = 42                       -- x: int
let greeting = "hello"           -- greeting: string
let double = \n. n + n           -- double: int -> int (via Num constraint)
let identity = \a. a             -- identity: a -> a (polymorphic)

-- Type annotations on exported functions
export func add(a: int, b: int) -> int ! {} {
  a + b
}

-- Arithmetic, comparison, and string interpolation
let sum = 1 + 2 + 3              -- 6
let calc = 10 * 5 - 20 / 4       -- 45
let place = "world"
let interpolated = "hello ${place}"  -- "hello world" (++ is list-only)
let eq1 = 42 == 42               -- true
let lt = 5 < 10                  -- true
```

**Key features:**
- Automatic type inference for `let` bindings, lambdas, and function bodies
- Polymorphic functions (e.g., `\a. a` works for any type)
- Row polymorphism for extensible records and effects
- Built-in type classes (`Num`, `Eq`, `Ord`, `Show`) and `deriving (Eq)` for ADTs

## See Also

- [Examples](/docs/examples) - Working code examples demonstrating syntax
- [Effect System](/docs/reference/effects) - Effect annotations and capabilities
- [Module Reference](/docs/reference/modules) - Module imports and exports
- [Known Limitations](/docs/reference/limitations) - Current limitations and workarounds
- [Architecture: Types](/docs/architecture/types) - Type system internals
