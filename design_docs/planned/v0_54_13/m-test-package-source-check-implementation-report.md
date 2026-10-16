# M-TEST-PACKAGE-SOURCE-CHECK implementation report

Implemented the approved default full package check, fail-fast execution, `--no-source-check`, additive suite reporting, and quality subprocess opt-out. `check.CheckPackageFiles` remains the single implementation of compilation, InterFunctionRefs, and StrictFallbacks. File/directory test mode and coordinator cascade were not changed.

## Regression evidence

The fresh baseline reproduced the report: a green test plus an unimported bad module exited 0 under package test and 1 under package check. New tests failed before wiring: broken zero-test packages incorrectly exited 0; zero-test JSON was a human warning; quality repeated validation. They pass after implementation.

Coverage: exported and non-exported errors; declared test modules; orphan/hidden/node_modules/_vendor/debris exclusions against standalone check; both JSON flags across green/broken and with/without tests; zero tests executed on source failure; allow-skips cannot waive source failure; opt-out and file mode omit source_check; bytecode flags; unchanged zero-test example; quality subprocess argument and summary compatibility. Reporter tests pin existing keys, round-trip source_check, and both success predicates.

Inter-function references use the unchanged checker without filtering any failure category. An independent post-compilation InterFunctionRefs failure fixture was not found: forward function references in the current compiler are reordered correctly, and invalid let references fail earlier during type checking. This is a coverage limitation for independent evaluation, rather than a claim that an AILANG interref-failure fixture was exercised.

No public `.ail` source was changed. AILANG prompt version loaded: v0.16.7 from the freshly built repository binary. Green temporary fixture was validated by `check --package`; deliberately invalid fixture failed with string/int unification diagnostics. Contracts were omitted because these fixtures isolate compiler entry-point behavior; source functions explicitly use `pure`, and named tests exercise the companion module. Performance modules use the same pure/no-contract/named-test pattern. Existing example declarations are unchanged.

## Cache measurements

Linux amd64 container, Go 1.26.9, 6 visible CPUs, about 7.75 GiB RAM. This is not the reporter’s 2-core macOS/CI environment. Other validation/tool installation ran concurrently, so these are single-sample observations, not a controlled performance benchmark.

Binary: `bin/ailang`, source base `b799aa607baf5411f72132733d650bda87a8bfe6`, built with dirty sources at 2026-10-16_13:56:39. SHA-256: `5d811cd663d6108d43b18940f786c8df23a3ffc3969cd7724fddd46d5dec017a`. It contains the implementation; later changes were tests/docs/state only.

Generated package: 52 modules, 51 exports: shared.ail defines pure one() = 1; m00–m49 each import one and return one() + index; perf_test imports m49 and asserts value() == 50. All commands exited 0. Temporary fixture and raw JSON/stdout/stderr were retained at `/tmp/ailang-source-measure-hulgq_s6` during this run (ephemeral, not repository artifacts).

Method: `time.perf_counter()` around subprocesses, stdout/stderr captured separately; no shared user cache deleted. `AILANG_CACHE_DIR` pointed to fresh task-owned integrated-cache, baseline-cache, and debug-cache directories under that temporary root. `AILANG_NO_CACHE` was absent. Before and after every command, ran `ailang cache compile-stats` with the same cache override and fixture cwd. `cache stats` is the unrelated brain cache; compile-stats is the supported compilation-cache command. Its displayed default path is hardcoded, but its counts honor AILANG_CACHE_DIR.

| Run | Command | Wall seconds | Cache entries before → after | Diagnostic hits / misses |
|---|---|---:|---:|---:|
| cold-integrated | `ailang test --package . --json` | 2.1118 | 0 → 54 | not exposed |
| warm-integrated | `ailang test --package . --json` | 1.4228 | 54 → 54 | not exposed |
| cold-baseline-check | `ailang check --package .` | 1.9851 | 0 → 54 | not exposed |
| baseline-optout-test | `ailang test --package . --no-source-check --json` | 0.1892 | 54 → 54 | not exposed |
| warm-baseline-check | `ailang check --package .` | 1.1023 | 54 → 54 | not exposed |
| warm-baseline-optout-test | `ailang test --package . --no-source-check --json` | 0.1857 | 54 → 54 | not exposed |
| cold-debug-check | `ailang check --package --debug-compile .` | 1.3239 | 0 → 54 | 50 / 54 |
| warm-debug-check | `ailang check --package --debug-compile .` | 0.9084 | 54 → 54 | 104 / 0 |

Recorded compile-time totals were 0ms throughout (tiny modules/submillisecond resolution); these totals cannot establish time saved. There were 54 entries: 52 package modules and two prelude modules. Diagnostic check collected 52 cache-summary records, cold 50/104 hits (48.1%), warm 104/104 hits (100%). These counts describe repeated checks, not the integrated test’s separate phases. The supported debug trace was collected with flags before the package path (`check --package --debug-compile .`).

Integrated test exposes no per-phase cache-hit counters. Entry counts and timing alone cannot establish pre-pass-to-test hit rate or a single compile. Diagnostic warm checking demonstrates cache reuse, and integration has the same compiler/cache path, but exact cross-phase attribution remains unavailable. The synthetic timings do not reproduce the reported ~200 seconds saved; no speedup or single-invocation guarantee is made. Artifact sharing remains deferred; investigate it with representative workload profiling if real CI reuse disappoints.

## Validation

- `make build`: passed using `/usr/local/go/bin` on PATH. The first attempt found no go on PATH; the existing installation was used.
- `bash .agents/skills/sprint-executor/scripts/validate_sprint_json.sh M-TEST-PACKAGE-SOURCE-CHECK`: passed.
- Session startup script ran; its bc-based display lacks bc and its swallowed preflight output was not usable as test evidence. Explicit gates below are authoritative.
- Reporter red/green: `go test ./internal/testing -run ^TestSourceCheckReport$`; initial missing type/field failure, then passed.
- CLI/quality red/green: `go test ./cmd/ailang -run "TestPackageSourceCheck|TestQualitySourceCheck" -count=1`; initial behavior failures, then passed.
- `go test ./internal/testing -count=1`: passed.
- Final targeted regressions: `go test ./cmd/ailang -run "TestPackageSourceCheck|TestQualitySourceCheck|TestTestCommand_JSON|TestTestCommand_.*Path|TestCheckPackage_StrictFallback" -count=1` passed (12.963s). Existing multi-path tests are also covered by the full cmd/ailang gate.
- Live `bin/ailang pkg quality --json <green-fixture>`: exit 0, schema `ailang.package-quality/v1`, compile ok (3 files), tests 1 passed/0 failed. `bin/ailang test --help` inspected successfully.
- `make test-changed TEST_CHANGED_BASE=b799aa60`: failed on existing policy/process tests in cmd/ailang (409.546s): `TestRunPolicy_DeadlineStopsExecutionWithoutProtocolGrace` requires missing `ps`; `TestRunPolicy_TimeoutKillsDescendants` and `TestRunPolicy_SupervisorDeathStopsWorker` report surviving subprocesses. New regressions did not fail; internal/testing passed (36.796s). Because the package gate failed, the target did not reach test-core; it was run explicitly.
- `make install-lint`, then `make lint` with `/home/ailang/go/bin` and `/usr/local/go/bin` on PATH: passed, 0 issues (repository-pinned v2.11.4; initially absent).
- `make check-boundaries check-file-sizes check-changelog`: passed.
- `git diff --check`: passed.

Full `make test` belongs to CI under sprint-executor instructions. No example source changed, so broad verify-examples was not required. No parser/type/effect/runtime semantics or registry dependencies changed. New tests use Go helpers without external binaries or native-path assertions. Independent sprint evaluation remains the coordinator’s next stage.

The first explicit `make test-core` ran alongside initial lint analysis and failed in unchanged `internal/effects/TestRuntimeNestedStopUsesRemainingDeadline` (nested fixture survived bounded stop); `internal/pipeline` was killed amid memory pressure. A clean detached worktree at baseline b799aa60 reproduced the exact effects failure using `GOFLAGS=-p=1 AILANG_TEST_FAST_LOOP=1 go test ./internal/effects -run '^TestRuntimeNestedStopUsesRemainingDeadline$' -count=1` (2.644s). No effects/pipeline sources were modified. `GOFLAGS=-p=1 make test-core` also failed: effects reproduces the same nested-stop failure (32.644s package run); pipeline is killed (16.682s), while all other core packages pass. A clean baseline worktree also reproduces the killed pipeline test process (`GOFLAGS=-p=1 AILANG_TEST_FAST_LOOP=1 go test ./internal/pipeline -count=1`, 13.771s). The same baseline worktree reproduced all three policy failures using `GOFLAGS=-p=1 go test ./cmd/ailang -run 'TestRunPolicy_(DeadlineStopsExecutionWithoutProtocolGrace|TimeoutKillsDescendants|SupervisorDeathStopsWorker)$' -count=1` (36.712s): missing ps, surviving grandchild, surviving worker/subprocess. These gates are not being weakened or silently waived.


## Delivery and remaining validation

Implementation is committed on `coordinator/task-0e0bbf9c` (the workspace-provided coordinator branch). M1–M5 pass. M6 measurements and all requested gates were executed, but its all-green acceptance criterion remains unmet because of reproduced baseline/container failures. Sprint execution is finished with M6 `passes:false`; this is not a merge approval or a clean evaluator verdict. Re-run the gates in an environment with ps, suitable process supervision, and enough memory before approval. The baseline worktree was clean and is removed after evidence collection.

The focused tests use the newly built `buildAilang` helper binary; cache measurements and help/quality smoke use freshly built bin/ailang. No shared cache was cleared. No production changes followed the measured build.

Friction: Go was installed off PATH; the linter required the repository install target; the startup display lacks bc; process-lifetime tests and pipeline memory requirements prevent a green container-wide gate. Future performance work needs integrated per-phase cache telemetry and representative CI profiling rather than assuming artifact sharing will save time.
