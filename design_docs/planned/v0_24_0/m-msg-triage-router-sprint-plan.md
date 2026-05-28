# Sprint Plan: M-MSG-TRIAGE-ROUTER (Build Order Step 1)

**Sprint ID**: M-MSG-TRIAGE-ROUTER
**Design doc**: [m-msg-auto-triage-pipeline.md](m-msg-auto-triage-pipeline.md) — Phase M1
**Target**: v0.24.0
**Risk**: Low (connects existing pieces; no new external deps)
**Estimated**: ~3–4 days (~590 LOC impl + ~400 LOC tests)

## Goal

Turn an inbound bug/feature message into a `design-doc-creator`-inbox task automatically: list unread intake-inbox messages → cluster (existing semantic triage) → classify promote/hold/drop → `forward` promotions. The existing coordinator agent runner + `ApprovalWatcher` then take over (unchanged). Human's first touch stays at the `design-approved` label.

## Discovery (grounding facts)

- Clustering is currently **private in the CLI**: `clusterMessages([]messaging.InboxMessage, slot string, threshold float64) []triageCluster` at [cmd/ailang/messages_triage.go:153](../../../cmd/ailang/messages_triage.go#L153). Not reusable → must be extracted.
- Store interface: `messaging.MessageStore` with `ListInboxMessages(InboxListOptions) ([]InboxMessage, error)`, `GetInboxMessage(id)`, `ForwardInboxMessage(id, toInbox string) error` ([internal/messaging/message_store.go](../../../internal/messaging/message_store.go), [inbox.go:529](../../../internal/messaging/inbox.go#L529)).
- `InboxMessage` has `Category` (bug|feature|general|docs|research|refactor|test), `Status`, `ToInbox`, `FromAgent`, `Envelope`, `DupOf` ([inbox.go:18](../../../internal/messaging/inbox.go#L18)).
- Lifecycle template: `ApprovalWatcher` — `NewApprovalWatcher(...)`, `Start(ctx)`, `Stop()`, `pollLoop(ctx)` with `ticker` + `stopCh` + an initial poll ([internal/coordinator/approval_watcher.go:55-296](../../../internal/coordinator/approval_watcher.go#L55)).
- Config root: `CoordinatorConfig` in [internal/coordinator/agent_config.go](../../../internal/coordinator/agent_config.go) → add `Triage *TriageConfig`.
- Daemon holds `d.msgStore messaging.MessageStore` ([daemon.go:74](../../../internal/coordinator/daemon.go#L74)); background workers start in `Run()`/`initTaskProcessing` via `go worker.Start(d.ctx)`.

## Classifier semantics (deterministic, the M1 decision)

For each unread intake message (duplicates already collapsed via `InboxListOptions.Collapsed`):

- **drop** if it's known noise — `FromAgent == "eval-suite"` or `MessageType`/`Category` is a notification/completion (e.g. status pings). Action: no-op (leave in place; archiving deferred).
- **promote** if `Category ∈ {bug, feature}` and `DupOf == ""`. Action: `ForwardInboxMessage(id, "design-doc-creator")`.
- **hold** otherwise (general/docs/research/ambiguous). Action: no-op (stays unread in the intake inbox, already visible).

Clustering (`--cluster-by intent`, threshold ~0.75) is used to **group related promotions** so two reports about the same thing forward together / can be batched; the promote/hold/drop decision is per-message category + dedupe. Idempotency is free: a forwarded message's `to_inbox` changes, so it drops out of the next intake listing.

## Milestones

### M1 — Extract clustering into a reusable library (~120 LOC + ~120 test)
- Move `clusterMessages` + `triageCluster` from `cmd/ailang/messages_triage.go` into `internal/messaging/triage.go` as exported `ClusterMessages(msgs []InboxMessage, slot string, threshold float64) []Cluster` + `Cluster` struct.
- Update the CLI command to call the exported function (delete the private copy).
- Unit tests for clustering (grouping, threshold boundaries, empty input).
- **Acceptance**: `make test` green; `ailang messages triage --cluster-by intent` still works identically.

### M2 — Promotion Classifier (~110 LOC + ~120 test)
- `internal/coordinator/triage_router.go`: pure `classify(msg InboxMessage, cfg TriageConfig) decision` returning promote/hold/drop per the semantics above.
- Table-driven tests across categories, noise sources, and `DupOf` set/unset.
- **Acceptance**: every category × noise × dup combination has a pinned expected decision; 100% branch coverage on `classify`.

### M3 — Triage Router + config + daemon tick (~200 LOC + ~80 test)
- `TriageConfig` struct added to `CoordinatorConfig` (`yaml:"triage"`): `Enabled`, `IntakeInboxes []string`, `PromoteInbox string` (default `design-doc-creator`), `ClusterSlot string` (default `intent`), `SimilarityThreshold float64` (default `0.75`), `PollIntervalSecs int` (default `120`).
- `TriageRouter` struct mirroring `ApprovalWatcher`: `NewTriageRouter(store, cfg)`, `Start(ctx)`, `Stop()`, `pollLoop` (ticker + stopCh + initial run). Each tick: for each intake inbox → `ListInboxMessages{Inbox, UnreadOnly:true, Collapsed:true}` → `ClusterMessages` → `classify` → `ForwardInboxMessage` promotions.
- Wire startup in the daemon gated by `cfg.Triage.Enabled` (off by default).
- Unit test of one `tickOnce` against a fake/in-memory store.
- **Acceptance**: router disabled by default; when enabled, a `tickOnce` forwards promotions and leaves holds; `make lint && make test` green.

### M4 — Integration test + docs (~80 test + docs)
- Integration test (real SQLite temp store): seed a `bug` message → one `tickOnce` → assert it now lives in `design-doc-creator`; a `general` message stays in the intake inbox; a `DupOf`-set message is not forwarded.
- CHANGELOG entry; flip the M1 checkboxes in the design doc; note router is opt-in via config.
- **Acceptance**: integration test green; `make ci` clean; docs updated.

## Success Metrics

- `make ci` green; ≥ 85% coverage on `triage_router.go` + extracted `triage.go`.
- CLI `ailang messages triage` behavior unchanged (no regression).
- Router is **opt-in** (`coordinator.triage.enabled: false` by default) — zero behavior change for existing deployments until enabled.
- End-to-end: synthetic bug → lands in `design-doc-creator` inbox with no manual step.

## Dependencies / Open

- None blocking. Channel notification (so a human gets paged) is Step 2/3 — not required for this sprint; promotions are visible via existing surfaces.
- "Drop = archive" and "hold → dashboard surfacing" are deferred (no-op for now) to keep M1 minimal.
