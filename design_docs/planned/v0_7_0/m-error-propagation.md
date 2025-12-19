# M-ERROR-PROP: Error Propagation Operator (`?`)

**Status**: Planned
**Target**: v0.7.0+
**Priority**: P2 (Medium) - Important for ergonomic error handling
**Estimated Time**: 8-12 hours
**Dependencies**: Result type must be well-established

## Problem Statement

Error handling with `Result` types requires verbose pattern matching:

```ailang
-- Current (verbose):
func processFile(path: string) -> Result[string, Error] ! {FS} =
  match readFile(path) {
    Ok(content) => match parseContent(content) {
      Ok(data) => match transform(data) {
        Ok(result) => Ok(result),
        Err(e) => Err(e)
      },
      Err(e) => Err(e)
    },
    Err(e) => Err(e)
  }

-- Desired:
func processFile(path: string) -> Result[string, Error] ! {FS} =
  let content = readFile(path)? in
  let data = parseContent(content)? in
  let result = transform(data)? in
  Ok(result)
```

## Design

### Syntax

The `?` operator is postfix and works on `Result[T, E]` expressions:

```ailang
expr?
```

**Semantics**:
- If `expr` evaluates to `Ok(v)`, the `?` expression evaluates to `v`
- If `expr` evaluates to `Err(e)`, early-return `Err(e)` from the enclosing function

### Type Rules

```
expr : Result[T, E]
----------------------
expr? : T

-- Constraint: enclosing function must return Result[_, E]
```

### Desugaring

```ailang
let x = foo()? in body
```

Desugars to:
```ailang
match foo() {
  Ok(x) => body,
  Err(e) => Err(e)
}
```

### AI-First Alignment

| Principle | Impact | Notes |
|-----------|--------|-------|
| Reduce Syntactic Noise | +2 | Dramatically reduces boilerplate |
| Preserve Semantic Clarity | +1 | Clear early-return semantics |
| Increase Determinism | 0 | Same behavior as explicit match |
| Lower Token Cost | +2 | Much shorter code for same logic |

**Decision**: Strong positive, move forward

## Implementation Steps

1. **Lexer** (`internal/lexer/`):
   - Add `QUESTION` token for `?`
   - Handle as postfix operator

2. **Parser** (`internal/parser/`):
   - Parse `?` as postfix operator with high precedence
   - Create `TryExpr` AST node

3. **Type Checking** (`internal/types/`):
   - `TryExpr` requires operand to be `Result[T, E]`
   - Expression type is `T`
   - Check enclosing function returns compatible `Result[_, E]`

4. **Elaboration** (`internal/elaborate/`):
   - Desugar `TryExpr` to match expression
   - Handle nested `?` operators

5. **Effect Tracking**:
   - The `?` operator doesn't introduce new effects
   - The underlying Result-returning function may have effects

## Edge Cases

### Nested `?`

```ailang
let x = foo()?.bar()? in ...
```

Desugars to nested matches (right-to-left).

### `?` in Non-Result Context

```ailang
func main() -> int =
  let x = foo()? in  -- Error: main returns int, not Result
  x
```

Compile-time error: `?` requires enclosing function to return Result.

### Multiple Error Types

```ailang
func process() -> Result[int, Error] =
  let a = foo()? in   -- Returns Result[_, FooError]
  let b = bar()? in   -- Returns Result[_, BarError]
  Ok(a + b)
```

Options:
- Require same error type (simple, strict)
- Allow if error types are compatible (subtype/Into relationship)
- Use unified Error enum

**Recommendation**: Start simple (same error type), extend later.

## Workaround (Current)

```ailang
-- Use explicit match:
match readFile(path) {
  Ok(content) => process(content),
  Err(e) => Err(e)
}

-- Or helper function:
func andThen[A, B, E](r: Result[A, E], f: A -> Result[B, E]) -> Result[B, E] =
  match r {
    Ok(a) => f(a),
    Err(e) => Err(e)
  }

readFile(path) |> andThen(\c. parseContent(c)) |> andThen(\d. transform(d))
```

## Success Criteria

1. `foo()?` syntax parses and type-checks
2. Early return works correctly at runtime
3. Type errors for `?` in non-Result functions
4. Chained `?` works: `a()?.b()?.c()`

## References

- [Limitations doc](/docs/reference/limitations#error-propagation-operator-)
- Rust's `?` operator: https://doc.rust-lang.org/book/ch09-02-recoverable-errors-with-result.html

---

## Website Links

**Update these when this feature is implemented:**
- [Limitations page](/docs/reference/limitations) — Remove from limitations list
- [Implementation Status](/docs/reference/implementation-status) — Update status
- Move this doc from `planned/` to `implemented/`
