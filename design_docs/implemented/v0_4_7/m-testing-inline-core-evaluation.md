# Core Program Evaluation for Inline Tests

**Status**: Planned
**Target**: v0.4.7
**Priority**: P1 - Blocking inline test feature (M-TESTING-INLINE)
**Estimated**: 2-3 days
**Dependencies**: None (M-TESTING-INLINE parser/AST complete, this unblocks execution)

## AI-First Alignment Check

**Score this feature against AILANG's core principles:**

| Principle | Impact | Score | Notes |
|-----------|--------|-------|-------|
| Reduce Syntactic Noise | Neutral | 0 | Internal compiler fix, no user-facing syntax change |
| Preserve Semantic Clarity | Positive | +1 | Makes inline test execution semantically correct |
| Increase Determinism | Positive | +1 | Enables deterministic test execution |
| Lower Token Cost | Neutral | 0 | No change to user-facing token costs |
| **Net Score** | | **+2** | **Decision: Move forward** |

**Decision rule:** Net score > +1 → Move forward | ≤ 0 → Reject or redesign

**Reference:** See [AI-first DX philosophy](../v0_3_15/example-parity-vision-alignment.md#-design-principle-ai-first-dx)

**Rationale:** This is an **internal compiler correctness fix** required to enable inline test execution. While it doesn't directly impact user-facing syntax, it's critical for the M-TESTING-INLINE feature to work, which itself provides significant AI-first DX benefits (inline tests reduce boilerplate and token costs for test code).

## Problem Statement

**The inline testing feature (M-TESTING-INLINE) cannot evaluate test expressions that reference functions defined in the same module due to Core AST scoping issues.**

When executing inline tests like:
```ailang
pure func factorial(n: int) -> int
  tests [(0, 1), (5, 120)]
{ ... }
```

The test executor needs to evaluate `factorial(0)` and `factorial(5)`, but Core LetRec bindings only scope to their body, not to subsequent declarations.

**Current State:**
- Parser, AST, collector, and generators are 100% complete (~4,892 LOC)
- Test expressions are properly collected and constructed
- **BLOCKER**: Cannot execute tests that call the function being tested
- Error: `undefined variable: factorial` when evaluating test expressions

**Root Cause:**
The Core AST elaborates module-level declarations as:
```
Program.Decls = [
  LetRec(factorial, lambda, body),  // Decl[0]
  factorial(0)                       // Decl[1] - factorial is undefined!
]
```

LetRec bindings in Core only scope to their **body**, not to subsequent array elements. Evaluating `Decl[1]` fails because `factorial` is not in scope.

**Impact:**
- **Blocks M-TESTING-INLINE completion** (95% done, last 5% blocked)
- **All inline tests fail at runtime** despite parsing correctly
- **Integration tests cannot verify end-to-end flow**
- Cannot ship inline testing feature to users

## Goals

**Primary Goal:** Enable evaluation of Core programs with multiple declarations where later declarations reference earlier ones.

**Success Metrics:**
- Inline tests execute successfully (e.g., `factorial(0)` returns `1`)
- `ailang test examples/factorial.ail` shows 4/4 tests passing
- All M-TESTING-INLINE integration tests pass
- No regression in existing module execution

## Solution Design

### Overview

We need a mechanism to evaluate Core programs where declarations reference each other. Three potential approaches:

1. **Nested LetRec Wrapping** - Elaborate to nested LetRecs instead of flat array
2. **Module-Level Environment** - Build persistent environment across declarations
3. **Synthetic Module Evaluation** - Use ModuleRuntime for proper scoping

### Approach 1: Per-Function Test Harness Transformation (RECOMMENDED)

**Problem:** Current elaboration creates flat array where bindings don't persist:
```
Decls = [LetRec(f, ...), App(f, ...)]  // f not in scope for App!
```

**Solution:** Build a synthetic Core expression per tested function (test-only, doesn't affect normal compilation):

```
TestHarness(f) :=
  LetRec("f", λ_f,
    Let("_test_1", App(f, arg_1),
      Let("_test_2", App(f, arg_2),
        Tuple([_test_1, _test_2])  // Returns actual results
      )
    )
  )
```

**Key semantic points:**
1. **Test-only transformation**: Normal compilation/execution is unchanged; this is purely for test evaluation
2. **Returns actuals, not pass/fail**: Harness evaluates to tuple of actual results; Go compares to expected values
3. **Type of harness body**: `Tuple([_test_1, ...])` has type `(τ₁, τ₂, ...)`, not the function's return type; this is fine
4. **Scoping**: Function `f` is in scope for all test calls via LetRec body

**Advantages:**
- ✅ Uses existing Core semantics (no new evaluation logic)
- ✅ Minimal blast radius (no whole-file elaboration changes)
- ✅ Easy to test in isolation
- ✅ Preserves all type information
- ✅ Clear separation: normal vs. test evaluation paths

**Architecture:**

**Component 1: Test Harness Builder** (NEW)
Add `internal/testing/harness.go` - builds synthetic Core expressions:

```go
// BuildInlineTestHarness creates a Core expression for evaluating inline tests
// Input: Core LetRec binding for function f, list of test specifications
// Output: Core expression that evaluates all tests and returns tuple of actuals
func BuildInlineTestHarness(
    binding *core.LetRecBinding,
    tests []TestSpec,
) core.Expr
```

**Component 2: Test Executor Update**
Update `internal/testing/executor.go`:
- Extract LetRec binding from elaborated module
- Call `BuildInlineTestHarness()` to construct test expression
- Evaluate harness expression in empty environment
- Compare returned tuple to expected values

**Implementation Complexity:** Low (~4-6 hours)
- NEW: `internal/testing/harness.go` (~80 LOC) - transformer
- UPDATE: `internal/testing/executor.go` (~30 LOC) - use transformer
- NO CHANGES to elaborator or evaluator
- All existing tests should still pass

### Approach 2: Test-Only Environment Accumulation (VIABLE FALLBACK)

**Problem:** Simple harness transformation (Approach 1) may not handle cross-definition references (helpers, constants).

**Solution:** Test-executor-only environment accumulation:

```go
// Test executor only - NOT exposed as general API
func evalDeclsWithEnv(prog *core.Program) (*eval.Env, error) {
    env := eval.NewEnvironment()
    evaluator := eval.NewCoreEvaluatorWithEnv(env)
    for _, decl := range prog.Decls {
        _, err := evaluator.Eval(decl) // env accumulates bindings
        if err != nil { return nil, err }
    }
    return env, nil
}

// Then evaluate test harness in that env:
testEnv, _ := evalDeclsWithEnv(module)
result, _ := evaluator.EvalInEnv(testHarness, testEnv)
```

**Advantages:**
- ✅ Handles cross-definition references (helpers, constants)
- ✅ No CoreEvaluator API changes (test-local helper)
- ✅ Can combine with Approach 1 harness transform

**Disadvantages:**
- ❌ Requires small test-executor helper (~50 LOC)
- ❌ Slightly more complex than pure transformation

**Use case:** If testing reveals that functions under test reference other top-level bindings (helpers, constants) and simple harness transformation doesn't handle them, use this as fallback.

**Implementation Complexity:** Low-Medium (~6-8 hours)
- NEW: `internal/testing/envaccum.go` (~50 LOC) - test-only helper
- UPDATE: `internal/testing/executor.go` (~40 LOC) - use accumulated env
- NO CHANGES to evaluator public API
- Scoped to test executor only

### Approach 3: ModuleRuntime Integration

**Problem:** Executor tries to manually run pipeline phases.

**Solution:** Use ModuleRuntime for proper module-level execution:
```go
runtime := NewModuleRuntime(...)
result := runtime.ExecuteModule(syntheticModule)
```

**Advantages:**
- ✅ Uses production execution path
- ✅ Handles all scoping correctly

**Disadvantages:**
- ❌ ModuleRuntime expects real module files
- ❌ Requires filesystem for module loading
- ❌ Heavy dependency for simple test execution
- ❌ Hard to pass synthetic AST

**Implementation Complexity:** High (~16-20 hours)
- Deep ModuleRuntime refactoring
- Filesystem abstraction layer
- Risk of breaking module system

### Recommended Approach: Test Harness Transformation (Approach 1)

**Rationale:**
- Lowest risk and complexity (~4-6 hours)
- Uses existing Core semantics correctly
- Zero changes to elaborator/evaluator (isolated to test code)
- Easy to test and verify in isolation
- Clear conceptual model: "per-function test harness"
- Approach 2 available as fallback if cross-definition scoping issues arise

**Decision:** Start with Approach 1 (simple harness transform). If testing reveals issues with functions that reference helpers/constants, add Approach 2's env accumulation as a compatibility layer.

### Implementation Plan

**Phase 1: Test Harness Builder** (~3-4 hours)
- [ ] Create `internal/testing/harness.go`
- [ ] Implement `BuildInlineTestHarness(binding, tests)` function
  - Takes Core LetRec binding + test specs
  - Returns synthetic Core expression with nested Lets
  - Handles single and multiple test cases
- [ ] Unit test harness building with simple examples
  - Single test case
  - Multiple test cases
  - Multi-argument functions

**Phase 2: Executor Integration** (~2-3 hours)
- [ ] Update `internal/testing/executor.go`
  - Extract LetRec binding from elaborated module
  - Call `BuildInlineTestHarness()` for each tested function
  - Evaluate harness expression
  - Extract tuple results and compare to expected values
- [ ] Integration test with factorial example
- [ ] **Verify cross-definition scoping**: Test functions that reference helpers/constants

**Phase 3: Testing & Validation** (~2-3 hours)
- [ ] Verify all M-TESTING-INLINE integration tests pass
- [ ] Test with multiple inline tests per function
- [ ] Test with recursive functions (factorial, fibonacci)
- [ ] **Test cross-module references** (helper functions, constants)
- [ ] Ensure no regression in normal module execution (run full test suite)
- [ ] Update documentation with implementation notes

**Contingency: If Phase 2 reveals cross-definition issues:**
- [ ] Add `internal/testing/envaccum.go` (Approach 2)
- [ ] Modify executor to build accumulated environment
- [ ] Evaluate harness in that environment
- [ ] Add integration tests for helper function references

### Files to Modify/Create

**New files:**
- `internal/testing/harness.go` - Test harness builder (~80 LOC)
  - `BuildInlineTestHarness(binding, tests)` - Core transformer

**Modified files:**
- `internal/testing/executor.go` - Use harness builder (~30 LOC changed)
- `internal/testing/harness_test.go` - Unit tests for builder (~120 LOC)
- `internal/testing/executor_test.go` - Integration tests (~50 LOC added)

**Contingency (if needed):**
- `internal/testing/envaccum.go` - Environment accumulator (~50 LOC)

**Total estimated changes:** ~280 LOC (base) or ~330 LOC (with contingency)

## Examples

### Example 1: Simple Inline Test

**Current (Broken):**
```ailang
module test/factorial

pure func factorial(n: int) -> int
  tests [(0, 1), (5, 120)]
{ if n <= 1 then 1 else n * factorial(n - 1) }
```

**Core AST (Current - Broken):**
```
Program.Decls = [
  LetRec("factorial", lambda(n, if ...), Unit),  // Decl[0]
  App(Var("factorial"), Lit(0)),                  // Decl[1] - ERROR!
  App(Var("factorial"), Lit(5))                   // Decl[2] - ERROR!
]
```

**Core AST (Proposed - Fixed):**
```
Program.Decls = [
  LetRec("factorial", lambda(n, if ...),
    Let("_test_1", App(Var("factorial"), Lit(0)),
      Let("_test_2", App(Var("factorial"), Lit(5)),
        Tuple([Var("_test_1"), Var("_test_2")])  // Return test results
      )
    )
  )
]
```

**Runtime behavior:**
1. LetRec binds `factorial` to lambda
2. Body evaluates: `_test_1 = factorial(0)` ✅ (factorial in scope)
3. Then: `_test_2 = factorial(5)` ✅ (factorial still in scope)
4. Returns tuple of results for test runner to verify

### Example 2: Multiple Tests

**Input:**
```ailang
pure func add(x: int, y: int) -> int
  tests [(1, 2, 3), (0, 0, 0), (5, 3, 8)]
{ x + y }
```

**Nested Core (Simplified):**
```
LetRec("add", lambda([x, y], BinOp(x, +, y)),
  Let("_test_1", App(App(Var("add"), 1), 2),
    Let("_test_2", App(App(Var("add"), 0), 0),
      Let("_test_3", App(App(Var("add"), 5), 3),
        Tuple([_test_1, _test_2, _test_3])
      )
    )
  )
)
```

## Success Criteria

- [ ] `ailang test examples/factorial.ail` shows 4/4 tests passing
- [ ] Nested LetRec elaboration produces valid Core AST
- [ ] Test expressions can reference the function being tested
- [ ] Multiple tests per function work correctly
- [ ] Recursive functions work in inline tests
- [ ] All existing 110+ tests still pass (no regression)
- [ ] M-TESTING-INLINE integration tests pass
- [ ] Documentation updated with implementation notes

## Testing Strategy

**Unit tests:**
- Elaborator nesting logic (6-8 tests)
  - Single test case
  - Multiple test cases
  - Nested Let structure validation
  - Type preservation through nesting

**Integration tests:**
- `examples/factorial.ail` with 4 inline tests
- Multi-argument functions (`add(x, y)`)
- Recursive functions
- Property-based tests (no changes needed)

**Manual testing:**
- `ailang test examples/factorial.ail`
- `ailang test examples/testing_basic.ail` (if applicable)
- Verify test output format unchanged

## Non-Goals

**Not in this feature:**
- **Effect handling in tests** - Tests remain pure only (deferred to future)
- **Property test execution** - Different code path, already works
- **REPL test execution** - Module files only for now
- **Performance optimization** - Correctness first, optimize later

## Timeline

**Day 1** (6-8 hours):
- Phase 1: Elaborator nesting implementation
- Unit tests for nesting logic

**Day 2** (4-6 hours):
- Phase 2: Executor integration
- Integration tests with factorial example

**Day 3** (3-4 hours):
- Phase 3: Testing & validation
- Documentation updates
- Verify no regressions

**Total: ~13-18 hours across 2-3 days**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| **Cross-definition scoping**: Functions under test that reference helpers/constants may fail if those bindings aren't in harness scope | High | **Phase 2 explicit verification**: Test functions that reference other top-level definitions. If fails, implement Approach 2 (env accumulation) as compatibility layer. |
| Nested harness breaks type checking | Medium | Harness returns `Tuple([actuals])`, not function type; this is expected. Validate with type checker in tests. |
| Performance degradation | Low | Harness evaluation is only for tests, not production code path. |
| Complex multi-test nesting | Low | Harness builder handles this mechanically (fold over tests to build nested Lets). |
| Breaks existing test infrastructure | Medium | No changes to collector/runner; only executor changes. Run full test suite to verify. |

## Alternative Approaches Tried

**What we attempted during development:**

### 1. Manual Pipeline Evaluation ❌
**Approach:** Manually run elaborate → typecheck → evaluate phases
**Problem:** CoreTypeInfo validation failures, complex phase orchestration
**Why failed:** Missing proper environment threading between phases

### 2. Source Reconstruction + `pipeline.Run` ❌
**Approach:** Reconstruct source text from AST, run through pipeline
**Problem:** Module loader tries to find file on filesystem
**Why failed:** Synthetic modules can't satisfy filesystem requirements

### 3. Separate Declaration Evaluation ❌
**Approach:** Evaluate each Core declaration independently
**Problem:** `factorial` binding from Decl[0] not available in Decl[1]
**Why failed:** LetRec scoping doesn't persist across array elements

**Lesson learned:** Core AST semantics require proper nesting, not flat arrays, for cross-declaration references.

## References

- **M-TESTING-INLINE Sprint**: `.ailang/state/sprints/sprint_M-TESTING-INLINE.json`
- **Parser Implementation**: `internal/parser/parser_test.go` (test parsing)
- **AST Structures**: `internal/ast/ast_decl.go` (FuncDecl.Tests)
- **Current Executor**: `internal/testing/executor.go` (220 LOC, needs fixing)
- **Elaborator**: `internal/elaborate/elaborator.go` (needs nesting support)
- **Core AST**: `internal/core/core.go` (LetRec semantics)
- **Similar Issue**: This is the same blocker from previous M-TESTING attempt

## Future Work

**Features that build on this:**
- **Effect-ful test execution** - Allow tests with IO, FS effects
- **REPL test execution** - Run tests in interactive mode
- **Incremental test caching** - Cache test results for faster re-runs
- **Test coverage tracking** - Track which code paths are tested
- **Parallel test execution** - Run tests concurrently (deterministic)

---

**Document created**: 2025-11-27
**Last updated**: 2025-11-27
**Blocking**: M-TESTING-INLINE sprint completion (95% → 100%)
