# Claude Instructions for AILANG Development

## Project Overview
AILANG is an AI-first programming language designed for AI-assisted development. It features:
- Pure functional programming with algebraic effects (planned)
- Typed quasiquotes for safe metaprogramming (planned)
- CSP-based concurrency with session types (planned)
- Deterministic execution for AI training data generation (planned)
- File extension: `.ail`

## Current Status: v0.1.0 MVP (Type System Complete)

**✅ COMPLETE:**
- Hindley-Milner type inference with let-polymorphism
- Type classes (Num, Eq, Ord, Show) with dictionary-passing
- Lambda calculus (first-class functions, closures, currying)
- Interactive REPL with full type checking
- Module system (type-checking only - execution in v0.2.0)
- Expression evaluation (arithmetic, strings, conditionals, let bindings)
- Structured error reporting with JSON schemas

**❌ NOT YET IMPLEMENTED:**
- Module execution runtime (coming in v0.2.0)
- Effect system (coming in v0.2.0)
- Pattern matching (coming in v0.2.0)
- Typed quasiquotes (v0.3.0+)
- CSP concurrency (v0.3.0+)
- AI training data export (v0.3.0+)

**⚠️ CRITICAL LIMITATION:** Module files (with `module` declarations) parse and type-check correctly but cannot execute. Only non-module `.ail` files can run. See [docs/LIMITATIONS.md](docs/LIMITATIONS.md).

## Key Design Principles
1. **Explicit Effects**: All side effects must be declared in function signatures
2. **Everything is an Expression**: No statements, only expressions that return values
3. **Type Safety**: Static typing with Hindley-Milner inference + row polymorphism
4. **Deterministic**: All non-determinism must be explicit (seeds, virtual time)
5. **AI-Friendly**: Generate structured execution traces for training

## Project Structure (v0.1.0)
```
ailang/
├── cmd/ailang/         # CLI entry point (main.go) ✅
├── internal/
│   ├── ast/            # AST definitions ✅ COMPLETE
│   ├── lexer/          # Tokenizer ✅ COMPLETE
│   ├── parser/         # Parser ✅ COMPLETE (some limitations*)
│   ├── types/          # Type system ✅ COMPLETE
│   ├── typeclass/      # Type classes ✅ COMPLETE
│   ├── eval/           # Evaluator ✅ PARTIAL (non-module files only)
│   ├── repl/           # Interactive REPL ✅ COMPLETE
│   ├── module/         # Module resolution ✅ COMPLETE (type-checking)
│   ├── errors/         # Error reporting ✅ COMPLETE
│   ├── schema/         # JSON schemas ✅ COMPLETE
│   ├── effects/        # Effect system ❌ TODO (v0.2.0)
│   ├── channels/       # CSP implementation ❌ TODO (v0.3.0+)
│   └── session/        # Session types ❌ TODO (v0.3.0+)
├── stdlib/             # Standard library ✅ PARTIAL (prelude, option, result)
├── tools/              # Development tools ✅ (audit-examples.sh)
├── examples/           # Example .ail programs (42 total, 12 working)
├── tests/              # Test suite ✅
└── docs/               # Documentation ✅ COMPLETE

*Parser limitations: 3-deep let nesting limit, no match expressions yet
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

### ✅ Working Syntax (v0.1.0)

**Basic Expressions:**
```ailang
-- Comments use double dash
let x = 5 in x * 2                     -- Let binding (works up to 3 nested)
\x. x * 2                               -- Lambda function
if x > 0 then "pos" else "neg"         -- Conditional expression
[1, 2, 3]                               -- List literal
{ name: "Alice", age: 30 }             -- Record literal
(1, "hello", true)                      -- Tuple literal
1 + 2 * 3                               -- Arithmetic with precedence
"Hello " ++ "World"                     -- String concatenation
```

**REPL-Only Features:**
```ailang
λ> :type \x. x + x
\x. x + x :: ∀α. Num α ⇒ α → α

λ> let double = \x. x * 2 in double(21)
42 :: Int

λ> :instances
Available instances: Num[Int], Num[Float], Eq[Int], Eq[Float], Ord[Int], Ord[Float]
```

**Module Syntax (Type-Checks Only, Does Not Execute):**
```ailang
module examples/demo

import stdlib/std/option (Option, Some, None)

export type MyData = {
  value: Int
}

export func process(x: Int) -> Option[Int] {
  if x > 0 then Some(x * 2) else None
}
```

**⚠️ Note**: The above module syntax parses and type-checks but cannot execute until v0.2.0.

### ❌ Planned Syntax (Not Yet Implemented)

**Pattern Matching (v0.2.0):**
```ailang
match value {
  Some(x) if x > 0 => x * 2,
  Some(x) => x,
  None => 0
}
```

**Effect Handlers (v0.2.0):**
```ailang
func readAndPrint() -> () ! {IO, FS} {
  let content = readFile("data.txt")?
  print(content)
}
```

**Quasiquotes (v0.3.0+):**
```ailang
let query = sql"""SELECT * FROM users WHERE age > ${minAge: int}"""
```

**Concurrency (v0.3.0+):**
```ailang
func worker(ch: Channel[Task]) ! {Async} {
  loop {
    let task <- ch
    ch <- process(task)
  }
}
```

## What Works & What Doesn't (v0.1.0)

### ✅ Working Examples
- `examples/hello.ail` - Simple print
- `examples/simple.ail` - Basic arithmetic
- `examples/arithmetic.ail` - Arithmetic with show
- `examples/type_classes_working_reference.ail` - Type classes demo
- `examples/showcase/*.ail` - Type inference, lambdas, closures, type classes
- **REPL** - Fully functional with all type system features

See [examples/STATUS.md](examples/STATUS.md) for complete inventory (12 working, 3 type-check only, 27 broken).

### ⚠️ Known Limitations (v0.1.0)

**Parser Limitations:**
1. ✅ Module/import/export **parse and type-check** but cannot execute (v0.2.0)
2. ⚠️ Let expressions limited to 3 nesting levels (4+ fails)
3. ✅ Pattern matching **works** (constructors, tuples, lists, wildcards) - v0.2.0 adds guards + exhaustiveness
4. ❌ `tests [...]` and `properties [...]` syntax not implemented
5. ⚠️ Non-module files cannot use `func`, `type`, `import`, `export` keywords

**Execution Limitations:**
1. ❌ **Critical**: Module files type-check but cannot execute
2. ⚠️ REPL and file execution use different code paths (intentional)
3. ⚠️ Type classes work in REPL and module type-checking, not in non-module file execution
4. ⚠️ Record field access has unification bugs in some cases
5. ⚠️ List operations have limited runtime support

See [docs/LIMITATIONS.md](docs/LIMITATIONS.md) for comprehensive details and workarounds.

### 🚀 v0.2.0 Roadmap (3.5-4.5 weeks)

**Status**: Design complete, ready for implementation

**M-R1: Module Execution Runtime** (~1,000-1,300 LOC, 1.5-2 weeks)
- Module instance creation and evaluation
- Cross-module imports at runtime
- Entrypoint execution (`--entry`, `--args-json`)
- `--runner=fallback` to preserve v0.1.0 wrapper

**M-R2: Minimal Effect Runtime** (~700-900 LOC, 1-1.5 weeks)
- Capability-based security (`--caps IO,FS`)
- IO effect: `print`, `println`, `readLine`
- FS effect: `readFile`, `writeFile`, `exists`
- Secure by default (no caps unless explicitly granted)

**M-R3: Pattern Matching Polish** (~450-650 LOC, 1 week) [STRETCH]
- Guards: `pattern if condition => body`
- Exhaustiveness warnings with suggested missing cases
- Decision tree compilation for performance

**See**: [v0.2.0 Implementation Plan](design_docs/planned/v0_2_0_implementation_plan.md)

### 📋 Future (v0.3.0+)
1. Effect composition DSL, budgets, async effects
2. Typed quasiquotes (SQL, HTML, JSON)
3. CSP concurrency with channels
4. Session types
5. Property-based testing
6. AI training data export

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

## v0.2.0 Development Guide

### Getting Started with v0.2.0

The v0.2.0 implementation is divided into three sequential milestones. Start with M-R1, as M-R2 and M-R3 depend on it.

**Design Documents**:
- [v0.2.0 Implementation Plan](design_docs/planned/v0_2_0_implementation_plan.md) - High-level overview
- [M-R1: Module Execution](design_docs/planned/m_r1_module_execution.md) - Detailed M-R1 design
- [M-R2: Effect System](design_docs/planned/m_r2_effect_system.md) - Detailed M-R2 design
- [M-R3: Pattern Matching](design_docs/planned/m_r3_pattern_matching.md) - Detailed M-R3 design

### M-R1: Module Execution (Week 1-2)

**Objective**: Make `ailang run module.ail` actually execute module exports

**Key Files to Create**:
- `internal/runtime/module.go` - `ModuleInstance` struct (~200 LOC)
- `internal/runtime/runtime.go` - `ModuleRuntime` (~300 LOC)
- `internal/runtime/resolver.go` - `GlobalResolver` implementation (~100 LOC)
- `internal/runtime/linker.go` - Topological module linking (~200 LOC)

**Key Files to Modify**:
- `cmd/ailang/main.go` - Wire module runtime (~100 LOC changes)
- `internal/eval/eval_core.go` - Add `SetGlobalResolver` (~50 LOC)

**Testing Strategy**:
1. Unit tests for `ModuleInstance`, `ModuleRuntime` (~500 LOC)
2. Integration tests with multi-module programs (~400 LOC)
3. Update `make verify-examples` to test module examples

**Acceptance**: 25-30 module examples execute successfully

### M-R2: Effect Runtime (Week 2.5-3.5)

**Objective**: Execute IO/FS effects with capability security

**Key Files to Create**:
- `internal/effects/capability.go` - `Capability` struct (~50 LOC)
- `internal/effects/context.go` - `EffContext` (~100 LOC)
- `internal/effects/ops.go` - Effect operation registry (~150 LOC)
- `internal/effects/io.go` - IO operations (~150 LOC)
- `internal/effects/fs.go` - FS operations (~200 LOC)

**Key Files to Modify**:
- `cmd/ailang/main.go` - Add `--caps` flag (~50 LOC)
- `internal/eval/eval_core.go` - Add `effContext` field (~100 LOC)

**Testing Strategy**:
1. Unit tests for each effect operation (~750 LOC)
2. Integration tests with capability checks (~300 LOC)
3. Test sandbox mode (`AILANG_FS_SANDBOX`)

**Acceptance**: IO/FS examples work; missing caps → clear error

### M-R3: Pattern Matching Polish (Week 3.5-4.5)

**Objective**: Add guards, exhaustiveness warnings, decision trees

**Key Files to Create**:
- `internal/elaborate/exhaustiveness.go` - Exhaustiveness checker (~200 LOC)
- `internal/elaborate/warnings.go` - Warning generation (~100 LOC)
- `internal/elaborate/decision_tree.go` - Decision tree compilation (~150 LOC)
- `internal/eval/decision_tree.go` - Decision tree evaluation (~100 LOC)

**Key Files to Modify**:
- `internal/ast/ast.go` - Add `Guard` field to `MatchClause` (~10 LOC)
- `internal/parser/parser.go` - Parse guards (~50 LOC)
- `internal/elaborate/match.go` - Elaborate guards (~50 LOC)
- `internal/eval/eval_core.go` - Evaluate guards (~30 LOC)

**Testing Strategy**:
1. Unit tests for exhaustiveness checker (~200 LOC)
2. Unit tests for decision trees (~200 LOC)
3. Integration tests for guards (~100 LOC)

**Acceptance**: Guards work, exhaustiveness warnings appear

### Development Commands

```bash
# Module runtime development
make test                          # Run all tests
go test ./internal/runtime/...     # Test module runtime
make verify-examples               # Verify examples work

# Effect runtime development
go test ./internal/effects/...     # Test effects
AILANG_FS_SANDBOX=/tmp make test  # Test with sandbox

# Pattern matching development
go test ./internal/elaborate/...   # Test exhaustiveness
go test ./internal/eval/...        # Test evaluation

# Full CI check
make ci                           # Run all checks locally
```

### Code Style for v0.2.0

**Module Runtime**:
- Use `sync.Once` for thread-safe initialization
- Cache module instances by canonical path
- Clear error messages for circular dependencies

**Effect Runtime**:
- Secure by default: no capabilities unless explicitly granted
- Use `EffContext` for all effect operations
- Sandbox FS operations when `AILANG_FS_SANDBOX` set

**Pattern Matching**:
- Warn (don't error) on non-exhaustive matches
- Format missing patterns clearly
- Keep naive evaluation as fallback if decision trees fail

### Common Pitfalls

1. **M-R1**: Circular imports - Use topological sort, detect cycles early
2. **M-R2**: Forgetting capability checks - Always call `ctx.RequireCap()`
3. **M-R3**: False positive warnings - Be conservative, allow `--no-exhaustive-check`

### Current Test Coverage
- **Overall**: 24.8% (as of v0.1.0)
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