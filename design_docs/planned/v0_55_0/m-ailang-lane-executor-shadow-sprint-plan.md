# Sprint Plan: M-AILANG-LANE-EXECUTOR-SHADOW

## Summary
Every sampled real `ailang_only`-lane coordinator task also runs on the other harness (pi ↔ motoko) as a side-effect-free child task. Both outcomes are graded, and a weekly report lists every disagreement.

**Design doc:** [m-ailang-lane-executor-shadow.md](m-ailang-lane-executor-shadow.md) (ratified by Mark 2026-10-13, D1–D7; quorum rounds 1–2 recorded).
**Duration:** 6 working days (7 milestones; each sized to one executor run of about 60–90 min, plus integration).
**Dependencies:** v0.54.8 released (motoko image with a warm compile cache, cache-stable compaction, OpenRouter routing parity); `OPENROUTER_API_KEY_SHADOW` provisioned with a monthly cap before M7's dev-plane enable.
**Risk level:** Medium. It touches the shared dispatch and finalization paths; the isolation guarantees are what must not regress.

## Current Status Analysis

### Velocity (last 7 days, `analyze_velocity.sh 7`)
- Coordinator and executor work landed at roughly 300–600 LOC/day, with tests: #1777 pi registry cost (~400 LOC in one run), #1780 motoko/opencode cost (~600 LOC in one run), the OpenRouter routing work (~350 LOC).
- Planned at **~260 LOC/day**, with a 30% buffer for cloud-path verification, which can't be done locally.

### What already exists (from the design doc's Verification Log)
- Dispatch: `dispatchTasksCloud` (`daemon_tasks_exec.go:101`, V25), `MarkTaskQueued` (V1), `DispatchParams` (`cloud_dispatcher.go:32`, V22), batch of 5 per tick (V2).
- Job steps, each a separate function (V23): `verifyDeployKeyWrite`, push (~`coordinator_cloud_task.go:405`), `openCascadePullRequest`, `collectGitEvidence`, `writeTaskArtifacts`. Git credential: `childGitCredential` (`coordinator_cloud_executor.go:418`, V5).
- Finalization: `wantsApproval` / `wantsHandoff` (`task_finalize.go:374/388`, V4). `applyMetrics` acts only when `StageID`/`ChainID` are set (V18).
- PR lookup: `settledPRLookup` → `lookupSettledPRREST` (`daemon_landed_cards.go:50/176`), `BranchForTask` (`pr_reconcile.go:178`), `LandedPR` with no merge SHA (V21).
- Missing (confirmed negatives): any shadow or variant config (V9); a metrics tag (V17); any A/B rerun path (V13).

## Registry Reuse Audit

| Milestone | Search terms (`ailang pkg search`) | Packages inspected | Decision |
|---|---|---|---|
| M1–M7 | `shadow`, `executor-comparison`, `coordinator`, `grading` | `sunholo/relativity`, `sunholo/celestial` (both hit "shadow"; astrophysics), `sunholo/test_pkg` (hit "coordinator"; test fixture) | **none**: this is Go coordinator code inside `internal/coordinator` and `cmd/ailang`, which no registry package provides |

**AILANG syntax gate:** not applicable; no milestone writes `.ail` code. The grade job only *invokes* `ailang check`/`ailang test` on other repos' code.

## Proposed Milestones

### M1: Shadow config and task identity ✅ (~220 LOC: 140 implementation + 80 tests)
- **New file** `internal/coordinator/shadow_config.go`: `ShadowConfig{Enabled, Provider, ExecutorVariant, Model, SampleRate, MaxCostUSD, OfferAlternative}`, plus the YAML field on `AgentConfig`. **Companion file required:** `agent_registry.go` is at 771/800 lines, so only the one-line field goes there.
- Validation in `ValidateAgentProviders` (`dispatch_provider.go:123`), as the design specifies:
  - `ailang_only` only;
  - the shadow provider must differ from the primary's;
  - the variant must agree with its provider, using the same rule as `checkVariantProviderAgreement` (`internal/dispatch/cloudrun/dispatcher.go:146`), mirrored or exported;
  - `sample_rate` in [0, 1].
- Add `ShadowOf string` to `Task` (`provider.go`), `TaskRecord` (`store.go`), and the SQLite and Firestore task stores (`internal/storage/firestore/coordinator_tasks.go`), with a SQLite migration.
- **Acceptance:**
  - [x] a table test of valid and invalid configs (wrong lane, same provider, mismatched variant, out-of-range rate) returns the expected errors;
  - [x] `ShadowOf` round-trips through SQLite and Firestore (Firestore via the existing fake or emulator test pattern);
  - [x] `make test-changed` and `make check-file-sizes` pass.

### M2: Cloud dispatch of the shadow child ✅ (~240 LOC: 150 + 90)
- New `internal/coordinator/shadow_dispatch.go`. After the primary's `MarkTaskQueued` in `dispatchTasksCloud`:
  1. make a deterministic sampling decision, a hash of the task ID against `sample_rate`, recorded on the child (A1 replayability);
  2. create the child `<id>-shadow` with `ShadowOf`, **no `StageID`/`ChainID`**, the same directive, tool policy and policy path;
  3. dispatch with the shadow provider, variant and model, plus `DispatchParams.Shadow=true`.
- Register `AILANG_TASK_SHADOW` in the `internal/config` registry (`job.go`, next to `AcknowledgeOnly`, V6), plus `make docs-env`.
- **Acceptance:**
  - [x] with a fake dispatcher, an eligible task gives two dispatches (primary unchanged; shadow with `Shadow=true` and the shadow provider);
  - [x] an ineligible agent, or a sample miss, gives exactly one dispatch;
  - [x] **regression:** a non-shadow agent's dispatch parameters are byte-identical to before;
  - [x] the sampling decision is stable for a given task ID.

### M3: The job enforces no side effects ✅ (~180 LOC: 90 + 90)
- In `executeCloudTask` (`cmd/ailang/coordinator_cloud_task.go`), when `config.TaskShadow()` is set:
  - skip `verifyDeployKeyWrite`, the push step and `openCascadePullRequest`;
  - keep `collectGitEvidence` and `writeTaskArtifacts`.
- In `coordinator_cloud_executor.go`, `childGitCredential` returns no credential for shadow tasks.
- **Acceptance:**
  - [x] a unit or integration test with a fake git remote shows a shadow run produces evidence (diff, artifacts) and **no** push or PR call;
  - [x] **negative test:** with a write key present and the agent attempting `git push`, nothing reaches the remote;
  - [x] **regression:** a non-shadow run still pushes and opens a PR;
  - [x] each guard is mutation-tested (revert it and watch its test fail).

### M4: Finalization and storage ✅ (~230 LOC: 140 + 90)
- `wantsApproval` and `wantsHandoff` return false when `ShadowOf != ""`. `applyMetrics` needs no change, because shadows have no `StageID`/`ChainID` (V18); a test pins that.
- Persist the shadow result (diff, cost, tokens, turns, finish reason, session ID, artifact path, `BaseCommit`) on the child record.
- New `ShadowComparison` record keyed by the parent ID: verdict enum (`primary_better | shadow_better | equivalent | both_failed | ungraded(reason)`), `base_drift`, and per-layer grades. Stored in SQLite and Firestore.
- **Acceptance:**
  - [x] a shadow completion creates no approval card or handoff, and leaves chain aggregates unchanged (test asserts `RecomputeChainAggregates` isn't called);
  - [x] a non-shadow completion behaves exactly as before (card and handoff);
  - [x] the `ShadowComparison` CRUD round-trip works;
  - [x] `base_drift` is set when the primary's and shadow's `BaseCommit` differ.

### M5: The grade job, mechanical layer ✅ (~300 LOC: 200 + 100)
- New `cmd/ailang/coordinator_shadow_grade_job.go`, a Cloud Run Job entry on the agent executor image (**as built: a mode of the existing executor job, `AILANG_JOB_MODE=shadow_grade`; Mark 2026-10-19**), with no git credential. It:
  1. clones at each run's `BaseCommit` (`git fetch --depth 1 origin <sha>`);
  2. applies each diff;
  3. runs `ailang check` on changed modules and `ailang test --package` on changed packages **through `ailang policy-tool` under the agent's restricted policy**, each with a hard timeout (300 s default), within a 900 s job bound;
  4. reports per-diff pass/fail, or `ungraded(timeout|apply_failed)`.
- New `internal/coordinator/shadow_grade.go` dispatches the grade job when the shadow child completes and its parent has a completion, and writes the mechanical layer to `ShadowComparison`.
- **Acceptance:**
  - [x] fixture repos where diff A passes and diff B fails `ailang check` grade correctly;
  - [x] a hanging test gives `ungraded(timeout)` within the bound;
  - [x] an unappliable diff gives `ungraded(apply_failed)`;
  - [x] **no grading code runs in the daemon process** (test asserts the daemon only dispatches);
  - [x] `make` targets are never invoked.

### M6: Merged-fix comparison and reviewer ✅ (~250 LOC: 170 + 80)
- When the parent's PR settles (`settledPRLookup` on `BranchForTask(parent)`), fetch the merged diff by PR number (`pulls/{n}/files`, because `LandedPR` has no SHA, V21). Record:
  - file and function overlap for each candidate (**as built: file overlap only**; AILANG diff hunks carry no function context);
  - whether the merged PR's new or changed `.ail` tests pass on each candidate (grade job, M5 path).

  Mark the comparison `circular` when the merged PR is the primary's own.
- Reviewer: only when the layers disagree or tie. An off-Anthropic reviewer via the existing design-review machinery (reject-by-default; must name a winner, its strongest objection and a confidence), cost-capped and counted against the shadow budget.
- **Acceptance:**
  - [x] a settled-PR fixture updates the comparison;
  - [x] the circular case is flagged;
  - [x] the reviewer is invoked only on disagree or tie, and an unreachable reviewer gives `ungraded(reviewer_unavailable)`, never a silent verdict.

### M7: The report and the dev-plane enable (~200 LOC: 140 + 60, plus config)
- `ailang coordinator shadow report [--since 7d] [--json]` gives per-agent and per-harness pass rates, cost, wall time and turns; **every disagreement** with `ailang chains chat` links and both diffs; and running totals.
- A weekly scheduled run posts to the `user` inbox. Docs: `docs/docs/guides/coordinator.md` section. Changelog fragment.
- ailang-multivac `config/config.cloud.yaml` (**dev branch only**): enable on `ailang-only-executor` (motoko primary, pi shadow) and `pkg-sunholo-ailang-parse` (pi primary, motoko shadow), `sample_rate: 1.0`; the shadow key wired from Secret Manager.
- **Acceptance:**
  - [ ] the report is golden-tested on fixture comparisons, including a disagreement row with links;
  - [ ] dev-plane smoke: one real task on each enabled agent gives a primary PR plus a completed shadow child, **no** PR or branch for `-shadow`, and a `ShadowComparison` record;
  - [ ] shadow spend appears only on the shadow key.

## Day-by-Day

| Day | Milestones |
|---|---|
| 1 | M1, then M2 |
| 2 | M3 (with the negative push test) |
| 3 | M4 |
| 4 | M5 |
| 5 | M6 |
| 6 | M7, dev-plane smoke, fixes |

## Success Metrics
- All Success Criteria in the design doc are met.
- The isolation guarantees hold under test, including the negative push test and the non-shadow regression tests.
- After one dev-plane week, the first weekly report is reviewed with Mark before prod enable (design doc Rollout step 2).

## Validation (per milestone)
- `make test-changed` (touched packages plus `make test-core`); never `make test`, which CI runs.
- `make check-file-sizes` (`agent_registry.go` is at 771/800), `make check-changelog` (fragments in `changelogs/unreleased/2026-10-..-m-ailang-lane-executor-shadow-*.md`, starting with `### `), and `make docs-env` after M2.
- Mutation-test every new guard (revert it via a scratchpad copy, never `git checkout <file>`).

## Risks
- **Cloud-only paths can't be fully exercised locally.** Mitigation: a fake dispatcher, fake git remote and fake stores for M2–M4; M7's dev-plane smoke is the real end-to-end check.
- **Firestore schema change.** Mitigation: additive fields only; existing documents read `ShadowOf == ""`.
- **The grade job image needs the target repos' AILANG dependencies.** Mitigation: the same `ailang install` and dependency resolution the agent image does; an apply or resolve failure is `ungraded`, never a harness failure.

## Dependencies / Open Questions
- Provision `OPENROUTER_API_KEY_SHADOW` (Secret Manager, ailang-multivac) with its monthly cap. **This needs Mark** (billing).
- Prod enable is a separate decision after the first weekly report.
