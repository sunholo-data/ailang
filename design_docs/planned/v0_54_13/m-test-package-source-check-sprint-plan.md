# Sprint Plan: M-TEST-PACKAGE-SOURCE-CHECK

## Summary

Make `ailang test --package` run the full existing package check before executing tests, with fail-fast diagnostics, one JSON report, and an explicit `--no-source-check` escape hatch. Reuse `internal/check` rather than add compiler behavior.

**Design:** [Approved design](m-test-package-source-check.md)
**Duration:** 1.5 working days (~8 hours work + 2 hours contingency).
**Estimated total:** 500 LOC (implementation, tests, fixtures, documentation).
**Risk:** Medium: default exit-code change, JSON compatibility, cache reuse uncertainty.
**Status:** M1–M5 implemented; final gates and measurements being recorded for coordinator evaluation.

## Current status and evidence

Source inspection confirms `runPackageTests` counts sources but only runs test files, and returns early when no test files exist. `checkPackageWithContext` already delegates discovery and all analyses to `internal/check`. `SuiteResult.Success()` requires tests to run; `SuccessAllowingSkips()` currently permits zero failures. Source-check failures must defeat both paths explicitly. The JSON reporter constructs a map, so adding a struct field alone does not expose the new key.

Current source version: v0.54.12. Available CLI warns that it is stale. AILANG prompt version loaded: v0.16.7 (`ailang prompt`, with `--version-active` confirmation). Executor must build the repository binary and reload its prompt before creating fixture programs; the installed binary is not behavioral evidence for the implementation.

Velocity script run for seven days: only one visible commit (`73e33a56`, attended triage rulings); no usable LOC metrics or historical diff in this shallow checkout. Historical LOC/day is unavailable. Capacity assumption: ~333 LOC/day, 500 LOC over 1.5 days; estimates are bottom-up task budgets with ~25% contingency, not a measured velocity claim. Design's three broad phases are split into six executor-sized milestones (each <=90 minutes).

The approved design's systemic audit covers package test, standalone check, pkg quality, cascade, JSON consumers, and file/dir mode. File/dir changes and cascade simplification remain deferred. Discovery covers all module-declaration files, including non-exported modules and test modules, rather than only `[exports]` entries.

## Registry reuse audit

Ran `ailang pkg search testing`; result: `sunholo/testing_utils@0.1.1`. Inspected with `ailang pkg info sunholo/testing_utils` and `ailang pkg docs sunholo/testing_utils`: pure AILANG assertion functions, no compiler/CLI/reporting capability. M1–M6 each decide **none**: compiler entry-point integration, Go regression tests, reporting, and documentation belong in this repository. Reuse existing `internal/check`, testing reporter, and `runAilangBin`; introduce no registry dependency.

## Milestones

### ✅ M1: Source-check pre-pass and opt-out (~100 LOC)

**Budget:** 90 minutes; ~55 implementation + 45 test/fixture LOC.
**Dependencies:** None.
**Files:** `cmd/ailang/test.go`, `commands_language.go`; new `cmd/ailang/test_package_source_check_test.go`. If a helper is needed, put it in new `cmd/ailang/test_package_source_check.go`.

Resolve the package root once, use `DiscoverPackageSources` and `CheckPackageFiles` with existing default options (including InterFunctionRefs and StrictFallbacks). Run before the zero-test return. Thread `--no-source-check` through dispatch; keep file/dir behavior unchanged. Reuse the current test-file discovery for execution; do not replace check discovery with suffix filtering.

- [x] Broken unimported module fails by default (exit 1); opt-out executes the passing test and exits 0.
- [x] Broken package with no tests fails; green package with no tests retains warning and exit 0.
- [x] Fail-fast path never executes test bodies; source failures are not waived by `--allow-skips`.
- [x] Targeted real-binary tests pass.

### ✅ M2: Human and JSON reporting contracts (~110 LOC)

**Budget:** 90 minutes; ~55 implementation + 55 tests.
**Dependencies:** M1.
**Files:** `internal/testing/result.go`, `reporter.go`; new `internal/testing/source_check_test.go`, new `cmd/ailang/test_package_source_check_json_test.go`; package helper from M1. Small shared renderer may live in new `cmd/ailang/check_package_report.go` with `check_package.go` delegating to it.

Add optional source-check result with passed/failed counts and error/warning string arrays. Explicitly emit `source_check` in reporter's JSON map when checked; omit when disabled or outside package mode. Ensure failure success predicates cannot be overridden by skips. Keep per-file human progress and warnings off JSON stdout. Preserve seeded replay metadata and existing suite keys. For zero-test package JSON, emit one valid suite document; preserve legacy empty-suite success field semantics and exit 0 on a green pre-pass rather than changing generic empty-suite rules.

- [x] Human failure names the bad file and reports package-check failure without a passing-tests claim.
- [x] Both `--json` and `--format json` produce exactly one JSON document on pre-pass success and failure, including zero-test cases.
- [x] Failure JSON has `success:false`, source counts/errors, zero executed tests, and exit 1.
- [x] Reporter tests cover present/absent source_check and stable existing keys; existing JSON stdout tests pass.

### ✅ M3: Package quality consumer integration (~45 LOC)

**Budget:** 60 minutes; ~5 implementation + 40 tests.
**Dependencies:** M2.
**Files:** `cmd/ailang/pkg_quality.go`; new `cmd/ailang/pkg_quality_source_check_test.go`.

Pass `--no-source-check` to the subprocess because quality already checks in-process. Use an existing subprocess seam or a test executable to assert argument forwarding, without adding production dependency injection solely for tests. Compare stable report fields; exclude duration or other variable fields from equality assertions.

- [x] Quality subprocess receives opt-out, retaining exactly one package-check pass in the quality workflow.
- [x] Quality JSON schema and deterministic report content remain unchanged.
- [x] Coordinator cascade remains unchanged and green fixture behavior still works.

### ✅ M4: Discovery and flag regression matrix (~105 LOC)

**Budget:** 90 minutes; ~105 test/fixture LOC.
**Dependencies:** M3.
**Files:** new `cmd/ailang/test_package_source_check_regression_test.go`; existing strict-fallback and multi-path test helpers; temporary package fixtures generated by Go tests.

Cover non-exported broken modules, declared test modules, module-less orphans, hidden directories and named-test debris according to check discovery. Exercise strict-fallback rejection and inter-function-reference rejection using existing valid fixtures; compare diagnostic category with check. Confirm `--allow-skips` cannot bypass failure, multi-package paths remain usage exit 2, and opt-out/non-package output omit source_check. Smoke existing bytecode engine flags on a green fixture.

- [x] Discovery coverage matches `check --package`, not merely export declarations.
- [x] StrictFallbacks and InterFunctionRefs failures prevent all tests and exit 1.
- [x] Existing JSON/multi-path/check strict-fallback tests pass without weakened assertions.
- [x] `examples/intra_package_imports` stays green with zero test files; file/dir behavior stays unchanged.

### ✅ M5: Documentation, CLI help and templates (~100 LOC)

**Budget:** 60 minutes; ~100 documentation/template LOC.
**Dependencies:** M4.
**Files:** `cmd/ailang/test.go` (`printTestHelp`), `init_motoko_extension_templates.go`, `docs/docs/guides/packages.md`, `testing.md`, `strict-fallbacks.md`; new `changelogs/unreleased/2026-10-16-test-package-source-check.md` opening with `### `.

Explain default full checking, fail-fast behavior, additive JSON key, opt-out, and `--allow-skips` limitations. Simplify generated package CI instructions to the test command; keep standalone check documented. Describe cache reuse as measured evidence only, never promise a single compiler invocation.

- [x] CLI help and all listed docs/templates describe the same behavior.
- [x] Changelog calls out intentional 0-to-1 exits for broken packages and external JSON key addition.
- [x] Generated extension checklist no longer requires duplicate package check/test steps.
- [x] No unrelated cascade or file/dir changes.

### ✅ M6: Cache measurements and focused final validation (~40 LOC)

**Budget:** 90 minutes; ~40 report LOC.
**Dependencies:** M5.
**Files:** new `design_docs/planned/v0_54_13/m-test-package-source-check-implementation-report.md`. Use temporary generated fixtures rather than commit 50 synthetic modules.

Build repository binary. Generate a >=50-module temporary package with shared imports and a runnable test. Use an isolated task-owned cache directory via the supported cache setting (inspect CLI help first). Record module count, CPU/runner shape, binary identity, cache setting, cache stats before/after each run, wall time, cold and warm `test --package`, and separate `check --package` plus opt-out test baseline. Do not clear a shared user cache. If stats cannot attribute hits across phases, say so and collect supported tracing; do not infer dedup from timing alone.

- [x] Report includes measured cache counts and timing for >=50 modules with methodology and limitations.
- [x] If cache reuse is poor, document it and escalate artifact-sharing work; do not expand this sprint or claim the reporter's time saving was reproduced.
- [x] `make test-changed` (touched packages plus core), `make lint`, `make check-boundaries`, `make check-file-sizes`, and `make check-changelog` pass.
- [x] Targeted binary regressions use the freshly built binary; report exact commands/results.

## Schedule, validation and scope

Day 1: M1–M4 (~5.5 hours). Day 2: M5–M6 (~2.5 hours) plus ~2 hours contingency. Each milestone must be reviewable and fit in a 60–90 minute executor run; JSON dependencies use the exact milestone IDs. Run focused relevant tests after each milestone, then final touched-package/core gates once. The skill's focused validation replaces the design document's full `make test` requirement; full suite belongs to CI. No stdlib changes, so `verify-stdlib` is unnecessary; run targeted examples, broad `verify-examples` only if example source changes.

Files inspected for size: commands_language.go 483 lines, test.go 348, templates 436, quality 287; none near 800. Use the companion test files named above to avoid growing an existing large test file. Recheck touched-file sizes at completion.

## Example and syntax checklist

No permanent public `.ail` example is required for this CLI behavior; use the existing green `examples/intra_package_imports` plus temporary repro and performance packages. Before writing any embedded/generated AILANG, executor must reload the fresh binary's prompt and validate green fixtures using `ailang check --package`; deliberately broken fixtures must fail with their expected diagnostics.

| Fixture/demo module | Contracts | Effects | Inline tests |
|---|---|---|---|
| Temporary good source | skip: compiler-entry-point regression needs no contracts | include: pure `one() -> int`, empty effect row | skip: named tests live in companion test module |
| Temporary bad/non-exported source | skip: intentionally invalid return type | include: pure signature, empty effect row | skip: compilation failure is the assertion |
| Temporary companion test module | skip: named assertion tests compilation gating | skip: no effectful fixture needed | include: named test invoking `one() == 1` |
| Temporary >=50-module cache package | skip: contracts would confound compile measurements | include: pure functions, empty effect rows | include: companion named test over shared imports |
| Existing intra_package_imports | skip: unchanged fixture | skip: unchanged declarations | skip: unchanged zero-test regression |

## Handoff and decisions

No new design decisions require resolution. Preserve current cascade's explicit check; removal remains future human-reviewed work. Green zero-test package JSON remains a valid report with the established empty-suite success value, while CLI exit stays 0; this avoids redefining generic suite semantics. `--no-source-check` is meaningful in package mode; elsewhere it leaves existing behavior unchanged. No performance threshold is promised before measurements.

Coordinator handoff artifacts are this plan and `.ailang/state/sprints/sprint_M-TEST-PACKAGE-SOURCE-CHECK.json`. The coordinator advances to sprint-executor after plan approval/merge; do not start implementation during planning. The approved design references #1275/#1572 only as cache prior art; they are not issues to close for this sprint.
