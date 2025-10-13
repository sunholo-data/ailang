# Function Body Block Expressions

**Status**: 📋 PLANNED
**Priority**: P0 (HIGH - Usability)
**Estimated Effort**: ~150 LOC, 4-6 hours
**Target Release**: v0.3.6 or v0.4.0
**Created**: 2025-10-13

---

## Problem Statement

Function declarations currently only support equation-form bodies (`func f() = expr`), but cannot use block syntax with multiple statements (`func f() { stmt1; stmt2 }`). This creates inconsistency and forces ugly workarounds.

### Current Behavior

```ailang
-- ❌ DOESN'T WORK: Block syntax in function body
func main() -> () ! {IO} {
    println("Hello");
    println("World")
}
-- Parser error: "expected next token to be }, got () instead"

-- ✅ WORKAROUND: Nested let bindings (ugly!)
func main() -> () ! {IO} =
    let _ = println("Hello") in
    println("World")
```

### Why This Matters

1. **AI Code Generation**: Models naturally generate block syntax
   - GPT/Claude output: `func f() { ... }` (doesn't parse)
   - Forces manual rewriting to equation form
   - Breaks M-EVAL benchmarks

2. **Inconsistency**: Blocks work everywhere else
   - Expressions: `let x = { e1; e2 } in ...` ✅
   - Lambda bodies: `\x. { e1; e2 }` ✅
   - Function bodies: `func f() { e1; e2 }` ❌

3. **Usability**: Workaround is non-obvious
   - Nested `let _ = ...` is verbose
   - Beginners confused by syntax restriction
   - Reduces language ergonomics

---

## Proposed Solution

Support **both** equation-form and block-form function bodies:

```ailang
-- Equation form (existing, keep working)
func pure(x: int) -> int = x * 2

-- Block form (new, to be added)
func impure() -> () ! {IO} {
    println("Statement 1");
    println("Statement 2")
}

-- Single-expression blocks (should work)
func simple() -> int { 42 }
```

---

## Design

### Syntax

```ebnf
FuncDecl ::=
    "func" IDENT TypeParams? "(" Params ")" ReturnType? Effects? FuncBody

FuncBody ::=
    | "=" Expr              -- Equation form (existing)
    | "{" BlockBody "}"     -- Block form (new)

BlockBody ::=
    | Expr                           -- Single expression
    | Expr ";" BlockBody             -- Multiple statements
    | "let" IDENT "=" Expr ";" BlockBody   -- Let binding
```

### Parser Changes

**File**: `internal/parser/parser.go`

Current parsing (equation form only):
```go
func (p *Parser) parseFuncDecl() *ast.FuncDecl {
    // ... parse name, params, type ...

    if !p.expectPeek(lexer.ASSIGN) {  // Expects '='
        return nil
    }
    p.nextToken()
    decl.Body = p.parseExpression(LOWEST)

    return decl
}
```

Proposed parsing (support both forms):
```go
func (p *Parser) parseFuncDecl() *ast.FuncDecl {
    // ... parse name, params, type ...

    // Detect which form based on next token
    if p.peekTokenIs(lexer.ASSIGN) {
        // Equation form: func f() = expr
        p.nextToken()
        p.nextToken()
        decl.Body = p.parseExpression(LOWEST)
    } else if p.peekTokenIs(lexer.LBRACE) {
        // Block form: func f() { body }
        p.nextToken() // move to LBRACE
        decl.Body = p.parseBlockOrExpression() // Reuse existing helper!
    } else {
        p.errors = append(p.errors, fmt.Errorf(
            "expected '=' or '{' for function body at %s",
            p.peekToken.Position()))
        return nil
    }

    return decl
}
```

**Key insight**: We can reuse `parseBlockOrExpression()` from the func expression implementation! It already handles:
- Single expressions: `{ 42 }` → `42`
- Multiple statements: `{ e1; e2 }` → nested Lets
- Empty blocks: `{}` → `()`

### Elaboration Changes

**File**: `internal/elaborate/elaborate.go`

**No changes needed!** Block expressions already elaborate correctly:
- `{ e1; e2; e3 }` → `let _ = e1 in let _ = e2 in e3`
- This is done in `normalizeBlock()`

Since `FuncDecl.Body` is already an `Expr`, and blocks are expressions, the existing elaboration "just works".

### Type Checking Changes

**File**: `internal/types/infer.go`

**No changes needed!** Type checking already handles block expressions:
- Infers type of last expression in sequence
- Handles let bindings in blocks
- No special cases for function bodies vs other blocks

---

## Implementation Plan

### Phase 1: Parser (2-3 hours, ~50 LOC)

1. Modify `parseFuncDecl()` to detect `=` vs `{`
2. Reuse existing `parseBlockOrExpression()` for block bodies
3. Add tests for both syntaxes

**Files Modified**:
- `internal/parser/parser.go` (~30 LOC)
- `internal/parser/parser_test.go` (~20 LOC)

### Phase 2: Testing & Examples (1-2 hours, ~100 LOC)

1. Add test cases for block-form functions
2. Create example files demonstrating both forms
3. Update teaching prompt with new syntax

**Files Modified/Created**:
- `internal/parser/parser_test.go` (~50 LOC - tests)
- `examples/function_blocks.ail` (~30 LOC - example)
- `prompts/v0.3.0.md` (~20 LOC - documentation)

### Phase 3: Validation (1 hour)

1. Run existing tests (should all pass)
2. Verify M-EVAL benchmarks improve
3. Test with AI code generation

---

## Examples

### Before (Workaround Required)

```ailang
func processFile(path: string) -> () ! {IO, FS} =
    let content = readFile(path) in
    let _ = println("Processing: " ++ path) in
    let lines = split(content, "\n") in
    let _ = println("Lines: " ++ show(length(lines))) in
    println("Done")
```

### After (Natural Syntax)

```ailang
func processFile(path: string) -> () ! {IO, FS} {
    let content = readFile(path);
    println("Processing: " ++ path);
    let lines = split(content, "\n");
    println("Lines: " ++ show(length(lines)));
    println("Done")
}
```

### Both Forms Supported

```ailang
-- Equation form: Good for pure, single-expression functions
func double(x: int) -> int = x * 2

func triple(x: int) -> int = x * 3

-- Block form: Good for multi-statement, effectful functions
func greet(name: string) -> () ! {IO} {
    println("Hello, " ++ name);
    println("Welcome to AILANG!")
}

-- Block form works with single expressions too
func answer() -> int { 42 }
```

---

## Backward Compatibility

✅ **Fully backward compatible**:
- All existing code uses `=` form
- Parser accepts both forms
- No breaking changes

---

## Testing Strategy

### Unit Tests

```go
func TestParseFuncDecl_EquationForm(t *testing.T) {
    input := "func f(x: int) -> int = x * 2"
    // ... assert parses correctly
}

func TestParseFuncDecl_BlockForm(t *testing.T) {
    input := "func f(x: int) -> int { x * 2 }"
    // ... assert parses correctly
}

func TestParseFuncDecl_BlockFormMultiStatement(t *testing.T) {
    input := `func f() -> () ! {IO} {
        println("a");
        println("b")
    }`
    // ... assert parses correctly
}
```

### Integration Tests

Run existing test suite - all should pass with no changes.

### M-EVAL Tests

Re-run M-EVAL benchmarks - expect improvements in:
- Benchmarks that use multi-statement functions
- AI-generated code that naturally uses blocks

---

## Risks & Mitigations

### Risk: Parser Ambiguity
**Issue**: Could `{` be confused with record literal?
**Mitigation**: Context makes it clear:
- After function signature → function body
- After `=` → expression (could be record)
- No ambiguity in practice

### Risk: Subtle Semantic Differences
**Issue**: Do `=` and `{}` behave identically?
**Mitigation**: Yes, both are expressions:
- `func f() = expr` → body is `expr`
- `func f() { expr }` → body is `expr` (block with single expr)
- Block elaboration is well-tested

---

## Success Criteria

1. ✅ Parser accepts both `=` and `{}` forms
2. ✅ All existing tests pass (backward compat)
3. ✅ New tests for block form pass
4. ✅ M-EVAL benchmarks improve
5. ✅ AI-generated code parses correctly

---

## Future Work

### Potential Extensions

1. **Statement syntax** (sugar for let bindings):
   ```ailang
   func f() {
       x = 1;        -- Sugar for: let x = 1 in ...
       y = x + 1;    -- Sugar for: let y = x + 1 in ...
       y
   }
   ```

2. **Early return** (via match on Result):
   ```ailang
   func f() -> Result<int, string> {
       let x = try(parse("123"));  -- Returns early if Error
       Ok(x * 2)
   }
   ```

3. **Mutable variables** (via State monad):
   ```ailang
   func counter() -> int ! {State} {
       var count = 0;
       count = count + 1;
       count
   }
   ```

---

## References

- [Block Expressions](../../docs/LIMITATIONS.md#block-expressions) - Current implementation
- [FuncLit Implementation](../implemented/v0_3/P0_anonymous_function_syntax.md) - Already uses `parseBlockOrExpression()`
- [M-EVAL Benchmarks](../../benchmarks/) - Tests that will benefit

---

## Implementation Checklist

- [ ] Modify `parseFuncDecl()` to support both forms
- [ ] Add parser tests for block-form functions
- [ ] Create example file demonstrating both syntaxes
- [ ] Update teaching prompt (prompts/v0.3.0.md)
- [ ] Run full test suite
- [ ] Run M-EVAL validation
- [ ] Update CHANGELOG.md
- [ ] Update README.md with new syntax
