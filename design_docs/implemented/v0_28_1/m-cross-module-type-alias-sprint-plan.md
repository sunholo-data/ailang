# Sprint Plan: M-XMOD-ALIAS

## Summary

Make transparent type aliases to **non-record** types (`type Row = Json`, `type UserId = int`,
`type Handler = (Req) -> Resp`) expand across module boundaries, closing the last gap in cross-module
alias transparency. The record half was fixed in M-TYPE-ALIAS (v0.9.11) and M-TRANSITIVE-ALIAS-ENV-IMPORT
(v0.22.0); this is the non-record analog. The fix is a **~6-line `*ast.TypeAlias` case** in the interface
builder that mirrors code the elaborator already runs, plus a regression-test pack derived from the design
doc's Conflict Surface.

**Design doc:** [m-cross-module-type-alias.md](m-cross-module-type-alias.md)
**Duration:** ~0.5–1 day (~4–6 hours)
**Target:** v0.28.1 (patch bug fix; v0.28.0 already shipped)
**Dependencies:** None — design doc is locked, no schema changes, digest-neutral.
**Risk Level:** Low — additive (only widens the imported alias env); dispatch on a disjoint AST node makes
it impossible to alter nominal ADTs/newtypes/records; no iface digest churn → no dependent cascade rebuilds.

## Current Status Analysis

### Completed recently (context / comparable velocity)

- ✅ `f101121a4` — fix(types): name-collision hint on failed application — surgical single-session type-system fix.
- ✅ `d85bef292` — feat(iface): render record-type fields in `iface --compact` — same file family (`internal/iface/`).
- ✅ `9195f525e` — fix(iface): compact interface carries ADT constructor fields — `internal/iface/` edit + tests.
- ✅ **Closest analog:** M-TRANSITIVE-ALIAS-ENV-IMPORT (v0.22.0, `8e3d2d30`) — the **record** sibling of this bug.
  ~15 LOC implementation + ~167 LOC test, ~4 hours actual, Low risk, additive. This sprint mirrors its shape.

### Velocity

Recent `internal/iface/` + `internal/types/` fixes are consistently small, single-session (≤ ~50 LOC impl +
a focused test). The core change here is **smaller** than the analog (~6 LOC vs ~15); the bulk of the effort
is the 6-fixture regression pack (~150–200 test LOC). Total ~0.5–1 day is realistic with a buffer.

### Remaining from design doc

- ⏳ M1: add `*ast.TypeAlias` case to the interface builder — ~6 LOC
- ⏳ M2: regression-test pack (5 Conflict-Surface fixtures + digest-stability) — ~150–200 LOC
- ⏳ M3: real-world validation (duckdb `type Row = Json` restored) + CHANGELOG + doc housekeeping — runtime/docs only

## Proposed Milestones

### Milestone 1: Register non-record aliases in the interface builder

**Goal:** Export the transparent expansion of `*ast.TypeAlias` declarations into `iface.TypeAliases`, so the
importing module's unifier can expand `TCon("Row") → Json` — exactly as the defining module already does.

**Estimated:** ~6 LOC implementation (test in M2)
**Duration:** ~30 min

**Tasks:**

- Edit [internal/iface/builder.go](internal/iface/builder.go) — add a sibling `if` immediately after the
  existing `*ast.RecordType` alias block ([builder.go:388](internal/iface/builder.go#L388)), inside the
  `if typeDecl.Exported {` body:
  ```go
  // M-XMOD-ALIAS: register transparent aliases to NON-record types
  // (e.g. `type Row = Json`, `type UserId = int`, `type Handler = (Req) -> Resp`)
  // so they expand across module boundaries. Mirrors the elaborator's
  // *ast.TypeAlias handling in elaborate/file_funcs.go:220. Record aliases
  // are already covered above via *ast.RecordType; sum types and nominal
  // newtypes are *ast.AlgebraicType and are intentionally NOT matched here.
  if aliasDef, ok := typeDecl.Definition.(*ast.TypeAlias); ok {
      if target := astTypeToInternalType(aliasDef.Target); target != nil {
          iface.AddTypeAlias(typeDecl.Name, target)
      }
  }
  ```
- `make quick-install`.
- Smoke-test: restore `type Row = Json` in a scratch 2-module package (or use the design-doc repro) and
  confirm it type-checks clean under `AILANG_RELAX_MODULES=1 ailang check`.

**Acceptance Criteria:**

- [ ] Cross-module `type Row = Json` repro (design doc § Symptom) type-checks — no `Json vs Row` error.
- [ ] `make test` shows zero **new** failures vs the pre-fix baseline (record any pre-existing baseline failures).
- [ ] `make verify-examples` unchanged vs baseline.
- [ ] `go vet` + `gofmt` clean on `internal/iface/builder.go` (run `make lint` if golangci-lint present).

**Risks:**

- Chained aliases (`type B = A`, `A = Json`) require the unifier's `expandAlias` to compose the new entries.
  *Mitigation:* covered by M2 fixture 5; if it fails, the fix is to seed both entries (both already flow
  through `ImportedTypeAliases`), not new machinery.

### Milestone 2: Regression-test pack (Conflict Surface → tests)

**Goal:** Turn the design doc's 5 Conflict-Surface fixtures + the digest-stability claim into locked tests.
The positive fixture must FAIL on pre-M1 HEAD and PASS with M1; the negative (nominal) fixtures must PASS
both before and after (guarding that nominal types stay nominal).

**Estimated:** ~150–200 LOC
**Duration:** ~2–3 hours

**Tasks:**

- Add `internal/iface/xmod_alias_test.go` (new file — keeps `builder.go` test surface focused; check existing
  `builder_test.go` size first and extend it instead if < 600 LOC). Use the existing multi-module iface test
  harness pattern (see `internal/iface/*_test.go`) or a `tests/` fixture package if a full pipeline run is needed.
- **Fixture 1 (positive, fail-before/pass-after):** module A `type Row = Json` + `type QR = { rows: [Row], … }`;
  module B builds a `QR` from `[Json]`. Assert type-checks; assert the pre-M1 error string `Json vs Row` is gone.
- **Fixture 2 (positive coverage):** non-record targets — `type UserId = int`, `type Res = Result[int, string]`,
  `type Pair = (int, string)`, `type Pred = (int) -> bool`, `type Names = [string]` — each imported and used
  cross-module in a value position. Assert all type-check.
- **Fixture 3 (negative / non-regression):** `type Gen = Gen(int)` in A; B must **not** be able to unify a `Gen`
  with a bare `int`. Assert the mismatch is STILL an error (this is `*ast.AlgebraicType` → stays nominal).
- **Fixture 4 (negative / non-regression):** `type Color = Red | Green` in A; `match` in B. Assert nominal
  behavior intact (constructors resolve; no structural unification introduced).
- **Fixture 5 (record aliases unchanged):** re-run / mirror the M-TYPE-ALIAS record case (`type Usage = {…}`)
  and M-STREAM-DX single-ctor-record field access — assert still passing.
- **Fixture 6 (chained alias):** `type A = Json`, `type B = A` in X; use `B` cross-module in Y; assert `B`
  expands transitively to `Json`.
- **Digest-stability test:** build a module's iface with and without a non-record alias added; assert
  `iface.Digest` (via `computeDigest`) is identical — locks the "no cascade rebuild" claim.

**Acceptance Criteria:**

- [ ] Fixture 1 FAILS on `dev` HEAD without M1 (verified by temporarily reverting M1), PASSES with M1.
- [ ] Fixtures 3 & 4 (nominal non-regression) PASS both with and without M1.
- [ ] Fixtures 2, 5, 6 PASS with M1.
- [ ] Digest-stability test passes (alias addition is digest-neutral).
- [ ] `make test` green (no new failures).

**Risks:**

- If the existing iface test harness can't exercise a full cross-module import, fall back to a `tests/`
  fixture package driven through the pipeline (as M-TRANSITIVE-ALIAS-ENV-IMPORT's repro did). Budget is in
  the 2–3h estimate.

### Milestone 3: Real-world validation + docs housekeeping

**Goal:** Prove the fix retires the package-level workaround and record the change.

**Estimated:** runtime + docs only (~10 LOC docs)
**Duration:** ~45 min

**Tasks:**

- **Real-world check:** in a local checkout of `sunholo/duckdb`, temporarily restore `types.ail` to
  `export type Row = Json` + `QueryResult.rows: [Row]` and confirm `ailang check` passes with the M1 binary
  (do **not** re-publish — 0.1.1 keeps the workaround as-is; this only proves the core fix subsumes it).
- **CHANGELOG:** add an entry under the active changelog (`changelogs/v0.10-current.md`) — group under a
  bug-fix/type-system heading; reference M-XMOD-ALIAS and the duckdb/eparse trigger.
- **Design doc:** on completion, `move_to_implemented.sh m-cross-module-type-alias v0_28_1` and add an
  implementation report (actual LOC, test file, before/after). Move this sprint plan alongside it.
- **No new example file required:** this is a cross-module type-checking fix, not a surface language feature;
  the regression fixtures are the executable evidence. (Note the deviation explicitly per repo docs policy.)

**Acceptance Criteria:**

- [ ] duckdb `type Row = Json` variant type-checks under the M1 binary (transcript captured in the impl report).
- [ ] CHANGELOG updated.
- [ ] Design doc + sprint plan moved to `implemented/v0_28_1/` with an implementation report.

## Success Metrics

- **Correctness:** cross-module non-record aliases unify; nominal ADTs/newtypes provably unchanged (fixtures 3/4).
- **No cascade churn:** iface digest identical (digest-stability test).
- **Zero regressions:** `make test` + `make verify-examples` at baseline.
- **Test coverage:** ≥ 6 new fixtures; the positive one is a verified fail-before/pass-after guard.

## Dependencies & Open Questions

- **Dependencies:** none. Elaborator, unifier (`expandAlias`), and `ImportedTypeAliases` plumbing already exist.
- **Out of scope (tracked in design doc):** parameterized non-record aliases (`type Pair[a] = (a, a)`) — needs
  type-param substitution at expansion; file as `M-XMOD-ALIAS-POLY` if a real case appears.
- **Open question:** confirm the REPL path (`internal/repl/module_registry_load.go`, which already threads
  `TypeAliases`) benefits automatically once the builder populates them — verify in M2 or note as a follow-up.

## Handoff

Not yet handed off to sprint-executor — awaiting user approval. On approval:
`ailang messages send sprint-executor` with `plan_ready` + progress JSON path.
