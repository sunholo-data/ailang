# M-DX26 Phase 5: `ensures` Result Binding — Sprint Plan

**Sprint ID**: M-DX26-P5
**Target Version**: v0.21.0
**Estimated Duration**: 0.5 days (~4 hours)
**Estimated LOC**: ~300 (180 prod + 120 tests)
**Risk Level**: low
**Created**: 2026-05-15

## Sprint Goal

Make `ensures` clauses actually verify by binding the special variable `result` to the function's real return value (computed against generated parameter inputs) and evaluating the postcondition predicate. Today every `ensures` clause silently fails to verify because the runner has no `result`-binding mechanism — the predicate evaluates with `result` unbound, surfacing as `evaluation failed: empty program` or `PAR_UNEXPECTED_TOKEN`.

This sprint implements **Phase 5 only** of [M-DX26](../v0_13_0/m-dx26-property-test-empty-program.md). The broader Option A refactor (replacing `EvaluateExpression` source synthesis with direct Core evaluation for `forall`-style properties) is **explicitly out of scope** and remains in the parent doc for a future sprint.

## Background & Recent Velocity

- 7-day commit window: 22 commits, dominated by M-STDLIB-XML-WALK-PERF (4 milestones, ~3 days) and M-STDLIB-URL-ENCODE (1 day, ~150 LOC).
- Comparable scope: M-STDLIB-URL-ENCODE shipped in <1 day. M-STDLIB-XML-WALK-PERF M1 (foldChildren/getAttrMap/nodeKind) ~150 LOC took half a day.
- Realistic pace for this sprint: half a day for the focused 3-file change with reused infrastructure.

## Out of Scope (Tracked in Parent Doc)

- ❌ **Option A**: Refactoring `EvaluateExpression` to use direct Core evaluation for `forall`-style properties. Forall properties remain on the existing broken path until a future sprint.
- ❌ **Smart generators**: Targeted string/int generators for tighter `ensures` coverage. v1 ships with the existing random `createGeneratorForType` infrastructure; weak coverage is acceptable for now.
- ❌ **`requires` clauses**: Same architectural fix would apply (no `result` to bind, just predicate over args), but adds scope. Tracked as a follow-up; ensures is the louder pain point per inbox traffic.
- ❌ **Counterexample shrinking**: Use existing `shrinkCounterexample` plumbing as-is.

## Milestones

### M1: Collector + Harness Builder (~2 hours, ~150 LOC)

Wire the data so the runner knows which function each `ensures` predicate is attached to, and build the Core harness that calls the function and binds `result`.

**Files to modify:**

| File | Change | LOC |
|------|--------|-----|
| `internal/testing/collector.go` | Add `FunctionCtx string` and `Function *ast.FuncDecl` to `PropertyCase`; populate in `collectInlineTests` | ~15 |
| `internal/testing/harness.go` | New `BuildEnsuresPropertyHarness(binding, paramValues, predicate) core.CoreExpr` | ~80 |
| `internal/testing/harness_test.go` | Unit tests: single-arg, multi-arg, predicate referencing `result`, predicate ignoring `result` | ~80 |

**Harness shape** (single test for clarity; runner will build one harness per generated input):

```
LetRec(f, λparams. body,
  Let("result", App(f, [arg_1, ..., arg_n]),
    <predicate-AST converted via astExprToCore>
  )
)
```

**Acceptance criteria:**
- [ ] `PropertyCase.FunctionCtx` populated for every inline property emitted from `decl.Properties` (which includes ensures).
- [ ] `BuildEnsuresPropertyHarness` returns a Core expression that evaluates to a `BoolValue`.
- [ ] Harness unit tests pass for: single-arg int→int, multi-arg (int, int)→int, string-returning func with predicate `result == "x"`, predicate that doesn't reference `result`.
- [ ] Harness unit tests use the same `astExprToCore` converter as `BuildInlineTestHarness`.
- [ ] No regression in existing `harness_test.go` cases.

**Risks:**
- Naming collision if user code also has a binding named `result` in scope. Low risk: `result` is a documented reserved name in `ensures` predicates.

### M2: Runner Branch + Acceptance Test (~2 hours, ~150 LOC)

Route `EnsuresKind` properties through the new harness, generate parameter values per function param type, report counterexamples on violation. Ship an example.

**Files to modify:**

| File | Change | LOC |
|------|--------|-----|
| `internal/testing/runner.go` | In `runProperty`: branch on `propCase.Property.Kind == EnsuresKind`; new helper `runEnsuresProperty` that generates param values, builds + evaluates harness per iteration, reports counterexample | ~100 |
| `internal/testing/runner_ensures_test.go` | New file: integration tests covering `clamp` violation, multi-arg ensures, string-returning ensures | ~80 |
| `examples/contracts_ensures_demo.ail` | Working `ailang test` demo with intentional violation | ~25 |
| `CHANGELOG.md` | "Fixed" entry under v0.21.0 referencing M-DX26 Phase 5 | ~5 |

**Algorithm** (`runEnsuresProperty`):

1. Resolve the function via `executor.ExtractFunctionBinding(propCase.FunctionCtx, sourceFile)`.
2. Build generators from **`decl.Params[i].Type`** (NOT from `Property.Binders` — empty for ensures).
3. For each of `numTests` iterations:
   a. Generate one value per parameter via existing `createGeneratorForType`.
   b. Convert generated values to Core literals via existing `valueToLiteral` + `astExprToCore`.
   c. Build harness via `BuildEnsuresPropertyHarness`.
   d. Evaluate via `evaluator.EvalCoreProgram` (same plumbing as `EvaluateInlineTestsWithHarness` — reuse `CombinedResolver`, `injectModuleBindings`, `injectADTConstructors`).
   e. If result is `BoolValue{false}`: report counterexample `(args, computed_result)`; stop.
4. If all 100 iterations pass: `Status = Pass`.
5. If any param type has no generator: `Status = Skip` with reason.

**Acceptance criteria:**
- [ ] `examples/contracts_ensures_demo.ail` (the `clamp` example from the design doc) reports `ensures violated for input: -1` (or another negative input) instead of `evaluation failed: empty program`.
- [ ] `examples/contracts_ensures_demo.ail` runs cleanly via `ailang test` exit code != 0 on violation.
- [ ] A correctly-implemented `clamp` (no bug) reports `Status: Pass`.
- [ ] Multi-arg ensures (e.g. `add(x: int, y: int) -> int ensures { result == x + y }`) verifies correctly.
- [ ] String-returning ensures (e.g. `tag(x: int) -> string ensures { result == "neg" || result == "pos" || result == "zero" }`) at least runs to completion (may pass trivially with random ints since the ensures is exhaustive).
- [ ] `forall`-style properties (`Property.Kind == PropertyKind`) continue to use the existing path — explicit `if` branch in `runProperty`.
- [ ] `make test` passes; `make ci` clean.
- [ ] CHANGELOG entry references both inbox msg `cdd55f9f` and the parent design doc.

**Risks:**
- `ExtractFunctionBinding` may have unexpected behavior on functions with effects in their body. Pure-only restriction is enforced by `stripNonPureFunctions` already; ensures is documented for `pure func` only.
- Random parameter generators may rarely hit the interesting branches in `if/else`-chain function bodies, making `ensures` "trivially pass" for most random inputs. Acceptable for v1 — better generators are a follow-up.

## Implementation Order

1. **Phase 0 (15 min)**: Write failing acceptance test first — create `examples/contracts_ensures_demo.ail` with the buggy `clamp`. Run `ailang test` and capture the current failure (`evaluation failed: empty program`). This proves the bug and gives us the target.
2. **M1 (2h)**: Collector field + harness builder + harness unit tests. Run `make test` after each sub-step.
3. **M2 (1.5h)**: Runner branch + integration test + acceptance test passes.
4. **Wrap (30 min)**: CHANGELOG, ack `cli` reporter, move M-DX26 doc to `implemented/v0_21_0/` with Phase 5 marked done (Option A still planned).

## Success Metrics

- [ ] `examples/contracts_ensures_demo.ail` working both ways (intentional violation reports counterexample; correct impl passes).
- [ ] All new tests in `harness_test.go` and `runner_ensures_test.go` pass.
- [ ] No regression in `internal/testing/...` test suite.
- [ ] `make ci` clean.
- [ ] Inbox reply sent to `cli` (msg `cdd55f9f`) confirming the fix and recommending they restore `ensures` clauses on the `sunholo/ailang-parse` codebase.
- [ ] CHANGELOG.md updated under v0.21.0 "Fixed".

## Dependencies

- None. All required infrastructure already exists in `internal/testing/`:
  - `ExtractFunctionBinding` ([executor.go:146](../../../internal/testing/executor.go#L146))
  - `astExprToCore` ([harness.go:155](../../../internal/testing/harness.go#L155))
  - `createGeneratorForType` ([runner.go:248](../../../internal/testing/runner.go#L248))
  - `valueToLiteral` ([runner.go:309](../../../internal/testing/runner.go#L309))
  - `CombinedResolver`, `injectModuleBindings`, `injectADTConstructors` (executor.go)

## Open Questions (resolved before sprint start)

- ✅ **What about `requires`?** Defer to follow-up sprint — same shape but no `result` binding. Out of scope.
- ✅ **Random vs targeted generators?** Random for v1; targeted is a follow-up.
- ✅ **Should ensures violations exit the test suite non-zero?** Yes — they're test failures. `runProperty` already returns `Status = Fail` which feeds into `SuiteResult` aggregation.

## Related

- **Parent doc**: [M-DX26: Property Test Empty Program Bug](../v0_13_0/m-dx26-property-test-empty-program.md) — see Update 2 (2026-05-15) for the full architectural finding.
- **Inbox message**: `cdd55f9f-7add-40b4-9d75-13b674287d8c` from `cli` (sunholo/ailang-parse).
- **M-VERIFY (v0.6.1)**: [m-verify-runtime-contracts.md](../../implemented/v0_6_1/m-verify-runtime-contracts.md) — the original contracts feature this sprint completes.
