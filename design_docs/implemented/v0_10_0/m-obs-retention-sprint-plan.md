# Sprint Plan: M-OBS-RETENTION

## Summary
Add automatic retention policies to the observatory database to prevent unbounded growth. 64GB DB caused severe laptop pressure. Fix: TTL-based cleanup, WAL checkpointing, startup health check.

**Duration:** 2 milestones, ~3 hours
**Risk Level:** Low — straightforward DELETE + VACUUM operations on well-understood tables
**Design Doc:** [m-obs-retention.md](m-obs-retention.md)

## Proposed Milestones

### Milestone 1: M1_RETENTION_AND_HEALTH
**Goal:** Core retention function + startup health check + WAL checkpoint on open.

**Estimated:** 80 LOC implementation + 60 LOC tests = 140 LOC

**Tasks:**
- Create `internal/observatory/retention.go` with `RunRetention()` and `RetentionStats`
- Create `internal/observatory/health.go` with `CheckHealth()` (os.Stat-based, no DB open for fast path)
- Add WAL checkpoint (`PRAGMA wal_checkpoint(TRUNCATE)`) to `store.go` Open
- Add `CleanupOldEvents()` to `internal/coordinator/store_sqlite_events.go`
- Call `CheckHealth()` from `cmd/ailang/main.go` startup
- Write tests: insert old + new rows, verify TTL, verify health thresholds

**Acceptance Criteria:**
- [ ] `RunRetention()` deletes spans older than 7 days
- [ ] `RunRetention()` deletes chat_messages older than 30 days
- [ ] `RunRetention()` checkpoints WAL
- [ ] `CheckHealth()` returns correct action at 200MB/500MB/2GB thresholds
- [ ] WAL checkpoint runs on store Open
- [ ] `CleanupOldEvents()` deletes task_events older than 30 days
- [ ] `make test` passes, `make lint` clean

### Milestone 2: M2_DAEMON_INTEGRATION
**Goal:** Wire retention into coordinator daemon + add CLI command.

**Estimated:** 30 LOC implementation + 20 LOC tests = 50 LOC

**Tasks:**
- Add hourly retention ticker to coordinator daemon `Run()` loop (line 325 area)
- Add `ailang observatory cleanup` CLI command for manual trigger
- Log retention stats on each run

**Acceptance Criteria:**
- [ ] Coordinator daemon runs retention hourly
- [ ] `ailang observatory cleanup` runs retention and reports stats
- [ ] `make test` passes, `make lint` clean
