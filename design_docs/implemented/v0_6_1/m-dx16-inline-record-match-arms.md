# M-DX16: Inline Record Literals in Match Arms

**Status**: IMPLEMENTED
**Target**: v0.6.1
**Priority**: P1 (Medium) - Significant DX friction for data-heavy code
**Estimated**: 4 hours
**Dependencies**: None

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No change to execution semantics |
| A2: Replayability | 0 | No impact on traces |
| A3: Effect Legibility | 0 | No change to effect handling |
| A4: Explicit Authority | 0 | No capability changes |
| A5: Bounded Verification | +1 | Enables inline type checking without helper indirection |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | Reduces boilerplate that AIs must generate for data-heavy code |
| A8: Minimal Syntax | +1 | No new syntax - enables existing syntax in more contexts |
| A9: Cost Visibility | 0 | Resource costs unchanged |
| A10: Composability | +1 | Record literals now compose with match expressions as expected |
| A11: Structured Failure | +1 | Better error messages explaining the ambiguity |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +5** → **Decision: Move forward**

### Hard Violation Check

**These axioms cannot have −1 scores (automatic rejection):**

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Improves machine analysis by reducing indirection

## Problem Statement

Inline record literals cannot be used directly in match arms, forcing users to create helper functions as workarounds.

**Current State:**
```ailang
-- ❌ This doesn't parse (user's reported issue)
match deckID {
    Bridge => {field1: val1, field2: val2}
}

-- ✅ Required workaround - creates unnecessary boilerplate
pure func bridgeInfo() -> DeckInfo = {field1: val1, field2: val2}

match deckID {
    Bridge => bridgeInfo()
}
```

**Root Cause (parser analysis):**

In `internal/parser/parser_expr.go:224-232`, `parseCase()` has a special case:

```go
if p.curTokenIs(lexer.LBRACE) {
    // Parse as block using the same logic as function bodies
    p.traceDelimiterOpen(delimCtxCase)
    c.Body = p.parseBlockOrExpression()  // ← Always treats { as block
    p.traceDelimiterClose(delimCtxCase)
} else {
    c.Body = p.parseExpression(LOWEST)   // ← Would dispatch to record parser
}
```

This bypasses the normal prefix parser dispatch (`parseRecordLiteral`), which correctly distinguishes:
- `{field: value}` → Record literal (via `IDENT COLON` lookahead)
- `{expr; expr}` → Block expression (otherwise)

**Impact:**
- Every record literal in match arms requires a helper function
- Nested records (`{pos: {x: 1.0, y: 2.0}}`) are especially painful
- Configuration/metadata-heavy code has excessive boilerplate
- Error messages say "expected =>" without explaining the actual conflict

## Goals

**Primary Goal:** Allow record literals directly in match arms without helper function workarounds.

**Success Metrics:**
- `match x { A => {field: val} }` parses successfully as record literal
- Nested records `{pos: {x: 1.0, y: 2.0}}` work in match arms
- Block expressions `{ expr1; expr2 }` continue to work in match arms
- Error messages explain the `{` ambiguity when parsing fails
- Zero regression in existing match expression tests

## Solution Design

### Overview

Apply the same lookahead logic from `parseRecordLiteral()` to `parseCase()` before deciding whether to parse as block or use normal expression parsing.

### Architecture

**The Fix Pattern:**

The existing `parseRecordLiteral()` (lines 205-276 of `parser_literals.go`) already handles disambiguating `{...}`:

```go
isRecordLiteral := p.curTokenIs(lexer.IDENT) && p.peekTokenIs(lexer.COLON)
isRecordUpdate := p.curTokenIs(lexer.IDENT) && p.peekTokenIs(lexer.PIPE)

if isRecordUpdate {
    // {base | field: value}
} else if isRecordLiteral {
    // {field: value}
} else {
    // Block expression
}
```

**Apply same logic to parseCase():**

```go
// In parseCase(), after p.nextToken() following FARROW:
if p.curTokenIs(lexer.LBRACE) {
    // Peek inside to disambiguate
    p.nextToken() // move past LBRACE

    isRecordLiteral := p.curTokenIs(lexer.IDENT) && p.peekTokenIs(lexer.COLON)
    isRecordUpdate := p.curTokenIs(lexer.IDENT) && p.peekTokenIs(lexer.PIPE)

    // Back up to let appropriate parser handle it
    // ... (implementation detail - may need token buffer)
}
```

**Alternative (simpler):** Just use `parseExpression(LOWEST)` unconditionally - it already dispatches to `parseRecordLiteral` for `LBRACE`, which handles all cases.

```go
// Replace the entire if-else with:
c.Body = p.parseExpression(LOWEST)
```

This works because `parseRecordLiteral` already falls back to block parsing when the pattern isn't `IDENT COLON` or `IDENT PIPE`.

### Implementation Plan

**Phase 1: Investigation** (~1 hour)
- [ ] Verify `parseRecordLiteral` handles all cases (record, update, block)
- [ ] Write failing test case for inline record in match arm
- [ ] Write test case for nested records in match arm
- [ ] Verify existing block-in-match tests still define expected behavior

**Phase 2: Fix** (~1.5 hours)
- [ ] Modify `parseCase()` to use `parseExpression(LOWEST)` instead of special-cased block parsing
- [ ] OR: Add lookahead logic matching `parseRecordLiteral` before choosing parser
- [ ] Preserve delimiter tracing for error recovery
- [ ] Run existing parser tests

**Phase 3: Error Messages** (~1 hour)
- [ ] Improve error when record literal parsing fails in match context
- [ ] Add hint: "Did you mean `{field: value}` (record) or `{ expr }` (block)?"
- [ ] Test error messages with malformed inputs

**Phase 4: Validation** (~0.5 hours)
- [ ] Add examples to `examples/runnable/`
- [ ] Run `make verify-examples`
- [ ] Update LIMITATIONS.md (remove this limitation if it was documented)

### Files to Modify/Create

**Modified files:**
- `internal/parser/parser_expr.go` - Modify `parseCase()` (~20 LOC change)
- `internal/parser/parser_test.go` - Add test cases (~50 LOC)

**New files:**
- `examples/runnable/record_in_match.ail` - Example file (~20 LOC)

## Examples

### Example 1: Basic Record in Match Arm

**Before (fails to parse):**
```ailang
type DeckID = Bridge | Engine | Cargo

pure func getDeckInfo(id: DeckID) -> {name: string, level: int} =
    match id {
        Bridge => {name: "Bridge", level: 1}
        Engine => {name: "Engine", level: -2}
        Cargo => {name: "Cargo", level: 0}
    }
```

**After (works):**
```ailang
-- Same code, now parses correctly
type DeckID = Bridge | Engine | Cargo

pure func getDeckInfo(id: DeckID) -> {name: string, level: int} =
    match id {
        Bridge => {name: "Bridge", level: 1}
        Engine => {name: "Engine", level: -2}
        Cargo => {name: "Cargo", level: 0}
    }
```

### Example 2: Nested Records

**Before (requires helper):**
```ailang
pure func makePosition(x: float, y: float) -> {x: float, y: float} = {x: x, y: y}

pure func getLocation(id: LocationID) -> {name: string, pos: {x: float, y: float}} =
    match id {
        Home => {name: "Home", pos: makePosition(0.0, 0.0)}
        Work => {name: "Work", pos: makePosition(10.5, 20.3)}
    }
```

**After (inline):**
```ailang
pure func getLocation(id: LocationID) -> {name: string, pos: {x: float, y: float}} =
    match id {
        Home => {name: "Home", pos: {x: 0.0, y: 0.0}}
        Work => {name: "Work", pos: {x: 10.5, y: 20.3}}
    }
```

### Example 3: Block Still Works

**This continues to work:**
```ailang
func process(x: Option[int]) -> int ! {IO} =
    match x {
        Some(n) => {
            _io_println("Got: " ++ intToString(n));
            n * 2
        }
        None => 0
    }
```

## Success Criteria

- [ ] `match x { A => {f: v} }` parses as record literal
- [ ] `match x { A => {f: {g: v}} }` parses nested records
- [ ] `match x { A => { e1; e2 } }` still parses as block (semicolon disambiguates)
- [ ] `match x { A => {} }` parses as empty block (existing behavior preserved)
- [ ] Error message for `{field value}` (missing colon) mentions record syntax
- [ ] All existing parser tests pass
- [ ] All existing match expression tests pass
- [ ] `make verify-examples` passes

## Testing Strategy

**Unit tests:**
- Parse `match x { A => {f: 1} }` - verify AST has Record node
- Parse `match x { A => {f: {g: 2}} }` - verify nested Record
- Parse `match x { A => {e; e} }` - verify Block node
- Parse `match x { A => {} }` - verify empty Block

**Integration tests:**
- Run examples through full pipeline (parse → elaborate → type check → eval)
- Verify record values are accessible in match result

**Manual testing:**
- Test in REPL with various record patterns
- Verify error messages are helpful

## Non-Goals

**Not in this feature:**
- Record shorthand syntax `{field}` for `{field: field}` - separate feature
- Anonymous record types - existing behavior preserved
- Record spread syntax `{...base, field: val}` - future feature

## Timeline

**Single Sprint** (~4 hours):
- Phase 1: 1 hour - Investigation and tests
- Phase 2: 1.5 hours - Implementation
- Phase 3: 1 hour - Error messages
- Phase 4: 0.5 hours - Validation

**Total: ~4 hours in one session**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Block/record ambiguity in edge cases | Medium | Use same proven lookahead logic from `parseRecordLiteral` |
| Delimiter tracking breaks | Low | Preserve existing tracing calls, add tests for nested matches |
| Existing match tests fail | High | Run full test suite before and after, fix any regressions |

## Related Documents

**Parser architecture:**
- `internal/parser/parser_expr.go:162-235` - Match expression parsing
- `internal/parser/parser_literals.go:205-320` - Record literal parsing (has the correct disambiguation logic)

**Similar issues:**
- Match-in-block nested delimiter tracking (already fixed, see line 219-228)

## References

- [Design Axioms](/docs/references/axioms) - The 12 non-negotiable principles
- [Issue report](https://github.com/sunholo-data/ailang/issues) - User DX feedback

## Future Work

- Record shorthand syntax `{field}` → `{field: field}` (DX enhancement)
- Punning syntax for destructuring records in match patterns
- Better IDE/tooling hints for record vs block disambiguation

---

**Document created**: 2025-12-20
**Last updated**: 2025-12-20
