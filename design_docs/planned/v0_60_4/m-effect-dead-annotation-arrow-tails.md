# M-EFFECT-DEAD-ANNOTATION-ARROW-TAILS — Scheme-borne latent tails fire the #616 invariant on pure programs

**Status**: Planned
**Target**: v0.60.4
**Priority**: P1 (High) — a whole class of observationally pure programs fails with an *internal
invariant* message; users route around it by abandoning function values in records (the reporter
already rewrote sunholo/worlds `selection.ail` to use an ADT tag + float instead of closures).
**Estimated**: 2–3 days (Phase 0 spike ≈ 0.5, Phase 1 ≈ 1, Phase 2 ≈ 0.5, corpus/docs ≈ 0.5–1)
**Dependencies**: None blocking. Builds on the landed F-B tail-closing family
(`internal/types/typechecker_lambda_open.go`, #1958 re-land) and the #1718/#1826 nested hand-off
publication (`internal/types/typechecker_app_effects.go`, `typechecker_latent_data.go`).

## Axiom Compliance

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | No new nondeterminism; charge results become deterministic across module re-elaboration instead of invariant-dependent |
| A2: Replayability | 0 | No impact on traces |
| A3: Effect Legibility | +1 | Pure programs stop being charged rows that resolve to nothing; genuine latent charges (six/seven class) remain visible |
| A4: Explicit Authority | 0 | No capability changes |
| A5: Bounded Verification | +1 | Effect decisions stay local to a declaration + its call sites; no whole-program analysis added |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | Replaces an "internal invariant" dead-end with either a clean check or a named, actionable diagnostic; removes the `ε_annotN`/`ρuNN` riddle from machine-facing output |
| A8: Minimal Syntax | +1 | No new syntax; existing `! {}`/`! {e}` arrow annotations gain full meaning |
| A9: Cost Visibility | 0 | Charge set unchanged for effectful programs |
| A10: Composability | +1 | Containers holding function values compose through helper functions (the reported `withBand` shape) instead of failing |
| A11: Structured Failure | +1 | Residual genuine conflicts get a coded, structured error instead of an internal-invariant string |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +5** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): no implicit nondeterminism introduced
- [x] A3 (Effects): no hidden side effects; every rejected effectful fixture stays rejected
- [x] A4 (Authority): no ambient access granted
- [x] A7 (Machines First): not optimizing for human convenience over machine analysis

## Problem Statement

### The report (v0.56.1, still live on dev v0.60.2)

`ailang check` on an 11-line pure program (module `r3`, Issue text, reproduced verbatim in
`internal/pipeline` test fixtures below) prints:

```
Error: effect checking failed in tmp/repro/r3: function run: internal invariant:
missing callback row for typed application 67 at r3.ail:10:17 (argument 0)
```

Type checking passes. There is not a single effect in the program: every function is `pure`,
every stored closure is pure, and the second list is empty. A related form fails with
`Unresolved effect tail: e_annot1` instead. Reproduced on the current dev binary
(v0.60.2, `33f3009`), same messages modulo tail numbering.

### Verified repro matrix (all run with `ailang check`, v0.60.2, `AILANG_NO_CACHE=1`)

Every program below is in the Phase 1 test plan verbatim; transcripts were captured in this
session (files preserved under the repro names used below).

| # | Shape | Result | Exact message |
|---|---|---|---|
| r3 | Reported repro: type alias with **two** `[float -> float]` fields, record built by a helper storing a caller callback, passed to a pure callee from inside a lambda | ✗ | `internal invariant: missing callback row for typed application 67 … (argument 0)` |
| ten | **Single** function field; the pure stored closure comes from a helper's scheme (`withOne`) | ✗ | `Unresolved effect tail: ε_annot1` |
| eleven | As ten, but the stored closure doesn't even reference the callback (`\v. v*2.0`) | ✗ | `Unresolved effect tail: ε_annot1` |
| three | Two fields, callee **not** declared pure, field it never reads | ✗ | `Unresolved effect tail: ε_annot2` (over-charges a dead field) |
| two | Two fields, record literal inline at the call | ✗ | `Unresolved effect tail: ε_annot2` |
| one | Single field, lambda stored **locally** in the caller's own record literal | ✓ clean | — |
| five | Two fields, record from a pure producer, passed directly | ✓ clean | — |
| four | Two fields, arrow rows explicitly `! {}` in the type decl | ✓ clean | — |
| eight | r3 with only the **dead** field annotated `! {}` | ✗ (moved) | `Unresolved effect tail: ρu14` — the *live* field's scheme-borne tail now leaks alone |
| six | Single field, callee pure, caller stores an `! {IO}` function in it | ✗ (correct) | `Missing effects: IO` — **must keep failing** |
| seven | As six with two fields | ✗ (correct) | `Unresolved effect tail: ε_annot2; Missing effects: IO` — must keep failing |
| threed/threee | Callee's row genuinely shares the callback tails (row-polymorphic callee) | ✓ clean | — |

**Impact**: the trigger is broader than the report and narrower than "records with function
values": any container whose function-typed fields flow through a *row-polymorphic scheme*
(the helper's return type) fails — even with one field and a constant pure closure (ten/eleven).
Containers built and consumed locally (one/five) work. The reporter's workaround — replace
closures in records with data — is a real loss of expressiveness for AI-generated code, and the
error text gives no path to the `! {}` fix that four proves exists.

### Root cause (mechanism; every link code- or transcript-verified)

1. **Annotation-less arrows mint open rows.** Every `FuncType` in a type annotation without an
   effect row gets a fresh open tail `ε_annot<N>` and `ImplicitEffectContract: true`
   (`internal/elaborate/file_funcs.go:405–432`; mint site `:412`). This is deliberate row
   polymorphism: `alias_two_callers`, `ok_rowpoly` and R5 require one alias to serve both a pure
   and an `! {IO}` caller.
2. **Schemes quantify those tails; call sites instantiate fresh copies owned by no caller
   scope.** The F-B closing family only tracks lambda-opened tails:
   `ctx.openedLambdaTails` is populated exclusively by `openLambdaRow`
   (`internal/types/typechecker_lambda_open.go:55–58`; verified by grep — no other writer), so
   `closeOpenedTailsForGeneralization`/`closeOpenedLambdaTails` never see `ε_annot` tails.
3. **The #1718 nested hand-off charges every masked argument's data-position arrows at every
   application** (`nestedLatentRow` over the *argument's* type,
   `internal/types/typechecker_latent_data.go:126–152`; validator union at
   `internal/pipeline/validate_effects.go:397–411`).
4. **The charge row can carry exactly one tail.** `nestedLatentRow` unions per-arrow rows with
   `CheckedUnionEffectRows`; two distinct surviving tails → conflict → `nil` → the validator's
   fail-loud `missing callback row` invariant. Exactly one surviving tail → it leaks into the
   caller's required row as `Unresolved effect tail: ε_annotN`/`ρuNN`.
5. **The erasure/ownership machinery that makes one/five pass does not fire for scheme-borne
   tails.** `implicitRowsAbove` (publish-time scan of binder types) and `resolveOwnedTypes`
   (finalize-time re-anchor) plus the F-B closing determine which tails are the caller's
   caller's to pay and which resolve to `{}`. In the ten/eleven/eight shapes a tail whose only
   provenance is a provably-pure stored closure (an F-B opened tail, unbound at its declaration)
   nonetheless survives publication as an open, unowned charge. The exact break is one of three
   code-grounded candidates (see Phase 0); the shipped binary's output pins *that* it breaks,
   not *where*.
6. **This is the documented residual risk of the v0.56.0 nested-callbacks design firing on an
   all-pure program.** `m-effect-latent-nested-callbacks.md` (Risk table): *"nestedLatentRow
   unions two distinct unresolved tails … A conflict is an explicit invariant (#616), never
   first-wins."* The fail-loud choice is right for genuinely polymorphic conflicts (the `fwdE`
   fixture); it is wrong when both tails resolve to nothing and the program is pure.

## Goals

**Primary Goal:** A program that stores only pure function values in data containers and hands
them to pure callees checks clean — regardless of how many function-typed fields the container
type has and whether the container flowed through a helper's scheme; genuinely effectful stored
callbacks keep today's rejections byte-for-byte.

**Success Metrics:**
1. r3, ten, eleven, two, three (the all-pure matrix) check clean; no `internal invariant` or
   `Unresolved effect tail` for any of them.
2. six, seven, and every existing nested-callback fixture (R1–R11, S1–S6, controls, intended
   flips, `fwdE`, `TestLatentNested_RunRejectsBeforeExecution`) produce byte-identical
   verdicts and first diagnostics.
3. The residual genuine conflict (`fwdE` class) reports a coded, actionable diagnostic naming
   the construct — never the string `internal invariant`.
4. Corpus sweep (all tracked `.ail` files, `examples/`, `std/`, motoko + package checkouts
   per the v0.56.0 doc's method) shows zero unplanned flips.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| D1: Pure-stored closures' scheme-borne tails must resolve to `{}` at hand-off (r3/ten/eleven check clean), not merely fail with a better message | Decides whether the reporter's workaround (no function values in records) stays necessary | human (reporter's stated expectation; freeze item) | design | med |
| D2: Root mechanism = extend the F-B "a tail that nothing constrains must not outlive its declaration" closing to scheme-borne data-position tails at publication; **not** multi-tail rows, **not** validator-side tail dropping | Chooses between three architecture families with different blast radii | agent (this doc), human veto via freeze | design | med |
| D3: The two-explicit-tails conflict (`fwdE`) stays fail-loud, but as a coded diagnostic (`TC_LATENT_001`, verified unallocated) with the fix spelled out | Keeps a pinned test green while removing the internal-invariant dead end | agent | design | low |
| D4: Whether a caller may pass an effectful-callback container into a field position the callee never reads (today: payable by declaring effects; after D2's closing: potentially a type error at that callee) | A semantic narrowing of "store-only hand-off" if the closing reaches linked-but-dead positions | human | design | med |
| D5: Exact leak-point fix chosen by Phase 0 instrumentation (three code-grounded candidates; see Phase 0) | The mechanism family is decided (D2) but the defective link must be measured, not guessed | agent | Phase 0 exit | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [ ] **D1 ratified**: pure programs in the matrix must check clean (not just diagnose better).
- [ ] **D4 ruled**: confirm that the closing rule's eligibility keeps *linked* store-only
  hand-offs (`keepAll`, `idLines`-style param→return flow) fully payable, and decide what
  happens for genuinely dead positions if the spike shows any flip there. Default per this
  doc: linked flow-through positions never close (see eligibility E2), so no store-only flip
  exists; the ruling records the spike's measurement either way.

## Solution Design

### Overview

Close the loop the F-B doctrine already opened: a tail that nothing constrains must not
outlive its declaration. Today that sentence is enforced only for lambda-opened tails
(`ctx.openedLambdaTails`). Extend the same lifecycle to **scheme-borne data-position tails** —
effect-row tails of implicit-contract arrows inside parameter/return/container types — so that
at the nested hand-off the published callback row for an argument contains only tails that
(translation-invariantly) can still be spent by someone. Concretely:

- **(C1)** At generalization, close an implicit-contract annotation arrow tail to `{}` when the
  declaration's own checking never constrained it (still unbound after the declaration's
  solve) **and** it is not owned by an enclosing scope and not an explicitly declared row
  variable. (`tot`'s never-read `upper` closes; `applyRec`'s called `f` does not; `keepAll`'s
  param→return linked tail does not; `! {e}`/`! {IO}` arrows are excluded by flag.)
- **(C2)** At publication/finalize, a tail whose only provenance is an F-B opened tail of a
  stored closure that the closing passes already resolve to `{}` must not reappear open in
  `CallbackRows` — the ownership/exclusion path that resurrects it (eight's `ρu14`) is the
  Phase 0 target.
- **(C3)** `nestedLatentRow`'s deliberate nil-on-conflict stays, but the validator's
  `missing callback row` invariant becomes a coded, actionable diagnostic for the residual
  genuine class (two *explicitly declared* distinct tails, `fwdE`).

After C1+C2, implicit-tail conflicts cannot reach the validator: every implicit arrow tail at a
charge site is either body-constrained (linked into the callee's row ⇒ shared, single tail),
caller-owned (erased), or closed. The only remaining conflicts are explicit-declared-tail
containers — exactly the class `TestLatentNested_DistinctDeclaredTailsFailLoud` pins as
intended fail-loud.

### Why the alternatives are rejected

| Alternative | Measured / reasoned outcome | Verdict |
|---|---|---|
| Multi-tail union rows (true #616 representation fix) | Even with a union row, the *caller's* required row is single-tail; a pure caller still cannot absorb ε1∪ε2. Fixes the diagnostic, not r3. Touches every `Row` consumer (iface, ceiling, smt, json encoding) | Rejected as primary; revisit only if the #616 class grows |
| L1-closed annotations (annotation-less arrows closed everywhere) | The human-gated Phase 3 of `m-effect-latent-function-values` (parent doc); changes inference of every function reference; on its own overshoots this bug by orders of magnitude | Out of scope; this doc's C1 is the *dead-position* slice of it, gated per-declaration |
| Validator-side dropping of `ε_annot*`/`ρ*` tails | Keyed on printed names — forbidden by #616's "do not equate tails by printed spelling"; leaves the inferred row wrong for iface/ceiling (rejected alternative E of v0.56.0, same reasoning) | Rejected |
| Spend-bounded charging (charge only tails free in the callee's CallRow) | Unsound for implicit-contract nested arrows: `six.ail` proves the charge at a *pure* callee is required (the callee invokes the stored callback; the supplier pays — the #1718 ruling) | Rejected |
| Diagnostic-only (keep rejects, better message) | Leaves the reporter's expressiveness loss in place; D1 rules it insufficient | Rejected as primary; C3 adopts the diagnostic for the residual class |

### Architecture

**Eligibility rule (C1), per declaration, at the same sites as `closeOpenedTailsForGeneralization`
(`internal/types/typechecker_functions.go:243`, `:404`):**

- **E1 (implicit only):** the tail belongs to an arrow with `ImplicitEffectContract: true` and
  `ConcreteEffectContract: false`. Explicit rows (`! {e}`, `! {IO}`, `! {}`) are never touched —
  this preserves `fwdE`, all S-fixtures, `ok_rowpoly`, and the concrete-contract compatibility
  tails (#1826).
- **E2 (unconstrained):** the tail is still unbound after the declaration's constraint solve —
  nothing in the body linked it (no call, no param→return flow). `keepAll`'s `fs` tail is
  linked by its body and survives; `tot`'s `upper` is unbound and closes.
- **E3 (unowned):** the tail is not owned by an enclosing scope (`env.effectRowsAbove(base)`,
  same check the F-B pass uses).
- **E4 (not lambda-opened):** lambda-opened tails keep their existing, already-landed lifecycle;
  the new pass must not double-close or fight `closeOpenedLambdaTails` (ordering: run the
  annotation-tail closing first, or merge the sets — implementer's choice, see Deferred).

The closing is recorded both as a substitution entry and a `TypeEq` constraint, exactly as
`closeOpenedTailsForGeneralization` does, so later solves agree.

**Publication repair (C2):** Phase 0 instrumentation dumps, for each repro application: the
published `CallbackRows`, `implicitOwned`, `owned`, the final `sub`, and
`publicationSubstitution`'s filtered view. The three candidate defects to discriminate
(all verified as *mechanisms present in the code*, none yet measured as *the* defect):

- **H1 — publish-time TVar blindness:** `publishApplicationWithCallee` captures
  `record.ownedTypes` from binder types that may still be unresolved type variables at publish
  time, so `implicitRowsAbove` sees no data arrows; `resolveOwnedTypes`' finalize re-anchor is
  supposed to repair this but may re-anchor into `owned` without `implicitOwned` for
  scheme-instantiated tails.
- **H2 — owned-exclusion resurrection:** `closeOpenedLambdaTails` binds the terminal tail to
  `{}` in `sub`, but `publicationSubstitution` drops bindings for names in `owned`; a tail
  marked owned (e.g. via the re-anchor) reverts to open in the published row.
- **H3 — cross-declaration opened-tail orphaning:** the stored closure's F-B tail is registered
  in the *helper's* checking context; after scheme quantification the caller-side copy is not
  in any `openedLambdaTails` set, so no closing pass ever claims it.

Whichever candidate(s) the spike confirms, the repair is confined to the closing/ownership
plumbing (`typechecker_lambda_open.go`, `typechecker_app_effects.go`); the mask, the validator
union, and the fail-loud conflict semantics do not change.

**Diagnostic (C3):** replace the `missing callback row` invariant string
(`internal/pipeline/validate_effects.go:405–407`) and the malformed-record sibling
(`internal/pipeline/validate_effects_latent.go:57`) with a coded error, `TC_LATENT_001`
(namespace verified unallocated: `grep -rn "TC_LATENT\|TC_CALLBACK" internal/ cmd/` → empty),
worded to name the construct and the fix:

```
TC_LATENT_001: this argument nests 2+ independent latent callback rows
  (e.g. fields 'lower' and 'upper' of 'Lines'); AILANG cannot charge them as one effect row.
  Fix: annotate the function-typed fields in the type declaration with a shared row
  (e.g. '[float -> float ! {e}]' and declare '! {e}'), close them ('! {}') if the
  stored functions are pure, or pass the callbacks as separate parameters.
```

### Implementation Plan

**Phase 0 — instrumented spike (~0.5 day)**
- [ ] Add a `DEBUG_APP_EFFECTS=1` dump (gated, test-only or env-gated like `DEBUG_EFFECTS`) of
  `ApplicationEffects`, `owned`, `implicitOwned`, `openedLambdaTails`, final `sub`, and the
  `publicationSubstitution`-filtered `sub` for every app in the repro matrix.
- [ ] Classify r3/ten/eleven/eight against H1/H2/H3; record which candidate(s) are the defect.
- [ ] Re-derive the expected post-fix `CallbackRows` for each repro row; write them down as the
  Phase 1 acceptance expectations.
- [ ] Exit criterion: a written table mapping every repro row to its leak path. If none of
  H1–H3 explains a row, stop and report — do not improvise a fourth mechanism.

**Phase 1 — closing + publication repair (~1 day)**
- [ ] Implement the C1 eligibility walk (data-position arrows via the existing
  `dataArrowsWith` traversal + `freeEffectRowVarsInType`; both already walk records, lists,
  tuples, aliases — verified in `internal/types/free_vars.go` and
  `typechecker_latent_data.go`) and the closing at the two generalization sites.
- [ ] Implement the C2 repair for the confirmed H-candidate(s).
- [ ] Fixtures: the full repro matrix as Go tests in `internal/pipeline`
  (r3, ten, eleven, two, three → clean; one, five, four, threed, threee → still clean;
  six, seven → still `Missing effects: IO`), plus mutation checks: revert C1 → three/two back to
  `ε_annot2`; revert C2 → ten/eleven/eight back to the leak.

**Phase 2 — diagnostic (~0.5 day)**
- [ ] `TC_LATENT_001` for the residual conflict and the malformed-record invariant; keep
  `TestLatentNested_DistinctDeclaredTailsFailLoud` green against the new wording (its
  assertion must be updated from `missing callback row` to the code — the test currently
  asserts the internal-invariant string at `effect_latent_nested_test.go:195`; read and cited).
- [ ] Sweep `internal/pipeline` tests for other assertions on the two invariant strings;
  update each with intent comments.

**Phase 3 — corpus, docs (~0.5–1 day)**
- [ ] Base-vs-fixed sweep over all tracked `.ail` files plus the external corpora used by the
  v0.56.0 doc (motoko, sunholo packages; `AILANG_NO_CACHE=1`, fresh copies); diff per-file
  status + first diagnostic. Any flip not in "What deliberately changes" blocks completion.
- [ ] `docs/LIMITATIONS.md`: add a Resolved entry for the pure-container class and document
  the residual explicit-tail conflict with the `TC_LATENT_001` migration note.
- [ ] Changelog fragment `changelogs/unreleased/…-dead-annotation-arrow-tails.md`.
- [ ] `examples/runnable/` example: the r3 shape (pure band arithmetic) as the accept arm;
  reject arms stay in Go tests.

### Files to Modify/Create

**Modified files:**
- `internal/types/typechecker_lambda_open.go` — the C1 closing pass + shared helpers (~60 LOC)
- `internal/types/typechecker_functions.go` — call sites at `:243`/`:404` (~10 LOC)
- `internal/types/typechecker_app_effects.go` — C2 publication repair (site depends on Phase 0; ~20–40 LOC)
- `internal/pipeline/validate_effects.go` + `validate_effects_latent.go` — C3 coded diagnostic (~30 LOC)
- `internal/pipeline/effect_latent_nested_test.go` — `fwdE` assertion wording + new fixtures (~120 LOC)

**New files:**
- `internal/pipeline/effect_scheme_borne_tails_test.go` — the repro matrix as table-driven
  cases (~200 LOC)

## Conflict Surface

This change touches `internal/types` (generalization-time closing, publication) and
`internal/pipeline` (a diagnostic string). It changes **no parser surface and no syntax**; the
conflict surface is semantic positions:

1. **Positions extended:** effect-row lifecycle of implicit-contract arrows inside type
   annotations (params, returns, type-decl aliases, container fields) at generalization time.
2. **Other constructs living in those positions:**
   - Explicit arrow rows (`! {e}`, `! {IO}`, `! {}`): excluded by E1; their compatibility tails
     (#1826) keep the `latentRow`/`applicationCallRow` closing.
   - Lambda-opened tails (F-B): keep their existing pass; E4 prevents interference.
   - Row-polymorphic declarations (`applyFstE[e]`, `myMapE[a,b,f:(a)->b ! {e}]`): tails are
     explicit → excluded by E1.
   - Unannotated *inferred* param rows (fresh vars, not `ε_annot`): already subject to
     existing defaulting; untouched.
3. **Disambiguation:** by type flags (`ImplicitEffectContract`/`ConcreteEffectContract`) and
   bindedness after the declaration solve — no source-syntax discrimination at all.
4. **Programs that MUST still work (fixtures, all read this session):**
   - `TestLatentNested_LeaksRejected` R1–R10 (`internal/pipeline/effect_latent_nested_test.go:93`) —
     every `sneaky` must still fail with `Missing effects: IO`.
   - `TestLatentNested_AnnotatedGenericContainers` R10/R11 definitions and uses (`:113`).
   - `TestLatentNested_ConcreteTailsClosed` S1–S5 incl. the `mcp_oauth revokeFamily` record-of-hooks (`:126`).
   - `TestLatentNested_Controls`: `ok_pure_cb`, `ok_storage` (incl. `pure func mkList() -> [(int) -> int] = [logIt]` —
     construct-only stays accepted), `ok_rowpoly`, `fwd_two`, `alias_two_callers`, `x_main_ok` (`:142`).
   - `TestLatentNested_IntendedFlips`: `store_only_nested_handoff` (store-only hand-off still
     charges a pure caller), `store_only_concrete_annotation_ok`, `reg_construct`,
     `map_concrete_callback_is_type_error` (`:171`).
   - `TestLatentNested_DistinctDeclaredTailsFailLoud` (`:191`): `fwdE` keeps failing, now with `TC_LATENT_001`.
   - `TestLatentNested_RunRejectsBeforeExecution` (`:204`): tuple/list/record rejects on both engines.
   - `six.ail`/`seven.ail` from this session's matrix (stored `! {IO}` callback → `Missing effects: IO`).
5. **What deliberately changes:**
   - r3/ten/eleven/two/three-class pure programs: rejected (invariant/unresolved tail) → clean.
   - `fwdE`-class conflicts: `internal invariant: missing callback row` → `TC_LATENT_001`.
   - Anything the Phase 3 sweep measures as a flip in *dead-position* schemes (D4) — expected
     zero given E2, but measured, not assumed.

## Examples

### Example 1: the reported program (r3) — Before / After

**Before (v0.60.2):**
```
Error: … function run: internal invariant: missing callback row for typed application 67 at r3.ail:10:17 (argument 0)
```

**After:** `✓ No errors found!` — `total`'s never-read `upper` tail closed at `total`'s
generalization; `withBand`'s linked tails resolve through `found`'s pure opened tail; the
charge row at the nested hand-off is `{}`.

### Example 2: what stays rejected

```ailang
func logIt(x: float) -> float ! {IO} = { let _ = println("boom"); x }
export pure func run() -> float = tot({lower:[logIt], upper:[], empty:true}, 1.0)
```
Before and after: `Effect checking failed for function 'run' … Missing effects: IO`. The
supplier-pays ruling of #1718 is untouched.

### Example 3: the residual conflict, now legible (fwdE class)

**Before:** `internal invariant: missing callback row for typed application …`
**After:** `TC_LATENT_001: this argument nests 2+ independent latent callback rows …` with the
three fixes spelled out.

## Success Criteria

- [ ] r3, ten, eleven, two, three check clean (`internal/pipeline/effect_scheme_borne_tails_test.go`, accept arms)
- [ ] six, seven still rejected with `Missing effects: IO` (same file, reject arms)
- [ ] All pre-existing nested-callback fixtures byte-identical verdicts (Conflicts Surface §4 list; `make test-core` green)
- [ ] `TC_LATENT_001` fires for `fwdE`; no test or user-facing path prints `missing callback row` as an internal invariant
- [ ] `make test-core`, `make verify-examples`, `tools/verify-stdlib.sh` green; `make check-boundaries` green (no new import directions)
- [ ] Corpus sweep: zero unplanned flips; measured flips recorded in the sprint report
- [ ] `docs/LIMITATIONS.md` and changelog updated

## Testing Strategy

**Unit tests:** C1 eligibility (each E1–E4 condition exercised with a minimal declaration:
closed, linked, owned, explicit); C2 publication (assert the published `CallbackRows` for each
repro app against the Phase 0 expectations); C3 message format and code allocation.

**Integration tests:** the repro matrix as `checkModules` table tests
(harness: `internal/pipeline/effect_latent_nested_test.go`'s `runNestedCases` pattern); engine
parity via `TestLatentNested_RunRejectsBeforeExecution`'s binary-invocation pattern for the
six/seven reject arms (evaluator + `--bytecode`).

**Manual testing:** none beyond the corpus sweep; `DEBUG_APP_EFFECTS=1` dumps reviewed during
Phase 0 and attached to the sprint report.

## Deferred Decisions

The following are intentionally left open for the implementer:

- Whether C1 merges into `closeOpenedTailsForGeneralization`'s loop or runs as a sibling pass
  first — agent may choose, but the mutation checks must still discriminate C1 from C2.
- The `DEBUG_APP_EFFECTS` dump's exact format and whether it ships env-gated or test-only —
  agent may choose (precedent: `DEBUG_EFFECTS` in `internal/pipeline/validate_effects.go:11`).
- The precise wording of `TC_LATENT_001` beyond the required content (container type, the
  conflicting positions, the three fixes) — agent may choose.
- If Phase 0 confirms more than one of H1–H3, the repair order — agent may choose, provided each
  fix carries its own mutation check.

## Non-Goals

- **Multi-tail union rows (#616 proper)** — representation change across the checker; does not
  fix pure callers by itself (see Alternatives).
- **L1-closed annotations globally** — human-gated Phase 3 of the parent doc; this doc takes
  only the dead-position slice, per-declaration.
- **Width subsumption for closed function values (RC4/Phase 2)** — separate deferred spike per
  `m-effect-row-open-closed-soundness.md`.
- **Any change to the supplier-pays charge at pure callees** — six/seven prove it sound; it stays.
- **Reprobing `map`/`mapE` semantics** — frozen by the v0.56.0 doc's intended flips.

## Timeline

**Week 1** (~2 days): Phase 0 spike + written leak-path table; Phase 1 closing/repair + fixtures.
**Week 2** (~1 day): Phase 2 diagnostic + test-wording updates; corpus sweep starts.
**Week 3** (~0.5–1 day): corpus triage, docs, changelog, example; release notes.

**Total: ~3.5 days across 3 weeks** (2× the honest estimate of ~1.75, per the skill's rule).

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Closing reaches a *linked* position in real code (store-only hand-off flips to type error) | Med | E2 requires still-unbound after the declaration solve; `keepAll`/`idLines` fixtures + the corpus sweep; D4 freeze ruling records the measurement |
| The H1–H3 spike is inconclusive (a fourth mechanism) | Med | Phase 0 exit criterion forbids improvising; escalate with the dump table instead |
| Corpus flips in packages that rely on open implicit tails in dead positions | Low–Med | Same measurement posture as v0.56.0: per-file status+first-diagnostic diff; any flip needs a changelog entry and, if semantic, a freeze ruling |
| Double-closing fights the landed F-B pass (regression of #1958 re-land fixtures) | Low | E4 ordering + the existing `cmd/ailang/effect_row_soundness_test.go` deferred cases + `make test-core` |
| Diagnostic code drift (TC_LATENT_001 collision later) | Low | Allocation grep recorded here; add to the error-code list doc at implementation time |

## Related Documents

**Implemented (may inform design):**
- [m-effect-latent-nested-callbacks](../implemented/v0_56_0/m-effect-latent-nested-callbacks.md) —
  created the nested hand-off charge and predicted this residual risk verbatim (risk table,
  "nestedLatentRow unions two distinct unresolved tails"); its fixtures are this doc's
  regression gates.
- [m-effect-row-open-closed-soundness](../implemented/v0_56_0/m-effect-row-open-closed-soundness.md) —
  landed the F-B closing family this doc extends; owns the "tail that nothing constrains must
  not outlive its declaration" doctrine.
- [m-effect-latent-function-values](../v0_48_0/m-effect-latent-function-values.md) —
  parent doc; L1/L2 rules and the human-gated Phase 3 this doc deliberately does not take.

**Planned (check for overlap):** neural/SimHash search found no planned doc on this topic
(search run by the create script, 2026-11-12; the two closest implemented docs are linked above
and cover the *charging* mechanism, not the *closing* gap).

## References

- Issue: reported against v0.56.1 (`05c860a`), sunholo/worlds `selection.ail`; reproduced on
  dev v0.60.2 (`33f3009`) this session.
- [#616](https://github.com/sunholo-data/ailang/issues/616) — per-App publication record and
  the explicit-conflict invariant.
- [#1718](https://github.com/sunholo-data/ailang/issues/1718),
  [#1826](https://github.com/sunholo-data/ailang/issues/1826) — the nested hand-off soundness
  rules this doc must preserve.
- [#1958](https://github.com/sunholo-data/ailang/issues/1958) — F-B open-on-use / close-on-generalize.
- [Design Axioms](/docs/references/axioms) — scoring above.

## Verification Log

| Claim | Evidence (command / file:line, captured this session) |
|---|---|
| r3 fails with `missing callback row` on dev v0.60.2 | `ailang check /tmp/repro/r3.ail` → exact message, app 67 at 10:17, argument 0 |
| Single-field scheme-borne pure closure fails (ten/eleven) | `ailang check ten.ail eleven.ail` → `Unresolved effect tail: ε_annot1`, both, incl. the constant `\v. v*2.0` closure |
| Locally-built containers pass (one/five) | `ailang check one.ail five.ail` → `✓ No errors found!` |
| `! {}` arrow annotations are a working user workaround | `ailang check four.ail` → clean |
| Closing the dead field alone moves, not fixes, the leak | `ailang check eight.ail` → `Unresolved effect tail: ρu14` |
| Effectful stored callbacks are correctly charged at pure callees (fix must not touch this) | `ailang check six.ail seven.ail` → `Missing effects: IO` |
| Row-sharing callees absorb correctly (no over-charge when the callee spends the tail) | `ailang check threed.ail threee.ail` → clean |
| The callee-never-read field is over-charged (three) | `ailang check three.ail` → `Unresolved effect tail: ε_annot2` |
| Annotation-less `FuncType` mints open `ε_annot<N>`, `ImplicitEffectContract: true` | `internal/elaborate/file_funcs.go:405–432` (read) |
| Only `openLambdaRow` registers `openedLambdaTails`; no closing pass sees `ε_annot` tails | `grep -rn "openedLambdaTails" internal/types/*.go` → `inference.go:58,60`, `typechecker_lambda_open.go:55–58` only (negative-existence verified) |
| `nestedLatentRow` returns nil on two distinct surviving tails (no first-wins) | `internal/types/typechecker_latent_data.go:126–152` (read); `CheckedUnionEffectRows` conflict |
| Validator invariant site | `internal/pipeline/validate_effects.go:405–407` (read) |
| `fwdE` two-explicit-tails conflict is pinned as intended fail-loud | `internal/pipeline/effect_latent_nested_test.go:191–200` (read; asserts `missing callback row`) |
| All Conflict-Surface fixtures exist and assert what §4 says | `internal/pipeline/effect_latent_nested_test.go:93–207` (read in full this session) |
| `TC_LATENT_001`/`TC_CALLBACK_*` unallocated | `grep -rhn "TC_[A-Z_]*[0-9]" internal/ cmd/ --include=*.go -o \| sort -u` → 12 codes, none in this family |
| `make test-core`, `make verify-examples`, `make check-boundaries` exist | `make/test.mk:446`, `make/examples.mk:18`, `make/ci.mk:11` (grep) |
| `freeEffectRowVarsInType` and `dataArrowsWith` already walk all data positions | `internal/types/free_vars.go` (varWalker, read), `internal/types/typechecker_latent_data.go:27` (read) |
| Which of H1–H3 is the actual ten/eleven/eight leak | **Unverified** — requires a debug build (no Go toolchain in this session's sandbox); Phase 0 exists to settle exactly this, with a no-improvisation exit criterion |

## Appendix A — Repro sources (verbatim, session-verified 2026-11-12)

Every file below was checked with `ailang check` on dev v0.60.2 (`AILANG_NO_CACHE=1`) in this
session; expected outputs are in the matrix above. `r3` is the Issue text verbatim.

```ailang
-- r3.ail — reported repro (fails: internal invariant: missing callback row)
module r3
type Lines = {lower:[float -> float],upper:[float -> float],empty:bool}
pure func noLines() -> Lines = {lower:[],upper:[],empty:false}
pure func withBand(acc:Lines,lo:float,c:float -> float) -> Lines = {acc|lower:acc.lower++[\v. lo-c(v)]}
pure func total(ln:Lines,v:float) -> float = match ln.lower { [] => 0.0, f::rest => f(v) }
pure func twice(rule:int -> float) -> float = rule(1)+rule(2)
export pure func run(x:float) -> float {
 let found=\v. v+x;
 let ln=withBand(noLines(),1.0,found);
 twice(\n. total(ln,1.0))
}
```

```ailang
-- ten.ail — single function field; pure closure through a helper's scheme
--         (fails: Unresolved effect tail: ε_annot1)
module ten
type L1 = {lower:[float -> float],empty:bool}
pure func withOne(acc:L1,lo:float,c:float -> float) -> L1 = {acc|lower:acc.lower++[\v. lo-c(v)]}
pure func total(ln:L1,v:float) -> float = match ln.lower { [] => 0.0, f::rest => f(v) }
export pure func run(x:float) -> float {
 let ln=withOne({lower:[],empty:true},1.0,\v. v+x);
 total(ln,1.0)
}
```

```ailang
-- eleven.ail — as ten, stored closure is a constant pure function
--           (fails: Unresolved effect tail: ε_annot1)
module eleven
type L1 = {lower:[float -> float],empty:bool}
pure func withOne(acc:L1,lo:float,c:float -> float) -> L1 = {acc|lower:acc.lower++[\v. v*2.0]}
pure func total(ln:L1,v:float) -> float = match ln.lower { [] => 0.0, f::rest => f(v) }
export pure func run(x:float) -> float {
 let ln=withOne({lower:[],empty:true},1.0,\v. v+x);
 total(ln,1.0)
}
```

```ailang
-- three.ail — callee not declared pure; the field it never reads is over-charged
--           (fails: Unresolved effect tail: ε_annot2)
module three
type L2 = {lower:[float -> float],upper:[float -> float],empty:bool}
pure func mk() -> L2 = {lower:[],upper:[],empty:false}
func tot(l:L2,v:float) -> float = match l.lower { [] => 0.0, f::rest => f(v) }
export pure func run(x:float) -> float {
 let l=mk();
 tot({lower:[\v. v+x],upper:[],empty:true},1.0)
}
```

```ailang
-- six.ail — soundness guard: effectful stored callback must STAY rejected
--          (fails today and after the fix: Missing effects: IO)
module six
import std/io (println)
type L1 = {lower:[float -> float],empty:bool}
pure func tot(l:L1,v:float) -> float = match l.lower { [] => 0.0, f::rest => f(v) }
func logIt(x:float) -> float ! {IO} = { let _ = println("boom"); x }
export pure func run() -> float = tot({lower:[logIt],empty:true},1.0)
```

```ailang
-- one.ail — locally-built container: passes today, must pass after
module one
type L1 = {lower:[float -> float],empty:bool}
pure func mk() -> L1 = {lower:[],empty:false}
pure func tot(l:L1,v:float) -> float = match l.lower { [] => 0.0, f::rest => f(v) }
export pure func run(x:float) -> float {
 let l=mk();
 let f=\v. v+x;
 tot({lower:[f],empty:true},1.0)
}
```

```ailang
-- four.ail — the existing user workaround: explicit ! {} arrow rows in the type decl
module four
type L2 = {lower:[float -> float ! {}],upper:[float -> float ! {}],empty:bool}
pure func mk() -> L2 = {lower:[],upper:[],empty:false}
pure func tot(l:L2,v:float) -> float = match l.lower { [] => 0.0, f::rest => f(v) }
export pure func run(x:float) -> float {
 let l=mk();
 tot({lower:[\v. v+x],upper:[],empty:true},1.0)
}
```

```ailang
-- eight.ail — r3 with only the dead field closed: the live field's tail leaks alone
--           (fails: Unresolved effect tail: ρu14) — the C2 publication-repair probe
module eight
type L2b = {lower:[float -> float],upper:[float -> float ! {}],empty:bool}
pure func noLines() -> L2b = {lower:[],upper:[],empty:false}
pure func withBand(acc:L2b,lo:float,c:float -> float) -> L2b = {acc|lower:acc.lower++[\v. lo-c(v)]}
pure func total(ln:L2b,v:float) -> float = match ln.lower { [] => 0.0, f::rest => f(v) }
pure func twice(rule:int -> float) -> float = rule(1)+rule(2)
export pure func run(x:float) -> float {
 let found=\v. v+x;
 let ln=withBand(noLines(),1.0,found);
 twice(\n. total(ln,1.0))
}
```

The remaining matrix rows (two, five, seven, threed, threee) are one-line mutations of the
above: two = three with a pure callee and the record literal inline; five = three with
`tot(l,1.0)`; seven = six with the `upper` field added; threed/threee = threeb/threec with
`export pure func run(l:L2) -> float = tot(l,1.0)` (callee row shares the tails).

## Future Work

- If D4's measurement shows dead-position flips matter in practice, propose the full L1-closed
  annotation semantics (parent doc Phase 3) as its own human-gated doc.
- Multi-tail union rows (#616 proper) if the explicit-tail container class grows beyond `fwdE`
  into real package code.
- Fold `DEBUG_APP_EFFECTS` into `ailang check --debug-effects` if a second consumer appears
  (friction-twice rule).

---

**Document created**: 2026-11-12
**Last updated**: 2026-11-12
