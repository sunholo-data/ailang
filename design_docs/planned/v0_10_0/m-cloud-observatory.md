# M-CLOUD-OBSERVATORY: Observatory Firestore Backend for Cloud Run

**Status**: Planned
**Target**: v0.10.0
**Priority**: P1 (High — blocks cloud dashboard functionality)
**Estimated**: 5-6 hours
**Dependencies**: M-CLOUD-STORAGE (v0.8.1, implemented), M-HTTP-HOOKS-CLOUD-TELEMETRY (v0.9.0, implemented)
**Author**: Claude + Mark
**Created**: 2026-03-11

---

## Executive Summary

Make the Observatory Control Plane endpoints work with the Firestore backend on Cloud Run. Currently, 6 handler type-assertions to `*observatory.SQLiteBackend` cause 503 errors on the cloud dashboard. The Firestore `ObservatoryStore` already implements most of the `Backend` interface — this doc covers the focused changes needed to close the gap.

**Scope**: Exec-hierarchy and span-hierarchy endpoints only. Analytics/history endpoints that need raw SQL are deferred to a BigQuery migration.

---

## Problem Statement

### Current: Cloud Dashboard Returns 503

The AILANG dashboard on Cloud Run uses `AILANG_STORAGE=gcp`, which creates a Firestore-backed `ObservatoryStore`. However, the exec-hierarchy handler does:

```go
// handlers_controlplane_exec_hierarchy.go:37-41
sqliteBackend, ok := s.obsBackend.(*observatory.SQLiteBackend)
if !ok {
    http.Error(w, "Exec hierarchy requires SQLite backend", http.StatusServiceUnavailable)
    return
}
```

This fails on Cloud Run because the backend is `*firestore.ObservatoryStore`, not `*observatory.SQLiteBackend`.

### Impact

- `/api/controlplane/exec-hierarchy` → **503** (blocks execution visualization)
- `/api/controlplane/span-hierarchy` → **503** (blocks span tree view)
- Dashboard shows empty control plane on Cloud Run

### Root Cause: Leaky Abstraction

The handlers bypass the `observatory.Backend` interface and reach into the concrete `*observatory.Store` (SQLite) for methods not on the interface:

| Method | On Backend Interface? | Used By |
|--------|-----------------------|---------|
| `GetExecTaskHierarchy(limit)` | Yes | exec-hierarchy |
| `GetExecTaskHierarchyWithMessages(limit)` | **No** | exec-hierarchy (include_messages mode) |
| `GetSpanHierarchy(limit)` | **No** | span-hierarchy |
| `GetToolsByTimestampRange(start, end, name)` | **No** | enrichExecHierarchy (tool display names) |

Additionally, 3 Firestore methods are stubbed with TODOs:

| Method | Current State |
|--------|---------------|
| `GetChainStatusCounts()` | Returns empty `ChainStatusCounts{}` |
| `GetChainStatsByAgent()` | Returns `nil, nil` |
| `GetSpanLitesByStageID()` | Returns empty `SpanLitePage{}` |

And 1 method has a bug:

| Method | Bug |
|--------|-----|
| `GetSpansByStageID(stageID)` | Ignores `stageID`, returns all spans |

---

## Solution Design

### Architecture

No new backends or abstractions. The fix is:
1. Expand the `Backend` interface with 3 missing methods
2. Implement those methods in the Firestore store
3. Fix the stubbed/buggy methods
4. Refactor handlers to use the interface instead of type-asserting

```
Before:
  handler → type-assert to *SQLiteBackend → call *Store method → 503 on Firestore

After:
  handler → call Backend.Method() → works with SQLite or Firestore
```

### Tiered Scope

**28 total SQLite type-assertions** exist across 7 handler files. Categorized by effort:

| Tier | Files | Assertions | Impact on Cloud | This Doc? |
|------|-------|-----------|-----------------|-----------|
| 1: Core | exec_hierarchy.go | 6 | Hard 503 | **Yes** |
| 2: Stats | stats.go, budget.go, heatmap.go, task_hierarchy.go, chains.go | ~15 | Graceful degradation (empty data) | No (future) |
| 3: Analytics | claudehistory.go, analytics.go, inbox.go | ~7 | Fundamentally local-only (raw SQL, local files) | No (BigQuery) |

Only Tier 1 is in scope.

---

## Implementation Plan

### Phase 1: Expand Backend Interface (~30 min)

**File**: `internal/observatory/backend.go`

Add 3 methods:

```go
// Under "Aggregate operations" section:
GetExecTaskHierarchyWithMessages(ctx context.Context, limit int) (*ExecHierarchyWithMessages, error)
GetSpanHierarchy(ctx context.Context, limit int) (*SpanHierarchyResult, error)
GetToolsByTimestampRange(ctx context.Context, start, end time.Time, toolName string) ([]SessionTool, error)
```

**File**: `internal/observatory/backend_sqlite.go`

Add forwarding methods that delegate to the existing `*Store` implementations:

```go
func (b *SQLiteBackend) GetExecTaskHierarchyWithMessages(ctx context.Context, limit int) (*ExecHierarchyWithMessages, error) {
    return b.store.GetExecTaskHierarchyWithMessages(limit)
}
// ... same pattern for GetSpanHierarchy, GetToolsByTimestampRange
```

Note: The existing `*Store` methods don't take `context.Context` — the forwarding methods accept it for interface consistency.

### Phase 2: Implement Firestore Methods (~2 hours)

**File**: `internal/storage/firestore/observatory_aggregates.go`

**2a. `GetExecTaskHierarchyWithMessages`**

```go
func (s *ObservatoryStore) GetExecTaskHierarchyWithMessages(ctx context.Context, limit int) (*obs.ExecHierarchyWithMessages, error) {
    // 1. Get flat exec hierarchy (reuse existing GetExecTaskHierarchy)
    // 2. Group exec spans by chain_id attribute
    // 3. For spans with chain_id: look up chain → get source_ref (message ID) → fetch message
    // 4. Return grouped by message, orphans separate
}
```

**2b. `GetSpanHierarchy`**

```go
func (s *ObservatoryStore) GetSpanHierarchy(ctx context.Context, limit int) (*obs.SpanHierarchyResult, error) {
    // 1. Fetch recent spans ordered by start_time DESC
    // 2. Build span map, identify roots (parent_span_id == "")
    // 3. Link children to parents
    // 4. Compute stats (total, cost, tokens, time range, max depth)
    // 5. Build sessions map from session_id attributes
}
```

**2c. `GetToolsByTimestampRange`**

```go
func (s *ObservatoryStore) GetToolsByTimestampRange(ctx context.Context, start, end time.Time, toolName string) ([]obs.SessionTool, error) {
    q := s.client.Collection(collObsSessionTools).
        Where("start_time", ">=", timeToFirestore(start)).
        Where("start_time", "<=", timeToFirestore(end)).
        OrderBy("start_time", firestore.Asc)
    if toolName != "" {
        q = q.Where("tool_name", "==", toolName)
    }
    // Iterate and convert
}
```

### Phase 3: Fix Stubbed & Buggy Methods (~1 hour)

**File**: `internal/storage/firestore/observatory_chains.go`

**3a. Fix `GetSpansByStageID` bug** (line 366-368 — currently ignores `stageID`):
```go
func (s *ObservatoryStore) GetSpansByStageID(ctx context.Context, stageID string) ([]*obs.Span, error) {
    iter := s.client.Collection(collObsSpans).
        Where("stage_id", "==", stageID).
        Documents(ctx)
    // ... iterate and return
}
```

**3b. Implement `GetChainStatusCounts`** (line 370-373):
```go
func (s *ObservatoryStore) GetChainStatusCounts(ctx context.Context, createdAfter *time.Time) (*obs.ChainStatusCounts, error) {
    q := s.client.Collection(collObsChains).Query
    if createdAfter != nil {
        q = q.Where("created_at", ">=", timeToFirestore(*createdAfter))
    }
    // Iterate chains, switch on status, increment counts
}
```

**3c. Implement `GetChainStatsByAgent`** (line 375-378):
```go
func (s *ObservatoryStore) GetChainStatsByAgent(ctx context.Context, createdAfter *time.Time) ([]*obs.AgentStatsResult, error) {
    // Query chain stages, aggregate by agent_id
    // Count completed/failed, sum cost/tokens/duration
}
```

**3d. Implement `GetSpanLitesByStageID`** (line 380-383):
```go
func (s *ObservatoryStore) GetSpanLitesByStageID(ctx context.Context, stageID string, limit, offset int) (*obs.SpanLitePage, error) {
    q := s.client.Collection(collObsSpans).
        Where("stage_id", "==", stageID).
        OrderBy("start_time", firestore.Asc)
    // Apply offset/limit, return lightweight span data
}
```

### Phase 4: Refactor Handlers (~45 min)

**File**: `internal/server/handlers_controlplane_exec_hierarchy.go`

Replace all 6 `*observatory.SQLiteBackend` type-assertions:

```go
// Before:
sqliteBackend, ok := s.obsBackend.(*observatory.SQLiteBackend)
if !ok {
    http.Error(w, "Exec hierarchy requires SQLite backend", http.StatusServiceUnavailable)
    return
}
hierarchy, err := sqliteBackend.Store().GetExecTaskHierarchy(limit)
enrichExecHierarchy(r.Context(), sqliteBackend.Store(), hierarchy)

// After:
if s.obsBackend == nil {
    http.Error(w, "Observatory not configured", http.StatusServiceUnavailable)
    return
}
hierarchy, err := s.obsBackend.GetExecTaskHierarchy(r.Context(), limit)
enrichExecHierarchyFromBackend(r.Context(), s.obsBackend, hierarchy)
```

Refactor `enrichExecHierarchy` signature:
```go
// Before:
func enrichExecHierarchy(ctx context.Context, store *observatory.Store, hierarchy []*observatory.ExecTaskNode)

// After:
func enrichExecHierarchyFromBackend(ctx context.Context, backend observatory.Backend, hierarchy []*observatory.ExecTaskNode)
```

### Phase 5: Verify OTLP & Hooks (~10 min)

Both already use the `Backend` interface correctly:

- **OTLP Receiver**: `NewOTLPReceiver(backend Backend)` → calls `backend.CreateSpan()` ✅
- **Claude Hooks**: Calls `UpsertSessionWithCorrelation()`, `InsertToolStart()`, `UpdateToolEnd()` — all on interface ✅

No changes needed.

---

## Firestore Indexes Required

New queries need composite indexes. Add to `firestore.indexes.json` or create manually:

| Collection | Fields | Order |
|------------|--------|-------|
| `obs_session_tools` | `start_time` | ASC |
| `obs_spans` | `stage_id`, `start_time` | ASC |
| `obs_spans` | `parent_span_id`, `start_time` | DESC |
| `obs_spans` | `chain_id`, `start_time` | DESC |

---

## Files to Modify

| File | Changes | LOC |
|------|---------|-----|
| `internal/observatory/backend.go` | Add 3 methods to Backend interface | ~10 |
| `internal/observatory/backend_sqlite.go` | Add 3 forwarding methods | ~15 |
| `internal/storage/firestore/observatory_aggregates.go` | Implement 3 new methods | ~120 |
| `internal/storage/firestore/observatory_chains.go` | Fix 3 stubbed + 1 buggy method | ~80 |
| `internal/server/handlers_controlplane_exec_hierarchy.go` | Replace 6 type-assertions, refactor enrichment | ~30 |

**Total: ~255 LOC changed**

---

## What Remains Out of Scope

| Feature | Why | Future Milestone |
|---------|-----|------------------|
| Claude History import | Local JSONL files, needs `*sql.DB` | N/A (local-only) |
| Analytics (time series, distributions) | Raw SQL via `backend.DB()` | BigQuery migration |
| Filtered metrics/breakdowns | Complex SQL aggregations | BigQuery or Firestore aggregation |
| Heatmap data | Tier 2, graceful degradation | Follow-up PR |
| Chain journey | Tier 2, returns 501 | Follow-up PR |

---

## Testing Strategy

1. **Unit tests**: New Firestore methods with mock client
2. **Interface compliance**: Ensure both SQLiteBackend and ObservatoryStore satisfy expanded Backend
3. **Regression**: `make test` — SQLite path unchanged
4. **Build**: `go build ./...`
5. **Smoke test**: Deploy to Cloud Run → `GET /api/controlplane/exec-hierarchy` returns 200

---

## Success Criteria

- [ ] `/api/controlplane/exec-hierarchy` returns 200 on Cloud Run (was 503)
- [ ] `/api/controlplane/span-hierarchy` returns 200 on Cloud Run (was 503)
- [ ] Tool enrichment works with Firestore session_tools data
- [ ] All existing tests pass (`make test`)
- [ ] Local SQLite path unchanged
- [ ] 3 stubbed Firestore methods return real data
- [ ] `GetSpansByStageID` actually filters by stageID

---

## Related Documents

- [M-CLOUD-STORAGE](../../implemented/v0_8_1/m-cloud-storage.md) — Backend abstraction layer (implemented)
- [M-HTTP-HOOKS-CLOUD-TELEMETRY](../../implemented/v0_9_0/m-http-hooks-cloud-telemetry.md) — Claude hooks → Observatory (implemented)
- [M-EXEC-HIERARCHY-REFACTOR](m-exec-hierarchy-refactor.md) — UI visualization improvements (planned, separate scope)
- [Observatory Architecture](../../implemented/v0_7_0/observatory-architecture.md) — Entity model and backend interface

---

**Document created**: 2026-03-11
**Last updated**: 2026-03-11
