# M-DX-RECORD-CONS: Record Literal + :: Cons Pattern Bug

**Status**: Planned
**Priority**: Low
**Source**: DX feedback from docparse-demo (Feb 2026)
**Milestone**: v0.8.0

## Problem

Using a record literal as the head of a `::` cons pattern fails with `PAT_INVALID_CONS`:

```ailang
-- FAILS:
match rows with
| { text: s, bold: b } :: rest -> ...

-- WORKAROUND (works):
match rows with
| first :: rest ->
  let { text: s, bold: b } = first
  ...
```

**Error**: `PAT_INVALID_CONS: :: constructor requires arguments in pattern. Use ::(head, tail) or x :: xs pattern`

## Analysis

### Parser Pattern Architecture

The AILANG parser uses a two-level pattern parsing hierarchy in `internal/parser/parser_pattern.go`:

1. **`parsePattern()`** (lines 10-53): Handles infix `::` (right-associative cons operator)
2. **`parseBasePattern()`** (lines 56-102): Parses atomic/non-infix patterns

### How `::` Desugaring Works

The infix `x :: xs` is desugared to `ConstructorPattern{Name: "::", Patterns: [x, xs]}`:

```go
// In parsePattern():
for p.peekTokenIs(lexer.DCOLON) {
    p.nextToken() // move to DCOLON
    p.nextToken() // move to RHS
    rhs := p.parsePattern()
    pat = &ast.ConstructorPattern{Name: "::", Patterns: []ast.Pattern{pat, rhs}}
}
```

### Root Cause: Parser Cursor Position

The bug is a **cursor position mismatch** between record pattern parsing and the `::` infix handler.

`parseBasePattern()` dispatches on `LBRACE` to `parseRecordPattern()`. The parser convention is "leave cursor AT the last token of the pattern." Record patterns should leave the cursor at `RBRACE`.

However, the `parsePattern()` infix handler checks `p.peekTokenIs(lexer.DCOLON)` to see if `::` follows. If `parseRecordPattern()` doesn't correctly implement the cursor convention, the parser may:

1. Advance past `RBRACE` to land ON `::` directly
2. Fall into `parseBasePattern()` again, which sees `DCOLON` at current position
3. Hit the PAT_INVALID_CONS error (line 86): bare `::` without `LPAREN` after it

### Evidence

The test suite in `internal/parser/list_cons_pattern_test.go` covers:
- Simple cons: `::(x, rest)`
- Nested cons: `::(h, ::(h2, t2))`
- Cons with tuples: `::((k, v), rest)`
- Cons with ADT constructors: `::(Some(x), rest)`
- **Missing: Cons with record patterns** (no test for `{ field } :: rest`)

## Proposed Fix

### Option A: Fix Record Pattern Cursor Position (Recommended)

Audit `parseRecordPattern()` to ensure it leaves the cursor AT `RBRACE`, not past it. Add a specific test case.

### Option B: Support Record Head in Canonical Form

If the infix form is hard to fix, support the canonical form explicitly:

```ailang
-- Canonical form (should always work):
| ::({ text: s }, rest) -> ...
```

### Option C: Both

Fix cursor position for infix form AND add canonical form test.

## Implementation Plan

1. Add failing parser test: `{ text: s } :: rest`
2. Add `DEBUG_PARSER=1` trace to observe cursor position after record pattern
3. Fix cursor position in `parseRecordPattern()` if incorrect
4. Add elaboration test: record cons pattern elaborates to `ListPattern` correctly
5. Run full test suite + `make verify-examples`

## Files to Modify

| File | Change |
|------|--------|
| `internal/parser/parser_pattern.go` | Fix record pattern cursor position |
| `internal/parser/list_cons_pattern_test.go` | Add record cons test case |
| `internal/elaborate/patterns_test.go` | Add elaboration test |
| `examples/record_cons_pattern.ail` | Example file |

## Risk Assessment

- **Low risk**: Parser cursor fix is localized to one code path
- **Testing**: Parser tests are comprehensive; regression unlikely
- **Workaround exists**: `let` binding first, then match fields
