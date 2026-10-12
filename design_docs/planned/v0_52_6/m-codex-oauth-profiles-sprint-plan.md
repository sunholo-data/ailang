# M-CODEX-OAUTH-PROFILES sprint

Status: In progress. Design and execution approved by Mark, 2026-10-09.

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
- [ ] Two user-completed logins validate same account and independent credentials; cloud and local smoke tests pass before activation.

## Files and checks

M1: tools/attended profile check/provision/test scripts and operational runbook. M2: tools/launchd driver/helpers and internal/mission quota plus explicit daemon execution adapter (CLI exec remote convergence is a measured prerequisite). M3: cmd/ailang cloud auth/lifecycle and internal/storage/firestore ownership adapter/test files. M4: changelog fragment, Dockerfile Codex pin if required, focused Go/shell checks, make test/lint/check-boundaries and independent evaluator report.

Day 1: failing regression tests and runtime spike. Day 2: local/provision implementation. Day 3: cloud ownership and failure recovery. Day 4: integration, evaluation, provisioning and smoke rollout.

Registry reuse: searched oauth and lease, inspected sunholo/oauth. None: Codex CLI owns its wire login/refresh, and Firestore host integration belongs in existing Go infrastructure. No new AILANG language feature; .ail showcase/contracts/effects/inline tests skip because all changes are host operational code.

## Rollout boundary

Do not activate incomplete ownership paths. Interactive home remains untouched. User completes new mission/cloud OAuth authorizations. No D8/provider migration or silent metered fallback. Never equate tests passing with live rollout; record pending OAuth/deployment evidence explicitly.

## Execution evidence (2026-10-09)

M1–M3 source and offline acceptance checks passed. A real Codex 0.162.0 daemon served eight concurrent quota clients plus a later read with exactly one fake-token refresh. Mission OAuth was independently authorized and a live daemon inference returned `MISSION_AUTH_OK`. The mission owner was restarted with a minimal environment. Final integrated gates, cloud authorization/profile comparison, deployment and cloud smoke remain pending; production fleets have not been switched.

Final full regression/race/driver suites and lint/boundary/size/CLI/changelog gates passed. Both device logins and the complete profile checker passed. The tested binary and six idle local mission profiles were activated with a reviewed source pin and rollback copies; the installed driver pin was verified without running a mission. Cloud IAM passed read-only checks. PR #1752 contains the fix. Cloud CI/merge/versioned deployment, secret publication and smoke remain pending.
