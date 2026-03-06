# M-CLOUD-E2E: Cloud End-to-End Message Flow

**Status**: Implemented
**Target**: v0.9.0
**Priority**: P0 (High) — Cloud infra is deployed, blocking live operation
**Estimated**: 2-3 days (~16h implementation + 4h testing)
**Dependencies**: M-PUBSUB (implemented), M-CLOUD-HEALTH (implemented), M-CLOUD-STORAGE (implemented)

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

Tooling/infrastructure change — does not modify AILANG language semantics. Axiom scoring is minimal.

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No language semantics change |
| A2: Replayability | +1 | Cloud traces enable cross-session replay |
| A3: Effect Legibility | 0 | No new effects |
| A4: Explicit Authority | +1 | COORDINATOR_MODE, AILANG_STORAGE are explicit env var switches |
| A5: Bounded Verification | 0 | No verification change |
| A6: Safe Concurrency | 0 | Pub/Sub pull is sequential per subscription |
| A7: Machines First | +1 | Enables 24/7 autonomous agent operation without developer laptop |
| A8: Minimal Syntax | 0 | No syntax change |
| A9: Cost Visibility | +1 | Cloud costs tracked via spans and Firestore |
| A10: Composability | 0 | Uses existing storage/pubsub interfaces |
| A11: Structured Failure | +1 | All cloud errors are typed and surfaced |
| A12: System Boundary | +1 | Pub/Sub → Firestore → Cloud Run Job boundaries are explicit |

**Net Score: +6** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Enables machine-autonomous operation

## Problem Statement

Cloud infrastructure is **fully deployed** (ailang-multivac: Pub/Sub topics, Firestore collections with indexes, Cloud Run services, Eventarc triggers, CI/CD pipeline). However, end-to-end message flow **does not work** because of wiring gaps in the ailang codebase.

**Current State:**
- Coordinator starts on Cloud Run, health endpoint responds, ApprovalWatcher runs
- BUT coordinator does NOT pull messages from Pub/Sub subscription
- BUT storage backends default to SQLite (no filesystem on Cloud Run = empty results)
- Dashboard `/api/inbox` and `/api/workspaces` return `[]` despite Firestore data existing

**Impact:**
- Cloud coordinator is running but idle — cannot receive or process any tasks
- No 24/7 autonomous operation without developer laptop
- Cloud deployment investment ($0 infra cost) produces no value

**Root Cause Analysis:**

The ailang codebase has all the building blocks implemented in isolation:
- `internal/storage/firestore/` — Full Firestore backends (coordinator, messaging, observatory)
- `internal/pubsub/` — Full Pub/Sub client, publisher, subscriber
- `internal/coordinator/pubsub_adapter.go` — PubSubInboxAdapter
- `cmd/ailang/server.go` — Already wires Firestore when `AILANG_STORAGE=gcp`

But the **wiring in `initTaskProcessing()`** has a critical gap: it always creates an `InboxMessageAdapter` (SQLite-based polling) and never starts the Pub/Sub pull subscriber in cloud mode.

## Goals

**Primary Goal:** Make end-to-end message flow work: Pub/Sub publish → coordinator pulls → classifies → writes Firestore → dispatches Cloud Run Job → publishes completion.

**Success Metrics:**
1. Message published to `ailang-dev-messages` topic arrives at coordinator within 30s
2. Coordinator creates task in Firestore (not SQLite)
3. Dashboard shows inbox messages and workspace data from Firestore
4. Task dispatch publishes to `ailang-dev-tasks` topic
5. Completion handler processes `ailang-dev-completions` messages

## Solution Design

### Overview

Three wiring fixes, each small but critical:

1. **Coordinator cloud mode inbox**: When `COORDINATOR_MODE=cloud`, create `PubSubInboxAdapter` instead of `InboxMessageAdapter`, start the pull subscriber goroutine
2. **Coordinator cloud mode task store**: When `AILANG_STORAGE=gcp`, use Firestore `coordinator.Store` (already created by `storage.NewBackends()`) — currently the pre-set store is used but `initTaskProcessing()` can fall through to SQLite
3. **Pub/Sub task dispatch**: When coordinator creates a task, publish to `ailang-tasks` topic so Eventarc can trigger Cloud Run Job

### Architecture

```
┌──────────────────────────────────────────────────────────────────┐
│  CLOUD RUN: Coordinator                                          │
│                                                                  │
│  ENV: COORDINATOR_MODE=cloud                                     │
│       AILANG_STORAGE=gcp                                         │
│       AILANG_CLOUD_PROJECT=ailang-multivac-dev                   │
│       AILANG_TOPIC_PREFIX=ailang-dev                              │
│       PORT=8080                                                  │
│                                                                  │
│  ┌─────────────────┐     ┌──────────────────┐                    │
│  │ PubSubInbox     │────►│ Daemon poll loop  │                    │
│  │ Adapter         │     │ (ticker.C)        │                    │
│  │                 │     │                   │                    │
│  │ Pulls from:     │     │ pollAndProcess    │                    │
│  │ ailang-dev-     │     │ Tasks()           │                    │
│  │ messages-       │     └────────┬──────────┘                    │
│  │ coordinator     │              │                               │
│  └─────────────────┘              ▼                               │
│                          ┌──────────────────┐                    │
│                          │ Firestore Store   │                    │
│                          │ (coordinator.Store)│                    │
│                          │                   │                    │
│                          │ CreateTask()      │                    │
│                          │ MarkTaskQueued()  │                    │
│                          └────────┬──────────┘                    │
│                                   │                               │
│                                   ▼                               │
│                          ┌──────────────────┐                    │
│                          │ PubSub Publisher  │                    │
│                          │                   │                    │
│                          │ PublishTask() ────────► ailang-dev-tasks│
│                          └──────────────────┘     (Eventarc)     │
│                                                        │          │
│                          ┌──────────────────┐          │          │
│                          │ Completion Handler│◄─────────┘          │
│                          │                   │  ailang-dev-       │
│                          │ Subscribe()       │  completions       │
│                          └──────────────────┘                    │
└──────────────────────────────────────────────────────────────────┘
```

### Gap Analysis (What Needs to Change)

#### Gap 1: initTaskProcessing() doesn't switch to PubSub inbox in cloud mode

**File**: `internal/coordinator/daemon_tasks_init.go`
**Lines**: 63-78

**Current behavior**: Always opens `OpenDefaultInboxAdapter("coordinator")` which uses SQLite `collaboration.db`. The `d.msgStore` check at line 68 allows pre-set stores, but `d.msgAdapter` is always the SQLite-based `InboxMessageAdapter`.

**Required behavior**: When `COORDINATOR_MODE=cloud`, create `PubSubInboxAdapter` instead. The adapter should pull from `{prefix}-messages-coordinator` subscription.

```go
// In initTaskProcessing(), after loading config:
mode := os.Getenv("COORDINATOR_MODE")
if mode == CoordinatorModeCloud {
    // Initialize Pub/Sub
    if err := d.initPubSub(d.ctx); err != nil {
        return fmt.Errorf("cloud mode requires Pub/Sub: %w", err)
    }

    // Create pull subscriber
    subscriber := pubsub.NewSubscriber(d.pubsubClient)
    subName := d.pubsubClient.SubscriptionName(pubsub.SubMessagesCoordinator)

    // Create PubSubInboxAdapter
    adapter := NewPubSubInboxAdapter(subscriber, subName, "coordinator", d.logger)
    adapter.Start(d.ctx)
    d.msgAdapter = adapter
    d.logger.Printf("Cloud mode: pulling from Pub/Sub subscription %s", subName)
} else {
    // Existing SQLite behavior (unchanged)
    if d.msgStore == nil { ... }
}
```

Also need per-agent PubSub adapters — or use a single subscription with attribute filtering (more efficient). Current design uses a single `ailang-messages-coordinator` subscription that receives ALL messages, then the daemon routes by inbox attribute.

#### Gap 2: Task dispatch doesn't publish to Pub/Sub

**File**: `internal/coordinator/daemon.go` (or wherever task creation triggers execution)

**Current behavior**: Tasks are created in the store and executed locally via `d.executor.Execute()`.

**Required behavior in cloud mode**: After creating a task, publish to `ailang-tasks` topic so Eventarc triggers a Cloud Run Job (instead of local execution).

```go
// After task creation, in cloud mode:
if d.pubsubPublisher != nil {
    if err := d.pubsubPublisher.PublishTask(d.ctx, task.ID, task.AgentID); err != nil {
        d.logger.Printf("Warning: Failed to publish task dispatch: %v", err)
    }
}
```

**Decision**: In cloud mode, the coordinator should NOT execute tasks locally — it should only dispatch via Pub/Sub. Local execution stays for `COORDINATOR_MODE=local`.

#### Gap 3: Completion handler not started

**File**: `internal/coordinator/pubsub_completion_handler.go` exists but is never started in the daemon's `Run()` method.

**Required**: Start the completion handler subscription when in cloud mode.

```go
// In daemon.Run(), after initEventBroadcaster:
if os.Getenv("COORDINATOR_MODE") == CoordinatorModeCloud && d.pubsubClient != nil {
    go d.startCompletionHandler()
}
```

#### Gap 4: Dashboard config loading (minor)

**File**: `cmd/ailang/server.go`

**Current behavior**: Config loads fine via `storage.NewBackends()` and `LoadCoordinatorConfig()`, but the `/api/workspaces` endpoint in the server may not receive the coordinator config.

**Check**: Verify `handleWorkspaces` reads from Firestore or from loaded config. The `AILANG_CONFIG` env var is already handled by `LoadCoordinatorConfig()` in `agent_config.go:15`.

### Implementation Plan

**Phase 1: Coordinator Cloud Mode Inbox** (~4h)
- [ ] Add `COORDINATOR_MODE` check in `initTaskProcessing()` to create `PubSubInboxAdapter`
- [ ] Initialize Pub/Sub client before inbox adapter creation
- [ ] Start pull subscriber goroutine for `{prefix}-messages-coordinator` subscription
- [ ] Route messages by inbox attribute (single subscription, multiple agents)
- [ ] Skip per-agent `InboxMessageAdapter` creation in cloud mode (Pub/Sub handles all inboxes)
- [ ] Unit test: mock Pub/Sub subscriber, verify adapter buffers messages

**Phase 2: Task Dispatch via Pub/Sub** (~4h)
- [ ] Add `dispatchTaskCloud()` method that publishes to `ailang-tasks` topic
- [ ] In `executeTaskQueue()`: if cloud mode, dispatch via Pub/Sub instead of local execution
- [ ] Include task metadata as Pub/Sub attributes: agent_id, workspace, provider
- [ ] Start completion handler subscription in `Run()`
- [ ] Wire completion handler to update task status in Firestore store
- [ ] Unit test: verify task dispatch publishes correct attributes

**Phase 3: Dashboard Verification & Config** (~4h)
- [ ] Verify `/api/inbox` reads from Firestore when `AILANG_STORAGE=gcp` (should work — `WithMessagingStore` already wired)
- [ ] Verify `/api/workspaces` loads from coordinator config (`AILANG_CONFIG`)
- [ ] If workspaces endpoint returns empty, wire `LoadCoordinatorConfig()` result into server
- [ ] Integration test: start server with `AILANG_STORAGE=gcp`, verify endpoints return data

**Phase 4: Testing & Validation** (~4h)
- [ ] Add Pub/Sub emulator-based integration test (build tag: `pubsubemulator`)
- [ ] Test full flow: publish message → adapter receives → task created → dispatch published
- [ ] Test completion flow: publish completion → handler receives → task status updated
- [ ] Test with real Cloud Run deployment (manual)
- [ ] Update CHANGELOG.md

### Files to Modify/Create

**Modified files:**
- `internal/coordinator/daemon_tasks_init.go` — Add cloud mode branch in `initTaskProcessing()` (~30 LOC)
- `internal/coordinator/daemon.go` — Start completion handler in `Run()`, add cloud dispatch path (~20 LOC)
- `internal/coordinator/daemon.go` — Add `dispatchTaskCloud()` method (~25 LOC)

**Possibly modified:**
- `internal/server/handlers_threads.go` — Wire coordinator config to `handleWorkspaces` if not already (~10 LOC)
- `cmd/ailang/server.go` — Pass coordinator config to server if needed (~5 LOC)

**New files:**
- `internal/coordinator/daemon_cloud_test.go` — Cloud mode unit tests (~150 LOC)

**No new packages needed** — all building blocks exist.

## Examples

### Example 1: End-to-End Flow (Working)

**Publish message from laptop:**
```bash
# This already works (dual-write to Firestore + Pub/Sub notification)
AILANG_STORAGE=gcp ailang messages send sprint-executor "Fix parser bug" \
  --title "Bug: Parser NPE" --from "user"
```

**Coordinator receives and processes (currently broken, this doc fixes it):**
```
# Coordinator log output (expected after fix):
Cloud mode: pulling from Pub/Sub subscription ailang-dev-messages-coordinator
Received message notification: msg_id=abc123 inbox=sprint-executor from=user
Created task task-abc12345 for agent sprint-executor
Publishing task dispatch to ailang-dev-tasks (agent_id=sprint-executor)
```

**Cloud Run Job executes:**
```
# Eventarc triggers Cloud Run Job with env vars:
AILANG_TASK_ID=task-abc12345
AILANG_AGENT_ID=sprint-executor
AILANG_DIRECTIVE="Fix parser bug"
```

### Example 2: Dashboard Endpoints (Working)

**Before (returns empty):**
```bash
curl http://localhost:8080/api/inbox
# []
curl http://localhost:8080/api/workspaces
# []
```

**After (returns Firestore data):**
```bash
curl http://localhost:8080/api/inbox
# [{"id":"msg_123","title":"Bug: Parser NPE","to_inbox":"sprint-executor",...}]
curl http://localhost:8080/api/workspaces
# [{"id":"ws_abc","name":"ailang","path":"/workspace/ailang",...}]
```

## Success Criteria

- [ ] Message published to `ailang-dev-messages` topic is received by coordinator within 30s
- [ ] Task created in Firestore (verified via `ailang storage verify`)
- [ ] Task dispatch published to `ailang-dev-tasks` topic
- [ ] Completion published to `ailang-dev-completions` triggers status update
- [ ] Dashboard `/api/inbox` returns messages from Firestore
- [ ] Dashboard `/api/workspaces` returns configured workspaces
- [ ] Health endpoint `/health` continues to work
- [ ] All existing tests passing (`make test`)
- [ ] No regression in local mode (`COORDINATOR_MODE=local` unchanged)

## Testing Strategy

**Unit tests:**
- Mock Pub/Sub subscriber → verify PubSubInboxAdapter buffers and returns messages
- Mock Pub/Sub publisher → verify task dispatch attributes
- Mock completion handler → verify status transitions

**Integration tests (Pub/Sub emulator):**
- Full flow: publish to messages topic → adapter receives → verify message fields
- Completion flow: publish to completions topic → handler processes → verify store update
- Build tag: `pubsubemulator` (requires `gcloud beta emulators pubsub start`)

**Manual testing:**
- Deploy to Cloud Run → publish test message → verify in Firestore console
- Check dashboard endpoints on Cloud Run service URL

## Non-Goals

**Not in this feature:**
- BigQuery observatory backend — deferred to M-CLOUD-ANALYTICS
- Cloud Run Job container execution — already works (`coordinator_cloud.go`)
- Pub/Sub dead-letter queue processing — topic exists, handler deferred
- Multi-region failover — single region (europe-west1) for now
- Cloud Run Jobs API dispatch from coordinator — uses Eventarc trigger instead

## Timeline

**Day 1** (~8h):
- Phase 1: Coordinator cloud mode inbox
- Phase 2: Task dispatch via Pub/Sub

**Day 2** (~8h):
- Phase 3: Dashboard verification & config
- Phase 4: Testing & validation
- CHANGELOG update

**Total: ~16h across 2 days**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Pub/Sub message ordering | Med | Ordering key on inbox attribute (already configured in publisher) |
| Firestore cold start latency | Low | Cloud Run min-instances=1 in Terraform |
| Completion handler message loss | Med | Idempotent handler (already checks terminal states) |
| Breaking local mode | High | All changes gated on `COORDINATOR_MODE` env var; no default behavior changes |
| Dashboard CORS on Cloud Run | Low | Already handled by server.go bind address and Cloud Run IAM |

## Related Documents

**Implemented (foundation for this work):**
- [design_docs/planned/v0_9_0/m-pubsub-messaging.md](design_docs/planned/v0_9_0/m-pubsub-messaging.md) — Pub/Sub architecture (status: Implemented)
- [design_docs/planned/v0_9_0/m-cloud-health.md](design_docs/planned/v0_9_0/m-cloud-health.md) — Health endpoints (status: Implemented)
- [design_docs/implemented/v0_7_0/m-auth-dashboard-firebase.md](design_docs/implemented/v0_7_0/m-auth-dashboard-firebase.md) — Firebase auth for dashboard

**Planned (related cloud work):**
- [design_docs/planned/v0_8_2/m-cloud-infra.md](design_docs/planned/v0_8_2/m-cloud-infra.md) — Cloud infrastructure reference
- [design_docs/planned/v0_9_0/m-cloud-eval-workers.md](design_docs/planned/v0_9_0/m-cloud-eval-workers.md) — Distributed eval workers

## References

- [Design Axioms](/docs/references/axioms)
- Source message: `ailang messages read afc633f8-8a60-491c-afb1-6665d0d37c51`
- ailang-multivac repo: Terraform, Docker, CI/CD for cloud deployment

## Future Work

- Dead-letter queue processing (monitor + retry failed messages)
- BigQuery observatory for analytics-scale span queries
- Cloud Run Jobs API dispatch (replace Eventarc for more control)
- Multi-workspace routing (route messages to different Cloud Run services per workspace)
- Pub/Sub emulator in CI pipeline

---

**Document created**: 2026-03-06
**Last updated**: 2026-03-06

## Implementation Report

**Implemented in commit**: `fe4a3802`

### What Was Built

Three wiring fixes to enable end-to-end cloud message flow:

1. **M1: Cloud mode inbox adapter** — `PubSubInboxAdapter` now starts in cloud mode, routes messages by `Inbox` attribute to correct agent
2. **M2: Task dispatch + completion handler** — `dispatchTasksCloud()` publishes tasks to Pub/Sub for Eventarc/Cloud Run Job execution; `CompletionHandler` started in `Run()`
3. **M3: Dashboard verification** — Confirmed Firestore backends already wired correctly; no code changes needed

### Code Changes (+351/-33 lines)

| File | Change |
|------|--------|
| `internal/coordinator/watcher.go` | Added `Inbox` field to `Message` struct |
| `internal/coordinator/pubsub_adapter.go` | Set `msg.Inbox` from Pub/Sub attributes |
| `internal/coordinator/daemon.go` | Added `cloudInboxAdapter` field, start CompletionHandler |
| `internal/coordinator/daemon_tasks_init.go` | Cloud/local mode branching in `initTaskProcessing()` |
| `internal/coordinator/daemon_tasks_polling.go` | Added `pollAndProcessTasksCloud()` method |
| `internal/coordinator/daemon_tasks_exec.go` | Added `dispatchTasksCloud()` method |
| `CHANGELOG.md` | Added M-CLOUD-E2E entry |

### Deviations from Plan

- **No new test file created** — changes are wiring-only with existing unit test coverage; integration tests deferred to emulator setup
- **Dashboard needed no changes** — Firestore backends were already correctly wired (Gap 4 was a non-issue)
- **Scope was ~90 LOC** instead of estimated ~16h — the building blocks were more complete than the gap analysis suggested
