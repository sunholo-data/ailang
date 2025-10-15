# List Pattern Spread Syntax

**Status**: 📋 PLANNED
**Priority**: P1 (MEDIUM - Ergonomics)
**Estimated Effort**: ~200 LOC, 6-8 hours
**Target Release**: v0.4.0 or v0.4.1
**Created**: 2025-10-13

---

## Problem Statement

List patterns currently require verbose Cons constructor syntax. AI models and users expect JavaScript/Python-style spread syntax (`...`), but it doesn't parse.

### Current Behavior

```ailang
-- ❌ DOESN'T WORK: Spread syntax in patterns
match xs {
    [] => 0,
    [x, ...rest] => x + sum(rest)
}
-- Parser error: "unexpected token: ..."

-- ✅ WORKAROUND: Use Cons constructor (verbose!)
match xs {
    [] => 0,
    Cons(x, rest) => x + sum(rest)
}
```

### Why This Matters

1. **AI Expectations**: Models generate `...` syntax naturally
   - GPT/Claude output: `[x, ...rest]` (doesn't parse)
   - Breaks M-EVAL benchmarks
   - Requires manual rewriting

2. **Ergonomics**: Spread is more intuitive
   - Matches JavaScript, Python, Rust syntax
   - Clearer intent than Cons constructor
   - Reduces cognitive load

3. **Consistency**: Lists use `[]` syntax everywhere else
   - Literal: `[1, 2, 3]` ✅
   - Pattern: `[1, 2, 3]` ✅
   - Pattern spread: `[x, ...rest]` ❌

---

## Proposed Solution

Support spread syntax (`...`) in list patterns:

```ailang
-- Head/tail destructuring
match xs {
    [] => "empty",
    [x, ...rest] => "head: " ++ show(x)
}

-- Multiple elements + rest
match xs {
    [x, y, ...rest] => x + y,
    _ => 0
}

-- Init/last destructuring (advanced)
match xs {
    [...init, last] => last,
    [] => 0
}

-- Just spread (match any non-empty list)
match xs {
    [...items] => length(items)
}
```

---

## Design

### Syntax

```ebnf
ListPattern ::=
    | "[]"                                    -- Empty list
    | "[" Pattern ("," Pattern)* "]"          -- Fixed-length list
    | "[" Pattern ("," Pattern)* "," "..." IDENT "]"   -- Head + spread
    | "[" "..." IDENT "," Pattern "]"         -- Spread + last (advanced)
    | "[" "..." IDENT "]"                     -- Just spread (any list)

Restrictions:
- At most ONE spread per pattern
- Spread must be at END: [x, y, ...rest] ✅
- Spread at START: [...init, x] (defer to v0.5.0+)
- Spread in MIDDLE: [x, ...mid, y] ❌ (too complex)
```

### Lexer Changes

**File**: `internal/lexer/token.go`

`ELLIPSIS` token already exists! Just need to handle it in list patterns.

```go
// Already defined:
ELLIPSIS  // ...
```

### Parser Changes

**File**: `internal/parser/parser.go`

Current parsing (fixed patterns only):
```go
func (p *Parser) parseListPattern() ast.Pattern {
    // Parses: [p1, p2, p3]
    // Returns: ListPattern{Elements: [p1, p2, p3]}
}
```

Proposed parsing (support spread):
```go
func (p *Parser) parseListPattern() ast.Pattern {
    patterns := []ast.Pattern{}
    var spreadName string
    var spreadPos ast.Pos

    for !p.curTokenIs(lexer.RBRACKET) {
        if p.curTokenIs(lexer.ELLIPSIS) {
            // Found spread: ...rest
            if spreadName != "" {
                p.errors = append(p.errors, "only one spread allowed per pattern")
                return nil
            }
            spreadPos = p.curPos()
            p.nextToken() // move to IDENT
            spreadName = p.curToken.Literal
            p.nextToken() // move past IDENT

            // Spread must be last element
            if !p.curTokenIs(lexer.RBRACKET) && !p.curTokenIs(lexer.COMMA) {
                p.errors = append(p.errors, "spread must be last element")
                return nil
            }
            break
        }

        patterns = append(patterns, p.parsePattern())

        if !p.peekTokenIs(lexer.RBRACKET) {
            p.expectPeek(lexer.COMMA)
            p.nextToken()
        }
    }

    if spreadName != "" {
        // Return ListPatternSpread
        return &ast.ListPatternSpread{
            Head: patterns,
            Rest: spreadName,
            Pos:  spreadPos,
        }
    } else {
        // Return ListPattern (existing)
        return &ast.ListPattern{
            Elements: patterns,
            Pos:      p.curPos(),
        }
    }
}
```

### AST Changes

**File**: `internal/ast/ast.go`

Add new pattern type:
```go
// ListPatternSpread represents a list pattern with spread
// Syntax: [p1, p2, ...rest]
type ListPatternSpread struct {
    Head []Pattern  // Patterns before spread
    Rest string     // Variable name for rest
    Pos  Pos
}

func (l *ListPatternSpread) String() string {
    heads := []string{}
    for _, h := range l.Head {
        heads = append(heads, h.String())
    }
    if len(heads) > 0 {
        return fmt.Sprintf("[%s, ...%s]", strings.Join(heads, ", "), l.Rest)
    }
    return fmt.Sprintf("[...%s]", l.Rest)
}
func (l *ListPatternSpread) Position() Pos { return l.Pos }
func (l *ListPatternSpread) patternNode()  {}
```

### Elaboration Changes

**File**: `internal/elaborate/elaborate.go`

Desugar spread patterns to Cons patterns:

```go
// Example transformation:
// [x, y, ...rest]
//   → Cons(x, Cons(y, rest))

func (e *Elaborator) elaboratePattern(p ast.Pattern) (core.Pattern, error) {
    switch pat := p.(type) {
    case *ast.ListPatternSpread:
        // Build nested Cons pattern from right to left
        // [x, y, ...rest] → Cons(x, Cons(y, rest))

        if len(pat.Head) == 0 {
            // [...rest] → just bind to rest (VarPattern)
            return &core.VarPattern{Name: pat.Rest}, nil
        }

        // Start with rest as tail
        result := &core.VarPattern{Name: pat.Rest}

        // Build Cons chain from right to left
        for i := len(pat.Head) - 1; i >= 0; i-- {
            headPat, err := e.elaboratePattern(pat.Head[i])
            if err != nil {
                return nil, err
            }

            result = &core.ConstructorPattern{
                TypeName:    "List",
                Constructor: "Cons",
                Fields:      []core.Pattern{headPat, result},
            }
        }

        return result, nil

    // ... existing cases ...
    }
}
```

**Example Elaboration**:
```
Input:  [x, y, ...rest]
Output: Cons(x, Cons(y, rest))

Input:  [x, ...rest]
Output: Cons(x, rest)

Input:  [...xs]
Output: xs  (just a variable binding)
```

### Type Checking Changes

**File**: `internal/types/patterns.go`

**Minimal changes needed**: Spread patterns elaborate to Cons patterns, which are already type-checked correctly.

Potential validation:
- Ensure `rest` variable isn't already bound
- Ensure pattern context expects a list type

---

## Implementation Plan

### Phase 1: Parser & AST (3-4 hours, ~100 LOC)

1. Add `ListPatternSpread` AST node
2. Modify `parseListPattern()` to handle `...`
3. Add validation (only one spread, must be last)

**Files Modified**:
- `internal/ast/ast.go` (~30 LOC)
- `internal/parser/parser.go` (~50 LOC)
- `internal/parser/parser_test.go` (~20 LOC)

### Phase 2: Elaboration (2-3 hours, ~50 LOC)

1. Implement `elaboratePattern` case for `ListPatternSpread`
2. Generate nested Cons patterns
3. Test elaboration output

**Files Modified**:
- `internal/elaborate/elaborate.go` (~40 LOC)
- `internal/elaborate/elaborate_test.go` (~10 LOC)

### Phase 3: Testing & Examples (1-2 hours, ~50 LOC)

1. Add pattern matching tests
2. Create example file with recursive functions
3. Update teaching prompt

**Files Modified/Created**:
- `internal/elaborate/elaborate_test.go` (~30 LOC)
- `examples/list_patterns.ail` (~15 LOC)
- `prompts/v0.3.0.md` (~5 LOC)

---

## Examples

### Before (Verbose Cons)

```ailang
-- Sum list recursively
func sum(xs: [int]) -> int =
    match xs {
        [] => 0,
        Cons(x, rest) => x + sum(rest)
    }

-- Take first N elements
func take(n: int, xs: [a]) -> [a] =
    match (n, xs) {
        (0, _) => [],
        (_, []) => [],
        (n, Cons(x, rest)) => Cons(x, take(n - 1, rest))
    }
```

### After (Natural Spread)

```ailang
-- Sum list recursively (clear intent)
func sum(xs: [int]) -> int =
    match xs {
        [] => 0,
        [x, ...rest] => x + sum(rest)
    }

-- Take first N elements (intuitive)
func take(n: int, xs: [a]) -> [a] =
    match (n, xs) {
        (0, _) => [],
        (_, []) => [],
        (n, [x, ...rest]) => [x, ...take(n - 1, rest)]
    }

-- Pattern match multiple elements
func process(xs: [int]) -> string =
    match xs {
        [] => "empty",
        [x] => "singleton: " ++ show(x),
        [x, y] => "pair: " ++ show(x) ++ ", " ++ show(y),
        [x, y, ...rest] => "many: starts with " ++ show(x)
    }
```

---

## Deferred Features (v0.5.0+)

### Init/Last Destructuring

```ailang
-- Deferred: Spread at beginning
match xs {
    [...init, last] => last  -- Get last element
}

-- Why deferred: Requires different elaboration strategy
-- Cons chains naturally from left (head first)
-- Getting "last" requires full traversal
```

### Middle Spread (Not Planned)

```ailang
-- NOT SUPPORTED: Spread in middle
match xs {
    [first, ...middle, last] => ...  -- Too complex
}

-- Problem: Ambiguous - how many elements in middle?
-- Other languages don't support this either
```

---

## Backward Compatibility

✅ **Fully backward compatible**:
- Existing patterns use Cons constructor
- New spread syntax is additive
- No breaking changes

---

## Testing Strategy

### Unit Tests

```go
func TestParseListPatternSpread_HeadRest(t *testing.T) {
    input := "[x, ...rest]"
    // Assert: ListPatternSpread{Head: [VarPattern("x")], Rest: "rest"}
}

func TestParseListPatternSpread_MultipleHeads(t *testing.T) {
    input := "[x, y, z, ...rest]"
    // Assert: 3 head patterns, rest name
}

func TestParseListPatternSpread_JustSpread(t *testing.T) {
    input := "[...xs]"
    // Assert: Empty head, rest = "xs"
}

func TestElaborateListPatternSpread(t *testing.T) {
    input := "[x, y, ...rest]"
    // Assert elaborates to: Cons(x, Cons(y, rest))
}
```

### Integration Tests

```ailang
-- Test: Sum function
tests {
    test "sum empty" = sum([]) == 0;
    test "sum single" = sum([5]) == 5;
    test "sum multiple" = sum([1,2,3]) == 6;
}
```

---

## Risks & Mitigations

### Risk: Parsing Ambiguity
**Issue**: Could `...` be confused with other syntax?
**Mitigation**: Context-dependent:
- In list pattern: spread
- In list literal: error (not supported)
- Clear from context

### Risk: Complex Elaboration
**Issue**: Generating correct Cons chains
**Mitigation**:
- Well-defined algorithm (right-to-left)
- Extensive unit tests
- Simple cases first (defer init/last)

### Risk: Type Checking Edge Cases
**Issue**: Spread variable might conflict with other bindings
**Mitigation**:
- Pattern elaboration already handles shadowing
- Spread becomes normal Cons pattern (already validated)

---

## Success Criteria

1. ✅ Parser accepts `[x, ...rest]` syntax
2. ✅ Elaboration generates correct Cons patterns
3. ✅ Type checking works correctly
4. ✅ Recursive list functions work (sum, map, filter)
5. ✅ M-EVAL benchmarks improve

---

## Future Work

### Spread in List Literals (v0.5.0+)

```ailang
-- Create lists with spread
let xs = [1, 2, 3]
let ys = [0, ...xs, 4]  -- [0, 1, 2, 3, 4]
```

### Spread in Function Arguments (v0.6.0+)

```ailang
-- Variadic functions
func sum(...nums: [int]) -> int =
    fold((+), 0, nums)
```

---

## References

- [Pattern Matching](../../docs/LIMITATIONS.md#pattern-matching) - Current limitations
- [List Operations](../../stdlib/std/list.ail) - Functions that will benefit
- [JavaScript Spread](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Operators/Spread_syntax) - Similar syntax
- [Rust Patterns](https://doc.rust-lang.org/book/ch18-03-pattern-syntax.html) - `..` rest patterns

---

## Implementation Checklist

- [ ] Add `ListPatternSpread` AST node
- [ ] Modify `parseListPattern()` to handle `...`
- [ ] Add parser validation (one spread, must be last)
- [ ] Implement elaboration to Cons patterns
- [ ] Add unit tests for parsing and elaboration
- [ ] Create example file with recursive functions
- [ ] Update teaching prompt
- [ ] Run M-EVAL validation
- [ ] Update CHANGELOG.md
