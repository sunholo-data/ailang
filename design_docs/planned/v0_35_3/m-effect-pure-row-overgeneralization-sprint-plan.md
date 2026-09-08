# Sprint Plan: M-EFFECT-PURE-ROW-OVERGENERALIZATION

**Design Doc**: [m-effect-pure-row-overgeneralization.md](m-effect-pure-row-overgeneralization.md)
**Sprint ID**: M-EFFECT-PURE-ROW
**GitHub Issue**: [#1091](https://github.com/sunholo-data/ailang/issues/1091)
**Target**: v0.35.3
**Duration**: 1.5 days (~10 hours)
**Risk Level**: Medium
**Base**: `origin/dev` = `3ee5bb177`

## Summary

**Goal**: A function whose declaration specifies a closed effect row exports exactly that row.
`pure` means `{}` at the module boundary, in every body shape — including recursive ones.

**Why now**: #1091 blocks a real de-duplication refactor in `ailang-parse` with no workaround
other than keeping three copies of the helpers. Root cause is fully isolated (design doc V1–V18),
the fix site is pinned, and both regression directions have concrete fixtures.

**Deliverables**:
1. Effect-row defaulting before generalization, keyed on the *declared* row
2. Scheme-level unit tests (the only level where `RowVars` is observable)
3. Cross-module pipeline fixtures — accept case (#1091) and reject case (V9)
4. End-to-end validation against the reconstructed `ailang-parse` extraction

## Current Status Analysis

**Root cause (from design doc, all rows re-derived this session):**
- `generalizeWithConstraints` (`internal/types/typechecker_functions.go:478-495`) quantifies every
  free effect-row var, never consulting the declared row (V11)
- A recursive self-call deliberately leaves the enclosing row var unbound (V12), so for a `pure`
  function there is nothing concrete to close it against
- Result: `pure` exports `! {...ρ2}` with `RowVars=[ρ2]` (V6); importers unify it with `FS` (V8)
- Asymmetric — any concrete declared row closes normally, so this is a false rejection, **not** a
  soundness hole (V9)

**Velocity context**: recent landed work in this repo is single-milestone bug fixes with tests in
the 150–400 LOC range (`#1102`, `#1104`, `#1106`). This sprint is ~280 LOC across 4 files — squarely
in that band, so estimates are not aspirational.

**Why the risk is Medium not Low**: the defect resists minimization. The reporter failed across four
attempts; an independent dependency-aware delta-debugger converged at **37/37 decls retained**
(every decl is transitively reachable from an `! {FS}` function, so any removal surfaces a type
error that masks the FS error). A fix can therefore look green on a toy case and still fail on the
real one. M3 treats the reconstructed `ailang-parse` tree as a **required** acceptance artifact.

## Milestones

### M1 — Regression tests first (red)

**Estimated**: 3 hours · ~230 LOC (tests only)

Write every test before touching the fix, so each is observed failing for the right reason.

**Tasks:**
- [x] `internal/types/effect_row_defaulting_test.go` — Scheme-level assertions:
  - recursive `pure` binding ⇒ `RowVars == nil/[]` and a closed effect row (currently RED)
  - declared `! {e}` binding ⇒ `RowVars == ["e"]` preserved (currently GREEN — pins V14)
  - recursive `! {IO}` binding ⇒ closed `{IO}` (currently GREEN — pins V9)
- [x] `internal/pipeline/validate_effects_xmod_test.go` — two-module fixtures:
  - accept case: `pure` export delegating to a private recursive scan, imported by a `pure`
    function that is reachable from an `! {FS}` function (currently RED — the #1091 shape)
  - reject case: importer calling a genuinely `! {IO}` cross-module function from a `pure`
    function is still rejected, with unchanged message text (currently GREEN)

**Acceptance criteria:**
- Both RED tests fail with `Missing effects: FS` / a quantified `RowVars`, not with a type error
- Both GREEN tests pass at base, proving they can catch a regression rather than passing vacuously
- Tests assert on the `Scheme`, not on `ailang iface` output (V16: the JSON view flattens the row
  var away and reports a row-polymorphic export as `"pure": true`)

**Risks**: writing an accept-case fixture that does not actually reproduce — mitigated by requiring
the RED test to fail with the *exact* `Missing effects: FS` message before M2 starts.

---

### M2 — The fix

**Estimated**: 4 hours · ~50 LOC

**Tasks:**
- [x] Add effect-row defaulting alongside the existing type-class defaulting pass in the LetRec
      path (`typechecker_functions.go:330-377`) and the corresponding Let/top-level binding path
- [x] Resolve the binding's declared effect row; bind unresolved effect-row variables to it when
      that declared row is closed
- [x] Leave declared row-polymorphic signatures untouched — discriminate on **the declaration**,
      never on the row variable's name (`ρN` vs `e`); the naming shortcut breaks the first time a
      user names a row `rho` (Design Freeze)
- [x] Default only *unresolved variables*, never concrete labels — a genuine effect must still
      reach the effect checker

**Acceptance criteria:**
- All M1 tests green
- `DEBUG_EFFECTS=1` reports `VarGlobal(docparse/services/pkg_template.pkgLastIndexOf) -> []`
- No existing effect test's expected text changed
- `make test` and `make lint` green

**Risks**: closing rows too eagerly could swallow a real effect — mitigated by the M1 reject case
being pinned in both directions, and by defaulting variables only.

---

### M3 — Validation + end-to-end

**Estimated**: 3 hours · ~6 LOC (CHANGELOG)

**Tasks:**
- [x] `make test`, `make lint`, `make verify-examples`
- [x] Reconstruct the `ailang-parse` extraction in a scratch copy (`160c7f1` + extract
      `pkgLastIndexOf`/`pkgDropSpans` and their private scans into `docparse/services/pkg_template`),
      then run `ailang check docparse/services/docx_template.ail` **and**
      `ailang run docparse/main.ail --entry main` — both must pass with no `.ail` edits
- [x] Sweep `std/` for any `pure`-declared export still carrying a quantified effect row — **criterion refined during execution**, see note below
- [x] Confirm `std/list.mapE` still shows `RowVars=[e]`
- [x] CHANGELOG.md entry under v0.35.3

**Acceptance criteria:**
- Every design-doc Success Criterion checked
- The `ailang-parse` end-to-end run is recorded in the sprint notes with its actual output — this
  milestone cannot be closed on unit tests alone

**Risks**: the reconstruction is manual — mitigated by scripting it so it is repeatable, and by
requiring the pre-fix baseline to fail first.

## Success Metrics

- [x] #1091 reproduction passes `check` and `run` with no `.ail` source edits
- [x] `std/list.mapE`/`filterE`/`foldlE` keep `RowVars=[e]`
- [x] ~~No `pure`-declared export in `std/` carries a quantified effect row~~ → **refined**: none carries an *unshared* quantified outer row (`TestStdlib_NoUnsharedQuantifiedEffectRows`). `std/option.flatMap` and `std/result.flatMap` are declared `pure` and legitimately keep a row shared with their callback
- [x] Genuine cross-module effect requirement still rejected (V9)
- [x] `make test` green, no existing expected-text changes
- [x] `make verify-examples` green
- [x] CHANGELOG.md updated

## Example Files

No new `examples/*.ail` file. This is a **defect fix restoring intended semantics**, not a language
feature — `pure` already means `{}` and is already documented. The behaviour is pinned by the
two-module fixtures in `internal/pipeline/validate_effects_xmod_test.go` instead, which is where a
cross-module effect contract can actually be asserted. (`examples/` is single-program and would not
exercise the import boundary that is the entire subject of this fix.)

## Dependencies

**Blocked by**: nothing.

**Interacts with**: [M-EFFECT-ROW-VAR-UNIFICATION](../v1_0_0/m-effect-row-var-unification.md)
(planned v1.0.0, P0, **parked**). Not a blocker in either direction — that doc is about *declared*
`! {e}` signatures. This sprint **refutes one of its pinned acceptance criteria**: its Conflict
Surface records *"Cross-module callees (`VarGlobal` → typeInfo path) | Already correct (V14/V15)"*.
Design-doc V6/V8 show that path is not already correct.

**Required follow-up (M3 task, do not skip):** append a note to
`m-effect-row-var-unification.md` recording that its V14/V15 pinned ACs must be re-scoped to
"declared row-polymorphic callees only" when it is unparked. Leaving that premise standing would
send the next author down a path this sprint has already measured as false.

## Open Questions

None blocking. All four Design Freeze decisions are resolved and agent-resolvable; no quorum
trigger fired (design doc, Quorum Trigger Check).

## Out of Scope

- `ailang check` silently ignoring every argument after the first (V17) — real, separate; it is
  what misled the reporter into the issue's "revealing part". To be filed as its own issue.
- Extending the `validate_effects.go` declared-effects bypass to `*core.VarGlobal` (Future Work).
- Row-variable resolution in `validate_effects.go` (the parked P0).

## Commit Convention

Development commits: `refs #1091`. Final sprint commit: `Fixes #1091`.

---

## Execution Notes (2026-09-08)

Recorded because several items landed differently from the plan. Deviations are within the
latitude the design doc's Deferred Decisions granted, but the plan's own text is now stale in
places, so the actual shape is written down here rather than left implied by ticked boxes.

### Deviation 1 — test files consolidated, and located in `internal/pipeline`

Planned: `internal/types/effect_row_defaulting_test.go` + `internal/pipeline/validate_effects_xmod_test.go`.
Actual: one file, `internal/pipeline/effect_pure_row_overgeneralization_test.go` (7 tests).

Reason: a pure `internal/types` unit test of `generalizeWithConstraints` **cannot** observe this
defect. The declared effect row is not one of that function's inputs — the whole point of the bug
is that generalization never sees the declaration — so the assertion has to be made after the
module interface is built, which is the `pipeline` layer. Asserting on `Result.Interface.Exports[...].Type`
gives the `Scheme` (and therefore `RowVars`) at the level where it actually matters: what an
importer receives.

### Deviation 2 — the fix is narrower than the design doc described

The design doc said to bind the unresolved variable to the declared row and apply the substitution
to the **whole** type, arguing that a variable shared with a callback row should close along with
the outer row. **That reasoning was wrong, and the end-to-end artifact caught it.**

A row variable can be load-bearing with **no** `! {e}` annotation at all: `std/list.flatMap`
declares no effects, yet its callback and result rows share an inferred row, and that sharing is
precisely what lets a caller pass an effectful lambda. Closing it made every such combinator
strictly pure and broke `docparse/services/epub_parser`:

```
epub_parser.ail:72:25: failed to unify parameter 0: failed to unify effect rows:
  incompatible closed rows: r1 has extra labels [], r2 has extra labels [FS]
```

(the line is `flatMap(\entry. epubParseContentFile(filepath, entry), contentFiles)`).

The rule now applies **only when the outer row's variable occurs nowhere else in the type**. The
#1091 shape is exactly that case — `(string, string) -> int ! {...ρ2}` has no function-typed
parameter. Pinned by `TestInferredRowPolymorphicCombinator_AcceptsEffectfulCallback`, which was
verified RED against the over-broad version with that same unification error.

**Consequence for the design doc**: its Solution Design paragraph beginning "The substitution is
returned rather than applied so the caller can apply it to the WHOLE type" is superseded. The
`internal/types/effect_row_declared_closure.go` doc comment carries the corrected rationale.

### Deviation 3 — the `std/` sweep criterion was wrong as written

Planned criterion: "No `pure`-declared export in `std/` carries a quantified effect row."
Measured: `std/option.flatMap` and `std/result.flatMap` are declared `export pure func` and **do**
carry a quantified outer row — shared with their callback, for the Deviation-2 reason. Four more
(`option.map`, `option.filter`, `result.map`, `result.mapErr`) carry one on the callback parameter
only.

`pure` constrains a function's **own** effects; it does not make a combinator opaque to its
callback's. The criterion is therefore "no **unshared** quantified outer row", encoded as
`TestStdlib_NoUnsharedQuantifiedEffectRows`.

That sweep is a **forward guard, not a #1091 regression test** — verified to pass with the fix
disabled, because nothing in `std/` currently pairs a `pure` declaration with a recursive,
callback-free body. The two tests that genuinely go red without the fix are
`TestPureExport_RecursiveBody_HasNoQuantifiedEffectRow` and (against the over-broad variant)
`TestInferredRowPolymorphicCombinator_AcceptsEffectfulCallback`.

### End-to-end artifact — actual output

Reconstruction: `ailang-parse` @ `160c7f1`, compile cache removed, `pkgLastIndexOf`/`pkgDropSpans`
plus their private scans extracted into `docparse/services/pkg_template`, call sites renamed, one
import line added. No other `.ail` edits.

| Tree | Binary | `check docx_template.ail` | `run docparse/main.ail --entry main` |
|---|---|---|---|
| unpatched | pre-fix | ✓ no errors | compiles; runtime `effect 'IO' requires capability` (expected without `--caps`) |
| unpatched | **fixed** | ✓ no errors | compiles; same expected capability error — **no regression** |
| patched | pre-fix | ✗ `Missing effects: FS` | ✗ same, before reaching runtime |
| patched | **fixed** | **✓ no errors** | **compiles**; same expected capability error |

The bottom row is the acceptance criterion: the extraction that #1091 blocked now compiles and
runs. The capability error is the program asking for `--caps IO,FS`, not a compile failure.

One false alarm worth recording: an early run showed a `type unification failed` error in
`epub_parser` on the *patched* tree with **both** binaries. That was a **stale compile cache** in a
scratch copy that had been warmed before the patch — not a code defect. All results above were
re-measured on trees with `docparse/.ailang/` removed. Controls are in the table: adding the
module alone (importing only the non-triggering `pkgDropSpans`) compiles clean pre- and post-fix.

### Gates

`make test` (exit 0), `make lint` (0 issues), `make fmt-check`, `make check-boundaries`,
`make check-file-sizes`, `make verify-examples` — all green.

Baseline note: the *first* `make test` of the session failed in `internal/smt`
(`TestSolve_HardTimeout_FakeSolverIgnoringT`), a pre-existing startup-race flake that
self-documents as `ailang#602`. It passes standalone and passed in both subsequent full runs.
Unrelated to this change (different package, no effect-system surface).

### Windows-safety scan (rule #10)

The new test file: no path assertions (it compares `RowVars` and error presence, never rendered
paths); no external binaries (no z3); no golden files. `findStdDir` uses `filepath.Join`/`Dir` and
`filepath.Walk`, and `strings.HasSuffix(p, ".ail")` is separator-independent. `t.TempDir()` handles
its own cleanup. No Windows-specific risk identified.
