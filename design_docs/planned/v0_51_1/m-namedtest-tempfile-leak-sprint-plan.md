# Sprint Plan: M-NAMEDTEST-TEMPFILE-LEAK

**Design:** [Approved design](m-namedtest-tempfile-leak.md)
**Target:** v0.51.1
**Status:** Planned; implementation not started. Design approval is supplied by task-f8f9d4f1; coordinator approval of this plan gates execution.
**Duration:** 2 days (12 hours implementation/verification + 3 hours contingency = 15 hours)
**Risk:** Medium — signal lifecycle, platform differences, concurrent runs, and package publication.
**Estimated size:** 800 LOC (310 implementation/docs + 490 tests).

## Goal and current status

Prevent interrupted named-test bodies from being compiled, executed again, staged, hashed, or published. Keep source-local temporary files so sibling imports and manifest lookup continue working. Use cleanup for handleable signals and consumer exclusions plus PUB024 for unavoidable leftovers.

The checkout is clean on coordinator/task-ec26fd48; std/VERSION is v0.51.0. executor.go still creates `_namedtest_body_*.ail` beside the source and relies on defer for removal. Neither the shared predicate nor PUB024 exists. Both test runners scan the files described by the design. measurePackageQuality is shared by quality and publish, while cmd/registry-validator/main.go independently assembles QualityInputs: the new field must reach both assemblers.

The design's systemic audit covers eight scanning entry points (seven logical consumers; the test command has two runners). Treat the explicit inventory below as authoritative rather than the shorthand count.

## Capacity and registry reuse

The 7-day velocity script found only the design-document commit in this shallow checkout and no usable LOC metrics. The current changelog records recent named-test float and VM builtin work but provides no reliable time denominator. Measured LOC/day is unavailable. Use the design's 12-hour estimate plus 25% contingency; 400 LOC/day is a planning target, not observed velocity. Re-estimate after M1 if lifecycle tests exceed the buffer.

Registry searches: `ailang pkg search tempfile` returned no candidates; `ailang pkg search testing` returned sunholo/testing_utils@0.1.1. Inspected it with `pkg info` and `pkg docs`: it contains pure AILANG assertion helpers, not Go process lifecycle or source discovery utilities. Decision for M1, M2, M3: **none**. These changes belong in the existing Go CLI/compiler tooling; adding an AILANG package cannot implement these boundaries. Search results were obtained with the installed binary, which warned that it may be stale.

## Milestones

### M1: Consumer immunity and PUB024 (~320 LOC)

**Estimate:** 100 implementation + 220 tests; 4 hours. **Dependencies:** None.

Create internal/pkg/testartifact.go and testartifact_test.go with the single prefix, PID-aware creation pattern, strict basename predicate for legacy/new numeric shapes, and sorted artifact discovery. Discovery must cover nested source directories and surface I/O failures rather than silently reporting a clean tree.

Update internal/pkg/discover.go, tarball.go, hasher.go, publish_validator.go; internal/check/package.go; cmd/ailang/check.go and both walks in test.go. Exclude artifacts while collecting TransientFiles for reporting. Add QualityInputs.TransientFiles and a hard PUB024 gate to internal/pkg/quality.go. Wire the shared CLI measurement in cmd/ailang/pkg_quality.go, confirm pkg_publish.go consumes it, and wire cmd/registry-validator/main.go's separate input assembly. Capture debris before attested test execution can sweep it, so the same quality request cannot hide a contaminated starting tree.

Extend existing discover/tarball/hasher/publish-validator/quality tests, check discovery tests, and CLI tests. Plant both legacy and PID-shaped debris at root and in a nested module directory. Compare unchanged clean trees under identical tarball metadata; assert byte equality and hash equality, not just archive member exclusion.

**Acceptance:**
- [ ] Every inventory entry skips artifacts using the shared predicate: check package discovery, pkg discovery, tarball, hash, smoke staging, plain test, package test, directory check.
- [ ] PUB024 is a hard gate in publisher and server modes, names sorted files and a deletion remedy, and is absent on clean trees.
- [ ] pkg quality exits 2 and publish --dry-run refuses planted debris; JSON output remains parseable.
- [ ] Source counts, test counts, tarball bytes and content hash equal the clean baseline despite debris.
- [ ] Root and nested artifacts are covered; nonmatching user filenames remain included.

**Risk:** Missing an independently assembled quality input. Mitigate by enumerating all BuildQualityReport callers and testing server-mode inputs, including a deliberately contaminated extracted archive.

### M2: Producer lifecycle and safe startup sweep (~360 LOC)

**Estimate:** 160 implementation + 200 tests; 4 hours. **Dependencies:** M1.

Create internal/testing/tempbody.go and tempbody_test.go; use platform-specific liveness files if required by build tags. Register paths immediately after creation, unregister/remove on all normal and error returns, and coordinate creation/registration with shutdown so a signal between CreateTemp and registration cannot leak a new file. Cleanup must prevent late registrations or new creations once shutdown starts.

Update executor.go to use the shared PID pattern. Install a command-scoped handler before dispatch in commands_language.go/test.go; library callers must not install handlers or exit. Handle SIGINT/SIGTERM and SIGHUP where supported with conventional exit status. Sweep source directories before either runner enumerates files, including nested directories and the parent of a single-file argument. Probe Unix PID liveness: ESRCH means dead, permission denial means potentially live. Keep live or uncertain owners; legacy files and platforms without probing use the approved one-hour mtime threshold. Report removals/skips/errors to stderr for JSON mode.

Add CLI subprocess tests in cmd/ailang/test_tempbody_test.go. Build/reuse the existing CLI integration harness; wait with a deadline for a real temp body before signalling, never rely on fixed sleep. Use Go-created scratch packages or fixtures under cmd/ailang/testdata/namedtest_tempbody/ with sibling imports and a bounded slow test. Obtain `ailang prompt` before authoring any .ail fixture and check it before running.

**Acceptance:**
- [ ] Normal success and failures leave no body; registered cleanup is race-safe and closes the creation/registration interruption window.
- [ ] SIGTERM exits 143 and SIGINT exits 130 with no body remaining; SIGHUP cleanup is covered on supported platforms.
- [ ] Dead-PID artifacts are removed; a real live child PID survives; permission/unknown probes never authorize deletion.
- [ ] Legacy files younger than one hour survive; older files are swept; unsupported platforms use age fallback and compile successfully.
- [ ] Both runners skip retained debris and sweep nested source directories; plain test executes the baseline count rather than doubling it.
- [ ] Relative sibling imports and package manifest resolution still succeed; internal/testing has no signal-handler side effects for embedded callers.

**Risk:** Flaky interruption tests and platform syscall differences. Bound subprocess waits, always reap children, isolate fixtures, test lifecycle coordination with the race detector, and use platform build constraints.

### M3: Hygiene, documentation and acceptance verification (~120 LOC)

**Estimate:** 50 implementation/docs + 70 tests; 4 hours plus 3-hour contingency. **Dependencies:** M1, M2.

Suppress only transient-body MOD010 warnings in internal/pipeline/mod010.go using the shared predicate. Update cmd/ailang/pkg_init.go to scaffold `_namedtest_body_*.ail` and `.ailang/` ignores without overwriting existing user .gitignore content. Extend MOD010 and pkg-init tests. Update cmd/ailang/guides/package-authoring.md and changelogs/v0.32-current.md's Unreleased section with cleanup, one-hour legacy fallback, manual deletion remedy and PUB024.

Examples/fixtures: create or update the scratch-package integration fixture from M2 and verify existing examples/inline_tests_arithmetic.ail and examples/inline_tests_recursive.ail. No new language feature or public example is necessary; interruption scenarios belong in subprocess fixtures. The named-test sibling-import regression in internal/testing/named_test_env_test.go must remain green.

**Acceptance:**
- [ ] MOD010 remains active for real mismatched source modules and is absent for recognized temp bodies.
- [ ] Newly initialized packages ignore body files; existing .gitignore content is preserved.
- [ ] Guide explains signal cleanup, SIGKILL limitations, safe sweep, PUB024 and deletion; changelog records the fix.
- [ ] SIGKILL subprocess scenario leaves debris, then quality/publish refuse it and the next test run sweeps it without extra tests.
- [ ] Focused tests, targeted race tests, make fmt, make lint, make test-core, full make test and make check-boundaries pass.
- [ ] Windows build checks for internal/testing and cmd/ailang pass; platform-specific signal assertions are explicitly gated.

**Risk:** Full suite prerequisites (e.g. Z3) and validation runtime. Confirm tools before execution, record environmental failures precisely, and spend contingency resolving new failures caused by this sprint.

## Day-by-day execution

Day 1: M1 (4h), then M2 (4h). Establish clean/debris baselines before edits; finish each milestone's focused tests before proceeding.

Day 2: M3 (4h), then contingency (3h) for interruption races, platform builds or full-suite failures. Run the final SIGKILL → PUB024 → sweep sequence with the built CLI in an isolated package and retain output as evaluation evidence.

## Validation and success metrics

Run focused Go tests for internal/pkg, internal/check, internal/testing, internal/pipeline, cmd/ailang and cmd/registry-validator, then `go test -race ./internal/testing` plus relevant CLI lifecycle tests. Run the repository checks listed in M3 once the focused checks pass. For coverage, inspect changed predicate, gate and lifecycle branches using targeted `go test -cover`; all listed behaviors must be exercised, without imposing an unsupported repository-wide coverage percentage.

Verify byte-level tarball equality, content hash equality, stable compile/test counts, PUB024 exit behavior and conventional signal exits. Signal fixtures must exercise source-local imports so a move to os.TempDir cannot accidentally satisfy cleanup tests while breaking packages. No release/tag or implementation is part of this planning task.

## Dependencies, assumptions and handoff

No external package dependency. M1 → M2 → M3. Go toolchain, built CLI, registry access for reuse audit and optional Z3 for existing checks are execution prerequisites. PID reuse may conservatively retain a stale file; PUB024 and explicit deletion remain the remedy. Existing repositories require their own .gitignore update; scaffolding protects newly initialized packages only. Virtual module loading, cache GC and unrelated repository scanners remain deferred.

Progress: `.ailang/state/sprints/sprint_M-NAMEDTEST-TEMPFILE-LEAK.json`. All milestone passes are null. No source issue number was supplied; #1507 is the design-doc PR, not a verified bug issue, so github_issues is empty. The coordinator should hand the plan and JSON to sprint-executor after sprint-plan approval; the final implementation then goes to sprint-evaluator against this checklist and the approved design.
