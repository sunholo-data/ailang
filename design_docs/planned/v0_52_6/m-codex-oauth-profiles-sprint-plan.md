# M-CODEX-OAUTH-PROFILES sprint

Status: Completed, 2026-10-09. Design and execution approved by Mark. The parent document retains its separate planned provider migration.

Design: [operational addendum](m-codex-subscription-lane.md#operational-addendum-three-oauth-profiles-on-one-account-2026-10-09).

Four days estimated, 2500 LOC including tests, medium/high integration risk. Recent executor auth fixes and Phase 1 landed the same day; daemon compatibility and cloud recovery dominate uncertainty, so estimates use a four-day buffer rather than extrapolating merge volume.

## Execution waves

M1 provisioning, M2 local ownership, M3 cloud ownership are independent and execute in parallel; M4 integrates all three. Parent owns artifacts, release evidence and rollout. Agents share the isolated tree with disjoint file ownership; no branch switching or unrelated changes.

## M1: Provision three independent OAuth authorizations on one account

Estimated 500 LOC; dependencies: none.

- [x] Checker refuses credential copies, account/workspace mismatch, unsafe paths and permissions without exposing secrets.
- [x] Provisioning creates independent file-backed homes and never copies interactive credentials.

## M2: Local mission home and one daemon refresh owner

Estimated 900 LOC; dependencies: none.

- [x] Mission controller, probes, quota reads and nested executors resolve the isolated mission home.
- [x] Measured daemon adapter uses one auth owner for concurrent local runs; unsupported CLI exec cannot silently bypass it.

## M3: Cloud exclusive credential ownership

Estimated 850 LOC; dependencies: none.

- [x] Transactionally acquire canonical credential before fetch and keep ownership through preflight/run/persist.
- [x] Cancellation/lease loss stops execution; failed persistence quarantines credential; no unsafe TTL takeover.

## M4: Integration, independent evaluation and rollout

Estimated 250 LOC; dependencies: M1, M2, M3.

- [x] Focused Go, shell, boundary and lint checks pass; independent evaluator reviews criteria.
- [x] Two user-completed logins validate same account and independent credentials; cloud and local smoke tests pass before activation.

## Files and checks

M1: tools/attended profile check/provision/test scripts and operational runbook. M2: tools/launchd driver/helpers and internal/mission quota plus explicit daemon execution adapter (CLI exec remote convergence is a measured prerequisite). M3: cmd/ailang cloud auth/lifecycle and internal/storage/firestore ownership adapter/test files. M4: changelog fragment, Dockerfile Codex pin if required, focused Go/shell checks, make test/lint/check-boundaries and independent evaluator report.

Day 1: failing regression tests and runtime spike. Day 2: local/provision implementation. Day 3: cloud ownership and failure recovery. Day 4: integration, evaluation, provisioning and smoke rollout.

Registry reuse: searched oauth and lease, inspected sunholo/oauth. None: Codex CLI owns its wire login/refresh, and Firestore host integration belongs in existing Go infrastructure. No new AILANG language feature; .ail showcase/contracts/effects/inline tests skip because all changes are host operational code.

## Rollout boundary

Do not activate incomplete ownership paths. Interactive home remains untouched. User completes new mission/cloud OAuth authorizations. No D8/provider migration or silent metered fallback. Never equate tests passing with live rollout; record pending OAuth/deployment evidence explicitly.

## Execution evidence (2026-10-09)

[PR #1752](https://github.com/sunholo-data/ailang/pull/1752) merged as
`9d9190082f02885c9426bc49240b9b312716f8d3`; its required checks and the
[post-merge CI](https://github.com/sunholo-data/ailang/actions/runs/37957167093)
passed. [v0.53.1](https://github.com/sunholo-data/ailang/releases/tag/v0.53.1)
published all platform binaries and signed artifacts. Full local tests passed
with `GOFLAGS=-p=4` after an unrelated OpenCode startup fixture failed under
CPU load; three isolated fixture reruns passed. Focused/race, Bash 3.2,
lint, boundaries, size and CLI gates passed. Sonar's external reporting quality
result remains separate from required CI; no finding was suppressed.

Two attended device logins and the complete profile checker established one
account/workspace with independent authorizations. Interactive sessions continued
through the rollout. Local missions use the dedicated home and one minimal
launchd owner. Six idle profiles (docs, fleet, motoko, stapledon, v1, world)
were switched without dispatching business missions. Their temporary feature
pin was restored to the normal `origin/dev` driver reference after merge;
configuration-only verification confirmed that it includes the fix. The tagged
binary's local inference returned `MISSION_AUTH_OK`, including a repeat after
cloud rotation. The native nine-client fixture observed exactly one refresh.

Cloud release build `7bd2ac92-17e8-45ba-8151-26abd580bdcb` succeeded: all 16
images, test deployment and health/version gates. Executor-family dry run
`8a188523-5b8e-4daa-987d-7a20a4091c89` and production promotion
`73792d9b-af25-48ec-a463-b03405d068e2` succeeded, promoting 12 executor images
and pinning all 15 jobs. The all-job secret audit found exactly codex and
codex-go. New digest pins and terminal old executions were checked before
publishing under a reviewed attended canonical credential reservation.

Publication advanced the enabled cloud secret from version 1 to 2. A reviewed,
RPC-only native `account/read` forced refresh used the actual cloud
restore/finalize path: confirmed process termination, durable version 2 to 3,
then lease release. A temporary Go overlay exercised the released source;
it was not committed into the production binary. The published staging seed
was retired from its active `auth.json`; latest in Secret Manager is authoritative.

| Cloud execution | Input / output tokens | Tools / changed files | Assistant reply |
| --- | ---: | ---: | --- |
| `ailang-agent-executor-codex-hf2xc` | 12166 / 8 | 0 / 0 | `AUTH_SMOKE_OK` |
| `ailang-agent-executor-codex-go-k8plw` | 12170 / 8 | 0 / 0 | `AUTH_SMOKE_OK` |

Both normal executions restored the saved credential and completed successfully.
Execution-only REST overrides cleared GitHub and API credentials; metadata
confirmed no secret reference in those overrides and unchanged persistent job
configuration. The CLI's local secret-to-literal validation rejected the same
request before execution, so the documented REST API was used after
`validateOnly` acceptance. Completion logs confirmed subscription auth, no
commits, no branch/PR creation and files=0. Final inspection found every retained
Codex execution terminal and the canonical lease unowned and not quarantined;
enabled version remained 3. Required release reindex populated 67 syntax,
390 builtin and 247 example chunks using the active prompt version.

Independent sprint evaluation passed at **97/100**. The operational addendum is
complete; the parent D8 migration and issue #903's broader provider work remain
outside this completion.
