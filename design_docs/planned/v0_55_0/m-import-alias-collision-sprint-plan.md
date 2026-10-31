# Sprint Plan: M-IMPORT-ALIAS-COLLISION

Design: [m-import-alias-collision.md](m-import-alias-collision.md). Issue: #1980 (P1).

## Summary and authorization

Deliver Phase 1: binding-time MOD016 for two selective type imports binding the same bare name from different import paths. Preserve MOD015 and the documented non-cases. Phase 2 use-site ambiguity, bulk imports, nominal identity and REPL merging are excluded.

The coordinator handoff states previous work is approved; that supersedes the design's stale pending human-approval checkbox for planning. This task authorizes planning only. Execution requires the separate execute-sprint gate. Independent quorum has not passed: the design records two attempts with all reviewers absent. Do not describe approval as a quorum verdict; resolve the recorded quorum caveat before execution or obtain an explicit waiver of that design prerequisite.

Duration: 3 working days, approximately 5–6 active hours plus corpus runtime/review. Four sequential milestones, each bounded to roughly 60–90 minutes active work. Risk: medium, because adding a namespace to existing collision code can accidentally change MOD015.

## Current status and estimates

Checkout source version: v0.54.19; planning HEAD: 5415c8c9. Working tree was clean. No MOD016 implementation is present. Existing import collision helpers and CLI/LSP tests can be extended. The seven-day velocity script finds only the design commit in available history and no usable recent LOC metrics; measured LOC/day is unavailable. Use a conservative planning capacity of 137 LOC/day, not an observed velocity. Total: 410 LOC (roughly 110 implementation, 230 tests/fixtures, 70 documentation). Investigation and corpus comparison dominate elapsed time; estimates are not throughput claims.

AILANG prompt version loaded: v0.16.7 (`ailang prompt`, active CLI version). The installed binary reports stale relative to v0.54.19 source. Executor must build this checkout and reload its teaching prompt before writing or checking fixtures; use matching stdlib for both baseline and candidate.

## Registry reuse audit

`ailang pkg search 'import collision'` returned no packages. This is compiler-internal namespace classification and diagnostics, not a package-like capability that an AILANG library can supply. M1–M4 each use decision `none`; reuse the existing Go elaborator, structured error, CLI and LSP infrastructure. No candidate packages warrant info/docs inspection.

## Milestones

### M1: Register MOD016 and preserve MOD015 namespaces (~80 LOC)

**Dependencies:** none

**Files:** `internal/errors/codes.go`, `internal/errors/codes_test.go`, `internal/elaborate/import_collision.go`; add focused regression tests in `internal/elaborate/import_type_collision_test.go`.

Reserve importType and explicitly exclude it from the existing value/local MOD015 loop. Existing file.go is 487 lines and codes.go 424; no touched file is near 800 lines. Use companion files if growth approaches that gate.

**Acceptance:**

- [ ] MOD016 constant, registry entry and code-list test agree; MOD015 text describes import-versus-local only.
- [ ] Adding importType cannot make MOD015 reject type/value sharing or change existing MOD015 reports.
- [ ] Focused internal/errors and internal/elaborate tests pass.

### M2: Reject ambiguous selective type bindings (~160 LOC)

**Dependencies:** M1

**Files:** `internal/elaborate/file.go`, `internal/elaborate/import_collision.go`; prefer companion `internal/elaborate/import_type_collision.go` and `import_type_collision_test.go` for the new checker/report helpers.

Classify in pipeline precedence order value → type → ctor using the loaded module; propagate load failures. Collect type binds in source order, compare bind name and import path, and call the checker at the shared ElaborateFile choke point. Leave pipeline alias writes unchanged. Reuse aliasSuggestion, but do not reuse value-specific fix text offering qualification: types have no qualified syntax. Include three-way collisions and aliased different original names converging on one binding.

**Acceptance:**

- [ ] Both orders of each alias-shape truth-table case produce MOD016 before return-type unification, including unused bindings.
- [ ] Report names all colliding imports in source order, carries name/imports data, and spans the first import declaration.
- [ ] Same-path duplicates, explicit symbol aliases, type/value sharing and qualified value imports remain legal; dual-export precedence remains unchanged.
- [ ] Suggested symbol alias compiles; module alias alone does not exempt a selective type collision.

### M3: Verify package CLI and LSP diagnostics (~100 LOC)

**Dependencies:** M2

**Files:** `cmd/ailang/import_type_collision_test.go`, `internal/lsp/import_type_collision_test.go`; fixture modules under `cmd/ailang/testdata/import_type_collision/` (package manifest, a.ail, b.ail, main.ail and alias-fixed case). Generate temporary order variants in tests rather than duplicate fixtures.

**Acceptance:**

- [ ] Package repro rejects both orders with MOD016 using a binary and stdlib from this checkout.
- [ ] check --json preserves code MOD016, phase elaborate, first-import span, name/imports data and fix.
- [ ] LSP emits a positioned MOD016 diagnostic at the first import; existing MOD015 tests still pass.

### M4: Document policy and verify compatibility corpus (~70 LOC)

**Dependencies:** M3

**Files:** `docs/docs/reference/errors/mod016.md`, `docs/docs/reference/errors/index.md`, `docs/docs/reference/modules.md`, `changelogs/unreleased/2026-10-28-import-alias-collision.md` (opens with `### `). Record corpus comparison and sibling scan evidence in sprint execution notes.

**Acceptance:**

- [ ] MOD016 reference, error index, modules guide and dated unreleased changelog fragment document Phase 1 and deferred surfaces.
- [ ] Before/after corpus verdict comparison over std/, examples/ and cmd/ailang/testdata/ has zero changes outside new fixtures.
- [ ] Motoko selective-import scan is recorded; unavailable sibling checkout is an explicit unresolved release check, never a claimed pass.
- [ ] make test-changed, make check-file-sizes and make check-changelog pass; affected Go files are formatted and targeted lint passes.

## Day-by-day work

- Day 1: baseline corpus verdict capture with the checkout-built compiler, then M1 and M2 with focused error/elaborator tests.
- Day 2: M3 package/JSON/LSP integration and fix-paste checks. Investigate any front door that bypasses elaboration before declaring coverage.
- Day 3: M4 documentation, candidate corpus comparison, sibling scan, focused final gates and evidence. Reserve time for compatibility regressions.

## Fixture and example requirements

The repro package is the example deliverable; a negative compile-time example belongs in testdata rather than the runnable examples corpus. Cover differing record bodies in both orders, identical alias bodies in both orders, unused imports, symbol aliases, module aliases, same-module repetition, type/value coexistence, qualified value idiom, and existing shadow/capture cases. The positive alias-fixed fixture must type-check using the intended record for its return annotation.

For every fixture module a.ail, b.ail, main.ail and the alias-fixed variant:

- Contracts: skip: these isolate import binding diagnostics; requires/ensures would not exercise the defect.
- Effects: skip: exported type declarations and pure mk() require no effect row or capabilities.
- Inline tests: skip: Go integration tests assert compile verdicts, structured reports and fix compilation, including intentionally invalid modules.

## Validation and evidence

Use existing Make targets and collision tests first. Run focused Go tests for internal/errors, internal/elaborate, internal/lsp and cmd/ailang during the relevant milestones; final gate is `make test-changed` (includes test-core), with `make check-file-sizes`, `make check-changelog`, formatting and targeted lint. Do not require the full `make test`; CI owns that suite. Run boundary checks if the implementation introduces cross-package dependencies.

Capture baseline and candidate corpus outcomes with identical flags, environment and checkout stdlib, using separately identified binaries; disable compilation caching for comparisons. Compare per-file verdict and diagnostic code, preserve known failures and record totals rather than assuming 729 files still exist. Inspect existing corpus tooling before creating a new harness. Run package-level check from the fixture package root, as in the issue. Record exact commands, binary identities, verdict deltas and any unavailable external checks. Existing MOD015 and type-shadow/capture tests must remain green.

Success means all new branch/policy cases are covered, zero unintended corpus verdict changes, MOD016's human and structured forms agree, the alias fix compiles, and focused gates pass. No numerical whole-repository coverage claim is needed for this narrow diagnostic.

## Risks and handoff

The primary risks are MOD015 treating importType as importValue, constructor/type classification precedence, value-specific fix suggestions, stale binary/cache results and an unavailable motoko sibling. The milestone criteria address each; unavailable external evidence remains explicit. Same-path duplicates stay legal per the approved design; path canonicalization policy is not added here.

The JSON is initialized not_started with every passes field null. Issue #1980 remains open. No code changes, merge, issue closure or executor dispatch occur in this planning task. After plan approval and resolution or waiver of the recorded quorum prerequisite, the coordinator can route these artifacts to sprint-executor; sprint-evaluator follows implementation.
