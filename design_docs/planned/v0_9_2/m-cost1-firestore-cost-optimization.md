# M-COST1: Firestore Cost Optimization

**Status**: Planned
**Target**: v0.9.2
**Priority**: P0 (Critical)
**Estimated**: 5 days (4 tiers)
**Dependencies**: None
**Author**: Mark + Claude
**Created**: 2026-03-20

---

## Executive Summary

The AILANG coordinator's Firestore usage has critical cost issues caused by full collection scans on hot paths. `GetCostByProvider()` reads ALL tasks on every task dispatch — with 100 tasks that's 100 Firestore reads per new task. `GetTaskStats()` does the same for the status API. Dashboard endpoints and the stale task detector compound the problem. Fix via in-memory counters, materialized aggregates, batch reads, and caching.

---

## Problem Statement

### Root Cause

The Firestore storage layer was ported from SQLite without adapting to Firestore's pricing model. SQLite full-table scans are cheap (local disk). Firestore charges **per document read** — every `iter.Next()` in a collection scan is a billed read.

### Cost Model

| Operation | Firestore Cost |
|-----------|---------------|
| Document read | $0.06 per 100K reads |
| Document write | $0.18 per 100K writes |
| Document delete | $0.02 per 100K deletes |

### Measured Impact (per coordinator poll cycle, 30s interval)

| Function | Collection | Reads/Cycle | Reads/Hour | Severity |
|----------|-----------|-------------|------------|----------|
| `GetCostByProvider()` | tasks | N (all tasks) | 120N | CRITICAL |
| `GetTaskStats()` | tasks | N (all tasks) | 120N | HIGH |
| `GetExecTaskHierarchyWithMessages()` | obs_tasks | N+1 per exec | Variable | HIGH |
| `GetDistinctWorkspaces()` | threads | All threads | Variable | MEDIUM |
| `GetThreadAggregateStats()` | threads | All threads | Variable | MEDIUM |
| `GetMetricsSummary()` | obs_spans + obs_workspaces + obs_tasks | All spans | Variable | MEDIUM |
| `GetMessageFlowEdges()` | inbox | All messages | Variable | MEDIUM |
| `GetActiveAgents()` | inbox | All messages | Variable | MEDIUM |
| `detectAndMarkStale()` | tasks (filtered) | Queued+Running | ~60/hour | MEDIUM |

With 100 tasks accumulating, `GetCostByProvider()` alone generates **12,000 reads/hour** — called on every budget check before task dispatch.

### Contrast with SQLite Store

The SQLite store at `internal/coordinator/store_sqlite.go:501-528` already uses `SELECT SUM(cost) ... GROUP BY provider` — a single efficient query. The Firestore store at `internal/storage/firestore/coordinator.go:458-480` iterates every document client-side. This pattern repeats across all aggregate functions.

---

## Solution Design

### Tier 1 — Immediate Wins (Days 1-2)

#### 1.1 In-Memory Cost Counter (replaces GetCostByProvider full scan)

**File**: `internal/storage/firestore/coordinator.go`

Replace the full collection scan with an in-memory counter that:
- Loads once on startup from a `_meta/cost_counters` document
- Updates in-memory on every `MarkTaskCompleted()` call
- Syncs to Firestore every 5 minutes via a background goroutine
- Falls back to a single aggregation query if the counter doc doesn't exist yet

```go
type CoordinatorStore struct {
    client *Client

    // In-memory cost tracking
    costMu       sync.RWMutex
    costCounters map[string]float64 // provider -> total cost
    costDirty    bool
}

func (s *CoordinatorStore) GetCostByProvider() (map[string]float64, error) {
    s.costMu.RLock()
    defer s.costMu.RUnlock()
    result := make(map[string]float64, len(s.costCounters))
    for k, v := range s.costCounters {
        result[k] = v
    }
    return result, nil
}

func (s *CoordinatorStore) addCost(provider string, cost float64) {
    s.costMu.Lock()
    s.costCounters[provider] += cost
    s.costDirty = true
    s.costMu.Unlock()
}
```

**Reads saved**: N reads per dispatch → 0 reads per dispatch. 1 read every 5 min for sync verification.

#### 1.2 Materialized Task Counters (replaces GetTaskStats full scan)

**File**: `internal/storage/firestore/coordinator.go`

Create a `_meta/task_stats` document updated atomically on each state transition:

```go
func (s *CoordinatorStore) MarkTaskCompleted(ctx context.Context, id string, ...) error {
    // Existing: update the task document
    // New: atomically increment counters in _meta/task_stats
    batch := s.client.Batch()
    batch.Update(s.client.Doc(collTasks, id), updates)
    batch.Update(s.client.Doc("_meta", "task_stats"), []firestore.Update{
        {Path: "completed", Value: firestore.Increment(1)},
        {Path: "running", Value: firestore.Increment(-1)},
        {Path: "total_cost", Value: firestore.Increment(cost)},
    })
    _, err := batch.Commit(ctx)
    return err
}

func (s *CoordinatorStore) GetTaskStats(ctx context.Context) (*coordinator.TaskStats, error) {
    doc, err := s.client.Doc("_meta", "task_stats").Get(ctx)
    // 1 read instead of N reads
}
```

**Reads saved**: N reads per status call → 1 read.

#### 1.3 Cache Stale Task Detector Results (90s TTL)

**File**: `internal/coordinator/stale_task_detector.go`

The detector runs every 2 minutes. Cache results for 90 seconds so consecutive checks within the same window reuse the query result:

```go
type StaleTaskDetector struct {
    // existing fields...
    cacheMu     sync.RWMutex
    cachedTasks []*TaskRecord
    cacheExpiry time.Time
}
```

**Reads saved**: Eliminates redundant queries within the 2-minute cycle.

---

### Tier 2 — N+1 and Aggregation Fixes (Day 3)

#### 2.1 Batch GetAll() for Exec Hierarchy

**File**: `internal/storage/firestore/observatory_aggregates.go:190-251`

Replace the N+1 loop:

```go
// BEFORE: N individual GetTask() calls
for _, exec := range execs {
    task, err := s.GetTask(ctx, exec.TaskID)  // 1 read each
}

// AFTER: Single batch read
var refs []*firestore.DocumentRef
for _, exec := range execs {
    refs = append(refs, s.client.Doc(collObsTasks, exec.TaskID))
}
docs, err := s.client.GetAll(ctx, refs)  // 1 batch read
```

**Reads saved**: N reads → 1 batch read (Firestore still charges per doc, but the round-trip latency drops from N to 1).

#### 2.2 Firestore Native Aggregation Queries

**Files**: `messaging.go`, `messaging_inbox.go`, `observatory_aggregates.go`

Use Firestore's server-side `Count()` and `Sum()` instead of client-side iteration:

```go
// BEFORE: Read all docs, count client-side
iter := s.client.Collection(collThreads).Documents(ctx)
count := 0
for { doc, err := iter.Next(); ... count++ }

// AFTER: Server-side count (billed at 1/1000th the cost)
result, err := s.client.Collection(collThreads).
    NewAggregationQuery().
    WithCount("total").
    Get(ctx)
```

Firestore aggregation queries are billed at **1 read per 1000 documents matched** — a 1000x cost reduction for counting operations.

**Applicable functions**:
- `GetDistinctWorkspaces()` — can use a grouped count or maintain a `_meta/workspaces` set
- `GetThreadAggregateStats()` — use `Count()` with status filter
- `GetMetricsSummary()` — use `Sum()` for tokens/cost, `Count()` for totals
- `GetMessageFlowEdges()` — maintain materialized edge counts
- `GetActiveAgents()` — use filtered `Count()` per inbox

#### 2.3 Add Missing Composite Indexes

Review and add Firestore indexes for approval queries that currently fall back to client-side filtering.

---

### Tier 3 — Dashboard Optimization (Day 4)

#### 3.1 Materialized Dashboard Stats

Create a `_meta/dashboard_stats` document refreshed by a background goroutine every 5 minutes:

```go
type DashboardStats struct {
    TotalThreads     int       `firestore:"total_threads"`
    ThreadsByStatus  map[string]int `firestore:"threads_by_status"`
    ActiveAgents     []string  `firestore:"active_agents"`
    MessageFlowEdges []Edge    `firestore:"message_flow_edges"`
    LastRefreshed    time.Time `firestore:"last_refreshed"`
}
```

Dashboard API endpoints read this single document instead of scanning collections.

#### 3.2 TTL on Dashboard Endpoint Caches

Add in-memory caching at the HTTP handler layer:

| Endpoint | Cache TTL |
|----------|-----------|
| `/api/status` | 30s |
| `/api/dashboard/stats` | 60s |
| `/api/observatory/metrics` | 120s |
| `/api/observatory/hierarchy` | 60s |
| `/api/messaging/flow` | 300s |

---

### Tier 4 — Architecture (Day 5)

#### 4.1 Document TTL / Auto-Cleanup

Old tasks, spans, and messages grow forever. Add a cleanup goroutine:

```go
func (s *CoordinatorStore) CleanupOldDocs(ctx context.Context, maxAge time.Duration) error {
    cutoff := time.Now().Add(-maxAge)
    // Delete completed tasks older than cutoff
    iter := s.client.Collection(collTasks).
        Where("status", "==", "completed").
        Where("completed_at", "<", cutoff).
        Documents(ctx)
    // Batch delete...
}
```

Default retention: 30 days for dev, 90 days for production.

#### 4.2 Dev Mode Optimizations

Add a `dev_mode` flag to the coordinator config:

```yaml
coordinator:
  dev_mode: true
  poll_interval: 5m        # vs 30s in production
  disable_stale_detector: true
  disable_approval_watcher: true
  firestore_emulator: true  # Use gcloud emulators firestore start
```

#### 4.3 Firestore Real-Time Listeners (Future)

Replace polling with Firestore `OnSnapshot()` listeners for the main daemon loop. This eliminates periodic full reads entirely — the listener only transfers changed documents.

**Not in this sprint** — requires significant refactoring of the daemon event loop. Captured here for future reference.

---

## Files to Modify

| File | Changes | Est. LOC |
|------|---------|----------|
| `internal/storage/firestore/coordinator.go` | In-memory counters, materialized stats, batch writes | +120, -40 |
| `internal/storage/firestore/observatory_aggregates.go` | Batch GetAll, native aggregation | +60, -30 |
| `internal/storage/firestore/messaging.go` | Native aggregation queries | +40, -30 |
| `internal/storage/firestore/messaging_inbox.go` | Native aggregation queries | +40, -30 |
| `internal/coordinator/stale_task_detector.go` | TTL cache | +25 |
| `internal/coordinator/daemon.go` | Start cost sync goroutine, dashboard refresh | +30 |
| `internal/coordinator/store.go` | Interface additions (if needed) | +5 |
| `internal/server/handlers_dashboard.go` (or equivalent) | HTTP-layer caching | +40 |

**Total**: ~+360, -130 LOC

---

## Migration Strategy

1. **Counter bootstrap**: On first startup after upgrade, if `_meta/cost_counters` doesn't exist, do ONE full scan to populate it, then never again
2. **Backward compatible**: All new fields use `firestore.Increment()` — safe for concurrent writers
3. **No schema migration**: Uses separate `_meta/` documents, not changes to existing task documents
4. **Rollback**: Delete `_meta/*` documents → falls back to full scan behavior

---

## Success Criteria

- [ ] `GetCostByProvider()` does 0 Firestore reads on the hot path (reads from memory)
- [ ] `GetTaskStats()` does 1 Firestore read instead of N
- [ ] `GetExecTaskHierarchyWithMessages()` uses batch read (1 round-trip)
- [ ] Dashboard endpoints serve from cached/materialized data
- [ ] Stale task detector uses 90s TTL cache
- [ ] Firestore read count drops >90% under typical coordinator load
- [ ] All existing tests pass
- [ ] New tests for counter sync, cache TTL, and batch read behavior
- [ ] Local dev mode with emulator documented and working
- [ ] CHANGELOG.md updated

---

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Counter drift (in-memory vs Firestore) | Medium | Low | Periodic full-rescan reconciliation every 24h; sync on graceful shutdown |
| Counter lost on crash | Low | Low | Counters are optimization only — next startup bootstraps from full scan |
| Firestore aggregation query limitations | Low | Medium | Fall back to materialized `_meta` docs if aggregation API doesn't support needed groupings |
| Concurrent counter updates (multi-instance) | Low | Medium | Use `firestore.Increment()` for atomic updates; in-memory counter is per-instance |

---

## Related Documents

- [M-PERF-OBSERVATORY: Tiered Loading](../../implemented/v0_9_0/m-perf-observatory-tiered-loading.md) — Similar pattern of eliminating full scans, but for SQLite/observatory
- [M-PERF3: Performance Quick Wins](../../implemented/v0_8_1/m-perf3-performance-quick-wins.md) — Prior performance optimization work

---

## Axiom Compliance

| Axiom | Score | Notes |
|-------|-------|-------|
| A1: Determinism | +1 | Counters converge to correct values; no non-determinism introduced |
| A2: Explicit Effects | +1 | Cache/counter side effects are internal optimization, not user-visible |
| A3: Type Safety | 0 | N/A — infrastructure change |
| A4: AI-Friendly | +1 | Reduces coordinator overhead, making AI agent execution faster |
| A5: Compositionality | 0 | N/A |
| A6: Error Handling | +1 | Fallback to full scan on counter miss (fail-safe, not fail-silent) |
| A7: Transparency | +1 | Budget checks return same results, just faster |
| **Net Score** | **+5** | Passes (threshold: +2) |

---

## Timeline

| Day | Work | Milestone |
|-----|------|-----------|
| 1 | In-memory cost counter + sync goroutine + tests | Tier 1a |
| 2 | Materialized task stats + stale detector cache + tests | Tier 1b |
| 3 | Batch GetAll + native aggregation queries + tests | Tier 2 |
| 4 | Dashboard materialization + HTTP caching + tests | Tier 3 |
| 5 | Document TTL + dev mode + emulator docs + final verification | Tier 4 |
