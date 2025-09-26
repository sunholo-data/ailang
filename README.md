# AILANG: The AI-First Programming Language

AILANG is a purely functional programming language designed specifically for AI-assisted software development. It provides static typing with algebraic effects, typed quasiquotes for safe string handling, CSP-based concurrency with session types, and automatic generation of training data for AI model improvement.

## Current Implementation Status

### ✅ Core Features Working

**Lambda Expressions & Functional Programming**
- Complete lambda syntax with `\x. body` notation
- Closures with proper environment capture
- Currying and partial application: `\x y. x + y`
- Higher-order functions and function composition
- Record field access with dot notation

**Type System (Foundation Complete)**
- Hindley-Milner type inference with let-polymorphism
- Principal row unification for records and effects
- Value restriction for sound polymorphism
- Kind system (Effect, Record, Row kinds)
- Linear capability capture analysis
- ~2,500 lines of working type system code

**Basic Language Features**
- Arithmetic operations with correct precedence
- String concatenation with `++` operator
- Conditional expressions (`if then else`)
- Let bindings with type inference
- Records and record field access
- Lists (parsing complete, evaluation partial)
- Built-in functions: `print`, `show`, `toText`

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
│   ├── arithmetic.ail   # Basic arithmetic ✅ WORKING
│   ├── lambda_expressions.ail # Lambda features ✅ WORKING
│   ├── type_demo_minimal.ail # Type system demo ✅ WORKING
│   ├── show_demo.ail    # Show/toText functions ✅ PARTIAL
│   ├── hello.ail        # Hello world (needs func syntax)
│   ├── factorial.ail    # Factorial (needs recursion)
│   └── simple.ail       # Simple expressions ✅ WORKING
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

### Lambda Expressions

```ailang
-- Lambda syntax with closures
let add = \x y. x + y in
let add5 = add(5) in  -- Partial application
print("Result: " ++ show(add5(3)))  -- Result: 8

-- Higher-order functions
let compose = \f g x. f(g(x)) in
let double = \x. x * 2 in
let inc = \x. x + 1 in
let doubleThenInc = compose(inc)(double) in
print("Composed: " ++ show(doubleThenInc(5)))  -- Composed: 11
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

#### **Parser** (Nearly Complete) 
- Recursive descent with Pratt parsing (~1,200 lines)
- ✅ **Working**: Basic expressions, let bindings, if-then-else, lists, records
- ✅ **Working**: Binary/unary operators with spec-compliant precedence
- ✅ **Working**: **Lambda expressions with `\x.` syntax and currying**
- ✅ **Working**: **Record field access with correct precedence** 
- ✅ **Working**: Module declarations and import statements
- ⚠️ **Parsed but not evaluated**: Pattern matching, type annotations
- ❌ **Not working**: `?` operator, effect handlers, tuples

#### **Evaluator** (Major Features Working)
- Tree-walking interpreter (~700 lines)
- ✅ **Working**: Arithmetic, booleans, strings, let bindings, if-then-else
- ✅ **Working**: Lists, records (creation and field access)
- ✅ **Working**: **Lambda expressions with proper closures**
- ✅ **Working**: **Higher-order functions and partial application**
- ✅ **Working**: **Record field access with chaining (a.b.c)**
- ✅ **Working**: `show` and `toText` builtins, `++` operator
- ❌ **Not working**: Pattern matching, tuples, effect handlers

#### **Type System** (Fully Implemented! ~2,500 lines)
- ✅ **Hindley-Milner type inference** with let-polymorphism
- ✅ **Principal row unification** for records and effects  
- ✅ **Kind system** with separate kinds for Effect/Record/Row
- ✅ **Value restriction** for sound polymorphism with effects
- ✅ **Linear capture analysis** for lambda expressions (compile-time errors)
- ✅ **Constraint collection** for type classes (Num, Ord, Eq, Show)
- ✅ **Rich error reporting** with paths and suggestions
- ⚠️ **Not integrated**: Type checker works standalone but not connected to evaluator

### Testing Status
- ✅ **Lexer tests**: All passing (including lambda tokens)
- ✅ **Parser tests**: Comprehensive coverage including lambdas
  - Lambda syntax: ✅ PASS (`\x.`, currying, precedence)
  - Body extent: ✅ PASS (correct precedence parsing)
  - Record access: ✅ PASS (field chaining)
- ✅ **Evaluator tests**: Core and advanced features tested
  - Lambda closures: ✅ PASS (environment capture, isolation)
  - Higher-order functions: ✅ PASS (partial application)
  - Record field access: ✅ PASS (chained access)
- ✅ **Type inference tests**: All algorithms passing
  - Row unification: ✅ PASS
  - Occurs check: ✅ PASS  
  - Kind mismatch detection: ✅ PASS
  - Value restriction: ✅ PASS
  - Linear capture analysis: ✅ PASS
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
- ✅ Lists: `[1, 2, 3]` (creation and usage)
- ✅ Records: `{name: "Alice", age: 30}` (creation and field access)
- ✅ **Lambda expressions**: `\x. x + 1`, `(\x y. x + y)(3)(4)` → `7`
- ✅ **Higher-order functions**: `(\f. f(5))(\x. x * 2)` → `10`
- ✅ **Closures**: `let y = 10 in (\x. x + y)(5)` → `15`
- ✅ **Record field access**: `person.name`, `user.profile.email`
- ✅ **Function composition**: `(\f g x. f(g(x)))(inc)(double)(5)`
- ✅ Builtins: `show(42)` → `"42"`, `toText(value)`, `print(value)`

### What Parses but Doesn't Evaluate
- ⚠️ Pattern matching: `match x { ... }`
- ⚠️ Type annotations: `let x: int = 5`
- ⚠️ Module imports: `import std/io`
- ⚠️ Function declarations: `func add(x, y) { x + y }`

### What Doesn't Parse Yet
- ❌ Tuples: `(1, "hello", true)`
- ❌ Effect handlers: `handle ... with { ... }`
- ❌ Result operator: `readFile(path)?`

### Builtin Functions
- `print(value)` - Outputs value to console
- `show(value)` - Converts any value to its string representation (quoted for strings)
- `toText(value)` - Converts value to string without quotes (for display)

### Operator Precedence (high to low) - Spec Compliant
1. **Field access** (`.`) - highest precedence
2. **Function application** (space: `f x`)
3. **Unary operators** (`not`, `-`)
4. **Multiplication/Division** (`*`, `/`, `%`)
5. **Addition/Subtraction** (`+`, `-`)
6. **String concatenation** (`++`)
7. **Comparisons** (`<`, `>`, `<=`, `>=`, `==`, `!=`)
8. **Logical AND** (`&&`)
9. **Logical OR** (`||`)
10. **Lambda expressions** (`\x. body`) - lowest precedence

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

The type system has been battle-tested with comprehensive test suites!

### Lambda Expression Examples (NEW!)

```ailang
-- Basic lambda expressions
let id = \x. x in id(42)                    -- Returns: 42
let add = \x y. x + y in add(3)(4)         -- Returns: 7

-- Closures (capturing environment)
let multiplier = 10 in
let scale = \x. x * multiplier in 
scale(5)                                   -- Returns: 50

-- Higher-order functions
let apply_twice = \f x. f(f(x)) in
let inc = \x. x + 1 in
apply_twice(inc)(5)                        -- Returns: 7

-- Function composition  
let compose = \f g x. f(g(x)) in
let double = \x. x * 2 in
let add_one = \x. x + 1 in
compose(double)(add_one)(3)                -- Returns: 8

-- Record field access with functions
let person = {name: "Alice", greet: \msg. "Hello, " ++ msg} in
person.greet(person.name)                  -- Returns: "Hello, Alice"
```

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
- No `?` operator for Result types
- No effect handler syntax  
- No tuple syntax `(a, b, c)`

#### Evaluator Limitations  
- Pattern matching doesn't execute
- Tuples not supported  
- Module imports not resolved
- Effect handlers not implemented

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