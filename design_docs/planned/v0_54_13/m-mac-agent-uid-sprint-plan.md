# M-MAC-AGENT-UID Sprint Plan

Design: [M-MAC-AGENT-UID](m-mac-agent-uid.md). Task: task-f0573def. Created: 2026-10-16.
Status: planning complete; implementation and host rollout not started. Prior design approval supplied by coordinator handoff; unchecked operational decisions remain explicit gates, not inferred approvals. No additional quorum is required for this planning-only handoff: the user explicitly says previous work is approved.

## Summary

Run local model-facing CLIs as a dedicated hidden macOS service account while preserving trusted coordinator ingestion and bounded cancellation. Duration: **12 engineering days**, including roughly 25% uncertainty allowance within the design's 10–12 day estimate; host windows and credential provisioning can extend elapsed time. Risk: **high**. Estimated changed code/tests/scripts/docs: **1750 LOC** (roughly 60% implementation/operations and 40% tests/fixtures). No `.ail` changes or language demos; teaching-prompt and showcase syntax gates are inapplicable.

## Current state and velocity

Clean checkout at 67c8eb6d; only the design commit is visible in the last seven days. The velocity script reports no usable recent LOC metrics; its historical changelog matches are not recent velocity. Therefore no measured LOC/day is claimed. Planning capacity is 146 changed LOC/day, a budget rather than demonstrated throughput. The existing five executor process seams and 161-line worktree lifecycle file were inspected; no local UID launcher/ingestion implementation was found. H-4 envpolicy and internal/gitexec are existing reuse seams.

Each milestone is scoped to one 60–90 minute executor work unit with targeted tests; days below reserve additional time for security review, host verification and fixes. If a unit exceeds 90 minutes, split it without weakening acceptance. M1 requires native rig access and gates all implementation, matching the approved design. A cloud executor can prepare scripts but must stop before M2 until the rig transcript is banked. No fabricated Phase 0 result.

## Registry reuse audit

Executed `ailang pkg search process`, `git`, and `macos`. Git/macOS returned no packages; process returned duckdb and external_backend. Inspected `sunholo/external_backend` info/docs: it wraps AILANG std/process JSON output and cannot provide the Go/macOS authority boundary. DuckDB is unrelated database access. All milestones choose **none** for registry dependencies and reuse repository Go helpers. Per-milestone decisions are recorded in JSON. CLI warned the local binary may be stale; registry search succeeded, and no runtime security behavior was inferred from it.

## Milestones

### M1: Host provisioning and positive-control probes (~180 LOC)

**Dependencies:** none
**Budget:** one 60–90 minute execution unit; code/test allocation approximately 60/40.
**Files and executable examples:** `tools/launchd/install_agent_user.sh; tools/launchd/agent_uid_probe.sh`. Shell probes and hostile-repository fixtures are the runnable examples for this infrastructure change.

Create collision-checked hidden account and root-owned bin root; inventory Daneel dependencies and CLI locations before changing permissions.

- [ ] Record same-UID sentinel visibility and cross-UID denial on the Mac; missing positive control or cross-UID leak stops all later milestones.
- [ ] Record home-deny choice, UID/GID and file/keychain/write probe results without logging secrets; never reuse daneel.
- [ ] Resolve staging group membership explicitly: agent needs read/traverse on parent staging; operator needs read on agent output; agent cannot write parent staging.

**Risk:** Rig access and D5 approval; do not infer denial from missing files.

### M2: Launcher protocol and validation (~190 LOC)

**Dependencies:** M1
**Budget:** one 60–90 minute execution unit; code/test allocation approximately 60/40.
**Files and executable examples:** `cmd/ailang-agent-launcher/main.go; internal/agentlaunch/spec.go; internal/agentlaunch/spec_test.go`. Shell probes and hostile-repository fixtures are the runnable examples for this infrastructure change.

Implement bounded stdin JSON spec, run/signal/cleanup contract, identity and path checks; allow root-owned CLI paths only.

- [ ] Malformed/oversized specs, wrong UID, escaping workspace and disallowed commands return named errors.
- [ ] Protocol preserves CLI stdin separately from spec framing and keeps stdout/stderr compatible with executor parsers.
- [ ] Restrict cleanup and signal requests to tracked task resources; no arbitrary path deletion.

**Risk:** Protocol framing and sudo child stdin compatibility.

### M3: Launcher lifecycle and cancellation (~180 LOC)

**Dependencies:** M2
**Budget:** one 60–90 minute execution unit; code/test allocation approximately 60/40.
**Files and executable examples:** `internal/agentlaunch/lifecycle.go; internal/agentlaunch/lifecycle_test.go`. Shell probes and hostile-repository fixtures are the runnable examples for this infrastructure change.

Enforce deadline inside agent UID; route cancellation through fixed launcher; preserve process-group error reporting.

- [ ] Hung child and grandchild terminate within configured deadline plus a documented bounded grace period.
- [ ] Cancellation, parent pipe loss, stuck sudo and stale/reused process IDs have explicit tested outcomes.
- [ ] D7 operational contract recorded before rig activation; no direct cross-UID kill assumption.

**Risk:** macOS process groups and parent loss require rig integration evidence.

### M4: Shared spawn seam and audit metadata (~180 LOC)

**Dependencies:** M3
**Budget:** one 60–90 minute execution unit; code/test allocation approximately 60/40.
**Files and executable examples:** `internal/executor/agentlaunch_darwin.go; internal/executor/agentlaunch_other.go; internal/executor/agentlaunch_test.go; internal/executor/executor.go`. Shell probes and hostile-repository fixtures are the runnable examples for this infrastructure change.

Add default-off host flag and typed unsupported-platform errors; consume H-4 environment and bank effective UID/spec digest.

- [ ] Child env equals BuildEnvironment output plus documented HOME/USER/LOGNAME/PATH overrides; no operator-home PATH entries.
- [ ] Missing launcher, sudo failure and wrong UID fail before CLI execution with no same-UID fallback.
- [ ] Flag-off path retains current behavior; task content cannot enable or disable host identity policy.

**Risk:** H-4 remains sole env authority; avoid storing credentials in audit digest inputs/output.

### M5: Wire all executor launch paths (~160 LOC)

**Dependencies:** M4
**Budget:** one 60–90 minute execution unit; code/test allocation approximately 60/40.
**Files and executable examples:** `internal/executor/{pi,claude,codex,opencode,motoko}/agentlaunch.go; internal/executor/*/agentlaunch_test.go`. Shell probes and hostile-repository fixtures are the runnable examples for this infrastructure change.

Integrate five executors including auxiliary health/auth/version subprocesses and codex persistent daemon lifecycle; audit every process spawn.

- [ ] Each lane uses the shared seam for model-facing children, including daemon reuse; reject any existing same-UID daemon in split mode.
- [ ] Deadline/cancellation and JSON output parsing pass targeted executor tests.
- [ ] Extract helpers from claude.go (798 lines), codex.go (782), opencode.go (768), motoko.go (770) and pi.go (682) into companion files before exceeding 800; add new test companions rather than growing near-limit tests.

**Risk:** Auxiliary commands and daemon reuse can bypass the boundary.

### M6: Parent-owned staging and agent workspace (~180 LOC)

**Dependencies:** M1, M4
**Budget:** one 60–90 minute execution unit; code/test allocation approximately 60/40.
**Files and executable examples:** `internal/coordinator/daemon_tasks_staging.go; internal/coordinator/daemon_tasks_staging_test.go`. Shell probes and hostile-repository fixtures are the runnable examples for this infrastructure change.

Prepare parent-owned bare staging and agent-owned per-task workspace via launcher; isolate caches and scratch.

- [ ] Agent can fetch staging and write its workspace; cannot mutate staging; operator can collect permitted output.
- [ ] Coordinator never treats agent workspace as its Git repository, including status, diff, commit and cleanup paths.
- [ ] Missing permissions, unsafe task paths and unreadable output fail loudly.

**Risk:** Design group table is ambiguous; M1 must prove actual permissions.

### M7: Hardened committed-output ingestion (~190 LOC)

**Dependencies:** M6
**Budget:** one 60–90 minute execution unit; code/test allocation approximately 60/40.
**Files and executable examples:** `internal/coordinator/daemon_tasks_ingest.go; internal/coordinator/daemon_tasks_ingest_test.go; internal/gitexec/`. Shell probes and hostile-repository fixtures are the runnable examples for this infrastructure change.

Fetch from parent-owned repository context with protected config and one-invocation safe.directory; do not use broad trust.

- [ ] T1 hooks, T2 fsmonitor, T3 filter and T4 include.path fixtures never execute an operator sentinel.
- [ ] Parent receives expected committed diff; poisoned config/env and external Git helpers cannot redirect ingestion.
- [ ] Reuse internal/gitexec and H-6 D4; do not assume cloud ingestion implementation already exists.

**Risk:** Local transport and Git configuration must be hardened across all parent invocations.

### M8: Uncommitted output and workspace lifecycle (~170 LOC)

**Dependencies:** M7
**Budget:** one 60–90 minute execution unit; code/test allocation approximately 60/40.
**Files and executable examples:** `internal/coordinator/daemon_tasks_collect.go; internal/coordinator/daemon_tasks_collect_test.go; internal/coordinator/daemon_tasks_worktrees.go`. Shell probes and hostile-repository fixtures are the runnable examples for this infrastructure change.

Collect dirty output using bounded symlink-safe copy into trusted staging; disable split-lane auto-commit in agent tree; add sweeper.

- [ ] T5 symlink traversal and replacement races, special files and out-of-root paths are rejected; no operator content reaches PR diff.
- [ ] Dirty files and deletions round-trip with explicit limits; failed collection does not silently lose output.
- [ ] Terminal-task cleanup only removes task-owned resources; parent does not execute agent Git config.

**Risk:** TOCTOU-safe collection is security-sensitive; split further if review cannot fit one run.

### M9: Credential and executable cutover inventory (~100 LOC)

**Dependencies:** M5, M8
**Budget:** one 60–90 minute execution unit; code/test allocation approximately 60/40.
**Files and executable examples:** `docs/docs/guides/mac-agent-uid.md; tools/launchd/agent_uid_probe.sh`. Shell probes and hostile-repository fixtures are the runnable examples for this infrastructure change.

Record D3 holder decisions and relocate executables outside operator home; use existing lane lease and H-4 grants.

- [ ] Claude uses agent-only setup-token; codex uses separately authorized auth.json or API-key mode; never copy operator/cloud refresh token.
- [ ] Pi/opencode/motoko provider grants, per-task cache, JSONL collection and ollama lease work under agent UID.
- [ ] Inventory contains holder identifiers and permissions only; no credential values; executable symlinks into operator home rejected.

**Risk:** Subscription provisioning and token ownership need operator access.

### M10: Coordinator lane canaries (~80 LOC)

**Dependencies:** M9
**Budget:** one 60–90 minute execution unit; code/test allocation approximately 60/40.
**Files and executable examples:** `docs/docs/guides/mac-agent-uid.md; design_docs/planned/v0_54_13/m-mac-agent-uid.md`. Shell probes and hostile-repository fixtures are the runnable examples for this infrastructure change.

Roll out pi, opencode, motoko, codex, claude in order; record one probe and real task per lane before next.

- [ ] Each lane proves UID, secret denial, bounded cancellation, expected PR diff, parent push and JSONL import.
- [ ] Flag remains off for lanes without green evidence; record mechanism for lane-scoped activation despite host-side flag.
- [ ] Rollback disables dispatch first and documents return to exposed same-UID mode; never silently roll back on spawn error.

**Risk:** Host flag alone does not express per-lane rollout; require explicit lane activation policy.

### M11: Mission and nightly integration (~70 LOC)

**Dependencies:** M10
**Budget:** one 60–90 minute execution unit; code/test allocation approximately 60/40.
**Files and executable examples:** `docs/docs/guides/mac-agent-uid.md; tools/launchd/agent_uid_probe.sh`. Shell probes and hostile-repository fixtures are the runnable examples for this infrastructure change.

Audit mission driver direct CLI and post-run Git paths; run one watched mission iteration then nightly canary.

- [ ] Direct agent invocations cross the seam; post-run Git inspections use trusted ingestion context, including read-only status/diff.
- [ ] Coordinator/watchdog/ollama remain operator LaunchAgents; gateway admission and lease rejection work.
- [ ] All new shell glue runs on rig bash 3.2; observation transcript and stop conditions recorded.

**Risk:** Mission driver Git inspection in design rollout prose conflicts with D4; D4 governs every Git invocation.

### M12: Documentation and focused release checks (~70 LOC)

**Dependencies:** M11
**Budget:** one 60–90 minute execution unit; code/test allocation approximately 60/40.
**Files and executable examples:** `docs/docs/reference/env-vars.md; docs/docs/guides/mac-agent-uid.md; changelogs/unreleased/2026-10-16-m-mac-agent-uid.md`. Shell probes and hostile-repository fixtures are the runnable examples for this infrastructure change.

Document account/sudoers install, permissions, rotation, rollout/rollback and residual same-agent-UID sharing/network access.

- [ ] make test-changed, make lint, make check-boundaries, make check-file-sizes and make check-changelog pass for final implementation.
- [ ] Shell bash -n and native bash 3.2 checks pass; sudoers validated with visudo before installation.
- [ ] Design AC1–AC6 evidence linked; AC7 uses focused executor tests in place of make test, which remains CI responsibility.

**Risk:** Do not report rig acceptance from Linux-only tests.

## Day-by-day schedule

| Day | Work and evidence |
|---|---|
| 1 | M1: account/bin inventory, approved home policy, live positive-control probes. Stop on failed premise. |
| 2 | M2: launcher spec and validation, review stdin framing. |
| 3 | M3: cancellation/deadline tests and D7 operational ratification. |
| 4 | M4: shared spawn seam and H-4 env/failure tests. |
| 5 | M5: all executor paths, daemon reuse audit and file-size extractions. |
| 6 | M6: staging/workspace permissions and trusted-context audit. |
| 7 | M7: committed ingestion and T1–T4 adversarial fixtures. |
| 8 | M8: dirty output, T5 race/symlink fixtures and cleanup. |
| 9 | M9: credential holders, executable relocation and logs/gateway probes. |
| 10 | M10: sequential coordinator canaries, subscription lanes last. |
| 11 | M11: watched mission iteration and nightly canary. |
| 12 | M12: final focused checks, runbook, evidence review and contingency fixes. |

## Validation and completion

For each changed Go milestone run targeted package tests and `make test-changed` (includes test-core); use cross-build checks for Darwin code on Linux, and native rig tests for actual UID behavior. Run final lint, boundaries, file-size and changelog gates. Do not require full `make test` in cloud executor; CI owns that suite. No stdlib/examples verification target needed unless scope actually touches those paths. Test coverage goal: every boundary failure and each AC1–AC6 path has a positive/negative assertion; no unsupported repository-wide percentage claim.

Bank only sanitized sentinel transcripts, never real process environments or secret file contents. Existing credential absence is not proof of denied access: create safe operator-owned sentinels at representative paths when necessary. Record exact OS/UID/GID and probe status. Tests on one shared UID cannot establish macOS isolation.

M1 resolves the design's staging group ambiguity with measured read/traverse permissions. M10 requires an explicit host-controlled per-lane activation mechanism. M11 follows D4 strictly even where the design's rollout prose suggests parent Git inspection inside an agent tree. If those require a substantive departure from the approved boundary, return the affected decision for review rather than weaken it.

## Handoff and resumption

Progress: `.ailang/state/sprints/sprint_M-MAC-AGENT-UID.json`. All milestones begin unpassed. Resume at the first unmet dependency; record native-rig evidence before passing host milestones. Coordinator plan approval/merge is the execution handoff; this planning task does not start sprint execution, change host accounts, install credentials or activate the flag. The coordinator should route the resulting approved plan to sprint-executor, then sprint-evaluator. No separate agent message is needed from this worktree.
