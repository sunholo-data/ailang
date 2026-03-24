# M-OBS-RETENTION: Observatory Database Retention & Growth Control

**Status**: Planned
**Target**: v0.10.0
**Priority**: P0 (Critical — caused 64GB disk usage, severe memory pressure, laptop unusable)
**Estimated**: 2 days
**Dependencies**: None

## Axiom Compliance

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No change to execution semantics |
| A2: Replayability | 0 | Recent traces preserved; old ones were never replayed |
| A3: Effect Legibility | +1 | Storage effects become bounded and visible |
| A4: Explicit Authority | 0 | No capability changes |
| A5: Bounded Verification | +1 | Bounded storage enables bounded analysis |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | Health checks + auto-cleanup = self-managing system |
| A8: Minimal Syntax | 0 | No language syntax changes |
| A9: Cost Visibility | +1 | Storage costs bounded, visible in startup log. Applies to cloud Firestore too |
| A10: Composability | 0 | No composition impact |
| A11: Structured Failure | +1 | Graceful degradation — DB recreated if corrupt/oversized |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +5** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): Cleanup is explicit and logged
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Fully automated retention

## Problem Statement

### The Incident

The observatory SQLite database grew to **64GB** on a developer laptop, causing severe memory pressure and unusable system:

| File | Size | Cause |
|------|------|-------|
| `observatory.db` | 24 GB | Unbounded span storage — 5.3M rows |
| `observatory.db-wal` | 40 GB | WAL never checkpointed (grows proportionally to DB) |
| Total | **64 GB** | On a 512GB laptop |

### Table-Level Breakdown

| Table | Rows | Size Estimate | Growth Rate | Retention? |
|-------|------|---------------|-------------|------------|
| **spans** | 5,369,843 | ~20 GB (incl 3.9GB attributes column) | ~200K/week | None |
| **chat_messages** | 470,298 | ~1 GB | Manual import | None |
| **trace_summaries** | 268,019 | ~500 MB | Derived from spans | None |
| **metrics** | 129,917 | ~200 MB | ~5K/day | None |
| **session_tools** | 117,378 | ~100 MB | ~2K/day | None |
| sessions | 1,743 | <1 MB | Slow | N/A |
| tasks | 224 | <1 MB | Slow | N/A |
| chain_stages | 1,599 | <1 MB | Slow | N/A |

**Also affected:** `coordinator.db` has 101K `task_events` (144MB, no cleanup).

### Root Causes

1. **No DELETE ever runs** — every `ailang` command emits OTEL spans, stored forever
2. **No WAL checkpoint** — WAL grows unbounded, causes mmap memory pressure
3. **No startup health check** — system never warns about growing DB
4. **Attribute column bloat** — `attributes` + `resource_attributes` JSON columns are 3.9GB of the 5.3M spans (avg 750 bytes/span, some up to 698KB)
5. **Same pattern in cloud** — Firestore `obs_spans` collection grows forever with no TTL enforcement

### Prior Work

| Design Doc | What It Fixed | What It Didn't Fix |
|------------|---------------|-------------------|
| M-PERF-OBSERVATORY (v0.8.2) | Materialized `trace_summaries`, eliminated N+1 queries | **No retention/cleanup** — DB grew 5.6GB → 24GB in 3 weeks |
| M-COST1 (v0.9.6) | In-memory cost counter, eliminated hot-path Firestore reads | **No Firestore document TTL** (Tier 4 unimplemented) |
| M-COST2 (v0.9.8) | Reduced dashboard polling 10x, increased cache TTLs | **No bounded queries** (Phase 2-3 unimplemented) |
| M-OBS-CONFIGURABLE-SPAN-FILTERING | Configurable deny patterns for span ingestion | **Not yet implemented** — planned for v0.9.3 |

## Goals

**Primary Goal:** Observatory DB stays under 500MB with no manual intervention, locally and in cloud.

**Success Metrics:**
- Local: DB < 500MB, WAL < 100MB during normal development
- Cloud: Firestore `obs_spans` collection stays under 500K documents
- Startup health check warns at 200MB, auto-cleans at 500MB, recreates at 2GB
- `coordinator.db` task_events capped at 30 days
- All retention configurable via `~/.ailang/config.yaml`

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Span TTL: 7 days | Controls ~90% of storage | human | design | low |
| WAL checkpoint strategy | Prevents WAL bloat | agent | compile | low |
| DB size threshold for auto-recreate | Nuclear recovery option | human | design | med |
| Apply same TTL to Firestore cloud | Prevents cloud cost growth | human | design | med |

### Design Freeze

- [x] 7-day TTL for spans, trace_summaries, metrics
- [x] 30-day TTL for chat_messages, session_tools, task_events
- [x] Startup health check: warn 200MB, cleanup 500MB, recreate 2GB
- [x] Apply TTL to Firestore documents (cloud parity)

## Solution Design

### Overview

Four layers of defense, applied to **both local SQLite and cloud Firestore**:

1. **Startup health check** — on every process start, check DB size
2. **Write-time TTL** — on span insert, delete spans older than TTL in same transaction (amortized cleanup)
3. **Periodic retention** — coordinator daemon runs full cleanup hourly
4. **WAL management** — checkpoint on startup + hourly

### Component 1: Retention Cleanup (`internal/observatory/retention.go`)

```go
// RunRetention deletes rows older than their TTL.
// Called periodically by coordinator daemon and on startup if DB is oversized.
func (s *Store) RunRetention(ctx context.Context) (RetentionStats, error) {
    now := time.Now()
    var stats RetentionStats

    // Spans: 7-day TTL (largest table)
    res, _ := s.db.ExecContext(ctx,
        "DELETE FROM spans WHERE start_time < ?", now.Add(-7*24*time.Hour).UnixNano())
    stats.SpansDeleted, _ = res.RowsAffected()

    // Trace summaries: 7-day TTL (derived from spans)
    res, _ = s.db.ExecContext(ctx,
        "DELETE FROM trace_summaries WHERE created_at < ?", now.Add(-7*24*time.Hour).Unix())
    stats.SummariesDeleted, _ = res.RowsAffected()

    // Metrics: 7-day TTL
    res, _ = s.db.ExecContext(ctx,
        "DELETE FROM metrics WHERE timestamp < ?", now.Add(-7*24*time.Hour).Unix())
    stats.MetricsDeleted, _ = res.RowsAffected()

    // Chat messages: 30-day TTL
    res, _ = s.db.ExecContext(ctx,
        "DELETE FROM chat_messages WHERE created_at < ?", now.Add(-30*24*time.Hour).Unix())
    stats.ChatDeleted, _ = res.RowsAffected()

    // Session tools: 30-day TTL
    res, _ = s.db.ExecContext(ctx,
        "DELETE FROM session_tools WHERE created_at < ?", now.Add(-30*24*time.Hour).Unix())
    stats.ToolsDeleted, _ = res.RowsAffected()

    // WAL checkpoint
    s.db.ExecContext(ctx, "PRAGMA wal_checkpoint(TRUNCATE)")

    return stats, nil
}
```

### Component 2: Coordinator Task Events Retention

In `internal/coordinator/store_sqlite_events.go`:

```go
// CleanupOldEvents deletes task events older than the given duration.
func (s *SQLiteStore) CleanupOldEvents(ctx context.Context, maxAge time.Duration) (int64, error) {
    cutoff := time.Now().Add(-maxAge)
    res, err := s.db.ExecContext(ctx, "DELETE FROM task_events WHERE created_at < ?", cutoff)
    if err != nil { return 0, err }
    return res.RowsAffected()
}
```

### Component 3: Startup Health Check (`internal/observatory/health.go`)

```go
func CheckHealth(dbPath string) HealthResult {
    // Fast: just os.Stat, no DB open needed
    totalMB := dbSizeMB(dbPath) + walSizeMB(dbPath)

    switch {
    case totalMB > 2048:
        return HealthResult{Action: "recreate", SizeMB: totalMB}
    case totalMB > 500:
        return HealthResult{Action: "cleanup", SizeMB: totalMB}
    case totalMB > 200:
        return HealthResult{Action: "warn", SizeMB: totalMB}
    default:
        return HealthResult{Action: "ok", SizeMB: totalMB}
    }
}
```

Called from `cmd/ailang/main.go` early in startup. On "recreate", deletes DB files. On "cleanup", opens DB and runs `RunRetention()`.

### Component 4: Cloud Firestore TTL

Firestore supports [TTL policies](https://cloud.google.com/firestore/docs/ttl) natively. Add TTL field to span documents:

```go
// In internal/storage/firestore/observatory_spans.go, on insert:
"expire_at": time.Now().Add(7 * 24 * time.Hour), // Firestore auto-deletes after TTL
```

Configure via Terraform in `ailang-multivac`:

```hcl
resource "google_firestore_field" "spans_ttl" {
  project    = var.project_id
  database   = google_firestore_database.observatory.name
  collection = "obs_spans"
  field      = "expire_at"

  ttl_config {}
}
```

This gives cloud parity — same 7-day retention, managed by Firestore infrastructure (no application cleanup needed).

### Component 5: WAL Checkpoint on Open

In `internal/observatory/store.go`, after opening:

```go
// Checkpoint WAL to prevent unbounded growth.
// TRUNCATE mode: moves WAL content to DB, then truncates WAL to zero.
db.Exec("PRAGMA wal_checkpoint(TRUNCATE)")
```

### Component 6: Periodic Retention in Coordinator Daemon

In `daemon.go` Run loop:

```go
retentionTicker := time.NewTicker(1 * time.Hour)
// On tick: obsStore.RunRetention(ctx) + taskStore.CleanupOldEvents(ctx, 30*24*time.Hour)
```

### Implementation Plan

**Phase 1: Local SQLite retention** (~3 hours)
- [ ] Create `internal/observatory/retention.go` with `RunRetention()`, `RetentionStats`
- [ ] Create `internal/observatory/health.go` with `CheckHealth()`
- [ ] Add WAL checkpoint to `store.go` Open function
- [ ] Add `CleanupOldEvents()` to coordinator `store_sqlite_events.go`
- [ ] Call `CheckHealth()` from `cmd/ailang/main.go` startup
- [ ] Tests for retention (insert old + new rows, verify TTL respected)

**Phase 2: Coordinator integration** (~1 hour)
- [ ] Add hourly retention ticker to coordinator daemon
- [ ] Add `ailang observatory cleanup` CLI command for manual trigger
- [ ] Log retention stats (rows deleted, time taken)

**Phase 3: Cloud Firestore TTL** (~1 hour)
- [ ] Add `expire_at` field to Firestore span document writes
- [ ] Add Terraform `google_firestore_field` TTL resource
- [ ] Add `expire_at` to other high-growth collections (metrics, session_tools)

**Phase 4: Unimplemented cost optimizations** (separate sprint, reference here)
- [ ] M-COST1 Tier 4: Firestore document TTL/cleanup
- [ ] M-COST2 Phase 2: Bounded queries, COUNT aggregation
- [ ] M-OBS-CONFIGURABLE-SPAN-FILTERING: Reduce ingestion volume
- [ ] SpanLite queries: Exclude 3.9GB attributes column for metadata views

### Files to Modify/Create

**New files:**
- `internal/observatory/retention.go` — RunRetention + RetentionStats (~70 LOC)
- `internal/observatory/retention_test.go` — TTL tests (~80 LOC)
- `internal/observatory/health.go` — CheckHealth startup check (~40 LOC)

**Modified files:**
- `internal/observatory/store.go` — WAL checkpoint on Open (~3 LOC)
- `internal/coordinator/store_sqlite_events.go` — CleanupOldEvents (~15 LOC)
- `internal/coordinator/daemon.go` — Hourly retention ticker (~10 LOC)
- `cmd/ailang/main.go` — Call CheckHealth on startup (~10 LOC)

**Cloud (ailang-multivac repo):**
- `internal/storage/firestore/observatory_spans.go` — Add `expire_at` field on write (~5 LOC)
- `terraform/firestore.tf` — Add TTL field resource (~10 LOC HCL)

## Success Criteria

- [ ] Observatory DB stays under 500MB during 1 week of normal development
- [ ] WAL file stays under 100MB (checkpointed on startup + hourly)
- [ ] Startup logs DB size: "Observatory: 45MB (ok)" or "Observatory: 650MB — running cleanup"
- [ ] `RunRetention()` deletes spans older than 7 days, chat older than 30 days
- [ ] `CleanupOldEvents()` deletes task_events older than 30 days
- [ ] Firestore spans have `expire_at` field, TTL policy configured
- [ ] No data loss for recent work (last 7 days always preserved)
- [ ] All tests pass, lint clean

## Testing Strategy

**Unit tests:**
- Insert spans with old timestamps → RunRetention deletes them
- Insert spans with recent timestamps → RunRetention preserves them
- CheckHealth returns correct action at each threshold
- CleanupOldEvents respects cutoff

**Integration tests:**
- Start coordinator → verify retention ticker fires after 1 hour
- Run `ailang observatory cleanup` → verify output shows deletion stats

**Manual testing:**
- Verify OTLP health check skips unreachable endpoint (already implemented)
- Run coordinator for a day → check DB stays under 500MB

## Non-Goals

- **Backup before delete** — traces are ephemeral diagnostics, not source of truth
- **Compression** — SQLite doesn't support it; size reduction via deletion
- **Migration to TimescaleDB/ClickHouse** — SQLite is correct for local; cloud uses Firestore
- **Per-span TTL configuration** — uniform 7-day policy is sufficient for v0.10.0
- **SpanLite query optimization** — important but separate concern (reduces query cost, not storage)

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Retention deletes spans needed for debugging | Low | 7-day TTL preserves all recent work |
| DELETE on 5M rows is slow | Med | Batch deletes (LIMIT 10000 per batch), incremental_vacuum |
| DB recreation loses task/chain history | Low | Tasks in coordinator.db; chains preserved; only spans lost |
| Firestore TTL has delay (up to 72 hours) | Low | Acceptable — billing is based on storage, not document count |

## Related Documents

**Implemented:**
- [m-perf-observatory-tiered-loading.md](../../implemented/v0_9_0/m-perf-observatory-tiered-loading.md) — Query optimization (5.6GB at that time)
- [observatory-architecture.md](../../implemented/v0_7_0/observatory-architecture.md) — Original design
- M-COST1 (v0.9.6) — In-memory cost counter
- M-COST2 (v0.9.8) — Dashboard polling optimization

**Planned:**
- [m-obs-configurable-span-filtering.md](../v0_9_3/m-obs-configurable-span-filtering.md) — Reduce ingestion volume (complementary)
- [m-cloud-observatory.md](m-cloud-observatory.md) — Cloud observatory (needs TTL for cost control)

---

**Document created**: 2026-03-24
**Last updated**: 2026-03-24
