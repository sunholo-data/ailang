# M-EXECUTOR-CREDENTIAL-REPOS Sprint Plan

**Design:** [M-EXECUTOR-CREDENTIAL-REPOS](m-executor-credential-repos.md)
**Target:** v0.52.6 | **Duration:** 4 engineering days (approximately 28 hours) | **Risk:** medium
**Status:** Planning complete; execution awaits coordinator sprint-plan approval.

Add explicit trusted-registry read grants for extra GitHub repositories, backed by a separate read-scoped token. Preserve the task repository credential and the zero-grant environment. Deliver four sequential milestones, approximately 1,080 LOC including implementation, tests and documentation.

## Authority and scope

The coordinator handoff says “Previous work has been approved.” This plan takes that as approval of the proposed D1/D2/D6/D7 design: trusted AgentConfig fields, separate read-scoped token, complementary shipping with task-inputs, and H-6 remaining open under UID-split. The design's unchecked freeze boxes are historical approval metadata; this plan records the later approval without changing the source artifact. Token type and provisioning remain deployment-owner decisions at staging; neither type requires different implementation of the name-based secret fetch. Sprint-plan approval through the coordinator remains the execution gate; this planning task does not start implementation.

Cloud/shell lanes only. No language or motoko core change, no local-lane grants, no task-input schema changes. Task-inputs remains the stronger parent-fetch path when inputs are enumerable. Read-only is enforced by token permissions; URL-keyed git config is advisory to a same-UID child. The existing fleet-token file remains readable by that child: H-6 is not closed.

## Current implementation and velocity

Reviewed GitCredentialScopes, WriteGitCredentialFile, GitCredentialEnv, BuildEnvironment, trusted registry fields, daemon dispatch copy site, DispatchParams and cloud override emitter. The current job creates one task-repo credential file. GitCredentialEnv indexes from zero, so separate emissions collide. No proposed credential-repo fields exist. The existing SSH secret fetch and environment override paths are the reuse points.

Ran analyze_velocity.sh for seven days. The checkout exposes one recent snapshot commit; its root diff reports 26,391 files and 2,780,112 insertions, so it cannot measure engineering velocity. The script reports no usable LOC metrics and scans historical changelog entries. Current changelog confirms v0.52.5 shipped October 7, including serve-api record binding fixes, but gives no comparable elapsed-time measures. Consequently 270 LOC/day is a planning capacity assumption, not observed throughput. Four days retain the design's conservative 3–4 day estimate and include approximately six hours of contingency within 28 hours. External staging lead time is excluded.

## Registry reuse audit

Executed `ailang pkg search credentials` and `ailang pkg search git`. Git returned no packages. Credentials returned sunholo/linkedin@0.5.1 and sunholo/economic@0.1.0; inspected both with `pkg info` and attempted `pkg docs`: economic supplied documentation; linkedin has no AGENT.md. These are AILANG service clients rather than Go harness git/Secret Manager authority components. The CLI reported a potentially stale binary; these results are discovery evidence, while the Go execution boundary independently excludes package substitution.

| Milestone | Decision | Reuse rationale |
|---|---|---|
| M1 | none | Extend existing internal/executor emitter and writer; no applicable registry git package. |
| M2 | none | Extend trusted Go registry/dispatch/config seam; service clients cannot own dispatch authority. |
| M3 | none | Reuse existing Go Secret Manager access pattern and credential writer; no new package needed. |
| M4 | none | Reuse existing Go tests and documentation; no package capability to introduce. |

## Milestones

### M1: Merge executor credential bindings (~220 LOC)

**Effort:** 5 hours, day 1. **Dependencies:** none

**Files:** internal/executor/gitcred.go, executor.go, environment.go; gitcred_test.go and environment_test.go.

**Acceptance criteria:**

- [ ] Single-binding and zero-extra output are byte-identical to pinned current GitCredentialEnv fixtures.
- [ ] Two bindings have contiguous indexes and correct GIT_CONFIG_COUNT; empty bindings and quoted paths work.
- [ ] BuildEnvironment wires both files without permitting GIT_CONFIG_* through ExtraEnv.

### M2: Validate trusted grants and transport dispatch metadata (~360 LOC)

**Effort:** 7 hours, day 2. **Dependencies:** M1

**Files:** internal/coordinator/agent_registry.go, cloud_dispatcher.go, daemon_tasks_exec.go; internal/dispatch/cloudrun/dispatcher.go; internal/config/executor.go; cmd/ailang/coordinator_agents_list.go and focused tests.

Normalize task repository coordinates from HTTPS and SSH workspace representations before overlap comparison. Reject duplicates rather than silently deduplicating, following the design success criteria. Validate before entering the cloud dispatcher and explicitly mark permanent task failure; wrapping an error outside the existing handler is insufficient. Test that no job is launched and the task is not retried.

**Acceptance criteria:**

- [ ] Only AgentConfig supplies grants; message/task content cannot override them.
- [ ] Invalid format, wildcard, duplicate, cap greater than eight, non-cloneable shape and normalized task-repo overlap fail permanently before cloud dispatch, naming agent, repo and rule.
- [ ] Nonempty grants require a secret name; newline serialization and config readers round-trip validated grants.
- [ ] Cloud overrides carry only repo names and secret name; zero grants preserve existing behavior.

### M3: Fetch separate read token and manage both credential files (~310 LOC)

**Effort:** 8 hours, day 3. **Dependencies:** M1, M2

**Files:** cmd/ailang/coordinator_cloud_executor.go and coordinator_cloud_gitcred_test.go; reuse coordinator_cloud_sshkey.go secret-access pattern.

Do not retain the current early return for absent fleet token or non-HTTPS task scopes if it would suppress valid extra grants. Validate configuration defensively at the job boundary. Introduce an injectable secret-fetch seam for offline failure tests; clean up the first file if second-file creation fails. Never copy secret material into dispatch metadata.

**Acceptance criteria:**

- [ ] Job revalidates grants and fetches the named secret with its service account; extra scopes use only the read-token file.
- [ ] Missing, inaccessible or whitespace-only secret fails loudly without fleet-token fallback or token logging.
- [ ] None mode writes neither file and fetches no secret; unknown mode preserves named error.
- [ ] Cleanup removes both files on success and partial failure; permissions remain directory 0700 and file 0600.
- [ ] Extra grants remain available independently of an absent task credential; PI_WORKSPACE_TRUST_REMOTES is not widened.

### M4: Verify scope routing and document staging (~190 LOC)

**Effort:** 8 hours, day 4. **Dependencies:** M2, M3

**Files:** internal/executor/gitcred_test.go; cloud dispatcher and job tests; docs/docs/guides/agent-tool-policy.md; docs/internal/cloud-coordinator-config.md; cmd/ailang/help.go where env/help tables live; design_docs/planned/v0_49_1/m-executor-uid-split.md; examples/config/credential-repos.yaml (new).

Use git credential fill with dummy secrets and isolated HOME/config to test HTTPS helper routing offline. A local bare remote cannot prove GitHub token permissions. A mocked HTTP 403 establishes credential selection/error propagation only; actual contents:read and denied push require the staging check. The YAML example supplies task workspace rdasouthwestgroup, extra daneel-memory, and a secret-name placeholder, never a token. No .ail files are planned, so the AILANG syntax gate is inapplicable.

**Acceptance criteria:**

- [ ] Offline git credential fill checks task, granted and ungranted URLs with and without .git; extra URLs return only the read credential.
- [ ] Mocked server refuses extra-repo push and records the read credential; documentation distinguishes this from real GitHub permission verification.
- [ ] Guide, CLI/env documentation, YAML example and UID-split H-6 incident cross-reference are updated.
- [ ] Make test, make lint and make check-boundaries pass; sprint-evaluator assesses the approved design.
- [ ] Daneel staging checklist records token permissions, Secret Manager IAM, private incoming ref clone, denied push and site PR evidence, or explicitly records staging blocked without claiming rollout success.

## Daily work and validation

Day 1: pin current single-binding output, implement merged emitter and Task seam, run executor credential/environment tests. Day 2: validate registry input and permanent-failure flow, wire dispatch overrides and config readers, run coordinator/cloudrun/config tests. Day 3: add separate secret fetch/file lifecycle, run job mode/grant matrix and partial-failure tests. Day 4: exercise real git helper routing offline, add mocked denial test and YAML documentation example, run final checks, prepare staging record and evaluator handoff.

Run `go test ./internal/executor ./internal/coordinator ./internal/dispatch/cloudrun ./internal/config ./cmd/ailang` during relevant milestones. At completion run `make test`, `make lint`, `make check-boundaries`; use gofmt on changed Go files. Measure coverage of new validation and error branches with focused `go test -cover`; every specified refusal, mode and cleanup branch must have a behavioral assertion. No numerical repository coverage baseline was measured during planning and none is claimed. Parse the YAML example through the actual AgentConfig loader and validator in a focused test rather than claiming it is a deployed entry. Finish with sprint-evaluator against design criteria.

## Staging dependencies and completion accounting

Deployment owner supplies a fine-grained PAT or App installation token restricted to Contents:read on exactly sunholo-data/daneel-memory, Secret Manager secret, and job-service-account accessor IAM. No token minting/renewal subsystem is part of this sprint. Confirm token expiry/rotation owner before enabling the entry. Stage site-rda in Daneel's actual trusted registry (outside this repository), then clone a private incoming/<ref>, confirm push denied with the read token, and record the resulting site PR. Do not perform a destructive push probe against production content; use a disposable test ref and a genuinely read-only token.

Implementation can be evaluated with offline checks if staging credentials are unavailable; the live rollout criterion remains pending and must be reported separately. Do not mark the design's end-to-end acceptance complete without evidence. Cross-repository Daneel issue numbers #335/#339/#344 are contextual references, not AILANG issues to auto-close; AILANG #1600 belongs to task-inputs and is not fixed by this sprint.

## Risks and mitigations

Read secret may be provisioned with write permissions: deployment owner verifies permissions and live denial; the job cannot infer token permission from a string. Scope overlap or malformed coordinates may shadow task credentials: normalize and reject before dispatch, test .git variants and SSH task coordinates. Same-UID credential extraction persists: record the measured October 8 incident in the UID-split evidence table and keep H-6 open. Secret fetch or second file write may leak a first temp file: test partial-failure cleanup. New env vars must be registered in existing CLI/env documentation, following cli-doc-maintainer during execution.

## Executor handoff

Artifacts are ready for coordinator review. Approval/merge of this sprint-plan task triggers sprint-executor through the configured coordinator workflow; no duplicate manual message or implementation run is sent from planning. Executor loads this plan and JSON, preserves milestone dependencies, records staging limitations, then hands implementation to sprint-evaluator.
