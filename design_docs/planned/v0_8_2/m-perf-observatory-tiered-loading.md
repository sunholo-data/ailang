# M-PERF-OBSERVATORY: Tiered Loading for Chain Performance

**Status**: Planned
**Target**: v0.8.2
**Priority**: P0 (Critical)
**Estimated**: 10 days (~5 phases)
**Dependencies**: None
**Author**: Claude + Mark
**Created**: 2026-03-04

---

## Executive Summary

The observatory chain system has slowed to a crawl at current data scale (5.6GB, 761K spans). Root causes: N+1 query patterns, eager loading of 3.9GB of span attributes, correlated subqueries across 213K traces, and frontend polling even on hidden tabs. Fix via 5-level tiered loading architecture where data is fetched only when the user navigates to it. Expected 30-100x improvement on worst queries.

---

## Problem Statement

### Current Scale

| Metric | Value |
|--------|-------|
| observatory.db size | 5.6GB |
| Total spans | 761,270 |
| Total stages | 322 |
| Total chains | 49 |
| Distinct traces | 213,647 |
| Span `attributes` column total | 3.9GB |
| Max single span attributes | 698KB |
| Spans with attributes >= 10KB | 183,189 |
| Max spans per trace | 19,797 |

### Measured Bottlenecks

| # | Issue | Location | Impact |
|---|-------|----------|--------|
| B1 | **Stats N+1**: `handleChainsStats` fetches 1000 chains, then `GetChainStages()` per chain | `internal/server/handlers_chains.go:454-530` | 1+N queries (50+ today) |
| B2 | **Double-fetch**: `handleGetChain` loads stages via `GetChain()`, then calls `GetChainStages()` again | `internal/server/handlers_chains.go:122-130` | 2x stage queries per detail view |
| B3 | **Per-stage span loading**: `GetChainStages()` calls `GetSpansByStageID()` per stage in a loop | `internal/observatory/store_chains.go:437-443` | O(stages) queries, each loading full attributes |
| B4 | **ListTraces correlated subqueries**: 3 subqueries per trace for root span info | `internal/observatory/store_spans.go:514-549` | 3 * 213K = 639K subqueries |
| B5 | **Attribute bloat**: Every span query SELECTs `attributes, resource_attributes` | All span queries | 3.9GB read through SQLite for metadata-only views |
| F1 | **Eager full load**: `useChainData` fetches `?include_spans=true&include_sessions=true` | `ui/src/features/controlplane/hooks/useChainData.ts:218-222` | Multi-MB JSON response |
| F2 | **Aggressive polling**: 5 hooks poll (5s/10s/30s), even when tab hidden | `ui/src/features/controlplane/ControlPlane.tsx:239-248` | Constant CPU/network |

---

## Solution: 5-Level Tiered Loading

**Core principle:** Never load data until the user navigates to it.

```
User opens dashboard
  └─ L0: Chain list (metadata only)          ~2KB/chain    1 query
       └─ User clicks chain
          └─ L1: Chain + stage metadata       ~5KB          1 query
               └─ User clicks stage
                  └─ L2: Stage spans (lite)   ~50KB/200     1 query
                       └─ User clicks span
                          └─ L3: Full span    1-700KB       1 query
               └─ User clicks "Chat" tab
                  └─ L4: Chat context         ~20KB/50msg   1 query
```

Each level is a separate API call with bounded response size. The UI renders progressively — chain header and stage cards appear instantly from L1, span trees load per-stage from L2.

### New Type: SpanLite

The key abstraction — a span without the 3.9GB `attributes`/`resource_attributes` columns:

```go
// SpanLite contains span metadata sufficient for tree, timeline, and waterfall views.
// Excludes attributes and resource_attributes (which account for 3.9GB of data).
type SpanLite struct {
    ID            string    `json:"id"`
    TraceID       string    `json:"trace_id"`
    ParentSpanID  string    `json:"parent_span_id,omitempty"`
    Name          string    `json:"name"`
    Kind          SpanKind  `json:"kind"`
    Status        string    `json:"status"`
    StatusMessage string    `json:"status_message,omitempty"`
    StartTime     time.Time `json:"start_time"`
    EndTime       time.Time `json:"end_time,omitempty"`
    DurationMs    int64     `json:"duration_ms"`
    TokensIn      int64     `json:"tokens_in,omitempty"`
    TokensOut     int64     `json:"tokens_out,omitempty"`
    CostUSD       float64   `json:"cost_usd,omitempty"`
    Model         string    `json:"model,omitempty"`
    Provider      string    `json:"provider,omitempty"`
}
```

### New Endpoints

| Endpoint | Level | Purpose |
|----------|-------|---------|
| `GET /api/chains/{id}/stages/{stageId}/spans?limit=200&offset=0` | L2 | Paginated lite spans for one stage |
| `GET /api/spans/{id}` | L3 | Full span with attributes (single span) |
| `GET /api/chains/{id}/stages/{stageId}/chat?limit=50&offset=0` | L4 | Chat messages for a stage |

Existing endpoints stay unchanged. `GET /api/chains/{id}` defaults to `include_spans=false`.

---

## Implementation Plan

### Phase 1: Backend Quick Wins (2 days)

#### 1.1 Fix double-fetch bug

**File:** `internal/server/handlers_chains.go:122-130`

The `handleGetChain` handler calls `GetChain(ctx, chainID, opts)` with `IncludeStages: true`, which internally loads stages. Then it calls `GetChainStages()` again when `include_spans=true`. Remove the redundant call.

**Before:**
```go
chain, err := s.obsBackend.GetChain(ctx, chainID, opts) // loads stages
if chain != nil && opts.IncludeSpans {
    stages, err := s.obsBackend.GetChainStages(ctx, chainID, opts) // loads stages AGAIN with spans
    chain.Stages = stages
}
```

**After:**
```go
chain, err := s.obsBackend.GetChain(ctx, chainID, opts)
// stages already loaded by GetChain when opts.IncludeStages is true
// spans loaded per-stage within GetChainStages (called by GetChain)
```

#### 1.2 Replace Stats N+1 with single SQL aggregation

**Files:**
- `internal/observatory/store_chains.go` — add `GetChainStatsByAgent()`
- `internal/observatory/backend.go` — add to `Backend` interface
- `internal/server/handlers_chains.go:454-530` — rewrite `handleChainsStats`
- `cmd/ailang/chains_stats.go` — rewrite CLI stats command

**New SQL (replaces 1+N queries with 1):**
```sql
SELECT cs.agent_id,
       COUNT(cs.id) as stages,
       SUM(CASE WHEN cs.status = 'completed' THEN 1 ELSE 0 END) as completed,
       SUM(CASE WHEN cs.status = 'failed' THEN 1 ELSE 0 END) as failed,
       COALESCE(SUM(cs.cost), 0) as total_cost,
       COALESCE(SUM(cs.tokens_in), 0) as total_tokens_in,
       COALESCE(SUM(cs.tokens_out), 0) as total_tokens_out
FROM chain_stages cs
JOIN execution_chains c ON cs.chain_id = c.id
WHERE ($1 IS NULL OR c.created_at > $1)
GROUP BY cs.agent_id
ORDER BY total_cost DESC
```

Also add a companion query for chain-level stats:
```sql
SELECT status, COUNT(*) as cnt,
       COALESCE(SUM(total_cost), 0) as cost,
       COALESCE(SUM(total_tokens), 0) as tokens
FROM execution_chains
WHERE ($1 IS NULL OR created_at > $1)
GROUP BY status
```

#### 1.3 Add SpanLite type and per-stage query

**Files:**
- `internal/observatory/models_chains.go` — add `SpanLite` struct
- `internal/observatory/store_chains.go` — add `GetSpanLitesByStageID(ctx, stageID, limit, offset) ([]*SpanLite, int, error)`
- `internal/observatory/backend.go` — add to interface

**SQL (no attributes columns):**
```sql
SELECT id, trace_id, parent_span_id, name, kind, status, status_message,
       start_time, end_time, duration_ms, tokens_in, tokens_out, cost_usd,
       model, provider
FROM spans
WHERE stage_id = ?
ORDER BY start_time ASC
LIMIT ? OFFSET ?
```

This avoids reading SQLite overflow pages for the `attributes` and `resource_attributes` columns.

---

### Phase 2: New API Endpoints (2 days)

#### 2.1 Stage spans endpoint

`GET /api/chains/{chainId}/stages/{stageId}/spans?limit=200&offset=0`

**Response:**
```json
{
  "spans": [...SpanLite],
  "total": 1523,
  "limit": 200,
  "offset": 0
}
```

**File:** `internal/server/handlers_chains.go` — add `handleStageSpans`, register route

#### 2.2 Span detail endpoint

`GET /api/spans/{spanId}`

Returns full `Span` with `attributes` and `resource_attributes`. Single row lookup by PK.

Reuse existing `GetSpan()` from `internal/observatory/store_spans.go`.

**File:** `internal/server/handlers_chains.go` — add `handleGetSpanDetail`, register route

#### 2.3 Stage chat endpoint

`GET /api/chains/{chainId}/stages/{stageId}/chat?limit=50&offset=0`

Looks up stage's `task_id` or `session_id`, then fetches chat messages.

Reuse existing `GetChatMessagesByTaskID` / `GetChatMessagesBySession` from backend interface.

**File:** `internal/server/handlers_chains.go` — add `handleStageChat`, register route

#### 2.4 Default `include_spans=false`

Change `GET /api/chains/{id}` to NOT include spans unless explicitly requested via `?include_spans=true`.

This is the biggest behavioral change — the dashboard must use the new L2 endpoint for spans.

---

### Phase 3: Database Optimizations (2 days)

#### 3.1 `trace_summaries` materialized table

**Problem:** `ListTraces` uses 3 correlated subqueries per trace across 213K traces.

**Solution:** Pre-compute trace metadata on span insert.

```sql
CREATE TABLE IF NOT EXISTS trace_summaries (
    trace_id TEXT PRIMARY KEY,
    root_span_name TEXT,
    root_span_status TEXT,
    span_count INTEGER DEFAULT 0,
    total_duration_ms INTEGER DEFAULT 0,
    start_time TIMESTAMP,
    end_time TIMESTAMP,
    task_id TEXT,
    service_name TEXT,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_trace_summaries_time ON trace_summaries(start_time DESC);
CREATE INDEX IF NOT EXISTS idx_trace_summaries_task ON trace_summaries(task_id);
```

- On every `CreateSpan`: upsert into `trace_summaries`
- Migration back-fills from existing 213K traces (one-time, uses `INSERT ... ON CONFLICT` from aggregation query)
- Rewrite `ListTraces` to read from `trace_summaries` — simple scan, no subqueries

**Files:**
- `internal/observatory/schema_chains.sql` — add table DDL
- `internal/observatory/migrate.go` — add migration + back-fill
- `internal/observatory/store_spans.go` — update `CreateSpan` to call `upsertTraceSummary`, rewrite `ListTraces`

#### 3.2 `chain_stats_cache` table (optional, if Phase 1.2 isn't fast enough)

Single-row pre-computed aggregates updated on chain/stage status changes. Stats reads become O(1).

This is optional because the SQL aggregation in Phase 1.2 should be fast enough with proper indexes. Only add if benchmarking shows the aggregation query taking > 500ms.

---

### Phase 4: Frontend Progressive Loading (3 days)

#### 4.1 Visibility-aware polling

**New hook:** `ui/src/hooks/useDocumentVisibility.ts`

```typescript
function useDocumentVisibility(): boolean {
    const [isVisible, setIsVisible] = useState(!document.hidden);
    useEffect(() => {
        const handler = () => setIsVisible(!document.hidden);
        document.addEventListener('visibilitychange', handler);
        return () => document.removeEventListener('visibilitychange', handler);
    }, []);
    return isVisible;
}
```

Wire into all polling hooks in `ControlPlane.tsx`:
```typescript
const isVisible = useDocumentVisibility();
const { stats } = useControlPlaneStats({ refreshInterval: isVisible ? 10000 : 0 });
const { data: topologyData } = useTopologyData({ refreshInterval: isVisible ? 5000 : 0 });
// etc.
```

#### 4.2 Refactor `useChainData` for progressive loading

**File:** `ui/src/features/controlplane/hooks/useChainData.ts`

Three-step load:
1. Fetch chain + stages (L1): `GET /api/chains/{id}` — render stage cards immediately
2. On stage click: `GET /api/chains/{id}/stages/{stageId}/spans?limit=200` — render span tree
3. On "Chat" tab: `GET /api/chains/{id}/stages/{stageId}/chat?limit=50` — render chat

#### 4.3 New `useStageSpans` hook

```typescript
function useStageSpans(chainId: string, stageId: string | null) {
    const [spans, setSpans] = useState<SpanLite[]>([]);
    const [total, setTotal] = useState(0);
    const cache = useRef<Record<string, SpanLite[]>>({});

    useEffect(() => {
        if (!stageId) return;
        if (cache.current[stageId]) {
            setSpans(cache.current[stageId]);
            return;
        }
        fetch(`/api/chains/${chainId}/stages/${stageId}/spans?limit=200`)
            .then(r => r.json())
            .then(data => {
                cache.current[stageId] = data.spans;
                setSpans(data.spans);
                setTotal(data.total);
            });
    }, [chainId, stageId]);

    return { spans, total };
}
```

#### 4.4 Skeleton loading

- Chain header + stage pipeline cards: render from L1 data (instant)
- Span tree panel: skeleton shimmer until L2 completes
- Chat tab: "Load conversation" button (not auto-fetched)
- Span detail panel: "Click a span to view details" placeholder until L3

#### 4.5 Virtual span tree (stretch goal)

For stages with > 500 spans, use `@tanstack/virtual` for windowed rendering.

---

### Phase 5: CLI Parity (1 day)

#### 5.1 `ailang chains view` tiered loading

- Default: chain + stages (L1)
- `--spans`: fetch spans per stage (L2), no attributes
- `--full`: include attributes (L3) — explicit opt-in

**File:** `cmd/ailang/chains.go`

#### 5.2 `ailang chains stats` batch SQL

Already done in Phase 1.2.

---

## Expected Performance

| Operation | Before | After | Improvement |
|-----------|--------|-------|-------------|
| Chain stats by agent | ~5s (50 queries) | ~50ms (1 query) | **100x** |
| Chain detail (10 stages) | ~3s (20 queries, MB of attributes) | ~100ms (1 query, 5KB) | **30x** |
| Stage spans (click) | N/A (loaded upfront) | ~200ms (1 query, 50KB) | **first interaction faster** |
| ListTraces | ~10s (639K subqueries) | ~200ms (simple scan) | **50x** |
| Dashboard mount | ~15s | ~2s | **7x** |
| Hidden tab CPU | 100% (continuous polling) | ~0% (paused) | **infinite** |

---

## Cloud Readiness

The tiered architecture maps directly to cloud backends:

| Level | SQLite (current) | Firestore (future) | CloudSQL (future) |
|-------|-------------------|--------------------|--------------------|
| L0: Chain list | Single table + LEFT JOIN | Collection query with projection | Same SQL |
| L1: Chain + stages | Single query | 1 doc + subcollection | Same SQL |
| L2: Stage spans (lite) | Index scan, no overflow pages | Subcollection with field mask | Same SQL without attributes |
| L3: Span detail | Single row by PK | Single document read | Single row by PK |
| L4: Chat context | Index scan on chat_messages | Subcollection query | Same SQL |

Each level is a separate network call — progressive loading naturally hides latency.

**`trace_summaries` table** maps to a Firestore document per trace, updated transactionally on writes.

---

## Migration Safety

All changes are backward-compatible:
- New `SpanLite` type is additive — existing `Span` type unchanged
- New endpoints are additions — existing endpoints keep working
- `trace_summaries` back-filled via migration, then maintained incrementally
- Frontend progressively enhanced — falls back to existing behavior if new endpoints unavailable
- No schema columns removed, no breaking API changes

---

## Files Modified

### Backend (Go)

| File | Changes |
|------|---------|
| `internal/observatory/models_chains.go` | Add `SpanLite` struct |
| `internal/observatory/store_chains.go` | Add `GetChainStatsByAgent`, `GetSpanLitesByStageID`; fix per-stage loop |
| `internal/observatory/store_spans.go` | Add `upsertTraceSummary`; rewrite `ListTraces` |
| `internal/observatory/backend.go` | Add new methods to interface |
| `internal/observatory/schema_chains.sql` | Add `trace_summaries` table |
| `internal/observatory/migrate.go` | Add migration v12 with back-fill |
| `internal/server/handlers_chains.go` | Fix double-fetch; rewrite stats handler; add 3 new endpoints |
| `internal/server/routes.go` | Register new routes |
| `cmd/ailang/chains.go` | Update `view` for tiered flags |
| `cmd/ailang/chains_stats.go` | Rewrite to use batch SQL |

### Frontend (TypeScript/React)

| File | Changes |
|------|---------|
| `ui/src/hooks/useDocumentVisibility.ts` | New: visibility-aware polling hook |
| `ui/src/features/controlplane/hooks/useChainData.ts` | Refactor for progressive 3-step loading |
| `ui/src/features/controlplane/hooks/useStageSpans.ts` | New: lazy per-stage span loading |
| `ui/src/features/controlplane/ControlPlane.tsx` | Wire visibility into polling hooks |
| `ui/src/features/controlplane/components/ChainExplorer/StageDetail.tsx` | Use `useStageSpans` instead of pre-loaded spans |

---

## Verification

1. **Reproduce**: `time ailang chains stats --by-agent` — expect ~5s today
2. **Phase 1 check**: Same command < 200ms
3. **Dashboard**: Dev tools Network tab → chain detail should NOT have `include_spans=true`
4. **Tab visibility**: Switch tab → verify zero network requests in dev tools
5. **Large chain**: Click stage → L2 loads < 200ms
6. **Tests**: `make test` — all pass
7. **CLI**: `ailang chains view <id> --spans` shows stage-by-stage output
