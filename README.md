# AILANG: The AI-First Programming Language

![CI](https://github.com/sunholo-data/ailang/workflows/CI/badge.svg)
![Coverage](https://img.shields.io/badge/coverage-24.5%25-orange.svg)
![Go Version](https://img.shields.io/badge/go-%3E%3D1.22-blue.svg)
![License](https://img.shields.io/badge/license-Apache%202.0-green.svg)

AILANG is a purely functional programming language designed specifically for AI-assisted software development. It features static typing with algebraic effects, typed quasiquotes for safe string handling, CSP-based concurrency with session types, and automatic generation of training data for AI model improvement.

## Current Version: v0.0.7

⚠️ **Early Development**: This language is in active development. Many planned features are not yet implemented. See [Implementation Status](docs/reference/implementation-status.md) for details.

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

### Hello World

```ailang
-- hello.ail
print("Hello, AILANG!")
```

```bash
ailang run hello.ail
```

### Interactive REPL

```bash
ailang repl

λ> 1 + 2
3 :: Int

λ> "Hello " ++ "World"
Hello World :: String

λ> let double = \x. x * 2 in double(21)
42 :: Int

λ> :quit
```

## Core Features

### ✅ Working Features

- **Lambda Expressions** - Full lambda calculus with closures and currying
- **Type Inference** - Hindley-Milner type system with let-polymorphism
- **Type Classes** - Num, Eq, Ord, Show with dictionary-passing (REPL only)
- **Interactive REPL** - Professional REPL with history, completion, and debugging
- **Basic Evaluation** - Arithmetic, strings, conditionals, let bindings
- **Records & Lists** - Creation and field access
- **Module System Foundation** - Path resolution, dependency management, cycle detection
- **Structured Error Reporting** - JSON error output with schema versioning, deterministic diagnostics

### 🚧 In Progress

- Function declarations (`func` syntax)
- Pattern matching
- Module imports in files
- Type definitions
- Effect system

### 📋 Planned

- Typed quasiquotes (SQL, HTML, JSON, etc.)
- CSP-based concurrency with channels
- Session types for protocol verification
- Property-based testing
- Capability-based security
- AI training data export

<!-- EXAMPLES_STATUS_START -->
## Status

![Examples](https://img.shields.io/badge/examples-21%20passing%2023%20failing-red.svg)

### Example Verification Status

*Last updated: 2025-09-30 15:43:04 UTC*

**Summary:** 21 passed, 23 failed, 16 skipped (Total: 60)

| Example File | Status | Notes |
|--------------|--------|-------|
| `ai_agent_integration.ail` | ❌ Fail | Error: module loading error: failed to load exa... |
| `arithmetic.ail` | ✅ Pass |  |
| `concurrent_pipeline.ail` | ❌ Fail | Error: module loading error: failed to load exa... |
| `debug1.ail` | ✅ Pass |  |
| `debug2.ail` | ✅ Pass |  |
| `debug3.ail` | ✅ Pass |  |
| `defaulting_trace.ail` | ⏭️ Skip | Test/demo file |
| `factorial.ail` | ❌ Fail | Error: module loading error: failed to load exa... |
| `hello.ail` | ✅ Pass |  |
| `lambda_expressions.ail` | ❌ Fail | Error: type error in examples/lambda_expression... |
| `lambdas_v2.ail` | ✅ Pass |  |
| `num_demo.ail` | ⏭️ Skip | Test/demo file |
| `phase1_demo.ail` | ⏭️ Skip | Test/demo file |
| `pure_lambdas.ail` | ✅ Pass |  |
| `quicksort.ail` | ❌ Fail | Error: module loading error: failed to load exa... |
| `repl_demo.ail` | ⏭️ Skip | Test/demo file |
| `repl_test.ail` | ⏭️ Skip | Test/demo file |
| `show_demo.ail` | ⏭️ Skip | Test/demo file |
| `simple.ail` | ✅ Pass |  |
| `simple_module.ail` | ✅ Pass |  |
| `test_basic.ail` | ✅ Pass |  |
| `test_export.ail` | ❌ Fail | Error: module loading error: failed to load exa... |
| `test_export_func.ail` | ✅ Pass |  |
| `test_instances.ail` | ✅ Pass |  |
| `test_operators.ail` | ✅ Pass |  |
| `test_simple_module.ail` | ❌ Fail | Error: MOD010: module declaration 'test_simple'... |
| `test_simple_ops.ail` | ✅ Pass |  |
| `test_v2.ail` | ❌ Fail | Error: type error in examples/test_v2 (decl 0):... |
| `type_class_showcase.ail` | ❌ Fail | Error: normalization not implemented for <nil> |
| `type_classes.ail` | ❌ Fail | Error: normalization not implemented for <nil> |
| `type_classes_complete.ail` | ❌ Fail | Error: normalization not implemented for <nil> |
| `type_classes_demo.ail` | ⏭️ Skip | Test/demo file |
| `type_classes_demo_working.ail` | ⏭️ Skip | Test/demo file |
| `type_classes_final.ail` | ❌ Fail | Error: module loading error: failed to load exa... |
| `type_classes_simple.ail` | ❌ Fail | Error: normalization not implemented for <nil> |
| `type_classes_working.ail` | ❌ Fail | Error: normalization not implemented for <nil> |
| `type_demo_minimal.ail` | ⏭️ Skip | Test/demo file |
| `type_inference_basic.ail` | ❌ Fail | Error: type error in examples/type_inference_ba... |
| `type_inference_demo.ail` | ⏭️ Skip | Test/demo file |
| `type_inference_simple.ail` | ❌ Fail | Error: module loading error: failed to load exa... |
| `v0_0_3_features_demo.ail` | ⏭️ Skip | Test/demo file |
| `v2_pipeline_demo.ail` | ⏭️ Skip | Test/demo file |
| `v2_type_inference.ail` | ✅ Pass |  |
| `v3_3/hello.ail` | ❌ Fail | Error: MOD010: module declaration 'hello' doesn... |
| `v3_3/import_conflict.ail` | ✅ Pass |  |
| `v3_3/imports.ail` | ✅ Pass |  |
| `v3_3/imports_basic.ail` | ✅ Pass |  |
| `v3_3/math.ail` | ❌ Fail | Error: module loading error: failed to load exa... |
| `v3_3/math/div.ail` | ✅ Pass |  |
| `v3_3/math/gcd.ail` | ✅ Pass |  |
| `v3_3/math/simple_gcd.ail` | ✅ Pass |  |
| `v3_3/poly_id.ail` | ❌ Fail | Error: module loading error: failed to load exa... |
| `v3_3/poly_imports.ail` | ❌ Fail | Error: module loading error: failed to load exa... |
| `v3_3/poly_use.ail` | ❌ Fail | Error: module loading error: failed to load exa... |
| `v3_3/polymorphic.ail` | ❌ Fail | Error: module loading error: failed to load exa... |
| `v3_3/properties_demo.ail` | ⏭️ Skip | Test/demo file |
| `v3_3/stdlib_demo.ail` | ⏭️ Skip | Test/demo file |
| `web_api.ail` | ❌ Fail | Error: module loading error: failed to load exa... |
| `working_demo.ail` | ⏭️ Skip | Test/demo file |
| `working_v0_0_3_demo.ail` | ⏭️ Skip | Test/demo file |

<!-- EXAMPLES_STATUS_END -->

## Documentation

- **[Getting Started](docs/guides/getting-started.md)** - Installation and quick tutorial
- **[Language Syntax](docs/reference/language-syntax.md)** - Complete language reference
- **[REPL Commands](docs/reference/repl-commands.md)** - Interactive REPL guide
- **[Development Guide](docs/guides/development.md)** - Contributing and development workflow
- **[Implementation Status](docs/reference/implementation-status.md)** - Detailed component status
- **[Changelog](CHANGELOG.md)** - Version history and release notes
- **[Design Documents](design_docs/)** - Architecture and design decisions

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

*For AI agents: This is an experimental functional language with Hindley-Milner type inference, type classes, and planned support for algebraic effects. The REPL is fully functional with type class resolution. File execution supports basic features. See [Implementation Status](docs/reference/implementation-status.md) for exact capabilities.*