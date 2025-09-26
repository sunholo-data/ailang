# AILANG: The AI-First Programming Language

AILANG is a purely functional programming language designed specifically for AI-assisted software development. It provides static typing with algebraic effects, typed quasiquotes for safe string handling, CSP-based concurrency with session types, and automatic generation of training data for AI model improvement.

## Current Implementation Status

### 🚀 NEW: REPL Fully Working with Type Classes! (v2.3)

**Full Type Class Resolution Pipeline**
- ✅ **Dictionary-Passing Transformation** (~770 lines)
  - Operators transformed to explicit dictionary calls in ANF
  - Idempotent elaboration safe for REPL multi-passes
  - Original source positions preserved for diagnostics
- ✅ **Type Normalization & Registry** (~500 lines)
  - Deterministic canonical type representations
  - Law-compliant Float instances (reflexive NaN equality)
  - Built-in Num, Eq, and Ord dictionaries
- ✅ **ANF Verifier & Linker** (~440 lines)
  - Validates A-Normal Form discipline
  - Resolves dictionary references with dry-run mode
  - Idempotency verification for transformations
- ✅ **Core Evaluator with Dictionary Support** (~850 lines)
  - Full DictApp/DictRef evaluation
  - Method dispatch through dictionary records
  - Wrapper functions for Go implementations
- ✅ **Spec-Aligned Numeric Defaulting** (Haskell-style)
  - Neutral classes (Eq, Ord, Show) don't affect defaulting
  - Primary classes (Num, Fractional) drive defaulting decisions
  - Module-scoped defaults: Num → Int, Fractional → Float
- **Total additions: ~2,560 lines of production code**
- **Status: Type classes fully operational with spec-compliant defaulting**

### 🎉 Phase 1 REPL & UX Integration Complete (v2.3)

**REPL Now Fully Operational!**
- ✅ **Professional Interactive REPL** with type class support (~850 lines)
  - **WORKING:** Arrow key history navigation (↑/↓ to browse command history)
  - **WORKING:** Persistent history across sessions (saved in ~/.ailang_history)
  - **WORKING:** Tab completion for REPL commands
  - **WORKING:** Proper :quit command that actually exits (also :q or Ctrl+D)
  - **WORKING:** Interactive evaluation with type inference and defaulting
  - **WORKING:** Full type class resolution with dictionary-passing
  - **WORKING:** Module import system for loading instances
  - **WORKING:** Rich diagnostic commands (dump-core, dump-typed, dry-link)
  - **WORKING:** Instance browser with superclass tracking
  - **WORKING:** Auto-imports std/prelude on startup
- ✅ Core AST with A-Normal Form (ANF) representation (~350 lines)
- ✅ Elaborator transforms surface AST to Core (~1,290 lines with dict support)  
- ✅ Type checker produces immutable TypedAST (~2,050 lines with defaulting)
- ✅ TypedAST evaluator with trace generation (~650 lines)
- ✅ Let-polymorphism with generalization at bindings only
- ✅ Recursive bindings via LetRec (parsing complete)
- ✅ Linear capability capture analysis
- ✅ Fail-fast on unsolved constraints
- **Total v2.0-2.2: ~6,360 lines of production code**
- **Status: Complete type class pipeline operational with interactive REPL**

### ✅ Core Features Working

**Lambda Expressions & Functional Programming**
- Complete lambda syntax with `\x. body` notation
- Closures with proper environment capture
- Currying and partial application: `\x y. x + y`
- Higher-order functions and function composition
- Record field access with dot notation

**Type System (Complete with Type Classes)**
- Hindley-Milner type inference with let-polymorphism
- Type class constraints with dictionary-passing
- Spec-aligned numeric defaulting (neutral vs primary classes)
- Principal row unification for records and effects
- Value restriction for sound polymorphism
- Kind system (Effect, Record, Row kinds)
- Linear capability capture analysis
- ~5,000+ lines total (includes Core typechecker + dictionaries)

**Basic Language Features**
- Arithmetic operations with correct precedence
- String concatenation with `++` operator
- Conditional expressions (`if then else`)
- Let bindings with type inference
- Records and record field access
- Lists (parsing complete, evaluation partial)
- Built-in functions: `print`, `show`, `toText`

## Installation

### From GitHub Releases

Download pre-built binaries for your platform from the [latest release](https://github.com/sunholo-data/ailang/releases/latest):

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

### From Source

```bash
# Clone the repository
git clone https://github.com/sunholo/ailang.git
cd ailang

# Build and install
make install

# Verify installation
ailang --version
```

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
├── cmd/ailang/          # CLI entry point with REPL ✅ (REPL working!)
├── internal/
│   ├── repl/            # Interactive REPL ✅ NEW (850 lines, fully working!)
│   ├── ast/             # Abstract syntax tree definitions ✅ (complete)
│   ├── lexer/           # Tokenizer with full Unicode support ✅ (fully working)
│   ├── parser/          # Recursive descent parser ✅ (1,059 lines, mostly complete)
│   ├── eval/            # Tree-walking interpreter ✅ (~2,100 lines total)
│   │   ├── eval_simple.go     # Main evaluator with show/toText functions
│   │   ├── eval_typed.go      # TypedAST evaluator ✅ (650 lines)
│   │   ├── eval_core.go       # Core evaluator with dictionaries ✅ NEW (850 lines)
│   │   ├── eval_simple_test.go # Comprehensive test suite
│   │   ├── value.go           # Value type definitions
│   │   └── env.go             # Variable environment
│   ├── core/            # Core AST with ANF ✅ (350 lines)
│   │   └── core.go            # A-Normal Form with dictionary nodes
│   ├── elaborate/       # Surface to Core elaboration ✅ (1,290 lines)  
│   │   ├── elaborate.go       # ANF transformation & dictionary-passing
│   │   └── verify.go          # ANF verifier & idempotency check ✅ NEW
│   ├── typedast/        # Typed AST ✅ (NEW - 260 lines)
│   │   └── typed_ast.go       # Immutable typed representation
│   ├── types/           # Type system with HM inference ✅ (~5,000 lines total)
│   │   ├── types.go           # Core type definitions
│   │   ├── types_v2.go        # Enhanced types with kinds
│   │   ├── kinds.go           # Kind system (Effect, Record, Row)
│   │   ├── inference.go       # HM type inference engine
│   │   ├── unification.go     # Type unification with occurs check
│   │   ├── row_unification.go # Principal row unifier
│   │   ├── normalize.go       # Type normalization ✅ NEW (260 lines)
│   │   ├── dictionaries.go    # Dictionary registry ✅ NEW (380 lines)
│   │   ├── env.go             # Type environments
│   │   ├── errors.go          # Rich error reporting
│   │   ├── typechecker.go     # Main type checking interface
│   │   └── typechecker_core.go # Core type checker with defaulting ✅ (2,050 lines)
│   ├── link/            # Dictionary linker ✅ NEW (270 lines)
│   │   └── linker.go          # Resolves dictionary references
│   ├── effects/         # Effect system (TODO)
│   ├── channels/        # CSP implementation (TODO)
│   ├── session/         # Session types (TODO)
│   └── typeclass/       # Type classes (TODO)
├── examples/            # Example AILANG programs
│   ├── arithmetic.ail   # Basic arithmetic ✅ WORKING
│   ├── lambda_expressions.ail # Lambda features ✅ WORKING
│   ├── simple.ail       # Simple expressions ✅ WORKING  
│   ├── lambdas_v2.ail   # Lambda currying ✅ WORKING
│   ├── type_class_showcase.ail # Type classes demo ✅ NEW
│   ├── num_demo.ail     # Num dictionary examples ✅ NEW
│   ├── debug*.ail       # Simple test files ✅ NEW
│   ├── show_demo.ail    # Show/toText functions ⚠️ PARTIAL
│   ├── hello.ail        # Hello world ⚠️ needs func syntax
│   ├── factorial.ail    # Factorial ⚠️ needs recursion
│   └── (30+ more examples in various states)
├── quasiquote/          # Typed templates (TODO)
├── stdlib/              # Standard library (TODO)
├── tools/               # Development tools (TODO)
├── cmd/test_v2/         # Phase 1 pipeline tester ✅ NEW
├── cmd/test_v2_verbose/ # Verbose pipeline demo ✅ NEW
└── design_docs/         # Language design documentation
```

## Getting Started

### Prerequisites

- Go 1.21 or later
- Make (optional but recommended)
- fswatch (optional, for auto-rebuild)
- readline library (included in most systems)

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
# Start the REPL with full type class support
ailang repl

# Run a file
ailang run examples/simple.ail

# Check a file (parsing only)
ailang check examples/hello.ail

# Show version
ailang --version
```

### REPL Usage (v2.3) - NOW WORKING!

The AILANG REPL is now fully operational with professional-grade interactive development:

#### Interactive Features (All Working!)
- **Arrow Key History**: Use ↑/↓ arrows to navigate through command history ✅
- **Persistent History**: Commands saved across sessions in `~/.ailang_history` ✅
- **Tab Completion**: Press Tab to complete REPL commands ✅
- **Auto-imports**: `std/prelude` loaded automatically on startup ✅
- **Clean Exit**: Use `:quit`, `:q`, or Ctrl+D to exit properly ✅

#### REPL Commands

```bash
# Basic Commands
λ> :help               # Show help
λ> :quit               # Exit REPL (also :q or Ctrl+D)
λ> :history            # Show command history
λ> :clear              # Clear screen
λ> :reset              # Reset environment

# Type System Commands
λ> :type <expr>        # Show type of expression
λ> :import <module>    # Import module instances
λ> :instances          # Show available type class instances

# Debugging Commands
λ> :dump-core          # Toggle Core AST display
λ> :dump-typed         # Toggle Typed AST display
λ> :dry-link           # Show required instances without evaluating
λ> :trace-defaulting on/off  # Enable/disable defaulting trace
```

#### Example REPL Session

```ailang
λ> 1 + 2
3 :: Int

λ> 3.14 * 2.0
6.28 :: Float

λ> "Hello " ++ "AILANG!"
Hello AILANG! :: String

λ> [1, 2, 3]
[1, 2, 3] :: [Int]

λ> {name: "Alice", age: 30}
{name: Alice, age: 30} :: {name: String, age: Int}

λ> :type \x. x + x
\x. x + x :: ∀α. Num α ⇒ α → α
λ> :trace-defaulting on
Defaulting trace enabled

λ> let double = \x. x + x in double(21)
42 :: Int

λ> :instances
Available instances:
  Num:
    • Num[Int], Num[Float]
  Eq:
    • Eq[Int], Eq[Float], Eq[String], Eq[Bool]
  Ord:
    • Ord[Int] (provides Eq[Int])
    • Ord[Float] (provides Eq[Float])
    • Ord[String] (provides Eq[String])
  Show:
    • Show[Int], Show[Float], Show[String], Show[Bool]

λ> :quit               # Exit REPL (also :q or Ctrl+D)
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
# Run all tests (ALL PASSING as of v2.1)
make test

# Run tests with coverage
make test-coverage

# Run specific package tests
go test ./internal/lexer        # Tokenization
go test ./internal/parser       # Parsing  
go test ./internal/eval          # Evaluation
go test ./internal/types         # Type inference & defaulting
go test ./internal/elaborate     # Dictionary elaboration
go test ./cmd/test_integration   # End-to-end type class tests

# Run with verbose output
go test -v ./...
```

**Test Coverage Highlights:**
- ✅ Complete type class resolution pipeline
- ✅ Spec-aligned defaulting (neutral vs primary classes)
- ✅ Dictionary-passing transformation
- ✅ ANF verification and idempotency
- ✅ Law-compliant Float instances
- ✅ Superclass provision (Ord provides Eq)

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

### Type Classes & Dictionary-Passing (NEW!)

```ailang
-- Type class constraints are resolved to dictionary calls
-- All numeric literals default to Int unless used with Float operations

-- Integer arithmetic (Num[Int] dictionary)
let sum = 1 + 2 + 3             -- Defaults to Int: 6
let calc = 10 * 5 - 20 / 4      -- All Int operations: 45

-- Float arithmetic (Num[Float] dictionary)
let pi = 3.14159
let area = pi * 2.0 * 2.0       -- Circle area: ~12.56

-- Equality tests (Eq dictionary)
let eq1 = 42 == 42              -- Eq[Int]: true
let eq2 = "hello" == "hello"    -- Eq[String]: true

-- Ordering comparisons (Ord dictionary)
let lt = 5 < 10                 -- Ord[Int]: true
let gt = "zebra" > "apple"      -- Ord[String]: true

-- Polymorphic functions with constraints
let double = \x. x + x          -- Num a => a -> a
let result1 = double(21)        -- Instantiated at Int: 42
let result2 = double(1.5)       -- Instantiated at Float: 3.0

-- After elaboration, operations become dictionary calls:
-- 1 + 2 transforms to:
-- let $dict = dict_Num_Int in
-- DictApp($dict, "add", [1, 2])
```

## Implementation Status

AILANG now has a complete type class resolution system with dictionary-passing transformation. Here's the current state:

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

#### **Type System** (Full Type Classes! ~5,000 lines)
- ✅ **Hindley-Milner type inference** with let-polymorphism
- ✅ **Principal row unification** for records and effects  
- ✅ **Kind system** with separate kinds for Effect/Record/Row
- ✅ **Value restriction** for sound polymorphism with effects
- ✅ **Linear capture analysis** for lambda expressions (compile-time errors)
- ✅ **Type class constraint solving** (Num, Ord, Eq with superclasses)
- ✅ **Numeric literal defaulting** (Haskell-style ambiguity resolution)
- ✅ **Dictionary-passing transformation** (operators → dictionary calls)
- ✅ **Law-compliant instances** (reflexive NaN, total float ordering)
- ✅ **ANF verification** ensures well-formed Core programs
- ✅ **Idempotent transformations** safe for REPL multi-passes
- ✅ **Rich error reporting** with paths and suggestions
- ✅ **FULLY INTEGRATED**: Complete pipeline from parsing to evaluation

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

### 🚧 What's Next (Phase 2 - Enhanced Language Features)

**Now that REPL is working, next priorities**:
1. **Function Declarations** - Support `func` syntax in elaboration
2. **Pattern Matching** - Elaborate match expressions to Core  
3. **Recursive Bindings** - Test LetRec with factorial/fibonacci
4. **Module System** - Import/export of functions and types
5. **Standard Library** - Core modules (io, collections, etc.)

### ❌ Not Yet Implemented
- **Effect System** - Algebraic effects with capabilities
- **Standard Library** - Core modules (io, collections, concurrent)
- **Quasiquotes** - Typed templates for SQL, HTML, regex, etc.
- **CSP Concurrency** - Channels with session types
- **Property Testing** - Built-in property-based testing
- **Training Export** - AI training data generation with typed traces
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

### Release Process

See [docs/RELEASE.md](docs/RELEASE.md) for details on:
- Creating releases with semantic versioning
- GitHub Actions CI/CD workflows
- Binary distribution and installation methods

### Areas Needing Help
- Parser completion for expressions and statements
- Type inference implementation
- Standard library modules
- Documentation and examples
- Testing and bug fixes

## License

Apache 2.0 - See LICENSE file for details