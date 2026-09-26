# Dashboard live data audit — 2026-09-08

**Status:** Initial production data audit complete; visual interaction, full fleet producer census, controlled lifecycle tests and repair are outstanding.
**Scope:** Read-only production HTTP probes, Firestore metadata projections, Cloud Run deployment configuration/logs, deployed UI asset comparison and source-path inspection.
**Verdict:** The cloud stores contain useful operational records and provider traces, but do not currently provide reliable end-to-end fleet provenance. Fix capture and query contracts before replacing the UI.

## Deployment and measurement boundary

- Project/database: `ailang-multivac` / Firestore `(default)`.
- Dashboard: `ailang-dashboard-00044-469`, `europe-west1`, 100% traffic; revision created `2026-09-07T20:51:48Z`.
- Image: `dashboard@sha256:9539c120c6b77156f166cad987127d02fe7ff256b04fe81aa787bbde347f2da4` in `europe-west1-docker.pkg.dev/ailang-multivac/ailang`.
- Coordinator: `ailang-coordinator-00077-tjr`, 100% traffic. Both services configure `AILANG_STORAGE=gcp` and `AILANG_CLOUD_PROJECT=ailang-multivac`.
- `/api/version` and `/health` report `dev`, not a commit/release. Backend source correspondence remains inferred; the deployment image digest is pinned above.
- Deployed JS `/assets/index-CCtNYueW.js` is byte-identical to `internal/server/dist/assets/index-CCtNYueW.js`; SHA256 `0a9bc85cb58a357bf6d1597c2ece39ea36cbbd07df16a1c2955277daa10e918c`. This verifies the shipped UI asset, not a browser interaction test.
- Fixed span cohort: `2026-09-01T00:00:00Z <= start_time < 2026-09-08T00:00:00Z`. Task/chain cohorts use `created_at`; stages use `started_at`. These are intentionally different denominators.
- Firestore read times fall within `2026-09-08T05:46:45.321Z`–`05:46:45.707Z`; collections were separate reads, not one transactional snapshot. Later API samples may reflect new activity or cache refresh.
- Chain/stage/task/approval/session inventories cover all retained documents at that read time. Span counts cover only the fixed cohort. Retention and never-received events limit completeness claims.
- No browser surface was available. Network access succeeded with approved read-only escalation. No production writes, index creation, message acknowledgement, agent dispatch, or approval action were performed.

## What works and should be retained

1. **Message → task → stage → chain identities.** All 425 stages reference an existing task and chain. All 17 task request-message IDs in the fixed task cohort resolve to an existing inbox document, checked with field masks without marking messages read. No duplicate `(source_type, source_ref)` chain keys were found among 420 retained chains; this observation is not a concurrency guarantee.
2. **Recent stage accounting rolls up to chains.** All 16 cohort chains have matching stage cost sums (tolerance `1e-8`) and exact token sums. Individual chain detail returns real stage/session/accounting fields, even where the list is wrong.
3. **Provider trace ingestion and intra-trace parenting.** 3,161 retained cohort spans can be read directly. The sample span hierarchy returns provider roots and children. Raw provider cost attributes survive ingestion even though normalized cost is wrong.
4. **Approval records and some persisted evidence.** The pending API returns two records matching the store. History contains recorded decisions. Sample approved `apr-3807b3e1` retains a diff, changed-file list, session reference and handoff targets; `handoffs_triggered=true` is persisted.
5. **Historical chain cleanup has visibly occurred.** The store has 315 abandoned chains and no active chains. Do not repeat the earlier historical claim that 311 active chains still exist. Abandonment deliberately preserved stage history; it is not proof that stage state is now reliable.

## Measured inventory

| Retained collection | Count | Observation |
|---|---:|---|
| `obs_chains` | 420 | All `source_type=message`; 315 abandoned, 86 failed, 10 completed, 9 pending approval |
| `obs_chain_stages` | 425 | 395 pending, 11 awaiting approval, 2 running, 4 failed, 13 completed |
| Coordinator `tasks` | 425 | 231 cancelled, 169 failed, 10 completed, 10 pending approval, 4 no_changes, 1 rejected |
| `approvals` | 16 | 13 approved, 2 pending, 1 rejected |
| `obs_sessions` | 0 | No retained session records |
| `obs_chat_messages` | 0 | No retained chat rows |
| `obs_session_tools` | 0 | No retained tool records |
| `obs_spans`, fixed cohort only | 3,161 | 993 `LLM Generation`, 1,013 `generation`, remainder provider-attempt spans |

The fixed cohort contains 16 newly created chains, 17 newly created tasks and 17 started stages. None of the 3,161 spans has a nonempty `chain_id`, `stage_id` or `task_id`; 206 have a `session_id`, but there are no session records to resolve. Twelve cohort stages have a session ID; none names a provider. Across all 425 stages, 26 have session IDs and none has a provider/model value in the projected fields.

This proves a cloud coverage/linkage gap. It does not prove that missing sessions or traces never existed on local machines or Cloud Trace. No feature, manual or eval chain appears in this cloud chain collection. Codex/motoko/direct-session fleet coverage remains unverified, and provider model names are not evidence of the originating harness.

## Findings ordered by repair priority

### F1 — Coordinator dashboard export is skipped, then reported enabled

Production coordinator logs at `05:26:27Z` on September 8 say the dashboard OTLP endpoint is unreachable and is being skipped. At `05:26:28Z`, startup reports dual telemetry export enabled and lists that same endpoint. The same sequence appears at `05:04`, `03:47`, and `03:11`.

The configured endpoint is `https://dashboard.ailang.sunholo.com` without an explicit port. In `internal/telemetry/otel.go`, `isOTLPReachable` appends port `4318` whenever the URL omits a port, including HTTPS URLs. `InitDual` skips exporter construction when that check fails. This source defect is consistent with the observed production disablement. HTTPS reachability was independently verified on its normal port through successful API requests.

**Repair:** honor URL scheme/default ports or use the actual exporter transport for bounded checks; make exporter registration/delivery status, not configured environment variables, drive health. Avoid permanently disabling a destination after a transient startup failure. Inspect all initializer paths for the same contract.

**Acceptance:** a fresh coordinator task produces linked cloud spans; restart/unreachable/recovery cases expose the actual destination state. Log output cannot say the destination is enabled after skipping it. Include HTTPS-without-port as the regression case.

### F2 — Session/transcript/tool capture and fleet coverage are absent in cloud

The three capture collections are empty, while stages carry session IDs. Every retained chain is message-driven, and every cohort span lacks a work link. Current session-keyed correlation cannot join through an empty session collection.

**Repair:** inventory actual producer launchers and destinations per node; persist baseline lifecycle and session linkage from deterministic driver code. Reuse the existing feature-chain design, but verify workspace/session concurrency and crash behavior before adopting its shared state-file proposal. Explicitly carry capture mode, delivery watermark, pending spool and source revision.

**Acceptance:** one task from each supported producer can be followed to its artifact with tracing off; tracing on adds function evidence. Expected dispatches supply the denominator for coverage. Missing producer coverage must remain visible rather than being counted as zero activity.

### F3 — Firestore query parity fails on real API requests

| Request/field | Live result | Implication |
|---|---|---|
| Chains `limit=5,offset=0` versus `offset=5` | Identical rows | Pagination cannot enumerate work |
| Spans `offset=100,limit=5` in fixed cohort | Repeats first five rows of initial page | Span enumeration also broken |
| Stats with dates in January 2099 | 6,306 spans and nonzero tokens/tasks | Filter does not constrain the summary |
| `/api/controlplane/stats/breakdown` | HTTP 200, empty dimensions, cost 0 | Unsupported analytics masquerade as no activity |
| `/api/controlplane/task-evolution` | HTTP 200, empty tasks | Same cloud capability gap |
| `/api/controlplane/usage-timeseries` | HTTP 200, empty points/cost 0 | Same cloud capability gap |
| Chain list | Latest five rows: `stage_count=0`, `max_stage=0`, empty `agent_flow` | Detail contains stages; list projection is incomplete |
| Chain journey | HTTP 501 | Explicitly unsupported on cloud backend |

The future-filter response is not byte-identical to the earlier unfiltered response: ingestion advanced the span count from 6,302 to 6,306. The violation is that a future-only filter returns existing activity at all.

Source evidence: Firestore `ListChains` and `ListSpans` omit `opts.Offset`; `ChainSummary` construction omits stage-count/flow fields; summary filtering uses SQLite-specific methods and otherwise unfiltered metrics. The shipped UI renders `stages_completed/stage_count` and derives agent-filter options from `agent_flow`, so these fields directly affect user-visible behavior.

**Repair:** one backend capability/response contract with validated filters, deterministic pagination, explicit errors/partial states and correct projections. Do not port every old analytics view by default: defer views that do not serve the retained product.

**Acceptance:** same cohort/filter/page means same records and accounting through API and remote CLI. Disjoint pages cover the expected IDs; impossible filters return an empty result; unsupported operations return a named capability error.

### F4 — Required transcript and stage-span indexes are missing

Sample completed chain `0dca4358-7dcf-4e8b-8aab-a6c47d3df2ed`, stage `5168f212-3f7b-4dd7-8934-e18e107b8417`: stage-span endpoint HTTP 500. Pending chain `f3738b59-d194-4ca8-8d16-3ebf32756724`, stage `a2d6bd66-9c9d-4c8c-9898-936b9b085c02`: chat endpoint HTTP 500. Dashboard logs explicitly report Firestore `FailedPrecondition: The query requires an index` for both.

**Repair:** declare the exact query/index set in deployment infrastructure and verify readiness before traffic. Return a useful named error instead of generic failed-to-get text. Fixing indexes alone will still leave chat empty until F2 is addressed.

### F5 — Costs survive raw ingestion but disappear from normalized metrics

In the initial 100-span API sample, 38 `LLM Generation` spans carry `gen_ai.usage.total_cost`, summing to `$0.487395437964`. All have zero/absent normalized `cost_usd`. Example span `6f31508ce4906cdc`: raw total `$0.0001996701`, normalized cost absent. The cohort projection has no nonzero normalized span costs. Aggregate observatory cost is therefore zero despite retained provider evidence.

`internal/observatory/otlp_receiver.go` cost extraction recognizes `gen_ai.usage.cost`, `ailang.cost.usd`, `ai.cost_usd`, and `task.cost_usd`; the inspected paths do not use the observed `gen_ai.usage.total_cost` key. Separately, ten cohort stages have positive cost while their coordinator task's `cost` is zero. That is why task-backed and stage-backed views disagree even where chain rollup is internally correct.

**Repair:** normalize supported producer schemas and name the accounting authority for each view. Parent/child/provider-attempt observations must not be added as separate charges for the same billed call. Track unknown, metered, estimated and subscription usage separately. Recompute historical projections only after the rule is approved and tested.

### F6 — Approval decisions do not provide a trustworthy delivery state

The approval queue contains two pending decisions; coordinator stats report ten pending approvals because ten tasks remain `pending_approval`. Eight of those tasks already have approved approval records. Sample `task-3807b3e1` has an approved decision and persisted `handoffs_triggered=true`, while the task and stage remain awaiting approval. This does **not** establish that the approved action failed to run; it establishes that the work state cannot explain its outcome.

For pending `task-10b5e305`, the diff endpoint returns HTTP 404 `Task has no worktree`. Its approval context does carry `diff_unavailable`, but the pending-list response provides neither the diff nor that explanation. The task record has zero cost/provider/tokens while its stage has real accounting. The sampled completed/no-change task has zero returned task events, so event history cannot supply the missing evidence either.

**Repair:** distinguish approval-needed, decided, dispatched, executed, failed and expired; retain immutable target/artifact identity and the executor's outcome receipt. Surface unavailable evidence explicitly. Preserve the existing persisted-diff path—it works for some records. Do not claim a stored decision is delivered work, and do not rewrite historical state based on age alone.

**Acceptance:** replay/duplicate decisions, stale artifacts, offline owners, handoffs and execution failures are tested with isolated fixtures. No real approval was exercised during this audit; authorization and action correctness are not certified.

### F7 — Legacy lifecycle and health semantics mislead

All-time stages include 230 pending stages whose tasks are cancelled, 164 pending stages whose tasks failed, and two running stages whose tasks failed. Most are historical and chain reconciliation deliberately did not fabricate stage transitions. A dashboard must identify historical/incomplete evidence instead of reporting those stages as currently running work.

`/health` still says healthy despite absent capture collections and failed indexed queries. The summary labels sources with local SQLite paths even on this verified Firestore deployment. Its success rate measures non-error spans, while `error_count` remains zero in the response despite errors in the span population.

**Repair:** separate service liveness, query capability, producer freshness and data completeness. Add actual version/revision/source identity. Make historical exceptions explicit and keep them separate from the fresh correctness cohort.

## Representative work dossiers

| Work | Established | Missing or contradictory |
|---|---|---|
| `task-18d1d05d` / chain `0dca4358…` | Request identity, stage, session ID, finalization steps marked done; stage/chain cost `$0.05244525`, 25,821 tokens; task `no_changes`, chain completed | Task cost/tokens remain zero and `completed_at` null; event history empty; stage spans fail index query; no cloud session/transcript record. No delivered change is implied by `no_changes`. |
| `task-10b5e305` / chain `f3738b59…` | Request, execution stage, pending merge approval, stage cost `$0.0405202`, 31,602 tokens | Diff unavailable; explanation does not reach pending-list response; chat query fails; cloud session/tool capture absent |
| `task-3807b3e1` | Approved merge/handoff decision, persisted diff and session reference, handoff marked triggered | Task still pending approval; decision is not linked to a verifiable final delivery receipt in audited surfaces |

## Dashboard disposition grounded in the audit

| Current surface | Decision for redesign | Reason |
|---|---|---|
| Chain/work list and detail | Keep concept, repair API, make primary | Useful existing identity and accounting spine |
| Approval queue/history | Keep, join to same work detail | Core user purpose; evidence and outcome gaps must be fixed |
| Chat/messages/tool history | Merge into work evidence | Empty capture must be repaired, not hidden behind another view |
| Span hierarchy/timeline | Keep as optional drilldown | Intra-trace parenting works; global work linkage does not |
| Execution hierarchy/evolution/tree alternatives | Consolidate after user-flow verification | Multiple representations of incomplete work data |
| Breakdown, usage and evolution analytics | Defer from initial replacement | Live empty-success responses and cloud-only gaps |
| Activity heatmap | Optional secondary view | Contains coordinator task data and cost, but does not represent all fleet activity |
| Function inspection | Stage after baseline capture acceptance | No function-execution spans in the retained cohort; trace-on/off behavior remains to be demonstrated |

The three proposed entry points remain Work, Approvals and Functions. No chart or route has been deleted. A visual audit is still needed before final route removal, layout and interaction decisions.

## Revised implementation sequence

1. **Restore capture and tell the truth about health (F1).** Smallest independently demonstrable fix; prove fresh coordinator export after restart and destination recovery. This is more specific than starting with another database redesign.
2. **Make work/session identity durable across producers (F2).** Amend the existing feature-provenance design. Enumerate node/harness responsibilities and baseline capture, including artifact outcomes and optional function depth.
3. **Certify cloud query/CLI parity and accounting (F3–F5).** Pagination/filter contract, stage projections, required indexes, schema normalization and one accounting authority. Defer unsupported analytics rather than quietly returning zeros.
4. **Complete approval-to-delivery provenance (F6).** Immutable evidence, state transitions, dispatch/execution receipts and explicit unavailable evidence.
5. **Observe a fresh cohort, then reconcile legacy history (F7).** Preserve raw facts; bounded dry-run before any repair. Seven-day reliability gate remains in the recovery plan.
6. **Replace the interface and remove redundant views.** Only certified fields enter the new UI. Complete browser-based acceptance and rollback-capable cutover before retirement.

Use repository design approval → sprint planning → execute sprint → evaluation gates for implementation. No implementation is approved by this audit. Reuse existing designs and primitives; do not create a second work identity or a parallel set of CLI commands.

## Reproduction and evidence

Base URL: `https://dashboard.ailang.sunholo.com`. All dashboard probes were GET requests with 10-second connect and 20–45-second total timeouts. Representative paths:

```text
/health
/api/version
/api/chains?limit=5
/api/chains?limit=5&offset=5
/api/controlplane/stats
/api/controlplane/stats?start_date=2099-01-01&end_date=2099-01-02
/api/controlplane/stats/breakdown
/api/observatory/spans?limit=100&start_after=2026-09-01T00:00:00Z&start_before=2026-09-08T00:00:00Z
/api/observatory/spans?limit=5&offset=100&start_after=2026-09-01T00:00:00Z&start_before=2026-09-08T00:00:00Z
/api/chains/{chain_id}
/api/chains/{chain_id}/journey
/api/chains/{chain_id}/stages/{stage_id}/spans?limit=5
/api/chains/{chain_id}/stages/{stage_id}/chat
/api/coordinator/tasks/{task_id}/events?limit=100
/api/coordinator/tasks/{task_id}/diff
/api/approvals?status=pending
/api/approvals?status=all
```

Because API offsets are broken, inventory was measured with Firestore `documents:runQuery` metadata projections. Caps: chains/tasks/approvals/sessions/chat/tools 2,000 each; stages 5,000; cohort spans 10,000. Every result was below its cap. Span query explicitly used a half-open timestamp window. Other inventories were unfiltered. Fields covered identity, links, state, timestamps, provider/model and normalized accounting; prompts/transcripts were excluded from projections. Sample task/approval details were examined separately without publishing their payloads.

Useful read-only instruments used: `gcloud run services describe`, `gcloud run revisions describe`, `gcloud firestore indexes composite list`, and bounded `gcloud logging read` queries scoped to the dashboard/coordinator services. The Firestore query API uses HTTP POST for a read; no document mutation API was called.

Raw responses and temporary projection/summary scripts are in `/tmp/ailang-dashboard-audit/`; they contain private payloads and configuration and are not repository artifacts. [Sanitized measurements](dashboard-live-audit-2026-09-08.json) retain counts and denominators. The raw API sample is not a random sample and its cost sum is not a fleet cost estimate.

## Remaining limits

- Browser unavailable: no rendered UI, navigation, WebSocket or visual accessibility certification.
- Full producer census across laptop/rig/other nodes and Cloud Trace ingestion not completed. The audit proves what is retained in this cloud database, not the total expected fleet workload.
- No synthetic production work was dispatched. Retry/offline recovery, function tracing on/off and approval execution need controlled fixtures during implementation.
- CLI code paths were reviewed, not remotely executed: CLI startup unconditionally invokes local observatory retention against the real home directory. The previous read-only listing attempted this cleanup; executing the binary with expanded filesystem access could mutate unrelated local data. `ailang dashboard stats` is already an HTTP client, but calls the broken breakdown endpoint; several `chains` views still explicitly refuse remote reads.
- GET probes succeeded without an Authorization header. Decide the intended read-access policy for raw prompts, transcripts and artifacts before expanding capture; write/approval authorization was not tested.
- Repair estimates and final contract design follow review of these findings. This audit does not justify replacing the physical databases yet.

Related: [Recovery plan](dashboard-recovery-plan-2026-09-08.md), [feature provenance](planned/m-feature-provenance-chains.md), [unified mission telemetry](implemented/v0_33_2/m-mission-loop-unified-telemetry.md).
