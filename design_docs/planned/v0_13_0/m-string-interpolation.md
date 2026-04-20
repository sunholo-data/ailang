# M-STRING-INTERP: String Interpolation Syntax

**Status**: Planned
**Target**: v0.7.0+
**Priority**: P3 (Low) - Convenience feature, workaround exists
**Estimated Time**: 6-10 hours
**Dependencies**: None

## Problem Statement

AILANG requires explicit concatenation for building strings with embedded values:

```ailang
-- Current (verbose):
let msg = "Hello, " ++ name ++ "! You have " ++ show(count) ++ " messages."

-- Desired:
let msg = "Hello, ${name}! You have ${count} messages."
```

## Design Considerations

### AI-First Alignment

| Principle | Impact | Notes |
|-----------|--------|-------|
| Reduce Syntactic Noise | +1 | Fewer `++` operators, cleaner strings |
| Preserve Semantic Clarity | 0 | Must be clear when interpolation happens |
| Increase Determinism | 0 | Desugars to `++`, same semantics |
| Lower Token Cost | +1 | Shorter syntax for same meaning |

**Decision**: Move forward if implementation is clean

### Syntax Options

**Option A: `${expr}` (JavaScript-style)**
```ailang
"Hello, ${name}!"
```
- Pros: Familiar to most developers, clear expression boundary
- Cons: Requires special lexer handling for `$`

**Option B: `{expr}` (Python f-string style)**
```ailang
f"Hello, {name}!"
```
- Pros: Simple, minimal syntax
- Cons: Needs prefix marker, harder to distinguish from record syntax

**Option C: `#{expr}` (Ruby-style)**
```ailang
"Hello, #{name}!"
```
- Pros: Distinct from other syntax, no conflict with records
- Cons: Less common

**Recommendation**: Option A (`${expr}`) for familiarity.

### Desugaring

```ailang
-- Source:
"Value: ${x}, Sum: ${a + b}"

-- Desugars to:
"Value: " ++ show(x) ++ ", Sum: " ++ show(a + b)
```

### Type Requirements

- Interpolated expressions must have a `Show` instance
- Compiler inserts `show()` calls automatically
- Type errors if no Show instance available

## Implementation Steps

1. **Lexer** (`internal/lexer/`):
   - Detect `${` inside string literals
   - Emit tokens: STRING_PART, INTERP_START, expr tokens, INTERP_END, STRING_PART
   - Handle nested `{}` in expressions

2. **Parser** (`internal/parser/`):
   - Parse interpolated string as a sequence of parts
   - Build concatenation AST

3. **Type Checking**:
   - Verify each interpolated expression has Show instance
   - Insert show() calls in elaboration

4. **Strict Mode**:
   - `--strict-syntax` rejects interpolation, requires explicit `++`

## Workaround (Current)

```ailang
-- Use explicit concatenation:
let msg = "Value: " ++ show(x) ++ ", Sum: " ++ show(a + b)
```

## Success Criteria

1. `"Hello, ${name}!"` compiles and runs correctly
2. Nested expressions work: `"Result: ${compute(a, b)}"`
3. Type errors for non-Show types in interpolation
4. `--strict-syntax` rejects interpolation

## References

- [Limitations doc](/docs/reference/limitations#string-interpolation)

---

## Website Links

**Update these when this feature is implemented:**
- [Limitations page](/docs/reference/limitations) — Remove from limitations list
- [Implementation Status](/docs/reference/implementation-status) — Update status
- Move this doc from `planned/` to `implemented/`
