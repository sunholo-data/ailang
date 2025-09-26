# AILANG: The AI-First Programming Language

AILANG is a purely functional programming language designed specifically for AI-assisted software development. It provides static typing with algebraic effects, typed quasiquotes for safe string handling, CSP-based concurrency with session types, and automatic generation of training data for AI model improvement.

## Features

- **Algebraic Effects with Row Polymorphism** - Making all effects explicit and composable
- **Typed Quasiquotes** - Eliminating injection vulnerabilities at compile time
- **CSP with Session Types** - Providing race-free concurrency by construction
- **Deterministic Execution** - Ensuring reproducible traces for AI training
- **Property-Based Testing** - Built-in support for correctness verification
- **Capability-Based Security** - Making permissions explicit in types

## Project Structure

```
ailang/
├── cmd/ailang/          # CLI entry point
├── internal/
│   ├── lexer/           # Tokenization
│   ├── parser/          # Recursive descent parser
│   ├── types/           # Type system implementation
│   ├── ast/             # Abstract syntax tree
│   ├── effects/         # Effect system (TODO)
│   ├── eval/            # Interpreter (TODO)
│   └── ...
├── examples/            # Example AILANG programs
├── stdlib/              # Standard library (TODO)
└── tests/              # Test suite
```

## Building

```bash
# Build the interpreter
make build

# Run tests
make test

# Start REPL
make repl

# Run an AILANG file
make run FILE=examples/hello.ail
```

## Quick Start

### Hello World

```ailang
-- hello.ail
module Hello

import std/io (Console)

func main() -> () ! {IO} {
  with Console {
    print("Hello, AILANG!")
  }
}
```

### Pure Functions with Tests

```ailang
pure func factorial(n: int) -> int
  tests [
    (0, 1),
    (5, 120),
    (10, 3628800)
  ]
{
  if n <= 1 then 1 else n * factorial(n - 1)
}
```

## Language Status

AILANG is currently in early development. The following components are implemented:

- ✅ Lexer with Unicode support
- ✅ Basic parser with Pratt parsing
- ✅ AST definitions
- ✅ Type system foundation
- ⚠️ Parser (partial - needs completion)
- ❌ Type inference (Hindley-Milner with effects)
- ❌ Effect system
- ❌ Interpreter
- ❌ Standard library
- ❌ Quasiquote validation
- ❌ CSP/Session types
- ❌ Training data export

## Development

```bash
# Install dependencies
go mod download

# Format code
make fmt

# Run linter
make lint

# Watch mode for development
make watch
```

## Contributing

AILANG is an experimental language exploring how programming languages can be designed specifically for AI-assisted development. Contributions and ideas are welcome!

## License

Apache 2.0 - See LICENSE file for details