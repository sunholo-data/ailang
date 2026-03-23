# M-COST2: Dashboard Firestore Read Optimization

**Status**: Planned
**Target**: v0.9.8
**Priority**: P0 (Critical — cost)
**Estimated**: 3 days (3 phases)
**Dependencies**: None (independent of M-COST1, but complementary)
**Author**: Mark + Claude
**Created**: 2026-03-23

---

## Executive Summary

The AILANG dashboard is generating **35 million Firestore QUERY reads per day** in the dev environment, costing ~$21/day ($630/month) — on a system budgeted for ~$60/month total. The root cause is aggressive frontend polling (10s intervals) combined with short backend cache TTLs (5-30s) and unbounded collection scans (no `.Limit()`). Fix via increased polling/cache intervals (Phase 1), bounded queries with COUNT aggregation (Phase 2), and a materialized stats document (Phase 3). Phase 1 alone is expected to achieve a 10x reduction by changing 4 numbers.

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No change to determinism — caching is transparent |
| A2: Replayability | 0 | No impact on traces |
| A3: Effect Legibility | 0 | No new side effects |
| A4: Explicit Authority | 0 | No authority changes |
| A5: Bounded Verification | 0 | No verification changes |
| A6: Safe Concurrency | 0 | No concurrency model changes |
| A7: Machines First | +1 | Reduces infrastructure cost for AI agent execution |
| A8: Minimal Syntax | 0 | No syntax changes |
| A9: Cost Visibility | +1 | Directly reduces and makes cloud costs sustainable |
| A10: Composability | 0 | No composability changes |
| A11: Structured Failure | +1 | Short-circuits unsupported breakdown queries instead of logging warnings |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +3** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Not optimizing for human convenience over machine analysis

---

## Problem Statement

The dashboard is generating **35 million Firestore QUERY reads per day** in dev, costing ~$21/day ($630/month).

### Evidence

```
Firestore QUERY reads (24h sample, 2026-03-23):
  Hour        Reads
  09:00    4,066,620
  10:00    8,487,912    ← escalating as dashboard stays open
  11:00   12,352,992
  ─────────────────
  24h total: 35,100,174

Pricing: $0.06 per 100K reads = ~$21/day
```

Even with the local coordinator killed, Cloud Run dashboard instances continued generating ~200K reads/minute. The coordinator at zero instances contributed nothing — **the dashboard is the sole source**.

**Current State:**
- Dashboard polls every 10s for stats, 30s for breakdowns and budget
- Go backend caches responses for only 5-60s
- Firestore layer caches stats for only 30s
- Every cache miss triggers a full collection scan with no `.Limit()`
- Each full scan of `tasks` + `obs_spans` ≈ 200K document reads
- 2 Cloud Run dashboard instances = doubled reads

**Impact:**
- $630/month on a $60/month budget (10.5x overspend)
- Costs escalate linearly with dashboard uptime
- Affects only dev environment currently but will hit prod at launch

### Root Cause Chain

```
React frontend (10s polling)
  → GET /api/controlplane/stats          every 10 seconds
  → GET /api/controlplane/stats/breakdown every 30 seconds
  → GET /api/budget/status               every 30 seconds
    → Go backend (5s response cache)
      → Firestore fullScanTaskStats()    (30s in-memory cache)
      → Firestore GetMetricsSummary()
      → Firestore ListTasks()
        → Each call scans ENTIRE collections (no .Limit())
```

**The multiplication effect:**
- **10-second poll** with **5-second response cache** = cache miss every other request
- **30-second Firestore cache** on `fullScanTaskStats()` = 2 full collection scans/minute
- Each full scan of `tasks` + `obs_spans` ≈ 200K document reads
- 2 dashboard instances = doubled reads

### Current Architecture

| Layer | TTL | File | Line |
|-------|-----|------|------|
| React `useControlPlaneStats` | `10,000ms` | `ui/src/features/controlplane/hooks/useControlPlaneStats.ts` | 73 |
| React `useBreakdownData` | `30,000ms` | `ui/src/features/controlplane/hooks/useBreakdownData.ts` | 40 |
| React `useBudgetStatus` | `30,000ms` | `ui/src/hooks/useBudgetStatus.ts` | 68 |
| Go response cache (stats) | `5s` | `internal/server/server.go` | 177 |
| Go response cache (breakdown) | `60s` | `internal/server/server.go` | 178 |
| Firestore `statsCacheTTL` | `30s` | `internal/storage/firestore/coordinator.go` | 25 |

Note: `ControlPlane.tsx:246` already pauses polling when the tab is hidden (`isVisible ? 10000 : 0`). This helps but doesn't cover the common case of a dashboard open in a visible browser tab.

---

## Goals

**Primary Goal:** Reduce Firestore reads from 35M/day to <500K/day, bringing monthly cost from $630 to under $10.

**Success Metrics:**
- Phase 1: <500K reads/hour with dashboard open (currently ~1.5M/hour)
- Phase 1+2: <200K reads/day total
- Monthly Firestore cost under $10
- No visible impact on dashboard responsiveness (stats are minutes-stale anyway)

---

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Polling interval values (60s/120s) | Directly determines read rate; too slow = stale UX, too fast = cost | human | design | low |
| Whether to use Firestore COUNT aggregation vs materialized docs | Determines long-term architecture for stats | agent | compile | med |
| Whether Phase 3 materialized stats doc uses triggers or in-code updates | Affects reliability and complexity | agent | compile | med |
| Short-circuit breakdown for Firestore backend | Changes API behavior (returns empty instead of querying) | human | design | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] Polling intervals approved (60s stats, 120s breakdown/budget)
- [x] Short-circuit breakdown for Firestore backends approved
- [ ] Confirm no downstream consumers depend on <10s stats freshness

---

## Solution Design

### Overview

Three-phase approach: tune existing numbers (Phase 1), bound queries and use efficient aggregation (Phase 2), then architect away the polling model (Phase 3). Phase 1 ships immediately and provides 10x reduction with zero risk.

### Architecture

**Phase 1** targets the three cache layers (React → Go → Firestore) by widening all TTLs:

```
React (60-120s poll) → Go (30-300s cache) → Firestore (300s cache)
                                                ↓
                                          Full scan only every 5 min
                                          (vs every 15 seconds today)
```

**Phase 2** ensures that when scans DO happen, they're bounded and use efficient Firestore operations.

**Phase 3** eliminates polling entirely with a materialized stats document.

---

### Phase 1: Quick Wins (~2 hours, 10x reduction)

No architecture changes. Change 6 numbers across 5 files.

#### 1.1 Increase React polling intervals

| Hook | Current | Proposed | Rationale |
|------|---------|----------|-----------|
| `useControlPlaneStats` | 10s | 60s | Stats don't change faster than task completion (~minutes) |
| `useBreakdownData` | 30s | 120s | Cost breakdowns change even slower |
| `useBudgetStatus` | 30s | 120s | Budget changes only on task completion |

**Files:**
- `ui/src/features/controlplane/hooks/useControlPlaneStats.ts:73` — `10000` → `60000`
- `ui/src/features/controlplane/hooks/useBreakdownData.ts:40` — `30000` → `120000`
- `ui/src/hooks/useBudgetStatus.ts:68` — `30000` → `120000`
- `ui/src/components/HeaderStats.tsx:11-12` — update if hardcoded there too

#### 1.2 Increase Go response cache TTLs

| Cache | Current | Proposed |
|-------|---------|----------|
| `pollingCache` | 5s | 30s |
| `breakdownCache` | 60s | 300s |

**File:** `internal/server/server.go:177-178`

#### 1.3 Increase Firestore stats cache TTL

| Cache | Current | Proposed |
|-------|---------|----------|
| `statsCacheTTL` | 30s | 300s (5 min) |

**File:** `internal/storage/firestore/coordinator.go:25`

#### 1.4 Short-circuit unsupported breakdown queries

The `/api/controlplane/stats/breakdown` endpoint currently logs `"Observatory backend does not support breakdown queries"` for Firestore backends but still processes the request. Return early with a 200 + empty breakdown response.

**File:** `internal/server/handlers_controlplane_stats.go:267-273`

```go
// Before: logs warning, continues processing
// After: return immediately with empty breakdown
if !supportsBreakdown {
    respondJSON(w, http.StatusOK, BreakdownResponse{})
    return
}
```

**Expected impact:** ~10x reduction → ~3.5M reads/day → ~$2/day

---

### Phase 2: Bounded Queries (~4 hours, additional 5-10x reduction)

#### 2.1 Add `.Limit()` to all unbounded collection scans

| Function | File | Current | Proposed |
|----------|------|---------|----------|
| `fullScanTaskStats()` | `internal/storage/firestore/coordinator.go:324` | No limit | `.Limit(10000)` + warning if hit |
| `fullScanCostByProvider()` | `internal/storage/firestore/coordinator.go:157` | No limit | `.Limit(10000)` |
| `GetDistinctWorkspaces()` | `internal/storage/firestore/messaging.go:270` | No limit | `.Limit(5000)` |
| `GetThreadAggregateStats()` | `internal/storage/firestore/messaging.go:302` | No limit | `.Limit(5000)` |
| `RecalculateTaskAggregates()` | `internal/storage/firestore/observatory_spans.go:99` | No limit per task | `.Limit(1000)` per task |
| `ListTraces()` | `internal/storage/firestore/observatory_spans.go:173` | Optional limit | Default `.Limit(100)` |

#### 2.2 Use Firestore COUNT aggregation instead of full scans

For stats that only need counts (total tasks, tasks by status), use Firestore's native `Count()` aggregation query which costs 1/1000th of document reads.

```go
// Before: reads 50K documents to count them
iter := s.client.Collection("tasks").Documents(ctx)
for { doc, err := iter.Next(); count++ }

// After: 1 aggregation query = charged as 1 read per 1000 docs
q := s.client.Collection("tasks").Where("status", "==", "running")
results, _ := q.NewAggregationQuery().WithCount("count").Get(ctx)
```

**Applicable functions in `internal/storage/firestore/coordinator.go`:**
- `fullScanTaskStats()` — replace count loops with `Count()` per status
- `fullScanCostByProvider()` — replace sum loops with `Sum("cost")` per provider

**Expected impact:** Phase 1+2 → ~200K reads/day → ~$0.12/day

---

### Phase 3: Architecture (longer-term)

#### 3.1 Materialized stats document

Instead of scanning collections to compute stats, maintain a single `_meta/dashboard_stats` document updated on each task state transition via `firestore.Increment()`. Dashboard reads 1 document instead of scanning 50K.

```go
type DashboardStats struct {
    TotalTasks      int                `firestore:"total_tasks"`
    TasksByStatus   map[string]int     `firestore:"tasks_by_status"`
    CostByProvider  map[string]float64 `firestore:"cost_by_provider"`
    LastRefreshed   time.Time          `firestore:"last_refreshed"`
}
```

This overlaps with M-COST1's `_meta/task_stats` proposal — should be unified into one implementation.

#### 3.2 Environment-aware polling

```typescript
const baseInterval = import.meta.env.VITE_ENV === 'prod' ? 30000 : 120000;
```

Dev environments should poll less aggressively than prod.

#### 3.3 WebSocket/SSE for real-time updates

Replace polling entirely with server-sent events. The server pushes updates only when data actually changes, eliminating all unnecessary reads. See also [M-DASHBOARD-PUBSUB-EVENTS](../v0_9_3/m-dashboard-pubsub-events.md) which covers the event transport layer — stats could piggyback on that infrastructure.

**Expected impact:** Phase 1+2+3 → ~10K reads/day → ~$0.006/day

---

## Implementation Plan

**Phase 1: Quick Wins** (~2 hours)
- [ ] Update React polling intervals in 3 hooks
- [ ] Check `HeaderStats.tsx` for hardcoded intervals
- [ ] Update Go response cache TTLs in `server.go`
- [ ] Update Firestore `statsCacheTTL` in `coordinator.go`
- [ ] Short-circuit breakdown endpoint for Firestore backend
- [ ] Deploy and verify read rate drops

**Phase 2: Bounded Queries** (~4 hours)
- [ ] Add `.Limit()` to all 6 unbounded collection scans
- [ ] Add log warning when limit is hit (indicates data growth)
- [ ] Replace `fullScanTaskStats()` count loops with `Count()` aggregation
- [ ] Replace `fullScanCostByProvider()` sum loops with `Sum()` aggregation
- [ ] Unit tests for bounded query behavior
- [ ] Deploy and verify further reduction

**Phase 3: Architecture** (future sprint, after Phase 2 impact measured)
- [ ] Implement materialized `_meta/dashboard_stats` document (coordinate with M-COST1)
- [ ] Add environment-aware polling
- [ ] Evaluate WebSocket/SSE (coordinate with M-DASHBOARD-PUBSUB-EVENTS)

### Files to Modify/Create

**Modified files (Phase 1):**
- `ui/src/features/controlplane/hooks/useControlPlaneStats.ts` — polling interval, ~1 LOC
- `ui/src/features/controlplane/hooks/useBreakdownData.ts` — polling interval, ~1 LOC
- `ui/src/hooks/useBudgetStatus.ts` — polling interval, ~1 LOC
- `ui/src/components/HeaderStats.tsx` — polling interval if hardcoded, ~1 LOC
- `internal/server/server.go` — cache TTLs, ~2 LOC
- `internal/storage/firestore/coordinator.go` — stats cache TTL, ~1 LOC
- `internal/server/handlers_controlplane_stats.go` — short-circuit breakdown, ~5 LOC

**Modified files (Phase 2):**
- `internal/storage/firestore/coordinator.go` — `.Limit()` + `Count()`/`Sum()` aggregation, ~+40 -20 LOC
- `internal/storage/firestore/messaging.go` — `.Limit()`, ~+10 -5 LOC
- `internal/storage/firestore/observatory_spans.go` — `.Limit()`, ~+10 -5 LOC

**Total: ~+70, -30 LOC across 10 files**

---

## Cost Projection

| State | Daily Reads | Daily Cost | Monthly Cost |
|-------|------------|------------|--------------|
| **Current** (no optimization) | 35M | $21 | $630 |
| **Phase 1** (polling + cache) | ~3.5M | $2.10 | $63 |
| **Phase 1 + 2** (bounded + COUNT) | ~200K | $0.12 | $3.60 |
| **Phase 1 + 2 + 3** (materialized) | ~10K | $0.006 | $0.18 |

---

## Examples

### Example 1: Stats Polling (Before vs After Phase 1)

**Before (10s poll, 5s cache):**
```
t=0s   React polls → cache MISS → Firestore scan (200K reads)
t=5s   cache expires
t=10s  React polls → cache MISS → Firestore scan (200K reads)
t=15s  cache expires
t=20s  React polls → cache MISS → Firestore scan (200K reads)
= 600K reads/minute
```

**After (60s poll, 30s cache):**
```
t=0s   React polls → cache MISS → Firestore scan (200K reads)
t=30s  cache expires (no poll yet)
t=60s  React polls → cache MISS → Firestore scan (200K reads)
= 200K reads/minute (3x reduction from polling alone)
```

Combined with 300s Firestore cache: only 1 scan per 5 minutes = ~40K reads/minute → **15x reduction**.

### Example 2: Breakdown Short-Circuit

**Before:**
```
React polls /api/controlplane/stats/breakdown every 30s
→ Go handler logs "Observatory backend does not support breakdown queries"
→ Still processes request, potentially triggering Firestore reads
→ Returns empty response anyway
```

**After:**
```
React polls /api/controlplane/stats/breakdown every 120s
→ Go handler detects Firestore backend
→ Returns 200 + empty BreakdownResponse immediately
→ Zero Firestore reads
```

---

## Success Criteria

- [ ] Firestore QUERY reads < 500K/hour with dashboard open (Phase 1)
- [ ] Firestore QUERY reads < 200K/day total (Phase 1+2)
- [ ] Monthly Firestore cost < $10 (Phase 1+2)
- [ ] No visible degradation in dashboard responsiveness
- [ ] Breakdown endpoint returns immediately for Firestore backends
- [ ] All unbounded collection scans have `.Limit()` with overflow warning
- [ ] All tests passing (`make test`)
- [ ] CHANGELOG.md updated

---

## Testing Strategy

**Unit tests:**
- Verify `.Limit()` is applied to Firestore queries (mock client)
- Verify `Count()` aggregation returns correct results
- Verify breakdown short-circuit returns empty response for Firestore backend

**Integration tests:**
- Deploy Phase 1, monitor Firestore reads via Cloud Monitoring

**Manual testing:**
```bash
# Check hourly read rate after Phase 1 deployment
curl -s -H "Authorization: Bearer $(gcloud auth print-access-token)" \
  "https://monitoring.googleapis.com/v3/projects/ailang-multivac-dev/timeSeries?filter=metric.type%3D%22firestore.googleapis.com%2Fdocument%2Fread_count%22%20AND%20metric.labels.type%3D%22QUERY%22&interval.startTime=$(date -u -v-1H '+%Y-%m-%dT%H:%M:%SZ')&interval.endTime=$(date -u '+%Y-%m-%dT%H:%M:%SZ')&aggregation.alignmentPeriod=3600s&aggregation.perSeriesAligner=ALIGN_SUM"
```

Target: <500K reads/hour with dashboard open.

---

## Deferred Decisions

The following are intentionally left open for the implementer:

- Exact `.Limit()` values per collection (10000/5000/1000 suggested, agent may adjust based on actual collection sizes) — agent may choose
- Whether `Count()` aggregation needs per-status breakdown or just totals — agent may choose based on what the handler actually uses
- Whether Phase 3 materialized doc should be shared with M-COST1's `_meta/task_stats` — deferred until Phase 2 impact measured

---

## Non-Goals

**Not attempted in this feature:**
- Replacing Firestore with a different backend — out of scope, different initiative
- Optimizing write costs — reads are 99%+ of the cost problem
- Dashboard feature changes — this is purely a cost optimization, no UX changes
- Production environment tuning — dev is the current problem; prod tuning comes later

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| 60s polling feels too stale to users | Low | Stats are already minutes-stale (tasks take minutes); can adjust if feedback received |
| `.Limit()` truncates actual data | Medium | Log warning when limit is hit; set limits well above current collection sizes |
| `Count()` aggregation API differences from documented behavior | Low | Fall back to bounded scan + client-side count |
| Phase 1 doesn't achieve 10x (other hidden read sources) | Medium | Cloud Monitoring will reveal exact source; Phase 2 provides additional reduction |
| Multiple dashboard instances compete for Firestore cache | Low | Each instance has its own in-memory cache; Firestore reads are per-instance |

---

## Implementation Order

1. **Phase 1.1-1.3** — Change 6 numbers (polling intervals + cache TTLs). Ship immediately.
2. **Phase 1.4** — Short-circuit breakdown. Ship with Phase 1.
3. **Phase 2.1** — Add limits. Low risk, ship next sprint.
4. **Phase 2.2** — COUNT/SUM aggregation. Requires testing, ship with 2.1.
5. **Phase 3** — Plan after Phase 2 impact is measured.

---

## Related Documents

<!-- Auto-populated by Ollama neural search on "dashboard firestore optimization" -->

**Implemented (informs design):**
- [M-PERF-OBSERVATORY: Tiered Loading](../../implemented/v0_9_0/m-perf-observatory-tiered-loading.md) (0.47) — Similar pattern of eliminating full scans and adding bounded queries
- [M-CONTROL-PLANE: Interactive Filtering](../../implemented/v0_7_0/m-control-plane-interactive-filtering.md) (0.48) — Dashboard polling architecture

**Planned (check for overlap):**
- [M-COST1: Firestore Cost Optimization](../v0_9_2/m-cost1-firestore-cost-optimization.md) (0.56) — **Complementary**: covers coordinator-side Firestore optimizations (in-memory counters, materialized task stats). Phase 3 of this doc should unify with M-COST1's `_meta/task_stats` and `_meta/cost_counters`.
- [M-COST1 Sprint Plan](../v0_9_2/m-cost1-firestore-cost-optimization-sprint-plan.md) (0.49)
- [M-DASHBOARD-PUBSUB-EVENTS](../v0_9_3/m-dashboard-pubsub-events.md) (related) — Phase 3.3 WebSocket/SSE could leverage this Pub/Sub event infrastructure
- [M-CLOUD-OBSERVATORY](../v0_10_0/m-cloud-observatory.md) (0.46) — Cloud observatory architecture

---

## References

- [Design Axioms](/docs/references/axioms) - The 12 non-negotiable principles
- [Firestore Pricing](https://cloud.google.com/firestore/pricing) — $0.06 per 100K reads
- [Firestore Aggregation Queries](https://cloud.google.com/firestore/docs/aggregation-queries) — Count/Sum at 1/1000th cost
- GCP Cloud Monitoring — `firestore.googleapis.com/document/read_count` metric

---

## Future Work

- **Firestore real-time listeners** (`OnSnapshot()`) — replace polling with change streams (requires significant refactoring)
- **Per-environment cost budgets** — alert when Firestore spend exceeds thresholds
- **Dashboard cost indicator** — show current Firestore read rate in the dashboard itself
- **Unify with M-COST1** — Phase 3 materialized stats should be a single implementation covering both coordinator and dashboard read paths

---

**Document created**: 2026-03-23
**Last updated**: 2026-03-23
