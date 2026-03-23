# Sprint Plan: M-COST2 Dashboard Firestore Read Optimization

## Summary

Reduce dashboard Firestore reads from 35M/day ($630/month) to <200K/day ($3.60/month) by increasing polling/cache intervals (Phase 1) and adding bounded queries with COUNT aggregation (Phase 2). Phase 1 is a 6-number change shipping immediately; Phase 2 adds `.Limit()` and Firestore aggregation queries.

**Duration:** 1 day (Phase 1: 1 hour, Phase 2: 3 hours)
**Dependencies:** None
**Risk Level:** Low — Phase 1 is configuration-only, Phase 2 is additive

## Current Status Analysis

### Completed Recently (from M-COST1)
- ✅ In-memory cost counters: `internal/storage/firestore/coordinator.go` — eliminates `GetCostByProvider()` full scans
- ✅ Stats cache (30s TTL): `GetTaskStats()` cached, avoids every-call full scans
- ✅ Cost sync goroutine: background Firestore sync every 5 minutes
- ✅ Breakdown short-circuit: `handlers_controlplane_stats.go:267-272` already returns empty for non-SQLite backends

### Velocity
- Recent average: ~200-400 LOC/day from recent M-COST1 and M-PKG-REGISTRY work
- This sprint is unusually small: ~70 LOC total (mostly number changes)

### Remaining from Design Doc
- ⏳ Phase 1: Polling + cache interval changes (~10 LOC)
- ⏳ Phase 2: Bounded queries + COUNT aggregation (~60 LOC)
- 📋 Phase 3: Materialized stats document (deferred to future sprint, after measuring Phase 1+2 impact)

## Proposed Milestones

### Milestone 1: Increase Polling and Cache Intervals (Phase 1)

**Goal:** 10x reduction in Firestore reads by widening all cache/poll TTLs across the three-layer stack (React → Go → Firestore).

**Estimated:** ~10 LOC changes (no tests needed — these are configuration values)
**Duration:** 30 minutes

**Tasks:**
1. `ui/src/features/controlplane/hooks/useControlPlaneStats.ts:73` — change `10000` → `60000`
2. `ui/src/features/controlplane/hooks/useBreakdownData.ts:40` — change `30000` → `120000`
3. `ui/src/hooks/useBudgetStatus.ts:68` — change `30000` → `120000`
4. `ui/src/components/HeaderStats.tsx:11` — change `10000` → `60000`
5. `ui/src/components/HeaderStats.tsx:12` — change `30000` → `120000`
6. `internal/server/server.go:177` — change `5 * time.Second` → `30 * time.Second`
7. `internal/server/server.go:178` — change `60 * time.Second` → `300 * time.Second`
8. `internal/storage/firestore/coordinator.go:25` — change `statsCacheTTL = 30 * time.Second` → `300 * time.Second`

**Acceptance Criteria:**
- [ ] React stats polling at 60s (was 10s)
- [ ] React breakdown/budget polling at 120s (was 30s)
- [ ] Go response cache at 30s/300s (was 5s/60s)
- [ ] Firestore stats cache at 300s (was 30s)
- [ ] `make test` passes
- [ ] `make lint` passes
- [ ] Frontend builds without errors

**Risks:**
- Stats feel stale → Mitigation: stats are already minutes-stale (tasks take minutes to complete); 60s polling is still responsive

### Milestone 2: Bounded Queries and COUNT Aggregation (Phase 2)

**Goal:** Ensure that when Firestore scans DO happen (cache miss), they're bounded and use efficient aggregation operations. Additional 5-10x reduction.

**Estimated:** ~60 LOC implementation + ~0 new test files (existing tests cover behavior)
**Duration:** 3 hours

**Tasks:**
1. Add `.Limit(10000)` to `fullScanTaskStats()` in `internal/storage/firestore/coordinator.go:325`
2. Add `.Limit(10000)` to `fullScanCostByProvider()` in `internal/storage/firestore/coordinator.go:158`
3. Add `.Limit(5000)` to `GetDistinctWorkspaces()` in `internal/storage/firestore/messaging.go:264`
4. Add `.Limit(5000)` to `GetThreadAggregateStats()` in `internal/storage/firestore/messaging.go:296`
5. Add `.Limit(1000)` to `RecalculateTaskAggregates()` per-task query in `internal/storage/firestore/observatory_spans.go:99`
6. Add default `.Limit(100)` to `ListTraces()` in `internal/storage/firestore/observatory_spans.go`
7. Add log warning when any limit is hit (indicates collection growth needs attention)
8. Replace count loops in `fullScanTaskStats()` with Firestore `Count()` aggregation where applicable
9. Replace sum loops in `fullScanCostByProvider()` with Firestore `Sum()` aggregation where applicable

**Acceptance Criteria:**
- [ ] All Firestore collection scans have explicit `.Limit()`
- [ ] Log warning emitted when limit is hit
- [ ] `Count()` aggregation used for status counts where possible
- [ ] `make test` passes
- [ ] `make lint` passes
- [ ] No behavior change in API responses (same data, just bounded)

**Risks:**
- `.Limit()` truncates actual data → Mitigation: limits set well above current collection sizes; log warning alerts if hit
- Firestore `Count()`/`Sum()` API may differ from expected → Mitigation: fall back to bounded scan + client-side count

## Success Metrics

- Firestore QUERY reads < 500K/hour with dashboard open (Phase 1)
- Firestore QUERY reads < 200K/day total (Phase 1+2)
- Monthly Firestore cost < $10
- All tests passing: `make test` ✅
- All linting passing: `make lint` ✅
- CHANGELOG.md updated

## Dependencies

- None — this sprint is independent of M-COST1 (which is already merged)

## Verification Plan

After deploying each milestone, verify via Cloud Monitoring:
```bash
curl -s -H "Authorization: Bearer $(gcloud auth print-access-token)" \
  "https://monitoring.googleapis.com/v3/projects/ailang-multivac-dev/timeSeries?filter=metric.type%3D%22firestore.googleapis.com%2Fdocument%2Fread_count%22%20AND%20metric.labels.type%3D%22QUERY%22&interval.startTime=$(date -u -v-1H '+%Y-%m-%dT%H:%M:%SZ')&interval.endTime=$(date -u '+%Y-%m-%dT%H:%M:%SZ')&aggregation.alignmentPeriod=3600s&aggregation.perSeriesAligner=ALIGN_SUM"
```

## Open Questions

- Should we also add environment-aware base intervals (dev=120s, prod=30s)? — Deferred to Phase 3

## Notes

- Phase 1.4 (short-circuit breakdown) is already implemented at `handlers_controlplane_stats.go:267-272` — it logs and returns empty JSON for non-SQLite backends
- M-COST1 already implemented in-memory cost counters and stats caching — this sprint focuses on the frontend polling cadence and unbounded scan issues that M-COST1 didn't address
- Phase 3 (materialized stats doc, WebSocket/SSE) is explicitly deferred until Phase 1+2 impact is measured
