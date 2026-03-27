# Common AILANG Patterns

## Recursion (NO LOOPS!)

### Countdown Example
```ailang
export func countdown(n: int) -> () ! {IO} {
  if n <= 0
  then ()
  else {
    println(show(n));
    countdown(n - 1)
  }
}
```

### Range Iteration
```ailang
export func printRange(start: int, end: int) -> () ! {IO} {
  if start > end
  then ()
  else {
    println(show(start));
    printRange(start + 1, end)
  }
}
```

## Pattern Matching

### Option Type
```ailang
type Option[a] = Some(a) | None

export func getOr[a](opt: Option[a], default: a) -> a {
  match opt {
    Some(x) => x,
    None => default
  }
}
```

### List Processing
```ailang
export func sum(xs: [int]) -> int {
  match xs {
    [] => 0,
    hd :: tl => hd + sum(tl)
  }
}

export func map[a,b](f: func(a) -> b, xs: [a]) -> [b] {
  match xs {
    [] => [],
    hd :: tl => f(hd) :: map(f, tl)
  }
}

export func filter[a](pred: func(a) -> bool, xs: [a]) -> [a] {
  match xs {
    [] => [],
    hd :: tl =>
      if pred(hd)
      then hd :: filter(pred, tl)
      else filter(pred, tl)
  }
}
```

## Effects and IO

### Basic IO
```ailang
module benchmark/solution

import std/io (println, readLine)

export func greet() -> () ! {IO} {
  println("What's your name?");
  let name = readLine();
  println("Hello, " ++ name ++ "!")
}

export func main() -> () ! {IO} {
  greet()
}
```

### File Operations
```ailang
module benchmark/solution

import std/io (println)
import std/fs (readFile, writeFile)

export func processFile(input: string, output: string) -> () ! {IO, FS} {
  let content = readFile(input);
  let processed = content ++ "\n-- Processed";
  writeFile(output, processed);
  println("File processed")
}

export func main() -> () ! {IO, FS} {
  processFile("input.txt", "output.txt")
}
```

## Record Updates

### Basic Update
```ailang
type Person = {name: string, age: int}

export func birthday(p: Person) -> Person {
  {p | age: p.age + 1}
}
```

### Multiple Field Update
```ailang
type Config = {debug: bool, verbose: bool, timeout: int}

export func makeProduction(cfg: Config) -> Config {
  {cfg | debug: false, verbose: false, timeout: 30}
}
```

## Common Mistakes and Fixes

### Mistake: Forgetting Module Declaration
```ailang
# WRONG
import std/io (println)
func main() { ... }

# CORRECT
module benchmark/solution
import std/io (println)
export func main() -> () ! {IO} { ... }
```

### Mistake: Using Semicolons Incorrectly
```ailang
# WRONG - no semicolon before closing brace
{ println("Hi"); println("Bye"); }

# CORRECT - semicolon between statements
{ println("Hi"); println("Bye") }
```

### Mistake: Pattern Matching Syntax
```ailang
# WRONG - using : or ->
match x {
  Some(v): v,
  None -> 0
}

# CORRECT - use =>
match x {
  Some(v) => v,
  None => 0
}
```

### Mistake: Trying to Use Loops
```ailang
# WRONG - no loops!
for i in 1..10 {
  println(show(i))
}

# CORRECT - use recursion
export func printRange(start: int, end: int) -> () ! {IO} {
  if start > end
  then ()
  else {
    println(show(start));
    printRange(start + 1, end)
  }
}
```

## Best Practices

### 1. Start with Module Declaration
Always start your file with `module path/name`

### 2. Import What You Use
```ailang
import std/io (println)        # Just what you need
import std/fs (readFile)       # Specific functions
```

### 3. Use Type Annotations for Public Functions
```ailang
export func add(x: int, y: int) -> int {
  x + y
}
```

### 4. Pattern Match Exhaustively
Cover all cases or use a wildcard:
```ailang
match opt {
  Some(x) => x,
  None => default
}
```

### 5. Handle Effects Properly
Declare all effects in function signatures:
```ailang
func readAndPrint() -> () ! {IO, FS} {
  let content = readFile("data.txt");
  println(content)
}
```

### 6. Design for Verification
Extract pure logic from effectful functions and add contracts:
```ailang
-- Pure core: Z3-verifiable
export func applyDiscount(price: int, rate: int) -> int ! {}
requires { price >= 0, rate >= 0, rate <= 100 }
ensures { result >= 0 }
{
  price - price * rate / 100
}

-- Effectful shell: just IO
export func main() -> () ! {IO} {
  let total = applyDiscount(1000, 15);
  println("Discounted: " ++ show(total))
}
```
Run `ailang verify file.ail` to prove contracts. See `resources/z3_verification_patterns.md` for full patterns.
