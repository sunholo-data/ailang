# Sprint Plan: M-COST1 Firestore Cost Optimization

## Summary

Eliminate Firestore full-collection scans on coordinator hot paths by introducing in-memory counters, materialized aggregate documents, batch reads, and server-side aggregation queries. Target: >90% reduction in Firestore document reads.

**Duration:** 4 days (milestones can be done incrementally)
**Dependencies:** None (all changes are in `internal/storage/firestore/` and `internal/coordinator/`)
**Risk Level:** Medium (Firestore-specific code requires emulator for testing; no existing Firestore tests)

## Current Status Analysis

### Completed Recently
- M-PKG Phase 1.5b: Git-based dependencies + AGENT.md discovery
- M-PERF5: Data-intensive workloads optimization (compilation)
- M-PERF-OBSERVATORY: Tiered loading for SQLite observatory (same pattern — eliminates full scans)

### Velocity
- Recent average: ~200 LOC/day (from 225 commits in 14 days, mix of features + infrastructure)
- Estimated capacity: ~800 LOC over 4 days

### Remaining from Design Doc
- Tier 1: In-memory cost counter + materialized task stats + stale detector cache (~170 LOC)
- Tier 2: Batch GetAll + native aggregation + client wrapper methods (~130 LOC)
- Tier 3: Dashboard materialization + HTTP caching (~100 LOC)
- Tier 4: Document TTL + dev mode config (~100 LOC)

**Total estimated: ~500 LOC implementation + ~150 LOC tests = ~650 LOC**

## Proposed Milestones

### M1: CLIENT_AND_COUNTERS — Client Methods + In-Memory Cost Counter
**Goal:** Add missing Client wrapper methods (GetAll, Batch) and replace `GetCostByProvider()` full scan with in-memory counter synced to Firestore.
**Estimated:** 180 LOC implementation + 40 LOC tests = 220 LOC
**Duration:** 1 day

**Tasks:**
1. Add `GetAll()` and `Batch()` methods to `internal/storage/firestore/client.go`
2. Add `sync.RWMutex` + `costCounters map[string]float64` fields to `CoordinatorStore`
3. Implement `loadCostCounters()` — reads `_meta/cost_counters` doc on startup, falls back to one-time full scan if doc doesn't exist
4. Implement `syncCostCounters()` — background goroutine writing to `_meta/cost_counters` every 5 min
5. Modify `GetCostByProvider()` to read from memory (0 Firestore reads)
6. Hook into `MarkTaskCompleted()` to call `addCost(provider, cost)`
7. Add `StartCostSync(ctx)` / `StopCostSync()` lifecycle methods
8. Unit tests for in-memory counter logic (no Firestore needed)

**Acceptance Criteria:**
- [ ] `client.GetAll()` and `client.Batch()` methods exist and compile
- [ ] `GetCostByProvider()` returns data from in-memory map (0 reads on hot path)
- [ ] Cost counter bootstraps from `_meta/cost_counters` doc on startup
- [ ] Counter syncs to Firestore every 5 minutes
- [ ] `MarkTaskCompleted()` updates in-memory counter atomically
- [ ] Unit tests pass for counter add/read/bootstrap logic
- [ ] `make lint` clean

**Risks:**
- Counter drift on crash — Mitigation: bootstrap from full scan on startup if `_meta` doc is stale
- Multi-instance concurrent updates — Mitigation: use `firestore.Increment()` for sync writes

### M2: MATERIALIZED_STATS — Task Stats + Stale Detector Cache
**Goal:** Replace `GetTaskStats()` full scan with materialized `_meta/task_stats` document. Add TTL cache to stale task detector.
**Estimated:** 150 LOC implementation + 40 LOC tests = 190 LOC
**Duration:** 1 day

**Tasks:**
1. Create `_meta/task_stats` document structure matching `coordinator.TaskStats`
2. Modify all state transition methods (`MarkTaskQueued`, `MarkTaskRunning`, `MarkTaskCompleted`, `MarkTaskFailed`) to atomically update `_meta/task_stats` using batch writes with `firestore.Increment()`
3. Rewrite `GetTaskStats()` to read single `_meta/task_stats` document
4. Add bootstrap logic: if `_meta/task_stats` doesn't exist, do one full scan to populate
5. Add `cachedTasks` / `cacheExpiry` fields to `StaleTaskDetector` with 90s TTL
6. Modify `detectAndMarkStale()` to check cache before querying
7. Tests for stats increment logic and cache TTL behavior

**Acceptance Criteria:**
- [ ] `GetTaskStats()` reads 1 document instead of N
- [ ] All state transitions update `_meta/task_stats` atomically via batch
- [ ] `_meta/task_stats` auto-bootstraps on first access
- [ ] Stale detector caches results for 90 seconds
- [ ] Existing coordinator tests pass
- [ ] `make lint` clean

**Risks:**
- Batch write failures leave counter inconsistent — Mitigation: reconciliation on next full bootstrap (24h or restart)

### M3: BATCH_AND_AGGREGATION — Batch Reads + Native Aggregation
**Goal:** Fix N+1 in exec hierarchy with batch GetAll. Replace client-side iteration with Firestore native Count()/Sum() aggregation.
**Estimated:** 130 LOC implementation + 30 LOC tests = 160 LOC
**Duration:** 1 day

**Tasks:**
1. Rewrite `GetExecTaskHierarchyWithMessages()` to collect all task IDs, then use `client.GetAll()` for a single batch read
2. Replace `GetDistinctWorkspaces()` — maintain `_meta/workspaces` set document, updated on thread creation
3. Replace `GetThreadAggregateStats()` — use Firestore `NewAggregationQuery().WithCount()` with status filters
4. Replace `GetMetricsSummary()` — use `Sum()` for tokens/cost fields, `Count()` for totals
5. Replace `GetMessageFlowEdges()` — materialized `_meta/message_flow_edges` doc updated on message send
6. Replace `GetActiveAgents()` — materialized `_meta/active_agents` doc or filtered aggregation
7. Tests for batch read grouping logic

**Acceptance Criteria:**
- [ ] `GetExecTaskHierarchyWithMessages()` uses single `GetAll()` batch (1 round-trip, not N)
- [ ] `GetDistinctWorkspaces()` reads from materialized doc (1 read)
- [ ] `GetMetricsSummary()` uses server-side `Sum()`/`Count()` (billed at 1/1000th cost)
- [ ] Dashboard aggregate functions do not iterate entire collections client-side
- [ ] `make lint` clean

**Risks:**
- Firestore aggregation API limitations (no GROUP BY) — Mitigation: fall back to materialized `_meta` docs where aggregation is insufficient
- `GetAll()` still charges per document — Mitigation: still eliminates N round-trips, which is the latency win

### M4: CLEANUP_AND_DEVMODE — TTL Cleanup + Dev Mode + Docs
**Goal:** Add document retention policy, dev mode configuration, and documentation.
**Estimated:** 100 LOC implementation + 30 LOC tests = 130 LOC
**Duration:** 0.5 day

**Tasks:**
1. Add `CleanupOldDocs()` method: deletes completed tasks older than retention period (30 days dev, 90 days prod)
2. Add dev mode config fields to coordinator YAML: `dev_mode`, `poll_interval`, `disable_stale_detector`, `disable_approval_watcher`
3. Wire dev mode into daemon startup to skip stale detector and approval watcher
4. Update CHANGELOG.md with all changes
5. Update design doc status to Implemented

**Acceptance Criteria:**
- [ ] `CleanupOldDocs()` deletes expired tasks/spans in batches
- [ ] Dev mode skips stale detector and approval watcher
- [ ] `coordinator.dev_mode: true` in YAML works correctly
- [ ] CHANGELOG.md updated
- [ ] Design doc moved to `implemented/v0_9_2/`
- [ ] `make lint` clean
- [ ] `make test` passes

**Risks:**
- Accidental deletion of active tasks — Mitigation: only delete `completed` or `failed` status, with age check

## Success Metrics

- Firestore reads drop >90% for `GetCostByProvider()` (N → 0 per dispatch)
- Firestore reads drop >99% for `GetTaskStats()` (N → 1 per status call)
- `GetExecTaskHierarchyWithMessages()` makes 1 batch call instead of N individual calls
- Dashboard endpoints serve from materialized data (1 read each)
- All tests passing
- All linting passing
- CHANGELOG.md updated

## Dependencies
- Firestore Go SDK v1.21.0 (already in go.mod — supports aggregation queries)
- `firestore.Increment()` already used in observatory_chains.go (proven pattern)
- No external dependencies needed

## Open Questions
- Should the cost counter reconciliation run on a schedule (e.g., daily) or only on startup?
- What's the desired retention policy for production (90 days suggested)?
- Should we add Firestore emulator to CI for integration tests, or keep these as manual verification?

## Notes
- No existing Firestore unit tests — all changes can be tested via unit tests on the in-memory/caching logic without requiring Firestore
- The SQLite store already implements most of these patterns correctly (GROUP BY, etc.) — this sprint brings Firestore to parity
- `firestore.Increment()` is atomic and safe for concurrent multi-instance updates
- The Client wrapper at `client.go` needs `GetAll()` and `Batch()` methods added (currently not exposed)
