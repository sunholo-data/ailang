# M-SQLITE-CONCURRENCY: Fix SQLite Database Lock Contention

**Status**: In Progress
**Target**: v0.9.1
**Priority**: P1 (High) — Data loss during agent evals
**Estimated**: 30 minutes (Fix 1 only — connection pool limits)
**Dependencies**: None
**Source**: Agent eval run 2026-03-10 lost 171/184 chain stage writes to "database is locked"

## Problem Statement

When running agent evals with `--agent-parallel 5`, concurrent goroutines write to `observatory.db` simultaneously. Despite WAL mode and `busy_timeout=5000`, writes fail with "database is locked" errors, causing:

1. **Eval chain tracking loss**: 13/184 stages recorded (93% data loss)
2. **Chat import failures**: `deleting existing messages: database is locked`
3. **Chain status update failures**: `failed to update eval chain status: database is locked`

This is a **local-only** issue (SQLite). Cloud mode (`AILANG_STORAGE=gcp`) uses Firestore for all 3 databases — including observatory chains — and handles concurrent writes natively. No cloud fix is needed.

## Root Cause Analysis

### Current State (all 3 databases)

| Database | WAL | busy_timeout | Connection Pool | Write Mutex | Retry |
|----------|-----|-------------|-----------------|-------------|-------|
| observatory.db | Yes | 5000ms | No limits | No | No |
| coordinator.db | Yes | 5000ms | No limits | No | No |
| collaboration.db | Yes | 5000ms | No limits | No | No |

### Why WAL + busy_timeout isn't enough

WAL mode allows concurrent reads, but still requires **exclusive access for writes**. With `--agent-parallel 5`:
- 5 goroutines finish benchmarks near-simultaneously
- Each calls `CreateSpan()` → `INSERT INTO spans` + `UPDATE task_aggregates` + `UPDATE agent_assignment_aggregates` (3 writes per span, in a transaction)
- `busy_timeout=5000` means each writer retries for up to 5 seconds
- Under sustained write load (184 results in 19 minutes), the 5s window is exceeded because writers queue behind each other

### Cloud implications

**Not affected.** Firestore backends already exist for ALL 3 databases:
- `internal/storage/firestore/observatory_spans.go` — spans
- `internal/storage/firestore/observatory_chains.go` — eval chains (previously thought missing)
- `internal/storage/firestore/coordinator.go` — coordinator tasks
- `internal/storage/firestore/messaging.go` — collaboration messages

Setting `AILANG_STORAGE=gcp` eliminates all SQLite concurrency issues. This design doc addresses **local mode only**.

## Design

### Fix 1: Connection Pool Configuration (all databases)

Add pool limits to prevent unbounded connection creation:

```go
db.SetMaxOpenConns(1)        // SQLite only supports 1 writer anyway
db.SetMaxIdleConns(1)        // Keep 1 idle connection
db.SetConnMaxLifetime(0)     // Don't expire connections
```

**Why `MaxOpenConns(1)`**: SQLite is single-writer. Multiple connections just queue at the SQLite lock level. A single connection with Go's `database/sql` serializes all writes through one connection, avoiding lock contention entirely. Reads still work concurrently through WAL snapshots.

**Files**: `observatory/backend_sqlite.go`, `coordinator/store_sqlite.go`, `messaging/schema.go`

### Deferred Fixes (implement only if Fix 1 proves insufficient)

**Fix 2 (Write Serialization)**, **Fix 3 (Retry with Backoff)**, and **Fix 4 (Batch Writes)** are deferred. `SetMaxOpenConns(1)` serializes writes at the Go `database/sql` pool level, which should eliminate lock contention since SQLite is single-writer anyway. Adding a write channel on top would create redundant serialization layers.

If `--agent-parallel 10` still shows lock errors after Fix 1, escalate to Fix 3 (retry) next.

### Cloud Mode Considerations

**No cloud fix needed.** Firestore backends exist for all components:

| Component | Local (SQLite) | Cloud (Firestore) | Cloud Fix Needed? |
|-----------|---------------|-------------------|-------------------|
| Observatory spans | Fix 1 | `firestore/observatory_spans.go` | No |
| Observatory chains | Fix 1 | `firestore/observatory_chains.go` | No |
| Coordinator tasks | Fix 1 | `firestore/coordinator.go` | No |
| Collaboration msgs | Fix 1 | `firestore/messaging.go` | No |

## Implementation Plan

### Milestone 1: Connection Pool Limits (~30 minutes)
1. Add `SetMaxOpenConns(1)`, `SetMaxIdleConns(1)`, `SetConnMaxLifetime(0)` to:
   - `internal/observatory/backend_sqlite.go` (NewSQLiteBackendFromPath)
   - `internal/coordinator/store_sqlite.go` (NewSQLiteStore)
   - `internal/messaging/schema.go` (InitDB, after configureDB)
2. Test: `make test` passes
3. Manual verification: `--agent-parallel 10` eval run records all stages

## Success Criteria

- [ ] `make test` passes
- [ ] Agent eval `--agent-parallel 10` records 100% of stages in chain
- [ ] Zero "database is locked" errors in eval output
- [ ] Cloud mode unaffected (Firestore paths unchanged — they don't use these SQLite init paths)

## Testing Strategy

1. **Regression**: `make test` passes, existing eval chains still queryable
2. **Stress test**: `ailang eval-suite --agent --agent-parallel 10 --benchmarks fizzbuzz,adt_option,fold_reduce --models claude-haiku-4-5,gemini-2-5-flash` — all 12 stages recorded

## Audit Notes (2026-03-10)

**Design doc scored 8.5/10 in spec audit.** Key findings:
- Firestore observatory backends already exist (`observatory_chains.go`, `observatory_spans.go`) — the "Need Firestore backend (v0.10+)" line was incorrect
- Fix 1 alone should solve the problem — `SetMaxOpenConns(1)` serializes at the Go pool level, making Fix 2 (write channel) redundant
- Fixes 2-4 deferred unless Fix 1 proves insufficient under `--agent-parallel 10`
