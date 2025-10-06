# AILANG Language Syntax Reference (v0.3.0)

This reference documents the **currently working** syntax in AILANG v0.3.0. Features marked with 🚧 are planned but not yet implemented.

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

### Function Declarations ✅

```ailang
-- Pure function (no effects)
export func add(x: int, y: int) -> int {
  x + y
}

-- Function with effects
export func greet(name: string) -> () ! {IO} {
  println("Hello, " ++ name ++ "!")
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

### 🚧 Inline Tests (Planned)
```ailang
-- NOT YET IMPLEMENTED
export func factorial(n: int) -> int
  tests [
    (0, 1),
    (5, 120)
  ]
{
  if n <= 1 then 1 else n * factorial(n - 1)
}
```

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
  println("Done at " ++ show(timestamp))
}
```

Run with capabilities:
```bash
ailang run --caps IO,FS,Clock,Net --entry processData demo.ail
```

### Available Effects (v0.3.0)

| Effect | Builtins | Description |
|--------|----------|-------------|
| **IO** | `println`, `print`, `readLine` | Console I/O |
| **FS** | `readFile`, `writeFile`, `exists` | File system access |
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

## Type Classes

```ailang
-- Type class instances (REPL only currently)
let sum = 1 + 2 + 3             -- Works: 6
let calc = 10 * 5 - 20 / 4      -- Works: 45
let greeting = "hello" ++ " world"  -- Works: "hello world"

-- Type-level operations
let eq1 = 42 == 42              -- true
let lt = 5 < 10                 -- true
let double = \x. x + x          -- polymorphic function
```