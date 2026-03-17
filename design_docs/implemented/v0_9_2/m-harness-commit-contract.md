# M-HARNESS-COMMIT-CONTRACT: Website Builder Commit Message & Payload Contract

**Status**: Planned
**Target**: v0.9.3
**Priority**: P1 — blocks website-builder integration with demos repo
**Estimated**: 2 hours (~100 LOC)
**Dependencies**: M-CLOUD-DISPATCH (implemented), M-GIT-GUARDRAILS (implemented)
**Source**: Harness Git Contract Alignment discussion (msg_20260316_110047_f10b6516)

## Problem Statement

The website-builder agent's commit messages use a generic format:
```
Task {taskID}: {directive}
Agent: {agentID}
Timestamp: {RFC3339}
```

The demos repo (portal) needs structured commit messages to identify which site was built:
```
Build: {siteSlug} [briefId={briefId}]
```

Currently `siteSlug` and `briefId` are available in the message payload JSON but are not parsed or passed through the dispatch pipeline to the Cloud Run Job where commits happen.

**Both sides have aligned on the contract:**
- Q1 (files array): `ChangedFiles` already in `TaskCompletion` (done)
- Q2 (commit template): siteSlug + briefId passed as env vars from dispatcher
- Q3 (post-processing): remains in sidecar (no action)
- Q4 (git instructions): removed from brief (demos side done)
- Q5 (error reporting): confirmed working

## Solution Design

### Data Flow

```
Message payload (JSON)     → Daemon task polling      → DispatchParams
  {"siteSlug": "foo",        parse siteSlug/briefId      SiteSlug: "foo"
   "briefId": "abc123"}                                  BriefID: "abc123"
                                                              ↓
Cloud Run Job env vars     ← CloudRun Dispatcher
  AILANG_SITE_SLUG=foo        inject env vars
  AILANG_BRIEF_ID=abc123
                                                              ↓
coordinator_cloud.go       → Commit message
  os.Getenv("AILANG_...")     "Build: foo [briefId=abc123]"
```

### Message Payload Format

The website-builder sends messages with JSON payload containing:
```json
{
  "siteSlug": "my-website",
  "briefId": "brief-abc123",
  "directive": "Build the landing page..."
}
```

The daemon task polling extracts `siteSlug` and `briefId` from the payload and stores them on the `TaskRecord` for dispatch.

## Files to Modify

- `internal/coordinator/cloud_dispatcher.go` — Add `SiteSlug`, `BriefID` fields to `DispatchParams` (~2 LOC)
- `internal/coordinator/store.go` — Add `SiteSlug`, `BriefID` fields to `TaskRecord` (~2 LOC)
- `internal/coordinator/daemon_tasks_polling.go` — Parse payload JSON for siteSlug/briefId (~15 LOC)
- `internal/dispatch/cloudrun/dispatcher.go` — Inject `AILANG_SITE_SLUG`, `AILANG_BRIEF_ID` env vars (~10 LOC)
- `cmd/ailang/coordinator_cloud.go` — Read env vars, format commit message (~15 LOC)

**Estimated total:** ~45 LOC

## Success Criteria

- [ ] `siteSlug` and `briefId` from message payload reach Cloud Run Job as env vars
- [ ] Commit message includes `Build: {siteSlug} [briefId={briefId}]` when available
- [ ] Fallback to existing format when siteSlug/briefId not present
- [ ] `make test` passes
- [ ] `make lint` clean

---

**Document created**: 2026-03-16
