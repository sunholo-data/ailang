# M-TESTING-INLINE-COMPLETION: Complete Inline Function Tests Feature

## Summary

**Complete the remaining 5-10% of inline function testing to enable AI-friendly test-driven development.** Parser, AST, generators, shrinking, and CLI are 100% done (4,892 LOC). Only pipeline integration (~200 LOC) and examples (~150 LOC) remain.

**Duration:** 1-2 days (12-16 hours)
**Dependencies:** None - all infrastructure exists
**Risk Level:** Low (mostly integration, no new algorithms)

## Current Status Analysis

### 🎉 Already Completed (90-95% done!)

**Discovered 2025-11-27:** The M-TESTING feature from v0.3.20 is nearly complete!

#### ✅ Parser & AST (100% complete - 318 LOC)
- `internal/parser/parser_testing.go` - Full implementation
  - `parseTestsBlock()` - Parses `tests [(input, expected)]`
  - `parsePropertiesBlock()` - Parses `properties [forall(...) => expr]`
  - `parseTestCase()`, `parseProperty()`, `parseBinder()` - All done
- AST nodes: `FuncDecl.Tests`, `FuncDecl.Properties` fields exist
- Lexer: `TESTS`, `PROPERTIES`, `FORALL` keywords defined

#### ✅ Testing Infrastructure (100% complete - 4,574 LOC)
- **Collector** (130 LOC): `internal/testing/collector.go`
- **Generators** (532 LOC): Basic + advanced type-aware generators
  - `generator.go` (264 LOC) - int, float, bool, string, list
  - `generator_advanced.go` (268 LOC) - ADT, Option, Result, records
- **Shrinking** (309 LOC): `internal/testing/shrink.go` - Minimal counterexample finder
- **Reporter** (267 LOC): JSON + human-readable output
- **Runner stub** (85 LOC): `internal/testing/runner.go` (returns `StatusSkip`)
- **Tests** (2,409 LOC): 100+ test cases covering all infrastructure

#### ✅ CLI Command (100% complete)
- `cmd/ailang/test.go` - Full implementation
- Format options: `--format json`, `--no-color`
- Exit codes: 0 (pass), 1 (fail)

### ❌ Remaining Work (5-10% - 350-400 LOC)

#### 1. Pipeline Integration (~200 LOC, 6-8 hours)
**Status:** Runner stubs return `StatusSkip` - need real evaluation

**Files to modify:**
- `internal/testing/runner.go` - Replace stubs with pipeline calls
- `internal/testing/executor.go` - NEW: Pipeline evaluation helpers

**What's needed:**
```go
// Current (runner.go:43-54):
func (r *Runner) runTest(testCase TestCase) TestResult {
    return TestResult{
        Status: StatusSkip,  // ← Need to actually evaluate!
        Error: "Test execution requires pipeline integration",
    }
}

// Need to add:
// 1. Evaluate test expression through pipeline
// 2. Bind property inputs to forall parameters
// 3. Generate test cases with existing generators
// 4. Use existing shrinking on failure
```

#### 2. Example Files (~100 LOC, 2-3 hours)
**Status:** Example files exist but lack inline tests

**Files to update:**
- `examples/factorial.ail` - Add inline tests
- `examples/quicksort.ail` - Add properties
- `examples/testing_basic.ail` - Fix parse error (add module declaration)
- `examples/testing_advanced.ail` - Fix parse error (add module declaration)

#### 3. Integration Tests (~100 LOC, 2-3 hours)
**Status:** Unit tests exist, need end-to-end tests

**Files to create:**
- `internal/testing/integration_inline_test.go` - Test inline tests actually execute
- Test pipeline integration works correctly

#### 4. Documentation (~50 LOC, 1-2 hours)
**Status:** Full documentation exists, needs inline syntax examples

**Files to update:**
- `prompts/vX.Y.Z.md` - Add inline test syntax to teaching prompt
- `CLAUDE.md` - Update testing section with inline examples
- `docs/TESTING.md` - Add inline tests section

### Recent Velocity Analysis

From last 14 days (CHANGELOG analysis):
- **Operator Logic Update**: ~260 LOC (implementation: 110, tests: 150)
- **Nullary Pattern Fix**: ~133 LOC (elaboration: 13, tests: 120)
- **Pattern Sugar**: ~519 LOC (parser: 49, tests: 470)

**Average velocity**: ~30-40 LOC/day net change
**This sprint requires**: ~350-400 LOC
**Timeline**: 1-2 days is realistic (10-12 hours of focused work)

## Proposed Milestones

### Milestone 1: Pipeline Integration (Day 1, ~200 LOC)

**Goal:** Replace runner stubs with actual test execution through the AILANG pipeline

**Duration:** 6-8 hours

**Tasks:**

**Morning (4 hours):**
- Create `internal/testing/executor.go`:
  - `EvaluateTestExpression(expr ast.Expr) (Value, error)` - Run through pipeline
  - Helper: `CompileAndEval(expr)` - Full pipeline: elaborate → typecheck → eval
  - ~100 LOC

- Update `internal/testing/runner.go`:
  - Replace `runTest()` stub with real execution
  - For inline tests: bind inputs, evaluate expected vs actual
  - ~50 LOC modified

**Afternoon (2-4 hours):**
- Implement property execution in `runner.go`:
  - Use existing generators to create test cases
  - Bind forall parameters to generated values
  - Use existing shrinking on failure
  - ~50 LOC modified

- Write unit tests for executor:
  - Test expression evaluation
  - Test error handling
  - ~50 LOC

**Acceptance Criteria:**
- [ ] `internal/testing/executor.go` created with pipeline helpers
- [ ] `runTest()` evaluates test expressions (not `StatusSkip`)
- [ ] `runProperty()` generates test cases with existing generators
- [ ] Shrinking finds minimal counterexamples on failure
- [ ] Unit tests verify executor works
- [ ] All existing 110+ tests still pass

**Example test case that should pass:**
```ailang
pure func factorial(n: int) -> int
  tests [
    (0, 1),
    (5, 120)
  ]
{
  if n <= 1 then 1 else n * factorial(n - 1)
}

// After implementation:
// $ ailang test examples/factorial.ail
// ✓ factorial tests passed (2/2)
```

**Risks:**
- Pipeline integration might reveal edge cases - **Mitigation:** Start with simple expressions, add complexity incrementally
- Type checking inline tests might need special handling - **Mitigation:** Use existing `EvaluateExpression` pattern from REPL

### Milestone 2: Examples & Integration Tests (Day 2, ~200 LOC)

**Goal:** Create working example files and verify end-to-end functionality

**Duration:** 4-6 hours

**Tasks:**

**Morning (2-3 hours):**
- Fix `examples/testing_basic.ail`:
  - Add `module testing/basic` declaration
  - Add `export func main()` for `ailang run`
  - Verify all tests parse and execute
  - ~30 LOC changes

- Fix `examples/testing_advanced.ail`:
  - Add module declaration
  - Verify ADT tests work
  - ~30 LOC changes

- Update `examples/factorial.ail`:
  - Add inline tests: `tests [(0, 1), (1, 1), (5, 120)]`
  - ~20 LOC added

- Update `examples/quicksort.ail` (or create if missing):
  - Add properties: idempotence, length preservation
  - ~40 LOC added

**Afternoon (2-3 hours):**
- Create `internal/testing/integration_inline_test.go`:
  - Test inline tests execute correctly
  - Test properties generate test cases
  - Test shrinking works end-to-end
  - Test error reporting
  - ~100 LOC

- Update documentation:
  - Add inline test syntax to latest teaching prompt
  - Update `CLAUDE.md` with examples
  - Update `docs/TESTING.md` with inline tests section
  - ~50 LOC total

**Acceptance Criteria:**
- [ ] `examples/testing_basic.ail` parses and all tests pass
- [ ] `examples/testing_advanced.ail` parses and all tests pass
- [ ] `examples/factorial.ail` has working inline tests
- [ ] `examples/quicksort.ail` has working properties (if implemented)
- [ ] `ailang test examples/` runs all examples successfully
- [ ] Integration tests verify end-to-end flow
- [ ] Documentation updated with inline test examples
- [ ] `make verify-examples` passes

**Example outputs to verify:**

1. **Inline tests passing:**
```bash
$ ailang test examples/factorial.ail

→ Running tests in examples/factorial.ail

Test Results
Module: factorial

Inline Tests:
  ✓ factorial (4 cases)

✓ All tests passed
1 function: 4 tests passed, 0 failed (0.1s)
```

2. **Properties with shrinking:**
```bash
$ ailang test examples/quicksort.ail

→ Running tests in examples/quicksort.ail

Test Results
Module: quicksort

Properties:
  ✓ idempotence (100 cases)
  ✓ length preservation (100 cases)

✓ All tests passed
2 properties: 200 cases passed, 0 failed (0.3s)
```

**Risks:**
- Example files might reveal parser edge cases - **Mitigation:** Start with simplest examples, gradually add complexity
- Module system integration might need adjustment - **Mitigation:** Follow existing example patterns

## Success Metrics

### Quantitative
- **LOC added/modified**: 350-400 (target: 200 implementation, 150 tests, 50 docs)
- **Test coverage**: Maintain >80% for `internal/testing/` package
- **Examples passing**: 4+ example files with inline tests working
- **Integration tests**: 5-10 end-to-end test cases
- **All CI checks passing**: `make test`, `make lint`, `make verify-examples`

### Qualitative
- [ ] `ailang test examples/factorial.ail` shows pass/fail (not skip)
- [ ] Property tests generate 100 cases per property
- [ ] Shrinking finds minimal counterexamples on failure
- [ ] Example files demonstrate both inline tests and properties
- [ ] Documentation clearly explains inline test syntax
- [ ] Teaching prompt includes inline test examples for AI models

## Example Files to Create/Update (CRITICAL)

**Per CLAUDE.md: Every new language feature MUST have corresponding example files!**

### Required Examples

1. **`examples/factorial.ail`** - Update with inline tests
```ailang
module examples/factorial

pure func factorial(n: int) -> int
  tests [
    (0, 1),
    (1, 1),
    (5, 120),
    (10, 3628800)
  ]
{
  if n <= 1 then 1 else n * factorial(n - 1)
}

export func main() -> () ! {IO} {
  println("Use `ailang test` to run inline tests")
}
```

2. **`examples/quicksort.ail`** - Update with properties (if quicksort exists)
```ailang
module examples/quicksort

pure func quicksort[a: Ord](list: [a]) -> [a]
  properties [
    forall(list: [int]) => length(quicksort(list)) == length(list),
    forall(list: [int]) => quicksort(quicksort(list)) == quicksort(list)
  ]
{
  -- implementation
}
```

3. **`examples/testing_basic.ail`** - Fix parse error
- Add: `module testing/basic` at top
- Add: `export func main()` at bottom

4. **`examples/testing_advanced.ail`** - Fix parse error
- Add: `module testing/advanced` at top
- Add: `export func main()` at bottom

## Dependencies

**None!** All infrastructure exists:
- ✅ Parser implementation complete
- ✅ AST nodes defined
- ✅ Generators implemented
- ✅ Shrinking algorithm working
- ✅ Reporter functional
- ✅ CLI command exists

## Open Questions

**Q1: Should inline tests execute automatically on function definition?**
- **Proposed answer**: No, only when `ailang test` is run
- **Rationale**: Keeps module loading fast, tests on demand

**Q2: Should properties run during `ailang run` or only `ailang test`?**
- **Proposed answer**: Only during `ailang test`
- **Rationale**: Property tests can be slow (100 cases each)

**Q3: Error handling for inline tests - fail at parse time or runtime?**
- **Proposed answer**: Parse successfully, fail at test execution time
- **Rationale**: Tests are metadata, shouldn't block parsing

## Implementation Notes

### Pipeline Integration Pattern

Use the same pattern as REPL evaluation:

```go
// From internal/repl/repl_eval.go as reference
func EvaluateExpression(expr ast.Expr) (Value, error) {
    // 1. Elaborate to Core
    coreProg := elaborate.Elaborate(surfaceExpr)

    // 2. Type check
    typeChecker := types.NewTypeChecker(coreProg, imports)
    typeChecker.Check()

    // 3. Evaluate
    evaluator := eval.NewEvaluator()
    result := evaluator.Eval(coreProg)

    return result, nil
}
```

### Binding Property Inputs

For `forall(x: int, y: int) => x + y == y + x`:

```go
// Generate test cases
for i := 0; i < 100; i++ {
    x := generator.GenerateInt()
    y := generator.GenerateInt()

    // Create Let bindings: let x = <generated> in let y = <generated> in (x + y == y + x)
    boundExpr := &ast.Let{
        Binder: x,
        Value: generated_x,
        Body: &ast.Let{
            Binder: y,
            Value: generated_y,
            Body: property.Expr
        }
    }

    // Evaluate through pipeline
    result := EvaluateExpression(boundExpr)

    // Check if result is true
    if !result.(BoolValue) {
        // Shrink and report counterexample
    }
}
```

## Timeline

### Day 1: Pipeline Integration (6-8 hours)
- **Morning**: Create executor, update runner stubs (4 hours)
- **Afternoon**: Property execution, unit tests (2-4 hours)
- **Output**: Tests and properties execute (not skip)

### Day 2: Examples & Integration (4-6 hours)
- **Morning**: Fix example files, add inline tests (2-3 hours)
- **Afternoon**: Integration tests, documentation (2-3 hours)
- **Output**: Complete feature with examples and docs

**Total: 10-14 hours across 1-2 days**

## Post-Sprint Tasks

After completing this sprint:
- [ ] Update CHANGELOG.md with v0.4.2 entry
- [ ] Run `make eval-baseline` to test AI code generation with inline tests
- [ ] Update teaching prompt with inline test examples
- [ ] Create release notes highlighting inline testing completion
- [ ] Archive M-TESTING design docs to `implemented/v0_4_2/`

## Related Documents

- Original design: [m-testing-property-based-testing.md](m-testing-property-based-testing.md)
- Original sprint plan: [M-TESTING-SPRINT-PLAN.md](M-TESTING-SPRINT-PLAN.md) (estimated 2 weeks, actually 90% done!)
- Pipeline integration: [m-testing-integration-complete.md](../v0_3_21/m-testing-integration-complete.md)
- Testing guide: [docs/TESTING.md](../../docs/TESTING.md)

---

**Sprint Plan Created**: 2025-11-27
**Status**: Ready for Approval
**Estimated Completion**: 1-2 days (10-14 hours)
**Risk Level**: Low (integration work, infrastructure exists)
