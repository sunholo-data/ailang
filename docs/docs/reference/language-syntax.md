---
title: Language Syntax
sidebar_position: 1
reviewBy: 2026-12-21
---

# AILANG Language Syntax Reference

This reference documents the syntax and features of AILANG v0.6.0. Features marked with a construction sign are planned but not yet implemented.

## Basic Constructs

```ailang
-- Comments use double dash
let x = 5 in x + 1                 -- Immutable binding with scope
\x. x * 2                          -- Lambda function
if x > 0 then "pos" else "neg"     -- Conditional expression
[1, 2, 3]                          -- List literal
{name: "Alice", age: 30}           -- Record literal
(1, "hello", true)                 -- Tuple literal
```

## Module System ✅

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

### Module-Level Bindings ✅

Besides functions, a module can declare top-level `let` bindings — constants shared
across the module's functions. Both the annotated and inferred forms work (module-level
`let` scoping has been supported since v0.5.0):

```ailang
module examples/config

let maxRetries : int = 3        -- with type annotation
let greeting = "hello"           -- inferred; may reference earlier bindings

export func banner() -> string = "${greeting} (retries=${show(maxRetries)})"
```

This is distinct from the expression form `let x = e in body`, which binds only within a
single expression. A top-level `let name = expr` is in scope for every declaration in the
module — use it for shared constants instead of zero-argument functions.

### Function Declarations ✅

```ailang
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

A function body is written in **exactly one** of two styles. Mixing them is a parse error
(`PAR017: ';' is only valid inside { ... } block bodies`):

| Style | Shape | `=`? | `;` between statements? |
|-------|-------|------|--------------------------|
| **Block body** | `func f(...) -> T { s1; s2; result }` | **No `=`** | Yes — statements separated by `;`, last expression is the return value |
| **Expression body** | `func f(...) -> T = expr` | **`=` required** | No — it is a *single* expression (use `let … in …` to sequence) |

```ailang
-- Block body: multiple statements, braces, NO '='
export func render() -> () ! {IO} {
  print("frame");
  println(" done")
}

-- Expression body: single expression, '=' REQUIRED, no ';'
export pure func clamp(n: int) -> int = if n < 0 then 0 else n
```

The trap: writing `= {` (an `=` *and* a brace) then using `;` inside. Pick one — if you need
multiple statements, use a **block body** (`{ … }` with no `=`). If it is one expression, use
`= expr`.

### ✅ Inline Tests (v0.4.5+)
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
pure func sum(xs: List[int]) -> int
  tests [
    ([], 0),
    ([1], 1),
    ([1, 2, 3], 6)
  ]
{
  match xs {
    [] => 0,
    ::(x, rest) => x + sum(rest)
  }
}
```

Inline row inputs and expected values support scalar literals (including negative
numbers), lists, tuples, records, local ADT constructors, and nested combinations.
For example, a row may expect `[2, 3]`, `{value: 2}`, `Allow`, or `Deny("nope")`.
Both arms share the same lightweight evaluator. Binary operators, lambdas,
and other expressions requiring elaboration are rejected by `check` and `test`
with TST001 (unsupported syntax) or TST002 (binary operator). Write the computed
value directly, or use a named test block for those cases.

This is a syntax gate: accepted identifiers and calls can still fail at test time
because of name resolution, types, or capabilities. Imported-module constructors
are not added to the harness environment by this feature. Row arity and tuple-input
disambiguation retain their existing behavior.

Run tests with:
```bash
ailang test examples/factorial.ail
```

**Limitations:**
- Only works for self-contained pure functions
- Functions calling other user-defined functions not yet supported (M-TESTING-DEPS)
- Effect mocking not yet supported (M-TESTING-EFFECTS)

## Lambda Expressions ✅

```ailang
-- Basic lambda syntax
let add = \x y. x + y in
let add5 = add(5) in  -- Partial application (currying)
add5(3)  -- Result: 8

-- Higher-order functions
let compose = \f g x. f(g(x)) in
let double = \x. x * 2 in
let inc = \x. x + 1 in
let doubleThenInc = compose(inc)(double) in
doubleThenInc(5)  -- Result: 11

-- Closures capture environment
let makeAdder = \x. \y. x + y in
let add10 = makeAdder(10) in
add10(5)  -- Result: 15
```

## Pattern Matching ✅

```ailang
-- ADT definition
type Option[a] = Some(a) | None

-- Pattern matching on constructors
match Some(42) {
  Some(x) => x * 2,
  None => 0
}
-- Result: 84

-- Pattern matching on lists
match [1, 2, 3] {
  [] => "empty",
  [x] => "single",
  _ => "multiple"  -- Wildcard pattern
}

-- Pattern matching with guards ✅
match value {
  Some(x) if x > 0 => x * 2,
  Some(x) if x < 0 => 0,
  Some(x) => x,
  None => 0
}

-- Tuple patterns
match (1, "hello") {
  (0, _) => "zero",
  (_, "hello") => "greeting",
  (x, y) => "other"
}
```

### Exhaustiveness ✅

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

## Records ✅

```ailang
-- Record literals
let person = {name: "Alice", age: 30}

-- Field access
person.name  -- "Alice"
person.age   -- 30

-- Nested records
let user = {
  profile: {name: "Bob", email: "bob@example.com"},
  admin: true
}

user.profile.name  -- "Bob"

-- Record subsumption (functions accept supersets)
export func getName(obj: {name: string}) -> string {
  obj.name
}

getName({name: "Alice", age: 30})  -- ✅ Works! Subsumption
getName({name: "Bob", id: 123})     -- ✅ Works! Subsumption
```

### 🚧 Row Polymorphism (Partial - requires AILANG_RECORDS_V2=1)

```ailang
-- Opt-in with environment variable
export AILANG_RECORDS_V2=1

-- Row polymorphism allows extensible records
func getName[r](obj: {name: string | r}) -> string {
  obj.name
}
```

## Block Expressions ✅

```ailang
-- Sequential statements in blocks
{
  let x = 5;
  let y = 10;
  x + y
}  -- Result: 15

-- Blocks with effects
export func main() -> () ! {IO} {
  println("Line 1");
  println("Line 2");
  println("Done")
}

-- Blocks in conditionals
if x > 0 then {
  println("Positive");
  x * 2
} else {
  println("Non-positive");
  0
}
```

## Effects and Capabilities ✅

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

  writeFile("output.txt", response);
  println("Done at ${show(timestamp)}")
}
```

Run with capabilities:
```bash
ailang run --caps IO,FS,Clock,Net --entry processData demo.ail
```

### Available Effects

| Effect | Builtins | Description |
|--------|----------|-------------|
| **IO** | `println`, `print`, `readLine` | Console I/O |
| **FS** | `readFile`, `writeFile`, `fileExists`, `listDir`, `mkdir`, `mkdirAll`, `isDir`, `isFile`, `removeFile` | File system access |
| **Clock** | `now`, `sleep` | Time operations (monotonic, deterministic mode available) |
| **Net** | `httpGet`, `httpPost` | HTTP requests with security (DNS rebinding prevention, IP blocking) |

### 🚧 Quasiquotes (Planned v0.4.0+)

```ailang
-- NOT YET IMPLEMENTED
let query = sql"""
  SELECT * FROM users
  WHERE age > ${minAge: int}
"""

let page = html"""
  <div>${content: SafeHtml}</div>
"""
```

### 🚧 Error Propagation (Planned)

```ailang
-- NOT YET IMPLEMENTED
func readAndProcess() -> Result[Data] ! {FS} {
  let content = readFile("input.txt")?  -- ? operator not yet available
    let response = httpGet(Net, "api.example.com")?
    Ok(process(data, response))
  }
}
```

## Concurrency (CSP)

```ailang
func worker(ch: Channel[Task]) ! {Async} {
  loop {
    let task <- ch       -- Receive from channel
    let result = process(task)
    ch <- result         -- Send to channel
  }
}

parallel {
  spawn { worker(ch1) }
  spawn { worker(ch2) }
}  -- Waits for all spawned tasks
```

## Type Inference

AILANG uses **Hindley-Milner type inference** — you rarely need to write type annotations. The compiler infers the most general type for every expression.

```ailang
-- Types are inferred automatically
let x = 42                       -- x: int
let greeting = "hello"           -- greeting: string
let double = \x. x + x          -- double: int -> int (via Num constraint)
let identity = \x. x            -- identity: a -> a (polymorphic)

-- Type annotations are optional but allowed
export func add(x: int, y: int) -> int ! {}
{ x + y }

-- The compiler catches type errors at compile time
-- let bad = 1 + "hello"  -- ERROR: Cannot unify int with string
```

**Key features:**
- Automatic type inference for `let` bindings, lambdas, and function bodies
- Polymorphic functions (e.g., `\x. x` works for any type)
- Row polymorphism for extensible records and effects
- Type annotations required only for exported function signatures

## Type Classes

```ailang
-- Type class instances (REPL only currently)
let sum = 1 + 2 + 3             -- Works: 6
let calc = 10 * 5 - 20 / 4      -- Works: 45
let greeting = "hello ${place}"     -- Interpolation: "hello world" (++ is list-only)

-- Type-level operations
let eq1 = 42 == 42              -- true
let lt = 5 < 10                 -- true
let double = \x. x + x          -- polymorphic function
```

## See Also

- [Examples](/docs/examples) - Working code examples demonstrating syntax
- [Effect System](/docs/reference/effects) - Effect annotations and capabilities
- [Module Reference](/docs/reference/modules) - Module imports and exports
- [Known Limitations](/docs/reference/limitations) - Current limitations and workarounds
- [Architecture: Types](/docs/architecture/types) - Type system internals