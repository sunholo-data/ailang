# M-EFFECT-ROW-OPEN-CLOSED-SOUNDNESS — An open row must not lose its labels when it meets a closed row

**Status**: M2 (`830934d23`, the open-vs-closed check in `RowUnifier.UnifyRows`) was **reverted in v0.56.1** (v0.56.0 was tagged but never published), then **RE-LANDED on dev 2026-11-02** with the inference fixes F-A1, F-A2 and F-B (`ad9aabc57`, `ef01e499f`, plus the perf follow-up). It is unreleased and targets v0.57.0. See "Re-land implementation report (2026-11-02)" at the end. The explicit-empty-callback contract (`8fd84a200`) is unaffected and still ships.
**Target**: v0.56.0 (next minor after the v0.55.0 tag). This is a soundness fix that rejects programs which check today, so it takes a minor bump. That is the posture Mark ruled for M-EFFECT-LATENT-FUNCTION-VALUES on 2026-10-08.
**Priority**: P0 (High). A `pure` function performs IO, `ailang check` reports it clean, and the side effect runs on both engines.
**Tracks**: #1958
**Refs**: #1718 and #1826 (sibling doc, see "Ordering and non-overlap"), #573 and #1326 (the L1-open and L2 rules this builds on), #1443 / PR #1790 (ordering only)
**Estimated**: 2–3 days. Phase 0 ≈ 0.5 day, Phase 1 ≈ 1 day, corpus, goldens and docs ≈ 0.5–1 day. Phase 2 is a separate spike that is not part of this estimate.
**Dependencies**: None blocking. Inside this doc, Phase 0 must land before Phase 1 or in the same commit. It does not depend on the #1718/#1826 work.
**Base measured**: `origin/dev` `7d29031ea`. Every binary was built from a `git archive 7d29031ea` copy in scratch (`go build ./cmd/ailang`) and checked by behaviour, not by its version stamp. All checks ran with `AILANG_NO_CACHE=1`, with `AILANG_STDLIB_PATH` pointed at the matching copy's `std/`.
**Quorum trigger**: #1 fires (there are design-freeze items). The requester is handling review, so no quorum was run in this session.


## Revert note and re-land brief (2026-11-02)

M2 (`830934d23`) broke the executor MCP (`executor_tools/executor.ail:191`), so every dev-plane
deploy failed from 2026-10-29. v0.56.0 was tagged with it and never published; v0.56.1 reverted it
(`511cb476e`, `05c860aaf`). M2's guard is correct. What it exposed are **two separate inference
defects that it depended on**, plus a test-scope gap that let it ship. This brief was researched
by spike at `05c860aaf` (M2 restored on top). Every number below was measured there. Nothing in it
is implemented.

### Why M2 failed: two causes, not one

Failing shapes, all accepted by v0.55.0 and rejected with M2 (`row {IO, ...ε6} carries labels [IO]
which the closed row {} does not permit`):

```ailang
func withDir(f: string -> int ! {IO}) -> int ! {IO} = f("d")
withDir(\d. ex(p(), 1))                         -- A: effectful argument (ANF let)
withDir(\d. if d == "x" then q(1) else q(2))    -- A: branches
each(\x. { println(x); println(x) }, xs)        -- A: sequence, slot `string -> () ! {IO}`
withDir(\d. 1)                                  -- B: pure lambda into a wider slot
```

**Cause A, the union drop.** When an effect union is built, a single call's row is still a bare
tail, because `inferApp` deliberately leaves the callee row unsubstituted
(`typechecker_functions.go`, the "intentionally NOT eagerly substituted" note). Two of those meet
in a sequence, let, or branch as `{...ε9} ∪ {...ε11}` (instrumented). `combineTails`
(`typechecker_data.go`) returns nil for distinct tails, so the lambda's row becomes **closed `{}`
with no labels**. The `combineEffects` doc comment says this case is "a loud invariant failure";
the code silently closes it.

**Cause B, width.** `\d. 1` has no union. Its row is legitimately closed `{}`, and the slot is
`{IO | ε_annot}`. This is the RC4 width class that F2 ruled to *reject*, with "0 corpus files"
as the premise. That premise was wrong, and the revert's must-accept test contradicts F2. The
recommendation below reopens F2 for lambdas. "Keep multi-effect rows open" cannot fix cause B.

**The test-scope gap.** AC6 swept only `examples/**` and `std/**` (515 files). `executor_tools/`
is outside that scope. Over all 810 tracked `.ail` files, M2 on its own flips exactly **one**:
`executor.ail:191`.

**What cause A already gets wrong without M2.** On `05c860aaf`, an IO lambda passed to a
**pure** `map`/`any` slot is rejected when the body makes one call, but accepted when it makes
two:

| probe (all in a `! {IO}` caller) | HEAD | after fix |
|---|---|---|
| `map(\x. ex(x), xs)` · `map(ex, xs)` · `any(\x. ex(x) > 0, xs)` | rejected | rejected |
| `map(\x. { println(show(x)); x }, xs)` · `map(\x. { ex(x); ex(x) }, xs)` · `map(\x. if … then ex(x) else ex(0), xs)` · `any(\x. { println(show(x)); true }, xs)` | **accepted** | rejected |

The "designed `any(\x. { println(…); true })` acceptance" cited during the release is not a
design rule. AC5 pinned it only as "must stay as on base". The language rule is that `map` and
`any` are pure, and `mapE` and its siblings are the effectful forms (V8, V25, `ailang prompt`).
The post-check validator still reports `Missing effects: IO` in a `pure` caller, so this is not a
live leak. It is an inconsistency that the fix removes.

### The fix: three inference changes, then restore M2

**F-A1: resolve union operands before combining.** In the distinct-tail branch of
`combineEffects`, solve the equality constraints added since the enclosing lambda began
(`solveLocalEqualities(checkpoint)`, already used by #386), and substitute each operand. Two
rules, both copied from `inferApp`, are load-bearing:
- drop bindings for the enclosing scope's variables (`ctx.env` free type, row and effect-row
  vars at lambda entry);
- use `closeIfResolved` semantics: only substitute an operand that resolves fully closed.

`{...ε9} ∪ {...ε11}` then becomes `{IO} ∪ {IO}`. In production, `combineEffects` becomes a method
that receives `ctx` plus a lambda checkpoint stack. The spike used a package-level hook, which
must not ship. Cost: one local solve per distinct-tail union, over that lambda's constraints only.
Measure it with `make test-core` timing and the `perf-sweep` skill.

**F-A2: fixpoint absorption for recursive self-calls.** F-A1 on its own regresses 2 corpus files
(`examples/runnable/debug_effect.ail` and `internal/embed/testdata/debug_concurrent.ail`, `same
row variable with different extensions`) and misdiagnoses `TestRecursiveConditionalStillRejectsRealEffect`.
Once `Debug.log`'s operand resolves, the union keeps the self-call tail, and the let-rec
constraint becomes `ρ ~ {Debug | ρ}`. HEAD avoided this only because the drop closed the row.

The rule: during F-A1, an operand whose resolved tail is the effect row of a let-rec binding
currently being inferred contributes **only its labels**, because `ρ = L ∪ ρ` has the least
solution `ρ = L`. Absorbing only at the let-rec boundary is not enough: six self-calls in nested
branches meet in unions first. The spike keeps a stack of the active let-rec binding types (the
`bindingTypes` built in `inferLetRec`). Absorb only the self row. A callback parameter's `! {e}`
is also an outer variable, and dropping it would be unsound.

**F-B: lambda open-on-use (Phase 2a).** If an unannotated lambda's latent row is closed after
inference, give it a fresh tail: `{L}` becomes `{L | ρ}`. Before both
`applyApplicationSubstitution(sub)` calls (`typechecker_core.go`), bind the terminal tail of every
still-unbound lambda-opened row to closed `{}`. V16's naive lambda opening regressed 16 files on
the `ApplicationEffects` invariant; the closing pass is the difference. In production, the opened
set is per declaration (on `ctx`), not the spike's process-global map, and it is registered
through the ownership model (`closingRows`, `owned` in `typechecker_app_effects.go`). Opened tails
must not appear in diagnostics: the spike printed `Eq[[int -> int ! {...ρ6}]]`.

**Then restore M2:**
- re-apply `830934d23`'s two guards;
- recover the unit matrix `internal/types/row_unification_open_closed_test.go` from `830934d23`
  (the revert deleted it; it was not skipped);
- unskip `TestEffectRowOpenClosedM2Deferred` and the deferred cases in
  `cmd/ailang/effect_row_soundness_test.go`;
- restore the two `EffectCeiling` subtests to their original `\c. 1` spelling (F-B makes the F2
  re-pin unnecessary).

### Measured on the spike (base `05c860aaf`)

| arm | must-accept guard (11 cases) | 810 tracked `.ail` | 256 external (`ailang-packages` `e998928`, `ailang-parse` `bf12238`) | Go `./internal/... ./cmd/...` |
|---|---|---|---|---|
| M2 only | 4 red (A×3, B) | 1 flip (`executor.ail`) | — | — |
| M2 + F-A1 + F-A2 | 2 red: B, and the intended `any` flip | **0 flips** | — | 146 ok; only `internal/pipeline` red, on exactly those 2 |
| M2 + F-A1 + F-A2 + F-B | 1 red: the intended `any` flip | **0 flips** | **0 flips** | 148 ok; see below |

All R1–R5 leak probes are still rejected. The `\c. 1` `EffectCeiling` originals pass. On HEAD,
595 of 810 tracked files pass `check` (the rest fail on both arms), and 249 of 256 external files.

Go reds with F-B still to resolve, none of them wrong-accepts:
- `TestLatentNested_IntendedFlips/map_concrete_callback_is_type_error`: same rejection, M2's
  wording. Update the assertion.
- `TestEqContainersNegatives/fn_elements`: the opened tail is printed in the message. Fix by
  hiding implicit tails.
- `TestLatentNested_DistinctDeclaredTailsFailLoud` and `TestPkgConfinementMatrix/pkg_tree`: pass
  alone and fail in the full run. This fits the spike's process-global tail set; re-check once it
  is per declaration.
- `TestGitProbes*`: environmental. A scratch copy is not a git checkout, and S2 fails the same way.

### Decisions for this re-land

**D-R1 and D-R2 ruled 2026-11-02 (Mark, attended): accepted as recommended** ("ok can you sprint
plan and execute", in reply to both recommendations). Plan:
`design_docs/implemented/v0_57_0/m-effect-row-open-closed-reland-sprint-plan.md`.

- **D-R1 (human):** reopen F2 for **lambdas** and adopt F-B, which restores the original
  `\c. 1` spelling. Recommended: the shape broke production, and F2's "0 corpus files" premise
  was measured on the narrow scope.
- **D-R2 (human):** the "any IO lambda in IO caller" case moves from must-accept to must-reject,
  with migration to `mapE`/`filterE`/`foldlE`/`forEachE`. There is no `anyE` today, so the
  executor should either add one or document `foldlE` as the migration. Recommended: this is the
  existing language rule, and 0 of 1,066 measured files depend on the old behaviour.
- **Still deferred: Phase 2b, width for *named* and *eta* function values.** `runWide(loud, "x")`
  and `runWide(\s. loud(s), "x")` into an `{IO, FS}` slot stay rejected. #1826 closes compatibility
  tails when a binding is read as a value, and an eta body's row closes only after the solve.
  Fixing either means open-on-use at instantiation, which conflicts with #1826 and needs its own
  design. 0 of 1,066 files are in this class. F2 stands for it.

### Tried and rejected (do not repeat)

- Substituting the resolved **callee** row in `inferApp`. #386 already recorded that this
  regresses recursive multi-arm std functions (std/sem), and it was rejected again in the release.
  F-A1 resolves union *operands* inside the lambda instead.
- Keeping one open tail in `combineTails`: it keeps the wrong one (a pure `show` tail closes to
  `{}`). No single surviving tail is correct while the operands are still bare tails.
- Exempting by unification direction: M2's unit tests require rejection in both orientations,
  and it is alternative G's second code path.
- F-A1 without enclosing-variable filtering or without F-A2: 2 corpus flips and a misdiagnosis.
- Unifying sequential effects Koka-style: a closed pure callee row (`{}`) then meets `{IO | ρ}`.
  That only works if *every* function instantiation is opened, which is Phase 2b.

### Re-land gates

- The sweep covers **all** `git ls-files '*.ail'` plus `ailang-packages` and `ailang-parse` from
  their package roots, and includes a known-flip positive control (`executor.ail` with M2 alone).
  macOS has no `timeout` binary: a sweep that uses it reports 0 flips because every check exits
  127. Use `perl -e 'alarm 90; exec @ARGV'`.
- Go: all of `./internal/... ./cmd/...` green, except reds named with their reason. Run the
  mutation checks from AC8 again.
- Perf: compare F-A1's local solves against `make test-core` time and a `perf-sweep` snapshot.
- Spike code (session scratch, may not persist):
  `/private/tmp/claude-501/-Users-voightkampff-dev-sunholo-data-ailang/ffdb5277-e2fb-47b1-b7bb-6f45be70771e/scratchpad/spike-s3.patch`
  (diff against `05c860aaf`). The sweep scripts `cmp.sh` and `extcmp.sh` are in the same
  directory. The spike is evidence, not a starting point: it uses package globals.

## Summary

`RowUnifier.UnifyRows` (`internal/types/row_unification.go:97-121`) unifies an open row with a closed row by binding the open tail to the closed side's extra labels. It never checks the open side's own extra labels. So `{IO | ε} ~ {}` succeeds with `ε := {}` and `IO` disappears. The guard used to exist on the wrong side. It was deleted rather than moved in the v0.3.11 fix (`7093db46b`, 2025-10-16, see RC1).

The leak became easy to reach in v0.53, when L1-open began elaborating every concrete annotation `! {IO}` as `{IO | ε_annotN}`. Such a row meets a closed callback slot whenever an annotated function value is passed to a std HOF whose callback is pure, such as `std/list.map`, `filter`, `foldl` or `any`. It is the same when the annotation is on a record field, or when the row mixes a label and a row variable (`! {IO, e}`).

**Fix (two parts that must ship together, plus one deferred part):**

- **Phase 0, migration.** The one place the corpus relied on the leak is a real latent bug. `std/ai.stepWithStream` and `stepWithStreamRecorded` declare `! {AI}`, yet invoke an `! {IO}` callback through builtins whose Go types close the callback row. The builtins' own comment says the row is left open. Make the two builtins and the two std wrappers effect-polymorphic (`! {e}` → `! {AI, e}`).
- **Phase 1, the fix.** Restore the Rémy side condition in both open/closed branches of `UnifyRows`. The open side's extra labels must be absent; otherwise the unification fails. The error names those labels.
- **Phase 2, deferred.** Width subsumption, meaning a function value whose row is closed and narrower than a wider, concretely annotated slot. Today this works only through the same permissive branch. It needs the Koka "open on use" rule, which is the parent doc's deferred Phase 3. Two pinned Go subtests and zero corpus files depend on it.

**Measured:** Phase 1 alone flips 17/515 in-repo files and 18 external files, all through one root (`std/ai.ail:303`), and all of them are true positives. Phase 0 plus Phase 1 flips **0/515** in-repo files, **0/165** files in 45 `ailang-packages` roots and **0/181** `ailang-parse` files. The first diagnostic of every file is byte-identical. In the Go suites, 2 subtests of `TestEffectCeiling_OwnTypesAndCallbacksNotCharged` go red. Both are false positives of the width class and both get re-pinned (Design Freeze F2).

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No runtime change; checking stays deterministic |
| A2: Replayability | 0 | No trace or format change |
| A3: Effect Legibility | +1 | A declared row (incl. `pure` and `! {AI}`) becomes true again: an `{IO}` value can no longer pass through a pure slot (V2–V5), and `std/ai.stepWithStream` stops hiding its callback's IO (V9) |
| A4: Explicit Authority | +1 | Closes a static ambient-IO hole; runtime caps stay the backstop |
| A5: Bounded Verification | +1 | Restores the standard row-unification side condition and aligns the two row unifiers in `internal/types` (RC5); the rule is local and checkable |
| A6: Safe Concurrency | 0 | No concurrency change |
| A7: Machines First | +1 | One answer for `map(g, xs)` whether `g` is a lambda, a name, an annotated parameter, a record field or a mixed-row value (today 3 of 5 leak); the diagnostic names the dropped label |
| A8: Minimal Syntax | 0 | No syntax change |
| A9: Cost Visibility | 0 | No cost surface |
| A10: Composability | +1 | `stepWithStream` accepts pure callbacks it wrongly rejects today (V10); effect-polymorphic wrappers compose |
| A11: Structured Failure | 0 | Still a type error. The text improves, but this is not a new structured code |
| A12: System Boundary | 0 | No boundary change |

**Net Score: +5** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): no implicit nondeterminism introduced
- [x] A3 (Effects): removes a hidden side effect; adds none
- [x] A4 (Authority): removes ambient access; grants none
- [x] A7 (Machines First): a uniform algebraic rule, no special case

## Problem Statement

### Reproduction (base `7d29031ea`)

```ailang
module t/rowleak

import std/list (map)
import std/io (println)

pure func viaMapPure(g: (string) -> bool ! {IO}, xs: [string]) -> [bool] = map(g, xs)

func loud(s: string) -> bool ! {IO} {
  println("side effect: ${s}");
  true
}

export func main() -> () ! {IO} {
  let r = viaMapPure(loud, ["a"]);
  ()
}
```

```
$ ailang check t/rowleak.ail                                  → ✓ No errors found!
$ ailang run --caps IO --entry main t/rowleak.ail              → side effect: a
$ ailang run --bytecode --caps IO --entry main t/rowleak.ail   → side effect: a
```

**Control (correctly rejected):** if the callback is called directly, `pure func direct(g: (string) -> bool ! {IO}, s: string) -> bool = g(s)`, the check fails with `Missing effects: IO` (V6).

### The same defect, other spellings (all `check` clean and all print on base)

| # | Shape | Base | Phase 0+1 |
|---|---|---|---|
| R1 | annotated parameter → `std/list.map` (above) | ✓ leaks | type error |
| R2 | annotated parameter → `filter` / `foldl` / `any` | ✓ leaks (all three) | type error |
| R3 | **record field** `type H = { f: (string) -> bool ! {IO} }`, `map(h.f, xs)` in a `pure` func | ✓ leaks (`run` prints) | type error |
| R4 | **mixed row variable** `pure func leakMixed[e](g: (string) -> bool ! {IO, e}, …) = map(g, xs)` | ✓ leaks (`run` prints) | type error |
| R5 | `func ask() -> bool ! {AI}` passing an `! {IO}` renderer to `std/ai.stepWithStream` | ✓ accepted, IO undeclared | `Missing effects: IO` |

R4 matters for choosing the fix. Its open row is **genuine row polymorphism**, not an L1-open compatibility tail, so a fix that only closes compatibility tails (alternative C) leaves it leaking.

### Impact

Anyone who annotates a callback parameter or a record field with its real effects, which is exactly what the language teaches, can launder those effects through a pure std combinator. A `pure` signature, or any closed signature such as `! {AI}`, then stops being an upper bound on what the function does. The runtime capability layer still refuses IO that the host did not grant. But `ailang check`, the iface's `pure` flag (PR #1790) and SMT admission all trust the static row.

## Root Cause

### RC1: the unification rule, as it is and as it should be

AILANG's rows follow Rémy-style row unification (`docs/docs/references/index.md:44`, Rémy 1993; also used for records in `design_docs/implemented/v0_3_0/M-R5_records.md:446`). The effect rows follow Koka (`docs/docs/references/index.md:84-90`, Leijen 2014, "AILANG's row-polymorphic effect types are directly influenced by Koka"). In both systems an open row `{l₁…lₙ | ρ}` means *at least* `l₁…lₙ`, and a closed row `{m₁…mₖ}` means *exactly* `m₁…mₖ`. Unifying them is solvable only if every `lᵢ` is among the `mⱼ`, and then `ρ := {mⱼ} \ {lᵢ}`. A closed row has no tail, so it cannot absorb a label it lacks. In Leijen's formulation this is the row-rewriting step: to unify `⟨l | ρ⟩` with a row `s`, you must rewrite `s ≅ ⟨l | s′⟩`, and that rewrite fails on a closed row without `l`.

| Case (`internal/types/row_unification.go`) | As it is | Should be |
|---|---|---|
| closed ~ closed (`:89-95`) | fail unless the label sets are equal | unchanged |
| **open r1 ~ closed r2** (`:97-108`) | `r1.Tail := only2`; `only1` **ignored** | fail if `only1 ≠ ∅`; else `r1.Tail := only2` |
| **closed r1 ~ open r2** (`:110-121`) | `r2.Tail := only1`; `only2` **ignored** | fail if `only2 ≠ ∅`; else `r2.Tail := only1` |
| open ~ open, different tails (`:124-155`) | fresh `ρ`; `r1.Tail := only2 ∪ ρ`, `r2.Tail := only1 ∪ ρ` | unchanged (already Rémy) |
| open ~ open, same tail (`:126-130`) | fail if either has unique labels | unchanged |

**History (V7):** the first version (`ef12f3eef`, 2025-09-26) checked the *wrong* side. With an open r1, it failed on `only2`, which is the case that should succeed, and it bound the tail to the open side's own labels. The v0.3.11 fix (`7093db46b`, 2025-10-16, "Critical v0.3.11 row unification regression") corrected the binding and **deleted** the guard instead of moving it to `only1`. The regression matrix that was added then (`internal/types/row_unification_regression_test.go:16`, `TestRowUnification_OpenClosedMatrix`) covers open sides with labels only where the closed side contains them. It has no row for "the open side has a label the closed side lacks", which is why nothing caught the deletion.

The record-row unifier in the same package already enforces the correct rule (`internal/types/unification_records.go:495-510`: "field '%s' in open row1 but closed row2"). The two row unifiers disagree on this case (RC5).

### RC2: why it became reachable in v0.53

Closed callback slots are everywhere. `std/list.map` is `pure func map[a, b](f: a -> b, xs: [a])` (`std/list.ail:58`) over the builtin `_list_map : (a -> b, list[a]) -> list[b]`, whose callback row is closed (`internal/pipeline/testdata/builtin_types.golden:171`). The same holds for `filter`, `foldl`, `foldr`, `any`, `findIndex`, `takeMap`, `mapAccumL`, the `std/string` folds and the `std/xml` folds. A census of the golden lists **30 builtins with a function-typed parameter: 25 with a closed pure callback row, 3 row-polymorphic (`_list_flatMap`, `_list_sortBy`, `_terminal_withTerminal`) and 2 with a closed concrete row (`_dom_subscribe`, `_msg_subscribe`)** (V12).

Open rows that carry labels became routine with L1-open (`internal/elaborate/file_funcs.go:414-417`, shipped in v0.53 by M-EFFECT-LATENT-FUNCTION-VALUES). A concrete annotation `! {IO}` is elaborated as `{IO | ε_annotN}`, where the tail exists for compatibility. Before that, annotations dropped their labels entirely (#573). So the unifier bug is old, but the observable leak through annotated values dates from v0.53. Mixed rows (`! {IO, e}`, R4) could reach it earlier.

Lambda and named-function arguments do **not** leak today. Their rows are closed (`{IO}`), so they hit the closed~closed branch and get the existing "incompatible closed rows" error. That is why `map(\x. println(x), xs)` and `map(loud, xs)` are rejected while `map(g, xs)` with an annotated `g` is not (V8).

### RC3: why the naive fix flips 17 in-repo + 18 external files, and why all 35 are true positives

Every flip has the same first diagnostic: `type error in std/ai (decl 7) … function application of _ai_step_with_stream at std/ai.ail:303 … failed to unify parameter 4`. The other 16 in-repo files and all 18 external files fail only because they import `std/ai` (V13, V14).

`std/ai.ail:302-303`:
```ailang
export func stepWithStream(…, on_chunk: StreamChunk -> () ! {IO}) -> Result[StepResult, AIError] ! {AI} =
  _ai_step_with_stream(model, messages, tools, cache_breakpoints, on_chunk)
```
`_ai_step_with_stream`'s Go type builds the callback with `T.Func(streamChunkType(T)).Returns(T.Unit()).Build()` (`internal/builtins/ai_step.go:607`; the recorded variant is at `:463`). That gives a **closed pure** row. The comment directly above (`:603-606`) says: *"the callback's effect row is intentionally left open in the Go-side type so AILANG's row polymorphism can flow the user's effects through"*. `std/ai.ail:296` says *"The callback's effect row is open (`\!{IO}` is typical but not required)"*. Neither statement is true. The builder supports `.RowTail("e")`, and `_terminal_withTerminal` uses it (`internal/builtins/terminal.go:29-30`).

So `std/ai.stepWithStream` passes `{IO | ε_annot}` into a closed `{}` slot, and the leak lets it through. Two consequences follow, and both are measured:

- **Unsound today (R5, V9).** `func ask() -> bool ! {AI}` passing an `! {IO}` renderer to `stepWithStream` checks clean. The iface publishes `stepWithStream`'s callback as a closed `{IO}` contract that the callee claims to discharge (the L2 rule), and the callee's `! {AI}` silently omits it.
- **Over-strict today (V10).** Passing a *pure* callback (`func quiet(c: StreamChunk) -> () = ()`) is **rejected** on base with `incompatible closed rows … [IO]`. That contradicts "not required".

So `std/ai` declares something false. The flips are the leak's only customer in 693 corpus files, and they are true positives.

### RC4: what the naive fix over-rejects (false positives)

Row unification is equality, not subsumption. The permissive branch has also been silently providing **width subsumption**: a function value whose latent row is *closed and narrower* fits a concretely annotated *wider* slot, because the slot's compatibility tail plus the ignored labels made `{} ~ {Stream, Process | ε}` succeed. With the Rémy guard, that becomes a type error:

| Probe | Base | Phase 1 | Sound? |
|---|---|---|---|
| `export func mk() -> (string) -> int ! {Stream, Process} = \c. 1` (`TestEffectCeiling_OwnTypesAndCallbacksNotCharged/hook typed wider than its body`) | ✓ | ✗ | yes, false positive |
| the same through a sibling module (`…/reference to own sibling whose type carries wider rows`) | ✓ | ✗ | yes, false positive |
| `runWide(f: (string) -> () ! {IO, FS}, …)` called with `loud ! {IO}` (`c_wide_slot`) | ✓ | ✗ | yes, false positive |
| the same, eta-expanded `runWide(\s. loud(s), "x")` | ✓ | ✗ | yes, false positive |

The **corpus contains no instance of this class**: 0 of 515 + 165 + 181 files (V13–V15). Only the two pinned subtests and the constructed probes hit it. The sound fix is the parent doc's deferred **Phase 3, "open on use"** (`design_docs/planned/v0_48_0/m-effect-latent-function-values.md:362-367`). A function value's closed latent row is opened with a fresh tail where the value flows into a slot, and an unconstrained tail is closed again afterwards. Then `{IO | μ} ~ {IO, FS | ε}` succeeds, while `{IO | ε} ~ {}` still fails.

A scratch spike measured the naive form (V16). Opening unannotated lambdas fixes both pinned subtests. But it **regresses 16 corpus files**, because the fresh tails reach the `ApplicationEffects` publication record unowned (`internal invariant: missing or malformed ApplicationEffects`, `internal/pipeline/validate_effects_latent.go:56`). Opening at the argument hand-off with explicit tail closing also trips the invariant on `mapE(\x. …)`. So Phase 2 needs real integration with the #616/#1708 publication record. It is not a one-line addition, and it is not needed to close the P0.

### RC5: a second row unifier in the same package (seam)

`(*Unifier).unifyRows` (`internal/types/unification_records.go:412`, for `TRecord2`) already rejects an open row's extra fields against a closed row (`:495-510`). It does, however, unify two open tails directly (`:480-494`) without redistributing unique labels. The rule this doc restores is therefore already the in-package intent. Merging the two implementations is out of scope (Non-Goals), but the open~open discrepancy is filed as a follow-up.

Record rows reach `RowUnifier` only through `RowEq` constraints (`internal/types/inference_helpers.go:203,236`). At the surface, `{name: string | r}` against `{age: int}` is already rejected by the record path (V11), so the record face of RC1 is not observable today. The Phase 1 unit tests cover it anyway.

### The intended language rule (what `map` should be)

- `std/list.map`/`filter`/`foldl`/… are **pure** combinators. Effectful iteration uses the declared row-polymorphic family `mapE`/`filterE`/`foldlE`/`flatMapE`/`forEachE` (`std/list.ail:205-264`, `export func mapE[a, b, e](f: a -> b ! {e}, xs: [a]) -> [b] ! {e}`). The teaching prompt says the same thing: `ailang prompt` lists `map(f, xs)` under pure functions and states *"Effectful: use mapE/filterE/foldlE (replaces hand-rolled recursion!)"*.
- Whether `map` & co. should become effect-transparent is an explicitly deferred **library** decision of the parent doc (`m-effect-latent-function-values.md:331, 603`). This doc does not reopen it. It is also not needed: none of the 35 flips involves `std/list`.
- M-EFFECT-ROW-VAR-UNIFICATION (`design_docs/implemented/v1_0_0/m-effect-row-var-unification.md`) made declared `! {e}` rows sound end to end. This doc relies on that for the Phase 0 signatures (V17).

## Goals

**Primary Goal:** no unification may drop an effect label. An open row meets a closed row only if the closed row already contains every label the open row carries.

**Success Metrics:**
- R1–R5 are rejected before execution on both engines.
- 0 status flips and 0 first-diagnostic changes over `examples/**` + `std/**` (515 files at base), 45 `ailang-packages` roots (165 files) and `ailang-parse` (2 roots, 181 files).
- `std/ai.stepWithStream` accepts a pure callback and charges an `! {IO}` callback to its caller.
- Go suites are green except the 2 subtests that are re-pinned on purpose (F2).

## Non-Goals

- **Width subsumption for closed function values (RC4).** That is Phase 2, the parent's deferred Phase 3. It needs its own spike and gate.
- **Making `std/list.map`/`filter`/`foldl` effect-transparent.** That is a human library decision held by the parent doc.
- **Merging the two row unifiers (RC5)**, and the `TRecord2` open~open tail redistribution. Filed as a follow-up.
- **The other 23 closed-pure callback builtins.** `_stream_onEvent`, `_stream_select_events` and `_stream_bridge` are wrapped by `std/stream` with `! {e}` handlers. Those rows carry no labels, so the strict rule cannot flip them (V13). Making them row-polymorphic for consistency is an agent-optional Deferred Decision.
- **#1718 (nested callbacks) and #1826 (compatibility-tail escape)**: see the sibling doc.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| D1: Restore the Rémy side condition in `RowUnifier` (reject, do not drop) | Shared row algebra used by every effect and record unification | compiler (row theory) | design | med |
| D2: `std/ai.stepWithStream`/`stepWithStreamRecorded` become `on_chunk: StreamChunk -> () ! {e}` → `! {AI, e}`, with builtins using `.RowTail("e")` | Changes a frozen std interface (`.stdlib-golden/ai.*`); callers' declared rows decide what they pay | human | design | med |
| D3: Ship Phase 1 **without** width subsumption and re-pin the 2 `EffectCeiling` subtests | Rejects a sound-but-unused program class until Phase 2 | human | design | low |
| D4: Breaking posture: minor bump, CHANGELOG "Breaking — soundness", no opt-out flag | Programs that check today stop checking | human | compile | low |
| D5: Error text names the dropped labels and the slot | Agents fix from the message | agent | compile | low |

### Design Freeze

- [x] **F1 (human) — Ruled 2026-10-26 (Mark, attended): accepted as recommended.** accept D2, the row-polymorphic `std/ai` signature, over the concrete `! {AI, IO}` (alternative D). Recommendation: accept. The concrete form over-charges pure callers, contradicts the documented "not required", and **cannot land before #1826**: in the prototype it fails in `std/ai` itself with `Unresolved effect tail: ε_annot1` (V18).
- [x] **F2 (human) — Ruled 2026-10-26 (Mark, attended): accepted as recommended. Proposed to REOPEN for lambdas, 2026-11-02 (D-R1 in the re-land brief): the "0 corpus files" premise was measured on `examples/**` + `std/**` only, and the shape broke `executor_tools/`.** accept D3. Ship the P0 with the width class (RC4) rejected, and re-pin the two subtests to spellings that pass under the strict rule: a named hook declaring the slot's rows (`func one(c: string) -> int ! {Stream, Process} = 1`), or an annotated lambda (`func(c: string) -> int ! {Stream, Process} { 1 }`). Both are verified (V19). The subtests still guard what they exist for, the ceiling not charging own types. Phase 2 restores the original spelling. Recommendation: accept, since 0 corpus files are in the class.
- [x] **F3 (human) — Ruled 2026-10-26 (Mark, attended): accepted as recommended.** confirm D4.

## Solution Design

### Phase 0: make `std/ai` say what it does (implemented; 0 flips on its own)

1. `internal/builtins/ai_step.go:463,475` and `:607,616`. The callback becomes `T.Func(streamChunkType(T)).Returns(T.Unit()).RowTail("e").Build()`, and the result becomes `….RowTail("e").Effects("AI")`. The type is then `(…, StreamChunk -> () ! {...e}) -> … ! {AI, ...e}`, the same shape as `_terminal_withTerminal`. Fix the `LongDesc`/comment to match.
2. `std/ai.ail:302` and `:318`:
   ```ailang
   export func stepWithStream(model: string, messages: [Message], tools: [ToolSchema], cache_breakpoints: [CacheBreakpoint], on_chunk: StreamChunk -> () ! {e}) -> Result[StepResult, AIError] ! {AI, e} =
     _ai_step_with_stream(model, messages, tools, cache_breakpoints, on_chunk)
   ```
   Do the same for `stepWithStreamRecorded`. Update the doc comment at `:296`.
3. Regenerate `internal/pipeline/testdata/builtin_types.golden` (rows 23–24) and `.stdlib-golden/ai.{json,sha256}` (`make freeze-stdlib`; the pre-push hook refuses otherwise).
4. Measured with Phase 0 alone, without the unifier change (variant `pc`, V20): 0/515 flips. R5 is already rejected (`Missing effects: IO`) and a pure callback is accepted. So Phase 0 is a sound improvement on its own and can merge first.

### Phase 1: the unifier rule (implemented)

`internal/types/row_unification.go`:

```go
case r1.Tail != nil && r2.Tail == nil:
    // Rémy side condition: a closed row cannot absorb labels it lacks.
    if len(only1) > 0 {
        return nil, fmt.Errorf("effect row %s carries %v, which the closed row %s does not permit",
            ru.rowString(r1), ru.labelNames(only1), ru.rowString(r2))
    }
    … existing binding r1.Tail := only2 …
case r1.Tail == nil && r2.Tail != nil:
    if len(only2) > 0 { …symmetric… }
    … existing binding r2.Tail := only1 …
```

The message wording is agent-chosen (D5). It must name the labels, and for `EffectRow` kind it must say "effect". For an App parameter the existing path prefix already adds `function application of map … failed to unify parameter 0`. An agent-optional hint ("use `mapE`") is a Deferred Decision. The record kind gets the same guard and the same wording, with "field".

Tests:
- Extend `TestRowUnification_OpenClosedMatrix` with the missing rows: `open{IO}|ε ~ closed{}` fails, `open{IO,Net}|ε ~ closed{IO}` fails, the symmetric pair, and the record-kind pair. **Mutation check:** revert the two guards, and these rows plus AC1–AC4 must go red (V21 did this at the binary level).
- Pipeline fixtures for R1–R5 and the negative controls (AC5), using the existing `checkModules`-style harness, and the two-engine run-rejects-before-execution harness.

### Phase 2 (deferred, separate spike): width subsumption by "open on use"

The goal is to restore RC4's sound programs. A closed latent row of a function **value** (lambda, function reference, field read) is opened with a fresh tail where it flows into a slot (App argument, return annotation, `let` annotation, record or constructor field). After the application-local solve, any such tail that is still unconstrained is closed to `{}` and registered with the `ApplicationEffects` record as owned or closing (`typechecker_app_effects.go:64-90`, `closingRows`). The spike's gate: the original two subtests green **as originally written**, the `c_wide_slot` and eta probes accepted, R1–R5 still rejected, and 0 corpus flips. Naive measurements are in V16.

### Alternatives considered

| | Alternative | Measured result | Verdict |
|---|---|---|---|
| A | Naive guard only (no Phase 0) | 17/515 + 18 external flips (one root, all true positives); 2 Go subtests red | Not shippable alone: it breaks `std/ai` |
| **B** | **Phase 0 + Phase 1 (this doc)** | **0 flips in all corpora**; 2 subtests re-pinned | **Recommended** |
| C | Keep the unifier; close concrete-annotation compatibility tails when a binding is *read as a value* (as the sibling doc does for effect rows) | Fixes R1–R3 by turning them into closed~closed errors, but **R4 (`! {IO, e}`) still leaks**, because its tail is real row polymorphism, not a compatibility tail | Insufficient: it treats a symptom of RC1 |
| D | Phase 1 + concrete `std/ai` (`! {AI, IO}`) | `std/ai` itself fails: `Unresolved effect tail: ε_annot1` (#1826) → 17/515 flips again (V18); over-charges pure callbacks | Blocked on #1826 and worse API |
| E | Phase 1 + "open on use" now | Lambda opening: fixes both subtests but **16/515 regressions** (publication invariant); hand-off opening: invariant on `mapE(\x. …)` (V16) | Needs integration work; this is Phase 2 |
| F | Make `std/list.map` & co. effect-polymorphic instead | Does not fix R3/R4 through other closed slots (25 closed-pure builtins); it is a library decision held elsewhere | Out of scope; not a fix for RC1 |
| G | Directional check only at App arguments | Misses record-construction and annotation hand-offs (R3's field position, `let`), so it is a second code path for one rule | Rejected (systemic-fix principle) |

## Migration plan: every flipped file

**In-repo (`examples/**` + `std/**`, 515 files).** Phase 1 alone flips 17. The root is `std/ai.ail` (1 file). The other 16 only import it: `examples/ai_openrouter_routing.ail`, `examples/configdriven_provider_demo/main.ail`, `examples/wasm_step_byo_key/chat.ail`, and in `examples/runnable/` the files `ai_caching`, `ai_call_json_simple_result`, `ai_call_result`, `ai_call_speech`, `ai_effect`, `ai_image_generation`, `ai_offline_cli`, `ai_streaming`, `ai_tool_loop`, `ai_vision_input`, `game_npc_dialogue`, `structured_ai_basic` and `structured_ai_schema`. **Migration: Phase 0 edits only `std/ai.ail`.** None of the 16 examples changes. The two `stepWithStream` call sites (`examples/runnable/ai_streaming.ail:63` in `main`, and `examples/wasm_step_byo_key/chat.ail:58` in `askStreaming`) are both in functions that already declare `! {AI, IO}`. `ai_vision_input.ail` mentions `stepWithStream` only in a comment. After Phase 0: 0 flips, and the first diagnostics are identical (V13).

**External (read-only scratch copies).** Phase 1 alone flips 18:

- `ailang-parse` (`0931946`): 10. `docparse/main.ail`, `docparse/services/{ai_generator, direct_ai_parser, docparse_browser, layout_ai, orchestrator, mcp/tools}.ail`, `scripts/{test_format_sniffing, test_mcp_parse, test_orchestrator}.ail`.
- `ailang-packages` (`e998928`): 8. `motoko-ext-ai-compat/ai_compat.ail` (1), `motoko-ext-compaction-ai/{_smoke, compaction_ai, register}.ail` (3), and `motoko-ext-compose/{author_loop, claimcheck, compose, register}.ail` (4).

All 18 have the `std/ai.ail:303` root. The one direct `stepWithStream` caller, `ai_compat.ail:199`, is in a function declared `! {AI, IO}`. **Migration: none.** After Phase 0 there are 0 flips (V14, V15). No package needs a pin bump.

**Go tests.** Over all of `./internal/...` (143 packages), only these change: `TestBuiltinTypes_GoldenSnapshot`, from Phase 0 (regenerate the golden), and the 2 `EffectCeiling` subtests, from Phase 1 (re-pin per F2). See V22.

**User code** (shapes that start failing, each with the fix verified on the prototype):

| Rejected shape | Fix | Verified |
|---|---|---|
| `map(g, xs)` / `filter` / `foldl` / `any` with an effectful annotated `g` | `mapE`/`filterE`/`foldlE`, with a row-polymorphic parameter: `func viaE[e](g: (int) -> int ! {e}, xs: [int]) -> [int] ! {e} = mapE(g, xs)` | V19 (`ok_mapE_rowpoly`, `m_rowleak_fix2`) |
| the same with a *concrete* `! {IO}` `g` into `mapE` | blocked on #1826 (`Unresolved effect tail` on base too); use the row-polymorphic form until it lands | V19 (`m_rowleak_fix`: TAIL on base and prototype) |
| a closed function value into a *wider* concrete slot (RC4) | declare the value with the slot's row, annotate the lambda, or make the slot row-polymorphic | V19 (`m_hook_a`, `m_hook_b`, `m_wide_poly`, `m_wide_annlam`) |

## Conflict Surface

This change touches `internal/types/`, so a Conflict Surface is required.

1. **Positions extended.** Every row unification of kind `EffectRow` or `RecordRow` reaching `RowUnifier.UnifyRows`. That means function types' effect rows via `unification_types.go:56-70` → `unification_core.go:299`, plus `RowEq` constraints (`inference_helpers.go:201-210, 234-243`).
2. **Other constructs in those positions:**
   - Open-empty-tail rows from unannotated arrows (`ε_annotN` with no labels). Not affected, since `only` on the open side is empty.
   - Mixed declared rows `! {Stream, e}` (std/stream, std/smoke). Affected only when they meet a closed row lacking `Stream`. The sweep shows 0.
   - #386's application-local solve (`solveLocalEqualities`). It now fails earlier, at the App, for the same program, which is the intended behaviour.
   - #1091 closure (`closeDeclaredEffectRow`). It produces closed rows, and the open side must now fit inside them. The sweep shows 0 flips.
   - Recursive alias-chain resolution (`applySubToRow`, `row_unification.go:161-217`, `3c0d5d8bf`). Untouched. The guard runs after canonicalisation.
   - Effect params and budgets (`effectParamsCompatible`). Untouched, since only labels are compared.
3. **Disambiguation.** None needed. This is a semantic rule, not a syntactic one.
4. **Programs that MUST still work** (all verified on the prototype, V19): `examples/runnable/effectful_list_t1_mapE_basic.ail` (passes on base and on pb), `map(\x. x * 2, xs)` in a `pure` func, `mapE(\x. { println(…); x * 2 }, xs)` in `! {IO}`, `stepWithStream` with an `! {IO}` renderer in `! {AI, IO}` (`examples/runnable/ai_streaming.ail`), and `std/stream`'s `! {Stream, e}` wrappers (std sweep).
5. **Deliberately changes:** R1–R5 (now rejected); a pure callback to `stepWithStream` (now accepted); the RC4 width class (now rejected until Phase 2, per F2).

## Examples

**Before (base):** R1 checks clean and prints `side effect: a`.
**After:** check fails with something like `type error … function application of map … failed to unify parameter 0: effect row {IO | ε} carries [IO], which the closed row {} does not permit`. The migration is `mapE` with a row-polymorphic parameter.

**std/ai, after Phase 0** (this is probe `ai_ok_pure.ail` exactly; `ai_leak.ail` is the same program with an `! {IO}` renderer `render` in place of `quiet`):
```ailang
module t/ai_ok_pure
import std/ai (stepWithStream, Message, StreamChunk)
func quiet(c: StreamChunk) -> () = ()
export func ask() -> bool ! {AI} {
  let msgs: [Message] = [{ role: "user", content: "hi", tool_calls: [], tool_call_id: "", images: [] }];
  match stepWithStream("", msgs, [], [], quiet) { Ok(_) => true, Err(_) => false }
}
```
Base: `incompatible closed rows … [IO]` (a pure callback is refused). Phase 0: accepted. With `render ! {IO}` instead, base accepts an undeclared IO in `! {AI}`, and Phase 0 gives `Missing effects: IO`.

## Acceptance Criteria

- [x] **AC1 (repro):** `t/rowleak.ail` is rejected by `ailang check`, and `ailang run` (tree-walker and `--bytecode`) refuses it before printing.
- [x] **AC2 (spellings):** R2 (`filter`, `foldl`, `any`), R3 (record field) and R4 (mixed `! {IO, e}`) are rejected, and the message names `IO`.
- [x] **AC3 (std/ai soundness):** R5 is rejected with `Missing effects: IO`. A pure callback to `stepWithStream` in a `! {AI}` function is accepted. An `! {IO}` renderer in `! {AI, IO}` is accepted.
- [x] **AC4 (unit matrix):** the new `OpenClosedMatrix` rows exist and pass. All existing rows pass unchanged, including `TestRowUnification_StdlibRegressionCase`.
- [x] **AC5 (negative controls, must stay as on base):** `ctl2` (`direct`) is still `Missing effects: IO`; `map(\x. println…)` in `pure` is still a type error; `ok_map_pure`, `ok_mapE_lambda` and `ok_mapE_rowpoly` are accepted; `hof_optmap` and `neg_mapE_pure` are still `Missing effects`; `c_any_lambda_io` is accepted.
- [x] **AC6 (corpus):** 0 status flips and 0 first-diagnostic changes over `examples/**` + `std/**`, all `ailang-packages` roots and `ailang-parse` (re-measured at the executor's merge base and recorded by SHA). `make verify-examples` and `tools/verify-stdlib.sh` pass.
- [x] **AC7 (re-pins, per F2):** the two `EffectCeiling` subtests are rewritten to a strict-compatible spelling and still assert "no authority entry point reaches Process". A comment links Phase 2.
- [x] **AC8 (mutation):** with the two guards reverted (keeping Phase 0), AC1, AC2 and the new matrix rows fail. With Phase 0 reverted (keeping the guards), `std/ai.ail` fails at `:303`. Both reverts are recorded in the validation doc, done on a scratch copy and never with `git checkout <file>`.
- [x] **AC9 (gates):** focused `go test ./internal/types/... ./internal/pipeline/... ./internal/elaborate/... ./internal/iface/... ./internal/builtins/...`, then `make test-core`, `make lint`, `make check-file-sizes`, `make freeze-stdlib` (golden committed), `make check-changelog` and `make verify-examples`, all run unpiped.
- [x] **AC10 (docs):** a changelog fragment marked "Breaking (soundness)". The `std/ai` doc comment and the builtin `LongDesc` are corrected. An `examples/` file shows `stepWithStream` with a pure callback.

## Ordering and non-overlap

**Against M-EFFECT-LATENT-NESTED-CALLBACKS (#1718/#1826, `design_docs/implemented/v0_56_0/m-effect-latent-nested-callbacks.md`):**
- **Different defects, no shared files.** That doc fixes *who is charged* for function values nested in data, and the compatibility tail escaping into published or inferred call rows. It touches `typechecker_latent_mask.go`, `typechecker_app_effects.go`, `typechecker_functions.go` and a new walker. This doc fixes the *unifier* and `std/ai` (`row_unification.go`, `builtins/ai_step.go`, `std/ai.ail`). That doc already lists this defect as "U1, out of scope, must be filed separately" (its line 186-190).
- **No fix subsumes the other.** The #1718 prototype binary still accepts R1, R2, R5 and `c_wide_slot` (V23). This doc's prototype still shows #1826's `Unresolved effect tail` on `mapE` with a concrete annotated parameter (V19).
- **Interaction.** That doc's intended flip `map(\d. g(d), xs)` and this doc's `map(g, xs)` are the same user-facing rule reached through two paths. They share one migration message (`mapE`). Alternative D becomes possible only after #1826.
- **Order:** either order works. Recommended: Phase 0 of this doc first (it is independent and fixes R5 by itself), then either doc, with this breaking row guard targeting **v0.56.0**. If the nested-callback fix ships earlier, retain its breaking note and validate the combined tree. The executor of whichever lands second re-runs the corpus sweep on the combined tree. Phase 1 + #1718 together were **not measured** here.

**Against PR #1790 (M-PURE-ROW-AND-IFACE-PURITY):** no code overlap with `row_unification.go`. #1790 touches `internal/iface/*`, `internal/smt/*`, `validate_effects.go` and **regenerates every `.stdlib-golden/*.json`, including `ai.json`**. Phase 0 also changes `ai.json`, so the second to land regenerates it with `make freeze-stdlib`; that is a textual conflict only. Semantically, #1790's iface `pure` flag trusts static rows, and R1–R4 make it publish `pure: true` for functions that perform IO. So this fix makes #1790's flag truthful, and it should land in the same release.

## Testing Strategy

- **Unit (`internal/types`):** the open/closed matrix in both orientations and both kinds. A record-kind row. A row whose open side is a mixed declared row.
- **Integration (`internal/pipeline`):** R1–R5 and AC5 as module fixtures; the two-engine run harness; the re-pinned ceiling subtests.
- **Corpus:** the sweeps from V13–V15, recorded in a validation doc next to the sprint plan. Scripts used here: `scratchpad/dd1/sweep.sh` (in-repo, 8-way parallel, `-timeout 90s`) and `scratchpad/dd6/extsweep.sh` (per package root, `check .`).

## Risks & Mitigations

| Risk | Likelihood | Mitigation |
|---|---|---|
| Hidden reliance on width subsumption outside the measured corpora (motoko hook ABIs, world) | Med | F2 makes it an explicit, documented rejection with three verified migrations; Phase 2 restores it; the executor sweeps motoko/world package roots and records SHAs |
| `std/ai` interface change breaks a package that pins the old signature | Low | The new signature is strictly more general for callers; all 4 external direct-or-transitive users pass (V14) |
| Error text churn breaks a golden or snapshot test | Low | V22 lists every red test; none asserts the old open/closed text |
| Combined with #1718 produces new interactions | Med | Whichever lands second re-sweeps the combined tree (AC6) |
| Content cache hides a regression | Med | Every arm uses `AILANG_NO_CACHE=1` and fresh copies |

## Deferred Decisions

- Exact error wording and whether to add a `mapE` hint when the callee has an `E`-suffixed sibling. Agent may choose.
- Whether to also give `_stream_onEvent`/`_stream_select_events`/`_stream_bridge` row tails for consistency (no flips expected). Agent may do it if the sweep stays at 0.
- Phase 2 design details (where exactly to open, and how tails are owned). Its own spike.

## Verification Log

All rows were measured at base `7d29031ea` with binaries built from scratch copies: `ailang-base` (base), `ailang-U` (guard only), `ailang-pb` (Phase 0 + 1), `ailang-pc` (Phase 0 only), `ailang-pa` (guard + concrete std/ai), `ailang-pl`/`ailang-po` (Phase 2 spikes). Probe files are in `scratchpad/dd6/w/t/`.

| # | Claim | How verified | Result |
|---|---|---|---|
| V1 | Issue text | `gh issue view 1958` | open, P0, area:effects |
| V2 | R1 leaks on both engines | `check` ✓; `run --caps IO` and `run --bytecode` print `side effect: a` | Confirmed |
| V3 | R2 leaks | `hof_filter`, `hof_foldl`, `hof_any`: base ✓; pb `incompatible rows` | Confirmed |
| V4 | R3, R4 leak | `z_field`, `z_mixed`: base ✓ and `run` prints; pb type error | Confirmed |
| V5 | Guard is the cause | pb vs pc on R1/R3/R4: pc (no guard) ✓, pb ✗ | Confirmed |
| V6 | Direct-call control | `ctl2.ail`: base and pb `Missing effects: IO` | Confirmed |
| V7 | Guard history | `git show 7093db46b -- internal/types/row_unification.go` removes `if len(only2) > 0` / `if len(only1) > 0`; `git show ef12f3eef:…` shows the original wrong-side check | Confirmed |
| V8 | Lambda / named callbacks hit closed~closed today | `a5` (`map(\s. {_io_println…})`), `c_named_io_map`: base `incompatible closed rows` | Confirmed |
| V9 | R5 unsound on base | `ai_leak` (`! {AI}` + `! {IO}` renderer): base ✓; pc and pb `Missing effects: IO` | Confirmed |
| V10 | Pure callback rejected on base | `ai_ok_pure`: base `incompatible closed rows … [IO]`; pc and pb ✓ | Confirmed |
| V11 | Record face not observable at the surface | `r1`/`r2` (`{name | r}` vs `{age}`): base `record field 'name' not found` (record path) | Confirmed |
| V12 | Builtin callback census | parse of `internal/pipeline/testdata/builtin_types.golden`: 30 builtins with a function parameter; 3 row-poly, 2 closed-concrete, 25 closed-pure | Confirmed |
| V13 | In-repo sweep | `sweep.sh` over 515 files: base 432 ✓; U 415 ✓ (17 flips, every first diagnostic `std/ai (decl 7) … ai.ail:303`); pb 432 ✓, 0 flips, diagnostics identical after path normalisation | Confirmed |
| V14 | `ailang-packages` sweep (`e998928`, 45 roots, 165 files, copied to scratch) | `extsweep.sh`: base 162 ✓; U 154 ✓ (8 flips, all std/ai:303); pb 162 ✓, 0 diffs | Confirmed |
| V15 | `ailang-parse` sweep (`0931946`, 2 roots, 181 files) | base 172 ✓; U 162 ✓ (10 flips, all std/ai:303); pb 172 ✓, 0 diffs | Confirmed |
| V16 | Phase 2 spikes | `pl` (open unannotated lambdas): `EffectCeiling` green, but 16/515 regressions (`ApplicationEffects` invariant, e.g. `effectful_list_t1_mapE_basic.ail`); `po` (open at App hand-off + close unconstrained tails): `c_wide_slot` ✓ but `mapE(\x. …)` invariant (`callrow={...ρ3}` unowned) | Measured; deferred |
| V17 | `! {AI, e}` std signature checks | pb `check std/ai.ail` ✓ (sweep) | Confirmed |
| V18 | Alternative D blocked on #1826 | `pa`: `std/ai` → `Unresolved effect tail: ε_annot1`; 17/515 flips | Confirmed |
| V19 | Migrations and controls | pb: `m_hook_a`, `m_hook_b`, `m_wide_poly`, `m_wide_annlam`, `m_rowleak_fix2`, `ok_*` ✓; `m_rowleak_fix` TAIL on base and pb (#1826) | Confirmed |
| V20 | Phase 0 alone | `pc` sweep: 0/515 flips; R5 rejected; `ai_ok_pure` accepted | Confirmed |
| V21 | Mutation (binary level) | pc = pb minus guard: R1/R3/R4 return to ✓ | Confirmed |
| V22 | Go suites | `go test ./internal/...`: base 143/143 packages ok. pb: 142 ok, and only `internal/pipeline` fails, with exactly `TestBuiltinTypes_GoldenSnapshot` (expected regeneration) and the 2 `EffectCeiling` subtests. U (guard only, `./internal/pipeline/`): additionally `TestCoreTIDAG_StdCorpusMatchesGob`, because `std/ai` no longer checks. `cmd/...` was not run | Confirmed |
| V23 | #1718 prototype does not fix #1958 | `scratchpad/dd1/ailang-p2`: R1, `ctl`, R2 (`filter`, `any`), R5, `c_wide_slot` all ✓ | Confirmed |
| V24 | No other open PR touches `row_unification.go` | `gh pr view 1790 --json files` lists no `internal/types/row_unification*`; #1790 does regenerate `.stdlib-golden/ai.json` | Confirmed |
| V25 | `map` is pure by intent and `mapE` is the effectful form | `std/list.ail:58,205`; `ailang prompt` ("Effectful: use mapE/filterE/foldlE") | Confirmed |
| V26 | The second row unifier enforces the rule | read `internal/types/unification_records.go:495-510` | Confirmed (code read) |
| V27 | No new error code introduced | the design reuses the type-error path; no `EFF_`/`TC` code proposed | n/a |
| V28 | Prior agent's "17 of 501" re-verified | `scratchpad/dd1/sweep_U.tsv` (older base, 501 files) and this session's `sw_U.tsv` (`7d29031ea`, 515 files) list the **same 17 files**; the extra 14 files at the new base are unaffected | Confirmed; the "why" was not in the prior evidence (RC3 adds it) |

## Related Documents

- [M-EFFECT-LATENT-FUNCTION-VALUES](../../planned/v0_48_0/m-effect-latent-function-values.md): L1-open (the compatibility tail) and L2; the deferred Phase 3 "open on use" is this doc's Phase 2.
- [M-EFFECT-ROW-VAR-UNIFICATION](../v1_0_0/m-effect-row-var-unification.md): declared `! {e}` soundness that Phase 0 relies on.
- [M-EFFECT-LATENT-NESTED-CALLBACKS](m-effect-latent-nested-callbacks.md): #1718/#1826; first recorded this defect as U1.
- [M-TYPE-EFFECT-ROW regression (v0.9.4)](../v0_9_4/m-type-effect-row-regression.md): the earlier closed-row regression in this area.
- `design_docs/planned/v0_53_0/m-pure-row-and-iface-purity-sprint-plan.md`: PR #1790.

## References

- Rémy, D. (1993). Type inference for records in a natural extension of ML (`docs/docs/references/index.md:44`).
- Leijen, D. (2014). Koka: Programming with Row Polymorphic Effect Types, MSFP 2014 (`docs/docs/references/index.md:88`).
- Issue #1958; sibling issues #1718, #1826; parent issues #573, #1326.

## Future Work

- Phase 2: width subsumption by open on use (spike, then sprint).
- Merge or align the two row unifiers (RC5), and fix the `TRecord2` open~open tail handling.
- Row-polymorphic callback types for the remaining effect-transparent builtins (`std/stream` family).


## Implementation Report — 2026-10-29

Merged in [PR #2005](https://github.com/sunholo-data/ailang/pull/2005) at `80793dbcf42423027be4bab1a0088da54e2323d8`, evaluated source `8de735cd9adfd4dcfe879686e1d7ac80d7a6f1af`. This is implemented on dev and **unreleased**, targeting v0.56.0; no tag, version bump, image build or deployment is implied.

Only approved Phase 0 (streaming row propagation) and Phase 1 (open/closed row side condition) are implemented here. **Phase 2 width subsumption remains deferred**; no open-on-use or upper-bound semantics are claimed complete.

Independent root evaluation: **PASS 95/100**, with regression coverage 10/10 reported separately. This final score includes an explicit function-length rubric correction; the initial acceptance report is preserved separately. See [review report](m-effect-row-open-closed-soundness-evaluation.md), [machine report](../../../.ailang/state/evaluations/eval_M-EFFECT-ROW-OPEN-CLOSED-SOUNDNESS_round_1.json), [plan](m-effect-row-open-closed-soundness-sprint-plan.md) and [validation](m-effect-row-open-closed-soundness-validation.md).

The native full production snapshot at aa3 passed; the final source differs only by an independently exercised interface publication test. Exact-head Linux/Windows, checks and Sonar passed; new-code coverage is 93.5% against the unchanged 80% threshold. One optional Mac job was still running at acceptance, so all remote jobs are not claimed green. All 49 interfaces regenerate and verify stable.

The exact four-field consumer migration merged on the pinned fork at `083ef43726981099a0bad4fedda18fc72105d6fa` after actual **60/60 core and 9/9 extension** proof. Initial fresh no-cache results retain three deadlines; only those three serial retries then passed. Original width 60→55, width-migrated 60→60, combined nested 55/60 and failed four-field-before-compiler measurements remain preserved as history in the companion evidence. Source pin changes do not switch the live rig. Operator-held #1790, IFC/cache-v8 work and the separate alias-ownership repair remain outside this acceptance.

## Re-land implementation report (2026-11-02)

Sprint `M-EFFECT-ROW-OPEN-CLOSED-RELAND` (attended). Plan:
`design_docs/implemented/v0_57_0/m-effect-row-open-closed-reland-sprint-plan.md`. Commits on `dev`:
`ad9aabc57` (M1: F-A1 + F-A2), `ef01e499f` (M2 + M3: F-B with the guard restored), and the M4 commit
(perf and docs). The work is implemented on dev and **unreleased**, targeting v0.57.0.

**What shipped:**
- `typechecker_effect_union.go`: `InferenceContext.unionEffects` / `unionEffectList` replace all 11
  `combineEffects` / `combineEffectList` inference sites. Each lambda gets a union frame
  (constraint checkpoint and entry env), and let-rec groups push their binding types for fixpoint
  absorption. There are no package globals.
- `typechecker_lambda_open.go`: unannotated closed lambda rows are opened with a per-context tail,
  and `closeOpenedLambdaTails` runs before both `applyApplicationSubstitution` sites.
- `row_unification.go`: `830934d23`'s guards are restored, along with the unit matrix
  `row_unification_open_closed_test.go`.
- `instances.go`: missing-instance errors close free effect tails for display.

**Deviations from the plan:**
- M2 and M3 landed as one commit. F-B **without** the guard reopens the leak: an opened IO lambda
  `{IO | ρ}` meets `map`'s closed `{}` through the permissive branch, which accepted 5 shapes when
  measured. F-B is only sound together with the guard.
- The mutation check "no F-A1 → the three cause-A must-accept cases go red" does not hold as
  written. With F-B present, those rows are opened and fit their slots. Without F-A1, the leak tests
  go red instead: 5 union shapes, the D-R2 case, and the recursive callback row. F-A1 is guarded
  either way.
- A new finding, consistent with D-R2's rule: base accepted a recursive function whose declared row
  omits the row of a callback it calls (`func bad[e](f: … ! {e}, n) -> () = … { f(n); bad(f, n - 1) }`),
  while rejecting the non-recursive `f(1)`. The union drop had erased `e`. It is now rejected
  consistently. 0 corpus files use this shape. It is pinned in
  `TestEffectUnionResolve_RecursionAbsorbsOnlyTheSelfRow`, and it is also the guard against
  over-absorption: a mutant that absorbs every tail accepts it.

**Measurements** (base = `05c860aaf`):

| gate | result |
|---|---|
| positive control | M2 alone flips `executor_tools/executor.ail`; this build checks it clean |
| sweep, 810 tracked `.ail` | 0 status flips (595 pass on both) |
| sweep, external (`ailang-packages` `e998928`, `ailang-parse` `bf12238`), 256 files | 0 status flips (249 pass on both) |
| Go `./internal/... ./cmd/ailang/` | 148 packages ok, 0 fail (after M2+M3 and again after the perf change) |
| `make lint`, `check-file-sizes`, `check-changelog`, `verify-examples`, `verify-stdlib` | green (49 stdlib interfaces stable) |
| mutations (scratch copies) | no guards → M2 rejections, D-R2, 5 union shapes and Phase 2b pins red; no F-A1 → as above; no F-B → both `EffectCeiling` originals and "pure lambda into wider slot" red; no absorption → ghost-effect recursion and `TestRecursiveConditional…` red; absorb-all → recursive callback-row case red |
| `check` time, 10 largest passing modules (best of 3) | base 1,717 ms → re-land 1,657 ms (within noise) |

**Perf note.** The first M2+M3 build measured +15.7% (1,698 → 1,964 ms). Two changes removed it:
(1) each lambda frame caches its local solve and extends it only with new constraints, since
folding a suffix into the prefix's result equals folding the whole; and (2) the frame snapshots its
environment pointer and walks it for owned variables only on its first distinct-tail union. That
walk (`FreeEffectRowVars`) was 15.7% of the `oauth.ail` profile. The `-cpuprofile` flag in
`docs/docs/guides/debugging.md` is not accepted by the current `ailang` binary, so the profile was
taken through `go test -cpuprofile` on the pipeline.

**Still deferred:** Phase 2b (named and eta function values into wider slots), pinned in
`TestEffectRowOpenClosedPhase2bDeferred`.

## Pre-release verification (2026-11-03)

The re-land sweep covered this repo plus two external repos (1,066 files). Before tagging v0.57.0
it was widened to **every git checkout under `~/dev` holding `.ail` files: 4,599 files across 29
repos** (each checked from its package root, v0.56.1 against the candidate), plus a boot of every
`serve-api` deployment with its real flags.

**A regression the re-land missed, now fixed.** A single concrete call in an unannotated lambda,
`w(\c. println("x"))` with `w`'s callback declared `! {IO, FS}`, was rejected:
`row {FS, IO, ...ε} carries labels [FS] which the closed row {IO} does not permit`. v0.56.1
accepted it, and it is valid. Cause: with one call, the body row is the application's *unresolved*
callee tail (`inferApp` deliberately leaves the callee row unsubstituted, and no union resolves a
single operand), so `openLambdaRow` saw an open row and skipped it. The tail later resolved to the
closed `{IO}`. The re-land's tests used only `\d. 1` (body row `{}`, closed at inference) and
multi-call bodies (resolved by the union frame), so neither exercised it. Found in
`ailang-demos/streaming/test_stream_basic.ail`. Fix: `resolveLambdaRow` resolves the body row
against the lambda's own frame (local bindings only) before `openLambdaRow`. A tail owned by the
enclosing scope (a callback's `! {e}`, a recursive self row) stays open and linked. The frame now
lives until `inferLambda` returns. The eta form `runWide(\s. loud(s), "x")`, previously pinned as
Phase 2b, is accepted by the same fix. Four must-accept cases were added; with the fix disabled
(scratch copy), all four fail.

**Phase 2b wording corrected.** The re-land said named function values into wider slots were
"still rejected". v0.56.1 accepted them (the unguarded unifier dropped the slot's extra labels), so
`runWide(loud, "x")` and a let-bound lambda passed by name are *newly* rejected valid programs.
The changelog now says so, with the workaround. 0 of 4,599 files hit it.

**Measurements** (v0.56.1 = `ailang-rel056`, candidate = dev + this fix):

| gate | result |
|---|---|
| sweep, 4,599 files / 29 repos | 98 newly rejected, 0 newly accepted. All 98 trace to 21 root sites, every one an effectful lambda (FS, IO, Process, SharedMem, Declassify) passed to a pure `std/list` `map`/`filter`/`foldl`/`any`: D-R2 as ruled. Repos: docparse (1 site), ailang-packages (4), stapledons-godot (4), Daneel (6), email-parse (2), claude-code-workshop (2), ailang-demos (0 after the fix). All of them pin their ailang release, so none breaks until its pin moves. |
| this repo's 810 tracked files | 0 flips |
| executor MCP (`Dockerfile.executor-mcp`, IAM and OAuth args) | v0.56.0 fails to boot with the original `executor.ail:191` error (positive control); the candidate boots all 11 module sets, and `executor_tools/smoke.sh` passes 6/6 |
| docparse `serve-api` | v0.56.1 boots; the candidate refuses `api_keys.ail:1309` (`foldl` with `readFile`, D-R2). Migration: `foldlE`. The pin (`ARG AILANG_VERSION=v0.56.1`) protects the deploy |
| migration targets | `mapE`/`forEachE`/`foldlE`/`filterE` accept multi-call IO/FS lambdas, including the docparse shape |
| Go `./internal/... ./cmd/ailang/` | 148 packages ok, 0 fail |
| `check` CPU time, 10 largest modules (load average ~100) | 2,531 → 2,526 ms (no measurable difference) |

**Known, pre-existing, not introduced here.** The post-check effect validator charges a caller's
IO to a recursive row-polymorphic function written as `{ f(h); each(f, t) }` when it is called
with a single-call IO lambda ("Missing effects: IO" on `each`). v0.56.1 rejects the same program
for `\x. println("x")`. The std combinators are written differently and are unaffected, and 0
corpus files hit it.

### Generated-code sweep (2026-11-03, before tagging)

The hand-written corpora above missed two more regressions; generated code found them. The checked
population was the `code` of every `eval_results/**/*.json` with `lang=ailang` and `compile_ok`,
deduplicated: 15,447 programs, run with `AILANG_RELAX_MODULES=1` because they all declare
`module benchmark/solution`. A first run without it failed every program on both arms (MOD010), a
false "0 flips"; the reruns carried a known-flip control.

- **#2074** (found by another session, fixed in `81649aa8e`): a local recursive IO `letrec` failed with
  "same row variable with different extensions". Bisected to M1.
- **16 more valid programs rejected**, bisected per program across the series: 1 to M1 (`ad9aabc57`),
  15 to M2+M3 (`ef01e499f`); none to M4, the single-call fix, env-scan or `81649aa8e`. There were
  two causes:
  1. An opened lambda tail unified with a return annotation's tail (`pipe … -> int -> int = \x.
     g(f(x))`) was quantified by `let`/`letrec` generalization before `closeOpenedLambdaTails` ran.
     Uses instantiated unowned tails, which surfaced as "missing or malformed ApplicationEffects" (7),
     "distinct unresolved effect tails" (2) and a spurious validator "Missing effects" (6).
     `closeOpenedTailsForGeneralization` now closes such a tail at both generalization sites.
  2. The row unifier's substitution is not idempotent (ρ1 ↦ {...ρ3}, ρ3 ↦ {...ρ6}), and the union
     resolver followed one link, so a self row was missed from the third recursive call on.
     `resolveEffectRow` follows the chain.

Final gate on `802670c5b` against v0.56.1, over 20,046 files (repos and eval code): **0 newly
accepted. 98 repo files are newly rejected (the 21 reviewed pure-slot sites) and 8 eval programs
are newly rejected (all of them effectful lambdas in pure `std/list` slots). Nothing else is newly
rejected.** Regression tests are in `effect_row_reland_eval_regressions_test.go`. Each fix was
mutation-tested.

**Still open (pre-existing):** the recursive `{ f(h); each(f, t) }` validator case described above.
