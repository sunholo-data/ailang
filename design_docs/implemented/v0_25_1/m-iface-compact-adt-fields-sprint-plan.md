# Sprint Plan: M-IFACE-COMPACT-ADT-FIELDS

**Design doc:** [m-iface-compact-adt-fields.md](m-iface-compact-adt-fields.md)
**Target:** v0.25.1 · **Priority:** P1 · **Risk:** low-medium · **Estimate:** ~1 day (8h)

## Summary

Make `ailang iface --compact` carry ADT constructor field signatures and render record types,
so the AST-iface-compaction feature is usable for ADT-construction tasks. Three milestones, each
independently testable, executed in order (M2 depends on M1's renderer; M3 depends on M2's JSON).

**Goal:** `type Shape = Circle({radius: float}) | Rect({w: float, h: float})` and no `<*types.TRecord>`.

## Velocity note
Self-contained type-rendering + plumbing change; comparable to prior single-day iface/type fixes.
Conservative: 8h with a 30% buffer already folded into the per-milestone estimates.

## Milestones

### M1 — Render records in `formatTypeCanonical` (cycle-safe)
- **Files:** `internal/iface/json.go`
- **LOC:** ~25 impl + ~30 test
- **Tasks:**
  - [ ] Add `case *types.TRecord` → `{label: type, ...}` (sorted labels), reusing `formatTypeCanonical` recursively.
  - [ ] Cycle-safety: thread a `visited`/depth guard (records can be recursive); never call `.String()`.
  - [ ] Open/row-polymorphic records: render known labels + an explicit open-row marker.
- **Acceptance:**
  - A record-returning exported function renders `{...}`, not `<*types.TRecord>`.
  - A recursive-record type renders without hanging (bounded).
- **Dependencies:** none

### M2 — Thread `FieldTypes` into the iface + JSON
- **Files:** `internal/iface/builder.go`, `internal/iface/json.go`
- **LOC:** ~20 impl + ~10 test
- **Tasks:**
  - [ ] Add `FieldTypes []types.Type` to iface `ConstructorInfo` (`builder.go`).
  - [ ] Populate from elaborate `ConstructorInfo.FieldTypes` at the iface-build site.
  - [ ] Extend `TypeJSON` ctor entries to `{name, fields}` (fields = rendered via M1); keep deterministic ordering.
- **Acceptance:**
  - `ailang iface` JSON for the test ADT includes each ctor's field types.
- **Dependencies:** M1 (uses the renderer)

### M3 — Compact rendering + tests + golden regen
- **Files:** `cmd/ailang/check.go`, `internal/iface/constructor_test.go`
- **LOC:** ~12 impl + ~40 test
- **Tasks:**
  - [ ] `compactInterface`: render `Ctor({...})` / `Ctor(t1, t2)` instead of bare name.
  - [ ] New tests: field rendering (record + positional ctors); record-return leak fixed.
  - [ ] Regenerate digest goldens; assert the same-ctors ⇒ same-digest / different-fields ⇒ different-digest invariants.
  - [ ] `make quick-install`; verify on the minimal ADT and on `docparse/types/document.ail`.
- **Acceptance:**
  - `ailang iface --compact` on the minimal ADT prints `type Shape = Circle({radius: float}) | Rect({w: float, h: float})`.
  - `grep -r '<\*' ` over regenerated stdlib ifaces returns nothing.
  - `make test` green (with new goldens).
- **Dependencies:** M2

## Success Metrics
- [ ] All three milestones' acceptance criteria pass.
- [ ] `make test` green; no `<*...>` in any iface output.
- [ ] CHANGELOG updated; design doc moved to implemented/v0_25_1 on completion.
- [ ] **Validation (post-rebuild):** re-run the docx eval; `cat`-fallback count < 3 (baseline 11).

## Conflict-surface guardrails (from design doc)
- Interface-hash churn for ADT/record modules is **expected and accepted** (one-time, semantically correct). Regenerate goldens; do NOT special-case to avoid the hash change.
- `formatTypeCanonical` is shared by ALL signatures — M1's test must cover non-ctor record returns too.

SPRINT_PLAN_PATH: design_docs/planned/m-iface-compact-adt-fields-sprint-plan.md
SPRINT_JSON_PATH: .ailang/state/sprints/sprint_M-IFACE-COMPACT.json
