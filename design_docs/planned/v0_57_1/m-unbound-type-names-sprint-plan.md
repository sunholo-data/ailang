# Sprint Plan: M-UNBOUND-TYPE-NAMES

## Summary

Implement the approved [design](m-unbound-type-names.md): reject unbound annotation names at their source with TC_UNBOUND_TYPE_001 and deterministic spelling/import guidance.

**Status:** Planning complete; implementation not started. Design approval is supplied by the coordinator handoff; its unchecked design-freeze boxes are historical, not fresh approval requests. Sprint execution follows coordinator approval/routing.
**Target:** v0.57.1. **Duration:** 3 working days, 8.25 focused implementation hours plus corpus execution, review and contingency. **Estimate:** 900 LOC including tests/docs. **Risk:** medium.

## Current status and velocity

Clean starting tree on coordinator/task-1d05d09c; remain on this branch. Source VERSION is v0.57.0. The seven-day velocity script found only one visible commit; its numstat represents a repository snapshot, so empirical LOC/day and coverage cannot be inferred. Budget 300 LOC/day as a planning assumption, retaining the approved design's doubled three-day estimate. No implementation milestone is already complete.

Source inspection locates compileFreshModule and shadowLocalTypeNames in internal/pipeline/pipeline_module_phases.go (not the design's pipeline_module_compile.go integration location). Keep the new walker and diagnostics in companion files: pipeline_single.go is 661 lines and pipeline_module_compile.go 657; avoid growing either toward the 800-line ceiling. New tests are also split by responsibility.

AILANG prompt version loaded: v0.16.11 (`ailang prompt`, `ailang prompt --version-active`). This is older than the source version and reports a stale binary. Before implementation, build the source CLI using the existing build workflow, reload its active prompt and record the result before writing .ail fixtures. Do not treat the installed CLI as current semantic verification.

## Registry reuse audit

Ran `ailang pkg search 'type validation'`: no packages found (installed CLI reports a stale binary). These are compiler-internal AST/import-table checks, not a package-like runtime capability. M1–M6 each choose **none**: source Go pipeline and local tests/docs must implement the work; no registry candidate warrants info/docs inspection or a dependency. No new scripts or shared edit-distance package are needed; prefer a local small helper to avoid coordinator imports.

## Rulings and conditional decision

D1 hard errors apply to all constructor annotation positions. D3 pipeline pass, D4 code, D5 deterministic suggestions and D6 captured-alias exemption are settled. D7 permits REPL coverage only if cheap; document deferral otherwise. Keep converter fallback branches intact.

D2's corpus contingency is the sole conditional human-decision gate, cited from design **High-Impact Decisions D2 / Design Freeze**. A clean scan permits strict lowercase checking. Legitimate reliance requires findings and the proposed warning-first migration to be surfaced; missing motoko_agent access is an incomplete gate, never evidence of no reliance. **STOP: commit current work, report `BLOCKED: D2 corpus decision` (or missing corpus access), and end the run.** Do not silently activate hard errors, invent warning infrastructure, or mark dependent milestones passed. No other human decision is open.

## Milestones

### M1: Constructor scope and pipeline integration (~180 LOC)

**Estimate:** 90 implementation/docs + 90 tests = 180 LOC; 90 minutes.
**Dependencies:** None.
**Files:** internal/pipeline/type_name_scope.go; internal/pipeline/type_name_scope_test.go; internal/pipeline/pipeline_module_phases.go.

- [ ] Assemble builtin, local, imported alias/parameterized alias/ADT and captured-alias sets after shadowLocalTypeNames in pipeline_module_phases.go.
- [ ] Reject unknown constructor names in signatures, alias bodies and ADT fields before elaboration with TC_UNBOUND_TYPE_001, source position and annotation context.
- [ ] Keep local recursive names, renamed imports and captured-alias diagnostics working.

**Milestone gate:** targeted Go tests for this behavior, then `make test-changed`; maintain gofmt and lint cleanliness.
**Risk/mitigation:** False positives from omitted scope sources: mirror actual elaborator import tables and preserve captured aliases.

### M2: Complete annotation traversal and implicit modules (~180 LOC)

**Estimate:** 70 implementation/docs + 110 tests = 180 LOC; 90 minutes.
**Dependencies:** M1.
**Files:** internal/pipeline/type_name_scope_walk.go; internal/pipeline/type_name_scope_walk_test.go; internal/pipeline/pipeline_single.go.

- [ ] Walk nested expression bodies, lambda parameters/returns, let/letrec annotations, record fields, TypeApp heads/args and structural list/array/tuple/function types.
- [ ] Cover V1–V11 with nonzero check exit, unknown name and annotation position; inspect pipeline_single.go and wire implicit modules without a module-less bypass.
- [ ] Skip effect and taint label namespaces; record whether REPL gets coverage incidentally, otherwise defer it under D7.

**Milestone gate:** targeted Go tests for this behavior, then `make test-changed`; maintain gofmt and lint cleanliness.
**Risk/mitigation:** Missing nested annotations or synthetic modules: enumerate all AST expression containers and test both check paths.

### M3: Deterministic suggestions and import hints (~140 LOC)

**Estimate:** 70 implementation/docs + 70 tests = 140 LOC; 75 minutes.
**Dependencies:** M1, M2.
**Files:** internal/pipeline/type_name_scope_diagnostics.go; internal/pipeline/type_name_scope_diagnostics_test.go.

- [ ] Add sorted close matches within edit distance 2, capped at 3; omit the section when none match.
- [ ] Produce direct-dependency import hints and bound imported-name context for stale Row and missing Option; never scan the network or all registry packages during checking.
- [ ] Golden tests pin Strng suggestions, stale-name import guidance, stable ordering and bounded diagnostics (first occurrence per name, at most 10).

**Milestone gate:** targeted Go tests for this behavior, then `make test-changed`; maintain gofmt and lint cleanliness.
**Risk/mitigation:** Unstable hints or wrong import scope: sort/cap candidates and inspect only loaded direct-dependency interfaces.

### M4: Lexical type variables and blocking corpus decision (~150 LOC)

**Estimate:** 60 implementation/docs + 90 tests = 150 LOC; 90 minutes.
**Dependencies:** M2.
**Files:** internal/pipeline/type_name_scope.go; internal/pipeline/type_name_scope_walk.go; internal/pipeline/type_name_scope_typevars_test.go; design_docs/planned/v0_57_1/m-unbound-type-names-sprint-report.md.

- [ ] Inherit function/type-declaration type parameters into nested lambdas and scope record row variables; reject undeclared qq in candidate strict mode.
- [ ] Scan std/, examples/, Go test fixtures and an explicitly located motoko_agent corpus; record commands, revisions, file counts, baseline failures and new failures in the sprint report.
- [ ] Enable lowercase hard errors only after a clean reliance scan; if legitimate reliance exists or required corpus is unavailable, STOP with corpus findings for the D2 decision before claiming completion.

**Milestone gate:** targeted Go tests for this behavior, then `make test-changed`; maintain gofmt and lint cleanliness.
**Risk/mitigation:** Implicit polymorphism compatibility: separate candidate strict validation from production activation and obey the D2 stop.

### M5: Regression fixtures and command integration (~170 LOC)

**Estimate:** 30 implementation/docs + 140 tests = 170 LOC; 90 minutes.
**Dependencies:** M3, M4.
**Files:** internal/pipeline/type_name_scope_regression_test.go; internal/pipeline/type_name_scope_integration_test.go; internal/pipeline/testdata/type_name_scope/ (new valid and invalid .ail fixtures).

- [ ] Pin std/array.ail, std/list.ail, std/json.ail, examples/deriving_eq.ail, examples/datetime_demo.ail and the renamed-import fixture as positive checks.
- [ ] Existing renamed_type_import_test.go remains unchanged and green; local shadow/capture keeps TC_TYPE_SHADOW_001 precedence.
- [ ] Add CLI/package and --args-json integration: stale type fails at check, corrected import checks and decodes successfully; verify --format agent retains name/code/location.

**Milestone gate:** targeted Go tests for this behavior, then `make test-changed`; maintain gofmt and lint cleanliness.
**Risk/mitigation:** Earlier errors invalidate unrelated fixture expectations: update only proven intentional changes and preserve existing rename tests.

### M6: Documentation and focused release gates (~80 LOC)

**Estimate:** 40 implementation/docs + 40 tests = 80 LOC; 60 minutes.
**Dependencies:** M5.
**Files:** docs/docs/reference/errors/index.md; docs/docs/reference/errors/tc_unbound_type_001.md; examples/type_name_scope.ail; changelogs/unreleased/2026-11-03-unbound-type-names.md; design_docs/planned/v0_57_1/m-unbound-type-names-sprint-report.md.

- [ ] Document TC_UNBOUND_TYPE_001 and migration by explicit type parameters; add a valid runnable example and unreleased changelog fragment.
- [ ] Record std/examples individual and package-check results against baseline, verify stdlib interfaces and examples, run focused tests and lint with no new regressions.
- [ ] Record D2 disposition, REPL coverage/defer decision and check-time-only engine parity rationale in the sprint report; evaluator/VM semantics remain untouched.

**Milestone gate:** targeted Go tests for this behavior, then `make test-changed`; maintain gofmt and lint cleanliness.
**Risk/mitigation:** Corpus contains intentional invalid examples: compare against baseline and require zero newly failing legitimate programs.

## Day-by-day schedule

- Day 1: rebuild CLI/reload prompt; M1, M2, M3; targeted red/green regressions.
- Day 2: M4 corpus inventory and strict trial; proceed only on clean gate, then M5 integrations.
- Day 3: M6 docs and corpus verification; review buffer and sprint-evaluator handoff after execution.

Each milestone fits a 60–90 minute executor run (M6 about 60 minutes). Corpus commands may outlast active coding; record partial progress and resume rather than claiming a pass.

## Examples and syntax checklist

New valid example: examples/type_name_scope.ail, demonstrating a local record and explicit generic parameters. Invalid Nope/stale-Row/qq examples belong in internal/pipeline/testdata/type_name_scope/, exercised by Go assertions, not the passing examples sweep. Two-module fixtures include types.ail, stale_name.ail and renamed_ok.ail; additional position cases can use inline Go test source strings.

| Showcase | Contracts | Effects | Inline tests |
|---|---|---|---|
| examples/type_name_scope.ail | skip: compile-time name resolution is asserted by Go tests, no runtime contract to demonstrate | skip: pure exported helpers need no effect row | include: a concrete pure projection helper with `tests [(input, expected)]`, checked against the rebuilt teaching prompt |

Negative fixture modules skip contracts, effects and inline tests because intentional compile failures are verified by Go/CLI tests. Existing positive corpus examples need no edits. Add scope guidance to the error-reference page; do not edit a frozen teaching prompt without its own permitted workflow.

## Validation and success metrics

Use the freshly built CLI. Establish a baseline before changing semantics, especially intentional negative examples and package boundaries. Use existing tools and check help to choose valid package roots; do not assume std/ or all examples/ form a single package. Record per-file check totals and package-root check commands/results for both trees.

Required executor gates: `make test-changed` (set TEST_CHANGED_BASE to the recorded starting commit if origin/dev is unavailable), targeted pipeline/CLI tests, `make verify-stdlib`, `make verify-examples`, `make check-boundaries`, `make check-file-sizes`, `make check-changelog`, and repository lint. The full suite remains CI-owned; this focused gate supersedes the design's blanket `make test` instruction for cloud execution.

Coverage target: every V1–V11 position, lexical binding/exemption, deterministic diagnostic branch and six positive regression fixtures gets an assertion; no fabricated baseline coverage percentage. Runtime --args-json correction succeeds. All legitimate corpus programs have zero new failures; intentional invalid examples are accounted for separately. No stdlib interface drift or accepted-program engine behavior change. Changelog fragment must open with `### `.

## Execution handoff

State JSON: .ailang/state/sprints/sprint_M-UNBOUND-TYPE-NAMES.json. Read this plan and approved design, execute milestones in dependency order, preserve progress across runs, and evaluate against the design after completion. Planning artifacts contain no completed implementation claims. No issue number was provided; github_issues stays empty rather than guessing from keyword matches.
