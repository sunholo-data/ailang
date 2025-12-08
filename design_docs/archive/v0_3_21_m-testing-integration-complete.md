# M-TESTING-INTEGRATION-COMPLETE: Pipeline Integration for Test Execution

**Status**: Planned
**Target**: v0.3.21
**Priority**: P0 (High)
**Estimated**: 1.5-2 days (12-16 hours)
**Dependencies**: None (all infrastructure from M-TESTING Days 1-10 exists)

## AI-First Alignment Check

**Score this feature against AILANG's core principles:**

| Principle | Impact | Score | Notes |
|-----------|--------|-------|-------|
| Reduce Syntactic Noise | + | +1 | Inline tests reduce separate test file boilerplate compared to traditional test frameworks |
| Preserve Semantic Clarity | + | +1 | Tests are explicit boolean expressions, properties formalize invariants |
| Increase Determinism | + | +1 | Property-based testing systematically finds edge cases with deterministic seeding |
| Lower Token Cost | 0 | 0 | Neutral - adds syntax for tests but increases AI confidence and reduces debugging tokens |
| **Net Score** | | **+3** | **Decision: Move forward** |

**Decision rule:** Net score > +1 → Move forward | ≤ 0 → Reject or redesign

**Reference:** See [AI-first DX philosophy](../v0_3_15/example-parity-vision-alignment.md#-design-principle-ai-first-dx)

## Problem Statement

**The Situation:** M-TESTING was shipped in v0.3.20 as "complete" but the test execution is stubbed out. Tests are parsed, collected, and reported, but never actually run.

**Current State:**
- `ailang test` command exists but returns `StatusSkip` for all tests
- Runner stubs at [internal/testing/runner.go:42-71](../../internal/testing/runner.go#L42-L71) say "requires pipeline integration"
- Example files (`testing_basic.ail`, `testing_advanced.ail`) lack module structure and fail to parse
- CHANGELOG v0.3.20 claims "property-based testing with automatic shrinking" is complete
- All infrastructure exists (parser, collector, generators, shrinking, reporter) but pipeline integration was never done

**Evidence:**
```bash
$ ailang test examples/testing_basic.ail
→ Running tests in examples/testing_basic.ail

Test Results
Module: All Tests

Tests:
  ✗ parse  # ← File doesn't even parse!

✗ Some tests failed
1 tests: 0 passed, 1 failed, 0 skipped (0s)
```

**Impact:**
- **Users:** Feature shipped but unusable - cannot write or run tests
- **AI Models:** Cannot learn AILANG testing patterns (prompts would show non-functional examples)
- **Development:** Cannot use tests to validate AI-generated code in M-EVAL
- **Credibility:** CHANGELOG claims completion of feature that doesn't work
- **Documentation:** 650+ lines of testing documentation describes non-functional feature

**Root Cause Analysis:**
M-TESTING was completed incrementally over 10 days (see CHANGELOG v0.3.20):
- Days 1-3: Parser and AST support ✅
- Days 4-5: Test collection and reporting ✅
- Days 6-7: Generators for property-based testing ✅
- Days 8: Shrinking algorithm ✅
- Days 9: CLI command ✅
- Days 10: Documentation and examples ✅

**But:** Pipeline integration to actually *execute* tests was never implemented. The runner [internal/testing/runner.go](../../internal/testing/runner.go) still has Day 4 stubs that return `StatusSkip`.

## Goals

**Primary Goal:** Make `ailang test` command fully functional by integrating test execution with the compilation pipeline, completing the M-TESTING feature as originally intended.

**Success Metrics:**
- `ailang test examples/testing_basic.ail` → Shows pass/fail results (not skip)
- Property tests generate 100 random cases and report actual results
- Shrinking finds minimal counterexamples when properties fail
- Test example files are runnable modules (`ailang run --entry main`)
- `make test-ailang` target runs AILANG tests in CI
- Zero tests with `StatusSkip` (unless explicitly marked with `skip` keyword)

**Completion Criteria:**
- All 110+ existing Go tests for testing infrastructure still pass
- New integration tests verify end-to-end test execution
- Documentation updated to reflect working implementation
- CHANGELOG entry clarifies v0.3.21 completes v0.3.20's testing feature

## Solution Design

### Overview

The solution completes M-TESTING by implementing the missing pipeline integration. All hard parts already exist:

✅ **Already Done:**
- Test syntax parsing (`test "name" = expr`)
- Property syntax parsing (`property "name" (x: type) = expr`)
- Test collection from AST (`internal/testing/collector.go`)
- Type-aware generators (`internal/testing/generator.go`, `generator_advanced.go`)
- Shrinking algorithm (`internal/testing/shrink.go`)
- Reporter (human & JSON formats)
- CLI command (`ailang test`)

❌ **Missing (this design doc):**
- Pipeline integration to evaluate test expressions
- Property execution with generated inputs
- Example files with proper module structure

**Key Insight:** This is a 12-16 hour task because we're just wiring existing components together. No new algorithms or infrastructure needed.

### Architecture

**Current Flow (v0.3.20):**
```
ailang test file.ail
  → Parse file
  → Collect tests (TestCase, PropertyCase structs)
  → Runner.runTest() → return StatusSkip "requires pipeline integration"
  → Reporter shows all tests skipped
```

**New Flow (v0.3.21):**
```
ailang test file.ail
  → Parse file
  → Collect tests
  → Runner.runTest():
      1. Take test.Expression (AST node)
      2. Run through pipeline:
         - Elaborate to Core
         - Type check
         - Evaluate with empty EffContext
      3. Check if result is BoolValue(true)
      4. Return TestResult{Status: Pass/Fail}
  → Runner.runProperty():
      1. Extract property parameters and types
      2. Use existing generators to create 100 test cases
      3. For each generated input:
         a. Bind inputs to property parameters (create Let bindings)
         b. Evaluate property expression through pipeline
         c. Check if result is BoolValue(true)
         d. If false, use existing shrinking to find minimal counterexample
      4. Return PropertyResult with pass/fail + counterexample
  → Reporter shows actual pass/fail results
```

**Components:**

1. **Test Executor** (`internal/testing/executor.go` - NEW)
   - `EvaluateTestExpression(expr ast.Expr) (bool, error)` - Run expression through pipeline
   - `BindPropertyInputs(params []ast.Param, values []eval.Value) ast.Expr` - Create Let bindings
   - Helper to run pipeline: parse → elaborate → typecheck → eval

2. **Runner Integration** (`internal/testing/runner.go` - MODIFY)
   - Replace `runTest()` stub with real implementation
   - Replace `runProperty()` stub with generator integration
   - Add error handling for compile/runtime failures

3. **Example Fixes** (`examples/testing_*.ail` - MODIFY)
   - Add module declarations
   - Add imports for any IO operations
   - Make examples executable

### Implementation Plan

**Phase 1: Core Pipeline Integration** (~8 hours)

- [ ] Create `internal/testing/executor.go`:
  - [ ] `EvaluateTestExpression()` - Run test expr through pipeline (~1 hour)
  - [ ] `BindPropertyInputs()` - Create Let bindings for property params (~1 hour)
  - [ ] `CompileAndEval()` - Helper for full pipeline execution (~1 hour)
  - [ ] Unit tests for executor (~1 hour)

- [ ] Update `internal/testing/runner.go`:
  - [ ] Replace `runTest()` stub with real execution (~1 hour)
  - [ ] Replace `runProperty()` stub with generator loop (~2 hours)
  - [ ] Add error handling for compile/runtime failures (~0.5 hour)
  - [ ] Integration tests for runner (~1.5 hours)

**Phase 2: Example Fixes & Documentation** (~2 hours)

- [ ] Fix `examples/testing_basic.ail`:
  - [ ] Add `module testing/basic`
  - [ ] Add `import std/io (println)` if needed
  - [ ] Add `export func main()` to make it runnable
  - [ ] Verify all tests parse and execute (~0.5 hour)

- [ ] Fix `examples/testing_advanced.ail`:
  - [ ] Same module structure changes
  - [ ] Verify ADT tests work (~0.5 hour)

- [ ] Update documentation:
  - [ ] Update `docs/TESTING.md` with working examples (~0.5 hour)
  - [ ] Update `prompts/testing_guide_ai.md` with executable examples (~0.5 hour)

**Phase 3: CI Integration** (~2 hours)

- [ ] Add `make test-ailang` target (~0.5 hour)
- [ ] Add AILANG tests to CI pipeline (~0.5 hour)
- [ ] Add pre-commit hook example to docs (~0.5 hour)
- [ ] Verify CI runs successfully (~0.5 hour)

**Buffer:** ~2-4 hours for unexpected issues

**Total: 12-16 hours**

### Files to Modify/Create

**New files:**
- `internal/testing/executor.go` - Pipeline execution helpers (~150 LOC)
- `internal/testing/executor_test.go` - Unit tests (~200 LOC)

**Modified files:**
- `internal/testing/runner.go` - Replace stubs (~100 LOC changed)
- `internal/testing/runner_test.go` - Add integration tests (~150 LOC added)
- `examples/testing_basic.ail` - Add module structure (~20 LOC added)
- `examples/testing_advanced.ail` - Add module structure (~20 LOC added)
- `Makefile` - Add test-ailang target (~15 LOC added)
- `docs/TESTING.md` - Update examples (~50 LOC changed)
- `prompts/testing_guide_ai.md` - Update examples (~50 LOC changed)

**Total new code: ~750 LOC**
**Test-to-code ratio: ~2:1** (good!)

## Examples

### Example 1: Basic Test Execution

**Before (v0.3.20 - Skipped):**
```bash
$ ailang test examples/testing_basic.ail

Test Results
Tests:
  ⊘ addition works (skipped)
  ⊘ string equality (skipped)

2 tests: 0 passed, 0 failed, 2 skipped
```

**After (v0.3.21 - Executed):**
```bash
$ ailang test examples/testing_basic.ail

Test Results
Tests:
  ✓ addition works
  ✓ string equality

2 tests: 2 passed, 0 failed, 0 skipped (0.2s)
```

### Example 2: Property-Based Test with Shrinking

**Test code:**
```ailang
property "all positive integers are less than 1000" (x: int) =
  x < 1000
```

**Before (v0.3.20 - Skipped):**
```bash
Properties:
  ⊘ all positive integers are less than 1000 (skipped)
```

**After (v0.3.21 - Fails with Counterexample):**
```bash
Properties:
  ✗ all positive integers are less than 1000 (100 cases)
    Counterexample found after shrinking: x = 1000
    Original failing input: x = 2817
```

### Example 3: Module Structure for Examples

**Before (v0.3.20 - No module structure):**
```ailang
// testing_basic.ail

test "addition works" = 1 + 1 == 2
test "string equality" = "hello" == "hello"
```

**After (v0.3.21 - Runnable module):**
```ailang
module testing/basic

import std/io (println)

// Tests (executed by `ailang test`)
test "addition works" = 1 + 1 == 2
test "string equality" = "hello" == "hello"

// Main function (for `ailang run`)
export func main() -> () ! {IO} {
  println("Testing examples - use `ailang test` to run tests")
}
```

## Success Criteria

- [ ] `ailang test examples/testing_basic.ail` shows pass/fail results (not skip)
- [ ] `ailang test examples/testing_advanced.ail` shows pass/fail results
- [ ] Property tests generate 100 cases per property and report actual results
- [ ] Failing properties report minimal counterexample via shrinking algorithm
- [ ] Test examples are runnable modules: `ailang run --caps IO --entry main testing_basic.ail` works
- [ ] All 110+ existing Go tests for testing infrastructure still pass
- [ ] New integration tests verify end-to-end test execution (at least 5 test cases)
- [ ] `make test-ailang` target works and runs in CI
- [ ] Documentation updated to reflect working implementation
- [ ] CHANGELOG entry for v0.3.21 clarifies completion of M-TESTING
- [ ] Zero tests with `StatusSkip` unless explicitly marked with `skip` keyword (future feature)

## Testing Strategy

**Unit tests:**
- `internal/testing/executor_test.go`:
  - Test `EvaluateTestExpression()` with various expressions (literals, operators, function calls)
  - Test `BindPropertyInputs()` with different parameter types
  - Test error handling for malformed expressions
  - (~10 test cases, ~200 LOC)

**Integration tests:**
- `internal/testing/runner_test.go` (additions):
  - Test full workflow: parse → collect → run → report
  - Test property execution with generators
  - Test shrinking on property failure
  - Test multiple tests in one file
  - (~5-7 test cases, ~150 LOC)

**Manual testing:**
- Run `ailang test` on all example files
- Verify pass/fail/skip counts are correct
- Verify counterexample reporting for failing properties
- Run `make test-ailang` to verify CI integration

**Regression testing:**
- Run full Go test suite: `make test`
- Verify no breaking changes to existing infrastructure

## Non-Goals

**Not in this feature:**
- **Eval harness integration** - Using AILANG tests to validate AI-generated code in M-EVAL benchmarks (future: v0.4.0)
- **Watch mode** - `ailang test --watch` for TDD workflow (future: v0.4.1)
- **Inline function tests** - `func factorial(n) tests [(0,1), (5,120)]` syntax (future: v0.4.2)
- **Coverage reporting** - Which code paths are tested (future: v0.4.x)
- **Parallel test execution** - Running tests concurrently (future: v0.5.0)
- **Test filtering** - `ailang test --filter "arithmetic"` (future: v0.4.1)

**Why deferred:**
- **Eval harness integration**: Needs design work on benchmark spec format
- **Watch mode**: Requires file watching infrastructure
- **Inline function tests**: Needs parser changes and syntax design
- **Other features**: Nice-to-have but not critical for v0.3.21

## Timeline

**Day 1** (8 hours):
- Morning: Create `executor.go` with pipeline integration (4 hours)
- Afternoon: Update `runner.go` with real execution (4 hours)

**Day 2** (4-6 hours):
- Morning: Integration tests and example fixes (2-3 hours)
- Afternoon: CI integration and documentation (2-3 hours)

**Buffer:** 2-4 hours for debugging and edge cases

**Total: 12-16 hours across 1.5-2 days**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Pipeline integration breaks existing tests | High | Run full test suite after each change; keep changes minimal |
| Property execution is too slow (100 cases × N properties) | Medium | Add timeout per property; make iteration count configurable |
| Shrinking algorithm doesn't work with pipeline | Medium | Shrinking is already implemented and tested; just needs integration |
| Example files reveal parser bugs | Low | Parser has 48 working examples; testing syntax is well-tested |
| CI integration causes flaky tests | Low | Use deterministic seeding for generators; no network/IO in tests |

## References

- **M-TESTING Design Doc** (original): [design_docs/planned/v0_4_2/m-testing-property-based-testing.md](../v0_4_2/m-testing-property-based-testing.md)
- **CHANGELOG v0.3.20**: M-TESTING Days 1-10 completion
- **Testing Documentation**: [docs/TESTING.md](../../docs/TESTING.md)
- **AI Teaching Guide**: [prompts/testing_guide_ai.md](../../prompts/testing_guide_ai.md)
- **Pipeline Architecture**: [internal/pipeline/pipeline.go](../../internal/pipeline/pipeline.go)
- **Generator Implementation**: [internal/testing/generator.go](../../internal/testing/generator.go)
- **Shrinking Algorithm**: [internal/testing/shrink.go](../../internal/testing/shrink.go)

## Future Work

**v0.4.0 - Eval Harness Integration:**
- Use AILANG tests to validate AI-generated code in benchmarks
- Add `validation: ailang_test` mode to benchmark specs
- Measure AI capability to generate working tests/properties

**v0.4.1 - Enhanced Testing Features:**
- Watch mode for TDD: `ailang test --watch`
- Test filtering: `ailang test --filter "arithmetic"`
- Verbose output: `ailang test --verbose`
- Seed control: `ailang test --seed 42`

**v0.4.2 - Inline Function Tests:**
- Syntax: `func factorial(n: int) -> int tests [(0,1), (5,120)] { ... }`
- Execute tests on function definition
- Validate examples in documentation

**v0.5.0 - Advanced Property Testing:**
- Custom generators: `property "foo" (x: MyType @ customGen) = ...`
- Stateful property testing: test sequences of operations
- Parallel test execution for faster feedback

---

**Document created**: 2025-10-27
**Last updated**: 2025-10-27
