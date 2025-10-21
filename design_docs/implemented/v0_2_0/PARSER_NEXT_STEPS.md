# Parser Next Steps: Pattern Matching in Function Bodies

**Date**: October 1, 2025
**Status**: BLOCKER for stdlib in AILANG
**Estimated Effort**: 1-2 days

---

## Problem Statement

Pattern matching works at the top-level of modules but fails inside function bodies.

### What Works ✅

```ailang
module test
type Option[a] = Some(a) | None

-- Top-level match expression: WORKS
match Some(42) {
  Some(n) => n,
  None => 0
}
-- Output: 42 ✅
```

### What Doesn't Work ❌

```ailang
module test
type Option[a] = Some(a) | None

export func getOrElse[a](opt: Option[a], d: a) -> a {
  match opt {  -- ❌ FAILS: "expected =>, got ] instead"
    Some(x) => x,
    None => d
  }
}
```

**Error Messages**:
- `PAR_UNEXPECTED_TOKEN: expected next token to be =>, got ] instead`
- `PAR_NO_PREFIX_PARSE: unexpected token in expression: ]`
- Affects list patterns: `[]`, `[x, ...rest]`
- Affects all pattern types inside function bodies

---

## Root Cause Analysis

### Evidence

1. **Parser has the features**:
   - `parseMatchExpression()` exists (line 968 in parser.go)
   - `parsePattern()` exists (line 1445 in parser.go)
   - Registered as prefix parser: `p.registerPrefix(lexer.MATCH, p.parseMatchExpression)` (line 120)

2. **Features work in some contexts**:
   - ✅ Top-level expressions in modules
   - ✅ REPL expressions
   - ❌ Inside function bodies

3. **Error location**:
   - Errors occur during pattern parsing inside match arms
   - Specifically when parsing list patterns `[]` and `[x, ...rest]`
   - Suggests expression context vs statement context issue

### Hypothesis

The parser may be treating function bodies differently from top-level expressions:
- Function bodies might use a different parsing mode
- Expression context might not be properly propagated
- Token lookahead might be different in nested contexts

**Most Likely**: `parseFunctionBody()` or block statement parsing doesn't properly set up the expression context for `parseMatchExpression()`.

---

## Investigation Steps

### 1. Trace Parsing Flow

```bash
# Add debug logging to parser.go
# Track context when parseMatchExpression() is called

func (p *Parser) parseMatchExpression() ast.Expr {
    log.Printf("DEBUG: parseMatchExpression called, curToken=%v, peekToken=%v", p.curToken, p.peekToken)
    // ... existing code
}
```

### 2. Compare Contexts

**Test Case 1** (works):
```ailang
module test
match Some(42) { Some(n) => n, None => 0 }
```

**Test Case 2** (fails):
```ailang
module test
export func f() -> int {
  match Some(42) { Some(n) => n, None => 0 }
}
```

**Questions to Answer**:
- What is `p.curToken` when entering `parseMatchExpression()` in each case?
- What is the parser state (in function body vs top-level)?
- Are there different parsing modes or contexts?

### 3. Check Block Statement Parsing

```go
// Find parseFunctionBody or parseBlockStatement
// Check how expressions are parsed inside blocks
// Ensure parseExpression() is called correctly
```

**Look for**:
- Does `parseBlockStatement()` call `parseExpressionStatement()`?
- Does expression parsing context propagate correctly?
- Are there any context flags that affect parsing?

---

## Potential Fixes

### Option 1: Fix Expression Context Propagation

If the issue is context not propagating:
```go
func (p *Parser) parseFunctionBody() {
    // Ensure expression context is set
    p.inFunctionBody = true  // Or similar flag
    // ... parse body
    p.inFunctionBody = false
}
```

### Option 2: Fix Block Statement Expression Parsing

If block statements don't parse expressions correctly:
```go
func (p *Parser) parseBlockStatement() *ast.BlockStatement {
    // Ensure all expression types are handled
    // Including match expressions
}
```

### Option 3: Unified Expression Parsing

Ensure `parseExpression()` is called consistently in all contexts:
```go
func (p *Parser) parseStatement() ast.Statement {
    switch p.curToken.Type {
    case lexer.MATCH:
        return p.parseExpressionStatement()  // Make sure this path works
    // ...
    }
}
```

---

## Testing Strategy

### Create Test Suite: `internal/parser/func_pattern_test.go`

```go
package parser

import "testing"

func TestPatternMatchingInFunctions(t *testing.T) {
    tests := []struct {
        name  string
        input string
        shouldParse bool
    }{
        {
            name: "simple pattern in function",
            input: `
                module test
                export func f(opt: Option) -> int {
                    match opt {
                        Some(n) => n,
                        None => 0
                    }
                }
            `,
            shouldParse: true,
        },
        {
            name: "list pattern in function",
            input: `
                module test
                export func length(xs: [int]) -> int {
                    match xs {
                        [] => 0,
                        [_, ...rest] => 1 + length(rest)
                    }
                }
            `,
            shouldParse: true,
        },
        {
            name: "nested pattern in function",
            input: `
                module test
                export func f(x: Option[Option[int]]) -> int {
                    match x {
                        Some(Some(n)) => n,
                        Some(None) => -1,
                        None => 0
                    }
                }
            `,
            shouldParse: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            p := New(tt.input, "<test>")
            program := p.ParseProgram()

            if tt.shouldParse && len(p.Errors()) > 0 {
                t.Errorf("Expected to parse but got errors: %v", p.Errors())
            }
            if !tt.shouldParse && len(p.Errors()) == 0 {
                t.Errorf("Expected parse errors but got none")
            }
        })
    }
}
```

### Incremental Testing

1. **Start simple**: Single constructor pattern
2. **Add complexity**: List patterns with `[]`
3. **Add more complexity**: List patterns with spread `...`
4. **Test nesting**: Patterns inside patterns
5. **Test multiple arms**: Multiple match cases
6. **Test guards**: Patterns with `if` guards (if supported)

---

## Success Criteria

- [ ] All list patterns work: `[]`, `[x]`, `[x, ...rest]`, `[x, y, ...rest]`
- [ ] Constructor patterns work: `Some(x)`, `Ok(y)`, `Cons(h, t)`
- [ ] Tuple patterns work: `(x, y)`, `(_, y, z)`
- [ ] Nested patterns work: `Some([x, ...xs])`, `Ok((a, b))`
- [ ] Multiple match arms work
- [ ] Wildcard and variable patterns work: `_`, `x`
- [ ] All stdlib modules parse without errors
- [ ] Test suite has 100% passing tests

---

## Timeline

**Estimated**: 1-2 days total

**Day 1 (Investigation + Fix)**:
- Morning: Trace parsing flow, identify root cause (4 hours)
- Afternoon: Implement fix, verify simple cases (4 hours)

**Day 2 (Testing + Polish)**:
- Morning: Create comprehensive test suite (3 hours)
- Afternoon: Test stdlib modules, fix edge cases (3 hours)
- Evening: Verify all acceptance criteria (2 hours)

---

## Impact

**Unblocks**:
- ✅ Stdlib implementation in AILANG (.ail files)
- ✅ Dogfooding the language (using AILANG to write AILANG code)
- ✅ Pattern matching best practices demonstration
- ✅ Full functional programming idioms

**Benefits**:
- Users can write pattern matching in all contexts
- Stdlib becomes a reference implementation
- Language consistency (no "works here but not there")
- v0.1.0 ships with usable stdlib

---

## Resources

**Parser Code**:
- `internal/parser/parser.go` (main parser)
- `internal/parser/parser.go:968` - `parseMatchExpression()`
- `internal/parser/parser.go:1445` - `parsePattern()`
- `internal/parser/parser.go:512` - `parseFunctionDeclaration()`

**Test Files**:
- `examples/adt_simple.ail` - Working pattern matching (top-level)
- `std_list.ail` - Blocked stdlib (patterns in functions)
- Create: `internal/parser/func_pattern_test.go` - New test suite

**References**:
- M-P3 implementation (pattern matching evaluation - COMPLETE)
- ADT runtime (proven to work with patterns)
- Parser precedence table (ensure MATCH has correct precedence)

---

## Next Developer

**Context**: You're inheriting a partially working parser. Pattern matching EXISTS and WORKS, but only in top-level contexts. Your job is to make it work everywhere.

**Start Here**:
1. Read this document completely
2. Run `examples/adt_simple.ail` - see pattern matching work
3. Try to run `std_list.ail` - see it fail
4. Add debug logging to `parseMatchExpression()`
5. Compare parser state in both cases
6. Fix the context issue
7. Create test suite
8. Celebrate! 🎉

**Questions?** Check commit history for "M-P3" to see how pattern matching was originally implemented.

---

**Good luck! You've got this.** 💪
