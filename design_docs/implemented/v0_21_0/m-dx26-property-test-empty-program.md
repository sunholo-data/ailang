# M-DX26: Property Test "Empty Program" Bug

**Status:** Planned (still unimplemented at v0.19.2; re-confirmed 2026-05-15)
**Target:** v0.21.0 (originally v0.7.2 — slipped through 12 minor releases)
**Priority:** P1 (High - blocks property-based testing AND silently makes `ensures` non-verifying)
**Estimated:** 4-6 hours (Option A) — see also **Update 2** for the deeper `result`-binding fix
**Dependencies:** None
**Created:** 2026-01-28
**Last Updated:** 2026-05-15

---

## Update 3 (2026-05-15, evening): Phase 5 Implemented in v0.21.0

Phase 5 (`ensures` result binding via dedicated harness) shipped on `dev` for v0.21.0:

- `internal/testing/collector.go` — `PropertyCase` now carries `FunctionCtx` and `Function` so the runner can resolve the function each `ensures` is attached to.
- `internal/testing/harness.go` — new `BuildEnsuresPropertyHarness` (and `EnsuresParam`) builds `LetRec(f, λ..., Let(p1, val_1, Let(p2, val_2, ..., Let("result", App(f, [p1, p2, ...]), <predicate>))))` — both `result` and each function parameter are in scope for the predicate.
- `internal/testing/executor.go` — new `EvaluateEnsuresHarness` evaluates the harness via direct Core eval, reusing the `CombinedResolver` / `injectModuleBindings` / `injectADTConstructors` plumbing already used by inline tests.
- `internal/testing/runner.go` — `runProperty` branches on `ast.EnsuresKind` to a new `runEnsuresProperty` helper. Generates one value per function parameter, runs 100 iterations, reports the first violating input as `ensures violated for input: name=value, ...`.
- `examples/runnable/contracts/ensures_violation_demo.ail` — acceptance demo (intentional `clampBuggy` reports counterexample, `clampOk` passes).
- 4 unit tests for `BuildEnsuresPropertyHarness`, 4 integration tests in `runner_ensures_test.go`. `make test` and `make lint` clean.
- Inbox `cdd55f9f` (sender: `cli`, sunholo/ailang-parse) reply pending.

**Phase 5.1 (shipped same day, follow-up):** the v1 limitation around arithmetic ops in predicates (`result == x + y` tripped `BinOp reached evaluator; dictionaries not elaborated`) is fixed. The runner now uses `EvaluateEnsuresHarnessFromCore` and pulls the *already-lowered* predicate from `result.Artifacts.Core.Meta[funcName].Contracts[i].Expr` — which has been through the same OpLowering pass that powers `--verify-contracts` (see [M-CONTRACTS-OPLOWERING](../../implemented/v0_8_0/m-contracts-oplowering.md), shipped v0.8.0). Two new tests in `runner_ensures_test.go` cover correct + buggy `add` with `result == x + y`. Implementation: cache `Core.Meta` on Executor (`lastMeta`), expose via `LastDeclMeta(funcName)`, locate the matching contract by counting ensures-position in `Function.Properties` and indexing into the kind-filtered `Contracts` slice. The AST-based `EvaluateEnsuresHarness` and `BuildEnsuresPropertyHarness` are kept as wrappers for unit tests but no production caller uses them.

**Phase 5.2 (also shipped same day):** `requires` clauses also routed through the lowered-Core path. The runner now branches on both `ast.EnsuresKind` and `ast.RequiresKind`. New `BuildRequiresPropertyHarnessFromCore` skips the function call (the predicate runs before the call, references parameters only) and skips the `result` binding. Generated inputs that violate `requires` are reported as `Skipped` (not Fail) — they're out-of-contract, not a function bug. Helper `findLoweredEnsuresPredicate` was generalized to `findLoweredContractPredicate(propCase, astKind, coreKind)` to serve both. 2 new tests in `runner_ensures_test.go`. After Phase 5.2, `examples/runnable/contracts/basic.ail` shows clean separation: requires either pass (`safeDivide_property_1` 100/100) or skip (`absolute_property_1` skipped because random ints aren't all ≥ 0); ensures fail with counterexamples for the genuinely buggy postconditions. **No more `evaluation failed: empty program` for either contract kind.**

**Still planned (not in this update):**
- **Option A (now scoped to forall only)**: Refactor `EvaluateExpression` to do direct Core evaluation for `properties [forall(...) => expr]` blocks. These are the only remaining victims of the broken source-synthesis path. Lower priority since no inbox traffic mentions forall properties.
- **Counterexample shrinking** for ensures violations.
- **Smart generators** for tighter `requires`/`ensures` coverage (random ints rarely satisfy `x >= 0` or hit interesting `if/else`-chain branches).

This doc stays in `planned/` until forall (Option A) also ships, but the door is now open to move it to `implemented/v0_21_0/` with a forall footnote if we decide forall isn't worth fixing. Sprint plan: [`design_docs/planned/v0_21_0/m-dx26-ensures-result-binding-sprint-plan.md`](../v0_21_0/m-dx26-ensures-result-binding-sprint-plan.md).

---

## Update 2 (2026-05-15): Re-confirmed on v0.19.2 + Deeper Architectural Issue Found

**Inbox**: msg `cdd55f9f-7add-40b4-9d75-13b674287d8c` from `cli` (sunholo/ailang-parse), 2026-05-15.
**Reporter** initially attributed this to a property-test generator emitting invalid `_test.ail` for `pure func ... ensures { ... } -> string`. Investigation confirms the underlying bug is **the same M-DX26**, but with two additional findings the original 2026-01-28 doc didn't capture:

### Finding 2.1 — The bug surfaces as either `empty program` OR `PAR_UNEXPECTED_TOKEN`

The original doc only documented the `evaluation failed: empty program` error path. The reporter saw `PAR_UNEXPECTED_TOKEN at _test.ail:10:25: expected next token to be ), got IDENT instead`. Both errors come from the **same root cause** — `EvaluateExpression` synthesising AILANG source via `fmt.Sprintf("%v", ast)` — but the surface error depends on whether the resulting string happens to parse to zero declarations (→ "empty program") or to a syntactically broken token stream (→ `PAR_UNEXPECTED_TOKEN`). The `_test.ail` file in the error message is the synthetic `Source.Filename` from [internal/testing/executor.go:85](../../../internal/testing/executor.go#L85), not a real file on disk.

The doc's existing diagnosis (Option A — direct Core evaluation) is still the right fix.

### Finding 2.2 — Even if source synthesis worked, `ensures` would still be unverified

The runner's `runProperty` ([internal/testing/runner.go:151](../../../internal/testing/runner.go#L151)) iterates `propCase.Property.Binders` to create generators. For `ensures`-derived properties, **`Binders` is `nil`** — see [parser_contracts.go:144](../../../internal/parser/parser_contracts.go#L144), where contract predicates are emitted with `Binders: nil` because `ensures` doesn't take a `forall` clause. This means:

1. The generator loop runs zero times.
2. `bindPropertyValues(prop, [])` returns the predicate unchanged.
3. The predicate references `result` — which is **never bound to anything** by the runner. There is no code that:
   - Generates inputs for the function's parameters,
   - Calls the function with those inputs,
   - Captures the return value,
   - Substitutes it for `result` in the predicate.

So even after Option A lands, every `ensures { result ⋯ }` clause will evaluate to "result is unbound". `ensures` clauses across the whole codebase are currently **silently non-verifying** — the runner reports failure for the wrong reason (source synthesis broke), masking the deeper fact that the property runner has no `result`-binding mechanism.

### Triangulation against reporter's narrowing

The reporter narrowed the trigger to `(pure + ensures + string return)`. Re-running their three "non-trigger" cases on the same binary (md5 `af124e1f30221cb274869cbcc414ab0a`, commit `24fd623d`) shows the same `evaluation failed: empty program` error for `int` and `bool` returns too — so the narrowing was an artefact of which functions in their codebase had the property test path actually exercised. **The bug is universal across all `ensures` clauses, regardless of return type.**

### Updated Implementation Plan

The original 4-phase plan is still correct for surfacing the failure (Option A). Add a new **Phase 5** before declaring this done:

#### Phase 5: Bind `result` in `ensures` Properties (2-3 hours, NEW)

**File: `internal/testing/runner.go` `runProperty()`**

For property cases derived from `ensures` (detectable by `propCase.Property.Kind == ast.EnsuresKind`):

1. Resolve the function the `ensures` clause is attached to (already known via `propCase.Name` — the collector emits `<funcName>_property_<i>`).
2. Build a generator for **each function parameter type** (not for `Binders`, which is empty).
3. Each test iteration:
   a. Generate parameter values.
   b. Evaluate the function call `funcName(arg1, arg2, ...)` to get the return value.
   c. Bind that return value to `result` in the predicate's environment.
   d. Evaluate the predicate.
   e. If false → the function violates its `ensures`; report counterexample (the input that produced the bad output).

This is what `ensures` is *supposed to mean* per the M-VERIFY design ([m-verify-runtime-contracts.md](../../implemented/v0_6_1/m-verify-runtime-contracts.md)): a postcondition checked against the function's actual output. Without Phase 5, `ensures` is a documentation comment, not a verifier.

`requires` clauses are simpler — they reference parameters directly, no `result` substitution needed. Still need parameter generators though.

### Reporter Workaround

Reporter (correctly) drops `ensures` and keeps inline `tests [...]` blocks. That avoids the failure but loses any contract coverage. Same recommendation stands until Phase 5 ships: prefer `tests [...]` over `ensures` if you want any verification at all.

### New Files to Modify (delta from original plan)

| File | Changes | LOC |
|------|---------|-----|
| `internal/testing/runner.go` | Add `runEnsuresProperty()` branch in `runProperty()` | ~80 |
| `internal/testing/executor.go` | Add `EvaluateFunctionCall(funcName, args, sourceFile)` helper | ~40 |
| `internal/testing/runner_ensures_test.go` | Tests for `ensures` violations & counterexamples | ~150 |
| `examples/contracts_ensures_demo.ail` | Working `ensures`-with-counterexample demo | ~30 |
| **Phase 5 subtotal** | | **~300** |
| **Combined with original ~265** | | **~565** |

### Acceptance test for Update 2

```ailang
module test/ensures_violation

export pure func clamp(x: int) -> int
  ensures { result >= 0 && result <= 10 }
{
  if x < 0 then -1     -- intentional bug
  else if x > 10 then 11
  else x
}
```

After Phase 5, `ailang test` on this module should report:

```
Properties:
  ✗ clamp_property_1 (3 cases, 12.4ms)
      ensures violated for input: -1
      result = -1, expected: result >= 0 && result <= 10
```

instead of the current `evaluation failed: empty program`.

---

## Original Doc (2026-01-28)


## Problem Statement

Property tests auto-generated from `requires`/`ensures` contracts fail with "empty program" error, blocking property-based testing functionality.

**Current behavior:**
```bash
$ ailang test examples/contracts_demo.ail

Tests:
  ✓ divide_test_1 (inline tests work)
  ✓ divide_test_2
  ✓ divide_test_3

Properties:
  ✗ divide_property_1 (1 cases, 138.666µs)
      test 0: evaluation failed: empty program
  ✗ divide_property_2 (1 cases, 142.333µs)
      test 0: evaluation failed: empty program
```

**Impact:**
- Critical - property-based testing completely broken
- Discovered during ecommerce demo (feedback message a7dd508c)
- Blocks M-VERIFY adoption (requires/ensures useless without property tests)
- User workaround: manually write inline tests instead of using contracts

## Root Cause Analysis

### The Execution Flow

1. **Property collection** (`internal/testing/collector.go:137`):
   - Extracts `forall` clauses from `requires`/`ensures`
   - Creates `PropertyCase` with AST expression

2. **Value generation** (`internal/testing/runner.go:183-188`):
   - Generates random values using type-based generators
   - Binds values to property parameters

3. **Expression evaluation** (`internal/testing/runner.go:194`):
   - Calls `r.executor.EvaluateExpression(boundExpr)`
   - **THIS IS WHERE IT FAILS**

4. **Source reconstruction** (`internal/testing/executor.go:116-147`):
   ```go
   func (e *Executor) EvaluateExpression(expr ast.Expr) (eval.Value, error) {
       var sourceParts []string

       // Reconstruct pure function sources
       for _, f := range e.sourceFile.Funcs {
           funcSrc := fmt.Sprintf("pure func %s(", f.Name)
           // ... build params ...
           funcSrc += fmt.Sprintf(") -> %v {\n", f.ReturnType)
           funcSrc += "  " + fmt.Sprintf("%v", f.Body) + "\n}\n\n"
           sourceParts = append(sourceParts, funcSrc)
       }

       // Add test expression
       sourceParts = append(sourceParts, fmt.Sprintf("%v", expr))

       // Concatenate and run through pipeline
       source := strings.Join(sourceParts, "")
       result, err := pipeline.Run(cfg, src)
   }
   ```

5. **Pipeline rejection** (`internal/pipeline/pipeline_single.go:297-300`):
   ```go
   if len(program.Items) == 0 {
       return result, fmt.Errorf("empty program: expected at least one item after parse")
   }
   ```

### The Bug

**Using `fmt.Sprintf("%v", ast.Expr)` to reconstruct source code is fundamentally broken!**

AST nodes may not have proper `String()` methods, resulting in:
- Empty strings
- Invalid AILANG syntax
- Incomplete expressions
- Missing operators or parentheses

**Evidence from code:**
- Line 135: `fmt.Sprintf(") -> %v {\n", f.ReturnType)` - may print empty
- Line 136: `fmt.Sprintf("%v", f.Body)` - may print empty or invalid syntax
- Line 142: `fmt.Sprintf("%v", expr)` - may print empty

When the reconstructed source is empty or unparseable, the pipeline sees an "empty program" and rejects it.

### Why Inline Tests Work

Inline tests use a different path (`EvaluateInlineTestsWithHarness` at runner.go:88) that:
1. Extracts Core binding directly from typed AST
2. Builds synthetic Core program with test harness
3. Evaluates Core directly (no source reconstruction)

This avoids the AST→Source conversion entirely!

## Comparison: Inline Tests vs Property Tests

| Aspect | Inline Tests (✓ Works) | Property Tests (✗ Broken) |
|--------|----------------------|--------------------------|
| Entry point | `EvaluateInlineTestsWithHarness()` | `EvaluateExpression()` |
| Input | Core binding + test cases | AST expression |
| Approach | Direct Core evaluation | AST→Source reconstruction |
| Pipeline | Bypassed (uses Core directly) | Full pipeline run |
| Source generation | Synthetic Core AST | `fmt.Sprintf("%v", ast)` |
| Result | ✓ Works | ✗ Empty program error |

## Proposed Solutions

### Option A: Direct Core Evaluation (Recommended)

**Make property tests use the same path as inline tests** - evaluate Core AST directly instead of reconstructing source.

```go
// internal/testing/executor.go - NEW METHOD
func (e *Executor) EvaluatePropertyExpression(expr ast.Expr) (eval.Value, error) {
    // 1. Type check the expression
    typeChecker := types.NewTypeChecker(e.coreProg, e.imports)
    typ, err := typeChecker.InferExpr(expr)
    if err != nil {
        return nil, fmt.Errorf("type inference failed: %w", err)
    }

    // 2. Elaborate to Core
    elaborator := elaborate.NewElaborator()
    coreExpr, err := elaborator.ElaborateExpr(expr)
    if err != nil {
        return nil, fmt.Errorf("elaboration failed: %w", err)
    }

    // 3. Evaluate Core expression directly (no source reconstruction)
    effCtx := effects.NewMockEffContext()
    evaluator := eval.NewEvaluator(e.coreProg, effCtx)
    return evaluator.EvalExpr(coreExpr)
}
```

**Pros:**
- Avoids AST→Source conversion entirely
- Consistent with inline test path
- Simpler and more reliable
- No dependency on AST String() methods

**Cons:**
- Requires adding `InferExpr()`, `ElaborateExpr()`, `EvalExpr()` entry points
- More work than fixing String() methods

**Estimated effort:** 4-6 hours

### Option B: Implement Proper AST→Source Unparser

Add a proper unparser that converts AST back to valid AILANG source code.

```go
// internal/unparse/unparse.go - NEW PACKAGE
func Unparse(node ast.Node) string {
    var buf strings.Builder
    unparseLet(&buf, node.(*ast.Let), 0)
    return buf.String()
}

func unparseLet(buf *strings.Builder, let *ast.Let, indent int) {
    buf.WriteString("let ")
    buf.WriteString(let.Name)
    buf.WriteString(" = ")
    unparseExpr(buf, let.Value, indent)
    buf.WriteString(" in\n")
    unparseExpr(buf, let.Body, indent)
}

// ... unparse functions for all AST node types
```

**Pros:**
- Useful for other features (code formatters, refactoring tools)
- Preserves current architecture

**Cons:**
- Large effort (need unparse for ALL AST node types)
- Fragile (must update when AST changes)
- Still doing unnecessary roundtrip (AST→Source→Parse→Core→Eval)

**Estimated effort:** 12-16 hours

### Option C: Fix AST String() Methods

Implement proper `String()` methods on all AST node types.

**Pros:**
- Minimal code changes
- Useful for debugging

**Cons:**
- Same fragility as Option B (must maintain for all node types)
- Still doing unnecessary AST→Source→AST roundtrip
- Doesn't address fundamental architectural issue

**Estimated effort:** 8-10 hours

## Recommendation

**Option A (Direct Core Evaluation)** - eliminates the root cause by avoiding source reconstruction entirely. This is the same approach that made inline tests work reliably.

## Implementation Plan

### Phase 1: Add Expression Entry Points (2 hours)

**File: `internal/types/checker.go`**
- [ ] Add `InferExpr(expr ast.Expr) (types.Type, error)` method
- [ ] Type check expression in isolation (empty initial environment)
- [ ] Tests: literals, binops, lambdas, let bindings

**File: `internal/elaborate/elaborate.go`**
- [ ] Add `ElaborateExpr(expr ast.Expr) (core.Expr, error)` method
- [ ] Convert Surface AST expression to Core
- [ ] Tests: all expression types

**File: `internal/eval/eval.go`**
- [ ] Add `EvalExpr(expr core.Expr) (Value, error)` method
- [ ] Evaluate Core expression directly
- [ ] Tests: arithmetic, closures, conditionals

### Phase 2: Refactor Property Test Execution (1.5 hours)

**File: `internal/testing/executor.go`**
- [ ] Add `EvaluatePropertyExpression(expr ast.Expr)` method
- [ ] Use new entry points (InferExpr → ElaborateExpr → EvalExpr)
- [ ] Include function definitions from source file in Core environment
- [ ] Remove old `EvaluateExpression()` or mark deprecated

**File: `internal/testing/runner.go`**
- [ ] Update `runProperty()` to call `EvaluatePropertyExpression()`
- [ ] No changes to property collection or value generation

### Phase 3: Testing & Validation (1.5 hours)

- [ ] Test with ecommerce demo contracts
- [ ] Test with builtin contracts from std/
- [ ] Verify property tests pass for all supported types (int, float, bool, string, lists)
- [ ] Update documentation in `docs/docs/guides/testing.md`

### Phase 4: Cleanup (1 hour)

- [ ] Remove or deprecate old `EvaluateExpression()` method
- [ ] Add warnings if AST String() methods produce empty results
- [ ] Update CHANGELOG.md
- [ ] Close related GitHub issues

## Test Cases

```ailang
module examples/property_test_demo

-- Test case 1: Simple arithmetic property
pure func add(x: int, y: int) -> int
  requires forall(x: int, y: int) => add(x, y) == add(y, x)  -- Commutativity
  requires forall(x: int) => add(x, 0) == x                   -- Identity
{
  x + y
}

-- Test case 2: Division with contracts
pure func divide(x: int, y: int) -> int
  requires y != 0
  ensures forall(x: int, y: int) => y != 0 ==> x == divide(x, y) * y + (x % y)
{
  x / y
}

-- Test case 3: List length property
pure func listLength(xs: [int]) -> int
  requires forall(xs: [int]) => listLength(xs) >= 0
  ensures forall(xs: [int], x: int) => listLength([x] ++ xs) == 1 + listLength(xs)
{
  match xs {
    [] => 0,
    [_, ...rest] => 1 + listLength(rest)
  }
}
```

**Expected output after fix:**
```bash
$ ailang test examples/property_test_demo.ail

Tests:
  (no inline tests)

Properties:
  ✓ add_property_1 (100 cases, 45.2ms)  # Commutativity
  ✓ add_property_2 (100 cases, 38.1ms)  # Identity
  ✓ divide_property_1 (100 cases, 52.7ms)
  ✓ listLength_property_1 (100 cases, 61.3ms)
  ✓ listLength_property_2 (100 cases, 58.9ms)

Summary: 5 properties passed, 0 failed (500 test cases, 256.2ms)
```

## Success Criteria

- [ ] Property tests evaluate successfully without "empty program" error
- [ ] All ecommerce demo contracts produce passing property tests
- [ ] Test execution time comparable to inline tests (<100ms for 100 cases)
- [ ] No AST→Source reconstruction in property test path
- [ ] Documentation updated in testing guide

## Files to Modify

| File | Changes | LOC |
|------|---------|-----|
| `internal/types/checker.go` | Add `InferExpr()` entry point | ~40 |
| `internal/elaborate/elaborate.go` | Add `ElaborateExpr()` entry point | ~30 |
| `internal/eval/eval.go` | Add `EvalExpr()` entry point | ~25 |
| `internal/testing/executor.go` | Add `EvaluatePropertyExpression()` | ~50 |
| `internal/testing/runner.go` | Update `runProperty()` | ~10 |
| `internal/testing/executor_test.go` | Tests for new methods | ~80 |
| `docs/docs/guides/testing.md` | Update property test docs | ~20 |
| `CHANGELOG.md` | Document fix | ~10 |
| **Total** | | ~265 |

## Related Issues

- **Message ID**: a7dd508c-a2a6-412e-81c5-85c7f5996f13 (demos/ecommerce)
- **User report**: "Property tests fail because AILANG's test framework tries to load _test.ail internally but that module doesn't exist"
  - **NOTE**: This was partially correct - the _test.ail issue was fixed in executor.go:158 by setting IsREPL: true
  - **However**: A deeper issue remains (this bug) - property test source reconstruction produces empty programs

## Related Documents

- [M-VERIFY Implementation](../../implemented/v0_7_0/m-verify-requires-ensures.md) - Contracts feature
- [M-TESTING Integration](../../archive/v0_3_21_m-testing-integration-complete.md) - Test framework
- [Testing Guide](../../../docs/docs/guides/testing.md) - Property test documentation

## Alternatives Considered

### Alternative 1: Skip Property Tests for Now

Mark property tests as "not implemented" and skip them during test execution.

**Pros:**
- Zero implementation effort
- Unblocks users who can use inline tests instead

**Cons:**
- Defeats purpose of M-VERIFY (contracts need property tests)
- Poor user experience (contracts silently ignored)
- Technical debt

**Rejected:** Property tests are core to M-VERIFY value proposition.

### Alternative 2: Use Python/Go Property Test Libraries

Generate property tests as external Python (Hypothesis) or Go (gopter) code.

**Pros:**
- Leverage mature property testing libraries
- More powerful generators/shrinking

**Cons:**
- Cross-language complexity
- Loses AILANG-native testing story
- Hard to integrate with `ailang test` command

**Rejected:** We want AILANG-native property testing.

## Notes

- This bug was discovered during ecommerce demo session
- Inline test fix (IsREPL: true) was applied in same session but didn't fix property tests
- Property tests use fundamentally different execution path than inline tests
- The issue is NOT with module loading - it's with AST→Source reconstruction
- Similar issues may exist elsewhere in codebase (check for other uses of `fmt.Sprintf("%v", ast)`)

## Migration Notes

**For users:**
- No migration needed - property tests will "just work" after fix
- Existing contracts will automatically generate working property tests

**For developers:**
- If you have custom test executors, update to use new `EvaluatePropertyExpression()` method
- Old `EvaluateExpression()` will be deprecated but not removed in v0.7.2
