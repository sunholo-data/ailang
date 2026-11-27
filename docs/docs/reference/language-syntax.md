---
title: Language Syntax
sidebar_position: 1
reviewed: 2026-11-26
reviewBy: 2027-05-26
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

### Re-exports (`export import`)

Imports are not transitive. A module can opt in for names it chooses with
`export import`, which makes it a **facade**: the listed names become its
exports, and they keep the original's identity. A constructor imported through
the facade *is* the original's, so values built on either side match the
other's patterns.

```ailang parse
module shapes/api
export import shapes/core (Shape, Circle, Square, area)
export import shapes/old_names (perimeter as edgeLength)
```

```ailang parse
module main
import std/io (println)
import shapes/api (Shape, Circle, area)    -- the same as importing from shapes/core

export func main() -> () ! {IO} = println(show(area(Circle(2.0))))
```

- An `export import` line sits in the import header with the other imports,
  and must list its names (`export import M` or `export import M as A` is
  `PAR_REEXPORT_FORM`). Names may be renamed with `as`.
- Re-exporting a name the source module does not export is `IMP010`. A chain
  of facades that returns to itself is `MOD016`.
- A facade's interface shows its re-exports exactly as the source declares
  them: type, constructors, effects and contracts. A re-exported type brings
  its constructors, as importing a type does.
- A trailing `@deprecated("new/module/path")` marks the line as moved. Every
  name imported through it warns `TC_DEPRECATED_001` with the fix
  `import X from new/module/path`. This applies to types and constructors as
  well as functions.
- Through a module alias (`import shapes/api as A`), re-exported functions
  and types are reachable as `A.area` and `A.Shape`.

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

-- Canonical block-form if/else branches use braces { ... } (`ailang fmt` normalizes to this)
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

### Unbraced Multi-Statement `match` Arms & `if`/`else` Branches

Multi-statement `let` sequences are also accepted directly in `match` arms and `if`/`then`/`else` branches without outer braces; `ailang fmt` normalizes them into canonical `{ ... }` blocks:

```ailang
module docs/unbraced_branches

export pure func adjust(opt: Option[int], flag: bool) -> int =
  match opt {
    Some(x) =>
      let doubled = x * 2;
      if flag then
        let bumped = doubled + 1;
        bumped
      else doubled,
    None => 0
  }
```

## Pipe Operator (`|>`)

The left-associative pipe operator `|>` passes its left-hand value as the single argument to the unary function on the right. It has the lowest operator precedence (evaluated left-to-right), and works across the evaluator and `--strict-bytecode`. Wrap multi-argument calls or inline lambdas in parentheses:

```ailang
module docs/pipe_operator

import std/list (filter, map, foldl)

export pure func sumDoubledPositives(xs: [int]) -> int =
  xs
    |> (\ys. filter(\x. x > 0, ys))
    |> (\ys. map(\x. x * 2, ys))
    |> (\ys. foldl(\acc x. acc + x, 0, ys))
```

## Destructuring (`let` & Lambdas)

Irrefutable tuple and record patterns work directly in `let` bindings and lambda parameters (including nested patterns, field renaming, and `...` rest patterns). Refutable constructors like `Some(x)` must still use `match`.

```ailang
module docs/destructuring

import std/list (map)

export pure func summarize(pair: (int, string), cfg: {host: string, port: int, tls: bool}) -> string {
  let (count, label) = pair;
  let {host, port: p, ...} = cfg;
  let pairs = [(1, "a"), (2, "b")];
  let labels = map(\(n, tag). "${tag}:${show(n)}", pairs);
  let describe = \{host: h, ...}. h;
  "${label}#${show(count)}@${describe(cfg)}:${show(p)} ${show(labels)}"
}
```

## Error & Option Propagation (`let x = expr?`)

Inside a function returning `Result[T, E]` or `Option[T]`, `let x = expr?` (including pattern forms like `let (a, b) = expr?`) unwraps `Ok`/`Some` along the function's return spine and returns `Err`/`None` early. It works in both `{ ... }` blocks and `let ... in ...` expression bodies.

The `?` suffix is valid only on the right-hand side of a `let` binding on the return spine: using `?` inside a nested sub-expression emits `PAR_TRY_LET_ONLY` (hoist it into its own `let`), and using it outside a matching `Result`/`Option` return spine emits `TC_TRY_NON_SPINE`.

```ailang
module docs/try_propagation

import std/string (stringToInt)

export pure func addParsed(a: string, b: string) -> Option[int] {
  let x = stringToInt(a)?;
  let y = stringToInt(b)?;
  Some(x + y)
}

pure func parsePositive(s: string) -> Result[int, string] =
  match stringToInt(s) {
    Some(n) => if n > 0 then Ok(n) else Err("must be positive"),
    None => Err("not an integer")
  }

export pure func addPositiveStrings(a: string, b: string) -> Result[int, string] {
  let x = parsePositive(a)?;
  let y = parsePositive(b)?;
  Ok(x + y)
}
```

## Deriving Type Classes & JSON Codecs

Monomorphic ADTs and records can derive equality (`deriving (Eq)`) and JSON codecs (`deriving (Json)`):

- `deriving (Eq)` generates structural `==` and `!=` for the type.
- `deriving (Json)` (requires `import std/json (Json)`) synthesizes pure `encode<Type>(v: <Type>) -> Json` and `decode<Type>(j: Json) -> Result[<Type>, string]` functions (exported iff the type is exported).
  - For records, `Option[T]` fields omit their key when `None` on encode and accept missing keys or `null` on decode.
  - For sum ADTs, variants encode with `"tag"` plus `"fields"` (positional arguments) or named keys (inline record arguments).
  - Polymorphic types cannot derive `Json` (`TC_DERIVE_JSON_POLY`).

```ailang
module docs/deriving_json

import std/json (Json, encode, decode)

export type ServerConfig = { host: string, port: int, tag: Option[string] } deriving (Eq, Json)

export type Command =
  | Ping
  | Scale(int)
  | Deploy(service: string, region: Option[string])
  deriving (Eq, Json)

export pure func roundTripConfig(cfg: ServerConfig) -> Result[bool, string] {
  let encoded = encode(encodeServerConfig(cfg));
  let j = decode(encoded)?;
  let decoded = decodeServerConfig(j)?;
  Ok(decoded == cfg)
}
```

## Typed Holes (`_?` and `?name`)

Write `_?` (anonymous) or `?name` (named) in any expression position to ask the type checker what type is expected at that site and which local bindings are in scope. `ailang check` emits a structured `TC_HOLE` diagnostic for every hole in the module in a single pass (`ailang check --json` or `--format=agent`), and `ailang fmt` preserves holes verbatim:

```ailang parse
module docs/typed_holes

export pure func transform(items: [int], offset: int) -> [string] =
  ?convertItems
```

Running `ailang check --format=agent` (or `--json`) reports the expected type and locals in scope:

```text
docs/typed_holes.ail:4:3 TC_HOLE: typed hole `?convertItems` has expected type `list[string]` → in scope: items: list[int], offset: int
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
| **Typed quasiquotes** | `"${expr}"` string interpolation |
| **CSP concurrency (channels / session types)** | Deferred |

## Tooling & Diagnostics

- **Semantic module inspection:** `ailang inspect <file-or-dir>` (`--format=agent` default, `--json`, `--exports-only`) prints a compact one-line-per-symbol summary of types, deriving clauses, exports, signatures, `! {Effects}`, contracts, and callees. See [CLI Reference](/docs/reference/cli).
- **Mechanical quick-fixes:** `ailang check --fix <file>` applies compiler-verified edits in place (missing `! {Effects}`, string `++` → `"${...}"`, `TC_MATCH_002` missing-arm `_?` stubs, `PAR017` `= {`, `PAR018` `let rec` → `letrec`, missing unambiguous stdlib imports). The same edits are exposed as `textDocument/codeAction` in `ailang lsp` — see the [LSP Guide](/docs/guides/lsp).

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
- Built-in type classes (`Num`, `Eq`, `Ord`, `Show`), `deriving (Eq)` for ADTs/records, and `deriving (Json)` for monomorphic ADTs/records

## See Also

- [Examples](/docs/examples) - Working code examples demonstrating syntax
- [CLI Reference](/docs/reference/cli) - `ailang inspect`, `ailang check --fix`, `ailang verify`
- [LSP Guide](/docs/guides/lsp) - Editor integration and `codeAction` quick-fixes
- [Effect System](/docs/reference/effects) - Effect annotations and capabilities
- [Module Reference](/docs/reference/modules) - Module imports and exports
- [Known Limitations](/docs/reference/limitations) - Current limitations and workarounds
- [Architecture: Types](/docs/architecture/types) - Type system internals
