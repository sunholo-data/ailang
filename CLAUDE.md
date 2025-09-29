# Claude Instructions for AILANG Development

## Project Overview
AILANG is an AI-first programming language designed for AI-assisted development. It features:
- Pure functional programming with algebraic effects
- Typed quasiquotes for safe metaprogramming
- CSP-based concurrency with session types
- Deterministic execution for AI training data generation
- File extension: `.ail`

## Key Design Principles
1. **Explicit Effects**: All side effects must be declared in function signatures
2. **Everything is an Expression**: No statements, only expressions that return values
3. **Type Safety**: Static typing with Hindley-Milner inference + row polymorphism
4. **Deterministic**: All non-determinism must be explicit (seeds, virtual time)
5. **AI-Friendly**: Generate structured execution traces for training

## Project Structure
```
ailang/
├── cmd/ailang/         # CLI entry point (main.go)
├── internal/
│   ├── ast/            # AST definitions (complete)
│   ├── lexer/          # Tokenizer (needs fixes)
│   ├── parser/         # Parser (partial implementation)
│   ├── types/          # Type system (foundation only)
│   ├── effects/        # Effect system (TODO)
│   ├── eval/           # Interpreter (TODO)
│   ├── channels/       # CSP implementation (TODO)
│   ├── session/        # Session types (TODO)
│   └── typeclass/      # Type classes (TODO)
├── quasiquote/         # Typed templates (TODO)
├── stdlib/             # Standard library (TODO)
├── tools/              # Development tools (TODO)
├── examples/           # Example .ail programs
└── tests/              # Test suite
```

## Development Workflow

### Building and Testing
```bash
make build          # Build the interpreter to bin/
make install        # Install ailang to system (makes it available everywhere)
make test           # Run all tests
make run FILE=...   # Run an AILANG file
make repl           # Start interactive REPL
```

### Code Quality & Coverage
```bash
make test-coverage-badge  # Quick coverage check (shows: "Coverage: 29.9%")
make test-coverage        # Run tests with coverage, generates HTML report
make lint                 # Run golangci-lint
make fmt                  # Format all Go code
make fmt-check            # Check if code is formatted
make vet                  # Run go vet
```

### Example Management
```bash
make verify-examples      # Verify all example files work/fail
make update-readme        # Update README with example status
make flag-broken          # Add warning headers to broken examples
```

### Development Helpers
```bash
make deps                 # Install all dependencies
make clean                # Remove build artifacts and coverage files
make ci                   # Run full CI verification locally
make help                 # Show all available make targets
```

### Making `ailang` Accessible System-Wide

#### First-Time Setup
1. Install ailang to your Go bin directory:
   ```bash
   make install
   ```

2. Add Go bin to your PATH (if not already done):
   ```bash
   # For zsh (macOS default)
   echo 'export PATH="/Users/mark/go/bin:$PATH"' >> ~/.zshrc
   source ~/.zshrc
   
   # For bash
   echo 'export PATH="/Users/mark/go/bin:$PATH"' >> ~/.bashrc
   source ~/.bashrc
   ```

3. Test it works:
   ```bash
   ailang --version
   ```

#### Keeping `ailang` Up to Date

**Option 1: Manual Update**
After making code changes, run:
```bash
make quick-install  # Fast reinstall
# OR
make install        # Full reinstall with version info
```

**Option 2: Auto-Update on File Changes**
For development, use watch mode to automatically reinstall on every code change:
```bash
make watch-install  # Automatically rebuilds and installs on file changes
```
This watches all Go files and automatically updates the global `ailang` command.

**Option 3: Alias for Quick Updates**
Add this to your shell profile for a quick update command:
```bash
# Add to ~/.zshrc or ~/.bashrc
alias ailang-update='cd /Users/mark/dev/sunholo/ailang && make quick-install && cd -'
```
Then just run `ailang-update` from anywhere to update.

### IMPORTANT: Keeping Documentation Updated

**Required documentation updates for every change:**

#### 1. README.md
- Update implementation status when adding new features
- Update current capabilities when functionality changes
- Update examples when they're fixed or new ones added
- Keep line counts and completion status accurate
- Document new builtin functions and operators
- Update the roadmap as items are completed

#### 2. CHANGELOG.md
**Must be updated for every feature or bug fix:**
- Follow semantic versioning (vMAJOR.MINOR.PATCH)
- Group changes by category: Added, Changed, Fixed, Deprecated, Removed
- Include code locations for new features (e.g., `internal/schema/`)
- Note breaking changes clearly
- Add migration notes if needed
- Include metrics (lines of code, test coverage)

Example entry:
```markdown
## [v3.2.0] - 2024-09-28

### Added
- Schema Registry (`internal/schema/`) - Versioned JSON schemas
- Error JSON Encoder (`internal/errors/`) - Structured error reporting
- Test coverage: 100% for new packages
- Total new code: ~1,500 lines
```

#### 3. Design Documentation
- **Before starting**: Create design doc in `design_docs/planned/`
- **After completing**: Move to `design_docs/implemented/vX_Y/`
- Include implementation report with metrics and limitations

**CRITICAL: Example Files Required**
**Every new language feature MUST have a corresponding example file:**
- Create `examples/feature_name.ail` for each new feature
- Include comprehensive examples showing all capabilities
- Add comments explaining the behavior and expected output
- ⚠️ **Test that examples actually work with current implementation**
- ⚠️ **Add warning headers to examples that don't work**
- These examples will be used in documentation and tutorials

**⚠️ IMPORTANT: Most current examples in `/examples/` are broken**
- Only `hello.ail`, `simple.ail`, `arithmetic.ail`, `lambda_expressions.ail` work
- Examples using `module`, `func`, `type`, `import` will fail
- Always test examples before documenting them as working

### Common Tasks

#### Adding a New Language Feature
1. Update token definitions in `internal/lexer/token.go`
2. Modify lexer in `internal/lexer/lexer.go` to recognize tokens
3. Add AST nodes in `internal/ast/ast.go`
4. Update parser in `internal/parser/parser.go`
5. Add type rules in `internal/types/`
6. Implement evaluation in `internal/eval/` (when created)
7. Write tests in corresponding `*_test.go` files
8. Add examples in `examples/`

#### Implementing a Missing Component
Check the implementation sizes from the design doc:
- Lexer: ~200 lines (mostly complete)
- Parser: ~500 lines (needs completion)
- Types: ~800 lines (foundation only)
- Effects: ~400 lines (TODO)
- Eval: ~500 lines (TODO)
- Channels: ~400 lines (TODO)

## Language Syntax Reference

### Basic Constructs
```ailang
-- Comments use double dash
let x = 5                          -- Immutable binding
let f = (x: int) -> int => x * 2  -- Lambda function
if x > 0 then "pos" else "neg"    -- Conditional expression
[1, 2, 3]                          -- List literal
{ name: "Alice", age: 30 }        -- Record literal
(1, "hello", true)                 -- Tuple
```

### Functions
```ailang
-- Pure function (no effects)
pure func add(x: int, y: int) -> int {
  x + y
}

-- Effectful function
func readAndPrint() -> () ! {IO, FS} {
  let content = readFile("data.txt")?  -- ? propagates errors
  print(content)
}

-- With inline tests
pure func factorial(n: int) -> int
  tests [
    (0, 1),
    (5, 120)
  ]
{
  if n <= 1 then 1 else n * factorial(n - 1)
}
```

### Pattern Matching
```ailang
match value {
  Some(x) if x > 0 => x * 2,
  Some(x) => x,
  None => 0
}

match list {
  [] => "empty",
  [x] => "single",
  [head, ...tail] => "multiple"
}
```

### Quasiquotes
```ailang
-- SQL with type checking
let query = sql"""
  SELECT * FROM users 
  WHERE age > ${minAge: int}
"""

-- HTML with sanitization
let page = html"""
  <div>${content: SafeHtml}</div>
"""

-- Other quasiquotes: regex/, json{}, shell""", url"
```

### Effects and Capabilities
```ailang
import std/io (FS, Net)

func processData() -> Result[Data] ! {FS, Net} {
  with FS, Net {
    let data = readFile(FS, "input.txt")?
    let response = httpGet(Net, "api.example.com")?
    Ok(process(data, response))
  }
}
```

### Concurrency (CSP)
```ailang
func worker(ch: Channel[Task]) ! {Async} {
  loop {
    let task <- ch       -- Receive from channel
    let result = process(task)
    ch <- result         -- Send to channel
  }
}

parallel {
  spawn { worker(ch1) }
  spawn { worker(ch2) }
}  -- Waits for all spawned tasks
```

## Known Issues & TODOs

### ⚠️ CRITICAL: Documentation vs Reality Gap
**Many documented features don't actually work. Use these working examples:**
- ✅ `examples/hello.ail` - Simple print
- ✅ `examples/simple.ail` - Basic arithmetic  
- ✅ `examples/arithmetic.ail` - Arithmetic with show
- ✅ `examples/lambda_expressions.ail` - Full lambda functionality
- ✅ REPL with basic expressions

### Immediate Fixes Needed
1. ❌ **Parser**: Module declarations completely broken (`module`, `import`)
2. ❌ **Parser**: Function declarations don't work (`func` syntax)
3. ❌ **Parser**: Type definitions not supported (`type` declarations)
4. ❌ **Parser**: Tests/properties syntax fails (`tests [...]`, `properties [...]`)
5. ⚠️ **Integration**: REPL vs file execution use different evaluators
6. ⚠️ **Type System**: Type classes work in REPL but not file execution

### Major Components to Implement
1. ❌ **Module System**: Make `module` and `import` statements work
2. ❌ **Function Declarations**: Implement `func` syntax
3. ❌ **Type Definitions**: Support `type` declarations
4. ❌ **Integration**: Unify REPL and file execution paths
5. ❌ **Effect System**: Capability checking and propagation (not started)
6. ❌ **Standard Library**: Core modules (not started)
7. ❌ **Quasiquotes**: Validation and AST generation (not started)
8. ❌ **Training Export**: Execution trace collection (not started)

## REPL Usage (v2.3)

The AILANG REPL now features professional-grade interactive development with full type class support:

### Interactive Features
- **Arrow Key History**: Navigate command history with ↑/↓ arrows
- **Persistent History**: Commands saved in `~/.ailang_history`
- **Tab Completion**: Auto-complete REPL commands with Tab key
- **Auto-imports**: `std/prelude` loaded automatically on startup
- **Clean Exit**: `:quit` command properly exits the REPL

### Basic Usage
```bash
ailang repl
```

The REPL auto-imports `std/prelude` on startup, providing:
- Numeric defaults: `Num → Int`, `Fractional → Float`  
- Type class instances for `Num`, `Eq`, `Ord`, `Show`
- String concatenation with `++` operator
- Record literals and field access

### Key Commands
- `:help, :h` - Show all available commands
- `:quit, :q` - Exit the REPL (also works: Ctrl+D)
- `:type <expr>` - Show qualified type with constraints
- `:import <module>` - Import type class instances
- `:instances` - List available instances with superclass provisions
- `:dump-core` - Toggle Core AST display for debugging
- `:dump-typed` - Toggle Typed AST display
- `:dry-link` - Show required dictionary instances without evaluating
- `:trace-defaulting on/off` - Enable/disable defaulting trace
- `:history` - Show command history
- `:clear` - Clear the screen
- `:reset` - Reset environment (auto-reimports prelude)

### Example REPL Session

```ailang
λ> 1 + 2
3 :: Int

λ> 3.14 * 2.0
6.28 :: Float

λ> "Hello " ++ "AILANG!"
Hello AILANG! :: String

λ> true && false
false :: Bool

λ> [1, 2, 3]
[1, 2, 3] :: [Int]

λ> {name: "Alice", age: 30}
{name: Alice, age: 30} :: {name: String, age: Int}

λ> :type \x. x + x
\x. x + x :: ∀α. Num α ⇒ α → α

λ> let double = \x. x * 2 in double(21)
42 :: Int
```

### Type Class Pipeline
The REPL executes the full pipeline:
1. **Parse** - Surface syntax to AST
2. **Elaborate** - AST to Core (ANF)
3. **TypeCheck** - Infer types with constraints
4. **Dictionary Elaboration** - Transform operators to dictionary calls
5. **ANF Verification** - Ensure well-formed Core
6. **Link** - Resolve dictionary references
7. **Evaluate** - Execute with runtime dictionaries

### Example Session
```
λ> 1 + 2 * 3
:: Int
7

λ> :type 42 == 42
42 == 42 :: Bool

λ> :instances
Available instances:
  Num:
    • Num[Int], Num[Float]
  Eq:
    • Eq[Int], Eq[Float]
  Ord:
    • Ord[Int] (provides Eq[Int])
    • Ord[Float] (provides Eq[Float])
```

### Architecture Notes
- **Type-level instances** (`instEnv`) - Used during type checking and defaulting
- **Runtime dictionaries** (`instances`) - Used during evaluation
- Both must be kept in sync when importing modules
- Method names are standardized: `eq`/`neq`, `lt`/`lte`/`gt`/`gte`

## Testing Guidelines

### Unit Tests
- Each module should have a corresponding `*_test.go` file
- Test both success and error cases
- Use table-driven tests for multiple inputs

### Integration Tests
- Test complete programs in `examples/`
- Verify type checking catches errors
- Test effect propagation
- Ensure deterministic execution

### Property-Based Tests
AILANG supports inline property tests:
```ailang
property "sort preserves length" {
  forall(list: [int]) =>
    length(sort(list)) == length(list)
}
```

## Code Style Guidelines

1. **Go Code**:
   - Follow standard Go conventions
   - Use descriptive names
   - Add comments for complex logic
   - Keep functions under 50 lines

2. **AILANG Code**:
   - Use 2-space indentation
   - Prefer pure functions
   - Make effects explicit
   - Include tests with functions
   - Use type annotations when helpful

## Error Handling

### In Go Implementation
- Return explicit errors, don't panic
- Include position information in parse errors
- Provide helpful error messages with suggestions

### In AILANG
- Use Result type for fallible operations
- Propagate errors with `?` operator
- Provide structured error context

## Performance Considerations
- Parser uses Pratt parsing for efficient operator precedence
- Type inference should cache resolved types
- Lazy evaluation for better performance (future)
- String interning for identifiers

## Debug Commands
```bash
# Parse and print AST (when implemented)
ailang parse file.ail

# Type check without running
ailang check file.ail

# Show execution trace
ailang run --trace file.ail

# Export training data
ailang export-training
```

## Common Patterns

### Adding a Binary Operator
1. Add token in `token.go`
2. Add to lexer switch statement
3. Define precedence in parser
4. Add to `parseInfixExpression`
5. Add type rule
6. Implement evaluation

### Adding a Built-in Function
1. Define type signature
2. Add to prelude or appropriate module
3. Implement in Go
4. Add tests

## Resources
- Design doc: `design_docs/20250926/initial_design.md`
- Examples: `examples/` directory
- Go tests: `*_test.go` files

## Testing Policy
**ALWAYS remove out-of-date tests. No backward compatibility.**
- When architecture changes, delete old tests completely
- Don't maintain legacy test suites
- Write new tests for new implementations
- Keep test suite clean and current

## 🚨 CRITICAL: Linting & "Unused" Code Warnings

**⚠️ LESSON LEARNED: Never blindly delete "unused" functions without understanding WHY they're unused!**

### The Import System Disaster (September 2025)
In commit `eae08b6`, working import functions were deleted because linter said they were "unused".
**What actually happened:**
1. Function **calls** were renamed from `parseModuleDecl()` to `_parseModuleDecl()` (note underscore)
2. Function **definitions** kept original names (no underscore)
3. Calls were then **commented out**
4. Linter correctly said "hey, `parseModuleDecl` is never called!"
5. Functions were **blindly deleted**
6. Result: **Working import system completely broken** 💥

### Rules to Prevent This:
1. **NEVER delete functions just because linter says "unused"**
   - First understand WHY they're unused
   - Check git history - were they just commented out?
   - Search entire codebase for references (including comments)
   - Run `make test-imports` and `make test` BEFORE deleting anything

2. **If renaming function calls, rename definitions too**
   - Use IDE refactoring tools, not manual find/replace
   - If adding `_` prefix to mark as TODO, add to BOTH call and definition
   - Better: use TODO comments instead of renaming

3. **Test between each change**
   - Don't combine: rename + comment out + delete
   - Run tests after EACH step:
     - After rename → `make test`
     - After commenting out → `make test-imports`
     - After deleting → `make test && make lint`

4. **When linter complains about unused code:**
   ```bash
   # Step 1: Check if it's really unused
   git log -p --all -S 'functionName' internal/
   grep -r "functionName" internal/

   # Step 2: Check recent changes
   git log --oneline internal/parser/parser.go | head -5
   git diff HEAD~1 internal/parser/parser.go | grep functionName

   # Step 3: If truly unused AND you know why, document it
   git commit -m "Remove unused parseOldFormat() - replaced by parseNewFormat() in commit abc123"
   ```

5. **Special warning for parser/module/import code**
   - These are **critical** for language functionality
   - If you break these, **nothing imports work**
   - Always run `make test-imports` before committing parser changes
   - Check that example files still work: `make verify-examples`

### Recovery Checklist (if this happens again):
1. Find last working commit: `git log --all --oneline | grep "import"`
2. Check what was deleted: `git diff working_commit broken_commit`
3. Restore deleted functions: `git show working_commit:file.go`
4. Test imports: `make test-imports`
5. Document in commit message what was broken and how it was fixed

### Current Test Coverage
- **Overall**: 29.9% (as of 2024-09-29)
- **Well-tested**: `test` (95.7%), `schema` (87.9%), `parser` (75.8%), `errors` (75.9%)
- **Needs work**: `typedast` (0%), `eval` (15.6%), `types` (20.3%)
- Run `make test-coverage-badge` for quick coverage check

## Important Notes
1. The language is expression-based - everything returns a value
2. Effects are tracked in the type system - never ignore them
3. Pattern matching must be exhaustive
4. All imports must be explicit
5. Row polymorphism allows extensible records and effects
6. Session types ensure protocol correctness in channels

## Quick Debugging Checklist
- [ ] Check lexer is producing correct tokens
- [ ] Verify parser is building proper AST
- [ ] Ensure all keywords are in the keywords map
- [ ] Confirm precedence levels are correct
- [ ] Check that all AST nodes implement correct interfaces
- [ ] Verify type substitution is working correctly

## Contact & Support
This is an experimental language. For questions or issues:
- Check the design documents in @design_docs
- Look at example programs
- Run tests for expected behavior
- Refer to similar functional languages (Haskell, OCaml, F#)