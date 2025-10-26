# M-DX9: Parser Developer Experience Improvements

**Status**: Planned
**Target**: v0.3.15
**Priority**: P0 - High
**Estimated**: 4-5 days (2 days implementation + 1 day testing + 1 day docs + buffer)
**Dependencies**: None

## AI-First Alignment Check

**Score this feature against AILANG's core principles:**

| Principle | Impact | Score | Notes |
|-----------|--------|-------|-------|
| Reduce Syntactic Noise | 0 | 0 | Internal DX improvement, no language syntax changes |
| Preserve Semantic Clarity | + | +1 | Clearer parser contracts improve code maintainability |
| Increase Determinism | + | +1 | Better documentation reduces debugging uncertainty |
| Lower Token Cost | + | +1 | Reduced debugging time = faster AI-assisted development |
| **Net Score** | | **+3** | **Decision: Move forward** |

**Decision rule:** Net score > +1 → Move forward | ≤ 0 → Reject or redesign

**Rationale:** While this doesn't change the AILANG language itself, it dramatically improves the **AI-assisted development loop** for working on the compiler. Reducing 30% of development time spent on token position bugs means AI assistants (like Claude) can implement parser features ~40% faster. This is essential for AILANG's vision of being **maintained primarily by AI agents**.

**Reference:** See [AI-first DX philosophy](../v0_3_15/example-parity-vision-alignment.md#-design-principle-ai-first-dx)

## Problem Statement

During M-TESTING sprint (Days 1-2), **30% of development time was spent debugging token position issues** in parser code. This is the #1 pain point when extending AILANG's parser.

**Current State:**
- Parser functions have unclear token position semantics (AT vs AFTER)
- No documentation of the convention that `parseExpression()` leaves parser AT the last token (not AFTER)
- Test failures don't print error details (errors hidden by `t.Fatalf`)
- AST type names must be guessed (compilation failures from wrong types)
- Lexer keyword discovery requires manual grep (wastes time checking if tokens exist)

**Impact:**
- **Time waste:** 30% of parser development time (~2h out of 7h sprint)
- **Who affected:** AI assistants (Claude) and human developers working on parser
- **Severity:** High - this compounds across every parser feature we add

**Metrics from M-TESTING Sprint:**

| Activity | Time Spent | % of Total |
|----------|------------|------------|
| Understanding existing code | 0.75h | 15% |
| Writing parser code | 1.75h | 35% |
| **Debugging token positions** | **1.5h** | **30%** ⚠️ |
| Writing tests | 2h | 40% |
| Fixing test helpers | 1h | 20% |
| **Total** | **7h** | |

## Goals

**Primary Goal:** Reduce parser development time by 30% by eliminating token position debugging overhead

**Success Metrics:**
- Zero time spent debugging token positions in next sprint
- Parser development time reduced from 7h to ~5h (-28%)
- New parser features can be implemented in one sprint day instead of two
- AI assistants can implement parser features without human intervention
- Test failures show actionable error messages immediately

## Solution Design

### Overview

Implement three complementary improvements:

1. **Documentation** - Add parser conventions to CLAUDE.md and code comments
2. **Debug tooling** - Add `DEBUG_PARSER=1` flag for token position tracing
3. **Test infrastructure** - Fix error printing and add test helpers

These three approaches reinforce each other:
- Docs prevent bugs from being written
- Debug tools make bugs visible when they occur
- Test helpers make bugs easier to diagnose

### Architecture

**Components:**

1. **Parser Convention Documentation** (`CLAUDE.md`)
   - Document token position semantics (AT vs AFTER)
   - Add AST type quick reference
   - Add lexer keyword lookup guide
   - Add common patterns (optional sections after optional sections)

2. **Debug Mode** (`internal/parser/debug.go`)
   - `DEBUG_PARSER=1` environment variable
   - Log token positions on entry/exit of parse functions
   - Minimal performance overhead when disabled

3. **Test Improvements** (`internal/parser/*_test.go`)
   - Fix error printing (print before Fatalf)
   - Add `AssertTokenPosition` helper
   - Add AST structure assertion helpers

### Implementation Plan

**Phase 1: Documentation (P0)** (~2 hours)
- [x] Add "Parser Token Position Convention" section to CLAUDE.md
- [x] Add "Common AST Types Reference" to CLAUDE.md
- [x] Add "Quick Token Lookup" guide to CLAUDE.md
- [x] Add "Parsing Optional Sections" pattern to CLAUDE.md
- [x] Add package documentation to `internal/parser/parser.go`

**Phase 2: Test Infrastructure (P0)** (~3 hours)
- [ ] Fix all parser test files to print errors before Fatalf
- [ ] Create `internal/parser/test_helpers.go` with helpers
- [ ] Add unit tests for test helpers
- [ ] Update existing tests to use new helpers

**Phase 3: Debug Tooling (P1)** (~4 hours)
- [ ] Create `internal/parser/debug.go` with debug logging
- [ ] Add entry/exit logging to all parse functions
- [ ] Add tests for debug mode
- [ ] Document DEBUG_PARSER flag in CLAUDE.md

**Phase 4: Documentation Polish (P2)** (~2 hours)
- [ ] Add usage examples to AST struct comments in `internal/ast/ast.go`
- [ ] Document int64 vs int gotcha
- [ ] Add quick reference diagrams
- [ ] Create parser development guide in `docs/`

### Files to Modify/Create

**New files:**
- `internal/parser/debug.go` - Debug logging utilities (~150 LOC)
- `internal/parser/test_helpers.go` - Test assertion helpers (~200 LOC)
- `docs/guides/parser_development.md` - Parser dev guide (~300 LOC)

**Modified files:**
- `CLAUDE.md` - Add parser DX sections (~200 LOC)
- `internal/parser/parser.go` - Add package documentation (~50 LOC)
- `internal/parser/expressions_test.go` - Fix error printing (~20 LOC)
- `internal/parser/statements_test.go` - Fix error printing (~30 LOC)
- `internal/parser/patterns_test.go` - Fix error printing (~15 LOC)
- `internal/parser/types_test.go` - Fix error printing (~10 LOC)
- `internal/ast/ast.go` - Add usage examples to comments (~100 LOC)

**Total new code:** ~650 LOC
**Total modified code:** ~425 LOC

## Examples

### Example 1: Token Position Convention

**Before (no documentation):**
```go
// Developer must guess the contract
p.nextToken() // move to first token of expression
expr := p.parseExpression(LOWEST)
// ??? Is parser now AT or AFTER the expression?
// ⚠️ Trial-and-error debugging required
```

**After (documented convention):**
```go
// From CLAUDE.md:
// "parseExpression() leaves parser AT the last token (NOT after)"

p.nextToken() // move to first token
expr := p.parseExpression(LOWEST)  // parses expr, leaves cur=last_token
p.nextToken() // NOW we're after the expression ✓
```

### Example 2: Debug Mode

**Before (blind debugging):**
```go
func (p *Parser) parseTestsBlock() *ast.TestsBlock {
    // Where are we? No idea!
    tests := []*ast.TestCase{}
    for !p.curTokenIs(lexer.RBRACE) {
        // Why is this failing? No visibility!
        test := p.parseTestCase()
        tests = append(tests, test)
    }
    return &ast.TestsBlock{Tests: tests}
}
```

**After (DEBUG_PARSER=1):**
```bash
$ DEBUG_PARSER=1 ailang run test.ail
[ENTER parseTestsBlock] cur=LBRACE peek=TEST
[ENTER parseTestCase] cur=TEST peek=LPAREN
[EXIT parseTestCase] cur=RPAREN peek=COMMA
[ENTER parseTestCase] cur=TEST peek=LPAREN
[EXIT parseTestCase] cur=RPAREN peek=RBRACE
[EXIT parseTestsBlock] cur=RBRACE peek=PROPERTIES
# ✓ Can see exact token flow!
```

### Example 3: Test Error Printing

**Before (errors hidden):**
```go
func TestParseTests(t *testing.T) {
    p := New(lexer.New(input))
    file := p.ParseFile()

    if len(p.Errors()) != 0 {
        t.Fatalf("parser had %d errors:", len(p.Errors()))
        // ⚠️ This never executes! Errors are NOT printed
        for _, err := range p.Errors() {
            t.Errorf("  %s", err)
        }
    }
}

// Output:
//   parser had 9 errors:
// (no details shown!)
```

**After (errors visible):**
```go
func TestParseTests(t *testing.T) {
    p := New(lexer.New(input))
    file := p.ParseFile()

    if len(p.Errors()) != 0 {
        // ✓ Print errors BEFORE Fatalf
        for _, err := range p.Errors() {
            t.Errorf("  %s", err)
        }
        t.Fatalf("parser had %d errors", len(p.Errors()))
    }
}

// Output:
//   expected next token to be ), got COMMA
//   expected next token to be RPAREN, got IDENT
//   ... (all 9 errors shown)
//   parser had 9 errors
```

### Example 4: Test Helpers

**Before (verbose assertions):**
```go
func TestParseExpression(t *testing.T) {
    p := New(lexer.New("42"))
    p.nextToken()
    expr := p.parseExpression(LOWEST)

    // ⚠️ Verbose and error-prone
    if !p.curTokenIs(lexer.INT) {
        t.Errorf("Expected cur=%s, got %s", lexer.INT, p.curToken.Type)
    }
    if !p.peekTokenIs(lexer.EOF) {
        t.Errorf("Expected peek=%s, got %s", lexer.EOF, p.peekToken.Type)
    }
}
```

**After (helper functions):**
```go
func TestParseExpression(t *testing.T) {
    p := New(lexer.New("42"))
    p.nextToken()
    expr := p.parseExpression(LOWEST)

    // ✓ Clear and concise
    AssertTokenPosition(t, p, lexer.INT, lexer.EOF)
}
```

### Example 5: AST Type Discovery

**Before (trial and error):**
```go
// Developer must guess type names
expr := p.parseExpression(LOWEST)
// Is it ast.IntLiteral? Compilation error!
// Is it ast.Literal with .Value int? Type error!
// Is it ast.Literal with .Value int64? Finally works!
```

**After (documented in CLAUDE.md):**
```markdown
### Common AST Types Reference

**Literals:** `ast.Literal` with `Kind` field and `Value interface{}`
- ⚠️ GOTCHA: Lexer returns `int64`, not `int`
- Access: `lit.Value.(int64)` NOT `lit.Value.(int)`

**Quick check:** `grep "^type.*struct" internal/ast/ast.go | head -20`
```

## Success Criteria

- [x] Documentation added to CLAUDE.md with token position convention
- [ ] All parser test files print errors before Fatalf
- [ ] Test helpers created and used in at least 5 tests
- [ ] DEBUG_PARSER flag implemented and tested
- [ ] Next parser sprint has <5% time spent on token position debugging (vs 30% baseline)
- [ ] All tests passing (no regressions)
- [ ] Documentation updated (CLAUDE.md, parser.go comments, AST comments)
- [ ] Examples added showing before/after workflow

**Quantitative success metric:**
- **Current:** 7h parser sprint with 30% debugging overhead (2.1h wasted)
- **Target:** 5h parser sprint with <5% debugging overhead (<0.25h wasted)
- **Improvement:** -28% total time, -86% debugging time

## Testing Strategy

**Unit tests:**
- Test helper functions (`AssertTokenPosition`, etc.)
- Debug mode logging (verify output format)
- Edge cases (empty input, EOF handling)

**Integration tests:**
- Run M-TESTING examples with DEBUG_PARSER=1
- Verify all test files print errors correctly
- Test that helpers work with real parser code

**Manual testing:**
- Next parser feature (M-TESTING Day 3?) should use new tools
- Track time spent on token debugging
- Collect feedback on documentation clarity

**Validation:**
- Run full test suite: `make test`
- Run parser tests specifically: `go test ./internal/parser/...`
- Verify no performance regression (debug mode off)

## Non-Goals

**Not in this feature:**
- Changing parser architecture or token consumption strategy
  - **Why deferred:** Would affect entire parser, too risky for DX improvement
- Automated parser generation or DSL
  - **Why out of scope:** Hand-written parser is intentional design choice
- LSP/IDE integration for parser development
  - **Why out of scope:** AILANG focuses on CLI tools, not IDE features
- Reformatting existing parser code
  - **Why deferred:** Risk of regressions, focus on documentation first

## Timeline

**Week 1** (8 hours):
- Phase 1: Documentation (2h)
- Phase 2: Test infrastructure (3h)
- Phase 3: Debug tooling (4h)

**Week 2** (6 hours):
- Testing and validation (3h)
- Documentation polish (2h)
- Integration with next parser sprint (1h)

**Total: ~14 hours across 2 weeks**

**Milestones:**
- Day 1: Documentation complete, can be used immediately
- Day 3: Test infrastructure ready
- Day 5: Debug tooling implemented
- Day 7: Full integration and validation

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Debug logging adds performance overhead | Medium | Make it opt-in via env var, benchmark to ensure <1% overhead when disabled |
| Documentation goes stale as parser evolves | Medium | Add CI check that verifies CLAUDE.md examples still compile |
| Test helpers don't cover all cases | Low | Start with common cases, expand based on usage feedback |
| Developers don't read documentation | Medium | Make errors point to relevant CLAUDE.md sections |
| Debug output too verbose/noisy | Low | Make output format configurable, add filtering options |

## References

- M-TESTING Sprint Days 1-2 experience (source of pain points)
- M-DX1: Developer Experience milestone (design_docs/implemented/v0_3_10/M-DX1_developer_experience.md)
- Parser implementation: `internal/parser/`
- AST definitions: `internal/ast/ast.go`
- Lexer implementation: `internal/lexer/lexer.go`

## Future Work

**Phase 2 improvements (v0.4.0+):**
- Parser visualization tool (show token stream graphically)
- AST diff tool (compare before/after for refactoring)
- Parser fuzzing (generate random valid inputs)
- Automated parser test generation from grammar
- Parser performance profiler (identify slow parse paths)

**Enhanced debug mode (v0.4.0+):**
- `DEBUG_PARSER=verbose` for full AST dump
- `DEBUG_PARSER=filter=Expression` to focus on specific functions
- Integration with `ailang debug ast` command

**Better error messages (v0.4.0+):**
- Parser errors include "did you mean" suggestions
- Errors reference CLAUDE.md documentation sections
- Multi-line error context (show surrounding code)

---

**Document created**: 2025-10-26
**Last updated**: 2025-10-26
