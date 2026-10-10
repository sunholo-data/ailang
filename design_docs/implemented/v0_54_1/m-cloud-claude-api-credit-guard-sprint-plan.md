# Sprint Plan: M-CLOUD-CLAUDE-API-CREDIT-GUARD

## Summary

Enable Haiku 5.5 in cloud executors through an authenticated Anthropic budget gateway,
with shared transactional request reservations and an operator credit-renewal command.

**Design:** [Approved design](m-cloud-claude-api-credit-guard.md)
**Status:** Completed — M0–M6 passed; repaired dev/prod canaries reconciled, audited promoted and production evaluator active. Independent evaluation PASS94/100.
**Duration:** 9 engineering days; infrastructure/credit access may add elapsed waiting time
**Estimated size:** 2,610 LOC including tests and retained operational evidence
**Risk:** High — correctness of request-cost bounds and failure accounting determines the spending guarantee
**Progress:** `.ailang/state/sprints/sprint_M-CLOUD-CLAUDE-API-CREDIT-GUARD.json`

User approval on 2026-10-09 covers the Claude-credit design and confirmation interface.
Generalizing to other credit pools is explicitly a follow-up. The user authorized execution with “execute sprint” on 2026-10-10. Local implementation
and local validation have completed, including the installed-CLI completion and tool-loop contract. Live rollout has not started. This plan does not replace or activate the unrelated existing sprint.

## Planning baseline and estimated velocity

Existing Claude API-key authentication, Cloud Run variants, event-driven task limits,
model registry, and Firestore transaction wrapper are reusable. The grant ledger,
request-level gateway guard, and `coordinator credits confirm/status` remain new work.

Seven-day source history in coordinator, Cloud Run dispatch, secrets, Firestore, and
Claude executor contains 656 insertions and 46 deletions across 16 file changes in five
commits dated 2026-10-08/09. Relevant commits include `07c7599bb` (cloud Claude setup-token
auth), `ba8245197` (cascade dispatch), and `54e6c9d4f` (per-agent code auto-merge).
This is aggregate fleet output concentrated on two dates, not measured single-agent
engineering velocity. The velocity script exited 141 in its `head` pipeline; historical
changelog samples are not used as recent velocity evidence.

Budget 300 LOC/day including tests: 2,610 / 300 = 8.7 days, rounded to nine. This refines
the design's initial 5–7 day estimate by explicitly budgeting emulator, failure-injection,
CLI, and operational validation. Re-estimate after M0 if gateway compatibility or billing
bounds change scope; seek a design amendment if a guard cannot be proved.

## Registry Reuse Audit

Read-only `ailang pkg search` queries: `credits budget quota`, `anthropic gateway`,
`firestore transactions`, then individual `budget`, `credits`, `gateway`, `anthropic`,
and `firestore`. Initial sandbox DNS failed; searches succeeded with network-enabled
execution. Inspected both `pkg info` and `pkg docs` for:

- `sunholo/firestore@0.7.3`: AILANG REST CRUD/query client.
- `sunholo/billing_store@0.9.3`: AILANG billing-domain CRUD repositories.
- `sunholo/gemini_live@0.5.0`: Gemini voice/protocol helpers; budget keyword match.
- `sunholo/decisions@0.4.0`: probabilistic decision-model bindings; budget keyword match.

Reuse the host's existing Go Firestore client and `RunTransaction` wrapper
(`internal/storage/firestore/client.go`), transaction patterns such as
`coordinator_status_cas.go`, and existing Claude/Cloud Run/modelreg modules. The registry
candidates are application-level AILANG modules; this sprint implements the existing Go
cloud execution boundary and requires deterministic transactional admission. No runtime
package dependency or contribution to those candidates is warranted for this scope.

| Milestone | Registry decision | Reason / in-tree reuse |
|---|---|---|
| M0 | none | Operational inventory and request-contract evidence, not a new package |
| M1 | none | Host Go credit contract and transactions; reuse existing Firestore client |
| M2 | none | Extend canonical Go model pricing, not Gemini or decision bindings |
| M3 | none | Narrow Anthropic Messages gateway; no gateway/Anthropic registry match |
| M4 | none | Extend existing executor, environment policy, and Cloud Run dispatch |
| M5 | none | Extend coordinator CLI and shared credit store; billing CRUD is a different boundary |
| M6 | none | Integrate/test existing host modules and document deployment |

## Milestones

### ✅ M0: Deployment and request-contract evidence (~80 LOC)

**Dependencies:** None
**Estimate:** 80 LOC retained documentation; 0.5 day
**Ownership:** Operational evidence and supported-request contract only
**Example/artifact:** `design_docs/verification/cloud-claude-api-credit-guard/deployment-contract.md`

Inventory actual dev/prod jobs, image/CLI versions, registry entries, infrastructure
checkout, secret references, and identities without reading keys into logs. Record
linked organization/workspace, claimed grant amount/expiry, other grant consumption,
and a defensible exclusive allocation. Establish the maximum billable input/output,
cache refresh/TTL behavior, supported beta/features, and hidden CLI inference models.
Identify the authenticated gateway connection and capability-issuance path.

**Acceptance criteria:**

- [x] Deployment evidence names exact cloud projects/jobs/images and external infrastructure ownership.
- [x] Supported request types, pricing bounds, helper models, and credential transport are verified or explicitly blocked.
- [x] Actual grant and allocation evidence are recorded without secret material; unavailable evidence blocks activation.
- [x] No live inference is required before the reserved smoke path exists.

### ✅ M1: Shared grant and request authority (~700 LOC)

**Dependencies:** M0
**Estimate:** 450 implementation + 250 tests; 2 days
**Ownership:** New focused credit-domain package and Firestore adapter
**Files:** proposed `internal/creditbudget/`, `internal/storage/firestore/credit_accounts.go`, associated tests
**Example/artifact:** Emulator scenario with two requests competing for the last allowance

Implement integer micro-USD accounts, grants, attempts, concurrency leases, and requests.
Reuse `Client.RunTransaction`. Admission reserves account/task/day exposure atomically;
settlement is idempotent. Persist intent before sending, preserve unknown exposure,
and retain ambiguous old-grant costs through renewal. Expose operator-authorized grant
confirmation primitives distinct from task approval.

**Acceptance criteria:**

- [x] Parallel emulator clients cannot exceed account/task/day ceilings or the two-task pilot limit.
- [x] Same-cycle reconfirmation adds zero allowance; key rotation/restarts/date changes do not replenish it.
- [x] Duplicate settlement is harmless; lease expiry never refunds forwarded or unknown exposure.
- [x] Expiry, kill switch, storage failure, and unresolved grant transitions prevent new admission.
- [x] Relevant tests and formatting/lint pass.

### ✅ M2: Canonical request pricing (~320 LOC)

**Dependencies:** M0
**Estimate:** 200 implementation + 120 tests; 1 day
**Ownership:** `internal/modelreg/models.go`, `models.yml`, canonical cost helper and consumers
**Example/artifact:** Deterministic short/long-context and mixed-TTL price fixtures

Represent context-tier and cache-TTL rates in the canonical registry. Compute
conservative supported-request upper bounds and actual settlements from explicit
usage categories. Preserve compatibility for unrelated models and make insufficient
pricing context visible. Do not change role ordering or eval suite membership.

**Acceptance criteria:**

- [x] Tests cover exactly 100K and above, cache reads, five-minute/one-hour writes, thinking/output limits, and upward rounding.
- [x] Unknown model/feature/rate or unproved token bound cannot authorize a request.
- [x] Existing pricing tests pass without changing unrelated model rates or ordering.
- [x] Request and settlement record the pricing revision used.

### ✅ M3: Authenticated budget gateway (~730 LOC)

**Dependencies:** M0, M1, M2
**Estimate:** 450 implementation + 280 tests; 2 days
**Ownership:** New focused Anthropic gateway package and service entry point
**Example/artifact:** Fake upstream with complete/truncated SSE and ambiguous-send fault injection

Authenticate job identity and server-bound task capability, enforce active leases,
and forward only supported Anthropic Messages requests to a fixed origin. Reserve
before every actual send, settle complete usage, and retain uncertainty after failures.
Preserve required supported headers, caching fields, and streaming. Read the provider
key server-side from Secret Manager; never return it or log credentials.

**Acceptance criteria:**

- [x] Every upstream send has its own durable sufficient reservation, including retry/helper/subagent paths admitted by the contract.
- [x] Crash around sending, truncated SSE, client cancellation, duplicate events, and gateway restart preserve conservative exposure.
- [x] Unsupported billable routes/features/models and invalid credentials produce zero upstream inference calls.
- [x] Provider key remains gateway-only; capabilities cannot choose another account or enlarge limits.
- [x] Fixed-origin and streaming contract tests pass.

### ✅ M4: Cloud executor and routing integration (~300 LOC)

**Dependencies:** M1, M3
**Estimate:** 160 implementation + 140 tests; 1 day
**Ownership:** Existing registry/dispatch, Claude executor/env policy, cloud executor wrapper, admission/status seams
**Files:** `internal/coordinator/agent_registry.go`, `cloud_dispatcher.go`, `daemon_tasks_exec.go`,
`internal/dispatch/cloudrun/dispatcher.go`, `internal/executor/claude/`,
`internal/executor/envpolicy.go`, `cmd/ailang/coordinator_cloud_executor.go`
**Example/artifact:** Explicit Haiku canary registry entry paired with its Claude API-key job variant

Carry trusted account/capability/gateway metadata. Require positive task budgets in
the credit lane and preserve billed auth/accounting. Exclude subscription credentials
and alternative provider keys. Report the separate `anthropic-api-credits` account.
Verify candidate traversal in the concrete path; pinned work remains visibly blocked
where a supported successor is unavailable.

**Acceptance criteria:**

- [x] Job image, provider, auth mode, model, and account resolve coherently before launch. Guarded dispatch tests and actual CLI2.1.295 completion/tool-loop fixtures pass; deployed immutable image verification is an M6 gate.
- [x] Invalid/missing budgets fail closed rather than become zero/unlimited.
- [x] Budget-blocked work does not spin on retries or silently switch credentials/models.
- [x] Existing request-scoped user keys, OAuth lanes, and task/PR approval rules retain their behavior.
- [x] Metered and subscription accounting remain distinguishable; relevant regression tests pass.

### ✅ M5: AILANG renewal and status commands (~300 LOC)

**Dependencies:** M1
**Estimate:** 180 implementation + 120 tests; 1 day
**Ownership:** Coordinator CLI, config/help, operator authorization and credit audit interface
**Files:** proposed `cmd/ailang/coordinator_credits.go`, command tests, existing coordinator/help/config owners
**Example/artifact:** Operator guide with `credits confirm/status --remote gcp --account anthropic-api-credits`

Implement attended confirmation and equivalent explicit input flags. Show project,
organization, previous grant, available exposure, and resulting allowance before the
final confirmation. Use provider grant ID or a stable organization/cycle identifier.
Record identity/time/evidence; resolve current allowance server-side and reject duplicate
or conflicting cycle confirmations. Commands must distinguish grant confirmation from
account enabling and ordinary task approval.

**Acceptance criteria:**

- [x] Confirm writes audited canonical cloud state atomically and never takes an API key as input.
- [x] Same-cycle repeat adds no credit; invalid dates/amounts, unauthorized identity, and stale conflicting state fail visibly.
- [x] Status shows expiry, confirmed/operating ceilings, settled/reserved/unresolved exposure and every blocker.
- [x] Executors observe the next allowed request without a config restart; disabled/unreconciled accounts stay blocked. Authority/gateway cases pass and installed CLI reaches that request path; live canonical service observation remains M6.
- [x] Help and operating examples agree with actual flags; CLI tests pass.

### ✅ M6: Integrated validation and rollout evidence (~180 LOC)

**Dependencies:** M0, M1, M2, M3, M4, M5
**Estimate:** 100 documentation/config + 80 integration tests; 1 day
**Ownership:** Integration fixtures, operating guide, scoped external infrastructure changes and canary evidence
**Examples/artifacts:** `docs/internal/cloud-coordinator-config.md` credit section;
`design_docs/verification/cloud-claude-api-credit-guard/` cost/expiry/kill-switch evidence

Run emulator and fake-upstream integration first. Prepare concrete gateway/job/IAM/secret
bindings in the identified infrastructure checkout. Deploy through its applicable
approval workflow, reserve a $5 aggregate smoke allowance within the grant/daily ceilings,
then execute the canary. Verify provider billing and every hidden inference call before
promoting selected agents. Document renewal, key rotation, status, disable, and rollback.

**Acceptance criteria:**

- [x] Dev/prod integration uses one credit authority for the shared grant; all smoke spend stays within the reserved $5.
- [x] Provider evidence reconciles supported billing categories; discrepancies block promotion.
- [x] Near-limit, expiry-without-renewal, reconfirmation, kill-switch, and unavailable-ledger scenarios show expected behavior.
- [x] All relevant tests, formatting/lint, boundary checks and required CI pass.
- [x] Deployment changes are reviewable under infrastructure policy; canary/rollout evidence and operator guide are complete.

## Day-by-Day Schedule

| Day | Work |
|---|---|
| 1 | M0; start M1 grant/attempt state and emulator concurrency cases |
| 2 | M1 request reservations, settlement, grant-transition failure cases |
| 3 | Complete M1; M2 canonical rates and request bound fixtures |
| 4 | M3 authenticated forwarder, reservation before send, streaming |
| 5 | Complete M3 crash/retry/uncertain-usage fault injection |
| 6 | M4 job metadata, strict budgets, environment and routing integration |
| 7 | M5 confirmation/status commands, audit and operator tests |
| 8 | Cross-module emulator integration; prepare external infrastructure review |
| 9 | M6 reserved canary, billing evidence, rollout/rollback docs and evaluation handoff |

M0 provider/infrastructure access and M6 provider report availability may add waiting
time. Preserve progress rather than replace missing evidence with assumptions.

## Success Metrics and Verification

Total estimates: 80 + 700 + 320 + 730 + 300 + 300 + 180 = **2,610 LOC**.
Target planning capacity: 300 LOC/day, 9 days. Use failure-mode/invariant coverage as
the testing metric: every budget transition and every ambiguous send has a deterministic
regression case. Arbitrary line coverage is not a spending guarantee.

Run targeted package checks and Firestore-emulator integration. Run `make fmt`, required
lint/tests, `make check-boundaries`, and required CI before completion. Broaden/repeat
checks only for changes or unresolved failures. Have sprint-evaluator assess all frozen
criteria and the live activation evidence. No `.ail` module changes are planned; the
AILANG syntax gate is inapplicable. Working examples are operator CLI/config flows and
request fixtures, verified against the implemented host behavior.

## Dependencies, Deferred Work, and Execution Gate

Before live activation: access to the receiving credit organization, available grant
and allocation, provider secret setup, exact infrastructure ownership, supported request
bounds, and authentication contract. These are M0/M6 conditions, not evidence already
obtained in this planning session.

Follow-up: other providers' credit accounts and confirmation flows. Retain account/provider
identifiers and generic grant/usage records needed by this Claude scope; avoid building
additional adapters, a broad provider gateway, or migrations in this sprint.

Execution requires the user's explicit execute instruction under [AGENTS.md](../../../AGENTS.md).
Execution was authorized on 2026-10-10. M0–M5 have passed local gates, including actual installed-CLI completion and a two-request tool loop with separate reservations and settlement. M6 remains pending deployed image compatibility, live infrastructure, provider ownership evidence, CI and canary reconciliation.
Keep the pre-existing uncommitted work and active sprint state untouched.

The progress JSON is authored using the inspected generator's schema. The generator
also imports GitHub messages and reads linked messages as a side effect; this attended
planning task has no linked issue and does not authorize message acknowledgement.
Accordingly, populate equivalent state directly, validate it with the existing sprint
validator, and defer executor handoff until the explicit execution instruction.

## Historical execution evidence before recovery — 2026-10-10

The user explicitly authorized this sprint. Implementation used an isolated worktree to preserve an unrelated unresolved merge. Full `make test`, `make lint`, architecture boundaries, focused race/fault tests and real Firestore emulator transaction contention passed. The final concurrent suite hit two existing process-start deadline failures; the cleanup regression passed in isolation and the full suite passed with `GOTEST="go test -p 4"` without source changes. That default-suite recheck did not resolve the initially failing installed-CLI protocol gate; the subsequent bounded client contract described below does. The infrastructure preview passes Terraform fmt/validate and its IAM patch passes apply-check.

The authority enforces an aggregate $5 canary sublimit until audited `credits promote` records provider reconciliation; promotion never resets spending. Review fixes added strict nested request contracts, paired Cloud Run authentication headers, attempt-scoped lease cleanup and downward-rounded task limits. UTC daily exposure is assigned by admission date, not invoice date.

The initial opt-in fake-only installed CLI diagnostic failed: Claude 2.1.295 sends `interleaved-thinking-2025-05-14`, `mid-conversation-system-2026-04-07`, `claude-code-20250219` and `effort-2025-11-24`. The gateway rejected these before reservation or send, and continues to reject them if presented. Automatic approval review rejected allowing the unverified features; no allowlist expansion was applied. The user approved continuing the bounded compatibility review. The safer implementation disables these extensions at the client using the reviewed2.1.295 proxy-compatibility switch, keeps gateway beta rejection intact, and admits only documented standard text system messages. Actual CLI completion and a two-request Bash printf tool loop now pass under the race detector. M4/M5 local readiness is restored; no real inference occurred. See [CLI contract](m-cloud-claude-api-credit-guard-cli-contract.md).

M6 remains pending: built/deployed image compatibility, isolated authority and gateway deployment, narrowing existing runtime secret IAM, reviewed image/release integration, verified organization/workspace/cycle evidence, CI, live canary and provider billing reconciliation. No inference credit was consumed and the lane remains disabled. See [operator guide](m-cloud-claude-api-credit-guard-operator-guide.md) and [deployment evidence](m-cloud-claude-api-credit-guard-deployment.md).

### M6 recovery continuation — 2026-10-10

The attended user authorized continuing the repair, audited conservative recovery,
and dev/prod canaries. Execute the concrete authority → operator commands →
CI/release → disabled-ledger recovery → canaries sequence in the
[recovery addendum](m-cloud-claude-api-credit-guard-recovery.md). No uncertainty is
refunded; external diagnostics and the conservative balance observation also
consume the original allocation and $5 canary allowance. M6 remains incomplete.

## Final rollout — 2026-10-10

v0.54.1 exact source/image CI passed. The disabled canonical ledger booked all
original and attended exposure conservatively before new sends. Repaired dev/prod
canaries retained four complete provider receipts, settling13,025microUSD.
Audited promotion preserved all2,563,069booked microUSD, including2,550,040
conservative. No holds or reconciliation errors remain.

The existing read-only Haiku evaluator is enabled through Terraform CI commit
82b12885f85d0c75ac3f433e76df6fd4c57f4a90: dev4dfffd91, test784d8ec2,
prode343eb32 allSUCCESS. Test is excluded; both ready dev/prod coordinators
serve the approved rendered metadata; all44otheragents are preserved.
No IAM or folder policy changed. [Retained proof](../../verification/cloud-claude-api-credit-guard/rollout-v0.54.1.json)
and [retrospective](../../../docs/sprint-retros/M-CLOUD-CLAUDE-API-CREDIT-GUARD-retro.md)
document exact receipts, conservative classification, limits and follow-ups.
