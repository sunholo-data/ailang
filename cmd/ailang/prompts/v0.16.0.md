# AILANG v0.16.0 - AI Teaching Prompt (with IFC Labels)

AILANG is a **pure functional language** with Hindley-Milner type inference and algebraic effects. Write code using **recursion** (no loops), **pattern matching**, and **explicit effect declarations**.

**⚠️ AILANG is NOT Python!** Do not use Python syntax like `def`, `for`, `while`, `class`, or `if x:`. See "What AILANG Does NOT Have" below.

## Required Program Structure

```ailang
module myproject/mymodule

export func main() -> () ! {IO} {
  println(show(42))  -- println is in prelude, no import needed!
}
```

**Rules:**
1. First line MUST be `module path/name` matching the file path
2. Use `println(show(value))` for numbers, `println(str)` for strings. **Prefer `show` over `intToStr`/`floatToStr`** — `show` works on ALL types
3. **Pick ONE function-body style per function and stick to it:**
   - **Block body** `func f() ! {IO} { let x = a; let y = b; finalExpr }` — semicolons between lets, NO `in`
   - **Expression body** `func f() ! {IO} = let x = a in let y = b in finalExpr` — `in` keyword, NO semicolons
   - **Common error**: `func f() = let x = a; ...` is a parse error (`unexpected token: ;`). Either wrap the body in `{ }` and use semicolons, or use `let .. in ..` chains.
4. No loops — use recursion
5. **Output raw AILANG code only** — no markdown fences (`` ``` ``), no prose, no JSON wrappers. The first line must be `module ...`.

## println vs print

| Function | Import Required? | Behavior |
|----------|-----------------|----------|
| `println` | NO (prelude) | WITH newline |
| `print` | YES (`import std/io (print)`) | NO newline |

```ailang
-- println works without import (prelude provides it)
println("A"); println("B")  -- Output: A\nB\n

-- print requires import for no-newline output
import std/io (print)
print("A"); print("B")      -- Output: AB
```

**Use `println` for most output.** Use `print` ONLY when building output on one line.

## CLI Exploration (USE THIS!)

**Before writing helper functions, check if the stdlib already has what you need:**

```bash
ailang docs --list              # List all stdlib modules
ailang docs std/string          # Show ALL exports with signatures
ailang docs std/list            # Show list operations
ailang builtins list            # All registered builtins
ailang check file.ail           # Type-check without running
ailang examples search "recursion"  # Find working code examples
```

## Quick Reference Examples

**Print calculation:**
```ailang
module benchmark/solution
import std/io (println)
export func main() -> () ! {IO} = println(show(5 % 3))
```

**If-then-else (NO braces!):**
```ailang
module benchmark/solution
import std/io (println)
export func main() -> () ! {IO} {
  let x = 10;
  let msg = if x > 0 then "positive" else "not positive";
  println(msg)
}
```

**HTTP GET:**
```ailang
module benchmark/solution
import std/net (httpGet)
export func main() -> () ! {IO, Net} = println(httpGet("https://example.com"))
```

**HTTP POST with JSON:**
```ailang
module benchmark/solution
import std/net (httpPost)
import std/json (encode, jo, kv, js, jnum)
export func main() -> () ! {IO, Net} {
  let data = encode(jo([kv("message", js("hello")), kv("count", jnum(42.0))]));
  println(httpPost("https://httpbin.org/post", data))
}
```

**Filter list recursively:**
```ailang
module benchmark/solution
export func filter(people: [{name: string, age: int}], minAge: int) -> [{name: string, age: int}] =
  match people {
    [] => [],
    p :: rest => if p.age >= minAge then p :: filter(rest, minAge) else filter(rest, minAge)
  }
```

**Record update (immutable):**
```ailang
module benchmark/solution
export func main() -> () ! {IO} {
  let person = {name: "Alice", age: 30, city: "NYC"};
  let older = {person | age: person.age + 1};         -- Update one field
  let moved = {older | city: "SF"};                   -- Chain updates
  println("${person.name}, ${show(person.age)}, ${person.city}");
  println("${older.name}, ${show(older.age)}, ${older.city}");
  println("${moved.name}, ${show(moved.age)}, ${moved.city}")
}
```

**Open records (width subtyping):**
```ailang
module benchmark/solution

-- EXACT record: only accepts {name: string}, rejects extra fields
pure func getNameExact(p: {name: string}) -> string = p.name

-- OPEN record with | r: accepts extra fields
pure func getName(p: {name: string | r}) -> string = p.name

-- OPEN record with ... sugar (equivalent to | r with fresh variable)
pure func getEmail(u: {email: string, ...}) -> string = u.email

export func main() -> () ! {IO} {
  -- Exact: only works with exact shape
  println(getNameExact({name: "Alice"}));

  -- Open: accepts any record with at least name field
  println(getName({name: "Bob", age: 30}));
  println(getName({name: "Charlie", age: 25, city: "NYC"}));

  -- Ellipsis sugar: same behavior
  println(getEmail({email: "test@example.com", name: "Test", id: 123}))
}
```

**AI effect (call LLM):**
```ailang
module benchmark/solution
import std/ai (call, callJson, callJsonSimple)
export func main() -> () ! {IO, AI} = println(call("What is 2+2?"))
```

## What AILANG Does NOT Have

| Invalid | Use Instead |
|-----------|----------------|
| `for`/`while` loops | Recursion |
| `[x*2 for x in xs]` | `map(\x. x*2, xs)` or recursion |
| `var`/`let mut` | Immutable `let` bindings |
| `list.map()` | `map(f, list)` |
| `import "std/io"` | `import std/io (println)` |
| `{"key": "val"}` | `jo([kv("key", js("val"))])` |
| `f(a)(b)` for multi-arg func | `f(a, b)` - multi-arg funcs use commas |
| `if x { ... }` | `if x then ... else ...` - NO braces! |
| mixing `let x = e in` with `;` | Use ONE style consistently |
| `let (x, y) = tuple` | Use `match tuple { (x, y) => ... }` |
| `\(a, b). body` pair syntax | Use `func(a: T, b: U) -> R { body }` |
| nested `func f(...) =` | Use `let f = \x. body` for nested functions |
| `!condition` | Both `!x` and `not x` work — prefer `not` for readability |
| `concat(a, b)` for strings | `"${a}${b}"` interpolation — `++` is list-only in v0.13.0+ |
| `concat(a, b)` for lists | `a ++ b` |

## Let Bindings: Block Style vs Expression Style

**Rule: Inside `{ }` blocks, use SEMICOLONS. With `=` bodies, use `in`.**

```ailang
-- Block style: curly braces + semicolons
export func main() -> () ! {IO} {
  let x = 1;
  let y = 2;
  println(show(x + y))
}

-- Expression style: equals + in
export func main() -> () ! {IO} =
  let x = 1 in
  let y = 2 in
  println(show(x + y))

-- WRONG: Using `in` inside `{ }` block causes scope errors
export func main() -> () ! {IO} {
  let x = 1 in   -- DON'T use `in` inside blocks!
  let y = 2;     -- ERROR: x out of scope
  println(show(x + y))
}
```

**Simple rule: See `{` -> use `;`. See `=` -> use `in`.**

## Function Calls: Multi-arg vs Curried (CRITICAL)

**Two styles with DIFFERENT call syntax:**

```ailang
-- MULTI-ARG FUNC: Call with commas f(a, b)
func add(a: int, b: int) -> int = a + b
let result = add(3, 4)      -- CORRECT: 7
-- let bad = add(3)(4)      -- WRONG: arity mismatch!

-- CURRIED LAMBDA: Call with chained f(a)(b)
let addC = \a. \b. a + b
let result = addC(3)(4)     -- CORRECT: 7
let add10 = addC(10)        -- Partial application!
-- let bad = addC(3, 4)     -- WRONG: arity mismatch!
```

**Rule: Match call style to definition style:**
| Definition | Call Style | Partial App? |
|------------|------------|--------------|
| `func f(a, b) = ...` | `f(a, b)` | No |
| `let f = \a. \b. ...` | `f(a)(b)` | Yes |

**Higher-order with curried lambdas:**
```ailang
module benchmark/solution
import std/io (println)

export func main() -> () ! {IO} {
  let compose = \f. \g. \x. f(g(x));
  let double = \x. x * 2;
  let addOne = \x. x + 1;
  let doubleThenAdd = compose(addOne)(double);  -- Curried: chain calls
  println(show(doubleThenAdd(5)))
}
```

**Using std/list foldl (IMPORTANT - use inline func syntax!):**
```ailang
module benchmark/solution
import std/io (println)
import std/list (foldl)

-- std/list foldl signature: foldl(f: (acc, elem) -> acc, initial, list)
-- RECOMMENDED: Use inline func syntax for reliable type inference

export func main() -> () ! {IO} {
  -- MOST RELIABLE: inline func with explicit types
  let sum = foldl(func(acc: int, x: int) -> int { acc + x }, 0, [1,2,3,4,5]);
  println(show(sum))
}
```

**foldl syntax options:**
```ailang
-- RECOMMENDED: space-separated multi-param lambda (concise)
foldl(\acc x. acc + x, 0, xs)

-- VERBOSE: inline func with explicit types (also works)
foldl(func(acc: int, x: int) -> int { acc + x }, 0, xs)

-- WRONG: curried lambda (arity mismatch error!)
foldl(\acc. \x. acc + x, 0, xs)
```

**Sorting with sortBy:**
```ailang
import std/list (sortBy)

-- Comparator returns: negative if a < b, 0 if equal, positive if a > b
func cmpInt(a: int, b: int) -> int = a - b

let sorted = sortBy(cmpInt, [3, 1, 4, 1, 5])  -- [1, 1, 3, 4, 5]

-- Reverse sort: swap arguments
func cmpDesc(a: int, b: int) -> int = b - a
let descending = sortBy(cmpDesc, [3, 1, 4])   -- [4, 3, 1]
```

## Recursive Lambdas (letrec)

**Use `letrec` when a lambda needs to call itself:**

```ailang
module benchmark/solution

export func main() -> () ! {IO} {
  -- WRONG: `let` for self-referential lambda
  -- let factorial = \n. if n <= 1 then 1 else n * factorial(n - 1);  -- ERROR: undefined 'factorial'

  -- RIGHT: use `letrec` for recursive lambdas
  letrec factorial = \n. if n <= 1 then 1 else n * factorial(n - 1);
  println(show(factorial(5)))  -- 120
}
```

**When to use `letrec` vs `func`:**
| Case | Use |
|------|-----|
| Top-level recursive function | `func f(...) = ...` (preferred) |
| Recursive lambda inside block | `letrec f = \x. ... f(...)` |
| Non-recursive lambda | `let f = \x. ...` |

**Mutual recursion also works:**
```ailang
letrec isEven = \n. if n == 0 then true else isOdd(n - 1);
letrec isOdd = \n. if n == 0 then false else isEven(n - 1);
println(show(isEven(4)))  -- true
```

## Type Annotations for Higher-Order Functions

**Annotate function types with parentheses around arrow types:**

```ailang
-- Single-argument function type
let double: int -> int = \x. x * 2

-- Higher-order: function taking a function
let apply: (int -> int) -> int -> int = \f. \x. f(x)

-- Multiple function parameters
let compose: (int -> int) -> (int -> int) -> int -> int = \f. \g. \x. f(g(x))
```

**Type annotation on multi-arg named function:**
```ailang
func twice(f: int -> int, x: int) -> int = f(f(x))
```

## Syntax Reference

| Construct | Syntax |
|-----------|--------|
| Module | `module path/name` |
| Import | `import std/io (println)` |
| Import alias | `import std/list as L` |
| Function | `export func name(x: int) -> int ! {IO} { body }` |
| Lambda | `\x. x * 2` |
| Recursive lambda | `letrec f = \x. ... f(x) ...` |
| Pattern match | `match x { 0 => a, n => b }` (use `=>`, commas between arms) |
| ADT | `type Tree = Leaf(int) \| Node(Tree, int, Tree)` |
| ADT with Eq | `type Color = Red \| Green \| Blue deriving (Eq)` |
| Record | `{name: "A", age: 30}` |
| Record update | `{base \| field: val}` |
| Open record type | `{name: string \| r}` or `{name: string, ...}` |
| List cons | `x :: xs` or `::(x, xs)` |
| Array literal | `#[1, 2, 3]` |
| Effect | `! {IO, FS, Net}` after return type |

## Effects (Side Effects Must Be Declared)

**Every function performing I/O must declare effects in the signature!**

```ailang
-- Pure (no effects) - use `pure func` or omit effect annotation
pure func add(x: int, y: int) -> int = x + y

-- IO effect - for print/println/readLine
func greet(name: string) -> () ! {IO} = println("Hello ${name}")

-- Single effect with return value
func ask(prompt: string) -> string ! {IO} {
  println(prompt);
  readLine()
}

-- Multiple effects
func process(path: string) -> () ! {IO, FS} {
  let content = readFile(path);
  println(content)
}

-- Main function typically needs effects
export func main() -> () ! {IO} {
  println("Hello, World!")
}

-- Exit with explicit code (for CLI tools)
import std/io (println, exit)
export func main() -> () ! {IO} {
  println("error: invalid argument");
  exit(1)
}

-- Main with file system access
export func main() -> () ! {IO, FS} {
  let content = readFile("data.txt");
  println(content)
}

-- Main with network
export func main() -> () ! {IO, Net} {
  let body = httpGet("https://example.com");
  println(body)
}

-- Main with CLI arguments (getArgs returns [string], needs Env)
-- Run: ailang run --caps IO,FS,Env --entry main program.ail -- arg1 arg2
export func main() -> () ! {IO, FS, Env} {
  let args = getArgs();
  match args {
    filename :: _ => {
      let content = readFile(filename);
      println(content)
    },
    _ => println("Usage: program <filename>")
  }
}

-- Reading lines from stdin (readLine reads one line, returns "" at EOF)
-- Run: printf "hello\nworld\n" | ailang run --caps IO --entry main program.ail
func loop() -> () ! {IO} {
  let line = readLine();
  if line == "" then ()
  else {
    println("Got: ${line}");
    loop()
  }
}
```

**Common effect combinations:**
| Task | Required Effects |
|------|------------------|
| Print output | `! {IO}` |
| Read files | `! {FS}` (add IO if also printing) |
| HTTP requests | `! {Net}` (add IO if also printing) |
| Environment vars | `! {Env}` |
| CLI arguments | `! {Env}` (use `getArgs()` from `std/env`) |
| Read from stdin | `! {IO}` (use `readLine()` from `std/io`) |
| AI calls | `! {AI}` |
| Run external commands | `! {Process}` or `! {IO, Process}` |
| Exit with code | `! {IO}` (use `exit(code)` from `std/io`) |
| Full program | `! {IO, FS, Net}` as needed |

**Effect errors and how to fix:**
```
Error: Effect checking failed for function 'main'
  Function uses effects not declared in signature
  Missing effects: IO
```

**Fix:** Add the missing effect to your function signature:
```ailang
-- WRONG: missing IO effect
export func main() -> () { println("hi") }

-- CORRECT: declare IO effect
export func main() -> () ! {IO} { println("hi") }
```
```

| Effect | Functions | Import |
|--------|-----------|--------|
| `IO` | `print`, `println`, `readLine`, `exit` | `std/io` (print is builtin) |
| `FS` | `readFile`, `writeFile`, `fileExists`, `listDir`, `mkdir`, `mkdirAll`, `isDir`, `isFile`, `removeFile`, `_zip_*` | `std/fs`, `std/zip` |
| `Net` | `httpGet`, `httpPost`, `httpRequest` | `std/net` |
| `Env` | `getArgs`, `getEnv`, `getEnvOr` | `std/env` |
| `AI` | `call`, `callJson`, `callJsonSimple` | `std/ai` |
| `Debug` | `log`, `check` | `std/debug` |
| `Process` | `exec` | `std/process` |
| `SharedMem` | `_sharedmem_get`, `_sharedmem_put`, `_sharedmem_cas` | builtins |
| `SharedIndex` | `_sharedindex_upsert`, `_sharedindex_find_simhash` | builtins |

### Parameterised Effects (v0.15.0+)

Effect rows can carry `[key=value]` parameters that ride alongside the
effect name. Phase 1 ships the syntax and unification rules; the
runtime currently treats all modes identically (mode-aware dispatch
lands later).

```ailang
-- Bare !{Rand} desugars to !{Rand[mode=os]} via the default-mode table.
func roll_d6() -> int ! {Rand} = rand_int(1, 6)

-- Same effect signature as bare !{Rand}; unifies via default-desugar.
func roll_d20() -> int ! {Rand[mode=os]} = rand_int(1, 20)

-- Distinct mode — does NOT unify with mode=os under invariant rules.
func deterministic() -> int ! {Rand[mode=seeded]} = {
  rand_seed(42);
  rand_int(1, 100)
}
```

**Rules:**
- Bare `!{Rand}` is shorthand for `!{Rand[mode=os]}`.
- Different param values do NOT unify: `!{Rand[mode=os]}` and
  `!{Rand[mode=seeded]}` are distinct effect rows.
- v0.15.0 ships two default-mode entries: `Rand → mode=os` and
  `AI → mode=fixed`. Other effects (`IO`, `FS`, `Net`, `Env`, ...)
  have no params yet — write `!{IO}`, not `!{IO[...]}`.
- Mode set is closed (compiler-known per effect); user code cannot
  introduce new modes.

See [Parameterised Effects guide](https://ailang.sunholo.com/docs/guides/parameterised-effects).

**AI modes (v0.15.0+):**

```ailang
-- Bare !{AI} desugars to !{AI[mode=fixed]} (direct provider call).
func summarize(text: string) -> string ! {AI} = call(text)

-- !{AI[mode=routeable]} opts into runtime provider routing
-- (e.g. OpenRouter). Skips the --allow-routing CLI gate.
func summarize_routed(text: string) -> string ! {AI[mode=routeable]} = call(text)
```

Rules:
- Different param values do NOT unify: `!{AI[mode=fixed]}` and
  `!{AI[mode=routeable]}` are distinct.
- Bare `!{AI}` continues to work; existing programs unchanged.
- See [AI Routing guide](https://ailang.sunholo.com/docs/guides/ai-routing) for routing flags.

### Debug Effect — True Ghost Effect for Logging

**Debug is a ghost effect** — fully invisible to callers, zero-cost in release mode. Use `Debug` instead of `IO` (println) for logging in packages. No effect declarations, no --caps, no config needed:

```ailang
import std/debug (log, check)

-- No ! {Debug} needed — ghost effect is invisible to callers
func process(data: string) -> string ! {Net} {
  log("processing: ${data}");
  check(length(data) > 0, "data must not be empty");
  httpGet("https://example.com/${data}")
}
```

Or use the `sunholo/logging` package for structured JSON logging:
```ailang
import pkg/sunholo/logging/logger (info, warn, infoWith)
import std/json (kv, js, jnum)

func handleRequest(path: string) -> Response ! {Net} {
  infoWith("Request", [kv("path", js(path))]);
  httpGet("https://api.example.com${path}")
}
```

**Key rules:**
- **Ghost effect**: No `! {Debug}` in signatures, no `[effects].max`, no `--caps Debug` needed
- **Write-only**: `log` and `check` write to host context; AILANG code cannot read it
- **Assertions don't throw**: `check(false, msg)` records failure but continues
- **Release mode erases**: `--release` flag removes all Debug calls (zero cost)
- **Log level filtering**: `--log-level warn` suppresses INFO/DEBUG output
- **Output**: Debug logs printed to stderr as structured JSON after execution

## Standard Library

**Auto-imported (no import needed):**
- `print` (entry modules only)
- `show(x)` - convert to string
- Comparison operators: `<`, `>`, `<=`, `>=`, `==`, `!=`

**Common imports:**
```ailang
import std/io (println, readLine)
import std/fs (readFile, writeFile, fileExists, listDir, mkdir, mkdirAll, isDir, isFile, removeFile)
import std/env (getArgs, getEnv, getEnvOr)
import std/net (httpGet, httpPost, httpRequest)
import std/json (encode, decode, get, getString, getNumber, getInt, getBool, getArray, getObject, asString, asNumber, asArray)
import std/json (filterStrings, filterNumbers, allStrings, allNumbers, getStringArrayOrEmpty)
import std/list (map, filter, foldl, length, concat, sortBy, take, drop, nth, last, any, findIndex, flatMap, zipWith, mapE, filterE, foldlE, flatMapE, forEachE)
import std/string (split, chars, trim, stringToInt, stringToFloat, contains, find, substring, intToStr, floatToStr, join, startsWith, endsWith, length, toUpper, toLower, compare, repeat)
import std/math (floatToInt, intToFloat, floor, ceil, round, sqrt, pow, abs_Float, abs_Int)
import std/ai (call, callJson, callJsonSimple)
import std/sem (make_frame_at, store_frame, load_frame, update_frame)
import std/option (Option, Some, None)
import std/result (Result, Ok, Err)
import std/bytes (fromString, toString, toBase64, fromBase64, length, slice)
import std/stream (connect, transmit, transmitBinary, onEvent, runEventLoop, disconnect, sseConnect, ssePost, withSSE, sourceOfConn, asyncReadStdinLines, asyncExecProcess, selectEvents, StreamConn, StreamSource, StreamEvent, Message, Binary, Opened, Closed, StreamError, Ping, SSEData, SourceText, SourceBytes)
import std/process (exec, spawnProcess, writeProcessStdin, closeProcessStdin, ProcessHandle)
import std/zip (_zip_listEntries, _zip_readEntry, _zip_readEntryBytes)
import std/xml (_xml_parse, _xml_findAll, _xml_findFirst, _xml_getText, _xml_getAttr, _xml_getChildren, _xml_getTag)
```

**List functions** (std/list) — all polymorphic, fully stable, **use these instead of hand-rolling recursion**:
- `map(f, xs)` - Apply function to each element
- `filter(p, xs)` - Keep elements matching predicate
- `foldl(f, acc, xs)` - Left fold (use `func(acc: T, x: U) -> T { ... }` for f)
- `foldr(f, acc, xs)` - Right fold
- `length(xs)` - List length
- `head(xs) -> Option[a]` - First element (None if empty)
- `tail(xs)` - All elements except first
- `reverse(xs)` - Reverse a list
- `concat(xs, ys)` - Concatenate two lists
- `zip(xs, ys)` - Pair elements from two lists
- `sortBy(cmp, xs)` - Sort with comparator
- `take(n, xs)` - Take first n elements
- `drop(n, xs)` - Drop first n elements
- `nth(xs, idx) -> Option[a]` - Get element at index (0-based), None if out of bounds
- `last(xs) -> Option[a]` - Get last element, None if empty
- `any(p, xs) -> bool` - Check if any element satisfies predicate
- `findIndex(p, xs) -> Option[int]` - Find index of first matching element
- `flatMap(f, xs)` - Apply f to each element, flatten results (pure)
- `zipWith(f, xs, ys)` - Combine two lists element-wise with function f

**Effectful list combinators** (v0.7.3) — use these instead of hand-rolling recursive traversals:
- `mapE(f, xs)` - Effectful map: apply effectful function to each element
- `filterE(p, xs)` - Effectful filter: keep elements matching effectful predicate
- `foldlE(f, acc, xs)` - Effectful left fold
- `flatMapE(f, xs)` - Effectful flatMap: apply f, flatten results
- `forEachE(f, xs)` - Effectful forEach: apply f for side-effects, discard results

All effectful combinators are **effect-polymorphic** — they work with any effect (`IO`, `FS`, `AI`, etc.) and guarantee **left-to-right evaluation order**.

**Prefer `map`/`filter`/`foldl` over manual recursion:**
```ailang
import std/list (map, filter, foldl, mapE, filterE, foldlE)

-- Pure: use map/filter/foldl
let doubled = map(\x. x * 2, [1, 2, 3])  -- [2, 4, 6]
let evens = filter(\x. x % 2 == 0, [1, 2, 3, 4])  -- [2, 4]
let sum = foldl(func(acc: int, x: int) -> int { acc + x }, 0, [1, 2, 3])  -- 6

-- Effectful: use mapE/filterE/foldlE (replaces hand-rolled recursion!)
let results = mapE(\x. { println("processing ${show(x)}"); x * 2 }, [1, 2, 3]);
let checked = filterE(\x. { println("checking ${show(x)}"); x > 2 }, [1, 2, 3, 4]);
let total = foldlE(func(acc: int, x: int) -> int ! {IO} { println("fold"); acc + x }, 0, [10, 20]);
```

**Note on foldlE:** The callback must use `func(acc: T, x: U) -> T ! {Effect} { ... }` syntax (not curried `\acc. \x.`).

**String functions** (std/string) — run `ailang docs std/string` for full list:
- `length(s) -> int` - String length
- `contains(hay, needle) -> bool` - Check if string contains substring
- `startsWith(s, prefix) -> bool` - Check if string starts with prefix
- `endsWith(s, suffix) -> bool` - Check if string ends with suffix
- `find(hay, needle) -> int` - Index of substring, or -1 if not found
- `substring(s, start, end) -> string` - Extract substring
- `split(s, delim) -> [string]` - Split string by delimiter
- `chars(s) -> [string]` - Convert string to list of single-character strings (Unicode-aware)
- `trim(s) -> string` - Trim whitespace
- `toUpper(s) -> string` / `toLower(s) -> string` - Case conversion
- `stringToInt(s) -> Option[int]` - Parse integer
- `stringToFloat(s) -> Option[float]` - Parse float
- `intToStr(n) -> string` - Convert int to string (prefer `show` for simple cases)
- `floatToStr(f) -> string` - Convert float to string (prefer `show` for simple cases)
- `join(delim, xs) -> string` - Join list of strings with delimiter
- `repeat(s, n) -> string` - Repeat string n times

**Math functions** (std/math):
- `floatToInt(x) -> int` - Convert float to int (truncates toward zero)
- `intToFloat(x) -> float` - Convert int to float
- `floor(x) -> float`, `ceil(x) -> float`, `round(x) -> float` - Rounding
- `sqrt(x)`, `pow(x, y)`, `abs_Float(x)`, `abs_Int(x)` - Math operations

**Bytes functions** (std/bytes) — pure binary data operations:
- `fromString(s) -> bytes` - UTF-8 encode string to bytes
- `toString(b) -> string` - Decode bytes to UTF-8 string
- `toBase64(b) -> string` - Base64 encode
- `fromBase64(s) -> Option[bytes]` - Base64 decode (None if invalid)
- `length(b) -> int` - Byte length
- `slice(b, start, len) -> Option[bytes]` - Extract subsequence (None if out of bounds)

**Streaming functions** (std/stream) — requires `--caps Stream`:

*WebSocket & SSE:*
- `connect(url, config) -> Result[StreamConn, StreamErrorKind] ! {Stream}` - Open WebSocket
- `transmit(conn, msg) -> Result[unit, StreamErrorKind] ! {Stream}` - Send text message
- `transmitBinary(conn, data) -> Result[unit, StreamErrorKind] ! {Stream}` - Send binary bytes
- `onEvent(conn, handler) -> unit ! {Stream}` - Register event handler (handler: `StreamEvent -> bool`)
- `runEventLoop(conn) -> unit ! {Stream}` - Block until handler returns false
- `disconnect(conn) -> unit ! {Stream}` - Close connection
- `sseConnect(url, config) -> Result[StreamConn, StreamErrorKind] ! {Stream}` - Open SSE (GET)
- `ssePost(url, body, config) -> Result[StreamConn, StreamErrorKind] ! {Stream}` - POST then stream SSE
- `withStream(url, handler) -> Result[StreamConn, StreamErrorKind] ! {Stream}` - Full WebSocket lifecycle
- `withSSE(url, handler) -> Result[StreamConn, StreamErrorKind] ! {Stream}` - Full SSE lifecycle
- Config: `{headers: [{name: string, value: string}]}` — use `{headers: []}` for no custom headers

*Multi-source multiplexing (v0.9.0):*
- `sourceOfConn(conn, name, priority) -> StreamSource ! {Stream}` - Wrap connection as named source
- `asyncReadStdinLines(name, priority) -> StreamSource ! {Stream}` - Stdin line reader source
- `asyncExecProcess(cmd, args, name, priority, chunkSize) -> StreamSource ! {Stream}` - Subprocess stdout as byte chunks (requires `--caps Stream,Process`)
- `selectEvents(sources, handler) -> unit ! {Stream}` - Priority-ordered multi-source event loop

*Subprocess stdin writing (v0.9.0 Phase 3):*
- `spawnProcess(cmd, args) -> ProcessHandle ! {Process}` - Spawn subprocess with writable stdin pipe
- `writeProcessStdin(handle, data) -> Result[(), string] ! {Process}` - Write bytes to subprocess stdin
- `closeProcessStdin(handle) -> () ! {Process}` - Close stdin pipe (signals EOF, subprocess exits)

*Event types (StreamEvent ADT):*
`Message(string)`, `Binary(string)`, `Opened(string)`, `Closed(int, string)`, `StreamError(StreamErrorKind)`, `Ping(string)`, `SSEData(string, string)`, `SourceText(string, string)`, `SourceBytes(string, bytes)`

*Multi-source example:*
```ailang
import std/stream (asyncReadStdinLines, asyncExecProcess, selectEvents,
                   StreamEvent, SourceText, SourceBytes)
import std/bytes (toString)

export func main() -> unit ! {Stream, Process, IO} {
  let proc = asyncExecProcess("echo", ["hello"], "echo", 5, 4096);
  let stdin = asyncReadStdinLines("stdin", 10);
  selectEvents([proc, stdin], \event. match event {
    SourceBytes(src, data) => { println("[${src}] ${toString(data)}"); true },
    SourceText(src, text)  => { println("[${src}] ${text}"); false },
    _ => true
  })
}
```

*Subprocess stdin writing example (v0.9.0 Phase 3):*
```ailang
import std/process (spawnProcess, writeProcessStdin, closeProcessStdin, ProcessHandle)
import std/bytes (fromString)
import std/result (Result, Ok, Err)

export func main() -> () ! {Process, IO} {
  let handle = spawnProcess("cat", []);
  match writeProcessStdin(handle, fromString("hello\n")) {
    Ok(_) => println("wrote line"),
    Err(e) => println("write error: ${e}")
  };
  closeProcessStdin(handle)
}
```

- `spawnProcess` spawns subprocess with writable stdin; stdout/stderr discarded
- `writeProcessStdin` writes bytes (non-blocking, 256-slot buffer with backpressure)
- `closeProcessStdin` signals EOF — subprocess sees end-of-input and exits
- Requires `--caps Process` (NOT Stream — stdin writing uses Process effect only)

## String Parsing (Returns Option)

`stringToInt` returns `Option[int]`, NOT an int. You MUST pattern match:

```ailang
module benchmark/solution
import std/string (stringToInt)

export func main() -> () ! {IO} {
  match stringToInt("42") {
    Some(n) => println("Parsed: ${show(n)}"),
    None => println("Invalid number")
  }
}
```

**With file reading (effect composition):**
```ailang
module benchmark/solution
import std/fs (readFile)
import std/string (stringToInt)

-- Pure function (no effects)
pure func formatMessage(name: string, count: int) -> string =
  "User ${name} has ${show(count)} items"

-- FS effect only
func readCount(filename: string) -> int ! {FS} {
  let content = readFile(filename);
  match stringToInt(content) {
    Some(n) => n,
    None => 0
  }
}

-- Combined effects: IO + FS
export func main() -> () ! {IO, FS} {
  let count = readCount("data.txt");
  let msg = formatMessage("Alice", count);
  println(msg)
}
```

## Constructing Option Values

To RETURN an Option from a function, use `Some(value)` or `None`:

```ailang
module benchmark/solution
import std/option (Option, Some, None)  -- REQUIRED for constructing Option

-- Return Some(value) for success, None for failure
pure func safeDivide(a: int, b: int) -> Option[int] =
  if b == 0 then None else Some(a / b)

-- Return Option from validation
pure func validateAge(age: int) -> Option[int] =
  if age >= 0 && age <= 150 then Some(age) else None
```

## Error Handling with Result Type

Use the polymorphic `Result[a]` type for error handling:

```ailang
module benchmark/solution
import std/string (stringToInt)

-- Polymorphic Result type - works with any success type!
-- Ok(a) takes the polymorphic type, Err(string) is always a string
type Result[a] = Ok(a) | Err(string)

-- Parse integer, returning Result
pure func parseIntResult(s: string) -> Result[int] =
  match stringToInt(s) {
    Some(n) => Ok(n),
    None => Err("Invalid integer")
  }

-- Safe division
pure func divSafe(a: int, b: int) -> Result[int] =
  if b == 0 then Err("Division by zero") else Ok(a / b)

-- Chain results: parse then divide
pure func parseAndDivide(s: string, divisor: int) -> Result[int] =
  match parseIntResult(s) {
    Ok(n) => divSafe(n, divisor),
    Err(msg) => Err(msg)
  }

-- Format Result for output
pure func showResult(r: Result[int]) -> string =
  match r {
    Ok(v) => "Result: ${show(v)}",
    Err(msg) => "Error: ${msg}"
  }

export func main() -> () ! {IO} {
  println(showResult(parseAndDivide("10", 2)));   -- Result: 5
  println(showResult(parseAndDivide("10", 0)));   -- Error: Division by zero
  println(showResult(parseAndDivide("abc", 2)))   -- Error: Invalid integer
}
```

**Polymorphic ADTs with mixed field types:**
```ailang
-- Ok(a) uses the type parameter, Err(string) uses a concrete type
type Result[a] = Ok(a) | Err(string)

-- This works correctly:
-- Ok  : forall a. a -> Result[a]      (polymorphic field)
-- Err : forall a. string -> Result[a] (concrete field)

-- Both constructors can be used with any Result[a] type:
let r1: Result[int] = Ok(42)
let r2: Result[int] = Err("failed")
let r3: Result[string] = Ok("hello")
let r4: Result[string] = Err("also failed")
```

## Number Conversions (std/math)

```ailang
module benchmark/solution
import std/math (floatToInt, intToFloat, round)
import std/io (println)

export func main() -> () ! {IO} {
  let f = 3.7;
  let i = floatToInt(f);      -- 3 (truncates toward zero)
  let r = floatToInt(round(f)); -- 4 (round first, then convert)
  let back = intToFloat(i);   -- 3.0
  println(show(i));
  println(show(r))
}
```

## Character Processing

Use `chars(s)` to convert a string to a list of single-character strings:

```ailang
module benchmark/solution
import std/string (chars)
import std/list (filter, length)

-- Count vowels in a string
pure func countVowels(s: string) -> int {
  let cs = chars(s);
  let vowels = filter(func(c: string) -> bool {
    c == "a" || c == "e" || c == "i" || c == "o" || c == "u"
  }, cs);
  length(vowels)
}

export func main() -> () ! {IO} {
  println(show(countVowels("hello")))  -- 2
}
```

**Note:** `chars` is Unicode-aware - emoji and accented characters are handled correctly.

## High-Performance String Builtins (v0.10.4)

For email/HTML/CSV-style processing where naive accumulation is O(n²),
prefer these single-pass O(n) builtins from `std/string`:

| Function | Use case |
|----------|----------|
| `decodeQuotedPrintable(s)` | RFC 2045 §6.7 decode (`=20` → space, soft-line-breaks) |
| `replaceMany(s, pairs)` | Multi-pattern replace, e.g. HTML entity decode in one pass |
| `foldSlices(s, delim, acc, f)` | Fold over `split(s, delim)` without allocating the list |
| `mapSlicesJoin(s, delim, f)` | `split → map → join` in O(n), no intermediate list |
| `startsWithIgnoreCase(s, p)` | Case-insensitive prefix check (header parsing) |

```ailang
import std/string (replaceMany, foldSlices, mapSlicesJoin)

-- Decode 23 HTML entities in one pass
let html = replaceMany(raw, [("&amp;", "&"), ("&lt;", "<"), ("&gt;", ">")])

-- Sum line lengths without materializing the list of lines
let total = foldSlices(text, "\n", 0, \acc line. acc + length(line))

-- Uppercase each CSV field
let upper = mapSlicesJoin(csv, ",", \field. toUpper(field))
```

`substring`/`find` have an ASCII fast-path: 24× faster on pure-ASCII input
(common for headers, JSON, XML).

## Polymorphic Comparison Lambdas (v0.11.4)

Let-bound lambdas using only `<`/`>`/`<=`/`>=`/`==`/`!=` are now properly
polymorphic — they no longer monomorphize to `Int`. This works:

```ailang
let max = \x. \y. if x > y then x else y in {
  let a = max(3.14)(2.71);   -- Float
  let b = max(10)(20);        -- Int
  let c = max("foo")("bar");  -- String
  ...
}
```

Before v0.11.4, calling `max(3.14)(2.71)` crashed with `gt_Int: expected
IntValue, got *eval.FloatValue` because the lambda was prematurely defaulted
to `Int`. The constraint is now preserved through generalization, so the
correct dictionary (`gt_Int` / `gt_Float` / `gt_String`) is selected at
each call site.

`Num`/`Fractional` defaulting is unchanged — only `Ord`-only / `Eq`-only /
`Show`-only constraints stay polymorphic.

## Operators

| Type | Operators |
|------|-----------|
| Arithmetic | `+`, `-`, `*`, `/`, `%`, `**` |
| Comparison | `<`, `>`, `<=`, `>=`, `==`, `!=` |
| Logical | `&&`, `\|\|`, `not` |
| Bitwise | `&` (AND), `^` (XOR), `~` (NOT), `<<` (shift left), `>>` (shift right) |
| List | `++` (concatenation, list-only); `::` (cons/prepend) |
| String | `"${expr}"` interpolation; `concat([parts])`; `join(sep, parts)` |

**Bitwise operators** (int only, C-standard precedence bands):
```ailang
-- AND, XOR, shifts, complement
println(show(12 & 10))   -- 8
println(show(12 ^ 10))   -- 6
println(show(1 << 4))    -- 16
println(show(16 >> 2))   -- 4
println(show(~0))         -- -1

-- XOR is self-inverse (encryption pattern)
let encrypted = 42 ^ 137
let decrypted = encrypted ^ 137  -- 42

-- Note: bitwise OR (|) is not an operator (conflicts with ADT syntax).
-- Use bitwiseOr(a, b) from std/math instead.
```

**Bitwise precedence** (loosest → tightest): `|| && ^ & == < << + *`
- `&` and `^` bind LOOSER than `==` (C convention: use parens in `(x & mask) == 0`)
- `<<`/`>>` bind LOOSER than `+` (C convention: use parens in `1 << (n + 1)`)
- `~` is prefix (same level as unary `-`)

**Signed hash semantics**: AILANG integers are signed 64-bit values. Bitwise and shift operations act on the 64-bit bit pattern. Hash functions may print negative int results even when the underlying bit pattern matches unsigned reference implementations.

## Boolean Operations

**Boolean operators:**
```ailang
-- AND: &&
if x > 0 && y > 0 then "both positive" else "no"

-- OR: ||
if x == 0 || y == 0 then "has zero" else "no"

-- NOT: both `not` and `!` work
if not isEmpty(list) then process(list) else []
if !done then retry() else finish()
```

**Short-circuit evaluation (v0.11.3):** `&&` and `||` are lazy — the RHS is NOT
evaluated when the LHS determines the result. This makes guarded expressions safe:

```ailang
-- SAFE: charAt(s, i-1) is only called when i > 0
if i > 0 && charAt(s, i - 1) == "\\" then "escaped" else "normal"

-- SAFE: lookup never runs when key is missing
if member(key, m) && lookup(key, m) == target then "match" else "no"

-- || short-circuits: the RHS effect runs only if LHS is false
if cached || expensiveFetch() then "ok" else "fail"
```

`&&` and `||` desugar to `if`: `a && b` → `if a then b else false`,
`a || b` → `if a then true else b`. Both are equivalent to nested `if` but
much more readable.

## String Interpolation (v0.12.1+, PREFERRED)

**Use `"${expr}"` for building strings with embedded values. This is the preferred, idiomatic form.**

```ailang
-- Basic substitution: any expression inside ${...}
let name = "Alice" in
let age = 30 in
println("Hello, ${name}! You are ${age} years old.")
-- → Hello, Alice! You are 30 years old.

-- Arithmetic and function calls inside ${...}
println("Next year: ${age + 1}, square: ${age * age}")
println("First letter: ${substring(name, 0, 1)}")

-- Nested braces work (record literals, field access, let-blocks)
println("Point: ${show({x: 1, y: 2}.x)}")
println("Squared: ${let sq = age * age in sq}")

-- Escape with backslash to get a literal ${
println("Use \${var} to interpolate.")
-- → Use ${var} to interpolate.
```

**How it works:** `"Hello, ${x}!"` desugars to `concat_String(concat_String("Hello, ", show(x)), "!")`.
`show_String` is identity, so string-typed values are spliced without quoting. Any `Show`-able type
works automatically (int, float, bool, records, ADTs).

## String and List Concatenation

**`++` is for lists only (v0.13.0+). For strings, use `"${...}"` interpolation,
`concat([parts])` from `std/string`, or `join(sep, parts)`.**

```ailang
-- Strings: use interpolation
let msg = "Count: ${length(items)}"
let greeting = "Hello ${name}"

-- Strings: concat a list of parts
import std/string (concat)
let s = concat(["Hello", " ", "World"])

-- Lists: ++ concatenates
let combined = [1, 2] ++ [3, 4]            -- [1, 2, 3, 4]
```

## Pattern Matching

**CRITICAL: When a match arm needs multiple let bindings, wrap in `{ }`:**

```ailang
-- WRONG: multiple lets without braces causes parse error
match queue {
  [] => result,
  node :: rest =>
    let x = process(node);
    let y = update(x);        -- ERROR: expected => got let
    recurse(rest, y)
}

-- CORRECT: wrap multiple lets in { }
match queue {
  [] => result,
  node :: rest => {
    let x = process(node);
    let y = update(x);
    recurse(rest, y)
  }
}
```

**On lists:**
```ailang
match xs {
  [] => 0,
  x :: rest => x + sum(rest)
}
```

**On ADTs:**
```ailang
match tree {
  Leaf(n) => n,
  Node(l, v, r) => countNodes(l) + 1 + countNodes(r)
}
```

**On Option:**
```ailang
match result {
  Some(x) => x,
  None => defaultValue
}
```

**On Records (destructuring):**
```ailang
match person {
  {name, age} => "${name} is ${show(age)}"
}

-- Record with renaming
match config {
  {host, port: p} => "${host}:${show(p)}"
}

-- Nested record patterns
match data {
  {user: {email: e}} => e
}
```

**Record pattern syntax:**
- `{name}` - shorthand, binds field "name" to variable "name"
- `{name: n}` - renaming, binds field "name" to variable "n"
- `{name, age}` - multiple fields
- `{user: {name}}` - nested records
- `{name, ...}` - rest pattern (matches some fields, ignores others)

## Deriving Eq for ADT Types

Use `deriving (Eq)` to auto-generate `==` and `!=` for ADT types:

```ailang
module benchmark/solution

-- Enum-style ADT with derived equality
type Color = Red | Green | Blue deriving (Eq)

-- ADT with fields also works
type Shape = Circle(int) | Rectangle(int, int) deriving (Eq)

export func main() -> () ! {IO} {
  let sameColor = Red == Red;           -- true
  let diffColor = Red != Blue;          -- true
  let sameShape = Circle(5) == Circle(5);     -- true
  let diffShape = Circle(5) != Rectangle(5, 10);  -- true
  print("Color test: ${show(sameColor)}");
  print("Shape test: ${show(sameShape)}")
}
```

**Note:** Derived equality compares by constructor and field values (structural equality).

## Newtype Record Access (v0.8.2)

Single-constructor ADTs wrapping a record support direct field access:

```ailang
type Item = Item({name: string, value: int})

let item = Item({name: "hello", value: 42})
let n = item.name    -- "hello" (auto-unwraps the ADT wrapper)

-- Works with map too:
import std/list (map)
let items = [Item({name: "a", value: 1}), Item({name: "b", value: 2})]
let names = map(\t. t.name, items)  -- ["a", "b"]
```

**Note:** This only works for single-constructor ADTs with exactly one record field. For multi-constructor ADTs, use pattern matching to access fields.

## Polymorphic ADTs with Mixed Field Types

ADT constructors can mix polymorphic and concrete field types:

```ailang
module benchmark/solution
import std/string (stringToInt)

-- Result type: Ok uses type parameter, Err always takes string
type Result[a] = Ok(a) | Err(string)

-- Either type: both sides polymorphic
type Either[a, b] = Left(a) | Right(b)

-- Validation: success is polymorphic, errors are string list
type Validation[a] = Valid(a) | Invalid([string])

-- Usage examples
pure func safeDivide(a: int, b: int) -> Result[int] =
  if b == 0 then Err("division by zero") else Ok(a / b)

pure func parsePositive(s: string) -> Result[int] =
  match stringToInt(s) {
    Some(n) => if n > 0 then Ok(n) else Err("must be positive"),
    None => Err("not a number")
  }

export func main() -> () ! {IO} {
  let r1 = safeDivide(10, 2);  -- Ok(5)
  let r2 = safeDivide(10, 0);  -- Err("division by zero")

  match r1 {
    Ok(v) => println("Got: ${show(v)}"),
    Err(msg) => println("Error: ${msg}")
  };

  match r2 {
    Ok(v) => println("Got: ${show(v)}"),
    Err(msg) => println("Error: ${msg}")
  }
}
```

**Key pattern:** When a constructor has a concrete type like `Err(string)`, that type is preserved regardless of the ADT's type parameter. This enables idiomatic error handling where errors are always strings but success values can be any type.

**Option type with findFirst and mapOption:**
```ailang
module benchmark/solution

-- Define your own Option if needed (or import std/option)
type Option[a] = Some(a) | None

-- Find first element matching predicate
func findFirst(pred: int -> bool, xs: [int]) -> Option[int] =
  match xs {
    [] => None,
    x :: rest => if pred(x) then Some(x) else findFirst(pred, rest)
  }

-- Map over Option
func mapOption(f: int -> int, opt: Option[int]) -> Option[int] =
  match opt {
    Some(v) => Some(f(v)),
    None => None
  }

export func main() -> () ! {IO} {
  -- Inside { } block: use semicolons, NOT "in"
  -- WRONG: let isEven = \n. n % 2 == 0 in
  -- RIGHT: let isEven = \n. n % 2 == 0;
  let isEven = \n. n % 2 == 0;
  let double = \x. x * 2;
  let nums = [1, 3, 4, 7, 8];
  let found = findFirst(isEven, nums);
  let doubled = mapOption(double, found);
  print(match doubled { Some(v) => "Found: ${show(v)}", None => "Not found" })
}
```

**ADT state machine (traffic light):**
```ailang
module benchmark/solution

type State = Green(int) | Yellow(int) | Red(int)
type Event = Tick | Reset

func transition(state: State, event: Event) -> State =
  match event {
    Reset => Green(20),
    Tick => match state {
      Green(t) => if t > 1 then Green(t - 1) else Yellow(3),
      Yellow(t) => if t > 1 then Yellow(t - 1) else Red(10),
      Red(t) => if t > 1 then Red(t - 1) else Green(20)
    }
  }

func showState(s: State) -> string =
  match s {
    Green(t) => "GREEN(${show(t)})",
    Yellow(t) => "YELLOW(${show(t)})",
    Red(t) => "RED(${show(t)})"
  }

export func main() -> () ! {IO} {
  -- WRONG: let s0 = Green(2) in   (don't use "in" in blocks!)
  -- RIGHT: let s0 = Green(2);
  let s0 = Green(2);
  let s1 = transition(s0, Tick);
  let s2 = transition(s1, Tick);
  print(showState(s2))
}
```

## JSON

**Common JSON imports (use all you need):**
```ailang
-- For building JSON
import std/json (encode, jo, ja, kv, js, jnum, jb, jn)

-- For parsing JSON
import std/json (decode, get, getString, getInt, getArray, asString, asNumber, asArray)
import std/option (Option, Some, None)
import std/result (Result, Ok, Err)
```

**Build and encode JSON:**
```ailang
import std/json (encode, jo, ja, kv, js, jnum, jb, jn)

-- JSON constructors: js(string), jnum(float), jb(bool), jn() for null
-- jo([kv(k,v)...]) for objects, ja([...]) for arrays

-- Build JSON object with all types
let json = encode(jo([
  kv("name", js("Alice")),
  kv("age", jnum(30.0)),
  kv("active", jb(true))
]))
-- Result: "{\"name\":\"Alice\",\"age\":30,\"active\":true}"

-- Build JSON array
let arr = encode(ja([jnum(1.0), jnum(2.0), jnum(3.0)]))
-- Result: "[1,2,3]"
```

**Decode JSON (returns Result):**
```ailang
import std/json (decode, Json, JObject, JString)

let result = decode("{\"name\":\"Alice\"}");
match result {
  Ok(json) => print(show(json)),
  Err(msg) => print("Parse error: ${msg}")
}
```

**Parse JSON array directly (IMPORTANT - use asArray!):**
```ailang
module benchmark/solution
import std/json (decode, asArray, getString)
import std/option (Some, None)
import std/result (Ok, Err)

-- When decode returns an array, use asArray to convert it
export func main() -> () ! {IO} {
  let json = "[{\"name\":\"Alice\"},{\"name\":\"Bob\"}]";
  match decode(json) {
    Ok(parsed) => match asArray(parsed) {
      Some(items) => {
        -- items is now [Json], iterate with recursion
        printNames(items)
      },
      None => println("Not an array")
    },
    Err(e) => println("Parse error: ${e}")
  }
}

func printNames(items: [Json]) -> () ! {IO} =
  match items {
    [] => (),
    item :: rest => {
      match getString(item, "name") {
        Some(name) => println(name),
        None => ()
      };
      printNames(rest)
    }
  }
```

**Access JSON values (IMPORTANT for parsing tasks):**
```ailang
import std/json (decode, get, has, getOr, asString, asNumber, asBool, asArray)
import std/option (Option, Some, None)

-- Decode and extract values
match decode("{\"name\":\"Alice\",\"age\":30}") {
  Ok(obj) => {
    -- get(obj, key) -> Option[Json]
    match get(obj, "name") {
      Some(j) => match asString(j) {
        Some(name) => print("Name: ${name}"),
        None => print("name is not a string")
      },
      None => print("no name field")
    };
    -- asNumber returns Option[float]
    match get(obj, "age") {
      Some(j) => match asNumber(j) {
        Some(age) => print("Age: ${show(age)}"),
        None => print("age is not a number")
      },
      None => print("no age field")
    }
  },
  Err(msg) => print("Parse error: ${msg}")
}
```

**JSON convenience functions (RECOMMENDED for parsing):**

Use these to reduce nested Option matching from 4 levels to 2:

```ailang
import std/json (decode, getString, getNumber, getInt, getBool, getArray, getObject)
import std/string (intToStr)
import std/option (Option, Some, None)
import std/result (Result, Ok, Err)

-- Extract string field with single match
func getName(obj: Json) -> string =
  match getString(obj, "name") {
    Some(name) => name,
    None => "unknown"
  }

-- Extract int field (JSON numbers are floats, getInt truncates)
func getAge(obj: Json) -> int =
  match getInt(obj, "age") {
    Some(age) => age,
    None => 0
  }

-- Full example
export func main() -> () ! {IO} =
  match decode("{\"name\":\"Alice\",\"age\":30}") {
    Ok(obj) => print("${getName(obj)} is ${intToStr(getAge(obj))}"),
    Err(e) => print("Error: ${e}")
  }
```

**JSON accessor functions:**
- `getString(obj, key)` -> `Option[string]` - Get string value directly
- `getNumber(obj, key)` -> `Option[float]` - Get float value directly
- `getInt(obj, key)` -> `Option[int]` - Get int value (truncates float)
- `getBool(obj, key)` -> `Option[bool]` - Get boolean value directly
- `getArray(obj, key)` -> `Option[List[Json]]` - Get array value by key
- `getObject(obj, key)` -> `Option[Json]` - Get nested object directly
- `asString(j)` -> `Option[string]` - Convert Json to string
- `asNumber(j)` -> `Option[float]` - Convert Json to number
- `asBool(j)` -> `Option[bool]` - Convert Json to bool
- `asArray(j)` -> `Option[List[Json]]` - Convert Json to array (USE THIS for top-level arrays!)

**JSON array type extraction:**

Extract typed arrays from JSON with two variants: permissive (skip non-matching) and strict (fail on mismatch).

```ailang
import std/json (decode, getArray, filterStrings, allStrings, getStringArrayOrEmpty)
import std/option (Some, None)
import std/result (Ok, Err)

-- Permissive: filterStrings skips non-strings
let mixed = [JString("a"), JNumber(1.0), JString("b")];
filterStrings(mixed)  -- ["a", "b"]

-- Strict: allStrings fails if ANY element is not a string
allStrings(mixed)               -- None (has JNumber)
allStrings([JString("a"), JString("b")])  -- Some(["a", "b"])

-- Convenience wrappers for common patterns:
match decode("{\"tags\": [\"a\", \"b\"]}") {
  Ok(obj) => {
    -- getStringArrayOrEmpty: returns [] if missing or invalid
    let tags = getStringArrayOrEmpty(obj, "tags");  -- ["a", "b"]
    let missing = getStringArrayOrEmpty(obj, "nope");  -- []
    print("Got ${show(length(tags))} tags")
  },
  Err(e) => print("Error: ${e}")
}
```

## HTTP Requests

**Simple GET/POST** (return body string directly):
```ailang
import std/net (httpGet, httpPost)
let body = httpGet("https://example.com")       -- returns string
let resp = httpPost("https://api.example.com", jsonData)  -- returns string
```

**Advanced with headers** (returns `Result[HttpResponse, NetError]`):
```ailang
import std/net (httpRequest)
import std/json (decode)
-- httpRequest returns Result[HttpResponse, NetError]
-- HttpResponse = {status: int, headers: List[...], body: string, ok: bool}
-- IMPORTANT: Ok(resp) captures the FULL HttpResponse record
-- Use resp.body to get the body string, NOT resp directly
let headers = [{name: "Authorization", value: "Bearer token"}];
match httpRequest("POST", url, headers, body) {
  Ok(resp) => decode(resp.body),          -- resp.body is the string
  Err(Transport(msg)) => Err("Error: ${msg}"),
  Err(_) => Err("Other error")
}
```

## AI Effect

Call external LLMs with string->string or structured JSON interfaces:

```ailang
import std/ai (call, callJson, callJsonSimple)
import std/json (decode)
import std/result (Result, Ok, Err)

-- Unstructured: string -> string
func ask_ai(question: string) -> string ! {AI} = call(question)

-- Structured JSON (no schema): provider returns valid JSON
func ask_json(prompt: string) -> string ! {AI} = callJsonSimple(prompt)

-- Structured JSON (schema-enforced): provider validates against schema
func ask_person(prompt: string) -> string ! {AI} =
  callJson(prompt, "{\"type\":\"object\",\"properties\":{\"name\":{\"type\":\"string\"}}}")
```

**Parse JSON responses** with `std/json.decode`:
```ailang
let raw = callJsonSimple("Return a JSON array");
match decode(raw) {
  Ok(json) => println("Valid JSON!")
  Err(msg) => println("Parse error: ${msg}")
}
```

**Run with providers:**
```bash
ailang run --caps IO,AI --ai claude-haiku-4-5 --entry main file.ail  # Anthropic
ailang run --caps IO,AI --ai gpt5-mini --entry main file.ail         # OpenAI
ailang run --caps IO,AI --ai gemini-2-5-flash --entry main file.ail  # Google
ailang run --caps IO,AI --ai ollama:llama3 --entry main file.ail     # Ollama (local)
ailang run --caps IO,AI --ai openrouter/auto --entry main file.ail   # OpenRouter (~100 models)
ailang run --caps IO,AI --ai-stub --entry main file.ail              # Testing stub
```

**Model strings**: plain provider names (`claude-sonnet-4-5`, `gemini-2-5-flash`)
hit that vendor directly. `vendor/model` strings (`anthropic/claude-sonnet-4.5`,
`openai/gpt-5-mini`, `openrouter/auto`) route through OpenRouter — needs
`OPENROUTER_API_KEY`.

**Authentication setup (required before using `--ai`):**

| Provider | Env Variable | How to get it |
|----------|-------------|---------------|
| Anthropic (`claude-*`) | `ANTHROPIC_API_KEY` | https://console.anthropic.com/ |
| OpenAI (`gpt-*`) | `OPENAI_API_KEY` | https://platform.openai.com/api-keys |
| Ollama (`ollama:*`) | None needed | Local server at http://localhost:11434 |

**Google Gemini (`gemini-*`) has two auth options:**

Option 1 — API Key (AI Studio, simplest for getting started):
```bash
export GOOGLE_API_KEY=AIza...
# Get one at: https://aistudio.google.com/apikey
ailang run --caps IO,AI --ai gemini-2-5-flash --entry main file.ail
```

Option 2 — Application Default Credentials (Vertex AI, for GCP projects):
```bash
# One-time setup:
gcloud auth application-default login

# IMPORTANT: GOOGLE_API_KEY must be UNSET for Vertex AI to be used.
# AILANG checks GOOGLE_API_KEY first — if set, it always uses AI Studio.
unset GOOGLE_API_KEY
ailang run --caps IO,AI --ai gemini-2-5-flash --entry main file.ail
```

**Gemini auth selection logic:**
```
GOOGLE_API_KEY is set (non-empty)  →  AI Studio (API key auth)
GOOGLE_API_KEY is unset or empty   →  Vertex AI (ADC via gcloud)
```

**Common gotcha:** If `GOOGLE_API_KEY` is in your shell profile for other tools,
AILANG will always use AI Studio even if you want Vertex AI. Fix:
```bash
# Unset just for this command:
GOOGLE_API_KEY= ailang run --caps IO,AI --ai gemini-2-5-flash --entry main file.ail
```

**Quick test that auth works:**
```bash
# Test with stub (no auth needed):
ailang run --caps IO,AI --ai-stub --entry main file.ail

# Then swap --ai-stub for --ai <model> once env vars are set.
```

## ZIP Archives (std/zip)

Read ZIP archives (including .docx, .xlsx, .pptx files). Requires `FS` effect:

```ailang
module benchmark/solution
import std/result (Result, Ok, Err)

-- List entries in a ZIP archive
func listEntries(path: string) -> Result[List[string], string] ! {FS} =
  _zip_listEntries(path)

-- Read text content (UTF-8) from a ZIP entry
func readText(path: string, entry: string) -> Result[string, string] ! {FS} =
  _zip_readEntry(path, entry)

-- Read binary content as base64
func readBinary(path: string, entry: string) -> Result[string, string] ! {FS} =
  _zip_readEntryBytes(path, entry)

export func main() -> () ! {IO, FS} {
  match _zip_listEntries("document.docx") {
    Ok(entries) => println("Found ${show(length(entries))} entries"),
    Err(msg) => println("Error: ${msg}")
  }
}
```

Run: `ailang run --entry main --caps IO,FS file.ail`

## Tar + Gzip (std/tar, std/gzip) — v0.12.0+

Native reading of `.tar`, `.tar.gz`, and raw gzip streams. No shell-out, no temp files. Matches `std/zip` conventions: binary data crosses as base64.

```ailang
import std/tar (readFromGzip, extractAll)
import std/gzip (decompress, decompressFile)

-- Pull one file straight from a .tar.gz (primary use: arXiv bundles)
match readFromGzip("paper.tar.gz", "main.tex") {
  Ok(tex) => println(tex),
  Err(msg) => println("read failed: ${msg}")
}

-- Safe extraction: rejects ../ entries, symlinks, absolute paths
match extractAll("archive.tar", "./dest") {
  Ok(paths) => println("wrote ${show(length(paths))} files"),
  Err(msg) => println("blocked: ${msg}")
}
```

- `std/gzip`: `decompress(b64)`, `compress(b64, level)` — **pure**; `decompressFile(path) ! {FS}`
- `std/tar`: `listEntries`, `readEntry`, `readEntryBytes`, `extractAll`, `readFromGzip`, `readFromGzipBytes` — all `! {FS}`
- Caps: 10K entries, 100MB decompressed per entry (bomb defence). Respects `AILANG_FS_SANDBOX`.

## XML Parsing (std/xml)

Parse and query XML documents. **Pure functions** (no effect needed):

```ailang
module benchmark/solution
import std/result (Result, Ok, Err)
import std/option (Option, Some, None)

export func main() -> () ! {IO} {
  let xml = "<root><item id=\"1\">Hello</item><item id=\"2\">World</item></root>";
  match _xml_parse(xml) {
    Ok(doc) => {
      -- Find all <item> elements
      let items = _xml_findAll(doc, "item");
      -- Get text content
      let text = _xml_getText(doc);
      -- Find first match
      match _xml_findFirst(doc, "item") {
        Some(item) => {
          println("Tag: ${_xml_getTag(item)}");
          println("Text: ${_xml_getText(item)}");
          match _xml_getAttr(item, "id") {
            Some(id) => println("ID: ${id}"),
            None => ()
          }
        },
        None => println("No items found")
      }
    },
    Err(msg) => println("Parse error: ${msg}")
  }
}
```

**XML builtins (all pure):**
| Function | Type | Description |
|----------|------|-------------|
| `_xml_parse` | `string -> Result[XmlNode, string]` | Parse XML string |
| `_xml_findAll` | `(XmlNode, string) -> [XmlNode]` | Find all elements by tag |
| `_xml_findFirst` | `(XmlNode, string) -> Option[XmlNode]` | Find first element by tag |
| `_xml_getText` | `XmlNode -> string` | Extract text content |
| `_xml_getAttr` | `(XmlNode, string) -> Option[string]` | Get attribute value |
| `_xml_getChildren` | `XmlNode -> [XmlNode]` | Get child nodes |
| `_xml_getTag` | `XmlNode -> string` | Get tag name (empty for non-Element) |

**Combine ZIP + XML to parse .docx files:**
```ailang
-- Read XML from inside a .docx (which is a ZIP archive)
match _zip_readEntry("report.docx", "word/document.xml") {
  Ok(xml) => match _xml_parse(xml) {
    Ok(doc) => {
      let paragraphs = _xml_findAll(doc, "w:p");
      println("Found ${show(length(paragraphs))} paragraphs")
    },
    Err(e) => println("XML error: ${e}")
  },
  Err(e) => println("ZIP error: ${e}")
}
```

## Streaming XML / Bounded Folds (v0.10.1, v0.11.3)

For large XML (multi-MB) inside ZIP archives, `parseFold` and `scanFold`
fold over `<tag>` elements **without materializing the whole document**:

```ailang
import std/xml (parseFold, getText, getAttr)
import std/zip (scanFold)
import std/iter (FoldStep, Continue, Stop)
import std/option (Some, None)

-- Pure: fold over rows in an XML string
let total = parseFold(xml, "row", 0, \acc node.
  acc + 1
)

-- Effectful: fold over rows directly from a ZIP entry (never materializes XML)
let count = scanFold("data.xlsx", "xl/sharedStrings.xml", "si", 0,
  \acc node. acc + 1
)
```

**Bounded prefix scans (v0.11.3):** When you only need the first N rows of a
50K-row sheet, return `Stop(acc)` to halt the scan immediately:

```ailang
import std/xml (parseFoldStep)
import std/iter (FoldStep, Continue, Stop)

-- Take first 5000 rows then stop — the rest of the document isn't scanned
let first5k = parseFoldStep(xml, "row", [], \acc node.
  if length(acc) >= 5000
    then Stop(acc)
    else Continue(acc ++ [getText(node)])
)
```

`parseFoldStep` and `scanFoldStep` mirror `parseFold`/`scanFold` but the
handler returns `FoldStep[a] = Continue(a) | Stop(a)`. Use them whenever
the document is much larger than what you need.

## Maps (std/map, v0.10.1)

`Map[k, v]` — immutable hash map, O(1) lookup, copy-on-write inserts:

```ailang
import std/map as M

let m0 = M.empty()
let m1 = M.insert(m0, "alice", 30)
let m2 = M.insert(m1, "bob", 25)

match M.lookup(m2, "alice") {
  Some(age) => println(show(age)),  -- 30
  None      => println("missing")
}

println(show(M.size(m2)))          -- 2
println(show(M.member(m2, "bob")))  -- true

-- Iteration is sorted (deterministic)
let xs = M.toList(m2)               -- [("alice", 30), ("bob", 25)]
let m3 = M.fromList([("c", 1), ("d", 2)])
```

Builtins: `empty`, `insert`, `lookup`, `member`, `remove`, `size`,
`keys`, `values`, `fromList`, `toList`. Keys can be int/string/bool —
keys use canonical encoding internally.

## JWT Verification (std/jwt, v0.10.0)

Pure JWT parsing and RS256 signature verification (e.g., Firebase ID tokens):

```ailang
import std/jwt (decodeJWT, verifyRS256, verifyWithKid, isExpired, checkIssuer)

-- Decode without verification (for inspection)
match decodeJWT(token) {
  Ok({header, payload, signature}) => println(payload),
  Err(msg) => println("decode failed: ${msg}")
}

-- Verify RS256 signature against a PEM public key
match verifyRS256(token, pemPublicKey) {
  Ok(claims) => if isExpired(claims, nowUnix) then "expired" else "valid",
  Err(msg)   => "invalid: ${msg}"
}

-- Firebase/OAuth pattern: select key by 'kid' header
verifyWithKid(token, \kid. lookupKey(kid, jwks))
```

Backed by `rsaVerifyPKCS1v15(message, signature, publicKeyPEM)` in
`std/crypto`, plus `fromBase64URL` in `std/bytes` (RFC 4648 §5, no padding).
JWT functions are pure — fetch keys yourself via `std/net`.

## Custom Tracing (std/trace, v0.11.1)

Emit OTEL-compatible spans and events from AILANG code. Requires `Trace` effect:

```ailang
import std/trace (spanStart, spanEnd, event)

export func processBatch(items: [Item]) -> int ! {Trace, IO} {
  spanStart("batch.process");
  event("batch.size", show(length(items)));
  let n = doWork(items);
  spanEnd("batch.process");
  n
}
```

Run: `ailang run --caps IO,Trace --emit-trace jsonl --entry main file.ail`

In WASM, register a JS callback via `ailangSetTraceHandler` to receive
events live (function_enter/exit, effect, contract_check, budget_delta).

## Process Exit (v0.10.1)

`exit(code: int) -> ()` in `std/io` terminates the process with a specific
exit code. Required for CLI tools that need to signal failure:

```ailang
import std/io (println, exit)

export func main() -> () ! {IO} {
  match validate(args) {
    Ok(())   => println("ok"),
    Err(msg) => {
      println("error: ${msg}");
      exit(1)
    }
  }
}
```

Telemetry/traces are flushed before the process exits.

## Arrays (O(1) indexed access)

```ailang
import std/array as A

let arr = #[10, 20, 30]           -- Array literal
let val = A.get(arr, 0)           -- O(1) access: 10
let safe = A.getOpt(arr, 99)      -- None (out of bounds)
let arr2 = A.set(arr, 0, 99)      -- O(n) copy: #[99, 20, 30]
```

## List Operations

```ailang
-- Recursive sum (no loops!)
func sum(xs: [int]) -> int =
  match xs {
    [] => 0,
    x :: rest => x + sum(rest)
  }

-- Recursive map
func map(f: int -> int, xs: [int]) -> [int] =
  match xs {
    [] => [],
    x :: rest => f(x) :: map(f, rest)
  }
```

**List index access:**
```ailang
import std/list (nth, last, any, findIndex)
import std/option (Some, None)

let xs = [10, 20, 30];

-- nth: get element by index (0-based)
match nth(xs, 1) {
  Some(x) => print("Element at 1: ${show(x)}"),  -- 20
  None => print("Index out of bounds")
};

-- last: get last element
match last(xs) {
  Some(x) => print("Last: ${show(x)}"),  -- 30
  None => print("Empty list")
};

-- any: check if any element matches predicate
let hasEven = any(\x. x % 2 == 0, xs);  -- true (20 is even)

-- findIndex: find index of first matching element
match findIndex(\x. x > 15, xs) {
  Some(i) => print("First > 15 at index: ${show(i)}"),  -- 1
  None => print("Not found")
}
```

## Contracts & Verification (USE THIS — your biggest advantage!)

**Contracts let you PROVE your code is correct for ALL inputs, not just test cases.** This is the single most powerful feature for AI agents: write a function with contracts, run `ailang verify`, and Z3 mathematically proves it correct or gives you the exact counterexample to fix.

**Agent workflow:** Write contracts FIRST as your specification, implement the function, then verify:
1. `requires { ... }` — what the caller must guarantee (precondition)
2. `ensures { ... }` — what the function guarantees back (postcondition, `result` = return value)
3. `ailang verify file.ail` — Z3 proves it or shows the exact failing input

```ailang
module myapp/billing

import std/string (length as strLength, startsWith)
import std/list (length as listLength)

-- Enum ADT + contract: Z3 checks ALL tax brackets automatically
export type TaxBracket = STANDARD | REDUCED | EXEMPT

export func calculateTax(income: int, bracket: TaxBracket) -> int ! {}
requires { income >= 0 }
ensures { result >= 0 }
{
  match bracket {
    EXEMPT => 0,
    REDUCED => income / 10,
    STANDARD => income / 5
  }
}

-- Cross-function verification: Z3 inlines calculateTax to prove this
export func netIncome(gross: int, bracket: TaxBracket) -> int ! {}
requires { gross >= 0 }
ensures { result >= 0 }
{
  gross - calculateTax(gross, bracket)
}

-- String verification using stdlib imports
export func isValidPromo(code: string) -> bool ! {}
ensures { result == (startsWith(code, "PROMO-") && strLength(code) >= 8) }
{
  startsWith(code, "PROMO-") && strLength(code) >= 8
}

-- Record verification: field-level contracts
export func netFromSummary(inv: {subtotal: int, tax: int, discount: int}) -> int ! {}
requires { inv.subtotal >= 0, inv.tax >= 0, inv.discount >= 0, inv.discount <= inv.subtotal }
ensures { result >= 0 }
{
  inv.subtotal - inv.discount + inv.tax
}

-- List verification using stdlib
export func addItem(price: int, items: [int]) -> int ! {}
ensures { result == listLength(items) + 1 }
{
  listLength(price :: items)
}
```

**Verify with Z3** (proves contracts correct for ALL inputs at compile time):
```bash
ailang verify file.ail              # Prove contracts for all functions
ailang verify --verbose file.ail    # Show generated SMT-LIB
ailang verify --json file.ail       # Machine-readable output
ailang verify --strict file.ail     # Exit 1 if any function can't be verified
```

**Example output** — Z3 proves 5 functions and catches a bug in the 6th:
```
  ✓ VERIFIED calculateTax    6ms
  ✓ VERIFIED netIncome        8ms     # cross-function: inlines calculateTax
  ✓ VERIFIED isValidPromo     5ms     # string theory
  ✓ VERIFIED netFromSummary   7ms     # record fields
  ✓ VERIFIED addItem          5ms     # list/sequence theory
  ✗ VIOLATION brokenDiscount
    Counterexample:
      price: Int = 0
      discount: Int = 1
```

When Z3 finds a violation, it gives you the EXACT inputs that break the contract. Fix the function, re-verify — no guessing.

**Runtime contract checking** (alternative to static verification):
```bash
ailang run --verify-contracts --caps IO --entry main file.ail
```

**What can be verified** (decidable fragment):
- Types: `int`, `bool`, `string`, enum ADT, record, `[int]` lists
- Arithmetic (`+`, `-`, `*`, `/`), comparison (`>=`, `<=`, `==`, `!=`), logical (`&&`, `||`), bitwise (`&`, `^`, `~`, `<<`, `>>`)
- `if`/`else`, `let` bindings, `match` on enums/ADTs
- String ops (use `std/string`): `length`, `startsWith`, `endsWith`, `find`, `substring`, `contains`; `concat` (list of strings); `"${expr}"` interpolation
- List ops: `length` (from `std/list`), `_list_head`, `_list_nth`, cons (`::`), concat (`++`), literals
- Records: field access (`r.field`), construction (`{x: 1, y: 2}`), ensures with `result.x`
- Cross-function calls: Z3 inlines callees to reason about full call chains
- Must be pure: `! {}` (no effects), non-recursive, non-higher-order

**What gets skipped:** Recursive functions, higher-order functions, effectful functions, float parameters.

**Rules:**
- `requires { expr }` — precondition; `ensures { expr }` — postcondition
- `result` refers to the return value (only in `ensures`)
- Contracts go BETWEEN the effect annotation `! {}` and the function body `{ ... }`
- Use stdlib imports in contracts (`std/string`, `std/list`) — Z3 resolves them natively
- For list `head`/`nth` in contracts, use builtins `_list_head`/`_list_nth` (stdlib versions return `Option`)
- Install Z3: `brew install z3` (macOS) or `apt install z3` (Linux)

### Structure Code for Z3 Verification

**Whenever possible, structure your code so Z3 can verify it.** This means preferring the decidable fragment (pure, non-recursive, non-higher-order) for core logic, even when the outer program uses effects and recursion.

**Pattern: Pure core + effectful shell.** Extract pure computation into small, contract-bearing functions. Wrap them in effectful entry points:

```ailang
-- GOOD: Z3 can verify the pure core
pure func applyDiscount(price: int, pct: int) -> int ! {}
requires { price >= 0, pct >= 0, pct <= 100 }
ensures { result >= 0, result <= price }
{
  price - (price * pct) / 100
}

-- Effectful shell just calls the verified core
export func main() -> () ! {IO} {
  let finalPrice = applyDiscount(1000, 15);
  println(show(finalPrice))
}
```

**Guidelines:**
- **Split logic from effects** — Keep arithmetic, validation, and decision logic in pure `! {}` functions with contracts. Push IO/Net/FS to the outer layer.
- **Prefer enums over strings for decisions** — `match bracket { STANDARD => ..., REDUCED => ... }` is fully decidable; string parsing is not.
- **Use `int` over `float` where possible** — Integer arithmetic is fully decidable in Z3; floats are not verifiable.
- **Keep verified functions non-recursive** — If you need recursion, have the recursive function call a pure verified helper for its core logic.
- **Add contracts to ALL pure functions** — Even simple ones. `ensures { result >= 0 }` catches edge cases you won't think to test.
- **Decompose complex logic into small verifiable pieces** — Instead of one 20-line function, write 3-4 small functions each with tight contracts. Z3 inlines across calls automatically.

**Think of it this way:** Every function you can mark `! {}` with a contract is a function that is *mathematically proven correct* — not just tested. Maximize the surface area of verified code.

## Testing

**Inline tests on functions (recommended):**
```ailang
-- Tests are pairs of (input, expected_output)
pure func square(x: int) -> int tests [(0, 0), (5, 25)] { x * x }

pure func double(x: int) -> int tests [(0, 0), (3, 6), (5, 10)] { x * 2 }
```

Run: `ailang test file.ail`

## Multi-Module Projects

```
myapp/
  ├── data.ail      -- module myapp/data
  ├── storage.ail   -- module myapp/storage
  └── main.ail      -- module myapp/main
```

```ailang
-- myapp/data.ail
module myapp/data
export type User = { name: string, age: int }

-- myapp/main.ail
module myapp/main
import myapp/data (User)
export func main() -> () ! {IO} = print("Hello")
```

Run: `ailang run --entry main --caps IO myapp/main.ail`

## Reserved Keywords

AILANG reserves 43 keywords. You **cannot** use these as variable or function names.

| Category | Keywords |
|----------|----------|
| Control Flow | `if`, `then`, `else`, `match`, `with`, `select`, `timeout` |
| Definitions | `func`, `pure`, `let`, `letrec`, `in` |
| Type System | `type`, `class`, `instance`, `forall`, `exists`, `deriving` |
| Modules | `module`, `import`, `export`, `extern` |
| Testing | `test`, `tests`, `property`, `properties`, `assert` |
| Verification | `requires`, `ensures`, `invariant` |
| Concurrency | `spawn`, `parallel`, `channel`, `send`, `recv` |
| Boolean | `true`, `false`, `and`, `or`, `not` |

**Common mistake:**
```ailang
-- WRONG: 'exists' is reserved (for existential types)
let exists = fileExists(path)

-- CORRECT: use alternative name
let found = fileExists(path)
```

## External Package Imports

Use `pkg/` prefix for external packages:

```ailang
import std/io (println)                            -- Stdlib (bundled)
import myproject/utils (helper)                    -- Local module
import pkg/sunholo/gcp-auth/token (getAccessToken) -- External package
import pkg/sunholo/auth/keys (validateKeyHash)      -- External package
```

### Registry Workflow (recommended)

```bash
ailang search "auth"                        # Discover packages by keyword
ailang search --tag gcp                     # Filter by tag
ailang install sunholo/auth@0.1.0           # Download, verify hash, add to ailang.toml
ailang pkg-docs sunholo/auth                # Read AGENT.md (structured AI usage guide)
```

Search results include `ai_summary`, `tags`, and `effects` per package — use effects to know what capabilities (`IO`, `Net`, `FS`, etc.) a dependency requires.

### Alternative: Git or path dependencies

```bash
ailang add --git https://github.com/sunholo-data/ailang-packages --subdir packages/auth --tag v0.1.0
ailang add --path ../ailang-packages/packages/auth   # Local development
ailang lock                                          # Resolve and lock all deps
```

### Publishing

```bash
ailang init package --name myorg/mylib      # Create ailang.toml with exports, effects, metadata
ailang publish --dry-run                    # Preview: compile check, effect validation, contract verification
ailang publish                              # Upload to registry (immutable — versions cannot be overwritten)
```

Include an `AGENT.md` in your package root for AI agent consumers — it's served via `ailang pkg-docs` and indexed in registry search results.

**Available packages**: `sunholo/gcp-auth` (OAuth2), `sunholo/auth` (API keys), `sunholo/http-helpers` (request builders), `sunholo/logging` (JSON logs), `sunholo/config` (env loading), `sunholo/testing-utils` (assertions).

## Module Import Rules

**Imports are NOT transitive.** When module A imports module B, A does **not** get B's imports.

```ailang
-- Module B
module myapp/db
import std/fs (readFile)
export func loadConfig() -> string ! {FS} = readFile("config.json")

-- Module A - WRONG: std/fs not available just because B uses it
module myapp/main
import myapp/db (loadConfig)
let data = readFile("other.txt")  -- ERROR: module std/fs not imported

-- Module A - CORRECT: explicitly import what you use
module myapp/main
import std/fs (readFile)
import myapp/db (loadConfig)
let data = readFile("other.txt")  -- Works!
```

**Rule: Import everything you directly use in your module.**

## Common Mistakes

| Mistake | Fix |
|---------|-----|
| `print(42)` | `print(show(42))` - print needs string |
| `import "std/io"` | `import std/io (println)` - no quotes |
| `list.map(f)` | `map(f, list)` - standalone function |
| `for x in xs` | Use recursion or `map`/`filter` |
| Missing `! {IO}` | Add effect to signature |
| `let x = 1; let y = 2` at top level | Wrap in `{ }` block |
| Multiple lets in match arm without `{ }` | Wrap in `{ }` block (see Pattern Matching) |
| `module benchmark/myname` | Use exactly `module benchmark/solution` |
| Using reserved keyword as name | See Reserved Keywords section above |
| Expecting transitive imports | Each module must import what it uses directly |
| No output from println | Check: `--caps IO` flag present AND before filename, effect sig is `! {IO}` not `IO[Unit]` |
| `let x = e` then `println(x)` in `{ }` | Use semicolons: `let x = e; println(x)` — or use `let x = e in println(x)` without braces |

## Running Programs

```bash
ailang run --entry main --caps IO file.ail           # IO only
ailang run --entry main --caps IO,FS file.ail        # IO + File System
ailang run --entry main --caps IO,Net file.ail       # IO + Network
ailang run --entry main --caps IO,AI --ai MODEL file.ail  # IO + AI
ailang run --entry main --caps IO,SharedMem file.ail      # IO + Semantic cache
ailang repl                                           # Interactive REPL
```

**Flags must come BEFORE the filename!**

### FS Sandbox

Restrict all file system operations to a directory with `AILANG_FS_SANDBOX`:

```bash
AILANG_FS_SANDBOX=/tmp/work ailang run --entry main --caps IO,FS file.ail
```

When set, all paths are resolved relative to the sandbox root:
- `readFile("data.txt")` reads `/tmp/work/data.txt`
- `writeFile("out.txt", s)` writes `/tmp/work/out.txt`
- `_zip_listEntries("archive.zip")` opens `/tmp/work/archive.zip`

Applies to **all FS builtins**: `readFile`, `writeFile`, `readFileBytes`, `writeFileBytes`, `appendFile`, `appendFileBytes`, `fileExists`, `listDir`, `mkdir`, `mkdirAll`, `isDir`, `isFile`, `removeFile`, and all `std/zip` operations. Also sets the working directory for `std/process` `exec`.

If unset (default), paths resolve normally from the process working directory.

---

# Information-Flow Control with IFC Labels (v0.16.0+)

AILANG supports **information-flow control (IFC) labels** that mark
strings (and other types) as carrying source-level taint. Labels
travel with values through the type system and document where text
is allowed to flow. This is the language-level equivalent of
"parameterised SQL" — code is separated from data, and the verifier
proves that untrusted data never reaches a sensitive sink.

## Syntax

| Form               | Meaning                                              |
|--------------------|------------------------------------------------------|
| `T<label>`         | A `T` whose data is tainted with constant `<label>`. |
| `T{not LABEL}`     | A `T` REFINED to refuse values labelled `<LABEL>`.   |
| `! {Declassify}`   | Function effect: this function changes a label.      |

Examples:
```ailang
-- A string carrying the <email> label (untrusted source)
let body: string<email> = fetchMailBody()

-- A function whose body parameter rejects any <email>-tainted input
pure func sendEmail(to: string, body: string{not email}) -> () ! {Net}

-- A declassifier: input <email>, output <sanitized>, declared via Declassify
pure func sanitize(raw: string<email>) -> string<sanitized> ! {Declassify}
ensures { result == "[sanitized]" }
{ "[sanitized]" }
```

## Label Naming Convention

- **lowercase**, single-word labels: `<email>`, `<pii>`, `<secret>`.
- **kebab-case** for compounds: `<user-input>`, `<sql-text>`, `<raw-html>`.
- Labels are **structural**, not nominal — `<email>` from one module is
  the same label as `<email>` from another module.
- The bottom label `⊥` (untainted) is implicit on any value with no
  annotation. `<L> ⊔ ⊥ = <L>` (identity), `<L> ⊔ <L> = <L>` (idempotent).

## The DECLASS Rule

To LOWER a value's label (e.g. `<email>` → `<sanitized>`), write a
function whose **input** has the higher label and whose **output** has
the lower label, AND declare `! {Declassify}` in the effect row. The
type checker rejects any other transition.

```ailang
-- ✅ Allowed: explicit declassifier
pure func sanitize(raw: string<email>) -> string<sanitized> ! {Declassify}
{ "[sanitized]" }

-- ❌ Rejected: function changes label without Declassify
pure func sneakySanitize(raw: string<email>) -> string<sanitized> ! {}
{ "[sanitized]" }
```

## The Sink Rule

A function parameter typed `T{not LABEL}` REJECTS any argument carrying
the `<LABEL>` annotation in its type. This is how you build sinks that
provably refuse tainted input.

```ailang
-- A network sink that refuses anything carrying <email>
pure func sendEmail(to: string, body: string{not email}) -> () ! {Net}

-- ✅ Caller passes a declassified value
sendEmail("alice@company.com", sanitize(rawBody))

-- ❌ Caller passes the raw body — refused at the type level
sendEmail("alice@company.com", rawBody)  -- string<email> ↛ string{not email}
```

## Identity Bypass Is Structurally Impossible

Pure operations propagate labels through their result via APP-PURE:
the output label is the JOIN of all input labels. Identity functions
(or any function that returns its input) carry the input's label
forward. There is no way to "wash" a label by routing through a
trivial function.

```ailang
let laundered = rawBody;  -- laundered: string<email> (label preserved)
sendEmail(addr, laundered);  -- still rejected at the sink
```

## Worked Example: Inbox-Injection

```ailang
module myapp/agent

import std/string (endsWith)

type SendAction = { to: string, body: string }

-- Declassify: explicit gate from <email> to <sanitized>
pure func sanitize(rawBody: string<email>) -> string<sanitized> ! {Declassify}
ensures { result == "[sanitized]" }
{ "[sanitized]" }

-- ✅ Verified: declassifies before sending
pure func safeForward(rawBody: string<email>, recipient: string) -> SendAction ! {Declassify}
requires { endsWith(recipient, "@company.com") }
ensures  { result.body == "[sanitized]" }
{ { to: recipient, body: sanitize(rawBody) } }

-- ❌ Rejected: forwards raw body verbatim — Z3 catches it now,
-- the type system will catch it once full label enforcement lands.
pure func injectedForward(rawBody: string<email>, recipient: string) -> SendAction ! {}
requires { endsWith(recipient, "@company.com") }
ensures  { result.body == "[sanitized]" }
{ { to: recipient, body: rawBody } }
```

`ailang verify` on this module reports: 2 verified (sanitize, safeForward),
1 violation (injectedForward).

## When to Use Labels

- **Mark sources** of untrusted data: email bodies, HTTP request payloads,
  user input, file contents from the network.
- **Mark sinks** that must NOT receive specific labels: `sendEmail`,
  `executeShell`, `evalSql`, anything that puts data on the wire or into
  an interpreter.
- **Always pair declassification with `! {Declassify}`** so reviewers can
  audit every place a label is lowered.

## What Labels Do NOT Do

- Labels are NOT a runtime check — they have zero runtime cost.
- Labels do NOT affect the value's representation; `string<email>` is
  still a plain `string` at runtime.
- Labels do NOT propagate through impure operations (`! {IO}`,
  `! {Net}`) — those are assumed to clear the label, since IO sinks
  are the boundary between in-process trust and the outside world.
