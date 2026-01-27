# M-DX24 Sprint Plan: Developer Experience Improvements

**Status:** Ready for Execution
**Target:** v0.7.0
**Duration:** 10 days (2 weeks with buffer)
**Total LOC Estimate:** 670 LOC (implementation + tests + docs)
**Risk Level:** Medium (pattern matching fix is complex)
**Created:** 2026-01-27

---

## Executive Summary

Sprint plan for 6 DX improvements addressing real-world BigQuery connector feedback. Focus on:
1. Documentation gaps (reserved keywords, module imports)
2. Critical runtime bugs (Option pattern matching, stdlib version mismatch)
3. Language feature gaps (if-then-else blocks, record type inference)

**Key Success Metric:** BigQuery demo code works without workarounds.

---

## Current Status Analysis

### Completed ✅
- Design analysis of 6 issues from production code
- Root cause analysis per issue
- Impact assessment
- Technical approach documented

### In Progress ⏳
- None (ready for sprint execution)

### Blocked ❌
- Cannot start until design-doc-creator approval

### Metrics
- **Recent velocity:** ~1,200 LOC per 2-week period (based on recent work)
- **Estimated this sprint:** 670 LOC (conservative estimate)
- **Expected duration:** 10 calendar days (fits within 2-week sprint window)
- **Test coverage target:** 85%+
- **No breaking changes:** All modifications backward compatible

---

## Proposed Milestones

### MILESTONE 1: Documentation & Error Messages (3 days, ~250 LOC)

**Goal:** Fix discoverability issues and improve error messages

#### Task 1.1: Create Reserved Keywords Reference
- **File:** `docs/reference/reserved-keywords.md` (NEW, ~150 LOC)
- **What to create:**
  - Complete list of 43 reserved keywords by category
  - Contextual keywords explanation (test, tests, properties)
  - Common mistakes and workarounds
  - Link from main docs navigation
- **Acceptance Criteria:**
  - [ ] All 43 keywords documented with explanation
  - [ ] Clear examples for each contextual keyword
  - [ ] File appears in docs/SUMMARY.md navigation
  - [ ] Docusaurus builds without errors
- **Estimated LOC:** 150
- **Dependencies:** None
- **Risk:** Low - pure documentation

#### Task 1.2: Improve Parser Error Messages for Keywords
- **File:** `internal/errors/errors.go` (MODIFY, +50 LOC)
- **What to implement:**
  - Detect when parse error involves reserved keyword
  - Generate helpful error: "Cannot use reserved keyword 'X' as identifier"
  - Add suggestion: "Try using alternative name like 'X_var'"
- **Test file:** `internal/errors/errors_test.go` (ADD, +40 LOC)
- **Example test case:**
  ```ailang
  -- Should produce: "Cannot use reserved keyword 'exists' as identifier"
  let exists = true;  -- Parser error with suggestion
  ```
- **Acceptance Criteria:**
  - [ ] Error message clearly states keyword is reserved
  - [ ] Suggestion provided for alternative naming
  - [ ] Test case verifies error message for each contextual keyword
  - [ ] No regression in other error messages
- **Estimated LOC:** 90 (code + tests)
- **Dependencies:** None
- **Risk:** Low - localized change

#### Task 1.3: Document Module Import Rules
- **File:** `docs/guides/modules.md` (MODIFY, +80 LOC)
- **What to add:**
  - New section: "Import Transitivity"
  - Explanation: Why imports are NOT transitive
  - Error message reference with solutions
  - Before/after example showing explicit import requirement
  - Link to reserved keywords reference
- **Acceptance Criteria:**
  - [ ] Import transitivity rule clearly explained
  - [ ] Example showing correct usage
  - [ ] Error message and solution documented
  - [ ] Docusaurus builds correctly
- **Estimated LOC:** 80
- **Dependencies:** Task 1.1 (reserved keywords doc)
- **Risk:** Low - documentation only

#### Task 1.4: Update Teaching Prompt
- **File:** `prompts/v0.7.0.md` (NEW, ~100 LOC)
- **What to create:**
  - New version of teaching prompt for v0.7.0
  - Section on reserved keywords with examples
  - Module import examples and rules
  - Pattern matching with ADTs (after fix)
  - If-then-else with blocks
- **Acceptance Criteria:**
  - [ ] All reserved keywords mentioned
  - [ ] Module import examples clear
  - [ ] Pattern matching examples work
  - [ ] Prompt validated with multi-model testing
- **Estimated LOC:** 100
- **Dependencies:** Tasks 1.1, 1.3, 2.1 (pattern matching fix)
- **Risk:** Medium - must ensure examples are accurate

**Milestone 1 Total:** ~450 LOC, 3 days
**Completion Criteria:** All docs updated, error messages improved, teaching prompt v0.7.0 ready

---

### MILESTONE 2: Fix Option Pattern Matching (3 days, ~200 LOC)

**Goal:** Fix critical runtime bug where pattern matching fails for Option type

#### Task 2.1: Investigate Pattern Matching Elaboration
- **Files to examine:**
  - `internal/elaborate/pattern.go` - Pattern elaboration
  - `internal/dtree/dtree.go` - Decision tree generation
  - `internal/eval/eval.go` - Match expression evaluation
- **Investigation steps:**
  - [ ] Write minimal test case: `match Some(1) { Some(x) => x, None => 0 }`
  - [ ] Trace elaboration output (verbose mode)
  - [ ] Check Core AST generated for pattern
  - [ ] Verify decision tree covers all cases
  - [ ] Identify where "no pattern matched" error comes from
- **Expected output:** Root cause identification and fix strategy
- **Duration:** 1 day
- **Acceptance Criteria:**
  - [ ] Root cause identified
  - [ ] Decision tree verified correct
  - [ ] Test case reproduces issue
  - [ ] Fix strategy documented

#### Task 2.2: Fix Pattern Matching Code
- **Files to modify:**
  - `internal/elaborate/pattern.go` (~100 LOC added/modified)
  - `internal/dtree/dtree.go` (debug + fix)
- **Fix strategy:** Based on investigation results
- **Likely fixes:**
  - Correct ADT constructor matching logic
  - Ensure pattern variables are bound correctly
  - Verify fallthrough behavior
- **Example file:** `examples/pattern_matching_adt.ail` (NEW, ~25 LOC)
  ```ailang
  import std/option (Some, None, isSome, getOrElse)

  -- This should work after fix:
  func testOption() -> string {
    let opt = Some("home");
    match opt {
      Some(h) => h,
      None => "/tmp"
    }
  }

  -- Also test with recursion:
  func sumOption(lst) -> int {
    match lst {
      [] => 0,
      (Some(x)) : rest => x + sumOption(rest),
      None : rest => sumOption(rest)
    }
  }
  ```
- **Tests:** `internal/elaborate/pattern_test.go` (NEW, +60 LOC)
  - Test simple ADT patterns (Some, None)
  - Test nested patterns
  - Test with recursion
  - Test Option type specifically
- **Duration:** 2 days
- **Acceptance Criteria:**
  - [ ] Test case `match Some(1) { Some(x) => x, None => 0 }` returns `1`
  - [ ] Pattern matching works with Option type
  - [ ] No regression in existing pattern matching tests
  - [ ] Example file `pattern_matching_adt.ail` works
- **Risk:** High - complex elaboration logic, potential for regressions

**Milestone 2 Total:** ~200 LOC, 3 days
**Completion Criteria:** Option pattern matching works, test cases pass, example file runs

---

### MILESTONE 3: Stdlib Version Mismatch Resolution (2 days, ~80 LOC)

**Goal:** Resolve or clarify stdlib version mismatch warnings

#### Task 3.1: Investigate Version Check Logic
- **Files to examine:**
  - `internal/loader/loader.go` - Module loading
  - `internal/manifest/` - Manifest checking
  - Search for "version mismatch" string
- **Investigation steps:**
  - [ ] Find version check code
  - [ ] Determine when version mismatch occurs
  - [ ] Check if it affects pattern matching or other runtime
  - [ ] Review version handling in stdlib
- **Expected output:** Root cause and resolution strategy
- **Duration:** 1 day
- **Acceptance Criteria:**
  - [ ] Version check code located and understood
  - [ ] Impact on pattern matching determined
  - [ ] Resolution strategy defined

#### Task 3.2: Implement Version Fix
- **Decision point:** Choose one of three options:
  1. **Relax version check** - Allow semver mismatches during development
  2. **Add flag** - Add `--skip-version-check` for dev/testing
  3. **Fix root cause** - Update version handling to be more flexible
- **Files to modify:** Likely `internal/loader/loader.go` (~50 LOC)
- **Test case:** Verify Option pattern matching works with version check
- **Duration:** 1 day
- **Acceptance Criteria:**
  - [ ] Version mismatch warning eliminated or clarified
  - [ ] Option pattern matching not affected by version check
  - [ ] Development experience improved
  - [ ] No regression in module loading

**Milestone 3 Total:** ~80 LOC, 2 days
**Completion Criteria:** Version mismatch resolved, pattern matching unaffected

---

### MILESTONE 4: Enable If-Then-Else Block Expressions (2 days, ~80 LOC)

**Goal:** Allow multi-statement blocks inside if-then-else branches

#### Task 4.1: Verify Block Expression Support
- **Test if basic blocks work:**
  - `{ let x = 1; x + 2 }` as standalone expression
  - `if true then { 1 + 2 } else { 3 + 4 }` in if-then-else
  - `if true then { let x = 1; x + 2 } else 0` with let binding
- **Files to check:**
  - `internal/parser/parser.go` - If-then-else parsing
  - `internal/elaborate/elaborate.go` - If elaboration
- **Expected output:** Identified gaps or parsing issues
- **Duration:** 0.5 day

#### Task 4.2: Implement Block Support
- **If gaps found, fix in:**
  - `internal/parser/parser.go` (~30 LOC)
  - `internal/elaborate/elaborate.go` (~30 LOC)
- **Example file:** `examples/if_then_else_blocks.ail` (NEW, ~25 LOC)
  ```ailang
  func testIfBlocks() -> int {
    let x = 5;
    let result = if x > 0 then {
      let doubled = x * 2;
      let squared = doubled * doubled;
      squared
    } else {
      let negative = 0 - x;
      negative * negative
    };
    result
  }
  ```
- **Tests:** `internal/parser/parser_test.go` (ADD, +40 LOC)
  - Test block expressions in if-then
  - Test block expressions in else
  - Test nested blocks
  - Test with let bindings inside
- **Duration:** 1.5 days
- **Acceptance Criteria:**
  - [ ] `if condition then { let x = ...; expr } else expr` parses and type-checks
  - [ ] Example file runs and produces correct result
  - [ ] Test cases cover nested blocks and let bindings
  - [ ] No regression in existing if-then-else parsing

**Milestone 4 Total:** ~80 LOC, 2 days
**Completion Criteria:** If-then-else blocks work, example file runs

---

### MILESTONE 5: Improve Record Type Inference (2 days, ~60 LOC)

**Goal:** Enable direct record construction in ADT constructor contexts

#### Task 5.1: Investigate Record Inference Issue
- **Test case:**
  ```ailang
  type ADCCredentials = { clientId: string, projectId: string }
  type Result[A, E] = Ok(A) | Err(E)

  -- This should work but currently fails:
  let result: Result[ADCCredentials, string] =
    Ok({ clientId: "...", projectId: "..." });
  ```
- **Files to examine:**
  - `internal/types/unify.go` - Type unification
  - `internal/elaborate/elaborate.go` - Record elaboration
  - `internal/types/` - Type inference
- **Investigation steps:**
  - [ ] Trace type inference for record literal
  - [ ] Check if type expectation propagates from Ok() parameter
  - [ ] Identify where unification fails
- **Expected output:** Root cause and fix strategy
- **Duration:** 1 day
- **Acceptance Criteria:**
  - [ ] Root cause identified
  - [ ] Fix approach documented
  - [ ] Test case reproduces issue

#### Task 5.2: Improve Type Inference Code
- **Files to modify:**
  - `internal/types/unify.go` (~40 LOC)
  - `internal/elaborate/elaborate.go` (~20 LOC)
- **Changes:**
  - Propagate type expectations to record literals
  - Improve error messages when inference fails
  - Add suggestion for intermediate binding workaround
- **Test case:** `examples/record_in_result.ail` (NEW, ~20 LOC)
- **Tests:** `internal/types/unify_test.go` (ADD, +30 LOC)
- **Duration:** 1 day
- **Acceptance Criteria:**
  - [ ] Record construction in ADT contexts works
  - [ ] Type inference error message helpful
  - [ ] Workaround documentation in error message
  - [ ] Example file runs correctly

**Milestone 5 Total:** ~60 LOC, 2 days
**Completion Criteria:** Record type inference improved, example file runs

---

## Summary of Changes

### New Files to Create (11 files, ~435 LOC)

**Documentation:**
1. `docs/reference/reserved-keywords.md` (~150 LOC)
2. `prompts/v0.7.0.md` (~100 LOC)

**Example Files:**
3. `examples/pattern_matching_adt.ail` (~25 LOC)
4. `examples/if_then_else_blocks.ail` (~25 LOC)
5. `examples/record_in_result.ail` (~20 LOC)

**Test Files:**
6. `internal/errors/errors_test.go` (NEW, +40 LOC)
7. `internal/elaborate/pattern_test.go` (~60 LOC)
8. `internal/parser/parser_test.go` (ADD, +40 LOC)
9. `internal/types/unify_test.go` (ADD, +30 LOC)

### Files to Modify (8 files, ~235 LOC added)

**Documentation Updates:**
1. `docs/guides/modules.md` (+80 LOC)
2. `docs/SUMMARY.md` (add references to new docs)

**Implementation:**
3. `internal/errors/errors.go` (+50 LOC)
4. `internal/elaborate/pattern.go` (+100 LOC)
5. `internal/parser/parser.go` (+30 LOC)
6. `internal/elaborate/elaborate.go` (+50 LOC)
7. `internal/types/unify.go` (+40 LOC)
8. `internal/loader/loader.go` (~50 LOC, version check)

---

## Day-by-Day Breakdown

### Day 1-2: Documentation & Keywords (Milestones 1.1-1.2)
- Create `docs/reference/reserved-keywords.md`
- Improve parser error messages
- Add test cases for keyword errors
- **Target:** 140 LOC, docs visible

### Day 3: Module Docs & Teaching Prompt (Milestones 1.3-1.4)
- Document module import rules
- Create teaching prompt v0.7.0
- Add examples
- **Target:** 180 LOC, docs complete

### Day 4-5: Pattern Matching Investigation & Fix (Milestone 2.1-2.2)
- Investigate pattern matching elaboration
- Fix Option pattern matching bug
- Write comprehensive tests
- Create example file
- **Target:** 200 LOC, pattern matching works

### Day 6-7: Version Mismatch & If-Then-Else (Milestones 3 & 4)
- Resolve version mismatch warning
- Enable if-then-else blocks
- Create example and tests
- **Target:** 160 LOC, features work

### Day 8-9: Record Type Inference (Milestone 5)
- Investigate record inference issue
- Improve type inference
- Create example file
- Add tests
- **Target:** 60 LOC, records work

### Day 10: Integration Testing & Polish
- Run full test suite
- Verify all examples work
- Update CHANGELOG
- Final validation
- **Target:** All tests pass, 0 regressions

---

## Success Metrics

### Primary Goals
- [ ] Reserved keywords documented and discoverable
- [ ] Parser error messages mention reserved keywords
- [ ] Option pattern matching works correctly
- [ ] Stdlib version mismatch resolved
- [ ] If-then-else with blocks works
- [ ] Record construction in ADT contexts works
- [ ] BigQuery demo runs without workarounds

### Code Quality
- [ ] Test coverage ≥ 85% for new code
- [ ] No regressions in existing tests
- [ ] All example files verified working
- [ ] CHANGELOG updated with changes
- [ ] Documentation links verified

### Performance
- [ ] No performance regression
- [ ] Build time unchanged
- [ ] No memory regression

---

## Risk Assessment

### High Risk
**Pattern Matching Fix** - Complex elaboration logic
- **Mitigation:** Comprehensive test suite, careful code review
- **Contingency:** If fix proves difficult, document workaround clearly

### Medium Risk
**Record Type Inference** - Type system changes
- **Mitigation:** Thorough testing, edge cases covered
- **Contingency:** If too complex, improve error messages instead

**Version Mismatch** - May affect multiple systems
- **Mitigation:** Investigate carefully before committing to fix
- **Contingency:** If complex, add flag and document

### Low Risk
**Documentation & Error Messages** - Pure text/error handling changes
**If-Then-Else Blocks** - Parser change, well-tested

---

## Dependencies & Prerequisites

### External Dependencies
- None

### Internal Dependencies
- Existing AILANG compiler infrastructure
- Existing test framework
- Teaching prompt versioning system

### Blockers
- None - ready to start immediately

---

## Acceptance Criteria

### Documentation
- [ ] Reserved keywords reference complete
- [ ] Module import rules documented
- [ ] All links working in Docusaurus
- [ ] Teaching prompt ready for v0.7.0

### Code Changes
- [ ] Parser error message test cases pass
- [ ] Pattern matching test cases pass
- [ ] If-then-else blocks test cases pass
- [ ] Record inference test cases pass
- [ ] No regressions in existing tests
- [ ] Test coverage ≥ 85%

### Example Files
- [ ] `pattern_matching_adt.ail` runs correctly
- [ ] `if_then_else_blocks.ail` runs correctly
- [ ] `record_in_result.ail` runs correctly
- [ ] All examples documented

### Real-World Validation
- [ ] BigQuery demo code (from design doc) runs without workarounds
- [ ] All 6 issues addressed
- [ ] Developer experience improved

---

## CHANGELOG Entry Template

```markdown
## [v0.7.0] - 2026-02-XX

### Fixed
- **Pattern Matching with Option Type** - Fixed runtime failure where `match Some(x) { Some(h) => h, None => ... }` threw "no pattern matched"
- **Stdlib Version Mismatch** - Resolved development version warnings
- **Type Inference in ADT Contexts** - Record literals now work directly in ADT constructors

### Added
- **Reserved Keywords Documentation** - Complete reference at `docs/reference/reserved-keywords.md`
- **If-Then-Else Blocks** - Multi-statement blocks now supported in if-then-else branches
- **Module Import Rules** - Documented explicit import requirement in `docs/guides/modules.md`
- **Teaching Prompt v0.7.0** - Updated with module, pattern matching, and block examples
- **Pattern Matching Tests** - Comprehensive test suite for ADT patterns
- **Error Messages** - Parser now suggests alternatives for reserved keywords

### Changed
- Parser error messages improved for reserved keywords
- Teaching prompt updated with v0.7.0 examples

### Files Modified
- `internal/elaborate/pattern.go` - Pattern matching fix (~100 LOC)
- `internal/errors/errors.go` - Better error messages (+50 LOC)
- `internal/parser/parser.go` - Block support (+30 LOC)
- `internal/types/unify.go` - Type inference improvements (+40 LOC)
- `docs/guides/modules.md` - Import rules documentation (+80 LOC)
- Multiple test files with new cases

### Test Coverage
- Pattern matching: 8 new test cases
- Error messages: 5 test cases per contextual keyword
- If-then-else: 6 new test cases
- Type inference: 4 new test cases
- Overall test coverage: 85%+

### Impact
- Developers can now use pattern matching with Option types without workarounds
- Reserved keywords are discoverable
- Module imports are well-documented
- Code is more readable with if-then-else blocks
- Better error messages guide developers to solutions
```

---

## Related Design Documents

- [M-DX24: Developer Experience Improvements](./m-dx24-developer-dx-improvements.md) - Original design doc
- [M-DX1: Developer Experience](../implemented/v0_3_10/m-dx1-developer-experience.md) - Previous DX work
- [Teaching Prompt System](../implemented/v0_5_0/prompt-system.md) - Documentation of prompt versioning

---

## Next Steps

1. **Review this sprint plan** - Adjust milestones, timelines, or scope as needed
2. **Approve and commit** - Once agreed, commit plan to git
3. **Hand off to sprint-executor** - Pass plan + JSON progress file to implementation agent
4. **Monitor progress** - Daily check-ins on milestone completion
5. **Test integration** - Run full suite before each major milestone
6. **Validate with BigQuery demo** - Ensure real-world use case works

---

## Appendix: BigQuery Demo Validation

After sprint completion, validate that the BigQuery demo code works without workarounds:

```ailang
-- This should all work without modifications:

module gcp_demo

-- 1. Pattern matching with Option (was broken, now fixed)
func getEnvOrDefault(key) -> string {
  match getEnv(key) {
    Some(h) => h,
    None => "/tmp"
  }
}

-- 2. If-then-else with blocks (now supported)
func buildConfig() -> config ! {IO} {
  if hasEnv("CREDENTIALS_PATH") then {
    let path = getEnv("CREDENTIALS_PATH");
    loadCredsFromFile(path)
  } else {
    getDefaultCredentials()
  }
}

-- 3. Record construction in Result (type inference improved)
func authenticate() -> Result[ADCCredentials, string] {
  Ok({
    clientId: "gcp-client",
    projectId: "my-project"
  })
}

-- All 6 DX issues resolved in this one file!
```

This represents the success criteria for the sprint.
