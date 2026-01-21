# Sprint Plan: M-COORD-ARTIFACT-DISCOVERY

**Sprint ID**: M-COORD-ARTIFACT
**Duration**: 1 day (4-6 hours)
**Risk Level**: Medium
**Design Doc**: [m-coord-artifact-discovery.md](m-coord-artifact-discovery.md)

## Sprint Summary

**Goal**: Fix coordinator workflow so that:
1. Tasks from inboxes have AgentID set
2. Directives include "invoke the X skill" instruction
3. Artifact discovery finds created files
4. Missing artifacts report errors (not empty comments)

**Deliverables**:
- 5 unit tests verifying each workflow step
- 1 integration test for end-to-end flow
- Bug fixes for AgentID and directive fallback
- Improved logging for debugging

## Current Status

**What's broken**:
- `Task.AgentID` not set when task created from inbox message
- `BuildStageDirective` uses `||` instead of `&&`, skipping skill invocation
- No error reported when artifacts missing

**Recent velocity**: ~150-200 LOC/day based on coordinator work

## Milestones

### M1: Unit Tests for Workflow Steps (~2 hours, 200 LOC)

Create `internal/coordinator/workflow_test.go` with tests for each step.

**Tasks**:
- [ ] `TestTaskCreationSetsAgentID` - Verify AgentID set from inbox
- [ ] `TestDirectiveIncludesSkillInvocation` - Verify directive text
- [ ] `TestArtifactDiscoveryFindsMatchingFiles` - Verify pattern matching
- [ ] `TestGitHubCommentIncludesContent` - Verify comment rendering
- [ ] `TestNoArtifactsReportsError` - Verify failure message

**Files**:
- `internal/coordinator/workflow_test.go` (NEW, ~200 LOC)

**Acceptance Criteria**:
- [ ] All 5 tests compile
- [ ] Tests fail initially (TDD - proving they test something real)
- [ ] Tests verify specific strings/values, not just "no error"

### M2: Fix Task Creation (~30 min, 5 LOC)

Set `AgentID` on task when created from inbox message.

**Tasks**:
- [ ] Add `AgentID: agentID` to TaskRecord creation in `daemon_tasks.go:257`

**Files**:
- `internal/coordinator/daemon_tasks.go` (~1 line change)

**Acceptance Criteria**:
- [ ] `TestTaskCreationSetsAgentID` passes
- [ ] Existing tests still pass

### M3: Fix Directive Fallback (~30 min, 20 LOC)

Fix `BuildStageDirective` to use AgentID when available.

**Tasks**:
- [ ] Change `||` to `&&` in the fallback check
- [ ] Use `task.AgentID` if available, fall back to stage-derived ID
- [ ] Ensure directive includes skill invocation text

**Files**:
- `internal/coordinator/stage_execution.go` (~15-20 LOC)

**Acceptance Criteria**:
- [ ] `TestDirectiveIncludesSkillInvocation` passes
- [ ] Directive contains "invoke the X skill"

### M4: Add Failure Reporting (~1 hour, 30 LOC)

Report errors when no artifacts found instead of empty comments.

**Tasks**:
- [ ] In `ProcessStageCompletion`, post failure message when no artifacts
- [ ] Include pattern and truncated agent output in error message
- [ ] Return error to caller for logging

**Files**:
- `internal/coordinator/stage_execution.go` (~20 LOC)
- `internal/coordinator/daemon_tasks.go` (~10 LOC for logging)

**Acceptance Criteria**:
- [ ] `TestNoArtifactsReportsError` passes
- [ ] GitHub comment shows "Failed to find artifacts" not empty

### M5: Integration Test (~1 hour, 150 LOC)

End-to-end test with mock executor.

**Tasks**:
- [ ] Create test that sends message to inbox
- [ ] Mock executor that creates design doc file
- [ ] Verify directive → artifact discovery → content extraction

**Files**:
- `internal/coordinator/e2e_workflow_test.go` (NEW, ~150 LOC)

**Acceptance Criteria**:
- [ ] Test runs without network/GitHub access (mocked)
- [ ] Full workflow verified from message to artifact

## Implementation Order

```
M1 (Tests) → M2 (AgentID) → M3 (Directive) → M4 (Errors) → M5 (E2E)
     ↓           ↓             ↓                ↓              ↓
  All fail    1 pass        2 pass          3 pass       All pass
```

## Success Metrics

| Metric | Target |
|--------|--------|
| Unit tests | 5 new, all passing |
| Integration tests | 1 new, passing |
| Test coverage | coordinator/ stays ≥60% |
| Manual verification | Message → skill invoked → artifact posted |

## Testing Commands

```bash
# Run workflow tests
go test ./internal/coordinator/... -run TestWorkflow -v

# Run E2E test
go test ./internal/coordinator/... -run TestEndToEnd -v

# Run all coordinator tests
go test ./internal/coordinator/... -v

# Manual verification
./bin/ailang messages send design-doc-creator "Test" --title "Test"
./bin/ailang messages unack <id>
tail -f ~/.ailang/logs/coordinator.log | grep -E "(AgentID|directive|artifact)"
```

## Risk Assessment

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| Breaking GitHub flow | Medium | High | Run existing tests after each change |
| Agent still doesn't invoke skill | Low | High | Verify worktree has .claude/skills/ |
| Artifact pattern doesn't match | Low | Medium | Log all discovered files before filtering |

## Estimated Timeline

| Milestone | Hours | Cumulative |
|-----------|-------|------------|
| M1: Tests | 2h | 2h |
| M2: AgentID | 0.5h | 2.5h |
| M3: Directive | 0.5h | 3h |
| M4: Errors | 1h | 4h |
| M5: E2E | 1h | 5h |
| Buffer | 1h | **6h total** |

---

**Created**: 2026-01-02
**Sprint Start**: Ready for approval
