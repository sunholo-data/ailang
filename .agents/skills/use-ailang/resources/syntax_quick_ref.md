# AILANG Syntax Quick Reference

## Basic Syntax Rules
- Use `func` NOT `fn`, `function`, or `def`
- Semicolons REQUIRED between statements in blocks
- Pattern matching uses `=>` NOT `:` or `->`
- NO `for`, `while`, `var`, `const`, or any imperative constructs
- NO namespace syntax (`::`), use names directly

## Functions

### Named Function
```ailang
export func add(x: int, y: int) -> int {
  x + y
}
```

### Anonymous Function
```ailang
let double = func(x: int) -> int { x * 2 }
```

### With Effects
```ailang
export func greet(name: string) -> () ! {IO} {
  println("Hello, " ++ name)
}
```

## Pattern Matching

```ailang
type Option[a] = Some(a) | None

export func getOr[a](opt: Option[a], default: a) -> a {
  match opt {
    Some(x) => x,
    None => default
  }
}
```

## Lists (Recursive Processing)

```ailang
export func sum(xs: [int]) -> int {
  match xs {
    [] => 0,
    hd :: tl => hd + sum(tl)
  }
}
```

## Records

```ailang
type Person = {name: string, age: int}

-- Record update
export func birthday(p: Person) -> Person {
  {p | age: p.age + 1}
}
```

## Block Expressions

```ailang
export func compute() -> int {
  let x = 10;  -- Semicolons between statements!
  let y = 20;
  x + y        -- Last expression is the return value
}
```

## What Works

### ✅ Core Language
- **Module declarations**: `module path/to/module`
- **Function declarations**: `export func name(params) -> Type { body }`
- **Anonymous functions**: `func(x: int) -> int { x * 2 }`
- **Recursive lambdas**: `letrec fib = \n. if n < 2 then n else fib(n-1) + fib(n-2) in ...`
- **Pattern matching**: Constructors, tuples, lists, wildcards, guards
- **ADTs**: `type Option[a] = Some(a) | None`
- **Records**: `{name: "Alice", age: 30}`, field access, updates `{rec | age: 31}`
- **Block expressions**: `{ stmt1; stmt2; result }`

### ✅ Type System
- **Hindley-Milner inference**: Types are inferred automatically
- **Polymorphism**: Generic types with `[a]` syntax
- **Effect tracking**: `! {IO, FS, Clock, Net}` for side effects
- **Numeric types**: `int`, `float` with conversions `intToFloat`, `floatToInt`

### ✅ Contracts & Verification
- **Preconditions**: `requires { x >= 0, y > 0 }`
- **Postconditions**: `ensures { result >= 0 }` (`result` = return value)
- **Bounded quantifiers**: `ensures { forall i: 0..n => pred }`
- **Bounded recursion**: `@verify(depth: 5)` annotation
- **Pure effect**: `! {}` marks function as Z3-eligible
- **CLI**: `ailang verify file.ail`

### ✅ Imports
- **Standard library**: `import std/io (println)` — bundled modules
- **External packages**: `import pkg/sunholo/firestore/client (getDoc)` — from dependencies
- **Intra-package siblings**: `import ./plan (Plan)` — modules in same package
- **Selective imports**: `import std/list (map, filter, foldl)` — import specific symbols
- **Note**: `Ok`, `Err` require `import std/result (Ok, Err)` (not in prelude)

### ✅ Standard Library
- **Auto-imported std/prelude**: Comparisons (`<`, `>`, `==`, `!=`) work without imports
- **std/io**: `println`, `readLine`, `readInt`
- **std/fs**: `readFile`, `writeFile`, `listDir`
- **std/clock**: `now`, `sleep`
- **std/net**: `httpGet`, `httpPost`, `httpRequest` (GET/POST/PUT/PATCH/DELETE/HEAD)
- **std/json**: `json.decode`, `json.encode`

## Limitations (Things That DON'T Work)

### ❌ NO Imperative Constructs
```
# WRONG - These don't exist in AILANG
for i in range(10) { ... }
while x < 10 { ... }
var x = 5; x = x + 1;
break; continue;
```

### ❌ NO Error Propagation Operator (yet)
```
# WRONG - '?' doesn't exist yet
let value = readFile("file.txt")?;
```

### ❌ NO Classes or Objects
```
# WRONG - AILANG is pure functional
class Person { ... }
new Person()
```

### ❌ NO Mutable State
```
# WRONG - Everything is immutable
let mut x = 5;
x = x + 1;
```
