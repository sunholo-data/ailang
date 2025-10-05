# M-R8: Block Expressions

**Status**: Planned for v0.3.0
**Priority**: HIGH (unblocks AI-generated code)
**Effort**: ~200-300 LOC
**Time**: 0.5-1 day
**Risk**: LOW (pure syntactic sugar)

## Motivation

AI models (Claude Sonnet 4.5, GPT-4, etc.) naturally generate code with block syntax:

```ailang
func fizzBuzz(n: int) -> () ! {IO} {
  if n > 0 then {
    println(show(n));
    fizzBuzz(n - 1)
  } else {
    ()
  }
}
```

Currently, this fails to parse. Users must manually rewrite to:

```ailang
func fizzBuzz(n: int) -> () ! {IO} {
  if n > 0 then
    fizzBuzz(n - 1, println(show(n)))
  else
    ()
}
```

This is a **major friction point** for AI-assisted development.

## Proposal

Add block expressions `{ e1; e2; ...; en }` as **syntactic sugar** that desugars to let-sequencing.

### Syntax

```
Expr -> "{" BlockBody "}"
BlockBody -> Expr (";" Expr)* (";")?
```

### Semantics (Desugaring)

```
{ e1; e2; ...; en } ⇒ let _ = e1 in let _ = e2 in ... in en
```

- Value of block = value of last expression
- Non-last expressions evaluated for effects, values discarded
- Empty blocks `{ }` rejected with error

### Examples

**Simple sequence:**
```ailang
{ println("a"); println("b"); 42 }
```
Desugars to:
```ailang
let _ = println("a") in let _ = println("b") in 42
```

**If-then-else with blocks:**
```ailang
if condition then {
  println("true branch");
  1
} else {
  println("false branch");
  0
}
```

**Nested blocks:**
```ailang
{
  let x = 10;
  {
    println("inner");
    x * 2
  }
}
```

## Implementation Plan

### Phase 1: Parser (~100 LOC)

**File**: `internal/parser/parser.go`

Add production for block expressions:
- Recognize `{` in expression position
- Parse semicolon-separated expression list
- Allow trailing semicolons
- Reject empty blocks with clear error

**AST Node** (reuse existing):
```go
// Already exists in ast.go
type Block struct {
    Exprs []Expr
    Pos   Pos
}
```

**Edge Cases**:
- ✅ Single expression: `{ e }` (no semicolon required)
- ✅ Trailing semicolon: `{ e1; e2; }`
- ❌ Empty block: `{ }` → error "BLOCK_EMPTY: block must contain at least one expression"

### Phase 2: Elaboration (~50 LOC)

**File**: `internal/elaborate/elaborate.go`

Add case for `*ast.Block` to desugar to nested Let:

```go
case *ast.Block:
    if len(e.Exprs) == 0 {
        return nil, fmt.Errorf("BLOCK_EMPTY: empty block at %v", e.Pos)
    }
    if len(e.Exprs) == 1 {
        return elab.elaborate(e.Exprs[0])
    }

    // Desugar to let chain: let _ = e1 in let _ = e2 in ... in en
    result := elab.elaborate(e.Exprs[len(e.Exprs)-1])
    for i := len(e.Exprs) - 2; i >= 0; i-- {
        value := elab.elaborate(e.Exprs[i])
        result = &core.Let{
            Name:  "_",  // Throwaway binding
            Value: value,
            Body:  result,
        }
    }
    return result, nil
```

### Phase 3: Tests (~100 LOC)

**Parser Tests** (`internal/parser/parser_test.go`):
- Single expression block
- Multiple expressions with semicolons
- Trailing semicolon allowed
- Empty block error
- Nested blocks

**Elaboration Tests** (`internal/elaborate/elaborate_test.go`):
- Verify desugaring to let chains
- Preserve span information for errors

**Integration Examples**:

**`examples/micro_block_seq.ail`**:
```ailang
module examples/micro_block_seq

import std/io (println)

export func main() -> int ! {IO} {
  {
    println("first");
    println("second");
    42
  }
}
```

**`examples/micro_block_if.ail`**:
```ailang
module examples/micro_block_if

import std/io (println)

export func main() -> int ! {IO} {
  if true then {
    println("true branch");
    1
  } else {
    println("false branch");
    0
  }
}
```

**`examples/block_recursion.ail`**:
```ailang
module examples/block_recursion

import std/io (println)

export func countdown(n: int) -> () ! {IO} {
  if n <= 0 then {
    println("Done!")
  } else {
    println(show(n));
    countdown(n - 1)
  }
}

export func main() -> () ! {IO} {
  countdown(5)
}
```

### Phase 4: Documentation

**Update `README.md`**:
Add block expressions to syntax examples.

**Create `docs/guides/expressions.md`**:
```markdown
## Block Expressions

Blocks allow sequencing multiple expressions:

```ailang
{
  expr1;
  expr2;
  expr3
}
```

- Value of the block is the value of the last expression
- Non-last expressions are evaluated for their effects
- Empty blocks are not allowed

### Common Use Cases

**Sequencing IO operations:**
```ailang
func greet(name: string) -> () ! {IO} {
  {
    println("Hello, " ++ name);
    println("Welcome to AILANG!")
  }
}
```

**Conditional branches with multiple statements:**
```ailang
if condition then {
  println("Debug: entering true branch");
  computeResult()
} else {
  println("Debug: entering false branch");
  defaultValue
}
```

### Semantics

Blocks are syntactic sugar that desugar to let-sequencing:

```ailang
{ a; b; c }
```

Becomes:
```ailang
let _ = a in let _ = b in c
```

This ensures:
- All expressions are evaluated in order
- Side effects occur sequentially
- Only the last value is returned
```

## Error Messages

**BLOCK_EMPTY**: Empty block `{}` at line X
- **Hint**: A block must contain at least one expression
- **Fix**: Add an expression inside the block

**Example**:
```
Error: BLOCK_EMPTY: empty block at test.ail:5:10
  {
  ^
Hint: A block must contain at least one expression
```

## Type Checking

**No changes required** - blocks desugar to let expressions before type checking.

The type of a block is the type of its last expression:
```ailang
{ println("hello"); 42 }  -- Type: int
```

## Runtime

**No changes required** - blocks are eliminated during elaboration.

## Performance

**Impact**: Negligible
- Desugaring is O(n) in number of expressions
- Runtime identical to hand-written let chains
- No additional allocation or indirection

## Compatibility

**Breaking changes**: None
- Pure addition, no changes to existing syntax
- Backwards compatible with all existing code

## Future Extensions (Post v0.3.0)

**Optional enhancements**:
1. **Block-local let bindings**: Allow `let x = ... in` to scope only within block
2. **Early return**: `return expr` exits block early (requires control flow analysis)
3. **Pattern matching in blocks**: `{ match x { ... }; nextExpr }`

**NOT in scope for v0.3.0**:
- Statement syntax (assignments, loops, etc.)
- Mutable variables
- Break/continue

## Testing Strategy

**Unit Tests**:
- Parser: 5 tests (single expr, multi expr, trailing semi, empty error, nested)
- Elaboration: 3 tests (desugar correctness, span preservation, single expr)

**Integration Tests**:
- 3 example files (seq, if-then-else, recursion)
- All must pass with appropriate `--caps` flags

**Acceptance Criteria**:
- ✅ `{ e }` works (single expression)
- ✅ `{ e1; e2; e3 }` works (multiple expressions)
- ✅ Trailing semicolon allowed: `{ e1; e2; }`
- ✅ Empty blocks rejected with clear error
- ✅ Works in all expression contexts (function bodies, branches, let RHS)
- ✅ Type of block = type of last expression
- ✅ Non-last expressions type-checked but values discarded
- ✅ Examples pass with AI evaluation

## Rollback Plan

**If issues arise**:
1. Gate behind env var: `AILANG_ENABLE_BLOCKS=1`
2. Default off for v0.3.0-alpha, on for v0.3.0 stable
3. Full rollback: revert parser + elaboration changes (< 300 LOC)

**Unlikely to be needed** - pure sugar with no runtime implications.

## Implementation Checklist

### Day 1: Core Implementation
- [ ] Add block parsing to `internal/parser/parser.go`
- [ ] Add block desugaring to `internal/elaborate/elaborate.go`
- [ ] Add parser tests (5 tests)
- [ ] Add elaboration tests (3 tests)

### Day 1: Examples & Integration
- [ ] Create `examples/micro_block_seq.ail`
- [ ] Create `examples/micro_block_if.ail`
- [ ] Create `examples/block_recursion.ail`
- [ ] Verify all examples pass

### Day 1: Documentation
- [ ] Update `README.md` with block syntax
- [ ] Create `docs/guides/expressions.md`
- [ ] Add to CHANGELOG.md

## Success Metrics

**Before**:
- ❌ AI-generated code with blocks fails to parse
- ❌ Manual rewriting required
- ❌ Poor developer experience

**After**:
- ✅ AI-generated code with blocks works out of the box
- ✅ No manual rewriting needed
- ✅ Matches intuition from other languages (Go, Rust, C, etc.)
- ✅ AI evaluation benchmarks pass

## Related Work

**Similar implementations**:
- **OCaml**: `begin e1; e2; e3 end` (keyword-based)
- **Haskell**: `do { e1; e2; e3 }` (monadic)
- **Scala**: `{ e1; e2; e3 }` (same syntax)
- **Rust**: `{ e1; e2; e3 }` (same syntax)

**Our approach**: Most similar to Scala/Rust - pure expression blocks with semicolon separators.

## Dependencies

**Requires**:
- None (pure addition)

**Enables**:
- AI-generated code compatibility
- Better ergonomics for imperative-style code
- Cleaner conditional branches

**Blocks**:
- Nothing (no downstream dependencies)

## Risk Assessment

**Overall Risk**: **LOW**

**Risks**:
1. ⚠️ **Parser conflicts** with record syntax `{ }` - **Mitigation**: Only recognize in expression position, not type position
2. ⚠️ **Span preservation** in desugared let chains - **Mitigation**: Preserve original block span on outer let
3. ⚠️ **Empty block edge case** - **Mitigation**: Explicit error message with hint

**High Confidence Areas**:
- Desugaring semantics (well-understood let-sequencing)
- Type checking (no changes needed)
- Runtime (no changes needed)
- Testing (straightforward examples)

## Conclusion

Block expressions are a **small, high-value feature** that significantly improves AI compatibility and developer ergonomics. With ~300 LOC and 0.5-1 day of focused work, this is an excellent addition to v0.3.0.

The pure syntactic sugar approach minimizes risk while maximizing compatibility with AI-generated code patterns.
