# Sprint Plan: M-DX24 - Developer Experience Improvements from BigQuery Connector Feedback

## Summary

Implement six focused developer experience improvements based on real-world BigQuery connector feedback. This sprint addresses critical runtime issues (Option pattern matching, stdlib version mismatches), documentation gaps (reserved keywords, module imports), and language feature limitations (if-then-else blocks, record type inference). The goal is to reduce developer friction for real-world AILANG programs and improve error messages.

**Duration:** 10 days (2 weeks)
**Dependencies:** None
**Risk Level:** Medium (pattern matching and type inference changes require careful testing)

---

## Current Status Analysis

### Design Document Status
- ✅ M-DX24 design doc completed and approved
- ✅ 6 distinct DX issues identified from BigQuery connector
- ✅ Root cause analysis for each issue
- ✅ Implementation strategy documented

### Velocity Data
Based on recent commits:
- Recent DX improvements (M-DX23, M-BUILTIN-SAFETY): ~300-400 LOC per 3-4 days
- Documentation work: ~150-200 LOC per day
- Bug fixes with tests: ~150-200 LOC per 2-3 days
- **Estimated velocity:** 80-100 LOC/day average (accounting for investigation time)

### Estimated Capacity
- **Sprint length:** 10 working days
- **Total capacity:** ~800-1000 LOC (implementation + tests)
- **Allocation:** 400 LOC implementation, 300 LOC tests, 300 LOC documentation

---

## Proposed Milestones

### Phase 1: Documentation & Error Messages (4 days)

#### M1.1: Reserved Keywords Documentation & Error Messages
**Goal:** Document all 43 reserved keywords and improve parser error messages to help developers recognize and resolve keyword conflicts early.

**Estimated:** 200 LOC implementation + 80 LOC tests = **280 LOC**
**Duration:** 1.5 days

**Tasks:**
- **Day 1 AM:** Create `docs/reference/reserved-keywords.md` (~150 lines) listing all 43 keywords, contextual keywords, and common mistakes
- **Day 1 PM:** Implement keyword detection in `internal/errors/errors.go` (~50 lines) to improve parser error messages
- **Day 2 AM:** Add test cases to `internal/errors/errors_test.go` (~30 lines) covering keyword error detection
- **Day 2 PM:** Update README.md with reference to reserved keywords doc

**Example Files to Create:**
- `docs/reference/reserved-keywords.md` - Complete keyword reference

**Acceptance Criteria:**
- [ ] All 43 keywords documented in reference doc
- [ ] Parser error message: "Cannot use reserved keyword 'exists' as identifier"
- [ ] Keyword error test cases passing
- [ ] Documentation links properly in SUMMARY.md
- [ ] No regressions in existing error handling

**Risks:**
- Error message enhancement may affect error parsing in CI - **Mitigation:** Run full test suite before merge

**Files to Modify/Create:**
- `docs/reference/reserved-keywords.md` (new)
- `internal/errors/errors.go` (+50 lines)
- `internal/errors/errors_test.go` (+30 lines)

---

#### M1.2: Module Import Transitivity Documentation
**Goal:** Document AILANG's explicit import design (imports are not transitive) and improve error messages to guide developers.

**Estimated:** 150 LOC implementation + 40 LOC tests = **190 LOC**
**Duration:** 1 day

**Tasks:**
- **Day 3 AM:** Create/update `docs/guides/modules.md` with new section on import transitivity (~80 lines)
- **Day 3 PM:** Add examples and improve error message in module loader (~50 lines)
- **Day 4 AM:** Update teaching prompt with module import examples (~40 lines)

**Example Files to Create:**
- Enhanced `docs/guides/modules.md` with import examples

**Acceptance Criteria:**
- [ ] Import transitivity design clearly documented
- [ ] Example showing implicit vs. explicit imports
- [ ] Error message mentions missing import with suggestion
- [ ] Teaching prompt includes module pattern
- [ ] All existing module tests still pass

**Risks:**
- Documentation may not cover all edge cases - **Mitigation:** Include "See also" references to implementation

**Files to Modify/Create:**
- `docs/guides/modules.md` (modify, +80 lines)
- `internal/loader/loader.go` (modify, +20 lines for better error messages)
- `prompts/v0.7.0.md` (modify, +40 lines)

---

#### M1.3: Teaching Prompt Update for Reserved Keywords & Patterns
**Goal:** Update teaching prompt to include information about reserved keywords, module imports, and common patterns discovered from real-world usage.

**Estimated:** 120 LOC = **120 LOC**
**Duration:** 0.5 days

**Tasks:**
- Update `prompts/v0.7.0.md` with new sections covering:
  - Reserved keywords list (15 lines)
  - Module import patterns (20 lines)
  - Pattern matching examples (20 lines)
  - Common gotchas (15 lines)
  - BigQuery-style real-world example (30 lines)

**Example Files:**
- Enhanced teaching prompt with real-world patterns

**Acceptance Criteria:**
- [ ] Teaching prompt includes reserved keywords section
- [ ] Real-world examples from BigQuery connector
- [ ] Module import rules explained
- [ ] All syntax examples valid and tested

**Files to Modify:**
- `prompts/v0.7.0.md` (+100 lines)

---

### Phase 2: Runtime Fixes (6 days)

#### M2.1: Fix Option Pattern Matching ⚠️ CRITICAL
**Goal:** Debug and fix pattern matching elaboration for ADT constructors (Option/Some/None). This is the highest-priority runtime fix as it affects core language feature.

**Estimated:** 200 LOC implementation + 120 LOC tests = **320 LOC**
**Duration:** 2 days

**Tasks:**
- **Day 5 AM:** Investigate pattern elaboration in `internal/elaborate/pattern.go` and decision tree generation in `internal/dtree/dtree.go`
  - Add debug output for pattern matching failures
  - Trace through `match Some(1) { Some(x) => x, None => 0 }` elaboration
  - Identify why "no pattern matched" occurs at runtime
- **Day 5 PM:** Implement fix based on investigation findings (~100 lines)
  - Correct decision tree generation for ADT patterns
  - Ensure pattern matching covers all constructor cases
  - Add runtime error location tracking
- **Day 6 AM:** Write comprehensive test suite (~120 lines)
  - Test `match Some(x) { Some(v) => v, None => 0 }`
  - Test nested ADTs
  - Test pattern matching in different contexts (if/let/function body)
- **Day 6 PM:** Create example file and verify BigQuery connector patterns work

**Example Files to Create:**
- `examples/pattern_matching_adt.ail` - ADT pattern matching examples
- `examples/pattern_matching_option.ail` - Option/Some/None patterns (if separate)

**Acceptance Criteria:**
- [ ] `match Some(1) { Some(x) => x, None => 0 }` works and returns 1
- [ ] `match None { Some(x) => x, None => 0 }` returns 0
- [ ] Nested ADT patterns work
- [ ] Error messages include location info
- [ ] All new tests passing
- [ ] No regression in existing pattern matching tests

**Risks:**
- Pattern matching fix may introduce regressions in other areas - **Mitigation:** Comprehensive test suite with edge cases
- Root cause may be in type checking, not elaboration - **Mitigation:** Trace full pipeline if elaboration investigation inconclusive

**Files to Modify/Create:**
- `internal/elaborate/pattern.go` (modify, +50-100 lines)
- `internal/dtree/dtree.go` (modify, debug + fix)
- `internal/elaborate/pattern_test.go` (new, +120 lines)
- `examples/pattern_matching_adt.ail` (new, ~20 lines)

---

#### M2.2: Debug & Resolve Stdlib Version Mismatch ⚠️ CRITICAL
**Goal:** Investigate stdlib version mismatch warnings and determine if they indicate real problems. Either relax version checking for development or fix the root cause affecting Option patterns.

**Estimated:** 100 LOC investigation + 50 LOC fix = **150 LOC**
**Duration:** 1.5 days

**Tasks:**
- **Day 6 PM / Day 7 AM:** Investigate version check code
  - Find version checking logic in `internal/loader/loader.go` or `internal/manifest/`
  - Determine when version mismatch occurs
  - Check if mismatch affects Option pattern matching
  - Trace stdlib loading to understand version expectations
- **Day 7 PM:** Implement one of three options:
  - **Option A:** Relax dev version check (allow semver mismatch) - 20 lines
  - **Option B:** Add `--skip-version-check` flag - 30 lines
  - **Option C:** Fix root cause if investigation reveals specific issue - variable

**Example Files:**
- No example files required (infrastructure fix)

**Acceptance Criteria:**
- [ ] Version mismatch warning either resolved or clearly explained
- [ ] Investigation documented in commit message
- [ ] If flag added, documented in `ailang run --help`
- [ ] No false warnings in normal development workflow
- [ ] Test for version handling

**Risks:**
- Version checking may be safety-critical - **Mitigation:** Research why version check exists before modifying
- Version mismatch may indicate real incompatibility - **Mitigation:** Add fallback to strict checking if needed

**Files to Modify/Create:**
- `internal/loader/loader.go` (modify, +20-50 lines depending on approach)
- Tests for version checking (+20 lines)
- `cmd/ailang/main.go` (if adding flag, +10 lines)

---

#### M2.3: Enable If-Then-Else Block Expressions
**Goal:** Allow block expressions (with `let` bindings) in if-then-else branches to reduce boilerplate for multi-statement conditions.

**Estimated:** 100 LOC implementation + 80 LOC tests = **180 LOC**
**Duration:** 1.5 days

**Tasks:**
- **Day 8 AM:** Test current limitations
  - Verify `if true then { let x = 1; x + 2 } else 0` fails
  - Identify where parsing/elaboration breaks
  - Check if blocks work in other contexts
- **Day 8 PM:** Implement block support in if-then-else (~50 lines)
  - Modify `internal/parser/parser.go` if needed
  - Modify `internal/elaborate/elaborate.go` if needed
  - Ensure type checking propagates through blocks
- **Day 9 AM:** Write test suite (~80 lines)
  - Block in then branch
  - Block in else branch
  - Nested blocks
  - Block with multiple let bindings
  - Type checking with blocks

**Example Files to Create:**
- `examples/if_then_else_blocks.ail` - If-then-else with block expressions

**Acceptance Criteria:**
- [ ] `if condition then { let x = foo(); doSomething(x) } else { let y = bar(); doOther(y) }` works
- [ ] Block expressions type-check correctly
- [ ] Nested blocks work
- [ ] All new tests passing
- [ ] No regressions in existing if-then-else tests

**Risks:**
- Block support may already be partially implemented - **Mitigation:** Thorough testing before implementation
- Parser changes may affect other constructs - **Mitigation:** Comprehensive test coverage

**Files to Modify/Create:**
- `internal/parser/parser.go` (modify if needed, ~30-50 lines)
- `internal/elaborate/elaborate.go` (modify if needed, ~20-30 lines)
- `internal/elaborate/elaborate_test.go` or new test file (+80 lines)
- `examples/if_then_else_blocks.ail` (new, ~20 lines)

---

#### M2.4: Improve Record Type Inference in ADT Constructor Contexts
**Goal:** Allow record literals in ADT constructor arguments to benefit from type information (e.g., `Ok({ field: value })` without intermediate binding).

**Estimated:** 120 LOC implementation + 80 LOC tests = **200 LOC**
**Duration:** 1.5 days

**Tasks:**
- **Day 9 PM / Day 10 AM:** Investigate type inference issue
  - Test current limitation: `let _: Result[Rec, str] = Ok({ field: value })`
  - Check type unification in `internal/types/unify.go`
  - Determine if inference can propagate into record literals
- **Day 10 AM:** Implement type propagation (~60 lines)
  - Modify record literal type inference
  - Ensure bidirectional type checking helps
  - Add error message suggesting intermediate binding as workaround
- **Day 10 PM:** Write test suite (~80 lines)
  - Direct record in Ok/Err
  - Nested records
  - Partial type inference
  - Error cases with helpful messages

**Example Files to Create:**
- `examples/record_type_inference.ail` - Record construction in ADTs

**Acceptance Criteria:**
- [ ] `let _: Result[{a: int}, str] = Ok({ a: 1 })` works without intermediate binding
- [ ] Type inference error messages suggest intermediate binding workaround
- [ ] Nested records work
- [ ] All new tests passing
- [ ] No regressions in record or ADT tests

**Risks:**
- Type inference changes may affect other constructs - **Mitigation:** Test unification thoroughly
- Bidirectional type checking may be complex - **Mitigation:** Start with simpler cases if needed

**Files to Modify/Create:**
- `internal/types/unify.go` (modify, +60 lines)
- `internal/elaborate/elaborate.go` (modify if needed, +20 lines)
- Type inference tests (+80 lines)
- `examples/record_type_inference.ail` (new, ~20 lines)

---

## Success Metrics

### Code Quality
- [ ] All new code follows AILANG conventions
- [ ] Test coverage ≥ 85% for new code
- [ ] All tests passing: `make test`
- [ ] All linting passing: `make lint`
- [ ] No regressions in existing functionality

### Documentation
- [ ] `docs/reference/reserved-keywords.md` created and linked
- [ ] `docs/guides/modules.md` updated with import transitivity section
- [ ] Teaching prompt updated with v0.7.0 version
- [ ] All example files included and verified working
- [ ] README updated with feature status if applicable

### Example Files (CRITICAL - Every new feature requires examples!)
- [ ] `examples/pattern_matching_adt.ail` - Works without errors
- [ ] `examples/if_then_else_blocks.ail` - Works without errors
- [ ] `examples/record_type_inference.ail` - Works without errors
- [ ] BigQuery connector example patterns all work (if provided)

### Real-World Validation
- [ ] Option pattern matching works in production context
- [ ] No stdlib version mismatch warnings in normal workflow
- [ ] Error messages helpful for debugging (reduced iteration time)
- [ ] Module import errors clearly guide to solutions

### Metrics Summary
- **Total Estimated LOC:**
  - Implementation: ~420 LOC
  - Tests: ~370 LOC
  - Documentation: ~310 LOC
  - **Total: ~1,100 LOC**
- **Estimated days:** 10 working days
- **Estimated LOC/day:** 110 (includes investigation, testing, documentation)

---

## Detailed Task Breakdown by Day

### Day 1: Reserved Keywords Documentation & Parser Error Messages
- **AM (4h):** Create `docs/reference/reserved-keywords.md` (~150 lines)
  - List all 43 keywords by category
  - Identify 5-7 contextual keywords
  - Add common mistakes section
  - Cross-reference with implementation
- **PM (4h):** Implement keyword detection in parser errors
  - Modify `internal/errors/errors.go` (~50 lines)
  - Add helper to detect reserved keywords
  - Update error message template

### Day 2: Keyword Error Testing & Module Import Docs
- **AM (4h):** Add tests for keyword error detection
  - Create test cases for each contextual keyword
  - Verify error message quality
  - Update `internal/errors/errors_test.go`
- **PM (4h):** Start module import documentation
  - Create/update `docs/guides/modules.md`
  - Add "Import Transitivity" section
  - Include examples of correct vs. incorrect patterns

### Day 3: Module Import Documentation & Teaching Prompt
- **AM (4h):** Complete module import documentation
  - Add error message reference
  - Document workarounds if any
  - Add to SUMMARY.md navigation
- **PM (4h):** Update teaching prompt
  - Add reserved keywords section
  - Add module import patterns
  - Include real-world examples

### Day 4: Improve Module Error Messages
- **AM (4h):** Improve error messages in module loader
  - Enhance "module not imported" error
  - Add suggestion for explicit import
  - Test error messages
- **PM (4h):** Investigation prep for pattern matching
  - Review existing pattern matching tests
  - Study `internal/elaborate/pattern.go`
  - Prepare test case for Option patterns

### Day 5: Pattern Matching Investigation & Fix
- **AM (4h):** Deep investigation of pattern matching issue
  - Trace `match Some(1) { Some(x) => x, None => 0 }` through elaboration
  - Check decision tree generation in `internal/dtree/dtree.go`
  - Add debug output to identify failure point
  - Run existing pattern matching tests
- **PM (4h):** Implement fix
  - Correct elaboration or decision tree logic (~100 lines)
  - Ensure all ADT patterns handled correctly
  - Update error messages with location info

### Day 6: Pattern Matching Tests & Version Mismatch Investigation
- **AM (4h):** Write comprehensive pattern matching test suite
  - Basic ADT patterns
  - Nested patterns
  - Multiple match arms
  - Different contexts (if/let/function)
  - Edge cases
- **PM (4h):** Start version mismatch investigation
  - Find version checking code
  - Understand version expectations
  - Test with different stdlib versions

### Day 7: Version Mismatch Fix & If-Then-Else Blocks Start
- **AM (4h):** Complete version mismatch fix
  - Implement chosen approach (relax check, add flag, or fix root cause)
  - Add test for version handling
  - Document decision in commit
- **PM (4h):** Start if-then-else blocks investigation
  - Test current limitations thoroughly
  - Identify exact point of failure
  - Plan implementation approach

### Day 8: If-Then-Else Blocks Implementation
- **AM (4h):** Implement block support in if-then-else
  - Modify parser if needed (~30-50 lines)
  - Modify elaborator if needed (~20-30 lines)
  - Ensure type checking works
- **PM (4h):** Start if-then-else test suite
  - Basic block in then branch
  - Block in else branch
  - Nested blocks
  - Type checking tests

### Day 9: Complete If-Then-Else Tests & Start Record Inference
- **AM (4h):** Complete if-then-else test suite
  - Finish all test cases
  - Verify no regressions
  - Create example file
- **PM (4h):** Investigate record type inference issue
  - Test current limitations
  - Study type unification logic
  - Plan inference improvement

### Day 10: Record Type Inference Fix & Final Testing
- **AM (4h):** Implement record type inference improvement
  - Modify type unification (~60 lines)
  - Add bidirectional checking if needed
  - Create example file
- **PM (4h):** Final testing and verification
  - Run full test suite
  - Verify all examples work
  - Check for regressions
  - Documentation review

---

## Dependencies

**None** - This sprint is self-contained and doesn't block other work.

**Prerequisites Assumed:**
- AILANG v0.7.0 development environment set up
- Go 1.21+ installed
- Familiar with pattern matching implementation details (for M2.1)

---

## Open Questions

1. **Pattern Matching Root Cause:** Is the issue in elaboration, decision tree generation, or runtime evaluation? Investigation Day 5 will clarify.

2. **Version Mismatch Action:** Should we relax the check, add a flag, or investigate if it affects Option patterns? Will be determined during investigation.

3. **Record Inference Scope:** How far should we push type inference? Should it work in all contexts or just ADT constructor arguments?

4. **Priority Adjustment:** If pattern matching fix takes longer than estimated, should we defer record inference to next sprint?

---

## Risk Assessment & Mitigation

### High-Risk Areas

**1. Pattern Matching Fix (M2.1)**
- **Risk:** Elaboration changes might break other pattern matching scenarios
- **Mitigation:**
  - Comprehensive test suite covering nested ADTs, different contexts, edge cases
  - Run existing test suite after each change
  - Use git bisect if regressions appear

**2. Type Inference Changes (M2.4)**
- **Risk:** Unification improvements might affect other type checking
- **Mitigation:**
  - Test type inference thoroughly before and after
  - Include tests for edge cases (partial inference, error cases)
  - Review changes carefully in PR

### Medium-Risk Areas

**3. If-Then-Else Blocks (M2.3)**
- **Risk:** Parser changes might affect other constructs
- **Mitigation:** Comprehensive test coverage, careful parser review

**4. Version Mismatch (M2.2)**
- **Risk:** Relaxing version check might hide real incompatibilities
- **Mitigation:** Investigate why check exists before modifying, add fallback if needed

### Low-Risk Areas

**5. Documentation Changes (M1.x)**
- **Risk:** Documentation inaccuracy
- **Mitigation:** Review examples thoroughly, test all code snippets

---

## Testing Strategy

### Test Coverage Requirements
- **Phase 1 (Documentation):** Linting, example validation (5 hours)
- **Phase 2.1 (Pattern Matching):** ~30 test cases, integration testing (6 hours)
- **Phase 2.2 (Version Mismatch):** Version checking tests (2 hours)
- **Phase 2.3 (If-Then-Else):** ~20 test cases (4 hours)
- **Phase 2.4 (Record Inference):** ~15 test cases (3 hours)

### Test Execution
```bash
# Unit tests for each component
make test                        # All tests
make test-coverage-badge        # Coverage check

# Specific test files
go test ./internal/elaborate/...
go test ./internal/types/...
go test ./internal/errors/...

# Example file validation
make verify-examples            # Ensure all examples run

# Full integration
make lint
make fmt-check
make ci                         # Full CI pipeline
```

---

## Known Limitations & Deferred Work

### Out of Scope for v0.7.0

The following related issues are important but deferred:
- **Nullary Function Calls (M-DX10):** Cannot call zero-arg functions from AILANG - requires syntax changes
- **Performance Optimization:** No optimization work in this DX pass
- **Syntax Additions:** Minimal changes per design principle Axiom A8
- **LSP/IDE Integration:** Deferred to v0.8.0+

---

## Success Definition

### Must Have (Blockers)
- ✅ Option pattern matching works at runtime
- ✅ No false stdlib version warnings
- ✅ Reserved keywords documented and error messages improved
- ✅ All tests passing

### Should Have (High Value)
- ✅ Module import rules clearly documented
- ✅ If-then-else blocks working
- ✅ Record type inference improved
- ✅ Teaching prompt updated

### Nice to Have (Polish)
- ✅ Enhanced error messages with examples
- ✅ Additional example files
- ✅ Performance improvements (if found during investigation)

---

## Coordinator Integration

This sprint is approved and ready for implementation. When ready to start execution:

```bash
ailang messages send sprint-executor '{
  "type": "plan_ready",
  "correlation_id": "sprint_M-DX24_20260127",
  "sprint_id": "M-DX24",
  "plan_path": "design_docs/planned/v0_7_0/m-dx24-sprint-plan.md",
  "progress_path": ".ailang/state/sprints/sprint_M-DX24.json",
  "estimated_duration": "10 days",
  "total_loc_estimate": 1100,
  "risk_level": "medium"
}'
```

---

## References

- **Design Document:** `design_docs/planned/v0_7_0/m-dx24-developer-dx-improvements.md`
- **CLAUDE.md:** Development instructions and patterns
- **Teaching Prompt:** `prompts/v0.7.0.md`
- **Pattern Matching Implementation:** `internal/elaborate/pattern.go`, `internal/dtree/dtree.go`
- **Type System:** `internal/types/unify.go`, `internal/elaborate/elaborate.go`

---

## Notes

- **Assumption:** The BigQuery connector example files will be used to validate fixes
- **Key Learning:** Real-world feedback is invaluable - these 6 issues came from actual developer experience
- **Success Factor:** Pattern matching fix (M2.1) is critical path; if it requires more time, consider deferring M2.4
- **Documentation First:** Documentation changes (Phase 1) should complete first to unblock communication
