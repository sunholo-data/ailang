# M-PARSER: Fix Nested Match Expression Regression

**Status**: Planned
**Target**: v0.3.21 (Hotfix)
**Priority**: P0 (Critical - blocks AI code generation)
**Estimated**: 1 day (4h investigation + 2h fix + 2h testing)
**Dependencies**: None

## AI-First Alignment Check

**Score this feature against AILANG's core principles:**

| Principle | Impact | Score | Notes |
|-----------|--------|-------|-------|
| Reduce Syntactic Noise | Restore | +1 | Fixes parser bug that rejects valid code |
| Preserve Semantic Clarity | Restore | +1 | Nested match expressions work as documented |
| Increase Determinism | Restore | +1 | Identical inputs produce identical results again |
| Lower Token Cost | Restore | +1 | AI-generated code compiles without workarounds |
| **Net Score** | | **+4** | **Decision: Move forward immediately (P0 hotfix)** |

**Decision rule:** Net score > +1 → Move forward | ≤ 0 → Reject or redesign

**Reference:** See [AI-first DX philosophy](../v0_3_15/example-parity-vision-alignment.md#-design-principle-ai-first-dx)

## Problem Statement

**Regression introduced in v0.3.20**: Parser reorganization during M-TESTING sprint broke nested match expression parsing, causing **18 benchmark regressions** and **-4.8% AILANG performance drop** (40.0% → 35.2%).

**Current State:**
- AI models generate syntactically correct AILANG code
- Parser rejects valid nested match expressions with `PAR_NO_PREFIX_PARSE` error
- Error message: "unexpected token in expression: }"
- **64 PAR_001 failures** in v0.3.20 eval baseline (0% repair success)
- Self-repair cannot fix parser bugs (models regenerate same valid code)

**Impact:**
- **Who**: All AI models using AILANG (gpt5, claude-sonnet-4-5, gemini-2-5-pro, etc.)
- **Severity**: Critical - blocks core language features (pattern matching, state threading, effect composition)
- **Scope**: 18 benchmarks failing that were passing in v0.3.17
- **User Impact**: AI-generated code won't compile even when syntactically correct

**Root Cause Hypothesis:**
Parser file reorganization in v0.3.20:
- `parser_decl.go` (1085 lines) → split into 5 files
- **`parser_testing.go` (318 lines)**: New inline test/property syntax parsing
- Likely conflict: Test block parsing interfering with match expression delimiter tracking

## Goals

**Primary Goal:** Restore v0.3.17 parser behavior for nested match expressions while preserving M-TESTING property-based testing syntax.

**Success Metrics:**
- All 18 regressed benchmarks pass again (AILANG success rate: 35.2% → 40.0%+)
- PAR_001 failures: 64 → <20 (expected baseline for genuinely invalid code)
- Zero new parser regressions introduced
- All existing parser tests continue passing
- New regression tests prevent future breakage

## Solution Design

### Overview

**Three-phase approach:**
1. **Identify**: Bisect commits to find exact parser change causing regression
2. **Fix**: Restore correct match expression parsing without breaking test syntax
3. **Prevent**: Add comprehensive regression tests for nested constructs

### Architecture

**Root Cause Investigation:**

The parser reorganization split expression parsing across multiple files. Likely issues:

1. **Delimiter Stack Corruption** (`parser_testing.go`):
   ```go
   // Test blocks use { } delimiters
   test "name" = { ... }

   // Match expressions also use { } delimiters
   match x { (a, b) => ... }

   // Hypothesis: Test parser consuming closing } meant for match
   ```

2. **Expression Context Confusion**:
   ```go
   // Match expression arms are expressions, not statements
   match r { (state, _) => println("x") }  // println is expression here
                                            // Not a block statement!
   ```

3. **Prefix Parser Registration**:
   - Match expressions need `{` as valid expression starter
   - Test blocks also use `{` as starter
   - Possible conflict in prefix parser table

**Components:**
1. **Diagnostic Tooling**: Add parser debug mode to trace delimiter stack
2. **Parser Fix**: Restore match expression delimiter handling
3. **Regression Tests**: Prevent future breakage

### Implementation Plan

**Phase 1: Bisect & Diagnose** (~4 hours)
- [ ] Create minimal reproduction case for nested match parsing
- [ ] Bisect commits between v0.3.17 and v0.3.20
  ```bash
  git bisect start v0.3.20 v0.3.17
  # Test script: ailang run examples/nested_match.ail
  ```
- [ ] Add `DEBUG_PARSER=1` mode to trace delimiter stack
- [ ] Compare v0.3.17 vs v0.3.20 parse trees for same code
- [ ] Identify exact line/function causing regression

**Phase 2: Fix Parser** (~2 hours)
- [ ] Restore correct match expression parsing
- [ ] Ensure test block parsing remains functional
- [ ] Add delimiter context tracking to distinguish test vs match
- [ ] Verify fix with minimal reproduction case
- [ ] Run full parser test suite

**Phase 3: Regression Prevention** (~2 hours)
- [ ] Add `TestNestedMatchExpressions` test
- [ ] Add `TestMatchWithinTestBlock` test (cross-feature)
- [ ] Add failing example from eval baseline as test case
- [ ] Create `examples/nested_match_comprehensive.ail`
- [ ] Update parser documentation with edge cases
- [ ] Re-run v0.3.20 eval baseline subset (18 regressed benchmarks)

### Files to Modify/Create

**New files:**
- `internal/parser/parser_match_test.go` - Nested match regression tests (~200 LOC)
- `examples/nested_match_comprehensive.ail` - Comprehensive match examples (~100 LOC)

**Modified files:**
- `internal/parser/parser_testing.go` - Fix delimiter handling (~10 LOC change)
  - OR `internal/parser/parser_expr.go` - If issue is in match parsing itself
- `internal/parser/parser.go` - Add DEBUG_PARSER mode (~30 LOC)

## Examples

### Example 1: Failing Code (v0.3.20)

**Current (BROKEN):**
```ailang
// From explicit_state_threading benchmark
export func main() -> () ! {IO} {
  let state0 = 0;
  println("Initial: " ++ show(state0));
  let r1 = add(5, state0);
  match r1 {
    (state1, _) => {
      println("After add: " ++ show(state1));
      let r2 = multiply(3, state1);
      match r2 {
        (state2, _) => {
          println("After multiply: " ++ show(state2));
          let r3 = add(10, state2);
          match r3 {
            (state3, _) => println("After add: " ++ show(state3))
          }
        }
      }
    }
  }
}
```

**Error:**
```
PAR_NO_PREFIX_PARSE at line 33:3: unexpected token in expression: }
Suggestion: Check for unmatched delimiters or missing expression
```

**After Fix (WORKING):**
Same code compiles successfully. Parser correctly tracks nested match expression delimiters.

### Example 2: Test Block + Match (Edge Case)

**Ensure this still works:**
```ailang
test "pattern matching with nested match" = {
  let result = match Some(42) {
    Some(x) => match Some(x + 1) {
      Some(y) => y
      None => 0
    }
    None => 0
  };
  result == 43
}
```

**Both test blocks AND nested matches must work together.**

### Example 3: Complex Nesting

**Create comprehensive example:**
```ailang
// examples/nested_match_comprehensive.ail
module nested_match_comprehensive

type Result = | Ok(int) | Err(string)
type Option = | Some(int) | None

export func complexNesting(r: Result) -> int {
  match r {
    Ok(x) => {
      let opt = if x > 0 then Some(x) else None;
      match opt {
        Some(y) => match Some(y * 2) {
          Some(z) => z
          None => 0
        }
        None => -1
      }
    }
    Err(msg) => 0
  }
}
```

## Success Criteria

- [ ] Minimal reproduction case passes (nested match expression compiles)
- [ ] All 18 regressed benchmarks pass again
  - explicit_state_threading (gpt5, gpt5-mini)
  - effect_composition (gpt5)
  - effect_tracking_io_fs (claude-sonnet-4-5, claude-haiku-4-5)
  - higher_order_functions (claude-sonnet-4-5)
  - immutable_data_structures (gemini-2-5-pro)
  - nested_records (gpt5)
  - recursion_fibonacci (gemini-2-5-flash)
  - referential_transparency (gpt5, claude-sonnet-4-5)
  - error_handling (gpt5)
  - string_manipulation (gpt5-mini)
  - targeted_repair_test (gpt5-mini, gemini-2-5-flash)
  - pattern_matching_complex (gpt5)
  - print_missing_effect (claude-sonnet-4-5)
  - records_person (gpt5-mini)
- [ ] All existing parser tests passing (zero new regressions)
- [ ] M-TESTING property-based testing syntax still works
- [ ] New regression tests added and passing
- [ ] Documentation updated with edge cases
- [ ] Examples added to verify fix

## Testing Strategy

**Unit tests:**
- `TestNestedMatchExpressions`: 3-level nesting, various arm types
- `TestMatchWithinTestBlock`: Test blocks containing match expressions
- `TestMatchWithinMatchArm`: Match expression as arm result
- `TestComplexDelimiterNesting`: Mix of {}, [], () in nested matches

**Integration tests:**
- Run all 18 regressed benchmarks with fix
- Verify AILANG success rate: 35.2% → 40.0%+
- Check PAR_001 count: 64 → <20

**Manual testing:**
- `make verify-examples` - Ensure all examples still work
- REPL test: Paste nested match expression, verify it parses
- Property-based tests: Run `ailang test examples/testing_*.ail`

## Non-Goals

**Not in this hotfix:**
- Parser refactoring or cleanup - Keep changes minimal for hotfix
- Improved error messages - Separate feature (M-DX series)
- Parser fuzzing infrastructure - Deferred to v0.4.0
- Test/property syntax enhancements - Keep existing behavior

## Timeline

**Day 1** (8 hours):
- Phase 1: Bisect & diagnose (4h)
- Phase 2: Implement fix (2h)
- Phase 3: Regression tests (2h)

**Hotfix release same day:**
- Run quick eval baseline (dev models only, ~5 min)
- Verify 18 benchmarks pass
- Tag v0.3.21
- Update dashboard

**Total: ~8 hours in 1 day (hotfix urgency)**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Fix breaks test syntax | High | Comprehensive tests for test blocks + matches together |
| Can't bisect (too many commits) | Med | Manual code review of parser_testing.go changes |
| Multiple independent bugs | High | Fix one at a time, test incrementally |
| Performance regression from fix | Low | Benchmark parser performance before/after |

## References

- **Regression Analysis**: See earlier analysis in this conversation
- **Eval Compare**: `ailang eval-compare eval_results/baselines/0.3.17 eval_results/baselines/0.3.20`
- **Example Failure**: `eval_results/baselines/0.3.20/explicit_state_threading_ailang_gpt5_1761508360.json`
- **M-TESTING Sprint**: Property-based testing implementation (v0.3.20)
- **Parser Organization**: [Code organization principles](../../CLAUDE.md#code-organization-principles-ai-first-design)

## Future Work

**Post-hotfix improvements (v0.3.22+):**
- Parser fuzzing to catch regressions early
- Improved PAR_001 error messages with delimiter tracking visualization
- Parser refactoring to separate test/expression contexts cleanly
- Automated parser regression detection in CI

---

**Document created**: 2025-10-26
**Last updated**: 2025-10-26
