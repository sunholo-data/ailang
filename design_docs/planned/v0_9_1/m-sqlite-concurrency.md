# M-SQLITE-CONCURRENCY: Fix SQLite Database Lock Contention

**Status**: Planned
**Target**: v0.9.1
**Priority**: P1 (High) — Data loss during agent evals, will worsen in cloud
**Estimated**: 1 day (~4 hours implementation + 2 hours testing + 1 hour docs)
**Dependencies**: None
**Source**: Agent eval run 2026-03-10 lost 171/184 chain stage writes to "database is locked"

## Problem Statement

When running agent evals with `--agent-parallel 5`, concurrent goroutines write to `observatory.db` simultaneously. Despite WAL mode and `busy_timeout=5000`, writes fail with "database is locked" errors, causing:

1. **Eval chain tracking loss**: 13/184 stages recorded (93% data loss)
2. **Chat import failures**: `deleting existing messages: database is locked`
3. **Chain status update failures**: `failed to update eval chain status: database is locked`

This is a **local-only** issue today (SQLite), but will become critical when running evals on Cloud Run where multiple Cloud Run Jobs may contend on the same Firestore collections or shared state.

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

**Firestore** (used in cloud mode for messages/spans): Natively handles concurrent writes — no SQLite lock issue. However:
- `coordinator.db` still uses SQLite locally for task state
- Cloud Run Jobs writing completion results to Pub/Sub → coordinator could contend
- Eval chains stored in `observatory.db` — if we move evals to cloud, need Firestore backend for observatory

**Pub/Sub**: Message-based, no contention. But completion handlers write to SQLite stores.

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

### Fix 2: Write Serialization for Observatory (eval hot path)

Add a write channel to serialize span creation:

```go
type SQLiteBackend struct {
    store   *Store
    writeCh chan writeRequest  // Buffered channel for write serialization
    done    chan struct{}
}

type writeRequest struct {
    span   *Span
    result chan error
}

func (b *SQLiteBackend) Start() {
    go b.writeLoop()
}

func (b *SQLiteBackend) writeLoop() {
    for req := range b.writeCh {
        err := b.doCreateSpan(req.span)
        req.result <- err
    }
}

func (b *SQLiteBackend) CreateSpan(ctx context.Context, span *Span) error {
    result := make(chan error, 1)
    b.writeCh <- writeRequest{span: span, result: result}
    return <-result
}
```

**Trade-off**: Writes become sequential, but SQLite writes are already sequential at the lock level. This just moves the serialization from SQLite's lock to a Go channel, which is more predictable and doesn't timeout.

**File**: `internal/observatory/backend_sqlite.go`

### Fix 3: Retry with Backoff (defense in depth)

For paths that can't use write serialization (e.g., cross-package writes):

```go
func withRetry(fn func() error, maxRetries int) error {
    for i := 0; i <= maxRetries; i++ {
        err := fn()
        if err == nil {
            return nil
        }
        if !strings.Contains(err.Error(), "database is locked") {
            return err
        }
        if i < maxRetries {
            time.Sleep(time.Duration(math.Pow(2, float64(i))) * 100 * time.Millisecond)
        }
    }
    return fmt.Errorf("database is locked after %d retries", maxRetries)
}
```

**File**: New `internal/storage/retry.go` (shared by all 3 databases)

### Fix 4: Batch Span Writes in Eval Harness

Instead of writing each span individually, batch completed benchmarks:

```go
// In eval runner, collect results then write in batch
results := make([]*Span, 0, batchSize)
for span := range completedCh {
    results = append(results, span)
    if len(results) >= batchSize {
        backend.CreateSpanBatch(ctx, results)
        results = results[:0]
    }
}
```

**File**: `internal/eval_harness/agent_runner.go`, `internal/observatory/backend_sqlite.go`

### Cloud Mode Considerations

| Component | Local (SQLite) | Cloud (Firestore/Pub/Sub) | Fix Needed? |
|-----------|---------------|--------------------------|-------------|
| Observatory spans | Fix 1+2+4 | Firestore handles concurrency | No |
| Coordinator tasks | Fix 1+3 | Pub/Sub + Firestore | No |
| Collaboration msgs | Fix 1+3 | Firestore | No |
| Eval chains | Fix 1+2+4 | Need Firestore backend | Yes (v0.10+) |
| Chat import | Fix 3 | N/A (no local import) | No |

**Key insight**: Cloud mode eliminates SQLite concurrency for messages and coordinator tasks, but eval chains are currently SQLite-only. When evals move to cloud, the observatory needs a Firestore backend (or eval results write to Pub/Sub and a single consumer writes to Firestore).

## Implementation Plan

### Milestone 1: Connection Pool + Retry (~2 hours)
1. Add `SetMaxOpenConns(1)` to all 3 database init paths
2. Create `internal/storage/retry.go` with `withRetry()` helper
3. Wrap eval chain writes with retry
4. Test: run agent eval with `--agent-parallel 10`, verify 0 lock errors

### Milestone 2: Write Serialization for Observatory (~2 hours)
1. Add `writeCh` channel to `SQLiteBackend`
2. Move `CreateSpan` to use write channel
3. Add `CreateSpanBatch` for eval harness
4. Test: concurrent span writes don't lose data

### Milestone 3: Eval Harness Batching (~1.5 hours)
1. Collect completed spans in eval runner
2. Flush batch on completion or timer
3. Test: full eval suite with both models, verify chain completeness

### Milestone 4: Docs + Cloud Assessment (~0.5 hours)
1. Document SQLite concurrency patterns
2. Assess Firestore observatory backend for v0.10
3. Update CLAUDE.md with database concurrency guidelines

## Success Criteria

- [ ] Agent eval `--agent-parallel 10` records 100% of stages in chain
- [ ] Zero "database is locked" errors in eval output
- [ ] No performance regression (write serialization ≤ 5% slower)
- [ ] Cloud mode unaffected (Firestore paths unchanged)

## Testing Strategy

1. **Stress test**: `ailang eval-suite --agent --agent-parallel 10 --benchmarks fizzbuzz,adt_option,fold_reduce --models claude-haiku-4-5,gemini-2-5-flash` — all 12 stages recorded
2. **Concurrent write test**: Spawn 20 goroutines writing spans simultaneously — 0 errors
3. **Regression**: `make test` passes, existing eval chains still queryable
