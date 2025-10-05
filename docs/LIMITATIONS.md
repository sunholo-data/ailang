# AILANG v0.3.0 Known Limitations

**Last Updated**: October 5, 2025
**Current Version**: v0.3.0 (Clock & Net Effects + Type System Fixes)

This document honestly describes what doesn't work yet in AILANG v0.3.0 and what's planned for future releases.

---

## TL;DR

**What Works**: Module execution ✅, recursion ✅, blocks ✅, records (basic) ✅, effects (IO, FS, Clock, Net) ✅, type system ✅, REPL ✅

**What Doesn't**: Record update syntax, pattern guards, error propagation `?`, deep let nesting (4+), quasiquotes, CSP concurrency

**When Fixed**: v0.4.0+ (see roadmap)

---

## Current Limitations (v0.3.0)

### 1. Record Update Syntax Not Implemented

**Status**: Record literals and field access work, but update syntax doesn't

**What works:**
```ailang
-- ✅ Record literal
let person = {name: "Alice", age: 30, city: "NYC"};

-- ✅ Field access
let name = person.name;  -- "Alice"
let age = person.age;    -- 30
```

**What doesn't work:**
```ailang
-- ❌ Record update syntax
let updated = {person | age: 31};  -- NOT IMPLEMENTED
```

**Error**: Parser error or type error

**Workaround**: Create new record literal with all fields:
```ailang
let updated = {name: person.name, age: 31, city: person.city};
```

**Planned**: M-R5b (Record Extension) - see [design_docs/planned/M-R5b_record_extension.md](../design_docs/planned/M-R5b_record_extension.md)

---

### 2. Pattern Guards Not Evaluated

**Status**: Guards parse but are not evaluated (always treated as true)

**What doesn't work:**
```ailang
match value {
  Some(x) if x > 0 => x * 2,  -- Guard 'if x > 0' is ignored!
  Some(x) => x,
  None => 0
}
```

**Behavior**: First pattern always matches (guard ignored)

**Workaround**: Use nested if-then-else:
```ailang
match value {
  Some(x) => if x > 0 then x * 2 else x,
  None => 0
}
```

**Planned**: M-R3 (Pattern Matching Polish) - guard evaluation

---

### 3. Nested Let Expression Limit

**Status**: Let expressions limited to 3 nesting levels

**What doesn't work:**
```ailang
let a = 1 in
let b = 2 in
let c = 3 in
let d = 4 in  -- ❌ 4th level fails!
a + b + c + d
```

**Error**: `PAR_NO_PREFIX_PARSE: unexpected token in expression: in`

**Workaround**: Use block expressions (v0.3.0+):
```ailang
{
  let a = 1;
  let b = 2;
  let c = 3;
  let d = 4;
  a + b + c + d
}
```

Or use module-level functions (recommended):
```ailang
func compute() -> int {
  let a = 1;
  let b = 2;
  let c = 3;
  let d = 4;
  a + b + c + d
}
```

**Planned**: May be addressed in parser refactor (v0.4.0+)

---

### 4. Error Propagation Operator `?` Not Implemented

**Status**: No syntactic sugar for error handling

**What doesn't work:**
```ailang
func readAndProcess() -> Result[string, string] ! {IO, FS} {
  let content = readFile("data.txt")?;  -- ❌ ? operator not implemented
  Ok(process(content))
}
```

**Workaround**: Manual match expressions:
```ailang
func readAndProcess() -> Result[string, string] ! {IO, FS} {
  match readFile("data.txt") {
    Ok(content) => Ok(process(content)),
    Err(e) => Err(e)
  }
}
```

**Planned**: v0.4.0+ (syntactic sugar milestone)

---

### 5. Row Polymorphism (Opt-In Experimental)

**Status**: Basic records use simple row types, full row polymorphism behind flag

**Default behavior** (v0.3.0):
- Record subsumption: functions accepting `{id: int}` work with larger records
- Row types hidden from user

**Experimental mode** (`AILANG_RECORDS_V2=1`):
- Full row polymorphism with explicit row variables
- Row types shown in error messages
- More flexible but more complex

**Limitation**: Row polymorphism not fully integrated into type system yet

**Planned**: Full integration in v0.4.0+

---

## Features Not Yet Implemented

### 1. Typed Quasiquotes

**Status**: Not implemented

**Planned syntax** (future):
```ailang
let query = sql"""SELECT * FROM users WHERE age > ${minAge: int}"""
```

**Planned**: v0.5.0+ (Quasiquotes milestone)

---

### 2. CSP Concurrency

**Status**: Not implemented

**Planned syntax** (future):
```ailang
func worker(ch: Channel[Task]) ! {Async} {
  loop {
    let task <- ch;
    ch <- process(task)
  }
}
```

**Planned**: v0.4.0+ (Concurrency milestone)

---

### 3. Session Types

**Status**: Not implemented

**Planned syntax** (future):
```ailang
type Session = Send[int] ; Receive[string] ; End
```

**Planned**: v0.5.0+ (with CSP concurrency)

---

### 4. Property-Based Testing

**Status**: Syntax not finalized

**Planned syntax** (future):
```ailang
property "associativity" {
  forall(x: int, y: int, z: int) =>
    (x + y) + z == x + (y + z)
}
```

**Planned**: v0.4.0+ (Testing milestone)

---

## What DOES Work (v0.3.0)

### ✅ Module Execution (v0.2.0+)

**Fully functional**:
```ailang
module examples/demo
import std/io (println)

export func main() -> () ! {IO} {
  println("Hello, World!")
}
```

```bash
ailang run --caps IO --entry main examples/demo.ail
# Output: Hello, World!
```

### ✅ Recursion (v0.3.0-alpha2)

**Self-recursive and mutually-recursive functions**:
```ailang
func factorial(n: int) -> int {
  if n <= 1 then 1
  else n * factorial(n - 1)
}

func isEven(n: int) -> bool {
  if n == 0 then true else isOdd(n - 1)
}

func isOdd(n: int) -> bool {
  if n == 0 then false else isEven(n - 1)
}
```

### ✅ Block Expressions (v0.3.0-alpha2)

**Sequencing with semicolons**:
```ailang
{
  println("Computing...");
  println("Result:");
  42
}
```

### ✅ Records (v0.3.0-alpha2)

**Literals and field access**:
```ailang
let person = {name: "Alice", age: 30};
let name = person.name;
```

### ✅ Effect System (v0.2.0, v0.3.0-alpha4)

**IO, FS, Clock, Net effects with capability security**:
```ailang
import std/io (println)
import std/clock (now, sleep)
import std/net (httpGet)

func demo() -> () ! {IO, Clock, Net} {
  let start = now();
  let response = httpGet("https://httpbin.org/get");
  println(response);
  let elapsed = now() - start;
  println("Took " ++ show(elapsed) ++ "ms")
}
```

### ✅ Type System

- Hindley-Milner type inference
- Type classes (Num, Eq, Ord, Show)
- Dictionary passing
- Row polymorphism (subsumption)
- Effect tracking
- Algebraic data types
- Pattern matching

### ✅ REPL

- Interactive type checking
- Full evaluation
- Command history
- Type inspection (`:type`)
- Instance inspection (`:instances`)

---

## Execution Modes

AILANG supports two execution modes:

### 1. Simple Scripts (No Module Declaration)

**What works**:
```ailang
-- examples/simple.ail
let x = 5 in x * 2
```

**Limitations**:
- Cannot use `func`, `type`, `import`, `export` keywords
- No effects (IO, FS, etc.)
- Single expression only

**Run with**: `ailang run simple.ail`

### 2. Module Files (With Module Declaration)

**What works**:
```ailang
module examples/demo

import std/io (println)

export func main() -> () ! {IO} {
  println("Hello!")
}
```

**Features**:
- Full language features
- Effects with capability security
- Cross-module imports
- Multiple functions

**Run with**: `ailang run --caps IO --entry main demo.ail`

---

## Migration Path

### From v0.2.0 to v0.3.0

**No breaking changes** - all v0.2.0 code runs in v0.3.0.

**New features**:
- Recursion support
- Block expressions
- Records (basic)
- Clock & Net effects
- Type system fixes (modulo, float comparison)

### To v0.4.0 (Planned)

**Expected additions**:
- Record update syntax
- Pattern guards
- Error propagation `?`
- Enhanced Net effect (custom headers, JSON parsing)
- More micro examples

**No breaking changes expected** - v0.4.0 will be additive.

---

## Getting Help

- **Canonical Syntax Reference**: [prompts/v0.3.0.md](../prompts/v0.3.0.md) (for AI code generation)
- **Documentation**: [docs/](.) directory
- **Examples**: [examples/STATUS.md](../examples/STATUS.md) - 48/66 passing (72.7%)
- **REPL Help**: Type `:help` in the REPL
- **Issue Tracker**: https://github.com/sunholo-data/ailang/issues

---

## Frequently Asked Questions

### Q: Can I use AILANG productively in v0.3.0?

**A**: Yes! Module execution works, recursion works, effects work. You can:
- Build command-line tools with IO/FS
- Write recursive algorithms
- Use HTTP APIs with Net effect
- Measure execution time with Clock effect

### Q: What should I avoid in v0.3.0?

**A**: Avoid:
- Record update syntax (use new literals instead)
- Pattern guards (use nested if-then-else)
- Deep let nesting (use blocks or functions)
- Quasiquotes (not implemented)

### Q: When is v0.4.0 coming?

**A**: TBD - see [design_docs/planned/](../design_docs/planned/) for roadmap.

### Q: Will my v0.3.0 code break in v0.4.0?

**A**: No. v0.4.0 will be additive (new features, no breaking changes).

---

## Conclusion

v0.3.0 represents a **functional programming language with module execution, recursion, effects, and basic records**.

**What works**: Module execution, recursion, blocks, records (literals + field access), effects (IO, FS, Clock, Net), type classes, ADTs, pattern matching, REPL.

**What's coming**: Record updates, pattern guards, error propagation, quasiquotes, concurrency.

---

*For current capabilities, see [prompts/v0.3.0.md](../prompts/v0.3.0.md) (canonical AI teaching prompt)*
*For implementation status, see [docs/reference/implementation-status.md](reference/implementation-status.md)*
