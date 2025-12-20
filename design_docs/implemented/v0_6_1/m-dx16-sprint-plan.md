# M-DX16 Sprint Plan: Inline Record Literals in Match Arms

**Sprint ID**: M-DX16
**Duration**: 4 hours (single session)
**Risk Level**: Low
**Design Doc**: [m-dx16-inline-record-match-arms.md](m-dx16-inline-record-match-arms.md)

## Sprint Goal

Enable record literals directly in match arms without requiring helper function workarounds.

## Current Status

**Velocity Analysis (last 7 days):**
- Recent parser-related work: M3-PARSER (contracts), M-CODEGEN-ADT-DOUBLE-PAREN
- Average milestone: ~150-300 LOC
- Team pace: Good - multiple features landing per week

**Implementation Status:**
- Parser has record literal disambiguation in `parseRecordLiteral()` ✅
- Match expression parsing bypasses this logic ❌
- Fix is well-understood: apply same lookahead logic

## Milestones

### M1: Investigation & Failing Tests (~1 hour, ~50 LOC)

**Goal**: Verify current behavior and establish test baseline

**Tasks:**
1. Write failing test: `TestParseRecordLiteralInMatchArm`
   - Parse `match x { A => {f: 1} }` - should produce Record node
   - Currently fails (produces Block)
2. Write failing test: `TestParseNestedRecordInMatchArm`
   - Parse `match x { A => {f: {g: 2}} }`
3. Verify existing tests:
   - `TestParseMatchWithBlock` still expects Block for `{ e1; e2 }`
   - `TestParseMatchWithEmptyBlock` expects `{}`→Block

**Acceptance Criteria:**
- [ ] Failing tests demonstrate the bug
- [ ] Existing block-in-match tests pass (baseline)
- [ ] Clear test names document expected behavior

**Files:**
- `internal/parser/parser_test.go` (+50 LOC)

---

### M2: Parser Fix (~1 hour, ~30 LOC)

**Goal**: Modify `parseCase()` to use record literal disambiguation

**Tasks:**
1. Option A (simpler): Replace special-case with `parseExpression(LOWEST)`
   - Relies on `parseRecordLiteral` handling all cases
   - Risk: may break delimiter tracing
2. Option B (safer): Add lookahead before block parsing
   - Check `IDENT COLON` pattern before treating as block
   - Matches existing `parseRecordLiteral` logic

**Recommended approach**: Start with Option A, fall back to Option B if tests fail.

**Implementation:**
```go
// In parseCase(), replace lines 224-232:
// Before:
if p.curTokenIs(lexer.LBRACE) {
    p.traceDelimiterOpen(delimCtxCase)
    c.Body = p.parseBlockOrExpression()
    p.traceDelimiterClose(delimCtxCase)
} else {
    c.Body = p.parseExpression(LOWEST)
}

// After (Option A):
c.Body = p.parseExpression(LOWEST)

// After (Option B - if delimiter tracing is needed):
if p.curTokenIs(lexer.LBRACE) {
    // Peek ahead to disambiguate record vs block
    saved := p.curToken
    p.nextToken() // move past LBRACE
    isRecord := p.curTokenIs(lexer.IDENT) && p.peekTokenIs(lexer.COLON)
    isUpdate := p.curTokenIs(lexer.IDENT) && p.peekTokenIs(lexer.PIPE)
    p.curToken = saved // restore position

    if isRecord || isUpdate {
        c.Body = p.parseExpression(LOWEST) // dispatch to parseRecordLiteral
    } else {
        p.traceDelimiterOpen(delimCtxCase)
        c.Body = p.parseBlockOrExpression()
        p.traceDelimiterClose(delimCtxCase)
    }
} else {
    c.Body = p.parseExpression(LOWEST)
}
```

**Acceptance Criteria:**
- [ ] M1 failing tests now pass
- [ ] `go test ./internal/parser/...` all green
- [ ] `make test` passes

**Files:**
- `internal/parser/parser_expr.go` (~20-30 LOC change)

---

### M3: Error Messages (~45 min, ~20 LOC)

**Goal**: Improve error messages when record parsing fails in match context

**Tasks:**
1. Add context hint when `{` followed by invalid record syntax
2. Suggest: "Did you mean `{field: value}` (record) or `{ expr }` (block)?"
3. Test with malformed inputs:
   - `match x { A => {field value} }` (missing colon)
   - `match x { A => {123: val} }` (non-identifier field)

**Acceptance Criteria:**
- [ ] Error message mentions record/block ambiguity
- [ ] Tests verify helpful error output

**Files:**
- `internal/parser/parser_expr.go` or `parser_literals.go` (~20 LOC)
- `internal/parser/parser_test.go` (+20 LOC)

---

### M4: Examples & Validation (~15 min, ~30 LOC)

**Goal**: Add working examples and verify full pipeline

**Tasks:**
1. Create `examples/runnable/record_in_match.ail`
2. Run `make verify-examples`
3. Check `docs/LIMITATIONS.md` - remove limitation if documented
4. Run `make test` for full validation

**Example file content:**
```ailang
module examples/runnable/record_in_match

type Color = Red | Green | Blue

pure func colorInfo(c: Color) -> {name: string, hex: string} =
    match c {
        Red => {name: "Red", hex: "#FF0000"}
        Green => {name: "Green", hex: "#00FF00"}
        Blue => {name: "Blue", hex: "#0000FF"}
    }

func main() -> () ! {IO} = {
    let info = colorInfo(Red);
    _io_println(info.name)
}
```

**Acceptance Criteria:**
- [ ] `examples/runnable/record_in_match.ail` parses and runs
- [ ] `make verify-examples` passes
- [ ] `make test` passes

**Files:**
- `examples/runnable/record_in_match.ail` (~30 LOC)

---

## Success Metrics

| Metric | Target |
|--------|--------|
| Failing tests → passing | 4+ tests |
| New example files | 1 |
| Regression tests passing | 100% |
| Total LOC | ~130 |

## Dependencies

- None - this is a self-contained parser fix

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|-----------|
| Delimiter tracking breaks | Low | Medium | Option B preserves tracing |
| Edge cases with nested braces | Low | Low | Existing `parseRecordLiteral` handles this |
| Existing tests fail | Low | High | Run full test suite at each step |

## Open Questions

1. **Option A vs B**: Try simpler Option A first. If delimiter tracing issues appear, fall back to Option B.
2. **Empty block `{}`**: Currently returns Block. Preserve this behavior.

---

**Created**: 2025-12-20
**Estimated completion**: 4 hours
