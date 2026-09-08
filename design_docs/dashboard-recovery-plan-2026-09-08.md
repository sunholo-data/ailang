# Dashboard recovery: prove the data, then simplify the interface

**Date:** 2026-09-08
**Status:** Proposed recovery plan; initial production data audit complete. Visual interaction and full fleet lifecycle certification remain pending.
**Priority:** P1. Release target and implementation estimates follow the audit.
**Scope:** Fleet provenance, data contracts, API/CLI parity, approvals, and a minimal human inspection UI.
**Authorization:** Mark requested an audit and plan. This document authorizes no implementation, production repair, or approval execution.

## Outcome

A human or agent can start with a request, follow every execution attempt and handoff, inspect the evidence and approval decisions, and reach the delivered artifact. This works across cloud and local agents and across sessions. Optional AILANG function tracing adds detail without being required to establish the work lifecycle.

The first deliverable is an evidence-backed data audit, not a replacement dashboard. The second is a reliable query and approval contract. The replacement interface follows only after those contracts pass acceptance tests against the cloud backend.

## Preliminary audit: what is actually established

**Update, 2026-09-08:** The user authorized continuation and production access succeeded. See the [live audit](dashboard-live-audit-2026-09-08.md) for measured counts, deployment identity, reproduced failures and revised repair order. It supersedes the access limitation and historical-only assessment below. First repair priority is coordinator telemetry export: its HTTPS destination is skipped at startup while logs subsequently report dual export enabled. No implementation or production repair has been performed.

These are findings from the current checkout, not claims about the deployed revision or today's production records. No successful live dashboard/API read was obtained in this session. The browser action was rejected by automatic approval review over the repository inbox prerequisite. The canonical inbox listing did not return during the inspection and was cancelled; startup attempted local observatory retention, which failed with read-only errors and reported zero deletions. No messages were acknowledged.

| Surface | Finding | Evidence |
|---|---|---|
| Cloud storage selection | GCP construction creates Firestore coordinator, messaging, and observatory backends. Separate logical stores still exist, but another database migration is not justified by this alone. | `internal/storage/backend.go`, `NewGCPBackends` |
| Execution/span hierarchy | Handlers call the shared backend interface; the old SQLite-only handler restriction has been removed here. Runtime correctness and completeness remain unverified. | `internal/server/handlers_controlplane_exec_hierarchy.go` |
| Filtered summary statistics | With non-SQLite backends, the handler calls the unfiltered `GetMetricsSummary` even when filters were supplied. It also labels data sources with local SQLite paths. | `internal/server/handlers_controlplane_stats.go`, `handleControlPlaneStats` |
| Breakdown statistics | Non-SQLite backend returns initialized empty arrays with HTTP 200. Unsupported capability can look like no activity. | Same file, breakdown handler |
| Chain journey | Explicit HTTP 501 for a non-SQLite backend. | `internal/server/handlers_chains_routes.go`, `handleChainJourney` |
| Verified-success cost endpoint | Explicit SQLite requirement and HTTP 503 otherwise. | `internal/server/handlers_chains.go`, cost-per-verified-success handler |
| CLI cloud parity | Shared backend selection exists, but local is default. Explicit remote reads are refused for several views including journey, live, and find-by-task. | `cmd/ailang/chains_read_backend.go` |
| Cloud aggregates | Metrics summary scans span documents on cache miss; some workspace/task/agent read errors can leave zero or partial values without failing the summary. The reported success rate counts non-error spans, not delivered or verified tasks. | `internal/storage/firestore/observatory_aggregates.go`, `GetMetricsSummary` |
| Cloud rollups | Mission rollups explicitly unsupported. Stage cost rollup code does not apply `sourcePrefix`. Verify which current clients call this path before assigning production impact. | `internal/storage/firestore/observatory_chains.go` |
| Approvals | Coordinator-backed list and enrichment exist, including persisted evaluation/diff fields. A messaging-store fallback remains. Routing, authorization, resolution and execution need end-to-end verification. | `internal/server/handlers_approvals.go` |
| Reconciliation | Existing CLI supports dry-run and explicit apply; production Firestore implementation exists. Preserve and audit it. | `cmd/ailang/chains_reconcile.go`, `internal/storage/firestore/observatory_chains_reconcile.go` |
| Documentation | Database guide describes three SQLite databases; storage package header describes BigQuery for GCP while construction uses Firestore. Implemented-directory docs can still say Planned. Document location/status is not deployment evidence. | `docs/docs/guides/database-architecture.md`, `internal/storage/backend.go`, related documents below |

Historical evidence requiring fresh measurement:

- The August 31 feature-provenance proposal records populated eval and message-driven chains alongside missing mission/session capture and an index failure. Do not reuse those counts as current results.
- September 4 reconciliation source comments record a September 3 production audit of 400 chains, 311 active and stranded. Current production counts, deployed revision, and any repair outcome remain unknown.

## Reuse and consolidate existing work

This is a recovery brief, not a competing feature implementation design. Continue and amend existing designs after their premises are checked:

- [Feature provenance](planned/m-feature-provenance-chains.md): already records Mark's one-chain-per-feature decision. Owns feature identity and cross-session linkage. Audit its proposed state-file mechanism for concurrent sessions and stale attachment before adopting it.
- [Unified mission telemetry](implemented/v0_33_2/m-mission-loop-unified-telemetry.md): owns cloud write routing, session correlation, and the deliberately narrowed remote-read scope. Preserve explicit offline behavior; broadening read coverage needs an explicit contract revision.
- [Cloud observatory](implemented/v0_29_0/m-cloud-observatory.md): deliberately fixed only part of cloud parity. Its excluded surfaces explain why earlier completion did not mean dashboard completion.
- [Archived dashboard simplification](archive/v0_29_0/m-dashboard-simplification.md): largely a component decomposition plan. This recovery instead reduces product scope according to verified user questions.

At audit completion, publish one keep/amend/supersede table for related designs. Do not resume an old sprint based on a completion banner or create another overlapping chain model.

## Phase 1 — Audit the deployed system and its records

**Planning allowance:** 2–4 engineering days, dependent on cloud access. This is an audit budget, not a promised repair estimate.

1. Identify the deployed revision, runtime backend/project/database, producer versions and configured destinations. Inventory producers independently of received traces: coordinator lanes, messages, mission drivers, Codex, Claude Code, Gemini, motoko, evals, and provider Broadcast. Confirm whether the user's “modex” means Codex or another producer.
2. Establish a fixed UTC cohort: the latest seven complete days, plus older pending approvals and active chains. Segment around deployment changes. Enumerate all source rows with bounded pagination; stratify detailed inspection by node, harness, provider, trigger, success/failure/cancellation, retries, and tracing on/off. A cohort with no examples of a path does not certify that path.
3. Map every retained dashboard field to its API endpoint, CLI surface, authoritative record, producer, transformation and freshness. Classify each as verified, partial, broken, unsupported, intentionally uncaptured, or untested. Exercise filters, page boundaries, empty results, missing indexes and authorization errors.
4. Reconcile independent denominators: dispatch records versus stages; sent/received messages versus task links; completed tasks versus finalized chains; approval decisions versus actual execution; session/tool logs versus stored transcripts; provider usage versus attributed cost. Received rows alone cannot prove absent producers healthy.
5. Record counts and concrete redacted IDs for dangling references, duplicate logical runs, missing attempts, stale states, unmatched messages/spans, truncated transcripts, incomplete artifacts, and cost disagreement. Distinguish absent, delayed, retained-away, denied, and intentionally uncaptured data. Never infer a successful outcome from missing errors.
6. Audit the deployed UI using known records: initial load, filtering, list-to-detail drilldown, approvals, task outcome, transcript and function detail. Determine whether each gap originates at capture, transport, storage, query, contract, or rendering.
7. Inventory existing diagnostics before adding one. Use the existing reconciliation dry-run where appropriate; review backend construction for background writers before calling anything a read-only diagnostic. Keep repairs separate from measurement.

**Deliverables:** dated source/route inventory; field-level capability matrix; reproducible query manifest with windows, pagination and denominators; representative task dossiers; prioritized defect list with owner and regression criterion. Store summaries and redacted evidence in-repo; keep raw prompts, tokens and sensitive payloads out of audit artifacts.

**Exit gate:** every claimed working path has observed cloud evidence; every unknown is named. A short list of high-impact fixes replaces a general dashboard rewrite backlog.

## Phase 2 — Make the data contract reliable

**Sequence:** lifecycle and identity → capture/delivery → query parity → approvals → historical reconciliation.

Use the existing feature chain as the durable work identity. Keep distinct IDs for stages, execution attempts, sessions, messages, approvals, artifacts and spans; link them explicitly. Non-feature tasks and evals retain their meaningful source types. Repeated attempts must remain visible without becoming duplicate features. Cross-feature dependencies and handoffs are links, not forced parent-child nesting.

Define authority per fact: task lifecycle and approval records own operational state; observatory stores execution evidence; messages own request/handoff content; artifacts identify the delivered output and revision. A shared application service/query contract joins these facts for API and CLI. The UI must not rebuild identities, completion state or accounting from strings and timestamps.

Specify these behaviors before coding:

- Stable identity and versioned response schemas; explicit backend/source, applied filters, UTC window, cursor, as-of time and completeness.
- Explicit missing/unsupported/partial/stale/error states. Unknown cost differs from zero, and span health differs from verified task success.
- Idempotent ingestion and finalization; distinct retry attempts; out-of-order arrival and restart recovery. Reuse the existing spool/finalization primitives where they meet the contract.
- Durable baseline capture independent of model compliance with a skill instruction. Offline execution continues under the existing policy, while unsynced work is visibly pending with delivery/backlog health.
- Cost attribution names its authority and coverage. Provider Broadcast and executor telemetry for the same call must not double-count; metered, estimated, subscription and unknown costs remain distinguishable.
- Cloud API and remote CLI share semantics, filtering, pagination and errors. Offline local mode remains explicit. Extend current commands instead of inventing a parallel command family.
- Function tracing is an optional attachment. Record capture mode, sampling, truncation and retention so “tracing disabled” cannot appear as “no functions ran.” Retained source revision/package/module/function identity must resolve to the code that actually executed.

**Approval contract:** show the exact requested action, immutable artifact/revision or digest, evidence, requester, owning executor, scope and expiry. Persist authenticated reviewer and decision, then track dispatch and execution outcome separately. Test duplicate decisions, races, stale revisions, denial and offline owners. Approval is not delivery. Validate with isolated fixtures; do not approve real pending work merely to test the UI.

**Database decision:** retain current physical storage provisionally. Audit whether identity, atomicity, joins, access control and query cost can satisfy this contract. Consolidate physical storage only if measured constraints justify it; require migration checks, dual-read comparison and rollback. A single database alone does not fix missing capture or ambiguous authority.

**Acceptance gate before UI replacement:**

- All controlled lifecycle scenarios pass on SQLite and Firestore: success, failure, retry, cancellation, handoff, approval, concurrent sessions, offline/resync, late spans, and tracing on/off.
- API and remote CLI return identical identities, states, artifact references and totals for the same fixed cohort and filters.
- Every baseline required link is present in fixtures; no duplicate logical writes under replay. Unknown and partial data are explicit.
- Initial proposed production target: at least 99% of eligible baseline events linked within five minutes while online, with counts and reasons for the remainder; zero unexplained terminal-state contradictions. Calibrate thresholds to the measured producer envelope before sprint approval.
- Seven consecutive days of reconciliation show no unexplained drift. Historical exceptions are separated from new correctness.
- Relevant tests, lint/build and `make check-boundaries` pass; documentation describes the actual cloud deployment.

After new writes and reads are correct, propose a bounded historical repair with dry-run counts and evidence. Preserve raw records and mark unrecoverable history explicitly; do not fabricate transcripts, costs or successful outcomes.

## Phase 3 — Replace the dashboard with three focused surfaces

Design these against the proven API using real audit fixtures, including partial and failed examples:

1. **Work:** searchable fleet work list, filters for project/agent/state/time, current capture health, and one task/feature detail page. Detail shows request → attempts/handoffs → decisions → delivered artifact, with transcript and trace drilldown. An agent inventory/last-seen indicator makes missing producers visible.
2. **Approvals:** one actionable queue and decision history. Each item opens the same work detail and the exact proposed change. Clearly distinguish pending, decided, dispatched, executed, failed and expired.
3. **Functions:** inspect module/function at an exact source revision, its available contracts and execution evidence, inputs/outputs/effects when captured, and links back to invoking work. Start with provenance browsing; advanced replay/editing needs its own justified scope.

Messages and traces become linked evidence within work detail rather than competing explanations of the same activity. Prefer one chronological/tree representation with progressive disclosure. Keep standalone raw diagnostics reachable when they answer a real audit question.

For each current chart/view, record **keep, merge, defer, remove** and the user question it serves. Initial removal candidates are duplicate execution visualizations and analytics that lack cloud support or a necessary operational purpose; the live audit must identify the exact routes and consumers before deletion.

**Exit gate:** a human can locate a failed task and its cause, inspect an approval's exact change, and find delivered work through one coherent navigation path. Trace-off examples remain useful; trace-on examples reach function evidence. Every displayed field has a certified API source. Delete superseded UI after parity checks and a rollback-capable cutover.

## Delivery and decision points

- The preliminary source audit and initial production data audit are recorded in this plan and its linked live report. Full fleet producer census and browser interaction remain outstanding.
- Next deliverable is an amended provenance/data design driven by the live findings, starting with exporter health and a demonstrable fresh task. No UI implementation starts ahead of the data gate.
- Per repository routing: user approval of the concrete design → `sprint-planner` → user says “execute sprint” → `sprint-executor` → `sprint-evaluator`.
- Estimate repair milestones after the audit. Give each milestone a live acceptance demonstration and a named owner; do not mark it complete because local tests pass or a document moved directories.
- Keep a small repeatable cloud smoke/contract suite and a daily integrity report after cutover. Track link completeness, freshness, stranded work, approval execution, and query latency/read cost. Alert on gaps instead of presenting zero activity.

## Risks and boundaries

- Source code and deployed revision may differ: pin both before attributing a live symptom.
- Historical gaps may be unrecoverable: display coverage and retention boundaries honestly.
- Broad scans may be expensive: bound windows, paginate, measure reads, and avoid repeated aggregate scans.
- Global visibility must preserve authorization and redact sensitive inputs/results at the service boundary.
- Current uncommitted mission/benchmark work is unrelated and remains parked. This plan does not modify harness core or language semantics.

## Axiom review of the proposed direction

| Axiom | Score | Rationale |
|---|---|---|
| A1 Determinism | 0 | No language semantic change; deterministic reconciliation criteria required |
| A2 Replayability | +1 | Explicit attempts, evidence and source revisions |
| A3 Effect legibility | +1 | Captured effects and capture limits visible |
| A4 Explicit authority | +1 | Approval actor, scope and immutable target |
| A5 Bounded verification | +1 | Fixed cohorts and executable invariants |
| A6 Safe concurrency | +1 | Retry/race/session isolation acceptance cases |
| A7 Machines first | +1 | API/CLI contract precedes UI |
| A8 Minimal syntax | 0 | No new language syntax |
| A9 Cost visibility | +1 | Attributed costs with coverage and no duplicate accounting |
| A10 Composability | +1 | Reuse existing stores and provenance identities |
| A11 Structured failure | +1 | Unsupported/partial/error are distinct |
| A12 System boundary | +1 | Explicit producer, transport, storage and query responsibilities |

Net +10; no proposed hard violation of A1/A3/A4/A7. These are design intentions, not implementation certification.
