# M-S-CALL0: Zero-Arg Call Sugar - Statement-Level Parsing Fix

**Status:** 📋 Planned
**Priority:** Medium
**Estimated Effort:** 4-6 hours
**Blocked By:** None
**Version:** v0.4.1 or v0.4.2

## Problem

The S-CALL0 syntactic sugar (`f()` → `f(())`) was partially implemented but doesn't work at the statement level due to AILANG's whitespace-sensitive parsing.

### Current Behavior

```ailang
// ❌ At top level - parsed as TWO statements
myFunc()
// Parsed as: [myFunc, ()]

// ✅ Inside expression - works!
let x = if true then myFunc() else 0
// Correctly parsed as: if true then myFunc(()) else 0

// ✅ Canonical syntax - always works
myFunc (())
```

### Root Cause

When parsing top-level statements, the parser treats each expression as a separate statement. The sequence `myFunc()` is tokenized as:
1. `IDENT(myFunc)`
2. `LPAREN`
3. `RPAREN`

At the statement level, after parsing `myFunc` as a complete expression, the parser moves to the next statement and encounters `()`, treating it as a unit literal instead of continuing the call expression.

**Why expression context works:**
Inside expressions (like `if` bodies, lambda bodies, etc.), the Pratt parser is used. When it parses `myFunc`, it looks for infix operators. `LPAREN` is registered as an infix operator (for function calls), so it triggers `parseCallExpression`, which then calls `parseCallArguments`. This is where our S-CALL0 logic lives and works correctly.

**Why statement level fails:**
At the top level, `ParseFile` iterates and parses each statement separately. After parsing `myFunc`, it doesn't know to look for more tokens that might continue the expression. It treats `myFunc` as complete and moves on.

## Discovered Implementation

The S-CALL0 sugar logic exists in `internal/parser/parser_expr.go`:

```go
func (p *Parser) parseCallArguments() []ast.Expr {
    args := []ast.Expr{}

    // S-CALL0: Check for zero-arg call sugar f()
    if p.peekTokenIs(lexer.RPAREN) {
        p.nextToken() // consume RPAREN

        // Check if strict syntax mode is enabled
        if p.strictSyntaxMode {
            p.reportSugarError("CALL0", "f()", "f ()")
            return args
        }

        // Sugar is allowed - desugar f() to f(())
        p.sugarUsed = true

        // Return unit literal as single argument
        unitLit := &ast.Literal{
            Kind:  ast.UnitLit,
            Value: nil,
            Pos:   p.curPos(),
        }
        return []ast.Expr{unitLit}
    }
    // ... rest of function
}
```

This code works perfectly when reached through the Pratt expression parser (inside `if`, `match`, lambda bodies, etc.), but is never reached at the statement level.

## Solution Approaches

### Option 1: Statement-Level Lookahead (Recommended)

Modify `ParseFile` or statement parsing to check for `()` immediately after an identifier.

**Pros:**
- Focused fix, doesn't affect expression parsing
- Maintains current architecture
- Clear separation between statement and expression handling

**Cons:**
- Duplicates some logic between statement and expression parsing
- Adds special case to statement parser

**Estimated effort:** 4 hours

**Implementation sketch:**
```go
func (p *Parser) parseStatement() ast.Node {
    expr := p.parseExpression(LOWEST)

    // S-CALL0: Check if identifier is immediately followed by ()
    if ident, ok := expr.(*ast.Identifier); ok {
        if p.curTokenIs(lexer.LPAREN) && p.peekTokenIs(lexer.RPAREN) {
            // Detected f() pattern at statement level
            return p.parseCallExpression(ident)
        }
    }

    return expr
}
```

### Option 2: Unified Expression Parsing

Always use Pratt parser for both statements and expressions.

**Pros:**
- Eliminates distinction between statement and expression parsing
- More consistent behavior
- Fixes S-CALL0 and potentially other similar issues

**Cons:**
- Larger architectural change
- Might affect other statement-level constructs
- Higher risk of regressions

**Estimated effort:** 6-8 hours

### Option 3: Lexer-Level Token Fusion

Modify lexer to recognize `()` as a special token when immediately following an identifier (no whitespace).

**Pros:**
- Fixes issue at tokenization level
- No parser changes needed

**Cons:**
- Violates lexer/parser separation of concerns
- Makes lexer context-sensitive
- Harder to maintain
- Against AILANG design principles (ML-style whitespace sensitivity)

**Estimated effort:** 3-4 hours (but not recommended)

## Recommended Approach

**Option 1: Statement-Level Lookahead**

Add special handling in statement parsing to detect `f()` pattern and route it through the call expression parser.

### Implementation Plan

1. **Add lookahead check in statement parser** (~1 hour)
   - Modify `parseStatement()` or equivalent
   - Check for `IDENT LPAREN RPAREN` pattern
   - Route to `parseCallExpression` when detected

2. **Handle strict mode** (~30 min)
   - Ensure `strictSyntaxMode` flag is respected
   - Emit helpful error when sugar disabled

3. **Add comprehensive tests** (~1.5 hours)
   - Top-level zero-arg calls
   - Nested contexts (already work, verify no regression)
   - Strict mode rejection
   - Edge cases (whitespace variations)

4. **Update existing tests** (~30 min)
   - Re-enable skipped S-CALL0 tests
   - Verify all sugar tests pass

5. **Documentation** (~30 min)
   - Update teaching prompt
   - Add examples showing top-level usage
   - Document in CHANGELOG

**Total:** ~4 hours

## Acceptance Criteria

- [ ] `myFunc()` works at top level (parsed as `myFunc(())`)
- [ ] Expression context continues to work (no regression)
- [ ] Strict mode correctly rejects `f()` with helpful error
- [ ] `sugarUsed` flag is set correctly
- [ ] All tests pass (existing + new)
- [ ] Linting clean
- [ ] Examples updated and verified

## Testing Strategy

```ailang
// Test 1: Top-level zero-arg call
myFunc()
// Expected: FuncCall(myFunc, [Unit])

// Test 2: Top-level with multiple calls
func1()
func2()
// Expected: Two separate FuncCalls

// Test 3: Nested context (already works)
let x = if true then func() else 0
// Expected: If(true, FuncCall(func, [Unit]), Lit(0))

// Test 4: Strict mode rejection
// With --strict-syntax
myFunc()
// Expected: Error "Use `myFunc (())` instead of `myFunc()`"

// Test 5: Canonical syntax still works
myFunc (())
// Expected: FuncCall(myFunc, [Unit])
```

## Dependencies

- None (self-contained parser change)

## Risks

- **Low risk**: Focused change to statement parsing
- **Mitigation**: Comprehensive test coverage, including regression tests
- **Rollback plan**: Revert single commit if issues arise

## Follow-up Tasks

- None (self-contained feature)

## Notes

- This issue was discovered during v0.4.1 sprint implementation
- S-CONS and S-ARROWTYPE work perfectly; only S-CALL0 has this limitation
- Workaround exists: Use canonical `f (())` syntax (with space)
- Not blocking for v0.4.1 release - can be follow-up in v0.4.2

## References

- Original design doc: `design_docs/planned/v0_4_2/surface-sugar-pack.md`
- Implementation PR: (to be added)
- Test file: `internal/parser/sugar_test.go`
