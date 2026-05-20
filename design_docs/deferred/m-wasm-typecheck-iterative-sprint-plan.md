# Sprint Plan: M-WASM-TYPECHECK-ITERATIVE

**Sprint ID**: M-WASM-TYPECHECK-ITERATIVE
**Design doc**: [m-wasm-typecheck-iterative.md](m-wasm-typecheck-iterative.md)
**Target version**: v1.1.0
**Risk**: **HIGH** — touches the type-checker hot path; regressions break every AILANG program
**Duration**: 7 working days (~1700 LOC across implementation + tests)
**Acceptance gate**: `node /Users/mark/dev/sunholo/demos/scripts/wasm-loadmodule-harness.js` exits 0 with the full original `cognitive_commons/services/citizen.ail` (currently stubbed in sister repo). Time-to-load < 1s.

## Summary

Convert recursive descent in the type-checker hot paths to iterative + explicit work-stack so WASM-compiled AILANG can load realistically-sized modules without overflowing the JS host call stack (~10–15K frames). Native Go grows goroutine stacks; WASM does not. CLI test suite is necessary but insufficient — every milestone gates on **both** `make test` AND the headless WASM harness in the sister demos repo.

## Background

Reproducer caught 2026-05-20: `citizen.ail` (7170 bytes, 11 imports, triple-nested `match`, three back-to-back matches on `Result[JudgeScore, string]` with `s.x`/`s.y` field access) throws "Maximum call stack size exceeded" 80–120 s into WASM type-check. CLI handles it in 18 ms. M-TYPECHECK-NO-AUTO-UNWRAP-RESULT (commit 9bda96a6, May 2026) added the cycle-safe tagged-union predicate analyses that pushed citizen.ail past the cliff. Full diagnostic trail: [demos/debug-notes/wasm-citizen-stack-overflow.md](../../../../demos/debug-notes/wasm-citizen-stack-overflow.md).

## Velocity Analysis

| Source | Data |
|---|---|
| Net LOC churn last 30 days in `internal/types/` | +3,281 / -190 = **+3,090 net** |
| Predecessor sprint (M-TYPECHECK-NO-AUTO-UNWRAP-RESULT M1, May 2026) | 703 LOC over 2 estimated days (350 LOC/day) |
| Recent comparable sprints (M-SCHEME-IMPORT-PRESERVE-ADT-HEAD, M-MATCH-ADT-XCHECK) | 200–400 LOC/day when actively coding |
| Conservative blended | **250 LOC/day** (30% buffer for HIGH risk) |
| `internal/types/` test suite runtime | 0.339s — fast iteration loop |

## Scope Adjustments from Design Doc

The design doc assumed 3 files (`typechecker_core.go`, `tagged_union_predicate.go`, `unification.go`). Actual scope:

| Design doc said | Reality |
|---|---|
| `internal/types/typechecker_core.go` (1 file) | `typechecker_core.go` (696 lines) + `typechecker_functions.go` (17.3 KB) + `inference.go` (14 KB) |
| `internal/types/unification.go` | `unification.go` is a stub (11 lines); real unifier is in `unification_core.go`, `unification_records.go`, `row_unification.go` |
| `internal/types/scheme.go` | Does not exist — scheme code is in `types_v2.go` + `inference.go` |

LOC and milestone estimates have been adjusted to reflect the wider footprint.

## Milestone Breakdown

### M1 — Iterative AST traversal (3 days, ~650 LOC)

**Risk**: highest. This is the keystone change. Wrap behind feature flag.

**Files modified**:
- `internal/types/typechecker_core.go` — refactor `checkExpr` dispatcher to use work-stack
- `internal/types/typechecker_functions.go` — function-body traversal (uses checkExpr)
- `internal/types/inference.go` — top-level inference loop calls into checkExpr
- `internal/types/iter_work_stack.go` *(new ~120 LOC)* — explicit work-stack with deferred-arm support
- `internal/types/typechecker_iterative_test.go` *(new ~200 LOC)* — regression tests for nested-match, deep let-binding, deeply-nested record field access

**Feature flag**: `AILANG_TYPECHECK_ITERATIVE=1` (env var). When unset, falls back to existing recursive descent so we can ship M1 disabled-by-default and toggle on after M2/M3 land.

**Acceptance criteria**:
1. Synthetic test fixture: 1000-deep nested `match` type-checks cleanly with the flag ON.
2. Same fixture passes WASM smoke (built + harness exit 0) under the flag.
3. `make test` passes unchanged with flag OFF (no behavior change).
4. `make test` passes with flag ON (parity).
5. No measurable native-CLI regression (`internal/types/` benchmarks, see M2).
6. Lint clean; file-size discipline (no file > 800 lines after refactor).

**Risks**:
- Type-checker is conditional — checking a `match` arm depends on knowing the scrutinee's type FIRST. The work-stack needs a "deferred" entry kind so arms can be revisited after their scrutinee constraint resolves.
- Subtle ordering bugs where recursive descent's natural traversal order matters (e.g. shadowing in nested lets).
- **Mitigation**: feature flag means we can land it dark, run the WASM harness + a tier-1 eval suite under both modes, and only flip the default after a clean week.

---

### M2 — Memoize `isTaggedUnion` per (type, ctorSet) (1 day, ~200 LOC)

**Risk**: medium. Cycle-safe memoization is subtle but small surface area.

**Files modified**:
- `internal/types/tagged_union_predicate.go` — add memo cache keyed on `(type.ID(), ctorSetHash)`
- `internal/types/typechecker_core.go` — clear cache at type-check session boundaries (per top-level entry)
- `internal/types/tagged_union_predicate_test.go` — add memo-cache tests
- `internal/types/benchmarks/typecheck_bench_test.go` *(new ~80 LOC)* — benchmark `isTaggedUnion` call count + total time on citizen.ail-shaped fixtures

**Acceptance criteria**:
1. `isTaggedUnion` call count on the citizen.ail repro drops by ≥80% vs unmemoized baseline.
2. Cache key correctness: cycle-safe types (recursive ADTs) still terminate.
3. Existing 13 tests in `tagged_union_predicate_test.go` still pass.
4. New memo test verifies same-receiver same-ctorSet hits cache; different receiver doesn't.
5. Benchmark added and snapshotted in `internal/types/benchmarks/`.

**Risks**:
- Memo invalidation across `:reset` REPL commands. **Mitigation**: clear cache in `repl.Reset()` path.

---

### M3 — Iterative row-extension unifier (2 days, ~600 LOC)

**Risk**: high. Row-polymorphism is the most subtle part of the type system.

**Files modified**:
- `internal/types/row_unification.go` — convert `unifyRow` to explicit walk via constraint queue
- `internal/types/unification_records.go` — record-row variant uses the new queue
- `internal/types/unification_core.go` — top-level `unify` dispatcher routes to iterative variants
- `internal/types/iter_work_stack.go` — extend shared work-stack helpers if needed
- `internal/types/row_iterative_test.go` *(new ~250 LOC)* — regression tests for deep extensions, recursive row variables, order-sensitive substitution

**Acceptance criteria**:
1. Synthetic test: record with 500 fields unifies with `{x: int | r}` correctly.
2. Existing row-polymorphism tests (`row_unification_regression_test.go`, `record_unification_test.go`) pass unchanged.
3. Order-of-substitution preserved — verify via existing tests, no expected-output diffs.
4. WASM smoke: harness still exits 0 (citizen.ail loads).
5. No CLI perf regression via benchmark added in M2.

**Risks**:
- Row-polymorphism's substitution-propagation order is load-bearing for some constraint shapes (especially when extension variables get bound to other extensions). **Mitigation**: add property-based test fuzzing row shapes; keep recursive variant accessible via the same flag as M1 for A/B comparison.

---

### M4 — Restore full `citizen.ail` + wire harness into demos CI (0.5 day, ~100 LOC + config)

**Risk**: low.

**Steps**:
1. In the sister demos repo (`/Users/mark/dev/sunholo/demos`), restore `cognitive_commons/services/citizen.ail` from git history (commit before 2026-05-20 stub).
2. Rebuild WASM: `make build-wasm` here, copy to sister repo.
3. Run harness: `node scripts/wasm-loadmodule-harness.js` → exit 0 required.
4. Add a GitHub Action in the demos repo: on PR touching `**/*.ail` under `cognitive_commons/`, run the harness. **Block merge on non-zero exit.**
5. Flip the `AILANG_TYPECHECK_ITERATIVE` feature flag default to ON in `cmd/wasm/main.go`.
6. Remove the feature flag entirely in a follow-up cleanup commit (tracked separately).
7. Update demos/debug-notes/wasm-citizen-stack-overflow.md to mark "fixed in v1.1.0".

**Acceptance criteria**:
1. Harness exit 0 with original citizen.ail.
2. Demos CI workflow file exists + runs harness on PR.
3. Postmortem updated.
4. Feature flag flipped to ON by default.

---

## Day-by-Day Schedule

| Day | Milestone | Tasks |
|---|---|---|
| 1 | M1 | Design work-stack data structure; spike `checkExpr` rewrite for `match`-only nodes |
| 2 | M1 | Extend rewrite to `let`, lambda, function bodies; deferred-arm logic |
| 3 | M1 | Test suite + WASM smoke under flag; lint pass; merge dark |
| 4 | M2 | Memo cache + cycle-safe tests + benchmark; verify ≥80% call-count drop |
| 5 | M3 | Row unifier rewrite (record + open-row cases); regression tests |
| 6 | M3 | Edge cases (deep extensions, recursive variables); flag-on/flag-off A/B verification |
| 7 | M4 | Restore citizen.ail; demos-CI wire-up; flag flip; postmortem; final acceptance gate |

Buffer: this is tight. If M1 slips a day, M4 moves to day 8. Don't compress the acceptance gate (M4); skip the cleanup follow-up instead.

## Tests + Acceptance

### Every milestone must pass

1. `make test` — full Go test suite
2. `make lint` — golangci-lint clean
3. `make check-file-sizes` — no file > 800 lines
4. `cd /Users/mark/dev/sunholo/demos && node scripts/wasm-loadmodule-harness.js` — exit 0

### M1 + M3 additionally need

5. Synthetic deep-nesting fixtures in `internal/types/testdata/` exercising 500+-level recursion
6. Same fixtures run under WASM smoke (build wasm + harness with synthetic .ail)

### M4 final gate

7. The original `citizen.ail` (recovered from sister repo git history) loads in < 1s via the harness
8. Demos repo CI workflow file added and verified locally
9. Feature flag default flipped to ON

## Example Files

Following the project's "every feature needs an example" rule, this sprint adds:

- `examples/typecheck/nested_match_deep.ail` — exercises the iterative path (1000-level nested match). Verifies CLI + WASM.
- `examples/typecheck/repeated_tagged_union_match.ail` — three back-to-back matches on the same `Result`, the citizen.ail shape. Verifies M2's memoization.
- `examples/typecheck/deep_record_extension.ail` — 500-field record + open-row unification. Verifies M3.

## Risk Register

| Risk | Probability | Impact | Mitigation |
|---|---|---|---|
| M1 introduces ordering bug in nested lets / shadowed bindings | Medium | Breaks every program | Feature flag dark by default; A/B test entire eval suite under both modes for 1 week before flipping |
| M3 changes row-substitution order, breaking polymorphic types | Medium | Breaks generic functions silently | Property-based fuzz test in CI; flag-gated like M1 |
| WASM perf still bad due to other recursion in `internal/iface/` (constructor resolution) | Low | Acceptance gate fails | Profile via Chrome DevTools wasm flamegraph; if hit, follow-up sprint to convert iface/ as well — out of scope here |
| Memo cache leaks across REPL sessions, causing wrong-type returns | Low | REPL gives wrong answers | Add explicit cache clear in `repl.Reset()`; test via REPL session test |
| Demos CI harness wedges on network/dependency drift | Low | False positives on PRs | Pin Node version + Go toolchain in the workflow |

## Dependencies + Coordination

- **Dev branch state**: must be clean. No outstanding type-system PRs at sprint start.
- **Sister demos repo**: needs read access for citizen.ail restore + write access for CI workflow. Confirm `sunholo-voight-kampff` account has push.
- **No upstream blockers**: this is a self-contained refactor with no API changes.

## Out of Scope (Explicit Deferrals)

- **Make `instantiate` on imported schemes iterative**. M-SCHEME-IMPORT-PRESERVE-ADT-HEAD added work there; probably needs the same treatment but not on the citizen.ail critical path.
- **Web Worker for off-main-thread type-check**. Lifts WASM stack limit further but doesn't fix the underlying recursion. Different sprint.
- **AOT type-check caching**. Skip type-check on subsequent loads — big perf win but big surface area, not blocking this fix.
- **`internal/iface/` constructor resolution iterative**. If profiling shows it's the next bottleneck after M3, follow-up sprint.

## SPRINT_PLAN_PATH + SPRINT_JSON_PATH

**SPRINT_PLAN_PATH**: `design_docs/planned/v1_1_0/m-wasm-typecheck-iterative-sprint-plan.md`
**SPRINT_JSON_PATH**: `.ailang/state/sprints/sprint_M-WASM-TYPECHECK-ITERATIVE.json`
