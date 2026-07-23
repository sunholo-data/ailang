# Sprint Plan — M-FMT-PROPERTIES-PRINTER-ROUNDTRIP

**Design doc**: [m-fmt-properties-printer-roundtrip.md](m-fmt-properties-printer-roundtrip.md)
**Sprint JSON**: `.ailang/state/sprints/sprint_M-FMT-PROPERTIES-PRINTER-ROUNDTRIP.json`
**Target**: v0.30.0 (fmt Phase-1 correctness follow-up)
**Risk level**: LOW
**Estimated**: ~1 day (~7h) · ~200 LOC (impl + tests)
**Status**: UNPARKED — Mark 2026-07-20 ("yes lets finish off ailang fmt"), no re-quorum. Routed to executor.
**Related issue**: #399

## Summary

Two live-reproduced `ailang fmt` correctness bugs at HEAD `v0.30.0-35-ge77ec5b48`:

1. **Exit-2 defect**: any file using `requires`/`ensures` contract clauses fails `fmt --check`
   with exit 2 ("formatter defect"). The whole Z3-verified contract corpus is formatter-ineligible
   (30 files fail via CLI; the Phase-2 gate counts 28 as `preexisting-Phase1-rt-bug`).
2. **Silent contract deletion (data loss, exit 0)**: a function with BOTH contracts and a
   `properties [...]` block loses its `requires` clause on `fmt` because the parser *assigns*
   (rather than appends) at `parser_func.go:169`, clobbering the already-parsed contract entries.

Root cause: the parser stores contracts and forall properties in one slice
(`FuncDecl.Properties`) discriminated by `Kind`, but the printer ignores `Kind` and routes
everything through `properties [...]`, which the parser only accepts with a leading `forall`.

Fix: a **print-time split by `ContractKind`** (no AST refactor) plus a **one-line parser append
fix**. This faithfully translates the design doc's M1/M2 — no new scope.

## Premise verification vs repo reality (HEAD e77ec5b48)

All doc premises re-confirmed against the current tree (doc verified at `5afa9a1e1`; still holds):

| Claim | Verified |
|-------|----------|
| `parser_func.go:169` is `fn.Properties = p.parsePropertiesBlock()` (assignment clobber) | ✅ exact match |
| Contracts appended at `parser_func.go:124-125` before tests/properties parse | ✅ |
| `decl.go` `funcDecl` calls `testsAndProperties(d)` at line 80, after the effect row (77-79), before `funcBody` (84+) — the contract insertion point | ✅ |
| `testsAndProperties`/`propertiesBlock`/`property` route all `d.Properties` through `properties [` with no `Kind` reference | ✅ (decl.go:184-304) |
| `corpus_comment_test.go` gate at line 122 excludes `preExistingRT` from `t.Fatalf` (tolerated) | ✅ (V16) |
| Corpus size | ✅ 31 files in `examples/runnable/contracts/`, 1 in `examples/ai_devtools_workflow/` |

**Premise corrections found**: none. The design doc is accurate against the current HEAD. The only
drift is cosmetic — the doc's Verification Log cites `5afa9a1e1` / `-24-g`; HEAD is now
`e77ec5b48` / `-35-g`, but every cited line number and behavior is unchanged.

## Milestones

### M1 — Printer emission split + parser clobber fix (~4h, ~195 LOC)

**Files**: `internal/format/decl.go` (~40 LOC), `internal/parser/parser_func.go` (1 line),
`internal/format/` test file (~150 LOC).

- In `funcDecl`, after the effect row, partition `d.Properties` by `Kind`; emit one
  `requires { p1, p2, ... }` clause (RequiresKind) and one `ensures { ... }` clause (EnsuresKind)
  in signature position, before `testsAndProperties`/body, predicates comma-separated in slice order.
- Refactor `testsAndProperties` to accept a **pre-filtered PropertyKind-only** `[]*ast.Property`
  so it structurally cannot re-emit contract clauses (gemini nit).
- `parser_func.go:169`: `=` → `append(fn.Properties, p.parsePropertiesBlock()...)` (explicit
  slice unpacking, gemini nit) — preserves contracts when a properties block follows.
- 7 round-trip unit fixtures (a–g): requires-only / ensures-only / both / multi-predicate /
  ForallExpr-predicate / synthetic genuine `forall(...)` properties block / contracts+properties
  combined. Each asserts parse→print→re-parse→`cmp.Diff` AST identity + idempotence.

**Acceptance**: see JSON `M1_PRINTER_SPLIT_AND_PARSER_APPEND.acceptance_criteria`. Key gates —
minimal `cf_contract.ail` never exits 2; combined file keeps its `requires`; `go test
./internal/format/ ./internal/parser/` green.

**Deferred to executor's choice** (doc §Deferred): clause indentation (column-0 vs one-level);
single-line vs multi-line multi-predicate clauses. Both re-parse identically — idempotence is the
only constraint.

### M2 — Regression guard + corpus green (~3h, ~100 LOC)

**Files**: `internal/format/testdata/contracts_and_properties.ail` (new fixture),
`internal/format/` (new `TestCombinedContractsAndPropertiesPipeline`),
`internal/format/corpus_comment_test.go` (~5 LOC gate hardening),
28 corpus `.ail` files (mechanical `fmt --write`), `CHANGELOG.md`.

- **Acceptance-gated integration test** `TestCombinedContractsAndPropertiesPipeline` over the
  combined fixture, four assertion groups (a) checks-clean + `ai-check` verify, (b) contract →
  `core.DeclMeta.Contracts` (exactly 1 Requires + 1 Ensures, no forall entry), (c) forall reaches
  only the property pipeline (`fn.Properties` exactly `[Requires, Ensures, Property]`; collector
  exactly one `PropertyCase` Kind==PropertyKind), (d) exact counts + no panic + round-trip identity
  with `requires` present.
- Harden `corpus_comment_test.go`: move `preExistingRT` into the `t.Fatalf` condition
  (`preExistingRT != 0` fails), driving `28 → 0` and locking it.
- `ailang fmt --write` on the 28 eligible files (dedicated commit to isolate churn); re-verify
  `ailang check` (+ `ai-check --json` on verify-corpus files) on every one.
- Acceptance sweep: zero exit-2 across `contracts/*.ail` + `ai_devtools_workflow/*.ail`, excepting
  exactly the 2 comment-attachment carve-out files (`inbox_injection_v2.ail`, `inbox_v2_app.ail`,
  unchanged error).
- `make test` green; `make verify-examples` green (manifest-drift is the expected red mode, not
  type regressions). CHANGELOG.md entry — **explicitly note the silent-deletion fix**.

**Acceptance**: see JSON `M2_REGRESSION_GUARD_AND_CORPUS_GREEN.acceptance_criteria`.

## Non-Goals (respect the doc — do NOT expand scope)

- No comment-attachment carve-out work (the 2 inbox files stay fail-closed).
- No Phase-2 comment features, no `properties [...]` grammar changes.
- No fmt adoption / CI formatting enforcement (that's m-ailang-fmt-adoption).
- **No AST refactor** splitting contracts out of `FuncDecl.Properties` — print-time split only.
- Only `internal/format/decl.go`, the one line `internal/parser/parser_func.go:169`, format test
  files, the 28 reformatted examples, and CHANGELOG are touched.

## Risks & Mitigations (from doc)

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Reformatting 28 checked-in examples churns diffs / breaks verify-examples manifest stats | Med | Dedicated reformat commit; re-run verify-examples; known failure mode is manifest drift, not type errors |
| A `.Properties` consumer assumed the clobber | Med | Consumer audit COMPLETE pre-approval (V17: 6 sites); locked by acceptance-gated M2 integration test |
| Emitted clause order/position fails re-parse in an untested corner | Low | fmt round-trip verifier is fail-closed (exit 2, never silent); add extern/equation-form variants if hit |
| Idempotence regression from new hardline placement | Low | Corpus idempotence check runs per-file automatically |

## Total estimate

- **Milestones**: 2 (M1, M2) — matching the doc's M1/M2 exactly.
- **Effort**: ~7h (M1 4h + M2 3h) → ~1 day.
- **LOC**: ~200 (M1 ~195 impl+fixtures, M2 ~100 test+gate+CHANGELOG; overlap in shared fixtures).
- **Risk**: LOW (printer-only + one parser line; fail-closed verifier is the safety net).

---

**SPRINT_PLAN_PATH**: `design_docs/planned/v0_30_0/m-fmt-properties-printer-roundtrip-sprint-plan.md`
**SPRINT_JSON_PATH**: `.ailang/state/sprints/sprint_M-FMT-PROPERTIES-PRINTER-ROUNDTRIP.json`
