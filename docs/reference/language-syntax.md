# AILANG Language Syntax Reference

## Basic Constructs

```ailang
-- Comments use double dash
let x = 5                          -- Immutable binding
let f = (x: int) -> int => x * 2  -- Lambda function
if x > 0 then "pos" else "neg"    -- Conditional expression
[1, 2, 3]                          -- List literal
{ name: "Alice", age: 30 }        -- Record literal
(1, "hello", true)                 -- Tuple
```

## Functions

### Pure Functions (No Effects)
```ailang
pure func add(x: int, y: int) -> int {
  x + y
}
```

### Effectful Functions
```ailang
func readAndPrint() -> () ! {IO, FS} {
  let content = readFile("data.txt")?  -- ? propagates errors
  print(content)
}
```

### Functions with Inline Tests
```ailang
pure func factorial(n: int) -> int
  tests [
    (0, 1),
    (5, 120)
  ]
{
  if n <= 1 then 1 else n * factorial(n - 1)
}
```

## Lambda Expressions

```ailang
-- Basic lambda syntax
let add = \x y. x + y in
let add5 = add(5) in  -- Partial application
print("Result: " ++ show(add5(3)))  -- Result: 8

-- Higher-order functions
let compose = \f g x. f(g(x)) in
let double = \x. x * 2 in
let inc = \x. x + 1 in
let doubleThenInc = compose(inc)(double) in
print("Composed: " ++ show(doubleThenInc(5)))  -- Composed: 11
```

## Pattern Matching

```ailang
match value {
  Some(x) if x > 0 => x * 2,
  Some(x) => x,
  None => 0
}

match list {
  [] => "empty",
  [x] => "single",
  [head, ...tail] => "multiple"
}
```

## Quasiquotes

```ailang
-- SQL with type checking
let query = sql"""
  SELECT * FROM users 
  WHERE age > ${minAge: int}
"""

-- HTML with sanitization
let page = html"""
  <div>${content: SafeHtml}</div>
"""

-- Other quasiquotes: regex/, json{}, shell""", url"
```

## Effects and Capabilities

```ailang
import std/io (FS, Net)

func processData() -> Result[Data] ! {FS, Net} {
  with FS, Net {
    let data = readFile(FS, "input.txt")?
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