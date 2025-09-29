# AILANG Development Guide

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
make test-coverage-badge  # Quick coverage check (shows current %)
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

### Auto-rebuild on File Changes
```bash
make watch-install        # Automatically rebuilds and installs on file changes
make quick-install        # Fast reinstall after changes
```

## Project Structure

```
ailang/
├── cmd/ailang/          # CLI entry point with REPL
├── internal/
│   ├── repl/            # Interactive REPL
│   ├── ast/             # Abstract syntax tree definitions
│   ├── lexer/           # Tokenizer with full Unicode support
│   ├── parser/          # Recursive descent parser
│   ├── eval/            # Tree-walking interpreter
│   ├── core/            # Core AST with ANF
│   ├── elaborate/       # Surface to Core elaboration
│   ├── typedast/        # Typed AST
│   ├── types/           # Type system with HM inference
│   ├── link/            # Dictionary linker
│   ├── schema/          # Schema registry for AI features
│   ├── errors/          # Error code taxonomy and JSON encoder
│   ├── test/            # Test reporter
│   ├── manifest/        # Example manifest system
│   ├── module/          # Module loader and path resolver
│   ├── effects/         # Effect system (TODO)
│   ├── channels/        # CSP implementation (TODO)
│   ├── session/         # Session types (TODO)
│   └── typeclass/       # Type classes (TODO)
├── testutil/            # Testing utilities
├── examples/            # Example AILANG programs
├── docs/                # Documentation
├── design_docs/         # Design documents
└── scripts/             # CI/CD scripts
```

## Adding a New Language Feature

1. **Update token definitions** in `internal/lexer/token.go`
2. **Modify lexer** in `internal/lexer/lexer.go` to recognize tokens
3. **Add AST nodes** in `internal/ast/ast.go`
4. **Update parser** in `internal/parser/parser.go`
5. **Add type rules** in `internal/types/`
6. **Implement evaluation** in `internal/eval/`
7. **Write tests** in corresponding `*_test.go` files
8. **Add examples** in `examples/`
9. **Update documentation**

## Adding a Binary Operator

1. Add token in `token.go`
2. Add to lexer switch statement
3. Define precedence in parser
4. Add to `parseInfixExpression`
5. Add type rule
6. Implement evaluation
7. Write tests

## Adding a Built-in Function

1. Define type signature
2. Add to prelude or appropriate module
3. Implement in Go
4. Add tests
5. Document in examples

## Testing Guidelines

### Running Tests
```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run specific package tests
go test ./internal/lexer        # Tokenization
go test ./internal/parser       # Parsing  
go test ./internal/eval         # Evaluation
go test ./internal/types        # Type inference & defaulting
go test ./internal/elaborate    # Dictionary elaboration
go test ./cmd/test_integration  # End-to-end type class tests

# Run with verbose output
go test -v ./...
```

### Test Coverage Highlights
- ✅ Complete type class resolution pipeline
- ✅ Spec-aligned defaulting (neutral vs primary classes)
- ✅ Dictionary-passing transformation
- ✅ ANF verification and idempotency
- ✅ Law-compliant Float instances
- ✅ Superclass provision (Ord provides Eq)

### Writing Tests
- Each module should have a corresponding `*_test.go` file
- Test both success and error cases
- Use table-driven tests for multiple inputs
- Include integration tests for complete programs

## Code Style Guidelines

### Go Code
- Follow standard Go conventions
- Use descriptive names
- Add comments for complex logic
- Keep functions under 50 lines

### AILANG Code
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

## Performance Considerations
- Parser uses Pratt parsing for efficient operator precedence
- Type inference should cache resolved types
- String interning for identifiers
- Lazy evaluation for better performance (future)

## Documentation Requirements

### Required Documentation Updates
Every change must update:

1. **README.md**
   - Implementation status when adding features
   - Current capabilities when functionality changes
   - Examples when fixed or added
   - Line counts and completion status

2. **CHANGELOG.md**
   - Follow semantic versioning
   - Group by: Added, Changed, Fixed, Deprecated, Removed
   - Include code locations
   - Note breaking changes
   - Add migration notes if needed

3. **Design Documentation**
   - Create design doc in `design_docs/planned/` before starting
   - Move to `design_docs/implemented/` after completing
   - Include implementation report with metrics

4. **Example Files**
   - Create `examples/feature_name.ail` for each new feature
   - Include comprehensive examples
   - Add comments explaining behavior
   - Test that examples actually work

## Testing Policy

**ALWAYS remove out-of-date tests. No backward compatibility.**
- When architecture changes, delete old tests completely
- Don't maintain legacy test suites  
- Write new tests for new implementations
- Keep test suite clean and current

## Quick Debugging Checklist
- [ ] Check lexer is producing correct tokens
- [ ] Verify parser is building proper AST
- [ ] Ensure all keywords are in the keywords map
- [ ] Confirm precedence levels are correct
- [ ] Check that all AST nodes implement correct interfaces
- [ ] Verify type substitution is working correctly