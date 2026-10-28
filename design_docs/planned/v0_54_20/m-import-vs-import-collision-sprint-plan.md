# Sprint Plan: M-IMPORT-VS-IMPORT-COLLISION

## Summary

Reject ambiguous explicit selective imports with MOD015: types and constructors at the second import, values at a bare use. Ship both behaviors in v0.54.20, preserving qualified values, lexical shadowing, independent namespaces, and benign duplicate imports.

- Design: [approved design](m-import-vs-import-collision.md), originating task `task-e8942a7d`.
- Duration: 3 working days; four sequential executor milestones of 60–90 minutes each, plus approximately 2 hours of investigation/validation buffer (8 hours total). Calendar review latency is additional.
- Estimated changes: 600 LOC (180 implementation, 340 tests/fixtures, 80 documentation).
- Risk: medium; namespace classification and diagnostic precedence are the main risks.
- Approval: design approved by handoff; sprint awaits coordinator review/merge before implementation.

## Current status and velocity

Repository version is v0.54.19. The working tree was clean at planning start, on `coordinator/task-5ed6fc7d`; work stays on this branch rather than switching to the earlier design branch.

Code inspection confirms `importKind` has values/constructors only, `file.go` skips type-only exports, and `checkImportCollisions` implements import-vs-local only. Bare value resolution has the shared `globalEnv` branch described in the design. No implementation milestone is complete.

The seven-day velocity script found only design commit `f67e6149` (#1981), no implementation LOC metrics, and no usable diff baseline in this checkout. Historical implementation velocity cannot be calculated honestly. Capacity is therefore an explicit estimate of 200 changed LOC/day, not an observed rate. The design's six-day schedule is replaced with three focused working days because implementation sessions are bounded and the feature excludes a release operation; approximately 25% effort buffer is included. If a milestone cannot fit 90 minutes, split it before execution rather than expanding a run indefinitely.

The design's systemic audit covers explicit types, constructors, values, multi-namespace exports, and auto-import channels. Auto-import ambiguity remains deferred follow-up M3 in the design (distinct from sprint milestone M3 below). No version bump, publication, new error code, or pipeline resolution rewrite is included.

## Registry reuse audit

`ailang pkg search import` returned no packages (the CLI also warned its binary may be stale). No candidate required `pkg info` or `pkg docs`. For M1–M4 the decision is **none**: these are compiler elaboration diagnostics, regression tests, and compiler documentation, which cannot be supplied by a language package. Reuse existing MOD015 report/fix helpers and the existing test harnesses. No package dependency is added.

## Syntax and examples

AILANG prompt version loaded: v0.16.7 (active teaching prompt; repository version is v0.54.19). `ailang prompt` was invoked. The executor must reload the prompt and build the checkout binary before validating fixtures; the preinstalled CLI is not the validation authority.

Regression modules belong in fixture directories or Go test source strings, not the passing showcase corpus. Planned fixture roots: `internal/pipeline/testdata/import_vs_import/` (package and single-file repros) and `examples/import_collision_aliases/` (positive example, subject to the existing example discovery conventions).

| Modules | Contracts | Effects | Inline tests |
|---|---|---|---|
| Negative collision fixtures (`a.ail`, `b.ail`, colliding main variants) | skip: elaboration must fail before contract verification | skip: pure type/constructor/function repros need no effect row | skip: Go harness asserts MOD015 and spans before evaluation |
| Positive alias example (`a.ail`, `b.ail`, `main.ail`) | skip: import disambiguation has no behavioral precondition | include: main has `! {IO}` to print distinct results; helper signatures are pure | skip: integration harness checks distinct results and successful type checking |

## Milestones

### M1: Classify binds and reject type/constructor collisions (~180 LOC)

**Dependencies:** None
**Budget:** 60–90 minutes; 80 implementation + 100 tests LOC.

Extend bind classification independently for each namespace, including type-only exports and symbols exported in multiple namespaces. Keep import-vs-local handling explicit for values/constructors: adding `importType` must not accidentally compare types against local functions. Add the second import-vs-import pass, using `(path, orig)` identity within `(bound name, namespace)`, preserving AST import order. Suppress type collisions when a local type wins. Report the earliest distinct conflicting pair deterministically, positioned at the second import; reuse MOD015 reports and pasteable alias fixes (including valid capitalization for types/constructors).

Files: `internal/elaborate/file.go`, `internal/elaborate/import_collision.go`; new `internal/elaborate/import_vs_import_types_test.go`. Example fixtures: reporter's Leg modules and constructor duplicates embedded in tests; persistent pipeline fixtures follow in M3.

- [ ] Both Leg import orders and explicit constructor collisions report MOD015 with both source imports and positions.
- [ ] Repeated identical exports are benign, including a module alias; different exports aliased to one name collide even within one module.
- [ ] Local type shadowing suppresses type ambiguity; type/value namespace independence and multi-namespace symbols are tested.
- [ ] MOD007 precedes import-vs-local MOD015, which precedes import-vs-import MOD015; three imports produce a stable first conflicting pair.
- [ ] Alias fix text is parsed/checked as a legal import, and targeted elaborate tests pass.

Risk: early `continue` can omit one namespace; mitigate with a multi-namespace export fixture.

### M2: Reject ambiguous bare value uses (~160 LOC)

**Dependencies:** M1
**Budget:** 60–90 minutes; 60 implementation + 100 tests LOC.

Carry value provenance on `Elaborator` before any body elaboration and consult it at the existing bare imported-value resolution branch, after builtin/lexical/nullary-constructor precedence. Retain file context for structured reports. Initialize/reset ambiguity state per file so reusing an elaborator cannot leak a previous module's collisions. Shared helpers may live in new `internal/elaborate/import_ambiguity.go`.

Files: `internal/elaborate/core.go`, `file.go`, `expressions.go`; new `import_vs_import_values_test.go`. Example fixtures: two value exporters and legal qualified/dead/shadowed forms embedded in tests.

- [ ] A bare duplicated value reports MOD015 at the identifier and names both import sites.
- [ ] Qualified references, unused duplicate binds, identical-export duplicates, and local parameter/let/match binders remain legal.
- [ ] Lambda, record-field, match-body, and module-let bare uses reach the shared check.
- [ ] Reused elaborator state cannot produce a false ambiguity in a subsequent file.
- [ ] Existing MOD015 fix and lexical-resolution tests plus targeted elaborate tests pass.

Risk: moving the check above lexical/constructor resolution changes valid programs; keep it at the existing global lookup.

### M3: Verify both pipelines, diagnostic spans, and examples (~160 LOC)

**Dependencies:** M1, M2
**Budget:** 60–90 minutes; 40 integration wiring + 120 tests/fixtures LOC.

Add actual filesystem fixtures with relative imports and a package manifest following existing pipeline test conventions. Exercise package checking and single-file checking, both import orders, and structured error transport. Extend LSP coverage using a companion test file. Add the positive alias example described above and check it with the built binary.

Files: new `internal/pipeline/import_vs_import_test.go`, `internal/pipeline/testdata/import_vs_import/`; new `internal/lsp/import_vs_import_test.go`; `examples/import_collision_aliases/{a,b,main}.ail` (and manifest if required by local convention).

- [ ] Package and single-file repros both report MOD015 rather than a downstream field-count mismatch.
- [ ] Structured diagnostics/LSP locate type/constructor errors at the second import and value errors at the bare identifier.
- [ ] Existing `internal/diag` MOD007/MOD015 fixtures and old LSP import-vs-local tests remain green.
- [ ] `examples/alias_body_closure/main.ail`, qualified-only imports, and r8-style unused duplicates retain their previous verdicts.
- [ ] Positive alias example checks and produces the expected distinct values using the checkout binary.

Risk: intentionally invalid fixtures can enter general passing-example validation; keep them in testdata and only positive modules in examples.

### M4: Corpus comparison, documentation, and focused gates (~100 LOC)

**Dependencies:** M1, M2, M3
**Budget:** 60–90 minutes; 20 test/report support + 80 documentation LOC.

Before M1, bank baseline check verdicts with the checkout binary for the tracked `.ail` corpus. In this milestone repeat identical invocations and compare file-by-file status and normalized error categories; line numbers and timings need not match. Count the actual corpus rather than assuming the design's 778 remains current. Exclude new deliberate negative fixtures from the unchanged-corpus comparison. Use existing repository audit tools first; record unsupported/environment-dependent checks explicitly rather than treating them as passes. No new general-purpose audit script is planned. If the corpus operation exceeds the run budget, split this milestone and preserve baseline/results for resumption.

Update `docs/docs/reference/errors/mod015.md` and `docs/docs/reference/modules.md` with the two diagnostic locations, aliases, benign repeats, and preserved qualified-value idiom. Create `changelogs/unreleased/2026-10-28-import-vs-import-collision.md`, opening with `### `; this supersedes the design's old monolithic changelog path.

- [ ] Existing tracked corpus has zero changed check verdicts; any mismatch is investigated and resolved or explicitly blocks completion.
- [ ] Documentation describes types/constructors at import sites and values at bare uses, with valid alias examples.
- [ ] Changelog fragment and targeted package tests pass; `make test-changed`, `make check-file-sizes`, and `make check-changelog` pass.
- [ ] Changed positive examples pass focused checks and the applicable `make verify-examples` gate; use `make verify-stdlib` only if stdlib files change.
- [ ] Plan/JSON acceptance criteria are updated with observed evidence; no release operation is performed.

Risk: environmental baseline failures obscure regressions; retain failures on both sides and compare rather than requiring every existing file to pass.

## Daily sequence

Day 1: build the baseline binary, bank corpus verdicts, complete M1.
Day 2: complete M2 and M3, retaining focused test evidence.
Day 3: complete M4, use investigation buffer for any mismatch, and prepare evaluator handoff.

## Validation and file size controls

Use the repository build workflow (`make build`) and the resulting checkout binary for CLI checks. Run focused tests in touched packages during each milestone, then `make test-changed` (touched packages plus core tests). Do not require the full `make test` suite in the executor; CI owns it. Run relevant format/lint checks for modified Go code.

Observed file sizes: `file.go` 487, `core.go` 425, `expressions.go` 162, `import_collision.go` 216; none is near the 800-line limit. Keep new test companions `import_vs_import_types_test.go`, `import_vs_import_values_test.go`, and `internal/lsp/import_vs_import_test.go` separate to avoid growing unrelated files. If shared logic grows, use `import_ambiguity.go`. Coverage is scenario-based: every new branch needs positive and negative tests; no unsupported numeric coverage claim is made.

## Dependencies and handoff

There are no external package dependencies or unresolved semantic decisions. The same-release sequence for value ambiguity is chosen here. Design PR #1981 and historical MOD015 issue #1467 are references, not automatically closing issues for this sprint; no bug issue number was supplied.

The coordinator consumes the plan/JSON paths and requests sprint approval; merge triggers the executor per the sprint-planner coordinator instructions. This planning stage does not launch implementation. After execution, sprint-evaluator checks the approved design, the milestone criteria, and retained corpus evidence.
