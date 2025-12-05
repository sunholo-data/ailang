# Sprint Plan: M-S-CALL0 Statement-Level Parsing

## Summary
Complete the S-CALL0 syntactic sugar implementation by adding statement-level support for zero-argument function calls (`f()` → `f(())`). Currently works in expression contexts (inside `if`, `match`, lambdas) but not at top-level statements.

**Duration:** 1 day (4-5 hours)
**Dependencies:** None (S-CONS and S-ARROWTYPE already complete)
**Risk Level:** Low (focused change, clear implementation path)

## Current Status Analysis

### Completed Recently (Last 14 Days)
- ✅ S-CONS sugar: ~95 LOC parser + ~150 LOC tests (expression context)
- ✅ S-ARROWTYPE sugar: ~45 LOC parser + tests
- ✅ S-CALL0 partial: ~15 LOC (expression context only)
- ✅ `::` cons builtin: ~200 LOC implementation + tests + unification fixes
- ✅ Strict syntax mode: ~120 LOC across parser/pipeline/REPL

### Velocity
- Recent average: ~100-150 LOC/day for parser features
- Estimated capacity: ~180 LOC for this sprint (4-5 hours)

### Current Limitations
- ⏳ S-CALL0 works in expressions but NOT at statement level
- ⏳ Workaround required: `f (())` with space instead of `f()`

## Proposed Milestone

### Milestone 1: Statement-Level S-CALL0 Support
**Goal:** Enable `f()` syntax at top-level/statement-level by adding lookahead detection after identifier parsing.

**Estimated:** ~60 LOC implementation + ~80 LOC tests = ~140 LOC total
**Duration:** 4-5 hours (1 day)

**Implementation Approach:** Option 1 from design doc (Statement-Level Lookahead)

**Tasks:**

**Hour 1: Parser Modification (~60 LOC)**
- Modify `parseStatement()` or statement parsing loop in `parser.go`
- Add lookahead check after parsing identifier expression
- Detect pattern: `IDENT LPAREN RPAREN` (no whitespace)
- Route to existing `parseCallExpression()` when detected
- Respect `strictSyntaxMode` flag (error if sugar disabled)
- Ensure `sugarUsed` flag is set correctly

**Hour 2-3: Test Implementation (~80 LOC)**
- Un-skip the `TestSugarCall0_Skip` test
- Add comprehensive test cases:
  - Top-level zero-arg call: `myFunc()`
  - Multiple sequential calls: `func1(); func2()`
  - Mixed with other statements
  - Strict mode rejection with helpful error
  - Canonical syntax still works: `f (())`
  - No regression in expression contexts
- Add edge cases:
  - Whitespace variations (should fail gracefully)
  - Nested calls
  - Inside blocks

**Hour 4: Integration & Documentation (~30 min)**
- Run full test suite (ensure no regressions)
- Verify linting clean
- Update prompt documentation:
  - Remove "expression context only" warning
  - Add top-level usage examples
  - Update v0.4.1 prompt hash
- Update CHANGELOG.md

**Hour 5: Verification (~30 min)**
- Create working example file: `examples/sugar_call0.ail`
- Test with `--strict-syntax` flag
- Verify REPL shows "(desugared)" feedback
- Run `make test` and `make lint`

**Acceptance Criteria:**
- [ ] `myFunc()` parses correctly at top-level (not just expressions)
- [ ] Desugars to `myFunc(())` with unit literal argument
- [ ] Expression context continues to work (no regression)
- [ ] Strict mode rejects `f()` with error: "Use `f (())` instead of `f()`"
- [ ] `sugarUsed` flag set correctly for REPL feedback
- [ ] All existing sugar tests pass
- [ ] New tests added: top-level, strict mode, edge cases
- [ ] Linting clean (no new warnings)
- [ ] Example file created and verified working
- [ ] Prompt v0.4.1 updated (remove "expression only" limitation)

**Files to Modify:**
- `internal/parser/parser.go` (~40 LOC) - Statement-level lookahead
- `internal/parser/parser_expr.go` (~10 LOC) - Minor adjustments if needed
- `internal/parser/sugar_test.go` (~80 LOC) - Un-skip + new tests
- `examples/sugar_call0.ail` (~30 LOC) - New example file
- `prompts/v0.4.1.md` (~10 LOC) - Remove limitation warning
- `prompts/versions.json` (~5 LOC) - Update hash
- `CHANGELOG.md` (~15 LOC) - Document fix

**Risks:**
- **Low risk**: Statement parsing isolation means changes won't affect expression parsing
- **Mitigation**: Comprehensive tests covering both statement and expression contexts
- **Rollback plan**: Single commit, easy to revert if issues arise

## Implementation Details

### Parser Modification Approach

```go
// In internal/parser/parser.go (approximate implementation)

func (p *Parser) parseStatement() ast.Node {
    expr := p.parseExpression(LOWEST)

    // S-CALL0: Check if identifier is immediately followed by ()
    // This handles the top-level case: myFunc()
    if ident, ok := expr.(*ast.Identifier); ok {
        if p.curTokenIs(lexer.LPAREN) && p.peekTokenIs(lexer.RPAREN) {
            // Detected f() pattern at statement level
            if p.strictSyntaxMode {
                p.reportSugarError("CALL0", "f()", "f (())")
                return expr
            }

            // Parse as call expression (routes to parseCallExpression)
            return p.parseCallExpression(ident)
        }
    }

    return expr
}
```

### Test Coverage Strategy

**Test Categories:**
1. **Basic functionality** - `myFunc()` at top level works
2. **Multiple calls** - `func1(); func2()` both work
3. **Strict mode** - Rejected with helpful error
4. **Canonical syntax** - `f (())` still works
5. **Expression context** - No regression (already tested)
6. **Edge cases** - Whitespace, nested, inside blocks

**Example Test:**
```go
func TestSugarCall0_TopLevel(t *testing.T) {
    input := `myFunc()`
    l := lexer.New(input, "test.ail")
    p := New(l)
    file := p.ParseFile()

    AssertNoErrors(t, p)

    // Should parse as FuncCall with unit argument
    funcCall, ok := file.Statements[0].(*ast.FuncCall)
    if !ok {
        t.Fatalf("Expected FuncCall, got %T", file.Statements[0])
    }

    AssertIdentifier(t, funcCall.Func, "myFunc")

    // Should have 1 arg: unit literal
    if len(funcCall.Args) != 1 {
        t.Fatalf("Expected 1 argument, got %d", len(funcCall.Args))
    }

    unitLit, ok := funcCall.Args[0].(*ast.Literal)
    if !ok || unitLit.Kind != ast.UnitLit {
        t.Fatalf("Expected unit literal argument")
    }
}
```

## Success Metrics
- Test coverage: Maintain >90% for parser package
- All 7 new S-CALL0 tests passing
- Example file: `examples/sugar_call0.ail` works correctly
- Documentation: Prompt v0.4.1 updated (no "expression only" limitation)
- All tests passing: ✅
- All linting passing: ✅
- No regressions in existing sugar features (S-CONS, S-ARROWTYPE)

## Dependencies
- None (standalone parser change)
- S-CONS and S-ARROWTYPE already complete
- Strict syntax mode infrastructure already in place

## Open Questions
- ❓ Should we backport this to v0.4.0 or keep in v0.4.1?
  - **Recommendation**: Keep in v0.4.1 (not breaking, just enhancement)
- ❓ Do we need special handling for method calls `obj.method()`?
  - **Answer**: Not initially - focus on simple function calls first

## Notes

### Design Decision: Why Option 1 (Statement-Level Lookahead)?
- **Pros**: Minimal change, clear separation of concerns, low risk
- **Cons**: Slight duplication between statement/expression handling
- **Alternatives considered**:
  - Option 2 (Unified parsing) - too large/risky
  - Option 3 (Lexer fusion) - violates design principles

### Implementation Strategy
1. Start with simplest case (single top-level call)
2. Add tests progressively (TDD approach)
3. Handle edge cases once basic functionality works
4. Keep commits small and focused

### Example Usage After Implementation

```ailang
module example/call0

import std/io (println)

export func greet() -> () ! {IO} {
  println("Hello!")
}

export func compute() -> int {
  42
}

export func main() -> () ! {IO} {
  greet()                  -- ✅ Works now! (was: greet (()))
  let x = compute();       -- ✅ Works (already worked in expressions)
  println(show(x))
}
```

### Prompt Update Preview

**Before:**
```markdown
### S-CALL0: Zero-Argument Calls (`f()`)

**⚠️ Known Limitation:** S-CALL0 only works in **expression contexts**
(inside lambdas, if/then, function arguments). At top-level/statement-level,
you MUST still use the canonical `f ()` syntax with a space.
```

**After:**
```markdown
### S-CALL0: Zero-Argument Calls (`f()`)

**Sugar:** `f()` (zero-arg call, v0.4.1+)
**Canonical:** `f (())` (function applied to unit)
**Works everywhere:** Both statement-level and expression contexts

```ailang
-- ✅ Works at top-level (v0.4.1+)
myFunc()

-- ✅ Works in expressions (v0.4.0+)
let result = if ready() then compute() else 0
```

### Known Risks & Mitigations

**Risk 1: Whitespace sensitivity edge cases**
- Example: `myFunc ( )` with space inside parens
- Mitigation: Only trigger sugar for `()` (no whitespace)
- Test coverage: Add explicit edge case tests

**Risk 2: Regression in expression context**
- Existing S-CALL0 logic in `parseCallArguments()` must still work
- Mitigation: Keep existing code path unchanged, add new path for statements
- Test coverage: Run all existing expression context tests

**Risk 3: Interaction with other syntax**
- Chained calls: `obj.method()`
- Nested calls: `outer(inner())`
- Mitigation: Start with simple function calls, test progressively
- Test coverage: Add integration tests

### Verification Checklist

Before marking complete:
- [ ] Parser changes: `git diff internal/parser/` looks reasonable
- [ ] Test changes: All new tests have meaningful assertions
- [ ] Example file: `ailang run --caps IO --entry main examples/sugar_call0.ail` works
- [ ] Strict mode: `ailang run --strict-syntax examples/sugar_call0.ail` errors correctly
- [ ] REPL feedback: `:type myFunc()` shows "(desugared)"
- [ ] No regressions: `make test` passes completely
- [ ] Clean build: `make lint` has no new warnings
- [ ] Docs updated: Prompt hash matches file hash

### Timeline

**Day 1 (4-5 hours total):**
- 09:00-10:00: Parser modification (~60 LOC)
- 10:00-12:00: Test implementation (~80 LOC)
- 12:00-12:30: Integration & docs (~30 min)
- 12:30-13:00: Verification & cleanup (~30 min)

**Expected completion:** Same day
**Buffer:** None needed (very focused change)
**Blocker potential:** None (standalone feature)

---

## References
- Design doc: `design_docs/planned/v0_4_1/m-s-call0-statement-parsing.md`
- Surface sugar design: `design_docs/planned/v0_4_1/surface-sugar-pack.md`
- Existing tests: `internal/parser/sugar_test.go`
- Parser code: `internal/parser/parser.go`, `internal/parser/parser_expr.go`
