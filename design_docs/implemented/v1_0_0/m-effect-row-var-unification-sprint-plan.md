# Sprint Plan: M-EFFECT-ROW-VAR-UNIFICATION

Refs #616. Re-plan dated 2026-10-09; supersedes PR #1678.

**Design:** [m-effect-row-var-unification.md](m-effect-row-var-unification.md)
**Duration:** 5 days, approximately 24–30 hours including contingency.
**Risk:** High: effect soundness, row solver interactions and streaming consumers.
**Base:** current dev; minimum required dependency `3d4f49720` (PR #1708,
M-EFFECT-LATENT-FUNCTION-VALUES). The sweep base is the executor branch's actual
merge-base with dev, recorded by SHA (see M4).
**Ordering:** this sprint runs BEFORE M-PURE-ROW-AND-IFACE-PURITY (#1443), which
re-measures its pins after this lands. They must not run in parallel.
**Approval:** approved by the coordinator handoff; execution completed 2026-10-09.
The design unpark/re-scope and this plan must travel in the same planning PR.

## Decision and current status

Mark's ruling on 2026-10-08: **D-10 = option A**, a minimal constraint repair at
App, where `inference.go` mints an independent `freshEffectRow`. Extend #1708's
per-App publication with the resolved call row alongside LatentParamMask, using
one backing authority. No standalone `CallEffects[appID]` store or second presence
lookup. The superseded review is
[PR #1678's maintainer comment](https://github.com/sunholo-data/ailang/pull/1678#issuecomment-6067485482).

Read both App paths: AST FuncCall in `internal/types/inference.go`, Core inferApp
in `internal/types/typechecker_functions.go`. The latter captures the mask before
unification, locally solves equalities, protects outer variables, preserves the
constraints for enclosing solving, and publishes its mask via the companion
`typechecker_latent_mask.go`. Do not undo #386's no-join or recursion protections.
The row in the unified publication is the callee's latent call effect, distinct
from evaluation of arguments. Publication must reflect final substitutions rather
than an early unsolved row or a guessed union of callback effects.

#1091 narrows V14/V15 to declared row-polymorphic imports. Add inferred combinator
and declared-pure importer controls; preserve the shipped closure-before-
generalization repair. Include both single importer checking and explicit
multiple-file checking. The issue's real extraction patch is not available here;
existing regression controls and synthetic fixtures establish bounded coverage,
not a claim to reproduce the external module in full.

#1718 (callbacks nested in list/tuple/ADT arguments) is out of scope unless the
same small repair addresses it without new structural traversal. Keep #1708's
open callback and storage-only behavior. Do not implement this sprint in parallel
with that work; it has already landed in this base.

## Velocity and capacity

The planner velocity script (7 days) finds only the shallow base commit; parent
statistics are unavailable. The previous latent-effects sprint recorded 858 raw
source/test lines against 750 planned, but no instrumented elapsed time. No
measured LOC/day rate is asserted. Budget **700 lines** (260 implementation,
350 tests, 90 examples/docs), a planning target of 140 lines/day, and about 25%
time contingency within five days. Validation and corpus diagnosis dominate the
uncertainty. Re-estimate at M1 if a global solver redesign would be necessary;
that exceeds the maintainer ruling and requires a design revision.

## Registry reuse audit

`ailang pkg search effects` returned `sunholo/billing_entitlements@0.4.2` and
`world/core@0.1.1`; neither supplies compiler inference or validation. No candidate
needs package info/docs inspection. M1–M4 each choose **none**: extend existing
compiler internals and repository tests/documentation, with no package dependency.
The JSON records a populated decision per milestone.

AILANG prompt version loaded: v0.16.6 (`ailang prompt`, checked with
`ailang prompt --version-active`). This is the installed active teaching prompt,
not the repository version (`std/VERSION` is v0.52.5). Executor reloads the prompt
from its freshly built base/fixed toolchain before creating `.ail` fixtures.

## M1 — Minimal App constraint repair and red controls

**Estimate:** 90 implementation + 100 tests = 190 LOC; Day 1, 6 hours.
**Dependencies:** landed #1708; no dependency on the superseded plan.

1. Build a provenance-pinned base CLI before editing implementation. Re-measure
   #616 pure caller/blank Suggested fix and the historical pure runTwice(noisy)
   printing twice with IO granted. Record whether #1708 already changes an arm.
   Never force an obsolete base-red expectation onto the new base.
2. Audit AST/Core App constraints, scheme row instantiation and substitution.
   Add focused regressions for callback/result sharing, pure and IO calls in
   either order, mixed `{IO, e}`, aliases and return-only row variables.
3. Implement only the necessary App equality/sharing repair. Keep outer-scope
   row bindings and deferred whole-program solving intact; avoid premature
   callee closure, global inference rewrite or validator-side name matching.
4. Exercise recursive multi-effect functions and existing #386 no-join tests.

**Files:** `internal/types/inference.go`, replacement edits in
`typechecker_functions.go`, new `internal/types/typechecker_app_effects.go` and
`internal/types/typechecker_app_effects_test.go`; existing substitution/row-unifier
files only if the minimal constraint repair demonstrably requires them.
**Size note:** `typechecker_functions.go` is 781 lines (CI limit 800). M1 edits it, but
new code goes in companion files (`typechecker_app_effects.go`); keep its net growth
at or near zero.
**Rebase note:** #1706 (alias closure) may land first. Expect to rebase over its edits
to `internal/pipeline/pipeline_module_compile.go` and `phases.go`.
**Example/fixture:** temporary pipeline source modules for runIt, runTwice and
retOnly; persistent runnable demos belong to M3.

**Acceptance:** design AC1 and the return-only boundary of AC5 pass; independent
calls resolve to pure/IO without order contamination; recursion/no-join controls
pass. Mutation removing the repaired constraint makes at least one new test fail.
**Risk:** independent fresh tails remain unconnected, or eager solving breaks
recursion. Use both AST/Core tests and surrounding solver controls to distinguish.

## M2 — Extend the existing publication; row validation and diagnostics

**Estimate:** 170 implementation + 130 tests = 300 LOC; Days 2–3, 10 hours.
**Dependencies:** M1.

1. Change the backing LatentParamMask publication into one per-App record with
   pre-unification mask and post-solve/zonked call row. Existing mask accessor may
   project from it; do not retain a parallel row map. Preserve immutable snapshots
   and distinguish a present pure/all-false record from absent metadata.
2. Finalize rows after substitutions/defaulting, preserving legitimate owned
   generic tails; missing/malformed/unowned rows report App ID/span invariants.
   Thread one lookup through both pipeline entry points and effect collection.
3. Consume the published row for row-polymorphic local/imported callees, declared
   or inferred. Keep concrete-declaration contamination protection and #1708's
   latent argument masks. Charge argument evaluation separately exactly once.
4. Tail-preserving union handles nil/closed empty, one tail, same tail and explicit
   distinct-tail conflict. Subsumption/diff reports unmatched tails; concrete IO
   is not absorbed by a declared `{e}`. Cover suggested and required-row callers.
5. Ensure tail failures have actionable text. An empty diff is an internal
   invariant; no identical Suggested fix is printed. Retain budget/param checks.

**Files:** `internal/types/typechecker_latent_mask.go` (or split companion),
new companion publication tests, `internal/types/effects.go`,
`internal/types/effect_subsumption.go`, `internal/pipeline/validate_effects.go`,
`validate_effects_latent.go`, `validate_effects_rows.go`, `pipeline_single.go`,
`pipeline_module_compile.go`, new pipeline publication/diagnostic tests.
`typechecker_core.go` is **800 lines**: any necessary field declaration must be a
replacement with no net growth; all new logic goes in companions. Check other
files before adding code so none exceeds its applicable CI limit.
**Example/fixture:** same-module pure/IO and imported inferred helper fixtures.

**Acceptance:** AC2/AC4/AC5 pass. One store/presence authority; masks remain open
when argument solving closes a callback; records cannot be mutated by consumers.
Wrong-key/missing/malformed publication tests fail loudly, owned open tails pass.
Both pipeline entry points use the extended publication. Distinct-tail failures
name the conflict rather than erasing a tail or silently accepting purity.
**Risk:** duplicate effect charging or concrete-row contamination. Pin #1708's
controls and assert reported function and missing IO rather than any error.

## M3 — Issue regression matrix and working examples

**Estimate:** 120 tests + 35 examples = 155 LOC; Day 4, 5 hours.
**Dependencies:** M1, M2.

1. Port historical arms a/b/d/e/g/k/l/m/n/h/i/j to cache-disabled pipeline tests:
   pure single/double calls, alias call, wrong FS, mixed IO/tail, owned generic
   body, declared-tail-not-absorbing-IO and imported mapE/forEachE controls.
2. Pure runTwice(f) with two callback invocations must reject before any print;
   include runTwice(noisy), whose historical accepted execution printed twice.
   Correct helper declaring `{e}` with IO caller checks/runs and prints exactly
   twice. Verify run without IO retains the capability rejection.
3. Test #1091 pure recursive import closure, standalone importer and explicit
   dependency checks, inferred option/result/list combinators, concrete-row
   recursive contamination and #386/#1708 regression suites. Test computed,
   alias and direct callees. Record #1718 as a separate gap if it persists.
4. Add `examples/runnable/effect_row_var_pure_caller.ail` (pure result 42) and
   `examples/runnable/effect_row_var_noisy_twice.ail` (generic helper, IO main,
   two print markers). Update the existing runnable-example manifest.

**Files:** new `internal/pipeline/effect_rowvar_discharge_test.go`, extend existing
latent invariant tests as needed, `internal/elaborate` regression companion if
lowering loses call identity, two runnable examples and existing manifest.

| Demo | Contracts | Effects | Inline tests |
|---|---|---|---|
| pure caller | skip: signature/effect regression, no precondition | include: callback/result `{e}`, concrete pure caller | include: zero-arg pure entry result 42, using current prompt syntax |
| noisy twice | skip: output measured by runtime harness | include: helper callback/result `{e}`, noisy/main `{IO}` | skip: IO stdout asserted by Go runtime harness |

**Acceptance:** AC3/AC6/AC9 pass. Every negative asserts an effect diagnostic at
the intended function, not an incidental parse/type/module failure. Positive
examples check and run with exact expected output. Mutation disabling tail
preservation or publication consumption trips the relevant tests.
**Risk:** historical arms already changed on base; record that evidence and test
the contract anyway. Use NoCache in pipeline fixtures and fresh runtime paths.

## M4 — Base/fixed corpus evidence, migrations and executor gates

**Estimate:** 55 documentation LOC; Day 5, 5 hours plus contingency.
**Dependencies:** M1–M3.

### Reproducible sweep protocol (AC7/AC8)

- At M1, compute `git merge-base HEAD origin/dev` on the executor branch and build
  the base binary from **that** SHA (record it). `3d4f49720` (#1708) is only the
  minimum required dependency, not the base: #1707 has since changed `effects.go`,
  `effect_subsumption.go`, `validate_effects*.go` and `pipeline_single.go`, so a
  `3d4f49720` base would misreport flips. If the branch is rebased (e.g. over #1706),
  rebuild the base from the new merge-base and record both. Preserve its matching source
  and stdlib in a detached worktree or read-only source snapshot. Never switch a
  dirty branch. Record full SHA, binary hash, build command/version and Go version.
  Build fixed binary from the final executor tree and record commit/tree hash.
- Inventory recursively **all** `.ail` files in `examples/` and `std/` using
  `rg --files --hidden --no-ignore examples std -g '*.ail'`. Preserve sorted path
  inventory and counts; intentional broken examples stay in the dataset. New
  demos are reported separately from the unchanged-file comparison.
- Run `AILANG_NO_CACHE=1 <binary> check <file>` once per file on each matching
  source tree with identical cwd, configuration and module resolution. Capture
  command, exit status, stdout/stderr and normalized diagnostic category/path.
  Use absolute binary paths and matching `AILANG_STDLIB_PATH` for each side; do
  not accidentally import the fixed stdlib into base. Bound each run and classify
  timeouts/infrastructure failures separately from semantic rejections.
- Direct std file checking historically had module-path failures (V30). Keep raw
  results for **every std file** and additionally use the supported relaxed-module
  checking mode identically on both trees (validate CLI placement from help).
  If root std checking still fails on resolution, add per-module import probes
  with the correct module root, retaining each original direct result. A shared
  resolution failure is not proof that a std module's effects work.
- Compare pass/fail sets and diagnostics. Explicitly inspect every import user of
  `std/stream` and `std/ai/streaming`, including mixed `{Stream, e}` and distinct
  callback tails; add targeted import/runtime controls where direct checks cannot
  exercise those calls. Same-tail generic streaming must remain valid; report
  distinct-tail conflicts and determine whether their constraints are incorrect.
- Classify **each flip**: newly accepted pure program (migration: none); newly
  rejected unsound program (declare concrete effects or propagate the correctly
  shared generic row); valid program newly rejected (regression, repair before
  completion). No blanket approval of new distinct-tail streaming rejections.
  Unexplained flips, infrastructure errors or uncovered std resolution failures
  block the corpus gate. Existing baseline failures remain visible by category.

**Artifacts:** `design_docs/planned/v1_0_0/m-effect-row-var-unification-validation.md`
with base/fixed commands/counts and diagnostic matrix; machine-readable evidence
under `.ailang/state/sprints/validation_M-EFFECT-ROW-VAR-UNIFICATION.json`, with
per-file records and flip classification. Reuse existing example audit tools where
possible, extending only the missing full recursive/std comparison support.

**Documentation:** create the implementation changelog fragment
`changelogs/unreleased/<YYYY-MM-DD>-effect-row-var-unification.md` (dated the day it is written;
do not edit root CHANGELOG or `changelogs/v*-current.md`). It must open with a
`### Fixed — …` heading and pass `make check-changelog`.
Name **every flipped path** with before/after, cause and migration, including new
accepts requiring no action; if there are none say so with corpus counts. Explain
static soundness versus the unchanged runtime capability backstop and remaining
#1718. Update relevant limitations/effect docs only where their claims change.

**Required gates, exactly:**

```sh
go test ./internal/types/... ./internal/pipeline/... ./internal/elaborate/...
make test-core
make lint
make check-boundaries
make check-file-sizes
make check-changelog
```

Run working-example checks/runs and corpus comparisons in addition to these gates.
Do **not** run full `make test`: the executor's RAM-backed `/tmp` has caused SIGBUS.
Do not replace a required failing gate with an unrecorded reduced run. Include
commands, statuses, hashes and relevant coverage for new branches in the report;
no arbitrary coverage percentage substitutes for the regression matrix.

**Acceptance:** AC7–AC10 pass; inventory fully accounted for, all flips have
per-file migrations, streaming users have explicit evidence, no touched file
exceeds the CI size limit, fragment passes `make check-changelog`. Re-run focused checks only after relevant changes.
Executor makes local commits with `Refs #616`, cannot push or merge. Send results
through the coordinator for sprint-evaluator review after execution.

## Completion and handoff

M1 → M2 → M3 → M4 is sequential because the shared row algebra and publication
are the same authority. Planning creates the revised design, this document and
populated sprint JSON; it does not implement the compiler repair or claim the
base/fixed sweep has run. Approval of this planning PR supplies the coordinator
handoff. No executor dispatch is authorized merely by re-planning. The unresolved
implementation choices are companion file organization and diagnostic wording;
D-10, publication authority, scope and completion gates are fixed.

## Executor completion — 2026-10-09

- [x] M1: shared App constraint, red measurements and constraint mutation.
- [x] M2: one sealed mask/call-row authority, diagnostics, unions and ownership.
- [x] M3: runtime, imports, recursion/no-join controls and both runnable demos.
- [x] M4: full corpus evidence, zero status flips, fragment and six required gates.

Actual branch: `coordinator/task-c86840f2`; the planner branch is represented by
the base planning commit. The scoped ownership and inferred-function origin
controls exceeded the 260 implementation-line estimate; no global solver or
structural callback traversal was added. Execution ran sequentially in one
session; elapsed development effort was not instrumented as a LOC/day metric.
See [validation report](m-effect-row-var-unification-validation.md) and the
JSON evidence for commands, hashes, status/category counts, mutations and bounded
#1091 coverage. Design stays planned until independent evaluator review.
Local commit only; no push. Refs #616.
