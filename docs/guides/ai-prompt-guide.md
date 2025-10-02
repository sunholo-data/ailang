# AI Prompt Guide: Teaching AILANG to Language Models

**Purpose**: This document contains the optimal prompt for teaching AI models how to write AILANG code.

**KPI**: One of AILANG's key success metrics is **"teachability to AI"** - how easily can an LLM learn to write correct AILANG code from a single prompt?

---

## The AILANG Prompt (v0.2.0-rc1)

Use this prompt when asking AI models to generate AILANG code:

```markdown
You are writing code in AILANG, a pure functional programming language with Hindley-Milner type inference and algebraic effects.

## Current Version: v0.2.0-rc1 (Module Execution + Effects)

**✅ WHAT WORKS (v0.2.0-rc1):**
- ✅ **Module declarations** - `module path/to/module`
- ✅ **Function declarations** - `export func name(params) -> Type { body }`
- ✅ **Import statements** - `import std/io (println)`
- ✅ **Export declarations** - `export func`, `export type`
- ✅ **Pattern matching** - Constructors, tuples, lists, wildcards
- ✅ **Effect system** - `! {IO, FS}` effect annotations with capability security
- ✅ **ADTs** - Algebraic data types: `type Option[a] = Some(a) | None`

**⚠️ CURRENT LIMITATIONS:**
- ⚠️ NO `for` loops or `while` - use recursion or list operations
- ⚠️ NO `var` or mutable state - everything is immutable
- ⚠️ NO pattern guards (`if` in match arms - parsed but not evaluated)
- ⚠️ NO error propagation operator `?`
- ⚠️ Let expressions limited to 3 nesting levels

## How to Structure AILANG Code

### ✅ Correct (Module with functions and effects):
```ailang
module examples/hello

import std/io (println)

export func main() -> () ! {IO} {
  println("Hello, World!")
}

export func greet(name: string) -> () ! {IO} {
  println("Hello, " ++ name)
}
```

### ✅ Correct (ADTs with pattern matching):
```ailang
module examples/option_demo

type Option[a] = Some(a) | None

export func getOrElse[a](opt: Option[a], default: a) -> a {
  match opt {
    Some(x) => x,
    None => default
  }
}

export func main() -> int {
  getOrElse(Some(42), 0)
}
```

### ❌ Wrong (using unimplemented features):
```ailang
-- ❌ NO for loops
for i in [1, 2, 3] {
  print(i)
}

-- ❌ NO mutable variables
var x = 5
x = x + 1

-- ❌ NO pattern guards (yet)
match value {
  Some(x) if x > 0 => x,  -- if guard parsed but not evaluated
  None => 0
}
```

## What WORKS in v0.2.0-rc1

### 1. Module Declarations
```ailang
module examples/my_module

import std/io (println, print)
import std/fs (readFile, writeFile)

-- All module code must be inside function declarations
export func main() -> () ! {IO} {
  println("Module executed!")
}
```

### 2. Function Declarations
```ailang
-- Simple function
export func add(x: int, y: int) -> int {
  x + y
}

-- Function with effects
export func greet() -> () ! {IO} {
  println("Hello!")
}

-- Generic function
export func identity[a](x: a) -> a {
  x
}

-- Multi-statement function body
export func compute() -> int {
  let x = 10;
  let y = 20;
  x + y
}
```

### 3. Import Statements
```ailang
import std/io (println, print, readLine)
import std/fs (readFile, writeFile, exists)
import std/option (Option, Some, None)
import std/list (map, filter, foldl)
```

### 4. Pattern Matching
```ailang
type Option[a] = Some(a) | None

match Some(42) {
  Some(x) => x * 2,
  None => 0
}
-- Result: 84

-- Nested patterns
match Some(Some(10)) {
  Some(Some(x)) => x,
  Some(None) => 0,
  None => -1
}

-- List patterns
match [1, 2, 3] {
  [] => "empty",
  [x] => "one element",
  [x, y] => "two elements",
  _ => "many elements"
}
```

### 5. Algebraic Data Types (ADTs)
```ailang
-- Simple enum
type Color = Red | Green | Blue

-- Parameterized type
type Option[a] = Some(a) | None

-- Multiple fields
type Result[a, e] = Ok(a) | Err(e)

-- Using constructors
let value = Some(42)
let error = Err("file not found")
```

### 6. Effect System
```ailang
-- Functions must declare their effects
export func readAndPrint() -> () ! {IO, FS} {
  let content = readFile("data.txt");
  println(content)
}

-- Pure functions have no effect annotation
export func double(x: int) -> int {
  x * 2
}
```

### 7. Built-in Functions
```ailang
-- IO builtins (require IO capability)
println("text")      -- Print with newline
print("text")        -- Print without newline
readLine()           -- Read from stdin

-- FS builtins (require FS capability)
readFile("path.txt")       -- Read file to string
writeFile("path.txt", "")  -- Write string to file
exists("path.txt")         -- Check if file exists
```

### 8. Type Inference
```ailang
-- Types are inferred automatically
export func add(x: int, y: int) -> int {
  x + y  -- Knows this is int + int
}

-- Generic types work
export func identity[a](x: a) -> a {
  x  -- Works for any type a
}
```

### 9. Lambda Functions (for local use)
```ailang
export func apply() -> int {
  let double = \x. x * 2;
  double(21)
}
```

### 10. Let Expressions (up to 3 levels)
```ailang
let x = 5;
let y = 10;
let z = x + y;
z * 2
```

## What DOESN'T WORK (Yet)

### ❌ DO NOT USE These:

```ailang
-- ❌ NO for loops (use recursion or list operations)
for i in range(10) {
  println(i)
}

-- ❌ NO while loops
while condition {
  doSomething()
}

-- ❌ NO mutable variables
var x = 5
x = x + 1

-- ❌ NO pattern guards (parsed but not evaluated)
match value {
  Some(x) if x > 0 => x,
  None => 0
}

-- ❌ NO error propagation operator
let content = readFile("file.txt")?  -- ? not implemented
```

## Running AILANG Programs

### Basic execution:
```bash
ailang run examples/hello.ail --entry main
```

### With effects (requires capability grants):
```bash
# Grant IO capability
ailang run app.ail --entry main --caps IO

# Grant multiple capabilities
ailang run app.ail --entry main --caps IO,FS

# With sandbox for FS
AILANG_FS_SANDBOX=/tmp ailang run app.ail --entry main --caps FS
```

### With arguments:
```bash
ailang run app.ail --entry process --args-json '42'
ailang run app.ail --entry greet --args-json '"Alice"'
```

## Common Patterns

### Recursion (instead of loops)
```ailang
export func factorial(n: int) -> int {
  if n <= 1
  then 1
  else n * factorial(n - 1)
}

export func sum(xs: [int]) -> int {
  match xs {
    [] => 0,
    [head, ...tail] => head + sum(tail)
  }
}
```

### Option/Result handling
```ailang
import std/option (Option, Some, None)

export func divide(x: int, y: int) -> Option[int] {
  if y == 0
  then None
  else Some(x / y)
}

export func getOrElse[a](opt: Option[a], default: a) -> a {
  match opt {
    Some(x) => x,
    None => default
  }
}
```

### List operations
```ailang
import std/list (map, filter, foldl)

export func doubleAll(xs: [int]) -> [int] {
  map(\x. x * 2, xs)
}

export func sum(xs: [int]) -> int {
  foldl(\acc. \x. acc + x, 0, xs)
}
```

## Error Messages

If you see these errors:

- `"entrypoint 'X' not found"` - Make sure function is exported and name matches
- `"capability X required"` - Add `--caps X` flag when running
- `"module not found"` - Check import path matches file location
- `"too many nested let expressions"` - Limit is 3 levels, split into functions

## Summary

**DO:**
- Use `module` declarations
- Use `export func` for all functions
- Declare effects with `! {IO, FS}`
- Use pattern matching for ADTs
- Use recursion instead of loops
- Import from `std/io`, `std/fs`, `std/option`
- Request capabilities when running: `--caps IO,FS`

**DON'T:**
- Use `for`, `while`, or `var`
- Use pattern guards (not evaluated yet)
- Use `?` error propagation
- Nest `let` more than 3 levels
- Forget to grant capabilities for effects

**WHEN IN DOUBT:** Look at working examples in `examples/` directory.
```

---

## Version History

- **v0.2.0-rc1** (October 2025): Module execution, effects (IO/FS), pattern matching, ADTs
- **v0.1.0** (October 2025): Type system foundation, REPL
- **v0.3.0+** (Future): Quasiquotes, concurrency, session types

---

## Testing Your Prompt

To test how well this prompt teaches AILANG:

```bash
# Run baseline evaluation with all 3 models
./tools/run_benchmark_suite.sh

# Or test individual model
ailang eval --benchmark fizzbuzz --model claude-sonnet-4-5 --seed 42

# Check success rate
cat eval_results/summary.csv | grep ailang | awk -F, '$9=="true" {s++} END {print s "/" NR " successes"}'
```

**Target KPI**: 80%+ first-attempt success rate for simple benchmarks (fizzbuzz, arithmetic).

**Current Baseline** (October 2, 2025):
- GPT-5: 0% (generates imperative code with features not yet implemented)
- Need to update benchmarks with this v0.2.0 prompt

---

## Improving This Prompt

If AI models consistently fail on certain patterns:

1. **Analyze failures**: Check `eval_results/*.json` for common errors
2. **Add examples**: Show the failing pattern in "What WORKS" section
3. **Strengthen warnings**: Move common mistakes to "DO NOT USE" section
4. **Re-test**: Run benchmarks again to measure improvement

**Example improvement cycle:**

```bash
# Run baseline
ailang eval --benchmark fizzbuzz --model gpt5 --seed 42 --langs ailang
# Check generated code and errors

# Update this prompt based on error patterns
# Add more examples of module structure

# Run again
ailang eval --benchmark fizzbuzz --model gpt5 --seed 42 --langs ailang
# Measure improvement
```

---

## Using This Prompt

### In Benchmark YAML files:

```yaml
prompt: |
  {PASTE FULL AILANG PROMPT HERE}

  Now write a program in AILANG that implements FizzBuzz from 1 to 100.

  Remember:
  - Use module declaration
  - Export a main() function
  - Use println from std/io
  - Program will be run with: ailang run fizzbuzz.ail --entry main --caps IO
```

### In AI chat interfaces:

```
<paste prompt above>

User: Write a function to calculate fibonacci numbers in AILANG
```

### In automated evaluation:

The M-EVAL system automatically includes this prompt when generating AILANG code. See:
- [docs/guides/evaluation/](evaluation/) - Evaluation framework
- [benchmarks/](../../benchmarks/) - Benchmark specifications

---

**Last Updated**: October 2, 2025 (v0.2.0-rc1)
**Maintained By**: AILANG Core Team
**Success Metric**: AI teachability (target: 80%+ success on simple benchmarks)
