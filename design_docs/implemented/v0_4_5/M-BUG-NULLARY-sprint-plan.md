# Sprint Plan: M-BUG-NULLARY - Nullary Constructor Pattern Matching Fix

## Summary
Fix critical pattern matching bug where nullary-only ADTs (simple enums) always match the first pattern, restoring type safety guarantees and enabling production use of enum types. This will bring eval benchmark success from 96.1% → 100%.

**Duration:** 1-2 days (3-5 hours total)
**Dependencies:** None (standalone bug fix)
**Risk Level:** Low (isolated fix, well-understood bug, comprehensive tests)
**Target:** v0.4.5

## Current Status Analysis

### What's Broken
- ❌ **Nullary-only ADTs fail**: `type E = A | B | C` - all match first pattern
- ❌ **Eval benchmark**: 96.1% success (73/76 total, 3 failures from this bug)
- ❌ **Production blocker**: Simple enums unusable without ugly workarounds

### What Works (Important Context)
- ✅ **List patterns**: `x :: rest`, `1 :: 2 :: 3 :: []` (confirmed v0.4.4)
- ✅ **Mixed ADTs**: `type Option = Some(int) | None` - both variants match correctly
- ✅ **Non-nullary only**: ADTs where all constructors take arguments
- ✅ **Core pattern matching**: System works, this is an edge case bug

**Key Insight:** The bug is in a code path that **only triggers for nullary-only ADTs**, not systemic to pattern matching.

### Velocity (Recent 14 Days)
- Pattern sugar support: ~639 LOC (49 impl + 313 parser tests + 157 integration + 120 example)
- StdlibResolver: ~750 LOC (290 impl + 60 platform + 400 tests)
- Prompt loader: ~630 LOC (110 impl + 180 CLI + 340 tests)
- **Total**: ~2019 LOC across 3 features (~144 LOC/day average)

**This Sprint:** ~80-100 LOC (well within capacity)

### Design Doc Status
- ✅ **Design doc complete**: `design_docs/planned/v0_4_5/nullary-constructor-pattern-matching-bug.md`
- ✅ **Root cause hypotheses**: 2 detailed hypotheses with evidence
- ✅ **Investigation plan**: 4-phase approach to locate bug
- ✅ **Fix locations**: 3 likely code paths identified
- ✅ **Test strategy**: Unit + integration + regression + benchmark
- ✅ **Success metrics**: Clear, measurable criteria

## Proposed Implementation Plan

### Day 1: Investigation & Fix (2-3 hours)

#### Session 1: Investigation (1-2 hours)

**Goal:** Locate exact bug location among 3 candidates: decision tree, elaboration, or runtime evaluation.

**Phase 1: Verify Core AST** (~30 min)
```bash
# Create minimal test case
cat > /tmp/test_nullary.ail <<EOF
type E = A | B | C
func f(e: E) -> string { match e { A => "a", B => "b", C => "c" } }
EOF

# Check elaboration
ailang debug ast /tmp/test_nullary.ail --show-types
```

**Expected:** Each variant pattern has distinct Tag (0, 1, 2)
**Deliverable:** Confirm tags are correct OR bug found in elaboration

**Phase 2: Check Decision Tree** (~30 min)
```bash
# Search nullary handling
grep -rn "Arity.*0\|nullary" internal/dtree/
grep -rn "TagTest\|compilePattern" internal/dtree/

# Read decision tree code
cat internal/dtree/decision_tree.go
```

**Look for:** Code that handles `arity == 0` without emitting TagTest
**Deliverable:** Confirm/deny decision tree bug

**Phase 3: Check Evaluator** (~30 min)
```bash
# Find pattern matching code
grep -rn "evalMatch" internal/eval/
cat internal/eval/eval_typed.go | grep -A 50 "evalMatch"
```

**Look for:** Tag comparison skipped when `arity == 0`
**Deliverable:** Confirm/deny runtime evaluation bug

**Phase 4: Debug Logging** (~30 min if needed)
```go
// Instrument suspected code path
if variant.Arity == 0 {
    fmt.Fprintf(os.Stderr, "DEBUG: Nullary %s tag=%d\n", variant.Name, variant.Tag)
}
```

**Run test:** `ailang run /tmp/test_nullary.ail`
**Deliverable:** Trace showing tag values for A, B, C

**Acceptance Criteria:**
- [ ] Bug location identified (decision tree, elaboration, or evaluator)
- [ ] Root cause understood (tag comparison missing/skipped)
- [ ] Why mixed ADTs work but nullary-only fail (documented)

**Estimated Files to Read:**
- `internal/dtree/decision_tree.go` (~200 LOC)
- `internal/elaborate/patterns.go` (~150 LOC)
- `internal/eval/eval_typed.go` (grep for `evalMatch`, ~100 LOC)

---

#### Session 2: Implement Fix (1-2 hours)

**Goal:** Apply minimal fix to ensure tag comparison for nullary patterns.

**Likely Fix Location 1: Decision Tree** (~30 min)
```go
// File: internal/dtree/decision_tree.go (or compile.go if exists)
func compilePattern(pat Pattern) DecisionTree {
    switch p := pat.(type) {
    case VariantPattern:
        // FIX: Always emit TagTest, even when arity=0
        return TagTest{
            Tag:      p.Tag,
            ThenTree: compileSubPatterns(p.Subpatterns),  // Empty for arity=0
            ElseTree: Fail,
        }
    }
}
```
**Estimated:** ~5-10 LOC change

**Likely Fix Location 2: Runtime Evaluation** (~30 min)
```go
// File: internal/eval/eval_typed.go
func (e *TypedEvaluator) evalMatch(match *typedast.TypedMatch) (Value, error) {
    // ... existing code ...

    // FIX: Check tag FIRST, before arity check
    if variantValue.Tag != patternTag {
        return false, nil  // Tags don't match
    }

    // Now handle arguments (if arity > 0)
    if arity == 0 {
        return true, emptyBindings()
    }
    // ... rest of matching logic
}
```
**Estimated:** ~5-10 LOC change

**Likely Fix Location 3: Pattern Lowering** (~15 min)
```go
// File: internal/elaborate/patterns.go
func (e *Elaborator) elaboratePattern(pat *ast.Pattern) *core.Pattern {
    switch p := pat.Kind.(type) {
    case *ast.VariantPattern:
        variant := e.lookupVariant(p.Name)
        // FIX: Ensure tag is set for nullary variants
        return &core.PatternVariant{
            Tag:   variant.Tag,
            Arity: variant.Arity,  // 0 for nullary
            Args:  nil,            // Empty, not omitted
        }
    }
}
```
**Estimated:** ~5 LOC change

**Test Fix Manually** (~15 min)
```bash
ailang run /tmp/test_nullary.ail
# Should print: f(A) = "a", f(B) = "b", f(C) = "c"
```

**Run Existing Tests** (~15 min)
```bash
# Ensure no regressions
go test ./internal/eval -v
go test ./internal/dtree -v
go test ./internal/elaborate -v
```

**Acceptance Criteria:**
- [ ] Fix implemented in 1-3 files (~15-25 LOC total)
- [ ] Manual test case works (all three variants match correctly)
- [ ] Existing tests still pass (zero regressions)
- [ ] Code compiles and lints clean

**Deliverables:**
- Modified files: 1-3 files (~15-25 LOC changed)
- Commit message documenting bug location and fix

---

### Day 2: Testing & Documentation (1-2 hours)

#### Session 3: Write Tests (1 hour)

**Unit Tests** (~40 min)
```go
// File: internal/eval/eval_typed_test.go (create if doesn't exist)
func TestNullaryConstructorMatching(t *testing.T) {
    tests := []struct {
        name     string
        code     string
        expected string
    }{
        {
            name: "nullary enum - first variant",
            code: `
                type Color = Red | Green | Blue
                func test() -> string {
                    match Red { Red => "R", Green => "G", Blue => "B" }
                }
                test()
            `,
            expected: "R",
        },
        {
            name: "nullary enum - second variant",
            code: `
                type Color = Red | Green | Blue
                func test() -> string {
                    match Green { Red => "R", Green => "G", Blue => "B" }
                }
                test()
            `,
            expected: "G",
        },
        {
            name: "nullary enum - third variant",
            code: `
                type Color = Red | Green | Blue
                func test() -> string {
                    match Blue { Red => "R", Green => "G", Blue => "B" }
                }
                test()
            `,
            expected: "B",
        },
        {
            name: "nullary enum - via parameter",
            code: `
                type Status = Pending | InProgress | Completed
                func describe(s: Status) -> string {
                    match s {
                        Pending => "Waiting",
                        InProgress => "Working",
                        Completed => "Done"
                    }
                }
                describe(InProgress)
            `,
            expected: "Working",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := evalCode(t, tt.code)
            if result != tt.expected {
                t.Errorf("expected %q, got %q", tt.expected, result)
            }
        })
    }
}
```

**Estimated:** ~40 LOC new test file

**Integration Test** (~20 min)
```ailang
// File: tests/nullary_pattern_matching_test.ail
module tests/nullary_pattern_matching

type Status = Pending | InProgress | Completed

export func describeStatus(s: Status) -> string {
  match s {
    Pending => "Waiting",
    InProgress => "Working",
    Completed => "Done"
  }
}

export func main() -> () ! {IO} {
  assert(describeStatus(Pending) == "Waiting");
  assert(describeStatus(InProgress) == "Working");
  assert(describeStatus(Completed) == "Done");
  print("✓ All nullary pattern matching tests passed")
}
```

**Estimated:** ~20 LOC new example file

**Run Test Suite** (~10 min)
```bash
# Unit tests
go test ./internal/eval -v -run TestNullary

# Integration test
ailang run --caps IO --entry main tests/nullary_pattern_matching_test.ail

# Regression tests
ailang run examples/option.ail  # Mixed ADT (CRITICAL)
ailang run examples/list_operations.ail  # List patterns
```

**Acceptance Criteria:**
- [ ] 4 unit tests pass (all variants match correctly)
- [ ] Integration test passes
- [ ] Regression tests pass (Option, lists still work)
- [ ] `make test` passes (all existing tests)

**Deliverables:**
- `internal/eval/eval_typed_test.go` (+40 LOC)
- `tests/nullary_pattern_matching_test.ail` (+20 LOC)

---

#### Session 4: Benchmarks & Documentation (30-60 min)

**Run Eval Benchmarks** (~20 min)
```bash
# Run exhaustive_pattern_matching benchmark
ailang eval-suite \
  --benchmarks exhaustive_pattern_matching \
  --langs ailang \
  --models gpt5,claude-sonnet-4-5
```

**Expected Results:**
- Before: 96.1% (73/76 passing, 3 failures)
- After: **100% (76/76 passing)**

**Update Documentation** (~20 min)

**CHANGELOG.md:**
```markdown
## [v0.4.5] - 2025-11-XX

### Fixed
- **Pattern matching**: Nullary constructor pattern matching now works correctly
  - Bug: ADTs with only nullary constructors (e.g., `type E = A | B | C`) always matched first pattern
  - Fix: Ensure tag comparison happens for all patterns, including arity=0 variants
  - Location: `internal/[dtree|eval|elaborate]/*.go` (~15-25 LOC)
  - Tests: 4 new unit tests, 1 integration test (~60 LOC)
  - Impact: `exhaustive_pattern_matching` benchmark: 96.1% → 100% success
  - Enables production use of simple enum types (Status, Color, Direction, etc.)
```

**docs/LIMITATIONS.md** (if applicable):
- Remove limitation about nullary ADTs if documented
- Or add note that it's fixed in v0.4.5

**Commit Messages:**
```bash
git add internal/[modified files] tests/nullary_pattern_matching_test.ail
git commit -m "Fix nullary constructor pattern matching bug (M-BUG-NULLARY)

- Bug: ADTs with only nullary constructors matched first pattern for all values
- Root cause: Tag comparison missing/skipped when arity=0
- Fix: [Describe specific fix location]
- Tests: 4 unit tests + 1 integration test
- Benchmark: exhaustive_pattern_matching 96.1% → 100%
- Closes v0.4.5 M-BUG-NULLARY
"
```

**Acceptance Criteria:**
- [ ] Benchmark shows 100% success (76/76)
- [ ] CHANGELOG.md updated with fix details
- [ ] Commit messages clear and descriptive
- [ ] All documentation accurate

**Deliverables:**
- Updated `CHANGELOG.md` (+10 LOC)
- Updated `docs/LIMITATIONS.md` (if applicable)
- Benchmark results showing 100% success

---

## Success Metrics

### Test Coverage
- **Unit tests**: 4 new tests covering all nullary enum cases
- **Integration test**: 1 end-to-end test with IO
- **Regression tests**: Option type, list patterns still work
- **Overall coverage**: No decrease from current ~90%

### Examples
- ✅ `tests/nullary_pattern_matching_test.ail` - Passes
- ✅ `examples/option.ail` - Still passes (regression check)
- ✅ `examples/list_operations.ail` - Still passes (regression check)

### Benchmarks
- **Target**: `exhaustive_pattern_matching` → **100% success**
- **Current**: 96.1% (73/76)
- **After fix**: 100% (76/76)
- **AI Codegen**: Successfully generates nullary enum code

### Documentation
- [x] `CHANGELOG.md` - Document fix
- [x] `docs/LIMITATIONS.md` - Remove limitation or note fix
- [x] Design doc - Complete (already done)
- [x] Commit messages - Clear root cause and fix

### Code Quality
- [x] All tests passing: `make test`
- [x] Linting clean: `make lint`
- [x] No regressions: Existing pattern matching works
- [x] Performance: No measurable regression (<1%)

---

## Task Checklist

### Day 1: Investigation & Fix
- [ ] **Phase 1**: Check Core AST representation (30 min)
  - [ ] Create minimal test case `/tmp/test_nullary.ail`
  - [ ] Run `ailang debug ast --show-types`
  - [ ] Verify tags are distinct (0, 1, 2)
- [ ] **Phase 2**: Check decision tree (30 min)
  - [ ] Grep for nullary handling in `internal/dtree/`
  - [ ] Read decision tree compilation code
  - [ ] Look for arity=0 special cases
- [ ] **Phase 3**: Check evaluator (30 min)
  - [ ] Find `evalMatch` in `internal/eval/eval_typed.go`
  - [ ] Check tag comparison logic
  - [ ] Look for arity checks before tag checks
- [ ] **Phase 4**: Debug logging (30 min if needed)
  - [ ] Add logging for nullary variants
  - [ ] Run test and observe tag values
- [ ] **Implement fix** (1-2 hours)
  - [ ] Apply fix to 1-3 files (~15-25 LOC)
  - [ ] Test manually with `/tmp/test_nullary.ail`
  - [ ] Run existing tests (ensure no regressions)
  - [ ] Commit with clear message

### Day 2: Testing & Documentation
- [ ] **Write unit tests** (40 min)
  - [ ] Create `internal/eval/eval_typed_test.go` if needed
  - [ ] Add `TestNullaryConstructorMatching` with 4 test cases
  - [ ] Run tests: `go test ./internal/eval -v`
- [ ] **Write integration test** (20 min)
  - [ ] Create `tests/nullary_pattern_matching_test.ail`
  - [ ] Test all three Status variants
  - [ ] Run: `ailang run --caps IO --entry main tests/...`
- [ ] **Regression tests** (10 min)
  - [ ] Run `examples/option.ail` (mixed ADT - CRITICAL)
  - [ ] Run `examples/list_operations.ail` (list patterns)
  - [ ] Run `make test` (all tests)
- [ ] **Run benchmarks** (20 min)
  - [ ] Run `exhaustive_pattern_matching` eval suite
  - [ ] Verify 100% success (76/76)
  - [ ] Document results
- [ ] **Update documentation** (20 min)
  - [ ] Update `CHANGELOG.md` with fix details
  - [ ] Update `docs/LIMITATIONS.md` if applicable
  - [ ] Write clear commit message
- [ ] **Final verification** (10 min)
  - [ ] `make test` - All tests pass
  - [ ] `make lint` - Linting clean
  - [ ] `make ci` - Local CI passes
  - [ ] Review all changes

---

## Risks & Mitigations

| Risk | Impact | Probability | Mitigation |
|------|--------|-------------|-----------|
| Fix breaks `Option` type `None` matching | **High** | Low (5%) | Test `None` explicitly before merge; if broken, fix applies to wrong code path |
| Fix breaks list pattern matching (`::`) | **High** | Very Low (1%) | Run `list_operations.ail` before and after; list patterns use different code |
| Bug exists in multiple locations | Medium | Medium (30%) | Fix all locations found during investigation; comprehensive testing |
| Decision tree optimization conflicts | Medium | Low (10%) | Preserve existing optimizations, only add tag test for arity=0 |
| Performance regression | Low | Low (5%) | Benchmark before/after; tag check is O(1) operation |
| Edge case missed (e.g., 2 vs 3+ variants) | Medium | Low (10%) | Test with 2, 3, and 4+ variant enums |

---

## Dependencies

### None (Standalone Bug Fix)
This is an isolated bug fix with no dependencies on other work.

### Blocks
- Production use of simple enum types
- `exhaustive_pattern_matching` benchmark reaching 100%
- AI code generation improvements

---

## Open Questions

### None Currently
All aspects are well-understood from investigation and design doc.

**If issues arise during implementation:**
1. **Multiple fix locations needed?** → Fix all, test each independently
2. **Performance impact?** → Benchmark and optimize if needed
3. **Backward compatibility?** → Not an issue (this is a bug fix, not breaking change)

---

## Notes

### Assumptions
- Bug is in one of three locations: decision tree, elaboration, or evaluator
- Mixed ADTs work correctly, so core pattern matching is sound
- Fix will be small (~15-25 LOC) and isolated
- Comprehensive tests will prevent regressions

### Context
- This bug was discovered during v0.4.4 eval analysis
- 3 out of 76 eval failures (96.1% success) are due to this single bug
- List pattern matching works correctly (confirmed), narrowing scope
- User can't use simple enums in production (high priority)

### Post-Sprint
After this fix ships:
- Consider adding exhaustiveness warnings for other edge cases
- Improve pattern match error messages
- Add pattern match compilation metrics for debugging
- Optimize decision tree for simple enums (compile to switch)

---

## Timeline Summary

**Total Estimated Time:** 3-5 hours across 1-2 days

| Day | Session | Duration | Tasks |
|-----|---------|----------|-------|
| 1 | Investigation | 1-2h | Locate bug (4 phases) |
| 1 | Fix | 1-2h | Implement fix + manual testing |
| 2 | Testing | 1h | Unit + integration + regression tests |
| 2 | Docs | 0.5-1h | Benchmarks + CHANGELOG + commit |

**Flexible Schedule:**
- Can be completed in one 4-5 hour focused session
- Or split across 2 days with 2-3 hour sessions
- Investigation may take less time if bug found quickly
- Testing may take longer if regressions found

**Realistic Estimate:** 4 hours (midpoint of 3-5 hour range)

---

## Approval

**Status:** Ready for implementation
**Reviewed by:** [User to approve]
**Approved:** [ ] Yes / [ ] No / [ ] Needs revision

**Feedback:**
[Space for user to provide feedback or request changes]