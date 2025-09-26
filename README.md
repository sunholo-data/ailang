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
│   ├── ast/             # Abstract syntax tree definitions ✅ (complete)
│   ├── lexer/           # Tokenizer with full Unicode support ✅ (fully working)
│   ├── parser/          # Recursive descent parser ✅ (1,059 lines, mostly complete)
│   ├── eval/            # Tree-walking interpreter ✅ (~600 lines, core features working)
│   │   ├── eval_simple.go     # Main evaluator with show/toText functions
│   │   ├── eval_simple_test.go # Comprehensive test suite
│   │   ├── value.go           # Value type definitions
│   │   └── environment.go     # Variable scoping
│   ├── types/           # Type system foundation (489 lines, needs inference)
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

AILANG is currently in early development. Here's the current implementation progress:

### ✅ Completed
- **Lexer** - Full tokenization with Unicode support, all tests passing
  - Keywords, operators, literals (int, float, string, bool, unit)
  - Comments, string escapes, scientific notation
  - `++` operator for string concatenation
  - Keyword recognition working correctly via `LookupIdent()`
- **AST Definitions** - Complete abstract syntax tree structure
- **Parser** - Recursive descent with Pratt parsing (1,059 lines, mostly complete)
  - ✅ Literals, identifiers, and all basic types
  - ✅ Binary and unary operations with proper precedence
  - ✅ Function declarations, lambdas, and function calls
  - ✅ Let expressions with type annotations
  - ✅ If-then-else conditionals
  - ✅ Lists, tuples, records, and field access
  - ✅ Module and import statements
  - ✅ Basic pattern matching structure
  - ✅ `++` operator with correct precedence (between arithmetic and comparisons)
- **Evaluator** - Tree-walking interpreter for core features (~600 lines)
  - ✅ Arithmetic and logical operations
  - ✅ Function definitions and calls
  - ✅ Let bindings with proper scoping
  - ✅ If-then-else conditionals
  - ✅ Lists and records
  - ✅ Lambda expressions
  - ✅ String concatenation via `++` operator
  - ✅ `show` builtin for type conversion to strings
  - ✅ `toText` builtin for unquoted string output
  - ✅ Deterministic output for all value types
  - ❌ Pattern matching evaluation
- **REPL** - Interactive mode with colored output
- **CLI** - Command-line interface with run, repl, check modes
- **Testing** - Comprehensive test suite
  - ✅ Lexer tests (all passing)
  - ✅ Parser tests (basic coverage)
  - ✅ Evaluator tests (including show/++ operators)

### ⚠️ Partially Implemented
- **Type System** - Foundation in place (489 lines)
  - ✅ Type variables, constructors, and function types
  - ✅ Type substitution mechanisms
  - ✅ Effect types foundation
  - ❌ Hindley-Milner type inference
  - ❌ Row polymorphism
- **Parser Advanced Features**
  - ❌ Advanced pattern matching (list/tuple/record patterns)
  - ❌ Type declarations and type classes
  - ❌ Test blocks and property blocks (stubs exist)
  - ❌ Quasiquotes (parsing infrastructure exists)
  - ❌ Advanced effect syntax

### ❌ Not Yet Implemented
- **Effect System** - Algebraic effects with capabilities
- **Standard Library** - Core modules (io, collections, concurrent)
- **Quasiquotes** - Typed templates for SQL, HTML, regex, etc.
- **CSP Concurrency** - Channels with session types
- **Property Testing** - Built-in property-based testing
- **Training Export** - AI training data generation
- **Module System** - Module loading and resolution

## Current Capabilities

The interpreter can currently parse and evaluate:

### Working Features
- Integer and float arithmetic: `2 + 3 * 4`, `10.5 / 2.0`
- Boolean operations: `true && false`, `not true`
- Comparisons: `5 > 3`, `x == y`, `a != b`
- Let bindings: `let x = 5 in x * 2`
- Conditionals: `if x > 0 then "positive" else "negative"`
- Functions: `let f = (x) => x * 2 in f(5)`
- Lists: `[1, 2, 3]`, list operations
- Records: `{ name: "Alice", age: 30 }`, field access with `.`
- Unit type: `()`
- **String concatenation**: `"hello " ++ "world"` (using `++` operator)
- **Type conversion**: `show(42)` returns `"42"` (with proper quoting for strings)
- **Pretty printing**: `toText(value)` for unquoted output

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
- Pattern matching not evaluated (parses but doesn't execute)
- Module imports not resolved
- Type annotations parsed but not checked
- Effect annotations parsed but not enforced
- Tuple expressions not fully supported in evaluator

## Development Roadmap

### Phase 1: Core Language (Current)
- [x] Lexer implementation
- [x] AST definitions  
- [x] Parser implementation (mostly complete)
- [x] Basic evaluator (core features working)
- [x] String operations (`++` operator, `show` function)
- [ ] Pattern matching evaluation
- [ ] Tuple evaluation support
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