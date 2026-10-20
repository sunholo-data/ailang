# Effectful callback passed to std/list `map`: error blames the import collision, not the effect

- **Date**: 2026-10-19
- **Class**: bug
- **Recommend**: design-doc
- **Searched**: "collisionHint", "also exported", "effect mismatch", "effect row hint", "callback effect", "pure function" across design_docs/ (excluding archive); read `design_docs/implemented/v0_28_0/m-length-collision-dx.md`; inspected `internal/types/import_hint.go`, `typechecker_app_diagnostics.go`, `unification_types.go`, `row_unification.go`, `std/list.ail`. Also grep'd `ailang-core-backlog.md` for effect-unification rows — none found.
- **Estimate**: omitted (design-doc route)

**Why (mechanism, verified in code, repro confirmed on dev v0.54.15 / 0384450):**
`collisionHint` in `internal/types/import_hint.go` is attached by `applicationDiagnosticPath`
(`internal/types/typechecker_app_diagnostics.go`) to **every** failing application constraint whose
callee is a bare name exported by ≥2 stdlib modules — it never inspects *why* the application failed.
So when the real failure is an effect-row mismatch (`map`'s callback param is checked against a pure
closed row; the lambda performs IO), the leading note still sends the reader to alias-import
`std/option`'s `map`, whose callback has the same pure restriction — the suggested fix cannot fix
anything. The actual cause surfaces only in the tail, in solver vocabulary: `failed to unify
parameter 0` (`unification_types.go`) wrapping `incompatible closed rows: r1 has extra labels [],
r2 has extra labels [IO]` from `UnifyRows` (`row_unification.go`). Reproduced verbatim on dev.

**Why this is not a direct fix (despite the report locating the mechanism):**
(a) The change spans ≥3 files plus tests: an effect-failure classification (sentinel error or kind
tag) must originate in `row_unification.go`/`unification_types.go`, the hint selection in
`typechecker_app_diagnostics.go` must consult it (suppress `collisionHint` on effect failures), and
the new effect hint text lives in `import_hint.go`. (b) There is more than one acceptable design:
suppress-only (remove the misleading note) vs. a positive effect-aware hint that names the callback
argument; and where the "effect mismatch on a callback parameter" classification lives (row layer vs.
app-solver layer — note `typechecker_effect_row_issue386.go` already splits app effect-row logic out
of the solver, so there is precedent for the latter). (c) The hint wording is a stdlib-fact question:
the report proposes "use a foldl/forEach that accepts effects", but std/list already exports
**`mapE`** (`std/list.ail:205`, plus `filterE` at :220) which is the actual effectful-map remedy —
there is no `forEach`, and `foldl` is `pure`. The doc must verify the suggested remedies against the
real inventory (and arguably generalize to `filter`/`foldl`/`sortBy`, whose callbacks have the same
pure restriction). The prior art `m-length-collision-dx.md` deliberately kept the collision note
dumb (path-attached, cause-blind); making it cause-aware is an extension of that contract, worth a
reviewer's sign-off rather than a hot patch.

**Relation to prior art:** `design_docs/implemented/v0_28_0/m-length-collision-dx.md` introduced
`collisionHint` (Felix ticket fb_942b7f) and is the doc a design must extend — not a duplicate, since
that doc's contract is "attach on any failed application" and this report is precisely about that
contract mis-firing.
