# M-SMT-CROSS-MODULE-TYPES — Sprint Plan

**Source design doc**: [m-smt-cross-module-types.md](m-smt-cross-module-types.md)
**Target version**: v0.14.3 (current: v0.14.2; next patch)
**Priority**: **P1 / urgent** — blocks downstream apps (docparse: 3/28 verified; ailang-parse `tex_parser.ail`: 18 of 19 `ensures` clauses skipped)
**Estimated**: 22 hours (~3 working days, with 2-hour buffer = 24h matching design doc estimate)
**Status**: Phase 1 (demand-driven filtering) **DONE** in v0.9.x; this sprint covers Phase 2, Phase 3, and the small Issue 7 follow-up.

## Why now

Two concrete blockers, both rolling today:

1. **docparse** — 3/28 functions verify; should be 15+. Cross-module record aliases and recursive ADTs cause cascade failures.
2. **ailang-parse / tex_parser.ail** — 18 of 19 `ensures` clauses SKIPPED. Bug message `ce6e078e` (2026-04-14) confirms the headline Z3 verification story does not apply to realistic multi-module parser code.

Plus this is the **hard prerequisite for M-TAINT-TYPES Phase 2** (cross-module taint flow). Landing this unblocks both downstream users *and* the next strategic milestone.

## Scope

The design doc identifies seven issues. Phase 1 (issue 3, demand-driven filtering) already shipped. Remaining work covers:

| Issue | Severity | Milestone | Status pre-sprint |
|---|---|---|---|
| 1. Record type aliases not declared | Critical | M1 | not started |
| 2. Recursive ADT circular deps need `declare-datatypes` | Critical | M2 | not started |
| 3. Preamble pollution | High | — | DONE (v0.9.x Phase 1) |
| 4. Declaration ordering (topological sort) | Medium | M1 | partial (Phase 1 helps) |
| 5. Parameter-accessor name collisions (`$p_` prefix) | Low | M3 | not started |
| 6. Field name collisions across record types | Medium | M3 | not started |
| 7. String builtins not encodable (`trim`/`toLower`/...) | Medium | M4 | partial (skip messages exist; need named-builtin tagging) |

## Recent velocity (context for estimates)

Last 14 days saw ~20 commits across small-to-medium milestones (M-SUPPLY-CHAIN-HARDENING-2, M-CI-BUILD-SPEED, MCP onboarding work). Healthy throughput; this sprint slots in a 3-day window comfortably.

## Milestone breakdown

Five milestones, ordered to front-load downstream-user impact. M1 alone delivers most of the docparse improvement; the cumulative effect across M1–M3 hits the design doc's 15+ verified target.

---

### M1 — Named record type aliases + topological ordering

**Estimated**: 6 hours
**Issues addressed**: 1 (record aliases), 4 (ordering)
**Files**:
- `cmd/ailang/verify.go` — resurrect stashed `collectNamedRecordAlias`, add `RecordTypeAliases` map (~50 LOC)
- `internal/smt/codegen.go` — register field-set keys; emit aliases before dependents (~50 LOC)
- `internal/smt/codegen_records.go` — skip anonymous sort emission when a named alias matches the field set (~30 LOC)
- `internal/smt/codegen_records_test.go` — alias declaration order test (~50 LOC)

**Implementation outline**:
1. Resurrect `collectNamedRecordAlias()` from the stashed prototype; verify the existing partial fix still applies cleanly to current `verify.go`.
2. Add a topological pass after collection: aliases referencing other aliases or ADTs are emitted in dependency order. Use multi-pass with resolved-check (per Design Freeze item) — simpler than full Tarjan since cycles aren't expected here (those are M2's domain).
3. Register a field-set fingerprint (`{colSpan, merged, rowSpan, text} → TableCell`) so body literals like `{text: "x", ...}` resolve to the named sort instead of an anonymous record.
4. Update `codegen_records.go` to skip emitting anonymous `Record_*_*` sorts when a named alias covers the same field set.

**Acceptance criteria**:
- [ ] Synthetic test: `type Point = {x: int, y: int}` produces `(declare-datatype Point ((mk_Point (x Int) (y Int))))` in Z3 output
- [ ] Synthetic test: `type ParsedDocument = {metadata: DocMetadata, ...}` is declared *after* `DocMetadata`
- [ ] **docparse `simpleCell`, `spanCell`, `mergedCell`, `emptyMetadata`, `headingLevelFromStyle` all VERIFIED** (currently ERROR)
- [ ] **docparse `format_router.ail` maintains 3/3 VERIFIED** (no regression)
- [ ] All existing AILANG tests pass
- [ ] `make lint` clean

**Test commands**:
```bash
go test ./internal/smt/ -count=1
ailang verify ../docparse/types/document.ail        # expect simpleCell, spanCell, mergedCell, emptyMetadata VERIFIED
ailang verify ../docparse/services/format_router.ail # expect 3/3 VERIFIED (no regression)
ailang verify ../docparse/services/docx_parser.ail   # expect headingLevelFromStyle VERIFIED
```

**Pause point**: After M1, run the full docparse `ailang verify` sweep and confirm verified count climbs from 3 → ~10. **STOP and report** before starting M2.

---

### M2 — Mutual recursion via `declare-datatypes` (plural)

**Estimated**: 6 hours
**Issues addressed**: 2 (recursive ADT circular deps)
**Files**:
- `internal/smt/codegen_mutual.go` (NEW) — Tarjan's SCC + `declare-datatypes` emission (~150 LOC)
- `internal/smt/codegen_mutual_test.go` (NEW) — synthetic recursive ADT tests (~100 LOC)
- `internal/smt/codegen.go` — wire SCC results into the type-emission pipeline (~30 LOC)

**Implementation outline**:
1. Build a type dependency graph from collected ADTs + record aliases (reuse M1's collection; do not duplicate).
2. Run Tarjan's SCC algorithm (per Design Freeze decision; alternative was manual annotation, rejected as error-prone). Output: list of SCCs, each either a singleton (use existing `declare-datatype` singular) or a mutual-recursion group (use `declare-datatypes` plural).
3. Implement `DeclareDatatypesMutual(group []TypeDecl)` emitting the Z3 syntax shown in design doc Example 3.
4. Update `codegen.go` to dispatch on SCC size: singleton → existing path; group → `DeclareDatatypesMutual`.

**Acceptance criteria**:
- [ ] Synthetic test: `type T = A({x: T}) | B(int)` (self-recursive ADT with inline record) emits `declare-datatypes` with both `T` and `Record_x` in one block
- [ ] Synthetic test: `type Block = TextBlock(...) | SectionBlock({blocks: [Block]})` (the docparse pattern) emits the expected mutual-recursion encoding
- [ ] **docparse `countBlocks` VERIFIED** (currently ERROR — recursive ADT)
- [ ] All existing AILANG tests pass — especially: existing non-recursive ADT examples (access_control, finance, etc.) still produce `declare-datatype` (singular), unchanged
- [ ] `make lint` clean

**Test commands**:
```bash
go test ./internal/smt/ -run "TestMutual|TestSCC" -count=1
ailang verify ../docparse/services/docx_parser.ail   # expect countBlocks VERIFIED
for f in examples/runnable/contracts/*.ail; do ailang verify "$f" 2>&1 | tail -2; done  # no regressions
```

**Pause point**: Verify no regression in existing contract examples. **STOP and report** verified counts for docparse before M3.

---

### M3 — Parameter prefix + qualified field accessors

**Estimated**: 4 hours
**Issues addressed**: 5 (parameter/accessor collision), 6 (field name collision across record types)
**Files**:
- `internal/smt/codegen.go` — `$p_` parameter prefix in `declare-const`, qualified accessor emission (~60 LOC)
- `internal/smt/codegen_records.go` — emit `RecordType_fieldName` accessor names; update `encodeRecordAccess` to match (~30 LOC)
- `internal/smt/encodable.go` (or wherever `EncodeExpr` resolves vars) — variable lookup uses prefixed names (~20 LOC)
- `internal/smt/codegen_test.go` — regression tests for both collisions (~80 LOC)

**Implementation outline**:
1. Add `$p_` prefix to parameter `declare-const` names. Update `EncodeExpr` variable resolution to accept either the bare name (lookup) and emit the prefixed form (output). Two-table approach: bare-name → prefixed-name on first reference.
2. Qualify record field accessor names: `(declare-datatype CheckResult ((mk_CheckResult (CheckResult_applicable Bool) ...)))`. Update `encodeRecordAccess` to emit `(CheckResult_applicable r)` instead of `(applicable r)` when `r` is known to be a `CheckResult`.
3. Update existing test expectations for the new accessor names (mechanical sweep over `internal/smt/*_test.go`).

**Acceptance criteria**:
- [ ] Synthetic test: `func f(text: string) -> TableCell` (where `TableCell` has a `text` field) emits `(declare-const $p_text String)` and uses `$p_text` in the body, no Z3 ambiguity error
- [ ] Synthetic test: two record types with shared field `applicable` (one Bool, one Int) emit `CheckResult_applicable` and `MetaAccum_applicable`, no collision
- [ ] **docparse `evalComputeScore` VERIFIED** (currently ERROR — Issue 6)
- [ ] All existing tests pass — especially: any test pattern matching `(declare-const text` or `(applicable ` is updated to match the new prefixed/qualified forms
- [ ] `make lint` clean

**Test commands**:
```bash
go test ./internal/smt/ -count=1
ailang verify ../docparse/services/eval.ail   # expect evalComputeScore VERIFIED
go test ./... -run "TestVerify|TestEncode" -count=1   # full regression
```

**Pause point**: Sweep `internal/smt/*_test.go` for any string assertions on accessor or parameter names; update mechanically. **STOP and report** if any test failures persist.

---

### M4 — Skip-reason tagging for unencodable builtins

**Estimated**: 2 hours
**Issues addressed**: 7 (string builtins) — partial fix per design doc §"Tracking"
**Files**:
- `internal/smt/encodable.go` — surface which builtin caused the rejection in the skip message (~30 LOC)
- `cmd/ailang/verify.go` — pretty-print the skip reason in the user-visible verify output (~20 LOC)
- `internal/smt/encodable_test.go` — tests for the surfaced reason (~40 LOC)

**Implementation outline**:
1. When `encodable.go` rejects a function due to an unencodable builtin (`_str_trim`, `_str_upper`, `_str_lower`, `_str_split`, `_str_charAt`), include the builtin name in the structured skip reason.
2. Update `verify.go` to print the named builtin in the skip line: `! SKIPPED myFunc — unencodable builtin: trim`. Today's message is generic ("string operations not encodable").
3. This is **NOT** the full string-theory fix (that's a separate `m-smt-string-theory.md`, deferred). This sprint just makes the skip messages actionable so users can narrow contracts or refactor.

**Acceptance criteria**:
- [ ] tex_parser.ail SKIP messages name the specific blocking builtin (e.g. "unencodable builtin: trim")
- [ ] Functions with no unencodable builtins still verify normally
- [ ] All existing tests pass
- [ ] `make lint` clean

**Test commands**:
```bash
go test ./internal/smt/ -count=1
ailang verify ../ailang-parse/tex_parser.ail 2>&1 | grep SKIP   # expect named builtin per skip
```

**Pause point**: None — small enough to roll directly into M5.

---

### M5 — End-to-end validation + downstream report + example + CHANGELOG

**Estimated**: 4 hours
**Issues addressed**: All — verification that the sprint hits its acceptance metrics
**Files**:
- `examples/runnable/contracts_cross_module.ail` (NEW) — minimal cross-module record-alias + recursive-ADT example (~80 LOC across 2-3 files)
- `CHANGELOG.md` — entry referencing this sprint plan and design doc
- `cmd/ailang/prompts/v0.14.3.md` (or current) — short note that cross-module record types now work in `ensures`

**Implementation outline**:
1. End-to-end run: `ailang verify` over the entire docparse project. Confirm 15+ verified (design doc target).
2. End-to-end run: `ailang verify` over ailang-parse `tex_parser.ail`. Confirm SKIP messages are now actionable (named builtins per M4). The 18 of 19 won't all verify — many genuinely need string theory — but the user should now know exactly which are blocked on string builtins vs other reasons.
3. Add the canonical example `examples/runnable/contracts_cross_module.ail` (plus 1-2 supporting modules) demonstrating: record alias, recursive ADT, cross-module import, all verifying.
4. CHANGELOG entry under v0.14.3, referencing sprint plan + design doc.
5. Send a `pkg-feedback` message to docparse and ailang-parse maintainers reporting the unblock.

**Acceptance criteria**:
- [ ] **docparse: 15+ verified functions** (design doc target hit)
- [ ] **docparse `format_router.ail`: 3/3 VERIFIED** (no regression — confirmed at every milestone, final check here)
- [ ] tex_parser.ail SKIP messages all name the specific blocking builtin
- [ ] `examples/runnable/contracts_cross_module.ail` verifies clean (and is wired into `make verify-examples`)
- [ ] CHANGELOG entry merged
- [ ] Feedback messages sent via `ailang messages send` to docparse and ailang-parse

**Test commands**:
```bash
# Final regression sweep
make ci

# Downstream verification
ailang verify ../docparse/  # full project sweep, count VERIFIED lines
ailang verify ../ailang-parse/tex_parser.ail

# Example
ailang verify examples/runnable/contracts_cross_module.ail
```

**Pause point**: This is the final milestone — checkpoint and ship.

---

## Dependencies and risks

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|-----------|
| Stashed Phase 2 prototype no longer applies cleanly to current `verify.go` | Med | Med | M1 has 6h budget — apply manually if rebase fails. Don't sink time into auto-merge. |
| Tarjan's SCC implementation has subtle bugs on edge cases (single-node cycles, disconnected components) | Med | High | Use a vetted Go SCC library if available; otherwise hand-test on docparse-mirror synthetic types before wiring through codegen. |
| `$p_` prefix or qualified-accessor change breaks tests in places far from `internal/smt/` | Med | Med | Run `grep -r "(declare-const " internal/` before M3 to inventory test expectations; sweep mechanically. |
| docparse maintainers rely on the current generic SKIP messages | Low | Low | M4 messages are strictly more informative; backwards-compatible at the sprint level. |
| Sprint stretches past 24h | Med | Low | M4 + part of M5 can be cut to a follow-up if M1-M3 hit budget. The downstream-user goal (15+ verified) is hit at end of M3. |

## Success metrics (sprint-level)

- **Headline**: docparse goes from 3/28 → 15+/28 verified (~5x improvement)
- **No regression**: `format_router.ail` still 3/3; all existing contract examples produce identical outcomes
- **Actionable skips**: every `SKIP` from the verifier names the specific blocker
- **Unblock M-TAINT-TYPES Phase 2**: cross-module label flow is now structurally possible (Phase 2 of M-TAINT-TYPES becomes implementable)
- **Test coverage**: ~250 LOC new tests (dependency graph, mutual recursion, name disambiguation)

## Pause points summary

| After | What to verify | Action |
|---|---|---|
| M1 | docparse verified count climbs to ~10 | Report and wait |
| M2 | No regression in existing contract examples | Report and wait |
| M3 | All test sweeps green | Report and wait |
| M4 | (none — folds into M5) | Continue |
| M5 | docparse 15+ verified; example added; CHANGELOG; feedback sent | Sprint complete |

## Handoff status

**Sprint plan ready for review.** No auto-handoff to sprint-executor — the user has a pattern of reviewing docs before approving execution. Once the user gives the go-ahead, run `sprint-executor` against this plan and the JSON progress file.

---

**Document created**: 2026-04-30
**Last updated**: 2026-04-30
