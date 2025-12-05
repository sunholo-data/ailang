# Sprint Plan: S-CONS Pattern Sugar (v0.4.3)

## Summary

Extend parser to allow `x :: xs` syntax in pattern matching, eliminating 36 PAR_001 errors (12% of parse failures) and achieving syntax parity with expressions.

**Duration:** 2 days (~5 hours total)
**Dependencies:** None (self-contained parser change)
**Risk Level:** Low (parser-only, well-defined scope, comprehensive test plan)

## Current Status Analysis

### Completed Recently (v0.4.0-v0.4.3)
- ✅ **String Parsing Builtins** (v0.4.3): ~370 LOC in 1 day
- ✅ **S-CALL0 Hotfix** (v0.4.2): ~50 LOC in 0.5 days
- ✅ **Surface Sugar Pack** (v0.4.1): ~400 LOC in 2 days
- ✅ **Agent Execution System** (v0.4.5): ~1500 LOC in 6 phases

### Velocity
- **Recent average**: ~250-400 LOC/day (implementation + tests)
- **Estimated capacity**: ~200 LOC total for this sprint (small, focused change)
- **Pattern**: Recent parser changes (S-CALL0, S-CONS expression sugar) took 0.5-1 day each

### Current Implementation Status
- ✅ **CONS token exists**: `lexer.DCOLON` (`::`), already lexed
- ✅ **Expression sugar exists**: `parseConsExpression()` handles `x :: xs` in expressions
- ❌ **Pattern sugar missing**: Match patterns still require verbose `::(x, xs)` form
- ✅ **Strict mode flag exists**: `--strict-syntax` flag infrastructure present

### Gap Analysis
**Design doc estimate**: 5 hours across 2 days
- Phase 1: Parser extension (2h)
- Phase 2: Strict mode (30min)
- Phase 3: Testing (2h)
- Phase 4: Documentation (30min)

**Reality check**: This matches recent parser sugar work (S-CALL0 took ~4 hours).

## Proposed Implementation Plan

### Day 1: Parser Extension & Core Tests (3 hours)

#### Morning Session (2 hours): Parser Implementation
**Goal:** Add right-associative `::` loop to pattern parsing

**Tasks:**
1. **Locate pattern parsing code** (15 min)
   - Find `parseMatchExpression()` in `internal/parser/parser.go`
   - Identify where patterns are currently parsed (likely inline within match arms)
   - Study existing expression-level `parseConsExpression()` for reference

2. **Implement pattern cons sugar** (1 hour)
   - Add right-associative `::` loop after base pattern parsing
   - Support: `x :: xs`, `_ :: xs`, `1 :: rest`, `a :: b :: c`
   - Desugar immediately to canonical `CtorPattern(name="::", args=[head, tail])`
   - Ensure right-associativity: `a :: b :: c` → `::(a, ::(b, c))`

   ```go
   // Pseudocode location: internal/parser/parser.go
   // After parsing base pattern (ident, _, literal, ctor, tuple):
   for p.curTokenIs(lexer.DCOLON) { // '::'
       p.nextToken()
       rhs := parsePatternElement() // Recursive call
       pat = &ast.CtorPattern{
           Name: "::",
           Args: []ast.Pattern{pat, rhs},
       }
   }
   ```

3. **Handle edge cases** (30 min)
   - Parenthesized patterns: `(x :: xs)`
   - Wildcards: `(_ :: xs)`, `(x :: _)`
   - Empty list terminator: `x :: []`
   - Nested patterns: `x :: (y :: ys)`

4. **Quick smoke test** (15 min)
   - Run `make test` to ensure no regressions
   - Manual test: Create `test_pattern_sugar.ail` with basic `x :: xs` pattern
   - Verify parser doesn't crash

#### Afternoon Session (1 hour): Core Tests & Strict Mode
**Goal:** Add parser tests and strict mode enforcement

**Tasks:**
1. **Add parser unit tests** (45 min) - `internal/parser/parser_test.go`
   - Test `x :: xs` parses to correct AST (matches `::(x, xs)`)
   - Test right-associativity: `a :: b :: c` → `::(a, ::(b, c))`
   - Test wildcards: `_ :: xs`, `x :: _`
   - Test literals: `1 :: rest`, `"a" :: rest`
   - Test empty list: `x :: []`
   - Test parentheses: `(x :: xs)`, `x :: (y :: ys)`
   - Negative: `(:: x xs)` → parse error
   - Negative: `x ::` at EOF → helpful error

2. **Add strict mode support** (15 min)
   - Check `p.strictSyntaxMode` flag in pattern parsing
   - If enabled and `::` found in pattern → error with message:
     `"Use canonical ::(x, xs) instead of x :: xs in patterns"`
   - Ensure `::(x, xs)` still works in strict mode

**End of Day 1 Acceptance:**
- [ ] Parser accepts `x :: xs` in patterns
- [ ] Desugars to identical AST as `::(x, xs)`
- [ ] Right-associativity works correctly
- [ ] 8+ parser unit tests passing
- [ ] Strict mode errors appropriately
- [ ] `make test` passes (no regressions)
- [ ] `make lint` clean

---

### Day 2: Integration Tests & Documentation (2 hours)

#### Morning Session (1 hour): Integration Testing
**Goal:** Verify end-to-end pipeline works with new syntax

**Tasks:**
1. **Create example file** (15 min) - `examples/pattern_sugar.ail`
   ```ailang
   -- S-CONS pattern sugar example (v0.4.3)
   module examples/pattern_sugar

   def main: !{IO} unit =
     let numbers = [1, 2, 3, 4, 5] in
     let result = match numbers {
       x :: xs => x           -- Sugar form
     | [] => 0
     } in
     print(show(result))
   ```

2. **Add integration tests** (30 min) - `internal/pipeline/pipeline_test.go`
   - Full pipeline: parse → elaborate → type check → evaluate
   - Test `match xs { x :: xs => ... }` produces correct value
   - Test mixed forms: `::(x, xs)` and `x :: xs` in same match
   - Test with guards: `x :: xs if p(x) => ...`
   - Verify type inference works identically for both forms

3. **Run verification suite** (15 min)
   ```bash
   make verify-examples    # Ensure new example works
   make test              # All tests pass
   make lint              # No linting issues
   make test-coverage     # Check coverage didn't drop
   ```

#### Afternoon Session (1 hour): Documentation & Release Prep
**Goal:** Update documentation and prepare for release

**Tasks:**
1. **Update teaching prompt** (15 min) - `prompts/v0.4.3.md`
   - Add section on pattern cons sugar under "Pattern Matching"
   - Note that both forms work: `x :: xs` (sugar) and `::(x, xs)` (canonical)
   - Mention strict mode behavior
   - Add to "Common Mistakes" if applicable

2. **Update guides** (15 min) - `docs/docs/guides/pattern-matching.md`
   - Add examples showing both syntax forms
   - Explain when to use which (strict mode vs normal)
   - Update "Pattern Syntax" section with cons sugar

3. **Update CHANGELOG.md** (15 min)
   ```markdown
   ## [v0.4.3] - 2025-11-12 (UNRELEASED)

   ### Added - S-CONS Pattern Sugar

   **User Impact**: Use natural `x :: xs` syntax in patterns (like OCaml/Haskell/F#) instead of verbose `::(x, xs)`.

   **What's New:**
   - Pattern matching now supports `x :: xs` cons syntax (desugars to `::(x, xs)`)
   - Right-associative: `a :: b :: c` means `::(a, ::(b, c))`
   - Works with wildcards, literals, guards, and nested patterns
   - In `--strict-syntax` mode, only canonical `::(x, xs)` allowed
   - Fixes 36 PAR_001 parse errors (12% reduction in failures)

   **Code Changes:**
   - Parser: ~20 LOC in `internal/parser/parser.go`
   - Tests: ~150 LOC in `internal/parser/parser_test.go`
   - Tests: ~50 LOC in `internal/pipeline/pipeline_test.go`
   - Docs: Updated prompts and guides (~30 LOC)
   - Example: `examples/pattern_sugar.ail` (~15 LOC)

   **Total:** ~265 LOC (implementation + tests + docs + examples)
   ```

4. **Final verification** (15 min)
   ```bash
   make ci                # Run full CI suite locally
   make verify-examples   # All examples work
   git status            # Review all changes
   ```

**End of Day 2 Acceptance:**
- [ ] Full pipeline tests passing (parse → eval with new syntax)
- [ ] Example file created and verified working
- [ ] `make verify-examples` succeeds
- [ ] Teaching prompt updated (prompts/v0.4.3.md)
- [ ] Pattern matching guide updated
- [ ] CHANGELOG.md entry added
- [ ] `make ci` passes
- [ ] All tests passing (no regressions)
- [ ] Test coverage ≥38.5% (no decrease)

---

## Success Metrics

### Functionality
- [ ] `x :: xs` parses correctly in all pattern positions
- [ ] Desugars to identical AST as `::(x, xs)` (bijective)
- [ ] Right-associativity works: `a :: b :: c` → `::(a, ::(b, c))`
- [ ] Wildcards work: `_ :: xs`, `x :: _`
- [ ] Guards work: `x :: xs if p(x) => ...`
- [ ] Mixed forms work: `::(x, xs)` and `x :: xs` in same match
- [ ] Strict mode rejects sugar with helpful message

### Testing
- [ ] Parser tests: 8+ test cases covering happy/edge/negative paths
- [ ] Integration tests: 3+ full pipeline tests
- [ ] Example file: `examples/pattern_sugar.ail` works correctly
- [ ] Zero regressions: All existing pattern tests pass
- [ ] Test coverage: ≥38.5% (no decrease)

### Documentation
- [ ] Teaching prompt updated (`prompts/v0.4.3.md`)
- [ ] Pattern matching guide updated (`docs/docs/guides/pattern-matching.md`)
- [ ] CHANGELOG.md entry added with metrics
- [ ] Example file included in examples directory

### Quality
- [ ] `make test` passes
- [ ] `make lint` clean
- [ ] `make ci` passes
- [ ] `make verify-examples` succeeds

### Impact
- [ ] 36 PAR_001 errors eliminated (measured in next eval baseline)
- [ ] Pattern/expression syntax parity achieved
- [ ] AI-first alignment score: +3 (reduces noise, preserves clarity, lowers token cost)

---

## Risks & Mitigations

### Risk 1: Regression in existing pattern matching
**Impact:** High
**Mitigation:**
- Add regression tests for all existing pattern forms before starting
- Run full test suite after each change
- Use `git bisect` if regressions appear
- Keep changes minimal and focused

### Risk 2: Ambiguity with other syntax
**Impact:** Medium
**Mitigation:**
- Parser already handles `::` in expressions; pattern context is unambiguous
- Test extensively with parentheses, tuples, and nested patterns
- Add negative tests for ambiguous cases

### Risk 3: Strict mode breaks existing code
**Impact:** Low
**Mitigation:**
- Strict mode is opt-in; most users unaffected
- Document strict mode behavior clearly
- Ensure `::(x, xs)` canonical form always works

### Risk 4: Performance impact
**Impact:** Low
**Mitigation:**
- Desugaring is O(1) per pattern; no runtime cost
- Happens at parse time, not runtime
- No new allocations beyond existing pattern AST nodes

---

## Dependencies

### Prerequisites (Already Complete)
- ✅ CONS token (`lexer.DCOLON`) exists
- ✅ Expression-level cons sugar (`parseConsExpression()`) implemented
- ✅ Strict mode flag (`--strict-syntax`) infrastructure exists
- ✅ Pattern matching system fully functional

### External Dependencies
- None (self-contained parser change)

### Blocking Items
- None (ready to start immediately)

---

## Open Questions

**None** - Design is clear, implementation is straightforward, and all prerequisites exist.

---

## Notes & Assumptions

### Assumptions
1. **Pattern parsing location**: Patterns are parsed within `parseMatchExpression()` or similar. Will locate exact function on Day 1.
2. **AST representation**: Will reuse existing `CtorPattern` node with `name="::"` (no new AST nodes needed).
3. **Test coverage target**: Maintain current 38.5% coverage (this change is small enough that coverage won't meaningfully increase).
4. **Strict mode behavior**: Mirrors S-CALL0 and other sugar (reject with helpful message suggesting canonical form).

### Context
- This is **pure syntactic sugar** - no semantic changes, no new language features
- Addresses top parse failure category (PAR_001: 12% of failures)
- Low risk: contained to parser, well-tested, clear specification
- High value: improves DX for AI models and ML-family developers

### Follow-up Work (Post-v0.4.3)
- Formatter flag `--format:sugar=cons` to print sugar instead of canonical form (deferred)
- Additional infix operators in patterns (requires broader design, out of scope)
- LSP integration for auto-conversion between forms (low priority, AI-first DX)

---

**Sprint created**: 2025-11-11
**Target release**: v0.4.3
**Estimated completion**: 2 days (~5 hours)
