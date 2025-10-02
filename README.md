# AILANG: The AI-First Programming Language

![CI](https://github.com/sunholo-data/ailang/workflows/CI/badge.svg)
![Coverage](https://img.shields.io/badge/coverage-24.8%25-orange.svg)
![Go Version](https://img.shields.io/badge/go-%3E%3D1.22-blue.svg)
![License](https://img.shields.io/badge/license-Apache%202.0-green.svg)

AILANG is a purely functional programming language designed specifically for AI-assisted software development. It features static typing with algebraic effects, typed quasiquotes for safe string handling, CSP-based concurrency with session types, and automatic generation of training data for AI model improvement.

## Current Version: v0.1.0 (MVP - Type System Complete)

**🎯 What Works**: Complete Hindley-Milner type inference, type classes (Num, Eq, Ord, Show), lambda calculus, REPL with full type checking, and expression evaluation.

**⚠️ Known Limitation**: Module execution is not yet implemented. Modules type-check correctly but cannot run. Non-module `.ail` files execute successfully. See [LIMITATIONS.md](docs/LIMITATIONS.md) for details.

**📖 Documentation**: [Implementation Status](docs/reference/implementation-status.md) | [What's Coming in v0.2.0](#whats-coming-in-v020)

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

### Hello World (Non-Module Files)

AILANG v0.1.0 executes **non-module** `.ail` files successfully:

```ailang
-- hello.ail
print("Hello, AILANG!")
```

```bash
ailang run examples/hello.ail
# Output: Hello, AILANG!
```

More examples:
```bash
ailang run examples/arithmetic.ail        # Arithmetic with operator precedence
ailang run examples/simple.ail            # Let bindings and evaluation
ailang run examples/showcase/01_type_inference.ail  # Type inference demo
```

**Note**: Files with `module` declarations type-check but cannot execute until v0.2.0. See [examples/STATUS.md](examples/STATUS.md) for complete example inventory.

### Interactive REPL (Fully Functional)

The REPL is the **most complete** part of AILANG v0.1.0, featuring full type inference and type classes:

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

λ> let compose = \f. \g. \x. f(g(x)) in compose (\x. x * 2) (\x. x + 1) 5
12 :: Int

λ> :quit
```

**REPL Commands**: `:help`, `:type <expr>`, `:instances`, `:import <module>`, `:history`, `:clear`

See [REPL Commands](docs/reference/repl-commands.md) for full reference.

## What Works in v0.1.0

### ✅ Complete Type System

- **Hindley-Milner Type Inference** - Full polymorphic type inference with let-polymorphism
- **Type Classes** - `Num`, `Eq`, `Ord`, `Show` with dictionary-passing semantics
- **Constraint Solving** - Type class constraint generation and resolution
- **Defaulting** - Automatic defaulting for ambiguous numeric types (Int, Float)
- **Type Checking** - Module interface checking, export resolution, import validation

### ✅ Lambda Calculus & Expressions

- **Lambda Expressions** - First-class functions with closures and currying
- **Function Composition** - Higher-order functions, partial application
- **Let Bindings** - Polymorphic let expressions (up to 3 nested levels)
- **Conditionals** - `if-then-else` expressions
- **Operators** - Arithmetic (`+`, `-`, `*`, `/`), comparison (`==`, `<`, `>`, etc.), string concatenation (`++`)

### ✅ Data Structures

- **Lists** - `[1, 2, 3]` with type inference
- **Records** - `{name: "Alice", age: 30}` with field access
- **Tuples** - `(1, "hello", true)` for heterogeneous data
- **Strings** - String literals with concatenation

### ✅ Module System (Type-Checking Only)

- **Module Declarations** - `module path/to/module`
- **Import/Export** - `import stdlib/std/io (println)`, `export func main() ...`
- **Path Resolution** - Correct module path resolution and validation
- **Dependency Analysis** - Import graph construction, cycle detection
- **Interface Generation** - Module signatures with exported types/functions

**Note**: Modules parse and type-check correctly but cannot execute until v0.2.0. See [LIMITATIONS.md](docs/LIMITATIONS.md#critical-limitation-module-execution-gap).

### ✅ Interactive Development

- **Professional REPL** - Arrow key history, tab completion, persistent history (`~/.ailang_history`)
- **Type Inspection** - `:type <expr>` shows qualified types with constraints
- **Instance Inspection** - `:instances` lists available type class instances
- **Debugging Tools** - `:dump-core`, `:dump-typed`, `:trace-defaulting`, `:dry-link`
- **Auto-imports** - `stdlib/std/prelude` loaded automatically

### ✅ Error Reporting

- **Structured Errors** - JSON error output with schema versioning
- **Deterministic Diagnostics** - Stable error messages, line/column positions
- **Helpful Messages** - Type errors, parse errors, module loading errors

## What's Coming in v0.2.0

### 🚀 v0.2.0 Roadmap (Module Execution & Effects)

**M-R1: Module Execution Runtime** (~1,200 LOC, 1.5-2 weeks)
- Module instance creation and initialization
- Import resolution and linking at runtime
- Top-level function execution
- Exported function calls

**M-R2: Algebraic Effects Foundation** (~800 LOC, 1-1.5 weeks)
- Effect declarations and checking
- Effect handler syntax (`with`, `handle`)
- Capability-based effect system
- Basic effects: `IO`, `FS`, `Net`

**M-R3: Pattern Matching** (~600 LOC, 1 week)
- `match` expressions
- Pattern guards
- Exhaustiveness checking
- Constructor patterns for ADTs

**Total Timeline**: 3.5-4.5 weeks for v0.2.0

See [v0.2.0 Roadmap](design_docs/planned/v0_2_0_module_execution.md) for details.

### 📋 Future Features (v0.3.0+)

- Typed quasiquotes (SQL, HTML, JSON, regex)
- CSP-based concurrency with channels
- Session types for protocol verification
- Property-based testing (`properties [...]`)
- AI training data export

<!-- EXAMPLES_STATUS_START -->
## Status

![Examples](https://img.shields.io/badge/examples-16%20passing%2025%20failing-red.svg)

### Example Verification Status

*Last updated: 2025-10-02 14:14:05 UTC*

**Summary:** 16 passed, 25 failed, 6 skipped (Total: 47)

| Example File | Status | Notes |
|--------------|--------|-------|
| `adt_option.ail` | ✅ Pass |  |
| `adt_simple.ail` | ✅ Pass |  |
| `arithmetic.ail` | ✅ Pass |  |
| `block_demo.ail` | ⏭️ Skip | Test/demo file |
| `demos/adt_pipeline.ail` | ❌ Fail | Error: entrypoint 'main' not found in module |
| `demos/effects_pure.ail` | ❌ Fail | Error: entrypoint 'main' not found in module |
| `demos/hello_io.ail` | ✅ Pass |  |
| `effects_basic.ail` | ❌ Fail | Error: module loading error: failed to load exa... |
| `effects_pure.ail` | ✅ Pass |  |
| `experimental/ai_agent_integration.ail` | ❌ Fail | Error: module loading error: failed to load exa... |
| `experimental/concurrent_pipeline.ail` | ❌ Fail | Error: module loading error: failed to load exa... |
| `experimental/factorial.ail` | ❌ Fail | Error: module loading error: failed to load exa... |
| `experimental/quicksort.ail` | ❌ Fail | Error: module loading error: failed to load exa... |
| `experimental/web_api.ail` | ❌ Fail | Error: module loading error: failed to load exa... |
| `hello.ail` | ✅ Pass |  |
| `lambda_expressions.ail` | ❌ Fail | Error: type error in examples/lambda_expression... |
| `list_patterns.ail` | ❌ Fail | Error: module loading error: failed to load exa... |
| `option_demo.ail` | ⏭️ Skip | Test/demo file |
| `patterns.ail` | ❌ Fail | Error: module loading error: failed to load exa... |
| `records.ail` | ❌ Fail | Error: type error in examples/records (decl 3):... |
| `showcase/01_type_inference.ail` | ✅ Pass |  |
| `showcase/02_lambdas.ail` | ✅ Pass |  |
| `showcase/03_lists.ail` | ❌ Fail | Error: evaluation error: builtin concat_String ... |
| `showcase/03_type_classes.ail` | ✅ Pass |  |
| `showcase/04_closures.ail` | ✅ Pass |  |
| `simple.ail` | ✅ Pass |  |
| `stdlib_demo.ail` | ⏭️ Skip | Test/demo file |
| `stdlib_demo_simple.ail` | ⏭️ Skip | Test/demo file |
| `test_import_ctor.ail` | ❌ Fail | Error: entrypoint 'main' not found in module |
| `test_import_func.ail` | ❌ Fail | Error: entrypoint 'main' not found in module |
| `test_use_constructor.ail` | ❌ Fail | Error: entrypoint 'main' not found in module |
| `type_classes_working_reference.ail` | ✅ Pass |  |
| `typeclasses.ail` | ❌ Fail | Error: type error in examples/typeclasses (decl... |
| `v3_3/hello.ail` | ❌ Fail | Error: MOD010: module declaration 'hello' doesn... |
| `v3_3/import_conflict.ail` | ✅ Pass |  |
| `v3_3/imports.ail` | ✅ Pass |  |
| `v3_3/imports_basic.ail` | ✅ Pass |  |
| `v3_3/math.ail` | ❌ Fail | Error: module loading error: failed to load exa... |
| `v3_3/math/div.ail` | ❌ Fail | Error: entrypoint 'main' not found in module |
| `v3_3/math/gcd.ail` | ❌ Fail | Error: entrypoint 'main' not found in module |
| `v3_3/math/simple_gcd.ail` | ❌ Fail | Error: entrypoint 'main' not found in module |
| `v3_3/poly_id.ail` | ❌ Fail | Error: entrypoint 'main' not found in module |
| `v3_3/poly_imports.ail` | ✅ Pass |  |
| `v3_3/poly_use.ail` | ❌ Fail | Error: type error in examples/v3_3/poly_use (de... |
| `v3_3/polymorphic.ail` | ❌ Fail | Error: module loading error: failed to load exa... |
| `v3_3/properties_demo.ail` | ⏭️ Skip | Test/demo file |
| `v3_3/stdlib_demo.ail` | ⏭️ Skip | Test/demo file |

<!-- EXAMPLES_STATUS_END -->

## Documentation

### User Documentation
- **[LIMITATIONS.md](docs/LIMITATIONS.md)** - ⚠️ Read this first! Current v0.1.0 limitations and workarounds
- **[Getting Started](docs/guides/getting-started.md)** - Installation and quick tutorial
- **[REPL Commands](docs/reference/repl-commands.md)** - Interactive REPL guide (fully functional)
- **[Language Syntax](docs/reference/language-syntax.md)** - Complete language reference
- **[Examples Status](examples/STATUS.md)** - Inventory of all 42 example files
- **[Examples README](examples/README.md)** - How to use and understand examples

### Development Documentation
- **[Implementation Status](docs/reference/implementation-status.md)** - Detailed component status with metrics
- **[Development Guide](docs/guides/development.md)** - Contributing and development workflow
- **[CLAUDE.md](CLAUDE.md)** - Instructions for AI assistants working on AILANG
- **[Changelog](CHANGELOG.md)** - Version history and release notes

### Design & Architecture
- **[AI-First Features](docs/ai-first-features.md)** - Why AILANG is designed for AI collaboration
- **[Design Documents](design_docs/)** - Architecture and design decisions
- **[v0.1.0 MVP Roadmap](design_docs/20250929/v0_1_0_mvp_roadmap.md)** - Current milestone plan
- **[Showcase Issues](docs/SHOWCASE_ISSUES.md)** - Known parser/execution limitations discovered during example creation

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
```

See the [Development Guide](docs/guides/development.md) for detailed instructions.

## Project Structure

```
ailang/
├── cmd/ailang/       # CLI entry point
├── internal/         # Core implementation
│   ├── repl/         # Interactive REPL
│   ├── lexer/        # Tokenizer
│   ├── parser/       # Parser
│   ├── types/        # Type system
│   ├── eval/         # Evaluator
│   └── ...           # Other components
├── examples/         # Example programs
├── docs/             # Documentation
├── design_docs/      # Design documents
└── scripts/          # CI/CD scripts
```

## Contributing

AILANG is an experimental language in active development. Contributions are welcome! Please see the [Development Guide](docs/guides/development.md) for guidelines.

## License

Apache 2.0 - See [LICENSE](LICENSE) for details.

## Acknowledgments

AILANG draws inspiration from:
- Haskell (type system, purity)
- OCaml (module system, effects)
- Rust (capability-based security)
- Erlang/Go (CSP concurrency)

---

## FAQ

**Q: Can I use AILANG for production code?**
A: Not yet. v0.1.0 is an MVP focused on the type system. Module execution arrives in v0.2.0 (planned ~4 weeks).

**Q: What's the difference between REPL and file execution?**
A: The REPL is fully functional with type classes. File execution works for non-module files. Module files type-check but cannot execute until v0.2.0.

**Q: Which examples actually work?**
A: See [examples/STATUS.md](examples/STATUS.md). 12 examples execute successfully. The showcase examples in `examples/showcase/` demonstrate working features.

**Q: How can I help or contribute?**
A: See [Development Guide](docs/guides/development.md). We especially welcome testing, example creation, and documentation improvements.

**Q: What makes AILANG "AI-first"?**
A: Explicit effects, deterministic execution traces, structured errors, and typed metaprogramming make AILANG ideal for AI-assisted development and training data generation. See [AI-First Features](docs/ai-first-features.md).

---

*For AI agents: This is an experimental functional language with complete Hindley-Milner type inference, type classes (Num/Eq/Ord/Show), and lambda calculus. The REPL is fully functional. Non-module file execution works for expressions and let bindings. Module files type-check but cannot execute (runtime coming in v0.2.0). See [LIMITATIONS.md](docs/LIMITATIONS.md) and [Implementation Status](docs/reference/implementation-status.md) for exact capabilities.*