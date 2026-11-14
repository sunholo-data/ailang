# Sprint Plan: M-IMPORT-VS-IMPORT-COLLISION

**Status:** Planned; conditional on duplicate-design ruling below
**Created:** 2026-11-03
**Task:** task-4027bf38; issue #1980
**Design:** [full-class design](v0_54_20/m-import-vs-import-collision.md)
**Sibling:** [types-only alternative](v0_55_0/m-import-alias-collision.md)
**Duration:** 2 working days, approximately 6 hours including 25% contingency
**Estimate:** 650 changed LOC (implementation, tests, fixtures and documentation)
**Risk:** Medium
**Target:** next unreleased patch after checkout v0.57.0; no version bump in this sprint.

## Summary and scope

Reject ambiguous explicit selective imports before they silently select the last binding. Recommended scope is the full-class design: types and constructors fail at the second import; values fail only at a bare use. Preserve qualified values, lexical shadowing, local-type precedence and identical repeated imports. Sequence both design M1 and M2 in one release so the same explicit-import defect is closed across namespaces.

This is a planning handoff only. No compiler implementation or execution approval is implied.

## Duplicate-design decision and executor gate

The full-class design's opening **Sibling design doc — reconcile before sprint-planning** note explicitly leaves MOD015 versus MOD016 to the human gate. The sibling also differs on constructor timing, diagnostic span, and release scope. The handoff approves previous work but supplies no explicit reconciliation of those conflicting policies. Do not treat the full-class Design Freeze checkboxes as a ruling that resolves its newer opening note.

Recommendation for approval: use the full-class design as canonical, reuse MOD015, diagnose explicit types/ctors at the second import and values at bare use, and defer all automatic channels. Approval must explicitly record this selection in the plan or canonical design; a generic approval of both conflicting documents does not resolve it. If the types-only MOD016 alternative is selected, revise this plan and JSON before execution.

**Before M1, if that selection is still unrecorded: STOP: commit current work, report `BLOCKED: duplicate-design ruling required (MOD015 full-class versus MOD016 types-only)`, and end the run.** This is the only human-decision gate; ordinary implementation choices do not reopen approved policy.

## Current implementation and velocity evidence

- Clean worktree at planning start; current branch coordinator/task-8e8ad12a, HEAD 2b272cef, std/VERSION v0.57.0. The original work branch is not checked out and was not switched.
- `file.go` imports skip types; `import_collision.go` checks imports against locals only. `expressions.go` resolves bare imported values from globalEnv without ambiguity checks. Both pipeline fronts reach ElaborateFile.
- Existing MOD015 report builder, source rendering and alias suggestions can be reused. Classification must collect every applicable namespace independently: GetExport succeeding must not hide a same-named type.
- Velocity script (7 days) found one commit, no usable LOC metrics; repository is shallow. No defensible historical LOC/day can be calculated. Bottom-up capacity assumption: 325 changed LOC/day, 650 total, with 25% time buffer. This is an estimate, not observed throughput.
- Current files are below the 800-line ceiling: file.go 487, import_collision.go 216, core.go 425, expressions.go 162. Use companion import_import_collision.go and import_vs_import_test.go; do not enlarge unrelated files or use the existing large pipeline files for fixtures.

## Registry reuse audit

Ran `ailang pkg search import` (no results), `ailang pkg search diagnostic` (sunholo/ail_diag@0.1.0), then `ailang pkg info sunholo/ail_diag` and `ailang pkg docs sunholo/ail_diag`. Candidate parses emitted CLI text; it cannot implement compiler namespace resolution or structured Go errors. Every milestone uses **none**: reuse internal MOD015 machinery and existing test/doc workflows, no external package dependency. No package-like runtime capability is introduced.

## Syntax and examples

AILANG prompt version loaded: v0.16.11 (`ailang prompt --version-active`); `ailang prompt` was loaded during planning. This is the active teaching-prompt version, not checkout std/VERSION. CLI warns its binary may be stale. Executor must build the checkout binary, load its prompt again, and use that binary consistently for before/after checks; do not use the PATH binary as a baseline for this checkout.

New passing showcase directory: `examples/import_collision_alias/`, modules a.ail, b.ail, main.ail. Export distinct record types under an aliased import, and demonstrate qualified value calls. Keep intentionally failing repro modules under `internal/elaborate/testdata/import_vs_import/` or existing test-helper temporary directories, outside the passing examples gate. Include a valid package manifest for the original package-mode repro. Module declarations must match paths.

| Showcase | Contracts | Effects | Inline tests |
|---|---|---|---|
| a.ail and b.ail | skip: trivial providers for import resolution, asserted by compiler integration tests | include: provider value functions use explicit `! {}` | skip: providers exercised through main and Go fixtures |
| main.ail | skip: import resolution rather than arithmetic contract behavior is under test | include: exported demo function uses explicit `! {}` | include: integer demo function with planned `tests [(0, 3)]`, providers returning 1 and 2 |

Executor validates syntax with check before publishing snippets; use test and verify-examples for the passing showcase. No new language syntax is planned.

## Milestones

### M1: Classify all explicit namespaces and establish fixtures (~150 LOC)

**Dependencies:** None, after the decision gate.
**Duration:** Day 1, 60 minutes. **Estimate:** 35 implementation + 115 test/fixture LOC.

Update file.go and import_collision.go to add importType and independent type/value/ctor classification. Put loader/classification helpers in import_import_collision.go if useful. Preserve loader errors; types must not accidentally enter the existing value-vs-local check. Establish regression tests in import_vs_import_test.go with a/b/main fixtures and a package manifest.

- [ ] Type-only, ctor-only and dual type/value exports produce the intended independent bindings.
- [ ] Existing MOD015 import/local behavior and local type precedence remain unchanged.
- [ ] Focused elaborate tests pass; failing collision expectations are introduced with M2, keeping this milestone green.

Risk: the old kind switch defaults to values; explicitly exclude importType from that pass.
Example files: fixture providers a.ail and b.ail; no public failing example.

### M2: Reject explicit type and constructor ambiguity (~180 LOC)

**Dependencies:** M1.
**Duration:** Day 1, 75 minutes. **Estimate:** 60 implementation + 120 test LOC.

Implement source-ordered import/import comparison in import_import_collision.go, wired after import/local checks. Group by bare name and namespace; compare (source path, exported name), not only module path. Suppress type collisions when a local type declaration wins. Reuse MOD015 and pasteable aliases.

- [ ] Original Leg repro fails with MOD015 naming both imports in both orders; structured span is second import.
- [ ] Explicit constructor duplicates fail; same-module different-export aliases collide.
- [ ] Identical repeated export, disambiguated aliases, independent namespaces, and local type winner remain legal.
- [ ] MOD007 precedes import/local MOD015, which precedes import/import MOD015; three-source results are stable in source order.
- [ ] Focused elaborate tests pass, including applying the proposed alias fix and rechecking.

Risk: map iteration destabilizes first diagnostic; iterate binds in source order.
Example files: negative fixture mains plus a positive aliased main in testdata.

### M3: Diagnose values at bare resolution (~150 LOC)

**Dependencies:** M2.
**Duration:** Day 2, 75 minutes. **Estimate:** 40 implementation + 110 test LOC.

Add per-file value-binding provenance to core.go, initialize it in file.go before body elaboration, and check it only at the globalEnv bare-identifier branch in expressions.go. Use shared collision identity/rendering helpers. Clear state on ElaborateFile reuse.

- [ ] Bare ambiguous imported value reports MOD015 at the use, naming both imports.
- [ ] Qualified calls from both modules, unused duplicate values, same-export repetition and lexical binders remain legal.
- [ ] Uses nested in lambda, record and match bodies fail consistently; successive files do not leak ambiguity state.
- [ ] Existing synthesized-builtin and nullary-constructor lookup precedence remains intact; focused elaborate tests pass.

Risk: recording ambiguity too late or checking before lexical lookup changes legal shadowing.
Example files: value fixture mains for bare, qualified and shadowed uses.

### M4: Front-door diagnostics and passing showcase (~100 LOC)

**Dependencies:** M3.
**Duration:** Day 2, 60 minutes. **Estimate:** 75 integration-test + 25 showcase LOC.

Add companion tests internal/pipeline/import_vs_import_test.go and internal/lsp/import_vs_import_test.go using existing fixture patterns. Create examples/import_collision_alias/{a,b,main}.ail. Verify human output, structured reports and LSP positions without depending on exact prose.

- [ ] Single-file and `check --package .` fronts reject both import orders and accept aliased fixes.
- [ ] CLI JSON/report code MOD015, phase elaborate, data.name and both import sites survive wrapping; LSP spans second import for type/ctor and bare use for value.
- [ ] Passing showcase checks and its inline test passes with the checkout binary.
- [ ] Targeted pipeline, LSP and diagnostic-fixture tests pass.

Risk: package-mode manifest or path mismatch can mask collision behavior; assert coded error and named sources.

### M5: Corpus compatibility and documentation (~70 LOC)

**Dependencies:** M4.
**Duration:** Day 2, 45 minutes, plus 45-minute contingency across sprint. **Estimate:** 70 documentation LOC.

Update docs/docs/reference/errors/mod015.md and docs/docs/reference/modules.md. Add changelogs/unreleased/2026-11-03-import-vs-import-collision.md opening with `### `. Explain timing, identical-export exemption, alias fix and deferred automatic imports. No MOD016 allocation or registry-description change under the recommended ruling.

- [ ] Baseline and changed checkout binaries check the same enumerated tracked .ail corpus; record actual count and compare verdicts, with only new negative fixtures excluded. Do not assume the old 778 count is current.
- [ ] Existing examples/alias_body_closure/main.ail and tools/motoko/r8_headroom_band.ail retain their baseline verdicts.
- [ ] Focused gates pass: make test-changed (base pinned to pre-sprint HEAD), make check-file-sizes, make check-changelog, make verify-examples and make verify-stdlib.
- [ ] Changed Go files are gofmt-clean; no unrelated compiler or resolution changes.

Risk: corpus has existing failures; compare verdicts to baseline rather than demanding universal success. Use existing tools/audit-examples.sh and tools/verify_examples.sh where applicable; retain sweep logs/counts for evaluator. If broader enumeration needs a shell loop, keep it temporary rather than adding a new repository tool. Baseline capture must happen before M1 edits.
Example files: document the M4 passing showcase; negative fixtures stay out of example discovery.

## Validation and resumption

Before M1, record the policy ruling, pre-sprint commit and corpus baseline using a checkout-built binary outside the tree. Retain that binary while rebuilding the changed checkout. After each milestone run the relevant package tests; at completion use make test-changed with TEST_CHANGED_BASE set to that recorded pre-sprint commit so committed milestones remain covered even in a shallow checkout. CI owns the full suite. Do not require make test on the executor. Run check-boundaries only if implementation crosses package boundaries; this plan adds helpers inside existing packages.

The JSON features M1–M5 have matching dependencies and estimates, and all passes remain null. Resume from that file; record evidence before marking a milestone complete. Submit for sprint-evaluator after execution. Issue #1980 is the driver; #1467 is historical context, not an issue to close.

## Deferred policy reconciliation (S1–S6)

This table folds the sibling's surface audit into the execution scope without silently adopting its contradictory rulings.

| Surface | Recommended sprint treatment |
|---|---|
| S1 explicit selective type | Import-site MOD015 at second import, conditional on ruling above |
| S2 explicit selective value | Use-site MOD015; qualified calls legal |
| S3 explicit selective constructor | Import-site MOD015 per full-class design; sibling's use-site deferral is superseded only by explicit ruling |
| S4 bulk type aliases | Deferred to separate follow-up design; preserve explicit-beats-auto and alias-body capture semantics |
| S5 bulk constructors | Deferred with S4; no new enforcement here |
| S6 transitive alias pull | Unchanged sorted, reach-limited gap fill |

No wildcard syntax, qualified type syntax, nominal identity changes, REPL alias merge changes or auto-import origin sets are in scope. If sibling motoko checkout is unavailable, record that coverage limit rather than claiming a sibling sweep. No release, merge or deployment is part of this plan.

## Handoff

Plan and JSON are ready for coordinator review with the cited policy gate intact. Coordinator approval/merge dispatches sprint-executor; planning does not self-dispatch implementation while the duplicate-design decision remains open.
