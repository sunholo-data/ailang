# AILANG: The AI-First Programming Language

AILANG is a purely functional programming language designed specifically for AI-assisted software development. It provides static typing with algebraic effects, typed quasiquotes for safe string handling, CSP-based concurrency with session types, and automatic generation of training data for AI model improvement.

## 🎉 Major Milestone: Type Inference Complete!

**The Hindley-Milner type inference engine with row polymorphism is now fully implemented!** This includes:
- ✅ Complete HM type inference with let-polymorphism
- ✅ Principal row unification for records and effects
- ✅ Kind system preventing type/row confusion  
- ✅ ~2,500 lines of production-ready type system code
- ✅ All core algorithms tested and passing

While not yet integrated with the evaluator, the type system provides the semantic foundation for AILANG v2.0.

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
│   ├── ast/             # Abstract syntax tree definitions ✅ (complete)
│   ├── lexer/           # Tokenizer with full Unicode support ✅ (fully working)
│   ├── parser/          # Recursive descent parser ✅ (1,059 lines, mostly complete)
│   ├── eval/            # Tree-walking interpreter ✅ (~600 lines, core features working)
│   │   ├── eval_simple.go     # Main evaluator with show/toText functions
│   │   ├── eval_simple_test.go # Comprehensive test suite
│   │   ├── value.go           # Value type definitions
│   │   └── environment.go     # Variable scoping
│   ├── types/           # Type system with HM inference ✅ (~2,500 lines, fully working!)
│   │   ├── types.go           # Core type definitions
│   │   ├── types_v2.go        # Enhanced types with kinds
│   │   ├── kinds.go           # Kind system (Effect, Record, Row)
│   │   ├── inference.go       # HM type inference engine
│   │   ├── unification.go     # Type unification with occurs check
│   │   ├── row_unification.go # Principal row unifier
│   │   ├── env.go             # Type environments
│   │   ├── errors.go          # Rich error reporting
│   │   └── typechecker.go     # Main type checking interface
│   ├── effects/         # Effect system (TODO)
│   ├── channels/        # CSP implementation (TODO)
│   ├── session/         # Session types (TODO)
│   └── typeclass/       # Type classes (TODO)
├── examples/            # Example AILANG programs
│   ├── hello.ail        # Hello world example (requires IO effects)
│   ├── arithmetic.ail   # Basic arithmetic ✅ (working with show and ++)
│   ├── factorial.ail    # Factorial implementations (advanced syntax WIP)
│   ├── simple.ail       # Simple test program ✅ (working)
│   └── show_demo.ail    # Demonstrates show/toText functions ✅ (working)
├── quasiquote/          # Typed templates (TODO)
├── stdlib/              # Standard library (TODO)
├── tools/               # Development tools (TODO)
└── design_docs/         # Language design documentation
```

## Getting Started

### Prerequisites

- Go 1.21 or later
- Make (optional but recommended)
- fswatch (optional, for auto-rebuild)

### Installation

#### Quick Install (Recommended)

```bash
# Clone the repository
git clone https://github.com/sunholo/ailang.git
cd ailang

# Install ailang globally (makes 'ailang' command available everywhere)
make install

# Add Go bin to PATH if not already configured
echo 'export PATH="$HOME/go/bin:$PATH"' >> ~/.zshrc  # For zsh (macOS default)
source ~/.zshrc
# OR
echo 'export PATH="$HOME/go/bin:$PATH"' >> ~/.bashrc  # For bash
source ~/.bashrc

# Verify installation
ailang --version
```

#### Local Build

```bash
# Build to local bin/ directory
make build

# Run locally
./bin/ailang --version
```

### Running AILANG

```bash
# Start the REPL
ailang repl

# Run a file
ailang run examples/simple.ail

# Check a file (parsing only)
ailang check examples/hello.ail

# Show version
ailang --version
```

### Development Workflow

```bash
# Auto-rebuild and install on file changes
make watch-install

# Quick reinstall after changes
make quick-install

# Run tests
make test

# Format code
make fmt
```

### Testing

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run specific package tests
go test ./internal/lexer
go test ./internal/parser
go test ./internal/eval
go test ./internal/types  # Type inference tests

# Run with verbose output
go test -v ./...

# Test type inference system
go run cmd/typecheck/main.go        # Interactive demos
go run cmd/typecheck/demo_ast.go    # Type check real files
```

## Quick Start

### Hello World

```ailang
-- hello.ail (simplified version that works today)
print("Hello, AILANG!")
```

### Working with Values

```ailang
-- values.ail
let name = "AILANG" in
let version = 0.1 in
print("Welcome to " ++ name ++ " v" ++ show(version))
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

AILANG is currently in early development with a fully functional type inference engine. Here's the current state:

### ✅ Completed Components

#### **Lexer** (Fully Working)
- Complete tokenization with Unicode support
- All token types: keywords, operators, literals, identifiers
- String escapes, comments, scientific notation
- `++` operator for string concatenation
- ~550 lines, all tests passing

#### **Parser** (Mostly Complete) 
- Recursive descent with Pratt parsing (~1,059 lines)
- ✅ **Working**: Basic expressions, let bindings, if-then-else, lists, records
- ✅ **Working**: Binary/unary operators with correct precedence
- ✅ **Working**: Module declarations and import statements
- ⚠️ **Parsed but not evaluated**: Pattern matching, type annotations
- ❌ **Not working**: Lambda syntax (`\x.` or `=>`), `?` operator, effect handlers

#### **Evaluator** (Core Features Working)
- Tree-walking interpreter (~630 lines)
- ✅ **Working**: Arithmetic, booleans, strings, let bindings, if-then-else
- ✅ **Working**: Lists, records (creation only, not field access)
- ✅ **Working**: `show` and `toText` builtins, `++` operator
- ❌ **Not working**: Lambdas, pattern matching, record field access, tuples

#### **Type System** (Fully Implemented! ~2,500 lines)
- ✅ **Hindley-Milner type inference** with let-polymorphism
- ✅ **Principal row unification** for records and effects  
- ✅ **Kind system** with separate kinds for Effect/Record/Row
- ✅ **Value restriction** for sound polymorphism with effects
- ✅ **Constraint collection** for type classes (Num, Ord, Eq, Show)
- ✅ **Rich error reporting** with paths and suggestions
- ⚠️ **Not integrated**: Type checker works standalone but not connected to evaluator

### Testing Status
- ✅ **Lexer tests**: All passing
- ✅ **Parser tests**: Basic coverage  
- ✅ **Evaluator tests**: Core features tested
- ✅ **Type inference tests**: All algorithms passing
  - Row unification: ✅ PASS
  - Occurs check: ✅ PASS  
  - Kind mismatch detection: ✅ PASS
  - Value restriction: ✅ PASS
  - Error reporting: ✅ PASS

### ❌ Not Yet Implemented
- **Effect System** - Algebraic effects with capabilities
- **Standard Library** - Core modules (io, collections, concurrent)
- **Quasiquotes** - Typed templates for SQL, HTML, regex, etc.
- **CSP Concurrency** - Channels with session types
- **Property Testing** - Built-in property-based testing
- **Training Export** - AI training data generation
- **Module System** - Module loading and resolution

## Current Capabilities

### What Actually Works (Can Run)
- ✅ Integer and float arithmetic: `2 + 3 * 4`, `10.5 / 2.0`
- ✅ Boolean operations: `true && false`, `not true`
- ✅ Comparisons: `5 > 3`, `x == y`, `a != b`
- ✅ Let bindings: `let x = 5 in x * 2`
- ✅ Conditionals: `if x > 0 then "positive" else "negative"`
- ✅ String concatenation: `"hello " ++ "world"`
- ✅ Lists (creation only): `[1, 2, 3]`
- ✅ Records (creation only): `{name: "Alice", age: 30}`
- ✅ Builtins: `show(42)` → `"42"`, `toText(value)`, `print(value)`

### What Parses but Doesn't Evaluate
- ⚠️ Pattern matching: `match x { ... }`
- ⚠️ Type annotations: `let x: int = 5`
- ⚠️ Module imports: `import std/io`
- ⚠️ Function declarations: `func add(x, y) { x + y }`

### What Doesn't Parse Yet
- ❌ Lambda expressions: `\x. x + 1` or `(x) => x + 1`
- ❌ Record field access: `person.name`
- ❌ Tuples: `(1, "hello", true)`
- ❌ Effect handlers: `handle ... with { ... }`
- ❌ Result operator: `readFile(path)?`

### Builtin Functions
- `print(value)` - Outputs value to console
- `show(value)` - Converts any value to its string representation (quoted for strings)
- `toText(value)` - Converts value to string without quotes (for display)

### Operator Precedence (high to low)
1. Unary operators (`not`, `-`)
2. Multiplication/Division (`*`, `/`, `%`)
3. Addition/Subtraction (`+`, `-`)
4. String concatenation (`++`)
5. Comparisons (`<`, `>`, `<=`, `>=`, `==`, `!=`)
6. Logical AND (`&&`)
7. Logical OR (`||`)

### Type Inference Examples (Working!)

The type inference engine is fully functional but not yet integrated with the parser/evaluator. 
Here's what it can infer when given proper AST:

```ailang
-- Simple let binding with arithmetic
let x = 5 in let y = x + 3 in y
-- Inferred: int
-- Effects: {}
-- Constraints: Num[int]

-- Polymorphic identity (when lambdas are supported)
\x. x
-- Inferred: ∀α. α -> α

-- Record field polymorphism (type system ready)
\r. r.name  
-- Inferred: ∀α ρ. {name: α | ρ} -> α

-- Effect tracking (types ready, runtime TODO)
\path. readFile(path)
-- Inferred: string -> Result[string, IOError] ! {FS}
```

Run `go run cmd/typecheck/main.go` to see live type inference demos!

### Example Working Programs

```ailang
-- arithmetic.ail (fully working)
let x = 5 + 3 * 2 in
print("x = " ++ show(x))  -- Prints: x = 11

let y = (10 - 2) / 4 in
print("y = " ++ show(y))  -- Prints: y = 2
```

```ailang
-- show_demo.ail (demonstrates show function)
let record = { name: "Alice", age: 30 } in
print("Record: " ++ show(record))
-- Prints: Record: {age: 30, name: "Alice"}

let text = "hello\nworld" in
print("Quoted: " ++ show(text))   -- Prints: Quoted: "hello\nworld"
print("Unquoted: " ++ toText(text)) -- Prints: Unquoted: hello
                                     --         world
```

### Known Limitations

#### Parser Limitations
- No lambda expression support (`\x.` or `=>` syntax)
- No `?` operator for Result types
- No effect handler syntax
- Record field access parses but causes runtime errors

#### Evaluator Limitations  
- Lambdas don't evaluate
- Pattern matching doesn't execute
- Record field access not implemented
- Tuples not supported
- Module imports not resolved

#### Integration Issues
- Type checker not connected to evaluator
- Type annotations parsed but ignored
- Effect annotations have no runtime support
- No type checking before evaluation

## Development Roadmap

### Phase 1: Core Language (✅ COMPLETE!)
- [x] Lexer implementation
- [x] AST definitions  
- [x] Parser implementation (mostly complete)
- [x] Basic evaluator (core features working)
- [x] **Hindley-Milner type inference with let-polymorphism**
- [x] **Row polymorphism for records and effects**
- [x] **Principal type inference with row unification**
- [x] **Kind system for type safety**
- [x] String operations (`++` operator, `show` function)

### Phase 2: Advanced Features (Current Focus)
- [ ] Pattern matching evaluation  
- [ ] Tuple evaluation support
- [ ] Type class dictionary elaboration
- [ ] Effect system runtime support
- [ ] Integration of type checker with evaluator

### Phase 3: Effects & Concurrency
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

### Available Make Commands

```bash
make build          # Build to bin/
make install        # Install globally to $GOPATH/bin
make quick-install  # Fast reinstall (for development)
make watch          # Auto-rebuild locally on changes
make watch-install  # Auto-install globally on changes
make test           # Run all tests
make test-coverage  # Run tests with coverage report
make fmt            # Format all Go code
make vet            # Run go vet
make lint           # Run golangci-lint
make clean          # Remove build artifacts
make repl           # Start the REPL
make run FILE=...   # Run an AILANG file
make help           # Show all available commands
```

### Keeping `ailang` Updated

After making code changes, update the global `ailang` command:

```bash
# Option 1: Manual update
make quick-install

# Option 2: Auto-update on file changes
make watch-install

# Option 3: Create an alias for quick updates
alias ailang-update='cd /path/to/ailang && make quick-install && cd -'
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