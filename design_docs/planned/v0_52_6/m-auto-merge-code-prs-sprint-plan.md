# Sprint Plan: M-AUTO-MERGE-CODE-PRS

Refs #1599 (sunholo-data/ailang)

**Design:** [Approved design](m-auto-merge-code-prs.md)
**Status:** Planned; design approved by handoff, sprint awaits coordinator approval.
**Target:** v0.52.6
**Duration:** 5 working days (32 engineering hours plus 8 hours contingency).
**Risk:** High at the GitHub authorization boundary; implementation is isolated from production enablement.

## Summary

Add trusted per-agent code auto-merge with declared-path bounds, explicit check names, a verified non-author approver, and PR audit. GitHub native auto-merge waits for repository requirements. All agents without the code opt-in retain the markdown floor.

## Current status and evidence

- Read issue #1599 and its single maintainer ruling comment via GitHub REST on 2026-10-08. The ruling permits code auto-merge with passing required checks and non-author approval. No new issue is needed.
- Reviewed the approved design and current `cmd/ailang/coordinator_cloud_github.go`: the markdown floor and shared declared-pattern guard are still present. The initial report names an internal path; the actual wrapper is under `cmd/ailang/`.
- Registry → daemon dispatch → Cloud Run env → job config is the existing authority path. Reuse the job-side Secret Manager fetch pattern and native GraphQL enable helper.
- Existing scope tests implement a separate `matchAll` helper. Add tests of production decisions so a copied matcher cannot conceal a regression.
- Working tree started clean on `coordinator/task-53d1dbd7`; design artifact from task-88edd78d is present. Keep work on this branch.

## Velocity and capacity

The skill velocity script ran for seven days. This checkout is shallow and contains one visible commit (`86195621`, 2026-10-08); its parent is unavailable, so diff-based throughput is unmeasurable. The script also picked up historical changelog LOC entries, which are not recent velocity evidence. Do not use those to claim measured daily throughput.

Estimate from concrete file changes: 480 implementation LOC, 400 test LOC, 70 documentation/example LOC = **950 LOC total**. Planning capacity is **190 LOC/day**, an assumption, not observed velocity. Five days adds 25% calendar buffer to the design’s four-day estimate. Credential/ruleset provisioning time is excluded and may delay deployment, not repository implementation.

## Registry reuse audit

`ailang pkg search github` returned no packages. `ailang pkg search coordinator` returned only `sunholo/test_pkg@0.1.3`; `pkg info` and `pkg docs` show pure greeting/version exports for integration tests and explicitly exclude production use. The CLI warned that its binary may be stale; these remote catalog results are advisory and no package is installed.

All four milestones use action **none**: M1 is Go registry/dispatch infrastructure, M2 is the existing Go wrapper’s shared guard and REST check query, M3 is job-side secret/REST approval plus audit, and M4 is tests and operational documentation. The candidate cannot implement any of these and introducing an AILANG package into the Go control plane would cross the wrong boundary. Reuse existing repository helpers rather than a new package.

## Technical sequencing and design clarifications

1. Resolve docs/code mode from trusted config only. Validate mandatory code-mode fields and declared scope.
2. Resolve base HEAD and enumerate check-runs, including pagination. Existence is not proof that a check is required; repository protection remains the deployment precondition.
3. Fetch the second-identity secret and verify actual login, declared login, and PR author before enabling auto-merge. This implements the design’s stated refusal guarantee even though its example helper bundles validation with review posting.
4. Record intent/audit, enable native auto-merge, then post approval after the final push. Do not claim enabled-and-approved until review success. On a later approval failure, attempt to disable auto-merge and report any cleanup failure; repository protection remains the ultimate guard.
5. Preserve audit when PR creation reuses an existing PR. A creation-time body note says configured/requested until success is known; best-effort label failures remain visible.

The original issue suggests approving after checks pass; the approved design delegates waiting to native auto-merge and posts approval after the final push. Follow the approved design, with required checks enforced by the target ruleset. No new polling is planned.

## Proposed milestones

### M1: Trusted configuration and dispatch plumbing (~240 LOC)

**Estimated:** 130 implementation + 110 tests + 0 docs/example LOC.
**Duration:** 1 day / 8 hours
**Dependencies:** None
**Files and examples:** `internal/coordinator/agent_registry.go`, `cloud_dispatcher.go`, `daemon_tasks_exec.go`; `internal/dispatch/cloudrun/dispatcher.go`; `internal/config/job.go`; registry/config/dispatcher tests.

**Tasks and acceptance criteria:**

- [ ] Four optional AgentConfig fields round-trip through YAML/JSON; absent values retain docs-only behavior.
- [ ] Only trusted registry fields supply dispatch parameters and env; newline-separated checks preserve names containing commas.
- [ ] Code mode requires auto_merge, non-empty check names, approver secret name, expected identity, and declared patterns; invalid configuration refuses loudly.
- [ ] Secret material is fetched only inside the job and never appears in dispatch specs, PR bodies, or logs.

**Risk and mitigation:** Authority leakage through message parameters: explicitly test registry-derived values and zero defaults.

### M2: Unified scope and required-check gates (~250 LOC)

**Estimated:** 130 implementation + 120 tests + 0 docs/example LOC.
**Duration:** 1.25 days / 10 hours
**Dependencies:** M1
**Files and examples:** `cmd/ailang/coordinator_cloud_github.go`, `coordinator_cloud_automerge_test.go`, `coordinator_cloud_github_test.go` (extend or create helper-focused tests as needed).

**Tasks and acceptance criteria:**

- [ ] Existing docs-mode acceptance and refusal semantics remain intact; HTML/images are eligible only in opted-in code mode and declared scope.
- [ ] Production guard tests cover empty diff, empty patterns, out-of-scope changes, git errors, and code-mode opt-in; tests do not merely duplicate the matcher.
- [ ] Named checks are resolved on base HEAD with pagination; missing names, empty sets, malformed responses and API failures refuse before enable.
- [ ] No direct merge API or polling loop is introduced; existing native SQUASH auto-merge remains the merge actor.

**Risk and mitigation:** Check drift or incomplete pagination: fail closed on unavailable/missing evidence and test multi-page results.

### M3: Non-author approval and durable audit (~310 LOC)

**Estimated:** 200 implementation + 110 tests + 0 docs/example LOC.
**Duration:** 1.5 days / 12 hours
**Dependencies:** M1, M2
**Files and examples:** `cmd/ailang/coordinator_cloud_github.go`, `coordinator_cloud_github_test.go`; reuse `coordinator_cloud_sshkey.go` secret fetch without broad auth refactoring.

**Tasks and acceptance criteria:**

- [ ] Approver secret fetch and actual-login/expected-login/PR-author checks finish before enable; identity comparison is case-insensitive.
- [ ] Approval is posted after final push and native enable; wrong identity, author collision and secret/API errors produce actionable refusal without tokens.
- [ ] Approval failure after enable cannot be reported as success; disable native auto-merge on failure where possible and surface cleanup failure explicitly.
- [ ] PR body records configured intent separately from successful enable/approval; code PRs carry auto-merge-code label, check names, scope, identity and Refs #1599.
- [ ] API tests assert request ordering, successful approval, failure paths and retried existing-PR audit updates; label failure is loud and body/log audit persists.

**Risk and mitigation:** Enable/review ordering and partial failure: preflight identity, test ordered calls and rollback diagnostics; never expose tokens.

### M4: Regression validation and Daneel deployment runbook (~150 LOC)

**Estimated:** 20 implementation + 60 tests + 70 docs/example LOC.
**Duration:** 0.25 day / 2 hours + 1 day shared contingency
**Dependencies:** M2, M3
**Files and examples:** `docs/docs/guides/coordinator.md`; wrapper and dispatch regression tests. Configuration examples live in this guide; no .ail program is needed for a Go deployment capability.

**Tasks and acceptance criteria:**

- [ ] Coordinator guide includes opt-in and docs-only YAML examples, required-check semantics, secret permissions, refusal troubleshooting and audit enumeration.
- [ ] Runbook distinguishes check existence from required-by-ruleset enforcement and requires Mark confirmation before production opt-in.
- [ ] Relevant package tests, make test, make lint and make check-boundaries pass; record command results and address regressions.
- [ ] Staging checklist proves failing required checks block merge and successful checks plus non-author review permit merge; deployment evidence is recorded or explicitly pending.
- [ ] Daneel rollout requires one staging merge and ten clean production merges; no live registry/ruleset/secret changes are made as part of this repo sprint.

**Risk and mitigation:** Environment-only proof is unavailable without credentials: distinguish repository validation from pending deployment evidence.

## Day-by-day work

| Day | Work | Outcome |
|---|---|---|
| 1 | M1 plumbing, config validation and dispatch tests | Trusted fields reach the wrapper; defaults verified |
| 2 | M2 shared guard, base check enumeration and negative tests | Scope/check evidence gates enabling |
| 3 | Finish M2; M3 secret and identity preflight, enable/review sequence | Non-author review path with loud failures |
| 4 | Finish M3 audit and existing-PR retries; M4 guide/runbook and broad checks | Reviewable implementation and operational examples |
| 5 | Contingency for API/test failures; evaluator evidence and staging if provisioned | Repository acceptance verified; deployment status explicit |

## Validation and success metrics

Use table tests and httptest/fake transport for decision and side-effect paths; keep dependency injection narrow. Cover every listed refusal and the successful enable/approve sequence. Target at least 90% statement coverage on new pure parsing/decision helpers, plus branch-oriented API tests; do not claim a repository coverage baseline because it was not measured during planning.

Run focused `go test ./internal/config ./internal/coordinator ./internal/dispatch/cloudrun ./cmd/ailang` while implementing. At completion run `make test`, `make lint`, and `make check-boundaries`; use gofmt for touched Go files. Existing docs-only fixtures must retain their behavior. Verify both guide YAML examples against the real config loader in tests.

Staging evidence must include an in-scope HTML/image PR, an out-of-scope refusal, a failing required check that leaves the PR unmerged, and a successful merge after required checks and non-author review. If credentials are unavailable, record this as pending rather than passed. Production activation remains pending until Mark confirms protection and the rollout gate.

## Deployment prerequisites and scope

- Mark confirms target repo PR/review protections, last-push approval, blocked force-push/deletion and restricted bypass. Confirm all configured checks are actually required, not merely present on base HEAD.
- Provision a repo-limited second-user PAT secret and job service-account access; ensure expected login differs from fleet PR author. PAT is v1; GitHub App token support is deferred.
- Ensure native auto-merge is enabled for the target repo, the check workflow already runs on base and PRs, and audit labels can be applied.
- Daneel-side workflow/registry activation and ten-clean-merges observation are deployment work outside this repository implementation. Do not weaken protections or grant bypass.

## Coordinator handoff and PR body

Keep sprint status `not_started` and every milestone `passes: null`. Coordinator review/merge of this plan is the execution gate; completion markers provide the automatic artifact handoff. Do not send a duplicate executor task or implement code during planning.

Use this PR body when the coordinator opens the plan PR:

> Refs #1599
>
> Plan the approved per-agent code auto-merge feature in four dependent milestones across five working days (~950 LOC). Covers trusted config plumbing, scope/check gates, non-author approval and audit, and regression tests plus the Daneel runbook. Production opt-in remains gated on target ruleset/credential confirmation.
>
> Validation: populated sprint JSON checked for milestone dependencies, LOC totals, issue linkage and reuse decisions; documentation-only planning change.

**Progress artifact:** `.ailang/state/sprints/sprint_M-AUTO-MERGE-CODE-PRS.json`
**Approval:** Design approved in task-88edd78d handoff; this sprint is prepared for coordinator review, not self-approved.
