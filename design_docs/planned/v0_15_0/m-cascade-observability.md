# M-CASCADE-OBSERVABILITY: Surface Cascade AI Conversation History

**Status**: Planned
**Target**: v0.15.x
**Priority**: P1 (High — cascade is the highest-stakes AI surface; auditability is currently `gsutil cat`)
**Estimated**: 6-8 hours
**Dependencies**: M-CLOUD-OBSERVATORY (v0.13.0, planned/in-flight) — completion-handler → Observatory wire reuses its Backend interface fixes
**Author**: Claude + Mark
**Created**: 2026-05-03

---

## Executive Summary

The autonomous package cascade (M-PKG-AUTONOMOUS-CASCADE-SAFE) was validated end-to-end on 2026-05-03 against fixture packages: PR #14 in `sunholo-data/ailang-packages` (Class A deterministic, $0) and PR #15 (Class C AI repair, $0.058 / 12 turns / 11 tools / haiku model). Both fired autonomously.

**The problem:** the AI's full conversation IS recorded — `transcript.txt` (1.5 KB plaintext turn summary) and `session.jsonl` (86 KB Claude Code stream) sit in `gs://ailang-multivac-dev-ailang-artifacts/tasks/{taskID}/` for both PRs. **Nothing reads them back.** No CLI command, no dashboard tab, no API endpoint. The Cloud Run Job emitted `cost=$0.0584` in its stdout and the Firestore task doc has `artifact_gcs_path=tasks/task-51173ca7`, but the cloud dashboard's chain row for that exact task is stuck at `total_cost=0, total_tokens=0, total_turns=0, status=active` — completion metrics never reach the Observatory.

**Scope:** wire the existing recorded data through to where humans look. Five concrete fixes in this doc, ~250 LOC, no new storage backends.

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Surfaces the deterministic record of an AI run; doesn't add nondeterminism |
| A2: Replayability | +1 | The full session.jsonl IS a replay artifact; this exposes it |
| A3: Effect Legibility | 0 | No effect-system change |
| A4: Explicit Authority | +1 | Read-only fetcher uses ADC for GCS, no ambient auth |
| A5: Bounded Verification | 0 | Not a type-check change |
| A6: Safe Concurrency | 0 | No concurrency change |
| A7: Machines First | +2 | The whole point: machine-readable JSONL becomes machine-queryable. Reduces "ask a human to gsutil cat" tax |
| A8: Minimal Syntax | 0 | No new syntax |
| A9: Cost Visibility | +2 | Currently `total_cost=0` for $0.058 actual — fixing this IS cost visibility |
| A10: Composability | +1 | New `chains transcript` composes with existing `chains view` |
| A11: Structured Failure | +1 | If GCS is down or path empty, return typed error not blank screen |
| A12: System Boundary | +1 | GCS read is at a system boundary; we make it explicit |

**Net Score: +10** → **Decision: ✅ Proceed to implementation**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism — fetcher reads existing artifacts
- [x] A3 (Effects): No hidden side effects — read-only GCS access
- [x] A4 (Authority): No ambient access — ADC is per-request
- [x] A7 (Machines First): Strongly positive — this is the axiom most directly served

---

## Problem Statement

### Current State (verified 2026-05-03)

For task `task-51173ca7` (PR #15, the Class C AI repair):

| Layer | What's there | Visible to humans? |
|-------|--------------|---------|
| GCS `gs://…/tasks/task-51173ca7/transcript.txt` | 1.5 KB, 9 turns of reasoning | ✅ via `gsutil cat` |
| GCS `…/session.jsonl` | 86 KB Claude Code stream | ✅ via `gsutil cat` |
| GCS `…/metrics.json` | Real cost/turns/tokens | ✅ via `gsutil cat` |
| Firestore `tasks/task-51173ca7` doc | `artifact_gcs_path: "tasks/task-51173ca7"`, `tokens_used: 2785`, `session_id`, `cost`, `duration` | ⚠️ via raw Firestore API only |
| Firestore `tasks/task-51173ca7/events` subcollection | Empty `{}` | ❌ |
| Cloud dashboard chain `044934ea` | `status: active`, `total_cost: 0`, `total_tokens: 0`, `total_turns: 0`, `stage_count: 0` — **chain row exists but never updated on completion** | ❌ Wrong data |
| Cloud dashboard `/api/chains/{id}` | Returns 404 | ❌ Inconsistent (list returns it, detail doesn't) |
| Cloud dashboard `/api/chains/{id}/stages/{id}/chat` | Endpoint exists, reads from Observatory; Observatory has no spans for cascade tasks | ❌ Empty |
| `ailang chains view <task-id>` | Reads local SQLite only — no cloud awareness | ❌ Cloud-blind |

86 chains in the cloud dashboard, **all 86 stuck at `status=active` with zero metrics.**

### Impact

**Cascade is the highest-blast-radius AI surface in the system.** It rewrites code in `sunholo-data/ailang-packages` autonomously, opens PRs against `main`, and the only review gate is the human PR reviewer — who currently has no way to see what the AI actually did beyond reading the diff. That diff doesn't show:

- Which tools the AI tried first
- What error the deterministic check returned
- How many turns of reasoning it took
- Whether the AI considered alternatives
- The actual cost (matters for the per-package `max_cost_usd` budget caps just landed)

For a class C repair on a package with many consumers, this opacity is operationally unacceptable.

---

## Goals

**Primary Goal:** Make the AI's full work auditable from `ailang chains view <task-id>` and from the cloud dashboard, with zero new storage backends.

**Success Metrics:**
- `ailang chains transcript task-51173ca7` prints the 9-turn reasoning to terminal in <2s
- Cloud dashboard chain row shows `total_cost=$0.058, total_turns=12, total_tokens=2785, status=completed` for that task
- Dashboard "Transcript" tab renders the JSONL turn-by-turn
- `/api/chains/{id}` returns 200 (not 404) for chains that appear in `/api/chains` list
- A human can review a class C cascade PR in <5 min by reading the transcript without ever opening `gsutil`

---

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Where transcript content lives: GCS-only (current) vs copy-to-Firestore vs both | GCS = cheap/scales/lazy; Firestore = fast/queryable but storage cost. Wrong choice = re-architecting later | human | design | high |
| Whether dashboard server proxies GCS reads or issues signed URLs | Proxy = simpler auth (server SA reads, browser gets bytes); signed URL = cheaper bandwidth, exposes bucket name. Affects every transcript view | human | design | med |
| Completion-handler-Observatory wire: write directly to obsBackend during `pubsub_completion_handler` vs emit a span and let the OTLP receiver pick it up | Direct write = simpler, immediately consistent. Span = more uniform with rest of telemetry. Affects how task → chain stage linking works | human | design | high |
| Whether `ailang chains` (CLI) becomes cloud-aware or we ship a separate `ailang chains transcript` | One unified command vs scoped new command. Affects all future cloud-vs-local CLI duality | agent | design | med |
| Should we backfill the 86 stuck-at-active chains or accept they're orphaned data | One-time data hygiene. Either run a backfill job or document "pre-fix chains are incomplete" | human | runtime | low |

### Design Freeze

Before implementation begins, the three "high"-cost decisions must be resolved:

- [ ] Storage location for transcripts: **GCS-only with on-demand fetch (recommended)** vs copy-on-completion to Firestore
- [ ] Dashboard fetch path: **server proxy (recommended)** vs signed URL
- [ ] Completion → Observatory wire: **direct write in `pubsub_completion_handler.go` (recommended)** vs OTLP span emission

Recommendation rationale (for human review):
- **GCS-only**: 86 KB × 1000 cascades/year = 86 MB total. Negligible vs Firestore document costs. Fetch is one HTTP GET.
- **Server proxy**: Browser already has dashboard auth; bucket name stays internal; no signed-URL TTL footguns.
- **Direct write**: The completion handler already writes to Firestore (line 104 of `pubsub_completion_handler.go`). One more write to `obsBackend.UpdateChainCompletion(...)` is the smallest possible change.

---

## Solution Design

### Overview

Five focused fixes, all read-side or wiring (no new collections, no new backends):

1. **Wire completion metrics to Observatory chain.** When a Pub/Sub completion arrives, update the chain row's `status`, `total_cost`, `total_turns`, `total_tokens`, `stages_completed` in the Observatory backend (Firestore in cloud).
2. **Add CLI `ailang chains transcript <task-id>`.** Reads the task's `artifact_gcs_path` from Firestore (or local SQLite for local tasks), fetches `transcript.txt` from GCS, prints to stdout. `--jsonl` flag for the full session.jsonl.
3. **Add dashboard "Transcript" tab.** Calls a new `GET /api/tasks/{taskID}/transcript` endpoint that proxies the GCS read. UI renders turns + tool calls.
4. **Fix `/api/chains/{id}` 404.** Investigate why detail returns 404 while list returns the same chain — likely a Firestore vs in-memory store mismatch. Make the two read paths consistent.
5. **Optional: backfill the 86 active chains.** One-shot script that walks Firestore tasks where `status` is terminal and pushes completion-handler updates into Observatory.

### Architecture

```
┌─────────────────────────────────────────────────────────────┐
│  Cloud Run Job (executor)                                   │
│    ├─ Claude Code session                                   │
│    ├─ writes transcript.txt + session.jsonl + metrics.json  │
│    │   to /artifacts/tasks/{taskID}/  (GCS volume mount)    │
│    └─ publishes Completion to Pub/Sub                       │
└─────────────────────────────────────────────────────────────┘
                          │
                          ▼ Pub/Sub completion
┌─────────────────────────────────────────────────────────────┐
│  Coordinator (Cloud Run service)                            │
│    └─ pubsub_completion_handler.go                          │
│        ├─ Updates Firestore tasks/{taskID} (TODAY ✓)        │
│        └─ ★ NEW: Updates Observatory chain row              │
│             obsBackend.UpdateChainCompletion(chainID, ...)  │
└─────────────────────────────────────────────────────────────┘
                                       │
                                       ▼
                              ┌─────────────────────┐
                              │  Observatory store  │
                              │  (Firestore in cloud,│
                              │   SQLite local)     │
                              └─────────────────────┘
                                       │
                          ┌────────────┴────────────┐
                          ▼                         ▼
              ┌─────────────────────┐   ┌─────────────────────┐
              │ Dashboard server    │   │ ailang chains CLI   │
              │  (already queries   │   │  (already queries   │
              │   Observatory ✓)    │   │   SQLite ✓)         │
              └─────────────────────┘   └─────────────────────┘
                          │                         │
                          │ ★ NEW                   │ ★ NEW
                          ▼                         ▼
              ┌─────────────────────┐   ┌─────────────────────┐
              │ GET /api/tasks/{id} │   │ ailang chains       │
              │     /transcript     │   │   transcript <id>   │
              │   → proxies GCS     │   │ → reads             │
              └─────────────────────┘   │   artifact_gcs_path │
                          │             │ → fetches GCS       │
                          ▼             └─────────────────────┘
              ┌─────────────────────┐
              │ Dashboard UI        │
              │   "Transcript" tab  │
              └─────────────────────┘
```

### Components

1. **Completion → Observatory writer** (`internal/coordinator/pubsub_completion_handler.go`): one new call after the existing Firestore write
2. **Transcript fetcher** (`internal/storage/gcs/transcript.go`, new): single function `FetchTranscript(ctx, gcsPath, kind) ([]byte, error)` where `kind` ∈ {"transcript", "session"}
3. **CLI subcommand** (`cmd/ailang/chains_transcript.go`, new): wires fetcher + Firestore lookup
4. **Server endpoint** (`internal/server/handlers_artifacts.go`, new): `GET /api/tasks/{id}/transcript` — calls fetcher, streams response
5. **Dashboard UI tab** (`ui/src/components/TranscriptTab.tsx`, new): renders JSONL turns

### Implementation Plan

**Phase 1: Completion → Observatory wire** (~2 hours, P0 — all other phases benefit)

- [ ] Add `UpdateChainCompletion(ctx, chainID, status, cost, turns, tokens, completedAt)` to `observatory.Backend` interface
- [ ] Implement on `SQLiteBackend` (forwarding to Store) and Firestore `ObservatoryStore`
- [ ] In `pubsub_completion_handler.go` after the existing Firestore task update, call `obsBackend.UpdateChainCompletion(...)` using `task.ChainID` from the task record
- [ ] Idempotency: if `status` is already terminal, no-op (don't double-count)
- [ ] Verify against PR #14 / PR #15: re-fire a completion and check `/api/chains` shows real metrics

**Phase 2: GCS transcript fetcher + CLI** (~2 hours)

- [ ] New file `internal/storage/gcs/transcript.go` — wraps `cloud.google.com/go/storage` client, takes a relative path like `tasks/task-51173ca7`, joins with the artifact bucket env var, returns bytes
- [ ] Bucket env var: `AILANG_ARTIFACT_BUCKET` (already used by Cloud Run Job — reuse the same name)
- [ ] New file `cmd/ailang/chains_transcript.go`:
  - `ailang chains transcript <task-id>` → fetch `transcript.txt`, print
  - `ailang chains transcript <task-id> --jsonl` → fetch `session.jsonl`, print
  - `ailang chains transcript <task-id> --jsonl --format pretty` → render JSONL turns with rich formatting (defer pretty render to Phase 3 if scope creeps)
- [ ] Wire into `cmd/ailang/chains.go` subcommand dispatch
- [ ] Help text + `chains --help` includes the new subcommand

**Phase 3: Server endpoint + dashboard UI tab** (~3 hours)

- [ ] New file `internal/server/handlers_artifacts.go`:
  - `GET /api/tasks/{taskID}/transcript` — fetches `transcript.txt`, returns text/plain
  - `GET /api/tasks/{taskID}/session` — fetches `session.jsonl`, returns application/x-ndjson
  - Auth: same middleware as `/api/chains`
- [ ] Wire into `internal/server/server.go` mux
- [ ] Add `TranscriptTab.tsx` that fetches `/api/tasks/{id}/session`, parses JSONL, renders turn-by-turn (assistant text, tool calls with their inputs/outputs collapsed by default)
- [ ] Add tab to existing chain detail page next to "Spans" / "Chat" tabs

**Phase 4: Fix `/api/chains/{id}` 404** (~1 hour)

- [ ] Reproduce: `curl https://…/api/chains/044934ea-…` → 404 even though `curl https://…/api/chains` lists it
- [ ] Investigate: is the list endpoint reading from a cache layer that the detail endpoint doesn't see? Or is the detail endpoint asserting on a backend type? Likely related to M-CLOUD-OBSERVATORY's "type-assertion to *SQLiteBackend" pattern
- [ ] Fix: route detail endpoint through `Backend` interface
- [ ] Test: chain detail returns 200 for any chain the list endpoint shows

**Phase 5 (Optional): Backfill** (~1 hour)

- [ ] One-shot script `scripts/backfill_chain_completions.go`:
  - For each chain in Observatory with `status=active` and no recent updates
  - Look up the corresponding task in Firestore
  - If task is in terminal state, call `UpdateChainCompletion(...)` with its metrics
- [ ] Run once against `ailang-multivac-dev` to fix the 86 stuck chains
- [ ] Document outcome; do NOT make this a recurring job (the wire-up in Phase 1 prevents the problem going forward)

### Files to Modify/Create

**New files:**
- `internal/storage/gcs/transcript.go` — GCS read wrapper, ~80 LOC
- `cmd/ailang/chains_transcript.go` — CLI subcommand, ~120 LOC
- `internal/server/handlers_artifacts.go` — HTTP handlers, ~80 LOC
- `ui/src/components/TranscriptTab.tsx` — React JSONL renderer, ~150 LOC
- `scripts/backfill_chain_completions.go` — one-shot backfill, ~100 LOC (optional)

**Modified files:**
- `internal/observatory/backend.go` — add `UpdateChainCompletion` method, ~5 LOC
- `internal/observatory/backend_sqlite.go` — forwarding method, ~10 LOC
- `internal/storage/firestore/observatory_chains.go` — Firestore impl, ~30 LOC
- `internal/coordinator/pubsub_completion_handler.go` — one new call after existing write, ~10 LOC
- `cmd/ailang/chains.go` — register new subcommand, ~5 LOC
- `internal/server/server.go` — mount new endpoints, ~5 LOC
- `ui/src/components/ChainDetail.tsx` (or equivalent) — add Transcript tab, ~10 LOC

**Total: ~600 LOC across 12 files** (5 new, 7 modified). Phases 1–4 = ~430 LOC; Phase 5 backfill = +100 LOC.

---

## Examples

### Example 1: CLI transcript review

**Before (today):**
```bash
$ ailang chains view task-51173ca7
Error: no chain found with prefix 'task-51173ca7'
# (CLI is local-only, doesn't see cloud cascade tasks)
$ gsutil cat gs://ailang-multivac-dev-ailang-artifacts/tasks/task-51173ca7/transcript.txt
# Manual GCS dive
```

**After:**
```bash
$ ailang chains transcript task-51173ca7
[TURN 1]
I'm the AILANG cascade repair agent. Let me start by examining the current state...
[TOOL] Read packages/test-pkg-consumer/wrap.ail
[TOOL] Read packages/test-pkg-consumer/ailang.toml
[TOOL] Bash ailang check --package .

[TURN 2]
[TOOL] Read packages/test-pkg/hello.ail
...
[TURN 9]
Perfect! The cascade repair is complete.

✓ 12 turns, 11 tool calls, $0.0584 cost, haiku model
```

### Example 2: Dashboard chain row

**Before (today):**
```
Chain 044934ea-c975-...  (msg_20260503_003449_51173ca7)
status: active
total_cost: $0
total_tokens: 0
total_turns: 0
stages_completed: 0
```

**After:**
```
Chain 044934ea-c975-...  (msg_20260503_003449_51173ca7)
status: completed
total_cost: $0.058
total_tokens: 2,785
total_turns: 12
stages_completed: 1
[Spans] [Chat] [Transcript ←NEW]
```

---

## Success Criteria

- [ ] `ailang chains transcript task-51173ca7` returns the full transcript in <2s with `--jsonl` flag for raw stream
- [ ] Cloud dashboard `/api/chains` shows real `total_cost`, `total_turns`, `total_tokens`, `status=completed` for tasks that completed (verified against PR #15's task)
- [ ] Cloud dashboard `/api/chains/{id}` returns 200 for any chain the list endpoint includes
- [ ] Dashboard "Transcript" tab renders the JSONL turn-by-turn with tool call details collapsed by default
- [ ] `GET /api/tasks/{taskID}/transcript` returns `transcript.txt` with `text/plain` content type
- [ ] `GET /api/tasks/{taskID}/session` returns `session.jsonl` with `application/x-ndjson` content type
- [ ] Local SQLite path unchanged for non-cloud tasks (regression-free)
- [ ] All `make test` passing
- [ ] Documentation: `docs/docs/guides/coordinator.md` includes a "Reviewing cascade AI work" section

---

## Testing Strategy

**Unit tests:**
- `gcs.FetchTranscript` against a fake GCS server (existing `httptest` patterns)
- `UpdateChainCompletion` for SQLite and Firestore backends with idempotency check
- CLI subcommand argument parsing

**Integration tests:**
- End-to-end: publish a fixture cascade message → wait for completion → verify Observatory chain row updated → verify `chains transcript` returns expected text
- Reuse the test_pkg / test_pkg_consumer fixtures (already validated in PR #14, #15)

**Manual testing:**
- Re-run the Class C cascade smoke (publish test_pkg@0.1.4 with another signature change) and verify the dashboard "Transcript" tab renders correctly
- Try `ailang chains transcript` against the existing 86 cloud chains to confirm fetch works for historical data

---

## Deferred Decisions

The following are intentionally left open for the implementer:

- **JSONL → React rendering format**: collapsed-by-default tool call display, which fields to show (full input vs summary), syntax highlighting library — agent may choose
- **Pagination on `chains transcript --jsonl`**: do we stream or buffer? — agent may choose; default to buffer until size becomes an issue
- **Caching layer for `/api/tasks/{id}/transcript`**: ETag headers, response caching — agent may add if needed; not required for v1
- **Whether `ailang chains view` (existing command) gets a new column for cloud tasks**: agent decides whether to extend or keep cloud-aware path scoped to `chains transcript`

---

## Non-Goals

**Not attempted in this feature:**
- **Real-time streaming of in-progress turns**: the AI's stdout is captured to GCS only at completion. Streaming would require a different transport (Pub/Sub events topic, server-sent events). Out of scope; tracked separately under M-CLOUD-PROGRESS-TRACKING.
- **Indexing transcripts for search ("find all cascade tasks where the AI used Edit on wrap.ail")**: would need a search backend (BigQuery, Elasticsearch). Defer to M-OBS-SEARCH or similar.
- **Cost dashboards / aggregate trends**: the existing `/api/chains/stats` endpoint covers this once Phase 1 lands. No new cost UI.
- **Replacing the existing chat endpoint**: `/api/chains/{id}/stages/{id}/chat` (which reads OTEL spans) stays. The new transcript endpoint is parallel — chat = OTEL, transcript = GCS artifacts.
- **Chat streaming via OTEL spans for cascade tasks**: would require Cloud Run Job to emit OTEL spans during execution. Possible but not required given GCS captures the same data — OTEL parity for cascade tasks is a separate milestone.

---

## Timeline

**Day 1** (~3 hours):
- Phase 1: Completion → Observatory wire + tests
- Smoke test against PR #15's existing data (re-fire the completion to populate)

**Day 2** (~3 hours):
- Phase 2: GCS fetcher + CLI subcommand
- Phase 4: Fix `/api/chains/{id}` 404

**Day 3** (~3 hours):
- Phase 3: Server endpoint + dashboard UI tab
- Documentation updates

**Optional Day 4** (~1 hour):
- Phase 5: Backfill script run once against dev

**Total: ~9 hours across 3-4 days**

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| GCS read latency dominates user experience | Medium | Fetcher uses streaming response; transcript.txt is 1.5 KB so latency is single-digit ms. session.jsonl is 86 KB — also fine. Add timeout (5s) and graceful error if bucket missing |
| Observatory backend interface expansion conflicts with M-CLOUD-OBSERVATORY (already adding 3 methods there) | Low | Coordinate: this doc adds a 4th method (`UpdateChainCompletion`). Same pattern. If M-CLOUD-OBSERVATORY ships first, rebase trivially |
| The `/api/chains/{id}` 404 is a deeper architectural issue (not a quick fix) | Medium | Time-box Phase 4 to 1 hour. If deeper, ship Phases 1-3 separately and split the 404 fix to a follow-up doc |
| Browser auth for new `/api/tasks/{id}/transcript` endpoint differs from `/api/chains` | Low | Reuse existing auth middleware. If the existing middleware is missing on /api/tasks routes, the test plan catches it |
| Backfill script causes a write storm in Firestore | Low | Phase 5 is optional and one-shot. Use Firestore batch writer with rate limit. Run at low traffic time |
| `transcript.txt` format changes upstream and parsers break | Medium | The format is plain text "[TURN N]" + "[TOOL] X" — stable since M-CLOUD-PROGRESS-TRACKING. Parser is forgiving (renders as plain text on unknown markers) |

---

## Related Documents

**Implemented (informs design):**
- [M-CLOUD-PROGRESS-TRACKING (v0.9.2)](../../implemented/v0_9_2/m-cloud-progress-tracking.md) — Producer-side: where transcript.txt and session.jsonl are written by the Cloud Run Job. This doc is the consumer-side counterpart
- [M-TASK-HIERARCHY-LINKING (v0.7.0)](../../implemented/v0_7_0/m-task-hierarchy-linking.md) — How tasks link to chains via `chain_id` attribute (we reuse this for the completion update)
- [M-CONTROL-PLANE-V4-INTEGRATION (v0.7.0)](../../implemented/v0_7_0/m-control-plane-v4-integration.md) — Dashboard control-plane patterns

**Planned (check for overlap):**
- [M-CLOUD-OBSERVATORY (v0.13.0)](../v0_13_0/m-cloud-observatory.md) — **Strong overlap on Backend interface expansion.** Coordinate: this doc adds `UpdateChainCompletion`; that doc adds `GetExecTaskHierarchyWithMessages`, `GetSpanHierarchy`, `GetToolsByTimestampRange`. Same pattern (interface method + SQLite forwarder + Firestore impl). If M-CLOUD-OBSERVATORY ships first, this doc rebases cleanly
- [M-DASHBOARD-SIMPLIFICATION (v0.13.0)](../v0_13_0/m-dashboard-simplification.md) — Look at where the new "Transcript" tab fits in the simplified UI
- [M-OBS-CONFIGURABLE-SPAN-FILTERING (v0.13.0)](../v0_13_0/m-obs-configurable-span-filtering.md) — Less overlap; this doc adds artifact reads, not span filtering

**Cascade context (this is what we're adding observability TO):**
- [M-PKG-AUTONOMOUS-CASCADE-SAFE](../../implemented/v0_12_x/m-pkg-autonomous-cascade-safe.md) — The cascade infra
- [M-PKG-CASCADE-DETERMINISTIC-FIRST](../../implemented/v0_13_x/m-pkg-cascade-deterministic-first.md) — Class A/B/C taxonomy and deterministic-vs-AI dispatch

---

## References

- [Design Axioms](/docs/references/axioms) — The 12 non-negotiable principles
- [Validation reference: PR #14](https://github.com/sunholo-data/ailang-packages/pull/14) — Class A deterministic cascade, $0
- [Validation reference: PR #15](https://github.com/sunholo-data/ailang-packages/pull/15) — Class C AI repair, $0.058 / 12 turns / 11 tools / haiku
- GCS bucket: `gs://ailang-multivac-dev-ailang-artifacts/tasks/{taskID}/`
- Coordinator: `https://ailang-dev-coordinator-ejjw6zt3bq-ew.a.run.app`
- Dashboard: `https://ailang-dev-dashboard-ejjw6zt3bq-ew.a.run.app`

---

## Future Work

- **OTEL parity for cascade tasks**: have the Cloud Run Job emit OTEL spans during execution so the existing `/api/chains/{id}/stages/{id}/chat` endpoint works for cascade. Removes the GCS-vs-Observatory split. Estimated 1-2 days; track as M-CASCADE-OTEL-PARITY
- **Searchable transcripts**: indexed search over historical cascade transcripts ("find all cascades where the AI used `git reset`"). Needs a search backend. Track as M-OBS-SEARCH
- **Cost-anomaly alerting**: when a cascade task exceeds 80% of `max_cost_usd`, fire a notification. Builds on Phase 1 metrics being correctly populated. Track as M-CASCADE-BUDGET-ALERTS

---

**Document created**: 2026-05-03
**Last updated**: 2026-05-03

---

DESIGN_DOC_PATH: design_docs/planned/v0_15_0/m-cascade-observability.md
