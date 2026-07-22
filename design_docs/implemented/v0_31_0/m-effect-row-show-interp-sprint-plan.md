# Sprint Plan: M-EFFECT-ROW-SHOW-INTERP (#386)

**Design doc**: `design_docs/planned/v0_31_0/m-effect-row-show-interp.md`
**Status**: MECHANISM RATIFIED by Mark 2026-07-22 — no re-quorum, no re-design.
**Target**: v0.31.0 · **Priority**: P0 (soundness regression / release gate) · **Issue**: #386
**Worktree**: `.claude/worktrees/effect-row-386` · **Branch**: `sprint/m-effect-row-show-interp` (base `origin/dev` a6d42a0a4)
**Estimate**: 12–16 hours (1.5–2 days) · **Risk**: HIGH (core type-inference constraint solver)

---

## Ratified Mechanism (NON-NEGOTIABLE — encode verbatim)

Three pillars. Every milestone acceptance is gated on all three where relevant.

1. **Application-local equality solver** — `inferApp` checkpoints the constraints added by
   the whole application subtree, solves those `TypeEq`/`RowEq` constraints *locally* (same
   `Unifier`/`RowUnifier` as `SolveConstraints`), and applies the accumulated substitution to
   `funcNode`, `argNodes`, `resultType`, every `argEffect`, and the callee `effectRow`
   BEFORE calling `combineEffectList`. **NO new `EffectJoin` type or constraint.**
   `internal/types/row_unification.go` is UNCHANGED (structural proof + invariant test required).

2. **Constraint plumbing = REPLACE-NOT-DELETE** — solved equality constraints are NOT deleted
   from `ctx.constraints`. Each is REPLACED with its flattened `a ~ T` substitution form (one
   simple equality per mapped variable), so the let-boundary `SolveConstraints` replay (which
   starts from a fresh substitution) re-derives identical facts and outer AST nodes / the type
   environment are NEVER left permanently unsubstituted. **A delete-based implementation FAILS
   acceptance** (M2-AC7 fixture below is designed to catch it).

3. **Row-variable generalization (Section B)** — `generalizeWithConstraints` currently emits
   `RowVars: []string{} // Simplified for now` (`typechecker_functions.go:464`). Replace with
   full-type free-row-variable collection + the HM environment side condition, so every use of
   an imported row-polymorphic function (`mapE`/`filterE`/`foldlE`/`flatMapE`/`forEachE`) gets
   FRESH row vars per instantiation.

---

## Live Reality Check (verified at HEAD a6d42a0a4, worktree binary rebuilt)

| Premise (doc) | Live result | Status |
|---|---|---|
| `combineEffects` drops tails → `Tail: nil` | `typechecker_data.go:377`, `// For now, ignore tail variables` at :409 area; returns `Tail: nil` | CONFIRMED |
| `RowVars: []string{} // Simplified for now` | `typechecker_functions.go:464` (inside `generalizeWithConstraints` @ :427) | CONFIRMED |
| `combineEffectList` | `typechecker_data.go:417` | CONFIRMED |
| `SolveConstraints` | `inference_helpers.go:173` | CONFIRMED |
| Minimal repro FAILs `incompatible closed rows … extra labels [IO]` at `7:21` | reproduced exactly | CONFIRMED |
| `println(show(x))` unannotated → `✓ No errors` (IO erased) | reproduced | CONFIRMED (soundness bug) |
| control `println("literal")` → `Missing effects: IO` | reproduced | CONFIRMED |
| `println(intToStr(x))` → `✓ No errors` (non-show, same bug) | reproduced | CONFIRMED |
| `std/list` combinators row-polymorphic `! {e}` | `std/list.ail:217–261` (mapE/filterE/foldlE/flatMapE/forEachE), flatMap @:202 | CONFIRMED |
| `intToStr` export | `std/string.ail:60 export pure func intToStr` | CONFIRMED |
| 4 examples quarantined in `scripts/verify_examples.go` `skippedExamples` | :99–103 (effectful_list, t7, stream_multi_source, stream_process_source; mcp_tools separate @:101) | CONFIRMED |
| Regression test files exist | `list_element_soundness_test.go`, `effect_soundness_test.go`, `row_unification_regression_test.go`, `nested_effects_test.go` all present | CONFIRMED |

**No premise errors found.** One filepath nuance: doc cites the stdlib as `std/list.ail` /
`std/string.ail` (repo-root `std/`, NOT `stdlib/std/`) — this is correct; the repo has both a
`std/` (source stdlib) and there is no `stdlib/std/`. Line numbers in the doc (`:377`, `:464`,
`:471`, `:179`) all match live.

---

## Milestones

### M1 — Pin the two unsound mechanisms (RED tests) (2–3h)

Add failing tests that lock the bug + the missing quantification BEFORE any fix. These are the
non-vacuity anchor: they must be RED now and GREEN only after M2+M3.

- Add `internal/pipeline/effect_row_show_interp_test.go` with the full **must-accept** matrix
  (9 fixtures, doc §Integration) as currently-RED `ModeCheck` cases.
- Add the **must-reject** matrix (4 fixtures) — the unannotated `println(show(x))` case is
  currently WRONGLY GREEN (accepts) → this test is RED now, proving the soundness hole.
- Add non-`show` reproducer `println(intToStr(x))` (imports `std/string (intToStr)`).
- Add scheme/interface assertion test showing `std/list` `mapE`/`foldlE` schemes currently have
  EMPTY `row_vars` (RED — will flip in M3).

**Acceptance M1**: New test file compiles and runs; the must-accept matrix + the
`println(show(x))`-must-reject case + the empty-`row_vars` assertion are all RED (documented
failing), proving each targets a real defect. No production code changed yet.

---

### M2 — Repair application effect composition (application-local solver + REPLACE-NOT-DELETE) (4–5h)

Files: `internal/types/typechecker_functions.go` (`inferApp`),
`internal/types/typechecker_data.go` (`combineEffects`, `combineEffectList`),
`internal/types/inference_helpers.go` (shared local solve/substitute/**replace** helper).
**`internal/types/row_unification.go` MUST NOT be modified.**

- **AC1** In `inferApp`: checkpoint `ctx.constraints` before inferring callee+args; after adding
  the `funcType ~ expectedFuncType` equality, solve every `TypeEq`/`RowEq` added since the
  checkpoint locally (reuse `SolveConstraints`'s `Unifier`/`RowUnifier` path); do NOT call
  whole-context `SolveConstraints` from `inferApp`.
- **AC2** Apply the accumulated substitution to `funcNode`, `argNodes`, `resultType`, every
  `argEffect`, and the callee `effectRow` before constructing `TypedApp.EffectRow`.
- **AC3 (REPLACE-NOT-DELETE)** The solved equality constraints are REPLACED in `ctx.constraints`
  with their flattened `a ~ T` substitution form (one equality per mapped var), NOT deleted.
  Substituted class constraints are retained for defaulting/generalization.
- **AC4** `combineEffects` no longer converts an unresolved tail into `Tail: nil`; it unions
  fully-closed rows directly and may preserve ONE shared tail. Budgets, params, provenance, and
  deterministic label ordering survive the new union path.
- **AC5 (distinct-tail invariant)** Two distinct unresolved tails after application-local solving
  are a LOUD invariant failure (fail-closed), not a new row value / not an implicit join.
- **AC6 (structural no-join proof)** Add an invariant/instrumentation test: after `inferApp`
  returns, `TypedApp.EffectRow` is an ordinary substituted `*Row` with ≤1 tail; `ctx.constraints`
  contains NO deferred join and no un-replaced solved application equality; an instrumented
  `RowUnifier.UnifyRows` observes only ordinary `*Row` values. Grep-assert no `EffectJoin`
  identifier exists in the type system.
- **AC7 (REPLACE-NOT-DELETE regression fixture — catches delete-based impl)** A `ModeCheck`
  fixture where the fix'd application sits INSIDE a `let` whose result type must be propagated to
  an OUTER node at the let boundary. With replace-not-delete the let-boundary `SolveConstraints`
  replay re-derives the substitution and the outer node is fully substituted → PASS. A
  delete-based implementation leaves the outer node unsubstituted → this fixture FAILS (asserts on
  the outer node's resolved type / a downstream unify that only closes if the outer node saw the
  substitution). This fixture MUST be present and MUST distinguish replace from delete.
- **AC8** `println(show(x))` and `println(intToStr(x))` now infer `{IO}` (never `{}`): the
  unannotated-pure must-reject cases from M1 flip to correctly REJECTED with a `Missing effects:
  IO` diagnostic; the control `println("literal")` rejection is unchanged.

**Acceptance M2**: AC1–AC8 all pass; `row_unification.go` diff is empty; no `EffectJoin` type
introduced; the REPLACE-NOT-DELETE fixture (AC7) is present and would fail under a delete impl;
`internal/pipeline/effect_soundness_test.go` (IO/FS/Env non-subsumption) still fully green.

---

### M3 — Quantify and freshen row variables (Section B) (3–4h)

Files: `typechecker_functions.go` (`generalizeWithConstraints`), `inference.go`
(row counterpart of `baseEnvFreeVars` if needed), `inference_helpers.go` (row collection through
complete `Type` trees), `env.go` (`FreeRowVars` traversal), `types_v2.go`
(`Scheme.InstantiateWithConstraints` — tests; change only if traversal exposes a defect),
`iface/builder.go` (`generalizeType`, `canonicalizeScheme`, `applyLabelsFromAST`,
`restoreNestedEffectRow`), `types/json.go` / cache key only if serialized scheme shape changes.

- **AC1** Replace `RowVars: []string{} // Simplified for now` with collection of free row vars
  across the ENTIRE generalized type: outer function effect rows, nested callback function effect
  rows, and rows nested inside collections/tuples/records/ADTs.
- **AC2** Quantify rows free in the type but NOT free in the environment (HM side condition,
  mirroring the existing type-var `withhold` logic). Track base-env row vars separately so an
  enclosing binder's row is never generalized. Deterministic sorted `RowVars` order.
- **AC3** At the interface boundary, restore an explicit outer effect row from
  `ast.FuncDecl.Effects` at the same point that already restores nested `ast.FuncType.Effects`,
  preserving the same source row-var name across callback and outer positions before canonical
  alpha-renaming.
- **AC4** `Scheme.InstantiateWithConstraints` replaces each quantified effect row with a fresh
  `RowVar` of `EffectRow` kind at every local and imported use; two calls produce DIFFERENT row
  names (unit test).
- **AC5** Regenerate/load `std/list` interfaces so `mapE`, `filterE`, `foldlE`, `flatMapE`,
  `forEachE` expose non-empty `row_vars`, with the source-level `e` alpha-equivalent across
  callback+outer positions and freshly instantiated per use. `std/list.ail` is NOT edited (fixture).
- **AC6** Serialized/cached interface round-trip preserves `RowVars`.

**Acceptance M3**: The M1 empty-`row_vars` assertion flips GREEN (non-empty, correct shape); the
fresh-instantiation must-accept fixtures pass (pure-`mapE`-then-`{IO}`-`foldlE`; reverse order;
two uses of same combinator with different effects); `internal/iface/nested_effects_test.go` and
`internal/types/row_unification_regression_test.go` still green; `std/list.ail` unchanged.

---

### M4 — Restore examples + full validation (3–4h)

- **AC1** Remove the four #386 effect-row entries from `scripts/verify_examples.go`
  `skippedExamples` (`effectful_list.ail`, `effectful_list_t7_chain_combinators.ail`,
  `stream_multi_source.ail`, `stream_process_source.ail`); LEAVE `mcp_tools.ail` quarantined
  (separate `Option[string]` issue).
- **AC2** Flip those four files' `examples/manifest.json` status `broken` → `working`, removing
  stale `broken` metadata; `mcp_tools.ail` stays quarantined.
- **AC3** Live acceptance: `ailang check` passes on `examples/runnable/effectful_list.ail`,
  `effectful_list_t7_chain_combinators.ail`, `stream_multi_source.ail`, `stream_process_source.ail`.
- **AC4** Full gates green: `make test`, `make lint`, `make verify-examples`,
  `go run ./scripts/validate_manifest.go --ci`.

**Acceptance M4**: Four examples un-quarantined + `working` + live `ailang check` clean;
`mcp_tools.ail` still quarantined; all four make/CI gates green with NO new quarantines.

---

## Conflict Surface (evaluator will check these — from doc §Conflict Surface)

**Positions touched (must all still function):**
1. Application-local equality solving in `inferApp` (checkpoint/solve/substitute/replace-drain).
2. Application effect construction (substituted arg effects ∪ substituted callee effect row).
3. Sequential/structural effect union via `combineEffects`/`combineEffectList` — also used by
   lets/blocks, binary operators, lists, records, tuples, branches.
4. Let/module generalization (`generalizeWithConstraints` + interface generation).
5. Scheme instantiation (freshening quantified vars on local/imported lookup).

**Constructs that MUST still work:**
- `println("x")` → `{IO}`; `println(show(x))` / `println(intToStr(x))` → `{IO}` never `{}`.
- effectful arg + effectful callee → union all labels, preserve budgets/params/provenance.
- polymorphic callback `(a)->b ! {e}` shares the intended FRESH `e` per instantiation.
- same combinator twice with different effects → fresh row each, no cross-use contamination.
- ordinary class constraints (`Num`/`Eq`/`Ord`/`Show`) survive (M-TYPE-LIST-SOUND preserved).
- enclosing-lambda binder vars/rows NOT generalized.
- genuine `{}` vs `{IO}` annotation mismatch → still REJECTED with closed-row diagnostic.
- budgets / min-budgets / effect params / provenance survive substitution+union.
- imported ADTs/aliases/pure/nested-callback schemes preserve nominality + repeated-row equality.

**Programs that MUST still work as fixtures:**
- `examples/runnable/effectful_list_t1_mapE_basic.ail`, `_t2_filterE_bool.ail`, `_t3_foldlE_acc.ail`
- `_t4_flatMapE_expand.ail`, `_t5_forEachE_unit.ail` (sibling combinators)
- `_t6_pure_flatMap.ail`, `_t8_string_list.ail` (pure-callback + non-numeric controls)
- `internal/pipeline/list_element_soundness_test.go` (class-constraint survival)
- `internal/pipeline/effect_soundness_test.go` (undeclared/distinct effects rejected)
- `internal/types/row_unification_regression_test.go` (open/closed symmetry + principality)
- `internal/iface/nested_effects_test.go` (nested callback effect rows preserved across ifaces)

---

## Non-Vacuity / Regression Guard (mandatory)

The fix must be proven to ACCEPT the now-accepted programs AND still REJECT the controls:
- **Accepts (were failing/erased):** minimal repro + 9 must-accept variants; four un-quarantined examples.
- **Still rejects (controls):** unannotated `println(show(x))` → missing `IO`; explicit `! {}`
  body doing IO via nested pure call → reject; genuine incompatible closed rows → reject; existing
  IO/FS/Env/Stream/Process non-subsumption tests → reject.
- **Distinguishes replace-from-delete:** M2-AC7 let-boundary outer-node fixture fails under delete.

A green run where the must-reject controls also pass (i.e., still reject) is required — a fix that
merely makes everything compile is a REGRESSION, not a pass.

## Non-Goals (do not drift)

- No special-casing `show`/interpolation/`mapE`/`filterE`/`foldlE`.
- No `std/list.ail` signature or runtime change.
- No `mcp_tools.ail` fix (`Option[string]` is separate).
- No new effect syntax / subtyping; no weakening of closed-row incompatibility.
- No `EffectJoin` type/constraint; no `row_unification.go` change.

## Testing Strategy (maps to doc §Testing Strategy)

- **Unit** (M2/M3): effect-union closed/open/tail/budget/param/provenance; no-tail-to-closed;
  application-local invariant (≤1 tail, no join in `ctx.constraints`, instrumented `UnifyRows`
  sees only `*Row`); full-type row-var collection; deterministic `RowVars` + env withhold; two
  `InstantiateWithConstraints` calls differ; iface round-trip preserves `RowVars`.
- **Integration** (M1→M2/M3): `effect_row_show_interp_test.go` — 9 must-accept + 4 must-reject.
- **Regression-surface** (M2/M3/M4): one fixture per MUST-still-work entry; pinned normalized
  `std/list` interface shape; deterministic row ordering + cache round-trip.
- **Live acceptance** (M4): `ailang check` on the four examples; `mcp_tools.ail` stays out.

## Risks (from doc)

| Risk | Mitigation |
|---|---|
| Local solve perturbs HM generalization/defaulting order | checkpoint only the app's equality suffix; REPLACE survivors (not delete); retain class constraints; lock M-TYPE-LIST-SOUND fixtures |
| App published before all effect-determining equalities available | assert ≤1 tail; cover nested+repeated polymorphic apps; loud distinct-tail failure |
| Row quantification generalizes enclosing-binder rows | `baseEnvFreeRowVars`/withhold mirroring type-var side condition |
| Iface cache serves old unquantified schemes | bump cache key/schema marker if needed; test round-trip/invalidation |
| Delete-based drain leaves outer nodes unsubstituted | REPLACE-NOT-DELETE + M2-AC7 let-boundary fixture |
| Over-accepts real closed-row incompatibilities | preserve closed-vs-closed negatives + `effect_soundness_test.go` |
| Scope creep into parser/stdlib | interpolation + `std/list.ail` are fixtures only |

**SPRINT_PLAN_PATH**: `design_docs/planned/v0_31_0/m-effect-row-show-interp-sprint-plan.md`
**SPRINT_JSON_PATH**: `.ailang/state/sprints/sprint_M-EFFECT-ROW-SHOW-INTERP.json`
