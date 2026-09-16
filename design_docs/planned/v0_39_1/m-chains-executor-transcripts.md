# M-CHAINS-EXECUTOR-TRANSCRIPTS — Executor transcripts into `ailang chains chat` (pi, codex, opencode — cloud and rig)

**Status**: Planned
**Target**: v0.39.1
**Priority**: P1
**Estimated**: ~1 day / 500-700 LOC
**Dependencies**: M-CHAINS-SOURCE-OF-TRUTH (implemented v0.7.2 — `chains chat` readers already exist); M-COMPLETION-PATH-PARITY (planned — idempotent finalize-write precedent this doc reuses)

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

Repository tooling (executor/observatory plane), not language surface. Scores on the axioms that concern agent workflow and data:

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No language/runtime impact |
| A2: Replayability | +1 | A banked per-turn transcript is what makes an agent run auditable and re-diagnosable; today the only record of a 456-turn fleet run is `claude-stream:` text lines in Cloud Logging |
| A3: Effect Legibility | 0 | No effect system change |
| A4: Explicit Authority | 0 | Option (a) of M3 adds a NEW, explicit, scoped SA grant (Firestore write) — documented in terraform, not ambient. Flagged in Risks |
| A5: Bounded Verification | 0 | No verification surface change |
| A6: Safe Concurrency | 0 | No new concurrent writers to a row (idempotent doc/row-keyed writes) |
| A7: Machines First | +1 | The whole point: machine-readable `chat_messages` rows replace grepping `claude-stream:` log lines — the exact anti-pattern CLAUDE.md §1 documents ("Asking what the agent DID and inferring the rest…") |
| A8: Minimal Syntax | 0 | No syntax |
| A9: Cost Visibility | +1 | Per-turn tokens_in/out (+cache) banked per message make per-turn cost attribution visible, not just run totals |
| A10: Composability | +1 | Reuses the existing EventHandler decorator pattern (RawStreamLineHandler/MetricsHandler) and the existing observatory Backend; no new read path — `chains chat`, the dashboard route, and chains_data.go all read by session/task id already |
| A11: Structured Failure | +1 | Write is idempotent (replayed Pub/Sub completions safe, M-COMPLETION-PATH-PARITY precedent); missing per-turn usage banks 0 ("not reported"), never invents a number |
| A12: System Boundary | 0 | No boundary change |

**Net Score: +5** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): no implicit nondeterminism introduced
- [x] A3 (Effects): no hidden side effects
- [x] A4 (Authority): the only new authority is the M3(a) Firestore grant, if chosen — explicit and scoped, decided by human
- [x] A7 (Machines First): the feature *is* machine-first data

### Decision Thresholds

| Net Score | Decision |
|-----------|----------|
| ≥ +2 | ✅ Proceed to implementation |
| 0 to +1 | ⚠️ Needs stronger justification |
| < 0 | ❌ Reject or redesign |
| Any −1 on A1/A3/A4/A7 | ❌ Automatic rejection |

## Problem Statement

`ailang chains chat <chain>` reads `chat_messages` by the stage's session id (cmd/ailang/chains_chat.go:87) or by task id (chains_data.go:176), but **almost nothing writes that table on the paths the fleet actually runs**.

**Current State (measured 2026-09-16 on chain 687d6ebc, task `inbox_1789566884640_60a2d39c` — reported measurement from the task brief; not re-measured in this session):**

- Exactly two chat writers exist in non-test code, both raw SQL INSERTs into **local SQLite** (V4): `internal/observatory/importer_motoko.go:253,282` (motoko session JSONL, via `chains import-motoko`) and `internal/claudehistory/importer.go:215` (Claude Code session JSONL).
- The pi executor **never sets `Result.Transcript`** (V5); opencode **never sets it either** (V6) — so the cloud job's `transcript.txt` artifact (`writeTaskArtifacts`, coordinator_cloud_github.go:509) is empty for every pi/opencode run. Only claude and codex populate it today.
- The cloud execute-job **never touches the observatory**: it publishes a `pubsub.TaskCompletion` (which has **no transcript field**, V7) and writes GCS artifacts. Its only transcript record is `claude-stream:` stderr lines in Cloud Logging (coordinator_cloud_executor.go:173-347) — rate-limited, truncated to 200-500 chars per event.
- `observatory.Backend` has **no chat write method** (V3) — only `GetChatMessagesByTaskID`/`GetChatMessagesBySession`/`CountChatMessages` — and the Firestore observatory store has **readers on `obs_chat_messages` but no writer** (V8).
- Result: every pi/codex/opencode run on the cloud fleet — including the new ailang_only lane (agents `ailang-only-executor`, `pkg-sunholo-test-pkg`, `pkg-sunholo-email`, `pkg-sunholo-ailang-parse`) — has its transcript only in Cloud Logging as `claude-stream:` lines. Opencode additionally has **no chat-transcript importer at all** (V16), so not even a manual import path exists.

**Impact:** every CLAUDE.md §1 workflow ("What was the agent actually told/did?") is dead for the entire cloud fleet. A 456-turn run is greppable only as log text. Post-hoc failure analysis of the ailang_only lane — the lanes with the most novel failure modes — has no turn-level substrate.

## Goals

**Primary Goal:** Every executor run (pi first; codex/opencode/claude gain it for free through the same recorder) produces per-turn `chat_messages` rows that `ailang chains chat`, the dashboard, and `chains data` can read — locally on rig and in the prod Firestore observatory for cloud runs.

**Success Metrics:**
- `ailang chains chat <cloud-chain-id> --stage N` shows real turns for a pi-run cloud chain (today: "No chat found").
- A killed cloud job still leaves the turns it completed (M3(a) property).
- A 456-turn run's banked transcript stays bounded (tool results capped at 8KB; measured doc size per turn ≪ Firestore 1MB limit).
- Per-turn token rows are non-zero for pi and match the Result contract (`tokens_out` disjoint from reasoning, V11).

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| D1: M3 cloud transport — (a) job writes Firestore directly / (b) JSONL→GCS artifact + daemon import / (c) inline in Pub/Sub payload | Determines job SA authority surface, kill-durability, and which repo (this one vs ailang-multivac terraform) changes | **human (Mark)** | design (before sprint) | high |
| D2: Record per-turn **reasoning tokens** as a new `reason_tokens` column (migrate_v22) vs folding into `content_json` metadata vs dropping | Banked-data schema change; quorum trigger 3 fires either way | human (Mark), with quorum | design | med |
| D3: Recorder's message type lives in `internal/executor` (`executor.ChatTurn`), NOT `[]observatory.ChatMessage` | Enforced by the compiler: `internal/observatory` already imports `internal/executor` (cost_classify.go:3), so executor→observatory is an import cycle (V12). The task brief's proposed shape ("producing []observatory.ChatMessage") would not compile | compiler | design (fixed) | low |
| D4: Per-turn usage delivered via a NEW optional handler interface (`TurnUsageHandler`), not by extending `OnTurnEnd`'s signature | Extending `OnTurnEnd(turnNum int)` breaks every existing EventHandler implementer; optional interfaces are the established pattern (RawStreamLineHandler, MetricsHandler — executor.go:481, 497) | agent | design | med |
| D5: `Backend.PutChatMessages` becomes the canonical chat write; existing importers migrate in a follow-up, not this sprint | Third chat writer: unbounded writer sprawl vs sprint scope (the claudehistory importer writes `content_text`/`content_thinking`/`request_id`/`stage_id`, columns the `ChatMessage` struct does not carry — migration needs struct work first, V18) | human (scope OK) | design | low |
| D6: Firestore `obs_chat_messages` gets an `expire_at` + TTL policy (30 days, matching local retention) | Without it the prod observatory accumulates chat rows forever (spans already do this via `spanTTL`, V10) | human (infra: terraform/gcloud in ailang-multivac) | design | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [ ] D1 — Mark picks the M3 transport (recommendation: (a), see Solution Design; verify job SA Firestore grant first)
- [ ] D2 — Mark approves (or rejects) the `reason_tokens` column; if rejected, D2 falls back to content_json metadata and the doc is updated
- [ ] D5 — Mark approves deferring importer migration to a follow-up

## Solution Design

### Overview

Three milestones, each independently shippable:

1. **M1 — `executor.TranscriptRecorder`**: an EventHandler decorator, executor-agnostic, that accumulates per-turn `executor.ChatTurn` records from the five stream events every CLI executor already drives; pi additionally drives a new optional `TurnUsageHandler` interface from its per-message usage events; pi sets `Result.Transcript`.
2. **M2 — `Backend.PutChatMessages`**: one canonical write method with SQLite + Firestore implementations; the rig coordinator daemon writes chat rows on task completion alongside its existing stage-metrics write.
3. **M3 — cloud transport**: get cloud-run turns into the prod Firestore observatory. **Decision for Mark**: (a) job writes `obs_chat_messages` directly as it runs — **recommended**; (b) chat JSONL to the existing GCS artifact mount + daemon import (the `chains import-motoko` pattern); (c) inline in the Pub/Sub payload (rejected, see below).

### Architecture

**M1 — the recorder (new file `internal/executor/transcript.go`, ~200 LOC):**

- `ChatTurn` (executor-local type — D3): `TurnNumber, Role, ContentJSON string, TokensIn, TokensOut, ReasonTokens, CacheRead, CacheCreation int, Model string, Timestamp time.Time`. `ContentJSON` is the same content-blocks array shape the motoko/claudehistory importers already write (`[{type:"text",...},{type:"tool_use",...}]` / `[{type:"tool_result",...}]`), so readers need zero changes.
- `TranscriptRecorder` implements `EventHandler` and *wraps* an inner handler (decorator): every event is forwarded unchanged, then recorded. `OnTurnStart` opens an assistant turn; `OnText` appends a text block; `OnToolUse` appends a `tool_use` block; `OnToolResult` closes into a following user turn with a `tool_result` block; `OnTurnEnd` closes the assistant turn. `Recorder() []ChatTurn` returns the frozen slice.
- **Caps (D4-adjacent, agent-tunable constants):** tool results capped at **8KB** (`maxToolResultBytes = 8192`, truncate + `…[truncated]` marker), assistant text capped at 64KB per turn. A 456-turn run with ~2 tool results per turn is bounded at roughly 456 × ~24KB ≈ 11MB in-memory worst case — fine locally, and per-turn docs stay far under the Firestore 1MB document limit (V17).
- **Per-turn tokens**: the base `EventHandler` carries no usage (V15), so M1 adds `TurnUsageHandler` (optional interface, D4):

  ```go
  type TurnUsage struct { Input, Output, Reasoning, CacheRead, CacheWrite int }
  type TurnUsageHandler interface { OnTurnUsage(turn int, u TurnUsage) }
  ```

  pi drives it from its existing `message_end` usage block (pi.go:370-380), applying **Result-contract semantics**: `Output - Reasoning` for tokens_out (D5 comment in pi.go — reasoning is inside output on the wire and the contract wants them disjoint, V11), reasoning reported separately for D2. Executors that don't implement/drive it bank 0s — schema-legal ("not reported"), never invented (no silent fallback).
- **pi sets `Result.Transcript`**: pi already builds `transcriptBuf` (pi.go:208) but only uses it as `Output`; wire it into `Result.Transcript` so the `transcript.txt` GCS artifact stops being empty (V5). Also make pi's executor report its turns through the recorder path in the same change.

**M2 — the canonical write path:**

- `Backend.PutChatMessages(ctx, msgs []*ChatMessage) error` added to the Backend interface (backend.go, after `CountChatMessages`), implemented by:
  - **SQLite** (`store_chat.go`): `INSERT OR REPLACE` keyed by `id` (uuid per message) inside one transaction — idempotent under replay.
  - **Firestore** (`internal/storage/firestore/observatory_chains_helpers.go`, new `PutChatMessages`): per-message document in `collObsChatMessages` via `s.client.Doc(collObsChatMessages, msg.ID).Set(...)` — same convention as `CreateSpan` (observatory_spans.go:25) — including `expire_at` from a `chatTTL` (30d, matching `DefaultSpanTTL`'s pattern, D6). The reader (`mapToChatMessage`) already exists; it needs a `reason_tokens` field added if D2 approves the column.
- **Correlation**: rows carry `session_id` (the executor-reported session id — for pi this is `ev.SessionID` from the stream, pi.go:283, which is exactly what the completion's `SessionID` carries and what `task_finalize.go:288` binds to the stage via `UpdateStageSession` — so `chains chat --stage N` joins without new plumbing, V13/V14) plus `task_id`. `chain_id` is left empty in the cloud path (readers use `GetChatMessagesByTaskID`, the preferred deterministic query per M-DETERMINISTIC-CHAT-LINKING); the rig daemon fills it where it already holds it.
- **Rig daemon (local completion path)**: the coordinator daemon already writes stage metrics on completion with `obsBackend` in hand (task_finalize.go; daemon_tasks_chain.go). In the same place, convert `Result`'s recorded turns → `PutChatMessages`. The local coordinator wraps its executor handler in `TranscriptRecorder` where it builds the EventHandler today (internal/coordinator/provider_executor.go).

**M3 — cloud transport (D1, decision for Mark):**

| Option | Mechanism | Needs | Killed job | Verdict |
|---|---|---|---|---|
| **(a) — recommended** | Job wraps `cloudEventHandler` in `TranscriptRecorder` and writes Firestore `obs_chat_messages` incrementally (batch per completed turn) using the existing Firestore observatory client (`AILANG_CLOUD_PROJECT` is already resolved in the job for Pub/Sub, coordinator_cloud_executor.go:104) | **New SA grant**: `datastore.documents.create/update` on the job's service account — today the execute-job binary has NO Firestore usage at all (V9); the grant lives in ailang-multivac terraform (`cloud_run_jobs.tf`), which must be checked first — **open verification, see Verification Log V9/Open item O1**. Plus D6 TTL policy. | Turns written so far persist — the durable-by-construction property | Cleanest read path; observatory stays source of truth; no new import step. Cost: one terraform change |
| (b) | `writeTaskArtifacts` additionally writes `chat.jsonl` (rows from the recorder) to the existing GCS artifact mount; the rig daemon, which already receives `ArtifactGCSPath` on the completion (topics.go:112, pubsub_completion_handler.go:125), downloads and calls `PutChatMessages` — the `chains import-motoko` pattern | Nothing new permission-wise (mount is read-write today); daemon needs a GCS read client for the artifact bucket | Full transcript survives (written at job end) — but a hard kill before artifact write loses everything | No new authority, but adds a download+import step that can silently not-run; transcript visibility depends on daemon liveness |
| (c) | Inline rows in the Pub/Sub `TaskCompletion` payload | Pub/Sub max message 10MB (documented GCP limit); a capped 456-turn run approaches it; every oversized publish silently fails the whole completion | N/A | **Rejected**: fragile at the limit and couples completion delivery to transcript bulk |

Recommendation **(a)** because a transcript banked only-if-delivered is the same silent-loss shape the `Summary` field was added to fix (topics.go:89-101: "a completion that reached the message plane carried an empty error_msg… and nothing else"); (b) is the fallback if the SA grant is blocked, and is also the natural shape for a later "re-historicize old runs from artifacts" tool.

### Implementation Plan

**Phase 1: M1 — recorder + pi (~4 hours)**
- [ ] `internal/executor/transcript.go`: `ChatTurn`, `TurnUsage`, `TurnUsageHandler`, `TranscriptRecorder` (+ unit tests with a fake handler)
- [ ] pi.go: drive `OnTurnUsage` from `message_end` usage deltas; set `Result.Transcript` from `transcriptBuf`
- [ ] Wire `TranscriptRecorder` as a decorator in the coordinator's local executor path (provider_executor.go)

**Phase 2: M2 — write path + rig (~3 hours)**
- [ ] `Backend.PutChatMessages` on the interface; SQLite impl (INSERT OR REPLACE, one tx); Firestore impl (per-message doc, `expire_at`)
- [ ] migrate_v22 (only if D2 approved): `ALTER TABLE chat_messages ADD COLUMN reason_tokens INTEGER DEFAULT 0`; extend `ChatMessage` struct + SQLite SELECT/scan + `mapToChatMessage`
- [ ] Daemon completion path writes turns via `PutChatMessages` next to the existing stage-metrics write
- [ ] Interface-implementation audit note: exactly two `observatory.Backend` implementations exist today (SQLiteBackend, Firestore ObservatoryStore) and both get real writers — the old backend_gcp/backend_jaeger stubs named in M-CHAINS-SOURCE-OF-TRUTH no longer exist in the tree (verified by grep of implementers)

**Phase 3: M3 — cloud (~3 hours + infra)**
- [ ] **O1 (first sprint task, gates (a))**: verify the job SA's Firestore grant in `ailang-multivac/terraform/cloud_run_jobs.tf` + `gcloud projects get-iam-policy`; if absent, add the grant (human/infra) or fall back to (b)
- [ ] (a): wrap `cloudEventHandler` in the recorder; construct a Firestore observatory client in the job; batch-write turns on `OnTurnEnd`
- [ ] D6: create the `obs_chat_messages` TTL policy (terraform/gcloud, human)
- [ ] End-to-end: dispatch one cloud pi task; `ailang chains chat <id> --remote gcp` shows turns

### Files to Modify/Create

**New files:**
- `internal/executor/transcript.go` — ChatTurn, TurnUsage, TurnUsageHandler, TranscriptRecorder, ~200 LOC
- `internal/executor/transcript_test.go` — fake-handler replay tests, cap tests, ~250 LOC
- (D2 only) `internal/observatory/migrate_v22.go` + test — reason_tokens column, ~60 LOC

**Modified files:**
- `internal/executor/pi/pi.go` — drive OnTurnUsage; set Result.Transcript (~20 LOC)
- `internal/executor/executor.go` — nothing (optional interface lives in transcript.go)
- `internal/observatory/backend.go` — `PutChatMessages` signature (~5 LOC)
- `internal/observatory/store_chat.go` — SQLite PutChatMessages + (D2) reason_tokens in SELECT (~60 LOC)
- `internal/storage/firestore/observatory_chains_helpers.go` — Firestore PutChatMessages + mapToChatMessage reason_tokens (~50 LOC)
- `internal/coordinator/provider_executor.go` — wrap local handler in recorder (~10 LOC)
- `internal/coordinator/task_finalize.go` or `daemon_tasks_chain.go` — PutChatMessages on completion (~30 LOC)
- `cmd/ailang/coordinator_cloud_executor.go` — wrap cloudEventHandler; (a) Firestore writer (~40 LOC)
- `cmd/ailang/coordinator_cloud_github.go` — (b fallback only) chat.jsonl artifact

## Examples

### Example 1: cloud pi run becomes readable

**Before:**
```
$ ailang chains chat 687d6ebc --stage 1 --remote gcp
No chat found for this chain.   # transcript lives only in Cloud Logging 'claude-stream:' lines
```

**After:**
```
$ ailang chains chat 687d6ebc --stage 1 --remote gcp
Stage 1: pi executor session a1b2c3...
Turn 1 [assistant] (tokens 17213→412, model glm-4.7)
  Reading the task... [tool_use: read]
  [tool_result: file content (8KB cap)] ...
Turn 2 [assistant] (tokens 18120→987, reasoning 402)
  ...
```

### Example 2: daemon replay is safe

A Pub/Sub completion delivered twice (at-least-once) re-runs finalize; `PutChatMessages` is `INSERT OR REPLACE` keyed by message id (SQLite) / `Doc(id).Set` (Firestore) — identical bytes, second apply a no-op. Same guarantee class as `SetStageStatus`/`SetStageMetrics` (M-COMPLETION-PATH-PARITY M0b).

## Success Criteria

- [ ] `ailang chains chat <cloud-pi-chain> --remote gcp --stage N` returns non-empty turns (acceptance: one dispatched cloud task, checked live)
- [ ] `ailang chains chat <local-rig-chain>` returns turns for a local coordinator pi run (acceptance: `ailang coordinator start` + one local dispatch)
- [ ] Killed-job durability: a task cancelled mid-run leaves completed turns in `obs_chat_messages` (acceptance: dispatch + cancel + query)
- [ ] Per-turn tokens: pi turns have non-zero `tokens_in`/`tokens_out`, and `sum(tokens_out) + sum(reason_tokens) == Result.OutputTokens` per the disjointness contract (acceptance: unit test + one live run)
- [ ] Bounded size: 456-turn synthetic replay produces capped rows; no row exceeds its caps (acceptance: unit test)
- [ ] `go build ./...`, `go vet ./...`, `make test` (observatory + executor + coordinator suites), `make check-boundaries` all pass

## Testing Strategy

**Unit tests:**
- Recorder: replay a synthetic 5-turn event sequence (turn/text/tool use/tool result/turn end), assert ChatTurn roles, block order, caps (tool result > 8KB truncated with marker)
- pi usage: fixture NDJSON with `message_end` usage incl. reasoning → `OnTurnUsage` deltas with `Output - Reasoning`; fixture WITHOUT usage on assistant `message_end` → 0s, no invention (D4 wire-drift guard stays intact)
- SQLite PutChatMessages: insert, re-insert same ids (idempotency), query via GetChatMessagesByTaskID round-trip
- Firestore impl: emulator-based round-trip if the repo's emulator harness covers it; otherwise the mapToChatMessage mapping is unit-tested

**Integration tests:**
- Daemon completion path: existing finalize parity test pattern (task_finalize_matrix_test.go) + PutChatMessages replay
- Cap at scale: a 456-turn synthetic stream completes within memory bound ( recorder stays O(turns), test with generated stream)

**Manual testing:**
- One cloud dispatch (ailang_only lane agent) + `ailang chains chat <id> --remote gcp`
- `gcloud` IAM diff after the SA grant (O1)

## Deferred Decisions

The following are intentionally left open for the implementer:

- Exact cap constants (8KB tool results, 64KB text) — agent may choose, runtime-tunable, document in code
- Whether the recorder writes a `content_text` summary column alongside `content_json` (readers today only use content_json; the motoko importer writes both) — agent may decide; recommendation: skip it, keep one source of truth
- Migrating importer_motoko.go and claudehistory/importer.go onto `PutChatMessages` — **follow-up sprint** (D5): needs ChatMessage struct extension for `content_text`/`content_thinking`/`request_id`/`stage_id` first
- codex/claude/opencode driving `OnTurnUsage` — follow-up; note codex emits **cumulative** tokens (EXECUTOR_SHAPE.md streaming contract), so per-turn deltas need diffing, not summing
- Backfill of historical `chat.jsonl`/session artifacts from GCS — follow-up tool once (a) or (b) exists

## Non-Goals

- **No new chat reader paths** — `chains chat`, `handlers_chains_routes.go`, `chains_data.go` all work today; this doc only adds writers.
- **Not migrating the two existing importers in-scope** (D5) — audit found they write columns the shared struct doesn't carry; forcing them through `PutChatMessages` now would silently drop `content_text`/`content_thinking`/`request_id`.
- **No real-time streaming of chat to the dashboard** — the Pub/Sub progress broadcaster (M-CLOUD-PROGRESS-TRACKING) already covers live visibility; this doc banks the durable record.
- **No managed_agents/opencode harness-log importers** — opencode has no chat-transcript importer (V16) and the memory note stands; the recorder fixes this via the stream path instead, which is executor-agnostic.

## Timeline

**Day 1** (~8 hours):
- Phase 1 (M1 recorder + pi): ~4h
- Phase 2 (M2 write path + rig daemon): ~3h
- Phase 3 (M3 cloud): ~1h code + infra coordination (O1 grant, D6 TTL — async, human)

**Total: ~1 day / 500-700 LOC**, plus two small ailang-multivac infra changes if (a) is chosen.

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Job SA lacks Firestore write (O1 unverifiable from this repo — terraform lives in ailang-multivac) | High for M3(a) | O1 is the first sprint task and gates the option; (b) is the no-new-grant fallback, already scoped |
| Write amplification on Firestore (456 turns × several docs) | Med | Per-turn batch on `OnTurnEnd` (one batched write per turn, not per event); TTL policy (D6) bounds growth; doc count ≈ turns, same order as spans already written |
| Third-writer sprawl (audit-before-patching, CLAUDE.md §3) | Med | `PutChatMessages` is the ONE canonical writer; the recorder is its only new caller; existing importers migrate later (D5) |
| Recorder memory on pathological runs | Low | Caps (8KB/64KB) bound per-turn size; recorder is O(turns); thrash/cost kills already terminate runaway runs before recorder grows unbounded |
| Schema change (D2 reason_tokens) breaks older readers | Low | `ALTER TABLE ... ADD COLUMN ... DEFAULT 0` is additive; quorum trigger 3 fires and the quorum reviews it; readers use COALESCE |
| Pub/Sub payload growth if (c) tempted | — | Rejected at design; cap discipline documented |

## Verification Log

Every load-bearing claim (including negative-existence claims) verified against the code in this session (2026-09-16):

| # | Claim | Method | Result |
|---|-------|--------|--------|
| V1 | `EventHandler` carries the five conversation events (OnTurnStart/OnText/OnToolUse/OnToolResult/OnTurnEnd) | Read `internal/executor/executor.go` (~line 454) | Confirmed |
| V2 | pi wrapper drives all five | Read `internal/executor/pi/pi.go:290-400` — OnTurnStart (~307), OnText, OnToolUse, OnToolResult, OnTurnEnd (~398) | Confirmed |
| V3 | `observatory.Backend` has NO chat write method (only Get by task/session + Count) | Read backend.go chat section; grep `PutChatMessages\|InsertChatMessage\|WriteChatMessage\|SaveChatMessage` over internal/ + cmd/ → no hits | Confirmed |
| V4 | Only two chat writers exist, both raw SQL INSERTs into local SQLite | grep `INSERT INTO chat_messages` (non-test): importer_motoko.go:253,282; claudehistory/importer.go:215 | Confirmed |
| V5 | pi never sets `Result.Transcript` | grep `Transcript` in internal/executor/pi/ → no hits (transcriptBuf feeds only `Output`, pi.go:466,488) | Confirmed |
| V6 | opencode never sets `Result.Transcript` | grep `Transcript` in internal/executor/opencode/ → no hits. **Refinement**: codex and claude DO set it (codex.go:529+, claude.go:557+) | Confirmed |
| V7 | `pubsub.TaskCompletion` has no transcript field | Read internal/pubsub/topics.go:59-112 — has metrics, Summary, `ArtifactGCSPath`; no transcript | Confirmed |
| V8 | Firestore backend has readers on obs_chat_messages but no writer | grep ChatMessage in internal/storage/firestore/ → mapToChatMessage + Get/Count only (observatory_chains_helpers.go:137-252) | Confirmed |
| V9 | The cloud execute-job binary never touches Firestore | grep `firestore` in cmd/ailang/coordinator_cloud*.go → no hits; job uses Pub/Sub + GCS mount only. **Open**: the SA's terraform grants live in ailang-multivac (cloud_run_jobs.tf, per docs/internal/EXECUTOR_SHAPE.md) — not present in this repo, so the grant question itself is O1, gated in-plan | Confirmed (binary); O1 open |
| V10 | retention.go covers chat_messages | Read internal/observatory/retention.go:43,102-108 — 30-day TTL on `created_at`, chunked deletes | Confirmed |
| V11 | Per-turn usage from pi: `output += u.Output - u.Reasoning` (reasoning subtracted, disjoint per Result contract) | Read pi.go:370-380 (D5 comment: "reasoning is inside output on the wire; the Result contract wants them disjoint") | Confirmed |
| V12 | `internal/observatory` imports `internal/executor` → executor CANNOT import observatory (import cycle) | observatory/cost_classify.go:3 imports executor; hence D3: recorder message type lives in executor | Confirmed — **corrects the task brief's proposed shape** |
| V13 | The daemon binds stage↔session on completion (`UpdateStageSession` from `Result.SessionID`) | Read task_finalize.go:288, daemon_tasks_chain.go:38 | Confirmed |
| V14 | `chains chat` reads by stage session id / task id | Read chains_chat.go:87, chains_data.go:176,200 | Confirmed |
| V15 | The EventHandler interface carries NO per-turn usage → new optional interface needed (D4) | Read executor.go EventHandler + optional interfaces (ContextAwareHandler, RawStreamLineHandler, MetricsHandler at ~481-497) | Confirmed |
| V16 | opencode has NO chat-transcript importer | grep importers: only claudehistory (Claude JSONL) and motoko exist; memory note consistent with code | Confirmed |
| V17 | Firestore doc limit ~1MB; Pub/Sub message limit 10MB | External documented GCP limits (Firestore: 1,048,487 bytes/doc; Pub/Sub: 10MB/message). Not re-run live (no gcloud in this session); stable published constants | Cited, not re-measured |
| V18 | `ChatMessage` struct lacks `content_text`/`content_thinking`/`request_id`/`stage_id` (importer-migration gap, D5) | Read observatory/store_chat.go:13-31 + claudehistory/importer.go INSERT column list; schema.sql:224-244 has the columns | Confirmed |
| V19 | Cloud artifacts: transcript.txt written only when `Result.Transcript != ""` (empty for pi today); session.jsonl only for Claude (CLAUDE_CONFIG_DIR redirect) | Read coordinator_cloud_github.go:495-565, coordinator_cloud.go:499-505 | Confirmed |
| V20 | No prior design doc covers executor chat-transcript banking | `ailang docs search` (SimHash + neural) on "chains executor transcripts" / "chat transcript": no match ≥0.45; nearest related docs read directly and cited below | Confirmed |
| V21 | The 456-turn chain-687d6ebc measurement | Reported in the task brief (measured 2026-09-16); NOT re-measured in this session (prod Cloud Logging not queried here) | Reported, not re-verified |

## Related Documents

**Implemented (may inform design):**
- [m-chains-source-of-truth.md](../../implemented/v0_7_2/m-chains-source-of-truth.md) — established `chains chat` readers, chat Backend methods, chains as canonical CLI
- [m-dx-pi-harness.md](../../implemented/v0_35_0/m-dx-pi-harness.md) — pi as first-class harness; executor-vs-development-harness split
- [EXECUTOR_SHAPE.md](../../../docs/internal/EXECUTOR_SHAPE.md) — executor contract; codex cumulative-token note used in Deferred Decisions
- [m-completion-path-parity.md](../m-completion-path-parity.md) — idempotent finalize-write precedent (`SetStage*`) reused by PutChatMessages

**Planned (check for overlap):**
- (search found none ≥0.45; M-COMPLETION-PATH-PARITY cited above is the nearest, distinct: finalize writes, not transcript capture)

## References

- [Design Axioms](/docs/references/axioms) - The 12 non-negotiable principles
- [docs/internal/message-plane-topology.md](../../../docs/internal/message-plane-topology.md) — store-plane switch env vars (the Firestore observatory client in the job must follow the same project-pinning discipline)
- CLAUDE.md §1 "Look the question up, not the tool" — the measured failure this doc fixes

## Future Work

- Migrate importer_motoko + claudehistory onto `PutChatMessages` (D5 follow-up; needs ChatMessage struct extension)
- codex/claude/opencode `OnTurnUsage` drivers (codex needs cumulative→delta diffing)
- Backfill tool: import historical `session.jsonl`/`chat.jsonl` GCS artifacts into `obs_chat_messages`
- Per-turn cost column derived from modelreg at bank time (today only tokens; cost attribution stays at stage level)

---

**Document created**: 2026-09-16
**Last updated**: 2026-09-16
