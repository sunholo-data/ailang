# Sprint Plan: M-PARSER-LOOP - Parser Infinite Loop Guard

## Summary
Fix the P0 critical bug where the parser enters infinite loops on unrecognized syntax (67+ GB memory consumption), by adding loop detection at the expression parsing level and enabling default timeouts.

**Duration:** 1 day (3-4 hours implementation)
**Dependencies:** None - self-contained parser fix
**Risk Level:** Low - parser changes are well-scoped, tests exist

## Current Status Analysis

### Root Cause Identified
The actual bug is in `parser_test_decl.go:39`:
```go
if p.curTokenIs(lexer.NEWLINE) {
    p.nextToken()
    continue
}
```

**The lexer NEVER generates NEWLINE tokens** (it skips `\n` as whitespace in `skipWhitespace()`). This condition is always false, so `parseStatement()` keeps getting called on the same token position, never advancing.

### Additional Issue
The `parseStatement()` function at line 138 calls `p.nextToken()` after parsing, but if `parseExpression(LOWEST)` returns `nil` (no prefix parser found), control returns at line 136 WITHOUT advancing - causing the infinite loop.

### Completed Recently
- v0.6.2: Telemetry with OpenTelemetry (~500 LOC)
- v0.6.2: std/list sortBy, take, drop (~150 LOC)
- v0.6.2: std/string contains() (~50 LOC)

### Velocity
- Recent average: 200-300 LOC/day for parser-related changes
- Estimated capacity: 300-400 LOC for this sprint

### Bug Reproduction
- File: `examples/bugs/parser_infinite_loop_on_test_syntax.ail`
- Aspirational: `examples/experimental/factorial.ail`

## Proposed Milestones

### Milestone 1: Fix NEWLINE Token Bug (Direct Fix)
**Goal:** Remove the dead code checking for NEWLINE tokens
**Estimated:** 20 LOC implementation + 50 LOC tests = 70 LOC
**Duration:** 30 minutes

**Tasks:**
1. Remove NEWLINE checks from `parser_test_decl.go` (lines 39-42, 51, 96-98, 107-109)
2. Remove `skipNewlinesAndComments()` calls if present
3. Fix `parseStatement()` to advance token even on `nil` expression

**Acceptance Criteria:**
- [ ] No `lexer.NEWLINE` references in parser_test_decl.go
- [ ] `parseStatement()` always advances (or returns with clear error)
- [ ] All existing tests pass

### Milestone 2: Add Loop Detection Guard
**Goal:** Add position-based loop detection to prevent ANY future infinite loops
**Estimated:** 80 LOC implementation + 100 LOC tests = 180 LOC
**Duration:** 1 hour

**Tasks:**
1. Add `lastExprPos` field to Parser struct in `parser.go`
2. Add loop detection at start of `parseExpression()` in `parser_expr.go`
3. Emit clear error: "PAR_INFINITE_LOOP: parser stuck at position X - unrecognized syntax"
4. Force advance token to break out

**Implementation:**
```go
// In parser.go - Parser struct
type Parser struct {
    // ... existing fields
    lastExprPos ast.Pos  // Loop detection
}

// In parser_expr.go - parseExpression
func (p *Parser) parseExpression(precedence int) ast.Expr {
    startPos := p.curPos()

    // Loop detection: if we see same position twice, we're stuck
    if p.lastExprPos == startPos && startPos.Line > 0 {
        p.report("PAR_INFINITE_LOOP",
            fmt.Sprintf("parser stuck at position %d:%d - unrecognized syntax",
                startPos.Line, startPos.Column),
            "Check for unimplemented syntax (tests [...], properties [...], etc.)")
        p.nextToken() // Force advance to prevent infinite loop
        return nil
    }
    p.lastExprPos = startPos

    // ... rest of parseExpression
}
```

**Acceptance Criteria:**
- [ ] Parser detects position stall
- [ ] Clear error message includes position
- [ ] Token advances to break loop
- [ ] Test for loop detection works

### Milestone 3: Add Default Timeout
**Goal:** Make --timeout default to 30s for `ailang check` and `ailang run`
**Estimated:** 40 LOC implementation + 30 LOC tests = 70 LOC
**Duration:** 30 minutes

**Tasks:**
1. Add default timeout constant in `cmd/ailang/check.go`
2. Add default timeout constant in `cmd/ailang/run.go`
3. Update `--help` text to document default
4. Allow `--timeout 0` to disable default

**Acceptance Criteria:**
- [ ] `ailang check file.ail` times out after 30s by default
- [ ] `ailang run file.ail` times out after 30s by default
- [ ] `--timeout 0` disables timeout
- [ ] `--help` shows default value

### Milestone 4: Testing and Verification
**Goal:** Verify all fixes and add regression tests
**Estimated:** 50 LOC tests = 50 LOC
**Duration:** 30 minutes

**Tasks:**
1. Add test for `tests [...]` syntax error (fast fail)
2. Add test for `test "name" {...}` syntax error (fast fail)
3. Add test for loop detection recovery
4. Verify `examples/experimental/factorial.ail` fails fast with clear message
5. Verify `examples/bugs/parser_infinite_loop_on_test_syntax.ail` fails fast

**Acceptance Criteria:**
- [ ] `ailang check examples/experimental/factorial.ail` completes within 1 second
- [ ] Clear error message shows line/column position
- [ ] No memory growth (process stays under 100MB)
- [ ] All existing parser tests pass
- [ ] `make test` passes
- [ ] `make lint` passes

## Success Metrics
- Test coverage: Parser tests include loop detection
- Bug examples: Both fail fast with clear errors
- Performance: Parsing fails within 1 second (not 67GB memory)
- All tests passing: `make test`
- All linting passing: `make lint`

## Files to Modify
1. `internal/parser/parser.go` - Add `lastExprPos` field
2. `internal/parser/parser_expr.go` - Add loop detection
3. `internal/parser/parser_test_decl.go` - Remove NEWLINE checks, fix parseStatement
4. `cmd/ailang/check.go` - Add default timeout
5. `cmd/ailang/run.go` - Add default timeout
6. `internal/parser/parser_test_decl_test.go` - Add tests (new file)

## Dependencies
- None - self-contained fix

## Open Questions
- None - design doc provides clear direction

## Notes
- The root cause is simpler than the design doc suggested - it's the dead NEWLINE token checks
- Loop detection is still valuable as a safety net for future parser issues
- Memory limit (Option 3 from design) is NOT implemented - timeout is sufficient
