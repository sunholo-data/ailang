# AILANG v0.6.3 Bug Fixes

**Status**: Planned
**Target**: v0.6.3
**Priority**: P0/P1 - Critical bugs affecting usability and safety
**Total Estimated**: 8-12 hours

## Overview

This release focuses on fixing two critical bugs discovered during the v0.6.2 examples audit:

| Bug ID | Severity | Description | Est. Time |
|--------|----------|-------------|-----------|
| M-PARSER-LOOP | P0 Critical | Parser infinite loop consumes 67GB+ memory | 2-4h |
| M-CONCAT-INFERENCE | P1 High | `++` operator defaults to string concat for lists | 4-8h |

## Bug 1: Parser Infinite Loop (M-PARSER-LOOP)

**Severity**: P0 Critical - Can crash system with OOM

### Problem

The parser enters an infinite loop when encountering unimplemented syntax:
- `tests [...]` blocks on function declarations
- `test "name" { ... }` top-level test declarations
- `properties [...]` blocks

**Impact**:
- Consumes 67+ GB memory before OS kills process
- No error message - silent death
- Blocks development on aspirational examples

**Reproduction**:
```bash
# DO NOT RUN WITHOUT TIMEOUT
ailang check --timeout 5s examples/experimental/factorial.ail
# Shows: TIMEOUT after 5s with stack dump
```

**Stack trace**:
```
parseTestDecl -> parseStatement -> parseExpression (infinite loop)
```

### Root Cause

In `internal/parser/parser_test_decl.go`:
1. Parser calls `parseStatement()` expecting a valid statement
2. `parseStatement()` calls `parseExpression()`
3. `parseExpression()` doesn't advance tokens on unrecognized input
4. Control returns to step 1 without progress → infinite loop

### Solution

**Option A: Loop Detection** (Recommended)
```go
// Add to Parser struct
type Parser struct {
    // ...
    lastExprPos token.Pos  // Track last expression position
}

// In parseExpression()
func (p *Parser) parseExpression(precedence int) ast.Expr {
    startPos := p.curToken.Pos

    // Loop detection
    if p.lastExprPos == startPos && p.loopCount > 0 {
        p.errorAtPos(startPos, "PAR_LOOP_DETECTED",
            "parser stuck - unrecognized syntax at %v", startPos)
        p.nextToken() // Force advance to break loop
        return nil
    }
    p.lastExprPos = startPos
    p.loopCount++
    defer func() { p.loopCount-- }()

    // ... rest of parseExpression
}
```

**Option B: Default Timeout**
```go
// In cmd/ailang/check.go and run.go
const defaultTimeout = 30 * time.Second
```

### Implementation Plan

- [ ] Add `lastExprPos` and `loopCount` fields to Parser struct
- [ ] Add loop detection in `parseExpression()`
- [ ] Add loop detection in `parseStatement()`
- [ ] Emit clear error: "unrecognized syntax at line X"
- [ ] Force token advance to break out of loop
- [ ] Add default 30s timeout to CLI commands
- [ ] Add test cases for each unimplemented syntax type
- [ ] Verify aspirational examples fail fast with clear errors

### Files to Modify

- `internal/parser/parser.go` - Add loop detection fields
- `internal/parser/parser_expr.go` - Add loop detection logic
- `internal/parser/parser_test_decl.go` - Add loop detection logic
- `cmd/ailang/check.go` - Add default timeout
- `cmd/ailang/run.go` - Add default timeout

### Success Criteria

- [ ] `ailang check examples/experimental/factorial.ail` fails within 1 second
- [ ] Clear error message pointing to problematic syntax
- [ ] No memory explosion - stays under 100MB
- [ ] All existing parser tests pass

---

## Bug 2: `++` Operator Type Inference (M-CONCAT-INFERENCE)

**Severity**: P1 High - Blocks common list operations

### Problem

The `++` operator incorrectly defaults to string concatenation when both operands have list types in simple expression contexts:

```ailang
-- FAILS: "cannot unify list[int] with string"
pure func concatLists(a: [int], b: [int]) -> [int] = a ++ b

-- WORKS: (same operator in match context)
pure func quicksort(list: [int]) -> [int] =
  match list {
    [pivot, ...rest] => {
      quicksort(less) ++ [pivot] ++ quicksort(greater)  -- Works!
    }
  }
```

**Error message**:
```
type unification failed at [string concat (default)]: cannot unify list[int] with string
```

**Impact**:
- Blocks simple list utility functions
- Forces use of `std/list.concat` instead of `++`
- Inconsistent behavior between expression contexts

### Root Cause

In `internal/types/typechecker_operators.go`, the `++` operator resolution:

```go
// Current decision tree for ++:
// 1. If at least one is a concrete list → list concat
// 2. If at least one is a concrete string → string concat
// 3. If both are type variables → ??? (this is where it goes wrong)
// 4. Otherwise → string concat (fallback)  ← BUG: wrong default
```

The issue is that in a simple function expression like `a ++ b` where `a` and `b` have **known list types** from the signature, the type checker isn't propagating that context to the operator resolution.

### Solution

**Option A: Context-Aware Inference** (Recommended)

Use the expected return type from the enclosing function signature:

```go
case "++":
    // Check expected type from context first
    expectedType := ctx.getExpectedType()

    if isListType(leftType) || isListType(rightType) {
        // At least one operand is a list → list concat
        resultType = inferListConcatType(leftType, rightType)
    } else if expectedType != nil && isListType(expectedType) {
        // Return type is list → list concat
        resultType = expectedType
        ctx.addConstraint(TypeEq{Left: leftType, Right: expectedType})
        ctx.addConstraint(TypeEq{Left: rightType, Right: expectedType})
    } else if isStringType(leftType) || isStringType(rightType) {
        // At least one operand is string → string concat
        resultType = TString
    } else {
        // Default to string (most common case)
        resultType = TString
    }
```

**Option B: Check Operand Types More Carefully**

The operands `a: [int]` and `b: [int]` ARE concrete list types from the function signature. The issue may be that they're not being recognized as concrete.

```go
// Add better list type detection
func isListType(t Type) bool {
    switch t := t.(type) {
    case *TList:
        return true
    case *TVar:
        // Check if this type variable has been unified with a list
        if resolved := t.Resolved(); resolved != nil {
            return isListType(resolved)
        }
    }
    return false
}
```

### Implementation Plan

**Phase 1: Diagnosis** (1 hour)
- [ ] Add debug logging to `++` operator resolution
- [ ] Trace what types are seen for `a` and `b` in failing case
- [ ] Compare with working match context case
- [ ] Identify exact point of divergence

**Phase 2: Fix** (2-3 hours)
- [ ] Implement chosen solution (A or B)
- [ ] Add type context threading if needed
- [ ] Ensure list concat is chosen for list operands

**Phase 3: Testing** (2 hours)
- [ ] Unit test: recursive string concat still works
- [ ] Unit test: simple list concat works
- [ ] Unit test: polymorphic list concat works
- [ ] Integration test: `bugs/list_concat_match.ail` compiles
- [ ] Regression test: all existing `++` tests pass

### Files to Modify

- `internal/types/typechecker_operators.go` - Fix `++` operator logic
- `internal/types/typechecker.go` - Add expected type context (if needed)
- `internal/types/context.go` - Add getExpectedType() (if needed)

### Success Criteria

- [ ] `pure func concat(a: [int], b: [int]) -> [int] = a ++ b` compiles
- [ ] `bugs/list_concat_match.ail` compiles and runs correctly
- [ ] `deterministic_list_transform` benchmark passes
- [ ] All existing string concat tests pass (no regression)

---

## Testing Strategy

### Unit Tests

```go
func TestConcatOperatorInference(t *testing.T) {
    tests := []struct {
        name        string
        code        string
        shouldError bool
    }{
        {
            name: "simple list concat",
            code: `pure func concat(a: [int], b: [int]) -> [int] = a ++ b`,
            shouldError: false,
        },
        {
            name: "recursive string concat",
            code: `func join(xs: [int]) -> string = match xs { [] => "", x::r => show(x) ++ join(r) }`,
            shouldError: false,
        },
        {
            name: "mixed types should error",
            code: `func bad(s: string, xs: [int]) = s ++ xs`,
            shouldError: true,
        },
    }
    // ...
}

func TestParserLoopDetection(t *testing.T) {
    tests := []struct {
        name string
        code string
    }{
        {name: "test block", code: `test "foo" { true }`},
        {name: "tests annotation", code: `func f(n: int) tests [(0, 1)] { n }`},
        {name: "properties annotation", code: `func f(n: int) properties [forall(x) => x > 0] { n }`},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            start := time.Now()
            _, err := parser.Parse(tt.code)
            elapsed := time.Since(start)

            if elapsed > 100*time.Millisecond {
                t.Errorf("parser took too long: %v (possible infinite loop)", elapsed)
            }
            if err == nil {
                t.Error("expected parse error for unimplemented syntax")
            }
        })
    }
}
```

### Integration Tests

```bash
# Parser loop detection
ailang check examples/experimental/factorial.ail  # Should fail fast with clear error

# List concat
ailang check examples/bugs/list_concat_match.ail  # Should compile after fix

# Regression
make test  # All existing tests pass
```

---

## Timeline

**Day 1** (4-5 hours):
- M-PARSER-LOOP: Implement loop detection
- Add default timeout to CLI
- Test aspirational examples fail fast

**Day 2** (4-5 hours):
- M-CONCAT-INFERENCE: Diagnose root cause
- Implement fix
- Test list concat scenarios

**Day 3** (2 hours):
- Full regression testing
- Documentation updates
- Move bugs from `broken` to `working` in manifest

**Total: ~10-12 hours across 2-3 days**

---

## Related Files

### Bug Reproductions
- `examples/bugs/list_concat_match.ail`
- `examples/bugs/concat_operator_list_inference.ail`
- `examples/bugs/parser_infinite_loop_on_test_syntax.ail`

### Aspirational Examples (trigger parser loop)
- `examples/experimental/factorial.ail`
- `examples/experimental/concurrent_pipeline.ail`
- `examples/experimental/web_api.ail`
- `examples/experimental/ai_agent_integration.ail`

### Design Docs
- `design_docs/planned/v0_6_3/m-parser-infinite-loop-guard.md`
- `design_docs/planned/v0_6_3/concat-operator-type-inference-bug.md`

---

## Success Metrics

After v0.6.3:
- [ ] Zero parser infinite loops on any .ail file
- [ ] `++` works consistently for both strings and lists
- [ ] All 98 working examples still work
- [ ] 3 broken examples → 1 or 0 broken (concat bugs fixed)
- [ ] 4 aspirational examples fail fast with clear errors
- [ ] Coverage ≥ 95% for working examples

---

**Document created**: 2026-01-03
**Last updated**: 2026-01-03
