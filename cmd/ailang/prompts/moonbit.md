# MoonBit Teaching Prompt

You are writing MoonBit code for a benchmark. The grader runs your file with `moon run solution.mbt` (single-file mode — no project scaffold needed).

This prompt covers the core MoonBit features the benchmark suite uses. For everything else, MoonBit is ML-family with type inference, ADTs, pattern matching, and generics.

---

## File Structure

A solution is a single `.mbt` file with a `main` function. There is no module/package declaration required.

```moonbit
// optional helper functions
fn square(x : Int) -> Int {
  x * x
}

fn main {
  println("Hello")
}
```

Notes:
- `fn main { ... }` — no parens, no return type annotation
- Statements separated by newlines; semicolons optional
- Last expression in a block is its value

---

## Types and Functions

```moonbit
// Named function with explicit types
fn add(x : Int, y : Int) -> Int {
  x + y
}

// Type inference also works
fn double(x : Int) {
  x * 2
}

// Generics
fn first[T](xs : Array[T]) -> T? {
  if xs.length() == 0 { None } else { Some(xs[0]) }
}

// Lambdas
let inc = fn(x : Int) { x + 1 }
```

Common scalar types: `Int`, `Double`, `Bool`, `String`, `Char`, `Unit`, `Bytes`.
`Int` is 32-bit signed. Use `Int64` for 64-bit.

---

## Output

```moonbit
println("hello")          // prints "hello\n"
print("no newline")        // no newline
println("Value: \{value}") // string interpolation — note the BACKSLASH before brace
```

String interpolation uses `\{expr}`, NOT `${expr}` and NOT `{expr}`. The backslash is required.

---

## Operators

Standard arithmetic and comparison:
- `+ - * / %` (integer modulo)
- `== != < > <= >=`
- `&& || !` (short-circuit boolean)
- `<<` `>>` `&` `|` `^` `~` (bitwise; `~` is bitwise NOT)
- `++` for **string and array concatenation**

```moonbit
let a = 5 << 3      // 40
let b = 16 >> 2     // 4
let c = 12 & 10     // 8
let d = 4 ^ 6       // 2
let s = "Hello, " ++ "World"
let xs = [1, 2] ++ [3, 4]   // [1, 2, 3, 4]
```

---

## Control Flow

```moonbit
// if is an expression
let parity = if n % 2 == 0 { "even" } else { "odd" }

// pattern matching
let label = match n {
  0 => "zero"
  1 => "one"
  _ => "many"
}

// guards
let sign = match n {
  x if x > 0 => "positive"
  x if x < 0 => "negative"
  _ => "zero"
}

// while loop (use only when necessary; prefer functional style)
let mut i = 0
while i < 5 {
  println("\{i}")
  i = i + 1
}

// for-each via iterator
for x in [1, 2, 3] {
  println("\{x}")
}
```

---

## Algebraic Data Types (enums)

```moonbit
enum Option[T] {
  Some(T)
  None
}

enum Result[T, E] {
  Ok(T)
  Err(E)
}

fn safe_div(a : Int, b : Int) -> Option[Int] {
  if b == 0 { None } else { Some(a / b) }
}

// pattern match on ADTs
match safe_div(10, 2) {
  Some(n) => println("Got \{n}")
  None => println("Division by zero")
}
```

`Option[T]` can also be written `T?` (sugar). Both `Some(x)` and `None` are auto-imported.

---

## Records (structs)

```moonbit
struct Person {
  name : String
  age : Int
} derive(Show)

fn main {
  let p = { name: "Alice", age: 30 }
  println("Name: \{p.name}")
  // record update
  let older = { ..p, age: 31 }
  println(older)   // requires derive(Show)
}
```

---

## Collections

### Arrays (mutable, indexable, O(1))

```moonbit
let xs = [1, 2, 3, 4, 5]
xs[0]                     // 1
xs.length()               // 5
xs.iter().fold(init=0, fn(acc, x) { acc + x })   // 15
xs.iter().map(fn(x) { x * x }).collect()         // [1,4,9,16,25]
xs.iter().filter(fn(x) { x % 2 == 0 }).collect() // [2,4]
```

### Lists (immutable, cons-based — separate from Arrays)

```moonbit
let l = @list.from_array([1, 2, 3])
l.length()
l.fold(init=0, fn(acc, x) { acc + x })
```

Most benchmarks should use `Array` unless you specifically need cons-list semantics.

### Maps

```moonbit
let m = { "alice": 30, "bob": 25 }
m["alice"]                      // Some(30)
let m2 = m.add("carol", 28)     // immutable insert
```

---

## Recursion

```moonbit
fn fib(n : Int) -> Int {
  match n {
    0 => 0
    1 => 1
    _ => fib(n - 1) + fib(n - 2)
  }
}

fn fact(n : Int) -> Int {
  if n <= 1 { 1 } else { n * fact(n - 1) }
}
```

---

## Higher-Order Functions

```moonbit
fn map_reduce[T, U](xs : Array[T], f : (T) -> U, g : (U, U) -> U, init : U) -> U {
  xs.iter().map(f).fold(init=init, g)
}

fn main {
  let sum_sq = map_reduce([1, 2, 3, 4, 5], fn(x) { x * x }, fn(a, b) { a + b }, 0)
  println(sum_sq)   // 55
}
```

---

## JSON

```moonbit
let data : Json = { "name": "Alice", "age": 30 }
let s = data.stringify()
println(s)
```

For benchmarks needing JSON manipulation, prefer building the value as a `Json` literal then `stringify()`.

---

## Idiomatic Style Notes

- Prefer pattern matching over chained `if/else if`.
- Prefer immutability (`let`) unless you specifically need `let mut`.
- Use iterators (`.iter().map()...`) over explicit loops for transformation.
- Use `derive(Show)` on structs/enums if you want `println` to print them.
- `Int` overflow wraps silently (it's i32); use `Int64` for large values.

---

## Common Gotchas

- String interpolation uses `\{expr}`, NOT `${expr}`. Forgetting the backslash is the #1 error.
- `fn main { ... }` has **no parens** — `fn main() { ... }` is a SYNTAX ERROR.
- The result of an `if` without `else` is `Unit`; use `else` if you need a value.
- `Array` and `List` are different types — `[1,2,3]` is an `Array`, use `@list.from_array(...)` for a List.
- `==` on records/enums requires `derive(Eq)`.

---

## Quick Reference

| Feature | Syntax |
|---------|--------|
| Function | `fn name(x : T) -> U { ... }` |
| Lambda | `fn(x) { ... }` |
| Let | `let x = expr` |
| Mut let | `let mut x = expr` |
| If expr | `if cond { ... } else { ... }` |
| Match | `match x { Pat => expr; ... }` |
| Array literal | `[1, 2, 3]` |
| String interp | `"text \{expr}"` |
| ADT | `enum Name[T] { Variant(T); Other }` |
| Struct | `struct Name { field : T }` |
| Generic | `fn f[T](x : T) -> T` |
| Concat | `"a" ++ "b"`, `[1] ++ [2]` |

Run with `moon run solution.mbt`.
