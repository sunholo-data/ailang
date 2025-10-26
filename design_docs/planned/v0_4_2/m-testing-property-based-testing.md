# M-TESTING: Property-Based Testing & Test Syntax

**Status**: Planned
**Target**: v0.4.2
**Priority**: P1 (Medium - High value for AI code generation)
**Estimated**: 2 weeks (~60 hours)
**Dependencies**: v0.4.0 deterministic tooling complete

## AI-First Alignment Check

**Score this feature against AILANG's core principles:**

| Principle | Impact | Score | Notes |
|-----------|--------|-------|-------|
| Reduce Syntactic Noise | + | +1 | Inline tests reduce separate test file boilerplate |
| Preserve Semantic Clarity | + | +1 | Tests are explicit, properties formalize invariants |
| Increase Determinism | + | +1 | Property-based tests find edge cases systematically |
| Lower Token Cost | 0 | 0 | Neutral - adds syntax but improves AI confidence |
| **Net Score** | | **+3** | **Decision: Move forward** |

**Decision rule:** Net score > +1 → Move forward | ≤ 0 → Reject or redesign

**Reference:** See [AI-first DX philosophy](../v0_3_15/example-parity-vision-alignment.md#-design-principle-ai-first-dx)

## Problem Statement

**Current State:**
- Examples use complex manual testing patterns (`factorial.ail`, `quicksort.ail`)
- No formal way to specify function properties/invariants
- AI-generated code lacks confidence verification
- Tests are separate from implementation (reduces AI context efficiency)

**Impact:**
- **Example files blocked**: 4+ example files cannot run (factorial, quicksort, concurrent_pipeline)
- **AI code quality**: AI-generated AILANG code cannot include inline tests/properties
- **Documentation value**: Examples would be more useful with embedded tests
- **Confidence**: No systematic way to verify correctness properties

## Goals

**Primary Goal:** Enable property-based testing and inline test syntax to improve AI code generation confidence and example quality

**Success Metrics:**
- All blocked example files (factorial.ail, quicksort.ail) work correctly
- AI models can generate code with inline tests (measured via M-EVAL)
- 50+ property-based tests across stdlib and examples
- Property test framework integrated with `ailang test` command

## Solution Design

### Overview

Add three complementary testing features:

1. **Inline Tests**: Attach test cases directly to functions
2. **Property Declarations**: Formal invariant specifications
3. **Test Blocks**: Standalone test suites with assertions

All integrated with deterministic test execution framework.

### Architecture

**Components:**

1. **Syntax Extensions** (`internal/parser/parser.go`, `internal/ast/ast.go`)
   - `tests [...]` clause for function inline tests
   - `properties [...]` clause for invariant declarations
   - `test "name" { ... }` blocks for test suites
   - `property "name" { ... }` blocks for property declarations
   - `assert` keyword for test assertions
   - `forall` quantifier for property-based testing

2. **Test Execution Engine** (`internal/testing/` - NEW)
   - Collect tests from AST
   - Execute inline tests with function
   - Generate test cases for properties (QuickCheck-style)
   - Report results in structured format

3. **CLI Integration** (`cmd/ailang/cmd_test.go` - NEW)
   - `ailang test file.ail` - Run all tests in file
   - `ailang test --watch` - Watch mode for TDD
   - `ailang test --property-tests=100` - Control property test count
   - `ailang test --seed=42` - Deterministic test generation

4. **Property-Based Test Generator** (`internal/testing/generator.go` - NEW)
   - Generate random test cases for `forall` properties
   - Type-aware generators (int, string, list, ADT, etc.)
   - Shrinking on failure (minimal failing case)

### Syntax Design

**1. Inline Tests**

```ailang
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
```

- Syntax: `tests [(input, expected), ...]`
- Executed automatically on function definition (verification)
- Part of function metadata (not runtime cost)
- Can be disabled with `--skip-tests` flag

**2. Property Declarations**

```ailang
pure func quicksort[a: Ord](list: [a]) -> [a]
  properties [
    forall(list: [int]) => length(quicksort(list)) == length(list),
    forall(list: [int]) => quicksort(quicksort(list)) == quicksort(list)
  ]
{
  -- implementation
}
```

- Syntax: `properties [forall(...) => predicate, ...]`
- Checked with generated test cases (default 100 per property)
- Failures report minimal failing input (shrinking)

**3. Test Blocks**

```ailang
test "factorial implementations equivalent" {
  assert factorial(0) == 1;
  assert factorial(5) == 120;

  forall(n: int) where n >= 0 && n <= 20 =>
    factorial(n) == factorialTail(n)
}
```

- Standalone test suite
- Multiple assertions per test
- Property-based tests with `forall`
- Conditional generation with `where` clause

**4. Property Blocks**

```ailang
property "sort is idempotent" {
  forall(list: [int]) =>
    quicksort(quicksort(list)) == quicksort(list)
}

property "sort preserves length" {
  forall(list: [int]) =>
    length(quicksort(list)) == length(list)
}
```

- Named properties for documentation
- Can be referenced in other tests
- Executed during `ailang test`

### Implementation Plan

**Phase 1: Syntax & Parsing** (~20 hours)
- [ ] Add `tests`, `properties`, `test`, `property`, `assert`, `forall` keywords
- [ ] Extend AST with test/property nodes
- [ ] Parse inline tests clause
- [ ] Parse property declarations
- [ ] Parse test/property blocks
- [ ] Unit tests for parser

**Phase 2: Test Execution Engine** (~25 hours)
- [ ] Create `internal/testing/` package
- [ ] Test collector (extract from AST)
- [ ] Test runner (execute test cases)
- [ ] Assertion framework (`assert` implementation)
- [ ] Test result reporting (JSON + human-readable)
- [ ] Integration with module loader
- [ ] Unit tests for execution engine

**Phase 3: Property-Based Testing** (~10 hours)
- [ ] Type-aware generators (`internal/testing/generator.go`)
- [ ] Random test case generation
- [ ] Shrinking algorithm (minimize failing input)
- [ ] Deterministic seed support
- [ ] Property test runner
- [ ] Unit tests for generators

**Phase 4: CLI & Integration** (~5 hours)
- [ ] `ailang test` command implementation
- [ ] `--watch`, `--property-tests`, `--seed` flags
- [ ] Exit codes (0 = pass, 1 = fail)
- [ ] CI integration examples
- [ ] Documentation

### Files to Modify/Create

**New files:**
- `internal/testing/testing.go` - Test framework (~400 LOC)
- `internal/testing/generator.go` - Property-based test generators (~300 LOC)
- `internal/testing/shrink.go` - Input shrinking (~200 LOC)
- `internal/testing/collector.go` - Test collection from AST (~150 LOC)
- `internal/testing/runner.go` - Test execution (~250 LOC)
- `cmd/ailang/cmd_test.go` - CLI command (~150 LOC)
- `internal/testing/testing_test.go` - Unit tests (~500 LOC)

**Modified files:**
- `internal/lexer/token.go` - Add test keywords (~10 LOC)
- `internal/parser/parser.go` - Parse test syntax (~300 LOC)
- `internal/ast/ast.go` - Add test AST nodes (~100 LOC)
- `internal/elaborate/elaborate.go` - Handle test nodes (~50 LOC)
- `cmd/ailang/main.go` - Register test command (~5 LOC)

**Total new code: ~2,410 LOC**

## Examples

### Example 1: Inline Tests

**Before (v0.3.x - doesn't work):**
```ailang
-- factorial.ail with separate test file
pure func factorial(n: int) -> int {
  if n <= 1 then 1 else n * factorial(n - 1)
}

-- tests/factorial_test.ail (separate file)
-- Manual testing, no automation
```

**After (v0.4.2):**
```ailang
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

-- Tests run automatically: ailang test factorial.ail
-- ✓ factorial tests passed (4/4)
```

### Example 2: Property-Based Testing

**Before:**
```ailang
-- No way to express "quicksort is idempotent"
-- Manual testing with hardcoded lists
```

**After:**
```ailang
pure func quicksort[a: Ord](list: [a]) -> [a]
  properties [
    -- Idempotence
    forall(list: [int]) =>
      quicksort(quicksort(list)) == quicksort(list),

    -- Length preservation
    forall(list: [int]) =>
      length(quicksort(list)) == length(list),

    -- Sorted property
    forall(list: [int]) =>
      isSorted(quicksort(list))
  ]
{
  match list {
    [] => [],
    [x] => [x],
    [pivot, ...rest] => {
      let less = filter(\x. x < pivot, rest);
      let greater = filter(\x. x >= pivot, rest);
      quicksort(less) ++ [pivot] ++ quicksort(greater)
    }
  }
}

-- ailang test quicksort.ail
-- ✓ quicksort properties passed (300 test cases)
--   - Idempotence: 100/100 passed
--   - Length preservation: 100/100 passed
--   - Sorted property: 100/100 passed
```

### Example 3: Test Blocks

```ailang
test "factorial implementations equivalent" {
  -- Direct assertions
  assert factorial(0) == 1;
  assert factorial(5) == 120;

  -- Property-based test in block
  forall(n: int) where n >= 0 && n <= 20 =>
    factorial(n) == factorialTail(n) &&
    factorial(n) == factorialMatch(n)
}

-- ailang test factorial.ail
-- ✓ factorial implementations equivalent (20 test cases)
```

## Success Criteria

- [ ] Inline tests work (4 example files updated: factorial.ail, quicksort.ail, etc.)
- [ ] Property-based testing generates 100+ test cases per property
- [ ] Shrinking finds minimal failing inputs (verified with failing test)
- [ ] `ailang test` command runs all tests in file/module
- [ ] Test results output JSON (machine-readable) and human-readable formats
- [ ] All 28 runnable examples have inline tests added
- [ ] M-EVAL benchmarks show AI models can generate code with tests
- [ ] All tests passing (100+ new tests for framework)
- [ ] Documentation updated (CLAUDE.md, teaching prompt)
- [ ] Examples added (factorial, quicksort working with tests)

## Testing Strategy

**Unit tests:**
- Parser correctly handles `tests`, `properties`, `test`, `property`, `assert`, `forall` syntax
- Test collector extracts all tests from AST
- Generators produce valid values for all AILANG types
- Shrinking algorithm finds minimal failing input
- Test runner correctly reports pass/fail

**Integration tests:**
- `ailang test examples/factorial.ail` runs inline tests
- Property-based tests generate and check 100 cases
- Test blocks execute assertions
- Failed tests report useful error messages
- `--seed` flag produces deterministic results

**Manual testing:**
- Update factorial.ail with inline tests, verify it works
- Update quicksort.ail with properties, verify all pass
- Intentionally break a property, verify shrinking works
- Run full test suite on stdlib

## Non-Goals

**Not in this feature:**
- **Mutation testing** - Deferred to v0.5.0+
- **Coverage analysis** - Deferred to v0.5.0+
- **Benchmark testing** - Deferred to v0.5.0+ (use M-EVAL for now)
- **Mocking framework** - Use `MockEffContext` for effects (exists)
- **Snapshot testing** - Out of scope
- **Parallel test execution** - Sequential for now (determinism)

## Timeline

**Week 1** (30 hours):
- Phase 1: Syntax & Parsing (20 hours)
- Phase 2: Test Execution Engine (start, 10 hours)

**Week 2** (30 hours):
- Phase 2: Test Execution Engine (complete, 15 hours)
- Phase 3: Property-Based Testing (10 hours)
- Phase 4: CLI & Integration (5 hours)

**Total: ~60 hours across 2 weeks**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| **Parser complexity** | High | Incremental approach - inline tests first, properties later |
| **Generator quality** | Medium | Use Hedgehog/QuickCheck as reference, test with known properties |
| **Shrinking performance** | Low | Limit shrinking iterations (default 100), make configurable |
| **Test execution time** | Medium | Make property test count configurable (`--property-tests=N`) |
| **AI model adoption** | Medium | Update teaching prompt v0.4.2 with test syntax examples |

## References

- **Property-based testing libraries**:
  - Haskell QuickCheck: https://hackage.haskell.org/package/QuickCheck
  - Hedgehog: https://hedgehog.qa/
  - Hypothesis (Python): https://hypothesis.readthedocs.io/

- **Example files requiring this feature**:
  - `examples/experimental/factorial.ail`
  - `examples/experimental/quicksort.ail`
  - `examples/experimental/concurrent_pipeline.ail`

- **Related design docs**:
  - [v0.4-roadmap.md](../v0.4-roadmap.md) - Overall v0.4 plan
  - [M-EVAL](../../implemented/v0_3_0/m_eval_ai_benchmarking.md) - AI code generation benchmarks

## Future Work

**v0.5.0+:**
- Mutation testing (verify tests detect bugs)
- Coverage analysis (which lines executed by tests)
- Benchmark framework (performance testing)
- Hypothesis-style stateful property testing
- Custom generators for user-defined ADTs
- Parallel test execution (deterministic scheduling)

---

**Document created**: 2025-10-26
**Last updated**: 2025-10-26
