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
**Always update the README.md when making changes to the codebase:**
- Update implementation status when adding new features
- Update current capabilities when functionality changes
- Update examples when they're fixed or new ones added
- Keep line counts and completion status accurate
- Document new builtin functions and operators
- Update the roadmap as items are completed

**CRITICAL: Example Files Required**
**Every new language feature MUST have a corresponding example file:**
- Create `examples/feature_name.ail` for each new feature
- Include comprehensive examples showing all capabilities
- Add comments explaining the behavior and expected output
- Test that examples actually work with current implementation
- These examples will be used in documentation and tutorials

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

### Immediate Fixes Needed
1. **Lexer**: Keywords not being recognized (all parsed as IDENT)
2. **Parser**: Import statements not parsing correctly
3. **Parser**: Module declarations incomplete
4. **Parser**: Pattern matching not fully implemented

### Major Components to Implement
1. **Type Inference**: Hindley-Milner with effect inference
2. **Interpreter**: Tree-walking evaluator
3. **Effect System**: Capability checking and propagation
4. **Standard Library**: Core modules (prelude, io, collections, concurrent)
5. **Quasiquotes**: Validation and AST generation
6. **Training Export**: Execution trace collection

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