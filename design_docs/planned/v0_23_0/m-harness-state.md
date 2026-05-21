# M-HARNESS-STATE: Shared Harness Substrate with Belief-State Synchronization

**Status**: Planned
**Target**: v0.23.0
**Priority**: P1 - Medium
**Estimated**: 1.5 weeks
**Dependencies**: Coordinator SQLite chain store (shipped), git integration in coordinator (shipped)

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Snapshot hash makes ground-truth state deterministic and content-addressable |
| A2: Replayability | +1 | Belief state snapshots enable replaying any executor's view at any point |
| A3: Effect Legibility | +1 | Read/write sets declared per task; conflicts are typed, not silent |
| A4: Explicit Authority | +1 | Executor write access to shared state is explicit and tracked |
| A5: Bounded Verification | +1 | Divergence check is local (hash comparison); no whole-repo scan |
| A6: Safe Concurrency | +1 | Conflict detection prevents silent races between parallel executor tasks |
| A7: Machines First | +1 | Machine-queryable harness state; no human reconstruction required |
| A8: Minimal Syntax | 0 | No language syntax changes |
| A9: Cost Visibility | 0 | No direct cost impact |
| A10: Composability | 0 | Adds state management complexity; net neutral |
| A11: Structured Failure | +1 | Typed `BeliefDivergence` error with diff |
| A12: System Boundary | +1 | Coordinator-executor boundary formalized with state protocol |

**Net Score: +9** → **Decision: ✅ Proceed**

### Hard Violation Check

- [x] A1 (Determinism): Ground truth is a content-addressed snapshot, not a mutable pointer
- [x] A3 (Effects): Read/write sets are declared per task — divergence is explicit
- [x] A4 (Authority): Executor write access tracked; no ambient mutation
- [x] A7 (Machines First): State is SQLite-queryable, not reconstructed from logs

## Problem Statement

AILANG's coordinator manages multiple executor chains, each operating on a shared repository. When two chains edit the same file, the conflict is currently detected only at `git merge` — after both chains have completed and invested compute.

**Current State:**
- Each executor maintains its own view of repository state (its local working tree or git worktree)
- The coordinator knows which chains are active but does not track what files each chain has read or written
- Divergence between an executor's belief ("file X has content Y") and ground truth ("file X has been modified by chain B") is undetected until merge
- No formal harness state object: the coordinator reconstructs chain context from conversational history at each invocation, making it expensive and lossy

**Impact:**
- Parallel chains waste compute on conflicting edits
- Long chains drift from shared state; late-detected conflicts require re-running from scratch
- "Code as Agent Harness" (arXiv:2605.18747) §3 finds "the majority of MAS literature resides in the implicit/file-only shared state category" and identifies this as the central architectural gap. AILANG's coordinator is already ahead of the field (SQLite chain store, explicit message-passing) but the formal belief-state model is missing.

## Goals

**Primary Goal:** Extend the coordinator's chain store to include a formal harness state object per chain, and add a synchronization protocol that detects executor belief-state divergence before it becomes a merge conflict.

**Success Metrics:**
- Divergence detected within one coordinator polling cycle (<30s) of a conflicting write
- `BeliefDivergence` error is typed and includes: conflicting chain IDs, diverged file path, diff between beliefs
- Zero silent merge conflicts from parallel chains in a 7-day production window after rollout
- `ailang chains --state` shows ground-truth snapshot hash and each executor's last-synced hash

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Snapshot granularity: file-level hashes vs. full repo hash | File-level enables precise conflict detection; full repo is cheaper | human | design | high |
| Reconciliation protocol: pause-and-merge vs. abort-and-restart | Pause-and-merge is complex but preserves work; abort is simpler | human | design | high |
| Polling vs. inotify/fsevents for divergence detection | inotify is faster but platform-specific; polling is portable | agent | compile | med |
| Belief state storage: SQLite vs. in-memory per coordinator instance | SQLite survives restarts; memory is faster | agent | compile | low |

### Design Freeze

Before implementation begins:

- [ ] Snapshot granularity: file-level hashes confirmed (recommendation: yes, needed for precise conflicts)
- [ ] Reconciliation protocol: abort-and-restart for v1 (pause-and-merge deferred)

## Solution Design

### Overview

Each coordinator chain maintains a `HarnessState` object in SQLite:

```go
type HarnessState struct {
  ChainID         string
  GroundTruthHash string     // content hash of coordinator's view of repo
  ExecutorHash    string     // content hash as last synced to executor
  ReadSet         []FilePath // files this chain has read
  WriteSet        []FilePath // files this chain has written
  DesignDocRef    string     // active design doc version (path + hash)
  PassingTests    []TestID   // tests passing at last sync point
  LastSyncTS      time.Time
}
```

A `HarnessMonitor` goroutine polls for writes. When chain A writes file F, the monitor checks all other active chains whose `ReadSet` contains F. If found, it emits a `BeliefDivergence` event.

### Architecture

**Components:**

1. **HarnessStateStore** (`internal/coordinator/harness_state.go`): CRUD on the `HarnessState` table in coordinator SQLite. Each chain has one row. Updated atomically.

2. **HarnessMonitor** (`internal/coordinator/harness_monitor.go`): Goroutine that polls the coordinator's git worktree for file changes (every 10s). On write detected to file F: looks up all chains with F in `ReadSet`. For each: computes divergence (their `ExecutorHash` vs. current `GroundTruthHash` for F). Emits `BeliefDivergence` if different.

3. **SyncProtocol** (`internal/coordinator/sync_protocol.go`): Handles `BeliefDivergence`. In v1: pauses the diverging chain, sends `StateDesync` message to the chain's executor via `ailang messages`, waits for acknowledgement, then re-syncs executor to ground truth. Chain resumes from last sync point.

4. **EffectSetTracker** (`internal/coordinator/effect_tracker.go`): Hooks into coordinator tool dispatch. On every file read/write by an executor, appends to `ReadSet`/`WriteSet` in HarnessState.

### Implementation Plan

**Phase 1: HarnessStateStore** (~2 days)
- [ ] Add `harness_state` table to coordinator SQLite schema
- [ ] `internal/coordinator/harness_state.go` — CRUD, migration
- [ ] Populate `GroundTruthHash` and `ExecutorHash` at chain start
- [ ] `ailang chains --state` CLI output showing hashes
- [ ] Unit tests for store CRUD

**Phase 2: EffectSetTracker** (~1.5 days)
- [ ] `internal/coordinator/effect_tracker.go` — hook into file read/write tool calls
- [ ] Append to `ReadSet`/`WriteSet` in HarnessState on each tool call
- [ ] Integration test: run two chains, verify read sets populated correctly

**Phase 3: HarnessMonitor + SyncProtocol** (~2 days)
- [ ] `internal/coordinator/harness_monitor.go` — polling goroutine, conflict detection
- [ ] `internal/coordinator/sync_protocol.go` — v1 abort-and-restart on divergence
- [ ] Integration test: simulate two chains editing same file; verify `BeliefDivergence` emitted
- [ ] E2E test: conflict caught before git merge

**Phase 4: Documentation** (~0.5 days)
- [ ] `docs/docs/guides/coordinator.md` — harness state section
- [ ] `docs/docs/guides/database-architecture.md` — new table schema

### Files to Modify/Create

**New files:**
- `internal/coordinator/harness_state.go` — HarnessStateStore (~150 LOC)
- `internal/coordinator/harness_monitor.go` — polling monitor (~120 LOC)
- `internal/coordinator/sync_protocol.go` — SyncProtocol v1 (~100 LOC)
- `internal/coordinator/effect_tracker.go` — read/write set tracking (~80 LOC)
- `internal/coordinator/harness_state_test.go` — unit + integration tests (~200 LOC)

**Modified files:**
- `internal/coordinator/chains.go` — add `harness_state` table to schema (~20 LOC)
- `internal/coordinator/daemon_tasks.go` — start HarnessMonitor goroutine (~15 LOC)
- `cmd/ailang/chains.go` — `--state` flag for harness state display (~30 LOC)
- `docs/docs/guides/coordinator.md` (~50 LOC)
- `docs/docs/guides/database-architecture.md` (~30 LOC)

## Examples

### Example 1: Normal Operation (no conflict)

```bash
$ ailang chains --state

CHAIN_ID     CAMP           GROUND_HASH  EXEC_HASH   READ_SET              WRITE_SET         SYNC
abc123       claude-agentic 4f7a2b1c     4f7a2b1c    [checker.go, ...]     [checker.go]      ✓ synced
def456       motoko         4f7a2b1c     4f7a2b1c    [elaborator.go, ...]  []                ✓ synced
```

### Example 2: Divergence Detected

```bash
# Chain abc123 writes internal/types/checker.go
# Chain def456 has checker.go in its ReadSet

$ ailang chains --state

CHAIN_ID     CAMP           GROUND_HASH  EXEC_HASH   SYNC
abc123       claude-agentic 8c3d1f2a     8c3d1f2a    ✓ synced  (wrote checker.go)
def456       motoko         8c3d1f2a     4f7a2b1c    ⚠ DESYNC  (checker.go changed under it)

HarnessMonitor: BeliefDivergence detected
  Chain def456 read internal/types/checker.go at hash 4f7a2b1c
  Ground truth is now 8c3d1f2a (written by chain abc123)
  Action: pausing def456, sending StateDesync message, re-syncing to 8c3d1f2a
```

### Example 3: Verbose state with design doc reference

```bash
$ ailang chains --state --verbose

chain_id: abc123
  design_doc: design_docs/planned/v0_23_0/m-harness-state.md (hash: d4e5f6)
  passing_tests: [TestHarnessStore, TestSyncProtocol, ...]
  last_sync: 2026-05-21T09:15:00Z
```

## Success Criteria

- [ ] `HarnessState` row exists in SQLite for every active chain
- [ ] `ReadSet` and `WriteSet` populated correctly for file read/write tool calls
- [ ] `BeliefDivergence` detected within one polling cycle when two chains write the same file
- [ ] Diverging chain paused and re-synced to ground truth without data loss
- [ ] `ailang chains --state` shows ground truth and executor hash, flags desyncs
- [ ] Zero silent merge conflicts in 7-day production window
- [ ] All tests passing (`make test`)
- [ ] Coordinator guide and DB architecture docs updated

## Testing Strategy

**Unit tests:**
- `TestHarnessStateStore_CRUD` — create, read, update, delete HarnessState rows
- `TestEffectTracker_ReadSet` — verify ReadSet populated on file read tool call
- `TestHarnessMonitor_DetectsDivergence` — simulate two chains writing same file

**Integration tests:**
- Spawn two test chains on the same worktree; have chain A write file F; verify `BeliefDivergence` emitted for chain B (which read F)
- Verify re-sync: chain B's `ExecutorHash` matches `GroundTruthHash` after sync

**Manual testing:**
- Run a real parallel sprint with two claude-agentic chains; observe `--state` output; confirm no silent merge conflict

## Deferred Decisions

- Pause-and-merge reconciliation (vs. abort-and-restart) — agent may implement in v2 if abort causes too much work loss
- inotify/fsevents-based monitoring (vs. polling) — agent may add for lower latency on Linux/macOS
- Cross-coordinator synchronization (multiple coordinator instances) — deferred to v1_0_0

## Non-Goals

- **OS-level file locking** — this is a coordinator-level protocol, not kernel-level
- **Three-way merge** — out of scope for v1; re-sync resets to ground truth
- **Distributed coordinator** — single-coordinator assumption holds for v0.x

## Timeline

**Week 1** (~5 days):
- Phase 1: HarnessStateStore (days 1–2)
- Phase 2: EffectSetTracker (days 3–4)

**Week 2** (~3 days):
- Phase 3: HarnessMonitor + SyncProtocol (days 1–2)
- Phase 4: Documentation + tests (day 3)

**Total: ~8 days**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Polling overhead slows coordinator | Med | Poll every 30s default; configurable via `AILANG_HARNESS_POLL_INTERVAL` |
| Re-sync discards executor's in-progress work | High | Checkpoint executor context before re-sync; restore on resume |
| ReadSet grows unbounded for long chains | Med | Trim to last N files; configurable |

## Related Documents

**Planned (same cluster):**
- [design_docs/planned/v0_23_0/m-harness-dsl.md](design_docs/planned/v0_23_0/m-harness-dsl.md) — Doc 4: harness DSL declares read/write sets per stage; this doc tracks them at runtime
- [design_docs/planned/v0_23_0/m-permission-model.md](design_docs/planned/v0_23_0/m-permission-model.md) — Doc 2: permission model pre-populates WriteSet from declared effects
- [design_docs/planned/v0_23_0/m-trace-feedback.md](design_docs/planned/v0_23_0/m-trace-feedback.md) — Doc 1: `MISSING_CONTEXT` failures are detectable from belief-state divergence

## References

- **Ning et al. (2026).** Code as Agent Harness. arXiv:[2605.18747](https://arxiv.org/abs/2605.18747) — §3 "implicit/file-only shared state is the central gap"; SyncMind formal shared substrate ($S_k$ ground truth vs. $B_k$ belief state); "topology complexity is a symptom of implicit state"
- [Design Axioms](/docs/references/axioms)
- [Database Architecture Guide](../../../docs/docs/guides/database-architecture.md)
- [Coordinator Guide](../../../docs/docs/guides/coordinator.md)

## Future Work

- **Three-way merge reconciliation**: instead of abort-and-restart, merge changes from both chains when the conflict is non-overlapping
- **Cross-instance harness state**: share HarnessState across multiple coordinator instances via a shared Firestore document
- **Belief-state time travel**: `ailang chains replay --at TIMESTAMP` reconstructs any executor's view at any past point

---

**Document created**: 2026-05-21
**Last updated**: 2026-05-21
