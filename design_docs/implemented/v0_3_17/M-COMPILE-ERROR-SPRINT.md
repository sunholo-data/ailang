# Sprint Plan: M-COMPILE-ERROR - Enhanced Error Messages for AI Code Generation

## Status: ✅ COMPLETE (v0.3.17 + Follow-up)

**Completed**: October 2025 (v0.3.17 base + loader fix Oct 22)
**Actual Duration**: ~8 hours (Phase 1: 6h, Phase 2: 2h)
**Success**: ✅ All code implemented, ✅ Tests passing, ⏳ Eval validation pending

## Summary
Implement enhanced parser error messages to guide AIs away from Python/JavaScript syntax patterns when writing HTTP/JSON code in AILANG. This sprint targets a 50pp improvement in the `api_call_json` benchmark success rate (75% failure → <25% failure).

**Original Estimate:** 1.5 days (12 hours)
**Dependencies:** Teaching prompt updates (✅ already completed in v0.3.17+)
**Risk Level:** Low (parser error improvements, no language changes)

## Current Status Analysis

### Completed Recently
- ✅ **v0.3.17**: Lambda DX fixes (~520 LOC including tests) in 2.5 hours
- ✅ **v0.3.16**: Lambda examples refactor (~297 LOC examples) in ~3 hours
- ✅ **Prompt updates**: HTTP/JSON examples added to teaching prompt (v0.3.17+)

### Velocity
- Recent average: **~150-200 LOC/day** (based on M-DX3: 520 LOC in 2.5 hours)
- Fast implementation pace for focused milestones
- Strong test coverage habit (100% for new code)

### Problem Status
- **Current**: 75% failure rate on `api_call_json` benchmark (3/4 models fail)
- **Root cause**: AIs generate Python/JS syntax (namespace imports, `const`, bare assignment)
- **Phase 1 (teaching prompt)**: ✅ Complete (v0.3.17+ includes HTTP examples)
- **Phase 2 (enhanced errors)**: ⏳ This sprint
- **Phase 3 (validation)**: ⏳ This sprint

### Remaining from Design Doc
- ⏳ **Enhanced parser error messages** (~230 LOC estimated)
- ⏳ **Eval baseline validation** (2 hours estimated)

## Proposed Milestones

### Milestone 1: Enhanced Parser Error Detection (Day 1 Morning, 4 hours)
**Goal:** Detect common Python/JS patterns and provide actionable suggestions

**Estimated:** 80 LOC implementation + 150 LOC tests = 230 LOC total

**Tasks:**
- **Hour 1-2**: Add `SuggestionError` type to `internal/errors/parser_errors.go`
  - New error type with `Suggestions []string` and `HelpURL string` fields
  - Pretty-print format that shows user code + suggestions
  - Unit tests for error formatting

- **Hour 3-4**: Detect wrong patterns in `internal/parser/parser.go`
  - Pattern 1: `import X from 'Y'` → Suggest `import std/net (httpRequest)`
  - Pattern 2: `const` keyword → Suggest `let ... in`
  - Pattern 3: Bare assignment `x = y` → Suggest `let x = y in`
  - Unit tests for each detection pattern

**Acceptance Criteria:**
- [ ] `SuggestionError` type implemented with pretty-print formatting
- [ ] Parser detects all 3 wrong patterns (namespace imports, const, bare assignment)
- [ ] Error messages include correct AILANG syntax examples
- [ ] Unit tests cover all detection paths (>100% coverage for new code)
- [ ] All existing tests still passing

**Risks:**
- **Risk**: Parser changes could break existing error messages
  - **Mitigation**: Run `make test` after each pattern detection added
- **Risk**: False positives on legitimate syntax
  - **Mitigation**: Only trigger on exact pattern matches (e.g., `from` keyword after import)

**Files Modified:**
- `internal/errors/parser_errors.go` (~30 LOC new)
- `internal/parser/parser.go` (~50 LOC new)
- `internal/parser/parser_errors_test.go` (~150 LOC new tests)

---

### Milestone 2: Integration Testing & Validation (Day 1 Afternoon, 2 hours)
**Goal:** Verify enhanced errors work end-to-end with real examples from eval failures

**Estimated:** 0 LOC implementation + 50 LOC integration tests = 50 LOC total

**Tasks:**
- **Hour 1**: Test with actual eval failure examples
  - Create `internal/parser/integration_test.go`
  - Test Error 1: `import http from 'http'` → Shows std/net suggestion
  - Test Error 2: `const URL = "..."` → Shows let suggestion
  - Test Error 3: `url = "..."` → Shows let binding suggestion
  - Verify error messages match design doc format

- **Hour 2**: CLI integration testing
  - Test `echo 'import http from "http"' | ailang check -`
  - Verify suggestion appears in terminal output
  - Test with all 3 failure examples from design doc
  - Document in CHANGELOG.md

**Acceptance Criteria:**
- [ ] Integration tests pass for all 3 eval failure patterns
- [ ] CLI output shows suggestions correctly formatted
- [ ] Error messages are helpful and actionable
- [ ] CHANGELOG.md updated with M-COMPILE-ERROR entry
- [ ] All tests passing (`make test`)

**Risks:**
- **Risk**: Error formatting breaks in CLI vs test harness
  - **Mitigation**: Test both programmatic API and CLI output

**Files Modified:**
- `internal/parser/integration_test.go` (~50 LOC new)
- `CHANGELOG.md` (~30 LOC new entry)

---

### Milestone 3: Eval Baseline Validation (Day 2 Morning, 4 hours)
**Goal:** Measure improvement on `api_call_json` benchmark with enhanced errors + updated prompt

**Estimated:** 0 LOC implementation (validation only)

**Tasks:**
- **Hour 1**: Re-run eval baseline
  ```bash
  # Clear old results
  rm -rf eval_results/api_call_json_*

  # Run with all 3 affected models
  ailang eval-suite \
    --benchmarks api_call_json \
    --models claude-haiku-4-5,gemini-2-5-flash,gpt5-mini
  ```

- **Hour 2**: Analyze results
  ```bash
  # Compare before/after
  ailang eval-compare \
    eval_results/baselines/v0.3.16 \
    eval_results/current

  # Expected: 75% failure → <25% failure
  # Target: 100% success (4/4 passing)
  ```

- **Hour 3**: Document findings
  - Create `design_docs/implemented/v0_3_18/m-compile-error-results.md`
  - Document actual vs expected improvement
  - Analyze any remaining failures
  - Recommend follow-up actions if needed

- **Hour 4**: Update benchmark dashboard (if success rate improves)
  ```bash
  # Only if we hit >75% success rate
  ailang eval-report eval_results/current v0.3.18 --format=json
  ailang eval-report eval_results/current v0.3.18 --format=docusaurus > docs/docs/benchmarks/performance.md
  ```

**Acceptance Criteria:**
- [ ] Eval baseline completed for `api_call_json` benchmark
- [ ] Results compared against v0.3.16 baseline
- [ ] Success rate improved by at least 25pp (75% fail → 50% fail minimum)
- [ ] Target: 100% success rate (4/4 models passing)
- [ ] Results documented in design_docs/implemented/
- [ ] Benchmark dashboard updated if improvement achieved

**Risks:**
- **Risk**: Teaching prompt updates alone may not be enough
  - **Mitigation**: Enhanced errors provide guardrails even if prompt ignored
- **Risk**: AIs may still use wrong syntax despite better errors
  - **Mitigation**: Document actual failure patterns for future prompt refinement

**Files Added:**
- `design_docs/implemented/v0_3_18/m-compile-error-results.md` (~100 LOC)

---

### Milestone 4: Documentation & Release Prep (Day 2 Afternoon, 2 hours)
**Goal:** Complete documentation and prepare for release

**Tasks:**
- **Hour 1**: Update documentation
  - Update README.md with error improvement note (if significant)
  - Verify CHANGELOG.md is complete
  - Move design doc to `design_docs/implemented/v0_3_18/`
  - Add implementation report with metrics

- **Hour 2**: Verification & cleanup
  - Run full test suite: `make test`
  - Run linting: `make lint`
  - Run example verification: `make verify-examples`
  - Verify file sizes: `make check-file-sizes`
  - Final review of CHANGELOG entry

**Acceptance Criteria:**
- [ ] All documentation updated (README, CHANGELOG, design doc moved)
- [ ] All tests passing: `make test`
- [ ] Linting clean: `make lint`
- [ ] Examples verified: `make verify-examples`
- [ ] File sizes acceptable: `make check-file-sizes`
- [ ] Ready for release

**Risks:**
- **Risk**: None (final verification step)

**Files Modified:**
- `README.md` (if significant improvement)
- `CHANGELOG.md` (final polish)

## Success Metrics

### Code Quality
- Test coverage: **100%** for new code (pattern: M-DX3 achieved 100%)
- Total new code: **~230 LOC** implementation + **~200 LOC** tests = **~430 LOC**
- All tests passing: ✅
- All linting passing: ✅

### Benchmark Improvement
- **Before**: api_call_json success rate: 25% (1/4 passing)
- **Target**: api_call_json success rate: >75% (3/4 passing)
- **Stretch**: api_call_json success rate: 100% (4/4 passing)

### Documentation
- CHANGELOG.md updated with M-COMPILE-ERROR entry
- Design doc moved to `design_docs/implemented/v0_3_18/`
- Implementation report with before/after metrics
- Benchmark dashboard updated (if improvement achieved)

## Timeline

### Day 1 (6 hours)
- **Morning (4h)**: Milestone 1 - Enhanced error detection
  - Hour 1-2: SuggestionError type + tests
  - Hour 3-4: Pattern detection + tests
- **Afternoon (2h)**: Milestone 2 - Integration testing
  - Hour 1: Integration tests with eval examples
  - Hour 2: CLI testing + CHANGELOG update

### Day 2 (6 hours)
- **Morning (4h)**: Milestone 3 - Eval validation
  - Hour 1: Re-run eval baseline
  - Hour 2: Analyze results
  - Hour 3: Document findings
  - Hour 4: Update dashboard (conditional)
- **Afternoon (2h)**: Milestone 4 - Documentation & release prep
  - Hour 1: Update docs
  - Hour 2: Final verification

**Total: 12 hours (1.5 days)**

## Dependencies

### Completed Dependencies ✅
- Teaching prompt updates (v0.3.17+) - includes HTTP/JSON examples
- Standard library (`std/net`, `std/json`) - already implemented

### No Blocking Dependencies
- All work can proceed immediately
- No external dependencies or API changes needed

## Open Questions

### Q1: Should we also detect `import X` (bare import without namespace)?
**Answer**: Yes, this is Pattern 3 from Error 3 (`import http`). Design doc already includes this.

### Q2: What if enhanced errors don't improve eval success rate?
**Answer**: Enhanced errors are still valuable for all users (not just AIs). Even if eval improvement is modest, error messages guide developers to correct syntax. We can iterate on teaching prompt in v0.3.19 if needed.

### Q3: Should we backport enhanced errors to older versions?
**Answer**: No, this is a DX improvement, not a bug fix. Include in next release (v0.3.18).

## Notes

### Assumptions
- Teaching prompt updates (v0.3.17+) already include HTTP/JSON examples
- Enhanced errors will reduce repair iterations (AIs see suggestions, regenerate correct code)
- Combined approach (prompt examples + enhanced errors) targets 50pp improvement

### Context from Recent Work
- **M-DX3** (v0.3.17): Achieved 100% test coverage on 520 LOC in 2.5 hours
- **Fast iteration**: Recent milestones completed in 2-3 hours each
- **Strong test culture**: Always write tests alongside implementation

### Success Pattern
This sprint follows the proven M-DX pattern:
1. **Focused scope**: One clear problem (AI syntax confusion)
2. **Data-driven**: Based on actual eval failures, not speculation
3. **Measurable**: Clear before/after metrics (75% fail → <25% fail)
4. **Incremental**: Build on existing work (teaching prompt already updated)

### Estimated ROI
- **Time investment**: 12 hours (1.5 days)
- **Expected improvement**: +50pp success rate on api_call_json
- **Broader impact**: Better error messages benefit all users, not just AIs
- **Long-term value**: Establishes pattern for future DX improvements

---

## Implementation Summary

### What Was Built

**Phase 1: Enhanced Parser Errors (v0.3.17)** ✅
- `internal/parser/parser_error.go`: Enhanced ParserError with Suggestions field (30 LOC)
- `internal/parser/parser_decl.go`: Detection for 3 patterns (50 LOC):
  1. JavaScript namespace imports: `import X from 'Y'` → IMP012_UNSUPPORTED_NAMESPACE
  2. Const keyword: `const x = y` → PAR_CONST_NOT_SUPPORTED
  3. Bare assignment: `x = y` → PAR_BARE_ASSIGNMENT
- Tests: 470 LOC (suggestion_errors_test.go: 320 LOC, cli_integration_test.go: 150 LOC)
- **All tests passing** ✅

**Phase 2: Loader Fix (Oct 22, commit f5919ed)** ✅
- `internal/loader/loader.go`: Fixed error formatting to iterate and call .Error() on each
- Commit: `f5919ed` "fix: Module loader now shows full error messages with suggestions"
- **Impact**: AIs now see full suggestions during self-repair attempts

### Metrics

- **Code**: 80 LOC implementation (as estimated)
- **Tests**: 470 LOC (vs. 200 estimated, +135% coverage)
- **Time**: ~8 hours (vs. 12 estimated, 33% under budget)
- **Test Coverage**: 100% for new code ✅

### Verification Status

- ✅ **Milestone 1**: Enhanced error detection complete
- ✅ **Milestone 2**: Integration testing complete
- ❌ **Milestone 3**: Eval baseline validation **NOT DONE** (follow-up pending)
- ✅ **Milestone 4**: Documentation complete (moved to implemented/)

### Known Limitations

**Eval validation pending**: The original success metric (75% failure → <25% failure on api_call_json) was NOT measured after the loader fix. This requires:
1. Re-run eval baseline with models: claude-haiku-4-5, gemini-2-5-flash, gpt5-mini
2. Compare against v0.3.16 baseline
3. Document actual improvement

**Why not validated**: The loader fix (f5919ed) was completed but no eval baseline was run to verify the improvement. This should be done in a future sprint to confirm the success metric.

**Files**:
- `internal/parser/parser_error.go` (+30 LOC)
- `internal/parser/parser_decl.go` (+50 LOC)
- `internal/parser/suggestion_errors_test.go` (320 LOC)
- `internal/parser/cli_integration_test.go` (150 LOC)
- `internal/loader/loader.go` (modified Oct 22)

**Versions**:
- Phase 1 (errors): v0.3.17
- Phase 2 (loader): Commit f5919ed (Oct 22, 2025)

---

*Sprint completed: October 2025*
*Eval validation: Pending future sprint*
