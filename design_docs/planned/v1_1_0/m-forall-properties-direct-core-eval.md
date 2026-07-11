# M-FORALL-PROPERTIES: Direct-Core Evaluation for `properties [forall(...) => ...]` Blocks

**Status**: Planned
**Target Version**: v1.0.0 (no urgency — defer until a real user asks)
**Priority**: P3 (Low — zero production users, zero inbox traffic, ergonomic alternative exists)
**Estimated Effort**: 3–4 hours, ~250 LOC
**Dependencies**: None (the surrounding infrastructure shipped with M-DX26 Phase 5/5.1/5.2)
**Created**: 2026-05-15
**Discovered**: 2026-05-15 — surfaced as the only remaining victim of the broken `EvaluateExpression` source-synthesis path after [M-DX26](../v0_13_0/m-dx26-property-test-empty-program.md) Phase 5/5.1/5.2 closed `ensures` and `requires`.

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No semantic change. Property tests already gated on a fixed RNG seed. |
| A2: Replayability | 0 | No trace surface change. |
| A3: Effect Legibility | 0 | Properties are over pure code. |
| A4: Explicit Authority | 0 | No capability change. |
| A5: Bounded Verification | +1 | Brings the third contract kind (forall) up to the same verification quality as requires/ensures (which now actually work). |
| A6: Safe Concurrency | 0 | No concurrency change. |
| A7: Machines First | 0 | Improves the experience for anyone who eventually writes forall properties — but there are zero such users today, so the immediate machine-facing impact is zero. |
| A8: Minimal Syntax | 0 | No syntax change — the syntax already exists, only its evaluation is broken. |
| A9: Cost Visibility | 0 | No cost model change. |
| A10: Composability | +1 | Property predicates already work for requires/ensures via lowered Core; this completes the set. |
| A11: Structured Failure | +1 | Eliminates the misleading `evaluation failed: empty program` for forall properties (the same error class as M-DX26 Phase 5). |
| A12: System Boundary | 0 | No FFI / no boundary change. |

**Net Score: +3** → **Decision: Move forward — but only when a real user asks.**

### Hard Violation Check

- [x] A1 (Determinism): no nondeterminism introduced
- [x] A3 (Effects): no effect change
- [x] A4 (Authority): no ambient access
- [x] A7 (Machines First): doesn't actively harm machines (just no immediate uplift)

## Problem Statement

After M-DX26 Phase 5.2 (commit `3ebf60b1`), the AILANG test runner correctly verifies both `requires` and `ensures` clauses by pulling already-lowered Core predicates from `result.Artifacts.Core.Meta[funcName].Contracts` and binding generated parameter values. The third inline-contract construct — **`properties [forall(x: T) => predicate, ...]`** — is **still broken** and emits `evaluation failed: empty program` for the same root cause that M-DX26 originally diagnosed.

**Reproduce on `dev` HEAD (post-Phase 5.2):**

```ailang
module repro
export pure func factorial(n: int) -> int {
  if n <= 0 then 1 else n * factorial(n - 1)
}
properties [
  forall(n: int) => factorial(n) >= 0
]
```

```
$ ailang test repro.ail
Properties:
  ✗ factorial_property_1 (1 cases, 624µs)
      test 0: evaluation failed: empty program
```

The runner's `runProperty` for `Property.Kind == ast.PropertyKind` (the forall case) still uses the broken `EvaluateExpression` source-synthesis path — see [internal/testing/runner.go:175–195](../../../internal/testing/runner.go#L175-L195). The same `fmt.Sprintf("%v", ast)` reconstruction that broke ensures/requires breaks forall.

## Why This Is Lower Priority Than Phase 5.2

Three signals:

1. **Zero production users.** Grep reveals exactly one usage of `properties [...]` in the codebase: [`examples/experimental/factorial.ail`](../../../examples/experimental/factorial.ail). Its README explicitly says: *"Status: Core algorithms work, but `tests [...]` and `properties [...]` syntax not implemented."*
2. **Zero inbox traffic.** Every other phase of M-DX26 was driven by an inbox report (`cdd55f9f` from `cli@sunholo/ailang-parse` for ensures; `basic.ail` violations for requires). Forall has no such report.
3. **Ergonomic alternative now works.** Anyone who would have written `forall(x: int) => f(x) >= 0` can now write `pure func f(x: int) -> int ensures { result >= 0 }` and get the same coverage with cleaner syntax — and it actually verifies as of v0.21.0.

This doc exists so the issue is recorded and not silently lost; it does **not** say the work is urgent.

## Why It's More Complex Than Phase 5.1/5.2

Three structural differences from `requires`/`ensures` that prevented us from shipping forall in the same session:

### Difference 1: The predicate isn't in `Meta.Contracts`

The elaborator's [`elaborateContracts`](../../../internal/elaborate/file_funcs.go#L73-L122) explicitly skips forall:

```go
// Only process requires/ensures contracts (skip forall properties)
if prop.Kind != ast.RequiresKind && prop.Kind != ast.EnsuresKind && prop.Kind != ast.InvariantKind {
    continue
}
```

So `result.Artifacts.Core.Meta[funcName].Contracts` contains zero forall predicates. The Phase 5.1 trick — pull the already-lowered Core predicate out and use it directly — does not apply.

### Difference 2: Quantifier variables, not function parameters

For `forall(x: int, y: int) => x + y == y + x`, the binders `x`, `y` are **new names introduced by the property**, not the surrounding function's parameters. The elaborator would need to introduce these into scope before lowering the predicate body. Slightly more work than requires/ensures (which inherit the function's own params).

### Difference 3: No `Meta.Properties` slot exists

Adding storage for lowered forall predicates means a new `[]*core.Property` (or a `Kind`-tagged unified slot) on `core.DeclMeta`, plus serializer updates if `gob.Register` cares.

## Goals

**Primary Goal:** `properties [forall(x: T) => predicate]` blocks evaluate correctly and report counterexamples on violation, just like `ensures` does today.

**Success Metrics:**

- The factorial example above (`forall(n: int) => factorial(n) >= 0`) reports `Status: Pass` with 100 iterations.
- A buggy variant (`forall(n: int) => n + 1 < n`) reports `property failed on input: n=N` as a counterexample.
- No regression in `requires`/`ensures` (M-DX26 Phase 5/5.1/5.2 still passes).
- No regression in the existing `internal/testing/integration_test.go` tests that already exercise the forall path (currently passing because they use only comparison ops on bare quantifier vars and the source synthesis happens to round-trip for trivial bodies).

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Lower forall in elaborator vs. lower at runtime in test harness | Elaborator path is cleaner (single source of truth for lowering) but touches new code paths and binder scoping. Runtime path keeps the elaborator unchanged but needs `OpLowerer` + `CoreTI` + binder env wired into the test runner. | human | design | med (elab) / high (runtime) |
| New `Meta.Properties` slot vs. extend `Contracts` with a new kind | Extending Contracts is smaller but conflates "runtime checked" (requires/ensures) with "test-only" (forall). New slot is cleaner. | human | design | low |
| Whether to also lower invariants the same way | Invariants are already lowered today (see `elaborateContracts`). Forall is the only outlier. Don't bundle. | agent | design | n/a |
| Whether to keep the broken `EvaluateExpression` path at all after this | Once forall is on direct-Core eval, no test-runner caller uses `EvaluateExpression`. It can be deleted, simplifying executor.go. | human | implementation | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [ ] **Pick the elaborator path** (preferred): extend `elaborateContracts` to also handle `PropertyKind`, introduce binders into scope, lower the predicate body, store under a new `DeclMeta.Properties []*core.PropertySpec` slot containing both `Binders []core.PropertyBinder` and `Expr core.CoreExpr`.
- [ ] **OR pick the runtime path** (fallback if elaborator changes are too invasive): in `runProperty`, when Kind == PropertyKind, build a synthetic `core.Lambda` wrapping the predicate body with the forall binders as parameters, run `OpLowerer.Lower` on it, then evaluate with generated values bound. Requires plumbing `OpLowerer` and `CoreTI` from the cached pipeline result into the test executor.
- [ ] **Lock the test corpus before changing code**: the existing `examples/experimental/factorial.ail` plus 3–4 new fixtures (multi-binder forall, forall referencing a function call, forall with arithmetic in the predicate, forall that fails to expose counterexample reporting).
- [ ] **Decide deletion of `EvaluateExpression`**: once forall is on direct-Core eval, can `internal/testing/executor.go:EvaluateExpression` and the source-synthesis machinery be deleted? Audit other callers first.

## Solution Design

### Overview (elaborator path, recommended)

1. **Extend `elaborateContracts`** ([file_funcs.go:73](../../../internal/elaborate/file_funcs.go#L73)) to handle `PropertyKind`:
   - Introduce each `Binder.Name : Binder.Type` into the elaboration env.
   - Lower the predicate body with binders in scope.
   - Emit a separate slice (or a kind-tagged entry on the existing `Contracts` slice).

2. **Add `DeclMeta.Properties []*core.PropertySpec`** to `core.DeclMeta` ([core.go:420](../../../internal/core/core.go#L420)). `PropertySpec` carries `{Binders []PropertyBinder, Expr core.CoreExpr, Location string}`.

3. **OpLowering** ([op_lowering.go:89–111](../../../internal/pipeline/op_lowering.go#L89-L111)): extend the `Meta.Contracts` walk to also walk `Meta.Properties`.

4. **Runner branch** in `runProperty` ([runner.go:158](../../../internal/testing/runner.go#L158)): mirror Phase 5.2's pattern — extract function binding (just to populate `lastMeta`), find the matching lowered `PropertySpec` by counting forall-position in `Function.Properties`, generate values per `PropertySpec.Binders[i].Type`, build a harness `Let("x", val_1, Let("y", val_2, <predicate-Core>))` (no function call, no `result`), evaluate.

5. **New harness builder** `BuildForallPropertyHarnessFromCore(binders []core.PropertyBinder, generatedValues []core.CoreExpr, predicateCore core.CoreExpr) core.CoreExpr` — essentially identical to `BuildRequiresPropertyHarnessFromCore` but parameterised on binder names from the `PropertySpec` rather than function `Params`.

6. **Counterexample reporting**: on `BoolValue{false}`, report `property failed on input: name=value, ...` — same format as ensures violations.

### Architecture

**Components:**

| File | Change | LOC |
|------|--------|-----|
| `internal/core/core.go` | New `PropertySpec`/`PropertyBinder` types; `DeclMeta.Properties` field | ~30 |
| `internal/elaborate/file_funcs.go` | Extend `elaborateContracts` to handle `PropertyKind` with binder scope | ~50 |
| `internal/elaborate/file.go` | Wire `Properties` into `DeclMeta` construction (mirrors Contracts) | ~10 |
| `internal/pipeline/op_lowering.go` | Walk `Meta.Properties` alongside `Meta.Contracts` | ~20 |
| `internal/testing/harness.go` | `BuildForallPropertyHarnessFromCore` | ~50 |
| `internal/testing/executor.go` | `EvaluateForallHarnessFromCore` | ~10 |
| `internal/testing/runner.go` | New `runForallProperty`; switch case in `runProperty`; generalise `findLoweredContractPredicate` to also handle Properties | ~80 |
| `internal/testing/runner_forall_test.go` | New: 4–5 integration tests | ~120 |
| `examples/experimental/factorial.ail` → `examples/runnable/contracts/forall_demo.ail` | Promote out of experimental, fix the README | ~40 |
| **Total** | | **~410** |

(Elaborator approach. Runtime approach lands closer to ~300 LOC but with more wiring in the test executor.)

### Out of Scope

- **`forall` over non-int types in v1**: the existing `createGeneratorForType` ([runner.go:248](../../../internal/testing/runner.go#L248)) supports int/float/bool/string/list. Lift exactly that surface; user-defined ADTs and records are out of scope until someone asks.
- **Nested foralls** (`forall(x) => forall(y) => p(x,y)`): grammar permits it, semantics are weird, no test corpus. Defer.
- **Counterexample shrinking**: same plumbing already exists for the broken path (`shrinkCounterexample`); reuse if cheap, defer if not.
- **Deletion of `EvaluateExpression`**: separate cleanup commit — only valid once *no* runner callers remain. Audit and propose in a follow-up.

## Implementation Plan (when picked up)

### Phase 1: Lock Tests First (45 min)

- [ ] Move the experimental factorial example to `examples/runnable/contracts/forall_demo.ail` with a working forall over `factorial(n) >= 0`.
- [ ] Add a buggy variant to verify counterexample reporting.
- [ ] Add `internal/testing/runner_forall_test.go` with 4 failing tests (one per success metric).

### Phase 2: Elaborator + Core (1.5h)

- [ ] Add `core.PropertySpec` / `core.PropertyBinder` and `DeclMeta.Properties`.
- [ ] Extend `elaborateContracts` to lower forall predicates with binder scope.
- [ ] Extend `OpLowerer.Lower` to walk `Meta.Properties`.
- [ ] Verify by inspecting `result.Artifacts.Core.Meta` on the new fixture.

### Phase 3: Runner + Harness (1h)

- [ ] Add `BuildForallPropertyHarnessFromCore`.
- [ ] Add `EvaluateForallHarnessFromCore` on Executor.
- [ ] Add `runForallProperty` in runner.
- [ ] Generalise `findLoweredContractPredicate` to also serve forall (or add a sister `findLoweredPropertyPredicate`).

### Phase 4: Validate + Docs (45 min)

- [ ] All 4 new tests pass.
- [ ] No regression in `requires`/`ensures` integration tests.
- [ ] CHANGELOG entry.
- [ ] Move parent doc M-DX26 to `implemented/v0_21_0/` (or to whatever version this lands in) since forall was its last outstanding item.
- [ ] Optionally: audit and delete `EvaluateExpression` if no callers remain.

## Success Criteria

- [ ] `examples/runnable/contracts/forall_demo.ail` runs cleanly and reports Pass for the correct case, counterexample for the buggy case.
- [ ] All 4 new integration tests in `runner_forall_test.go` pass.
- [ ] No regression in M-DX26 Phase 5/5.1/5.2 acceptance tests.
- [ ] `make test` + `make lint` clean.
- [ ] CHANGELOG entry under the target version.
- [ ] M-DX26 parent doc moves to `implemented/` (forall was the last outstanding item).

## Related

- **Parent doc**: [M-DX26: Property Test Empty Program Bug](../v0_13_0/m-dx26-property-test-empty-program.md) — see Updates 2 + 3 for the architectural backstory and the Phase 5/5.1/5.2 work that handles the requires/ensures siblings.
- **Sibling sprint**: [M-DX26-P5 Sprint Plan](../v0_21_0/m-dx26-ensures-result-binding-sprint-plan.md) — the 4-milestone sprint that shipped on 2026-05-15.
- **Foundational pass**: [M-CONTRACTS-OPLOWERING (v0.8.0)](../../implemented/v0_8_0/m-contracts-oplowering.md) — the OpLowering pass over `Meta.Contracts` we'd extend to also walk `Meta.Properties`.

## Alternatives Considered

### Alternative 1: Do nothing — leave forall properties broken

**Pros:** Zero engineering cost.
**Cons:** Misleading error message (`evaluation failed: empty program`) the next time a user accidentally writes a forall property thinking it works. Documentation says it's "not implemented" but the syntax parses without complaint.

**Rejected — but only weakly.** This doc exists so the failure mode is recorded; if no user files a real bug in 12 months, consider permanently removing the syntax (see Alternative 4).

### Alternative 2: Runtime lowering in the test runner

Lower the forall predicate AST → Core inside the test runner using the existing `OpLowerer`, plumbed via `CoreTI` from the cached pipeline result.

**Pros:** Smaller diff (no elaborator changes).
**Cons:** Test runner becomes a second site that knows about operator lowering. Requires `CoreTI` plumbing through the test executor that doesn't exist today.

**Deferred to fallback** if the elaborator path turns out invasive.

### Alternative 3: Generator-driven harness without operator lowering

Build the harness with raw `core.BinOp` like Phase 5 v0 did. Force users to write forall predicates that only use comparison/boolean ops (no arithmetic).

**Pros:** Fastest possible patch.
**Cons:** Carries the same "arithmetic doesn't work" footgun that Phase 5.1 had to fix in the same week. Will produce another inbox report.

**Rejected.**

### Alternative 4: Remove `properties [...]` syntax entirely

Document `properties [forall(...) => ...]` as deprecated, route the parser to a deprecation warning, eventually remove.

**Pros:** Removes a misleading affordance. `ensures` covers the common case.
**Cons:** Breaking change to a documented (if non-functional) language feature. Requires a deprecation cycle.

**Worth revisiting** if no implementation work happens within ~2 versions.

## Notes

- This was deferred at the end of M-DX26 Phase 5.2 because: (a) zero production users, (b) ~3× the engineering cost of 5.1/5.2 due to elaborator changes, (c) the user-facing alternative (`ensures`) works.
- The doc is filed under `v1_0_0/` to signal "no urgency, no fixed version commitment". Move to a concrete version when an inbox report or user request lands.
- If the experimental factorial example is the only existing forall in the codebase and the README already says "not implemented", consider whether **Alternative 4 (remove the syntax)** is the better long-term answer. That's a separate design conversation.
