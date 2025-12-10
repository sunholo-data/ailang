# M-FIX-IF-ELSE-LET: Better Error for Let in If-Else Branches

**Status**: ✅ Implemented
**Target**: v0.5.9
**Completed**: December 10, 2025
**Priority**: P1 (Medium)
**Actual Duration**: 1 hour
**Dependencies**: None
**Bug Report**: `msg_20251209_200820_dce74a87` from stapledons_voyage

## Implementation Summary

Improved error diagnostics for let bindings in if-else branches:
- ✅ Clear error message explains the parsing ambiguity
- ✅ Provides working code example with explicit braces
- ✅ Implemented in `internal/elaborate/expressions.go`
- ✅ Commit: `e7d304ac M-FIX-IF-ELSE-LET: Better error for let in if-else branches`

**Key Change:** When elaborator detects a let binding as the immediate expression of an if-else branch (without braces), it now emits a helpful error explaining that explicit braces are required.

## AI-First Alignment Check

**Score this feature against AILANG's core principles:**

| Principle | Impact | Score | Notes |
|-----------|--------|-------|-------|
| Reduce Syntactic Noise | 0 | 0 | No syntax change - just better diagnostics |
| Preserve Semantic Clarity | + | +1 | Clearer error explains what's happening |
| Increase Determinism | + | +1 | Explicit braces = unambiguous parsing |
| Lower Token Cost | 0 | 0 | No change to required tokens |
| **Net Score** | | **+2** | **Decision: Move forward** |

**Decision rule:** Net score > +1 → Move forward | ≤ 0 → Reject or redesign

## Problem Statement

Type inference fails when an if-else expression returns a list and the else branch contains let bindings followed by an expression, without explicit braces.

**Reproducer:**
```ailang
module test/bug

pure func buildList(x: int, maxX: int) -> [int] {
    if x > maxX then []
    else
        let v = x * 2;
        let rest = buildList(x + 1, maxX);
        v :: rest
}
```

**Error:**
```
Error: type error in test/bug (decl 0): type unification failed at [if branches at test.ail:4:5]: cannot unify list type with *types.TCon
```

**Current State:**
- Parser parses `let v = x;` as the else branch expression (a let without body)
- The subsequent `[v]` or `v :: rest` is parsed as a separate statement after the entire if-else
- Variable `v` is not in scope for the final expression
- Type checker sees else branch type as `TCon` (from the bodyless let) instead of `[int]`

**Impact:**
- Affects users from ML/Haskell background who expect implicit blocks
- Confusing error message ("cannot unify list type with *types.TCon")
- Forces verbose explicit braces for a common functional programming pattern

**Workaround (current):**
```ailang
-- This works with explicit braces
pure func buildList(x: int, maxX: int) -> [int] {
    if x > maxX then [] else {
        let v = x * 2;
        let rest = buildList(x + 1, maxX);
        v :: rest
    }
}
```

## Goals

**Primary Goal:** Provide clear error message when users attempt let bindings in if-else branches without braces

**Success Metrics:**
- Error message clearly explains the issue and provides working code example
- LIMITATIONS.md documents the required syntax
- No breaking changes to existing code
- Users from ML/Haskell background understand immediately what to do

## Solution Design

### Overview

Improve diagnostics only - no syntax changes. When the type checker detects an if-else where one branch is a "let without body", emit a specialized error explaining that explicit braces are required.

**Why not implicit blocks?** (Deferred to Future Work)

Without layout-sensitive parsing, implicit blocks have ambiguous boundaries:

```ailang
if cond then a
else
    let v = ...
    v
foo()  -- Is this after the if, or still part of else?
```

Options for implicit blocks all have tradeoffs:
- **"Until end of enclosing block"**: Surprising - `foo()` becomes part of else
- **"Lookahead for semicolons"**: Requires complex grammar rules
- **Layout-sensitive parsing**: Major lexer change (Python/Haskell style)
- **Explicit `do` syntax**: `else do let v = ...; v` - new keyword

For now, explicit braces `{}` are the cleanest solution - unambiguous and familiar.

### Architecture

**Detection Strategy:**

In `inferIf`, when unification fails between then/else branches:
1. Check if either branch is a `*core.Let` (let without body elaborates to Let with body = continuation)
2. Actually, the issue is earlier: check in elaboration if we have `*ast.Let` with `Body == nil`
3. Emit specialized error with suggestion

**Error Message Format:**
```
Error: if-else branches require explicit braces when using let bindings

  4 |     if x > maxX then []
  5 |     else
  6 |         let v = x * 2;
             ^-- let binding parsed as branch expression (no body)

The parser cannot determine where multi-statement branches end without braces.

Fix: Wrap the else branch in braces:
    else {
        let v = x * 2;
        v :: buildList(x + 1, maxX)
    }
```

**Components:**
1. **Pattern Detection**: Check for let-without-body in if-else during elaboration or type checking
2. **Error Message**: Structured error with source location and fix suggestion
3. **Documentation**: Update LIMITATIONS.md

### Implementation Plan

**Phase 1: Better Error Message** (~2 hours)
- [ ] Add detection for "let without body in if-else branch" pattern
- [ ] Create helpful error message with code example
- [ ] Add test case for error message quality
- [ ] Update docs/LIMITATIONS.md with workaround

### Files to Modify/Create

**Modified files:**
- `internal/elaborate/expressions.go` - Detect let-without-body in if branches (~30 LOC)
- `internal/errors/structured.go` - New error type with suggestion (~40 LOC)
- `docs/LIMITATIONS.md` - Document required syntax (~30 LOC)

## Examples

### Example 1: List Building with Let Bindings

**Current behavior (confusing error):**
```ailang
pure func buildList(x: int, maxX: int) -> [int] {
    if x > maxX then []
    else
        let v = x * 2;
        v :: buildList(x + 1, maxX)
}
-- Error: cannot unify list type with *types.TCon
```

**After (clear error with fix):**
```
Error: if-else branches require explicit braces when using let bindings

  4 |     if x > maxX then []
  5 |     else
  6 |         let v = x * 2;
             ^-- let binding parsed as branch expression (no body)

The parser cannot determine where multi-statement branches end without braces.

Fix: Wrap the else branch in braces:
    else {
        let v = x * 2;
        v :: buildList(x + 1, maxX)
    }
```

### Example 2: Correct Syntax

**With explicit braces (required):**
```ailang
pure func buildList(x: int, maxX: int) -> [int] {
    if x > maxX then [] else {
        let v = x * 2;
        v :: buildList(x + 1, maxX)
    }
}
```

**Alternative: Accumulator pattern (no let needed):**
```ailang
pure func buildList(acc: [int], x: int, maxX: int) -> [int] {
    if x > maxX then acc
    else buildList(x :: acc, x + 1, maxX)
}
```

## Success Criteria

- [ ] Reproducer from bug report gives clear, helpful error message
- [ ] Error message includes working code example with braces
- [ ] All existing tests passing
- [ ] Documentation updated (docs/LIMITATIONS.md)
- [ ] Test case for error message quality

## Testing Strategy

**Unit tests:**
- Elaboration: detect let-without-body in if-else branch
- Error message: verify suggestion text includes correct fix
- Verify error includes source location

**Integration tests:**
- Test file with the bug reproducer pattern
- Verify error message is actionable

**Manual testing:**
- Verify error message is clear to FP developers
- Test with stapledons_voyage project patterns

## Non-Goals

**Not in this feature:**
- Implicit blocks in if-else branches - Ambiguous without layout sensitivity (see Future Work)
- Match expressions with let bindings - Same ambiguity issues
- Changing let syntax to require `in` - Breaking change
- Layout-sensitive parsing - Major lexer change

## Timeline

**Day 1** (2 hours):
- Implement error detection
- Create helpful error message
- Update LIMITATIONS.md
- Add test case

**Total: ~2 hours**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Error detection misses edge cases | Low | Cover common patterns, iterate based on user feedback |
| Users want implicit blocks | Low | Document reasoning, consider for future version |

## References

- Bug report: `ailang messages read msg_20251209_200820_dce74a87`
- Parser: [internal/parser/parser_expr.go:55](internal/parser/parser_expr.go#L55) (parseIfExpression)
- Type checker: [internal/types/typechecker_functions.go:385](internal/types/typechecker_functions.go#L385) (inferIf)
- Block elaboration: [internal/elaborate/expressions.go:499](internal/elaborate/expressions.go#L499)

## Future Work: Implicit Blocks (Deferred)

**Why deferred:** Without layout-sensitive parsing, implicit blocks have ambiguous end-of-block semantics.

**Options for future consideration:**

1. **Layout-sensitive parsing** (Python/Haskell style)
   - Requires lexer changes to emit INDENT/DEDENT tokens
   - Clean solution but significant implementation effort
   - Would enable: `else` followed by indented let-sequence

2. **Explicit `do` syntax**
   ```ailang
   if x > maxX then []
   else do
       let v = x * 2;
       v :: buildList(x + 1, maxX)
   ```
   - Clear delimiter without braces
   - Familiar to Haskell users
   - Unambiguous end-of-block (next dedent or `}`/`)`/`,`)

3. **Constrained implicit blocks**
   - Only desugar when implicit block runs to end of enclosing `{}`
   - Limited but predictable
   - Users must use braces if code follows the if-else

**Decision:** Gather user feedback on the error message approach. If demand is high, consider Option 2 (`do` syntax) as least disruptive.

---

**Document created**: 2025-12-09
**Last updated**: 2025-12-09
