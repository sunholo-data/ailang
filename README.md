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
├── cmd/ailang/          # CLI entry point with REPL
├── internal/
│   ├── ast/             # Abstract syntax tree definitions ✅
│   ├── lexer/           # Tokenizer with full Unicode support ✅
│   ├── parser/          # Recursive descent parser (partial)
│   ├── eval/            # Tree-walking interpreter ✅
│   ├── types/           # Type system foundation
│   ├── effects/         # Effect system (TODO)
│   ├── channels/        # CSP implementation (TODO)
│   ├── session/         # Session types (TODO)
│   └── typeclass/       # Type classes (TODO)
├── examples/            # Example AILANG programs
│   ├── hello.ail        # Hello world example
│   ├── arithmetic.ail   # Basic arithmetic
│   ├── factorial.ail    # Factorial implementations
│   └── simple.ail       # Simple test program
├── quasiquote/          # Typed templates (TODO)
├── stdlib/              # Standard library (TODO)
├── tools/               # Development tools (TODO)
└── design_docs/         # Language design documentation
```

## Getting Started

### Installation

```bash
# Clone the repository
git clone https://github.com/sunholo/ailang.git
cd ailang

# Build the interpreter
go build -o ailang ./cmd/ailang

# Or use make
make build
```

### Running AILANG

```bash
# Start the REPL
./ailang repl

# Run a file
./ailang run examples/simple.ail

# Check a file (parsing only)
./ailang check examples/hello.ail

# Show version
./ailang --version
```

### Testing

```bash
# Run all tests
make test

# Run lexer tests specifically
go test ./internal/lexer

# Run with verbose output
go test -v ./...
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

## Implementation Status

AILANG is currently in early development. Here's the current implementation progress:

### ✅ Completed
- **Lexer** - Full tokenization with Unicode support, all tests passing
  - Keywords, operators, literals (int, float, string, bool, unit)
  - Comments, string escapes, scientific notation
- **AST Definitions** - Complete abstract syntax tree structure
- **Basic Evaluator** - Tree-walking interpreter for core features
  - Arithmetic and logical operations
  - Function definitions and calls
  - Let bindings and conditionals
  - Lists and records
  - Built-in print function
- **REPL** - Interactive mode with colored output
- **CLI** - Command-line interface with run, repl, check modes

### ⚠️ Partially Implemented
- **Parser** - Recursive descent with Pratt parsing (needs completion)
  - ✅ Literals and identifiers
  - ❌ Binary/unary operations
  - ❌ Function declarations
  - ❌ Pattern matching
  - ❌ Module system

### ❌ Not Yet Implemented
- **Type System** - Hindley-Milner type inference with row polymorphism
- **Effect System** - Algebraic effects with capabilities
- **Standard Library** - Core modules (io, collections, concurrent)
- **Quasiquotes** - Typed templates for SQL, HTML, regex, etc.
- **CSP Concurrency** - Channels with session types
- **Property Testing** - Built-in property-based testing
- **Training Export** - AI training data generation

## Current Capabilities

The interpreter can currently evaluate:
- Integer and float literals
- String literals
- Boolean values
- Unit type `()`
- Simple expressions (once parser is complete)

Example working code:
```ailang
42           -- Returns: 42
"hello"      -- Returns: "hello"
true         -- Returns: true
()           -- Returns: ()
```

## Development Roadmap

### Phase 1: Core Language (Current)
- [x] Lexer implementation
- [x] AST definitions
- [x] Basic evaluator
- [ ] Complete parser
- [ ] Basic type checking

### Phase 2: Type System
- [ ] Hindley-Milner type inference
- [ ] Row polymorphism
- [ ] Type classes
- [ ] Effect inference

### Phase 3: Advanced Features
- [ ] Algebraic effects
- [ ] Pattern matching
- [ ] Quasiquotes (SQL, HTML, etc.)
- [ ] Module system
- [ ] Standard library

### Phase 4: Concurrency & AI
- [ ] CSP-based concurrency
- [ ] Session types
- [ ] Training data export
- [ ] Property-based testing
- [ ] AI-assisted debugging

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

# Clean build artifacts
make clean
```

## Contributing

AILANG is an experimental language exploring how programming languages can be designed specifically for AI-assisted development. Contributions and ideas are welcome!

### Areas Needing Help
- Parser completion for expressions and statements
- Type inference implementation
- Standard library modules
- Documentation and examples
- Testing and bug fixes

## License

Apache 2.0 - See LICENSE file for details