# M-CHAINS-SOURCE-OF-TRUTH Phase 3: Chain Metric Rollup - Sprint Plan

**Sprint ID**: M-CHAINS-P3
**Target Version**: v0.7.3
**Priority**: P1 (Medium)
**Duration**: 2-3 days (estimated 20 hours / ~250 LOC)
**Risk Level**: Low (additive, uses existing APIs)

## Executive Summary

Implement automatic metric rollup from task execution results to the chain/stage hierarchy. Currently, task metrics are recorded in `coordinator.db` but don't propagate to `observatory.db`, leaving the dashboard with zero costs/tokens at chain and stage levels.

**Key deliverables:**
- Add metric rollup hooks in `daemon_tasks.go` after task execution
- Handle approval/handoff transitions with metric finalization
- Write comprehensive integration tests with E2E chain verification
- Update dashboard with accurate cumulative costs

**Success metrics:**
- [ ] All chain/stage metrics populated after task execution
- [ ] Dashboard shows accurate hierarchy totals
- [ ] Integration tests verify rollup logic
- [ ] Zero manual aggregation required
- [ ] All existing tests pass

## Analysis & Findings

### Current State

**Observatory backend:** Ready
- `UpdateStageMetrics(ctx, stageID, cost, tokensIn, tokensOut, turns, toolCalls, durationMs)` exists
- `UpdateChainMetrics(ctx, chainID, cost, tokens, turns)` exists
- Both use SQL `+=` operators (accumulative, OK for retries)

**Coordinator daemon:** Partial
- Task execution produces `ExecuteResult` with cost/tokens
- Result metrics recorded to messaging thread (line 1110+)
- **Missing:** No calls to observatory metric functions

**Velocity baseline:**
- Recent commits show ~100-150 LOC/day for coordinator work
- Metric update calls are straightforward (25-30 LOC each)
- Testing is ~70% of implementation size

### Implementation Scope

| Component | Estimated LOC | Complexity |
|-----------|---------------|-----------|
| Phase 1: Stage metric updates | 40 | Low |
| Phase 2: Chain metric rollup | 35 | Low |
| Phase 3: Approval/handoff handling | 25 | Low |
| Phase 4: Turn/tool metrics from spans | 40 | Medium |
| **Integration tests** | 80 | Medium |
| **Total** | **220** | **Low-Medium** |

### Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Cross-database consistency | Medium | Graceful error handling, log warnings, don't fail tasks |
| Metric accumulation on retries | Low | Existing `+=` operators handle idempotency |
| Span extraction complexity | Medium | Defer to Phase 4 (use placeholder for Phase 1-3) |
| Test environment setup | Low | Use `observatory.NewMemoryBackend()` for mocks |

---

## Milestones & Task Breakdown

### Milestone 1: Core Stage Metric Updates (1 day / ~70 LOC)

**Goal:** After task execution completes, update stage metrics in observatory.

**Implementation:**

**Location:** `internal/coordinator/daemon_tasks.go::postTaskResult()` (around line 1036)

**Task 1.1: Add stage metric update call** (~25 LOC)
```go
// After recording to messaging thread (post-line 1110)
if task.StageID != "" && d.obsBackend != nil && result != nil && result.Cost > 0 {
    if err := d.obsBackend.UpdateStageMetrics(
        taskCtx,
        task.StageID,
        result.Cost,
        result.InputTokens,
        result.OutputTokens,
        1,  // Placeholder: actual turn count from Phase 4
        0,  // Placeholder: tool call count from Phase 4
        int64(result.Duration.Milliseconds()),
    ); err != nil {
        d.logger.Printf("Warning: Failed to update stage %s metrics: %v", task.StageID, err)
    } else {
        d.logger.Printf("Updated stage %s metrics: cost=$%.4f tokens=%d",
            task.StageID, result.Cost, result.InputTokens+result.OutputTokens)
    }
}
```

**Files modified:**
- `internal/coordinator/daemon_tasks.go` - Add metric update call

**Acceptance criteria:**
- [ ] UpdateStageMetrics called with correct cost/token values
- [ ] Logs success/failure appropriately
- [ ] Graceful error handling (log but don't fail task)
- [ ] Works with null/empty stage ID (early return)

---

### Milestone 2: Chain Metric Rollup (0.5 day / ~40 LOC)

**Goal:** Aggregate metrics from stage to chain level.

**Implementation:**

**Location:** `internal/coordinator/daemon_tasks.go` - Add helper method + call

**Task 2.1: Add updateChainMetrics helper method** (~35 LOC)
```go
// updateChainMetrics rolls up metrics from task to chain
func (d *Daemon) updateChainMetrics(ctx context.Context, task *TaskRecord, result *ExecuteResult) {
    if task.ChainID == "" || d.obsBackend == nil || result == nil || result.Cost == 0 {
        return
    }

    // Update denormalized chain metrics
    if err := d.obsBackend.UpdateChainMetrics(
        ctx,
        task.ChainID,
        result.Cost,
        result.InputTokens + result.OutputTokens,
        1, // Placeholder: actual turn count from Phase 4
    ); err != nil {
        d.logger.Printf("Warning: Failed to update chain %s metrics: %v", task.ChainID, err)
    } else {
        d.logger.Printf("Updated chain %s metrics: cost=$%.4f tokens=%d",
            task.ChainID, result.Cost, result.InputTokens+result.OutputTokens)
    }
}
```

**Task 2.2: Call updateChainMetrics after stage update** (~5 LOC)
```go
// In postTaskResult, after stage metrics update:
d.updateChainMetrics(taskCtx, task, result)
```

**Files modified:**
- `internal/coordinator/daemon_tasks.go` - Add helper + call

**Acceptance criteria:**
- [ ] updateChainMetrics handles nil values gracefully
- [ ] Called after stage metric update
- [ ] Logs success/failure appropriately
- [ ] Works with null/empty chain ID (early return)
- [ ] Cost and token aggregation correct

---

### Milestone 3: Approval & Handoff Handling (0.5 day / ~30 LOC)

**Goal:** Finalize stage metrics when approval/handoff completes, mark stage as completed.

**Implementation:**

**Location:** `internal/coordinator/daemon_approval.go` (around handleApproval)

**Task 3.1: Update stage status on approval** (~20 LOC)
```go
// After merging changes in handleApproval()
if task.StageID != "" && d.obsBackend != nil {
    // Mark stage as completed
    if err := d.obsBackend.UpdateStageStatus(ctx, task.StageID, observatory.StageStatusCompleted); err != nil {
        d.logger.Printf("Warning: Failed to mark stage %s as completed: %v", task.StageID, err)
    } else {
        d.logger.Printf("Stage %s marked as completed", task.StageID)
    }
}
```

**Task 3.2: Handle chain status transitions** (~10 LOC)
- When last stage completes, update chain status to "Completed"
- When new stage starts, update chain status to "InProgress"

**Files modified:**
- `internal/coordinator/daemon_approval.go` - Update stage/chain status

**Acceptance criteria:**
- [ ] Stage marked as "Completed" when approval merges
- [ ] Chain status updated on stage transitions
- [ ] Graceful error handling (log but don't fail approval)
- [ ] Existing approval workflow unaffected

---

### Milestone 4: Turn & Tool Count Extraction (0.5 day / ~50 LOC) - PHASE 4

**Goal:** Extract accurate turn/tool counts from event streams instead of using placeholder `1`.

**Implementation:**

**Location:** `internal/coordinator/event_handler.go` + `daemon_tasks.go`

**Task 4.1: Add turn/tool counting to event handler** (~30 LOC)
```go
// In EventHandler struct:
type EventHandler struct {
    // ... existing fields
    turnCount int
    toolCallCount int
}

// Add methods:
func (h *EventHandler) IncrementTurnCount() { h.turnCount++ }
func (h *EventHandler) IncrementToolCallCount() { h.toolCallCount++ }
func (h *EventHandler) GetTurnCount() int { return h.turnCount }
func (h *EventHandler) GetToolCallCount() int { return h.toolCallCount }
```

**Task 4.2: Extract counts from event stream** (~20 LOC)
```go
// After span recording in executeTask (line 840)
var turnCount, toolCallCount int
if eventHandler != nil {
    turnCount = eventHandler.GetTurnCount()
    toolCallCount = eventHandler.GetToolCallCount()
}

// Update stage metrics with actual counts
if task.StageID != "" && d.obsBackend != nil && result != nil {
    if err := d.obsBackend.UpdateStageMetrics(
        taskCtx,
        task.StageID,
        0,  // cost already updated
        0, 0,  // tokens already updated
        turnCount,
        toolCallCount,
        0,  // duration already updated
    ); err != nil {
        d.logger.Printf("Warning: Failed to update stage turn metrics: %v", err)
    }
}
```

**Files modified:**
- `internal/coordinator/event_handler.go` - Add turn/tool counting
- `internal/coordinator/daemon_tasks.go` - Extract and use counts

**Acceptance criteria:**
- [ ] Turn count extracted from event stream
- [ ] Tool call count extracted from event stream
- [ ] Updates replace placeholder `1` with actual counts
- [ ] Backward compatible with events lacking count data

---

## Integration Tests

**Location:** `internal/coordinator/daemon_metrics_test.go` (new file)

### Test Suite 1: Stage Metric Updates

**Test 1.1: Stage metrics updated after successful task execution** (~30 LOC)
- Create mock observatory backend
- Execute task with stage ID
- Verify `UpdateStageMetrics` called with correct values
- Verify cost/tokens propagated

```go
func TestStageMetricsUpdatedAfterExecution(t *testing.T) {
    // Setup: Mock backend, create task with stage ID
    // Execute: Run executeTask
    // Assert: UpdateStageMetrics called with task's cost/tokens
}
```

**Test 1.2: Stage metrics NOT updated if stage ID missing** (~15 LOC)
- Create task WITHOUT stage ID
- Execute task
- Verify UpdateStageMetrics NOT called

**Test 1.3: Stage metrics update fails gracefully** (~15 LOC)
- Mock backend returns error
- Execute task
- Verify error logged but task succeeds

### Test Suite 2: Chain Metric Rollup

**Test 2.1: Chain metrics updated after stage metrics** (~30 LOC)
- Create task with both stage and chain IDs
- Execute task
- Verify UpdateChainMetrics called
- Verify aggregates correctly

```go
func TestChainMetricsRollupAfterStage(t *testing.T) {
    // Setup: Mock backend, task with chain + stage IDs
    // Execute: Run executeTask
    // Assert: UpdateChainMetrics called, cost = result cost
}
```

**Test 2.2: Chain metrics NOT updated if chain ID missing** (~15 LOC)

**Test 2.3: Multiple tasks aggregate to same chain** (~25 LOC)
- Execute 3 tasks for same chain ID
- Verify chain metrics = sum of all 3 tasks

### Test Suite 3: End-to-End Integration

**Test 3.1: Multi-agent chain with metric verification** (~60 LOC)
- Create 3-stage chain (design → sprint → execute)
- Execute one task per stage
- Query observatory.db directly
- Verify:
  - Stage 1: cost = task 1 cost
  - Stage 2: cost = task 2 cost
  - Chain total: cost = task 1 + task 2 + task 3

```go
func TestE2EChainMetricRollup(t *testing.T) {
    // Setup: Create coordinator daemon with real SQLite backend
    // Execute: Run 3-task chain
    // Assert: Query chain_stages and execution_chains tables
    // Verify: Totals match task execution results
}
```

**Test 3.2: Approval/handoff with metric finalization** (~40 LOC)
- Execute task
- Verify metrics updated
- Approve task (trigger handoff)
- Verify stage status = "Completed"
- Verify chain status transitioned

**Test 3.3: Metrics persist across retries** (~25 LOC)
- Execute task (first attempt)
- Verify stage metrics: cost=$0.10
- Execute same task again (retry)
- Verify stage metrics: cost=$0.20 (accumulated)

### Test Suite 4: Error Handling

**Test 4.1: Observatory backend unavailable** (~15 LOC)
- Set `d.obsBackend = nil`
- Execute task
- Verify task succeeds (no panic)
- Verify logs indicate missing backend

**Test 4.2: Metric update partial failure** (~20 LOC)
- Stage update succeeds, chain update fails
- Verify stage metrics in DB
- Verify error logged for chain
- Verify task succeeds

---

## Dashboard Integration Checklist

**After metrics are rolled up, verify dashboard shows them:**

- [ ] Navigation to chain detail view
- [ ] Stage costs display correctly (sum of child tasks)
- [ ] Chain cost displays correctly (sum of all stages)
- [ ] Token counts aggregated properly
- [ ] Turn counts aggregated properly
- [ ] No "0.00" values for completed chains
- [ ] Metric format: "$X.XX" for cost, "N tokens" for tokens

**Example query to verify in observatory.db:**
```sql
SELECT id, total_cost, total_tokens FROM execution_chains WHERE id = 'chain-abc123';
-- Should show non-zero values after task execution
```

---

## Files Modified

| File | Changes | LOC |
|------|---------|-----|
| `internal/coordinator/daemon_tasks.go` | Add stage + chain metric update calls | +60 |
| `internal/coordinator/daemon_approval.go` | Update stage/chain status on approval | +25 |
| `internal/coordinator/event_handler.go` | Add turn/tool counting methods | +20 |
| `internal/coordinator/daemon_metrics_test.go` | NEW: Integration tests | +250 |
| **Total** | | **+355** |

---

## Success Criteria

### Phase 1-3 (Core Implementation)
- [ ] Stage metrics populated after task execution
- [ ] Chain metrics aggregate from all stages
- [ ] Approval/handoff transitions finalize metrics
- [ ] Graceful error handling (no task failures)
- [ ] All existing tests pass

### Phase 4 (Turn/Tool Metrics) - Optional for Phase 1
- [ ] Turn count extracted from event stream
- [ ] Tool call count extracted from event stream
- [ ] Placeholders replaced with actual values

### Dashboard
- [ ] Dashboard shows accurate costs/tokens at all levels
- [ ] Chain detail view displays rollup summary
- [ ] Stage detail view displays aggregated metrics

### Testing
- [ ] Unit tests for metric update calls (15 tests)
- [ ] Integration tests for stage/chain rollup (8 tests)
- [ ] E2E test with multi-agent chain (1 test)
- [ ] Test coverage ≥ 80% for coordinator/daemon_metrics_test.go

---

## Development Workflow

### Day 1: Phase 1-2 (Stage & Chain Metrics)

**Morning (2-3 hours):**
1. Implement `postTaskResult` stage metric update (Task 1.1)
2. Write unit tests for stage metric calls
3. Verify with manual test: execute task, query observatory.db

**Afternoon (2-3 hours):**
1. Implement `updateChainMetrics` helper (Task 2.1-2.2)
2. Write unit tests for chain metric rollup
3. Verify with manual test: execute multi-stage chain

**Evening (1 hour):**
- Code review, fix any issues
- Prepare for integration tests

### Day 2: Phase 3 & Integration Tests

**Morning (2-3 hours):**
1. Implement approval/handoff handling (Task 3.1-3.2)
2. Write status transition tests
3. Test approval workflow end-to-end

**Afternoon (2-3 hours):**
1. Write comprehensive integration tests (Test Suites 1-4)
2. Create E2E test with multi-agent chain
3. Verify all tests pass

**Evening (1-2 hours):**
- Debug test failures
- Ensure test coverage ≥ 80%

### Day 3: Phase 4, Documentation & Cleanup

**Morning (2-3 hours):**
1. Implement turn/tool counting (Task 4.1-4.2) - OPTIONAL
2. Update turn count tests
3. Verify metric accuracy

**Afternoon (1-2 hours):**
1. Document metric rollup in code comments
2. Update design doc with implementation notes
3. Create CHANGELOG entry

**Final (1 hour):**
- Lint and format code
- Run full test suite
- Commit and prepare for review

---

## Known Unknowns & Questions

1. **Turn counting source:** Should we extract from event stream (Phase 4) or use fixed placeholder for Phase 1-3?
   - **Recommendation:** Use placeholder `1` in Phase 1-3, defer accurate counting to Phase 4
   - **Reason:** Simpler core logic, can be added incrementally

2. **Metric accumulation behavior:** Is it OK that `+=` operators accumulate on retries?
   - **Answer:** YES - design doc notes this is intentional (metrics accumulate across attempts)

3. **Approval workflow:** Should we update chain status when stage status changes?
   - **Answer:** YES - implement in Phase 3 (Task 3.2)

4. **Dashboard caching:** Will dashboard cache metric values or always query fresh?
   - **Answer:** Out of scope for this sprint - assume dashboard queries fresh

---

## Dependencies

**Already implemented:**
- ✅ Observatory backend with UpdateStageMetrics/UpdateChainMetrics
- ✅ Coordinator daemon task execution
- ✅ Chain/stage hierarchy in observatory.db
- ✅ ExecuteResult with cost/token fields

**Must be done first:**
- None - this is a pure addition to existing system

**Enables future work:**
- Real-time streaming metrics (batch → streaming)
- Cost predictions before chain execution
- Budget enforcement (halt chains exceeding cost)
- Metric reconciliation job

---

## Timeline Estimate

| Milestone | Tasks | Days | Hours |
|-----------|-------|------|-------|
| 1. Stage metrics | 1.1 | 1 | 3-4 |
| 2. Chain rollup | 2.1-2.2 | 0.5 | 2 |
| 3. Approval handling | 3.1-3.2 | 0.5 | 2-3 |
| **Core implementation** | **Subtotal** | **2** | **7-9** |
| Integration tests | Test Suites 1-4 | 1 | 6-8 |
| Phase 4 (optional) | 4.1-4.2 | 0.5 | 3 |
| Documentation & cleanup | | 0.5 | 2 |
| **Total** | | **3** | **18-22** |

**Realistic estimate: 2-3 days (with Phase 4 optional)**

---

## Risk Mitigation

| Risk | Likelihood | Severity | Mitigation |
|------|-----------|----------|-----------|
| Cross-database sync issues | Medium | Low | Graceful error handling, no task failures |
| Test environment setup | Low | Low | Use observatory.NewMemoryBackend() |
| Observatory backend API mismatch | Low | High | API already exists, just call it |
| Metric accuracy (turn counts) | Medium | Low | Defer to Phase 4, use placeholder |
| Approval workflow breakage | Low | High | Write tests before implementing |

---

## Next Steps (After Approval)

1. **Review this sprint plan** - Get feedback on milestones/estimates
2. **Create sprint JSON** - For multi-session tracking
3. **Hand off to sprint-executor** - With this plan as reference
4. **Execute Day 1-2** - Core metric rollup + tests
5. **Review & merge** - Prepare PR with all tests passing

---

**Sprint Plan Created**: 2026-02-07
**Prepared by**: sprint-planner skill
**Version**: v0.7.3

