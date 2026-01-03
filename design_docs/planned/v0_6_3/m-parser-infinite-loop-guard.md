# Parser Infinite Loop Guard

**Status**: Planned
**Target**: v0.6.3
**Priority**: P0 (Critical) - Can consume 67+ GB memory
**Estimated**: 2-4 hours
**Bug ID**: M-PARSER-LOOP

## Problem Statement

The parser enters an infinite loop when encountering unimplemented syntax like:
- `tests [...]` blocks on function declarations
- `test "name" { ... }` top-level test declarations
- `properties [...]` blocks

This causes:
1. Unlimited memory consumption (67+ GB observed)
2. Process killed by OS OOM killer
3. No useful error message

**Stack trace:**
```
parseTestDecl -> parseStatement -> parseExpression (loop)
```

## Root Cause

In `internal/parser/parser_test_decl.go`, when parsing unrecognized syntax:
1. Parser calls `parseStatement()` expecting a valid statement
2. `parseStatement()` calls `parseExpression()`
3. `parseExpression()` doesn't advance tokens on unrecognized input
4. Loop continues indefinitely

## Solution Design

### Option 1: Parser Loop Detection (Recommended)

Add loop detection at the expression/statement parsing level:

```go
// In parser_expr.go
func (p *Parser) parseExpression(precedence int) ast.Expr {
    startPos := p.curToken.Pos

    // Loop detection: if we see same position twice, we're stuck
    if p.lastExprPos == startPos {
        p.errorAtPos(startPos, "PAR_INFINITE_LOOP",
            "parser stuck at position %v - unrecognized syntax", startPos)
        p.nextToken() // Force advance
        return nil
    }
    p.lastExprPos = startPos

    // ... rest of parseExpression
}
```

### Option 2: Default Timeout

Make `--timeout` default to 30s for all CLI commands:

```go
// In cmd/ailang/check.go
var defaultTimeout = 30 * time.Second

func checkFile(...) {
    timeout := opts.Timeout
    if timeout == 0 {
        timeout = defaultTimeout
    }
    // ...
}
```

### Option 3: Memory Limit

Add memory limit via `runtime.SetMemoryLimit()`:

```go
func init() {
    // Limit to 4GB to prevent OOM
    debug.SetMemoryLimit(4 << 30)
}
```

## Implementation Plan

**Phase 1: Loop Detection** (1-2 hours)
- [ ] Add `lastExprPos` field to Parser struct
- [ ] Check for position stall in `parseExpression()`
- [ ] Check for position stall in `parseStatement()`
- [ ] Emit clear error message with position
- [ ] Force advance token to break out of loop

**Phase 2: Default Timeout** (30 min)
- [ ] Add default 30s timeout to `ailang check`
- [ ] Add default 30s timeout to `ailang run`
- [ ] Document in `--help`

**Phase 3: Testing** (1 hour)
- [ ] Add test for `tests [...]` syntax error
- [ ] Add test for `test "name" {...}` syntax error
- [ ] Add test for loop detection recovery
- [ ] Verify aspirational examples fail fast

## Success Criteria

- [ ] `ailang check examples/experimental/factorial.ail` fails within 1 second
- [ ] Clear error message: "unrecognized syntax at line X"
- [ ] No infinite memory growth
- [ ] All existing tests pass

## Files to Modify

- `internal/parser/parser.go` - Add lastExprPos field
- `internal/parser/parser_expr.go` - Add loop detection
- `internal/parser/parser_test_decl.go` - Add loop detection
- `cmd/ailang/check.go` - Add default timeout
- `cmd/ailang/run.go` - Add default timeout

## Related

- Bug reproduction: `examples/bugs/parser_infinite_loop_on_test_syntax.ail`
- Aspirational examples: `examples/experimental/factorial.ail`
- Existing timeout: `ailang check --timeout 5s` works

---
**Document created**: 2026-01-03
