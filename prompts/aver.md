# Aver Teaching Prompt

Authoritative sources for this prompt:
- [docs/language.md](https://github.com/jasisz/aver/blob/main/docs/language.md) (official language reference)
- [docs/pushback.md](https://github.com/jasisz/aver/blob/main/docs/pushback.md) (common AI-codegen mistakes)
- [examples/core/](https://github.com/jasisz/aver/tree/main/examples/core) (canonical example programs)

You are writing Aver code for a benchmark. The grader runs your file with `aver run solution.av`.

---

## What Aver Is (and Is NOT)

Aver borrows functional discipline — immutability, purity, pattern matching, recursion — but **deliberately omits** much of the abstraction machinery of modern FP. The constraints are the point.

**Aver does NOT have:**
- `if`/`else`. Use `match` on booleans: `match cond { true -> ...; false -> ... }`.
- Anonymous functions / lambdas. Define a named top-level `fn` and pass its name.
- Higher-order list functions (`map`, `filter`, `fold` in the std lib are intentionally absent). Use **explicit recursion** + pattern matching.
- Mutation. All bindings are immutable.
- Loops (no `while`/`for`). Use recursion.
- Bitwise operators (`&`, `|`, `^`, `<<`, `>>`, `~`). Not in the language.
- Implicit type promotion. `1 + 1.0` is a type error. Use `Float.fromInt(1) + 1.0`.
- `%` operator. Use `Int.mod(a, b)` as a function call.
- Generics. Container types like `List<T>`, `Result<T,E>` are built-in, but you cannot declare your own generic functions.
- Closures. `Fn(A) -> B` parameters exist only for callbacks; not a general composition tool.
- Null. Use `Option.Some(v)` / `Option.None`.
- Exceptions. Use `Result.Ok(v)` / `Result.Err(e)`.

**Aver DOES have:**
- Algebraic types (`type` for sums, `record` for products)
- Pattern matching as the only branching construct
- Effects in the signature: `! [Console.print, Disk.readText]`
- Named, namespaced constructors: `Result.Ok`, `Option.Some`, `Shape.Circle`
- String interpolation: `"text {expr}"`
- `verify` blocks for inline test cases
- `decision` blocks for architectural decision records embedded in code

---

## File Structure

A typical Aver file uses a `module` declaration. For minimal single-file solutions, the body can be just `fn main()` — but `module` + `effects` declarations help the type checker.

```aver
module Hello
    intent =
        "Demonstrates basic Aver syntax."
    effects [Console]

fn main() -> Unit
    ! [Console.print]
    Console.print("Hello")
```

- File extension is **`.av`**
- Function bodies use **indentation** (no braces)
- The last expression in a body is the return value

---

## Output

```aver
Console.print("hello")          // prints to stdout, adds newline in CLI mode
```

**Console.print takes `String` ONLY.** Converting other types is the caller's job — use string interpolation:

```aver
Console.print("Value: {n}")            // OK — interpolation converts {n}
Console.print(n)                       // ❌ TYPE ERROR if n is not String
Console.print("{42}")                  // OK — interpolation
Console.print("{1 + 2}")               // OK — expressions allowed
Console.print("{showListInt(xs)}")     // OK — calls user-defined formatter
```

Required effect: `! [Console.print]`.

---

## Types

| Category | Types |
|----------|-------|
| Primitive | `Int`, `Float`, `String`, `Bool`, `Unit` |
| Compound | `Result<T,E>`, `Option<T>`, `List<T>`, `Vector<T>`, `Map<K,V>`, `(A, B, ...)`, `Fn(A) -> B`, `Fn(A) -> B ! [Effect]` |
| User-defined sum | `type Shape` with variants `Shape.Circle(Float)`, `Shape.Rect(Float, Float)` |
| User-defined product | `record User` with field declarations |

There is **no Set type** — use `Map<T, Unit>`.

---

## Bindings (Immutable)

```aver
name = "Alice"
age: Int = 30
xs: List<Int> = []
```

Empty list literal without a type annotation is a type error. Duplicate bindings in the same scope are a type error.

---

## Match (the ONLY Branching Construct)

There is no `if`/`else`. Use `match` on a boolean for binary conditions, on values for n-way dispatch.

```aver
fn parity(n: Int) -> String
    match Int.mod(n, 2)
        0 -> "even"
        _ -> "odd"

fn classify(c: Float) -> String
    match c <= 0.0
        true  -> "freezing"
        false -> match c < 20.0
            true  -> "cold"
            false -> "warm"
```

**Match arm body MUST be on the same line as `->`.** If you need multi-line logic, extract to a named function:

```aver
// ❌ WRONG — multi-line match arm body
match xs
    [] -> 0
    _ ->
        h = head(xs)
        h + sum(tail(xs))

// ✅ RIGHT — extract or inline
match xs
    [] -> 0
    [h, ..t] -> h + sum(t)
```

Patterns:

```aver
match value
    42 -> "exact"                          // literal
    _ -> "anything"                        // wildcard
    x -> "bound to {x}"                    // identifier binding
    [] -> "empty list"                     // empty list
    [h, ..t] -> "head {h}"                 // list cons
    Result.Ok(v) -> "ok: {v}"              // constructor (qualified!)
    Result.Err(e) -> "err: {e}"
    (a, b) -> "pair: {a}, {b}"             // tuple destructuring
```

Constructor patterns are **always** qualified: `Result.Ok`, `Option.None`, `Shape.Circle`. No bare `Ok` / `Err` / `Some` / `None`.

---

## Functions

```aver
fn add(a: Int, b: Int) -> Int
    ? "Adds two integers."
    a + b

fn greet(name: String) -> Unit
    ? "Prints a greeting."
    ! [Console.print]
    Console.print("Hello, {name}")

fn safeDiv(a: Int, b: Int) -> Result<Int, String>
    match b
        0 -> Result.Err("division by zero")
        _ -> Result.Ok(a / b)
```

- `? "..."` — optional prose description (part of signature)
- `! [Effect]` — effect declaration, required if function uses any effect
- Function bodies use indentation
- The last expression is the return value
- Functions are top-level (no nested fn, no lambdas)

---

## Recursion (Required for Iteration)

Aver has no loops and no `List.map`/`filter`/`fold`. Iterate with recursion + pattern matching.

```aver
fn sum(xs: List<Int>) -> Int
    match xs
        [] -> 0
        [h, ..t] -> h + sum(t)

fn length(xs: List<Int>) -> Int
    match xs
        [] -> 0
        [_, ..t] -> 1 + length(t)

fn doubleAll(xs: List<Int>) -> List<Int>
    match xs
        [] -> []
        [h, ..t] -> List.prepend(h * 2, doubleAll(t))
```

Built-in `List.*` functions you CAN use (from examples):
`List.len`, `List.prepend`, `List.take`, `List.drop`, `List.concat`, `List.reverse`, `List.contains`.

There is no `List.map`, `List.filter`, or `List.fold` — Aver's stdlib intentionally omits them.

---

## Records

```aver
record User
    name: String
    age: Int

fn main() -> Unit
    ! [Console.print]
    u = User(name = "Alice", age = 30)
    Console.print("Name: {u.name}, Age: {u.age}")
    older = User.update(u, age = 31)
    Console.print("Updated: {older.age}")
```

Records use **named arguments** in construction. Variants use **positional** arguments.

---

## Sum Types

```aver
type Shape
  Circle(Float)
  Rect(Float, Float)
  Point

fn area(s: Shape) -> Float
    match s
        Shape.Circle(r) -> r * r * 3.14159
        Shape.Rect(w, h) -> w * h
        Shape.Point -> 0.0
```

Constructors are namespaced (`Shape.Circle`). Zero-arg constructors are bare singletons (`Shape.Point`, `Option.None`).

---

## Operators

- Arithmetic: `+ - * /` — operands MUST match types. No implicit promotion.
- Comparison: `== != < > <= >=`
- Modulo: `Int.mod(a, b)` — NOT `%`
- Float math: `Float.fromInt(n)`, `Int.fromFloat(f)`, `Float.abs`, etc.
- String concat: use interpolation: `"{a}{b}"`
- Error propagation: `expr?` unwraps `Result.Ok`, propagates `Result.Err`

**NO bitwise operators.** `&`, `|`, `^`, `<<`, `>>`, `~` are syntax errors.

---

## String Interpolation

Single braces, no backslash:

```aver
greeting = "Hello, {name}! Age = {age}"
result = "sum = {sum(xs)}, len = {List.len(xs)}"
```

NOT `${name}`, NOT `\{name}`. Just `{name}`.

---

## Effects

Every effectful function MUST declare its effects on the line after the signature:

```aver
fn process(path: String) -> Unit
    ! [Disk.readText, Console.print]
    content = Disk.readText(path)?
    Console.print(content)
```

Common effects: `Console.print`, `Console.error`, `Disk.readText`, `Disk.writeText`, `Random.int`, `Random.float`, `Time.unixMs`, `Args.get`.

Namespace shorthand: `! [Console]` covers all `Console.*`.

---

## Verify Blocks (Optional but Idiomatic)

```aver
fn add(a: Int, b: Int) -> Int
    a + b

verify add
    add(0, 0) => 0
    add(2, 3) => 5
```

Run with `aver verify file.av`. For benchmarks that just print to stdout, you do NOT need verify blocks — just write `main`.

---

## Decision Blocks

```aver
decision UseLinearScan
    date = "2026-05-21"
    reason =
        "Single linear pass for O(n) time and O(1) space."
    chosen = "LinearScan"
```

First-class top-level syntax for architectural decision records.

---

## Critical Reminders

1. **No `if`/`else`** — use `match cond { true -> ...; false -> ... }`.
2. **No lambdas / anonymous functions** — define a named top-level `fn`.
3. **No bitwise operators** — they don't exist.
4. **No higher-order map/filter/fold** — use explicit recursion.
5. **Match arm bodies on the same line as `->`** — extract complex logic to functions.
6. **`Console.print` takes String** — interpolate to convert: `Console.print("{value}")`.
7. **Constructors always namespaced** — `Result.Ok` not `Ok`, `Option.None` not `None`.
8. **String interpolation is `{expr}`** — single braces, no backslash.
9. **Modulo is `Int.mod(a, b)`** — `%` is a syntax error.
10. **No implicit numeric promotion** — use `Float.fromInt` / `Int.fromFloat`.
11. **File extension is `.av`**.

Run with `aver run solution.av`.
