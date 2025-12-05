# M-DX10: Parser Bug with List Pattern Match After Multi-Line Formatting

**Status**: Planned
**Target**: v0.4.2
**Priority**: P0 (High - blocks correct code)
**Estimated**: 0.5 days (3h investigation + 1h fix + 2h testing)
**Dependencies**: None

## AI-First Alignment Check

**Score this feature against AILANG's core principles:**

| Principle | Impact | Score | Notes |
|-----------|--------|-------|-------|
| Reduce Syntactic Noise | 0 | 0 | Bug fix, no new syntax |
| Preserve Semantic Clarity | + | +1 | Fixes incorrect parse rejection of valid code |
| Increase Determinism | + | +1 | Parser behavior becomes consistent with grammar |
| Lower Token Cost | + | +1 | Models can use natural multi-line formatting |
| **Net Score** | | **+3** | **Decision: Move forward (CRITICAL BUG)** |

**Decision rule:** Net score > +1 → Move forward | ≤ 0 → Reject or redesign

**Reference:** See [AI-first DX philosophy](../v0_3_15/example-parity-vision-alignment.md#-design-principle-ai-first-dx)

## Problem Statement

The parser incorrectly rejects valid list pattern matching code when formatted across multiple lines with `::(h, t) =>` syntax. This causes PAR_001 errors for correct AILANG code.

**Current State:**
- Code using `::(kv, rest) =>` after `[] =>` gets PAR_UNEXPECTED_TOKEN errors
- Error message: `expected next token to be =>, got ( instead`
- Happens at specific line/column positions (e.g., line 11:7)
- Parser seems to lose track of context after previous pattern arm

**Impact:**
- **False negatives**: Correct AILANG code rejected by parser
- **AI confusion**: Models write correct code, then get cryptic errors
- **Prompt effectiveness reduced**: Anti-patterns can't fix parser bugs
- **v0.4.1 regression**: New examples use this pattern, models copy it, code fails

**Evidence from v0.4.1 test:**
```ailang
-- This code SHOULD work but gets PAR_001:
export func findKey(kvs: [{key: string, value: Json}], target: string) -> Json {
  match kvs {
    [] => JNull,
    ::(kv, rest) =>   -- ❌ Parser error here!
      if _str_eq(kv.key, target)
      then kv.value
      else findKey(rest, target)
  }
}
```

**Error:**
```
PAR_UNEXPECTED_TOKEN at benchmark/solution.ail:11:7:
expected next token to be =>, got ( instead
```

## Goals

**Primary Goal:** Fix parser to correctly handle `::(h, t) =>` patterns in multi-line match expressions

**Success Metrics:**
- Zero PAR_001 errors for valid list pattern code
- json_parse benchmark (and similar) pass with correct `::(kv, rest) =>` syntax
- Parser correctly handles both `[] =>` and `::(h, t) =>` in same match
- Works regardless of whitespace/formatting

## Solution Design

### Overview

The parser is likely failing to properly consume tokens after the first pattern arm `[] =>`. When it encounters the second pattern `::(kv, rest)`, it's already in an incorrect state and expects `=>` but sees `(` from the cons constructor.

**Hypothesis**: Delimiter tracking or token consumption bug in `parseMatchArm()` or `parsePattern()`.

### Architecture

**Root Cause Investigation:**
1. **Check delimiter stack**: Is `{` from match block properly tracked?
2. **Check pattern parsing**: Does `parsePattern()` correctly consume `::(` as constructor?
3. **Check arm separation**: Is comma after first arm properly handled?
4. **Check lookahead**: Is `peekToken` correct after `[] => JNull,`?

**Likely Culprits:**
- `internal/parser/parser.go` - `parseMatchExpression()` or `parseMatchArm()`
- `internal/parser/patterns.go` - `parsePattern()` or `parseConstructorPattern()`
- Delimiter tracking in match expressions (known issue from CLAUDE.md line 263)

### Implementation Plan

**Phase 1: Reproduce & Diagnose** (~2 hours)
- [ ] Create minimal failing test case (just match with 2 arms)
- [ ] Add DEBUG_PARSER=1 logging to show token flow
- [ ] Trace exact token sequence parser sees
- [ ] Identify where parser state becomes incorrect
- [ ] Write failing unit test in `internal/parser/parser_test.go`

**Phase 2: Fix** (~1 hour)
- [ ] Apply fix based on diagnosis (likely token consumption or delimiter tracking)
- [ ] Verify unit test passes
- [ ] Test with original failing code from json_parse benchmark
- [ ] Ensure other match expressions still work (regression check)

**Phase 3: Testing & Documentation** (~2 hours)
- [ ] Add unit tests for edge cases:
  - Match with 2+ pattern arms using `::`
  - Match with nested patterns `::(h, ::(h2, t2))`
  - Match with guards `n if n > 0 =>`
  - Match with complex patterns in records
- [ ] Update parser documentation
- [ ] Remove "Known parser bug" warning from CLAUDE.md line 263 if this fixes it
- [ ] Test with full eval suite to verify no regressions

### Files to Modify/Create

**New files:**
- None (or maybe `internal/parser/match_patterns_test.go` if we split tests)

**Modified files:**
- `internal/parser/parser.go` - Fix in `parseMatchExpression()` or `parseMatchArm()` (~10 LOC change)
- `internal/parser/patterns.go` - Possible fix in `parsePattern()` or `parseConstructorPattern()` (~5 LOC)
- `internal/parser/parser_test.go` - Add failing/passing test cases (~80 LOC)
- `CLAUDE.md` - Remove "Known parser bug" warning if fixed (~3 LOC removed)

**Total new code: ~90 LOC tests, ~15 LOC fix**

## Examples

### Example 1: List Pattern Matching (Failing in v0.4.1)

**Current behavior:**
```ailang
-- ❌ Parser rejects this:
export func findKey(kvs: List[{key: string, value: Json}], target: string) -> Json {
  match kvs {
    [] => JNull,
    ::(kv, rest) =>   -- PAR_UNEXPECTED_TOKEN: expected =>, got (
      if _str_eq(kv.key, target)
      then kv.value
      else findKey(rest, target)
  }
}
```

**Error:** PAR_UNEXPECTED_TOKEN at line 4:7

**Expected behavior after fix:**
```ailang
-- ✅ Parser accepts this:
export func findKey(kvs: List[{key: string, value: Json}], target: string) -> Json {
  match kvs {
    [] => JNull,
    ::(kv, rest) =>   -- ✅ Works!
      if _str_eq(kv.key, target)
      then kv.value
      else findKey(rest, target)
  }
}
```

**Compiles and runs correctly**

### Example 2: Simplified Repro Case

**Minimal failing case:**
```ailang
module test/parser

export func test(xs: List[int]) -> int {
  match xs {
    [] => 0,
    ::(x, rest) => x + test(rest)   -- Should parse but doesn't
  }
}
```

**After fix:**
```ailang
module test/parser

export func test(xs: List[int]) -> int {
  match xs {
    [] => 0,
    ::(x, rest) => x + test(rest)   -- ✅ Parses correctly
  }
}
```

### Example 3: Workaround in v0.4.1 (Remove After Fix)

**Current workaround (using Cons instead of ::):**
```ailang
-- ✅ This works as workaround:
export func findKey(kvs: List[{key: string, value: Json}], target: string) -> Json {
  match kvs {
    [] => JNull,
    Cons(kv, rest) =>   -- Use Cons instead of ::
      if _str_eq(kv.key, target)
      then kv.value
      else findKey(rest, target)
  }
}
```

**After fix, both should work:**
- `::(kv, rest) =>` ← Preferred (prompt examples use this)
- `Cons(kv, rest) =>` ← Also works (alias for `::`)

## Success Criteria

- [ ] Parser accepts `::(h, t) =>` in match expressions
- [ ] Parser accepts `[] => value, ::(h, t) => expr` pattern
- [ ] json_parse benchmark passes with claude-haiku-4-5 (was failing with PAR_001)
- [ ] All existing parser tests still pass (no regressions)
- [ ] New unit tests cover edge cases (nested patterns, guards, etc.)
- [ ] Documentation updated (remove "Known parser bug" note if appropriate)
- [ ] DEBUG_PARSER=1 output shows correct token flow

## Testing Strategy

**Unit tests:**
- Match with 2 arms: `[] => x, ::(h,t) => y`
- Match with 3+ arms: `[] => x, ::(h, []) => y, ::(h, t) => z`
- Nested cons patterns: `::(h, ::(h2, t2)) => ...`
- With guards: `::(x, xs) if x > 0 => ...`
- With complex expressions after `=>`

**Integration tests:**
- Run json_parse benchmark with claude-haiku (was PAR_001)
- Run deterministic_list_transform with all models
- Run list_operations benchmark
- Verify all examples/ files with list patterns still work

**Manual testing:**
- Create test file with problematic pattern
- Run `DEBUG_PARSER=1 ailang run test.ail`
- Verify token flow is correct
- Check error messages are clear if pattern still wrong

## Non-Goals

**Not in this fix:**
- **Match inside blocks** - Separate known issue (CLAUDE.md line 263 mentions this)
- **Infix cons syntax `x :: rest`** - Requires grammar change, not parser bug (see M-SYNTAX-SUGAR)
- **Pattern exhaustiveness checking** - Typechecker responsibility, not parser
- **Pretty error messages** - Focus on accepting valid code first

## Timeline

**Day 1 Morning** (3 hours):
- Reproduce bug with minimal test case
- Add DEBUG_PARSER logging
- Trace token flow and identify bug location

**Day 1 Afternoon** (3 hours):
- Implement fix
- Write unit tests
- Test with original failing code
- Run full parser test suite

**Total: ~6 hours across 0.5 days**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Fix breaks other match expressions | High | Comprehensive test suite before/after |
| Root cause is deeper (lexer issue) | Medium | DEBUG_PARSER will reveal if lexer tokens wrong |
| Multiple bugs, not just one | Medium | Fix most common case first, file issues for others |
| Workaround (Cons) becomes preferred | Low | Prompt examples use `::`, models will follow |

## References

- Parser code: `internal/parser/parser.go` (`parseMatchExpression()`)
- Pattern parsing: `internal/parser/patterns.go` (`parsePattern()`, `parseConstructorPattern()`)
- Known parser bug warning: `CLAUDE.md:263` (delimiter tracking in match blocks)
- Parser developer guide: `docs/guides/parser_development.md` (token position conventions)
- V0.4.1 test failure: `eval_results/test_v0.4.1_phase2/standard/json_parse_ailang_claude-haiku-4-5_*.json`
- Error in eval: PAR_UNEXPECTED_TOKEN at line 11:7 (expected `=>`, got `(`)

## Future Work

- **Match inside blocks** (separate issue): Known parser bug with delimiter tracking
- **Infix cons syntax** (v0.6.0): Support `x :: rest` instead of `::(x, rest)` (M-SYNTAX-SUGAR)
- **Better parse error recovery**: Continue parsing after error to find more issues
- **Pattern validation**: Warn about non-exhaustive matches (typechecker)

---

**Document created**: 2025-11-01
**Last updated**: 2025-11-01
