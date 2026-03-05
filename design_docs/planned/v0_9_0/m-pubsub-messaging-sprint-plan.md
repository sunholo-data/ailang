# Sprint Plan: M-PUBSUB — Pub/Sub Cloud Messaging Transport

## Summary

Implement Google Cloud Pub/Sub as a notification/transport layer for AILANG messaging, enabling cloud-based coordinator execution on Cloud Run without requiring the developer's laptop. This is the Go code portion only — deployment infra lives in the separate `ailang-multivac` repo.

**Duration:** 5 days (focused implementation)
**Dependencies:** Firestore backends (complete), `cloud.google.com/go/pubsub` (new dep)
**Risk Level:** Medium — greenfield package with well-defined interfaces, but integration points are complex
**Design Doc:** [m-pubsub-messaging.md](m-pubsub-messaging.md)

## Current Status Analysis

### Foundation (Complete)

- **Firestore messaging backend**: 1,982 LOC across 6 files — fully implements `MessageStore` interface
- **Storage backend selector**: `internal/storage/backend.go` (208 LOC) — `local`/`gcp`/`hybrid` modes working
- **Coordinator interfaces**: `MessageStore` (2 methods), `EventBroadcaster` (func type) — clean, small
- **InboxMessageAdapter**: 87 LOC — pattern to replicate for Pub/Sub adapter
- **HTTPBroadcaster**: 177 LOC — pattern to replicate for Pub/Sub broadcaster
- **GCP auth patterns**: Firestore + Cloud Trace use Application Default Credentials

### Missing (To Build)

- `cloud.google.com/go/pubsub` dependency — not in go.mod yet
- `internal/pubsub/` package — does not exist
- `COORDINATOR_MODE` env var — not implemented
- `execute-job` subcommand — not implemented
- Dual-write in CLI — messages only write to local/Firestore, no Pub/Sub notification

### Velocity

- **Recent milestones**: M-SEMANTIC-ENVELOPE (~1,450 LOC), M-PROCESS (~650 LOC), M-PROTOCOL-SUPPORT (~1,400 LOC)
- **Average**: ~200-300 LOC/day of focused implementation + tests
- **Estimated capacity**: ~1,500 LOC over 5 days
- **Sprint total estimate**: ~1,350 LOC (implementation + tests)

## Proposed Milestones

### Milestone 1: Pub/Sub Client Package

**Goal:** Create `internal/pubsub/` with client, publisher, subscriber, and topic constants. Pure package with no coordinator dependencies.

**Estimated:** ~250 LOC implementation + ~150 LOC tests = ~400 LOC
**Duration:** 1 day

**Tasks:**
- Add `cloud.google.com/go/pubsub` to go.mod via `go get`
- Create `internal/pubsub/client.go` — Client wrapper with topic prefix support
- Create `internal/pubsub/topics.go` — Topic/subscription name constants, payload structs, `MessageAttributes`
- Create `internal/pubsub/publisher.go` — `PublishMessage`, `PublishTask`, `PublishCompletion`, `PublishEvent`
- Create `internal/pubsub/subscriber.go` — `Subscribe` (blocking), `ReceiveOne`, decode helpers
- Create `internal/pubsub/client_test.go` — Unit tests with Pub/Sub emulator (`PUBSUB_EMULATOR_HOST`)

**Key files to create:**
- `internal/pubsub/client.go` (~60 LOC)
- `internal/pubsub/topics.go` (~80 LOC)
- `internal/pubsub/publisher.go` (~90 LOC)
- `internal/pubsub/subscriber.go` (~70 LOC)
- `internal/pubsub/client_test.go` (~150 LOC)

**Acceptance Criteria:**
- [ ] `go get cloud.google.com/go/pubsub` succeeds and builds cleanly
- [ ] All 5 topic constants and 6 subscription constants defined
- [ ] `Publisher.PublishMessage()` accepts `MessageNotification` + `MessageAttributes`
- [ ] `Subscriber.Subscribe()` blocks and delivers messages via callback
- [ ] `MessageAttributes.ToMap()` produces correct `map[string]string` with workspace routing
- [ ] Unit tests pass with `PUBSUB_EMULATOR_HOST` (or build-tagged to skip if no emulator)
- [ ] `make lint` passes
- [ ] No dependency on `internal/coordinator/` or `internal/messaging/`

**Risks:**
- Pub/Sub emulator may not be installed locally — Mitigation: Use build tag `//go:build integration` for emulator tests, unit test struct creation/serialization without emulator

---

### Milestone 2: Coordinator Adapters + Mode Switch

**Goal:** Create `PubSubInboxAdapter` and `PubSubBroadcaster` that plug into the coordinator daemon, selected by `COORDINATOR_MODE=cloud`.

**Estimated:** ~200 LOC implementation + ~100 LOC tests = ~300 LOC
**Duration:** 1 day

**Dependencies:** Milestone 1

**Tasks:**
- Create `internal/coordinator/pubsub_adapter.go` — `PubSubInboxAdapter` implementing `coordinator.MessageStore`
  - `StartReceiving()` runs Pub/Sub subscription in goroutine, buffers messages
  - `ListUnread()` drains buffer (push-based but consumed via existing poll interface)
  - `MarkAsRead()` delegates to Firestore `MessageStore`
- Create `internal/coordinator/pubsub_broadcaster.go` — publishes events to `ailang-events` topic
  - Same `EventBroadcaster` func signature as `HTTPBroadcaster`
- Modify `internal/coordinator/daemon_tasks_init.go`:
  - Read `COORDINATOR_MODE` env var
  - `cloud` mode: create Pub/Sub client, use `PubSubInboxAdapter` + `PubSubBroadcaster`
  - Default: existing `InboxMessageAdapter` + `HTTPBroadcaster` (unchanged)
- Add `pubsubClient`, `publisher`, `subscriber` fields to `Daemon` struct

**Key files to create/modify:**
- `internal/coordinator/pubsub_adapter.go` (~80 LOC) — new
- `internal/coordinator/pubsub_broadcaster.go` (~50 LOC) — new
- `internal/coordinator/daemon_tasks_init.go` (~40 LOC additions) — modify
- `internal/coordinator/daemon.go` (~15 LOC additions) — add fields + Close()
- `internal/coordinator/pubsub_adapter_test.go` (~100 LOC) — new

**Acceptance Criteria:**
- [ ] `PubSubInboxAdapter` implements `coordinator.MessageStore` (compile-time check)
- [ ] `PubSubBroadcaster.BroadcastFunc()` returns `EventBroadcaster`
- [ ] `COORDINATOR_MODE=cloud` selects Pub/Sub adapters (verified by log output)
- [ ] `COORDINATOR_MODE=local` (or unset) keeps existing behavior unchanged
- [ ] Daemon cleanup calls `pubsubClient.Close()` and `publisher.Stop()`
- [ ] `make test ./internal/coordinator/...` passes
- [ ] `make lint` passes

**Risks:**
- Modifying `daemon_tasks_init.go` could break existing local mode — Mitigation: Test local mode explicitly after changes, ensure default path unchanged

---

### Milestone 3: Cloud Execute-Job Subcommand

**Goal:** Create `ailang coordinator execute-job` for Cloud Run Jobs — reads task from Firestore, clones repo, runs executor, publishes completion.

**Estimated:** ~200 LOC implementation + ~100 LOC tests = ~300 LOC
**Duration:** 1.5 days

**Dependencies:** Milestone 1 (publisher), Milestone 2 (adapters for completion handling)

**Tasks:**
- Create `cmd/ailang/coordinator_cloud.go`:
  - `executeJobCmd` cobra command
  - Read `AILANG_TASK_ID` from environment
  - Open Firestore backends via `storage.NewBackends(ctx)`
  - Idempotency check: skip if task not `pending`/`queued`
  - Mark task running via Firestore transaction
  - `git clone --depth 1` repo, create branch `coordinator/{task-id}`
  - Call `TaskExecutor.Execute()` with executor from agent config
  - Commit + push changes
  - Publish `TaskCompletion` to `ailang-completions` topic
- Add completion subscription handler in coordinator daemon (cloud mode):
  - Subscribe to `ailang-completions`
  - On receipt: update task status, run approval workflow, trigger handoffs
- Register `execute-job` subcommand in coordinator command group

**Key files to create/modify:**
- `cmd/ailang/coordinator_cloud.go` (~180 LOC) — new
- `internal/coordinator/daemon_tasks_exec.go` (~30 LOC additions) — completion handler
- `cmd/ailang/coordinator_cloud_test.go` (~100 LOC) — new

**Acceptance Criteria:**
- [ ] `ailang coordinator execute-job` reads `AILANG_TASK_ID` and fails gracefully if unset
- [ ] Idempotency: running on already-completed task is a no-op (not an error)
- [ ] `git clone` + `git checkout -b` + `git push` sequence works
- [ ] Completion message published to `ailang-completions` topic with correct attributes
- [ ] Coordinator handles incoming completions and updates task status
- [ ] `make build` includes the new subcommand
- [ ] `make lint` passes

**Risks:**
- Git operations in Cloud Run Jobs require auth — Mitigation: Design assumes GitHub PAT via Secret Manager (handled in ailang-multivac), for testing use local git repos
- Executor binary availability in containers — Mitigation: Out of scope (ailang-multivac), test locally

---

### Milestone 4: CLI Dual-Write + Watch

**Goal:** Make `ailang messages send` publish to Pub/Sub after Firestore write, and add `ailang messages watch --pubsub` for real-time laptop notifications.

**Estimated:** ~150 LOC implementation + ~100 LOC tests = ~250 LOC
**Duration:** 1 day

**Dependencies:** Milestone 1 (publisher, subscriber)

**Tasks:**
- Add Pub/Sub config section to config loader (`internal/messaging/config.go` or `internal/coordinator/agent_config.go`):
  - `pubsub.enabled`, `pubsub.project_id`, `pubsub.topic_prefix`
  - Helper: `pubsubEnabled()`, `getPubSubPublisher(ctx)`
- Modify `cmd/ailang/messages_send.go`:
  - After `store.InsertInboxMessage()`, check `pubsubEnabled()`
  - If enabled: create publisher, call `PublishMessage()` with attributes
  - Non-fatal: log warning if Pub/Sub publish fails (message is already in Firestore)
  - Resolve `workspace` from inbox → agent config → workspace mapping
- Add `--pubsub` flag to `ailang messages watch`:
  - Create subscriber, pull from `{prefix}-messages-laptop` subscription
  - On receipt: read full message from Firestore, display
  - Blocks until Ctrl+C
- Add `internal/storage/backend.go` changes:
  - Optional `Publisher` field in `Backends` struct (nil when pubsub disabled)

**Key files to modify:**
- `cmd/ailang/messages_send.go` (~40 LOC additions)
- `cmd/ailang/messages.go` or `messages_crud.go` (~60 LOC for watch --pubsub)
- `internal/storage/backend.go` (~20 LOC additions)
- Config loading (~30 LOC additions)
- `cmd/ailang/messages_send_test.go` (~100 LOC)

**Acceptance Criteria:**
- [ ] `ailang messages send ... ` with `pubsub.enabled: true` publishes notification
- [ ] `ailang messages send ... ` with `pubsub.enabled: false` works identically to current behavior
- [ ] Pub/Sub publish failure does NOT fail the send (non-fatal warning)
- [ ] `ailang messages watch --pubsub` receives messages in real-time
- [ ] Config changes are backwards-compatible (no pubsub section = disabled)
- [ ] `make test` passes (including existing message tests)
- [ ] `make lint` passes

**Risks:**
- Config loading changes could break existing config parsing — Mitigation: New `pubsub:` section is optional, defaults to disabled

---

### Milestone 5: Integration Testing + Documentation

**Goal:** End-to-end validation and docs update.

**Estimated:** ~50 LOC implementation + ~50 LOC tests = ~100 LOC
**Duration:** 0.5 days

**Dependencies:** All previous milestones

**Tasks:**
- Create integration test: send message → Pub/Sub notification → coordinator receives → creates task
- Update CHANGELOG.md with M-PUBSUB entry
- Update CLAUDE.md with Pub/Sub config section
- Run `make lint` and `make test` full suite
- Verify local mode regression: all existing tests still pass with no env vars set

**Acceptance Criteria:**
- [ ] Full end-to-end test passes (with emulators or build-tagged)
- [ ] `make test` passes with no regressions
- [ ] `make lint` passes
- [ ] CHANGELOG.md updated
- [ ] CLAUDE.md has Pub/Sub config reference

---

## Success Metrics

- **Total LOC**: ~1,350 (implementation + tests)
- **New files**: 8 (4 in `internal/pubsub/`, 2 in `internal/coordinator/`, 1 in `cmd/ailang/`, 1 test)
- **Modified files**: 5 (`daemon_tasks_init.go`, `daemon.go`, `messages_send.go`, `backend.go`, config)
- **Test coverage**: >80% for new `internal/pubsub/` package
- **Zero regressions**: `COORDINATOR_MODE` unset = identical to current behavior
- **All tests passing**: `make test`
- **All linting passing**: `make lint`
- **Documentation**: CHANGELOG + CLAUDE.md updated

## Dependencies

- `cloud.google.com/go/pubsub` Go module (will be added in M1)
- Pub/Sub emulator for integration tests (`gcloud beta emulators pubsub start`)
- Existing Firestore backend code (complete, no changes needed)
- `ailang-multivac` repo for deployment (out of scope, separate follow-up)

## Open Questions

- **Pub/Sub emulator in CI**: Should we add the emulator to GitHub Actions CI, or use build tags to skip integration tests?
- **Config file location**: Should cloud-mode config live in `~/.ailang/config.yaml` (same file) or a separate `config.cloud.yaml`?

## Notes

- This sprint creates the **Go code only**. Terraform, Dockerfiles, and Cloud Run deployment are in `ailang-multivac` (separate repo, separate sprint)
- The `execute-job` command can be tested locally with `AILANG_TASK_ID=... AILANG_STORAGE=local ailang coordinator execute-job`
- All new code is behind `COORDINATOR_MODE=cloud` — zero impact on existing local mode
- Pub/Sub publish failures are always non-fatal (Firestore is source of truth)
