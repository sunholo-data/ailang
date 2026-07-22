# M-EFFECT-ROW-SHOW-INTERP: Preserve Effect Rows Across Pure Nested Calls

**Status**: Planned — **QUORUM-BLOCKED, needs human design decision** (see Quorum Review Log). The problem analysis + root causes are verified and stable; the OPEN question is the *constraint-plumbing mechanism* for the application-local solver.
**Target**: v0.31.0
**Priority**: P0 (soundness regression / release gate)
**Estimated**: 1.5–2 days (12–16 hours)
**Dependencies**: None
**Issue**: GitHub #386
**Designer**: codex `gpt-5.6-sol` (rotation) · **Quorum**: gemini-3-1-pro (reject ×2) + controller (pass) → BLOCKED after 1 bounded revision round → parked for human.

## Quorum Review Log (2026-07-22, mission iteration 80)

Two independent quorum rounds by `gemini-3-1-pro` (distinct provider from the codex designer → generator≠judge). Both objections are **legitimate and soundness-relevant**; the doc was revised once (bounded), then re-quorum surfaced a second, deeper objection → parked per mission Gate-2 rule.

- **R1 objection (RESOLVED in revision):** the original Section A hedged between a new `EffectJoin` deferred constraint and an application-local solver, but never specified how `RowUnifier.UnifyRows` (`internal/types/row_unification.go`) would handle an un-reduced `EffectJoin` → panic / false-reject risk. **Resolution:** committed decisively to the **application-local equality solver**, removed `EffectJoin` entirely, added a structural proof obligation that no join can reach the unifier, and an invariant test. `row_unification.go` needs no change.
- **R2 objection (OPEN — the human decision):** Section A.3 says to *remove* solved equality constraints from `ctx.constraints`. But the let-boundary `SolveConstraints` "creates a fresh substitution but replays all accumulated constraints" — so deleting the locally-solved equalities means the let-boundary solver never sees them, leaving outer AST nodes and the type environment **permanently unsubstituted** (a new correctness bug). **gemini's proposed fix:** instead of deleting, *replace* the checkpointed constraints with the flattened accumulated substitution (emit a simple `a ~ T` per mapped variable) so the let-boundary solver still propagates the unifications to outer nodes without redundantly re-solving the application boundary.

**Why parked (not a 2nd revision):** the mission's bounded-quorum rule is one revision + one re-quorum. Two independent legitimate objections on a **soundness-critical inference constraint-solver** change is exactly where a human architect should ratify the mechanism before execution. The R2 fix is precise and likely correct, but it changes the drain semantics globally; committing to it (vs. an alternative like keeping constraints and idempotent re-solving, or a scoped substitution snapshot) is a design call worth one human confirmation. **To unpark:** Mark (or a human architect) picks the constraint-preservation mechanism for the application-local solver (gemini's flattened-substitution replacement is the leading candidate); then route straight to sprint-planner (no re-quorum — the analysis is settled).

## Problem Statement / Motivation

AILANG v0.30.0 incorrectly loses effects when an effectful call receives an argument that itself contains a pure function application. The smallest issue reproducer uses `show` inside an effectful `mapE` callback and then calls `foldlE`:

```ailang
module repro
import std/io (println)
import std/list (mapE, filterE, foldlE, flatMap)

export func main() -> () ! {IO} {
  let doubled = mapE(\x. { println("mapping ${show(x)}"); x * 2 }, [1,2,3]);
  let total = foldlE(func(acc: int, x: int) -> int ! {IO} { println("acc"); acc + x }, 0, [10,20,30]);
  println("done")
}
```

Verified on July 22, 2026 with the live binary `AILANG v0.30.0-105-ga6ebf50e7-dirty` at commit `a6ebf50e7`:

```text
Error: type error in repro (decl 0): type unification failed at [function application at repro.ail:7:21]: failed to unify parameter 0: failed to unify effect rows: incompatible closed rows: r1 has extra labels [], r2 has extra labels [IO]
```

This is a genuine type-system soundness bug, not merely a false rejection. An unannotated helper containing `println(show(x))` is accepted as pure, while replacing `show(x)` with a string literal correctly reports the missing `IO` effect:

```ailang
export func callback(x: int) -> int {
  println(show(x));
  x * 2
}
```

Observed result: `✓ No errors found!`. The corresponding `println("mapping")` helper fails with:

```text
Error: effect checking failed in .tmp_issue386/callback_plain: Effect checking failed for function 'callback'
  Function uses effects not declared in signature

  Missing effects: IO
```

The issue title names `show`/interpolation because interpolation desugars to `show` plus `concat_String`, making the bug common. The defect is broader: `println(intToStr(x))` is also accepted as pure, and a `mapE` callback containing that expression produces the same later closed-row failure. The fix must therefore repair application-effect composition and row polymorphism generally; it must not special-case `show`, interpolation, or list combinators.

**Impact:**

- Effect inference can silently erase `IO`, violating the declared-effect contract.
- Valid programs become order/use-history dependent because separate instantiations reuse unquantified row-variable names.
- Four runnable examples are quarantined under #386.
- The standard library signatures are not the cause; `std/list.ail` has exposed the same row-polymorphic signatures since v0.13.0-era code:
  - `mapE[a,b,e](f: (a) -> b ! {e}, ...) -> ... ! {e}`
  - `filterE[a,e](p: (a) -> bool ! {e}, ...) -> ... ! {e}`
  - `foldlE[a,b,e](f: (b,a) -> b ! {e}, ...) -> ... ! {e}`

## Goals

**Primary Goal:** Restore sound, fresh, compositional inference for effect rows across nested pure/effectful applications and repeated uses of row-polymorphic functions.

**Success Metrics:**

- `println(show(x))` and `println(intToStr(x))` infer `IO`, never `{}`.
- Every use of an imported row-polymorphic function receives fresh row variables.
- The minimal #386 reproducer and all regression-surface variants type-check.
- Existing missing-effect tests continue to reject undeclared `IO`, `FS`, `Env`, `Stream`, and `Process` effects.
- Four #386 effect-row examples return to `working` and leave `skippedExamples`.

## Root-Cause Analysis

Two independently unsound mechanisms interact.

### 1. `combineEffects` closes unresolved open rows

`internal/types/typechecker_functions.go:471` (`inferApp`) assigns every application a fresh open effect row and adds a delayed `TypeEq` constraint tying that row to the callee's actual function effect. It then immediately combines argument effects and the still-unresolved callee row:

```go
appEffects := append(argEffects, effectRow)
EffectRow: combineEffectList(appEffects)
```

`internal/types/typechecker_data.go:377` (`combineEffects`) unions known labels but explicitly ignores tail variables and returns `Tail: nil` whenever both inputs are non-empty/open rows:

```go
// For now, ignore tail variables in combination
return &Row{Kind: EffectRow, Labels: combined, Tail: nil}
```

For `println(show(x))` the sequence is:

1. `show(x)` receives a fresh application row later constrained to closed `{}`.
2. `println(...)` receives a separate fresh application row later constrained to `{IO}`.
3. Before either equality is solved, `combineEffectList` sees two open rows with empty known labels.
4. `combineEffects` drops both tails and materializes closed `{}`.
5. The lambda body therefore records `{}` even though the pending `println` constraint separately resolves to `{IO}`.

Interpolation amplifies the path because `internal/parser/parser_literals.go:69` lowers `"mapping ${show(x)}"` to synthesized `show(...)` and one or more `concat_String(...)` applications. However, the live `intToStr` reproducer proves dictionary resolution and interpolation are not the root cause; any nested pure application can create the extra unresolved application row that exposes the tail-dropping bug.

### 2. Effect-row variables are not generalized or freshened

`internal/types/typechecker_functions.go:427` (`generalizeWithConstraints`) collects free type variables but always emits:

```go
RowVars: []string{}, // Simplified for now
```

The interface builder preserves that omission. The live `std/list` interface cache contains, for example:

```text
mapE   row_vars=[]   callback_tail=e   outer_tail=ρ5
foldlE row_vars=[]   callback_tail=e   outer_tail=ρ3
```

The callback tail named `e` is present inside both types but absent from `Scheme.RowVars`. In addition, the source-level relation “callback uses `{e}` and the combinator itself uses the same `{e}`” has already drifted in the interface (`mapE` callback `e`, outer `ρ5`; `foldlE` callback `e`, outer `ρ3`). `internal/iface/builder.go:154` (`applyLabelsFromAST`) restores nested callback effect annotations from the AST but leaves the outer inferred effect row unchanged. Consequently the interface loses both quantification and equality of the two source occurrences.

Because `e` is not quantified, `Scheme.InstantiateWithConstraints` in `internal/types/types_v2.go:533` does not replace it with a fresh `RowVar`; separate imported uses share the literal substitution key `e`.

At each let boundary, `internal/types/typechecker_functions.go:179` calls `InferenceContext.SolveConstraints`. `SolveConstraints` creates a fresh substitution but replays all accumulated constraints. Once the first callback has incorrectly constrained shared `e` to closed `{}`, a later `foldlE` callback requiring `{IO}` collides with that stale identity and reaches the legitimate closed-vs-closed rejection in `RowUnifier.UnifyRows`:

```text
incompatible closed rows: r1 has extra labels [], r2 has extra labels [IO]
```

This also explains the broader live matrix:

| Program shape | Live result | Interpretation |
|---|---|---|
| effectful `mapE` callback with plain `println`, then `{IO}` `foldlE` | PASS | both uses happen to constrain shared `e` compatibly |
| `mapE` with `show`/interpolation, then `{IO}` `foldlE` | FAIL | first callback effect was unsoundly closed to `{}` |
| pure `mapE(\x. x * 2, ...)`, then `{IO}` `foldlE` | FAIL | proves missing fresh row instantiation independently of `show` |
| `mapE` with `show`, then a pure second `mapE` | PASS | both uses agree on closed `{}` |
| `mapE` with `show`, then an effectful second `mapE` | FAIL | second effectful combinator also consumes shared `e` |
| `foldlE` first, then `mapE` with `show` | FAIL at the later `mapE`, reversed error direction | ordering changes which call reports the collision, not whether incompatible shared uses fail |
| pure `flatMap`, top-level `show`, then `foldlE` | PASS | `flatMap` has no row-polymorphic callback effect to poison |

The last two rows refine the initial issue characterization using the required live-binary gate. They must be encoded as tests rather than preserving the narrower assumption that only a later `foldlE` or only one source order fails.

## Proposed Fix

The implementation must fix both mechanisms. Fixing only effect composition would leave pure-then-effectful combinator uses coupled through shared `e`; fixing only row freshening would still allow `println(show(x))` to be accepted as pure.

### A. Solve application effects locally before combining rows

**Decision:** use an application-local equality solver. This design does **not** add an `EffectJoin` type or constraint. Every application resolves and substitutes the equality constraints that determine its effect operands before calling `combineEffectList`; therefore no join representation can enter `InferenceContext`, a typed node, or `RowUnifier.UnifyRows`.

Modify:

- `internal/types/typechecker_functions.go`
  - `inferApp`
- `internal/types/typechecker_data.go`
  - `combineEffects`
  - `combineEffectList`
- `internal/types/inference_helpers.go`
  - add the application-local equality-solving helper and share its substitution/drain logic with `InferenceContext.SolveConstraints`

No modification is required in `internal/types/row_unification.go`. `RowUnifier.UnifyRows` continues to receive only ordinary `*Row` values; the application-local solver calls it only through existing `TypeEq`/`RowEq` decomposition, and there is no `EffectJoin` value in the type system for it to encounter.

Design:

1. In `inferApp`, take a constraint checkpoint before inferring the callee and arguments. After adding the application's `funcType ~ expectedFuncType` equality, solve every `TypeEq`/`RowEq` added since that checkpoint before constructing `TypedApp.EffectRow`; do not call the whole-context `SolveConstraints` from `inferApp`.
2. The local helper must process the checkpointed `TypeEq`/`RowEq` constraints in source order using the same `Unifier`/`RowUnifier` code as `SolveConstraints`, accumulate one substitution, and apply it to `funcNode`, `argNodes`, `resultType`, every `argEffect`, and the callee `effectRow` before effect union. For known callees this makes `show` closed `{}` and `println` closed `{IO}`, so their union is `{IO}`.
3. Rewrite all surviving constraints under the accumulated substitution. Remove the solved equality constraints from `ctx.constraints` so a later let-boundary `SolveConstraints` cannot replay them; retain substituted class constraints for defaulting/generalization.
4. Only then call `combineEffectList`. `combineEffects` may directly union fully closed rows and may preserve one unresolved tail shared by the substituted operands. It must never choose one of two distinct tails, equate independent tails merely to make combination possible, or convert an unresolved tail into `Tail:nil`.
5. Treat two distinct unresolved tails after application-local solving as an invariant failure in the implementation, not as a new row value. The local solver must include all equality constraints generated by the application subtree that can determine those tails before publishing the node. Add a regression for nested/repeated polymorphic applications that proves the supported rank-1 application paths reduce to closed rows or one shared tail before combination.

The invariant is mandatory and mechanically testable: **after `inferApp` returns, its stored `EffectRow` is an ordinary substituted `*Row` containing at most one tail; no `EffectJoin` type exists, no deferred join is present in `ctx.constraints`, and `RowUnifier.UnifyRows` can only observe ordinary rows.** The proof obligation is structural: Section A adds no new `TypeConstraint` or row representation; the checkpoint covers the whole application subtree; all equality constraints in that suffix are solved and drained; substitution happens before the distinct-tail guard and `combineEffectList`; and `UnifyRows` retains its existing `(*Row, *Row, Substitution)` input boundary. This keeps the core row unifier unchanged and fails loudly at the application boundary if a future language feature introduces a genuinely irreducible multi-tail union.

### B. Generalize every free row variable in a scheme

Modify:

- `internal/types/typechecker_functions.go`
  - `generalizeWithConstraints`
- `internal/types/inference.go`
  - add the row-variable counterpart of `baseEnvFreeVars` if needed
- `internal/types/inference_helpers.go`
  - extend row-variable collection through complete `Type` trees
- `internal/types/env.go`
  - verify `FreeRowVars` traverses outer and nested function/record positions
- `internal/types/types_v2.go`
  - `Scheme.InstantiateWithConstraints` (existing freshening path; add tests, change only if traversal exposes a defect)
- `internal/iface/builder.go`
  - `generalizeType`
  - `canonicalizeScheme`
  - `applyLabelsFromAST`
  - `restoreNestedEffectRow` (factor/reuse its AST-effect-row conversion)
- `internal/types/json.go` / interface cache key only if the serialized scheme shape or cache compatibility changes

Design:

1. Collect free row variables from the entire generalized type, including:
   - outer function effect rows;
   - nested callback function effect rows;
   - record rows and function types nested inside collections/tuples/ADTs.
2. Quantify rows free in the type but not free in the environment, mirroring the HM side condition already used for type variables. Preserve deterministic sorted order.
3. Track base-environment row variables separately from type variables so an enclosing binder's row cannot be unsoundly generalized.
4. When a function declaration has an explicit outer effect row, restore that row from `ast.FuncDecl.Effects` at the same interface boundary that already restores nested `ast.FuncType.Effects`. This must preserve the same source row-variable name across callback and outer positions before canonical alpha-renaming.
5. Ensure `Scheme.InstantiateWithConstraints` replaces each quantified effect row with a fresh `RowVar` of `EffectRow` kind at every local and imported use.
6. Regenerate/load `std/list` interfaces such that `mapE`, `filterE`, `foldlE`, `flatMapE`, and `forEachE` expose non-empty `row_vars`, and the callback/outer occurrence representing source-level `e` is alpha-equivalent and freshly instantiated per use.

Do not patch `std/list.ail`: its signatures are already correct and are required fixtures proving the inference/interface repair.

## Conflict Surface

*(Touches `internal/types/` application-local constraint plumbing and `internal/iface/`; no new constraint kind and no `row_unification.go` change.)*

### Position touched

1. **Application-local equality solving:** `inferApp` checkpoints, solves, substitutes, and drains its equality constraints before constructing the typed application effect.
2. **Application effect construction:** `inferApp` combines substituted argument-evaluation effects with the substituted called-function effect row.
3. **Sequential/structural effect union:** `combineEffects` / `combineEffectList` are used by applications, lets/blocks, binary operators, lists, records, tuples, branches, and other typed-expression constructors.
4. **Let/module generalization:** `generalizeWithConstraints` and interface generation decide which type and row variables become universally quantified.
5. **Scheme instantiation:** local/imported variable lookup freshens quantified variables before a use contributes constraints.

### Modification list

- `internal/types/typechecker_functions.go` — application checkpoint, local solve, substitution, and typed-node construction.
- `internal/types/typechecker_data.go` — tail-preserving effect combination with a loud distinct-tail invariant failure.
- `internal/types/inference_helpers.go` — shared equality solve/substitute/drain helper; no `EffectJoin` constraint.
- `internal/types/inference.go`, `internal/types/row_unification.go` — explicitly unchanged for Section A; no new constraint or row representation.
- Section B files remain as listed above for row quantification/freshening and interface preservation.

### Other constructs at those positions that MUST still work

| Position | Existing valid construct | Required behavior |
|---|---|---|
| Application arguments | pure literal passed to effectful call: `println("x")` | infer `{IO}` |
| Nested application | effectful outer call with pure inner call: `println(show(x))`, `println(intToStr(x))` | infer `{IO}`, never `{}` |
| Nested application | effectful argument plus effectful callee | union all required labels and preserve budgets/params/provenance |
| Higher-order call | polymorphic callback `(a)->b ! {e}` | caller and callback share the intended fresh `e` for that instantiation |
| Repeated polymorphic use | same combinator used twice with different effects | each use gets a fresh row; no cross-use contamination |
| Let generalization | ordinary type-class constraints (`Num`, `Eq`, `Ord`, `Show`) | retain the M-TYPE-LIST-SOUND constraint-survival fix |
| Let generalization | enclosing lambda type variables/rows | do not generalize binder-owned variables |
| Closed-row unification | genuinely incompatible `{}` versus `{IO}` annotations | continue to reject with the closed-row diagnostic |
| Effect refinements | budgets, minimum budgets, effect parameters, provenance | survive substitution and union unchanged except for defined budget composition |
| Imported schemes | ADTs, aliases, pure functions, nested callback types | preserve nominality, alias expansion, nested effect annotations, and equality between repeated source row variables |

### Programs that MUST still work as fixtures

1. `examples/runnable/effectful_list_t1_mapE_basic.ail` — single effectful `mapE`.
2. `examples/runnable/effectful_list_t2_filterE_bool.ail` — single effectful `filterE`.
3. `examples/runnable/effectful_list_t3_foldlE_acc.ail` — single effectful `foldlE`.
4. `examples/runnable/effectful_list_t4_flatMapE_expand.ail` and `effectful_list_t5_forEachE_unit.ail` — sibling row-polymorphic combinators.
5. `examples/runnable/effectful_list_t6_pure_flatMap.ail` and `effectful_list_t8_string_list.ail` — pure callback and non-numeric element controls.
6. `internal/pipeline/list_element_soundness_test.go` — class constraints survive scheme instantiation; no reopening of the prior list-element soundness hole.
7. `internal/pipeline/effect_soundness_test.go` — undeclared and distinct effects remain rejected.
8. `internal/types/row_unification_regression_test.go` — open/closed row unification remains symmetric and principal.
9. `internal/iface/nested_effects_test.go` — nested callback effect rows remain present across module interfaces.

### Intentional change

- Programs whose effects were silently erased by nested pure calls will now expose their real effects and may correctly fail missing-effect validation.
- Separate instantiations of a row-polymorphic scheme will no longer share row identities by name.
- The four #386 effect-row examples will move from quarantined/broken to working.
- No syntax, stdlib signature, runtime behavior, or valid closed-row incompatibility changes intentionally.

Anything else that stops compiling is a regression, not an intentional tightening.

## Testing Strategy

### Unit tests

- Add focused tests for effect union with closed/open rows, identical tails, budgets, params, and provenance; the application-boundary guard must reject distinct residual tails before `combineEffectList` can choose or discard one.
- Test that no `combineEffects` path turns an unresolved tail into a closed row.
- Test the application-local invariant: after nested and repeated polymorphic applications, `TypedApp.EffectRow` has at most one tail, `ctx.constraints` contains no solved application equality or deferred join, and an instrumented `RowUnifier.UnifyRows` receives only ordinary `*Row` values.
- Test full-type row-variable collection for outer functions, nested callbacks, records, tuples, lists, and ADT arguments.
- Test `generalizeWithConstraints` produces deterministic `RowVars` and withholds environment-owned rows.
- Test two calls to `Scheme.InstantiateWithConstraints` produce different row names.
- Test serialized/cached interface round-trips preserve `RowVars`.

### Integration tests

Add `internal/pipeline/effect_row_show_interp_test.go` (or equivalent) with `ModeCheck` fixtures for:

**Must accept:**

1. Minimal `mapE` + interpolation/`show` + `foldlE` reproducer.
2. Same program with an unannotated `foldlE` lambda.
3. Direct `println(show(x))` inside a callback.
4. `println(intToStr(x))` inside a callback (non-`show` guard); the fixture must include `import std/string (intToStr)`.
5. Pure `mapE` callback followed by `{IO}` `foldlE` (fresh-instantiation guard).
6. Effectful `mapE` followed by effectful `filterE`/second `mapE`.
7. Reverse source order (`foldlE` then `mapE`) with independent fresh rows.
8. Two uses of the same imported combinator with different effect rows.
9. Pure `flatMap` + top-level `show` + effectful `foldlE` control.

**Must reject:**

1. Unannotated pure function containing `println(show(x))` with a diagnostic naming missing `IO`.
2. Explicit callback annotation `! {}` whose body performs `IO` through a nested pure call.
3. Genuine incompatible closed rows unrelated to scheme reuse.
4. Existing `IO`/`FS`/`Env` non-subsumption cases.

### Regression-surface tests

- Run one automated fixture for every “Programs that MUST still work” entry above.
- Pin the normalized `std/list` interface shape: each effectful combinator's source `e` is represented in `row_vars`; callback and result effects preserve the intended relation.
- Pin deterministic row-variable ordering and cache/interface round-trip behavior.
- Run `make test`, `make lint`, `make verify-examples`, and `go run ./scripts/validate_manifest.go --ci`.

### Live acceptance checks

After implementation, rerun `ailang check` directly on:

- `examples/runnable/effectful_list.ail`
- `examples/runnable/effectful_list_t7_chain_combinators.ail`
- `examples/runnable/stream_multi_source.ail`
- `examples/runnable/stream_process_source.ail`

`examples/runnable/mcp_tools.ail` is explicitly out of scope: it currently fails with `cannot unify Option[string] with string`, a separate `getString` API/example migration issue.

## Milestones

### M1 — Pin the two unsound mechanisms (2–3h)

- [ ] Add E2E reproducer matrix and must-reject lost-`IO` tests.
- [ ] Add scheme/interface assertions showing missing row quantification.
- [ ] Add non-`show` nested-pure-call reproducer.

### M2 — Repair application effect composition (4–5h)

- [ ] Resolve and substitute application effect rows locally before union.
- [ ] Prove no deferred join or solved application equality escapes `inferApp`.
- [ ] Remove tail-dropping behavior from `combineEffects`.
- [ ] Preserve budgets, params, provenance, and deterministic ordering.
- [ ] Make `println(show(x))` infer `IO` and fail when undeclared.

### M3 — Quantify and freshen row variables (3–4h)

- [ ] Traverse complete types for free row variables.
- [ ] Apply the HM environment side condition to rows.
- [ ] Preserve row quantifiers through iface/cache serialization.
- [ ] Verify each combinator use receives fresh rows.

### M4 — Restore examples and validate (3–4h)

- [ ] Remove the four effect-row files from `scripts/verify_examples.go` `skippedExamples`.
- [ ] Change their `examples/manifest.json` status from `broken` to `working` and remove stale `broken` metadata.
- [ ] Leave `mcp_tools.ail` quarantined as the separate `Option[string]` issue.
- [ ] Run focused tests, full tests/lint, example verification, and manifest validation.

**Total:** 12–16 hours (approximately 1.5–2 days).

## Risks & Mitigations

| Risk | Impact | Mitigation |
|---|---|---|
| Application-local solving changes HM generalization/defaulting order | High | Checkpoint only the application's equality suffix, rewrite survivors under its substitution, retain class constraints, and lock M-TYPE-LIST-SOUND fixtures |
| The local solver publishes an application before all effect-determining equalities are available | High | Assert every published application row has at most one tail; cover nested and repeated polymorphic applications; fail loudly on distinct residual tails rather than adding an implicit join |
| Row quantification accidentally generalizes enclosing-binder rows | High | Add `baseEnvFreeRowVars`/withhold logic mirroring the proven type-variable side condition |
| Interface cache serves old unquantified schemes | Medium | Bump cache key/schema compatibility marker if required and test cache round-trip/invalidation |
| Budgets or parameterized effects are lost during new union path | High | Reuse/extend `UnionEffectRows` semantics and add budget/params/min-budget/provenance tests |
| Fix over-accepts real closed-row incompatibilities | High | Preserve direct closed-vs-closed negative tests and `effect_soundness_test.go` |
| Scope expands into parser or stdlib changes | Low | Treat interpolation and `std/list.ail` as fixtures only; no syntax or signature edits |

## Non-Goals

- Special-casing `show`, string interpolation, `mapE`, `filterE`, or `foldlE`.
- Changing `std/list.ail` signatures or combinator runtime behavior.
- Fixing `examples/runnable/mcp_tools.ail`; its `Option[string]` mismatch is separate.
- Redesigning the public effect syntax or adding effect subtyping.
- Weakening closed-row incompatibility checks.

## Acceptance Criteria

- [ ] `ailang check` accepts the minimal #386 reproducer.
- [ ] `ailang check` accepts pure-then-effectful and effectful-then-effectful repeated combinator uses in either source order.
- [ ] `ailang check` rejects an unannotated function containing `println(show(x))` and reports missing `IO`.
- [ ] `println(intToStr(x))` follows the same sound behavior, proving no `show` special case; its fixture imports `std/string (intToStr)`.
- [ ] `std/list` exported schemes quantify their effect row variables, and repeated instantiations are fresh.
- [ ] No unresolved effect tail is silently converted to a closed row by effect combination.
- [ ] No `EffectJoin` representation exists or reaches `RowUnifier.UnifyRows`; application tests prove every published row is ordinary and has at most one tail.
- [ ] Existing row-unification, nested-interface-effect, type-class constraint survival, and effect non-subsumption tests pass.
- [ ] `examples/runnable/effectful_list.ail` passes and is un-quarantined.
- [ ] `examples/runnable/effectful_list_t7_chain_combinators.ail` passes and is un-quarantined.
- [ ] `examples/runnable/stream_multi_source.ail` passes and is un-quarantined.
- [ ] `examples/runnable/stream_process_source.ail` passes and is un-quarantined.
- [ ] `examples/runnable/mcp_tools.ail` remains explicitly quarantined/out of scope until its `Option[string]` migration is fixed.
- [ ] `make test`, `make lint`, `make verify-examples`, and manifest CI validation pass with no new quarantines.

## Axiom Compliance

| Axiom | Score | Rationale |
|---|---:|---|
| A1 — The Language Must Compose With Itself | +1 | Independent higher-order combinator uses regain compositional row inference. |
| A2 — Failure Must Be Representable | 0 | Diagnostics remain explicit; no new failure representation. |
| A3 — The Language Is a System Boundary | +1 | Missing effects can no longer disappear across nested calls. |
| A4 — Determinism Is a Semantic Invariant | +1 | Fresh row identities and sorted quantifiers remove source-history/name-collision dependence. |
| A5 — Execution Must Be Replayable and Auditable | +1 | Static effect rows again match operations that execution traces can observe. |
| A6 — Effects Are Real and Must Be Legible | +1 | Directly repairs erased `IO` and preserves explicit effect accounting. |
| A7 — Authority Must Be Explicit | +1 | Effectful operations cannot masquerade as pure and bypass declared capability review. |
| A8 — Verification Should Be Local, Bounded, and Automatable | +1 | Adds focused inference/interface fixtures plus full example gates. |
| A9 — Concurrency Must Not Destroy Meaning | 0 | Stream examples benefit, but concurrency semantics do not change. |
| A10 — Machines Are Primary Readers | +1 | Principal, deterministic schemes reduce order-sensitive compiler behavior. |
| A11 — Syntax Is a Liability | 0 | No syntax change. |
| A12 — Cost Is Part of Meaning | 0 | No runtime cost model change; compiler overhead must remain bounded. |

**Net score:** +8. **Hard violations:** none.

## References

- GitHub issue #386.
- `internal/types/typechecker_functions.go` — `inferApp`, `inferLet`, `generalizeWithConstraints`.
- `internal/types/typechecker_data.go` — `combineEffects`, `combineEffectList`.
- `internal/types/inference_helpers.go` — `SolveConstraints`, row-variable collection.
- `internal/types/row_unification.go` — closed/open row unification.
- `internal/types/types_v2.go` — `Scheme.InstantiateWithConstraints`, `Row.Substitute`.
- `internal/iface/builder.go` — scheme generalization/canonicalization and nested effect restoration.
- `internal/parser/parser_literals.go` — interpolation lowering to `show`/`concat_String`.
- `std/list.ail` — unchanged row-polymorphic combinator signatures.
