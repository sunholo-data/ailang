---
title: AI Agent Integration Guide
sidebar_position: 1
description: Quick start guide for AI coding agents to integrate AILANG
---

# AILANG for AI Agents - Quick Integration Guide

**AILANG is an AI-first programming language designed for AI-assisted development.**

This guide helps AI coding agents (like Claude, GPT, Gemini) write correct AILANG code.

---

## 🚀 Quick Start (4 Steps)

### Step 1: Verify AILANG is Installed

Check if the user has AILANG installed:

```bash
ailang --version
```

**If not installed**, provide installation instructions:

```bash
# macOS (Apple Silicon)
curl -L https://github.com/sunholo-data/ailang/releases/latest/download/ailang-darwin-arm64.tar.gz | tar -xz
sudo mv ailang /usr/local/bin/

# macOS (Intel)
curl -L https://github.com/sunholo-data/ailang/releases/latest/download/ailang-darwin-amd64.tar.gz | tar -xz
sudo mv ailang /usr/local/bin/

# Linux
curl -L https://github.com/sunholo-data/ailang/releases/latest/download/ailang-linux-amd64.tar.gz | tar -xz
sudo mv ailang /usr/local/bin/
```

### Step 2: Load the AILANG Teaching Prompt

**CRITICAL**: Before writing ANY AILANG code, load the full syntax guide:

📖 **[prompts/v0.3.8.md](prompts/v0.3.8.md)** - Complete AILANG teaching prompt

**Why this matters:**
- AILANG is NOT Python, NOT Rust, NOT JavaScript
- Pure functional programming with unique syntax
- Algebraic effects with capability security
- Common mistakes are documented in the prompt

**How to use it:**
1. Read the entire prompt before generating code
2. Reference it for syntax questions
3. Follow the examples - they show the exact correct syntax

### Step 3: Write AILANG Code

**Every AILANG program MUST have this structure:**

```typescript
module benchmark/solution

import std/io (println)

export func main() -> () ! {IO} {
  println("Hello, AILANG!")
}
```

**Key syntax rules:**
- ✅ Use `func` NOT `fn`, `function`, or `def`
- ✅ Use `type Name[a] = Constructor(a) | Constructor2` for ADTs
- ✅ NO `for`/`while` loops - use recursion
- ✅ Effects must be declared: `! {IO}`, `! {FS}`, etc.
- ✅ Semicolons REQUIRED between statements in blocks
- ✅ `show` is builtin - do NOT import it
- ✅ std/prelude is AUTO-IMPORTED (no imports for `<`, `>`, `==`, `!=`)

**Common mistakes to avoid:**
- ❌ Writing Python/JavaScript/Rust syntax
- ❌ Forgetting `module` declaration
- ❌ Using loops instead of recursion
- ❌ Missing `export` keyword on functions
- ❌ Importing `show` or comparison operators

### Step 4: Run the Code

```bash
ailang run --caps IO,FS --entry main solution.ail
```

**Note**: Flags MUST come BEFORE the filename!

---

## 📋 Quick Reference Card

### Module Structure
```typescript
module benchmark/solution         // REQUIRED first line
import std/io (println)          // Import what you need
export func main() -> () ! {IO}  // Export entry point
```

### Common Imports
```typescript
import std/io (println, print, readLine)        // Console IO
import std/fs (readFile, writeFile, exists)    // File operations
import std/clock (now, sleep)                  // Time operations
import std/net (httpGet, httpPost)             // HTTP requests
```

**NO IMPORTS NEEDED:**
- Comparisons: `<`, `>`, `<=`, `>=`, `==`, `!=` (auto-imported)
- `show` function (builtin)

### Recursion (instead of loops)
```typescript
// ❌ WRONG - No loops in AILANG
for i in [1, 2, 3] { println(i) }

// ✅ CORRECT - Use recursion
export func printAll(xs: [int]) -> () ! {IO} {
  match xs {
    [] => (),
    _ => {
      println(show(head(xs)));
      printAll(tail(xs))
    }
  }
}
```

### Effects Declaration
```typescript
export func readConfig() -> string ! {FS}           // File system
export func greet(name: string) -> () ! {IO}        // Console IO
export func fetch(url: string) -> string ! {Net}   // Network
export func main() -> () ! {IO, FS, Net}            // Multiple effects
```

### Records with Updates (NEW in v0.3.6)
```typescript
let person = {name: "Alice", age: 30, city: "NYC"};

// Update fields (immutable - creates new record)
let older = {person | age: 31};
let moved = {older | city: "SF"};
```

### Pattern Matching
```typescript
type Option[a] = Some(a) | None

export func getOrElse[a](opt: Option[a], default: a) -> a {
  match opt {
    Some(x) => x,
    None => default
  }
}
```

### Multi-line ADTs (NEW in v0.3.8)
```typescript
// Single-line
type Tree = Leaf(int) | Node(Tree, int, Tree)

// Multi-line (optional leading pipe)
type Tree =
  | Leaf(int)
  | Node(Tree, int, Tree)
```

---

## 🎯 Current Success Rates (v0.3.8)

**Benchmark results (October 2025):**
- **AILANG**: 49.1% success rate (28/57 benchmarks)
- **Python** (baseline): 82.5% (47/57)
- **Improvement**: +10.5% from v0.3.7 (38.6% → 49.1%)

**Best performing model:**
- Claude Sonnet 4.5: 68.4% across all tasks

**What this means for you:**
- Expect ~50% success on complex tasks
- Simple programs work very well
- Complex recursion/pattern matching needs careful attention
- Always validate output by running `ailang run`

---

## 🚨 Common Pitfalls & Fixes

### Pitfall 1: Writing Python/JavaScript Syntax
**Problem:**
```python
for i in range(10):
    print(i)
```

**Fix:**
```typescript
module benchmark/solution
import std/io (println)

export func loop(n: int) -> () ! {IO} {
  if n >= 10
  then ()
  else {
    println(show(n));
    loop(n + 1)
  }
}

export func main() -> () ! {IO} {
  loop(0)
}
```

### Pitfall 2: Forgetting Module Declaration
**Problem:**
```typescript
func main() {
  println("hello")
}
```

**Fix:**
```typescript
module benchmark/solution
import std/io (println)

export func main() -> () ! {IO} {
  println("hello")
}
```

### Pitfall 3: Missing Semicolons in Blocks
**Problem:**
```typescript
{
  println("First")
  println("Second")  -- ❌ Parse error!
}
```

**Fix:**
```typescript
{
  println("First");
  println("Second")  -- ✅ Last statement doesn't need semicolon
}
```

### Pitfall 4: Importing show or comparisons
**Problem:**
```typescript
import std/io (println, show)        -- ❌ show not in std/io
import std/prelude (Ord, Eq)         -- ❌ auto-imported
```

**Fix:**
```typescript
import std/io (println)              -- ✅ Only println
-- show and comparisons work automatically!
```

---

## 🔍 Debugging Failed Code

If your generated code fails:

1. **Check the error message** - AILANG provides detailed type errors
2. **Verify module structure** - Must start with `module benchmark/solution`
3. **Check for Python/JS syntax** - Common mistake!
4. **Verify effects match** - `! {IO}` must match actual effects used
5. **Test incrementally** - Use REPL for quick testing: `ailang repl`

**Example debugging session:**
```bash
# Start REPL
ailang repl

# Test small expressions
λ> 1 + 2
3 :: Int

# Test your function logic
λ> let double = \x. x * 2 in double(21)
42 :: Int

# Check types
λ> :type \x. x + x
\x. x + x :: ∀α. Num α ⇒ α → α
```

---

## 📚 Full Documentation

**Essential reading:**
- **[prompts/v0.3.8.md](prompts/v0.3.8.md)** - Complete syntax guide (MUST READ before coding)
- **[GitHub examples](https://github.com/sunholo-data/ailang/tree/main/examples)** - 48+ working examples
- **limitations documentation** - Known limitations

**Online docs:**
- https://sunholo-data.github.io/ailang/ - Full documentation site
- https://sunholo-data.github.io/ailang/docs/prompts/v0.3.8 - Teaching prompt
- https://sunholo-data.github.io/ailang/docs/guides/getting-started - Getting started guide

---

## ✅ Pre-Flight Checklist

Before generating AILANG code, confirm:

- [ ] Read the full teaching prompt ([prompts/v0.3.8.md](prompts/v0.3.8.md))
- [ ] Understand: AILANG is NOT Python/JavaScript/Rust
- [ ] Know: Use recursion, NOT loops
- [ ] Know: Effects must be declared (`! {IO}`)
- [ ] Know: std/prelude is auto-imported (no manual imports for comparisons)
- [ ] Know: Every program needs `module benchmark/solution` first line

**If unsure about syntax:**
1. Check the examples in [prompts/v0.3.8.md](prompts/v0.3.8.md)
2. Look at working examples in [GitHub examples](https://github.com/sunholo-data/ailang/tree/main/examples)
3. When in doubt, use simpler constructs

---

## 🎓 Learning Path for AI Agents

**Phase 1: Basic Programs**
- Hello World with `println`
- Simple arithmetic functions
- Basic recursion (factorial, fibonacci)

**Phase 2: Data Structures**
- Records (literals, field access, updates)
- ADTs (Option, Result, List)
- Pattern matching

**Phase 3: Effects**
- IO (console input/output)
- FS (file operations)
- Error handling with Option/Result

**Phase 4: Advanced**
- Complex recursion (quicksort, trees)
- Higher-order functions
- Type classes

---

## 🤝 Best Practices

1. **Start simple** - Get basic structure working first
2. **Test incrementally** - Use REPL for quick validation
3. **Follow examples** - The teaching prompt has proven patterns
4. **Ask for clarification** - If user's request is ambiguous
5. **Provide runnable code** - Include full module structure
6. **Add comments** - Explain non-obvious recursion/pattern matching

**Good response format:**
```typescript
-- solution.ail
-- Solves [problem description]
-- Uses [key techniques]

module benchmark/solution

import std/io (println)

-- [Explain main logic]
export func main() -> () ! {IO} {
  [implementation]
}
```

---

## 📞 Getting Help

**For users:**
- Report issues: https://github.com/sunholo-data/ailang/issues
- Documentation: https://sunholo-data.github.io/ailang/

**For AI agents:**
- If syntax is unclear, reference [prompts/v0.3.8.md](prompts/v0.3.8.md)
- If examples fail, check limitations documentation
- When in doubt, generate simpler code

---

**Remember**: AILANG is designed for AI-assisted development. Your success rate will improve as you internalize the syntax patterns from the teaching prompt. Always read [prompts/v0.3.8.md](prompts/v0.3.8.md) before generating code!
