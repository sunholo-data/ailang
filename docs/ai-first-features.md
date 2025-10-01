# AI-First Language Features

This document explains how AILANG's unique features support AI-assisted software development. Each feature is designed to make code more **predictable**, **analyzable**, and **safe** for AI reasoning.

---

## 🎯 What Makes AILANG "AI-First"?

Traditional programming languages are designed for **human developers**. AILANG is designed for **human-AI collaboration**:

- **Explicit over Implicit**: Side effects, types, and resources are declared, not hidden
- **Deterministic by Default**: Execution is reproducible for training data generation
- **Machine-Readable Semantics**: Static guarantees that AI can verify without running code
- **Structured Traces**: Execution history captures reasoning for model training

---

## ✅ Implemented Features

### 1. Effect System (Type-Level) — v0.1.0

**Status**: ✅ Complete (M-P4, ~1,060 LOC)

#### What It Does
Functions declare **side effects** in their type signatures:

```ailang
pure func calculate(x: int) -> int {
  x * 2  -- Guaranteed no I/O, no network, no mutation
}

func readConfig(path: string) -> string ! {FS} {
  readFile(path)  -- Declares filesystem access upfront
}

func main() -> () ! {IO, FS} {
  let config = readConfig("app.conf")
  print(config)  -- Combines IO and FS effects
}
```

#### Why AI Needs This

**Problem in Traditional Languages:**
```python
def do_stuff():
    print("Debug")          # Hidden I/O!
    requests.get("api.com") # Hidden network call!
    open("data.txt")        # Hidden filesystem access!
    return 42
```
AI must **execute or deeply analyze** the function body to understand what resources it touches.

**Solution in AILANG:**
```ailang
func doStuff() -> int ! {IO, Net, FS}
```
AI sees `! {IO, Net, FS}` in the signature → **instant understanding** of effects. No execution required.

#### AI Benefits

1. **Static Analysis**
   - AI can verify: "Does this function access the network?" → Check signature
   - No need to trace through call chains or analyze implementation

2. **Safe Refactoring**
   - AI can suggest: "This function uses `{IO}` but isn't declared effectful—add `! {IO}`"
   - Prevents moving effectful code into pure contexts (compile error)

3. **Security Auditing**
   - AI can identify: "This function is marked `pure` but calls `readFile()`" → Security violation
   - Effect boundaries enforce capability discipline

4. **Training Data Quality**
   - Pure functions have **deterministic traces** (same input → same output)
   - Effectful functions have **labeled traces** (IO happened at line 42)
   - Enables high-quality supervised learning datasets

#### Example: AI Reasoning

**Query**: "Can I safely cache the result of `processData()`?"

**Without Effects:**
```python
def process_data(x):
    return x * 2  # Looks pure... but is it?
```
AI must guess or execute to verify (expensive, risky).

**With Effects:**
```ailang
pure func processData(x: int) -> int
```
AI sees `pure` → **guaranteed cacheable** (no side effects).

```ailang
func processData(x: int) -> int ! {DB}
```
AI sees `! {DB}` → **not cacheable** (reads from database).

---

### 2. Structured Error Reporting — v3.2.0

**Status**: ✅ Complete (~1,500 LOC)

#### What It Does
Errors are **machine-readable JSON** with stable schemas:

```json
{
  "schemaVersion": "v1.0.0",
  "errors": [{
    "code": "TYPE001",
    "severity": "error",
    "message": "Type mismatch: expected Int, got String",
    "location": {
      "file": "main.ail",
      "line": 42,
      "column": 10
    },
    "context": {
      "expected": "Int",
      "actual": "String"
    }
  }]
}
```

#### Why AI Needs This

**Problem in Traditional Languages:**
```
Error: type mismatch (line 42)
```
Unstructured, inconsistent, hard to parse programmatically.

**Solution in AILANG:**
Every error has:
- **Stable error code** (TYPE001, PAR_EFF002_UNKNOWN)
- **Schema version** (semantic versioning for breaking changes)
- **Structured context** (expected type, actual type, suggestions)
- **Precise location** (file, line, column)

#### AI Benefits

1. **Error Classification**
   - AI can group errors by code: "All TYPE001 errors in this project"
   - Enables pattern detection: "This module has 12 effect mismatches"

2. **Automated Fixes**
   - Error code `PAR_EFF002_UNKNOWN` → AI knows to suggest valid effect names
   - Structured context provides **all information** needed to generate fix

3. **Training on Errors**
   - Collect (error, fix) pairs with stable schemas
   - Train models to predict fixes for error codes
   - Schema versioning prevents training data corruption

4. **Deterministic Diagnostics**
   - Same code → same error messages (stable across runs)
   - Reproducible bug reports for AI debugging assistance

#### Example: AI Error Fixing

**Error:**
```json
{
  "code": "PAR_EFF002_UNKNOWN",
  "message": "Unknown effect 'io'",
  "context": {
    "provided": "io",
    "suggestion": "IO"
  }
}
```

**AI Action:**
```diff
- func readFile(path: string) -> string ! {io}
+ func readFile(path: string) -> string ! {IO}
```

AI extracts `suggestion` field → applies fix automatically.

---

### 3. Module System with Deterministic Resolution — v3.3.0

**Status**: ✅ Complete (~2,800 LOC)

#### What It Does
Modules load **predictably** with cycle detection and dependency ordering:

```ailang
-- math.ail
module math

export func add(x: int, y: int) -> int { x + y }

-- main.ail
import math (add)

add(2, 3)  -- 5
```

#### Why AI Needs This

**Problem in Traditional Languages:**
- Python: Import side effects, circular imports fail at runtime
- JavaScript: Module loading order is fragile
- Result: AI can't reason about dependencies without execution

**Solution in AILANG:**
- **Manifest files** (`manifest.json`) declare dependencies explicitly
- **Cycle detection** at compile time (no runtime surprises)
- **Deterministic ordering** (same code → same load order)

#### AI Benefits

1. **Dependency Analysis**
   - AI can read manifest → know all dependencies upfront
   - No hidden imports or dynamic module loading

2. **Safe Refactoring**
   - AI can suggest: "Move function `foo` to module `utils`"
   - Compiler verifies no cycles introduced

3. **Training Data Isolation**
   - Each module has **well-defined boundaries**
   - AI can train on module-level patterns without cross-contamination

4. **Reproducible Builds**
   - Same source + manifest → same compiled output
   - Enables deterministic training data generation

---

### 4. Type Classes with Dictionary-Passing — v2.3.0

**Status**: ✅ Complete for REPL (~1,200 LOC)

#### What It Does
Type classes provide **ad-hoc polymorphism** with explicit dictionaries:

```ailang
-- Surface syntax
let identity = \x. x  -- Polymorphic: ∀α. α → α

-- Core representation (ANF + dictionaries)
let identity = λ(x: α). x

-- Type class constraint
let double = \x. x + x  -- ∀α. Num α ⇒ α → α

-- Dictionary passing (automatic)
double[Int](21) ≡ (+)[Int](21, 21)
```

#### Why AI Needs This

**Problem in Traditional Languages:**
- Java: Overloading resolved at compile time (complex rules)
- Python: Duck typing (runtime errors, no static guarantees)
- Result: AI struggles to predict which method gets called

**Solution in AILANG:**
- Type class instances are **first-class values** (dictionaries)
- Method resolution is **explicit** in Core representation
- AI sees: `(+)[Int]` vs `(+)[Float]` (no ambiguity)

#### AI Benefits

1. **Predictable Dispatch**
   - AI can trace: `x + y` → `(+)[Dict](x, y)` → method lookup
   - No hidden overload resolution or method tables

2. **Instance Reasoning**
   - AI can verify: "Does type `T` implement `Eq`?" → Check instance environment
   - Static guarantee (no runtime type errors)

3. **Training on Polymorphism**
   - Dictionary passing is **explicit** in Core AST
   - AI learns: "Num constraint → dictionary parameter"
   - Generalizes to new type classes

4. **Debugging Support**
   - REPL command `:instances` shows all available dictionaries
   - AI can explain: "This code fails because `Ord[MyType]` is missing"

#### Example: AI Type Debugging

**User Code:**
```ailang
let compare = \x y. x < y
```

**REPL Output:**
```
compare :: ∀α. Ord α ⇒ α → α → Bool
```

**AI Analysis:**
- Sees constraint: `Ord α`
- Checks instance environment: `Ord[Int]` ✅, `Ord[String]` ❌
- Suggests: "Add `instance Ord[String]` to use with strings"

---

### 5. Inline Testing with Properties — v0.1.0

**Status**: ✅ Syntax defined, implementation planned

#### What It Does
Tests and properties live **inside function definitions**:

```ailang
pure func reverse[a](list: [a]) -> [a]
  tests [
    ([], []),
    ([1], [1]),
    ([1,2,3], [3,2,1])
  ]
  properties [
    forall(xs: [a]) => reverse(reverse(xs)) == xs,
    forall(xs: [a]) => length(reverse(xs)) == length(xs)
  ]
{
  match list {
    [] => [],
    [x, ...xs] => reverse(xs) ++ [x]
  }
}
```

#### Why AI Needs This

**Problem in Traditional Languages:**
```python
def reverse(lst):
    return lst[::-1]

# Tests in separate file: test_reverse.py
def test_reverse():
    assert reverse([]) == []
    assert reverse([1,2,3]) == [3,2,1]
```
Tests are **disconnected** from implementation. AI must search multiple files to understand correctness.

**Solution in AILANG:**
- Tests **embedded** in function definition
- Properties specify **invariants** (algebraic laws)
- AI sees function + tests + properties as **single unit**

#### AI Benefits

1. **Specification Inference**
   - AI reads: `tests [...]` → understands expected behavior
   - No need to parse docstrings or search test files
   - Tests are **executable documentation**

2. **Property-Based Verification**
   - AI sees: `forall(xs) => reverse(reverse(xs)) == xs`
   - Learns: "reverse is its own inverse"
   - Generalizes to suggest properties for other functions

3. **Contract Learning**
   - Training data: (function, tests, properties) → implementation
   - AI learns: "If function has property P, use algorithm A"
   - Enables specification-driven code generation

4. **Regression Prevention**
   - AI suggests: "Add test case for edge case X"
   - Tests evolve with implementation (same file)
   - No orphaned test suites

#### Example: AI Test Generation

**User provides:**
```ailang
pure func factorial(n: int) -> int {
  if n <= 1 then 1 else n * factorial(n - 1)
}
```

**AI suggests:**
```ailang
pure func factorial(n: int) -> int
  tests [
    (0, 1),
    (1, 1),
    (5, 120)
  ]
  properties [
    forall(n: int) where n >= 0 => factorial(n) >= 1,
    forall(n: int) where n > 0 => factorial(n) == n * factorial(n-1)
  ]
{
  if n <= 1 then 1 else n * factorial(n - 1)
}
```

---

### 6. Deterministic Execution — v0.2.0 (Planned)

**Status**: 🚧 Design complete

#### What It Does
All **non-determinism** is explicit and controlled:

```ailang
import std/random (Rand)
import std/time (Clock)

// Random with explicit seed
func simulation(seed: int) -> Result ! {Rand} {
  withSeed(seed) { rng =>
    let x = random(rng)  -- Deterministic given seed
    processData(x)
  }
}

// Virtual time for testing
func timedProcess() -> Result ! {Clock} {
  withVirtualTime(epoch) { clock =>
    let start = now(clock)
    doWork()
    let elapsed = now(clock) - start
    elapsed
  }
}
```

#### Why AI Needs This

**Problem in Traditional Languages:**
```python
import random
import time

def process():
    x = random.random()  # Different every run!
    time.sleep(1)        # Real wall-clock time
    return x * 2
```
Same code → **different results** every execution. AI can't reproduce bugs or verify fixes.

**Solution in AILANG:**
- Randomness requires `Rand` capability with **explicit seed**
- Time operations use `Clock` capability with **virtual time**
- Same seed + virtual time → **identical execution trace**

#### AI Benefits

1. **Reproducible Training Data**
   - Run program with seed=42 → get trace T
   - Run again with seed=42 → get **same trace T**
   - No flaky training examples

2. **Deterministic Debugging**
   - User reports bug with seed=12345
   - AI reproduces **exact same failure**
   - Fix verification is deterministic

3. **Testing Without Flakiness**
   - Tests use fixed seeds: `withSeed(0)`
   - Time-dependent tests use virtual time
   - 100% reproducible test suites

4. **Simulation Accuracy**
   - Monte Carlo simulations with known seeds
   - AI can verify: "Did simulation run correctly?"
   - Compare outputs deterministically

---

### 7. Structured Error Context — v3.2.0

**Status**: ✅ Complete (extends JSON error infrastructure)

#### What It Does
Errors carry **typed, structured context** with suggestions:

```ailang
type Error = {
  message: string,
  code: ErrorCode,
  context: [(string, exists a. (a, Show[a]))],  // Typed bindings!
  suggestions: [string],
  trace: [StackFrame],
  effects: [Effect]
}

// Example error
Error {
  message: "Division by zero",
  code: "MATH_DIV_ZERO",
  context: [
    ("numerator", (100.0, Show[Float])),
    ("denominator", (0.0, Show[Float])),
    ("operation", ("divide", Show[String]))
  ],
  suggestions: [
    "Check if denominator is zero before dividing",
    "Use Result type: divide(a, b) -> Result[float, string]"
  ],
  trace: [...],
  effects: [Trace]
}
```

#### Why AI Needs This

**Problem in Traditional Languages:**
```python
# Runtime error
ZeroDivisionError: division by zero
  File "main.py", line 42, in calculate
```
Error message is **unstructured text**. AI must parse strings to extract context.

**Solution in AILANG:**
- Errors are **typed values** (not exceptions)
- Context is **key-value pairs with types**
- Suggestions are **machine-readable** (not prose)
- Every value is **showable** (existential types + Show constraint)

#### AI Benefits

1. **Automated Error Analysis**
   - AI extracts: `code: "MATH_DIV_ZERO"` → known error pattern
   - Reads context: `denominator: 0.0` → root cause identified
   - No string parsing required

2. **Smart Suggestions**
   - Error includes: `suggestions: [...]`
   - AI applies suggestion directly
   - Learns: "For error X, apply fix Y"

3. **Error Classification**
   - Group errors by `code` field
   - Identify patterns: "All MATH errors in module M"
   - Prioritize fixes by frequency

4. **Training on Failures**
   - Collect: (code, context, error, fix) triples
   - Train model to predict fixes from context
   - Rich typed context improves accuracy

---

## 🚧 Planned Features (Future Versions)

### 8. Typed Quasiquotes — v0.2.0 (Planned)

**Goal**: Prevent injection attacks with **compile-time validation**

```ailang
-- SQL quasiquote with type checking
let query = sql"""
  SELECT * FROM users
  WHERE age > ${minAge: int}
"""
-- Compiler verifies: minAge is Int (not String)
-- Runtime: Automatic parameter binding (no SQL injection)

-- HTML quasiquote with XSS prevention
let page = html"""
  <div>${content: SafeHtml}</div>
"""
-- Compiler ensures: content is sanitized
```

#### AI Benefits
- **Static Security**: AI verifies no injection vulnerabilities (compile-time check)
- **Type-Safe Templates**: AI suggests correct parameter types
- **Training Data**: (template, type, output) triples for learning safe patterns

---

### 9. Capability-Based Security — v0.2.0 (Planned)

**Status**: 🚧 Design complete

#### What It Does
Effects require **explicit capability passing** for permission control:

```ailang
import std/io (FS, Net)

// Function declares what capabilities it needs
func processData(fs: FS, net: Net) -> Result ! {FS, Net} {
  with fs, net {
    let config = readFile(fs, "config.ail")?
    let response = httpGet(net, config.url)?
    writeFile(fs, "output.ail", response)?
  }
}

// Pure functions cannot receive capabilities
pure func calculate(x: int) -> int {
  // readFile(...) would be compile error - no FS capability!
  x * 2
}

// Caller must explicitly pass capabilities
func main() -> () ! {IO, FS, Net} {
  with FS, Net {
    processData(FS, Net)
  }
}
```

#### Why AI Needs This

**Problem in Traditional Languages:**
```python
def process_data():
    open("config.txt")      # Can access ANY file!
    requests.get("...")     # Can reach ANY URL!
    os.system("rm -rf /")   # Can execute ANYTHING!
```
Functions have **ambient authority** - unlimited access to resources.

**Solution in AILANG:**
- Capabilities are **values** passed explicitly
- Functions declare required capabilities in signature
- No ambient authority (principle of least privilege)

#### AI Benefits

1. **Permission Reasoning**
   - AI sees: `func f(fs: FS)` → knows function needs filesystem access
   - No hidden global state or implicit permissions
   - Static verification of resource access

2. **Security Analysis**
   - AI traces: "Does this code access the network?" → Check for `Net` capability
   - Audit permission flow: "How did this function get `FS` access?"
   - Prevents privilege escalation at compile time

3. **Safe Sandboxing**
   - AI can suggest: "This function doesn't need `Net`, remove it"
   - Create restricted environments by limiting capabilities
   - Test dangerous code with mock capabilities

4. **Training on Security**
   - Collect: (function, required capabilities) pairs
   - AI learns: "File I/O functions need `FS` capability"
   - Suggest capability requirements for new code

#### Example: AI Security Audit

**User Code:**
```ailang
func readConfig() -> string ! {FS} {
  readFile("config.ail")
}
```

**AI Error:**
```
Error: Missing capability parameter
  Function has effect {FS} but no FS capability in parameters

Suggestion:
  func readConfig(fs: FS) -> string ! {FS} {
    readFile(fs, "config.ail")
  }
```

---

### 10. CSP Concurrency with Session Types — v0.3.0 (Planned)

**Goal**: Verify **protocol correctness** at compile time

```ailang
-- Session type: send Int, receive String, close
type Protocol = !Int . ?String . End

func worker(ch: Channel[Protocol]) ! {Async} {
  ch <- 42             -- Send: OK (matches !Int)
  let response <- ch   -- Receive: OK (matches ?String)
  close(ch)            -- Close: OK (matches End)
}

-- Protocol violation caught at compile time
func bad(ch: Channel[Protocol]) ! {Async} {
  let x <- ch  -- ERROR: Expected !Int, got ?String
}
```

#### AI Benefits
- **Deadlock Prevention**: AI verifies communication patterns (no runtime hangs)
- **Protocol Generation**: AI suggests session types from usage patterns
- **Training on Concurrency**: Safe concurrent programs for model learning

---

### 11. AI Training Data Export — v0.4.0 (Planned)

**Goal**: Generate **high-quality datasets** for AI model training

```bash
ailang export-training --format jsonl program.ail
```

**Output:**
```jsonl
{"input": "let x = 1 + 2", "output": 3, "trace": [...], "effects": []}
{"input": "readFile(\"data.txt\")", "output": "...", "trace": [...], "effects": ["FS"]}
```

#### AI Benefits
- **Supervised Learning**: (input, output, trace) triples with full context
- **Effect Labeling**: Training data tagged with side effects
- **Deterministic Traces**: Same input → same trace (reproducible datasets)
- **Incremental Learning**: Export new programs → update model

---

## 📊 Feature Comparison

| Feature | Traditional Languages | AILANG (AI-First) |
|---------|----------------------|-------------------|
| **Side Effects** | Hidden (Python, JS) | Explicit (`! {IO}`) |
| **Error Reporting** | Unstructured strings | JSON with stable schemas |
| **Module Loading** | Runtime, fragile | Compile-time, deterministic |
| **Type Classes** | Complex overloading | Explicit dictionaries |
| **Inline Testing** | Separate test files | Tests + properties in function |
| **Determinism** | Random/time implicit | Explicit seeds/virtual time |
| **Error Context** | String messages | Typed context + suggestions |
| **String Templates** | Injection-prone | Type-checked quasiquotes |
| **Capabilities** | Ambient authority | Explicit capability passing |
| **Concurrency** | Race conditions | Session-type verified |
| **Training Data** | Manual curation | Automatic export |

---

## 🎓 Design Principles

1. **Explicitness**: AI sees what code does without execution
2. **Determinism**: Same input → same output (reproducible traces)
3. **Machine Readability**: JSON schemas, stable error codes, typed ASTs
4. **Static Guarantees**: Compile-time proofs (no runtime surprises)
5. **Training-Friendly**: Every language feature exports structured data

---

## 🚀 Getting Started with AI Features

### Run Effect-Checked Code
```bash
ailang run examples/effects_basic.ail
```

### Export Training Data (planned)
```bash
ailang export-training program.ail > training.jsonl
```

### Query Type Information
```bash
ailang repl
λ> :type readFile
readFile :: string -> string ! {FS}
```

### Analyze Module Dependencies
```bash
cat examples/manifest.json
```

---

## 📚 Further Reading

- [Effect System Design (M-P4)](../design_docs/20251001/M-P4.md)
- [Structured Errors (v3.2.0)](../design_docs/20241001/v3.2.0-implementation.md)
- [Module System (v3.3.0)](../design_docs/20241001/v3.3.0-implementation.md)
- [Type Classes (v2.3.0)](../design_docs/20241001/v2.3.0-implementation.md)

---

## 🤝 Contributing

Help make AILANG more AI-friendly! Areas for contribution:

- **New Effect Types**: Propose additional effects (GPU, Memory, etc.)
- **Error Schemas**: Extend JSON error formats
- **Training Exporters**: Add support for new ML frameworks
- **Quasiquote Validators**: Implement new DSL embeddings

See [Development Guide](guides/development.md) for details.

---

*Last Updated: October 1, 2025 (v0.1.0-dev)*
