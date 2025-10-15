# Add `letrec` Syntax for Recursive Lambdas

**Status**: 📋 PLANNED
**Priority**: P2 (High - UX improvement)
**Estimated**: ~200 LOC (100 impl + 50 tests + 50 docs)
**Duration**: 1-2 hours
**Target**: v0.3.5 or v0.4.0

## Problem Statement

**User Pain Point**: Recursive lambdas don't work in the REPL or in let-bindings.

```ailang
-- ❌ CURRENT: This fails with "undefined variable: fib"
λ> let fib = \n. if n < 2 then n else fib(n - 1) + fib(n - 2) in fib(10)
Type error: undefined variable: fib at <repl>:1:40

-- ✅ WORKAROUND: Must use module file with func declaration
-- examples/fib.ail
export func fib(n: int) -> int {
  if n < 2 then n else fib(n - 1) + fib(n - 2)
}
```

**Root Cause**:
- `core.LetRec` exists and works internally (used for module func declarations)
- Parser does NOT recognize `letrec` keyword in surface syntax
- `let` bindings evaluate RHS before making name available (non-recursive by design)

**Impact**:
- Poor REPL experience - can't experiment with recursive algorithms interactively
- Users confused why `let fib = \n. ... fib ...` doesn't work (natural expectation)
- Gap between module syntax (recursive by default) and expression syntax (not recursive)

## Goals

### Primary Goals (Must Achieve)
1. **`letrec` keyword works**: Parse `letrec name = expr in body` syntax
2. **REPL support**: Can write recursive lambdas in REPL
3. **Type inference**: Same as `let` - full HM inference
4. **Elaboration**: Lower to existing `core.LetRec` (already implemented!)

### Secondary Goals (Nice to Have)
5. **Mutual recursion**: `letrec f = ... and g = ... in body` (can defer)
6. **Documentation**: Update prompt, examples, playground docs

## Design

### Syntax

```ailang
-- Single recursive binding
letrec name = expr in body

-- Examples:
letrec fib = \n. if n < 2 then n else fib(n - 1) + fib(n - 2) in fib(10)

letrec factorial = \n. if n <= 1 then 1 else n * factorial(n - 1) in factorial(5)

-- Can still have type annotations
letrec sum: [int] -> int = \xs. match xs {
  [] => 0,
  [x, ...rest] => x + sum(rest)
} in sum([1, 2, 3, 4, 5])
```

### Implementation Plan

**Phase 1: Lexer (15 minutes, ~20 LOC)**

File: `internal/lexer/lexer.go`, `internal/lexer/token.go`

1. Add `LETREC` token type
2. Add `"letrec"` to keywords map
3. Test: `letrec` is recognized as LETREC token

**Phase 2: Parser (30 minutes, ~50 LOC)**

File: `internal/parser/parser.go`

1. Add `ast.LetRec` node to surface AST:
   ```go
   type LetRec struct {
       Name  string
       Type  Type      // Optional type annotation
       Value Expr
       Body  Expr
       Pos   Pos
   }
   ```

2. Parse `letrec` in `parseExpression()`:
   ```go
   case token.LETREC:
       return p.parseLetRec()
   ```

3. Implement `parseLetRec()`:
   ```go
   func (p *Parser) parseLetRec() ast.Expr {
       pos := p.curToken.Pos
       p.nextToken() // consume 'letrec'

       name := p.curToken.Literal
       p.nextToken() // consume name

       // Optional type annotation
       var typeAnnot ast.Type
       if p.curTokenIs(token.COLON) {
           typeAnnot = p.parseTypeAnnotation()
       }

       p.expectPeek(token.ASSIGN) // '='
       value := p.parseExpression(LOWEST)
       p.expectPeek(token.IN)     // 'in'
       body := p.parseExpression(LOWEST)

       return &ast.LetRec{
           Name:  name,
           Type:  typeAnnot,
           Value: value,
           Body:  body,
           Pos:   pos,
       }
   }
   ```

**Phase 3: Type Checking (10 minutes, ~20 LOC)**

File: `internal/types/typechecker.go`

Add case for `*ast.LetRec`:
```go
case *ast.LetRec:
    // Same logic as existing LetRec handling in elaborate
    // 1. Create recursive environment
    // 2. Infer type of value in recursive env
    // 3. Check body in extended env
    return tc.inferLetRec(expr.Name, expr.Type, expr.Value, expr.Body)
```

**Phase 4: Elaboration (15 minutes, ~20 LOC)**

File: `internal/elaborate/elaborate.go`

Add case for `*ast.LetRec`:
```go
case *ast.LetRec:
    // Elaborate value in recursive environment
    coreValue := e.elaborate(expr.Value)

    // Elaborate body
    coreBody := e.elaborate(expr.Body)

    return &core.LetRec{
        Bindings: []core.RecBinding{
            {Name: expr.Name, Value: coreValue},
        },
        Body: coreBody,
    }
```

**Phase 5: Testing (30 minutes, ~50 LOC)**

File: `internal/parser/parser_test.go`, `internal/eval/eval_test.go`

Tests:
1. Lexer recognizes `letrec` keyword
2. Parser handles `letrec name = expr in body`
3. Parser handles type annotations: `letrec f: int -> int = ...`
4. Type inference works correctly
5. Evaluation produces correct results:
   - Fibonacci: `letrec fib = \n. ... in fib(10)` => 55
   - Factorial: `letrec fac = \n. ... in fac(5)` => 120
   - List sum: recursive list processing
6. REPL integration test

**Phase 6: Documentation (20 minutes, ~50 LOC)**

Files: `prompts/v0.3.0.md`, `docs/docs/playground.mdx`, `examples/`

1. Update teaching prompt with `letrec` examples
2. Update playground docs with working recursive lambda
3. Create `examples/letrec_examples.ail` demonstrating usage
4. Update README.md feature list

## Acceptance Criteria

- [ ] `letrec fib = \n. if n < 2 then n else fib(n - 1) + fib(n - 2) in fib(10)` works in REPL
- [ ] Type annotations work: `letrec sum: [int] -> int = ...`
- [ ] All existing tests still pass
- [ ] New tests for letrec parsing, typing, evaluation
- [ ] Documentation updated (prompt, playground, examples)
- [ ] Example file demonstrating letrec patterns

## Implementation Notes

**Why this is easy:**
- `core.LetRec` already exists and works perfectly (used for module funcs)
- Elaboration phase already knows how to handle recursive bindings
- Evaluator already implements RefCell-based recursion
- Just need to expose existing functionality via surface syntax

**Edge Cases:**
- Non-lambda values: `letrec x = x + 1 in x` (should error - infinite loop in eval)
- Type annotations: Should work same as `let`
- Shadowing: `letrec` bindings shadow outer scope as expected

**Future Extensions (deferred):**
- Mutual recursion: `letrec f = ... and g = ... in body`
- Multiple bindings: `letrec f = ..., g = ... in body`

## Estimated LOC

| Component | Implementation | Tests | Total |
|-----------|---------------|-------|-------|
| Lexer | 10 | 5 | 15 |
| Parser | 50 | 20 | 70 |
| Type Checker | 20 | 10 | 30 |
| Elaboration | 20 | 10 | 30 |
| Documentation | 50 | 0 | 50 |
| **Total** | **150** | **45** | **195** |

## Dependencies

None - all infrastructure exists.

## Risks

**Low Risk**:
- Core implementation already exists
- Just adding surface syntax
- No semantic changes needed

## Related Work

- [M-R4_recursion.md](../implemented/v0_3_0/M-R4_recursion.md) - Core recursion implementation
- Standard ML, OCaml, Haskell all have `let rec` / `letrec`
- This brings AILANG closer to ML-family syntax expectations

## Success Metrics

- REPL can run recursive lambdas without module files
- User satisfaction: No more "why doesn't my fib lambda work?" questions
- Example count: 2-3 new examples showing letrec patterns

---

**Reported by**: User feedback (2025-10-13)
**Motivation**: Poor REPL UX, documentation showed broken code, natural user expectation
