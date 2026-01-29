# E2E Test: Coordinator Handoff Chains - Sprint Plan

**Status**: Ready for Sprint Execution
**Sprint ID**: E2E-HANDOFF
**Duration**: 2 days (8 hours)
**Risk Level**: LOW
**Date**: 2026-01-29

## Sprint Goal

Create comprehensive E2E tests verifying that the AILANG coordinator successfully chains handoffs between autonomous agents:
- `design-doc-creator` → creates design docs
- `sprint-planner` → creates sprint plans
- `sprint-executor` → executes implementation

This sprint validates the multi-agent handoff pattern used in the full dev-cycle pipeline.

## Problem Statement

The coordinator implements a three-stage pipeline where agents hand off work to each other. Currently:
- ✅ Individual agents can be invoked and complete tasks
- ✅ Messages can be sent between agents
- ⏳ **Handoff chains need E2E validation** - no test verifies the full pipeline works end-to-end
- ⏳ **Message passing between agents** needs integration tests
- ⏳ **Session continuity across agents** needs validation

### Why This Matters

The coordinator is designed for autonomous multi-agent workflows. Without E2E tests:
- Breaking changes to the handoff mechanism go undetected
- Message passing bugs only surface in production coordinator runs
- Session ID propagation issues cause silent failures
- Integration failures between agents aren't caught until humans run manual tests

## Current State Analysis

### What's Working
- Individual agent execution (design-doc-creator, sprint-planner, sprint-executor)
- Message queueing in SQLite database
- Task state transitions
- Basic handoff message formatting

### What Needs Testing
1. **Message flow** - Messages move correctly between agent inboxes
2. **Handoff triggers** - Agent completion triggers next agent in chain
3. **Session continuity** - Session IDs propagate through handoff messages
4. **Error recovery** - Failed handoffs don't hang the pipeline
5. **Dashboard visibility** - Handoff events show in UI

## Sprint Milestones

### M1: Integration Test Harness (~120 LOC, 4 hours)

**Goal**: Create reusable test utilities for multi-agent scenarios

**What we're building**:
- `TestCoordinatorChain` helper that:
  - Starts coordinator services (server + daemon)
  - Sends initial task to first agent (design-doc-creator)
  - Waits for agent completion
  - Verifies handoff message was created
  - Returns completion status

- Database inspection utilities:
  - `GetAgentMessages(inbox)` - Read all messages for an agent
  - `GetTaskStatus(taskID)` - Check current task state
  - `GetHandoffChain(rootTaskID)` - Trace full handoff sequence

**Files to create/update**:
- `tests/coordinator/e2e_harness_test.go` - New test harness (70 LOC)
- `internal/coordinator/testutil/test_helpers.go` - Helper functions (40 LOC)
- `internal/coordinator/testutil/cleanup.go` - Service cleanup (10 LOC)

**Acceptance Criteria**:
- [ ] `TestCoordinatorChain` creates/starts services without hanging
- [ ] Services can be cleanly shut down without timeout
- [ ] Database inspection helpers return correct data
- [ ] Example test using harness passes

**Dependencies**: None - builds on existing coordinator code

**Risk Factors**:
- Service startup/teardown timing issues (mitigate: generous timeouts, explicit waits)
- Database state pollution between tests (mitigate: separate test DBs)

---

### M2: Single Handoff Test (~100 LOC, 2 hours)

**Goal**: Verify design-doc-creator → sprint-planner handoff works

**What we're testing**:
```
1. Send message to design-doc-creator inbox
2. Agent processes message, creates design doc
3. Agent sends handoff message to sprint-planner
4. Verify message appears in sprint-planner inbox
5. Verify session_id is preserved
6. Verify correlation tracking works
```

**Test flow**:
```go
TestDesignDocToSprintPlannerHandoff:
  - Send "create design doc" message to design-doc-creator
  - Wait for task completion (max 5s)
  - Check sprint-planner inbox for new message
  - Verify message.CorrelationID matches original task
  - Verify message.SessionID is present and valid
```

**Files to create/update**:
- `tests/coordinator/handoff_single_test.go` - Single handoff test (100 LOC)

**Acceptance Criteria**:
- [ ] Handoff message appears in correct inbox
- [ ] Message timing is reasonable (<2s from completion to appearance)
- [ ] Correlation IDs are preserved
- [ ] Session IDs are propagated
- [ ] Test passes consistently (5 runs)

**Dependencies**: M1 (uses test harness)

**Risk Factors**:
- Timing issues (mitigate: polling with timeout vs blocking wait)
- Message visibility lag (mitigate: configurable polling interval)

---

### M3: Full Chain Test (~130 LOC, 2 hours)

**Goal**: Verify complete design-doc → sprint-plan → execution handoff chain

**What we're testing**:
```
1. Send task to design-doc-creator
2. Wait for handoff to sprint-planner
3. Sprint-planner processes and handoff to sprint-executor
4. Verify all agents completed their stages
5. Verify complete audit trail exists
```

**Test flow**:
```go
TestFullDesignToExecutionChain:
  - Send "implement feature X" to design-doc-creator
  - Poll for: design doc created → sprint plan created → execution started
  - Verify task_events table has complete chain: start → dc_done → sp_done → se_started
  - Verify no errors in coordinator logs
  - Check spans in observatory database for full chain
```

**Files to create/update**:
- `tests/coordinator/handoff_full_chain_test.go` - Full chain test (130 LOC)

**Acceptance Criteria**:
- [ ] All three agents execute in sequence
- [ ] Final task state shows successful completion
- [ ] task_events table has entries from all three agents
- [ ] Observatory traces show full call chain
- [ ] No timeouts or error states
- [ ] Test completes in <30 seconds
- [ ] Test passes consistently (3 runs)

**Dependencies**: M1, M2

**Risk Factors**:
- Cumulative timing (3 agents → more timeout risk)
- Agent-specific issues blocking chain
- Observable spans missing/incomplete

---

### M4: Error Recovery Tests (~80 LOC, 1 hour)

**Goal**: Verify handoff pipeline handles failures gracefully

**What we're testing**:
1. Agent fails → next agent doesn't auto-trigger
2. Approval rejection → agent re-triggered with feedback
3. Message queue corruption → graceful error
4. Missing design doc → sprint-planner rejects with message

**Test cases**:
- `TestHandoffWithoutApproval` - Chain pauses if `auto_approve_handoffs=false`
- `TestApprovalRejection` - Rejected task doesn't advance chain
- `TestMissingDesignDoc` - Sprint-planner handles missing input gracefully

**Files to create/update**:
- `tests/coordinator/handoff_errors_test.go` - Error scenario tests (80 LOC)

**Acceptance Criteria**:
- [ ] Failed agent doesn't crash subsequent agents
- [ ] Error messages are descriptive
- [ ] Manual intervention needed is clear
- [ ] All tests pass

**Dependencies**: M1, M2, M3

**Risk Factors**:
- Hard to trigger specific failure modes (mitigate: inject test stubs)
- Error paths not actually exercised (mitigate: chaos testing later)

---

## Task List

### Day 1 (4 hours)

**09:00-10:30 (1.5h)**: M1 - Create Integration Test Harness
- Create test helpers for coordinator startup/shutdown
- Create database inspection utilities
- Set up test cleanup/teardown

**10:30-12:00 (1.5h)**: M1 - Harness Validation
- Write basic test using harness
- Verify services start/stop cleanly
- Document gotchas for other tests

**12:00-13:00 (1h)**: Break + lunch

**13:00-14:00 (1h)**: M2 - Single Handoff Test
- Create first test (design-doc-creator → sprint-planner)
- Verify message appears in correct inbox
- Check session/correlation ID preservation

**14:00-15:30 (1.5h)**: M2 - Single Handoff Validation & Fixes
- Run test multiple times, fix timing issues
- Verify test is stable and repeatable
- Document any workarounds

### Day 2 (4 hours)

**09:00-11:00 (2h)**: M3 - Full Chain Test Implementation
- Build test for complete design → plan → execution chain
- Add task_events verification
- Add observable trace validation

**11:00-12:30 (1.5h)**: M3 - Chain Test Stabilization
- Run multiple times, fix any flakiness
- Optimize timeouts
- Verify consistent behavior

**12:30-13:30 (1h)**: Break + lunch

**13:30-14:30 (1h)**: M4 - Error Recovery Tests
- Implement failure scenario tests
- Test rejection flow
- Verify error messages

**14:30-15:30 (1h)**: M4 - Final Validation & Commit
- Run full test suite
- Verify all tests pass
- Commit with PR ready for review

## Success Metrics

### During Sprint
- [ ] All 4 milestones implemented
- [ ] All test files created (4 new files, ~430 LOC total)
- [ ] Helper utilities in place (70 LOC)
- [ ] No broken existing tests

### Test Results
- [ ] M2 (single handoff): Passes 5+ consecutive runs
- [ ] M3 (full chain): Passes 3+ consecutive runs
- [ ] M4 (error cases): All scenarios covered with tests
- [ ] Overall test execution: <60 seconds for full suite

### Code Quality
- [ ] All tests have clear naming and documentation
- [ ] Helper functions are reusable for future tests
- [ ] No test interdependencies (can run in any order)
- [ ] Coverage for happy path + error cases
- [ ] Examples of test patterns in CONTRIBUTING.md

### Documentation
- [ ] Test patterns documented in test files
- [ ] Gotchas and timing issues documented
- [ ] Example test using harness provided
- [ ] Future test developers have clear template

## Known Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Service startup timeout | Tests hang indefinitely | Use generous timeouts (10s), explicit kill switch |
| Database state pollution | Tests fail sporadically | Use separate test DBs per test, cleanup hooks |
| Message visibility lag | Tests flake on timing | Polling loop with configurable interval |
| Agent crashes | Chain breaks silently | Mock agents for error scenarios, explicit error checks |
| Session ID loss | Audit trail broken | Log all message state changes in test |
| Observatory spans missing | Trace validation fails | Build test without traces first, add later |

## Definition of Done

✅ Checklist for sprint completion:

- [ ] All 4 milestones implemented
- [ ] All tests pass consistently
- [ ] No regressions in existing tests
- [ ] New test helpers documented with examples
- [ ] Commit message explains what's tested
- [ ] Code review approved
- [ ] Ready for merge to `dev` branch

## Estimated Velocity

**Based on recent work:**
- M-TELEMETRY-HOOKS handoff doc: 185 LOC, 3 hours investigation
- Recent test infrastructure: 200 LOC, 2 hours per test suite
- **Estimate for this sprint**: 430 LOC tests + 70 LOC helpers = 500 LOC, 8 hours
- **Actual allocation**: 8 hours (2 days) → **62.5 LOC/hour velocity**

This assumes:
- Helper functions reusable and no rewrites needed
- No major blocking issues with coordinator implementation
- Timing/flakiness fixes don't exceed 30 minutes per test

## Next Steps

1. **Immediate** (sprint-executor):
   - Execute M1: Create test harness
   - Validate harness with simple example
   - Green light to proceed to M2

2. **After M1-M2**:
   - Review single handoff behavior
   - Identify any message format issues
   - Proceed to full chain test

3. **After Sprint**:
   - Merge test code to `dev` branch
   - Consider adding chaos testing (random failures)
   - Add coordinator stress tests (100+ concurrent handoffs)

4. **Future Phases**:
   - M-HANDOFF-CHAOS: Fault injection tests
   - M-COORDINATOR-PERF: Load testing with latency
   - M-COORDINATOR-HA: High availability testing

## Related Design Docs

- **M-TELEMETRY-HOOKS** (`design_docs/planned/v0_7_2/m-telemetry-hooks-handoff.md`) - Provides context on coordinator architecture
- **Coordinator Design** (`docs/guides/coordinator.md`) - Full coordinator documentation

## Appendix: Test Environment Setup

**Services that must run for E2E tests**:
```bash
# Option A: Automatic (recommended)
make services-start           # Starts server + coordinator

# Option B: Manual
ailang serve &                # Start server
ailang coordinator start &    # Start coordinator daemon
```

**Cleanup**:
```bash
make services-stop            # Stops both services
# Or manually:
kill $(lsof -t -i:1957)       # Kill server
ailang coordinator stop        # Stop daemon
```

**Verification**:
```bash
# Check both services are running
make services-status

# Check databases
sqlite3 ~/.ailang/state/coordinator.db ".tables"
sqlite3 ~/.ailang/state/collaboration.db ".tables"
sqlite3 ~/.ailang/state/observatory.db ".tables"
```

## Sprint Checklist

```
Day 1:
[ ] Start services (make services-start)
[ ] Implement M1 test harness
[ ] Validate M1 with example test
[ ] Implement M2 single handoff test
[ ] Fix timing issues in M2

Day 2:
[ ] Implement M3 full chain test
[ ] Stabilize M3 (run 3+ times)
[ ] Implement M4 error recovery tests
[ ] Run full test suite
[ ] Verify all tests pass
[ ] Commit and clean up

Post-sprint:
[ ] Code review
[ ] Address feedback
[ ] Merge to dev
```
