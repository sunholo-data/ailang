---
sidebar_position: 1
title: Introduction
description: AILANG is an AI-first programming language designed for AI-assisted development
slug: /
---

# AILANG: AI-First Programming Language

Welcome to the official documentation for AILANG, an experimental programming language designed from the ground up for AI-assisted software development.

## What is AILANG?

AILANG is a pure functional programming language that makes AI assistance a first-class citizen. It features:

- **Pure Functional Programming**: Immutable data and explicit effects
- **Algebraic Effects System**: Track and control side effects in the type system
- **Type Classes**: Num, Eq, Ord, Show with dictionary-passing semantics
- **Pattern Matching**: Algebraic data types with exhaustiveness checking
- **Auto-Import Prelude**: Zero imports needed for type classes and comparisons
- **AI-Optimized Design**: Generate structured execution traces for model training

## Quick Example

```typescript
-- Module with effects (v0.3.0)
module examples/hello

import std/io (println)
import std/net (httpGet)

-- Recursive factorial function
export func factorial(n: int) -> int {
  if n <= 1 then 1 else n * factorial(n - 1)
}

-- Function with multiple effects
export func main() -> () ! {IO, Net} {
  println("Factorial of 5 is:");
  println(show(factorial(5)));

  let response = httpGet("https://api.example.com/data");
  println(response)
}
```

Run with: `ailang run --caps IO,Net --entry main examples/hello.ail`

## Getting Started

- **[Installation Guide](/docs/guides/getting-started)**
  Install AILANG and run your first program

- **[Language Tutorial](/docs/guides/development)**
  Learn the basics of AILANG programming

- **[Language Reference](/docs/reference/language-syntax)**
  Complete syntax and semantics reference

- **[REPL Commands](/docs/reference/repl-commands)**
  Interactive development with the AILANG REPL

## Current Status: v0.3.8 (October 2025)

AILANG v0.3.8 is now available with multi-line ADT parser fixes and operator lowering improvements! Check the [implementation status](/docs/reference/implementation-status) for complete details.

### ✅ Working Features (v0.3.8)
- **Recursion** - Self-recursion, mutual recursion, with stack overflow protection
- **Block Expressions** - Multi-statement blocks with proper scoping
- **Records** - Record literals, field access, subsumption
- **Type System** - Hindley-Milner inference with type classes (Num, Eq, Ord, Show)
- **Pattern Matching** - Constructors, tuples, lists, wildcards, guards
- **Module System** - Cross-module imports, entrypoint execution
- **Effect System** - IO, FS, Clock, Net with capability-based security
  - **Clock Effect**: Monotonic time, sleep, deterministic mode
  - **Net Effect**: HTTP GET/POST with DNS rebinding prevention, IP blocking
- **REPL** - Full type checking, command history, tab completion
- **Lambda Calculus** - First-class functions, closures, currying

### 🚧 Planned Features (v0.4.0+)
- **Pattern Guards** - Boolean conditions in pattern matching
- **Error Propagation** - `?` operator for Result types
- **Typed Quasiquotes** - Safe metaprogramming with compile-time validation
- **CSP Concurrency** - Channel-based communication with session types
- **AI Training Export** - Execution trace collection (v1.0+)

## Documentation Structure

- **[Guides](/docs/guides/getting-started)** - Tutorials and how-to guides
- **[Reference](/docs/reference/language-syntax)** - Language specification and API docs
- **[Examples](https://github.com/sunholo-data/ailang/tree/main/examples)** - Sample AILANG programs

## Contributing

AILANG is open source and welcomes contributions! Visit our [GitHub repository](https://github.com/sunholo-data/ailang) to:

- Report issues
- Submit pull requests
- Join design discussions
- Review the roadmap

## Design Philosophy

AILANG is built on several key principles:

1. **Explicit Over Implicit**: All effects and dependencies are visible in types
2. **Correctness by Construction**: Make invalid states unrepresentable
3. **AI-Friendly**: Every language feature considers AI tooling needs
4. **Deterministic**: Same input always produces same output
5. **Traceable**: Complete execution history for debugging and training

## Resources

- [GitHub Repository](https://github.com/sunholo-data/ailang)
- [Design Documentation](https://github.com/sunholo-data/ailang/tree/main/design_docs)
- [Change Log](https://github.com/sunholo-data/ailang/blob/main/CHANGELOG.md)
- [Development Setup](/docs/guides/development)
- [llms.txt](https://sunholo-data.github.io/ailang/llms.txt) - AI-readable project documentation

---

*AILANG is an experimental language under active development. APIs and syntax may change.*
