# M-CLOUD-NATIVE-SESSION-JSONL Sprint Plan

**Status:** Implementation delivered on coordinator/task-53b8873a; focused provider/artifact verification passes; broad-gate limitations recorded below. Independent coordinator evaluation and post-deployment canaries remain pending.
**Design:** [Approved design](m-cloud-native-session-jsonl.md) · **Issue:** #1967 · **Target:** v0.54.19
**Duration:** 2 working days, approximately 11 hours including 3.5 hours of integration/operational buffer.
**Risk:** Medium (session identity, early returns, gcsfuse visibility).

## Goal and current status

Persist the native Claude, pi, Codex and motoko session at the existing flat task artifact path, preserving every byte and exposing misses. Keep pi persistence cloud-only. No Terraform, registry dependency, artifact layout or language semantics changes.

Code inspection confirms pi disables persistence, cloud artifact writing searches only Claude paths, and motoko already banks a path that cloud ignores. Provider persistence/banking and cloud copy/metrics implementation are complete. The approved design audits all four active provider paths and supplies a unified ProviderData resolution contract.

## Estimate basis

The velocity script was run for seven days. This shallow checkout contains only the design commit (71ee9ab6); it supplies no measured implementation LOC/day. Old changelog counts are not current velocity. Estimates below are provisional task estimates, not measured throughput. Budget 7.5 hours across five milestones (each 60–90 minutes) plus 3.5 hours for integration, canary scheduling and uncertainty. This reconciles the design’s inconsistent 11-hour and 14-hour summaries into a two-day engineering budget; cloud validation may finish after merge.

## Registry reuse audit

Registry searches session and jsonl found no native CLI-session file banking/copy facility. sunholo/agui info/docs inspected: AILANG AG-UI codecs cannot preserve arbitrary native JSONL or implement Go executor filesystem plumbing. Reuse existing ProviderData, motoko banking, config accessors and Go os/filepath instead.

Every milestone has action `none` in registry_reuse. Search results for session were billing_stripe/audio (unrelated domains); jsonl returned agui. No applicable package dependency is added.

## Milestones

### M1: Persist and bank pi cloud sessions (~210 LOC)
**Estimate:** 110 implementation/docs + 100 tests; up to 90 minutes.
**Dependencies:** None
**Files:** internal/executor/pi/session.go, session_test.go, args.go, pi.go, pi_args_extensions_test.go
**Approach and risk mitigation:** Generate UUID once before launch and pass the same identity through argv and resolution; use companion helpers to keep pi.go below 800 lines.
**Acceptance:**

- [x] Cloud args use a generated UUID with --session-dir and --session-id; local/eval args retain --no-session.
- [x] Suffix match precedes newest-file fallback in the task-scoped directory; missing files do not invent a path.
- [x] Completed and failed post-launch results preserve existing ProviderData and bank pi_session_jsonl when available.

### M2: Bank Codex and Claude native session paths (~210 LOC)
**Estimate:** 100 implementation/docs + 110 tests; up to 90 minutes.
**Dependencies:** None
**Files:** internal/executor/codex/session.go, session_test.go, codex.go; internal/executor/claude/session.go, session_test.go, claude.go
**Approach and risk mitigation:** Keep new logic in session.go companions: codex.go is 782 lines and claude.go is 798. Avoid duplicating banking across five result-return branches; use a finalization helper or deferred result enrichment.
**Acceptance:**

- [x] Codex matches session UUID in rollout filenames before newest fallback under isolated CODEX_HOME/sessions.
- [x] Claude resolves the actual session ID under CLAUDE_CONFIG_DIR or the default ~/.claude/projects.
- [x] All post-launch return paths with results bank native paths without dropping existing ProviderData; fixtures cover match, fallback and missing files.

### M3: Resolve and copy provider sessions in cloud artifacts (~220 LOC)
**Estimate:** 120 implementation/docs + 100 tests; up to 90 minutes.
**Dependencies:** M1, M2
**Files:** cmd/ailang/coordinator_cloud_session.go, coordinator_cloud_session_test.go, coordinator_cloud_github.go, coordinator_cloud.go
**Approach and risk mitigation:** Extract session resolution/copy out of the 797-line github file, moving findSessionJSONL if useful. Inject read/sleep behavior for retry tests; keep production retries bounded.
**Acceptance:**

- [x] writeTaskArtifacts takes the canonical provider name and its call sites compile.
- [x] ProviderData is primary; failed or empty candidates permit provider-specific fallback; Claude searches bucket, config and default roots, pi its task dir, Codex its isolated home, motoko its banked path.
- [x] Native bytes are copied unchanged with os.WriteFile to tasks/{taskID}/session.jsonl, with no size cap.
- [x] Reads retry up to three total attempts at two-second intervals; success emits the exact written byte-count log, exhausted failure names provider, session and searched paths.
- [x] Deterministic-bump skips session handling; nil result and transcript behavior remain valid.

### M4: Verify artifact landing and metrics for every provider (~165 LOC)
**Estimate:** 25 implementation/docs + 140 tests; up to 90 minutes.
**Dependencies:** M3
**Files:** cmd/ailang/coordinator_cloud_session_test.go, coordinator_cloud_session.go, coordinator_cloud_github.go
**Approach and risk mitigation:** Compute copy outcome before serializing metrics. Use temp directories and isolated environment; avoid real GCS and provider credentials.
**Acceptance:**

- [x] Fixture tests for claude, pi, codex and motoko each assert byte-identical session.jsonl at the task artifact destination.
- [x] metrics.json includes session_provider and session_jsonl_bytes matching the successful copy; missing sessions report zero bytes.
- [x] Tests capture success and warning output, including searched paths, unreadable/empty files, retry exhaustion, write failure and deterministic-bump silence.
- [x] A payload larger than the existing log truncation threshold retains full tool-result content; unrelated transcript and metrics fields survive.

### M5: Run focused gates and document cloud canaries (~30 LOC)
**Estimate:** 20 implementation/docs + 10 tests; up to 90 minutes.
**Dependencies:** M4
**Files:** changelogs/unreleased/2026-10-27-cloud-native-session-jsonl.md; design_docs/planned/v0_54_19/m-cloud-native-session-jsonl-sprint-plan.md
**Approach and risk mitigation:** Run gofmt on touched Go files. CI owns the full suite. No .ail code or runnable language example is needed for this Go artifact plumbing change.
**Acceptance:**

- [ ] make test-changed, make check-file-sizes, make check-changelog and make lint pass; make check-boundaries verifies touched imports.
- [x] Add changelogs/unreleased/2026-10-27-cloud-native-session-jsonl.md opening with a ### heading.
- [x] Plan documents post-merge canaries for pi, codex, claude and motoko plus pi-go/codex-go routing verification, with task IDs, artifact byte counts and log evidence recorded when executed.
- [x] No live deployment or canary is claimed by offline tests; blocked cloud access is recorded as pending operational validation.

## Day-by-day execution

Day 1: M1 pi persistence and fixtures; M2 Codex/Claude banking and return-path audit; M3 cloud copy helper and initial tests. Reserve remaining time for identity/lookup integration.
Day 2: M4 cross-provider artifact tests and metrics; M5 focused gates and changelog. Use the buffer for integration fixes and preparing operational validation. Execute sequentially; no parallel workers required.

## Validation and success metrics

Run provider package tests while implementing M1/M2 and focused artifact tests in cmd/ailang for M3/M4. Finish with `make test-changed` (changed packages plus test-core), `make check-file-sizes`, `make check-changelog`, `make check-boundaries`, and `make lint`. If origin/dev is unavailable, set TEST_CHANGED_BASE to the recorded pre-implementation commit rather than skipping changed-package tests. CI runs the full suite. No coverage percentage is invented; require all four provider landing fixtures and the failure/retry matrix above.

Fixture examples replace .ail demos: representative native JSONL remains opaque and byte-identical, including tool results beyond 4.7 KB. No .ail files are planned, so the AILANG syntax gate is inapplicable.

Post-merge operational check: dispatch one canary for each active provider, inspect tasks/{taskID}/session.jsonl and metrics.json in gs://ailang-multivac-ailang-artifacts, and correlate exactly one success/miss outcome per run in Cloud Logging. Check pi-go/codex-go select canonical pi/codex names. Record task IDs, object sizes and log evidence here; verify Holosun’s next nightly mirror can count these tasks. These checks are pending deployment and are distinct from offline sprint acceptance. Native session absence on a real failed run must produce a searched-path warning.

## Dependencies, decisions and handoff

No external dependency blocks implementation. Reuse config.CloudRunExecution and config.CodexHome; do not add environment variables. Keep newest fallback restricted to task-scoped pi storage and isolated cloud Codex storage. Preserve motoko’s existing key. Reuse existing UUID tooling. Return actual write byte counts; never substitute reconstructed transcript content for a missing native session.

This plan and its populated JSON are the coordinator handoff artifacts. Per sprint-planner/resources/coordinator.md, merging the sprint-plan PR approves the plan and triggers sprint-executor. Do not self-merge or start implementation during planning. Sprint-evaluator follows execution. Live canary evidence remains pending until deployment.

## Execution record — 2026-10-27

- Actual work branch: `coordinator/task-53b8873a` (the execution checkout differs from the planner handoff branch). Branch-sync helper reports `skipped:no-task-id`; no branch switch, merge, deployment or remote publication is claimed.
- Implementation uses `internal/executor/nativesession` as a shared companion for identity lookup and evidence banking, avoiding duplicated traversal across providers. Newest fallback remains restricted to task-scoped pi storage and explicitly isolated cloud Codex homes. Claude projects-root symlinks are resolved before walking.
- Claude banking uses the resumed identity and the CLI-reported final identity rather than a generated task UUID. Result finalizers preserve provider evidence on failed runs.
- Fixtures cover all four native artifact destinations, full tool-result bytes, metrics, retries, searched-path warnings, empty/unreadable/missing sessions, write errors, provider fallback and deterministic-run silence. No `.ail` edits or runnable-language examples are involved.
- `make lint` cannot run: `golangci-lint` is absent. No toolchain installation was attempted under the unattended-run policy.
- Baseline `b7b7093e` reproduces `TestCodexCancellationKillsMCPProcessGroup` and `TestCodexExitedLeaderStillStopsOwnedChildBeforeReturn`; these process-group failures predate this sprint.

### Operational validation after deployment

| Route | Task ID | Artifact bytes | Success/miss log | Status |
|---|---|---|---|---|
| pi | pending | pending | pending | Await deployment and dispatch |
| codex | pending | pending | pending | Await deployment and dispatch |
| claude | pending | pending | pending | Await deployment and dispatch |
| motoko | pending | pending | pending | Await deployment and dispatch |
| pi-go → pi | pending | pending | pending | Verify canonical provider routing |
| codex-go → codex | pending | pending | pending | Verify canonical provider routing |

For each task inspect `tasks/{taskID}/session.jsonl` and `metrics.json` in the artifacts bucket, reconcile the object size with `session_jsonl_bytes`, and record exactly one session success/miss outcome from Cloud Logging. Confirm the next Holosun nightly mirror counts these sessions. These are operational follow-ups, not claims made by local fixtures.

### Offline validation results

- PASS: focused native-session fixtures across `cmd/ailang`, `internal/executor/pi`, `internal/executor/codex`, `internal/executor/claude`, and `internal/executor/nativesession`.
- PASS: `make check-file-sizes`, `make check-changelog`, `make check-boundaries`, formatting and `git diff --check`.
- `make test-changed TEST_CHANGED_BASE=b7b7093e4abcbfce6fa8b0f7e58d8294978e0a4d` failed in unchanged process-policy fixtures and Codex process-group fixtures. `TestRunPolicy_DeadlineStopsExecutionWithoutProtocolGrace` requires the absent `ps` executable; `TestRunPolicy_TimeoutKillsDescendants` and `TestRunPolicy_SupervisorDeathStopsWorker` report surviving subprocesses. The two Codex failures were independently reproduced on baseline. Claude, pi and shared lookup packages passed their full changed-package runs. No full `make test` was run.
- `make test-core` is not green: `TestRuntimeNestedStopUsesRemainingDeadline` and `TestProcessExec_TimeoutKillsGrandchildren` fail identically on the unchanged baseline. The pipeline test process was killed (`signal: killed`) during resource-intensive verification; pipeline verification remains for CI.
- Lint remains unverified because `golangci-lint` is absent. Independent sprint evaluation is the coordinator's next stage; no self-evaluation score or merge approval is asserted.

Final checkpoint: implementation and fixture milestones M1–M4 pass. M5 documentation, formatting, file-size, changelog and boundary work is complete, but its all-green test/lint acceptance remains unchecked for the explicit environment/baseline failures above. The sprint record preserves `M5.passes=false`; implementation completion does not assert gate approval. All changes are committed for coordinator evaluation, and no deployment or canary was attempted.
