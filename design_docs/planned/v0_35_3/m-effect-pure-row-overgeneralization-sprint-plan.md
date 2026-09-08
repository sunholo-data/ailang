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
- [ ] `internal/types/effect_row_defaulting_test.go` — Scheme-level assertions:
  - recursive `pure` binding ⇒ `RowVars == nil/[]` and a closed effect row (currently RED)
  - declared `! {e}` binding ⇒ `RowVars == ["e"]` preserved (currently GREEN — pins V14)
  - recursive `! {IO}` binding ⇒ closed `{IO}` (currently GREEN — pins V9)
- [ ] `internal/pipeline/validate_effects_xmod_test.go` — two-module fixtures:
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
- [ ] Add effect-row defaulting alongside the existing type-class defaulting pass in the LetRec
      path (`typechecker_functions.go:330-377`) and the corresponding Let/top-level binding path
- [ ] Resolve the binding's declared effect row; bind unresolved effect-row variables to it when
      that declared row is closed
- [ ] Leave declared row-polymorphic signatures untouched — discriminate on **the declaration**,
      never on the row variable's name (`ρN` vs `e`); the naming shortcut breaks the first time a
      user names a row `rho` (Design Freeze)
- [ ] Default only *unresolved variables*, never concrete labels — a genuine effect must still
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
- [ ] `make test`, `make lint`, `make verify-examples`
- [ ] Reconstruct the `ailang-parse` extraction in a scratch copy (`160c7f1` + extract
      `pkgLastIndexOf`/`pkgDropSpans` and their private scans into `docparse/services/pkg_template`),
      then run `ailang check docparse/services/docx_template.ail` **and**
      `ailang run docparse/main.ail --entry main` — both must pass with no `.ail` edits
- [ ] Sweep `std/` for any `pure`-declared export still carrying a quantified effect row
- [ ] Confirm `std/list.mapE` still shows `RowVars=[e]`
- [ ] CHANGELOG.md entry under v0.35.3

**Acceptance criteria:**
- Every design-doc Success Criterion checked
- The `ailang-parse` end-to-end run is recorded in the sprint notes with its actual output — this
  milestone cannot be closed on unit tests alone

**Risks**: the reconstruction is manual — mitigated by scripting it so it is repeatable, and by
requiring the pre-fix baseline to fail first.

## Success Metrics

- [ ] #1091 reproduction passes `check` and `run` with no `.ail` source edits
- [ ] `std/list.mapE`/`filterE`/`foldlE` keep `RowVars=[e]`
- [ ] No `pure`-declared export in `std/` carries a quantified effect row
- [ ] Genuine cross-module effect requirement still rejected (V9)
- [ ] `make test` green, no existing expected-text changes
- [ ] `make verify-examples` green
- [ ] CHANGELOG.md updated

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
