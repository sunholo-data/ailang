# M-CLOUD-CLAUDE-API-CREDIT-GUARD: Claude cloud execution within confirmed API credits

**Status:** Design approved; execution authorized 2026-10-10. Budget guard locally validated; M4 CLI compatibility blocked and M6 rollout pending
**Created:** 2026-10-09
**Target:** Next scheduled release after v0.53.0; version assigned during sprint planning
**Priority:** P1
**Estimated:** Initial design estimate 5–7 engineering days; refined sprint estimate 9 days including tests and rollout verification
**Dependencies:** Existing Claude executor, Cloud Run dispatch, model registry, Firestore; infrastructure changes in the multivac infrastructure repository
**Scope:** AILANG harness/cloud tooling. No language semantics or motoko core changes.

## Problem Statement

Mark wants to reintroduce Claude, initially Haiku 5.5, into cloud executor rotation using a newly created Anthropic API key and a $200 credit allowance. The allowance must bound total API consumption across jobs, retries, and environments.

**Confirmed user decision:** Renew the allowance only after fresh credits are confirmed. A calendar rollover, coordinator restart, key rotation, or task approval must not replenish it.

**Approval and scope:** Mark accepted the design and AILANG confirmation interface in
the attended conversation on 2026-10-09: “yeah looks good and for all credits actually
but we will do that as a follow up”. This sprint implements Claude API credits only.
Extending the confirmation/status workflow and guards to other credit accounts is a
recorded follow-up, requiring its own provider-specific design and rollout.

The existing API-key execution path is reusable. Its per-task, event-driven spending checks are insufficient for a shared credit allowance. A task can incur costs before its next usage event, while multiple tasks can independently pass the same spend check. Existing Anthropic quota observation measures subscription windows; API credits require a separate dollar budget.

## Verified Current State

| Area | Evidence from current checkout | Consequence |
|---|---|---|
| Cloud authentication | `internal/coordinator/daemon_tasks_exec.go` carries `agent.AuthMode`; `internal/dispatch/cloudrun/dispatcher.go` selects `agent-executor[-variant]-apikey`; `internal/executor/claude/claude.go` uses API-key mode | Extend this path rather than introduce another agent harness |
| Key transport | Implemented M-CLOUD-DUAL-AUTH accepts request-scoped keys through a KMS-encrypted, in-memory cache | Fleet credentials need durable secret ownership; do not depend on a ten-minute message cache |
| Task limits | Cloud wrapper parses `AILANG_MAX_COST_USD`, cancels through `cloudEventHandler`; malformed values currently become zero | Credit-backed execution must require valid positive limits and refuse malformed configuration |
| Accounting | Claude distinguishes billed API-key runs from subscription runs in `internal/executor/claude/cost.go` | Preserve that distinction in routing and reports |
| Quota observation | `internal/mission/anthropic_quota.go` queries `/api/oauth/usage` | This subscription endpoint is not evidence of remaining API credits |
| Shared admission | `internal/mission/dispatch/admission.go` explicitly says admission is an observation, not a reservation | Add atomic spending reservations; retain existing admission/routing machinery |
| Haiku registration | `internal/modelreg/models.yml` already registers explicit `claude-haiku-5-5` and includes it in the executor role chain | Do not add a duplicate model or treat local placement as cloud readiness |
| Pricing gap | That row explicitly warns its single rate card understates turns above 100K prompt tokens | Credit enforcement must account for both pricing tiers and cache TTLs |
| Deployed registry | Cloud registry is in per-project GCS config; see `docs/internal/cloud-coordinator-config.md` | Inspect live jobs/config before rollout; repository state alone does not prove deployed readiness |

The checkout version is v0.53.0 according to the design-doc scaffolder, newer than the ambient session banner. Live job images, enabled agents, secret bindings, and linked Console organization were not inspected in this planning session.

## Provider Facts and Boundaries

Anthropic documents $200 monthly API credits for Max 20x. Grants follow the subscription billing cycle, expire without rollover, and are shared across the linked Console organization. Purchased credits or auto-reload can keep usage running after the grant is exhausted. The new key's organization and actual credit amount/expiry must therefore be confirmed before enabling it. API-key runs of `claude -p` are covered; subscription login and Vertex AI usage do not draw on these credits. [Monthly API credit documentation](https://support.claude.com/en/articles/17154008-monthly-api-credits-for-max-and-team-plans).

A dedicated non-default workspace can carry its own monthly spend cap and workspace-scoped key. This is a second safeguard, rather than a grant-period ledger: a monthly cap does not prove that a fresh credit grant arrived. Do not assume the key itself has a $200 ceiling. [Workspace limits](https://platform.claude.com/docs/en/manage-claude/workspaces).

`--max-budget-usd` uses a client estimate and can overshoot. It is useful to stop wasteful agent work, but cannot enforce the shared hard limit alone. [Claude CLI reference](https://code.claude.com/docs/en/cli-reference).

## Goals

1. Enable explicit Haiku 5.5 API-backed cloud runs with the existing Claude harness.
2. Keep the sum of settled spend and conservatively reserved requests within a confirmed allowance of at most $200.
3. Fail closed when credits, request pricing, accounting, or authorization cannot be established.
4. Remove the API lane from eligible rotation when its allowance is unavailable; retain the existing work approval rules.
5. Expose remaining credit, reservations, expiry, task spend, and refusal reasons for operators.

## High-Impact Decisions

| Decision | Rationale | Authority | Deadline | Change cost |
|---|---|---|---|---|
| Renewal requires confirmed new credits | Explicit user choice; prevents accidental paid continuation | Human — confirmed 2026-10-09 | Design | Medium |
| Initial model allowlist is Haiku 5.5 only | Predictable pricing and bounded rollout; other Claude models need explicit inclusion | Human — approved 2026-10-09 | Design | Low |
| Provider key stays in a shared budget gateway | Every billable request, including hidden CLI calls, crosses one spending boundary | Human — approved 2026-10-09 | Design | High |
| Firestore transactions reserve before forwarding | Concurrent jobs cannot spend the same remaining allowance | Human — approved 2026-10-09 | Design | High |
| Start with a $190 operating ceiling within $200 confirmed credits | Leaves $10 for reconciliation uncertainty; margin is not a substitute for reservations | Human — approved 2026-10-09 | Design | Low |
| All users of the linked grant must be accounted for | Workspace isolation does not isolate the organization's credit balance | Human — confirm deployment facts | Before rollout | Medium |

### Design Freeze

The gateway/reservation approach, initial model scope, and starting limits are approved.
Before rollout, record the Console organization/workspace, grant amount and expiry,
other consumption, and infrastructure repository/job ownership. Do not substitute an
approximate task-only guard for the request guard during implementation.

## Solution Design

### 1. Credit account and provider-side setup

Create/use a dedicated non-default workspace such as `ailang-cloud-executors` in the organization that receives the credits. Verify the newly created key is scoped there; if it is not, provision a correctly scoped replacement during setup. Set the workspace monthly cap to $200 or less and verify organization limits permit the intended allowance.

Record evidence that the credits were claimed and are available, along with the grant expiry. Disable automatic replenishment with purchased credits for this workload. Confirm whether other keys/workspaces spend the same grant: for the strict credits-only pilot, require an exclusive credit allocation and account for every consumer of that allocation. If other organization traffic is uncontrolled, do not claim that this workload can use the full $200 free allowance; keep activation blocked until a defensible smaller allocation or exclusive usage arrangement is established.

Store the real API key in GCP Secret Manager, readable only by the gateway service account. Jobs receive a gateway credential, never the provider key. Keep dev and prod on the same canonical credit ledger if they use this grant. A key rotation preserves the credit account and spend history.

Proposed pilot limits:

| Control | Starting value |
|---|---:|
| Confirmed grant ceiling | At most $200, adjusted down for prior/shared consumption |
| Operating ceiling | $190 for an untouched $200 grant; proportionally reduced if less is available |
| Per-task ceiling | $2 |
| UTC daily spending ceiling | $6, covering settled and reserved requests |
| Concurrent Claude API tasks | 2 across dev and prod |
| Live smoke allowance | $5 total, included in the grant and daily limit |
| Expiry admission buffer | Hard request deadline plus 5 minutes before grant expiry |

These values are proposed policy, adjustable within the confirmed credit ceiling. The daily ceiling paces spending; renewal remains tied to the actual grant, not UTC midnight or the first of the month.

### 2. Reuse the Claude executor through a narrow API gateway

```text
Existing coordinator/rotation eligibility
  → registered Claude task + attempt + shared concurrency lease
  → existing Cloud Run Claude job, explicit model, metered auth
  → Claude -p → authenticated Anthropic-format budget gateway
  → atomic request reservation in canonical Firestore
  → direct Anthropic Messages API using Secret Manager key
  → usage settlement + normal streaming result/telemetry
```

Use `ANTHROPIC_BASE_URL` and a task/attempt-scoped gateway credential in the Claude job. Preserve the metered accounting lane. The current executor expects `ANTHROPIC_API_KEY` in API-key mode; it can carry a gateway credential there when configured for this trusted gateway. That value must never authenticate directly to Anthropic. Do not expose an OAuth token or saved subscription credential in these jobs.

Anthropic supports this gateway arrangement, including provider keys held server-side and Anthropic-format endpoints. The gateway must preserve streaming, `anthropic-version`, supported `anthropic-beta` headers, and caching fields. [Gateway overview](https://code.claude.com/docs/en/llm-gateway), [compatibility guide](https://code.claude.com/docs/en/llm-gateway-protocol).

Provide `/v1/messages` and compatible token-counting/model discovery routes as needed by the pinned CLI. Reject unsupported billable endpoints/features. Start with standard Messages inference and local client tools; paid server tools, batches, managed agents, priority/fast services, and automatic model switching are outside the pilot. Compaction, helper calls, and subagents must use allowlisted, correctly priced models through the same gateway or be disabled. Inspect actual requests to establish this before rollout.

Gateway credentials are short-lived, bound server-side to task ID, attempt ID, credit account, authorized model set, and job identity. The container cannot choose a different credit account or increase its task limit. Authenticate the job identity as well as its task capability; check task registration and current lease on every request. Pin the gateway URL in trusted managed settings, restrict job identity access to the provider secret, and exclude alternative provider credentials. The gateway forwards only to the fixed Anthropic origin and strips its own credentials from upstream requests.

This is a spending boundary for the specified provider account. Existing code execution permissions and git/PR approval contracts remain separate.

### 3. Durable grant and request ledger

Use the repository's Firestore storage conventions, with a small credit-account store consumed by coordinator admission, gateway, and reporting. Do not use local SQLite or a machine-local JSON file as spending authority for cloud jobs.

Store money as integer micro-USD, rounding reservations upward. Store:

- Credit account: organization/workspace identifiers, secret reference, enabled flag, confirmed grant ID/amount/start/expiry/evidence, operating ceiling, settled total, outstanding reservations, daily totals, pricing revision.
- Task attempt: authorized model set, task ceiling, cumulative settled/reserved spend, job identity, lease, state.
- Request: unique gateway request ID, task attempt, maximum reserved cost, upstream request ID, usage categories, actual price, pricing revision, and lifecycle state.

Enforce atomically before any billable request leaves the gateway:

```text
settled_account + reserved_account + upper_bound(request) <= operating_ceiling
settled_task    + reserved_task    + upper_bound(request) <= task_ceiling
settled_day     + reserved_day     + upper_bound(request) <= daily_ceiling
```

Also require active confirmed grant, sufficient expiry headroom, valid task lease, enabled account, and verified pricing. If any check or transaction fails, do not call Anthropic.

Request reservation lifecycle: `reserved → forwarding → settled`, or `reserved → released` only when non-forwarding is proved. A crash or timeout after forwarding starts becomes `unresolved`; retain its full reservation. Lease expiry alone must never release a request reservation. Settlement is idempotent; duplicate completion events cannot refund or charge twice.

Each actual upstream send has its own reservation. A retried HTTP request may be a new billable call even if its body matches an earlier one. Do not deduplicate billable attempts solely by prompt hash or reuse a settled reservation. Persist intent before sending; after a crash, do not automatically repeat an ambiguous send. HTTP errors may release reservations only when evidence proves the request was not billable; otherwise retain them pending reconciliation.

### 4. Conservative request pricing before sending

Haiku 5.5 has a 100K prompt threshold with different input/output/cache prices above it. Cache creation has distinct five-minute and one-hour rates. Thinking counts in the allowed output budget. Use explicit model IDs and versioned verified pricing; never enforce the credit ceiling using the current single short-context rate card. [Haiku specifications and pricing](https://platform.claude.com/docs/en/models/haiku-5-5/overview).

For the first pilot, reserve a conservative maximum using the verified maximum billable input per accepted request and the submitted `max_tokens`, with the highest allowed context-tier/cache-write prices. A full-context bound is intentionally conservative and can be settled down after completion. A token-count estimate alone is not a proven upper bound.

For example, if the supported request contract proves at most 1,000,000 billable input tokens and 128,000 output tokens, using the published maximum $1/M cache-write rate and $2.50/M output rate reserves $1.32 before the call. This is a planning example, not authorization to assume those bounds for every API feature: verify cache refresh accounting, thinking, beta flags, and any input/output multipliers in the supported contract first. Reduce the allowed output limit if required by the $2 task ceiling or available daily budget; refuse a request that cannot be bounded.

On a complete response, calculate actual spend from uncached input, cache reads, cache writes split by TTL, output including thinking, context tier, and allowed service tier. Keep the gateway ledger authoritative for budget decisions; bank CLI totals as a second observation. Unknown usage or an actual cost above its reservation freezes new admission and retains conservative exposure until reconciled.

Add long-context/cache-TTL rate support to the canonical model pricing schema/helper, rather than copy a second permanent price table into the gateway. Keep existing consumers compatible and explicitly report when an old caller lacks the context information needed for accurate pricing.

### 5. Credit renewal and reconciliation

The first implementation uses an operator-confirmed grant record from Console billing evidence. It does not require giving executor jobs an Admin API credential. A normal workspace inference key cannot access the organization Usage and Cost API; automated reconciliation needs a separately authorized Admin credential in a control-plane service. Those reports provide cost/usage evidence, not proof that a new credit grant has been deposited. [Usage and Cost API](https://platform.claude.com/docs/en/manage-claude/usage-cost-api).

**Proposed AILANG operator interface — not implemented yet:**

```bash
ailang coordinator credits confirm --remote gcp --account anthropic-api-credits
ailang coordinator credits status --remote gcp --account anthropic-api-credits
```

`confirm` opens an attended renewal flow. It displays the canonical cloud project,
linked organization/workspace, previous grant, current exposure, and blocking reason.
The operator enters the fresh credit amount, currently available allocation, start/expiry,
and a billing evidence reference. Use a provider grant ID when available; otherwise
derive a stable cycle ID from the organization and grant validity interval. Show the
resulting allowance and require confirmation before persisting it. Provide equivalent
explicit flags for scripted entry; neither the API key nor an Admin credential is an input.

The command writes the new grant and an audit event atomically to the shared Firestore
credit account, using operator authorization distinct from task approval. It reports
the resulting lane eligibility and each remaining blocker. Executors and admission
read that canonical record on subsequent requests, so a successful renewal takes
effect without restarting jobs or changing agent configuration. A disabled account,
accounting discrepancy, or expired task capability remains blocked after renewal.
`status` shows the grant ID, confirmation time, expiry, settled/reserved/unresolved
exposure, available allowance, and eligibility. Reconfirming the same cycle returns
the existing result without adding credit.

At expiry/exhaustion, the lane is blocked until fresh-credit evidence is recorded. Confirmation must include a new unique grant identifier, amount, validity interval, current available allowance, and confirmation identity/time. Reconfirming the same grant is idempotent and adds zero allowance. Use the lower of $200 and the defensible allocation from fresh evidence.

Freeze new requests while closing an old grant. Existing requests remain reserved against the old grant; charge uncertain boundary-crossing costs conservatively until attribution is known. Do not discard unresolved old exposure when activating a new grant; subtract any exposure that could consume new credits. Retain history and reconciliation adjustments. Do not auto-increase limits from a task-cost approval, key replacement, new deployment, or a date change.

Reconcile settled requests against provider cost evidence before promotion and periodically in operation. Discrepancies increase exposure conservatively and freeze admission when they cannot be explained. Gateway/ledger unavailability blocks new Claude calls without preventing other eligible execution lanes.

### 6. Rotation and operator visibility

Treat this as a separate metered lane, labeled `anthropic-api-credits`, rather than the subscription `anthropic` quota bucket. Add its grant position and blocking reason to existing quota/status output. Keep provider, harness, auth mode, and credit account as separate attributes.

Start with an explicitly registered canary Claude cloud agent using Haiku 5.5 and the existing default/Go Claude image as appropriate. Resolve the matching image and API-key job; do not change only `model:` on a Codex job. Inventory live agents/jobs first. Promote the lane into selected executor rotation only after the canary and guard tests pass.

Use existing admission/routing seams for candidates that can run Claude. Verify the particular coordinator path walks eligible candidates; the older cloud-fallback proposal is explicitly untrusted and must not be treated as implemented. Where a cloud agent is pinned to Claude and cannot reroute safely, leave work visibly budget-blocked rather than spinning on retries. Do not quietly move API-credit work onto subscription credentials or another paid Anthropic model. A configured non-Anthropic successor must pass its own admission and existing work authorization.

Status should show grant expiry, confirmed/operating limits, settled/reserved/unresolved amounts, spend today, available task slots, pricing revision, and eligibility. Proposed refusal reasons are `credit_unconfirmed`, `credit_expired`, `credit_exhausted`, and `credit_accounting_unknown`, with structured amounts and task/grant IDs. Alert at 50%, 75%, 90%, and blocking states through existing operational alerting; the pilot does not send messages during planning.

Kill switch: disabling the credit account atomically prevents every subsequent upstream send, including further turns of an existing job. Already forwarded requests remain accounted for. Removing the lane from rotation is a secondary operational action.

## Implementation Plan and Timeline

| Milestone | Work and acceptance | Estimate |
|---|---|---:|
| M0 — deployment contract | Inventory current cloud jobs/images/registry, credit organization/workspace, key scope, grant evidence, other spend, infrastructure ownership; establish supported request/pricing bounds. All activation assumptions resolved or recorded blocked. | 0.5–1 day |
| M1 — shared credit authority | Firestore grant/task/request state, integer arithmetic, transactional admission, conservative crash handling, explicit operator grant confirmation, status output. Concurrency and renewal tests pass. | 1–1.5 days |
| M2 — pricing and gateway | Canonical tier/TTL pricing, authenticated Messages/SSE forwarding, reservation/settlement, supported-feature validation, fixed upstream, secret isolation. Failure-injection suite proves no unreserved send. | 1.5–2 days |
| M3 — executor and routing | Trusted gateway configuration, strict cloud task budgets, metered accounting, task capabilities, cloud job/variant pairing, quota reporting and blocked routing behavior. Existing approval contracts preserved. | 1 day |
| M4 — dev canary and production rollout | Allocate shared $5 smoke budget, compare provider and gateway costs, test near-ceiling/expiry/kill switch, promote selected agents and document rollback/renewal. | 1 day |

This is a design estimate, not an approved sprint plan. After design approval, use sprint-planner to refine scope/velocity, then sprint-executor only on an explicit execute instruction, followed by sprint-evaluator.

The [sprint plan](m-cloud-claude-api-credit-guard-sprint-plan.md) now provides the refined
nine-day breakdown and machine-readable milestone state. The table above retains the
initial design estimates for comparison.

### Expected Files / Ownership

| Area | Existing/new locations | Approximate change |
|---|---|---:|
| Pricing authority | `internal/modelreg/models.go`, `models.yml`, pricing helpers/tests | 200–350 LOC |
| Credit contract and Firestore storage | New focused credit-budget package + adapter under `internal/storage/firestore/` | 450–700 LOC |
| Gateway | New focused Anthropic budget gateway package + service entry point | 400–650 LOC |
| Dispatch and executor | `internal/coordinator/agent_registry.go`, `cloud_dispatcher.go`, `daemon_tasks_exec.go`, `internal/dispatch/cloudrun/dispatcher.go`, `cmd/ailang/coordinator_cloud_executor.go`, `internal/executor/claude/`, `internal/executor/envpolicy.go` | 200–350 LOC |
| Config/CLI/status | `internal/config/`, `cmd/ailang/`, quota reporting integration | 150–250 LOC |
| Infrastructure and operating guide | Multivac Terraform/service definitions, live config merge, existing cloud config guide | Scoped during M0 |

Estimates exclude tests. Reuse existing storage, secret, transport, and CLI conventions; do not introduce a general multi-provider gateway or edit motoko core. Identify the external infrastructure checkout before assigning exact Terraform files.

## Examples

**Concurrent requests near the operating ceiling:** settled spend is $188.00, pending exposure is $0.50, and operating ceiling is $190.00. Two jobs each request a $1.00 reservation. Only one can be admitted; the second sees insufficient capacity after the first transaction commits. A provider timeout leaves that $1.00 reserved until resolved.

**Expiry without renewal:** grant expires on the recorded billing-cycle date. The account moves to `credit_expired`; a job restart or next calendar month does not reset it. Confirming a genuinely new grant activates a new allowance after unresolved exposure is accounted for.

**Bad configuration:** malformed task limit or missing rate bound stops before model inference. It cannot become zero/unlimited. An ordinary task approval does not bypass the credit guard.

## Testing Strategy and Success Criteria

- [ ] Parallel requests at the ceiling cannot over-reserve; include multiple gateway instances and both projects against the Firestore emulator.
- [ ] Every upstream inference send has a durable sufficient reservation, including helper/compaction calls, retries, and subagents supported by the pilot.
- [ ] Crash after reservation, crash around sending, truncated SSE, lost completion, duplicate event, and task/job retry never release unknown exposure or spend twice on one reservation.
- [ ] Monetary rounding, both sides of 100K, mixed cache TTLs, thinking/output caps, and unsupported billable features have meaningful pricing tests.
- [ ] Missing/malformed budgets, unavailable ledger, unauthorized job, unknown model/pricing, expired grant, stale capability, and disabled account prevent upstream calls.
- [ ] Calendar rollover, key rotation, coordinator restart, same-grant reconfirmation, and task approval cannot replenish credits.
- [ ] Grant transition retains uncertain old exposure and refuses new admission until the next allowance is confirmed.
- [ ] Cloud jobs never receive the real API key or Admin credential; gateway credentials cannot authenticate to Anthropic or select another credit account.
- [ ] Dev canary runs use the same ledger; aggregate live validation stays inside the $5 smoke allowance.
- [ ] Provider-side billing evidence agrees with gateway cost categories within documented rounding/reporting timing; investigate discrepancies before promotion.
- [ ] Successful Claude canary produces the usual reviewable artifacts; expiry/exhaustion yields the expected blocked/eligible-next-candidate behavior.
- [ ] All relevant tests, formatting/lint, `make check-boundaries`, and required CI checks pass; operator guide documents setup, grant confirmation, secret rotation, status, and kill switch.

Use targeted tests for modelreg, storage, gateway, executor, dispatch, and coordinator, plus Firestore-emulator integration. The existing test suite is run during implementation; this planning change requires document/link verification only.

## Conflict Surface and Risks

- **Request-scoped keys versus fleet key:** M-CLOUD-DUAL-AUTH's KMS message-key flow serves external users. Preserve it and attach the credit account only through trusted registry/task metadata. An inbound user key must not inherit this fleet allowance.
- **Shared model registry:** Extend pricing without changing unrelated model ordering or global eval suite membership. Cloud promotion is independently validated.
- **Quota namespaces:** Keep subscription windows separate from actual API dollars; do not relabel historical subscription list-price estimates as credit consumption.
- **Firestore and gateway availability:** Fail closed, show a reason, retain unknown reservations, and allow other eligible lanes through existing routing. Reduced availability is preferable to erasing the spending cap.
- **Gateway maintenance:** Pin supported CLI behavior and forward supported headers/body fields without logging prompts or credentials unnecessarily. A CLI upgrade requires gateway-contract smoke validation.
- **Reporting authority:** Provider reports reconcile the ledger; delayed/history reports cannot authorize live requests or automatically confirm grants.
- **Shared organization credits:** A workspace cap alone cannot protect a free-credit allocation from other organization consumers. Resolve this before activation, rather than assume the new key owns $200.
- **Infrastructure cost:** The $200 guard applies to Anthropic API consumption. Cloud Run, Firestore, logging, and existing job compute remain separate infrastructure costs.

## Deferred Decisions

The implementer may choose focused package/file names, HTTP client/server wiring, and compatible CLI presentation. Token-count-based optimization of the conservative reservation can follow after its bound is proved. Automated grant confirmation is deferred until a provider interface that exposes actual credit grants is verified; manual evidence is sufficient for the first rollout.

**User-requested follow-up:** Extend the same account-based confirmation/status flow
and spending guards to all applicable credit pools. Preserve account/provider identifiers
in this implementation, but defer other provider adapters, grant rules, and migrations.

## Non-Goals

Subscription OAuth execution in cloud, Vertex/Bedrock funding, automatic purchased-credit top-ups, other Claude models at launch, a general gateway platform, broad cloud-fallback rewrites, changes to work approval authority, or changes to language semantics/motoko core.

## Axiom Compliance

| Axiom | Score | Basis |
|---|---:|---|
| A1 Determinism | +1 | Admission follows persisted limits and versioned prices |
| A2 Replayability | +1 | Grant, reservation, and settlement evidence retained |
| A3 Effect Legibility | +1 | Every API spend has an explicit reservation |
| A4 Explicit Authority | +2 | Confirmed grants and task capabilities bound spend |
| A5 Bounded Verification | +1 | Local arithmetic plus bounded transactional checks |
| A6 Safe Concurrency | +2 | Shared reservations serialize credit consumption |
| A7 Machines First | +1 | Structured eligibility/status and refusal reasons |
| A8 Minimal Syntax | 0 | No language syntax changes |
| A9 Cost Visibility | +2 | Actual metered dollars distinguished from subscription usage |
| A10 Composability | +1 | Existing Claude harness and routing retained |
| A11 Structured Failure | +1 | Unknown accounting blocks rather than goes unlimited |
| A12 System Boundary | +2 | Fixed provider origin and isolated provider credentials |

**Net +15.** No negative score for A1/A3/A4/A7. Compliance depends on proving request-cost bounds before activating the lane.

## Verification Log

| Claim | Check / source | Result |
|---|---|---|
| Existing API-key cloud execution | Read dispatcher, dispatch parameters, Claude authentication branch, environment grants | Confirmed in current source |
| Current cloud guard is event-driven; malformed limit becomes zero | Read `cmd/ailang/coordinator_cloud_executor.go` parsing and streaming handler | Confirmed; strict credit lane must remove this path to uncapped execution |
| Existing Anthropic quota reader is subscription observation | Read `internal/mission/anthropic_quota.go` URL and credential handling | Confirmed |
| Existing generic admission is not a reservation | Read `internal/mission/dispatch/admission.go` | Confirmed |
| Grant/reservation implementation to reuse | Search reservation/grant/monthly-budget terms across coordinator, mission, storage, dispatch, CLI | Found observational admission and unrelated quorum reserves; new credit authority needed in audited paths |
| Haiku already registered; short-context price undercounts long prompts | Read `internal/modelreg/models.yml` Haiku row and `internal/modelreg/models.go` pricing schema | Confirmed |
| Proposed four credit refusal names are unallocated in Go | `rg` across `internal/` and `cmd/` | No matches |
| Credit eligibility, renewal, sharing, and `claude -p` coverage | Official monthly API-credit support page, checked 2026-10-09 | Confirmed provider policy; this account's grant still requires evidence |
| CLI task cap can overshoot | Official CLI reference, checked 2026-10-09 | Confirmed |
| Gateway compatibility and rate bounds | Official gateway and Haiku docs | Supported approach; exact pilot request contract remains an M0 gate |
| Related-doc duplicate search | Scaffolder + `ailang docs search --neural --timeout 15s` | Neural reader reported `fallback-simhash`, zero embeddings; scores are not semantic evidence. Directly inspected the related auth/quota/fallback designs instead. |
| Canonical unread inbox | Attempted GCP `messages list --unread` | Timed out after 90 seconds; no messages read or acknowledged. Ambient banner reports 20 unread and 12 operator approvals. |
| AILANG-facing renewal interface | `ailang coordinator --help` and source search of coordinator/quota commands | Current CLI has no API-credit confirmation command; `coordinator credits confirm/status` above are proposed implementation scope. |

## Related Documents

- [M-CLOUD-DUAL-AUTH](../implemented/v0_9_2/m-cloud-dual-auth.md): shipped API-key transport; this design adds fleet secret ownership and grant-wide request enforcement.
- [M-CLOUD-PROGRESS-TRACKING](../implemented/v0_9_2/m-cloud-progress-tracking.md): existing streaming task cost cancellation, reused as a secondary stop.
- [M-QUOTA-RATIONING-ROUTING](m-quota-rationing-routing.md): provider bucket rationing; observational subscription admission does not implement API credit reservations.
- [M-CLOUD-PLANE-FALLBACK-LANES](m-cloud-plane-fallback-lanes.md): useful routing analysis; proposal explicitly untrusted. This design does not assume its implementation.
- [Cloud coordinator config](../../docs/internal/cloud-coordinator-config.md): live configuration location and dev/prod promotion workflow.
- [PROGRAM](../PROGRAM.md): harness/tooling change; motoko core remains frozen.
