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

### Approach 1: Nested LetRec Elaboration (RECOMMENDED)

**Problem:** Current elaboration creates flat array:
```
Decls = [LetRec(f, ...), App(f, ...)]
```

**Solution:** Nest subsequent declarations in LetRec body:
```
Decls = [
  LetRec(f, lambda,
    Let(_result, App(f, arg),  // Nested test expression
      Var(_result)              // Return test result
    )
  )
]
```

**Advantages:**
- ✅ Uses existing Core semantics (no new evaluation logic)
- ✅ Works with current pipeline phases
- ✅ Minimal changes to executor
- ✅ Preserves all type information

**Architecture:**

**Component 1: Elaborator Enhancement**
Modify `internal/elaborate/elaborator.go` to support "synthetic module mode":
- When elaborating test expressions, wrap in nested Let instead of appending to Decls[]
- Function LetRec body becomes: `Let(_test_expr, test_call, body)`
- Preserves function body but adds test evaluation

**Component 2: Test Executor Update**
Update `internal/testing/executor.go` to request nested elaboration:
- Pass flag/mode to elaborator: `ElaborateWithNesting()`
- Extract result from nested structure

**Implementation Complexity:** Low (~4-6 hours)
- Modify 1 file significantly: `elaborator.go`
- Update 1 file minimally: `executor.go`
- All existing tests should still pass

### Approach 2: Module-Level Environment

**Problem:** CoreEvaluator evaluates each declaration in fresh environment.

**Solution:** Maintain persistent environment across Eval() calls:
```go
env := eval.NewEnvironment()
evaluator := eval.NewCoreEvaluatorWithEnv(env)
for _, decl := range prog.Decls {
    value, err := evaluator.Eval(decl) // env persists
}
```

**Advantages:**
- ✅ No elaborator changes
- ✅ Straightforward implementation

**Disadvantages:**
- ❌ Requires CoreEvaluator API changes
- ❌ May affect REPL and other consumers
- ❌ LetRec semantics unclear with persistent env

**Implementation Complexity:** Medium (~8-12 hours)
- Modify `internal/eval/eval_evaluator.go`
- Update all CoreEvaluator call sites
- Extensive testing needed

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

### Recommended Approach: Nested LetRec (Approach 1)

**Rationale:**
- Lowest risk and complexity
- Uses existing Core semantics correctly
- Minimal API surface changes
- Easy to test and verify

### Implementation Plan

**Phase 1: Elaborator Nesting Support** (~3-4 hours)
- [ ] Add `ElaborateFileWithTestEval()` method to elaborator
- [ ] Modify function declaration elaboration to nest test expressions
- [ ] Create synthetic Let bindings for test results
- [ ] Unit test nested elaboration with simple examples

**Phase 2: Executor Integration** (~2-3 hours)
- [ ] Update Executor to use new elaboration mode
- [ ] Extract test result from nested structure
- [ ] Handle multiple tests per function (nested Lets)
- [ ] Integration test with factorial example

**Phase 3: Testing & Validation** (~2-3 hours)
- [ ] Verify all M-TESTING-INLINE integration tests pass
- [ ] Test with multiple inline tests per function
- [ ] Test with recursive functions
- [ ] Ensure no regression in normal module execution
- [ ] Update documentation

### Files to Modify/Create

**Modified files:**
- `internal/elaborate/elaborator.go` - Add nesting mode (~60 LOC added)
- `internal/testing/executor.go` - Use nesting mode (~20 LOC changed)
- `internal/testing/executor_test.go` - Add tests (~100 LOC added)

**No new files needed.**

**Total estimated changes:** ~180 LOC

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
| Nested elaboration breaks type checking | High | Run full type checker on nested AST, validate CoreTypeInfo |
| Performance degradation | Low | Nesting is only for test execution, not production code |
| Breaks existing elaboration | High | Feature flag + extensive regression testing |
| Complex multi-test nesting | Medium | Start with single test, iterate to multiple |

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
