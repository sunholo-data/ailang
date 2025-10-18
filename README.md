# 🧠 AILANG: The Deterministic Language for AI Coders

![CI](https://github.com/sunholo-data/ailang/workflows/CI/badge.svg)
![Coverage](https://img.shields.io/badge/coverage-32.6%25-orange.svg)
![Go Version](https://img.shields.io/badge/go-%3E%3D1.22-blue.svg)
![License](https://img.shields.io/badge/license-Apache%202.0-green.svg)

AILANG is a purely functional, effect-typed language designed for **autonomous code synthesis and reasoning**. Unlike human-oriented languages built around IDEs, concurrency, and sugar, AILANG's design goal is **machine decidability, semantic transparency, and compositional determinism**.

---

## 🧩 Core Philosophy

**For humans, a language is a tool for expression.**
**For AIs, it's a substrate for reasoning.**

AILANG minimizes ambiguity and maximizes predictability. Every construct — type, effect, or expression — has **deterministic semantics** that can be reflected, verified, and serialized.

---

## 🏗️ Architecture Overview

| Layer | Description | Status |
|-------|-------------|--------|
| **1. Core Semantics** | Pure functional core with algebraic data types (ADTs), first-class effects, and Hindley-Milner type inference. | ✅ Stable |
| **2. Type System** | Polymorphic effects (`! {IO, ε}`), `Result` and `Option` types, and fully deterministic unification (TApp-aware). | ✅ Stable |
| **3. Reflection & Meta-Programming** | Typed quasiquotes and semantic reflection (`reflect(typeOf(f))`) for deterministic code generation. | 🔜 v0.4.x |
| **4. Deterministic Tooling** | Canonical `normalize`, `suggest`, and `apply` commands; JSON schema output; `--emit-trace jsonl` for training data. | 🔜 v0.3.15 |
| **5. Schema & Hashing Layer** | Machine-readable type/effect registry and versioned semantic hashes for reproducible builds. | 🔜 v0.4.x |
| **6. Runtime & Effects** | Deterministic evaluator with explicit effect rows; supports IO, FS, Net, Clock; no hidden state or global scheduler. | ✅ Stable |
| **7. Cognitive Interfaces** | JSONL trace export for AI self-training; deterministic edit plans for autonomous refactoring. | 🔜 v0.4.x |
| **8. Future Extensions** | Capability budgets (`! {IO @limit=2}`), semantic DAG scheduler (`schedule { a >> b \| c }`). | 🔮 v0.5.x+ |

---

## ❌ Removed / Deprecated Human-Oriented Features

| Removed Feature | Reason for Removal |
|----------------|-------------------|
| **CSP Concurrency / Session Types** | Replaced by static effect-typed task graphs; no runtime scheduler needed. |
| **Unstructured Macros** | Replaced by typed quasiquotes (deterministic AST templates). |
| **Type Classes** | Replaced by structural reflection and record-based traits; removes implicit resolution. |
| **LSP Server** | Superseded by deterministic JSON-RPC API (`ailangd`) exposing parser/typechecker directly. |
| **IDE-centric DX Features** | AIs interact via CLI / API; autocompletion and hover text are unnecessary. |

---

## 🔮 AI-Native Roadmap

| Milestone | Goal | Example Deliverable |
|-----------|------|-------------------|
| **v0.3.15 – Deterministic Tooling** | Canonical normalization, symbol import suggestion, JSON trace export | `ailang suggest-imports file.ail` |
| **v0.4.0 – Meta & Reflection Layer** | Typed quasiquotes + reflection API | `quote (x) -> x + 1 : (int)->int` |
| **v0.4.2 – Schema Registry** | Machine-readable type/effect schemas for deterministic builds | `/schemas/std/io.json` |
| **v0.5.x – Unified Registry Runtime** | Remove legacy builtin registry; single spec source | `RegisterBuiltin(spec)` unified |
| **v0.6.x – Capability Budgets & DAG Scheduler** | Deterministic parallelism via static scheduling | `schedule { parse >> decode \| validate }` |
| **v1.0 – Cognitive Autonomy** | Full round-trip reasoning: AI reads, edits, compiles, evaluates, and self-trains from traces | `--emit-trace jsonl` → fine-tuned validator |

---

## 🧪 Current Milestone: v0.3.14 (JSON Decode)

- ✅ Added `std/json.decode : string -> Result[Json, string]` with streaming parser
- ✅ Fixed list/record pattern matching at runtime
- ✅ Unified primitive type casing (`string`, `int`, `float`, `bool`)
- ✅ DX overhaul: operators (`==`, `!=`, `<`, `>=`) now work naturally
- ✅ All **2,847 tests passing**; 100% coverage on new builtin
- 🔜 **Next**: deterministic tooling (`normalize`, `suggest`, `apply`) in v0.3.15

### Major Milestones

- **v0.3.14 (Oct 2025)**: JSON Decode Release - JSON parsing + pattern matching fixes
- **v0.3.12 (Oct 2025)**: Recovery Release - `show()` builtin restored (recovers 51% of benchmarks)
- **v0.3.11 (Oct 2025)**: Critical row unification fix
- **v0.3.10 (Oct 2025)**: M-DX1 Developer Experience - Builtin system migration (-67% dev time)
- **v0.3.9 (Oct 2025)**: AI API Integration - HTTP headers, JSON encoding, OpenAI example
- **v0.3.6 (Oct 2025)**: AI usability - auto-import, record updates, error detection
- **v0.3.5 (Oct 2025)**: Anonymous functions, `letrec`, numeric conversions

For detailed version history, see [CHANGELOG.md](CHANGELOG.md).

---

## 💡 Why AILANG Works Better for AIs

| Human Need | Human Feature | AI Equivalent in AILANG |
|-----------|---------------|------------------------|
| IDE assistance | LSP / autocompletion | Deterministic type/query API |
| Asynchronous code | Threads / goroutines | Static task DAGs with effects |
| Code reuse | Inheritance / traits | Structural reflection & records |
| Debugging | Interactive debugger | Replayable evaluation trace |
| Logging | `print` / `console` | `--emit-trace jsonl` structured logs |
| Macros | text substitution | Typed quasiquotes (semantic macros) |

---

## Quick Start

### Installation

```bash
# From source
git clone https://github.com/sunholo/ailang.git
cd ailang
make install

# Verify installation
ailang --version
```

For detailed installation instructions, see the [Getting Started Guide](docs/guides/getting-started.md).

### Hello World (Module Execution)

```ailang
-- examples/demos/hello_io.ail
module examples/demos/hello_io

import std/io (println)

export func main() -> () ! {IO} {
  println("Hello from AILANG v0.3.14!")
}
```

```bash
ailang run --caps IO examples/demos/hello_io.ail
# Output: Hello from AILANG v0.3.14!
```

**Important**: Flags must come BEFORE the filename:
```bash
# ✅ CORRECT:
ailang run --caps IO --entry main file.ail

# ❌ WRONG:
ailang run file.ail --caps IO --entry main
```

### Interactive REPL

The REPL features full type inference and deterministic evaluation:

```bash
ailang repl

λ> 1 + 2
3 :: Int

λ> "Hello " ++ "World"
Hello World :: String

λ> let double = \x. x * 2 in double(21)
42 :: Int

λ> :type \x. x + x
\x. x + x :: ∀α. Num α ⇒ α → α

λ> :quit
```

**REPL Commands**: `:help`, `:type <expr>`, `:instances`, `:import <module>`, `:history`, `:clear`

See [REPL Commands](docs/reference/repl-commands.md) for full reference.

---

## What AILANG Can Do (Implementation Status)

### ✅ Core Language

- **Pure functional programming** - Lambda calculus, closures, recursion
- **Hindley-Milner type inference** - Row polymorphism, let-polymorphism
- **Built-in type class instances** - `Num`, `Eq`, `Ord`, `Show` (structural reflection planned for v0.4.0)
- **Algebraic effects** - Capability-based security (IO, FS, Clock, Net)
- **Pattern matching** - ADTs with exhaustiveness checking
- **Module system** - Runtime execution, cross-module imports
- **Block expressions** - `{ e1; e2; e3 }` for sequencing
- **JSON support** - Parsing (`std/json.decode`), encoding (`std/json.encode`)

### ✅ Development Tools

- **M-EVAL** - AI code generation benchmarks (multi-model support)
- **M-EVAL-LOOP v2.0** - Native Go eval tools with 90%+ test coverage
- **Structured error reporting** - JSON schemas for deterministic diagnostics
- **Effect system runtime** - Hermetic testing with `MockEffContext`

### 🔜 Deterministic Tooling (v0.3.15)

- **`ailang normalize`** - Canonical code formatting
- **`ailang suggest-imports`** - Automatic import resolution
- **`ailang apply`** - Deterministic code edits from JSON plans
- **`--emit-trace jsonl`** - Structured execution traces for training

### 🔮 Future (v0.4.0+)

- **Typed quasiquotes** - Deterministic AST templates
- **Structural reflection** - Replace hardcoded type classes
- **Schema registry** - Machine-readable type/effect definitions
- **Capability budgets** - Resource-bounded effects

---

## 📊 Test Coverage

**Examples**: 48/66 passing (72.7%)

All record subsumption, effect system (IO, FS, Clock, Net), type class, ADT, recursion, and block expression examples working.

See [examples/STATUS.md](examples/STATUS.md) for detailed status.

<!-- EXAMPLES_STATUS_START -->
## Status

![Examples](https://img.shields.io/badge/examples-52%20passing%2032%20failing-red.svg)

### Example Verification Status

*Last updated: 2025-10-18 21:01:33 UTC*

**Summary:** 52 passed, 32 failed, 4 skipped (Total: 88)

| Example File | Status | Notes |
|--------------|--------|-------|
| `adt_option.ail` | ✅ Pass |  |
| `adt_simple.ail` | ✅ Pass |  |
| `ai_call.ail` | ❌ Fail | Warning: import path 'stdlib/std/*' is deprecat... |
| `arithmetic.ail` | ❌ Fail | Error: type error in examples/arithmetic (decl ... |
| `block_demo.ail` | ⏭️ Skip | Test/demo file |
| `block_recursion.ail` | ✅ Pass |  |
| `bug_float_comparison.ail` | ✅ Pass |  |
| `bug_modulo_operator.ail` | ✅ Pass |  |
| `claude_haiku_call.ail` | ❌ Fail | Warning: import path 'stdlib/std/*' is deprecat... |
| `demo_ai_api.ail` | ❌ Fail | Error: type error in examples/demo_ai_api (decl... |
| `demo_openai_api.ail` | ❌ Fail | Error: module loading error: failed to load exa... |
| `demos/adt_pipeline.ail` | ✅ Pass |  |
| `demos/effects_pure.ail` | ❌ Fail | Warning: import path 'stdlib/std/*' is deprecat... |
| `demos/hello_io.ail` | ✅ Pass |  |
| `effects_basic.ail` | ✅ Pass |  |
| `effects_pure.ail` | ✅ Pass |  |
| `experimental/ai_agent_integration.ail` | ❌ Fail | Error: module loading error: failed to load exa... |
| `experimental/concurrent_pipeline.ail` | ❌ Fail | Error: module loading error: failed to load exa... |
| `experimental/factorial.ail` | ❌ Fail | Error: module loading error: failed to load exa... |
| `experimental/quicksort.ail` | ❌ Fail | Error: module loading error: failed to load exa... |
| `experimental/web_api.ail` | ❌ Fail | Error: module loading error: failed to load exa... |
| `func_expressions.ail` | ❌ Fail | Error: module loading error: failed to load exa... |
| `guards_basic.ail` | ✅ Pass |  |
| `hello.ail` | ❌ Fail | Error: type error in examples/hello (decl 0): u... |
| `json_basic_decode.ail` | ✅ Pass |  |
| `lambda_expressions.ail` | ❌ Fail | Error: type error in examples/lambda_expression... |
| `letrec_recursion.ail` | ✅ Pass |  |
| `list_patterns.ail` | ❌ Fail | Error: module loading error: failed to load exa... |
| `micro_block_if.ail` | ✅ Pass |  |
| `micro_block_seq.ail` | ✅ Pass |  |
| `micro_clock_measure.ail` | ❌ Fail | Error: type error in examples/micro_clock_measu... |
| `micro_io_echo.ail` | ✅ Pass |  |
| `micro_net_fetch.ail` | ❌ Fail | Error: type error in examples/micro_net_fetch (... |
| `micro_option_map.ail` | ✅ Pass |  |
| `micro_record_person.ail` | ✅ Pass |  |
| `numeric_conversion.ail` | ❌ Fail | Error: type error in examples/numeric_conversio... |
| `option_demo.ail` | ⏭️ Skip | Test/demo file |
| `patterns.ail` | ✅ Pass |  |
| `records.ail` | ❌ Fail | Error: type error in examples/records (decl 3):... |
| `recursion_error.ail` | ✅ Pass |  |
| `recursion_factorial.ail` | ✅ Pass |  |
| `recursion_fibonacci.ail` | ✅ Pass |  |
| `recursion_mutual.ail` | ✅ Pass |  |
| `recursion_quicksort.ail` | ✅ Pass |  |
| `showcase/01_type_inference.ail` | ❌ Fail | Error: type error in examples/showcase/01_type_... |
| `showcase/02_lambdas.ail` | ❌ Fail | Error: type error in examples/showcase/02_lambd... |
| `showcase/03_lists.ail` | ❌ Fail | Error: type error in examples/showcase/03_lists... |
| `showcase/03_type_classes.ail` | ❌ Fail | Error: type error in examples/showcase/03_type_... |
| `showcase/04_closures.ail` | ❌ Fail | Error: type error in examples/showcase/04_closu... |
| `simple.ail` | ✅ Pass |  |
| `stdlib_demo.ail` | ⏭️ Skip | Test/demo file |
| `stdlib_demo_simple.ail` | ⏭️ Skip | Test/demo file |
| `test_effect_annotation.ail` | ✅ Pass |  |
| `test_effect_capability.ail` | ✅ Pass |  |
| `test_effect_fs.ail` | ✅ Pass |  |
| `test_effect_io.ail` | ✅ Pass |  |
| `test_effect_io_simple.ail` | ❌ Fail | Error: evaluation error: _io_println: no effect... |
| `test_exhaustive_bool_complete.ail` | ✅ Pass |  |
| `test_exhaustive_bool_incomplete.ail` | ✅ Pass |  |
| `test_exhaustive_wildcard.ail` | ✅ Pass |  |
| `test_fizzbuzz.ail` | ✅ Pass |  |
| `test_float_comparison.ail` | ✅ Pass |  |
| `test_float_eq_works.ail` | ✅ Pass |  |
| `test_float_modulo.ail` | ✅ Pass |  |
| `test_guard_bool.ail` | ✅ Pass |  |
| `test_guard_debug.ail` | ✅ Pass |  |
| `test_guard_false.ail` | ✅ Pass |  |
| `test_import_ctor.ail` | ✅ Pass |  |
| `test_import_func.ail` | ✅ Pass |  |
| `test_integral.ail` | ✅ Pass |  |
| `test_invocation.ail` | ✅ Pass |  |
| `test_io_builtins.ail` | ✅ Pass |  |
| `test_m_r7_comprehensive.ail` | ❌ Fail | Error: module loading error: failed to load exa... |
| `test_module_minimal.ail` | ✅ Pass |  |
| `test_modulo_works.ail` | ✅ Pass |  |
| `test_net_file_protocol.ail` | ❌ Fail | Error: type error in examples/test_net_file_pro... |
| `test_net_localhost.ail` | ❌ Fail | Error: type error in examples/test_net_localhos... |
| `test_net_security.ail` | ❌ Fail | Error: type error in examples/test_net_security... |
| `test_no_import.ail` | ✅ Pass |  |
| `test_record_subsumption.ail` | ✅ Pass |  |
| `test_single_guard.ail` | ✅ Pass |  |
| `test_use_constructor.ail` | ✅ Pass |  |
| `test_with_import.ail` | ✅ Pass |  |
| `type_classes_working_reference.ail` | ❌ Fail | Error: type error in examples/type_classes_work... |
| `typeclasses.ail` | ❌ Fail | Error: type error in examples/typeclasses (decl... |
| `v3_3/imports.ail` | ✅ Pass |  |
| `v3_3/imports_basic.ail` | ✅ Pass |  |
| `v3_3/math/gcd.ail` | ❌ Fail | Error: entrypoint 'main' not found in module |

<!-- EXAMPLES_STATUS_END -->

---

## Documentation

📖 **[Complete Documentation](https://sunholo-data.github.io/ailang/)** - Visit our full documentation site

**Quick Links:**
- **[Getting Started](https://sunholo-data.github.io/ailang/docs/guides/getting-started)** - Installation and tutorial
- **[Language Guide](https://sunholo-data.github.io/ailang/docs/category/language-guide)** - Syntax and features
- **[REPL Guide](https://sunholo-data.github.io/ailang/docs/guides/repl)** - Interactive development
- **[Benchmarks](https://sunholo-data.github.io/ailang/docs/benchmarks/performance)** - AI code generation performance
- **[Examples](https://sunholo-data.github.io/ailang/docs/examples/overview)** - Code examples and patterns

---

## Development

```bash
# Build and install
make install

# Run tests
make test

# Start REPL
make repl

# Run example
make run FILE=examples/hello.ail

# Auto-rebuild on changes
make watch-install

# Check coverage
make test-coverage-badge

# AI Evaluation & Design Generation
make eval-suite          # Run AI benchmarks (AILANG vs Python)
make eval-report         # Generate evaluation report
make eval-analyze        # Analyze failures, generate design docs
```

See the [Development Guide](https://sunholo-data.github.io/ailang/docs/guides/development) for detailed instructions.

---

## 📚 Specification Reference

- **Core**: `/internal/types/`, `/internal/eval/`
- **Effects**: `/internal/effects/`
- **Builtins**: `/internal/builtins/spec.go`
- **Standard Library**: `/stdlib/std/*`
- **Design Docs**: `/design_docs/`

---

## Project Structure

```
ailang/
├── cmd/ailang/         # CLI entry point
├── internal/           # Core implementation
│   ├── repl/           # Interactive REPL
│   ├── lexer/          # Tokenizer
│   ├── parser/         # Parser
│   ├── types/          # Type system
│   ├── eval/           # Evaluator
│   ├── effects/        # Effect system runtime
│   ├── builtins/       # Builtin registry
│   └── eval_harness/   # AI evaluation framework
├── stdlib/             # Standard library
├── examples/           # Example programs
├── docs/               # Documentation
└── design_docs/        # Design documents
```

---

## Contributing

AILANG is an experimental language in active development. Contributions are welcome! Please see the [Development Guide](https://sunholo-data.github.io/ailang/docs/guides/development) for guidelines.

---

## ⚖️ License & Philosophy

AILANG is **open infrastructure for Cognitive DevOps** — systems that write, test, and deploy themselves deterministically.

**Our design north star: build languages AIs enjoy using.**

Apache 2.0 - See [LICENSE](LICENSE) for details.

---

## Acknowledgments

AILANG draws inspiration from:
- **Haskell** (type system, purity)
- **OCaml** (module system, effects)
- **Rust** (capability-based security)
- **Idris/Agda** (reflection and metaprogramming)

---

*For AI agents: This is a deterministic functional language with Hindley-Milner type inference, algebraic effects, and explicit effect tracking. The REPL is fully functional. Module execution works with capability-based security. See [CLAUDE.md](CLAUDE.md) and [Complete Documentation](https://sunholo-data.github.io/ailang/) for exact capabilities.*
