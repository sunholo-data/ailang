# M-CALL-SUGAR: Optional Parenthesized Call Syntax

**Status:** Planned (v0.5.x candidate)
**Type:** Syntactic Sugar (Optional)
**Priority:** Low (ergonomics improvement)
**Effort:** ~8-16 hours

---

## Overview

Add **optional** syntactic sugar for C-style function call syntax `f()` and `f(x, y)`, desugaring to AILANG's canonical ML-style application `f ()` and `f x y`.

**Key principle:** This is parser-level rewriting only. The Core language remains unchanged (curried, single-argument application).

---

## Current State (v0.4.0)

AILANG uses **ML-style call syntax** (juxtaposition):
- `f x` - apply f to x
- `f x y` - apply f to x, then result to y
- `f ()` - apply f to unit value (zero-arg functions)
- `f()` - **PARSE ERROR** (parentheses have no meaning without space)

**Why this is by design:**
1. **Functional purity:** Application is by juxtaposition, not special syntax
2. **Unambiguous:** Parentheses always group expressions, never mark calls
3. **Deterministic parsing:** No context-sensitive rules for `()`
4. **AI-friendly:** Clear semantics with no dual interpretations

**Status:** Documented as intentional design in [prompts/v0.3.24.md](../../prompts/v0.3.24.md).

---

## Problem

Users coming from C/Python/JavaScript expect `f()` and `f(x, y)` syntax and are confused by parse errors.

**Evidence:**
- AI models frequently generate `f()` syntax despite teaching prompt warnings
- User questions: "Why doesn't `getMessage()` work?"
- Requires constant correction in evals and code reviews

**Impact:**
- Minor DX friction for users familiar with C-style syntax
- Teaching overhead (must explain ML vs C syntax)
- AI code generation errors (even with prompt warnings)

---

## Proposed Solution

### Phase 1: Parser-Level Desugaring

Add **optional** syntactic sugar that desugars during parsing:

| Sugar Syntax | Desugars To | Meaning |
|--------------|-------------|---------|
| `f()` | `f ()` | Apply f to unit |
| `f(x)` | `f x` | Apply f to x |
| `f(x, y)` | `f x y` | Apply f to x, then to y |
| `f(x, y, z)` | `f x y z` | Curried application |

**Crucially:** This is a **parser-only** rewrite. By the time elaboration/type-checking happens, the AST contains canonical forms (`f ()`, `f x y`).

### Phase 2: Opt-In Mechanism

Sugar is **off by default** to maintain backward compatibility and language purity. Users opt in via:

**Option A: Command-line flag**
```bash
ailang run --syntax sugar --entry main module.ail
ailang repl --syntax sugar
```

**Option B: Pragma in file**
```ailang
#pragma syntax(sugar)

module mymodule

export func main() -> () ! {IO} {
    println(getMessage())  -- ✅ Works with pragma
}
```

**Option C: Profile/config file**
```json
// .ailang/config.json
{
  "syntax": {
    "call_sugar": true
  }
}
```

**Recommendation:** Start with Option A (flag), add Option B (pragma) in v0.5.1+.

---

## Implementation Plan

### Step 1: Tokenizer Changes (~1-2h)

No tokenizer changes needed! `(`, `)`, and `,` already tokenized.

### Step 2: Parser Rewriting Layer (~4-6h)

**File:** `internal/parser/parser_expr.go`

Add sugar-aware call parsing:

```go
func (p *Parser) parseCallExpression(fn ast.Expr) ast.Expr {
    // Check if sugar is enabled
    if !p.opts.CallSugar {
        // Existing behavior: LPAREN starts argument list (space required)
        return p.parseCallExpressionCanonical(fn)
    }

    // Sugar mode: f() or f(args)
    if p.peekTokenIs(lexer.LPAREN) {
        p.nextToken() // consume LPAREN

        // Empty parens: f() → f ()
        if p.peekTokenIs(lexer.RPAREN) {
            p.nextToken() // consume RPAREN
            return &ast.FuncCall{
                Func: fn,
                Args: []ast.Expr{&ast.Literal{Kind: ast.UnitLit}},
                Pos:  fn.Position(),
            }
        }

        // Parse comma-separated args: f(x, y) → f x y
        args := p.parseCallArgumentsSugar()

        // Desugar to nested applications
        return p.desugarCurriedCall(fn, args)
    }

    // Not a parenthesized call, use canonical syntax
    return p.parseCallExpressionCanonical(fn)
}

func (p *Parser) parseCallArgumentsSugar() []ast.Expr {
    args := []ast.Expr{}

    args = append(args, p.parseExpression(LOWEST))

    for p.peekTokenIs(lexer.COMMA) {
        p.nextToken() // consume COMMA
        p.nextToken() // move to next expression
        args = append(args, p.parseExpression(LOWEST))
    }

    p.expectPeek(lexer.RPAREN)
    return args
}

func (p *Parser) desugarCurriedCall(fn ast.Expr, args []ast.Expr) ast.Expr {
    // f(x, y, z) → ((f x) y) z
    result := fn
    for _, arg := range args {
        result = &ast.FuncCall{
            Func: result,
            Args: []ast.Expr{arg},
            Pos:  fn.Position(),
        }
    }
    return result
}
```

### Step 3: Add --syntax Flag (~1-2h)

**File:** `cmd/ailang/run.go`

```go
var syntaxProfile string

func init() {
    runCmd.Flags().StringVar(&syntaxProfile, "syntax", "canonical",
        "Syntax profile: canonical (default) | sugar")
}

func runCommand(cmd *cobra.Command, args []string) error {
    // ...
    parserOpts := parser.Options{
        CallSugar: syntaxProfile == "sugar",
    }
    p := parser.NewWithOptions(l, parserOpts)
    // ...
}
```

**File:** `internal/parser/parser.go`

```go
type Options struct {
    CallSugar bool  // Enable f() and f(x,y) syntax sugar
}

func NewWithOptions(l *lexer.Lexer, opts Options) *Parser {
    p := &Parser{
        l:    l,
        opts: opts,
        // ...
    }
    return p
}
```

### Step 4: Testing (~2-4h)

**File:** `internal/parser/parser_call_sugar_test.go`

```go
func TestCallSugar_ZeroArgs(t *testing.T) {
    input := `getMessage()`

    // Without sugar: parse error
    p := parser.New(lexer.New(input, "test.ail"))
    _, err := p.ParseExpression()
    assert.Error(t, err)

    // With sugar: desugars to getMessage ()
    p = parser.NewWithOptions(lexer.New(input, "test.ail"),
        parser.Options{CallSugar: true})
    expr, err := p.ParseExpression()
    assert.NoError(t, err)

    call, ok := expr.(*ast.FuncCall)
    assert.True(t, ok)
    assert.Equal(t, "getMessage", call.Func.(*ast.Variable).Name)
    assert.Len(t, call.Args, 1)
    assert.IsType(t, &ast.Literal{}, call.Args[0])
    assert.Equal(t, ast.UnitLit, call.Args[0].(*ast.Literal).Kind)
}

func TestCallSugar_MultiArgs(t *testing.T) {
    input := `add(1, 2)`

    p := parser.NewWithOptions(lexer.New(input, "test.ail"),
        parser.Options{CallSugar: true})
    expr, err := p.ParseExpression()
    assert.NoError(t, err)

    // Should desugar to: (add 1) 2
    call2, ok := expr.(*ast.FuncCall)
    assert.True(t, ok)
    assert.Equal(t, "2", call2.Args[0].(*ast.Literal).Value)

    call1, ok := call2.Func.(*ast.FuncCall)
    assert.True(t, ok)
    assert.Equal(t, "1", call1.Args[0].(*ast.Literal).Value)
    assert.Equal(t, "add", call1.Func.(*ast.Variable).Name)
}

func TestCallSugar_NestedCalls(t *testing.T) {
    input := `outer(inner(x))`

    p := parser.NewWithOptions(lexer.New(input, "test.ail"),
        parser.Options{CallSugar: true})
    expr, err := p.ParseExpression()
    assert.NoError(t, err)

    // Should desugar to: outer (inner x)
    // Verify structure matches
}
```

### Step 5: Documentation (~1-2h)

**Update files:**
1. **prompts/v0.5.0.md**
   - Remove "parse error" warnings for `f()`
   - Add note: "Sugar available with `--syntax sugar`"

2. **docs/guides/syntax.md**
   - Document sugar as optional extension
   - Show canonical vs sugar forms side-by-side

3. **CHANGELOG.md**
   ```markdown
   ## [v0.5.0] - TBD

   ### Added
   - Optional call syntax sugar: `f()` and `f(x, y)` (opt-in with `--syntax sugar`)
   - Parser desugars to canonical ML-style application
   ```

---

## Design Principles

### 1. **Sugar is Optional**
   - Default behavior unchanged (canonical ML syntax)
   - Users opt in explicitly (`--syntax sugar`)
   - No surprise behavior changes

### 2. **Parser-Only Rewrite**
   - Sugar desugars during parsing
   - Core AST unchanged (always canonical form)
   - Type checker sees only `f ()` and `f x y`
   - No changes to elaboration, type checking, or evaluation

### 3. **Preserve Language Purity**
   - Core language semantics unchanged
   - Sugar is "lossy" - AST can't round-trip to sugared form
   - Teaching materials still emphasize canonical syntax

### 4. **Gradual Adoption**
   - v0.5.0: `--syntax sugar` flag only
   - v0.5.1+: Add `#pragma syntax(sugar)` for per-file opt-in
   - v0.6.0: Consider making sugar default (breaking change)

---

## Alternatives Considered

### Alternative 1: Make Sugar Default
**Rejected:** Would require breaking change announcement and migration period. Better to start opt-in.

### Alternative 2: Add Separate `call` Construct
**Rejected:** Would complicate Core language. Sugar is parser-level only.

### Alternative 3: Support Mixed Syntax
**Rejected:** Would create two "dialects" - confusing for users and AIs. Pick one mode per file.

### Alternative 4: Never Add Sugar
**Considered:** Keeps language pure, but increases DX friction for C/Python users. Sugar is a reasonable compromise if opt-in.

---

## Success Criteria

- ✅ `f()` parses correctly with `--syntax sugar`
- ✅ `f(x, y)` desugars to curried application
- ✅ Nested calls work: `outer(inner(x))`
- ✅ Sugar disabled by default (canonical mode)
- ✅ Core AST unchanged (type checker unaware of sugar)
- ✅ All existing tests pass with canonical mode
- ✅ New sugar tests pass with `--syntax sugar`

---

## Non-Goals

- **NOT changing** Core language semantics
- **NOT making** sugar mandatory
- **NOT supporting** partial application sugar (`f(x, _)`)
- **NOT adding** variadic functions
- **NOT changing** how type inference works

---

## Migration / Breaking Changes

**None.** Sugar is opt-in. Existing code continues to work unchanged.

Users can migrate gradually:
1. Try `--syntax sugar` on new code
2. If it works well, adopt more widely
3. Consider making default in v0.6.0+

---

## Open Questions

1. **Should sugar be per-file or per-project?**
   - Per-project: `--syntax sugar` flag (easier, but less granular)
   - Per-file: `#pragma syntax(sugar)` (more control, but more boilerplate)
   - **Recommendation:** Start with per-project (flag), add per-file in v0.5.1

2. **Should teaching prompts show sugar or canonical?**
   - Show canonical first (pure semantics)
   - Mention sugar as optional convenience
   - AIs should default to canonical unless user requests sugar

3. **What about partial application sugar?**
   - e.g., `map(double, _)` for `\xs. map double xs`
   - **Deferred:** Not in scope for v0.5.0. Evaluate demand first.

---

## Related Issues

- User question: "Why doesn't `f()` work?"
- Documented as by-design in: [prompts/v0.3.24.md](../../prompts/v0.3.24.md)
- Related to: [M-PARSER](m-parser-nested-match-delimiter-fix.md) (parser improvements)

---

## Implementation Checklist

- [ ] **Phase 1:** Parser desugaring (~4-6h)
  - [ ] Add `Options.CallSugar` flag
  - [ ] Implement `parseCallArgumentsSugar()`
  - [ ] Implement `desugarCurriedCall()`
  - [ ] Handle zero-arg case: `f()` → `f ()`
  - [ ] Handle multi-arg case: `f(x,y)` → `(f x) y`
- [ ] **Phase 2:** CLI integration (~1-2h)
  - [ ] Add `--syntax` flag to `ailang run`
  - [ ] Add `--syntax` flag to `ailang repl`
  - [ ] Add `--syntax` flag to `ailang check`
- [ ] **Phase 3:** Testing (~2-4h)
  - [ ] Write `parser_call_sugar_test.go`
  - [ ] Test zero-arg calls
  - [ ] Test multi-arg calls
  - [ ] Test nested calls
  - [ ] Test mixed canonical/sugar (should fail)
  - [ ] Run `make test` - ensure all pass
- [ ] **Phase 4:** Documentation (~1-2h)
  - [ ] Update teaching prompts (mention sugar as optional)
  - [ ] Update syntax guide
  - [ ] Add CHANGELOG entry
  - [ ] Move design doc to `implemented/v0_5_0/`

---

## Timeline

**Target:** v0.5.x (after v0.4.1 parser fixes)

**Rationale:** Sugar should wait until parser delimiter tracking is solid (after M-PARSER-NESTED-MATCH-FIX).

---

## Notes

This is **syntactic sugar**, not a semantic change. The Core language remains pure ML-style application.

Users who prefer canonical syntax can ignore this feature entirely. Those who want C-style ergonomics can opt in with a flag.

**Guiding principle:** Make AILANG easier to use without compromising semantic purity.
